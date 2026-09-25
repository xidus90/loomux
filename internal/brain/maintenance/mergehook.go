package maintenance

// The post-merge hook: where git looks for it, which areas want it, and the
// one line it leaves behind.
//
// Unlike the reference's hook, this one bakes nothing of the machine into the
// file. The reference filled in the repository's git directory, the branch and
// the event path at install time, and so needed a whole class of states for
// a hook that no longer said what the manifests said (`stale path`, `stale
// branch`), a refusal for a branch name that would have run as shell
// (`unsafe branch`), and one for two areas fighting over one file (`shared
// hook path`). Here the hook only calls `loomux merge-hook record`, which
// asks the registry at merge time, so the file is the same text everywhere
// and all four states have nothing left to describe.
//
// The original is `src/brain/maintenance/merge_events.py`.

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// Git runs git in dir and returns its trimmed stdout.
type Git func(dir string, args ...string) (string, error)

// HookState is one line of install, status or remove.
//
// Repo is the repository the area sits in, or the area's own path when it
// sits in none; Detail the hook file that was written, kept or refused.
// install reaches into a repository the user never named, so the file it
// touched belongs in the output rather than in the reader's assumptions.
type HookState struct{ State, Scope, Repo, Detail string }

// HookMarker tells our hook from one the user wrote: install refuses to
// overwrite a file without it and remove refuses to delete one.
const HookMarker = "# loomux post-merge hook"

// referenceMarker is the marker of the reference's hook. Such a file is the
// predecessor of this one in the same place and is replaced like our own.
const referenceMarker = "# brain post-merge hook"

const hookName = "post-merge"

// recordsRelative is where the installations are remembered. Without it
// status could not name an orphan at all: once an area is out of the
// registry, nothing else points at the repository its hook still sits in.
var recordsRelative = filepath.Join("maintenance", "hooks.tsv")

// HookText is the whole hook. No path of this machine and no branch is in
// it, so the same file is right in every repository and may be checked in.
func HookText() string {
	return "#!/bin/sh\n" +
		HookMarker + " -- records that something landed. Nothing more.\n" +
		"#\n" +
		"# It runs inside git merge: it may not block, fail or print. Which repository\n" +
		"# and branch count is decided by `loomux merge-hook record` from the registry,\n" +
		"# so nothing of this machine is baked in and the file may be checked in.\n" +
		"\"${LOCALAPPDATA}/loomux/bin/loomux.exe\" merge-hook record >/dev/null 2>&1\n" +
		"exit 0\n"
}

// HookFailed reports whether a run failed to do what it was asked. An
// orphan, a missing hook or an unrecorded one is a finding to report; a
// refusal and an area outside any repository are the command failing.
func HookFailed(states []HookState) bool {
	for _, s := range states {
		if s.State == "refused" || s.State == "no repository" {
			return true
		}
	}
	return false
}

// hookTarget is a registered area whose declaration consents to the hook.
// repo and hook are empty when the area's path is in no repository.
type hookTarget struct {
	scope, path, repo, hook string
}

// hookRecord is one installation as it is remembered between runs.
type hookRecord struct {
	scope, repo, hook string
}

// hookScene is what every command starts from: what was installed, and who
// consents now.
func hookScene(areas []config.Area, lookup config.ArtifactLookup, git Git) ([]hookRecord, []hookTarget, error) {
	records, err := readRecords(lookup)
	if err != nil {
		return nil, nil, err
	}
	manifests, err := Manifests(areas, lookup)
	if err != nil {
		return nil, nil, err
	}
	var targets []hookTarget
	for _, area := range areas {
		// A missing declaration is a silent no: consent has to be written
		// down, and an area that never declared itself has not declared this.
		manifest := manifests[area.Scope]
		if manifest == nil || !manifest.OnMerge {
			continue
		}
		targets = append(targets, targetOf(area, git))
	}
	return records, targets, nil
}

// targetOf finds the area's repository and the file git runs after a merge
// there. Not `.git/hooks`: core.hooksPath moves them, and a hook written into
// the wrong place is never run while looking fine.
func targetOf(area config.Area, git Git) hookTarget {
	// The path in the platform's spelling, like every repository and hook
	// path beside it: a registry writes slashes, and the reference printed
	// an area outside any repository as Windows spells it.
	target := hookTarget{scope: area.Scope, path: cleanPath(area.Path)}
	root, err := git(area.Path, "rev-parse", "--path-format=absolute", "--show-toplevel")
	if err != nil {
		return target
	}
	hooks, err := git(area.Path, "rev-parse", "--path-format=absolute", "--git-path", "hooks")
	if err != nil {
		return target
	}
	target.repo = cleanPath(root)
	target.hook = filepath.Join(cleanPath(hooks), hookName)
	return target
}

