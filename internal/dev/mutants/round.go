package mutants

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Options says what a round mutates. Root is the absolute directory the
// packages are named relative to and go test runs in.
type Options struct {
	Packages []string
	Root     string
	Only     string
	Family   string
	Workers  int
	// Now is the clock the unchanged suite is timed with; nil is time.Now.
	Now func() time.Time
}

// baselineBound is the time limit on the unchanged suite, the toolchain's own
// default. The baseline is what every later bound is measured from, so it
// must not be cut short by one.
const baselineBound = 10 * time.Minute

// minBound is the least time limit a mutant run gets. A run past its limit
// counts as killed, and each mutant needs one: striking out a cycle guard
// turns a walk into an endless one.
const minBound = time.Minute

// boundFor is the time limit on each mutant run of a package whose unchanged
// suite took took: three times that, rounded up to whole seconds, and never
// below minBound. A mutant the suite does not notice runs about as long as
// the unchanged suite, and a limit close to that would count it as killed
// whenever parallel runs slow it down.
func boundFor(took time.Duration) time.Duration {
	bound := 3 * took
	if rest := bound % time.Second; rest != 0 {
		bound += time.Second - rest
	}
	return max(bound, minBound)
}

// describeBound is the report's line on the bound of a package.
func describeBound(pkg string, took, bound time.Duration) string {
	took = took.Round(100 * time.Millisecond)
	if bound == minBound {
		return fmt.Sprintf("%s: each mutant run is bounded at %s, the floor (3 × the unchanged suite's %s does not exceed it)", pkg, bound, took)
	}
	return fmt.Sprintf("%s: each mutant run is bounded at %s (3 × the unchanged suite's %s)", pkg, bound, took)
}

// Summary adds a round up over all its packages.
type Summary struct {
	Killed, Survived, NotCompiled, NoMutant int
	// TimedOut are the killed that ran into their bound instead of failing.
	TimedOut  int
	Survivors []Mutant
}

// ErrBaselineRed refuses a round whose suite fails before the first mutant:
// every mutant would count as killed.
var ErrBaselineRed = errors.New("the suite is not green before the round")

// ErrNoSources refuses a package, or an Only filter, that leaves no file to
// mutate.
var ErrNoSources = errors.New("no source files")

// DefaultWorkers is half the processors and at least one: one go test run
// already compiles and tests on several cores.
func DefaultWorkers() int {
	return max(runtime.NumCPU()/2, 1)
}

