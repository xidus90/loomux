package benchcorpus

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// committedDocs is the docs tree of this repository, seen from the package.
const committedDocs = "../../../docs"

// decodeStrict reads a committed store and refuses any key the types do not
// know, so a field left behind by a conversion cannot vanish unnoticed.
func decodeStrict(t *testing.T, name string, into any) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(committedDocs, name))
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return raw
}

// committedTimestamp is the stamp the committed matrix names, which the
// store does not hold.
func committedTimestamp(t *testing.T, matrix []byte, label string) string {
	t.Helper()
	prefix := "- **" + label + ":** "
	scanner := bufio.NewScanner(bytes.NewReader(matrix))
	for scanner.Scan() {
		if stamp, ok := strings.CutPrefix(scanner.Text(), prefix); ok {
			return stamp
		}
	}
	t.Fatalf("matrix has no %q line", label)
	return ""
}

// TestRegeneratedPagesMatchTheCommittedOnes renders matrix.md and every
// per-repo page from docs/benchmarks.json and compares them byte for byte
// with the committed files. It guards the one-off conversion of the store
// to milliseconds.
func TestRegeneratedPagesMatchTheCommittedOnes(t *testing.T) {
	var audits []*RepoAudit
	raw := decodeStrict(t, "benchmarks.json", &audits)
	var skipped []SkippedRepo
	decodeStrict(t, "benchmarks-skipped.json", &skipped)
	if len(audits) == 0 {
		t.Fatal("store holds no audits")
	}

	// The next SaveReport must not rewrite a byte of the store it merges into.
	again, err := json.MarshalIndent(audits, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, raw) {
		t.Error("benchmarks.json does not round-trip through RepoAudit unchanged")
	}

	labels := map[string]string{"en": "Last Updated", "de": "Letzte Aktualisierung"}
	for _, lang := range []string{"en", "de"} {
		dir := filepath.Join(committedDocs, lang, "benchmarks")
		matrix, err := os.ReadFile(filepath.Join(dir, "matrix.md"))
		if err != nil {
			t.Fatal(err)
		}
		report := &BenchmarkReport{Timestamp: committedTimestamp(t, matrix, labels[lang]), Repos: audits, Skipped: skipped}
		var buf bytes.Buffer
		_ = FormatMatrixMarkdown(report, lang, &buf)
		if !bytes.Equal(buf.Bytes(), matrix) {
			t.Errorf("%s/matrix.md differs from its rendering", lang)
		}

		committed, err := filepath.Glob(filepath.Join(dir, "*", "*.md"))
		if err != nil {
			t.Fatal(err)
		}
		if len(committed) != len(audits) {
			t.Errorf("%s: %d committed pages for %d audits", lang, len(committed), len(audits))
		}
		for _, a := range audits {
			page := filepath.Join(dir, LanguageSlug(a.Language, a.DetectedStacks), RepoSlug(a)+".md")
			want, err := os.ReadFile(page)
			if err != nil {
				t.Error(err)
				continue
			}
			buf.Reset()
			_ = FormatDetailMarkdown(a, lang, &buf)
			if !bytes.Equal(buf.Bytes(), want) {
				t.Errorf("%s differs from its rendering", page)
			}
		}
	}
}
