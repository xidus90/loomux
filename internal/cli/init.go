package cli

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/edit"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/pathkey"
	"github.com/xidus90/loomux/internal/selfupdate"
	"github.com/xidus90/loomux/internal/setup"
	"github.com/xidus90/loomux/internal/swap"
)

const initUsage = "usage: loomux init [--root DIR] [--dry-run] [--detect-only] [--yes] " +
	"[--hooks=all|each|none] [--brain=all|each|none] [--graph=all|each|none] [--hosts=claude,antigravity]"

// noTerminal is the refusal of a run that has to ask and cannot.
const noTerminal = "loomux init: init asks questions; run it in a terminal, or pass --yes or --dry-run"

// movedStateDir is the failure of the binary step when LOOMUX_STATE_DIR
// points elsewhere: the entries call the binary by its fixed place, not by
// the state directory, so an install there would leave them calling nothing.
const movedStateDir = `the state directory is moved by LOOMUX_STATE_DIR; host entries would call %LOCALAPPDATA%\loomux\bin\loomux.exe`

// installBinary, buildCheckout, runAction and binaryVersion are the seams
// tests replace: the real ones reach GitHub, run the Go toolchain, start qmd,
// and run the installed binary.
var (
	installBinary = installRelease
	buildCheckout = buildPilot
	runAction     = runSubcommand
	binaryVersion = installedVersion
)

// versionRunner runs the installed binary for its version; versionDeadline
// bounds that call. Every init asks, a detect-only one too, so a hung binary
// must not hold it for selfupdate's two-minute limit: past the deadline it
// has no version. Variables so that a test can hang without waiting.
var (
	versionRunner   = selfupdate.ExecRunner
	versionDeadline = 10 * time.Second
)

// installedVersion is what the binary at path names with --version, "" when
// it names none or does not answer in time.
func installedVersion(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), versionDeadline)
	defer cancel()
	if v, ok := selfupdate.InstalledVersion(ctx, versionRunner, path); ok {
		return v
	}
	return ""
}

// installRelease puts the newest release at the canonical place.
//
//coverage:exempt it reaches GitHub through gh; every test replaces installBinary
func installRelease(ctx context.Context) selfupdate.Result {
	return selfupdate.Install(ctx, selfUpdateOptions(selfupdate.SourceCLI))
}

// buildPilot builds the pilot binary of a checkout the way
// .githooks/pre-commit does: next to the running one, then swapped in.
func buildPilot(root string) error {
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		return err
	}
	build := exec.Command("go", "build", "-o", filepath.Join("bin", "loomux.new.exe"), "./cmd/loomux")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("go build: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return swap.Swap(bin)
}

// runSubcommand runs one of the commands init hands work to. It is a switch
// and not a lookup in commands: commands names initCommand, and reading it
// from here would make the package's initialization a cycle.
func runSubcommand(name string, args []string, stdout, stderr io.Writer) int {
	switch name {
	case "area":
		return areaCommand(args, nil, stdout, stderr)
	case "graph":
		return graphCommand(args, nil, stdout, stderr)
	}
	fmt.Fprintf(stderr, "loomux init: no command %s to run\n", name)
	return 2
}

// initModules are the modules init asks about, in the order it asks.
var initModules = []schema.Module{schema.Base, schema.Hooks, schema.Brain, schema.Graph}

// initOptions is a parsed command line of init.
type initOptions struct {
	root               string
	dryRun, detectOnly bool
	yes                bool
	modules            map[schema.Module]string // "all", "each" or "none"; missing when not given
	hosts              []hosts.Host
}

