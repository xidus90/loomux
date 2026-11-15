package resolve_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/resolve"
)

// cargoOf is the manifest of the only Cargo module in bodies.
func cargoOf(t *testing.T, bodies map[string]string) *resolve.CargoManifest {
	t.Helper()
	mods, err := resolve.ModulesFrom(bodies)
	if err != nil || len(mods) != 1 || mods[0].Cargo == nil {
		t.Fatalf("got %+v, %v, want one Cargo module", mods, err)
	}
	return mods[0].Cargo
}

// writeTree writes files under a fresh root and returns it.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCargoPackageAndLib(t *testing.T) {
	c := cargoOf(t, map[string]string{"Cargo.toml": "[package]\nname = \"my-game\"\n"})
	if c.Package != "my-game" || c.Lib != "my_game" {
		t.Fatalf("got package %q, lib %q", c.Package, c.Lib)
	}
	c = cargoOf(t, map[string]string{"Cargo.toml": "[package]\nname = \"my-game\"\n[lib]\nname = \"core\"\n"})
	if c.Package != "my-game" || c.Lib != "core" {
		t.Fatalf("got package %q, lib %q", c.Package, c.Lib)
	}
}

func TestCargoRootsFromText(t *testing.T) {
	c := cargoOf(t, map[string]string{"app/Cargo.toml": `
[package]
name = "app"
[lib]
path = "src/engine.rs"
[[bin]]
name = "tool"
path = "tools/tool.rs"
[[bin]]
name = "unnamed"
[[test]]
path = "it/all.rs"
[[bench]]
path = "perf/b.rs"
[[example]]
path = "ex/e.rs"
`})
	want := []string{
		"app/ex/e.rs", "app/it/all.rs", "app/perf/b.rs", "app/src/engine.rs",
		"app/src/lib.rs", "app/src/main.rs", "app/tools/tool.rs",
	}
	if !reflect.DeepEqual(c.Roots, want) {
		t.Fatalf("got %v, want %v", c.Roots, want)
	}
	if c.LibRoot != "app/src/engine.rs" {
		t.Fatalf("lib root %q, want the [lib] path", c.LibRoot)
	}
	if c := cargoOf(t, map[string]string{"Cargo.toml": appManifest}); c.LibRoot != "src/lib.rs" {
		t.Fatalf("lib root %q, want src/lib.rs", c.LibRoot)
	}
}

