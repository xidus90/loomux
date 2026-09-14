package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// posix spells a path the way the registry on this machine spells one:
// forward slashes, drive letter kept. `_registry` in the removed Python
// suite wrote `area.as_posix()` for the same reason -- a backslash inside
// a TOML string is an escape.
func posix(p string) string { return filepath.ToSlash(p) }

// registryOf mirrors `_registry`: one area `project/demo` rooted at
// <tmp>/repo, with the wiki and whichever flags the case varies.
func registryOf(t *testing.T, tmp, wiki string, flags ...string) string {
	t.Helper()
	state := filepath.Join(tmp, "state")
	mkdir(t, state)
	mkdir(t, filepath.Join(tmp, "repo"))
	body := "[[area]]\nscope = \"project/demo\"\npath = \"" +
		posix(filepath.Join(tmp, "repo")) + "\"\n"
	if wiki != "" {
		body += "wiki = \"" + posix(wiki) + "\"\n"
	}
	for _, flag := range flags {
		body += flag + "\n"
	}
	write(t, filepath.Join(state, "registry.toml"), body)
	return state
}

// nestedOf mirrors `_nested_registry`, the shape the real registration
// has: a writable `hub` whose wiki encloses the read-only wiki of
// `project/space`.
func nestedOf(t *testing.T, tmp string) string {
	t.Helper()
	state := filepath.Join(tmp, "state")
	mkdir(t, state)
	repo := filepath.Join(tmp, "repo")
	mkdir(t, repo)
	hub := filepath.Join(tmp, "vault", "91")
	body := "[[area]]\nscope = \"hub\"\npath = \"" + posix(hub) +
		"\"\nwiki = \"" + posix(hub) + "\"\n\n" +
		"[[area]]\nscope = \"project/space\"\npath = \"" + posix(repo) +
		"\"\nwiki = \"" + posix(filepath.Join(hub, "space")) +
		"\"\nreadonly = true\n"
	write(t, filepath.Join(state, "registry.toml"), body)
	return state
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	mkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeCall is `_write`.
func writeCall(path string) map[string]any {
	return map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]any{"file_path": path},
	}
}

func allow(t *testing.T, payload map[string]any, state string) {
	t.Helper()
	reason, refused := Decide(payload, state)
	if refused {
		t.Fatalf("expected allowed, refused with %q", reason)
	}
}

func deny(t *testing.T, payload map[string]any, state, want string) string {
	t.Helper()
	reason, refused := Decide(payload, state)
	if !refused {
		t.Fatal("expected a refusal, the call was allowed")
	}
	if want != "" && !strings.Contains(reason, want) {
		t.Fatalf("reason %q does not carry %q", reason, want)
	}
	return reason
}

// --- the writable tree --------------------------------------------------

func TestAPathInsideAWikiBundlePasses(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	allow(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state)
}

func TestARawSourcePathIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	deny(t, writeCall(filepath.Join(tmp, "repo", "src", "a.py")), state,
		"lies outside every writable tree")
}

func TestDotDotOutOfTheBundleIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	climb := filepath.Join(tmp, "vault", "demo", "..", "..", "repo", "a.py")
	deny(t, writeCall(climb), state, "lies outside every writable tree")
}

func TestToolsThatDoNotWriteAreNoneOfItsBusiness(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{
		"tool_name":  "Read",
		"tool_input": map[string]any{"file_path": filepath.Join(tmp, "x")},
	}
	allow(t, payload, state)
}

func TestEveryWritingToolIsJudged(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	// The list is `WRITING_TOOLS`, and it is
	// the whole scope of the promise -- a name missing from it is a tool
	// the barrier never sees.
	for _, tool := range []string{
		"Write", "Edit", "MultiEdit", "NotebookEdit", "write_to_file",
		"replace_file_content", "multi_replace_file_content",
	} {
		payload := map[string]any{
			"tool_name": tool,
			"tool_input": map[string]any{
				"file_path": filepath.Join(tmp, "repo", "a.py"),
			},
		}
		deny(t, payload, state, "lies outside every writable tree")
	}
}

func TestACallWithoutAFilePathIsNoneOfItsBusiness(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]any{"content": "x"},
	}
	allow(t, payload, state)
}

func TestAnEmptyTargetStringIsNoTarget(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	allow(t, writeCall(""), state)
}

func TestATargetThatIsNotAStringIsNoTarget(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]any{"file_path": 1.0},
	}
	allow(t, payload, state)
}

