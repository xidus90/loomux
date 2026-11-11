package detect

import (
	"io/fs"
	"reflect"
	"slices"
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

func TestCSharpSignalsAreDetected(t *testing.T) {
	for _, name := range []string{"x.csproj", "x.sln", "x.slnx", "global.json", "Directory.Build.props"} {
		facts := Detect(fstest.MapFS{name: {Data: []byte("<Project />\n")}})
		if !reflect.DeepEqual(facts.Stacks, []string{"csharp"}) {
			t.Errorf("%s: stacks = %v, want csharp", name, facts.Stacks)
		}
	}
}

// global.json choosing Microsoft.Testing.Platform is what switches dotnet
// test out of its VSTest mode; a global.json pinning only the SDK does not.
func TestAGlobalJSONChoosingMTPIsDetected(t *testing.T) {
	mtp := Detect(fstest.MapFS{"global.json": {Data: []byte(`{"test": {"runner": "Microsoft.Testing.Platform"}}`)}})
	if !reflect.DeepEqual(mtp.Stacks, []string{"csharp", "mtp"}) {
		t.Errorf("runner: stacks = %v", mtp.Stacks)
	}
	pin := Detect(fstest.MapFS{"global.json": {Data: []byte(`{"sdk": {"version": "10.0.100"}}`)}})
	if !reflect.DeepEqual(pin.Stacks, []string{"csharp"}) {
		t.Errorf("sdk pin: stacks = %v", pin.Stacks)
	}
}

// A solution in the root builds the projects below it: they are no areas of
// their own, or the root and each member would build the same obj/ and bin/.
// Without one, every project folder is its own area.
func TestARootSolutionOwnsItsProjects(t *testing.T) {
	members := fstest.MapFS{
		"App/App.csproj":             {Data: []byte("<Project />\n")},
		"App.Tests/App.Tests.csproj": {Data: []byte("<Project />\n")},
	}
	if got := Detect(members).Areas["csharp"]; !reflect.DeepEqual(got, []string{"App", "App.Tests"}) {
		t.Errorf("without a solution: %v", got)
	}
	for _, sln := range []string{"App.sln", "App.slnx"} {
		tree := fstest.MapFS{sln: {Data: []byte("\n")}}
		for k, v := range members {
			tree[k] = v
		}
		if got := Detect(tree).Areas["csharp"]; !reflect.DeepEqual(got, []string{"."}) {
			t.Errorf("%s: %v", sln, got)
		}
	}
}

// A root build file builds the sources below it into one build/: the folders
// holding them are no areas of their own.
func TestARootBuildFileOwnsTheCppSources(t *testing.T) {
	members := fstest.MapFS{
		"src/calc.cpp":        {Data: []byte("int f() { return 1; }\n")},
		"tests/calc_test.cpp": {Data: []byte("int g() { return 1; }\n")},
	}
	if got := Detect(members).Areas["cpp"]; !reflect.DeepEqual(got, []string{"src", "tests"}) {
		t.Errorf("without a build file: %v", got)
	}
	for _, file := range []string{"CMakeLists.txt", "CMakePresets.json", "meson.build"} {
		tree := fstest.MapFS{file: {Data: []byte("\n")}}
		for k, v := range members {
			tree[k] = v
		}
		if got := Detect(tree).Areas["cpp"]; !reflect.DeepEqual(got, []string{"."}) {
			t.Errorf("%s: %v", file, got)
		}
	}
}

// Maven and Gradle each name their variant beside the stack; a Gradle build
// in Kotlin DSL counts only where a .java file lies below it, so a Kotlin
// project stays unrecognised.
func TestJavaSignalsAreDetected(t *testing.T) {
	java := fstest.MapFile{Data: []byte("class A {}\n")}
	for name, tree := range map[string]fstest.MapFS{
		"pom.xml":             {"pom.xml": {Data: []byte("<project/>\n")}},
		"build.gradle":        {"build.gradle": {Data: []byte("plugins {}\n")}},
		"settings.gradle":     {"settings.gradle": {Data: []byte("\n")}},
		"build.gradle.kts":    {"build.gradle.kts": {Data: []byte("\n")}, "src/main/java/a/A.java": &java},
		"settings.gradle.kts": {"settings.gradle.kts": {Data: []byte("\n")}, "app/src/test/java/A.java": &java},
	} {
		want := []string{"gradle", "java"}
		if name == "pom.xml" {
			want = []string{"java", "maven"}
		}
		if got := Detect(tree).Stacks; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: stacks = %v, want %v", name, got, want)
		}
	}
	kotlin := fstest.MapFS{
		"build.gradle.kts":           {Data: []byte("\n")},
		"settings.gradle.kts":        {Data: []byte("\n")},
		"src/main/kotlin/a/A.kt":     {Data: []byte("class A\n")},
		"build/generated/B.java":     {Data: []byte("class B {}\n")},
		".gradle/cache/C.java":       {Data: []byte("class C {}\n")},
		"lib/build.gradle.kts":       {Data: []byte("\n")},
		"lib/target/classes/D.java":  {Data: []byte("class D {}\n")},
		"node_modules/x/E.java":      {Data: []byte("class E {}\n")},
		"lib/src/main/kotlin/a/L.kt": {Data: []byte("class L\n")},
	}
	if got := Detect(kotlin).Stacks; len(got) != 0 {
		t.Errorf("pure Kotlin: stacks = %v, want none", got)
	}
}

