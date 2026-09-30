package hooks

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/sessions"
	"github.com/xidus90/loomux/internal/verify"
)

const stopWorld = `
[[commit]]
message = "base"
paths = ["go.mod", "a.go", "a_test.go"]

[[commit]]
message = "second"
[commit.files]
"a.go" = "package a\n\nfunc A() int { return 22 }\n"

[worktree]
"a.go" = "package a\n\nfunc A() int { return 333 }\n"
`

// noTestWorld is stopWorld without the test file, so the go stack has a test
// lane with nothing to check.
const noTestWorld = `
[[commit]]
message = "base"
paths = ["go.mod", "a.go"]

[[commit]]
message = "second"
[commit.files]
"a.go" = "package a\n\nfunc A() int { return 22 }\n"

[worktree]
"a.go" = "package a\n\nfunc A() int { return 333 }\n"
`

// gitWorld stages a Go project with two commits and a change on top, and a
// session s1 whose base is what the state names. An empty declaration leaves
// git.toml out, which is a directory no repository covers; the names in omit
// are not written at all.
func gitWorld(t *testing.T, decl string, state string, omit ...string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod":                      "module a\n\ngo 1.25\n",
		"a.go":                        "package a\n\nfunc A() int { return 1 }\n",
		"a_test.go":                   "package a\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) { A() }\n",
		".loomux/state/hooks/s1.json": state,
		// The fake go test writes no profile, and the coverage lane checks
		// that one is there; this is a test of the gate, not of gocover.
		".loomux/config.toml": "[verify.go]\ncoverage = false\n",
	}
	if decl != "" {
		files["git.toml"] = decl
	}
	for _, name := range omit {
		delete(files, name)
	}
	for name, body := range files {
		writeWorldFile(t, root, name, body)
	}
	if err := cases.BuildGitWorld(root); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeWorldFile(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeTools answers every process from its first two words; a line it does
// not know exits 0. {loomux} check gofmt and gocover answer 0 as well: this
// is a test of the gate, not of the tools.
func fakeTools(exits map[string]int) StopEnv {
	return StopEnv{
		Start: func(s child.Spec) child.Result {
			key := strings.Join(s.Argv[:min(2, len(s.Argv))], " ")
			return child.Result{Code: exits[key], Stdout: key + "\n"}
		},
		Look:   func(name string) (string, error) { return name, nil },
		Loomux: "loomux",
		Budget: DefaultStopBudget,
		Now:    time.Now,
	}
}

// countTools counts what an environment starts, so a test can say that
// nothing ran at all.
func countTools(env StopEnv) (StopEnv, *atomic.Int32) {
	var n atomic.Int32
	inner := env.Start
	env.Start = func(s child.Spec) child.Result {
		n.Add(1)
		return inner(s)
	}
	return env, &n
}

func runStop(t *testing.T, root, payload string, env StopEnv) (int, string) {
	t.Helper()
	var errOut strings.Builder
	code := RunStop(strings.NewReader(payload), &errOut, root, "claude", env)
	return code, errOut.String()
}

const s1 = `{"session_id":"s1","hook_event_name":"Stop"}`

// greenTools answer every lane 0, redVet lets `go vet` fail.
func greenTools() StopEnv { return fakeTools(map[string]int{}) }

func redVet() StopEnv { return fakeTools(map[string]int{"go vet": 1}) }

func stateOf(t *testing.T, root string) sessions.SessionState {
	t.Helper()
	return sessions.ReadState(root, "s1")
}

func headOf(t *testing.T, root string) string {
	t.Helper()
	head, err := gitwork.Head(root)
	if err != nil {
		t.Fatal(err)
	}
	return head
}

// The payload is reported where it fails to parse. Carried on, the empty one
// left behind would fail the session check instead and blame the host for a
// field it did send.
func TestStopRefusesAPayloadThatIsNoJSON(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"blocks":0}`)
	code, se := runStop(t, root, "nope", greenTools())
	if code != ExitInternal {
		t.Fatalf("%d %q", code, se)
	}
	if strings.Contains(se, "session_id") {
		t.Fatalf("the message blames the session, not the payload: %q", se)
	}
}

func TestStopRefusesAPayloadWithoutSession(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"blocks":0}`)
	code, se := runStop(t, root, `{}`, greenTools())
	if code != ExitInternal || !strings.Contains(se, "session_id") {
		t.Fatalf("%d %q", code, se)
	}
}

