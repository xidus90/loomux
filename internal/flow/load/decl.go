package load

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/runs"
)

// SchemaVersion is the one flow file version this build reads.
const SchemaVersion = 1

// nameRule is what a node, field, parameter or role name may look like. It is
// the rule a condition can name: expr scans identifiers and nothing else. A
// flow's own name is its folder's and follows config.FlowNameRule; Find holds
// it to that.
const nameRule = config.IdentifierRule

// The keys each part of a flow file may hold. A key the reader does not
// know is a finding: `instructon` would otherwise be an instruction nobody
// reads, and a later format would find its key taken by a typo. A node holds
// flow.SharedKeys and its block's own keys, and the block's Check refuses the
// ones it does not know.
var (
	topKeys  = []string{"edge", "flow", "node", "params", "schema_version", "state"}
	flowKeys = []string{"role", "start"}
	edgeKeys = []string{"from", "on_error", "to", "when"}
)

// Findings are the problems one stage found, one line each.
type Findings []string

// Error is why a flow will not load: every finding of the stage that stopped it.
func (f Findings) Error() string { return strings.Join(f, "\n") }

// add records one finding behind the file name, the way the spec shows it.
func (f *Findings) add(file, format string, args ...any) {
	*f = append(*f, file+": "+fmt.Sprintf(format, args...))
}

// draft is what stage 1 hands the stages after it: the graph as far as the
// declarations reach. An edge carries its `when` as text only; the text
// becomes a Condition in stage 4 -- after the names are known.
type draft struct {
	graph *flow.Graph
}

// declarations is stage 1: what the file declares, and whether it may.
//
// The document arrives as a map and not as a struct, because BurntSushi's
// decoder passes a wrongly typed value into a struct field without a word --
// `agent = 3` for a string field was measured doing exactly that -- and
// because a map shows the keys nobody declared.
func declarations(file string, doc map[string]any) (*draft, Findings) {
	graph := &flow.Graph{
		Params: make(map[string]flow.Field),
		State:  make(map[string]flow.Field),
	}
	made := &draft{graph: graph}
	var found Findings
	unknown(file, "", doc, topKeys, &found)
	version(file, doc, &found)
	head(file, doc, graph, &found)
	nodes(file, doc, graph, &found)
	fields(file, doc, graph, &found)
	edges(file, doc, made, &found)
	return made, found
}

func version(file string, doc map[string]any, found *Findings) {
	raw, ok := doc["schema_version"]
	if !ok {
		found.add(file, "schema_version is missing; loomux knows version %d", SchemaVersion)
		return
	}
	number, ok := raw.(int64)
	if !ok || number != SchemaVersion {
		found.add(file, "schema_version %v is unknown; loomux knows version %d", raw, SchemaVersion)
	}
}

func head(file string, doc map[string]any, graph *flow.Graph, found *Findings) {
	raw, present := doc["flow"]
	if !present {
		found.add(file, "[flow] is missing")
		return
	}
	head, ok := raw.(map[string]any)
	if !ok {
		found.add(file, "[flow] is %#v, not a table", raw)
		return
	}
	// Two keys an author may well write are named apart: the name, which the
	// folder gives, and model, which a flow no longer chooses.
	for _, key := range sortedNames(head) {
		switch {
		case key == "name":
			found.add(file, "[flow] name: a flow's name is its folder's name")
		case key == "model":
			found.add(file, "[flow] model: a flow names a role, write role = ...")
		case !slices.Contains(flowKeys, key):
			found.add(file, "[flow]: unknown key %q; known keys: %s", key, strings.Join(flowKeys, ", "))
		}
	}
	graph.Start, _ = text(file, "[flow]", "start", head, found)
	graph.Role, _ = text(file, "[flow]", "role", head, found)
	checkRole(file, "[flow]", graph.Role, found)
	if graph.Start == "" {
		found.add(file, "[flow] has no start")
	}
}

// checkRole holds a role to the rule [agent.roles] holds its names to. A role
// is never read by a condition, so the names expr has taken are free for it.
func checkRole(file, where, role string, found *Findings) {
	if role != "" && !config.IsIdentifier(role) {
		found.add(file, "%s: role %q is not a name; a name is %s", where, role, nameRule)
	}
}

