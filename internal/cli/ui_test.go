package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/serve"
	"github.com/xidus90/loomux/internal/web"
)

// uiWorld replaces every seam of `loomux ui`: nothing is started, killed or
// opened, and every call is remembered.
type uiWorld struct {
	stateDir  string
	stopped   bool
	stopErr   error
	stops     int
	detachErr error
	spec      serve.DetachSpec
	input     string
	detaches  int
	spawner   func(*exec.Cmd) error
	state     *web.State
	waitErr   error
	opened    []string
	openErr   error
	runOpts   web.Options
	runErr    error
	runs      int
	released  bool
	order     []string
	// spawnErr is what starting the child answers; waitedFor is the PID the
	// start waited for.
	spawnErr  error
	waitedFor int
}

// uiChild is the PID the fake start gives the child.
const uiChild = 4242

func newUIWorld(t *testing.T) *uiWorld {
	t.Helper()
	w := &uiWorld{stateDir: serveStateDir(t), state: &web.State{PID: 7, Port: 7411}}
	savedRun, savedStop, savedWait, savedDetach, savedOpen, savedStart := uiRun, uiStop, uiWait, uiDetach, openBrowser, startProcess
	uiStop = func(string) (bool, error) {
		w.stops++
		w.order = append(w.order, "stop")
		return w.stopped, w.stopErr
	}
	uiDetach = func(spec serve.DetachSpec, spawner func(*exec.Cmd) error) (bool, error) {
		w.detaches++
		w.spec, w.spawner = spec, spawner
		w.order = append(w.order, "detach")
		if spec.Stdin != nil {
			data, _ := io.ReadAll(spec.Stdin)
			w.input = string(data)
		}
		// The real Detach hands the spawner a command and returns its error.
		err := w.detachErr
		if err == nil {
			err = spawner(&exec.Cmd{})
		}
		return true, err
	}
	startProcess = func(cmd *exec.Cmd) error {
		cmd.Process = &os.Process{Pid: uiChild}
		return w.spawnErr
	}
	uiWait = func(_ string, pid int, _ time.Duration) (*web.State, error) {
		w.order = append(w.order, "wait")
		w.waitedFor = pid
		return w.state, w.waitErr
	}
	openBrowser = func(url string) error {
		w.opened = append(w.opened, url)
		return w.openErr
	}
	uiRun = func(_ context.Context, o web.Options) error {
		w.runs++
		w.runOpts = o
		return w.runErr
	}
	t.Cleanup(func() {
		uiRun, uiStop, uiWait, uiDetach, openBrowser, startProcess = savedRun, savedStop, savedWait, savedDetach, savedOpen, savedStart
	})
	return w
}

func runUI(args ...string) (code int, stdout, stderr string) {
	return runUIWith("", args...)
}

