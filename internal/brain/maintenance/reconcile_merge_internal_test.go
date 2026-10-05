package maintenance

// The parity of the merge trigger: the case file and its package, against the
// bytes the reference wrote from the very same repository.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/gitenv"
)

// mergeGoldenPage is the page testdata/merge-package.golden.md was rendered
// from, byte for byte: CRLF in the frontmatter, a lone carriage return inside
// a paragraph, an indented paragraph and one of nothing but whitespace.
const mergeGoldenPage = "---\r\ntype: note\r\nrealization: planned\r\n---\r\n\r\n" +
	"  eingerueckt, mit einem einsamen \r darin\n\n   \t  \n\nzweiter Absatz\n"

// mergeGoldenCase is the case both merge goldens were rendered from: what
// `landCase` writes for this trigger and nothing else -- no sources at all,
// `due` out of the state vocabulary, and the `change` both producers carry.
func mergeGoldenCase() Case {
	return Case{
		ID: "a-2026-09-20-abcd", Area: "project/a",
		Target: "docs/wiki/a.md", TargetHash: "sha256:cc",
		State: "due", Trigger: "merge", Weight: "change",
		Created: time.Date(2026, 9, 20, 8, 0, 0, 0, time.FixedZone("", 2*3600)),
	}
}

// mergeGoldenRepository is the repository the reference's oracle built, commit
// for commit: a commit before the range, then two inside it. The oracle's
// script is in the archive release archive/parity-recordings.
//
// The three are chosen for what they decide. `Zeta.txt` beside `alpha.txt`
// pins that the path sort folds no case -- `Z` is below `a` in code points and
// above it in every case-insensitive order. One subject carries a fenced run,
// which pins that the package's fence grows only around a fence at the start
// of a line. One carries an umlaut, which pins the encoding on the way through
// git.
//
// Its own git helpers rather than the ones beside the world: those live in the
// external test package, which this file cannot see, and Go offers no third
// place for both.
func mergeGoldenRepository(t *testing.T) (string, string, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	root := t.TempDir()
	run := func(arguments ...string) string {
		t.Helper()
		command := exec.Command("git", arguments...)
		command.Dir = root
		command.Env = gitenv.Environ()
		out, err := command.Output()
		if err != nil {
			t.Fatalf("git %v: %v", arguments, err)
		}
		return string(out)
	}
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	commit := func(name, text, subject string) {
		t.Helper()
		write(name, text)
		run("add", "--all")
		run("-c", "user.name=Test", "-c", "user.email=test@example.invalid",
			"-c", "commit.gpgsign=false", "commit", "-q", "-m", subject)
	}
	run("init", "-q")
	// The same two settings the world helper explains: git for Windows sets
	// `core.autocrlf` globally, and with it on a fixture would arrive in the
	// object store already folded.
	run("config", "core.autocrlf", "false")
	write(".gitattributes", "* -text\n")
	commit("base.txt", "base\n", "the commit before the range")
	first := run("rev-parse", "HEAD")
	commit("Zeta.txt", "zeta\n", "füge Zeta hinzu")
	commit("alpha.txt", "alpha\n", "fix ```code``` fences")
	last := run("rev-parse", "HEAD")
	return root, strings.TrimSpace(first), strings.TrimSpace(last)
}

// The whole package of a merge case at once, against a file the reference
// wrote from the same three commits.
//
// This is what the source package's golden cannot give: the two evidence
// blocks in front of the page -- their headings, the sorted names, the
// unsorted subjects git reports newest first -- and that they carry the `D`
// kind rather than one of their own.
func TestTheMergePackageMatchesThePythonPackage(t *testing.T) {
	repo, first, last := mergeGoldenRepository(t)
	evidence, err := mergeEvidence(MergeEvent{Repo: repo, First: first, Last: last})
	if err != nil {
		t.Fatalf("mergeEvidence: %v", err)
	}
	page := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(page, []byte(mergeGoldenPage), 0o644); err != nil {
		t.Fatalf("WriteFile page: %v", err)
	}
	segments, err := segmentsOf(nil, page, evidence)
	if err != nil {
		t.Fatalf("segmentsOf: %v", err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "merge-package.golden.md"))
	if err != nil {
		t.Fatalf("ReadFile golden: %v", err)
	}
	if got := RenderPackage(mergeGoldenCase(), segments); got != string(want) {
		t.Fatalf("the merge package differs from the Python rendering:\ngot:\n%q\nwant:\n%q",
			got, string(want))
	}
}

// The case file of a merge, against one `write_case` wrote. A merge case is
// the one shape whose `sources` is empty, and the bytes are an interface in
// both directions: a file that differed in one character would be rewritten,
// and reported as changed, on every pass across the two.
func TestTheMergeCaseFileMatchesThePythonCase(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("testdata", "merge-case.golden.toml"))
	if err != nil {
		t.Fatalf("ReadFile golden: %v", err)
	}
	if got := renderCase(mergeGoldenCase()); got != string(want) {
		t.Fatalf("the merge case differs from the Python rendering:\ngot:\n%q\nwant:\n%q",
			got, string(want))
	}
}
