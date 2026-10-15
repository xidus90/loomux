package linecov

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// Active is a profile the project uses, with its scope and configured
// markers read from its settings file.
type Active struct {
	Profile  Profile
	Settings string // settings file read, relative to the root; "" without one
	include  []Matcher
	exclude  []Matcher
	markers  []Marker
}

func (a Active) Owns(rel string) bool { return slices.Contains(a.Profile.Extensions, path.Ext(rel)) }

// Scoped says whether the profile knows which files must be measured.
func (a Active) Scoped() bool { return a.Profile.Scope == "all" || len(a.include) > 0 }

func (a Active) Excludes(rel string) bool {
	return slices.ContainsFunc(a.exclude, func(m Matcher) bool { return m(rel) })
}

func (a Active) Expects(rel string) bool {
	if !a.Owns(rel) || !a.Scoped() || a.Excludes(rel) {
		return false
	}
	return a.Profile.Scope == "all" || slices.ContainsFunc(a.include, func(m Matcher) bool { return m(rel) })
}

// Activate finds the profiles the project uses: by a settings file in the
// root, or by a report read from one of the profile's default locations.
func Activate(t *Tree, reports []string, ps []Profile) ([]Active, []Finding, []string) {
	var act []Active
	var finds []Finding
	var notes []string
	root := filepath.ToSlash(t.Root)
	for _, p := range ps {
		vals, file, f, n := readSettings(t.Root, p)
		finds, notes = append(finds, f...), append(notes, n...)
		byReport := slices.ContainsFunc(p.Reports, func(r string) bool { return slices.Contains(reports, r) })
		if file == "" && !byReport {
			continue
		}
		a := Active{Profile: p, Settings: file}
		finds = append(finds, a.build(vals, root)...)
		act = append(act, a)
	}
	return act, finds, notes
}

// readSettings follows the profile's lookup order; the first file holding
// the profile's section wins.
func readSettings(root string, p Profile) (Values, string, []Finding, []string) {
	for _, sf := range p.Settings {
		// Globbing below the root keeps glob characters in its name literal.
		matches, _ := fs.Glob(os.DirFS(root), sf.File)
		switch {
		case len(matches) == 0:
			continue
		case len(matches) > 1:
			return Values{}, "", nil, []string{fmt.Sprintf("%s: %d files match %s; none is read", p.Name, len(matches), sf.File)}
		}
		rel := matches[0]
		content, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return Values{}, "", []Finding{{rel, 0, "unreadable settings: " + err.Error()}}, nil
		}
		vals, found, err := syntaxes()[sf.Syntax](content, Where{Section: sf.Section, Table: sf.Table, Path: sf.Path})
		if err != nil {
			return Values{}, "", []Finding{{rel, 0, "unreadable settings: " + err.Error()}}, nil
		}
		if found {
			return vals, rel, nil, nil
		}
	}
	return Values{}, "", nil, nil
}

func (a *Active) build(vals Values, root string) []Finding {
	p := a.Profile
	var finds []Finding
	add := func(list *[]Matcher, d Dialect, pats []string) {
		for _, pat := range pats {
			m, err := d(pat, root)
			if err != nil {
				finds = append(finds, Finding{a.Settings, 0, fmt.Sprintf("unreadable pattern %q: %v", pat, err)})
				continue
			}
			*list = append(*list, m)
		}
	}
	// The dialect is nil for a profile without patterns; validate guarantees
	// it then has no globs, so add never calls it.
	dialect := dialects()[p.Patterns]
	for _, k := range p.Include.Dirs {
		add(&a.include, dirDialect, vals[k])
	}
	add(&a.include, dialect, slices.Concat(pick(vals, p.Include.Globs, p.Include.Defaults), p.Include.Always))
	for _, k := range p.Exclude.Dirs {
		add(&a.exclude, dirDialect, vals[k])
	}
	add(&a.exclude, dialect, slices.Concat(pick(vals, p.Exclude.Globs, p.Exclude.Defaults), p.Exclude.Always))
	for _, s := range p.Switches {
		on := s.Default
		if v := vals[s.Key]; len(v) > 0 {
			on = v[len(v)-1] == "true"
		}
		if on {
			add(&a.exclude, dialect, s.Exclude)
		}
	}
	for _, spec := range p.MarkerKeys {
		for _, v := range vals[spec.Key] {
			m, err := spec.compile(v)
			if err != nil {
				finds = append(finds, Finding{a.Settings, 0, fmt.Sprintf("unreadable marker %q: %v", v, err)})
				continue
			}
			a.markers = append(a.markers, m)
		}
	}
	return finds
}

// pick returns the values of keys, or defaults when none of them is set.
func pick(vals Values, keys, defaults []string) []string {
	var out []string
	for _, k := range keys {
		out = append(out, vals[k]...)
	}
	if len(out) == 0 {
		return defaults
	}
	return out
}

// Natives is every built-in marker of every profile, so an LCOV_EXCL_LINE
// needs no .lcovrc to count in a C file, plus the markers active settings
// configure. Each carries the extensions of the profile that brings it.
func Natives(ps []Profile, act []Active) []Marker {
	var out []Marker
	seen := map[string]bool{}
	add := func(m Marker, exts []string) {
		m.Extensions = exts
		if k := m.Block + "\x00" + m.Re.String() + "\x00" + strings.Join(exts, " "); !seen[k] {
			seen[k] = true
			out = append(out, m)
		}
	}
	for _, p := range ps {
		for _, s := range p.Markers {
			// validate compiled every built-in marker already.
			m, _ := s.compile("")
			add(m, p.Extensions)
		}
	}
	for _, a := range act {
		for _, m := range a.markers {
			add(m, a.Profile.Extensions)
		}
	}
	return out
}
