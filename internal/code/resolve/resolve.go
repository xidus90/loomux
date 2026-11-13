package resolve

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// Reason says why a base class stayed unresolved.
type Reason string

const (
	// ReasonNone: the name binds to nothing the project defines -- an engine
	// class, a builtin or an installed package.
	ReasonNone Reason = "none"
	// ReasonAmbiguous: the name binds to several definitions, so no edge is
	// drawn.
	ReasonAmbiguous Reason = "ambiguous"
)

// Unresolved is an extends edge that gave no edge: the base class as written,
// and where.
type Unresolved struct {
	Source model.NodeID
	File   string
	Line   int
	Name   string
	Reason Reason
}

// Graph is GraphWith without the unresolved bases.
func Graph(files []extract.Result, mods []Module, extractor string) *model.Graph {
	g, _ := GraphWith(files, mods, extractor)
	return g
}

// GraphWith resolves every raw edge and returns a whole wiring graph, and the
// extends edges that stayed unresolved, in the order of the families and
// files. extractor is the stamp the graph's meta carries; the caller names it,
// because this package knows the rules of each language and not the list of
// them.
//
// Each family (familyOf) is resolved against its own files alone: a Go call
// never lands on a Python function of the same name, and a name defined once
// per family is still unique within each; JavaScript and TypeScript import
// each other and form one family. A Godot path alone may land on a file of
// another language, by its path. A symbol of another family is reached only
// over a bridge: a name some language offers under a namespace, looked up by
// that namespace's rules once every family is resolved (bridge.go).
func GraphWith(files []extract.Result, mods []Module, extractor string) (*model.Graph, []Unresolved) {
	// An embedded part joins its own family's group, beside its file.
	var flat []extract.Result
	for _, f := range files {
		flat = append(flat, f.Flatten()...)
	}
	files = flat
	groups := byFamily(files)
	g := &model.Graph{Meta: model.Meta{
		Version: schemaVersion, Extractor: extractor, Languages: languagesOf(files),
	}}
	for _, f := range files {
		g.Nodes = append(g.Nodes, f.Nodes...)
	}
	// Every file of the build, of any language: a Godot path, a page's
	// <script src> or <link href> and a stylesheet's import may name a file
	// another extractor reads.
	paths := map[string]bool{}
	for _, f := range files {
		paths[f.Path] = true
	}
	// What each HTML page loads and links: they scope the web handovers of
	// the pages and the scripts.
	web := webOf(files)
	// A family resolves at its first language in the sorted list: the edge
	// sort below is not stable, so the order of the families is part of the
	// bytes of the graph, and a family of one language keeps its old place.
	var outs []langOut
	done := map[string]bool{}
	for _, lang := range g.Meta.Languages {
		family := familyOf(lang)
		if !done[family] {
			done[family] = true
			outs = append(outs, edgesOf(family, groups[family], mods, paths, web))
		}
	}
	edges, unresolved, offers, handovers := mergeLangs(outs)
	all := make([]string, 0, len(files))
	for _, f := range files {
		all = append(all, f.Path)
		for _, e := range f.Exports {
			offers = append(offers, offer{Export: e, file: f.Path, lang: f.Language})
		}
		for _, b := range f.Bridges {
			handovers = append(handovers, handover{bridge: b, unresolved: -1})
		}
	}
	bridged, unresolved := bridgeEdges(builtinRules, all, offers, handovers, edges, unresolved)
	g.Edges = append(edges, bridged...)
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
	return g, unresolved
}

// schemaVersion is the version Graph writes. model refuses anything else, and
// the two are bumped together.
const schemaVersion = 3

// familyOf is the group a language resolves in: languages that import each
// other's files share one, every other language is its own.
func familyOf(lang string) string {
	switch lang {
	case "javascript", "typescript":
		return "ecmascript"
	case "css", "scss":
		return "stylesheet"
	}
	return lang
}

// byFamily groups the files by the family of the language that extracted
// them, each group in the order the files arrived. That order is kept on
// purpose: the edge sort in Graph is not stable, so two edges alike but for
// their confidence may come out in any order -- but the same input order gives
// the same output order, and that is what a byte-identical graph needs.
func byFamily(files []extract.Result) map[string][]extract.Result {
	groups := map[string][]extract.Result{}
	for _, f := range files {
		family := familyOf(f.Language)
		groups[family] = append(groups[family], f)
	}
	return groups
}

// languagesOf is the sorted names of the languages of files: the meta's
// list. Never nil, so a graph of no files says `[]` and not `null`.
func languagesOf(files []extract.Result) []string {
	seen := map[string]bool{}
	langs := []string{}
	for _, f := range files {
		if !seen[f.Language] {
			seen[f.Language] = true
			langs = append(langs, f.Language)
		}
	}
	sort.Strings(langs)
	return langs
}

