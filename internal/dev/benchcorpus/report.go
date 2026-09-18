package benchcorpus

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"time"
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

		for cIdx, comp := range repo.Cold.Components {
			if !comp.Applicable {
				fmt.Fprintf(w, "| **%s** | n/a | n/a | n/a | n/a | n/a |\n", comp.Name)
				continue
			}

			coldStr := formatDuration(comp.Elapsed)
			var warmCompTimes []time.Duration
			for _, wRun := range repo.Warm {
				if cIdx < len(wRun.Components) {
					warmCompTimes = append(warmCompTimes, wRun.Components[cIdx].Elapsed)
				}
			}
			slices.Sort(warmCompTimes)
			medStr := formatDuration(calculateMedian(warmCompTimes))
			minStr := formatDuration(warmCompTimes[0])
			maxStr := formatDuration(warmCompTimes[len(warmCompTimes)-1])

			fmt.Fprintf(w, "| **%s** | %s | %s | %s | %s | %s |\n", comp.Name, coldStr, medStr, minStr, maxStr, exitStatus(componentRuns(repo.Cold, repo.Warm, cIdx)))
		}

		totalCold := formatDuration(repo.Cold.Total)
		totalMed := formatDuration(repo.WarmMedian)
		totalMin := formatDuration(repo.WarmMin)
		totalMax := formatDuration(repo.WarmMax)
		fmt.Fprintf(w, "| **Gesamt** | %s | %s | %s | %s | %s |\n", totalCold, totalMed, totalMin, totalMax, exitStatus(allRuns(repo.Cold, repo.Warm)))

		if repo.ClaudeWarmMed > 0 {
			fmt.Fprintf(w, "\n- **Baseline Claude Hook:** %s%s (Speedup: %.1fx%s)\n", formatDuration(repo.ClaudeWarmMed), hookComparison(repo), repo.Speedup, baselineStatus(repo))
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

func formatDuration(d time.Duration) string {
	ms := float64(d) / float64(time.Millisecond)
	return fmt.Sprintf("%.1f ms", ms)
}

// FormatJSON formats a BenchmarkReport as formatted JSON.
func FormatJSON(report *BenchmarkReport, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
