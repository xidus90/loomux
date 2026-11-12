package resolve_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestModulesNeverReadsAFileThatIsNotNamedGoMod(t *testing.T) {
	root := t.TempDir()

	// "main.go" is not on disk at all. If Modules tried to read it rather than
	// skipping it by name first, this would fail with an error instead of
	// returning an empty, nil-error result.
	mods, err := resolve.Modules(root, []string{"main.go"})
	if err != nil {
		t.Fatalf("err = %v, want nil: a non-go.mod path must never be read", err)
	}
	if len(mods) != 0 {
		t.Fatalf("got %+v, want none", mods)
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

// A nested package with its own tsconfig: comments, trailing commas and a
// paths entry with a star all read; a jsconfig beside a tsconfig loses; a
// config without aliases and an unreadable one give nothing; go.mod modules
// come first, the longest path first.
func TestModulesFromReadsTSConfigAndJSConfig(t *testing.T) {
	mods, err := resolve.ModulesFrom(map[string]string{
		"go.mod":       "module example.com/repo\n",
		"tools/go.mod": "module example.com/repo/tools\n",
		"tsconfig.json": `{
  // root config
  "compilerOptions": { "baseUrl": "." },
}`,
		"packages/app/tsconfig.json": `{
  /* nested */
  "compilerOptions": {
    "paths": { "@app/*": ["src/*",], },
  },
}`,
		"packages/app/jsconfig.json": `{"compilerOptions": {"baseUrl": "js"}}`,
		"web/jsconfig.json":          `{"compilerOptions": {"paths": {"~/*": ["./*"]}}}`,
		"plain/tsconfig.json":        `{"compilerOptions": {"strict": true}}`,
		"broken/tsconfig.json":       `{"compilerOptions": `,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []resolve.Module{
		{Dir: "tools", Path: "example.com/repo/tools"},
		{Dir: "", Path: "example.com/repo"},
		{Dir: "packages/app", TS: &resolve.TSConfig{Paths: map[string][]string{"@app/*": {"src/*"}}}},
		{Dir: "", TS: &resolve.TSConfig{BaseURL: "."}},
		{Dir: "web", TS: &resolve.TSConfig{Paths: map[string][]string{"~/*": {"./*"}}}},
	}
	if !reflect.DeepEqual(mods, want) {
		t.Fatalf("got %s", dump(mods))
	}
}

func TestModulesReadsATSConfigFromDisk(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tsconfig.json"), []byte(`{"compilerOptions":{"baseUrl":"src"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	mods, err := resolve.Modules(root, []string{"tsconfig.json", "a.ts"})
	if err != nil || len(mods) != 1 || mods[0].TS == nil || mods[0].TS.BaseURL != "src" {
		t.Fatalf("got %s, %v", dump(mods), err)
	}
}

// tsc and tsserver strip a leading byte order mark, as Windows editors write.
func TestModulesFromReadsATSConfigWithAByteOrderMark(t *testing.T) {
	mods, _ := resolve.ModulesFrom(map[string]string{"tsconfig.json": string(rune(0xfeff)) + "{\"compilerOptions\": {\"baseUrl\": \".\"}}"})
	if len(mods) != 1 || mods[0].TS == nil || mods[0].TS.BaseURL != "." {
		t.Fatalf("got %s", dump(mods))
	}
}

// A tsconfig the listing names but the disk cannot give is skipped; only a
// go.mod fails the build.
func TestModulesSkipsAnUnreadableTSConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/repo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mods, err := resolve.Modules(root, []string{"go.mod", "tsconfig.json", "web/jsconfig.json"})
	if err != nil || len(mods) != 1 || mods[0].Path != "example.com/repo" {
		t.Fatalf("got %s, %v", dump(mods), err)
	}
}

func TestModuleFile(t *testing.T) {
	for rel, want := range map[string]bool{
		"go.mod": true, "a/tsconfig.json": true, "jsconfig.json": true,
		"tsconfig.base.json": false, "package.json": false, "a.go": false,
	} {
		if resolve.ModuleFile(rel) != want {
			t.Errorf("ModuleFile(%q) != %v", rel, want)
		}
	}
}

func dump(mods []resolve.Module) string {
	var b strings.Builder
	for _, m := range mods {
		fmt.Fprintf(&b, "%+v", m)
		if m.TS != nil {
			fmt.Fprintf(&b, " TS=%+v", *m.TS)
		}
		b.WriteString("; ")
	}
	return b.String()
}
