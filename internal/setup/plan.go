package setup

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/setup/gitfiles"
	"github.com/xidus90/loomux/internal/setup/hostfile"
	"github.com/xidus90/loomux/internal/setup/templates"
)

// Change is one file init writes, with the text before and after.
type Change struct {
	Part          string
	Path          string // slash-separated, relative to Root
	Before, After string
	Exists        bool
	// Binary is the binary the written file calls, hostfile.Canonical or
	// hostfile.Checkout, and "" for a file that calls none. Without it the
	// file would call nothing, so Apply drops the change.
	Binary string
	// Redo makes After again from the text that stands when Apply writes,
	// if that is no longer Before: area-add runs first and may have written
	// the file since the plan was made. nil for a change nothing else in a
	// run writes.
	Redo func(current string) (string, error)
}

// Empty says whether the change leaves the file as it is.
func (c Change) Empty() bool { return c.Before == c.After }

// Action is one step init runs rather than writes; ID is one of
// "binary-install", "binary-build", "hooks-path", "area-add", "merge-hook",
// "graph-build".
type Action struct{ Part, ID, Describe string }

// Plan is everything a run of init would do, for a human to read first.
type Plan struct {
	Changes []Change // Empty() ones are dropped
	Actions []Action
	Notes   []string // skipped files, foreign entries, missing tools, the user-scope MCP server
}

// checkoutGitBinary is what a git hook of a checkout calls. Git sets no
// CLAUDE_PROJECT_DIR, so the host entries' form would call nothing; git
// runs a hook in the top of the working tree, and the leading ./ keeps exec
// from searching the PATH.
const checkoutGitBinary = "./bin/loomux.exe"

// builder collects a plan and the first error; every later step runs on,
// but the error decides what Build returns.
type builder struct {
	f    Facts
	c    Choice
	read func(rel string) ([]byte, bool, error)
	plan Plan
	err  error
}

