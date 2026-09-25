package tui

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
)

func parts() []Row {
	return []Row{
		{Group: "hooks", Label: "host entries", Note: "claude"},
		{Group: "hooks", Label: "git hooks"},
		{Group: "hooks", Label: "verify-until-green skill"},
	}
}

func TestPickTogglesAndConfirms(t *testing.T) {
	in := []bool{true, true, true}
	term := Script(80, 20, Keys("down", " ", "enter")...)
	got, ok, err := Pick(term, "hooks", parts(), in)
	if err != nil || !ok || !slices.Equal(got, []bool{true, false, true}) {
		t.Fatalf("got %v %v %v", got, ok, err)
	}
	if !slices.Equal(in, []bool{true, true, true}) {
		t.Fatal("Pick changed its input")
	}
	out := term.Output()
	for _, want := range []string{"[x] host entries", "[ ] git hooks", "space toggle"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q", want)
		}
	}
}

func TestPickAllTogglesEveryRow(t *testing.T) {
	got, ok, err := Pick(Script(80, 20, Keys("a", "enter")...), "t", parts(), []bool{true, false, true})
	if !ok || err != nil || !slices.Equal(got, []bool{true, true, true}) {
		t.Fatalf("a with one off turns all on: %v", got)
	}
	got, ok, err = Pick(Script(80, 20, Keys("a", "enter")...), "t", parts(), []bool{true, true, true})
	if !ok || err != nil || !slices.Equal(got, []bool{false, false, false}) {
		t.Fatalf("a with all on turns all off: %v", got)
	}
}

func TestPickCancels(t *testing.T) {
	for _, k := range [][]Key{Keys("esc"), Keys("q"), Keys("ctrl-c")} {
		got, ok, err := Pick(Script(80, 20, append(Keys(" "), k...)...), "t", parts(), []bool{true, true, true})
		if ok || err != nil || !slices.Equal(got, []bool{true, true, true}) {
			t.Fatalf("%v: %v %v %v", k, got, ok, err)
		}
	}
}

func TestPickStopsAtTheEnds(t *testing.T) {
	got, ok, err := Pick(Script(80, 20, Keys("up", " ", "down", "down", "down", " ", "enter")...), "t", parts(), []bool{true, true, true})
	if !ok || err != nil || !slices.Equal(got, []bool{false, true, false}) {
		t.Fatal(got, ok, err)
	}
}

func TestPickMovesUp(t *testing.T) {
	got, ok, err := Pick(Script(80, 20, Keys("down", "down", "up", " ", "enter")...), "t", parts(), []bool{true, true, true})
	if !ok || err != nil || !slices.Equal(got, []bool{true, false, true}) {
		t.Fatal(got, ok, err)
	}
}

func TestPickReportsTheEndOfInput(t *testing.T) {
	if _, _, err := Pick(Script(80, 20), "t", parts(), make([]bool, 3)); err != io.EOF {
		t.Fatal(err)
	}
}

func TestPickIgnoresSpaceWithoutRows(t *testing.T) {
	got, ok, err := Pick(Script(80, 20, Keys(" ", "enter")...), "t", nil, nil)
	if err != nil || !ok || len(got) != 0 {
		t.Fatalf("%v %v %v", got, ok, err)
	}
}

func TestPickScrollsInASmallWindow(t *testing.T) {
	many := make([]Row, 30)
	for i := range many {
		many[i] = Row{Group: "g", Label: fmt.Sprintf("k%02d", i)}
	}
	keys := make([]Key, 0, 27)
	for range 25 {
		keys = append(keys, Key{Name: "down"})
	}
	term := Script(40, 8, append(keys, Key{Rune: ' '}, Key{Name: "enter"})...)
	got, ok, err := Pick(term, "t", many, make([]bool, 30))
	if err != nil || !ok || !got[25] {
		t.Fatalf("%v %v", ok, err)
	}
	// Output() strips the clear-screen sequence, so frames are split on the
	// raw bytes, as checkFrame does for List.
	for _, frame := range strings.Split(term.out.String(), clearScreen)[1:] {
		if n := len(strings.Split(frame, eol)); n > 8 || strings.HasSuffix(frame, eol) {
			t.Fatalf("a frame of %d lines in a window of 8:\n%s", n, frame)
		}
	}
	checkFrame(t, term, "t", "k25")
}
