package hooks

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// The local web interface writes configuration behind a login that only the
// terminal which started it knows. An agent that starts it, restarts it or
// stops it in any spelling would hand that login to nobody, so every spelling
// of a loomux call that hides a configuration write hides this one too.
func TestAnAgentMayNotRunTheInterfaceInAnySpelling(t *testing.T) {
	for _, row := range loomuxSpellings() {
		line := inPlaceOfCommand(t, row, "ui", `u\i`, "u`i", "u^i")
		if !writesConfiguration(line, false) {
			t.Errorf("%q passes (from %q)", line, row)
		}
	}
}

func TestTheInterfaceStaysReservedForAHumanInEveryForm(t *testing.T) {
	for _, line := range []string{
		"loomux ui",
		"loomux ui --no-browser",
		"loomux ui stop",
		"loomux ui --foreground",
		"loomux ui --foreground < code.txt",
		"loomux ui --help --no-browser",
		"loomux ui -h stop",
		"loomux ui --",
		"go run ./cmd/loomux ui",
		"go run ./cmd/loomux ui --no-browser",
		"go run -tags x ./cmd/loomux ui stop",
		"./bin/loomux.exe ui",
		`.\bin\loomux.exe ui`,
		`& "$env:LOCALAPPDATA\loomux\dev\loomux\loomux.exe" ui`,
		`& "$env:LOCALAPPDATA\loomux\dev\loomux\loomux.exe" ui --no-browser`,
		`"${LOCALAPPDATA}/loomux/dev/loomux/loomux.exe" ui`,
		`'C:\Users\x\AppData\Local\loomux\bin\loomux.exe' ui`,
		`Start-Process loomux -ArgumentList ui`,
		`Start-Process -FilePath loomux.exe -ArgumentList "ui","--no-browser" -WindowStyle Hidden`,
		`start loomux ui`,
		`saps bin\loomux.exe ui`,
		`cmd /c loomux ui`,
		`cmd /c "loomux ui"`,
		`cmd.exe /c start "" loomux ui`,
		`powershell -Command "loomux ui"`,
		`pwsh -c "& loomux ui"`,
		`sh -c "loomux ui"`,
		`bash -c 'go run ./cmd/loomux ui'`,
		`echo ok && loomux ui`,
		`loomux version; loomux ui`,
		`loomux version & loomux ui`,
		`loomux version || loomux ui`,
		`loomux version | loomux ui`,
		`(loomux ui)`,
		`{ loomux ui; }`,
		`$(loomux ui)`,
		"`loomux ui`",
		`nohup loomux ui &`,
		`setsid loomux ui`,
		`env LOOMUX_STATE_DIR=x loomux ui`,
		`LOOMUX_STATE_DIR=x loomux ui`,
		`$env:LOOMUX_STATE_DIR="x"; loomux ui`,
		`time loomux ui`,
		`sudo loomux ui`,
		`exec loomux ui`,
		`command loomux ui`,
		`xargs loomux ui`,
		`cd x && loomux ui`,
		"loomux \\\nui",
		"loomux `\nui",
		"cd x\nloomux ui",
		`l=loomux; $l ui`,
		`alias l=loomux; l ui`,
		`U=ui; loomux $U`,
		`loomux u""i`,
		`loomux 'u'i`,
		`loomux "u"'i'`,
		`loomux u\i`,
		"loomux u`i",
		"loomux u^i",
		`LOOMUX.EXE ui`,
		`Loomux ui`,
		`loomux ui;`,
		`loomux ui # --help`,
		`loomux ui --help stop`,
		`try { loomux ui } catch {}`,
		`if ($true) { loomux ui }`,
		`foreach ($i in 1) { loomux ui }`,
		`function f { loomux ui }; f`,
		`echo "x" > out.txt; loomux ui`,
	} {
		if !writesConfiguration(line, false) {
			t.Errorf("must refuse %q", line)
		}
	}
}

func TestTheInterfaceHelpAndOtherCommandsStayOpen(t *testing.T) {
	for _, line := range []string{
		"loomux ui --help",
		"loomux ui -h",
		"go run ./cmd/loomux ui --help",
		"loomux status",
		"loomux version",
		"echo loomux ui",
		"grep -r ui internal",
		"git log --oneline -- internal/web/ui.go",
		"go test ./internal/web",
		"ls ui",
	} {
		if writesConfiguration(line, false) {
			t.Errorf("must allow %q", line)
		}
	}
}

