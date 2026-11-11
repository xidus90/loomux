package detect

// A signal is a file that means something, plus what it means.
//
// Data rather than a chain of ifs: the table is what a reader of the spec
// compares against, and a new stack is a row instead of a branch.
type signal struct {
	// path is matched literally; glob is matched with path.Match. Exactly
	// one of them is set.
	path   string
	glob   string
	stacks []string
	// contains, when set, must appear in the file for the signal to count.
	// That is what separates a Godot project with C# from one without.
	contains string
	// besides, when set, must exist beside the match in the same area. The
	// spec's TypeScript row is a conjunction, not two rows: a tsconfig.json
	// without a package.json is a configuration file, not a project.
	besides string
	// owns, when set and matched in the root, makes the root the stack's
	// only area: a solution or a CMake or Meson project builds what lies
	// below it, and a member checked on its own would build it a second
	// time, or in C++ find no build tree at all.
	owns bool
	// below, when set, is an extension some file under the area must carry:
	// a Gradle build in Kotlin DSL is a Java project only with Java sources.
	below string
	// joins, when set, names a stack whose ownership of the root takes this
	// signal's matches below the root in as well: an Android plugin applied
	// in app/ belongs to the root's Gradle build.
	joins string
}

var signals = []signal{
	{path: "pyproject.toml", stacks: []string{"python"}},
	{path: "pyproject.toml", contains: "[tool.pyright]", stacks: []string{"python", "pyright"}},
	{path: "pyrightconfig.json", stacks: []string{"python", "pyright"}},
	{path: "pyproject.toml", contains: "[tool.mypy]", stacks: []string{"python", "mypy"}},
	{path: "mypy.ini", stacks: []string{"python", "mypy"}},
	{path: ".mypy.ini", stacks: []string{"python", "mypy"}},
	{path: "setup.cfg", contains: "[mypy]", stacks: []string{"python", "mypy"}},
	{path: "uv.lock", stacks: []string{"python", "uv"}},
	{path: "requirements.txt", stacks: []string{"python"}},
	{path: "manage.py", stacks: []string{"python", "django"}},
	{path: "project.godot", stacks: []string{"godot", "gdscript"}},
	{path: "CMakeLists.txt", stacks: []string{"cpp", "cmake"}, owns: true},
	{path: "CMakePresets.json", stacks: []string{"cpp", "cmake"}, owns: true},
	{path: "meson.build", stacks: []string{"cpp", "meson"}, owns: true},
	{glob: "*.cpp", stacks: []string{"cpp"}},
	{glob: "*.hpp", stacks: []string{"cpp"}},
	{path: ".clang-tidy", stacks: []string{"cpp", "clang-tidy"}},
	{path: ".clang-format", stacks: []string{"cpp", "clang-format"}},
	{path: "tsconfig.json", besides: "package.json", stacks: []string{"typescript"}},
	{path: "biome.json", stacks: []string{"biome", "typescript"}},
	{path: "pnpm-workspace.yaml", stacks: []string{"pnpm"}},
	{path: ".gdlintrc", stacks: []string{"gdlint", "gdscript"}},
	{path: "addons/gut/plugin.cfg", stacks: []string{"gut", "gdscript"}},
	{path: "docker-compose.yml", stacks: []string{"docker"}},
	{path: "compose.yaml", stacks: []string{"docker"}},
	{path: "pom.xml", stacks: []string{"java", "maven"}, owns: true},
	{path: "build.gradle", stacks: []string{"java", "gradle"}, owns: true},
	{path: "settings.gradle", stacks: []string{"java", "gradle"}, owns: true},
	{path: "build.gradle.kts", below: ".java", stacks: []string{"java", "gradle"}, owns: true},
	{path: "settings.gradle.kts", below: ".java", stacks: []string{"java", "gradle"}, owns: true},
	// The Android plugin's id, its buildscript classpath, or the version
	// catalog a plugins alias reads it from.
	{path: "build.gradle", contains: "com.android", stacks: []string{"android"}, joins: "gradle"},
	{path: "build.gradle.kts", contains: "com.android", stacks: []string{"android"}, joins: "gradle"},
	{path: "gradle/libs.versions.toml", contains: "com.android", stacks: []string{"android"}, joins: "gradle"},
	// Composer means PHP without a .php file yet. The root's vendor/ serves
	// its packages, which checked on their own would share it side by side.
	{path: "composer.json", stacks: []string{"php"}, owns: true},
	{path: "phpunit.xml", stacks: []string{"php"}},
	{path: "phpunit.xml.dist", stacks: []string{"php"}},
	{path: "phpunit.dist.xml", stacks: []string{"php"}},
	{glob: "*.sln", stacks: []string{"csharp"}, owns: true},
	{glob: "*.slnx", stacks: []string{"csharp"}, owns: true},
	{glob: "*.csproj", stacks: []string{"csharp"}},
	{path: "global.json", stacks: []string{"csharp"}},
	{path: "Directory.Build.props", stacks: []string{"csharp"}},
	{path: "Cargo.toml", stacks: []string{"rust"}},
	{path: "go.mod", stacks: []string{"go"}},
	{path: ".golangci.yml", stacks: []string{"go", "golangci-lint"}},
	{path: ".golangci.yaml", stacks: []string{"go", "golangci-lint"}},
	{path: ".stylelintrc", stacks: []string{"css", "stylelint"}},
	{path: ".stylelintrc.json", stacks: []string{"css", "stylelint"}},
	{path: ".stylelintrc.js", stacks: []string{"css", "stylelint"}},
	{path: "stylelint.config.js", stacks: []string{"css", "stylelint"}},
	{glob: "*.css", stacks: []string{"css"}},
	{glob: "*.scss", stacks: []string{"css"}},
	{path: ".htmlhintrc", stacks: []string{"html", "htmlhint"}},
	{glob: "*.html", stacks: []string{"html"}},
	{path: "vue.config.js", stacks: []string{"vue", "typescript"}},
	{glob: "*.vue", stacks: []string{"vue"}},
	{path: "svelte.config.js", stacks: []string{"svelte", "typescript"}},
	{glob: "*.svelte", stacks: []string{"svelte"}},
	{path: "astro.config.mjs", stacks: []string{"astro", "typescript"}},
	{path: ".shellcheckrc", stacks: []string{"shell", "shellcheck"}},
	{glob: "*.sh", stacks: []string{"shell"}},
	{path: ".sqlfluff", stacks: []string{"sql", "sqlfluff"}},
}

