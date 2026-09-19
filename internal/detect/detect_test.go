package detect

import (
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestUvManagedPythonIsDetected(t *testing.T) {
	tree := fstest.MapFS{
		"pyproject.toml": {Data: []byte("[project]\nname = \"x\"\n")},
		"uv.lock":        {Data: []byte("")},
	}
	facts := Detect(tree)
	if !has(facts.Stacks, "python") {
		t.Fatalf("stacks = %v, want python", facts.Stacks)
	}
	if !has(facts.Stacks, "uv") {
		t.Fatalf("stacks = %v, want uv", facts.Stacks)
	}
}

func TestGodotKeepsGdscriptStack(t *testing.T) {
	tree := fstest.MapFS{
		"project.godot": {Data: []byte("config_version=5\n\nproject/assembly_name=\"space\"\n")},
	}
	facts := Detect(tree)
	for _, want := range []string{"godot", "gdscript"} {
		if !has(facts.Stacks, want) {
			t.Fatalf("stacks = %v, want %s", facts.Stacks, want)
		}
	}
	if has(facts.Stacks, "csharp") {
		t.Fatalf("stacks = %v, want no csharp", facts.Stacks)
	}
}

func TestAnEmptyTreeDetectsNothingAndSaysSo(t *testing.T) {
	facts := Detect(fstest.MapFS{})
	if len(facts.Stacks) != 0 {
		t.Fatalf("stacks = %v, want none", facts.Stacks)
	}
}

func TestCPPSignalsAreDetected(t *testing.T) {
	tree := fstest.MapFS{
		"CMakeLists.txt": {Data: []byte("cmake_minimum_required(VERSION 3.20)\n")},
		".clang-format":  {Data: []byte("BasedOnStyle: LLVM\n")},
		".clang-tidy":    {Data: []byte("Checks: '*'\n")},
	}
	facts := Detect(tree)
	for _, want := range []string{"cpp", "cmake", "clang-format", "clang-tidy"} {
		if !has(facts.Stacks, want) {
			t.Errorf("stacks = %v, want %s", facts.Stacks, want)
		}
	}
	if has(facts.Stacks, "csharp") {
		t.Fatalf("stacks = %v, want no csharp", facts.Stacks)
	}
}

func TestCPPGlobSignalsAreDetected(t *testing.T) {
	tree := fstest.MapFS{
		"main.cpp":   {Data: []byte("int main() { return 0; }\n")},
		"header.hpp": {Data: []byte("#pragma once\n")},
	}
	facts := Detect(tree)
	if !has(facts.Stacks, "cpp") {
		t.Fatalf("stacks = %v, want cpp from *.cpp and *.hpp globs", facts.Stacks)
	}
}

func TestAWorkspaceMemberIsSeenOneLevelDown(t *testing.T) {
	tree := fstest.MapFS{
		"README.md":               {Data: []byte("workspace\n")},
		"services/api/Cargo.toml": {Data: []byte("[package]\n")},
		"backend/pyproject.toml":  {Data: []byte("[project]\n")},
		"frontend/tsconfig.json":  {Data: []byte("{}")},
		"frontend/package.json":   {Data: []byte("{}")},
		"game/CMakeLists.txt":     {Data: []byte("cmake_minimum_required(VERSION 3.20)\n")},
	}
	facts := Detect(tree)
	for _, want := range []string{"python", "typescript", "cpp", "cmake"} {
		if !has(facts.Stacks, want) {
			t.Fatalf("stacks = %v, want %s", facts.Stacks, want)
		}
	}
	if has(facts.Stacks, "csharp") {
		t.Fatalf("stacks = %v, want no csharp", facts.Stacks)
	}
	// Two levels down is a member's own layout, not a member.
	if has(facts.Stacks, "rust") {
		t.Fatalf("stacks = %v, want no rust from two levels down", facts.Stacks)
	}
}

// Dot directories carry tooling, not workspace members -- `.godot/` alone
// holds an import cache wide enough to make several rows fire on it.
func TestDotDirectoriesAreNotWorkspaceMembers(t *testing.T) {
	tree := fstest.MapFS{
		".cache/Cargo.toml": {Data: []byte("[package]\n")},
	}
	facts := Detect(tree)
	if len(facts.Stacks) != 0 {
		t.Fatalf("stacks = %v, want none", facts.Stacks)
	}
}

// TypeScript is a conjunction in the spec, and each half alone says something
// else: a lone tsconfig.json is a configuration file, a lone package.json is
// the question below.
func TestTypescriptNeedsBothHalves(t *testing.T) {
	lone := Detect(fstest.MapFS{"tsconfig.json": {Data: []byte("{}")}})
	if has(lone.Stacks, "typescript") {
		t.Fatalf("stacks = %v, want no typescript without package.json", lone.Stacks)
	}
	both := Detect(fstest.MapFS{
		"tsconfig.json": {Data: []byte("{}")},
		"package.json":  {Data: []byte("{}")},
	})
	if !has(both.Stacks, "typescript") {
		t.Fatalf("stacks = %v, want typescript", both.Stacks)
	}
	if len(both.Ambiguous) != 0 {
		t.Fatalf("ambiguous = %v, want none once tsconfig.json answers it", both.Ambiguous)
	}
}

// A lone package.json is not a stack but a question: it is as often tooling
// for another language as it is a project of its own.
func TestALonePackageJsonIsAskedAboutRatherThanDecided(t *testing.T) {
	facts := Detect(fstest.MapFS{"package.json": {Data: []byte("{}")}})
	if len(facts.Stacks) != 0 {
		t.Fatalf("stacks = %v, want none", facts.Stacks)
	}
	if len(facts.Ambiguous) != 1 || !strings.Contains(facts.Ambiguous[0], "package.json") {
		t.Fatalf("ambiguous = %v, want the package.json question", facts.Ambiguous)
	}
}

// Django is detected, and what it means for migrations is asked.
func TestDjangoAsksAboutMigrations(t *testing.T) {
	facts := Detect(fstest.MapFS{"manage.py": {Data: []byte("import django\n")}})
	if !has(facts.Stacks, "django") {
		t.Fatalf("stacks = %v, want django", facts.Stacks)
	}
	if len(facts.Ambiguous) != 1 || !strings.Contains(facts.Ambiguous[0], "migrations") {
		t.Fatalf("ambiguous = %v, want the migrations question", facts.Ambiguous)
	}
}

func TestMesonIsDetectedAsCPP(t *testing.T) {
	facts := Detect(fstest.MapFS{
		"meson.build": {Data: []byte("project('myproj', 'cpp')\n")},
	})
	for _, want := range []string{"cpp", "meson"} {
		if !has(facts.Stacks, want) {
			t.Fatalf("stacks = %v, want %s", facts.Stacks, want)
		}
	}
	if has(facts.Stacks, "csharp") {
		t.Fatalf("stacks = %v, want no csharp", facts.Stacks)
	}
}

func TestGitAndWikiAreFacts(t *testing.T) {
	tree := fstest.MapFS{
		".git/HEAD":     {Data: []byte("ref: refs/heads/master\n")},
		"wiki/index.md": {Data: []byte("---\nokf_version: \"0.2\"\n---\n# Concepts\n")},
	}
	facts := Detect(tree)
	if !facts.HasGit {
		t.Fatal("HasGit = false, want true")
	}
	if facts.WikiMode != "brain" || facts.WikiPath != "wiki/" {
		t.Fatalf("wiki = %q %q, want \"brain\" \"wiki/\"", facts.WikiMode, facts.WikiPath)
	}
}

func TestATreeWithoutGitOrWikiSaysNeither(t *testing.T) {
	facts := Detect(fstest.MapFS{"README.md": {Data: []byte("nothing here\n")}})
	if facts.HasGit {
		t.Fatal("HasGit = true, want false")
	}
	if facts.WikiMode != "" || facts.WikiPath != "" {
		t.Fatalf("wiki = %q %q, want empty", facts.WikiMode, facts.WikiPath)
	}
}

// A docs folder called wiki is the false alarm that must not become brain
// mode: the marker decides, and without it the interview does.
func TestAPlainWikiFolderIsAskedAboutRatherThanAsserted(t *testing.T) {
	facts := Detect(fstest.MapFS{"wiki/notes.md": {Data: []byte("# notes\n")}})
	if facts.WikiMode != "" || facts.WikiPath != "" {
		t.Fatalf("wiki = %q %q, want empty", facts.WikiMode, facts.WikiPath)
	}
	if len(facts.Ambiguous) != 1 || !strings.Contains(facts.Ambiguous[0], "wiki/") {
		t.Fatalf("ambiguous = %v, want the wiki question", facts.Ambiguous)
	}
}

// An index without the marker is the same case as no index at all.
func TestAWikiIndexWithoutTheMarkerIsNotABundle(t *testing.T) {
	facts := Detect(fstest.MapFS{"wiki/index.md": {Data: []byte("# just an index\n")}})
	if facts.WikiMode != "" {
		t.Fatalf("wiki mode = %q, want empty", facts.WikiMode)
	}
	if len(facts.Ambiguous) != 1 {
		t.Fatalf("ambiguous = %v, want the wiki question", facts.Ambiguous)
	}
}

func TestLoomuxConfigWithWikiIsDetectedAsBrainMode(t *testing.T) {
	facts := Detect(fstest.MapFS{
		".loomux/config.toml": {Data: []byte("[area]\nscope = \"my-scope\"\nwiki = true\n")},
		"docs/wiki/index.md":  {Data: []byte("# docs wiki\n")},
	})
	if facts.WikiMode != "brain" || facts.WikiPath != "docs/wiki/" {
		t.Fatalf("expected brain mode and docs/wiki/ path, got %q %q", facts.WikiMode, facts.WikiPath)
	}
	hasWikiStack := false
	for _, s := range facts.Stacks {
		if s == "wiki" {
			hasWikiStack = true
		}
	}
	if !hasWikiStack {
		t.Fatalf("expected wiki stack in %v", facts.Stacks)
	}
}

// An unreadable tree is a fact, not an error: what cannot be read carries no
// signal, and detection reports what is left of the tree.
func TestAnUnlistableTreeYieldsNoStacks(t *testing.T) {
	facts := Detect(closedFS{})
	if len(facts.Stacks) != 0 {
		t.Fatalf("stacks = %v, want none", facts.Stacks)
	}
	if facts.HasGit {
		t.Fatal("HasGit = true, want false")
	}
}

// The narrower failure: the names are there, the contents are not. A row that
// needs to look inside must then decline rather than guess.
func TestAnUnreadableFileDoesNotSatisfyContains(t *testing.T) {
	facts := Detect(listableFS{fstest.MapFS{
		"project.godot": {Data: []byte("config_version=5\n\n[dotnet]\n")},
	}})
	if !has(facts.Stacks, "gdscript") {
		t.Fatalf("stacks = %v, want gdscript", facts.Stacks)
	}
	if has(facts.Stacks, "csharp") {
		t.Fatalf("stacks = %v, want no csharp", facts.Stacks)
	}
}

// closedFS refuses every access, which is what an unreadable root looks like
// from here: no listing, no stat, no content.
type closedFS struct{}

func (closedFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
}

// listableFS answers about names and refuses their contents. Stat and ReadDir
// are delegated; Open, which is all that is left for reading a file, is not.
type listableFS struct {
	tree fstest.MapFS
}

func (f listableFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
}

func (f listableFS) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(f.tree, name)
}

