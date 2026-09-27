package cli

import (
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow/blocks"
	"github.com/xidus90/loomux/internal/flow/load"
	"github.com/xidus90/loomux/internal/flow/runs"
)

func TestRunPausesAtTheGateAndRecordsTheRun(t *testing.T) {
	root := t.TempDir()
	h := newFlowHarness(flowDone())
	if code := h.run(root, "run", "example"); code != flowExitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "run 0001 (example, bundled): paused\nApprove the plan after 2 rounds?\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := flowLines(t, runs.JournalPath(root, "0001")); len(got) != 2 {
		t.Fatalf("journal = %q", got)
	}
	marker := flowReadMarker(t, root, "0001")
	if marker.Flow != "example" || marker.Origin != load.OriginBundled || marker.Overlays != nil ||
		marker.Version != bareVersion() || len(marker.Options) != 0 {
		t.Fatalf("marker = %+v", marker)
	}
	if marker.Baseline != nil {
		t.Fatalf("a temporary directory is no repository, and a run there has no baseline: %+v", marker.Baseline)
	}
}

// An option reaches the run in its declared type and the marker as text.
func TestRunTakesAnOptionIntoTheRunAndTheMarker(t *testing.T) {
	root := t.TempDir()
	h := newFlowHarness(flowDone())
	if code := h.run(root, "run", "example", "--option", "max_rounds=3"); code != flowExitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	seen := h.fake.Seen()
	if len(seen) != 1 || !strings.Contains(seen[0].Prompt, "You have 3 rounds.") {
		t.Fatalf("requests = %+v", seen)
	}
	if marker := flowReadMarker(t, root, "0001"); marker.Options["max_rounds"] != "3" {
		t.Fatalf("marker = %+v", marker)
	}
}

// Inside a repository a run takes the commit it starts from, and the marker
// carries it.
func TestRunInARepositoryRecordsItsBaseline(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	flowWrite(t, root, load.Dir+"/measure/flow.toml", measureFlow)
	h := newFlowHarness()
	h.deps.Blocks = append(h.deps.Blocks, measureBlock{})
	if code := h.run(root, "run", "measure"); code != flowExitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if marker := flowReadMarker(t, root, "0001"); marker.Baseline == nil || len(marker.Baseline.Commit) != 40 {
		t.Fatalf("marker = %+v", marker)
	}
}

