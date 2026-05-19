// Package safety compares files on disk to a manifest's recorded hashes.
package safety

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/djleonskennedy/ferry/internal/manifest"
)

type Status int

const (
	Same Status = iota
	Modified
	Missing
)

func (s Status) String() string {
	switch s {
	case Same:
		return "same"
	case Modified:
		return "modified"
	case Missing:
		return "missing"
	}
	return "unknown"
}

type FileCompare struct {
	Path   string
	Status Status
}

// CompareToManifest hashes each manifest file at destRoot and reports drift.
func CompareToManifest(destRoot string, m *manifest.Manifest) ([]FileCompare, error) {
	out := make([]FileCompare, 0, len(m.Files))
	for _, mf := range m.Files {
		abs := filepath.Join(destRoot, filepath.FromSlash(mf.Path))
		_, err := os.Stat(abs)
		if errors.Is(err, os.ErrNotExist) {
			out = append(out, FileCompare{Path: mf.Path, Status: Missing})
			continue
		}
		if err != nil {
			return nil, err
		}
		hash, _, err := manifest.HashFile(abs)
		if err != nil {
			return nil, err
		}
		st := Modified
		if hash == mf.SHA256 {
			st = Same
		}
		out = append(out, FileCompare{Path: mf.Path, Status: st})
	}
	return out, nil
}

// HasModified reports whether any file in the comparison drifted.
func HasModified(cmp []FileCompare) bool {
	for _, c := range cmp {
		if c.Status == Modified {
			return true
		}
	}
	return false
}

// ModifiedPaths returns just the drifted paths from a comparison.
func ModifiedPaths(cmp []FileCompare) []string {
	var out []string
	for _, c := range cmp {
		if c.Status == Modified {
			out = append(out, c.Path)
		}
	}
	return out
}
