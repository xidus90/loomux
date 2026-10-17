package hooks

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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

// switchWorld is world with the area's llm_config switch on.
func switchWorld(t *testing.T) (root, state string) {
	t.Helper()
	root, state = world(t, "")
	registry, err := os.ReadFile(filepath.Join(state, "registry.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "registry.toml"), append(registry, "llm_config = true\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, state
}

// Every refusal in a project whose switch is on ends by saying so, the
// policy's and the barrier's alike; with the switch off, none does.
func TestARefusalSaysTheSwitchIsOn(t *testing.T) {
	const want = "llm_config is on for this project: agents may edit .loomux/config.toml (turn it off with loomux config llm off)\n"
	refused := func(root string) []string {
		return []string{
			payload("Write", "file_path", filepath.Join(root, ".env")),
			payload("Write", "file_path", filepath.Join(t.TempDir(), "elsewhere.md")),
		}
	}
	root, state := switchWorld(t)
	for _, input := range refused(root) {
		if code, out, errOut := call(t, root, state, input); code != 2 || !strings.HasSuffix(errOut, "\n\n"+want) || !strings.Contains(out, "llm_config is on") {
			t.Errorf("%s, switch on: code %d, err %q", input, code, errOut)
		}
	}
	root, state = world(t, "")
	for _, input := range refused(root) {
		if code, _, errOut := call(t, root, state, input); code != 2 || strings.Contains(errOut, "llm_config") {
			t.Errorf("%s, switch off: code %d, err %q", input, code, errOut)
		}
	}
}

// With the switch on, the project's own manifest passes both steps.
func TestTheOwnManifestPassesWhenTheSwitchIsOn(t *testing.T) {
	root, state := switchWorld(t)
	if code, _, errOut := call(t, root, state, payload("Edit", "file_path", filepath.Join(root, ".loomux", "config.toml"))); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

// registerArea adds an area with its switch off at dir, with a manifest of
// its own, to the registry of state.
func registerArea(t *testing.T, state, scope, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".loomux", "config.toml"), []byte(foreignText), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(state, "registry.toml"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("\n[[area]]\nscope = \"" + scope + "\"\npath = " + quote(dir) + "\nworkspace = true\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// shellAt is a Bash call of line from a shell standing in cwd; "" sends no cwd.
func shellAt(line, cwd string) string {
	if cwd == "" {
		return `{"tool_name":"Bash","tool_input":{"command":"` + line + `"}}`
	}
	return `{"tool_name":"Bash","cwd":` + quote(cwd) + `,"tool_input":{"command":"` + line + `"}}`
}

// The hook judges for the root it was installed with, while a shell keeps
// the directory it moved to. After a cd into another project -- nested or
// beside -- a config set and a relative redirect would land in that
// project's manifest, so the root's switch does not open them there; nor
// does it when the call does not say where the shell stands.
func TestTheSwitchOpensTheShellOnlyAtTheRoot(t *testing.T) {
	const set = "loomux config set guard.mode off --yes"
	const redirect = "echo x > .loomux/config.toml"
	root, state := switchWorld(t)
	nested := filepath.Join(root, "sub", "C")
	beside := t.TempDir()
	registerArea(t, state, "project/nested", nested)
	registerArea(t, state, "project/beside", beside)
	for _, cwd := range []string{nested, beside, ""} {
		if code, _, errOut := call(t, root, state, shellAt(set, cwd)); code != 2 || !strings.Contains(errOut, "a human runs them") {
			t.Errorf("config set in %q: code %d, err %q", cwd, code, errOut)
		}
		if code, _, errOut := call(t, root, state, shellAt(redirect, cwd)); code != 2 || !strings.Contains(errOut, manifestReason) {
			t.Errorf("redirect in %q: code %d, err %q", cwd, code, errOut)
		}
	}
	for _, line := range []string{set, redirect} {
		if code, _, errOut := call(t, root, state, shellAt(line, root)); code != 0 {
			t.Errorf("%q at the root: code %d, err %q", line, code, errOut)
		}
	}
	// A writing tool names its file in full, so where the shell stands
	// does not matter to it.
	edit := `{"tool_name":"Edit","cwd":` + quote(nested) + `,"tool_input":{"file_path":` + quote(filepath.Join(root, ".loomux", "config.toml")) + `}}`
	if code, _, errOut := call(t, root, state, edit); code != 0 {
		t.Errorf("Edit from %q: code %d, err %q", nested, code, errOut)
	}
}

// Another area's switch opens nothing in this project: with the root's
// switch off, a path inside it that a link leads to a switched-on area's
// manifest stays shut.
func TestAnotherAreasSwitchOpensNothingHere(t *testing.T) {
	root, state := world(t, "")
	other := t.TempDir()
	if err := os.MkdirAll(filepath.Join(other, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(state, "registry.toml"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("\n[[area]]\nscope = \"project/other\"\npath = " + quote(other) + "\nworkspace = true\nllm_config = true\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "lnk")
	if err := os.Symlink(filepath.Join(other, ".loomux"), link); err != nil {
		if out, junction := exec.Command("cmd", "/c", "mklink", "/J", link, filepath.Join(other, ".loomux")).CombinedOutput(); junction != nil {
			t.Skipf("no such link here: %v (%s)", junction, out)
		}
	}
	for _, path := range []string{filepath.Join(link, "config.toml"), filepath.Join(other, ".loomux", "config.toml")} {
		if code, _, errOut := call(t, root, state, payload("Write", "file_path", path)); code != 2 || !strings.Contains(errOut, "the manifest is where the barrier reads its own limits") {
			t.Errorf("%s: code %d, err %q", path, code, errOut)
		}
	}
}

// linkedWorld is switchWorld whose manifest at the root is reached through a
// link that link makes to the manifest of a second, registered area with
// the switch off; it answers that manifest. A link the OS refuses skips.
func linkedWorld(t *testing.T, link func(foreign, root string) error) (root, state, foreign string) {
	t.Helper()
	root, state = switchWorld(t)
	other := t.TempDir()
	foreign = filepath.Join(other, ".loomux", "config.toml")
	if err := os.MkdirAll(filepath.Dir(foreign), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreign, []byte(foreignText), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(state, "registry.toml"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("\n[[area]]\nscope = \"project/other\"\npath = " + quote(other) + "\nwiki = " + quote(filepath.Join(other, "docs", "wiki")) + "\nworkspace = true\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := link(foreign, root); err != nil {
		t.Skipf("no such link here: %v", err)
	}
	return root, state, foreign
}

const foreignText = "[commit]\nlanguage = \"en\"\n"

// linkFile makes the root's .loomux/config.toml a symlink to foreign.
func linkFile(foreign, root string) error {
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		return err
	}
	return os.Symlink(foreign, filepath.Join(root, ".loomux", "config.toml"))
}

// hardLinkFile makes the root's .loomux/config.toml a second name of foreign.
func hardLinkFile(foreign, root string) error {
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		return err
	}
	return os.Link(foreign, filepath.Join(root, ".loomux", "config.toml"))
}

// linkFolder makes the root's .loomux a link to foreign's folder: a symlink
// where the OS allows one, otherwise a Windows junction, which needs no
// privilege.
func linkFolder(foreign, root string) error {
	link := filepath.Join(root, ".loomux")
	err := os.Symlink(filepath.Dir(foreign), link)
	if err != nil && filepath.Separator == '\\' {
		if out, junction := exec.Command("cmd", "/c", "mklink", "/J", link, filepath.Dir(foreign)).CombinedOutput(); junction != nil {
			return fmt.Errorf("%v (%s)", junction, out)
		}
		return nil
	}
	return err
}

// Through a link the manifest at the root is another area's: with this
// project's switch on, the write is refused all the same, and so is the
// config set, which would land in that file. No note says agents may edit
// the manifest, since here they may not.
func TestALinkedManifestStaysShutWhenTheSwitchIsOn(t *testing.T) {
	for name, link := range map[string]func(string, string) error{"file": linkFile, "folder": linkFolder, "hard": hardLinkFile} {
		t.Run(name, func(t *testing.T) {
			root, state, foreign := linkedWorld(t, link)
			code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, ".loomux", "config.toml")))
			if code != 2 || !strings.Contains(errOut, "the manifest is where the barrier reads its own limits") || strings.Contains(errOut, "llm_config is on") {
				t.Errorf("Write: code %d, err %q", code, errOut)
			}
			code, _, errOut = call(t, root, state, shellAt("loomux config set commit.language de --yes", root))
			if code != 2 || !strings.Contains(errOut, "a human runs them") || strings.Contains(errOut, "llm_config is on") {
				t.Errorf("config set: code %d, err %q", code, errOut)
			}
			if data, err := os.ReadFile(foreign); err != nil || string(data) != foreignText {
				t.Errorf("the linked file changed: %q %v", data, err)
			}
		})
	}
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
		t.Fatalf("an exit of 1 would let it through here; loomux refuses with 2, got %d", code)
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

// A guard mode the reader refuses refuses every call, as a broken
// configuration does.
func TestAGuardModeTheReaderRefusesRefusesTheCall(t *testing.T) {
	root := project(t)
	manifest(t, root, "[guard]\nmode = \"hard\"\n")
	payload := map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": "git status"}}
	if _, err := policyReasons(payload, root, switchOff, switchOff); err == nil || !strings.Contains(err.Error(), "[guard] mode must be one of") {
		t.Fatalf("err %v", err)
	}
}
