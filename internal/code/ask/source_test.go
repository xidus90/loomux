package ask_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/ask"
)

func fileWithLines(t *testing.T, n int) (root, rel string) {
	t.Helper()
	root = t.TempDir()
	rel = "p/big.go"
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString("line ")
		b.WriteString(strings.Repeat("x", 3))
		b.WriteString("\n")
	}
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, rel
}

func TestInlineCutsTheSpanOutOfTheFile(t *testing.T) {
	root, rel := fileWithLines(t, 20)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "L3-L5"}}}

	ask.Inline(root, &a, false)
	got := a.Hits[0].Code
	if strings.Count(got, "\n") != 2 {
		t.Fatalf("got %q, want exactly the three lines of the span", got)
	}
}

func TestInlineCapsAtEightyLinesAndSaysWhereTheRestIs(t *testing.T) {
	root, rel := fileWithLines(t, 300)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "L1-L250"}}}

	ask.Inline(root, &a, false)
	got := a.Hits[0].Code
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	// 80 lines plus the marker. The marker matters more than the cap: a reader
	// who needs the rest has to be told how to get it.
	if len(lines) != 81 {
		t.Fatalf("got %d lines, want 80 plus a marker", len(lines))
	}
	marker := lines[len(lines)-1]
	if !strings.Contains(marker, "170") || !strings.Contains(marker, rel+":L1-L250") {
		t.Errorf("marker %q must name how many lines are left and where they are", marker)
	}
}

func TestInlineFullLiftsTheCap(t *testing.T) {
	root, rel := fileWithLines(t, 300)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "L1-L250"}}}

	ask.Inline(root, &a, true)
	lines := strings.Split(strings.TrimRight(a.Hits[0].Code, "\n"), "\n")
	if len(lines) != 250 {
		t.Fatalf("got %d lines, want all 250", len(lines))
	}
}

func TestInlineClampsASpanPastTheEndOfTheFile(t *testing.T) {
	root, rel := fileWithLines(t, 5)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "L3-L900"}}}

	ask.Inline(root, &a, false)
	// A stale graph against an edited file: the span may point past the end.
	// Clamping beats an empty answer, and beats a panic by more.
	if a.Hits[0].Code == "" {
		t.Fatal("a span reaching past the file must still yield what is there")
	}
}

func TestInlineClampsZeroFromToOne(t *testing.T) {
	root, rel := fileWithLines(t, 5)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "L0-L3"}}}

	ask.Inline(root, &a, false)
	if a.Hits[0].Code == "" {
		t.Fatal("a span starting at L0 must clamp from to 1")
	}
}

func TestInlineReturnsEmptyWhenFromExceedsClampedTo(t *testing.T) {
	root, rel := fileWithLines(t, 5)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "L10-L20"}}}

	ask.Inline(root, &a, false)
	if a.Hits[0].Code != "" {
		t.Errorf("got %q, want empty code when from is past the entire file", a.Hits[0].Code)
	}
}

func TestInlineLeavesAHitAloneWhenTheFileIsGone(t *testing.T) {
	root := t.TempDir()
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: "gone.go", Span: "L1-L2"}}}

	ask.Inline(root, &a, false)
	// A file deleted since the build is drift, not a failure of the answer: the
	// hit keeps its location and loses only its excerpt.
	if a.Hits[0].Code != "" {
		t.Errorf("got %q, want no code and no error", a.Hits[0].Code)
	}
}

func TestInlineIgnoresAnUnreadableSpan(t *testing.T) {
	root, rel := fileWithLines(t, 5)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "nonsense"}}}

	ask.Inline(root, &a, false)
	if a.Hits[0].Code != "" {
		t.Errorf("got %q, want nothing for a span that does not parse", a.Hits[0].Code)
	}
}

func TestInlineSingleLineSpan(t *testing.T) {
	root, rel := fileWithLines(t, 10)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "L3-L3"}}}

	ask.Inline(root, &a, false)
	got := a.Hits[0].Code
	if got != "line xxx" {
		t.Fatalf("got %q, want exactly single line of span", got)
	}
}

func TestInlineExactlyEightyLinesHasNoMarker(t *testing.T) {
	root, rel := fileWithLines(t, 100)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "L1-L80"}}}

	ask.Inline(root, &a, false)
	got := a.Hits[0].Code
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 80 {
		t.Fatalf("got %d lines, want exactly 80 without a marker", len(lines))
	}
	if strings.Contains(got, "more lines") {
		t.Errorf("got marker in %q, want no marker at exactly maxSpanLines", got)
	}
}
