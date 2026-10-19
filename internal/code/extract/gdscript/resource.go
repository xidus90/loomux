package gdscript

import (
	"path"
	"strings"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// resource extracts a scene, a resource or the project file: its file node,
// an import of every external resource it names, and for project.godot the
// autoloads and the main scene. Every autoload entry gets its imports edge,
// but only a singleton (a value starting with "*") is bound as an alias:
// Godot gives only those a global name. An external resource is named by its
// path, or by its uid where it has none.
//
// A scene also says what its nodes run. Each node's script, or the scene it
// instances, is known by the node's path below the root ("." is the root,
// then "A", "A/B"); the root's is the alias "." of the scene, which no
// GDScript name can be. A [connection] references the handler in the target
// node's script and the signal in the source node's, each a reference with
// the script (or the instanced scene) as Specifier and the member as Name.
//
// The uid in a scene's or resource's header is its Package, for the
// resolver to read uid:// paths with. External resource ids are integers in
// format=2 (Godot 3) and strings in format=3 (Godot 4); both are read as
// written.
func resource(lang *gts.Language, rel, source string) (extract.Result, error) {
	doc, err := treesitter.Parse(lang, rel, []byte(source))
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()

	r := extract.Result{
		Path: rel, Language: langName,
		Nodes: []model.Node{doc.FileNode()}, ParseErrors: doc.ParseErrors(),
	}
	edge := func(spec string, line int) {
		r.Edges = append(r.Edges, extract.RawEdge{
			Source: model.NodeID(rel), Relation: model.RelationImports, Specifier: spec, File: rel, Line: line,
		})
	}
	project := path.Base(rel) == "project.godot"
	ext := map[string]string{}  // external resource id -> path or uid
	runs := map[string]string{} // node path -> its script, or the scene it instances
	var links []*gts.Node       // the [connection] sections, read once every node is known
	for _, s := range doc.Root.Children() {
		if doc.Type(s) != "section" {
			continue
		}
		attrs := pairs(doc, s, "attribute")
		switch doc.Text(s.Child(1)) {
		case "gd_scene", "gd_resource":
			r.Package = attrs["uid"].val
		case "ext_resource":
			spec := attrs["path"]
			if spec.val == "" {
				spec = attrs["uid"]
			}
			if spec.val != "" {
				edge(spec.val, spec.line)
				ext[idOf(doc, s)] = spec.val
			}
		case "node":
			at := nodePath(attrs)
			if spec, ok := ext[extID(doc, s, "attribute", "instance")]; ok {
				runs[at] = spec
			}
			if spec, ok := ext[extID(doc, s, "property", "script")]; ok {
				runs[at] = spec
			}
		case "connection":
			links = append(links, s)
		case "autoload":
			for _, e := range ordered(doc, s, "property") {
				if project {
					p, singleton := strings.CutPrefix(e.val, "*")
					if singleton {
						r.Imports = append(r.Imports, extract.Import{Alias: e.key, Path: p})
					}
					edge(p, e.line)
				}
			}
		case "application":
			if p, ok := pairs(doc, s, "property")["run/main_scene"]; ok && project {
				edge(p.val, p.line)
			}
		}
	}
	if root, ok := runs["."]; ok {
		r.Imports = append(r.Imports, extract.Import{Alias: ".", Path: root})
	}
	for _, s := range links {
		attrs := pairs(doc, s, "attribute")
		for _, end := range [][2]string{{"to", "method"}, {"from", "signal"}} {
			spec, ok := runs[attrs[end[0]].val]
			if name := attrs[end[1]].val; ok && name != "" {
				r.Edges = append(r.Edges, extract.RawEdge{
					Source: model.NodeID(rel), Relation: model.RelationReferences, Name: name, Specifier: spec, File: rel, Line: line(s),
				})
			}
		}
	}
	return r, nil
}

// nodePath is where a [node] section stands below the scene's root: "." for
// the root, which has no parent, then its name joined to its parent's path.
func nodePath(attrs map[string]entry) string {
	parent, ok := attrs["parent"]
	switch {
	case !ok:
		return "."
	case parent.val == ".":
		return attrs["name"].val
	}
	return parent.val + "/" + attrs["name"].val
}

// extID is the id of the ExtResource(id) that the child of s of type typ
// and key holds: `script = ExtResource( 1 )`, `instance=ExtResource("2_a")`.
// "" for anything else.
func extID(doc *treesitter.Doc, s *gts.Node, typ, key string) string {
	for _, c := range s.Children() {
		v := c.Child(2)
		if doc.Type(c) != typ || doc.Text(c.Child(0)) != key || doc.Type(v) != "constructor" || doc.Text(v.Child(0)) != "ExtResource" {
			continue
		}
		id := ""
		treesitter.Walk(v, func(a *gts.Node) bool {
			if id == "" {
				id = resID(doc, a)
			}
			return id == ""
		})
		return id
	}
	return ""
}

// idOf is the id attribute of an [ext_resource] header.
func idOf(doc *treesitter.Doc, s *gts.Node) string {
	for _, c := range s.Children() {
		if doc.Type(c) == "attribute" && doc.Text(c.Child(0)) == "id" {
			return resID(doc, c.Child(2))
		}
	}
	return ""
}

// resID is an external resource's id as written: an integer in format=2
// (Godot 3), a string in format=3 (Godot 4); "" for anything else.
func resID(doc *treesitter.Doc, v *gts.Node) string {
	switch doc.Type(v) {
	case "string", "integer":
		return unquote(doc.Text(v))
	}
	return ""
}

// entry is one key and string value of a resource file, with the 1-based line
// it stands on.
type entry struct {
	key, val string
	line     int
}

// ordered is the key and unquoted string value of every child of s of type
// typ, in source order: attribute in a section's header, property below it.
// A value that is no string literal is left out.
func ordered(doc *treesitter.Doc, s *gts.Node, typ string) []entry {
	var out []entry
	for _, c := range s.Children() {
		if doc.Type(c) == typ && doc.Type(c.Child(2)) == "string" {
			out = append(out, entry{doc.Text(c.Child(0)), unquote(doc.Text(c.Child(2))), line(c)})
		}
	}
	return out
}

// pairs is ordered as a map, for a section read by key.
func pairs(doc *treesitter.Doc, s *gts.Node, typ string) map[string]entry {
	m := map[string]entry{}
	for _, e := range ordered(doc, s, typ) {
		m[e.key] = e
	}
	return m
}
