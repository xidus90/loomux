package benchsearch

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// corpusDir is the checked-in corpus v1, as an absolute path.
func corpusDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "testdata", "bench", "search", "v1"))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// copyCorpus gives a test its own corpus v1 to break.
func copyCorpus(t *testing.T) string {
	t.Helper()
	stand := filepath.Join(t.TempDir(), "v1")
	if err := os.CopyFS(stand, os.DirFS(corpusDir(t))); err != nil {
		t.Fatal(err)
	}
	return stand
}

func writeYAML(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "questions.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func problemsOf(t *testing.T, err error) Problems {
	t.Helper()
	var p Problems
	if !errors.As(err, &p) {
		t.Fatalf("err = %v, want Problems", err)
	}
	return p
}

func TestDefaultShape(t *testing.T) {
	shape := DefaultShape()
	if shape[Exact] != 13 || shape[Paraphrase] != 13 || shape[Mixed] != 10 || shape[CrossLingual] != 14 {
		t.Fatalf("shape = %v", shape)
	}
}

func TestProblemsJoinsOnePerLine(t *testing.T) {
	if got := (Problems{"a", "b"}).Error(); got != "a\nb" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadQuestionsReadsTheCorpusSet(t *testing.T) {
	qs, err := LoadQuestions(filepath.Join(corpusDir(t), "questions.yaml"), DefaultShape())
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) != 50 || qs[0].ID != "c01" || qs[0].Kind != Exact {
		t.Fatalf("got %d, first %+v", len(qs), qs[0])
	}
	if !filepath.IsAbs(qs[0].Expect) || filepath.Base(qs[0].Expect) != "baustatik-04.md" {
		t.Fatalf("expect = %q", qs[0].Expect)
	}
	if qs[0].Query != "Eurocode 58 Teilnormen" || qs[0].Evidence != "Es gibt insgesamt 58 Teilnormen." || qs[0].Note == "" {
		t.Fatalf("first = %+v", qs[0])
	}
}

func TestLoadQuestionsCollectsEveryProblem(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "Es gibt\ninsgesamt 58 Teilnormen.")
	writeYAML(t, dir, `
- {id: q1, sort: exakt, query: a, expect: a.md, beleg: "Es gibt insgesamt 58 Teilnormen."}
- {id: q1, sort: exakt, query: b, expect: a.md, beleg: "fehlt"}
- {id: q3, sort: raten, query: c, expect: a.md, beleg: "Es gibt"}
- {id: q4, query: d, expect: nope.md, beleg: x}
- {id: q5, sort: exakt, query: e, expect: nope.md, beleg: x}
`)
	_, err := LoadQuestions(filepath.Join(dir, "questions.yaml"), Shape{Exact: 1})
	p := problemsOf(t, err)
	want := []string{
		"q1: duplicate id \"q1\"",
		"q1: evidence not found verbatim in " + filepath.Join(dir, "a.md"),
		"q3: unknown sort \"raten\"",
		"q4: missing field \"sort\"",
		"q5: expect does not exist: " + filepath.Join(dir, "nope.md"),
		"exakt: 3 questions, expected 1",
	}
	if !slices.Equal(p, want) {
		t.Fatalf("problems:\n%s", strings.Join(p, "\n"))
	}
}

// The reference names the sort before the target, both for the same entry.
func TestLoadQuestionsNamesTheSortBeforeTheTarget(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "- {sort: raten, query: a, expect: nope.md, beleg: x}\n- {id: 7, sort: raten, query: a, expect: nope.md, beleg: x}\n")
	_, err := LoadQuestions(filepath.Join(dir, "questions.yaml"), Shape{})
	want := []string{
		"entry 1: missing field \"id\"",
		"7: unknown sort \"raten\"",
		"7: expect does not exist: " + filepath.Join(dir, "nope.md"),
	}
	if p := problemsOf(t, err); !slices.Equal(p, want) {
		t.Fatalf("problems:\n%s", strings.Join(p, "\n"))
	}
}

func TestLoadQuestionsNamesBrokenYAML(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "- [")
	_, err := LoadQuestions(filepath.Join(dir, "questions.yaml"), DefaultShape())
	p := problemsOf(t, err)
	if len(p) != 1 || !strings.HasPrefix(p[0], "questions.yaml is not valid YAML: ") {
		t.Fatalf("problems = %q", p)
	}
}

