package verify

import (
	"path/filepath"
	"slices"
	"strings"
)

// dirCandidates are the engines a directory holds, sorted by name so that a
// second run on the same machine asks the same binary first.
//
// A name counts as an engine when it begins with "godot", whatever its case,
// which already excludes a project.godot someone keeps beside the downloads,
// and the README with it. console turns a Windows window build into the
// console build beside it; since Godot's download ships both, two names can
// map to one path, and the duplicate is dropped rather than asked twice. The
// reader's error is handed back: a directory that is not there is a different
// thing to report than one that holds nothing, and only the caller knows
// which it wants to say.
func dirCandidates(dir string, readDir func(string) ([]string, error), console func(string) string) ([]string, error) {
	if dir == "" {
		return nil, nil
	}
	names, err := readDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, name := range slices.Sorted(slices.Values(names)) {
		if !strings.HasPrefix(strings.ToLower(name), "godot") {
			continue
		}
		p := filepath.ToSlash(filepath.Join(dir, name))
		if console != nil {
			p = console(p)
		}
		out = append(out, p)
	}
	return slices.Compact(out), nil
}
