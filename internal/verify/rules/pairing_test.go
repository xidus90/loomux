package rules

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

const godotWith = "config_version=5\r\n\r\n[gdunit4]\r\n\r\n"

func gdFiles(settings string, extra ...File) []File {
	return append([]File{
		{Rel: "godot/project.godot", Content: []byte(godotWith + settings)},
		{Rel: "godot/addons/gdUnit4/plugin.cfg", Content: []byte("name=\"gdUnit4\"")},
	}, extra...)
}

func pair(t *testing.T, tests bool, files []File) ([]Finding, []string, error) {
	t.Helper()
	rels := make([]string, len(files))
	for i, f := range files {
		rels[i] = f.Rel
	}
	m, errs := Assign(config.Rules{Layers: []config.Layer{{Name: "core", Paths: []string{"**"}, Tests: tests, Checked: true}}}, rels)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	return Pairing(files, m)
}

func texts(fs []Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.String()) // as printed: path once, then the text
	}
	return out
}

func TestPairing(t *testing.T) {
	src := File{Rel: "godot/core/seeded_rng.gd"}
	const missing = "godot/core/seeded_rng.gd has no test module; expected "
	for _, tc := range []struct {
		name  string
		files []File
		want  []string
	}{
		{"missing", gdFiles("", src), []string{missing + "godot/test/core/seeded_rng_test.gd"}},
		{"present", gdFiles("", src, File{Rel: "godot/test/core/seeded_rng_test.gd"}), nil},
		{"folder with CRLF", gdFiles("settings/test/test_lookup_folder=\"spec\"\r\n", src),
			[]string{missing + "godot/spec/core/seeded_rng_test.gd"}},
		{"empty folder is beside", gdFiles("settings/test/test_lookup_folder=\"\"\r\n", src),
			[]string{missing + "godot/core/seeded_rng_test.gd"}},
		{"slash folder is beside", gdFiles("settings/test/test_lookup_folder=\"/\"\n", src),
			[]string{missing + "godot/core/seeded_rng_test.gd"}},
		{"pascal name", gdFiles("", File{Rel: "godot/core/Foo.gd"}),
			[]string{"godot/core/Foo.gd has no test module; expected godot/test/core/FooTest.gd"}},
		{"snake forced", gdFiles("settings/test/test_suite_naming_convention=1\n", File{Rel: "godot/core/Foo.gd"}),
			[]string{"godot/core/Foo.gd has no test module; expected godot/test/core/foo_test.gd"}},
		{"pascal forced", gdFiles("settings/test/test_suite_naming_convention=2\n", src),
			[]string{missing + "godot/test/core/SeededRngTest.gd"}},
		{"unknown convention", gdFiles("settings/test/test_suite_naming_convention=7\n", src),
			[]string{missing + "godot/test/core/-<Unexpected>-.gd"}},
		{"test folder files are no sources", gdFiles("", File{Rel: "godot/test/core/x_test.gd"}, File{Rel: "godot/test/helper.gd"}), nil},
		{"test suffix beside", gdFiles("settings/test/test_lookup_folder=\"\"\n", File{Rel: "godot/core/x_test.gd"}, File{Rel: "godot/core/YTest.gd"}), nil},
		{"suffix is no test under a test folder", gdFiles("", File{Rel: "godot/core/x_test.gd"}), []string{"godot/core/x_test.gd has no test module; expected godot/test/core/x_test_test.gd"}},
		{"project at the root", []File{
			{Rel: "project.godot", Content: []byte(godotWith)},
			{Rel: "addons/gdUnit4/plugin.cfg", Content: []byte("gdUnit4")},
			{Rel: "core/aa.gd"},
		}, []string{"core/aa.gd has no test module; expected test/core/aa_test.gd"}},
		{"go", []File{{Rel: "a.go"}, {Rel: "a_test.go"}, {Rel: "b.go"}},
			[]string{"b.go has no test module; expected b_test.go"}},
	} {
		got, errs, err := pair(t, true, tc.files)
		if err != nil || len(errs) != 0 || !slices.Equal(texts(got), tc.want) {
			t.Errorf("%s: got %q %q %v, want %q", tc.name, texts(got), errs, err, tc.want)
		}
	}
}

