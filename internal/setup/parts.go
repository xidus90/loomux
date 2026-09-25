package setup

import (
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/hosts"
)

// Part is one thing a module brings into a project, chosen or not.
type Part struct {
	Module schema.Module
	// ID is one of "binary", "config", "gitignore", "agents-md", "mcp-json",
	// "tools", "host-entries", "git-hooks", "verify-skill", "area",
	// "merge-hook", "brain-skills", "graph-build".
	ID      string
	Label   string
	Default bool
}

// Parts lists the parts of every module in the order init asks for them,
// each with the default the project's facts give it.
//
// A checkout of loomux gets on by default only what its checked-in files
// already have, so `init --yes` on a fresh clone leaves `git status` empty;
// a human may still pick the rest.
func Parts(f Facts) []Part {
	git := f.Detect.HasGit
	fresh := !f.Checkout
	return []Part{
		{schema.Base, "binary", "the loomux binary", true},
		{schema.Base, "config", ".loomux/config.toml: modules, commit language, rules for the stack", true},
		{schema.Base, "gitignore", ".gitignore: the state directory", true},
		{schema.Base, "agents-md", "AGENTS.md, when the project has none", fresh},
		{schema.Base, "mcp-json", ".mcp.json: the loomux MCP server", fresh},
		{schema.Base, "tools", "check git, qmd, pdftotext, yt-dlp and ollama", true},
		{schema.Hooks, "host-entries", "the hook entries of each host", true},
		{schema.Hooks, "git-hooks", "the git hooks pre-commit, pre-push and commit-msg", git},
		{schema.Hooks, "verify-skill", "the skill verify-until-green", fresh},
		{schema.Brain, "area", "register the project as an area with area add's default wiki, " + areaWiki(f.Root), fresh && !hasArea(f.Config) && !f.Registered},
		// Off in a checkout: the hook goes where core.hooksPath points, and
		// there that is the tracked .githooks.
		{schema.Brain, "merge-hook", "the post-merge hook that records merges", git && fresh},
		{schema.Brain, "brain-skills", "the five brain skills", fresh},
		{schema.Graph, "graph-build", "build the code graph", fresh && len(f.Detect.Stacks) > 0},
	}
}

// Choice is what a run of init does: for which hosts, which modules and
// parts, and with which values the configuration gets.
type Choice struct {
	Hosts []hosts.Host
	// Modules says which optional modules are on; a module missing from the
	// map is on. It is kept apart from Parts because a module can be on
	// while none of its parts is installed now -- a checkout of loomux runs
	// the graph without building it on every init.
	Modules        map[schema.Module]bool
	Parts          map[string]bool
	CommitLanguage string // "en" unless asked otherwise
	Scope          string // "project/<directory name>" unless asked otherwise
}

// moduleOn says whether c runs module m; the base module always runs.
func (c Choice) moduleOn(m schema.Module) bool {
	on, set := c.Modules[m]
	return m == schema.Base || !set || on
}

// scopeName is the last segment of the default scope: the directory's name
// with its blanks joined by a hyphen, since the interactive form refuses a
// scope with blanks and --yes takes this one without asking. A name that
// leaves nothing -- blanks alone, a volume root -- becomes root, as area add
// refuses an empty segment.
func scopeName(root string) string {
	name := strings.Join(strings.Fields(filepath.Base(root)), "-")
	if name == "" || name == "." || strings.ContainsAny(name, `/\`) {
		return "root"
	}
	return name
}

// DefaultChoice starts from what already holds: modules and the commit
// language from the configuration, hosts and parts from the answers of an
// earlier run, and only then the defaults of Parts. `init --yes` therefore
// never changes a value that is already there.
func DefaultChoice(f Facts, answers Answers) Choice {
	c := Choice{
		Hosts:          f.Hosts,
		Modules:        map[schema.Module]bool{schema.Hooks: true, schema.Brain: true, schema.Graph: true},
		Parts:          map[string]bool{},
		CommitLanguage: "en",
		Scope:          "project/" + scopeName(f.Root),
	}
	// A configuration that does not parse keeps the defaults here; Build
	// refuses it and names the file.
	entries, _ := schema.Current(f.Config)
	for _, e := range entries {
		switch {
		case e.Key.Section == "modules" && e.Origin == schema.Set:
			c.Modules[schema.Module(e.Key.Name)] = e.Input == "true"
		case e.Key.ID() == "commit.language" && e.Origin == schema.Set:
			c.CommitLanguage = e.Input
		}
	}
	if len(answers.Hosts) > 0 {
		var chosen []hosts.Host
		for _, name := range answers.Hosts {
			// A host this build does not know is dropped rather than kept:
			// init would have nothing to write for it.
			if h, err := hosts.ParseHost(name); err == nil {
				chosen = append(chosen, h)
			}
		}
		if len(chosen) > 0 {
			c.Hosts = chosen
		}
	}
	for _, p := range Parts(f) {
		c.Parts[p.ID] = p.Default
		if on, ok := answers.Parts[p.ID]; ok {
			c.Parts[p.ID] = on
		}
	}
	return c
}
