package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/lock"
)

const areaUsage = "usage: loomux area add [--path P] [--scope S] [--wiki W] [--sources S] " +
	"[--merge-branch B] [--privacy M] [--no-reindex] [-y|--yes]\n" +
	"       loomux area check <path>"

// routingHeading opens the routing rule, and its presence alone decides
// whether the rule is written: a second copy would read as a second,
// competing rule rather than as a repetition.
const routingHeading = "## Wohin welches Wissen gehört"

// routingRule is `ROUTING_RULE` of src/brain/init.py, byte for byte. German,
// unlike everything else in this file: it is not a message about the program
// but a paragraph the program copies into a foreign repository, where it is
// read by the people and the models working there.
//
// It names `docs/wiki/` even in a repository whose bundle sits at `wiki/`;
// the question it settles -- project or shared -- does not depend on how the
// project half is spelled.
const routingRule = routingHeading + `

Wissen, das nur für dieses Projekt gilt — Architektur, Entscheidungen,
Messungen, Betriebswissen dieses Repos — kommt nach ` + "`docs/wiki/`" + `. Wissen, das
ein zweites Projekt genauso brauchen könnte — Werkzeuge, Sprachen, Verfahren,
Fremdprodukte — kommt in einen geteilten Bereich (` + "`engineering/*`" + `,
` + "`knowledge`" + `). Im Zweifel: geteilt, und aus dem Projekt per Verweis darauf
zeigen.
`

// privacyModes are the three modes a declaration reader accepts. A mode
// outside them is refused here, before anything is written, although the
// reference writes whatever it is given: a manifest with a misspelt mode makes
// the write barrier refuse every edit in the repository.
var privacyModes = []string{"automatic_cloud", "local_only", "manual_cloud"}

// areaCommand is `loomux area`. It has two subcommands, `add` and `check`; `loomux init`
// (stage 4) will call it for the half of onboarding that is not the host's.
func areaCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "check" {
		return areaCheck(args[1:], stdout, stderr)
	}
	if len(args) == 0 || args[0] != "add" {
		fmt.Fprintln(stderr, areaUsage)
		return 2
	}
	return areaAdd(args[1:], stdin, stdout, stderr)
}

// areaPlan is everything `area add` will write, settled before the first
// write so that a refusal leaves nothing behind.
type areaPlan struct {
	area config.Area
	// manifest is the new configuration, or "" when one stands and is kept.
	manifest     string
	manifestPath string
	// keptScope is the scope a kept configuration declares, "" when it
	// declares no [area].
	keptScope string
	// rule is the new project instruction, or "" when the rule is there.
	rule     string
	rulePath string
}

// areaAdd is `loomux area add`, `brain init` without its host half: the area
// into the registry, `[area]` and its neighbours into the repository's
// `.loomux/config.toml`, the routing rule into its `AGENTS.md`, the bundle
// frame, and then an index run.
//
// Three places where it departs from `run_init` (src/brain/init.py:408-450),
// each for a reason of its own:
//
//   - The registry comes first, not after the repository. The reference
//     answers a known scope by skipping it; `config.AddArea` refuses it, and
//     a refusal after the repository writes would leave the half-onboarded
//     repository `_check_entry` exists to prevent.
//   - The index run happens. `run_init` takes `run_reindex` and never reads
//     it, so `--no-reindex` is a flag without an effect there. Here the run is
//     `loomux reindex` itself, catch-up included, because both ways into the
//     index must pass the review gate the same way.
//   - `.mcp.json` and the agent hooks are not written; they are the host's,
//     and `loomux init` (stage 4) installs them.
//
// `-y` is accepted and changes nothing, exactly as in the reference, where
// `args.yes` is never read: nothing here asks, so nothing can wait for an
// answer that never comes.
func areaAdd(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux area add", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("path", "", "the repository to onboard; the working directory when empty")
	scope := flags.String("scope", "", "the area's scope; project/<directory name> when empty")
	wikiPath := flags.String("wiki", "", "the bundle directory, absolute; <repo>/<layout wiki> when empty")
	sources := flags.String("sources", "", "the source directory; docs when there is one, else .")
	branch := flags.String("merge-branch", "", "the branch merges land on; read from git when empty")
	privacy := flags.String("privacy", "manual_cloud", "the privacy mode: "+strings.Join(privacyModes, ", "))
	noReindex := flags.Bool("no-reindex", false, "skip the index run")
	yes := new(bool)
	flags.BoolVar(yes, "y", false, "accepted for the reference's sake; nothing is asked")
	flags.BoolVar(yes, "yes", false, "accepted for the reference's sake; nothing is asked")
	if err := flags.Parse(args); err != nil || refusesArguments(flags, stderr) {
		return 2
	}
	if !knownPrivacyMode(*privacy) {
		fmt.Fprintf(stderr, "loomux area add: --privacy must be one of %s, found %q\n",
			strings.Join(privacyModes, ", "), *privacy)
		return 2
	}

	plan, err := planArea(*path, *scope, *wikiPath, *sources, *branch, *privacy)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if err := config.AddArea(config.StateDir(), plan.area); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "registry: area %q added\n", plan.area.Scope)
	if err := writeArea(plan, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if *noReindex {
		return 0
	}
	return reindexCommand(nil, nil, stdout, stderr)
}

