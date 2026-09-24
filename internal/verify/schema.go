package verify

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/shellwords"
)

// placeholderFile is replaced by the edited file, which only an on_file
// command is given.
const placeholderFile = "{file}"

// Kinds are the lanes every stack can have, in the order they run.
func Kinds() []string { return []string{"lint", "types", "test", "coverage", "graph"} }

// StackNames are the stacks [verify] may configure.
func StackNames() []string {
	return []string{"go", "python", "typescript", "vue", "svelte", "css", "html",
		"gdscript", "cpp", "shell", "sql", "rust", "wiki", "project"}
}

// Reserved reports whether name is taken by a built-in check, a kind or the
// profile of everything, so a profile cannot shadow it.
func Reserved(name string) bool {
	return slices.Contains([]string{"gofmt", "commit-msg", "gocover", "graph-fresh", "blast-audit", "all"}, name) || slices.Contains(Kinds(), name)
}

// Lane is how one kind runs for one stack.
type Lane struct {
	Commands  []string
	OnFile    []string
	Threaded  bool
	Measuring string
	Measure   string
	After     string
	Off       bool
	// Needs are files, relative to the lane's directory, without which the
	// lane's commands cannot mean anything, such as a build tree nobody
	// configured. They do not guard on_file, which reads only the edited file.
	Needs []string
}

// Override is one [verify.<stack>].<kind> entry. A string or a list stands
// for the whole lane: whoever writes a command means it as written, which is
// also what the old chain did when a project configured a kind. A table only
// changes the keys it names.
type Override struct {
	Lane    Lane
	Replace bool
	Set     map[string]bool
}

// Config is the [verify] section of .loomux/config.toml.
type Config struct {
	MaxParallel int
	Timeout     time.Duration
	Profiles    map[string][]string
	Stacks      map[string]map[string]Override
	ImportCheck bool
}

func defaults() Config {
	return Config{
		MaxParallel: runtime.NumCPU(),
		Timeout:     600 * time.Second,
		Profiles: map[string][]string{
			"edit":      {"lint", "types"},
			"precommit": {"lint", "types", "test", "coverage", "graph"},
			// What the stop gate runs at every turn end. The four kinds that
			// check code, without graph: its lane reads the index, which is
			// empty at a turn end. A project whose suite is too slow for
			// every turn end narrows it here and keeps a gate that moves.
			"stop": {"lint", "types", "test", "coverage"},
		},
		Stacks:      map[string]map[string]Override{},
		ImportCheck: true,
	}
}