// unknown reports every key of entry that known does not hold, in name order,
// with the keys that are known.
func unknown(file, where string, entry map[string]any, known []string, found *Findings) {
	for _, key := range sortedNames(entry) {
		if !slices.Contains(known, key) {
			found.add(file, "%sunknown key %q; known keys: %s", where, key, strings.Join(known, ", "))
		}
	}
}

func nodes(file string, doc map[string]any, graph *flow.Graph, found *Findings) {
	declared := entries(file, "node", doc, found)
	shared := flow.SharedKeys()
	claimed := make(map[string]bool, len(declared))
	for i, entry := range declared {
		where := fmt.Sprintf("node %d", i+1)
		name, _ := text(file, where, "name", entry, found)
		if name == "" {
			found.add(file, "node %d has no name", i+1)
			continue
		}
		if claimed[name] {
			found.add(file, "node %q is declared twice", name)
			continue
		}
		claimed[name] = true
		kind, _ := text(file, where, "kind", entry, found)
		if kind == "" {
			found.add(file, "node %q has no kind", name)
		}
		checkName(file, "node", name, found)
		finite(file, name, entry, found)
		if _, present := entry["model"]; present {
			found.add(file, "node %q: model: a node names a role, write role = ...", name)
		}
		role, _ := text(file, where, "role", entry, found)
		checkRole(file, fmt.Sprintf("node %q", name), role, found)
		keys := make(map[string]any, len(entry))
		for key, value := range entry {
			if !slices.Contains(shared, key) {
				keys[key] = value
			}
		}
		graph.Nodes = append(graph.Nodes, flow.Node{
			Name: name, Kind: kind, Role: role,
			// Stage 4 reads the cap: max_visits may name a parameter, and which
			// parameters there are is not settled before this stage ends.
			MaxVisits: flow.Cap{Add: 1},
			Keys:      keys,
			Raw:       entry,
		})
	}
}

// finite refuses nan and inf anywhere in a node entry. TOML writes both, the
// journal's canonical JSON cannot, and without this the file would load and
// fail after the first paid model call instead.
func finite(file, node string, entry map[string]any, found *Findings) {
	for _, key := range sortedNames(entry) {
		finiteValue(file, node, key, entry[key], found)
	}
}

func finiteValue(file, node, key string, value any, found *Findings) {
	switch held := value.(type) {
	case float64:
		if math.IsNaN(held) || math.IsInf(held, 0) {
			found.add(file, "node %q key %q is %v; a flow file carries only finite numbers", node, key, held)
		}
	case map[string]any:
		for _, inner := range sortedNames(held) {
			finiteValue(file, node, key+"."+inner, held[inner], found)
		}
	case []any:
		for i, item := range held {
			finiteValue(file, node, fmt.Sprintf("%s[%d]", key, i), item, found)
		}
	case []map[string]any:
		for i, item := range held {
			finiteValue(file, node, fmt.Sprintf("%s[%d]", key, i), item, found)
		}
	}
}

func fields(file string, doc map[string]any, graph *flow.Graph, found *Findings) {
	params := table(file, "params", doc, found)
	state := table(file, "state", doc, found)
	tables := []struct {
		label string
		raw   map[string]any
		into  map[string]flow.Field
	}{
		{"parameter", params, graph.Params},
		{"field", state, graph.State},
	}
	for _, table := range tables {
		for _, name := range sortedNames(table.raw) {
			checkName(file, table.label, name, found)
		}
	}
	// A name in both tables would mean whichever one a condition asked first.
	// A parameter the marker keeps a line for could never be given: its option
	// would be refused when the run's marker is written.
	for _, name := range sortedNames(params) {
		if _, both := state[name]; both {
			found.add(file, "%q is declared under [params] and under [state]", name)
		}
		if slices.Contains(runs.Reserved(), name) {
			found.add(file, "parameter %q is a name the run's marker keeps for itself; name it otherwise", name)
		}
	}
	for _, table := range tables {
		for _, name := range sortedNames(table.raw) {
			table.into[name] = declared(file, table.label, name, table.raw[name], found)
		}
	}
}

