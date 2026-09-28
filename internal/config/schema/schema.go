// Package schema describes every key of a project's .loomux/config.toml:
// where it sits, what it holds, what it is when unset and which module reads
// it. `loomux config` lists, reads and edits through it; the readers stay the
// judges of what a value may be.
package schema

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/xidus90/loomux/internal/config"
)

// Kind is the shape of a value, and so how an editor asks for it.
type Kind int

const (
	String Kind = iota
	Int
	Float
	Bool
	Enum
	StringList
	Table     // shown, not edited
	TableList // [[…]] blocks: shown, appended, never rewritten
)

// Module is the part of loomux that reads a key; switching the module off
// makes its keys inert.
type Module string

const (
	Base  Module = "base"
	Hooks Module = "hooks"
	Brain Module = "brain"
	Graph Module = "graph"
)

// Key is one entry of the configuration.
type Key struct {
	Section string // "commit", "policy.paths.rules", "verify"
	Name    string // "language"; "" for a Table/TableList entry itself
	Kind    Kind
	Default string   // TOML literal of the default; "" when there is none
	Choices []string // Enum only
	Module  Module
	Doc     string // one English sentence
}

// ID is the dotted name a user types; a table entry has no key of its own,
// so its section is its name.
func (k Key) ID() string {
	if k.Name == "" {
		return k.Section
	}
	return k.Section + "." + k.Name
}

// Keys lists every key, sorted by module, section and name. It builds the
// list on each call: the package is linked into every hook, and a caller may
// change what it gets.
func Keys() []Key {
	keys := []Key{
		{Section: "modules", Name: "hooks", Kind: Bool, Default: "true", Module: Base, Doc: "Run the session hooks: post-edit, stop, session-start and subagent. The guard always runs."},
		{Section: "modules", Name: "brain", Kind: Bool, Default: "true", Module: Base, Doc: "Run the brain module: its MCP tools, the wiki lane, convert and fetch."},
		{Section: "modules", Name: "graph", Kind: Bool, Default: "true", Module: Base, Doc: "Offer the code graph's MCP tools."},
		{Section: "commit", Name: "language", Kind: Enum, Choices: []string{"en", "de"}, Default: `"en"`, Module: Base, Doc: "The language commit messages are written in."},
		{Section: "commit", Name: "conventional", Kind: Bool, Default: "true", Module: Base, Doc: "Check the header against Conventional Commits."},
		{Section: "commit", Name: "threshold", Kind: Int, Default: "2", Module: Base, Doc: "How many words of the other language a line may carry."},
		{Section: "commit.allow", Kind: TableList, Module: Base, Doc: "Lines the language check lets through, each with a regex and a reason."},
		{Section: "policy.paths.rules", Kind: TableList, Module: Base, Doc: "Paths no agent may write, each with a match and a reason."},
		{Section: "policy.commands.rules", Kind: TableList, Module: Base, Doc: "Shell commands no agent may run, each with a regex and a reason."},
		{Section: "worktree", Name: "mirror", Kind: StringList, Default: "[]", Module: Base, Doc: "Directories a worktree links to the main checkout instead of owning."},
		{Section: "agent", Name: "default", Kind: String, Module: Base, Doc: "The model every flow role without a binding runs on; a name under agent.models."},
		{Section: "agent", Name: "mcp_servers", Kind: StringList, Default: "[]", Module: Base, Doc: "The MCP servers a flow node with the mcp tool profile may use."},
		{Section: "agent.models.*", Name: "provider", Kind: String, Module: Base, Doc: "Who answers for this model name, e.g. claude or agy."},
		{Section: "agent.models.*", Name: "model", Kind: String, Module: Base, Doc: "The provider's model; unset, the provider CLI's own default."},
		{Section: "agent.roles", Name: "*", Kind: String, Module: Base, Doc: "The model name a flow role runs on."},
		{Section: "flow", Name: "default", Kind: String, Module: Base, Doc: "The flow `loomux flow run` starts without a name."},
		{Section: "flow", Name: "overrides", Kind: StringList, Default: "[]", Module: Base, Doc: "Bundled flows a project flow of the same name may hide or overlay."},
		{Section: "verify", Name: "max_parallel", Kind: Int, Module: Hooks, Doc: "How many lanes run at once; the number of CPUs when unset."},
		// The reader counts whole seconds as an integer; a duration string
		// such as "600s" is refused there.
		{Section: "verify", Name: "timeout", Kind: Int, Default: "600", Module: Hooks, Doc: "How many seconds one command may run."},
		{Section: "verify", Name: "profiles", Kind: Table, Default: `{ edit = ["lint", "types"], precommit = ["lint", "types", "test", "coverage"], stop = ["lint", "types", "test", "coverage"] }`, Module: Hooks, Doc: "Which kinds each profile runs: edit, precommit, stop."},
		{Section: "area", Name: "scope", Kind: String, Module: Brain, Doc: "The scope this project is registered under, e.g. project/loomux."},
		{Section: "layout", Name: "wiki", Kind: String, Module: Brain, Doc: "Where the wiki bundle lives, relative to the root."},
		{Section: "layout", Name: "hub", Kind: String, Module: Brain, Doc: "Where the hub pages live."},
		{Section: "layout", Name: "review", Kind: String, Module: Brain, Doc: "Where review cases are filed."},
		{Section: "layout", Name: "inbox", Kind: String, Module: Brain, Doc: "Where files wait to be converted."},
		{Section: "wiki", Name: "types", Kind: StringList, Module: Brain, Doc: "Page types this area declares beyond the known ones."},
		{Section: "wiki", Name: "untouched_days", Kind: Int, Default: strconv.Itoa(config.DefaultUntouchedDays), Module: Brain, Doc: "After how many days a page counts as untouched."},
		{Section: "index", Name: "include", Kind: StringList, Module: Brain, Doc: "Globs of the files the index reads."},
		{Section: "index", Name: "exclude", Kind: StringList, Module: Brain, Doc: "Globs the index skips."},
		{Section: "index", Name: "unsearched", Kind: StringList, Module: Brain, Doc: "Globs that are registered but never given to qmd."},
		{Section: "privacy", Name: "mode", Kind: Enum, Choices: []string{"automatic_cloud", "local_only", "manual_cloud"}, Default: `"manual_cloud"`, Module: Brain, Doc: "What may leave the machine."},
		{Section: "privacy", Name: "never", Kind: StringList, Default: "[]", Module: Brain, Doc: "Globs that never leave the machine."},
		{Section: "maintenance", Name: "on_merge", Kind: Bool, Module: Brain, Doc: "Record merges for reconciliation."},
		{Section: "maintenance", Name: "branch", Kind: String, Module: Brain, Doc: "The branch whose merges count."},
		{Section: "model", Name: "enabled", Kind: Bool, Module: Brain, Doc: "Let the local model be asked for this area."},
		{Section: "model", Name: "roles", Kind: Table, Module: Brain, Doc: "Which roles the local model takes: describe, place and propose, each true or false."},
	}
	slices.SortStableFunc(keys, func(a, b Key) int {
		return cmp.Or(cmp.Compare(moduleOrder(a.Module), moduleOrder(b.Module)),
			cmp.Compare(a.Section, b.Section), cmp.Compare(a.Name, b.Name))
	})
	return keys
}

