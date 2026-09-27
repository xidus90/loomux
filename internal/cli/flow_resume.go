package cli

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/flow/journal"
	"github.com/xidus90/loomux/internal/flow/load"
	"github.com/xidus90/loomux/internal/flow/runs"
)

func flowResumeCommand(args []string, deps flowDeps) int {
	flags, root := flowFlags("resume")
	text := flags.String("answer", "", "the answer to the run's open gate")
	id, code, ok := flowArguments("resume", args, requiredName, flags, root, deps.Stderr)
	if !ok {
		return code
	}
	if code, ok := flowRunNumber("resume", id, deps.Stderr); !ok {
		return code
	}
	// No --answer at all is a look at the gate, which asks again. An empty
	// --answer is an answer, and one no choice matches.
	var answer *string
	flags.Visit(func(set *flag.Flag) {
		if set.Name == "answer" {
			answer = text
		}
	})
	s, gate, err := flowRecorded(*root, id, deps)
	if err != nil {
		return flowRefuse(deps, err)
	}
	if gate == nil {
		// A resume over a complete journal would execute nothing and report
		// done -- exit 0 for a run nobody carried onward.
		return flowRefuse(deps, fmt.Errorf(
			"run %s is not waiting at a gate; there is nothing to answer. Use `loomux flow replay` to re-derive it, or `loomux flow run` to start a new one", id))
	}
	if s.model, err = s.modelFor(deps); err != nil {
		return flowRefuse(deps, err)
	}
	result, err := s.walker(deps, false).Resume(context.Background(), answer)
	return flowFinish(deps, s, result, err)
}

func flowReplayCommand(args []string, deps flowDeps) int {
	flags, root := flowFlags("replay")
	id, code, ok := flowArguments("replay", args, requiredName, flags, root, deps.Stderr)
	if !ok {
		return code
	}
	if code, ok := flowRunNumber("replay", id, deps.Stderr); !ok {
		return code
	}
	s, gate, err := flowRecorded(*root, id, deps)
	if err != nil {
		return flowRefuse(deps, err)
	}
	if gate != nil {
		return flowRefuse(deps, fmt.Errorf(
			"run %s never finished: it is waiting at gate %q; answer it with `loomux flow resume` before replaying", id, gate.Node))
	}
	// A replay executes no node, so it asks for no model.
	result, err := s.walker(deps, true).Run(context.Background())
	return flowFinish(deps, s, result, err)
}

// flowRunNumber lets through a run number, digits as runs.NextID hands them
// out, and refuses anything else as a usage error. The number is joined into
// the paths of the journal and the marker: a `../`, a separator or a drive
// would read a forged pair elsewhere in the project, and a resume without an
// answer would walk past a gate no human answered.
func flowRunNumber(command, id string, stderr io.Writer) (int, bool) {
	if id != "" && strings.Trim(id, "0123456789") == "" {
		return flowExitOK, true
	}
	fmt.Fprintf(stderr, "loomux flow %s: %q is no run number; a run number is digits, as loomux flow run hands them out\n%s\n", command, id, flowUsage)
	return flowExitUsage, false
}

// flowRecorded reads back what a run was started with -- its flow, where that
// came from, its options, its baseline -- and the gate it waits at, if any.
func flowRecorded(root, id string, deps flowDeps) (flowSession, *journal.PendingGate, error) {
	path := runs.JournalPath(root, id)
	if _, err := os.Stat(path); err != nil {
		return flowSession{}, nil, flowNoRun(root, id)
	}
	marker, err := runs.ReadMarker(runs.MarkerPath(root, id))
	if err != nil {
		return flowSession{}, nil, err
	}
	if marker == nil {
		return flowSession{}, nil, fmt.Errorf("run %q does not say which flow it belongs to", id)
	}
	gate, err := journal.Pending(path)
	if err != nil {
		return flowSession{}, nil, err
	}
	project, found, err := findFlow(root, marker.Flow, deps)
	if err != nil {
		return flowSession{}, nil, err
	}
	// Another flow.toml is another graph, and the open gate may not be in it.
	// Asked before that flow.toml is loaded, so that its parameters or its
	// findings do not speak first. Other overlays are the same graph with
	// other texts: the definition hashes name the nodes whose text changed.
	if flowSource(marker.Origin) != flowSource(found.Origin) {
		// A marker without an origin was written by hand; it names no source.
		return flowSession{}, nil, fmt.Errorf(
			"run %s started on %s and %s now resolves to %s, another flow.toml; start a new run with loomux flow run %s",
			id, cmp.Or(marker.Origin, "(none)"), marker.Flow, found.Origin, marker.Flow)
	}
	if !slices.Equal(marker.Overlays, found.Overlays) {
		flowWarn(deps, fmt.Sprintf("run %s started with overlays %s and now has %s",
			id, flowNames(marker.Overlays), flowNames(found.Overlays)))
	}
	s, err := project.loadFlow(root, found, marker.Options)
	if err != nil {
		return flowSession{}, nil, err
	}
	s.id, s.baseline = id, marker.Baseline
	if s.needsBaseline() && s.baseline == nil {
		// Taking one now would measure against the tree the run has meanwhile
		// edited, and everything it changed would count as untouched.
		return flowSession{}, nil, fmt.Errorf(
			"run %s was started outside a repository, and flow %s measures against a commit; start a new run with `loomux flow run`", id, marker.Flow)
	}
	return s, gate, nil
}

// flowSource is the flow.toml an origin reads: the project's folder or the
// catalog's. An origin this build does not write names no source, and so
// matches none.
func flowSource(origin string) string {
	switch origin {
	case load.OriginProject, load.OriginHides:
		return "project"
	case load.OriginBundled, load.OriginOverlay:
		return "bundled"
	}
	return origin
}

// flowNoRun refuses a run number nothing was journalled under, naming the
// runs folder with forward slashes as every other path is named.
func flowNoRun(root, id string) error {
	return fmt.Errorf("no run %q under %s", id, filepath.ToSlash(filepath.Join(root, runs.Dir)))
}
