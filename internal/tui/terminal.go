package tui

import (
	"io"
	"regexp"
	"strings"
	"sync"
)

// Key is one key press: a printable rune, or a named key such as "up",
// "down", "left", "right", "enter", "esc", "backspace", "tab" or "ctrl-c".
// "unknown" is any other key sequence; the widgets ignore it.
type Key struct {
	Rune rune   // printable input; 0 otherwise
	Name string // the named key; empty for printable input
}

// Terminal is what the widgets draw on and read from: a raw console, or a
// Scripted one in tests.
type Terminal interface {
	ReadKey() (Key, error)
	Write(p []byte) (int, error)
	Size() (width, height int)
}

// Scripted is a terminal that plays keys and records what was drawn.
type Scripted struct {
	keys          []Key
	out           strings.Builder
	width, height int
}

// Script returns a terminal of the given size that answers ReadKey with keys
// and then with io.EOF.
func Script(width, height int, keys ...Key) *Scripted {
	return &Scripted{keys: keys, width: width, height: height}
}

// ReadKey plays the next scripted key.
func (s *Scripted) ReadKey() (Key, error) {
	if len(s.keys) == 0 {
		return Key{}, io.EOF
	}
	k := s.keys[0]
	s.keys = s.keys[1:]
	return k, nil
}

// Write records p.
func (s *Scripted) Write(p []byte) (int, error) { return s.out.Write(p) }

// Size is the size given to Script.
func (s *Scripted) Size() (int, int) { return s.width, s.height }

// ansi compiles on first use: every package is linked into the binary each
// hook runs, and a package-level compile would cost every one of them.
var ansi = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`) })

// Output is what was written with the escape sequences taken out and the raw
// mode line ends turned back into plain ones.
func (s *Scripted) Output() string {
	return strings.ReplaceAll(ansi().ReplaceAllString(s.out.String(), ""), "\r\n", "\n")
}

// Keys builds a key list from a short spelling: "down", "enter", or text.
func Keys(parts ...string) []Key {
	var out []Key
	for _, p := range parts {
		switch p {
		case "up", "down", "left", "right", "enter", "esc", "backspace", "tab", "ctrl-c":
			out = append(out, Key{Name: p})
		default:
			for _, r := range p {
				out = append(out, Key{Rune: r})
			}
		}
	}
	return out
}