// Build plans what c does to the project f describes; read gives the
// current text of a file relative to the root and whether it exists.
// A file init has to merge and cannot -- a configuration the loaders
// refuse, a hook file or .mcp.json that is not the object it should be --
// stops the plan with an error naming the file.
func Build(f Facts, c Choice, read func(rel string) ([]byte, bool, error)) (Plan, error) {
	b := &builder{f: f, c: c, read: read}
	modules := map[string]schema.Module{}
	for _, p := range Parts(f) {
		modules[p.ID] = p.Module
	}
	on := func(id string) bool { return c.Parts[id] && c.moduleOn(modules[id]) }
	var targets []hosts.Host
	for _, h := range c.Hosts {
		if hostfile.Path(h) == "" {
			b.note(fmt.Sprintf("%s: init writes nothing for this host yet", h))
			continue
		}
		targets = append(targets, h)
	}
	if slices.Contains(targets, hosts.HostAntigravity) {
		b.note("antigravity: .agents/hooks.json loads only in a folder agy trusts (trustedWorkspaces)")
	}

	// An action is planned only while its result is missing, so a second
	// run over a finished project plans none.
	switch {
	case !on("binary") || f.BinaryThere:
	case f.Checkout || f.Binary == hostfile.Checkout:
		b.action("binary", "binary-build", "build bin/loomux.exe from this checkout")
	default:
		b.action("binary", "binary-install", "install loomux to ${LOCALAPPDATA}/loomux/bin/loomux.exe")
	}
	if on("config") {
		before, exists := b.file(configPath)
		redo := func(current string) (string, error) { return configText(current, f.Detect.Stacks, c) }
		after, err := redo(string(before))
		b.fail(err)
		b.redoable("config", configPath, before, exists, after, redo)
	}
	if on("gitignore") {
		before, exists := b.file(".gitignore")
		after, _ := gitfiles.WithGitignore(string(before))
		b.change("gitignore", ".gitignore", before, exists, after)
	}
	if on("agents-md") {
		b.agentsMD()
	}
	if on("mcp-json") {
		b.mcpJSON()
	}
	if on("tools") {
		for _, n := range missingTools() {
			b.note(n)
		}
	}
	// Antigravity's entries and the merge hook call the installed binary in
	// every project, a checkout included.
	installed := f.CanonicalThere || slices.ContainsFunc(b.plan.Actions, func(a Action) bool { return a.ID == "binary-install" })
	if on("host-entries") {
		for _, h := range targets {
			// Antigravity's hook file is neither read nor refused while
			// nothing would be written into it.
			switch {
			case h == hosts.HostAntigravity && f.LocalAppDataSpaced:
				b.note("antigravity: no entries; %LOCALAPPDATA% contains a space, and cmd.exe would split the unquoted path")
			case h == hosts.HostAntigravity && !installed:
				b.note("antigravity: no entries; they call " + hostfile.AntigravityBinary + ", which is not installed; run loomux self-update")
			default:
				b.hostEntries(h)
			}
		}
	}
	git := b.hasGit(on)
	if git && on("git-hooks") {
		b.gitHooks()
	}
	if on("verify-skill") {
		b.skills("verify-skill", "hooks", targets)
	}
	// declares says whether area add will write the declaration, and with
	// it the consent on_merge = true: it does so only into a configuration
	// that is not there, and it runs before init writes one.
	declares := false
	if on("area") {
		switch {
		case hasArea(f.Config):
			b.note("area: skipped; .loomux/config.toml declares [area] already")
		case f.Registered:
			b.note("area: skipped; the registry has an area at this root already")
		default:
			b.action("area", "area-add", "loomux area add --scope "+c.Scope+" (wiki "+areaWiki(f.Root)+")")
			_, exists := b.file(configPath)
			declares = !exists
			if declares && slices.ContainsFunc(b.plan.Changes, func(ch Change) bool { return ch.Path == configPath }) {
				b.note(configPath + ": area-add writes [area], [layout], [index], [privacy] and [maintenance] " +
					"first; the change above is made over them")
			}
		}
	}
	if git && on("merge-hook") && !b.mergeHookThere() {
		// The hook is installed for areas that consent.
		switch {
		case !installed:
			b.note("merge-hook: skipped; the hook calls ${LOCALAPPDATA}/loomux/bin/loomux.exe, which is not installed; run loomux self-update")
		case f.HookWanted() || declares:
			b.action("merge-hook", "merge-hook", "install the post-merge hook of this project")
		case f.OnMerge:
			b.note("merge-hook: skipped; the declaration consents, but the registry of this machine has no area here")
		default:
			b.note("merge-hook: skipped; no declaration here consents with [maintenance] on_merge = true")
		}
	}
	if on("brain-skills") {
		b.skills("brain-skills", "brain", targets)
	}
	if on("graph-build") && !f.Graph {
		b.action("graph-build", "graph-build", "loomux graph build")
	}
	if b.err != nil {
		return Plan{}, b.err
	}
	return b.plan, nil
}

// areaWiki is the wiki area add picks without --wiki: docs/wiki when root has
// a docs directory, wiki otherwise. It mirrors planArea in internal/cli/area.go,
// which setup cannot import; init passes no --wiki, because the flag moves
// only the bundle and would split [layout] wiki from the registry entry.
func areaWiki(root string) string {
	if info, err := os.Stat(filepath.Join(root, "docs")); err == nil && info.IsDir() {
		return "docs/wiki"
	}
	return "wiki"
}

// file reads rel; a failed read is kept as the plan's error.
func (b *builder) file(rel string) ([]byte, bool) {
	data, exists, err := b.read(rel)
	if err != nil {
		b.fail(fmt.Errorf("%s: %w", rel, err))
		return nil, false
	}
	return data, exists
}

func (b *builder) fail(err error) {
	if b.err == nil {
		b.err = err
	}
}

