package cli

import (
	"bytes"
	"context"
	"crypto/rand"
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
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/brain/index"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/dev/benchcorpus"
	"github.com/xidus90/loomux/internal/dev/benchhooks"
	"github.com/xidus90/loomux/internal/dev/benchreport"
	"github.com/xidus90/loomux/internal/dev/benchsearch"
	"github.com/xidus90/loomux/internal/dev/fakeollama"
	"github.com/xidus90/loomux/internal/dev/importcases"
	"github.com/xidus90/loomux/internal/dev/mutants"
	"github.com/xidus90/loomux/internal/dev/recordcase"
	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/swap"
)

var benchExec = benchhooks.Exec

var mutantsTest = mutants.GoTest

var mutantsRoot = os.Getwd

var mutantsNotify = signal.NotifyContext

// recordMCPCase is the seam of the MCP recorder. A real recording starts a
// Python reference and a daemon beside it, and no test of this command may do
// either.
var recordMCPCase = recordcase.RecordMCP

// fakeOllamaNotify is the seam of the fake Ollama's lifetime. The command
// serves until Ctrl+C, which a test on Windows cannot send; a test hands it a
// context that has already ended instead.
var fakeOllamaNotify = signal.NotifyContext

var devCommands = map[string]command{
	"bench":           devBenchGroup,
	"fake-ollama":     devFakeOllama,
	"import-cases":    devImportCases,
	"mutants":         devMutants,
	"record-case":     devRecordCase,
	"record-mcp-case": devRecordMCPCase,
	"release":         devRelease,
	"swap-binary":     devSwapBinary,
}

func devCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "loomux dev: subcommand required")
		return 2
	}
	sub, ok := devCommands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "loomux dev: unknown subcommand %q\n", args[0])
		return 2
	}
	return sub(args[1:], stdin, stdout, stderr)
}

// devMutants runs a mutation round over packages named relative to the
// working directory. Packages and flags may be mixed: Go's flag package stops
// at the first argument that is no flag, so parsing resumes behind each
// package.
func devMutants(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev mutants", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts mutants.Options
	fs.StringVar(&opts.Only, "only", "", "restrict to files whose name contains this")
	fs.StringVar(&opts.Family, "family", "", "restrict to one of a1, a2, a3, a4")
	fs.IntVar(&opts.Workers, "workers", mutants.DefaultWorkers(), "go test runs at the same time")
	for rest := args; ; rest = rest[1:] {
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		opts.Packages = append(opts.Packages, rest[0])
	}
	if len(opts.Packages) == 0 {
		fmt.Fprintln(stderr, "loomux dev mutants: at least one package is required")
		return 2
	}
	switch opts.Family {
	case "", "a1", "a2", "a3", "a4":
	default:
		fmt.Fprintf(stderr, "loomux dev mutants: --family must be one of a1, a2, a3, a4, got %q\n", opts.Family)
		return 2
	}
	if opts.Workers < 1 {
		fmt.Fprintf(stderr, "loomux dev mutants: --workers must be at least 1, got %d\n", opts.Workers)
		return 2
	}
	root, err := mutantsRoot()
	if err == nil {
		// Ctrl+C would end the process before any deferred removal of an
		// overlay directory; caught, it ends the round through its error path.
		ctx, stop := mutantsNotify(context.Background(), os.Interrupt)
		defer stop()
		opts.Root = root
		_, err = mutants.Round(opts, untilInterrupted(ctx, stop, mutantsTest(ctx, root)), stdout)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev mutants: %v\n", err)
		// A red suite or a package without sources is a wrong question, as
		// the script answers it with 2; anything else broke on the way.
		if errors.Is(err, mutants.ErrBaselineRed) || errors.Is(err, mutants.ErrNoSources) {
			return 2
		}
		return 1
	}
	return 0
}