func (f listableFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(f.tree, name)
}

func TestEcosystemToolingDetection(t *testing.T) {
	tree := fstest.MapFS{
		"biome.json":          {Data: []byte("{}")},
		"pnpm-workspace.yaml": {Data: []byte("packages:\n  - 'apps/*'")},
		".gdlintrc":           {Data: []byte("")},
		"docker-compose.yml":  {Data: []byte("version: '3'")},
	}
	facts := Detect(tree)
	for _, want := range []string{"biome", "typescript", "pnpm", "gdlint", "gdscript", "docker"} {
		if !has(facts.Stacks, want) {
			t.Errorf("stacks = %v, want %s", facts.Stacks, want)
		}
	}
}

func TestDetectWikiBundleAndContainsVariants(t *testing.T) {
	// 1. .loomux/config.toml with [wiki] header
	tree1 := fstest.MapFS{
		".loomux/config.toml": {Data: []byte("[area]\nname=\"test\"\n[wiki]\n")},
	}
	facts1 := Detect(tree1)
	if facts1.WikiMode != "brain" {
		t.Fatalf("expected WikiMode=brain from [wiki] header, got %s", facts1.WikiMode)
	}

	// 2. bundle.toml with okf marker
	tree2 := fstest.MapFS{
		"wiki/bundle.toml": {Data: []byte("format = \"" + okfMarker + "\"\n")},
	}
	facts2 := Detect(tree2)
	if facts2.WikiMode != "brain" || facts2.WikiPath != "wiki/" {
		t.Fatalf("expected WikiMode=brain from bundle.toml, got mode=%s path=%s", facts2.WikiMode, facts2.WikiPath)
	}

	// 3. pyproject.toml without [tool.pyright]
	tree3 := fstest.MapFS{
		"pyproject.toml": {Data: []byte("[project]\nname=\"x\"\n")},
	}
	facts3 := Detect(tree3)
	if has(facts3.Stacks, "pyright") {
		t.Fatalf("did not expect pyright without contains match, got %v", facts3.Stacks)
	}

	// 4. tsconfig.json without package.json (besides check)
	tree4 := fstest.MapFS{
		"tsconfig.json": {Data: []byte("{}")},
	}
	facts4 := Detect(tree4)
	if has(facts4.Stacks, "typescript") {
		t.Fatalf("did not expect typescript without package.json, got %v", facts4.Stacks)
	}

	// 5. Unreadable file with sig.contains
	facts5 := Detect(listableFS{tree: tree3})
	if has(facts5.Stacks, "pyright") {
		t.Fatalf("did not expect pyright with unreadable pyproject.toml")
	}
}

