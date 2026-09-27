package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/xidus90/loomux/flows"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/flow/load"
	"github.com/xidus90/loomux/internal/flow/model"
	"github.com/xidus90/loomux/internal/flow/runs"
)

func TestUsageErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		code int
		want string
	}{
		{"no command", nil, flowExitUsage, "usage:"},
		{"an unknown command", []string{"fly"}, flowExitUsage, `loomux flow: unknown command "fly"`},
		{"resume without a run", []string{"resume"}, flowExitUsage, "loomux flow resume: the name is missing"},
		{"a flag where the run belongs", []string{"resume", "--answer", "yes"}, flowExitUsage, "loomux flow resume: the name is missing"},
		{"an argument too many", []string{"show", "0001", "0002"}, flowExitUsage, `loomux flow show: unexpected argument "0002"`},
		{"a run with an argument too many", []string{"run", "example", "extra"}, flowExitUsage, `loomux flow run: unexpected argument "extra"`},
		{"an option without a value", []string{"run", "example", "--option", "max_rounds"}, flowExitUsage, `want name=value, got "max_rounds"`},
		{"an option given twice", []string{"run", "example", "--option", "a=1", "--option", "a=2"}, flowExitUsage, "option a is given twice"},
		{"an unknown flag", []string{"replay", "0001", "--nope"}, flowExitUsage, "flag provided but not defined: -nope"},
		{"asking for help", []string{"list", "-h"}, flowExitOK, "-root"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newFlowHarness()
			code := h.runHere(c.args...)
			if code != c.code || !strings.Contains(h.stderr.String(), c.want) {
				t.Fatalf("exit %d, stderr %q, want %d and %q", code, h.stderr.String(), c.code, c.want)
			}
		})
	}
}

