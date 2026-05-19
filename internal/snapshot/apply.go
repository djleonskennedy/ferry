package snapshot

import (
	"fmt"
	"io"
	"os"

	"filippo.io/age"

	"github.com/djleonskennedy/ferry/internal/archive"
	"github.com/djleonskennedy/ferry/internal/backup"
	"github.com/djleonskennedy/ferry/internal/cliutil"
	"github.com/djleonskennedy/ferry/internal/crypto"
	"github.com/djleonskennedy/ferry/internal/manifest"
	"github.com/djleonskennedy/ferry/internal/safety"
)

type ApplyOpts struct {
	Project        string
	Dest           string        // where to write files (typically project root)
	Identity       age.Identity  // nil only when the snapshot is plaintext
	Version        int           // 0 = latest
	Force          bool          // overwrite locally-modified files
	BackupOnForce  bool          // create a backup tree when forcing
	BackupRetain   int           // prune to this many; 0 = keep all
}

type ApplyResult struct {
	Version     int
	Restored    []string // files extracted (newly written or overwritten)
	SkippedSame []string // files already byte-equal to the snapshot
	BackedUp    []string // files backed up before overwrite
	BackupDir   string
}

// Apply decrypts snapshot vN and restores files under opts.Dest.
// It refuses to overwrite locally-modified files without opts.Force.
func Apply(opts ApplyOpts) (*ApplyResult, error) {
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

	// Pre-flight: compare on-disk state to manifest.
	cmp, err := safety.CompareToManifest(opts.Dest, m)
	if err != nil {
		return nil, err
	}
	modified := safety.ModifiedPaths(cmp)
	if len(modified) > 0 && !opts.Force {
		return nil, fmt.Errorf("%w: %d file(s) differ from snapshot v%d:\n  %s\n(use --force to overwrite; modified files will be backed up)",
			cliutil.ErrAbort, len(modified), version, joinPaths(modified))
	}

	// Backup any modified files before we overwrite.
	res := &ApplyResult{Version: version}
	if len(modified) > 0 && opts.BackupOnForce {
		dir, err := backup.Create(opts.Project, opts.Dest, modified)
		if err != nil {
			return nil, err
		}
		res.BackupDir = dir
		res.BackedUp = modified
	}

	// Build the set of "same" files (skipped) for skip-during-unpack.
	sameSet := map[string]struct{}{}
	for _, c := range cmp {
		if c.Status == safety.Same {
			sameSet[c.Path] = struct{}{}
			res.SkippedSame = append(res.SkippedSame, c.Path)
		}
	}

	// Open the archive (encrypted or plaintext).
	archivePath := EnvAge(opts.Project, version)
	var rc io.ReadCloser
	rc, err = os.Open(archivePath)
	if err != nil {
		if os.IsNotExist(err) {
			plain := EnvTar(opts.Project, version)
			rc, err = os.Open(plain)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	defer rc.Close()

	var r io.Reader = rc
	if opts.Identity != nil {
		dr, err := crypto.DecryptStream(rc, []age.Identity{opts.Identity})
		if err != nil {
			return nil, fmt.Errorf("%w: %v", cliutil.ErrKey, err)
		}
		r = dr
	}

	err = archive.Unpack(r, opts.Dest, archive.UnpackOpts{
		Skip: func(rel string) bool {
			_, ok := sameSet[rel]
			return ok
		},
		OnFile: func(u archive.UnpackResult) error {
			if u.Skipped {
				return nil
			}
			mf := m.Find(u.RelPath)
			if mf == nil {
				return fmt.Errorf("%w: tar entry %s not in manifest", cliutil.ErrIntegrity, u.RelPath)
			}
			got, _, err := manifest.HashFile(u.AbsPath)
			if err != nil {
				return err
			}
			if got != mf.SHA256 {
				return fmt.Errorf("%w: hash mismatch for %s", cliutil.ErrIntegrity, u.RelPath)
			}
			res.Restored = append(res.Restored, u.RelPath)
			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	if opts.BackupRetain > 0 {
		_ = backup.Prune(opts.Project, opts.BackupRetain)
	}
	return res, nil
}

func joinPaths(p []string) string {
	out := ""
	for i, s := range p {
		if i > 0 {
			out += "\n  "
		}
		out += s
	}
	return out
}

