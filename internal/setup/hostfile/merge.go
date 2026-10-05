// Package hostfile holds the hook entries loomux installs per host and adds
// them to a hook file that belongs to the project.
//
// An entry is ours when its command calls a loomux binary (Owned); nothing
// else marks it. An entry of ours already on its event and matcher is kept
// as it is, whatever its command says, and a hook of the project on the
// same slot is reported and left alone. An entry of ours under an older
// matcher is kept and named too; where it and the wanted matcher are flat
// lists of tool names, a block for the tools it lacks is appended beside
// it, so an upgrade guards a tool that joined the matcher since. Nothing is
// rewritten and nothing removed: where the merge has nothing to add, the
// file comes back byte for byte.
//
// Antigravity's .agents/hooks.json is a map of named groups, and the name is
// the identity: the group loomux is ours, every other group is carried over
// token for token, and one that already runs a command of ours is named,
// never repaired. No owner field is written into a hook object; whether agy
// tolerates a key it does not know has not been measured.
//
// The JSON is carried as map[string]any throughout, and that is the reason
// this package may use it at all: the hook file belongs to the project, and
// everything in it that is not one of our own hook entries has to come back
// out the way it went in. A typed struct would model the keys this package
// knows about and drop the rest on the next encode -- somebody's
// permissions, somebody's env, somebody's model setting, gone without a word.
// The untyped map is the round trip.
//
// Where the file is not shaped the way this expects, nothing is repaired.
// A `"hooks": []` is someone's decision or someone's mistake; writing an
// object over it would take their configuration with it, and a hook file
// silently rewritten is worse than an install that stops and says so.
package hostfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/hosts"
)

// Result is what Merge made of a hook file. Every slot is named
// "<Event>/<Matcher>".
type Result struct {
	Merged  []byte   // equal to the input when Added is empty
	Added   []string // an entry of ours appended
	Kept    []string // an entry of ours already there
	Foreign []string // someone else's hook on the same event and matcher, kept
	// Notes names an entry of ours kept under another matcher, with the
	// tools a block was appended for, and, in
	// Antigravity's file, another group that already runs one of our
	// commands.
	Notes []string
}

// Merge adds the wanted entries of host to existing, the content of its hook
// file (nil when there is none yet).
func Merge(host hosts.Host, existing []byte, wanted []Entry) (Result, error) {
	file := Path(host)
	if file == "" {
		return Result{}, fmt.Errorf("host %s has no hook file", host)
	}
	key := container(host)
	root := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			return Result{}, fmt.Errorf("%s is not a JSON object: %w", file, err)
		}
		// null is the one document that parses into a nil map without an
		// error; writing over it would repair a file that is no object.
		if root == nil {
			return Result{}, fmt.Errorf("%s is not a JSON object: its root is null", file)
		}
	}
	raw, present := root[key]
	hooks, ok := raw.(map[string]any)
	if present && !ok {
		return Result{}, fmt.Errorf("%s: [%s] is not an object", file, key)
	}
	if hooks == nil {
		hooks = map[string]any{}
	}

	result := Result{Merged: existing}
	if host == hosts.HostAntigravity {
		result.Notes = twiceRun(root, key, wanted)
	}
	for _, entry := range wanted {
		rawList, listed := hooks[entry.Event]
		list, ok := rawList.([]any)
		if listed && !ok {
			return Result{}, fmt.Errorf("%s: [%s].%s is not a list", file, key, entry.Event)
		}
		slot := entry.Event + "/" + entry.Matcher
		own, foreign, elsewhere, stale := find(list, entry)
		if foreign {
			result.Foreign = append(result.Foreign, slot)
		}
		if stale != "" {
			result.Notes = append(result.Notes, slot+": kept an own entry that runs "+stale+
				" instead of "+entry.Command+"; update it by hand")
		}
		if !own && len(elsewhere) > 0 {
			// A block without a matcher key runs for every tool; the note
			// names it so rather than with an empty name.
			matchers := slices.Clone(elsewhere)
			for i, m := range matchers {
				if m == "" {
					matchers[i] = "(none)"
				}
			}
			kept := entry.Event + ": kept an own entry under matcher " + strings.Join(matchers, ", ")
			if missing, counted := missingTools(entry.Matcher, elsewhere); counted && len(missing) > 0 {
				rest := entry
				rest.Matcher = strings.Join(missing, "|")
				hooks[entry.Event] = append(list, blockFor(rest))
				result.Added = append(result.Added, entry.Event+"/"+rest.Matcher)
				result.Notes = append(result.Notes, kept+"; added one for "+rest.Matcher)
				continue
			}
			result.Notes = append(result.Notes, kept)
			own = true
		}
		if own {
			result.Kept = append(result.Kept, slot)
			continue
		}
		hooks[entry.Event] = append(list, blockFor(entry))
		result.Added = append(result.Added, slot)
	}
	if len(result.Added) == 0 {
		return result, nil
	}
	root[key] = orderHooks(hooks)
	result.Merged = formatRoot(extractTopEntries(existing), root, key)
	return result, nil
}

