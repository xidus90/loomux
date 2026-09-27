package load

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestEachStageHasAFlowThatFailsInIt(t *testing.T) {
	cases := []struct {
		dir  string
		want []string
	}{
		{dir: "stage1-version", want: []string{
			"schema_version 2 is unknown; loomux knows version 1",
		}},
		{dir: "stage1-names", want: []string{
			`"END" is reserved and cannot be a node name`,
			`field "max-rounds" is not a name; a name is [A-Za-z_][A-Za-z0-9_]*`,
			`"count" is declared under [params] and under [state]`,
			`field "count" default: cannot read x as int`,
		}},
		{dir: "stage1-edges", want: []string{
			"the error edge draft→fix cannot carry a when; an error edge is unconditional",
			"edge draft→approve has an empty when",
			"edge 3 has no to",
		}},
		{dir: "stage2-kinds", want: []string{
			`node "run" has kind "command"; command nodes are planned but not built`,
			`node "w" has kind "widget"; known kinds: agent, exit, gate`,
			`node "stop": code 3 is the runtime's own; a block may not choose 0, 1 or 3`,
		}},
		{dir: "stage4-names", want: []string{
			`node "draft" instruction names "verdit"; known fields: verdict`,
			`edge draft→approve: reads "verdit"; known fields: verdict`,
			`node "draft": max_visits "rounds + 1" names no parameter; known parameters: max_rounds`,
		}},
		{dir: "stage5-writes", want: []string{
			`node "draft" writes "verdict" as int, but [state] declares it as string`,
			`node "approve" writes "choice", which [state] does not declare`,
		}},
		{dir: "stage6-graph", want: []string{
			`start node "nowhere" does not exist`,
		}},
		{dir: "stage6-cycle", want: []string{
			`node "a" sits on a cycle but allows one visit; raise its max_visits`,
			`node "b" sits on a cycle but allows one visit; raise its max_visits`,
		}},
		{dir: "stage6-island", want: []string{
			"unreachable node(s): island",
			`node "stop" has no outgoing edge`,
		}},
	}
	for _, c := range cases {
		t.Run(c.dir, func(t *testing.T) {
			_, err := Load(testFlow(t, c.dir), catalog(t))
			if err == nil {
				t.Fatal("want findings")
			}
			lines := strings.Split(err.Error(), "\n")
			if len(lines) != len(c.want) {
				t.Fatalf("got %d findings, want %d:\n%s", len(lines), len(c.want), err)
			}
			prefix := c.dir + "/flow.toml: "
			for i, want := range c.want {
				if !strings.HasSuffix(lines[i], want) || !strings.HasPrefix(lines[i], prefix) {
					t.Fatalf("finding %d is %q, want %q behind %q", i, lines[i], want, prefix)
				}
			}
		})
	}
}

// Stage 3 is the one stage whose finding cannot be matched from the end: it
// carries the file system's own words for a missing file. The node and the
// path it looked for are what this stage owes the reader, and those stand at
// the front.
func TestAMissingInstructionFileNamesTheNodeAndThePath(t *testing.T) {
	_, err := Load(testFlow(t, "stage3-texts"), catalog(t))
	if err == nil {
		t.Fatal("want findings")
	}
	const want = `stage3-texts/flow.toml: node "draft": reading instruction instructions/draft.md: `
	if lines := strings.Split(err.Error(), "\n"); len(lines) != 1 || !strings.HasPrefix(lines[0], want) {
		t.Fatalf("err = %v", err)
	}
}

// The point of collecting: a file with three mistakes in one stage is read once.
func TestAStageReportsEveryFindingItHas(t *testing.T) {
	_, err := Load(testFlow(t, "stage1-edges"), catalog(t))
	if err == nil || len(strings.Split(err.Error(), "\n")) != 3 {
		t.Fatalf("err = %v", err)
	}
}

// And the point of stopping: stage 4 reads names stage 1 has not cleared yet,
// so a file that fails in stage 1 is not also reported against later stages.
func TestLoadStopsAfterTheFirstStageWithFindings(t *testing.T) {
	_, err := Load(testFlow(t, "stage1-names"), catalog(t))
	if err == nil {
		t.Fatal("want findings")
	}
	if strings.Contains(err.Error(), "max_visits") {
		t.Fatalf("a later stage ran anyway:\n%s", err)
	}
}