// moduleOrder puts the keys every project has first and the optional
// modules after them.
func moduleOrder(m Module) int {
	return slices.Index([]Module{Base, Hooks, Brain, Graph}, m)
}

// Lookup finds a key by its ID, a named key by an ID that fills its name.
func Lookup(id string) (Key, bool) {
	return Match(Keys(), id)
}

// Wildcard stands for a name in a key: agent.roles.* is every role,
// agent.models.*.provider every model's provider.
const Wildcard = "*"

// Named reports whether the key stands for a family of keys, one per name.
func (k Key) Named() bool {
	return strings.Contains(k.Section, Wildcard) || k.Name == Wildcard
}

// withName is the member of a named key's family that name picks.
func (k Key) withName(name string) Key {
	k.Section = strings.Replace(k.Section, Wildcard, name, 1)
	if k.Name == Wildcard {
		k.Name = name
	}
	return k
}

// parent is the path of the table that holds a named key's names: the
// segments of its ID before the wildcard.
func (k Key) parent() []string {
	segments := strings.Split(k.ID(), ".")
	return segments[:slices.Index(segments, Wildcard)]
}

// Match finds the key id names among keys: an exact ID first, then a named key
// whose wildcard id fills with a name. The key comes back as id spells it, so
// an editor writes [agent.roles] reviewer and never a star.
func Match(keys []Key, id string) (Key, bool) {
	for _, k := range keys {
		if !k.Named() && k.ID() == id {
			return k, true
		}
	}
	parts := strings.Split(id, ".")
	for _, k := range keys {
		if !k.Named() {
			continue
		}
		pattern := strings.Split(k.ID(), ".")
		if len(pattern) != len(parts) {
			continue
		}
		if name, ok := fill(pattern, parts); ok {
			return k.withName(name), true
		}
	}
	return Key{}, false
}

// fill is the name that turns pattern into parts, when one does: every other
// segment equal, and the name a valid one.
func fill(pattern, parts []string) (string, bool) {
	name := ""
	for i, segment := range pattern {
		if segment == Wildcard {
			if !config.IsIdentifier(parts[i]) {
				return "", false
			}
			name = parts[i]
			continue
		}
		if segment != parts[i] {
			return "", false
		}
	}
	return name, name != ""
}

// GlobalKeys are the keys of the per-user file `<state>/config.toml`: the
// local model's settings, of which an area may only narrow enabled and roles,
// and the search engine's backbone, which no area has a say in.
func GlobalKeys() []Key {
	return []Key{
		{Section: "model", Name: "enabled", Kind: Bool, Default: "false", Module: Brain, Doc: "Let the local model be asked at all; an area can only switch it off."},
		{Section: "model", Name: "endpoint", Kind: String, Default: strconv.Quote(config.DefaultModelEndpoint), Module: Brain, Doc: "Where Ollama listens; it must stay on the loopback."},
		{Section: "model", Name: "name", Kind: String, Default: strconv.Quote(config.DefaultModelName), Module: Brain, Doc: "The Ollama model that is asked."},
		{Section: "model", Name: "roles", Kind: Table, Default: "{ describe = true, place = true, propose = true }", Module: Brain, Doc: "Which roles the model takes; once set, an unnamed role is off."},
		{Section: "model", Name: "temperature", Kind: Float, Default: "0.0", Module: Brain, Doc: "The sampling temperature, between 0 and 2."},
		{Section: "search", Name: "backbone", Kind: String, Default: strconv.Quote(config.DefaultSearchBackbone), Module: Brain, Doc: "What qmd computes on: cuda, vulkan or cpu; a running qmd daemon keeps its backbone until it is restarted."},
	}
}
