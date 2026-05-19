package backup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/djleonskennedy/ferry/internal/paths"
)

func TestCreatePreservesRelPaths(t *testing.T) {
	t.Setenv("FERRY_HOME", t.TempDir())
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "apps/api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, ".env"), []byte("ROOT"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "apps/api/.env.local"), []byte("API"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir, err := Create("proj", src, []string{".env", "apps/api/.env.local", "missing.env"})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, ".env")); string(got) != "ROOT" {
		t.Errorf(".env content = %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "apps/api/.env.local")); string(got) != "API" {
		t.Errorf("api content = %q", got)
	}
	// missing.env should not exist (silently skipped, not an error)
	if _, err := os.Stat(filepath.Join(dir, "missing.env")); !os.IsNotExist(err) {
		t.Errorf("missing.env should not exist in backup, got %v", err)
	}
}

func TestPruneKeepsNewest(t *testing.T) {
	t.Setenv("FERRY_HOME", t.TempDir())
	root := paths.BackupRoot("proj")
	stamps := []string{
		"2026-01-01T00-00-00Z",
		"2026-02-01T00-00-00Z",
		"2026-03-01T00-00-00Z",
		"2026-04-01T00-00-00Z",
		"2026-05-01T00-00-00Z",
	}
	for _, s := range stamps {
		if err := os.MkdirAll(filepath.Join(root, s), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := Prune("proj", 2); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d, want 2", len(entries))
	}
	names := []string{entries[0].Name(), entries[1].Name()}
	for _, want := range []string{"2026-04-01T00-00-00Z", "2026-05-01T00-00-00Z"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Errorf("expected %s to be kept, got %v", want, names)
		}
	}
}

func TestPruneZeroKeepsAll(t *testing.T) {
	t.Setenv("FERRY_HOME", t.TempDir())
	root := paths.BackupRoot("proj")
	for _, s := range []string{"a", "b", "c"} {
		if err := os.MkdirAll(filepath.Join(root, s), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := Prune("proj", 0); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 3 {
		t.Errorf("keep=0 should be a no-op, got %d", len(entries))
	}
}
