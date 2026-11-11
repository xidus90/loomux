// Package cpp extracts the nodes and raw edges of one C++ file on the
// shared tree-sitter core.
//
// Raw, as in the other extractors: a call, a base class or an include names
// its target, and resolving the name needs every file of the repository,
// which is internal/code/resolve's job.
//
// Nodes are the file, every class, struct, union and enum with a body --
// nested ones as Outer.Inner -- and every function with a body: a free
// function, a method inside its class, and a method defined outside it
// (`void ns::A::m() {}`), whose owner is the last segment of the qualifier:
// a class, or for `void util::z() {}` the namespace util, which the
// resolver tells apart. Namespaces name no node; Result.Enclosing carries
// the namespace of each type and free function, and for a method defined
// outside its class the namespace plus the whole qualifier. Every call
// belongs to the innermost function with a node, else to its type, else to
// the file.
//
// For Godot, a class whose body holds GDCLASS(<Name>, …) is offered as
// godot.class <Name>, and the bindings of a _bind_methods body become
// references edges for the resolver (extract.RawEdge.Member).
//
// Two departures from the other tree-sitter extractors, both measured on
// real repositories (docs/.superpowers/specs/2026-11-10-graph-sprachen-programm.md):
//
//   - A line holding nothing but an upper-case macro -- GDCLASS(X, Base),
//     Q_OBJECT -- is blanked before the parse (mask), since the grammar
//     cannot expand it and breaks the class around it. Offsets do not move,
//     so spans and body text are cut from the original.
//   - A definition counts when its own subtree holds no ERROR, whatever lies
//     above it: an unexpanded macro elsewhere wraps most of a file in ERROR
//     nodes, and the definitions inside are whole.
//
// gotreesitter can end a tree before the end of the file without a word
// (docs/.superpowers/specs/2026-11-10-cpp-design.md); the rest is read
// again chunk by chunk. Known limits of that:
//
//   - A chunk starts at a line in column 0 with a name, as a namespace-level
//     declaration does. Indented text between the cut and the first such
//     line is not read, a class cut in its body included.
//   - Namespaces and classes open at the cut are lost: what follows names
//     none, and the brace closing one breaks the definition in front of it,
//     which then gets no node.
//
// The node types and fields read here, as gotreesitter builds them
// (treesitter.Parser):
//
//	translation_unit                   everything as children
//	namespace_definition               field name (namespace_identifier or nested_namespace_specifier,
//	                                   none when anonymous), field body
//	class_specifier, struct_specifier,
//	union_specifier, enum_specifier    fields name, body; a base_class_clause child
//	field_declaration_list             a class body: members and access_specifier children
//	template_declaration               the templated definition as its last child
//	function_definition                fields declarator (a function_declarator, possibly wrapped in a
//	                                   pointer or reference declarator), body; storage_class_specifier
//	function_declarator                field declarator: identifier, field_identifier, qualified_identifier,
//	                                   destructor_name, operator_name, template_function
//	operator_cast                      a conversion operator in place of a function_declarator: field type
//	qualified_identifier               fields scope (absent for ::x), name
//	template_type, template_function,
//	template_method                    field name
//	dependent_name                     obj.template f<T>: the name as last child
//	preproc_include                    field path: string_literal or system_lib_string
//	using_declaration                  a namespace child when it opens a namespace; the name as a child
//	call_expression                    fields function, arguments
//	field_expression                   fields argument, field
//	new_expression                     field type
package cpp

