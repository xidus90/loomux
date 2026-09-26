package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
)

// cacheVersion is the shape of extract.json itself. It moves when cacheEntry,
// extract.Result or model.Node change how they read back. What one language
// extracts is that language's Version, which every entry carries, so a
// changed extractor needs no bump here.
const cacheVersion = 1

// cacheFile is extract.json: the extraction of every source file of the last
// build, by repo-relative path.
type cacheFile struct {
	Version int                   `json:"version"`
	Files   map[string]cacheEntry `json:"files"`
}

// cacheEntry is one file's extraction and what it was taken from: the
// language's Version and the hash of the bytes it read.
//
// Bodies is a field of its own because model.Node's BodyText is json:"-", so
// that wiring.json never carries it. A result that went through JSON alone
// would come back with every body empty, and a reused file would lose each
// word the ask sidecar knows only from its code.
type cacheEntry struct {
	Extractor string                  `json:"extractor"` // the language's Version()
	SHA256    string                  `json:"sha256"`
	Result    extract.Result          `json:"result"`
	Bodies    map[model.NodeID]string `json:"bodies,omitempty"` // Node.BodyText is json:"-"
}

// cachePath is where the extract cache lives, beside the other sidecars.
func cachePath(root string) string { return store.CachePath(root, "extract.json") }

// readCache loads the entries the last build wrote, with each node's body
// back in its BodyText.
//
// A missing file is (nil, nil): a first build has no cache, and that is no
// news. Anything else that keeps the file from being read as this version's
// cache is an error, and the caller parses every file.
func readCache(root string) (map[string]cacheEntry, error) {
	path := cachePath(root)
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f cacheFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if f.Version != cacheVersion {
		return nil, fmt.Errorf("%s: version %d, want %d", path, f.Version, cacheVersion)
	}
	for rel, e := range f.Files {
		for i, n := range e.Result.Nodes {
			e.Result.Nodes[i].BodyText = e.Bodies[n.ID]
		}
		e.Bodies = nil
		f.Files[rel] = e
	}
	return f.Files, nil
}

// writeCache replaces extract.json with entries, each node's non-empty
// BodyText moved into Bodies.
//
// Compact JSON and not indented: the file holds every node of the repository
// and nobody reads it by eye. Temp plus rename with the pid in the temp name,
// as store.Write does, so a concurrent reader never sees half a file.
//
//coverage:exempt json.Marshal's error arm needs a value json cannot encode (a channel, a func, a cyclic pointer, or a NaN/Inf float); a cacheFile holds strings, ints, bools and slices and maps of the same -- no cacheFile this program builds makes it fail
func writeCache(root string, entries map[string]cacheEntry) error {
	files := make(map[string]cacheEntry, len(entries))
	for rel, e := range entries {
		e.Bodies = nil
		for _, n := range e.Result.Nodes {
			if n.BodyText == "" {
				continue
			}
			if e.Bodies == nil {
				e.Bodies = map[model.NodeID]string{}
			}
			e.Bodies[n.ID] = n.BodyText
		}
		files[rel] = e
	}
	body, err := json.Marshal(cacheFile{Version: cacheVersion, Files: files})
	if err != nil {
		return err
	}
	final := cachePath(root)
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return err
	}
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
