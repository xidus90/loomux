// Package guard is the write barrier: no file-writing tool reaches outside
// a bundle or a workspace, except to a case's `proposal.md`, to the
// agents' memory and to a Claude Code session's scratchpad.
//
// It is a port of a Python module, hooks/wiki_guard.py, which was removed
// in 6bceba4 once this package replaced it. Comments below name its
// functions -- `_extract_call`, `_inside`, `_spelling` -- because that is
// where each rule came from and the wording of a refusal still matches.
// The file is readable at `git show 6bceba4^:hooks/wiki_guard.py`; line
// numbers into it are not carried any more, since nothing in a working
// tree can be checked against them.
//
// The scope of that promise is exactly `writingTools`, and it is written
// this narrowly on purpose, because the wider claim would be false.
// `Bash` is not in the list, so `sed -i`, a redirect or a here-document
// reaches every path this package refuses -- including `package.md`, the
// evidence a proposal is checked against. Nothing here stops that, and a
// session whose habits favour the shell is outside this barrier
// altogether. Widening the list is not a small change: a shell line has
// no `file_path` to read, so covering it means deciding about command
// text, which is a different problem with a different failure mode.
// The one exception is the manifest: the policy in `internal/hooks`
// refuses a shell line that writes `.loomux/config.toml`, by reading the
// command text, before this package is asked.
//
// Everything here rests on how the host reads an exit code: 2 blocks the
// tool call whatever stdout carries, while 1 is a *non-blocking* error
// after which the call proceeds. A refusal therefore leaves with 2, and
// every way this package could end otherwise -- a panic, an unreadable
// file, a dead pipe -- is a way for the barrier to open, because a Go
// panic leaves with 1. Measured on 2026-09-06. That is why `Run` catches
// everything and why no path through it may reach exit 1.
//
// A refusal writes the deny envelope to stdout as well. The host reads the
// reason out of it when the JSON parses and off stderr when it does not,
// and neither can be checked from in here, so both are written and neither
// is required to succeed. The blocking rests on the exit code alone: on
// this host a refusal that relied on the envelope was not read at all, and
// a probe file landed under `10 Rohquellen` while it sat on stdout.
//
// This package exists so the barrier has a *name* rather than a path: a host
// calls a command, not a script that binds it to a checkout and its virtual
// environment. Wrapping a script in a subprocess was the other option and was
// refused for the reason above: a process start is one more place where the
// barrier can end with the one code that lets a write through.
package guard

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/config"
)

// writingTools is `WRITING_TOOLS`.
var writingTools = map[string]bool{
	"Write":                      true,
	"Edit":                       true,
	"MultiEdit":                  true,
	"NotebookEdit":               true,
	"write_to_file":              true,
	"replace_file_content":       true,
	"multi_replace_file_content": true,
}

// targetKeys is `TARGET_KEYS`: where each writing tool puts its
// target. `Write`, `Edit` and `MultiEdit` say `file_path`; `NotebookEdit`
// says `notebook_path`; Antigravity says `TargetFile` or `target_file`.
var targetKeys = []string{
	"file_path", "notebook_path", "TargetFile", "target_file",
}

// proposalName is `PROPOSAL`: the one file a review skill writes.
// Its neighbours in the case directory -- `case.toml` and `package.md` --
// are the code's own bookkeeping and the evidence a proposal is checked
// against, and a model free to rewrite either would rewrite the very
// ground the check stands on.
const proposalName = "proposal.md"

// Call is `_extract_call`: the tool name and its
// arguments, under whichever of the two payload shapes the host uses.
func Call(payload map[string]any) (name string, args map[string]any) {
	if call, present := payload["toolCall"]; present {
		nested, ok := call.(map[string]any)
		if !ok {
			return "", nil
		}
		name, _ = nested["name"].(string)
		args, _ = nested["args"].(map[string]any)
		return name, args
	}
	name, _ = payload["tool_name"].(string)
	args, _ = payload["tool_input"].(map[string]any)
	return name, args
}

// IsWritingTool says whether a tool name is in `WRITING_TOOLS`, the whole
// scope of this barrier.
func IsWritingTool(name string) bool {
	return writingTools[name]
}