// ReadConfig reads the [verify] section of the project at root. A missing
// file or section is the default; every other failure names the file, since
// the human who fixes it has to find it.
func ReadConfig(root string) (Config, error) {
	path := config.ManifestPath(root)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return defaults(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	doc := map[string]any{}
	if _, err := toml.Decode(string(data), &doc); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return ParseConfig(path, doc)
}

// ParseConfig reads [verify] from a decoded document, and [modules] too,
// because the brain's switch there owns the wiki lane. It refuses every key
// of [verify] it does not know: a misspelt lane that silently kept its preset
// would look configured and not be.
func ParseConfig(path string, doc map[string]any) (Config, error) {
	cfg := defaults()
	if raw, ok := doc["verify"]; ok {
		if err := parseVerify(&cfg, raw); err != nil {
			return Config{}, fmt.Errorf("%s: %w", path, err)
		}
	}
	modules, err := config.ParseModules(path, doc)
	if err != nil {
		return Config{}, err
	}
	if !modules.Brain {
		// The wiki lane is the brain module's lane. Off, it takes the very
		// value `[verify.wiki] lint = false` leaves, so the edit lane, the
		// check chain and the stop gate ask no second question.
		if cfg.Stacks["wiki"] == nil {
			cfg.Stacks["wiki"] = map[string]Override{}
		}
		cfg.Stacks["wiki"]["lint"] = laneOff()
	}
	return cfg, nil
}

// laneOff is a kind switched off with `<kind> = false`.
func laneOff() Override { return Override{Lane: Lane{Off: true}, Replace: true} }

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func isTable(v any) bool {
	_, ok := v.(map[string]any)
	return ok
}

// TopKeys are the keys of [verify] that are not stacks; parseVerify reads
// every other key as a stack and refuses one that names none.
func TopKeys() []string { return []string{"max_parallel", "timeout", "profiles"} }

func parseVerify(cfg *Config, raw any) error {
	table, ok := raw.(map[string]any)
	if !ok {
		return errors.New("[verify] must be a table")
	}
	for _, key := range sortedKeys(table) {
		value := table[key]
		switch {
		case key == "max_parallel":
			n, ok := value.(int64)
			if !ok || n < 1 {
				return fmt.Errorf("[verify].max_parallel must be a positive integer, found %v", value)
			}
			cfg.MaxParallel = int(n)
		case key == "timeout":
			n, ok := value.(int64)
			if !ok || n < 1 {
				return fmt.Errorf("[verify].timeout must be a positive number of seconds, found %v", value)
			}
			cfg.Timeout = time.Duration(n) * time.Second
		case key == "profiles":
			if err := parseProfiles(cfg, value); err != nil {
				return err
			}
		case slices.Contains(Kinds(), key):
			return fmt.Errorf("[verify].%s is the old form; name the stack: [verify.<stack>].%s", key, key)
		case slices.Contains(StackNames(), key):
			if err := parseStack(cfg, key, value); err != nil {
				return err
			}
		case isTable(value):
			return fmt.Errorf("[verify.%s] is not a stack; stacks are: %s", key, strings.Join(StackNames(), ", "))
		default:
			return fmt.Errorf("[verify] has unknown key %q", key)
		}
	}
	return nil
}

func parseProfiles(cfg *Config, value any) error {
	table, ok := value.(map[string]any)
	if !ok {
		return errors.New("[verify.profiles] must be a table")
	}
	for _, name := range sortedKeys(table) {
		if Reserved(name) {
			return fmt.Errorf("[verify.profiles].%s collides with a reserved name", name)
		}
		list, ok := table[name].([]any)
		if !ok {
			return fmt.Errorf("[verify.profiles].%s must be a list of kinds", name)
		}
		if len(list) == 0 {
			return fmt.Errorf("[verify.profiles].%s is empty", name)
		}
		kinds := []string{}
		for _, item := range list {
			kind, ok := item.(string)
			if !ok || !slices.Contains(Kinds(), kind) {
				return fmt.Errorf("[verify.profiles].%s names unknown kind %q", name, fmt.Sprint(item))
			}
			kinds = append(kinds, kind)
		}
		cfg.Profiles[name] = kinds
	}
	return nil
}

func parseStack(cfg *Config, stack string, value any) error {
	table, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("[verify.%s] must be a table", stack)
	}
	lanes := map[string]Override{}
	for _, key := range sortedKeys(table) {
		v := table[key]
		switch {
		case key == "import_check" && stack == "gdscript":
			b, ok := v.(bool)
			if !ok {
				return errors.New("[verify.gdscript].import_check must be a boolean")
			}
			cfg.ImportCheck = b
		case slices.Contains(Kinds(), key):
			// The wiki stack has no tools of its own; its one lane can only
			// be switched off.
			if stack == "wiki" && (key != "lint" || v != false) {
				return errors.New("[verify.wiki].lint can only be false")
			}
			o, err := parseLane("verify."+stack, key, v)
			if err != nil {
				return err
			}
			lanes[key] = o
		default:
			return fmt.Errorf("[verify.%s] has unknown key %q", stack, key)
		}
	}
	if err := checkCycle("verify."+stack, lanes); err != nil {
		return err
	}
	cfg.Stacks[stack] = lanes
	return nil
}

// parseLane reads one kind of the table owner names, e.g. "verify.go"; the
// presets reuse it under their own owner, so no message names [verify] itself.
func parseLane(owner, kind string, value any) (Override, error) {
	label := "[" + owner + "]." + kind
	switch v := value.(type) {
	case bool:
		if v {
			return Override{}, fmt.Errorf("%s = true is not a command; leave the key out to keep the preset", label)
		}
		return laneOff(), nil
	case string:
		cmds, err := commands(label, []any{v}, false)
		return Override{Lane: Lane{Commands: cmds}, Replace: true}, err
	case []any:
		cmds, err := commands(label, v, false)
		return Override{Lane: Lane{Commands: cmds}, Replace: true}, err
	case map[string]any:
		if len(v) == 0 {
			return Override{}, fmt.Errorf("%s is empty", label)
		}
		return parseLaneTable("["+owner+"."+kind+"]", kind, v)
	default:
		return Override{}, fmt.Errorf("%s must be false, a command, a list of commands or a table", label)
	}
}

