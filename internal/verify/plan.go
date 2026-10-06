package verify

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/shellwords"
)

// Scope says who asked: `loomux check` for the whole project, or post-edit
// for one file.
type Scope int

const (
	ScopeCheck Scope = iota
	ScopeEdit
)

// Request is what a run should check. File is relative to the repository
// root with forward slashes, and only the edit scope has one.
type Request struct {
	Kinds []string
	Scope Scope
	File  string
}

// State is how a lane ended, or why it never started.
type State string

const (
	StateOK            State = "ok"
	StateFailed        State = "failed"
	StateTimedOut      State = "timed-out"
	StateBudget        State = "budget"
	StateBlocked       State = "blocked"
	StateMissingTool   State = "missing-tool"
	StateUnready       State = "unready"
	StateUnavailable   State = "unavailable"
	StateNotApplicable State = "not-applicable"
)

// Job is one lane of a run: a kind of one stack in one area. After is the
// index of the job it waits for, or -1. Pre is a state the plan already
// decided, so the job never starts. Reads are the coverage files it expects
// a predecessor or its Measure step to have written.
type Job struct {
	Name, Kind, Stack, Area, Origin string
	Dir                             string
	Argvs                           [][]string
	Threaded                        bool
	Measure                         []string
	After                           int
	Env                             []string
	Pre                             State
	Note                            string
	Fn                              func() (string, error)
	Reads                           []string
}

// PlanEnv is what a plan needs from outside: where it runs, which binary
// {loomux} names, and the three questions that touch the disk.
type PlanEnv struct {
	Root, Loomux, RunID string
	HasTests            func(root string, patterns []string) bool
	ImportReady         func(dir string) bool
	// GraphReady says whether the graph lanes can mean anything at root, and
	// why not. nil means they cannot.
	GraphReady func(root string) (bool, string)
	// GraphEnv is the extra environment of the graph job: a child's
	// environment loses the git pointers a surrounding hook exports, and
	// the lane needs the index that hook hands in. nil means none.
	GraphEnv func(root string) []string
}

// ImportReady says whether Godot has imported the project in dir: only then
// does it know the global classes a headless test run resolves.
func ImportReady(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".godot", "global_script_class_cache.cfg"))
	return err == nil
}

// Strict says whether request names its kinds, so that each must have had
// something to check (CheckVerdict): a list of kinds, or a profile the
// project's file sets. `all` and a built-in profile name none -- they are
// what loomux runs, not what the project asked for.
func Strict(cfg Config, request string) bool {
	if _, ok := cfg.Profiles[request]; ok {
		return cfg.SetProfiles[request]
	}
	return request != "all"
}

