package tui

import (
	"bufio"
	"io"
	"unicode/utf8"
)

type decoder struct{ r *bufio.Reader }

func newDecoder(r io.Reader) *decoder { return &decoder{r: bufio.NewReader(r)} }

// next reads one key. A lone ESC is esc; ESC [ A..D and ESC O A..D are the
// arrows, and any other sequence is "unknown". Windows consoles in VT input
// mode send the same sequences; term.MakeRaw switches that mode on.
func (d *decoder) next() (Key, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return Key{}, err
	}
	switch b {
	case '\r', '\n':
		return Key{Name: "enter"}, nil
	case 0x7f, 0x08:
		return Key{Name: "backspace"}, nil
	case '\t':
		return Key{Name: "tab"}, nil
	case 0x03:
		return Key{Name: "ctrl-c"}, nil
	case 0x1b:
		// A sequence arrives in one read; ESC with nothing behind it was
		// pressed on its own.
		if d.r.Buffered() == 0 {
			return Key{Name: "esc"}, nil
		}
		return d.sequence(), nil
	}
	if b < utf8.RuneSelf {
		return Key{Rune: rune(b)}, nil
	}
	_ = d.r.UnreadByte()
	r, _, err := d.r.ReadRune()
	return Key{Rune: r}, err
}

// sequence reads what follows an ESC that did not come alone. Delete, the
// page keys, modified arrows and the like decode as "unknown", which every
// widget ignores: read as esc they would cancel the form, and their tail
// would arrive as typed text.
func (d *decoder) sequence() Key {
	// next saw this byte buffered, so the read cannot fail.
	intro, _ := d.r.ReadByte()
	switch intro {
	case '[':
		return d.csi()
	case 'O':
		// SS3: some POSIX terminals send the arrows as ESC O A..D.
		final, err := d.r.ReadByte()
		if err != nil {
			return Key{Name: "unknown"}
		}
		return arrow(final, true)
	}
	return Key{Name: "unknown"}
}

// csi reads a control sequence to its final byte. Only a bare ESC [ A..D is
// an arrow; with parameters (ctrl, shift) it is a key the widgets do not use.
func (d *decoder) csi() Key {
	bare := true
	for {
		b, err := d.r.ReadByte()
		switch {
		case err != nil:
			return Key{Name: "unknown"}
		case b >= 0x20 && b <= 0x3f:
			// parameter and intermediate bytes
			bare = false
		case b >= 0x40 && b <= 0x7e:
			return arrow(b, bare)
		default:
			// Not part of any sequence: leave it to be read as the next key.
			_ = d.r.UnreadByte()
			return Key{Name: "unknown"}
		}
	}
}

func arrow(final byte, bare bool) Key {
	if final < 'A' || final > 'D' || !bare {
		return Key{Name: "unknown"}
	}
	return Key{Name: [...]string{"up", "down", "right", "left"}[final-'A']}
}
