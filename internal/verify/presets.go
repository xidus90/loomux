package verify

import (
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/xidus90/loomux/internal/detect"
)

// placeholderCoverProfile is the profile file a measuring run writes and the
// coverage lane reads.
const placeholderCoverProfile = "{coverprofile}"

//go:embed presets.toml
var presetsText string

// loadPresets holds a function, not parsed data: the start rule forbids a
// package variable that parses embedded data, and OnceValues parses on the
// first call only.
var loadPresets = sync.OnceValues(func() (*Presets, error) { return parsePresets(presetsText) })

// Variant replaces the lanes it names when its signal was detected.
type Variant struct {
	When  string
	Lanes map[string]Lane
}

// PresetStack is what loomux runs for one stack unless a project says
// otherwise.
type PresetStack struct {
	TestsWhen []string
	Lanes     map[string]Lane
	Variants  []Variant
}

// Presets are the built-in lanes per stack and which file ending belongs to
// which stack.
type Presets struct {
	Ignored    []string
	Extensions map[string]string
	Stacks     map[string]PresetStack
}

// LoadPresets parses the embedded presets on the first call and hands every
// later caller the same result.
func LoadPresets() (*Presets, error) { return loadPresets() }

// parsePresets reads presets in the schema of [verify], so a preset can say
// nothing a project could not; every failure names the file it came from.
func parsePresets(data string) (*Presets, error) {
	p, err := decodePresets(data)
	if err != nil {
		return nil, fmt.Errorf("presets.toml: %w", err)
	}
	return p, nil
}

