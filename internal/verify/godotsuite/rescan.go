package godotsuite

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/child"
)

// RescanOptions is one `loomux godot rescan`.
type RescanOptions struct {
	Root, Godot string
	Start       func(child.Spec) child.Result
	Status      func(root string) (string, error) // git status --porcelain -- <root>
}

const classCache = ".godot/global_script_class_cache.cfg"

// Rescan has the engine import the project, which refreshes the class cache,
// and puts project.godot back as it was: the import rewrites it. Other files
// the import touched are named, not reset.
func Rescan(o RescanOptions) (notes []string, err error) {
	project := filepath.Join(o.Root, "project.godot")
	before, err := os.ReadFile(project)
	if err != nil {
		return nil, err
	}
	statusBefore, err := o.Status(o.Root)
	if err != nil {
		return nil, err
	}
	// Deferred so that a failing engine, which has rewritten the file all
	// the same, does not leave the rewrite behind.
	defer func() {
		after, readErr := os.ReadFile(project)
		if readErr != nil || string(after) == string(before) {
			return
		}
		if writeErr := os.WriteFile(project, before, 0o644); writeErr != nil {
			err = errors.Join(err, writeErr)
			return
		}
		notes = append([]string{"restored project.godot, which the engine rewrote"}, notes...)
	}()

	start := o.Start
	if start == nil {
		start = child.Run
	}
	r := start(child.Spec{Dir: o.Root, Argv: []string{o.Godot, "--path", o.Root, "--headless", "--import"}})
	if r.Err != nil || r.Code != 0 {
		return nil, errors.New("the engine failed: " + outputOf(r))
	}
	if _, statErr := os.Stat(filepath.Join(o.Root, filepath.FromSlash(classCache))); statErr != nil {
		return nil, errors.New("the engine ran but wrote no " + classCache)
	}
	statusAfter, err := o.Status(o.Root)
	if err != nil {
		return nil, err
	}
	return changedBeside(statusBefore, statusAfter), nil
}

// changedBeside names the paths of after that before did not have, and says
// what the engine's own products among them are.
func changedBeside(before, after string) []string {
	had := map[string]bool{}
	for _, l := range strings.Split(before, "\n") {
		had[l] = true
	}
	var paths []string
	for _, l := range strings.Split(after, "\n") {
		if len(l) > 3 && !had[l] {
			paths = append(paths, l[3:])
		}
	}
	if len(paths) == 0 {
		return nil
	}
	slices.Sort(paths)
	notes := []string{"the engine also changed: " + strings.Join(paths, ", ")}
	if slices.ContainsFunc(paths, func(p string) bool { return strings.HasSuffix(p, ".uid") || strings.HasSuffix(p, ".import") }) {
		notes = append(notes, "*.uid and *.import files are products of the engine; they are reported, not reset")
	}
	return notes
}
