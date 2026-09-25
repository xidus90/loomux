package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/tui"
)

func proposalDir(root string) string {
	return filepath.Join(root, ".loomux", "state", "config", "proposals")
}

func proposalFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// forgeProposal writes a proposal file the way an agent with file access
// could, to show that apply acts on op, key and input alone.
func forgeProposal(t *testing.T, dir string, p proposal) {
	t.Helper()
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, p.ID+".json"), string(data))
}

func TestConfigProposeStoresTheChangeAndWritesNothing(t *testing.T) {
	const text = "[commit]\nlanguage = \"en\"\n"
	root := configRoot(t, text)
	code, out, errOut := runConfig(t, "", "set", "commit.language", "de", "--propose", "--root", root)
	if code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	if got := readConfig(t, root); got != text {
		t.Fatalf("file written:\n%s", got)
	}
	names := proposalFiles(t, proposalDir(root))
	if len(names) != 1 {
		t.Fatalf("files %v", names)
	}
	id := strings.TrimSuffix(names[0], ".json")
	if !strings.Contains(out, "+ language = \"de\"") || !strings.Contains(out, "loomux config apply "+id) {
		t.Fatalf("out:\n%s", out)
	}
	var p proposal
	data, _ := os.ReadFile(filepath.Join(proposalDir(root), names[0]))
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	created, err := time.Parse(time.RFC3339, p.Created)
	if p.ID != id || p.Op != "set" || p.Key != "commit.language" || p.Input != "de" || p.Target != "project" ||
		err != nil || !strings.HasSuffix(p.Created, "Z") || created.IsZero() {
		t.Fatalf("proposal %+v (%v)", p, err)
	}
	// The diff is computed whenever it is shown, so none is kept.
	if strings.Contains(string(data), `"diff"`) {
		t.Fatalf("stored a diff:\n%s", data)
	}
}

// forgeRaw writes a proposal file as bytes, for what the proposal type
// itself cannot carry.
func forgeRaw(t *testing.T, dir, id, text string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, id+".json"), text)
}

// Key, input and the diff come from an agent and reach a human's terminal:
// control bytes are shown escaped, never sent.
func TestConfigProposalTextIsEscapedForTheTerminal(t *testing.T) {
	root := configRoot(t, "")
	dir := proposalDir(root)
	forgeProposal(t, dir, proposal{ID: "a-1", Op: "set", Key: "area.scope", Input: "x\x1b[2Jy"})
	forgeProposal(t, dir, proposal{ID: "a-2", Op: "set", Key: "commit.\x1b]0;t\x07", Input: "\u009b1m"})
	forgeProposal(t, dir, proposal{ID: "a-3", Op: "set", Key: "area.scope", Input: "d\x7fe\u009bf"})
	_, out, errOut := runConfig(t, "", "proposals", "--root", root)
	// JSON escapes only C0 controls itself; DEL and the C1 range would pass.
	_, asJSON, _ := runConfig(t, "", "proposals", "--json", "--root", root)
	if strings.ContainsAny(asJSON, "\x7f\u009b") || !strings.Contains(asJSON, `d\\x7fe\\x9bf`) {
		t.Fatalf("JSON:\n%s", asJSON)
	}
	_, _, applied := runConfig(t, "", "apply", "--all", "--yes", "--root", root)
	for _, text := range []string{out, errOut, applied} {
		if strings.ContainsAny(text, "\x1b\x07\u009b") {
			t.Fatalf("raw control bytes:\n%q", text)
		}
	}
	if !strings.Contains(out, `\x1b[2J`) || !strings.Contains(applied, `\x1b[2J`) {
		t.Fatalf("not escaped:\n%s\n%s", out, applied)
	}
	if code, out, _ := runConfig(t, "", "set", "area.scope", "a\x1bb", "--propose", "--root", configRoot(t, "")); code != 0 || strings.Contains(out, "\x1b") {
		t.Fatalf("propose: %d %q", code, out)
	}
}

