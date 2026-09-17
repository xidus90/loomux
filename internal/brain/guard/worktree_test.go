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
	// An administration directory that lost its `commondir`: the back
	// pointer still agrees, so this is the one check left to refuse it.
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

func TestANestedRepositorysWorktreeCannotBeRetargetedAtTheWorkspace(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	nested := fakeRepository(t, main, filepath.Join("vendor", "other"))
	outside := filepath.Join(base, "outside")
	admin := filepath.Join(nested, ".git", "worktrees", "outside")
	write(t, filepath.Join(outside, ".git"), "gitdir: "+posix(admin)+"\n")
	write(t, filepath.Join(admin, "gitdir"),
		posix(filepath.Join(outside, ".git"))+"\n")
	// The administration directory lies inside the workspace, so a writing
	// tool may point its `commondir` at the workspace's own repository.
	write(t, filepath.Join(admin, "commondir"),
		posix(filepath.Join(main, ".git"))+"\n")
	state := workspaceRegistry(t, base, main)
	deny(t, writeCall(filepath.Join(outside, "a.go")), state,
		"lies outside every writable tree")
}

func TestASelfMadeAdministrationDirectoryOpensNothing(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	planted := filepath.Join(base, "planted")
	write(t, filepath.Join(planted, ".git"), "gitdir: .\n")
	write(t, filepath.Join(planted, "gitdir"),
		posix(filepath.Join(planted, ".git"))+"\n")
	write(t, filepath.Join(planted, "commondir"),
		posix(filepath.Join(main, ".git"))+"\n")
	state := workspaceRegistry(t, base, main)
	deny(t, writeCall(filepath.Join(planted, "a.go")), state,
		"lies outside every writable tree")
}

func TestASymlinkedGitFileIsNoPointer(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	other := filepath.Join(base, "other")
	mkdir(t, other)
	if err := os.Symlink(filepath.Join(linked, ".git"),
		filepath.Join(other, ".git")); err != nil {
		t.Skip("this machine does not let the test make a file symlink")
	}
	state := workspaceRegistry(t, base, main)
	deny(t, writeCall(filepath.Join(other, "a.go")), state,
		"lies outside every writable tree")
}

func TestARegisteredRootIsItsOwnRepository(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	want := mustResolve(t, filepath.Join(main, ".git"))
	if got := registeredCommon(main); got != want {
		t.Fatalf("registeredCommon(root) = %q, want %q", got, want)
	}
}

func TestARegisteredSubdirectoryClimbsToItsRepository(t *testing.T) {
	// `git rev-parse --git-common-dir` answers from any directory inside a
	// checkout, and a registered area is not always a checkout root.
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	area := filepath.Join(main, "vault", "demo")
	mkdir(t, area)
	want := mustResolve(t, filepath.Join(main, ".git"))
	if got := registeredCommon(area); got != want {
		t.Fatalf("registeredCommon(subdirectory) = %q, want %q", got, want)
	}
}

func TestAMissingRegisteredPathHasNoRepository(t *testing.T) {
	// git cannot start in a directory that is not there; a climb from one
	// would borrow the repository of whatever ancestor still stands.
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	if got := registeredCommon(filepath.Join(main, "gone")); got != "" {
		t.Fatalf("registeredCommon(missing) = %q, want the empty answer", got)
	}
}

func TestARegisteredFileHasNoRepository(t *testing.T) {
	// git cannot start in a file either; a climb from one would borrow the
	// repository of the directory that holds it.
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	file := filepath.Join(main, "notes.md")
	write(t, file, "not a directory\n")
	if got := registeredCommon(file); got != "" {
		t.Fatalf("registeredCommon(file) = %q, want the empty answer", got)
	}
}

func TestARegisteredPathOutsideEveryRepositoryHasNone(t *testing.T) {
	// Assumes no directory above the test's temporary directory carries a
	// `.git`; on 2026-09-16 none from C:\ to %TEMP% did.
	base := t.TempDir()
	plain := filepath.Join(base, "plain")
	mkdir(t, plain)
	if got := registeredCommon(plain); got != "" {
		t.Fatalf("registeredCommon(plain) = %q, want the empty answer", got)
	}
}

func TestARegisteredLinkedWorktreeNamesItsCommonDirectory(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	want := mustResolve(t, filepath.Join(main, ".git"))
	if got := registeredCommon(linked); got != want {
		t.Fatalf("registeredCommon(linked) = %q, want %q", got, want)
	}
}

