// Package paths centralizes all filesystem locations used by ferry.
//
// FERRY_HOME (env var) overrides ~/.ferry when set; tests rely on this.
package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	projectConfigFile = "ferry.toml"
	globalConfigFile  = "config.toml"
)

// FerryHome returns the root directory for all ferry state.
// Honors $FERRY_HOME; falls back to $HOME/.ferry; falls back to ./.ferry
// if HOME is unset (rare).
func FerryHome() string {
	if v := os.Getenv("FERRY_HOME"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".ferry"
	}
	return filepath.Join(home, ".ferry")
}

func GlobalConfigFile() string { return filepath.Join(FerryHome(), globalConfigFile) }
func KeysDir() string          { return filepath.Join(FerryHome(), "keys") }
func KeyFile(id string) string { return filepath.Join(KeysDir(), id+".txt") }
func SnapshotsDir() string     { return filepath.Join(FerryHome(), "snapshots") }
func BackupsDir() string       { return filepath.Join(FerryHome(), "backups") }

func SnapshotsRoot(project string) string { return filepath.Join(SnapshotsDir(), project) }
func VersionDir(project string, n int) string {
	return filepath.Join(SnapshotsRoot(project), fmt.Sprintf("v%d", n))
}
func LatestLink(project string) string { return filepath.Join(SnapshotsRoot(project), "latest") }
func BackupRoot(project string) string { return filepath.Join(BackupsDir(), project) }
func BackupDir(project, stamp string) string {
	return filepath.Join(BackupRoot(project), stamp)
}

// ProjectConfigPath returns the path to ferry.toml under root.
func ProjectConfigPath(root string) string { return filepath.Join(root, projectConfigFile) }

// FindProjectRoot walks up from start looking for ferry.toml.
// Returns the directory that contains it, or ErrProjectNotFound.
func FindProjectRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, projectConfigFile)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrProjectNotFound
		}
		dir = parent
	}
}

// ErrProjectNotFound is returned when no ferry.toml exists on the path from
// cwd up to the filesystem root.
var ErrProjectNotFound = errors.New("ferry.toml not found in this directory or any parent; run `ferry init` first")
