package schema

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
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

// segments are the keys directly under section as a reader sees them: a key's
// own name, or the first segment of a table below the section.
func segments(section string) []string {
	var out []string
	for _, k := range Keys() {
		switch rest, below := strings.CutPrefix(k.Section, section+"."); {
		case k.Section == section:
			out = append(out, k.Name)
		case below:
			first, _, _ := strings.Cut(rest, ".")
			out = append(out, first)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// A named key stands for one key per name, and Match fills exactly one name:
// every key with a star has one, as a whole segment of its ID. A star glued
// to a name would be neither a name nor a key Named sees.
func TestEveryNamedKeyHasOneWholeSegmentStar(t *testing.T) {
	for _, k := range Keys() {
		if !strings.Contains(k.ID(), Wildcard) {
			continue
		}
		if strings.Count(k.ID(), Wildcard) != 1 || strings.Count("."+k.ID()+".", "."+Wildcard+".") != 1 {
			t.Errorf("%s: want one %s as a whole segment", k.ID(), Wildcard)
		}
	}
}

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
	if got := ids("flow"); !slices.Equal(got, sorted(config.FlowKeys())) {
		t.Errorf("[flow]: schema %v, reader %v", got, config.FlowKeys())
	}
	// [agent] holds two of its keys as tables of their own, one row per name.
	if got := segments("agent"); !slices.Equal(got, sorted(config.AgentKeys())) {
		t.Errorf("[agent]: schema %v, reader %v", got, config.AgentKeys())
	}
	if got := ids("agent.models.*"); !slices.Equal(got, sorted(config.ModelSpecKeys())) {
		t.Errorf("[agent.models.*]: schema %v, reader %v", got, config.ModelSpecKeys())
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

func TestMatchFillsANamedKey(t *testing.T) {
	keys := Keys()
	for id, want := range map[string][2]string{
		"agent.roles.reviewer":         {"agent.roles", "reviewer"},
		"agent.models.gemini.provider": {"agent.models.gemini", "provider"},
		"agent.models.gemini.model":    {"agent.models.gemini", "model"},
		"agent.default":                {"agent", "default"},
		"flow.overrides":               {"flow", "overrides"},
	} {
		k, ok := Match(keys, id)
		if !ok || k.Section != want[0] || k.Name != want[1] || k.Named() {
			t.Errorf("Match(%q) = %+v, %v", id, k, ok)
		}
	}
	for _, id := range []string{"agent.roles.*", "agent.roles.a-b", "agent.roles", "agent.models.gemini", "agent.models.gemini.size", "agent.roles.x.y"} {
		if k, ok := Match(keys, id); ok {
			t.Errorf("Match(%q) = %+v, want no key", id, k)
		}
	}
}

func TestEveryNewKeyIsBase(t *testing.T) {
	for _, id := range []string{"agent.default", "agent.mcp_servers", "agent.models.*.provider", "agent.models.*.model", "agent.roles.*", "flow.default", "flow.overrides"} {
		found := false
		for _, k := range Keys() {
			if k.ID() == id {
				found = true
				if k.Module != Base {
					t.Errorf("%s is in module %s", id, k.Module)
				}
			}
		}
		if !found {
			t.Errorf("no key %s", id)
		}
	}
}

func TestTheGlobalKeysAreTheirReadersKeys(t *testing.T) {
	got := map[string][]string{}
	for _, k := range GlobalKeys() {
		if k.Module != Brain || k.Doc == "" {
			t.Errorf("%+v", k)
		}
		got[k.Section] = append(got[k.Section], k.Name)
	}
	readers := map[string][]string{"model": config.GlobalModelKeys(), "search": config.GlobalSearchKeys()}
	if len(got) != len(readers) {
		t.Fatalf("sections %v", got)
	}
	for section, keys := range readers {
		if !slices.Equal(sorted(got[section]), sorted(keys)) {
			t.Fatalf("[%s]: schema %v, reader %v", section, got[section], keys)
		}
	}
	defaults := map[string]string{}
	for _, k := range GlobalKeys() {
		defaults[k.ID()] = k.Default
	}
	if defaults["model.enabled"] != "false" || defaults["model.temperature"] != "0.0" ||
		defaults["model.endpoint"] != strconv.Quote(config.DefaultModelEndpoint) ||
		defaults["model.name"] != strconv.Quote(config.DefaultModelName) ||
		defaults["model.roles"] != "{ describe = true, place = true, propose = true }" ||
		defaults["search.backbone"] != strconv.Quote(config.DefaultSearchBackbone) {
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

// The switch says what it gates today: brain = false also makes convert and
// fetch refuse, and a doc without them would hide that.
func TestTheBrainSwitchNamesWhatItGatesToday(t *testing.T) {
	k, _ := Lookup("modules.brain")
	if k.Doc != "Run the brain module: its MCP tools, the wiki lane, convert and fetch." {
		t.Fatalf("doc %q", k.Doc)
	}
}

// model.enabled gates every role, describe and place as well as propose,
// so its doc names no single one.
func TestTheModelSwitchNamesNoSingleRole(t *testing.T) {
	k, _ := Lookup("model.enabled")
	if k.Doc != "Let the local model be asked for this area." {
		t.Fatalf("doc %q", k.Doc)
	}
}
