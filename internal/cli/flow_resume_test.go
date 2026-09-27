package cli

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow/load"
	"github.com/xidus90/loomux/internal/flow/runs"
)

func TestResumeWithARejectionEndsAtTheExit(t *testing.T) {
	h := newFlowHarness(flowDone())
	root := flowPausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "no: too thin"); code != 4 {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "run 0001 (example, bundled): error\nrejected after 2 rounds\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := flowLines(t, runs.JournalPath(root, "0001")); len(got) != 4 {
		t.Fatalf("journal = %q", got)
	}
	if seen := h.fake.Seen(); len(seen) != 1 {
		t.Fatalf("the resume asked the model again: %d requests", len(seen))
	}
}

func TestResumeWithApprovalIsDone(t *testing.T) {
	h := newFlowHarness(flowDone())
	root := flowPausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "yes"); code != flowExitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got := h.stdout.String(); got != "run 0001 (example, bundled): done\n" {
		t.Fatalf("stdout = %q", got)
	}
}

// Without --answer the gate asks again and nothing is written: a run somebody
// looks at is not a run that moved.
func TestResumeWithoutAnAnswerAsksAgain(t *testing.T) {
	h := newFlowHarness(flowDone())
	root := flowPausedExample(t, h)
	before := flowReadBytes(t, runs.JournalPath(root, "0001"))
	if code := h.run(root, "resume", "0001"); code != flowExitPaused {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "run 0001 (example, bundled): paused\nApprove the plan after 2 rounds?\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if after := flowReadBytes(t, runs.JournalPath(root, "0001")); !bytes.Equal(before, after) {
		t.Fatalf("the look wrote to the journal:\n%s\nbecame\n%s", before, after)
	}
}

// An answer no choice matches is refused, names the choices, and leaves the
// gate open for the next one.
func TestResumeRefusesAnAnswerNoChoiceMatches(t *testing.T) {
	h := newFlowHarness(flowDone())
	root := flowPausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "maybe"); code != flowExitFailed || !strings.Contains(h.stderr.String(), "the choices are yes, no") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if code := h.run(root, "resume", "0001", "--answer", "yes"); code != flowExitOK {
		t.Fatalf("the gate did not stay open: exit %d, stderr %q", code, h.stderr.String())
	}
}