// The switch that lets an agent edit its own project's configuration does not
// open the interface: that one reaches the machine and every project.
func TestTheInterfaceStaysClosedWhenTheSwitchIsOn(t *testing.T) {
	root := t.TempDir()
	for _, line := range []string{
		"loomux ui",
		"loomux ui stop",
		"loomux ui --yes",
		"loomux ui --no-browser --yes",
		"./bin/loomux.exe ui",
		"go run ./cmd/loomux ui",
		"loomux config set a b --yes && loomux ui",
	} {
		if got := shellReasonsOpen(t, root, line, switchOn, atRoot); !slices.Contains(got, configurationReason) {
			t.Errorf("%q, switch on: reasons %q, want the configuration's", line, got)
		}
	}
}

func TestTheRefusalOfTheInterfaceNamesIt(t *testing.T) {
	for _, tool := range []string{"Bash", "PowerShell"} {
		got := checkTool(t.TempDir(), tool, map[string]any{"command": "loomux ui"}, config.Policy{})
		if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "config, ui and area add") }) {
			t.Errorf("%s: reasons %v, want the one that names ui", tool, got)
		}
	}
}

// mustRefuseUI fails for every line the configuration rule lets through,
// although each of them starts the local web interface.
func mustRefuseUI(t *testing.T, lines ...string) {
	t.Helper()
	for _, line := range lines {
		if !writesConfiguration(line, false) {
			t.Errorf("passes, yet runs loomux ui: %q", line)
		}
	}
}

// mustAllow fails for every line the configuration rule refuses.
func mustAllow(t *testing.T, lines ...string) {
	t.Helper()
	for _, line := range lines {
		if writesConfiguration(line, false) {
			t.Errorf("must allow %q", line)
		}
	}
}

// go accepts more spellings of the main package than its plain path: a folder
// below it, a glob, a doubled slash, and a folder chosen with -C.
func TestGoRunOfTheMainPackageInAnotherSpelling(t *testing.T) {
	mustRefuseUI(t,
		"go run ./cmd/loomux/... ui",
		"go run ./cmd/loomux/. ui",
		"go run ./cmd//loomux ui",
		"go run ./cmd/... ui",
		"go run ./... ui",
		"go run -C cmd/loomux . ui",
		"go run -C=cmd/loomux . ui",
		"go -C cmd/loomux run . ui",
		"go -C cmd -C loomux run main.go ui",
		"go run cmd/loomux/m*.go ui",
		"go run cmd/loomux/*.go ui",
		`go run .\cmd\loomux\... ui`,
	)
	mustAllow(t, "go run ./cmd/other/... ui", "go run ./mycmd/loomux/. ui", "go run -C cmd/other . ui",
		"go run cmd/other/m*.go ui", "go run -C", "go -C cmd/loomux build . ui")
}

// A shell standing in cmd/loomux (the hook payload's cwd, or a cd on the
// line) runs loomux as `go run .`.
func TestGoRunDotFromInsideTheMainPackage(t *testing.T) {
	mustRefuseUI(t,
		"cd cmd/loomux; go run . ui",
		"cd cmd/loomux && go run . ui",
		"cd cmd; cd loomux; go run . ui",
		"pushd ./cmd/loomux/ && go run main.go ui",
		`Set-Location "C:\x\cmd\loomux"; go run . ui`,
	)
	mustAllow(t, "go run . ui", "cd cmd/other && go run . ui")
	root, state := world(t, "")
	main := filepath.Join(root, "cmd", "loomux")
	if code, _, errOut := call(t, root, state, shellAt("go run . ui", main)); code != 2 {
		t.Errorf("go run . ui from %q: code %d, err %q", main, code, errOut)
	}
	if code, _, errOut := call(t, root, state, shellAt("go run . ui", root)); code != 0 {
		t.Errorf("go run . ui from the root: code %d, err %q", code, errOut)
	}
}

// bash turns each of these words into `ui` before loomux sees it.
func TestBashExpansionsSpellTheSubcommand(t *testing.T) {
	mustRefuseUI(t,
		`loomux $'ui'`,
		`loomux $"ui"`,
		`loomux u$()i`,
		`loomux u${x}i`,
		`loomux u$@i`,
		`loomux {ui,}`,
		`loomux $(echo ui)`,
		"loomux `echo ui`",
		`touch ui; loomux u?`,
		`touch ui; loomux u[i]`,
		`go run ./cmd/loomux $(echo ui)`,
		`env X=1 loomux "$sub"`,
	)
}