// InstallHooks writes the hook into every consenting repository and
// remembers where.
//
// Two areas behind one hooks directory share one file: the second writes the
// same text again and gets its own record.
func InstallHooks(areas []config.Area, lookup config.ArtifactLookup, git Git) ([]HookState, error) {
	records, targets, err := hookScene(areas, lookup, git)
	if err != nil {
		return nil, err
	}
	var found []HookState
	for _, target := range targets {
		if target.repo == "" {
			found = append(found, HookState{State: "no repository", Scope: target.scope, Repo: target.path})
			continue
		}
		if _, err := os.Stat(target.hook); err == nil && !isOurs(target.hook) {
			found = append(found, HookState{State: "refused", Scope: target.scope, Repo: target.repo, Detail: target.hook})
			continue
		}
		if err := writeHook(target.hook); err != nil {
			// The hooks written before this one keep their records; without
			// them status would call those hooks unrecorded.
			return nil, errors.Join(err, writeRecords(lookup, records))
		}
		records = upsert(records, hookRecord{scope: target.scope, repo: target.repo, hook: target.hook})
		found = append(found, HookState{State: "installed", Scope: target.scope, Repo: target.repo, Detail: target.hook})
	}
	if err := writeRecords(lookup, records); err != nil {
		return nil, err
	}
	return found, nil
}

// HookStatus is what is installed, what went missing, and what nobody wants
// any more.
func HookStatus(areas []config.Area, lookup config.ArtifactLookup, git Git) ([]HookState, error) {
	records, targets, err := hookScene(areas, lookup, git)
	if err != nil {
		return nil, err
	}
	byScope := map[string]hookTarget{}
	for _, target := range targets {
		byScope[target.scope] = target
	}
	var found []HookState
	recorded := map[string]bool{}
	for _, record := range records {
		recorded[record.scope] = true
		state := "installed"
		target, ok := byScope[record.scope]
		switch {
		case !ok || !samePath(target.repo, record.repo):
			// The area left the registry, withdrew its consent or moved; the
			// hook is still in that repository and still calls record.
			state = "orphaned"
		case !isFile(record.hook):
			state = "missing"
		}
		found = append(found, HookState{State: state, Scope: record.scope, Repo: record.repo, Detail: record.hook})
	}
	for _, target := range targets {
		if recorded[target.scope] {
			continue
		}
		if target.repo == "" {
			found = append(found, HookState{State: "not installed", Scope: target.scope, Repo: target.path})
			continue
		}
		state := "not installed"
		if isOurs(target.hook) {
			// The record was lost, the hook was not: calling that "not
			// installed" would send the user looking for a hook they have.
			state = "unrecorded"
		}
		found = append(found, HookState{State: state, Scope: target.scope, Repo: target.repo, Detail: target.hook})
	}
	return found, nil
}

// RemoveHooks takes back every hook this tool wrote, orphans and hooks whose
// record was lost included. A hook somebody replaced with their own is
// refused and its record kept.
func RemoveHooks(areas []config.Area, lookup config.ArtifactLookup, git Git) ([]HookState, error) {
	records, targets, err := hookScene(areas, lookup, git)
	if err != nil {
		return nil, err
	}
	var found []HookState
	var kept []hookRecord
	// handled answers a hook file a second area shares with the same state,
	// rather than calling it missing because the first one took it.
	handled := map[string]string{}
	for _, record := range records {
		state, err := takeOnce(record.hook, handled)
		if err != nil {
			return nil, err
		}
		if state == "refused" {
			kept = append(kept, record)
		}
		found = append(found, HookState{State: state, Scope: record.scope, Repo: record.repo, Detail: record.hook})
	}
	for _, target := range targets {
		// Without this, the answer to a lost record would be "nothing to do"
		// while the hooks keep running -- a success that is not one.
		if target.repo == "" || !isOurs(target.hook) {
			continue
		}
		if _, err := takeOnce(target.hook, handled); err != nil {
			return nil, err
		}
		found = append(found, HookState{State: "removed", Scope: target.scope, Repo: target.repo, Detail: target.hook})
	}
	if err := writeRecords(lookup, kept); err != nil {
		return nil, err
	}
	return found, nil
}

// takeOnce is take, answering a file this run already handled from memory.
func takeOnce(hook string, handled map[string]string) (string, error) {
	key := pathKey(hook)
	if state, ok := handled[key]; ok {
		return state, nil
	}
	state, err := take(hook)
	handled[key] = state
	return state, err
}

func take(hook string) (string, error) {
	if !isFile(hook) {
		return "missing", nil
	}
	if !isOurs(hook) {
		return "refused", nil
	}
	return "removed", os.Remove(hook)
}

