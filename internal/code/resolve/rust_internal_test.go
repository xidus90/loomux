package resolve

import (
	"fmt"
	"maps"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/rust"
)

// pcSeg is a path segment: a keyword or item name, or the module mod (>= 0),
// which a crate renders under whatever name it is given.
type pcSeg struct {
	mod  int
	name string
}

type pcUse struct {
	path  []pcSeg
	glob  bool
	alias string
	pub   bool
}

// pcFile is a file module: its uses, its functions, an inline module x with
// functions of its own, and callers with the paths they call.
type pcFile struct {
	uses    []pcUse
	fns     []string
	hasX    bool
	xFns    []string
	callers []string
	calls   map[string][][]pcSeg
}

// pcCrate is files m0..m(k-1) and the crate root lib, at index k.
type pcCrate struct {
	k     int
	files []*pcFile
}

// randomCrate draws a crate of sibling modules that define f and g, some in
// an inline module x, glob each other and their x, re-export and rename
// them, and call them by every form of path.
func randomCrate(rng *rand.Rand) *pcCrate {
	k := 3 + rng.Intn(3)
	c := &pcCrate{k: k}
	mod := func() pcSeg { return pcSeg{mod: rng.Intn(k)} }
	nm := func() string { return []string{"f", "g"}[rng.Intn(2)] }
	kw := func(s string) pcSeg { return pcSeg{mod: -1, name: s} }
	item := kw
	for fi := range k {
		f := &pcFile{calls: map[string][][]pcSeg{}}
		for range 1 + rng.Intn(7) {
			switch rng.Intn(13) {
			case 0, 1:
				f.uses = append(f.uses, pcUse{path: []pcSeg{kw("crate"), mod()}, glob: true, pub: true})
			case 2:
				f.uses = append(f.uses, pcUse{path: []pcSeg{kw("crate"), mod()}, glob: true})
				f.uses = append(f.uses, pcUse{path: []pcSeg{kw("crate"), mod(), item(nm())}})
			case 3:
				f.uses = append(f.uses, pcUse{path: []pcSeg{kw("crate"), mod(), item(nm())}, pub: true})
			case 4:
				f.uses = append(f.uses, pcUse{path: []pcSeg{kw("crate"), mod(), item("f")}, alias: "g", pub: true})
				// Uses that read the file's own name back, as inline modules
				// hoisted to the file write them.
				n := nm()
				for range rng.Intn(3) {
					f.uses = append(f.uses, pcUse{path: []pcSeg{kw("crate"), {mod: fi}, item(n)}, pub: rng.Intn(2) == 0})
				}
			case 5:
				f.uses = append(f.uses, pcUse{path: []pcSeg{kw("crate"), mod()}, pub: true})
			case 6:
				f.uses = append(f.uses, pcUse{path: []pcSeg{kw("crate"), mod(), item("x")}, glob: true, pub: true})
			case 7:
				f.uses = append(f.uses, pcUse{path: []pcSeg{kw("crate"), mod(), item("x")}, pub: true})
				f.uses = append(f.uses, pcUse{path: []pcSeg{kw("crate"), mod(), item("x"), item(nm())}, pub: rng.Intn(2) == 0})
			case 8:
				f.uses = append(f.uses, pcUse{path: []pcSeg{kw("super")}, glob: true})
			case 9:
				if n := nm(); !slices.Contains(f.fns, n) {
					f.fns = append(f.fns, n)
				}
			case 10:
				if n := nm(); !slices.Contains(f.xFns, n) {
					f.hasX = true
					f.xFns = append(f.xFns, n)
				}
			default:
				h := fmt.Sprintf("h%d", len(f.callers))
				f.callers = append(f.callers, h)
				f.calls[h] = [][]pcSeg{
					{item(nm())},
					{mod(), item(nm())},
					{mod(), mod(), item(nm())},
					{kw("super"), mod(), item(nm())},
					{mod(), item("x"), item(nm())},
					{kw("crate"), mod(), item("x"), item(nm())},
				}
			}
		}
		c.files = append(c.files, f)
	}
	lib := &pcFile{callers: []string{"go"}, calls: map[string][][]pcSeg{}}
	for range rng.Intn(3) {
		lib.uses = append(lib.uses, pcUse{path: []pcSeg{kw("crate"), mod()}, glob: true, pub: true})
	}
	lib.calls["go"] = [][]pcSeg{
		{item("f")}, {item("g")}, {mod(), item(nm())}, {mod(), mod(), item(nm())}, {mod(), item("x"), item(nm())},
	}
	c.files = append(c.files, lib)
	return c
}

