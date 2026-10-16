package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/pathkey"
)

// Map assigns files to the layers of the layer map.
type Map struct {
	layers map[string]config.Layer
	of     map[string]string // rel -> layer name
}

// Of returns the layer a file belongs to; false when it lies outside every layer.
func (m *Map) Of(rel string) (config.Layer, bool) {
	name, ok := m.of[rel]
	return m.layers[name], ok
}

// specificity ranks a glob; see moreSpecific.
type specificity struct {
	anywhere bool // matches at any depth: a leading **/ or no slash at all
	literal  bool // no placeholder in it
	fixed    int  // segments before the first one with a placeholder
}

func specificityOf(glob string) specificity {
	rest, anywhere := pathkey.CutAnywhere(glob)
	s := specificity{anywhere: anywhere || !strings.Contains(glob, "/"), literal: !strings.ContainsAny(rest, "*?[")}
	for _, seg := range strings.Split(rest, "/") {
		if strings.ContainsAny(seg, "*?[") {
			break
		}
		s.fixed++
	}
	return s
}

// moreSpecific orders two globs matching the same file, a before b:
//  1. a glob anchored at the root beats one that matches at any depth (a
//     leading **/ or a pattern without a slash, which pathkey.Glob matches
//     against the base name);
//  2. a glob without a placeholder beats one with;
//  3. more fixed segments before the first placeholder beat fewer.
//
// Equal on all three is a tie.
func moreSpecific(a, b specificity) bool {
	if a.anywhere != b.anywhere {
		return !a.anywhere
	}
	if a.literal != b.literal {
		return a.literal
	}
	return a.fixed > b.fixed
}

// Assign gives every file of rels the layer with the most specific matching
// glob. The map errors come back as text, sorted: a glob that matches no
// file or cannot be read, and a file two layers claim equally.
func Assign(r config.Rules, rels []string) (*Map, []string) {
	type globKey struct{ layer, glob string }
	m := &Map{layers: map[string]config.Layer{}, of: map[string]string{}}
	var errs []string
	used := map[globKey]bool{} // globs that matched a file
	for _, l := range r.Layers {
		m.layers[l.Name] = l
	}
	for _, rel := range rels {
		var best specificity
		var owners []string // layers holding the best glob so far
		for _, l := range r.Layers {
			for _, g := range l.Paths {
				// A glob that cannot be read is reported once, below.
				if ok, err := pathkey.Glob(g, rel); err != nil || !ok {
					continue
				}
				used[globKey{l.Name, g}] = true
				s := specificityOf(g)
				switch {
				case owners == nil || moreSpecific(s, best):
					best, owners = s, []string{l.Name}
				case !moreSpecific(best, s) && !slices.Contains(owners, l.Name):
					owners = append(owners, l.Name)
				}
			}
		}
		switch len(owners) {
		case 0:
		case 1:
			m.of[rel] = owners[0]
		default:
			slices.Sort(owners)
			errs = append(errs, fmt.Sprintf("%s matches layers %s and %s equally", rel,
				strings.Join(owners[:len(owners)-1], ", "), owners[len(owners)-1]))
		}
	}
	for _, l := range r.Layers {
		for _, g := range l.Paths {
			if _, err := pathkey.Glob(g, ""); err != nil {
				errs = append(errs, fmt.Sprintf("layer %s: glob %q: %v", l.Name, g, err))
			} else if !used[globKey{l.Name, g}] {
				errs = append(errs, fmt.Sprintf("layer %s: %s matches no file", l.Name, g))
			}
		}
	}
	slices.Sort(errs)
	return m, errs
}
