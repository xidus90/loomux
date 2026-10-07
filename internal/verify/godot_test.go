package verify

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

func TestParseGodotVersion(t *testing.T) {
	for out, want := range map[string]struct {
		version  string
		mono, ok bool
	}{
		"4.7.1.stable.mono.official.a13da4feb\n":              {"4.7", true, true},
		"4.7.1.stable.official.a13da4feb\r\n":                 {"4.7", false, true},
		"4.8.beta1.mono.official.0123abcd\n":                  {"4.8", true, true},
		"WARNING: no audio driver\n4.7.stable.official.abc\n": {"4.7", false, true},
		"4.7\n":          {"", false, false},
		"4.x.stable.a\n": {"", false, false},
		"x.7.stable.a\n": {"", false, false},
		"4..stable.a\n":  {"", false, false},
		"":               {"", false, false},
		"Godot Engine\n": {"", false, false},
	} {
		v, mono, ok := parseGodotVersion(out)
		if v != want.version || mono != want.mono || ok != want.ok {
			t.Errorf("%q: %q %v %v", out, v, mono, ok)
		}
	}
}

func TestProjectGodotReadsFeaturesAndDotnet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project.godot")
	body := "[application]\nconfig/features=PackedStringArray(\"4.7\", \"Mobile\")\n\n[dotnet]\nproject/assembly_name=\"Kontari\"\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, mono, err := projectGodot(path); v != "4.7" || !mono || err != nil {
		t.Fatalf("%q %v %v", v, mono, err)
	}
	if err := os.WriteFile(path, []byte("config/features=PackedStringArray(\"Forward Plus\")\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := projectGodot(path); err == nil {
		t.Fatal("no version in config/features must be an error")
	}
	if _, _, err := projectGodot(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("a missing project.godot must be an error")
	}
}

// A file that cannot be read (here a directory in its place) is not the same
// as one that names no version.
func TestProjectGodotSaysWhenTheFileCannotBeRead(t *testing.T) {
	if _, _, err := projectGodot(t.TempDir()); err == nil || !strings.Contains(err.Error(), "cannot be read") {
		t.Fatalf("%v", err)
	}
}

// A long line, as an editor writes for a big packed array, is no reason to
// give up on the version, whichever side of config/features it stands.
func TestProjectGodotReadsPastALongLine(t *testing.T) {
	long := "x=" + strings.Repeat("y", 70<<10) + "\n"
	feature := "config/features=PackedStringArray(\"4.7\")\n"
	for name, body := range map[string]string{"before": long + feature + "[dotnet]\n", "after": feature + long + "[dotnet]\n"} {
		path := filepath.Join(t.TempDir(), "project.godot")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if v, mono, err := projectGodot(path); v != "4.7" || !mono || err != nil {
			t.Errorf("%s: %q %v %v", name, v, mono, err)
		}
	}
}

// The first config/features line that names a version is the one that counts.
func TestProjectGodotTakesTheFirstFeaturesLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "project.godot")
	body := "config/features=PackedStringArray(\"4.7\")\nconfig/features=PackedStringArray(\"4.8\")\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, _, err := projectGodot(path); v != "4.7" || err != nil {
		t.Fatalf("%q %v", v, err)
	}
}