func TestCleanKeepsLinesAndEscapesControls(t *testing.T) {
	// A byte that is no UTF-8 may be half of a C1 control on a terminal
	// that reads Latin-1, so it is escaped as well.
	if got := clean("a\tb\nc\x1bd\x7fe\u009bf\rg\x9bh"); got != "a\tb\nc\\x1bd\\x7fe\\x9bf\\x0dg\\x9bh" {
		t.Fatalf("%q", got)
	}
	for in, want := range map[string]string{"de": "de", "": `""`, "a b": `"a b"`, "x\x1b": `"x\x1b"`, "ü": "ü"} {
		if got := word(in); got != want {
			t.Errorf("word(%q) = %q, want %q", in, got, want)
		}
	}
}

// Without hard links the name is claimed by an exclusive create instead.
func TestStoreProposalWithoutHardLinks(t *testing.T) {
	restore := linkFile
	linkFile = func(string, string) error { return errors.New("not supported") }
	t.Cleanup(func() { linkFile = restore })
	dir := t.TempDir()
	now := time.Date(2026, 9, 24, 10, 15, 30, 0, time.UTC)
	first, err1 := storeProposal(dir, now, proposal{Op: "set", Key: "a"})
	second, err2 := storeProposal(dir, now, proposal{Op: "set", Key: "b"})
	if err1 != nil || err2 != nil || first != "20260924T101530Z-001" || second != "20260924T101530Z-002" {
		t.Fatalf("%s %v %s %v", first, err1, second, err2)
	}
	data, _ := os.ReadFile(filepath.Join(dir, second+".json"))
	if names := proposalFiles(t, dir); len(names) != 2 || !strings.Contains(string(data), `"key": "b"`) {
		t.Fatalf("%v %s", names, data)
	}
	// A directory that cannot be made is reported, not retried.
	if _, err := storeProposal(filepath.Join(dir, "missing", "\x00"), now, proposal{}); err == nil {
		t.Fatal("no error")
	}
}

// The write succeeded, only the proposal file stayed: the human hears that
// the change is in, not that it is still waiting.
func TestConfigApplyWhenTheProposalCannotBeRemoved(t *testing.T) {
	restore := removeProposalFile
	removeProposalFile = func(string) error { return errors.New("locked") }
	t.Cleanup(func() { removeProposalFile = restore })
	root := configRoot(t, "[commit]\nlanguage = \"de\"\n")
	dir := proposalDir(root)
	forgeProposal(t, dir, proposal{ID: "a-1", Op: "set", Key: "commit.threshold", Input: "4"})
	forgeProposal(t, dir, proposal{ID: "a-2", Op: "set", Key: "commit.language", Input: "de"})
	code, _, errOut := runConfig(t, "", "apply", "--all", "--yes", "--root", root)
	if code != 1 || strings.Contains(errOut, "stays") ||
		!strings.Contains(errOut, "wrote") || !strings.Contains(errOut, "could not be removed: locked") ||
		!strings.Contains(errOut, "already so, but its file could not be removed") {
		t.Fatalf("%d %s", code, errOut)
	}
	if got := readConfig(t, root); got != "[commit]\nlanguage = \"de\"\nthreshold = 4\n" {
		t.Fatalf("file:\n%s", got)
	}
}