// ExpandProfile turns what a user asked for into kinds: all of them, a
// profile, or a comma list of kinds, each named once in the order given.
func ExpandProfile(cfg Config, request string) ([]string, error) {
	if request == "all" {
		return Kinds(), nil
	}
	if kinds, ok := cfg.Profiles[request]; ok {
		return slices.Clone(kinds), nil
	}
	out := []string{}
	for _, part := range strings.Split(request, ",") {
		part = strings.TrimSpace(part)
		if part == "" || slices.Contains(out, part) {
			continue
		}
		if !slices.Contains(Kinds(), part) {
			return nil, fmt.Errorf("unknown check %q; kinds: %s; profiles: %s", part,
				strings.Join(Kinds(), ", "), strings.Join(slices.Sorted(maps.Keys(cfg.Profiles)), ", "))
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%q names no check", request)
	}
	return out, nil
}

// target is one stack and the areas a plan runs it in.
type target struct {
	stack string
	areas []string
}

// link is what a job needs to settle its after, which can only happen once
// every job exists. Hidden are files the job reads without naming them, so
// they find a writer but are not checked for. Measuring says the job runs
// its measuring form in this run, which makes it a writer of its stack's
// coverage files.
type link struct {
	after, measure string
	reads, hidden  []string
	repl           *strings.Replacer
	measuring      bool
}

// Plan lays out the lanes of a run: kinds as requested, stacks in byte
// order, areas in byte order. Edges come only from after; what cannot run
// is decided here and carried as Pre. The graph kind runs at most once: the
// highest-ranked stack with a graph command (carrierRank) carries it, wherever
// it stands in the walk, and every other stack with one stands aside with a
// note naming that stack -- unless one stack's table
// switches the graph off, which switches it off for the whole project.
func Plan(eff Effective, req Request, env PlanEnv) ([]Job, error) {
	jobs := []Job{}
	links := []link{}
	for _, kind := range req.Kinds {
		// carrier is the stack whose graph job this run plans; only the graph
		// kind sets it.
		carrier, off := "", ""
		if kind == "graph" {
			off, carrier = graphOff(eff, req), graphCarrier(eff, req)
		}
		for _, t := range targets(eff, req) {
			areas := t.areas
			if kind == "graph" {
				// The graph belongs to the root: one job, not one rebuild per
				// area, nor one per stack with a graph lane.
				areas = []string{"."}
			}
			for _, area := range areas {
				note := ""
				switch {
				case off != "":
					note = "graph switched off under [verify." + off + "]"
				case kind == "graph" && carrier != "" && carrier != t.stack && hasCommand(eff, req, t.stack, kind):
					note = "graph covered by graph/" + carrier
				}
				if note != "" {
					job := baseJob(eff, env, kind, t.stack, area)
					job.Pre, job.Note = StateNotApplicable, note
					jobs = append(jobs, job)
					links = append(links, link{})
					continue
				}
				job, l, ok, err := planJob(eff, req, env, kind, t.stack, area)
				if err != nil {
					return nil, err
				}
				if ok {
					jobs = append(jobs, job)
					links = append(links, l)
				}
			}
		}
	}
	for i := range jobs {
		if err := settle(jobs, links, i, req); err != nil {
			return nil, err
		}
	}
	return jobs, nil
}

// carrierRank lists the stacks that carry the graph ahead of the others,
// highest first: the first of them with a graph command carries it. A stack
// outside the list ranks after all of them, and among those the byte order
// decides.
var carrierRank = []string{"go", "python", "gdscript"}

// graphCarrier is the stack whose graph job a check plans: the highest-ranked
// active stack with a graph command. "" in an edit, which plans no graph job
// and so has no carrier, and when no stack has a graph command.
func graphCarrier(eff Effective, req Request) string {
	if req.Scope != ScopeCheck {
		return ""
	}
	carrier, best := "", len(carrierRank)+1
	for _, t := range targets(eff, req) {
		if !hasCommand(eff, req, t.stack, "graph") {
			continue
		}
		rank := slices.Index(carrierRank, t.stack)
		if rank < 0 {
			rank = len(carrierRank)
		}
		if rank < best {
			carrier, best = t.stack, rank
		}
	}
	return carrier
}

// graphOff is the first active stack, in byte order, whose table switches the
// graph lane off with `graph = false`, when some active stack would still
// carry the graph without it. "" otherwise: when no table switches it off,
// in an edit, which plans no graph job anyway, and when no stack is left
// with a graph command -- the lanes then say "no command" as they did before
// a second stack could carry the graph, and a Go repository with `graph =
// false` reads as it always has. A switch under a stack the project does not
// have is none of its graph's business.
func graphOff(eff Effective, req Request) string {
	if req.Scope != ScopeCheck {
		return ""
	}
	off, carried := "", false
	for _, t := range targets(eff, req) {
		if off == "" && eff.Config.Stacks[t.stack]["graph"].Lane.Off {
			off = t.stack
		}
		carried = carried || hasCommand(eff, req, t.stack, "graph")
	}
	if !carried {
		return ""
	}
	return off
}

// targets are every active stack in every area for a check. For an edit they
// are the file's stack in the one area holding the file, and beside it the
// project lanes; a file no active stack claims gets nothing, not even those.
func targets(eff Effective, req Request) []target {
	out := []target{}
	fileStack := eff.Extensions[strings.ToLower(filepath.Ext(req.File))]
	if req.Scope == ScopeEdit && !slices.Contains(eff.Active, fileStack) {
		return out
	}
	for _, stack := range slices.Sorted(slices.Values(eff.Active)) {
		switch {
		case req.Scope == ScopeCheck:
			out = append(out, target{stack, slices.Sorted(slices.Values(eff.Areas[stack]))})
		case stack == fileStack:
			out = append(out, target{stack, []string{editArea(eff.Areas[stack], req.File)}})
		case stack == "project":
			out = append(out, target{stack, []string{"."}})
		}
	}
	return out
}

// editArea is the deepest area holding file, the root if none does.
func editArea(areas []string, file string) string {
	best := ""
	for _, a := range areas {
		if a != "." && strings.HasPrefix(file, a+"/") && len(a) > len(best) {
			best = a
		}
	}
	if best == "" {
		return "."
	}
	return best
}

// commandsFor picks what a lane runs in a scope: an edit prefers the form
// for one file, and a project lane has nothing to run on one file without it.
func commandsFor(req Request, stack string, lane Lane) []string {
	if req.Scope == ScopeCheck {
		return lane.Commands
	}
	if len(lane.OnFile) > 0 {
		return lane.OnFile
	}
	if stack == "project" {
		return nil
	}
	return lane.Commands
}

// hasCommand says whether a stack's lane of kind has something to run in the
// request's scope.
func hasCommand(eff Effective, req Request, stack, kind string) bool {
	r := eff.Stacks[stack][kind]
	return r.Defined && len(commandsFor(req, stack, r.Lane)) > 0
}

// baseJob is a lane before the plan has decided anything about it. The graph
// kind runs once at the root, so its name carries no area.
func baseJob(eff Effective, env PlanEnv, kind, stack, area string) Job {
	r := eff.Stacks[stack][kind]
	job := Job{Name: kind + "/" + stack, Kind: kind, Stack: stack, Area: area, Origin: r.Origin,
		Dir: filepath.Join(env.Root, area), Threaded: r.Lane.Threaded, After: -1}
	if len(eff.Areas[stack]) > 1 && kind != "graph" {
		job.Name += "@" + area
	}
	return job
}

func planJob(eff Effective, req Request, env PlanEnv, kind, stack, area string) (Job, link, bool, error) {
	r := eff.Stacks[stack][kind]
	job := baseJob(eff, env, kind, stack, area)
	dir := job.Dir
	// A graph lane rebuilds the graph, which no edit should wait for.
	if kind == "graph" && req.Scope == ScopeEdit {
		return Job{}, link{}, false, nil
	}
	if !hasCommand(eff, req, stack, kind) {
		if req.Scope == ScopeEdit {
			return Job{}, link{}, false, nil
		}
		job.Pre, job.Note = StateNotApplicable, "no command"
		return job, link{}, true, nil
	}
	cmds := commandsFor(req, stack, r.Lane)
	// Asked only for a stack with a graph lane: the probe costs git calls.
	if kind == "graph" {
		ready, note := false, "graph lanes need a graph probe"
		if env.GraphReady != nil {
			ready, note = env.GraphReady(env.Root)
		}
		if !ready {
			job.Pre, job.Note = StateNotApplicable, note
			return job, link{}, true, nil
		}
	}
	measured := kind == "test" || kind == "coverage"
	patterns := eff.TestsWhen[stack]
	if measured && len(patterns) > 0 && !eff.Configured[stack] && !env.HasTests(dir, patterns) {
		job.Pre, job.Note = StateUnavailable, "no tests found"
		return job, link{}, true, nil
	}
	if measured && stack == "gdscript" && eff.Config.ImportCheck && !env.ImportReady(dir) {
		job.Pre, job.Note = StateUnready, "run the Godot editor once to import the project"
		return job, link{}, true, nil
	}
	// Needs guard the whole-project commands: the form for one file reads
	// that file, not a build tree.
	needs := r.Lane.Needs
	if req.Scope == ScopeEdit && len(r.Lane.OnFile) > 0 {
		needs = nil
	}
	for _, need := range needs {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(need))); err != nil {
			job.Pre, job.Note = StateUnready, need+" is missing: configure the build first"
			return job, link{}, true, nil
		}
	}
	measuring := kind == "test" && r.Lane.Measuring != "" && slices.Contains(req.Kinds, "coverage")
	if measuring {
		cmds = []string{r.Lane.Measuring}
	}
	profile, data := CoverPaths(env.Root, env.RunID, stack, area)
	pairs := []string{"{area}", dir, "{coverprofile}", profile, "{coverdata}", data, "{loomux}", env.Loomux}
	if req.Scope == ScopeEdit {
		file := req.File
		if area != "." {
			file = strings.TrimPrefix(file, area+"/")
		}
		pairs = append(pairs, placeholderFile, file)
	}
	l := link{after: r.Lane.After, measure: r.Lane.Measure, repl: strings.NewReplacer(pairs...), measuring: measuring}
	for _, c := range cmds {
		argv, err := expand(c, l.repl)
		if err != nil {
			return Job{}, link{}, false, err
		}
		job.Argvs = append(job.Argvs, argv)
	}
	if slices.ContainsFunc(cmds, namesProfile) {
		l.reads = append(l.reads, profile)
	}
	if slices.ContainsFunc(cmds, func(c string) bool { return strings.Contains(c, "{coverdata}") }) {
		l.reads = append(l.reads, data)
	}
	// coverage.py's reporting finds its data file through the environment,
	// so a Python coverage lane that reports with it reads the file whether
	// or not its command names it. Any other lane reads nothing from there.
	if stack == "python" {
		job.Env = []string{"COVERAGE_FILE=" + data}
		if kind == "coverage" && reportsWithCoveragePy(job.Argvs) {
			l.hidden = []string{data}
		}
	}
	if kind == "graph" && env.GraphEnv != nil {
		job.Env = append(job.Env, env.GraphEnv(env.Root)...)
	}
	return job, l, true, nil
}