// untilInterrupted turns an interrupt into the error of a run. A run asked
// for afterwards does not start; a run under way when it came reports the
// interrupt instead of its verdict. Round stops at the first error and waits
// for every run it started, so each overlay directory is gone before the
// command returns. Seeing the interrupt also ends the signal registration
// through stop: while it stands, a second Ctrl+C is swallowed, and the user
// has no way out of the drain.
func untilInterrupted(ctx context.Context, stop func(), test mutants.TestFunc) mutants.TestFunc {
	return func(pkg, overlay string) (mutants.Outcome, error) {
		if err := ctx.Err(); err != nil {
			stop()
			return 0, err
		}
		outcome, err := test(pkg, overlay)
		if interrupted := ctx.Err(); interrupted != nil {
			stop()
			return 0, interrupted
		}
		return outcome, err
	}
}

var benchCommands = map[string]command{
	"hooks":  devBenchHooks,
	"repos":  devBenchRepos,
	"search": devBenchSearch,
}

const benchUsage = `usage: loomux dev bench <hooks|repos|search> [flags]
  hooks   time the hook commands of a case file
  repos   time the hooks on repositories and audit their lanes
  search  measure the rank of search hits and the chain's latency
`

func devBenchGroup(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, benchUsage)
		return 2
	}
	sub, ok := benchCommands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "loomux dev bench: unknown subcommand %q\n%s", args[0], benchUsage)
		return 2
	}
	return sub(args[1:], stdin, stdout, stderr)
}

// benchHead opens the markdown of a bench run with what tells two runs apart.
func benchHead(command string, env benchreport.Environment, stamp string) string {
	return fmt.Sprintf("# loomux dev bench %s — %s\n\n- system: %s/%s, %s\n- go: %s\n- loomux: %s\n\n",
		command, stamp, env.OS, env.Arch, env.CPU, env.Go, env.Loomux)
}

// benchTargets names the two report files of a run under --out, or none
// without it. It runs before the first measurement, so a run that could not
// save does not measure for minutes first.
func benchTargets(dir, command, stamp string) (string, string, error) {
	if dir == "" {
		return "", "", nil
	}
	return benchreport.Targets(dir, "bench-"+stamp+"-"+command)
}

// devBenchHooks measures the hook commands of a case file. The file may
// stand before or after the flags: Go's flag package stops at the first
// argument that is no flag, so the rest is parsed once more behind it.
func devBenchHooks(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev bench hooks", flag.ContinueOnError)
	fs.SetOutput(stderr)
	n := fs.Int("n", 20, "warm runs per case, after one cold run")
	out := fs.String("out", "", "directory to write the markdown and JSON report to")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(stderr, "loomux dev bench hooks: a case file is required")
		return 2
	}
	if err := fs.Parse(rest[1:]); err != nil {
		return 2
	}
	if extra := fs.Args(); len(extra) > 0 {
		fmt.Fprintf(stderr, "loomux dev bench hooks: unexpected argument %q after the case file\n", extra[0])
		return 2
	}
	// A run of zero warm runs reports a row of noughts; a negative one
	// would ask for a slice of negative capacity.
	if *n < 1 {
		fmt.Fprintf(stderr, "loomux dev bench hooks: -n must be at least 1, got %d\n", *n)
		return 2
	}
	stamp := benchreport.Stamp(benchClock())
	md, js, err := benchTargets(*out, "hooks", stamp)
	var cases []benchhooks.Case
	if err == nil {
		var data []byte
		data, err = os.ReadFile(rest[0])
		if err == nil {
			err = json.Unmarshal(data, &cases)
		}
	}
	var timings []benchreport.Timing
	if err == nil {
		timings, err = benchhooks.Measure(cases, *n, benchExec, time.Now)
	}
	if err == nil {
		env := benchreport.Current(Version)
		text := benchHead("hooks", env, stamp) + benchhooks.Table(timings)
		fmt.Fprint(stdout, text)
		if md != "" {
			// Measured durations are finite, so the report always encodes.
			payload, _ := benchreport.Report{Schema: benchreport.Schema, Command: "hooks", Stamp: stamp,
				Environment: env, Timings: timings}.JSON()
			err = benchWriteBoth(md, js, []byte(text), payload)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev bench hooks: %v\n", err)
		return 1
	}
	return 0
}

