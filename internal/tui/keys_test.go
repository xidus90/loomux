package tui

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDecodeReadsKeysAndEscapeSequences(t *testing.T) {
	in := bytes.NewReader([]byte("a\x1b[A\x1b[B\r\x7f\t\x03\x1b"))
	d := newDecoder(in)
	want := []Key{{Rune: 'a'}, {Name: "up"}, {Name: "down"}, {Name: "enter"}, {Name: "backspace"}, {Name: "tab"}, {Name: "ctrl-c"}, {Name: "esc"}}
	for i, w := range want {
		got, err := d.next()
		if err != nil || got != w {
			t.Fatalf("key %d: got %+v, %v; want %+v", i, got, err, w)
		}
	}
	if _, err := d.next(); err != io.EOF {
		t.Fatalf("want EOF, got %v", err)
	}
}

func TestDecodeReadsUTF8(t *testing.T) {
	d := newDecoder(strings.NewReader("ü"))
	if k, _ := d.next(); k.Rune != 'ü' {
		t.Fatalf("%+v", k)
	}
}

func TestDecodeReadsTheRestOfTheKeys(t *testing.T) {
	// Each case is decoded alone, so what follows ESC is exactly the bytes
	// given. A sequence the widgets do not know must not read as esc, which
	// cancels, and must be consumed whole.
	for in, want := range map[string]Key{
		"\n":        {Name: "enter"},
		"\x08":      {Name: "backspace"},
		"\x1b[C":    {Name: "right"},
		"\x1b[D":    {Name: "left"},
		"\x1bOA":    {Name: "up"},
		"\x1bOB":    {Name: "down"},
		"\x1bOC":    {Name: "right"},
		"\x1bOD":    {Name: "left"},
		"\x1bOP":    {Name: "unknown"},
		"\x1bO":     {Name: "unknown"},
		"\x1b[Z":    {Name: "unknown"},
		"\x1b[3~":   {Name: "unknown"},
		"\x1b[5~":   {Name: "unknown"},
		"\x1b[6~":   {Name: "unknown"},
		"\x1b[1;5A": {Name: "unknown"},
		"\x1b[":     {Name: "unknown"},
		"\x1b[1;":   {Name: "unknown"},
		"\x1bx":     {Name: "unknown"},
		// The edges of the byte ranges: space and ? are intermediate or
		// parameter bytes, @ is a final byte.
		"\x1b[ A": {Name: "unknown"},
		"\x1b[?A": {Name: "unknown"},
		"\x1b[@":  {Name: "unknown"},
		// A lone continuation byte is no rune of its own.
		"\x80": {Rune: utf8.RuneError},
	} {
		d := newDecoder(strings.NewReader(in))
		if got, err := d.next(); err != nil || got != want {
			t.Errorf("%q: got %+v, %v; want %+v", in, got, err, want)
		}
		if k, err := d.next(); err != io.EOF {
			t.Errorf("%q: left %+v behind", in, k)
		}
	}
}

func TestDecodeLeavesAByteThatEndsNoSequence(t *testing.T) {
	// A control byte cannot belong to a sequence; it is the next key.
	d := newDecoder(strings.NewReader("\x1b[1\r"))
	for _, want := range []Key{{Name: "unknown"}, {Name: "enter"}} {
		if got, err := d.next(); err != nil || got != want {
			t.Fatalf("got %+v, %v; want %+v", got, err, want)
		}
	}
}

func TestWidgetsIgnoreUnknownKeys(t *testing.T) {
	unknown := Key{Name: "unknown"}
	if got, _ := List(Script(80, 20, unknown, Key{Name: "enter"}), "t", rows()); got != 0 {
		t.Fatalf("List: %d", got)
	}
	if got, ok, _ := Input(Script(80, 10, unknown, Key{Name: "enter"}), "v", "x", nil, nil); !ok || got != "x" {
		t.Fatalf("Input: %q %v", got, ok)
	}
	if got, _ := Confirm(Script(80, 10, unknown, Key{Rune: 'y'}), "", "q?"); !got {
		t.Fatal("Confirm")
	}
}
