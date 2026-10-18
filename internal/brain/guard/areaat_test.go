package guard

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// switchRegistry writes `body` as the registry below <base>/state.
func switchRegistry(t *testing.T, base, body string) string {
	t.Helper()
	state := filepath.Join(base, "state")
	write(t, filepath.Join(state, "registry.toml"), body)
	return state
}

// areaBlock is one `[[area]]` block rooted at `path`, with whatever flag
// lines the case adds.
func areaBlock(scope, path string, flags ...string) string {
	body := "[[area]]\nscope = \"" + scope + "\"\npath = \"" + posix(path) + "\"\n"
	for _, flag := range flags {
		body += flag + "\n"
	}
	return body + "\n"
}

func wantArea(t *testing.T, root, state, scope string, llmConfig bool) {
	t.Helper()
	gotScope, gotSwitch, err := AreaAt(root, state)
	if err != nil {
		t.Fatalf("AreaAt(%q): %v", root, err)
	}
	if gotScope != scope || gotSwitch != llmConfig {
		t.Fatalf("AreaAt(%q) = %q, %v; want %q, %v",
			root, gotScope, gotSwitch, scope, llmConfig)
	}
}

func TestAreaAtFindsTheAreaWhosePathIsRoot(t *testing.T) {
	base := t.TempDir()
	on := filepath.Join(base, "on")
	off := filepath.Join(base, "off")
	mkdir(t, filepath.Join(on, "sub"))
	mkdir(t, off)
	state := switchRegistry(t, base,
		areaBlock("project/on", on, "llm_config = true")+
			areaBlock("project/off", off))
	wantArea(t, on, state, "project/on", true)
	wantArea(t, off, state, "project/off", false)
	wantArea(t, filepath.Join(on, "sub"), state, "", false)
	wantArea(t, base, state, "", false)
}

func TestAreaAtFollowsALinkedWorktree(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	mkdir(t, filepath.Join(linked, "sub"))
	state := switchRegistry(t, base,
		areaBlock("project/demo", main, "workspace = true", "llm_config = true"))
	wantArea(t, linked, state, "project/demo", true)
	wantArea(t, filepath.Join(linked, "sub"), state, "", false)
}

func TestAreaAtSkipsTheWorktreeOfAnAreaWithoutWorkspace(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	state := switchRegistry(t, base,
		areaBlock("project/demo", main, "llm_config = true"))
	wantArea(t, linked, state, "", false)
}

func TestAreaAtComparesPathsAsTheFileSystemDoes(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only Windows equates paths that differ in case")
	}
	base := t.TempDir()
	area := filepath.Join(base, "loomux")
	mkdir(t, area)
	state := switchRegistry(t, base,
		areaBlock("project/loomux", area, "llm_config = true"))
	upper := filepath.Join(base, "LOOMUX")
	lowerDrive := strings.ToLower(upper[:1]) + upper[1:]
	wantArea(t, lowerDrive, state, "project/loomux", true)
}

func TestAreaAtRefusesAnUnreadableRegistry(t *testing.T) {
	base := t.TempDir()
	state := switchRegistry(t, base, "[[area]\nnot toml")
	if _, _, err := AreaAt(base, state); err == nil {
		t.Fatal("a broken registry answered without an error")
	}
}

func TestAreaAtRefusesARootItCannotResolve(t *testing.T) {
	base := t.TempDir()
	state := switchRegistry(t, base, areaBlock("project/demo", base))
	withoutWorkingDirectory(t)
	if _, _, err := AreaAt("relative", state); err == nil {
		t.Fatal("an unresolvable root answered without an error")
	}
}

// The manifest at a root is reached directly, whether it exists yet or not;
// through a linked .loomux it is another file, and a root that does not
// resolve answers no.
func TestManifestUnlinked(t *testing.T) {
	root := t.TempDir()
	if !ManifestUnlinked(root) {
		t.Error("a root without .loomux: linked")
	}
	write(t, filepath.Join(root, bundleDir, manifestName), "")
	if !ManifestUnlinked(root) {
		t.Error("a plain manifest: linked")
	}
	linked := t.TempDir()
	other := t.TempDir()
	write(t, filepath.Join(other, bundleDir, manifestName), "")
	linkDir(t, filepath.Join(other, bundleDir), filepath.Join(linked, bundleDir))
	if ManifestUnlinked(linked) {
		t.Error("a manifest through a linked .loomux: unlinked")
	}
	// A hard link is the other project's manifest under this root's name:
	// `config set --yes` writes it in place.
	hard := t.TempDir()
	hardLink(t, filepath.Join(other, bundleDir, manifestName), filepath.Join(hard, bundleDir, manifestName))
	if ManifestUnlinked(hard) {
		t.Error("a hard-linked manifest: unlinked")
	}
	withoutWorkingDirectory(t)
	if ManifestUnlinked("relative") {
		t.Error("an unresolvable root: unlinked")
	}
}