func knownPrivacyMode(mode string) bool {
	for _, known := range privacyModes {
		if mode == known {
			return true
		}
	}
	return false
}

// planArea resolves every default the way `_init` and `detect_defaults` do
// and reads the two files it may extend, refusing whatever would make an
// unusable entry or damage a file.
func planArea(path, scope, wikiPath, sources, branch, privacy string) (areaPlan, error) {
	repo := path
	// Resolved before anything is derived from it: `--path .` has the empty
	// string as its name, and the scope is built from that name.
	if !filepath.IsAbs(repo) {
		dir, err := getwd()
		if err != nil {
			return areaPlan{}, err
		}
		repo = filepath.Join(dir, repo)
	}
	if info, err := os.Stat(repo); err != nil || !info.IsDir() {
		return areaPlan{}, fmt.Errorf("%s is not a directory", repo)
	}
	if scope == "" {
		scope = "project/" + filepath.Base(repo)
	}
	for _, segment := range strings.Split(scope, "/") {
		if strings.TrimSpace(segment) == "" {
			return areaPlan{}, fmt.Errorf("scope must name every segment, found %q", scope)
		}
	}

	// The bundle is a subdirectory of the source root, never the root
	// itself, and it follows the *detected* sources even where --sources or
	// --wiki says otherwise: the flags move the bundle, while the manifest
	// records the place detection proposes (cli.py:835-838).
	detected := "."
	layoutWiki := "wiki"
	if info, err := os.Stat(filepath.Join(repo, "docs")); err == nil && info.IsDir() {
		detected, layoutWiki = "docs", "docs/wiki"
	}
	if sources == "" {
		sources = detected
	}
	if wikiPath == "" {
		wikiPath = filepath.Join(repo, filepath.FromSlash(layoutWiki))
	}
	// The registry applies machine-wide, and a relative path would be read
	// against the working directory of every reader instead of the writer's.
	if !filepath.IsAbs(wikiPath) {
		return areaPlan{}, fmt.Errorf("wiki must be absolute in a machine-wide registry, found %q", wikiPath)
	}
	if branch == "" {
		branch = detectBranch(repo)
	}

	plan := areaPlan{
		area: config.Area{
			Scope:     scope,
			Path:      filepath.ToSlash(repo),
			WikiPath:  filepath.ToSlash(wikiPath),
			Workspace: true,
		},
		manifestPath: filepath.Join(repo, ".loomux", "config.toml"),
		rulePath:     filepath.Join(repo, "AGENTS.md"),
	}
	// A configuration that stands is kept whole, as `write_manifest` keeps
	// it: it is the repository's policy and check chain as well as its
	// declaration, and a merge into it is the one step of this command that
	// could cost a repository its gate.
	_, present, err := readIfPresent(plan.manifestPath)
	if err != nil {
		return areaPlan{}, err
	}
	if !present {
		plan.manifest = renderManifest(scope, sources, layoutWiki, privacy, branch)
	} else if plan.keptScope, err = keptDeclaration(plan.manifestPath); err != nil {
		return areaPlan{}, err
	}
	instruction, _, err := readIfPresent(plan.rulePath)
	if err != nil {
		return areaPlan{}, err
	}
	if !strings.Contains(instruction, routingHeading) {
		plan.rule = withRoutingRule(instruction)
	}
	return plan, nil
}

// readIfPresent is a file's text and whether it is there. Only its absence is
// an answer; a file that is there and does not read is refused, because
// taking it for absent would write over it.
func readIfPresent(path string) (string, bool, error) {
	body, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("%s: cannot be read: %w", path, err)
	}
	return string(body), true, nil
}

// detectBranch is `_detect_branch`: the branch git names, and master where
// there is no repository at the path itself, git fails -- an unborn branch --
// or HEAD is detached.
//
// The `.git` beside the path is asked first so that a directory inside some
// other repository does not take that repository's branch.
func detectBranch(repo string) string {
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		return "master"
	}
	command := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	command.Dir = repo
	// A hook exports GIT_DIR, which outranks command.Dir; see gitenv.
	command.Env = gitenv.Environ()
	out, err := command.Output()
	name := strings.TrimSpace(string(out))
	if err != nil || name == "HEAD" {
		return "master"
	}
	return name
}

