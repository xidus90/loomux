package edit

import (
	"fmt"
	"slices"
	"strings"
)

// RemoveItem takes the index-th entry (from 0, in document order) out of the
// rule list at the dotted section, in whichever form the file wrote that
// list: a [[section]] block or an element of key = [ ... ]. Comments that sit
// directly above the entry go with it; the rest stays as the human wrote it.
func RemoveItem(text, section string, index int) (string, error) {
	switch RuleNode(text, section).(type) {
	case []map[string]any:
		return removeBlock(text, section, index)
	case []any:
		return removeInline(text, section, index)
	}
	// Absent, a scalar or text that is no TOML: there is no list to take from.
	return "", noItem(section, index)
}

func noItem(section string, index int) error {
	return fmt.Errorf("no item %d in %s", index, section)
}

func isBlank(s string) bool { return strings.TrimSpace(s) == "" }

func isComment(s string) bool { return strings.HasPrefix(strings.TrimSpace(s), "#") }

// removeBlock drops one [[section]] block: its header, its keys and the
// comment lines right above it. A comment after its last key is the next
// block's and stays.
func removeBlock(text, section string, index int) (string, error) {
	bom, lines, eol := split(text)
	entries, err := scan(lines, section, "")
	if err != nil {
		return "", err
	}
	var heads []int
	for k, e := range entries {
		if e.isHeader && e.isList && e.section == section {
			heads = append(heads, k)
		}
	}
	if index < 0 || index >= len(heads) {
		return "", noItem(section, index)
	}
	k := heads[index]
	start, end := entries[k].first, entries[k].last
	for start > 0 && isComment(lines[start-1]) {
		start--
	}
	for _, e := range entries[k+1:] {
		// A sub-table of the block belongs to it; any other header ends it.
		if e.isHeader && (e.section == section || !strings.HasPrefix(e.section, section+".")) {
			break
		}
		if t := strings.TrimSpace(lines[e.first]); t != "" && t[0] != '#' {
			end = e.last
		}
	}
	return join(bom, withoutLines(lines, start, end), eol), nil
}

// withoutLines deletes lines start..end and, so that no double gap or
// leading or trailing blank line is left, one adjacent blank line.
func withoutLines(lines []string, start, end int) []string {
	out := slices.Delete(slices.Clone(lines), start, end+1)
	switch {
	case start > 0 && isBlank(out[start-1]) && (start == len(out) || isBlank(out[start])):
		out = slices.Delete(out, start-1, start)
	case start == 0 && len(out) > 0 && isBlank(out[0]):
		out = out[1:]
	}
	return out
}

// span is one element of an array: its first byte, the byte after its last,
// and where its comma stands (-1 for none), all as offsets into the value.
type span struct{ start, end, comma int }

type array struct {
	open, close int
	items       []span
}

// arrayOf reads the elements of the array in s, the text of a key's value
// from its key on. Strings, nested arrays, inline tables and comments inside
// an element do not end it.
func arrayOf(s string) array {
	a := array{}
	depth, quote := 0, byte(0)
	cur := span{-1, -1, -1}
	for i := strings.IndexByte(s, '=') + 1; i < len(s); i++ {
		c := s[i]
		switch {
		case depth == 0:
			if c == '[' {
				a.open, depth = i, 1
			}
			continue
		case quote != 0:
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			continue
		case c == '#':
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		case c == ',' && depth == 1:
			cur.comma = i
			a.items = append(a.items, cur)
			cur = span{-1, -1, -1}
			continue
		case c == ']' && depth == 1:
			if cur.start >= 0 {
				a.items = append(a.items, cur)
			}
			a.close = i
			return a
		case c == '"' || c == '\'':
			quote = c
		case c == '[' || c == '{':
			depth++
		case c == ']' || c == '}':
			depth--
		}
		if cur.start < 0 {
			cur.start = i
		}
		cur.end = i + 1
	}
	a.close = -1
	return a
}

// removeInline drops one element of key = [ ... ] with its comma, and its own
// lines where it has them.
func removeInline(text, section string, index int) (string, error) {
	dot := strings.LastIndexByte(section, '.')
	table, key := section[:max(dot, 0)], section[dot+1:]
	bom, lines, eol := split(text)
	entries, err := scan(lines, table, key)
	if err != nil {
		return "", err
	}
	// RuleNode found an array here, and a key that scan could not place has
	// already failed, so exactly one line holds it.
	_, hits, _ := locate(entries, table, key)
	hit := hits[0]
	body := strings.Join(lines[hit.first:hit.last+1], "\n")
	arr := arrayOf(body)
	if index < 0 || index >= len(arr.items) {
		return "", noItem(section, index)
	}
	it := arr.items[index]
	switch {
	case len(arr.items) == 1:
		body = body[:arr.open+1] + body[arr.close:]
	case it.comma >= 0:
		to := it.comma + 1
		for to < len(body) && (body[to] == ' ' || body[to] == '\t') {
			to++
		}
		from := it.start
		// A last element with a trailing comma takes the blank before it too:
		// the array keeps ending on `,]` instead of `, ]`.
		if to < len(body) && body[to] == ']' {
			for body[from-1] == ' ' || body[from-1] == '\t' {
				from--
			}
		}
		body = cut(body, from, to)
	default:
		// The last element has no comma of its own, so the one before it
		// must lose its comma, or the array would end on a dangling one.
		from := it.start
		for body[from-1] == ' ' || body[from-1] == '\t' {
			from--
		}
		comma := arr.items[index-1].comma
		body = cut(cut(body, from, it.end), comma, comma+1)
	}
	lines = slices.Concat(lines[:hit.first], strings.Split(body, "\n"), lines[hit.last+1:])
	return join(bom, lines, eol), nil
}

// cut deletes body[from:to]; where that leaves nothing but a comment on the
// lines it touched, those lines and the comment lines right above them go.
func cut(body string, from, to int) string {
	ls := strings.LastIndexByte(body[:from], '\n') + 1
	nl := strings.IndexByte(body[to:], '\n')
	if nl >= 0 {
		le := to + nl
		if isBlank(body[ls:from]) && (isBlank(body[to:le]) || isComment(body[to:le])) {
			from, to = ls, le+1
			for from > 0 {
				prev := strings.LastIndexByte(body[:from-1], '\n') + 1
				if !isComment(body[prev : from-1]) {
					break
				}
				from = prev
			}
		}
	}
	return body[:from] + body[to:]
}
