package query

import (
	"errors"
	"os"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
)

// readGraph is a seam: a test fails it to show which answers do without the
// graph.
var readGraph = store.Read

// loadGraph verifies the graph exists, refreshes it if needed and allowed, and reads it.
func loadGraph(root string, noRefresh bool) (*model.Graph, []string, error) {
	if _, err := os.Stat(store.WiringPath(root)); errors.Is(err, os.ErrNotExist) {
		return nil, nil, ErrNoGraph
	}
	var notes []string
	if !noRefresh {
		say := func(s string) { notes = append(notes, s) }
		ask.EnsureFresh(root, all.Version(),
			func() error { _, _, err := Build(root, say); return err }, say)
	}
	g, err := readGraph(root)
	if err != nil {
		return nil, notes, err
	}
	return g, notes, nil
}
