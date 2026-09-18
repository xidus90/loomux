package benchcorpus

import (
	"io/fs"
	"path"
	"strings"
)

// gateLanes reads .githooks/pre-commit and returns trimmed non-empty, non-comment lines.
// A hook that hands its work to a script (`sh ci/gate.sh`) keeps its commands
// there, so each script the hook runs is read in after the line that runs it —
// one level deep, which is where a gate puts its commands.
func gateLanes(fsys fs.FS) []string {
	lines := scriptLines(fsys, ".githooks/pre-commit")
	var lanes []string
	for _, line := range lines {
		lanes = append(lanes, line)
		if script, ok := sourcedScript(line); ok {
			lanes = append(lanes, scriptLines(fsys, script)...)
		}
	}
	return lanes
}

// scriptLines returns the trimmed non-empty, non-comment lines of one file, or
// nil when it cannot be read.
func scriptLines(fsys fs.FS, name string) []string {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines
}

// sourcedScript reports the repository-relative script a `sh <path>` or
// `. <path>` line runs.
func sourcedScript(line string) (string, bool) {
	fields := strings.Fields(line)
	if len(fields) != 2 || (fields[0] != "sh" && fields[0] != ".") {
		return "", false
	}
	return path.Clean(fields[1]), true
}
