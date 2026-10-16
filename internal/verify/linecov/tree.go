package linecov

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Tree is the root's files, walked once and shared by path resolution, the
// suffix search and the expected sources.
type Tree struct {
	Root   string   // absolute and clean
	Files  []string // slash paths below Root, sorted; .git and other checkouts left out
	has    map[string]bool
	nested []string
}

// foldCase compares the root prefix of an absolute path without case, as the
// file system does on Windows.
var foldCase = runtime.GOOS == "windows"

// absPath is a seam: filepath.Abs only fails when the working directory is gone.
var absPath = filepath.Abs

// ScanTree walks root. A directory named .godot is left out at any depth: it
// is Godot's import cache and where check godot-suite writes its glue, and no
// tool's sources live there. A directory holding a .git entry is another checkout
// (a linked worktree, a submodule): its files belong to another project.
// ponytail: walks node_modules and build output too; skip them when a large
// JavaScript tree makes the walk measurably slow.
func ScanTree(root string) (*Tree, error) {
	abs, err := absPath(root)
	if err != nil {
		return nil, err
	}
	// WalkDir does not descend into a root that is a link; with a trailing
	// separator the root resolves as a directory, a junction included (which
	// EvalSymlinks leaves alone). Root keeps the spelling reports and
	// settings use.
	walk := abs
	if !os.IsPathSeparator(walk[len(walk)-1]) {
		walk += string(filepath.Separator)
	}
	t := &Tree{Root: abs, has: map[string]bool{}}
	err = filepath.WalkDir(walk, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == walk {
			return err
		}
		rel, _ := filepath.Rel(walk, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".godot" {
				return filepath.SkipDir
			}
			if _, err := os.Lstat(filepath.Join(p, ".git")); err == nil {
				t.nested = append(t.nested, rel)
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && d.Name() != ".git" {
			t.Files = append(t.Files, rel)
			t.has[rel] = true
		}
		return nil
	})
	slices.Sort(t.Files)
	return t, err
}

func (t *Tree) Has(rel string) bool { return t.has[rel] }

func (t *Tree) InNested(rel string) bool {
	return slices.ContainsFunc(t.nested, func(n string) bool { return rel == n || strings.HasPrefix(rel, n+"/") })
}

type Outcome int

const (
	Resolved  Outcome = iota
	Outside           // not below the root: a system header, a dependency
	Nested            // inside another checkout below the root
	Missing           // below the root but not on disk
	Ambiguous         // a suffix search found more than one file
)

type Resolution struct {
	Rel        string
	Outcome    Outcome
	Candidates []string
}

// Resolve turns a report's spelling of a path into a slash path relative to
// the root: backslashes and res:// first, then a suffix search, an absolute
// path, the report's bases in order, and last the path as it stands.
func (t *Tree) Resolve(rec Record) Resolution {
	p := strings.TrimPrefix(strings.ReplaceAll(rec.Path, `\`, "/"), "res://")
	if rec.Suffix {
		return t.bySuffix(p)
	}
	if isAbs(p) {
		rel, ok := t.relTo(p)
		if !ok {
			return Resolution{Rel: p, Outcome: Outside}
		}
		return t.check(rel)
	}
	for _, b := range rec.Bases {
		joined := path.Join(strings.ReplaceAll(b, `\`, "/"), p)
		rel, ok := cleanRel(joined)
		if isAbs(joined) {
			rel, ok = t.relTo(joined)
		}
		if ok && t.has[rel] {
			return t.check(rel)
		}
	}
	rel, ok := cleanRel(p)
	if !ok {
		return Resolution{Rel: p, Outcome: Outside}
	}
	return t.check(rel)
}

func (t *Tree) check(rel string) Resolution {
	switch {
	case t.InNested(rel):
		return Resolution{Rel: rel, Outcome: Nested}
	case !t.has[rel]:
		return Resolution{Rel: rel, Outcome: Missing}
	}
	return Resolution{Rel: rel, Outcome: Resolved}
}

func (t *Tree) bySuffix(p string) Resolution {
	var found []string
	for _, f := range t.Files {
		if f == p || strings.HasSuffix(f, "/"+p) {
			found = append(found, f)
		}
	}
	switch len(found) {
	case 0:
		return Resolution{Rel: p, Outcome: Missing}
	case 1:
		return Resolution{Rel: found[0], Outcome: Resolved}
	}
	return Resolution{Rel: p, Outcome: Ambiguous, Candidates: found}
}

// relTo cuts the root off an absolute slash path.
func (t *Tree) relTo(p string) (string, bool) {
	prefix := strings.TrimSuffix(filepath.ToSlash(t.Root), "/") + "/"
	if len(p) < len(prefix) {
		return "", false
	}
	got := p[:len(prefix)]
	if got != prefix && !(foldCase && strings.EqualFold(got, prefix)) {
		return "", false
	}
	return cleanRel(p[len(prefix):])
}

// cleanRel cleans a relative slash path and refuses one that leaves the root.
func cleanRel(p string) (string, bool) {
	c := path.Clean(p)
	return c, c != "." && c != ".." && !strings.HasPrefix(c, "../")
}

func isAbs(p string) bool {
	return strings.HasPrefix(p, "/") || (len(p) >= 3 && p[1] == ':' && p[2] == '/')
}
