package load

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"github.com/xidus90/loomux/internal/flow"
)

// Params are a run's parameters: every declared default, with the options a run
// was started with laid over them.
//
// An option is text wherever it comes from -- the command line, or a run's
// marker on resume -- so it is read here into the type its parameter declares.
// An int is a decimal number, a bool is true or false and nothing else, a
// string is the text itself, and a list is a JSON list of strings as given on
// the command line. The marker keeps that text unchanged, so a resume reads the
// same text again. Every option that is no parameter or does not read as its
// type is a finding, and all of them come back at once.
func Params(graph *flow.Graph, options map[string]string) (flow.Params, error) {
	params := make(flow.Params, len(graph.Params))
	for name, field := range graph.Params {
		params[name] = field.Default
	}
	names := make([]string, 0, len(options))
	for name := range options {
		names = append(names, name)
	}
	slices.Sort(names)
	var found Findings
	for _, name := range names {
		field, declared := graph.Params[name]
		if !declared {
			found = append(found, fmt.Sprintf("option %s is no parameter of flow %q; known parameters: %s",
				name, graph.Name, list(graph.Params)))
			continue
		}
		value, ok := readOption(field.Type, options[name])
		if !ok {
			found = append(found, fmt.Sprintf("option %s: cannot read %q as %s", name, options[name], field.Type))
			continue
		}
		params[name] = value
	}
	if len(found) > 0 {
		return nil, found
	}
	return params, nil
}

// readOption reads one option's text as the type its parameter declares.
func readOption(t flow.Type, text string) (flow.Value, bool) {
	switch t {
	case flow.Int:
		number, err := strconv.Atoi(text)
		return number, err == nil
	case flow.Bool:
		return text == "true", text == "true" || text == "false"
	case flow.StringList:
		var items []string
		err := json.Unmarshal([]byte(text), &items)
		return items, err == nil && items != nil
	default:
		return text, true
	}
}
