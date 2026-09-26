package benchhooks

import (
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/dev/benchreport"
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

func exitZero(Case, Step) (int, error) { return 0, nil }

// table is the command's path: measure, then render.
func table(cases []Case, n int, run func(Case, Step) (int, error), now func() time.Time) (string, error) {
	timings, err := Measure(cases, n, run, now)
	if err != nil {
		return "", err
	}
	return Table(timings), nil
}

func TestMeasureReportsEveryCaseAsATiming(t *testing.T) {
	// start/stop pairs: cold 10ms, warm 2, 4, 3
	got, err := Measure([]Case{{Name: "guard", Steps: []Step{{Argv: []string{"x"}}}}}, 3, exitZero,
		clock(0, 10, 0, 2, 0, 4, 0, 3))
	if err != nil {
		t.Fatal(err)
	}
	want := []benchreport.Timing{{Name: "guard", ColdMS: 10, WarmMS: []float64{2, 4, 3},
		MedianMS: 3, MinMS: 2, MaxMS: 4, ExitCodes: []int{0}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestTableIsTheHeaderAndOneRowPerTiming(t *testing.T) {
	got := Table([]benchreport.Timing{
		{Name: "a", ColdMS: 10, MedianMS: 2.5, MinMS: 1, MaxMS: 3, ExitCodes: []int{0, 2}},
		{Name: "b", ColdMS: 1, MedianMS: 1, MinMS: 1, MaxMS: 1},
	})
	want := "| case | cold (1st run) | warm median | warm min | warm max | exit codes |\n" +
		"|---|---:|---:|---:|---:|---|\n" +
		"| a | 10.0 ms | 2.5 ms | 1.0 ms | 3.0 ms | [0 2] |\n" +
		"| b | 1.0 ms | 1.0 ms | 1.0 ms | 1.0 ms | [] |\n"
	if got != want {
		t.Fatalf("got:\n%s", got)
	}
}

func TestMeasureReportsColdAndTheWarmMedian(t *testing.T) {
	ticks := []time.Duration{0, 10, 10, 13, 13, 15, 15, 17} // start/stop pairs: cold 10ms, warm 3, 2, 2
	c := Case{Name: "probe", Mode: "single", Steps: []Step{{Argv: []string{"x"}}}}
	out, err := table([]Case{c}, 3, exitZero, clock(ticks...))
	if err != nil || !strings.Contains(out, "| probe | 10.0 ms | 2.0 ms | 2.0 ms | 3.0 ms | [0] |") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestTableWritesTheHeader(t *testing.T) {
	c := Case{Name: "probe", Steps: []Step{{Argv: []string{"x"}}}}
	out, err := table([]Case{c}, 1, exitZero, clock(0, 1, 1, 2))
	if err != nil {
		t.Fatal(err)
	}
	const header = "| case | cold (1st run) | warm median | warm min | warm max | exit codes |"
	if !strings.Contains(out, header) {
		t.Fatalf("no header:\n%s", out)
	}
}

func TestSeqRunsBothStepsInOrder(t *testing.T) {
	var seen []string
	c := Case{Name: "seq", Mode: "seq", Steps: []Step{{Argv: []string{"a"}}, {Argv: []string{"b"}}}}
	out, err := table([]Case{c}, 1, func(_ Case, s Step) (int, error) {
		seen = append(seen, s.Argv[0])
		return len(s.Argv[0]), nil
	}, clock(0, 4, 4, 6))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(seen, ",") != "a,b,a,b" {
		t.Fatalf("order %v", seen)
	}
	if !strings.Contains(out, "| [1 1] |") {
		t.Fatalf("exit codes:\n%s", out)
	}
}

func TestParStartsTheStepsAtTheSameTime(t *testing.T) {
	var arrived atomic.Int32
	both := make(chan struct{})
	c := Case{Name: "par", Mode: "par", Steps: []Step{{Argv: []string{"a"}}, {Argv: []string{"b"}}}}
	out, err := table([]Case{c}, 0, func(_ Case, s Step) (int, error) {
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
	if err != nil || !strings.Contains(out, "| par | 9.0 ms |") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestAnEvenNumberOfWarmRunsAveragesTheMiddle(t *testing.T) {
	c := Case{Name: "even", Steps: []Step{{Argv: []string{"x"}}}}
	// warm runs: 5, 2, 4, 1 -> sorted 1, 2, 4, 5 -> median 3.0
	out, err := table([]Case{c}, 4, exitZero, clock(0, 8, 0, 5, 0, 2, 0, 4, 0, 1))
	if err != nil || !strings.Contains(out, "| even | 8.0 ms | 3.0 ms | 1.0 ms | 5.0 ms | [0] |") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestAFailingExecStopsTheMeasurement(t *testing.T) {
	c := Case{Name: "broken", Steps: []Step{{Argv: []string{"x"}}}}
	_, err := Measure([]Case{c}, 1, func(Case, Step) (int, error) {
		return 0, errors.New("no such binary")
	}, clock(0, 1, 1, 2))
	if err == nil || !strings.Contains(err.Error(), "broken: no such binary") {
		t.Fatalf("err %v", err)
	}
}

func TestAFailingParStepStopsTheMeasurement(t *testing.T) {
	c := Case{Name: "broken", Mode: "par", Steps: []Step{{Argv: []string{"x"}}}}
	_, err := Measure([]Case{c}, 0, func(Case, Step) (int, error) {
		return 0, errors.New("no such binary")
	}, clock(0, 1))
	if err == nil || !strings.Contains(err.Error(), "broken: no such binary") {
		t.Fatalf("err %v", err)
	}
}

func TestMeasureRefusesACaseWithoutSteps(t *testing.T) {
	got, err := Measure([]Case{{Name: "empty"}}, 1, exitZero, clock(0))
	if err == nil || !strings.Contains(err.Error(), "empty: no steps") {
		t.Fatalf("err %v", err)
	}
	if got != nil {
		t.Fatalf("measured anyway: %+v", got)
	}
}

// A step with no argv names no process. validate once let it through, and the
// run then panicked in Exec on argv[0] -- exactly what validate exists to
// prevent.
func TestMeasureRefusesAStepWithoutAnArgv(t *testing.T) {
	got, err := Measure([]Case{{Name: "silent", Steps: []Step{{Argv: []string{"x"}}, {}}}}, 1, exitZero, clock(0))
	if err == nil || !strings.Contains(err.Error(), "silent: step #2 names no command") {
		t.Fatalf("err %v", err)
	}
	if got != nil {
		t.Fatalf("measured anyway: %+v", got)
	}
}

func TestMeasureRefusesAnUnknownMode(t *testing.T) {
	c := Case{Name: "odd", Mode: "diagonal", Steps: []Step{{Argv: []string{"x"}}}}
	got, err := Measure([]Case{c}, 1, exitZero, clock(0, 1))
	if err == nil || !strings.Contains(err.Error(), `odd: unknown mode "diagonal"`) {
		t.Fatalf("err %v", err)
	}
	if got != nil {
		t.Fatalf("measured anyway: %+v", got)
	}
}

// A broken case behind a good one must not start the good one's processes:
// every case is judged before the first is measured.
func TestMeasureStartsNothingWhenALaterCaseIsInvalid(t *testing.T) {
	good := Case{Name: "good", Steps: []Step{{Argv: []string{"x"}}}}
	bad := Case{Name: "bad", Mode: "diagonal", Steps: []Step{{Argv: []string{"x"}}}}
	started := 0
	got, err := Measure([]Case{good, bad}, 1, func(Case, Step) (int, error) {
		started++
		return 0, nil
	}, clock(0, 1, 1, 2))
	if err == nil || !strings.Contains(err.Error(), `bad: unknown mode "diagonal"`) {
		t.Fatalf("err %v", err)
	}
	if got != nil || started != 0 {
		t.Fatalf("measured anyway: %+v, %d starts", got, started)
	}
}

// once keeps the mode switch total although Measure validates ahead of it.
func TestOnceRefusesAnUnknownMode(t *testing.T) {
	c := Case{Name: "odd", Mode: "diagonal", Steps: []Step{{Argv: []string{"x"}}}}
	_, _, err := once(c, exitZero, clock(0))
	if err == nil || !strings.Contains(err.Error(), `odd: unknown mode "diagonal"`) {
		t.Fatalf("err %v", err)
	}
}
