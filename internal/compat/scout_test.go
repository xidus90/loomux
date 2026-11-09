package compat

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// fakeUp answers from a map keyed by source; a missing source is an error.
// A version that is not plain digits stands for a pre-release. The key
// "eol:<runtime>" lists the runtime's versions for EOL, a trailing "!" marking
// one past end of support.
type fakeUp map[string][]string

func (f fakeUp) EOL(runtime string) (map[string]bool, error) {
	vs, ok := f["eol:"+runtime]
	if !ok {
		return nil, errors.New("no end-of-support answer")
	}
	out := map[string]bool{}
	for _, v := range vs {
		name, past := strings.CutSuffix(v, "!")
		out[name] = past
	}
	return out, nil
}

func (f fakeUp) Releases(source string) ([]Release, error) {
	r, ok := f[source]
	if !ok {
		return nil, errors.New("no answer")
	}
	out := make([]Release, len(r))
	for i, v := range r {
		_, plain := parseVer(v)
		out[i] = Release{Version: v, Pre: !plain}
	}
	return out, nil
}

func gdunit(godot string) Entry {
	return Entry{Stack: "gdscript", Runtime: "godot", Version: godot, Fixture: "godot4-gdunit4",
		Kinds:  []string{"lint", "test"},
		Addons: []Addon{{"gdUnit4", "6.2.2"}, {"NanoCoverage", "0.1.1"}},
		Tools:  map[string]string{"gdtoolkit": "4.*"}}
}

func godot3() Entry {
	return Entry{Stack: "gdscript", Runtime: "godot", Version: "3.6", Fixture: "godot3-gut",
		Kinds: []string{"lint"}, Addons: []Addon{{"GUT", "7.4.3"}}, Tools: map[string]string{"gdtoolkit": "3.*"}}
}

func py(v string) Entry {
	return Entry{Stack: "python", Runtime: "python", Version: v, Fixture: "python-mypy", Kinds: []string{"lint"},
		Tools: map[string]string{"ruff": "0.17.*", "pytest": "9.1.*"}}
}

// calm is an upstream that holds nothing newer than the entries above.
func calm() fakeUp {
	return fakeUp{
		godotSource:  {"4.7", "4.7.1", "3.6.2", "4.5"},
		pythonSource: {"3.15.0", "3.14.2"},
		goSource:     {"1.27.2"},
		"eol:godot":  {"3.6", "4.5", "4.7", "4.9", "9.9"},
		"eol:python": {"3.11", "3.14", "3.15"},
		"eol:go":     {"1.26", "1.27", "1.28"},
		"https://api.github.com/repos/godot-gdunit-labs/gdUnit4/releases?per_page=100":      {"6.2.2", "6.2.1"},
		"https://api.github.com/repos/IgorBayerl/nano-coverage-godot/releases?per_page=100": {"0.1.1"},
		"https://api.github.com/repos/bitwes/Gut/releases?per_page=100":                     {"7.4.3"},
		"https://pypi.org/pypi/gdtoolkit/json":                                              {"3.6.0", "4.5.0"},
		"https://pypi.org/pypi/ruff/json":                                                   {"0.17.4"},
		"https://pypi.org/pypi/pytest/json":                                                 {"9.1.9"},
	}
}

