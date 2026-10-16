package rules

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
	"github.com/xidus90/loomux/internal/verify"
)

// Finding is one violation of a rule; Line 0 means the finding has no line.
type Finding struct {
	Path string
	Line int
	Text string
}

// String is "path:line text", or "path text" without a line.
func (f Finding) String() string {
	if f.Line == 0 {
		return f.Path + " " + f.Text
	}
	return f.Path + ":" + strconv.Itoa(f.Line) + " " + f.Text
}

// Edges reports the edges and the unresolved bases the layer map forbids. The
// graph is built from files alone, with the extractors of `loomux graph`.
func Edges(files []File, m *Map) ([]Finding, error) {
	var results []extract.Result
	gomods := map[string]string{}
	for _, f := range files {
		if path.Base(f.Rel) == "go.mod" {
			gomods[f.Rel] = string(f.Content)
		}
		lang, ok := all.For(f.Rel)
		if !ok {
			continue
		}
		r, err := lang.File(f.Rel, string(f.Content))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Rel, err)
		}
		results = append(results, r)
	}
	// ModulesFrom has no error path today; only its signature leaves room for one.
	mods, _ := resolve.ModulesFrom(gomods)
	g, unresolved := resolve.GraphWith(results, mods, all.Version())
	nodes := map[model.NodeID]model.Node{}
	for _, n := range g.Nodes {
		nodes[n.ID] = n
	}

	// One use per (source file, target file): the first line wins, then the
	// edge that names a symbol rather than a call into it.
	type pair struct{ from, to string }
	type use struct {
		line, rank int
		symbol     string
	}
	uses := map[pair]use{}
	for _, e := range g.Edges {
		if e.Relation == model.RelationContains {
			continue
		}
		src, ok := nodes[e.Source]
		dst, ok2 := nodes[e.Target]
		if !ok || !ok2 || !verify.IsSource(src.Path) {
			continue
		}
		u := use{e.Line, 0, symbolOf(dst)}
		if e.Relation == model.RelationCalls {
			u.rank = 1
		}
		k := pair{src.Path, dst.Path}
		if old, seen := uses[k]; !seen || u.line < old.line || (u.line == old.line && u.rank < old.rank) {
			uses[k] = u
		}
	}

	var out []Finding
	for k, u := range uses {
		from, ok := m.Of(k.from)
		to, inLayer := m.Of(k.to)
		if !ok || !from.Checked || !inLayer || to.Name == from.Name ||
			!from.HasMayUse || slices.Contains(from.MayUse, to.Name) {
			continue
		}
		out = append(out, Finding{k.from, u.line, fmt.Sprintf(
			"%s uses %s (%s) — not in may_use of %s; allow it with: loomux config set rules.layers.%s.may_use … --propose",
			from.Name, to.Name, u.symbol, from.Name, from.Name)})
	}
	for _, un := range unresolved {
		l, ok := m.Of(un.File)
		if !ok || !l.Checked || !verify.IsSource(un.File) {
			continue
		}
		switch {
		case un.Reason == resolve.ReasonAmbiguous:
			out = append(out, Finding{un.File, un.Line, fmt.Sprintf("ambiguous base %s: more than one class declares it", un.Name)})
		case l.HasBases && !slices.Contains(l.Bases, un.Name):
			out = append(out, Finding{un.File, un.Line, fmt.Sprintf("%s extends %s — not in bases of %s", l.Name, un.Name, l.Name)})
		}
	}
	slices.SortFunc(out, func(a, b Finding) int {
		return cmp.Or(strings.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line), strings.Compare(a.Text, b.Text))
	})
	return out, nil
}

// symbolOf names what an edge points at: the path of a file, else the class
// that owns the symbol, else the symbol itself.
func symbolOf(n model.Node) string {
	switch {
	case n.Kind == model.KindFile:
		return n.Path
	case n.Owner != "":
		return n.Owner
	}
	return n.Name
}
