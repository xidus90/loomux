package resolve

import (
	"path"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// resolvePython resolves the raw edges of the Python files against an index
// of those files alone: containment as the extractor resolved it, an import
// to the file of the module it names, a class to its base class, a call to
// the function, method or constructor it names (call).
//
// An import that names no file of the repository -- the standard library, an
// installed package, a module this tree does not hold -- gives no edge at
// all, where Go keeps the package path as the target. A Go import path is an
// address; a Python module name that is no file here says nothing about
// where it would come from.
//
// Import edges are deduplicated per file and never point at the importing
// file itself. Python reaches one module in several ways (`from . import a`
// and `from . import b` both name the package, `import pkg.a` and `from pkg
// import a` both name pkg/a.py) and all of them are one dependency; `from .
// import sub` in a package's __init__.py names that very file.
func resolvePython(files []extract.Result) []model.Edge {
	x := pyIndexOf(files)
	var out []model.Edge
	var calls []extract.RawEdge
	imported := map[[2]model.NodeID]bool{}
	for _, f := range files {
		for _, raw := range f.Edges {
			switch raw.Relation {
			case model.RelationContains:
				out = append(out, model.Edge{
					Source: raw.Source, Target: raw.TargetID,
					Relation: model.RelationContains, Confidence: model.ConfidenceExtracted,
				})
			case model.RelationImports:
				target, ok := x.module(raw.File, raw.Specifier)
				key := [2]model.NodeID{raw.Source, model.NodeID(target)}
				if !ok || key[1] == raw.Source || imported[key] {
					continue
				}
				imported[key] = true
				out = append(out, model.Edge{
					Source: raw.Source, Target: key[1],
					Relation: model.RelationImports, Confidence: model.ConfidenceExtracted,
				})
			case model.RelationExtends:
				base, ok := x.classNamed(raw.File, raw.Receiver, raw.Name, raw.Source)
				if !ok {
					continue
				}
				// Recorded for the lookups that walk a class's bases, which
				// therefore run after this loop.
				c := x.classes[raw.Source]
				c.bases = append(c.bases, base)
				x.classes[raw.Source] = c
				out = append(out, model.Edge{
					Source: raw.Source, Target: base,
					Relation: model.RelationExtends, Confidence: model.ConfidenceExtracted,
				})
			case model.RelationCalls:
				calls = append(calls, raw)
			}
		}
	}
	// A call in the first file may need the bases of a class in the last.
	for _, raw := range calls {
		if e, ok := x.call(raw); ok {
			out = append(out, e)
		}
	}
	return out
}

// pyIndex is every lookup the Python resolver makes, over the Python files
// alone.
type pyIndex struct {
	roots     map[string]map[string]string       // source root -> dotted module path -> file under that root ("a.b" -> "a/b.py" or "a/b/__init__.py")
	modules   map[string]string                  // dotted module path -> file across every root, a path two roots map to different files left out
	files     map[string]bool                    // every Python file, by path: a relative import names a directory
	perFile   map[string]map[string][]model.Node // file -> name -> its module-level functions and classes
	classes   map[model.NodeID]pyClass           // class id -> its methods and bases
	classOf   map[model.NodeID]model.NodeID      // method id -> the class that contains it
	global    map[string][]model.Node            // name -> functions and classes of every Python file
	importsOf map[string][]extract.Import        // file -> its imports
	names     map[string]bool                    // every directory and module name among the files (pyNames)
}

// pyClass is what a class carries into the resolution of a call on it.
type pyClass struct {
	methods map[string][]model.Node // name -> the methods defined in the class itself
	bases   []model.NodeID          // its base classes in this repository, as written
}

// pyIndexOf indexes the Python files. The bases of a class are left empty:
// resolvePython records them as it resolves the extends edges.
func pyIndexOf(files []extract.Result) *pyIndex {
	x := &pyIndex{
		files: map[string]bool{}, perFile: map[string]map[string][]model.Node{},
		classes: map[model.NodeID]pyClass{}, classOf: map[model.NodeID]model.NodeID{},
		global: map[string][]model.Node{}, importsOf: map[string][]extract.Import{},
	}
	nodes := map[model.NodeID]model.Node{}
	var paths []string
	for _, f := range files {
		x.files[f.Path] = true
		paths = append(paths, f.Path)
		x.importsOf[f.Path] = f.Imports
		for _, n := range f.Nodes {
			nodes[n.ID] = n
			if n.Kind == "class" {
				x.classes[n.ID] = pyClass{methods: map[string][]model.Node{}}
			}
			if n.Kind == "class" || n.Kind == "function" {
				if x.perFile[n.Path] == nil {
					x.perFile[n.Path] = map[string][]model.Node{}
				}
				x.perFile[n.Path][n.Name] = append(x.perFile[n.Path][n.Name], n)
				x.global[n.Name] = append(x.global[n.Name], n)
			}
		}
	}
	// A method belongs to the class that contains it. Its owner's name would
	// not tell apart two classes of one name, `if` and `else` each defining
	// one.
	for _, f := range files {
		for _, raw := range f.Edges {
			if c, ok := x.classes[raw.Source]; ok && raw.Relation == model.RelationContains {
				m := nodes[raw.TargetID]
				c.methods[m.Name] = append(c.methods[m.Name], m)
				x.classOf[m.ID] = raw.Source
			}
		}
	}
	x.roots, x.modules = pyModules(paths, x.files)
	x.names = pyNames(paths)
	return x
}

// pyNames is every name a directory or a module has among the Python files:
// "backend/apps/users/models.py" gives backend, apps, users and models.
func pyNames(paths []string) map[string]bool {
	names := map[string]bool{}
	for _, p := range paths {
		for _, s := range strings.Split(strings.TrimSuffix(p, ".py"), "/") {
			names[s] = true
		}
	}
	return names
}

// pyModules maps every dotted module path to its file: under each source
// root on its own (roots, by root), and across all of them (modules).
//
// Within one root, a.py and a/__init__.py are both "a", and the package wins,
// as it does for CPython, whose finder looks for the directory first. Across
// two roots, a name that each maps to another file -- lib/ at the repository
// root and src/lib/ -- is left out of modules. A file under one of the two
// finds it through that root's own map (absolute); for a file under neither,
// which one Python finds depends on its search path, which no file here
// states, and a guess would be an edge that may be false.
func pyModules(paths []string, files map[string]bool) (roots map[string]map[string]string, modules map[string]string) {
	roots, modules = map[string]map[string]string{}, map[string]string{}
	ambiguous := map[string]bool{}
	// The order of the roots does not matter: each keeps a map of its own,
	// and a clash is found whichever root comes first, and dropped either
	// way.
	for root := range pyRoots(paths, files) {
		under := map[string]string{}
		for _, p := range paths {
			m, ok := modulePath(root, p)
			if !ok {
				continue
			}
			if _, taken := under[m]; !taken || path.Base(p) == "__init__.py" {
				under[m] = p
			}
		}
		roots[root] = under
		for m, p := range under {
			if cur, ok := modules[m]; ok && cur != p {
				ambiguous[m] = true
			}
			modules[m] = p
		}
	}
	for m := range ambiguous {
		delete(modules, m)
	}
	return roots, modules
}

// pyRoots is where an absolute import starts, paths being every Python file
// and files the same as a set:
//
//   - the repository root;
//   - the directory above every top-level package, a directory with an
//     __init__.py whose own parent has none. That is src/ for a package
//     installed from src/<package>, and the project directory of a package
//     that lives one level down in a monorepo;
//   - every directory that holds a manage.py: Django runs it with its own
//     directory on the path, and its apps import each other from there,
//     packages or not.
//
// A loose module, or a directory without an __init__.py, makes no root.
func pyRoots(paths []string, files map[string]bool) map[string]bool {
	roots := map[string]bool{"": true}
	for _, p := range paths {
		switch path.Base(p) {
		case "manage.py":
			roots[dirOf(p)] = true
		case "__init__.py":
			// The root's own __init__.py finds itself as its parent's and
			// adds nothing.
			if parent := dirOf(dirOf(p)); !files[path.Join(parent, "__init__.py")] {
				roots[parent] = true
			}
		}
	}
	return roots
}

// modulePath is the dotted module path of file p under root: its path below
// the root without ".py", a package's __init__ dropped. False when p lies
// outside the root, is the root's own __init__.py, or has a segment with a
// dot in it -- no import can name a.b.py, Python would look for b in a.
func modulePath(root, p string) (string, bool) {
	rel := p
	if root != "" {
		if !strings.HasPrefix(p, root+"/") {
			return "", false
		}
		rel = p[len(root)+1:]
	}
	segs := strings.Split(strings.TrimSuffix(rel, ".py"), "/")
	if segs[len(segs)-1] == "__init__" {
		segs = segs[:len(segs)-1]
	}
	if len(segs) == 0 {
		return "", false
	}
	for _, s := range segs {
		if strings.Contains(s, ".") {
			return "", false
		}
	}
	return strings.Join(segs, "."), true
}

// module is the file a module spec names from file. An absolute spec is
// looked up among the module paths (absolute); a relative one keeps its
// leading dots, and each dot past the first climbs one directory from file's
// own. A spec that climbs above the repository root, or names nothing here,
// has no file.
//
// A relative spec is resolved on the paths themselves, not the module paths:
// it names a directory, and needs no source root to say which. The package
// beats the module here too.
func (x *pyIndex) module(file, spec string) (string, bool) {
	if !strings.HasPrefix(spec, ".") {
		return x.absolute(file, spec)
	}
	rest := strings.TrimLeft(spec, ".")
	dir := dirOf(file)
	for range len(spec) - len(rest) - 1 {
		if dir == "" {
			return "", false
		}
		dir = dirOf(dir)
	}
	p := path.Join(dir, strings.ReplaceAll(rest, ".", "/"))
	for _, candidate := range []string{path.Join(p, "__init__.py"), p + ".py"} {
		if x.files[candidate] {
			return candidate, true
		}
	}
	return "", false
}

// absolute is the file an absolute module spec names from file: under the
// source roots that hold file, the deepest first, the first root that maps
// the spec decides. The tests of tools/cli/tests/ import tests.helpers from
// the root tools/cli/, the ones of a tests/ at the repository root from
// there, and a copy of a package in an example leaves the original's
// importers alone. Walking up from file's directory meets exactly those
// roots, in that order, and the repository root, which holds every file,
// last.
//
// Failing all of them, a root that does not hold file still answers, through
// the map across every root, where a path two roots map to different files
// is left out (pyModules): main.py beside backend/manage.py reaches the apps
// of backend/, which Python would find only with backend/ on its path, and
// the edge is the likely one.
func (x *pyIndex) absolute(file, spec string) (string, bool) {
	for dir := dirOf(file); ; dir = dirOf(dir) {
		if target, ok := x.roots[dir][spec]; ok {
			return target, true
		}
		if dir == "" {
			break
		}
	}
	target, ok := x.modules[spec]
	return target, ok
}

// classNamed is the class that name, or receiver.name, stands for in file:
// through a module the file binds, when a receiver is written, `mod.Base`;
// otherwise in file's own module, and failing that through a from-import.
// Either way a module that passes the name on is followed (offered). Exactly
// one class, or none. A base class is named this way, and so is the receiver
// of `Class.foo()`.
//
// defining is the class whose base is looked for, and never the answer:
// Python reads the bases before it binds the class's name, so the S of
// `class S(S)` is not the class itself. When it is the only S of its module,
// the module has no S to offer, and the one `from base import S` bound is
// the base; with no such import, there is none. Beside a second S of the
// module the two are a guess, as any two classes of one name are -- which
// one is bound where needs the order and the branches, and this index keeps
// neither -- and the import decides the same way. A receiver names no class
// being defined and passes "".
func (x *pyIndex) classNamed(file, receiver, name string, defining model.NodeID) (model.NodeID, bool) {
	if receiver != "" {
		target, ok := x.moduleNamed(file, receiver)
		if !ok {
			return "", false
		}
		return onlyClass(x.offered(target, name), defining)
	}
	if id, ok := onlyClass(x.perFile[file][name], defining); ok {
		return id, true
	}
	nodes, _ := x.imported(file, name)
	return onlyClass(nodes, defining)
}

// call resolves one raw call by its shape:
//
//   - self.foo(), cls.foo() (Owner): foo as the class the calling method is
//     in finds it, itself first and then its bases (method)
//   - x.foo() (Receiver): foo in the module x names; failing a module, the
//     method foo of the class x names; failing both, nothing (selector)
//   - foo(): the module's own foo, then the foo a from-import binds, then
//     the one foo of the repository (bare)
//
// Every answer is exactly one definition, or no edge. A class that the last
// two find is a constructor call (construct).
func (x *pyIndex) call(raw extract.RawEdge) (model.Edge, bool) {
	switch {
	case raw.Owner != "":
		// By the calling method's id and not by Owner, a class name: two
		// classes of one name, `if` and `else` each defining one, each have
		// their own methods. Only a method's body has self.
		return one(raw, x.method(x.classOf[raw.Source], raw.Name), model.ConfidenceExtracted)
	case raw.Receiver != "":
		return x.selector(raw)
	default:
		return x.bare(raw)
	}
}

// selector resolves x.foo(). A receiver that names a module decides there:
// a name the module neither defines nor passes on (offered) is not looked
// for anywhere else. One that names no module and no class -- an instance, a
// parameter, a module outside the repository -- gives no edge: a guess by
// the bare name foo would fill callers and blast with noise.
func (x *pyIndex) selector(raw extract.RawEdge) (model.Edge, bool) {
	if target, ok := x.moduleNamed(raw.File, raw.Receiver); ok {
		return x.construct(one(raw, x.offered(target, raw.Name), model.ConfidenceExtracted))
	}
	receiver, name := "", raw.Receiver
	if i := strings.LastIndexByte(raw.Receiver, '.'); i >= 0 {
		receiver, name = raw.Receiver[:i], raw.Receiver[i+1:]
	}
	class, ok := x.classNamed(raw.File, receiver, name, "")
	if !ok {
		return model.Edge{}, false
	}
	return one(raw, x.method(class, raw.Name), model.ConfidenceExtracted)
}

// bare resolves foo(): the module's own foo, then the definition a
// from-import binds foo to, re-exports followed, both extracted; failing
// both, the one definition of that name in all the Python files, inferred --
// shadowing could fool it.
//
// Three kinds of name are never guessed, because the file says where they
// come from, or that the answer is not one definition:
//
//   - a builtin: one repo function named open would otherwise collect every
//     open() of the repository. A module that defines or imports its own open
//     still finds it.
//   - a name whose from-import settles it (imported): every module it is
//     bound from lies outside the repository, `from json import load`, and
//     the one load of the repository is not json's; or the re-exports it
//     leads through end outside, go round in a circle or run too deep.
//   - a name whose from-import leads to two definitions: which one is bound
//     is a guess already, and the one definition of the name the file binds
//     it to (`from m import foo as bar`, and a bar elsewhere) is not it.
//
// A from-import whose chain ends in a file here that neither defines nor
// passes on the name -- a wildcard, an assignment -- is guessed: the one foo
// of the repository most likely is the foo that file hands on.
func (x *pyIndex) bare(raw extract.RawEdge) (model.Edge, bool) {
	if e, ok := one(raw, x.perFile[raw.File][raw.Name], model.ConfidenceExtracted); ok {
		return x.construct(e, ok)
	}
	nodes, settled := x.imported(raw.File, raw.Name)
	if len(nodes) > 0 || settled || pyBuiltin(raw.Name) {
		return x.construct(one(raw, nodes, model.ConfidenceExtracted))
	}
	return x.construct(one(raw, x.global[raw.Name], model.ConfidenceInferred))
}

// construct turns a call of a class into a call of its constructor: the
// class's own __init__, when it defines exactly one, and else the class
// itself. An __init__ inherited from a base is not the class's: a change to
// it is a change to the base. The edge goes to __init__ for blast's sake --
// contains is no walk edge, and an edge to the class alone would never reach
// the __init__ a change touches. Any other target passes unchanged, and so
// does a call that found nothing.
func (x *pyIndex) construct(e model.Edge, ok bool) (model.Edge, bool) {
	if inits := x.classes[e.Target].methods["__init__"]; len(inits) == 1 {
		e.Target = inits[0].ID
	}
	return e, ok
}

// method is the definitions of name in the first class that has any: class
// itself, then its bases breadth first, each class's bases in the order they
// are written. The first class that defines name decides even when it
// defines it twice -- it shadows every base, and its own two are for the
// caller to refuse as a guess. Each class is visited once, so a cycle of
// bases, `class A(B)` / `class B(A)`, ends the walk instead of looping.
func (x *pyIndex) method(class model.NodeID, name string) []model.Node {
	queue := []model.NodeID{class}
	visited := map[model.NodeID]bool{class: true}
	for len(queue) > 0 {
		c := x.classes[queue[0]]
		queue = queue[1:]
		if found := c.methods[name]; len(found) > 0 {
			return found
		}
		for _, b := range c.bases {
			if !visited[b] {
				visited[b] = true
				queue = append(queue, b)
			}
		}
	}
	return nil
}

// moduleNamed is the file of the module a dotted name stands for in file:
//
//   - `import a.b as c` binds c to a.b
//   - `import a.b` binds a, and a.b with it: the name is the path or a
//     leading part of it
//   - `from X import n [as k]` binds n (or k) to the submodule X.n, when
//     X.n is one
//
// A binding that names no file here is passed over for the next one:
// `try: import fast as impl` / `except ImportError: from . import slow as
// impl` binds impl twice, and the one that is a file here is the one the
// graph can point at.
func (x *pyIndex) moduleNamed(file, name string) (string, bool) {
	for _, imp := range x.importsOf[file] {
		var spec string
		switch {
		case imp.Name == "" && imp.Alias == name:
			spec = imp.Path
		case imp.Name == "" && imp.Alias == "" && (imp.Path == name || strings.HasPrefix(imp.Path, name+".")):
			spec = name
		case imp.Name != "" && boundName(imp) == name:
			spec = pyJoin(imp.Path, imp.Name)
		default:
			continue
		}
		if target, ok := x.module(file, spec); ok {
			return target, true
		}
	}
	return "", false
}

// maxReExports is how many re-exports a name is followed through before the
// resolver gives up on it; lib/__init__.py passing on the Base of
// lib/impl.py is one. A package's chain is one or two long in practice, and
// the bound keeps a pathological one from costing more.
const maxReExports = 8

// imported is the definitions a `from m import n [as k]` in file binds the
// bare name to: what m offers under n (offered), its own definitions or the
// ones it passes on. A wildcard binds no name this can see. A binding from a
// module that is no file here is passed over for the next one, as in
// moduleNamed, and the first binding to a file here decides.
//
// settled reports that the answer is final, and nothing is to be guessed:
// every module file binds the name from lies outside the repository
// (outside) -- the standard library, an installed package -- or the
// re-exports it leads through end outside, go round in a circle, or run
// deeper than maxReExports. A binding from a module of the repository that
// no source root reaches settles nothing.
func (x *pyIndex) imported(file, name string) (nodes []model.Node, settled bool) {
	return x.bound(file, name, 0, map[[2]string]bool{})
}

// bound is imported at a depth of re-exports, with the files and names the
// chain has passed.
func (x *pyIndex) bound(file, name string, depth int, seen map[[2]string]bool) ([]model.Node, bool) {
	outside, inside := false, false
	for _, imp := range x.importsOf[file] {
		if imp.Name == "" || boundName(imp) != name {
			continue
		}
		// The spec is relative to file, the one that holds the import, and
		// the name looked for there is the one imported, not its alias.
		if target, ok := x.module(file, imp.Path); ok {
			return x.definitions(target, imp.Name, depth, seen)
		}
		if x.outside(imp.Path) {
			outside = true
		} else {
			inside = true
		}
	}
	return nil, outside && !inside
}

// outside reports that a module spec that names no file here comes from
// outside the repository: its first dotted segment is the name of no
// directory and no module among the Python files (pyNames). `from json import
// load` is outside; `from apps.users import helper`, with a directory apps
// somewhere, is a module of the repository that no source root reaches -- a
// project nested deeper than a root, a module path two roots made ambiguous
// for a file under neither, a sibling a script imports by its bare name --
// and not outside. A relative spec names a place in the repository by its
// form and is never outside.
func (x *pyIndex) outside(spec string) bool {
	if strings.HasPrefix(spec, ".") {
		return false
	}
	first, _, _ := strings.Cut(spec, ".")
	return !x.names[first]
}

// offered is what module file target offers under name, as definitions.
func (x *pyIndex) offered(target, name string) []model.Node {
	nodes, _ := x.definitions(target, name, 0, map[[2]string]bool{})
	return nodes
}

// definitions is what module file target offers under name: its own
// module-level functions and classes of that name, and failing those, what
// target's own from-import of name binds -- a re-export, lib/__init__.py
// passing on the Base of lib/impl.py with `from .impl import Base` --
// followed from file to file. The definition at the end counts as much as
// one imported directly: the file that re-exports it says where it is.
//
// A chain that comes back to a file and name it passed, or that would take
// more than maxReExports steps, ends with nothing, settled. One that ends in
// a file neither defining nor binding the name ends with nothing, unsettled:
// a wildcard or an assignment may pass it on, and this index sees neither.
func (x *pyIndex) definitions(target, name string, depth int, seen map[[2]string]bool) ([]model.Node, bool) {
	if own := x.perFile[target][name]; len(own) > 0 {
		return own, false
	}
	key := [2]string{target, name}
	if seen[key] || depth == maxReExports {
		return nil, true
	}
	seen[key] = true
	return x.bound(target, name, depth+1, seen)
}

// boundName is the name a from-import binds in its file: its alias, or the
// name it imports.
func boundName(imp extract.Import) string {
	if imp.Alias != "" {
		return imp.Alias
	}
	return imp.Name
}

// pyJoin is the module a name of `from module import name` would be, were it
// a submodule: "pkg" and "a" make "pkg.a"; a module of dots alone takes the
// name without a separator, "." and "a" make ".a".
func pyJoin(module, name string) string {
	if strings.HasSuffix(module, ".") {
		return module + name
	}
	return module + "." + name
}

// onlyClass is the id of the one class among nodes; false when there is none,
// or more than one, or when the one is except. except is counted, not
// skipped: beside another class of its name it leaves the two a guess.
func onlyClass(nodes []model.Node, except model.NodeID) (model.NodeID, bool) {
	var found []model.NodeID
	for _, n := range nodes {
		if n.Kind == "class" {
			found = append(found, n.ID)
		}
	}
	if len(found) != 1 || found[0] == except {
		return "", false
	}
	return found[0], true
}

// pyBuiltin reports whether name is one of Python's builtins: every public
// name of the builtins module of CPython 3.14 on Windows, those site adds
// (exit, help, license) and WindowsError included. A switch and not a map:
// it costs no work at the start of the binary.
func pyBuiltin(name string) bool {
	switch name {
	case "ArithmeticError", "AssertionError", "AttributeError", "BaseException", "BaseExceptionGroup",
		"BlockingIOError", "BrokenPipeError", "BufferError", "BytesWarning", "ChildProcessError",
		"ConnectionAbortedError", "ConnectionError", "ConnectionRefusedError", "ConnectionResetError",
		"DeprecationWarning", "EOFError", "Ellipsis", "EncodingWarning", "EnvironmentError", "Exception",
		"ExceptionGroup", "False", "FileExistsError", "FileNotFoundError", "FloatingPointError",
		"FutureWarning", "GeneratorExit", "IOError", "ImportError", "ImportWarning", "IndentationError",
		"IndexError", "InterruptedError", "IsADirectoryError", "KeyError", "KeyboardInterrupt",
		"LookupError", "MemoryError", "ModuleNotFoundError", "NameError", "None", "NotADirectoryError",
		"NotImplemented", "NotImplementedError", "OSError", "OverflowError", "PendingDeprecationWarning",
		"PermissionError", "ProcessLookupError", "PythonFinalizationError", "RecursionError",
		"ReferenceError", "ResourceWarning", "RuntimeError", "RuntimeWarning", "StopAsyncIteration",
		"StopIteration", "SyntaxError", "SyntaxWarning", "SystemError", "SystemExit", "TabError",
		"TimeoutError", "True", "TypeError", "UnboundLocalError", "UnicodeDecodeError",
		"UnicodeEncodeError", "UnicodeError", "UnicodeTranslateError", "UnicodeWarning", "UserWarning",
		"ValueError", "Warning", "WindowsError", "ZeroDivisionError",
		"abs", "aiter", "all", "anext", "any", "ascii", "bin", "bool", "breakpoint", "bytearray", "bytes",
		"callable", "chr", "classmethod", "compile", "complex", "copyright", "credits", "delattr", "dict",
		"dir", "divmod", "enumerate", "eval", "exec", "exit", "filter", "float", "format", "frozenset",
		"getattr", "globals", "hasattr", "hash", "help", "hex", "id", "input", "int", "isinstance",
		"issubclass", "iter", "len", "license", "list", "locals", "map", "max", "memoryview", "min",
		"next", "object", "oct", "open", "ord", "pow", "print", "property", "quit", "range", "repr",
		"reversed", "round", "set", "setattr", "slice", "sorted", "staticmethod", "str", "sum", "super",
		"tuple", "type", "vars", "zip":
		return true
	}
	return false
}
