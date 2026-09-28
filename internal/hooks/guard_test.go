package hooks

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

func TestSafeFileWritesCarryNoReason(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{
		"src/main.py",
		filepath.Join(root, "src", "main.py"),
		"../outside.py",
	} {
		if reasons := checkTool(root, "Write", map[string]any{"file_path": path}, config.Policy{}); len(reasons) != 0 {
			t.Fatalf("%s: reasons %v, want none", path, reasons)
		}
	}
}

// A leading **/ stands for any directory, the root included, and is anchored
// at an element: my.loomux is no .loomux.
func TestMatchGlobReadsALeadingDoubleStarAsAnyDirectory(t *testing.T) {
	for _, row := range []struct {
		pattern, name string
		want          bool
	}{
		{"**/.loomux/config.toml", ".loomux/config.toml", true},
		{"**/.loomux/config.toml", "../sibling/.loomux/config.toml", true},
		{"**/.loomux/config.toml", "/repo/.loomux/config.toml", true},
		{"**/.loomux/config.toml", "C:/repo/.loomux/config.toml", true},
		{"**/.loomux/config.toml", "my.loomux/config.toml", false},
		{"**/.loomux/config.toml", ".loomux/config.toml.bak", false},
		{"**/.loomux/state/runs/**", "x/.loomux/state/runs", true},
		{"**/.loomux/state/runs/**", "x/.loomux/state/runs/0001.jsonl", true},
		{"**/.loomux/state/runs/**", ".loomux/state/runsx/0001.jsonl", false},
	} {
		got, err := matchGlob(row.pattern, row.name)
		if err != nil || got != row.want {
			t.Errorf("matchGlob(%q, %q) = %v, %v; want %v", row.pattern, row.name, got, err, row.want)
		}
	}
	if _, err := matchGlob("**/foo/*[x", "a/foo/b"); err == nil {
		t.Fatal("a bad class behind **/ must still be an error")
	}
}

// loomux's own files are kept under any directory: a sibling worktree's
// .loomux, or one named by its absolute path, like this project's.
func TestLoomuxsOwnRulesHoldUnderAnyDirectory(t *testing.T) {
	root := t.TempDir()
	for path, want := range map[string]string{
		"/repo/.loomux/config.toml":                          manifestReason,
		"../sibling/.loomux/config.toml":                     manifestReason,
		filepath.Join(t.TempDir(), ".loomux", "config.toml"): manifestReason,
		"../sibling/.loomux/state/runs/0001.jsonl":           runFilesWant,
		"../sibling/.loomux/state/hooks/stop.py":             "the stop gate's own controls are not written by the party it gates",
	} {
		if got := checkTool(root, "Write", map[string]any{"file_path": path}, config.Policy{}); len(got) != 1 || got[0] != want {
			t.Errorf("%s: reasons %q, want %q", path, got, want)
		}
	}
	for _, path := range []string{"my.loomux/config.toml", ".loomux/config.toml.bak", "docs/loomux/config.toml"} {
		if got := checkTool(root, "Write", map[string]any{"file_path": path}, config.Policy{}); len(got) != 0 {
			t.Errorf("%s: reasons %q, want none", path, got)
		}
	}
}

func TestProtectedBuiltinPathsCarryTheirReason(t *testing.T) {
	root := t.TempDir()
	tests := []struct{ name, path, reason string }{
		{"env file", ".env", "secrets are not written by an agent"},
		{"env local", ".env.local", "secrets are not written by an agent"},
		{"ssh key", "id_rsa", "secrets are not written by an agent"},
		{"pem file", "server.pem", "secrets are not written by an agent"},
		{"key file", "cert.key", "secrets are not written by an agent"},
		{"p12 file", "cert.p12", "secrets are not written by an agent"},
		{"npm token", ".npmrc", "secrets are not written by an agent"},
		{"pypi token", ".pypirc", "secrets are not written by an agent"},
		{"service account", "credentials.json", "secrets are not written by an agent"},
		{"aws secret", ".aws/credentials", "secrets are not written by an agent"},
		{"no-verify", ".loomux/no-verify", "the stop gate's own controls are not written by the party it gates"},
		{"manifest", ".loomux/config.toml", manifestReason},
		{"manifest in capitals", ".LOOMUX/Config.toml", manifestReason},
		{"hook script", ".loomux/state/hooks/stop.py", "the stop gate's own controls are not written by the party it gates"},
		// loomux's own rules fold case, as the barrier does for the manifest:
		// Windows and macOS keep one file under both spellings.
		{"hook script in capitals", ".LOOMUX/State/hooks/stop.py", "the stop gate's own controls are not written by the party it gates"},
		{"no-verify in capitals", ".LOOMUX/No-Verify", "the stop gate's own controls are not written by the party it gates"},
		{"env file in capitals", ".ENV", "secrets are not written by an agent"},
		{"go sum in capitals", "GO.SUM", "lock files are written by their package manager, not by hand"},
		{"run journal", ".loomux/state/runs/0001.jsonl", "a flow's journal and marker are written by loomux, not by the party the gates ask"},
		{"run journal in capitals", ".LOOMUX/State/runs/0001.jsonl", "a flow's journal and marker are written by loomux, not by the party the gates ask"},
		{"uv lock", "uv.lock", "lock files are written by their package manager, not by hand"},
		{"poetry lock", "poetry.lock", "lock files are written by their package manager, not by hand"},
		{"npm lock", "package-lock.json", "lock files are written by their package manager, not by hand"},
		{"pnpm lock", "pnpm-lock.yaml", "lock files are written by their package manager, not by hand"},
		{"yarn lock", "yarn.lock", "lock files are written by their package manager, not by hand"},
		{"cargo lock", "Cargo.lock", "lock files are written by their package manager, not by hand"},
		{"go sum", "go.sum", "lock files are written by their package manager, not by hand"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reasons := checkTool(root, "Edit", map[string]any{"file_path": tc.path}, config.Policy{})
			if len(reasons) != 1 || reasons[0] != tc.reason {
				t.Fatalf("reasons %v, want %q", reasons, tc.reason)
			}
		})
	}
}

// A configured glob with a slash in it is matched as a whole path, so the rule
// reaches a file the same pattern would miss by base name alone.
func TestAConfiguredPathRuleCarriesItsReason(t *testing.T) {
	root := t.TempDir()
	policy := config.Policy{Paths: []config.PathRule{{
		Match:  []string{"docs/*.md", "migrations/[0-9][0-9][0-9][0-9]_*.py"},
		Reason: "Django migrations are protected",
	}}}
	reasons := checkTool(root, "Write", map[string]any{"file_path": "migrations/0001_initial.py"}, policy)
	if len(reasons) != 1 || reasons[0] != "Django migrations are protected" {
		t.Fatalf("reasons %v", reasons)
	}
	if reasons := checkTool(root, "Write", map[string]any{"file_path": "src/0001_initial.py"}, policy); len(reasons) != 0 {
		t.Fatalf("reasons %v, want none: the glob names a directory", reasons)
	}
}

