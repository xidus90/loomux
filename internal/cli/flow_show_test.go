package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow/load"
	"github.com/xidus90/loomux/internal/flow/runs"
)

func TestShowPrintsOneLinePerEntry(t *testing.T) {
	h := newFlowHarness(flowDone())
	root := flowPausedExample(t, h)
	if code := h.run(root, "resume", "0001", "--answer", "no: too thin"); code != 4 {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if code := h.run(root, "show", "0001"); code != flowExitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	got := strings.Split(strings.TrimSuffix(h.stdout.String(), "\n"), "\n")
	want := [][]string{
		{"draft", "agent", "ok", "120", "tok", "0.50s", "edit"},
		{"approve", "gate", "paused", "0", "tok", "0.50s", "-"},
		{"approve", "gate", "ok", "0", "tok", "0.00s", "-"},
		{"stop", "exit", "error", "0", "tok", "0.50s", "-"},
	}
	if len(got) != len(want) {
		t.Fatalf("lines = %q", got)
	}
	for i := range want {
		if fields := strings.Fields(got[i]); !slices.Equal(fields, want[i]) {
			t.Fatalf("line %d = %q, want %v", i+1, got[i], want[i])
		}
	}
	// The columns are fixed: node in 24, kind in 6.
	if strings.Index(got[0], "agent") != 25 || strings.Index(got[0], "ok") != 32 {
		t.Fatalf("columns moved: %q", got[0])
	}
}

func TestShowRefuses(t *testing.T) {
	h := newFlowHarness(flowDone())
	root := flowPausedExample(t, h)
	if code := h.run(root, "show", "0009"); code != flowExitFailed || !strings.Contains(h.stderr.String(), `no run "0009" under`) {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	flowOverwrite(t, runs.JournalPath(root, "0001"), "{not json\n")
	if code := h.run(root, "show", "0001"); code != flowExitFailed || !strings.Contains(h.stderr.String(), "line 1 is not a journal entry") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	if code := h.run(root, "show", "ghost"); code != flowExitFailed || !strings.Contains(h.stderr.String(), `no flow named "ghost"`) {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
}

// A flow that will not load is listed with every finding, the further ones
// indented under the first; a file where a flow's folder belongs is listed by
// its name, with no origin.
func TestListNamesEveryFlowAndWhyOneWillNotLoad(t *testing.T) {
	root := t.TempDir()
	flowWrite(t, root, load.Dir+"/broken/flow.toml", "schema_version = 7\n")
	flowWrite(t, root, load.Dir+"/old.toml", "schema_version = 1\n")
	h := newFlowHarness()
	if code := h.run(root, "list"); code != flowExitOK {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
	got := strings.Split(strings.TrimSuffix(h.stdout.String(), "\n"), "\n")
	if len(got) < 4 {
		t.Fatalf("lines = %q", got)
	}
	if want := "broken    project  .loomux/flows/broken/flow.toml: schema_version 7 is unknown"; !strings.HasPrefix(got[0], want) {
		t.Fatalf("first line = %q, want it to start with %q", got[0], want)
	}
	if !strings.HasPrefix(got[1], "    .loomux/flows/broken/flow.toml: ") {
		t.Fatalf("a further finding is indented: %q", got[1])
	}
	if want := []string{
		"example   bundled  ok",
		"old.toml  -        .loomux/flows/old.toml is a file; a flow is a folder with flow.toml",
	}; !slices.Equal(got[len(got)-2:], want) {
		t.Fatalf("last lines = %q, want %q", got[len(got)-2:], want)
	}
}

func TestListRefusesAProjectItCannotRead(t *testing.T) {
	root := t.TempDir()
	flowConfig(t, root, "[agent]\ndefault = 3\n")
	h := newFlowHarness()
	if code := h.run(root, "list"); code != flowExitFailed || !strings.Contains(h.stderr.String(), "[agent] default must be a non-empty string") {
		t.Fatalf("exit %d, stderr %q", code, h.stderr.String())
	}
}
