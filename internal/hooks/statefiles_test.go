package hooks

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// stateCall judges one tool call at root with stateDir as the state directory.
func stateCall(root, stateDir, tool string, input map[string]any) []string {
	return checkToolOpen(root, stateDir, "", tool, input, config.Policy{}, switchOff, switchOff)
}

func TestTheGuardRefusesAgentWritesToTheMachineWideFiles(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	slash := filepath.ToSlash(state)
	for _, name := range []string{"config.toml", "ui.toml", "ui.json", "registry.toml"} {
		file := filepath.Join(state, name)
		for _, tool := range []string{"Write", "Edit"} {
			if got := stateCall(root, state, tool, map[string]any{"file_path": file}); !slices.Equal(got, []string{stateFilesReason}) {
				t.Errorf("%s %s: reasons %q", tool, name, got)
			}
		}
	}
	for _, line := range []string{
		"echo x > " + slash + "/config.toml",
		"rm " + slash + "/config.toml",
		"rm -r " + slash,
		"mv " + slash + "/config.toml x",
	} {
		if got := stateCall(root, state, "Bash", map[string]any{"command": line}); !slices.Equal(got, []string{stateFilesReason}) {
			t.Errorf("%q: reasons %q", line, got)
		}
	}
	if got := stateCall(root, state, "PowerShell", map[string]any{"command": "Set-Content " + slash + "/config.toml x"}); !slices.Equal(got, []string{stateFilesReason}) {
		t.Errorf("Set-Content: reasons %q", got)
	}
}

func TestTheGuardLeavesTheRestOfTheStateDirectoryOpen(t *testing.T) {
	root, state, elsewhere := t.TempDir(), t.TempDir(), t.TempDir()
	for _, file := range []string{
		filepath.Join(state, "logs", "serve.log"),
		filepath.Join(state, "config", "proposals", "x.json"),
		filepath.Join(elsewhere, "config.toml"),
	} {
		if got := stateCall(root, state, "Write", map[string]any{"file_path": file}); len(got) != 0 {
			t.Errorf("%s: reasons %q", file, got)
		}
	}
	if got := stateCall(root, state, "Bash", map[string]any{"command": "rm -r " + filepath.ToSlash(filepath.Join(state, "logs"))}); len(got) != 0 {
		t.Errorf("rm of a sibling folder: reasons %q", got)
	}
	// A removal is the only thing that reaches a file through its folder.
	if got := stateCall(root, state, "Bash", map[string]any{"command": "echo x > " + filepath.ToSlash(filepath.Join(state, "logs", "a"))}); len(got) != 0 {
		t.Errorf("write below the state directory: reasons %q", got)
	}
	if got := stateCall(root, "", "Write", map[string]any{"file_path": filepath.Join(state, "config.toml")}); len(got) != 0 {
		t.Errorf("no state directory: reasons %q", got)
	}
}

// A path through a link, or in another case where the file system ignores
// case, is the same file.
func TestTheMachineWideFilesAreFoundThroughOtherSpellings(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	if runtime.GOOS == "windows" {
		upper := filepath.Join(strings.ToUpper(state), "CONFIG.TOML")
		if got := stateCall(root, state, "Write", map[string]any{"file_path": upper}); !slices.Equal(got, []string{stateFilesReason}) {
			t.Errorf("%s: reasons %q", upper, got)
		}
	}
	link := filepath.Join(t.TempDir(), "link")
	if runtime.GOOS == "windows" {
		mklink(t, link, state)
	} else if err := os.Symlink(state, link); err != nil {
		t.Fatal(err)
	}
	if got := stateCall(root, state, "Write", map[string]any{"file_path": filepath.Join(link, "ui.toml")}); !slices.Equal(got, []string{stateFilesReason}) {
		t.Errorf("through a link: reasons %q", got)
	}
}

// The hook hands its state directory to the policy: a write to the registry
// is refused with the policy's reason, not only by the barrier behind it.
func TestThePreToolHookGuardsItsStateDirectory(t *testing.T) {
	root, state := world(t, "")
	code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(state, "ui.toml")))
	if code != 2 || !strings.Contains(errOut, stateFilesReason) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// stateLines judges each shell line as its tool at root, with the host's
// cwd, in the mode policy names, and fails for every one that is not refused
// with the state files' reason. A key may end in #n to repeat a tool.
func stateLines(t *testing.T, root, state, cwd string, policy config.Policy, lines map[string]string) {
	t.Helper()
	for tool, line := range lines {
		tool, _, _ = strings.Cut(tool, "#")
		got := checkToolOpen(root, state, cwd, tool, map[string]any{"command": line}, policy, switchOff, switchOff)
		if !slices.Contains(got, stateFilesReason) {
			t.Errorf("%s %q (cwd %q): reasons %q, want the state files' reason", tool, line, cwd, got)
		}
	}
}

