package cli

import (
	"bytes"
	"cmp"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/hooks"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/verify"
	"github.com/xidus90/loomux/internal/verify/commit"
	"github.com/xidus90/loomux/internal/verify/gocover"
)

var coverFunc = runCoverFunc

// runCoverFunc keeps the tool's stderr in the error: exec alone would report
// only an exit status, never why the profile was unusable. The tool runs in
// dir, so it resolves the module there and a relative profile against it.
func runCoverFunc(dir, profile string) ([]byte, error) {
	cmd := exec.Command("go", "tool", "cover", "-func="+profile)
	cmd.Dir = dir
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out, fmt.Errorf("%w: %s", err, bytes.TrimSpace(exit.Stderr))
	}
	return out, err
}

// The seams of a check: what starts a tool, finds it on the PATH, names this
// binary for {loomux} and tells the time. The others stand in for what no
// world can provoke: presets that fail to load, a plan that fails, a file
// that cannot be written, and the index a commit hook is handed, which no
// test process has.
var (
	checkStart       = child.Run
	checkLook        = exec.LookPath
	checkExecutable  = os.Executable
	checkNow         = time.Now
	checkPresets     = verify.LoadPresets
	checkPlan        = verify.Plan
	checkWriteArmed  = verify.WriteArmed
	checkCommitIndex = gitwork.CommitIndex
	checkStage       = gitwork.Stage
)

func checkCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "loomux check: name a profile or kinds (lint,types,test,coverage,graph), or one of: commit-msg, gofmt, gocover, graph-fresh, blast-audit")
		return 2
	}
	switch args[0] {
	case "commit-msg":
		return checkCommitMsg(args[1:], stdout, stderr)
	case "gocover":
		return checkGocover(args[1:], stdout, stderr)
	case "graph-fresh":
		return checkGraphFresh(args[1:], stdout, stderr)
	case "blast-audit":
		return checkBlastAudit(args[1:], stdout, stderr)
	case "gofmt":
		paths := args[1:]
		if len(paths) == 0 {
			paths = []string{"."}
		}
		unformatted, err := verify.CheckGoFormat(paths)
		if err != nil {
			fmt.Fprintf(stderr, "loomux check gofmt: %v\n", err)
			return 1
		}
		for _, file := range unformatted {
			fmt.Fprintln(stdout, file)
		}
		if len(unformatted) > 0 {
			return 1
		}
		return 0
	}
	return checkRun(args, stdout, stderr)
}

// checkGraphFresh is the first half of the graph lane: the graph must match
// the tree before blast-audit reads it.
func checkGraphFresh(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check graph-fresh", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; found upwards when empty")
	wait := fs.Duration("wait", 30*time.Second, "how long to wait for another run's rebuild")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux check graph-fresh: %v\n", err)
		return 1
	}
	status, err := query.RefreshGraph(project, *wait,
		func(s string) { fmt.Fprintf(stderr, "loomux check graph-fresh: %s\n", s) })
	if err != nil {
		fmt.Fprintf(stderr, "loomux check graph-fresh: %v\n", err)
		return 1
	}
	if status == ask.StatusRebuilt {
		fmt.Fprintln(stdout, "graph rebuilt")
	} else {
		fmt.Fprintln(stdout, "graph is fresh")
	}
	return 0
}

// checkBlastAudit is the second half of the graph lane: red when a changed
// area with enough callers has no changed test. It reads the graph as it is.
func checkBlastAudit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check blast-audit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; found upwards when empty")
	base := fs.String("base", "", "audit base...HEAD instead of the working tree")
	cached := fs.Bool("cached", false, "audit the index against HEAD")
	threshold := fs.Int("threshold", 3, "callers a seed needs before a missing test is a finding")
	skipTests := fs.Bool("skip-test-callers", false, "count only callers outside test files")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux check blast-audit: %v\n", err)
		return 1
	}
	// Audit never refreshes the graph, so it has no refresh notes to pass on.
	ans, _, err := query.Audit(project, query.AuditOptions{
		Base: *base, Cached: *cached, Threshold: *threshold, SkipTestCallers: *skipTests,
	})
	if err != nil {
		fmt.Fprintf(stderr, "loomux check blast-audit: %v\n", err)
		return 1
	}
	fmt.Fprint(stdout, query.AuditReport(ans, *threshold))
	if len(ans.Findings) > 0 {
		return 1
	}
	return 0
}

