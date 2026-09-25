package tui

import (
	"slices"
	"strings"
)

// Pick shows rows with a check box each and returns which are chosen: space
// toggles the row under the cursor, a toggles every row (all off when all are
// on, else all on), enter confirms, esc, q or ctrl-c cancel and hand back the
// input unchanged. The input is never written to.
func Pick(t Terminal, title string, rows []Row, chosen []bool) ([]bool, bool, error) {
	state := slices.Clone(chosen)
	cursor := 0
	for {
		drawPick(t, title, rows, state, cursor)
		k, err := t.ReadKey()
		if err != nil {
			return slices.Clone(chosen), false, err
		}
		switch {
		case k.Name == "up" && cursor > 0:
			cursor--
		case k.Name == "down" && cursor < len(rows)-1:
			cursor++
		case k.Rune == ' ' && len(rows) > 0:
			state[cursor] = !state[cursor]
		case k.Rune == 'a':
			all := !slices.Contains(state, false)
			for i := range state {
				state[i] = !all
			}
		case k.Name == "enter":
			return state, true, nil
		case k.Name == "esc" || k.Name == "ctrl-c" || k.Rune == 'q':
			return slices.Clone(chosen), false, nil
		}
	}
}

// drawPick is draw with a box in front of every label; it shares List's
// window arithmetic, so a frame never scrolls its title away.
func drawPick(t Terminal, title string, rows []Row, state []bool, cursor int) {
	width, height := t.Size()
	lines := []string{fit(title, width), fit("↑↓ move · space toggle · a all · enter accept · q cancel", width)}
	visible := make([]int, len(rows))
	for i := range rows {
		visible[i] = i
	}
	room := max(height-len(lines), 2)
	top := windowTop(rows, visible, cursor, room)
	lines = append(lines, layout(rows, visible, top, cursor, room, width, func(i int) string {
		box := "[ ] "
		if state[i] {
			box = "[x] "
		}
		return "  " + box + rows[i].Label + "  " + rows[i].Note
	})...)
	_, _ = t.Write([]byte(clearScreen + strings.Join(lines, eol)))
}
