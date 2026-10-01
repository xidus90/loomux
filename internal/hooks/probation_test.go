package hooks

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/sessions"
)

// probationProject is a Go project loomux recognises as a root, committed,
// with the file holding text.
func probationProject(t *testing.T, armed string) string {
	t.Helper()
	root := project(t)
	writeWorldFile(t, root, "go.mod", "module m\n")
	writeWorldFile(t, root, "a_test.go", "package m\n")
	gitInit(t, root)
	if armed != "" {
		writeWorldFile(t, root, ".loomux/armed.toml", armed)
	}
	return root
}

// leaveSeen files a stand of the stop gate as some earlier session left it,
// with the armed lanes it ran under.
func leaveSeen(t *testing.T, root, head, report string, armed ...string) {
	t.Helper()
	if err := sessions.WriteState(root, "earlier", sessions.SessionState{Seen: &sessions.Seen{Tree: "t", Head: head, Armed: armed, Report: report, At: time.Now()}}); err != nil {
		t.Fatal(err)
	}
}

func TestSessionStartNamesTheLanesInProbation(t *testing.T) {
	root := probationProject(t, "armed = [\"lint/go@.\"]\n")
	lines := probationLines(root)
	want := []string{
		"loomux: lane coverage/go@. is in probation: what it finds warns and fails no gate until a green commit arms it",
		"loomux: lane test/go@. is in probation: what it finds warns and fails no gate until a green commit arms it",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("%q", lines)
	}
	// Through the hook, in the channel session start already answers in.
	var out, errOut bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1","hook_event_name":"SessionStart"}`), &out, &errOut, root, "claude", "1.0.0")
	if code != ExitOK || !strings.Contains(out.String(), "additionalContext") || !strings.Contains(out.String(), "lane test/go@. is in probation") {
		t.Fatalf("%d %q %q", code, out.String(), errOut.String())
	}
}

// Without the file, and with every lane armed, session start says what it
// says today: nothing.
func TestSessionStartSaysNothingWithoutALaneInProbation(t *testing.T) {
	for name, armed := range map[string]string{"no file": "", "every lane armed": "armed = [\"coverage/go@.\", \"lint/go@.\", \"test/go@.\"]\n"} {
		root := probationProject(t, armed)
		// A stand some session left while lanes were in probation says nothing
		// once none is.
		leaveSeen(t, root, headOf(t, root), "lint/go: failed (probation)\n")
		var out, errOut bytes.Buffer
		code := SessionStart(strings.NewReader(`{"session_id":"s1","hook_event_name":"SessionStart"}`), &out, &errOut, root, "claude", "1.0.0")
		if code != ExitOK || out.Len() != 0 {
			t.Errorf("%s: %d %q %q", name, code, out.String(), errOut.String())
		}
	}
}

func TestSessionStartSaysAnUnreadableArmedFile(t *testing.T) {
	root := probationProject(t, "armed = 1\n")
	lines := probationLines(root)
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "loomux: .loomux/armed.toml ") || !strings.HasSuffix(lines[0], "every lane is armed") {
		t.Fatalf("%q", lines)
	}
}

// What the stop gate last saw is passed on while HEAD is where it saw it,
// from whichever session saw it, and cut where it runs long.
func TestSessionStartPassesOnWhatTheStopGateLastSaw(t *testing.T) {
	root := probationProject(t, "armed = []\n")
	head := headOf(t, root)
	leaveSeen(t, root, head, "lint/go: failed (probation) [preset] 0.1s\na.go:1: bad\nprobation: lint/go@. (warn only until a green commit arms them)\n")
	joined := strings.Join(probationLines(root), "\n")
	for _, want := range []string{
		"loomux: lane test/go@. is in probation",
		"loomux: at the last turn end that checked this commit, the lanes in probation reported:\nlint/go: failed (probation) [preset] 0.1s\na.go:1: bad\nprobation: lint/go@. (warn only until a green commit arms them)",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in %q", want, joined)
		}
	}
	if strings.HasSuffix(joined, "\n") || strings.Contains(joined, "\n\n") {
		t.Errorf("an empty line is passed on: %q", joined)
	}
	// Up to the limit a report is passed on whole; beyond it, it is cut and
	// says how much is left out.
	for _, n := range []int{maxSeenLines - 1, maxSeenLines, maxSeenLines + 1, maxSeenLines + 5} {
		var long strings.Builder
		for i := 1; i <= n; i++ {
			fmt.Fprintf(&long, "finding %d\n", i)
		}
		leaveSeen(t, root, head, long.String())
		joined = strings.Join(probationLines(root), "\n")
		last := fmt.Sprintf("finding %d", min(n, maxSeenLines))
		if rest := n - maxSeenLines; rest > 0 {
			want := fmt.Sprintf("%s\nloomux: %d more lines; `loomux check stop` shows all", last, rest)
			if !strings.HasSuffix(joined, want) || strings.Contains(joined, fmt.Sprintf("finding %d\n", maxSeenLines+1)) {
				t.Errorf("%d lines: want the cut %q in %q", n, want, joined)
			}
		} else if !strings.HasSuffix(joined, last) || strings.Contains(joined, "more lines") {
			t.Errorf("%d lines: want them whole in %q", n, joined)
		}
	}
	// A commit since then: the stand is about another HEAD and stays unsaid.
	writeWorldFile(t, root, "b.txt", "x")
	git(t, root, "add", "b.txt")
	git(t, root, "commit", "-q", "-m", "second")
	if joined = strings.Join(probationLines(root), "\n"); strings.Contains(joined, "finding") || strings.Contains(joined, "last turn end") || !strings.Contains(joined, "lane lint/go@. is in probation") {
		t.Errorf("after a commit: %q", joined)
	}
	// And a stand about the HEAD now is said again.
	leaveSeen(t, root, headOf(t, root), "lint/go: failed (probation)\n")
	if joined = strings.Join(probationLines(root), "\n"); !strings.HasSuffix(joined, "reported:\nlint/go: failed (probation)") {
		t.Errorf("a stand of the new HEAD: %q", joined)
	}
}

