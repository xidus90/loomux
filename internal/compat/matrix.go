package compat

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/gitenv"
)

// Job is one cell of the CI matrix: an entry on an operating system.
type Job struct {
	Entry string `json:"entry"`
	OS    string `json:"os"`
	// Probe is the JSON of an entry that is not in the list; empty for the
	// entries of the list.
	Probe string `json:"probe,omitempty"`
}

// ProbeMatrix is the probes, each on linux then windows. The entry key may
// equal one of the list (an addon probe keeps the runtime version), so the
// job carries the entry itself.
func ProbeMatrix(probes []Entry) []Job {
	jobs := []Job{}
	for _, e := range probes {
		data, _ := json.Marshal(e) // plain strings always encode
		jobs = append(jobs, Job{e.Key(), "linux", string(data)}, Job{e.Key(), "windows", string(data)})
	}
	return jobs
}

// Selection says which entries a run needs. Full selects all; otherwise an
// entry is selected when its key is in Keys or ChangedEntries, or its fixture
// is in ChangedFixtures.
type Selection struct {
	Full            bool
	Keys            []string
	ChangedFixtures []string
	ChangedEntries  []string
}

// Matrix is the selected entries, in list order, each on linux then windows.
// It is never nil, so it encodes as an empty array.
func Matrix(entries []Entry, sel Selection) []Job {
	jobs := []Job{}
	for _, e := range entries {
		k := e.Key()
		if sel.Full || slices.Contains(sel.Keys, k) || slices.Contains(sel.ChangedEntries, k) ||
			slices.Contains(sel.ChangedFixtures, e.Fixture) {
			jobs = append(jobs, Job{Entry: k, OS: "linux"}, Job{Entry: k, OS: "windows"})
		}
	}
	return jobs
}

const (
	listPath    = "internal/compat/compat.toml"
	fixturesDir = "internal/compat/testdata/"
)

func git(repo string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	cmd.Env = gitenv.Environ()
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

// ChangedSince names the fixtures touched between ref and HEAD (the commit CI
// checks out) and the entries that are new, differ from the list at ref, or
// are selected by a [[watch]] rule for a file changed since.
// A ref without a list makes every entry new.
func ChangedSince(repo, ref string) (fixtures, entries []string, err error) {
	if _, err = git(repo, "rev-parse", "--verify", ref+"^{commit}"); err != nil {
		return nil, nil, err
	}
	names, err := git(repo, "diff", "--name-only", ref, "HEAD")
	if err != nil {
		return nil, nil, err
	}
	changed := strings.Fields(string(names))
	for _, n := range changed {
		if rest, ok := strings.CutPrefix(n, fixturesDir); ok {
			if dir, _, _ := strings.Cut(rest, "/"); !slices.Contains(fixtures, dir) {
				fixtures = append(fixtures, dir)
			}
		}
	}
	cur, _ := Load() // the embedded list is validated by this package's tests
	var old []Entry
	if data, gerr := git(repo, "show", ref+":"+listPath); gerr == nil {
		if old, err = decode(data, ref+":"+listPath, false); err != nil {
			return nil, nil, err
		}
	}
	before := map[string]Entry{}
	for _, e := range old {
		e.Line = 0
		before[e.Key()] = e
	}
	for _, e := range cur {
		k := e.Key()
		e.Line = 0
		if !reflect.DeepEqual(before[k], e) {
			entries = append(entries, k)
		}
	}
	watched, _ := WatchedEntries(cur, changed) // the globs are checked at load
	for _, k := range watched {
		if !slices.Contains(entries, k) {
			entries = append(entries, k)
		}
	}
	return fixtures, entries, nil
}