// A `*` stops at a slash on every platform. On Windows `filepath.Match`
// separates on `\` only, so there `docs/*.md` reached into `docs/sub/`.
func TestConfiguredGlobDoesNotMatchAcrossDirectories(t *testing.T) {
	root := t.TempDir()
	policy := config.Policy{Paths: []config.PathRule{{
		Match:  []string{"docs/*.md"},
		Reason: "docs root files are protected",
	}}}
	if reasons := checkTool(root, "Write", map[string]any{"file_path": "docs/sub/nested.md"}, policy); len(reasons) != 0 {
		t.Fatalf("docs/*.md matched docs/sub/nested.md: %v", reasons)
	}
	if reasons := checkTool(root, "Write", map[string]any{"file_path": "docs/top.md"}, policy); len(reasons) != 1 {
		t.Fatalf("docs/*.md missed docs/top.md: %v", reasons)
	}
}

// A glob the matcher cannot read is a refusal, not a miss.
//
// Load-time validation catches nearly all of them, but not this one:
// `path.Match(glob, "")` stops at the first chunk that does not match an
// empty name, so a bad class in a later chunk reaches the matcher. Discarding
// the error there turned the rule into one that protects nothing, without a
// word -- the very defect the policy refuses at load.
func TestAGlobTheMatcherCannotReadRefuses(t *testing.T) {
	root := t.TempDir()
	if _, err := config.ReadPolicy(root); err != nil {
		t.Fatalf("an empty root carries no policy: %v", err)
	}
	policy := config.Policy{Paths: []config.PathRule{{
		Match:  []string{`foo/*[x`},
		Reason: "protects nothing",
	}}}
	reasons := checkTool(root, "Write", map[string]any{"file_path": "foo/bar.go"}, policy)
	if len(reasons) != 1 || !strings.Contains(reasons[0], `foo/*[x`) {
		t.Fatalf("reasons %v, want one naming the glob", reasons)
	}
	if !strings.Contains(reasons[0], "loomux cannot read") {
		t.Fatalf("reason %q does not say the matcher could not read the glob", reasons[0])
	}
}

// A notebook names its target under its own key, and every target of a call is
// judged, not the first one found: the forbidden file_path beside the notebook
// answers as well, each with its own reason.
func TestNotebookEditIsJudgedByItsOwnTargetKey(t *testing.T) {
	root := t.TempDir()
	reasons := checkTool(root, "NotebookEdit", map[string]any{
		"notebook_path": ".env",
		"file_path":     "go.sum",
	}, config.Policy{})
	// In the order the targets are read: file_path comes before notebook_path.
	want := []string{
		"lock files are written by their package manager, not by hand",
		"secrets are not written by an agent",
	}
	if len(reasons) != len(want) || reasons[0] != want[0] || reasons[1] != want[1] {
		t.Fatalf("reasons %v, want %v", reasons, want)
	}
}

// Both books are read for one target: a built-in rule and a configured one that
// match the same path each contribute their reason.
func TestABuiltinAndAConfiguredRuleBothCarryTheirReason(t *testing.T) {
	root := t.TempDir()
	policy := config.Policy{Paths: []config.PathRule{{
		Match:  []string{".env"},
		Reason: "this project keeps its secrets out of the tree entirely",
	}}}
	reasons := checkTool(root, "Write", map[string]any{"file_path": ".env"}, policy)
	want := []string{
		"secrets are not written by an agent",
		"this project keeps its secrets out of the tree entirely",
	}
	if len(reasons) != len(want) || reasons[0] != want[0] || reasons[1] != want[1] {
		t.Fatalf("reasons %v, want %v", reasons, want)
	}
}

func TestMultiEditIsJudgedLikeAWrite(t *testing.T) {
	root := t.TempDir()
	if reasons := checkTool(root, "MultiEdit", map[string]any{"file_path": ".env"}, config.Policy{}); len(reasons) != 1 {
		t.Fatalf("reasons %v", reasons)
	}
}

// A tool that writes nothing is judged by no path rule, whatever it carries.
func TestAReadingToolIsNotJudgedByThePathRules(t *testing.T) {
	root := t.TempDir()
	if reasons := checkTool(root, "Read", map[string]any{"file_path": ".env"}, config.Policy{}); len(reasons) != 0 {
		t.Fatalf("reasons %v, want none", reasons)
	}
}

func TestGitPushIsRefusedOnBashAndPowerShell(t *testing.T) {
	root := t.TempDir()
	for _, tool := range []string{"Bash", "PowerShell"} {
		reasons := checkTool(root, tool, map[string]any{"command": "git push origin main"}, config.Policy{})
		if len(reasons) != 1 || reasons[0] != "Whether commits reach the remote is a human's decision." {
			t.Fatalf("[%s] reasons %v", tool, reasons)
		}
	}
}

// Every command tool is judged by the same command rules, under each of its
// keys in any case. The loop runs over the table itself, so a tool that joins
// it is held to the rules without a new test.
func TestGitPushIsRefusedOnEveryCommandTool(t *testing.T) {
	root := t.TempDir()
	checked := 0
	for name, tool := range commandTools {
		line := "git push origin main"
		if !tool.whole {
			line += "\n"
		}
		for _, key := range tool.keys {
			for _, spelling := range []string{key, strings.ToLower(key), strings.ToUpper(key)} {
				reasons := checkTool(root, name, map[string]any{spelling: line}, config.Policy{})
				if len(reasons) != 1 || reasons[0] != "Whether commits reach the remote is a human's decision." {
					t.Fatalf("[%s %s] reasons %v", name, spelling, reasons)
				}
				checked++
			}
		}
	}
	if checked == 0 {
		t.Fatal("commandTools names no key, so nothing was judged")
	}
}

// A second spelling of a key is judged beside the first, not in its place:
// a harmless line under one cannot carry a forbidden one under the other.
func TestEveryCasingOfALineKeyIsJudged(t *testing.T) {
	root := t.TempDir()
	const push = "Whether commits reach the remote is a human's decision."
	for tool, input := range map[string]map[string]any{
		"manage_task": {"Action": "send_input", "Input": "echo hi\n", "input": "git push origin main\n"},
		"run_command": {"CommandLine": "go test ./...", "COMMANDLINE": "git push origin main"},
	} {
		if reasons := checkTool(root, tool, input, config.Policy{}); len(reasons) != 1 || reasons[0] != push {
			t.Fatalf("[%s] reasons %v", tool, reasons)
		}
	}
}

