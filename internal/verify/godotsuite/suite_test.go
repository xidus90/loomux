package godotsuite

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

// engine stands in for the Godot binary: it recognises disk.gd and shard
// calls by argv and writes what a shard would leave behind.
type engine struct {
	mu       sync.Mutex
	log      []string
	specs    []child.Spec
	code     map[int]int
	noReport map[int]bool
	noLCOV   map[int]bool
	result   map[string]child.Result // by "instrument", "restore" or "shard:<i>"
	onMode   func(mode string)
}

func (e *engine) start(s child.Spec) child.Result {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.specs = append(e.specs, s)
	argv := s.Argv
	if strings.Contains(strings.Join(argv, " "), "disk.gd") {
		mode := argv[len(argv)-1]
		e.log = append(e.log, mode)
		if e.onMode != nil {
			e.onMode(mode)
		}
		return e.result[mode]
	}
	var rd string
	var suites []string
	for i, a := range argv {
		if a == "-rd" {
			rd = argv[i+1]
		}
		if a == "-a" {
			suites = append(suites, argv[i+1])
		}
	}
	idx := 0
	fmt.Sscan(filepath.Base(filepath.Dir(rd)), &idx)
	e.log = append(e.log, fmt.Sprintf("shard:%d", idx))
	if r, ok := e.result[fmt.Sprintf("shard:%d", idx)]; ok {
		return r
	}
	if !e.noReport[idx] {
		var b strings.Builder
		b.WriteString("<testsuites>")
		for _, p := range suites {
			n := strings.TrimSuffix(filepath.Base(p), ".gd")
			pkg := filepath.ToSlash(filepath.Dir(strings.TrimPrefix(p, "res://")))
			fmt.Fprintf(&b, `<testsuite name=%q package=%q time="0.5"><testcase name="test_x"/></testsuite>`, n, pkg)
		}
		b.WriteString("</testsuites>")
		dir := filepath.Join(rd, "report_1")
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "results.xml"), []byte(b.String()), 0o644)
	}
	for _, kv := range s.Env {
		if d, ok := strings.CutPrefix(kv, "LOOMUX_COVERAGE_SHARD_DIR="); ok && !e.noLCOV[idx] {
			os.MkdirAll(filepath.Join(d, "report"), 0o755)
			os.WriteFile(filepath.Join(d, "report", "lcov.info"), []byte(fmt.Sprintf("SF:s%d.gd\nend_of_record\n", idx)), 0o644)
		}
	}
	return child.Result{Code: e.code[idx], Stdout: "out", Stderr: "err"}
}

type world struct {
	root, checkout string
	e              *engine
	o              Options
}

