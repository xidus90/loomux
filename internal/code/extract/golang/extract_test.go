package golang_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/model"
)

// fixture reads a .go.txt template. The templates do not end in .go because
// the gate's first lane, `gofmt -l cmd internal`, walks into testdata and
// would report a deliberately crooked one like any other file.
func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name+".go.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func nodeByID(r extract.Result, id model.NodeID) *model.Node {
	for i := range r.Nodes {
		if r.Nodes[i].ID == id {
			return &r.Nodes[i]
		}
	}
	return nil
}

func TestFileEmitsOneNodePerDefinitionShape(t *testing.T) {
	r, err := golang.File("pkg/shapes.go", fixture(t, "shapes"))
	if err != nil {
		t.Fatal(err)
	}

	want := map[model.NodeID]model.Kind{
		"pkg/shapes.go":            model.KindFile,
		"pkg/shapes.go#Exported":   "function",
		"pkg/shapes.go#unexported": "function",
		"pkg/shapes.go#useFmt":     "function",
		"pkg/shapes.go#User":       "struct",
		"pkg/shapes.go#Reader":     "interface",
		"pkg/shapes.go#ID":         "type",
		"pkg/shapes.go#First":      "type",
		"pkg/shapes.go#Second":     "type",
		"pkg/shapes.go#User.Save":  "method",
		"pkg/shapes.go#User.name":  "method",
	}
	if len(r.Nodes) != len(want) {
		var got []model.NodeID
		for _, n := range r.Nodes {
			got = append(got, n.ID)
		}
		t.Fatalf("got %d nodes %v, want %d", len(r.Nodes), got, len(want))
	}
	for id, kind := range want {
		n := nodeByID(r, id)
		if n == nil {
			t.Errorf("node %q missing", id)
			continue
		}
		if n.Kind != kind {
			t.Errorf("node %q has kind %q, want %q", id, n.Kind, kind)
		}
	}
	// A grouped `type ( ... )` yields one node per spec, never one for the
	// declaration: otherwise a change to one type would change the hash of
	// every other in the group and `check` would report them all.
	if nodeByID(r, "pkg/shapes.go#First") == nil || nodeByID(r, "pkg/shapes.go#Second") == nil {
		t.Error("a grouped type declaration must yield one node per name")
	}
	// No constants, no variables. The original emits none for Go, and a const
	// block reaches a query through the file node's residual text instead.
	for _, n := range r.Nodes {
		if n.Kind == "const" || n.Kind == "var" {
			t.Errorf("node %q: this extractor emits no %s nodes", n.ID, n.Kind)
		}
	}
}

func TestLanguageDescribesGo(t *testing.T) {
	var l extract.Language = golang.Language{}
	if l.Name() != "go" {
		t.Errorf("Name() = %q, want go", l.Name())
	}
	if l.Version() != golang.Version {
		t.Errorf("Version() = %q, want %q", l.Version(), golang.Version)
	}
	if !reflect.DeepEqual(l.Extensions(), []string{".go"}) {
		t.Errorf("Extensions() = %v, want [.go]", l.Extensions())
	}
	r, err := l.File("pkg/a.go", "package pkg\n\nfunc F() {}\n")
	if err != nil {
		t.Fatal(err)
	}
	if r.Language != "go" || r.Path != "pkg/a.go" || r.Package != "pkg" || nodeByID(r, "pkg/a.go#F") == nil {
		t.Errorf("File through the interface = %+v, want the Go extraction stamped go", r)
	}
}

func TestFileStampsItsLanguage(t *testing.T) {
	r, err := golang.File("p.go", "package p\n")
	if err != nil {
		t.Fatal(err)
	}
	// The resolver groups files by this field; a Go file without it would be
	// resolved by no rule at all.
	if r.Language != "go" {
		t.Errorf("Language = %q, want go", r.Language)
	}
}

func TestFileQualifiesAMethodByItsReceiverAndKeepsTheBareName(t *testing.T) {
	r, err := golang.File("pkg/shapes.go", fixture(t, "shapes"))
	if err != nil {
		t.Fatal(err)
	}

	save := nodeByID(r, "pkg/shapes.go#User.Save")
	if save == nil {
		t.Fatal("method node missing")
	}
	// The id qualifies, because Go methods do not nest and the id would collide
	// per receiver. The name stays bare so a call `u.Save()` can hit it.
	if save.Name != "Save" {
		t.Errorf("Name = %q, want the bare name", save.Name)
	}
	// The pointer is unwrapped: `func (u *User)` owns `User`.
	if save.Owner != "User" {
		t.Errorf("Owner = %q, want User", save.Owner)
	}
	if !save.Exported {
		t.Error("Save must be exported: its own name starts uppercase")
	}
	lower := nodeByID(r, "pkg/shapes.go#User.name")
	if lower == nil || lower.Exported {
		t.Error("name must be unexported: the own name is the part after the dot")
	}
}

