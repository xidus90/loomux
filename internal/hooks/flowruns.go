package hooks

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/flows"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/flow/journal"
	"github.com/xidus90/loomux/internal/flow/runs"
)

// flowsDir is where a project keeps its own flows. internal/flow/load names
// it too; the hooks may not link the loader for one constant.
const flowsDir = ".loomux/flows"

// waitingRuns announces every run of the project that waits at a gate, one
// context line each, by run number. The answer is a human's -- the guard
// refuses it to an agent -- so the line says who answers and names the binary
// this hook runs from: loomux is not on every PATH, and a project runs its
// hooks from bin/ or from the machine-wide install.
//
// A journal that will not read is named on stderr and the other runs are still
// announced. When its marker says another loomux wrote the run, the line names
// both versions instead: a newer binary's journal is not damaged, only newer.
// A marker that will not read, or none at all, is named the same way and hides
// only its own run, since resume refuses such a run and there is no command
// to offer.
func waitingRuns(root, version string, stderr io.Writer) []string {
	// An absent runs directory means no runs. One that cannot be listed is
	// named and announces nothing: this hook has no verdict to give on a
	// project it cannot read.
	entries, err := listed(filepath.Join(root, filepath.FromSlash(runs.Dir)))
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %s cannot be read as a folder of runs: %v\n", runs.Dir, err)
	}
	binary := "loomux"
	if path, err := executable(); err == nil {
		binary = shellWord(filepath.ToSlash(path))
	}
	var lines []string
	// ReadDir sorts by name, so two machines announce the same runs in the
	// same order.
	for _, entry := range entries {
		id, isJournal := strings.CutSuffix(entry.Name(), ".jsonl")
		if entry.IsDir() || !isJournal {
			continue
		}
		gate, err := journal.Pending(runs.JournalPath(root, id))
		if err == nil && gate == nil {
			continue
		}
		marker, markerErr := runs.ReadMarker(runs.MarkerPath(root, id))
		switch {
		case err != nil && marker != nil && marker.Version != "" && marker.Version != version:
			fmt.Fprintf(stderr, "loomux hook session-start: run %s was written by loomux %s, this is %s\n", id, marker.Version, version)
		case err != nil:
			fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		case markerErr != nil:
			fmt.Fprintf(stderr, "loomux hook session-start: %v\n", markerErr)
		case marker == nil:
			fmt.Fprintf(stderr, "loomux hook session-start: run %q does not say which flow it belongs to\n", id)
		default:
			// A question read from a file ends in a newline of its own, which
			// would leave an empty line before the command.
			lines = append(lines, fmt.Sprintf(
				"run %s (%s) is waiting at %s: %s\n  a human answers it with: %s flow resume %s --answer \"your answer\"",
				id, origin(marker), gate.Node, strings.TrimRight(gate.Question, "\n"), binary, id))
		}
	}
	return lines
}

// shellWord is path as a human pastes it into a shell: bare when no shell
// would split or read it, else in double quotes -- a user name with a blank
// would otherwise cut the command in two.
func shellWord(path string) string {
	if strings.ContainsAny(path, " \t'\"`$&|;<>()!*?[]{}#~%^") {
		return `"` + path + `"`
	}
	return path
}

// listed is dir's entries, none when it is absent. One that cannot be listed
// -- a file, or a folder the process may not read -- is an error. Stat tells a
// file from an absent folder, as load's underDir does: Windows lists a file as
// a path it cannot find, POSIX as no directory.
func listed(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	info, statErr := os.Stat(dir)
	switch {
	case errors.Is(statErr, fs.ErrNotExist):
		return nil, nil
	case statErr == nil && !info.IsDir():
		err = errors.New("it is a file")
	}
	return entries, err
}

// origin is a run's flow and where it came from, the origin spelled as run,
// show and list print it.
func origin(marker *runs.Marker) string {
	if len(marker.Overlays) == 0 {
		return marker.Flow + ", " + marker.Origin
	}
	return marker.Flow + ", " + marker.Origin + ": " + strings.Join(marker.Overlays, ", ")
}

// ignoredFlowFolders names what the project holds under a bundled flow's name
// while [flow] overrides does not let it hide that flow: the bundled flow runs
// and the project's files do not, which flow run says too, but only once
// somebody runs it. The same entries count as for the loader, spelled exactly
// as the directory lists them, folder or file.
//
// [flow] is read only when such an entry exists. A project without flows pays
// no second decode of its config, and a config that will not read is said by
// whoever reads it for a reason. A flows folder that cannot be listed is said
// in load.Find's words, since a bundled flow it would hide runs in its place.
func ignoredFlowFolders(root string) []string {
	entries, err := listed(filepath.Join(root, filepath.FromSlash(flowsDir)))
	if err != nil {
		return []string{flowsDir + " cannot be read as a folder of flows: " + err.Error()}
	}
	bundled := flows.Names()
	var named []string
	for _, entry := range entries {
		if slices.Contains(bundled, entry.Name()) {
			named = append(named, entry.Name())
		}
	}
	if len(named) == 0 {
		return nil
	}
	settings, err := config.ReadFlowSettings(root)
	if err != nil {
		return []string{"loomux: [flow] cannot be read: " + err.Error()}
	}
	var lines []string
	for _, name := range named {
		if !settings.Allows(name) {
			lines = append(lines, flowsDir+"/"+name+" is ignored: [flow] overrides does not name it")
		}
	}
	return lines
}