// loadSource loads one flow file that lives nowhere on disk. The fixtures
// under testdata show a reader what a stage is for; the refusals below are
// shapes nobody writes on purpose, and a folder each would bury the ones that
// matter.
func loadSource(t *testing.T, source string) error {
	t.Helper()
	_, err := Load(inline(source, nil), catalog(t))
	return err
}

func findings(t *testing.T, err error, want []string) {
	t.Helper()
	if err == nil {
		t.Fatal("want findings")
	}
	lines := strings.Split(err.Error(), "\n")
	if len(lines) != len(want) {
		t.Fatalf("got %d findings, want %d:\n%s", len(lines), len(want), err)
	}
	for i, one := range want {
		if lines[i] != "f/flow.toml: "+one {
			t.Fatalf("finding %d is %q, want %q", i, lines[i], one)
		}
	}
}

func TestStageOneRefusesEveryShapeItCannotRead(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   []string
	}{
		{name: "no toml at all", source: "this is not toml\n", want: []string{
			"toml: line 1: expected '.' or '=', but got 'i' instead",
		}},
		{name: "nothing declared", source: "", want: []string{
			"schema_version is missing; loomux knows version 1",
			"[flow] is missing",
		}},
		{name: "a version that is no number", source: `
schema_version = "one"

[flow]
start = "a"
`, want: []string{
			`schema_version one is unknown; loomux knows version 1`,
		}},
		{name: "a head without its keys", source: `
schema_version = 1

[flow]
`, want: []string{
			"[flow] has no start",
		}},
		{name: "nodes without a name, a kind or a first claim", source: `
schema_version = 1

[flow]
start = "a"

[[node]]
kind = "exit"

[[node]]
name = "a"

[[node]]
name = "a"
kind = "exit"
`, want: []string{
			"node 1 has no name",
			`node "a" has no kind`,
			`node "a" is declared twice`,
		}},
		{name: "declarations that are no tables", source: `
schema_version = 1

[flow]
start = "a"

[params]
rounds = 3

[state]
verdict = { type = "verdict", default = "" }
count   = { default = 0 }
`, want: []string{
			`parameter "rounds" is not a table; write rounds = { type = ..., default = ... }`,
			`field "count" has type ""; types are bool, int, list[string], string`,
			`field "verdict" has type "verdict"; types are bool, int, list[string], string`,
		}},
		// A run's marker keeps these names for itself, so an option of that
		// name could never be written: refused here, not at the first run.
		{name: "parameters named like the marker's own keys", source: `
schema_version = 1

[flow]
start = "a"

[params]
origin          = { type = "string", default = "" }
overlays        = { type = "string", default = "" }
baseline        = { type = "string", default = "" }
baseline_commit = { type = "string", default = "" }
loomux_version  = { type = "string", default = "" }
rounds          = { type = "int", default = 1 }

[state]
origin_note = { type = "string", default = "" }
`, want: []string{
			`parameter "baseline" is a name the run's marker keeps for itself; name it otherwise`,
			`parameter "baseline_commit" is a name the run's marker keeps for itself; name it otherwise`,
			`parameter "loomux_version" is a name the run's marker keeps for itself; name it otherwise`,
			`parameter "origin" is a name the run's marker keeps for itself; name it otherwise`,
			`parameter "overlays" is a name the run's marker keeps for itself; name it otherwise`,
		}},
		{name: "a field without a default", source: `
schema_version = 1

[flow]
start = "a"

[state]
count = { type = "int" }
`, want: []string{
			`field "count" has no default; every field and parameter needs one`,
		}},
		{name: "an edge without a from", source: `
schema_version = 1

[flow]
start = "a"

[[node]]
name    = "a"
kind    = "exit"
message = "done"

[[edge]]
to = "END"
`, want: []string{
			"edge 1 has no from",
		}},
		// The decoder is silent about a wrongly typed key, and at on_error and
		// when that silence changes where a run goes: an error edge would
		// become an ordinary one, a condition would vanish.
		{name: "an error edge whose on_error is text", source: `
schema_version = 1

[flow]
start = "a"

[[node]]
name    = "a"
kind    = "exit"
message = "done"

[[edge]]
from     = "a"
to       = "END"
on_error = "true"
`, want: []string{
			`edge 1: on_error is "true", not true or false`,
		}},
		{name: "a when that is no text", source: `
schema_version = 1

[flow]
start = "a"

[[node]]
name    = "a"
kind    = "exit"
message = "done"

[[edge]]
from = "a"
to   = "END"
when = 3
`, want: []string{
			"edge 1: when is 3, not text",
		}},
		{name: "a role that is no text", source: `
schema_version = 1

[flow]
start = "a"
role  = 3

[[node]]
name    = "a"
kind    = "exit"
role    = 4
message = "done"
`, want: []string{
			"[flow]: role is 3, not text",
			"node 1: role is 4, not text",
		}},
		{name: "a role that is no name", source: `
schema_version = 1

[flow]
start = "a"
role  = "a-b"
`, want: []string{
			`[flow]: role "a-b" is not a name; a name is [A-Za-z_][A-Za-z0-9_]*`,
		}},
		{name: "a head that is no table", source: `
schema_version = 1
flow = 3
`, want: []string{
			"[flow] is 3, not a table",
		}},
		{name: "tables that are no tables", source: `
schema_version = 1
params = 3
node   = "x"

[flow]
start = "a"
`, want: []string{
			`node is "x", not [[node]] entries`,
			"[params] is 3, not a table",
		}},
		{name: "a number a journal cannot write", source: `
schema_version = 1

[flow]
start = "a"

[[node]]
name    = "a"
kind    = "exit"
message = "done"
score   = nan
sizes   = [1.0, inf]

[[node.run]]
weight = nan
`, want: []string{
			`node "a" key "run[0].weight" is NaN; a flow file carries only finite numbers`,
			`node "a" key "score" is NaN; a flow file carries only finite numbers`,
			`node "a" key "sizes[1]" is +Inf; a flow file carries only finite numbers`,
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			findings(t, loadSource(t, c.source), c.want)
		})
	}
}

