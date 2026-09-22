package hooks

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/wiki"
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
// refuses every code-only commit is no gate. ultraloom installed drift only
// where [wiki] mode = "brain" said so; `loomux wiki-gate` keeps it for whoever
// wants it. What runs at every turn end is the structure of the bundle.
//
// In `loomux check` and the stop gate alike: ultraloom ran the gate as a Stop
// entry of its own, beside the chain, with a count of its own; one lane in one
// chain is one verdict and one counter.
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
