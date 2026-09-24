package guard

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/testlock"
)

// --- the renderings Python owns -----------------------------------------

// TestPythonJSONStringRendersWhatJSONDumpsRenders holds the hand-written
// escaper against `json.dumps`. Every expectation was measured by running
// it, not derived from the documentation:
//
//	python -c "import json; print(json.dumps(...))"
//
// The three that matter are the ones `encoding/json` would get wrong:
// `<&>` stays bare where Go escapes it, `ü` becomes `\u00fc` where Go
// emits the byte, and a character outside the basic plane becomes the
// surrogate pair Go would write as four bytes of UTF-8.
func TestPythonJSONStringRendersWhatJSONDumpsRenders(t *testing.T) {
	for _, row := range []struct{ value, want string }{
		{"plain", `"plain"`},
		{`a"b`, `"a\"b"`},
		{`a\b`, `"a\\b"`},
		{"\b\f\n\r\t", `"\b\f\n\r\t"`},
		{"\x01\x1f\x7f", `"\u0001\u001f\u007f"`},
		{"\u00fc", `"\u00fc"`},
		{"\u2028", `"\u2028"`},
		{"\U0001F600", `"\ud83d\ude00"`},
		{"<&>", `"<&>"`},
		{"\u00df Pr\u00fcfzentrum", `"\u00df Pr\u00fcfzentrum"`},
	} {
		if got := pythonJSONString(row.value); got != row.want {
			t.Errorf("pythonJSONString(%q) = %s, want %s",
				row.value, got, row.want)
		}
	}
}

func TestPythonTypeNameNamesWhatPythonNames(t *testing.T) {
	// The two number arms are covered through `Run` in run_test.go; this
	// holds the fallback, which no payload can reach because every value
	// `encoding/json` produces is one of the six above it.
	if got := pythonTypeName(struct{}{}); got != "struct {}" {
		t.Errorf("pythonTypeName(struct{}{}) = %q", got)
	}
}

func TestUTF16PairSplitsWhereJSONDumpsSplits(t *testing.T) {
	high, low := utf16Pair('\U0001F600')
	if high != 0xd83d || low != 0xde00 {
		t.Errorf("utf16Pair = %04x %04x, want d83d de00", high, low)
	}
}

// --- the path arithmetic ------------------------------------------------

func TestARelativeTargetIsResolvedAgainstTheWorkingDirectory(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	// `Path("x.md").resolve()` reaches for the working directory, and a
	// host may well hand the barrier a relative path.
	t.Chdir(filepath.Join(tmp, "repo"))
	deny(t, writeCall("a.py"), state, "lies outside every writable tree")
}

func TestResolveKeepsAPathItCannotReachAtAll(t *testing.T) {
	// Nothing on the way down resolves, so the cleaned path comes back
	// unchanged -- which is the ordinary case for a write, since the
	// file it names is not there yet.
	absent := filepath.Join(absentVolume(t), "no", "such", "x.md")
	got, err := ResolvePath(absent)
	if err != nil {
		t.Fatalf("ResolvePath(%q): %v", absent, err)
	}
	if got != filepath.Clean(absent) {
		t.Errorf("ResolvePath(%q) = %q", absent, got)
	}
}

