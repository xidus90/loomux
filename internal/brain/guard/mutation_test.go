package guard

import (
	"errors"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The tests the second mutation round asked for. They are gathered here
// rather than scattered because what they have in common is how they were
// found: each one is the case a surviving mutant proved nobody had asked
// for. Every expectation below was measured against the Python barrier or
// against the library function it mirrors, never derived from the Go code
// that has to satisfy it.

func TestAReadonlyAreaWithoutAWikiDeclaresNoZoneAtAll(t *testing.T) {
	tmp := t.TempDir()
	// `resolvePath("")` answers the working directory, so a zone built
	// from an unsaid wiki would forbid everything below wherever the
	// session happens to stand. The chdir is what makes that visible:
	// without it the mutant's zone and the target never meet.
	t.Chdir(tmp)
	state := registryOf(t, tmp, "", "readonly = true", "workspace = true")
	allow(t, writeCall(filepath.Join(tmp, "repo", "a.py")), state)
}

func TestTheReviewCentreSurvivesTheAreasThatDeclareNone(t *testing.T) {
	// Three areas and one review centre. The first carries no manifest
	// at all -- normal, an area may be registered before it declares
	// itself -- and the second carries one that says nothing about a
	// review. Neither may close the exemption, and a barrier that read a
	// missing file or asked a missing key for a string would let both.
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	quiet := filepath.Join(tmp, "quiet")
	plain := filepath.Join(tmp, "plain")
	owner := filepath.Join(tmp, "owner")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"quiet\"\npath = \""+posix(quiet)+"\"\n\n"+
			"[[area]]\nscope = \"plain\"\npath = \""+posix(plain)+
			"\"\n\n[[area]]\nscope = \"owner\"\npath = \""+posix(owner)+
			"\"\nwiki = \""+posix(filepath.Join(owner, "w"))+"\"\n")
	write(t, filepath.Join(plain, ".loomux", "config.toml"),
		"[area]\nscope = \"plain\"\n")
	write(t, filepath.Join(owner, ".loomux", "config.toml"),
		"[area]\nscope = \"owner\"\n\n[layout]\nreview = \"95\"\n")
	allow(t, writeCall(filepath.Join(owner, "95", "c1", "proposal.md")),
		state)
}

func TestACallWithNoPathIsNoneOfItsBusinessEvenWithNothingOpen(t *testing.T) {
	// "Not a file write at all" has to be decided before the registry is
	// consulted, or a vault with no writable tree would answer a call
	// that names no file with a complaint about the registry -- a
	// refusal of nothing, and a reason about the wrong thing.
	tmp := t.TempDir()
	state := registryOf(t, tmp, "")
	payload := map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]any{"content": "x"},
	}
	allow(t, payload, state)
}

func TestAThresholdOfOneDayIsAcceptedAndZeroIsNot(t *testing.T) {
	// `untouched_days must be an integer >= 1`, so 1 is the first value
	// that passes and 0 the last that does not. Both asked, because a
	// bound tested from one side is a bound half tested.
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	target := writeCall(filepath.Join(tmp, "vault", "demo", "x.md"))
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[wiki]\nuntouched_days = 1\n")
	allow(t, target, state)
	write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[wiki]\nuntouched_days = 0\n")
	deny(t, target, state, "[wiki] untouched_days must be an integer >= 1")
}

func TestADriveIsOnlyADriveWithTheSeparatorBehindIt(t *testing.T) {
	// Measured by running `wiki_layout`, value by value: `C:x` comes
	// back untouched -- `PureWindowsPath("C:x").is_absolute()` is False,
	// a drive-relative path is not an absolute one -- while `C:/` is
	// refused. The pair holds both halves of the drive test: without the
	// separator check `C:x` would be refused, and without the length
	// check `C:/` would pass.
	for value, want := range map[string]bool{
		"C:x": true, "ab:/c": true, ":/x": true,
		"C:/": false, "/srv/w": false, "//": false,
	} {
		got, err := wikiLayout(map[string]any{"wiki": value})
		if want && (err != nil || got != value) {
			t.Errorf("wikiLayout(%q) = %q, %v; wanted it kept", value,
				got, err)
		}
		if !want && err == nil {
			t.Errorf("wikiLayout(%q) was kept", value)
		}
	}
}

func TestARelativeTargetInsideTheBundleIsAllowed(t *testing.T) {
	// The other half of the relative-path case. One test alone cannot
	// tell "resolved against the working directory" from "never resolved
	// at all", because both refuse a path outside the tree -- only a
	// relative path that *is* inside says which of the two happened.
	tmp := t.TempDir()
	bundle := filepath.Join(tmp, "vault", "demo")
	mkdir(t, bundle)
	state := registryOf(t, tmp, bundle)
	t.Chdir(bundle)
	allow(t, writeCall("x.md"), state)
}

