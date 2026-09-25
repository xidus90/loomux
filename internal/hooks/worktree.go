// The worktree side of the guard: what a working tree cannot own itself.

package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/sessions"
	"github.com/xidus90/loomux/internal/worktree/junction"
	"github.com/xidus90/loomux/internal/worktree/mirror"
	"github.com/xidus90/loomux/internal/worktree/topo"
)

// The two prefixes a stored reparse target can start with. `\??\` is the NT
// object-manager form, it is not a path Go opens, and it is the only one
// measured here: on 2026-09-07 both junction.Create and `mklink /J` stored
// it. `\\?\` is taken off as well so that both forms compare alike -- and not
// because anything here was seen to store it. Neither prefix identifies
// anything: the reparse tag does that, and junction.Target has checked it.
var ntPrefixes = []string{`\??\`, `\\?\`}

// WorktreeLink puts the configured directories in place and sweeps what is
// left of gone worktrees.
//
// Silent on success, because it runs at every session start in every project
// on the machine. The only thing it reports is a junction that was needed and
// could not be made -- and that one it reports loudly, since the directory it
// stands for may be the pinned runtime every other hook needs.
//
// stdout is taken and not written to: every answer this subcommand has is
// either silence or a fault, and a fault belongs on stderr.
func WorktreeLink(stdout, stderr io.Writer, root string) int {
	topology, err := topo.Read(root)
	if err != nil {
		// Every failure of that call is the same answer, and the answer is
		// "nothing to do": topo.Read wraps ErrNoRepository around each
		// of its two error returns rather than telling a broken git from an
		// unrepositoried directory. Distinguishing them here would be a branch
		// nothing can enter.
		return ExitOK
	}
	mirrored, err := mirror.Mirror(topology.Main)
	if err != nil {
		fmt.Fprintf(stderr, "loomux worktree link: %v\n", err)
		return ExitInternal
	}
	if len(mirrored) == 0 {
		return ExitOK
	}

	code := ExitOK
	if topology.IsWorktree(root) {
		if err := link(root, topology.Main, mirrored); err != nil {
			fmt.Fprintf(stderr, "loomux worktree link: %v\n", err)
			code = ExitInternal
		}
	}
	// The sweep runs whether or not this directory is a worktree: a session in
	// the main checkout is the ordinary way to notice that a worktree is gone.
	if err := sweep(topology, mirrored); err != nil {
		fmt.Fprintf(stderr, "loomux worktree link: %v\n", err)
		code = ExitInternal
	}
	return code
}

// How long a session's state file counts for. Nothing deletes these files,
// and a session that ends without a SessionEnd gets no end marker, so for it
// the mtime is the only liveness there is to read, and it is as good as the
// writes: `ulguard hook session-start` writes at session start -- as
// session_start.py (fa3dd38):59 did before it -- stop.py on every block and
// every pass (:250, :283), and subagent_start.py on every subagent
// dispatch (:38) -- the last of those gated on the payload alone and on no
// configuration at all. So the file is as young as the last turn that ended,
// or the last subagent dispatched, and only a session that does neither ages
// past its start.
//
// A day, because the two errors are not the same size. Too long leaves a
// junction standing, and that costs nothing: it occupies no disk, `link` skips
// it at the next session start as already there, and `sweep` takes it out once
// the worktree is gone -- the same state `link` deliberately creates. Too
// short takes it out from under a live session or an open Godot editor reading
// 4.2 GB through it. A working day is also the unit in which a human answers
// "is that session still mine?".
//
// No number closes this hole for a session that is still running and writes
// nothing. A session that ends is marked ended by the SessionEnd side (Retire),
// and one that resumes is revived by session-start, which writes its file
// back or makes it young; the age decides only for a session that ended without a SessionEnd,
// or one still running that has written nothing for a day.
const sessionStale = 24 * time.Hour

// WorktreeUnlink takes the junctions back out -- but only if this was the
// last session on the tree.
//
// CLAUDE.md documents sessions that share a checkout. Unlinking
// unconditionally would pull `.tools` out from under a session still running,
// or under an open Godot editor, and the 4.2 GB behind it is exactly what that
// editor is reading from.
//
// stdout is taken and not written to, for worktree-link's reason: every answer
// this subcommand has is either silence or a fault.
func WorktreeUnlink(stdout, stderr io.Writer, stdin io.Reader, root string) int {
	var payload struct {
		SessionID string `json:"session_id"`
	}
	// A payload we cannot read is not a reason to remove anything: without an
	// id there is no way to tell our own state file from somebody else's, so
	// the count would always say "somebody else is here" -- and retiring would
	// be worse than useless, since safeName turns an empty id into "unnamed"
	// and that may be the collapsed name of a session that is still running.
	if err := json.NewDecoder(stdin).Decode(&payload); err != nil || payload.SessionID == "" {
		return ExitOK
	}

	topology, err := topo.Read(root)
	if err != nil {
		// Nothing to do, and the same collapse worktree-link makes: Read wraps
		// ErrNoRepository around each of its two error returns, so telling them
		// apart here would be a branch nothing can enter.
		return ExitOK
	}
	if !topology.IsWorktree(root) {
		return ExitOK
	}
	mirrored, err := mirror.Mirror(topology.Main)
	if err != nil {
		fmt.Fprintf(stderr, "loomux worktree unlink: %v\n", err)
		return ExitInternal
	}
	if len(mirrored) == 0 {
		return ExitOK
	}

	// Our own session is marked ended before the count and before the early
	// return below: whether or not this was the last session, the next run
	// must not read a session that has ended as one still standing. Marked,
	// not removed: a resume under the same id needs the stop gate's base.
	if err := sessions.Retire(root, payload.SessionID); err != nil {
		fmt.Fprintf(stderr, "loomux worktree unlink: %v\n", err)
		return ExitInternal
	}
	others, err := sessions.Others(root, payload.SessionID, sessionStale)
	if err != nil {
		fmt.Fprintf(stderr, "loomux worktree unlink: %v\n", err)
		return ExitInternal
	}
	if others > 0 {
		return ExitOK
	}
	if err := unlink(root, topology.Main, mirrored); err != nil {
		fmt.Fprintf(stderr, "loomux worktree unlink: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

// WorktreeRemove is the safe way to get rid of a worktree.
//
// Measured on 2026-09-07 in a t.TempDir() fixture: `git worktree remove
// --force` on a worktree holding a junction exits 0 with no output, drops the
// porcelain entry, and leaves both the directory and the junction standing.
// The directory is then a tree git no longer knows, with a link into the main
// checkout still in it. So the junctions come out first, and git is asked
// afterwards.
//
// Two refusals, and both before anything is touched -- the read of the
// topology is the only thing that happens ahead of them, and it only asks git
// a question. The main checkout is refused first, because a wrapper whose
// worst outcome is deleting the repository has to say no to that one before
// anything else; a directory git holds no working tree at is refused because
// there is then nothing here to remove and every candidate is somebody's data.
// Both rest on os.SameFile identity and not on the spelling of the argument,
// so no relative form, trailing separator or link path gets past them.
//
// This one writes to stdout, unlike worktree-link and worktree-unlink: those
// fire at every session start and end in every project, this one is run by
// hand, and a person deleting something should read what was deleted.
func WorktreeRemove(stdout, stderr io.Writer, target string) int {
	topology, err := topo.Read(target)
	if err != nil {
		// A fault here and not the hook commands' silent "nothing to do": this
		// directory was named by hand, and the name was wrong.
		fmt.Fprintf(stderr, "loomux worktree remove: %v\n", err)
		return ExitInternal
	}
	if sameDir(target, topology.Main) {
		fmt.Fprintf(stderr,
			"loomux worktree remove: %s is the main checkout\n", target)
		return ExitInternal
	}
	worktree := registeredAs(topology, target)
	if worktree == "" {
		fmt.Fprintf(stderr,
			"loomux worktree remove: git does not hold %s as a worktree\n", target)
		return ExitInternal
	}
	mirrored, err := mirror.Mirror(topology.Main)
	if err != nil {
		// Reported rather than treated as "no mirror": which junctions are
		// ours is then unknown, and asking git while that is unknown is the
		// order this whole subcommand exists to avoid.
		fmt.Fprintf(stderr, "loomux worktree remove: %v\n", err)
		return ExitInternal
	}
	if err := unlink(worktree, topology.Main, mirrored); err != nil {
		fmt.Fprintf(stderr, "loomux worktree remove: %v\n", err)
		return ExitInternal
	}
	command := exec.Command("git", "worktree", "remove", "--force", worktree)
	command.Dir = topology.Main
	// See gitenv: GIT_DIR and its relatives outrank command.Dir, so without
	// the strip the removal would be asked of whatever GIT_DIR names instead
	// of this repository.
	command.Env = gitenv.Environ()
	if out, err := command.CombinedOutput(); err != nil {
		// git's own words go through: it is the only one that knows why it
		// refused. The junctions are already out by then, and nothing here
		// puts them back -- the next session's worktree-link does that.
		fmt.Fprintf(stderr, "loomux worktree remove: git: %v (%s)\n", err, out)
		return ExitInternal
	}
	// git's exit code says nothing about the directory. Measured on
	// 2026-09-07: with a junction inside it that nothing here took out -- no
	// mirror configured -- `git worktree remove --force` exited 0, dropped the
	// porcelain entry and left the tree standing. That is the leftover this
	// whole subcommand exists to prevent, so it is not something to print
	// "removed" over.
	//
	// The message says what was observed and nothing about the cause: a
	// junction none of ours, an open handle, an ACL and a virus scanner all
	// leave exactly this behind, and os.Lstat tells the four apart in no way
	// at all. The operator is the one who can look.
	if _, err := os.Lstat(worktree); err == nil {
		fmt.Fprintf(stderr,
			"loomux worktree remove: git dropped its entry but %s still stands;"+
				" nothing here removed it\n", worktree)
		return ExitInternal
	}
	fmt.Fprintf(stdout, "removed %s\n", worktree)
	return ExitOK
}

// registeredAs answers with git's own spelling of `target`, or "" if git holds
// no working tree there.
//
// The caller passes that spelling on rather than its own argument, because
// the two do not have to name the same directory: its git call runs with its
// working directory in the main checkout, so a relative argument would resolve
// there, while sameDir resolves it against this process's own. Taking the path
// out of the porcelain closes the gap and needs no filepath.Abs -- whose only
// failure mode, a working directory that cannot be read, no test can reach.
//
// Main is in Worktrees as well, so this matches it too -- which is harmless
// only because the caller has already refused it above.
func registeredAs(topology topo.Topology, target string) string {
	for _, worktree := range topology.Worktrees {
		if sameDir(target, worktree) {
			return worktree
		}
	}
	return ""
}

// unlink removes the configured junctions from one worktree.
//
// A configured path that is a real directory is left alone: it was never ours.
// The target is checked as well -- a junction pointing somewhere else is
// somebody's own arrangement, and removing it would be the same overreach as
// removing a real directory.
//
// standsInside first, for the reason written at that function: without it the
// candidate is inside the worktree by spelling only, and a junction at
// `<worktree>/.ultraloom` would make this remove the main checkout's own
// `.ultraloom/vendor` -- the pinned runtime every other hook needs, taken out
// at session end by the mechanism that exists to put it there.
//
//coverage:exempt junction.Remove fails only when the OS refuses the delete -- an open handle, a virus scanner or an ACL -- and a denied right on the link or its parent still let the removal through when it was measured on 2026-09-14
func unlink(worktree, main string, mirrored []string) error {
	for _, relative := range mirrored {
		if !standsInside(worktree, relative) {
			continue
		}
		path := filepath.Join(worktree, filepath.FromSlash(relative))
		target, err := junction.Target(path)
		if err != nil {
			return err
		}
		if target == "" || !leadsInto(target, main) {
			continue
		}
		if err := junction.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

// link makes every configured path in `worktree` lead into `main`.
//
// A path the worktree already owns is left alone, and so is one the main
// checkout does not have: the first may be this tree's own build output, and
// the second is nothing to mirror. Only an *absent* path here is ours to fill.
//
// An Lstat that fails for some other reason than absence is not caught, and
// the property that makes that safe is that such a path always ends in a
// reported fault rather than a silent skip -- by one of two routes since
// parentsPlainOrAbsent arrived, not one: that function refuses the path when
// the unreadable component is an intermediate, and junction.Create's os.Mkdir
// refuses it when it is the candidate itself. Measured on 2026-09-07 with a worktree that may
// not gain a subdirectory -- os.Lstat of the absent child still answers
// IsNotExist there, and the Mkdir inside Create is what refuses. Mkdir
// refusing an occupied path is the same property from the other side: it is
// why nothing here can overwrite what already stands at the path.
//
// parentsPlainOrAbsent before the create, for the reason written at that
// function: without it "inside the worktree" is spelling only, and this is
// the create side of what standsInside does for the two remove sides. The
// refusal is loud and not a skip, because a mirror that was needed and could
// not be made is the one thing this subcommand reports.
func link(worktree, main string, mirrored []string) error {
	for _, relative := range mirrored {
		target := filepath.Join(main, filepath.FromSlash(relative))
		if info, err := os.Stat(target); err != nil || !info.IsDir() {
			continue
		}
		path := filepath.Join(worktree, filepath.FromSlash(relative))
		if _, err := os.Lstat(path); err == nil {
			continue
		}
		if blocker, ok := parentsPlainOrAbsent(worktree, relative); !ok {
			return fmt.Errorf(
				"not making %s: %s is not a plain directory as far as Lstat can see",
				path, blocker)
		}
		// The parent may be missing: a configured path can be more than one
		// level deep, and only its first level is necessarily something git
		// brought along.
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("making room for %s: %w", path, err)
		}
		if err := junction.Create(path, target); err != nil {
			return fmt.Errorf("%s: %w", relative, err)
		}
	}
	return nil
}

// sweep removes our junctions from directories git no longer holds.
//
// What makes a junction ours is where it is and what it points at: a reparse
// point at a configured path inside an unregistered worktree directory,
// leading into the main checkout. All three conditions are load-bearing.
// CLAUDE.md records directories under `.claude/worktrees/` that share the main
// index and are not worktrees, so Orphans names live work as well -- a real
// directory there is somebody's data, and a junction pointing anywhere else is
// somebody's own arrangement.
//
// No ledger -- a list of junctions we made would be a second state that can
// drift, and after a `Remove-Item -Recurse` on a worktree it would be wrong
// immediately. The price is that the hand-made junctions in this repository's
// worktrees are adopted, which is right: they are indistinguishable from ours
// by target and place, and they were made for the same reason.
//
//coverage:exempt junction.Remove fails only when the OS refuses the delete, for the reason written at unlink
func sweep(topology topo.Topology, mirrored []string) error {
	orphans, err := topology.Orphans()
	if err != nil {
		return err
	}
	for _, orphan := range orphans {
		for _, relative := range mirrored {
			if !standsInside(orphan, relative) {
				continue
			}
			path := filepath.Join(orphan, filepath.FromSlash(relative))
			target, err := junction.Target(path)
			if err != nil {
				return err
			}
			if target == "" {
				// Not a link: somebody's directory, and none of our business.
				continue
			}
			if !leadsInto(target, topology.Main) {
				continue
			}
			if err := junction.Remove(path); err != nil {
				return err
			}
		}
	}
	return nil
}

// standsInside says whether a configured path really lies where its spelling
// says: every component between `dir` and the candidate itself a plain
// directory, and none of them a link. Both *removing* callers need it -- the
// sweep about an orphaned worktree directory, unlink about a live one. The
// create side has a rule of its own, at parentsPlainOrAbsent, because a
// missing component is nothing for link to refuse.
//
// Without it a caller establishes "inside `dir`" by spelling alone, and a
// path's spelling does not decide where it goes: an open with
// FILE_FLAG_OPEN_REPARSE_POINT keeps the *final* component from being
// followed and nothing else, so a junction at `<dir>/.ultraloom` makes
// `<dir>/.ultraloom/vendor` read and remove the reparse point of
// `<main>/.ultraloom/vendor` -- the pinned runtime, taken out by the very
// mechanism that exists to put it there.
//
// Mode().IsDir() is the test, and it is false for a junction under both
// GODEBUG settings this module can be built with; the measurement behind that
// is recorded at junction.go's Target. The last component is left out because
// junction.Target is what decides about that one, and being a link is exactly
// what qualifies it.
//
// A component that cannot be stat'ed counts as not a plain directory. The safe
// direction is to leave a candidate alone: a junction skipped costs a stale
// directory nobody deletes, and the other way costs somebody's data.
func standsInside(dir, relative string) bool {
	// mirrorcfg hands out cleaned, slash-separated paths, so this split has no
	// empty component and no trailing one.
	components := strings.Split(relative, "/")
	path := dir
	for _, component := range components[:len(components)-1] {
		path = filepath.Join(path, component)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsDir() {
			return false
		}
	}
	return true
}

// parentsPlainOrAbsent says whether every component between `dir` and the
// candidate itself is either a plain directory or not there at all, and names
// the first one that is neither.
//
// This is what the create side needs, and standsInside is not it: standsInside
// wants each of those components to *exist* as a plain directory, while link
// legitimately fills missing ones in -- a configured path can be more than one
// level deep, and only its first level is necessarily something git brought
// along. Absent is therefore allowed here and refused there.
//
// What may not stand between the two is a link. Windows follows every
// component of a path but the last, so with a junction at `<worktree>/.tools`
// the Lstat of `<worktree>/.tools/godot` asks about the junction's target:
// absent there reads as "ours to fill", and the MkdirAll and junction.Create
// that follow write into whatever the junction points at -- outside the
// worktree, and at a path no sweep of ours ever looks at. Measured on
// 2026-09-08 without this check: exit 0, no output, and a junction standing at
// `<main>/elsewhere/godot`.
//
// Mode().IsDir() is the test, for the reason and the measurement written at
// standsInside. A component Lstat cannot read at all is refused with the same
// statement -- it is not *shown* to be a plain directory, and the message
// says no more than that.
func parentsPlainOrAbsent(dir, relative string) (string, bool) {
	// mirrorcfg hands out cleaned, slash-separated paths, so this split has no
	// empty component and no trailing one.
	components := strings.Split(relative, "/")
	path := dir
	for _, component := range components[:len(components)-1] {
		path = filepath.Join(path, component)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			// Nothing below an absent component exists either, so there is no
			// further component to look at.
			return "", true
		}
		if err != nil || !info.Mode().IsDir() {
			return path, false
		}
	}
	return "", true
}

// leadsInto says whether a link target sits inside the main checkout.
//
// The comparison is by what the two paths open and never by how they are
// spelled. A stored target's trailing separator depends on who made the
// junction -- measured on 2026-09-07, junction.Create writes `\??\C:\dir\` and
// `mklink /J` writes `\??\C:\dir`, and both resolve -- so a text comparison
// against an absolute path would miss exactly the hand-made links this
// repository's worktrees already carry.
//
// Walking up rather than testing the directory itself, because a configured
// path is a path and not only a name: `.ultraloom/vendor` names a target two
// levels below the main checkout.
func leadsInto(target, main string) bool {
	for path := stripNTPrefix(target); ; path = filepath.Dir(path) {
		if sameDir(path, main) {
			return true
		}
		if parent := filepath.Dir(path); parent == path {
			return false
		}
	}
}

// stripNTPrefix turns a stored reparse target into a path Go can stat.
//
// One of the two prefixes comes off if it is there, and a target carrying
// neither passes through untouched -- junction.Target promises the substitute
// name as the filesystem stored it and nothing about which form that is. The
// Clean is what takes the trailing separator off, so no caller has to know
// whether one was stored.
func stripNTPrefix(target string) string {
	for _, prefix := range ntPrefixes {
		if rest, found := strings.CutPrefix(target, prefix); found {
			return filepath.Clean(rest)
		}
	}
	return filepath.Clean(target)
}

// sameDir compares two paths by what they open, not by how they are spelled.
// A path that cannot be stat'ed is not the directory in question: os.Stat
// fails on "" as it does on anything else absent, so nothing here needs a
// guard against the empty string.
func sameDir(a, b string) bool {
	left, err := os.Stat(a)
	if err != nil {
		return false
	}
	right, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(left, right)
}
