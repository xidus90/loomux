package resolve_test

import (
	"fmt"
	"path"
	"sort"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/extract/rust"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// rustTree resolves a tree as a graph build over the index does: every file
// a registered extractor claims, handed over in reverse byte order, and the
// Cargo manifests read from their text alone (ModulesFrom), so the crate
// roots Cargo finds by convention come from the files themselves.
func rustTree(t *testing.T, tree map[string]string) (*model.Graph, []resolve.Unresolved) {
	t.Helper()
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	bodies := map[string]string{}
	var files []extract.Result
	for _, p := range paths {
		if resolve.ModuleFile(p) {
			bodies[p] = tree[p]
			continue
		}
		lang, ok := all.For(p)
		if !ok {
			if path.Base(p) == "project.godot" {
				continue
			}
			t.Fatalf("no extractor claims %s", p)
		}
		r, err := lang.File(p, tree[p])
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	mods, err := resolve.ModulesFrom(bodies)
	if err != nil {
		t.Fatal(err)
	}
	g, un := resolve.GraphWith(files, mods, stamp)
	return validated(t, g), un
}

const appManifest = "[package]\nname = \"app\"\n"

func TestRustModuleTree(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml": appManifest,
		"src/lib.rs": "mod a;\nmod b;\n#[path = \"other/c_impl.rs\"]\nmod c;\nmod missing;\n" +
			"pub fn root() { a::fa(); b::fb(); c::fc(); a::x::fx(); }\n",
		// a.rs is no mod.rs: its own modules sit in a/.
		"src/a.rs":            "pub mod x;\npub fn fa() {}\n",
		"src/a/x.rs":          "pub fn fx() { super::fa(); crate::root(); }\n",
		"src/b/mod.rs":        "pub fn fb() {}\n",
		"src/other/c_impl.rs": "pub fn fc() {}\n",
		// Reached by no tree: self:: is the file, super:: and crate:: nothing.
		"src/stray.rs": "pub fn s() { self::t(); super::fa(); crate::root(); }\nfn t() {}\n",
		// Roots by convention: a binary in a directory with a module of its
		// own, and an integration test; both name the library by its name.
		"src/bin/tool/main.rs":   "mod helper;\nfn main() { helper::h(); app::root(); crate::helper::h(); }\n",
		"src/bin/tool/helper.rs": "pub fn h() {}\n",
		"tests/it.rs":            "#[test]\nfn t() { app::a::fa(); }\n",
	})
	wantEdges(t, g,
		"src/a.rs -imports-> src/a/x.rs extracted",
		"src/a/x.rs#fx -calls-> src/a.rs#fa extracted",
		"src/a/x.rs#fx -calls-> src/lib.rs#root extracted",
		"src/bin/tool/main.rs -imports-> src/bin/tool/helper.rs extracted",
		"src/bin/tool/main.rs#main -calls-> src/bin/tool/helper.rs#h extracted",
		"src/bin/tool/main.rs#main -calls-> src/bin/tool/helper.rs#h extracted",
		"src/bin/tool/main.rs#main -calls-> src/lib.rs#root extracted",
		"src/lib.rs -imports-> src/a.rs extracted",
		"src/lib.rs -imports-> src/b/mod.rs extracted",
		"src/lib.rs -imports-> src/other/c_impl.rs extracted",
		"src/lib.rs#root -calls-> src/a.rs#fa extracted",
		"src/lib.rs#root -calls-> src/a/x.rs#fx extracted",
		"src/lib.rs#root -calls-> src/b/mod.rs#fb extracted",
		"src/lib.rs#root -calls-> src/other/c_impl.rs#fc extracted",
		"src/stray.rs#s -calls-> src/stray.rs#t extracted",
		"tests/it.rs#t -calls-> src/a.rs#fa extracted",
	)
	if lineOf(t, g, "src/lib.rs", "src/other/c_impl.rs", model.RelationImports) != 4 {
		t.Error("mod c; makes its imports edge at its own line")
	}
	found := false
	for _, n := range g.Nodes {
		found = found || n.ID == "src/stray.rs#t"
	}
	if !found {
		t.Error("a file no tree reaches keeps its nodes")
	}
}

