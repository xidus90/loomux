package resolve

import (
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

// CargoManifest is what Rust resolution takes from one Cargo.toml. Every path
// is repo-relative and slash-separated.
type CargoManifest struct {
	Package string            // [package] name; "" for a virtual workspace
	Lib     string            // library name: [lib] name, or Package with '-' -> '_'
	LibRoot string            // the library's root: [lib] path, or src/lib.rs; "" for a virtual workspace
	Roots   []string          // crate roots: lib, bins, tests, benches, examples
	Deps    map[string]string // name in code -> repo-relative dir of a path dependency
}

// cargoFile is the part of a Cargo.toml this stage reads.
type cargoFile struct {
	Package struct {
		Name string `toml:"name"`
	} `toml:"package"`
	Lib struct {
		Name string `toml:"name"`
		Path string `toml:"path"`
	} `toml:"lib"`
	Bin       []cargoTarget `toml:"bin"`
	Test      []cargoTarget `toml:"test"`
	Bench     []cargoTarget `toml:"bench"`
	Example   []cargoTarget `toml:"example"`
	Workspace *struct {
		Dependencies map[string]cargoDep `toml:"dependencies"`
	} `toml:"workspace"`
	Dependencies      map[string]cargoDep `toml:"dependencies"`
	DevDependencies   map[string]cargoDep `toml:"dev-dependencies"`
	BuildDependencies map[string]cargoDep `toml:"build-dependencies"`
}

type cargoTarget struct {
	Path string `toml:"path"`
}

// cargoDep is one dependency entry: only a `path` and `workspace = true` matter,
// so a version string (a registry dependency) reads as empty.
type cargoDep struct {
	Path      string
	Workspace bool
}

// UnmarshalTOML takes an arbitrary value, because an entry is a version string
// or a table, and the decoder hands either over untyped.
func (d *cargoDep) UnmarshalTOML(v any) error {
	if t, ok := v.(map[string]any); ok {
		d.Path, _ = t["path"].(string)
		d.Workspace, _ = t["workspace"].(bool)
	}
	return nil
}

// cargoConventionalRoots are the crate roots Cargo finds by their place under
// the manifest's directory, besides src/lib.rs and src/main.rs.
var cargoConventionalRoots = []string{
	"src/bin/*.rs", "src/bin/*/main.rs", "tests/*.rs", "tests/*/main.rs",
	"benches/*.rs", "benches/*/main.rs", "examples/*.rs", "examples/*/main.rs",
}

// parsedCargo is a manifest with the directory it lies in.
type parsedCargo struct {
	dir  string
	file cargoFile
}

// cargoModules reads every Cargo.toml among bodies. fsys is the repository, or
// nil when only text is at hand: then globs (workspace members, src/bin/*.rs,
// tests/*.rs, ...) expand to nothing and literal member dirs stand as written.
//
// A manifest that does not parse, or that is neither a package nor a
// workspace, gives no module.
func cargoModules(bodies map[string]string, rels []string, fsys fs.FS) []Module {
	var all []parsedCargo
	for _, rel := range rels {
		if path.Base(rel) != "Cargo.toml" {
			continue
		}
		var f cargoFile
		if _, err := toml.Decode(bodies[rel], &f); err != nil || (f.Package.Name == "" && f.Workspace == nil) {
			continue
		}
		all = append(all, parsedCargo{dir: dirOf(rel), file: f})
	}
	var out []Module
	for _, p := range all {
		out = append(out, Module{Dir: p.dir, Cargo: p.manifest(all, fsys)})
	}
	return out
}

func (p parsedCargo) manifest(all []parsedCargo, fsys fs.FS) *CargoManifest {
	f := p.file
	m := &CargoManifest{Package: f.Package.Name, Deps: map[string]string{}}
	m.Lib = f.Lib.Name
	if m.Lib == "" {
		m.Lib = f.Package.Name
	}
	m.Lib = codeName(m.Lib)

	roots := map[string]bool{}
	add := func(rels ...string) {
		for _, r := range rels {
			roots[path.Join(p.dir, r)] = true
		}
	}
	if f.Package.Name != "" {
		add("src/lib.rs", "src/main.rs")
		m.LibRoot = path.Join(p.dir, "src/lib.rs")
		if f.Lib.Path != "" {
			add(f.Lib.Path)
			m.LibRoot = path.Join(p.dir, f.Lib.Path)
		}
		for _, t := range slices.Concat(f.Bin, f.Test, f.Bench, f.Example) {
			if t.Path != "" {
				add(t.Path)
			}
		}
		for _, pattern := range cargoConventionalRoots {
			for _, r := range globIn(fsys, path.Join(p.dir, pattern)) {
				roots[r] = true
			}
		}
	}
	m.Roots = slices.Sorted(maps.Keys(roots))

	for _, deps := range []map[string]cargoDep{f.Dependencies, f.DevDependencies, f.BuildDependencies} {
		for name, d := range deps {
			dir, base := p.dir, d.Path
			if d.Workspace {
				// The entry lives in the nearest workspace root above, and its
				// path is relative to that root.
				ws, ok := workspaceOf(all, p.dir)
				if !ok {
					continue
				}
				dir, base = ws.dir, ws.file.Workspace.Dependencies[name].Path
			}
			if base == "" {
				continue
			}
			// A path outside the repository names no file the graph holds.
			if target := path.Join(dir, base); target != ".." && !strings.HasPrefix(target, "../") {
				if target == "." {
					target = ""
				}
				m.Deps[codeName(name)] = target
			}
		}
	}
	return m
}

// workspaceOf is the manifest with a [workspace] table whose directory is dir
// or lies deepest above it.
func workspaceOf(all []parsedCargo, dir string) (parsedCargo, bool) {
	var best parsedCargo
	found := false
	for _, c := range all {
		if c.file.Workspace == nil || (c.dir != "" && c.dir != dir && !strings.HasPrefix(dir, c.dir+"/")) {
			continue
		}
		if !found || len(c.dir) > len(best.dir) {
			best, found = c, true
		}
	}
	return best, found
}

// globIn is the files of fsys matching pattern; without a filesystem a glob
// matches nothing.
func globIn(fsys fs.FS, pattern string) []string {
	if fsys == nil {
		return nil
	}
	found, _ := fs.Glob(fsys, pattern)
	return found
}

// codeName is the name a crate has in Rust code: Cargo spells it with '-',
// code with '_'.
func codeName(n string) string {
	return strings.ReplaceAll(n, "-", "_")
}