// The shell stands where the host's cwd says, not at the root: a relative
// name there is the state directory's file.
func TestARelativeNameIsJudgedFromTheHostsFolder(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	stateLines(t, root, state, state, config.Policy{}, map[string]string{
		"Bash":       "echo x > config.toml",
		"PowerShell": "Set-Content config.toml x",
		"Bash#2":     "cd logs && echo x > ../ui.toml",
	})
	if got := stateCall(root, state, "Bash", map[string]any{"command": "echo x > config.toml"}); len(got) != 0 {
		t.Errorf("a shell without a cwd: reasons %q", got)
	}
	if got := checkToolOpen(root, state, root, "Bash", map[string]any{"command": "echo x > config.toml"}, config.Policy{}, switchOff, switchOff); len(got) != 0 {
		t.Errorf("a shell at another folder: reasons %q", got)
	}
}

// The hook hands the payload's cwd to the policy.
func TestThePreToolHookReadsTheHostsFolder(t *testing.T) {
	root, state := world(t, "")
	input := `{"tool_name":"Bash","cwd":` + quote(state) + `,"tool_input":{"command":"echo x > ui.toml"}}`
	if code, _, errOut := call(t, root, state, input); code != 2 || !strings.Contains(errOut, stateFilesReason) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// The state directory is usually spelled through an environment variable or
// the home folder, which the shell expands and the line does not hold.
func TestTheStateDirectoryIsFoundThroughTheEnvironment(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	parent, name := filepath.Dir(state), filepath.Base(state)
	t.Setenv("LOCALAPPDATA", parent)
	t.Setenv("HOME", parent)
	t.Setenv("USERPROFILE", parent)
	sep := string(filepath.Separator)
	lines := map[string]string{
		"Bash#1":       "echo x > $LOCALAPPDATA/" + name + "/config.toml",
		"Bash#2":       "echo x > ${LOCALAPPDATA}/" + name + "/config.toml",
		"Bash#3":       "echo x > ~/" + name + "/config.toml",
		"Bash#4":       "echo x > $HOME/" + name + "/config.toml",
		"PowerShell#1": "Set-Content $env:LOCALAPPDATA" + sep + name + sep + "config.toml x",
		"PowerShell#2": `Set-Content "$env:LOCALAPPDATA` + sep + name + sep + `config.toml" x`,
		"PowerShell#3": "Set-Content ~" + sep + name + sep + "config.toml x",
		"PowerShell#4": "Set-Content ${env:LOCALAPPDATA}" + sep + name + sep + "config.toml x",
	}
	if runtime.GOOS == "windows" {
		lines["Bash#5"] = "cmd /c \"echo x > %LOCALAPPDATA%\\" + name + "\\config.toml\""
	}
	stateLines(t, root, state, "", config.Policy{}, lines)
	// A variable nobody set names nothing the guard can follow, and a ~user
	// is no home folder of ours.
	for _, line := range []string{"echo x > $NOT_SET_ANYWHERE/config.toml", "echo x > ~" + name + "/config.toml"} {
		if got := stateCall(root, state, "Bash", map[string]any{"command": line}); len(got) != 0 {
			t.Errorf("%q: reasons %q", line, got)
		}
	}
}

func TestTheHomeFolderIsOnlyAnExpansionWhereTheGuardKnowsOne(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	if got := homeForms("~/x"); len(got) != 0 {
		t.Errorf("no home folder: %q", got)
	}
	if got := homeForms("~x/y"); got != nil {
		t.Errorf("~x: %q", got)
	}
	t.Setenv("HOME", "/h")
	if got := homeForms("~"); !slices.Contains(got, "/h") {
		t.Errorf("~ alone: %q", got)
	}
}

// Git Bash names C:\ as /c/, and Windows reaches a local drive through its
// administrative share; other systems have neither.
func TestTheStateDirectoryIsFoundThroughOtherDriveSpellings(t *testing.T) {
	if got := driveForms("linux", "/c/x"); got != nil {
		t.Errorf("not Windows: %q", got)
	}
	for in, want := range map[string]string{
		"/c/Users/x":             "C:/Users/x",
		"/cygdrive/d":            "D:/",
		"//localhost/c$/Users/x": "C:/Users/x",
		`\\127.0.0.1\e$\x`:       "E:/x",
		`\\?\C:\Users\x`:         "C:/Users/x",
		"//./C:/Users/x":         "C:/Users/x",
		"relative/c/x":           "",
		"//server/share/Users/x": "",
		"C:/Users/x":             "",
		"/cygdrive/cc/x":         "",
	} {
		got := driveForms("windows", in)
		if (want == "") != (got == nil) || (want != "" && got[0] != want) {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
	if runtime.GOOS != "windows" {
		return
	}
	root, state := t.TempDir(), t.TempDir()
	s := filepath.ToSlash(state)
	drive := strings.ToLower(s[:1])
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash#1":       "echo x > /" + drive + s[2:] + "/config.toml",
		"Bash#2":       "echo x > //localhost/" + drive + "$" + s[2:] + "/config.toml",
		"PowerShell#1": `Set-Content \\?\` + state + `\config.toml x`,
	})
	// Drive-relative paths do not resolve; both then have the key of the
	// empty path, which is no state directory.
	if got := stateFileReasons("C:other", "C:state", false); len(got) != 0 {
		t.Errorf("unresolvable: reasons %q", got)
	}
}

// A link the line makes does not exist when the guard resolves the path
// written through it, so the guard refuses the link onto a machine-wide file.
func TestALinkOntoAMachineWideFileIsRefused(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	s := filepath.ToSlash(state)
	lines := map[string]string{
		"Bash#1":       "ln -s " + s + "/config.toml l && echo x > l",
		"Bash#2":       "ln " + s + "/config.toml l && echo x > l",
		"Bash#3":       "ln -sf " + s + " j && echo x > j/config.toml",
		"PowerShell#1": "New-Item -ItemType SymbolicLink -Path l -Target " + s + "/ui.toml",
		"PowerShell#2": "New-Item -ItemType Junction -Path j -Value " + s,
		"Bash#4":       "cmd /c mklink /J j " + state + " && echo x > j/config.toml",
		"Bash#5":       "cmd /c mklink /H l " + filepath.Join(state, "registry.toml"),
	}
	stateLines(t, root, state, "", config.Policy{}, lines)
	// Any other link is nobody's business here.
	for _, line := range []string{
		"ln -s README.md l",
		"ln -s " + s + "/logs l",
		"New-Item -ItemType SymbolicLink -Path l -Target README.md",
		"cmd /c mklink l README.md",
		"cmd /c mklink",
	} {
		if got := stateCall(root, state, "Bash", map[string]any{"command": line}); len(got) != 0 {
			t.Errorf("%q: reasons %q", line, got)
		}
	}
}

// A program that is given the state directory to write below names no file
// the guard can see, so the folder itself counts.
func TestAnArchiveOrACopyIntoTheStateDirectoryIsRefused(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	s := filepath.ToSlash(state)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash#1":       "tar -xf a.tar -C " + s,
		"Bash#2":       "unzip -o a.zip -d " + s,
		"Bash#3":       "cp -r staged " + s,
		"PowerShell#1": "Expand-Archive a.zip -DestinationPath " + s,
	})
	for _, line := range []string{"tar -xf a.tar -C " + s + "/logs", "unzip a.zip -d " + s + "/logs", "cp x " + s + "/logs/y"} {
		if got := stateCall(root, state, "Bash", map[string]any{"command": line}); len(got) != 0 {
			t.Errorf("%q: reasons %q", line, got)
		}
	}
}

// Named gap, recorded as a pass: a program that opens the file itself is
// not read in the default mode, so python -c passes there. Strict mode
// refuses an unknown program that names a machine-wide file; a script that
// does not name it passes in both.
func TestAProgramThatOpensAMachineWideFileItselfPassesTheDefaultMode(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	line := `python -c "open(r'` + filepath.Join(state, "config.toml") + `','w')"`
	if got := stateCall(root, state, "Bash", map[string]any{"command": line}); len(got) != 0 {
		t.Errorf("default mode: reasons %q", got)
	}
	strict := config.Policy{Strict: true}
	if got := checkToolOpen(root, state, "", "Bash", map[string]any{"command": line}, strict, switchOff, switchOff); len(got) == 0 {
		t.Error("strict mode passes a program that names a machine-wide file")
	}
	if got := checkToolOpen(root, state, "", "Bash", map[string]any{"command": "python tools/w.py"}, strict, switchOff, switchOff); len(got) != 0 {
		t.Errorf("a script that names nothing: reasons %q", got)
	}
}

// PowerShell's $env:NAME is a variable, not a stream: the colon stays.
func TestAnEnvironmentVariableIsNoStreamName(t *testing.T) {
	for in, want := range map[string]string{
		"$env:X/a":      "$env:X/a",
		"${ENV:X}/a:s":  "${ENV:X}/a",
		"a.toml:backup": "a.toml",
		"$env:":         "$env:",
	} {
		if got := withoutStream(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}
