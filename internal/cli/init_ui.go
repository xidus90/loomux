package cli

import (
	"cmp"
	"errors"
	"slices"
	"strings"
	"unicode"

	"github.com/xidus90/loomux/internal/config/edit"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/setup"
	"github.com/xidus90/loomux/internal/tui"
)

// moduleAnswers are the answers to a module's question.
var moduleAnswers = []string{"all", "each", "none"}

// moduleTitle names a module the way the human knows it.
func moduleTitle(m schema.Module) string {
	if m == schema.Brain {
		return "brain (wiki)"
	}
	return string(m)
}

// interview asks per module which parts to set up, then the commit
// language and, when the project becomes an area, its scope. It answers
// false when the human cancelled.
func interview(t tui.Terminal, f setup.Facts, c *setup.Choice, given map[schema.Module]string) (bool, error) {
	parts := setup.Parts(f)
	for _, m := range initModules {
		var mine []setup.Part
		var labels []string
		for _, p := range parts {
			if p.Module == m {
				mine = append(mine, p)
				labels = append(labels, p.ID)
			}
		}
		offered := cmp.Or(given[m], preset(*c, m, mine))
		answer, ok, err := tui.Input(t, "module "+moduleTitle(m)+" ("+strings.Join(labels, ", ")+")",
			offered, moduleAnswers, nil)
		if err != nil || !ok {
			return false, err
		}
		switch {
		case answer == offered && answer != "each":
			// The offer is what c already holds, a flag included; taking a
			// none that only says no part is on now keeps the module on.
			continue
		case answer != "each":
			switchModule(c, f, m, answer == "all")
			continue
		}
		rows := make([]tui.Row, len(mine))
		chosen := make([]bool, len(mine))
		for i, p := range mine {
			rows[i] = tui.Row{Group: string(m), Label: p.ID, Note: p.Label}
			chosen[i] = c.Parts[p.ID]
		}
		picked, ok, err := tui.Pick(t, "module "+moduleTitle(m)+": the parts to set up", rows, chosen)
		if err != nil || !ok {
			return false, err
		}
		for i, p := range mine {
			c.Parts[p.ID] = picked[i]
		}
		if m != schema.Base {
			c.Modules[m] = true
		}
	}
	lang, ok, err := tui.Input(t, "commit language", c.CommitLanguage, []string{"en", "de"}, nil)
	if err != nil || !ok {
		return false, err
	}
	c.CommitLanguage = lang
	if !c.Parts["area"] || moduleOff(*c, schema.Brain) {
		return true, nil
	}
	scope, ok, err := tui.Input(t, "scope of the area", c.Scope, nil, checkScope)
	if err != nil || !ok {
		return false, err
	}
	c.Scope = scope
	return true, nil
}

// preset is the answer c already gives for m: all when every part is on,
// none when every part or the module is off, each otherwise.
func preset(c setup.Choice, m schema.Module, mine []setup.Part) string {
	on := 0
	for _, p := range mine {
		if c.Parts[p.ID] {
			on++
		}
	}
	switch {
	case moduleOff(c, m) || on == 0:
		return "none"
	case on == len(mine):
		return "all"
	}
	return "each"
}

// checkScope refuses a scope area add could not take.
func checkScope(s string) error {
	if s == "" || strings.ContainsFunc(s, unicode.IsSpace) {
		return errors.New("a scope is not empty and has no spaces")
	}
	return nil
}

// approvePlan asks for every change and every action; approved gets the
// path or id of each one the human took. It answers false when the human
// cancelled.
func approvePlan(t tui.Terminal, p setup.Plan, approved map[string]bool) (bool, error) {
	for _, ch := range p.Changes {
		yes, err := tui.Confirm(t, "--- "+ch.Path+"\n"+edit.Diff(ch.Before, ch.After), "write "+ch.Path+"?")
		if err != nil {
			return false, err
		}
		approved[ch.Path] = yes
	}
	hooks := hookFiles(p)
	for _, a := range p.Actions {
		if a.ID == "hooks-path" && len(hooks) > 0 {
			// Asked with the hook files it points at; see hooksPathApproved.
			continue
		}
		yes, err := tui.Confirm(t, a.ID+": "+a.Describe, "run "+a.ID+"?")
		if err != nil {
			return false, err
		}
		approved[a.ID] = yes
	}
	return true, nil
}

// hookFiles are the paths of the git hooks the plan writes.
func hookFiles(p setup.Plan) []string {
	var paths []string
	for _, ch := range p.Changes {
		if ch.Part == "git-hooks" {
			paths = append(paths, ch.Path)
		}
	}
	return paths
}

// hooksPathApproved makes core.hooksPath and the hook files one unit:
// pointing git at .githooks without a hook there would switch off the
// hooks git ran before, and a hook written there without the setting is
// never run. Where the plan writes hook files, the setting follows
// whether at least one was approved; where it writes none -- a clone
// whose .githooks is tracked -- the setting was asked on its own.
func hooksPathApproved(p setup.Plan, approved map[string]bool) {
	hooks := hookFiles(p)
	if len(hooks) > 0 {
		approved["hooks-path"] = slices.ContainsFunc(hooks, func(path string) bool { return approved[path] })
	}
}