func TestFileCutsASignatureAtTheHeader(t *testing.T) {
	r, err := golang.File("pkg/shapes.go", fixture(t, "shapes"))
	if err != nil {
		t.Fatal(err)
	}

	cases := map[model.NodeID]string{
		"pkg/shapes.go#Exported":  "func Exported()",
		"pkg/shapes.go#User.Save": "func (u *User) Save() error",
		// The original's own code would cut a struct down to the bare name here: its
		// type_spec starts at the name and the header ends at the `struct`
		// keyword. Its comment says "where the body opens", which is what this
		// port writes instead -- see 5.2.1 of the spec.
		"pkg/shapes.go#User":   "type User struct",
		"pkg/shapes.go#Reader": "type Reader interface",
		"pkg/shapes.go#ID":     "type ID int",
		"pkg/shapes.go#First":  "type First int",
	}
	for id, want := range cases {
		n := nodeByID(r, id)
		if n == nil {
			t.Errorf("node %q missing", id)
			continue
		}
		if n.Signature != want {
			t.Errorf("node %q signature = %q, want %q", id, n.Signature, want)
		}
	}
}

func TestFileLeavesTheDocCommentOutOfTheBodyHash(t *testing.T) {
	const withDoc = "package p\n\n// Doc says something.\nfunc F() { return }\n"
	const without = "package p\n\nfunc F() { return }\n"

	a, err := golang.File("p.go", withDoc)
	if err != nil {
		t.Fatal(err)
	}
	b, err := golang.File("p.go", without)
	if err != nil {
		t.Fatal(err)
	}
	// go/ast's Pos() starts at `func`, so the comment is outside the range --
	// the same as tree-sitter, where a comment is a sibling of the
	// declaration. Rewording a doc comment must not make `check` report drift.
	if nodeByID(a, "p.go#F").BodyHash != nodeByID(b, "p.go#F").BodyHash {
		t.Error("a doc comment must not change the body hash")
	}
	// The file node hashes the whole file, so there it must differ.
	if nodeByID(a, "p.go").BodyHash == nodeByID(b, "p.go").BodyHash {
		t.Error("the file node hashes the whole file, comment included")
	}
}

func TestFileNodeCarriesPathAsItsID(t *testing.T) {
	r, err := golang.File("pkg/shapes.go", fixture(t, "shapes"))
	if err != nil {
		t.Fatal(err)
	}

	f := nodeByID(r, "pkg/shapes.go")
	if f == nil {
		t.Fatal("file node missing")
	}
	// The id is the path itself, without a '#'. Validate's rule that an edge
	// source must be a node depends on it: an import edge's source is the file.
	if f.Name != "shapes.go" || f.Kind != model.KindFile {
		t.Errorf("file node = %+v, want name shapes.go and kind file", f)
	}
	if f.Signature != "" || !f.Exported {
		t.Errorf("file node signature = %q, exported = %v; want empty and true", f.Signature, f.Exported)
	}
	if !strings.HasPrefix(string(f.Span), "L1-L") {
		t.Errorf("file span = %q, want L1-L<lines>", f.Span)
	}
}

func TestFileNodeCountsTheLineAfterATrailingNewline(t *testing.T) {
	// Five newlines, so six lines: the Go graph has always counted the empty
	// line after the last newline in the file node's span, where F's own span
	// ends on line 5. The body text is what F's line leaves over.
	const src = "package p\n\nimport \"strings\"\n\nfunc F() { _ = strings.TrimSpace(\"x\") }\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	want := model.Node{
		ID: "p.go", Name: "p.go", Kind: model.KindFile, Path: "p.go",
		Span: "L1-L6", Exported: true, BodyHash: extract.Hash(src),
		BodyText: `package p import "strings"`,
	}
	if got := nodeByID(r, "p.go"); got == nil || !reflect.DeepEqual(*got, want) {
		t.Errorf("file node =\n %+v\nwant\n %+v", got, want)
	}
	if got := nodeByID(r, "p.go#F"); got == nil || got.Span != "L5-L5" {
		t.Errorf("F = %+v, want span L5-L5", got)
	}
}

