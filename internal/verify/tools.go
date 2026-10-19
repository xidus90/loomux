package verify

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// toolArg fills one argument that is a tool placeholder as a whole and
// reports false for one that drops out. {tool:NAME} is the requirement the
// tool runs from: NAME, or NAME with the project's pinned version.
// {from:NAME} is uvx's option to run a tool from such a requirement: gone
// when NAME is not pinned, so an unpinned preset keeps the arguments it
// always had. A version without an operator means exactly that version.
// Any other argument comes back as it is.
func toolArg(arg string, tools map[string]string) (string, bool) {
	inner, ok := strings.CutPrefix(arg, "{")
	if !ok {
		return arg, true
	}
	inner, ok = strings.CutSuffix(inner, "}")
	if !ok {
		return arg, true
	}
	kind, name, ok := strings.Cut(inner, ":")
	if !ok || (kind != "tool" && kind != "from") {
		return arg, true
	}
	spec, pinned := tools[name]
	if kind == "from" {
		if !pinned {
			return "", false
		}
		return "--from=" + requirement(name, spec), true
	}
	if !pinned {
		return name, true
	}
	return requirement(name, spec), true
}

// requirement is name pinned to spec in the form uv reads: ruff==0.6.*, or
// mypy>=1.16,<2 where spec brings its own operator.
func requirement(name, spec string) string {
	if strings.IndexAny(spec, "=<>!~") == 0 {
		return name + spec
	}
	return name + "==" + spec
}

// ShowTools is a lane's command as a person reads it: the tool placeholders
// filled from tools as toolArg fills them, every other word as written. It
// splits at spaces, not as a shell would, since it only shows.
func ShowTools(command string, tools map[string]string) string {
	var out []string
	for _, w := range strings.Fields(command) {
		if w, keep := toolArg(w, tools); keep {
			out = append(out, w)
		}
	}
	return strings.Join(out, " ")
}

// ToolsOf names the tools the presets of stack bring through a placeholder,
// variants included, sorted: the ones [verify.<stack>].tools may pin.
func ToolsOf(stack string) ([]string, error) {
	p, err := LoadPresets()
	if err != nil {
		return nil, err
	}
	s := p.Stacks[stack]
	lanes := []Lane{}
	for _, l := range s.Lanes {
		lanes = append(lanes, l)
	}
	for _, v := range s.Variants {
		for _, l := range v.Lanes {
			lanes = append(lanes, l)
		}
	}
	seen := map[string]bool{}
	for _, l := range lanes {
		for _, c := range slices.Concat(l.Commands, l.OnFile, []string{l.Measure, l.Measuring}) {
			for _, w := range strings.Fields(c) {
				if name, ok := toolName(w); ok {
					seen[name] = true
				}
			}
		}
	}
	return slices.Sorted(maps.Keys(seen)), nil
}

// toolName is the tool a whole-word placeholder names.
func toolName(w string) (string, bool) {
	for _, prefix := range []string{"{tool:", "{from:"} {
		if name, ok := strings.CutPrefix(w, prefix); ok {
			return strings.CutSuffix(name, "}")
		}
	}
	return "", false
}

// parseTools reads [verify.<stack>].tools: a table of tool to version, each
// a tool the stack's presets bring and a version uv can read as one argument.
func parseTools(stack string, value any) (map[string]string, error) {
	owner := "[verify." + stack + "].tools"
	table, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New(owner + " must be a table")
	}
	known, err := ToolsOf(stack)
	if err != nil {
		return nil, err
	}
	if len(known) == 0 {
		return nil, fmt.Errorf("%s: %s has no tools to pin", owner, stack)
	}
	out := map[string]string{}
	for _, name := range sortedKeys(table) {
		if !slices.Contains(known, name) {
			return nil, fmt.Errorf("%s has unknown tool %q; known: %s", owner, name, strings.Join(known, ", "))
		}
		spec, ok := table[name].(string)
		if !ok || !validSpec(spec) {
			return nil, fmt.Errorf("%s.%s must be a version", owner, name)
		}
		out[name] = spec
	}
	return out, nil
}

// validSpec is whether spec is a version or specifier that stays one argument
// and means the same with no shell between: digits, letters and . * = < > ! ~
// , + - only.
func validSpec(spec string) bool {
	return spec != "" && strings.Trim(spec, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ.*=<>!~,+-") == ""
}