// WriteTargets is `_targets`: every path a call with these arguments would
// write, under whichever names its tool uses.
//
// All of them, not the first one found. A payload may legitimately carry
// more than one of these keys, and a barrier that judged the first would
// make the order of `targetKeys` a security property with nothing to
// record it: an allowed `file_path` beside a forbidden `notebook_path`
// would walk straight through.
//
// `_targets` opens with `if not isinstance(tool_input, dict): return ()`
// and there is no arm for it here, because in Go there is nothing for one
// to decide: reading a key out of a nil map is legal and answers the zero
// value, so an absent `tool_input` falls through the loop below and
// yields no target either way. The arm was written and struck out again
// when its mutant survived; the three tests about a `tool_input`, a
// `toolCall` and its `args` that are no object hold what matters.
func WriteTargets(args map[string]any) []string {
	found := make([]string, 0, len(targetKeys))
	for _, key := range targetKeys {
		if value, ok := args[key].(string); ok && value != "" {
			found = append(found, value)
		}
	}
	return found
}

// writableRoots is `_writable_roots`: every tree the agent may
// write in -- wiki bundles, plus declared workspaces.
//
// A workspace is where code is built, and the vault is not one of them:
// the barrier's whole point is that the session writing pages cannot
// rewrite the sources those pages cite. `readonly` is the registration's
// answer to *whether* this area may be written, so a read-only area's
// wiki is no root; the workspace half is untouched by that on purpose,
// because the field asks a different question.
//
// A registered tree this side cannot resolve comes back as an error and
// the caller refuses the whole call. Dropping the tree instead would be
// the more dangerous reading in one direction and the more annoying in
// the other -- a lost writable root closes an area nobody meant to close,
// and a lost zone opens one -- so neither is taken silently.
func writableRoots(areas []area) ([]string, error) {
	roots := make([]string, 0, len(areas)*2)
	for _, registered := range areas {
		if registered.wikiPath != "" && !registered.readOnly {
			resolved, err := ResolvePath(registered.wikiPath)
			if err != nil {
				return nil, err
			}
			roots = append(roots, resolved)
		}
		if registered.workspace {
			resolved, err := ResolvePath(registered.path)
			if err != nil {
				return nil, err
			}
			roots = append(roots, resolved)
		}
	}
	return roots, nil
}

// forbiddenRoots is `_forbidden_roots`: every tree a read-only
// area declares as its wiki, refused from above.
//
// Dropping those trees from `writableRoots` does not close them, and the
// real registration shows why: `hub` is not read-only and its wiki is
// `91 Projekte`, the *parent* of three read-only project wikis below it.
// Whatever the read-only areas are denied, an enclosing area grants
// again. So this is not a second allow-list but a zone that is decided
// first and wins against one.
func forbiddenRoots(areas []area) ([]string, error) {
	zones := make([]string, 0, len(areas))
	for _, registered := range areas {
		if registered.readOnly && registered.wikiPath != "" {
			resolved, err := ResolvePath(registered.wikiPath)
			if err != nil {
				return nil, err
			}
			zones = append(zones, resolved)
		}
	}
	return zones, nil
}

// reviewCentre is `_review_centre` composed with `review_root`
// (src/brain/maintenance/reconcile.py:196-245): where cases live, or ""
// if no area declares one.
//
// A vault without a review centre is an ordinary state, not a fault: the
// failure is confined to the proposal exemption instead of closing the
// wiki trees along with it, because a missing `layout.review` says
// nothing at all about where pages may be written. Every failure comes
// back as "" for the same reason -- the path from here reads a manifest
// off disk, a file the vault's own editor may hold open, and a refusal
// reached by traceback is no refusal at all.
func reviewCentre(areas []area, stateDir string) string {
	found := ""
	for _, registered := range areas {
		declaration := manifestPath(registered, stateDir)
		if !isRegularFile(declaration) {
			// Registering an area before it writes a manifest is normal,
			// and such an area carries nothing to compare.
			continue
		}
		read, err := config.ReadDeclaration(declaration)
		if errors.Is(err, config.ErrNoArea) {
			continue
		}
		if err != nil {
			return ""
		}
		if read.LayoutReview == "" {
			continue
		}
		root := filepath.Join(registered.path, read.LayoutReview)
		// `is_relative_to` after `resolve`, not `is_absolute`: on Windows
		// a rooted path without a drive is not called absolute while
		// joining one still replaces the area's root, and the same test
		// catches `..` on the way out, which no type check would
		// (reconcile.py:258-262). This value is the ground of the one
		// exemption, so a value reaching out of the area would make every
		// file named `proposal.md` on the disk writable.
		centre, centreErr := ResolvePath(root)
		area, areaErr := ResolvePath(registered.path)
		// A path this side cannot resolve joins the failures above: this
		// function answers "" for every one of them, which withdraws the
		// exemption and closes nothing that was open.
		if centreErr != nil || areaErr != nil {
			return ""
		}
		if !IsRelativeTo(centre, area) {
			return ""
		}
		if found != "" {
			// Two review centres are none.
			return ""
		}
		found = centre
	}
	return found
}