// A file no tree reaches finds its modules where its name puts them: a.rs
// below a/, a mod.rs beside itself. A use of nothing (`use {*};`) binds
// nothing and breaks nothing.
func TestRustStrayFileModules(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml":   appManifest,
		"src/lib.rs":   "",
		"src/a.rs":     "use {*};\nuse ::*;\nmod x;\nfn f() { x::g(); }\n",
		"src/x.rs":     "pub fn g() {}\n",
		"src/a/x.rs":   "pub fn g() {}\n",
		"src/m/mod.rs": "mod y;\nfn f() { y::h(); }\n",
		"src/m/y.rs":   "pub fn h() {}\n",
	})
	wantEdges(t, g,
		"src/a.rs -imports-> src/a/x.rs extracted",
		"src/a.rs#f -calls-> src/a/x.rs#g extracted",
		"src/m/mod.rs -imports-> src/m/y.rs extracted",
		"src/m/mod.rs#f -calls-> src/m/y.rs#h extracted",
	)
}

// A library at [lib] path is what the crate name reaches.
func TestRustLibraryAtLibPath(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml":    "[package]\nname = \"app\"\n[lib]\npath = \"src/engine.rs\"\n",
		"src/engine.rs": "pub fn start() {}\n",
		"tests/it.rs":   "fn t() { app::start(); }\n",
	})
	wantCalls(t, g, "tests/it.rs#t", "src/engine.rs#start extracted")
}

// A name two globs offer stays ambiguous whichever lookup runs first, also
// when the globs go round a circle: the crate root and its prelude glob each
// other, the root also globs a and the prelude c, and a and c both define x.
// Looking into the prelude first must not leave the root's x cut down to a's.
func TestRustGlobCircleKeepsAmbiguity(t *testing.T) {
	tree := map[string]string{
		"Cargo.toml":     appManifest,
		"src/lib.rs":     "pub mod prelude;\npub mod a;\npub mod b;\npub mod c;\npub use prelude::*;\npub use a::*;\n",
		"src/prelude.rs": "pub use crate::*;\npub use crate::c::*;\n",
		"src/a.rs":       "pub fn x() {}\n",
		"src/c.rs":       "pub fn x() {}\n",
		"src/b.rs":       "fn go() { crate::prelude::x(); crate::x(); }\nfn back() { crate::x(); crate::prelude::x(); }\n",
	}
	g, _ := rustTree(t, tree)
	wantCalls(t, g, "src/b.rs#go")
	wantCalls(t, g, "src/b.rs#back")
	// The same files in byte order.
	var files []extract.Result
	for _, p := range []string{"src/a.rs", "src/b.rs", "src/c.rs", "src/lib.rs", "src/prelude.rs"} {
		r, err := rust.File(p, tree[p])
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	mods, _ := resolve.ModulesFrom(map[string]string{"Cargo.toml": appManifest})
	g2, _ := resolve.GraphWith(files, mods, stamp)
	wantCalls(t, g2, "src/b.rs#go")
	wantCalls(t, g2, "src/b.rs#back")
}

// Modules that each glob all the others form one dense circle. The tables
// fill in polynomial time, so the build finishes, a name none of them
// defines gives no edge and one module's function is still found.
func TestRustGlobCliqueIsBounded(t *testing.T) {
	for _, n := range []int{12, 30} {
		tree := map[string]string{"Cargo.toml": appManifest}
		lib := ""
		var imports []string
		for i := range n {
			lib += fmt.Sprintf("pub mod m%d;\n", i)
			imports = append(imports, fmt.Sprintf("src/m%d.rs", i))
			body := ""
			for j := range n {
				if j != i {
					body += fmt.Sprintf("pub use crate::m%d::*;\n", j)
				}
			}
			tree[fmt.Sprintf("src/m%d.rs", i)] = body
		}
		tree["src/lib.rs"] = lib + "use m0::Gone;\ntrait T: m0::Gone {}\nfn go() { m0::nothing(); m0::here(); }\n"
		tree["src/m5.rs"] += "pub fn here() {}\n"
		start := time.Now()
		g, un := rustTree(t, tree)
		if d := time.Since(start); d > 5*time.Second {
			t.Fatalf("resolving the clique of %d took %v", n, d)
		}
		wantCalls(t, g, "src/lib.rs#go", "src/m5.rs#here extracted")
		sort.Strings(imports)
		wantTargets(t, g, "src/lib.rs", model.RelationImports, imports...)
		wantUnresolved(t, un, "m0::Gone none")
	}
}

// c binds a over p's globs to the module q (s names it so), and q offers a
// as two functions: a::a() names neither for sure.
func TestRustGlobCircleAliasStaysAmbiguous(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml": appManifest,
		"src/lib.rs": "pub mod c;\npub mod p;\npub mod q;\npub mod s;\npub mod t;\npub mod u;\n",
		"src/c.rs":   "use crate::p::*;\npub fn go() { a::a(); }\n",
		"src/p.rs":   "pub use crate::q::*;\npub use crate::s::*;\npub use crate::u::*;\n",
		"src/q.rs":   "pub use crate::p::*;\npub use crate::t::*;\n",
		"src/s.rs":   "pub use crate::q as a;\n",
		"src/t.rs":   "pub fn a() {}\n",
		"src/u.rs":   "pub fn a() {}\n",
	})
	wantCalls(t, g, "src/c.rs#go")
}

