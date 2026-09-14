package wiki

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/check"
)

var defaultValidTypes = map[string]bool{
	"concept":      true,
	"guide":        true,
	"decision":     true,
	"reference":    true,
	"architecture": true,
	"log":          true,
	"person":       true,
	"domain":       true,
	"system":       true,
	"component":    true,
	"spec":         true,
	"topic":        true,
}

// The findings of this file and of LintBundle leave Axis empty. Their five
// rules predate the split into an OKF axis and a house axis, and sorting them
// into the two is the work of the checks under pkg/check/okf and
// pkg/check/house -- guessing an axis here would put an answer into the model
// that nobody has given yet. Nobody reads the axis in the meantime: brain lint
// and the wiki gate go by Severity, Rule, Relative and Message.
func LintSingleFile(filePath, wikiRoot string) ([]check.Finding, error) {
	if IsScaffoldFile(filePath) {
		return nil, nil
	}

	page, err := ReadPage(filePath, wikiRoot)
	if err != nil {
		return nil, err
	}

	var findings []check.Finding

	// 1. Missing / Invalid Type
	// Read from the model, not from page.Frontmatter: the raw block is the
	// reader's container, and a rule going by it would judge a second copy of
	// the same value.
	//
	// This rule still folds an absent `type` together with an empty one: both
	// end up as "" below and get the same message. `WikiPage.PageType` keeps
	// the two apart, and `wiki/lint.py:97-109` gives them different messages
	// -- so the distinction the model gained is not yet cashed in here. This
	// rule predates the OKF split and gets replaced by the checks under
	// pkg/check/okf; whoever writes those owes the second message.
	var t string
	if page.PageType != nil {
		t = strings.ToLower(strings.TrimSpace(*page.PageType))
	}
	if t == "" {
		findings = append(findings, check.Finding{
			Relative: page.Relative,
			Rule:     "missing-type",
			Severity: check.Error,
			Message:  "document is missing required 'type' in frontmatter",
		})
	} else if !defaultValidTypes[t] {
		findings = append(findings, check.Finding{
			Relative: page.Relative,
			Rule:     "missing-type",
			Severity: check.Warning,
			Message:  fmt.Sprintf("unknown document type %q", t),
		})
	}

	// 2. Conflict Count
	if page.DeclaredConflicts == nil {
		if page.FoundConflicts > 0 {
			findings = append(findings, check.Finding{
				Relative: page.Relative,
				Rule:     "conflict-count",
				Severity: check.Error,
				Message:  fmt.Sprintf("%d conflict box(es) present, open_conflicts is missing in frontmatter", page.FoundConflicts),
			})
		}
	} else if *page.DeclaredConflicts != page.FoundConflicts {
		findings = append(findings, check.Finding{
			Relative: page.Relative,
			Rule:     "conflict-count",
			Severity: check.Error,
			Message:  fmt.Sprintf("open_conflicts says %d, but %d conflict box(es) found in document", *page.DeclaredConflicts, page.FoundConflicts),
		})
	}

	return findings, nil
}

//coverage:exempt the WalkDir err arm needs the callback to return an error, and the callback returns nil on every path, including the error WalkDir hands it
func LintBundle(wikiRoot string) ([]check.Finding, error) {
	var pages []*WikiPage
	var findings []check.Finding

	inboundLinks := make(map[string]int)

	err := filepath.WalkDir(wikiRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		page, err := ReadPage(path, wikiRoot)
		if err != nil {
			return nil
		}

		pages = append(pages, page)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// First pass: collect pages and link targets
	for _, page := range pages {
		// Whether this page is a *subject* of the rules below. Scaffold
		// files are not: `_schema.md` is the bundle's own example document,
		// so its `/pfad/a.md` is material to look at and not a target -- and
		// judging it made `brain wiki-gate` report two dead links on a
		// bundle `brain check` calls clean. The new catalog draws the same
		// line for the same six rules and gives the reason
		// (`pkg/check/house/bundle.go:58-77`).
		//
		// A subject, not a source: the counter below keeps counting the
		// links of every page, scaffold included, because a catalog is a
		// valid source of links for the orphan rule
		// (`pkg/wiki/parse.go:195-196`). Reading the exemption as "skip the
		// page" instead would turn every page that only its `index.md`
		// lists into an orphan.
		judged := !IsScaffoldFile(page.Relative)
		if judged {
			singleFindings, _ := LintSingleFile(page.Path, wikiRoot)
			findings = append(findings, singleFindings...)
		}

		// Dead Link Checks
		pageDir := filepath.Dir(page.Path)
		for _, rawLink := range page.Links {
			u, err := url.Parse(rawLink)
			if err != nil || u.Scheme != "" {
				continue
			}

			targetPath := u.Path
			if targetPath == "" {
				continue
			}

			var resolved string
			if strings.HasPrefix(targetPath, "/") {
				resolved = filepath.Join(wikiRoot, strings.TrimPrefix(targetPath, "/"))
			} else {
				resolved = filepath.Join(pageDir, targetPath)
			}

			// Clean and check existence
			resolved = filepath.Clean(resolved)
			relToWiki, _ := filepath.Rel(wikiRoot, resolved)
			relToWiki = filepath.ToSlash(relToWiki)

			if strings.HasPrefix(relToWiki, "..") {
				// Outside area
				if judged {
					findings = append(findings, check.Finding{
						Relative: page.Relative,
						Rule:     "outside-area",
						Severity: check.Warning,
						Message:  fmt.Sprintf("target leaves the wiki area: %s", rawLink),
					})
				}
			} else if _, err := os.Stat(resolved); os.IsNotExist(err) {
				if judged {
					findings = append(findings, check.Finding{
						Relative: page.Relative,
						Rule:     "dead-link",
						Severity: check.Error,
						Message:  fmt.Sprintf("target does not exist: %s (resolved: %s)", rawLink, relToWiki),
					})
				}
			} else {
				inboundLinks[relToWiki]++
			}
		}
	}

	// Second pass: Orphan detection
	for _, page := range pages {
		if IsScaffoldFile(page.Relative) {
			continue
		}
		if inboundLinks[page.Relative] == 0 {
			findings = append(findings, check.Finding{
				Relative: page.Relative,
				Rule:     "orphan",
				Severity: check.Warning,
				Message:  "page is not linked by any other page in the wiki bundle",
			})
		}
	}

	return findings, nil
}
