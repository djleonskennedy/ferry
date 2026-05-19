// Package archive packs and unpacks tar streams used by ferry snapshots.
//
// Pack writes entries sorted by Path with zeroed mtimes for byte-stable output.
// Unpack rejects absolute paths and traversal via "..".
package archive

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry describes one file to pack.
type Entry struct {
	Path string      // forward-slash relative path inside the archive
	Mode fs.FileMode // POSIX permission bits
	Size int64
	Open func() (io.ReadCloser, error)
}

// Pack writes entries to w in deterministic order (sorted by Path).
// Mtime is zeroed so byte output is stable across runs when content is stable.
func Pack(w io.Writer, entries []Entry) error {
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	tw := tar.NewWriter(w)
	for _, e := range sorted {
		if err := writeOne(tw, e); err != nil {
			return err
		}
	}
	return tw.Close()
}

func writeOne(tw *tar.Writer, e Entry) error {
	hdr := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     e.Path,
		Mode:     int64(e.Mode.Perm()),
		Size:     e.Size,
		Format:   tar.FormatPAX,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header %s: %w", e.Path, err)
	}
	rc, err := e.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", e.Path, err)
	}
	n, err := io.Copy(tw, rc)
	rc.Close()
	if err != nil {
		return fmt.Errorf("copy %s: %w", e.Path, err)
	}
	if n != e.Size {
		return fmt.Errorf("%s: size mismatch (declared %d, wrote %d)", e.Path, e.Size, n)
	}
	return nil
}

// UnpackResult describes one extracted file.
type UnpackResult struct {
	RelPath string
	AbsPath string
	Mode    fs.FileMode
	Skipped bool // entry was present in archive but not written (skip predicate)
}

// OnFileFunc is invoked after each file is processed (written or skipped).
type OnFileFunc func(res UnpackResult) error

// UnpackOpts controls Unpack behavior.
type UnpackOpts struct {
	// Skip, when non-nil and returning true for relPath, causes the entry to be
	// drained from the tar stream without being written to disk.
	Skip func(relPath string) bool
	// OnFile is called for each regular file entry after it is processed.
	OnFile OnFileFunc
}

// Unpack reads a tar stream from r and writes files under destRoot.
func Unpack(r io.Reader, destRoot string, opts UnpackOpts) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if err := validatePath(hdr.Name); err != nil {
			return err
		}
		mode := fs.FileMode(hdr.Mode) & 0o777
		if mode == 0 {
			mode = 0o600
		}
		abs := filepath.Join(destRoot, filepath.FromSlash(hdr.Name))
		skipped := opts.Skip != nil && opts.Skip(hdr.Name)
		if skipped {
			if _, err := io.Copy(io.Discard, tr); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
				return err
			}
			if err := writeFile(abs, tr, mode); err != nil {
				return err
			}
		}
		if opts.OnFile != nil {
			if err := opts.OnFile(UnpackResult{RelPath: hdr.Name, AbsPath: abs, Mode: mode, Skipped: skipped}); err != nil {
				return err
			}
		}
	}
}

func writeFile(path string, r io.Reader, mode fs.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return err
	}
	if err := os.Chmod(path, mode); err != nil {
		return err
	}
	return nil
}

func validatePath(name string) error {
	if name == "" {
		return errors.New("empty tar entry name")
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") {
		return fmt.Errorf("absolute path not allowed in tar: %s", name)
	}
	clean := filepath.ToSlash(filepath.Clean(name))
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("path traversal not allowed in tar: %s", name)
	}
	return nil
}
