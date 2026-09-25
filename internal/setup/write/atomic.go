// Package write decides first and writes second.
//
// The split is what makes --dry-run honest: Prepare answers "what would
// change?" by reading only, and Commit is the same answer applied. Nothing
// Commit does can change an answer Prepare gave, because every decision was
// taken before the first byte landed.
//
// It does not make the write a transaction. A failure on the third file
// leaves the first two on disk, and this package offers no undo -- Commit
// names them instead, so a caller can report exactly what landed rather than
// claiming nothing did. Temporary names and a rename at the end would not
// change that: renaming three of five can fail on the fourth just as well.
// What it does guarantee is that loomux init only ever adds: anything already standing under
// a name -- file, directory, symlink, even one pointing nowhere -- is skipped,
// and Commit refuses rather than overwrites whatever is there when it arrives.
// Whatever a failed run leaves behind, none of it is somebody else's content.
package write

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Plan is what Prepare decided, in the same slash-separated names the
// renderer used: Create maps each new file to its body, Skip names the files
// left alone.
//
// A dry-run summary prints both halves, and two runs over an unchanged
// project should read alike, so both halves come with an order: Skip is
// sorted, and Names gives the sorted Create keys. Ranging over Create
// directly gives a different order every time -- use Names.
type Plan struct {
	Create map[string]string
	Skip   []string
}

// Names is the order Commit works in, so what a caller printed beforehand and
// what a failed run left behind line up.
func (p Plan) Names() []string {
	names := make([]string, 0, len(p.Create))
	for name := range p.Create {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Prepare sorts the rendered files into those that would be written and those
// that already exist. It touches nothing.
func Prepare(root string, files map[string]string) (Plan, error) {
	plan := Plan{Create: map[string]string{}}
	for name, body := range files {
		if err := checkName(name); err != nil {
			return Plan{}, err
		}
		full := filepath.Join(root, filepath.FromSlash(name))
		// Lstat, not Stat: a symlink pointing nowhere is still somebody's
		// property. Stat would follow it, report "not found" and put the name
		// up for creation, and the exclusive create would then fail on a link
		// that had been there all along. Anything present under the name is
		// something this run did not put there.
		switch _, err := os.Lstat(full); {
		case err == nil:
			plan.Skip = append(plan.Skip, name)
		case os.IsNotExist(err):
			plan.Create[name] = body
		default:
			return Plan{}, fmt.Errorf("looking at %s: %w", name, err)
		}
	}
	sort.Strings(plan.Skip)
	return plan, nil
}

// checkName refuses a name that could escape root. It reads the name and
// nothing else -- it does not resolve the path, so what the name means on
// this disk is CheckParents' question, not this one's. The renderer is the
// only caller today, but a tool that writes into a stranger's project earns
// its trust by not needing any: a name that escapes is a bug wherever it came
// from, and the safe answer to a bug is to stop.
func checkName(name string) error {
	// fs.ValidPath is the slash-world rule: no empty name, no leading slash,
	// no "." or ".." element, no doubled or trailing separator.
	if !fs.ValidPath(name) || name == "." {
		return fmt.Errorf("refusing the file name %q: it is not a relative path inside the project", name)
	}
	// Windows reads two more characters as structure that fs.ValidPath, which
	// knows only slashes, hands through as ordinary letters: a backslash
	// separates, and a colon opens a drive or an alternate data stream.
	if strings.ContainsAny(name, `\:`) {
		return fmt.Errorf("refusing the file name %q: it contains a separator or a drive letter", name)
	}
	return nil
}

// Commit writes the files Prepare chose and reports which of them landed. A
// Plan can also be built by hand, so the names are checked again here: the
// promise not to write outside root belongs to the package, not to one of its
// two entry points.
//
// The written names come back on the failure path too, and that is what they
// are for: this is not a transaction, so a failure on the third file leaves
// the first two standing, and a caller that reported "nothing written" over
// that would be lying by two files. Whatever stands there is this run's own
// -- Commit never overwrites -- but it stands, and the caller has to be able
// to say which.
func Commit(root string, plan Plan) ([]string, error) {
	var written []string
	for _, name := range plan.Names() {
		if err := checkName(name); err != nil {
			return written, err
		}
		if err := CheckParents(root, name); err != nil {
			return written, err
		}
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return written, fmt.Errorf("creating the directory for %s: %w", name, err)
		}
		if err := writeNew(full, plan.Create[name]); err != nil {
			return written, commitFailure(name, err)
		}
		written = append(written, name)
	}
	return written, nil
}

// commitFailure tells the two ways a planned file can fail to appear apart.
// A refusal is the package working as designed and reads as such; anything
// else is the disk saying no. Callers are owed that distinction by name --
// one is a project to leave alone, the other is a run to retry.
func commitFailure(name string, err error) error {
	if errors.Is(err, fs.ErrExist) {
		// Worded so it cannot be wrong: the plan listed the name as new, and
		// the disk disagrees. Who put it there and when is not knowable here.
		return fmt.Errorf("refusing to overwrite %s: the plan had it as new, but it exists: %w", name, err)
	}
	return fmt.Errorf("writing %s: %w", name, err)
}

// CheckParents is the half of the promise checkName cannot give.
//
// Exported, because one file does not travel through Commit: settings.json
// belongs to the project and may be written over, which is the one thing this
// package never does, so that write cannot move in here. The guard moves out
// instead -- "inside root" is a promise about the project, not about a code
// path, and every file loomux init writes answers to the same check.
//
// A name may be flawless and still land outside the project: if
// .loomux is a symlink or a junction pointing somewhere else, MkdirAll and
// OpenFile follow it without a word and a new file appears there. Nothing is overwritten -- the
// exclusive create sees to that -- but "inside root" was the promise, and a
// new file outside it breaks that promise just as well.
//
// Lstat each directory above the name, and stop at the first one that is not
// there: everything below a missing component is created by this run, so
// there is nothing left to follow.
func CheckParents(root, name string) error {
	parts := strings.Split(name, "/")
	full := root
	for _, part := range parts[:len(parts)-1] {
		full = filepath.Join(full, part)
		info, err := os.Lstat(full)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("looking at the directory above %s: %w", name, err)
		}
		// Symlink and irregular together, because Windows has two of these:
		// Go reports a symlink as ModeSymlink and a directory junction --
		// which any account may create, no privilege needed -- as
		// ModeIrregular. Both redirect, and neither is a directory of this
		// project.
		if info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			return fmt.Errorf(
				"refusing %s: %s is a link, and writing through it would put a new file outside the project",
				name, part)
		}
	}
	return nil
}

// writeNew creates a file that must not exist yet. O_EXCL is the skip
// decision enforced at the last moment: between Prepare and here somebody may
// have created the file, and this run has no claim on what they wrote.
func writeNew(full, body string) error {
	perm := fs.FileMode(0o644)
	if strings.Contains(full, ".githooks") || strings.HasSuffix(full, ".sh") {
		perm = 0o755
	}
	f, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	_, werr := f.WriteString(body)
	return errors.Join(werr, f.Close())
}
