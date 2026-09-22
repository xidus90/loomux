package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/lock"
)

// registryLockName is the lock file next to the registry. A name of its own
// rather than the registry itself: the lock holds a byte range, and a locked
// range would be unreadable for readers exactly as long as it means anything.
const registryLockName = "registry.lock"

// basicStringEscapes are the characters a TOML basic string spells with a
// short escape (`_BASIC_STRING_ESCAPES`, src/brain/maintenance/case.py:90).
var basicStringEscapes = map[rune]string{
	'\\': `\\`,
	'"':  `\"`,
	'\b': `\b`,
	'\t': `\t`,
	'\n': `\n`,
	'\f': `\f`,
	'\r': `\r`,
}

// QuoteTOML renders value as a TOML basic string, escaping every character
// the format forbids raw.
//
// Hand-written rather than strconv.Quote, because the registry is compared
// with the Python side byte for byte and the two disagree where it counts:
// Go spells U+0007 `\a`, U+000B `\v` and U+007F `\x7f`, where `_quote`
// writes `\u0007`, `\u000b` and `\u007f` -- and a file that differs there is
// a different file to every byte comparison with the reference. For `\a` and
// `\v` there is a second reason: TOML defines neither, and this binary's own
// reader (third_party/toml) refuses both, so such a file would be written and
// then refused. `\x7f` it reads, since its lexer knows `\x`; for DEL the
// reason is parity alone. This follows `_quote` instead -- a short escape
// where one exists, `\uXXXX` for every other C0 control character and for
// DEL, and the character itself otherwise.
//
// Exported because the registry is not the only hand-rendered TOML file in
// this binary: `maintenance.WriteCase` renders a case file in a fixed field
// order for the same reason and needs the same escaping. A second copy of
// this loop would be a second place to get U+007F wrong.
func QuoteTOML(value string) string {
	var out strings.Builder
	out.WriteByte('"')
	for _, char := range value {
		switch {
		case basicStringEscapes[char] != "":
			out.WriteString(basicStringEscapes[char])
		case char < 0x20 || char == 0x7F:
			fmt.Fprintf(&out, `\u%04x`, char)
		default:
			out.WriteRune(char)
		}
	}
	out.WriteByte('"')
	return out.String()
}

// RenderRegistry writes the areas as [[area]] tables.
//
// Sorted by scope, so a parity case can compare the file byte for byte; a
// flag that is false is left out, so the file of an ordinary area stays the
// one it was before the flag existed. For `wiki` leaving it out is not a
// choice but the only correct rendering: the reader answers "" for an absent
// key and refuses a present empty one (optionalString).
func RenderRegistry(areas []Area) string {
	sorted := append([]Area(nil), areas...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Scope < sorted[j].Scope })
	var out strings.Builder
	for i, area := range sorted {
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString("[[area]]\n")
		fmt.Fprintf(&out, "scope = %s\n", QuoteTOML(area.Scope))
		fmt.Fprintf(&out, "path = %s\n", QuoteTOML(area.Path))
		if area.WikiPath != "" {
			fmt.Fprintf(&out, "wiki = %s\n", QuoteTOML(area.WikiPath))
		}
		for _, flag := range []struct {
			name string
			set  bool
		}{
			{"readonly", area.ReadOnly},
			{"signpost", area.Signpost},
			{"shared", area.Shared},
			{"workspace", area.Workspace},
		} {
			if flag.set {
				fmt.Fprintf(&out, "%s = true\n", flag.name)
			}
		}
	}
	return out.String()
}

// WriteRegistry replaces the registry under the lock, and only if the reader
// accepts the result.
//
// The counter-check before the swap is the point: ReadRegistry refuses a
// registry it cannot use whole -- a duplicate scope, two areas whose state
// directories collide, a second signpost -- and a write side that did not
// know those rules could leave a file nobody reads again.
func WriteRegistry(stateDir string, areas []Area) error {
	return underRegistryLock(stateDir, func() error {
		return writeRegistryLocked(stateDir, areas)
	})
}

// AddArea takes one area into the registry.
func AddArea(stateDir string, area Area) error {
	return underRegistryLock(stateDir, func() error {
		areas, err := readRegistryIfPresent(stateDir)
		if err != nil {
			return err
		}
		for _, standing := range areas {
			if standing.Scope == area.Scope {
				return fmt.Errorf("%s: scope %q is already registered", stateDir, area.Scope)
			}
		}
		return writeRegistryLocked(stateDir, append(areas, area))
	})
}

// underRegistryLock runs body with the registry lock held. Read and write
// live inside one body so AddArea never hands the lock back between them.
func underRegistryLock(stateDir string, body func() error) error {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}
	handle, err := lock.Acquire(filepath.Join(stateDir, registryLockName))
	if err != nil {
		return err
	}
	defer handle.Release()
	return body()
}

// writeRegistryLocked renders, checks and swaps. Nothing is written when the
// check refuses: the standing registry survives untouched, down to its bytes.
func writeRegistryLocked(stateDir string, areas []Area) error {
	path := filepath.Join(stateDir, registryName)
	text := RenderRegistry(areas)
	if _, err := parseRegistry(stateDir, path, []byte(text)); err != nil {
		return fmt.Errorf("refusing to write a registry the reader rejects: %w", err)
	}
	return lock.ReplaceText(path, text)
}

// readRegistryIfPresent answers an empty list for the machine where no area
// was ever registered, and hands every other error on -- a registry that is
// there but unreadable must not be mistaken for one that is absent.
func readRegistryIfPresent(stateDir string) ([]Area, error) {
	areas, err := ReadRegistry(stateDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return areas, err
}