// RecordMerge appends one event for the merge that just landed in dir, when
// a registered area of the same repository consents to merges on the branch
// dir is on. It answers whether it wrote one.
//
// The repository is compared by its common git directory, not its top level:
// every linked worktree has its own top level while sharing the one hook, so
// comparing top levels would go silent for every merge outside the checkout
// the area was registered from. The top level is still what is recorded: it
// says which working tree the merge happened in.
//
// A git question that fails -- no repository, a detached HEAD, no ORIG_HEAD
// -- is no merge to record and no error: this runs inside git merge.
func RecordMerge(dir string, areas []config.Area, lookup config.ArtifactLookup, git Git, now time.Time) (bool, error) {
	facts, ok := askGit(git, dir,
		[]string{"rev-parse", "--path-format=absolute", "--git-common-dir"},
		[]string{"rev-parse", "--show-toplevel"},
		[]string{"symbolic-ref", "--quiet", "--short", "HEAD"},
		[]string{"rev-parse", "--quiet", "--verify", "ORIG_HEAD"},
		[]string{"rev-parse", "HEAD"},
	)
	if !ok {
		return false, nil
	}
	common, here, branch, first, last := facts[0], facts[1], facts[2], facts[3], facts[4]
	manifests, err := Manifests(areas, lookup)
	if err != nil {
		return false, err
	}
	for _, area := range areas {
		manifest := manifests[area.Scope]
		if manifest == nil || !manifest.OnMerge || manifest.MergeBranch != branch {
			continue
		}
		theirs, err := git(area.Path, "rev-parse", "--path-format=absolute", "--git-common-dir")
		if err != nil || !samePath(theirs, common) {
			continue
		}
		// Once, however many areas share the repository: the event names a
		// repository and a range, and which area it concerns is derived
		// again at reconciliation time.
		event := MergeEvent{Repo: here, First: first, Last: last, Branch: branch, At: now.UTC()}
		if err := AppendEvent(lookup.Primary, event); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// askGit asks git each question in dir, or answers false at the first that
// fails.
func askGit(git Git, dir string, questions ...[]string) ([]string, bool) {
	answers := make([]string, 0, len(questions))
	for _, question := range questions {
		answer, err := git(dir, question...)
		if err != nil {
			return nil, false
		}
		answers = append(answers, answer)
	}
	return answers, true
}

// isOurs reports whether hook is a file we or the reference wrote. Read as
// bytes: a foreign post-merge may be a compiled program, and failing to
// decode a file we are only refusing to touch would be an error over nothing.
func isOurs(hook string) bool {
	data, err := os.ReadFile(hook)
	return err == nil && OwnsHook(data)
}

// OwnsHook reports whether data is the text of a post-merge hook we or the
// reference wrote.
func OwnsHook(data []byte) bool {
	return bytes.Contains(data, []byte(HookMarker)) || bytes.Contains(data, []byte(referenceMarker))
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// writeHook writes HookText with LF endings -- this writes into a foreign
// repository, where no .gitattributes of ours reaches, and git's sh chokes on
// a carriage return after the shebang -- and makes it executable.
func writeHook(path string) error {
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err == nil {
		err = lock.ReplaceText(path, HookText())
	}
	if err == nil {
		err = os.Chmod(path, 0o755)
	}
	return err
}

// upsert replaces the record of the same scope in place, or appends it.
func upsert(records []hookRecord, record hookRecord) []hookRecord {
	for i := range records {
		if records[i].scope == record.scope {
			records[i] = record
			return records
		}
	}
	return append(records, record)
}

// readRecords is every remembered installation. A line of the reference
// carries four or five fields -- the branch and the event path it baked in --
// and is read by its first three; any other count is a damaged line and
// skipped, like a damaged event.
func readRecords(lookup config.ArtifactLookup) ([]hookRecord, error) {
	lines, err := readLines(lookup.WritePath(recordsRelative))
	if err != nil {
		return nil, err
	}
	var records []hookRecord
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) < 3 || len(fields) > 5 {
			continue
		}
		records = append(records, hookRecord{scope: fields[0], repo: fields[1], hook: fields[2]})
	}
	return records, nil
}

// writeRecords rewrites the record file whole, and deletes it when nothing
// is left to remember.
func writeRecords(lookup config.ArtifactLookup, records []hookRecord) error {
	path := lookup.WritePath(recordsRelative)
	if len(records) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	var body strings.Builder
	for _, r := range records {
		body.WriteString(r.scope + "\t" + r.repo + "\t" + r.hook + "\n")
	}
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err == nil {
		err = lock.ReplaceText(path, body.String())
	}
	return err
}

// cleanPath is git's forward-slash path in the platform's spelling.
func cleanPath(path string) string {
	return filepath.Clean(filepath.FromSlash(path))
}

// samePath compares two paths as Windows does, without regard to case or
// slash direction.
func samePath(a, b string) bool {
	return strings.EqualFold(cleanPath(a), cleanPath(b))
}

func pathKey(path string) string {
	return strings.ToLower(cleanPath(path))
}
