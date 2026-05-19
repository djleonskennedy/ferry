package safety

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/djleonskennedy/ferry/internal/manifest"
)

func write(t *testing.T, p, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	h, _, err := manifest.HashFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestCompareToManifest(t *testing.T) {
	root := t.TempDir()
	envHash := write(t, filepath.Join(root, ".env"), "FOO=1")
	apiHash := write(t, filepath.Join(root, "apps/api/.env.local"), "API=2")
	_ = apiHash // we'll intentionally lie about this one below
	// modified: on-disk has API=2 but manifest claims a different hash
	m := &manifest.Manifest{
		Files: []manifest.ManifestFile{
			{Path: ".env", SHA256: envHash},
			{Path: "apps/api/.env.local", SHA256: "0000"},
			{Path: "missing.env", SHA256: "deadbeef"},
		},
	}
	cmp, err := CompareToManifest(root, m)
	if err != nil {
		t.Fatal(err)
	}
	if cmp[0].Status != Same {
		t.Errorf(".env: want Same, got %s", cmp[0].Status)
	}
	if cmp[1].Status != Modified {
		t.Errorf("api: want Modified, got %s", cmp[1].Status)
	}
	if cmp[2].Status != Missing {
		t.Errorf("missing: want Missing, got %s", cmp[2].Status)
	}
	if !HasModified(cmp) {
		t.Error("HasModified should be true")
	}
	if mods := ModifiedPaths(cmp); len(mods) != 1 || mods[0] != "apps/api/.env.local" {
		t.Errorf("ModifiedPaths = %v", mods)
	}
}
