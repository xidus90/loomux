package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/dev/benchhooks"
	"github.com/xidus90/loomux/internal/dev/covergate"
	"github.com/xidus90/loomux/internal/dev/importcases"
	"github.com/xidus90/loomux/internal/dev/recordcase"
	"github.com/xidus90/loomux/internal/dev/swap"
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

var devCommands = map[string]command{
	"bench-hooks":  devBenchHooks,
	"covergate":    devCovergate,
	"import-cases": devImportCases,
	"record-case":  devRecordCase,
	"swap-binary":  devSwapBinary,
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

func devImportCases(args []string, _ io.Reader, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev import-cases", flag.ContinueOnError)
	fs.SetOutput(stderr)
	mapFile := fs.String("map", "", "TOML file of [[command]] rules")
	from := fs.String("from", "", "directory of recorded cases")
	to := fs.String("to", "", "directory to write the translated cases to")
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
	if err := importcases.Import(*from, *to, m); err != nil {
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