// `loomux flow` is in the command table, and runs on the real outside: the
// bundled catalog lists its example.
func TestFlowIsACommandOfLoomux(t *testing.T) {
	code, _, stderr := run("flow")
	if code != flowExitUsage || !strings.Contains(stderr, "loomux flow run [<flow>]") {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	code, stdout, stderr := run("flow", "list", "--root", t.TempDir())
	if code != flowExitOK || stdout != "example  bundled  ok\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

// The real outside refuses every provider until an adapter exists, keeps time
// with the wall clock, knows the three blocks the runtime ships and reads the
// catalog compiled into the binary.
func TestFlowProductionHasNoAdapterYet(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := flowProduction(&stdout, &stderr)
	if _, err := deps.Models("claude"); err == nil || err.Error() != "no adapter for provider claude yet" {
		t.Fatalf("err = %v", err)
	}
	if deps.Clock == nil || deps.Stdout != &stdout || deps.Stderr != &stderr {
		t.Fatalf("deps = %+v", deps)
	}
	kinds := make([]string, 0, len(deps.Blocks))
	for _, block := range deps.Blocks {
		kinds = append(kinds, block.Kind())
	}
	if !slices.Equal(kinds, []string{"agent", "gate", "exit"}) {
		t.Fatalf("kinds = %v", kinds)
	}
	if _, err := fs.Stat(deps.Bundled, "example/flow.toml"); err != nil {
		t.Fatalf("the catalog has no example: %v", err)
	}
}

func TestFlowRunRefusesAnAgentFlowWithoutAnAdapter(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := flowCLI([]string{"run", "example", "--root", root}, flowProduction(&stdout, &stderr))
	if code != flowExitFailed || stderr.String() != "no adapter for provider claude yet\n" {
		t.Fatalf("exit %d, stderr %q", code, stderr.String())
	}
	if files := flowRunFiles(t, root); len(files) != 0 {
		t.Fatalf("a refused run left %v behind", files)
	}
}

func TestFlowRunWithoutANameStartsTheDefault(t *testing.T) {
	root := t.TempDir()
	flowConfig(t, root, "[flow]\ndefault = \"example\"\n")
	h := newFlowHarness(flowDone())
	if code := h.run(root, "run", "--option", "max_rounds=3"); code != flowExitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "run 0001 (example, bundled): paused\nApprove the plan after 2 rounds?\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if marker := flowReadMarker(t, root, "0001"); marker.Flow != "example" || marker.Options["max_rounds"] != "3" {
		t.Fatalf("marker = %+v", marker)
	}
}

func TestFlowRunWithoutANameOrADefaultIsRefused(t *testing.T) {
	root := t.TempDir()
	h := newFlowHarness(flowDone())
	if code := h.run(root, "run"); code != flowExitFailed ||
		h.stderr.String() != "no flow named and [flow] default is unset; known flows: example\n" {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if files := flowRunFiles(t, root); len(files) != 0 {
		t.Fatalf("a refused run left %v behind", files)
	}
	h.deps.Bundled = fstest.MapFS{}
	if code := h.run(root, "run"); code != flowExitFailed || !strings.HasSuffix(h.stderr.String(), "known flows: none\n") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
}

func TestFlowRunNamesAMissingDefault(t *testing.T) {
	root := t.TempDir()
	flowConfig(t, root, "[flow]\ndefault = \"ghost\"\n")
	h := newFlowHarness(flowDone())
	if code := h.run(root, "run"); code != flowExitFailed ||
		h.stderr.String() != "[flow] default names \"ghost\", which is no flow here\n" {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
}

// Without --root a flow command finds the project the way `loomux config`
// does: the nearest .loomux/config.toml above the working directory.
func TestFlowFindsTheRootFromASubdirectory(t *testing.T) {
	root := t.TempDir()
	flowConfig(t, root, "")
	flowWrite(t, root, load.Dir+"/ask/flow.toml", askFlow)
	flowWrite(t, root, load.Dir+"/ask/questions/confirm.md", "Ship it?\n")
	below := filepath.Join(root, "internal")
	if err := os.MkdirAll(below, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(below)
	h := newFlowHarness()
	if code := h.runHere("run", "ask"); code != flowExitPaused || h.stdout.String() != "run 0001 (ask, project): paused\nShip it?\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, h.stdout.String(), h.stderr.String())
	}
	if files := flowRunFiles(t, root); !slices.Equal(files, []string{"0001.flow", "0001.jsonl"}) {
		t.Fatalf("files under the root = %v", files)
	}
	if code := h.runHere("list"); code != flowExitOK || !strings.HasPrefix(h.stdout.String(), "ask      project  ok\n") {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, h.stdout.String(), h.stderr.String())
	}
	if code := h.runHere("resume", "0001", "--answer", "yes"); code != flowExitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
}

// A project without .loomux/config.toml still has flows of its own: the
// working directory is its root.
func TestFlowWithoutAConfigUsesTheWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	flowWrite(t, root, load.Dir+"/mine/flow.toml", askFlow)
	flowWrite(t, root, load.Dir+"/mine/questions/confirm.md", "Ship it?\n")
	t.Chdir(root)
	h := newFlowHarness()
	if code := h.runHere("run", "mine"); code != flowExitPaused || h.stdout.String() != "run 0001 (mine, project): paused\nShip it?\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, h.stdout.String(), h.stderr.String())
	}
	if files := flowRunFiles(t, root); !slices.Equal(files, []string{"0001.flow", "0001.jsonl"}) {
		t.Fatalf("files = %v", files)
	}
}

// A working directory the process cannot name is no directory to fall back
// to; hosts.FindRoot answers that only when os.Getwd fails.
func TestFlowRootRefusesAWorkingDirectoryItCannotName(t *testing.T) {
	root := ""
	var stderr bytes.Buffer
	failing := func(string) (string, error) { return "", errors.New("getwd: the directory is gone") }
	if code, ok := flowRootOrRefuse("list", &root, failing, &stderr); ok || code != flowExitFailed ||
		stderr.String() != "loomux flow list: getwd: the directory is gone\n" {
		t.Fatalf("exit %d, ok %v, stderr %q", code, ok, stderr.String())
	}
}

