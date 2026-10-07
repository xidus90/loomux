package hooks

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/verify"
)

// godotRun is a run whose clock only the Godot probe moves: it answers
// --version after cost on that clock, notes the timeout the probe and the lane
// were each given, and counts both.
type godotRun struct {
	mu                        sync.Mutex
	now                       time.Time
	cost                      time.Duration
	probes, lanes             int
	probeTimeout, laneTimeout time.Duration
}

func newGodotRun(t *testing.T, root string, cost time.Duration) *godotRun {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "godot")
	if err := os.WriteFile(bin, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GODOT_BIN", bin)
	writeWorldFile(t, root, "project.godot", "config/features=PackedStringArray(\"4.7\")\n")
	return &godotRun{now: time.Now(), cost: cost}
}

func (g *godotRun) clock() time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.now
}

func (g *godotRun) start(s child.Spec) child.Result {
	g.mu.Lock()
	defer g.mu.Unlock()
	if s.Argv[len(s.Argv)-1] == "--version" {
		g.probes++
		g.probeTimeout = s.Timeout
		g.now = g.now.Add(g.cost)
		return child.Result{Stdout: "4.7.1.stable.official.x\n"}
	}
	g.lanes++
	g.laneTimeout = s.Timeout
	return child.Result{}
}

// plan asks the finder the run built, as planning does for a gdscript lane,
// and plans one lane that starts a tool.
func (g *godotRun) plan(root string) func(verify.Effective, verify.Request, verify.PlanEnv) ([]verify.Job, error) {
	return func(_ verify.Effective, _ verify.Request, env verify.PlanEnv) ([]verify.Job, error) {
		env.Godot(root)
		return []verify.Job{{Name: "lint/go", Kind: "lint", Stack: "go", Area: ".", Dir: root, Argvs: [][]string{{"lane"}}, After: -1}}, nil
	}
}

func (g *godotRun) edit(t *testing.T, root string, budget time.Duration, payload string) {
	t.Helper()
	plan := editPlan
	t.Cleanup(func() { editPlan = plan })
	editPlan = g.plan(root)
	env := EditEnv{Host: hosts.HostClaude, Start: g.start, Look: func(s string) (string, error) { return s, nil },
		Loomux: "loomux", Budget: budget, Now: g.clock}
	var so, se bytes.Buffer
	RunPostEdit(strings.NewReader(payload), &so, &se, root, env)
}

func (g *godotRun) stop(t *testing.T, root string, budget time.Duration) {
	t.Helper()
	plan := stopPlan
	t.Cleanup(func() { stopPlan = plan })
	stopPlan = g.plan(root)
	env := StopEnv{Start: g.start, Look: func(s string) (string, error) { return s, nil }, Loomux: "loomux", Budget: budget, Now: g.clock}
	runStop(t, root, s1, env)
}

// The probe is part of the hook's budget, not a fixed 30 s on top of it.
func TestPostEditBoundsTheGodotProbeByTheBudget(t *testing.T) {
	root := goProject(t)
	g := newGodotRun(t, root, 0)
	g.edit(t, root, 10*time.Second, filePayload(t, filepath.Join(root, "a.go")))
	if g.probes != 1 || g.probeTimeout != 10*time.Second {
		t.Fatalf("%d probes, the last with timeout %v", g.probes, g.probeTimeout)
	}
}

// What planning spent is gone from what the lanes get, and a budget planning
// used up leaves them unstarted.
func TestPostEditChargesTheGodotProbeToTheBudget(t *testing.T) {
	for cost, want := range map[time.Duration]time.Duration{40 * time.Second: 10 * time.Second, 60 * time.Second: 0} {
		root := goProject(t)
		g := newGodotRun(t, root, cost)
		g.edit(t, root, 50*time.Second, filePayload(t, filepath.Join(root, "a.go")))
		if g.laneTimeout != want || (want == 0) != (g.lanes == 0) {
			t.Errorf("probe of %v: %d lanes started, the last with timeout %v, want %v", cost, g.lanes, g.laneTimeout, want)
		}
	}
}

// One --version per binary and run: every file of a call plans with the same
// finder.
func TestPostEditAsksGodotOncePerRun(t *testing.T) {
	root := goProject(t)
	g := newGodotRun(t, root, 0)
	g.edit(t, root, DefaultBudget, agyCall(t, root, "a.go", "b.go"))
	if g.probes != 1 || g.lanes != 2 {
		t.Fatalf("%d probes, %d lanes", g.probes, g.lanes)
	}
}

func TestStopBoundsTheGodotProbeByTheBudget(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	g := newGodotRun(t, root, 0)
	g.stop(t, root, 10*time.Second)
	if g.probes != 1 || g.probeTimeout != 10*time.Second {
		t.Fatalf("%d probes, the last with timeout %v", g.probes, g.probeTimeout)
	}
}

func TestStopChargesTheGodotProbeToTheBudget(t *testing.T) {
	for cost, want := range map[time.Duration]time.Duration{200 * time.Second: 70 * time.Second, 300 * time.Second: 0} {
		root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
		g := newGodotRun(t, root, cost)
		g.stop(t, root, DefaultStopBudget)
		if g.laneTimeout != want || (want == 0) != (g.lanes == 0) {
			t.Errorf("probe of %v: %d lanes started, the last with timeout %v, want %v", cost, g.lanes, g.laneTimeout, want)
		}
	}
}
