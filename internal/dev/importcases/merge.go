package importcases

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/dev/faketool"
)

// extraFile is the file MergeFixture reads: answers every world gets, and
// Same, which gives a command line the answer the world recorded for another
// one. A world that failed `uv run pytest` fails `uv run --with pytest pytest`
// the same way; a fixed answer could not say so.
type extraFile struct {
	Answers []faketool.Answer `json:"answers"`
	Same    []struct {
		Prefix string `json:"prefix"`
		As     string `json:"as"`
	} `json:"same"`
}

// MergeFixture appends the answers in extra to the fixture of every world in
// the corpus at to. loomux's presets call some tools with other command lines
// than the old chain did; the recorded fixture is evidence and stays as it
// was, so what only loomux asks is answered from one file beside it. A world
// without a fixture gets one.
func MergeFixture(to, extra string) error {
	data, err := os.ReadFile(extra)
	if err != nil {
		return fmt.Errorf("reading the extra answers: %w", err)
	}
	var add extraFile
	if err := json.Unmarshal(data, &add); err != nil {
		return fmt.Errorf("reading the extra answers in %s: %w", extra, err)
	}
	found, err := cases.DiscoverCases(to, "")
	if err != nil {
		return err
	}
	for _, c := range found {
		if err := mergeInto(filepath.Join(c.Path, "world", faketool.FixtureName), add); err != nil {
			return err
		}
	}
	return nil
}

// mergeInto puts the recorded answers first, so a reader of the file sees the
// recording before what was added to it: then the extra answers, then the
// copies Same asks for, each in the order of the extra file.
func mergeInto(path string, add extraFile) error {
	fixture, err := faketool.Load(path)
	if err != nil {
		return err
	}
	recorded := fixture.Answers
	fixture.Answers = append(fixture.Answers, add.Answers...)
	for _, same := range add.Same {
		// Only a recording counts, not an answer the extra file adds under
		// the same prefix. Of two recordings the later one is copied, once:
		// on a tie that is the one the fixture gives.
		found := -1
		for i, a := range recorded {
			if a.Prefix == same.As {
				found = i
			}
		}
		if found >= 0 {
			a := recorded[found]
			a.Prefix = same.Prefix
			fixture.Answers = append(fixture.Answers, a)
		}
	}
	// A fixture of plain strings and numbers always encodes.
	encoded, _ := json.MarshalIndent(fixture, "", "  ")
	return os.WriteFile(path, append(encoded, '\n'), 0o644)
}