func (b *builder) note(n string) { b.plan.Notes = append(b.plan.Notes, n) }

func (b *builder) action(part, id, describe string) {
	b.plan.Actions = append(b.plan.Actions, Action{Part: part, ID: id, Describe: describe})
}

func (b *builder) change(part, path string, before []byte, exists bool, after string) {
	b.redoable(part, path, before, exists, after, nil)
}

// redoable plans a change that redo makes again over the text standing when
// Apply writes it.
func (b *builder) redoable(part, path string, before []byte, exists bool, after string, redo func(string) (string, error)) {
	b.add(Change{Part: part, Path: path, Before: string(before), After: after, Exists: exists, Redo: redo})
}

// add plans c unless it leaves its file as it is.
func (b *builder) add(c Change) {
	if !c.Empty() {
		b.plan.Changes = append(b.plan.Changes, c)
	}
}

// create plans a new file at path, or names the one already there.
func (b *builder) create(part, path, text string) {
	if _, exists := b.file(path); exists {
		b.note(path + ": kept; the project has its own")
		return
	}
	b.change(part, path, nil, false, text)
}

func (b *builder) agentsMD() {
	// The embedded template always renders; its own tests hold that.
	text, _ := templates.AgentsMD(templates.Vars{
		Project:        filepath.Base(b.f.Root),
		CommitLanguage: cmp.Or(b.c.CommitLanguage, "en"),
		Hosts:          b.c.Hosts,
	})
	if _, exists := b.file("AGENTS.md"); exists {
		b.note("AGENTS.md: kept; the project has its own")
		return
	}
	// area add runs first and writes its routing rule into a missing
	// AGENTS.md; the template then goes before the rule, as area add
	// would have appended it to the template.
	b.redoable("agents-md", "AGENTS.md", nil, false, text, func(current string) (string, error) {
		return strings.TrimRight(text, "\n") + "\n\n" + current, nil
	})
}

func (b *builder) mcpJSON() {
	if b.f.UserMCP {
		b.note(mcpPath + ": left alone; the user scope of Claude Code already has a server loomux")
		return
	}
	before, exists := b.file(mcpPath)
	after, err := withMCPServer(before)
	if err != nil {
		b.fail(err)
		return
	}
	// The server entry calls the installed binary in every project.
	b.add(Change{Part: "mcp-json", Path: mcpPath, Before: string(before), After: string(after),
		Exists: exists, Binary: hostfile.Canonical})
}

func (b *builder) hostEntries(h hosts.Host) {
	path := hostfile.Path(h)
	before, exists := b.file(path)
	result, err := hostfile.Merge(h, before, hostfile.Entries(h, b.f.Binary))
	if err != nil {
		b.fail(err)
		return
	}
	for _, slot := range result.Foreign {
		b.note(path + ": " + slot + " keeps a hook of the project beside ours")
	}
	for _, n := range result.Notes {
		b.note(path + ": " + n)
	}
	// Antigravity's entries call the installed binary in every project, a
	// checkout included (hostfile.AntigravityBinary).
	binary := b.f.Binary
	if h == hosts.HostAntigravity {
		binary = hostfile.Canonical
	}
	b.add(Change{Part: "host-entries", Path: path, Before: string(before), After: string(result.Merged),
		Exists: exists, Binary: binary})
}

// hasGit says whether the project is a repository, and names the git parts
// a chosen part cannot have without one.
func (b *builder) hasGit(on func(string) bool) bool {
	if b.f.Detect.HasGit {
		return true
	}
	for _, id := range []string{"git-hooks", "merge-hook"} {
		if on(id) {
			b.note(id + ": skipped; the project is no git repository")
		}
	}
	return false
}

