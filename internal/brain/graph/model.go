// Package graph renders what the indexer knows about one area, reads it back
// and answers neighbour questions over it.
//
// Rendering and reading sit in one package because they share this model: a
// field renamed on the write side and forgotten on the read side would be a
// graph.json nobody can decode, and here the compiler catches it.
//
// A read model of its own rather than decoding into a map: a map would let a
// field the indexer renames reach a caller unnoticed.
package graph

// Node is a document in the graph.
type Node struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

// Edge represents a directed link from one document to another.
//
// Both ends are document-relative paths inside the same area; a link climbing
// past the area root never becomes an Edge, and its target is kept nowhere
// (`src/brain/graph.py:130-141`) -- which is why links between areas cannot be
// recovered by joining these files: what left an area was never written down.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// LinksInfo contains statistics on link occurrences and drop reasons.
//
// Dropped is a map and not a struct because the reasons are the indexer's to
// name (`src/brain/graph.py:87-95`), and a struct here would silently swallow
// a reason it gained.
type LinksInfo struct {
	Total    int            `json:"total"`
	Resolved int            `json:"resolved"`
	Dropped  map[string]int `json:"dropped"`
}

// Graph is the full payload of graph.json.
type Graph struct {
	Scope string    `json:"scope"`
	Nodes []Node    `json:"nodes"`
	Edges []Edge    `json:"edges"`
	Links LinksInfo `json:"links"`
}

// Document is an input document for graph rendering.
type Document struct {
	Relative string
	Title    string
	Tags     []string
	Links    []string
}