// The same alias read twice in one path, a::a::f(): both segments are q.
func TestRustGlobCircleAliasTwice(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml": appManifest,
		"src/lib.rs": "pub mod c;\npub mod p;\npub mod q;\npub mod s;\n",
		"src/c.rs":   "use crate::p::*;\npub fn go() { a::a::f(); }\n",
		"src/p.rs":   "pub use crate::q::*;\npub use crate::s::*;\n",
		"src/q.rs":   "pub use crate::p::*;\npub fn f() {}\n",
		"src/s.rs":   "pub use crate::q as a;\n",
	})
	wantCalls(t, g, "src/c.rs#go", "src/q.rs#f extracted")
}

// a offers x from two globs, b's and d's (through c). crate::a::x names
// both modules, whatever order the tables fill in: a use through it and a
// glob of it stand for both f, and a path through it for no item at all,
// though only b's x has g.
func TestRustAmbiguousSegmentStaysAmbiguous(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml": appManifest,
		"src/lib.rs": "pub mod a;\npub mod b;\npub mod c;\npub mod d;\npub mod z;\npub mod z2;\npub mod z3;\n" +
			"fn go() { crate::z::f(); crate::z2::f(); crate::z3::g(); crate::a::x::g(); }\n",
		"src/a.rs":  "pub use crate::b::*;\npub use crate::c::*;\n",
		"src/b.rs":  "pub mod x {\n    pub fn f() {}\n    pub fn g() {}\n}\n",
		"src/c.rs":  "pub use crate::d::*;\n",
		"src/d.rs":  "pub mod x {\n    pub fn f() {}\n}\n",
		"src/z.rs":  "pub use crate::a::x::f;\n",
		"src/z2.rs": "pub use crate::a::x::*;\n",
		"src/z3.rs": "pub use crate::a::x::g;\n",
	})
	wantCalls(t, g, "src/lib.rs#go")
}

// selfSustaining is a crate where p globs crate::a::x, which names b's x and
// d's x, and p and q glob each other: f reaches p from both modules and
// through q's copy of p. c, the module that leads to d, is named by the
// caller.
func selfSustaining(c string) map[string]string {
	return map[string]string{
		"Cargo.toml": appManifest,
		"src/lib.rs": "pub mod a;\npub mod b;\npub mod " + c + ";\npub mod d;\npub mod p;\npub mod q;\n" +
			"fn go() { crate::p::f(); }\n",
		"src/a.rs":         "pub use crate::b::*;\npub use crate::" + c + "::*;\n",
		"src/b.rs":         "pub mod x {\n    pub fn f() {}\n}\n",
		"src/" + c + ".rs": "pub use crate::d::*;\n",
		"src/d.rs":         "pub mod x {\n    pub fn f() {}\n}\n",
		"src/p.rs":         "pub use crate::a::x::*;\npub use crate::q::*;\n",
		"src/q.rs":         "pub use crate::p::*;\n",
	}
}

