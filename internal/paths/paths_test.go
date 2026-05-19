package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFerryHomeUsesEnvVar(t *testing.T) {
	t.Setenv("FERRY_HOME", "/tmp/custom-ferry")
	if got := FerryHome(); got != "/tmp/custom-ferry" {
		t.Fatalf("FerryHome = %q, want /tmp/custom-ferry", got)
	}
}

func TestFerryHomeFallsBackToHome(t *testing.T) {
	t.Setenv("FERRY_HOME", "")
	t.Setenv("HOME", "/tmp/fake-home")
	if got := FerryHome(); got != "/tmp/fake-home/.ferry" {
		t.Fatalf("FerryHome = %q, want /tmp/fake-home/.ferry", got)
	}
}

func TestPathHelpers(t *testing.T) {
	t.Setenv("FERRY_HOME", "/tmp/fh")
	cases := map[string]string{
		"GlobalConfigFile": GlobalConfigFile(),
		"KeysDir":          KeysDir(),
		"KeyFile":          KeyFile("default"),
		"SnapshotsRoot":    SnapshotsRoot("myproj"),
		"VersionDir":       VersionDir("myproj", 3),
		"LatestLink":       LatestLink("myproj"),
		"BackupDir":        BackupDir("myproj", "2026-05-19T10-00-00Z"),
	}
	want := map[string]string{
		"GlobalConfigFile": "/tmp/fh/config.toml",
		"KeysDir":          "/tmp/fh/keys",
		"KeyFile":          "/tmp/fh/keys/default.txt",
		"SnapshotsRoot":    "/tmp/fh/snapshots/myproj",
		"VersionDir":       "/tmp/fh/snapshots/myproj/v3",
		"LatestLink":       "/tmp/fh/snapshots/myproj/latest",
		"BackupDir":        "/tmp/fh/backups/myproj/2026-05-19T10-00-00Z",
	}
	for k, got := range cases {
		if got != want[k] {
			t.Errorf("%s = %q, want %q", k, got, want[k])
		}
	}
}

func TestFindProjectRootWalksUp(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ferry.toml"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := FindProjectRoot(nested)
	if err != nil {
		t.Fatalf("FindProjectRoot: %v", err)
	}
	wantAbs, _ := filepath.Abs(root)
	if got != wantAbs {
		t.Fatalf("FindProjectRoot = %q, want %q", got, wantAbs)
	}
}

func TestFindProjectRootMissing(t *testing.T) {
	root := t.TempDir()
	if _, err := FindProjectRoot(root); err == nil {
		t.Fatal("expected ErrProjectNotFound, got nil")
	}
}
