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
}

// VisibleAreas answers the areas the caller may see, resolved before anything
// is asked. It is `_visible_areas` (src/brain/core.py:248-263), whose reason
// holds here too: "filtering afterwards would mean the invisible area was
// queried -- and a query is already a disclosure of the question".
//
// The registry comes from registryDir. The manifest of each area comes from
// config.ManifestDir(area, legacyDir): a read-only area keeps it in
// ultra-brain's state directory until stage 3. Python reads the manifest of
// every registered area before it looks at scope, so the first registry or
// manifest error ends the call, whichever area it belongs to.
//
// scope "all" answers every visible area in registry order. Any other scope
// answers the visible areas of that name, or the UnknownScope error when there
// are none.
func VisibleAreas(registryDir, legacyDir, scope string, ch Channel) ([]VisibleArea, error) {
	areas, err := config.ReadRegistry(registryDir)
	if err != nil {
		return nil, err
	}
	var visible []VisibleArea
	for _, area := range areas {
		manifest, seen, err := VisibleManifest(config.ManifestDir(area, legacyDir), ch)
		if err != nil {
			return nil, err
		}
		if seen {
			visible = append(visible, VisibleArea{Area: area, Manifest: manifest})
		}
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
// that scope, or the UnknownScope error.
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
