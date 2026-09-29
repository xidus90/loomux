// Package switchover holds the pieces of the switch-over from the old tools
// to loomux that are code rather than a script.
package switchover

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

var (
	errNotObject = errors.New("not a JSON object")
	errTrailing  = errors.New("data after the JSON document")
)

// PruneHooks removes the hook groups of a Claude settings.json that a
// switch-over replaces: a group goes when every one of its commands contains
// at least one of the needles. A group that mixes such commands with others
// stays and is reported in kept as "<event>: <first matching command>"; each
// removed group is reported in removed as "<event>: <commands joined by "; ">".
// An event left without a group goes with them.
//
// Nothing outside the value of "hooks" is touched. When nothing is removed,
// out is settings itself. When something is, the new "hooks" value is written
// in the indent and with the line endings the file already uses.
func PruneHooks(settings []byte, needles []string) (out []byte, removed []string, kept []string, err error) {
	needles = nonEmpty(needles)
	if len(needles) == 0 {
		return settings, nil, nil, nil
	}
	start, end, found := -1, -1, false
	err = readObject(settings, func(key string, value json.RawMessage, valueEnd int) {
		if key == "hooks" {
			start, end, found = valueEnd-len(value), valueEnd, true
		}
	})
	if err != nil {
		return nil, nil, nil, err
	}
	if !found {
		return settings, nil, nil, nil
	}

	var events []event
	err = readObject(settings[start:end], func(key string, value json.RawMessage, _ int) {
		events = append(events, event{name: key, value: value})
	})
	if err != nil {
		return nil, nil, nil, err
	}
	for i := range events {
		r, k := events[i].prune(needles)
		removed = append(removed, r...)
		kept = append(kept, k...)
	}
	if len(removed) == 0 {
		return settings, nil, kept, nil
	}

	var block bytes.Buffer
	block.WriteByte('{')
	first := true
	for _, e := range events {
		if e.gone {
			continue
		}
		if !first {
			block.WriteByte(',')
		}
		first = false
		block.Write(marshalKey(e.name))
		block.WriteByte(':')
		block.Write(e.value)
	}
	block.WriteByte('}')

	out = append(out, settings[:start]...)
	out = append(out, format(block.Bytes(), settings, start)...)
	out = append(out, settings[end:]...)
	return out, removed, kept, nil
}

// nonEmpty drops the empty needles: an empty string is a substring of every
// command and would match all of them.
func nonEmpty(needles []string) []string {
	var out []string
	for _, n := range needles {
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

// readObject walks the members of the JSON object that data holds, in file
// order, and hands each to visit with the exact bytes of its value and the
// offset in data just after it. data must hold nothing but that object.
func readObject(data []byte, visit func(key string, value json.RawMessage, end int)) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	open, err := dec.Token()
	if err != nil {
		return err
	}
	if open != json.Delim('{') {
		return errNotObject
	}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return err
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return err
		}
		visit(key.(string), value, int(dec.InputOffset()))
	}
	if _, err := dec.Token(); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return errTrailing
	}
	return nil
}

// event is one member of the hooks object.
type event struct {
	name  string
	value json.RawMessage // the list of groups, or whatever the file has there
	gone  bool            // every group was removed, so the event is dropped
}

// prune removes the replaced groups from the event's list, rewrites value to
// the groups that are left, and reports what it did. A value that is no list
// is left alone.
func (e *event) prune(needles []string) (removed, kept []string) {
	var groups []json.RawMessage
	if json.Unmarshal(e.value, &groups) != nil {
		return nil, nil
	}
	var left []json.RawMessage
	for _, g := range groups {
		commands := commandsOf(g)
		matching := matches(commands, needles)
		switch {
		case len(commands) > 0 && len(matching) == len(commands):
			removed = append(removed, e.name+": "+strings.Join(commands, "; "))
		case len(matching) > 0:
			kept = append(kept, e.name+": "+matching[0])
			left = append(left, g)
		default:
			left = append(left, g)
		}
	}
	if len(left) == len(groups) {
		return removed, kept
	}
	e.gone = len(left) == 0
	e.value = append(append([]byte{'['}, bytes.Join(rawStrings(left), []byte{','})...), ']')
	return removed, kept
}

// rawStrings converts raw messages for bytes.Join.
func rawStrings(groups []json.RawMessage) [][]byte {
	out := make([][]byte, len(groups))
	for i, g := range groups {
		out[i] = g
	}
	return out
}

// commandsOf returns the "command" of every hook of a group, an empty string
// for a hook that has none. A group that cannot be read this way (not an
// object, a hooks list that is no list, a command that is no string) yields
// nothing, so it is never removed.
func commandsOf(group json.RawMessage) []string {
	var g struct {
		Hooks []struct {
			Command string `json:"command"`
		} `json:"hooks"`
	}
	if json.Unmarshal(group, &g) != nil {
		return nil
	}
	commands := make([]string, len(g.Hooks))
	for i, h := range g.Hooks {
		commands[i] = h.Command
	}
	return commands
}

// matches returns the commands that contain at least one needle.
func matches(commands, needles []string) []string {
	var out []string
	for _, c := range commands {
		for _, n := range needles {
			if strings.Contains(c, n) {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

// marshalKey writes an object key as JSON without escaping <, > and &. The
// newline the encoder appends is whitespace, which format drops.
func marshalKey(key string) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	// A string always encodes; the error is unreachable.
	_ = enc.Encode(key)
	return buf.Bytes()
}

// format lays the new hooks value out like the file around it. The value
// starts at offset start of settings; the whitespace that leads its line is
// the indent unit, because "hooks" is a member of the top-level object. A
// line without such whitespace means a compact file, and the value stays
// compact. The value gets the line ending of the line the key stands on: CRLF
// when a CR precedes the LF before that line.
func format(compact, settings []byte, start int) []byte {
	lineStart := bytes.LastIndexByte(settings[:start], '\n') + 1
	line := settings[lineStart:start]
	lead := line[:len(line)-len(bytes.TrimLeft(line, " \t"))]

	var buf bytes.Buffer
	// The block is built from valid JSON, so neither call can fail.
	if len(lead) == 0 {
		_ = json.Compact(&buf, compact)
		return buf.Bytes()
	}
	_ = json.Indent(&buf, compact, string(lead), string(lead))
	if lineStart >= 2 && settings[lineStart-2] == '\r' {
		return bytes.ReplaceAll(buf.Bytes(), []byte("\n"), []byte("\r\n"))
	}
	return buf.Bytes()
}
