package importcases

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// buildOldWorld writes the configuration of both old tools into dir.
func buildOldWorld(t *testing.T, dir string, manifest string) {
	t.Helper()
	write := func(rel, content string) {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".brain.toml", manifest)
	write(".ultraloom/policy.toml", "[[policy.paths.rules]]\nmatch = \"generated/*\"\nreason = \"generated files are rebuilt, not edited\"\n")
	write(".ultraloom/config.toml", "[verify]\ngofmt = true\n\n[worktree]\nmirror = [\".env\"]\n")
	write(".ultraloom/answers.toml", "bundle = \"docs/wiki/\"\n")
}

const manifestWithLayout = "[area]\nwiki = true\n\n[layout]\nwiki = \"docs/handbook\"\n\n[check.lanes]\nwiki = [\"lint\"]\n"

const manifestWithoutLayout = "[area]\nwiki = true\n\n[check.lanes]\nwiki = [\"lint\"]\n"

func decodeConfig(t *testing.T, dir string) map[string]any {
	t.Helper()
	var got map[string]any
	if _, err := toml.DecodeFile(filepath.Join(dir, ".loomux", "config.toml"), &got); err != nil {
		t.Fatalf("decoding the translated config in %s: %v", dir, err)
	}
	return got
}

func entries(t *testing.T, dir string) []string {
	t.Helper()
	items, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, item := range items {
		names = append(names, item.Name())
	}
	sort.Strings(names)
	return names
}

func TestTranslateWorldWritesOneConfigAndDropsTheOldFiles(t *testing.T) {
	dir := t.TempDir()
	buildOldWorld(t, dir, manifestWithLayout)
	buildOldWorld(t, filepath.Join(dir, "areas", "notes"), manifestWithLayout)

	if err := TranslateWorld(dir); err != nil {
		t.Fatal(err)
	}

	for _, d := range []string{dir, filepath.Join(dir, "areas", "notes")} {
		got := decodeConfig(t, d)
		for _, key := range []string{"area", "layout", "policy", "worktree"} {
			if _, ok := got[key]; !ok {
				t.Errorf("%s: missing %q in %v", d, key, got)
			}
		}
		for _, key := range []string{"check", "verify"} {
			if _, ok := got[key]; ok {
				t.Errorf("%s: %q survived the translation", d, key)
			}
		}
		layout, _ := got["layout"].(map[string]any)
		if layout["wiki"] != "docs/handbook" {
			t.Errorf("%s: layout.wiki %v, want the manifest's", d, layout["wiki"])
		}
		policy, _ := got["policy"].(map[string]any)
		paths, _ := policy["paths"].(map[string]any)
		rules, _ := paths["rules"].([]map[string]any)
		if len(rules) != 1 || rules[0]["match"] != "generated/*" {
			t.Errorf("%s: policy.paths.rules %#v", d, rules)
		}
	}

	if got := entries(t, dir); strings.Join(got, " ") != ".loomux areas" {
		t.Errorf("leftovers in the translated world: %v", got)
	}
	if got := entries(t, filepath.Join(dir, "areas", "notes")); strings.Join(got, " ") != ".loomux" {
		t.Errorf("leftovers in the translated area: %v", got)
	}
}

func TestTranslateWorldTakesTheWikiFromTheBundleOnlyWithoutAManifestLayout(t *testing.T) {
	dir := t.TempDir()
	buildOldWorld(t, dir, manifestWithoutLayout)

	if err := TranslateWorld(dir); err != nil {
		t.Fatal(err)
	}

	layout, _ := decodeConfig(t, dir)["layout"].(map[string]any)
	if layout["wiki"] != "docs/wiki" {
		t.Errorf("layout.wiki %v, want the bundle without its trailing slash", layout["wiki"])
	}
}

func TestTranslateWorldLeavesAWorldWithoutOldConfigurationAlone(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := TranslateWorld(dir); err != nil {
		t.Fatal(err)
	}
	if got := entries(t, dir); strings.Join(got, " ") != "a.txt" {
		t.Errorf("entries %v", got)
	}
}