// twiceRun names every group of an Antigravity hook file, other than ours
// under key, that already runs one of the wanted hooks: a command of ours
// (Owned) running the same `hook <event>`, in whatever form -- another
// program path, quoted or not, another --root -- as find judges Claude's.
// Such a group is reported and never repaired: it is not ours, so the honest
// answer is that the hook now fires twice. The group is searched at any
// depth, because a group somebody else wrote need not have the shape ours
// has.
func twiceRun(root map[string]any, key string, wanted []Entry) []string {
	names := make([]string, 0, len(root))
	for name := range root {
		names = append(names, name)
	}
	sort.Strings(names)
	var notes []string
	for _, name := range names {
		if name == key {
			continue
		}
		runs := map[string]bool{}
		collectCommands(root[name], runs)
		events := map[string]bool{}
		for command := range runs {
			if Owned(command) {
				events[hookEventOf(command)] = true
			}
		}
		for _, entry := range wanted {
			if event := hookEventOf(entry.Command); event != "" && events[event] {
				notes = append(notes, "the group "+name+" already runs loomux hook "+event+"; it now fires twice")
				break
			}
		}
	}
	return notes
}

// collectCommands adds to into every string under a key "command" in v, at
// any depth.
func collectCommands(v any, into map[string]bool) {
	switch v := v.(type) {
	case map[string]any:
		for k, child := range v {
			if command, ok := child.(string); ok && k == "command" {
				into[command] = true
			}
			collectCommands(child, into)
		}
	case []any:
		for _, child := range v {
			collectCommands(child, into)
		}
	}
}

// BinaryOf is the binary the entries of host already call in existing:
// Checkout when an entry of ours runs the checkout's bin/loomux.exe,
// Canonical otherwise, including for a file that does not parse.
func BinaryOf(host hosts.Host, existing []byte) string {
	var root map[string]any
	if json.Unmarshal(existing, &root) != nil {
		return Canonical
	}
	hooks, _ := root[container(host)].(map[string]any)
	for _, rawList := range hooks {
		list, _ := rawList.([]any)
		for _, rawBlock := range list {
			block, _ := rawBlock.(map[string]any)
			for _, command := range commandsOf(block) {
				word := firstWord(command)
				if Owned(command) && strings.HasSuffix(word, "/bin/loomux.exe") &&
					strings.Contains(word, "${CLAUDE_PROJECT_DIR}") {
					return Checkout
				}
			}
		}
	}
	return Canonical
}

// find looks at the blocks of list for entry: own when a block on its
// matcher calls loomux in any of its commands, foreign when a block there
// does not. elsewhere is the matcher of every own block under another
// matcher that runs the same hook, in file order -- an entry from before the
// matcher changed, such as one of ours under the matcher from before
// MultiEdit joined it, and a block Merge
// appended beside it for the tools it lacked -- and empty when there is
// none; Merge counts what they cover, so no tool runs the hook twice. A
// block without a matcher key counts as "". stale is the first
// loomux command of an own block on the matcher when none of them is
// entry's command -- an old binary or an old subcommand, kept but not
// current -- and "" otherwise.
func find(list []any, entry Entry) (own, foreign bool, elsewhere []string, stale string) {
	event := hookEventOf(entry.Command)
	current := false
	for _, raw := range list {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		var mine []string
		sameHook := false
		for _, command := range commandsOf(item) {
			if Owned(command) {
				mine = append(mine, command)
				sameHook = sameHook || (event != "" && hookEventOf(command) == event)
			}
		}
		switch {
		case matcherOf(item) == entry.Matcher && len(mine) > 0:
			own = true
			current = current || slices.Contains(mine, entry.Command)
			if stale == "" {
				stale = mine[0]
			}
		case matcherOf(item) == entry.Matcher:
			foreign = true
		case sameHook:
			elsewhere = append(elsewhere, matcherOf(item))
		}
	}
	if current {
		stale = ""
	}
	return own, foreign, elsewhere, stale
}

