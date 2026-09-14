package detect

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHooksPathReportsWhatGitHasSet(t *testing.T) {
	var asked []string
	run := func(dir string, argv ...string) (string, error) {
		asked = append(asked, dir+": "+strings.Join(argv, " "))
		return ".githooks\n", nil
	}
	got, err := HooksPath(run, "/project")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != ".githooks" {
		t.Fatalf("hooks path = %q, want %q", got, ".githooks")
	}
	want := "/project: git config --get core.hooksPath"
	if len(asked) != 1 || asked[0] != want {
		t.Fatalf("asked = %v, want [%q]", asked, want)
	}
}

// The common case: nothing is set, and that is an answer rather than a fault.
func TestHooksPathIsEmptyWhenUnset(t *testing.T) {
	got, err := HooksPath(func(string, ...string) (string, error) { return "", nil }, ".")
	if err != nil || got != "" {
		t.Fatalf("hooks path = %q, err = %v, want empty and nil", got, err)
	}
}

func TestHooksPathPassesGitsFailureOn(t *testing.T) {
	broken := errors.New("git not found")
	got, err := HooksPath(func(string, ...string) (string, error) { return "", broken }, ".")
	if !errors.Is(err, broken) {
		t.Fatalf("err = %v, want %v", err, broken)
	}
	if got != "" {
		t.Fatalf("hooks path = %q, want empty", got)
	}
}

func TestNeighbourWikiFindsTheSiblingRepository(t *testing.T) {
	parent := fstest.MapFS{
		"iam_backend/go.mod":         {Data: []byte("module x\n")},
		"iam_backend_wiki/.git/HEAD": {Data: []byte("ref: refs/heads/master\n")},
		"iam_backend_wiki/index.md":  {Data: []byte("# wiki\n")},
	}
	mode, wikiPath := NeighbourWiki(parent, "iam_backend")
	if mode != "neighbour_repo" || wikiPath != "iam_backend_wiki/" {
		t.Fatalf("got %q %q, want \"neighbour_repo\" \"iam_backend_wiki/\"", mode, wikiPath)
	}
}

// A folder of notes is not a repository, and the project must not find itself.
func TestNeighbourWikiDeclinesWhatIsNotARepository(t *testing.T) {
	cases := map[string]fstest.MapFS{
		// Named for the family of "project", so each case reaches the check
		// it is about instead of failing on the family first.
		"no .git":                 {"project_wiki/index.md": {Data: []byte("# notes\n")}},
		"wrong name":              {"docs/.git/HEAD": {Data: []byte("ref\n")}},
		"a file, not a directory": {"project_wiki": {Data: []byte("not a directory\n")}},
	}
	for name, parent := range cases {
		t.Run(name, func(t *testing.T) {
			if mode, wikiPath := NeighbourWiki(parent, "project"); mode != "" || wikiPath != "" {
				t.Fatalf("got %q %q, want empty", mode, wikiPath)
			}
		})
	}
}

func TestNeighbourWikiDoesNotFindTheProjectItself(t *testing.T) {
	parent := fstest.MapFS{"space_wiki/.git/HEAD": {Data: []byte("ref\n")}}
	if mode, _ := NeighbourWiki(parent, "space_wiki"); mode != "" {
		t.Fatalf("mode = %q, want empty", mode)
	}
}

func TestNeighbourWikiSaysNothingAboutAnUnreadableParent(t *testing.T) {
	if mode, wikiPath := NeighbourWiki(closedFS{}, "project"); mode != "" || wikiPath != "" {
		t.Fatalf("got %q %q, want empty", mode, wikiPath)
	}
}

// The iam repositories share one wiki: iam_backend, iam_frontend and
// iam_workers all keep theirs in iam_wiki. The convention is <family>_wiki,
// for the family itself and for every <family>_* sibling.
func TestNeighbourWikiServesTheWholeFamily(t *testing.T) {
	parent := fstest.MapFS{"iam_wiki/.git/HEAD": {Data: []byte("ref\n")}}
	for _, project := range []string{"iam", "iam_backend", "iam_frontend", "iam_workers"} {
		t.Run(project, func(t *testing.T) {
			mode, wikiPath := NeighbourWiki(parent, project)
			if mode != "neighbour_repo" || wikiPath != "iam_wiki/" {
				t.Fatalf("got %q %q, want \"neighbour_repo\" \"iam_wiki/\"", mode, wikiPath)
			}
		})
	}
}

// Measured on 2026-09-10: ultraloom, which only stands beside iam_wiki in the
// same parent directory, had that wiki recorded as its own. A shared parent
// directory is not a family, and neither is a shared first few letters.
func TestNeighbourWikiDoesNotTakeAnotherFamilysWiki(t *testing.T) {
	parent := fstest.MapFS{"iam_wiki/.git/HEAD": {Data: []byte("ref\n")}}
	for _, project := range []string{"ultraloom", "space", "iamx", "iamx_backend"} {
		t.Run(project, func(t *testing.T) {
			if mode, wikiPath := NeighbourWiki(parent, project); mode != "" || wikiPath != "" {
				t.Fatalf("got %q %q, want empty", mode, wikiPath)
			}
		})
	}
}
