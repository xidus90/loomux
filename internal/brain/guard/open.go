package guard

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/BurntSushi/toml"
)

// open.toml in the state directory names single files outside the registered
// trees that every writer may write, on the terms memory is open on. It exists
// for files a user's own conventions put there -- a global CLAUDE.md that has
// agents keep AGENT_LEARNINGS.md -- which the barrier cannot know in code.
//
// A file of its own and not a table in registry.toml: `loomux area add`
// rewrites the registry from its [[area]] entries, and a table beside them
// would vanish on the next add. It lies in the state directory, outside every
// tree an agent may write, so no agent can grant itself an entry. Read whole
// or not at all: an entry the barrier cannot use ignores the file, the refusal
// says why, and nothing that was shut before opens.

// openName is the file in the state directory.
const openName = "open.toml"

// statEntry is the stat of a listed entry; a test swaps it for an error no
// unprivileged machine produces on demand.
var statEntry = os.Stat

// openFiles resolves every file open.toml lists, or nil where there is no
// open.toml. An error means the file is there and ignored; it says why.
//
// Only single files open. An entry must be absolute, may not name a
// directory (a file not yet written opens; one whose kind cannot be read
// ignores the file), and may not lie in the state directory, where the registry and
// this file keep the barrier's own limits.
func openFiles(stateDir string) ([]string, error) {
	state, err := ResolvePath(stateDir)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(state, openName))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var document struct {
		Files []string `toml:"files"`
	}
	meta, err := toml.Decode(string(data), &document)
	if err != nil {
		return nil, fmt.Errorf("not valid TOML: %w", err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return nil, fmt.Errorf("unknown key %q", undecoded[0].String())
	}
	files := make([]string, 0, len(document.Files))
	for i, entry := range document.Files {
		numbered := fmt.Sprintf("files #%d %q", i+1, entry)
		if !filepath.IsAbs(filepath.FromSlash(entry)) {
			return nil, fmt.Errorf("%s is not an absolute path", numbered)
		}
		resolved, err := ResolvePath(entry)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", numbered, err)
		}
		info, err := statEntry(resolved)
		if err == nil && info.IsDir() {
			return nil, fmt.Errorf("%s names a directory; only single files open", numbered)
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%s: %w", numbered, err)
		}
		if inside(resolved, []string{state}) {
			return nil, fmt.Errorf("%s lies in the state directory, which keeps the barrier's own limits", numbered)
		}
		files = append(files, resolved)
	}
	return files, nil
}

// isOpen reports whether a resolved target is one of the open files,
// compared in components the way the other exemptions compare.
func isOpen(resolved string, files []string) bool {
	target := spelled(resolved)
	return slices.ContainsFunc(files, func(file string) bool {
		return slices.Equal(target, spelled(file))
	})
}