func TestUnmappedDriveAnswersOnlyALetterTheMaskLeavesFree(t *testing.T) {
	// A mask with exactly one letter free has exactly one right answer,
	// and the two ends of the alphabet are where a loop bound slips.
	const all = uint32(1<<26 - 1)
	for _, free := range []int{0, 16, 25} {
		want := string(rune('A'+free)) + `:\`
		got, ok := unmappedDrive(all &^ (1 << free))
		if !ok || got != want {
			t.Errorf("unmappedDrive(all but %s) = %q, %v", want, got, ok)
		}
	}
	// With every letter taken there is no absent volume to name, and any
	// guess would put the test back on one that exists.
	if got, ok := unmappedDrive(all); ok {
		t.Errorf("unmappedDrive(all) = %q, want none", got)
	}
}

// unmappedDrive is the root of the first letter `assigned` leaves free.
// The mask is `GetLogicalDrives`' own, bit 0 for A:, and it is taken as a
// parameter so that the choice can be asked about every machine's mask
// rather than only this one's.
func unmappedDrive(assigned uint32) (string, bool) {
	for bit := range 26 {
		if assigned&(1<<bit) == 0 {
			return string(rune('A'+bit)) + `:\`, true
		}
	}
	return "", false
}

// strangeVolume is an anchor for path arithmetic that never reaches the
// disk, so any spelling of a volume serves and whether it is mapped
// decides nothing. The one test that opens its path takes `absentVolume`.
func strangeVolume() string {
	if filepath.Separator == '\\' {
		return `Q:\`
	}
	return "/proc/self/no-such-root"
}

func TestSpellingClosesTheThreeSpellingsItWasMeasuredOn(t *testing.T) {
	for _, row := range []struct{ name, want string }{
		{"CONFIG.TOML", "config.toml"},
		{"config.toml.", "config.toml"},
		{"config.toml ", "config.toml"},
		{"config.toml::$DATA", "config.toml"},
		{"C:\\", "c"},
		{"plain", "plain"},
	} {
		if got := spelling(row.name); got != row.want {
			t.Errorf("spelling(%q) = %q, want %q", row.name, got, row.want)
		}
	}
}

func TestContainmentIsMeasuredOnComponentsAndNotOnCharacters(t *testing.T) {
	root := filepath.Join(strangeVolume(), "a", "b")
	if !inside(filepath.Join(root, "x.md"), []string{root}) {
		t.Error("a path in its own tree was called outside")
	}
	if inside(filepath.Join(strangeVolume(), "a", "b-neu", "x.md"),
		[]string{root}) {
		t.Error("a neighbour that only looks like a prefix got in")
	}
	if inside(filepath.Join(strangeVolume(), "a"), []string{root}) {
		t.Error("the parent of a root was called inside it")
	}
}

func TestTheAnchorIsPartOfTheContainment(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("one anchor per path on posix, so there is nothing to mix")
	}
	// Were the anchor dropped, `D:\a\x.md` would match a root of
	// `C:\a` -- the drive is the first component, not decoration.
	if inside(`D:\a\x.md`, []string{`C:\a`}) {
		t.Error("a path on another volume was called inside")
	}
	if !inside(`C:\a\x.md`, []string{`c:\A`}) {
		t.Error("the same tree spelt differently was called outside")
	}
}

func TestOnAPosixFileSystemTheCaseIsPartOfTheName(t *testing.T) {
	// `posixpath.normcase` does nothing, so `Path("/A") == Path("/a")` is
	// False there. This build folds; the other branch is driven here so
	// that neither answer rests on the machine the tests happen to run
	// on.
	old := pathsFold
	t.Cleanup(func() { pathsFold = old })

	upper := filepath.Join(strangeVolume(), "A", "x.md")
	lower := filepath.Join(strangeVolume(), "a")
	pathsFold = true
	if !IsRelativeTo(upper, lower) {
		t.Error("a folding file system kept the two apart")
	}
	if !pathsEqual(lower, filepath.Join(strangeVolume(), "A")) {
		t.Error("a folding file system called one tree two")
	}
	pathsFold = false
	if IsRelativeTo(upper, lower) {
		t.Error("a case-keeping file system ran them together")
	}
}

func TestComponentsKeepADriveRelativeAnchorApartFromARootedOne(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("a drive-relative path is a Windows spelling")
	}
	// `PureWindowsPath("C:x").parts` is ('C:', 'x'): the anchor is the
	// drive without a separator, and that is a different place from
	// `C:\x`.
	if got := components(`C:x`); len(got) != 2 || got[0] != "C:" {
		t.Errorf("components(`C:x`) = %q", got)
	}
	if got := components(`C:\x`); len(got) != 2 || got[0] != `C:\` {
		t.Errorf("components(`C:\\x`) = %q", got)
	}
}

// --- git, and the answers a failure has to give -------------------------

// mustResolve is `ResolvePath` where the test's own fixture is the thing
// being resolved: nothing there leads in a circle, so a failure is a
// broken fixture rather than a case.
func mustResolve(t *testing.T, path string) string {
	t.Helper()
	resolved, err := ResolvePath(path)
	if err != nil {
		t.Fatalf("ResolvePath(%q): %v", path, err)
	}
	return resolved
}

func TestAnInheritedGitDirMakesNoTwoTreesOneRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine")
	}
	base := t.TempDir()
	main := filepath.Join(base, "main")
	mkdir(t, main)
	run(t, main, "init")
	run(t, main, "-c", "user.email=t@t", "-c", "user.name=t",
		"commit", "--allow-empty", "-m", "first")
	linked := filepath.Join(base, "linked")
	run(t, main, "worktree", "add", "-b", "side", linked)
	unrelated := filepath.Join(base, "unrelated")
	mkdir(t, unrelated)
	run(t, unrelated, "init")
	// Set after the fixture, and at the registered repository: inherited by
	// a git child, GIT_DIR outranks the directory it is pointed at, so every
	// tree would answer with this common directory and the unrelated one
	// would pass for a worktree. Whoever brings a git call back without
	// `gitenv` fails here.
	t.Setenv("GIT_DIR", filepath.Join(main, ".git"))
	if !sameRepository(linked, main) {
		t.Error("a real linked worktree is not the same repository")
	}
	if sameRepository(unrelated, main) {
		t.Error("an unrelated repository passed for a worktree")
	}
}

func TestAWorktreeOfTheRegisteredRepositoryDeclaresItsWiki(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine")
	}
	base := t.TempDir()
	main := filepath.Join(base, "main")
	mkdir(t, main)
	run(t, main, "init")
	run(t, main, "-c", "user.email=t@t", "-c", "user.name=t",
		"commit", "--allow-empty", "-m", "first")
	linked := filepath.Join(base, "linked")
	run(t, main, "worktree", "add", "-b", "side", linked)

	state := filepath.Join(base, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/demo\"\npath = \""+posix(main)+"\"\n")
	write(t, filepath.Join(linked, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"90 W\"\n")
	// The registration names the main checkout; the manifest sits in a
	// linked worktree of it. Sharing the common directory is what makes
	// the two one repository, and it is the only way this write is
	// allowed at all.
	allow(t, writeCall(filepath.Join(linked, "90 W", "x.md")), state)
	deny(t, writeCall(filepath.Join(linked, "src", "a.py")), state,
		"lies outside every writable tree")
}

func TestAPlantedGitBesideAPlantedManifestBuysNothing(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine")
	}
	base := t.TempDir()
	registered := filepath.Join(base, "repo")
	mkdir(t, registered)
	run(t, registered, "init")
	planted := filepath.Join(base, "planted")
	mkdir(t, planted)
	run(t, planted, "init")

	state := filepath.Join(base, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/demo\"\npath = \""+
			posix(registered)+"\"\n")
	write(t, filepath.Join(planted, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"w\"\n")
	// A `.git` of its own is the second demand, and it is not the whole
	// of it: the common directory has to be the registered repository's.
	deny(t, writeCall(filepath.Join(planted, "w", "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	command.Env = append(gitenv.Environ(), "GIT_AUTHOR_NAME=t",
		"GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t",
		"GIT_COMMITTER_EMAIL=t@t")
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v (%s)", args, err, out)
	}
}

// linkDir points `link` at `target`, by whatever means this machine
// allows. `os.Symlink` needs developer mode or admin rights on Windows
// and a test that cannot build its own fixture would fail for a reason
// that has nothing to do with the barrier -- but a *junction* needs
// neither, and `EvalSymlinks` follows one exactly as it follows a
// symlink. That is what keeps the two cases below measured on this
// machine rather than skipped on it.
func linkDir(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err == nil {
		return
	}
	if filepath.Separator != '\\' {
		t.Skip("this machine does not let the test make a symlink")
	}
	made := exec.Command("cmd", "/c", "mklink", "/J", link, target)
	if out, err := made.CombinedOutput(); err != nil {
		t.Skipf("neither a symlink nor a junction: %v (%s)", err, out)
	}
}

func TestASymlinkOutOfTheBundleIsRefused(t *testing.T) {
	tmp := t.TempDir()
	outside := filepath.Join(tmp, "outside")
	mkdir(t, outside)
	bundle := filepath.Join(tmp, "vault", "demo")
	mkdir(t, bundle)
	linkDir(t, outside, filepath.Join(bundle, "link"))
	state := registryOf(t, tmp, bundle)
	deny(t, writeCall(filepath.Join(bundle, "link", "a.py")), state,
		"lies outside every writable tree")
}

// --- the arms a failing reader has to take ------------------------------

func TestEverySectionThatMustBeATableIsOne(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	// `data.get(section, {})` followed by `.get(...)` raises
	// AttributeError where the value is no mapping, and `decide`'s catch
	// turns every one of them into the same refusal.
	for _, section := range []string{
		"privacy", "wiki", "maintenance", "index",
	} {
		write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
			section+" = \"x\"\n\n[area]\nscope = \"project/demo\"\n")
		deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")),
			state, "loomux cannot read the registry, so it refuses")
	}
}

func TestATypesListOfNonStringsIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[wiki]\ntypes = [1]\n")
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"[wiki] types #1 must be a string, found integer")
}

func TestATypesListOfStringsIsFine(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[wiki]\ntypes = [\"Topic\"]\n")
	allow(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state)
}

func TestAGoodUntouchedDaysAndBranchAreFine(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[wiki]\nuntouched_days = 7\n"+
			"\n[maintenance]\non_merge = true\nbranch = \"main\"\n")
	allow(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state)
}

func TestAReadonlyAreaReadsItsManifestFromTheStateDirectory(t *testing.T) {
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	shut := filepath.Join(tmp, "shut")
	open := filepath.Join(tmp, "open")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/shut\"\npath = \""+posix(shut)+
			"\"\nreadonly = true\n\n[[area]]\nscope = \"project/open\""+
			"\npath = \""+posix(open)+"\"\nwiki = \""+
			posix(filepath.Join(open, "w"))+"\"\n")
	target := writeCall(filepath.Join(open, "w", "x.md"))
	// A read-only area is one we index but do not own, so its artefacts
	// and its manifest live under the state directory
	// (src/brain/registry.py:123-138). A broken file beside the tree we
	// do not own is never read -- and the target is deliberately in the
	// *other* area, so that `declaredWikiRoot`'s walk does not find it
	// either and only `_inbox_of`'s choice of directory can decide.
	write(t, filepath.Join(shut, ".loomux", "config.toml"), "not = [toml\n")
	allow(t, target, state)

	moved := config.ManifestDir(
		config.Area{Scope: "project/shut", ReadOnly: true}, state)
	write(t, filepath.Join(moved, ".loomux", "config.toml"), "not = [toml\n")
	deny(t, target, state,
		"loomux cannot read the registry, so it refuses")
}

func TestADeclarationDirectoryIsNoDeclaration(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	// `is_file()` is not `exists()`: a directory named `config.toml`
	// declares nothing, and the walk has to climb past it rather than
	// try to read it.
	mkdir(t, filepath.Join(tmp, "repo", "sub", ".loomux", "config.toml"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"sub\"\n")
	allow(t, writeCall(filepath.Join(tmp, "repo", "sub", "x.md")), state)
}

func TestAWikiLayoutSpeltWithADriveIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	// `PurePosixPath("C:/x").is_absolute()` is False, so only the windows
	// flavour catches this one -- and pathlib does not ask that the drive
	// be a letter, which is why `1:/x` is refused too.
	for _, value := range []string{"C:/w", "1:/w", "\u00c4:/w"} {
		write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
			"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \""+
				value+"\"\n")
		deny(t, writeCall(filepath.Join(tmp, "repo", "w", "x.md")), state,
			"[layout] wiki must stay inside the repository")
	}
}

func TestABrokenPayloadCannotSlipThroughAPanic(t *testing.T) {
	old := readGitFile
	t.Cleanup(func() { readGitFile = old })
	readGitFile = func(string) ([]byte, error) { panic("git file blew up") }

	tmp := t.TempDir()
	base := filepath.Join(tmp, "repo")
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/demo\"\npath = \""+posix(base)+"\"\n")
	planted := filepath.Join(tmp, "planted")
	write(t, filepath.Join(planted, ".git"), "gitdir: elsewhere\n")
	write(t, filepath.Join(planted, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"w\"\n")

	// Without the recover in `answer` the panic would end the process with
	// a code the host reads as "carry on", which is the one outcome that
	// must not happen. Caught, it becomes an ordinary refusal, and an
	// ordinary refusal blocks.
	payload := "{\"tool_name\": \"Write\", \"tool_input\": " +
		"{\"file_path\": \"" + posix(
		filepath.Join(planted, "w", "x.md")) + "\"}}"
	out := &strings.Builder{}
	code := Run(strings.NewReader(payload), out, &strings.Builder{}, state)
	if code != blockingExit {
		t.Fatalf("got %d", code)
	}
	if !strings.Contains(out.String(), "loomux broke down") {
		t.Fatalf("stdout %q", out)
	}
}

// --- the arms only a direct call or a symlink reaches -------------------

func TestABrokenManifestOnTheWalkRefusesAndNamesItself(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	// Deeper than the registered area's own declaration, so the registry
	// read passes and only `declaredWikiRoot`'s walk meets this file.
	write(t, filepath.Join(tmp, "repo", "sub", ".loomux", "config.toml"),
		"not = [toml\n")
	reason := deny(t, writeCall(filepath.Join(tmp, "repo", "sub", "x.md")),
		state, "loomux cannot read the registry, so it refuses")
	if !strings.Contains(reason, "sub") {
		t.Fatalf("the file that broke was not named: %q", reason)
	}
}

func TestASymlinkedBundleCannotOpenWhatIsAboveIt(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	mkdir(t, repo)
	outside := filepath.Join(tmp, "outside")
	mkdir(t, outside)
	linkDir(t, outside, filepath.Join(repo, "w"))
	state := registryOf(t, tmp, "")
	write(t, filepath.Join(repo, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"w\"\n")
	// `wiki_layout` judges the string; where it lands is a question only
	// the disk answers, so the resolved root has to be back inside the
	// directory that declared it. The target is a plain child of the
	// repository, so the walk does find this manifest and the
	// containment test is the only thing left to refuse it -- without
	// it the bundle would open every tree beneath `tmp`.
	deny(t, writeCall(filepath.Join(repo, "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
	// And the place the link leads to, which the walk never reaches:
	deny(t, writeCall(filepath.Join(outside, "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

func TestAReviewCentreIsDroppedWhenItsManifestBreaksUnderIt(t *testing.T) {
	// Reached by calling `reviewCentre` rather than through `Decide`:
	// the registry read looks at the same file first and refuses there.
	// The arm exists because the two reads are not one -- the file may
	// be rewritten between them, and a refusal reached by traceback is
	// no refusal at all.
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	write(t, filepath.Join(repo, ".loomux", "config.toml"), "not = [toml\n")
	got := reviewCentre([]area{{scope: "project/demo", path: repo}}, tmp)
	if got != "" {
		t.Errorf("reviewCentre = %q, want none", got)
	}
}

func TestAManifestThatCannotBeReadRefusesAWriteInItsArea(t *testing.T) {
	// The stat succeeds and the read fails: the file is held open the way an
	// editor or a sync tool can. Read as "no manifest", the area would be
	// judged without its declaration; it has to refuse.
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	declaration := filepath.Join(tmp, "repo", ".loomux", "config.toml")
	write(t, declaration, "[area]\nscope = \"project/demo\"\n")
	allow(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state)
	testlock.Lock(t, declaration)
	deny(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state,
		"loomux cannot read the registry, so it refuses")
}

func TestGoodGlobListsPass(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[index]\n"+
			"include = [\"*.md\"]\nexclude = [\"x/**\"]\n"+
			"unsearched = []\n\n[privacy]\nnever = [\"secret/**\"]\n")
	allow(t, writeCall(filepath.Join(tmp, "vault", "demo", "x.md")), state)
}

func TestPythonTypeNameKnowsAnObjectWhenItSeesOne(t *testing.T) {
	// No payload reaches this arm -- an object is judged rather than
	// named -- but the name is the one `type(payload).__name__` gives,
	// and a mapping that answered "struct" would be a lie in waiting.
	if got := pythonTypeName(map[string]any{}); got != "dict" {
		t.Errorf("pythonTypeName(map) = %q", got)
	}
}

func TestAJunctionOutOfTheBundleIsRefusedLikeASymlink(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("a junction is a Windows reparse point")
	}
	tmp := t.TempDir()
	outside := filepath.Join(tmp, "outside")
	mkdir(t, outside)
	bundle := filepath.Join(tmp, "vault", "demo")
	mkdir(t, bundle)
	made := exec.Command("cmd", "/c", "mklink", "/J",
		filepath.Join(bundle, "j"), outside)
	if out, err := made.CombinedOutput(); err != nil {
		t.Skipf("no junction on this machine: %v (%s)", err, out)
	}
	// Measured: `filepath.EvalSymlinks` answers the junction's own path
	// and `os.Lstat` calls it irregular rather than a symlink, while
	// `Path.resolve()` follows it. A resolver built on `EvalSymlinks`
	// allowed this write; the Python barrier refuses it, and a Go
	// barrier weaker than the one it replaces is no replacement.
	state := registryOf(t, tmp, bundle)
	deny(t, writeCall(filepath.Join(bundle, "j", "a.py")), state,
		"lies outside every writable tree")
}

func TestALinkThatEatsItsOwnTailIsRefusedAsUnresolvable(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("mklink is the unelevated way to build this fixture")
	}
	tmp := t.TempDir()
	loop := filepath.Join(tmp, "loop")
	made := exec.Command("cmd", "/c", "mklink", "/J", loop, loop)
	if out, err := made.CombinedOutput(); err != nil {
		t.Skipf("no junction on this machine: %v (%s)", err, out)
	}
	// A path nobody can resolve is a path nobody can place, and the
	// barrier's whole answer is where a path lies. Refusing names that
	// directly instead of comparing whatever the walk happened to hold
	// when it gave up.
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	deny(t, writeCall(filepath.Join(loop, "x.md")), state,
		"cannot resolve this path")
}

// zoneWithALongName is `nestedOf` with a zone whose name is long enough
// for the file system to keep an alias for it: `space` is short enough
// to need none, and the alias is the whole point here.
func zoneWithALongName(t *testing.T, tmp string) (state, zone string) {
	t.Helper()
	hub := filepath.Join(tmp, "vault", "91")
	zone = filepath.Join(hub, "obsidian-ai-notes")
	mkdir(t, zone)
	repo := filepath.Join(tmp, "repo")
	mkdir(t, repo)
	state = filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"hub\"\npath = \""+posix(hub)+
			"\"\nwiki = \""+posix(hub)+"\"\n\n"+
			"[[area]]\nscope = \"project/obsidian\"\npath = \""+
			posix(repo)+"\"\nwiki = \""+posix(zone)+
			"\"\nreadonly = true\n")
	return state, zone
}

// cycleOfTwo builds the fixture the finding is about: two junctions, each
// pointing at the other, so that following one link never ends.
func cycleOfTwo(t *testing.T, first, second string) {
	t.Helper()
	if filepath.Separator != '\\' {
		t.Skip("mklink is the unelevated way to build this fixture")
	}
	for _, pair := range [][2]string{{first, second}, {second, first}} {
		made := exec.Command("cmd", "/c", "mklink", "/J",
			pair[0], pair[1])
		if out, err := made.CombinedOutput(); err != nil {
			t.Skipf("no junction on this machine: %v (%s)", err, out)
		}
	}
}

func TestACycleOfTwoJunctionsIsRefusedRatherThanCounted(t *testing.T) {
	tmp := t.TempDir()
	hub := filepath.Join(tmp, "vault", "91")
	zone := filepath.Join(hub, "space")
	mkdir(t, zone)
	state := nestedOf(t, tmp)
	first := filepath.Join(zone, "a")
	cycleOfTwo(t, first, filepath.Join(hub, "b"))
	// A cycle of even length is the case a hop *count* cannot decide:
	// every turn swaps one link for the other, so the path alternates
	// between the zone and the writable tree and the parity of whatever
	// bound stops the walk picks the verdict. Refusing is the one answer
	// that does not depend on it.
	deny(t, writeCall(filepath.Join(first, "x.md")), state,
		"cannot resolve this path")
}

func TestAWritableTreeThatLeadsInACircleRefusesEveryCall(t *testing.T) {
	tmp := t.TempDir()
	mkdir(t, filepath.Join(tmp, "vault"))
	wiki := filepath.Join(tmp, "vault", "demo")
	cycleOfTwo(t, wiki, filepath.Join(tmp, "vault", "other"))
	// A registered tree nobody can place is not quietly dropped: losing
	// a writable root would close an area nobody meant to close, and the
	// same silence over a zone would open one.
	state := registryOf(t, tmp, wiki)
	deny(t, writeCall(filepath.Join(tmp, "repo", "a.py")), state,
		"cannot resolve a registered tree")
}

func TestAWorkspaceThatLeadsInACircleRefusesEveryCall(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	state := registryOf(t, tmp, "", "workspace = true")
	if err := os.Remove(repo); err != nil {
		t.Fatal(err)
	}
	cycleOfTwo(t, repo, filepath.Join(tmp, "other"))
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"cannot resolve a registered tree")
}

func TestAForbiddenZoneThatLeadsInACircleRefusesEveryCall(t *testing.T) {
	tmp := t.TempDir()
	mkdir(t, filepath.Join(tmp, "vault"))
	zone := filepath.Join(tmp, "vault", "demo")
	cycleOfTwo(t, zone, filepath.Join(tmp, "vault", "other"))
	state := registryOf(t, tmp, zone, "readonly = true")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"cannot resolve a registered tree")
}

func TestAReviewCentreThatLeadsInACircleGrantsNoExemption(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	cycleOfTwo(t, filepath.Join(repo, "95"), filepath.Join(repo, "96"))
	write(t, filepath.Join(repo, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nreview = \"95\"\n")
	// Every failure of the review centre answers "": the exemption is
	// withdrawn, and nothing that was open closes. So the proposal is
	// refused for lying outside the trees, not for the circle.
	deny(t, writeCall(filepath.Join(tmp, "elsewhere", "proposal.md")),
		state, "lies outside every writable tree")
}

func TestADeclaredWikiThatLeadsInACircleRefusesTheCall(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	cycleOfTwo(t, filepath.Join(repo, "w"), filepath.Join(repo, "w2"))
	write(t, filepath.Join(repo, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"w\"\n")
	// The declared place is the half of `declaredWikiRoot` that can lead
	// into a circle; the directory it is declared in came out of the
	// parents of a path that resolved before this was ever asked.
	deny(t, writeCall(filepath.Join(repo, "src", "a.py")), state,
		"cannot read the registry")
}

func TestAnAreaWhosePathLeadsInACircleIsNobodysRepository(t *testing.T) {
	tmp := t.TempDir()
	circle := filepath.Join(tmp, "circle")
	cycleOfTwo(t, circle, filepath.Join(tmp, "other"))
	state := filepath.Join(tmp, "state")
	// Two areas: one to give the registry a writable tree, and one whose
	// own path leads in a circle and is named by the manifest below.
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"open\"\npath = \""+posix(tmp)+
			"\"\nwiki = \""+posix(filepath.Join(tmp, "vault"))+"\"\n\n"+
			"[[area]]\nscope = \"gone\"\npath = \""+posix(circle)+"\"\n")
	here := filepath.Join(tmp, "here")
	write(t, filepath.Join(here, ".loomux", "config.toml"),
		"[area]\nscope = \"gone\"\n\n[layout]\nwiki = \"w\"\n")
	// `sameRepository` answers false rather than raising, which reads as
	// "this manifest declares nothing" -- and the call is refused for
	// lying outside the one tree the registration opens.
	deny(t, writeCall(filepath.Join(here, "w", "x.md")), state,
		"lies outside every writable tree")
}

func TestAJunctionOntoAMissingDirectoryCarriesTheWriteOut(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("a junction is a Windows reparse point")
	}
	tmp := t.TempDir()
	bundle := filepath.Join(tmp, "vault", "demo")
	mkdir(t, bundle)
	link := filepath.Join(bundle, "j")
	away := filepath.Join(tmp, "outside", "absent")
	made := exec.Command("cmd", "/c", "mklink", "/J", link, away)
	if out, err := made.CombinedOutput(); err != nil {
		t.Skipf("no junction on this machine: %v (%s)", err, out)
	}
	// The junction points at a directory that is not there, so the open
	// fails and the file system answers nothing. `realpath` reads the
	// link itself at that point and follows it; a resolver that only
	// opened paths would answer `bundle/j/a.py`, which lies inside the
	// bundle, and let the write out of it.
	state := registryOf(t, tmp, bundle)
	deny(t, writeCall(filepath.Join(link, "a.py")), state,
		"lies outside every writable tree")
}

func TestAPathWithNoAnchorComesBackAsItWasSpelt(t *testing.T) {
	// `finalPath` is asked directly, because `ResolvePath` anchors every
	// path before it gets here. `_getfinalpathname_nonstrict` ends its
	// walk by answering the tail it collected, and this is that arm:
	// nothing to open, nothing to shorten, and no anchor to stop at.
	got, err := finalPath(filepath.Join("no-such-name", "x.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join("no-such-name", "x.md") {
		t.Errorf("finalPath = %q", got)
	}
}

func TestALinkTargetOfItsOwnReplacesTheDirectoryItSatIn(t *testing.T) {
	// Both arms asked here rather than through a fixture: on Windows
	// every junction target is absolute, so the relative arm -- the
	// ordinary spelling of a posix symlink -- has no fixture this
	// machine can build without the privilege `os.Symlink` wants.
	base := filepath.Join(strangeVolume(), "a", "b")
	away := filepath.Join(strangeVolume(), "elsewhere")
	if got := under(base, away); got != filepath.Clean(away) {
		t.Errorf("under(absolute) = %q, want %q", got, away)
	}
	want := filepath.Join(strangeVolume(), "a", "c")
	if got := under(base, filepath.Join("..", "c")); got != want {
		t.Errorf("under(relative) = %q, want %q", got, want)
	}
}

// --- the anchor a rooted path without a drive gets -----------------------

func TestARootedTargetIsAnchoredAtTheDriveNotTheDirectory(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("only Windows spells a rooted path without a drive")
	}
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	// The working directory lies in a writable tree, which is the
	// regular case: a PreToolUse hook runs where the session stands, and
	// the session stands in what it edits. That is what makes the
	// difference visible -- joining onto this directory lands *inside*
	// the tree that is open, so a barrier that joins allows.
	work := filepath.Join(tmp, "vault", "91")
	mkdir(t, work)
	t.Chdir(work)
	zone := filepath.Join(tmp, "vault", "91", "space", "x.md")
	rooted := zone[len(filepath.VolumeName(zone)):]
	// Measured on this machine against Python 3.14: `ntpath.isabs`
	// answers False for a rooted path carrying no drive, so `realpath`
	// falls into `join(cwd, path)` -- and `ntpath.join` keeps only that
	// directory's *drive*. `\Users\x` from anywhere on C: is `C:\Users\x`.
	// Node's `path.resolve` answers the same, so the runtime that
	// performs the write puts the file where Python looked.
	deny(t, writeCall(rooted), state, "read-only")
}

func TestARootedTargetSpeltWithSlashesIsAnchoredAsWell(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("only Windows spells a rooted path without a drive")
	}
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	work := filepath.Join(tmp, "vault", "91")
	mkdir(t, work)
	t.Chdir(work)
	zone := filepath.Join(tmp, "vault", "91", "space", "x.md")
	// The spelling a JSON payload carries most often: a host that built
	// the path in a posix idiom hands the slashes on unchanged, and both
	// `ntpath` and this side read them as separators.
	rooted := filepath.ToSlash(zone[len(filepath.VolumeName(zone)):])
	deny(t, writeCall(rooted), state, "read-only")
}

// --- the spelling that names a drive but no root -------------------------

// TestADriveRelativeTargetIsRefused holds the barrier to the one Windows
// spelling it used to answer about a place the write never reaches.
//
// `D:evil.txt` carries a volume and no root, so `filepath.IsAbs` says false
// and `anchor` joins it onto the hook's working directory -- which
// concatenates: `Join("D:\\work", "C:secrets.txt")` is `D:\\work\\C:secrets.txt`.
// Windows resolves the spelling against *that drive's* own current directory
// instead, so the write lands outside every registered tree while the barrier
// was answering about a path inside one. `spelled` then folds the bogus
// component to its part before the colon, which is what made the fake path a
// component prefix of the workspace.
//
// No legitimate host sends this spelling, so the answer is a refusal rather
// than a reimplementation of ntpath's per-drive current directories.
func TestADriveRelativeTargetIsRefused(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("only Windows spells a path relative to a drive")
	}
	tmp := t.TempDir()
	state := nestedOf(t, tmp)
	// The working directory lies in the writable tree, which is the regular
	// case and the one that made the defect visible: joining lands inside the
	// tree that is open, so a barrier that joins allows.
	work := filepath.Join(tmp, "vault", "91")
	mkdir(t, work)
	t.Chdir(work)
	for _, target := range []string{"D:evil.txt", `C:foo\bar`} {
		deny(t, writeCall(target), state, "cannot resolve this path")
	}
}

func TestResolvePathRefusesAVolumeWithoutARoot(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("only Windows spells a path relative to a drive")
	}
	// The bare volume is the same spelling with nothing after it: `C:` is
	// that drive's current directory, not its root.
	for _, target := range []string{"D:evil.txt", `C:foo\bar`, "C:"} {
		if _, err := ResolvePath(target); err == nil {
			t.Errorf("ResolvePath(%q) answered a place", target)
		}
	}
}

func TestResolvePathKeepsTheRootedSpellingsItAlwaysTook(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("only Windows spells a path relative to a drive")
	}
	tmp := t.TempDir()
	// A drive with a root, a rooted path without a drive and a relative one:
	// the three spellings the refusal above must not touch.
	rooted := tmp[len(filepath.VolumeName(tmp)):]
	for _, target := range []string{tmp, rooted, "x.md"} {
		if _, err := ResolvePath(target); err != nil {
			t.Errorf("ResolvePath(%q): %v", target, err)
		}
	}
}
