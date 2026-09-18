package mcptools_test

import (
	"encoding/json"
	"testing"

	"github.com/xidus90/loomux/internal/mcptools"
)

func TestToolsAreTheFiveInCanonicalOrder(t *testing.T) {
	got := mcptools.Tools()
	want := []string{"brain_search", "brain_catalog", "brain_read", "brain_neighbors", "brain_status"}
	if len(got) != len(want) {
		t.Fatalf("got %d tools, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i] {
			t.Errorf("tool %d is %q, want %q", i, got[i].Name, want[i])
		}
	}
}

func TestToolsAreByteIdenticalAcrossCalls(t *testing.T) {
	// In-process this can only fail if sync.Once is broken or build() grew a
	// map ranged into a slice: it pins that one process always serves the same
	// bytes, nothing more. It cannot see drift between the bridge's list and
	// serve's, because both read this very slice here; the test that compares
	// the two fronts is Task 11's.
	first, err := json.Marshal(mcptools.Tools())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	second, err := json.Marshal(mcptools.Tools())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(first) != string(second) {
		t.Error("two calls to Tools() marshal differently")
	}
}

func TestSearchProfilesAreTheReferenceNames(t *testing.T) {
	schema := schemaOf(t, "brain_search")
	props := schema["properties"].(map[string]any)
	profile := props["profile"].(map[string]any)
	got := profile["enum"].([]any)
	want := []string{"fast", "full", "keyword"}
	if len(got) != len(want) {
		t.Fatalf("profile enum has %d values, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].(string) != want[i] {
			t.Errorf("profile %d is %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSearchCountDefaultsToTenOverMCP(t *testing.T) {
	// The command line defaults to 5 (cli.py:483), the MCP front to 10
	// (daemon/tools.py). Parity here is with the MCP front.
	schema := schemaOf(t, "brain_search")
	props := schema["properties"].(map[string]any)
	n := props["n"].(map[string]any)
	if n["default"].(float64) != 10 {
		t.Errorf("n default is %v, want 10", n["default"])
	}
}

func TestSearchRequiresQuery(t *testing.T) {
	schema := schemaOf(t, "brain_search")
	required := schema["required"].([]any)
	if len(required) != 1 {
		t.Fatalf("search requires %d fields, want 1", len(required))
	}
	if required[0].(string) != "query" {
		t.Errorf("search requires %v, want [query]", required)
	}
}

func TestOnlyCountCarriesASchemaDefault(t *testing.T) {
	// The reference puts exactly one `default` into the tool schemas:
	// `daemon/tools.py:53` gives `n` its 10, while `_SCOPE` (line 17) and the
	// profile enum (line 52) carry none. That scope falls back to "all" and
	// profile to "fast" is behaviour of serve/brain/tools.go -- scope() and
	// profile() there -- when the field is absent, not a schema field; the
	// spec's table lists both kinds in one column. Reading that table as
	// schema defaults and adding them here would break the parity this stage
	// is accepted on, so the absence is pinned.
	//
	// `n` is the one field that is both: the schema default here, for the
	// host, and count() in serve/brain/tools.go, for the handler, because
	// nothing on the tool-call path applies a schema. The reference does the
	// same twice over (daemon/tools.py:53 and :167).
	search := schemaOf(t, "brain_search")["properties"].(map[string]any)
	for _, field := range []string{"scope", "profile"} {
		if _, ok := search[field].(map[string]any)["default"]; ok {
			t.Errorf("brain_search.%s carries a schema default; the reference has none", field)
		}
	}
	if _, ok := search["n"].(map[string]any)["default"]; !ok {
		t.Error("brain_search.n lost its default; the reference gives it 10")
	}
	catalog := schemaOf(t, "brain_catalog")["properties"].(map[string]any)
	if _, ok := catalog["scope"].(map[string]any)["default"]; ok {
		t.Error("brain_catalog.scope carries a schema default; the reference has none")
	}
}

func TestReadRequiresScopeAndRelative(t *testing.T) {
	schema := schemaOf(t, "brain_read")
	required := schema["required"].([]any)
	if len(required) != 2 {
		t.Fatalf("read requires %d fields, want 2", len(required))
	}
	if required[0].(string) != "scope" || required[1].(string) != "relative" {
		t.Errorf("read requires %v, want [scope relative]", required)
	}
}

func TestStatusTakesNoArguments(t *testing.T) {
	schema := schemaOf(t, "brain_status")
	// No `ok &&` guard: an empty properties object and a missing one are not
	// the same answer to a host, and the schema is to carry the empty one.
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("status has no properties object: %v", schema["properties"])
	}
	if len(props) != 0 {
		t.Errorf("status takes arguments: %v", props)
	}
}

func schemaOf(t *testing.T, name string) map[string]any {
	t.Helper()
	for _, tool := range mcptools.Tools() {
		if tool.Name != name {
			continue
		}
		raw, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatalf("marshal schema of %s: %v", name, err)
		}
		var schema map[string]any
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatalf("unmarshal schema of %s: %v", name, err)
		}
		return schema
	}
	t.Fatalf("no tool named %s", name)
	return nil
}