// The own manifest is the one directly below the resolved root, named by a
// full path; a relative path, another root's manifest, one reached through
// a link and a hook without a root are not.
func TestOwnManifest(t *testing.T) {
	root := t.TempDir()
	if !OwnManifest(filepath.Join(root, bundleDir, manifestName), root) {
		t.Error("the root's manifest: not its own")
	}
	if runtime.GOOS == "windows" && !OwnManifest(filepath.Join(root, ".LOOMUX", "Config.toml"), root) {
		t.Error("the root's manifest in another case: not its own")
	}
	other := t.TempDir()
	write(t, filepath.Join(other, bundleDir, manifestName), "")
	linkDir(t, filepath.Join(other, bundleDir), filepath.Join(root, "lnk"))
	for _, path := range []string{
		filepath.Join(bundleDir, manifestName),
		filepath.Join(other, bundleDir, manifestName),
		filepath.Join(root, "lnk", manifestName),
		filepath.Join(root, "sub", bundleDir, manifestName),
	} {
		if OwnManifest(path, root) {
			t.Errorf("%s: the root's own", path)
		}
	}
	if OwnManifest(filepath.Join(root, bundleDir, manifestName), "") {
		t.Error("no root: an own manifest")
	}
	withoutWorkingDirectory(t)
	if OwnManifest(filepath.Join(root, bundleDir, manifestName), "relative") {
		t.Error("an unresolvable root: an own manifest")
	}
}

// The manifest's base is the root or a linked worktree of the root's
// repository; another repository's worktree, another file in the worktree,
// a relative path and a root outside any repository have none.
func TestManifestBase(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	foreign := fakeLinked(t, fakeRepository(t, base, "other"), "foreign", false)
	for _, c := range []struct{ path, root, want string }{
		{manifestOf(main), main, main},
		{manifestOf(linked), main, linked},
		{manifestOf(foreign), main, ""},
		{filepath.Join(linked, "sub", manifestName), main, ""},
		{filepath.Join(bundleDir, manifestName), main, ""},
		{manifestOf(linked), t.TempDir(), ""},
		{manifestOf(linked), "", ""},
	} {
		if got := ManifestBase(c.path, c.root); got != c.want {
			t.Errorf("ManifestBase(%q, %q) = %q, want %q", c.path, c.root, got, c.want)
		}
	}
}

// A directory is at the root when both resolve to the same path; an empty
// or relative directory, one below the root and one that does not resolve
// are not.
func TestAtRoot(t *testing.T) {
	root := t.TempDir()
	if !AtRoot(root, root) || !AtRoot(root+string(filepath.Separator), root) {
		t.Error("the root itself: not at the root")
	}
	if runtime.GOOS == "windows" && !AtRoot(strings.ToUpper(root), root) {
		t.Error("the root in another case: not at the root")
	}
	sub := filepath.Join(root, "sub")
	mkdir(t, sub)
	for _, dir := range []string{"", ".", "sub", sub, t.TempDir()} {
		if AtRoot(dir, root) {
			t.Errorf("%q: at the root", dir)
		}
	}
	if AtRoot(root, "") {
		t.Error("no root: at the root")
	}
	withoutWorkingDirectory(t)
	if AtRoot(root, "relative") {
		t.Error("an unresolvable root: at the root")
	}
}

func TestAreaAtRefusesAnAreaItCannotResolve(t *testing.T) {
	base := t.TempDir()
	state := switchRegistry(t, base, areaBlock("project/demo", "relative"))
	withoutWorkingDirectory(t)
	if _, _, err := AreaAt(base, state); err == nil {
		t.Fatal("an unresolvable area answered without an error")
	}
}
