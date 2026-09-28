package convert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func file(t *testing.T, name string, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDetectReadsTheEndingAndThenTheHead(t *testing.T) {
	for _, c := range []struct {
		name, data string
		want       Format
	}{
		{"video.txt", "[00:00] Hallo zusammen.\n\n[00:36] Und weiter.\n", TranscriptBracket},
		{"export.txt", "00:00:00 - 00:00:57\nHallo zusammen.\n", TranscriptRange},
		{"notiz.txt", "Nur Text, keine Zeitmarken.\n", Unsupported},
		{"buch.pdf", "%PDF-1.7\n", PDF},
		{"BUCH.PDF", "irrelevant", PDF},
		{"video.md", "[00:00] Hallo zusammen.\n", Unsupported},
		{"gross.txt", strings.Repeat("Prosa.\n", 20000) + "[00:00] spät.\n", Unsupported},
		{"kaputt.txt", "\xff\xfe\x00\x00rubbish", Unsupported},
		// Universal newlines, as Python's text mode reads: CRLF and a lone
		// CR both end a line.
		{"crlf.txt", "00:00:00 - 00:00:57\r\nHallo.\r\n", TranscriptRange},
		{"cr.txt", "Titel\r[00:00] Hallo.\r", TranscriptBracket},
		// CRLF is one character: 3000 lines of "a\r\n" are 6000 of the 8192,
		// so the mark after them is still in the head; counted as two, it
		// would stand at character 9000.
		{"crlf-far.txt", strings.Repeat("a\r\n", 3000) + "[00:00] Hallo.\r\n", TranscriptBracket},
		// Path.suffix of a dotfile is empty, and so is that of a name whose
		// only dots lead it.
		{".txt", "[00:00] Hallo.\n", Unsupported},
		{"..txt", "[00:00] Hallo.\n", Unsupported},
		{"..pdf", "%PDF-1.7\n", Unsupported},
		// Python's \s closes the range line: whatever str.isspace() holds may
		// stand after it.
		{"nbsp.txt", "00:00:00 - 00:00:57\U000000a0\nHallo.\n", TranscriptRange},
		{"vt.txt", "00:00:00 - 00:00:57\v\nHallo.\n", TranscriptRange},
		{"fs.txt", "00:00:00 - 00:00:57\x1c\nHallo.\n", TranscriptRange},
		{"nel.txt", "00:00:00 - 00:00:57\u0085\nHallo.\n", TranscriptRange},
		{"ideo.txt", "00:00:00 - 00:00:57\U00003000\nHallo.\n", TranscriptRange},
	} {
		if got := Detect(file(t, c.name, c.data)); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDetectCallsWhatItCannotReadUnsupported(t *testing.T) {
	if got := Detect(filepath.Join(t.TempDir(), "gone.txt")); got != Unsupported {
		t.Fatal(got)
	}
	// A directory opens but does not read.
	dir := filepath.Join(t.TempDir(), "ordner.txt")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := Detect(dir); got != Unsupported {
		t.Fatal(got)
	}
}

// Broken bytes past the sample pass detection; the full read fails later.
func TestDetectLooksAtTheHeadOnly(t *testing.T) {
	data := "[00:00] Anfang ist sauber.\n" + strings.Repeat("x", 45000) + "\n" + strings.Repeat("\xff\xfe", 50)
	if got := Detect(file(t, "spaet.txt", data)); got != TranscriptBracket {
		t.Fatal(got)
	}
}

// The reference's TextIOWrapper decodes whole chunks and calls this file
// unsupported: "[00:00] " and 4092 "ä" make its first chunk of 8192 bytes, 4092
// "x" complete the 8192 characters, and the broken byte ten characters later
// lies in the second chunk it read. loomux reads no further than the head; a
// row of the parity list.
func TestDetectReadsNoFurtherThanTheHead(t *testing.T) {
	data := "[00:00] " + strings.Repeat("ä", 4092) + strings.Repeat("x", 4092+10) + "\xff" + strings.Repeat("y", 100)
	if got := Detect(file(t, "chunk.txt", data)); got != TranscriptBracket {
		t.Fatal(got)
	}
}

// Python's \d takes Arabic-Indic digits, and the reference calls this file a
// bracket transcript; loomux's marks are ASCII digits only. A row of the
// parity list.
func TestDetectTakesASCIIDigitsOnly(t *testing.T) {
	if got := Detect(file(t, "ziffern.txt", "[\U00000660\U00000660:\U00000660\U00000665] Hallo.\n")); got != Unsupported {
		t.Fatal(got)
	}
}

// Path.suffix of Python 3.14 (pathlib/__init__.py:461) strips the leading
// dots, then cuts from the last dot on.
func TestPySuffixIsPathSuffix(t *testing.T) {
	for name, want := range map[string]string{
		"a.txt": ".txt", ".txt": "", "a.": ".", "a": "", "a.b.PDF": ".PDF", "..x": "",
		"a..": ".", "x.txt.": ".", ".": "", "..": "", "a. ": ". ",
	} {
		if got := pySuffix(name); got != want {
			t.Errorf("pySuffix(%q) = %q, want %q", name, got, want)
		}
	}
}
