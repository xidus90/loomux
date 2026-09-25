// Package hostfile holds the hook entries loomux installs per host and adds
// them to a hook file that belongs to the project.
//
// An entry is ours when its command calls a loomux binary (Owned); nothing
// else marks it. An entry of ours already on its event and matcher is kept
// as it is, whatever its command says, and a hook of the project on the
// same slot is reported and left alone. Nothing is rewritten and nothing
// removed: where the merge has nothing to add, the file comes back byte for
// byte.
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
	Notes   []string // an entry of ours kept under another matcher
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
	}
	if root == nil {
		root = map[string]any{}
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
	for _, entry := range wanted {
		rawList, listed := hooks[entry.Event]
		list, ok := rawList.([]any)
		if listed && !ok {
			return Result{}, fmt.Errorf("%s: [%s].%s is not a list", file, key, entry.Event)
		}
		slot := entry.Event + "/" + entry.Matcher
		own, foreign, elsewhere := find(list, entry)
		if foreign {
			result.Foreign = append(result.Foreign, slot)
		}
		if !own && elsewhere != "" {
			result.Notes = append(result.Notes, entry.Event+": kept an own entry under matcher "+elsewhere)
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
// does not. elsewhere is the matcher of an own block under another matcher
// that runs the same hook -- an entry from before the matcher changed, such
// as ulinit's -- or "" when there is none; it counts as own too, so the hook
// never runs twice.
func find(list []any, entry Entry) (own, foreign bool, elsewhere string) {
	event := hookEventOf(entry.Command)
	for _, raw := range list {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		mine := false
		sameHook := false
		for _, command := range commandsOf(item) {
			if Owned(command) {
				mine = true
				sameHook = sameHook || (event != "" && hookEventOf(command) == event)
			}
		}
		switch {
		case matcherOf(item) == entry.Matcher && mine:
			own = true
		case matcherOf(item) == entry.Matcher:
			foreign = true
		case sameHook && elsewhere == "":
			elsewhere = matcherOf(item)
		}
	}
	return own, foreign, elsewhere
}

// commandsOf is every command of a block, in order.
func commandsOf(item map[string]any) []string {
	hooks, _ := item["hooks"].([]any)
	var out []string
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
