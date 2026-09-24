package tui

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestInputEditsAndChecks(t *testing.T) {
	check := func(s string) error {
		if s == "bad" {
			return errors.New("no good")
		}
		return nil
	}
	term := Script(80, 10, Keys("backspace", "backspace", "backspace", "bad", "enter", "backspace", "backspace", "backspace", "fine", "enter")...)
	got, ok, err := Input(term, "value", "old", nil, check)
	if err != nil || !ok || got != "fine" {
		t.Fatalf("%q %v %v", got, ok, err)
	}
	if !strings.Contains(term.Output(), "no good") {
		t.Fatal("the check's error must be shown")
	}
}

func TestInputCyclesChoices(t *testing.T) {
	term := Script(80, 10, Keys("tab", "enter")...)
	got, ok, _ := Input(term, "language", "en", []string{"en", "de"}, nil)
	if !ok || got != "de" {
		t.Fatalf("%q %v", got, ok)
	}
	term = Script(80, 10, Keys("tab", "tab", "enter")...)
	if got, _, _ := Input(term, "language", "en", []string{"en", "de"}, nil); got != "en" {
		t.Fatalf("tab wraps, got %q", got)
	}
}

func TestInputCancels(t *testing.T) {
	if _, ok, err := Input(Script(80, 10, Keys("esc")...), "v", "x", nil, nil); ok || err != nil {
		t.Fatal(ok, err)
	}
	if _, _, err := Input(Script(80, 10), "v", "x", nil, nil); err != io.EOF {
		t.Fatal(err)
	}
}

func TestInputAcceptsOnlyChoices(t *testing.T) {
	term := Script(80, 10, Keys("backspace", "enter", "tab", "enter")...)
	got, ok, err := Input(term, "language", "xx", []string{"en", "de"}, nil)
	if err != nil || !ok || got != "en" {
		t.Fatalf("%q %v %v", got, ok, err)
	}
	if n := strings.Count(term.Output(), "choose one of"); n != 1 {
		t.Fatalf("the problem must show once, shown %d times:\n%s", n, term.Output())
	}
	term = Script(80, 10, Keys("backspace", "enter")...)
	if got, ok, _ := Input(term, "language", "de", []string{"en", "de"}, nil); !ok || got != "de" {
		t.Fatalf("backspace must not edit a choice, got %q", got)
	}
}

// TestInputDrawsHintAndProblemOnlyWhenThereIsOne: a frame without choices
// and without a problem ends with the value itself.
func TestInputDrawsHintAndProblemOnlyWhenThereIsOne(t *testing.T) {
	term := Script(80, 10)
	_, _, _ = Input(term, "v", "x", nil, nil)
	if got := term.out.String(); got != clearScreen+"v: x" {
		t.Fatalf("%q", got)
	}
	term = Script(80, 10)
	_, _, _ = Input(term, "v", "en", []string{"en", "de"}, nil)
	if got := term.out.String(); got != clearScreen+"v: en  (tab: [en de])" {
		t.Fatalf("%q", got)
	}
}

func TestInputIgnoresTabAndBackspaceWithNothingToDo(t *testing.T) {
	term := Script(80, 10, Keys("tab", "backspace", "enter")...)
	got, ok, err := Input(term, "v", "", nil, nil)
	if err != nil || !ok || got != "" {
		t.Fatalf("%q %v %v", got, ok, err)
	}
}