func TestStopPassesAndMovesTheBase(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	tree, err := gitwork.ContentTree(root, os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	state := stateOf(t, root)
	if state.Base != headOf(t, root) || state.Green != tree || state.Blocks != 0 {
		t.Fatalf("%+v against head %s tree %s", state, headOf(t, root), tree)
	}
	// A base that resolves is no base that is gone, and a cleanup that
	// succeeds says nothing: an ordinary pass writes neither line.
	if strings.Contains(se, "is gone") || strings.Contains(se, "cleaning coverage files") {
		t.Fatalf("a quiet pass said something: %q", se)
	}
}

func TestStopHoldsARedChainAndCounts(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	base := stateOf(t, root).Base
	code, se := runStop(t, root, s1, redVet())
	if code != ExitDenied || !strings.Contains(se, "lint/go: failed") {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Blocks != 1 || state.Base != base {
		t.Fatalf("%+v against base %s", state, base)
	}
}

func TestStopShowsOnlyRedLanes(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	code, se := runStop(t, root, s1, redVet())
	if code != ExitDenied || strings.Contains(se, ": ok [") {
		t.Fatalf("%d %q", code, se)
	}
}

func TestStopGivesUpAfterThreeInARow(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"blocks":3}`)
	env, started := countTools(redVet())
	code, se := runStop(t, root, s1, env)
	if code != ExitOK || !strings.Contains(se, "gave up after 3 consecutive blocks") {
		t.Fatalf("%d %q", code, se)
	}
	// This session never had a base, and a SHA of no characters is not a
	// short SHA: the give-up line says so in words.
	if !strings.Contains(se, "base stays at no commit") {
		t.Fatalf("the give-up line spells an empty base: %q", se)
	}
	if state := stateOf(t, root); state.Blocks != 0 {
		t.Fatalf("%+v", state)
	}
	if started.Load() != 0 {
		t.Fatalf("%d tools started", started.Load())
	}
}

func TestStopResetsTheCounterWhenGreen(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":2}`)
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Blocks != 0 {
		t.Fatalf("%+v", state)
	}
}

func TestStopSkipsAGreenTree(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK {
		t.Fatalf("first run: %d %q", code, se)
	}
	env, started := countTools(redVet())
	code, se := runStop(t, root, s1, env)
	if code != ExitOK || started.Load() != 0 {
		t.Fatalf("%d %q, %d tools started", code, se, started.Load())
	}
}

func TestStopSkipsATreeEqualToTheBase(t *testing.T) {
	root := gitWorld(t, twoCommitsOnly, `{"base":"{{COMMIT:2}}","blocks":0}`)
	env, started := countTools(redVet())
	code, se := runStop(t, root, s1, env)
	if code != ExitOK || started.Load() != 0 {
		t.Fatalf("%d %q, %d tools started", code, se, started.Load())
	}
}

// twoCommitsOnly is stopWorld without the change on top: the working tree is
// the second commit's.
const twoCommitsOnly = `
[[commit]]
message = "base"
paths = ["go.mod", "a.go", "a_test.go"]

[[commit]]
message = "second"
[commit.files]
"a.go" = "package a\n\nfunc A() int { return 22 }\n"
`

