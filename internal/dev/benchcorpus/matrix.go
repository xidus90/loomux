package benchcorpus

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
)

// MergeAudits upserts an incoming RepoAudit into an existing list of audits,
// deduplicating by RepoSlug and sorting deterministically by LanguageSlug, Framework, and RepoSlug.
// A single-repo run knows nothing about the corpus, so descriptive fields it
// leaves empty are taken from the audit it replaces; incoming is filled in place
// so the detail page written from it carries them too.
func MergeAudits(existing []*RepoAudit, incoming *RepoAudit) []*RepoAudit {
	incomingSlug := RepoSlug(incoming)
	var merged []*RepoAudit
	replaced := false

	for _, a := range existing {
		if RepoSlug(a) == incomingSlug {
			inheritMetadata(incoming, a)
			merged = append(merged, incoming)
			replaced = true
		} else {
			merged = append(merged, a)
		}
	}
	if !replaced {
		merged = append(merged, incoming)
	}

	sortAudits(merged)
	return merged
}

func inheritMetadata(dst, src *RepoAudit) {
	for _, f := range []struct{ dst, src *string }{
		{&dst.RepoURL, &src.RepoURL},
		{&dst.Language, &src.Language},
		{&dst.Framework, &src.Framework},
		{&dst.Tier, &src.Tier},
		{&dst.CommitSHA, &src.CommitSHA},
	} {
		if *f.dst == "" {
			*f.dst = *f.src
		}
	}
}

func sortAudits(audits []*RepoAudit) {
	slices.SortFunc(audits, func(a, b *RepoAudit) int {
		langA := LanguageSlug(a.Language, a.DetectedStacks)
		langB := LanguageSlug(b.Language, b.DetectedStacks)
		if langA != langB {
			return strings.Compare(langA, langB)
		}
		if a.Framework != b.Framework {
			return strings.Compare(a.Framework, b.Framework)
		}
		return strings.Compare(RepoSlug(a), RepoSlug(b))
	})
}

// FormatMatrixMarkdown formats a collection of repository benchmark audits into a central matrix.
func FormatMatrixMarkdown(report *BenchmarkReport, lang string, w io.Writer) error {
	isDE := lang == "de"
	audits := make([]*RepoAudit, len(report.Repos))
	copy(audits, report.Repos)
	sortAudits(audits)

	var totalCoverage float64
	var totalSpeedup float64
	var speedupCount int

	for _, a := range audits {
		totalCoverage += a.CoverageRate
		if a.Speedup > 0 {
			totalSpeedup += a.Speedup
			speedupCount++
		}
	}

	avgCoverage := 0.0
	if len(audits) > 0 {
		avgCoverage = totalCoverage / float64(len(audits))
	}
	avgSpeedup := 0.0
	if speedupCount > 0 {
		avgSpeedup = totalSpeedup / float64(speedupCount)
	}

	if isDE {
		fmt.Fprintf(w, "# Loomux Open-Source Benchmark-Matrix\n\n")
		fmt.Fprintln(w, "> Umfassender Performance- und Lücken-Vergleich aller getesteten Open-Source-Projekte (Top-Sprachen und Frameworks).")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "- **Letzte Aktualisierung:** %s\n", report.Timestamp)
		fmt.Fprintf(w, "- **Gesamtzahl getesteter Repositories:** %d\n", len(audits))
		if avgSpeedup > 0 {
			fmt.Fprintf(w, "- **Durchschnittlicher Hook-Speedup:** %.1fx\n", avgSpeedup)
		}
		fmt.Fprintf(w, "- **Durchschnittliche Abdeckungsquote:** %.1f %%\n\n", avgCoverage)

		fmt.Fprintln(w, "| Sprache | Framework | Repository | Sterne-Tier | Pre-Tool (warm) | Post-Tool (warm) | Graph Build | Claude Speedup | Abdeckung | Detailbericht |")
		fmt.Fprintln(w, "|---|---|---|---|---:|---:|---:|---:|---:|---|")
	} else {
		fmt.Fprintf(w, "# Loomux Open-Source Benchmark Matrix\n\n")
		fmt.Fprintln(w, "> Comprehensive performance and gap comparison of tested open-source projects across top languages and frameworks.")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "- **Last Updated:** %s\n", report.Timestamp)
		fmt.Fprintf(w, "- **Total Repositories Tested:** %d\n", len(audits))
		if avgSpeedup > 0 {
			fmt.Fprintf(w, "- **Average Hook Speedup:** %.1fx\n", avgSpeedup)
		}
		fmt.Fprintf(w, "- **Average Coverage Rate:** %.1f %%\n\n", avgCoverage)

		fmt.Fprintln(w, "| Language | Framework | Repository | Star Tier | Pre-Tool (warm) | Post-Tool (warm) | Graph Build | Claude Speedup | Coverage | Details |")
		fmt.Fprintln(w, "|---|---|---|---|---:|---:|---:|---:|---:|---|")
	}

	for _, a := range audits {
		langSlug := LanguageSlug(a.Language, a.DetectedStacks)
		repoSlug := RepoSlug(a)

		langDisplay := a.Language
		if langDisplay == "" {
			langDisplay = langSlug
		}
		fwDisplay := a.Framework
		if fwDisplay == "" {
			fwDisplay = "-"
		}
		tierDisplay := a.Tier
		if tierDisplay == "" {
			tierDisplay = "-"
		}

		repoDisplayStr := fmt.Sprintf("`%s`", repoSlug)

		preToolStr := formatComponentTiming(a, "pre-tool-use")
		postToolStr := formatComponentTiming(a, "post-tool-use")
		graphBuildStr := formatComponentTiming(a, "graph build")

		speedupStr := "-"
		if a.Speedup > 0 {
			speedupStr = fmt.Sprintf("%.1fx", a.Speedup)
		}

		coverageStr := fmt.Sprintf("%.1f %%", a.CoverageRate)
		detailsLink := fmt.Sprintf("[Details](%s/%s.md)", langSlug, repoSlug)

		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			langDisplay, fwDisplay, repoDisplayStr, tierDisplay, preToolStr, postToolStr, graphBuildStr, speedupStr, coverageStr, detailsLink)
	}

	if len(report.Skipped) > 0 {
		// A list, not a table: git's reasons quote paths, and a path may hold a `|`.
		if isDE {
			fmt.Fprintln(w, "\n## Übersprungene Repositories")
		} else {
			fmt.Fprintln(w, "\n## Skipped Repositories")
		}
		fmt.Fprintln(w)
		for _, s := range report.Skipped {
			fmt.Fprintf(w, "- `%s` (%s / %s): %s\n", RepoSlug(&RepoAudit{RepoURL: s.RepoURL}), s.Language, s.Framework, s.Reason)
		}
	}

	return nil
}

func formatComponentTiming(audit *RepoAudit, name string) string {
	cIdx := -1
	for i, c := range audit.Cold.Components {
		if c.Name == name {
			if !c.Applicable {
				return "n/a"
			}
			cIdx = i
			break
		}
	}
	if cIdx == -1 {
		return "n/a"
	}

	var warmTimes []time.Duration
	for _, wRun := range audit.Warm {
		if cIdx < len(wRun.Components) {
			warmTimes = append(warmTimes, wRun.Components[cIdx].Elapsed)
		}
	}
	if len(warmTimes) == 0 {
		return "n/a"
	}
	slices.Sort(warmTimes)
	med := calculateMedian(warmTimes)
	return formatDuration(med)
}