// Round mutates every package and writes the script's report to w. Every
// package is checked for sources and for a green suite before the first
// mutant runs, and that green run is timed to bound its mutant runs; then
// each package gets its line on that bound and its block of verdicts and sums. A
// survivor is a finding and not an error; the first error ends the round.
func Round(opts Options, test TestFunc, w io.Writer) (Summary, error) {
	var total Summary
	// An overlay names the replaced file by its absolute path, and Round does
	// not guess one from the working directory of this process.
	if !filepath.IsAbs(opts.Root) {
		return total, fmt.Errorf("root %q is not an absolute path", opts.Root)
	}
	files := make([][]string, len(opts.Packages))
	for i, pkg := range opts.Packages {
		found, err := sources(filepath.Join(opts.Root, pkg), opts.Only)
		if err != nil {
			return total, err
		}
		if len(found) == 0 {
			return total, fmt.Errorf("%s: %w", pkg, ErrNoSources)
		}
		files[i] = found
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	took := make([]time.Duration, len(opts.Packages))
	for i, pkg := range opts.Packages {
		started := now()
		outcome, err := test(pkg, "", baselineBound)
		if err != nil {
			return total, err
		}
		if outcome != Passed {
			return total, fmt.Errorf("%s: %w", pkg, ErrBaselineRed)
		}
		took[i] = now().Sub(started)
	}
	workers := opts.Workers
	if workers < 1 {
		workers = DefaultWorkers()
	}
	for i, pkg := range opts.Packages {
		sum, err := roundOf(pkg, files[i], opts.Family, workers, took[i], test, w)
		total.Killed += sum.Killed
		total.Survived += sum.Survived
		total.NotCompiled += sum.NotCompiled
		total.NoMutant += sum.NoMutant
		total.TimedOut += sum.TimedOut
		total.Survivors = append(total.Survivors, sum.Survivors...)
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// sources lists the non-test Go files of a package directory whose name
// contains only, in name order.
func sources(dir, only string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") && strings.Contains(name, only) {
			found = append(found, filepath.Join(dir, name))
		}
	}
	return found, nil
}

// finished is one mutant's turn as the report needs it.
type finished struct {
	outcome   Outcome
	unchanged bool
	skipped   bool
	err       error
}

// roundOf runs one package's mutants on a pool of workers and reports each in
// generation order as soon as its turn has come, so the report reads the same
// whatever order the runs finish in. took is how long the unchanged suite
// ran; every mutant run is bounded by boundFor of it.
func roundOf(pkg string, files []string, family string, workers int, took time.Duration, test TestFunc, w io.Writer) (Summary, error) {
	var sum Summary
	originals := map[string][]byte{}
	var all []Mutant
	for _, path := range files {
		source, err := os.ReadFile(path)
		if err != nil {
			return sum, err
		}
		originals[path] = source
		for _, m := range Generate(path, source) {
			if family == "" || m.Family == family {
				all = append(all, m)
			}
		}
	}
	bound := boundFor(took)
	fmt.Fprintln(w, describeBound(pkg, took, bound))

	results := make([]chan finished, len(all))
	for i := range results {
		results[i] = make(chan finished, 1)
	}
	var first atomic.Pointer[error]
	var wg sync.WaitGroup
	defer wg.Wait()
	slots := make(chan struct{}, workers)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i, m := range all {
			slots <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i] <- runOne(pkg, m, originals[m.Path], bound, test, &first)
				<-slots
			}()
		}
	}()

	for i, m := range all {
		r := <-results[i]
		if r.skipped || r.err != nil {
			return sum, *first.Load()
		}
		var verdict string
		switch {
		case r.unchanged:
			verdict = "no mutant"
			sum.NoMutant++
		case r.outcome == BuildFailed:
			verdict = "no mutant"
			sum.NotCompiled++
		case r.outcome == Passed:
			verdict = "SURVIVED "
			sum.Survived++
			sum.Survivors = append(sum.Survivors, m)
		case r.outcome == TimedOut:
			// A mutant that never answers counts as killed, like one that
			// answers wrongly, but the report names it: under load a suite
			// that would have passed runs into its bound as well.
			verdict = "timed out"
			sum.Killed++
			sum.TimedOut++
		default:
			verdict = "killed   "
			sum.Killed++
		}
		fmt.Fprintf(w, "[%d/%d] %s %s\n", i+1, len(all), verdict, m)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%d mutants over %s, oracle go\n", len(all), pkg)
	fmt.Fprintf(w, "%d do not compile and are no mutants\n", sum.NotCompiled)
	if sum.NoMutant > 0 {
		fmt.Fprintf(w, "%d change nothing and are no mutants\n", sum.NoMutant)
	}
	if sum.TimedOut > 0 {
		fmt.Fprintf(w, "%d of the killed ran into its bound; a timeout is a kill only where the suite would never have finished\n", sum.TimedOut)
	}
	fmt.Fprintf(w, "%d survived:\n", len(sum.Survivors))
	for _, m := range sum.Survivors {
		fmt.Fprintf(w, "  %s\n", m)
	}
	return sum, nil
}

// runOne is one mutant's turn. A mutant that leaves its line as it was is no
// mutant and never reaches the suite; once a run has failed with an error,
// no further mutant starts.
func runOne(pkg string, m Mutant, source []byte, bound time.Duration, test TestFunc, first *atomic.Pointer[error]) finished {
	if m.Now == m.Was {
		return finished{unchanged: true}
	}
	if first.Load() != nil {
		return finished{skipped: true}
	}
	outcome, err := try(pkg, m, source, bound, test)
	if err != nil {
		first.CompareAndSwap(nil, &err)
	}
	return finished{outcome: outcome, err: err}
}

// try hands one mutant to the suite through an overlay in a directory of its
// own, which is removed again whatever the run came to.
func try(pkg string, m Mutant, source []byte, bound time.Duration, test TestFunc) (Outcome, error) {
	dir, err := os.MkdirTemp("", "loomux-mutant-")
	if err == nil {
		defer os.RemoveAll(dir)
		replacement := filepath.Join(dir, filepath.Base(m.Path))
		overlay := filepath.Join(dir, "overlay.json")
		// A write into the directory this call has just made fails only
		// through the system; both writes share the error check that a
		// missing temporary directory already exercises.
		err = errors.Join(
			os.WriteFile(replacement, mutate(source, m), 0o644),
			os.WriteFile(overlay, overlayJSON(m.Path, replacement), 0o644),
		)
		if err == nil {
			return test(pkg, overlay, bound)
		}
	}
	return 0, err
}

// overlayJSON is the file go test -overlay reads:
// {"Replace": {"<source>": "<replacement>"}}.
func overlayJSON(source, replacement string) []byte {
	// A map of strings holds no value json.Marshal can refuse.
	data, _ := json.Marshal(map[string]map[string]string{"Replace": {source: replacement}})
	return data
}
