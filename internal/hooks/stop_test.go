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
// has to be able to read what the lane measured -- red in probation as well,
// though the turn ends. The clock stands still, so the test can name the run
// the gate is about to make.
func TestStopKeepsThisRunsCoverageOnlyWhenItHolds(t *testing.T) {
	for _, c := range []struct {
		name  string
		armed string
		env   func() StopEnv
		code  int
		kept  bool
	}{
		{"a pass takes them away", "", greenTools, ExitOK, false},
		{"a hold leaves them", "", redVet, ExitDenied, true},
		{"a warning in probation leaves them", probing, redVet, ExitOK, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
			if c.armed != "" {
				writeWorldFile(t, root, ".loomux/armed.toml", c.armed)
			}
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

// The built-in stop profile leaves out a kind the project has no lane for: a
// module without tests ends its turn on lint and types.
func TestStopLeavesOutAKindOfTheBuiltInProfile(t *testing.T) {
	root := gitWorld(t, noTestWorld, `{"base":"{{COMMIT:1}}","blocks":0}`, "a_test.go")
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK || strings.Contains(se, "nothing to check") {
		t.Fatalf("%d %q", code, se)
	}
}

// A stop profile the project sets names its kinds: one with nothing to check
// holds the turn.
func TestStopReportsNothingVerified(t *testing.T) {
	root := gitWorld(t, noTestWorld, `{"base":"{{COMMIT:1}}","blocks":0}`, "a_test.go")
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify.profiles]\nstop = [\"lint\", \"types\", \"test\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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

const probing = "armed = []\n"

// probationWorld is stopWorld with the file; blocks is the counter the
// session starts with.
func probationWorld(t *testing.T, armed string, blocks string) string {
	t.Helper()
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":`+blocks+`}`)
	writeWorldFile(t, root, ".loomux/armed.toml", armed)
	return root
}

// A chain red only in lanes in probation is not green and not a block: the
// turn ends, the base and the green tree stay where they are, the tree is
// remembered, and the row of blocks is over, because the turn end went
// through.
func TestStopEndsTheTurnOnAChainRedOnlyInProbation(t *testing.T) {
	root := probationWorld(t, probing, "2")
	base := stateOf(t, root).Base
	at := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	env := redVet()
	env.Now = func() time.Time { return at }
	code, se := runStop(t, root, s1, env)
	if code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	for _, want := range []string{"lint/go: failed (probation) [preset]", "go vet", "probation: lint/go@., test/go@. (warn only until a green commit arms them)\n"} {
		if !strings.Contains(se, want) {
			t.Errorf("missing %q in %q", want, se)
		}
	}
	tree, err := gitwork.ContentTree(root, os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	state := stateOf(t, root)
	if state.Base != base || state.Green != "" || state.Blocks != 0 {
		t.Fatalf("the run counted as green, or the row of blocks went on: %+v", state)
	}
	if state.Seen == nil || state.Seen.Tree != tree || state.Seen.Head != headOf(t, root) || state.Seen.Report != se {
		t.Fatalf("seen %+v, want tree %s head %s", state.Seen, tree, headOf(t, root))
	}
	// When the chain ran, and which lanes were armed then: none.
	if !state.Seen.At.Equal(at) || len(state.Seen.Armed) != 0 {
		t.Fatalf("seen at %v with %v armed, want %v and none", state.Seen.At, state.Seen.Armed, at)
	}
}

// A row of two blocks, then a turn that ends red only in probation, then a
// block: the counter reads 1, not 3. The gate gives up after three blocks in
// a row, and a turn end that went through is what ends a row.
func TestAProbationOnlyTurnEndsTheRowOfBlocks(t *testing.T) {
	root := probationWorld(t, probing, "2")
	if code, se := runStop(t, root, s1, redVet()); code != ExitOK || stateOf(t, root).Blocks != 0 {
		t.Fatalf("%d %q %+v", code, se, stateOf(t, root))
	}
	// The lane is armed now and the tree has changed: the next turn end is held.
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = [\"lint/go@.\"]\n")
	writeWorldFile(t, root, "a.go", "package a\n\nfunc A() int { return 4444 }\n")
	code, se := runStop(t, root, s1, redVet())
	if code != ExitDenied || stateOf(t, root).Blocks != 1 {
		t.Fatalf("%d %q: blocks %d, want 1", code, se, stateOf(t, root).Blocks)
	}
}

// The same holds for a turn end that only says again what it has seen: a
// counter left over from before is back at 0 afterwards.
func TestSayingTheSeenStandAgainEndsTheRowOfBlocks(t *testing.T) {
	root := probationWorld(t, probing, "0")
	runStop(t, root, s1, redVet())
	state := stateOf(t, root)
	state.Blocks = 2
	if err := sessions.WriteState(root, "s1", state); err != nil {
		t.Fatal(err)
	}
	env, started := countTools(redVet())
	if code, se := runStop(t, root, s1, env); code != ExitOK || started.Load() != 0 || stateOf(t, root).Blocks != 0 || stateOf(t, root).Seen == nil {
		t.Fatalf("%d %q, %d tools, %+v", code, se, started.Load(), stateOf(t, root))
	}
}

// A project that ignores .loomux keeps the file out of the tree: arming a
// lane there changes no tree. The stand remembers which lanes were armed,
// so the turn end after a `gate arm` runs the chain and holds. The bench's
// repository is such a project: it excludes all of .loomux.
func TestArmingBetweenTwoTurnEndsRunsTheChainAgain(t *testing.T) {
	root := probationWorld(t, probing, "0")
	before, err := gitwork.ContentTree(root, os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if code, se := runStop(t, root, s1, redVet()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = [\"lint/go@.\"]\n")
	if after, _ := gitwork.ContentTree(root, os.TempDir()); after != before {
		t.Fatalf("the world must keep the file out of the tree: %s %s", before, after)
	}
	env, started := countTools(redVet())
	code, se := runStop(t, root, s1, env)
	if code != ExitDenied || started.Load() == 0 || !strings.Contains(se, "lint/go: failed [preset]") {
		t.Fatalf("%d, %d tools, %q", code, started.Load(), se)
	}
}

// A file that is gone arms every lane, and one that does not read does too:
// neither lets the turn end on what was seen while lanes were in probation.
func TestLosingTheFileRunsTheChainAgain(t *testing.T) {
	for name, write := range map[string]func(root string){
		"gone":       func(root string) { os.Remove(filepath.Join(root, ".loomux", "armed.toml")) },
		"unreadable": func(root string) { writeWorldFile(t, root, ".loomux/armed.toml", "armed = 1\n") },
	} {
		root := probationWorld(t, probing, "0")
		runStop(t, root, s1, redVet())
		write(root)
		env, started := countTools(redVet())
		if code, se := runStop(t, root, s1, env); code != ExitDenied || started.Load() == 0 {
			t.Errorf("%s: %d, %d tools, %q", name, code, started.Load(), se)
		}
	}
}

// The nearest wrong neighbour: the same red lane, armed. It holds the turn
// and counts, with or without other lanes in probation beside it.
func TestStopStillHoldsAnArmedRedLane(t *testing.T) {
	root := probationWorld(t, "armed = [\"lint/go@.\"]\n", "1")
	code, se := runStop(t, root, s1, redVet())
	if code != ExitDenied || !strings.Contains(se, "lint/go: failed [preset]") || strings.Contains(se, "probation") {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Blocks != 2 || state.Seen != nil {
		t.Fatalf("%+v", state)
	}
}

func TestStopStartsNoToolOnATreeItHasSeen(t *testing.T) {
	root := probationWorld(t, probing, "0")
	_, first := runStop(t, root, s1, redVet())
	env, started := countTools(redVet())
	code, again := runStop(t, root, s1, env)
	if code != ExitOK || started.Load() != 0 || again != first {
		t.Fatalf("%d, %d tools, %q against %q", code, started.Load(), again, first)
	}
	// Still not green: the third turn end says it again.
	if code, third := runStop(t, root, s1, env); code != ExitOK || third != first || stateOf(t, root).Green != "" {
		t.Fatalf("%d %q", code, third)
	}
	// The same with a lane armed beside the one in probation: the stand holds
	// the lanes it was taken under, and they are the ones the file names now.
	root = probationWorld(t, "armed = [\"test/go@.\"]\n", "0")
	runStop(t, root, s1, redVet())
	if seen := stateOf(t, root).Seen; seen == nil || !slices.Equal(seen.Armed, []string{"test/go@."}) {
		t.Fatalf("seen %+v", seen)
	}
	env, started = countTools(redVet())
	if code, se := runStop(t, root, s1, env); code != ExitOK || started.Load() != 0 {
		t.Fatalf("with a lane armed: %d, %d tools, %q", code, started.Load(), se)
	}
}

func TestStopRunsTheChainAgainWhenTheTreeChanged(t *testing.T) {
	root := probationWorld(t, probing, "0")
	runStop(t, root, s1, redVet())
	writeWorldFile(t, root, "a.go", "package a\n\nfunc A() int { return 4444 }\n")
	env, started := countTools(redVet())
	if code, _ := runStop(t, root, s1, env); code != ExitOK || started.Load() == 0 {
		t.Fatalf("%d, %d tools", code, started.Load())
	}
}

// The tree alone is not the key: the graph lane judges against HEAD, so a
// commit inside the session that leaves the tree alone runs the chain again.
func TestStopRunsTheChainAgainWhenHeadMoved(t *testing.T) {
	root := probationWorld(t, probing, "0")
	runStop(t, root, s1, redVet())
	seen := stateOf(t, root).Seen
	if seen == nil {
		t.Fatal("the first turn end remembered nothing")
	}
	git(t, root, "add", "a.go")
	// The identity on the command line: the bench's repository carries none.
	git(t, root, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "--no-verify", "-m", "third")
	if tree, _ := gitwork.ContentTree(root, os.TempDir()); tree != seen.Tree || headOf(t, root) == seen.Head {
		t.Fatalf("the world must move HEAD and keep the tree: %s %s", tree, headOf(t, root))
	}
	env, started := countTools(redVet())
	if code, _ := runStop(t, root, s1, env); code != ExitOK || started.Load() == 0 {
		t.Fatalf("%d, %d tools", code, started.Load())
	}
}

// .loomux/armed.toml is part of the tree the gate measures -- everything git
// does not ignore, less .loomux/state. A colleague's pull that arms the red
// lane changes the tree, so the remembered stand does not let the turn end.
func TestStopRunsTheChainAgainWhenOnlyTheArmedFileChanged(t *testing.T) {
	root := probationWorld(t, probing, "0")
	// The test bench excludes all of .loomux; a project ignores only its state.
	writeWorldFile(t, root, ".git/info/exclude", "/git.toml\n/faketool.json\n/.loomux/state/\n")
	if code, se := runStop(t, root, s1, redVet()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	// The run itself leaves nothing in the tree; only the file moves it.
	before, err := gitwork.ContentTree(root, os.TempDir())
	if seen := stateOf(t, root).Seen; err != nil || seen == nil || before != seen.Tree {
		t.Fatalf("the run left something in the tree: %s %+v %v", before, seen, err)
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = [\"lint/go@.\"]\n")
	after, err := gitwork.ContentTree(root, os.TempDir())
	if err != nil || after == before {
		t.Fatalf("the file is not part of the tree: %s %s %v", before, after, err)
	}
	env, started := countTools(redVet())
	code, se := runStop(t, root, s1, env)
	if code != ExitDenied || started.Load() == 0 || !strings.Contains(se, "lint/go: failed [preset]") {
		t.Fatalf("%d, %d tools, %q", code, started.Load(), se)
	}
}

func TestAGreenRunForgetsWhatWasSeen(t *testing.T) {
	root := probationWorld(t, probing, "0")
	runStop(t, root, s1, redVet())
	writeWorldFile(t, root, "a.go", "package a\n\nfunc A() int { return 4444 }\n")
	code, se := runStop(t, root, s1, greenTools())
	state := stateOf(t, root)
	if code != ExitOK || state.Seen != nil || state.Base != headOf(t, root) || state.Green == "" || strings.Contains(se, "probation") {
		t.Fatalf("%d %q %+v", code, se, state)
	}
}

// Without the file the state file gains no key.
func TestStopWithoutTheFileWritesTheStateOfToday(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".loomux", "state", "hooks", "s1.json"))
	if err != nil || strings.Contains(string(raw), "seen") {
		t.Fatalf("%s %v", raw, err)
	}
}

func TestStopSaysAnUnreadableArmedFile(t *testing.T) {
	root := probationWorld(t, "<<<<<<< HEAD\n", "0")
	code, se := runStop(t, root, s1, redVet())
	if code != ExitDenied || !strings.Contains(se, "loomux hook stop: .loomux/armed.toml is no TOML") || !strings.Contains(se, "every lane is armed") {
		t.Fatalf("%d %q", code, se)
	}
}

// Outside a repository there is no tree to remember: the chain runs at every
// turn end, as it does today.
func TestStopRemembersNothingOutsideARepository(t *testing.T) {
	root := gitWorld(t, "", `{"blocks":0}`)
	writeWorldFile(t, root, ".loomux/armed.toml", probing)
	if code, se := runStop(t, root, s1, redVet()); code != ExitOK || stateOf(t, root).Seen != nil {
		t.Fatalf("%d %q %+v", code, se, stateOf(t, root).Seen)
	}
	env, started := countTools(redVet())
	if code, _ := runStop(t, root, s1, env); code != ExitOK || started.Load() == 0 {
		t.Fatalf("%d, %d tools", code, started.Load())
	}
}

// The verdict's order, on outcomes laid out by hand: an armed red lane holds
// whatever else happened; a budget that ran out and a kind with nothing to
// check leave the turn unjudged, and then nothing is remembered, whatever
// stands in probation beside them.
func TestTheVerdictRemembersOnlyAChainItJudgedWhole(t *testing.T) {
	lane := func(kind string, s verify.State, probation bool) verify.Outcome {
		return verify.Outcome{Job: verify.Job{Name: kind + "/go", Kind: kind, Stack: "go", Area: ".", Origin: "preset"},
			State: s, Output: kind + " said\n", Probation: probation}
	}
	none := func(verify.Job) bool { return false }
	for name, c := range map[string]struct {
		strict bool
		kinds  []string
		outs   []verify.Outcome
		code   int
		warned bool
	}{
		"red only in probation":                {true, []string{"lint"}, []verify.Outcome{lane("lint", verify.StateFailed, true)}, ExitOK, true},
		"an armed red lane beside it":          {true, []string{"lint", "test"}, []verify.Outcome{lane("lint", verify.StateFailed, true), lane("test", verify.StateFailed, false)}, ExitDenied, false},
		"a budget that ran out beside it":      {true, []string{"lint", "test"}, []verify.Outcome{lane("lint", verify.StateFailed, true), lane("test", verify.StateBudget, true)}, ExitInternal, false},
		"a kind with nothing to check":         {true, []string{"lint", "test"}, []verify.Outcome{lane("lint", verify.StateFailed, true)}, ExitInternal, false},
		"all green in probation":               {true, []string{"lint"}, []verify.Outcome{lane("lint", verify.StateOK, true)}, ExitOK, false},
		"a named kind with nothing to check":   {true, []string{"lint", "test"}, []verify.Outcome{lane("lint", verify.StateOK, false)}, ExitInternal, false},
		"a default kind with nothing to check": {false, []string{"lint", "test"}, []verify.Outcome{lane("lint", verify.StateOK, false)}, ExitOK, false},
		"a default profile where nothing ran":  {false, []string{"types", "test"}, []verify.Outcome{lane("lint", verify.StateOK, false)}, ExitInternal, false},
	} {
		var se strings.Builder
		code, warned := stopVerdict(&se, c.kinds, c.strict, c.outs, none)
		if code != c.code || (warned != "") != c.warned {
			t.Errorf("%s: code %d, warned %q, stderr %q", name, code, warned, se.String())
		}
		// Held by an armed lane: only that lane is said. What stands in
		// probation beside it is not the agent's to fix before the turn ends.
		if c.code == ExitDenied && (!strings.Contains(se.String(), "test/go: failed [preset]") || strings.Contains(se.String(), "lint/go") || strings.Contains(se.String(), "probation")) {
			t.Errorf("%s: stderr %q", name, se.String())
		}
		if c.warned && (warned != se.String() || !strings.Contains(warned, "lint/go: failed (probation) [preset]") || !strings.Contains(warned, "probation: lint/go@. (")) {
			t.Errorf("%s: warned %q, stderr %q", name, warned, se.String())
		}
	}
}