func declared(file, label, name string, raw any, found *Findings) flow.Field {
	entry, ok := raw.(map[string]any)
	if !ok {
		found.add(file, "%s %q is not a table; write %s = { type = ..., default = ... }", label, name, name)
		return flow.Field{}
	}
	text, _ := entry["type"].(string)
	kind := flow.Type(text)
	switch kind {
	case flow.String, flow.Int, flow.Bool, flow.StringList:
	default:
		found.add(file, "%s %q has type %q; types are %s, %s, %s, %s",
			label, name, text, flow.Bool, flow.Int, flow.StringList, flow.String)
		return flow.Field{}
	}
	value, ok := entry["default"]
	if !ok {
		found.add(file, "%s %q has no default; every field and parameter needs one", label, name)
		return flow.Field{Type: kind}
	}
	// Coerce is the one type table: the int64 TOML hands over becomes the int a
	// declared int field holds, here and nowhere else.
	coerced, err := flow.Coerce(kind, value)
	if err != nil {
		found.add(file, "%s %q default: %v", label, name, err)
		return flow.Field{Type: kind}
	}
	return flow.Field{Type: kind, Default: coerced}
}

func edges(file string, doc map[string]any, made *draft, found *Findings) {
	declared := entries(file, "edge", doc, found)
	for i, entry := range declared {
		where := fmt.Sprintf("edge %d", i+1)
		unknown(file, where+": ", entry, edgeKeys, found)
		from, _ := text(file, where, "from", entry, found)
		to, _ := text(file, where, "to", entry, found)
		if from == "" {
			found.add(file, "edge %d has no from", i+1)
			continue
		}
		if to == "" {
			found.add(file, "edge %d has no to", i+1)
			continue
		}
		onError := flag(file, where, "on_error", entry, found)
		when, hasWhen := text(file, where, "when", entry, found)
		switch {
		case hasWhen && onError:
			found.add(file, "the error edge %s→%s cannot carry a when; an error edge is unconditional", from, to)
		case hasWhen && when == "":
			found.add(file, "edge %s→%s has an empty when", from, to)
		}
		made.graph.Edges = append(made.graph.Edges, flow.Edge{From: from, To: to, Text: when, OnError: onError})
	}
}

// text, flag, table and entries read one key of the document and report one
// that is there but holds the wrong kind of value.
//
// This is the other half of decoding into a map: the decoder is silent about a
// wrong type, and so was this file until every key went through here. Silence
// costs more than a missing diagnostic at two of these keys -- `on_error` and
// `when` decide where a run goes next, and a value the reader cannot use would
// turn an error edge into an ordinary one, or drop a condition, without a
// word. An absent key is never a finding; whether it may be absent is the
// caller's question.
func text(file, where, key string, entry map[string]any, found *Findings) (string, bool) {
	raw, present := entry[key]
	value, ok := raw.(string)
	if present && !ok {
		found.add(file, "%s: %s is %#v, not text", where, key, raw)
	}
	return value, ok
}

func flag(file, where, key string, entry map[string]any, found *Findings) bool {
	raw, present := entry[key]
	value, ok := raw.(bool)
	if present && !ok {
		found.add(file, "%s: %s is %#v, not true or false", where, key, raw)
	}
	return value
}

func table(file, name string, doc map[string]any, found *Findings) map[string]any {
	raw, present := doc[name]
	value, ok := raw.(map[string]any)
	if present && !ok {
		found.add(file, "[%s] is %#v, not a table", name, raw)
	}
	return value
}

func entries(file, name string, doc map[string]any, found *Findings) []map[string]any {
	raw, present := doc[name]
	value, ok := raw.([]map[string]any)
	if present && !ok {
		found.add(file, "%s is %#v, not [[%s]] entries", name, raw, name)
	}
	return value
}

// reserved names are the ones expr and the graph have already taken: expr
// reads true and false as literals before it looks a name up, and END is the
// pseudo node an edge ends at. Either name would be unreachable.
func reserved(name string) bool {
	return name == "true" || name == "false" || name == flow.End
}

func checkName(file, label, name string, found *Findings) {
	if reserved(name) {
		found.add(file, "%q is reserved and cannot be a %s name", name, label)
		return
	}
	if !config.IsIdentifier(name) {
		found.add(file, "%s %q is not a name; a name is %s", label, name, nameRule)
	}
}

// sortedNames orders what a map holds, so that two runs over the same file
// report the same findings in the same order.
func sortedNames[V any](held map[string]V) []string {
	names := make([]string, 0, len(held))
	for name := range held {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// list is how a message offers what there is.
func list[V any](held map[string]V) string {
	names := sortedNames(held)
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}
