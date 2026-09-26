package benchcorpus

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// RepoSlug returns a clean, filesystem-safe identifier for the repository.
func RepoSlug(audit *RepoAudit) string {
	if audit.RepoURL != "" {
		trimmed := strings.TrimSuffix(audit.RepoURL, ".git")
		parts := strings.Split(trimmed, "://")
		urlPath := parts[len(parts)-1]
		subParts := strings.Split(urlPath, "/")
		if len(subParts) >= 3 {
			return sanitizeRepoDir(subParts[len(subParts)-2] + "_" + subParts[len(subParts)-1])
		}
		return sanitizeRepoDir(subParts[len(subParts)-1])
	}
	if audit.Dir == "." || audit.Dir == "" {
		return "loomux"
	}
	base := filepath.Base(audit.Dir)
	if base == "." || base == "/" || base == "\\" || base == "" {
		return "loomux"
	}
	return sanitizeRepoDir(base)
}

// LanguageSlug normalizes language names to clean lowercase directory slugs.
func LanguageSlug(lang string, stacks []string) string {
	cleaned := strings.ToLower(strings.TrimSpace(lang))
	switch cleaned {
	case "python", "py":
		return "python"
	case "go", "golang":
		return "go"
	case "javascript", "js", "node":
		return "javascript"
	case "typescript", "ts":
		return "typescript"
	case "rust", "rs":
		return "rust"
	case "c#", "csharp", "dotnet":
		return "csharp"
	case "c++", "cpp":
		return "cpp"
	case "java":
		return "java"
	case "php":
		return "php"
	case "ruby", "rb":
		return "ruby"
	case "c":
		return "c"
	}
	if cleaned != "" {
		return cleaned
	}
	known := []string{"python", "go", "javascript", "typescript", "rust", "csharp", "cpp", "java", "php", "ruby", "c"}
	for _, s := range stacks {
		sLower := strings.ToLower(s)
		for _, k := range known {
			if sLower == k || strings.Contains(sLower, k) {
				return k
			}
		}
	}
	return "other"
}