func scout(t *testing.T, up fakeUp, entries ...Entry) []Finding {
	t.Helper()
	got, err := Scout(entries, up)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestScoutNothingNew(t *testing.T) {
	if got := scout(t, calm(), gdunit("4.5"), gdunit("4.7"), godot3(), py("3.15")); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

func TestScoutNewMinorHasProbeAndTOML(t *testing.T) {
	up := calm()
	up[godotSource] = []string{"4.8-rc1", "4.8", "4.8.1", "4.7", "3.7"}
	got := scout(t, up, gdunit("4.5"), gdunit("4.7"), godot3())
	// Godot 3 gets no finding for a 4.x; its own line has 3.7 (a new minor).
	if len(got) != 2 {
		t.Fatalf("findings %+v", got)
	}
	f := got[0]
	if f.Line != "godot4-gdunit4/godot" || f.Version != "4.8" || IssueTitle(f) != "compat: godot4-gdunit4/godot 4.8 is out" {
		t.Errorf("finding %+v", f)
	}
	if f.Probe.Version != "4.8" || f.Probe.Fixture != "godot4-gdunit4" || len(f.Probe.Addons) != 2 {
		t.Errorf("probe %+v", f.Probe)
	}
	if got[1].Line != "godot3-gut/godot" || got[1].Version != "3.7" {
		t.Errorf("godot 3 finding %+v", got[1])
	}
	want := "[[entry]]\nstack = \"gdscript\"\nruntime = \"godot\"\nversion = \"4.8\"\nfixture = \"godot4-gdunit4\"\n" +
		"kinds = [\"lint\", \"test\"]\naddons = [{ name = \"gdUnit4\", version = \"6.2.2\" }, { name = \"NanoCoverage\", version = \"0.1.1\" }]\n" +
		"tools = { gdtoolkit = \"4.*\" }\n"
	if f.TOML != want {
		t.Errorf("toml\n%s", f.TOML)
	}
	// The block is the form compat.toml reads back.
	es, err := parse([]byte(f.TOML), "probe")
	if err != nil || es[0].Key() != f.Probe.Key() {
		t.Errorf("parse: %v %+v", err, es)
	}
}

func TestScoutPatchNeverCounts(t *testing.T) {
	up := calm()
	up[godotSource] = []string{"4.7.5"}
	up[pythonSource] = []string{"3.15.9"}
	if got := scout(t, up, gdunit("4.7"), py("3.15")); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

func goE(v string) Entry {
	return Entry{Stack: "go", Runtime: "go", Version: v, Fixture: "go-module", Kinds: []string{"lint"}}
}

func TestScoutFindsNewestPreReleaseOfNextVersion(t *testing.T) {
	up := calm()
	up[goSource] = []string{"1.28rc1", "1.28rc2", "1.27.3", "1.26.10"}
	got := scout(t, up, goE("1.26"), goE("1.27"))
	if len(got) != 1 {
		t.Fatalf("findings %+v", got)
	}
	f := got[0]
	if f.Kind != "pre" || f.Line != "go-module/go" || f.Version != "1.28rc2" ||
		f.Probe.Version != "1.28" || f.Probe.Pre != "1.28rc2" || f.TOML != "" {
		t.Errorf("finding %+v probe %+v", f, f.Probe)
	}
	if f.Title != "compat: go-module/go 1.28rc2 is out" || f.Probe.Key() != "go/go-1.28rc2/go-module" {
		t.Errorf("title %q key %q", f.Title, f.Probe.Key())
	}
}

func TestScoutIgnoresPreReleaseOfListedVersion(t *testing.T) {
	up := calm()
	up[goSource] = []string{"1.27rc3", "1.27.3"}
	if got := scout(t, up, goE("1.27")); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

func TestScoutPrefersStableOverPreRelease(t *testing.T) {
	up := calm()
	up[goSource] = []string{"1.28", "1.28rc2", "1.27.3"}
	got := scout(t, up, goE("1.27"))
	if len(got) != 1 || got[0].Kind != "new" || got[0].Version != "1.28" || got[0].Probe.Pre != "" {
		t.Fatalf("findings %+v", got)
	}
}

func TestScoutPreReleaseOfEveryRuntime(t *testing.T) {
	up := calm()
	up[pythonSource] = []string{"3.15.0", "3.16.0a1"}
	up[godotSource] = []string{"4.7.2", "4.8-dev7", "4.8-beta2"}
	got := scout(t, up, py("3.15"), gdunit("4.7"))
	pres := map[string]string{}
	for _, f := range got {
		if f.Kind == "pre" {
			pres[f.Line] = f.Probe.Pre
		}
	}
	if pres["python-mypy/python"] != "3.16.0a1" || pres["godot4-gdunit4/godot"] != "4.8-beta2" || len(got) != 2 {
		t.Fatalf("findings %+v", got)
	}
}

func TestScoutFindsPreReleaseOfNextMajor(t *testing.T) {
	up := calm()
	up[godotSource] = []string{"4.7.1", "5.0-beta1"}
	got := scout(t, up, gdunit("4.7"))
	if len(got) != 1 || got[0].Kind != "pre" || got[0].Line != "godot4-gdunit4/godot" ||
		got[0].Probe.Version != "5.0" || got[0].Probe.Pre != "5.0-beta1" {
		t.Fatalf("findings %+v", got)
	}
}

// A next major belongs to the line with the highest major only, as for a
// stable release: Godot 3 gets no finding for a 4.x candidate.
func TestScoutPreReleaseOfNextMajorOnlyForTopLine(t *testing.T) {
	up := calm()
	up[godotSource] = []string{"4.7.1", "3.6.2", "4.8-beta2"}
	got := scout(t, up, godot3(), gdunit("4.7"))
	if len(got) != 1 || got[0].Kind != "pre" || got[0].Line != "godot4-gdunit4/godot" || got[0].Version != "4.8-beta2" {
		t.Fatalf("findings %+v", got)
	}
}

func TestScoutPreReleaseOnlyForRuntimes(t *testing.T) {
	up := calm()
	up["https://pypi.org/pypi/ruff/json"] = []string{"0.17.4", "0.18.0rc1"}
	up["https://api.github.com/repos/godot-gdunit-labs/gdUnit4/releases?per_page=100"] = []string{"6.2.2", "6.3.0-rc1"}
	if got := scout(t, up, gdunit("4.7"), py("3.15")); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

func TestScoutReportsEndOfSupport(t *testing.T) {
	up := calm()
	up["eol:python"] = []string{"3.10!", "3.11", "3.15"}
	got := scout(t, up, py("3.10"), py("3.11"), py("3.15"))
	if len(got) != 1 {
		t.Fatalf("findings %+v", got)
	}
	f := got[0]
	if f.Kind != "eol" || f.Line != "python" || f.Version != "3.10" || f.Probe != nil || f.TOML != "" ||
		f.Title != "compat: python 3.10 is past end of support" {
		t.Errorf("finding %+v", f)
	}
}

func TestScoutEOLIsPerRuntimeNotPerFixture(t *testing.T) {
	up := calm()
	up["eol:godot"] = []string{"4.5!", "4.7"}
	g := gdunit("4.5")
	g.Fixture = "godot4-plain"
	// The second fixture's line also gets a "new" finding for 4.7; only the
	// end of support is counted here.
	var eol []Finding
	for _, f := range scout(t, up, gdunit("4.5"), g, gdunit("4.7")) {
		if f.Kind == "eol" {
			eol = append(eol, f)
		}
	}
	if len(eol) != 1 || eol[0].Line != "godot" || eol[0].Version != "4.5" {
		t.Fatalf("findings %+v", eol)
	}
}

func TestScoutFailsWhenEOLUnknown(t *testing.T) {
	up := calm()
	delete(up, "eol:python")
	if _, err := Scout([]Entry{py("3.15")}, up); err == nil || !strings.Contains(err.Error(), "python") {
		t.Errorf("no answer: %v", err)
	}
	up = calm()
	up["eol:python"] = []string{"3.15"}
	if _, err := Scout([]Entry{py("3.14")}, up); err == nil || !strings.Contains(err.Error(), "python 3.14") {
		t.Errorf("unknown version: %v", err)
	}
}

func TestSupersededPreRelease(t *testing.T) {
	rc1 := "compat: go-module/go 1.28rc1 is out"
	rc2 := Finding{Kind: "pre", Line: "go-module/go", Version: "1.28rc2"}
	rc2.Title = IssueTitle(rc2)
	for _, c := range []struct {
		title    string
		findings []Finding
		want     bool
	}{
		{rc1, []Finding{rc2}, true},
		{rc1, nil, true},
		{rc2.Title, []Finding{rc2}, false},
		{"compat: go-module/go 1.28 is out", nil, false},
		{"compat: python 3.10 is past end of support", nil, false},
		{"something else", nil, false},
	} {
		if got := Superseded(c.title, c.findings); got != c.want {
			t.Errorf("%q with %d findings = %v, want %v", c.title, len(c.findings), got, c.want)
		}
	}
}

func TestResolvedEOL(t *testing.T) {
	title := "compat: python 3.10 is past end of support"
	if Resolved([]Entry{py("3.10"), py("3.11")}, title) {
		t.Error("resolved while 3.10 is listed")
	}
	if !Resolved([]Entry{py("3.11")}, title) {
		t.Error("open after 3.10 left the list")
	}
}

func TestParsePre(t *testing.T) {
	for s, want := range map[string][]int{
		"1.28rc1": {1, 28}, "1.28beta1": {1, 28}, "3.16.0a1": {3, 16}, "3.16.0b2": {3, 16}, "3.16.0rc1": {3, 16},
		"4.8-beta2": {4, 8}, "4.8-rc1": {4, 8}, "4.8-dev3": {4, 8},
	} {
		if got, ok := parsePre(s); !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %v %v", s, got, ok)
		}
	}
	for _, s := range []string{"1.28", "1.28.1", "", "x1.2rc1", "1rc1", "1.28rc", "4.8-gamma1", "3.16.0a"} {
		if _, ok := parsePre(s); ok {
			t.Errorf("%q read as a pre-release", s)
		}
	}
}

func TestScoutNewMajorOnlyForHighestMajorLine(t *testing.T) {
	up := calm()
	up[pythonSource] = []string{"4.0.1", "3.15.0"}
	up[godotSource] = []string{"5.0", "4.7", "3.6"}
	got := scout(t, up, godot3(), gdunit("4.7"), py("3.15"))
	var lines []string
	for _, f := range got {
		lines = append(lines, f.Line+" "+f.Version)
	}
	want := []string{"godot4-gdunit4/godot 5.0", "python-mypy/python 4.0"}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("findings %v, want %v", lines, want)
	}
}

func TestScoutMajorKeptByNewestEntryOfLine(t *testing.T) {
	up := calm()
	up[pythonSource] = []string{"3.16.1"}
	got := scout(t, up, py("3.15"), py("3.11"), py("3.14"))
	if len(got) != 1 || got[0].Probe.Version != "3.16" || got[0].Probe.Tools["ruff"] != "0.17.*" {
		t.Fatalf("findings %+v", got)
	}
}

func TestScoutAddonLine(t *testing.T) {
	up := calm()
	up["https://api.github.com/repos/godot-gdunit-labs/gdUnit4/releases?per_page=100"] = []string{"6.3.2", "6.3.0", "6.2.9", "7.0.0-rc1"}
	up["https://api.github.com/repos/bitwes/Gut/releases?per_page=100"] = []string{"9.9.0", "9.7.1", "7.5.0"}
	gut4 := Entry{Stack: "gdscript", Runtime: "godot", Version: "4.7", Fixture: "godot4-gut", Kinds: []string{"test"}, Addons: []Addon{{"GUT", "9.7.1"}}}
	got := scout(t, up, godot3(), gut4, gdunit("4.7"))
	if len(got) != 3 {
		t.Fatalf("findings %+v", got)
	}
	g := got[0]
	if g.Line != "godot3-gut/GUT" || g.Version != "7.5" || g.Probe.Addons[0].Version != "7.5.0" {
		t.Errorf("gut %+v", g)
	}
	if g4 := got[1]; g4.Line != "godot4-gut/GUT" || g4.Version != "9.9" {
		t.Errorf("gut 4 %+v", g4)
	}
	d := got[2]
	if d.Line != "godot4-gdunit4/gdUnit4" || d.Version != "6.3" || d.Probe.Addons[0].Version != "6.3.2" || d.Probe.Addons[1].Version != "0.1.1" {
		t.Errorf("gdunit %+v", d)
	}
	if g.Probe.Addons[0].Name != "GUT" || godot3().Addons[0].Version != "7.4.3" {
		t.Error("probe aliases the entry")
	}
}

func TestScoutToolLines(t *testing.T) {
	up := calm()
	up["https://pypi.org/pypi/ruff/json"] = []string{"0.17.9", "0.18.0", "0.18.3"}
	up["https://pypi.org/pypi/pytest/json"] = []string{"9.1.1", "10.0.0"}
	up["https://pypi.org/pypi/gdtoolkit/json"] = []string{"3.6.0", "4.5.0", "5.0.0", "4.6.0"}
	got := scout(t, up, godot3(), gdunit("4.7"), py("3.15"))
	if len(got) != 2 {
		t.Fatalf("findings %+v", got)
	}
	byLine := map[string]Finding{}
	for _, f := range got {
		byLine[f.Line] = f
	}
	if f := byLine["python-mypy/ruff"]; f.Version != "0.18" || f.Probe.Tools["ruff"] != "0.18.*" || f.Probe.Tools["pytest"] != "9.1.*" {
		t.Errorf("ruff %+v", f)
	}
	// "9.1.*" pins a minor; a new major of pytest counts, as 10 is above 9.
	if f := byLine["python-mypy/pytest"]; f.Version != "10.0" || f.Probe.Tools["pytest"] != "10.0.*" {
		t.Errorf("pytest %+v", f)
	}
	if _, ok := byLine["godot4-gdunit4/gdtoolkit"]; ok {
		t.Error("gdtoolkit follows Godot's major")
	}
}

func TestScoutPinnedMajorTool(t *testing.T) {
	up := calm()
	up["https://pypi.org/pypi/ruff/json"] = []string{"0.17.4", "1.0.0", "1.2.0"}
	e := py("3.15")
	e.Tools = map[string]string{"ruff": "0.*"}
	got := scout(t, up, e)
	if len(got) != 1 || got[0].Version != "1" || got[0].Probe.Tools["ruff"] != "1.*" {
		t.Fatalf("findings %+v", got)
	}
	if IssueTitle(got[0]) != "compat: python-mypy/ruff 1 is out" {
		t.Errorf("title %q", IssueTitle(got[0]))
	}
}

func TestScoutComparesNumerically(t *testing.T) {
	up := calm()
	up[godotSource] = []string{"4.10", "4.9"}
	got := scout(t, up, gdunit("4.9"))
	if len(got) != 1 || got[0].Version != "4.10" {
		t.Fatalf("findings %+v", got)
	}
}

func TestScoutSourceErrors(t *testing.T) {
	up := calm()
	delete(up, pythonSource)
	if _, err := Scout([]Entry{py("3.15")}, up); err == nil || !strings.Contains(err.Error(), pythonSource) {
		t.Errorf("failing source: %v", err)
	}
	up[pythonSource] = []string{"nightly", "3.15rc1"}
	if _, err := Scout([]Entry{py("3.15")}, up); err == nil || !strings.Contains(err.Error(), pythonSource) {
		t.Errorf("junk source: %v", err)
	}
	e := gdunit("4.7")
	e.Addons = []Addon{{"Mystery", "1.0"}}
	if _, err := Scout([]Entry{e}, calm()); err == nil || !strings.Contains(err.Error(), "Mystery") {
		t.Errorf("unknown addon: %v", err)
	}
	e = py("3.15")
	e.Tools = map[string]string{"ruff": "latest"}
	if _, err := Scout([]Entry{e}, calm()); err == nil || !strings.Contains(err.Error(), "latest") {
		t.Errorf("junk pin: %v", err)
	}
	e = py("3.15")
	e.Runtime = "ruby"
	if _, err := Scout([]Entry{e}, calm()); err == nil || !strings.Contains(err.Error(), "ruby") {
		t.Errorf("unknown runtime: %v", err)
	}
}

// A finding's title is the one Resolved reads: open until the probe is in
// the list, resolved after.
func TestFindingTitleRoundTripsThroughResolved(t *testing.T) {
	up := calm()
	up[godotSource] = []string{"4.8"}
	up["https://api.github.com/repos/godot-gdunit-labs/gdUnit4/releases?per_page=100"] = []string{"6.3.0"}
	up["https://pypi.org/pypi/ruff/json"] = []string{"0.18.0"}
	up["https://pypi.org/pypi/pytest/json"] = []string{"10.0.0"}
	entries := []Entry{gdunit("4.7"), py("3.15")}
	got := scout(t, up, entries...)
	if len(got) != 4 {
		t.Fatalf("findings %+v", got)
	}
	for _, f := range got {
		if f.Title != IssueTitle(f) {
			t.Errorf("title %q, want %q", f.Title, IssueTitle(f))
		}
		if Resolved(entries, f.Title) {
			t.Errorf("%q resolved before the probe is in the list", f.Title)
		}
		if !Resolved(append(slices.Clone(entries), *f.Probe), f.Title) {
			t.Errorf("%q not resolved once the probe is in the list", f.Title)
		}
	}
}

func TestResolved(t *testing.T) {
	es := []Entry{gdunit("4.7"), godot3(), py("3.15")}
	for title, want := range map[string]bool{
		"compat: godot4-gdunit4/godot 4.7 is out":        true,
		"compat: godot4-gdunit4/godot 4.6 is out":        true,
		"compat: godot4-gdunit4/godot 4.8 is out":        false,
		"compat: godot4-gdunit4/godot 5.0 is out":        false,
		"compat: godot4-gdunit4/gdUnit4 6.2 is out":      true,
		"compat: godot4-gdunit4/gdUnit4 6.3 is out":      false,
		"compat: python-mypy/ruff 0.17 is out":           true,
		"compat: python-mypy/ruff 0.18 is out":           false,
		"compat: python-mypy/pytest 10 is out":           false,
		"compat: gone-fixture/godot 4.7 is out":          false,
		"compat: python-mypy/python x is out":            false,
		"something else":                                 false,
		"compat: python-mypy/python 3.15 is out":         true,
		"compat: godot3-gut/godot 3.7 is out":            false,
		"compat: godot4-gdunit4/NanoCoverage 0.1 is out": true,
	} {
		if got := Resolved(es, title); got != want {
			t.Errorf("%q = %v, want %v", title, got, want)
		}
	}
}

// The bodies below are trimmed from the real answers of 2026-11-08: go.dev
// with include=all lists an RC as stable:false, python.org flags an alpha as
// pre_release, and godot-builds flags betas as prerelease and tags a stable
// release with its patch and -stable.
func TestParseReleasesMarksPreReleases(t *testing.T) {
	gh := `[{"tag_name":"4.8-beta2","prerelease":true},{"tag_name":"4.8-dev7","prerelease":true},{"tag_name":"4.7.2-stable","prerelease":false},{"tag_name":"3.6.3-stable"},{"tag_name":"4.9-rc1"},{"tag_name":"4.5-stable","draft":true}]`
	want := []Release{{"4.8-beta2", true}, {"4.8-dev7", true}, {"4.7.2", false}, {"3.6.3", false}}
	if got, err := parseReleases(godotSource, []byte(gh)); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("godot %v %v", got, err)
	}
	gh = `[{"tag_name":"v6.3.1"},{"tag_name":"v6.3.0-rc1"},{"tag_name":"v6.2.2","prerelease":true}]`
	if got, err := parseReleases("https://api.github.com/repos/x/y/releases", []byte(gh)); err != nil || !reflect.DeepEqual(got, []Release{{"6.3.1", false}, {"6.3.0-rc1", false}}) {
		t.Errorf("addon %v %v", got, err)
	}
	pyBody := `[{"name":"Python 3.15.0","pre_release":false},{"name":"Python 3.16.0a1","pre_release":true},{"name":"Python install manager 26.4 beta 3","pre_release":true},{"name":"Jython 2.7","pre_release":false}]`
	if got, err := parseReleases(pythonSource, []byte(pyBody)); err != nil || !reflect.DeepEqual(got, []Release{{"3.15.0", false}, {"3.16.0a1", true}}) {
		t.Errorf("python %v %v", got, err)
	}
	goBody := `[{"version":"go1.27.3","stable":true},{"version":"go1.27rc3","stable":false},{"version":"go1.26.10","stable":true}]`
	if got, err := parseReleases(goSource, []byte(goBody)); err != nil || !reflect.DeepEqual(got, []Release{{"1.27.3", false}, {"1.27rc3", true}, {"1.26.10", false}}) {
		t.Errorf("go %v %v", got, err)
	}
	pypi := `{"releases":{"0.17.1":[{"yanked":false}],"0.18.0":[{"yanked":false},{"yanked":true}],"0.19.0":[{"yanked":true}],"0.20.0":[]}}`
	if got, err := parseReleases("https://pypi.org/pypi/ruff/json", []byte(pypi)); err != nil || !reflect.DeepEqual(got, []Release{{"0.17.1", false}, {"0.18.0", false}}) {
		t.Errorf("pypi %v %v", got, err)
	}
	for _, src := range []string{godotSource, pythonSource, goSource, "https://pypi.org/pypi/ruff/json"} {
		if _, err := parseReleases(src, []byte("<html>")); err == nil {
			t.Errorf("%s: junk read", src)
		}
	}
}

// roundTrip serves canned answers and keeps the last request.
type roundTrip struct {
	status int
	body   string
	seen   *http.Request
	err    error
}

func (r *roundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	r.seen = req
	if r.err != nil {
		return nil, r.err
	}
	return &http.Response{StatusCode: r.status, Status: strconv.Itoa(r.status) + " status", Body: io.NopCloser(strings.NewReader(r.body)), Header: http.Header{}}, nil
}

func TestHTTPUpstream(t *testing.T) {
	rt := &roundTrip{status: 200, body: `[{"tag_name":"4.8-stable"}]`}
	h := HTTPUpstream{Client: &http.Client{Transport: rt}, Token: "tok"}
	got, err := h.Releases(godotSource)
	if err != nil || !reflect.DeepEqual(got, []Release{{"4.8", false}}) {
		t.Fatalf("%v %v", got, err)
	}
	if rt.seen.Header.Get("Authorization") != "Bearer tok" {
		t.Errorf("GitHub request carries %q", rt.seen.Header.Get("Authorization"))
	}
	rt.body = `{"releases":{}}`
	if _, err := h.Releases("https://pypi.org/pypi/ruff/json"); err != nil || rt.seen.Header.Get("Authorization") != "" {
		t.Errorf("pypi: %v, auth %q", err, rt.seen.Header.Get("Authorization"))
	}
	rt.status = 503
	if _, err := h.Releases(godotSource); err == nil || !strings.Contains(err.Error(), "503") {
		t.Errorf("status: %v", err)
	}
	rt.status, rt.body = 200, "junk"
	if _, err := h.Releases(godotSource); err == nil || !strings.Contains(err.Error(), "unreadable") {
		t.Errorf("junk: %v", err)
	}
	rt.err = errors.New("offline")
	if _, err := h.Releases(godotSource); err == nil {
		t.Error("transport error read")
	}
	if _, err := h.Releases("://bad"); err == nil {
		t.Error("bad url read")
	}
	// The zero value uses the default client; only the failure path runs here.
	if _, err := (HTTPUpstream{}).Releases("http://127.0.0.1:1/"); err == nil {
		t.Error("closed port read")
	}
	rt.err, rt.status = nil, 200
	rt.body = ""
	if _, err := (HTTPUpstream{Client: &http.Client{Transport: errBody{}}}).Releases(goSource); err == nil {
		t.Error("broken body read")
	}
}

type errBody struct{}

func (errBody) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(iotestErr{})}, nil
}

