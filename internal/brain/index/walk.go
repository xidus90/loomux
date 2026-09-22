package index

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
)

// AlwaysExcludes defines directory and file patterns that are unconditionally
// excluded from the index across all areas (Spec 5.3 & 16.12).
var AlwaysExcludes = []string{
	"**/.git/**",
	"**/.obsidian/**",
	"**/.claude/**",
	"**/.tools/**",
	"**/.superpowers/sdd/**",
	"**/node_modules/**",
	"**/tests/fixtures/**",
	"**/.brain.toml",
	"**/.ultra-brain/**",
}

// BundleArtifacts defines generated files inside a wiki bundle that must be
// excluded so index reruns do not feed on previous run outputs (Spec 14).
// Note that index.md is intentionally omitted here because it serves as the
// wiki bundle catalog (OKF §3.1) and is human-readable.
var BundleArtifacts = []string{
	"**/index.intro.md",
	"**/graph.json",
	"**/_identities.tsv",
}

// OwnArtifacts includes index.md when located outside a wiki bundle.
var OwnArtifacts = append([]string{"**/index.md"}, BundleArtifacts...)

// OwnWikiPrefix determines the area's wiki path relative to its repository root.
// Returns "" when wiki and root coincide, a posix relative path when inside root,
// or nil when the wiki is absent or outside the repository (e.g. vault placement).
func OwnWikiPrefix(area config.Area) *string {
	if area.WikiPath == "" {
		return nil
	}
	rel, err := filepath.Rel(filepath.Clean(area.Path), filepath.Clean(area.WikiPath))
	if err != nil {
		return nil
	}
	relSlash := filepath.ToSlash(rel)
	if relSlash == ".." || strings.HasPrefix(relSlash, "../") {
		return nil
	}
	if relSlash == "." {
		empty := ""
		return &empty
	}
	return &relSlash
}

// InOwnWiki checks whether a relative path lies within the area's wiki bundle.
func InOwnWiki(relative string, prefix *string) bool {
	if prefix == nil {
		return false
	}
	if *prefix == "" {
		return true
	}
	relSlash := filepath.ToSlash(relative)
	prefSlash := filepath.ToSlash(*prefix)
	return relSlash == prefSlash || strings.HasPrefix(relSlash, prefSlash+"/")
}

// ArtifactExcludes returns the generated artifact exclusions for the area.
// Read-only areas exclude no artifacts so hand-written foreign index.md and
// index.intro.md files are preserved (Decision 43).
func ArtifactExcludes(area config.Area, inWiki bool) []string {
	if area.ReadOnly {
		return nil
	}
	if inWiki {
		return BundleArtifacts
	}
	return OwnArtifacts
}

// matchesAnyGlob is `_matches_any` (src/brain/walk.py:116-117). It goes
// through the translator the read gate uses, unfolded, because the reference
// compares include and exclude as written.
func matchesAnyGlob(rel string, patterns []string) bool {
	return privacy.MatchesGlobsUnfolded(patterns, rel)
}

// FindFiles walks the area tree, returning matching files sorted deterministically.
// Excludes always-excluded directories, nested areas, review centre outputs,
// generated index artifacts, and privacy never globs.
func FindFiles(
	area config.Area,
	manifest *config.Manifest,
	nested []string,
) ([]string, error) {
	var includes []string
	var excludes []string
	var neverGlobs []string

	if manifest != nil {
		includes = manifest.IndexInclude
		excludes = append(excludes, manifest.IndexExclude...)
		neverGlobs = manifest.NeverGlobs

		reviewExc, err := privacy.ReviewExcludes(manifest)
		if err != nil {
			return nil, err
		}
		excludes = append(excludes, reviewExc...)
	}

	if len(includes) == 0 {
		includes = []string{"**/*.md"}
	}
	excludes = append(excludes, AlwaysExcludes...)

	prefix := OwnWikiPrefix(area)

	var nestedAbs []string
	for _, n := range nested {
		if abs, err := filepath.Abs(n); err == nil {
			nestedAbs = append(nestedAbs, filepath.Clean(abs))
		}
	}

	areaRootClean := filepath.Clean(area.Path)
	// The walk starts from the resolved root and hands back paths under the
	// registered one. Resolved, because a root that is a junction is
	// irregular to Go and WalkDir would not enter it: the area would have no
	// files and no error, and a reindex would write an empty register. The
	// resolver is the one `declaredReview` uses for the same field, so the
	// two readers of area.Path agree. Handed back under area.Path, because
	// every caller measures the paths against it. A junction further down is
	// still not entered -- `rglob` would, and would bring twins of whatever
	// it points at past the excludes.
	walkRoot, err := guard.ResolvePath(areaRootClean)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", areaRootClean, err)
	}

	var found []string
	err = filepath.WalkDir(walkRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		below, _ := filepath.Rel(walkRoot, path)
		pathClean := filepath.Join(areaRootClean, below)
		for _, n := range nestedAbs {
			relNested, err := filepath.Rel(n, pathClean)
			if err == nil {
				relSlash := filepath.ToSlash(relNested)
				if relSlash != ".." && !strings.HasPrefix(relSlash, "../") {
					return nil
				}
			}
		}

		relSlash := filepath.ToSlash(below)

		if !matchesAnyGlob(relSlash, includes) {
			return nil
		}
		if matchesAnyGlob(relSlash, excludes) {
			return nil
		}

		inWiki := InOwnWiki(relSlash, prefix)
		if matchesAnyGlob(relSlash, ArtifactExcludes(area, inWiki)) {
			return nil
		}

		if privacy.MatchesGlobs(neverGlobs, relSlash) {
			return nil
		}

		found = append(found, pathClean)
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(found, func(i, j int) bool {
		relI, _ := filepath.Rel(areaRootClean, found[i])
		relJ, _ := filepath.Rel(areaRootClean, found[j])
		return filepath.ToSlash(relI) < filepath.ToSlash(relJ)
	})

	return found, nil
}
