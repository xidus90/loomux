package extract_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestMintIDAppendsLowestFreeOrdinal(t *testing.T) {
	minted := map[string]bool{}
	for i, want := range []string{"a.go#F", "a.go#F~2", "a.go#F~3"} {
		if got := extract.MintID("a.go#F", minted); got != want {
			t.Errorf("mint %d = %q, want %q", i+1, got, want)
		}
	}

	// A source name that itself ends in an ordinal takes that id first; the
	// bare name after it has to skip past it to the lowest ordinal still free.
	minted = map[string]bool{}
	extract.MintID("a.py#g", minted)
	if got := extract.MintID("a.py#g~2", minted); got != "a.py#g~2" {
		t.Errorf("a source name ending in ~2 = %q, want it unchanged", got)
	}
	if got := extract.MintID("a.py#g", minted); got != "a.py#g~3" {
		t.Errorf("the next bare g = %q, want a.py#g~3: ~2 is taken", got)
	}

	// Past nine the ordinal has two digits.
	minted = map[string]bool{}
	var last string
	for range 12 {
		last = extract.MintID("a.go#H", minted)
	}
	if last != "a.go#H~12" {
		t.Errorf("twelfth mint = %q, want a.go#H~12", last)
	}
}

func TestCollapse(t *testing.T) {
	if got, want := extract.Collapse("  func F()\t{\r\n\treturn\n}\n"), "func F() { return }"; got != want {
		t.Errorf("Collapse = %q, want %q: every whitespace run is one space, the ends trimmed", got, want)
	}
	if got := extract.Collapse(" \n\t "); got != "" {
		t.Errorf("Collapse(whitespace) = %q, want empty", got)
	}
	long := strings.Repeat("ab ", extract.MaxBodyChars)
	got := extract.Collapse(long)
	if len(got) != extract.MaxBodyChars {
		t.Errorf("Collapse(long) has %d bytes, want the cap %d", len(got), extract.MaxBodyChars)
	}
	if !strings.HasPrefix(long, got) {
		t.Error("Collapse(long) must keep the start of the text")
	}
	if extract.MaxBodyChars != 5000 {
		t.Errorf("MaxBodyChars = %d, want Graft's 5000", extract.MaxBodyChars)
	}
}

func TestFileNode(t *testing.T) {
	const src = "package p\n\nconst marker = 1\n\nfunc F() {\n\treturn\n}\n"
	want := model.Node{
		ID: "pkg/p.go", Name: "p.go", Kind: model.KindFile, Path: "pkg/p.go",
		// Seven lines and a trailing newline: the file node counts the empty
		// line after the last newline, where a symbol span would stop at 7.
		Span: "L1-L8", Exported: true, BodyHash: extract.Hash(src),
		// Lines 5 to 7 belong to F; the package clause and the constant are
		// what is left.
		BodyText: "package p const marker = 1",
	}
	got := extract.FileNode("pkg/p.go", src, map[int]bool{5: true, 6: true, 7: true})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FileNode =\n %+v\nwant\n %+v", got, want)
	}

	// No trailing newline, nothing covered: every line is residual.
	got = extract.FileNode("a.py", "x = 1\ny = 2", nil)
	if got.Span != "L1-L2" || got.BodyText != "x = 1 y = 2" {
		t.Errorf("FileNode(no newline, nothing covered) span %q, body %q; want L1-L2 and the whole text", got.Span, got.BodyText)
	}
}

func TestHash(t *testing.T) {
	// sha256 as full hex: the two vectors of FIPS 180-2 anyone can check.
	for text, want := range map[string]string{
		"":    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"abc": "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
	} {
		if got := extract.Hash(text); got != want {
			t.Errorf("Hash(%q) = %s, want %s", text, got, want)
		}
	}
}

// The field names are part of a format once a result is written to disk, so
// a rename must show up here and not in a cache that no longer reads back.
func TestResultJSONShape(t *testing.T) {
	cases := []struct {
		name string
		r    extract.Result
		want string
	}{
		{
			"empty fields left out",
			extract.Result{Path: "a.go", Language: "go", Nodes: []model.Node{}},
			`{"path":"a.go","language":"go","nodes":[]}`,
		},
		{
			"every field set",
			extract.Result{
				Path: "pkg/a.py", Language: "python", Package: "pkg",
				Imports: []extract.Import{{Alias: "np", Path: "numpy", Name: "array"}},
				Nodes:   []model.Node{},
				Edges: []extract.RawEdge{{
					Source: "pkg/a.py#f", Relation: model.RelationCalls, TargetID: "pkg/a.py#g",
					Name: "g", Owner: "C", Receiver: "self", Specifier: "numpy", File: "pkg/a.py",
				}},
				ParseErrors: 2,
			},
			`{"path":"pkg/a.py","language":"python","package":"pkg",` +
				`"imports":[{"alias":"np","path":"numpy","name":"array"}],"nodes":[],` +
				`"edges":[{"source":"pkg/a.py#f","relation":"calls","target_id":"pkg/a.py#g",` +
				`"name":"g","owner":"C","receiver":"self","specifier":"numpy","file":"pkg/a.py"}],` +
				`"parse_errors":2}`,
		},
		{
			"an edge and an import with only their required fields",
			extract.Result{
				Path: "a.go", Language: "go", Nodes: []model.Node{},
				Imports: []extract.Import{{Path: "fmt"}},
				Edges:   []extract.RawEdge{{Source: "a.go", Relation: model.RelationImports, File: "a.go"}},
			},
			`{"path":"a.go","language":"go","imports":[{"path":"fmt"}],"nodes":[],` +
				`"edges":[{"source":"a.go","relation":"imports","file":"a.go"}]}`,
		},
	}
	for _, c := range cases {
		b, err := json.Marshal(c.r)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if string(b) != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, b, c.want)
		}
	}
}
