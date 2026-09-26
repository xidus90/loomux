package query

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
)

// Drift is what `graph check` found.
type Drift struct {
	OK       bool     `json:"ok"`
	Missing  bool     `json:"missing"`
	Foreign  string   `json:"foreign,omitempty"`
	Outdated bool     `json:"outdated,omitempty"`
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Changed  []string `json:"changed"`
	Hidden   int      `json:"hidden,omitempty"`

	// paths holds the node path of every reported id, as Check read it from
	// the nodes. A path cannot be cut out of an id: a file named "#gen.go"
	// or a directory "dir#1" puts a '#' inside it, and Only decides privacy
	// by it; an id missing here counts as hidden. Unexported, so the JSON
	// form stays what it was.
	paths map[string]string
}

// Check compares a fresh extraction against the graph on disk.
func Check(root string) (Drift, error) {
	written, err := store.Read(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Drift{Missing: true}, nil
		}
		if errors.Is(err, model.ErrSchemaVersion) {
			// A schema-1 graph has no extractor stamp to check next, so this
			// has to be caught before that comparison and given the same
			// pointer to a fix as every other unusable-graph case.
			return Drift{Outdated: true}, nil
		}
		return Drift{}, err
	}
	// The stamp decides before a single node is compared: a graph from another
	// extractor is not stale, it is foreign, and diffing it node by node would
	// report the whole repository. Equality and not Meta.BuiltBy: a graph that
	// lacks one of this binary's languages would diff as all of that
	// language's files added.
	if written.Meta.Extractor != all.Version() {
		return Drift{Foreign: written.Meta.Extractor}, nil
	}

	// Reads the extract cache and never writes it: writing is the build's, and
	// a check that cannot read the cache only parses more, so it stays silent.
	fresh, _, err := Extract(root, ExtractOptions{Reuse: true})
	if err != nil {
		return Drift{}, err
	}

	was := map[model.NodeID]model.Node{}
	for _, n := range written.Nodes {
		was[n.ID] = n
	}
	res := Drift{}
	now := map[model.NodeID]bool{}
	for _, n := range fresh.Nodes {
		now[n.ID] = true
		old, known := was[n.ID]
		switch {
		case !known:
			res.Added = append(res.Added, res.record(n))
		case old.BodyHash != n.BodyHash:
			res.Changed = append(res.Changed, res.record(n))
		}
	}
	for id, n := range was {
		if !now[id] {
			res.Removed = append(res.Removed, res.record(n))
		}
	}
	sort.Strings(res.Added)
	sort.Strings(res.Removed)
	sort.Strings(res.Changed)
	res.OK = len(res.Added)+len(res.Removed)+len(res.Changed) == 0
	return res, nil
}

// record notes the node's path under its id and returns the id.
func (d *Drift) record(n model.Node) string {
	if d.paths == nil {
		d.paths = map[string]string{}
	}
	d.paths[string(n.ID)] = n.Path
	return string(n.ID)
}

// pathFor is the path of a reported id: the node's own, as Check recorded
// it, and false when nothing was recorded for the id.
func (d Drift) pathFor(id string) (string, bool) {
	p, ok := d.paths[id]
	return p, ok
}

// Only drops the ids whose path keep refuses and counts them in Hidden. OK
// stays what it was: drift in a hidden file is still drift, and reporting OK
// would be a lie about the graph.
//
// An id with no recorded path is dropped and counted too, without asking
// keep: its path is unknown, and a path cut from the id could be wrong. A
// privacy filter that cannot tell fails closed.
func (d Drift) Only(keep func(path string) bool) Drift {
	if keep == nil {
		return d
	}
	filter := func(ids []string) []string {
		var out []string
		for _, id := range ids {
			if p, ok := d.pathFor(id); ok && keep(p) {
				out = append(out, id)
			} else {
				d.Hidden++
			}
		}
		return out
	}
	// The closure counts into d, the receiver's copy, which is what is
	// returned; the assignment writes the three slices only, so the count
	// survives.
	d.Added, d.Removed, d.Changed = filter(d.Added), filter(d.Removed), filter(d.Changed)
	return d
}

// CheckReport is the human form.
func CheckReport(d Drift) string {
	switch {
	case d.Missing:
		return "loomux graph check: NO GRAPH\n\nNothing built yet. Run `loomux graph build` first.\n"
	case d.Foreign != "":
		return fmt.Sprintf(
			"loomux graph check: FOREIGN GRAPH\n\nThe graph was written by extractor %q, this binary is %q.\nRun `loomux graph build`.\n",
			d.Foreign, all.Version())
	case d.Outdated:
		return "loomux graph check: OUTDATED GRAPH\n\nThe graph on disk predates this binary's schema.\nRun `loomux graph build`.\n"
	case d.OK:
		return "loomux graph check: OK\n"
	}
	out := "loomux graph check: DRIFT\n\n"
	for _, group := range []struct {
		label string
		ids   []string
	}{
		{"added", d.Added}, {"removed", d.Removed}, {"changed", d.Changed},
	} {
		for _, id := range group.ids {
			out += fmt.Sprintf("  %-8s %s\n", group.label, id)
		}
	}
	if d.Hidden > 0 {
		out += fmt.Sprintf("  (%d more under the area's never globs)\n", d.Hidden)
	}
	return out + "\nRun `loomux graph build`.\n"
}
