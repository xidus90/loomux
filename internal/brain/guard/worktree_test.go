package guard

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRepository lays out the one part of a main checkout this reading
// looks at: a `.git` directory.
func fakeRepository(t *testing.T, base, name string) string {
	t.Helper()
	main := filepath.Join(base, name)
	mkdir(t, filepath.Join(main, ".git"))
	return main
}

// fakeLinked lays out by hand what `git worktree add` leaves on disk for a
// linked worktree `name` beside `main`, as Git 2.54 wrote it on 2026-09-15:
// a `.git` file naming the administration directory, and there `commondir`
// and a `gitdir` pointing back. `relative` spells all three pointers the
// way `worktree.useRelativePaths` does.
func fakeLinked(t *testing.T, main, name string, relative bool) string {
	t.Helper()
	linked := filepath.Join(filepath.Dir(main), name)
	admin := filepath.Join(main, ".git", "worktrees", name)
	gitdir := admin
	commondir := filepath.Join(main, ".git")
	back := filepath.Join(linked, ".git")
	if relative {
		gitdir = filepath.Join("..", filepath.Base(main), ".git", "worktrees", name)
		commondir = filepath.Join("..", "..")
		back = filepath.Join("..", "..", "..", "..", name, ".git")
	}
	write(t, filepath.Join(linked, ".git"), "gitdir: "+posix(gitdir)+"\n")
	write(t, filepath.Join(admin, "commondir"), posix(commondir)+"\n")
	write(t, filepath.Join(admin, "gitdir"), posix(back)+"\n")
	return linked
}

// adminOf is the administration directory fakeLinked wrote for `name`.
func adminOf(main, name string) string {
	return filepath.Join(main, ".git", "worktrees", name)
}

// workspaceRegistry registers `path` as the workspace `project/demo`, with
// whatever further `[[area]]` blocks the case adds.
func workspaceRegistry(t *testing.T, base, path string, more ...string) string {
	t.Helper()
	state := filepath.Join(base, "state")
	body := "[[area]]\nscope = \"project/demo\"\npath = \"" + posix(path) +
		"\"\nworkspace = true\n"
	for _, area := range more {
		body += "\n" + area
	}
	write(t, filepath.Join(state, "registry.toml"), body)
	return state
}

func TestALinkedWorktreeOfAWorkspaceMayBeWritten(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	state := workspaceRegistry(t, base, main)
	allow(t, writeCall(filepath.Join(linked, "src", "a.go")), state)
}

func TestRelativePointersAreFollowedAsGitWritesThem(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", true)
	state := workspaceRegistry(t, base, main)
	allow(t, writeCall(filepath.Join(linked, "src", "a.go")), state)
}

func TestAWorktreeOfAnAreaWithoutWorkspaceStaysShut(t *testing.T) {
	tmp := t.TempDir()
	main := fakeRepository(t, tmp, "repo")
	linked := fakeLinked(t, main, "linked", false)
	// registryOf names <tmp>/repo with a wiki and no `workspace`.
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	deny(t, writeCall(filepath.Join(linked, "src", "a.go")), state,
		"lies outside every writable tree")
}

func TestAWorktreeOfAnUnregisteredRepositoryStaysShut(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	other := fakeRepository(t, base, "other")
	foreign := fakeLinked(t, other, "foreign", false)
	state := workspaceRegistry(t, base, main)
	deny(t, writeCall(filepath.Join(foreign, "a.go")), state,
		"lies outside every writable tree")
}

func TestAWorkspaceWithoutGitOpensNoWorktree(t *testing.T) {
	base := t.TempDir()
	plain := filepath.Join(base, "plain")
	mkdir(t, plain)
	other := fakeRepository(t, base, "other")
	foreign := fakeLinked(t, other, "foreign", false)
	state := workspaceRegistry(t, base, plain)
	deny(t, writeCall(filepath.Join(foreign, "a.go")), state,
		"lies outside every writable tree")
}