func TestStopSeesAnUntrackedFile(t *testing.T) {
	root := gitWorld(t, twoCommitsOnly, `{"base":"{{COMMIT:2}}","blocks":0}`)
	writeWorldFile(t, root, "b.go", "package a\n")
	if code, se := runStop(t, root, s1, redVet()); code != ExitDenied {
		t.Fatalf("%d %q", code, se)
	}
}

func TestStopWithoutBaseMeasuresFromHead(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":null,"blocks":0}`)
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitOK || !strings.Contains(se, "no base commit") {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Base != headOf(t, root) {
		t.Fatalf("%+v against head %s: %q", state, headOf(t, root), se)
	}
}

func TestStopOutsideARepositoryRunsTheChain(t *testing.T) {
	root := gitWorld(t, "", `{"blocks":0}`)
	env, started := countTools(greenTools())
	code, se := runStop(t, root, s1, env)
	if code != ExitOK || started.Load() == 0 {
		t.Fatalf("%d %q, %d tools started", code, se, started.Load())
	}
	if state := stateOf(t, root); state.Base != "" || state.Green != "" {
		t.Fatalf("%+v", state)
	}
}

// unbornWorld is a repository git init made and nobody committed in.
const unbornWorld = "\n"

func TestStopInAnUnbornRepository(t *testing.T) {
	root := gitWorld(t, unbornWorld, `{"blocks":0}`)
	env, started := countTools(greenTools())
	code, se := runStop(t, root, s1, env)
	if code != ExitOK || started.Load() == 0 {
		t.Fatalf("%d %q, %d tools started", code, se, started.Load())
	}
	if state := stateOf(t, root); state.Base != "" {
		t.Fatalf("%+v", state)
	}
	// There is no HEAD to measure from, so neither line belongs here: a
	// warning about measuring from HEAD names something that is not there,
	// and a base that was never set is not one that is gone.
	if strings.Contains(se, "no base commit") || strings.Contains(se, "is gone") {
		t.Fatalf("an unborn repository was warned about a base: %q", se)
	}
}

func TestStopHoldsAGitFailure(t *testing.T) {
	tree := stopTree
	t.Cleanup(func() { stopTree = tree })
	stopTree = func(string, string) (string, error) { return "", errors.New("boom") }
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitDenied || !strings.Contains(se, "boom") {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Blocks != 1 {
		t.Fatalf("%+v", state)
	}
}

const goneSHA = "0000000000000000000000000000000000000001"

func TestStopMeasuresFromHeadWhenTheBaseIsGone(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"`+goneSHA+`","blocks":0}`)
	env, started := countTools(greenTools())
	code, se := runStop(t, root, s1, env)
	if code != ExitOK || !strings.Contains(se, "is gone") || started.Load() == 0 {
		t.Fatalf("%d %q, %d tools started", code, se, started.Load())
	}
	if state := stateOf(t, root); state.Base != headOf(t, root) {
		t.Fatalf("%+v against head %s", state, headOf(t, root))
	}
}

// Measuring from HEAD means HEAD's own tree, not the empty one. With a
// working tree that holds exactly what HEAD holds there is nothing since the
// base and no lane has to run; the empty tree in its place would put every
// file of the repository on the pile.
func TestStopMeasuresFromTheTreeHeadHoldsWhenTheBaseIsGone(t *testing.T) {
	root := gitWorld(t, twoCommitsOnly, `{"base":"`+goneSHA+`","blocks":0}`)
	env, started := countTools(redVet())

	code, se := runStop(t, root, s1, env)

	if code != ExitOK || !strings.Contains(se, "is gone") || started.Load() != 0 {
		t.Fatalf("%d %q, %d tools started", code, se, started.Load())
	}
}

// A base an unborn repository cannot resolve leaves the empty tree to measure
// against, so every file the working tree holds is new.
func TestStopMeasuresFromTheEmptyTreeWhenThereIsNoHead(t *testing.T) {
	root := gitWorld(t, unbornWorld, `{"base":"`+goneSHA+`","blocks":0}`)
	env, started := countTools(greenTools())
	code, se := runStop(t, root, s1, env)
	if code != ExitOK || !strings.Contains(se, "is gone") || started.Load() == 0 {
		t.Fatalf("%d %q, %d tools started", code, se, started.Load())
	}
}

func TestStopHoldsALoadErrorWhenItDeliveredFindings(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, ".loomux/config.toml", "[verify]\nnope = 1\n")
	writeFinding(t, root, "a1", "origin x is new at c")
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitDenied || !strings.Contains(se, "subagent a1: origin x is new at c") {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Blocks != 0 {
		t.Fatalf("%+v", state)
	}
}

func writeFinding(t *testing.T, root, agentID, line string) {
	t.Helper()
	if err := sessions.WriteAgent(root, "s1", agentID, sessions.AgentFile{Finding: []string{line}}); err != nil {
		t.Fatal(err)
	}
}

func TestStopMarkerLetsTheTurnEnd(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, NoVerifyMarker, "")
	env, started := countTools(redVet())
	code, se := runStop(t, root, s1, env)
	if code != ExitOK || started.Load() != 0 {
		t.Fatalf("%d %q, %d tools started", code, se, started.Load())
	}
}

func TestStopDeliversFindingsAndHolds(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeFinding(t, root, "a1", "origin x is new at c")
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitDenied || !strings.HasPrefix(se, "subagent a1: origin x is new at c\n") {
		t.Fatalf("%d %q", code, se)
	}
	if _, ok := sessions.ReadAgent(root, "s1", "a1"); ok {
		t.Fatal("the agent file is still there")
	}
	if state := stateOf(t, root); state.Blocks != 0 {
		t.Fatalf("%+v", state)
	}
}

func TestStopDeliversFindingsEvenWithTheMarker(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, NoVerifyMarker, "")
	writeFinding(t, root, "a1", "origin x is new at c")
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitDenied || !strings.Contains(se, "subagent a1: origin x is new at c") {
		t.Fatalf("%d %q", code, se)
	}
}

// An agent id that safeName spells differently reaches a path of its own; a
// directory there is a removal that fails. The turn is held, and this one
// counts: the same finding arrives again at the next turn end.
func TestStopReportsAFindingFileItCannotRemove(t *testing.T) {
	root := stuckFindingWorld(t)
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitDenied || !strings.Contains(se, "loomux hook stop: removing ") {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Blocks != 1 {
		t.Fatalf("%+v", state)
	}
}

// stuckFindingWorld holds a finding whose file no removal reaches.
func stuckFindingWorld(t *testing.T) string {
	t.Helper()
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	agents := filepath.Join(root, filepath.FromSlash(sessions.StateDir), "s1", "agents")
	writeWorldFile(t, root, sessions.StateDir+"/s1/agents/a.1.json", `{"finding":["origin x is new at c"]}`)
	if err := os.MkdirAll(filepath.Join(agents, "a1.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeWorldFile(t, root, sessions.StateDir+"/s1/agents/a1.json/keep", "")
	return root
}

// A green chain does not end that row, so the counter reaches MaxBlocks and
// the give-up rule lets a turn end rather than holding every one for ever.
// The row is a red chain's: three turns held, the fourth let go -- the
// give-up arm reads the counter before any finding is cleared, so the turn
// that lifts it to MaxBlocks still holds.
func TestStopGivesUpOnAFindingItCannotClearAway(t *testing.T) {
	root := stuckFindingWorld(t)
	for turn := 1; turn <= MaxBlocks; turn++ {
		code, se := runStop(t, root, s1, greenTools())
		if code != ExitDenied {
			t.Fatalf("turn %d: %d %q", turn, code, se)
		}
		if state := stateOf(t, root); state.Blocks != turn {
			t.Fatalf("turn %d: %+v", turn, state)
		}
	}
	base := stateOf(t, root).Base
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitOK || !strings.Contains(se, "gave up after 3 consecutive blocks") {
		t.Fatalf("%d %q", code, se)
	}
	// A base that is there is named, and named short: the give-up line is
	// read by a human, and the whole SHA says nothing more.
	if !strings.Contains(se, "base stays at "+base[:12]) || strings.Contains(se, "no commit") {
		t.Fatalf("the give-up line does not name the base %s: %q", base, se)
	}
	if state := stateOf(t, root); state.Blocks != 0 {
		t.Fatalf("%+v", state)
	}
}

// A file rewritten shorter between the delivery and the clear: what it still
// holds counts as delivered, and nothing is cut off past its end.
func TestStopClearsAFindingFileThatShrankWhileItDelivered(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	if err := sessions.WriteAgent(root, "s1", "a1", sessions.AgentFile{
		Finding: []string{"origin x is new at c", "origin y is new at d"},
	}); err != nil {
		t.Fatal(err)
	}
	read := readAgent
	t.Cleanup(func() { readAgent = read })
	readAgent = func(string, string, string) (sessions.AgentFile, bool) {
		return sessions.AgentFile{Finding: []string{"origin x is new at c"}}, true
	}

	code, se := runStop(t, root, s1, greenTools())

	if code != ExitDenied {
		t.Fatalf("%d %q", code, se)
	}
	if file, ok := sessions.ReadAgent(root, "s1", "a1"); ok {
		t.Fatalf("the file is still there: %+v", file)
	}
}

// Green takes this run's profiles away; red leaves them, because the agent
// has to be able to read what the lane measured. The clock stands still, so
// the test can name the run the gate is about to make.
func TestStopKeepsThisRunsCoverageOnlyWhenItHolds(t *testing.T) {
	for _, c := range []struct {
		name string
		env  func() StopEnv
		code int
		kept bool
	}{
		{"a pass takes them away", greenTools, ExitOK, false},
		{"a hold leaves them", redVet, ExitDenied, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
			env := c.env()
			now := time.Now()
			env.Now = func() time.Time { return now }
			profile := filepath.Join(root, ".loomux", "state", "cover",
				verify.NewRunID(now, os.Getpid())+"-go-root.out")
			writeWorldFile(t, root, ".loomux/state/cover/"+filepath.Base(profile), "mode: set\n")

			if code, se := runStop(t, root, s1, env); code != c.code {
				t.Fatalf("%d %q", code, se)
			}
			if _, err := os.Stat(profile); (err == nil) != c.kept {
				t.Fatalf("kept = %v, want %v", err == nil, c.kept)
			}
		})
	}
}

func TestStopLeavesRunningSubagentsAlone(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	if err := sessions.WriteAgent(root, "s1", "a1", sessions.AgentFile{
		Snapshot: &sessions.Snapshot{Head: "c", Remote: sessions.RemoteOK},
	}); err != nil {
		t.Fatal(err)
	}
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	if _, ok := sessions.ReadAgent(root, "s1", "a1"); !ok {
		t.Fatal("the agent file is gone")
	}
}

// A subagent that started again under the same id has a snapshot in its file
// beside the finding. The gate takes the lines and leaves the snapshot: the
// run under way must still have something to compare against at its stop.
func TestStopClearsAFindingAndKeepsALiveSnapshot(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeLiveFinding(t, root)
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitDenied || !strings.Contains(se, "subagent a1: origin x is new at c") {
		t.Fatalf("%d %q", code, se)
	}
	file, ok := sessions.ReadAgent(root, "s1", "a1")
	if !ok || len(file.Finding) != 0 {
		t.Fatalf("%+v %v", file, ok)
	}
	if file.Snapshot == nil || file.Snapshot.Head != "c" || file.Snapshot.Remote != sessions.RemoteOK {
		t.Fatalf("%+v", file.Snapshot)
	}
	if state := stateOf(t, root); state.Blocks != 0 {
		t.Fatalf("%+v", state)
	}
}

// A subagent that stops while the gate delivers appends a line the gate never
// printed. The clear drops what it delivered and keeps that tail.
func TestStopKeepsAFindingThatArrivesWhileItDelivers(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeFinding(t, root, "a1", "origin x is new at c")
	read := readAgent
	t.Cleanup(func() { readAgent = read })
	readAgent = func(root, sessionID, agentID string) (sessions.AgentFile, bool) {
		file, ok := read(root, sessionID, agentID)
		if ok {
			// The stop hook of the same agent, landing between the delivery
			// and this read.
			late := sessions.AgentFile{Finding: append(slices.Clone(file.Finding), "branch work is new at d")}
			if err := sessions.WriteAgent(root, sessionID, agentID, late); err != nil {
				t.Error(err)
			}
			return late, true
		}
		return file, ok
	}
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitDenied || strings.Contains(se, "branch work is new at d") {
		t.Fatalf("%d %q", code, se)
	}
	file, ok := sessions.ReadAgent(root, "s1", "a1")
	if !ok || !slices.Equal(file.Finding, []string{"branch work is new at d"}) {
		t.Fatalf("%+v %v", file, ok)
	}
}

// A rewrite that fails is a finding that stays, which is what a removal that
// fails is: the turn is held and the block counts.
func TestStopCountsAFindingRewriteItCannotWrite(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeLiveFinding(t, root)
	write := writeAgent
	t.Cleanup(func() { writeAgent = write })
	writeAgent = func(string, string, string, sessions.AgentFile) error { return errors.New("boom") }
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitDenied || !strings.Contains(se, "loomux hook stop: boom") {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Blocks != 1 {
		t.Fatalf("%+v", state)
	}
}

// writeLiveFinding files what a subagent running again under its old id
// leaves behind: the finding of its last stop and the snapshot of this start.
func writeLiveFinding(t *testing.T, root string) {
	t.Helper()
	if err := sessions.WriteAgent(root, "s1", "a1", sessions.AgentFile{
		Snapshot: &sessions.Snapshot{Head: "c", Remote: sessions.RemoteOK},
		Finding:  []string{"origin x is new at c"},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestStopReportsABadConfig(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, ".loomux/config.toml", "[verify]\nnope = 1\n")
	if code, se := runStop(t, root, s1, greenTools()); code != ExitInternal {
		t.Fatalf("%d %q", code, se)
	}
}

// The plan is where ImportReady lands, so the plan is where the fallback can
// be read off: a stop that was given none hands on verify's own, and one that
// was given one hands on exactly that one. Without the fallback the plan
// would be handed nil, and verify.Plan calls it in a Godot project that names
// a test command.
func TestStopHandsThePlanAnImportReady(t *testing.T) {
	var seen func(string) bool
	plan := stopPlan
	t.Cleanup(func() { stopPlan = plan })
	stopPlan = func(_ verify.Effective, _ verify.Request, env verify.PlanEnv) ([]verify.Job, error) {
		seen = env.ImportReady
		return nil, errors.New("boom")
	}

	t.Run("none given", func(t *testing.T) {
		root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
		seen = nil

		if code, se := runStop(t, root, s1, greenTools()); code != ExitInternal {
			t.Fatalf("%d %q", code, se)
		}
		if seen == nil {
			t.Fatal("the plan was handed no ImportReady at all")
		}
	})

	t.Run("one given", func(t *testing.T) {
		root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
		seen = nil
		asked := false
		env := greenTools()
		env.ImportReady = func(string) bool { asked = true; return true }

		if code, se := runStop(t, root, s1, env); code != ExitInternal {
			t.Fatalf("%d %q", code, se)
		}
		if seen == nil || !seen(root) || !asked {
			t.Fatal("the plan was handed an ImportReady the caller never gave it")
		}
	})
}

func TestStopReportsAPlanItCannotMake(t *testing.T) {
	plan := stopPlan
	t.Cleanup(func() { stopPlan = plan })
	stopPlan = func(verify.Effective, verify.Request, verify.PlanEnv) ([]verify.Job, error) {
		return nil, errors.New("boom")
	}
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitInternal || !strings.Contains(se, "boom") {
		t.Fatalf("%d %q", code, se)
	}
}

// A cover directory that cannot be made ends the gate like a broken config,
// before any tool starts.
func TestStopReportsACoverDirectoryItCannotMake(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	if err := os.RemoveAll(filepath.Join(root, ".loomux", "state")); err != nil {
		t.Fatal(err)
	}
	writeWorldFile(t, root, ".loomux/state", "")
	env, started := countTools(greenTools())
	code, se := runStop(t, root, s1, env)
	if code != ExitInternal || started.Load() != 0 {
		t.Fatalf("%d %q, %d tools started", code, se, started.Load())
	}
}

// Files left behind cost disk, not the verdict: the turn ends and the gate
// says what it could not remove.
func TestStopWarnsWhenItCannotCleanTheCoverDirectory(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	stale := filepath.Join(root, ".loomux", "state", "cover", "old")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	writeWorldFile(t, root, ".loomux/state/cover/old/x", "")
	past := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(stale, past, past); err != nil {
		t.Fatal(err)
	}
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitOK || !strings.Contains(se, "cleaning coverage files: ") {
		t.Fatalf("%d %q", code, se)
	}
}

// stuckClock answers t0 while the run is being laid out -- the run id and the
// deadline -- and an hour later from then on, so the budget is spent before
// the first process starts.
func stuckClock(t0 time.Time, settled int) func() time.Time {
	var mu sync.Mutex
	calls := 0
	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls <= settled {
			return t0
		}
		return t0.Add(time.Hour)
	}
}

func TestStopReportsABudgetThatRanOut(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	base := stateOf(t, root).Base
	env := greenTools()
	env.Budget = time.Second
	env.Now = stuckClock(time.Now(), 2)
	code, se := runStop(t, root, s1, env)
	if code != ExitInternal || !strings.Contains(se, "not everything was verified") {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Base != base {
		t.Fatalf("%+v against base %s", state, base)
	}
}

func TestStopReportsNothingVerified(t *testing.T) {
	root := gitWorld(t, noTestWorld, `{"base":"{{COMMIT:1}}","blocks":0}`, "a_test.go")
	base := stateOf(t, root).Base
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitInternal || !strings.Contains(se, "nothing to check for `test`") {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Base != base || state.Green != "" {
		t.Fatalf("%+v against base %s", state, base)
	}
}

func TestStopWithAnUnknownHost(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"blocks":0}`)
	var errOut strings.Builder
	if code := RunStop(strings.NewReader(s1), &errOut, root, "nope", greenTools()); code != ExitInternal {
		t.Fatalf("%d %q", code, errOut.String())
	}
	// The host is reported where it is parsed. Carried on, the empty host
	// would fail at the read and the name nobody knows would go unnamed.
	if !strings.Contains(errOut.String(), `"nope"`) {
		t.Fatalf("the message does not name the host: %q", errOut.String())
	}
}