func TestAToolInputThatIsNotAnObjectIsNoneOfItsBusiness(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{"tool_name": "Write", "tool_input": "x"}
	allow(t, payload, state)
}

func TestAToolNameThatIsNotAStringIsNoneOfItsBusiness(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{
		"tool_name":  42.0,
		"tool_input": map[string]any{"file_path": filepath.Join(tmp, "x")},
	}
	allow(t, payload, state)
}

// --- the two payload forms ----------------------------------------------

func TestTheAntigravityPayloadFormIsJudgedToo(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{"toolCall": map[string]any{
		"name": "write_to_file",
		"args": map[string]any{
			"TargetFile": filepath.Join(tmp, "repo", "a.py"),
		},
	}}
	deny(t, payload, state, "lies outside every writable tree")
}

func TestTheAntigravityLowercaseTargetIsReadToo(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{"toolCall": map[string]any{
		"name": "replace_file_content",
		"args": map[string]any{
			"target_file": filepath.Join(tmp, "repo", "a.py"),
		},
	}}
	deny(t, payload, state, "lies outside every writable tree")
}

func TestAToolCallThatIsNotAnObjectIsNoneOfItsBusiness(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	allow(t, map[string]any{"toolCall": "x"}, state)
}

func TestAToolCallNameThatIsNotAStringIsNoneOfItsBusiness(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{"toolCall": map[string]any{
		"name": 1.0,
		"args": map[string]any{
			"TargetFile": filepath.Join(tmp, "repo", "a.py"),
		},
	}}
	allow(t, payload, state)
}

func TestAToolCallArgsThatAreNotAnObjectIsNoneOfItsBusiness(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{"toolCall": map[string]any{
		"name": "write_to_file",
		"args": "x",
	}}
	allow(t, payload, state)
}

func TestAToolCallKeyOutranksTheClaudeKeys(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	// `_extract_call` asks for `toolCall`
	// first and returns out of that branch whatever it finds, so the
	// Claude keys beside it are never read.
	payload := map[string]any{
		"toolCall":   map[string]any{"name": "Write", "args": "no"},
		"tool_name":  "Write",
		"tool_input": map[string]any{"file_path": filepath.Join(tmp, "a")},
	}
	allow(t, payload, state)
}

func TestANotebookWriteIsJudgedLikeAnyOther(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{
		"tool_name": "NotebookEdit",
		"tool_input": map[string]any{
			"notebook_path": filepath.Join(tmp, "repo", "a.ipynb"),
		},
	}
	deny(t, payload, state, "lies outside every writable tree")
}

func TestAPayloadNamingTwoPathsIsJudgedOnBoth(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{
		"tool_name": "Write",
		"tool_input": map[string]any{
			"file_path":     filepath.Join(tmp, "vault", "demo", "ok.md"),
			"notebook_path": filepath.Join(tmp, "repo", "bad.ipynb"),
		},
	}
	reason := deny(t, payload, state, "lies outside every writable tree")
	if strings.Contains(reason, "ok.md") {
		t.Fatalf("the allowed path was named as refused: %q", reason)
	}
}

func TestAPayloadNamingTwoAllowedPathsPasses(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{
		"tool_name": "Write",
		"tool_input": map[string]any{
			"file_path":     filepath.Join(tmp, "vault", "demo", "a.md"),
			"notebook_path": filepath.Join(tmp, "vault", "demo", "b.ipynb"),
		},
	}
	allow(t, payload, state)
}

// --- workspace and readonly ---------------------------------------------

func TestAWorkspaceAreaMayBeWritten(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"),
		"workspace = true")
	allow(t, writeCall(filepath.Join(tmp, "repo", "src", "a.py")), state)
}

func TestAWorkspaceDoesNotOpenItsNeighbours(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"),
		"workspace = true")
	deny(t, writeCall(filepath.Join(tmp, "other", "a.py")), state,
		"lies outside every writable tree")
}

func TestWithoutTheFieldTheAreaStaysShut(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	deny(t, writeCall(filepath.Join(tmp, "repo", "src", "a.py")), state,
		"lies outside every writable tree")
}

func TestAWorkspaceAloneIsEnoughWithoutAWiki(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "", "workspace = true")
	allow(t, writeCall(filepath.Join(tmp, "repo", "src", "a.py")), state)
}

func TestAReadonlyAreaIsNotAWritableTree(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"),
		"readonly = true")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"the registration calls this area read-only")
}

