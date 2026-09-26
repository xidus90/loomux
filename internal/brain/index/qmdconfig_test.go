package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const testExistingYAML = `collections:
  space:
    path: C:/corpus
    pattern: "**/*.md"
    ignore:
      - "**/.git/**"
  foreign:
    path: C:/elsewhere
    pattern: "**/*.md"
    ignore: []
models:
  embed: hf:some/model.gguf
`

func writeTestYAML(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "index.yml")
	if err := os.WriteFile(p, []byte(testExistingYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func writeTestOwned(t *testing.T, dir string, names []string) string {
	t.Helper()
	p := filepath.Join(dir, "owned.json")
	data, err := json.Marshal(names)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestQmdConfigPath(t *testing.T) {
	t.Run("follows XDG_CONFIG_HOME", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmp)
		got := QmdConfigPath()
		want := filepath.Join(tmp, "qmd", "index.yml")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("falls back to user home directory", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("no home directory available")
		}
		got := QmdConfigPath()
		want := filepath.Join(home, ".config", "qmd", "index.yml")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestModelsBlockSurvives(t *testing.T) {
	tmp := t.TempDir()
	config := writeTestYAML(t, tmp)
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: []string{"**/.git/**"}},
	}
	if _, err := SyncCollections(config, wanted, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	models, ok := doc["models"].(map[string]any)
	if !ok || models["embed"] != "hf:some/model.gguf" {
		t.Errorf("expected models block preserved, got %#v", doc["models"])
	}
}

func TestCollectionWeDoNotManageSurvives(t *testing.T) {
	tmp := t.TempDir()
	config := writeTestYAML(t, tmp)
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: nil},
	}
	if _, err := SyncCollections(config, wanted, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	cols := doc["collections"].(map[string]any)
	if _, found := cols["foreign"]; !found {
		t.Error("expected foreign collection to survive")
	}
}

func TestIgnoreListIsTakenOver(t *testing.T) {
	tmp := t.TempDir()
	config := writeTestYAML(t, tmp)
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: []string{"**/.git/**", "**/node_modules/**"}},
	}
	outcome, err := SyncCollections(config, wanted, sameRecord(owned))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcome.Changed) != 1 || outcome.Changed[0] != "space" {
		t.Errorf("expected space in Changed, got %#v", outcome.Changed)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	cols := doc["collections"].(map[string]any)
	space := cols["space"].(map[string]any)
	ignores := space["ignore"].([]any)
	if len(ignores) != 2 || ignores[0] != "**/.git/**" || ignores[1] != "**/node_modules/**" {
		t.Errorf("unexpected ignores: %#v", ignores)
	}
}

func TestUnchangedConfigIsNotRewritten(t *testing.T) {
	tmp := t.TempDir()
	config := writeTestYAML(t, tmp)
	owned := writeTestOwned(t, tmp, []string{"space"})

	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: []string{"**/.git/**"}},
	}
	outcome, err := SyncCollections(config, wanted, sameRecord(owned))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcome.Changed) != 0 {
		t.Errorf("expected no changes, got %#v", outcome.Changed)
	}

	after, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("expected file byte-for-byte unchanged on no-op sync")
	}
}

func TestBackupIsKeptBeforeFirstWrite(t *testing.T) {
	tmp := t.TempDir()
	config := writeTestYAML(t, tmp)
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: []string{"**/.git/**", "new"}},
	}
	if _, err := SyncCollections(config, wanted, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}

	backup := config + ".brain-backup"
	backupData, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("expected backup file %s: %v", backup, err)
	}
	if string(backupData) != testExistingYAML {
		t.Errorf("backup content mismatch: got %q", string(backupData))
	}
}