func TestPairingSkipsLayersWithoutTests(t *testing.T) {
	got, errs, err := pair(t, false, append(gdFiles(""), File{Rel: "godot/core/a.gd"}, File{Rel: "x.py"}, File{Rel: "readme.md"}))
	if len(got) != 0 || len(errs) != 0 || err != nil {
		t.Fatal(got, errs, err)
	}
}

func TestPairingFindingsAreSorted(t *testing.T) {
	got, _, _ := pair(t, true, []File{{Rel: "z.go"}, {Rel: "a.go"}})
	if len(got) != 2 || got[0].Path != "a.go" || got[1].Path != "z.go" {
		t.Fatal(got)
	}
}

func TestPairingMapErrors(t *testing.T) {
	_, errs, err := pair(t, true, []File{{Rel: "a.py"}, {Rel: "b.py"}})
	want := []string{"layer core: tests = true, but no test profile knows .py files"}
	if err != nil || !slices.Equal(errs, want) {
		t.Fatalf("%q %v", errs, err)
	}

	// No addon, a plugin.cfg of another tool, and a project outside the addon's directory.
	for name, files := range map[string][]File{
		"no addon":      {{Rel: "godot/project.godot"}, {Rel: "godot/core/a.gd"}},
		"other tool":    {{Rel: "godot/project.godot"}, {Rel: "godot/addons/x/plugin.cfg", Content: []byte("name=\"Nano\"")}, {Rel: "godot/core/a.gd"}},
		"other project": {{Rel: "project.godot"}, {Rel: "godot/addons/gdUnit4/plugin.cfg", Content: []byte("gdUnit4")}, {Rel: "other/project.godot"}, {Rel: "other/core/a.gd"}},
	} {
		got, errs, err := pair(t, true, files)
		if len(got) != 0 || err != nil || len(errs) != 1 ||
			!strings.HasPrefix(errs[0], "layer core: tests = true, but .gd files need a plugin.cfg mentioning gdUnit4 (none found under ") {
			t.Errorf("%s: %q %q %v", name, got, errs, err)
		}
	}
	_, errs, _ = pair(t, true, []File{{Rel: "a.gd"}})
	if !slices.Equal(errs, []string{"layer core: tests = true, but .gd files need a plugin.cfg mentioning gdUnit4 (none found under .)"}) {
		t.Fatal(errs)
	}
}

func TestPairingReadErrorNamesTheFile(t *testing.T) {
	files := []File{{Rel: "godot/project.godot", Content: []byte("[gdunit4\n")}, {Rel: "godot/addons/gdUnit4/plugin.cfg", Content: []byte("gdUnit4")}, {Rel: "godot/core/a.gd"}}
	_, _, err := pair(t, true, files)
	if err == nil || !strings.HasPrefix(err.Error(), "godot/project.godot: ") {
		t.Fatal(err)
	}
}

func TestParseTestsRejects(t *testing.T) {
	for _, text := range []string{
		"[profile",
		"[profile.x]\nextension = \".x\"\ntest_suffixes = [\"_t.x\"]\nmapping = \"nope\"\n",
		"[profile.x]\ntest_suffixes = [\"_t.x\"]\nmapping = \"beside\"\n",
		"[profile.x]\nextension = \".x\"\nmapping = \"beside\"\n",
		"[profile.x]\nextension = \".x\"\ntest_suffixes = [\"_t.x\"]\nmapping = \"beside\"\ndetect = \"plugin.cfg:y\"\n",
		"[profile.x]\nextension = \".x\"\ntest_suffixes = [\"_t.x\"]\nmapping = \"beside\"\n[profile.y]\nextension = \".x\"\ntest_suffixes = [\"T.x\"]\nmapping = \"gdunit4\"\n",
	} {
		if _, err := parseTests(text); err == nil {
			t.Errorf("accepted %q", text)
		}
	}
	ps, err := testProfiles()
	if err != nil || len(ps) != 2 {
		t.Fatal(ps, err)
	}
}

