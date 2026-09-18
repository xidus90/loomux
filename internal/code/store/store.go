// Package store holds the wiring graph on disk.
//
// The graph is machine state and lives under .loomux/state/graph/, which
// .gitignore already excludes. The derived sidecars sit beside it in cache/:
// they are regenerable at any time, the graph is the result.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/write.ts.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/xidus90/loomux/internal/code/model"
)

// Dir is where a repository's graph state lives.
func Dir(root string) string {
	return filepath.Join(root, ".loomux", "state", "graph")
}

// WiringPath is the graph itself.
func WiringPath(root string) string {
	return filepath.Join(Dir(root), "wiring.json")
}

// CachePath is a derived sidecar beside the graph.
func CachePath(root, name string) string {
	return filepath.Join(Dir(root), "cache", name)
}

// Write serializes the graph, atomically.
//
// Temp plus rename, with the pid in the temp name and the temp removed when the
// write fails. Graft's reasoning holds: a fixed temp name lets a concurrent run
// write the same scratch file and hand the loser a truncated graph, and a
// failed rename would leave a full-size orphan nothing ever cleans up.
//
// The caller has sorted the graph; this function adds no order of its own, so
// there is exactly one place where order is decided.
//
//coverage:exempt two arms are unreachable here: MkdirAll's needs a directory that refuses child creation, and testlock.LockDir denies opening the directory itself (a Windows sharing violation), which os.MkdirAll's CreateDirectory call does not go through, so it does not block it -- confirmed empirically, no test manufactured to force it; MarshalIndent's needs a value json cannot encode (a channel, a func, a cyclic pointer, or a NaN/Inf float), and *model.Graph is built entirely of strings, ints, bools and slices of Node/Edge, which are the same -- no *model.Graph this program can construct makes it fail
func Write(root string, g *model.Graph) error {
	if err := os.MkdirAll(Dir(root), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')

	final := WiringPath(root)
	tmp := final + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, final); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// Read loads the graph and validates it.
//
// A missing graph wraps os.ErrNotExist, so a caller can tell "never built" from
// "built and broken" -- the two want different answers, and conflating them is
// how a query starts reporting a clean tree it never looked at.
func Read(root string) (*model.Graph, error) {
	f, err := os.Open(WiringPath(root))
	if err != nil {
		return nil, fmt.Errorf("read graph: %w", err)
	}
	defer f.Close()
	return model.Decode(f)
}
