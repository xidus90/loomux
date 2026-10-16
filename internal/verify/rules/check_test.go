package rules

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/verify/linecov"
)

// checkRepo commits nothing; it writes the manifest and the files and adds all.
func checkRepo(t *testing.T, manifest string, files map[string]string) string {
	t.Helper()
	root := repo(t)
	if manifest != "" {
		write(t, root, ".loomux/config.toml", manifest)
	}
	for rel, body := range files {
		write(t, root, rel, body)
	}
	git(t, root, "add", "-A")
	return root
}

func paths(fs []Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Path)
	}
	return out
}

func TestCheckWithoutRulesDoesNothing(t *testing.T) {
	res, err := Check(Options{Root: checkRepo(t, "", nil)})
	if err != nil || len(res.Findings) != 0 || len(res.MapErrors) != 0 ||
		!slices.Equal(res.Notes, []string{"no [rules] in .loomux/config.toml"}) {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestCheckReportsAnUnreadableTableAsAMapError(t *testing.T) {
	res, err := Check(Options{Root: checkRepo(t, "[rules]\nnope = 1\n", nil)})
	if err != nil || len(res.MapErrors) != 1 || !strings.Contains(res.MapErrors[0], `does not know "nope"`) {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestCheckReportsAGlobThatMatchesNothing(t *testing.T) {
	root := checkRepo(t, "[rules.layers.a]\npaths = [\"a/**\"]\n", map[string]string{"x.go": "package x\n"})
	res, err := Check(Options{Root: root})
	if err != nil || len(res.Findings) != 0 || !slices.Equal(res.MapErrors, []string{"layer a: a/** matches no file"}) {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestCheckReportsAFileListError(t *testing.T) {
	root := t.TempDir()
	write(t, root, ".loomux/config.toml", "[rules]\n")
	if _, err := Check(Options{Root: root}); err == nil {
		t.Fatal("no error")
	}
}

const orderManifest = `
[rules.layers.a]
paths = ["a/**"]
may_use = []

[rules.layers.g]
paths = ["g/**"]
may_use = []
`

func orderFiles() map[string]string {
	return map[string]string{
		"go.mod":        "module m\n\ngo 1.22\n",
		"project.godot": "[nano_coverage]\n\ngeneral/x=1\n",
		"a/a.go":        "package a\n\nimport _ \"m/b\"\n",
		"b/b.go":        "package b\n",
		"z/z.go":        "package z\n",
		"g/i.gd":        "# __NANO_COVERAGE_INSTRUMENTED__\nextends Node\n",
	}
}

func TestCheckOrdersInstrumentationEdgesAndUnlayered(t *testing.T) {
	manifest := orderManifest + "\n[rules.layers.b]\npaths = [\"b/**\"]\nmay_use = []\n"
	root := checkRepo(t, manifest, orderFiles())

	res, err := Check(Options{Root: root, Cached: true})
	want := []string{"g/i.gd", "a/a.go", "z/z.go"}
	if err != nil || len(res.MapErrors) != 0 || !slices.Equal(paths(res.Findings), want) {
		t.Fatalf("%v %+v %v", paths(res.Findings), res, err)
	}

	// Instrumentation is a matter of the index only.
	res, err = Check(Options{Root: root})
	if err != nil || !slices.Equal(paths(res.Findings), want[1:]) {
		t.Fatalf("%v %v", paths(res.Findings), err)
	}
}

func TestCheckReportsPairingMapErrors(t *testing.T) {
	root := checkRepo(t, "[rules.layers.a]\npaths = [\"**\"]\ntests = true\n", map[string]string{"a.py": "x = 1\n"})
	res, err := Check(Options{Root: root})
	if err != nil || len(res.Findings) != 0 ||
		!slices.Equal(res.MapErrors, []string{"layer a: tests = true, but no test profile knows .py files"}) {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestCheckReportsAnExtractionError(t *testing.T) {
	root := checkRepo(t, "[rules.layers.a]\npaths = [\"**\"]\n", map[string]string{"a.go": "package"})
	if _, err := Check(Options{Root: root}); err == nil || !strings.HasPrefix(err.Error(), "a.go: ") {
		t.Fatal(err)
	}
}

func TestCheckReportsAPairingError(t *testing.T) {
	old := testProfiles
	defer func() { testProfiles = old }()
	testProfiles = func() (map[string]testProfile, error) { return nil, errors.New("boom") }
	root := checkRepo(t, "[rules.layers.a]\npaths = [\"**\"]\n", map[string]string{"a.go": "package a\n"})
	if _, err := Check(Options{Root: root}); err == nil {
		t.Fatal("no error")
	}
}

func TestCheckReportsAnInstrumentationError(t *testing.T) {
	old := instrumentationProfiles
	defer func() { instrumentationProfiles = old }()
	instrumentationProfiles = func() ([]linecov.Profile, error) { return nil, errors.New("boom") }
	root := checkRepo(t, "[rules.layers.a]\npaths = [\"**\"]\n", map[string]string{"a.go": "package a\n"})
	if _, err := Check(Options{Root: root, Cached: true}); err == nil {
		t.Fatal("no error")
	}
}

func TestCheckKeepsInstrumentationWhenTheMapIsBroken(t *testing.T) {
	manifest := "[rules.layers.g]\npaths = [\"g/**\"]\n[rules.layers.n]\npaths = [\"n/**\"]\n"
	files := map[string]string{
		"project.godot": "[nano_coverage]\n\ngeneral/x=1\n",
		"g/i.gd":        "# __NANO_COVERAGE_INSTRUMENTED__\nextends Node\n",
	}
	res, err := Check(Options{Root: checkRepo(t, manifest, files), Cached: true})
	if err != nil || !slices.Equal(res.MapErrors, []string{"layer n: n/** matches no file"}) ||
		!slices.Equal(paths(res.Findings), []string{"g/i.gd"}) {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestCheckKeepsInstrumentationWhenPairingFindsMapErrors(t *testing.T) {
	manifest := "[rules.layers.a]\npaths = [\"**\"]\ntests = true\n"
	files := map[string]string{
		"project.godot": "[nano_coverage]\n\ngeneral/x=1\n",
		"a.gd":          "# __NANO_COVERAGE_INSTRUMENTED__\nextends Node\n",
	}
	res, err := Check(Options{Root: checkRepo(t, manifest, files), Cached: true})
	if err != nil || len(res.MapErrors) != 1 || !slices.Equal(paths(res.Findings), []string{"a.gd"}) {
		t.Fatalf("%+v %v", res, err)
	}
}
