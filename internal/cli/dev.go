package cli

import (
	"bytes"
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
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/dev/benchcorpus"
	"github.com/xidus90/loomux/internal/dev/benchhooks"
	"github.com/xidus90/loomux/internal/dev/covergate"
	"github.com/xidus90/loomux/internal/dev/importcases"
	"github.com/xidus90/loomux/internal/dev/mutants"
	"github.com/xidus90/loomux/internal/dev/recordcase"
	"github.com/xidus90/loomux/internal/dev/swap"
	"github.com/xidus90/loomux/internal/gitenv"
)

const module = "github.com/xidus90/loomux"

var coverFunc = runCoverFunc

// runCoverFunc keeps the tool's stderr in the error: exec alone would report
// only an exit status, never why the profile was unusable.
func runCoverFunc(profile string) ([]byte, error) {
	out, err := exec.Command("go", "tool", "cover", "-func="+profile).Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out, fmt.Errorf("%w: %s", err, bytes.TrimSpace(exit.Stderr))
	}
	return out, err
}

var benchExec = benchhooks.Exec

var mutantsTest = mutants.GoTest

var mutantsRoot = os.Getwd

var mutantsNotify = signal.NotifyContext

// recordMCPCase is the seam of the MCP recorder. A real recording starts a
// Python reference and a daemon beside it, and no test of this command may do
// either.
var recordMCPCase = recordcase.RecordMCP

var devCommands = map[string]command{
	"bench":           devBench,
	"bench-hooks":     devBenchHooks,
	"covergate":       devCovergate,
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

// devBenchHooks measures the hook commands of a case file. The file may
// stand before or after the flags: Go's flag package stops at the first
// argument that is no flag, so the rest is parsed once more behind it.
func devBenchHooks(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev bench-hooks", flag.ContinueOnError)
	fs.SetOutput(stderr)
	n := fs.Int("n", 20, "warm runs per case, after one cold run")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(stderr, "loomux dev bench-hooks: a case file is required")
		return 2
	}
	if err := fs.Parse(rest[1:]); err != nil {
		return 2
	}
	if extra := fs.Args(); len(extra) > 0 {
		fmt.Fprintf(stderr, "loomux dev bench-hooks: unexpected argument %q after the case file\n", extra[0])
		return 2
	}
	// A run of zero warm runs reports a row of noughts; a negative one
	// would ask for a slice of negative capacity.
	if *n < 1 {
		fmt.Fprintf(stderr, "loomux dev bench-hooks: -n must be at least 1, got %d\n", *n)
		return 2
	}
	var cases []benchhooks.Case
	data, err := os.ReadFile(rest[0])
	if err == nil {
		err = json.Unmarshal(data, &cases)
	}
	if err == nil {
		err = benchhooks.Run(cases, *n, stdout, benchExec, time.Now)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev bench-hooks: %v\n", err)
		return 1
	}
	return 0
}

func devCovergate(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev covergate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	profile := fs.String("profile", "coverage.out", "coverage profile written by go test")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	out, err := coverFunc(*profile)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev covergate: %v\n", err)
		return 1
	}
	lines, err := covergate.Parse(bytes.NewReader(out))
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev covergate: %v\n", err)
		return 1
	}
	// A gate that finds nothing to judge must not pass.
	if len(lines) == 0 {
		fmt.Fprintf(stderr, "loomux dev covergate: no functions in %s\n", *profile)
		return 1
	}
	return covergate.Gate(lines, module, os.ReadFile, stdout)
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
var benchWriteFile = os.WriteFile
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

func devBench(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev bench", flag.ContinueOnError)
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
	fs.StringVar(&opts.OutFile, "out", "", "markdown report output file")
	fs.StringVar(&opts.JSONOutFile, "json-out", "", "JSON report output file")
	var save bool
	var reportDir string
	fs.BoolVar(&save, "save", false, "save benchmark reports and update matrix in documentation")
	fs.StringVar(&reportDir, "report-dir", "docs", "documentation root directory for saving benchmarks")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if opts.WarmRuns < 1 {
		fmt.Fprintln(stderr, "loomux dev bench: --warm must be at least 1")
		return 2
	}
	if opts.CorpusFile != "" && opts.Languages < 1 {
		fmt.Fprintln(stderr, "loomux dev bench: --languages must be at least 1")
		return 2
	}

	var report *benchcorpus.BenchmarkReport
	if opts.CorpusFile != "" {
		matrixData, err := benchReadFile(opts.CorpusFile)
		if err != nil {
			fmt.Fprintf(stderr, "loomux dev bench: reading corpus file: %v\n", err)
			return 1
		}
		rep, err := benchCorpusRun(matrixData, opts, benchCloner, func(dir string, o benchcorpus.Options) (*benchcorpus.RepoAudit, error) {
			return benchRepoRun(dir, o, benchProcessRunner, benchClock, benchOpenFS, benchLookPath)
		})
		if err != nil {
			fmt.Fprintf(stderr, "loomux dev bench: corpus benchmark: %v\n", err)
			return 1
		}
		for _, s := range rep.Skipped {
			fmt.Fprintf(stderr, "loomux dev bench: skipped %s: %s\n", s.RepoURL, s.Reason)
		}
		report = rep
	} else {
		audit, err := benchRepoRun(opts.TargetDir, opts, benchProcessRunner, benchClock, benchOpenFS, benchLookPath)
		if err != nil {
			fmt.Fprintf(stderr, "loomux dev bench: repository benchmark: %v\n", err)
			return 1
		}
		report = &benchcorpus.BenchmarkReport{
			Timestamp: benchClock().UTC().Format(time.RFC3339),
			Mode:      "single",
			WarmRuns:  opts.WarmRuns,
			Repos:     []*benchcorpus.RepoAudit{audit},
		}
	}

	if opts.OutFile != "" {
		var buf bytes.Buffer
		_ = benchcorpus.FormatMarkdown(report, &buf)
		if err := benchWriteFile(opts.OutFile, buf.Bytes(), 0o644); err != nil {
			fmt.Fprintf(stderr, "loomux dev bench: writing markdown output: %v\n", err)
			return 1
		}
	} else {
		_ = benchcorpus.FormatMarkdown(report, stdout)
	}

	if opts.JSONOutFile != "" {
		var buf bytes.Buffer
		_ = benchcorpus.FormatJSON(report, &buf)
		if err := benchWriteFile(opts.JSONOutFile, buf.Bytes(), 0o644); err != nil {
			fmt.Fprintf(stderr, "loomux dev bench: writing JSON output: %v\n", err)
			return 1
		}
	}

	if save {
		if err := benchSaveReport(report, reportDir, benchStorageOps()); err != nil {
			fmt.Fprintf(stderr, "loomux dev bench: saving benchmark reports: %v\n", err)
			return 1
		}
	}

	return 0
}