func TestPairingBadProfilesAreAnError(t *testing.T) {
	old := testProfiles
	defer func() { testProfiles = old }()
	testProfiles = func() (map[string]testProfile, error) { return nil, os.ErrInvalid }
	if _, _, err := Pairing(nil, nil); err == nil {
		t.Fatal("no error")
	}
}

// The three worked examples of the gdUnit4 mapping, by hand from
// resolve_test_suite_path, and the cases around them.
func TestGdunit4Path(t *testing.T) {
	for _, tc := range []struct {
		src, folder string
		naming      int
		want        string
	}{
		{"core/seeded_rng.gd", "test", 0, "test/core/seeded_rng_test.gd"},
		{"ui/main_menu/main_menu.gd", "test", 0, "test/ui/main_menu_test/main_menu_test.gd"},
		{"addons/p/x/foo.gd", "test", 0, "addons/p/test/x/foo_test.gd"},
		{"core/src/foo.gd", "test", 0, "core/test/foo_test.gd"},
		{"foo.gd", "test", 0, "test/foo_test.gd"},
		{"core/a.gd", "test", 0, "test/core/atest.gd"}, // to_snake_case skips the underscore this early
		{"core/Foo.gd", "spec", 0, "spec/core/FooTest.gd"},
		{"core/foo.gd", "", 0, "core/foo_test.gd"},
		{"core/foo.gd", "/", 0, "core/foo_test.gd"},
	} {
		if got := gdunit4Path(tc.src, tc.folder, tc.naming); got != tc.want {
			t.Errorf("%s %q: got %s, want %s", tc.src, tc.folder, got, tc.want)
		}
	}
}

func TestNameConversions(t *testing.T) {
	for in, want := range map[string]string{"FooTest": "foo_test", "fooTest": "foo_test", "ab": "ab", "AbcD": "abc_d", "A": "a", "Ab": "ab"} {
		if got := toSnakeCase(in); got != want {
			t.Errorf("snake %s: %s, want %s", in, got, want)
		}
	}
	for in, want := range map[string]string{
		"FooTest": "FooTest", "foo_test": "FooTest", "MainMenuTest": "MainMenuTest", "HTTPServerTest": "HttpServerTest",
		"a1bTest": "A1bTest", "Foo2Test": "Foo2Test", "foo22x": "Foo22x", "ab12": "Ab12", "_x": "X",
	} {
		if got := toPascalCase(in); got != want {
			t.Errorf("pascal %s: %s, want %s", in, got, want)
		}
	}
	for in, want := range map[string]bool{"foo_bar": true, "foo": true, "Foo": false, "foo2": false, "_": true} {
		if got := isSnakeCase(in); got != want {
			t.Errorf("isSnake %s: %v", in, got)
		}
	}
	if got := suiteName("x", 9); got != "-<Unexpected>-" {
		t.Fatal(got)
	}
}

// A profile without a fixture project is not tested: every entry in
// tests.toml brings testdata/tests/<name>/, a tree whose sources all lie in
// one layer with tests = true, and want.txt lists the findings it must give.
func TestEveryTestProfileHasAFixtureThatPairs(t *testing.T) {
	profiles, err := testProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for name := range profiles {
		dir := filepath.Join("testdata", "tests", name)
		var files []File
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || d.Name() == "want.txt" {
				return err
			}
			c, err := os.ReadFile(p)
			rel, _ := filepath.Rel(dir, p)
			files = append(files, File{filepath.ToSlash(rel), c})
			return err
		})
		if err != nil || len(files) == 0 {
			t.Errorf("%s: no fixture project at %s (%v)", name, dir, err)
			continue
		}
		got, errs, err := pair(t, true, files)
		if err != nil || len(errs) != 0 {
			t.Errorf("%s: %q %v", name, errs, err)
		}
		w, _ := os.ReadFile(filepath.Join(dir, "want.txt"))
		var want []string
		if s := strings.TrimSpace(string(w)); s != "" {
			want = strings.Split(s, "\n")
		}
		if len(want) == 0 || !slices.Equal(texts(got), want) {
			t.Errorf("%s: findings %q, want %q (and at least one)", name, texts(got), want)
		}
	}
}
