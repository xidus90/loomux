package godotsuite

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

// Options is one godot-suite run.
type Options struct {
	Root         string // the Godot project (area), absolute
	CheckoutRoot string // where .loomux/state lives
	Area         string // slash path of Root under CheckoutRoot, "." for the root
	Godot        string // binary
	Coverage     string // "" or the directory for <i>.info
	RestoreOnly  bool
	RunID        string
	CPUs         int
	GOOS         string // "" means runtime.GOOS
	Start        func(child.Spec) child.Result
	Now          func() time.Time
}

// Result is what a run found and what it wants the reader to know.
type Result struct{ Findings, Notes []string }

const (
	gdunitCmd         = "addons/gdUnit4/bin/GdUnitCmdTool.gd"
	stillInstrumented = "sources are still instrumented; run loomux check godot-suite --restore"
	staleAfter        = 24 * time.Hour
)

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o Options) start(s child.Spec) child.Result {
	if o.Start != nil {
		return o.Start(s)
	}
	return child.Run(s)
}

func outputOf(r child.Result) string {
	if r.Err != nil {
		return r.Err.Error()
	}
	return r.Stdout + r.Stderr
}

// instrument runs disk.gd, which rewrites the sources; it exits non-zero for
// anything but status ok, so the exit code is the verdict.
func (o Options) instrument() (ok bool, r child.Result) {
	r = o.start(child.Spec{Dir: o.Root, Argv: []string{o.Godot, "--path", o.Root, "--headless",
		"-s", "res://" + GlueDir + "/disk.gd"}})
	return r.Err == nil && r.Code == 0, r
}

// putBack undoes a disk instrumentation from the addon's manifest and names
// what stays instrumented afterwards. No engine is started: this is the
// restore of a run that was killed while the sources were rewritten, and a
// killed run leaves no process to start one.
func (o Options) putBack() ([]string, error) {
	findings, err := RestoreManifest(o.Root)
	if err != nil {
		return nil, err
	}
	left, err := Instrumented(o.Root)
	if err != nil {
		return nil, err
	}
	if len(left) > 0 {
		findings = append(findings, stillInstrumented)
	}
	return findings, nil
}

// prechecks are the findings that stop the run before any engine starts. A
// registered coverage hook counts in every run: in a plain one it still
// instruments in memory and lets every shard write into the working tree.
func (o Options) prechecks() ([]string, error) {
	var f []string
	if _, err := os.Stat(filepath.Join(o.Root, filepath.FromSlash(gdunitCmd))); err != nil {
		f = append(f, "gdUnit4 is missing: "+gdunitCmd)
	}
	f = append(f, ClassCacheFindings(o.Root, o.Area)...)
	conflict, err := HookConflict(o.Root)
	if err != nil {
		return nil, err
	}
	if conflict != "" {
		f = append(f, conflict)
	}
	return f, nil
}

// Run follows the flow of the design: prechecks, suites, plan, glue, the
// optional disk bracket, the shards side by side, verdict, timings.
func Run(o Options) (res Result, err error) {
	if o.RestoreOnly {
		res.Findings, err = o.putBack()
		return res, err
	}
	// Before every other check: sources the user can see must not stay
	// instrumented because a later check is red.
	if err = WriteGlue(o.Root); err != nil {
		return res, err
	}
	left, err := Instrumented(o.Root)
	if err != nil {
		return res, err
	}
	if len(left) > 0 {
		if res.Findings, err = o.putBack(); err != nil || len(res.Findings) > 0 {
			return res, err
		}
		res.Notes = append(res.Notes, "restored a leftover disk instrumentation")
	}
	if res.Findings, err = o.prechecks(); err != nil || len(res.Findings) > 0 {
		return res, err
	}
	suites, err := Suites(o.Root)
	if err != nil {
		return res, err
	}
	if len(suites) == 0 {
		res.Findings = []string{"no gdUnit4 test suite under " + o.Root}
		return res, nil
	}

	body := func() error { return o.shards(&res, suites) }
	if o.Coverage != "" {
		if err = os.MkdirAll(o.Coverage, 0o755); err != nil {
			return res, err
		}
		// A reused directory may hold the reports of an earlier run with
		// more shards, which linecov would sum.
		old, _ := os.ReadDir(o.Coverage)
		for _, e := range old {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".info" {
				os.Remove(filepath.Join(o.Coverage, e.Name()))
			}
		}
		onDisk, err := NeedsDisk(o.Root)
		if err != nil {
			return res, err
		}
		if onDisk {
			err = o.bracket(&res, body)
			return res, err
		}
	}
	err = body()
	return res, err
}

