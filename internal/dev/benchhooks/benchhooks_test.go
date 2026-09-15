package benchhooks

import (
	"bytes"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// clock hands out the ticks of a measurement in order, so a table can be
// asserted on to the tenth of a millisecond.
func clock(ticks ...time.Duration) func() time.Time {
	i := 0
	start := time.Unix(0, 0)
	return func() time.Time {
		d := ticks[i] * time.Millisecond
		i++
		return start.Add(d)
	}
}

func TestRunReportsColdAndTheWarmMedian(t *testing.T) {
	ticks := []time.Duration{0, 10, 10, 13, 13, 15, 15, 17} // start/stop pairs: cold 10ms, warm 3, 2, 2
	var out bytes.Buffer
	c := Case{Name: "probe", Mode: "single", Steps: []Step{{Argv: []string{"x"}}}}
	err := Run([]Case{c}, 3, &out, func(Case, Step) (int, error) { return 0, nil }, clock(ticks...))
	if err != nil || !strings.Contains(out.String(), "| probe | 10.0 ms | 2.0 ms | 2.0 ms | 3.0 ms | [0] |") {
		t.Fatalf("%v\n%s", err, out.String())
	}
}

func TestRunWritesTheHeader(t *testing.T) {
	var out bytes.Buffer
	c := Case{Name: "probe", Steps: []Step{{Argv: []string{"x"}}}}
	if err := Run([]Case{c}, 1, &out, func(Case, Step) (int, error) { return 0, nil }, clock(0, 1, 1, 2)); err != nil {
		t.Fatal(err)
	}
	const header = "| case | cold (1st run) | warm median | warm min | warm max | exit codes |"
	if !strings.Contains(out.String(), header) {
		t.Fatalf("no header:\n%s", out.String())
	}
}

func TestSeqRunsBothStepsInOrder(t *testing.T) {
	var seen []string
	var out bytes.Buffer
	c := Case{Name: "seq", Mode: "seq", Steps: []Step{{Argv: []string{"a"}}, {Argv: []string{"b"}}}}
	err := Run([]Case{c}, 1, &out, func(_ Case, s Step) (int, error) {
		seen = append(seen, s.Argv[0])
		return len(s.Argv[0]), nil
	}, clock(0, 4, 4, 6))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(seen, ",") != "a,b,a,b" {
		t.Fatalf("order %v", seen)
	}
	if !strings.Contains(out.String(), "| [1 1] |") {
		t.Fatalf("exit codes:\n%s", out.String())
	}
}

func TestParStartsTheStepsAtTheSameTime(t *testing.T) {
	var arrived atomic.Int32
	both := make(chan struct{})
	var out bytes.Buffer
	c := Case{Name: "par", Mode: "par", Steps: []Step{{Argv: []string{"a"}}, {Argv: []string{"b"}}}}
	err := Run([]Case{c}, 0, &out, func(_ Case, s Step) (int, error) {
		if arrived.Add(1) == 2 {
			close(both)
		}
		// A sequential regression never reaches two, so the rendezvous
		// fails with a reading instead of hanging until the timeout.
		select {
		case <-both:
		case <-time.After(5 * time.Second):
			return 0, errors.New("steps did not overlap")
		}
		return 0, nil
	}, clock(0, 9))
	if err != nil || !strings.Contains(out.String(), "| par | 9.0 ms |") {
		t.Fatalf("%v\n%s", err, out.String())
	}
}

func TestAnEvenNumberOfWarmRunsAveragesTheMiddle(t *testing.T) {
	var out bytes.Buffer
	c := Case{Name: "even", Steps: []Step{{Argv: []string{"x"}}}}
	// warm runs: 5, 2, 4, 1 -> sorted 1, 2, 4, 5 -> median 3.0
	err := Run([]Case{c}, 4, &out, func(Case, Step) (int, error) { return 0, nil },
		clock(0, 8, 0, 5, 0, 2, 0, 4, 0, 1))
	if err != nil || !strings.Contains(out.String(), "| even | 8.0 ms | 3.0 ms | 1.0 ms | 5.0 ms | [0] |") {
		t.Fatalf("%v\n%s", err, out.String())
	}
}

func TestAFailingExecStopsTheRun(t *testing.T) {
	var out bytes.Buffer
	c := Case{Name: "broken", Steps: []Step{{Argv: []string{"x"}}}}
	err := Run([]Case{c}, 1, &out, func(Case, Step) (int, error) {
		return 0, errors.New("no such binary")
	}, clock(0, 1, 1, 2))
	if err == nil || !strings.Contains(err.Error(), "broken: no such binary") {
		t.Fatalf("err %v", err)
	}
}

func TestAFailingParStepStopsTheRun(t *testing.T) {
	var out bytes.Buffer
	c := Case{Name: "broken", Mode: "par", Steps: []Step{{Argv: []string{"x"}}}}
	err := Run([]Case{c}, 0, &out, func(Case, Step) (int, error) {
		return 0, errors.New("no such binary")
	}, clock(0, 1))
	if err == nil || !strings.Contains(err.Error(), "broken: no such binary") {
		t.Fatalf("err %v", err)
	}
}

func TestRunRefusesACaseWithoutSteps(t *testing.T) {
	var out bytes.Buffer
	err := Run([]Case{{Name: "empty"}}, 1, &out,
		func(Case, Step) (int, error) { return 0, nil }, clock(0))
	if err == nil || !strings.Contains(err.Error(), "empty: no steps") {
		t.Fatalf("err %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("wrote a table anyway:\n%s", out.String())
	}
}

// A step with no argv names no process. validate let it through, so Run wrote
// the header and then panicked in Exec on argv[0] -- exactly what validate's
// own comment says cannot happen.
func TestRunRefusesAStepWithoutAnArgv(t *testing.T) {
	var out bytes.Buffer
	err := Run([]Case{{Name: "silent", Steps: []Step{{Argv: []string{"x"}}, {}}}}, 1, &out,
		func(Case, Step) (int, error) { return 0, nil }, clock(0))
	if err == nil || !strings.Contains(err.Error(), "silent: step #2 names no command") {
		t.Fatalf("err %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("wrote a table anyway:\n%s", out.String())
	}
}

func TestRunRefusesAnUnknownMode(t *testing.T) {
	var out bytes.Buffer
	c := Case{Name: "odd", Mode: "diagonal", Steps: []Step{{Argv: []string{"x"}}}}
	err := Run([]Case{c}, 1, &out,
		func(Case, Step) (int, error) { return 0, nil }, clock(0, 1))
	if err == nil || !strings.Contains(err.Error(), `odd: unknown mode "diagonal"`) {
		t.Fatalf("err %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("wrote a table anyway:\n%s", out.String())
	}
}

// A broken case behind a good one must not leave the good one's header and
// row on stdout beside the error.
func TestRunWritesNothingWhenALaterCaseIsInvalid(t *testing.T) {
	var out bytes.Buffer
	good := Case{Name: "good", Steps: []Step{{Argv: []string{"x"}}}}
	bad := Case{Name: "bad", Mode: "diagonal", Steps: []Step{{Argv: []string{"x"}}}}
	err := Run([]Case{good, bad}, 1, &out,
		func(Case, Step) (int, error) { return 0, nil }, clock(0, 1, 1, 2))
	if err == nil || !strings.Contains(err.Error(), `bad: unknown mode "diagonal"`) {
		t.Fatalf("err %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("wrote a table anyway:\n%s", out.String())
	}
}

// once keeps the mode switch total although Run validates ahead of it.
func TestOnceRefusesAnUnknownMode(t *testing.T) {
	c := Case{Name: "odd", Mode: "diagonal", Steps: []Step{{Argv: []string{"x"}}}}
	_, _, err := once(c, func(Case, Step) (int, error) { return 0, nil }, clock(0))
	if err == nil || !strings.Contains(err.Error(), `odd: unknown mode "diagonal"`) {
		t.Fatalf("err %v", err)
	}
}