// reportsWithCoveragePy says whether some argv runs coverage.py's reporting:
// the word coverage, and after it in the same argv one of its report commands.
func reportsWithCoveragePy(argvs [][]string) bool {
	return slices.ContainsFunc(argvs, func(argv []string) bool {
		i := slices.Index(argv, "coverage")
		return i >= 0 && slices.ContainsFunc(argv[i+1:], func(a string) bool {
			return slices.Contains([]string{"report", "xml", "json", "html", "lcov"}, a)
		})
	})
}

// expand splits a command and fills the placeholders of each argument; the
// argv is new, so the lane it came from stays as the presets hold it.
func expand(command string, repl *strings.Replacer) ([]string, error) {
	argv, err := shellwords.Split(command)
	if err != nil {
		return nil, err
	}
	for i, a := range argv {
		argv[i] = repl.Replace(a)
	}
	return argv, nil
}

// settle ties a job to the predecessor its after names, when that runs in
// this plan and writes what the job reads. Otherwise the job measures by
// itself, and without a measure step a job that reads coverage has nothing
// to read. A predecessor switched off counts as not requested: it would
// measure nothing, while one that found no tests still hands on its state.
func settle(jobs []Job, links []link, i int, req Request) error {
	job, l := &jobs[i], links[i]
	if job.Pre != "" || l.after == "" {
		return nil
	}
	note := "`" + l.after + "` did not run and there is no measure step"
	needs := slices.Concat(l.reads, l.hidden)
	if slices.Contains(req.Kinds, l.after) {
		for j, p := range jobs {
			if p.Kind != l.after || p.Stack != job.Stack || p.Area != job.Area || p.Pre == StateNotApplicable {
				continue
			}
			if p.Pre != "" || writesAny(p, links[j], needs) {
				job.After, job.Reads = j, l.reads
				return nil
			}
			note = "`" + l.after + "` does not write what this lane reads and there is no measure step"
		}
	}
	if l.measure != "" {
		argv, err := expand(l.measure, l.repl)
		if err != nil {
			return err
		}
		job.Measure, job.Reads = argv, l.reads
		return nil
	}
	if len(needs) > 0 {
		job.Pre, job.Note = StateFailed, note
	}
	return nil
}

// writesAny says whether a planned job writes one of paths: it runs its
// measuring form, or its argv names the path. The environment does not
// count, since every Python lane carries COVERAGE_FILE, measuring or not.
// With nothing to read, any predecessor will do: the order is all it gives.
func writesAny(p Job, pl link, paths []string) bool {
	if len(paths) == 0 || pl.measuring {
		return true
	}
	args := slices.Concat(p.Argvs...)
	return slices.ContainsFunc(paths, func(path string) bool {
		return slices.ContainsFunc(args, func(a string) bool { return strings.Contains(a, path) })
	})
}
