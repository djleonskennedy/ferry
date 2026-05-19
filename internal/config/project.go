// Package config loads and saves ferry's two TOML files: the per-project
// ferry.toml (committed) and the user-global ~/.ferry/config.toml (private).
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type ProjectConfig struct {
	Project    ProjectSection    `toml:"project"`
	Encryption EncryptionSection `toml:"encryption"`
	Files      []FileEntry       `toml:"files"`
}

type ProjectSection struct {
	Name string `toml:"name"`
}

type EncryptionSection struct {
	KeyID    string `toml:"key_id"`
	Required bool   `toml:"required"`
}

type FileEntry struct {
	Path     string `toml:"path"`
	Required bool   `toml:"required"`
}

func DefaultProject(name string) *ProjectConfig {
	return &ProjectConfig{
		Project:    ProjectSection{Name: name},
		Encryption: EncryptionSection{KeyID: "default", Required: true},
		Files:      nil,
	}
}

func LoadProject(path string) (*ProjectConfig, error) {
	var cfg ProjectConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if cfg.Project.Name == "" {
		return nil, errors.New("ferry.toml: [project].name is required")
	}
	if cfg.Encryption.KeyID == "" {
		cfg.Encryption.KeyID = "default"
	}
	return &cfg, nil
}

func SaveProject(path string, c *ProjectConfig) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}
