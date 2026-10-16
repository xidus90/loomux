package godotsuite

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func project(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestSuitesListsOnlyGdUnitSuites(t *testing.T) {
	root := project(t, map[string]string{
		"test/a_test.gd": "extends GdUnitTestSuite\n",
		"test/helper.gd": "extends Node\n",
		"project.godot":  "extends GdUnitTestSuite\n",
	})
	got, err := Suites(root)
	if err != nil || !reflect.DeepEqual(got, []string{"res://test/a_test.gd"}) {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestSuitesReadsAfterABomAndWithCRLF(t *testing.T) {
	root := project(t, map[string]string{"x/b.gd": "\xef\xbb\xbfextends GdUnitTestSuite\r\n"})
	got, _ := Suites(root)
	if !reflect.DeepEqual(got, []string{"res://x/b.gd"}) {
		t.Fatalf("got %v", got)
	}
}

func TestSuitesSkipsAddonsImportCacheAndCheckouts(t *testing.T) {
	const s = "extends GdUnitTestSuite\n"
	root := project(t, map[string]string{
		"addons/gdUnit4/a.gd": s, ".godot/b.gd": s, "wt/c.gd": s, "wt/.git": "gitdir: x",
	})
	got, err := Suites(root)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestSuitesSortsTheResult(t *testing.T) {
	const s = "extends GdUnitTestSuite\n"
	root := project(t, map[string]string{"b.gd": s, "a.gd": s})
	got, _ := Suites(root)
	if !reflect.DeepEqual(got, []string{"res://a.gd", "res://b.gd"}) {
		t.Fatalf("got %v", got)
	}
}

func TestWalkGDReportsAnUnreadableRootAndFile(t *testing.T) {
	if _, err := Suites(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("want an error for a missing root")
	}
	failRead(t)
	if _, err := Suites(project(t, map[string]string{"a.gd": ""})); err == nil {
		t.Fatal("want an error for an unreadable script")
	}
}

func failRead(t *testing.T) {
	t.Helper()
	old := readFile
	readFile = func(string) ([]byte, error) { return nil, os.ErrPermission }
	t.Cleanup(func() { readFile = old })
}

const cachePath = ".godot/global_script_class_cache.cfg"

func TestClassCacheFindingsNamesAMissingCache(t *testing.T) {
	root := project(t, nil)
	if got := ClassCacheFindings(root, "."); !reflect.DeepEqual(got, []string{".godot/global_script_class_cache.cfg does not exist; run loomux godot rescan"}) {
		t.Fatalf("got %q", got)
	}
	if got := ClassCacheFindings(root, "godot"); !reflect.DeepEqual(got, []string{".godot/global_script_class_cache.cfg does not exist; run loomux godot rescan --root godot"}) {
		t.Fatalf("got %q", got)
	}
}

func TestClassCacheFindingsListsMissingNames(t *testing.T) {
	root := project(t, map[string]string{
		cachePath:     "\"class\": &\"A\"\r\n",
		"a.gd":        "class_name A\n",
		"core/b.gd":   "class_name B\n",
		"addons/c.gd": "class_name C\n",
	})
	got := ClassCacheFindings(root, ".")
	if !reflect.DeepEqual(got, []string{"class_name not in the class cache: B (core/b.gd); run loomux godot rescan"}) {
		t.Fatalf("got %q", got)
	}
}

func TestClassCacheFindingsAcceptsTheFormWithoutAmpersand(t *testing.T) {
	root := project(t, map[string]string{cachePath: `"class": "A"`, "a.gd": "class_name A\n"})
	if got := ClassCacheFindings(root, "."); len(got) != 0 {
		t.Fatalf("got %q", got)
	}
}

func TestClassCacheFindingsAddsTheRootToTheRescanHint(t *testing.T) {
	root := project(t, map[string]string{cachePath: "", "a.gd": "class_name A\n"})
	got := ClassCacheFindings(root, "godot")
	if len(got) != 1 || !strings.HasSuffix(got[0], "; run loomux godot rescan --root godot") {
		t.Fatalf("got %q", got)
	}
}

func TestClassCacheFindingsReportsAnUnreadableCacheAndScript(t *testing.T) {
	root := project(t, map[string]string{cachePath + "/x": ""})
	got := ClassCacheFindings(root, ".")
	if len(got) != 1 || !strings.HasPrefix(got[0], "unreadable class cache:") {
		t.Fatalf("got %q", got)
	}
	root = project(t, map[string]string{cachePath: "", "a.gd": ""})
	failRead(t)
	got = ClassCacheFindings(root, ".")
	if len(got) != 1 || !strings.HasPrefix(got[0], "unreadable scripts:") {
		t.Fatalf("got %q", got)
	}
}

func TestNeedsDiskForAnAutoloadScript(t *testing.T) {
	cfg := "[autoload]\nClock=\"*res://autoload/clock.gd\"\n"
	root := project(t, map[string]string{"project.godot": cfg, "autoload/clock.gd": ""})
	if ok, err := NeedsDisk(root); err != nil || !ok {
		t.Fatalf("got %v, %v", ok, err)
	}
}

func TestNeedsDiskWithCRLFAndBom(t *testing.T) {
	cfg := "\xef\xbb\xbf[autoload]\r\nClock=\"*res://autoload/clock.gd\"\r\n"
	root := project(t, map[string]string{"project.godot": cfg, "autoload/clock.gd": ""})
	if ok, _ := NeedsDisk(root); !ok {
		t.Fatal("want true")
	}
}

func TestNeedsDiskHonoursIgnorePaths(t *testing.T) {
	cfg := "[autoload]\nClock=\"*res://autoload/clock.gd\"\n[nano_coverage]\ngeneral/ignore_paths=PackedStringArray(\"res://autoload/**\")\n"
	root := project(t, map[string]string{"project.godot": cfg, "autoload/clock.gd": ""})
	if ok, _ := NeedsDisk(root); ok {
		t.Fatal("want false")
	}
}

func TestNeedsDiskIgnoresAddonsUnlessAskedNotTo(t *testing.T) {
	const auto = "[autoload]\nA=\"res://addons/x/a.gd\"\n"
	if ok, _ := NeedsDisk(project(t, map[string]string{"addons/x/a.gd": "", "project.godot": auto})); ok {
		t.Fatal("addons are ignored by default")
	}
	cfg := auto + "[nano_coverage]\ngeneral/ignore_addons=false\n"
	if ok, _ := NeedsDisk(project(t, map[string]string{"addons/x/a.gd": "", "project.godot": cfg})); !ok {
		t.Fatal("ignore_addons=false covers addons")
	}
}

func TestNeedsDiskSkipsMissingFilesAndNonResPaths(t *testing.T) {
	cfg := "[autoload]\nA=\"*res://gone.gd\"\nC=\"plain\"\n"
	root := project(t, map[string]string{"project.godot": cfg})
	if ok, _ := NeedsDisk(root); ok {
		t.Fatal("want false")
	}
}

func TestNeedsDiskForASceneAutoload(t *testing.T) {
	cfg := "[autoload]\nMain=\"*res://main.tscn\"\n"
	root := project(t, map[string]string{"project.godot": cfg, "main.tscn": ""})
	if ok, err := NeedsDisk(root); err != nil || !ok {
		t.Fatalf("got %v, %v", ok, err)
	}
}

func TestNeedsDiskWithoutAutoload(t *testing.T) {
	root := project(t, map[string]string{"project.godot": "[application]\nx=1\n"})
	if ok, err := NeedsDisk(root); err != nil || ok {
		t.Fatalf("got %v, %v", ok, err)
	}
}

func TestNeedsDiskReportsUnreadableAndBrokenSettings(t *testing.T) {
	if _, err := NeedsDisk(project(t, nil)); err == nil {
		t.Fatal("want an error for a missing project.godot")
	}
	if _, err := NeedsDisk(project(t, map[string]string{"project.godot": "[a\n"})); err == nil {
		t.Fatal("want an error for a broken project.godot")
	}
}

func TestNeedsDiskReportsABrokenIgnorePattern(t *testing.T) {
	cfg := "[autoload]\nA=\"res://a.gd\"\n[nano_coverage]\ngeneral/ignore_paths=PackedStringArray(\"[a\")\n"
	root := project(t, map[string]string{"project.godot": cfg, "a.gd": ""})
	if _, err := NeedsDisk(root); err == nil {
		t.Fatal("want an error")
	}
}

const hookCfg = "[gdunit4]\r\nhooks/session_hooks=Dictionary[String, bool]({\r\n\"res://h.gd\": true\r\n})\r\n"

func TestHookConflictNamesACoverageHook(t *testing.T) {
	for _, word := range []string{"CoverageApi", "NanoCoverage"} {
		root := project(t, map[string]string{"project.godot": "\xef\xbb\xbf" + hookCfg, "h.gd": "extends X\n# " + word})
		want := "project.godot registers a coverage session hook, res://h.gd; remove it from [gdunit4] hooks/session_hooks and set [nano_coverage] integrations/gdunit4=false, or the Godot editor registers it again"
		if got, err := HookConflict(root); err != nil || got != want {
			t.Fatalf("got %q, %v", got, err)
		}
	}
}

func TestHookConflictIgnoresOtherHooks(t *testing.T) {
	root := project(t, map[string]string{"project.godot": hookCfg, "h.gd": "extends X\n"})
	if got, err := HookConflict(root); err != nil || got != "" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestHookConflictSkipsAMissingHookFileAndPicksTheFirstHit(t *testing.T) {
	cfg := "[gdunit4]\nhooks/session_hooks=Dictionary[String, bool]({\n\"res://gone.gd\": true,\n\"res://h.gd\": true\n})\n"
	root := project(t, map[string]string{"project.godot": cfg, "h.gd": "NanoCoverage"})
	if got, _ := HookConflict(root); !strings.Contains(got, "res://h.gd;") {
		t.Fatalf("got %q", got)
	}
}

func TestHookConflictWithoutTheKeyOrSection(t *testing.T) {
	for _, cfg := range []string{"[application]\nx=1\n", "[gdunit4]\nother=1\n"} {
		root := project(t, map[string]string{"project.godot": cfg})
		if got, err := HookConflict(root); err != nil || got != "" {
			t.Fatalf("got %q, %v", got, err)
		}
	}
}

func TestHookConflictStopsAtTheClosingParenthesis(t *testing.T) {
	cfg := "[gdunit4]\nhooks/session_hooks=Dictionary[String, bool]({\n})\nother=\"res://h.gd\"\n"
	root := project(t, map[string]string{"project.godot": cfg, "h.gd": "NanoCoverage"})
	if got, _ := HookConflict(root); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestHookConflictToleratesAnUnterminatedValue(t *testing.T) {
	cfg := "[gdunit4]\nhooks/session_hooks=Dictionary[String, bool]({\n\"res://h.gd\": true\n"
	root := project(t, map[string]string{"project.godot": cfg, "h.gd": "NanoCoverage"})
	if got, _ := HookConflict(root); got == "" {
		t.Fatal("want a finding")
	}
}

func TestHookConflictReportsAMissingProjectFile(t *testing.T) {
	if _, err := HookConflict(project(t, nil)); err == nil {
		t.Fatal("want an error")
	}
}

func TestInstrumentedNamesFilesWithTheMarkerOnLineOne(t *testing.T) {
	root := project(t, map[string]string{
		"a.gd": "# __NANO_COVERAGE_INSTRUMENTED__\nextends Node\n",
		"b.gd": "extends Node\n# __NANO_COVERAGE_INSTRUMENTED__\n",
		"c.gd": "\xef\xbb\xbf# __NANO_COVERAGE_INSTRUMENTED__\r\nx\r\n",
		"d.gd": "# __NANO_COVERAGE_INSTRUMENTED__xyz\n",
	})
	got, err := Instrumented(root)
	if err != nil || !reflect.DeepEqual(got, []string{"a.gd", "c.gd"}) {
		t.Fatalf("got %v, %v", got, err)
	}
}

// Godot allows one class_name per script; a later line is text in a string.
func TestClassCacheFindingsCountsOnlyTheFirstClassNameOfAFile(t *testing.T) {
	root := project(t, map[string]string{
		cachePath: "\"class\": &\"A\"\r\n",
		"a.gd":    "class_name A\nvar s = \"\"\"\nclass_name Fake\n\"\"\"\n",
	})
	if got := ClassCacheFindings(root, "."); len(got) != 0 {
		t.Fatalf("got %q", got)
	}
}