// renderManifest is `write_manifest`'s text with two changes: every value is
// quoted as TOML, and the branch goes under `branch`. The reference writes
// `merge_branch`, which its own reader never reads (`manifest.py:41` asks
// `maintenance.get("branch", "main")`), and neither does loomux's.
func renderManifest(scope, sources, layoutWiki, privacy, branch string) string {
	includes := config.QuoteTOML("**/*.md")
	if sources == "docs" {
		includes = config.QuoteTOML("docs/**/*.md") + ", " + config.QuoteTOML("README*.md")
	}
	return "[area]\n" +
		"scope = " + config.QuoteTOML(scope) + "\n" +
		"wiki = true\n" +
		"\n" +
		"[layout]\n" +
		"sources = " + config.QuoteTOML(sources) + "\n" +
		"wiki = " + config.QuoteTOML(layoutWiki) + "\n" +
		"\n" +
		"[index]\n" +
		"include = [" + includes + "]\n" +
		"\n" +
		"[privacy]\n" +
		"mode = " + config.QuoteTOML(privacy) + "\n" +
		"\n" +
		"[maintenance]\n" +
		"on_merge = true\n" +
		"branch = " + config.QuoteTOML(branch) + "\n"
}

// withRoutingRule appends the rule to an instruction with one blank line
// before its heading: markdown needs the separation, and a last line without
// a newline would swallow the heading. Only line feeds are trimmed, as
// `rstrip("\n")` trims them, so the blank line is exact for a file with LF
// endings however many it ended with. A CRLF file loses only its last line
// feed, so its last carriage return stands right before the `\n\n`: `…\r\n`
// becomes `…\r` + `\n\n`, i.e. `…\r\n\n` -- the reference's bytes, kept for
// parity.
func withRoutingRule(instruction string) string {
	if instruction == "" {
		return routingRule
	}
	return strings.TrimRight(instruction, "\n") + "\n\n" + routingRule
}

// writeArea carries the repository half of a plan out.
func writeArea(plan areaPlan, stdout, stderr io.Writer) error {
	if plan.manifest == "" {
		fmt.Fprintf(stdout, "manifest: %s (kept as it stood)\n", plan.manifestPath)
		warnAboutKeptManifest(plan, stderr)
	}
	for _, file := range []struct{ label, path, text string }{
		{"manifest", plan.manifestPath, plan.manifest},
		{"routing rule", plan.rulePath, plan.rule},
	} {
		if file.text == "" {
			continue
		}
		if err := writeInto(file.path, file.text); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "%s: %s\n", file.label, file.path)
	}
	written, err := wiki.InitBundle(filepath.FromSlash(plan.area.WikiPath))
	if err != nil {
		return err
	}
	if len(written) > 0 {
		fmt.Fprintf(stdout, "wiki: scaffolded at %s\n", filepath.FromSlash(plan.area.WikiPath))
	}
	return nil
}

// writeInto swaps text in at path, making the directory it lies in first.
func writeInto(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return lock.ReplaceText(path, text)
}

// keptDeclaration is the scope a kept configuration declares, "" for one
// that declares no [area] -- a policy-only file, which is kept and warned
// about. Any other refusal of the declaration reader refuses the command
// before its first write: the reconcile pass ahead of every index run reads
// every registered declaration and stops at one that does not read, so
// registering this area would stop `reindex` and `reconcile` for every area
// on the machine, and a second `area add` to repair it would be refused as a
// duplicate.
func keptDeclaration(path string) (string, error) {
	declared, err := config.ReadDeclaration(path)
	if errors.Is(err, config.ErrNoArea) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("%w; the area is not registered, because a declaration that does not read "+
			"stops every reconcile and reindex on this machine", err)
	}
	return declared.Scope, nil
}

// warnAboutKeptManifest says what a kept configuration means for the area
// just registered. The file is not touched -- it is the repository's policy
// and check chain as well as its declaration -- so what it lacks is for a
// person to add, and they are owed the sentence.
func warnAboutKeptManifest(plan areaPlan, stderr io.Writer) {
	switch plan.keptScope {
	case "":
		fmt.Fprintf(stderr, "warning: %s declares no [area]; add [area] scope = %s and [layout] wiki by hand, "+
			"or the area is registered and never indexed\n", plan.manifestPath, config.QuoteTOML(plan.area.Scope))
	case plan.area.Scope:
		// The configuration already declares this very area: nothing to say.
	default:
		fmt.Fprintf(stderr, "warning: %s declares scope %q, the registry now names %q\n",
			plan.manifestPath, plan.keptScope, plan.area.Scope)
	}
}
