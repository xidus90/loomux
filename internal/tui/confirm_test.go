package tui

import (
	"io"
	"strings"
	"testing"
)

func TestConfirm(t *testing.T) {
	for keys, want := range map[string]bool{"y": true, "n": false} {
		term := Script(80, 10, Keys(keys)...)
		got, err := Confirm(term, "- a\n+ b\n", "write?")
		if err != nil || got != want || !strings.Contains(term.Output(), "+ b") {
			t.Fatalf("%s: %v %v", keys, got, err)
		}
	}
	term := Script(80, 10, Keys("x", "esc")...)
	if got, _ := Confirm(term, "", "write?"); got {
		t.Fatal("esc is no")
	}
	if _, err := Confirm(Script(80, 10), "", "write?"); err != io.EOF {
		t.Fatal(err)
	}
}
