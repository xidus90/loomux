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
			"pre-commit": `exec "` + inner + `" check precommit` + "\n",
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
	if !strings.Contains(got, `exec "/opt/loomux" check precommit`) {
		t.Errorf("pre-commit = %q", got)
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
		{`"C:/x/loomux.exe"`, `exec "C:/x/loomux.exe" check precommit`},
		{`""`, `exec "" check precommit`},
		{`"`, `exec """ check precommit`},
		{`"C:/x`, `exec ""C:/x" check precommit`},
		{`C:/x"`, `exec "C:/x"" check precommit`},
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