func runUIWith(stdin string, args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = uiCommand(args, strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

func (w *uiWorld) loginCode(t *testing.T) string {
	t.Helper()
	line, rest, _ := strings.Cut(w.input, "\n")
	if rest != "" || line == "" {
		t.Fatalf("the child was handed %q, want exactly one line", w.input)
	}
	return line
}

func TestUITheCommandIsRegistered(t *testing.T) {
	if _, ok := commands["ui"]; !ok {
		t.Fatal("ui is not a command")
	}
}

func TestUIStartsADetachedInterfaceAndOpensTheBrowserOnTheLogin(t *testing.T) {
	w := newUIWorld(t)
	code, stdout, stderr := runUI()
	if code != 0 || stderr != "" {
		t.Fatalf("ui = %d, %q", code, stderr)
	}
	if strings.Join(w.spec.Args, " ") != "ui --foreground" || w.spec.StateDir != w.stateDir || w.spec.LogPath != web.LogPath(w.stateDir) ||
		!w.spec.Console {
		t.Errorf("the child is started as %+v", w.spec)
	}
	secret := w.loginCode(t)
	if len(secret) < 40 {
		t.Errorf("the login code %q is too short to be random", secret)
	}
	if len(w.opened) != 1 || w.opened[0] != "http://127.0.0.1:7411/login?code="+secret {
		t.Errorf("the browser was opened on %v", w.opened)
	}
	// The address that stays on the terminal carries no code.
	if strings.Contains(stdout, secret) || !strings.Contains(stdout, "http://127.0.0.1:7411/") {
		t.Errorf("stdout %q must name the interface without the code", stdout)
	}
	if w.spawner == nil {
		t.Error("the start was handed no spawner")
	}
}

func TestUIStartsTheChildOnlyAfterEndingTheOldOneAndWaitsForIt(t *testing.T) {
	w := newUIWorld(t)
	w.stopped = true
	runUI()
	if got := strings.Join(w.order, ","); got != "stop,detach,wait" {
		t.Errorf("the order is %s, want stop, detach, wait", got)
	}
}

// The login code goes to the child this start spawned, so the wait names its
// PID; a child that did not start is waited for as no process at all.
func TestUIWaitsForTheChildItSpawned(t *testing.T) {
	w := newUIWorld(t)
	runUI()
	if w.waitedFor != uiChild {
		t.Errorf("waited for PID %d, want the child's %d", w.waitedFor, uiChild)
	}
	w = newUIWorld(t)
	w.spawnErr = errors.New("no")
	if code, _, stderr := runUI(); code != 1 || !strings.Contains(stderr, "no") || w.waitedFor != 0 {
		t.Errorf("a child that did not start: %d, %q, waited for %d", code, stderr, w.waitedFor)
	}
}

func TestUIDrawsANewCodeEveryTime(t *testing.T) {
	w := newUIWorld(t)
	runUI()
	first := w.loginCode(t)
	runUI()
	if second := w.loginCode(t); first == second {
		t.Error("two starts used the same login code")
	}
}

func TestUINoBrowserPrintsTheLoginAddress(t *testing.T) {
	w := newUIWorld(t)
	code, stdout, _ := runUI("--no-browser")
	if code != 0 || len(w.opened) != 0 {
		t.Fatalf("ui --no-browser = %d, opened %v", code, w.opened)
	}
	if want := "http://127.0.0.1:7411/login?code=" + w.loginCode(t); !strings.Contains(stdout, want) {
		t.Errorf("stdout %q does not print %s", stdout, want)
	}
}

func TestUIWhenTheBrowserWillNotOpenItSaysWhereToGo(t *testing.T) {
	w := newUIWorld(t)
	w.openErr = errors.New("no handler")
	code, stdout, stderr := runUI()
	if code != 0 || !strings.Contains(stderr, "no handler") ||
		!strings.Contains(stdout, "http://127.0.0.1:7411/login?code="+w.loginCode(t)) {
		t.Errorf("ui = %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestUIFailsWhereTheStartFails(t *testing.T) {
	for name, set := range map[string]func(*uiWorld){
		"the old one cannot be ended": func(w *uiWorld) { w.stopErr = errors.New("access is denied") },
		"the child cannot be started": func(w *uiWorld) { w.detachErr = errors.New("no such program") },
		"the child never answers":     func(w *uiWorld) { w.waitErr = errors.New("did not answer") },
	} {
		w := newUIWorld(t)
		set(w)
		code, stdout, stderr := runUI()
		if code != 1 || !strings.HasPrefix(stderr, "loomux ui: ") || len(w.opened) != 0 || stdout != "" {
			t.Errorf("%s: ui = %d, stdout %q, stderr %q, opened %v", name, code, stdout, stderr, w.opened)
		}
	}
}

func TestUIStopsWhatRuns(t *testing.T) {
	w := newUIWorld(t)
	w.stopped = true
	code, stdout, _ := runUI("stop")
	if code != 0 || !strings.Contains(stdout, "stopped") || w.stops != 1 || w.detaches != 0 {
		t.Errorf("ui stop = %d %q, stops %d, detaches %d", code, stdout, w.stops, w.detaches)
	}
}

func TestUIStopWithNothingRunningIsNoFailure(t *testing.T) {
	newUIWorld(t)
	code, stdout, stderr := runUI("stop")
	if code != 0 || !strings.Contains(stdout, "not running") || stderr != "" {
		t.Errorf("ui stop = %d, %q, %q", code, stdout, stderr)
	}
}

func TestUIStopReportsAProcessItCannotEnd(t *testing.T) {
	w := newUIWorld(t)
	w.stopErr = errors.New("access is denied")
	code, _, stderr := runUI("stop")
	if code != 1 || !strings.Contains(stderr, "access is denied") {
		t.Errorf("ui stop = %d, %q", code, stderr)
	}
}

func TestUIRefusesWhatItDoesNotKnow(t *testing.T) {
	w := newUIWorld(t)
	for _, args := range [][]string{
		{"start"}, {"stop", "--force"}, {"--no-browsr"}, {"--foreground", "--no-browser"}, {"stop", "now"}, {"--no-browser", "stop"},
	} {
		code, stdout, stderr := runUI(args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, "usage: loomux ui") || !strings.Contains(stderr, "unrecognized arguments") {
			t.Errorf("ui %v = %d, %q, %q, want usage and exit 2", args, code, stdout, stderr)
		}
	}
	if w.stops != 0 || w.detaches != 0 || w.runs != 0 {
		t.Error("an unknown argument started or stopped something")
	}
}

func TestUIHelpIsUsage(t *testing.T) {
	newUIWorld(t)
	for _, flag := range []string{"--help", "-h"} {
		code, stdout, _ := runUI(flag)
		if code != 0 || !strings.Contains(stdout, "usage: loomux ui") {
			t.Errorf("ui %s = %d, %q", flag, code, stdout)
		}
	}
}

func TestUIForegroundServesWithTheCodeFromStdin(t *testing.T) {
	w := newUIWorld(t)
	released := stubServeNotify(t)
	code, _, stderr := runUIWith("the-code\n", "--foreground")
	if code != 0 || stderr != "" || w.runs != 1 {
		t.Fatalf("ui --foreground = %d, %q, runs %d", code, stderr, w.runs)
	}
	if w.runOpts.Code != "the-code" || w.runOpts.StateDir != config.StateDir() || w.runOpts.Version != bareVersion() {
		t.Errorf("the interface runs with %+v", w.runOpts)
	}
	if w.runOpts.SetupDeps == nil {
		t.Error("the interface was given no way to run a setup")
	}
	if w.runOpts.Gather == nil {
		t.Error("the interface was given no way to read a project")
	}
	if !*released {
		t.Error("the interrupt registration was never given up")
	}
}

func TestUIForegroundTakesOnlyTheFirstLineAndTrimsIt(t *testing.T) {
	w := newUIWorld(t)
	stubServeNotify(t)
	runUIWith("  code-1 \r\nsecond line\n", "--foreground")
	if w.runOpts.Code != "code-1" {
		t.Errorf("the code is %q, want the first line trimmed", w.runOpts.Code)
	}
}

func TestUIForegroundWithoutACodeRefuses(t *testing.T) {
	w := newUIWorld(t)
	for _, stdin := range []string{"", "\n", "   \n"} {
		code, _, stderr := runUIWith(stdin, "--foreground")
		if code != 1 || !strings.Contains(stderr, "no login code") || w.runs != 0 {
			t.Errorf("stdin %q: ui --foreground = %d, %q, runs %d", stdin, code, stderr, w.runs)
		}
	}
}

func TestUIForegroundReportsWhatEndsTheInterface(t *testing.T) {
	w := newUIWorld(t)
	stubServeNotify(t)
	w.runErr = errors.New("listen: no ports")
	code, _, stderr := runUIWith("c\n", "--foreground")
	if code != 1 || !strings.Contains(stderr, "no ports") {
		t.Errorf("ui --foreground = %d, %q", code, stderr)
	}
}

// The dashboard's search state is the line set `brain status` prints, asked of
// the state directory it is given.
func TestUIGivesTheDashboardTheLinesOfBrainStatus(t *testing.T) {
	w := newUIWorld(t)
	stubServeNotify(t)
	runUIWith("the-code\n", "--foreground")
	writeFile(t, filepath.Join(w.stateDir, "registry.toml"), "")
	stubBrainStatusPort(t, search.NewFakePort())
	out, err := w.runOpts.SearchStatus(w.stateDir)
	if err != nil || !strings.HasPrefix(out, "last reconcile: never;") {
		t.Fatalf("SearchStatus = %q, %v", out, err)
	}
}