func TestAPausedRunNamesItsOrigin(t *testing.T) {
	root := t.TempDir()
	flowConfig(t, root, "[flow]\noverrides = [\"example\"]\n")
	flowWrite(t, root, load.Dir+"/example/questions/approve.md", "Sign off after {{count}} rounds?\n")
	h := newFlowHarness(flowDone())
	if code := h.run(root, "run", "example"); code != flowExitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "run 0001 (example, bundled+overlay: questions/approve.md): paused\nSign off after 2 rounds?\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if marker := flowReadMarker(t, root, "0001"); marker.Origin != load.OriginOverlay || !slices.Equal(marker.Overlays, []string{"questions/approve.md"}) {
		t.Fatalf("marker = %+v", marker)
	}
}

func TestAPausedRunPrintsTheQuestionWithoutCarriageReturns(t *testing.T) {
	root := t.TempDir()
	flowConfig(t, root, "[flow]\noverrides = [\"example\"]\n")
	flowWrite(t, root, load.Dir+"/example/questions/approve.md", "Sign off\r\nafter {{count}} rounds?\r\n")
	h := newFlowHarness(flowDone())
	if code := h.run(root, "run", "example"); code != flowExitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	got := h.stdout.String()
	if want := "run 0001 (example, bundled+overlay: questions/approve.md): paused\nSign off\nafter 2 rounds?\n"; got != want || strings.Contains(got, "\r") {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

// A run started on the project's own flow.toml is not carried on by the
// catalog's: its open gate may belong to another graph.
func TestResumeRefusesAnotherFlowFile(t *testing.T) {
	root := t.TempDir()
	flowProjectExample(t, root)
	flowConfig(t, root, "[flow]\noverrides = [\"example\"]\n")
	h := newFlowHarness(flowDone())
	if code := h.run(root, "run", "example"); code != flowExitPaused ||
		!strings.HasPrefix(h.stdout.String(), "run 0001 (example, project (hides bundled)): paused\n") {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, h.stdout.String(), h.stderr.String())
	}
	before := flowReadBytes(t, runs.JournalPath(root, "0001"))
	flowConfig(t, root, "")
	code := h.run(root, "resume", "0001", "--answer", "yes")
	want := "run 0001 started on project (hides bundled) and example now resolves to bundled, another flow.toml; start a new run with loomux flow run example\n"
	if code != flowExitFailed || !strings.HasSuffix(h.stderr.String(), want) {
		t.Fatalf("exit %d, stderr %q, want it to end in %q", code, h.stderr.String(), want)
	}
	if after := flowReadBytes(t, runs.JournalPath(root, "0001")); !bytes.Equal(before, after) {
		t.Fatal("the refused resume wrote to the journal")
	}
}

// Another set of overlays is the same flow.toml with other texts: the run goes
// on, and the definition hashes name the nodes whose text changed.
func TestResumeWarnsWhenTheOverlaysChanged(t *testing.T) {
	root := t.TempDir()
	flowConfig(t, root, "[flow]\noverrides = [\"example\"]\n")
	flowWrite(t, root, load.Dir+"/example/questions/approve.md", "Approve the plan after {{count}} rounds?\n")
	h := newFlowHarness(flowDone())
	if code := h.run(root, "run", "example"); code != flowExitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	flowWrite(t, root, load.Dir+"/example/instructions/draft.md", "Write a draft plan. You have {{max_rounds}} rounds.\n")
	if code := h.run(root, "resume", "0001", "--answer", "yes"); code != flowExitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if want := "warning: run 0001 started with overlays questions/approve.md and now has instructions/draft.md, questions/approve.md\n"; h.stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", h.stderr.String(), want)
	}
	if got, want := h.stdout.String(), "run 0001 (example, bundled+overlay: instructions/draft.md, questions/approve.md): done\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

// A run whose flow now resolves to another flow.toml is refused before that
// flow.toml is loaded: its parameters or its findings would otherwise speak
// first, and the reason -- another graph -- would go unsaid. Resume and replay
// both refuse, and neither writes.
func TestARunIsNotCarriedOnByAnotherFlowFile(t *testing.T) {
	cases := []struct {
		name          string
		flow          string
		options       []string
		before, after func(t *testing.T, root string, h *flowHarness)
		want          string
	}{
		{name: "a parameter only the project's flow.toml declares", flow: "example",
			options: []string{"--option", "foo=1"},
			before: func(t *testing.T, root string, _ *flowHarness) {
				flowProjectExample(t, root)
				path := filepath.Join(root, filepath.FromSlash(load.Dir), "example", "flow.toml")
				body := string(flowReadBytes(t, path))
				declared := strings.Replace(body, "[params]\n", "[params]\nfoo = { type = \"int\", default = 0 }\n", 1)
				if declared == body {
					t.Fatal("the example declares no [params] to add to")
				}
				flowOverwrite(t, path, declared)
				flowConfig(t, root, "[flow]\noverrides = [\"example\"]\n")
			},
			after: func(t *testing.T, root string, _ *flowHarness) { flowConfig(t, root, "") },
			want:  "run 0001 started on project (hides bundled) and example now resolves to bundled, another flow.toml; start a new run with loomux flow run example\n"},
		{name: "a catalog flow.toml that does not load", flow: "mine",
			before: func(t *testing.T, root string, h *flowHarness) {
				h.deps.Bundled = fstest.MapFS{"mine/flow.toml": {Data: []byte("schema_version = 7\n")}}
				flowAsk(t, root, "mine")
				flowConfig(t, root, "[flow]\noverrides = [\"mine\"]\n")
			},
			after: func(t *testing.T, root string, _ *flowHarness) { flowConfig(t, root, "") },
			want:  "run 0001 started on project (hides bundled) and mine now resolves to bundled, another flow.toml; start a new run with loomux flow run mine\n"},
		{name: "the project's flow.toml over a run of the catalog's", flow: "example",
			after: func(t *testing.T, root string, _ *flowHarness) {
				flowProjectExample(t, root)
				flowConfig(t, root, "[flow]\noverrides = [\"example\"]\n")
			},
			want: "run 0001 started on bundled and example now resolves to project (hides bundled), another flow.toml; start a new run with loomux flow run example\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			h := newFlowHarness(flowDone())
			if c.before != nil {
				c.before(t, root, h)
			}
			if code := h.run(root, append([]string{"run", c.flow}, c.options...)...); code != flowExitPaused {
				t.Fatalf("the run did not pause: exit %d, stderr %q", code, h.stderr.String())
			}
			before := flowReadBytes(t, runs.JournalPath(root, "0001"))
			c.after(t, root, h)
			for _, args := range [][]string{{"resume", "0001", "--answer", "yes"}, {"replay", "0001"}} {
				if code := h.run(root, args...); code != flowExitFailed || !strings.HasSuffix(h.stderr.String(), c.want) {
					t.Fatalf("%s: exit %d, stderr %q, want it to end in %q", args[0], code, h.stderr.String(), c.want)
				}
			}
			if after := flowReadBytes(t, runs.JournalPath(root, "0001")); !bytes.Equal(before, after) {
				t.Fatal("a refused run wrote to its journal")
			}
		})
	}
}

