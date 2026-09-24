package tui

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func rows() []Row {
	return []Row{
		{Group: "base", Label: "commit.language", Value: `"de"`, Note: "set"},
		{Group: "base", Label: "commit.threshold", Value: "2", Note: "default"},
		{Group: "brain", Label: "layout.wiki", Value: `"docs/wiki"`, Note: "set"},
	}
}

func TestListMovesAndChooses(t *testing.T) {
	term := Script(80, 20, Keys("down", "down", "enter")...)
	got, err := List(term, "loomux config", rows())
	if err != nil || got != 2 {
		t.Fatalf("got %d, %v", got, err)
	}
	out := term.Output()
	for _, want := range []string{"base", "brain", "commit.language", `"docs/wiki"`, "default"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q", want)
		}
	}
}

func TestListStopsAtTheEnds(t *testing.T) {
	term := Script(80, 20, Keys("up", "enter")...)
	if got, _ := List(term, "t", rows()); got != 0 {
		t.Fatal(got)
	}
	term = Script(80, 20, Keys("down", "down", "down", "enter")...)
	if got, _ := List(term, "t", rows()); got != 2 {
		t.Fatal(got)
	}
}

func TestListFilters(t *testing.T) {
	term := Script(80, 20, Keys("/", "wiki", "enter", "enter")...)
	if got, _ := List(term, "t", rows()); got != 2 {
		t.Fatalf("filtered choice must map back to the full list, got %d", got)
	}
}

func TestListFilterBackspaceAndEsc(t *testing.T) {
	term := Script(80, 20, Keys("/", "wx", "backspace", "esc", "enter")...)
	if got, _ := List(term, "t", rows()); got != 0 {
		t.Fatalf("esc in the filter clears it, got %d", got)
	}
}

func TestListQuits(t *testing.T) {
	for _, k := range [][]Key{Keys("esc"), Keys("q"), Keys("ctrl-c")} {
		if got, err := List(Script(80, 20, k...), "t", rows()); got != -1 || err != nil {
			t.Fatalf("%v: %d %v", k, got, err)
		}
	}
}

func TestListReportsTheEndOfInput(t *testing.T) {
	if _, err := List(Script(80, 20), "t", rows()); err != io.EOF {
		t.Fatal(err)
	}
}

func TestListScrollsInASmallWindow(t *testing.T) {
	many := make([]Row, 30)
	for i := range many {
		many[i] = Row{Group: "g", Label: fmt.Sprintf("k%02d", i)}
	}
	keys := make([]Key, 0, 31)
	for range 25 {
		keys = append(keys, Key{Name: "down"})
	}
	term := Script(40, 8, append(keys, Key{Name: "enter"})...)
	if got, _ := List(term, "t", many); got != 25 {
		t.Fatal(got)
	}
	checkFrame(t, term, "t", "k25")
}

func TestListFitsSeveralGroupsInASmallWindow(t *testing.T) {
	many := make([]Row, 30)
	for i := range many {
		// A new group every second row costs a header line per pair.
		many[i] = Row{Group: fmt.Sprintf("g%02d", i/2), Label: fmt.Sprintf("k%02d", i), Value: strings.Repeat("v", 60)}
	}
	for _, down := range []int{0, 3, 17, 29} {
		keys := make([]Key, 0, down+1)
		for range down {
			keys = append(keys, Key{Name: "down"})
		}
		term := Script(40, 8, append(keys, Key{Name: "enter"})...)
		if got, _ := List(term, "title", many); got != down {
			t.Fatalf("down %d: got %d", down, got)
		}
		checkFrame(t, term, "title", fmt.Sprintf("k%02d", down))
	}
}

