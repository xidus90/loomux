package hooks

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/selfupdate"
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

// SessionStart writes down the commit the session starts on and announces the
// flow runs that wait for an answer. version is this binary's, spelled as flow
// run writes it into a run's marker.
//
// Never blocks. This is an announcement, so the only codes it can leave with
// are 0 and 1 -- exit 2 in payload.py's protocol (payload.py:13-15) means
// blocked, and there is nothing here to hold a turn over.
func SessionStart(stdin io.Reader, stdout, stderr io.Writer, root, hostName, version string) int {
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

	// A session worktree unlink retired and that now resumes under the same
	// id counts again, and before the base is looked at, so that a base that
	// cannot be written does not leave it uncounted. A session that stays
	// uncounted is told in its context, the one channel a host reads at exit
	// 0: exit 1 would drop the context lines with it.
	//
	// Only at the first start: nothing retires an agy conversation between
	// two of its model calls (Retire is reached only through worktree unlink,
	// which is wired for Claude alone), so a later PreInvocation has nothing
	// to revive and a marker it cannot remove is not news. Should unlink ever
	// be wired for agy, this is to be decided again.
	var lines []string
	if payload.SessionID != "" && !payload.Repeat {
		if err := sessions.Revive(root, payload.SessionID); err != nil {
			lines = append(lines, "loomux: this session may not count for worktree unlink, so another session ending here may remove its junctions: "+err.Error())
		}
	}
	if err := recordBase(payload.SessionID, root); err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}

	// The base is filed and the warnings were said at the first start, and
	// the lanes in probation are named once: agy's PreInvocation comes before
	// every model call, and planning the lanes each time would cost every one
	// of them.
	if !payload.Repeat {
		lines = append(lines, staleBinary(root)...)
		lines = append(lines, updateWarnings(config.StateDir(), runtime.GOOS)...)
		lines = append(lines, probationLines(root)...)
	}
	// What the project's flows say is said at every start, a repeated one
	// included: a waiting question stays open until a human answers it, and a
	// session that heard it once may since have dropped it from its context.
	lines = append(lines, waitingRuns(root, version, stderr)...)
	lines = append(lines, ignoredFlowFolders(root)...)

	if err := hosts.WriteContext(host, "SessionStart", stdout, lines); err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

// recordBase keeps the commit this session starts on, if there is one to keep
// and the session has none yet.
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
	state := sessions.ReadState(root, sessionID)
	if state.Base != "" {
		// SessionStart fires again on resume, clear and compact under the
		// same id. Moving the base to HEAD there would put everything
		// committed since the last green run inside the baseline, and the
		// next stop would find nothing to check. The base stays; a green
		// run advances it (spec, Nachtrag 22).
		return nil
	}
	commit, err := gitwork.HeadCommit(root)
	if err != nil {
		return nil
	}
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

// updateWarnings reads what serve's last self-update pass left in
// update.json. Only the file: session start asks neither the network nor the
// service, and hooks may not import serve. Without the file it says nothing,
// because a machine without a running service is not at fault. goos is a
// parameter so that a test reaches both platforms' rules on either.
//
// There is deliberately no warning by age. serve checks a minute after it
// starts, so after two days off the first session would read one while
// nothing is wrong.
func updateWarnings(stateDir, goos string) []string {
	st, err := selfupdate.ReadStatus(stateDir)
	if err != nil {
		return []string{fmt.Sprintf("loomux cannot read the update status: %v", err)}
	}
	if st == nil {
		return nil
	}
	var lines []string
	// A pass by hand may run from a checkout; only serve's record says where
	// the service runs from. And only on Windows: elsewhere the pass skips
	// before it looks at the path, and no location there is ever canonical.
	if canonical := selfupdate.Canonical(stateDir); goos == "windows" && st.Source == selfupdate.SourceServe && !selfupdate.IsCanonical(st.Executable, stateDir) {
		lines = append(lines, fmt.Sprintf(
			"loomux serve runs from %s, not from %s; point the MCP entry at %s",
			st.Executable, canonical, canonical))
	}
	if st.Result == selfupdate.Failed {
		lines = append(lines, fmt.Sprintf("updating loomux failed at %s: %s; run loomux upgrade to retry",
			st.CheckedAt.UTC().Format(time.RFC3339), st.Error))
	}
	// A pass that kept or replaced the binary but could not read or write the
	// channel marker leaves the machine quietly on stable.
	if (st.Result == selfupdate.Current || st.Result == selfupdate.Updated) && st.Error != "" {
		lines = append(lines, fmt.Sprintf("loomux update channel at %s: %s; run loomux upgrade --beta or --stable to set it",
			st.CheckedAt.UTC().Format(time.RFC3339), st.Error))
	}
	return lines
}

// newestSource is the latest modification among what goes into the build, with
// that file's slash-separated path: go.mod, go.sum, the .go files under cmd/
// and internal/, and every file under flows/, which holds Go source and the
// catalog the binary embeds. Under flows/ a name starting with "_" or "." is
// left out with everything below it, as go:embed leaves it out -- a flow's
// _test/ folder never reaches the binary.
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
	for _, dir := range []string{"cmd", "internal", "flows"} {
		err := walkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if errors.Is(err, fs.ErrNotExist) && path == filepath.Join(root, dir) {
				return filepath.SkipDir
			}
			if err != nil {
				return err
			}
			embedded := dir == "flows"
			if embedded && (strings.HasPrefix(entry.Name(), "_") || strings.HasPrefix(entry.Name(), ".")) {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() || (!embedded && filepath.Ext(path) != ".go") {
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
