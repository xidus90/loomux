package verify

import (
	"slices"
	"strings"
	"testing"
)

const withRules = "[rules]\n"

func TestRulesLaneNeedsARulesTable(t *testing.T) {
	kinds := Request{Kinds: []string{"rules"}}
	jobs, err := Plan(effFor(t, "", goOnly), kinds, env(t.TempDir()))
	if err != nil || len(jobs) != 0 {
		t.Fatalf("no [rules], yet planned %s: %v", names(jobs), err)
	}
	jobs, err = Plan(effFor(t, withRules, goOnly), kinds, env(t.TempDir()))
	if err != nil || names(jobs) != "rules/go" || jobs[0].Area != "." {
		t.Fatalf("%v %+v", err, jobs)
	}
	if got := jobs[0].Argvs; len(got) != 1 || !slices.Equal(got[0], []string{`C:\bin\loomux.exe`, "check", "rules", "--cached"}) {
		t.Fatalf("argv = %v", got)
	}
}

// The rules belong to the root: one job in a stack in two areas, one job
// among stacks, and `rules = false` under any of them switches the kind off.
func TestRulesCarrierIsOne(t *testing.T) {
	jobs, err := Plan(effFor(t, withRules, goTwoAreas), Request{Kinds: []string{"rules"}}, env(t.TempDir()))
	if err != nil || names(jobs) != "rules/go" {
		t.Fatalf("%v %s", err, names(jobs))
	}
	both := factsOf("gdscript", "go")
	jobs, err = Plan(effFor(t, withRules, both), Request{Kinds: []string{"rules"}}, env(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	wantRulesJobs(t, jobs,
		[3]string{"rules/gdscript", "not-applicable", "rules covered by rules/go"},
		[3]string{"rules/go", "", ""})
	if len(jobs[1].Argvs) != 1 {
		t.Fatalf("the carrier runs: %+v", jobs[1])
	}
	jobs, err = Plan(effFor(t, withRules+"[verify.go]\nrules = false\n", both), Request{Kinds: []string{"rules"}}, env(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	wantRulesJobs(t, jobs,
		[3]string{"rules/gdscript", "not-applicable", "rules switched off under [verify.go]"},
		[3]string{"rules/go", "not-applicable", "rules switched off under [verify.go]"})
}

func TestRulesJobCarriesTheGraphEnvironment(t *testing.T) {
	e := env(t.TempDir())
	e.GraphEnv = func(string) []string { return []string{"GIT_INDEX_FILE=copy"} }
	jobs, err := Plan(effFor(t, withRules, goOnly), Request{Kinds: []string{"rules"}}, e)
	if err != nil || !slices.Contains(jobs[0].Env, "GIT_INDEX_FILE=copy") {
		t.Fatalf("%v %+v", err, jobs)
	}
}

func TestRulesOnEdit(t *testing.T) {
	edit := func(src, file string, facts ...string) string {
		t.Helper()
		jobs, err := Plan(effFor(t, src, factsOf(facts...)), Request{Kinds: []string{"rules"}, Scope: ScopeEdit, File: file}, env(t.TempDir()))
		if err != nil {
			t.Fatal(err)
		}
		for _, j := range jobs {
			if j.Area != "." || j.Dir == "" || len(j.Argvs) != 1 || slices.Contains(j.Argvs[0], "--cached") {
				t.Fatalf("%+v", j)
			}
		}
		return names(jobs)
	}
	for _, c := range []struct {
		name, src, file string
		stacks          []string
		want            string
	}{
		{"a gd source", withRules, "a/b.gd", []string{"gdscript"}, "rules/gdscript"},
		{"a source of the other stack", withRules, "a.go", []string{"gdscript", "go"}, "rules/go"},
		{"a source in capitals is none", withRules, "a.GO", []string{"go"}, ""},
		{"the config", withRules, ".loomux/config.toml", []string{"go"}, "rules/go"},
		{"prose", withRules, "README.md", []string{"go"}, ""},
		{"another toml", withRules, "x/config.toml", []string{"go"}, ""},
		{"no [rules]", "", "a.go", []string{"go"}, ""},
		{"switched off", withRules + "[verify.go]\nrules = false\n", "a.go", []string{"go", "gdscript"}, ""},
		{"no stack with a rules command", withRules, "a.go", []string{"rust"}, ""},
		{"no stack", withRules, ".loomux/config.toml", nil, ""},
	} {
		if got := edit(c.src, c.file, c.stacks...); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestEditsRules(t *testing.T) {
	for file, want := range map[string]bool{
		"a.go": true, "x/y.PY": false, "A.GD": false, "z.gd": true, ".loomux/config.toml": true,
		"a.md": false, "go.mod": false, "sub/.loomux/config.toml": false, "a.toml": false,
	} {
		if EditsRules(file) != want {
			t.Errorf("%s: %v", file, !want)
		}
	}
}

func TestDefaultProfilesCarryRules(t *testing.T) {
	cfg, err := parse(t, "")
	if err != nil {
		t.Fatal(err)
	}
	for profile, want := range map[string]string{
		"edit": "lint,types", "precommit": "lint,types,test,coverage,graph", "stop": "lint,types,test,coverage,graph",
	} {
		got, _ := ExpandProfile(cfg, profile)
		if strings.Join(got, ",") != want {
			t.Errorf("%s without [rules]: %v", profile, got)
		}
		if raw := strings.Join(cfg.Profiles[profile], ","); raw != want+",rules" {
			t.Errorf("%s defaults: %v", profile, raw)
		}
	}
	cfg, _ = parse(t, withRules)
	got, _ := ExpandProfile(cfg, "edit")
	all, _ := ExpandProfile(cfg, "all")
	if strings.Join(got, ",") != "lint,types,rules" || !slices.Contains(all, "rules") {
		t.Errorf("edit = %v, all = %v", got, all)
	}
	// Profiles of the project's own that leave rules out plan none.
	cfg, _ = parse(t, withRules+"[verify.profiles]\nedit = [\"lint\"]\n")
	if got, _ := ExpandProfile(cfg, "edit"); strings.Join(got, ",") != "lint" {
		t.Errorf("own profile = %v", got)
	}
}

func TestGDScriptLintFormats(t *testing.T) {
	eff := effFor(t, "", gdFacts())
	lint := eff.Stacks["gdscript"]["lint"].Lane
	if !slices.Contains(lint.Commands, "uvx --from {tool:gdtoolkit} gdformat --check .") ||
		!slices.Contains(lint.OnFile, "uvx --from {tool:gdtoolkit} gdformat --check {file}") {
		t.Fatalf("%+v", lint)
	}
}

// wantRulesJobs checks name, state and note of each job; only the carrier
// (an empty state) has something to run.
func wantRulesJobs(t *testing.T, jobs []Job, want ...[3]string) {
	t.Helper()
	if len(jobs) != len(want) {
		t.Fatalf("jobs %s, want %d", names(jobs), len(want))
	}
	for i, w := range want {
		j := jobs[i]
		if j.Name != w[0] || string(j.Pre) != w[1] || j.Note != w[2] || (w[1] == "") != (len(j.Argvs) == 1) {
			t.Errorf("job %d = %s %q %q %v, want %q", i, j.Name, j.Pre, j.Note, j.Argvs, w)
		}
	}
}

// Each root kind has its own switch: turning one off under a stack leaves the
// other planned.
func TestRootKindsSwitchOffSeparately(t *testing.T) {
	e := env(t.TempDir())
	e.GraphReady = func(string) (bool, string) { return true, "" }
	req := Request{Kinds: []string{"graph", "rules"}, Scope: ScopeCheck}
	jobs, err := Plan(effFor(t, withRules+"[verify.go]\ngraph = false\n", goOnly), req, e)
	if err != nil {
		t.Fatal(err)
	}
	// A kind switched off under the only stack has no command left to run.
	if len(jobs) != 2 || jobs[0].Pre != StateNotApplicable || jobs[1].Name != "rules/go" || jobs[1].Pre != "" || len(jobs[1].Argvs) != 1 {
		t.Fatalf("graph = false: %+v", jobs)
	}
	jobs, err = Plan(effFor(t, withRules+"[verify.go]\nrules = false\n", goOnly), req, e)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 || jobs[0].Name != "graph/go" || jobs[0].Pre != "" || len(jobs[0].Argvs) == 0 || jobs[1].Pre != StateNotApplicable {
		t.Fatalf("rules = false: %+v", jobs)
	}
}

// A caller without an index to read says so through RulesReady, and the rules
// job sits out with the reason.
func TestRulesJobSitsOutWhenNoIndexIsReady(t *testing.T) {
	e := env(t.TempDir())
	e.RulesReady = func(string) (bool, string) { return false, "no HEAD yet" }
	jobs, err := Plan(effFor(t, withRules, goOnly), Request{Kinds: []string{"rules"}, Scope: ScopeCheck}, e)
	if err != nil {
		t.Fatal(err)
	}
	wantRulesJobs(t, jobs, [3]string{"rules/go", "not-applicable", "no HEAD yet"})
	e.RulesReady = func(string) (bool, string) { return true, "" }
	jobs, err = Plan(effFor(t, withRules, goOnly), Request{Kinds: []string{"rules"}, Scope: ScopeCheck}, e)
	if err != nil {
		t.Fatal(err)
	}
	wantRulesJobs(t, jobs, [3]string{"rules/go", "", ""})
}
