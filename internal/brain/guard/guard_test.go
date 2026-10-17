package guard

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteTargetsCollectsEveryKnownKey(t *testing.T) {
	got := WriteTargets(map[string]any{"file_path": "a", "notebook_path": "b", "TargetFile": "c", "target_file": "d", "other": "e", "file_path_empty": ""})
	if strings.Join(got, ",") != "a,b,c,d" {
		t.Fatalf("%v", got)
	}
}

func TestIsWritingTool(t *testing.T) {
	if !IsWritingTool("Edit") || !IsWritingTool("write_to_file") || IsWritingTool("Bash") {
		t.Fatal("wrong classification")
	}
}

func TestCallReadsBothPayloadShapes(t *testing.T) {
	name, args := Call(map[string]any{"tool_name": "Write", "tool_input": map[string]any{"file_path": "a"}})
	if name != "Write" || args["file_path"] != "a" {
		t.Fatalf("host shape: %q %v", name, args)
	}
	name, args = Call(map[string]any{"toolCall": map[string]any{"name": "write_to_file", "args": map[string]any{"TargetFile": "b"}}})
	if name != "write_to_file" || args["TargetFile"] != "b" {
		t.Fatalf("antigravity shape: %q %v", name, args)
	}
}

const manifestRefusal = "the manifest is where the barrier reads its own limits"

func manifestOf(root string) string {
	return filepath.Join(root, ".loomux", "config.toml")
}

func TestDecideOpensTheOwnManifestWhenTheSwitchIsOn(t *testing.T) {
	base := t.TempDir()
	area := filepath.Join(base, "area")
	mkdir(t, area)
	state := switchRegistry(t, base,
		areaBlock("project/demo", area, "workspace = true", "llm_config = true"))
	allowAt(t, writeCall(manifestOf(area)), state, area)
	if runtime.GOOS == "windows" {
		allowAt(t, writeCall(filepath.Join(area, ".LOOMUX", "Config.toml")), state, area)
	}
}

func TestDecideKeepsEveryOtherManifestShut(t *testing.T) {
	base := t.TempDir()
	on := filepath.Join(base, "on")
	off := filepath.Join(base, "off")
	mkdir(t, on)
	mkdir(t, off)
	state := switchRegistry(t, base,
		areaBlock("project/on", on, "workspace = true", "llm_config = true")+
			areaBlock("project/off", off, "workspace = true"))
	for _, target := range []string{
		manifestOf(off),
		manifestOf(filepath.Join(on, "sub")),
		manifestOf(filepath.Join(base, "elsewhere")),
		// The manifest by another spelling: the switch names that spelling
		// up to case, and every other one stays shut rather than argued.
		filepath.Join(on, ".loomux ", "config.toml"),
		manifestOf(on) + "::$DATA",
	} {
		denyAt(t, writeCall(target), state, on, manifestRefusal)
	}
	both := map[string]any{
		"tool_name": "Write",
		"tool_input": map[string]any{
			"file_path":     manifestOf(on),
			"notebook_path": manifestOf(off),
		},
	}
	reason := denyAt(t, both, state, on, manifestRefusal)
	if !strings.HasPrefix(reason, manifestOf(off)+": ") {
		t.Fatalf("the refusal should name only the shut manifest: %q", reason)
	}
}

// A switch opens the manifest of the hook's own root and no other area's,
// however the path reaches it: named outright, or through a link inside
// the hook's project.
func TestDecideOpensOnlyTheHookRootsManifest(t *testing.T) {
	base := t.TempDir()
	on := filepath.Join(base, "on")
	off := filepath.Join(base, "off")
	mkdir(t, filepath.Join(on, bundleDir))
	mkdir(t, off)
	state := switchRegistry(t, base,
		areaBlock("project/on", on, "workspace = true", "llm_config = true")+
			areaBlock("project/off", off, "workspace = true"))
	denyAt(t, writeCall(manifestOf(on)), state, off, manifestRefusal)
	denyAt(t, writeCall(manifestOf(on)), state, "", manifestRefusal)
	linkDir(t, filepath.Join(on, bundleDir), filepath.Join(off, "lnk"))
	denyAt(t, writeCall(filepath.Join(off, "lnk", manifestName)), state, off, manifestRefusal)
}

func TestDecideLeavesAnOpenManifestToTheAllowList(t *testing.T) {
	base := t.TempDir()
	area := filepath.Join(base, "area")
	mkdir(t, area)
	state := switchRegistry(t, base,
		areaBlock("project/demo", area, "llm_config = true")+
			areaBlock("project/other", filepath.Join(base, "other"), "workspace = true"))
	denyAt(t, writeCall(manifestOf(area)), state, area,
		"lies outside every writable tree")
}

func TestDecideKeepsTheManifestShutWhenTheRegistryBreaks(t *testing.T) {
	base := t.TempDir()
	area := filepath.Join(base, "area")
	mkdir(t, area)
	state := switchRegistry(t, base,
		areaBlock("project/demo", area, "workspace = true", "llm_config = true"))
	allowAt(t, writeCall(manifestOf(area)), state, area)
	switchRegistry(t, base, "[[area]\nnot toml")
	denyAt(t, writeCall(manifestOf(area)), state, area, manifestRefusal)
}

func TestDecideOpensTheManifestOfALinkedWorktree(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	state := switchRegistry(t, base,
		areaBlock("project/demo", main, "workspace = true", "llm_config = true"))
	allowAt(t, writeCall(manifestOf(linked)), state, linked)
}
