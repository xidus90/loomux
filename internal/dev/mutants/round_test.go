package mutants

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

const signSource = "package p\n\nfunc Sign(a int) int {\n\tif a > 0 {\n\t\treturn 1\n\t}\n\treturn 0\n}\n"

const oneSource = "package p\n\nfunc One(x int) bool {\n\tif x == 1 {\n\t\treturn true\n\t}\n\treturn false\n}\n"

// writePackage lays out one package directory under root.
func writePackage(t *testing.T, root, pkg string, files map[string]string) {
	t.Helper()
	dir := filepath.Join(root, pkg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// call is one run the fake suite was asked for.
type call struct {
	pkg, overlay string
	bound        time.Duration     // the time limit the run was given
	line         string            // the line the replacement changes; "" for a baseline
	replace      map[string]string // the overlay's Replace; nil for a baseline
}

// fakeSuite is a TestFunc that starts nothing. It reads the overlay go test
// would get, finds the line the replacement changes, and answers from it.
// Its clock stands still except while a baseline runs, which moves it on by
// took[pkg]; handed to Round as Options.Now, it times the unchanged suite.
type fakeSuite struct {
	mu       sync.Mutex
	calls    []call
	baseline func(pkg string) (Outcome, error)
	mutant   func(line string) (Outcome, error)
	took     map[string]time.Duration
	clock    time.Time
}

func (f *fakeSuite) now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.clock
}

func (f *fakeSuite) test(pkg, overlay string, bound time.Duration) (Outcome, error) {
	c := call{pkg: pkg, overlay: overlay, bound: bound}
	if overlay != "" {
		var o struct{ Replace map[string]string }
		data, err := os.ReadFile(overlay)
		if err == nil {
			err = json.Unmarshal(data, &o)
		}
		if err != nil {
			return 0, err
		}
		c.replace = o.Replace
		for source, replacement := range o.Replace {
			c.line = changedLine(source, replacement)
		}
	}
	f.mu.Lock()
	f.calls = append(f.calls, c)
	if overlay == "" {
		f.clock = f.clock.Add(f.took[pkg])
	}
	f.mu.Unlock()
	if overlay == "" {
		return f.baseline(pkg)
	}
	return f.mutant(c.line)
}

// mutantCalls are the calls that carried an overlay.
func (f *fakeSuite) mutantCalls() []call {
	var out []call
	for _, c := range f.calls {
		if c.overlay != "" {
			out = append(out, c)
		}
	}
	return out
}

// changedLine is the first line of replacement that differs from source.
func changedLine(source, replacement string) string {
	a, _ := os.ReadFile(source)
	b, _ := os.ReadFile(replacement)
	was, now := strings.Split(string(a), "\n"), strings.Split(string(b), "\n")
	for i := range min(len(was), len(now)) {
		if was[i] != now[i] {
			return now[i]
		}
	}
	return ""
}

func green(string) (Outcome, error) { return Passed, nil }

// floor is the report's bound line for a package whose unchanged suite took
// no time on a clock that stands still.
func floor(pkg string) string {
	return pkg + ": each mutant run is bounded at 1m0s, the floor (3 × the unchanged suite's 0s does not exceed it)\n"
}

func TestRoundReportsEveryVerdictInGenerationOrder(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{
		"p.go":      signSource,
		"p_test.go": "package p\n\nfunc probe(a int) {\n\tif a > 1 {\n\t}\n}\n",
	})
	release := make(chan struct{})
	suite := &fakeSuite{baseline: green, mutant: func(line string) (Outcome, error) {
		switch line {
		case "\tif true {":
			// Ends only after the last mutant has started, so the report
			// has to wait for it instead of printing in finishing order.
			<-release
			return Passed, nil
		case "\tif false {":
			return Failed, nil
		case "\tif !(a > 0) {":
			return BuildFailed, nil
		case "\tif a >= 0 {":
			close(release)
			return TimedOut, nil
		}
		return 0, fmt.Errorf("unexpected mutated line %q", line)
	}}
	var out strings.Builder
	sum, err := Round(Options{Packages: []string{"p"}, Root: root, Workers: 4, Now: suite.now}, suite.test, &out)
	if err != nil {
		t.Fatal(err)
	}
	want := floor("p") + "[1/4] SURVIVED  (a1) p.go:4  if a > 0 {  ->  if true {\n" +
		"[2/4] killed    (a1) p.go:4  if a > 0 {  ->  if false {\n" +
		"[3/4] no mutant (a4) p.go:4  if a > 0 {  ->  if !(a > 0) {\n" +
		"[4/4] timed out (a3) p.go:4  if a > 0 {  ->  if a >= 0 {\n" +
		"\n" +
		"4 mutants over p, oracle go\n" +
		"1 do not compile and are no mutants\n" +
		"1 of the killed ran into its bound; a timeout is a kill only where the suite would never have finished\n" +
		"1 survived:\n" +
		"  (a1) p.go:4  if a > 0 {  ->  if true {\n"
	if out.String() != want {
		t.Fatalf("report:\n%s", out.String())
	}
	if sum.Killed != 2 || sum.TimedOut != 1 || sum.Survived != 1 || sum.NotCompiled != 1 || sum.NoMutant != 0 ||
		len(sum.Survivors) != 1 || sum.Survivors[0].Now != "\tif true {" {
		t.Fatalf("summary %+v", sum)
	}
	source := filepath.Join(root, "p", "p.go")
	if len(suite.calls) != 5 || suite.calls[0].pkg != "p" || suite.calls[0].overlay != "" {
		t.Fatalf("calls %+v", suite.calls)
	}
	for _, c := range suite.mutantCalls() {
		if c.pkg != "p" || len(c.replace) != 1 || filepath.Base(c.replace[source]) != "p.go" || filepath.Base(c.overlay) != "overlay.json" {
			t.Fatalf("overlay %+v", c)
		}
		if _, err := os.Stat(filepath.Dir(c.overlay)); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("%s outlived the round: %v", filepath.Dir(c.overlay), err)
		}
	}
	if data, _ := os.ReadFile(source); string(data) != signSource {
		t.Fatalf("the tree was written: %q", data)
	}
}

