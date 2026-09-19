package verify

import (
	"maps"
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
	Stacks     map[string]map[string]Resolved
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
		Areas:      map[string][]string{},
		Active:     []string{},
		Configured: map[string]bool{},
		TestsWhen:  map[string][]string{},
		Extensions: maps.Clone(p.Extensions),
		Ignored:    slices.Clone(p.Ignored),
		Config:     cfg,
	}
	for _, stack := range StackNames() {
		// The wiki lane is the hooks' business; it has no tools to layer.
		if stack == "wiki" {
			continue
		}
		lanes, err := resolveStack(stack, cfg.Stacks[stack], p.Stacks[stack], facts.Stacks)
		if err != nil {
			return Effective{}, err
		}
		eff.Stacks[stack] = lanes
		eff.TestsWhen[stack] = slices.Clone(p.Stacks[stack].TestsWhen)
		// Whoever writes a test command has tests; `test = false` writes none,
		// and a table that leaves commands alone keeps the preset's.
		test, ok := cfg.Stacks[stack]["test"]
		eff.Configured[stack] = ok && !test.Lane.Off && (test.Replace || test.Set["commands"])
		detected := stack != "project" && slices.Contains(facts.Stacks, stack)
		if !detected && !configuresCommand(cfg.Stacks[stack]) {
			continue
		}
		eff.Active = append(eff.Active, stack)
		eff.Areas[stack] = []string{"."}
		if detected && len(facts.Areas[stack]) > 0 {
			eff.Areas[stack] = slices.Clone(facts.Areas[stack])
		}
	}
	slices.Sort(eff.Active)
	return eff, nil
}

func resolveStack(stack string, overrides map[string]Override, preset PresetStack, signals []string) (map[string]Resolved, error) {
	lanes := preset.Lanes
	origin := map[string]string{}
	for _, v := range preset.Variants {
		if slices.Contains(signals, v.When) {
			lanes = layer(lanes, v.Lanes)
			for kind := range v.Lanes {
				origin[kind] = "preset, variant " + v.When
			}
			break
		}
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
	if o.Set["commands"] {
		base.Commands = o.Lane.Commands
	}
	if o.Set["on_file"] {
		base.OnFile = o.Lane.OnFile
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
