package snapshot

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"

	"github.com/djleonskennedy/ferry/internal/cliutil"
	"github.com/djleonskennedy/ferry/internal/config"
	"github.com/djleonskennedy/ferry/internal/crypto"
)

// repoFixture sets up FERRY_HOME and a fresh repo with given files.
type repoFixture struct {
	Root   string
	Cfg    *config.ProjectConfig
	Ident  *age.X25519Identity
	Recips []age.Recipient
}

func setupFixture(t *testing.T, files map[string]string) *repoFixture {
	t.Helper()
	t.Setenv("FERRY_HOME", t.TempDir())
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.DefaultProject("testproj")
	for rel := range files {
		cfg.Files = append(cfg.Files, config.FileEntry{Path: rel, Required: true})
	}
	id, err := crypto.GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	return &repoFixture{
		Root:   root,
		Cfg:    cfg,
		Ident:  id,
		Recips: []age.Recipient{crypto.RecipientFromIdentity(id)},
	}
}

func TestCreateAndApplyRoundtrip(t *testing.T) {
	fx := setupFixture(t, map[string]string{
		".env":                "ROOT=1",
		"apps/api/.env.local": "API=2",
	})
	res, err := Create(CreateOpts{
		Root:       fx.Root,
		ProjectCfg: fx.Cfg,
		Recipients: fx.Recips,
		Now:        time.Date(2026, 5, 19, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.Version != 1 {
		t.Errorf("Version = %d, want 1", res.Version)
	}

	// Remove the files and apply.
	os.Remove(filepath.Join(fx.Root, ".env"))
	os.Remove(filepath.Join(fx.Root, "apps/api/.env.local"))
	ar, err := Apply(ApplyOpts{
		Project:  "testproj",
		Dest:     fx.Root,
		Identity: fx.Ident,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(ar.Restored) != 2 {
		t.Errorf("Restored = %v", ar.Restored)
	}
	got, _ := os.ReadFile(filepath.Join(fx.Root, ".env"))
	if string(got) != "ROOT=1" {
		t.Errorf(".env content = %q", got)
	}
}

func TestApplyAbortsOnDiff(t *testing.T) {
	fx := setupFixture(t, map[string]string{".env": "ORIG"})
	if _, err := Create(CreateOpts{
		Root: fx.Root, ProjectCfg: fx.Cfg, Recipients: fx.Recips,
	}); err != nil {
		t.Fatal(err)
	}
	// Modify on disk.
	if err := os.WriteFile(filepath.Join(fx.Root, ".env"), []byte("CHANGED"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Apply(ApplyOpts{Project: "testproj", Dest: fx.Root, Identity: fx.Ident})
	if !errors.Is(err, cliutil.ErrAbort) {
		t.Fatalf("expected ErrAbort, got %v", err)
	}
}

func TestApplyForceBacksUpAndOverwrites(t *testing.T) {
	fx := setupFixture(t, map[string]string{".env": "ORIG"})
	if _, err := Create(CreateOpts{
		Root: fx.Root, ProjectCfg: fx.Cfg, Recipients: fx.Recips,
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fx.Root, ".env"), []byte("LOCAL"), 0o600); err != nil {
		t.Fatal(err)
	}
	ar, err := Apply(ApplyOpts{
		Project:       "testproj",
		Dest:          fx.Root,
		Identity:      fx.Ident,
		Force:         true,
		BackupOnForce: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ar.BackedUp) != 1 || ar.BackedUp[0] != ".env" {
		t.Errorf("BackedUp = %v", ar.BackedUp)
	}
	// Backed-up content is the LOCAL version
	backed, _ := os.ReadFile(filepath.Join(ar.BackupDir, ".env"))
	if string(backed) != "LOCAL" {
		t.Errorf("backup content = %q", backed)
	}
	// On-disk is now ORIG
	got, _ := os.ReadFile(filepath.Join(fx.Root, ".env"))
	if string(got) != "ORIG" {
		t.Errorf("restored = %q", got)
	}
}

func TestCreateRefusesPlaintextWhenRequired(t *testing.T) {
	fx := setupFixture(t, map[string]string{".env": "X"})
	fx.Cfg.Encryption.Required = true
	_, err := Create(CreateOpts{
		Root: fx.Root, ProjectCfg: fx.Cfg, AllowPlain: true, // no recipients, plain requested
	})
	if !errors.Is(err, cliutil.ErrKey) {
		t.Fatalf("expected ErrKey, got %v", err)
	}
}

func TestNextVersionAndUpdateLatest(t *testing.T) {
	fx := setupFixture(t, map[string]string{".env": "A"})
	for i := 1; i <= 3; i++ {
		if _, err := Create(CreateOpts{Root: fx.Root, ProjectCfg: fx.Cfg, Recipients: fx.Recips}); err != nil {
			t.Fatal(err)
		}
	}
	n, err := ReadLatest("testproj")
	if err != nil || n != 3 {
		t.Fatalf("ReadLatest = %d, %v; want 3", n, err)
	}
	next, err := NextVersion("testproj")
	if err != nil || next != 4 {
		t.Fatalf("NextVersion = %d, %v; want 4", next, err)
	}
}

func TestPruneKeepsLatest(t *testing.T) {
	fx := setupFixture(t, map[string]string{".env": "A"})
	for i := 1; i <= 5; i++ {
		if _, err := Create(CreateOpts{Root: fx.Root, ProjectCfg: fx.Cfg, Recipients: fx.Recips}); err != nil {
			t.Fatal(err)
		}
	}
	if err := Prune("testproj", 3); err != nil {
		t.Fatal(err)
	}
	versions, err := listVersions("testproj")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 3 {
		t.Errorf("versions = %v, want 3 entries", versions)
	}
	if !contains(versions, 5) {
		t.Error("latest v5 must be retained")
	}
}

func TestListReturnsAllSnapshots(t *testing.T) {
	fx := setupFixture(t, map[string]string{".env": "A"})
	for i := 1; i <= 2; i++ {
		if _, err := Create(CreateOpts{
			Root: fx.Root, ProjectCfg: fx.Cfg, Recipients: fx.Recips,
			Message: "msg-" + string(rune('A'+i-1)),
		}); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := List("testproj")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries", len(entries))
	}
	if !entries[1].IsLatest {
		t.Error("v2 should be latest")
	}
	if !strings.HasPrefix(entries[0].Message, "msg-") {
		t.Errorf("message lost: %q", entries[0].Message)
	}
}

func TestDiffDetectsDrift(t *testing.T) {
	fx := setupFixture(t, map[string]string{".env": "ORIG"})
	if _, err := Create(CreateOpts{Root: fx.Root, ProjectCfg: fx.Cfg, Recipients: fx.Recips}); err != nil {
		t.Fatal(err)
	}
	d, err := Diff(DiffOpts{Project: "testproj", Dest: fx.Root})
	if err != nil {
		t.Fatal(err)
	}
	if d.HasDrift() {
		t.Error("clean tree should not drift")
	}
	if err := os.WriteFile(filepath.Join(fx.Root, ".env"), []byte("CHANGED"), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err = Diff(DiffOpts{Project: "testproj", Dest: fx.Root})
	if err != nil {
		t.Fatal(err)
	}
	if !d.HasDrift() {
		t.Error("expected drift")
	}
}

func contains(xs []int, want int) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