func TestStopReportsAStateItCannotWrite(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	hooks := filepath.Join(root, filepath.FromSlash(sessions.StateDir))
	if err := os.RemoveAll(hooks); err != nil {
		t.Fatal(err)
	}
	writeWorldFile(t, root, sessions.StateDir, "")
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitOK || !strings.Contains(se, "loomux hook stop: creating the directory for ") {
		t.Fatalf("%d %q", code, se)
	}
}

func TestStopFindsTheBinaryItNames(t *testing.T) {
	var errOut strings.Builder
	if code := Stop(strings.NewReader(`{}`), &errOut, t.TempDir(), "claude", DefaultStopBudget); code != ExitInternal {
		t.Fatalf("%d %q", code, errOut.String())
	}
}

func TestStopFallsBackToTheCommandName(t *testing.T) {
	executable := editExecutable
	t.Cleanup(func() { editExecutable = executable })
	editExecutable = func() (string, error) { return "", errors.New("no path") }
	var errOut strings.Builder
	if code := Stop(strings.NewReader(`{}`), &errOut, t.TempDir(), "claude", DefaultStopBudget); code != ExitInternal {
		t.Fatalf("%d %q", code, errOut.String())
	}
}

// The give-up turn lets the turn end, and stderr at exit 0 reaches nobody:
// what it prints of the findings stays on disk, and the next turn end -- the
// counter back at 0 -- delivers them again and holds. The marker changes
// nothing about that; it skips the chain, not the findings.
func TestStopKeepsFindingsThroughTheGiveUpTurn(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":3}`)
	writeWorldFile(t, root, NoVerifyMarker, "")
	writeFinding(t, root, "a1", "origin x is new at c")

	code, se := runStop(t, root, s1, greenTools())
	if code != ExitOK || !strings.Contains(se, "gave up after 3 consecutive blocks") {
		t.Fatalf("give-up turn: %d %q", code, se)
	}
	if _, ok := sessions.ReadAgent(root, "s1", "a1"); !ok {
		t.Fatal("the give-up turn cleared a finding nobody read")
	}

	code, se = runStop(t, root, s1, greenTools())
	if code != ExitDenied || !strings.Contains(se, "subagent a1: origin x is new at c") {
		t.Fatalf("next turn: %d %q", code, se)
	}
	if _, ok := sessions.ReadAgent(root, "s1", "a1"); ok {
		t.Fatal("the finding was delivered and held for, but its file is still there")
	}
}

