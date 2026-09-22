// Package sessions counts the agent sessions standing on one working tree.
//
// One file per session under `.loomux/state/hooks/`, and a directory beside
// it holding one file per subagent. loomux writes them all: `session-start`
// puts the session file down, `stop` rewrites it on every block and every
// pass, and `subagent-start` and `subagent-stop` keep the subagents' files.
// ultraloom is no longer a writer here -- it keeps its own state under
// `.ultraloom/hooks/`, a different directory.
package sessions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// StateDir is where a session's file lives, relative to the working tree.
// One constant for the whole package; two spellings of one directory would
// drift, and every hook that writes here goes through this one.
const StateDir = ".loomux/state/hooks"

// Others counts the sessions on `root` that are not `sessionID`.
//
// Nothing that writes these files deletes one (checked on 2026-09-07: no
// removal anywhere in src/ultraloom/hooks; `Forget` below is the first, and it
// reaches only the sessions that run through it), so a file older than `stale`
// is not counted. Without that, one abandoned session would hold a junction
// for ever, and the fix for the case this whole count exists for -- a second
// session in the same tree -- would have broken the ordinary case instead.
func Others(root, sessionID string, stale time.Duration) (int, error) {
	dir := filepath.Join(root, filepath.FromSlash(StateDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// No session ever wrote here. Nobody else is holding anything.
			return 0, nil
		}
		return 0, fmt.Errorf("reading %s: %w", dir, err)
	}
	mine := safeName(sessionID) + ".json"
	cutoff := time.Now().Add(-stale)
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || entry.Name() == mine {
			continue
		}
		// An entry whose age cannot be had is not counted, the same as a stale
		// one: gone between the listing and the question is one session fewer,
		// not a failure. The `||` short-circuits, so ModTime is only asked of
		// an info that is there.
		info, err := entry.Info()
		if err != nil || info.ModTime().Before(cutoff) {
			continue
		}
		count++
	}
	return count, nil
}

// Forget removes this session's own file and its subagents' files.
//
// Nobody did this before, which is why `Others` needs a staleness rule at all.
// What is not there is not an error: a session that never wrote state still
// ends.
func Forget(root, sessionID string) error {
	path := statePath(root, sessionID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %s: %w", path, err)
	}
	dir := filepath.Join(root, filepath.FromSlash(StateDir), safeName(sessionID))
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing %s: %w", dir, err)
	}
	return nil
}

// safeName is state.py's rule, spelled in Go: the id comes from outside, so it
// may not decide where the file lands. Anything but a letter, a number, a dash
// or an underscore is dropped, and an id that leaves nothing becomes "unnamed"
// rather than the directory itself.
//
// Letter and number in the Unicode sense, because state.py's own test is
// `char.isalnum()` and that answers true well outside ASCII -- measured on
// 2026-09-07 with CPython 3.13: `'ä'`, `'٣'`, `'五'`, `'²'` and `'Ⅰ'` are all
// alnum, which is the L* and N* categories. An ASCII-only rule here would send
// this package looking for `unnamed.json` where Python wrote the letter.
func safeName(sessionID string) string {
	var builder strings.Builder
	for _, char := range sessionID {
		if unicode.IsLetter(char) || unicode.IsNumber(char) || char == '-' || char == '_' {
			builder.WriteRune(char)
		}
	}
	if builder.Len() == 0 {
		return "unnamed"
	}
	return builder.String()
}
