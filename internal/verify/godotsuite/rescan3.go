package godotsuite

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/verify"
)

// rescanWindow is how long a Godot 3 editor gets to write its class list.
// Godot 3.6's editor does not quit on --quit, so the run always ends by this
// limit; it wrote the list of a small project within 3 s (measured on
// Windows, Godot 3.6.3).
const rescanWindow = 30 * time.Second

// classListKeys are the keys of project.godot that hold Godot 3's global
// classes, the only part of its rewrite a rescan keeps.
var classListKeys = []string{"_global_script_classes=", "_global_script_class_icons="}

// rescan3 has a Godot 3 editor write the class list into project.godot and
// keeps that list alone: Godot 3 has no class cache beside the project, and
// the editor's other rewrites (its header, settings it fills in) go back.
func rescan3(o RescanOptions, start func(child.Spec) child.Result, before []byte) ([]string, error) {
	project := filepath.Join(o.Root, "project.godot")
	statusBefore, err := o.Status(o.Root)
	if err != nil {
		return nil, err
	}
	window := o.Window
	if window == 0 {
		window = rescanWindow
	}
	r := start(child.Spec{Dir: o.Root, Argv: []string{o.Godot, "--no-window", "--editor", "--path", o.Root}, Timeout: window})
	// The editor ends only by the window; one that ended before it with an
	// error never ran.
	if r.Err != nil || (!r.TimedOut && r.Code != 0) {
		return nil, errors.New("the engine failed: " + outputOf(r))
	}
	after, _ := os.ReadFile(project)
	blocks, _ := cutClassList(string(after))
	missing := func() error {
		m, _ := verify.ClassListMissing(o.Root)
		return errors.New("the editor listed no class for: " + strings.Join(m, ", "))
	}
	if blocks == "" {
		// Nothing written, so the project stays exactly as it was.
		if err := os.WriteFile(project, before, 0o644); err != nil {
			return nil, err
		}
		return nil, missing()
	}
	// A list that lacks a class is still the editor's true list: it stays,
	// and the error names what it lacks.
	if err := os.WriteFile(project, []byte(withClassList(string(before), blocks)), 0o644); err != nil {
		return nil, err
	}
	if m, _ := verify.ClassListMissing(o.Root); len(m) > 0 {
		return nil, missing()
	}
	statusAfter, err := o.Status(o.Root)
	if err != nil {
		return nil, err
	}
	return append([]string{"kept the class list the editor wrote into project.godot; restored the rest"},
		changedBeside(statusBefore, statusAfter)...), nil
}

// cutClassList splits a project file into its class-list keys, each with its
// whole value (a value spans lines until its brackets close), and the rest.
func cutClassList(src string) (blocks, rest string) {
	var in, out strings.Builder
	lines := strings.SplitAfter(src, "\n")
	for i := 0; i < len(lines); i++ {
		if !startsClassList(lines[i]) {
			out.WriteString(lines[i])
			continue
		}
		for depth := 0; i < len(lines); i++ {
			in.WriteString(lines[i])
			depth += strings.Count(lines[i], "[") + strings.Count(lines[i], "{") - strings.Count(lines[i], "]") - strings.Count(lines[i], "}")
			if depth <= 0 {
				break
			}
		}
	}
	return in.String(), out.String()
}

// withClassList is original with its class-list keys replaced by blocks,
// which go right after config_version, or first where there is none.
func withClassList(original, blocks string) string {
	_, rest := cutClassList(original)
	lines := strings.SplitAfter(rest, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "config_version=") {
			return strings.Join(lines[:i+1], "") + blocks + strings.Join(lines[i+1:], "")
		}
	}
	return blocks + rest
}

func startsClassList(line string) bool {
	for _, k := range classListKeys {
		if strings.HasPrefix(line, k) {
			return true
		}
	}
	return false
}
