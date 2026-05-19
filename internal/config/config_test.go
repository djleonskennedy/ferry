package config

import (
	"path/filepath"
	"testing"
)

func TestProjectRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ferry.toml")
	in := &ProjectConfig{
		Project:    ProjectSection{Name: "demo"},
		Encryption: EncryptionSection{KeyID: "default", Required: true},
		Files: []FileEntry{
			{Path: ".env", Required: true},
			{Path: "apps/api/.env.local", Required: false},
		},
	}
	if err := SaveProject(path, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := LoadProject(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if out.Project.Name != "demo" {
		t.Errorf("Name = %q, want demo", out.Project.Name)
	}
	if !out.Encryption.Required || out.Encryption.KeyID != "default" {
		t.Errorf("encryption = %+v", out.Encryption)
	}
	if len(out.Files) != 2 || out.Files[0].Path != ".env" || out.Files[1].Path != "apps/api/.env.local" {
		t.Errorf("files = %+v", out.Files)
	}
}

func TestLoadProjectRequiresName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ferry.toml")
	if err := SaveProject(path, &ProjectConfig{
		Encryption: EncryptionSection{KeyID: "k"},
		Files:      []FileEntry{{Path: ".env"}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadProject(path); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestGlobalDefaults(t *testing.T) {
	t.Setenv("FERRY_HOME", t.TempDir())
	g, err := LoadGlobal()
	if err != nil {
		t.Fatal(err)
	}
	if !g.Defaults.BackupOnApply || g.Defaults.BackupRetention != 10 {
		t.Errorf("defaults = %+v", g.Defaults)
	}
	if g.Keys == nil {
		t.Error("Keys map should be initialized, not nil")
	}
}

func TestGlobalSaveLoad(t *testing.T) {
	t.Setenv("FERRY_HOME", t.TempDir())
	in := &GlobalConfig{
		Keys: map[string]KeyEntry{
			"default": {Path: "/abs/key.txt"},
			"shared":  {Path: "~/keys/shared.txt"},
		},
		Defaults: Defaults{BackupOnApply: false, BackupRetention: 5, SnapshotRetention: 3},
	}
	if err := SaveGlobal(in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := LoadGlobal()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if out.Defaults.BackupRetention != 5 || out.Defaults.SnapshotRetention != 3 {
		t.Errorf("defaults round-trip lost values: %+v", out.Defaults)
	}
	p, err := out.ResolveKeyPath("default")
	if err != nil || p != "/abs/key.txt" {
		t.Errorf("ResolveKeyPath(default) = %q, %v", p, err)
	}
	p, err = out.ResolveKeyPath("missing")
	if err != nil || p == "" {
		t.Errorf("ResolveKeyPath(missing) should fall back, got %q, %v", p, err)
	}
}
