package manifest

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.toml")
	now := time.Date(2026, 5, 19, 14, 30, 22, 0, time.UTC)
	in := &Manifest{
		Version:      3,
		Project:      "demo",
		CreatedAt:    now,
		KeyID:        "default",
		Message:      "rotated db password",
		FerryVersion: "v0.1.0",
		Files: []ManifestFile{
			{Path: ".env", Size: 12, SHA256: "abc", Mode: 0o600},
		},
	}
	if err := Write(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if out.Version != 3 || out.Project != "demo" || out.Message != "rotated db password" {
		t.Errorf("roundtrip lost values: %+v", out)
	}
	if !out.CreatedAt.Equal(now) {
		t.Errorf("createdAt = %v, want %v", out.CreatedAt, now)
	}
	f := out.Find(".env")
	if f == nil || f.SHA256 != "abc" || f.Mode != 0o600 {
		t.Errorf("Find(.env) = %+v", f)
	}
	if out.Find("missing") != nil {
		t.Error("Find(missing) should be nil")
	}
}
