package hooks

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/verify"
)

// WikiGateJobs is the wiki's lane for a whole project: the lint over the
// bundle, in this process. It runs where lint is asked for, the project has a
// wiki, and [verify.wiki] lint = false did not switch it off -- the same three
// conditions the edit lane has, with the whole bundle in place of the edited
// page.
//
// The bundle and not the drift rule `loomux wiki-gate` also carries: drift is
// a judgement about whether documentation accompanies code, and a lane that
// refuses every code-only commit is no gate. `loomux wiki-gate` keeps drift
// for whoever wants it. What runs at every turn end is the structure of the bundle.
//
// In `loomux check` and the stop gate alike: one lane in one chain is one
// verdict and one counter.
//
// The facts come from the caller, who has them already: a second
// detect.Detect would be a second walk of the tree.
func WikiGateJobs(eff verify.Effective, facts detect.Facts, root string, kinds []string) []verify.Job {
	if !slices.Contains(kinds, "lint") || eff.Config.Stacks["wiki"]["lint"].Lane.Off {
		return nil
	}
	if !slices.Contains(stacksWithWiki(facts.Stacks, root), "wiki") {
		return nil
	}
	// The bundle directory and not the project root, as the edit lane
	// resolves it: the manifest's [layout] wiki outranks what detection
	// guessed.
	wikiRoot := filepath.Join(root, filepath.FromSlash(wikiDirFor(root)))
	return []verify.Job{{
		Name: "lint/wiki", Kind: "lint", Stack: "wiki", Area: ".", Origin: "in-process", Dir: root, After: -1,
		Fn: func() (string, error) { return lintBundle(wikiRoot, wiki.LintBundle) },
	}}
}

// lintBundle reports the bundle's errors the way GateReport reports its
// wiki-lint violations, so the two say the same thing about the same finding.
// A warning is written by neither: the gate counted errors alone.
//
// The lint arrives as a parameter, because wiki.LintBundle reports an error
// no bundle on disk can provoke -- its walk swallows every entry it cannot
// read -- and a refusal nobody can reach is a refusal nobody has read.
func lintBundle(wikiRoot string, lint func(string) ([]check.Finding, error)) (string, error) {
	findings, err := lint(wikiRoot)
	if err != nil {
		return "", fmt.Errorf("linting the wiki bundle: %w", err)
	}
	var report strings.Builder
	errorCount := 0
	for _, f := range findings {
		if f.Severity != check.Error {
			continue
		}
		errorCount++
		fmt.Fprintf(&report, "  • [wiki-lint:%s] %s: %s\n", f.Rule, f.Relative, f.Message)
	}
	if errorCount > 0 {
		return report.String(), fmt.Errorf("the wiki bundle has %d error(s)", errorCount)
	}
	return "", nil
}

// WikiSourceJobs is the lane of the wiki's sources: the pins of the
// project's own scope against the register at the root, and the register
// against the files they cite. Its own stack, so its own lane key: in
// probation apart from the bundle lint, and switched off on its own with
// [verify.wiki-sources] lint = false.
//
// In `loomux check` and the gate's lane list, never at the turn end: a source
// changes over several turns, and its citing pages are checked and repinned
// once, before the commit.
func WikiSourceJobs(eff verify.Effective, facts detect.Facts, root string, kinds []string) []verify.Job {
	if !slices.Contains(kinds, "lint") || eff.Config.Stacks["wiki-sources"]["lint"].Lane.Off {
		return nil
	}
	if !slices.Contains(stacksWithWiki(facts.Stacks, root), "wiki") {
		return nil
	}
	wikiRoot := filepath.Join(root, filepath.FromSlash(wikiDirFor(root)))
	return []verify.Job{{
		Name: "lint/wiki-sources", Kind: "lint", Stack: "wiki-sources", Area: ".", Origin: "in-process", Dir: root, After: -1,
		Fn: func() (string, error) { return sourceLane(root, wikiRoot, wiki.SourceDrift) },
	}}
}

// sourceHints says per rule what to do, in the order of the report.
var sourceHints = []struct{ rules, hint string }{
	{"row-missing", "repin adds no register row; a newly cited file gets one from `loomux reindex` in the registry's checkout (see `loomux wiki repin` in the CLI reference)"},
	{"source-missing, doc-id", "correct the pin by hand"},
	{"cycle", "pages that pin each other can never both be current; drop one of the pins"},
	{"register-stale, pin-stale", "check the citing pages against the changed files, fix them, then run `loomux wiki repin`"},
}

// sourceLane reports the findings the way lintBundle reports the bundle's,
// then one line per kind of finding present saying what to do. The drift
// arrives as a parameter for the failure no tree provokes.
func sourceLane(root, wikiRoot string, drift func(projectRoot, wikiRoot, scope string) ([]wiki.SourceFinding, error)) (string, error) {
	// A manifest that does not read counts as none, as the lint reads it;
	// the write barrier is what refuses a broken one.
	scope := ""
	if manifest, err := config.ReadManifest(root); err == nil {
		scope = manifest.Scope
	}
	findings, err := drift(root, wikiRoot, scope)
	if err != nil {
		return "", fmt.Errorf("checking the wiki sources: %w", err)
	}
	if len(findings) == 0 {
		return "", nil
	}
	var report strings.Builder
	present := map[string]bool{}
	for _, f := range findings {
		fmt.Fprintf(&report, "  • [wiki-sources:%s] %s: %s\n", f.Rule, f.Page, f.Message)
		present[f.Rule] = true
	}
	for _, h := range sourceHints {
		for _, rule := range strings.Split(h.rules, ", ") {
			if present[rule] {
				fmt.Fprintf(&report, "  %s: %s\n", h.rules, h.hint)
				break
			}
		}
	}
	return report.String(), fmt.Errorf("the wiki sources have %d finding(s)", len(findings))
}
