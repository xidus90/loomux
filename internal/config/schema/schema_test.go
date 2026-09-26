package schema

import (
	"cmp"
	"slices"
	"strconv"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/verify"
	"github.com/xidus90/loomux/internal/verify/commit"
)

func ids(section string) []string {
	var out []string
	for _, k := range Keys() {
		if k.Section == section && k.Name != "" {
			out = append(out, k.Name)
		}
	}
	slices.Sort(out)
	return out
}

func sorted(s []string) []string { c := slices.Clone(s); slices.Sort(c); return c }

func TestTheSchemaKnowsEveryKeyTheReadersRead(t *testing.T) {
	for section, keys := range config.DeclarationKeys() {
		if got := ids(section); !slices.Equal(got, sorted(keys)) {
			t.Errorf("[%s]: schema %v, reader %v", section, got, sorted(keys))
		}
	}
	if got := ids("modules"); !slices.Equal(got, sorted(config.ModuleKeys())) {
		t.Errorf("[modules]: %v", got)
	}
	if got := ids("commit"); !slices.Equal(got, sorted(slices.DeleteFunc(commit.KnownKeys(), func(k string) bool { return k == "allow" }))) {
		t.Errorf("[commit]: %v", got)
	}
	if got := ids("verify"); !slices.Equal(got, sorted(verify.TopKeys())) {
		t.Errorf("[verify]: %v", got)
	}
	if got := ids("worktree"); !slices.Equal(got, []string{"mirror"}) {
		t.Errorf("[worktree]: %v", got)
	}
	for _, list := range []string{"commit.allow", "policy.paths.rules", "policy.commands.rules"} {
		k, ok := Lookup(list)
		if !ok || k.Kind != TableList {
			t.Errorf("%s must be a TableList", list)
		}
	}
}

func TestEveryKeyHasAModuleAndADoc(t *testing.T) {
	for _, k := range Keys() {
		if k.Module == "" || k.Doc == "" {
			t.Errorf("%s lacks module or doc", k.ID())
		}
		if k.Kind == Enum && len(k.Choices) == 0 {
			t.Errorf("%s is an enum without choices", k.ID())
		}
	}
}

func TestLookupFindsByID(t *testing.T) {
	k, ok := Lookup("commit.language")
	if !ok || k.Kind != Enum || !slices.Equal(k.Choices, []string{"en", "de"}) || k.Default != `"en"` {
		t.Fatalf("%+v %v", k, ok)
	}
	if _, ok := Lookup("commit.nope"); ok {
		t.Fatal("unknown id found")
	}
}

func TestTheGlobalKeysAreTheModelReadersKeys(t *testing.T) {
	var got []string
	for _, k := range GlobalKeys() {
		if k.Section != "model" || k.Module != Brain || k.Doc == "" {
			t.Errorf("%+v", k)
		}
		got = append(got, k.Name)
	}
	if !slices.Equal(sorted(got), sorted(config.GlobalModelKeys())) {
		t.Fatalf("schema %v, reader %v", got, config.GlobalModelKeys())
	}
	defaults := map[string]string{}
	for _, k := range GlobalKeys() {
		defaults[k.Name] = k.Default
	}
	if defaults["enabled"] != "false" || defaults["temperature"] != "0.0" ||
		defaults["endpoint"] != strconv.Quote(config.DefaultModelEndpoint) ||
		defaults["name"] != strconv.Quote(config.DefaultModelName) ||
		defaults["roles"] != "{ describe = true, place = true, propose = true }" {
		t.Fatalf("%v", defaults)
	}
}

func TestCurrentOfReadsTheKeysItIsGiven(t *testing.T) {
	entries, err := CurrentOf("global.toml", GlobalKeys(), "[model]\nendpoint = \"http://localhost:1\"\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		switch e.Key.Name {
		case "endpoint":
			if e.Origin != Set || e.Input != "http://localhost:1" {
				t.Errorf("%+v", e)
			}
		case "temperature":
			if e.Origin != Default || e.Value != "0.0" {
				t.Errorf("%+v", e)
			}
		}
	}
}

func TestATableIsNamedByItsSection(t *testing.T) {
	if id := (Key{Section: "policy.paths.rules"}).ID(); id != "policy.paths.rules" {
		t.Fatalf("got %q", id)
	}
	if id := (Key{Section: "commit", Name: "language"}).ID(); id != "commit.language" {
		t.Fatalf("got %q", id)
	}
}

// The listing groups keys by module first, so a project that switches a
// module off can skip one contiguous block.
func TestKeysAreSortedByModuleSectionName(t *testing.T) {
	keys := Keys()
	order := []Module{Base, Hooks, Brain, Graph}
	if !slices.IsSortedFunc(keys, func(a, b Key) int {
		return cmp.Or(cmp.Compare(slices.Index(order, a.Module), slices.Index(order, b.Module)),
			cmp.Compare(a.Section, b.Section), cmp.Compare(a.Name, b.Name))
	}) {
		t.Fatal("keys are not sorted by module, section, name")
	}
	if keys[0].Module != Base || keys[len(keys)-1].Module != Brain {
		t.Fatalf("first %s, last %s", keys[0].Module, keys[len(keys)-1].Module)
	}
}

// Callers may edit what they get; the next call must not see it.
func TestKeysAreBuiltAfresh(t *testing.T) {
	Keys()[0].Doc = "changed"
	if Keys()[0].Doc == "changed" {
		t.Fatal("Keys shares its slice between calls")
	}
}

// Defaults the readers set in code must be the ones the schema shows.
func TestDefaultsFollowTheReaders(t *testing.T) {
	for id, want := range map[string]string{
		"wiki.untouched_days": "180",
		"verify.timeout":      "600",
		"privacy.mode":        `"manual_cloud"`,
		"commit.threshold":    "2",
	} {
		if k, _ := Lookup(id); k.Default != want {
			t.Errorf("%s: default %q, want %q", id, k.Default, want)
		}
	}
	if k, _ := Lookup("privacy.mode"); !slices.Equal(k.Choices, []string{"automatic_cloud", "local_only", "manual_cloud"}) {
		t.Errorf("privacy.mode choices %v", k.Choices)
	}
	if k, _ := Lookup("verify.max_parallel"); k.Default != "" {
		t.Errorf("max_parallel depends on the machine, found default %q", k.Default)
	}
}

// The switch says what it gates today; convert and fetch do not exist yet,
// and a doc that names them promises a switch over nothing.
func TestTheBrainSwitchNamesWhatItGatesToday(t *testing.T) {
	k, _ := Lookup("modules.brain")
	if k.Doc != "Run the brain module: its MCP tools and the wiki lane." {
		t.Fatalf("doc %q", k.Doc)
	}
}