// envFlags collects a KEY=VALUE flag that may be given more than once.
type envFlags []string

func (e *envFlags) String() string { return strings.Join(*e, " ") }

func (e *envFlags) Set(value string) error {
	if !strings.Contains(value, "=") {
		return fmt.Errorf("%q is not KEY=VALUE", value)
	}
	*e = append(*e, value)
	return nil
}

func devRecordCase(args []string, _ io.Reader, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev record-case", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var s recordcase.Spec
	var argv string
	var env envFlags
	fs.StringVar(&s.Exe, "exe", "", "path of the old binary")
	fs.StringVar(&argv, "argv", "", "program and leading arguments in place of the command's first token")
	fs.Var(&env, "env", "KEY=VALUE for the recorded process, {{WORLD}} allowed; repeatable")
	fs.StringVar(&s.PathPrepend, "path-prepend", "", "directory put in front of the recorded process's PATH")
	fs.StringVar(&s.Cmd, "cmd", "", "command line with {{WORLD}}")
	fs.StringVar(&s.World, "world", "", "directory to stage")
	fs.StringVar(&s.Stdin, "stdin", "", "file with the payload")
	fs.StringVar(&s.Out, "out", "", "case directory to write")
	fs.StringVar(&s.Notes, "notes", "", "text for notes.md")
	fs.StringVar(&s.Compare, "compare", "", `"" (data) or "message"`)
	fs.BoolVar(&s.GitAfter, "git-after", false, "pin the commit the run made in git.after of the git world's repository")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if s.Exe != "" && argv != "" {
		fmt.Fprintln(stderr, "loomux dev record-case: --exe and --argv exclude each other")
		return 2
	}
	tokens, err := cases.SplitCommand(argv)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev record-case: --argv: %v\n", err)
		return 2
	}
	s.Argv, s.Env = tokens, env
	if (s.Exe == "" && len(s.Argv) == 0) || s.Cmd == "" || s.World == "" || s.Out == "" {
		fmt.Fprintln(stderr, "loomux dev record-case: --exe or --argv, --cmd, --world and --out are required")
		return 2
	}
	if err := recordcase.Record(s); err != nil {
		fmt.Fprintf(stderr, "loomux dev record-case: %v\n", err)
		return 1
	}
	return 0
}

// devRecordMCPCase records one call of the reference's MCP front. It is the
// second recorder rather than a flag on the first, because what it pins is a
// tool call and a CallToolResult, not a command line and a stream of stdout.
func devRecordMCPCase(args []string, _ io.Reader, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev record-mcp-case", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var s recordcase.MCPSpec
	var argv string
	var env envFlags
	fs.StringVar(&argv, "argv", "", "the reference's program and its leading arguments")
	fs.Var(&env, "env", "KEY=VALUE for the recorded process, {{WORLD}} allowed; repeatable")
	fs.StringVar(&s.PathPrepend, "path-prepend", "", "directory put in front of the recorded process's PATH")
	fs.StringVar(&s.Tool, "tool", "", "the tool to call")
	fs.StringVar(&s.Arguments, "arguments", "", "the call's arguments as a JSON object, {{WORLD}} allowed")
	fs.StringVar(&s.Channel, "channel", "", "the channel the case records")
	fs.StringVar(&s.World, "world", "", "directory to stage")
	fs.StringVar(&s.Out, "out", "", "case directory to write")
	fs.StringVar(&s.Notes, "notes", "", "text for notes.md")
	fs.StringVar(&s.Compare, "compare", "", `"" (text) or "outcome"`)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	tokens, err := cases.SplitCommand(argv)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev record-mcp-case: --argv: %v\n", err)
		return 2
	}
	s.Argv, s.Env = tokens, env
	if len(s.Argv) == 0 || s.Tool == "" || s.World == "" || s.Out == "" {
		fmt.Fprintln(stderr, "loomux dev record-mcp-case: --argv, --tool, --world and --out are required")
		return 2
	}
	if s.Compare != "" && s.Compare != cases.CompareText && s.Compare != cases.CompareOutcome {
		fmt.Fprintf(stderr, "loomux dev record-mcp-case: --compare must be %q or %q, got %q\n",
			cases.CompareText, cases.CompareOutcome, s.Compare)
		return 2
	}
	if err := recordMCPCase(s); err != nil {
		fmt.Fprintf(stderr, "loomux dev record-mcp-case: %v\n", err)
		return 1
	}
	return 0
}