// render is the crate as source, module i named name(i).
func (c *pcCrate) render(name func(int) string) map[string]string {
	segs := func(p []pcSeg) string {
		var out []string
		for _, s := range p {
			if s.mod >= 0 {
				out = append(out, name(s.mod))
			} else {
				out = append(out, s.name)
			}
		}
		return strings.Join(out, "::")
	}
	tree := map[string]string{}
	for i, f := range c.files {
		var b strings.Builder
		if i == c.k {
			for j := range c.k {
				fmt.Fprintf(&b, "pub mod %s;\n", name(j))
			}
		}
		for _, u := range f.uses {
			if u.pub {
				b.WriteString("pub ")
			}
			fmt.Fprintf(&b, "use %s", segs(u.path))
			switch {
			case u.glob:
				b.WriteString("::*")
			case u.alias != "":
				fmt.Fprintf(&b, " as %s", u.alias)
			}
			b.WriteString(";\n")
		}
		for _, fn := range f.fns {
			fmt.Fprintf(&b, "pub fn %s() {}\n", fn)
		}
		if f.hasX {
			b.WriteString("pub mod x {\n")
			for _, fn := range f.xFns {
				fmt.Fprintf(&b, "    pub fn %s() {}\n", fn)
			}
			b.WriteString("}\n")
		}
		for _, h := range f.callers {
			fmt.Fprintf(&b, "pub fn %s() {", h)
			for _, p := range f.calls[h] {
				fmt.Fprintf(&b, " %s();", segs(p))
			}
			b.WriteString(" }\n")
		}
		tree[c.path(i, name)] = b.String()
	}
	return tree
}

func (c *pcCrate) path(i int, name func(int) string) string {
	if i == c.k {
		return "src/lib.rs"
	}
	return "src/" + name(i) + ".rs"
}

// refMod is a module of the reference: file i, or the inline x in it.
type refMod struct {
	file int
	x    bool
}

// refSet is what a name stands for: its candidates, "mod:" or "fn:" plus
// the node, and whether a path with two candidates for one segment led there.
type refSet struct {
	defs map[string]bool
	amb  bool
}

// add puts the candidates of o in ns into a fresh copy of s's, so a set the
// last pass left stays as it was.
func (s *refSet) add(o refSet, ns string) {
	defs := map[string]bool{}
	maps.Copy(defs, s.defs)
	for d := range o.defs {
		if ns == "any" || strings.HasPrefix(d, map[string]string{"type": "mod:", "value": "fn:"}[ns]) {
			defs[d] = true
		}
	}
	s.defs, s.amb = defs, s.amb || o.amb
}

type refKey struct {
	file int
	pub  bool
	ns   string
	name string
}

// refEval is the least fixpoint of the resolution rules over candidate
// sets, written apart from the resolver: every explicit use's candidates and
// every name the globs of a file offer, recomputed from the last pass until
// a pass changes nothing.
type refEval struct {
	c     *pcCrate
	useT  map[[2]int]refSet
	globT map[refKey]refSet
}

func (r *refEval) id(i int, rest string) string {
	return "fn:" + r.c.path(i, canonical) + "#" + rest
}

func canonical(i int) string { return fmt.Sprintf("m%d", i) }

func (r *refEval) holds(a, from refMod) bool {
	return a == from || a.file == r.c.k || !a.x && a.file == from.file
}

// selfReading says whether the use u of file names file's own name back:
// crate::<file>::n, binding n.
func (r *refEval) selfReading(u [2]int) bool {
	p := r.c.files[u[0]].uses[u[1]]
	return !p.glob && p.alias == "" && len(p.path) == 3 && p.path[0].name == "crate" && p.path[1].mod == u[0]
}

// past says whether the use excl, while its path is resolved, reads past
// the use i of a binder list (excl reads one name, so i binds it too):
// itself, and all self-reading uses when it is one.
func (r *refEval) past(excl, i [2]int) bool {
	return excl == i || excl[0] == i[0] && r.selfReading(excl) && r.selfReading(i)
}

func (r *refEval) binds(u pcUse) string {
	if u.alias != "" {
		return u.alias
	}
	last := u.path[len(u.path)-1]
	if last.mod >= 0 {
		return canonical(last.mod)
	}
	return last.name
}