func decodePresets(data string) (*Presets, error) {
	doc := map[string]any{}
	if _, err := toml.Decode(data, &doc); err != nil {
		return nil, err
	}
	p := &Presets{Extensions: map[string]string{}, Stacks: map[string]PresetStack{}}
	for _, key := range sortedKeys(doc) {
		var err error
		switch key {
		case "ignored":
			p.Ignored, err = stringList("ignored", doc[key])
		case "extensions":
			err = parseExtensions(p, doc[key])
		case "stack":
			err = parsePresetStacks(p, doc[key])
		default:
			err = fmt.Errorf("unknown key %q", key)
		}
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

// stringList reads a list whose every item is a non-empty string.
func stringList(label string, value any) ([]string, error) {
	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a list of strings", label)
	}
	out := make([]string, 0, len(list))
	for i, item := range list {
		s, ok := item.(string)
		if !ok || s == "" {
			return nil, fmt.Errorf("%s #%d must be a non-empty string", label, i+1)
		}
		out = append(out, s)
	}
	return out, nil
}

func parseExtensions(p *Presets, value any) error {
	table, ok := value.(map[string]any)
	if !ok {
		return errors.New("[extensions] must be a table")
	}
	for _, ext := range sortedKeys(table) {
		if !strings.HasPrefix(ext, ".") {
			return fmt.Errorf("[extensions] key %q must start with a dot", ext)
		}
		stack, ok := table[ext].(string)
		if !ok || !slices.Contains(StackNames(), stack) {
			return fmt.Errorf("[extensions].%q names unknown stack %q", ext, fmt.Sprint(table[ext]))
		}
		p.Extensions[ext] = stack
	}
	return nil
}

func parsePresetStacks(p *Presets, value any) error {
	table, ok := value.(map[string]any)
	if !ok {
		return errors.New("[stack] must be a table")
	}
	for _, name := range sortedKeys(table) {
		if !slices.Contains(StackNames(), name) {
			return fmt.Errorf("[stack.%s] is not a stack; stacks are: %s", name, strings.Join(StackNames(), ", "))
		}
		stack, err := parsePresetStack(name, table[name])
		if err != nil {
			return err
		}
		p.Stacks[name] = stack
	}
	return nil
}

func parsePresetStack(name string, value any) (PresetStack, error) {
	owner := "stack." + name
	table, ok := value.(map[string]any)
	if !ok {
		return PresetStack{}, fmt.Errorf("[%s] must be a table", owner)
	}
	stack := PresetStack{Lanes: map[string]Lane{}}
	var variants any
	for _, key := range sortedKeys(table) {
		v := table[key]
		var err error
		switch {
		case key == "tests_when":
			stack.TestsWhen, err = stringList("["+owner+"].tests_when", v)
		case key == "variant":
			variants = v
		case slices.Contains(Kinds(), key):
			var o Override
			o, err = parseLane(owner, key, v)
			stack.Lanes[key] = o.Lane
		default:
			err = fmt.Errorf("[%s] has unknown key %q", owner, key)
		}
		if err != nil {
			return PresetStack{}, err
		}
	}
	if err := checkGodotPlaceholder(owner, name, func(kind string) Lane { return stack.Lanes[kind] }); err != nil {
		return PresetStack{}, err
	}
	if err := checkLanes(owner, stack.Lanes); err != nil {
		return PresetStack{}, err
	}
	if variants != nil {
		var err error
		if stack.Variants, err = parseVariants(owner, variants, stack.Lanes); err != nil {
			return PresetStack{}, err
		}
	}
	return stack, nil
}

// parseVariants reads the variants of one stack. Each is checked as the
// table it produces, its lanes laid over the stack's, since that is what
// runs when its signal is found.
func parseVariants(owner string, value any, base map[string]Lane) ([]Variant, error) {
	label := "[[" + owner + ".variant]]"
	list, ok := value.([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a list of tables", label)
	}
	out := make([]Variant, 0, len(list))
	for i, table := range list {
		when, ok := table["when"].(string)
		if !ok {
			return nil, fmt.Errorf("%s #%d needs when, the name of a signal", label, i+1)
		}
		if !slices.Contains(detect.SignalNames(), when) {
			return nil, fmt.Errorf("%s #%d names unknown signal %q", label, i+1, when)
		}
		variant := Variant{When: when, Lanes: map[string]Lane{}}
		for _, key := range sortedKeys(table) {
			if key == "when" {
				continue
			}
			if !slices.Contains(Kinds(), key) {
				return nil, fmt.Errorf("%s #%d has unknown key %q", label, i+1, key)
			}
			o, err := parseLane(owner+".variant", key, table[key])
			if err != nil {
				return nil, err
			}
			variant.Lanes[key] = o.Lane
		}
		if err := checkLanes(owner+".variant."+when, layer(base, variant.Lanes)); err != nil {
			return nil, err
		}
		out = append(out, variant)
	}
	return out, nil
}

// checkLanes applies the rules that span lanes, to a preset and again to what
// a project lays over it: no ring of after, and no profile read that nothing
// writes.
func checkLanes(owner string, lanes map[string]Lane) error {
	overrides := map[string]Override{}
	for kind, lane := range lanes {
		overrides[kind] = Override{Lane: lane}
	}
	if err := checkCycle(owner, overrides); err != nil {
		return err
	}
	return checkCoverProfile(owner, lanes)
}

// checkCoverProfile refuses a coverage lane that reads the profile when no
// run of test or coverage writes it: the gate would read a file that is never
// there.
func checkCoverProfile(owner string, lanes map[string]Lane) error {
	coverage := lanes["coverage"]
	if !slices.ContainsFunc(coverage.Commands, namesProfile) {
		return nil
	}
	test := lanes["test"]
	if slices.ContainsFunc(test.Commands, namesProfile) || namesProfile(test.Measuring) || namesProfile(coverage.Measure) {
		return nil
	}
	return fmt.Errorf("[%s].coverage reads %s, but nothing in test or coverage.measure writes it", owner, placeholderCoverProfile)
}

func namesProfile(command string) bool { return strings.Contains(command, placeholderCoverProfile) }

// layer lays a variant over lanes: each kind it names replaces the lane of
// that kind whole, the rest stay.
func layer(lanes, over map[string]Lane) map[string]Lane {
	out := maps.Clone(lanes)
	maps.Copy(out, over)
	return out
}