// edgesOf resolves the raw edges of one family's files against an index of
// those files alone: Go's here, Python's in resolvePython, Godot's in
// resolveGDScript, C#'s in resolveCSharp, C++'s in resolveCPP, Java's in
// resolveJava, PHP's in resolvePHP, JavaScript's and TypeScript's in
// resolveECMAScript, HTML's in resolveHTML, CSS's and SCSS's in
// resolveStylesheet. Every index holds its own family's files alone; paths
// is every file of the build, which a Godot path, a page's link and a
// stylesheet's import may name, and w, what the HTML pages load and link,
// scopes the web handovers of pages and scripts.
//
// A family without resolution rules keeps its containment, which its
// extractor already resolved, and loses every other edge: a call or an import
// resolved by another language's rules would be a guess dressed up as an
// edge.
func edgesOf(family string, files []extract.Result, mods []Module, paths map[string]bool, w webLinks) langOut {
	switch family {
	case "ecmascript":
		return resolveECMAScript(files, mods, w, paths)
	case "html":
		return resolveHTML(files, paths, w)
	case "stylesheet":
		return resolveStylesheet(files, paths)
	case "python":
		edges, un := resolvePython(files)
		return langOut{edges: edges, unresolved: un}
	case "gdscript":
		return resolveGDScript(files, paths)
	case "csharp":
		return resolveCSharp(files)
	case "cpp":
		return resolveCPP(files)
	case "java":
		return resolveJava(files)
	case "php":
		return resolvePHP(files)
	}
	var out []model.Edge
	if family != "go" {
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
		return langOut{edges: out}
	}
	idx := index(files)
	for _, f := range files {
		for _, raw := range f.Edges {
			if e, ok := idx.resolveEdge(raw, mods); ok {
				out = append(out, e)
			}
		}
	}
	return langOut{edges: out}
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
	packageOf  map[string]string                  // file path -> its package clause
}

func index(files []extract.Result) *repoIndex {
	x := &repoIndex{
		nodes: map[model.NodeID]model.Node{}, perFile: map[string]map[string][]model.Node{},
		global: map[string][]model.Node{}, byOwner: map[string][]model.Node{},
		perDir: map[string]map[string][]model.Node{}, filesInDir: map[string][]string{},
		clauseOf: map[string]string{}, importsOf: map[string][]extract.Import{},
		packageOf: map[string]string{},
	}
	for _, f := range files {
		x.importsOf[f.Path] = f.Imports
		x.packageOf[f.Path] = f.Package
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
//coverage:exempt the default arm needs an extract.RawEdge whose Relation is neither contains, imports, calls nor references; the extractor (internal/code/extract/golang/extract.go) only ever constructs those four, so no real caller can produce one
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
			Relation: model.RelationImports, Confidence: model.ConfidenceExtracted, Line: raw.Line,
		}, true
	case model.RelationCalls:
		return x.resolveCall(raw, mods)
	case model.RelationReferences:
		return x.resolveReference(raw, mods)
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
	case raw.Via != nil:
		return x.resolveChained(raw, mods)
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

// resolveReference answers a function used as a value: `Deps{F: pkg.Run}`,
// `register(s.handle)`, `use(target)`.
//
// A selector and a member resolve as a call would. A bare name never falls
// back to the repo-wide index: in Go an undeclared bare identifier is a name
// of the same package or a builtin, so the package directory is the whole
// search and an answer there is certain.
func (x *repoIndex) resolveReference(raw extract.RawEdge, mods []Module) (model.Edge, bool) {
	if raw.Receiver != "" || raw.Owner != "" || raw.Via != nil {
		return x.resolveCall(raw, mods)
	}
	if e, ok := one(raw, x.perFile[raw.File][raw.Name], model.ConfidenceExtracted); ok {
		return e, true
	}
	if strings.HasSuffix(x.packageOf[raw.File], "_test") {
		// An external test package sees its directory's package only through
		// an import, never by a bare name.
		return model.Edge{}, false
	}
	return one(raw, x.perDir[dirOf(raw.File)][raw.Name], model.ConfidenceExtracted)
}

// resolveChained answers `New(g).Reach()`, and `x.Reach()` after
// `x := New(g)`: the inner call resolves by the usual rules, its first result
// type is read from the target's signature, and the method is looked up on
// that type in the target's own package. The chain is as sure as its weakest
// link.
func (x *repoIndex) resolveChained(raw extract.RawEdge, mods []Module) (model.Edge, bool) {
	inner, ok := x.resolveCall(*raw.Via, mods)
	if !ok {
		return model.Edge{}, false
	}
	target := x.nodes[inner.Target]
	typ := resultType(target.Signature)
	if typ == "" {
		return model.Edge{}, false
	}
	var candidates []model.Node
	for _, m := range x.byOwner[typ+"."+raw.Name] {
		if dirOf(m.Path) == dirOf(target.Path) {
			candidates = append(candidates, m)
		}
	}
	return one(raw, candidates, inner.Confidence)
}

// resultType is the type name of a function's first result, read from its
// signature: `func New(g *model.Graph) *Index` gives "Index". A result in
// another package (`*other.T`), no result at all, or a signature that does not
// parse gives "".
func resultType(sig string) string {
	file, err := parser.ParseFile(token.NewFileSet(), "", "package p\n"+sig+"{}", parser.SkipObjectResolution)
	if err != nil || len(file.Decls) != 1 {
		return ""
	}
	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok || fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		// No result, or an empty list written out: `func f() ()`.
		return ""
	}
	t := fn.Type.Results.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	switch g := t.(type) {
	case *ast.IndexExpr:
		t = g.X
	case *ast.IndexListExpr:
		t = g.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
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
		Relation: raw.Relation, Confidence: conf, Line: raw.Line,
	}, true
}