// parseInit reads the command line; a wrong one is reported on stderr.
func parseInit(args []string, stderr io.Writer) (initOptions, bool) {
	o := initOptions{modules: map[schema.Module]string{}}
	// The guard lets a command line through that carries --dry-run or
	// --detect-only as a word of its own; a value after = could take that
	// back, so the two never take one.
	for _, a := range args {
		name := strings.TrimLeft(a, "-")
		if name != a && (strings.HasPrefix(name, "dry-run=") || strings.HasPrefix(name, "detect-only=")) {
			fmt.Fprintf(stderr, "loomux init: %s takes no value\n%s\n", a, initUsage)
			return o, false
		}
	}
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&o.root, "root", "", "the project; the working directory when empty")
	flags.BoolVar(&o.dryRun, "dry-run", false, "show the plan and write nothing")
	flags.BoolVar(&o.detectOnly, "detect-only", false, "print what the project is, as JSON")
	flags.BoolVar(&o.yes, "yes", false, "take the defaults and approve every change")
	given := map[schema.Module]*string{}
	for _, m := range initModules[1:] {
		given[m] = flags.String(string(m), "", "all, each or none")
	}
	hostList := flags.String("hosts", "", "comma-separated hosts")
	// A flag with a value would otherwise take a following --dry-run as
	// that value, and the run the guard let through would write.
	err := refuseOptionValue(flags, args)
	if err == nil {
		err = flags.Parse(args)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux init: %v\n%s\n", err, initUsage)
		return o, false
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "loomux init: unexpected argument %q\n%s\n", flags.Arg(0), initUsage)
		return o, false
	}
	var problem string
	flags.Visit(func(f *flag.Flag) {
		if o.detectOnly && f.Name != "detect-only" && f.Name != "root" {
			problem = cmp.Or(problem, "--detect-only takes no other flag than --root")
		}
	})
	for _, m := range initModules[1:] {
		switch v := *given[m]; v {
		case "":
		case "all", "each", "none":
			o.modules[m] = v
		default:
			problem = cmp.Or(problem, fmt.Sprintf("--%s=%s: choose all, each or none", m, v))
		}
	}
	if *hostList != "" {
		for _, name := range strings.Split(*hostList, ",") {
			h, err := hosts.ParseHost(strings.TrimSpace(name))
			if err != nil {
				problem = cmp.Or(problem, fmt.Sprintf("--hosts: %v", err))
				continue
			}
			o.hosts = append(o.hosts, h)
		}
	}
	if problem != "" {
		fmt.Fprintf(stderr, "loomux init: %s\n%s\n", problem, initUsage)
		return o, false
	}
	return o, true
}

// initCommand is `loomux init`: it reads what the project is, asks what to
// set up, shows every change, and writes what the human approves.
func initCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	o, ok := parseInit(args, stderr)
	if !ok {
		return 2
	}
	// Abs fails only when the working directory is gone; the Stat below
	// then refuses the empty root.
	root, _ := filepath.Abs(cmp.Or(o.root, "."))
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		fmt.Fprintf(stderr, "loomux init: %s is no directory\n", root)
		return 2
	}
	home, _ := os.UserHomeDir()
	facts, err := setup.Gather(root, home, setup.Running{Version: Version, VersionOf: binaryVersion}, gitFacts)
	if err != nil {
		fmt.Fprintf(stderr, "loomux init: %v\n", err)
		// A file init has to merge and cannot read stops it as a plan
		// that fails does: nothing is written, and the file is named.
		if setup.Unmergeable(err) {
			return 2
		}
		return 1
	}
	if o.detectOnly {
		// Facts holds strings, booleans and lists of them, which always encode.
		data, _ := json.MarshalIndent(facts, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	answers, err := setup.ReadAnswers(root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux init: %v\n", err)
		return 2
	}
	c := setup.DefaultChoice(facts, answers)
	if len(o.hosts) > 0 {
		c.Hosts = o.hosts
	}
	for m, answer := range o.modules {
		if answer != "each" {
			switchModule(&c, facts, m, answer == "all")
		}
	}
	if !o.yes {
		if code, done := askChoice(facts, &c, o, stderr); done {
			return code
		}
	}
	p, err := setup.Build(facts, c, projectReader(root))
	if err != nil {
		fmt.Fprintf(stderr, "loomux init: %v\n", err)
		return 2
	}
	p.Notes = append(p.Notes, initNotes(facts, c, p)...)
	printPlan(stdout, p)
	if o.dryRun {
		return 0
	}
	approved := map[string]bool{}
	if o.yes {
		for _, ch := range p.Changes {
			approved[ch.Path] = true
		}
		for _, a := range p.Actions {
			approved[a.ID] = true
		}
	} else if code, done := askApproval(p, approved, stderr); done {
		return code
	}
	hooksPathApproved(p, approved)
	// Without area add the consent the merge hook waits for is not written.
	if !approved["area-add"] && !facts.HookWanted() {
		approved["merge-hook"] = false
	}
	var refused []string
	p.Actions = slices.DeleteFunc(p.Actions, func(a setup.Action) bool {
		if !approved[a.ID] {
			refused = append(refused, a.ID)
		}
		return !approved[a.ID]
	})
	run := &initRun{root: root, facts: facts, scope: c.Scope, stdout: stdout, stderr: stderr}
	present := binaryThere(root, facts)
	report, err := setup.Apply(root, p, c, func(ch setup.Change) bool { return approved[ch.Path] },
		run.action, present, Version, time.Now())
	report.Refused = append(report.Refused, refused...)
	switch {
	case run.binaryErr != nil:
		fmt.Fprintf(stderr, "loomux init: %v\n", run.binaryErr)
	case len(report.Failed) > 0 && !present():
		// The binary step was switched off or declined, and none stood there.
		fmt.Fprintln(stderr, noBinary(report.Failed))
	}
	printReport(stdout, report)
	if err != nil {
		fmt.Fprintf(stderr, "loomux init: %v\n", err)
		return 1
	}
	// What a binary switched off or declined leaves out is the human's
	// answer, not a failure; a binary step that failed is one.
	if run.binaryErr != nil {
		return 1
	}
	return 0
}

// askChoice runs the interview on the console, or keeps the defaults for a
// dry run that has none. done says the command ends with code.
func askChoice(f setup.Facts, c *setup.Choice, o initOptions, stderr io.Writer) (int, bool) {
	term, restore, err := openTerminal()
	if err != nil {
		if o.dryRun {
			return 0, false
		}
		fmt.Fprintln(stderr, noTerminal)
		return 2, true
	}
	ok, err := interview(term, f, c, o.modules)
	return endSession(restore, ok, err, stderr)
}

// askApproval asks for every change and action on the console.
func askApproval(p setup.Plan, approved map[string]bool, stderr io.Writer) (int, bool) {
	term, restore, err := openTerminal()
	if err != nil {
		fmt.Fprintln(stderr, noTerminal)
		return 2, true
	}
	ok, err := approvePlan(term, p, approved)
	return endSession(restore, ok, err, stderr)
}

// endSession hands the console back and says how a session ended: a
// cancelled one ends the run with nothing written.
func endSession(restore func() error, ok bool, err error, stderr io.Writer) (int, bool) {
	if rerr := restore(); rerr != nil {
		fmt.Fprintf(stderr, "loomux init: restoring the terminal: %v\n", rerr)
		return 1, true
	}
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(stderr, "loomux init: %v\n", err)
		return 1, true
	}
	// Cancelling is an answer, not a failure.
	if err != nil || !ok {
		fmt.Fprintln(stderr, "loomux init: cancelled; nothing written")
		return 0, true
	}
	return 0, false
}

