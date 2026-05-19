package snapshot

import (
	"fmt"

	"github.com/djleonskennedy/ferry/internal/cliutil"
	"github.com/djleonskennedy/ferry/internal/manifest"
	"github.com/djleonskennedy/ferry/internal/safety"
)

type DiffOpts struct {
	Project string
	Dest    string
	Version int // 0 = latest
}

type DiffResult struct {
	Version int
	Files   []safety.FileCompare
}

// HasDrift reports whether any file differs from the snapshot.
func (d *DiffResult) HasDrift() bool {
	for _, f := range d.Files {
		if f.Status != safety.Same {
			return true
		}
	}
	return false
}

// Diff compares files on disk under Dest against snapshot vN's manifest.
// Status-only: no contents are decrypted or printed.
func Diff(opts DiffOpts) (*DiffResult, error) {
	version := opts.Version
	if version == 0 {
		v, err := ReadLatest(opts.Project)
		if err != nil {
			return nil, fmt.Errorf("%w: no snapshot found for %s", cliutil.ErrUsage, opts.Project)
		}
		version = v
	}
	m, err := manifest.Read(ManifestPath(opts.Project, version))
	if err != nil {
		return nil, err
	}
	cmp, err := safety.CompareToManifest(opts.Dest, m)
	if err != nil {
		return nil, err
	}
	return &DiffResult{Version: version, Files: cmp}, nil
}
