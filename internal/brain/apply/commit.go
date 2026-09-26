package apply

import (
	"errors"
	"fmt"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/brain/vcs"
)

// commitPaths is vcs.CommitPaths, as a variable so that a test can hand
// out a moved ref, a git failure or an unchanged tree on demand -- Python's
// tests monkeypatch `commit_paths` for the same reason.
var commitPaths = vcs.CommitPaths

// commit is `_commit` and `_report` (apply.py:1318-1378): the one commit of
// a decision, with exactly one retry after a foreign process moved the ref.
//
// It never fails the decision. By the time it runs the wiki, the protocols
// and the removal of the case are on disk, and an error would leave the
// caller to tell "nothing written" from "everything written, only not
// committed" by its kind alone; a git problem comes back as the warning
// instead, and sha is then empty. said is what the warning claims was done:
// `written` for an approval, `decision recorded` for a rejection, which
// leaves the page's text as it was and moves only its `sources[]`.
//
// scratch is the directory vcs keeps its scratch index in; the caller owns
// it, as Python's CLI hands over one below the state directory.
func commit(vault, caseID, said, message string, add, remove []string, scratch string) (sha, warning string) {
	message += "\n"
	retried := false
	done, err := commitPaths(vault, message, add, remove, scratch)
	if errors.Is(err, vcs.ErrRefMoved) {
		// Once only: the window is milliseconds wide, so losing it twice
		// means something commits continuously, and a loop would not end.
		retried = true
		done, err = commitPaths(vault, message, add, remove, scratch)
	}
	switch {
	case err != nil:
		return "", fmt.Sprintf("%s: %s, but not committed (%v)", caseID, said, err)
	case done == nil:
		return "", fmt.Sprintf("%s: no git repository in the vault; %s but not committed", caseID, said)
	case !done.Created:
		// After a retry this is the case going back into the queue, not a
		// success: a checkout in between can have overwritten the page the
		// guard cleared.
		where := "on the first attempt"
		if retried {
			where = "after the retry"
		}
		return "", fmt.Sprintf("%s: nothing to commit %s; the case belongs back in the queue", caseID, where)
	}
	return done.Head, ""
}

// staged is `_staged` (apply.py:683-695): the `git add` entry for a written
// path, or none for one outside the vault. vcs runs git inside the vault,
// so a write under a wiki outside it lands on disk there but cannot be
// staged, and is left out of the commit rather than handed to git as a
// path it cannot place.
func (p *place) staged(path string) ([]string, error) {
	resolved, err := resolvePath(path)
	if err != nil {
		return nil, err
	}
	vault, err := resolvePath(p.anchor)
	if err != nil {
		return nil, err
	}
	if !guard.IsRelativeTo(resolved, vault) {
		return nil, nil
	}
	return []string{p.relative(path)}, nil
}