// checkRun checks a project: the request comes first and the flags after it,
// since flag stops at the first argument that is not one. A project without
// a config is still checked, by the presets alone, where the call was made.
func checkRun(args []string, stdout, stderr io.Writer) int {
	request := args[0]
	if strings.HasPrefix(request, "-") {
		fmt.Fprintf(stderr, "loomux check: name the profile or kinds before the flags, got %q\n", request)
		return 2
	}
	fs := flag.NewFlagSet("loomux check "+request, flag.ContinueOnError)
	fs.SetOutput(stderr)
	rootFlag := fs.String("root", "", "path to the project root; found upwards when empty")
	verbose := fs.Bool("v", false, "print the output of green lanes too")
	show := fs.Bool("show", false, "print what would run as a [verify] table and run nothing")
	arm := fs.Bool("arm", false, "after a green run, write every lane that ended ok into .loomux/armed.toml and stage it")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "loomux check: one profile or list of kinds only, got also %q\n", fs.Args())
		return 2
	}
	// Only the pre-commit run arms: what the file holds comes from a commit
	// that went through, and no other profile is one.
	if *arm && request != "precommit" {
		fmt.Fprintln(stderr, "loomux check: --arm belongs to the precommit profile")
		return 2
	}
	root := *rootFlag
	if root == "" {
		root, _ = hosts.FindRoot(".")
	}
	// Tools run in their area, so every path handed to them must be absolute.
	if abs, err := filepath.Abs(cmp.Or(root, ".")); err == nil {
		root = abs
	}
	fail := func(err error) int {
		fmt.Fprintf(stderr, "loomux check: %v\n", err)
		return 1
	}
	eff, kinds, facts, err := checkLoad(root, request)
	if err != nil {
		return fail(err)
	}
	loomux, err := checkExecutable()
	if err != nil {
		loomux = "loomux"
	}
	runID := verify.NewRunID(checkNow(), os.Getpid())
	env := verify.PlanEnv{Root: root, Loomux: loomux, RunID: runID, HasTests: verify.HasTests,
		ImportReady: verify.ImportReady, GraphReady: query.GraphReady, GraphEnv: query.GraphEnv}
	// The table asks no probe, so it needs no copy of the index.
	if *show {
		verify.WriteShow(stdout, eff, kinds, env)
		return 0
	}
	// Inside a commit hook git hands its index in: a lane may sit out a
	// commit of paths it names. By hand there is no index, and every lane
	// runs. An index that does not answer leaves the list empty, too.
	if index := os.Getenv("GIT_INDEX_FILE"); index != "" {
		if changed, err := gitwork.StagedPaths(root, index); err == nil {
			env.Changed = changed
		}
	}
	// check stop replays the stop gate, which judges the working tree against
	// HEAD through a copy of the index; every other request keeps the real one.
	if request == "stop" {
		idx, err := hooks.OpenStopIndex(root, kinds)
		if err != nil {
			return fail(err)
		}
		defer idx.Close()
		// Without a copy too: the hook asks the same nil view.
		env.GraphReady, env.GraphEnv = idx.Ready, idx.Env
	}
	if err := verify.PrepareCover(root); err != nil {
		return fail(err)
	}
	jobs, err := checkPlan(eff, verify.Request{Kinds: kinds, Scope: verify.ScopeCheck}, env)
	if err != nil {
		return fail(err)
	}
	// The wiki's lane is built by the hooks, like its edit lane; appended
	// behind the plan, because a job's After is an index into it.
	jobs = append(jobs, hooks.WikiGateJobs(eff, facts, root, kinds)...)
	// A file that does not read arms every lane, and every run says so.
	armed, err := verify.ReadArmed(root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux check: %v\n", err)
	}
	outs := verify.Run(jobs, verify.RunOptions{
		Scope: verify.ScopeCheck, MaxParallel: eff.Config.MaxParallel, Timeout: eff.Config.Timeout,
		Start: checkStart, Look: checkLook, Now: checkNow, Armed: armed.Arms,
	})
	verify.WriteCheck(stdout, outs, *verbose)
	code, notes := verify.CheckVerdict(kinds, outs, verify.Strict(eff.Config, request))
	for _, note := range notes {
		fmt.Fprintln(stdout, note)
	}
	// Armed only by a run that ends green as a whole, and only where the
	// project has the file: without it every lane is armed already.
	if added := armed.Missing(verify.GreenKeys(outs)); *arm && code == 0 && armed.Exists && len(added) > 0 {
		armed = checkArm(root, armed, added, stdout, stderr)
	}
	if line := verify.ProbationLine(outs, armed.Arms); line != "" {
		fmt.Fprintln(stdout, line)
	}
	// A file left behind costs disk, not correctness: the verdict stands.
	if err := verify.CleanCover(root, runID, code == 0); err != nil {
		fmt.Fprintf(stderr, "loomux check: cleaning coverage files: %v\n", err)
	}
	return code
}

