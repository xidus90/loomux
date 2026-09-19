package verify

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// skipped directories hold code nobody here wrote, build output or no code
// at all; a test found only there says nothing about the project. Every dot
// directory is among them, as in detection: .git, .loomux, .venv, .tox.
func skipped(name string) bool {
	return strings.HasPrefix(name, ".") ||
		slices.Contains([]string{"vendor", "node_modules", "third_party", "venv", "build", "target", "dist"}, name)
}

// HasTests reports whether anything under root matches one of patterns: a
// name ending in "/" is a directory, "file:text" a file holding text, and
// anything else a glob on the base name. The walk stops at the first match.
func HasTests(root string, patterns []string) bool {
	found := false
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && path != root && skipped(d.Name()) {
			return filepath.SkipDir
		}
		if slices.ContainsFunc(patterns, func(p string) bool { return matchesTest(path, d, p) }) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func matchesTest(path string, d fs.DirEntry, pattern string) bool {
	if dir, ok := strings.CutSuffix(pattern, "/"); ok {
		return d.IsDir() && d.Name() == dir
	}
	if d.IsDir() {
		return false
	}
	if name, text, ok := strings.Cut(pattern, ":"); ok {
		if d.Name() != name {
			return false
		}
		data, err := os.ReadFile(path)
		return err == nil && strings.Contains(string(data), text)
	}
	ok, _ := filepath.Match(pattern, d.Name())
	return ok
}
