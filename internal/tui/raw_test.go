package tui

import (
	"os"
	"strings"
	"testing"
)

func TestIsTerminalRefusesAFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if IsTerminal(f) {
		t.Fatal("a plain file is no terminal")
	}
}

func TestRawTerminalReadsAndWrites(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := rawTerminal{in: newDecoder(strings.NewReader("a")), out: f}
	if k, err := r.ReadKey(); err != nil || k.Rune != 'a' {
		t.Fatalf("%+v %v", k, err)
	}
	if n, err := r.Write([]byte("hi")); n != 2 || err != nil {
		t.Fatal(n, err)
	}
	got, err := os.ReadFile(f.Name())
	if err != nil || string(got) != "hi" {
		t.Fatalf("%q %v", got, err)
	}
}