// checkArm enters the lanes a green pre-commit run found ok, and lays the
// file into the commit under way; it answers the set that holds afterwards.
// Writing and staging sit here and not in the hook's script, so that one
// place knows a commit of paths and then does neither: staged into the index
// such a commit hands its hook, the file would be committed while the real
// index kept the old entry as a staged revert. The commit hangs on nothing
// here: a file that cannot be written or staged is said, and the verdict
// stands.
func checkArm(root string, armed verify.ArmedSet, added []string, stdout, stderr io.Writer) verify.ArmedSet {
	index, whole := checkCommitIndex(root, os.Getenv("GIT_INDEX_FILE"))
	if !whole {
		fmt.Fprintln(stdout, "not armed: this commit takes only some paths; the next whole commit arms the lanes")
		return armed
	}
	next := armed.With(added...)
	if err := checkWriteArmed(root, next); err != nil {
		fmt.Fprintf(stderr, "loomux check: %s not written: %v\n", verify.ArmedFile, err)
		return armed
	}
	fmt.Fprintf(stdout, "armed: %s\n", strings.Join(added, ", "))
	if err := checkStage(root, index, verify.ArmedFile); err != nil {
		fmt.Fprintf(stderr, "loomux check: %s not staged, commit it by hand: %v\n", verify.ArmedFile, err)
	}
	return next
}

// checkLoad lays the project's config over the presets for what root holds,
// and turns the request into kinds.
//
// The facts it detects go back to the caller: the wiki lane needs the same
// ones, and detecting them twice would walk the tree twice.
func checkLoad(root, request string) (verify.Effective, []string, detect.Facts, error) {
	cfg, err := verify.ReadConfig(root)
	if err != nil {
		return verify.Effective{}, nil, detect.Facts{}, err
	}
	presets, err := checkPresets()
	if err != nil {
		return verify.Effective{}, nil, detect.Facts{}, err
	}
	facts := detect.Detect(os.DirFS(root))
	eff, err := verify.Resolve(cfg, presets, facts)
	if err != nil {
		return verify.Effective{}, nil, detect.Facts{}, err
	}
	kinds, err := verify.ExpandProfile(cfg, request)
	if err != nil {
		return verify.Effective{}, nil, detect.Facts{}, err
	}
	return eff, kinds, facts, nil
}