func TestRoundRefusesARedBaselineBeforeAnyMutant(t *testing.T) {
	for _, red := range []Outcome{Failed, BuildFailed, TimedOut} {
		root := t.TempDir()
		writePackage(t, root, "p", map[string]string{"p.go": signSource})
		writePackage(t, root, "q", map[string]string{"q.go": oneSource})
		suite := &fakeSuite{
			baseline: func(pkg string) (Outcome, error) {
				if pkg == "q" {
					return red, nil
				}
				return Passed, nil
			},
			mutant: func(string) (Outcome, error) { return Failed, nil },
		}
		var out strings.Builder
		_, err := Round(Options{Packages: []string{"p", "q"}, Root: root}, suite.test, &out)
		if !errors.Is(err, ErrBaselineRed) || err.Error() != "q: the suite is not green before the round" {
			t.Fatalf("outcome %d: %v", red, err)
		}
		if len(suite.calls) != 2 || out.Len() != 0 {
			t.Fatalf("outcome %d: calls %+v, report %q", red, suite.calls, out.String())
		}
	}
}

func TestRoundPassesOnAnErrorOfTheBaseline(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	suite := &fakeSuite{baseline: func(string) (Outcome, error) { return 0, errors.New("go: not found") }}
	if _, err := Round(Options{Packages: []string{"p"}, Root: root}, suite.test, &strings.Builder{}); err == nil || err.Error() != "go: not found" {
		t.Fatalf("%v", err)
	}
}

func TestRoundNeedsSourceFilesBeforeItAsksTheSuite(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "tests", map[string]string{"p_test.go": "package p\n"})
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	for _, c := range []struct {
		pkg, only, want string
	}{
		{"tests", "", "tests: no source files"},
		{"p", "nothing", "p: no source files"},
	} {
		suite := &fakeSuite{baseline: green}
		_, err := Round(Options{Packages: []string{c.pkg}, Root: root, Only: c.only}, suite.test, &strings.Builder{})
		if !errors.Is(err, ErrNoSources) || err.Error() != c.want || len(suite.calls) != 0 {
			t.Fatalf("%s: %v, calls %+v", c.pkg, err, suite.calls)
		}
	}
	suite := &fakeSuite{baseline: green}
	_, err := Round(Options{Packages: []string{"gone"}, Root: root}, suite.test, &strings.Builder{})
	if err == nil || errors.Is(err, ErrNoSources) || len(suite.calls) != 0 {
		t.Fatalf("missing package: %v", err)
	}
}

func TestRoundNeedsAnAbsoluteRoot(t *testing.T) {
	_, err := Round(Options{Packages: []string{"p"}, Root: "relative"}, (&fakeSuite{}).test, &strings.Builder{})
	if err == nil || err.Error() != `root "relative" is not an absolute path` {
		t.Fatalf("%v", err)
	}
}

