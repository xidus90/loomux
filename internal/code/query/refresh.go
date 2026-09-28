package query

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
)

// RefreshGraph drives the graph at root to fresh for a gate: drift, a missing
// record, an outdated schema and a foreign extractor all rebuild. Only a
// missing graph, a failed probe or rebuild and a lock held past wait fail.
func RefreshGraph(root string, wait time.Duration, notice func(string)) (ask.Status, error) {
	if _, err := os.Stat(store.WiringPath(root)); errors.Is(err, os.ErrNotExist) {
		return ask.StatusClean, ErrNoGraph
	}
	force := ""
	g, err := store.Read(root)
	switch {
	case errors.Is(err, model.ErrSchemaVersion):
		force = "graph schema is outdated, rebuilding"
	case err != nil:
		force = fmt.Sprintf("graph is unreadable (%v), rebuilding", err)
	case g.Meta.Extractor != all.Version():
		force = fmt.Sprintf("graph was built by extractor %q, rebuilding", g.Meta.Extractor)
	}
	return ask.Refresh(root, all.Version(),
		func() error { _, _, err := Build(root, notice); return err },
		ask.RefreshOptions{Force: force, Wait: wait, Notice: notice})
}

// inProgress are the files and directories git keeps while an operation that
// fills the index with another commit's changes waits for the user, each with
// the note GraphPrereq gives (and GraphReady passes on). A rebase comes before a cherry-pick: it replays
// commits by picking them.
var inProgress = []struct{ path, note string }{
	{"MERGE_HEAD", "a merge is in progress"},
	{"rebase-merge", "a rebase is in progress"},
	{"rebase-apply", "a rebase is in progress"},
	{"CHERRY_PICK_HEAD", "a cherry-pick is in progress"},
	{"REVERT_HEAD", "a revert is in progress"},
}

// GraphPrereq says whether the graph lane can run at root at all: a graph to
// read, a HEAD to diff against, and no merge, rebase, cherry-pick or revert in
// progress. It leaves the index alone, so a probe after the commit can use it.
func GraphPrereq(root string) (bool, string) {
	if _, err := os.Stat(store.WiringPath(root)); err != nil {
		return false, "no graph at .loomux/state/graph/wiring.json"
	}
	// One git call: --git-path for each marker, since .git is a file in a
	// linked worktree, and HEAD last. Git prints one line per argument.
	args := []string{"rev-parse"}
	for _, m := range inProgress {
		args = append(args, "--git-path", m.path)
	}
	out, err := gitOutput(root, append(args, "HEAD")...)
	if err != nil {
		return false, "no HEAD to compare with: " + err.Error()
	}
	// Meanwhile the index holds another commit's changes as if they were
	// this commit's.
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i, p := range lines[:min(len(lines), len(inProgress))] {
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		if _, err := os.Stat(p); err == nil {
			return false, inProgress[i].note
		}
	}
	return true, ""
}

// GraphReady is the plan-time probe of the graph lane: GraphPrereq, and
// something staged for the lane to diff.
func GraphReady(root string) (bool, string) {
	if ok, note := GraphPrereq(root); !ok {
		return false, note
	}
	_, err := gitOutput(root, "diff", "--cached", "--quiet")
	var exit *exec.ExitError
	switch {
	case err == nil:
		return false, "nothing staged"
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		return true, ""
	}
	return false, err.Error()
}
