package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAreasCarryTheirProperties(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "registry.toml"), []byte(
		"[[area]]\nscope = \"knowledge\"\npath = \"/v\"\nwiki = \"/v/90 Wiki\"\n"+
			"signpost = true\nshared = true\n\n"+
			"[[area]]\nscope = \"project/p\"\npath = \"/p\"\nwiki = \"/p/docs/wiki\"\nreadonly = true\n"), 0o644)
	areas, err := ReadRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 2 {
		t.Fatalf("%d areas, want 2", len(areas))
	}
	if !areas[0].Signpost || !areas[0].Shared {
		t.Error("signpost/shared not read; wrong-direction and unlisted-area hang on them")
	}
	if !areas[1].ReadOnly {
		t.Error("readonly not read")
	}
}

func TestTheThreeStringsComeThroughUnchanged(t *testing.T) {
	// The test above asserts only the flags, so a reader that never fills
	// Scope, Path or WikiPath passes it -- and the later unlisted-area rule
	// reads exactly WikiPath and Signpost together. Asserted verbatim because
	// the registry on this machine writes forward slashes on Windows: a
	// reader that cleans or converts them would answer a path the file never
	// contained.
	dir := t.TempDir()
	writeRegistry(t, dir, "[[area]]\nscope = \"knowledge\"\npath = \"C:/v\"\nwiki = \"C:/v/90 Wiki\"\n")
	areas, err := ReadRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 1 {
		t.Fatalf("%d areas, want 1", len(areas))
	}
	if areas[0].Scope != "knowledge" {
		t.Errorf("Scope = %q, want \"knowledge\"", areas[0].Scope)
	}
	if areas[0].Path != "C:/v" {
		t.Errorf("Path = %q, want \"C:/v\" unchanged", areas[0].Path)
	}
	if areas[0].WikiPath != "C:/v/90 Wiki" {
		t.Errorf("WikiPath = %q, want \"C:/v/90 Wiki\" unchanged", areas[0].WikiPath)
	}
}

func TestAnAreaWithoutAWikiKeepsAnEmptyWikiPath(t *testing.T) {
	// `wiki` is the one optional string: an area that names no wiki is
	// registered with WikiPath "", while one that names an empty wiki is
	// refused (TestARegistryRefusesWhatItCannotUse). No entry of the registry on this
	// machine omits it -- all nine name a wiki -- so nothing but this test
	// keeps `wiki` from being demanded like `scope` and `path`.
	dir := t.TempDir()
	writeRegistry(t, dir, "[[area]]\nscope = \"a\"\npath = \"/a\"\n")
	areas, err := ReadRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 1 {
		t.Fatalf("%d areas, want 1; a missing wiki does not make an area unusable", len(areas))
	}
	if areas[0].WikiPath != "" {
		t.Errorf("WikiPath = %q, want the empty string for an area that names no wiki", areas[0].WikiPath)
	}
}

