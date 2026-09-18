package benchcorpus

import (
	"fmt"
	"strings"
)

// AuditGaps matches native project tools against detected stacks and executed Loomux check lanes.
func AuditGaps(project ProjectSignals, detectedStacks []string, executedLanes []string, lookPath func(string) (string, error)) ([]CheckAudit, []string, float64) {
	if len(project.NativeTools) == 0 {
		return nil, nil, 100.0
	}

	var audits []CheckAudit
	var gaps []string
	matchedLanes := 0

	for _, tool := range project.NativeTools {
		execName := toolExecutable(tool.Tool)
		_, err := lookPath(execName)
		onPath := err == nil

		matchedLane := findMatchingLane(tool.Tool, executedLanes)
		if matchedLane != "" {
			matchedLanes++
		}

		audits = append(audits, CheckAudit{
			Tool:     tool.Tool,
			Category: tool.Category,
			Native:   tool.ConfigFile,
			Lane:     matchedLane,
			OnPath:   onPath,
		})

		if matchedLane == "" {
			gaps = append(gaps, fmt.Sprintf("%s deklariert (%s), aber keine Lane in Loomux vorhanden", tool.Tool, tool.ConfigFile))
		} else if !onPath {
			gaps = append(gaps, fmt.Sprintf("Lane für %s vorhanden (%s), aber Werkzeug nicht im PATH", tool.Tool, matchedLane))
		}
	}

	coverageRate := float64(matchedLanes) / float64(len(project.NativeTools)) * 100.0
	return audits, gaps, coverageRate
}

func toolExecutable(tool string) string {
	switch tool {
	case "cargo-test":
		return "cargo"
	case "dotnet-test":
		return "dotnet"
	case "mvn-test":
		return "mvn"
	case "go-test", "go-vet", "gofmt":
		return "go"
	default:
		return tool
	}
}

func findMatchingLane(tool string, lanes []string) string {
	target := tool
	switch tool {
	case "go-vet":
		target = "go vet"
	case "go-test":
		target = "go test"
	case "cargo-test":
		target = "cargo test"
	case "dotnet-test":
		target = "dotnet test"
	case "mvn-test":
		target = "mvn test"
	}

	for _, lane := range lanes {
		trimmed := strings.TrimSpace(lane)
		if containsCommand(trimmed, target) {
			return trimmed
		}
	}
	return ""
}

// containsCommand reports whether target appears in line as a command of its
// own, so that "cargo test" does not pass for "go test".
func containsCommand(line, target string) bool {
	for start := 0; ; {
		idx := strings.Index(line[start:], target)
		if idx < 0 {
			return false
		}
		idx += start
		end := idx + len(target)
		before := idx == 0 || strings.ContainsRune(" \t\n;&|(", rune(line[idx-1]))
		after := end == len(line) || strings.ContainsRune(" \t\n", rune(line[end]))
		if before && after {
			return true
		}
		start = idx + 1
	}
}