import (
	"bytes"
	"regexp"
	"slices"
	"strings"
	"sync"

	gts "github.com/odvcencio/gotreesitter"
	cppgrammar "github.com/odvcencio/gotreesitter/grammars/cpp"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// version is this extractor's identity. Bump it by hand whenever the nodes or
// edges below change shape or meaning; the extract cache and the graph's meta
// key on it (see golang.Version for what a forgotten bump costs).
const version = "cpp/1"

// langName is the language as the graph's meta lists it and as a Result carries it.
const langName = "cpp"

// Language is the C++ extractor as extract.Language describes one.
type Language struct{}

// Name is "cpp".
func (Language) Name() string { return langName }

// Version is this package's version plus the tree-sitter runtime: a new
// runtime can build other trees from the same source.
func (Language) Version() string { return version + "@" + treesitter.Parser }

// Extensions are the sources and headers of C++. A ".h" may as well be C;
// the C++ grammar reads most C.
func (Language) Extensions() []string {
	return []string{".cpp", ".cc", ".cxx", ".c++", ".hpp", ".hh", ".hxx", ".h++", ".h", ".inl", ".ipp", ".tpp"}
}

// File is this package's File.
func (Language) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// File extracts one C++ file. rel is its repo-relative, slash-separated path;
// source is its contents.
func File(rel, source string) (extract.Result, error) {
	return file(cppgrammar.Language(), rel, source)
}

// maxChunks bounds how many parses read the rest of a file its first tree
// left behind. A rest with more declarations is read in fewer, larger
// chunks.
const maxChunks = 256

// file is File with the grammar handed in: the C++ grammar builds a tree from
// any input, and only a missing grammar makes the parse fail.
func file(lang *gts.Language, rel, source string) (extract.Result, error) {
	masked := mask([]byte(source))
	doc, err := treesitter.Parse(lang, rel, masked)
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()
	// The mask keeps every offset, so the tree cuts the original as well.
	doc.Src = []byte(source)

	x := &extractor{doc: doc, fileID: model.NodeID(rel), marks: gdclasses(doc.Src)}
	x.walk(doc.Root, scope{parent: x.fileID, source: x.fileID})
	errs := doc.ParseErrors()
	// gotreesitter can end a tree before the source without a word
	// (docs/.superpowers/specs/2026-11-10-cpp-design.md). The rest is read
	// chunk by chunk, each parsed alone at its own offsets: one long parse
	// of the rest can take minutes where its parts take milliseconds.
	starts := chunks(masked, doc.Root.EndByte())
	if len(starts) > 0 {
		errs++ // the text the first tree left behind
	}
	for i := 0; i < len(starts) && reparseChunk(doc, masked, starts, i); i++ {
		x.walk(doc.Root, scope{parent: x.fileID, source: x.fileID})
		errs += doc.ParseErrors()
	}
	return extract.Result{
		Path:      rel,
		Language:  langName,
		Enclosing: x.enclosing,
		Imports:   x.imports,
		// walk has run every Symbol by now, as FileNode requires: its body
		// text is what no symbol covers.
		Nodes:       append([]model.Node{doc.FileNode()}, x.nodes...),
		Edges:       slices.Concat(x.contains, x.importEdges, x.extends, x.calls, x.bindings),
		ParseErrors: errs,
		Exports:     x.exports,
	}, nil
}

// macroLine is a line holding one upper-case name and at most an argument
// list without ';', '{' or '}'.
var macroLine = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?m)^[ \t]*[A-Z_][A-Z0-9_]*[ \t]*(?:\([^;{}\n]*\))?[ \t]*\r?$`)
})

// aliasDecl is an alias declaration, its right side as group 1. The right
// side holds no brace: it ends at the first ';' of the same declaration.
var aliasDecl = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`\busing[ \t]+[A-Za-z_][A-Za-z0-9_]*[ \t]*=([^;={}]*);`)
})

// mask hides from the parser what it cannot read, keeping each offset and
// line break:
//
//   - every macro line, blanked with spaces. A macro line followed by a '{'
//     stays: it heads a definition, as `TEST_F(Suite, Name)` does with its
//     brace on the next line.
//   - the right side of an alias holding a template argument list and a
//     qualifier both: `using X = A<B<T>>::type;` ends gotreesitter's tree
//     there (docs/.superpowers/specs/2026-11-10-cpp-design.md). It becomes
//     int: no node or edge reads an alias's target. Text in a comment or a
//     literal is never taken for an alias.
func mask(src []byte) []byte {
	out := bytes.Clone(src)
	for _, m := range macroLine().FindAllIndex(src, -1) {
		if rest := bytes.TrimLeft(src[m[1]:], " \t\r\n"); len(rest) > 0 && rest[0] == '{' {
			continue
		}
		blank(out[m[0]:m[1]])
	}
	inLiteral := literals(src)
	for _, m := range aliasDecl().FindAllSubmatchIndex(src, -1) {
		if inLiteral[m[0]] || inLiteral[m[3]] {
			continue // the text reads like an alias inside a comment or a literal
		}
		rhs := out[m[2]:m[3]]
		if !bytes.Contains(rhs, []byte("<")) || !bytes.Contains(rhs, []byte("::")) {
			continue
		}
		start := len(rhs) - len(bytes.TrimLeft(rhs, " \t\r\n"))
		if bytes.ContainsAny(rhs[start:min(start+3, len(rhs))], "\r\n") {
			// "int" would not fit before the line break; the alias stays.
			continue
		}
		blank(rhs)
		copy(rhs[start:], "int")
	}
	return out
}

// literals marks every byte of src that lies in a comment, a string, a raw
// string or a character literal. An unterminated comment or raw string runs to
// the end, an unterminated string or character literal to its line's end.
func literals(src []byte) []bool {
	in := make([]bool, len(src))
	for i := 0; i < len(src); {
		var end int
		switch {
		case bytes.HasPrefix(src[i:], []byte("//")):
			end = lineEnd(src, i)
		case bytes.HasPrefix(src[i:], []byte("/*")):
			end = closing(src, i+2, "*/")
		case src[i] == '"' && i > 0 && src[i-1] == 'R':
			end = rawStringEnd(src, i)
		case src[i] == '"':
			end = quotedEnd(src, i)
		case src[i] == '\'' && !digitSeparator(src, i):
			end = quotedEnd(src, i)
		default:
			i++
			continue
		}
		for ; i < end; i++ {
			in[i] = true
		}
	}
	return in
}

// lineEnd is the offset of the line break after from, or the end of src.
func lineEnd(src []byte, from int) int {
	if nl := bytes.IndexByte(src[from:], '\n'); nl >= 0 {
		return from + nl
	}
	return len(src)
}

// closing is the offset after the next end at or after from, or the end of src.
func closing(src []byte, from int, end string) int {
	if k := bytes.Index(src[from:], []byte(end)); k >= 0 {
		return from + k + len(end)
	}
	return len(src)
}

// quotedEnd is the offset after the quote that closes the literal opening at
// src[open], honouring backslash escapes, or its line's end when it has none.
func quotedEnd(src []byte, open int) int {
	for i := open + 1; i < len(src); i++ {
		switch src[i] {
		case '\\':
			i++
		case src[open]:
			return i + 1
		case '\n':
			return i
		}
	}
	return len(src)
}

// maxRawDelimiter is the longest delimiter of a raw string the language allows.
const maxRawDelimiter = 16

// rawStringEnd is the offset after the raw string whose opening quote is
// src[open], R"delim( ... )delim"; a delimiter that is none reads the quote as
// an ordinary string's.
func rawStringEnd(src []byte, open int) int {
	paren := bytes.IndexByte(src[open:min(open+maxRawDelimiter+2, len(src))], '(')
	if paren < 0 || bytes.ContainsAny(src[open+1:open+paren], " )\\\t\r\n\"") {
		return quotedEnd(src, open)
	}
	return closing(src, open+paren, ")"+string(src[open+1:open+paren])+`"`)
}

