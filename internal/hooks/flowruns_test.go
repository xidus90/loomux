package hooks

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/flow/journal"
	"github.com/xidus90/loomux/internal/flow/runs"
)

func TestWaitingRunsAnnouncesEachPausedRun(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "0001", runs.Marker{Flow: "example", Origin: "bundled", Version: "3.3.0"}, pausedLine("approve", "Ship it?"))
	writeRun(t, root, "0002", runs.Marker{Flow: "dev-cycle", Origin: "bundled+overlay", Overlays: []string{"instructions/review.md"}, Version: "3.3.0"}, pausedLine("approve_plan", "Plan ok?"))
	writeRun(t, root, "0003", runs.Marker{Flow: "example", Origin: "bundled", Version: "3.3.0"}, okLine("draft"))
	runningAs(t, filepath.FromSlash("C:/x/loomux.exe"))
	var stderr bytes.Buffer
	got := waitingRuns(root, "3.3.0", &stderr)
	want := []string{
		"run 0001 (example, bundled) is waiting at approve: Ship it?\n  a human answers it with: C:/x/loomux.exe flow resume 0001 --answer \"your answer\"",
		"run 0002 (dev-cycle, bundled+overlay: instructions/review.md) is waiting at approve_plan: Plan ok?\n  a human answers it with: C:/x/loomux.exe flow resume 0002 --answer \"your answer\"",
	}
	if !reflect.DeepEqual(got, want) || stderr.Len() != 0 {
		t.Fatalf("got %q\nstderr %q", got, stderr.String())
	}
}

// A binary that cannot name itself still leaves a command a human can type.
func TestWaitingRunsFallsBackToTheCommandName(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "0001", runs.Marker{Flow: "example", Origin: "project"}, pausedLine("approve", "Ship it?"))
	executable = func() (string, error) { return "", errors.New("no executable") }
	t.Cleanup(func() { executable = os.Executable })
	var stderr bytes.Buffer
	got := waitingRuns(root, "3.3.0", &stderr)
	want := []string{"run 0001 (example, project) is waiting at approve: Ship it?\n  a human answers it with: loomux flow resume 0001 --answer \"your answer\""}
	if !reflect.DeepEqual(got, want) || stderr.Len() != 0 {
		t.Fatalf("got %q\nstderr %q", got, stderr.String())
	}
}

// A path the shell would split or read is pasted in double quotes; a plain one
// stays bare, as above.
func TestWaitingRunsQuotesABinaryPathTheShellWouldSplit(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "0001", runs.Marker{Flow: "example", Origin: "bundled"}, pausedLine("approve", "Ship it?"))
	for path, want := range map[string]string{
		"C:/Users/John Doe/bin/loomux.exe": `"C:/Users/John Doe/bin/loomux.exe"`,
		"C:/R&D/loomux.exe":                `"C:/R&D/loomux.exe"`,
		"C:/x (y)/loomux.exe":              `"C:/x (y)/loomux.exe"`,
		"C:/Users/Wübbels/bin/loomux.exe":  "C:/Users/Wübbels/bin/loomux.exe",
	} {
		runningAs(t, filepath.FromSlash(path))
		var stderr bytes.Buffer
		got := waitingRuns(root, "3.3.0", &stderr)
		line := "  a human answers it with: " + want + " flow resume 0001 --answer \"your answer\""
		if len(got) != 1 || !strings.HasSuffix(got[0], "\n"+line) || stderr.Len() != 0 {
			t.Errorf("%s: got %q, want the line %q\nstderr %q", path, got, line, stderr.String())
		}
	}
}

// A question read from a file ends in a newline of its own; the announcement
// ends it where the command follows, as flow run prints the question.
func TestWaitingRunsTrimsTheQuestionsTrailingNewlines(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "0001", runs.Marker{Flow: "example", Origin: "bundled"}, pausedLine("approve", "Ship it?\n\n"))
	runningAs(t, filepath.FromSlash("C:/x/loomux.exe"))
	var stderr bytes.Buffer
	got := waitingRuns(root, "3.3.0", &stderr)
	want := []string{"run 0001 (example, bundled) is waiting at approve: Ship it?\n  a human answers it with: C:/x/loomux.exe flow resume 0001 --answer \"your answer\""}
	if !reflect.DeepEqual(got, want) || stderr.Len() != 0 {
		t.Fatalf("got %q\nstderr %q", got, stderr.String())
	}
}

// A runs folder that cannot be listed is named, without a verdict on the
// project; an absent one is a project without runs.
func TestWaitingRunsNamesARunsFolderItCannotList(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, filepath.FromSlash(runs.Dir))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	got := waitingRuns(root, "3.3.0", &stderr)
	want := "loomux hook session-start: .loomux/state/runs cannot be read as a folder of runs: it is a file\n"
	if len(got) != 0 || stderr.String() != want {
		t.Fatalf("got %q\nstderr %q", got, stderr.String())
	}
}

