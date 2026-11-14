package setup

import (
	"fmt"
	"slices"
	"time"

	"github.com/xidus90/loomux/internal/config/schema"
)

// NotCheckout is why binary-build cannot run in a project that is no
// checkout of loomux but whose entries call the checkout's binary.
const NotCheckout = "the host entries call the checkout's own build (under ${LOCALAPPDATA}/loomux/dev/), but go.mod does not declare " +
	"github.com/xidus90/loomux, so init cannot build it here; build it, or remove the entries to get the installed binary"

// Deps are what Execute cannot do itself: run an action and say whether a
// binary stands where the entries call it.
type Deps struct {
	Run         Runner
	BinaryThere func() bool
}

// Notes are what the plan cannot say itself: a module switched off that no
// configuration records, and a build no checkout can do.
func Notes(f Facts, c Choice, p Plan) []string {
	var notes []string
	for _, m := range schema.Switchable() {
		if !c.moduleOn(m) && !c.Parts["config"] {
			notes = append(notes, fmt.Sprintf("modules.%s = false is not written; the part config is off", m))
		}
	}
	for _, a := range p.Actions {
		if a.ID == "binary-build" && !f.Checkout {
			notes = append(notes, "binary-build: "+NotCheckout)
		}
	}
	return notes
}

// HookFiles are the paths of the git hooks the plan writes.
func HookFiles(p Plan) []string {
	var paths []string
	for _, ch := range p.Changes {
		if ch.Part == "git-hooks" {
			paths = append(paths, ch.Path)
		}
	}
	return paths
}

// Execute settles what approved leaves of p, drops the actions it does not
// approve into Report.Refused and applies the rest. approved is keyed by the
// path of a change and the id of an action; Execute adds to it what follows
// from the answers.
//
// core.hooksPath and the hook files are one unit: pointing git at .githooks
// without a hook there would switch off the hooks git ran before, and a hook
// written there without the setting is never run. Where the plan writes hook
// files, the setting follows whether at least one was approved; where it
// writes none -- a clone whose .githooks is tracked -- the setting was asked
// on its own. Without area add the consent the merge hook waits for is not
// written, so the hook falls with a declined area-add.
func Execute(root string, f Facts, c Choice, p Plan, approved map[string]bool, d Deps, version string, now time.Time) (Report, error) {
	if hooks := HookFiles(p); len(hooks) > 0 {
		approved["hooks-path"] = slices.ContainsFunc(hooks, func(path string) bool { return approved[path] })
	}
	if !approved["area-add"] && !f.HookWanted() {
		approved["merge-hook"] = false
	}
	var refused []string
	p.Actions = slices.DeleteFunc(slices.Clone(p.Actions), func(a Action) bool {
		if !approved[a.ID] {
			refused = append(refused, a.ID)
		}
		return !approved[a.ID]
	})
	report, err := Apply(root, p, c, func(ch Change) bool { return approved[ch.Path] }, d.Run, d.BinaryThere, version, now)
	report.Refused = append(report.Refused, refused...)
	return report, err
}