// A lane armed or disarmed since the stand changes what the report would
// say, on the same HEAD: where .loomux/ is ignored no commit follows a
// `loomux gate arm`. The report is passed on only under the lanes it ran
// under.
func TestSessionStartDropsAStandOfOtherArmedLanes(t *testing.T) {
	root := probationProject(t, "armed = [\"lint/go@.\"]\n")
	head := headOf(t, root)
	for name, c := range map[string]struct {
		armed []string
		said  bool
	}{
		"the same lanes":        {[]string{"lint/go@."}, true},
		"another lane":          {[]string{"test/go@."}, false},
		"one more lane":         {[]string{"lint/go@.", "test/go@."}, false},
		"before the lane armed": {nil, false},
	} {
		leaveSeen(t, root, head, "test/go: failed (probation)\n", c.armed...)
		joined := strings.Join(probationLines(root), "\n")
		if said := strings.Contains(joined, "test/go: failed (probation)"); said != c.said || !strings.Contains(joined, "lane test/go@. is in probation") {
			t.Errorf("%s: report said %v, want %v: %q", name, said, c.said, joined)
		}
	}
	// The stop hook keeps the file's keys, an orphaned one among them: that
	// is what the stand is compared with, not the lanes alone.
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = [\"lint/go@.\", \"lint/python@old\"]\n")
	leaveSeen(t, root, head, "test/go: failed (probation)\n", "lint/go@.", "lint/python@old")
	if joined := strings.Join(probationLines(root), "\n"); !strings.Contains(joined, "test/go: failed (probation)") {
		t.Errorf("with an orphaned entry the report is not said: %q", joined)
	}
}

// agy fires the start before every model call; the lanes are named at the
// first and not again, as the stale binary is.
func TestARepeatedStartDoesNotNameTheProbationAgain(t *testing.T) {
	root := probationProject(t, "armed = []\n")
	for payload, named := range map[string]bool{
		`{"conversationId":"s1","invocationNum":0}`: true,
		`{"conversationId":"s1","invocationNum":2}`: false,
	} {
		var out, errOut bytes.Buffer
		code := SessionStart(strings.NewReader(payload), &out, &errOut, root, "antigravity", "1.0.0")
		if code != ExitOK || strings.Contains(out.String(), "is in probation") != named {
			t.Errorf("%s: %d %q %q", payload, code, out.String(), errOut.String())
		}
	}
}

func TestStatusShowsTheProbation(t *testing.T) {
	root := probationProject(t, "")
	var b bytes.Buffer
	if renderProbation(&b, root); b.Len() != 0 {
		t.Fatalf("no file: %q", b.String())
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = [\"lint/go@.\", \"lint/python@.\"]\n")
	renderProbation(&b, root)
	for _, want := range []string{
		"\n--------------------------------------------------------------------------------\n Lane Probation (.loomux/armed.toml)\n--------------------------------------------------------------------------------\n",
		" [ARMED] lint/go@.\n",
		" [PROBATION] coverage/go@.: warns only until a green commit arms it\n",
		" [PROBATION] test/go@.: warns only until a green commit arms it\n",
		" [ORPHAN] lint/python@.: no lane answers to this entry\n",
		" [WARN] no pre-commit hook arms lanes: call `loomux check precommit --arm` there, or arm by hand with `loomux gate arm`\n",
	} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("missing %q in %q", want, b.String())
		}
	}
	for _, unwanted := range []string{"[ARMED] coverage", "[ARMED] test", "[PROBATION] lint", "[ORPHAN] lint/go@.", "ignored by git"} {
		if strings.Contains(b.String(), unwanted) {
			t.Errorf("%q in %q", unwanted, b.String())
		}
	}
	b.Reset()
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = 1\n")
	if renderProbation(&b, root); !strings.Contains(b.String(), " Lane Probation (") || !strings.Contains(b.String(), " [WARN] .loomux/armed.toml ") ||
		strings.Contains(b.String(), "pre-commit hook") {
		t.Errorf("unreadable: %q", b.String())
	}
}