// devFakeOllama answers every request to an Ollama endpoint from one fixture
// until the process is interrupted. The log of requests goes to a file of its
// own, never into a case world: the recorder stages the world in a directory
// the fake does not know.
func devFakeOllama(args []string, _ io.Reader, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev fake-ollama", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fixturePath := fs.String("fixture", "", "JSON file with the one answer every request gets")
	addr := fs.String("addr", "127.0.0.1:11435", "address to listen on")
	logPath := fs.String("log", "", "file the request lines are appended to; stderr without it")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *fixturePath == "" {
		fmt.Fprintln(stderr, "loomux dev fake-ollama: --fixture is required")
		return 2
	}
	fixture, err := fakeollama.Load(*fixturePath)
	log := stderr
	if err == nil && *logPath != "" {
		var file *os.File
		file, err = os.OpenFile(*logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			defer file.Close()
			log = file
		}
	}
	if err == nil {
		ctx, stop := fakeOllamaNotify(context.Background(), os.Interrupt)
		defer stop()
		err = fakeollama.Serve(ctx, *addr, fixture, log)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev fake-ollama: %v\n", err)
		return 1
	}
	return 0
}

func devImportCases(args []string, _ io.Reader, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev import-cases", flag.ContinueOnError)
	fs.SetOutput(stderr)
	mapFile := fs.String("map", "", "TOML file of [[command]] or [[tool]] rules")
	from := fs.String("from", "", "directory of recorded cases")
	to := fs.String("to", "", "directory to write the translated cases to")
	// The two corpora are read by two discoveries -- a command-line case holds
	// a `cmd`, an MCP case a `call` -- so which one this is has to be said, not
	// guessed from what happens to lie in the directory.
	mcp := fs.Bool("mcp", false, "the recordings are MCP calls, not command lines")
	merge := fs.String("merge-fixture", "", "faketool fixture whose answers are appended to every translated world")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *mapFile == "" || *from == "" || *to == "" {
		fmt.Fprintln(stderr, "loomux dev import-cases: --map, --from and --to are required")
		return 2
	}
	var m importcases.Mapping
	if _, err := toml.DecodeFile(*mapFile, &m); err != nil {
		fmt.Fprintf(stderr, "loomux dev import-cases: %v\n", err)
		return 1
	}
	importer := importcases.Import
	if *mcp {
		importer = importcases.ImportMCP
	}
	if err := importer(*from, *to, m); err != nil {
		fmt.Fprintf(stderr, "loomux dev import-cases: %v\n", err)
		return 1
	}
	if *merge == "" {
		return 0
	}
	if err := importcases.MergeFixture(*to, *merge); err != nil {
		fmt.Fprintf(stderr, "loomux dev import-cases: %v\n", err)
		return 1
	}
	return 0
}

func devSwapBinary(args []string, _ io.Reader, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev swap-binary", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", "bin", "directory holding loomux.new.exe")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := swap.Swap(*dir); err != nil {
		fmt.Fprintf(stderr, "loomux dev swap-binary: %v\n", err)
		return 1
	}
	return 0
}