func TestARegistryRefusesWhatItCannotUse(t *testing.T) {
	// Every rule of the registry, and every TOML type a refusal can name.
	for name, row := range map[string]struct{ body, want string }{
		"area as a table": {"[area]\nscope = \"x\"\npath = \"/a\"\n",
			"area must be an array of [[area]] tables, found table"},
		"entry not a table": {"area = [1]\n",
			"[[area]] #1 must be a table, found integer"},
		"missing scope": {"[[area]]\npath = \"/a\"\n",
			`[[area]] #1 is missing "scope"`},
		"scope an integer": {"[[area]]\nscope = 3\npath = \"/a\"\n",
			"[[area]] #1: scope must be a non-empty string, found integer"},
		"scope a boolean": {"[[area]]\nscope = true\npath = \"/a\"\n",
			"[[area]] #1: scope must be a non-empty string, found boolean"},
		"empty scope": {"[[area]]\nscope = \"\"\npath = \"/a\"\n",
			`[[area]] #1: scope must be a non-empty string, found ""`},
		"duplicate scope": {"[[area]]\nscope = \"x\"\npath = \"/a\"\n\n[[area]]\nscope = \"x\"\npath = \"/b\"\n",
			`[[area]] #2: duplicate scope "x" (first at #1)`},
		"unusable scope": {"[[area]]\nscope = \"///\"\npath = \"/a\"\n",
			`[[area]] "///": scope has no letter, digit, "_", "." or "-" and cannot name a state directory`},
		"shared state directory": {"[[area]]\nscope = \"a/b\"\npath = \"/a\"\n\n[[area]]\nscope = \"a-b\"\npath = \"/b\"\n",
			`scopes "a/b" and "a-b" share the state directory "a-b"`},
		"missing path": {"[[area]]\nscope = \"x\"\n",
			`[[area]] "x" is missing "path"`},
		"path a datetime": {"[[area]]\nscope = \"x\"\npath = 1979-05-27\n",
			`[[area]] "x": path must be a non-empty string, found datetime`},
		"empty path": {"[[area]]\nscope = \"x\"\npath = \"\"\n",
			`[[area]] "x": path must be a non-empty string, found ""`},
		"wiki an array": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nwiki = [\"/a\"]\n",
			`[[area]] "x": wiki must be a non-empty string, found array`},
		"empty wiki": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nwiki = \"\"\n",
			`[[area]] "x": wiki must be a non-empty string, found ""`},
		"readonly a string": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nreadonly = \"yes\"\n",
			`[[area]] "x": readonly must be a boolean, found string`},
		"signpost an integer": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nsignpost = 1\n",
			`[[area]] "x": signpost must be a boolean, found integer`},
		"shared a float": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nshared = 1.5\n",
			`[[area]] "x": shared must be a boolean, found float`},
		"workspace an integer": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nworkspace = 0\n",
			`[[area]] "x": workspace must be a boolean, found integer`},
		"two signposts": {"[[area]]\nscope = \"x\"\npath = \"/a\"\nsignpost = true\n\n[[area]]\nscope = \"y\"\npath = \"/b\"\nsignpost = true\n",
			`scopes "x" and "y" both declare signpost; only one area may`},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeRegistry(t, dir, row.body)
			areas, err := ReadRegistry(dir)
			want := filepath.Join(dir, "registry.toml") + ": " + row.want
			if err == nil || err.Error() != want {
				t.Fatalf("ReadRegistry = %+v, %v; want %q", areas, err, want)
			}
		})
	}
}

func TestARegistryWithoutAreasAnswersNone(t *testing.T) {
	for _, body := range []string{"", "area = []\n"} {
		dir := t.TempDir()
		writeRegistry(t, dir, body)
		areas, err := ReadRegistry(dir)
		if err != nil || len(areas) != 0 {
			t.Errorf("%q: ReadRegistry = %+v, %v; want no areas", body, areas, err)
		}
	}
}

func TestFalseFlagsAndOneSignpostAreFine(t *testing.T) {
	dir := t.TempDir()
	writeRegistry(t, dir, "[[area]]\nscope = \"x\"\npath = \"/a\"\nreadonly = false\nsignpost = true\n\n"+
		"[[area]]\nscope = \"y\"\npath = \"/b\"\nsignpost = false\n")
	areas, err := ReadRegistry(dir)
	if err != nil || len(areas) != 2 || areas[0].ReadOnly || !areas[0].Signpost || areas[1].Signpost {
		t.Fatalf("ReadRegistry = %+v, %v", areas, err)
	}
}

func TestAMissingRegistryIsAnError(t *testing.T) {
	// The whole file missing is not one broken entry: nothing is registered,
	// and answering that with an empty list would let every later rule pass
	// on a machine that was never set up.
	areas, err := ReadRegistry(t.TempDir())
	if err == nil {
		t.Fatalf("ReadRegistry = %+v, want an error when there is no registry.toml", areas)
	}
	// Asked through errors.Is rather than by text: `os.ReadFile` already puts
	// the path into its own message, so a substring check on "registry.toml"
	// passes even when the reader drops the wrapping entirely -- measured, a
	// mutant that returns the bare read error keeps this test green. What the
	// wrapping must not lose is the cause, so that a caller can tell "never
	// set up" from "cannot be read".
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error %q does not carry the cause; a caller cannot tell a missing registry from an unreadable one", err)
	}
	if n := strings.Count(err.Error(), "registry.toml"); n != 1 {
		t.Errorf("error %q names the file %d times; the read error already names it once", err, n)
	}
}