// PowerShell computes the argument: a subexpression, an array or a splat.
func TestPowerShellExpressionsSpellTheSubcommand(t *testing.T) {
	mustRefuseUI(t,
		`loomux ('u'+'i')`,
		`loomux ("u"+"i")`,
		`loomux $("ui")`,
		`loomux (echo ui)`,
		`loomux @('ui')`,
		`$a=@('ui'); loomux @a`,
	)
}

// Only the word that names the subcommand is read as text: a computed word
// later in the call, or a name that only holds loomux, hides nothing.
func TestAnArgumentAfterTheSubcommandMayBeComputed(t *testing.T) {
	mustAllow(t,
		"loomux status $(pwd)",
		"loomux version --root $PWD",
		`loomux status "$x"`,
		"ls cmd/loomux",
		"try{ loomux con`fig list }",
	)
}

// The text rules read loomux where a program stands, so a loomux that is an
// operand of another program, or a launcher a reader names, hides nothing.
func TestLoomuxAsAnOperandOfAnotherProgramIsNoCall(t *testing.T) {
	mustAllow(t,
		"grep -r loomux *.go",
		"rg loomux internal/*",
		"grep -rn loomux docs/*.md",
		"echo loomux $HOME",
		`git commit -m "loomux $x"`,
		"ls ~/loomux ~",
		"rg -n wmic ~/src/loomux",
		"FOO=loomux ls $x",
		"cat loomux.exe $x",
		"rg loomux ~/x",
		"grep loomux $f",
		"grep -rn loomux *.go",
		"ls cmd/loomux *",
	)
	mustRefuseUI(t,
		"sudo loomux $x",
		"env A=1 loomux $x",
		"cd x && time loomux $(echo ui)",
		`& "C:\bin\loomux.exe" $x`,
		`if ($a) { loomux $x }`,
		"cmd /c wmic process call create 'loomux ui'",
		"echo hi; wmic process call create 'loomux ui'",
	)
}

// xargs takes the subcommand from stdin, which the words never show.
func TestXargsMayFeedTheSubcommand(t *testing.T) {
	mustRefuseUI(t, `echo ui | xargs loomux`, `printf ui | xargs loomux`, `echo ui | xargs -n1 loomux --root x`,
		`echo ui | xargs -I{} loomux {}`, `echo ui | xargs loomux ''`)
	mustAllow(t, "echo status | xargs loomux status", "echo x | xargs echo")
}

// Windows drops a trailing dot from a file name, so loomux.exe. runs
// loomux.exe, while its base name is no longer loomux.exe.
func TestTrailingDotOnTheBinary(t *testing.T) {
	mustRefuseUI(t,
		`& "$env:LOCALAPPDATA\loomux\bin\loomux.exe." ui`,
		`& "C:\Users\x\AppData\Local\loomux\bin\loomux.exe." ui`,
		`./bin/loomux.exe. ui`,
		`./bin/loomux.exe.. ui`,
	)
}

// PowerShell runs whatever command object the parenthesis yields; one that
// names loomux is loomux.
func TestCallOperatorOnALookedUpCommand(t *testing.T) {
	mustRefuseUI(t, `& (gcm loomux) ui`, `& (Get-Command loomux) ui`, `& $(Get-Command loomux.exe) ui`)
	mustAllow(t, `& (gcm loomux) version`)
}

// Process start APIs and task schedulers take loomux and its arguments as
// strings, the way Start-Process does.
func TestProcessStartOutsideStartProcess(t *testing.T) {
	mustRefuseUI(t,
		`[Diagnostics.Process]::Start('loomux','ui')`,
		`[System.Diagnostics.Process]::Start("loomux.exe", "ui")`,
		`Invoke-CimMethod -ClassName Win32_Process -MethodName Create -Arguments @{CommandLine='loomux ui'}`,
		`wmic process call create "loomux ui"`,
		`schtasks /create /tn x /tr "loomux ui" /sc once /st 00:00`,
		`Register-ScheduledTask -Action (New-ScheduledTaskAction -Execute loomux -Argument ui) -TaskName x`,
	)
	mustAllow(t, `schtasks /query`)
}

