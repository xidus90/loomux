package query

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
)

// CallerHit is one hit reached from the start symbol.
type CallerHit struct {
	ID       model.NodeID   `json:"id"`
	Name     string         `json:"name,omitempty"`
	Kind     model.Kind     `json:"kind,omitempty"`
	Path     string         `json:"path,omitempty"`
	Span     model.Span     `json:"span,omitempty"`
	Relation model.Relation `json:"relation"`
	Depth    int            `json:"depth"`
	Inferred bool           `json:"inferred,omitempty"`
	// QuotePath, Line and Quote are the evidence for a direct edge: the line
	// that names the other end, in the caller's file. For a callee that is
	// the start's file, not Path.
	QuotePath string `json:"quote_path,omitempty"`
	Line      int    `json:"line,omitempty"`
	Quote     string `json:"quote,omitempty"`
}

// CallersAnswer packages the start node and hits for CLI and MCP responses.
type CallersAnswer struct {
	Start  *model.Node `json:"start"`
	Hits   []CallerHit `json:"hits"`
	Hidden int         `json:"hidden,omitempty"`
}

// CallersOptions configures the callers/callees walk.
type CallersOptions struct {
	Depth     blast.Depth
	Direction blast.Direction
	In        string
	NoRefresh bool
	Keep      func(path string) bool
}

// Callers resolves symbol and traverses callers or callees along direction.
func Callers(root, symbol string, opts CallersOptions) (CallersAnswer, []string, error) {
	g, notes, err := loadGraph(root, opts.NoRefresh)
	if err != nil {
		return CallersAnswer{}, notes, err
	}

	nodes, err := blast.Resolve(g, symbol, opts.In)
	if err != nil {
		return CallersAnswer{}, notes, err
	}
	if len(nodes) == 0 {
		return CallersAnswer{}, notes, fmt.Errorf("symbol not found: %s", symbol)
	}

	// A rejected match must not shadow a readable one behind it.
	var start *model.Node
	for _, n := range nodes {
		if opts.Keep == nil || opts.Keep(n.Path) {
			start = n
			break
		}
	}
	if start == nil {
		return CallersAnswer{}, notes, fmt.Errorf("symbol not found: %s", symbol)
	}

	x := blast.New(g)
	depth := opts.Depth
	if depth <= 0 && depth != blast.All {
		depth = 1
	}
	hits := x.EdgeWalk(start, opts.Direction, depth)

	sites := edgeSites(g, start.ID, opts.Direction)
	ans := CallersAnswer{Start: start}
	read := func(p string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	}

	for _, h := range hits {
		if h.Node != nil && opts.Keep != nil && !opts.Keep(h.Node.Path) {
			ans.Hidden++
			continue
		}
		ch := CallerHit{
			ID:       h.ID,
			Relation: h.Relation,
			Depth:    h.Depth,
			Inferred: h.Inferred,
		}
		if h.Node != nil {
			ch.Name = h.Node.Name
			ch.Kind = h.Node.Kind
			ch.Path = h.Node.Path
			ch.Span = h.Node.Span
			if h.Depth == 1 {
				at := sites[site{h.ID, h.Relation}]
				ch.QuotePath, ch.Line, ch.Quote = evidence(read, start, h.Node, opts.Direction, at)
			}
		}
		ans.Hits = append(ans.Hits, ch)
	}

	return ans, notes, nil
}

// evidence quotes the line behind the direct edge between start and next:
// the caller's span naming the callee. A hit further out has no edge to start,
// so a word match there would be no evidence and is not looked for.
func evidence(read func(string) ([]byte, error), start, next *model.Node, dir blast.Direction, at int) (string, int, string) {
	caller, callee := next, start
	if dir == blast.Out {
		caller, callee = start, next
	}
	if quote, ok := quoteAt(read, caller.Path, at); ok {
		return caller.Path, at, quote
	}
	line, quote, ok := blast.QuoteLine(read, caller.Path, caller.Span, callee.Name)
	if !ok {
		return "", 0, ""
	}
	return caller.Path, line, quote
}

// site is a direct neighbour of the start by one relation.
type site struct {
	id       model.NodeID
	relation model.Relation
}

// edgeSites is the first line each direct edge of the start was written on.
// The edge knows its line; a word search through the caller's span would
// stop at the first mention, a comment included.
func edgeSites(g *model.Graph, start model.NodeID, dir blast.Direction) map[site]int {
	out := map[site]int{}
	for _, e := range g.Edges {
		other := e.Source
		if dir == blast.Out {
			if e.Source != start {
				continue
			}
			other = e.Target
		} else if e.Target != start {
			continue
		}
		k := site{other, e.Relation}
		if e.Line > 0 && (out[k] == 0 || e.Line < out[k]) {
			out[k] = e.Line
		}
	}
	return out
}

// quoteAt is line n of a file, trimmed; false when there is no such line.
func quoteAt(read func(string) ([]byte, error), path string, n int) (string, bool) {
	if n < 1 {
		return "", false
	}
	data, err := read(path)
	if err != nil {
		return "", false
	}
	lines := strings.Split(string(data), "\n")
	if n > len(lines) {
		return "", false
	}
	return strings.TrimSpace(lines[n-1]), true
}

// mark closes the parenthesis of a hit whose path runs through a guessed edge;
// a sure hit stays as it always read.
func mark(inferred bool) string {
	if inferred {
		return ", inferred"
	}
	return ""
}

// CallersReport formats CallersAnswer for human-readable CLI output.
func CallersReport(a CallersAnswer) string {
	if a.Start == nil {
		return "no start symbol\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s) in %s:%s\n", a.Start.Name, a.Start.Kind, a.Start.Path, a.Start.Span)
	if len(a.Hits) == 0 {
		b.WriteString("  no callers found\n")
	} else {
		for _, h := range a.Hits {
			if h.Name != "" {
				fmt.Fprintf(&b, "  - %s (%s, depth %d, %s%s) in %s:%s\n",
					h.Name, h.Kind, h.Depth, h.Relation, mark(h.Inferred), h.Path, h.Span)
			} else {
				fmt.Fprintf(&b, "  - %s (depth %d, %s%s)\n", h.ID, h.Depth, h.Relation, mark(h.Inferred))
			}
			switch {
			case h.Quote == "":
			case h.QuotePath == h.Path:
				fmt.Fprintf(&b, "    L%d: %s\n", h.Line, h.Quote)
			default:
				fmt.Fprintf(&b, "    %s:L%d: %s\n", h.QuotePath, h.Line, h.Quote)
			}
		}
	}
	if a.Hidden > 0 {
		fmt.Fprintf(&b, "  (%d hidden by privacy policy)\n", a.Hidden)
	}
	return b.String()
}
