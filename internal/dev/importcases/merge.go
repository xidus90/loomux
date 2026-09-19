package importcases

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/dev/faketool"
)

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
	var add faketool.Fixture
	if err := json.Unmarshal(data, &add); err != nil {
		return fmt.Errorf("reading the extra answers in %s: %w", extra, err)
	}
	found, err := cases.DiscoverCases(to, "")
	if err != nil {
		return err
	}
	for _, c := range found {
		if err := mergeInto(filepath.Join(c.Path, "world", faketool.FixtureName), add.Answers); err != nil {
			return err
		}
	}
	return nil
}

// mergeInto puts the recorded answers first, so a reader of the file sees the
// recording before what was added to it.
func mergeInto(path string, answers []faketool.Answer) error {
	fixture, err := faketool.Load(path)
	if err != nil {
		return err
	}
	fixture.Answers = append(fixture.Answers, answers...)
	// A fixture of plain strings and numbers always encodes.
	encoded, _ := json.MarshalIndent(fixture, "", "  ")
	return os.WriteFile(path, append(encoded, '\n'), 0o644)
}