func TestLoadQuestionsWantsAList(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "a: 1")
	_, err := LoadQuestions(filepath.Join(dir, "questions.yaml"), DefaultShape())
	if p := problemsOf(t, err); !slices.Equal(p, Problems{"the question set must be a list of entries, found map"}) {
		t.Fatalf("problems = %q", p)
	}
}

func TestLoadQuestionsWantsMappings(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "- 3\n- [a]\n- x\n")
	_, err := LoadQuestions(filepath.Join(dir, "questions.yaml"), Shape{})
	want := Problems{
		"entry 1: must be a mapping, found int",
		"entry 2: must be a mapping, found list",
		"entry 3: must be a mapping, found string",
	}
	if p := problemsOf(t, err); !slices.Equal(p, want) {
		t.Fatalf("problems = %q", p)
	}
}

func TestLoadQuestionsReadsMappingsWithOtherKeys(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "Text.")
	writeYAML(t, dir, "- {1: one, id: q1, sort: exakt, query: a, expect: a.md, beleg: Text.}\n")
	qs, err := LoadQuestions(filepath.Join(dir, "questions.yaml"), Shape{Exact: 1})
	if err != nil || len(qs) != 1 || qs[0].ID != "q1" {
		t.Fatalf("questions = %+v, err = %v", qs, err)
	}
	writeYAML(t, dir, "1: one\n")
	_, err = LoadQuestions(filepath.Join(dir, "questions.yaml"), Shape{})
	if p := problemsOf(t, err); !slices.Equal(p, Problems{"the question set must be a list of entries, found map"}) {
		t.Fatalf("problems = %q", p)
	}
}

func TestLoadQuestionsReportsAMissingFile(t *testing.T) {
	_, err := LoadQuestions(filepath.Join(t.TempDir(), "questions.yaml"), DefaultShape())
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v", err)
	}
}

func TestHinweisIsOptional(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "Text.")
	writeYAML(t, dir, "- {id: q1, sort: gemischt, query: a, expect: a.md, beleg: Text.}\n- {id: q2, sort: gemischt, query: b, expect: a.md, beleg: Text., hinweis: 12}\n")
	qs, err := LoadQuestions(filepath.Join(dir, "questions.yaml"), Shape{Mixed: 2})
	if err != nil {
		t.Fatal(err)
	}
	if qs[0].Note != "" || qs[1].Note != "12" || qs[0].Kind != Mixed {
		t.Fatalf("questions = %+v", qs)
	}
	if qs[0].Expect != filepath.Join(dir, "a.md") {
		t.Fatalf("expect = %q", qs[0].Expect)
	}
}

// An absolute expect names its file wherever the set lies, as pathlib's /
// does with an absolute right side.
func TestLoadQuestionsTakesAnAbsoluteExpectAsIs(t *testing.T) {
	notes := t.TempDir()
	target := filepath.Join(notes, "a.md")
	writeFile(t, target, "Text.")
	dir := t.TempDir()
	writeYAML(t, dir, "- {id: q1, sort: exakt, query: a, expect: '"+target+"', beleg: Text.}\n")
	qs, err := LoadQuestions(filepath.Join(dir, "questions.yaml"), Shape{Exact: 1})
	if err != nil {
		t.Fatal(err)
	}
	if qs[0].Expect != target {
		t.Fatalf("expect = %q, want %q", qs[0].Expect, target)
	}
}

func TestEvidenceMatchesAcrossANoBreakSpace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "eine Wichte von 2 kN/m³ hat")
	writeYAML(t, dir, "- {id: q1, sort: exakt, query: a, expect: a.md, beleg: \"2 kN/m³\"}\n")
	if _, err := LoadQuestions(filepath.Join(dir, "questions.yaml"), Shape{Exact: 1}); err != nil {
		t.Fatal(err)
	}
}

func TestShapeProblemsFollowTheOrderOfKinds(t *testing.T) {
	got := shapeProblems([]Question{{Kind: CrossLingual}}, Shape{CrossLingual: 2, Exact: 1, Mixed: 0})
	want := []string{"exakt: 0 questions, expected 1", "sprachuebergreifend: 1 questions, expected 2"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q", got)
	}
}
