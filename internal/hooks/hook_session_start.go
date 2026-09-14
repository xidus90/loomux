package hooks

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/sessions"
)

// The three seams the stale check is tested through. Variables rather than
// parameters because every one of them answers a question about the running
// process, not about the project: in a test os.Executable names the test
// binary in a temp directory outside any project, and the two failure arms of
// the walk need a directory the filesystem refuses.
var (
	executable = os.Executable
	absPath    = filepath.Abs
	walkDir    = filepath.WalkDir
)

// SessionStart writes down the commit the session starts on.
//
// Never blocks. This is an announcement, so the only codes it can leave with
// are 0 and 1 -- exit 2 in payload.py's protocol (payload.py:13-15) means
// blocked, and there is nothing here to hold a turn over.
//
// Waiting flow runs are not announced here: internal/journal does not move in
// stage 1a, so the report comes back with the flow migration.
func SessionStart(stdin io.Reader, stdout, stderr io.Writer, root, hostName string) int {
	host, err := hosts.ParseHost(hostName)
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}
	payload, err := hosts.Read(host, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}

	if err := recordBase(payload.SessionID, root); err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}

	lines := staleBinary(root)

	if err := hosts.WriteContext(host, stdout, lines); err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

// recordBase keeps the commit this session starts on, if there is one to keep.
//
// Here and nowhere else: by the time the first Stop fires, the turn has
// already run, and anything it committed would sit inside the baseline that is
// supposed to expose it.
//
// Silent in two of the three cases, which was _record_base's decision
// (session_start.py (fa3dd38):43-59, deleted in 6a7037a). Without a session id
// there is nowhere to file it,
// and outside a repository there is nothing to file -- neither is a defect of
// the project, and neither is worth a line in every session of every checkout
// that is not a git repository. The stop gate is where the absence matters,
// and that is where it is said out loud.
//
// A write that fails is the third case and it is *not* silent, because the
// Python original is not silent about it either: state.py's `write`
// (state.py:56-65) catches nothing, so an OSError there leaves `run` by
// itself. Swallowing it here would be a departure dressed up as parity.
func recordBase(sessionID, root string) error {
	if sessionID == "" {
		// hosts.Read hands a missing id and a wrongly typed one over as the
		// empty string alike; that contract is on hosts.Payload.SessionID, and
		// the type assertion keeping it is in the Claude adapter. It is the
		// `isinstance(session_id, str)` test of session_start.py (fa3dd38):52
		// with the one divergence that a literal `"session_id": ""` files
		// nothing here while Python filed it under `unnamed` (state.py:75).
		return nil
	}
	commit, err := gitwork.HeadCommit(root)
	if err != nil {
		return nil
	}
	state := sessions.ReadState(root, sessionID)
	state.Base = commit
	return sessions.WriteState(root, sessionID, state)
}

// staleBinary names the running binary when it lives inside the project and a
// source that decides its behaviour changed after it was built: a pilot hook
// then judges with yesterday's rules.
//
// The comparison is against the sources and deliberately not against HEAD.
// The pre-commit gate builds bin/loomux.exe before the commit exists, so the
// commit time would always lie after the binary, and the warning would stand
// after every commit.
func staleBinary(root string) []string {
	path, err := executable()
	if err != nil {
		return nil
	}
	absRoot, err := absPath(root)
	if err != nil {
		return nil
	}
	rel, err := filepath.Rel(absRoot, path)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	newest, name, err := newestSource(absRoot)
	if err != nil || !info.ModTime().Before(newest) {
		return nil
	}
	return []string{fmt.Sprintf(
		"loomux binary %s is older than %s; rebuild with: go build -o bin/loomux.new.exe ./cmd/loomux, then go run ./cmd/loomux dev swap-binary --dir bin",
		filepath.ToSlash(rel), name)}
}

// newestSource is the latest modification among go.mod, go.sum and the .go
// files under cmd/ and internal/, with that file's slash-separated path.
func newestSource(root string) (time.Time, string, error) {
	var newest time.Time
	var name string
	consider := func(path string, info fs.FileInfo) {
		if info.ModTime().After(newest) {
			newest = info.ModTime()
			rel, _ := filepath.Rel(root, path)
			name = filepath.ToSlash(rel)
		}
	}
	for _, file := range []string{"go.mod", "go.sum"} {
		if info, err := os.Stat(filepath.Join(root, file)); err == nil {
			consider(filepath.Join(root, file), info)
		}
	}
	for _, dir := range []string{"cmd", "internal"} {
		err := walkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if errors.Is(err, fs.ErrNotExist) && path == filepath.Join(root, dir) {
				return filepath.SkipDir
			}
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			consider(path, info)
			return nil
		})
		if err != nil {
			return time.Time{}, "", err
		}
	}
	return newest, name, nil
}
