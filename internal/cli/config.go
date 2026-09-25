package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/edit"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/lock"
)

// configTarget is the one file a run reads and writes, and the keys it knows.
type configTarget struct {
	path   string
	keys   []schema.Key
	global bool
}

func configUsage(stderr io.Writer) int {
	fmt.Fprintln(stderr, "usage: loomux config [list [--json] | get <key> | set <key> <value> [--yes | --propose] | unset <key> [--yes | --propose] |")
	fmt.Fprintln(stderr, "                      proposals [--json] | apply <id>|--all [--yes] | reject <id>|--all] [--root DIR | --global]")
	return 2
}

func configCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	sub := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub, args = args[0], args[1:]
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
	target, err := resolveConfigTarget(*root, *global)
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	switch sub {
	case "":
		return runConfigUI(target, stderr)
	case "list":
		return configList(target, *asJSON, stdout, stderr)
	case "get":
		return configGet(target, positional[0], stdout, stderr)
	case "proposals":
		return configProposals(target, *asJSON, stdout, stderr)
	case "apply":
		return configApply(target, strings.Join(positional, ""), *all, *yes, stdin, stderr)
	case "reject":
		return configReject(target, strings.Join(positional, ""), *all, stderr)
	}
	if *propose {
		p := proposal{Op: sub, Key: positional[0]}
		if sub == "set" {
			p.Input = positional[1]
		}
		return configPropose(target, p, stdout, stderr)
	}
	switch sub {
	case "unset":
		return configWrite(target, func(text string) (string, error) {
			return proposeUnset(target, text, positional[0])
		}, *yes, stdin, stderr)
	default:
		return configWrite(target, func(text string) (string, error) {
			return proposeChange(target, text, positional[0], positional[1])
		}, *yes, stdin, stderr)
	}
}

// namesPropose says whether an argument is any spelling of the propose flag.
func namesPropose(arg string) bool {
	name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
	return name != arg && (name == "propose" || strings.HasPrefix(name, "propose="))
}

func resolveConfigTarget(root string, global bool) (configTarget, error) {
	if global {
		return configTarget{path: filepath.Join(config.StateDir(), "config.toml"), keys: schema.GlobalKeys(), global: true}, nil
	}
	if root == "" {
		found, err := hosts.FindRoot(".")
		if err != nil {
			return configTarget{}, err
		}
		root = found
	}
	return configTarget{path: config.ManifestPath(root), keys: schema.Keys()}, nil
}