// read is what name means in ns in m, from inside it or not: m's own items,
// else the explicit uses of its file that bind it (but excl), else its globs.
func (r *refEval) read(m refMod, inside bool, ns, name string, excl [2]int) refSet {
	f := r.c.files[m.file]
	var own refSet
	if m.x {
		if ns == "value" && slices.Contains(f.xFns, name) {
			own.add(refSet{defs: map[string]bool{r.id(m.file, "x::"+name): true}}, "any")
		}
		return own
	}
	if ns == "value" && (slices.Contains(f.fns, name) || slices.Contains(f.callers, name)) {
		own.add(refSet{defs: map[string]bool{r.id(m.file, name): true}}, "any")
	}
	if ns == "type" && name == "x" && f.hasX {
		own.add(refSet{defs: map[string]bool{fmt.Sprintf("mod:%d:x", m.file): true}}, "any")
	}
	if ns == "type" && m.file == r.c.k {
		for j := range r.c.k {
			if canonical(j) == name {
				own.add(refSet{defs: map[string]bool{fmt.Sprintf("mod:%d", j): true}}, "any")
			}
		}
	}
	if len(own.defs) > 0 {
		return own
	}
	var out refSet
	bound := false
	for i, u := range f.uses {
		if !u.glob && (inside || u.pub) && r.binds(u) == name && !r.past(excl, [2]int{m.file, i}) {
			bound = true
			out.add(r.useT[[2]int{m.file, i}], ns)
		}
	}
	if bound {
		return out
	}
	return r.globT[refKey{m.file, !inside, ns, name}]
}

func refModOf(d string) refMod {
	var m refMod
	rest := strings.TrimPrefix(d, "mod:")
	file, x, _ := strings.Cut(rest, ":")
	fmt.Sscan(file, &m.file)
	m.x = x == "x"
	return m
}

// resolve is what the path p written in file from names in ns.
func (r *refEval) resolve(from int, p []pcSeg, ns string, excl [2]int) refSet {
	here := refMod{file: from}
	name := func(s pcSeg) string {
		if s.mod >= 0 {
			return canonical(s.mod)
		}
		return s.name
	}
	readIn := func(m refMod, ns, n string) refSet {
		if ns != "any" {
			return r.read(m, r.holds(m, here), ns, n, excl)
		}
		var out refSet
		for _, one := range []string{"type", "value"} {
			out.add(r.read(m, r.holds(m, here), one, n, excl), "any")
		}
		return out
	}
	var cur refSet
	switch p[0].name {
	case "crate":
		cur.add(refSet{defs: map[string]bool{fmt.Sprintf("mod:%d", r.c.k): true}}, "any")
	case "super":
		cur.add(refSet{defs: map[string]bool{fmt.Sprintf("mod:%d", r.c.k): true}}, "any")
	default:
		if len(p) == 1 {
			return readIn(here, ns, name(p[0]))
		}
		cur = readIn(here, "type", name(p[0]))
	}
	for i := 1; i < len(p); i++ {
		want := "type"
		if i == len(p)-1 {
			want = ns
		}
		next := refSet{amb: cur.amb || len(cur.defs) > 1}
		for d := range cur.defs {
			next.add(readIn(refModOf(d), want, name(p[i])), "any")
		}
		cur = next
	}
	return cur
}

func (r *refEval) pass() bool {
	useT, globT := map[[2]int]refSet{}, map[refKey]refSet{}
	maps.Copy(useT, r.useT)
	maps.Copy(globT, r.globT)
	// Every name the crate has: what a glob may offer.
	names := []string{"f", "g", "x"}
	for j, f := range r.c.files {
		names = append(names, canonical(j))
		names = append(names, f.callers...)
	}
	for fi, f := range r.c.files {
		for ui, u := range f.uses {
			if u.glob {
				continue
			}
			s := useT[[2]int{fi, ui}]
			s.add(r.resolve(fi, u.path, "any", [2]int{fi, ui}), "any")
			useT[[2]int{fi, ui}] = s
		}
		for _, pub := range []bool{false, true} {
			for _, u := range f.uses {
				if !u.glob || pub && !u.pub {
					continue
				}
				op := r.resolve(fi, u.path, "type", [2]int{-1, -1})
				amb := op.amb || len(op.defs) > 1
				for d := range op.defs {
					o := refModOf(d)
					for _, n := range names {
						for _, ns := range []string{"type", "value"} {
							got := r.read(o, r.holds(o, refMod{file: fi}), ns, n, [2]int{-1, -1})
							if len(got.defs) == 0 && !got.amb {
								continue
							}
							got.amb = got.amb || amb
							key := refKey{fi, pub, ns, n}
							s := globT[key]
							s.add(got, "any")
							globT[key] = s
						}
					}
				}
			}
		}
	}
	changed := false
	for k, v := range useT {
		changed = changed || len(v.defs) != len(r.useT[k].defs) || v.amb != r.useT[k].amb
	}
	for k, v := range globT {
		changed = changed || len(v.defs) != len(r.globT[k].defs) || v.amb != r.globT[k].amb
	}
	r.useT, r.globT = useT, globT
	return changed
}

