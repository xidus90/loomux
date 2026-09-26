package benchcorpus

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// FormatMarkdown formats a BenchmarkReport as a deterministic Markdown document.
func FormatMarkdown(report *BenchmarkReport, w io.Writer) error {
	fmt.Fprintf(w, "# Loomux Benchmark & Lücken-Audit\n\n")
	fmt.Fprintf(w, "- **Datum:** %s\n", report.Timestamp)
	fmt.Fprintf(w, "- **Modus:** %s\n", report.Mode)
	fmt.Fprintf(w, "- **Methode:** 1x kalt, %dx warmer Median (gemäß benchmarks.md-Standard)\n\n", report.WarmRuns)

	for i, repo := range report.Repos {
		if len(report.Repos) > 1 {
			fmt.Fprintf(w, "## Repository #%d: %s\n\n", i+1, repoDisplay(repo))
			if repo.Language != "" {
				fmt.Fprintf(w, "- **Sprache:** %s | **Tier:** %s\n", repo.Language, repo.Tier)
			}
			if repo.CommitSHA != "" {
				fmt.Fprintf(w, "- **Commit:** %s\n", repo.CommitSHA)
			}
			fmt.Fprintln(w)
		}

		if repo.SampleFile != "" {
			fmt.Fprintf(w, "- **Beispieldatei:** `%s`\n\n", repo.SampleFile)
		}
		fmt.Fprintln(w, "### 1. Performance & Latenzen")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |")
		fmt.Fprintln(w, "|---|---:|---:|---:|---:|---|")

		for _, comp := range repo.Components() {
			if comp.Applicable != nil && !*comp.Applicable {
				fmt.Fprintf(w, "| **%s** | n/a | n/a | n/a | n/a | n/a |\n", comp.Name)
				continue
			}
			fmt.Fprintf(w, "| **%s** | %s | %s |\n", comp.Name, timingCells(comp), exitStatus(comp))
		}

		total, _ := repo.Timing(TotalTiming)
		fmt.Fprintf(w, "| **Gesamt** | %s | %s |\n", timingCells(total), exitStatus(allRuns(repo)...))

		if repo.ClaudeWarmMed > 0 {
			fmt.Fprintf(w, "\n- **Baseline Claude Hook:** %s%s (Speedup: %.1fx%s)\n", benchreport.FormatMS(repo.ClaudeWarmMed), hookComparison(repo), repo.Speedup, baselineStatus(repo))
		} else if repo.BaselineError != "" {
			fmt.Fprintf(w, "\n- **Baseline Claude Hook:** %s (%s)\n", "nicht verfügbar", repo.BaselineError)
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, "### 2. Test- & Lücken-Audit (Gap Analysis)")
		fmt.Fprintln(w)

		if len(repo.Audit) > 0 {
			fmt.Fprintln(w, "| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |")
			fmt.Fprintln(w, "|---|---|---|---|---|---|")
			for _, check := range repo.Audit {
				pathStr := "Nein"
				if check.OnPath {
					pathStr = "Ja"
				}
				laneStr := check.Lane
				if laneStr == "" {
					laneStr = "*keine*"
				}
				statusStr := "✅ Aktiv"
				if check.Lane == "" {
					statusStr = "⚠️ Fehlt in Loomux"
				} else if !check.OnPath {
					statusStr = "❌ Nicht im PATH"
				}
				fmt.Fprintf(w, "| `%s` | %s | `%s` | `%s` | %s | %s |\n",
					check.Tool, check.Category, check.Native, laneStr, pathStr, statusStr)
			}
			fmt.Fprintln(w)
		}

		fmt.Fprintf(w, "- **Abdeckungsquote:** **%.1f %%**\n", repo.CoverageRate)
		if len(repo.MissingGaps) > 0 {
			fmt.Fprintln(w, "- **Identifizierte Lücken (Gaps):**")
			for _, gap := range repo.MissingGaps {
				fmt.Fprintf(w, "  - ⚠️ %s\n", gap)
			}
		}
		fmt.Fprintln(w)
	}

	return nil
}

func repoDisplay(repo *RepoAudit) string {
	if repo.RepoURL != "" {
		return repo.RepoURL
	}
	return repo.Dir
}

// runPayload is what a repos run adds to the shared report: the full rows,
// and the repositories it could not measure.
type runPayload struct {
	Repos   []*RepoAudit  `json:"repos"`
	Skipped []SkippedRepo `json:"skipped"`
}

// ReportJSON is the shared report of a run as written next to its markdown.
// Its timings are each repository's total, named as its markdown section is,
// so two runs compare without reading the payload.
func ReportJSON(report *BenchmarkReport, stamp string, env benchreport.Environment) ([]byte, error) {
	payload, err := json.Marshal(runPayload{
		Repos:   append([]*RepoAudit{}, report.Repos...),
		Skipped: append([]SkippedRepo{}, report.Skipped...),
	})
	if err != nil {
		return nil, err
	}
	totals := []benchreport.Timing{}
	for _, repo := range report.Repos {
		if total, ok := repo.Timing(TotalTiming); ok {
			total.Name = repoDisplay(repo)
			totals = append(totals, total)
		}
	}
	// The head holds nothing the payload did not already encode, so it
	// cannot fail where the payload succeeded.
	data, _ := benchreport.Report{Schema: benchreport.Schema, Command: "repos", Stamp: stamp,
		Environment: env, Timings: totals, Payload: payload}.JSON()
	return data, nil
}
