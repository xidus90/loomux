package godotsuite

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/child"
)

// Runner starts one process; it is the seam Options.Start and
// RescanOptions.Start fill.
type Runner func(child.Spec) child.Result

// rustEntrySymbol is what a godot-rust extension names as its entry point;
// every other .gdextension (C++, a third party's addon) is left alone.
const rustEntrySymbol = "gdext_rust_init"

// RustBuild is one cargo build an extension of the project needs.
type RustBuild struct {
	Crate   string // directory of the Cargo.toml, absolute
	Profile string // "debug", "release" or a custom profile name
}

// RustBuilds lists the builds the Rust extensions of a Godot project need:
// one per crate and profile that a [libraries] path points into as
// <dir>/target/<profile>/. Godot picks a library by feature tags, so no
// platform key is chosen here. A library whose key carries the `release`
// feature tag is skipped: the suite and the import load the debug or editor
// library. A library or crate outside root (the repository) is never built: the
// search for a Cargo.toml stops there. What cannot be placed is named in notes.
func RustBuilds(project, root string) (builds []RustBuild, notes []string) {
	seen := map[RustBuild]bool{}
	wanted, releaseOnly := 0, 0
	err := filepath.WalkDir(project, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != project && (d.Name() == ".godot" || d.Name() == ".git" || d.Name() == "target") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(p) != ".gdextension" {
			return nil
		}
		data, err := readFile(p)
		if err != nil {
			notes = append(notes, fmt.Sprintf("cannot read %s: %v", p, err))
			return nil
		}
		libs, ok := rustLibraries(data)
		if !ok {
			return nil
		}
		for _, lib := range libs {
			target, profile, ok := profileDir(filepath.Join(project, filepath.FromSlash(lib.path)))
			if !ok {
				continue
			}
			if lib.release {
				releaseOnly++
				continue
			}
			wanted++
			if !within(root, target) {
				notes = append(notes, fmt.Sprintf("%s is outside the repository %s: not built", target, root))
				continue
			}
			crate, ok := manifestAbove(target, root)
			if !ok {
				notes = append(notes, "no Cargo.toml above "+target)
				continue
			}
			seen[RustBuild{Crate: crate, Profile: profile}] = true
		}
		return nil
	})
	if err != nil {
		notes = append(notes, "cannot list the project for extensions: "+err.Error())
	}
	if wanted == 0 && releaseOnly > 0 {
		notes = append(notes, "only release libraries are listed and the suite loads debug or editor ones: nothing was built")
	}
	for b := range seen {
		builds = append(builds, b)
	}
	if env := os.Getenv("CARGO_TARGET_DIR"); env != "" && len(seen) > 0 {
		notes = append(notes, "CARGO_TARGET_DIR is set ("+env+"): cargo builds there, not into the target/ the .gdextension names, and Godot loads the library found there")
	}
	slices.SortFunc(builds, func(a, b RustBuild) int {
		return strings.Compare(a.Crate+"\x00"+a.Profile, b.Crate+"\x00"+b.Profile)
	})
	slices.Sort(notes)
	return builds, slices.Compact(notes)
}

// rustLib is one [libraries] entry: its project-relative path and whether its
// key carries the release feature tag.
type rustLib struct {
	path    string
	release bool
}

// rustLibraries reads a .gdextension (Godot's config format) and returns its
// [libraries], when its entry symbol is the Rust one.
func rustLibraries(data []byte) (libs []rustLib, rust bool) {
	var section string
	sc := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))))
	for sc.Scan() {
		line := strings.TrimSpace(stripComment(sc.Text()))
		if name, ok := strings.CutPrefix(line, "["); ok {
			section = strings.TrimSuffix(name, "]")
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch {
		case section == "configuration" && strings.TrimSpace(key) == "entry_symbol":
			rust = value == rustEntrySymbol
		case section == "libraries":
			if rel, ok := strings.CutPrefix(value, "res://"); ok {
				tags := strings.Split(strings.TrimSpace(key), ".")
				libs = append(libs, rustLib{rel, slices.Contains(tags, "release") || slices.Contains(tags, "template_release")})
			}
		}
	}
	return libs, rust
}

// stripComment cuts a trailing `;` comment; a `;` inside a quoted value stays.
func stripComment(line string) string {
	quoted := false
	for i, c := range line {
		switch {
		case c == '"':
			quoted = !quoted
		case c == ';' && !quoted:
			return line[:i]
		}
	}
	return line
}

// profileDir splits <target>/<profile>/<file> into its target directory and
// profile; any other shape (a target triple, deps/, no target at all) is not
// the host build a plain `cargo build` writes.
func profileDir(lib string) (target, profile string, ok bool) {
	dir := filepath.Dir(lib)
	target = filepath.Dir(dir)
	return target, filepath.Base(dir), filepath.Base(target) == "target"
}

// within reports whether p is root or below it.
func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// manifestAbove is the nearest directory above target, up to root, that holds
// a Cargo.toml: for a workspace that is its root, where cargo builds from.
func manifestAbove(target, root string) (string, bool) {
	for dir := filepath.Dir(target); within(root, dir); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
			return dir, true
		}
		if dir == root {
			break
		}
	}
	return "", false
}

// BuildRust runs cargo for one build; the error carries cargo's output.
func BuildRust(run Runner, b RustBuild) error {
	argv := []string{"cargo", "build"}
	switch b.Profile {
	case "debug":
	case "release":
		argv = append(argv, "--release")
	default:
		argv = append(argv, "--profile", b.Profile)
	}
	r := run(child.Spec{Argv: argv, Dir: b.Crate})
	cmd := strings.Join(argv, " ")
	switch {
	case errors.Is(r.Err, exec.ErrNotFound):
		return errors.New("cargo is not on PATH: " + cmd)
	case r.Err != nil:
		return fmt.Errorf("%s could not start: %w", cmd, r.Err)
	case r.TimedOut:
		return fmt.Errorf("%s timed out:\n%s", cmd, outputOf(r))
	case r.Code != 0:
		return fmt.Errorf("%s failed (exit %d):\n%s", cmd, r.Code, outputOf(r))
	}
	return nil
}

// repositoryOf is the git checkout holding project: the nearest directory with
// a .git, which a worktree has as a file. Outside one it is the directory
// around the project, where the godot-rust layout keeps its crate beside it.
func repositoryOf(project string) string {
	for dir := project; ; dir = filepath.Dir(dir) {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		if filepath.Dir(dir) == dir {
			return filepath.Dir(project)
		}
	}
}

// buildExtensions builds every Rust extension of the project before an
// engine starts, stopping at the first build that fails.
func buildExtensions(run Runner, project string) (notes []string, err error) {
	builds, notes := RustBuilds(project, repositoryOf(project))
	for _, b := range builds {
		if err = BuildRust(run, b); err != nil {
			return notes, err
		}
	}
	return notes, nil
}
