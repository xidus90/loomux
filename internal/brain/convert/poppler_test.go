package convert

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/dev/faketool"
)

// recordedPoppler answers from what the real Poppler printed for the test
// PDFs (recorded by a human with loomux dev record-poppler); the fusion spec
// (in the working papers of the archive release `archive/parity-recordings`)
// wants every external program replayed from a real run. The recording holds
// no stderr, so a failing PDF replays as one that said nothing.
func recordedPoppler(t *testing.T) Tools {
	t.Helper()
	fixture, err := faketool.Load(filepath.Join("..", "..", "..", "testdata", "convert", "poppler", faketool.FixtureName))
	if err != nil || len(fixture.Answers) == 0 {
		t.Fatalf("no Poppler recording: %v", err)
	}
	return Tools{
		Look: func(name string) (string, error) { return name, nil },
		Run: func(spec child.Spec) child.Result {
			answer, ok := fixture.Match(spec.Argv)
			if !ok {
				t.Fatalf("Poppler was not recorded for %q", spec.Argv)
			}
			return child.Result{Code: answer.Exit, Stdout: answer.Stdout}
		},
	}
}

func TestPopplersOwnOutputConvertsAsTheSeamSays(t *testing.T) {
	p := newPDFToText(recordedPoppler(t))
	// One text page of the test PDFs is a single line of 390 characters.
	textPage := func(e extraction) bool {
		return utf8.RuneCountInString(e.text) == 390 && strings.HasPrefix(e.text, "Hallo aus dem Pruefbestand.")
	}
	checks := map[string]func(extraction, error) bool{
		"text.pdf": func(e extraction, err error) bool {
			return err == nil && e.pages == 1 && e.skipped == 0 && textPage(e)
		},
		"Bericht M\xc3\xa4rz.pdf": func(e extraction, err error) bool {
			return err == nil && e.pages == 1 && e.skipped == 0 && textPage(e)
		},
		// One line: the escaped line breaks between the two copies yield
		// nothing in pdftotext, where pypdf makes two paragraphs of them.
		"paragraphs.pdf": func(e extraction, err error) bool {
			return err == nil && e.pages == 1 && strings.Count(e.text, "Pruefbestand") == 2 && !strings.Contains(e.text, "\n") &&
				utf8.RuneCountInString(e.text) == 780 && strings.Contains(e.text, "pruefen kann.Hallo aus dem")
		},
		"blank.pdf": func(e extraction, err error) bool {
			return err == nil && e.text == "" && e.pages == 1 && e.skipped == 1
		},
		// Three text pages among seven scans keep their three, one paragraph each.
		"mixed.pdf": func(e extraction, err error) bool {
			return err == nil && e.pages == 10 && e.skipped == 7 && strings.Count(e.text, "\n\n") == 2 &&
				strings.Count(e.text, "Pruefbestand") == 3
		},
		"allscan.pdf": func(e extraction, err error) bool {
			return err == nil && e.text == "" && e.pages == 2 && e.skipped == 2
		},
		"corrupt.pdf": func(e extraction, err error) bool {
			return err != nil && err.Error() == "pdftotext exited 1 and said nothing"
		},
		"encrypted.pdf": func(e extraction, err error) bool {
			return err != nil && err.Error() == "pdftotext exited 1 and said nothing"
		},
		// Poppler ends with 99 on a PDF without pages; the reference saw none.
		"pageless.pdf": func(e extraction, err error) bool {
			return err != nil && err.Error() == "pdftotext exited 99 and said nothing"
		},
	}
	if len(checks) != 9 {
		t.Fatalf("%d checks", len(checks))
	}
	for name, check := range checks {
		if e, err := p.extract(filepath.Join("testdata", "convert", "pdf", name)); !check(e, err) {
			t.Errorf("%s: %+v %v", name, e, err)
		}
	}
}
