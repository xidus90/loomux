package tui

import (
	"fmt"
	"strings"
)

// Confirm shows text (a diff) and asks y/n.
func Confirm(t Terminal, text, question string) (bool, error) {
	// text comes with plain line ends; raw mode needs them to return to the
	// first column as well.
	_, _ = fmt.Fprintf(t, "%s%s"+eol+"%s [y/n] ", clearScreen, strings.ReplaceAll(text, "\n", eol), question)
	for {
		k, err := t.ReadKey()
		if err != nil {
			return false, err
		}
		switch {
		case k.Rune == 'y' || k.Rune == 'Y':
			return true, nil
		case k.Rune == 'n' || k.Rune == 'N' || k.Name == "esc" || k.Name == "ctrl-c":
			return false, nil
		}
	}
}