// parseLaneTable reads the table form, which names only the keys it changes.
func parseLaneTable(table, kind string, v map[string]any) (Override, error) {
	o := Override{Set: map[string]bool{}}
	measured := kind == "test" || kind == "coverage"
	for _, key := range sortedKeys(v) {
		value := v[key]
		var err error
		switch key {
		case "commands", "on_file":
			o.Lane.Commands, o.Lane.OnFile, err = laneList(table, key, value, o.Lane.Commands, o.Lane.OnFile)
		case "threaded":
			b, ok := value.(bool)
			if !ok {
				return Override{}, fmt.Errorf("%s.threaded must be a boolean", table)
			}
			o.Lane.Threaded = b
		case "measuring", "measure", "after":
			if !measured {
				return Override{}, fmt.Errorf("%s cannot have %s", table, key)
			}
			err = laneString(table, key, value, &o.Lane)
		case "needs":
			o.Lane.Needs, err = laneNeeds(table, value)
		default:
			return Override{}, fmt.Errorf("%s has unknown key %q", table, key)
		}
		if err != nil {
			return Override{}, err
		}
		o.Set[key] = true
	}
	return o, nil
}

// laneList reads commands or on_file and returns both lists, the one it read
// replaced.
func laneList(table, key string, value any, cmds, onFile []string) ([]string, []string, error) {
	list, ok := value.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("%s.%s must be a list of commands", table, key)
	}
	got, err := commands(table+" "+key, list, key == "on_file")
	if key == "on_file" {
		return cmds, got, err
	}
	return got, onFile, err
}

// laneNeeds reads needs: files inside the lane's directory, since a lane
// cannot wait for what lies outside the project.
func laneNeeds(table string, value any) ([]string, error) {
	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s.needs must be a list of files", table)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("%s.needs is empty", table)
	}
	out := make([]string, 0, len(list))
	for i, item := range list {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%s.needs #%d must be a string", table, i+1)
		}
		if !filepath.IsLocal(filepath.FromSlash(s)) {
			return nil, fmt.Errorf("%s.needs #%d %q must be a path inside the lane's directory", table, i+1, s)
		}
		out = append(out, s)
	}
	return out, nil
}

// laneString reads measuring, measure or after into lane.
func laneString(table, key string, value any, lane *Lane) error {
	s, ok := value.(string)
	if key == "after" {
		if !ok || !slices.Contains(Kinds(), s) {
			return fmt.Errorf("%s.after names unknown kind %q", table, fmt.Sprint(value))
		}
		lane.After = s
		return nil
	}
	if !ok {
		return fmt.Errorf("%s.%s must be a string", table, key)
	}
	if _, err := commands(table+" "+key, []any{s}, false); err != nil {
		return err
	}
	if key == "measuring" {
		lane.Measuring = s
	} else {
		lane.Measure = s
	}
	return nil
}

// commands checks a list of command lines: each one splits by shell rules,
// and only an on_file command may name the edited file.
func commands(label string, items []any, allowFile bool) ([]string, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("%s is empty", label)
	}
	out := make([]string, 0, len(items))
	for i, item := range items {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%s #%d must be a string", label, i+1)
		}
		if strings.TrimSpace(s) == "" {
			return nil, fmt.Errorf("%s #%d is empty", label, i+1)
		}
		if _, err := shellwords.Split(s); err != nil {
			return nil, fmt.Errorf("%s #%d: %w", label, i+1, err)
		}
		if !allowFile && strings.Contains(s, placeholderFile) {
			return nil, fmt.Errorf("%s uses %s, which only on_file knows", label, placeholderFile)
		}
		out = append(out, s)
	}
	return out, nil
}

// checkCycle follows after from every kind in turn; a path that comes back to
// a kind it already passed is a ring no scheduler could start.
func checkCycle(owner string, lanes map[string]Override) error {
	for _, start := range Kinds() {
		path := []string{start}
		for kind := start; ; {
			next := lanes[kind].Lane.After
			if next == "" {
				break
			}
			path = append(path, next)
			if slices.Contains(path[:len(path)-1], next) {
				return fmt.Errorf("[%s] after forms a cycle: %s", owner, strings.Join(path, " -> "))
			}
			kind = next
		}
	}
	return nil
}
