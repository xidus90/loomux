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

// A Same entry copies the world's recorded answer to another command line, so
// a red world stays red; a world that never recorded it gets nothing.
func TestMergeFixtureGivesACommandTheAnswerOfAnother(t *testing.T) {
	to := t.TempDir()
	red := buildCase(t, to, "check", "red", "loomux check test", "")
	writeFile(t, filepath.Join(red, "world", faketool.FixtureName), `{"answers": [{"prefix": "uv run pytest", "exit": 1, "stdout": "failed\n"}]}`)
	none := buildCase(t, to, "check", "none", "loomux check test", "")
	extra := filepath.Join(t.TempDir(), "extra.json")
	writeFile(t, extra, `{"answers": [], "same": [{"prefix": "uv run --with pytest pytest", "as": "uv run pytest"}]}`)

	if err := MergeFixture(to, extra); err != nil {
		t.Fatal(err)
	}

	for dir, want := range map[string][]faketool.Answer{
		red:  {{Prefix: "uv run pytest", Exit: 1, Stdout: "failed\n"}, {Prefix: "uv run --with pytest pytest", Exit: 1, Stdout: "failed\n"}},
		none: nil,
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

// Same copies what the world recorded, not what the extra file adds: an extra
// answer under the `as` prefix is no recording. A world that recorded the
// prefix twice gets one copy, of the later answer, since that is the one the
// fixture gives on a tie.
func TestMergeFixtureCopiesTheOneRecordedAnswerTheFixtureWouldGive(t *testing.T) {
	to := t.TempDir()
	unrecorded := buildCase(t, to, "check", "unrecorded", "loomux check test", "")
	writeFile(t, filepath.Join(unrecorded, "world", faketool.FixtureName), `{"answers": [{"prefix": "ruff", "exit": 0}]}`)
	twice := buildCase(t, to, "check", "twice", "loomux check test", "")
	writeFile(t, filepath.Join(twice, "world", faketool.FixtureName),
		`{"answers": [{"prefix": "uv run pytest", "exit": 0, "stdout": "first\n"}, {"prefix": "uv run pytest", "exit": 1, "stdout": "second\n"}, {"prefix": "ruff", "exit": 0}]}`)
	extra := filepath.Join(t.TempDir(), "extra.json")
	writeFile(t, extra, `{"answers": [{"prefix": "uv run pytest", "exit": 0, "stdout": "extra\n"}],
		"same": [{"prefix": "uv run --with pytest pytest", "as": "uv run pytest"}]}`)

	if err := MergeFixture(to, extra); err != nil {
		t.Fatal(err)
	}

	added := faketool.Answer{Prefix: "uv run pytest", Stdout: "extra\n"}
	for dir, want := range map[string][]faketool.Answer{
		unrecorded: {{Prefix: "ruff"}, added},
		twice: {{Prefix: "uv run pytest", Stdout: "first\n"}, {Prefix: "uv run pytest", Exit: 1, Stdout: "second\n"}, {Prefix: "ruff"}, added,
			{Prefix: "uv run --with pytest pytest", Exit: 1, Stdout: "second\n"}},
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
