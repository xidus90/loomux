package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func decodeRules(t *testing.T, text string) map[string]any {
	t.Helper()
	doc := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

const rulesExample = `
[rules.layers.core]
paths   = ["godot/core/**", "godot/model/**"]
may_use = ["core", "data"]
bases   = ["RefCounted", "Resource"]
tests   = true

[rules.layers.data]
paths = "godot/data/**"
may_use = []

[rules.layers.harness]
paths   = ["godot/harness/**"]
checked = false
`

func TestParseRules(t *testing.T) {
	cases := map[string]struct {
		text string
		want Rules
		errs []string // every piece must be in the one message
	}{
		"no section": {text: "[flow]\n", want: Rules{}},
		"empty":      {text: "[rules]\n", want: Rules{Present: true}},
		"example": {text: rulesExample, want: Rules{Present: true, Layers: []Layer{
			{Name: "core", Paths: []string{"godot/core/**", "godot/model/**"}, MayUse: []string{"core", "data"}, Bases: []string{"RefCounted", "Resource"}, HasMayUse: true, HasBases: true, Tests: true, Checked: true},
			{Name: "data", Paths: []string{"godot/data/**"}, MayUse: []string{}, HasMayUse: true, Checked: true},
			{Name: "harness", Paths: []string{"godot/harness/**"}},
		}}},
		"rules not a table":  {text: "rules = 1\n", errs: []string{"[rules] must be a table, found integer"}},
		"unknown rules key":  {text: "[rules]\nnope = 1\n", errs: []string{`[rules] does not know "nope"`}},
		"layers not a table": {text: "[rules]\nlayers = 1\n", errs: []string{"[rules.layers] must be a table of layers, found integer"}},
		"layer not a table":  {text: "[rules.layers]\ncore = 1\n", errs: []string{"[rules.layers.core] must be a table, found integer"}},
		"unknown layer key":  {text: "[rules.layers.a]\npaths = [\"a/**\"]\nnope = 1\n", errs: []string{`[rules.layers.a] does not know "nope"`}},
		"bad layer name":     {text: "[rules.layers.\"A b\"]\npaths = [\"a\"]\n", errs: []string{`"A b" is not a layer name`}},
		"missing paths":      {text: "[rules.layers.a]\n", errs: []string{"[rules.layers.a] needs paths"}},
		"empty paths":        {text: "[rules.layers.a]\npaths = []\n", errs: []string{"paths needs at least one glob"}},
		"empty glob":         {text: "[rules.layers.a]\npaths = [\"\"]\n", errs: []string{"glob #1 is empty"}},
		"malformed glob":     {text: "[rules.layers.a]\npaths = [\"[\"]\n", errs: []string{"glob #1", "malformed"}},
		"backslash glob":     {text: "[rules.layers.a]\npaths = [\"godot\\\\core\\\\**\"]\n", errs: []string{"contains a backslash"}},
		"paths wrong type":   {text: "[rules.layers.a]\npaths = 1\n", errs: []string{"must be a string or a list of strings"}},
		"unknown may_use":    {text: "[rules.layers.a]\npaths = [\"a\"]\nmay_use = [\"b\"]\n", errs: []string{`may_use names "b", which is not under [rules.layers]; known: a`}},
		"may_use not list":   {text: "[rules.layers.a]\npaths = [\"a\"]\nmay_use = 1\n", errs: []string{"may_use must be a list of strings, found integer"}},
		"bases bad element":  {text: "[rules.layers.a]\npaths = [\"a\"]\nbases = [\"\", 2]\n", errs: []string{"bases #1 must be a non-empty string", "bases #2 must be a non-empty string"}},
		"tests not bool":     {text: "[rules.layers.a]\npaths = [\"a\"]\ntests = \"yes\"\n", errs: []string{"tests must be a boolean"}},
		"checked not bool":   {text: "[rules.layers.a]\npaths = [\"a\"]\nchecked = \"no\"\n", errs: []string{"checked must be a boolean, found string"}},
		"unchecked with all": {
			text: "[rules.layers.a]\npaths = [\"a\"]\nchecked = false\nmay_use = []\nbases = []\ntests = true\n",
			errs: []string{"checked = false cannot carry may_use", "checked = false cannot carry bases", "checked = false cannot carry tests"},
		},
		"all at once": {
			text: "[rules.layers.a]\nnope = 1\npaths = []\nmay_use = [\"zz\"]\n",
			errs: []string{`does not know "nope"`, "needs at least one glob", `names "zz"`},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ParseRules("c.toml", decodeRules(t, c.text))
			if len(c.errs) == 0 {
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, c.want) {
					t.Fatalf("got %+v, want %+v", got, c.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("want an error, got %+v", got)
			}
			if !strings.HasPrefix(err.Error(), "c.toml: ") {
				t.Errorf("the file must be named first: %v", err)
			}
			for _, piece := range c.errs {
				if !strings.Contains(err.Error(), piece) {
					t.Errorf("want %q in: %v", piece, err)
				}
			}
			if strings.Count(err.Error(), "c.toml") != 1 {
				t.Errorf("one message, one file name: %v", err)
			}
		})
	}
}

func TestAllFindingsAreJoinedWithSemicolons(t *testing.T) {
	_, err := ParseRules("c.toml", decodeRules(t, "[rules.layers.a]\nnope = 1\npaths = []\n"))
	if err == nil || !strings.Contains(err.Error(), "; ") {
		t.Fatalf("want findings joined by %q: %v", "; ", err)
	}
}

func TestRulesKeysAreSorted(t *testing.T) {
	if !reflect.DeepEqual(RulesKeys(), []string{"layers"}) {
		t.Fatal(RulesKeys())
	}
}

func TestReadRules(t *testing.T) {
	root := t.TempDir()
	if rules, err := ReadRules(root); err != nil || rules.Present {
		t.Fatalf("no file: %+v %v", rules, err)
	}
	path := ManifestPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(rulesExample), 0o644); err != nil {
		t.Fatal(err)
	}
	rules, err := ReadRules(root)
	if err != nil || !rules.Present || len(rules.Layers) != 3 {
		t.Fatalf("%+v %v", rules, err)
	}
	if err := os.WriteFile(path, []byte("[rules.layers.a]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRules(root); err == nil || !strings.Contains(err.Error(), "needs paths") {
		t.Fatalf("a broken map must fail: %v", err)
	}
}