// godotWorld is a Godot project dir whose project.godot wants version, with
// .NET when mono is set.
func godotWorld(t *testing.T, version string, mono bool) string {
	t.Helper()
	dir := t.TempDir()
	body := "config/features=PackedStringArray(\"" + version + "\")\n"
	if mono {
		body += "[dotnet]\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "project.godot"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// sources answers every outside question from maps: variables, PATH, files
// that exist, and what each binary says to --version.
func sources(root, configured, goos string, vars map[string]string, path map[string]string, files []string, versions map[string]string) godotSources {
	return godotSources{
		root: root, configured: configured, goos: goos,
		getenv: func(k string) string { return vars[k] },
		abs:    func(p string) string { return p },
		look: func(name string) (string, error) {
			if p, ok := path[name]; ok {
				return p, nil
			}
			return "", errors.New("not on PATH")
		},
		exists: func(p string) bool {
			for _, f := range files {
				if filepath.Clean(f) == filepath.Clean(p) {
					return true
				}
			}
			return false
		},
		version: func(bin string) (string, error) {
			if v, ok := versions[bin]; ok {
				return v, nil
			}
			return "", errors.New("exit 1")
		},
	}
}

// All three sources set: the variable wins, then the configured path, then
// PATH; a source naming a file that is not there is passed over.
func TestGodotSearchOrder(t *testing.T) {
	dir := godotWorld(t, "4.7", false)
	root := filepath.Join("C:", "repo")
	conf := filepath.Join(root, ".tools", "godot.exe")
	v := "4.7.1.stable.official.x\n"
	all := map[string]string{"/env/godot": v, conf: v, "/path/godot": v}
	vars := map[string]string{"GODOT_BIN": "/env/godot"}
	onPath := map[string]string{"godot": "/path/godot"}
	files := []string{"/env/godot", conf, "/path/godot"}
	for name, row := range map[string]struct {
		files []string
		want  string
	}{
		"variable":   {files, "/env/godot"},
		"configured": {files[1:], conf},
		"path":       {files[2:], "/path/godot"},
	} {
		s := sources(root, ".tools/godot.exe", "linux", vars, onPath, row.files, all)
		if bin, pre, note := s.resolve(dir); bin != row.want || pre != "" {
			t.Errorf("%s: %q %q %q", name, bin, pre, note)
		}
	}
}

// godot4 on PATH is as good as godot, and godot comes first.
func TestGodotLooksForBothNamesOnPath(t *testing.T) {
	dir := godotWorld(t, "4.7", false)
	v := "4.7.1.stable.official.x\n"
	versions := map[string]string{"/p/godot": v, "/p/godot4": v}
	both := map[string]string{"godot": "/p/godot", "godot4": "/p/godot4"}
	s := sources("/repo", "", "linux", nil, both, nil, versions)
	if bin, pre, note := s.resolve(dir); bin != "/p/godot" || pre != "" {
		t.Fatalf("both: %q %q %q", bin, pre, note)
	}
	s = sources("/repo", "", "linux", nil, map[string]string{"godot4": "/p/godot4"}, nil, versions)
	if bin, pre, note := s.resolve(dir); bin != "/p/godot4" || pre != "" {
		t.Fatalf("godot4 only: %q %q %q", bin, pre, note)
	}
}

// A configured path is relative to the root, and an absolute one stays.
func TestGodotConfiguredPathRelativeAndAbsolute(t *testing.T) {
	dir := godotWorld(t, "4.7", false)
	root := t.TempDir()
	abs := filepath.Join(t.TempDir(), "godot")
	rel := filepath.Join(root, "tools", "godot")
	v := "4.7.1.stable.official.x\n"
	versions := map[string]string{abs: v, rel: v}
	files := []string{abs, rel}
	if bin, pre, note := sources(root, "tools/godot", "linux", nil, nil, files, versions).resolve(dir); bin != rel || pre != "" {
		t.Errorf("relative: %q %q %q", bin, pre, note)
	}
	if bin, pre, note := sources(root, filepath.ToSlash(abs), "linux", nil, nil, files, versions).resolve(dir); bin != abs || pre != "" {
		t.Errorf("absolute: %q %q %q", bin, pre, note)
	}
}

func TestGodotPrefersTheConsoleExeOnWindows(t *testing.T) {
	dir := godotWorld(t, "4.7", true)
	gui := `C:\tools\Godot_v4.7.1-stable_mono_win64.exe`
	console := `C:\tools\Godot_v4.7.1-stable_mono_win64_console.exe`
	v := map[string]string{console: "4.7.1.stable.mono.official.x\n"}
	// Through the variable, not the configured path: a configured `C:\…` is
	// no absolute path on the Linux runner and would be joined to the root.
	s := sources(`C:\repo`, "", "windows", map[string]string{"GODOT_BIN": gui}, nil, []string{gui, console}, v)
	if bin, pre, _ := s.resolve(dir); bin != console || pre != "" {
		t.Fatalf("%q %q", bin, pre)
	}
	s.goos = "linux"
	if bin, pre, _ := s.resolve(dir); bin != "" || pre != StateUnready {
		t.Fatalf("not windows: %q %q", bin, pre)
	}
}

// Where the console build is not there, or the name already is one, the
// binary found is the one that counts.
func TestGodotKeepsTheBinaryItFoundOnWindows(t *testing.T) {
	dir := godotWorld(t, "4.7", false)
	v := "4.7.1.stable.official.x\n"
	for name, row := range map[string]struct {
		found string
		files []string
	}{
		"no console beside it":      {`C:\tools\godot.exe`, []string{`C:\tools\godot.exe`}},
		"already the console build": {`C:\tools\godot_console.exe`, []string{`C:\tools\godot_console.exe`, `C:\tools\godot_console_console.exe`}},
		"upper-case extension":      {`C:\tools\GODOT.EXE`, []string{`C:\tools\GODOT.EXE`}},
		"not an exe":                {`C:\tools\godot.sh`, []string{`C:\tools\godot.sh`, `C:\tools\godot_console.sh`}},
	} {
		s := sources(`C:\repo`, "", "windows", map[string]string{"GODOT_BIN": row.found}, nil, row.files, map[string]string{row.found: v})
		if bin, pre, note := s.resolve(dir); bin != row.found || pre != "" {
			t.Errorf("%s: %q %q %q", name, bin, pre, note)
		}
	}
	// An upper-case .EXE gets its console twin all the same.
	gui, console := `C:\tools\GODOT.EXE`, `C:\tools\GODOT_console.EXE`
	s := sources(`C:\repo`, "", "windows", map[string]string{"GODOT_BIN": gui}, nil, []string{gui, console}, map[string]string{console: v})
	if bin, pre, note := s.resolve(dir); bin != console || pre != "" {
		t.Errorf("upper-case twin: %q %q %q", bin, pre, note)
	}
}

func TestGodotJudgesTheBinaryAgainstTheProject(t *testing.T) {
	bin := "/path/godot"
	for name, row := range map[string]struct {
		want    string
		mono    bool
		version string
		pre     State
		note    string
	}{
		"same minor, other patch": {"4.7", false, "4.7.3.stable.official.x\n", "", ""},
		"other minor":             {"4.7", false, "4.8.stable.official.x\n", StateUnready, "Godot 4.8 found (PATH: /path/godot), project.godot wants 4.7"},
		"mono missing":            {"4.7", true, "4.7.1.stable.official.x\n", StateUnready, "wants 4.7 with .NET (mono)"},
		"mono where none needed":  {"4.7", false, "4.7.1.stable.mono.official.x\n", "", ""},
		"no version":              {"4.7", false, "", StateUnready, "/path/godot (PATH) gave no version: exit 1"},
		"output without version":  {"4.7", false, "Godot Engine\n", StateUnready, "/path/godot (PATH) gave no version: its --version output names none"},
	} {
		dir := godotWorld(t, row.want, row.mono)
		versions := map[string]string{}
		if row.version != "" {
			versions[bin] = row.version
		}
		s := sources("/repo", "", "linux", nil, map[string]string{"godot": bin}, []string{bin}, versions)
		got, pre, note := s.resolve(dir)
		if pre != row.pre || !strings.Contains(note, row.note) || (pre == "" && got != bin) {
			t.Errorf("%s: %q %q %q", name, got, pre, note)
		}
	}
}

// The hint to name the _console.exe is for a Windows build that is not one
// already; elsewhere it would only mislead.
func TestGodotHintsAtTheConsoleBuildOnlyOnWindows(t *testing.T) {
	dir := godotWorld(t, "4.7", false)
	for name, row := range map[string]struct {
		goos, bin string
		hint      bool
	}{
		"windows window build":  {"windows", `C:\tools\godot.exe`, true},
		"windows console build": {"windows", `C:\tools\Godot_CONSOLE.exe`, false},
		"linux":                 {"linux", "/p/godot", false},
	} {
		s := sources(`C:\repo`, "", row.goos, map[string]string{"GODOT_BIN": row.bin}, nil, []string{row.bin}, nil)
		_, pre, note := s.resolve(dir)
		if pre != StateUnready || !strings.Contains(note, "gave no version: exit 1") || strings.Contains(note, "_console.exe") != row.hint {
			t.Errorf("%s: %q %q", name, pre, note)
		}
	}
}

// A relative GODOT_BIN is read against the directory loomux runs in, but the
// lane runs it from its area: the answer is the absolute path.
func TestGodotForMakesARelativeVariableAbsolute(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("godot-rel", nil, 0o755); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs("godot-rel")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GODOT_BIN", "godot-rel")
	var argv []string
	find := GodotFor(t.TempDir(), "", func(s child.Spec) child.Result {
		argv = s.Argv
		return child.Result{Stdout: "4.7.1.stable.official.x\n"}
	}, nil)
	if bin, pre, note := find(godotWorld(t, "4.7", false)); bin != want || pre != "" || len(argv) == 0 || argv[0] != want {
		t.Fatalf("%q %q %q %v, want %q", bin, pre, note, argv, want)
	}
}

func TestGodotWithoutABinaryOrAProject(t *testing.T) {
	dir := godotWorld(t, "4.7", false)
	s := sources("/repo", "", "linux", map[string]string{"GODOT_BIN": "/gone"}, nil, nil, nil)
	if _, pre, note := s.resolve(dir); pre != StateMissingTool || !strings.Contains(note, "GODOT_BIN names /gone, which is not there") {
		t.Fatalf("%q %q", pre, note)
	}
	s = sources("/repo", "", "linux", nil, map[string]string{"godot": "/p/godot"}, []string{"/p/godot"}, map[string]string{"/p/godot": "4.7.stable.official.x"})
	if _, pre, note := s.resolve(t.TempDir()); pre != StateUnready || !strings.Contains(note, "project.godot") {
		t.Fatalf("%q %q", pre, note)
	}
}

// Nothing found says what was looked at, source by source.
func TestGodotSaysWhatItLookedAt(t *testing.T) {
	dir := godotWorld(t, "4.7", false)
	s := sources("/repo", ".tools/godot", "linux", nil, nil, nil, nil)
	_, pre, note := s.resolve(dir)
	want := "GODOT_BIN is unset; [verify.gdscript] godot names " + filepath.Join("/repo", ".tools", "godot") + ", which is not there; neither godot nor godot4 is on PATH"
	if pre != StateMissingTool || !strings.HasSuffix(note, want) {
		t.Fatalf("%q %q", pre, note)
	}
	s = sources("/repo", "", "linux", nil, nil, nil, nil)
	if _, _, note := s.resolve(dir); !strings.Contains(note, "[verify.gdscript] godot is unset") {
		t.Fatalf("%q", note)
	}
}

// A directory named like the binary is no binary.
func TestGodotForPassesOverADirectory(t *testing.T) {
	t.Setenv("GODOT_BIN", t.TempDir())
	t.Setenv("PATH", "")
	find := GodotFor(t.TempDir(), "", func(child.Spec) child.Result { t.Fatal("started"); return child.Result{} }, nil)
	if bin, pre, note := find(godotWorld(t, "4.7", false)); bin != "" || pre != StateMissingTool {
		t.Fatalf("%q %q %q", bin, pre, note)
	}
}

// One --version per binary and run, whatever the number of areas; a binary
// that fails says so every time without a second start.
func TestGodotForAsksEachBinaryOnce(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "godot")
	if err := os.WriteFile(bin, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GODOT_BIN", bin)
	for _, row := range []struct {
		res child.Result
		pre State
	}{
		{child.Result{Stdout: "4.7.1.stable.official.x\n"}, ""},
		// Output that would pass does not count when the run failed.
		{child.Result{Code: 1, Stdout: "4.7.1.stable.official.x\n"}, StateUnready},
		{child.Result{Err: errors.New("cannot start"), Stdout: "4.7.1.stable.official.x\n"}, StateUnready},
	} {
		starts := 0
		find := GodotFor(t.TempDir(), "", func(s child.Spec) child.Result {
			starts++
			if !slices.Equal(s.Argv, []string{bin, "--version"}) {
				t.Errorf("argv %v", s.Argv)
			}
			return row.res
		}, nil)
		dir := godotWorld(t, "4.7", false)
		for range 2 {
			if _, pre, note := find(dir); pre != row.pre {
				t.Errorf("%+v: %q %q", row.res, pre, note)
			}
		}
		if starts != 1 {
			t.Errorf("%+v: %d starts", row.res, starts)
		}
	}
}

// The probe takes the smaller of 30 s and what the budget has left; a budget
// with nothing left starts no process, and one that ran out while the binary
// was asked says so, and is asked again rather than remembered.
func TestGodotForSpendsTheBudgetOnTheProbe(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "godot")
	if err := os.WriteFile(bin, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GODOT_BIN", bin)
	dir := godotWorld(t, "4.7", false)
	answer := child.Result{Stdout: "4.7.stable.x\n"}
	for name, row := range map[string]struct {
		left    time.Duration
		res     child.Result
		pre     State
		starts  int
		timeout time.Duration
		note    string
	}{
		"plenty":          {time.Minute, answer, "", 1, 30 * time.Second, ""},
		"as much as 30 s": {30 * time.Second, answer, "", 1, 30 * time.Second, ""},
		"less":            {7 * time.Second, answer, "", 1, 7 * time.Second, ""},
		"none":            {0, answer, StateBudget, 0, 0, "budget is spent"},
		"gone":            {-time.Second, answer, StateBudget, 0, 0, "budget is spent"},
		"ran out":         {7 * time.Second, child.Result{TimedOut: true}, StateBudget, 2, 7 * time.Second, "budget is spent"},
		"its own timeout": {time.Minute, child.Result{TimedOut: true}, StateUnready, 1, 30 * time.Second, "gave no version"},
	} {
		starts, timeout := 0, time.Duration(0)
		find := GodotFor(t.TempDir(), "", func(s child.Spec) child.Result {
			starts++
			timeout = s.Timeout
			return row.res
		}, func() time.Duration { return row.left })
		for range 2 {
			if _, pre, note := find(dir); pre != row.pre || !strings.Contains(note, row.note) {
				t.Errorf("%s: %q %q", name, pre, note)
			}
		}
		if starts != row.starts || timeout != row.timeout {
			t.Errorf("%s: %d starts, timeout %v", name, starts, timeout)
		}
	}
}
