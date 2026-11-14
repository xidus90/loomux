package hooks

import (
	"encoding/json"
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
	if got := stateFileReasons("C:other", "C:state", false, false); len(got) != 0 {
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

// bypassWorld is a project root and a state directory named loomux below
// parent, which LOCALAPPDATA, USERPROFILE and HOME all name; the registry
// exists, config.toml and ui.toml do not, as on a machine without a
// machine-wide configuration yet.
func bypassWorld(t *testing.T) (root, parent, state string) {
	t.Helper()
	parent = t.TempDir()
	state = filepath.Join(parent, "loomux")
	if err := os.MkdirAll(filepath.Join(state, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "registry.toml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"LOCALAPPDATA", "USERPROFILE", "HOME"} {
		t.Setenv(name, parent)
	}
	return t.TempDir(), parent, state
}

func windowsOnly(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("Windows spelling")
	}
}

// stateWrites fails for every writing-tool path that is not refused with the
// state files' reason.
func stateWrites(t *testing.T, root, state string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		for _, tool := range []string{"Write", "Edit"} {
			if got := stateCall(root, state, tool, map[string]any{"file_path": p}); !slices.Contains(got, stateFilesReason) {
				t.Errorf("%s %q: reasons %q, want the state files' reason", tool, p, got)
			}
		}
	}
}

// The removal of a file system root takes the machine-wide files with it:
// the root is a folder above them like any other, though its path already
// ends in a separator.
func TestRemovingTheFileSystemRootIsRefused(t *testing.T) {
	state := t.TempDir()
	top := filepath.VolumeName(state) + string(filepath.Separator)
	if got := stateFileReasons(top, state, true, false); !slices.Equal(got, []string{stateFilesReason}) {
		t.Errorf("remove %q: reasons %q, want the state files' reason", top, got)
	}
	if got := stateFileReasons(top, state, false, false); len(got) != 0 {
		t.Errorf("write %q: reasons %q, want none", top, got)
	}
	line := "rm -rf " + filepath.ToSlash(top)
	if got := stateCall(t.TempDir(), state, "Bash", map[string]any{"command": line}); !slices.Contains(got, stateFilesReason) {
		t.Errorf("%q: reasons %q, want the state files' reason", line, got)
	}
}

// A link onto the folder above the state directory is no link onto a
// machine-wide file, and the write through it names a path that does not
// exist when the guard resolves it.
func TestALinkOntoAFolderAboveTheStateDirectoryIsRefused(t *testing.T) {
	root, parent, state := bypassWorld(t)
	p := filepath.ToSlash(parent)
	lines := map[string]string{
		"Bash#1":     "ln -s " + p + " j && echo x > j/loomux/config.toml",
		"PowerShell": "New-Item -ItemType Junction -Path j -Value " + p + "; Set-Content j/loomux/config.toml x",
	}
	if runtime.GOOS == "windows" {
		lines["Bash#2"] = "cmd /c mklink /J j " + parent + " && echo x > j/loomux/config.toml"
	}
	stateLines(t, root, state, "", config.Policy{}, lines)
	if got := stateCall(root, state, "Bash", map[string]any{"command": "ln -s " + p + "/elsewhere l"}); len(got) != 0 {
		t.Errorf("a link below the folder: reasons %q", got)
	}
}

// A relative link target is resolved from the link's folder, not from the
// root or the shell's folder the guard joins it with.
func TestARelativeLinkTargetIsResolvedFromTheLinksFolder(t *testing.T) {
	root, _, state := bypassWorld(t)
	if err := os.MkdirAll(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(filepath.Join(root, "d"), state)
	if err != nil {
		t.Skip("another volume")
	}
	lines := map[string]string{
		"Bash#1":     "ln -s " + filepath.ToSlash(rel) + "/config.toml d/l && echo x > d/l",
		"PowerShell": "New-Item -ItemType SymbolicLink -Path d/l -Target " + filepath.ToSlash(rel) + "/ui.toml",
	}
	if runtime.GOOS == "windows" {
		lines["Bash#2"] = `cmd /c mklink d\l ` + rel + `\config.toml && echo x > d/l`
	}
	stateLines(t, root, state, "", config.Policy{}, lines)
}

// Win32 drops a trailing dot or space from an element, so these name
// config.toml; while it does not exist yet the resolution keeps the suffix.
func TestATrailingDotOrSpaceNamesTheSameStateFile(t *testing.T) {
	windowsOnly(t)
	root, _, state := bypassWorld(t)
	stateWrites(t, root, state, filepath.Join(state, "config.toml."), filepath.Join(state, "config.toml "), filepath.Join(state, "ui.toml..."))
	stateLines(t, root, state, "", config.Policy{Strict: true}, map[string]string{
		"Bash": "echo x > " + filepath.ToSlash(state) + "/config.toml.",
	})
}

func TestWin32DropsTheDotsAndBlanksThatEndAnElement(t *testing.T) {
	for in, want := range map[string]string{
		`C:\s\config.toml.`:   `C:\s\config.toml`,
		`C:\s\ui.toml...`:     `C:\s\ui.toml`,
		`C:\s\config.toml `:   `C:\s\config.toml`,
		`C:\s.\config.toml`:   `C:\s\config.toml`,
		`C:\s\..\config.toml`: `C:\s\..\config.toml`,
		`C:\s\.\config.toml`:  `C:\s\.\config.toml`,
		"a/b./c":              "a/b/c",
	} {
		if got := win32Name("windows", in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
	if got := win32Name("linux", "a/config.toml."); got != "a/config.toml." {
		t.Errorf("not Windows: %q", got)
	}
}

// ::$DATA is the file's unnamed stream, its content; a shell target loses
// the stream name, a writing tool's path keeps it.
func TestTheUnnamedStreamOfAWritingToolIsTheStateFile(t *testing.T) {
	root, _, state := bypassWorld(t)
	stateWrites(t, root, state, filepath.Join(state, "config.toml::$DATA"), filepath.Join(state, "registry.toml::$DATA"))
}

// \\?\UNC\host\c$\… is the administrative share in device form.
func TestTheDeviceFormOfTheAdministrativeShareIsTheStateFile(t *testing.T) {
	for in, want := range map[string]string{
		`\\?\UNC\localhost\c$\Users\x`: "C:/Users/x",
		`//./unc/127.0.0.1/e$/x`:       "E:/x",
	} {
		if got := driveForms("windows", in); len(got) != 1 || got[0] != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
	if got := driveForms("windows", `\\?\UNC\server\share\x`); got != nil {
		t.Errorf("a share that is no drive: %q", got)
	}
	windowsOnly(t)
	root, _, state := bypassWorld(t)
	unc := `\\?\UNC\localhost\` + strings.ToLower(state[:1]) + "$" + state[2:] + `\config.toml`
	stateWrites(t, root, state, unc)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{"PowerShell": "Set-Content " + unc + " x"})
}

// PowerShell takes a provider-qualified path anywhere it takes a path.
func TestAPowerShellProviderPathIsThePath(t *testing.T) {
	root, _, state := bypassWorld(t)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"PowerShell#1": "Set-Content FileSystem::" + state + `\config.toml x`,
		"PowerShell#2": `Set-Content Microsoft.PowerShell.Core\FileSystem::` + state + `\config.toml x`,
		"PowerShell#3": "Remove-Item FileSystem::" + state + `\registry.toml`,
	})
	stateLines(t, root, state, "", config.Policy{Strict: true}, map[string]string{
		"PowerShell": "Remove-Item filesystem::" + state + `\registry.toml`,
	})
}

// $env:NAME followed by a forward slash is expanded as with a backslash,
// and a Set-Location through it moves the shell there.
func TestAPowerShellVariableBeforeAForwardSlashIsExpanded(t *testing.T) {
	root, _, state := bypassWorld(t)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"PowerShell#1": "Set-Content $env:LOCALAPPDATA/loomux/config.toml x",
		"PowerShell#2": "Set-Content $env:USERPROFILE/loomux/ui.toml x",
		"PowerShell#3": "Remove-Item $env:LOCALAPPDATA/loomux/registry.toml",
		"PowerShell#4": `Set-Location $env:LOCALAPPDATA\loomux; Set-Content config.toml x`,
		"PowerShell#5": `cd $env:LOCALAPPDATA\loomux; Set-Content config.toml x`,
		"PowerShell#6": "Set-Location $env:LOCALAPPDATA/loomux; Remove-Item registry.toml",
	})
}

// Bash's ${NAME:-default} and its relatives expand to the variable's value
// when it is set.
func TestABashParameterExpansionWithAnOperatorIsExpanded(t *testing.T) {
	root, _, state := bypassWorld(t)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash#1": "echo x > ${LOCALAPPDATA:-/tmp}/loomux/config.toml",
		"Bash#2": "echo x > ${LOCALAPPDATA%/}/loomux/config.toml",
		"Bash#3": "rm ${HOME:?}/loomux/registry.toml",
		"Bash#4": "rm ${NOT_SET_ANYWHERE:-" + filepath.ToSlash(state) + "}/registry.toml",
	})
}

func TestEnvironmentVariablesAreExpandedByTheirOperator(t *testing.T) {
	t.Setenv("LX_SET", "/set")
	t.Setenv("LX_EMPTY", "")
	t.Setenv("LX_UNSET", "")
	if err := os.Unsetenv("LX_UNSET"); err != nil {
		t.Fatal(err)
	}
	for in, want := range map[string]string{
		"${LX_SET:-d}":        "/set",
		"${LX_EMPTY:-d}":      "d",
		"${LX_EMPTY-d}":       "",
		"${LX_UNSET:=d}":      "d",
		"${LX_SET:+alt}":      "alt",
		"${LX_EMPTY:+alt}":    "",
		"${LX_UNSET+alt}":     "",
		"${LX_SET%/x}":        "/set",
		"${LX_SET:?}":         "/set",
		"${LX_UNSET:?}":       "${LX_UNSET:?}",
		"$($env:LX_SET)/x":    "/set/x",
		"$( $LX_SET )":        "/set",
		"$(Get-Item $LX_SET)": "$(Get-Item /set)",
		"%LX_SET%/x":          "/set/x",
		"$LX_UNSET/x":         "$LX_UNSET/x",
	} {
		if got := expandEnv(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}

// "$($env:NAME)" is a subexpression around the variable.
func TestAPowerShellSubexpressionAroundAVariableIsExpanded(t *testing.T) {
	root, _, state := bypassWorld(t)
	sep := string(filepath.Separator)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"PowerShell#1": `Set-Content "$($env:LOCALAPPDATA)` + sep + "loomux" + sep + `config.toml" x`,
		"PowerShell#2": `Remove-Item "$($env:LOCALAPPDATA)` + sep + "loomux" + sep + `registry.toml"`,
	})
}

// A glob is expanded by the shell in the host's cwd; the root's own files
// do not stand in for it.
func TestAGlobIsExpandedInTheHostsFolder(t *testing.T) {
	root, _, state := bypassWorld(t)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	stateLines(t, root, state, state, config.Policy{}, map[string]string{
		"Bash#1":     "rm -rf *",
		"Bash#2":     "rm -f regist*",
		"PowerShell": "Remove-Item * -Recurse",
	})
	stateLines(t, root, state, state, config.Policy{Strict: true}, map[string]string{"Bash": "rm -rf *"})
	// A glob that holds no state file, and one at another folder, is free.
	for cwd, line := range map[string]string{state: "rm -rf lo*", root: "rm -rf *"} {
		if got := checkToolOpen(root, state, cwd, "Bash", map[string]any{"command": line}, config.Policy{}, switchOff, switchOff); slices.Contains(got, stateFilesReason) {
			t.Errorf("%q at %q: reasons %q", line, cwd, got)
		}
	}
}

// Named gap, recorded as a pass: a path the shell computes (a command
// substitution, a string concatenation) is no word the guard can read, and
// strict mode does not refuse it either. A cmdlet that joins the path is
// read as a word of the line, and is refused.
func TestAPathTheShellComputesPassesBothModes(t *testing.T) {
	root, _, state := bypassWorld(t)
	computed := map[string]string{
		"Bash":       `printf x > "$(cygpath -u "$LOCALAPPDATA")/loomux/config.toml"`,
		"PowerShell": `Set-Content ($env:LOCALAPPDATA + '/loomux/config.toml') x`,
	}
	for _, policy := range []config.Policy{{}, {Strict: true}} {
		for tool, line := range computed {
			if got := checkToolOpen(root, state, "", tool, map[string]any{"command": line}, policy, switchOff, switchOff); slices.Contains(got, stateFilesReason) {
				t.Errorf("%s %q (strict %v): reasons %q", tool, line, policy.Strict, got)
			}
		}
	}
	stateLines(t, root, state, "", config.Policy{Strict: true}, map[string]string{
		"PowerShell": `Set-Content (Join-Path $env:LOCALAPPDATA 'loomux/config.toml') x`,
	})
}

// cp makes links as ln does: -s and --symbolic-link a symbolic one, -rs a
// tree of them, -l and --link a hard one. ln onto a state file is refused,
// so is cp, whose write through the new link names a path that does not
// exist when the guard resolves it.
func TestACopyThatMakesALinkIsALink(t *testing.T) {
	root, _, state := bypassWorld(t)
	s := filepath.ToSlash(state)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash#1": "cp -s " + s + "/config.toml l",
		"Bash#2": "cp -s " + s + "/config.toml l && echo x > l",
		"Bash#3": "cp --symbolic-link " + s + "/config.toml l && echo x > l",
		"Bash#4": "cp -rs " + s + " j && echo x > j/config.toml",
		"Bash#5": "cp -l " + s + "/registry.toml l && echo x > l",
		"Bash#6": "cp --link " + s + "/registry.toml l && echo x > l",
		"Bash#7": "cp --sym " + s + "/config.toml l",
	})
	for _, line := range []string{"cp -r " + s + "/logs l", "cp -a " + s + "/logs l", "cp --recursive " + s + "/logs l", "cp --sy"} {
		if got := stateCall(root, state, "Bash", map[string]any{"command": line}); len(got) != 0 {
			t.Errorf("%q: reasons %q", line, got)
		}
	}
}

// Expand-Archive takes its destination as the second positional argument,
// or the first after -Path.
func TestExpandArchiveWithAPositionalDestination(t *testing.T) {
	root, _, state := bypassWorld(t)
	s := filepath.ToSlash(state)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"PowerShell#1": "Expand-Archive a.zip " + s,
		"PowerShell#2": "Expand-Archive -Path a.zip " + s,
		"PowerShell#3": "Expand-Archive -LiteralPath a.zip -Force " + s,
	})
	stateLines(t, root, state, state, config.Policy{}, map[string]string{
		"PowerShell": "Expand-Archive a.zip .",
	})
}

// An extractor without a destination writes into the shell's folder; after a
// cd into the state directory, or with the host's cwd there, that is the state
// directory, which counts for any write when it is named.
func TestExtractionIntoTheShellsFolder(t *testing.T) {
	root, _, state := bypassWorld(t)
	s := filepath.ToSlash(state)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash#1": "cd " + s + " && tar -xf a.tar",
		"Bash#2": "cd $LOCALAPPDATA/loomux && unzip -o a.zip",
		"Bash#3": "cd ~/loomux && 7z x a.zip",
		"Bash#4": "cd " + s + " && cpio -i < a.cpio",
		"Bash#5": "7z x a.zip -o" + s,
		"Bash#6": "cpio -id -D " + s + " < a.cpio",
		"Bash#7": "find . | cpio -pd " + s,
	})
	stateLines(t, root, state, state, config.Policy{}, map[string]string{
		"Bash#1": "unzip a.zip",
		"Bash#2": "tar -xf a.tar",
		"Bash#3": "7z x a.zip",
		"Bash#4": "bsdtar xf a.tar",
	})
	for _, line := range []string{"unzip -l a.zip", "tar -tf a.tar", "7z l a.zip", "tar -xf a.tar -C sub"} {
		if got := checkToolOpen(root, state, state, "Bash", map[string]any{"command": line}, config.Policy{}, switchOff, switchOff); len(got) != 0 {
			t.Errorf("%q from the state directory: reasons %q", line, got)
		}
	}
}

// A decompressor writes the name it is given without its suffix, so
// config.toml.gz names config.toml without spelling it; a compressor replaces
// its file, which removes it.
func TestADecompressorWritesTheNameWithoutItsSuffix(t *testing.T) {
	root, _, state := bypassWorld(t)
	s := filepath.ToSlash(state)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash#1": "gzip -d " + s + "/config.toml.gz",
		"Bash#2": "gunzip " + s + "/config.toml.gz",
		"Bash#3": "gzip -dk " + s + "/ui.json.gz",
		"Bash#4": "bunzip2 " + s + "/config.toml.bz2",
		"Bash#5": "xz -d " + s + "/config.toml.xz",
		"Bash#6": "xz --decompress " + s + "/config.toml.xz",
		"Bash#7": "zstd -d a.zst -o " + s + "/config.toml",
		"Bash#8": "gzip " + s + "/config.toml",
	})
	for _, line := range []string{"gzip -dc " + s + "/config.toml.gz", "gzip -k " + s + "/config.toml", "xz --stdout " + s + "/config.toml"} {
		if got := stateCall(root, state, "Bash", map[string]any{"command": line}); len(got) != 0 {
			t.Errorf("%q: reasons %q", line, got)
		}
	}
}

// git's --output writes the command's output to the file it names.
func TestGitOutputOptionWritesAMachineWideFile(t *testing.T) {
	root, _, state := bypassWorld(t)
	s := filepath.ToSlash(state)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash#1": "git show HEAD:x --output=" + s + "/config.toml",
		"Bash#2": "git show HEAD:x --output " + s + "/config.toml",
		"Bash#3": "git diff --output=" + s + "/ui.json",
		"Bash#4": "git log -p --output=" + s + "/registry.toml",
		"Bash#5": "git archive -o " + s + "/config.toml HEAD",
	})
}

// git checks a path out below the work tree it is pointed at; -C or
// --work-tree on the state directory makes config.toml a state file.
func TestGitCheckoutIntoTheStateDirectory(t *testing.T) {
	root, _, state := bypassWorld(t)
	s := filepath.ToSlash(state)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash#1": "git -C " + s + " checkout x -- config.toml",
		"Bash#2": "git --work-tree=" + s + " checkout HEAD -- config.toml",
		"Bash#3": "git --work-tree " + s + " restore config.toml -- ui.json",
		"Bash#4": "cd " + s + " && git -C . checkout x -- config.toml",
		"Bash#5": "git -C " + s + " rm registry.toml",
	})
	if got := stateCall(root, state, "Bash", map[string]any{"command": "git -C " + s + "/logs checkout x -- a.log"}); len(got) != 0 {
		t.Errorf("a folder below the state directory: reasons %q", got)
	}
}

// A write form of an archiver or compressor that the verb table does not
// read is still refused in strict mode, where the path it names is judged
// for every program the table does not fully know.
func TestStrictModeRefusesAnArchiverWriteFormTheTableDoesNotRead(t *testing.T) {
	root, _, state := bypassWorld(t)
	s := filepath.ToSlash(state)
	for _, line := range []string{
		"cpio -o -O " + s + "/config.toml < list",
		"cpio -o -F " + s + "/config.toml",
		"zstd -d a.zst --output-dir-flat=" + s,
		"bsdtar -xf a.tar --one-top-level=" + s,
		"gtar -xf a.tar --one-top-level=" + s,
		"7z x a.7z -spf2 -o" + s,
		"gzip -d --suffix=.x " + s + "/config.toml",
	} {
		got := checkToolOpen(root, state, "", "Bash", map[string]any{"command": line}, config.Policy{Strict: true}, switchOff, switchOff)
		if len(got) == 0 {
			t.Errorf("strict %q: no reasons", line)
		}
	}
}

// Bash's ANSI-C quoting $'…' and locale quoting $"…" are quotes around a
// literal word; the shell writes the file they spell. The guard keeps the $
// and takes the word for an expansion.
func TestBypassAnsiCQuotedPathWritesAStateFile(t *testing.T) {
	root, _, state := bypassWorld(t)
	s := filepath.ToSlash(state)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash#1": "echo x > $'" + s + "/config.toml'",
		"Bash#2": "echo x > $\"" + s + "/config.toml\"",
		"Bash#3": "rm $'" + s + "/registry.toml'",
		"Bash#4": "rm $'" + s + "'/registry.toml",
		"Bash#5": "echo x > " + s + "/$'config.toml'",
		"Bash#6": "echo x | tee " + s + "/$'ui.toml'",
	})
	stateLines(t, root, state, "", config.Policy{Strict: true}, map[string]string{
		"Bash": "echo x > $'" + s + "/config.toml'",
	})
}

// git config --file writes the file it names, as git's --output does.
func TestBypassGitConfigFileWritesAStateFile(t *testing.T) {
	root, _, state := bypassWorld(t)
	s := filepath.ToSlash(state)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash#1": "git config --file " + s + "/config.toml a.b c",
		"Bash#2": "git config -f " + s + "/registry.toml a.b c",
		"Bash#3": "git config --file=" + s + "/config.toml a.b c",
		"Bash#4": "git config --file " + s + "/registry.toml --unset a.b",
		"Bash#5": "git -C " + s + " config --file config.toml a.b c",
		"Bash#6": "cd " + s + " && git config -f ui.toml a.b c",
	})
}

// The full hook, write barrier included, lets git config --file onto the
// registry through.
func TestBypassGitConfigFilePassesTheWholeHook(t *testing.T) {
	root, state := world(t, "")
	line := "git config --file " + filepath.ToSlash(filepath.Join(state, "registry.toml")) + " area.llm_config true"
	b, err := json.Marshal(map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": line}})
	if err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := call(t, root, state, string(b)); code != 2 || !strings.Contains(errOut, stateFilesReason) {
		t.Errorf("%q: code %d, err %q", line, code, errOut)
	}
}

// env -C and sudo -D move the program they run; a shell started there writes
// a relative name into the state directory.
func TestBypassEnvChdirIntoANestedShell(t *testing.T) {
	root, _, state := bypassWorld(t)
	s := filepath.ToSlash(state)
	lines := map[string]string{
		"Bash#1": "env -C " + s + " sh -c 'echo x > config.toml'",
		"Bash#2": "env -C " + s + " bash -c 'rm registry.toml'",
		"Bash#3": "env --chdir=" + s + " sh -c 'echo x > ui.toml'",
		"Bash#4": "sudo -D " + s + " sh -c 'echo x > config.toml'",
	}
	stateLines(t, root, state, "", config.Policy{}, lines)
	stateLines(t, root, state, "", config.Policy{Strict: true}, lines)
}

// pwsh -WorkingDirectory starts the nested shell in the state directory.
func TestBypassPwshWorkingDirectory(t *testing.T) {
	root, _, state := bypassWorld(t)
	lines := map[string]string{
		"PowerShell#1": "pwsh -wd " + state + " -c \"Set-Content config.toml x\"",
		"PowerShell#2": "pwsh -WorkingDirectory " + state + " -c \"Remove-Item registry.toml\"",
	}
	stateLines(t, root, state, "", config.Policy{}, lines)
	stateLines(t, root, state, "", config.Policy{Strict: true}, lines)
}

// GNU time -o writes its report into the file it names.
func TestBypassTimeOutputFile(t *testing.T) {
	root, _, state := bypassWorld(t)
	s := filepath.ToSlash(state)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash#1": "/usr/bin/time -o " + s + "/config.toml true",
		"Bash#2": "/usr/bin/time --output=" + s + "/config.toml true",
	})
}

// Cmdlets that write the file at their path, siblings of Out-File and
// Invoke-WebRequest -OutFile, which are refused.
func TestBypassPowerShellCmdletsThatWriteTheirPath(t *testing.T) {
	root, _, state := bypassWorld(t)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"PowerShell#1": "Get-Process | Export-Csv -Path " + state + "\\config.toml",
		"PowerShell#2": "Get-Process | Export-Csv " + state + "\\config.toml",
		"PowerShell#3": "Get-Process | Export-Csv -LiteralPath " + state + "\\config.toml",
		"PowerShell#4": "Get-Process | Export-Clixml " + state + "\\config.toml",
		"PowerShell#5": "Start-Transcript " + state + "\\config.toml",
		"PowerShell#6": "Start-Transcript -Path " + state + "\\ui.toml",
		"PowerShell#7": "Invoke-RestMethod http://x -OutFile " + state + "\\config.toml",
		"PowerShell#8": "Start-BitsTransfer http://x " + state + "\\config.toml",
	})
}

// fsutil makes a hard link as mklink /H does, and the write through it
// names a path that does not exist when the guard resolves it.
func TestBypassFsutilHardLinkOntoAStateFile(t *testing.T) {
	windowsOnly(t)
	root, _, state := bypassWorld(t)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash":       "fsutil hardlink create l " + state + "\\registry.toml && echo x > l",
		"PowerShell": "fsutil hardlink create l " + state + "\\registry.toml; Set-Content l x",
	})
}

// subst and New-PSDrive give the state directory a drive letter, a second
// name as a junction does; the write through it names a drive that does not
// exist when the guard resolves it.
func TestBypassDriveLetterOntoTheStateDirectory(t *testing.T) {
	windowsOnly(t)
	root, _, state := bypassWorld(t)
	stateLines(t, root, state, "", config.Policy{}, map[string]string{
		"Bash":         "subst Q: " + state + " && echo x > Q:/config.toml",
		"PowerShell#1": "subst Q: " + state + "; Set-Content Q:\\config.toml x",
		"PowerShell#2": "New-PSDrive -Name Q -PSProvider FileSystem -Root " + state + "; Set-Content Q:\\config.toml x",
	})
}

// The same holes through spellings that need no Windows path, and the
// neighbours of each that name no state file.
func TestBypassSiblingSpellings(t *testing.T) {
	root, _, state := bypassWorld(t)
	s := filepath.ToSlash(state)
	lines := map[string]string{
		"Bash#1":       "env -C " + s + " sh <<< 'echo x > config.toml'",
		"Bash#2":       "fsutil hardlink create l " + s + "/registry.toml && echo x > l",
		"Bash#3":       "subst Q: " + s,
		"Bash#4":       "mount -o bind " + s + " /mnt/x",
		"Bash#5":       "time -o " + s + "/ui.toml true",
		"Bash#6":       "env -C " + s + " time -ao config.toml true",
		"Bash#7":       "echo x > $'" + s + "/\\x63onfig.toml'",
		"Bash#8":       "echo x > $'" + s + "/\\143onfig.toml'",
		"Bash#9":       "echo x > $'" + s + "/\\u0063onfig.toml'",
		"PowerShell#1": "Start-BitsTransfer -Source http://x -Destination " + s + "/config.toml",
		"PowerShell#2": "ndr Q FileSystem " + s,
		"PowerShell#3": "pwsh -WorkingDirectory:" + s + " -c \"Set-Content config.toml x\"",
		"PowerShell#4": "Get-Process | Export-Alias " + s + "/ui.toml",
	}
	stateLines(t, root, state, "", config.Policy{}, lines)
	stateLines(t, root, state, "", config.Policy{Strict: true}, lines)
	for _, line := range []string{
		"fsutil volume diskfree C:",
		"fsutil hardlink list " + s + "/registry.toml",
		"Start-BitsTransfer http://x",
		"pwsh -wd",
		"git config --file other.cfg a.b c",
		"echo $'a\\x' $HOME",
		"time -o out.txt true",
	} {
		if got := stateCall(root, state, "Bash", map[string]any{"command": line}); slices.Contains(got, stateFilesReason) {
			t.Errorf("%q: refused with the state files' reason", line)
		}
	}
}
