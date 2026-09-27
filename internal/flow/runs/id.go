package runs

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Dir is where runs live, relative to the project root.
const Dir = ".loomux/state/runs"

// NextID is one more than the highest number a journal or a marker carries,
// four digits. A counter and never a clock.
//
// Markers count because Claim takes a number with the marker before the
// journal exists, and a count over journals alone would hand the same number
// out again.
//
// A directory that cannot be read counts as one without runs. On Windows even
// a file in its place reads as absent (measured on 2026-09-11), and a write
// into it fails loudly in WriteMarker.
func NextID(root string) string {
	entries, _ := os.ReadDir(filepath.Join(root, Dir))
	highest := 0
	for _, entry := range entries {
		stem, isRunFile := strings.CutSuffix(entry.Name(), ".jsonl")
		if !isRunFile {
			stem, isRunFile = strings.CutSuffix(entry.Name(), ".flow")
		}
		if !isRunFile || !digits(stem) {
			continue
		}
		number, err := strconv.Atoi(stem)
		if err != nil {
			continue
		}
		highest = max(highest, number)
	}
	return fmt.Sprintf("%04d", highest+1)
}

func digits(text string) bool {
	if text == "" {
		return false
	}
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return false
		}
	}
	return true
}

// JournalPath is where run id writes its journal.
func JournalPath(root, id string) string {
	return filepath.Join(root, Dir, id+".jsonl")
}

// MarkerPath is where run id writes its marker.
func MarkerPath(root, id string) string {
	return filepath.Join(root, Dir, id+".flow")
}

// Files are the two files a run writes itself, spelled as
// changed-file lists spell paths, so a guard can subtract exactly these two and
// not every other run's files along with them.
func Files(id string) []string {
	return []string{Dir + "/" + id + ".flow", Dir + "/" + id + ".jsonl"}
}