// The section stands in the status report, as its last before the closing
// rule, and a project without the file has none.
func TestStatusCarriesTheProbationSection(t *testing.T) {
	root := probationProject(t, "")
	var out, errOut bytes.Buffer
	if Status(&out, &errOut, root); strings.Contains(out.String(), "Lane Probation") {
		t.Fatalf("no file: %q", out.String())
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = []\n")
	out.Reset()
	Status(&out, &errOut, root)
	section := strings.Index(out.String(), " Lane Probation (.loomux/armed.toml)\n")
	if section < 0 || section < strings.Index(out.String(), " Lane Tools On This Machine\n") || !strings.HasSuffix(out.String(), howToArm+"\n"+strings.Repeat("=", 80)+"\n") {
		t.Fatalf("%q", out.String())
	}
}

// A project that ignores .loomux keeps the file out of every commit; the
// report says so, and says nothing where only the state is ignored.
func TestStatusWarnsAboutAnIgnoredFile(t *testing.T) {
	const warning = " [WARN] .loomux/armed.toml is ignored by git: it reaches no commit and holds on this machine only\n"
	root := probationProject(t, "armed = []\n")
	var b bytes.Buffer
	if renderProbation(&b, root); !strings.Contains(b.String(), "Lane Probation") || strings.Contains(b.String(), "ignored by git") {
		t.Fatalf("nothing is ignored: %q", b.String())
	}
	writeWorldFile(t, root, ".gitignore", "/.loomux/state/\n")
	b.Reset()
	if renderProbation(&b, root); strings.Contains(b.String(), "ignored by git") {
		t.Fatalf("only the state is ignored: %q", b.String())
	}
	writeWorldFile(t, root, ".gitignore", ".loomux/\n")
	b.Reset()
	if renderProbation(&b, root); !strings.Contains(b.String(), warning) {
		t.Fatalf("%q", b.String())
	}
}

func TestStatusNamesAHookThatDoesNotArm(t *testing.T) {
	const notArming = " does not arm lanes: call `loomux check precommit --arm` there, or arm by hand with `loomux gate arm`\n"
	root := probationProject(t, "armed = []\n")
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	render := func() string {
		t.Helper()
		var b bytes.Buffer
		renderProbation(&b, root)
		return b.String()
	}
	for text, arms := range map[string]bool{
		"#!/bin/sh\nexec loomux check precommit\n":                    false,
		"#!/bin/sh\n# loomux check precommit --arm\nsh ci/gate.sh\n":  false,
		"#!/bin/sh\nloomux check precommit --armed\n":                 false,
		"#!/bin/sh\nloomux check precommit --arm\n":                   true,
		"#!/bin/sh\n\"/opt/loomux\" check precommit --root . --arm\n": true,
	} {
		if err := os.WriteFile(hook, []byte(text), 0o755); err != nil {
			t.Fatal(err)
		}
		out := render()
		// By its tail: git spells the directory its own way, and a runner's
		// TEMP is a short name.
		named := strings.Contains(out, "/.git/hooks/pre-commit"+notArming)
		if named == arms || strings.Contains(out, "no pre-commit hook") || (arms && strings.Contains(out, "[WARN]")) {
			t.Errorf("%q: %q", text, out)
		}
	}
	// A hook directory moved by core.hooksPath is where the hook is looked
	// for: the arming hook left in .git/hooks no longer runs.
	git(t, root, "config", "core.hooksPath", ".githooks")
	if out := render(); !strings.Contains(out, " [WARN] no pre-commit hook arms lanes: ") {
		t.Errorf("an empty hooks path: %q", out)
	}
	writeWorldFile(t, root, ".githooks/pre-commit", "#!/bin/sh\nexec loomux check precommit\n")
	if out := render(); !strings.Contains(out, "/.githooks/pre-commit"+notArming) {
		t.Errorf("a hook on the hooks path: %q", out)
	}
	// Outside a repository there is no hook to name.
	outside := project(t)
	writeWorldFile(t, outside, ".loomux/armed.toml", "armed = []\n")
	var b bytes.Buffer
	if renderProbation(&b, outside); !strings.Contains(b.String(), " [WARN] no pre-commit hook arms lanes: ") {
		t.Fatalf("no repository: %q", b.String())
	}
}