// lockedDirFS refuses to list one directory, as a folder without read
// permission does.
type lockedDirFS struct {
	fstest.MapFS
	locked string
}

func (f lockedDirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == f.locked {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrPermission}
	}
	return f.MapFS.ReadDir(name)
}

// The search for a .java file passes over a directory it cannot list.
func TestTheJavaSearchPassesAnUnreadableDirectory(t *testing.T) {
	tree := fstest.MapFS{"build.gradle.kts": {Data: []byte("\n")}, "a/A.java": {Data: []byte("class A {}\n")}}
	if got := Detect(lockedDirFS{tree, "a"}).Stacks; len(got) != 0 {
		t.Errorf("only behind the lock: %v", got)
	}
	tree["b/B.java"] = &fstest.MapFile{Data: []byte("class B {}\n")}
	if got := Detect(lockedDirFS{tree, "a"}).Stacks; !reflect.DeepEqual(got, []string{"gradle", "java"}) {
		t.Errorf("beside the lock: %v", got)
	}
}

// A root pom.xml or Gradle build builds its modules: they are no areas of
// their own, or two builds would compile the same module side by side.
func TestARootJavaBuildOwnsItsModules(t *testing.T) {
	for _, file := range []string{"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts"} {
		tree := fstest.MapFS{
			file:                       {Data: []byte("\n")},
			"core/pom.xml":             {Data: []byte("<project/>\n")},
			"app/build.gradle":         {Data: []byte("\n")},
			"app/src/main/java/A.java": {Data: []byte("class A {}\n")},
		}
		if got := Detect(tree).Areas["java"]; !reflect.DeepEqual(got, []string{"."}) {
			t.Errorf("%s: %v", file, got)
		}
	}
}