// agy 1.2.11 types into a task run_command left open with manage_task,
// Action send_input and the line under Input (measured 2026-09-25): that line
// is judged. list, status and kill carry none and pass; any other Action,
// or none, is judged, so one without Input is refused.
func TestManageTaskSendInputIsJudgedByTheCommandRules(t *testing.T) {
	root := t.TempDir()
	const push = "Whether commits reach the remote is a human's decision."
	send := map[string]any{"Action": "send_input", "Input": "git push origin main\n", "TaskId": "c/task-6"}
	if reasons := checkTool(root, "manage_task", send, config.Policy{}); len(reasons) != 1 || reasons[0] != push {
		t.Fatalf("send_input: reasons %v", reasons)
	}
	for _, action := range []string{"kill", "list", "status"} {
		quiet := map[string]any{"Action": action, "TaskId": "c/task-2"}
		if reasons := checkTool(root, "manage_task", quiet, config.Policy{}); len(reasons) != 0 {
			t.Fatalf("%s: reasons %v, want none", action, reasons)
		}
	}
	// An action agy's schema does not name may carry a line all the same.
	other := map[string]any{"Action": "sendInput", "Input": "git push origin main\n"}
	if reasons := checkTool(root, "manage_task", other, config.Policy{}); len(reasons) != 1 || reasons[0] != push {
		t.Fatalf("sendInput: reasons %v", reasons)
	}
	for _, input := range []map[string]any{{"Action": "send_input"}, {"TaskId": "c/task-6", "Text": "git push"}} {
		reasons := checkTool(root, "manage_task", input, config.Policy{})
		if len(reasons) != 1 || !strings.Contains(reasons[0], "no command line in this manage_task call") {
			t.Fatalf("%v: reasons %v", input, reasons)
		}
	}
	// Whole lines that break no rule pass, an empty one included.
	for _, line := range []string{"y\n", "echo hi\r\n", "\n"} {
		typed := map[string]any{"Action": "send_input", "Input": line, "TaskId": "c/task-6"}
		if reasons := checkTool(root, "manage_task", typed, config.Policy{}); len(reasons) != 0 {
			t.Fatalf("%q: reasons %v, want none", line, reasons)
		}
	}
}

// The lines a call carries are judged whatever its Action says; a quiet
// Action only excuses a call that carries none, and only when every key
// spelled action names a quiet one.
func TestAQuietActionExcusesOnlyAMissingLine(t *testing.T) {
	root := t.TempDir()
	const push = "Whether commits reach the remote is a human's decision."
	for _, input := range []map[string]any{
		{"Action": "kill", "Input": "git push origin main\n"},
		{"Action": "status", "Input": "git push origin main\n"},
		{"Action": "kill", "action": "send_input", "Input": "git push origin main\n"},
	} {
		if reasons := checkTool(root, "manage_task", input, config.Policy{}); len(reasons) != 1 || reasons[0] != push {
			t.Fatalf("%v: reasons %v", input, reasons)
		}
	}
	for _, input := range []map[string]any{
		{"Action": "kill", "action": "send_input"},
		{"Action": "kill", "action": 42},
		{"Action": 42},
		// A line the guard cannot read might be the one the tool types, and
		// an empty one types nothing: neither is a line to judge.
		{"Action": "kill", "Input": 42},
		{"Action": "kill", "Input": nil},
		{"Action": "send_input", "Input": ""},
	} {
		reasons := checkTool(root, "manage_task", input, config.Policy{})
		if len(reasons) != 1 || !strings.Contains(reasons[0], "no command line in this manage_task call") {
			t.Fatalf("%v: reasons %v", input, reasons)
		}
	}
	for _, input := range []map[string]any{
		{"Action": "kill", "TaskId": "t"},
		{"Action": "list"},
		{"action": "status"},
		{"Action": "kill", "ACTION": "list"},
		{"Action": "kill", "Input": ""},
	} {
		if reasons := checkTool(root, "manage_task", input, config.Policy{}); len(reasons) != 0 {
			t.Fatalf("%v: reasons %v, want none", input, reasons)
		}
	}
}

// A value the guard cannot read does not stop the others from being judged:
// a Bash call, whose line is optional, still has its push found beside it.
func TestAnUnreadableLineValueHidesNoOther(t *testing.T) {
	root := t.TempDir()
	input := map[string]any{"command": 42, "COMMAND": "git push origin main"}
	if reasons := checkTool(root, "Bash", input, config.Policy{}); len(reasons) != 1 || reasons[0] != "Whether commits reach the remote is a human's decision." {
		t.Fatalf("reasons %v", reasons)
	}
}

// What an agent types into an open task is judged only as whole lines
// without control characters: a fragment may be finished by the next call,
// a backspace or an escape sequence edits the line, and a continuation lets
// the next line finish it, after the guard has read it. All are refused, a
// quiet Action no excuse for any.
func TestATypedInputIsJudgedOnlyAsWholeLines(t *testing.T) {
	root := t.TempDir()
	const rule = "loomux judges what an agent types into a task only as whole lines without control characters; this "
	cases := []struct{ input, want string }{
		{"git pu", "does not end its line, so it refuses"},
		{"x", "does not end its line, so it refuses"},
		{"\x03", "does not end its line, so it refuses"},
		{"\x1b[A\n", "carries a control character, so it refuses"},
		{"git pusx\bh origin main\n", "carries a control character, so it refuses"},
		{"ls\x00\n", "carries a control character, so it refuses"},
		{"ls\x7f\n", "carries a control character, so it refuses"},
		{"ls\U0000009b2J\n", "carries a control character, so it refuses"},
		// An interactive shell completes a word at a tab: "git pus<Tab>"
		// becomes "git push" after the guard has read "git pus".
		{"git pus\t origin main\n", "carries a control character, so it refuses"},
		// A backslash or a backtick at a line end continues the line: bash
		// and PowerShell run "loomux init" from the two lines below.
		{"loomux \\\ninit\n", "does not end its line, so it refuses"},
		{"loomux `\ninit\n", "does not end its line, so it refuses"},
		{"git \\\npush origin main\n", "does not end its line, so it refuses"},
	}
	for _, tool := range []string{"manage_task", "send_command_input"} {
		for _, c := range cases {
			input := map[string]any{"Action": "send_input", "Input": c.input}
			reasons := checkTool(root, tool, input, config.Policy{})
			if len(reasons) != 1 || reasons[0] != rule+tool+" input "+c.want {
				t.Fatalf("[%s] %q: reasons %v", tool, c.input, reasons)
			}
		}
	}
	quiet := map[string]any{"Action": "kill", "Input": "x"}
	if reasons := checkTool(root, "manage_task", quiet, config.Policy{}); len(reasons) != 1 || reasons[0] != rule+"manage_task input does not end its line, so it refuses" {
		t.Fatalf("kill with a fragment: reasons %v", reasons)
	}
	// Each value is judged on its own: a fragment under one spelling does not
	// stop the whole line under the other from being judged.
	both := map[string]any{"Action": "send_input", "Input": "git pu", "input": "git push origin main\n"}
	want := []string{rule + "manage_task input does not end its line, so it refuses", "Whether commits reach the remote is a human's decision."}
	if reasons := checkTool(root, "manage_task", both, config.Policy{}); !slices.Equal(reasons, want) {
		t.Fatalf("two values: reasons %v", reasons)
	}
	// A whole command line is no typing: run_command ends no line and may
	// carry what a shell reads as a control character.
	if reasons := checkTool(root, "run_command", map[string]any{"CommandLine": "printf '\x1b[0m'"}, config.Policy{}); len(reasons) != 0 {
		t.Fatalf("run_command: reasons %v, want none", reasons)
	}
}

