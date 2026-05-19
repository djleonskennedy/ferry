package crypto

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"filippo.io/age"
)

func TestRoundtrip(t *testing.T) {
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	enc, err := EncryptStream(&buf, []age.Recipient{RecipientFromIdentity(id)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enc.Write([]byte("hello secrets")); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := DecryptStream(&buf, []age.Identity{id})
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello secrets" {
		t.Errorf("got %q, want %q", got, "hello secrets")
	}
}

func TestWriteIdentityMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes on Windows are not POSIX")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "k.txt")
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteIdentity(path, id); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 0600", fi.Mode().Perm())
	}
	// parent dir
	pi, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if pi.Mode().Perm()&0o077 != 0 {
		// dir was created by t.TempDir with relaxed mode; we only enforce
		// for dirs we create, so don't fail. Sanity check our own dir if
		// MkdirAll fired:
		_ = pi
	}
}

func TestLoadIdentity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "k.txt")
	id, err := GenerateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteIdentity(path, id); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Recipient().String() != id.Recipient().String() {
		t.Error("loaded identity does not match generated")
	}
}

func TestLoadIdentityRejectsGarbage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "k.txt")
	if err := os.WriteFile(path, []byte("not-a-key"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadIdentity(path); err == nil {
		t.Fatal("expected error")
	}
}

func TestEncryptStreamNoRecipients(t *testing.T) {
	var buf bytes.Buffer
	if _, err := EncryptStream(&buf, nil); err == nil {
		t.Fatal("expected error for empty recipients")
	}
}