// read answers an empty configuration for a file that is not there yet:
// `set` creates it, and every key then shows its default.
func (t configTarget) read() (string, error) {
	data, err := os.ReadFile(t.path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return string(data), err
}

// entries is schema.Current narrowed to the keys this target knows: the
// global file shares the project file's reader but not its keys.
func (t configTarget) entries(text string) ([]schema.Entry, error) {
	all, err := schema.Current(text)
	if err != nil {
		return nil, err
	}
	var out []schema.Entry
	for _, e := range all {
		if _, ok := t.lookup(e.Key.ID()); ok {
			out = append(out, e)
		}
	}
	return out, nil
}

func (t configTarget) lookup(id string) (schema.Key, bool) {
	for _, k := range t.keys {
		if k.ID() == id {
			return k, true
		}
	}
	return schema.Key{}, false
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

func configList(t configTarget, asJSON bool, stdout, stderr io.Writer) int {
	entries, err := t.readEntries()
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
	if n := t.openProposals(); n > 0 {
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
func (t configTarget) readEntries() ([]schema.Entry, error) {
	text, err := t.read()
	if err != nil {
		return nil, err
	}
	return t.entries(text)
}

func configGet(t configTarget, id string, stdout, stderr io.Writer) int {
	if _, ok := t.lookup(id); !ok {
		fmt.Fprintf(stderr, "loomux config: unknown key %q\n", id)
		return 1
	}
	entries, err := t.readEntries()
	if err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	for _, e := range entries {
		if e.Key.ID() == id {
			fmt.Fprintln(stdout, shownValue(e))
		}
	}
	return 0
}

// editableKey finds a key that a line edit can change: known to this target
// and not a table.
func (t configTarget) editableKey(id string) (schema.Key, error) {
	key, ok := t.lookup(id)
	if !ok {
		return schema.Key{}, fmt.Errorf("unknown key %q", id)
	}
	if key.Kind == schema.Table || key.Kind == schema.TableList {
		return schema.Key{}, fmt.Errorf("%s is a table; edit it by hand", id)
	}
	return key, nil
}

// proposeChange computes the new text for one key; callers show and confirm
// it. It is shared with the interactive form.
func proposeChange(t configTarget, text, id, input string) (string, error) {
	key, err := t.editableKey(id)
	if err != nil {
		return "", err
	}
	// Without [area] the declaration reader stops before the brain's keys,
	// so Validate below would pass any value there unseen.
	if key.Module == schema.Brain && id != "area.scope" {
		hasArea, err := declaresArea(text)
		if err != nil {
			return "", err
		}
		if !hasArea {
			return "", fmt.Errorf("%s belongs to the brain, which reads nothing without [area]; set area.scope first", id)
		}
	}
	literal, err := edit.Render(key.Kind, input)
	if err != nil {
		return "", err
	}
	// Defaults are never written: a file that repeats them would pin them
	// against a later change of the default.
	var next string
	if literal == key.Default {
		next, err = edit.Remove(text, key.Section, key.Name)
	} else {
		next, err = edit.Set(text, key.Section, key.Name, literal)
	}
	if err != nil {
		return "", err
	}
	return t.validated(next)
}

// proposeUnset computes the text without the key's line. It needs no [area]
// for a brain key: a line the brain cannot read is the one worth removing.
func proposeUnset(t configTarget, text, id string) (string, error) {
	key, err := t.editableKey(id)
	if err != nil {
		return "", err
	}
	next, err := edit.Remove(text, key.Section, key.Name)
	if err != nil {
		return "", err
	}
	return t.validated(next)
}

// validated hands a new project text to the readers that run in operation;
// the global file has no keys yet and so no reader.
func (t configTarget) validated(next string) (string, error) {
	if !t.global {
		if err := schema.Validate(next); err != nil {
			return "", err
		}
	}
	return next, nil
}

func declaresArea(text string) (bool, error) {
	doc := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &doc); err != nil {
		return false, fmt.Errorf(".loomux/config.toml: not valid TOML: %w", err)
	}
	_, ok := doc["area"].(map[string]any)
	return ok, nil
}

// configWrite is set and unset after the argument check: propose, show the
// diff, confirm, write.
func configWrite(t configTarget, propose func(text string) (string, error), yes bool, stdin io.Reader, stderr io.Writer) int {
	text, err := t.read()
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
	if err := writeConfig(t, text, next); err != nil {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	fmt.Fprintf(stderr, "loomux config: wrote %s\n", t.path)
	return 0
}

// confirmed reads the answer to a [y/N] question: y or yes, in any case.
func confirmed(answer string) bool {
	a := strings.ToLower(strings.TrimSpace(answer))
	return a == "y" || a == "yes"
}

// writeConfig is the one way every form puts a new text in place: set,
// unset, the interactive form and apply. before is the text next was made
// from; a file that no longer holds it was changed by someone else while
// the human read the diff, and next would take that change back unseen.
func writeConfig(t configTarget, before, next string) error {
	current, err := t.read()
	if err != nil {
		return err
	}
	if current != before {
		return fmt.Errorf("%s changed since it was read; nothing written", t.path)
	}
	err = os.MkdirAll(filepath.Dir(t.path), 0o755)
	if err == nil {
		err = lock.ReplaceText(t.path, next)
	}
	return err
}