type iotestErr struct{}

func (iotestErr) Read([]byte) (int, error) { return 0, errors.New("reset") }

func TestSourceOf(t *testing.T) {
	if _, err := sourceOf("runtime", "ruby"); err == nil {
		t.Error("unknown runtime has a source")
	}
	if _, err := sourceOf("addon", "Mystery"); err == nil {
		t.Error("unknown addon has a source")
	}
	if s, err := sourceOf("tool", "ruff"); err != nil || s != "https://pypi.org/pypi/ruff/json" {
		t.Errorf("tool %q %v", s, err)
	}
}

func TestLoadEntryAndProbes(t *testing.T) {
	dir := t.TempDir()
	good := `{"stack":"go","runtime":"go","version":"1.28","fixture":"go-module","kinds":["lint"]}`
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	e, err := LoadEntry(write("e.json", good))
	if err != nil || e.Key() != "go/go-1.28/go-module" {
		t.Fatalf("%+v %v", e, err)
	}
	es, err := LoadProbes(write("p.json", "["+good+"]"))
	if err != nil || len(es) != 1 {
		t.Fatalf("%+v %v", es, err)
	}
	for name, body := range map[string]string{
		"nofixture.json": `{"stack":"go","runtime":"go","version":"1.28","fixture":"nope","kinds":["lint"]}`,
		"unknown.json":   `{"stack":"go","bogus":1}`,
		"junk.json":      `nonsense`,
	} {
		p := write(name, body)
		if _, err := LoadEntry(p); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: entry %v", name, err)
		}
		if _, err := LoadProbes(write("l"+name, "["+body+"]")); err == nil {
			t.Errorf("%s: probes read", name)
		}
	}
	if _, err := LoadEntry(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("missing file read")
	}
}

