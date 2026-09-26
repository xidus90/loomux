package benchsearch

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// edit rewrites one file of a stand, failing when old is not there.
func edit(t *testing.T, path, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(old)) {
		t.Fatalf("%s holds no %q", path, old)
	}
	writeFile(t, path, strings.Replace(string(data), old, replacement, 1))
}

func corpusProblems(t *testing.T, stand string) Problems {
	t.Helper()
	return problemsOf(t, CheckCorpus(stand))
}

func TestTheCheckedInCorpusPasses(t *testing.T) {
	if err := CheckCorpus(corpusDir(t)); err != nil {
		t.Fatal(err)
	}
}

func TestCheckCorpusFindsAChangedByte(t *testing.T) {
	stand := copyCorpus(t)
	note := filepath.Join(stand, "notes", "baustatik-01.md")
	data, _ := os.ReadFile(note)
	writeFile(t, note, string(bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))))
	var p Problems
	if !errors.As(CheckCorpus(stand), &p) || !strings.Contains(p[0], "baustatik-01.md: checksum ") {
		t.Fatalf("problems = %v", p)
	}
}

func TestCheckCorpusStopsOnlyForFilesItReads(t *testing.T) {
	stand := copyCorpus(t)
	os.Remove(filepath.Join(stand, "HERKUNFT.md"))
	var p Problems
	errors.As(CheckCorpus(stand), &p)
	if !slices.Contains(p, "HERKUNFT.md is missing") || len(p) != 1 {
		t.Fatalf("problems = %v", p)
	}
	os.Remove(filepath.Join(stand, "manifest.json"))
	errors.As(CheckCorpus(stand), &p)
	if !slices.Equal(p, Problems{"manifest.json is missing"}) {
		t.Fatalf("problems = %v", p)
	}
	os.RemoveAll(filepath.Join(stand, "notes"))
	os.Remove(filepath.Join(stand, "questions.yaml"))
	errors.As(CheckCorpus(stand), &p)
	if !slices.Equal(p, Problems{"notes is missing", "questions.yaml is missing", "manifest.json is missing"}) {
		t.Fatalf("problems = %v", p)
	}
}

func TestCheckCorpusCountsTheNotes(t *testing.T) {
	stand := copyCorpus(t)
	writeFile(t, filepath.Join(stand, "notes", "extra.md"), "x")
	p := corpusProblems(t, stand)
	want := []string{
		"101 notes, expected 100",
		"extra.md: not named in HERKUNFT.md",
		"extra.md: not in the manifest",
		"extra.md: claimed by no theme",
	}
	if !slices.Equal(p, want) {
		t.Fatalf("problems:\n%s", strings.Join(p, "\n"))
	}
}

func TestCheckCorpusHoldsHerkunftBothWays(t *testing.T) {
	stand := copyCorpus(t)
	herkunft := filepath.Join(stand, "HERKUNFT.md")
	edit(t, herkunft, "`baustatik-01.md`", "`geist.md`")
	p := corpusProblems(t, stand)
	want := []string{
		"baustatik-01.md: not named in HERKUNFT.md",
		"geist.md: named in HERKUNFT.md but not among the notes",
	}
	if !slices.Equal(p, want) {
		t.Fatalf("problems:\n%s", strings.Join(p, "\n"))
	}
}

// Names in HERKUNFT.md are words in any script, not only ASCII ones.
func TestCheckCorpusReadsUnicodeNamesInHerkunft(t *testing.T) {
	stand := copyCorpus(t)
	edit(t, filepath.Join(stand, "HERKUNFT.md"), "`baustatik-01.md`", "`baustatik-01.md` `grüße.md`")
	if p := corpusProblems(t, stand); !slices.Equal(p, Problems{"grüße.md: named in HERKUNFT.md but not among the notes"}) {
		t.Fatalf("problems = %q", p)
	}
}

func TestCheckCorpusHoldsTheManifest(t *testing.T) {
	stand := copyCorpus(t)
	manifest := filepath.Join(stand, "manifest.json")
	data, _ := os.ReadFile(manifest)
	body := string(data)
	body = strings.Replace(body, `"baustatik-01.md"`, `"weg.md"`, 1)
	body = strings.Replace(body, `"baustatik-03.md": {`, `"baustatik-03.md": 3, "unused": {`, 1)
	writeFile(t, manifest, body)
	p := corpusProblems(t, stand)
	want := []string{
		"baustatik-01.md: not in the manifest",
		"baustatik-03.md: entry must be a mapping, found float64",
		"unused: in the manifest but not among the notes",
		"weg.md: in the manifest but not among the notes",
	}
	if !slices.Equal(p, want) {
		t.Fatalf("problems:\n%s", strings.Join(p, "\n"))
	}
}

func TestCheckCorpusWantsProvenanceFields(t *testing.T) {
	stand := copyCorpus(t)
	writeFile(t, filepath.Join(stand, "manifest.json"), `{"baustatik-01.md": {"quelle": "", "abgerufen": 2026}}`)
	p := corpusProblems(t, stand)
	want := []string{
		"baustatik-01.md: missing \"sha256\"",
		"baustatik-01.md: missing \"lizenz\"",
		"baustatik-01.md: empty \"quelle\"",
	}
	if !slices.Equal(p[:3], want) {
		t.Fatalf("problems:\n%s", strings.Join(p, "\n"))
	}
}

