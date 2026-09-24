package tui

import (
	"os"

	"golang.org/x/term"
)

// IsTerminal says whether f is a console the interactive forms can use.
func IsTerminal(f *os.File) bool { return term.IsTerminal(int(f.Fd())) }

type rawTerminal struct {
	in  *decoder
	out *os.File
}

func (r rawTerminal) ReadKey() (Key, error)       { return r.in.next() }
func (r rawTerminal) Write(p []byte) (int, error) { return r.out.Write(p) }

// Size falls back to the classic 80x24 when the console will not say.
//
//coverage:exempt asks the real console for its size; a test process has none
func (r rawTerminal) Size() (int, int) {
	w, h, err := term.GetSize(int(r.out.Fd()))
	if err != nil {
		return 80, 24
	}
	return w, h
}

// Open puts the console into raw mode and returns a restore function that
// also clears the screen.
//
//coverage:exempt raw mode needs a real console; the widgets are tested through Scripted
func Open(in, out *os.File) (Terminal, func() error, error) {
	state, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return nil, nil, err
	}
	restoreVT := enableVT(out)
	restore := func() error {
		_, _ = out.WriteString(clearScreen)
		restoreVT()
		return term.Restore(int(in.Fd()), state)
	}
	return rawTerminal{in: newDecoder(in), out: out}, restore, nil
}