func TestProbeMatrix(t *testing.T) {
	e := Entry{Stack: "go", Runtime: "go", Version: "1.28", Fixture: "go-module", Kinds: []string{"lint"}}
	jobs := ProbeMatrix([]Entry{e})
	if len(jobs) != 2 || jobs[0].OS != "linux" || jobs[1].OS != "windows" || jobs[0].Entry != e.Key() ||
		jobs[0].Probe != `{"stack":"go","runtime":"go","version":"1.28","fixture":"go-module","kinds":["lint"]}` {
		t.Fatalf("%+v", jobs)
	}
	if got := ProbeMatrix(nil); got == nil || len(got) != 0 {
		t.Errorf("empty = %v", got)
	}
}

func TestRunWithEntryRunsProbe(t *testing.T) {
	e := Entry{Stack: "go", Runtime: "go", Version: "9.9", Fixture: "go-module", Kinds: []string{"lint"}}
	rep, err := Run(RunOptions{Entry: &e, Work: t.TempDir(),
		Check: func(string, []string) ([]LaneResult, error) {
			return []LaneResult{{Lane: "lint/go", State: "ok"}}, nil
		}})
	if err != nil || rep.Entry != e.Key() {
		t.Fatalf("%+v %v", rep, err)
	}
}

// The body is trimmed from the real answer of endoflife.date on 2026-11-08.
func TestHTTPUpstreamEOL(t *testing.T) {
	rt := &roundTrip{status: 200, body: `{"schema_version":"1.2.1","result":{"name":"python","releases":[{"name":"3.11","isEol":false,"eolFrom":"2027-10-31"},{"name":"3.10","isEol":true,"isMaintained":false}]}}`}
	h := HTTPUpstream{Client: &http.Client{Transport: rt}, Token: "tok"}
	got, err := h.EOL("python")
	if err != nil || !reflect.DeepEqual(got, map[string]bool{"3.11": false, "3.10": true}) {
		t.Fatalf("%v %v", got, err)
	}
	if rt.seen.URL.String() != "https://endoflife.date/api/v1/products/python" || rt.seen.Header.Get("Authorization") != "" {
		t.Errorf("asked %s with auth %q", rt.seen.URL, rt.seen.Header.Get("Authorization"))
	}
	rt.status = 404
	if _, err := h.EOL("nope"); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("status: %v", err)
	}
	rt.status, rt.body = 200, "junk"
	if _, err := h.EOL("python"); err == nil || !strings.Contains(err.Error(), "unreadable") {
		t.Errorf("junk: %v", err)
	}
}