func TestCargoRootsFromTheDirectoryListing(t *testing.T) {
	root := writeTree(t, map[string]string{
		"crate/Cargo.toml":           "[package]\nname = \"c\"\n",
		"crate/src/lib.rs":           "",
		"crate/src/bin/a.rs":         "",
		"crate/src/bin/b/main.rs":    "",
		"crate/src/bin/b/other.rs":   "",
		"crate/tests/t1.rs":          "",
		"crate/tests/t2/main.rs":     "",
		"crate/tests/t2/helper.rs":   "",
		"crate/benches/bench.rs":     "",
		"crate/examples/ex.rs":       "",
		"crate/examples/ex2/main.rs": "",
	})
	mods, err := resolve.Modules(root, []string{"crate/Cargo.toml"})
	if err != nil || len(mods) != 1 {
		t.Fatalf("got %+v, %v", mods, err)
	}
	want := []string{
		"crate/benches/bench.rs", "crate/examples/ex.rs", "crate/examples/ex2/main.rs",
		"crate/src/bin/a.rs", "crate/src/bin/b/main.rs", "crate/src/lib.rs", "crate/src/main.rs",
		"crate/tests/t1.rs", "crate/tests/t2/main.rs",
	}
	if got := mods[0].Cargo.Roots; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCargoWorkspace(t *testing.T) {
	root := writeTree(t, map[string]string{
		"Cargo.toml":                   "[workspace]\nmembers = [\"crates/*\", \"tools/cli\"]\nexclude = [\"crates/skip\"]\n",
		"crates/a/Cargo.toml":          "[package]\nname = \"a\"\n",
		"crates/b/Cargo.toml":          "[package]\nname = \"b\"\n",
		"crates/skip/Cargo.toml":       "[package]\nname = \"skip\"\n",
		"crates/skip/inner/Cargo.toml": "[package]\nname = \"inner\"\n",
		"crates/notes/readme.md":       "",
		"crates/file.txt":              "",
		"tools/cli/Cargo.toml":         "[package]\nname = \"cli\"\n",
	})
	mods, err := resolve.Modules(root, []string{"Cargo.toml"})
	if err != nil || len(mods) != 1 {
		t.Fatalf("got %+v, %v", mods, err)
	}
	c := mods[0].Cargo
	if c.Package != "" || c.Lib != "" || c.LibRoot != "" || len(c.Roots) != 0 {
		t.Fatalf("a virtual workspace has no package: %+v", c)
	}
}

func TestCargoRootPackageInAWorkspaceKeepsItsRoots(t *testing.T) {
	root := writeTree(t, map[string]string{
		"w/Cargo.toml":       "[package]\nname = \"w\"\n[workspace]\nmembers = [\".\"]\n",
		"w/src/bin/extra.rs": "",
	})
	mods, _ := resolve.Modules(root, []string{"w/Cargo.toml"})
	c := mods[0].Cargo
	if !reflect.DeepEqual(c.Roots, []string{"w/src/bin/extra.rs", "w/src/lib.rs", "w/src/main.rs"}) {
		t.Fatalf("got %+v", c)
	}
}

func TestCargoPathDeps(t *testing.T) {
	c := cargoOf(t, map[string]string{"crates/app/Cargo.toml": `
[package]
name = "app"
[dependencies]
foo = { path = "../foo" }
bar = { package = "real-bar", path = "../bar" }
my-util = { path = "../util" }
serde = "1"
tokio = { version = "1" }
outside = { path = "../../../elsewhere" }
up = { path = "../.." }
vendored = { path = "../../.vendor/foo" }
[dev-dependencies]
dev = { path = "../dev" }
[build-dependencies]
build = { path = "../build" }
`})
	want := map[string]string{
		"foo": "crates/foo", "bar": "crates/bar", "my_util": "crates/util",
		"dev": "crates/dev", "build": "crates/build", "up": "", "vendored": ".vendor/foo",
	}
	if !reflect.DeepEqual(c.Deps, want) {
		t.Fatalf("got %v, want %v", c.Deps, want)
	}
}

func TestCargoWorkspaceDeps(t *testing.T) {
	mods, err := resolve.ModulesFrom(map[string]string{
		"Cargo.toml":          "[workspace]\nmembers = [\"crates/a\"]\n[workspace.dependencies]\nbaz = { path = \"crates/baz\" }\nreg = \"1\"\n",
		"crates/a/Cargo.toml": "[package]\nname = \"a\"\n[dependencies]\nbaz.workspace = true\nreg.workspace = true\nmissing.workspace = true\n",
	})
	if err != nil || len(mods) != 2 {
		t.Fatalf("got %+v, %v", mods, err)
	}
	for _, m := range mods {
		if m.Dir == "crates/a" {
			if want := map[string]string{"baz": "crates/baz"}; !reflect.DeepEqual(m.Cargo.Deps, want) {
				t.Fatalf("got %v, want %v", m.Cargo.Deps, want)
			}
		}
	}
}

func TestCargoWorkspaceDepsNeedAWorkspaceAbove(t *testing.T) {
	c := cargoOf(t, map[string]string{"app/Cargo.toml": "[package]\nname = \"a\"\n[dependencies]\nbaz.workspace = true\n"})
	if len(c.Deps) != 0 {
		t.Fatalf("got %v", c.Deps)
	}
}

func TestCargoWorkspaceDepsUseTheNearestWorkspace(t *testing.T) {
	mods, _ := resolve.ModulesFrom(map[string]string{
		"Cargo.toml":            "[workspace]\n[workspace.dependencies]\nx = { path = \"outer\" }\n",
		"sub/Cargo.toml":        "[workspace]\n[workspace.dependencies]\nx = { path = \"inner\" }\n",
		"sub/member/Cargo.toml": "[package]\nname = \"m\"\n[dependencies]\nx.workspace = true\n",
		"subway/Cargo.toml":     "[package]\nname = \"s\"\n[dependencies]\nx.workspace = true\n",
	})
	got := map[string]string{}
	for _, m := range mods {
		if m.Cargo.Package != "" {
			got[m.Dir] = m.Cargo.Deps["x"]
		}
	}
	want := map[string]string{"sub/member": "sub/inner", "subway": "outer"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCargoBrokenManifest(t *testing.T) {
	for name, body := range map[string]string{
		"syntax":      "[package\nname = ",
		"empty":       "",
		"neither":     "[dependencies]\nfoo = \"1\"\n",
		"wrong type":  "[package]\nname = 3\n",
		"not a table": "package = \"x\"\n",
	} {
		mods, err := resolve.ModulesFrom(map[string]string{"Cargo.toml": body})
		if err != nil || len(mods) != 0 {
			t.Errorf("%s: got %+v, %v, want no module and no error", name, mods, err)
		}
	}
}

func TestCargoUnreadableManifestIsNoError(t *testing.T) {
	mods, err := resolve.Modules(t.TempDir(), []string{"Cargo.toml"})
	if err != nil || len(mods) != 0 {
		t.Fatalf("got %+v, %v", mods, err)
	}
}

func TestModuleFileKnowsCargoToml(t *testing.T) {
	for rel, want := range map[string]bool{"Cargo.toml": true, "crates/a/Cargo.toml": true, "cargo.toml": false, "Cargo.lock": false} {
		if resolve.ModuleFile(rel) != want {
			t.Errorf("ModuleFile(%q) != %v", rel, want)
		}
	}
}