func TestBackupKeepsOriginalAcrossLaterWrites(t *testing.T) {
	tmp := t.TempDir()
	config := writeTestYAML(t, tmp)
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted1 := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: []string{"first"}},
	}
	if _, err := SyncCollections(config, wanted1, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}

	wanted2 := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: []string{"second"}},
		"later": {Path: "C:/later", Pattern: "**/*.md", Ignore: nil},
	}
	if _, err := SyncCollections(config, wanted2, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}

	backup := config + ".brain-backup"
	backupData, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if string(backupData) != testExistingYAML {
		t.Errorf("backup must preserve initial original, got %q", string(backupData))
	}
}

func TestMissingConfigIsCreatedWithOnlyOurCollections(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "subdir", "index.yml")
	owned := filepath.Join(tmp, "owned.json")

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: nil},
	}
	outcome, err := SyncCollections(config, wanted, sameRecord(owned))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcome.Changed) != 1 || outcome.Changed[0] != "space" {
		t.Errorf("expected space changed, got %#v", outcome.Changed)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	cols := doc["collections"].(map[string]any)
	if len(cols) != 1 || cols["space"] == nil {
		t.Errorf("unexpected collections: %#v", cols)
	}
}

func TestConfigNotAMappingIsReplaced(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	if err := os.WriteFile(config, []byte("- just a list\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: nil},
	}
	outcome, err := SyncCollections(config, wanted, sameRecord(owned))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcome.Changed) != 1 || outcome.Changed[0] != "space" {
		t.Errorf("expected space changed, got %#v", outcome.Changed)
	}
}

func TestBlankConfigIsFilledIn(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	if err := os.WriteFile(config, []byte("\n  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: nil},
	}
	outcome, err := SyncCollections(config, wanted, sameRecord(owned))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcome.Changed) != 1 || outcome.Changed[0] != "space" {
		t.Errorf("expected space changed, got %#v", outcome.Changed)
	}
}

func TestCollectionsBlockOfWrongShapeIsReplaced(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	if err := os.WriteFile(config, []byte("collections: nonsense\nmodels: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: nil},
	}
	if _, err := SyncCollections(config, wanted, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["collections"].(map[string]any); !ok {
		t.Errorf("expected collections to be a mapping, got %#v", doc["collections"])
	}
}

func TestBrokenYAMLIsReportedAndFileUntouched(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	content := "collections: [unclosed\n"
	if err := os.WriteFile(config, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: nil},
	}
	_, err := SyncCollections(config, wanted, sameRecord(owned))
	if err == nil {
		t.Fatal("expected error on broken YAML")
	}
	if !strings.Contains(err.Error(), "not valid YAML") {
		t.Errorf("unexpected error: %v", err)
	}

	current, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if string(current) != content {
		t.Error("expected file untouched on broken YAML")
	}
	if _, err := os.Stat(config + ".brain-backup"); !os.IsNotExist(err) {
		t.Error("backup should not be created on error")
	}
}

func TestKeyWeDoNotManageSurvivesInsideOurOwnEntry(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	content := `collections:
  space:
    path: C:/corpus
    pattern: "**/*.md"
    ignore: []
    embed_model: hf:some/model.gguf
`
	if err := os.WriteFile(config, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: []string{"a"}},
	}
	if _, err := SyncCollections(config, wanted, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	space := doc["collections"].(map[string]any)["space"].(map[string]any)
	if space["embed_model"] != "hf:some/model.gguf" {
		t.Errorf("expected embed_model to survive, got %#v", space["embed_model"])
	}
}

func TestExistingPathKeepsUsersSpelling(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	content := `collections:
  space:
    path: C:\corpus
    pattern: "**/*.md"
    ignore: []
`
	if err := os.WriteFile(config, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: nil},
	}
	outcome, err := SyncCollections(config, wanted, sameRecord(owned))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcome.Changed) != 0 {
		t.Errorf("expected no change for identical path with different slashes, got %#v", outcome.Changed)
	}
}

