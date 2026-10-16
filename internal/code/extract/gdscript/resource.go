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
// an import of every external resource it names by path, and for
// project.godot the autoloads and the main scene. Every autoload entry gets
// its imports edge, but only a singleton (a value starting with "*") is bound
// as an alias: Godot gives only those a global name. A reference by uid alone
// names no file the graph could point at.
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
	for _, s := range doc.Root.Children() {
		if doc.Type(s) != "section" {
			continue
		}
		switch doc.Text(s.Child(1)) {
		case "ext_resource":
			if p, ok := pairs(doc, s, "attribute")["path"]; ok {
				edge(p.val, p.line)
			}
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
	return r, nil
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