// gitHooks plans the three hooks into the directory hookDir picks; a hook
// already there is kept and named.
func (b *builder) gitHooks() {
	binary := b.f.Binary
	if binary == hostfile.Checkout {
		binary = checkoutGitBinary
	}
	hooks := gitfiles.Hooks(binary)
	names := make([]string, 0, len(hooks))
	for name := range hooks {
		names = append(names, name)
	}
	slices.Sort(names)
	dir, ok := b.hookDir()
	if !ok {
		return
	}
	for _, name := range names {
		path := dir + "/" + name
		existing, exists := b.file(path)
		switch {
		case exists && gitfiles.RunsAGate(string(existing)):
			b.note(path + ": kept; it runs a gate already")
		case exists:
			b.note(path + ": kept; a hook of the project is already there")
		default:
			b.add(Change{Part: "git-hooks", Path: path, After: hooks[name], Binary: b.f.Binary})
		}
	}
}

// mergeHookThere says whether our post-merge hook stands where git will run
// it after this plan: in .githooks when the plan sets core.hooksPath there,
// else in the directory git uses now.
func (b *builder) mergeHookThere() bool {
	if !slices.ContainsFunc(b.plan.Actions, func(a Action) bool { return a.ID == "hooks-path" }) {
		return b.f.MergeHook
	}
	data, _ := b.file(".githooks/post-merge")
	return maintenance.OwnsHook(data)
}

// hookDir is the directory the hooks go into, relative to the root, or
// false when init must write none: core.hooksPath where it is set; git's
// own hook directory where the project already has live hooks there,
// because core.hooksPath would silently switch them off; .githooks with
// the hooks-path action otherwise. Git's directory outside the root belongs
// to a linked worktree or a submodule and is shared with other checkouts,
// and without extensions.worktreeConfig the hooks-path action would write
// the shared configuration too, so init leaves both alone.
func (b *builder) hookDir() (string, bool) {
	switch {
	case b.f.HooksPath != "":
		rel, inside := within(b.f.Root, b.f.HooksPath)
		if !inside {
			b.note("git-hooks: skipped; core.hooksPath " + b.f.HooksPath + " lies outside the project")
		}
		return rel, inside
	case b.f.GitHooksLive:
		rel, inside := within(b.f.Root, b.f.GitHooksDir)
		if !inside {
			b.note("git-hooks: git hooks live in " + filepath.ToSlash(b.f.GitHooksDir) +
				", shared with other checkouts; init leaves them")
		}
		return rel, inside
	default:
		b.action("git-hooks", "hooks-path", "git config core.hooksPath .githooks")
		return ".githooks", true
	}
}

// within is path relative to root, slash-separated, and whether it lies
// inside root; a relative path is taken from root.
//
// Directories are compared by identity, not by spelling: git answers with
// the long spelling, and a root given as an 8.3 short name, through a
// symlink or through a junction reads differently. filepath.EvalSymlinks
// is not enough -- since Go 1.23 it leaves a junction as it is (measured
// 2026-09-24, go1.27 on Windows 11) -- while os.Stat follows all three. So
// path's directories are walked upwards until one is root itself. A root
// that does not exist is compared by its cleaned spelling.
func within(root, path string) (string, bool) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", false
		}
		return filepath.ToSlash(rel), true
	}
	var tail []string
	for dir := filepath.Clean(path); ; dir = filepath.Dir(dir) {
		if info, err := os.Stat(dir); err == nil && os.SameFile(info, rootInfo) {
			slices.Reverse(tail)
			return cmp.Or(strings.Join(tail, "/"), "."), true
		}
		if filepath.Dir(dir) == dir {
			return "", false
		}
		tail = append(tail, filepath.Base(dir))
	}
}

// skills plans the skills of module for every host that has a skill
// location.
func (b *builder) skills(part, module string, targets []hosts.Host) {
	for _, h := range targets {
		// targets holds only hosts with a hook file, and templates knows a
		// skill location, or knowingly none, for each of them.
		files, _ := templates.Skills(templates.SkillNames(module), h)
		for _, f := range files {
			b.create(part, f.Path, f.Text)
		}
	}
}