// One turn end is one block, however many reasons it has: a finding it cannot
// clear and a red chain in the same turn count once.
func TestStopCountsAStuckFindingAndARedChainOnce(t *testing.T) {
	root := stuckFindingWorld(t)
	for turn := 1; turn <= MaxBlocks; turn++ {
		if code, se := runStop(t, root, s1, redVet()); code != ExitDenied {
			t.Fatalf("turn %d: %d %q", turn, code, se)
		}
		if state := stateOf(t, root); state.Blocks != turn {
			t.Fatalf("turn %d: %+v", turn, state)
		}
	}
}

// The same for a git failure beside a stuck finding.
func TestStopCountsAStuckFindingAndAGitFailureOnce(t *testing.T) {
	root := stuckFindingWorld(t)
	defer func(old func(string, string) (string, error)) { stopTree = old }(stopTree)
	stopTree = func(string, string) (string, error) { return "", errors.New("boom") }
	if code, se := runStop(t, root, s1, greenTools()); code != ExitDenied {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Blocks != 1 {
		t.Fatalf("%+v", state)
	}
}

// A tree already found green is a green pass: it ends the row of blocks as a
// green chain does.
func TestStopResetsTheCounterOnAGreenTree(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK {
		t.Fatalf("first run: %d %q", code, se)
	}
	state := stateOf(t, root)
	state.Blocks = 2
	if err := sessions.WriteState(root, "s1", state); err != nil {
		t.Fatal(err)
	}
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Blocks != 0 {
		t.Fatalf("%+v", state)
	}
}

// A stuck finding keeps its row even across the shortcut.
func TestStopKeepsTheCounterOfAStuckFindingOnAGreenTree(t *testing.T) {
	root := stuckFindingWorld(t)
	if code, se := runStop(t, root, s1, greenTools()); code != ExitDenied {
		t.Fatalf("first run: %d %q", code, se)
	}
	if code, se := runStop(t, root, s1, greenTools()); code != ExitDenied {
		t.Fatalf("second run: %d %q", code, se)
	}
	if state := stateOf(t, root); state.Blocks != 2 {
		t.Fatalf("%+v", state)
	}
}