var benchCorpusRun = benchcorpus.BenchmarkCorpus
var benchRepoRun = benchcorpus.BenchmarkRepo
var benchProcessRunner = defaultProcessRunner
var benchCloner = defaultCloner
var benchOpenFS = func(dir string) (fs.FS, error) { return os.DirFS(dir), nil }
var benchLookPath = exec.LookPath
var benchClock = time.Now
var benchReadFile = os.ReadFile
var benchWriteBoth = benchreport.WriteBoth
var benchSaveReport = benchcorpus.SaveReport
var benchStorageOps = benchcorpus.DefaultStorageOps

// defaultProcessRunner kills a component at its deadline so one hanging lane
// cannot stall a corpus run; only the direct child is killed, its own
// children may outlive it until WaitDelay gives up on their pipes.
//
//coverage:exempt runs external process in default process runner
func defaultProcessRunner(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
	cmdName := argv[0]
	if cmdName == "loomux" {
		if self, err := os.Executable(); err == nil {
			cmdName = self
		}
	}
	ctx, cancel := context.Background(), context.CancelFunc(func() {})
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdName, argv[1:]...)
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.WaitDelay = 5 * time.Second
	out, err := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return string(out), -1, true, nil
	}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return string(out), exit.ExitCode(), false, nil
		}
		return string(out), -1, false, err
	}
	return string(out), 0, false, nil
}

