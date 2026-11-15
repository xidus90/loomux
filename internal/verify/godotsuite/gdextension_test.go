package godotsuite

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/child"
)

const rustHeader = "[configuration]\nentry_symbol = \"gdext_rust_init\"\ncompatibility_minimum = 4.3\nreloadable = true\n\n[libraries]\n"

// rustWorld is a Godot project beside a crate: <top>/godot is the project,
// <top>/rust the crate whose target/ the .gdextension points into.
type rustWorld struct{ top, project, crate string }

func newRustWorld(t *testing.T, gdextension string) rustWorld {
	t.Helper()
	top, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := rustWorld{top: top, project: filepath.Join(top, "godot"), crate: filepath.Join(top, "rust")}
	write(t, filepath.Join(w.project, "project.godot"), plain)
	write(t, filepath.Join(w.crate, "Cargo.toml"), "[package]\nname = \"game\"\n")
	write(t, filepath.Join(top, ".git", "HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(w.project, "game.gdextension"), gdextension)
	return w
}

func TestRustBuildsDedup(t *testing.T) {
	w := newRustWorld(t, rustHeader+
		"linux.debug.x86_64 = \"res://../rust/target/debug/libgame.so\"\n"+
		"linux.editor = \"res://../rust/target/debug/libgame.so\"\n"+
		"windows.debug = \"res://../rust/target/debug/game.dll\"\n"+
		"macos.debug.arm64 = \"res://../rust/target/debug/libgame.dylib\"\n")
	builds, notes := RustBuilds(w.project, w.top)
	if want := []RustBuild{{Crate: w.crate, Profile: "debug"}}; !reflect.DeepEqual(builds, want) {
		t.Errorf("builds %v want %v", builds, want)
	}
	if len(notes) != 0 {
		t.Errorf("notes %q", notes)
	}
}

func TestRustBuildsProfiles(t *testing.T) {
	w := newRustWorld(t, rustHeader+
		"linux.editor.x86_64 = \"res://../rust/target/release/libgame.so\"\n"+
		"linux.debug.x86_64 = \"res://../rust/target/dist/libgame.so\"\n"+
		// Not a profile directory of the host build: another target triple, a
		// deps subdirectory, a path outside any target/ and one that is no res:// path.
		"web.debug.wasm32 = \"res://../rust/target/wasm32-unknown-emscripten/debug/game.wasm\"\n"+
		"linux.template = \"res://../rust/target/debug/deps/libgame.so\"\n"+
		"linux.bin = \"res://bin/libgame.so\"\n"+
		"linux.abs = \"/opt/rust/target/debug/libgame.so\"\n"+
		"not a key value line\n")
	builds, notes := RustBuilds(w.project, w.top)
	want := []RustBuild{{Crate: w.crate, Profile: "dist"}, {Crate: w.crate, Profile: "release"}}
	if !reflect.DeepEqual(builds, want) {
		t.Errorf("builds %v want %v", builds, want)
	}
	if len(notes) != 0 {
		t.Errorf("notes %q", notes)
	}
}

// The godot-rust template lists a debug and a release path per platform; the
// suite and the import load the debug (or editor) library, so release is not built.
func TestRustBuildsSkipsReleaseFeatureKeys(t *testing.T) {
	w := newRustWorld(t, rustHeader+
		"linux.debug.x86_64 = \"res://../rust/target/debug/libgame.so\"\n"+
		"linux.release.x86_64 = \"res://../rust/target/release/libgame.so\"\n"+
		"windows.debug.x86_64 = \"res://../rust/target/debug/game.dll\"\n"+
		"windows.release = \"res://../rust/target/release/game.dll\"\n"+
		"macos.release.arm64 = \"res://../rust/target/release/libgame.dylib\"\n"+
		// A segment that merely starts with the tag is no release tag, and the
		// profile still comes from the path, not from the key.
		"linux.releases = \"res://../rust/target/dist/libgame.so\"\n")
	builds, notes := RustBuilds(w.project, w.top)
	want := []RustBuild{{Crate: w.crate, Profile: "debug"}, {Crate: w.crate, Profile: "dist"}}
	if !reflect.DeepEqual(builds, want) {
		t.Errorf("builds %v want %v", builds, want)
	}
	if len(notes) != 0 {
		t.Errorf("notes %q", notes)
	}
}

func TestRustBuildsReleaseOnlyIsANote(t *testing.T) {
	w := newRustWorld(t, rustHeader+
		"linux.release.x86_64 = \"res://../rust/target/release/libgame.so\"\n"+
		"windows.release = \"res://../rust/target/release/game.dll\"\n")
	builds, notes := RustBuilds(w.project, w.top)
	if len(builds) != 0 || len(notes) != 1 || !strings.Contains(notes[0], "release") || !strings.Contains(notes[0], "nothing") {
		t.Errorf("builds %v notes %q", builds, notes)
	}
}

func TestRustBuildsReleaseKeysOutsideTargetAreNoNote(t *testing.T) {
	w := newRustWorld(t, rustHeader+"linux.release.x86_64 = \"res://bin/libgame.so\"\n")
	builds, notes := RustBuilds(w.project, w.top)
	if len(builds) != 0 || len(notes) != 0 {
		t.Errorf("builds %v notes %q", builds, notes)
	}
}

func TestSuiteBuildsOnlyDebugOfTheTemplateShape(t *testing.T) {
	rw := newRustWorld(t, rustHeader+
		"linux.debug.x86_64 = \"res://../rust/target/debug/libgame.so\"\n"+
		"linux.release.x86_64 = \"res://../rust/target/release/libgame.so\"\n")
	w := newWorldAt(t, rw.project, plain)
	// Without .loomux/config.toml the checkout loomux finds is the project
	// itself; the crate beside it is still inside the git repository.
	w.o.CheckoutRoot = rw.project
	var cargo [][]string
	inner := w.e.start
	w.o.Start = func(s child.Spec) child.Result {
		if s.Argv[0] == "cargo" {
			cargo = append(cargo, append([]string{s.Dir}, s.Argv...))
			return child.Result{}
		}
		return inner(s)
	}
	run(t, w)
	if !reflect.DeepEqual(cargo, [][]string{{rw.crate, "cargo", "build"}}) {
		t.Fatalf("cargo calls %q", cargo)
	}
}

func TestRepositoryOf(t *testing.T) {
	top, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo, worktree := filepath.Join(top, "repo"), filepath.Join(top, "wt")
	write(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(worktree, ".git"), "gitdir: "+filepath.Join(repo, ".git", "worktrees", "wt")+"\n")
	for _, c := range []struct{ project, want string }{
		{repo, repo},
		{filepath.Join(repo, "game", "godot"), repo},
		{filepath.Join(worktree, "godot"), worktree},
	} {
		if got := repositoryOf(c.project); got != c.want {
			t.Errorf("repositoryOf(%s) = %s, want %s", c.project, got, c.want)
		}
	}
}

// Outside a git checkout, as the compat work copies are, the crate beside the
// project is still found.
func TestRepositoryOfOutsideACheckout(t *testing.T) {
	top, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for dir := top; ; dir = filepath.Dir(dir) {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			t.Skip("the temporary directory lies inside a git checkout: " + dir)
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	project := filepath.Join(top, "work", "godot")
	if got := repositoryOf(project); got != filepath.Join(top, "work") {
		t.Errorf("repositoryOf = %s", got)
	}
}

func TestSuiteSaysWhenOnlyReleaseLibrariesAreListed(t *testing.T) {
	rw := newRustWorld(t, rustHeader+"linux.release.x86_64 = \"res://../rust/target/release/libgame.so\"\n")
	w := newWorldAt(t, rw.project, plain)
	w.o.CheckoutRoot = rw.top
	res := run(t, w)
	if len(res.Notes) != 1 || !strings.Contains(res.Notes[0], "release") || len(res.Findings) != 0 {
		t.Fatalf("notes %q findings %q", res.Notes, res.Findings)
	}
}

func TestRustBuildsReadsTheFileLikeGodotDoes(t *testing.T) {
	w := newRustWorld(t, "; entry_symbol = \"nope\"\n[configuration]\n  entry_symbol   =   \"gdext_rust_init\"  \n"+
		"[libraries]\n; linux.debug = \"res://../rust/target/dist/x.so\"\n"+
		"linux.debug = \"res://../rust/target/debug/libgame.so\"\n"+
		"[other]\nlinux.debug = \"res://../rust/target/release/libgame.so\"\n")
	builds, _ := RustBuilds(w.project, w.top)
	if want := []RustBuild{{Crate: w.crate, Profile: "debug"}}; !reflect.DeepEqual(builds, want) {
		t.Errorf("builds %v want %v", builds, want)
	}
}

func TestRustBuildsFindsAnExtensionBelowAnyDirectory(t *testing.T) {
	w := newRustWorld(t, "")
	os.Remove(filepath.Join(w.project, "game.gdextension"))
	write(t, filepath.Join(w.project, "addons/game/game.gdextension"), rustHeader+
		"linux.debug = \"res://../rust/target/debug/libgame.so\"\n")
	// What the import cache and a Rust target directory hold is nobody's extension.
	write(t, filepath.Join(w.project, ".godot/x.gdextension"), rustHeader+"linux.release = \"res://../rust/target/release/x.so\"\n")
	write(t, filepath.Join(w.project, "target/x.gdextension"), rustHeader+"linux.dist = \"res://../rust/target/dist/x.so\"\n")
	builds, _ := RustBuilds(w.project, w.top)
	if want := []RustBuild{{Crate: w.crate, Profile: "debug"}}; !reflect.DeepEqual(builds, want) {
		t.Errorf("builds %v want %v", builds, want)
	}
}

func TestRustBuildsIgnoreCpp(t *testing.T) {
	w := newRustWorld(t, "[configuration]\nentry_symbol = \"example_library_init\"\n[libraries]\n"+
		"linux.debug.x86_64 = \"res://../rust/target/debug/libgame.so\"\n")
	write(t, filepath.Join(w.project, "nosymbol.gdextension"), "[libraries]\nlinux.debug = \"res://../rust/target/debug/x.so\"\n")
	builds, notes := RustBuilds(w.project, w.top)
	if len(builds) != 0 || len(notes) != 0 {
		t.Errorf("builds %v notes %q", builds, notes)
	}
}

func TestRustBuildsNoCargoToml(t *testing.T) {
	w := newRustWorld(t, rustHeader+"linux.debug = \"res://../rust/target/debug/libgame.so\"\n")
	os.Remove(filepath.Join(w.crate, "Cargo.toml"))
	builds, notes := RustBuilds(w.project, w.top)
	want := "no Cargo.toml above " + filepath.Join(w.crate, "target")
	if len(builds) != 0 || len(notes) != 1 || notes[0] != want {
		t.Errorf("builds %v notes %q want %q", builds, notes, want)
	}
}

func TestRustBuildsTakesTheNearestCargoTomlAboveTarget(t *testing.T) {
	// A workspace: target/ sits beside the root manifest, the member is one
	// level down, and cargo builds the workspace from its root.
	w := newRustWorld(t, rustHeader+"linux.debug = \"res://../target/debug/libgame.so\"\n")
	os.Remove(filepath.Join(w.crate, "Cargo.toml"))
	write(t, filepath.Join(w.top, "Cargo.toml"), "[workspace]\nmembers = [\"rust\"]\n")
	builds, _ := RustBuilds(w.project, w.top)
	if want := []RustBuild{{Crate: w.top, Profile: "debug"}}; !reflect.DeepEqual(builds, want) {
		t.Errorf("builds %v want %v", builds, want)
	}
}

func TestRustBuildsNamesAnUnreadableExtension(t *testing.T) {
	w := newRustWorld(t, rustHeader)
	old := readFile
	readFile = func(string) ([]byte, error) { return nil, os.ErrPermission }
	t.Cleanup(func() { readFile = old })
	builds, notes := RustBuilds(w.project, w.top)
	if len(builds) != 0 || len(notes) != 1 || !strings.Contains(notes[0], "game.gdextension") || !strings.Contains(notes[0], "permission") {
		t.Errorf("builds %v notes %q", builds, notes)
	}
}

func TestRustBuildsOfAProjectThatIsNotThere(t *testing.T) {
	builds, notes := RustBuilds(filepath.Join(t.TempDir(), "absent"), t.TempDir())
	if len(builds) != 0 || len(notes) != 1 {
		t.Errorf("builds %v notes %q", builds, notes)
	}
}

func TestBuildRustArgv(t *testing.T) {
	for profile, want := range map[string][]string{
		"debug":   {"cargo", "build"},
		"release": {"cargo", "build", "--release"},
		"dist":    {"cargo", "build", "--profile", "dist"},
	} {
		var got child.Spec
		err := BuildRust(func(s child.Spec) child.Result { got = s; return child.Result{} },
			RustBuild{Crate: "/c", Profile: profile})
		if err != nil || !reflect.DeepEqual(got.Argv, want) || got.Dir != "/c" {
			t.Errorf("%s: err %v spec %+v", profile, err, got)
		}
	}
}

func TestMissingCargo(t *testing.T) {
	run := func(child.Spec) child.Result {
		return child.Result{Code: -1, Err: errors.Join(errors.New("cargo"), &exec.Error{Name: "cargo", Err: exec.ErrNotFound})}
	}
	err := BuildRust(run, RustBuild{Crate: "/c", Profile: "debug"})
	if err == nil || !strings.Contains(err.Error(), "cargo is not on PATH") {
		t.Fatalf("err %v", err)
	}
}

func TestBuildRustOtherStartErrors(t *testing.T) {
	err := BuildRust(func(child.Spec) child.Result {
		return child.Result{Code: -1, Err: errors.New("access denied")}
	}, RustBuild{Crate: "/c", Profile: "debug"})
	if err == nil || !strings.Contains(err.Error(), "access denied") || strings.Contains(err.Error(), "not on PATH") {
		t.Fatalf("err %v", err)
	}
}

func TestBuildFailureCarriesCargosOutput(t *testing.T) {
	err := BuildRust(func(child.Spec) child.Result {
		return child.Result{Code: 101, Stdout: "out line\n", Stderr: "error[E0425]: cannot find value\n"}
	}, RustBuild{Crate: "/c", Profile: "release"})
	if err == nil || !strings.Contains(err.Error(), "cargo build --release failed (exit 101)") ||
		!strings.Contains(err.Error(), "error[E0425]: cannot find value") {
		t.Fatalf("err %v", err)
	}
}

func TestBuildRustTimeout(t *testing.T) {
	err := BuildRust(func(child.Spec) child.Result {
		return child.Result{Code: -1, TimedOut: true, Stderr: "Compiling godot-core\n"}
	}, RustBuild{Crate: "/c", Profile: "debug"})
	if err == nil || !strings.Contains(err.Error(), "timed out") || !strings.Contains(err.Error(), "Compiling godot-core") {
		t.Fatalf("err %v", err)
	}
}

// The repository bound: cargo is never started in a directory above the checkout.
func TestRustBuildsNeverSearchAboveTheRepository(t *testing.T) {
	outer, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(outer, "repo")
	write(t, filepath.Join(outer, "Cargo.toml"), "[workspace]\n")
	write(t, filepath.Join(repo, ".git/HEAD"), "ref: refs/heads/main\n")
	project := filepath.Join(repo, "godot")
	write(t, filepath.Join(project, "project.godot"), plain)
	write(t, filepath.Join(project, "r.gdextension"), rustHeader+"linux.debug = \"res://../rust/target/debug/libx.so\"\n")
	builds, notes := RustBuilds(project, repo)
	if len(builds) != 0 || len(notes) != 1 || notes[0] != "no Cargo.toml above "+filepath.Join(repo, "rust", "target") {
		t.Errorf("builds %v notes %q", builds, notes)
	}
}

func TestRustBuildsRejectATargetOutsideTheRepository(t *testing.T) {
	w := newRustWorld(t, rustHeader+"linux.debug = \"res://../../elsewhere/target/debug/libx.so\"\n")
	write(t, filepath.Join(filepath.Dir(w.top), "elsewhere", "Cargo.toml"), "[package]\n")
	builds, notes := RustBuilds(w.project, w.top)
	want := filepath.Join(filepath.Dir(w.top), "elsewhere", "target") + " is outside the repository " + w.top + ": not built"
	if len(builds) != 0 || len(notes) != 1 || notes[0] != want {
		t.Errorf("builds %v notes %q want %q", builds, notes, want)
	}
}

func TestRustBuildsFindsTheManifestAtTheRepositoryRoot(t *testing.T) {
	w := newRustWorld(t, rustHeader+"linux.debug = \"res://../target/debug/libgame.so\"\n")
	os.Remove(filepath.Join(w.crate, "Cargo.toml"))
	write(t, filepath.Join(w.top, "Cargo.toml"), "[workspace]\n")
	builds, _ := RustBuilds(w.project, w.top)
	if want := []RustBuild{{Crate: w.top, Profile: "debug"}}; !reflect.DeepEqual(builds, want) {
		t.Errorf("builds %v want %v", builds, want)
	}
}

func TestRustBuildsSayWhenCargoTargetDirIsSet(t *testing.T) {
	w := newRustWorld(t, rustHeader+"linux.debug = \"res://../rust/target/debug/libgame.so\"\n")
	t.Setenv("CARGO_TARGET_DIR", "/fast/target")
	builds, notes := RustBuilds(w.project, w.top)
	if len(builds) != 1 || len(notes) != 1 || !strings.Contains(notes[0], "CARGO_TARGET_DIR is set (/fast/target)") {
		t.Errorf("builds %v notes %q", builds, notes)
	}
	t.Setenv("CARGO_TARGET_DIR", "")
	if _, notes = RustBuilds(w.project, w.top); len(notes) != 0 {
		t.Errorf("notes %q", notes)
	}
}

func TestRustBuildsNoCargoTargetDirNoteWithoutABuild(t *testing.T) {
	w := newRustWorld(t, "[libraries]\n")
	t.Setenv("CARGO_TARGET_DIR", "/fast/target")
	if builds, notes := RustBuilds(w.project, w.top); len(builds) != 0 || len(notes) != 0 {
		t.Errorf("builds %v notes %q", builds, notes)
	}
}

// Godot's config format: a byte order mark and trailing `;` comments are
// legal, a `;` inside a quoted value is not a comment.
func TestRustBuildsReadsBOMAndTrailingComments(t *testing.T) {
	w := newRustWorld(t, "\xef\xbb\xbf[configuration] ; the entry\nentry_symbol = \"gdext_rust_init\" ; rust\n"+
		"[libraries] ; paths\nlinux.debug = \"res://../rust/target/debug/lib;game.so\" ; debug\n")
	builds, notes := RustBuilds(w.project, w.top)
	if want := []RustBuild{{Crate: w.crate, Profile: "debug"}}; !reflect.DeepEqual(builds, want) || len(notes) != 0 {
		t.Errorf("builds %v notes %q want %v", builds, notes, want)
	}
}

func TestRustBuildsSkipsTemplateReleaseKeys(t *testing.T) {
	w := newRustWorld(t, rustHeader+
		"linux.template_release.x86_64 = \"res://../rust/target/release/libgame.so\"\n"+
		"linux.template_debug.x86_64 = \"res://../rust/target/debug/libgame.so\"\n")
	builds, notes := RustBuilds(w.project, w.top)
	if want := []RustBuild{{Crate: w.crate, Profile: "debug"}}; !reflect.DeepEqual(builds, want) || len(notes) != 0 {
		t.Errorf("builds %v notes %q", builds, notes)
	}
}

// buildWorld is a godot-suite project with a Rust extension whose runner
// logs the order of cargo and engine starts.
func buildWorld(t *testing.T) (*world, rustWorld, *[]string) {
	t.Helper()
	rw := newRustWorld(t, rustHeader+"linux.debug = \"res://../rust/target/debug/libgame.so\"\n")
	w := newWorldAt(t, rw.project, plain)
	w.o.CheckoutRoot = rw.top
	var order []string
	inner := w.e.start
	w.o.Start = func(s child.Spec) child.Result {
		if s.Argv[0] == "cargo" {
			order = append(order, "cargo")
			return child.Result{}
		}
		order = append(order, "godot")
		return inner(s)
	}
	return w, rw, &order
}

func TestSuiteBuildsBeforeGodot(t *testing.T) {
	w, _, order := buildWorld(t)
	res := run(t, w)
	if len(res.Findings) != 0 {
		t.Fatalf("findings %q", res.Findings)
	}
	if len(*order) < 2 || (*order)[0] != "cargo" || (*order)[1] != "godot" {
		t.Fatalf("order %q", *order)
	}
	n := 0
	for _, s := range *order {
		if s == "cargo" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d builds", n)
	}
}

func TestSuiteBuildsBeforeTheDiskInstrumentation(t *testing.T) {
	w, _, order := buildWorld(t)
	w.o.Coverage = filepath.Join(w.checkout, "cov")
	write(t, filepath.Join(w.root, "project.godot"), withAutoload)
	run(t, w)
	if len(*order) < 2 || (*order)[0] != "cargo" {
		t.Fatalf("order %q", *order)
	}
}

func TestSuiteBuildsNothingWhenNoSuiteRuns(t *testing.T) {
	w, _, order := buildWorld(t)
	os.Remove(filepath.Join(w.root, "test/a_test.gd"))
	os.Remove(filepath.Join(w.root, "test/b_test.gd"))
	run(t, w)
	if len(*order) != 0 {
		t.Fatalf("order %q", *order)
	}
}

func TestNoExtensionNoBuild(t *testing.T) {
	w := newWorld(t, plain)
	var argv0 []string
	inner := w.e.start
	w.o.Start = func(s child.Spec) child.Result { argv0 = append(argv0, s.Argv[0]); return inner(s) }
	res := run(t, w)
	if len(res.Findings) != 0 || len(res.Notes) != 0 || slices.Contains(argv0, "cargo") {
		t.Fatalf("findings %q notes %q starts %q", res.Findings, res.Notes, argv0)
	}
}

func TestSuiteMissingCargoIsAFindingBeforeAnyEngine(t *testing.T) {
	w, _, order := buildWorld(t)
	inner := w.o.Start
	w.o.Start = func(s child.Spec) child.Result {
		if s.Argv[0] == "cargo" {
			return child.Result{Code: -1, Err: &exec.Error{Name: "cargo", Err: exec.ErrNotFound}}
		}
		return inner(s)
	}
	res := run(t, w)
	if len(res.Findings) != 1 || !strings.Contains(res.Findings[0], "cargo is not on PATH") || slices.Contains(*order, "godot") {
		t.Fatalf("findings %q order %q", res.Findings, *order)
	}
}

func TestBuildFailureFailsLane(t *testing.T) {
	w, _, order := buildWorld(t)
	inner := w.o.Start
	w.o.Start = func(s child.Spec) child.Result {
		if s.Argv[0] == "cargo" {
			return child.Result{Code: 101, Stderr: "error: could not compile `game`\n"}
		}
		return inner(s)
	}
	res := run(t, w)
	if len(res.Findings) != 1 || !strings.Contains(res.Findings[0], "could not compile `game`") || slices.Contains(*order, "godot") {
		t.Fatalf("findings %q order %q", res.Findings, *order)
	}
}

func TestSuiteCarriesTheBuildNotes(t *testing.T) {
	w, rw, _ := buildWorld(t)
	os.Remove(filepath.Join(rw.crate, "Cargo.toml"))
	res := run(t, w)
	if want := "no Cargo.toml above " + filepath.Join(rw.crate, "target"); len(res.Notes) != 1 || res.Notes[0] != want {
		t.Fatalf("notes %q", res.Notes)
	}
	if len(res.Findings) != 0 {
		t.Fatalf("findings %q", res.Findings)
	}
}

func TestRescanBuildsBeforeTheImport(t *testing.T) {
	rw := newRustWorld(t, rustHeader+"linux.debug = \"res://../rust/target/debug/libgame.so\"\n")
	w := &rescanWorld{root: rw.project, cache: true, rewrit: "config_version=5\n"}
	var order []string
	notes, err := Rescan(RescanOptions{Root: rw.project, Godot: "godot", Status: func(string) (string, error) { return "", nil },
		Start: func(s child.Spec) child.Result {
			order = append(order, s.Argv[0])
			if s.Argv[0] == "cargo" {
				return child.Result{}
			}
			return w.start(s)
		}})
	if err != nil || len(notes) != 0 {
		t.Fatalf("err %v notes %q", err, notes)
	}
	if !reflect.DeepEqual(order, []string{"cargo", "godot"}) {
		t.Fatalf("order %q", order)
	}
}

func TestRescanStopsOnABuildFailure(t *testing.T) {
	rw := newRustWorld(t, rustHeader+"linux.debug = \"res://../rust/target/debug/libgame.so\"\n")
	var order []string
	_, err := Rescan(RescanOptions{Root: rw.project, Godot: "godot", Status: func(string) (string, error) { return "", nil },
		Start: func(s child.Spec) child.Result {
			order = append(order, s.Argv[0])
			return child.Result{Code: 101, Stderr: "error: boom\n"}
		}})
	if err == nil || !strings.Contains(err.Error(), "error: boom") || !reflect.DeepEqual(order, []string{"cargo"}) {
		t.Fatalf("err %v order %q", err, order)
	}
	if b, _ := os.ReadFile(filepath.Join(rw.project, "project.godot")); string(b) != plain {
		t.Fatalf("project.godot %q", b)
	}
}

func TestRescanCarriesTheBuildNotes(t *testing.T) {
	rw := newRustWorld(t, rustHeader+"linux.debug = \"res://../rust/target/debug/libgame.so\"\n")
	os.Remove(filepath.Join(rw.crate, "Cargo.toml"))
	w := &rescanWorld{root: rw.project, cache: true, rewrit: "config_version=5\n"}
	notes, err := Rescan(RescanOptions{Root: rw.project, Godot: "godot", Status: func(string) (string, error) { return "", nil }, Start: w.start})
	if want := "no Cargo.toml above " + filepath.Join(rw.crate, "target"); err != nil || len(notes) != 1 || notes[0] != want {
		t.Fatalf("err %v notes %q", err, notes)
	}
}
