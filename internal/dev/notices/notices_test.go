package notices

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGo answers go list and go env from a scripted module world.
type fakeGo struct {
	root    string
	goroot  string
	listErr error
	envErr  error
	listOut string
	extra   string // packages go list names after the scripted ones
}

// tsModule is the module part of a gotreesitter package.
func (f *fakeGo) tsModule() string {
	ts := filepath.ToSlash(filepath.Join(f.root, "ts"))
	return `"Module": {"Path": "github.com/odvcencio/gotreesitter", "Version": "v0.55.0", "Dir": "` + ts + `"}`
}

func (f *fakeGo) run(dir string, args ...string) ([]byte, error) {
	switch strings.Join(args, " ") {
	case "env GOROOT":
		return []byte(f.goroot + "\n"), f.envErr
	case "list -deps -json ./cmd/loomux":
		if f.listErr != nil || f.listOut != "" {
			return []byte(f.listOut), f.listErr
		}
		mod := filepath.ToSlash(filepath.Join(f.root, "mod"))
		modb := filepath.ToSlash(filepath.Join(f.root, "modb"))
		ts := f.tsModule()
		// b before a and python before c: go list orders by dependency, and
		// the notice must not.
		return []byte(`{"ImportPath": "fmt", "Standard": true}
{"ImportPath": "example.com/b", "Module": {"Path": "example.com/b", "Version": "v0.1.0", "Dir": "` + modb + `"}}
{"ImportPath": "example.com/a", "Module": {"Path": "example.com/a", "Version": "v1.2.3", "Dir": "` + mod + `"}}
{"ImportPath": "example.com/a/sub", "Module": {"Path": "example.com/a", "Version": "v1.2.3", "Dir": "` + mod + `"}}
{"ImportPath": "github.com/odvcencio/gotreesitter/grammars/grammar_blobs", ` + ts + `}
{"ImportPath": "github.com/odvcencio/gotreesitter/grammars/internal/pythonruntime", ` + ts + `}
{"ImportPath": "github.com/odvcencio/gotreesitter/grammars/runtime", ` + ts + `}
{"ImportPath": "github.com/odvcencio/gotreesitter/grammars/python", ` + ts + `}
{"ImportPath": "github.com/odvcencio/gotreesitter/grammars/c", ` + ts + `}
{"ImportPath": "github.com/xidus90/loomux/cmd/loomux", "Module": {"Path": "github.com/xidus90/loomux", "Main": true}}
` + f.extra), nil
	}
	return nil, errors.New("unexpected " + strings.Join(args, " "))
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func world(t *testing.T) *fakeGo {
	t.Helper()
	root := t.TempDir()
	write := func(rel, text string) {
		path := filepath.Join(root, rel)
		must(t, os.MkdirAll(filepath.Dir(path), 0o755))
		must(t, os.WriteFile(path, []byte(text), 0o644))
	}
	write("go/LICENSE", "Go license\r\n")
	write("go/PATENTS", "Go patents\n")
	write("mod/LICENSE", "A license\n")
	write("mod/LICENSE-MIT", "A MIT license\n")
	write("mod/COPYRIGHT", "A copyright\n")
	write("mod/README.md", "not a license\n")
	write("mod/copyright_test.go", "package a // not a license either\n")
	write("modb/COPYING", "B license\n")
	write("ts/LICENSE", "TS license\n")
	write("ts/licenses/texts/MIT.txt", "MIT text\n")
	write("ts/licenses/notices/python-NOTICE.txt", "Python grammar notice\n")
	// notice_file is relative to the module's root, as gotreesitter's own
	// audit writes it for elixir and pkl.
	write("ts/licenses/grammars.json", `{"entries": [
 {"name": "python", "repo": "https://github.com/tree-sitter/tree-sitter-python", "ref": "abc", "spdx": "MIT", "copyright_holders": ["Copyright (c) 2016 Max Brunsfeld"], "notice_file": "licenses/notices/python-NOTICE.txt", "copyleft": false},
 {"name": "c", "repo": "https://github.com/tree-sitter/tree-sitter-c", "ref": "def", "spdx": "MIT", "copyright_holders": ["Copyright (c) 2014 Max Brunsfeld"], "copyleft": false},
 {"name": "gpl", "repo": "r", "ref": "x", "spdx": "GPL-3.0", "copyright_holders": [], "copyleft": true}
]}`)
	return &fakeGo{root: root, goroot: filepath.Join(root, "go")}
}

func TestRenderNamesEveryLinkedPieceOnce(t *testing.T) {
	text, err := Render("repo", world(t).run)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"## Go standard library and runtime\n", "Go license\n", "Go patents\n",
		"## example.com/a v1.2.3\n", "A license\n", "## example.com/b v0.1.0\n", "B license\n",
		"### LICENSE-MIT\n\n```\nA MIT license\n```\n", "### COPYRIGHT\n\n```\nA copyright\n```\n",
		"## github.com/odvcencio/gotreesitter v0.55.0\n", "TS license\n",
		"## tree-sitter grammar python\n", "https://github.com/tree-sitter/tree-sitter-python@abc", "Copyright (c) 2016 Max Brunsfeld", "MIT text\n", "Python grammar notice\n",
		"## tree-sitter grammar c\n", "https://github.com/tree-sitter/tree-sitter-c@def",
		"# Third-party notice: German word frequencies",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Count(text, "## example.com/a ") != 1 || strings.Contains(text, "not a license") || strings.Contains(text, "copyright_test.go") || strings.Contains(text, "\r") || strings.Contains(text, "grammar gpl") {
		t.Fatal(text)
	}
}

// The notice reads the same on every run: Go first, the modules and then the
// grammars each by name, the word frequencies last.
func TestRenderOrdersTheSections(t *testing.T) {
	text, err := Render("repo", world(t).run)
	if err != nil {
		t.Fatal(err)
	}
	last := -1
	for _, heading := range []string{
		"## Go standard library and runtime\n",
		"## example.com/a v1.2.3\n",
		"## example.com/b v0.1.0\n",
		"## github.com/odvcencio/gotreesitter v0.55.0\n",
		"## tree-sitter grammar c\n",
		"## tree-sitter grammar python\n",
		"# Third-party notice: German word frequencies",
	} {
		at := strings.Index(text, heading)
		if at <= last {
			t.Fatalf("%q at %d, after %d:\n%s", heading, at, last, text)
		}
		last = at
	}
}

func TestRenderRefusesALinkedCopyleftGrammar(t *testing.T) {
	f := world(t)
	must(t, os.WriteFile(filepath.Join(f.root, "ts", "licenses", "grammars.json"), []byte(`{"entries": [
 {"name": "c", "spdx": "MIT", "copyleft": false},
 {"name": "python", "spdx": "GPL-3.0", "copyleft": true}
]}`), 0o644))
	_, err := Render("repo", f.run)
	if err == nil || err.Error() != "grammar python: GPL-3.0 is copyleft and would bind the whole binary" {
		t.Fatal(err)
	}
}

// gotreesitter's registry package carries the scanners of every grammar, the
// copyleft ones among them, and the audit has no entry for it: linked, it
// would ship terms the notice never names.
func TestRenderRefusesGotreesittersGrammarRegistry(t *testing.T) {
	f := world(t)
	f.extra = `{"ImportPath": "github.com/odvcencio/gotreesitter/grammars", ` + f.tsModule() + "}\n"
	_, err := Render("repo", f.run)
	if err == nil || !strings.HasPrefix(err.Error(), "github.com/odvcencio/gotreesitter/grammars links every grammar in gotreesitter's registry, copyleft ones included") {
		t.Fatal(err)
	}
}

func TestRenderRefusesWhatItCannotRead(t *testing.T) {
	noGo, noEnv := errors.New("no go"), errors.New("no env")
	for name, c := range map[string]struct {
		spoil func(t *testing.T, f *fakeGo)
		want  string // in the error
		is    error  // the error wraps it, when set
	}{
		"go list fails":   {func(t *testing.T, f *fakeGo) { f.listErr = noGo }, "go list: no go", noGo},
		"go list garbles": {func(t *testing.T, f *fakeGo) { f.listOut = "{" }, "go list: unexpected EOF", io.ErrUnexpectedEOF},
		"go env fails":    {func(t *testing.T, f *fakeGo) { f.envErr = noEnv }, "go env: no env", noEnv},
		"no Go license": {func(t *testing.T, f *fakeGo) { must(t, os.Remove(filepath.Join(f.goroot, "LICENSE"))) },
			"Go standard library and runtime: no license file in ", nil},
		"no module license": {func(t *testing.T, f *fakeGo) {
			must(t, os.Remove(filepath.Join(f.root, "mod", "LICENSE")))
			must(t, os.Remove(filepath.Join(f.root, "mod", "LICENSE-MIT")))
		}, "example.com/a v1.2.3: no license file in ", nil},
		"no grammar entry": {func(t *testing.T, f *fakeGo) {
			must(t, os.WriteFile(filepath.Join(f.root, "ts", "licenses", "grammars.json"), []byte(`{"entries": []}`), 0o644))
		}, "grammar c: not in gotreesitter's license audit", nil},
		"broken grammars": {func(t *testing.T, f *fakeGo) {
			must(t, os.WriteFile(filepath.Join(f.root, "ts", "licenses", "grammars.json"), []byte(`{`), 0o644))
		}, "grammars.json: unexpected end of JSON input", nil},
		"no license text": {func(t *testing.T, f *fakeGo) {
			must(t, os.Remove(filepath.Join(f.root, "ts", "licenses", "texts", "MIT.txt")))
		}, "MIT.txt", fs.ErrNotExist},
		"no notice file": {func(t *testing.T, f *fakeGo) {
			must(t, os.Remove(filepath.Join(f.root, "ts", "licenses", "notices", "python-NOTICE.txt")))
		}, "python-NOTICE.txt", fs.ErrNotExist},
		"no grammar audit": {func(t *testing.T, f *fakeGo) {
			must(t, os.Remove(filepath.Join(f.root, "ts", "licenses", "grammars.json")))
		}, "grammars.json", fs.ErrNotExist},
		"no module dir": {func(t *testing.T, f *fakeGo) { must(t, os.RemoveAll(filepath.Join(f.root, "mod"))) },
			"mod", fs.ErrNotExist},
	} {
		f := world(t)
		c.spoil(t, f)
		_, err := Render("repo", f.run)
		if err == nil || !strings.Contains(err.Error(), c.want) || (c.is != nil && !errors.Is(err, c.is)) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}

// A license file that is listed but cannot be read stops the notice rather
// than leaving its section short.
func TestRenderRefusesALicenseFileItCannotRead(t *testing.T) {
	saved := readFile
	t.Cleanup(func() { readFile = saved })
	readFile = func(path string) ([]byte, error) {
		if filepath.Base(path) == "PATENTS" {
			return nil, os.ErrPermission
		}
		return saved(path)
	}
	if _, err := Render("repo", world(t).run); !errors.Is(err, os.ErrPermission) {
		t.Fatal(err)
	}
}

func TestGoCommandRunsGo(t *testing.T) {
	out, err := GoCommand(".", "version")
	if err != nil || !strings.Contains(string(out), "go version") {
		t.Fatalf("%q, %v", out, err)
	}
	if _, err := GoCommand(".", "no-such-subcommand"); err == nil || !strings.Contains(err.Error(), "no-such-subcommand") {
		t.Fatalf("the error keeps what go said: %v", err)
	}
}

// The release builds without cgo, so the graph the notice lists is the one
// without cgo, whatever the shell running the generator set.
func TestGoCommandAsksWithoutCgo(t *testing.T) {
	t.Setenv("CGO_ENABLED", "1")
	out, err := GoCommand(".", "env", "CGO_ENABLED")
	if err != nil || strings.TrimSpace(string(out)) != "0" {
		t.Fatalf("%q, %v", out, err)
	}
}