//coverage:exempt runs real git clone in default cloner
func defaultCloner(repoURL, targetDir string) (string, error) {
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", "-c", "core.longpaths=true", "--depth=1", repoURL, targetDir)
		cmd.Env = gitenv.Environ()
		if out, err := cmd.CombinedOutput(); err != nil {
			// A clone whose checkout failed leaves a partial tree; kept, the
			// next run would find the directory and benchmark it silently.
			_ = os.RemoveAll(targetDir)
			return "", fmt.Errorf("git clone: %w: %s", err, string(out))
		}
	}
	cmd := exec.Command("git", "-C", targetDir, "rev-parse", "HEAD")
	cmd.Env = gitenv.Environ()
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// devBenchRepos times the hooks on one repository or on the open-source
// corpus and audits which native tools loomux leaves uncovered.
func devBenchRepos(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev bench repos", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var opts benchcorpus.Options
	fs.StringVar(&opts.TargetDir, "dir", ".", "path to target repository")
	fs.StringVar(&opts.CorpusFile, "corpus", "", "path to open-source matrix markdown file")
	fs.IntVar(&opts.Languages, "languages", 5, "number of top languages in corpus mode")
	fs.StringVar(&opts.Tier, "tier", "Sehr viel", "tier category to filter in corpus mode")
	fs.IntVar(&opts.WarmRuns, "warm", 3, "number of warm runs for median calculation")
	fs.StringVar(&opts.CacheDir, "cache-dir", ".cache/benchcorpus", "directory for cloned repositories")
	fs.DurationVar(&opts.Timeout, "timeout", 5*time.Minute, "timeout per repository")
	fs.DurationVar(&opts.ComponentTimeout, "component-timeout", 60*time.Second, "deadline for each measured command")
	out := fs.String("out", "", "directory to write the markdown and JSON report to")
	var save bool
	var reportDir string
	fs.BoolVar(&save, "save", false, "save benchmark reports and update matrix in documentation")
	fs.StringVar(&reportDir, "report-dir", "docs", "documentation root directory for saving benchmarks")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if opts.WarmRuns < 1 {
		fmt.Fprintln(stderr, "loomux dev bench repos: --warm must be at least 1")
		return 2
	}
	if opts.CorpusFile != "" && opts.Languages < 1 {
		fmt.Fprintln(stderr, "loomux dev bench repos: --languages must be at least 1")
		return 2
	}

	stamp := benchreport.Stamp(benchClock())
	md, js, err := benchTargets(*out, "repos", stamp)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev bench repos: %v\n", err)
		return 1
	}

	var report *benchcorpus.BenchmarkReport
	if opts.CorpusFile != "" {
		matrixData, err := benchReadFile(opts.CorpusFile)
		if err != nil {
			fmt.Fprintf(stderr, "loomux dev bench repos: reading corpus file: %v\n", err)
			return 1
		}
		rep, err := benchCorpusRun(matrixData, opts, benchCloner, func(dir string, o benchcorpus.Options) (*benchcorpus.RepoAudit, error) {
			return benchRepoRun(dir, o, benchProcessRunner, benchClock, benchOpenFS, benchLookPath)
		})
		if err != nil {
			fmt.Fprintf(stderr, "loomux dev bench repos: corpus benchmark: %v\n", err)
			return 1
		}
		for _, s := range rep.Skipped {
			fmt.Fprintf(stderr, "loomux dev bench repos: skipped %s: %s\n", s.RepoURL, s.Reason)
		}
		report = rep
	} else {
		audit, err := benchRepoRun(opts.TargetDir, opts, benchProcessRunner, benchClock, benchOpenFS, benchLookPath)
		if err != nil {
			fmt.Fprintf(stderr, "loomux dev bench repos: repository benchmark: %v\n", err)
			return 1
		}
		report = &benchcorpus.BenchmarkReport{
			Timestamp: benchClock().UTC().Format(time.RFC3339),
			Mode:      "single",
			WarmRuns:  opts.WarmRuns,
			Repos:     []*benchcorpus.RepoAudit{audit},
		}
	}

	env := benchreport.Current(Version)
	var text bytes.Buffer
	text.WriteString(benchHead("repos", env, stamp))
	_ = benchcorpus.FormatMarkdown(report, &text)
	if md == "" {
		_, _ = stdout.Write(text.Bytes())
	} else {
		payload, err := benchcorpus.ReportJSON(report, stamp, env)
		if err != nil {
			fmt.Fprintf(stderr, "loomux dev bench repos: encoding the report: %v\n", err)
			return 1
		}
		if err := benchWriteBoth(md, js, text.Bytes(), payload); err != nil {
			fmt.Fprintf(stderr, "loomux dev bench repos: writing the report: %v\n", err)
			return 1
		}
	}

	if save {
		if err := benchSaveReport(report, reportDir, benchStorageOps()); err != nil {
			fmt.Fprintf(stderr, "loomux dev bench repos: saving benchmark reports: %v\n", err)
			return 1
		}
	}

	return 0
}

// benchSearchRun is the seam of the search bench: a real run asks the qmd
// engine, and no test of this command may.
var benchSearchRun = benchsearch.Bench

// benchRepoRoot names the checkout --corpus v1 is read from.
var benchRepoRoot = gitTopLevel

//coverage:exempt runs git in the working directory
func gitTopLevel() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Env = gitenv.Environ()
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// benchQmdVersion is what the engine calls itself, for the report's head.
// Anything but an answer is "unknown", never an abort: the head is a
// document, and a missing line must not throw away a finished measurement.
// It starts qmd through the launcher, since a bare "qmd" on Windows is a
// batch shim that no process start takes.
//
//coverage:exempt starts the qmd process
func benchQmdVersion() string {
	launched, err := search.Launcher("qmd")
	if err != nil {
		return "unknown"
	}
	out, _ := exec.Command(launched[0], append(launched[1:], "--version")...).Output()
	if version := strings.TrimSpace(string(out)); version != "" {
		return version
	}
	return "unknown"
}

