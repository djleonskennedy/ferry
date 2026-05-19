// Package snapshot orchestrates create/apply/list/diff for ferry snapshots.
package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/djleonskennedy/ferry/internal/paths"
)

// SnapshotsRoot is the project's snapshot directory.
func SnapshotsRoot(project string) string { return paths.SnapshotsRoot(project) }

// VersionDir is vN inside SnapshotsRoot.
func VersionDir(project string, n int) string { return paths.VersionDir(project, n) }

// EnvAge returns the path to the encrypted archive inside vN/.
func EnvAge(project string, n int) string {
	return filepath.Join(paths.VersionDir(project, n), "env.age")
}

// EnvTar returns the plaintext fallback path (used only when explicitly opted in).
func EnvTar(project string, n int) string {
	return filepath.Join(paths.VersionDir(project, n), "env.tar")
}

// ManifestPath returns vN/manifest.toml.
func ManifestPath(project string, n int) string {
	return filepath.Join(paths.VersionDir(project, n), "manifest.toml")
}

// listVersions returns existing version numbers in ascending order.
func listVersions(project string) ([]int, error) {
	root := paths.SnapshotsRoot(project)
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n, ok := parseV(e.Name())
		if !ok {
			continue
		}
		out = append(out, n)
	}
	sort.Ints(out)
	return out, nil
}

// NextVersion returns 1 + max existing version (1 if none exist).
func NextVersion(project string) (int, error) {
	versions, err := listVersions(project)
	if err != nil {
		return 0, err
	}
	if len(versions) == 0 {
		return 1, nil
	}
	return versions[len(versions)-1] + 1, nil
}

func parseV(name string) (int, bool) {
	if !strings.HasPrefix(name, "v") {
		return 0, false
	}
	n, err := strconv.Atoi(name[1:])
	if err != nil {
		return 0, false
	}
	return n, true
}

// Prune deletes oldest snapshots to keep only `keep` (newest). keep<=0 = no-op.
// If `latest` points to a deleted version, the symlink is left as-is and a
// caller-visible error is returned (shouldn't happen if Prune runs after
// UpdateLatest in Create's flow).
func Prune(project string, keep int) error {
	if keep <= 0 {
		return nil
	}
	versions, err := listVersions(project)
	if err != nil {
		return err
	}
	if len(versions) <= keep {
		return nil
	}
	toDelete := versions[:len(versions)-keep]
	latest, err := ReadLatest(project)
	if err == nil {
		for _, n := range toDelete {
			if n == latest {
				return fmt.Errorf("refusing to prune v%d: it is the latest", n)
			}
		}
	}
	for _, n := range toDelete {
		if err := os.RemoveAll(VersionDir(project, n)); err != nil {
			return err
		}
	}
	return nil
}

// ensureDir mkdir-p's path with 0700.
func ensureDir(path string) error { return os.MkdirAll(path, 0o700) }
