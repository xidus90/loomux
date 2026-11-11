package all_test

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/sourceset"
)

func TestLanguagesPutGoFirst(t *testing.T) {
	var names []string
	for _, l := range all.Languages() {
		names = append(names, l.Name())
	}
	if !reflect.DeepEqual(names, []string{"go", "python", "gdscript", "csharp", "cpp", "java"}) {
		t.Fatalf("Languages() = %v, want [go python gdscript csharp cpp java]", names)
	}
}

func TestVersionIsSortedJoin(t *testing.T) {
	var vs []string
	for _, l := range all.Languages() {
		vs = append(vs, l.Version())
	}
	sort.Strings(vs)
	if got, want := all.Version(), strings.Join(vs, "+"); got != want {
		t.Fatalf("Version() = %q, want the sorted versions joined by +: %q", got, want)
	}
	// The stamp every graph of this binary carries: Go's own version, then
	// Python's with the tree-sitter runtime it parses on (cpp, csharp and
	// gdscript sort before go, java between go and python). A graph built by
	// a binary that knew Go alone reads as foreign and is rebuilt.
	if got, want := all.Version(), "cpp/1@"+treesitter.Parser+"+csharp/1@"+treesitter.Parser+"+gdscript/7@"+treesitter.Parser+"+"+golang.Version+"+java/1@"+treesitter.Parser+"+python/2@"+treesitter.Parser; got != want {
		t.Fatalf("Version() = %q, want %q", got, want)
	}
}

func TestForPicksByExtension(t *testing.T) {
	for rel, name := range map[string]string{
		"a/b.go": "go", "main.go": "go", "a/b.py": "python", "setup.py": "python",
		"g/a.gd": "gdscript", "g/project.godot": "gdscript", "g/x.tscn": "gdscript", "g/x.tres": "gdscript", "g/x.gd.uid": "gdscript",
		"c/Game.cs": "csharp", "Program.cs": "csharp",
		"src/a.cpp": "cpp", "a.cc": "cpp", "a.cxx": "cpp", "a.c++": "cpp", "include/a.hpp": "cpp", "a.hh": "cpp",
		"a.hxx": "cpp", "a.h++": "cpp", "include/a.h": "cpp", "a.inl": "cpp", "a.ipp": "cpp", "a.tpp": "cpp",
		"src/main/java/a/App.java": "java", "Main.java": "java",
	} {
		l, ok := all.For(rel)
		if !ok || l.Name() != name {
			t.Errorf("For(%q) = %v, %v; want the %s extractor", rel, l, ok, name)
		}
	}
	// The extension is matched exactly: the Go toolchain ignores A/B.GO, and
	// so does the graph. A .pyi stub repeats the module it describes.
	for _, rel := range []string{"Makefile", "a/b.go.txt", "A/B.GO", "A/B.PY", "a/b.pyi", "g/A.GD", "g/x.import", "c/Game.CS", "c/Game.csproj", "c/Game.sln", "src/a.c", "src/a.CPP", "a.cpp.txt", "A.JAVA", "a/B.class", "lib.jar"} {
		if l, ok := all.For(rel); ok {
			t.Errorf("For(%q) = %s, want no extractor", rel, l.Name())
		}
	}
}

// The walk that lists source files lives in sourceset, which the hook path
// imports and which must therefore not import this list. This test is what
// keeps the two from drifting: a file the walk lists that no extractor
// claims would be dropped from the graph without a word.
func TestExtensionsMatchSourceset(t *testing.T) {
	want := []string{".c++", ".cc", ".cpp", ".cs", ".cxx", ".gd", ".go", ".godot", ".h", ".h++", ".hh", ".hpp", ".hxx", ".inl", ".ipp", ".java", ".py", ".tpp", ".tres", ".tscn", ".uid"}
	if got := all.Extensions(); !reflect.DeepEqual(got, want) {
		t.Fatalf("all.Extensions() = %v, want %v", got, want)
	}
	if got, want := all.Extensions(), sourceset.Extensions(); !reflect.DeepEqual(got, want) {
		t.Fatalf("all.Extensions() = %v, sourceset.Extensions() = %v; they must agree", got, want)
	}
}