func TestFileMintsAnOrdinalForACollidingID(t *testing.T) {
	// Two definitions of one name in one file: invalid Go, but the extractor
	// must not lose one. It parses far enough for the walk.
	const src = "package p\n\nfunc F() {}\n\nfunc F() {}\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if nodeByID(r, "p.go#F") == nil || nodeByID(r, "p.go#F~2") == nil {
		var got []model.NodeID
		for _, n := range r.Nodes {
			got = append(got, n.ID)
		}
		t.Fatalf("got %v, want p.go#F and p.go#F~2", got)
	}
}

func TestFileMintsPastASourceNameEndingInAnOrdinal(t *testing.T) {
	// The loop, not a single `~2` guess, is what makes this collision-proof: a
	// source name may itself end in ~N-looking text after qualification.
	const src = "package p\n\nfunc F() {}\n\nfunc F() {}\n\nfunc F() {}\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if nodeByID(r, "p.go#F~3") == nil {
		t.Error("a third collision must mint ~3")
	}
}

func TestFileRefusesSourceItCannotParse(t *testing.T) {
	_, err := golang.File("p.go", "package ???\n")
	if err == nil {
		t.Fatal("got nil, want a parse error")
	}
}

func TestFileTrimsBodyTextAndCapsIt(t *testing.T) {
	long := "package p\n\nfunc F() {\n" + strings.Repeat("\tprintln(\"x\")\n", 2000) + "}\n"
	r, err := golang.File("p.go", long)
	if err != nil {
		t.Fatal(err)
	}
	n := nodeByID(r, "p.go#F")
	if len(n.BodyText) > 5000 {
		t.Errorf("body text is %d chars, want at most 5000", len(n.BodyText))
	}
	if strings.Contains(n.BodyText, "\n") || strings.Contains(n.BodyText, "\t") {
		t.Error("body text must be whitespace-collapsed")
	}
}

func TestFileNodeBodyTextIsTheResidualOutsideEverySymbol(t *testing.T) {
	const src = "package p\n\nimport \"strings\"\n\nconst marker = \"needle\"\n\nfunc F() { _ = strings.TrimSpace(\"inside\") }\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	f := nodeByID(r, "p.go")
	// The residual is what no symbol span covers -- the import header, package
	// constants -- so a file is findable by a word that lives in no function,
	// without storing any symbol body twice.
	if !strings.Contains(f.BodyText, "needle") {
		t.Errorf("file residual = %q, want the package constant in it", f.BodyText)
	}
	if strings.Contains(f.BodyText, "inside") {
		t.Errorf("file residual = %q, must not repeat a symbol body", f.BodyText)
	}
}

func TestFileLeavesAMethodUnqualifiedWhenItsReceiverTypeCannotBeRead(t *testing.T) {
	// go/parser accepts a parenthesized receiver type even though the Go spec
	// does not allow one; receiverType only unwraps *ast.StarExpr and the two
	// generic index forms, so it reads nothing here and returns "".
	const src = "package p\n\ntype Cache struct{}\n\nfunc (c (Cache)) Get() {}\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if nodeByID(r, "p.go#Get") == nil {
		t.Errorf("a method whose receiver type this extractor cannot read must keep its bare name (owner stays empty); nodes: %+v", r.Nodes)
	}
}

func TestFileCutsASignatureAtTheHeaderForABodylessDeclaration(t *testing.T) {
	// A body-less function declaration is valid Go (an assembly stub, or a
	// //go:linkname target); its header IS the whole declaration.
	const src = "package p\n\nfunc Stub(x int)\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	n := nodeByID(r, "p.go#Stub")
	if n == nil {
		t.Fatal("node missing")
	}
	if n.Signature != "func Stub(x int)" {
		t.Errorf("Signature = %q, want the whole declaration: a body-less function has no body to cut at", n.Signature)
	}
}

func TestFileHandlesASymbolEndingAtTheVeryLastByte(t *testing.T) {
	// No trailing newline: the last declaration's end offset equals len(source).
	const src = "package p\n\nfunc Last() {}"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	n := nodeByID(r, "p.go#Last")
	if n == nil {
		t.Fatal("node missing")
	}
	if n.BodyText != "func Last() {}" {
		t.Errorf("BodyText = %q, want the full declaration even though it ends at the file's last byte", n.BodyText)
	}
}
