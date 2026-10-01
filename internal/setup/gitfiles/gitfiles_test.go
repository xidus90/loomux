package gitfiles

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/setup/hostfile"
)

// The two binary paths a host project's hooks may call, quoted as the
// installer hands them over.
const (
	canonical = `"${LOCALAPPDATA}/loomux/bin/loomux.exe"`
	checkout  = `"${CLAUDE_PROJECT_DIR}/bin/loomux.exe"`
)

// The copies above test what the installer really hands over only while they
// match the binaries the hook entries call.
func TestTheBinariesAreTheOnesOfTheHookEntries(t *testing.T) {
	if canonical != hostfile.Canonical || checkout != hostfile.Checkout {
		t.Fatalf("canonical %s, checkout %s; hostfile has %s and %s",
			canonical, checkout, hostfile.Canonical, hostfile.Checkout)
	}
}

func TestHooksCallTheGivenBinary(t *testing.T) {
	for _, binary := range []string{canonical, checkout} {
		hooks := Hooks(binary)
		if len(hooks) != 3 {
			t.Fatalf("Hooks(%s) has %d entries, want 3", binary, len(hooks))
		}
		inner := strings.Trim(binary, `"`)
		want := map[string]string{
			"pre-commit": `exec "` + inner + `" check precommit --arm` + "\n",
			"commit-msg": `exec "` + inner + `" check commit-msg "$1"` + "\n",
			"pre-push":   "refs/heads/main | refs/heads/master)",
		}
		for name, fragment := range want {
			text, ok := hooks[name]
			if !ok {
				t.Fatalf("Hooks(%s) lacks %s", binary, name)
			}
			if !strings.HasPrefix(text, "#!/bin/sh\n") {
				t.Errorf("%s does not start with a sh shebang: %q", name, text)
			}
			if strings.Contains(text, "\r") {
				t.Errorf("%s contains a carriage return", name)
			}
			if !strings.Contains(text, fragment) {
				t.Errorf("%s lacks %q:\n%s", name, fragment, text)
			}
			if !strings.HasSuffix(text, "\n") {
				t.Errorf("%s lacks a final newline", name)
			}
		}
	}
}

func TestHooksAcceptAnUnquotedBinary(t *testing.T) {
	got := Hooks("/opt/loomux")["pre-commit"]
	if !strings.Contains(got, `exec "/opt/loomux" check precommit --arm`) {
		t.Errorf("pre-commit = %q", got)
	}
}

func TestThePreCommitHookArms(t *testing.T) {
	want := "#!/bin/sh\n" +
		"# loomux pre-commit hook: the check chain of .loomux/config.toml.\n" +
		"exec \"/opt/loomux\" check precommit --arm\n"
	if got := Hooks("/opt/loomux")["pre-commit"]; got != want {
		t.Fatalf("pre-commit = %q", got)
	}
	if got := Hooks(`"/opt/loomux"`)["pre-commit"]; got != want {
		t.Fatalf("a quoted binary: %q", got)
	}
	if !RunsAGate(want) {
		t.Fatal("our own hook no longer counts as a gate")
	}
	// The command stages the file itself, into the index git handed the hook;
	// a `git add` in the script would pull it into a commit of paths.
	if got := Hooks("/opt/loomux")["pre-commit"]; strings.Contains(got, "git add") {
		t.Fatal("the hook stages")
	}
}

// Only the hook init wrote before is ours to replace: the marker line and
// the one call, nothing else. Anything a human added makes it theirs.
func TestUpgradeKnowsOnlyTheHookInitWroteBefore(t *testing.T) {
	const old = "#!/bin/sh\n# loomux pre-commit hook: the check chain of .loomux/config.toml.\nexec \"${LOCALAPPDATA}/loomux/bin/loomux.exe\" check precommit\n"
	got, ok := Upgrade(old)
	if !ok || got != Hooks("${LOCALAPPDATA}/loomux/bin/loomux.exe")["pre-commit"] {
		t.Fatalf("%v %q", ok, got)
	}
	if got, ok = Upgrade(strings.ReplaceAll(old, "\n", "\r\n")); !ok || strings.Contains(got, "\r") {
		t.Fatalf("CRLF: %v %q", ok, got)
	}
	if got, ok = Upgrade(strings.Replace(old, "${LOCALAPPDATA}/loomux/bin/loomux.exe", "./bin/loomux.exe", 1)); !ok || !strings.Contains(got, "exec \"./bin/loomux.exe\" check precommit --arm\n") {
		t.Fatalf("the binary the old hook called is kept: %v %q", ok, got)
	}
	for name, text := range map[string]string{
		"the new form":    Hooks("/opt/loomux")["pre-commit"],
		"no marker":       "#!/bin/sh\n# our gate\nexec \"/opt/loomux\" check precommit\n",
		"a line added":    old + "echo done\n",
		"another call":    strings.Replace(old, "check precommit", "check all", 1),
		"no exec":         strings.Replace(old, "exec \"", "\"", 1),
		"another shebang": strings.Replace(old, "#!/bin/sh", "#!/bin/bash", 1),
		"a foreign gate":  "#!/bin/sh\nsh ci/gate.sh\n",
		"empty":           "",
	} {
		if got, ok := Upgrade(text); ok || got != "" {
			t.Errorf("%s: %v %q", name, ok, got)
		}
	}
}

