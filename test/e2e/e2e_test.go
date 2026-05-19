package e2e

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djleonskennedy/ferry/cmd/ferry/commands"
	"github.com/djleonskennedy/ferry/internal/cliutil"
)

// run executes ferry as if from cwd with args. Returns combined stdout+stderr
// and the error from Execute (already typed via cliutil errors).
func run(t *testing.T, cwd string, args ...string) (string, error) {
	t.Helper()
	prev, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	root := commands.NewRoot()
	root.SetArgs(args)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	err := root.Execute()
	return buf.String(), err
}

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestHappyRoundtrip(t *testing.T) {
	ferryHome := t.TempDir()
	repo := t.TempDir()
	t.Setenv("FERRY_HOME", ferryHome)

	writeFile(t, filepath.Join(repo, ".env"), "FOO=bar")
	writeFile(t, filepath.Join(repo, "apps/api/.env.local"), "API=secret")

	if _, err := run(t, repo, "key", "generate"); err != nil {
		t.Fatalf("key generate: %v", err)
	}
	if _, err := run(t, repo, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := run(t, repo, "snapshot", "-m", "initial"); err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	// Delete files, then apply.
	os.Remove(filepath.Join(repo, ".env"))
	os.Remove(filepath.Join(repo, "apps/api/.env.local"))
	if _, err := run(t, repo, "apply"); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if got := readFile(t, filepath.Join(repo, ".env")); got != "FOO=bar" {
		t.Errorf(".env = %q", got)
	}
	if got := readFile(t, filepath.Join(repo, "apps/api/.env.local")); got != "API=secret" {
		t.Errorf("api = %q", got)
	}
}

func TestApplyAbortsThenForceBacksUp(t *testing.T) {
	ferryHome := t.TempDir()
	repo := t.TempDir()
	t.Setenv("FERRY_HOME", ferryHome)

	writeFile(t, filepath.Join(repo, ".env"), "ORIG")
	for _, args := range [][]string{
		{"key", "generate"},
		{"init"},
		{"snapshot"},
	} {
		if _, err := run(t, repo, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}

	// Modify locally.
	writeFile(t, filepath.Join(repo, ".env"), "LOCAL")

	_, err := run(t, repo, "apply")
	if !errors.Is(err, cliutil.ErrAbort) {
		t.Fatalf("expected ErrAbort, got %v", err)
	}
	if got := readFile(t, filepath.Join(repo, ".env")); got != "LOCAL" {
		t.Errorf("apply without force should not overwrite; got %q", got)
	}

	// Force.
	if _, err := run(t, repo, "apply", "--force"); err != nil {
		t.Fatalf("apply --force: %v", err)
	}
	if got := readFile(t, filepath.Join(repo, ".env")); got != "ORIG" {
		t.Errorf(".env after force = %q, want ORIG", got)
	}
	// Backup file exists with the local content.
	backupRoot := filepath.Join(ferryHome, "backups", filepath.Base(repo))
	entries, _ := os.ReadDir(backupRoot)
	if len(entries) != 1 {
		t.Fatalf("expected 1 backup dir, got %d", len(entries))
	}
	backed := readFile(t, filepath.Join(backupRoot, entries[0].Name(), ".env"))
	if backed != "LOCAL" {
		t.Errorf("backup contents = %q, want LOCAL", backed)
	}
}

func TestSecondCheckoutWorkflow(t *testing.T) {
	ferryHome := t.TempDir()
	repo1 := t.TempDir()
	t.Setenv("FERRY_HOME", ferryHome)

	writeFile(t, filepath.Join(repo1, ".env"), "FOO=bar")
	for _, args := range [][]string{
		{"key", "generate"},
		{"init"},
		{"snapshot"},
	} {
		if _, err := run(t, repo1, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}

	// Simulate a fresh checkout: copy ferry.toml, run apply with the same FERRY_HOME.
	repo2 := t.TempDir()
	if err := copyTree(filepath.Join(repo1, "ferry.toml"), filepath.Join(repo2, "ferry.toml")); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, repo2, "apply"); err != nil {
		t.Fatalf("apply in second checkout: %v", err)
	}
	if got := readFile(t, filepath.Join(repo2, ".env")); got != "FOO=bar" {
		t.Errorf("second checkout .env = %q", got)
	}
}

func TestDiffExitCodes(t *testing.T) {
	ferryHome := t.TempDir()
	repo := t.TempDir()
	t.Setenv("FERRY_HOME", ferryHome)

	writeFile(t, filepath.Join(repo, ".env"), "ORIG")
	for _, args := range [][]string{
		{"key", "generate"},
		{"init"},
		{"snapshot"},
	} {
		if _, err := run(t, repo, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	// Clean: diff returns nil.
	if _, err := run(t, repo, "diff"); err != nil {
		t.Fatalf("diff (clean): %v", err)
	}
	// Drifted: diff returns ErrDrift (exit 1).
	writeFile(t, filepath.Join(repo, ".env"), "CHANGED")
	_, err := run(t, repo, "diff")
	if !errors.Is(err, cliutil.ErrDrift) {
		t.Fatalf("expected ErrDrift, got %v", err)
	}
	if code := cliutil.ExitCodeFor(err); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestPlainRefusedWhenRequired(t *testing.T) {
	ferryHome := t.TempDir()
	repo := t.TempDir()
	t.Setenv("FERRY_HOME", ferryHome)
	writeFile(t, filepath.Join(repo, ".env"), "X")
	for _, args := range [][]string{
		{"key", "generate"},
		{"init"},
	} {
		if _, err := run(t, repo, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	_, err := run(t, repo, "snapshot", "--plain")
	if !errors.Is(err, cliutil.ErrAbort) {
		t.Fatalf("expected ErrAbort, got %v", err)
	}
}

func TestListAndMessage(t *testing.T) {
	ferryHome := t.TempDir()
	repo := t.TempDir()
	t.Setenv("FERRY_HOME", ferryHome)
	writeFile(t, filepath.Join(repo, ".env"), "X")
	for _, args := range [][]string{
		{"key", "generate"},
		{"init"},
		{"snapshot", "-m", "first"},
		{"snapshot", "-m", "second"},
	} {
		if _, err := run(t, repo, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	out, err := run(t, repo, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "v1") || !strings.Contains(out, "v2") {
		t.Errorf("list output missing versions:\n%s", out)
	}
	if !strings.Contains(out, "first") || !strings.Contains(out, "second") {
		t.Errorf("list output missing messages:\n%s", out)
	}
}

func copyTree(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
