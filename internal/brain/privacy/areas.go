package privacy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// VisibleArea is one registered area a channel may see, together with the
// declaration that decided it.
type VisibleArea struct {
	Area     config.Area
	Manifest *config.Manifest

	// Hidden are the trees of every area this channel may not see: the
	// source tree and the wiki of each. Areas nest -- a hub's wiki can hold
	// the wiki of a `local_only` project -- and a path inside such a tree
	// stays hidden whichever visible area it is reached through; Conceals
	// asks it. Empty on the local channel, which hides no area.
	Hidden []string
}

// VisibleAreas answers the areas the caller may see, resolved before anything
// is asked. It is `_visible_areas` (src/brain/core.py:248-263), whose reason
// holds here too: "filtering afterwards would mean the invisible area was
// queried -- and a query is already a disclosure of the question".
//
// The registry comes from registryDir, which is also the state directory the
// artefacts of read-only areas are read from. The manifest of each area comes
// from config.ResolvedAreaDir: a read-only area keeps it there, with
// fallbackDir -- ultra-brain's -- as the fallback. Every registered area's
// declaration is read and its inbox checked before scope and visibility are
// asked, so the first registry or declaration error ends the call, whichever
// area it belongs to -- a hidden area included. The one exception is a
// workspace entry without a declaration, which is left out silently.
//
// scope "all" answers every visible area in registry order. Any other scope
// answers the visible areas of that name, or the UnknownScope error when there
// are none.
func VisibleAreas(registryDir, fallbackDir, scope string, ch Channel) ([]VisibleArea, error) {
	areas, err := config.ReadRegistry(registryDir)
	if err != nil {
		return nil, err
	}
	var visible []VisibleArea
	var hidden []string
	for _, area := range areas {
		manifest, seen, err := VisibleManifest(config.ResolvedAreaDir(area, registryDir, fallbackDir), ch)
		if area.Workspace && config.IsUndeclared(err) {
			// A workspace that declares no [area] is no brain area: there is
			// nothing to see in it and nothing to hide.
			continue
		}
		if err != nil {
			return nil, err
		}
		if _, err := manifest.InboxLayout(); err != nil {
			return nil, err
		}
		if seen {
			visible = append(visible, VisibleArea{Area: area, Manifest: manifest})
			continue
		}
		hidden = append(hidden, area.Path)
		if area.WikiPath != "" {
			hidden = append(hidden, area.WikiPath)
		}
	}
	// One shared slice, read and never written: every visible area is asked
	// about the same hidden trees.
	for i := range visible {
		visible[i].Hidden = hidden
	}
	if scope == "all" {
		return visible, nil
	}
	var named []VisibleArea
	for _, entry := range visible {
		if entry.Area.Scope == scope {
			named = append(named, entry)
		}
	}
	if len(named) == 0 {
		return nil, UnknownScope(scope, visible)
	}
	return named, nil
}

// Single is `_single` (src/brain/core.py:535-539): the first visible area of
// that scope, or the UnknownScope error. The registry refuses two entries of
// one scope, so there is at most one.
func Single(areas []VisibleArea, scope string) (VisibleArea, error) {
	for _, entry := range areas {
		if entry.Area.Scope == scope {
			return entry, nil
		}
	}
	return VisibleArea{}, UnknownScope(scope, areas)
}

// UnknownScope is `_unknown_scope` (src/brain/core.py:266-275), "the one
// message every tool gives for a scope it cannot serve". It names the visible
// scopes sorted, and only those: one message for an unknown and a hidden scope
// is the point, since telling a cloud caller that an area exists would
// disclose what `local_only` hides. Go sorts strings by bytes, Python by code
// points; for UTF-8 the two orders agree.
func UnknownScope(scope string, areas []VisibleArea) error {
	known := make([]string, 0, len(areas))
	for _, entry := range areas {
		known = append(known, entry.Area.Scope)
	}
	sort.Strings(known)
	return fmt.Errorf("unknown scope %s; known scopes are: %s", pytext.Repr(scope), strings.Join(known, ", "))
}