// The marker line says loomux wrote the hook, whatever form it has and
// whatever a human added since; a hook that only mentions the words does not.
func TestIsLoomuxPreCommitGoesByTheMarkerLine(t *testing.T) {
	const old = "#!/bin/sh\n# loomux pre-commit hook: the check chain of .loomux/config.toml.\nexec \"/opt/loomux\" check precommit\n"
	for name, c := range map[string]struct {
		text string
		want bool
	}{
		"the old form":       {old, true},
		"the new form":       {Hooks("/opt/loomux")["pre-commit"], true},
		"a line added":       {old + "echo done\n", true},
		"CRLF":               {strings.ReplaceAll(old, "\n", "\r\n"), true},
		"a foreign gate":     {"#!/bin/sh\nsh ci/gate.sh\n", false},
		"the words mid-line": {"#!/bin/sh\necho '# loomux pre-commit hook: no'\n", false},
		"the other hook":     {Hooks("/opt/loomux")["commit-msg"], false},
		"empty":              {"", false},
	} {
		if got := IsLoomuxPreCommit(c.text); got != c.want {
			t.Errorf("%s: %v, want %v", name, got, c.want)
		}
	}
}

func TestRunsAGateKnowsTheOldAndNewChains(t *testing.T) {
	own, err := os.ReadFile("../../../.githooks/pre-commit")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"own pre-commit", string(own), true},
		{"ulguard", "#!/bin/sh\nulguard --root .\n", true},
		{"ultraloom", "#!/bin/sh\nuv run ultraloom check precommit\n", true},
		{"loomux", "#!/bin/sh\nloomux check precommit\n", true},
		{"gate script", "#!/bin/sh\nsh ci/gate.sh\n", true},
		{"empty", "", false},
		{"echo", "#!/bin/sh\necho hi\n", false},
		{"only a comment", "#!/bin/sh\n# once ran loomux check precommit\necho hi\n", false},
	}
	for _, c := range cases {
		if got := RunsAGate(c.text); got != c.want {
			t.Errorf("%s: RunsAGate = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestPrePushRefusesTheDefaultBranches(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH")
	}
	script := t.TempDir() + "/pre-push"
	if err := os.WriteFile(script, []byte(Hooks(canonical)["pre-push"]), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		stdin  string
		code   int
		stderr string
	}{
		{"refs/heads/a 1 refs/heads/main 2\n", 1, "pre-push: pushing to main is refused; open a pull request\n"},
		{"refs/heads/a 1 refs/heads/master 2\n", 1, "pre-push: pushing to master is refused; open a pull request\n"},
		{"refs/heads/feat 1 refs/heads/feat 2\n", 0, ""},
		{"", 0, ""},
	}
	for _, c := range cases {
		cmd := exec.Command(sh, script)
		// A redirection gone wrong writes a file into the working directory
		// instead of onto stderr; it lands in a directory of its own.
		cmd.Dir = t.TempDir()
		cmd.Stdin = strings.NewReader(c.stdin)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		code := 0
		if err := cmd.Run(); err != nil {
			exit, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatal(err)
			}
			code = exit.ExitCode()
		}
		if code != c.code {
			t.Errorf("stdin %q: exit %d, want %d", c.stdin, code, c.code)
		}
		if stderr.String() != c.stderr {
			t.Errorf("stdin %q: stderr %q, want %q", c.stdin, stderr.String(), c.stderr)
		}
	}
}

func TestHooksTakeOffOnlyAPairOfQuotes(t *testing.T) {
	cases := []struct{ binary, exec string }{
		{`"C:/x/loomux.exe"`, `exec "C:/x/loomux.exe" check precommit --arm`},
		{`""`, `exec "" check precommit --arm`},
		{`"`, `exec """ check precommit --arm`},
		{`"C:/x`, `exec ""C:/x" check precommit --arm`},
		{`C:/x"`, `exec "C:/x"" check precommit --arm`},
	}
	for _, c := range cases {
		if got := Hooks(c.binary)["pre-commit"]; !strings.Contains(got, c.exec) {
			t.Errorf("Hooks(%q) pre-commit = %q, want %q in it", c.binary, got, c.exec)
		}
	}
}

func TestGitignoreLines(t *testing.T) {
	got := GitignoreLines()
	if len(got) != 1 || got[0] != "/.loomux/state/" {
		t.Errorf("GitignoreLines = %q", got)
	}
}

func TestWithGitignoreAppendsOnlyWhatIsMissing(t *testing.T) {
	block := gitignoreComment + "\n/.loomux/state/\n"
	cases := []struct {
		name    string
		in      string
		want    string
		changed bool
	}{
		{"empty", "", block, true},
		{"other lines", "bin/\n", "bin/\n" + block, true},
		{"no final newline", "bin/", "bin/\n" + block, true},
		{"crlf", "bin/\r\nout/\r\n", "bin/\r\nout/\r\n" + strings.ReplaceAll(block, "\n", "\r\n"), true},
		{"crlf without final newline", "bin/\r\nout/", "bin/\r\nout/\r\n" + strings.ReplaceAll(block, "\n", "\r\n"), true},
		{"present", "bin/\n/.loomux/state/\n", "bin/\n/.loomux/state/\n", false},
		{"present unanchored", ".loomux/state/\n", ".loomux/state/\n", false},
		{"present without slash", "/.loomux/state\n", "/.loomux/state\n", false},
		{"present bare", ".loomux/state", ".loomux/state", false},
		{"present crlf with spaces", "  /.loomux/state/  \r\n", "  /.loomux/state/  \r\n", false},
	}
	for _, c := range cases {
		got, changed := WithGitignore(c.in)
		if got != c.want || changed != c.changed {
			t.Errorf("%s: WithGitignore(%q) = %q, %v; want %q, %v", c.name, c.in, got, changed, c.want, c.changed)
		}
	}
}
