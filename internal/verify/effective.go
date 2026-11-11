package verify

import (
	"maps"
	"reflect"
	"slices"

	"github.com/xidus90/loomux/internal/detect"
)

// Resolved is one lane as it runs, and which layer last shaped it.
type Resolved struct {
	Lane    Lane
	Origin  string
	Defined bool
}

// Effective is what a check runs: every lane of every stack after presets,
// variants and config were laid over each other, and which stacks run where.
type Effective struct {
	Stacks map[string]map[string]Resolved
	// InArea holds, per stack, the lanes of each area whose signals chose
	// another variant than the stack's first area (Lanes).
	InArea     map[string]map[string]map[string]Resolved
	Areas      map[string][]string
	Active     []string
	Configured map[string]bool
	TestsWhen  map[string][]string
	Extensions map[string]string
	Ignored    []string
	Config     Config
}

// Resolve lays the first variant whose signal was detected over the preset,
// and the project's config over that. The merged lanes are checked again: a
// preset and a config that are each sound can still form a ring together.
func Resolve(cfg Config, p *Presets, facts detect.Facts) (Effective, error) {
	eff := Effective{
		Stacks:     map[string]map[string]Resolved{},
		InArea:     map[string]map[string]map[string]Resolved{},
		Areas:      map[string][]string{},
		Active:     []string{},
		Configured: map[string]bool{},
		TestsWhen:  map[string][]string{},
		Extensions: maps.Clone(p.Extensions),
		Ignored:    slices.Clone(p.Ignored),
		Config:     cfg,
	}
	for _, stack := range StackNames() {
		// The wiki lanes are the hooks' business; they have no tools to layer.
		if stack == "wiki" || stack == "wiki-sources" {
			continue
		}
		detected := stack != "project" && slices.Contains(facts.Stacks, stack)
		areas := []string{"."}
		if detected && len(facts.Areas[stack]) > 0 {
			areas = slices.Clone(facts.Areas[stack])
		}
		// Each area chooses its variant from its own signals; the first
		// area's lanes are the stack's, and an area that chose otherwise
		// keeps its own beside them.
		for i, area := range areas {
			lanes, err := resolveStack(stack, cfg.Stacks[stack], p.Stacks[stack], signalsIn(facts, area))
			if err != nil {
				return Effective{}, err
			}
			switch {
			case i == 0:
				eff.Stacks[stack] = lanes
			case !reflect.DeepEqual(lanes, eff.Stacks[stack]):
				if eff.InArea[stack] == nil {
					eff.InArea[stack] = map[string]map[string]Resolved{}
				}
				eff.InArea[stack][area] = lanes
			}
		}
		eff.TestsWhen[stack] = slices.Clone(p.Stacks[stack].TestsWhen)
		// Whoever writes a test command has tests; `test = false` writes none,
		// and a table that leaves commands alone keeps the preset's.
		test, ok := cfg.Stacks[stack]["test"]
		eff.Configured[stack] = ok && !test.Lane.Off && (test.Replace || test.Set["commands"])
		if !detected && !configuresCommand(cfg.Stacks[stack]) {
			continue
		}
		eff.Active = append(eff.Active, stack)
		eff.Areas[stack] = areas
	}
	slices.Sort(eff.Active)
	return eff, nil
}

// Lanes are the lanes of stack as they run in area.
func (e Effective) Lanes(stack, area string) map[string]Resolved {
	if lanes, ok := e.InArea[stack][area]; ok {
		return lanes
	}
	return e.Stacks[stack]
}

// signalsIn are the signals that count in area: those matched there, those
// matched in the root, whose files speak for what lies below it, and those
// detection placed in no area.
func signalsIn(facts detect.Facts, area string) []string {
	var out []string
	for _, s := range facts.Stacks {
		at := facts.Areas[s]
		if len(at) == 0 || slices.Contains(at, area) || slices.Contains(at, ".") {
			out = append(out, s)
		}
	}
	return out
}

func resolveStack(stack string, overrides map[string]Override, preset PresetStack, signals []string) (map[string]Resolved, error) {
	lanes := preset.Lanes
	origin := map[string]string{}
	for i, v := range preset.Variants {
		if !slices.Contains(signals, v.When) {
			continue
		}
		lanes = layer(lanes, v.Lanes)
		for kind := range v.Lanes {
			origin[kind] = "preset, variant " + v.When
		}
		// A later variant found too is passed over: each lane it would have
		// shaped names it, so --show tells why its tool does not run.
		for _, other := range preset.Variants[i+1:] {
			if slices.Contains(signals, other.When) {
				for kind := range other.Lanes {
					origin[kind] = "preset, variant " + v.When + " over " + other.When
				}
			}
		}
		break
	}
	merged := map[string]Lane{}
	out := map[string]Resolved{}
	for _, kind := range Kinds() {
		lane, from := lanes[kind], "preset"
		if o, ok := overrides[kind]; ok {
			lane, from = merge(lane, o), "config"
		} else if v, ok := origin[kind]; ok {
			from = v
		}
		merged[kind] = lane
		out[kind] = Resolved{Lane: lane, Origin: from, Defined: defined(lane)}
	}
	if err := checkLanes("verify."+stack, merged); err != nil {
		return nil, err
	}
	return out, nil
}

// merge lays one config entry over a lane: the string form replaces it but
// keeps its place in the order, the table form changes only what it names.
func merge(base Lane, o Override) Lane {
	if o.Replace {
		if o.Lane.After == "" {
			o.Lane.After = base.After
		}
		return o.Lane
	}
	// Commands written over a lane a preset switched off switch it on.
	if o.Set["commands"] {
		base.Commands, base.Off = o.Lane.Commands, false
		// when_found guards the base's commands, not these.
		base.WhenFound = nil
	}
	if o.Set["on_file"] {
		base.OnFile, base.Off = o.Lane.OnFile, false
	}
	if o.Set["threaded"] {
		base.Threaded = o.Lane.Threaded
	}
	if o.Set["measuring"] {
		base.Measuring = o.Lane.Measuring
	}
	if o.Set["measure"] {
		base.Measure = o.Lane.Measure
	}
	if o.Set["after"] {
		base.After = o.Lane.After
	}
	if o.Set["needs"] {
		base.Needs = o.Lane.Needs
	}
	if o.Set["when_found"] {
		base.WhenFound = o.Lane.WhenFound
	}
	if o.Set["lock"] {
		base.Lock = o.Lane.Lock
	}
	if o.Set["skip_when_only"] {
		base.SkipWhenOnly = o.Lane.SkipWhenOnly
	}
	return base
}

// defined reports whether a lane has anything to run.
func defined(l Lane) bool { return !l.Off && (len(l.Commands) > 0 || len(l.OnFile) > 0) }

// configuresCommand reports whether a project gave a stack something to run,
// which makes the stack active even where detection found nothing.
func configuresCommand(overrides map[string]Override) bool {
	for _, kind := range Kinds() {
		if o, ok := overrides[kind]; ok && defined(o.Lane) {
			return true
		}
	}
	return false
}
