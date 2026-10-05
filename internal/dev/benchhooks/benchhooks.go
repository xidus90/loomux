// Package benchhooks measures hook commands the way an editing agent
// meets them: one cold run, then n warm runs, reported as a markdown
// table. It is the harness of the 2026-09-14 baseline measurement, moved
// into the binary so a later measurement repeats the earlier one instead
// of inventing its own method.
//
// The clock and the process start are injected, so everything that
// decides a number -- the pairing of start and stop, the median of an
// even count, the mode that runs two steps at once -- is tested without
// starting anything.
package benchhooks

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// Step is one process of a case.
type Step struct {
	Argv []string `json:"argv"`
}

// Case is one measured line of the table. Mode says how its steps are
// run: "single" (the default) and "seq" run them in order inside one
// measured span, "par" starts them at the same time -- that is how an
// edit met the two old guards.
type Case struct {
	Name  string `json:"name"`
	Dir   string `json:"dir"`
	Stdin string `json:"stdin"`
	Mode  string `json:"mode"`
	Steps []Step `json:"steps"`
}

// Measure runs every case once cold and n times warm and reports each as
// a timing, the exit codes of its last run attached.
func Measure(cases []Case, n int, run func(Case, Step) (int, error), now func() time.Time) ([]benchreport.Timing, error) {
	// Every case is judged before the first process starts: a measurement
	// that breaks off halfway leaves nothing worth reporting.
	if err := validate(cases); err != nil {
		return nil, err
	}
	timings := make([]benchreport.Timing, 0, len(cases))
	for _, c := range cases {
		var cold float64
		warm := make([]float64, 0, n)
		var codes []int
		for i := 0; i <= n; i++ {
			d, exits, err := once(c, run, now)
			if err != nil {
				return nil, err
			}
			codes = exits
			if i == 0 {
				cold = benchreport.MS(d)
				continue
			}
			warm = append(warm, benchreport.MS(d))
		}
		t := benchreport.Summarize(c.Name, cold, warm)
		t.ExitCodes = codes
		timings = append(timings, t)
	}
	return timings, nil
}

// Table is the markdown the command prints: one row per case, the exit
// codes of the last run in the last column.
func Table(timings []benchreport.Timing) string {
	var b strings.Builder
	b.WriteString("| case | cold (1st run) | warm median | warm min | warm max | exit codes |\n")
	b.WriteString("|---|---:|---:|---:|---:|---|\n")
	for _, t := range timings {
		fmt.Fprintf(&b, "%s %v |\n", t.Row(), t.ExitCodes)
	}
	return b.String()
}

func validate(cases []Case) error {
	for _, c := range cases {
		if len(c.Steps) == 0 {
			return fmt.Errorf("%s: no steps", c.Name)
		}
		for i, step := range c.Steps {
			// A step with no argv names no process, and Exec reaches for
			// argv[0]: without this the measurement would panic halfway,
			// after the cases before it had already run, which is the one
			// thing this function exists to prevent.
			if len(step.Argv) == 0 {
				return fmt.Errorf("%s: step #%d names no command", c.Name, i+1)
			}
		}
		switch c.Mode {
		case "", "single", "seq", "par":
		default:
			return fmt.Errorf("%s: unknown mode %q", c.Name, c.Mode)
		}
	}
	return nil
}

// once is one measured span: the clock is read before the first step
// starts and after the last one ended, whatever the mode does in between.
func once(c Case, run func(Case, Step) (int, error), now func() time.Time) (time.Duration, []int, error) {
	codes := make([]int, len(c.Steps))
	start := now()
	var err error
	switch c.Mode {
	case "", "single", "seq":
		err = sequential(c, run, codes)
	case "par":
		err = parallel(c, run, codes)
	default:
		// Unreachable through Measure, which validates first; kept so the
		// switch stays total and tested directly.
		return 0, nil, fmt.Errorf("%s: unknown mode %q", c.Name, c.Mode)
	}
	d := now().Sub(start)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: %w", c.Name, err)
	}
	return d, codes, nil
}

func sequential(c Case, run func(Case, Step) (int, error), codes []int) error {
	for i, s := range c.Steps {
		code, err := run(c, s)
		if err != nil {
			return err
		}
		codes[i] = code
	}
	return nil
}

// parallel is the shape an edit imposes: the host starts every hook at
// once and waits for the slowest.
func parallel(c Case, run func(Case, Step) (int, error), codes []int) error {
	errs := make([]error, len(c.Steps))
	var wg sync.WaitGroup
	for i, s := range c.Steps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i], errs[i] = run(c, s)
		}()
	}
	wg.Wait()
	return firstError(errs)
}

func firstError(errs []error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// Exec is the real process start. A non-zero exit is a reading and not a
// failure -- a PreToolUse hook that refuses ends with 2, and that belongs
// in the exit-code column; only a process that never
// started aborts the measurement.
//
//coverage:exempt starts a real process (exec.Command) and opens Case.Stdin; the timing and the exit code as data are tested through the injected run
func Exec(c Case, s Step) (int, error) {
	cmd := exec.Command(s.Argv[0], s.Argv[1:]...)
	cmd.Dir = c.Dir
	if c.Stdin != "" {
		payload, err := os.Open(c.Stdin)
		if err != nil {
			return 0, err
		}
		defer payload.Close()
		cmd.Stdin = payload
	}
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return 0, err
}
