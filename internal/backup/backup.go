// Package backup copies files to ~/.ferry/backups/<project>/<timestamp>/
// before ferry overwrites them, and prunes old timestamp dirs.
package backup

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/djleonskennedy/ferry/internal/paths"
)

// Stamp returns a filesystem-safe RFC3339 timestamp.
func Stamp(t time.Time) string {
	s := t.UTC().Format(time.RFC3339)
	return strings.ReplaceAll(s, ":", "-")
}

// Create copies the given relative paths from destRoot into a new timestamped
// backup directory. Returns the absolute backup directory path.
// Missing source files are silently skipped (nothing to back up).
func Create(project, destRoot string, relPaths []string) (string, error) {
	stamp := Stamp(time.Now())
	dir := paths.BackupDir(project, stamp)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	for _, rel := range relPaths {
		src := filepath.Join(destRoot, filepath.FromSlash(rel))
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return "", err
		}
		if err := copyFile(src, dst); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
	}
	return dir, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// Prune keeps the newest `keep` timestamp directories for project and removes
// the rest. keep<=0 disables pruning.
func Prune(project string, keep int) error {
	if keep <= 0 {
		return nil
	}
	root := paths.BackupRoot(project)
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	dirs := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	for i := keep; i < len(dirs); i++ {
		if err := os.RemoveAll(filepath.Join(root, dirs[i])); err != nil {
			return err
		}
	}
	return nil
}