func write(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newWorld(t *testing.T, project string) *world {
	t.Helper()
	return newWorldAt(t, filepath.Join(t.TempDir(), "proj"), project)
}

func newWorldAt(t *testing.T, root, project string) *world {
	t.Helper()
	w := &world{root: root, checkout: t.TempDir(), e: &engine{}}
	write(t, filepath.Join(root, "project.godot"), project)
	write(t, filepath.Join(root, "addons/gdUnit4/bin/GdUnitCmdTool.gd"), "")
	write(t, filepath.Join(root, ".godot/global_script_class_cache.cfg"), "")
	write(t, filepath.Join(root, "test/a_test.gd"), "extends GdUnitTestSuite\n")
	write(t, filepath.Join(root, "test/b_test.gd"), "extends GdUnitTestSuite\n")
	write(t, filepath.Join(root, "src/a.gd"), "extends Node\n")
	w.o = Options{Root: root, CheckoutRoot: w.checkout, Area: ".", Godot: "godot", RunID: "r1", CPUs: 2,
		GOOS: "linux", Start: w.e.start, Now: time.Now}
	return w
}

const plain = "config_version=5\n"
const withAutoload = "config_version=5\n[autoload]\nClock=\"*res://src/a.gd\"\n"

func run(t *testing.T, w *world) Result {
	t.Helper()
	res, err := Run(w.o)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func (w *world) stateDir() string {
	return filepath.Join(w.checkout, ".loomux", "state", "godot-suite")
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func TestGreenRunPlansTwoShardsAndSavesTimes(t *testing.T) {
	w := newWorld(t, plain)
	res := run(t, w)
	if len(res.Findings) != 0 {
		t.Fatalf("findings: %q", res.Findings)
	}
	if len(w.e.specs) != 2 {
		t.Fatalf("want 2 shards, got %d", len(w.e.specs))
	}
	for _, s := range w.e.specs {
		suite, i := "res://test/a_test.gd", 0
		if len(s.Argv) > 9 && s.Argv[9] == "res://test/b_test.gd" {
			suite, i = s.Argv[9], 1
		}
		want := []string{"godot", "--path", w.root, "-s", "-d", "--remote-debug", "tcp://127.0.0.1:0",
			"res://.godot/loomux/run_GdUnitCmdTool.gd", "-a", suite, "-c", "--headless", "--ignoreHeadlessMode", "-rd",
			filepath.Join(w.stateDir(), "r1-root", fmt.Sprint(i), "reports")}
		if !reflect.DeepEqual(s.Argv, want) {
			t.Errorf("argv %q want %q", s.Argv, want)
		}
		env := strings.Join(s.Env, "\n")
		if !strings.Contains(env, "APPDATA=") || !strings.Contains(env, "XDG_DATA_HOME=") {
			t.Errorf("env: %q", s.Env)
		}
		if strings.Contains(env, "LOOMUX_COVERAGE_SHARD_DIR") {
			t.Errorf("no coverage asked: %q", s.Env)
		}
		if s.Dir != w.root {
			t.Errorf("dir %q", s.Dir)
		}
	}
	times, _ := LoadTimes(filepath.Join(w.stateDir(), "root.json"))
	if len(times) != 2 || times["res://test/a_test.gd"] != 0.5 {
		t.Errorf("times: %v", times)
	}
}

func TestRootWithSpacesAndHashStaysOneArgvElement(t *testing.T) {
	w := newWorldAt(t, filepath.Join(t.TempDir(), "a #b"), plain)
	run(t, w)
	if len(w.e.specs) == 0 {
		t.Fatal("no shard ran")
	}
	for _, s := range w.e.specs {
		n := 0
		for i, a := range s.Argv {
			if a == "--path" {
				n++
				if s.Argv[i+1] != w.root {
					t.Errorf("--path value %q", s.Argv[i+1])
				}
			}
		}
		if n != 1 {
			t.Errorf("--path count %d in %q", n, s.Argv)
		}
	}
}

func TestPrechecksStopBeforeAnyEngine(t *testing.T) {
	t.Run("gdUnit4 missing", func(t *testing.T) {
		w := newWorld(t, plain)
		os.Remove(filepath.Join(w.root, "addons/gdUnit4/bin/GdUnitCmdTool.gd"))
		res := run(t, w)
		if res.Findings[0] != "gdUnit4 is missing: addons/gdUnit4/bin/GdUnitCmdTool.gd" || len(w.e.specs) != 0 {
			t.Fatalf("%q %d", res.Findings, len(w.e.specs))
		}
	})
	t.Run("stale class cache", func(t *testing.T) {
		w := newWorld(t, plain)
		write(t, filepath.Join(w.root, "src/k.gd"), "class_name Known\n")
		res := run(t, w)
		if len(res.Findings) != 1 || !strings.Contains(res.Findings[0], "class_name not in the class cache: Known") || len(w.e.specs) != 0 {
			t.Fatalf("%q %d", res.Findings, len(w.e.specs))
		}
	})
	t.Run("hook conflict counts in every run", func(t *testing.T) {
		for _, cov := range []bool{false, true} {
			w := newWorld(t, "[gdunit4]\nhooks/session_hooks=Dictionary(\"x\", \"res://h.gd\")\n")
			write(t, filepath.Join(w.root, "h.gd"), "CoverageApi.go()")
			if cov {
				w.o.Coverage = filepath.Join(w.checkout, "cov")
			}
			res := run(t, w)
			if len(res.Findings) != 1 || !strings.Contains(res.Findings[0], "registers a coverage session hook") || len(w.e.specs) != 0 {
				t.Fatalf("coverage=%v: %q %d", cov, res.Findings, len(w.e.specs))
			}
		}
	})
}

func TestNoSuite(t *testing.T) {
	w := newWorld(t, plain)
	os.Remove(filepath.Join(w.root, "test/a_test.gd"))
	os.Remove(filepath.Join(w.root, "test/b_test.gd"))
	res := run(t, w)
	if want := "no gdUnit4 test suite under " + w.root; len(res.Findings) != 1 || res.Findings[0] != want {
		t.Fatalf("%q", res.Findings)
	}
}

func TestOneShardDyingIsExactlyItsOwnFindingAndRestoreStillRuns(t *testing.T) {
	w := newWorld(t, withAutoload)
	w.o.Coverage = filepath.Join(w.checkout, "cov")
	w.e.noReport = map[int]bool{1: true}
	w.e.code = map[int]int{1: 139}
	res := run(t, w)
	if len(res.Findings) != 1 || !strings.HasPrefix(res.Findings[0], "shard 1 wrote no test report (exit 139)") {
		t.Fatalf("%q", res.Findings)
	}
	if w.e.log[len(w.e.log)-1] != "restore" {
		t.Fatalf("log %v", w.e.log)
	}
}

func TestFindingsFollowShardOrderWhateverFinishesFirst(t *testing.T) {
	w := newWorld(t, plain)
	w.e.noReport = map[int]bool{0: true, 1: true}
	res := run(t, w)
	if len(res.Findings) != 2 || !strings.HasPrefix(res.Findings[0], "shard 0 ") || !strings.HasPrefix(res.Findings[1], "shard 1 ") {
		t.Fatalf("%q", res.Findings)
	}
}

func TestCoverageInMemoryCopiesEachShardsLCOVAndNeverTouchesTheDisk(t *testing.T) {
	w := newWorld(t, plain)
	w.o.Coverage = filepath.Join(w.checkout, "cov")
	w.e.noLCOV = map[int]bool{1: true}
	res := run(t, w)
	for _, l := range w.e.log {
		if l == "instrument" || l == "restore" {
			t.Fatalf("disk used: %v", w.e.log)
		}
	}
	if !exists(filepath.Join(w.o.Coverage, "0.info")) || exists(filepath.Join(w.o.Coverage, "1.info")) {
		t.Fatal("coverage files wrong")
	}
	if !reflect.DeepEqual(res.Findings, []string{"shard 1 wrote no coverage report"}) {
		t.Fatalf("%q", res.Findings)
	}
	for _, s := range w.e.specs {
		if !strings.Contains(strings.Join(s.Env, "\n"), "LOOMUX_COVERAGE_SHARD_DIR=") {
			t.Errorf("env %q", s.Env)
		}
	}
}

func TestDiskCoverageBracketsTheShardsAndSettles(t *testing.T) {
	w := newWorld(t, withAutoload)
	w.o.Coverage = filepath.Join(w.checkout, "cov")
	src := filepath.Join(w.root, "src/a.gd")
	w.e.onMode = func(mode string) {
		if mode == "instrument" { // leaves the source changed, as a failed restore would
			os.WriteFile(src, []byte("# __NANO_COVERAGE_INSTRUMENTED__\n"), 0o644)
		}
	}
	w.e.result = map[string]child.Result{"shard:0": {Err: fmt.Errorf("cannot start")}}
	res := run(t, w)
	l := w.e.log
	if l[0] != "instrument" || l[len(l)-1] != "restore" || len(l) != 4 {
		t.Fatalf("log %v", l)
	}
	if len(res.Findings) != 3 || !strings.HasPrefix(res.Findings[0], "shard 0 wrote no test report") ||
		res.Findings[1] != "shard 0 wrote no coverage report" || res.Findings[2] != "src/a.gd differs after restoring the instrumentation" {
		t.Fatalf("%q", res.Findings)
	}
}

func TestLeftoverMarkerIsRestoredFirst(t *testing.T) {
	w := newWorld(t, plain)
	src := filepath.Join(w.root, "src/a.gd")
	write(t, src, "# __NANO_COVERAGE_INSTRUMENTED__\nextends Node\n")
	w.e.onMode = func(string) { os.WriteFile(src, []byte("extends Node\n"), 0o644) }
	res := run(t, w)
	if w.e.log[0] != "restore" || len(res.Findings) != 0 {
		t.Fatalf("%v %q", w.e.log, res.Findings)
	}
	if !reflect.DeepEqual(res.Notes, []string{"restored a leftover disk instrumentation"}) {
		t.Fatalf("notes %q", res.Notes)
	}
}

func TestLeftoverThatCannotBeRestoredStops(t *testing.T) {
	w := newWorld(t, plain)
	write(t, filepath.Join(w.root, "src/a.gd"), "# __NANO_COVERAGE_INSTRUMENTED__\n")
	w.e.result = map[string]child.Result{"restore": {Code: 1}}
	res := run(t, w)
	if !reflect.DeepEqual(res.Findings, []string{stillInstrumented}) || len(w.e.log) != 1 {
		t.Fatalf("%q %v", res.Findings, w.e.log)
	}
}

func TestFailedRestoreIsTheOnlyFinding(t *testing.T) {
	w := newWorld(t, withAutoload)
	w.o.Coverage = filepath.Join(w.checkout, "cov")
	w.e.noReport = map[int]bool{0: true}
	w.e.result = map[string]child.Result{"restore": {Code: 1}}
	res := run(t, w)
	if !reflect.DeepEqual(res.Findings, []string{stillInstrumented}) {
		t.Fatalf("%q", res.Findings)
	}
}

func TestFailedInstrumentSkipsTheShardsButRestores(t *testing.T) {
	w := newWorld(t, withAutoload)
	w.o.Coverage = filepath.Join(w.checkout, "cov")
	w.e.result = map[string]child.Result{"instrument": {Code: 1, Stdout: "bad"}}
	res := run(t, w)
	if !reflect.DeepEqual(w.e.log, []string{"instrument", "restore"}) ||
		len(res.Findings) != 1 || !strings.HasPrefix(res.Findings[0], "disk instrumentation failed (exit 1)") {
		t.Fatalf("%v %q", w.e.log, res.Findings)
	}
}

func TestRestoreOnlyRunsOnlyTheRestore(t *testing.T) {
	w := newWorld(t, plain)
	w.o.RestoreOnly = true
	res := run(t, w)
	if !reflect.DeepEqual(w.e.log, []string{"restore"}) || len(res.Findings) != 0 {
		t.Fatalf("%v %q", w.e.log, res.Findings)
	}
	w.e.result = map[string]child.Result{"restore": {Code: 1}}
	if res := run(t, w); !reflect.DeepEqual(res.Findings, []string{stillInstrumented}) {
		t.Fatalf("%q", res.Findings)
	}
}

func TestRestoreOnlyReportsAGlueWriteError(t *testing.T) {
	w := newWorld(t, plain)
	w.o.RestoreOnly = true
	write(t, filepath.Join(w.root, ".godot/loomux"), "a file where the directory goes")
	if _, err := Run(w.o); err == nil {
		t.Fatal("want an error")
	}
}

func TestCleanupKeepsRedRunsAndSweepsOldOnes(t *testing.T) {
	w := newWorld(t, plain)
	old := filepath.Join(w.stateDir(), "old-root")
	fresh := filepath.Join(w.stateDir(), "fresh-root")
	for _, d := range []string{old, fresh} {
		os.MkdirAll(d, 0o755)
	}
	write(t, filepath.Join(w.stateDir(), "stray.json"), "{}")
	then := time.Now().Add(-25 * time.Hour)
	os.Chtimes(old, then, then)
	own := filepath.Join(w.stateDir(), "r1-root")

	w.e.noReport = map[int]bool{0: true}
	run(t, w)
	if !exists(own) {
		t.Error("a red run keeps its shard directory")
	}
	if exists(old) || !exists(fresh) || !exists(filepath.Join(w.stateDir(), "stray.json")) {
		t.Error("sweep must remove only the old directory")
	}

	w.e.noReport = nil
	run(t, w)
	if exists(own) {
		t.Error("a green run removes its shard directory")
	}
}

func TestAreaNamesTheShardDirectoryAndTheTimings(t *testing.T) {
	w := newWorld(t, plain)
	w.o.Area = "game/godot"
	w.e.noReport = map[int]bool{0: true}
	run(t, w)
	if !exists(filepath.Join(w.stateDir(), "r1-game_godot", "0")) || !exists(filepath.Join(w.stateDir(), "game_godot.json")) {
		entries, _ := os.ReadDir(w.stateDir())
		t.Fatalf("state: %v", entries)
	}
}

func TestMacOSNoteOnlyWithSeveralShards(t *testing.T) {
	w := newWorld(t, plain)
	w.o.GOOS = "darwin"
	if res := run(t, w); !reflect.DeepEqual(res.Notes, []string{"on macOS the shards share user://"}) {
		t.Fatalf("%q", res.Notes)
	}
	w.o.GOOS = ""
	if res := run(t, w); (len(res.Notes) == 1) != (runtime.GOOS == "darwin") {
		t.Fatalf("host platform: %q", res.Notes)
	}
	w.o.GOOS = "darwin"
	w.o.CPUs = 1
	if res := run(t, w); len(res.Notes) != 0 {
		t.Fatalf("%q", res.Notes)
	}
}

func TestUnreadableTimingsAreANote(t *testing.T) {
	w := newWorld(t, plain)
	write(t, filepath.Join(w.stateDir(), "root.json"), "not json")
	if res := run(t, w); len(res.Notes) != 1 || !strings.Contains(res.Notes[0], "unreadable") {
		t.Fatalf("%q", res.Notes)
	}
}

func TestCoverageDataDirectoryIsRemovedOnlyWhenTheRunMadeIt(t *testing.T) {
	for _, existed := range []bool{false, true} {
		w := newWorld(t, withAutoload)
		w.o.Coverage = filepath.Join(w.checkout, "cov")
		dir := filepath.Join(w.root, "coverage-data")
		if existed {
			os.MkdirAll(dir, 0o755)
		}
		w.e.onMode = func(mode string) {
			if mode == "instrument" {
				os.MkdirAll(dir, 0o755)
			}
		}
		run(t, w)
		if exists(dir) != existed {
			t.Errorf("existed=%v but present=%v afterwards", existed, exists(dir))
		}
	}
}

func TestErrors(t *testing.T) {
	failing := func(w *world, mutate func()) {
		t.Helper()
		mutate()
		if _, err := Run(w.o); err == nil {
			t.Fatal("want an error")
		}
	}
	t.Run("state directory is a file", func(t *testing.T) {
		w := newWorld(t, plain)
		failing(w, func() { write(t, w.stateDir(), "x") })
	})
	t.Run("timings cannot be saved", func(t *testing.T) {
		w := newWorld(t, plain)
		failing(w, func() { os.MkdirAll(filepath.Join(w.stateDir(), "root.json"), 0o755) })
	})
	t.Run("glue cannot be written", func(t *testing.T) {
		w := newWorld(t, plain)
		failing(w, func() { write(t, filepath.Join(w.root, ".godot/loomux"), "x") })
	})
	t.Run("project.godot unreadable with coverage", func(t *testing.T) {
		w := newWorld(t, plain)
		w.o.Coverage = filepath.Join(w.checkout, "cov")
		failing(w, func() { os.Remove(filepath.Join(w.root, "project.godot")) })
	})
	t.Run("autoload patterns invalid", func(t *testing.T) {
		w := newWorld(t, withAutoload+"[nano_coverage]\ngeneral/ignore_paths=PackedStringArray(\"[\")\n")
		w.o.Coverage = filepath.Join(w.checkout, "cov")
		failing(w, func() {})
	})
	t.Run("coverage directory is a file", func(t *testing.T) {
		w := newWorld(t, plain)
		w.o.Coverage = filepath.Join(w.checkout, "cov")
		failing(w, func() { write(t, w.o.Coverage, "x") })
	})
	for n, name := range map[int]string{1: "leftover scan", 3: "suites", 4: "snapshot"} {
		t.Run("unreadable script during "+name, func(t *testing.T) {
			w := newWorld(t, withAutoload)
			w.o.Coverage = filepath.Join(w.checkout, "cov")
			reads := 0
			readFile = func(p string) ([]byte, error) {
				if filepath.Base(p) == "a.gd" {
					if reads++; reads == n {
						return nil, fmt.Errorf("denied")
					}
				}
				return os.ReadFile(p)
			}
			defer func() { readFile = os.ReadFile }()
			failing(w, func() {})
		})
	}
}

func TestLeftoverIsRestoredEvenWhenALaterCheckIsRed(t *testing.T) {
	marker := "# __NANO_COVERAGE_INSTRUMENTED__\nextends Node\n"
	t.Run("stale class cache", func(t *testing.T) {
		w := newWorld(t, plain)
		write(t, filepath.Join(w.root, "src/a.gd"), marker)
		write(t, filepath.Join(w.root, "src/k.gd"), "class_name Known\n")
		res := run(t, w)
		if !reflect.DeepEqual(w.e.log, []string{"restore"}) ||
			!reflect.DeepEqual(res.Notes, []string{"restored a leftover disk instrumentation"}) ||
			len(res.Findings) != 1 || !strings.Contains(res.Findings[0], "class_name not in the class cache") {
			t.Fatalf("%v %q %q", w.e.log, res.Notes, res.Findings)
		}
	})
	t.Run("no suite", func(t *testing.T) {
		w := newWorld(t, plain)
		os.Remove(filepath.Join(w.root, "test/a_test.gd"))
		os.Remove(filepath.Join(w.root, "test/b_test.gd"))
		write(t, filepath.Join(w.root, "src/a.gd"), marker)
		res := run(t, w)
		if !reflect.DeepEqual(w.e.log, []string{"restore"}) || len(res.Notes) != 1 ||
			len(res.Findings) != 1 || !strings.HasPrefix(res.Findings[0], "no gdUnit4 test suite") {
			t.Fatalf("%v %q %q", w.e.log, res.Notes, res.Findings)
		}
	})
}

func TestFailedRestoreWinsOverABodyError(t *testing.T) {
	w := newWorld(t, withAutoload)
	w.o.Coverage = filepath.Join(w.checkout, "cov")
	os.MkdirAll(filepath.Join(w.stateDir(), "root.json"), 0o755) // SaveTimes fails
	w.e.result = map[string]child.Result{"restore": {Code: 1}}
	res, err := Run(w.o)
	if err != nil || !reflect.DeepEqual(res.Findings, []string{stillInstrumented}) {
		t.Fatalf("%v %q", err, res.Findings)
	}
}

func TestCopyingALCOVThatCannotBeStoredIsAFinding(t *testing.T) {
	w := newWorld(t, plain)
	w.o.Coverage = filepath.Join(w.checkout, "cov")
	os.MkdirAll(filepath.Join(w.o.Coverage, "0.info"), 0o755)
	res := run(t, w)
	if len(res.Findings) != 1 || !strings.HasPrefix(res.Findings[0], "shard 0 coverage report cannot be stored") {
		t.Fatalf("%q", res.Findings)
	}
}

func TestDefaultsRunTheRealChildAndClock(t *testing.T) {
	o := Options{Root: t.TempDir(), Godot: filepath.Join(t.TempDir(), "no-such-binary"), RestoreOnly: true}
	if o.now().IsZero() {
		t.Fatal("clock")
	}
	res, err := Run(o)
	if err != nil || !reflect.DeepEqual(res.Findings, []string{stillInstrumented}) {
		t.Fatalf("%v %q", err, res.Findings)
	}
}

// A reused --coverage directory must not keep the report of a shard an
// earlier run had and this one does not: linecov would sum it.
func TestCoverageRunClearsReportsOfAnEarlierRunAndNothingElse(t *testing.T) {
	w := newWorld(t, plain)
	w.o.Coverage = filepath.Join(w.checkout, "cov")
	write(t, filepath.Join(w.o.Coverage, "5.info"), "old")
	write(t, filepath.Join(w.o.Coverage, "notes.txt"), "mine")
	run(t, w)
	if exists(filepath.Join(w.o.Coverage, "5.info")) || !exists(filepath.Join(w.o.Coverage, "notes.txt")) {
		t.Fatal("the stale report stays or a foreign file went")
	}
}
