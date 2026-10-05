package freshness_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/sourceset"
)

// benchExtractor stands for internal/code/extract/golang.Version ("go/1" as
// of this writing). It is a literal, not an import, for the same reason this
// package carries no import of the extractor at all: the dependency
// direction is part of what this benchmark is checking.
const benchExtractor = "go/1"

// repoRoot finds this repository's own root from this test file's path,
// three directories up from internal/code/freshness -- the tree BenchmarkProbe
// measures is this repository's real file set, not a synthetic one.
func repoRoot(b *testing.B) string {
	b.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		b.Fatal("runtime.Caller could not resolve this file's path")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	if err != nil {
		b.Fatal(err)
	}
	return root
}

// BenchmarkProbe times Probe alone, over this repository's own source files,
// against a freshness record written for the tree exactly as it stands -- the
// state right after a `graph build`, where every file is trusted on size and
// mtime and Probe never has to open one. Reference: the original measures
// ~3ms for 280 files (src/graph/fingerprint.ts).
func BenchmarkProbe(b *testing.B) {
	root := repoRoot(b)
	files, err := sourceset.Stat(root)
	if err != nil {
		b.Fatal(err)
	}
	hashes := make(map[string]string, len(files))
	for _, f := range files {
		body, err := os.ReadFile(f.Abs)
		if err != nil {
			b.Fatal(err)
		}
		sum := sha256.Sum256(body)
		hashes[f.Rel] = hex.EncodeToString(sum[:])
	}
	if err := freshness.Write(root, benchExtractor, files, hashes); err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = os.Remove(freshness.Path(root)) })
	b.Logf("probing %d files", len(files))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := freshness.Probe(root, benchExtractor); err != nil {
			b.Fatal(err)
		}
	}
}