// A candidate one order of filling the tables sees first does not outlive
// the second in a circle, and the names of the modules change nothing.
func TestRustGlobCircleKeepsNoEarlyCandidate(t *testing.T) {
	for _, c := range []string{"c", "_c"} {
		g, _ := rustTree(t, selfSustaining(c))
		wantCalls(t, g, "src/lib.rs#go")
	}
}

// y binds x to both modules crate::a::x names; z's glob of it offers d's f
// beside e's, and f stays ambiguous.
func TestRustGlobThroughAmbiguousUse(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml": appManifest,
		"src/lib.rs": "pub mod a;\npub mod b;\npub mod c;\npub mod d;\npub mod e;\npub mod y;\npub mod z;\n" +
			"fn go() { crate::z::f(); }\n",
		"src/a.rs": "pub use crate::b::*;\npub use crate::c::*;\n",
		"src/b.rs": "pub mod x {\n    pub fn g() {}\n}\n",
		"src/c.rs": "pub use crate::d::*;\n",
		"src/d.rs": "pub mod x {\n    pub fn f() {}\n}\n",
		"src/e.rs": "pub fn f() {}\n",
		"src/y.rs": "pub use crate::a::x;\n",
		"src/z.rs": "pub use crate::y::x::*;\npub use crate::e::*;\n",
	})
	wantCalls(t, g, "src/lib.rs#go")
}

// y binds h through the ambiguous crate::a::x, where neither module has h;
// z's glob of y still offers that h beside e's, so h stays ambiguous.
func TestRustGlobOffersAnAmbiguousNothing(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml": appManifest,
		"src/lib.rs": "pub mod a;\npub mod b;\npub mod c;\npub mod d;\npub mod e;\npub mod y;\npub mod z;\n" +
			"fn go() { crate::z::h(); }\n",
		"src/a.rs": "pub use crate::b::*;\npub use crate::c::*;\n",
		"src/b.rs": "pub mod x {\n    pub fn f() {}\n}\n",
		"src/c.rs": "pub use crate::d::*;\n",
		"src/d.rs": "pub mod x {\n    pub fn f() {}\n}\n",
		"src/e.rs": "pub fn h() {}\n",
		"src/y.rs": "pub use crate::a::x::h;\n",
		"src/z.rs": "pub use crate::y::*;\npub use crate::e::*;\n",
	})
	wantCalls(t, g, "src/lib.rs#go")
}

// Inline modules of the root each write `use crate::f;` for the f the root
// takes from a glob. Hoisted to the file, every such use reads past all of
// them, so they name a's f, and so does the root; one alone does the same.
func TestRustSelfReadingUsesReadPastEachOther(t *testing.T) {
	for _, n := range []int{1, 2, 3} {
		lib := "pub mod a;\npub mod m;\npub mod o;\npub use crate::a::*;\npub fn go() { f(); }\n"
		for i := range n {
			lib += fmt.Sprintf("mod t%d {\n    use crate::f;\n    pub fn c() { f(); }\n}\n", i)
		}
		g, _ := rustTree(t, map[string]string{
			"Cargo.toml": appManifest,
			"src/lib.rs": lib,
			"src/a.rs":   "pub fn f() {}\n",
			"src/o.rs":   "pub fn o() { crate::f(); }\n",
			// In a module file the extractor leaves super:: past the file.
			"src/m.rs": "pub use crate::a::*;\n" +
				"mod t1 {\n    use super::super::m::f;\n    pub fn c() { f(); }\n}\n" +
				"mod t2 {\n    use crate::m::f;\n}\n",
		})
		wantCalls(t, g, "src/o.rs#o", "src/a.rs#f extracted")
		wantCalls(t, g, "src/lib.rs#go", "src/a.rs#f extracted")
		wantCalls(t, g, "src/lib.rs#t0::c", "src/a.rs#f extracted")
		wantCalls(t, g, "src/m.rs#t1::c", "src/a.rs#f extracted")
	}
}

