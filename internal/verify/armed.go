package verify

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// ArmedFile is where a project keeps its armed lanes, relative to its root.
// It is versioned: every contributor and the CI read the same lanes.
const ArmedFile = ".loomux/armed.toml"

// armedHead opens the file; a human who finds it learns who writes it.
const armedHead = "# Lanes that are armed: a red run of one of them fails the gate.\n" +
	"# Written by the pre-commit gate; a human edits it through `loomux gate`.\n"

// ArmedSet is the lanes of a project whose red fails a run. The zero value
// is a project without the file, where every lane is armed; with the file a
// lane it does not name is in probation: it runs and reports, and fails
// nothing.
type ArmedSet struct {
	Exists bool
	Keys   []string // sorted, each once
}

// LaneKey names a lane in the file: kind, stack and area, the area always
// written out. The name a report prints will not do: it carries the area
// only where a stack has several, so a second area would rename the lane.
func LaneKey(j Job) string {
	// Not filepath.ToSlash: that leaves a backslash alone on POSIX, and one
	// project would then have two keys for one lane.
	return j.Kind + "/" + j.Stack + "@" + strings.ReplaceAll(j.Area, `\`, "/")
}

// ReadArmed reads root's file. A missing file is no error and arms every
// lane. A file that does not read arms every lane as well and says why: a
// truncated or conflicted file must not switch a gate off.
func ReadArmed(root string) (ArmedSet, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ArmedFile)))
	if errors.Is(err, fs.ErrNotExist) {
		return ArmedSet{}, nil
	}
	if err != nil {
		return ArmedSet{}, unreadable("cannot be read: " + err.Error())
	}
	doc := map[string]any{}
	if _, err := toml.Decode(string(data), &doc); err != nil {
		return ArmedSet{}, unreadable("is no TOML: " + err.Error())
	}
	for _, key := range slices.Sorted(maps.Keys(doc)) {
		if key != "armed" {
			return ArmedSet{}, unreadable("holds the unknown key " + key)
		}
	}
	list, ok := doc["armed"].([]any)
	if !ok {
		return ArmedSet{}, unreadable("needs `armed` as a list of lane keys")
	}
	keys := make([]string, 0, len(list))
	for _, entry := range list {
		key, ok := entry.(string)
		if !ok {
			return ArmedSet{}, unreadable("needs `armed` as a list of lane keys")
		}
		keys = append(keys, key)
	}
	return ArmedSet{}.With(keys...), nil
}

// unreadable is the one refusal ReadArmed has, with what follows from it.
func unreadable(why string) error {
	return fmt.Errorf("%s %s; every lane is armed", ArmedFile, why)
}

// Arms says whether a red run of j fails the gate.
func (a ArmedSet) Arms(j Job) bool {
	return !a.Exists || slices.Contains(a.Keys, LaneKey(j))
}

// With is the set with keys added; it exists afterwards.
func (a ArmedSet) With(keys ...string) ArmedSet {
	all := slices.Concat(a.Keys, keys)
	slices.Sort(all)
	return ArmedSet{Exists: true, Keys: slices.Compact(all)}
}

// Without is the set with keys taken out; it exists afterwards.
func (a ArmedSet) Without(keys ...string) ArmedSet {
	kept := slices.DeleteFunc(slices.Clone(a.Keys), func(k string) bool { return slices.Contains(keys, k) })
	return ArmedSet{Exists: true, Keys: kept}
}

// Missing are the keys the set does not hold, sorted, each once.
func (a ArmedSet) Missing(keys []string) []string {
	var out []string
	for _, k := range keys {
		if !slices.Contains(a.Keys, k) && !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

// Text is the file: one key a line, so that two branches that armed
// different lanes merge by keeping both sides.
func (a ArmedSet) Text() string {
	if len(a.Keys) == 0 {
		return armedHead + "armed = []\n"
	}
	var b strings.Builder
	b.WriteString(armedHead + "armed = [\n")
	for _, k := range a.Keys {
		b.WriteString("  " + config.QuoteTOML(k) + ",\n")
	}
	b.WriteString("]\n")
	return b.String()
}

// WriteArmed swaps root's file for a's text, with LF on every platform.
func WriteArmed(root string, a ArmedSet) error {
	path := filepath.Join(root, filepath.FromSlash(ArmedFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return lock.ReplaceText(path, a.Text())
}

// HookArms says whether a pre-commit hook's text runs the gate with --arm.
// A hook that mentions the call is not one that makes it: comments do not
// count, nor does a quoted string, which is one word. precommit and --arm
// must be words of the same command, read as the shell reads them, so --arm;
// and "--arm" count and --armed, another flag, does not.
//
// Known limits, as shellCommands reads a line: escapes, here-documents, line
// continuations and quotes over several lines are not read, and a redirect
// such as 2>&1 before --arm ends the command, so that hook reads as one that
// does not arm.
func HookArms(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		for _, words := range shellCommands(line) {
			if slices.Contains(words, "precommit") && slices.Contains(words, "--arm") {
				return true
			}
		}
	}
	return false
}

// shellCommands splits a shell line into its commands, each a list of words,
// enough to find a call in a hook: a quoted string belongs to its word
// without its quotes, a blank ends a word, and ; & | ( ) end a word and a
// command. A # that begins a word outside quotes ends the line. Escapes,
// expansions and here-documents are not read.
func shellCommands(line string) [][]string {
	var commands [][]string
	var words []string
	var word strings.Builder
	inWord := false
	var quote rune
	endWord := func() {
		if inWord {
			words = append(words, word.String())
			word.Reset()
			inWord = false
		}
	}
	endCommand := func() {
		endWord()
		if len(words) > 0 {
			commands = append(commands, words)
			words = nil
		}
	}
	for _, r := range line {
		switch {
		case quote != 0 && r == quote:
			quote = 0
		case quote != 0:
			word.WriteRune(r)
		case r == '"' || r == '\'':
			quote, inWord = r, true
		case r == '#' && !inWord:
			endCommand()
			return commands
		case unicode.IsSpace(r):
			endWord()
		case strings.ContainsRune(";&|()", r):
			endCommand()
		default:
			word.WriteRune(r)
			inWord = true
		}
	}
	endCommand()
	return commands
}
