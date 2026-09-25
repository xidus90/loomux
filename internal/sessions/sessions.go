// Package sessions counts the agent sessions standing on one working tree.
//
// One file per session under `.loomux/state/hooks/`, and a directory beside
// it holding one file per subagent. loomux writes them all: `session-start`
// puts the session file down, `stop` rewrites it on every block and every
// pass, and `subagent-start` and `subagent-stop` keep the subagents' files.
// `worktree unlink` puts an end marker beside the session file, and
// `session-start` takes it away when the session resumes.
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
// A session marked ended by `Retire` is not counted. Nothing removes a
// session's file (checked on 2026-09-07: no removal anywhere in
// src/ultraloom/hooks, and no hook calls `Forget`), so a file older than
// `stale` is not counted either: that is a session that ended without a
// SessionEnd. Without that, one abandoned session would hold a junction for
// ever, and the fix for the case this whole count exists for -- a second
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
	ended := map[string]bool{}
	for _, entry := range entries {
		if name, ok := strings.CutSuffix(entry.Name(), endedSuffix); ok && !entry.IsDir() {
			ended[name+".json"] = true
		}
	}
	mine := safeName(sessionID) + ".json"
	cutoff := time.Now().Add(-stale)
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || entry.Name() == mine || ended[entry.Name()] {
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

// Forget removes this session's own file, its end marker and its subagents'
// files. No hook calls it: a session that ends is retired, so that a resume
// under the same id still finds its state. What is not there is not an
// error: a session that never wrote state still goes.
func Forget(root, sessionID string) error {
	for _, path := range []string{statePath(root, sessionID), endedPath(root, sessionID)} {
		if err := removeFile(path, "removing"); err != nil {
			return err
		}
	}
	dir := filepath.Join(root, filepath.FromSlash(StateDir), safeName(sessionID))
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing %s: %w", dir, err)
	}
	return nil
}

// removeFile removes one file; one that is not there is gone.
func removeFile(path, verb string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("%s %s: %w", verb, path, err)
	}
	return nil
}

// endedSuffix names the marker Retire puts beside a session's file.
const endedSuffix = ".ended"

func endedPath(root, sessionID string) string {
	return sessionFile(root, sessionID, endedSuffix)
}

// Retire marks this session as ended for `Others` and leaves what it holds.
//
// A marker beside the session's file, not a change to it: the stop gate's
// base and green tree, and the subagents' undelivered findings, are what a
// resume under the same id reads (see recordBase), and a stop still running
// when the session ends rewrites the file after this -- a marker in the file,
// or its age, would be undone by that write. A session that never wrote its
// file has nothing to count and gets no marker. Revive takes it away.
func Retire(root, sessionID string) error {
	if _, err := os.Stat(statePath(root, sessionID)); os.IsNotExist(err) {
		return nil
	}
	path := endedPath(root, sessionID)
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		return fmt.Errorf("retiring %s: %w", path, err)
	}
	return nil
}

// Revive counts a retired session again when it resumes under the same id: the
// marker goes, and the file is written back with its row of blocks reset, which
// also makes it young for `Others` however long the session rested. A resume
// is a new start for the stop gate's give-up counter, not for its base. A
// session that was never retired -- one killed without a SessionEnd, say --
// only has its file made young, so that it counts again too; a file that will
// not take the new time stays as old as it was, which is no worse than before.
func Revive(root, sessionID string) error {
	path := endedPath(root, sessionID)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		now := time.Now()
		_ = os.Chtimes(statePath(root, sessionID), now, now)
		return nil
	}
	if err != nil {
		return fmt.Errorf("reviving %s: %w", path, err)
	}
	state := ReadState(root, sessionID)
	state.Blocks = 0
	return WriteState(root, sessionID, state)
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