func TestResumeAndReplayRefuse(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		arrange func(t *testing.T, root string, h *flowHarness)
		want    string
	}{
		{name: "a run that is not there", args: []string{"resume", "0009"}, want: `no run "0009" under`},
		{name: "a replay of a run that is not there", args: []string{"replay", "0009"}, want: `no run "0009" under`},
		{name: "a run without its marker", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, _ *flowHarness) { flowRemove(t, runs.MarkerPath(root, "0001")) },
			want:    `run "0001" does not say which flow it belongs to`},
		{name: "a damaged marker", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, _ *flowHarness) {
				flowOverwrite(t, runs.MarkerPath(root, "0001"), "example\nno equals sign\n")
			},
			want: "option line without '='"},
		{name: "a marker of an origin this build does not write", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, _ *flowHarness) {
				flowOverwrite(t, runs.MarkerPath(root, "0001"), "example\norigin=\"elsewhere\"\n")
			},
			want: "run 0001 started on elsewhere and example now resolves to bundled, another flow.toml"},
		{name: "a marker that names no origin", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, _ *flowHarness) {
				flowOverwrite(t, runs.MarkerPath(root, "0001"), "example\n")
			},
			want: "run 0001 started on (none) and example now resolves to bundled, another flow.toml"},
		{name: "an option the flow no longer declares", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, _ *flowHarness) {
				flowOverwrite(t, runs.MarkerPath(root, "0001"), "example\nfoo=\"1\"\norigin=\"bundled\"\n")
			},
			want: `option foo is no parameter of flow "example"`},
		{name: "a damaged journal", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, _ *flowHarness) {
				flowOverwrite(t, runs.JournalPath(root, "0001"), "{not json\n")
			},
			want: "line 1 is not a journal entry"},
		{name: "a flow that is gone", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, _ *flowHarness) {
				flowOverwrite(t, runs.MarkerPath(root, "0001"), "gone\norigin=\"project\"\n")
			},
			want: `no flow named "gone"`},
		{name: "a provider without an adapter", args: []string{"resume", "0001", "--answer", "yes"},
			arrange: func(_ *testing.T, _ string, h *flowHarness) {
				h.deps.Models = flowProduction(h.stdout, h.stderr).Models
			},
			want: "no adapter for provider claude yet"},
		{name: "a run that waits at no gate", args: []string{"resume", "0001"},
			arrange: func(t *testing.T, root string, h *flowHarness) {
				if code := h.run(root, "resume", "0001", "--answer", "yes"); code != flowExitOK {
					t.Fatalf("exit %d", code)
				}
			},
			want: "run 0001 is not waiting at a gate; there is nothing to answer"},
		{name: "a replay of a run that waits at a gate", args: []string{"replay", "0001"},
			want: `run 0001 never finished: it is waiting at gate "approve"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newFlowHarness(flowDone())
			root := flowPausedExample(t, h)
			if c.arrange != nil {
				c.arrange(t, root, h)
			}
			code := h.run(root, c.args...)
			if code != flowExitFailed || !strings.Contains(h.stderr.String(), c.want) {
				t.Fatalf("exit %d, stderr %q, want %q", code, h.stderr.String(), c.want)
			}
		})
	}
}

// A run number is digits, as flow run hands them out, and it is joined into a
// path: anything else would let a forged journal elsewhere in the project walk
// past a gate no human answered. It is a usage error, refused before anything
// is read.
func TestResumeAndReplayRefuseARunIDThatIsNotDigits(t *testing.T) {
	root := t.TempDir()
	for _, id := range []string{"../x", `..\x`, filepath.Join(root, "x"), "0001x", ""} {
		for _, command := range []string{"resume", "replay"} {
			h := newFlowHarness()
			code := h.run(root, command, id)
			want := "loomux flow " + command + ": " + strconv.Quote(id) + " is no run number; a run number is digits, as loomux flow run hands them out"
			if code != flowExitUsage || !strings.HasPrefix(h.stderr.String(), want+"\n") {
				t.Errorf("%s %q: exit %d, stderr %q, want %q", command, id, code, h.stderr.String(), want)
			}
		}
	}
}

// The runs folder is named with forward slashes, as every other path.
func TestANoRunRefusalNamesTheRunsFolderWithForwardSlashes(t *testing.T) {
	root := t.TempDir()
	h := newFlowHarness()
	if code := h.run(root, "resume", "0009"); code != flowExitFailed {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stderr.String(), `no run "0009" under `+filepath.ToSlash(root)+"/.loomux/state/runs\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

// A run whose flow measures against a commit, started where there was none,
// cannot be carried on: taking a baseline now would measure against the tree
// the run has meanwhile edited.
func TestARunWithoutTheBaselineItsFlowNeedsIsNotCarriedOn(t *testing.T) {
	root := t.TempDir()
	flowWrite(t, root, load.Dir+"/measure/flow.toml", measureFlow)
	marker := runs.Marker{Flow: "measure", Origin: load.OriginProject, Version: "0.0.0-test"}
	if err := runs.WriteMarker(runs.MarkerPath(root, "0001"), marker); err != nil {
		t.Fatal(err)
	}
	flowWrite(t, root, runs.Dir+"/0001.jsonl", "")
	h := newFlowHarness()
	h.deps.Blocks = append(h.deps.Blocks, measureBlock{})
	if code := h.run(root, "replay", "0001"); code != flowExitFailed || !strings.Contains(h.stderr.String(), "run 0001 was started outside a repository") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
}

// A replay reproduces how the run ended, runs no node and writes nothing. The
// journal carries no exit code, so a replayed exit is a failure without one.
func TestReplayReproducesTheEndingAndWritesNothing(t *testing.T) {
	h := newFlowHarness(flowDone())
	root := flowPausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "no: too thin"); code != 4 {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	before := flowReadBytes(t, runs.JournalPath(root, "0001"))
	if code := h.run(root, "replay", "0001"); code != flowExitFailed {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if got, want := h.stdout.String(), "run 0001 (example, bundled): error\nrejected after 2 rounds\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if after := flowReadBytes(t, runs.JournalPath(root, "0001")); !bytes.Equal(before, after) {
		t.Fatal("the replay wrote to the journal")
	}
	if seen := h.fake.Seen(); len(seen) != 1 {
		t.Fatalf("the replay asked the model: %d requests", len(seen))
	}
}

// An instruction improved since the run is reported, never refused. The
// bundled example gets its new instruction through an overlay, and the
// overlay the run did not have is reported too.
func TestReplayWarnsAboutAChangedInstruction(t *testing.T) {
	h := newFlowHarness(flowDone())
	root := flowPausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "yes"); code != flowExitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	flowConfig(t, root, "[flow]\noverrides = [\"example\"]\n")
	flowWrite(t, root, load.Dir+"/example/instructions/draft.md",
		"Write a better draft plan. You have {{max_rounds}} rounds.\n")
	code := h.run(root, "replay", "0001")
	stderr := h.stderr.String()
	if code != flowExitOK || !strings.Contains(stderr, `warning: node "draft" ran on a definition that has changed since`) ||
		!strings.Contains(stderr, "warning: run 0001 started with overlays none and now has instructions/draft.md\n") {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if got, want := h.stdout.String(), "run 0001 (example, bundled+overlay: instructions/draft.md): done\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}
