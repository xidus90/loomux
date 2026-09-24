// Package schema describes every key of a project's .loomux/config.toml:
// where it sits, what it holds, what it is when unset and which module reads
// it. `loomux config` lists, reads and edits through it; the readers stay the
// judges of what a value may be.
package schema

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/xidus90/loomux/internal/config"
)

// Kind is the shape of a value, and so how an editor asks for it.
type Kind int

const (
	String Kind = iota
	Int
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
		{Section: "modules", Name: "brain", Kind: Bool, Default: "true", Module: Base, Doc: "Run the brain module: its MCP tools and the wiki lane."},
		{Section: "modules", Name: "graph", Kind: Bool, Default: "true", Module: Base, Doc: "Offer the code graph's MCP tools."},
		{Section: "commit", Name: "language", Kind: Enum, Choices: []string{"en", "de"}, Default: `"en"`, Module: Base, Doc: "The language commit messages are written in."},
		{Section: "commit", Name: "conventional", Kind: Bool, Default: "true", Module: Base, Doc: "Check the header against Conventional Commits."},
		{Section: "commit", Name: "threshold", Kind: Int, Default: "2", Module: Base, Doc: "How many words of the other language a line may carry."},
		{Section: "commit.allow", Kind: TableList, Module: Base, Doc: "Lines the language check lets through, each with a regex and a reason."},
		{Section: "policy.paths.rules", Kind: TableList, Module: Base, Doc: "Paths no agent may write, each with a match and a reason."},
		{Section: "policy.commands.rules", Kind: TableList, Module: Base, Doc: "Shell commands no agent may run, each with a regex and a reason."},
		{Section: "worktree", Name: "mirror", Kind: StringList, Default: "[]", Module: Base, Doc: "Directories a worktree links to the main checkout instead of owning."},
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
		{Section: "model", Name: "enabled", Kind: Bool, Module: Brain, Doc: "Let the local model write proposals for this area."},
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

// Lookup finds a key by its ID.
func Lookup(id string) (Key, bool) {
	for _, k := range Keys() {
		if k.ID() == id {
			return k, true
		}
	}
	return Key{}, false
}

// GlobalKeys are the keys of the per-user file. None exist before [model]
// moves there.
func GlobalKeys() []Key { return nil }
