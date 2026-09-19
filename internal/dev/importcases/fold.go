package importcases

import (
	"errors"
	"maps"
	"os"
	"slices"
	"sort"

	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/verify"
)

// foldVerify translates the old [verify] of the world in dir so that loomux
// runs what the old chain ran. A kind the old configuration named replaced
// the preset of the one stack the old chain knew; loomux runs every stack it
// detects, so the commands move to the project and every detected stack gets
// that kind switched off.
//
// Only the shape changes here. A value the old chain refused is carried as it
// stands, so that loomux refuses it as well; a key the old chain never read,
// and the dropped `tests` and `threshold` (deviation 12), are left behind.
func foldVerify(dir string, old map[string]any) (map[string]any, error) {
	folded := map[string]any{}
	project := map[string]any{}
	stacks := map[string]map[string]any{}
	stack := func(name string) map[string]any {
		if stacks[name] == nil {
			stacks[name] = map[string]any{}
		}
		return stacks[name]
	}
	var replaced []string
	afters := map[string]any{}
	for _, key := range sortedKeys(old) {
		value := old[key]
		switch key {
		case "lint", "types", "test", "coverage":
			if lane, ok := foldLane(key, value); ok {
				project[key] = lane
				replaced = append(replaced, key)
			}
		case "after":
			table, ok := value.(map[string]any)
			if !ok {
				return nil, errors.New("[verify.after] must be a table")
			}
			afters = table
		case "profiles", "max_parallel", "timeout":
			folded[key] = value
		case "godot_import":
			stack("gdscript")["import_check"] = value
		}
	}
	for _, kind := range sortedKeys(afters) {
		project[kind] = withAfter(project[kind], afters[kind])
	}
	for _, name := range detect.Detect(os.DirFS(dir)).Stacks {
		// The wiki lane is no tool the old chain ran, and a signal such as
		// cmake is no stack at all.
		if name == "wiki" || !slices.Contains(verify.StackNames(), name) {
			continue
		}
		for _, kind := range replaced {
			stack(name)[kind] = false
		}
	}
	if len(project) > 0 {
		folded["project"] = project
	}
	for name, table := range stacks {
		folded[name] = table
	}
	return folded, nil
}

// foldLane reads one kind of the old [verify]. [verify.coverage] named its
// command `report` beside a `threshold` nothing enforced; without a report it
// left the preset in place.
func foldLane(kind string, value any) (any, bool) {
	table, ok := value.(map[string]any)
	if !ok || kind != "coverage" {
		return value, true
	}
	report, ok := table["report"]
	return report, ok
}

// withAfter orders a project lane after another kind; after exists only in
// the table form, so a command or a list becomes that table.
func withAfter(lane, after any) any {
	switch v := lane.(type) {
	case nil:
		return map[string]any{"after": after}
	case string:
		return map[string]any{"commands": []any{v}, "after": after}
	case []any:
		return map[string]any{"commands": v, "after": after}
	case map[string]any:
		table := maps.Clone(v)
		table["after"] = after
		return table
	}
	return lane
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