// Both project origins read the project's own flow.toml, whether or not the
// catalog ships a flow of the name: a run carries on across them.
func TestARunCarriesOnAcrossTheProjectsOwnOrigins(t *testing.T) {
	catalog := fstest.MapFS{"mine/flow.toml": {Data: []byte(askFlow)}}
	cases := []struct {
		name          string
		first, second fs.FS
		want          string
	}{
		{"project, then project (hides bundled)", fstest.MapFS{}, catalog, "run 0001 (mine, project (hides bundled)): done\n"},
		{"project (hides bundled), then project", catalog, fstest.MapFS{}, "run 0001 (mine, project): done\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			flowAsk(t, root, "mine")
			flowConfig(t, root, "[flow]\noverrides = [\"mine\"]\n")
			h := newFlowHarness()
			h.deps.Bundled = c.first
			if code := h.run(root, "run", "mine"); code != flowExitPaused {
				t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
			}
			h.deps.Bundled = c.second
			if code := h.run(root, "resume", "0001", "--answer", "yes"); code != flowExitOK || h.stdout.String() != c.want {
				t.Fatalf("exit %d, stdout %q, stderr %q", code, h.stdout.String(), h.stderr.String())
			}
		})
	}
}

func TestFlowShowPrintsAFlow(t *testing.T) {
	root := t.TempDir()
	h := newFlowHarness()
	if code := h.run(root, "show", "example"); code != flowExitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	want := `example (bundled)
nodes:
  draft    agent  writer  claude:cli-default  role writer (node), the CLI's own default
  approve  gate
  stop     exit
edges:
  draft -> approve [verdict == "done"]
  draft -> draft
  approve -> END [answer == "yes"]
  approve -> stop
  stop -> END
`
	if got := h.stdout.String(); got != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", got, want)
	}
}

