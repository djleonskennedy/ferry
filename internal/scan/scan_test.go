package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverFindsEnvFilesAndSkipsNoise(t *testing.T) {
	root := t.TempDir()
	// dotfile-style names
	writeFile(t, filepath.Join(root, ".env"), "FOO=1")
	writeFile(t, filepath.Join(root, ".env.local"), "BAR=2")
	writeFile(t, filepath.Join(root, "apps/api/.env.local"), "BAZ=3")
	writeFile(t, filepath.Join(root, ".envrc"), "use direnv")
	// suffix-style names (dev.env, prod.env, etc.)
	writeFile(t, filepath.Join(root, "dev.env"), "DEV=1")
	writeFile(t, filepath.Join(root, "prod.env"), "PROD=1")
	writeFile(t, filepath.Join(root, "apps/api/staging.env"), "STAGING=1")
	// noise
	writeFile(t, filepath.Join(root, "README.md"), "ignore me")
	writeFile(t, filepath.Join(root, "node_modules/pkg/.env"), "SECRET=nope")
	writeFile(t, filepath.Join(root, ".git/config"), "[core]")
	writeFile(t, filepath.Join(root, "dist/bundle/.env"), "SHOULDNT=appear")

	got, err := Discover(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		".env",
		".env.local",
		".envrc",
		"apps/api/.env.local",
		"apps/api/staging.env",
		"dev.env",
		"prod.env",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d candidates, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].RelPath != w {
			t.Errorf("candidates[%d] = %q, want %q", i, got[i].RelPath, w)
		}
	}
}

func TestDiscoverEmptyTree(t *testing.T) {
	root := t.TempDir()
	got, err := Discover(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %d candidates, want 0", len(got))
	}
}

func TestDiscoverHonorsExtraGlobs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "secrets.toml"), "")
	writeFile(t, filepath.Join(root, ".env"), "")
	got, err := Discover(root, []string{"secrets.toml"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].RelPath != "secrets.toml" {
		t.Fatalf("got %+v", got)
	}
}
