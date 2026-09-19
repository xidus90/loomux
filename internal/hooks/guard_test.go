package hooks

import (
	"path/filepath"
	"regexp"
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
		{"no-verify", ".claude/.no-verify", "the stop gate's own controls are not written by the party it gates"},
		{"hook script", ".loomux/state/hooks/stop.py", "the stop gate's own controls are not written by the party it gates"},
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

// A glob the matcher cannot read is a refusal, not a miss.
//
// Load-time validation catches nearly all of them, but not this one:
// `filepath.Match(glob, "")` stops at the first chunk that does not match an
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
