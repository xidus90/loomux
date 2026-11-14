package cli

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/edit"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/config/target"
	"github.com/xidus90/loomux/internal/hosts"
)

func configUsage(stderr io.Writer) int {
	fmt.Fprintln(stderr, "usage: loomux config [list [--json] | get <key> | set <key> <value> [--yes | --propose] | unset <key> [--yes | --propose] |")
	fmt.Fprintln(stderr, "                      proposals [--json] | apply <id>|--all [--yes] | reject <id>|--all] [--root DIR | --global]")
	fmt.Fprintln(stderr, "       loomux config llm [on | off] [--root DIR]")
	return 2
}

func configCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	sub := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub, args = args[0], args[1:]
	}
	if sub == "llm" {
		return configLLM(args, stdout, stderr)
	}
	flags := flag.NewFlagSet("loomux config", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "project root; found upwards when empty")
	global := flags.Bool("global", false, "the machine-wide file in the state directory")
	yes := flags.Bool("yes", false, "write without asking")
	asJSON := flags.Bool("json", false, "list as JSON")
	propose := flags.Bool("propose", false, "store the change for a human to apply instead of writing it")
	all := flags.Bool("all", false, "apply or reject every open proposal")
	positional, err := parseInterspersed(flags, args)
	// `config --root DIR list`: the subcommand stood behind the flags.
	if sub == "" && len(positional) > 0 {
		sub, positional = positional[0], positional[1:]
	}
	// The call is judged before the file is looked for: a wrong call is a 2
	// wherever it is typed, also outside any project.
	arity := map[string]int{"": 0, "list": 0, "get": 1, "set": 2, "unset": 1, "proposals": 0, "apply": 1, "reject": 1}
	want, known := arity[sub]
	// A flag the subcommand has no use for is refused rather than ignored:
	// `get --yes` would read as a write that happened.
	takes := map[string][]string{
		"list": {"json"}, "proposals": {"json"},
		"set": {"yes", "propose"}, "unset": {"yes", "propose"},
		"apply": {"yes", "all"}, "reject": {"all"},
	}
	flags.Visit(func(f *flag.Flag) {
		if f.Name != "root" && f.Name != "global" && !slices.Contains(takes[sub], f.Name) {
			known = false
		}
	})
	// apply and reject name one proposal or all of them, never both.
	if *all {
		want--
	}
	writes := sub == "set" || sub == "unset"
	picks := sub == "apply" || sub == "reject"
	// The guard lets an agent's set through because it names --propose; a
	// later --propose=false, or a --propose past -- that the parse took as a
	// value, must not turn that into a write.
	if writes && !*propose && slices.ContainsFunc(args, namesPropose) {
		fmt.Fprintln(stderr, "loomux config: --propose given and switched off; say what you mean")
		return 2
	}
	// --propose with --yes is refused rather than read as either: it would
	// ask to write and not to write at once.
	if err != nil || (*global && *root != "") || !known || len(positional) != want ||
		(*propose && (!writes || *yes)) || (*all && !picks) {
		return configUsage(stderr)
	}
	tgt, err := resolveConfigTarget(*root, *global)
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	switch sub {
	case "":
		return runConfigUI(tgt, stderr)
	case "list":
		return configList(tgt, *asJSON, stdout, stderr)
	case "get":
		return configGet(tgt, positional[0], stdout, stderr)
	case "proposals":
		return configProposals(tgt, *asJSON, stdout, stderr)
	case "apply":
		return configApply(tgt, strings.Join(positional, ""), *all, *yes, stdin, stderr)
	case "reject":
		return configReject(tgt, strings.Join(positional, ""), *all, stderr)
	}
	if *propose {
		p := target.Proposal{Op: sub, Key: positional[0]}
		if sub == "set" {
			p.Input = positional[1]
		}
		return configPropose(tgt, p, stdout, stderr)
	}
	after := writtenHint(tgt, positional[0])
	switch sub {
	case "unset":
		return configWrite(tgt, func(text string) (string, error) {
			return tgt.Unset(text, positional[0])
		}, after, *yes, stdin, stderr)
	default:
		return configWrite(tgt, func(text string) (string, error) {
			return tgt.Change(text, positional[0], positional[1])
		}, after, *yes, stdin, stderr)
	}
}

// backboneHint follows a written search.backbone: the qmd daemon is a process
// of its own that outlives every loomux command, loomux serve included, and it
// reads its backbone once, when it starts. It names the port rather than
// `qmd mcp stop`, which loses the daemon once a `qmd status` -- loomux runs
// one itself -- has removed its PID file.
const backboneHint = "loomux config: a running qmd daemon keeps its backbone until its process ends; " +
	"stop the process listening on port 8765 (Windows: `Get-NetTCPConnection -LocalPort 8765 -State Listen | " +
	"ForEach-Object { Stop-Process -Id $_.OwningProcess }`; POSIX: `lsof -ti tcp:8765 -sTCP:LISTEN | xargs kill`), " +
	"since `qmd mcp stop` may answer \"Not running\" after a `qmd status` although the daemon still runs; " +
	"the next search starts it with the new backbone"