// switchModule turns every part of m on or off, and m itself unless it is
// the base, which always runs.
func switchModule(c *setup.Choice, f setup.Facts, m schema.Module, on bool) {
	for _, p := range setup.Parts(f) {
		if p.Module == m {
			c.Parts[p.ID] = on
		}
	}
	if m != schema.Base {
		c.Modules[m] = on
	}
}

// moduleOff says whether c switches m off.
func moduleOff(c setup.Choice, m schema.Module) bool {
	on, set := c.Modules[m]
	return set && !on
}

// initNotes are what the plan cannot say itself: a module switched off
// that no configuration records, and a build no checkout can do.
func initNotes(f setup.Facts, c setup.Choice, p setup.Plan) []string {
	var notes []string
	for _, m := range initModules[1:] {
		if moduleOff(c, m) && !c.Parts["config"] {
			notes = append(notes, fmt.Sprintf("modules.%s = false is not written; the part config is off", m))
		}
	}
	for _, a := range p.Actions {
		if a.ID == "binary-build" && !f.Checkout {
			notes = append(notes, "binary-build: "+notCheckout)
		}
	}
	return notes
}

// noBinary says why a run without a binary left out what calls one, built
// from what was dropped: the merge hook calls the installed binary and
// stays in where that stands, even in a checkout without bin/loomux.exe.
func noBinary(failed []string) string {
	left := "host entries and git hooks are left out"
	if slices.Contains(failed, "merge-hook") {
		left = "host entries, git hooks and the merge hook are left out"
	}
	return "loomux init: no loomux binary where the entries call it; " + left
}

// notCheckout is why binary-build cannot run in a project that is no
// checkout of loomux but whose entries call the checkout's binary.
const notCheckout = "the host entries call ${CLAUDE_PROJECT_DIR}/bin/loomux.exe, but go.mod does not declare " +
	"github.com/xidus90/loomux, so init cannot build it here; build it, or remove the entries to get the installed binary"

// printPlan writes the summary: every change as a diff under its path, then
// the actions, then the notes.
func printPlan(w io.Writer, p setup.Plan) {
	if len(p.Changes)+len(p.Actions) == 0 {
		fmt.Fprintln(w, "nothing to change")
	}
	for _, ch := range p.Changes {
		fmt.Fprintf(w, "--- %s\n%s\n", ch.Path, strings.TrimSuffix(edit.Diff(ch.Before, ch.After), "\n"))
	}
	if len(p.Actions) > 0 {
		fmt.Fprintln(w, "actions:")
	}
	for _, a := range p.Actions {
		fmt.Fprintf(w, "  %s: %s\n", a.ID, a.Describe)
	}
	if len(p.Notes) > 0 {
		fmt.Fprintln(w, "notes:")
	}
	for _, n := range p.Notes {
		fmt.Fprintf(w, "  %s\n", n)
	}
}

