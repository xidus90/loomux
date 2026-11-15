// Package graphview shapes the code graph and the wiki into the one graph the
// web interface draws: folded into directories, small enough to read.
package graphview

// DefaultLimit is how many nodes a view shows before the reader opens more.
const DefaultLimit = 150

// MaxNodes is the most nodes any view shows, however much the reader opens: the
// whole code graph never goes to the browser.
const MaxNodes = 1000

// Node is one drawn element: a directory, a file, a symbol or a document.
type Node struct {
	ID      string  `json:"id"` // "dir:internal/web", "file:internal/web/a.go", a symbol's NodeID, "doc:topics/x.md"
	Label   string  `json:"label"`
	Kind    string  `json:"kind"` // "dir" | "file" | "symbol" | "doc"
	Path    string  `json:"path"`
	Group   string  `json:"group"` // colour key: top directory, or top folder of a document
	Size    float64 `json:"size"`
	Members int     `json:"members"`
	// Wiki marks a document inside the wiki folder.
	Wiki bool `json:"wiki,omitempty"`
	// Missing lists a document's refs that are not in the code graph.
	Missing []string `json:"missing,omitempty"`
}

// Edge joins two nodes; Counts says what the folded edges were.
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"` // "code" | "link" | "ref"
	// Counts maps "calls/extracted" to a number; "link" and "ref" use their
	// kind as key.
	Counts map[string]int `json:"counts"`
}

// View is what the endpoint answers.
type View struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
	// Open lists the directories and files shown expanded, after auto-open and
	// cleaning.
	Open []string `json:"open"`
}