// Another language's process API takes loomux and its arguments as separate
// strings; a program that only reads, or git on a subcommand that records
// text, may name one without starting anything.
func TestAnotherLanguageStartsLoomux(t *testing.T) {
	mustRefuseUI(t,
		`python -c "import subprocess; subprocess.run(['loomux','ui'])"`,
		`node -e "require('child_process').spawn('loomux',['ui'])"`,
		`python -c "import os; os.system('loomux ui')"`,
		`git -c alias.x='!wmic process call create "loomux ui"' x`,
	)
	mustAllow(t,
		"grep -rn subprocess loomux/",
		`git commit -m "fix subprocess loomux"`,
		`git log --grep "wmic loomux"`,
	)
}

// huntRefuses fails for every line a shell standing in root/rel may run
// although the guard lets it through, under both shell tools. Each line runs
// `loomux ui`.
func huntRefuses(t *testing.T, rel string, lines ...string) {
	t.Helper()
	root := t.TempDir()
	cwd := ""
	if rel != "" {
		cwd = filepath.Join(root, filepath.FromSlash(rel))
	}
	for _, line := range lines {
		for _, tool := range []string{"Bash", "PowerShell"} {
			got := checkToolOpen(root, "", cwd, tool, map[string]any{"command": line}, config.Policy{}, switchOff, atRoot)
			if len(got) == 0 {
				t.Errorf("%s in %q passes, yet runs loomux ui: %q", tool, rel, line)
			}
		}
	}
}

// go resolves a relative package against the shell's folder: from cmd,
// ./loomux is the main package.
func TestGoRunOfLoomuxFromTheCmdFolder(t *testing.T) {
	huntRefuses(t, "cmd", "go run ./loomux ui", "cd loomux && go run . ui")
	huntRefuses(t, "", "cd cmd && go run ./loomux ui", "cd cmd; go run ./loomux ui")
	root, state := world(t, "")
	if code, _, errOut := call(t, root, state, shellAt("go run ./loomux ui", filepath.Join(root, "cmd"))); code != 2 {
		t.Errorf("go run ./loomux ui from cmd: code %d, err %q", code, errOut)
	}
	mustAllow(t, "cd cmd/other && go run ./loomux ui", "cd docs; go run . ui", "go run ./loomux ui")
}

// bash expands a glob to the folder before go sees it.
func TestGoRunOfAGlobbedPackage(t *testing.T) {
	huntRefuses(t, "", "go run ./cmd/l* ui", "go run ./cmd/* ui", "go run ./c?d/loo?ux ui", "go run ./cmd/loo*/main.go ui")
	mustAllow(t, "go run ./tools/* ui", "go run ./cmd/o*/main.go ui")
}

// go's build flags include undocumented ones that take a value, which the
// guard must not read as the package.
func TestGoRunWithADebugFlagValue(t *testing.T) {
	huntRefuses(t, "",
		"go run -debug-trace t.json ./cmd/loomux ui",
		"go run -debug-actiongraph a.json ./cmd/loomux ui",
		"go run -debug-runtime-trace r.out ./cmd/loomux ui",
	)
	huntRefuses(t, "cmd/loomux", "go run -debug-trace t.json . ui")
}

// Push-Location and chdir are PowerShell's names for Set-Location; chdir is
// also cmd's cd.
func TestOtherNamesOfCd(t *testing.T) {
	huntRefuses(t, "",
		"Push-Location cmd/loomux; go run . ui",
		"Push-Location -Path cmd/loomux; go run . ui",
		`chdir cmd\loomux && go run . ui`,
		`cmd /c "chdir cmd\loomux && go run . ui"`,
	)
}

// A cd whose folder a shell computes still ends in cmd/loomux.
func TestCdIntoAComputedMainFolder(t *testing.T) {
	huntRefuses(t, "", `cd "$(git rev-parse --show-toplevel)/cmd/loomux" && go run . ui`)
	mustAllow(t, `cd "$(git rev-parse --show-toplevel)/docs" && go run . ui`)
}

// The payload's cwd reaches the lines a line runs from a string too.
func TestNestedLineInTheMainFolder(t *testing.T) {
	huntRefuses(t, "cmd/loomux", `sh -c "go run . ui"`, `pwsh -c "go run . ui"`, `echo "go run . ui" | sh`)
	huntRefuses(t, "", `sh -c "cd cmd && go run ./loomux ui"`)
}

// GNU parallel appends the words after ::: to the call.
func TestParallelFeedsTheSubcommand(t *testing.T) {
	huntRefuses(t, "", "parallel loomux ::: ui", "parallel loomux {} ::: ui")
	mustAllow(t, "parallel loomux ::: status", "parallel echo ::: ui")
}