// checkFrame asserts that the last screen drawn fits the terminal, keeps the
// title on its first line and shows the cursor row.
func checkFrame(t *testing.T, term *Scripted, title, cursor string) {
	t.Helper()
	frames := strings.Split(term.out.String(), clearScreen)
	frame := frames[len(frames)-1]
	width, height := term.Size()
	lines := strings.Split(frame, eol)
	if len(lines) > height || strings.HasSuffix(frame, eol) {
		t.Fatalf("%d lines, trailing eol %v, for height %d:\n%s", len(lines), strings.HasSuffix(frame, eol), height, frame)
	}
	if lines[0] != title {
		t.Fatalf("title scrolled away:\n%s", frame)
	}
	found := false
	for _, l := range lines {
		plain := ansi().ReplaceAllString(l, "")
		if n := len([]rune(plain)); n > width {
			t.Fatalf("line of %d runes for width %d: %q", n, width, plain)
		}
		if strings.Contains(l, reverse) && strings.Contains(plain, cursor) {
			found = true
		}
	}
	if !found {
		t.Fatalf("cursor row %s not shown:\n%s", cursor, frame)
	}
}

func TestListMovesUp(t *testing.T) {
	term := Script(80, 20, Keys("down", "down", "up", "enter")...)
	if got, _ := List(term, "t", rows()); got != 1 {
		t.Fatal(got)
	}
}

func TestListIgnoresEnterWhenNothingMatches(t *testing.T) {
	term := Script(80, 20, Keys("/", "nothing", "enter", "enter", "q")...)
	if got, err := List(term, "t", rows()); got != -1 || err != nil {
		t.Fatalf("%d %v", got, err)
	}
}

// lastFrame is the last screen drawn, split into its lines, escapes kept.
func lastFrame(term *Scripted) []string {
	frames := strings.Split(term.out.String(), clearScreen)
	return strings.Split(frames[len(frames)-1], eol)
}

func TestListShowsTheFilterWhileTypedAndAfterwards(t *testing.T) {
	hint := "↑↓ move · enter change · / filter · q quit"
	for keys, want := range map[string][]string{
		hint:  nil,
		"/":   {"/"},
		"/w":  {"/", "w", "enter"},
		"/wi": {"/", "wx", "backspace", "i"},
	} {
		term := Script(80, 20, Keys(want...)...)
		_, _ = List(term, "t", rows())
		if got := lastFrame(term)[1]; got != keys {
			t.Errorf("%v: second line %q, want %q", want, got, keys)
		}
	}
}

func TestListKeepsTheCursorOnARowWhenTheFilterNarrows(t *testing.T) {
	// The cursor stood on row 2; only rows 0 and 1 match, so it moves up
	// to the last of them.
	term := Script(80, 20, Keys("down", "down", "/", "commit", "enter", "enter")...)
	if got, _ := List(term, "t", rows()); got != 1 {
		t.Fatal(got)
	}
}

func TestListBackspaceOnAnEmptyFilterDoesNothing(t *testing.T) {
	term := Script(80, 20, Keys("/", "backspace", "enter", "enter")...)
	if got, _ := List(term, "t", rows()); got != 0 {
		t.Fatal(got)
	}
}

// TestListShowsAsManyRowsAboveTheCursorAsFit: seven lines hold the title,
// the hint, two headers and all three rows exactly; five hold only the
// cursor row under its header.
func TestListShowsAsManyRowsAboveTheCursorAsFit(t *testing.T) {
	term := Script(80, 7, Keys("down", "down")...)
	_, _ = List(term, "t", rows())
	if frame := strings.Join(lastFrame(term), "\n"); !strings.Contains(frame, "commit.language") {
		t.Fatalf("the first row fits and must be shown:\n%s", frame)
	}
	term = Script(80, 5, Keys("down", "down")...)
	_, _ = List(term, "t", rows())
	checkFrame(t, term, "t", "layout.wiki")
}

func TestFitLeavesALineAloneWhenTheWidthIsUnknown(t *testing.T) {
	if got := fit("abc", 0); got != "abc" {
		t.Fatalf("%q", got)
	}
}
