package compat

import (
	"fmt"
	"slices"
	"sync"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/pathkey"
	"github.com/xidus90/loomux/internal/verify"
)

// Watch ties files that are not an entry or a fixture, but decide what an
// entry proves (the code that runs the checks), to the entries a change to
// them must run: all of them, those of some stacks, or those with some addons.
type Watch struct {
	Paths  []string `toml:"paths"`
	Except []string `toml:"except"`
	All    bool     `toml:"all"`
	Stacks []string `toml:"stacks"`
	Addons []string `toml:"addons"`
}

// Watches reads the rules of the embedded list, once.
var Watches = sync.OnceValues(func() ([]Watch, error) { return decodeWatches(listText, "compat.toml") })

// decodeWatches reads the [[watch]] tables strictly and holds each to the
// rules: paths, exactly one selector, known stacks and addons.
func decodeWatches(data []byte, path string) ([]Watch, error) {
	var doc struct {
		Entry []map[string]any `toml:"entry"` // checked by decode
		Watch []map[string]any `toml:"watch"`
	}
	if _, err := toml.Decode(string(data), &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	known := []string{"paths", "except", "all", "stacks", "addons"}
	for i, m := range doc.Watch {
		for k := range m {
			if !slices.Contains(known, k) {
				return nil, fmt.Errorf("%s: watch %d: unknown key %q", path, i+1, k)
			}
		}
	}
	var typed struct {
		Watch []Watch `toml:"watch"`
	}
	if _, err := toml.Decode(string(data), &typed); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err) // a key of the wrong type
	}
	refs := references()
	for i, w := range typed.Watch {
		if err := checkWatch(w, refs); err != nil {
			return nil, fmt.Errorf("%s: watch %d: %w", path, i+1, err)
		}
	}
	return typed.Watch, nil
}

func checkWatch(w Watch, refs refs) error {
	if len(w.Paths) == 0 {
		return fmt.Errorf("no paths")
	}
	for _, g := range slices.Concat(w.Paths, w.Except) {
		if _, err := pathkey.Glob(g, ""); err != nil {
			return fmt.Errorf("glob %q: %w", g, err)
		}
	}
	n := 0
	for _, set := range []bool{w.All, len(w.Stacks) > 0, len(w.Addons) > 0} {
		if set {
			n++
		}
	}
	if n != 1 {
		return fmt.Errorf("needs exactly one of all = true, stacks, addons; has %d", n)
	}
	for _, s := range w.Stacks {
		if _, ok := runtimeOf[s]; !ok {
			return fmt.Errorf("stack %q; stacks are gdscript, python, go", s)
		}
	}
	for _, a := range w.Addons {
		if !slices.ContainsFunc(refs.frameworks, func(f verify.Framework) bool { return f.Name == a }) {
			return fmt.Errorf("addon %q is no framework", a)
		}
	}
	return nil
}

// WatchedEntries is the keys of the entries, in list order, that the rules of
// the list select for the changed files (repo-relative, slash separated).
func WatchedEntries(entries []Entry, changed []string) ([]string, error) {
	ws, _ := Watches() // the embedded list is validated by this package's tests
	return matchWatches(entries, ws, changed)
}

func matchWatches(entries []Entry, ws []Watch, changed []string) ([]string, error) {
	hit := map[string]bool{}
	for _, w := range ws {
		touched, err := touches(w, changed)
		if err != nil {
			return nil, err
		}
		if !touched {
			continue
		}
		for _, e := range entries {
			if w.All || slices.Contains(w.Stacks, e.Stack) ||
				slices.ContainsFunc(e.Addons, func(a Addon) bool { return slices.Contains(w.Addons, a.Name) }) {
				hit[e.Key()] = true
			}
		}
	}
	var keys []string
	for _, e := range entries {
		if hit[e.Key()] {
			keys = append(keys, e.Key())
		}
	}
	return keys, nil
}

// touches says whether a changed file is under a path of w and not under an
// except of it.
func touches(w Watch, changed []string) (bool, error) {
	for _, f := range changed {
		in, err := matchesAny(w.Paths, f)
		if err != nil {
			return false, err
		}
		if !in {
			continue
		}
		out, err := matchesAny(w.Except, f)
		if err != nil || !out {
			return err == nil, err
		}
	}
	return false, nil
}

func matchesAny(globs []string, f string) (bool, error) {
	for _, g := range globs {
		if ok, err := pathkey.Glob(g, f); err != nil || ok {
			return ok, err
		}
	}
	return false, nil
}