// missingTools is the names of want that none of have names, in the order
// of want. counted is false when a matcher is no flat list of names -- a
// regular expression, an empty one: what it covers cannot be counted, so
// nothing is added beside it.
func missingTools(want string, have []string) (missing []string, counted bool) {
	names, counted := toolNames(want)
	if !counted {
		return nil, false
	}
	covered := map[string]bool{}
	for _, matcher := range have {
		got, counted := toolNames(matcher)
		if !counted {
			return nil, false
		}
		for _, name := range got {
			covered[name] = true
		}
	}
	for _, name := range names {
		if !covered[name] {
			missing = append(missing, name)
		}
	}
	return missing, true
}

// toolNames splits a flat matcher at | into its names, each of ASCII
// letters, digits and underscores; ok is false for anything else.
func toolNames(matcher string) (names []string, ok bool) {
	names = strings.Split(matcher, "|")
	for _, name := range names {
		if name == "" || strings.ContainsFunc(name, func(r rune) bool {
			return r != '_' && (r < '0' || r > '9') && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z')
		}) {
			return nil, false
		}
	}
	return names, true
}

// commandsOf is every command of a block, in order: a flat handler's own
// command first, then those of its hooks list. loomux never writes a block
// with both, and whether a host runs one of them or both is not measured, so
// both are read, and a command of ours in either makes the block ours.
func commandsOf(item map[string]any) []string {
	var out []string
	if command, ok := item["command"].(string); ok {
		out = append(out, command)
	}
	hooks, _ := item["hooks"].([]any)
	for _, raw := range hooks {
		hook, _ := raw.(map[string]any)
		if command, ok := hook["command"].(string); ok {
			out = append(out, command)
		}
	}
	return out
}

func matcherOf(item map[string]any) string {
	matcher, _ := item["matcher"].(string)
	return matcher
}

func blockFor(entry Entry) map[string]any {
	command := map[string]any{"type": "command", "command": entry.Command}
	if entry.Timeout > 0 {
		command["timeout"] = entry.Timeout
	}
	if entry.Flat {
		return command
	}
	block := map[string]any{"hooks": []any{command}}
	if entry.Matcher != "" {
		block["matcher"] = entry.Matcher
	}
	return block
}

type topEntry struct {
	key string
	raw json.RawMessage
}

// extractTopEntries reads the top-level keys of data in file order, so the
// keys of the project come back where they were.
func extractTopEntries(data []byte) []topEntry {
	if len(data) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil
	}
	var entries []topEntry
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			break
		}
		if key, ok := t.(string); ok {
			var raw json.RawMessage
			if err := dec.Decode(&raw); err == nil {
				entries = append(entries, topEntry{key: key, raw: raw})
			}
		}
	}
	return entries
}

// formatRoot writes the top-level keys in the order of existingEntries,
// with key taken from root, then any key root has beyond them.
func formatRoot(existingEntries []topEntry, root map[string]any, key string) []byte {
	if len(existingEntries) == 0 && len(root) == 0 {
		return []byte("{}\n")
	}
	seen := map[string]bool{}
	var buf bytes.Buffer
	buf.WriteString("{\n")
	first := true

	writeRaw := func(k string, raw []byte) {
		if !first {
			buf.WriteString(",\n")
		}
		first = false
		keyJSON := EncodeJSON(k, "", "")
		buf.WriteString("  " + string(keyJSON) + ": ")
		buf.Write(raw)
	}
	writeField := func(k string, rawVal json.RawMessage) {
		// rawVal comes from the decoder or from json.Marshal, so it is valid
		// JSON and Indent cannot fail on it.
		var indented bytes.Buffer
		_ = json.Indent(&indented, rawVal, "  ", "  ")
		writeRaw(k, indented.Bytes())
	}
	writeContainer := func() {
		if raw, ok := root[key].(json.RawMessage); ok {
			writeRaw(key, raw)
		}
	}

	for _, entry := range existingEntries {
		seen[entry.key] = true
		if entry.key == key {
			writeContainer()
		} else {
			writeField(entry.key, entry.raw)
		}
	}
	if !seen[key] {
		writeContainer()
	}

	var otherKeys []string
	for k := range root {
		if !seen[k] && k != key {
			otherKeys = append(otherKeys, k)
		}
	}
	sort.Strings(otherKeys)
	for _, k := range otherKeys {
		valJSON := EncodeJSON(root[k], "", "")
		writeField(k, valJSON)
	}

	buf.WriteString("\n}\n")
	return buf.Bytes()
}