// benchSearchDeps are the search bench's reach into the running system.
func benchSearchDeps(stderr io.Writer) benchsearch.Deps {
	return benchsearch.Deps{
		StateDir:    config.StateDir(),
		FallbackDir: config.LegacyBrainDirUntilStage3(),
		Daemon: func() search.SearchPort {
			return search.NewQmdMcpPort(search.WithNotice(prefixedLine(stderr, "note")))
		},
		CLI:        func(name string) search.SearchPort { return &search.QmdPort{Executable: "qmd", Index: name} },
		QmdVersion: benchQmdVersion,
		Models:     index.Models,
		Loomux:     Version,
		Now:        benchClock,
		Clock:      time.Now,
		// Lower case, so the index name reads alike on every file system.
		Random: func() string { return strings.ToLower(rand.Text()) },
		Warn:   prefixedLine(stderr, "warning"),
		Getenv: os.Getenv,
	}
}

// prefixedLine writes each message as one line behind its kind.
func prefixedLine(w io.Writer, kind string) func(string) {
	return func(message string) { fmt.Fprintf(w, "%s: %s\n", kind, message) }
}

// devBenchSearch measures how well the search finds a note: over the
// registered areas through the search service, or over a corpus stand
// through the qmd command line.
func devBenchSearch(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev bench search", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var o benchsearch.Options
	fs.StringVar(&o.Scope, "scope", "knowledge", "the area to measure, or all")
	profile := fs.String("profile", string(search.ProfileFast), "keyword, fast or full")
	channel := fs.String("channel", string(privacy.ChannelLocal), "local or cloud")
	fs.StringVar(&o.Out, "out", "", "directory for the report; default <area>/98 Messung")
	fs.StringVar(&o.Questions, "questions", "", "question set; default <out>/questions.yaml")
	fs.StringVar(&o.Corpus, "corpus", "", "v1 for the checked-in corpus, or a stand's directory")
	fs.BoolVar(&o.Latency, "latency", false, "also time catalog, read and the three profiles")
	fs.StringVar(&o.LatencyQuery, "latency-query", "latenz", "the query the latency searches ask")
	fs.IntVar(&o.Repeat, "repeat", 10, "warm runs per timed operation, after one cold run")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if extra := fs.Args(); len(extra) > 0 {
		fmt.Fprintf(stderr, "loomux dev bench search: unexpected argument %q\n", extra[0])
		return 2
	}
	// Given or not decides whether --corpus is told a scope twice; the
	// value cannot, since the default is a scope too.
	fs.Visit(func(f *flag.Flag) { o.ScopeSet = o.ScopeSet || f.Name == "scope" })
	o.Profile = search.Profile(*profile)
	if o.Profile != search.ProfileKeyword && o.Profile != search.ProfileFast && o.Profile != search.ProfileFull {
		fmt.Fprintf(stderr, "loomux dev bench search: --profile must be keyword, fast or full, got %q\n", *profile)
		return 2
	}
	ch, err := privacy.ParseChannel(*channel)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev bench search: --channel: %v\n", err)
		return 2
	}
	o.Channel = ch
	if o.Repeat < 1 {
		fmt.Fprintf(stderr, "loomux dev bench search: --repeat must be at least 1, got %d\n", o.Repeat)
		return 2
	}
	if o.Corpus == "v1" {
		root, err := benchRepoRoot()
		stand := filepath.Join(root, "testdata", "bench", "search", "v1")
		if err == nil {
			_, err = os.Stat(stand)
		}
		if err != nil {
			fmt.Fprintln(stderr, "error: --corpus v1 needs a loomux checkout; name the stand's directory instead")
			return 1
		}
		o.Corpus = stand
	}
	text, err := benchSearchRun(o, benchSearchDeps(stderr))
	if err != nil {
		// Problems joins its findings by line, and qmd's stderr arrives
		// inside an error: every line is one error line of its own.
		for _, line := range strings.Split(err.Error(), "\n") {
			if line != "" {
				fmt.Fprintf(stderr, "error: %s\n", line)
			}
		}
		return 1
	}
	fmt.Fprint(stdout, text)
	return 0
}