func TestTranslateWorldReportsAConfigItCannotRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte("[area\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := TranslateWorld(dir); err == nil {
		t.Fatal("want error")
	}
}

// buildCase writes one recorded case below from.
func buildCase(t *testing.T, from, verb, name, cmd, stdin string) string {
	t.Helper()
	dir := filepath.Join(from, verb, name)
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	for rel, content := range map[string]string{
		"cmd":      cmd + "\n",
		"exit":     "2\n",
		"stdout":   "",
		"stdin":    stdin,
		"notes.md": "notes\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestImportRewritesTheCommandAndTranslatesTheWorlds(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	dir := buildCase(t, from, "guard", "deny-readonly",
		"ulguard --root {{WORLD}}",
		`{"file_path":"{{WORLD}}/.ultraloom/policy.toml","manifest":".brain.toml"}`)
	buildOldWorld(t, filepath.Join(dir, "world"), manifestWithLayout)
	if err := os.MkdirAll(filepath.Join(dir, "world_after"), 0o755); err != nil {
		t.Fatal(err)
	}
	buildOldWorld(t, filepath.Join(dir, "world_after"), manifestWithLayout)

	m := Mapping{Commands: []Rule{{
		From: "ulguard --root {{WORLD}}",
		To:   "loomux hook pre-tool-use --host claude --root {{WORLD}}",
	}}}
	if err := Import(from, to, m); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(to, "guard", "deny-readonly")
	cmd, err := os.ReadFile(filepath.Join(out, "cmd"))
	if err != nil || string(cmd) != "loomux hook pre-tool-use --host claude --root {{WORLD}}\n" {
		t.Fatalf("%v %q", err, cmd)
	}
	stdin, err := os.ReadFile(filepath.Join(out, "stdin"))
	want := `{"file_path":"{{WORLD}}/.loomux/config.toml","manifest":".loomux/config.toml"}`
	if err != nil || string(stdin) != want {
		t.Fatalf("%v %q", err, stdin)
	}
	for _, world := range []string{"world", "world_after"} {
		if _, err := os.Stat(filepath.Join(out, world, ".loomux", "config.toml")); err != nil {
			t.Errorf("%s not translated: %v", world, err)
		}
	}
	// The recording stays as it was.
	if _, err := os.Stat(filepath.Join(dir, "world", ".brain.toml")); err != nil {
		t.Errorf("the source recording was changed: %v", err)
	}
}

func TestImportRefusesACaseWithoutARule(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	buildCase(t, from, "guard", "deny-readonly", "brain guard {{WORLD}}", "")

	err := Import(from, to, Mapping{Commands: []Rule{{From: "ulguard", To: "loomux hook pre-tool-use"}}})
	if err == nil || !strings.Contains(err.Error(), "guard/deny-readonly") {
		t.Fatalf("want an error naming the case, got %v", err)
	}
}

func TestImportReportsWhatItCannotRead(t *testing.T) {
	to := t.TempDir()
	if err := Import(filepath.Join(t.TempDir(), "gone"), to, Mapping{}); err == nil {
		t.Fatal("want error for a missing source")
	}

	from := t.TempDir()
	buildCase(t, from, "guard", "one", "ulguard --root {{WORLD}}", "")
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Mapping{Commands: []Rule{{From: "ulguard", To: "loomux hook pre-tool-use"}}}
	if err := Import(from, blocked, m); err == nil {
		t.Fatal("want error for a target under a file")
	}
}

