package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFlattenKeysStopsAtDepthTwo(t *testing.T) {
	document := map[string]any{
		"top":       "x",
		"area":      map[string]any{"scope": "s"},
		"llm":       map[string]any{"local": map[string]any{"enabled": false}},
		"relevance": []map[string]any{{"a": "b"}},
		"empty":     map[string]any{},
	}
	got := flattenKeys(document)
	want := []string{"area.scope", "empty", "llm.local", "relevance", "top"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flattenKeys = %v, want %v", got, want)
	}
}

func TestClassifyKeysSeparatesReadElsewhereAndIgnored(t *testing.T) {
	document := map[string]any{
		"area":     map[string]any{"scope": "knowledge"},
		"layout":   map[string]any{"inbox": "00", "sources": "10"},
		"commit":   map[string]any{"language": "de"},
		"llm":      map[string]any{"local": map[string]any{"enabled": false}},
		"nonsense": map[string]any{"k": int64(1)},
	}
	got := map[string]keyReport{}
	for _, r := range classifyKeys(document) {
		got[r.ID] = r
	}
	cases := map[string]keyClass{
		"area.scope":      classRead,
		"layout.inbox":    classRead,
		"layout.sources":  classIgnored, // the reader reads no `sources`
		"commit.language": classElsewhere,
		"llm.local":       classIgnored,
		"nonsense.k":      classIgnored,
	}
	for id, class := range cases {
		if got[id].Class != class {
			t.Errorf("%s: class %q, want %q", id, got[id].Class, class)
		}
	}
	if got["llm.local"].Hint == "" {
		t.Errorf("llm.local carries no hint")
	}
	if got["nonsense.k"].Hint != "" {
		t.Errorf("nonsense.k carries hint %q", got["nonsense.k"].Hint)
	}
	if got["layout.sources"].Hint == "" {
		t.Errorf("layout.sources carries no hint")
	}
}

func TestClassifyKeysKnowsTheVerifyTablesTheSchemaDoesNotName(t *testing.T) {
	// internal/verify reads [verify.<stack>.<kind>] and
	// [verify.gdscript] import_check; the schema names neither.
	document := map[string]any{"verify": map[string]any{"go": map[string]any{"lint": "x"}, "gdscript": map[string]any{"import_check": "y"}}}
	reports := classifyKeys(document)
	if len(reports) == 0 {
		t.Fatalf("no reports")
	}
	for _, r := range reports {
		if r.Class == classIgnored {
			t.Errorf("%s reported ignored", r.ID)
		}
	}
}

func TestClassifyKeysCallsALanesTableIgnoredWithAHint(t *testing.T) {
	// ReadManifest decodes [check] lanes into Manifest.Lanes, which nothing
	// uses: a move by hand loses nothing, but no reader acts on it either.
	reports := classifyKeys(map[string]any{"check": map[string]any{"lanes": []any{}}})
	if len(reports) != 1 || reports[0].Class != classIgnored || reports[0].Hint == "" {
		t.Fatalf("got %+v", reports)
	}
}

func TestClassifyKeysKnowsATableTheSchemaOnlyNamesBelow(t *testing.T) {
	// `policy.paths` is not a key of its own; the schema knows
	// `policy.paths.rules`.
	document := map[string]any{"policy": map[string]any{"paths": map[string]any{}}}
	reports := classifyKeys(document)
	if len(reports) != 1 || reports[0].Class != classElsewhere {
		t.Fatalf("got %+v, want one elsewhere entry", reports)
	}
}

func TestClassifyKeysDoesNotTakeAnotherTableForVerify(t *testing.T) {
	// `verifyx` shares the prefix `verify` but is not the verify table.
	reports := classifyKeys(map[string]any{"verifyx": map[string]any{"a": "b"}})
	if len(reports) != 1 || reports[0].ID != "verifyx.a" || reports[0].Class != classIgnored {
		t.Fatalf("got %+v, want verifyx.a ignored", reports)
	}
	// Nor does a table that only contains `verify.`.
	reports = classifyKeys(map[string]any{"xverify": map[string]any{"a": "b"}})
	if len(reports) != 1 || reports[0].ID != "xverify.a" || reports[0].Class != classIgnored {
		t.Fatalf("got %+v, want xverify.a ignored", reports)
	}
}

func TestClassifyKeysKnowsOnlyWholeSegmentsOfASchemaKey(t *testing.T) {
	// `agent.models` is a table of the schema; `agent.mod` is only the start
	// of its name, and `models` alone is only its middle.
	for _, document := range []map[string]any{
		{"agent": map[string]any{"mod": "x"}},
		{"models": "x"},
	} {
		reports := classifyKeys(document)
		if len(reports) != 1 || reports[0].Class != classIgnored {
			t.Errorf("%v: got %+v, want one ignored entry", document, reports)
		}
	}
}

