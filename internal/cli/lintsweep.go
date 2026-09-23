package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/check"
	checkrun "github.com/xidus90/loomux/internal/brain/check/run"
	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/config"
)

// lintSweep is `_lint`: the registered areas the scope names, each linted by
// the rules of `lint.py`, printed under its scope, then one line counting
// them. Age alone never fails the run; only an error finding does.
//
// A refusal about the registry or an area is `error: …` and 1, as `main`
// turns a LookupError into it. A declaration that does not read stops the run
// where it stands, after the areas already printed.
func lintSweep(scope string, stdout, stderr io.Writer) int {
	lookup := config.NewArtifactLookup()
	areas, err := config.ReadRegistry(lookup.Primary)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	targets, err := checkrun.Targets(areas, scope)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	shared := map[string]bool{}
	for _, area := range areas {
		if area.Shared {
			shared[area.Scope] = true
		}
	}
	var findings []check.Finding
	for _, area := range targets {
		ctx, err := sweepContext(area, areas, shared, lookup)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		found, err := wiki.SweepBundle(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, area.Scope)
		for _, f := range found {
			fmt.Fprintf(stdout, "  %s:%s: %s\n", f.Relative, f.Rule, f.Message)
		}
		if len(found) == 0 {
			fmt.Fprintln(stdout, "  no findings")
		}
		findings = append(findings, found...)
	}
	if len(findings) == 0 {
		fmt.Fprintln(stdout, "no findings")
		return 0
	}
	errorCount := 0
	for _, f := range findings {
		if f.Severity == check.Error {
			errorCount++
		}
	}
	fmt.Fprintf(stdout, "%d findings (%d errors, %d warnings)\n", len(findings), errorCount, len(findings)-errorCount)
	if errorCount > 0 {
		return 1
	}
	return 0
}

// sweepContext is what `_lint` hands `lint_bundle` for one area: the
// threshold and the vocabulary of its declaration, the shared set, whether it
// is a project, and -- for the signpost alone -- what it must link.
func sweepContext(area config.Area, areas []config.Area, shared map[string]bool, lookup config.ArtifactLookup) (wiki.SweepContext, error) {
	ctx := wiki.SweepContext{
		Root:          area.WikiPath,
		UntouchedDays: config.DefaultUntouchedDays,
		Now:           time.Now(),
		IsShared:      area.Shared,
		SharedScopes:  shared,
		DeclaredTypes: map[string]bool{},
		IsProject:     strings.SplitN(area.Scope, "/", 2)[0] == "project",
	}
	manifest, err := config.ReadAreaManifestUntilStage4(config.ResolvedAreaDir(area, lookup.Primary, lookup.Fallback))
	switch {
	case errors.Is(err, config.ErrNoManifest):
		manifest = nil
	case err != nil:
		return ctx, err
	}
	hub := ""
	if manifest != nil {
		ctx.UntouchedDays = manifest.UntouchedDays
		for _, t := range manifest.DeclaredTypes {
			ctx.DeclaredTypes[t] = true
		}
		if hub, err = manifest.HubLayout(); err != nil {
			return ctx, err
		}
	}
	if area.Signpost {
		ctx.ExpectedTargets = expectedTargets(areas, area, hub)
	}
	return ctx, nil
}

// expectedTargets is `_expected_targets`: every other area with a wiki, named
// by its wiki or, when the signpost declares a hub folder, by the hub page
// that carries the last segment of its scope.
func expectedTargets(areas []config.Area, signpost config.Area, hub string) []wiki.ExpectedTarget {
	var out []wiki.ExpectedTarget
	for _, area := range areas {
		if area.Scope == signpost.Scope || area.WikiPath == "" {
			continue
		}
		targets := []string{wiki.AbsoluteClean(area.WikiPath)}
		if hub != "" {
			name := area.Scope[strings.LastIndex(area.Scope, "/")+1:] + ".md"
			targets = append(targets, wiki.AbsoluteClean(filepath.Join(signpost.Path, filepath.FromSlash(hub), name)))
		}
		out = append(out, wiki.ExpectedTarget{Scope: area.Scope, Targets: targets})
	}
	return out
}