// An Android plugin names the area it is applied in, and a root Gradle
// build takes it in from its modules, where the plugin usually is: through
// its id in a build file, the buildscript's classpath or the version
// catalog the build's alias reads.
func TestAndroidJoinsTheGradleBuildThatOwnsIt(t *testing.T) {
	java := fstest.MapFile{Data: []byte("class A {}\n")}
	for name, tree := range map[string]fstest.MapFS{
		"module": {
			"settings.gradle": {Data: []byte("include ':app'\n")}, "build.gradle": {Data: []byte("plugins {}\n")},
			"app/build.gradle": {Data: []byte("plugins { id 'com.android.application' }\n")}, "app/src/main/java/A.java": &java,
		},
		"classpath": {
			"build.gradle":             {Data: []byte("buildscript { dependencies { classpath 'com.android.tools.build:gradle:8.13.0' } }\n")},
			"app/src/main/java/A.java": &java,
		},
		"catalog": {
			"settings.gradle.kts": {Data: []byte("\n")}, "app/build.gradle.kts": {Data: []byte("plugins { alias(libs.plugins.android.application) }\n")},
			"gradle/libs.versions.toml": {Data: []byte("android-application = { id = \"com.android.application\", version.ref = \"agp\" }\n")},
			"app/src/main/java/A.java":  &java,
		},
	} {
		if got := Detect(tree).Areas["android"]; !slices.Contains(got, ".") {
			t.Errorf("%s: android areas %v", name, got)
		}
	}
	// Without a root build each module is its own area, and the plugin stays
	// in the one that applies it.
	tree := fstest.MapFS{
		"app/build.gradle": {Data: []byte("plugins { id 'com.android.application' }\n")}, "app/A.java": &java,
		"lib/build.gradle": {Data: []byte("plugins { id 'java' }\n")}, "lib/B.java": &java,
	}
	facts := Detect(tree)
	if !reflect.DeepEqual(facts.Areas["android"], []string{"app"}) || !reflect.DeepEqual(facts.Areas["java"], []string{"app", "lib"}) {
		t.Errorf("modules: %v", facts.Areas)
	}
}

// Composer means PHP even before the first .php file, and so does a PHPUnit
// configuration.
func TestPHPSignalsAreDetected(t *testing.T) {
	for _, name := range []string{"composer.json", "phpunit.xml", "phpunit.xml.dist", "phpunit.dist.xml"} {
		tree := fstest.MapFS{name: {Data: []byte("{}\n")}}
		if got := Detect(tree).Stacks; !reflect.DeepEqual(got, []string{"php"}) {
			t.Errorf("%s: stacks = %v", name, got)
		}
	}
}

// A root composer.json installs one vendor/ for its packages: checked on
// their own, they would run on that vendor/ side by side.
func TestARootComposerJSONOwnsItsPackages(t *testing.T) {
	tree := fstest.MapFS{
		"app/composer.json": {Data: []byte("{}\n")},
		"tools/phpunit.xml": {Data: []byte("<phpunit/>\n")},
	}
	if got := Detect(tree).Areas["php"]; !reflect.DeepEqual(got, []string{"app", "tools"}) {
		t.Errorf("without a root composer.json: %v", got)
	}
	tree["composer.json"] = &fstest.MapFile{Data: []byte("{}\n")}
	if got := Detect(tree).Areas["php"]; !reflect.DeepEqual(got, []string{"."}) {
		t.Errorf("with one: %v", got)
	}
}

// Review Focus 4: with pom.xml and build.gradle side by side both variants
// are detected; the presets list maven first, so Maven checks (verify).
func TestMavenBesideGradleReportsBoth(t *testing.T) {
	tree := fstest.MapFS{"pom.xml": {Data: []byte("<project/>\n")}, "build.gradle": {Data: []byte("\n")}}
	if got := Detect(tree).Stacks; !reflect.DeepEqual(got, []string{"gradle", "java", "maven"}) {
		t.Fatalf("stacks = %v", got)
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

// mypy is named where a file configures it, and only there: its
// configuration may choose the files to check.
func TestAMypyConfigurationNamesMypy(t *testing.T) {
	for name, data := range map[string]string{
		"pyproject.toml": "[project]\nname=\"x\"\n\n[tool.mypy]\nfiles = [\"src\"]\n",
		"mypy.ini":       "[mypy]\n",
		".mypy.ini":      "[mypy]\n",
		"setup.cfg":      "[metadata]\nname = x\n\n[mypy]\nfiles = src\n",
	} {
		if facts := Detect(fstest.MapFS{name: {Data: []byte(data)}}); !has(facts.Stacks, "mypy") {
			t.Errorf("%s: stacks = %v, want mypy", name, facts.Stacks)
		}
	}
	for name, data := range map[string]string{
		"pyproject.toml": "[project]\nname=\"x\"\n",
		"setup.cfg":      "[metadata]\nname = x\n",
	} {
		if facts := Detect(fstest.MapFS{name: {Data: []byte(data)}}); has(facts.Stacks, "mypy") {
			t.Errorf("%s without a mypy section: stacks = %v", name, facts.Stacks)
		}
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