func TestRoundFiltersByFileNameAndFamily(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"sign.go": signSource, "one.go": oneSource})
	suite := &fakeSuite{baseline: green, mutant: func(string) (Outcome, error) { return Failed, nil }}
	var out strings.Builder
	sum, err := Round(Options{Packages: []string{"p"}, Root: root, Only: "on", Family: "a3", Workers: 1, Now: suite.now}, suite.test, &out)
	if err != nil {
		t.Fatal(err)
	}
	want := floor("p") + "[1/1] killed    (a3) one.go:4  if x == 1 {  ->  if x != 1 {\n" +
		"\n" +
		"1 mutants over p, oracle go\n" +
		"0 do not compile and are no mutants\n" +
		"0 survived:\n"
	if out.String() != want || sum.Killed != 1 {
		t.Fatalf("summary %+v, report:\n%s", sum, out.String())
	}
}

func TestRoundCountsAnUnchangedLineAsNoMutant(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"t.go": "package p\n\nfunc T() int {\n\tif true {\n\t\treturn 1\n\t}\n\treturn 0\n}\n"})
	suite := &fakeSuite{baseline: green, mutant: func(string) (Outcome, error) { return Failed, nil }}
	var out strings.Builder
	sum, err := Round(Options{Packages: []string{"p"}, Root: root, Workers: 1, Now: suite.now}, suite.test, &out)
	if err != nil {
		t.Fatal(err)
	}
	want := floor("p") + "[1/3] no mutant (a1) t.go:4  if true {  ->  if true {\n" +
		"[2/3] killed    (a1) t.go:4  if true {  ->  if false {\n" +
		"[3/3] killed    (a4) t.go:4  if true {  ->  if !(true) {\n" +
		"\n" +
		"3 mutants over p, oracle go\n" +
		"0 do not compile and are no mutants\n" +
		"1 change nothing and are no mutants\n" +
		"0 survived:\n"
	if out.String() != want || sum.NoMutant != 1 || sum.Killed != 2 || len(suite.mutantCalls()) != 2 {
		t.Fatalf("summary %+v, calls %d, report:\n%s", sum, len(suite.mutantCalls()), out.String())
	}
}

func TestRoundStopsAtTheFirstErrorAndLeavesNoOverlay(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	suite := &fakeSuite{baseline: green}
	suite.mutant = func(string) (Outcome, error) {
		if len(suite.mutantCalls()) > 1 {
			return 0, errors.New("a mutant ran after the error")
		}
		return 0, errors.New("go vanished")
	}
	var out strings.Builder
	_, err := Round(Options{Packages: []string{"p"}, Root: root, Workers: 1, Now: suite.now}, suite.test, &out)
	if err == nil || err.Error() != "go vanished" || out.String() != floor("p") {
		t.Fatalf("%v, report %q", err, out.String())
	}
	calls := suite.mutantCalls()
	if len(calls) != 1 {
		t.Fatalf("calls %+v", calls)
	}
	if _, err := os.Stat(filepath.Dir(calls[0].overlay)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("overlay directory left behind: %v", err)
	}
}

func TestRoundReportsATemporaryDirectoryItCannotMake(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	missing := filepath.Join(t.TempDir(), "missing")
	t.Setenv("TMP", missing)
	t.Setenv("TEMP", missing)
	t.Setenv("TMPDIR", missing)
	suite := &fakeSuite{baseline: green}
	_, err := Round(Options{Packages: []string{"p"}, Root: root, Workers: 1}, suite.test, &strings.Builder{})
	if err == nil || !strings.Contains(err.Error(), "missing") || len(suite.mutantCalls()) != 0 {
		t.Fatalf("%v", err)
	}
}

func TestRoundReportsAnUnreadableSource(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	if err := os.Mkdir(filepath.Join(root, "p", "dir.go"), 0o755); err != nil {
		t.Fatal(err)
	}
	suite := &fakeSuite{baseline: green}
	var out strings.Builder
	_, err := Round(Options{Packages: []string{"p"}, Root: root}, suite.test, &out)
	if err == nil || len(suite.mutantCalls()) != 0 || out.Len() != 0 {
		t.Fatalf("%v, report %q", err, out.String())
	}
}