var lifecycleOrder = []string{
	"SessionStart",
	"PreToolUse",
	"PostToolUse",
	"SubagentStart",
	"SubagentStop",
	"Stop",
}

// orderHooks renders the events in lifecycle order, then the rest by name.
func orderHooks(hooks map[string]any) json.RawMessage {
	if len(hooks) == 0 {
		return json.RawMessage("{}")
	}
	var keys []string
	seen := map[string]bool{}
	for _, event := range lifecycleOrder {
		if _, ok := hooks[event]; ok {
			keys = append(keys, event)
			seen[event] = true
		}
	}
	var rest []string
	for k := range hooks {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	keys = append(keys, rest...)

	var buf strings.Builder
	buf.WriteString("{\n")
	for i, key := range keys {
		keyJSON := EncodeJSON(key, "", "")
		buf.WriteString("    " + string(keyJSON) + ": ")
		list, ok := hooks[key].([]any)
		if ok {
			buf.WriteString("[\n")
			for j, block := range list {
				buf.WriteString(formatBlock(block, "      "))
				if j < len(list)-1 {
					buf.WriteString(",")
				}
				buf.WriteString("\n")
			}
			buf.WriteString("    ]")
		} else {
			valJSON := EncodeJSON(hooks[key], "    ", "  ")
			buf.WriteString(string(valJSON))
		}
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("  }")
	return json.RawMessage(buf.String())
}

// orderedKeys is first in its order where present, then the remaining keys
// of m by name.
func orderedKeys(m map[string]any, first ...string) []string {
	seen := map[string]bool{}
	var keys []string
	for _, k := range first {
		if _, ok := m[k]; ok {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	var rest []string
	for k := range m {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(keys, rest...)
}

func formatCommand(cmd map[string]any, indent string) string {
	keys := orderedKeys(cmd, "type", "command", "timeout")
	var buf strings.Builder
	buf.WriteString(indent + "{\n")
	for i, k := range keys {
		vJSON := EncodeJSON(cmd[k], "", "")
		kJSON := EncodeJSON(k, "", "")
		buf.WriteString(indent + "  " + string(kJSON) + ": " + string(vJSON))
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString(indent + "}")
	return buf.String()
}

func formatBlock(item any, indent string) string {
	block, ok := item.(map[string]any)
	if !ok {
		b := EncodeJSON(item, indent, "  ")
		return indent + string(b)
	}
	if _, flat := block["command"]; flat {
		return formatCommand(block, indent)
	}
	keys := orderedKeys(block, "matcher", "hooks")
	var buf strings.Builder
	buf.WriteString(indent + "{\n")
	for i, k := range keys {
		kJSON := EncodeJSON(k, "", "")
		buf.WriteString(indent + "  " + string(kJSON) + ": ")
		hooksList, isList := block[k].([]any)
		switch {
		case k == "hooks" && isList:
			buf.WriteString("[\n")
			for j, h := range hooksList {
				if hMap, ok := h.(map[string]any); ok {
					buf.WriteString(formatCommand(hMap, indent+"    "))
				} else {
					hJSON := EncodeJSON(h, indent+"    ", "  ")
					buf.WriteString(indent + "    " + string(hJSON))
				}
				if j < len(hooksList)-1 {
					buf.WriteString(",")
				}
				buf.WriteString("\n")
			}
			buf.WriteString(indent + "  ]")
		case k == "hooks":
			vJSON := EncodeJSON(block[k], indent+"  ", "  ")
			buf.WriteString(string(vJSON))
		default:
			vJSON := EncodeJSON(block[k], "", "")
			buf.WriteString(string(vJSON))
		}
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString(indent + "}")
	return buf.String()
}

// EncodeJSON is json.Marshal, or json.MarshalIndent with prefix and indent,
// that leaves <, > and & as they are. The file belongs to the project, and a
// command such as `a && b` has to come back out the way it went in, not as
// `a \u0026\u0026 b`. v is a value decoded from JSON or a struct of strings,
// which always encodes.
func EncodeJSON(v any, prefix, indent string) []byte {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	e.SetIndent(prefix, indent)
	_ = e.Encode(v)
	return bytes.TrimSuffix(b.Bytes(), []byte("\n"))
}
