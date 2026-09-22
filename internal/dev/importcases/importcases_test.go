package importcases

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/testlock"
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

// A case dropped from the recordings has to disappear from the translated
// corpus too. It used to survive: the import only ever wrote, so the corpus
// kept a case no recording backs any more -- and a pinned case count, which
// counts what is there, does not notice a swap. The verb directory goes with
// its last case, and a second import leaves a case still backed alone.
func TestImportDropsACaseNoRecordingBacksAnyMore(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	buildCase(t, from, "guard", "kept", "ulguard --root {{WORLD}}", "")
	buildCase(t, to, "guard", "dropped", "loomux hook pre-tool-use", "")
	buildCase(t, to, "lint", "gone", "loomux lint x.md", "")
	m := Mapping{Commands: []Rule{{From: "ulguard", To: "loomux hook pre-tool-use"}}}

	for round := 1; round <= 2; round++ {
		if err := Import(from, to, m); err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		if _, err := os.Stat(filepath.Join(to, "guard", "dropped")); !os.IsNotExist(err) {
			t.Errorf("round %d: the dropped case survived: %v", round, err)
		}
		if _, err := os.Stat(filepath.Join(to, "lint")); !os.IsNotExist(err) {
			t.Errorf("round %d: the emptied verb directory survived: %v", round, err)
		}
		if _, err := os.Stat(filepath.Join(to, "guard", "kept", "cmd")); err != nil {
			t.Errorf("round %d: the recorded case is not there: %v", round, err)
		}
	}
}