// isManifest is `_is_manifest`: whether this path names a
// manifest under a spelling the file system equates.
//
// `config.toml` counts only inside `.loomux`, and that condition is the
// whole difference between a lock and a nuisance: the name is an ordinary
// one, and refusing every `config.toml` in a workspace would be a limit
// nobody could derive from what this barrier is for.
func isManifest(resolved string) bool {
	name := spelling(filepath.Base(resolved))
	return name == manifestName &&
		spelling(filepath.Base(filepath.Dir(resolved))) == bundleDir
}

// isProposal is `_is_proposal`: whether this one path is a
// case proposal, the proposal exemption -- one of the exemptions outside
// the registered trees. The others are the agents' memory and the session
// scratchpad: `Decide` lets a call that writes only there through before
// the registry is read, and in a mixed call exempts those targets where
// the targets are compared with the trees.
func isProposal(resolved, centre string) bool {
	return centre != "" && filepath.Base(resolved) == proposalName &&
		IsRelativeTo(resolved, centre)
}

// declaredWikiRoot is `_declared_wiki_root`: the bundle this
// write would land in, or "" -- and an error where the declaration itself
// cannot be believed, which the caller turns into a refusal.
//
// Three conditions, all of them required: a manifest above the target, a
// scope the registration knows and does not call read-only, and a
// directory that *is* the registered tree or a worktree of it. Without
// the third, any tree could name a registered scope and write itself
// open. The walk climbs until a directory meets all three rather than
// stopping at the first manifest it finds: a file planted deeper in the
// tree fails the third condition, and skipping it is what keeps the walk
// honest.
func declaredWikiRoot(target string, areas []area) (string, error) {
	byScope := make(map[string]area, len(areas))
	for _, registered := range areas {
		byScope[registered.scope] = registered
	}
	for _, directory := range parents(target) {
		declaration := declarationIn(directory)
		if declaration == "" {
			continue
		}
		// Read once and asked twice: a second read would be a second
		// chance for the file to have changed between the two questions.
		read, err := config.ReadDeclaration(declaration)
		if errors.Is(err, config.ErrNoArea) {
			continue
		}
		if err != nil {
			return "", err
		}
		registered, known := byScope[read.Scope]
		if !known || registered.readOnly {
			continue
		}
		if !sameRepository(directory, registered.path) {
			continue
		}
		place, err := read.WikiLayout()
		if err != nil {
			return "", err
		}
		if place == "" {
			return "", nil
		}
		root, rootErr := ResolvePath(filepath.Join(directory, place))
		here, hereErr := ResolvePath(directory)
		// One arm for the two, because only one of them can ever be
		// taken: `directory` comes out of `parents` of a path that was
		// resolved before this function was called, so it resolves. The
		// declared place is the half that can lead into a circle.
		if rootErr != nil || hereErr != nil {
			return "", errors.Join(rootErr, hereErr)
		}
		// Two roles, and the quiet one matters more. The obvious role is
		// a symlinked bundle pointing at an ancestor, which would
		// otherwise open every tree beneath it. The other is
		// containment: whatever a directory manages to declare, it can
		// only ever open ground below itself -- so a planted `.git`
		// beside a planted manifest buys at most its own subtree, never
		// a neighbour's and never the vault.
		if IsRelativeTo(root, here) {
			return root, nil
		}
		return "", nil
	}
	return "", nil
}

// parents is `Path.parents`: every directory above this path, nearest
// first, ending at the anchor.
func parents(path string) []string {
	found := []string{}
	for current := filepath.Dir(path); ; {
		found = append(found, current)
		next := filepath.Dir(current)
		if next == current {
			return found
		}
		current = next
	}
}