// A self-reading use beside an explicit use of the name: it reads past the
// self-reading uses alone, so both name b's f over the glob's.
func TestRustSelfReadingUseBesideAnExplicitUse(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml": appManifest,
		"src/lib.rs": "pub mod a;\npub mod b;\npub use crate::a::*;\nuse crate::b::f;\npub fn go() { f(); }\n" +
			"mod t {\n    use crate::f;\n    pub fn c() { f(); }\n}\n",
		"src/a.rs": "pub fn f() {}\n",
		"src/b.rs": "pub fn f() {}\n",
	})
	wantCalls(t, g, "src/lib.rs#go", "src/b.rs#f extracted")
	wantCalls(t, g, "src/lib.rs#t::c", "src/b.rs#f extracted")
}

// The known gap: `use me::f` reaches the root through an alias, so it is no
// self-reading use and reads t's use, which reads it back. Both stay empty
// and shadow the glob: no edge, where a's f is right.
func TestRustAliasedSelfUseLosesTheGlob(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml": appManifest,
		"src/lib.rs": "pub mod a;\npub use crate::a::*;\nuse crate as me;\nuse me::f;\npub fn go() { f(); }\n" +
			"mod t {\n    use crate::f;\n}\n",
		"src/a.rs": "pub fn f() {}\n",
	})
	wantCalls(t, g, "src/lib.rs#go")
}

// A glob that offers a module under the crate's own name leaves the name
// ambiguous: whether the crate counts is read from definitions and uses
// alone.
func TestRustCrateNameBesideAGlob(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml":  appManifest,
		"src/lib.rs":  "pub fn root() {}\n",
		"src/main.rs": "mod m;\nuse m::*;\nfn main() { app::root(); }\n",
		"src/m.rs":    "pub mod app {\n    pub fn root() {}\n}\n",
	})
	wantCalls(t, g, "src/main.rs#main")
}

// A crate root another tree reaches first belongs to that tree.
func TestRustRootReachedByTheLibrary(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml":  appManifest,
		"src/lib.rs":  "#[path = \"main.rs\"]\nmod entry;\nfn helper() {}\n",
		"src/main.rs": "pub fn run() { crate::helper(); }\n",
	})
	wantCalls(t, g, "src/main.rs#run", "src/lib.rs#helper extracted")
}

// mod x; in an inline module sits in that module's directory, and in a
// non-mod.rs file below the file's own name.
func TestRustModuleInInlineModule(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml":        appManifest,
		"src/lib.rs":        "mod outer { pub mod deep; }\nmod a;\nfn f() { outer::deep::d(); a::inner::w::fw(); }\n",
		"src/outer/deep.rs": "pub fn d() { self::e(); super::super::f(); }\nfn e() {}\n",
		"src/a.rs":          "pub mod inner { pub mod w; }\n",
		"src/a/inner/w.rs":  "pub fn fw() {}\n",
	})
	wantEdges(t, g,
		"src/a.rs -imports-> src/a/inner/w.rs extracted",
		"src/lib.rs -imports-> src/a.rs extracted",
		"src/lib.rs -imports-> src/outer/deep.rs extracted",
		"src/lib.rs#f -calls-> src/a/inner/w.rs#fw extracted",
		"src/lib.rs#f -calls-> src/outer/deep.rs#d extracted",
		"src/outer/deep.rs#d -calls-> src/lib.rs#f extracted",
		"src/outer/deep.rs#d -calls-> src/outer/deep.rs#e extracted",
	)
}

