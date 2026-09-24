package commit_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/verify/commit"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	configDir := filepath.Join(root, ".loomux")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPolicyDefaultWhenNoConfigFile(t *testing.T) {
	policy, err := commit.ReadPolicy(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Language != "en" || policy.Threshold != 2 || !policy.Conventional {
		t.Errorf("unexpected default policy: %+v", policy)
	}
}

func TestPolicyDefaultWhenNoCommitSection(t *testing.T) {
	root := writeConfig(t, "[area]\nscope = \"test\"\n")
	policy, err := commit.ReadPolicy(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Language != "en" || policy.Threshold != 2 || !policy.Conventional {
		t.Errorf("unexpected policy: %+v", policy)
	}
}

func TestPolicyLanguageAndThresholdAndConventional(t *testing.T) {
	root := writeConfig(t, `
[commit]
language = "de"
threshold = 3
conventional = false
`)
	policy, err := commit.ReadPolicy(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Language != "de" || policy.Threshold != 3 || policy.Conventional {
		t.Errorf("unexpected policy: %+v", policy)
	}
}

func TestPolicyAllowPatterns(t *testing.T) {
	root := writeConfig(t, `
[commit]
language = "en"

[[commit.allow]]
regex  = "^Quelle:"
reason = "Zitierte Quelle, keine Prosa."
`)
	policy, err := commit.ReadPolicy(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policy.Allow) != 1 {
		t.Fatalf("expected 1 allow rule, got %d", len(policy.Allow))
	}
	if !policy.Allow[0].MatchString("Quelle: der Bericht") {
		t.Errorf("expected pattern to match")
	}
	if policy.Allow[0].MatchString("Normal text") {
		t.Errorf("expected pattern not to match")
	}
}

func TestPolicyUnknownCommitKey(t *testing.T) {
	root := writeConfig(t, `
[commit]
language = "en"
thresold = 4
`)
	_, err := commit.ReadPolicy(root)
	if err == nil {
		t.Fatal("expected error for typo key, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "'thresold'") || !strings.Contains(msg, "'threshold'") {
		t.Errorf("expected error to name unknown and known keys, got: %s", msg)
	}
}

func TestPolicyUnknownAllowKey(t *testing.T) {
	root := writeConfig(t, `
[commit]
language = "en"

[[commit.allow]]
regx = "^Fixes"
reason = "a trailer"
`)
	_, err := commit.ReadPolicy(root)
	if err == nil {
		t.Fatal("expected error for typo allow key, got nil")
	}
	if !strings.Contains(err.Error(), "'regx'") {
		t.Errorf("expected error to name 'regx', got: %v", err)
	}
}

func TestPolicyAllowNoPatternKey(t *testing.T) {
	root := writeConfig(t, `
[commit]
language = "en"

[[commit.allow]]
reason = "a trailer"
`)
	_, err := commit.ReadPolicy(root)
	if err == nil || !strings.Contains(err.Error(), "needs a `regex`") {
		t.Errorf("expected error asking for regex, got: %v", err)
	}
}

func TestPolicySchemaErrors(t *testing.T) {
	cases := []struct {
		body    string
		contain string
	}{
		{`[commit]\nlanguage = "fr"`, "must be one of"},
		{`[commit]\nlanguage = 1`, "must be one of"},
		{`[commit]\nthreshold = "zwei"`, "must be an integer"},
		{`[commit]\nthreshold = 0`, "must be greater than zero"},
		{`[commit]\nthreshold = -1`, "must be greater than zero"},
		{`[commit]\nthreshold = true`, "must be an integer"},
		{`[commit]\nconventional = "yes"`, "must be a boolean"},
		{`commit = "no table"`, "[commit] must be a table"},
		{`[commit]\nallow = "no list"`, "[[commit.allow]] must be a list of tables"},
		{`[commit]\nallow = [1]`, "[[commit.allow]] must be a list of tables"},
		{`[commit]\n[[commit.allow]]\nmatch = "a"\nreason = "x"`, "has no `match`"},
		{`[commit]\n[[commit.allow]]\nregex = "["\nreason = "x"`, "invalid regex"},
		{`[commit]\n[[commit.allow]]\nregex = "^x"`, "needs a `reason`"},
		{`[commit]\n[[commit.allow]]\nregex = "^x"\nreason = ""`, "needs a `reason`"},
		{`[commit]\n[[commit.allow]]\nregex = 1\nreason = "x"`, "must be a string"},
		{`[commit`, "config.toml"},
	}

	for _, c := range cases {
		// Replace literal \n with newline
		body := strings.ReplaceAll(c.body, `\n`, "\n")
		root := writeConfig(t, body)
		_, err := commit.ReadPolicy(root)
		if err == nil {
			t.Errorf("body %q: expected error containing %q, got nil", c.body, c.contain)
			continue
		}
		if !strings.Contains(err.Error(), c.contain) {
			t.Errorf("body %q: error %q does not contain %q", c.body, err.Error(), c.contain)
		}
	}
}

func TestPolicyMoreAllowVariations(t *testing.T) {
	// allow as inline array of tables
	root := writeConfig(t, `
[commit]
allow = [ { regex = "^x", reason = "r" } ]
`)
	p, err := commit.ReadPolicy(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Allow) != 1 {
		t.Errorf("expected 1 allow rule, got %d", len(p.Allow))
	}

	// allow as array of non-tables
	rootBad := writeConfig(t, `
[commit]
allow = [ "not a table" ]
`)
	_, err = commit.ReadPolicy(rootBad)
	if err == nil || !strings.Contains(err.Error(), "must be a list of tables") {
		t.Errorf("expected list of tables error, got: %v", err)
	}
}

func TestPolicyReadDirError(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".loomux", "config.toml"), 0o755)
	_, err := commit.ReadPolicy(root)
	if err == nil {
		t.Fatal("expected error reading directory, got nil")
	}
}

func TestPolicyAllowArrayAnyItemError(t *testing.T) {
	root := writeConfig(t, `
[commit]
allow = [ { regex = "[", reason = "r" } ]
`)
	_, err := commit.ReadPolicy(root)
	if err == nil || !strings.Contains(err.Error(), "invalid regex") {
		t.Errorf("expected invalid regex error, got: %v", err)
	}
}

func TestKnownKeysAreTheOnesTheReaderAccepts(t *testing.T) {
	if !slices.Equal(commit.KnownKeys(), []string{"allow", "conventional", "language", "threshold"}) {
		t.Fatal(commit.KnownKeys())
	}
}
