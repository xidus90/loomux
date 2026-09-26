package python

import (
	"strings"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// importStatement records `import a.b` and `import a.b as c`: one import and
// one edge per module the statement names.
func (x *extractor) importStatement(n *gts.Node) {
	for i := range n.ChildCount() {
		if n.FieldNameForChild(i, x.doc.Lang) != "name" {
			continue
		}
		path, alias := x.named(n.Child(i))
		x.imports = append(x.imports, extract.Import{Path: path, Alias: alias})
		x.importEdge(path)
	}
}

// importFrom records `from X import n, m as k` and `from X import *`.
//
// X keeps its leading dots; resolving them needs the file's package, and that
// is the resolver's. The statement gets one edge to X, and every imported
// name one more to X.name: the name may be a submodule rather than a
// definition in X, and only the resolver can tell which of the two is a
// file. A wildcard names nothing that could be one.
func (x *extractor) importFrom(n *gts.Node) {
	module := x.dotted(x.doc.Field(n, "module_name"))
	x.importEdge(module)
	for i := range n.ChildCount() {
		c := n.Child(i)
		switch {
		case x.doc.Type(c) == "wildcard_import":
			x.imports = append(x.imports, extract.Import{Path: module, Name: "*"})
		case n.FieldNameForChild(i, x.doc.Lang) == "name":
			name, alias := x.named(c)
			x.imports = append(x.imports, extract.Import{Path: module, Name: name, Alias: alias})
			x.importEdge(join(module, name))
		}
	}
}

// named reads one name of an import: a dotted_name alone, or an
// aliased_import with its alias.
func (x *extractor) named(n *gts.Node) (string, string) {
	if x.doc.Type(n) == "aliased_import" {
		return x.dotted(x.doc.Field(n, "name")), x.doc.Text(x.doc.Field(n, "alias"))
	}
	return x.dotted(n), ""
}

// dotted is a dotted_name or a relative_import as Python reads it: its dots
// and names, read from the tokens and not the text, so a line continuation
// or a space between them stays out.
func (x *extractor) dotted(n *gts.Node) string {
	var b strings.Builder
	treesitter.Walk(n, func(c *gts.Node) bool {
		if t := x.doc.Type(c); t == "." || t == "identifier" {
			b.WriteString(x.doc.Text(c))
		}
		return true
	})
	return b.String()
}

// join is the module a name of `from module import name` would be, were it a
// submodule: "pkg.mod" and "t" make "pkg.mod.t". A module of dots alone takes
// the name without a separator: "." and "sib" make ".sib".
func join(module, name string) string {
	if strings.HasSuffix(module, ".") {
		return module + name
	}
	return module + "." + name
}

// importEdge is one imports edge from the file.
func (x *extractor) importEdge(spec string) {
	x.importEdges = append(x.importEdges, extract.RawEdge{
		Source: x.fileID, Relation: model.RelationImports, Specifier: spec, File: x.doc.Rel,
	})
}