func TestClassifyKeysStopsAtTheTableTheSchemaNames(t *testing.T) {
	// flattenKeys folds everything below depth two into its table, so a key
	// below a table the schema names is judged by schema.Validate, not here.
	// This pins that boundary.
	document := map[string]any{
		"policy": map[string]any{"paths": map[string]any{"bogus": "x"}},
		"agent":  map[string]any{"models": map[string]any{"foo": map[string]any{"provider": "p"}}},
	}
	got := map[string]keyClass{}
	for _, r := range classifyKeys(document) {
		got[r.ID] = r.Class
	}
	want := map[string]keyClass{"policy.paths": classElsewhere, "agent.models": classElsewhere}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLegacyHintsAreIgnoredWithAHint(t *testing.T) {
	ids := []string{
		"llm.local", "layout.sources", "check.lanes", "area.wiki", "maintenance.merge_branch",
		"maintenance.watch", "maintenance.stale_after_days", "area.readonly",
	}
	if len(legacyHints) != len(ids) {
		t.Fatalf("legacyHints has %d entries, the test names %d", len(legacyHints), len(ids))
	}
	for _, id := range ids {
		section, key, _ := strings.Cut(id, ".")
		reports := classifyKeys(map[string]any{section: map[string]any{key: "x"}})
		if len(reports) != 1 || reports[0].ID != id || reports[0].Class != classIgnored || reports[0].Hint == "" {
			t.Errorf("%s: got %+v, want ignored with a hint", id, reports)
		}
	}
}

func TestClassifyKeysKnowsAnEmptyVerifyTable(t *testing.T) {
	// The schema names keys below `verify`, so the prefix loop alone knows
	// the empty table.
	reports := classifyKeys(map[string]any{"verify": map[string]any{}})
	if len(reports) != 1 || reports[0].ID != "verify" || reports[0].Class == classIgnored {
		t.Fatalf("got %+v, want verify not ignored", reports)
	}
}

func TestClassifyKeysIgnoresATopLevelScalarWithoutAHint(t *testing.T) {
	reports := classifyKeys(map[string]any{"top": int64(2)})
	if len(reports) != 1 || reports[0].ID != "top" || reports[0].Class != classIgnored || reports[0].Hint != "" {
		t.Fatalf("got %+v, want top ignored without a hint", reports)
	}
}

func runAreaCheck(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := areaCheck(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func writeManifest(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAreaCheckAcceptsAFullyReadManifest(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".loomux/config.toml", "[area]\nscope = \"x\"\n")
	code, out, _ := runAreaCheck(t, root)
	name := filepath.Join(".loomux", "config.toml")
	if code != 0 || !strings.Contains(out, name+": area.scope  read") || !strings.Contains(out, "chosen: "+name) ||
		strings.Contains(out, "shadowed") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckFailsOnAnIgnoredKeyAndNamesIt(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".brain.toml", "[area]\nscope = \"x\"\n\n[llm.local]\nenabled = false\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || !strings.Contains(out, "llm.local  ignored") || !strings.Contains(out, "[model]") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckRefusesAnOldNameWithoutScope(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".brain.toml", "[layout]\ninbox = \"00\"\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || !strings.Contains(out, ".brain.toml: refused:") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckTreatsPolicyOnlyLoomuxConfigAsFine(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".loomux/config.toml", "[commit]\nlanguage = \"en\"\n")
	writeManifest(t, root, ".brain.toml", "[area]\nscope = \"x\"\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 0 || !strings.Contains(out, "no [area], policy only") || !strings.Contains(out, "chosen: .brain.toml") || strings.Contains(out, "config.toml: shadowed") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckSaysWhichNameShadowsTheOther(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".loomux/config.toml", "[area]\nscope = \"x\"\n")
	writeManifest(t, root, ".brain.toml", "[area]\nscope = \"x\"\n")
	_, out, _ := runAreaCheck(t, root)
	if strings.Count(out, "shadowed") != 1 || !strings.Contains(out, ".brain.toml: shadowed") ||
		strings.Contains(out, filepath.Join(".loomux", "config.toml")+": shadowed") {
		t.Fatalf("out %q", out)
	}
}

func TestAreaCheckChoosesAnUltraBrainConfigurationAlone(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".ultra-brain/config.toml", "[area]\nscope = \"x\"\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 0 || !strings.Contains(out, "chosen: "+filepath.Join(".ultra-brain", "config.toml")) || strings.Contains(out, "shadowed") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckLetsUltraBrainShadowBrainToml(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".ultra-brain/config.toml", "[area]\nscope = \"x\"\n")
	writeManifest(t, root, ".brain.toml", "[area]\nscope = \"x\"\n")
	_, out, _ := runAreaCheck(t, root)
	if !strings.Contains(out, "chosen: "+filepath.Join(".ultra-brain", "config.toml")) ||
		strings.Count(out, "shadowed") != 1 || !strings.Contains(out, ".brain.toml: shadowed") {
		t.Fatalf("out %q", out)
	}
	// The manifests are reported in the order the reader tries them.
	ultra := strings.Index(out, filepath.Join(".ultra-brain", "config.toml")+": area.scope")
	brain := strings.Index(out, ".brain.toml: area.scope")
	if ultra < 0 || brain < 0 || ultra > brain {
		t.Fatalf("manifests out of order in %q", out)
	}
}

func TestAreaCheckCallsAShadowedDeclarationThatIsBrokenShadowed(t *testing.T) {
	// A file whose [area] the reader would refuse still declares an area.
	root := t.TempDir()
	writeManifest(t, root, ".ultra-brain/config.toml", "[area]\nscope = \"x\"\n")
	writeManifest(t, root, ".brain.toml", "[area]\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || !strings.Contains(out, ".brain.toml: shadowed") || !strings.Contains(out, ".brain.toml: refused:") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckRefusesALoomuxConfigWhoseAreaIsBroken(t *testing.T) {
	// Only a missing [area] makes .loomux/config.toml policy only; a broken
	// one is refused like any other manifest.
	root := t.TempDir()
	writeManifest(t, root, ".loomux/config.toml", "[area]\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || strings.Contains(out, "policy only") || !strings.Contains(out, "config.toml: refused:") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckRefusesAnEmptyManifest(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".brain.toml", "")
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || !strings.Contains(out, ".brain.toml: refused:") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

// The reader stops at the first regular file among the old names and refuses
// it without a scope, so a later .brain.toml with an [area] is neither chosen
// nor shadowed: this pins that behaviour of ReadAreaManifestUntilStage4.
func TestAreaCheckStopsAtAnOldNameWithoutAScope(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".ultra-brain/config.toml", "[commit]\nlanguage = \"en\"\n")
	writeManifest(t, root, ".brain.toml", "[area]\nscope = \"x\"\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || !strings.Contains(out, "chosen: none") || strings.Contains(out, "shadowed") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckReportsAFileThatIsNotTOML(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".brain.toml", "[area\n")
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || !strings.Contains(out, "not valid TOML") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckFailsWhenNoManifestIsThere(t *testing.T) {
	code, out, _ := runAreaCheck(t, t.TempDir())
	if code != 1 || !strings.Contains(out, "chosen: none") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckRefusesWrongArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"a", "b"}, {filepath.Join(t.TempDir(), "missing")}} {
		code, _, errOut := runAreaCheck(t, args...)
		if code != 2 || !strings.Contains(errOut, "usage: loomux area check <path>") {
			t.Fatalf("args %v: code %d, stderr %q", args, code, errOut)
		}
	}
}

func TestAreaCheckRefusesAFileAsThePath(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "plain.txt", "x")
	code, _, errOut := runAreaCheck(t, filepath.Join(root, "plain.txt"))
	if code != 2 || !strings.Contains(errOut, "usage: loomux area check <path>") {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
}

func TestAreaCheckWritesNothing(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".brain.toml", "[area]\nscope = \"x\"\n")
	before := snapshotTree(t, root)
	runAreaCheck(t, root)
	if after := snapshotTree(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("the tree changed: %v -> %v", before, after)
	}
}

// snapshotTree lists every path under root with its size and modification time.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	seen := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		seen[path] = fmt.Sprintf("%d %d", info.Size(), info.ModTime().UnixNano())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return seen
}

func TestAreaCheckIsReachableAsAnAreaSubcommand(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".loomux/config.toml", "[area]\nscope = \"x\"\n")
	var stdout, stderr bytes.Buffer
	if code := areaCommand([]string{"check", root}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("code %d, stderr %q", code, stderr.String())
	}
}

func TestAreaCheckSkipsADirectoryNamedLikeAManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".brain.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || strings.Contains(out, ".brain.toml:") || !strings.Contains(out, "chosen: none") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestAreaCheckReportsAManifestThatCannotBeRead(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, ".brain.toml", "[area]\nscope = \"x\"\n")
	original := readManifest
	t.Cleanup(func() { readManifest = original })
	readManifest = func(string) ([]byte, error) { return nil, errors.New("denied") }
	code, out, _ := runAreaCheck(t, root)
	if code != 1 || !strings.Contains(out, ".brain.toml: refused: denied") {
		t.Fatalf("code %d, out %q", code, out)
	}
}