func TestAColonAtTheStartLeavesNothingOfTheName(t *testing.T) {
	// `name.split(":", 1)[0]` on `:x` is the empty string, because
	// everything from the first colon is a stream name -- including a
	// colon that stands first. Measured in Python, not reasoned from the
	// index.
	if got := spelling(":x"); got != "" {
		t.Errorf("spelling(\":x\") = %q, want the empty string", got)
	}
}

func TestComponentsSplitsWhereAndOnlyWherePathlibSplits(t *testing.T) {
	// A volume with nothing behind it: `PureWindowsPath("C:").parts` is
	// ('C:',), and a reader that indexed past the volume would fault on
	// it rather than answer.
	if got := components("C:"); len(got) != 1 || got[0] != "C:" {
		t.Errorf("components(`C:`) = %q", got)
	}
	// A relative path has no anchor, and an empty first component would
	// make every comparison against it start one place too late.
	if got := components(filepath.Join("a", "b")); len(got) != 2 ||
		got[0] != "a" {
		t.Errorf("components(`a/b`) = %q", got)
	}
	// U+012F is one rune whose low byte is the forward slash. A split
	// that asked `os.IsPathSeparator(byte(r))` without first asking that
	// the rune be ASCII would cut a name in half here.
	if got := components("a\u012fb"); len(got) != 1 {
		t.Errorf("components(\"a\\u012fb\") = %q, want one name", got)
	}
}

func TestIsRelativeToIsReflexiveAndSurvivesALongerBase(t *testing.T) {
	// Both measured in Python: `Path("C:/a/b").is_relative_to(itself)` is
	// True, and a base longer than the path is False rather than an
	// error. The second is the one that matters here -- a comparison
	// that sliced first and asked the length afterwards would fault.
	own := filepath.Join(strangeVolume(), "a", "b")
	if !isRelativeTo(own, own) {
		t.Error("a path was called no relative of itself")
	}
	deeper := filepath.Join(own, "c", "d")
	if isRelativeTo(own, deeper) {
		t.Error("a path was called a relative of something below it")
	}
}

func TestAProposalAboveTheReviewCentreIsRefused(t *testing.T) {
	// The same question through `Decide`: the centre lies deeper than the
	// file, so the exemption cannot reach it and nothing may fault while
	// finding that out.
	tmp := t.TempDir()
	state := withReview(t, tmp, "\"a/b/c\"")
	deny(t, writeCall(filepath.Join(tmp, "proposal.md")), state,
		"lies outside every writable tree")
}

func TestTheReviewCentreIsMatchedTheWayTheFileSystemMatches(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("posix keeps the case, so there is nothing to fold")
	}
	// `is_relative_to` goes through `normcase`, which lowers on Windows:
	// measured, `Path("C:/a/b").is_relative_to(Path("C:/A"))` is True.
	// So a centre declared `95 X` covers a case written into `95 x`, and
	// a barrier that compared the two letter for letter would refuse a
	// proposal the Python side allows.
	tmp := t.TempDir()
	state := withReview(t, tmp, "\"95 X\"")
	allow(t, writeCall(filepath.Join(tmp, "repo", "95 x", "c1",
		"proposal.md")), state)
}

func TestTheLastCharacterOfThePlaneIsNoSurrogatePair(t *testing.T) {
	// U+FFFF is the last code point of the basic plane and needs one
	// escape, not two: measured, `json.dumps("\uffff")` is `"\uffff"`.
	// The bound is asked from the far side as well, at U+10000.
	if got := pythonJSONString("\uffff"); got != `"\uffff"` {
		t.Errorf("pythonJSONString(U+FFFF) = %s", got)
	}
	if got := pythonJSONString("\U00010000"); got != `"\ud800\udc00"` {
		t.Errorf("pythonJSONString(U+10000) = %s", got)
	}
}