// Decide is `decide`: "" and false to allow, or the reason and
// true to refuse. The order below is the whole construction and is not
// free to change.
//
// The path is resolved before it is compared. A prefix test on the raw
// path is exactly the hole such barriers usually have:
// `bundle/../repo/README.md` starts with the bundle and is not in it.
// Resolving also follows symlinks, so a link planted inside a bundle
// cannot lead the write back out.
//
// An unreadable or wiki-less registry refuses rather than allows, for every
// call that does not write only memory. A barrier that opens when its own
// bookkeeping fails is not a barrier --
// which is why every reader called from here answers an error rather than
// a guess, and why every one of those errors becomes the same refusal.
// Enumerating what can go wrong down there is the game this package
// cannot win; refusing whatever comes out of it is one it cannot lose.
//
// The manifest is decided before any tree is consulted, because a
// workspace area may write its entire repository -- including the
// declaration this barrier reads its own limits from.
//
// The read-only zones are decided before the allow-list: the wiki of an
// area the registration calls read-only is a forbidden zone, and a zone
// outranks every writable tree that encloses it. It has to work in that
// direction rather than as a missing entry in the allow-list, because in the real
// registration a writable area declares the parent directory of three
// read-only project wikis -- so subtracting them would change nothing.
//
// Four places lie open outside those trees -- the two agents' memory bases,
// a Claude Code session's scratchpad and the files open.toml lists (open.go),
// all on the same terms -- and one file name is exempted besides.
// A call that writes only memory is let through right after the manifest
// and before the registry, because memory is open whatever the registry
// says and a registry that cannot be read must not close it (spec
// 2026-09-13-schranke-memory-offen). In a mixed call the memory targets
// are exempted only where the targets are compared with the writable
// trees; a memory target inside a read-only zone is still refused by the
// zone check. The exempted file name is a case's `proposal.md`: the review
// skill has to put its suggestion somewhere, and the review centre lives in
// the vault rather than in a bundle. It is granted by file name, not by depth,
// because a scope may carry slashes.
func Decide(payload map[string]any, stateDir string) (string, bool) {
	toolName, args := Call(payload)
	if !IsWritingTool(toolName) {
		return "", false
	}
	targets := WriteTargets(args)
	if len(targets) == 0 {
		// Not a file write at all -- nothing here has an opinion.
		return "", false
	}
	resolved := make([]string, len(targets))
	for i, target := range targets {
		place, err := ResolvePath(target)
		if err != nil {
			// The barrier's entire answer is *where* a write lands. A
			// path with no place to land is not a call it may wave
			// through, and this is the first thing decided about a
			// target for that reason.
			return fmt.Sprintf(
				"%s: loomux cannot resolve this path, so it "+
					"refuses: %v", target, err), true
		}
		resolved[i] = place
	}
	// Before the registry and before any manifest is read, because
	// `declaredWikiRoot` reads manifests up the tree and a broken one
	// anywhere above would answer this call with the registry's complaint
	// instead. Both refuse, so nothing was ever open -- but the reason
	// would have named the wrong file.
	if named := pick(resolved, isManifest); named != "" {
		return named + ": the manifest is where the barrier reads its " +
			"own limits, so no writing tool may touch it", true
	}
	claude, antigravity := memoryBases()
	scratch := scratchpadBase()
	opened, openErr := openFiles(stateDir)
	// The session scratchpad and the files open.toml lists are open on
	// exactly the terms memory is, so they join memory here rather than being
	// asked again at each use below.
	inMemory := func(path string) bool {
		return isMemory(path, claude, antigravity) ||
			isScratchpad(path, scratch) || isOpen(path, opened)
	}
	// A refusal says why open.toml opened nothing, or a user who wrote it
	// would see the file refused and never learn it was ignored.
	ignored := ""
	if openErr != nil {
		ignored = "; " + filepath.Join(stateDir, openName) + " is ignored: " + openErr.Error()
	}
	if !slices.ContainsFunc(resolved, func(path string) bool { return !inMemory(path) }) {
		return "", false
	}
	areas, err := readRegistry(stateDir)
	if err != nil {
		return fmt.Sprintf(
			"loomux cannot read the registry, so it refuses: %v",
			err), true
	}
	roots, err := writableRoots(areas)
	if err != nil {
		return fmt.Sprintf("loomux cannot resolve a registered "+
			"tree, so it refuses: %v", err), true
	}
	zones, err := forbiddenRoots(areas)
	if err != nil {
		return fmt.Sprintf("loomux cannot resolve a registered "+
			"tree, so it refuses: %v", err), true
	}
	// Declared roots are collected per target and pooled. Each one is
	// earned on its own -- manifest, registration and repository had to
	// agree about it -- so a root one path opens can grant the other path
	// nothing it was not entitled to anyway.
	for _, path := range resolved {
		declared, err := declaredWikiRoot(path, areas)
		if err != nil {
			return fmt.Sprintf("loomux cannot read the registry,"+
				" so it refuses: %v", err), true
		}
		if declared != "" {
			roots = append(roots, declared)
		}
	}
	// Before the allow-list and before the proposal exemption, which is
	// the whole construction: an enclosing writable area would otherwise
	// grant what a read-only area denies, and a `proposal.md` would reach
	// in as well. A call that writes only memory was already let through
	// above.
	if named := pick(resolved, func(path string) bool {
		return inside(path, zones)
	}); named != "" {
		return named + ": the registration calls this area read-only, " +
			"and that outranks any writable tree around it", true
	}
	if len(roots) == 0 {
		open := []string{"the agents' memory"}
		if len(opened) > 0 {
			open = append(open, "the files "+openName+" opens")
		}
		if scratch != "" {
			open = append(open, "the session scratchpad below: "+
				filepath.Join(scratch, "*", "*", "scratchpad"))
		}
		last := len(open) - 1
		named := open[last]
		if last > 0 {
			named = strings.Join(open[:last], ", ") + " and " + open[last]
		}
		return "the registry declares no writable wiki path and no " +
			"workspace, so nothing outside " + named + " may be written" + ignored, true
	}
	outside := []string{}
	for _, path := range resolved {
		if !inside(path, roots) && !inMemory(path) {
			outside = append(outside, path)
		}
	}
	// A cost, not a decision: with nothing outside, the loop below finds
	// nothing to refuse and answers the same either way. Its mutant
	// survives every round for that reason and is left standing anyway,
	// because what it saves is one manifest read per registered area on
	// the call the host makes most often.
	if len(outside) == 0 {
		return "", false
	}
	// A linked worktree of a workspace is asked about only here, for the
	// targets nothing else opened: a write inside a registered tree has
	// already returned above and reads no git file. The zones were decided
	// before, so a worktree root cannot reopen one, and "no root at all"
	// cannot change -- a worktree root needs a workspace area, which is a
	// root itself.
	linked := linkedWorktreeRoots(outside, areas)
	roots = append(roots, linked...)
	outside = slices.DeleteFunc(outside, func(path string) bool {
		return inside(path, linked)
	})
	if len(outside) == 0 {
		return "", false
	}
	// Only now, for the same reason: finding the review centre reads a
	// manifest per area and the overwhelmingly common call is a page
	// inside a bundle.
	centre := reviewCentre(areas, stateDir)
	refused := []string{}
	for _, path := range outside {
		if !isProposal(path, centre) {
			refused = append(refused, path)
		}
	}
	if len(refused) == 0 {
		return "", false
	}
	permitted := strings.Join(roots, ", ")
	if centre != "" {
		permitted += ", plus " + proposalName + " below " + centre
	}
	if scratch != "" {
		permitted += ", plus the session scratchpad below: " +
			filepath.Join(scratch, "*", "*", "scratchpad")
	}
	if len(opened) > 0 {
		permitted += ", plus the files " + openName + " opens: " + strings.Join(opened, ", ")
	}
	if shown := memoryShown(claude, antigravity); shown != "" {
		permitted += ", plus the agents' memory below: " + shown
	}
	return strings.Join(refused, ", ") + " lies outside every writable " +
		"tree; writing is allowed only below: " + permitted + ignored, true
}

// pick names every path a test answers yes to, in the order they were
// given, or "" where none does -- the `", ".join(...)` of the three
// refusals that list their paths.
func pick(paths []string, test func(string) bool) string {
	named := []string{}
	for _, path := range paths {
		if test(path) {
			named = append(named, path)
		}
	}
	return strings.Join(named, ", ")
}