// A role the flow names, bound under [agent.roles], and an edge taken on an
// error, as show spells them.
func TestFlowShowNamesABindingAndAnErrorEdge(t *testing.T) {
	root := t.TempDir()
	flowWrite(t, root, load.Dir+"/guarded/flow.toml", guardedFlow)
	flowWrite(t, root, load.Dir+"/guarded/instructions/judge.md", "Judge the plan.\n")
	flowConfig(t, root, "[agent.models.w]\nprovider = \"agy\"\nmodel = \"m1\"\n\n[agent.roles]\ncritic = \"w\"\n")
	h := newFlowHarness()
	if code := h.run(root, "show", "guarded"); code != flowExitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	want := `guarded (project)
nodes:
  judge  agent  critic  agy:m1  role critic (flow), bound to w
  stop   exit
edges:
  judge -> END
  judge -> stop [on error]
  stop -> END
`
	if got := h.stdout.String(); got != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", got, want)
	}
}

func TestFlowListMarksTheDefaultAndWarns(t *testing.T) {
	root := t.TempDir()
	flowConfig(t, root, "[flow]\ndefault = \"example\"\noverrides = [\"ghost\"]\n")
	flowWrite(t, root, load.Dir+"/example/questions/approve.md", "Sign off?\n")
	h := newFlowHarness()
	if code := h.run(root, "list"); code != flowExitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got := h.stdout.String(); got != "example  bundled  ok  (default)\n" {
		t.Fatalf("stdout = %q", got)
	}
	for _, want := range []string{
		"warning: [flow] overrides names \"ghost\", which no bundled flow has\n",
		"warning: .loomux/flows/example is ignored: [flow] overrides does not name it\n",
	} {
		if !strings.Contains(h.stderr.String(), want) {
			t.Fatalf("stderr = %q, want it to hold %q", h.stderr.String(), want)
		}
	}
}

// A default that names no flow fails list as it fails run: the list is still
// printed, the other [flow] findings stay warnings.
func TestFlowListFailsOnADefaultThatNamesNoFlow(t *testing.T) {
	root := t.TempDir()
	flowConfig(t, root, "[flow]\ndefault = \"ghost\"\noverrides = [\"phantom\"]\n")
	h := newFlowHarness()
	if code := h.run(root, "list"); code != flowExitFailed || h.stdout.String() != "example  bundled  ok\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, h.stdout.String(), h.stderr.String())
	}
	want := "warning: [flow] overrides names \"phantom\", which no bundled flow has\n" +
		"[flow] default names \"ghost\", which is no flow here\n"
	if got := h.stderr.String(); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestFlowRunReadsTheAgentTable(t *testing.T) {
	root := t.TempDir()
	flowConfig(t, root, "[agent.models.w]\nprovider = \"agy\"\n\n[agent.roles]\nwriter = \"w\"\n")
	h := newFlowHarness(flowDone())
	var asked []string
	h.deps.Models = func(provider string) (model.Model, error) {
		asked = append(asked, provider)
		return h.fake, nil
	}
	if code := h.run(root, "run", "example"); code != flowExitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if seen := h.fake.Seen(); !slices.Equal(asked, []string{"agy"}) || len(seen) != 1 || seen[0].Provider != "agy" || seen[0].Model != "" {
		t.Fatalf("asked %v, requests %+v", asked, seen)
	}
}

