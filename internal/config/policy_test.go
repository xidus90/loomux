package config

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
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

// An empty expression matches every command, so a rule whose regex key is
// missing or misspelled would refuse every shell command.
func TestReadPolicyRefusesACommandRuleWithoutARegex(t *testing.T) {
	for name, body := range map[string]string{
		"missing":    "[[policy.commands.rules]]\nreason = \"x\"\n",
		"misspelled": "[[policy.commands.rules]]\nregexp = 'pip'\nreason = \"x\"\n",
		"empty":      "[[policy.commands.rules]]\nregex = ''\nreason = \"x\"\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := writeConfig(t, body)
			_, err := ReadPolicy(root)
			if err == nil || !strings.Contains(err.Error(), "config.toml") || !strings.Contains(err.Error(), "[[policy.commands.rules]] #1 needs a regex") {
				t.Fatalf("err %v", err)
			}
		})
	}
}

// A path rule without a glob, or with an empty one, never matches: the rule
// would load and protect nothing.
func TestReadPolicyRefusesAPathRuleWithAnEmptyGlob(t *testing.T) {
	for name, tc := range map[string]struct{ body, want string }{
		"empty list":         {"match = []", "match: needs at least one glob"},
		"empty string":       {"match = \"\"", "match: glob #1 is empty"},
		"empty list element": {"match = [\"a\", \"\"]", "match: glob #2 is empty"},
	} {
		t.Run(name, func(t *testing.T) {
			root := writeConfig(t, "[[policy.paths.rules]]\n"+tc.body+"\nreason = \"x\"\n")
			_, err := ReadPolicy(root)
			if err == nil || !strings.Contains(err.Error(), "config.toml") || !strings.Contains(err.Error(), "[[policy.paths.rules]] #1 "+tc.want) {
				t.Fatalf("err %v", err)
			}
		})
	}
}

// A glob that does not compile never matches either, and the rule then names
// a path it does not protect. The regex half of the same defect is refused two
// rules above; this is the path half.
func TestReadPolicyRefusesAPathRuleWithAMalformedGlob(t *testing.T) {
	for name, tc := range map[string]struct{ body, want string }{
		"unclosed class":  {"match = \"secrets/[a-z.env\"", "glob #1 \"secrets/[a-z.env\" is malformed"},
		"in a list":       {"match = [\"bin/*\", \"[\"]", "glob #2 \"[\" is malformed"},
		"bare class open": {"match = \"a[\"", "glob #1 \"a[\" is malformed"},
		// The guard's path.Match reads a backslash as an escape on every
		// platform, so one with nothing after it is refused on Windows too.
		"trailing escape": {`match = 'bin\'`, `glob #1 "bin\\" is malformed`},
	} {
		t.Run(name, func(t *testing.T) {
			root := writeConfig(t, "[[policy.paths.rules]]\n"+tc.body+"\nreason = \"x\"\n")
			_, err := ReadPolicy(root)
			if err == nil || !strings.Contains(err.Error(), "config.toml") || !strings.Contains(err.Error(), "[[policy.paths.rules]] #1 match: "+tc.want) {
				t.Fatalf("err %v", err)
			}
		})
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

func TestPolicyKeysNameBothRuleLists(t *testing.T) {
	keys := PolicyKeys()
	if !slices.Equal(keys["policy.paths.rules"], []string{"match", "reason"}) ||
		!slices.Equal(keys["policy.commands.rules"], []string{"regex", "reason"}) {
		t.Fatal(keys)
	}
}

// The reader decodes through struct tags, not through PolicyKeys; this holds
// the two equal so the schema cannot list a key the reader ignores.
func TestPolicyKeysAreTheTagsTheReaderDecodes(t *testing.T) {
	var file policyFile
	tags := func(rules reflect.Type) []string {
		var names []string
		for i := range rules.NumField() {
			names = append(names, rules.Field(i).Tag.Get("toml"))
		}
		return names
	}
	keys := PolicyKeys()
	if got := tags(reflect.TypeOf(file.Policy.Paths.Rules).Elem()); !slices.Equal(got, keys["policy.paths.rules"]) {
		t.Errorf("paths rule tags %v, PolicyKeys %v", got, keys["policy.paths.rules"])
	}
	if got := tags(reflect.TypeOf(file.Policy.Commands.Rules).Elem()); !slices.Equal(got, keys["policy.commands.rules"]) {
		t.Errorf("commands rule tags %v, PolicyKeys %v", got, keys["policy.commands.rules"])
	}
}
