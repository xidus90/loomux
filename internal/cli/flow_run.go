package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/runs"
	"github.com/xidus90/loomux/internal/gitwork"
)

// flowOptions collects --option name=value, as often as it is given. A name
// given twice is refused: which of two values a run meant is not a guess to
// make.
type flowOptions map[string]string

func (o flowOptions) String() string { return "" }

func (o flowOptions) Set(text string) error {
	name, value, found := strings.Cut(text, "=")
	if !found || name == "" {
		return fmt.Errorf("want name=value, got %q", text)
	}
	if _, taken := o[name]; taken {
		return fmt.Errorf("option %s is given twice", name)
	}
	o[name] = value
	return nil
}

func flowRunCommand(args []string, deps flowDeps) int {
	flags, root := flowFlags("run")
	options := flowOptions{}
	flags.Var(options, "option", "a parameter of the flow as name=value; may be repeated")
	name, code, ok := flowArguments("run", args, optionalName, flags, root, deps.Stderr)
	if !ok {
		return code
	}
	s, err := openFlow(*root, name, options, deps)
	if err != nil {
		return flowRefuse(deps, err)
	}
	if s.model, err = s.modelFor(deps); err != nil {
		return flowRefuse(deps, err)
	}
	// Taken once, here, and carried in the marker from now on: asked again on a
	// resume, git would answer with the tree the run has meanwhile edited.
	s.baseline = flowBaseline(*root)
	if s.needsBaseline() && s.baseline == nil {
		return flowRefuse(deps, fmt.Errorf(
			"%s measures against the commit it starts from, and git gives %s none; start it inside a repository", s.found.Name, *root))
	}
	// The origin and the overlays go into the marker so that a resume can
	// tell whether the flow it finds then is still the one the run started on.
	marker := runs.Marker{
		Flow: s.found.Name, Origin: s.found.Origin, Overlays: s.found.Overlays, Options: options,
		Baseline: s.baseline, Version: bareVersion(),
	}
	if s.id, err = runs.Claim(*root, marker); err != nil {
		return flowRefuse(deps, err)
	}
	result, err := s.walker(deps, false).Run(context.Background())
	if err != nil && forgetUnstarted(*root, s.id, os.Stat) {
		// The run stays on disk, so the error names it: the next step is
		// `loomux flow show` of that run, and nothing else says its number.
		err = fmt.Errorf("run %s: %w", s.id, err)
	}
	return flowFinish(deps, s, result, err)
}

// flowBaseline is what a run starts from, or nil where git cannot say. Both
// questions are asked either way, so that one branch answers them: outside a
// repository both fail, and a commit without its changed files beside it would
// read like a whole baseline.
func flowBaseline(root string) *flow.Baseline {
	commit, headErr := gitwork.HeadCommit(root)
	dirty, changedErr := gitwork.ChangedFiles(root)
	if headErr != nil || changedErr != nil {
		return nil
	}
	return &flow.Baseline{Commit: commit, Dirty: dirty}
}

// forgetUnstarted takes back the marker of a run the runner refused before its
// first step, and reports whether the marker stays. Such a run has not
// happened, and a marker with no journal beside it would keep its number taken
// for nothing. A run with a journal keeps its marker: the journal records what
// happened, and the marker says which flow. So does a run whose journal cannot
// be looked at: only a journal that is absent makes a run unstarted. stat is
// os.Stat outside the tests, which cannot make it fail any other way portably.
//
// A marker that cannot be removed stays behind and costs one run number; the
// refusal the caller is about to print says what went wrong with the run,
// and a second message about its marker would bury it.
func forgetUnstarted(root, id string, stat func(string) (fs.FileInfo, error)) bool {
	if _, err := stat(runs.JournalPath(root, id)); !errors.Is(err, fs.ErrNotExist) {
		return true
	}
	_ = os.Remove(runs.MarkerPath(root, id))
	return false
}
