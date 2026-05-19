package archive

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkEntry(path, content string) Entry {
	data := []byte(content)
	return Entry{
		Path: path,
		Mode: 0o600,
		Size: int64(len(data)),
		Open: func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil },
	}
}

func TestPackUnpackRoundtrip(t *testing.T) {
	var buf bytes.Buffer
	entries := []Entry{
		mkEntry("apps/web/.env", "WEB=1"),
		mkEntry(".env", "ROOT=2"),
		mkEntry("apps/api/.env.local", "API=3"),
	}
	if err := Pack(&buf, entries); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	got := map[string]string{}
	err := Unpack(&buf, dest, UnpackOpts{
		OnFile: func(r UnpackResult) error {
			b, err := os.ReadFile(r.AbsPath)
			if err != nil {
				return err
			}
			got[r.RelPath] = string(b)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		".env":                 "ROOT=2",
		"apps/web/.env":        "WEB=1",
		"apps/api/.env.local":  "API=3",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: got %q, want %q", k, got[k], v)
		}
	}
}

func TestPackIsDeterministic(t *testing.T) {
	build := func() []byte {
		var buf bytes.Buffer
		entries := []Entry{mkEntry("b.env", "B"), mkEntry("a.env", "A"), mkEntry("c.env", "C")}
		if err := Pack(&buf, entries); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	a := build()
	b := build()
	if !bytes.Equal(a, b) {
		t.Fatal("Pack output is not deterministic")
	}
}

func TestPackOrdering(t *testing.T) {
	var buf bytes.Buffer
	entries := []Entry{mkEntry("z.env", "z"), mkEntry("a.env", "a"), mkEntry("m.env", "m")}
	if err := Pack(&buf, entries); err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(&buf)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, hdr.Name)
	}
	want := []string{"a.env", "m.env", "z.env"}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("entries[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestUnpackRejectsTraversal(t *testing.T) {
	// Hand-craft a tar with a "../escape" entry.
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{Typeflag: tar.TypeReg, Name: "../escape", Mode: 0o600, Size: 3}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	tw.Write([]byte("bad"))
	tw.Close()

	dest := t.TempDir()
	err := Unpack(&buf, dest, UnpackOpts{})
	if err == nil || !strings.Contains(err.Error(), "traversal") {
		t.Fatalf("expected traversal error, got %v", err)
	}
}

func TestUnpackRejectsAbsolute(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{Typeflag: tar.TypeReg, Name: "/etc/passwd", Mode: 0o600, Size: 3}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	tw.Write([]byte("bad"))
	tw.Close()

	dest := t.TempDir()
	err := Unpack(&buf, dest, UnpackOpts{})
	if err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("expected absolute path error, got %v", err)
	}
}

func TestUnpackCreatesNestedDirs(t *testing.T) {
	var buf bytes.Buffer
	if err := Pack(&buf, []Entry{mkEntry("a/b/c/.env", "deep")}); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := Unpack(&buf, dest, UnpackOpts{}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dest, "a", "b", "c", ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "deep" {
		t.Errorf("got %q, want deep", b)
	}
}
