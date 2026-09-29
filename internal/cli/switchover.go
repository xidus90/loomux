package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/xidus90/loomux/internal/switchover"
)

// switchoverWriteFile is the seam of the one write of `prune-hooks`: a test
// makes it fail, or proves it is not called.
var switchoverWriteFile = os.WriteFile

var switchoverCommands = map[string]command{
	"render":      devSwitchoverRender,
	"prune-hooks": devSwitchoverPruneHooks,
}

const switchoverUsage = `usage: loomux dev switchover <render|prune-hooks> [flags]
  render       write a project's apply.sh from a JSON file of its parameters
  prune-hooks  remove the hook groups of a settings.json that loomux replaces
`

func devSwitchoverGroup(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, switchoverUsage)
		return 2
	}
	sub, ok := switchoverCommands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "loomux dev switchover: unknown subcommand %q\n%s", args[0], switchoverUsage)
		return 2
	}
	return sub(args[1:], stdin, stdout, stderr)
}

// switchoverReadParams reads a parameter file. A key Params does not know is
// an error: a misspelt old_files would leave the old files in place, silently.
func switchoverReadParams(path string) (switchover.Params, error) {
	var p switchover.Params
	data, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return p, fmt.Errorf("%s: %w", path, err)
	}
	if dec.Decode(new(json.RawMessage)) != io.EOF {
		return p, fmt.Errorf("%s: data after the JSON document", path)
	}
	return p, nil
}

// devSwitchoverRender writes the apply.sh of one project. It never replaces a
// script that is there: the human may have read that one already.
func devSwitchoverRender(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev switchover render", flag.ContinueOnError)
	fs.SetOutput(stderr)
	params := fs.String("params", "", "JSON file of the parameters")
	out := fs.String("out", "", "the script to write; it must not exist")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "loomux dev switchover render: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	if *params == "" || *out == "" {
		fmt.Fprintln(stderr, "loomux dev switchover render: --params and --out are required")
		return 2
	}
	p, err := switchoverReadParams(*params)
	script := ""
	if err == nil {
		script, err = switchover.Render(p)
	}
	if err == nil {
		err = writeNewFile(*out, []byte(script))
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev switchover render: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, *out)
	return 0
}

// matchFlags collects a --match that may be given more than once.
type matchFlags []string

func (m *matchFlags) String() string { return fmt.Sprint(*m) }

func (m *matchFlags) Set(value string) error {
	*m = append(*m, value)
	return nil
}

// devSwitchoverPruneHooks removes the replaced hook groups from a settings
// file. It writes only when a group goes, so a second run leaves the file as
// it is, down to its modification time.
func devSwitchoverPruneHooks(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev switchover prune-hooks", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", "", "the settings.json to prune")
	var needles matchFlags
	fs.Var(&needles, "match", "a group goes when every command of it contains one of these; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "loomux dev switchover prune-hooks: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	if *file == "" || len(needles) == 0 {
		fmt.Fprintln(stderr, "loomux dev switchover prune-hooks: --file and at least one --match are required")
		return 2
	}
	settings, err := os.ReadFile(*file)
	var out []byte
	var removed, kept []string
	if err == nil {
		out, removed, kept, err = switchover.PruneHooks(settings, needles)
	}
	if err == nil && len(removed) > 0 {
		err = switchoverWriteFile(*file, out, 0o644)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev switchover prune-hooks: %v\n", err)
		return 1
	}
	if len(removed) == 0 {
		fmt.Fprintln(stdout, "removed nothing")
	}
	for _, r := range removed {
		fmt.Fprintf(stdout, "removed: %s\n", r)
	}
	for _, k := range kept {
		fmt.Fprintf(stdout, "kept: %s\n", k)
	}
	return 0
}