// digitSeparator reports whether the quote at src[i] separates the digits of
// a number (1'000), not opens a character literal.
func digitSeparator(src []byte, i int) bool {
	return i > 0 && i+1 < len(src) && isHexDigit(src[i-1]) && isHexDigit(src[i+1])
}

func isHexDigit(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

// chunks are where the chunks of the text after a tree that ends at end
// start: every line after it that starts in column 0 with a name or an
// attribute, as a declaration at namespace level does, and a chunk runs to
// the next start. More than maxChunks starts are thinned to every k-th.
// Text before the first start is not read: a chunk starting inside a
// construct -- a class body after `#else`, say -- only costs an error
// recovery that wraps the definitions after it in its ERROR.
func chunks(src []byte, end uint32) []int {
	var starts []int
	for pos := int(end); ; {
		nl := bytes.IndexByte(src[pos:], '\n')
		if nl < 0 {
			break
		}
		pos += nl + 1
		if pos < len(src) && opensDeclaration(src[pos]) {
			starts = append(starts, pos)
		}
	}
	if len(starts) <= maxChunks {
		return starts
	}
	k := (len(starts) + maxChunks - 1) / maxChunks
	thinned := make([]int, 0, maxChunks)
	for i := 0; i < len(starts); i += k {
		thinned = append(thinned, starts[i])
	}
	return thinned
}

// opensDeclaration is whether a line starting with c can open a declaration.
func opensDeclaration(c byte) bool {
	return c == '_' || c == '[' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

// reparseChunk parses chunk i of src into doc, everything before it blanked
// and everything after it left off; false when the parse fails.
func reparseChunk(doc *treesitter.Doc, src []byte, starts []int, i int) bool {
	end := len(src)
	if i+1 < len(starts) {
		end = starts[i+1]
	}
	chunk := bytes.Clone(src[:end])
	blank(chunk[:starts[i]])
	return doc.Reparse(chunk) == nil
}

// blank overwrites b with spaces, keeping its line breaks.
func blank(b []byte) {
	for i, c := range b {
		if c != '\n' && c != '\r' {
			b[i] = ' '
		}
	}
}

// mark is a GDCLASS line: where it starts, the class it names, and whether
// a class has offered it yet.
type mark struct {
	at      int
	name    string
	claimed bool
}

// gdclassLine is a line opening with GDCLASS(<Name>, masked or not: godot-cpp
// writes it with a ';' as well.
var gdclassLine = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?m)^[ \t]*GDCLASS[ \t]*\([ \t]*([A-Za-z_][A-Za-z0-9_]*)`)
})

// gdclasses are the GDCLASS lines of src.
func gdclasses(src []byte) []*mark {
	var marks []*mark
	for _, m := range gdclassLine().FindAllSubmatchIndex(src, -1) {
		marks = append(marks, &mark{at: m[0], name: string(src[m[2]:m[3]])})
	}
	return marks
}

// extractor gathers one file's nodes and edges in source order.
type extractor struct {
	doc    *treesitter.Doc
	fileID model.NodeID
	marks  []*mark

	enclosing map[model.NodeID]string // nil until the first entry
	nodes     []model.Node
	contains  []extract.RawEdge
	extends   []extract.RawEdge
	calls     []extract.RawEdge
	bindings  []extract.RawEdge

	imports     []extract.Import
	importEdges []extract.RawEdge

	exports []extract.Export
}

// scope is what a node sees of the definitions around it.
type scope struct {
	ns     string       // "a::b", "" for the global namespace
	anon   bool         // inside an anonymous namespace
	class  string       // the innermost class as "Outer.Inner", "" outside one
	public bool         // the access of the class body read so far
	parent model.NodeID // the node a new node hangs under
	source model.NodeID // the node a call belongs to; "" where calls count for nobody
	binder string       // the class whose _bind_methods this is, "" elsewhere
	binds  bool         // inside a _bind_methods with a node
}

// walk reads n and everything below it.
func (x *extractor) walk(n *gts.Node, s scope) {
	switch x.doc.Type(n) {
	case "ERROR":
		// A call in rejected text could be credited to the wrong scope; a
		// whole definition below still counts.
		s.source, s.binds = "", false
	case "namespace_definition":
		name := x.namespace(x.doc.Field(n, "name"))
		s.ns, s.anon = join(s.ns, name), s.anon || name == ""
	case "class_specifier", "struct_specifier", "union_specifier", "enum_specifier":
		if x.typeDef(n, s) {
			return
		}
	case "function_definition":
		if x.function(n, s) {
			return
		}
	case "field_declaration_list":
		for _, c := range n.Children() {
			if x.doc.Type(c) == "access_specifier" {
				s.public = x.doc.Text(c) == "public"
				continue
			}
			x.walk(c, s)
		}
		return
	case "preproc_include":
		x.include(n)
		return
	case "using_declaration":
		x.using(n)
		return
	case "call_expression":
		x.call(n, s)
	case "new_expression":
		x.construct(n, s)
	}
	for _, c := range n.Children() {
		x.walk(c, s)
	}
}

// typeDef makes the node of one class, struct, union or enum and reads its
// body; false for one without a name or a body, which is no definition of
// its own.
func (x *extractor) typeDef(def *gts.Node, s scope) bool {
	nameNode, body := x.doc.Field(def, "name"), x.doc.Field(def, "body")
	segs := x.segments(nameNode)
	if len(segs) == 0 || body == nil {
		return false
	}
	name := segs[len(segs)-1]
	qualified, owner, exported := name, "", !s.anon
	if s.class != "" {
		qualified, owner, exported = s.class+"."+name, s.class, s.public
	}
	inner := s
	inner.class = qualified
	inner.public = x.doc.Type(def) != "class_specifier"
	inner.source, inner.binds = "", false
	if !treesitter.HoldsError(def) {
		headEnd := uint32(0)
		for _, c := range def.Children() {
			if c.StartByte() >= body.StartByte() {
				break
			}
			headEnd = c.EndByte()
		}
		n := x.symbol(def, headEnd, name, qualified, "class", owner, exported, s.parent)
		x.enclose(n.ID, s.ns)
		inner.parent, inner.source = n.ID, n.ID
		for _, c := range def.Children() {
			if x.doc.Type(c) == "base_class_clause" {
				x.bases(n.ID, c)
			}
		}
	}
	for _, c := range def.Children() {
		x.walk(c, inner)
	}
	// After the body: a nested class has claimed its own GDCLASS by now.
	x.offerClass(body, inner.source)
	return true
}

// function makes the node of one function with a body and reads the body;
// false for one without a body or with a declarator this package does not
// read.
func (x *extractor) function(def *gts.Node, s scope) bool {
	body := x.doc.Field(def, "body")
	nameNode, head := x.declarator(def)
	segs := x.segments(nameNode)
	if body == nil || len(segs) == 0 {
		return false
	}
	name, qualifier := segs[len(segs)-1], segs[:len(segs)-1]
	kind, owner, qualified := model.Kind("function"), "", name
	exported, enclosing, enclosed := !s.anon && !x.static(def), s.ns, true
	switch {
	case len(qualifier) > 0:
		// Defined outside its class: ns::A::m belongs to A, and whether ns
		// names a namespace or a class is the resolver's to find out.
		kind, owner = "method", qualifier[len(qualifier)-1]
		qualified, enclosing = owner+"."+name, join(s.ns, strings.Join(qualifier, "::"))
	case s.class != "":
		kind, owner, qualified, exported, enclosed = "method", s.class, s.class+"."+name, s.public, false
	}
	inner := s
	// A type in a function body is local, not nested in the class around.
	inner.class, inner.source, inner.binds = "", "", false
	if !treesitter.HoldsError(def) {
		n := x.symbol(def, head.EndByte(), name, qualified, kind, owner, exported, s.parent)
		if enclosed {
			x.enclose(n.ID, enclosing)
		}
		inner.parent, inner.source = n.ID, n.ID
		inner.binder, inner.binds = owner, name == "_bind_methods"
	}
	for _, c := range def.Children() {
		x.walk(c, inner)
	}
	return true
}

// declarator is the name a function definition declares and the node that
// ends its signature: the first function_declarator below the declarator,
// which a pointer or reference return type wraps. A conversion operator
// has none; its operator_cast holds the name and the parameters both.
func (x *extractor) declarator(def *gts.Node) (name, head *gts.Node) {
	d := x.doc.Field(def, "declarator")
	var fd *gts.Node
	treesitter.Walk(d, func(n *gts.Node) bool {
		if fd == nil && x.doc.Type(n) == "function_declarator" {
			fd = n
		}
		return fd == nil
	})
	if fd == nil {
		return d, d
	}
	return x.doc.Field(fd, "declarator"), fd
}

// static is whether a definition carries the static storage class.
func (x *extractor) static(def *gts.Node) bool {
	for _, c := range def.Children() {
		if x.doc.Type(c) == "storage_class_specifier" && x.doc.Text(c) == "static" {
			return true
		}
	}
	return false
}

// segments is a name as its "::"-separated parts with template arguments
// dropped: ns::Box<T>::put is ns, Box, put. A leading :: adds no part. Nil
// when anything but a name stands in it.
func (x *extractor) segments(n *gts.Node) []string {
	if n == nil {
		return nil
	}
	switch x.doc.Type(n) {
	case "identifier", "field_identifier", "type_identifier", "namespace_identifier":
		return []string{x.doc.Text(n)}
	case "destructor_name":
		return []string{strings.Join(strings.Fields(x.doc.Text(n)), "")}
	case "operator_name":
		// "operator +=" is operator+=, "operator  new" is operator new.
		op := strings.Join(strings.Fields(strings.TrimPrefix(x.doc.Text(n), "operator")), " ")
		if op != "" && (op[0] == '_' || 'a' <= op[0] && op[0] <= 'z') {
			op = " " + op
		}
		return []string{"operator" + op}
	case "operator_cast":
		// Named by its target type, as C++ itself names it.
		return []string{"operator " + strings.Join(strings.Fields(x.doc.Text(x.doc.Field(n, "type"))), " ")}
	case "template_type", "template_function", "template_method":
		return x.segments(x.doc.Field(n, "name"))
	case "dependent_name":
		// obj.template f<T>: the name after the keyword.
		return x.segments(n.Child(n.ChildCount() - 1))
	case "qualified_identifier":
		var head []string
		if scope := x.doc.Field(n, "scope"); scope != nil {
			if head = x.segments(scope); head == nil {
				return nil
			}
		}
		if tail := x.segments(x.doc.Field(n, "name")); tail != nil {
			return append(head, tail...)
		}
	}
	return nil
}

// namespace is a namespace definition's name, "a::b" for `namespace a::b`,
// "" for an anonymous one.
func (x *extractor) namespace(n *gts.Node) string {
	var parts []string
	treesitter.Walk(n, func(c *gts.Node) bool {
		if x.doc.Type(c) == "namespace_identifier" {
			parts = append(parts, x.doc.Text(c))
		}
		return true
	})
	return strings.Join(parts, "::")
}

// enclose records the namespace or qualifier id is declared in.
func (x *extractor) enclose(id model.NodeID, ns string) {
	if x.enclosing == nil {
		x.enclosing = map[model.NodeID]string{}
	}
	x.enclosing[id] = ns
}

// symbol mints the node of one definition and the contains edge from parent
// to it, and returns the node. Its id is the one every edge must use: a
// repeated name -- an overload -- gets an ordinal, and recomputing the id
// from the name would name a node that does not exist.
func (x *extractor) symbol(def *gts.Node, headerEnd uint32, name, qualified string, kind model.Kind, owner string, exported bool, parent model.NodeID) model.Node {
	// A template header belongs to the extent, like a decorator, and not
	// to the signature.
	outer := def
	for p := def.Parent(); p != nil && x.doc.Type(p) == "template_declaration"; p = p.Parent() {
		outer = p
	}
	n := x.doc.Symbol(treesitter.Sym{
		Outer:     outer,
		Head:      def.StartByte(),
		HeaderEnd: headerEnd,
		Name:      name,
		Qualified: qualified,
		Kind:      kind,
		Owner:     owner,
		Exported:  exported,
	})
	x.nodes = append(x.nodes, n)
	x.contains = append(x.contains, extract.RawEdge{
		Source: parent, Relation: model.RelationContains, TargetID: n.ID, File: x.doc.Rel,
	})
	return n
}

// join is namespace inner inside outer, either of them "" for none.
func join(outer, inner string) string {
	if outer == "" || inner == "" {
		return outer + inner
	}
	return outer + "::" + inner
}
