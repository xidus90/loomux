package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/testlock"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestReadPolicyWithoutAConfigIsEmpty(t *testing.T) {
	policy, err := ReadPolicy(t.TempDir())
	if err != nil || len(policy.Paths) != 0 || len(policy.Commands) != 0 {
		t.Fatalf("%+v, %v", policy, err)
	}
}

func TestReadPolicyAcceptsAStringOrAListOfGlobs(t *testing.T) {
	root := writeConfig(t, `
[[policy.paths.rules]]
match  = "generated/*"
reason = "generated"

[[policy.paths.rules]]
match  = ["coverage.out", "bin/*"]
reason = "build output"

[[policy.commands.rules]]
regex  = '(^|[\n;&|(`+"`"+`])\s*pip\s+install([^\w-]|$)'
reason = "uv, never pip"
`)
	policy, err := ReadPolicy(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(policy.Paths) != 2 || strings.Join(policy.Paths[1].Match, ",") != "coverage.out,bin/*" || policy.Paths[0].Match[0] != "generated/*" {
		t.Fatalf("%+v", policy.Paths)
	}
	if len(policy.Commands) != 1 || !policy.Commands[0].Regex.MatchString("uv run x; pip install y") {
		t.Fatalf("%+v", policy.Commands)
	}
}

func TestReadPolicyRefusesARegexGoCannotCompile(t *testing.T) {
	root := writeConfig(t, `
[[policy.commands.rules]]
regex  = 'git\s+push(?![\w-])'
reason = "lookahead"
`)
	_, err := ReadPolicy(root)
	if err == nil || !strings.Contains(err.Error(), "config.toml") || !strings.Contains(err.Error(), "git\\s+push(?![\\w-])") {
		t.Fatalf("err %v", err)
	}
}

func TestReadPolicyRefusesUnreadableTOML(t *testing.T) {
	root := writeConfig(t, "[[policy.paths.rules]\n")
	if _, err := ReadPolicy(root); err == nil || !strings.Contains(err.Error(), "config.toml") {
		t.Fatalf("err %v", err)
	}
}

func TestReadPolicyRefusesAMatchOfTheWrongType(t *testing.T) {
	root := writeConfig(t, "[[policy.paths.rules]]\nmatch = 3\nreason = \"x\"\n")
	if _, err := ReadPolicy(root); err == nil {
		t.Fatal("want error")
	}
}

func TestReadPolicyRefusesAPathRuleWithoutAReason(t *testing.T) {
	root := writeConfig(t, "[[policy.paths.rules]]\nmatch = \"bin/*\"\n")
	if _, err := ReadPolicy(root); err == nil || !strings.Contains(err.Error(), "needs a reason") {
		t.Fatalf("err %v", err)
	}
}

func TestReadPolicyRefusesACommandRuleWithoutAReason(t *testing.T) {
	root := writeConfig(t, "[[policy.commands.rules]]\nregex = 'pip'\n")
	if _, err := ReadPolicy(root); err == nil || !strings.Contains(err.Error(), "needs a reason") {
		t.Fatalf("err %v", err)
	}
}

func TestReadPolicyRefusesAListElementThatIsNotAString(t *testing.T) {
	root := writeConfig(t, "[[policy.paths.rules]]\nmatch = [\"bin/*\", 3]\nreason = \"x\"\n")
	if _, err := ReadPolicy(root); err == nil || !strings.Contains(err.Error(), "is not a string") {
		t.Fatalf("err %v", err)
	}
}

func TestReadPolicyRefusesAConfigThatExistsButCannotBeRead(t *testing.T) {
	root := writeConfig(t, "")
	testlock.Lock(t, ManifestPath(root))
	if _, err := ReadPolicy(root); err == nil || !strings.Contains(err.Error(), "config.toml") {
		t.Fatalf("err %v", err)
	}
}

func TestManifestPathIsTheOneManifestName(t *testing.T) {
	if got, want := ManifestPath("r"), filepath.Join("r", ".loomux", "config.toml"); got != want {
		t.Fatalf("ManifestPath = %q, want %q", got, want)
	}
}