// A target that does not exist yet holds nothing to prune, and is no error.
func TestImportWritesIntoATargetThatIsNotThereYet(t *testing.T) {
	from := t.TempDir()
	to := filepath.Join(t.TempDir(), "fresh")
	buildCase(t, from, "guard", "one", "ulguard --root {{WORLD}}", "")

	if err := Import(from, to, Mapping{Commands: []Rule{{From: "ulguard", To: "loomux hook pre-tool-use"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(to, "guard", "one", "cmd")); err != nil {
		t.Fatal(err)
	}
}

// A corpus in the target that cannot be read is an error: what may be removed
// is not decidable without it.
func TestImportReportsATargetCorpusItCannotRead(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	buildCase(t, from, "guard", "one", "ulguard --root {{WORLD}}", "")
	broken := filepath.Join(to, "guard", "broken")
	if err := os.MkdirAll(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	// A case directory without an exit code is one LoadCase refuses.
	if err := os.WriteFile(filepath.Join(broken, "cmd"), []byte("loomux x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Import(from, to, Mapping{Commands: []Rule{{From: "ulguard", To: "loomux hook pre-tool-use"}}})
	if err == nil || !strings.Contains(err.Error(), "reading the corpus") {
		t.Fatalf("want an error naming the target corpus, got %v", err)
	}
}

// A dropped case it cannot remove is an error, not a corpus half of two
// imports.
func TestImportReportsADroppedCaseItCannotRemove(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	buildCase(t, from, "guard", "one", "ulguard --root {{WORLD}}", "")
	dropped := buildCase(t, to, "guard", "dropped", "loomux hook pre-tool-use", "")
	// notes.md, not cmd: LoadCase has to read the case before the prune can
	// decide that no recording backs it.
	testlock.Lock(t, filepath.Join(dropped, "notes.md"))

	err := Import(from, to, Mapping{Commands: []Rule{{From: "ulguard", To: "loomux hook pre-tool-use"}}})
	if err == nil || !strings.Contains(err.Error(), "clearing") {
		t.Fatalf("want an error about the case it could not clear, got %v", err)
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

// writeFile writes content to path, making its directory first.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const worldRegistry = `[[area]]
scope = "project/a"
path = "{{WORLD}}/repo-a"

[[area]]
scope = "notes"
path = "{{WORLD}}/areas/notes"
readonly = true

[[area]]
scope = "project/away"
path = "{{WORLD}}/../escape"

[[area]]
scope = "project/elsewhere"
path = "C:/elsewhere/repo-b"

[[area]]
scope = "project/gone"
path = "{{WORLD}}/repo-gone"
`

// A writable area of a 1b-1 world lives where the registry's path puts it,
// not under areas/. Its old manifest has to become loomux's configuration
// where it lives, or loomux reads a manifest the reference never saw.
func TestTranslateWorldFoldsEveryAreaTheRegistryPlacesInTheWorld(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "world")
	writeFile(t, filepath.Join(dir, "registry.toml"), worldRegistry)
	writeFile(t, filepath.Join(dir, "repo-a", ".ultra-brain", "config.toml"),
		"[area]\nscope = \"project/a\"\n\n[privacy]\nmode = \"local_only\"\nnever = [\"secret/**\"]\n")
	writeFile(t, filepath.Join(dir, "areas", "notes", ".brain.toml"), "[area]\nscope = \"notes\"\n")
	writeFile(t, filepath.Join(parent, "escape", ".brain.toml"), "[area]\nscope = \"project/away\"\n")

	if err := TranslateWorld(dir); err != nil {
		t.Fatal(err)
	}

	privacy, _ := decodeConfig(t, filepath.Join(dir, "repo-a"))["privacy"].(map[string]any)
	never, _ := privacy["never"].([]any)
	if privacy["mode"] != "local_only" || len(never) != 1 || never[0] != "secret/**" {
		t.Errorf("privacy of repo-a %v", privacy)
	}
	for _, area := range []string{"repo-a", filepath.Join("areas", "notes")} {
		if got := entries(t, filepath.Join(dir, area)); strings.Join(got, " ") != ".loomux" {
			t.Errorf("%s: leftovers %v", area, got)
		}
	}
	if _, err := os.Stat(filepath.Join(parent, "escape", ".brain.toml")); err != nil {
		t.Errorf("a path outside the world was translated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "repo-gone")); !os.IsNotExist(err) {
		t.Errorf("a registered directory that is not there was made: %v", err)
	}
}

// A registry that does not read is a case of its own: the replay has to meet
// it as the reference did. The import folds the rest of the world and leaves
// the registry as it was recorded.
func TestTranslateWorldLeavesAnUnreadableRegistryToTheReplay(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "registry.toml"), "[[area\n")
	writeFile(t, filepath.Join(dir, "areas", "notes", ".brain.toml"), "[area]\nscope = \"notes\"\n")
	if err := TranslateWorld(dir); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "registry.toml")); err != nil || string(got) != "[[area\n" {
		t.Errorf("registry %q, err %v", got, err)
	}
	if got := entries(t, filepath.Join(dir, "areas", "notes")); strings.Join(got, " ") != ".loomux" {
		t.Errorf("the area beside the registry was not folded: %v", got)
	}
}

func TestTranslateWorldReportsARegisteredAreaItCannotTranslate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "registry.toml"), "[[area]]\nscope = \"project/a\"\npath = \"{{WORLD}}/repo-a\"\n")
	writeFile(t, filepath.Join(dir, "repo-a", ".brain.toml"), "[area\n")
	if err := TranslateWorld(dir); err == nil || !strings.Contains(err.Error(), "repo-a") {
		t.Fatalf("want an error naming the area's manifest, got %v", err)
	}
}

// The worlds of a case the recordings still back are staged afresh: a file
// an earlier import left there -- a merged fixture the recording never had --
// would otherwise survive and be merged into again.
func TestImportStagesTheWorldsOfABackedCaseAfresh(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	buildCase(t, from, "guard", "one", "ulguard --root {{WORLD}}", "")
	stale := filepath.Join(to, "guard", "one", "world", "faketool.json")
	writeFile(t, stale, "{}")
	m := Mapping{Commands: []Rule{{From: "ulguard", To: "loomux hook pre-tool-use"}}}

	if err := Import(from, to, m); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("the file of an earlier import survived: %v", err)
	}

	locked := filepath.Join(to, "guard", "one", "world", "kept.txt")
	writeFile(t, locked, "x")
	testlock.Lock(t, locked)
	if err := Import(from, to, m); err == nil || !strings.Contains(err.Error(), "clearing") {
		t.Fatalf("want an error about the case it could not clear, got %v", err)
	}
}

// A file the recording no longer holds leaves the translated case with it:
// a `compare` left behind would grade a re-recorded data case as a message.
func TestImportDropsAFileTheRecordingDropped(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	buildCase(t, from, "reconcile", "one", "brain-mcp reconcile", "")
	stale := filepath.Join(to, "reconcile", "one", "compare")
	writeFile(t, stale, "message\n")
	m := Mapping{Commands: []Rule{{From: "brain-mcp reconcile", To: "loomux reconcile"}}}
	if err := Import(from, to, m); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("the dropped compare survived: %v", err)
	}

	writeFile(t, stale, "message\n")
	testlock.Lock(t, stale)
	if err := Import(from, to, m); err == nil || !strings.Contains(err.Error(), "clearing") {
		t.Fatalf("want an error about the file it could not drop, got %v", err)
	}
}

func TestTranslateWorldCarriesCommitConfig(t *testing.T) {
	dir := t.TempDir()
	buildOldWorld(t, dir, manifestWithLayout)
	commitConfig := "[verify]\ngofmt = true\n\n[commit]\nlanguage = \"de\"\nthreshold = 3\n"
	writeFile(t, filepath.Join(dir, ".ultraloom", "config.toml"), commitConfig)

	if err := TranslateWorld(dir); err != nil {
		t.Fatal(err)
	}

	got := decodeConfig(t, dir)
	commit, ok := got["commit"].(map[string]any)
	if !ok {
		t.Fatalf("want [commit] table in translated config, got %v", got)
	}
	if commit["language"] != "de" || commit["threshold"] != int64(3) {
		t.Errorf("commit config mismatch: %v", commit)
	}
}

// A stage whose command answers a refusal with another code says so in its
// map, so the translated case still names one expected code and not "differs".
func TestImportMapsTheExitCode(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	dir := filepath.Join(from, "check-commit-msg", "refused")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"cmd": "ultraloom commit-msg {{WORLD}}/msg.txt\n", "exit": "2\n", "stdout": "", "compare": "message\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := Mapping{
		Commands: []Rule{{From: "ultraloom commit-msg ", To: "loomux check commit-msg "}},
		Exits:    []ExitRule{{From: 2, To: 1}},
	}
	if err := Import(from, to, m); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(to, "check-commit-msg", "refused", "exit"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != "1" {
		t.Errorf("exit = %q, want 1", got)
	}
}

func TestImportKeepsAnUnmappedExitCode(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	dir := filepath.Join(from, "check-commit-msg", "refused")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"cmd": "ultraloom commit-msg {{WORLD}}/msg.txt\n", "exit": "1\n", "stdout": "",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := Mapping{
		Commands: []Rule{{From: "ultraloom commit-msg ", To: "loomux check commit-msg "}},
		Exits:    []ExitRule{{From: 2, To: 1}},
	}
	if err := Import(from, to, m); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(to, "check-commit-msg", "refused", "exit"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != "1" {
		t.Errorf("exit = %q, want 1", got)
	}
}

// A mapped exit the import cannot write is an error, not a case carrying the
// recorded code as if no rule had asked for another one.
func TestImportReportsAnExitItCannotWrite(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	dir := filepath.Join(from, "check-commit-msg", "refused")
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"cmd": "ultraloom commit-msg {{WORLD}}/msg.txt\n", "exit": "2\n", "stdout": "", "compare": "message\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	locked := filepath.Join(to, "check-commit-msg", "refused", "exit")
	if err := os.MkdirAll(filepath.Dir(locked), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, locked, "2\n")
	testlock.Lock(t, locked)

	m := Mapping{
		Commands: []Rule{{From: "ultraloom commit-msg ", To: "loomux check commit-msg "}},
		Exits:    []ExitRule{{From: 2, To: 1}},
	}
	if err := Import(from, to, m); err == nil {
		t.Fatal("expected an error for the locked exit file")
	}
}
