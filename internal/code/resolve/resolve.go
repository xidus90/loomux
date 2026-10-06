package resolve

import (
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// Graph resolves every raw edge and returns a whole wiring graph. extractor is
// the stamp the graph's meta carries; the caller names it, because this
// package knows the rules of each language and not the list of them.
//
// Each language is resolved against its own files alone: a Go call never
// lands on a Python function of the same name, and a name defined once per
// language is still unique within each; a Godot path alone may land on a file
// of another language, by its path.
func Graph(files []extract.Result, mods []Module, extractor string) *model.Graph {
	groups := byLanguage(files)
	g := &model.Graph{Meta: model.Meta{
		Version: schemaVersion, Extractor: extractor, Languages: languagesOf(groups),
	}}
	for _, f := range files {
		g.Nodes = append(g.Nodes, f.Nodes...)
	}
	// Every file of the build, of any language: a Godot path may name a file
	// another extractor reads. Only Godot's resolver reads it.
	var paths map[string]bool
	if len(groups["gdscript"]) > 0 {
		paths = map[string]bool{}
		for _, f := range files {
			paths[f.Path] = true
		}
	}
	for _, lang := range g.Meta.Languages {
		g.Edges = append(g.Edges, edgesOf(lang, groups[lang], mods, paths)...)
	}
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	sort.Slice(g.Edges, func(i, j int) bool {
		a, b := g.Edges[i], g.Edges[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Relation != b.Relation {
			return a.Relation < b.Relation
		}
		return a.Target < b.Target
	})
	g.Meta.NodeCount = len(g.Nodes)
	g.Meta.EdgeCount = len(g.Edges)
	return g
}

// schemaVersion is the version Graph writes. model refuses anything else, and
// the two are bumped together.
const schemaVersion = 2

// byLanguage groups the files by the language that extracted them, each group
// in the order the files arrived. That order is kept on purpose: the edge sort
// in Graph is not stable, so two edges alike but for their confidence may come
// out in any order -- but the same input order gives the same output order,
// and that is what a byte-identical graph needs.
func byLanguage(files []extract.Result) map[string][]extract.Result {
	groups := map[string][]extract.Result{}
	for _, f := range files {
		groups[f.Language] = append(groups[f.Language], f)
	}
	return groups
}

// languagesOf is the sorted names of the groups: the meta's list, and the
// order the groups are resolved in. Never nil, so a graph of no files says
// `[]` and not `null`.
func languagesOf(groups map[string][]extract.Result) []string {
	langs := make([]string, 0, len(groups))
	for lang := range groups {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}

// edgesOf resolves the raw edges of one language's files against an index of
// those files alone: Go's here, Python's in resolvePython, Godot's in
// resolveGDScript. Go's and Python's indexes hold their own language's files
// alone; the paths a Godot file names see every file of the build.
//
// A language without resolution rules keeps its containment, which its
// extractor already resolved, and loses every other edge: a call or an import
// resolved by another language's rules would be a guess dressed up as an
// edge.
func edgesOf(lang string, files []extract.Result, mods []Module, paths map[string]bool) []model.Edge {
	switch lang {
	case "python":
		return resolvePython(files)
	case "gdscript":
		return resolveGDScript(files, paths)
	}
	var out []model.Edge
	if lang != "go" {
		for _, f := range files {
			for _, raw := range f.Edges {
				if raw.Relation == model.RelationContains {
					out = append(out, model.Edge{
						Source: raw.Source, Target: raw.TargetID,
						Relation: model.RelationContains, Confidence: model.ConfidenceExtracted,
					})
				}
			}
		}
		return out
	}
	idx := index(files)
	for _, f := range files {
		for _, raw := range f.Edges {
			if e, ok := idx.resolveEdge(raw, mods); ok {
				out = append(out, e)
			}
		}
	}
	return out
}

// repoIndex is every lookup the resolver makes, built once.
type repoIndex struct {
	nodes      map[model.NodeID]model.Node
	perFile    map[string]map[string][]model.Node // path -> name -> functions
	global     map[string][]model.Node            // name -> functions, repo-wide
	byOwner    map[string][]model.Node            // "Owner.name" -> methods
	perDir     map[string]map[string][]model.Node // dir -> name -> functions, no tests
	filesInDir map[string][]string                // dir -> file node paths, no tests
	clauseOf   map[string]string                  // dir -> package clause, no tests
	importsOf  map[string][]extract.Import        // file path -> its imports
}

func index(files []extract.Result) *repoIndex {
	x := &repoIndex{
		nodes: map[model.NodeID]model.Node{}, perFile: map[string]map[string][]model.Node{},
		global: map[string][]model.Node{}, byOwner: map[string][]model.Node{},
		perDir: map[string]map[string][]model.Node{}, filesInDir: map[string][]string{},
		clauseOf: map[string]string{}, importsOf: map[string][]extract.Import{},
	}
	for _, f := range files {
		x.importsOf[f.Path] = f.Imports
		// The clause of a package is what an importer binds without an alias.
		// Test files are skipped: `package X_test` is a different clause for the
		// same directory, and no importer ever sees it.
		if !isTestFile(f.Path) && f.Package != "" {
			x.clauseOf[dirOf(f.Path)] = f.Package
		}
		for _, n := range f.Nodes {
			x.nodes[n.ID] = n
			switch {
			case n.Kind == model.KindFile:
				if !isTestFile(n.Path) {
					dir := dirOf(n.Path)
					x.filesInDir[dir] = append(x.filesInDir[dir], n.Path)
				}
			case n.Kind == "method":
				x.byOwner[n.Owner+"."+n.Name] = append(x.byOwner[n.Owner+"."+n.Name], n)
			case n.Kind == "function":
				if x.perFile[n.Path] == nil {
					x.perFile[n.Path] = map[string][]model.Node{}
				}
				x.perFile[n.Path][n.Name] = append(x.perFile[n.Path][n.Name], n)
				x.global[n.Name] = append(x.global[n.Name], n)
				// A selector's candidates: functions of the target package,
				// test files excluded -- an importer never sees them.
				if !isTestFile(n.Path) {
					dir := dirOf(n.Path)
					if x.perDir[dir] == nil {
						x.perDir[dir] = map[string][]model.Node{}
					}
					x.perDir[dir][n.Name] = append(x.perDir[dir][n.Name], n)
				}
			}
		}
	}
	for dir := range x.filesInDir {
		sort.Strings(x.filesInDir[dir])
	}
	return x
}

// isTestFile reports whether an importer would ever see this file. It would
// not: neither the `package X` tests nor the `package X_test` ones.
func isTestFile(p string) bool {
	return strings.HasSuffix(p, "_test.go")
}

// resolveEdge turns one raw edge into an edge, or reports that it is not sound
// enough to keep.
//
//coverage:exempt the default arm needs an extract.RawEdge whose Relation is neither contains, imports nor calls; the extractor (internal/code/extract/golang/extract.go) only ever constructs those three, so no real caller can produce one
func (x *repoIndex) resolveEdge(raw extract.RawEdge, mods []Module) (model.Edge, bool) {
	switch raw.Relation {
	case model.RelationContains:
		// Already resolved by the extractor, and structural containment is
		// certain by construction.
		return model.Edge{
			Source: raw.Source, Target: raw.TargetID,
			Relation: model.RelationContains, Confidence: model.ConfidenceExtracted,
		}, true
	case model.RelationImports:
		return model.Edge{
			Source: raw.Source, Target: x.importTarget(raw.Specifier, mods),
			Relation: model.RelationImports, Confidence: model.ConfidenceExtracted,
		}, true
	case model.RelationCalls:
		return x.resolveCall(raw, mods)
	default:
		return model.Edge{}, false
	}
}

// importTarget is a representative file node of the imported package, or the
// raw package path when the import points outside every module of the
// repository.
//
// The representative is the lowest id among the package's NON-test files.
// The original takes the lowest id outright; in Go that could point an import
// at `index_test.go`, a file the importer never sees.
func (x *repoIndex) importTarget(spec string, mods []Module) model.NodeID {
	dir, ok := importDir(spec, mods)
	if !ok {
		return model.NodeID(spec)
	}
	files := x.filesInDir[dir]
	if len(files) == 0 {
		// Inside a module of the repository, but nothing indexed there: an
		// empty package directory, or one holding only tests.
		return model.NodeID(spec)
	}
	return model.NodeID(files[0])
}

// resolveCall applies the three shapes a raw call may have.
func (x *repoIndex) resolveCall(raw extract.RawEdge, mods []Module) (model.Edge, bool) {
	switch {
	case raw.Receiver != "":
		return x.resolveSelector(raw, mods)
	case raw.Owner != "":
		// A member call on a known local type: the owner-qualified index is the
		// only sound answer, because a unique bare method name says nothing
		// about its receiver.
		return one(raw, x.byOwner[raw.Owner+"."+raw.Name], model.ConfidenceExtracted)
	default:
		if e, ok := one(raw, x.perFile[raw.File][raw.Name], model.ConfidenceExtracted); ok {
			return e, true
		}
		// One match across files. Shadowing could in principle fool this, so it
		// is inferred; several matches are dropped rather than guessed.
		return one(raw, x.global[raw.Name], model.ConfidenceInferred)
	}
}

// resolveSelector answers `pkg.Fn()` against the one package it can mean.
//
// This is where the port goes past the original, which drops the case for Go.
// The rule is the original's own, for a reference with a specifier
// (resolve.ts:229-237): resolve inside the named target alone, and only when
// exactly one candidate is there, "so a same-named symbol elsewhere in the repo
// cannot become a false edge".
//
// Candidates are functions. A call `pkg.T(x)` on a type is a conversion, and
// methods are excluded because a package may hold `func New()` and
// `func (x *T) New()` at once -- only the first can be what the selector means.
func (x *repoIndex) resolveSelector(raw extract.RawEdge, mods []Module) (model.Edge, bool) {
	dir, ok := x.packageDir(raw.File, raw.Receiver, mods)
	if !ok {
		// The receiver names no package of this repository: the standard
		// library, a third-party package, or something this extractor cannot
		// see at all. Nothing to point at.
		return model.Edge{}, false
	}
	return one(raw, x.perDir[dir][raw.Name], model.ConfidenceExtracted)
}

// packageDir answers which directory of the repository a selector's receiver
// names, for the file the selector stands in.
//
// Two rounds, and the order is Go's own:
//
//  1. an ALIAS binds outright -- `import b ".../blast"` makes `b.New` that
//     package, whatever the target calls itself
//  2. otherwise the bound name is the TARGET's package clause, which is why
//     this lives here and not in the extractor: `import "gopkg.in/yaml.v3"`
//     binds `yaml`, and only a side that has read the target directory knows
//     that
//
// "_" and "." bind no selector and are skipped in both rounds.
func (x *repoIndex) packageDir(file, receiver string, mods []Module) (string, bool) {
	var plain []extract.Import
	for _, imp := range x.importsOf[file] {
		switch imp.Alias {
		case "":
			plain = append(plain, imp)
		case "_", ".":
			// Binds nothing.
		case receiver:
			return importDir(imp.Path, mods)
		}
	}
	for _, imp := range plain {
		dir, ok := importDir(imp.Path, mods)
		if !ok {
			continue
		}
		if x.clauseOf[dir] == receiver {
			return dir, true
		}
	}
	return "", false
}

// one keeps an edge when the candidate set names exactly one node.
func one(raw extract.RawEdge, candidates []model.Node, conf model.Confidence) (model.Edge, bool) {
	if len(candidates) != 1 {
		return model.Edge{}, false
	}
	return model.Edge{
		Source: raw.Source, Target: candidates[0].ID,
		Relation: raw.Relation, Confidence: conf,
	}, true
}