func TestABorrowedGitFileOpensNothing(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	borrowed, err := os.ReadFile(filepath.Join(linked, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	planted := filepath.Join(base, "planted")
	write(t, filepath.Join(planted, ".git"), string(borrowed))
	state := workspaceRegistry(t, base, main)
	// The pointer is genuine and so is the administration directory; only
	// its `gitdir` leads to the real worktree and not to the copy.
	deny(t, writeCall(filepath.Join(planted, "a.go")), state,
		"lies outside every writable tree")
	allow(t, writeCall(filepath.Join(linked, "a.go")), state)
}

func TestAMissingBackPointerOpensNothing(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	if err := os.Remove(filepath.Join(adminOf(main, "linked"), "gitdir")); err != nil {
		t.Fatal(err)
	}
	state := workspaceRegistry(t, base, main)
	deny(t, writeCall(filepath.Join(linked, "a.go")), state,
		"lies outside every writable tree")
}

func TestWithoutCommondirTheGitFileIsNoLinkedWorktree(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	// The shape a submodule has: a `.git` file and an administration
	// directory, but no `commondir` in it.
	if err := os.Remove(filepath.Join(adminOf(main, "linked"), "commondir")); err != nil {
		t.Fatal(err)
	}
	state := workspaceRegistry(t, base, main)
	deny(t, writeCall(filepath.Join(linked, "a.go")), state,
		"lies outside every writable tree")
}

func TestAGitFileWithoutThePrefixIsNoPointer(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	write(t, filepath.Join(linked, ".git"), posix(adminOf(main, "linked"))+"\n")
	state := workspaceRegistry(t, base, main)
	deny(t, writeCall(filepath.Join(linked, "a.go")), state,
		"lies outside every writable tree")
}

func TestARegisteredLinkedWorktreeOpensItsSiblings(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	first := fakeLinked(t, main, "first", false)
	second := fakeLinked(t, main, "second", false)
	state := workspaceRegistry(t, base, first)
	allow(t, writeCall(filepath.Join(second, "a.go")), state)
}

func TestTheSearchClimbsPastANestedRepository(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	nested := filepath.Join(linked, "vendor", "nested")
	mkdir(t, filepath.Join(nested, ".git"))
	state := workspaceRegistry(t, base, main)
	allow(t, writeCall(filepath.Join(nested, "x.go")), state)
}

func TestAReadonlyZoneInsideAWorktreeOutranksIt(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	zoneRepo := filepath.Join(base, "zone")
	mkdir(t, zoneRepo)
	wiki := filepath.Join(linked, "docs", "wiki")
	state := workspaceRegistry(t, base, main,
		"[[area]]\nscope = \"project/zone\"\npath = \""+posix(zoneRepo)+
			"\"\nwiki = \""+posix(wiki)+"\"\nreadonly = true\n")
	deny(t, writeCall(filepath.Join(wiki, "x.md")), state,
		"the registration calls this area read-only")
}

func TestAMixedCallNamesTheWorktreeAmongWhatIsAllowed(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	state := workspaceRegistry(t, base, main)
	outsidePath := filepath.Join(base, "outside", "n.ipynb")
	payload := map[string]any{
		"tool_name": "Write",
		"tool_input": map[string]any{
			"file_path":     filepath.Join(linked, "a.go"),
			"notebook_path": outsidePath,
		},
	}
	reason := deny(t, payload, state, "lies outside every writable tree")
	if !strings.HasPrefix(reason, mustResolve(t, outsidePath)+" lies outside") {
		t.Fatalf("the refusal names more than the outside target: %q", reason)
	}
	if !strings.Contains(reason, mustResolve(t, linked)) {
		t.Fatalf("the worktree root is not among the allowed trees: %q", reason)
	}
}

func TestAWriteInsideARegisteredTreeReadsNoGitFile(t *testing.T) {
	old := readGitFile
	t.Cleanup(func() { readGitFile = old })
	readGitFile = func(name string) ([]byte, error) {
		t.Errorf("read %s for a write the registry already opens", name)
		return old(name)
	}
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	state := workspaceRegistry(t, base, main)
	allow(t, writeCall(filepath.Join(main, "src", "a.go")), state)
}

func TestAPointerThatCannotBeResolvedIsNoPointer(t *testing.T) {
	tmp := t.TempDir()
	first := filepath.Join(tmp, "a")
	cycleOfTwo(t, first, filepath.Join(tmp, "b"))
	if got := resolvedOrEmpty(filepath.Join(first, "x")); got != "" {
		t.Fatalf("a path through a cycle resolved to %q", got)
	}
}

func TestARealLinkedWorktreeOfAWorkspaceMayBeWritten(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine")
	}
	base := t.TempDir()
	main := filepath.Join(base, "main")
	mkdir(t, main)
	run(t, main, "init")
	run(t, main, "commit", "--allow-empty", "-m", "first")
	absolute := filepath.Join(base, "absolute")
	run(t, main, "worktree", "add", "-b", "absolute", absolute)
	relative := filepath.Join(base, "relative")
	run(t, main, "-c", "worktree.useRelativePaths=true",
		"worktree", "add", "-b", "relative", relative)
	other := filepath.Join(base, "other")
	mkdir(t, other)
	run(t, other, "init")

	state := workspaceRegistry(t, base, main)
	allow(t, writeCall(filepath.Join(absolute, "internal", "probe.go")), state)
	allow(t, writeCall(filepath.Join(relative, "internal", "probe.go")), state)
	deny(t, writeCall(filepath.Join(other, "a.go")), state,
		"lies outside every writable tree")
}
