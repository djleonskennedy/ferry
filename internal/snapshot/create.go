package snapshot

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"filippo.io/age"

	"github.com/djleonskennedy/ferry/internal/archive"
	"github.com/djleonskennedy/ferry/internal/cliutil"
	"github.com/djleonskennedy/ferry/internal/config"
	"github.com/djleonskennedy/ferry/internal/crypto"
	"github.com/djleonskennedy/ferry/internal/manifest"
)

type CreateOpts struct {
	Root         string                // project root on disk
	ProjectCfg   *config.ProjectConfig // ferry.toml contents
	Recipients   []age.Recipient       // empty => plaintext (only allowed when AllowPlaintext)
	AllowPlain   bool                  // honors encryption.required=false
	Message      string                // optional, recorded in manifest
	FerryVersion string                // build version stamped into manifest
	Now          time.Time             // injected for tests; zero => time.Now()
}

type CreateResult struct {
	Version  int
	Dir      string
	Manifest *manifest.Manifest
	Plain    bool
}

// Create produces a new vN snapshot under ~/.ferry/snapshots/<project>/.
func Create(opts CreateOpts) (*CreateResult, error) {
	cfg := opts.ProjectCfg
	if cfg == nil || cfg.Project.Name == "" {
		return nil, fmt.Errorf("%w: missing project config", cliutil.ErrUsage)
	}
	plain := opts.AllowPlain && !cfg.Encryption.Required && len(opts.Recipients) == 0
	if !plain && len(opts.Recipients) == 0 {
		return nil, fmt.Errorf("%w: no recipient and plaintext not allowed", cliutil.ErrKey)
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	files, mfiles, err := gatherFiles(opts.Root, cfg.Files)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%w: no files to snapshot", cliutil.ErrUsage)
	}

	version, err := NextVersion(cfg.Project.Name)
	if err != nil {
		return nil, err
	}
	dir := VersionDir(cfg.Project.Name, version)
	if err := ensureDir(dir); err != nil {
		return nil, err
	}

	archivePath := EnvAge(cfg.Project.Name, version)
	if plain {
		archivePath = EnvTar(cfg.Project.Name, version)
	}
	if err := writeArchive(archivePath, files, opts.Recipients, plain); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}

	m := &manifest.Manifest{
		Version:      version,
		Project:      cfg.Project.Name,
		CreatedAt:    now,
		KeyID:        cfg.Encryption.KeyID,
		Message:      opts.Message,
		FerryVersion: opts.FerryVersion,
		Files:        mfiles,
	}
	if err := manifest.Write(ManifestPath(cfg.Project.Name, version), m); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	if err := UpdateLatest(cfg.Project.Name, version); err != nil {
		return nil, err
	}
	return &CreateResult{Version: version, Dir: dir, Manifest: m, Plain: plain}, nil
}

// gatherFiles validates each [[files]] entry against the filesystem and
// computes sha256 + size + mode for the manifest. Returns archive entries
// in the same order as the manifest files (sorted by path).
func gatherFiles(root string, entries []config.FileEntry) ([]archive.Entry, []manifest.ManifestFile, error) {
	var (
		ar    []archive.Entry
		mflds []manifest.ManifestFile
	)
	for _, e := range entries {
		abs := filepath.Join(root, filepath.FromSlash(e.Path))
		fi, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				if e.Required {
					return nil, nil, fmt.Errorf("%w: required file missing: %s", cliutil.ErrUsage, e.Path)
				}
				continue
			}
			return nil, nil, err
		}
		hash, size, err := manifest.HashFile(abs)
		if err != nil {
			return nil, nil, err
		}
		if size != fi.Size() {
			return nil, nil, fmt.Errorf("%w: size changed during hash for %s", cliutil.ErrIntegrity, e.Path)
		}
		mode := fi.Mode().Perm()
		if mode == 0 {
			mode = 0o600
		}
		filePath := abs
		ar = append(ar, archive.Entry{
			Path: e.Path,
			Mode: mode,
			Size: size,
			Open: func() (io.ReadCloser, error) { return os.Open(filePath) },
		})
		mflds = append(mflds, manifest.ManifestFile{
			Path:   e.Path,
			Size:   size,
			SHA256: hash,
			Mode:   uint32(mode),
		})
	}
	// Sort both slices by Path for stable manifest + archive order.
	sort.Slice(ar, func(i, j int) bool { return ar[i].Path < ar[j].Path })
	sort.Slice(mflds, func(i, j int) bool { return mflds[i].Path < mflds[j].Path })
	return ar, mflds, nil
}

// writeArchive opens the destination file, optionally wraps with age, and
// pipes tar bytes through. Modes: 0600 on the encrypted file, 0600 on plain.
func writeArchive(path string, entries []archive.Entry, recipients []age.Recipient, plain bool) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	var w io.Writer = f
	var enc io.WriteCloser
	if !plain {
		enc, err = crypto.EncryptStream(f, recipients)
		if err != nil {
			return err
		}
		w = enc
	}
	if err := archive.Pack(w, entries); err != nil {
		if enc != nil {
			_ = enc.Close()
		}
		return err
	}
	if enc != nil {
		if err := enc.Close(); err != nil {
			return err
		}
	}
	return nil
}