func TestWaitingRunsSaysNothingWithoutRuns(t *testing.T) {
	var stderr bytes.Buffer
	if got := waitingRuns(t.TempDir(), "3.3.0", &stderr); len(got) != 0 || stderr.Len() != 0 {
		t.Fatalf("got %q\nstderr %q", got, stderr.String())
	}
}

func TestWaitingRunsNamesAJournalAnotherBinaryWrote(t *testing.T) {
	// A line with a key this reader does not know, from a marker with another
	// version: the hook names both versions instead of calling it damaged.
	root := t.TempDir()
	writeRun(t, root, "0001", runs.Marker{Flow: "example", Origin: "bundled", Version: "9.9.9"})
	writeJournal(t, root, "0001", `{"definition_hash":null,"delta":{},"detail":"Ship it?","effort":null,"input_hash":"h","kind":"gate","model":null,"node":"approve","outcome":"paused","role":null,"seconds":0,"tokens":0,"tools":null,"attempt":2}`)
	var stderr bytes.Buffer
	got := waitingRuns(root, "3.3.0", &stderr)
	want := "loomux hook session-start: run 0001 was written by loomux 9.9.9, this is 3.3.0\n"
	if len(got) != 0 || stderr.String() != want {
		t.Fatalf("got %q\nstderr %q", got, stderr.String())
	}
}

// A marker that names no version, or this binary's, leaves the journal
// damaged: nothing else could have written it.
func TestWaitingRunsCallsAJournalOfThisVersionDamaged(t *testing.T) {
	for _, version := range []string{"", "3.3.0"} {
		root := t.TempDir()
		writeRun(t, root, "0001", runs.Marker{Flow: "example", Origin: "bundled", Version: version})
		writeJournal(t, root, "0001", "{not json")
		var stderr bytes.Buffer
		got := waitingRuns(root, "3.3.0", &stderr)
		if len(got) != 0 || !strings.HasPrefix(stderr.String(), "loomux hook session-start: ") ||
			!strings.Contains(stderr.String(), "0001.jsonl: line 1 is not a journal entry") {
			t.Fatalf("version %q: got %q\nstderr %q", version, got, stderr.String())
		}
	}
}

func TestWaitingRunsNamesADamagedJournalAndGoesOn(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "0001", runs.Marker{Flow: "example", Origin: "bundled", Version: "3.3.0"})
	writeJournal(t, root, "0001", "{not json")
	writeRun(t, root, "0002", runs.Marker{Flow: "example", Origin: "bundled", Version: "3.3.0"}, pausedLine("approve", "Ship it?"))
	runningAs(t, filepath.FromSlash("C:/x/loomux.exe"))
	var stderr bytes.Buffer
	got := waitingRuns(root, "3.3.0", &stderr)
	want := []string{"run 0002 (example, bundled) is waiting at approve: Ship it?\n  a human answers it with: C:/x/loomux.exe flow resume 0002 --answer \"your answer\""}
	if !reflect.DeepEqual(got, want) || !strings.Contains(stderr.String(), "0001.jsonl: line 1 is not a journal entry") {
		t.Fatalf("got %q\nstderr %q", got, stderr.String())
	}
}

// A damaged marker is named like a damaged journal, and hides only its own run:
// resume refuses a run whose marker will not read, so no command is offered.
func TestWaitingRunsNamesADamagedMarkerAndGoesOn(t *testing.T) {
	root := t.TempDir()
	writeJournal(t, root, "0001", mustLine(t, pausedLine("approve", "Ship it?")))
	if err := os.WriteFile(runs.MarkerPath(root, "0001"), []byte("example\norigin\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeRun(t, root, "0002", runs.Marker{Flow: "example", Origin: "bundled", Version: "3.3.0"}, pausedLine("approve", "Ship it?"))
	runningAs(t, filepath.FromSlash("C:/x/loomux.exe"))
	var stderr bytes.Buffer
	got := waitingRuns(root, "3.3.0", &stderr)
	if len(got) != 1 || !strings.HasPrefix(got[0], "run 0002 ") ||
		!strings.Contains(stderr.String(), "0001.flow: option line without '='") {
		t.Fatalf("got %q\nstderr %q", got, stderr.String())
	}
}

// A journal without its marker says no flow to resume it with; resume refuses
// it in the same words.
func TestWaitingRunsNamesARunWithoutAMarker(t *testing.T) {
	root := t.TempDir()
	writeJournal(t, root, "0001", mustLine(t, pausedLine("approve", "Ship it?")))
	var stderr bytes.Buffer
	got := waitingRuns(root, "3.3.0", &stderr)
	want := "loomux hook session-start: run \"0001\" does not say which flow it belongs to\n"
	if len(got) != 0 || stderr.String() != want {
		t.Fatalf("got %q\nstderr %q", got, stderr.String())
	}
}

// Only journals are runs: a marker alone, a folder named like a journal and
// anything else in the directory announce nothing.
func TestWaitingRunsReadsOnlyJournals(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "0001", runs.Marker{Flow: "example", Origin: "bundled"})
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(runs.Dir), "0002.jsonl"), 0o755); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if got := waitingRuns(root, "3.3.0", &stderr); len(got) != 0 || stderr.Len() != 0 {
		t.Fatalf("got %q\nstderr %q", got, stderr.String())
	}
}

