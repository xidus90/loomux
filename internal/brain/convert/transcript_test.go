package convert

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func bracket(count, chars int) string {
	var parts []string
	for i := range count {
		parts = append(parts, fmt.Sprintf("[%02d:%02d] ", i/60, i%60)+strings.Repeat("wort ", chars/5))
	}
	return strings.Join(parts, "\n\n")
}

func TestTheThresholdDecidesWhereAParagraphEnds(t *testing.T) {
	if got := toParagraphs(bracket(3, 100), TranscriptBracket, 1200); strings.Contains(got, "\n\n") {
		t.Fatal("three short fragments made more than one paragraph")
	}
	got := toParagraphs(bracket(20, 100), TranscriptBracket, 1200)
	paragraphs := strings.Split(got, "\n\n")
	if len(paragraphs) != 2 {
		t.Fatalf("%d paragraphs", len(paragraphs))
	}
	for _, p := range paragraphs {
		if strings.Count(p, "[") != 1 || !strings.HasPrefix(p, "[") {
			t.Fatalf("a paragraph keeps more than the first mark: %.40q", p)
		}
	}
}

func TestNotOneWordIsLostOrAdded(t *testing.T) {
	text := "[00:00] Hallo zusammen, das hier ist mein\n\n[00:04] Second Brain mit 2000 Notizen.\n"
	words := func(s string) []string {
		return slices.DeleteFunc(strings.Fields(s), func(w string) bool { return strings.HasPrefix(w, "[") })
	}
	if got := ToParagraphs(text, TranscriptBracket); !slices.Equal(words(got), words(text)) {
		t.Fatalf("%q", got)
	}
}

func TestMarksBecomeMinutes(t *testing.T) {
	for _, c := range []struct {
		text string
		f    Format
		want string
	}{
		{"00:01:05 - 00:01:57\nHallo zusammen.\n\n00:02:10 - 00:02:44\nUnd weiter.\n", TranscriptRange, "[01:05] "},
		{"01:02:03 - 01:02:44\nSpät im Video.\n", TranscriptRange, "[62:03] "},
		{"[02:00:00] Zwei Stunden rein.\n", TranscriptBracket, "[120:00] "},
	} {
		if got := ToParagraphs(c.text, c.f); !strings.HasPrefix(got, c.want) {
			t.Errorf("%q: %q", c.text, got)
		}
	}
	if got := ToParagraphs("00:01:05 - 00:01:57\nHallo.\n", TranscriptRange); strings.Contains(got, "00:01:05") {
		t.Fatal("the range line survived")
	}
}

func TestTheLeadInIsKeptAsItsOwnParagraph(t *testing.T) {
	got := strings.Split(ToParagraphs("Titel des Videos\n\n[00:00] Hallo zusammen.\n", TranscriptBracket), "\n\n")
	if len(got) != 2 || got[0] != "Titel des Videos" || !strings.HasPrefix(got[1], "[00:00]") {
		t.Fatalf("%q", got)
	}
}

func TestEmptyPiecesFallAway(t *testing.T) {
	if got := ToParagraphs("", TranscriptBracket); got != "" {
		t.Fatalf("%q", got)
	}
	if got := ToParagraphs("[00:00]\n[00:01] Hallo.\n", TranscriptBracket); got != "[00:01] Hallo." {
		t.Fatalf("%q", got)
	}
}

// Python's str.split breaks on \x1c..\x1f, U+3000 and U+00A0; strings.Fields
// does not break on \x1c..\x1f.
func TestWhitespaceIsPythons(t *testing.T) {
	if got := ToParagraphs("[00:00] a\x1fb\U00003000c\U000000a0d\n", TranscriptBracket); got != "[00:00] a b c d" {
		t.Fatalf("%q", got)
	}
}

// The range line ends in Python's \s: a no-break space after it belongs to
// the mark, not to the speech.
func TestARangeLineEndsInPythonsSpace(t *testing.T) {
	for _, tail := range []string{"\U000000a0", "\v", "\x1c", "\u0085", "\U00003000"} {
		if got := ToParagraphs("00:00:00 - 00:00:57"+tail+"\nHallo.\n", TranscriptRange); got != "[00:00] Hallo." {
			t.Errorf("%q: %q", tail, got)
		}
	}
}

// A mark in Arabic-Indic digits is no mark to loomux: it stays as text in the
// fragment before it, and a paragraph cannot start at it. The reference cuts
// it out as a mark: '[00:00] Eins. Zwei. Drei.', and with threshold 1
// '[00:00] Eins.\n\n[00:05] Zwei.\n\n[00:09] Drei.'. A row of the parity list.
func TestAMarkInOtherDigitsStaysText(t *testing.T) {
	text := "[00:00] Eins.\n[\U00000660\U00000660:\U00000660\U00000665] Zwei.\n[00:09] Drei.\n"
	if got := ToParagraphs(text, TranscriptBracket); got != "[00:00] Eins. [\U00000660\U00000660:\U00000660\U00000665] Zwei. Drei." {
		t.Errorf("%q", got)
	}
	if got := toParagraphs(text, TranscriptBracket, 1); got != "[00:00] Eins. [\U00000660\U00000660:\U00000660\U00000665] Zwei.\n\n[00:09] Drei." {
		t.Errorf("threshold 1: %q", got)
	}
}

// The threshold counts characters, not bytes.
func TestTheThresholdCountsCharacters(t *testing.T) {
	text := "[00:00] " + strings.Repeat("ä", 700) + "\n[00:01] " + strings.Repeat("ö", 400) + "\n[00:02] x\n"
	if got := strings.Count(ToParagraphs(text, TranscriptBracket), "\n\n"); got != 0 {
		t.Fatalf("1103 characters were cut into %d+1 paragraphs", got)
	}
}