// A flow without an agent node needs no model, so the real outside -- which has
// no adapter -- runs it.
func TestRunOfAFlowWithoutAgentsAsksForNoModel(t *testing.T) {
	root := t.TempDir()
	flowWrite(t, root, load.Dir+"/ask/flow.toml", askFlow)
	flowWrite(t, root, load.Dir+"/ask/questions/confirm.md", "Ship it?\n")
	h := newFlowHarness()
	h.deps.Models = flowProduction(h.stdout, h.stderr).Models
	if code := h.run(root, "run", "ask"); code != flowExitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "run 0001 (ask, project): paused\nShip it?\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

// Every refusal before a run begins leaves nothing under the runs directory:
// no journal, and no marker holding a number for a run that never took a step.
func TestRunRefusesBeforeTheRunExists(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		arrange func(t *testing.T, root string, h *flowHarness)
		want    string
	}{
		{name: "a flow that is not there", args: []string{"run", "nope"}, want: `no flow named "nope"`},
		{name: "a flow that does not load", args: []string{"run", "broken"},
			arrange: func(t *testing.T, root string, _ *flowHarness) {
				flowWrite(t, root, load.Dir+"/broken/flow.toml", "schema_version = 7\n")
			},
			want: "schema_version 7 is unknown"},
		{name: "an option that is no parameter", args: []string{"run", "example", "--option", "rounds=3"},
			want: `option rounds is no parameter of flow "example"`},
		{name: "an option of the wrong type", args: []string{"run", "example", "--option", "max_rounds=five"},
			want: `option max_rounds: cannot read "five" as int`},
		{name: "a provider without an adapter", args: []string{"run", "example"},
			arrange: func(_ *testing.T, _ string, h *flowHarness) {
				h.deps.Models = flowProduction(h.stdout, h.stderr).Models
			},
			want: "no adapter for provider claude yet"},
		{name: "two providers in one flow", args: []string{"run", "pair"},
			arrange: func(t *testing.T, root string, _ *flowHarness) {
				flowWrite(t, root, load.Dir+"/pair/flow.toml", pairFlow)
				flowWrite(t, root, load.Dir+"/pair/instructions/one.md", "Judge the plan.\n")
				flowConfig(t, root, pairConfig)
			},
			want: `flow "pair" asks claude and gemini; a run holds one model, so a flow asks one provider`},
		{name: "a damaged config", args: []string{"run", "example"},
			arrange: func(t *testing.T, root string, _ *flowHarness) { flowConfig(t, root, "[agent]\ndefault = 3\n") },
			want:    "[agent] default must be a non-empty string"},
		{name: "two blocks of one kind", args: []string{"run", "example"},
			arrange: func(_ *testing.T, _ string, h *flowHarness) { h.deps.Blocks = append(h.deps.Blocks, blocks.Gate{}) },
			want:    `two blocks claim kind "gate"`},
		{name: "a flow that needs a baseline outside a repository", args: []string{"run", "measure"},
			arrange: func(t *testing.T, root string, h *flowHarness) {
				flowWrite(t, root, load.Dir+"/measure/flow.toml", measureFlow)
				h.deps.Blocks = append(h.deps.Blocks, measureBlock{})
			},
			want: "measure measures against the commit it starts from"},
		{name: "a ceiling an option makes zero", args: []string{"run", "example", "--option", "max_rounds=-1"},
			want: `node "draft" allows 0 visits`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			h := newFlowHarness(flowDone())
			if c.arrange != nil {
				c.arrange(t, root, h)
			}
			code := h.run(root, c.args...)
			if code != flowExitFailed || !strings.Contains(h.stderr.String(), c.want) {
				t.Fatalf("exit %d, stderr %q, want %q", code, h.stderr.String(), c.want)
			}
			if files := flowRunFiles(t, root); len(files) != 0 {
				t.Fatalf("a refused run left %v behind", files)
			}
		})
	}
}

// A run that took a step before its call failed keeps its marker: the journal
// is a record of what happened, and the marker says which flow it belongs to.
// The error names the run, or nobody could tell which one to show.
func TestARunThatTookAStepKeepsItsMarker(t *testing.T) {
	root := t.TempDir()
	flowWrite(t, root, load.Dir+"/flaky/flow.toml", flakyFlow)
	h := newFlowHarness()
	h.deps.Blocks = append(h.deps.Blocks, flakyBlock{})
	code := h.run(root, "run", "flaky")
	if code != flowExitFailed || !strings.HasPrefix(h.stderr.String(), "run 0001: ") ||
		!strings.Contains(h.stderr.String(), "the definition of b is gone") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if h.stdout.Len() != 0 {
		t.Fatalf("stdout = %q", h.stdout.String())
	}
	if files := flowRunFiles(t, root); !slices.Equal(files, []string{"0001.flow", "0001.jsonl"}) {
		t.Fatalf("files = %v", files)
	}
}

// A journal that cannot be looked at is not a journal that is absent: the
// marker stays, since the run may well have taken its step.
func TestAJournalThatCannotBeStattedKeepsItsMarker(t *testing.T) {
	root := t.TempDir()
	marker := runs.MarkerPath(root, "0001")
	flowWrite(t, root, runs.Dir+"/0001.flow", "flaky\n")
	denied := func(string) (fs.FileInfo, error) { return nil, fs.ErrPermission }
	if kept := forgetUnstarted(root, "0001", denied); !kept {
		t.Fatal("forgetUnstarted says the marker is gone")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("the marker is gone: %v", err)
	}
}

// A file where the runs directory belongs: the claim fails, and says where.
func TestRunRefusesWhenItCannotClaimANumber(t *testing.T) {
	root := t.TempDir()
	flowWrite(t, root, runs.Dir, "a file where the runs directory belongs")
	h := newFlowHarness(flowDone())
	if code := h.run(root, "run", "example"); code != flowExitFailed || !strings.Contains(h.stderr.String(), "creating") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
}
