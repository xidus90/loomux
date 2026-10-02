package setup

import (
	"path/filepath"
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
		"workspace": false, "area": false, "merge-hook": false, "brain-skills": false, "model": false, "graph-build": false,
	}
	if len(Parts(f)) != len(want) {
		t.Errorf("%d parts, want %d", len(Parts(f)), len(want))
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

// The default scope comes from the directory name, and the interactive form
// refuses a scope with blanks; --yes takes the default without asking, so
// the default must be one the form would take.
func TestTheDefaultScopeHasNoBlanks(t *testing.T) {
	for dir, want := range map[string]string{
		"my project": "project/my-project",
		"a  b\tc":    "project/a-b-c",
		"plain":      "project/plain",
	} {
		c := DefaultChoice(Facts{Root: filepath.Join(t.TempDir(), dir)}, Answers{})
		if c.Scope != want {
			t.Errorf("%q: scope %q, want %q", dir, c.Scope, want)
		}
	}
	// A name of blanks alone, or a volume root, leaves nothing to take the
	// scope from; area add refuses an empty segment.
	for _, root := range []string{filepath.Join(t.TempDir(), "   "), filepath.VolumeName(t.TempDir()) + string(filepath.Separator), ""} {
		if c := DefaultChoice(Facts{Root: root}, Answers{}); c.Scope != "project/root" {
			t.Errorf("%q: scope %q, want project/root", root, c.Scope)
		}
	}
}

func TestTheWorkspaceRegistersTheProjectOnlyWithoutTheBrain(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	if !c.Parts["workspace"] {
		t.Fatal("workspace is not chosen for a fresh project")
	}
	for _, tc := range []struct {
		name         string
		brain, hooks bool
		want         []string // the registering actions planned
	}{
		{"brain on", true, true, []string{"area-add"}},
		{"brain off", false, true, []string{"workspace-add"}},
		{"brain and hooks off", false, false, nil},
	} {
		c.Modules[schema.Brain], c.Modules[schema.Hooks] = tc.brain, tc.hooks
		p, err := Build(f, c, reader(root))
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, id := range actions(p) {
			if id == "area-add" || id == "workspace-add" {
				got = append(got, id)
			}
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: registering actions %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestADeselectedWorkspaceIsNamedWithoutTheBrain(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	c.Parts["workspace"] = false
	for _, tc := range []struct {
		name         string
		brain, hooks bool
		want         bool
	}{
		{"brain off", false, true, true},
		{"brain on", true, true, false},
		{"hooks off", false, false, false},
	} {
		c.Modules[schema.Brain], c.Modules[schema.Hooks] = tc.brain, tc.hooks
		p, err := Build(f, c, reader(root))
		if err != nil {
			t.Fatal(err)
		}
		if got := hasNote(p, "workspace: skipped by choice; the write barrier opens no tree"); got != tc.want {
			t.Errorf("%s: note %v, want %v; notes = %v", tc.name, got, tc.want, p.Notes)
		}
	}
	// Where the tree is open already, there is nothing to warn about.
	c.Modules[schema.Brain], c.Modules[schema.Hooks] = false, true
	// The registered root gets its own note (TestARegisteredRootGetsNoWorkspaceAdd);
	// a declared area and a checkout get none.
	for _, tc := range []struct {
		name   string
		edit   func(*Facts)
		absent string
	}{
		{"registered root", func(f *Facts) { f.Registered = true }, "workspace: skipped by choice"},
		{"declared area", func(f *Facts) { f.Config = "[area]\nscope = \"project/demo\"\n" }, "workspace:"},
		{"checkout", func(f *Facts) { f.Checkout = true }, "workspace:"},
	} {
		g := f
		tc.edit(&g)
		p, err := Build(g, c, reader(root))
		if err != nil {
			t.Fatal(err)
		}
		if hasNote(p, tc.absent) {
			t.Errorf("%s: notes = %v", tc.name, p.Notes)
		}
	}
}

func TestARegisteredRootGetsNoWorkspaceAdd(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod})
	f := gather(t, root, "")
	f.Registered = true
	c := DefaultChoice(f, Answers{})
	if c.Parts["workspace"] {
		t.Errorf("workspace is chosen for a registered root")
	}
	// Off by default here, and still the note says why nothing registers.
	c.Modules[schema.Brain] = false
	if p, _ := Build(f, c, reader(root)); !hasNote(p, "workspace: skipped; the registry has an area") {
		t.Errorf("unchosen: notes = %v", p.Notes)
	}
	c.Modules[schema.Hooks] = false
	if p, _ := Build(f, c, reader(root)); hasNote(p, "workspace:") {
		t.Errorf("hooks off: notes = %v", p.Notes)
	}
	c.Modules[schema.Hooks] = true
	// Chosen anyway, through an earlier answer: Build still holds back.
	c.Parts["workspace"] = true
	c.Modules[schema.Brain] = false
	p, _ := Build(f, c, reader(root))
	if slices.Contains(actions(p), "workspace-add") || !hasNote(p, "workspace: skipped; the registry has an area") {
		t.Errorf("actions = %v, notes = %v", actions(p), p.Notes)
	}
	f.Registered = false
	f.Config = "[area]\nscope = \"project/demo\"\n"
	if DefaultChoice(f, Answers{}).Parts["workspace"] {
		t.Errorf("workspace is chosen over a declared area")
	}
	p, _ = Build(f, c, reader(root))
	if slices.Contains(actions(p), "workspace-add") || !hasNote(p, "workspace: skipped; .loomux/config.toml declares [area]") {
		t.Errorf("actions = %v, notes = %v", actions(p), p.Notes)
	}
}