func TestCheckCorpusNamesABrokenManifest(t *testing.T) {
	stand := copyCorpus(t)
	manifest := filepath.Join(stand, "manifest.json")
	writeFile(t, manifest, "{,")
	p := corpusProblems(t, stand)
	if !slices.ContainsFunc(p, func(s string) bool { return strings.HasPrefix(s, "manifest.json is not valid JSON: ") }) {
		t.Fatalf("problems:\n%s", strings.Join(p, "\n"))
	}
	writeFile(t, manifest, "[]")
	if p := corpusProblems(t, stand); !slices.Contains(p, "manifest.json must be a mapping, found list") {
		t.Fatalf("problems:\n%s", strings.Join(p, "\n"))
	}
}

func TestCheckCorpusHoldsThePartition(t *testing.T) {
	stand := copyCorpus(t)
	themes := filepath.Join(stand, "themes.yaml")
	// baustatik-01.md moves to baustoffe as well: two themes claim it.
	edit(t, themes, "    - baustoffe-01.md\n", "    - baustoffe-01.md\n    - baustatik-01.md\n")
	// baustatik loses a note: nine notes, and baustatik-10.md is claimed by none.
	edit(t, themes, "    - baustatik-10.md\n", "")
	edit(t, themes, "nachbar: baustatik\n", "nachbar: baustoffe\n")
	edit(t, themes, "nachbar: baustoffe\n  notizen:\n    - baustatik-01.md", "nachbar: fehlt\n  notizen:\n    - baustatik-01.md")
	edit(t, themes, "    - baustoffe-02.md\n", "    - geist.md\n    - baustoffe-02.md\n")
	p := corpusProblems(t, stand)
	want := []string{
		"baustatik: 9 notes, expected 10",
		"baustatik: neighbour \"fehlt\" is not a theme",
		"baustoffe: 12 notes, expected 10",
		"baustoffe: is its own neighbour",
		"baustoffe: geist.md is not among the notes",
		"baustatik-01.md: claimed by two themes",
		"baustatik-10.md: claimed by no theme",
	}
	if !slices.Equal(p, want) {
		t.Fatalf("problems:\n%s", strings.Join(p, "\n"))
	}
}

func TestCheckCorpusReadsBrokenThemes(t *testing.T) {
	cases := map[string][]string{
		"- [":   {"themes.yaml is not valid YAML: "},
		"a: 1":  {"themes.yaml must be a list of entries, found map"},
		"- 3\n": {"theme 1: must be a mapping, found int", "0 themes, expected 10"},
		"- {nachbar: x}\n- {name: n, nachbar: x, notizen: 3}\n": {
			"theme 1: missing field \"name\"",
			"theme 1: missing field \"notizen\"",
			"n: \"notizen\" must be a list, found int",
			"0 themes, expected 10",
		},
	}
	for body, want := range cases {
		stand := copyCorpus(t)
		writeFile(t, filepath.Join(stand, "themes.yaml"), body)
		p := corpusProblems(t, stand)
		for i, w := range want {
			if i >= len(p) || !strings.HasPrefix(p[i], w) {
				t.Errorf("%q: problems:\n%s", body, strings.Join(p, "\n"))
				break
			}
		}
	}
}

func TestCheckCorpusReportsQuestionProblems(t *testing.T) {
	stand := copyCorpus(t)
	edit(t, filepath.Join(stand, "questions.yaml"), "sort: exakt", "sort: raten")
	if p := corpusProblems(t, stand); !slices.Equal(p, Problems{"c01: unknown sort \"raten\"", "exakt: 12 questions, expected 13"}) {
		t.Fatalf("problems:\n%s", strings.Join(p, "\n"))
	}
}

func TestCheckCorpusNamesAnUnreadableQuestionSet(t *testing.T) {
	stand := copyCorpus(t)
	questions := filepath.Join(stand, "questions.yaml")
	os.Remove(questions)
	if err := os.Mkdir(questions, 0o755); err != nil {
		t.Fatal(err)
	}
	p := corpusProblems(t, stand)
	if len(p) != 1 || !strings.Contains(p[0], "questions.yaml") {
		t.Fatalf("problems = %q", p)
	}
}

func TestCheckCorpusWantsFiveReverseQuestions(t *testing.T) {
	stand := copyCorpus(t)
	questions := filepath.Join(stand, "questions.yaml")
	// The corpus holds six English questions; two in German leave four.
	for _, opening := range []string{`query: "How long`, `query: "Which German`} {
		edit(t, questions, opening, `query: "Was sagt der die das und von dem den zu mit der die das und von dem den`)
	}
	if p := corpusProblems(t, stand); !slices.Equal(p, Problems{"4 questions in the reverse direction, expected at least 5"}) {
		t.Fatalf("problems = %q", p)
	}
}

func TestIsReverseCountsFunctionWords(t *testing.T) {
	cases := map[string]bool{
		"What does the Eurocode say about snow?": true,
		"Was sagt der Eurocode zum Schnee?":      false,
		"in is":                                  false, // both languages: counted for neither
		"WHAT DOES THE":                          true,
		"Für die":                                false,
	}
	for query, want := range cases {
		if got := isReverse(Question{Kind: CrossLingual, Query: query}); got != want {
			t.Errorf("%q: %v", query, got)
		}
	}
	if isReverse(Question{Kind: Exact, Query: "What does the"}) {
		t.Error("only cross-lingual questions can be reverse")
	}
}