func TestAReadonlyAreaKeepsItsWorkspace(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"),
		"workspace = true", "readonly = true")
	allow(t, writeCall(filepath.Join(tmp, "repo", "src", "a.py")), state)
}

func TestAReadonlyAreaWithoutAWikiForbidsNothing(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "", "workspace = true", "readonly = true")
	allow(t, writeCall(filepath.Join(tmp, "repo", "a.py")), state)
}

func TestAReadonlyWikiBeatsTheAreaThatEnclosesIt(t *testing.T) {
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	target := filepath.Join(tmp, "vault", "91", "space", "x.md")
	deny(t, writeCall(target), state,
		"the registration calls this area read-only")
}

func TestTheEnclosingAreaKeepsItsOwnGround(t *testing.T) {
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	allow(t, writeCall(filepath.Join(tmp, "vault", "91", "x.md")), state)
}

func TestANeighbourThatOnlyLooksLikeAPrefixStaysWritable(t *testing.T) {
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	target := filepath.Join(tmp, "vault", "91", "space-neu", "x.md")
	allow(t, writeCall(target), state)
}

func TestTheReadonlyWikiPathItselfIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	deny(t, writeCall(filepath.Join(tmp, "vault", "91", "space")), state,
		"the registration calls this area read-only")
}

func TestAClimbIntoAReadonlyAreaIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	climb := filepath.Join(tmp, "vault", "91", "a", "..", "space", "x.md")
	deny(t, writeCall(climb), state,
		"the registration calls this area read-only")
}

func TestTheRefusalDoesNotOfferAReadonlyWikiAsAPlaceToWrite(t *testing.T) {
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	reason := deny(t, writeCall(filepath.Join(tmp, "out.md")), state,
		"lies outside every writable tree")
	if strings.Contains(reason, filepath.Join("91", "space")) {
		t.Fatalf("the read-only wiki was offered: %q", reason)
	}
}

