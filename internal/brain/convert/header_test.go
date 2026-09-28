package convert

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/index"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestSourceURLFromTheName(t *testing.T) {
	for name, want := range map[string]string{
		"Transkript_Video1_Second-Brain-Bauanleitung (mHSOsy_usAg).txt": "https://www.youtube.com/watch?v=mHSOsy_usAg",
		"NoteGPT_Transcript_RAG, Hybrid-Suche oder Wiki.txt":            "",
		"Notiz (Entwurf).txt":     "",
		"Notiz (Draft-Notes).txt": "https://www.youtube.com/watch?v=Draft-Notes",
	} {
		if got := SourceURLFrom(name); got != want {
			t.Errorf("%q: %q", name, got)
		}
	}
}

func TestTheHeadHasFourLinesAndAFifthForADescription(t *testing.T) {
	four := Head{SourceURL: "https://www.youtube.com/watch?v=mHSOsy_usAg", Retrieved: day(2026, 8, 24), Converter: TranscriptConverter, ASR: true}.String()
	want := "---\nsource_url: https://www.youtube.com/watch?v=mHSOsy_usAg\nretrieved: 2026-08-24\nconverter: brain-transcript/1\nasr: true\n---\n\n"
	if four != want {
		t.Fatalf("%q", four)
	}
	// The reference's header() for the same values (stufe-4d-orakel/convert_detect.py).
	empty := Head{Retrieved: day(2026, 8, 24), Converter: PDFConverter}.String()
	if want := "---\nsource_url:\nretrieved: 2026-08-24\nconverter: brain-pdf/2\nasr: false\n---\n\n"; empty != want {
		t.Fatalf("%q", empty)
	}
	five := Head{Retrieved: day(2026, 9, 7), Converter: "pdf", Description: "Der Bericht beschreibt die Abnahme."}.String()
	if want := "---\nsource_url:\nretrieved: 2026-09-07\nconverter: pdf\nasr: false\ndescription: Der Bericht beschreibt die Abnahme.\n---\n\n"; five != want {
		t.Fatalf("%q", five)
	}
}

func TestConvertedByKnowsOurOwnFileOnly(t *testing.T) {
	ours := "---\nsource_url:\nretrieved: 2026-08-24\nconverter: brain-pdf/1\nasr: false\n---\n\nText.\n"
	if got, ok := ConvertedBy(ours); !ok || got != "brain-pdf/1" {
		t.Fatalf("%q %v", got, ok)
	}
	for _, text := range []string{
		"---\ntitle: Meine Notiz\n---\n\nText.\n",
		"Einfach Text.\n",
		"---\nconverter: brain-pdf/1\nasr: false\n",
	} {
		if _, ok := ConvertedBy(text); ok {
			t.Errorf("%q counted as ours", text)
		}
	}
}

func TestDescriptionOfReadsTheHeadOnly(t *testing.T) {
	head := Head{Retrieved: day(2026, 9, 7), Converter: "pdf", Description: "Der Bericht beschreibt die Abnahme."}.String()
	if got, ok := DescriptionOf(head); !ok || got != "Der Bericht beschreibt die Abnahme." {
		t.Fatalf("%q %v", got, ok)
	}
	for _, text := range []string{
		Head{Retrieved: day(2026, 9, 7), Converter: "pdf"}.String(),
		"Einfach Text.\n",
		"---\nsource_url:\nretrieved: 2026-09-07\nconverter: brain-pdf/1\nasr: false\n---\n\ndescription: Das ist ein Satz aus dem Text.\n",
	} {
		if _, ok := DescriptionOf(text); ok {
			t.Errorf("%q yielded a description", text)
		}
	}
}

// The head lines are read with Python's \s and \S: a no-break or ideographic
// space around the value is not part of it, and \s crosses a line break as
// it does in the reference (stufe-4d-orakel/convert_detect.py).
func TestTheHeadLinesReadPythonsSpace(t *testing.T) {
	for text, want := range map[string]string{
		"---\ndescription:\U000000a0Satz.\n---\n":  "Satz.",
		"---\ndescription: Satz.\U000000a0\n---\n": "Satz.",
		"---\ndescription: Satz.\U00003000\n---\n": "Satz.",
		"---\ndescription:\nsource_url: x\n---\n":  "source_url: x",
	} {
		if got, ok := DescriptionOf(text); !ok || got != want {
			t.Errorf("DescriptionOf(%q) = %q %v", text, got, ok)
		}
	}
	for text, want := range map[string]string{
		"---\nconverter:\U00003000brain-pdf/1\n---\n":  "brain-pdf/1",
		"---\nconverter: brain-pdf/1\U000000a0\n---\n": "brain-pdf/1",
		"---\nconverter:\nfoo\n---\n":                  "foo",
	} {
		if got, ok := ConvertedBy(text); !ok || got != want {
			t.Errorf("ConvertedBy(%q) = %q %v", text, got, ok)
		}
	}
}

// acceptedSentences are the sentences of the judge battery the reference's
// PyYAML reads back: the ones describe can write into a head.
func acceptedSentences(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("../model/testdata/judge-battery.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Text      string `json:"text"`
		ReadsBack bool   `json:"reads_back"`
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	var sentences []string
	for _, row := range rows {
		if row.ReadsBack {
			sentences = append(sentences, row.Text)
		}
	}
	if len(sentences) == 0 {
		t.Fatal("the battery holds no sentence that reads back")
	}
	return sentences
}

// What describe accepts comes out of the head unchanged, for both readers
// loomux has: yaml.v3 through index.ParseFrontmatter, and DescriptionOf.
func TestAnAcceptedSentenceComesBackOutOfTheHead(t *testing.T) {
	sentences := append([]string{
		"Der Bericht beschreibt die Abnahme der zweiten Scheibe.",
		`Der Bericht nennt die Regel "Aus schlägt An" und ihre Grenzen.`,
	}, acceptedSentences(t)...)
	for _, sentence := range sentences {
		text := Head{Retrieved: day(2026, 9, 7), Converter: PDFConverter, Description: sentence}.String()
		meta, err := index.ParseFrontmatter(text, true)
		if err != nil || meta["description"] != sentence {
			t.Errorf("yaml.v3 reads %q as %v (%v)", sentence, meta["description"], err)
		}
		if got, ok := DescriptionOf(text); !ok || got != sentence {
			t.Errorf("DescriptionOf reads %q as %q", sentence, got)
		}
	}
}
