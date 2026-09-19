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
	"github.com/xidus90/loomux/internal/detect"
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
// binary for {loomux} and tells the time. The last two stand in for what no
// world can provoke: presets that fail to load and a plan that fails.
var (
	checkStart      = child.Run
	checkLook       = exec.LookPath
	checkExecutable = os.Executable
	checkNow        = time.Now
	checkPresets    = verify.LoadPresets
	checkPlan       = verify.Plan
)

func checkCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "loomux check: name a profile or kinds (lint,types,test,coverage), or one of: commit-msg, gofmt, gocover")
		return 2
	}
	switch args[0] {
	case "commit-msg":
		return checkCommitMsg(args[1:], stdout, stderr)
	case "gocover":
		return checkGocover(args[1:], stdout, stderr)
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
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "loomux check: one profile or list of kinds only, got also %q\n", fs.Args())
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
	eff, kinds, err := checkLoad(root, request)
	if err != nil {
		return fail(err)
	}
	loomux, err := checkExecutable()
	if err != nil {
		loomux = "loomux"
	}
	runID := verify.NewRunID(checkNow(), os.Getpid())
	env := verify.PlanEnv{Root: root, Loomux: loomux, RunID: runID, HasTests: verify.HasTests, ImportReady: verify.ImportReady}
	if *show {
		verify.WriteShow(stdout, eff, kinds, env)
		return 0
	}
	if err := verify.PrepareCover(root); err != nil {
		return fail(err)
	}
	jobs, err := checkPlan(eff, verify.Request{Kinds: kinds, Scope: verify.ScopeCheck}, env)
	if err != nil {
		return fail(err)
	}
	outs := verify.Run(jobs, verify.RunOptions{
		Scope: verify.ScopeCheck, MaxParallel: eff.Config.MaxParallel, Timeout: eff.Config.Timeout,
		Start: checkStart, Look: checkLook, Now: checkNow,
	})
	verify.WriteCheck(stdout, outs, *verbose)
	code, notes := verify.CheckVerdict(kinds, outs)
	for _, note := range notes {
		fmt.Fprintln(stdout, note)
	}
	// A file left behind costs disk, not correctness: the verdict stands.
	if err := verify.CleanCover(root, runID, code == 0); err != nil {
		fmt.Fprintf(stderr, "loomux check: cleaning coverage files: %v\n", err)
	}
	return code
}

// checkLoad lays the project's config over the presets for what root holds,
// and turns the request into kinds.
func checkLoad(root, request string) (verify.Effective, []string, error) {
	cfg, err := verify.ReadConfig(root)
	if err != nil {
		return verify.Effective{}, nil, err
	}
	presets, err := checkPresets()
	if err != nil {
		return verify.Effective{}, nil, err
	}
	eff, err := verify.Resolve(cfg, presets, detect.Detect(os.DirFS(root)))
	if err != nil {
		return verify.Effective{}, nil, err
	}
	kinds, err := verify.ExpandProfile(cfg, request)
	if err != nil {
		return verify.Effective{}, nil, err
	}
	return eff, kinds, nil
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
