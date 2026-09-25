package setup

import (
	"strings"
	"testing"
)

func TestABrokenMCPFileStopsThePlan(t *testing.T) {
	for _, text := range []string{"{not json", "[]", `{"mcpServers": []}`} {
		root := world(t, map[string]string{".mcp.json": text})
		_, err := Build(gather(t, root, ""), DefaultChoice(gather(t, root, ""), Answers{}), reader(root))
		if err == nil || !strings.Contains(err.Error(), ".mcp.json") {
			t.Errorf("%s: err = %v", text, err)
		}
	}
}

func TestAnExistingMCPFileKeepsItsServers(t *testing.T) {
	text := "{\"other\": 1, \"mcpServers\": {\"z\": {\"b\": 1, \"a\": 2}}}"
	root := world(t, map[string]string{".mcp.json": text})
	mcp, _ := changeOf(plan(t, gather(t, root, "")), ".mcp.json")
	want := "{\n  \"other\": 1,\n  \"mcpServers\": {\n    \"z\": {\n      \"b\": 1,\n      \"a\": 2\n    },\n    \"loomux\": {"
	if !mcp.Exists || !strings.HasPrefix(mcp.After, want) {
		t.Errorf(".mcp.json =\n%s", mcp.After)
	}
	// A server loomux of the project stays as it is.
	root = world(t, map[string]string{".mcp.json": `{"mcpServers": {"loomux": {"command": "x"}}}`})
	if _, ok := changeOf(plan(t, gather(t, root, "")), ".mcp.json"); ok {
		t.Errorf("an existing loomux server is rewritten")
	}
	// An object without mcpServers gains it.
	root = world(t, map[string]string{".mcp.json": `{"a": true}`})
	mcp, _ = changeOf(plan(t, gather(t, root, "")), ".mcp.json")
	if !strings.HasPrefix(mcp.After, "{\n  \"a\": true,\n  \"mcpServers\": {") {
		t.Errorf(".mcp.json =\n%s", mcp.After)
	}
}

func TestAUserScopeServerLeavesMCPJsonAlone(t *testing.T) {
	root := world(t, map[string]string{})
	f := gather(t, root, "")
	f.UserMCP = true
	p := plan(t, f)
	if _, ok := changeOf(p, ".mcp.json"); ok || !hasNote(p, "user scope") {
		t.Errorf("changes = %v, notes = %v", paths(p), p.Notes)
	}
}

func TestAnMCPFileKeepsForeignCommandsUnescaped(t *testing.T) {
	text := `{"mcpServers": {"a<b>": {"command": "a && b", "args": ["<x>"]}}}`
	root := world(t, map[string]string{".mcp.json": text})
	mcp, _ := changeOf(plan(t, gather(t, root, "")), ".mcp.json")
	for _, want := range []string{`"a<b>"`, `"a && b"`, `"<x>"`} {
		if !strings.Contains(mcp.After, want) {
			t.Errorf("no %s in\n%s", want, mcp.After)
		}
	}
}
