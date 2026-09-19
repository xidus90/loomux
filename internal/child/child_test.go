package child

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// TestHelperProcess is the child every test here starts: the test binary
// itself, told by environment what to do.
func TestHelperProcess(t *testing.T) {
	switch os.Getenv("CHILD_HELPER") {
	case "":
		return
	case "echo":
		fmt.Print("out\r\nline\r")
		fmt.Fprint(os.Stderr, "err\n")
		os.Exit(3)
	case "env":
		fmt.Print(os.Getenv("PYTHONIOENCODING"), "|", os.Getenv("GIT_DIR"), "|", os.Getenv("EXTRA"))
	case "sleep":
		time.Sleep(time.Minute)
	case "badutf8":
		os.Stdout.Write([]byte{'a', 0xff, 'b'})
	}
	os.Exit(0)
}

func helper(mode string, extra ...string) Spec {
	return Spec{
		Argv: []string{os.Args[0], "-test.run=^TestHelperProcess$"},
		Env:  append([]string{"CHILD_HELPER=" + mode}, extra...),
	}
}

func TestRunSeparatesStreamsAndNormalisesNewlines(t *testing.T) {
	r := Run(helper("echo"))
	if r.Code != 3 || r.Stdout != "out\nline\n" || r.Stderr != "err\n" || r.TimedOut || r.Err != nil {
		t.Fatalf("%+v", r)
	}
}

func TestRunForcesTheEnvironmentAndStripsGit(t *testing.T) {
	t.Setenv("GIT_DIR", "/elsewhere")
	r := Run(helper("env", "EXTRA=1", "PYTHONIOENCODING=latin-1"))
	if r.Stdout != "utf-8||1" {
		t.Fatalf("%q", r.Stdout)
	}
}

func TestRunKillsAtTheDeadline(t *testing.T) {
	s := helper("sleep")
	s.Timeout = 300 * time.Millisecond
	start := time.Now()
	r := Run(s)
	if !r.TimedOut || r.Code == 0 || time.Since(start) > 10*time.Second {
		t.Fatalf("%+v after %v", r, time.Since(start))
	}
}

func TestRunReplacesInvalidUTF8(t *testing.T) {
	if r := Run(helper("badutf8")); r.Stdout != "a�b" {
		t.Fatalf("%q", r.Stdout)
	}
}

func TestRunReportsAStartFailure(t *testing.T) {
	r := Run(Spec{Argv: []string{"loomux-no-such-tool-xyz"}})
	if r.Err == nil || r.Code != -1 || !strings.Contains(r.Err.Error(), "loomux-no-such-tool-xyz") {
		t.Fatalf("%+v", r)
	}
}

// failingPipe lets the first ok calls of os.Pipe through and fails the next.
func failingPipe(t *testing.T, ok int) {
	old := pipe
	calls := 0
	pipe = func() (*os.File, *os.File, error) {
		calls++
		if calls > ok {
			return nil, nil, errors.New("no handles")
		}
		return old()
	}
	t.Cleanup(func() { pipe = old })
}

func TestRunReportsAFailedFirstPipe(t *testing.T) {
	failingPipe(t, 0)
	if r := Run(helper("echo")); r.Err == nil || r.Code != -1 {
		t.Fatalf("%+v", r)
	}
}

func TestRunReportsAFailedSecondPipe(t *testing.T) {
	failingPipe(t, 1)
	if r := Run(helper("echo")); r.Err == nil || r.Code != -1 {
		t.Fatalf("%+v", r)
	}
}

// Only the first pipe fails: the second would open, and nothing may start.
func TestRunStopsAtTheFirstPipeThatFails(t *testing.T) {
	old := pipe
	calls := 0
	pipe = func() (*os.File, *os.File, error) {
		calls++
		if calls == 1 {
			return nil, nil, errors.New("no handles")
		}
		return old()
	}
	t.Cleanup(func() { pipe = old })
	if r := Run(helper("echo")); r.Err == nil || r.Err.Error() != "no handles" || calls != 1 {
		t.Fatalf("%+v after %d pipes", r, calls)
	}
}

// A reader that has finished is never abandoned, even when the grace ran out
// before anyone looked: a select between a closed channel and an expired
// timer picks either, so the check for the end has to come first.
func TestAFinishedReaderIsNeverAbandoned(t *testing.T) {
	done := &reader{done: make(chan struct{})}
	close(done.done)
	past := time.Now().Add(-time.Second)
	for i := range 500 {
		if abandonedBy([]*reader{done, done}, past) {
			t.Fatalf("iteration %d: a finished reader was reported abandoned", i)
		}
	}
	open := &reader{done: make(chan struct{})}
	if !abandonedBy([]*reader{done, open}, past) {
		t.Fatal("a reader still open after the grace is abandoned")
	}
}
