package hooks

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/xidus90/loomux/internal/config"
)

// world builds a project with a registry that makes the whole project writable.
func world(t *testing.T, config string) (root, state string) {
	t.Helper()
	root = t.TempDir()
	state = t.TempDir()
	if config != "" {
		os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
		os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte(config), 0o644)
	}
	registry := "[[area]]\nscope = \"project/x\"\npath = " + quote(root) + "\nwiki = " + quote(filepath.Join(root, "docs", "wiki")) + "\nworkspace = true\n"
	os.WriteFile(filepath.Join(state, "registry.toml"), []byte(registry), 0o644)
	return root, state
}

func quote(path string) string { return `"` + filepath.ToSlash(path) + `"` }

func payload(tool, key, value string) string {
	return `{"tool_name":"` + tool + `","tool_input":{"` + key + `":` + quote(value) + `}}`
}

func call(t *testing.T, root, state, input string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := PreToolUse(strings.NewReader(input), &out, &errb, root, state)
	return code, out.String(), errb.String()
}

func TestAWriteInsideTheProjectIsAllowed(t *testing.T) {
	root, state := world(t, "")
	if code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, "main.go"))); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

// A configuration that declares only a policy declares no area, and the write
// barrier lets a write into the workspace through all the same (R7a).
func TestAWriteIsAllowedWhereTheConfigCarriesOnlyAPolicy(t *testing.T) {
	root, state := world(t, "[[policy.paths.rules]]\nmatch = \"generated/*\"\nreason = \"generated files are rebuilt, not edited\"\n")
	if code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, "main.go"))); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestABuiltinPolicyRuleRefusesFirst(t *testing.T) {
	root, state := world(t, "")
	code, out, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, ".env")))
	if code != 2 || !strings.Contains(errOut, "secrets are not written by an agent") || !strings.Contains(out, `"permissionDecision": "deny"`) {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestAConfiguredCommandRuleRefusesABashCall(t *testing.T) {
	root, state := world(t, "[[policy.commands.rules]]\nregex = '(^|\\s)pip\\s+install([^\\w-]|$)'\nreason = \"uv, never pip\"\n")
	code, _, errOut := call(t, root, state, `{"tool_name":"Bash","tool_input":{"command":"pip install requests"}}`)
	if code != 2 || !strings.Contains(errOut, "uv, never pip") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestAnUnreadablePolicyRefusesAndNamesTheFile(t *testing.T) {
	root, state := world(t, "[[policy.paths.rules]\n")
	code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, "main.go")))
	if code != 2 || !strings.Contains(errOut, "config.toml") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestTheConfigItselfIsNeverWritable(t *testing.T) {
	root, state := world(t, "[area]\nscope = \"project/x\"\n")
	code, _, _ := call(t, root, state, payload("Edit", "file_path", filepath.Join(root, ".loomux", "config.toml")))
	if code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestAWriteOutsideEveryAreaIsRefusedByTheBarrier(t *testing.T) {
	root, state := world(t, "")
	code, _, _ := call(t, root, state, payload("Write", "file_path", filepath.Join(t.TempDir(), "elsewhere.md")))
	if code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestAnUnreadablePayloadIsRefusedNotPassed(t *testing.T) {
	root, state := world(t, "")
	if code, _, _ := call(t, root, state, "not json"); code != 2 {
		t.Fatalf("ulguard answered 1 here; loomux refuses, got %d", code)
	}
}

// A payload that decodes but names no tool is nothing the policy can judge,
// and a call nobody judged does not pass: a host that spells its tool name
// under another key would otherwise run every call past every rule. The
// refusal comes before the policy file is read, so a broken one does not
// change its wording.
func TestACallWithoutAToolNameIsRefused(t *testing.T) {
	root, state := world(t, "[[policy.paths.rules]\n")
	const want = "loomux found no tool name in this call, so it cannot judge it and refuses"
	for _, input := range []string{
		`{"tool_input":{"file_path":"main.go"}}`,
		`{"tool_input":{"command":"git push"}}`,
		`{"toolCall":{"args":{"CommandLine":"git push"}}}`,
		`{"toolCall":"x"}`,
		`{"tool_name":42,"tool_input":{"command":"git push"}}`,
		`{}`,
	} {
		if code, _, errOut := call(t, root, state, input); code != 2 || !strings.Contains(errOut, want) {
			t.Fatalf("%s: code %d, err %q", input, code, errOut)
		}
	}
}

// A payload that is no object is the barrier's to refuse, in the barrier's
// words: the policy never sees it.
func TestAPayloadThatIsNoObjectKeepsTheBarriersWording(t *testing.T) {
	root, state := world(t, "")
	// null decodes into a nil map, and the barrier refuses it as no object.
	for _, input := range []string{"not json", "[]", `"x"`, "null"} {
		code, _, errOut := call(t, root, state, input)
		if code != 2 || strings.Contains(errOut, "no tool name") || strings.Contains(errOut, "loomux policy refused") {
			t.Fatalf("%s: code %d, err %q", input, code, errOut)
		}
	}
}

func TestAnAntigravityWriteIsJudgedByPolicyToo(t *testing.T) {
	root, state := world(t, "")
	input := `{"toolCall":{"name":"write_to_file","args":{"TargetFile":` + quote(filepath.Join(root, ".env")) + `}}}`
	code, _, errOut := call(t, root, state, input)
	if code != 2 || !strings.Contains(errOut, "secrets") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// A stdin that fails mid-read is not a call that may pass: nothing was judged.
func TestAnUnreadableStdinRefuses(t *testing.T) {
	root, state := world(t, "")
	var out, errb bytes.Buffer
	code := PreToolUse(iotest.ErrReader(os.ErrClosed), &out, &errb, root, state)
	if code != 2 || !strings.Contains(errb.String(), "cannot read the hook payload") {
		t.Fatalf("code %d, err %q", code, errb.String())
	}
}

func TestAPanicInThePolicyStepRefuses(t *testing.T) {
	root, state := world(t, "")
	readPolicy = func(string) (config.Policy, error) { panic("boom") }
	defer func() { readPolicy = config.ReadPolicy }()
	code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, "main.go")))
	if code != 2 || !strings.Contains(errOut, "boom") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}
