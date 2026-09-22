package cases

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/sessions"
)

// compareState pins what the stop gate decided: for every session file the
// recording left, the base it measures from and the blocks it counted. The
// green tree and every other file are loomux's own and not compared.
func compareState(expectedRoot, actualRoot string) []string {
	var out []string
	for _, id := range sessionIDs(expectedRoot) {
		want, got := sessions.ReadState(expectedRoot, id), sessions.ReadState(actualRoot, id)
		if want.Base != got.Base {
			out = append(out, fmt.Sprintf("session %s: base expected %q, got %q", id, want.Base, got.Base))
		}
		if want.Blocks != got.Blocks {
			out = append(out, fmt.Sprintf("session %s: blocks expected %d, got %d", id, want.Blocks, got.Blocks))
		}
	}
	return out
}

func sessionIDs(root string) []string {
	entries, _ := os.ReadDir(filepath.Join(root, filepath.FromSlash(sessions.StateDir)))
	var ids []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			ids = append(ids, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return ids
}

// compareFindings pins what subagent-stop found: the lines the Python hook
// printed against the finding lines of every agent file, each with the
// prefix Python put in front, both sides sorted.
func compareFindings(expected []byte, actualRoot string) []string {
	want := lines(string(expected))
	var got []string
	paths, _ := filepath.Glob(filepath.Join(actualRoot, filepath.FromSlash(sessions.StateDir), "*", "agents", "*.json"))
	for _, path := range paths {
		session := filepath.Base(filepath.Dir(filepath.Dir(path)))
		agent := strings.TrimSuffix(filepath.Base(path), ".json")
		f, ok := sessions.ReadAgent(actualRoot, session, agent)
		if !ok {
			continue
		}
		for _, line := range f.Finding {
			got = append(got, "subagent "+agent+": "+line)
		}
	}
	slices.Sort(want)
	slices.Sort(got)
	if slices.Equal(want, got) {
		return nil
	}
	return []string{fmt.Sprintf("findings expected:\n%s\ngot:\n%s", strings.Join(want, "\n"), strings.Join(got, "\n"))}
}

func lines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
