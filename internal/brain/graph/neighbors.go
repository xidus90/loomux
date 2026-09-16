package graph

import (
	"fmt"
	"sort"
	"strings"
)

// Neighbors returns the sorted incoming and outgoing edges for the given relative document path.
func Neighbors(g *Graph, relative string) ([]string, []string) {
	if g == nil {
		return nil, nil
	}

	var incoming []string
	var outgoing []string

	for _, edge := range g.Edges {
		if edge.To == relative {
			incoming = append(incoming, edge.From)
		}
		if edge.From == relative {
			outgoing = append(outgoing, edge.To)
		}
	}

	sort.Strings(incoming)
	sort.Strings(outgoing)
	return incoming, outgoing
}

// RenderNeighbors formats incoming and outgoing document slices into the canonical two-line string.
func RenderNeighbors(incoming, outgoing []string) string {
	inStr := "-"
	if len(incoming) > 0 {
		inStr = strings.Join(incoming, ", ")
	}
	outStr := "-"
	if len(outgoing) > 0 {
		outStr = strings.Join(outgoing, ", ")
	}
	return fmt.Sprintf("incoming: %s\noutgoing: %s\n", inStr, outStr)
}