func TestABrokenRegistryFileIsAnError(t *testing.T) {
	// A file that does not parse yields zero entries from the decoder, which
	// is indistinguishable from an empty registry unless the error is passed
	// on.
	dir := t.TempDir()
	writeRegistry(t, dir, "[[area]\nscope = \"a\"\n")
	areas, err := ReadRegistry(dir)
	if err == nil {
		t.Fatalf("ReadRegistry = %+v, want an error for a file that is not TOML", areas)
	}
	if !strings.Contains(err.Error(), "registry.toml") {
		t.Errorf("error %q does not name the file it failed on", err)
	}
}

// writeRegistry fails the test instead of returning the error, so a fixture
// that cannot be written is reported as itself and not as a missing registry.
func writeRegistry(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceIsReadLikeTheOtherFlags(t *testing.T) {
	// The fourth flag of `Area` in registry.py, and the one the tests of the
	// brief never name. Three of the nine areas in the real registry set it
	// (project/ultra-brain, project/space, project/ecoflow) and one of those
	// three sets `readonly` beside it, while project/iam-wiki sets `readonly`
	// alone -- so a reader that answered workspace from readonly would look
	// right on project/space and wrong on both of its neighbours. Both
	// directions are asserted for that reason.
	dir := t.TempDir()
	writeRegistry(t, dir,
		"[[area]]\nscope = \"w\"\npath = \"/w\"\nworkspace = true\n\n"+
			"[[area]]\nscope = \"r\"\npath = \"/r\"\nreadonly = true\n")
	areas, err := ReadRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 2 {
		t.Fatalf("%d areas, want 2", len(areas))
	}
	if !areas[0].Workspace {
		t.Error("workspace not read")
	}
	if areas[1].Workspace {
		t.Error("workspace true for an area that only declares readonly")
	}
}

func TestTheEnvironmentBeatsThePlatformDefault(t *testing.T) {
	// `resolve_state_dir` (src/brain/paths.py) reads BRAIN_STATE_DIR before it
	// asks the platform. Without that step every test and every second
	// installation would read the one registry under %LOCALAPPDATA%.
	dir := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", dir)
	if got := StateDir(); got != dir {
		t.Errorf("StateDir() = %q, want %q from LOOMUX_STATE_DIR", got, dir)
	}
}

func TestAnEmptyEnvironmentVariableIsNoAnswer(t *testing.T) {
	// Measured against the original: with BRAIN_STATE_DIR set to the empty
	// string, `resolve_state_dir()` answers
	// 'C:\Users\micro\AppData\Local\brain' -- `if from_env:` treats the
	// empty value as no value. Go cannot tell empty from unset through
	// os.Getenv at all, so the same behaviour comes out for free here; the
	// test pins it because the alternative is a state directory at the
	// working directory's root.
	t.Setenv("LOOMUX_STATE_DIR", "")
	if got := StateDir(); got == "" {
		t.Error("StateDir() = \"\"; an empty LOOMUX_STATE_DIR must fall through to the platform default")
	}
}

func TestTheWindowsDefaultIsLocalAppDataBrain(t *testing.T) {
	// LOCALAPPDATA deliberately does not lie under the home directory. With
	// the usual `C:\Users\x\AppData\Local` beneath `C:\Users\x` the two
	// branches of defaultStateDir answer the same string, and dropping the
	// LOCALAPPDATA branch altogether kept this test green -- measured.
	got := defaultStateDir("windows", stub(map[string]string{"LOCALAPPDATA": `D:\state`}), `C:\Users\x`)
	if want := filepath.Join(`D:\state`, "loomux"); got != want {
		t.Errorf("defaultStateDir = %q, want %q", got, want)
	}
}

func TestAWindowsWithoutLocalAppDataFallsBackToTheHome(t *testing.T) {
	// `_platform_default` calls such a Windows broken and falls back rather
	// than raising, so that an explicit location still works. The fallback is
	// spelt out there as home/AppData/Local/brain.
	got := defaultStateDir("windows", stub(map[string]string{}), `C:\Users\x`)
	if want := filepath.Join(`C:\Users\x`, "AppData", "Local", "loomux"); got != want {
		t.Errorf("defaultStateDir = %q, want %q", got, want)
	}
}

func TestElsewhereXdgStateHomeDecides(t *testing.T) {
	got := defaultStateDir("linux", stub(map[string]string{"XDG_STATE_HOME": "/s"}), "/home/x")
	if want := filepath.Join("/s", "loomux"); got != want {
		t.Errorf("defaultStateDir = %q, want %q", got, want)
	}
}

func TestElsewhereWithoutXdgItIsLocalStateBrain(t *testing.T) {
	got := defaultStateDir("linux", stub(map[string]string{}), "/home/x")
	if want := filepath.Join("/home/x", ".local", "state", "loomux"); got != want {
		t.Errorf("defaultStateDir = %q, want %q", got, want)
	}
}

func TestStateDirAsksThePlatformItRunsOn(t *testing.T) {
	// The two above are pure and would stay green if StateDir passed a fixed
	// operating system, or the wrong home, or read nothing from the real
	// environment at all. This one binds the wiring to the machine.
	t.Setenv("LOOMUX_STATE_DIR", "")
	// Both platform variables are moved off the home directory for the same
	// reason as above: with the real values of this machine, a StateDir that
	// passed a lookup answering nothing produced the identical string and
	// this test stayed green -- measured.
	t.Setenv("LOCALAPPDATA", `D:\state`)
	t.Setenv("XDG_STATE_HOME", "/state")
	home, _ := os.UserHomeDir()
	if want := defaultStateDir(runtime.GOOS, os.Getenv, home); StateDir() != want {
		t.Errorf("StateDir() = %q, want %q", StateDir(), want)
	}
}

// stub answers from a fixed map, so a platform default can be measured for an
// operating system the test is not running on.
func stub(env map[string]string) func(string) string {
	return func(key string) string { return env[key] }
}

func TestStateDirPassesTheRealHomeOn(t *testing.T) {
	// The test above cannot see the home directory: with LOCALAPPDATA set,
	// defaultStateDir never reads it, and a StateDir that passed the empty
	// string stayed green -- measured. Emptying both platform variables makes
	// the home the only remaining input on either operating system.
	t.Setenv("LOOMUX_STATE_DIR", "")
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("XDG_STATE_HOME", "")
	home, _ := os.UserHomeDir()
	if want := defaultStateDir(runtime.GOOS, os.Getenv, home); StateDir() != want {
		t.Errorf("StateDir() = %q, want %q", StateDir(), want)
	}
	if !strings.Contains(StateDir(), home) {
		t.Errorf("StateDir() = %q does not sit under the home directory %q", StateDir(), home)
	}
}

func TestSharedWithoutASignpostStaysSharedWithoutOne(t *testing.T) {
	// TestAreasCarryTheirProperties sets `signpost` and `shared` on the same
	// area and neither on the other, so it cannot tell the two fields apart:
	// swapping them in the struct literal, or filling either from the other,
	// kept it green -- all three measured. This entry mirrors
	// `engineering/python` in the registry on this machine, which sets
	// `shared` and no `signpost` (registry.toml:9-12). The numbers are what
	// makes the confusion visible: of the nine areas exactly one declares
	// `signpost` (`knowledge`) and three declare `shared`. So a reader that
	// filled Signpost from Shared would hand the later rules three signposts
	// where the registry has one, and one that filled Shared from Signpost a
	// single shared area where it has three.
	dir := t.TempDir()
	writeRegistry(t, dir, "[[area]]\nscope = \"engineering/python\"\npath = \"/e/p\"\nshared = true\n")
	areas, err := ReadRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 1 {
		t.Fatalf("%d areas, want 1", len(areas))
	}
	if !areas[0].Shared {
		t.Error("shared not read; the area declares it")
	}
	if areas[0].Signpost {
		t.Error("signpost true for an area that only declares shared; the two fields are being read from one another")
	}
}
