package tui

import (
	"fmt"
	"strings"
)

const (
	clearScreen = "\x1b[H\x1b[2J"
	reverse     = "\x1b[7m"
	reset       = "\x1b[0m"
	// eol ends every line written to the terminal: in raw mode a bare "\n"
	// moves down without returning to the first column.
	eol = "\r\n"
)

// Row is one line of a List.
type Row struct {
	Group string // module; rows of one group are contiguous
	Label string // "commit.language"
	Value string
	Note  string // origin
}

// List shows rows, lets the human move and filter with '/', and returns the
// index chosen with enter, or -1 on esc/q.
func List(t Terminal, title string, rows []Row) (int, error) {
	cursor, filter, filtering := 0, "", false
	for {
		visible := matching(rows, filter)
		if cursor >= len(visible) {
			cursor = max(len(visible)-1, 0)
		}
		draw(t, title, rows, visible, cursor, filter, filtering)
		k, err := t.ReadKey()
		if err != nil {
			return -1, err
		}
		if filtering {
			switch {
			case k.Name == "enter":
				filtering = false
			case k.Name == "esc":
				filtering, filter = false, ""
			case k.Name == "backspace" && filter != "":
				rs := []rune(filter)
				filter = string(rs[:len(rs)-1])
			case k.Rune != 0:
				filter += string(k.Rune)
			}
			continue
		}
		switch {
		case k.Name == "up" && cursor > 0:
			cursor--
		case k.Name == "down" && cursor < len(visible)-1:
			cursor++
		case k.Name == "enter" && len(visible) > 0:
			return visible[cursor], nil
		case k.Name == "esc" || k.Name == "ctrl-c" || k.Rune == 'q':
			return -1, nil
		case k.Rune == '/':
			filtering = true
		}
	}
}

// matching returns the indexes into rows whose label or value contains
// filter, ignoring case, so a choice maps back to the full list.
func matching(rows []Row, filter string) []int {
	var out []int
	for i, r := range rows {
		if strings.Contains(strings.ToLower(r.Label+" "+r.Value), strings.ToLower(filter)) {
			out = append(out, i)
		}
	}
	return out
}

// draw repaints the whole screen; the window scrolls just far enough to keep
// the cursor in view. A frame is at most height lines, none wider than the
// terminal, and the last one has no line end: one line more, or one that
// wraps, scrolls the title off the top.
func draw(t Terminal, title string, rows []Row, visible []int, cursor int, filter string, filtering bool) {
	width, height := t.Size()
	lines := []string{fit(title, width)}
	if filtering || filter != "" {
		lines = append(lines, fit("/"+filter, width))
	} else {
		lines = append(lines, fit("↑↓ move · enter change · / filter · q quit", width))
	}
	// A group header and the cursor row are the least worth showing, even
	// when the terminal is too small for them.
	room := max(height-len(lines), 2)
	top := windowTop(rows, visible, cursor, room)
	lines = append(lines, layout(rows, visible, top, cursor, room, width, func(i int) string {
		r := rows[i]
		return fmt.Sprintf("  %-28s %-24s %s", r.Label, r.Value, r.Note)
	})...)
	_, _ = t.Write([]byte(clearScreen + strings.Join(lines, eol)))
}

// layout lays the visible rows from top on into at most room lines, each
// group under its header and the cursor row inverted; line renders the row
// at an index into rows, so List and Pick differ only in their row text.
func layout(rows []Row, visible []int, top, cursor, room, width int, line func(i int) string) []string {
	var lines []string
	used, group := 0, ""
	for n, i := range visible[top:] {
		r := rows[i]
		need := 1
		if r.Group != group {
			need++
		}
		if used+need > room {
			break
		}
		if r.Group != group {
			lines = append(lines, fit("["+r.Group+"]", width))
			group = r.Group
		}
		l := fit(line(i), width)
		if top+n == cursor {
			l = reverse + l + reset
		}
		lines = append(lines, l)
		used += need
	}
	return lines
}

// windowTop is the first visible row to draw: the earliest one from which
// the rows down to the cursor, with their group headers, fit into room lines.
func windowTop(rows []Row, visible []int, cursor, room int) int {
	top, used := cursor, 2 // the cursor row under its header
	for top > 0 {
		// The row above brings a header of its own; the header of the row
		// below stays only when the group changes between them.
		need := 1
		if rows[visible[top-1]].Group != rows[visible[top]].Group {
			need++
		}
		if used+need > room {
			break
		}
		top, used = top-1, used+need
	}
	return top
}

// fit cuts s to width runes so a line never wraps.
func fit(s string, width int) string {
	if rs := []rune(s); width > 0 && len(rs) > width {
		return string(rs[:width])
	}
	return s
}
