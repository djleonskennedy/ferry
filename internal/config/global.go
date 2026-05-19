package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/djleonskennedy/ferry/internal/paths"
)

type GlobalConfig struct {
	Keys     map[string]KeyEntry `toml:"keys"`
	Defaults Defaults            `toml:"defaults"`
}

type KeyEntry struct {
	Path string `toml:"path"`
}

type Defaults struct {
	BackupOnApply     bool `toml:"backup_on_apply"`
	BackupRetention   int  `toml:"backup_retention"`
	SnapshotRetention int  `toml:"snapshot_retention"`
}

func defaultGlobal() *GlobalConfig {
	return &GlobalConfig{
		Keys: map[string]KeyEntry{},
		Defaults: Defaults{
			BackupOnApply:     true,
			BackupRetention:   10,
			SnapshotRetention: 0,
		},
	}
}

// LoadGlobal reads ~/.ferry/config.toml. If missing, returns defaults
// (without writing the file).
func LoadGlobal() (*GlobalConfig, error) {
	path := paths.GlobalConfigFile()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultGlobal(), nil
	}
	if err != nil {
		return nil, err
	}
	cfg := defaultGlobal()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if cfg.Keys == nil {
		cfg.Keys = map[string]KeyEntry{}
	}
	return cfg, nil
}

func SaveGlobal(c *GlobalConfig) error {
	path := paths.GlobalConfigFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}

// ResolveKeyPath returns the on-disk path for the named key entry.
// If unset, falls back to the conventional location under KeysDir().
func (g *GlobalConfig) ResolveKeyPath(id string) (string, error) {
	if id == "" {
		return "", errors.New("empty key id")
	}
	if e, ok := g.Keys[id]; ok && e.Path != "" {
		return expandHome(e.Path), nil
	}
	return paths.KeyFile(id), nil
}

func expandHome(p string) string {
	if len(p) >= 2 && p[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