func TestMovedAreaGetsOurOwnSpelling(t *testing.T) {
	tmp := t.TempDir()
	config := writeTestYAML(t, tmp)
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/moved", Pattern: "**/*.md", Ignore: nil},
	}
	outcome, err := SyncCollections(config, wanted, sameRecord(owned))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcome.Changed) != 1 || outcome.Changed[0] != "space" {
		t.Errorf("expected space changed, got %#v", outcome.Changed)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	space := doc["collections"].(map[string]any)["space"].(map[string]any)
	if space["path"] != "C:/moved" {
		t.Errorf("expected C:/moved, got %#v", space["path"])
	}
}

func TestRepeatedIgnorePatternIsWrittenOnce(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: []string{"a", "b", "a"}},
	}
	if _, err := SyncCollections(config, wanted, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	space := doc["collections"].(map[string]any)["space"].(map[string]any)
	ignores := space["ignore"].([]any)
	if len(ignores) != 2 || ignores[0] != "a" || ignores[1] != "b" {
		t.Errorf("expected deduplicated ignores [a, b], got %#v", ignores)
	}
}

func TestNothingIsLeftBesideFileAfterWrite(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: nil},
	}
	if _, err := SyncCollections(config, wanted, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".brain-tmp") {
			t.Errorf("found leftover temp file: %s", e.Name())
		}
	}
}

func TestNonStringKeyInDocumentSurvives(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	content := "collections: {}\n7: seven\n"
	if err := os.WriteFile(config, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	owned := writeTestOwned(t, tmp, []string{"space"})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/corpus", Pattern: "**/*.md", Ignore: nil},
	}
	if _, err := SyncCollections(config, wanted, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[any]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc[7] != "seven" && doc["7"] != "seven" {
		t.Errorf("expected key 7 to survive, got %#v", doc)
	}
}