// calls are the call edges of the crate as the reference resolves them.
func (c *pcCrate) calls() []string {
	r := &refEval{c: c, useT: map[[2]int]refSet{}, globT: map[refKey]refSet{}}
	for r.pass() {
	}
	var out []string
	for fi, f := range c.files {
		for _, h := range f.callers {
			for _, p := range f.calls[h] {
				s := r.resolve(fi, p, "value", [2]int{-1, -1})
				if s.amb || len(s.defs) != 1 {
					continue
				}
				for d := range s.defs {
					out = append(out, fmt.Sprintf("%s#%s -calls-> %s extracted", c.path(fi, canonical), h, strings.TrimPrefix(d, "fn:")))
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

// rsLines is a resolution as sorted lines, for comparing two.
func rsLines(out langOut) []string {
	var lines []string
	for _, e := range out.edges {
		lines = append(lines, fmt.Sprintf("%s -%s-> %s %s", e.Source, e.Relation, e.Target, e.Confidence))
	}
	for _, u := range out.unresolved {
		lines = append(lines, fmt.Sprintf("unresolved %s %s", u.Source, u.Name))
	}
	sort.Strings(lines)
	return lines
}

func resolveTree(t *testing.T, tree map[string]string, reverse bool) []string {
	t.Helper()
	mods, err := ModulesFrom(map[string]string{"Cargo.toml": "[package]\nname = \"app\"\n"})
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	if reverse {
		slices.Reverse(paths)
	}
	var files []extract.Result
	for _, p := range paths {
		r, err := rust.File(p, tree[p])
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	return rsLines(resolveRust(files, mods))
}

// The resolver gives the least fixpoint the reference computes, whatever
// order the files come in and whatever the modules are called.
func TestRustTablesMatchTheReference(t *testing.T) {
	for seed := range int64(2000) {
		rng := rand.New(rand.NewSource(seed))
		c := randomCrate(rng)
		tree := c.render(canonical)
		got := resolveTree(t, tree, false)
		if back := resolveTree(t, tree, true); !slices.Equal(got, back) {
			t.Fatalf("seed %d: the file order changes the answer:\n%v\n%v", seed, got, back)
		}
		// Renamed modules sort in another order; their files map back.
		perm := rng.Perm(c.k)
		renamed := func(i int) string { return fmt.Sprintf("q%c", 'a'+perm[i]) }
		var pairs []string
		for i := range c.k {
			pairs = append(pairs, c.path(i, renamed), c.path(i, canonical))
		}
		back := strings.NewReplacer(pairs...)
		var again []string
		for _, l := range resolveTree(t, c.render(renamed), false) {
			again = append(again, back.Replace(l))
		}
		sort.Strings(again)
		if !slices.Equal(got, again) {
			t.Fatalf("seed %d: the module names change the answer:\n%s\n%v\n%v", seed, dumpTree(tree), got, again)
		}
		var calls []string
		for _, l := range got {
			if strings.Contains(l, " -calls-> ") {
				calls = append(calls, l)
			}
		}
		if want := c.calls(); !slices.Equal(calls, want) {
			t.Fatalf("seed %d:\n%s\ngot  %v\nwant %v", seed, dumpTree(tree), calls, want)
		}
	}
}

func dumpTree(tree map[string]string) string {
	var b strings.Builder
	for _, p := range slices.Sorted(maps.Keys(tree)) {
		fmt.Fprintf(&b, "--- %s\n%s", p, tree[p])
	}
	return b.String()
}
