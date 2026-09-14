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
	"io"
	"os"
	"os/exec"
	"slices"
	"sync"
	"time"
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

// Run measures every case and writes the table to w.
func Run(cases []Case, n int, w io.Writer, run func(Case, Step) (int, error), now func() time.Time) error {
	// Every case is judged before the first byte is written: a table whose
	// header stands above nothing is worse than no table.
	if err := validate(cases); err != nil {
		return err
	}
	fmt.Fprintln(w, "| case | cold (1st run) | warm median | warm min | warm max | exit codes |")
	fmt.Fprintln(w, "|---|---:|---:|---:|---:|---|")
	for _, c := range cases {
		var cold time.Duration
		warm := make([]time.Duration, 0, n)
		var codes []int
		for i := 0; i <= n; i++ {
			d, exits, err := once(c, run, now)
			if err != nil {
				return err
			}
			codes = exits
			if i == 0 {
				cold = d
				continue
			}
			warm = append(warm, d)
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %v |\n", c.Name,
			ms(cold), ms(median(warm)), ms(low(warm)), ms(high(warm)), codes)
	}
	return nil
}

func validate(cases []Case) error {
	for _, c := range cases {
		if len(c.Steps) == 0 {
			return fmt.Errorf("%s: no steps", c.Name)
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
		// Unreachable through Run, which validates first; kept so the
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

// median averages the two middle values of an even count, so a run of
// four does not silently report the upper one as the middle.
func median(d []time.Duration) time.Duration {
	if len(d) == 0 {
		return 0
	}
	s := slices.Clone(d)
	slices.Sort(s)
	half := len(s) / 2
	if len(s)%2 == 1 {
		return s[half]
	}
	return (s[half-1] + s[half]) / 2
}

func low(d []time.Duration) time.Duration {
	if len(d) == 0 {
		return 0
	}
	return slices.Min(d)
}

func high(d []time.Duration) time.Duration {
	if len(d) == 0 {
		return 0
	}
	return slices.Max(d)
}

func ms(d time.Duration) string {
	return fmt.Sprintf("%.1f ms", float64(d)/float64(time.Millisecond))
}

// Exec is the real process start. A non-zero exit is a reading and not a
// failure -- `brain guard` ends with 2 on a repository it does not know,
// and that belongs in the exit-code column; only a process that never
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