func TestModels(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	content := `models:
  embed: hf:ggml-org/embeddinggemma-300M-GGUF/embeddinggemma-300M-Q8_0.gguf
  generate: hf:tobil/qmd-query-expansion-1.7B-gguf/qmd-query-expansion-1.7B-q4_k_m.gguf
  rerank: hf:ggml-org/Qwen3-Reranker-0.6B-Q8_0-GGUF/qwen3-reranker-0.6b-q8_0.gguf
`
	if err := os.WriteFile(config, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m := Models(config)
	if m["embedding"] != "hf:ggml-org/embeddinggemma-300M-GGUF/embeddinggemma-300M-Q8_0.gguf" {
		t.Errorf("unexpected embedding: %s", m["embedding"])
	}
	if m["query_expansion"] != "hf:tobil/qmd-query-expansion-1.7B-gguf/qmd-query-expansion-1.7B-q4_k_m.gguf" {
		t.Errorf("unexpected query_expansion: %s", m["query_expansion"])
	}
	if m["rerank"] != "hf:ggml-org/Qwen3-Reranker-0.6B-Q8_0-GGUF/qwen3-reranker-0.6b-q8_0.gguf" {
		t.Errorf("unexpected rerank: %s", m["rerank"])
	}

	t.Run("absent config yields unknowns", func(t *testing.T) {
		m := Models(filepath.Join(tmp, "absent.yml"))
		if m["embedding"] != "unknown" || m["query_expansion"] != "unknown" || m["rerank"] != "unknown" {
			t.Errorf("expected all unknown, got %#v", m)
		}
	})

	t.Run("missing single model entry yields unknown", func(t *testing.T) {
		partial := filepath.Join(tmp, "partial.yml")
		if err := os.WriteFile(partial, []byte("models:\n  embed: embed-me\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		m := Models(partial)
		if m["embedding"] != "embed-me" || m["query_expansion"] != "unknown" || m["rerank"] != "unknown" {
			t.Errorf("unexpected partial models: %#v", m)
		}
	})

	t.Run("wrong structure yields unknowns", func(t *testing.T) {
		wrong := filepath.Join(tmp, "wrong.yml")
		if err := os.WriteFile(wrong, []byte("models: nonsense\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		m := Models(wrong)
		if m["embedding"] != "unknown" {
			t.Errorf("expected unknown, got %#v", m)
		}
	})

	t.Run("broken YAML yields unknowns", func(t *testing.T) {
		broken := filepath.Join(tmp, "broken_models.yml")
		if err := os.WriteFile(broken, []byte("models: [\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		m := Models(broken)
		if m["embedding"] != "unknown" {
			t.Errorf("expected unknown on broken YAML, got %#v", m)
		}
	})

	t.Run("document without models key yields unknowns", func(t *testing.T) {
		noModels := filepath.Join(tmp, "no_models.yml")
		if err := os.WriteFile(noModels, []byte("collections: {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		m := Models(noModels)
		if m["embedding"] != "unknown" {
			t.Errorf("expected unknown when models omitted, got %#v", m)
		}
	})
}

func TestDropCollections(t *testing.T) {
	tmp := t.TempDir()
	config := writeTestYAML(t, tmp)

	dropped, err := DropCollections(config, []string{"space", "never-there"})
	if err != nil {
		t.Fatal(err)
	}
	if len(dropped) != 1 || dropped[0] != "space" {
		t.Errorf("expected space dropped, got %#v", dropped)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	cols := doc["collections"].(map[string]any)
	if _, found := cols["space"]; found {
		t.Error("expected space removed")
	}
	if _, found := cols["foreign"]; !found {
		t.Error("expected foreign retained")
	}

	t.Run("dropping nothing does not rewrite file", func(t *testing.T) {
		before, _ := os.ReadFile(config)
		dropped, err := DropCollections(config, []string{"absent"})
		if err != nil {
			t.Fatal(err)
		}
		if len(dropped) != 0 {
			t.Errorf("expected none dropped, got %#v", dropped)
		}
		after, _ := os.ReadFile(config)
		if string(before) != string(after) {
			t.Error("file was rewritten unnecessarily")
		}
	})

	t.Run("dropping from missing config or non-mapping is no-op", func(t *testing.T) {
		missing := filepath.Join(tmp, "missing.yml")
		dropped, err := DropCollections(missing, []string{"a"})
		if err != nil || len(dropped) != 0 {
			t.Errorf("expected empty no-op, got %v, %v", dropped, err)
		}

		noCols := filepath.Join(tmp, "no_cols.yml")
		if err := os.WriteFile(noCols, []byte("models: {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		dropped, err = DropCollections(noCols, []string{"a"})
		if err != nil || len(dropped) != 0 {
			t.Errorf("expected empty no-op, got %v, %v", dropped, err)
		}
	})

	t.Run("dropping from unreadable YAML returns error", func(t *testing.T) {
		broken := filepath.Join(tmp, "broken.yml")
		if err := os.WriteFile(broken, []byte("collections: [\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := DropCollections(broken, []string{"a"}); err == nil {
			t.Fatal("expected error on broken YAML")
		}
	})
}

func TestSyncCollectionsCollisionRefused(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	content := `collections:
  space:
    path: C:/someone/space
    pattern: "**/*.md"
`
	if err := os.WriteFile(config, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	owned := writeTestOwned(t, tmp, []string{}) // space not in owned

	wanted := map[string]CollectionSpec{
		"space": {Path: filepath.Join(tmp, "ours"), Pattern: "**/*.md", Ignore: nil},
	}
	outcome, err := SyncCollections(config, wanted, sameRecord(owned))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcome.Refused) != 1 || outcome.Refused[0] != "space" {
		t.Errorf("expected space refused, got %#v", outcome.Refused)
	}
	if len(outcome.Changed) != 0 {
		t.Errorf("expected nothing changed, got %#v", outcome.Changed)
	}

	raw, _ := os.ReadFile(config)
	if !strings.Contains(string(raw), "someone") {
		t.Error("foreign collection was modified despite refusal")
	}
}

func TestCollectionWeMadeBeforeIsOursToUpdate(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	owned := filepath.Join(tmp, "owned.json")

	first := map[string]CollectionSpec{
		"space": {Path: "C:/a", Pattern: "**/*.md", Ignore: nil},
	}
	outcome, err := SyncCollections(config, first, sameRecord(owned))
	if err != nil || len(outcome.Changed) != 1 {
		t.Fatalf("first sync failed: %v, %#v", err, outcome)
	}

	// Move area
	second := map[string]CollectionSpec{
		"space": {Path: "C:/b", Pattern: "**/*.md", Ignore: nil},
	}
	outcome, err = SyncCollections(config, second, sameRecord(owned))
	if err != nil || len(outcome.Changed) != 1 {
		t.Fatalf("second sync failed: %v, %#v", err, outcome)
	}

	raw, _ := os.ReadFile(config)
	if !strings.Contains(string(raw), "C:/b") {
		t.Errorf("expected C:/b in config, got:\n%s", raw)
	}
}

func TestOwnershipRecordSurvivesUnchangedRun(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	owned := filepath.Join(tmp, "owned.json")

	spec := map[string]CollectionSpec{
		"space": {Path: "C:/a", Pattern: "**/*.md", Ignore: nil},
	}
	if _, err := SyncCollections(config, spec, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncCollections(config, spec, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(owned)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "space") {
		t.Errorf("expected space in owned: %s", data)
	}
}

func TestUnreadableOwnershipRecordRefuses(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	owned := filepath.Join(tmp, "owned.json")
	if err := os.WriteFile(owned, []byte("{broken json"), 0o644); err != nil {
		t.Fatal(err)
	}

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/ours", Pattern: "**/*.md", Ignore: nil},
	}
	if _, err := SyncCollections(config, wanted, sameRecord(owned)); err == nil {
		t.Fatal("expected error on broken owned.json")
	}
}

func TestWrongShapeOwnershipRecordRefuses(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	owned := filepath.Join(tmp, "owned.json")
	if err := os.WriteFile(owned, []byte(`{"space": true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/ours", Pattern: "**/*.md", Ignore: nil},
	}
	if _, err := SyncCollections(config, wanted, sameRecord(owned)); err == nil {
		t.Fatal("expected error on object-shaped owned.json")
	}
}

func TestStrangerDoesNotStopAreasBesideIt(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	content := `collections:
  space:
    path: C:/someone/space
    pattern: "**/*.md"
`
	if err := os.WriteFile(config, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	owned := writeTestOwned(t, tmp, []string{})

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/ours", Pattern: "**/*.md", Ignore: nil},
		"notes": {Path: "C:/notes", Pattern: "**/*.md", Ignore: nil},
	}
	outcome, err := SyncCollections(config, wanted, sameRecord(owned))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcome.Refused) != 1 || outcome.Refused[0] != "space" {
		t.Errorf("expected space refused, got %#v", outcome.Refused)
	}
	if len(outcome.Changed) != 1 || outcome.Changed[0] != "notes" {
		t.Errorf("expected notes changed, got %#v", outcome.Changed)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	cols := doc["collections"].(map[string]any)
	if cols["space"].(map[string]any)["path"] != "C:/someone/space" {
		t.Error("space was overwritten")
	}
	if cols["notes"].(map[string]any)["path"] != "C:/notes" {
		t.Error("notes was not created")
	}

	// Refused name is never recorded in owned
	ownedData, _ := os.ReadFile(owned)
	var ownedList []string
	if err := json.Unmarshal(ownedData, &ownedList); err != nil {
		t.Fatal(err)
	}
	if len(ownedList) != 1 || ownedList[0] != "notes" {
		t.Errorf("expected only notes in owned, got %#v", ownedList)
	}
}

func TestPruneCollections(t *testing.T) {
	tmp := t.TempDir()
	config := filepath.Join(tmp, "index.yml")
	owned := filepath.Join(tmp, "owned.json")

	wanted := map[string]CollectionSpec{
		"space": {Path: "C:/space", Pattern: "**/*.md", Ignore: nil},
		"notes": {Path: "C:/notes", Pattern: "**/*.md", Ignore: nil},
	}
	if _, err := SyncCollections(config, wanted, sameRecord(owned)); err != nil {
		t.Fatal(err)
	}

	// Prune space (keep only notes)
	dropped, err := PruneCollections(config, []string{"notes"}, sameRecord(owned))
	if err != nil {
		t.Fatal(err)
	}
	if len(dropped) != 1 || dropped[0] != "space" {
		t.Errorf("expected space pruned, got %#v", dropped)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	cols := doc["collections"].(map[string]any)
	if _, found := cols["space"]; found {
		t.Error("expected space removed from config")
	}
	if _, found := cols["notes"]; !found {
		t.Error("expected notes kept in config")
	}

	// Space removed from owned record
	ownedData, _ := os.ReadFile(owned)
	var ownedList []string
	if err := json.Unmarshal(ownedData, &ownedList); err != nil {
		t.Fatal(err)
	}
	if len(ownedList) != 1 || ownedList[0] != "notes" {
		t.Errorf("expected only notes in owned, got %#v", ownedList)
	}

	t.Run("never touches collections we did not make", func(t *testing.T) {
		foreignConfig := filepath.Join(tmp, "foreign.yml")
		if err := os.WriteFile(foreignConfig, []byte("collections:\n  someone_else:\n    path: C:/theirs\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		foreignOwned := filepath.Join(tmp, "foreign_owned.json")
		if _, err := SyncCollections(foreignConfig, map[string]CollectionSpec{
			"space": {Path: "C:/space", Pattern: "**/*.md", Ignore: nil},
		}, sameRecord(foreignOwned)); err != nil {
			t.Fatal(err)
		}
		dropped, err := PruneCollections(foreignConfig, nil, sameRecord(foreignOwned))
		if err != nil {
			t.Fatal(err)
		}
		if len(dropped) != 1 || dropped[0] != "space" {
			t.Errorf("expected space dropped, got %#v", dropped)
		}
		raw, _ := os.ReadFile(foreignConfig)
		if !strings.Contains(string(raw), "someone_else") {
			t.Error("someone_else was pruned")
		}
	})

	t.Run("pruning with nothing orphaned writes nothing", func(t *testing.T) {
		before, _ := os.ReadFile(config)
		dropped, err := PruneCollections(config, []string{"notes"}, sameRecord(owned))
		if err != nil {
			t.Fatal(err)
		}
		if len(dropped) != 0 {
			t.Errorf("expected nothing dropped, got %#v", dropped)
		}
		after, _ := os.ReadFile(config)
		if string(before) != string(after) {
			t.Error("file was rewritten on no-op prune")
		}
	})
}

func TestQmdConfigPathUserHomeError(t *testing.T) {
	orig := userHomeDir
	t.Cleanup(func() { userHomeDir = orig })
	userHomeDir = func() (string, error) {
		return "", os.ErrNotExist
	}
	t.Setenv("XDG_CONFIG_HOME", "")

	got := QmdConfigPath()
	want := filepath.Join("", ".config", "qmd", "index.yml")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToStringSliceAndEntriesEqual(t *testing.T) {
	if s := toStringSlice(123); s != nil {
		t.Errorf("expected nil for non-slice, got %#v", s)
	}
	mixed := toStringSlice([]any{123, "text"})
	if len(mixed) != 2 || mixed[0] != "123" || mixed[1] != "text" {
		t.Errorf("expected string conversion for non-string items, got %#v", mixed)
	}

	// entriesEqual tests
	if entriesEqual("not a map", map[string]any{"a": "1"}) {
		t.Error("expected false for non-map existing")
	}
	if entriesEqual(map[string]any{"a": "1", "b": "2"}, map[string]any{"a": "1"}) {
		t.Error("expected false for different lengths")
	}
	if entriesEqual(map[string]any{"a": "1"}, map[string]any{"b": "1"}) {
		t.Error("expected false for missing key")
	}
	if entriesEqual(map[string]any{"ignore": []string{"a"}}, map[string]any{"ignore": []string{"b"}}) {
		t.Error("expected false for mismatched ignore")
	}
	if entriesEqual(map[string]any{"custom": "x"}, map[string]any{"custom": "y"}) {
		t.Error("expected false for mismatched custom field")
	}
}

func TestReadOwnedDirectoryError(t *testing.T) {
	tmp := t.TempDir()
	// Reading a directory as a file produces an OS read error (not IsNotExist)
	if _, err := readOwned(tmp); err == nil {
		t.Fatal("expected error reading directory as owned JSON file")
	}
}

func TestSyncCollectionsErrorPaths(t *testing.T) {
	tmp := t.TempDir()

	t.Run("configPath is a directory", func(t *testing.T) {
		wanted := map[string]CollectionSpec{"space": {Path: "C:/corpus"}}
		owned := filepath.Join(tmp, "owned.json")
		if _, err := SyncCollections(tmp, wanted, sameRecord(owned)); err == nil {
			t.Fatal("expected error when configPath is a directory")
		}
	})

	t.Run("backup failure", func(t *testing.T) {
		cfg := filepath.Join(tmp, "sub1", "index.yml")
		_ = os.MkdirAll(filepath.Dir(cfg), 0o755)
		_ = os.WriteFile(cfg, []byte(testExistingYAML), 0o644)
		// Block temp file creation for backup by making the temp path a directory
		_ = os.MkdirAll(cfg+backupSuffix+tempSuffix, 0o755)
		owned := writeTestOwned(t, tmp, []string{"space"})
		wanted := map[string]CollectionSpec{"space": {Path: "C:/corpus", Pattern: "changed"}}
		if _, err := SyncCollections(cfg, wanted, sameRecord(owned)); err == nil {
			t.Fatal("expected error when backup cannot be written")
		}
	})

	t.Run("replaceWith config failure", func(t *testing.T) {
		// Parent path blocking directory creation
		blockingFile := filepath.Join(tmp, "blocked_cfg")
		_ = os.WriteFile(blockingFile, []byte("block"), 0o644)
		cfg := filepath.Join(blockingFile, "index.yml")
		wanted := map[string]CollectionSpec{"space": {Path: "C:/corpus"}}
		owned := filepath.Join(tmp, "owned.json")
		if _, err := SyncCollections(cfg, wanted, sameRecord(owned)); err == nil {
			t.Fatal("expected error when config write fails")
		}
	})

	t.Run("remember failure", func(t *testing.T) {
		cfg := filepath.Join(tmp, "sub2", "index.yml")
		_ = os.MkdirAll(filepath.Dir(cfg), 0o755)
		_ = os.WriteFile(cfg, []byte("collections: {}\n"), 0o644)
		// Owned path cannot be created because blocking file sits where dir is expected
		blockingFile := filepath.Join(tmp, "blocked_owned")
		_ = os.WriteFile(blockingFile, []byte("block"), 0o644)
		owned := filepath.Join(blockingFile, "owned.json")
		wanted := map[string]CollectionSpec{"space": {Path: "C:/corpus"}}
		if _, err := SyncCollections(cfg, wanted, sameRecord(owned)); err == nil {
			t.Fatal("expected error when remember write fails")
		}
	})
}

func TestPruneCollectionsErrorPaths(t *testing.T) {
	tmp := t.TempDir()

	t.Run("readOwned error", func(t *testing.T) {
		if _, err := PruneCollections(filepath.Join(tmp, "cfg.yml"), []string{"a"}, sameRecord(tmp)); err == nil {
			t.Fatal("expected error when ownedPath is directory")
		}
	})

	t.Run("DropCollections error propagates", func(t *testing.T) {
		cfg := filepath.Join(tmp, "broken.yml")
		_ = os.WriteFile(cfg, []byte("collections: [\n"), 0o644)
		owned := writeTestOwned(t, tmp, []string{"space"})
		if _, err := PruneCollections(cfg, []string{"other"}, sameRecord(owned)); err == nil {
			t.Fatal("expected error from broken YAML in DropCollections")
		}
	})

	t.Run("remember failure", func(t *testing.T) {
		cfg := writeTestYAML(t, tmp)
		owned := filepath.Join(tmp, "prune_owned.json")
		_ = os.WriteFile(owned, []byte(`["space"]`), 0o644)
		// Blocking temp file creation for remember by making temp file path a directory
		_ = os.MkdirAll(owned+tempSuffix, 0o755)
		if _, err := PruneCollections(cfg, nil, sameRecord(owned)); err == nil {
			t.Fatal("expected error when remember fails during prune")
		}
	})
}

func TestDropCollectionsErrorPaths(t *testing.T) {
	tmp := t.TempDir()

	t.Run("configPath is a directory", func(t *testing.T) {
		if _, err := DropCollections(tmp, []string{"a"}); err == nil {
			t.Fatal("expected error when configPath is directory")
		}
	})

	t.Run("collections is not a map", func(t *testing.T) {
		cfg := filepath.Join(tmp, "str_cols.yml")
		_ = os.WriteFile(cfg, []byte("collections: string_val\n"), 0o644)
		dropped, err := DropCollections(cfg, []string{"a"})
		if err != nil || len(dropped) != 0 {
			t.Errorf("expected no-op, got %v, %v", dropped, err)
		}
	})

	t.Run("replaceWith failure", func(t *testing.T) {
		cfg := filepath.Join(tmp, "drop_target.yml")
		_ = os.WriteFile(cfg, []byte("collections:\n  space: {}\n"), 0o644)
		// Block temp file creation for replaceWith by creating temp path as directory
		_ = os.MkdirAll(cfg+tempSuffix, 0o755)
		if _, err := DropCollections(cfg, []string{"space"}); err == nil {
			t.Fatal("expected error when replacing directory target")
		}
	})
}

func TestReplaceWithRenameError(t *testing.T) {
	tmp := t.TempDir()
	destDir := filepath.Join(tmp, "dest_dir")
	_ = os.MkdirAll(destDir, 0o755)
	_ = os.WriteFile(filepath.Join(destDir, "child"), []byte("data"), 0o644)

	err := replaceWith(destDir, []byte("new data"))
	if err == nil {
		t.Fatal("expected error renaming file onto existing directory")
	}
}

// sameRecord is an ownership record that is read and written in one place,
// the state of a machine that never had the reference's directory.
func sameRecord(path string) OwnershipRecord {
	return OwnershipRecord{Read: path, Write: path}
}

func TestQmdConfigPathForNamesTheIndexFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/x")
	if got := QmdConfigPathFor("loomux-bench-a"); got != filepath.Join("/x", "qmd", "loomux-bench-a.yml") {
		t.Fatalf("got %q", got)
	}
	if QmdConfigPath() != QmdConfigPathFor("index") {
		t.Fatal("QmdConfigPath is not the index named index")
	}
}

func TestQmdCacheDirHonoursXDG(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/c")
	if got := QmdCacheDir(); got != filepath.Join("/c", "qmd") {
		t.Fatalf("got %q", got)
	}
}

func TestQmdCacheDirFallsBackToTheHomeCache(t *testing.T) {
	orig := userHomeDir
	t.Cleanup(func() { userHomeDir = orig })
	userHomeDir = func() (string, error) { return "/home/u", nil }
	t.Setenv("XDG_CACHE_HOME", "")
	if got := QmdCacheDir(); got != filepath.Join("/home/u", ".cache", "qmd") {
		t.Fatalf("got %q", got)
	}
}
