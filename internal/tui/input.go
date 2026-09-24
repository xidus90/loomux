package tui

import (
	"fmt"
	"slices"
	"strings"
)

// Input edits one line; choices, when given, cycle with tab and are the only
// accepted answers; check runs on enter and shows its error in place.
func Input(t Terminal, prompt, initial string, choices []string, check func(string) error) (string, bool, error) {
	value, problem := []rune(initial), ""
	for {
		msg := ""
		if problem != "" {
			msg = eol + "  " + problem
		}
		hint := ""
		if len(choices) > 0 {
			hint = fmt.Sprintf("  (tab: %v)", choices)
		}
		_, _ = fmt.Fprintf(t, "%s%s: %s%s%s", clearScreen, prompt, string(value), hint, msg)
		k, err := t.ReadKey()
		if err != nil {
			return "", false, err
		}
		switch {
		case k.Name == "esc" || k.Name == "ctrl-c":
			return "", false, nil
		case k.Name == "enter":
			if len(choices) > 0 && !slices.Contains(choices, string(value)) {
				problem = fmt.Sprintf("choose one of %s with tab", strings.Join(choices, ", "))
				continue
			}
			if check != nil {
				if err := check(string(value)); err != nil {
					problem = err.Error()
					continue
				}
			}
			return string(value), true, nil
		case k.Name == "tab" && len(choices) > 0:
			// A value outside the choices has index -1 and starts at the first.
			i := slices.Index(choices, string(value))
			value = []rune(choices[(i+1)%len(choices)])
		case k.Name == "backspace" && len(value) > 0 && len(choices) == 0:
			// With choices the value is picked, never typed, so there is
			// nothing to delete.
			value = value[:len(value)-1]
		case k.Rune != 0 && len(choices) == 0:
			value = append(value, k.Rune)
		}
		problem = ""
	}
}