func TestARegistryOfNothingButReadonlyAreasSaysSo(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"),
		"readonly = true")
	deny(t, writeCall(filepath.Join(tmp, "out.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

func TestARegistryWithoutAnyWikiRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	deny(t, writeCall(filepath.Join(tmp, "out.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

// --- the spellings a file system equates --------------------------------

func TestATrailingDotDoesNotSlipPastTheZone(t *testing.T) {
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	target := filepath.Join(tmp, "vault", "91", "space.") +
		string(filepath.Separator) + "x.md"
	deny(t, writeCall(target), state,
		"the registration calls this area read-only")
}

func TestATrailingBlankDoesNotSlipPastTheZone(t *testing.T) {
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	target := filepath.Join(tmp, "vault", "91", "space ") +
		string(filepath.Separator) + "x.md"
	deny(t, writeCall(target), state,
		"the registration calls this area read-only")
}

// The pair above with the directory present. This is the case the last
// finding sat in (`_inside`): `resolve`
// closes a spelling only where the path exists, so the answer may differ
// between the two states of the disk and both have to be measured.
func TestATrailingDotIsStoppedWithTheZoneDirectoryPresent(t *testing.T) {
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	mkdir(t, filepath.Join(tmp, "vault", "91", "space"))
	target := filepath.Join(tmp, "vault", "91", "space.") +
		string(filepath.Separator) + "x.md"
	deny(t, writeCall(target), state, "")
}

func TestATrailingBlankIsStoppedWithTheZoneDirectoryPresent(t *testing.T) {
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	mkdir(t, filepath.Join(tmp, "vault", "91", "space"))
	target := filepath.Join(tmp, "vault", "91", "space ") +
		string(filepath.Separator) + "x.md"
	deny(t, writeCall(target), state, "")
}

func TestACaseVariantOfTheZoneIsStopped(t *testing.T) {
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	target := filepath.Join(tmp, "vault", "91", "SPACE", "x.md")
	deny(t, writeCall(target), state,
		"the registration calls this area read-only")
}

func TestADirectoryIsJudgedLikeAFile(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	mkdir(t, filepath.Join(tmp, "repo", "src"))
	deny(t, writeCall(filepath.Join(tmp, "repo", "src")), state,
		"lies outside every writable tree")
}

// --- the manifest, and the guard name that is no longer special ----------

func TestTheBundledManifestIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	target := filepath.Join(tmp, "vault", "demo", ".loomux",
		"config.toml")
	deny(t, writeCall(target), state,
		"the manifest is where the barrier reads its own limits")
}

func TestAConfigTomlOutsideTheBundleDirectoryIsOrdinary(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	target := filepath.Join(tmp, "vault", "demo", "config.toml")
	allow(t, writeCall(target), state)
}

// loomux reads one manifest. ultra-brain's legacy `.brain.toml` and its
// `.ultra-brain/config.toml` are ordinary names here, and a barrier that
// still locked them would refuse writes nothing reads its limits from.
func TestTheOldManifestNamesAreOrdinaryFiles(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	bundle := filepath.Join(tmp, "vault", "demo")
	for _, name := range []string{
		".brain.toml",
		filepath.Join(".ultra-brain", "config.toml"),
	} {
		allow(t, writeCall(filepath.Join(bundle, name)), state)
	}
}

func TestTheManifestSpellingsAreEquated(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	bundle := filepath.Join(tmp, "vault", "demo")
	for _, name := range []string{
		filepath.Join(".loomux", "CONFIG.TOML"),
		filepath.Join(".loomux", "config.toml::$DATA"),
		filepath.Join(".LOOMUX", "CONFIG.TOML"),
		filepath.Join(".loomux ", "config.toml"),
	} {
		target := filepath.Join(bundle, name)
		deny(t, writeCall(target), state,
			"the manifest is where the barrier reads its own limits")
	}
}

func TestATrailingDotOnTheManifestIsTheManifest(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	target := filepath.Join(tmp, "vault", "demo", ".loomux", "config.toml") + "."
	deny(t, writeCall(target), state,
		"the manifest is where the barrier reads its own limits")
}

// The barrier used to pin `<bundle>/.ultra-brain/hooks/wiki_guard.py`: the
// copy of itself that `brain init` installed into a guarded repository,
// refused because a barrier a writing tool can overwrite is no barrier.
//
// There is no such copy any more. `brain init` writes the command
// `brain guard` and installs no file (src/brain/init.py: "A bare command,
// and no copy of the barrier beside it"), the Python module went in
// 6bceba4, and no `.ultra-brain/hooks` directory exists in any registered
// area on this machine -- checked, not assumed. The rule guarded a path
// that nothing creates, in a way of building that was abandoned.
//
// So the name is ordinary now, in all three of the places the pin used to
// tell apart. What replaced the installed copy is a binary outside every
// writable tree, and the geometry keeps that one.
func TestTheOldGuardNameIsAnOrdinaryFileEverywhere(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	bundle := filepath.Join(tmp, "vault", "demo")
	for _, target := range []string{
		// The whole installed path, which was the pin itself.
		filepath.Join(bundle, ".loomux", "hooks", "wiki_guard.py"),
		// The two the pin already let through, kept so a rule creeping
		// back in a wider shape is caught as well.
		filepath.Join(bundle, "hooks", "wiki_guard.py"),
		filepath.Join(bundle, ".loomux", "wiki_guard.py"),
	} {
		allow(t, writeCall(target), state)
	}
}

func TestTheManifestIsRefusedBeforeTheRegistryIsRead(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	mkdir(t, state)
	// No registry at all: were the manifest check to stand after the
	// registry read, this would come back with the registry's complaint
	// and name the wrong file.
	target := filepath.Join(tmp, "anywhere", ".loomux", "config.toml")
	reason := deny(t, writeCall(target), state,
		"the manifest is where the barrier reads its own limits")
	if strings.Contains(reason, "cannot read the registry") {
		t.Fatalf("the registry answered first: %q", reason)
	}
}

// The manifest keeps its place at the front of the queue even when a call
// names two targets and the manifest is the second of them. The pin that
// used to stand behind it is gone, so the runner-up here is an ordinary
// path outside every writable tree -- the reason the reader would get if
// the manifest check had slipped behind the registry read.
func TestTheManifestIsNamedBeforeAnOrdinaryRefusal(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{
		"tool_name": "Write",
		"tool_input": map[string]any{
			"file_path":     filepath.Join(tmp, "elsewhere", "x.md"),
			"notebook_path": filepath.Join(tmp, "a", ".loomux", "config.toml"),
		},
	}
	deny(t, payload, state,
		"the manifest is where the barrier reads its own limits")
}

// --- the registry that cannot be read -----------------------------------

func TestAnUnreadableRegistryRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	mkdir(t, state)
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"the wiki guard cannot read the registry, so it refuses")
}

func TestABrokenRegistryRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"), "[[area]\nscope =")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"the wiki guard cannot read the registry, so it refuses")
}

func TestAreasDeclaredAsATableRefuse(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[area]\nscope = \"a\"\npath = \"/x\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"areas must be declared as [[area]] tables, not [area]")
}

func TestAnEntryWithoutAPathRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/demo\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"is missing the 'path' key")
}

func TestAnEntryWithoutAScopeRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\npath = \"/x\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"is missing the 'scope' key")
}

func TestAScopeThatIsNotAStringRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = 1\npath = \"/x\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"[[area]] scope must be a non-empty string, found 1")
}

func TestADuplicateScopeRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"a\"\npath = \"/x\"\n\n"+
			"[[area]]\nscope = \"a\"\npath = \"/y\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"duplicate scope 'a'")
}

func TestTwoScopesSharingAStateDirectoryRefuse(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"a/b\"\npath = \"/x\"\n\n"+
			"[[area]]\nscope = \"a-b\"\npath = \"/y\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"share the state directory")
}

func TestAScopeWithoutUsableCharactersRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"///\"\npath = \"/x\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"has no usable characters")
}

func TestTwoSignpostsRefuse(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"a\"\npath = \"/x\"\nsignpost = true\n\n"+
			"[[area]]\nscope = \"b\"\npath = \"/y\"\nsignpost = true\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"both declare signpost; two starting points are none")
}

func TestOneSignpostIsFine(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"),
		"signpost = true")
	allow(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state)
}

func TestAnEntryThatIsNotATableRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"), "area = [1]\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"expected an [[area]] table")
}

func TestAReadonlyStringIsReadAsTrue(t *testing.T) {
	tmp := t.TempDir()
	// `bool(entry.get("readonly", False))` (src/brain/registry.py:76) is
	// truthiness, not a type test: `readonly = "yes"` is a read-only area
	// on the Python side, and a reader that failed the whole file here
	// would answer a different registry.
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"),
		"readonly = \"yes\"")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"the registration calls this area read-only")
}

func TestAZeroWorkspaceIsFalse(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"),
		"workspace = 0")
	deny(t, writeCall(filepath.Join(tmp, "repo", "a.py")), state,
		"lies outside every writable tree")
}

func TestAnEmptyWikiStringIsNoWiki(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"a\"\npath = \"/x\"\nwiki = \"\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

func TestAWikiThatIsNotAStringRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	// `Path(wiki)` on a number raises TypeError, which `decide`'s catch
	// turns into a refusal (src/brain/registry.py:75).
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"a\"\npath = \"/x\"\nwiki = 1\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"the wiki guard cannot read the registry, so it refuses")
}

func TestAnEmptyRegistryFileRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"), "")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

// --- the manifest each registered area carries --------------------------

func TestABrokenManifestOfARegisteredAreaRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"), "scope = [\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"the wiki guard cannot read the registry, so it refuses")
}

func TestAManifestWithoutAScopeRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	// A present `[area]` without a scope; a file with no `[area]` at all
	// declares nothing and is judged like a missing manifest.
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\n\n[layout]\nx = 1\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"[area] scope is required and must be a non-empty string")
}

func TestAnAbsoluteInboxRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\ninbox = \""+
			posix(filepath.Join(tmp, "in"))+"\"\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"[layout] inbox must be relative to the area")
}

func TestARootedInboxWithoutADriveIsNotCalledAbsolute(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("on posix `/in` is absolute and the question is moot")
	}
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	// Measured, not assumed: `WindowsPath("/in").is_absolute()` is False,
	// so `_inbox_of` lets this value stand although joining it would
	// still replace the area's root. `_declared_review` names that gap
	// and uses containment instead
	// (src/brain/maintenance/reconcile.py:258-261); `_inbox_of` does not,
	// and this side is no stricter than the side it mirrors.
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\ninbox = \"/in\"\n")
	allow(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state)
}

func TestARelativeInboxIsFine(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\ninbox = \"00 In\"\n")
	allow(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state)
}

func TestAnEmptyInboxIsNoInbox(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\ninbox = \"\"\n")
	allow(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state)
}

func TestAnInboxThatIsNotAStringRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\ninbox = 1\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"the wiki guard cannot read the registry, so it refuses")
}

func TestABadPrivacyModeRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[privacy]\nmode = \"x\"\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"[privacy] mode must be one of automatic_cloud, local_only, "+
			"manual_cloud, found 'x'")
}

func TestABadUntouchedDaysRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[wiki]\nuntouched_days = 0\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"[wiki] untouched_days must be an integer >= 1")
}

func TestATrueUntouchedDaysRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	// `bool` passes as an `int` in Python, and `untouched_days = true`
	// would otherwise become a threshold of one day
	// (src/brain/manifest.py:138-140).
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n"+
			"[wiki]\nuntouched_days = true\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"[wiki] untouched_days must be an integer >= 1")
}

func TestABadTypesListRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[wiki]\ntypes = \"a\"\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"[wiki] types must be a list of strings")
}

func TestABadOnMergeRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[maintenance]\non_merge = 1\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"[maintenance] on_merge must be a boolean, found 1")
}

func TestABadBranchRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[maintenance]\nbranch = \"\"\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"[maintenance] branch must be a non-empty string")
}

func TestABadGlobListRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[index]\ninclude = [1]\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"[index] include contains a non-string: 1")
}

func TestAGlobThatIsNoListRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[privacy]\nnever = \"x\"\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"[privacy] never must be an array of strings")
}

func TestALayoutThatIsNotATableRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"layout = \"x\"\n\n[area]\nscope = \"project/demo\"\n")
	// `dict("x")` is the plain ValueError `decide`'s docstring names.
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"the wiki guard cannot read the registry, so it refuses")
}

func TestAnAreaTableThatIsNotATableRefusesEverything(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"), "area = \"x\"\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"the wiki guard cannot read the registry, so it refuses")
}

// --- a project configuration that declares no area ----------------------

// policyOnly is a `.loomux/config.toml` of a project that uses loomux's
// policy and no brain area at all.
const policyOnly = "[[policy.paths.rules]]\nmatch = \"bin/*\"\n" +
	"reason = \"built output\"\n"

func TestAWorkspaceWhoseConfigHasOnlyAPolicyIsWritable(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "", "workspace = true")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"), policyOnly)
	allow(t, writeCall(filepath.Join(tmp, "repo", "src", "a.py")), state)
}

func TestAConfigWithoutAnAreaOnTheWalkIsClimbedPast(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"w\"\n")
	// Deeper than the declaration, so the walk meets it first; it must
	// neither refuse the call nor end the walk before the real one.
	write(t, filepath.Join(tmp, "repo", "w", "sub", ".loomux", "config.toml"),
		policyOnly)
	allow(t, writeCall(filepath.Join(tmp, "repo", "w", "sub", "x.md")), state)
}

func TestAConfigWithoutAnAreaDoesNotCloseTheReviewCentre(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	plain := filepath.Join(tmp, "plain")
	owner := filepath.Join(tmp, "owner")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"plain\"\npath = \""+posix(plain)+"\"\n\n"+
			"[[area]]\nscope = \"owner\"\npath = \""+posix(owner)+
			"\"\nwiki = \""+posix(filepath.Join(owner, "w"))+"\"\n")
	write(t, filepath.Join(plain, ".loomux", "config.toml"), policyOnly)
	write(t, filepath.Join(owner, ".loomux", "config.toml"),
		"[area]\nscope = \"owner\"\n\n[layout]\nreview = \"95\"\n")
	allow(t, writeCall(filepath.Join(owner, "95", "c1", "proposal.md")),
		state)
}

func TestAnAreaTableWithoutAScopeStillRefuses(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "", "workspace = true")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		policyOnly+"\n[area]\nscope = \"\"\n")
	deny(t, writeCall(filepath.Join(tmp, "repo", "src", "a.py")), state,
		"[area] scope is required and must be a non-empty string")
}

// --- the tree a manifest declares ---------------------------------------

func TestAManifestDeclaresAThirdWritableTree(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"90 Wiki\"\n")
	allow(t, writeCall(filepath.Join(tmp, "repo", "90 Wiki", "x.md")), state)
}

func TestTheDeclaredTreeDoesNotOpenTheRepository(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"90 Wiki\"\n")
	deny(t, writeCall(filepath.Join(tmp, "repo", "src", "a.py")), state,
		"lies outside every writable tree")
}