func TestAnAreaInASubmoduleDoesNotClimbIntoTheSuperproject(t *testing.T) {
	// A submodule's `.git` file points at `<super>/.git/modules/<name>`,
	// which holds neither `gitdir` nor `commondir`. The climb has to stop
	// there: git names the submodule's own directory, and borrowing the
	// superproject's would make its worktrees one repository with an area
	// git keeps apart.
	base := t.TempDir()
	super := fakeRepository(t, base, "super")
	module := filepath.Join(super, "mod")
	mkdir(t, filepath.Join(super, ".git", "modules", "mod"))
	write(t, filepath.Join(module, ".git"), "gitdir: ../.git/modules/mod\n")
	area := filepath.Join(module, "vault")
	mkdir(t, area)
	if got := registeredCommon(area); got != "" {
		t.Fatalf("registeredCommon(in submodule) = %q, want the empty answer", got)
	}
}

func TestAnAreaInABareRepositoryClimbsToTheCheckoutAboveIt(t *testing.T) {
	// A deviation that opens, approved in the parity list of the barrier's
	// worktrees: git stops in a bare repository and names it, so Python
	// called no checkout the same repository as an area inside one. The
	// climb looks only for `.git` entries, walks past the bare repository
	// and lands on the checkout that holds it -- and a worktree of that
	// checkout becomes the area's repository.
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	bare := filepath.Join(main, "mirror.git")
	mkdir(t, filepath.Join(bare, "objects"))
	mkdir(t, filepath.Join(bare, "refs"))
	write(t, filepath.Join(bare, "HEAD"), "ref: refs/heads/master\n")
	area := filepath.Join(bare, "vault")
	mkdir(t, area)
	want := mustResolve(t, filepath.Join(main, ".git"))
	if got := registeredCommon(area); got != want {
		t.Fatalf("registeredCommon(in bare repository) = %q, want %q", got, want)
	}
	linked := fakeLinked(t, main, "linked", false)
	if !sameRepository(linked, area) {
		t.Fatal("a worktree of the checkout above a bare repository is not the same repository as an area inside it")
	}
}

func TestAWorktreeIsTheSameRepositoryAsItsRegisteredTree(t *testing.T) {
	// Laid out by hand, so git itself would not recognise either side:
	// only a reading of the files can say yes here.
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	area := filepath.Join(main, "vault", "demo")
	mkdir(t, area)
	if !sameRepository(linked, main) {
		t.Error("a linked worktree is not the same repository as its main checkout")
	}
	if !sameRepository(linked, area) {
		t.Error("a linked worktree is not the same repository as an area inside its main checkout")
	}
}

func TestAnotherCheckoutIsNotTheSameRepository(t *testing.T) {
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	other := fakeRepository(t, base, "other")
	if sameRepository(other, main) {
		t.Fatal("two unrelated checkouts compared as one repository")
	}
}

func TestAWorktreeIsNoRepositoryOfAnAreaOutsideEveryRepository(t *testing.T) {
	// The empty answer of registeredCommon against a real common directory.
	// Two empty answers compare equal, which the emptiness test in
	// `sameRepository` catches; here only the registered side is empty,
	// and `pathsEqual` has to keep the two apart.
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	plain := filepath.Join(base, "plain")
	mkdir(t, plain)
	if sameRepository(linked, plain) {
		t.Fatal("a worktree matched an area that lies in no repository")
	}
}

func TestAWriteInTheRegisteredCheckoutUnderAManifestReadsNoGitFile(t *testing.T) {
	// The manifest makes `declaredWikiRoot` ask `sameRepository` about the
	// registered directory itself, which has to answer before any pointer
	// file is read. The registered checkout is a linked worktree, whose
	// `.git` is a file: past the equal-path answer, `repositoryCommon` would
	// read it. The target lies inside the registered tree, so the worktree
	// lookup for targets outside every tree reads nothing either.
	old := readGitFile
	t.Cleanup(func() { readGitFile = old })
	readGitFile = func(name string) ([]byte, error) {
		t.Errorf("read %s for a write in the registered checkout", name)
		return old(name)
	}
	base := t.TempDir()
	main := fakeRepository(t, base, "main")
	linked := fakeLinked(t, main, "linked", false)
	state := workspaceRegistry(t, base, linked)
	write(t, filepath.Join(linked, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n")
	allow(t, writeCall(filepath.Join(linked, "src", "a.go")), state)
}

func TestACommondirThatLeadsInACircleIsNoCommonDirectory(t *testing.T) {
	tmp := t.TempDir()
	circle := filepath.Join(tmp, "circle")
	cycleOfTwo(t, circle, filepath.Join(tmp, "other"))
	main := fakeRepository(t, tmp, "main")
	linked := fakeLinked(t, main, "linked", false)
	write(t, filepath.Join(adminOf(main, "linked"), "commondir"),
		posix(circle)+"\n")
	// "" is what every other failed reading says, and the caller reads it
	// as "not the same repository".
	if got := repositoryCommon(linked); got != "" {
		t.Errorf("repositoryCommon = %q, want the empty answer", got)
	}
	if sameRepository(linked, main) {
		t.Error("a worktree whose commondir leads in a circle matched")
	}
}