// FormatDetailMarkdown formats a single repository audit as a rich detail markdown page.
func FormatDetailMarkdown(audit *RepoAudit, lang string, w io.Writer) error {
	isDE := lang == "de"

	titleName := RepoSlug(audit)
	if audit.RepoURL != "" {
		trimmed := strings.TrimSuffix(audit.RepoURL, ".git")
		pathOnly := trimmed
		if idx := strings.Index(pathOnly, "://"); idx != -1 {
			pathOnly = pathOnly[idx+3:]
		}
		parts := strings.Split(strings.Trim(pathOnly, "/"), "/")
		if len(parts) >= 2 {
			titleName = fmt.Sprintf("[%s/%s](%s)", parts[len(parts)-2], parts[len(parts)-1], audit.RepoURL)
		} else {
			titleName = fmt.Sprintf("[%s](%s)", parts[0], audit.RepoURL)
		}
	}

	if isDE {
		fmt.Fprintf(w, "# Benchmark & Lücken-Audit: %s\n\n", titleName)
		fmt.Fprintln(w, "- [← Zurück zur Gesamt-Matrix](../matrix.md)")
		if audit.Language != "" || audit.Tier != "" {
			fmt.Fprintf(w, "- **Sprache:** %s | **Framework:** %s | **Tier:** %s\n", audit.Language, audit.Framework, audit.Tier)
		}
		if audit.CommitSHA != "" {
			fmt.Fprintf(w, "- **Commit:** `%s`\n", audit.CommitSHA)
		}
		if audit.SampleFile != "" {
			fmt.Fprintf(w, "- **Beispieldatei:** `%s`\n", audit.SampleFile)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "## 1. Performance & Latenzen")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |")
		fmt.Fprintln(w, "|---|---:|---:|---:|---:|---|")
	} else {
		fmt.Fprintf(w, "# Benchmark & Gap Audit: %s\n\n", titleName)
		fmt.Fprintln(w, "- [← Back to Matrix](../matrix.md)")
		if audit.Language != "" || audit.Tier != "" {
			fmt.Fprintf(w, "- **Language:** %s | **Framework:** %s | **Tier:** %s\n", audit.Language, audit.Framework, audit.Tier)
		}
		if audit.CommitSHA != "" {
			fmt.Fprintf(w, "- **Commit:** `%s`\n", audit.CommitSHA)
		}
		if audit.SampleFile != "" {
			fmt.Fprintf(w, "- **Sample file:** `%s`\n", audit.SampleFile)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "## 1. Performance & Latencies")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |")
		fmt.Fprintln(w, "|---|---:|---:|---:|---:|---|")
	}

	for _, comp := range audit.Components() {
		if comp.Applicable != nil && !*comp.Applicable {
			statusTxt := "n/a (non-Go)"
			if comp.Name != "graph build" {
				statusTxt = "n/a"
			}
			fmt.Fprintf(w, "| **%s** | n/a | n/a | n/a | n/a | %s |\n", comp.Name, statusTxt)
			continue
		}
		fmt.Fprintf(w, "| **%s** | %s | %s |\n", comp.Name, timingCells(comp), exitStatus(comp))
	}

	total, _ := audit.Timing(TotalTiming)
	totalLabel := "Total"
	if isDE {
		totalLabel = "Gesamt"
	}
	fmt.Fprintf(w, "| **%s** | %s | %s |\n", totalLabel, timingCells(total), exitStatus(allRuns(audit)...))

	if audit.ClaudeWarmMed > 0 {
		fmt.Fprintf(w, "\n- **Baseline Claude Hook:** %s%s (Speedup: %.1fx%s)\n", benchreport.FormatMS(audit.ClaudeWarmMed), hookComparison(audit), audit.Speedup, baselineStatus(audit))
	} else if audit.BaselineError != "" {
		fmt.Fprintf(w, "\n- **Baseline Claude Hook:** %s (%s)\n", unavailableWord(isDE), audit.BaselineError)
	}

	fmt.Fprintln(w)
	if isDE {
		fmt.Fprintln(w, "## 2. Test- & Lücken-Audit (Gap Analysis)")
		fmt.Fprintln(w)
		if len(audit.Audit) > 0 {
			fmt.Fprintln(w, "| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |")
			fmt.Fprintln(w, "|---|---|---|---|---|---|")
			for _, check := range audit.Audit {
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
		fmt.Fprintf(w, "- **Abdeckungsquote:** **%.1f %%**\n", audit.CoverageRate)
		if len(audit.MissingGaps) > 0 {
			fmt.Fprintln(w, "- **Identifizierte Lücken:**")
			for _, gap := range audit.MissingGaps {
				fmt.Fprintf(w, "  - ⚠️ %s\n", gap)
			}
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, "## 3. Erkannte Stacks & Lanes")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "- **Stacks:** `%s`\n", strings.Join(audit.DetectedStacks, ", "))
		fmt.Fprintf(w, "- **Lanes:** `%s`\n", strings.Join(audit.ExecutedLanes, ", "))
	} else {
		fmt.Fprintln(w, "## 2. Check & Gap Audit (Gap Analysis)")
		fmt.Fprintln(w)
		if len(audit.Audit) > 0 {
			fmt.Fprintln(w, "| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |")
			fmt.Fprintln(w, "|---|---|---|---|---|---|")
			for _, check := range audit.Audit {
				pathStr := "No"
				if check.OnPath {
					pathStr = "Yes"
				}
				laneStr := check.Lane
				if laneStr == "" {
					laneStr = "*none*"
				}
				statusStr := "✅ Active"
				if check.Lane == "" {
					statusStr = "⚠️ Missing in Loomux"
				} else if !check.OnPath {
					statusStr = "❌ Missing from PATH"
				}
				fmt.Fprintf(w, "| `%s` | %s | `%s` | `%s` | %s | %s |\n",
					check.Tool, check.Category, check.Native, laneStr, pathStr, statusStr)
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "- **Coverage Rate:** **%.1f %%**\n", audit.CoverageRate)
		if len(audit.MissingGaps) > 0 {
			fmt.Fprintln(w, "- **Identified Gaps:**")
			for _, gap := range audit.MissingGaps {
				fmt.Fprintf(w, "  - ⚠️ %s\n", gap)
			}
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, "## 3. Detected Stacks & Lanes")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "- **Stacks:** `%s`\n", strings.Join(audit.DetectedStacks, ", "))
		fmt.Fprintf(w, "- **Lanes:** `%s`\n", strings.Join(audit.ExecutedLanes, ", "))
	}

	return nil
}

// timingCells are the cold, warm median, minimum and maximum cells of a row.
func timingCells(t benchreport.Timing) string {
	return strings.Join([]string{
		benchreport.FormatMS(t.ColdMS), benchreport.FormatMS(t.MedianMS),
		benchreport.FormatMS(t.MinMS), benchreport.FormatMS(t.MaxMS),
	}, " | ")
}