func TestRustPathsAcrossAWorkspace(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml":       "[workspace]\nmembers = [\"core\", \"app\"]\n",
		"core/Cargo.toml":  "[package]\nname = \"game-core\"\n",
		"core/src/lib.rs":  "pub mod util;\npub fn c() {}\n",
		"core/src/util.rs": "pub fn u() {}\n",
		"app/Cargo.toml": "[package]\nname = \"app\"\n[dependencies]\n" +
			"game-core = { path = \"../core\" }\nserde = \"1\"\n",
		"app/src/main.rs": "extern crate game_core;\nuse ::game_core as gc;\nuse game_core::util;\nuse serde::Serialize;\nuse std::fmt;\n" +
			"fn main() { util::u(); ::game_core::c(); gc::c(); serde::to_string(); fmt::format(); ::nope::f(); }\n",
	})
	wantEdges(t, g,
		"app/src/main.rs -imports-> core/src/lib.rs extracted",
		"app/src/main.rs -imports-> core/src/util.rs extracted",
		"app/src/main.rs#main -calls-> core/src/lib.rs#c extracted",
		"app/src/main.rs#main -calls-> core/src/lib.rs#c extracted",
		"app/src/main.rs#main -calls-> core/src/util.rs#u extracted",
		"core/src/lib.rs -imports-> core/src/util.rs extracted",
	)
}

// A name binds to a definition of its own module, then a use, then a glob;
// two hits in one step give no edge.
func TestRustBindingOrder(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml": appManifest,
		"src/lib.rs": "mod a;\nmod b;\nmod c;\nmod d;\n",
		"src/a.rs":   "pub fn f() {}\npub fn g() {}\npub fn h() {}\n",
		"src/b.rs":   "pub fn f() {}\npub fn g() {}\npub fn h() {}\n",
		"src/c.rs":   "use crate::a::*;\nuse crate::b::g;\nfn f() {}\nfn call() { f(); g(); h(); }\n",
		"src/d.rs":   "use crate::a::*;\nuse super::b::*;\nfn call() { f(); }\n",
	})
	wantCalls(t, g, "src/c.rs#call", "src/a.rs#h extracted", "src/b.rs#g extracted", "src/c.rs#f extracted")
	wantCalls(t, g, "src/d.rs#call")
	wantTargets(t, g, "src/c.rs", model.RelationImports, "src/a.rs", "src/b.rs")
}

func TestRustReexportChain(t *testing.T) {
	g, _ := rustTree(t, map[string]string{
		"Cargo.toml":   appManifest,
		"src/lib.rs":   "mod inner;\nmod mid;\nmod api;\nmod user;\nmod cyc1;\nmod cyc2;\nmod loops;\n",
		"src/inner.rs": "pub struct Thing;\nimpl Thing {\n    pub fn new() -> Self { Thing }\n}\n",
		"src/mid.rs":   "pub use crate::inner::*;\n",
		"src/api.rs":   "pub use crate::mid::Thing as Item;\n",
		"src/user.rs":  "use crate::api::Item;\nfn go() { Item::new(); }\n",
		// A circle of re-exports ends with nothing, and ends.
		"src/cyc1.rs":  "pub use crate::cyc2::Loop;\npub use crate::cyc2::*;\n",
		"src/cyc2.rs":  "pub use crate::cyc1::Loop;\npub use crate::cyc1::*;\n",
		"src/loops.rs": "use crate::cyc1::Loop;\nfn go() { Loop::new(); crate::cyc2::Gone::new(); }\n",
	})
	wantCalls(t, g, "src/user.rs#go", "src/inner.rs#Thing.new extracted")
	wantTargets(t, g, "src/user.rs", model.RelationImports, "src/inner.rs")
	wantCalls(t, g, "src/loops.rs#go")
}