func TestABrokenManifestSaysWhichDefectItFound(t *testing.T) {
	// Each defect `readManifest` knows has its own answer, and a mutant that
	// drops one of the early arms lets the file fall through to a later one:
	// dropped TOML or `[area]`-shape errors reach "scope is required", and a
	// dropped read error parses the empty data into a document without an
	// area and answers errNoArea -- which every caller reads as "no
	// manifest", so the barrier would open instead of refusing. A reason
	// that names the wrong line sends the reader to the wrong file.
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	target := writeCall(filepath.Join(tmp, "vault", "demo", "x.md"))
	for body, want := range map[string]string{
		"not = [toml\n":                        "not valid TOML",
		"area = \"x\"\n":                       "[area] must be a table",
		"[area]\nscope = \"project/demo\"\n\n": "",
	} {
		write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"), body)
		if want == "" {
			allow(t, target, state)
			continue
		}
		reason := deny(t, target, state, want)
		if strings.Contains(reason, "scope is required") {
			t.Fatalf("the wrong defect was named: %q", reason)
		}
	}
	// And the read error, which no caller can reach through a stat that
	// succeeded -- so it is asked of the function.
	_, err := readManifest(filepath.Join(tmp, "gone.toml"))
	if errors.Is(err, errNoArea) || !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a missing manifest answered %v", err)
	}
}

func TestAnUnreadableGitFileMakesNoTwoTreesOneRepository(t *testing.T) {
	// Neither side is a repository this reading understands: the planted
	// `.git` names an administration directory that is not there, and the
	// registered tree has no `.git` above it. Both answers are "", and
	// without the emptiness test in `sameRepository` two empty answers
	// would compare equal -- so any directory carrying a `.git` would pass
	// for a worktree of a tree outside every repository.
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	repo := filepath.Join(tmp, "repo")
	mkdir(t, repo)
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/demo\"\npath = \""+posix(repo)+"\"\n")
	planted := filepath.Join(tmp, "planted")
	write(t, filepath.Join(planted, ".git"), "gitdir: elsewhere\n")
	write(t, filepath.Join(planted, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"w\"\n")
	deny(t, writeCall(filepath.Join(planted, "w", "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

func TestAValueOfNothingButDotsAndSlashesNamesTheRoot(t *testing.T) {
	// `PurePosixPath("./").parts` is empty, so `wiki_layout` refuses it
	// as the repository root -- measured, along with `.`, `./.` and
	// `.//`. This is the pair `namesAPart` needs both halves for: `"./"`
	// splits into `"."` and `""`, and a test that only refused the `"."`
	// would take the empty component for a name and hand the whole
	// repository over as the declared bundle. Reached, unlike `"//"`,
	// because nothing about `"./"` looks absolute to `escapes`.
	for _, value := range []string{".", "./", ".//", "./.", "a/../.."} {
		if got, err := wikiLayout(
			map[string]any{"wiki": value}); err == nil {
			t.Errorf("wikiLayout(%q) = %q, want a refusal", value, got)
		}
	}
}

func TestAMissingKeyIsBlamedOnTheEntryThatCanBeFound(t *testing.T) {
	// `entry.get("scope", entry)` names the scope where there is one and
	// falls back to the whole entry where there is not, so the reader can
	// find the line they wrote. Measured on both shapes:
	//
	//   [[area]] entry 'project/demo' is missing the 'path' key
	//   [[area]] entry {'path': '/x'} is missing the 'scope' key
	//
	// A mutant that took the fallback whenever a scope *was* present left
	// the refusal standing and named a table instead of the name -- the
	// one piece of information the message exists to carry.
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/demo\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"[[area]] entry 'project/demo' is missing the 'path' key")
	// The other way round the scope cannot be named, and the entry has
	// to stand in for it. Only that it is *not* the empty answer is held
	// here: Go has no order in a map, so the rendering of a table is a
	// documented difference to Python's.
	write(t, filepath.Join(state, "registry.toml"), "[[area]]\npath = \"/x\"\n")
	reason := deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		"is missing the 'scope' key")
	if strings.Contains(reason, "entry None is missing") {
		t.Fatalf("the entry was named as nothing: %q", reason)
	}
	if !strings.Contains(reason, "/x") {
		t.Fatalf("the entry itself was not named: %q", reason)
	}
}

func TestASubdirectoryOfTheRegisteredRepositoryIsNoWorktreeOfIt(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine")
	}
	base := t.TempDir()
	registered := filepath.Join(base, "repo")
	mkdir(t, registered)
	run(t, registered, "init")
	// A subdirectory shares the registered repository's common directory
	// -- `git rev-parse --git-common-dir` answers the same place from
	// both -- so the common directory alone says nothing about being a
	// worktree. The `.git` of its own is the whole of the difference,
	// and a barrier without it would let a manifest planted in any
	// subdirectory declare that subdirectory's own bundle.
	inside := filepath.Join(registered, "sub")
	mkdir(t, inside)
	state := filepath.Join(base, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/demo\"\npath = \""+
			posix(registered)+"\"\n")
	write(t, filepath.Join(inside, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"w\"\n")
	deny(t, writeCall(filepath.Join(inside, "w", "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}
