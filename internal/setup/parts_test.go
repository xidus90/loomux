package setup

import (
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/config/schema"
)

func TestACheckoutGetsOnlyWhatItHasCheckedIn(t *testing.T) {
	files := fixture(t)
	files["go.mod"] = checkoutGoMod
	files[".git/"] = ""
	root := world(t, files)
	f := gather(t, root, ".githooks")
	want := map[string]bool{
		"binary": true, "config": true, "gitignore": true, "agents-md": false, "mcp-json": false,
		"tools": true, "host-entries": true, "git-hooks": true, "verify-skill": false,
		"area": false, "merge-hook": false, "brain-skills": false, "graph-build": false,
	}
	for _, p := range Parts(f) {
		if p.Default != want[p.ID] {
			t.Errorf("part %s default = %v, want %v", p.ID, p.Default, want[p.ID])
		}
	}
	if p := plan(t, f); len(p.Changes) != 0 {
		t.Errorf("a checkout gets new files: %v", paths(p))
	}
}

func TestTheDefaultChoiceKeepsWhatTheConfigSays(t *testing.T) {
	text := "[commit]\nlanguage = \"de\"\n\n[modules]\ngraph = false\n"
	root := world(t, map[string]string{"go.mod": goMod, configPath: text})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	if c.CommitLanguage != "de" || c.Modules[schema.Graph] {
		t.Fatalf("choice = %+v", c)
	}
	p := plan(t, f)
	if _, ok := changeOf(p, configPath); ok {
		t.Errorf("the default choice changes the configuration")
	}
	if slices.Contains(actions(p), "graph-build") {
		t.Errorf("graph off still builds the graph")
	}
}

func TestARegisteredRootGetsNoAreaAdd(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod})
	f := gather(t, root, "")
	f.Registered = true
	c := DefaultChoice(f, Answers{})
	if c.Parts["area"] {
		t.Errorf("area is chosen for a registered root")
	}
	// Chosen anyway, through an earlier answer: Build still holds back.
	c.Parts["area"] = true
	p, _ := Build(f, c, reader(root))
	if slices.Contains(actions(p), "area-add") || !hasNote(p, "the registry has an area") {
		t.Errorf("actions = %v, notes = %v", actions(p), p.Notes)
	}
	f.Registered = false
	f.Config = "[area]\nscope = \"project/demo\"\n"
	p, _ = Build(f, c, reader(root))
	if slices.Contains(actions(p), "area-add") || !hasNote(p, "declares [area]") {
		t.Errorf("actions = %v, notes = %v", actions(p), p.Notes)
	}
}

func TestAnswersOverrideTheDefaults(t *testing.T) {
	root := world(t, map[string]string{})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{Hosts: []string{"nobody"}, Parts: map[string]bool{"agents-md": false}})
	if c.Parts["agents-md"] || !slices.Equal(c.Hosts, f.Hosts) {
		t.Errorf("choice = %+v", c)
	}
}
