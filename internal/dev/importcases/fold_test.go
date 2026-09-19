package importcases

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// decodeTOML reads a document the way translateDir reads the old config.
func decodeTOML(t *testing.T, doc string) map[string]any {
	t.Helper()
	decoded := map[string]any{}
	if _, err := toml.Decode(doc, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

// A world like mixed-config: the old configuration replaced the preset of
// every kind it named, so every stack the world holds loses that kind to
// false, and the commands move to the project.
func TestFoldVerifyMovesTheKindsToTheProjectAndSwitchesThePresetsOff(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"world\"\n")
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/world\n")
	old := decodeTOML(t, `
types = "uv run mypy ."
test = ["uv run pytest", "go test ./..."]
max_parallel = 2
timeout = 30
godot_import = false
tests = ["tests/**"]
gofmt = true

[lint]
commands = ["uvx ruff check .", "go vet ./..."]
threaded = true

[coverage]
report = "uv run coverage report"
threshold = 90

[after]
test = "lint"
coverage = "test"

[profiles]
quick = ["lint"]
`)
	got, err := foldVerify(dir, old)
	if err != nil {
		t.Fatal(err)
	}
	off := map[string]any{"lint": false, "types": false, "test": false, "coverage": false}
	want := map[string]any{
		"max_parallel": int64(2),
		"timeout":      int64(30),
		"profiles":     map[string]any{"quick": []any{"lint"}},
		"project": map[string]any{
			"lint":     map[string]any{"commands": []any{"uvx ruff check .", "go vet ./..."}, "threaded": true},
			"types":    "uv run mypy .",
			"test":     map[string]any{"commands": []any{"uv run pytest", "go test ./..."}, "after": "lint"},
			"coverage": map[string]any{"commands": []any{"uv run coverage report"}, "after": "test"},
		},
		"gdscript": map[string]any{"import_check": false},
		"go":       off,
		"python":   off,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %#v\nwant %#v", got, want)
	}
}

// [verify.after] alone replaced no preset: the kind it names is only
// ordered, and a table the project already carries keeps its keys.
func TestFoldVerifyOrdersWithoutReplacing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"world\"\n")
	old := decodeTOML(t, "[lint]\ncommands = [\"ruff\"]\n\n[coverage]\nthreshold = 90\n\n[after]\nlint = \"types\"\ntest = \"coverage\"\n")
	got, err := foldVerify(dir, old)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"project": map[string]any{
			"lint": map[string]any{"commands": []any{"ruff"}, "after": "types"},
			"test": map[string]any{"after": "coverage"},
		},
		"python": map[string]any{"lint": false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %#v\nwant %#v", got, want)
	}
}

// A shape the old chain refused is carried as it stands, so the new schema
// refuses it too; a wiki holds no lanes to switch off, and a signal that is
// not a stack has none either.
func TestFoldVerifyCarriesWhatItCannotReadAndSkipsTheWiki(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "wiki", "index.md"), "---\nokf_version: 1\n---\n")
	writeFile(t, filepath.Join(dir, "project.godot"), "config_version=5\n")
	old := decodeTOML(t, "lint = 3\n\n[after]\nlint = \"test\"\n")
	got, err := foldVerify(dir, old)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"project":  map[string]any{"lint": int64(3)},
		"gdscript": map[string]any{"lint": false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %#v\nwant %#v", got, want)
	}
}

// Nothing the old chain read leaves nothing to write.
func TestFoldVerifyOfNothingIsEmpty(t *testing.T) {
	got, err := foldVerify(t.TempDir(), decodeTOML(t, "gofmt = true\n"))
	if err != nil || len(got) != 0 {
		t.Fatalf("%v %#v", err, got)
	}
}

func TestFoldVerifyRefusesAnAfterThatIsNoTable(t *testing.T) {
	_, err := foldVerify(t.TempDir(), decodeTOML(t, "after = \"test\"\n"))
	if err == nil || !strings.Contains(err.Error(), "[verify.after] must be a table") {
		t.Fatalf("%v", err)
	}
}

// The fold is part of the translation: the old file goes, the new [verify]
// stands in the one config, and a [verify] that is no table stops the import.
func TestTranslateWorldFoldsTheOldVerify(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".ultraloom", "config.toml"), "[verify]\nlint = []\n")
	if err := TranslateWorld(dir); err != nil {
		t.Fatal(err)
	}
	got := decodeConfig(t, dir)
	want := map[string]any{"verify": map[string]any{"project": map[string]any{"lint": []any{}}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%#v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, ".ultraloom")); !os.IsNotExist(err) {
		t.Fatalf("the old directory survived: %v", err)
	}

	broken := t.TempDir()
	writeFile(t, filepath.Join(broken, ".ultraloom", "config.toml"), "verify = 1\n")
	if err := TranslateWorld(broken); err == nil || !strings.Contains(err.Error(), "[verify] must be a table") {
		t.Fatalf("%v", err)
	}
	writeFile(t, filepath.Join(broken, ".ultraloom", "config.toml"), "[verify]\nafter = 1\n")
	if err := TranslateWorld(broken); err == nil || !strings.Contains(err.Error(), "[verify.after] must be a table") {
		t.Fatalf("%v", err)
	}
}
