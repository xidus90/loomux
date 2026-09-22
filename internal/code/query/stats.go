package query

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
)

// StatsAnswer summarizes graph counts and wiring file size.
type StatsAnswer struct {
	Files       int            `json:"files"`
	Symbols     int            `json:"symbols"`
	Edges       int            `json:"edges"`
	Relations   map[string]int `json:"relations"`
	Languages   []string       `json:"languages"`
	WiringBytes int64          `json:"wiring_bytes"`
}

// GraphStats computes graph metrics: node counts, edges per relation, languages, and wiring size.
func GraphStats(root string) (StatsAnswer, error) {
	st, err := os.Stat(store.WiringPath(root))
	if errors.Is(err, os.ErrNotExist) {
		return StatsAnswer{}, ErrNoGraph
	}
	if err != nil {
		return StatsAnswer{}, err
	}

	g, err := store.Read(root)
	if err != nil {
		return StatsAnswer{}, err
	}

	ans := StatsAnswer{
		WiringBytes: st.Size(),
		Relations:   make(map[string]int),
	}

	langSet := make(map[string]bool)
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if n.Kind == model.KindFile {
			ans.Files++
			ext := strings.ToLower(path.Ext(n.Path))
			if ext != "" {
				langSet[strings.TrimPrefix(ext, ".")] = true
			}
		} else {
			ans.Symbols++
		}
	}

	ans.Edges = len(g.Edges)
	for _, e := range g.Edges {
		ans.Relations[string(e.Relation)]++
	}

	for l := range langSet {
		ans.Languages = append(ans.Languages, l)
	}
	sort.Strings(ans.Languages)

	return ans, nil
}

// StatsReport formats StatsAnswer for human-readable CLI display.
func StatsReport(s StatsAnswer) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Code Graph Stats:\n")
	fmt.Fprintf(&b, "  Files:        %d\n", s.Files)
	fmt.Fprintf(&b, "  Symbols:      %d\n", s.Symbols)
	fmt.Fprintf(&b, "  Edges:        %d\n", s.Edges)
	fmt.Fprintf(&b, "  Wiring size:  %d bytes\n", s.WiringBytes)
	if len(s.Languages) > 0 {
		fmt.Fprintf(&b, "  Languages:    %s\n", strings.Join(s.Languages, ", "))
	}
	if len(s.Relations) > 0 {
		b.WriteString("  Relations:\n")
		var rels []string
		for r := range s.Relations {
			rels = append(rels, r)
		}
		sort.Strings(rels)
		for _, r := range rels {
			fmt.Fprintf(&b, "    %-12s %d\n", r, s.Relations[r])
		}
	}
	return b.String()
}