func TestConfigApplyWhenTheWriteFails(t *testing.T) {
	restore := applyWrite
	applyWrite = func(configTarget, string, string) error { return errors.New("disk full") }
	t.Cleanup(func() { applyWrite = restore })
	root := configRoot(t, "")
	forgeProposal(t, proposalDir(root), proposal{ID: "a-1", Op: "set", Key: "commit.threshold", Input: "4"})
	code, _, errOut := runConfig(t, "", "apply", "a-1", "--yes", "--root", root)
	if code != 1 || !strings.Contains(errOut, "disk full; nothing written, the proposal stays") || len(proposalFiles(t, proposalDir(root))) != 1 {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestConfigProposeUnset(t *testing.T) {
	root := configRoot(t, "[commit]\nthreshold = 5\n")
	code, out, _ := runConfig(t, "", "unset", "commit.threshold", "--propose", "--root", root)
	if code != 0 || !strings.Contains(out, "- threshold = 5") || readConfig(t, root) != "[commit]\nthreshold = 5\n" {
		t.Fatalf("%d %s", code, out)
	}
	_, list, _ := runConfig(t, "", "proposals", "--root", root)
	if !strings.Contains(list, "unset commit.threshold") {
		t.Fatalf("proposals:\n%s", list)
	}
}

func TestConfigProposeOfWhatIsAlreadySoStoresNothing(t *testing.T) {
	root := configRoot(t, "[commit]\nthreshold = 3\n")
	code, _, errOut := runConfig(t, "", "set", "commit.threshold", "3", "--propose", "--root", root)
	if code != 0 || !strings.Contains(errOut, "already so") || len(proposalFiles(t, proposalDir(root))) != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestConfigProposeRefusesWhatSetRefuses(t *testing.T) {
	root := configRoot(t, "")
	for _, args := range [][]string{
		{"set", "commit.language", "fr"},
		{"set", "commit.threshold", "abc"},
		{"set", "privacy.mode", "local_only"},
		{"unset", "commit.nope"},
	} {
		args = append(args, "--propose", "--root", root)
		if code, _, errOut := runConfig(t, "", args...); code != 1 || errOut == "" {
			t.Errorf("%v: %d %q", args, code, errOut)
		}
	}
	if names := proposalFiles(t, proposalDir(root)); len(names) != 0 {
		t.Fatalf("stored %v", names)
	}
	unreadable := t.TempDir()
	if err := os.MkdirAll(filepath.Join(unreadable, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := runConfig(t, "", "set", "commit.threshold", "4", "--propose", "--root", unreadable); code != 1 || errOut == "" {
		t.Fatalf("unreadable: %d %q", code, errOut)
	}
}

func TestConfigProposeReportsAStoreThatFails(t *testing.T) {
	// .loomux is a file, so the proposal directory cannot be made.
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux"), "")
	if code, _, errOut := runConfig(t, "", "set", "commit.threshold", "4", "--propose", "--root", root); code != 1 || errOut == "" {
		t.Fatalf("%d %q", code, errOut)
	}
}

// Two proposals of one second get two names, in the order they were made.
func TestStoreProposalTakesTheNextFreeName(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 9, 24, 10, 15, 30, 0, time.UTC)
	first, err := storeProposal(dir, now, proposal{Op: "set", Key: "a"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := storeProposal(dir, now, proposal{Op: "set", Key: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if first != "20260924T101530Z-001" || second != "20260924T101530Z-002" {
		t.Fatalf("%s %s", first, second)
	}
	if names := proposalFiles(t, dir); len(names) != 2 {
		t.Fatalf("leftovers: %v", names)
	}
}

func TestConfigProposalsListsOpenProposals(t *testing.T) {
	root := configRoot(t, "")
	if code, out, _ := runConfig(t, "", "proposals", "--root", root); code != 0 || out != "" {
		t.Fatalf("none: %d %q", code, out)
	}
	if code, out, _ := runConfig(t, "", "proposals", "--json", "--root", root); code != 0 || out != "[]\n" {
		t.Fatalf("none as JSON: %d %q", code, out)
	}
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[commit]\nlanguage = \"de\"\n")
	dir := proposalDir(root)
	// Each diff is computed now against the current file; what the file
	// says its diff is counts for nothing.
	forgeRaw(t, dir, "20260101T000000Z-002", `{"op":"set","key":"commit.threshold","input":"7","created":"2026-01-01T00:00:00Z","diff":"+ evil = true\n"}`)
	forgeProposal(t, dir, proposal{ID: "20260101T000000Z-001", Op: "unset", Key: "commit.language", Created: "2026-01-01T00:00:00Z"})
	forgeProposal(t, dir, proposal{ID: "20260101T000000Z-003", Op: "set", Key: "commit.language", Input: "fr"})
	forgeProposal(t, dir, proposal{ID: "20260101T000000Z-004", Op: "set", Key: "commit.language", Input: "de"})
	code, out, _ := runConfig(t, "", "proposals", "--root", root)
	first := strings.Index(out, "20260101T000000Z-001  2026-01-01T00:00:00Z  unset commit.language\n- [commit]")
	second := strings.Index(out, "20260101T000000Z-002  2026-01-01T00:00:00Z  set commit.threshold 7\n  language = \"de\"\n+ threshold = 7")
	third := strings.Index(out, "20260101T000000Z-003")
	if code != 0 || first < 0 || second < first || strings.Contains(out, "evil") ||
		!strings.Contains(out[third:], "refused now:") || !strings.Contains(out, "already so; apply removes it") {
		t.Fatalf("%d\n%s", code, out)
	}
	code, out, _ = runConfig(t, "", "proposals", "--json", "--root", root)
	var list []proposalView
	if code != 0 || json.Unmarshal([]byte(out), &list) != nil || len(list) != 4 || list[0].ID != "20260101T000000Z-001" ||
		list[1].Input != "7" || !strings.Contains(list[1].Diff, "+ threshold = 7") || list[1].Error != "" ||
		list[2].Diff != "" || !strings.Contains(list[2].Error, "language") || strings.Contains(out, "evil") {
		t.Fatalf("%d %s", code, out)
	}
	// The file that does not read cannot be compared with.
	if err := os.Remove(filepath.Join(root, ".loomux", "config.toml")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := runConfig(t, "", "proposals", "--root", root); code != 1 || errOut == "" {
		t.Fatalf("unreadable: %d %q", code, errOut)
	}
}

func TestConfigProposalsReportsWhatItCannotRead(t *testing.T) {
	root := configRoot(t, "")
	writeFile(t, filepath.Join(proposalDir(root), "a.json"), "{")
	forgeProposal(t, proposalDir(root), proposal{ID: "b", Op: "set", Key: "commit.threshold", Input: "4"})
	// The file that does not read is named; the others are still listed.
	if code, out, errOut := runConfig(t, "", "proposals", "--root", root); code != 1 || !strings.Contains(errOut, "proposal a") || !strings.Contains(out, "b  ") {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
	if code, out, _ := runConfig(t, "", "proposals", "--json", "--root", root); code != 1 || !strings.Contains(out, `"id": "b"`) {
		t.Fatalf("JSON: %d %q", code, out)
	}
	// A directory ending in .json is no proposal.
	root = configRoot(t, "")
	if err := os.MkdirAll(filepath.Join(proposalDir(root), "x.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := runConfig(t, "", "proposals", "--root", root); code != 0 || out != "" {
		t.Fatalf("dir: %d %q", code, out)
	}
	// A directory no system can open: a file in its place reads as absent
	// on Windows, a NUL byte fails the same everywhere.
	root = t.TempDir() + "\x00x"
	for _, args := range [][]string{{"proposals"}, {"apply", "--all"}, {"reject", "--all"}} {
		if code, _, errOut := runConfig(t, "", append(args, "--root", root)...); code != 1 || errOut == "" {
			t.Errorf("%v: %d %q", args, code, errOut)
		}
	}
}

func TestConfigListCountsOpenProposals(t *testing.T) {
	root := configRoot(t, "")
	if _, out, _ := runConfig(t, "", "list", "--root", root); strings.Contains(out, "proposals open") {
		t.Fatalf("hint without proposals:\n%s", out)
	}
	runConfig(t, "", "set", "commit.threshold", "4", "--propose", "--root", root)
	if _, out, _ := runConfig(t, "", "list", "--root", root); !strings.HasSuffix(out, "\n1 proposal open — loomux config proposals\n") {
		t.Fatalf("list with one:\n%s", out)
	}
	runConfig(t, "", "set", "commit.language", "de", "--propose", "--root", root)
	_, out, _ := runConfig(t, "", "list", "--root", root)
	if !strings.HasSuffix(out, "\n2 proposals open — loomux config proposals\n") {
		t.Fatalf("list:\n%s", out)
	}
	_, out, _ = runConfig(t, "", "list", "--json", "--root", root)
	var rows []map[string]any
	if json.Unmarshal([]byte(out), &rows) != nil || len(rows) == 0 {
		t.Fatalf("JSON changed:\n%s", out)
	}
}

func TestConfigUIShowsOpenProposalsInItsTitle(t *testing.T) {
	root, target := projectTarget(t, "")
	runConfig(t, "", "set", "commit.threshold", "4", "--propose", "--root", root)
	term := tui.Script(500, 60, tui.Keys("q")...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(term.Output(), "1 proposal open — loomux config proposals") {
		t.Fatal(term.Output())
	}
}

func TestConfigApplyWritesAndRemovesTheProposal(t *testing.T) {
	root := configRoot(t, "[commit]\nlanguage = \"en\"\n")
	runConfig(t, "", "set", "commit.language", "de", "--propose", "--root", root)
	id := strings.TrimSuffix(proposalFiles(t, proposalDir(root))[0], ".json")
	code, _, errOut := runConfig(t, "", "apply", id, "--yes", "--root", root)
	if code != 0 || !strings.Contains(errOut, "+ language = \"de\"") {
		t.Fatalf("%d %s", code, errOut)
	}
	if got := readConfig(t, root); got != "[commit]\nlanguage = \"de\"\n" {
		t.Fatalf("file:\n%s", got)
	}
	if names := proposalFiles(t, proposalDir(root)); len(names) != 0 {
		t.Fatalf("left %v", names)
	}
}

func TestConfigApplyAsksAndKeepsADeclinedProposal(t *testing.T) {
	root := configRoot(t, "")
	runConfig(t, "", "set", "commit.threshold", "4", "--propose", "--root", root)
	code, _, errOut := runConfig(t, "n\n", "apply", "--all", "--root", root)
	if code != 0 || !strings.Contains(errOut, "declined") || readConfig(t, root) != "" || len(proposalFiles(t, proposalDir(root))) != 1 {
		t.Fatalf("%d %s", code, errOut)
	}
	if code, _, errOut := runConfig(t, "yes\n", "apply", "--all", "--root", root); code != 0 || readConfig(t, root) != "[commit]\nthreshold = 4\n" {
		t.Fatalf("%d %s", code, errOut)
	}
}

// The file changed after the proposal was made: apply computes the change
// against the file as it is, so the human's later edit survives.
func TestConfigApplyRecomputesAgainstTheCurrentFile(t *testing.T) {
	root := configRoot(t, "[commit]\nlanguage = \"en\"\n")
	runConfig(t, "", "set", "commit.threshold", "4", "--propose", "--root", root)
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[commit]\nlanguage = \"de\"\n")
	if code, _, errOut := runConfig(t, "", "apply", "--all", "--yes", "--root", root); code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	if got := readConfig(t, root); got != "[commit]\nlanguage = \"de\"\nthreshold = 4\n" {
		t.Fatalf("file:\n%s", got)
	}
}

// A proposal file whose diff says something else than its op, key and input
// is applied by what it asks for, never by what it shows.
func TestConfigApplyIgnoresTheStoredDiff(t *testing.T) {
	root := configRoot(t, "")
	forgeRaw(t, proposalDir(root), "20260101T000000Z-001",
		`{"op":"set","key":"commit.threshold","input":"4","target":"global","diff":"+ [policy]\n+ evil = true\n"}`)
	code, _, errOut := runConfig(t, "", "apply", "20260101T000000Z-001", "--yes", "--root", root)
	if code != 0 || strings.Contains(errOut, "evil") {
		t.Fatalf("%d %s", code, errOut)
	}
	if got := readConfig(t, root); got != "[commit]\nthreshold = 4\n" {
		t.Fatalf("file:\n%s", got)
	}
}

// Proposals run oldest first, and one reader serves every question: two
// answers on stdin reach two proposals.
func TestConfigApplyAllGoesInIdOrder(t *testing.T) {
	root := configRoot(t, "")
	dir := proposalDir(root)
	forgeProposal(t, dir, proposal{ID: "20260101T000000Z-002", Op: "set", Key: "commit.threshold", Input: "7"})
	forgeProposal(t, dir, proposal{ID: "20260101T000000Z-001", Op: "set", Key: "commit.threshold", Input: "5"})
	code, _, errOut := runConfig(t, "y\ny\n", "apply", "--all", "--root", root)
	if code != 0 || strings.Index(errOut, "-001") > strings.Index(errOut, "-002") {
		t.Fatalf("%d %s", code, errOut)
	}
	if got := readConfig(t, root); got != "[commit]\nthreshold = 7\n" {
		t.Fatalf("file:\n%s", got)
	}
	if names := proposalFiles(t, dir); len(names) != 0 {
		t.Fatalf("left %v", names)
	}
}

// One proposal that no longer holds stays with its reason; the others are
// applied and the run ends in 1.
func TestConfigApplyAllKeepsWhatFailsAndGoesOn(t *testing.T) {
	root := configRoot(t, "")
	dir := proposalDir(root)
	forgeProposal(t, dir, proposal{ID: "a-1", Op: "set", Key: "commit.threshold", Input: "4"})
	forgeProposal(t, dir, proposal{ID: "a-2", Op: "set", Key: "commit.language", Input: "fr"})
	forgeProposal(t, dir, proposal{ID: "a-3", Op: "rename", Key: "commit.language"})
	writeFile(t, filepath.Join(dir, "a-4.json"), "{")
	forgeProposal(t, dir, proposal{ID: "a-5", Op: "set", Key: "verify.timeout", Input: "30"})
	code, _, errOut := runConfig(t, "", "apply", "--all", "--yes", "--root", root)
	for _, want := range []string{"proposal a-2 (set commit.language fr)", "unknown operation rename", "proposal a-4", "the proposal stays"} {
		if !strings.Contains(errOut, want) {
			t.Errorf("lacks %q:\n%s", want, errOut)
		}
	}
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	if got := readConfig(t, root); got != "[commit]\nthreshold = 4\n\n[verify]\ntimeout = 30\n" {
		t.Fatalf("file:\n%s", got)
	}
	if names := proposalFiles(t, dir); strings.Join(names, " ") != "a-2.json a-3.json a-4.json" {
		t.Fatalf("left %v", names)
	}
}

func TestConfigApplyOfWhatIsAlreadySoRemovesTheProposal(t *testing.T) {
	root := configRoot(t, "[commit]\nthreshold = 4\n")
	forgeProposal(t, proposalDir(root), proposal{ID: "a-1", Op: "unset", Key: "commit.language"})
	code, _, errOut := runConfig(t, "", "apply", "a-1", "--root", root)
	if code != 0 || !strings.Contains(errOut, "already so; removed") || len(proposalFiles(t, proposalDir(root))) != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestConfigApplyAndRejectOfAnUnknownProposal(t *testing.T) {
	root := configRoot(t, "")
	forgeProposal(t, proposalDir(root), proposal{ID: "a-1", Op: "unset", Key: "commit.language"})
	writeFile(t, filepath.Join(root, ".loomux", "x.json"), "{}")
	for _, sub := range []string{"apply", "reject"} {
		for _, id := range []string{"nope", "../x", "a-1.json"} {
			args := []string{sub, id, "--root", root}
			if sub == "apply" {
				args = append(args, "--yes")
			}
			if code, _, errOut := runConfig(t, "", args...); code != 1 || !strings.Contains(errOut, "no open proposal") {
				t.Errorf("%s %s: %d %q", sub, id, code, errOut)
			}
		}
	}
	if len(proposalFiles(t, proposalDir(root))) != 1 {
		t.Fatal("a proposal went away")
	}
}

func TestConfigApplyWithNothingOpen(t *testing.T) {
	root := configRoot(t, "")
	for _, sub := range []string{"apply", "reject"} {
		if code, _, errOut := runConfig(t, "", sub, "--all", "--root", root); code != 0 || !strings.Contains(errOut, "no proposals open") {
			t.Errorf("%s: %d %q", sub, code, errOut)
		}
	}
}

func TestConfigApplyReportsAFileItCannotRead(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	forgeProposal(t, proposalDir(root), proposal{ID: "a-1", Op: "set", Key: "commit.threshold", Input: "4"})
	if code, _, errOut := runConfig(t, "", "apply", "a-1", "--yes", "--root", root); code != 1 || !strings.Contains(errOut, "stays") {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestConfigRejectRemovesProposals(t *testing.T) {
	root := configRoot(t, "")
	dir := proposalDir(root)
	for _, id := range []string{"a-1", "a-2", "a-3"} {
		forgeProposal(t, dir, proposal{ID: id, Op: "unset", Key: "commit.language"})
	}
	if code, _, errOut := runConfig(t, "", "reject", "a-2", "--root", root); code != 0 || !strings.Contains(errOut, "rejected a-2") {
		t.Fatalf("%d %s", code, errOut)
	}
	if names := proposalFiles(t, dir); strings.Join(names, " ") != "a-1.json a-3.json" {
		t.Fatalf("left %v", names)
	}
	if code, _, errOut := runConfig(t, "", "reject", "--all", "--root", root); code != 0 || !strings.Contains(errOut, "rejected a-1, a-3") {
		t.Fatalf("%d %s", code, errOut)
	}
	if names := proposalFiles(t, dir); len(names) != 0 {
		t.Fatalf("left %v", names)
	}
	if got := readConfig(t, root); got != "" {
		t.Fatalf("reject wrote:\n%s", got)
	}
}

// The machine-wide file keeps its proposals beside it in the state
// directory. It knows no key yet, so a proposal for it fails like set does;
// with keys, the same functions store and apply there.
func TestConfigProposalsForTheGlobalFile(t *testing.T) {
	state := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", state)
	dir := filepath.Join(state, "config", "proposals")
	if code, _, _ := runConfig(t, "", "set", "commit.threshold", "4", "--propose", "--global"); code != 1 || len(proposalFiles(t, dir)) != 0 {
		t.Fatalf("code %d", code)
	}
	forgeProposal(t, dir, proposal{ID: "g-1", Op: "set", Key: "commit.threshold", Input: "4", Target: "global"})
	if code, out, _ := runConfig(t, "", "proposals", "--global"); code != 0 || !strings.Contains(out, "g-1") {
		t.Fatalf("%d %q", code, out)
	}
	if code, _, errOut := runConfig(t, "", "apply", "g-1", "--yes", "--global"); code != 1 || !strings.Contains(errOut, "unknown key") {
		t.Fatalf("%d %q", code, errOut)
	}
	if code, _, _ := runConfig(t, "", "reject", "g-1", "--global"); code != 0 || len(proposalFiles(t, dir)) != 0 {
		t.Fatalf("reject: %d", code)
	}

	target := configTarget{path: filepath.Join(state, "config.toml"), keys: schema.Keys(), global: true}
	var out, errOut bytes.Buffer
	if code := configPropose(target, proposal{Op: "set", Key: "commit.threshold", Input: "4"}, &out, &errOut); code != 0 {
		t.Fatalf("%d %s", code, errOut.String())
	}
	names := proposalFiles(t, dir)
	data, _ := os.ReadFile(filepath.Join(dir, names[0]))
	if len(names) != 1 || !strings.Contains(string(data), `"target": "global"`) {
		t.Fatalf("%v %s", names, data)
	}
	if code := configApply(target, "", true, true, strings.NewReader(""), &errOut); code != 0 {
		t.Fatalf("%d %s", code, errOut.String())
	}
	data, _ = os.ReadFile(filepath.Join(state, "config.toml"))
	if string(data) != "[commit]\nthreshold = 4\n" || len(proposalFiles(t, dir)) != 0 {
		t.Fatalf("global file:\n%s", data)
	}
}

// A propose flag that ends up false is refused, never read as a plain set:
// the guard let the call through because it named --propose.
func TestConfigRefusesAProposeFlagSwitchedOff(t *testing.T) {
	root := configRoot(t, "[commit]\nthreshold = 5\n")
	for _, args := range [][]string{
		{"set", "commit.threshold", "4", "--propose", "--propose=false", "--yes"},
		{"set", "commit.threshold", "4", "--propose=false", "--yes"},
		{"set", "commit.threshold", "4", "-propose=0"},
		{"unset", "commit.threshold", "-propose=false", "--yes"},
		{"set", "commit.threshold", "--yes", "--", "--propose"},
	} {
		code, _, errOut := runConfig(t, "", append(args, "--root", root)...)
		if code != 2 || !strings.Contains(errOut, "--propose given and switched off; say what you mean") {
			t.Errorf("%v: %d %q", args, code, errOut)
		}
	}
	if got := readConfig(t, root); got != "[commit]\nthreshold = 5\n" {
		t.Fatalf("written:\n%s", got)
	}
	if code, _, errOut := runConfig(t, "", "set", "commit.threshold", "4", "--propose=true", "--root", root); code != 0 || len(proposalFiles(t, proposalDir(root))) != 1 {
		t.Fatalf("=true: %d %s", code, errOut)
	}
}

func TestConfigProposalUsage(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{
		{"set", "a", "b", "--propose", "--yes"},
		{"unset", "a", "--yes", "--propose"},
		{"list", "--propose"},
		{"get", "a", "--propose"},
		{"list", "--all"},
		{"set", "a", "b", "--all"},
		{"apply"},
		{"reject"},
		{"apply", "x", "--all"},
		{"reject", "x", "y"},
		{"proposals", "x"},
	} {
		if code, _, _ := runConfig(t, "", args...); code != 2 {
			t.Errorf("%v: code %d, want 2", args, code)
		}
	}
}
