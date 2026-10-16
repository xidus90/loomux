package hooks

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/gitwork"
)

// stopKeptTree is the seam for git failing on a repository it just answered
// about, as stopTree is for the copy that is not kept.
var stopKeptTree = gitwork.KeptContentTree

// StopIndex is the turn end's view for the root lanes (graph and rules):
// everything git does not ignore, against HEAD, through a copy of the index
// the caller closes.
type StopIndex struct {
	Index, Tree, HeadTree string
}

// OpenStopIndex builds the copy for a profile with the rules kind, or with
// the graph kind in a project that has a graph to read; nil without error for
// any other, and where there is no repository or no HEAD, which Ready and
// RulesReady then report.
func OpenStopIndex(root string, kinds []string) (*StopIndex, error) {
	if !wantsStopIndex(root, kinds) {
		return nil, nil
	}
	head, err := gitwork.Head(root)
	if err != nil || head == "" {
		return nil, nil
	}
	return keepStopIndex(root, head)
}

// keepStopIndex builds the copy against a HEAD the caller has already asked
// git for.
func keepStopIndex(root, head string) (*StopIndex, error) {
	tree, index, err := stopKeptTree(root)
	if err != nil {
		return nil, err
	}
	return &StopIndex{Index: index, Tree: tree, HeadTree: headTree(root, head)}, nil
}

// Ready is the graph lane's plan-time probe at a turn end, asking what
// query.GraphPrereq asks without git: the view itself proves a HEAD, and the
// copy lies in the worktree's own git directory, where git keeps the markers
// of a pending operation too. Then a change against HEAD for the lane to
// judge. The real index is not asked; at a turn end it is usually empty. The
// hook package stays clear of query, whose extractors carry tree-sitter.
func (s *StopIndex) Ready(root string) (bool, string) {
	if _, err := os.Stat(store.WiringPath(root)); err != nil {
		return false, "no graph at .loomux/state/graph/wiring.json"
	}
	if s == nil {
		return false, "no HEAD to compare with"
	}
	for _, m := range gitwork.InProgress {
		if _, err := os.Stat(filepath.Join(filepath.Dir(s.Index), m.Path)); err == nil {
			return false, m.Note
		}
	}
	if s.Tree == s.HeadTree {
		return false, "nothing changed against HEAD"
	}
	return true, ""
}

// RulesReady is the rules lane's plan-time probe at a turn end: it reads the
// copy, and without one (no HEAD yet) the real index is partial, so the lane
// sits out.
func (s *StopIndex) RulesReady(string) (bool, string) {
	if s == nil {
		return false, "no HEAD yet"
	}
	return true, ""
}

// Env hands the root jobs (graph and rules) the copy in place of the real
// index.
func (s *StopIndex) Env(string) []string {
	if s == nil {
		return nil
	}
	return []string{"GIT_INDEX_FILE=" + s.Index}
}

// Close removes the copy from the git directory.
func (s *StopIndex) Close() {
	if s != nil {
		os.Remove(s.Index)
	}
}

// wantsStopIndex says whether a turn end keeps its copy of the index: for a
// profile with the rules kind (which verify.ExpandProfile leaves out of a
// project without a [rules] table), whose check reads the index it is handed,
// and for one with the graph kind in a project that has a graph to read.
func wantsStopIndex(root string, kinds []string) bool {
	return slices.Contains(kinds, "rules") || wantsStopGraph(root, kinds)
}

// wantsStopGraph says whether the profile plans the graph lane in a project
// that has a graph to read: the only lane that judges against HEAD.
func wantsStopGraph(root string, kinds []string) bool {
	if !slices.Contains(kinds, "graph") {
		return false
	}
	_, err := os.Stat(store.WiringPath(root))
	return err == nil
}