func has(all []string, one string) bool {
	for _, candidate := range all {
		if candidate == one {
			return true
		}
	}
	return false
}

// Where the Godot project stands is a fact and not a detail: gdlint looks its
// configuration up from the working directory upwards, so a run started at the
// repository root of a project that keeps its tree under godot/ finds no
// .gdlintrc at all and lints on defaults.
func TestGodotBelowTheRootIsNamed(t *testing.T) {
	tree := fstest.MapFS{
		"godot/project.godot": {Data: []byte("config_version=5\n")},
		"godot/.gdlintrc":     {Data: []byte("max-line-length: 100\n")},
	}

	facts := Detect(tree)

	if facts.GodotDir != "godot" {
		t.Fatalf("GodotDir = %q, want %q", facts.GodotDir, "godot")
	}
}

// A tree with the project at its root says so with an empty string rather than
// with ".": the caller joins this onto a root path, and joining "." there adds
// a segment that means nothing.
func TestGodotAtTheRootNamesNoDirectory(t *testing.T) {
	tree := fstest.MapFS{"project.godot": {Data: []byte("config_version=5\n")}}

	facts := Detect(tree)

	if facts.GodotDir != "" {
		t.Fatalf("GodotDir = %q, want empty", facts.GodotDir)
	}
}

// A .loomux/config.toml without a wiki decides, and an old .brain.toml beside
// it is never read -- or the manifest and an old leftover would answer the
// same repository differently.
func TestLoomuxConfigWinsOverBrainToml(t *testing.T) {
	facts := Detect(fstest.MapFS{
		".loomux/config.toml": {Data: []byte("[area]\nscope = \"project/x\"\n")},
		".brain.toml":         {Data: []byte("[area]\nwiki = true\n")},
		"docs/wiki/index.md":  {Data: []byte("# Katalog\n")},
	})
	if facts.WikiMode != "" {
		t.Fatalf("mode = %q: .brain.toml was read although .loomux/config.toml exists", facts.WikiMode)
	}
}