// printReport writes what the run did, one line per path or action.
func printReport(w io.Writer, r setup.Report) {
	for _, group := range []struct {
		name  string
		items []string
	}{{"written", r.Written}, {"skipped", r.Skipped}, {"refused", r.Refused}, {"failed", r.Failed}, {"note", r.Notes}} {
		for _, item := range group.items {
			fmt.Fprintf(w, "%s: %s\n", group.name, item)
		}
	}
}

// projectReader reads a file of the project relative to root.
func projectReader(root string) func(string) ([]byte, bool, error) {
	return func(rel string) ([]byte, bool, error) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return data, err == nil, err
	}
}

// gitFacts runs git for Gather. An exit 1 without output is an unset
// setting, as detect.Runner asks; GIT_DIR and its kin are dropped, so a run
// inside a git hook reads the project and not the repository committing.
func gitFacts(dir string, argv ...string) (string, error) {
	command := exec.Command(argv[0], argv[1:]...)
	command.Dir = dir
	command.Env = gitenv.Environ()
	out, err := command.Output()
	var exit *exec.ExitError
	if err != nil && len(out) == 0 && errors.As(err, &exit) && exit.ExitCode() == 1 {
		return "", nil
	}
	return string(out), err
}

// binaryThere looks for the binary at the path the entries call word for
// word: the checkout's own, or the installed one under LOCALAPPDATA.
func binaryThere(root string, f setup.Facts) func() bool {
	path := setup.BinaryPath(root, f.Binary)
	return func() bool {
		info, err := os.Stat(path)
		return path != "" && err == nil && !info.IsDir()
	}
}

// initRun runs the actions of a plan.
type initRun struct {
	root, scope    string
	facts          setup.Facts
	stdout, stderr io.Writer
	binaryErr      error // Apply reports the binary step failed and goes on; init says why
}

// action is the setup.Runner of init.
func (r *initRun) action(a setup.Action) error {
	switch a.ID {
	case "binary-install":
		r.binaryErr = installStep()
		return r.binaryErr
	case "binary-build":
		r.binaryErr = errors.New(notCheckout)
		if r.facts.Checkout {
			r.binaryErr = buildCheckout(r.root)
		}
		return r.binaryErr
	case "hooks-path":
		_, err := runGit(r.root, "config", "core.hooksPath", ".githooks")
		return err
	case "area-add":
		return r.sub("area", "add", "--path", r.root, "--scope", r.scope, "--yes")
	case "merge-hook":
		return r.mergeHook()
	case "model-pull":
		// Ctrl+C ends the download, not init: Apply notes the failure and
		// writes what is left.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		return setup.PullModel(ctx, r.facts.Model, r.stderr)
	case "graph-build":
		return r.sub("graph", "build", "--root", r.root)
	}
	return fmt.Errorf("no action %s", a.ID)
}

// mergeHook installs the post-merge hook for the areas at this root only.
// `merge-hook install` works on the whole registry, where one stale area or
// one foreign hook in another repository would fail every init; here only
// this root's lines count, and a run without one has installed nothing.
func (r *initRun) mergeHook() error {
	areas, err := config.ReadRegistry(config.StateDir())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	areas = slices.DeleteFunc(areas, func(a config.Area) bool { return !setup.SameDir(r.root, a.Path) })
	found, err := maintenance.InstallHooks(areas, config.NewArtifactLookup(), runGit)
	if err != nil {
		return err
	}
	printHookStates(r.stdout, found)
	if len(found) == 0 {
		return errors.New("no area at " + r.root + " consents with [maintenance] on_merge = true")
	}
	// The line that failed, not the first one: two areas at one root give
	// two lines, and only one of them may be refused.
	for _, one := range found {
		if maintenance.HookFailed([]maintenance.HookState{one}) {
			return fmt.Errorf("the post-merge hook of %s is %s [%s]", one.Scope, one.State, cmp.Or(one.Detail, one.Repo))
		}
	}
	return nil
}

// sub runs `loomux <args>` and turns a failing exit into an error.
func (r *initRun) sub(args ...string) error {
	if code := runAction(args[0], args[1:], r.stdout, r.stderr); code != 0 {
		return fmt.Errorf("loomux %s exited %d", strings.Join(args, " "), code)
	}
	return nil
}

// installStep installs the machine-wide binary, but only where the entries
// will look for it.
func installStep() error {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" || !pathkey.Same(config.StateDir(), filepath.Join(local, "loomux")) {
		return errors.New(movedStateDir)
	}
	res := installBinary(context.Background())
	switch res.Outcome {
	case selfupdate.Current, selfupdate.Updated:
		return nil
	case selfupdate.Failed:
		return fmt.Errorf("installing loomux: %v; run `gh auth login`, then `loomux upgrade`", res.Err)
	}
	return fmt.Errorf("installing loomux: %s: %v", res.Outcome, res.Err)
}
