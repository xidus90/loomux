// Package gdscript extracts the nodes and raw edges of one file of a Godot 4
// project on the shared tree-sitter core: a GDScript file (.gd), a scene or a
// resource (.tscn, .tres), or the project file (project.godot).
//
// One language for all four, because the resolver resolves a language
// against its own files alone, and a scene reaches the script it attaches,
// a script the scene it preloads, only inside one group.
//
// Raw, as in the other extractors: a base class, a call or a name a script
// uses names its target, and resolving it needs every file of the project,
// which is internal/code/resolve's job.
//
// Every script is a class. A script with `class_name X` gets a class node X
// over the whole file, and its functions and signals are X's members; a
// script without one gets no class node, and its functions hang on the file.
// Inner classes get nodes at every depth, qualified by the classes around
// them. var, const and enum get none; their text stays in the body of the
// class or the file.
//
// A definition the parser rebuilt around text it rejected gets no node, and
// nothing inside it counts, as in the Python extractor (header).
//
// The node types and fields read here, as gotreesitter builds them
// (treesitter.Parser), GDScript first:
//
//	source                the statements as children
//	class_name_statement  field name, on the keyword's line; field extends in `class_name X extends Y`
//	extends_statement     a type child (an identifier, or an attribute of identifiers) or a string child
//	function_definition   fields name, parameters, return_type, body; a ":" child closes the header
//	constructor_definition  `func _init`: no name field, otherwise as function_definition
//	class_definition     fields name, extends, body (a class_body); a ":" child closes the header
//	signal_statement      field name
//	const_statement       fields name, value; so is variable_statement
//	call                  the callee identifier first, field arguments
//	attribute             identifier and "." children; an attribute_call last for a member call
//	attribute_call        the member identifier first, field arguments
//	string                its text, quotes included
//
// and the resource grammar of scenes, resources and the project file:
//
//	resource   the sections as children, and properties ahead of the first
//	section    "[", its identifier, attribute children, "]", then property children
//	attribute  key identifier, "=", value
//	property   key path, "=", value
package gdscript