// A glob in the program path names loomux.exe without spelling it.
func TestGlobbedProgramPath(t *testing.T) {
	huntRefuses(t, "", "./bin/loo*.exe ui", "./bin/*.exe ui")
	mustAllow(t, "./bin/*.txt ui", "./bin/loo*.exe status")
}

// A reading program that is handed a quoted alternation names the launcher
// and loomux as patterns, not as a launch: the pipe inside the quotes is no
// pipe. A folder called loomux in a path is no loomux either.
func TestAnAlternationOrAFolderNamedLoomuxIsNoLaunch(t *testing.T) {
	mustAllow(t,
		`grep -E "loomux|subprocess" internal/hooks/*.go`,
		`grep -rnE 'wmic|loomux' .`,
		`rg "schtasks|scheduledtask" ~/src/loomux`,
		`rg -n "loomux|os.system" docs`,
		`git log --grep="subprocess|loomux"`,
		`git grep -E "loomux|child_process" -- internal`,
		`cd /home/me/loomux/.claude/worktrees/x && python -c "import subprocess; subprocess.run(['go','test'])"`,
		`python -c "import subprocess; subprocess.run(['git','status'])" --path "$LOCALAPPDATA/loomux/state"`,
		`gh pr create --title "fix subprocess" --body "the launcher rule fires on loomux"`,
		`gh issue comment 3 --body "wmic loomux"`,
	)
	mustRefuseUI(t,
		// A pipe outside any quote still starts a new program.
		`echo "a|b" | python -c "import subprocess; subprocess.run(['loomux','ui'])"`,
		`grep "a|b" f; wmic process call create 'loomux ui'`,
		`grep -E "x" f | wmic process call create "loomux ui"`,
		// A substitution inside double quotes runs, so its pipe counts.
		`echo "$(echo x | wmic process call create 'loomux ui')"`,
		"echo \"`echo x | wmic process call create 'loomux ui'`\"",
		// An escaped quote may open or close a string, depending on the
		// shell; the line is read without masking.
		`echo \" | wmic process call create "loomux ui"`,
		`echo "a \" b" | wmic process call create 'loomux ui'`,
		// A folder named loomux, followed by the program that is one.
		`cd /x/loomux/y && python -c "import subprocess; subprocess.run(['loomux','ui'])"`,
		`wmic process call create "C:\bin\loomux\loomux.exe ui"`,
		`python -c "import subprocess; subprocess.run(['go','run','./cmd/loomux/main.go','ui'])"`,
		`schtasks /create /tn x /tr "go run ./cmd/loomux/... ui" /sc once /st 00:00`,
		`gh alias set x '!wmic process call create "loomux ui"'`,
	)
}

// A glob stands for loomux where a program stands, but a bare wildcard among
// the arguments of another program is no loomux.
func TestABareWildcardAmongArgumentsIsNoLoomux(t *testing.T) {
	mustAllow(t,
		"rsync * ui",
		"taskset 0x1 * init",
		"Start-Process notepad -ArgumentList *",
		"start /wait x ? ",
		"tar cf x.tar *.* ui",
	)
	mustRefuseUI(t,
		"taskset 0x1 loomux ui",
		"taskset 0x1 loo*.exe ui",
		"taskset 0x1 ./bin/loomux ui",
		"Start-Process loo*.exe -ArgumentList ui",
		"./bin/* ui",
	)
}

// PowerShell's dot operator runs the command object a parenthesis yields,
// as the call operator does.
func TestDotSourceOnALookedUpCommand(t *testing.T) {
	mustRefuseUI(t, `. (gcm loomux) ui`, `. (Get-Command loomux.exe) ui`)
	mustAllow(t, `. (gcm loomux) version`)
}

// A relative path that starts at loomux/ is the main package once the shell
// stands in cmd, so a launcher text that holds it is a launch even where the
// folder rule would otherwise read loomux/ as a folder.
func TestARelativeLoomuxFolderIsThePackageFromCmd(t *testing.T) {
	lines := []string{
		`cd cmd; python3 -c "import subprocess; subprocess.run(['go','run','./loomux/.','ui'])"`,
		`python3 -c "import subprocess; subprocess.run(['go','run','./loomux/.','ui'], cwd='cmd')"`,
	}
	mustRefuseUI(t, lines...)
	huntRefuses(t, "", lines...)
}