// bracket instruments the sources, runs body and puts the sources back
// whatever body did, so a dead shard or an error cannot skip the restore. A
// kill of this process can; putBack in the next run is what covers that.
func (o Options) bracket(res *Result, body func() error) (err error) {
	_, statErr := os.Stat(filepath.Join(o.Root, "coverage-data"))
	hadData := statErr == nil
	snap, err := Take(o.Root)
	if err != nil {
		return err
	}
	ok, r := o.instrument()
	defer func() {
		findings, restoreErr := o.putBack()
		if restoreErr != nil {
			findings = []string{"putting the sources back: " + restoreErr.Error()}
		}
		if len(findings) > 0 {
			// The findings are what the CLI prints; an error would hide them.
			res.Findings, err = findings, nil
			return
		}
		// The engine creates the store when it is missing and never removes it.
		if !hadData {
			os.RemoveAll(filepath.Join(o.Root, "coverage-data"))
		}
		res.Findings = append(res.Findings, snap.Settle(o.Root)...)
	}()
	if !ok {
		res.Findings = []string{fmt.Sprintf("disk instrumentation failed (exit %d):\n%s", r.Code, outputOf(r))}
		return nil
	}
	return body()
}

func (o Options) runShard(s *Shard) {
	argv := []string{o.Godot, "--path", o.Root, "-s", "-d", "--remote-debug", "tcp://127.0.0.1:0",
		"res://" + GlueDir + "/run_GdUnitCmdTool.gd"}
	for _, suite := range s.Suites {
		argv = append(argv, "-a", suite)
	}
	// Relative to the project, which is the engine's working directory:
	// gdUnit4 turns a path with a leading slash into one under res://.
	reports := filepath.Join(s.Dir, "reports")
	if rel, err := filepath.Rel(o.Root, reports); err == nil {
		reports = filepath.ToSlash(rel)
	}
	argv = append(argv, "-c", "--headless", "--ignoreHeadlessMode", "-rd", reports)
	// Both variables: the engine reads the one its platform uses, and a
	// fresh one apart keeps user:// of the shards apart.
	userdata := filepath.Join(s.Dir, "userdata")
	env := []string{"APPDATA=" + userdata, "XDG_DATA_HOME=" + userdata}
	if o.Coverage != "" {
		env = append(env, "LOOMUX_COVERAGE_SHARD_DIR="+filepath.Join(s.Dir, "coverage"))
	}
	r := o.start(child.Spec{Argv: argv, Dir: o.Root, Env: env})
	s.Code, s.Output = r.Code, outputOf(r)
}

// collect copies the LCOV of a shard to <Coverage>/<i>.info.
func (o Options) collect(s Shard) []string {
	data, err := os.ReadFile(filepath.Join(s.Dir, "coverage", "report", "lcov.info"))
	if err != nil {
		return []string{fmt.Sprintf("shard %d wrote no coverage report", s.Index)}
	}
	if err = os.WriteFile(filepath.Join(o.Coverage, strconv.Itoa(s.Index)+".info"), data, 0o644); err != nil {
		return []string{fmt.Sprintf("shard %d coverage report cannot be stored: %v", s.Index, err)}
	}
	return nil
}

// shards plans, runs and judges the suites and files the timings.
func (o Options) shards(res *Result, suites []string) error {
	state := filepath.Dir(TimesPath(o.CheckoutRoot, o.Area))
	times, note := LoadTimes(TimesPath(o.CheckoutRoot, o.Area))
	if note != "" {
		res.Notes = append(res.Notes, note)
	}
	plan := Plan(suites, times, ShardCount(len(suites), o.CPUs))
	goos := o.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	if goos == "darwin" && len(plan) > 1 {
		res.Notes = append(res.Notes, "on macOS the shards share user://")
	}

	runDir := filepath.Join(state, o.RunID+"-"+areaName(o.Area))
	shards := make([]Shard, len(plan))
	for i, s := range plan {
		shards[i] = Shard{Index: i, Suites: s, Dir: filepath.Join(runDir, strconv.Itoa(i))}
		if err := os.MkdirAll(shards[i].Dir, 0o755); err != nil {
			return err
		}
	}
	var wg sync.WaitGroup
	for i := range shards {
		wg.Go(func() { o.runShard(&shards[i]) })
	}
	wg.Wait()

	update := map[string]float64{}
	var findings []string
	for _, s := range shards {
		f, t := Verdict(s)
		findings = append(findings, f...)
		maps.Copy(update, t)
		if o.Coverage != "" {
			findings = append(findings, o.collect(s)...)
		}
	}
	res.Findings = append(res.Findings, findings...)
	if len(update) > 0 {
		if err := SaveTimes(TimesPath(o.CheckoutRoot, o.Area), update); err != nil {
			return err
		}
	}
	// Findings of the disk bracket arrive later, but a shard finding is
	// enough to keep the directory for the investigation.
	if len(findings) == 0 {
		os.RemoveAll(runDir)
	}
	o.sweep(state, runDir)
	return nil
}

// sweep removes the shard directories of runs older than a day; the timings
// files beside them are files and stay.
func (o Options) sweep(state, keep string) {
	entries, _ := os.ReadDir(state)
	for _, e := range entries {
		p := filepath.Join(state, e.Name())
		if !e.IsDir() || p == keep {
			continue
		}
		if fi, err := e.Info(); err == nil && o.now().Sub(fi.ModTime()) > staleAfter {
			os.RemoveAll(p)
		}
	}
}