import (
	"path"
	"slices"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	gdgrammar "github.com/odvcencio/gotreesitter/grammars/gdscript"
	resgrammar "github.com/odvcencio/gotreesitter/grammars/godot_resource"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// version is this extractor's identity. Bump it by hand whenever the nodes or
// edges below change shape or meaning; the extract cache and the graph's meta
// key on it.
const version = "gdscript/1"

// langName is the language as the graph's meta lists it and as a Result carries it.
const langName = "gdscript"

// Language is the GDScript extractor as extract.Language describes one.
type Language struct{}

// Name is "gdscript".
func (Language) Name() string { return langName }

// Version is this package's version plus the tree-sitter runtime: a new
// runtime can build other trees from the same source.
func (Language) Version() string { return version + "@" + treesitter.Parser }

// Extensions are the files of a Godot project this package reads: scripts,
// the project file, resources and scenes.
func (Language) Extensions() []string { return []string{".gd", ".godot", ".tres", ".tscn"} }

// File is this package's File.
func (Language) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// File extracts one file. rel is its repo-relative, slash-separated path;
// source is its contents. A script is read as GDScript, anything else as a
// Godot resource.
func File(rel, source string) (extract.Result, error) {
	if path.Ext(rel) == ".gd" {
		return script(gdgrammar.Language(), rel, source)
	}
	return resource(resgrammar.Language(), rel, source)
}

// script extracts one GDScript file with the grammar handed in: the grammar
// builds a tree from any input, and only a missing grammar makes the parse
// fail.
func script(lang *gts.Language, rel, source string) (extract.Result, error) {
	doc, err := treesitter.Parse(lang, rel, []byte(source))
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()

	x := &extractor{doc: doc, fileID: model.NodeID(rel), referenced: map[[2]string]bool{}}
	x.script()
	return extract.Result{
		Path:     rel,
		Language: langName,
		Package:  x.className,
		Imports:  x.imports,
		// script has run every Symbol by now, as FileNode requires.
		Nodes:       append([]model.Node{doc.FileNode()}, x.nodes...),
		Edges:       slices.Concat(x.contains, x.importEdges, x.extends, x.calls, x.refs),
		ParseErrors: doc.ParseErrors(),
	}, nil
}

// extractor gathers one script's nodes and edges in source order.
type extractor struct {
	doc       *treesitter.Doc
	fileID    model.NodeID
	className string

	nodes       []model.Node
	contains    []extract.RawEdge
	extends     []extract.RawEdge
	calls       []extract.RawEdge
	refs        []extract.RawEdge
	importEdges []extract.RawEdge
	imports     []extract.Import
	referenced  map[[2]string]bool // source and name of every reference so far
}

// scope is what a definition or a call at one place belongs to.
type scope struct {
	source model.NodeID // the innermost definition with a node, or the script's class
	class  string       // the qualified class around it; "" at the top of a script without class_name
	fn     string       // the function it is in, for super(); "" outside one
}

// script reads the whole file: the class_name first, wherever it stands,
// because every definition of the script belongs to it, then the statements.
func (x *extractor) script() {
	root := x.doc.Root
	top := scope{source: x.fileID}
	if stmt := x.classNameStatement(root); stmt != nil {
		name := x.doc.Text(x.doc.Field(stmt, "name"))
		n := x.doc.Symbol(treesitter.Sym{
			Outer: root, Head: stmt.StartByte(), HeaderEnd: stmt.EndByte(),
			Name: name, Qualified: name, Kind: "class", Exported: exported(name),
		})
		x.add(n, x.fileID)
		x.className = name
		top = scope{source: n.ID, class: name}
		x.extendsOf(x.doc.Field(stmt, "extends"), n.ID)
	}
	x.body(root, top, true)
}

// classNameStatement is the script's class_name, or nil: the first one at
// the top whose name stands on the keyword's line and that holds no ERROR.
func (x *extractor) classNameStatement(root *gts.Node) *gts.Node {
	for _, stmt := range root.Children() {
		if x.doc.Type(stmt) != "class_name_statement" || treesitter.HoldsError(stmt) {
			continue
		}
		row := stmt.StartPoint().Row
		if x.doc.Field(stmt, "name").StartPoint().Row != row {
			continue
		}
		// The same quirk for the base: it too must stand on the keyword's line.
		if ext := x.doc.Field(stmt, "extends"); ext != nil && ext.EndPoint().Row != row {
			continue
		}
		return stmt
	}
	return nil
}

// body reads the statements of a script (top) or of a class body.
func (x *extractor) body(n *gts.Node, sc scope, top bool) {
	for _, stmt := range n.Children() {
		switch x.doc.Type(stmt) {
		case "class_name_statement":
			// Read by script, its extends with it.
		case "extends_statement":
			x.extendsOf(stmt, sc.source)
		case "function_definition", "constructor_definition":
			x.function(stmt, sc)
		case "class_definition":
			x.class(stmt, sc)
		case "signal_statement":
			x.signal(stmt, sc)
		default:
			if top {
				x.alias(stmt)
			}
			x.collect(stmt, sc)
		}
	}
}

// function makes the node of a func and collects what its header and body
// call and name.
func (x *extractor) function(def *gts.Node, sc scope) {
	end, ok := x.header(def)
	if !ok {
		return
	}
	name := "_init"
	if x.doc.Type(def) == "function_definition" {
		name = x.doc.Text(x.doc.Field(def, "name"))
	}
	kind := model.Kind("method")
	if sc.class == "" {
		kind = "function"
	}
	n := x.symbol(def, end, name, kind, sc)
	inner := scope{source: n.ID, class: sc.class, fn: name}
	for _, c := range def.Children() {
		x.collect(c, inner)
	}
}

// class makes the node of an inner class, reads its base and its body.
func (x *extractor) class(def *gts.Node, sc scope) {
	end, ok := x.header(def)
	if !ok {
		return
	}
	name := x.doc.Text(x.doc.Field(def, "name"))
	n := x.symbol(def, end, name, "class", sc)
	x.extendsOf(x.doc.Field(def, "extends"), n.ID)
	x.body(x.doc.Field(def, "body"), scope{source: n.ID, class: qualify(sc.class, name)}, false)
}

// signal makes the node of a signal declaration; its signature is the whole
// statement.
func (x *extractor) signal(stmt *gts.Node, sc scope) {
	if treesitter.HoldsError(stmt) {
		return
	}
	n := x.symbol(stmt, stmt.EndByte(), x.doc.Text(x.doc.Field(stmt, "name")), "signal", sc)
	// The signal's own name is no identifier node in this grammar, so walking
	// the statement names only its parameters and their types.
	inner := scope{source: n.ID, class: sc.class}
	for _, c := range stmt.Children() {
		x.collect(c, inner)
	}
}

// symbol mints the node of one definition, whose header ends at end, and the
// contains edge from the scope's source to it.
func (x *extractor) symbol(def *gts.Node, end uint32, name string, kind model.Kind, sc scope) model.Node {
	n := x.doc.Symbol(treesitter.Sym{
		Outer: def, Head: def.StartByte(), HeaderEnd: end,
		Name: name, Qualified: qualify(sc.class, name), Kind: kind,
		Owner: sc.class, Exported: exported(name),
	})
	x.add(n, sc.source)
	return n
}

// add records a node and the contains edge from parent to it.
func (x *extractor) add(n model.Node, parent model.NodeID) {
	x.nodes = append(x.nodes, n)
	x.contains = append(x.contains, extract.RawEdge{
		Source: parent, Relation: model.RelationContains, TargetID: n.ID, File: x.doc.Rel,
	})
}

// header is where the signature of def ends, the end of its first direct ":"
// child, and false when the parser rebuilt the definition around text it
// rejected: an ERROR in or under a child ahead of that colon, or no colon.
// `func broken(:` swallows the next definition into its parameters, and its
// span would claim lines of another; no node beats a wrong one.
func (x *extractor) header(def *gts.Node) (uint32, bool) {
	for _, c := range def.Children() {
		if treesitter.HoldsError(c) {
			break
		}
		if x.doc.Type(c) == ":" {
			return c.EndByte(), true
		}
	}
	return 0, false
}

// qualify is name inside the class owner: "Outer.Inner", or name alone at
// the top of a script without class_name.
func qualify(owner, name string) string {
	if owner == "" {
		return name
	}
	return owner + "." + name
}

// exported is whether a name is public by GDScript's convention: it does not
// start with an underscore. The engine's callbacks (_ready, _process) follow
// the same convention and count as not exported.
func exported(name string) bool { return !strings.HasPrefix(name, "_") }
