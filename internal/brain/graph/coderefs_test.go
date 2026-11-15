package graph_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/brain/graph"
)

func TestCandidateRefs_BacktickSpansAreRootRelative(t *testing.T) {
	got := graph.CandidateRefs("topics/x.md", "see `internal/hooks/guard.go:67` and `internal/web`")
	want := []graph.CodeRef{
		{From: "topics/x.md", Path: "internal/hooks/guard.go", Line: 67},
		{From: "topics/x.md", Path: "internal/web"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestCandidateRefs_LineRange(t *testing.T) {
	got := graph.CandidateRefs("d.md", "`a/b.go:3-9`")
	want := []graph.CodeRef{{From: "d.md", Path: "a/b.go", Line: 3, End: 9}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestCandidateRefs_TrailingSlashIsDropped(t *testing.T) {
	got := graph.CandidateRefs("d.md", "`internal/web/`")
	want := []graph.CodeRef{{From: "d.md", Path: "internal/web"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestCandidateRefs_FencedBlocksDoNotCount(t *testing.T) {
	text := "```\n`a/b.go`\n```\n~~~go\n`c/d.go`\n~~~\n`e/f.go`\n```\n`g/h.go`\n"
	got := graph.CandidateRefs("d.md", text)
	want := []graph.CodeRef{{From: "d.md", Path: "e/f.go"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestCandidateRefs_NeedsASlash(t *testing.T) {
	if got := graph.CandidateRefs("d.md", "`go.mod` `docs` `.`"); len(got) != 0 {
		t.Fatalf("got %+v, want none", got)
	}
}

func TestCandidateRefs_RefusesEscapesAndPrivateDirs(t *testing.T) {
	text := "`../x/y` `/abs/p` `.git/config` `.loomux/state/x` `a/../../b`"
	if got := graph.CandidateRefs("d.md", text); len(got) != 0 {
		t.Fatalf("got %+v, want none", got)
	}
}

func TestCandidateRefs_MarkdownLinksResolveFromTheDocument(t *testing.T) {
	doc := "docs/wiki/topics/x.md"
	text := "[g](../../../internal/a.go#L3) [up](../../../../out.go) [w](http://example.com/a/b) " +
		"[m](mailto:a@b.c) [f](#frag) [root](../../..) [abs](/x/y.go) [gd](../../../.git/config) [ok](sibling/y.md)"
	got := graph.CandidateRefs(doc, text)
	want := []graph.CodeRef{
		{From: doc, Path: "internal/a.go"},
		{From: doc, Path: "docs/wiki/topics/sibling/y.md"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestCandidateRefs_KeepsTextOrderWithoutDedup(t *testing.T) {
	got := graph.CandidateRefs("d.md", "`a/b` [x](c/d) `a/b`")
	want := []graph.CodeRef{
		{From: "d.md", Path: "a/b"},
		{From: "d.md", Path: "c/d"},
		{From: "d.md", Path: "a/b"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestCandidateRefs_ProseInBackticksIsNoPath(t *testing.T) {
	if got := graph.CandidateRefs("d.md", "`loomux graph build` `a b/c`"); len(got) != 0 {
		t.Fatalf("got %+v, want none", got)
	}
}

func TestExistsExact(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "web", "a.go"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]bool{
		"internal/web/a.go":       true,
		"internal/web":            true,
		"Internal/web":            false,
		"internal/web/missing.go": false,
		"internal/web/a.go/x":     false,
		"":                        false,
	} {
		if got := graph.ExistsExact(root, rel); got != want {
			t.Errorf("ExistsExact(%q) = %v, want %v", rel, got, want)
		}
	}
}
