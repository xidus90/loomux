// Package mutants mutates the Go decisions of a package and reports which
// mutants its suite does not notice. It ports ultra-brain's
// tools/go_mutants.py with two changes of mechanism: a mutant reaches the
// suite through `go test -overlay` instead of being written into the tree,
// and several mutants run at once.
//
// Four families, named as the reports name them: (a1) the whole condition of
// an `if`, struck out as `true` and as `false`; (a2) each operand of a `&&`
// or `||` at the top level of the condition on its own; (a3) every
// comparison operator flipped; (a4) the condition negated, because a1 gives
// no signal at a `value, ok := x.(T)` guard.
//
// A mutant that does not compile is no mutant and is counted apart. A mutant
// the suite still passes has survived, and every survivor has to be read by
// hand: it is either a missing test or a place where the code cannot tell
// the difference.
package mutants

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// Mutant is one change: which file, which line, and what it says instead.
type Mutant struct {
	Family   string
	Path     string
	Line     int
	Was, Now string
}

// String is the script's report form, which names the file and not its path.
func (m Mutant) String() string {
	return fmt.Sprintf("(%s) %s:%d  %s  ->  %s", m.Family, filepath.Base(m.Path), m.Line, pytext.Strip(m.Was), pytext.Strip(m.Now))
}

// Outcome is what one run of a package suite came to.
type Outcome int

const (
	Passed Outcome = iota
	Failed
	BuildFailed
	TimedOut
)

// TestFunc runs the suite of pkg; an empty overlay runs the tree as it stands.
type TestFunc func(pkg, overlay string) (Outcome, error)

// goTimeout is the suite's own bound on one run. A mutant that strikes out a
// cycle guard turns a walk into an endless one, and without a bound each such
// mutant costs the toolchain's ten-minute default.
const goTimeout = "60s"

// patience is the backstop above goTimeout, the script's PATIENCE["go"]. The
// test binary's bound fires first and cleanly; this one ends a go command
// that does not come back at all.
const patience = 120 * time.Second

// GoTest runs `go test` in root with the script's flags. A go command that
// does not start is an error; a run that ends, however it ends, is an Outcome.
// Every run hangs under ctx, so cancelling it ends the go test processes a
// round has started instead of leaving them to run out the patience above
// them.
func GoTest(ctx context.Context, root string) TestFunc {
	return func(pkg, overlay string) (Outcome, error) {
		args := []string{"test"}
		if overlay != "" {
			args = append(args, "-overlay", overlay)
		}
		args = append(args, "-count=1", "-failfast", "-timeout", goTimeout, "./"+pkg+"/")
		run, cancel := context.WithTimeout(ctx, patience)
		defer cancel()
		cmd := exec.CommandContext(run, "go", args...)
		cmd.Dir = root
		cmd.WaitDelay = 5 * time.Second
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		timedOut := run.Err() != nil
		if !finishedRun(err, timedOut) {
			return 0, err
		}
		return classify(stdout.String()+stderr.String(), err == nil, timedOut), nil
	}
}

// finishedRun tells a run that came to an end from a go command that never
// ran. An exit status is a verdict, and so is a child that held the output
// pipes past WaitDelay: the command itself is over, and what it printed until
// then is what classify reads. Anything else — no go on the PATH, a working
// directory that is gone — broke before the suite could answer.
func finishedRun(err error, timedOut bool) bool {
	var exit *exec.ExitError
	return err == nil || timedOut || errors.As(err, &exit) || errors.Is(err, exec.ErrWaitDelay)
}

// classify reads a finished run the way the script's _run does: a run cut off
// by patience compiled and did not pass; "[build failed]" (compiler or vet)
// or a line opening with "# " (the compiler's package header) means the
// mutant did not build; otherwise the exit status decides.
func classify(printed string, passed, timedOut bool) Outcome {
	switch {
	case timedOut:
		return TimedOut
	case strings.Contains(printed, "[build failed]") || strings.Contains(printed, "\n# "):
		return BuildFailed
	case passed:
		return Passed
	}
	return Failed
}
