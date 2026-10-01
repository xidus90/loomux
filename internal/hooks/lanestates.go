package hooks

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/verify"
)

// GateLanes are the keys of every lane root's gate can run, sorted: the lanes
// of every kind as a check plans them, and the wiki's. A lane with nothing to
// run here -- no command, a graph lane another stack carries -- is none, and
// so is one with nothing to check: a stack without tests has no test and no
// coverage lane. Neither can ever turn green, so neither is armed by a
// commit, and listing them would list them for good. The graph lane is one
// only where a graph was built. A tool that is missing and an import that
// was not made do not take a lane away: it is there, and red.
//
// Every kind and not one profile's: the stop hook and the pre-commit gate
// run different profiles, a config may narrow either, and a lane any of them
// runs is in probation until the file names it. A lane no profile runs is
// listed all the same and stays in probation for good: no commit arms it,
// only `loomux gate arm` does.
func GateLanes(root string) ([]string, error) {
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	facts := detect.Detect(os.DirFS(root))
	eff, err := editLoad(root, facts)
	if err != nil {
		return nil, err
	}
	kinds := verify.Kinds()
	jobs, err := stopPlan(eff, verify.Request{Kinds: kinds, Scope: verify.ScopeCheck}, verify.PlanEnv{
		Root: root, Loomux: "loomux", RunID: "gate",
		HasTests: verify.HasTests, ImportReady: verify.ImportReady,
		GraphReady: func(root string) (bool, string) {
			_, err := os.Stat(store.WiringPath(root))
			return err == nil, "no graph"
		},
	})
	if err != nil {
		return nil, err
	}
	jobs = append(jobs, WikiGateJobs(eff, facts, root, kinds)...)
	var keys []string
	for _, j := range jobs {
		if j.Pre != verify.StateNotApplicable && j.Pre != verify.StateUnavailable {
			keys = append(keys, verify.LaneKey(j))
		}
	}
	slices.Sort(keys)
	return slices.Compact(keys), nil
}

// LaneStates sorts the lanes of a gate by what the file says about them, and
// names the entries no lane answers to: a stack that left, an area renamed.
type LaneStates struct {
	Armed, Probation, Orphans []string
	// Ignored says git ignores the file: it reaches no commit then, and
	// what it says holds on this machine only.
	Ignored bool
}

// ReadLaneStates reads root's file and lays it over the lanes. Without the
// file every lane is armed, exists is false, and the lanes are not even
// planned: a project without probation pays nothing for the question.
func ReadLaneStates(root string) (states LaneStates, exists bool, err error) {
	armed, err := verify.ReadArmed(root)
	if err != nil || !armed.Exists {
		return LaneStates{}, false, err
	}
	lanes, err := GateLanes(root)
	if err != nil {
		return LaneStates{}, true, err
	}
	states.Ignored = gitwork.IgnoredPath(root, verify.ArmedFile)
	for _, key := range lanes {
		if slices.Contains(armed.Keys, key) {
			states.Armed = append(states.Armed, key)
		} else {
			states.Probation = append(states.Probation, key)
		}
	}
	for _, key := range armed.Keys {
		if !slices.Contains(lanes, key) {
			states.Orphans = append(states.Orphans, key)
		}
	}
	return states, true, nil
}