func TestAPlantedManifestNamingAForeignScopeDeclaresNothing(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	// The directory is not the registered tree and carries no `.git`, so
	// the third condition of `_declared_wiki_root` fails and the walk
	// goes on.
	write(t, filepath.Join(tmp, "planted", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"w\"\n")
	deny(t, writeCall(filepath.Join(tmp, "planted", "w", "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

func TestAManifestNamingAnUnknownScopeDeclaresNothing(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"other\"\n\n[layout]\nwiki = \"w\"\n")
	deny(t, writeCall(filepath.Join(tmp, "repo", "w", "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

func TestAManifestOfAReadonlyAreaDeclaresNothing(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "", "readonly = true")
	// A read-only area reads its manifest from the state directory
	// (`area_artifact_dir`, src/brain/registry.py:123-129), so the file
	// beside the repository is only the declaration the walk finds.
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"w\"\n")
	deny(t, writeCall(filepath.Join(tmp, "repo", "w", "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

func TestAManifestWithoutALayoutDeclaresNothing(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n")
	deny(t, writeCall(filepath.Join(tmp, "repo", "w", "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

func TestAWikiLayoutReachingOutIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"../out\"\n")
	deny(t, writeCall(filepath.Join(tmp, "repo", "w", "x.md")), state,
		"[layout] wiki must stay inside the repository, found '../out'")
}

func TestAnAbsoluteWikiLayoutIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"/srv/w\"\n")
	deny(t, writeCall(filepath.Join(tmp, "repo", "w", "x.md")), state,
		"[layout] wiki must stay inside the repository, found '/srv/w'")
}

func TestAWikiLayoutWithABackslashIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = 'a\\b'\n")
	deny(t, writeCall(filepath.Join(tmp, "repo", "w", "x.md")), state,
		"[layout] wiki must use forward slashes")
}

func TestAWikiLayoutNamingTheRepositoryRootIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \".\"\n")
	deny(t, writeCall(filepath.Join(tmp, "repo", "w", "x.md")), state,
		"[layout] wiki must not be the repository root")
}

func TestAWikiLayoutThatIsNotAStringIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = 1\n")
	deny(t, writeCall(filepath.Join(tmp, "repo", "w", "x.md")), state,
		"the wiki guard cannot read the registry, so it refuses")
}

func TestAFalseWikiLayoutDeclaresNothing(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	// `if not value: return None` reads truthiness, so a falsy value is
	// the unsaid key and never reaches the tests below it
	// (src/brain/manifest.py:68-69).
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = false\n")
	deny(t, writeCall(filepath.Join(tmp, "repo", "w", "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

func TestTheWalkClimbsPastADirectoryWithoutADeclaration(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"w\"\n")
	deep := filepath.Join(tmp, "repo", "w", "a", "b", "c", "x.md")
	allow(t, writeCall(deep), state)
}

// --- the one exemption --------------------------------------------------

// reviewName is the review centre this vault uses, spelt as the real one
// is -- the umlaut is what makes the envelope's escaping measurable.
const reviewName = "95 Prüfzentrum"

func withReview(t *testing.T, tmp, layout string) string {
	t.Helper()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nreview = "+
			layout+"\n")
	return state
}

func caseFile(tmp, name string) string {
	return filepath.Join(tmp, "repo", reviewName, "knowledge", "c1", name)
}

func TestTheGuardAllowsWritingAProposal(t *testing.T) {
	tmp := t.TempDir()
	state := withReview(t, tmp, "\""+reviewName+"\"")
	allow(t, writeCall(caseFile(tmp, "proposal.md")), state)
}

func TestTheGuardStillRefusesTheCaseFile(t *testing.T) {
	tmp := t.TempDir()
	state := withReview(t, tmp, "\""+reviewName+"\"")
	reason := deny(t, writeCall(caseFile(tmp, "case.toml")), state,
		"lies outside every writable tree")
	if !strings.Contains(reason, "plus proposal.md below") {
		t.Fatalf("the exemption was not named: %q", reason)
	}
}

func TestTheGuardStillRefusesThePackage(t *testing.T) {
	tmp := t.TempDir()
	state := withReview(t, tmp, "\""+reviewName+"\"")
	deny(t, writeCall(caseFile(tmp, "package.md")), state,
		"lies outside every writable tree")
}

func TestAProposalOutsideTheReviewCentreIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := withReview(t, tmp, "\""+reviewName+"\"")
	deny(t, writeCall(filepath.Join(tmp, "elsewhere", "proposal.md")),
		state, "lies outside every writable tree")
}

func TestTheProposalNameIsComparedLetterForLetter(t *testing.T) {
	tmp := t.TempDir()
	state := withReview(t, tmp, "\""+reviewName+"\"")
	// `resolved.name == PROPOSAL` is a plain
	// string comparison, unlike the containment beside it -- so a capital
	// P names a different file even where the disk does not care.
	deny(t, writeCall(caseFile(tmp, "Proposal.md")), state,
		"lies outside every writable tree")
}

func TestAReviewThatIsNotAStringClosesTheExemption(t *testing.T) {
	tmp := t.TempDir()
	state := withReview(t, tmp, "1")
	reason := deny(t, writeCall(caseFile(tmp, "proposal.md")), state,
		"lies outside every writable tree")
	if strings.Contains(reason, "plus proposal.md below") {
		t.Fatalf("the exemption survived a broken review: %q", reason)
	}
}

func TestAReviewReachingOutOfTheAreaClosesTheExemption(t *testing.T) {
	tmp := t.TempDir()
	state := withReview(t, tmp, "\"../outside\"")
	deny(t, writeCall(filepath.Join(tmp, "outside", "proposal.md")),
		state, "lies outside every writable tree")
}

func TestAnAreaWithoutAReviewDeclarationHasNoCentre(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n")
	reason := deny(t, writeCall(filepath.Join(tmp, "x", "proposal.md")),
		state, "lies outside every writable tree")
	if strings.Contains(reason, "plus proposal.md below") {
		t.Fatalf("an exemption appeared from nowhere: %q", reason)
	}
}

func TestWithoutAManifestTheWikiStillOpens(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	allow(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state)
}

func TestAProposalInAReadonlyAreaIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	write(t, filepath.Join(tmp, "vault", "91", ".loomux", "config.toml"),
		"[area]\nscope = \"hub\"\n\n[layout]\nreview = \"95\"\n")
	target := filepath.Join(tmp, "vault", "91", "space", "95", "c1",
		"proposal.md")
	deny(t, writeCall(target), state,
		"the registration calls this area read-only")
}

func TestTwoReviewCentresCloseTheExemption(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	a := filepath.Join(tmp, "a")
	b := filepath.Join(tmp, "b")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"a\"\npath = \""+posix(a)+"\"\nwiki = \""+
			posix(filepath.Join(a, "w"))+"\"\n\n"+
			"[[area]]\nscope = \"b\"\npath = \""+posix(b)+"\"\nwiki = \""+
			posix(filepath.Join(b, "w"))+"\"\n")
	write(t, filepath.Join(a, ".loomux", "config.toml"),
		"[area]\nscope = \"a\"\n\n[layout]\nreview = \"r\"\n")
	write(t, filepath.Join(b, ".loomux", "config.toml"),
		"[area]\nscope = \"b\"\n\n[layout]\nreview = \"r\"\n")
	reason := deny(t, writeCall(filepath.Join(a, "r", "c", "proposal.md")),
		state, "lies outside every writable tree")
	if strings.Contains(reason, "plus proposal.md below") {
		t.Fatalf("two review centres still granted one: %q", reason)
	}
}

// --- what the mutation round asked for ----------------------------------

func TestOneSignpostBesideAnOrdinaryAreaIsFine(t *testing.T) {
	// Both orders, because the two halves of `if signpost && signposted
	// != ""` fail differently: without the left half a signpost followed
	// by any area at all refuses the whole registry, and without the
	// guard on `signposted = scope` an ordinary area followed by the
	// signpost does. The registration on this machine is exactly one
	// signpost among nine areas, so either mutant would shut the vault.
	// Measured against Python: both orders are allowed there.
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	repo := filepath.Join(tmp, "repo")
	wiki := posix(filepath.Join(tmp, "w"))
	first := "[[area]]\nscope = \"a\"\npath = \"" + posix(repo) +
		"\"\nwiki = \"" + wiki + "\"\n"
	second := "[[area]]\nscope = \"b\"\npath = \"" + posix(repo) + "\"\n"
	target := writeCall(filepath.Join(tmp, "w", "x.md"))

	write(t, filepath.Join(state, "registry.toml"),
		first+"signpost = true\n\n"+second)
	allow(t, target, state)

	write(t, filepath.Join(state, "registry.toml"),
		first+"\n"+second+"signpost = true\n")
	allow(t, target, state)
}

func TestAnEmptyScopeOrPathRefuses(t *testing.T) {
	// `_required` asks two questions of one value -- is it a string, and
	// is it non-empty -- and the second is the one a type test alone
	// would drop. Measured: Python answers "must be a non-empty string,
	// found ''" for both keys.
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	for key, body := range map[string]string{
		"scope": "[[area]]\nscope = \"\"\npath = \"/x\"\n",
		"path":  "[[area]]\nscope = \"a\"\npath = \"\"\n",
	} {
		write(t, filepath.Join(state, "registry.toml"), body)
		deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
			"[[area]] "+key+" must be a non-empty string, found ''")
	}
}
