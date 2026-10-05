package wiki

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A bundle that does not exist yet gets its frame: five files, no page, and
// no directory for pages either.
func TestInitBundleWritesTheFrame(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "wiki")
	written, err := InitBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"_schema.md", "index.md", "log.md", "audit.md", "_identities.tsv"}
	if len(written) != len(want) {
		t.Fatalf("written = %v", written)
	}
	for i, name := range want {
		if written[i] != filepath.Join(root, name) {
			t.Fatalf("written[%d] = %s, want %s", i, written[i], name)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(want) {
		t.Fatalf("the bundle holds %d entries, want %d", len(entries), len(want))
	}
	register, err := os.ReadFile(filepath.Join(root, "_identities.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if string(register) != "doc_id\trelative\tcontent_hash\trevision\n" {
		t.Fatalf("register = %q", register)
	}
	schema, err := os.ReadFile(filepath.Join(root, "_schema.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(schema), "# Schema dieses Bundles\n") {
		t.Fatalf("schema = %q", schema)
	}
}

// Every file of the frame is the reference's, byte for byte. The goldens in
// testdata/bundle were written by an oracle that calls `init_bundle` of
// src/brain/wiki/scaffold.py; the script is
// docs/.superpowers/parity/stufe-3a-orakel/bundle_oracle.py.
func TestInitBundleWritesTheReferenceBytes(t *testing.T) {
	root := t.TempDir()
	written, err := InitBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 5 {
		t.Fatalf("written = %v", written)
	}
	for _, path := range written {
		name := filepath.Base(path)
		want, err := os.ReadFile(filepath.Join("testdata", "bundle", name+".golden"))
		if err != nil {
			t.Fatalf("golden for %s: %v", name, err)
		}
		if got := readBytes(t, path); !bytes.Equal(got, want) {
			t.Errorf("%s =\n%q\nwant\n%q", name, got, want)
		}
	}
}

func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// An existing file is never rewritten, not even back to the shipped text:
// `_schema.md` is meant to be edited, and a second run must not undo that.
func TestInitBundleLeavesAnExistingFileAlone(t *testing.T) {
	root := t.TempDir()
	edited := filepath.Join(root, "_schema.md")
	if err := os.WriteFile(edited, []byte("# Our own schema\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	written, err := InitBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 4 {
		t.Fatalf("written = %v", written)
	}
	body, err := os.ReadFile(edited)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "# Our own schema\n" {
		t.Fatalf("the edited schema was rewritten: %q", body)
	}
	again, err := InitBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatalf("a second run wrote %v", again)
	}
}

// A bundle path that is taken by a file cannot hold a bundle, and the refusal
// comes before anything is written.
func TestInitBundleRefusesAPathThatIsAFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "wiki")
	if err := os.WriteFile(root, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InitBundle(root); err == nil {
		t.Fatal("InitBundle accepted a file as its directory")
	}
}

// A directory standing where the log belongs is no log: the bundle is broken,
// and the run says so instead of passing over the name as present.
func TestInitBundleRefusesADirectoryWhereAFileBelongs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "log.md", "inside"), 0o755); err != nil {
		t.Fatal(err)
	}
	written, err := InitBundle(root)
	if err == nil {
		t.Fatal("InitBundle wrote over a directory")
	}
	if !strings.Contains(err.Error(), "log.md") {
		t.Fatalf("the error does not name the file: %v", err)
	}
	if len(written) != 2 {
		t.Fatalf("written before the refusal = %v, want the two files ahead of log.md", written)
	}
}