// An ambiguity is a finding that must not be decided here.
//
// The rule it serves is the spec's: what means the same everywhere is built
// in, what is project-dependent is asked. A false alarm costs more than a
// missing rule, so these leave the table as a question for the interview
// rather than as a stack.
type ambiguity struct {
	path string
	// unless names the file that resolves the question. A package.json is
	// only in doubt while no tsconfig.json stands beside it.
	unless string
	note   string
}

var ambiguities = []ambiguity{
	{
		path: "manage.py",
		note: "django: migrations may be generated or hand-written -- protect `migrations/[0-9][0-9][0-9][0-9]_*.py`?",
	},
	{
		path:   "package.json",
		unless: "tsconfig.json",
		note:   "package.json without tsconfig.json: a JavaScript project, or tooling for another language?",
	},
}

// The wiki question, kept beside the others although its shape differs: a
// directory rather than a file, and answered by a marker inside it.
const wikiNote = "wiki/ without an OKF marker in index.md: a brain bundle, or a plain docs folder?"

// SignalNames lists every stack name a signal can report; the presets of
// verify select their variants by these names and are checked against them.
func SignalNames() []string {
	set := map[string]bool{}
	for _, sig := range signals {
		for _, s := range sig.stacks {
			set[s] = true
		}
	}
	return sorted(set)
}