// Several lines typed in one call are split at every line end and each is
// judged on its own, so a rule anchored at the start of a line reaches the
// second one too. The rule is ^rm\s, not the built-in push rule: its (^|\s)
// matches a line end, so a push in line 2 would be caught without any
// splitting and the test could not fail.
func TestATypedInputIsJudgedLineByLine(t *testing.T) {
	root := t.TempDir()
	policy := config.Policy{Commands: []config.CommandRule{{
		Regex: regexp.MustCompile(`^rm\s`), Source: `^rm\s`, Reason: "no rm at the start of a line",
	}}}
	for _, typed := range []string{"echo hi\nrm -rf x\n", "echo hi\r\nrm -rf x\r\n", "echo hi\rrm -rf x\r"} {
		for _, tool := range []string{"manage_task", "send_command_input"} {
			reasons := checkTool(root, tool, map[string]any{"Action": "send_input", "Input": typed}, policy)
			if len(reasons) != 1 || reasons[0] != "no rm at the start of a line" {
				t.Fatalf("[%s] %q: reasons %v", tool, typed, reasons)
			}
		}
	}
}

// A tool whose line is not optional is refused when the guard finds no line
// under any of its keys, not waved through; a Bash or PowerShell call without
// a command stays unjudged, as it was. The loop runs over the table, so the
// closed default reaches a tool that joins it.
func TestEveryCommandToolWithoutALineIsRefused(t *testing.T) {
	root := t.TempDir()
	checked := 0
	for name, tool := range commandTools {
		if tool.lineOptional {
			if reasons := checkTool(root, name, map[string]any{}, config.Policy{}); len(reasons) != 0 {
				t.Fatalf("[%s] reasons %v, want none", name, reasons)
			}
			continue
		}
		reasons := checkTool(root, name, map[string]any{"Cmd": "git push"}, config.Policy{})
		if len(reasons) != 1 || reasons[0] != "loomux found no command line in this "+name+" call, so it cannot judge it and refuses" {
			t.Fatalf("[%s] reasons %v", name, reasons)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no command tool in the table refuses a call without a line")
	}
}

// The manifest is refused to a shell line that writes it on the same terms as
// to a writing tool, and left alone by one that only reads it. The allowed
// lines are the ones a careless expression would refuse: a copy *from* the
// manifest, a redirect *after* reading it, a redirect in another segment.
func TestAShellLineThatWritesTheManifestIsRefused(t *testing.T) {
	root := t.TempDir()
	const want = "the manifest is where the barrier reads its own limits, so no shell command may write it"
	refused := []string{
		"cat >> .loomux/config.toml <<'EOF'\n[policy]\nEOF",
		"echo x >.loomux/config.toml",
		`echo x > "./.loomux/config.toml"`,
		`echo x > .loomux\config.toml`,
		"echo x >| /repo/.loomux/config.toml",
		"echo x 2>&1 >> .LOOMUX/Config.toml",
		"sed -i 's/a/b/' .loomux/config.toml",
		"sed -i.bak 's/a/b/' .loomux/config.toml",
		"sed -Ei 's/a/b/' .loomux/config.toml",
		"sed --in-place -e 's/a/b/' .loomux/config.toml",
		"sed 's/a/b/' .loomux/config.toml -i",
		"/usr/bin/sed -i x .loomux/config.toml",
		"perl -pi -e 's/a/b/' .loomux/config.toml",
		"printf x | tee -a .loomux/config.toml",
		"git status; tee .loomux/config.toml < x",
		"sudo tee .loomux/config.toml",
		"Set-Content -Path .loomux/config.toml -Value x",
		`"x" | Out-File .loomux\config.toml`,
		`Out-File -FilePath:.loomux\config.toml`,
		`Add-Content .loomux/config.toml "x"`,
		"Clear-Content .loomux/config.toml",
		"'x' | Tee-Object -FilePath .loomux/config.toml",
		"[IO.File]::WriteAllText('.loomux/config.toml', 'x')",
		"cp other.toml .loomux/config.toml",
		"cp -f other.toml '.loomux/config.toml'",
		"mv tmp .loomux/config.toml && ls",
		"mv .loomux/config.toml elsewhere.toml",
		"rm .loomux/config.toml",
		"Remove-Item .loomux\\config.toml",
		"Copy-Item -Destination .loomux\\config.toml -Path x.toml",
		"Copy-Item x.toml .loomux\\config.toml",
		"git checkout -- .loomux/config.toml",
		"dd if=x of=.loomux/config.toml",
		"x=$(sed -i s/a/b/ .loomux/config.toml)",
	}
	for _, tool := range []string{"Bash", "PowerShell"} {
		for _, line := range refused {
			reasons := checkTool(root, tool, map[string]any{"command": line}, config.Policy{})
			if len(reasons) != 1 || !strings.HasSuffix(reasons[0], want) {
				t.Errorf("[%s] %q: reasons %v, want exactly the manifest refusal", tool, line, reasons)
			}
		}
	}
	allowed := []string{
		"cat .loomux/config.toml",
		"grep policy .loomux/config.toml",
		"Get-Content .loomux/config.toml",
		"sed -n '1,5p' .loomux/config.toml",
		"sed -n '/min/p' .loomux/config.toml",
		"cp .loomux/config.toml backup.toml",
		"Copy-Item .loomux\\config.toml -Destination backup.toml",
		"cat .loomux/config.toml > out.txt",
		"echo x > other.toml; cat .loomux/config.toml",
		"echo x > .loomux/config.toml.bak",
		"echo x > my.loomux/config.toml",
		"grep tee .loomux/config.toml",
		"git show HEAD:.loomux/config.toml",
		"dd if=.loomux/config.toml of=copy.toml",
		"sed -i s/a/b/ other.toml; cat .loomux/config.toml",
	}
	for _, line := range allowed {
		if reasons := checkTool(root, "Bash", map[string]any{"command": line}, config.Policy{}); len(reasons) != 0 {
			t.Errorf("%q: reasons %v, want none", line, reasons)
		}
	}
}

func TestASafeCommandCarriesNoReason(t *testing.T) {
	root := t.TempDir()
	if reasons := checkTool(root, "Bash", map[string]any{"command": "git status"}, config.Policy{}); len(reasons) != 0 {
		t.Fatalf("reasons %v, want none", reasons)
	}
}

// A command line that is no string is no command line: there is nothing to
// match an expression against.
func TestACommandThatIsNoStringCarriesNoReason(t *testing.T) {
	root := t.TempDir()
	if reasons := checkTool(root, "Bash", map[string]any{"command": 7}, config.Policy{}); len(reasons) != 0 {
		t.Fatalf("reasons %v, want none", reasons)
	}
}

func TestAConfiguredCommandRuleCarriesItsReason(t *testing.T) {
	root := t.TempDir()
	policy := config.Policy{Commands: []config.CommandRule{{
		Regex:  regexp.MustCompile(`(^|\s)pip\s+install`),
		Source: `(^|\s)pip\s+install`,
		Reason: "use uv add instead",
	}}}
	reasons := checkTool(root, "Bash", map[string]any{"command": "pip install requests"}, policy)
	if len(reasons) != 1 || reasons[0] != "use uv add instead" {
		t.Fatalf("reasons %v", reasons)
	}
}

func TestRelativePathHelper(t *testing.T) {
	root := t.TempDir()
	rel := relativePath(filepath.Join(root, "a", "b.txt"), root)
	if rel != "a/b.txt" {
		t.Fatalf("rel = %q, want a/b.txt", rel)
	}
	outside := relativePath("../outside.txt", root)
	if !strings.Contains(outside, "outside.txt") {
		t.Fatalf("outside = %q", outside)
	}
	otherDir := t.TempDir()
	absOutside := relativePath(filepath.Join(otherDir, "other.txt"), root)
	if !strings.Contains(absOutside, "other.txt") {
		t.Fatalf("absOutside = %q", absOutside)
	}
}

// loomuxSpellings are ways to write a loomux call that a reader of the
// line has to see through: paths to the binary, quotes, prefixes, wrappers,
// blocks, continuations and escapes. Each row is a configuration write
// writesConfiguration refuses; the gate rule reads the same rows with its
// own command in place of the row's, so the two lists cannot drift apart.
func loomuxSpellings() []string {
	return []string{
		"loomux init",
		"bin/loomux.exe init --hooks=all",
		`"${LOCALAPPDATA}/loomux/bin/loomux.exe" config`,
		"go run ./cmd/loomux config set commit.language de",
		"cd x && loomux config",
		"LOOMUX_STATE_DIR=x loomux area add",
		"go run ./cmd/loomux/ init",
		"cd x\nloomux init",
		`.\bin\loomux.exe config set a b`,
		`& "C:\x\loomux.exe" init`,
		`& 'C:\x\loomux.exe' area add`,
		`go run .\cmd\loomux config set a b`,
		"sudo loomux init",
		"command loomux init",
		"exec loomux config set a b",
		"nohup loomux area add",
		"env X=1 loomux init",
		"time loomux config",
		"sudo X=1 env loomux init",
		"(loomux init)",
		"echo $(loomux config set a b)",
		"{ loomux area add; }",
		`loomux config set commit.message "say \"hi"`,
		`loomux config set a 'it'\''s'`,
		`loomux config set a "(x)"`,
		`loomux config set a "x;y"`,
		`"C:\Program Files (x86)\loomux\loomux.exe" init`,
		`.\bin\loomux.exe config set a "say \"hi"`,
		`.\bin\loomux.exe init \'`,
		`"C:\Program Files\loomux\loomux.exe" init`,
		`"C:\Program Files\loomux\loomux.exe" init \'`,
		`"C:\Program Files\loomux\loomux.exe" config set a "say \"hi"`,
		`"C:\Program Files\loomux\loomux.exe" config set a 'x`,
		`"C:\Program Files (x86)\loomux\loomux.exe" config set a "b"`,
		`'C:\Program Files\loomux\loomux.exe' config set a "it's" ""`,
		`"C:\Program Files (x86)\My Tools\loomux.exe" init`,
		`"C:\Program Files (x86)\loomux tools\loomux.exe" init`,
		`"C:\R&D Tools\loomux.exe" init`,
		`"C:\x;y z\loomux.exe" init`,
		`& "C:\R&D Tools\loomux.exe" config set a b`,
		`echo "x; loomux init"`,
		`loomux con\fig set a b`,
		// An earlier escaped quote mispairs both cuts; the plain field
		// reading still finds the program behind the ( of the path.
		`echo "a \" b"; "C:\Program Files (x86)\loomux\loomux.exe" init`,
		`echo 'a \' b'; "C:\Program Files (x86)\loomux\loomux.exe" init`,
		`echo "a \" b"; "C:\Program Files (x86)\loomux\loomux.exe" config set a b`,
		// Shell reserved words in front of the program.
		"for i in 1; do loomux config set a b --yes; done",
		"if true; then loomux config set a b; fi",
		"if false; then :; else loomux init; fi",
		"if x; then :; elif loomux init; then :; fi",
		"if loomux init; then :; fi",
		"! loomux config set a b",
		"while x; do loomux init; done",
		"until x; do loomux area add; done",
		// Line continuations, bash and PowerShell, with either line end.
		"loomux \\\nconfig set a b --yes",
		"loomux \\\r\nconfig set a b",
		"loom\\\nux init",
		"loomux `\nconfig set a b",
		"loomux `\r\ninit",
		// Backtick substitution, and a PowerShell backtick escape.
		"echo `loomux config set a b`",
		"x=\"`loomux init`\"",
		"loomux con`fig set a b",
		// Wrappers without flags, wrapper flags without a value, and
		// redirections in front of the program.
		"cmd /c loomux config set a b",
		"cmd.exe /c loomux init",
		`C:\Windows\System32\CMD.EXE /S /C loomux init`,
		`cmd /c "loomux init"`,
		"xargs loomux config set a b",
		"xargs -0 loomux init",
		"timeout 60 loomux config set a b",
		"timeout --foreground 60 loomux init",
		"sudo -E loomux init",
		"env -i loomux init",
		"2>/dev/null loomux config set a b",
		">out loomux init",
		"< f loomux init",
		"2> err loomux init",
		"2>&1 loomux init",
		">&2 loomux init",
		"&>out loomux init",
		// Wrapper flags that take a value.
		"sudo -u root loomux init",
		"xargs -n 1 loomux init",
		"timeout -s KILL 60 loomux init",
		"env -u HOME loomux init",
		"sudo -- loomux init",
		"sudo -R /srv loomux init",
		"sudo --chroot /srv loomux init",
		// env -S takes the command line itself as its value.
		"env -S loomux init",
		"env -S 'loomux init'",
		// Every cmd switch before /c or /k.
		"cmd /v:on /c loomux init",
		"cmd /d /s /c loomux init",
		"CMD /E:ON /K loomux init",
		// Function bodies, coprocesses and PowerShell's try, catch, finally.
		"function f { loomux init; }",
		"function f { loomux init }",
		"coproc loomux init",
		"try { loomux init } catch {}",
		"try { x } catch { loomux init }",
		"try { x } finally { loomux init }",
		"}; catch { loomux init }",
		// nice, bare, with -n N, and with the old -N.
		"nice loomux init",
		"nice -n 10 loomux init",
		"nice -10 loomux init",
		// A false refusal kept on purpose: -v only looks the name up.
		"command -v loomux init",
		// Braces glued to the words around them.
		"try{ loomux init }catch{}",
		"try {loomux init} catch {}",
		"& {loomux init}",
		"function f {loomux init}",
		"Invoke-Command -ScriptBlock {loomux init}",
		"${X}/loomux init",
		// Every combination of the line variants: a backtick escape inside
		// a block whose braces are glued to the words.
		"try{ loomux con`fig set a b }",
		"{loomux con`fig set a b}",
		"try {loomux con`fig set a b} catch {}",
		"try{ loomux `\ninit }",
		// nice with -n N and then more flags or --.
		"nice -n 10 -- loomux init",
		"nice -n 10 -x loomux init",
		// False refusals kept on purpose: a word after a lone brace.
		"awk '{ print }' loomux init",
		"echo } loomux config set a b",
		"echo ${X} loomux init",
		// go run with build flags, a file, a module path or no ./.
		"go run -race ./cmd/loomux config set a b",
		"go run -tags x ./cmd/loomux init",
		"go run -ldflags=-s ./cmd/loomux init",
		"go run -C . ./cmd/loomux init",
		"go run ./cmd/loomux/main.go config set a b",
		"go run github.com/xidus90/loomux/cmd/loomux config set a b",
		"go run github.com/xidus90/loomux/cmd/loomux@latest init",
		"go run cmd/loomux init",
		// Start-Process hands its arguments on as one list, so loomux
		// anywhere behind it counts, whatever the command.
		"Start-Process loomux -ArgumentList 'config','set','a','b'",
		"Start-Process -FilePath loomux.exe -ArgumentList 'init'",
		`saps .\bin\loomux.exe`,
		"Start-Process loomux init",
		"Start-Process cmd '/c loomux init'",
		// A false refusal kept on purpose: Start-Process hides the
		// arguments from the words.
		"start loomux config list",
		// loomux anywhere among Start-Process's arguments.
		"Start-Process -NoNewWindow loomux init",
		"Start-Process -Wait -FilePath loomux.exe -ArgumentList init",
		"Start-Process -ArgumentList 'init' -FilePath loomux.exe",
		`start "" loomux init`,
		"start /b loomux init",
		// A PowerShell parameter glued to its value with a colon.
		"Start-Process -FilePath:loomux.exe -ArgumentList init",
		"Start-Process -FilePath:loomux init",
		"Start-Process -FilePath:'loomux.exe' -ArgumentList init",
		`Start-Process -FilePath:"C:\x\loomux.exe"`,
		// A false refusal kept on purpose: loomux as an argument of another
		// program Start-Process runs.
		"Start-Process code -ArgumentList loomux",
		// A wrapper's quoted inner command, which the quote-blind cut breaks
		// at the break inside the quotes.
		"cmd /c 'echo; loomux init'",
		"cmd /c 'x&loomux config set a b'",
		"cmd /c 'x|loomux init'",
		"cmd /c 'x (loomux init)'",
		"& cmd /c 'x; loomux init'",
		"sh -c 'true\nloomux init'",
		"sh -c 'x `loomux init` y'",
		"pwsh -c 'x\nloomux config set a b'",
		// A lone & is PowerShell's call operator or a background job.
		"& loomux init",
		"& 'loomux' config set a b",
		"sleep 1 & loomux config set a b",
	}
}

func TestTheGuardRefusesCommandsThatWriteTheConfiguration(t *testing.T) {
	// The spellings, then what makes init, config, area and merge-hook
	// write in them.
	refused := append(loomuxSpellings(),
		"loomux init --yes",
		"loomux config set commit.language de --yes",
		"loomux config --global",
		"loomux config set model.enabled true --global",
		// config takes its flags before the subcommand too; a write that
		// names them first is still a write.
		"loomux config --root . set commit.language en",
		"loomux config --global unset model.enabled",
		"loomux config --root . set commit.language en --propose=false",
		"loomux config --root .",
		"loomux config --global",
		"loomux config --root",
		"loomux config --root . apply x",
		"loomux config --yes list",
		"loomux area add --path .",
		// A PowerShell line-end continuation is no bash one: PowerShell
		// runs the first line on its own.
		"loomux init \\\n--dry-run",
		// Only list, get, proposals and a lone --help or -h read.
		"loomux config ''",
		"loomux config unset commit.language",
		"loomux config lis",
		"loomux config --help --global",
		"loomux config 'unclosed",
		// set and unset pass only as a proposal, and applying or
		// rejecting one is the human's half.
		"loomux config apply --all --yes",
		"loomux config apply 20260924T101530Z-001",
		"loomux config reject 20260924T101530Z-001",
		"loomux config reject --all",
		"loomux config set a b --yes",
		"loomux config set a b --propos",
		"loomux config set a b ---propose",
		// Past -- the word is the value, not the flag.
		"loomux config set a -- --propose",
		"loomux config unset -- --propose",
		// A later =false turns the flag off again.
		"loomux config set a b --propose=false",
		"loomux config set a b --propose --propose=false",
		"loomux config set a b -propose=false",
		"loomux config set a b --propose=true",
		"loomux config --propose set a b",
		// The same for init: a --dry-run the program never receives.
		"loomux init # --dry-run",
		"loomux init #--dry-run",
		// A quoted # word still ends the flags flagOn reads.
		`loomux init "#" --dry-run`,
		`loomux init "# x" --dry-run`,
		"loomux init > --dry-run",
		"loomux init 2> --detect-only",
		"loomux init <<< --dry-run",
		"loomux init <# --detect-only #>",
		"loomux init -- --dry-run",
		// An expansion may bring a word that takes the flag back or turns
		// it into a value; the words are judged as written, so any
		// expansion among them refuses.
		"loomux config set a b --propose $X",
		"loomux config set a b --propose ${X}",
		"loomux config set a b --propose $(echo --propose=false)",
		"loomux config set a b --propose $env:X",
		"loomux config set a b --propose @x",
		"loomux config set a $X --propose",
		"loomux config set a b --propose `echo --propose=false`",
		"loomux init --dry-run $X",
		"loomux init --dry-run @args",
		// The exemption holds only for a line of plain words: brace
		// expansion, PowerShell sub-expressions and globs can each bring
		// a word that takes the flag back.
		"loomux config set a b --propose --propose{,=false}",
		"loomux init --dry-run --dry-run{,=false}",
		"loomux config set a b --propose ('--propose=false')",
		`loomux config set a b --propose ("--propose" + "=false")`,
		"loomux init --dry-run ('--dry-run=false')",
		"loomux config set a b --propose ?-propose=false",
		"loomux config set a b --propose [-]-propose=false",
		"loomux config set a b --propose *",
		`loomux config set a "$X" --propose`,
		`loomux config set a "x\y" --propose`,
		"loomux config set a b --propose \\",
		"loomux config set a b --propose~",
		"loomux config set a b --propose !x",
		"loomux config set a b --propose a#b",
		"loomux config set a 'b --propose",
		"loomux config set a 'ü’ $(x)' --propose",
		"loomux config set a b --propose --% %X%",
		"cmd /c loomux init --dry-run %X%",
		// A redirection does not end the arguments: words after its
		// target still reach the program.
		"loomux config set a b --propose > out --propose=false",
		"loomux config set a b --propose > out x",
		"loomux init --dry-run 2>&1 --dry-run=false",
		// Only a direct call is exempt: a wrapper may read the words a
		// second time (cmd reads ^, %X%, !X! and " inside '…'), so
		// loomux behind any prefix is judged without its flag.
		"cmd /c loomux config set a b --propose '--propose^=false'",
		"cmd /c loomux init --dry-run '--dry-run^=false'",
		"cmd /c loomux init --dry-run '%X%'",
		"X=--dry-run=false cmd //c loomux init --dry-run '%X%'",
		"cmd /v:on /c loomux init --dry-run '!X!'",
		`cmd /c loomux config set a b --propose '-"-propose=false"'`,
		"cmd /c loomux init --dry-run",
		"sudo loomux init --dry-run",
		"env loomux config set a b --propose",
		"nice loomux init --dry-run",
		"timeout 60 loomux init --dry-run",
		"xargs loomux init --dry-run",
		"exec loomux init --dry-run",
		"command loomux config set a b --propose",
		"X=1 loomux init --dry-run",
		"2>/dev/null loomux init --dry-run",
		"if true; then loomux init --dry-run; fi",
		"! loomux init --dry-run",
		"Start-Process loomux init --dry-run",
		// A wrapper's quoted inner command: a cut inside the quotes puts
		// loomux at the head of a segment, while the wrapper reads the
		// string again (cmd resolves ^, % and !), so only a segment whose
		// bounds lie outside quotes may exempt, and single quotes may hold
		// neither a break nor a character cmd rereads.
		"cmd /c 'echo; loomux init --dry-run --dry-run^=false'",
		"cmd /c 'x&loomux config set a b --propose -propose^=0'",
		"cmd /c 'x|loomux init --dry-run %X%'",
		"cmd /c 'x (loomux init --dry-run)'",
		"Start-Process cmd '/c loomux init --dry-run --dry-run^=false'",
		"& cmd /c 'x; loomux init --dry-run'",
		"sh -c 'true\nloomux init --dry-run'",
		"sh -c 'x `loomux init --dry-run` y'",
		"pwsh -c 'x\nloomux config set a b --propose'",
		"loomux config set a 'x!y' --propose",
		"loomux config set a 'x^y' --propose",
		"loomux config set a '50%' --propose",
		"loomux config set a 'a>b' --propose",
		// A quoted # is a word to the program, yet flagOn reads a word
		// starting with # as a comment and stops there.
		"loomux init --dry-run '#x' --dry-run=false",
		"loomux config set a b --propose '#x' --propose=false",
		// A lone & is PowerShell's call operator or a background job.
		"& loomux init --dry-run",
		"& 'loomux' config set a b --propose",
		"sleep 1 & loomux config set a b --propose",
		"loomux init --dry-run &",
		// Blocks and a program path with an expansion are no plain line.
		"try { loomux config set a b --propose}",
		"try { loomux init --dry-run } catch {}",
		"${X}/loomux init --dry-run",
		"{loomux init --dry-run}",
		"try { loomux init --dry-run}",
		// A later =false takes the flag back for init as well.
		"loomux init --dry-run --dry-run=false",
		"loomux init --detect-only -detect-only=false",
		// A comment or a redirect puts the word on the line, not in the
		// program's arguments.
		"loomux config set a b --yes # --propose",
		"loomux config set a b --yes #--propose",
		"loomux config set a b --yes > --propose",
		"loomux config set a b --yes >--propose",
		"loomux config set a b --yes 2> --propose",
		"loomux config set a b --yes 2>--propose",
		"loomux config set a b --yes &> --propose",
		"loomux config set a b --yes <<< --propose",
		"loomux config set a b --yes < --propose",
		"loomux config set a b --yes <# --propose #>",
		"loomux config set a b --propose; loomux config apply --all --yes",
		// merge-hook install and remove put executable hooks into other
		// repositories, as area add writes into another configuration.
		"loomux merge-hook install",
		"loomux merge-hook remove",
		"loomux merge-hook install --dry-run",
		"sudo loomux merge-hook install",
		"cmd /c loomux merge-hook remove",
		"go run ./cmd/loomux merge-hook install",
		`& "C:\x\loomux.exe" merge-hook remove`,
		"{ loomux merge-hook install; }",
		"cd x && loomux merge-hook remove",
		// convert and fetch write into an area's inbox, which the write
		// barrier keeps from agents.
		"loomux convert",
		"loomux convert x.pdf",
		`loomux convert "C:\vault\00 Eingang\x.pdf"`,
		"loomux fetch https://www.youtube.com/watch?v=mHSOsy_usAg --scope knowledge",
		"go run ./cmd/loomux convert",
		`& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" fetch https://x`,
		"loomux convert --help x",
	)
	// Known holes, pinned so that closing one shows up here and the readings
	// comment and both cli-references are corrected with it.
	holes := []string{
		// After an earlier escaped quote, a quoted path whose part after
		// its last break character holds a blank.
		`echo "a \" b"; "C:\R&D Tools\loomux.exe" init`,
		`echo "a \" b"; "C:\Program Files (x86)\My Tools\loomux.exe" init`,
		// A command inside a string, an alias, a program in a variable.
		`sh -c "loomux init"`,
		`pwsh -c "loomux init"`,
		"alias l=loomux; l init",
		"M=loomux; $M init",
		// go run of a package that names no cmd/loomux.
		"cd cmd/loomux && go run . init",
	}
	allowed := []string{
		"loomux init --dry-run",
		"loomux init --detect-only",
		"loomux config list",
		"loomux config get commit.language",
		"loomux config list --global",
		"loomux config --help",
		"loomux config -h",
		"loomux config proposals",
		"loomux config proposals --json --global",
		"loomux config set commit.language de --propose",
		"loomux config set a b -propose",
		"loomux config unset commit.language --propose --global",
		"loomux config set --propose a b --root d",
		// config takes --root and --global before the subcommand too.
		"loomux config --root . list",
		"loomux config --root=d get commit.language",
		"loomux config --global list --json",
		"loomux config -root d --global proposals",
		"loomux config --root . set commit.language de --propose",
		"loomux config set layout.wiki docs/wiki --propose",
		"loomux config set index.include 'docs/**/*.md' --propose",
		`loomux config set commit.message "a b" --propose`,
		"loomux config set a 'x {y} * ? [z] ~ @w' --propose",
		"loomux init --detect-only > plan.txt",
		"cd docs && loomux init --dry-run | tee plan.txt",
		"loomux config set commit.message 'it''s' --propose",
		"loomux config unset a --propose",
		"go run ./cmd/loomux config set a b --propose",
		"go run -race ./cmd/loomux init --dry-run",
		"loomux init --dry-run && echo done",
		"loomux init --dry-run &> plan.txt",
		"loomux init --dry-run # a $note {x}\nloomux config list",
		"loomux config set a b --propose > out.txt",
		"loomux config set a b --propose 2>&1",
		"loomux config set a b --propose # note",
		"loomux init --dry-run > plan.txt",
		"loomux init --detect-only 2>&1 # note",
		"loomux check precommit",
		"echo loomux config set",
		"grep 'loomux init' docs",
		"loomux",
		"loomux area list",
		"loomux area",
		"go run",
		"go run ./cmd/other init",
		"go run ./mycmd/loomux init",
		"go run -race ./cmd/other init",
		"go run -tags",
		"X=1",
		"a ;; b",
		"sudo",
		"{ }",
		"cmd /c loomux config list",
		"cmd /c",
		"timeout",
		"timeout 60",
		"2>/dev/null loomux config get a",
		">",
		"echo `date`",
		"Start-Process notepad",
		"Start-Process",
		"Start-Process -FilePath",
		"echo hi >&2",
		"Start-Process -Wait notepad",
		"cmd /v:on /c loomux config list",
		"function f { loomux config list; }",
		"nice -n 10 loomux config get a",
		"nice -n",
		"function",
		"}",
		"try{ loomux config list }catch{}",
		"Start-Process -FilePath:notepad",
		"echo ${HOME}",
		"grep ' { ' loomux.txt",
		`"${LOCALAPPDATA}/loomux/bin/loomux.exe" config list`,
		"loomux config get {a}",
		"try { loomux config list}",
		"try{ loomux con`fig list }",
		"loomux merge-hook status",
		"loomux merge-hook record",
		"loomux merge-hook",
		"sudo loomux merge-hook status",
		"cmd /c loomux merge-hook record",
		"go run ./cmd/loomux merge-hook status",
		"loomux convert --help",
		"loomux convert -h",
		"loomux fetch --help",
		"loomux fetch -h",
		// A # inside double quotes is a byte of the word to both shells.
		`cd "C:/x/#GIT/loomux" && loomux init --dry-run`,
		`Set-Location "C:/x/#GIT/loomux"; loomux init --dry-run`,
		`loomux init --dry-run --root "C:/x/#GIT/loomux"`,
	}
	for _, line := range refused {
		if !writesConfiguration(line) {
			t.Errorf("must refuse %q", line)
		}
	}
	for _, line := range allowed {
		if writesConfiguration(line) {
			t.Errorf("must allow %q", line)
		}
	}
	for _, line := range holes {
		if writesConfiguration(line) {
			t.Errorf("hole %q is closed: move it to refused and correct the readings comment", line)
		}
	}
}

func TestCheckToolNamesTheConfigurationReason(t *testing.T) {
	got := checkTool(t.TempDir(), "Bash", map[string]any{"command": "loomux config set x y"}, config.Policy{})
	if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "a human runs them") }) {
		t.Fatalf("reasons %v", got)
	}
	got = checkTool(t.TempDir(), "PowerShell", map[string]any{"command": `& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" config set x y`}, config.Policy{})
	if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "a human runs them") }) {
		t.Fatalf("PowerShell reasons %v", got)
	}
	got = checkTool(t.TempDir(), "Bash", map[string]any{"command": "loomux merge-hook install"}, config.Policy{})
	if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "merge-hook install and remove") }) {
		t.Fatalf("merge-hook reasons %v", got)
	}
	got = checkTool(t.TempDir(), "Bash", map[string]any{"command": "loomux convert"}, config.Policy{})
	if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "convert and fetch write into an area's inbox") }) {
		t.Fatalf("convert reasons %v", got)
	}
	if got := checkTool(t.TempDir(), "Bash", map[string]any{"command": "loomux config list"}, config.Policy{}); len(got) != 0 {
		t.Fatalf("reasons %v", got)
	}
	// The refusal names the way an agent may take instead.
	got = checkTool(t.TempDir(), "Bash", map[string]any{"command": "loomux config set x y"}, config.Policy{})
	if !slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, "--propose") }) {
		t.Fatalf("no --propose in %v", got)
	}
}
