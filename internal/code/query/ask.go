package query

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/store"
)

// ErrNoGraph is a query against a repository nobody has built.
var ErrNoGraph = errors.New("no graph. Run `loomux graph build` first.")

// AskOptions are one question's knobs. Source and Full decide what the answer
// carries, Keep what it may consider at all.
type AskOptions struct {
	Limit     int
	In        string
	Source    bool
	Full      bool
	NoRefresh bool
	Keep      func(path string) bool
}

// Ask answers one question from the graph of root.
//
// A missing graph is refused before anything else, and nothing is built: the
// reference's reason holds (src/graph/refresh.ts) -- building a whole
// repository under a query is a surprise, and the one case where nobody has
// switched the graph on yet. Over MCP it is worse than a surprise: any model
// that can name a visible area could start a full build on this machine. A
// graph that exists is refreshed as before.
//
// The notes are what the command line writes to stderr and a tool call sends
// as progress: a rebuild, a missing sidecar.
func Ask(root, question string, opts AskOptions) (ask.Answer, []string, error) {
	if _, err := os.Stat(store.WiringPath(root)); errors.Is(err, os.ErrNotExist) {
		return ask.Answer{}, nil, ErrNoGraph
	}
	var notes []string
	if !opts.NoRefresh {
		say := func(s string) { notes = append(notes, s) }
		ask.EnsureFresh(root, all.Version(),
			func() error { _, _, err := Build(root, say); return err }, say)
	}
	g, err := store.Read(root)
	if err != nil {
		return ask.Answer{}, notes, err
	}
	ix, err := lexicon.Read(root)
	if err != nil {
		// The sidecar is a cache: without it, tokenize live off the graph. The
		// body text is gone from the written graph, so a body-only word will
		// not be found -- say so rather than answer worse in silence.
		notes = append(notes, fmt.Sprintf("no ask index, ranking on names and paths only: %v", err))
		ix = lexicon.Build(g)
	}
	a := ask.Run(g, ix, question, ask.Options{Limit: opts.Limit, In: opts.In, Keep: opts.Keep})
	if opts.Source {
		ask.Inline(root, &a, opts.Full)
	}
	return a, notes, nil
}

// AskReport is the human form: one block per hit, location first.
func AskReport(a ask.Answer) string {
	if a.Note != "" {
		return a.Note + "\n"
	}
	var b strings.Builder
	for i, h := range a.Hits {
		fmt.Fprintf(&b, "%d. %s  %s:%s  (%.3f lex %.3f graph %.3f)\n",
			i+1, h.ID, h.Path, h.Span, h.Score, h.Lexical, h.Graph)
		if h.Signature != "" {
			fmt.Fprintf(&b, "   %s\n", h.Signature)
		}
		if h.Code != "" {
			for _, line := range strings.Split(h.Code, "\n") {
				fmt.Fprintf(&b, "   | %s\n", line)
			}
		}
	}
	return b.String()
}