func TestRoundAddsUpEveryPackage(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	writePackage(t, root, "q", map[string]string{"q.go": oneSource})
	suite := &fakeSuite{baseline: green, mutant: func(string) (Outcome, error) { return Passed, nil }}
	var out strings.Builder
	sum, err := Round(Options{Packages: []string{"p", "q"}, Root: root, Family: "a3", Now: suite.now}, suite.test, &out)
	if err != nil {
		t.Fatal(err)
	}
	want := floor("p") + "[1/1] SURVIVED  (a3) p.go:4  if a > 0 {  ->  if a >= 0 {\n" +
		"\n" +
		"1 mutants over p, oracle go\n" +
		"0 do not compile and are no mutants\n" +
		"1 survived:\n" +
		"  (a3) p.go:4  if a > 0 {  ->  if a >= 0 {\n" +
		floor("q") + "[1/1] SURVIVED  (a3) q.go:4  if x == 1 {  ->  if x != 1 {\n" +
		"\n" +
		"1 mutants over q, oracle go\n" +
		"0 do not compile and are no mutants\n" +
		"1 survived:\n" +
		"  (a3) q.go:4  if x == 1 {  ->  if x != 1 {\n"
	if out.String() != want || sum.Survived != 2 || len(sum.Survivors) != 2 {
		t.Fatalf("summary %+v, report:\n%s", sum, out.String())
	}
}

// The timeouts of every package add up like the kills they are counted in.
func TestRoundAddsUpTheTimeoutsOfEveryPackage(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	writePackage(t, root, "q", map[string]string{"q.go": oneSource})
	suite := &fakeSuite{baseline: green, mutant: func(string) (Outcome, error) { return TimedOut, nil }}
	var out strings.Builder
	sum, err := Round(Options{Packages: []string{"p", "q"}, Root: root, Family: "a3", Now: suite.now}, suite.test, &out)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Killed != 2 || sum.TimedOut != 2 || sum.Survived != 0 {
		t.Fatalf("summary %+v, report:\n%s", sum, out.String())
	}
}

// A fixed bound on every run turns a mutant the suite does not notice into a
// kill wherever the unchanged suite needs nearly that long; the bound follows
// each package's own baseline instead, and the baseline is not cut short.
func TestRoundBoundsEachMutantRunByThreeTimesItsUnchangedSuite(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, "p", map[string]string{"p.go": signSource})
	writePackage(t, root, "q", map[string]string{"q.go": oneSource})
	suite := &fakeSuite{
		baseline: green,
		mutant:   func(string) (Outcome, error) { return Failed, nil },
		took:     map[string]time.Duration{"p": 50 * time.Second, "q": 5 * time.Second},
	}
	var out strings.Builder
	_, err := Round(Options{Packages: []string{"p", "q"}, Root: root, Family: "a3", Now: suite.now}, suite.test, &out)
	if err != nil {
		t.Fatal(err)
	}
	want := "p: each mutant run is bounded at 2m30s (3 × the unchanged suite's 50s)\n" +
		"[1/1] killed    (a3) p.go:4  if a > 0 {  ->  if a >= 0 {\n" +
		"\n" +
		"1 mutants over p, oracle go\n" +
		"0 do not compile and are no mutants\n" +
		"0 survived:\n" +
		"q: each mutant run is bounded at 1m0s, the floor (3 × the unchanged suite's 5s does not exceed it)\n" +
		"[1/1] killed    (a3) q.go:4  if x == 1 {  ->  if x != 1 {\n" +
		"\n" +
		"1 mutants over q, oracle go\n" +
		"0 do not compile and are no mutants\n" +
		"0 survived:\n"
	bounds := map[string]time.Duration{"p": 150 * time.Second, "q": 60 * time.Second}
	for _, c := range suite.calls {
		bound := bounds[c.pkg]
		if c.overlay == "" {
			bound = 10 * time.Minute
		}
		if c.bound != bound {
			t.Errorf("%s, overlay %q: bound %s, want %s", c.pkg, c.overlay, c.bound, bound)
		}
	}
	if len(suite.calls) != 4 || out.String() != want {
		t.Fatalf("calls %+v, report:\n%s", suite.calls, out.String())
	}
}

func TestBoundForIsThreeTimesTheBaselineInWholeSecondsAndAtLeastAMinute(t *testing.T) {
	for _, c := range []struct{ took, want time.Duration }{
		{0, time.Minute},
		{20 * time.Second, time.Minute},
		{20*time.Second + 100*time.Millisecond, 61 * time.Second},
		{50 * time.Second, 150 * time.Second},
		{58*time.Second + 300*time.Millisecond, 175 * time.Second},
	} {
		if got := boundFor(c.took); got != c.want {
			t.Errorf("baseline %s: bound %s, want %s", c.took, got, c.want)
		}
	}
}

func TestDefaultWorkersIsHalfTheProcessors(t *testing.T) {
	if got := DefaultWorkers(); got != max(runtime.NumCPU()/2, 1) {
		t.Fatalf("%d", got)
	}
}