// writtenHint is the line a write of key prints after it, or "".
func writtenHint(t target.Target, key string) string {
	if t.Global && key == "search.backbone" {
		return backboneHint
	}
	return ""
}

// namesPropose says whether an argument is any spelling of the propose flag.
func namesPropose(arg string) bool {
	name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
	return name != arg && (name == "propose" || strings.HasPrefix(name, "propose="))
}

func resolveConfigTarget(root string, global bool) (target.Target, error) {
	if global {
		return target.Global(config.StateDir()), nil
	}
	if root == "" {
		found, err := hosts.FindRoot(".")
		if err != nil {
			return target.Target{}, err
		}
		root = found
	}
	return target.Project(root), nil
}

// shownValue is what a human reads for an entry. A preset row has no value
// of its own; the schema default mirrors what the presets fill in, and an
// empty column would read as "unset".
func shownValue(e schema.Entry) string {
	if e.Origin == schema.Preset {
		return e.Key.Default
	}
	return e.Value
}

func configList(t target.Target, asJSON bool, stdout, stderr io.Writer) int {
	entries, err := readEntries(t)
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	if asJSON {
		rows := []map[string]any{}
		for _, e := range entries {
			rows = append(rows, map[string]any{"key": e.Key.ID(), "module": e.Key.Module, "value": shownValue(e), "origin": e.Origin, "count": e.Count})
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rows)
		return 0
	}
	if !t.Global && llmConfigOn(filepath.Dir(filepath.Dir(t.Path))) {
		fmt.Fprintln(stdout, "llm_config: on — agents may edit this file")
	}
	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	for _, e := range entries {
		value := shorten(shownValue(e))
		if e.Key.Kind == schema.TableList {
			value = entryCount(e.Count)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Key.Module, e.Key.ID(), value, e.Origin)
	}
	_ = w.Flush()
	// The human who reads the list is the one who applies what agents
	// proposed; the JSON form stays the bare rows a script expects.
	if n := t.OpenCount(); n > 0 {
		fmt.Fprintln(stdout, proposalHint(n))
	}
	return 0
}

// listWidth is the widest value `config list` prints; the tabwriter sizes
// the column by its longest cell, and one preset table would push the
// origin of every row far to the right. get prints a value whole.
const listWidth = 60

// shorten cuts value to listWidth runes, the last one an ellipsis.
func shorten(value string) string {
	if rs := []rune(value); len(rs) > listWidth {
		return string(rs[:listWidth-1]) + "…"
	}
	return value
}

func entryCount(n int) string {
	if n == 1 {
		return "1 entry"
	}
	return fmt.Sprintf("%d entries", n)
}

// readEntries is read and entries in one step, the way list and get both
// need them.
func readEntries(t target.Target) ([]schema.Entry, error) {
	text, err := t.Read()
	if err != nil {
		return nil, err
	}
	return t.Entries(text)
}

func configGet(t target.Target, id string, stdout, stderr io.Writer) int {
	key, ok := t.Lookup(id)
	if !ok {
		fmt.Fprintf(stderr, "loomux config: unknown key %q\n", id)
		return 1
	}
	// Only the key asked for: the full list has no row for a name the file
	// does not hold, and that name is unset like any other key.
	t.Keys = []schema.Key{key}
	entries, err := readEntries(t)
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	// Exactly one entry: Match hands back a key with its name filled in,
	// never a family, and CurrentOf gives a key that is no family one entry.
	fmt.Fprintln(stdout, shownValue(entries[0]))
	return 0
}

// configWrite is set and unset after the argument check: propose, show the
// diff, confirm, write.
// after, when not empty, is one more line once the file is written.
func configWrite(t target.Target, propose func(text string) (string, error), after string, yes bool, stdin io.Reader, stderr io.Writer) int {
	text, err := t.Read()
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	next, err := propose(text)
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	diff := edit.Diff(text, next)
	if diff == "" {
		fmt.Fprintln(stderr, "loomux config: already so; nothing written")
		return 0
	}
	fmt.Fprint(stderr, diff)
	if !yes {
		fmt.Fprint(stderr, "write these changes? [y/N] ")
		answer, _ := bufio.NewReader(stdin).ReadString('\n')
		if !confirmed(answer) {
			fmt.Fprintln(stderr, "loomux config: declined; nothing written")
			return 0
		}
	}
	if err := t.Write(text, next); err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	fmt.Fprintf(stderr, "loomux config: wrote %s\n", t.Path)
	if after != "" {
		fmt.Fprintln(stderr, after)
	}
	return 0
}

// confirmed reads the answer to a [y/N] question: y or yes, in any case.
func confirmed(answer string) bool {
	a := strings.ToLower(strings.TrimSpace(answer))
	return a == "y" || a == "yes"
}
