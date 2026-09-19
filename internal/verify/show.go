package verify

import (
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// WriteShow prints what a check would run, as the [verify] table a project
// could write: the run's limits, then each active stack with its areas and
// test state, and each requested kind with the layer that shaped every key.
func WriteShow(w io.Writer, eff Effective, kinds []string, env PlanEnv) {
	fmt.Fprintf(w, "# max_parallel = %d, timeout = %ss\n",
		eff.Config.MaxParallel, strconv.FormatFloat(eff.Config.Timeout.Seconds(), 'f', -1, 64))
	for _, stack := range slices.Sorted(slices.Values(eff.Active)) {
		areas := slices.Sorted(slices.Values(eff.Areas[stack]))
		fmt.Fprintf(w, "\n# %s: areas %s, %s\n", stack, strings.Join(areas, ", "), testState(eff.TestsWhen[stack], areas, env))
		writeStack(w, stack, kinds, eff.Stacks[stack])
	}
}

// writeStack prints the requested lanes of one stack so that they load back:
// the lanes switched off as `<kind> = false` in the stack's own table first,
// then a table for each other lane.
func writeStack(w io.Writer, stack string, kinds []string, lanes map[string]Resolved) {
	blocks := 0
	for _, kind := range kinds {
		if r := lanes[kind]; r.Lane.Off {
			if blocks == 0 {
				fmt.Fprintf(w, "[verify.%s]\n", stack)
			}
			fmt.Fprintf(w, "%s = false  # %s\n", kind, r.Origin)
			blocks = 1
		}
	}
	for _, kind := range kinds {
		if lanes[kind].Lane.Off {
			continue
		}
		if blocks > 0 {
			fmt.Fprintln(w)
		}
		writeLane(w, stack, kind, lanes[kind])
		blocks++
	}
}

// testState says whether any area of a stack holds tests, or that the stack
// has no signal to look for.
func testState(patterns, areas []string, env PlanEnv) string {
	if len(patterns) == 0 {
		return "tests not detected"
	}
	for _, area := range areas {
		if env.HasTests(filepath.Join(env.Root, area), patterns) {
			return "tests found"
		}
	}
	return "no tests found"
}

// writeLane prints one lane as a TOML table, every key it sets in schema
// order and marked with its origin. A lane with nothing to run is a comment:
// as a table it would be empty, which does not load.
func writeLane(w io.Writer, stack, kind string, r Resolved) {
	head := "[verify." + stack + "." + kind + "]"
	if !r.Defined {
		fmt.Fprintf(w, "# %s not defined\n", head)
		return
	}
	fmt.Fprintln(w, head)
	key := func(name, value string) { fmt.Fprintf(w, "%s = %s  # %s\n", name, value, r.Origin) }
	l := r.Lane
	if len(l.Commands) > 0 {
		key("commands", quoteList(l.Commands))
	}
	if len(l.OnFile) > 0 {
		key("on_file", quoteList(l.OnFile))
	}
	if l.Threaded {
		key("threaded", "true")
	}
	for _, s := range []struct{ name, value string }{{"measuring", l.Measuring}, {"measure", l.Measure}, {"after", l.After}} {
		if s.value != "" {
			key(s.name, strconv.Quote(s.value))
		}
	}
	if len(l.Needs) > 0 {
		key("needs", quoteList(l.Needs))
	}
}

func quoteList(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = strconv.Quote(s)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
