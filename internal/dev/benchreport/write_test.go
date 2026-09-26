package benchreport

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTargetsRefusesAnExistingFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bench-s-hooks.json"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Targets(dir, "bench-s-hooks"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v", err)
	}
}

func TestTargetsRefusesAMissingDirectory(t *testing.T) {
	_, _, err := Targets(filepath.Join(t.TempDir(), "nope"), "b")
	if err == nil || !strings.Contains(err.Error(), "no directory at") {
		t.Fatalf("err = %v", err)
	}
}

func TestTargetsRefusesAFileWhereTheDirectoryShouldBe(t *testing.T) {
	file := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Targets(file, "b"); err == nil || !strings.Contains(err.Error(), "no directory at") {
		t.Fatalf("err = %v", err)
	}
}

func TestWriteBothWritesLFAsGiven(t *testing.T) {
	dir := t.TempDir()
	md, js, err := Targets(dir, "b")
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteBoth(md, js, []byte("# a\n"), []byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(md)
	if string(got) != "# a\n" {
		t.Fatalf("md = %q", got)
	}
	got, _ = os.ReadFile(js)
	if string(got) != "{}\n" {
		t.Fatalf("json = %q", got)
	}
}

func TestWriteBothRemovesTheMarkdownWhenTheJSONFails(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "b.md")
	js := filepath.Join(dir, "missing", "b.json") // parent absent: the write fails
	if err := WriteBoth(md, js, []byte("x"), []byte("y")); err == nil {
		t.Fatal("no error")
	}
	if _, err := os.Stat(md); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("markdown left behind: %v", err)
	}
}

// Targets checks at the start of a run; a second run of the same minute can
// take the names while the first still measures. Neither file may then be
// overwritten.
func TestWriteBothNeverOverwritesAReport(t *testing.T) {
	for _, existing := range []string{"b.md", "b.json"} {
		t.Run(existing, func(t *testing.T) {
			dir := t.TempDir()
			md, js := filepath.Join(dir, "b.md"), filepath.Join(dir, "b.json")
			taken := filepath.Join(dir, existing)
			if err := os.WriteFile(taken, []byte("the other run"), 0o644); err != nil {
				t.Fatal(err)
			}
			err := WriteBoth(md, js, []byte("x"), []byte("y"))
			if !errors.Is(err, fs.ErrExist) {
				t.Fatalf("err = %v", err)
			}
			if got, _ := os.ReadFile(taken); string(got) != "the other run" {
				t.Fatalf("%s overwritten: %q", existing, got)
			}
			other := md
			if taken == md {
				other = js
			}
			if _, err := os.Stat(other); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("%s left behind: %v", other, err)
			}
		})
	}
}

func TestWriteBothWritesNoJSONWhenTheMarkdownFails(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "missing", "b.md") // parent absent: the write fails
	js := filepath.Join(dir, "b.json")
	if err := WriteBoth(md, js, []byte("x"), []byte("y")); err == nil {
		t.Fatal("no error")
	}
	if _, err := os.Stat(js); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("json written: %v", err)
	}
}
