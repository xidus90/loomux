package golang_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xidus90/loomux/internal/code/sourceset"
)

// repoRoot finds this repository's own root from this test file's path,
// four directories up from internal/code/extract/golang -- the tree these
// benchmarks measure is this repository's real file set, the same one
// docs/{en,de}/benchmarks.md's graph build/check/probe entries measure.
func repoRoot(b *testing.B) string {
	b.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		b.Fatal("runtime.Caller could not resolve this file's path")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	if err != nil {
		b.Fatal(err)
	}
	return root
}

// sources reads every .go file sourceset would hand the extractor, once,
// outside any benchmark timer.
func sources(b *testing.B, root string) []string {
	b.Helper()
	files, err := sourceset.Stat(root)
	if err != nil {
		b.Fatal(err)
	}
	out := make([]string, 0, len(files))
	for _, f := range files {
		body, err := os.ReadFile(f.Abs)
		if err != nil {
			b.Fatal(err)
		}
		out = append(out, string(body))
	}
	b.Logf("parsing %d files", len(out))
	return out
}

// BenchmarkParseSkipObjectResolution is the floor the extractor's own scope
// stack (go/ast.Object is deprecated, see extract.go) is measured against:
// go/parser alone, over this repository's real file set, with
// parser.SkipObjectResolution set so no Scope/Object bookkeeping runs.
func BenchmarkParseSkipObjectResolution(b *testing.B) {
	root := repoRoot(b)
	bodies := sources(b, root)
	fset := token.NewFileSet()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, src := range bodies {
			if _, err := parser.ParseFile(fset, "src.go", src, parser.SkipObjectResolution); err != nil {
				b.Fatalf("file %d: %v", j, err)
			}
		}
	}
}

// BenchmarkParseWithObjectResolution is the same walk without
// SkipObjectResolution -- the cost of the Scope/Object bookkeeping the
// extractor avoids by keeping its own scope stack instead.
func BenchmarkParseWithObjectResolution(b *testing.B) {
	root := repoRoot(b)
	bodies := sources(b, root)
	fset := token.NewFileSet()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, src := range bodies {
			if _, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments); err != nil {
				b.Fatalf("file %d: %v", j, err)
			}
		}
	}
}