// A row with contains counts once the file carries the text: pyright is only
// named where pyproject.toml configures it.
func TestPyprojectWithPyrightSectionNamesPyright(t *testing.T) {
	facts := Detect(fstest.MapFS{
		"pyproject.toml": {Data: []byte("[project]\nname=\"x\"\n\n[tool.pyright]\n")},
	})
	if !has(facts.Stacks, "pyright") {
		t.Fatalf("stacks = %v, want pyright", facts.Stacks)
	}
}

func TestDetectRecordsEveryAreaAStackIsFoundIn(t *testing.T) {
	root := fstest.MapFS{
		"go.mod":              {Data: []byte("module x\n")},
		"web/package.json":    {Data: []byte("{}")},
		"web/tsconfig.json":   {Data: []byte("{}")},
		"admin/package.json":  {Data: []byte("{}")},
		"admin/tsconfig.json": {Data: []byte("{}")},
		"game/project.godot":  {Data: []byte("")},
	}
	f := Detect(root)
	want := map[string][]string{
		"go":         {"."},
		"typescript": {"admin", "web"},
		"godot":      {"game"},
		"gdscript":   {"game"},
	}
	for stack, areas := range want {
		if !reflect.DeepEqual(f.Areas[stack], areas) {
			t.Errorf("%s: %v, want %v", stack, f.Areas[stack], areas)
		}
	}
	if f.GodotDir != "game" {
		t.Errorf("GodotDir %q", f.GodotDir)
	}
}
