package guard

import (
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
	// Each defect `config.ReadDeclaration` knows has its own answer, and a mutant that
	// drops one of the early arms lets the file fall through to a later one:
	// dropped TOML or `[area]`-shape errors reach `is missing "scope"`, and a
	// dropped read error parses the empty data into a document without an
	// area and answers config.ErrNoArea -- which every caller reads as "no
	// manifest", so the barrier would open instead of refusing. A reason
	// that names the wrong line sends the reader to the wrong file.
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	target := writeCall(filepath.Join(tmp, "vault", "demo", "x.md"))
	for body, want := range map[string]string{
		"not = [toml\n":                        "not valid TOML",
		"area = \"x\"\n":                       "[area] must be a table, found string",
		"[area]\nscope = \"project/demo\"\n\n": "",
	} {
		write(t, filepath.Join(tmp, "repo", ".loomux", "config.toml"), body)
		if want == "" {
			allow(t, target, state)
			continue
		}
		reason := deny(t, target, state, want)
		if strings.Contains(reason, "is missing \"scope\"") {
			t.Fatalf("the wrong defect was named: %q", reason)
		}
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
	// Where the temporary directory lies inside a checkout, the registered
	// side names that checkout's repository, and the test would pass even
	// with the emptiness test in `sameRepository` removed.
	if got := registeredCommon(repo); got != "" {
		t.Fatalf("the fixture lies inside a repository (%q); this test "+
			"needs a temporary directory outside every checkout", got)
	}
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/demo\"\npath = \""+posix(repo)+"\"\n")
	planted := filepath.Join(tmp, "planted")
	write(t, filepath.Join(planted, ".git"), "gitdir: elsewhere\n")
	write(t, filepath.Join(planted, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[layout]\nwiki = \"w\"\n")
	deny(t, writeCall(filepath.Join(planted, "w", "x.md")), state,
		"the registry declares no writable wiki path and no workspace")
}

func TestAMissingKeyIsBlamedOnTheEntryThatCanBeFound(t *testing.T) {
	// Where the scope is known the entry is named by it; where it is the
	// missing key, by its position.
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/demo\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		`[[area]] "project/demo" is missing "path"`)
	write(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"a\"\npath = \"/a\"\n\n[[area]]\npath = \"/x\"\n")
	deny(t, writeCall(filepath.Join(tmp, "x.md")), state,
		`[[area]] #2 is missing "scope"`)
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
