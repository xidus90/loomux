package importcases

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/dev/faketool"
)

const extraAnswers = `{"answers": [{"prefix": "npx eslint", "exit": 0, "stdout": "faketool: npx eslint\n"}]}`

// The recorded answers stay first and keep their order; the extra ones follow
// them, and a world that had no fixture gets one.
func TestMergeFixtureAppendsToEveryWorld(t *testing.T) {
	to := t.TempDir()
	withFixture := buildCase(t, to, "check", "node-lint", "loomux check lint", "")
	writeFile(t, filepath.Join(withFixture, "world", faketool.FixtureName), `{"answers": [{"prefix": "eslint", "exit": 1}]}`)
	without := buildCase(t, to, "check", "config-empty-list-all", "loomux check all", "")
	extra := filepath.Join(t.TempDir(), "extra.json")
	writeFile(t, extra, extraAnswers)

	if err := MergeFixture(to, extra); err != nil {
		t.Fatal(err)
	}

	eslint := faketool.Answer{Prefix: "npx eslint", Stdout: "faketool: npx eslint\n"}
	for dir, want := range map[string][]faketool.Answer{
		withFixture: {{Prefix: "eslint", Exit: 1}, eslint},
		without:     {eslint},
	} {
		got, err := faketool.Load(filepath.Join(dir, "world", faketool.FixtureName))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Answers, want) {
			t.Errorf("%s: %+v", dir, got.Answers)
		}
	}
}

func TestMergeFixtureReportsWhatItCannotRead(t *testing.T) {
	to := t.TempDir()
	world := filepath.Join(buildCase(t, to, "check", "one", "loomux check all", ""), "world")
	extra := filepath.Join(t.TempDir(), "extra.json")

	for name, setup := range map[string]func() string{
		"a missing fixture": func() string { return filepath.Join(t.TempDir(), "gone.json") },
		"a broken fixture": func() string {
			writeFile(t, extra, "{")
			return extra
		},
	} {
		if err := MergeFixture(to, setup()); err == nil || !strings.Contains(err.Error(), "extra answers") {
			t.Errorf("%s: %v", name, err)
		}
	}

	writeFile(t, extra, extraAnswers)
	if err := MergeFixture(filepath.Join(t.TempDir(), "gone"), extra); err == nil {
		t.Error("want an error for a corpus that is not there")
	}
	writeFile(t, filepath.Join(world, faketool.FixtureName), "{")
	if err := MergeFixture(to, extra); err == nil || !strings.Contains(err.Error(), faketool.FixtureName) {
		t.Errorf("want an error naming the world's fixture, got %v", err)
	}
	if err := os.Remove(filepath.Join(world, faketool.FixtureName)); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(world, faketool.FixtureName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := MergeFixture(to, extra); err == nil {
		t.Error("want an error for a fixture that is a directory")
	}
}
