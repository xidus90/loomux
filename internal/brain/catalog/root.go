package catalog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/config"
)

// RenderRootCatalog renders the root catalog in markdown listing all visible areas,
// sorted by area scope.
//
// Format:
// # brain
//
// * [scope](brain://scope/)
// ...
func RenderRootCatalog(areas []config.Area) string {
	sortedAreas := make([]config.Area, len(areas))
	copy(sortedAreas, areas)
	sort.Slice(sortedAreas, func(i, j int) bool {
		return sortedAreas[i].Scope < sortedAreas[j].Scope
	})

	var sb strings.Builder
	sb.WriteString("# brain\n\n")
	for _, a := range sortedAreas {
		fmt.Fprintf(&sb, "* [%s](brain://%s/)\n", a.Scope, a.Scope)
	}
	return sb.String()
}