// What the fixtures do not reach: a flow that declares no field at all.
func TestTheLaterStagesRefuseWhatNoFixtureShows(t *testing.T) {
	const placeholder = `
schema_version = 1

[flow]
start = "a"

[[node]]
name    = "a"
kind    = "exit"
message = "stopped after {{rounds}} rounds"

[[edge]]
from = "a"
to   = "END"
`
	findings(t, loadSource(t, placeholder), []string{
		`node "a" message names "rounds"; known fields: none`,
	})
}

// A flow that names no role loads in any project: which model plays a role is
// the project's [agent.roles] and no load stage asks for it.
func TestAFlowWithoutRolesLoads(t *testing.T) {
	good := mustRead(t, filepath.Join("testdata", "minimal", "flow.toml"))
	if err := loadSource(t, good); err != nil {
		t.Fatal(err)
	}
}

// A text whose braces are no placeholder is stage 4's other finding: tmpl
// refuses it, and the node and the key it came from are what the reader needs.
func TestATextThatIsNoTemplateNamesItsNodeAndKey(t *testing.T) {
	const source = `
schema_version = 1

[flow]
start = "a"

[[node]]
name    = "a"
kind    = "exit"
message = "{{ rounds }}"

[[edge]]
from = "a"
to   = "END"
`
	err := loadSource(t, source)
	if err == nil {
		t.Fatal("want findings")
	}
	const want = `f/flow.toml: node "a" message: {{ rounds }} is not a placeholder`
	if !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("err = %v", err)
	}
}

// Rule 2 of stage 6, which the fixtures skip because rule 1 already stops
// them: an edge that names a node the file never declared.
func TestStageSixRefusesAnEdgeToANodeThatIsNotThere(t *testing.T) {
	const source = `
schema_version = 1

[flow]
start = "a"

[[node]]
name    = "a"
kind    = "exit"
message = "done"

[[edge]]
from = "a"
to   = "nowhere"

[[edge]]
from = "ghost"
to   = "END"
`
	findings(t, loadSource(t, source), []string{
		`edge from "a" to unknown node "nowhere"`,
		`edge from unknown node "ghost"`,
	})
}