func TestImportReportsAWorldItCannotTranslate(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	dir := buildCase(t, from, "guard", "one", "ulguard --root {{WORLD}}", "")
	if err := os.WriteFile(filepath.Join(dir, "world", ".brain.toml"), []byte("[area\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Mapping{Commands: []Rule{{From: "ulguard", To: "loomux hook pre-tool-use"}}}
	if err := Import(from, to, m); err == nil {
		t.Fatal("want error for a world that does not decode")
	}
}

// brokenWorld writes a valid manifest and one unreadable old file beside it.
func brokenWorld(t *testing.T, rel string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte(manifestWithLayout), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[broken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestTranslateWorldReportsEveryOldFileItCannotRead(t *testing.T) {
	for _, rel := range []string{".ultraloom/policy.toml", ".ultraloom/config.toml", ".ultraloom/answers.toml"} {
		if err := TranslateWorld(brokenWorld(t, rel)); err == nil {
			t.Errorf("%s: want error", rel)
		}
	}
}

func TestTranslateWorldReportsAConfigItCannotWrite(t *testing.T) {
	dir := t.TempDir()
	buildOldWorld(t, dir, manifestWithLayout)
	if err := os.WriteFile(filepath.Join(dir, ".loomux"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := TranslateWorld(dir); err == nil {
		t.Fatal("want error for a .loomux blocked by a file")
	}

	dir = t.TempDir()
	buildOldWorld(t, dir, manifestWithLayout)
	if err := os.MkdirAll(filepath.Join(dir, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := TranslateWorld(dir); err == nil {
		t.Fatal("want error for a config blocked by a directory")
	}
}

func TestTranslateWorldPrefersTheUltraBrainManifestAndReportsWhatItCannotDelete(t *testing.T) {
	dir := t.TempDir()
	buildOldWorld(t, dir, manifestWithoutLayout)
	if err := os.MkdirAll(filepath.Join(dir, ".ultra-brain"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".ultra-brain", "config.toml"), []byte(manifestWithLayout), 0o644); err != nil {
		t.Fatal(err)
	}
	// A .brain.toml that is a non-empty directory is read past and cannot be
	// deleted: the ultra-brain manifest won.
	if err := os.Remove(filepath.Join(dir, ".brain.toml")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".brain.toml", "held"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := TranslateWorld(dir); err == nil {
		t.Fatal("want error for an old file that cannot be removed")
	}
	layout, _ := decodeConfig(t, dir)["layout"].(map[string]any)
	if layout["wiki"] != "docs/handbook" {
		t.Errorf("layout.wiki %v, want the ultra-brain manifest's", layout["wiki"])
	}
}

func TestTranslateWorldWalksOnlyTheAreaDirectories(t *testing.T) {
	dir := t.TempDir()
	buildOldWorld(t, dir, manifestWithLayout)
	if err := os.MkdirAll(filepath.Join(dir, "areas"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "areas", "README.md"), []byte("not an area\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := filepath.Join(dir, "areas", "notes")
	if err := os.MkdirAll(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, ".brain.toml"), []byte("[area\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := TranslateWorld(dir); err == nil {
		t.Fatal("want error from the area")
	}
}

func TestImportCarriesACaseWithoutAWorldAfterOrAStdin(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	dir := buildCase(t, from, "guard", "one", "ulguard --root {{WORLD}}", "")
	if err := os.Remove(filepath.Join(dir, "stdin")); err != nil {
		t.Fatal(err)
	}
	m := Mapping{Commands: []Rule{{From: "ulguard --root", To: "loomux hook pre-tool-use --root"}}}
	if err := Import(from, to, m); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(to, "guard", "one")
	if got := entries(t, out); strings.Join(got, " ") != "cmd exit notes.md stdout world" {
		t.Errorf("entries %v", got)
	}
}

func TestImportReportsTheFilesItRewritesAndCannotWrite(t *testing.T) {
	m := Mapping{Commands: []Rule{{From: "ulguard", To: "loomux hook pre-tool-use"}}}
	for _, name := range []string{"cmd", "stdin"} {
		from, to := t.TempDir(), t.TempDir()
		buildCase(t, from, "guard", "one", "ulguard --root {{WORLD}}", `{"x":1}`)
		if err := os.MkdirAll(filepath.Join(to, "guard", "one", name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := Import(from, to, m); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}
