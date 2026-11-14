package setup

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config/schema"
)

// executed runs Execute over a synthetic plan in a throwaway project and
// returns the report and the ids of the actions that ran.
func executed(t *testing.T, f Facts, p Plan, approved map[string]bool) (Report, []string) {
	t.Helper()
	root := world(t, map[string]string{"go.mod": goMod})
	installBinary(t)
	var ran []string
	d := Deps{Run: func(a Action) error { ran = append(ran, a.ID); return nil }, BinaryThere: there}
	r, err := Execute(root, f, DefaultChoice(f, Answers{}), p, approved, d, "2.12.1", applyTime)
	if err != nil {
		t.Fatal(err)
	}
	return r, ran
}

func hookPlan() Plan {
	return Plan{
		Changes: []Change{
			{Part: "git-hooks", Path: ".githooks/pre-commit", After: "#!/bin/sh\n"},
			{Part: "git-hooks", Path: ".githooks/pre-push", After: "#!/bin/sh\n"},
		},
		Actions: []Action{{Part: "git-hooks", ID: "hooks-path"}},
	}
}

func TestExecuteDropsAnActionNobodyApproved(t *testing.T) {
	p := Plan{Actions: []Action{{ID: "graph-build"}, {ID: "model-pull"}}}
	r, ran := executed(t, Facts{}, p, map[string]bool{"model-pull": true})
	if !slices.Equal(ran, []string{"model-pull"}) || !slices.Equal(r.Refused, []string{"graph-build"}) {
		t.Errorf("ran %v, refused %v", ran, r.Refused)
	}
}

func TestExecuteLeavesThePlanItWasGiven(t *testing.T) {
	p := Plan{Actions: []Action{{ID: "graph-build"}}}
	executed(t, Facts{}, p, map[string]bool{})
	if len(p.Actions) != 1 || p.Actions[0].ID != "graph-build" {
		t.Errorf("actions = %v", p.Actions)
	}
}

func TestExecuteLetsTheHooksPathFollowItsHookFiles(t *testing.T) {
	for name, tc := range map[string]struct {
		approved map[string]bool
		runs     bool
	}{
		"none":  {map[string]bool{"hooks-path": true}, false},
		"one":   {map[string]bool{".githooks/pre-push": true}, true},
		"other": {map[string]bool{".githooks/pre-commit": true, "hooks-path": false}, true},
	} {
		r, ran := executed(t, Facts{}, hookPlan(), tc.approved)
		if slices.Contains(ran, "hooks-path") != tc.runs || slices.Contains(r.Refused, "hooks-path") == tc.runs {
			t.Errorf("%s: ran %v, refused %v", name, ran, r.Refused)
		}
	}
}

func TestExecuteAsksTheHooksPathOnItsOwnWithoutHookFiles(t *testing.T) {
	p := Plan{Actions: []Action{{Part: "git-hooks", ID: "hooks-path"}}}
	if _, ran := executed(t, Facts{}, p, map[string]bool{"hooks-path": true}); !slices.Equal(ran, []string{"hooks-path"}) {
		t.Errorf("approved: ran %v", ran)
	}
	if r, ran := executed(t, Facts{}, p, map[string]bool{}); len(ran) != 0 || !slices.Equal(r.Refused, []string{"hooks-path"}) {
		t.Errorf("not approved: ran %v, refused %v", ran, r.Refused)
	}
}

func TestExecuteDropsTheMergeHookWithADeclinedAreaAdd(t *testing.T) {
	p := Plan{Actions: []Action{{ID: "area-add"}, {ID: "merge-hook"}}}
	approved := func() map[string]bool { return map[string]bool{"merge-hook": true} }
	r, ran := executed(t, Facts{}, p, approved())
	if len(ran) != 0 || !slices.Equal(r.Refused, []string{"area-add", "merge-hook"}) {
		t.Errorf("no consent: ran %v, refused %v", ran, r.Refused)
	}
	// A declaration that already consents and is registered needs no area add.
	r, ran = executed(t, Facts{OnMerge: true, Registered: true}, p, approved())
	if !slices.Equal(ran, []string{"merge-hook"}) || !slices.Equal(r.Refused, []string{"area-add"}) {
		t.Errorf("consent held: ran %v, refused %v", ran, r.Refused)
	}
}

func TestNotesNameAModuleSwitchedOffWithoutAConfiguration(t *testing.T) {
	c := Choice{Modules: map[schema.Module]bool{schema.Brain: false, schema.Graph: true}, Parts: map[string]bool{}}
	notes := Notes(Facts{Checkout: true}, c, Plan{})
	if want := []string{"modules.brain = false is not written; the part config is off"}; !slices.Equal(notes, want) {
		t.Errorf("notes = %v", notes)
	}
	c.Parts["config"] = true
	if notes := Notes(Facts{Checkout: true}, c, Plan{}); len(notes) != 0 {
		t.Errorf("config on: notes = %v", notes)
	}
}

func TestNotesNameABuildNoCheckoutCanDo(t *testing.T) {
	p := Plan{Actions: []Action{{ID: "binary-build"}}}
	c := Choice{Parts: map[string]bool{}}
	if notes := Notes(Facts{}, c, p); len(notes) != 1 || !strings.HasPrefix(notes[0], "binary-build: "+NotCheckout) {
		t.Errorf("notes = %v", notes)
	}
	if notes := Notes(Facts{Checkout: true}, c, p); len(notes) != 0 {
		t.Errorf("checkout: notes = %v", notes)
	}
}
