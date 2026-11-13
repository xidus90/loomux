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
	if !reflect.DeepEqual(names, []string{"go", "python", "gdscript", "csharp", "cpp", "java", "php", "javascript", "typescript", "css", "scss", "html"}) {
		t.Fatalf("Languages() = %v, want [go python gdscript csharp cpp java php javascript typescript css scss html]", names)
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
	// Python's with the tree-sitter runtime it parses on (cpp, csharp, css and
	// gdscript sort before go, html, java, javascript and php between go and
	// python, scss between python and typescript, typescript last). A graph
	// built by a binary that knew Go alone reads as foreign and is rebuilt.
	if got, want := all.Version(), "cpp/1@"+treesitter.Parser+"+csharp/1@"+treesitter.Parser+"+css/1@"+treesitter.Parser+"+gdscript/7@"+treesitter.Parser+"+"+golang.Version+"+html/1@"+treesitter.Parser+"+java/1@"+treesitter.Parser+"+javascript/1@"+treesitter.Parser+"+php/1@"+treesitter.Parser+"+python/2@"+treesitter.Parser+"+scss/1@"+treesitter.Parser+"+typescript/1@"+treesitter.Parser; got != want {
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
		"src/Http/Kernel.php": "php", "index.php": "php",
		"src/a.js": "javascript", "a.mjs": "javascript", "a.cjs": "javascript", "ui/A.jsx": "javascript",
		"src/a.ts": "typescript", "a.mts": "typescript", "a.cts": "typescript", "ui/A.tsx": "typescript", "types/a.d.ts": "typescript",
		"web/site.css": "css", "web/_base.scss": "scss",
		"web/index.html": "html", "a.htm": "html",
	} {
		l, ok := all.For(rel)
		if !ok || l.Name() != name {
			t.Errorf("For(%q) = %v, %v; want the %s extractor", rel, l, ok, name)
		}
	}
	// The extension is matched exactly: the Go toolchain ignores A/B.GO, and
	// so does the graph. A .pyi stub repeats the module it describes. The
	// indented .sass and .less are other grammars, .xhtml is XML.
	for _, rel := range []string{"Makefile", "a/b.go.txt", "A/B.GO", "A/B.PY", "a/b.pyi", "g/A.GD", "g/x.import", "c/Game.CS", "c/Game.csproj", "c/Game.sln", "src/a.c", "src/a.CPP", "a.cpp.txt", "A.JAVA", "a/B.class", "lib.jar", "A.PHP", "a.phtml", "a.php.txt", "A.JS", "a.TS", "a.vue", "a.svelte", "a.js.map", "A.CSS", "A.SCSS", "a.sass", "a.less", "A.HTML", "a.xhtml"} {
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
	want := []string{".c++", ".cc", ".cjs", ".cpp", ".cs", ".css", ".cts", ".cxx", ".gd", ".go", ".godot", ".h", ".h++", ".hh", ".hpp", ".htm", ".html", ".hxx", ".inl", ".ipp", ".java", ".js", ".jsx", ".mjs", ".mts", ".php", ".py", ".scss", ".tpp", ".tres", ".ts", ".tscn", ".tsx", ".uid"}
	if got := all.Extensions(); !reflect.DeepEqual(got, want) {
		t.Fatalf("all.Extensions() = %v, want %v", got, want)
	}
	if got, want := all.Extensions(), sourceset.Extensions(); !reflect.DeepEqual(got, want) {
		t.Fatalf("all.Extensions() = %v, sourceset.Extensions() = %v; they must agree", got, want)
	}
}
