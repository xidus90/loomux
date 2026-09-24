package hooks

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
)

// monitorRead is a seam: a test counts the reads through it to show that
// only a green Go edit reads the graph, never another stack's file or an
// edit a lane refused.
var monitorRead = store.Read

// maxAsideCallers is how many callers the aside names before it counts.
const maxAsideCallers = 10

// typeNote is the aside for a changed type: the Go graph has no edges to
// types, so silence would read as "nothing depends on this".
const typeNote = "[graph] Modified struct/interface/type: type coupling not wired in graph v1 (check references via grep)"

// blastAside tells the model who calls what an edit just changed: the
// symbols of rel whose body hash differs from the graph's, or that the graph
// has and the file no longer does, and their direct callers in other files.
// It never fails and never blocks; whatever it cannot know is silence.
func blastAside(root, rel string, read func(string) ([]byte, error)) string {
	g, err := monitorRead(root)
	if err != nil || g.Meta.Extractor != golang.Version {
		return ""
	}
	src, err := read(rel)
	if err != nil {
		return ""
	}
	now, err := golang.File(rel, string(src))
	if err != nil {
		return ""
	}
	// The file node is no seed: Reach does not expand a file node to its
	// symbols, and its incoming edges are imports, which land only on one
	// representative file per package, so the aside would depend on which
	// file of a package was edited.
	hashes := map[model.NodeID]string{}
	for _, n := range now.Nodes {
		if n.Kind != model.KindFile {
			hashes[n.ID] = n.BodyHash
		}
	}
	var removed, changed []*model.Node
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if n.Path != rel || n.Kind == model.KindFile {
			continue
		}
		h, ok := hashes[n.ID]
		switch {
		case !ok:
			removed = append(removed, n)
		case h != n.BodyHash:
			changed = append(changed, n)
		}
	}
	seeds := append(removed, changed...)
	if len(seeds) == 0 {
		return ""
	}
	ids := make([]model.NodeID, len(seeds))
	typed := false
	for i, s := range seeds {
		ids[i] = s.ID
		typed = typed || s.Kind == "struct" || s.Kind == "interface" || s.Kind == "type"
	}
	var callers []string
	for _, h := range blast.New(g).Reach(ids, blast.In, 1) {
		if h.Node != nil && h.Node.Path != rel {
			callers = append(callers, fmt.Sprintf("  %s (%s)", h.Node.Name, h.Node.Path))
		}
	}
	var b strings.Builder
	if len(callers) > 0 {
		fmt.Fprintf(&b, "[graph] %s: %s; callers in other files:\n", rel, seedList(removed, changed))
		for i, c := range callers {
			if i == maxAsideCallers {
				fmt.Fprintf(&b, "  … and %d more\n", len(callers)-maxAsideCallers)
				break
			}
			b.WriteString(c + "\n")
		}
	}
	if typed {
		b.WriteString(typeNote + "\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// seedList names the removed symbols first: their callers break for sure.
func seedList(removed, changed []*model.Node) string {
	var parts []string
	if len(removed) > 0 {
		parts = append(parts, "removed "+names(removed))
	}
	if len(changed) > 0 {
		parts = append(parts, "changed "+names(changed))
	}
	return strings.Join(parts, ", ")
}

func names(nodes []*model.Node) string {
	out := make([]string, len(nodes))
	for i, n := range nodes {
		out[i] = n.Name
	}
	return strings.Join(out, ", ")
}

// relInRoot is the edited file as a slash path inside root; a file outside
// it belongs to no lane and no graph of this project.
func relInRoot(root, raw string) (string, bool) {
	file := raw
	if !filepath.IsAbs(file) {
		file = filepath.Join(root, file)
	}
	rel, err := filepath.Rel(root, file)
	if err != nil || !filepath.IsLocal(rel) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}