// checkGocover judges a coverage profile of the module in --dir: without
// --floor every function at 100% or exempt, with it the total at the floor.
// Everything is read relative to --dir, so a caller can judge another module
// without changing the working directory of the whole process.
func checkGocover(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check gocover", flag.ContinueOnError)
	fs.SetOutput(stderr)
	profile := fs.String("profile", "", "coverage profile written by go test, relative to --dir")
	floor := fs.Float64("floor", 0, "minimum total coverage in percent instead of the per-function gate")
	dir := fs.String("dir", ".", "directory holding go.mod")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *profile == "" {
		fmt.Fprintln(stderr, "loomux check gocover: --profile is required")
		return 2
	}
	floorSet := false
	fs.Visit(func(f *flag.Flag) { floorSet = floorSet || f.Name == "floor" })
	fail := func(err error) int {
		fmt.Fprintf(stderr, "loomux check gocover: %v\n", err)
		return 1
	}
	gomod, err := os.ReadFile(filepath.Join(*dir, "go.mod"))
	if err != nil {
		return fail(err)
	}
	module, err := gocover.ModulePath(gomod)
	if err != nil {
		return fail(err)
	}
	out, err := coverFunc(*dir, *profile)
	if err != nil {
		return fail(err)
	}
	if floorSet {
		total, err := gocover.Total(out)
		if err != nil {
			return fail(err)
		}
		if total < *floor {
			return fail(fmt.Errorf("coverage %.1f%% is below the floor of %g%%", total, *floor))
		}
		fmt.Fprintf(stdout, "coverage %.1f%%\n", total)
		return 0
	}
	lines, err := gocover.Parse(bytes.NewReader(out))
	if err != nil {
		return fail(err)
	}
	// A gate that finds nothing to judge must not pass.
	if len(lines) == 0 {
		return fail(fmt.Errorf("no functions in %s", *profile))
	}
	read := func(p string) ([]byte, error) { return os.ReadFile(filepath.Join(*dir, p)) }
	return gocover.Gate(lines, module, read, stdout)
}

func checkCommitMsg(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("loomux check commit-msg", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var calibrate int
	var language string
	var rootFlag string

	fs.IntVar(&calibrate, "calibrate", 0, "measure the thresholds against the last N commits")
	fs.StringVar(&language, "language", "", "the language to check or calibrate against (en, de)")
	fs.StringVar(&rootFlag, "root", "", "path to project root")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	calibratePassed := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "calibrate" {
			calibratePassed = true
		}
	})

	root := rootFlag
	if root == "" {
		root, _ = hosts.FindRoot(".")
	}
	if abs, err := filepath.Abs(cmp.Or(root, ".")); err == nil {
		root = abs
	}

	if calibratePassed {
		if fs.NArg() > 0 {
			fmt.Fprintln(stderr, "loomux check commit-msg: cannot pass both a file and --calibrate")
			return 2
		}
		if calibrate < 1 {
			fmt.Fprintf(stderr, "loomux check commit-msg: --calibrate needs a count of at least 1, not %d\n", calibrate)
			return 2
		}
		if language != "" && language != "en" && language != "de" {
			fmt.Fprintf(stderr, "loomux check commit-msg: --language must be one of (\"en\", \"de\"), not %q\n", language)
			return 2
		}

		policy, err := commit.ReadPolicy(root)
		if err != nil {
			fmt.Fprintf(stderr, "loomux check commit-msg: %v\n", err)
			return 1
		}

		lang := cmp.Or(language, policy.Language, "en")

		messages, err := commit.ReadMessages(root, calibrate)
		if err != nil {
			fmt.Fprintf(stderr, "loomux check commit-msg: %v\n", err)
			return 1
		}

		commit.Render(messages, lang, commit.DefaultThresholds, stdout, policy.Allow)
		return 0
	}

	if language != "" {
		fmt.Fprintln(stderr, "loomux check commit-msg: --language cannot be used when checking a file; a commit cannot choose the rule it is judged by")
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "loomux check commit-msg: exactly one message file required")
		return 2
	}

	filePath := fs.Arg(0)
	policy, err := commit.ReadPolicy(root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux check commit-msg: %v\n", err)
		return 1
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(stderr, "loomux check commit-msg: %v\n", err)
		return 1
	}

	if err := commit.Check(string(content), policy); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