func TestIgnoredFlowFoldersWarns(t *testing.T) {
	root := project(t)
	flowFolder(t, root, "example/flow.toml")
	flowFolder(t, root, "mine/flow.toml")
	want := []string{".loomux/flows/example is ignored: [flow] overrides does not name it"}
	if got := ignoredFlowFolders(root); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}

	manifest(t, root, "[flow]\noverrides = [\"example\"]\n")
	if got := ignoredFlowFolders(root); len(got) != 0 {
		t.Fatalf("an override the config names: got %q", got)
	}
}

// load.Find warns for a file named like a bundled flow too: the project cannot
// hide the bundled flow with anything it holds under that name.
func TestIgnoredFlowFoldersWarnsForAFileNamedLikeABundledFlow(t *testing.T) {
	root := project(t)
	flowFolder(t, root, "example")
	want := []string{".loomux/flows/example is ignored: [flow] overrides does not name it"}
	if got := ignoredFlowFolders(root); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}

// The listing decides, spelled exactly: Example is not the bundled example on
// a file system that ignores case either.
func TestIgnoredFlowFoldersComparesNamesExactly(t *testing.T) {
	root := project(t)
	flowFolder(t, root, "Example/flow.toml")
	if got := ignoredFlowFolders(root); len(got) != 0 {
		t.Fatalf("got %q", got)
	}
}

// A project without flows pays no second decode of its config, so a config
// that will not read costs this check nothing and says nothing either.
func TestIgnoredFlowFoldersReadsNoConfigWithoutFlows(t *testing.T) {
	root := project(t)
	manifest(t, root, "[flow\n")
	if got := ignoredFlowFolders(root); len(got) != 0 {
		t.Fatalf("got %q", got)
	}
	flowFolder(t, root, "mine/flow.toml")
	if got := ignoredFlowFolders(root); len(got) != 0 {
		t.Fatalf("a folder no bundled flow is named like: got %q", got)
	}
}

// A flows folder that cannot be listed is said in load.Find's words.
func TestIgnoredFlowFoldersNamesAFlowsFolderItCannotList(t *testing.T) {
	root := project(t)
	if err := os.WriteFile(filepath.Join(root, ".loomux", "flows"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := []string{".loomux/flows cannot be read as a folder of flows: it is a file"}
	if got := ignoredFlowFolders(root); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}

func TestIgnoredFlowFoldersNamesAConfigItCannotRead(t *testing.T) {
	root := project(t)
	manifest(t, root, "[flow\n")
	flowFolder(t, root, "example/flow.toml")
	got := ignoredFlowFolders(root)
	if len(got) != 1 || !strings.HasPrefix(got[0], "loomux: [flow] cannot be read: ") {
		t.Fatalf("got %q", got)
	}
}

// writeRun writes a run's marker and its journal, one entry per line.
func writeRun(t *testing.T, root, id string, marker runs.Marker, entries ...journal.Entry) {
	t.Helper()
	if err := runs.WriteMarker(runs.MarkerPath(root, id), marker); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if err := journal.Append(runs.JournalPath(root, id), entry); err != nil {
			t.Fatal(err)
		}
	}
}

// writeJournal writes a run's journal as it stands, for lines Append would not
// write.
func writeJournal(t *testing.T, root, id, text string) {
	t.Helper()
	path := runs.JournalPath(root, id)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mustLine is an entry as Append spells it.
func mustLine(t *testing.T, entry journal.Entry) string {
	t.Helper()
	line, err := journal.Canonical(entry)
	if err != nil {
		t.Fatal(err)
	}
	return string(line)
}

// pausedLine is a gate that stopped its run with a question.
func pausedLine(node, question string) journal.Entry {
	return journal.Entry{Node: node, Kind: "gate", InputHash: "h-" + node, Delta: map[string]any{}, Outcome: "paused", Detail: &question}
}

// okLine is a node that finished.
func okLine(node string) journal.Entry {
	return journal.Entry{Node: node, Kind: "work", InputHash: "h-" + node, Delta: map[string]any{}, Outcome: "ok"}
}

// flowFolder puts a file under the project's .loomux/flows/.
func flowFolder(t *testing.T, root, path string) {
	t.Helper()
	full := filepath.Join(root, ".loomux", "flows", filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// manifest replaces the project's config.
func manifest(t *testing.T, root, text string) {
	t.Helper()
	if err := os.WriteFile(config.ManifestPath(root), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
