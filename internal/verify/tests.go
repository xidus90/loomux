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
// name ending in "/" is a directory, "file:text" a file holding text (file a
// glob on the base name, as in "*.csproj:Microsoft.NET.Test.Sdk"), any other
// name with a slash one literal path below root, which may lie in a skipped
// directory (vendor/bin/phpunit), and anything else a glob on the base name.
// The walk stops at the first match.
func HasTests(root string, patterns []string) bool {
	var walked []string
	for _, p := range patterns {
		if !literalPath(p) {
			walked = append(walked, p)
		} else if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err == nil {
			return true
		}
	}
	if len(walked) == 0 {
		return false
	}
	patterns = walked
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

// found is HasTests in dir, and for what Node resolves from a directory
// upwards also in each directory above it up to the root: a workspace
// hoists its members' tools into the root's node_modules and declares their
// dependencies in the root's package.json. Only a path in node_modules/ and
// a package.json:text pattern look up there; for the latter only the
// package.json beside each directory counts, never one in a sibling.
func (env PlanEnv) found(dir string, patterns []string) bool {
	if env.HasTests(dir, patterns) {
		return true
	}
	root := filepath.Clean(env.Root)
	for d := filepath.Clean(dir); d != root && filepath.Dir(d) != d; {
		d = filepath.Dir(d)
		for _, p := range patterns {
			if strings.HasPrefix(p, "node_modules/") && env.HasTests(d, []string{p}) {
				return true
			}
			if text, ok := strings.CutPrefix(p, "package.json:"); ok {
				data, err := os.ReadFile(filepath.Join(d, "package.json"))
				if err == nil && strings.Contains(string(data), text) {
					return true
				}
			}
		}
	}
	return false
}

// literalPath says whether a pattern names one path rather than a glob, a
// directory or a file holding text.
func literalPath(p string) bool {
	return strings.Contains(p, "/") && !strings.HasSuffix(p, "/") && !strings.Contains(p, ":")
}

func matchesTest(path string, d fs.DirEntry, pattern string) bool {
	if dir, ok := strings.CutSuffix(pattern, "/"); ok {
		return d.IsDir() && d.Name() == dir
	}
	if d.IsDir() {
		return false
	}
	if name, text, ok := strings.Cut(pattern, ":"); ok {
		if ok, _ := filepath.Match(name, d.Name()); !ok {
			return false
		}
		data, err := os.ReadFile(path)
		return err == nil && strings.Contains(string(data), text)
	}
	ok, _ := filepath.Match(pattern, d.Name())
	return ok
}