// rustShapes is a crate whose type, impl and callers sit in three files.
func rustShapes() map[string]string {
	return map[string]string{
		"Cargo.toml": appManifest,
		"src/lib.rs": "mod shapes;\nmod foo_impl;\nmod run;\nmod twins;\n" +
			"macro_rules! my_macro { () => {} }\n",
		"src/shapes.rs": "pub struct Foo;\npub struct Pt(pub i32);\npub enum Kind { A(i32) }\n" +
			"pub trait Greet {\n    fn hello(&self) {}\n    fn name(&self);\n}\n" +
			"pub trait Loud: Greet + std::fmt::Debug {\n    fn shout(&self) { self.hello(); }\n}\n",
		"src/foo_impl.rs": "use crate::shapes::{Foo, Greet, Loud};\n" +
			"impl Foo {\n" +
			"    pub fn new() -> Self { Self::build() }\n" +
			"    fn build() -> Self { Foo }\n" +
			"    pub fn run(&self) { self.step(); self.hello(); self.absent(); self.fmt(); }\n" +
			"    fn step(&self) {}\n" +
			"    fn flag(&self) -> bool { true }\n" +
			"}\n" +
			"impl Greet for Foo {\n    fn name(&self) {}\n}\nimpl Loud for Foo {}\n" +
			// Two trait impls with a method of one name: neither x.fmt(),
			// self.fmt() nor Foo::fmt() can tell them apart.
			"impl std::fmt::Display for Foo {\n    fn fmt(&self) {}\n}\n" +
			"impl std::fmt::Debug for Foo {\n    fn fmt(&self) {}\n}\n",
		"src/run.rs": "use crate::shapes::{Foo, Pt, Kind};\n" +
			"macro_rules! local_m { () => {} }\n" +
			"fn go(x: Foo) {\n" +
			"    Foo::new();\n" +
			"    Foo::Inner::make();\n" +
			"    Pt(1);\n" +
			"    Kind::A(2);\n" +
			"    Some(3);\n" +
			"    x.run();\n" +
			"    x.name();\n" +
			"    x.unknown();\n" +
			"    x.fmt();\n" +
			"    Foo::fmt(&x);\n" +
			"    assert!(helper(1));\n" +
			"    assert!(x.flag());\n" +
			"    local_m!();\n" +
			"    my_macro!();\n" +
			"    println!(\"{}\", 1);\n" +
			"    std::println!(\"{}\", 1);\n" +
			"}\n" +
			"fn helper(_: i32) -> bool { true }\n",
		// Two types named A in one crate: whose methods make and other are
		// cannot be told. Two glob-imported traits Tr leave the impl
		// unresolved.
		"src/twins.rs": "mod m1 { pub struct A; }\nmod m2 {\n    pub struct A;\n" +
			"    impl A {\n        pub fn make() { Self::other() }\n        fn other() {}\n    }\n}\n" +
			"mod t1 { pub trait Tr {} }\nmod t2 { pub trait Tr {} }\nuse t1::*;\nuse t2::*;\nimpl Tr for m1::A {}\n" +
			"fn go() { m1::A::make(); }\n",
	}
}

func TestRustCalls(t *testing.T) {
	g, _ := rustTree(t, rustShapes())
	wantCalls(t, g, "src/run.rs#go",
		"src/foo_impl.rs#Foo.flag inferred",
		"src/foo_impl.rs#Foo.new extracted",
		"src/foo_impl.rs#Foo.run inferred",
		"src/lib.rs#my_macro inferred",
		"src/run.rs#helper inferred",
		"src/run.rs#local_m extracted",
		"src/shapes.rs#Kind extracted",
		"src/shapes.rs#Pt extracted",
	)
	wantCalls(t, g, "src/foo_impl.rs#Foo.new", "src/foo_impl.rs#Foo.build extracted")
	// self.f(): the type's own method, then a default of a trait it implements.
	wantCalls(t, g, "src/foo_impl.rs#Foo.run", "src/foo_impl.rs#Foo.step extracted", "src/shapes.rs#Greet.hello extracted")
	// A trait's default reaches a default of its supertrait.
	wantCalls(t, g, "src/shapes.rs#Loud.shout", "src/shapes.rs#Greet.hello extracted")
	wantCalls(t, g, "src/twins.rs#go")
	wantCalls(t, g, "src/twins.rs#m2::A.make")
}

func TestRustTraitRelations(t *testing.T) {
	g, un := rustTree(t, rustShapes())
	wantTargets(t, g, "src/shapes.rs#Foo", model.RelationImplements, "src/shapes.rs#Greet", "src/shapes.rs#Loud")
	wantTargets(t, g, "src/shapes.rs#Loud", model.RelationExtends, "src/shapes.rs#Greet")
	wantUnresolved(t, un, "Tr ambiguous", "std::fmt::Debug none", "std::fmt::Display none", "std::fmt::Debug none")
}
