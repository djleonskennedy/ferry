package snapshot

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/djleonskennedy/ferry/internal/paths"
)

// UpdateLatest atomically points latest -> v<version> in the project dir.
// On POSIX this is a symlink swap via Rename. On Windows it falls back to
// writing latest.txt with the version dir name.
func UpdateLatest(project string, version int) error {
	link := paths.LatestLink(project)
	target := fmt.Sprintf("v%d", version)

	if runtime.GOOS == "windows" {
		return os.WriteFile(link+".txt", []byte(target), 0o600)
	}

	tmp := link + fmt.Sprintf(".tmp.%d", os.Getpid())
	_ = os.Remove(tmp)
	if err := os.Symlink(target, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, link); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// ReadLatest returns the version pointed to by latest, or fs.ErrNotExist.
func ReadLatest(project string) (int, error) {
	link := paths.LatestLink(project)
	if fi, err := os.Lstat(link); err == nil && fi.Mode()&fs.ModeSymlink != 0 {
		target, err := os.Readlink(link)
		if err != nil {
			return 0, err
		}
		n, ok := parseV(filepath.Base(target))
		if !ok {
			return 0, fmt.Errorf("malformed latest symlink: %q", target)
		}
		return n, nil
	}
	if data, err := os.ReadFile(link + ".txt"); err == nil {
		n, ok := parseV(strings.TrimSpace(string(data)))
		if !ok {
			return 0, errors.New("malformed latest.txt")
		}
		return n, nil
	}
	return 0, fs.ErrNotExist
}
