package cli

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/setup"
	"github.com/xidus90/loomux/internal/tui"
)

// graphOnNotBuilt is a checkout of loomux as init sees it: the graph module
// runs, but no part of it is set up by this run.
func graphOnNotBuilt(t *testing.T) (setup.Facts, setup.Choice) {
	t.Helper()
	f := setup.Facts{Root: t.TempDir()}
	c := setup.DefaultChoice(f, setup.Answers{})
	// Every other part on, so the other modules are offered as all and ask
	// for no part list.
	for _, p := range setup.Parts(f) {
		c.Parts[p.ID] = p.Module != schema.Graph
	}
	return f, c
}

// none offered where the module is on would read as "off" to a human; a
// module that is on with no part on is offered as each.
func TestPresetOffersEachForAModuleThatIsOnWithoutAPart(t *testing.T) {
	_, c := graphOnNotBuilt(t)
	graph := []setup.Part{{Module: schema.Graph, ID: "graph-build"}}
	if got := preset(c, schema.Graph, graph); got != "each" {
		t.Errorf("module on, no part on: %q, want each", got)
	}
	c.Modules[schema.Graph] = false
	if got := preset(c, schema.Graph, graph); got != "none" {
		t.Errorf("module off: %q, want none", got)
	}
	// The base module cannot be switched off; none only says no part.
	for id := range c.Parts {
		c.Parts[id] = false
	}
	if got := preset(c, schema.Base, []setup.Part{{Module: schema.Base, ID: "binary"}}); got != "none" {
		t.Errorf("base without a part: %q, want none", got)
	}
}

func TestTakingEveryOfferKeepsAModuleThatIsOnWithoutAPart(t *testing.T) {
	f, c := graphOnNotBuilt(t)
	// base, hooks, brain, graph, the graph's part list, language, scope.
	keys := tui.Keys("enter", "enter", "enter", "enter", "enter", "enter", "enter")
	if ok, err := interview(tui.Script(100, 40, keys...), f, &c, nil); !ok || err != nil {
		t.Fatalf("%v %v", ok, err)
	}
	if moduleOff(c, schema.Graph) || c.Parts["graph-build"] {
		t.Errorf("graph module off %v, graph-build %v; want on and not built", moduleOff(c, schema.Graph), c.Parts["graph-build"])
	}
}

// The console can end at any question; approvePlan passes that on for an
// action as it does for a change.
func TestApprovePlanReportsTheEndOfInput(t *testing.T) {
	for name, p := range map[string]setup.Plan{
		"change": {Changes: []setup.Change{{Path: "AGENTS.md", After: "x"}}},
		"action": {Actions: []setup.Action{{ID: "graph-build"}}},
	} {
		if ok, err := approvePlan(tui.Script(100, 40), p, map[string]bool{}); ok || err == nil {
			t.Errorf("%s: %v %v, want the end of input", name, ok, err)
		}
	}
}

func TestNoneSwitchesAModuleOff(t *testing.T) {
	f, c := graphOnNotBuilt(t)
	// Tab from the offered each is none.
	keys := tui.Keys("enter", "enter", "enter", "tab", "enter", "enter", "enter")
	if ok, err := interview(tui.Script(100, 40, keys...), f, &c, nil); !ok || err != nil {
		t.Fatalf("%v %v", ok, err)
	}
	if !moduleOff(c, schema.Graph) {
		t.Error("none left the graph module on")
	}
}

// llmQuestion is the text of the last question of init.
const llmQuestion = "let the LLM edit .loomux/config.toml of this project? (you can change this later with loomux config llm on|off)"

func TestAskLLMConfigAsksOnce(t *testing.T) {
	term := tui.Script(100, 40, tui.Keys("enter")...)
	on, err := askLLMConfig(term, "project/x")
	if on || err != nil {
		t.Fatalf("%v %v, want false and no error", on, err)
	}
	if strings.Count(term.Output(), llmQuestion) != 1 {
		t.Errorf("screen:\n%s", term.Output())
	}
	if on, _ := askLLMConfig(tui.Script(100, 40, tui.Keys("y")...), "x"); !on {
		t.Error("y did not turn it on")
	}
	if _, err := askLLMConfig(brokenTerminal{tui.Script(100, 40)}, "x"); err == nil {
		t.Error("a console error was swallowed")
	}
}