// Every command that reads a flow reads both tables first, and stops at a
// table its reader refuses.
func TestAConfigTheReadersRefuseStopsEveryFlowCommand(t *testing.T) {
	configs := []struct{ body, want string }{
		{"[agent]\ndefault = \"x\"\n", `[agent] default names "x", which is not under [agent.models]; known: none`},
		{"[flow]\ndefault = 3\n", "[flow] default must be a flow name"},
	}
	commands := [][]string{
		{"list"},
		{"run", "example"},
		{"run"},
		{"show", "example"},
		{"resume", "0001", "--answer", "yes"},
		{"replay", "0001"},
	}
	for _, config := range configs {
		for _, args := range commands {
			t.Run(config.body+strings.Join(args, " "), func(t *testing.T) {
				h := newFlowHarness(flowDone())
				root := flowPausedExample(t, h)
				flowConfig(t, root, config.body)
				if code := h.run(root, args...); code != flowExitFailed || !strings.Contains(h.stderr.String(), config.want) {
					t.Fatalf("exit %d, stderr %q, want %q", code, h.stderr.String(), config.want)
				}
			})
		}
	}
}

// guardedFlow names its role for the whole flow and ends at an exit when its
// one agent fails.
const guardedFlow = `schema_version = 1

[flow]
start = "judge"
role  = "critic"

[state]
verdict = { type = "string", default = "" }

[[node]]
name        = "judge"
kind        = "agent"
instruction = "instructions/judge.md"
reply       = { verdict = "string" }

[[node]]
name    = "stop"
kind    = "exit"
code    = 4
message = "the judge failed"

[[edge]]
from = "judge"
to   = "END"

[[edge]]
from     = "judge"
to       = "stop"
on_error = true

[[edge]]
from = "stop"
to   = "END"
`

// The session start speaks of flows in the words the flow commands use: the
// origin flow run prints, and the warnings load.Find gives for an ignored
// folder and for a flows folder that is no folder. The hooks may not link the
// loader, so they keep a copy of its folder and its words; this test holds the
// copies to the originals.
func TestTheSessionStartSpeaksOfFlowsAsTheFlowCommandsDo(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())

	overlaid := project(t)
	flowConfig(t, overlaid, "[flow]\noverrides = [\"example\"]\n")
	flowWrite(t, overlaid, load.Dir+"/example/questions/approve.md", "Really?\n")
	h := newFlowHarness(flowDone())
	if code := h.run(overlaid, "run", "example"); code != flowExitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	printed, _, _ := strings.Cut(h.stdout.String(), "\n")
	announced, found := strings.CutSuffix(printed, ": paused")
	if !found {
		t.Fatalf("flow run printed %q", h.stdout.String())
	}
	if got := sessionStartContext(t, overlaid); !strings.Contains(got, announced+" is waiting at approve: Really?\n") {
		t.Errorf("the session start announces\n%s\nand flow run printed %q", got, printed)
	}

	ignored := project(t)
	flowWrite(t, ignored, load.Dir+"/example/questions/approve.md", "Really?\n")
	notAFolder := project(t)
	flowWrite(t, notAFolder, load.Dir, "not a folder")
	for _, root := range []string{ignored, notAFolder} {
		found, err := load.Find(root, flows.FS(), config.FlowSettings{}, "example")
		if err != nil || len(found.Warnings) != 1 {
			t.Fatalf("found %+v, err %v", found, err)
		}
		if got := sessionStartContext(t, root); !slices.Contains(strings.Split(got, "\n"), found.Warnings[0]) {
			t.Errorf("the session start says\n%s\nand load.Find warns %q", got, found.Warnings[0])
		}
	}
}

// sessionStartContext is what the session start of root tells Claude Code.
func sessionStartContext(t *testing.T, root string) string {
	t.Helper()
	code, out, errOut := runWith(`{"session_id":"s1"}`, "hook", "session-start", "--host", "claude", "--root", root)
	var answer struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(out), &answer); code != 0 || err != nil {
		t.Fatalf("code %d, out %q, err %q: %v", code, out, errOut, err)
	}
	return answer.HookSpecificOutput.AdditionalContext
}
