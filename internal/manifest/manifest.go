// Package manifest models the per-snapshot manifest.toml.
package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Manifest struct {
	Version      int            `toml:"version"`
	Project      string         `toml:"project"`
	CreatedAt    time.Time      `toml:"created_at"`
	KeyID        string         `toml:"key_id"`
	Message      string         `toml:"message,omitempty"`
	FerryVersion string         `toml:"ferry_version"`
	Files        []ManifestFile `toml:"files"`
}

type ManifestFile struct {
	Path   string `toml:"path"`
	Size   int64  `toml:"size"`
	SHA256 string `toml:"sha256"`
	Mode   uint32 `toml:"mode"`
}

func Write(path string, m *Manifest) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(m)
}

func Read(path string) (*Manifest, error) {
	var m Manifest
	if _, err := toml.DecodeFile(path, &m); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return &m, nil
}

// Find returns the entry for relPath, or nil if absent.
func (m *Manifest) Find(relPath string) *ManifestFile {
	for i := range m.Files {
		if m.Files[i].Path == relPath {
			return &m.Files[i]
		}
	}
	return nil
}

// HashReader streams r through sha256 and returns the hex digest along with
// the number of bytes read.
func HashReader(r io.Reader) (string, int64, error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// HashFile returns the hex sha256 of the file's contents.
func HashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	return HashReader(f)
}
