package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/code/resolve"
)

func TestModulesReadsOnlyTheModuleDirective(t *testing.T) {
	root := t.TempDir()
	body := "// a comment\nmodule example.com/repo\n\ngo 1.25.0\n\nrequire github.com/x/y v1.2.3\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	mods, err := resolve.Modules(root, []string{"go.mod"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Path != "example.com/repo" || mods[0].Dir != "" {
		t.Fatalf("got %+v, want one module example.com/repo at the root", mods)
	}
}

func TestModulesFindsANestedGoMod(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(rel, body string) {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/repo\n")
	write("tools/go.mod", "module example.com/repo/tools\n")

	mods, err := resolve.Modules(root, []string{"go.mod", "tools/go.mod"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 2 {
		t.Fatalf("got %+v, want both modules", mods)
	}
	var tools resolve.Module
	for _, m := range mods {
		if m.Dir == "tools" {
			tools = m
		}
	}
	if tools.Path != "example.com/repo/tools" {
		t.Fatalf("got %+v, want the nested module at tools", mods)
	}
}

func TestModulesSkipsNonGoModFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/repo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A real caller hands Modules the whole file list of a repository walk,
	// not a pre-filtered one; only "go.mod" is a candidate.
	mods, err := resolve.Modules(root, []string{"go.mod", "main.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 {
		t.Fatalf("got %+v, want the one go.mod, main.go is not a module file", mods)
	}
}

func TestModulesPropagatesAReadError(t *testing.T) {
	root := t.TempDir()

	// The file list names a go.mod that is not actually on disk -- a stale
	// listing, which Modules must report rather than silently skip.
	if _, err := resolve.Modules(root, []string{"go.mod"}); err == nil {
		t.Fatal("want an error for a go.mod that does not exist")
	}
}

func TestModulesIgnoresAGoModWithoutADirective(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("go 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mods, err := resolve.Modules(root, []string{"go.mod"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 0 {
		t.Fatalf("got %+v, want none: a file without a module directive declares nothing", mods)
	}
}
