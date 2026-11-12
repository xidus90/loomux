# Supported languages and versions

Which languages loomux checks, what it runs for each, and which versions that
has been verified against. One row per stack of `[verify]`; the details are on
the pages linked below.

The guard and the write barrier read paths and commands, not code, so they
hold for every language alike and have no column here.

## Overview

A cell names what the built-in preset runs; `—` means loomux ships nothing
for that kind and a project that wants one configures it under
`[verify.<stack>]` ([Configuration](configuration.md#stacks)). A test holds
every kind column against the embedded presets (`internal/verify/presets.toml`),
in English and in German, so a cell cannot promise a lane the binary does not
have.

| Stack | Recognised by | lint | types | test | coverage | graph | rules | Versions |
|---|---|---|---|---|---|---|---|---|
| go | `go.mod` | `go vet`, `gofmt` | — | `go test` | `gocover`, 100 % per function | code graph | layer map | Go syntax up to 1.27, the `go/parser` of the toolchain loomux is built with; a Go file that does not parse fails the graph build; Go 1.26 and 1.27 verified ([Verified combinations](#verified-combinations)) |
| python | `pyproject.toml`, `requirements.txt`, `uv.lock`, `manage.py` | ruff | mypy; pyright or the project's mypy where configured | pytest | coverage.py | code graph | layer map | Python 3, through `uv`; the preset's mypy needs 1.16 or later; tool versions pinnable with [`tools`](configuration.md#stacks); Python 3.11 to 3.15 verified ([Verified combinations](#verified-combinations)) |
| typescript | `tsconfig.json` beside `package.json`; `package.json` with a `.js`, `.mjs`, `.cjs`, `.jsx`, `.ts`, `.mts`, `.cts` or `.tsx` file below, where tooling configuration at its side (`*.config.*`, `.*rc.*`) does not count and the search stops at nested packages; `biome.json`; each member the root declares in `package.json`'s `workspaces` or in `pnpm-workspace.yaml` is an area of its own | eslint; biome where configured | `tsc`, where the area has a `tsconfig.json` of its own | vitest, where a `package.json` of the area or above names it and the area holds `*.test.*`, `*.spec.*` or `__tests__/`; Jest instead, where `package.json` names `jest` or a `jest.config.*` exists; Mocha and `node --test` not | vitest's lcov report through `check linecov`, where `@vitest/coverage-v8` is installed; Jest's with `--coverage --coverageReporters=lcov` | code graph | layer map | JavaScript counts as TypeScript; the tools from the `node_modules/.bin` of the area or of a directory above it up to the root, as `npx` finds them (in a workspace the root's): before `npm install` their lanes are unavailable, since `npx` would fetch the newest version unpinned, and Yarn Plug'n'Play, which has no `node_modules`, keeps them unavailable; Node 22, 24 and 26 verified with ESLint 10.14.0, typescript-eslint 8.72.2, TypeScript 6.0.3 and Vitest 5.1.0, Node 26 also with TypeScript 7.0.2 ([Verified combinations](#verified-combinations)); Jest is in no verified combination yet |
| vue | `*.vue`, `vue.config.js` | — | `vue-tsc` | — | — | — | — | no version pinned |
| svelte | `*.svelte`, `svelte.config.js` | — | `svelte-check` | — | — | — | — | no version pinned |
| css | `*.css`, `*.scss`, `.stylelintrc*` | stylelint | — | — | — | — | — | no version pinned |
| html | `*.html`, `.htmlhintrc` | htmlhint | — | — | — | — | — | no version pinned |
| gdscript | `project.godot`, `.gdlintrc` | gdlint, gdformat (gdtoolkit) | — | gdUnit4 through `check godot-suite`, or GUT through `check gut-suite`, where installed | NanoCoverage through `check linecov`, where installed | code graph | layer map | graph: Godot 3.x and 4.x, verified on 3.x, 4.2 and 4.7; test: gdUnit4 on Godot 4.3 to 4.8, GUT 7 on Godot 3 and GUT 9 on Godot 4 (verified on 3.6 and 4.7); coverage: Godot 4.3 to 4.8; lint: Godot 3 and 4 (on Godot 3 a project with addons in the tree fails `gdformat --check`, which cannot skip `addons/`), gdtoolkit pinnable with [`tools`](configuration.md#stacks); verified on Godot 3.6, 4.5 and 4.7 ([Verified combinations](#verified-combinations), [details](stacks/gdscript.md)) |
| csharp | `*.sln`, `*.slnx`, `*.csproj`, `global.json`, `Directory.Build.props` | `dotnet format` | `dotnet build -warnaserror` | `dotnet test`, where a `.csproj` names `Microsoft.NET.Test.Sdk` or `xunit.v3` | coverlet (`XPlat Code Coverage`) through `check linecov`; on Microsoft.Testing.Platform Microsoft's code coverage extension (Cobertura), where a `.csproj` names `Microsoft.Testing.Extensions.CodeCoverage` | code graph | layer map | the .NET SDK of the machine; a `global.json` that chooses `Microsoft.Testing.Platform` as its test runner switches test and coverage to MTP; .NET 10 and 11 verified, with VSTest and with MTP ([Verified combinations](#verified-combinations)) |
| cpp | `CMakeLists.txt`, `CMakePresets.json`, `meson.build`, `*.cpp`, `*.hpp` | clang-tidy over the compile commands (`run-clang-tidy`); clang-format on the edited file | the CMake build | ctest, after the build, where `CMakeLists.txt` enables testing | gcovr, 100 % of lines | code graph | layer map | needs a configured `build/` (`CMakeCache.txt`); a root `CMakeLists.txt`, `CMakePresets.json` or `meson.build` makes the root the only area; C++17, 20 and 23 verified with GCC on Linux ([Verified combinations](#verified-combinations)); no compiler version pinned |
| java | `pom.xml` (Maven); `build.gradle`, `settings.gradle`, and their `.kts` forms where a `.java` file lies below (Gradle) | Maven: Checkstyle, where `pom.xml` configures `maven-checkstyle-plugin`; Gradle: `checkstyleMain checkstyleTest`, where a `build.gradle*` names `checkstyle`; a compile error shows here too | Maven: `compile test-compile`; Gradle: `compileJava compileTestJava` | Maven: `mvn test`; Gradle: `test`; after the compile | JaCoCo through `check linecov`: Maven through the plugin 0.8.16 without configuration, Gradle where the build applies `jacoco` | code graph | layer map | the JDK, Maven and Gradle of the machine, Gradle through the project's wrapper where it has one; a root `pom.xml` or Gradle build makes the root the only area; with both in one area, Maven checks, and each area chooses its own build; an Android plugin switches the build lanes off; Java 8, 11, 17, 21, 25 and 27 verified on Temurin with Maven 3.10.0, Java 21 also with Gradle 9.9.1 ([Verified combinations](#verified-combinations)) |
| php | `composer.json`, `phpunit.xml`, `phpunit.xml.dist`, `phpunit.dist.xml` | PHP_CodeSniffer, PSR-12 where the project configures none; `php -l` on the edited file too, also before `composer install` | PHPStan; where the project configures none, level `max` on the autoload paths of `composer.json`, else on `src`, else unavailable | PHPUnit | PHPUnit's Cobertura report through `check linecov`, with Xdebug | code graph | layer map | the `php` of the machine, the tools from the project's `vendor/bin`: before `composer install` their lanes are unavailable; a root `composer.json` makes the root the only area; without a `composer.json` (or `phpunit.xml`) no php stack is detected and none of these lanes runs, while the code graph still reads the `.php` files; PHP 8.2, 8.3, 8.4 and 8.5 verified with PHPUnit 11 to 13, PHPStan 2.3.5 and PHP_CodeSniffer 4.0.4 ([Verified combinations](#verified-combinations)) |
| shell | `*.sh`, `.shellcheckrc` | shellcheck, on the edited file only | — | — | — | — | — | no version pinned |
| sql | `.sqlfluff` | sqlfluff | — | — | — | — | — | no version pinned |
| rust | `Cargo.toml` | clippy, `cargo fmt` | — | — | — | — | — | no version pinned |

"No version pinned" means loomux calls whatever version of the tool the
project or the machine provides and has not checked one against another.

## Verified combinations

Each row is a combination loomux runs for real before a major release and whenever the list changes: the runtime, the addons and the tools at the versions named, over a fixture project, on Linux and Windows (C++ on Linux only). The table is `internal/compat/compat.toml`; a test holds it against this page cell by cell, and against the `proven` flag of every framework row ([GDScript](stacks/gdscript.md)). How the runs work is in the [CLI reference](cli-reference.md#loomux-dev-compat-matrixrunscout-flags), the hold they put on a release in [Releases](releases.md#how-a-pull-request-becomes-a-release).

| Stack | Runtime | Version | Fixture | Addons | Tools | Kinds |
|---|---|---|---|---|---|---|
| gdscript | godot | 3.6 | godot3-gut | GUT 7.4.3 | — | test, graph |
| gdscript | godot | 3.6 | godot3-plain | — | gdtoolkit 3.* | lint, graph |
| gdscript | godot | 4.5 | godot4-plain | — | gdtoolkit 4.* | lint, graph |
| gdscript | godot | 4.7 | godot4-plain | — | gdtoolkit 4.* | lint, graph |
| gdscript | godot | 4.5 | godot4-gdunit4 | gdUnit4 6.2.2, NanoCoverage 0.1.1 | gdtoolkit 4.* | lint, test, coverage, graph |
| gdscript | godot | 4.7 | godot4-gdunit4 | gdUnit4 6.2.2, NanoCoverage 0.1.1 | gdtoolkit 4.* | lint, test, coverage, graph |
| gdscript | godot | 4.7 | godot4-gut | GUT 9.7.1 | gdtoolkit 4.* | lint, test, graph |
| python | python | 3.11 | python-mypy | — | coverage 7.16.*, mypy 2.4.*, pytest 9.1.*, ruff 0.17.* | lint, types, test, coverage, graph |
| python | python | 3.12 | python-mypy | — | coverage 7.16.*, mypy 2.4.*, pytest 9.1.*, ruff 0.17.* | lint, types, test, coverage, graph |
| python | python | 3.13 | python-mypy | — | coverage 7.16.*, mypy 2.4.*, pytest 9.1.*, ruff 0.17.* | lint, types, test, coverage, graph |
| python | python | 3.14 | python-mypy | — | coverage 7.16.*, mypy 2.4.*, pytest 9.1.*, ruff 0.17.* | lint, types, test, coverage, graph |
| python | python | 3.15 | python-mypy | — | coverage 7.17.*, mypy 2.4.*, pytest 9.2.*, ruff 0.17.* | lint, types, test, coverage, graph |
| python | python | 3.15 | python-pyright | — | coverage 7.17.*, pytest 9.2.*, ruff 0.17.* | lint, types, test, coverage, graph |
| go | go | 1.26 | go-module | — | — | lint, test, coverage, graph |
| go | go | 1.27 | go-module | — | — | lint, test, coverage, graph |
| go | go | 1.26 | go-syntax-1.26 | — | — | graph |
| go | go | 1.27 | go-syntax-1.27 | — | — | graph |
| csharp | dotnet | 10.0 | csharp-xunit | — | — | lint, types, test, coverage, graph |
| csharp | dotnet | 11.0 | csharp-xunit | — | — | lint, types, test, coverage, graph |
| csharp | dotnet | 10.0 | csharp-xunit-mtp | — | — | lint, types, test, coverage, graph |
| csharp | dotnet | 11.0 | csharp-xunit-mtp | — | — | lint, types, test, coverage, graph |
| csharp | dotnet | 10.0 | csharp-syntax-10.0 | — | — | graph |
| csharp | dotnet | 11.0 | csharp-syntax-11.0 | — | — | graph |
| csharp | dotnet | 10.0 | godot4-csharp | — | — | graph |
| cpp | cpp | 17 | cpp-cmake-gtest | — | — | lint, types, test, coverage, graph |
| cpp | cpp | 20 | cpp-cmake-gtest | — | — | lint, types, test, coverage, graph |
| cpp | cpp | 23 | cpp-cmake-gtest | — | — | lint, types, test, coverage, graph |
| cpp | cpp | 17 | cpp-syntax-17 | — | — | graph |
| cpp | cpp | 20 | cpp-syntax-20 | — | — | graph |
| cpp | cpp | 23 | cpp-syntax-23 | — | — | graph |
| cpp | cpp | 17 | godot4-cpp | — | — | graph |
| java | java | 8 | java-maven-junit5 | — | — | lint, types, test, coverage, graph |
| java | java | 11 | java-maven-junit5 | — | — | lint, types, test, coverage, graph |
| java | java | 17 | java-maven-junit6-17 | — | — | lint, types, test, coverage, graph |
| java | java | 21 | java-maven-junit6 | — | — | lint, types, test, coverage, graph |
| java | java | 25 | java-maven-junit6 | — | — | lint, types, test, coverage, graph |
| java | java | 27 | java-maven-junit6 | — | — | lint, types, test, coverage, graph |
| java | java | 21 | java-gradle-junit6 | — | — | lint, types, test, coverage, graph |
| java | java | 8 | java-syntax-8 | — | — | graph |
| java | java | 11 | java-syntax-11 | — | — | graph |
| java | java | 17 | java-syntax-17 | — | — | graph |
| java | java | 21 | java-syntax-21 | — | — | graph |
| java | java | 25 | java-syntax-25 | — | — | graph |
| java | java | 27 | java-syntax-27 | — | — | graph |
| php | php | 8.2 | php-phpunit11 | — | — | lint, types, test, coverage, graph |
| php | php | 8.3 | php-phpunit12 | — | — | lint, types, test, coverage, graph |
| php | php | 8.4 | php-phpunit13 | — | — | lint, types, test, coverage, graph |
| php | php | 8.5 | php-phpunit13 | — | — | lint, types, test, coverage, graph |
| php | php | 8.2 | php-syntax-8.2 | — | — | graph |
| php | php | 8.3 | php-syntax-8.3 | — | — | graph |
| php | php | 8.4 | php-syntax-8.4 | — | — | graph |
| php | php | 8.5 | php-syntax-8.5 | — | — | graph |
| typescript | node | 22 | ts-vitest-eslint | — | — | lint, types, test, coverage, graph |
| typescript | node | 24 | ts-vitest-eslint | — | — | lint, types, test, coverage, graph |
| typescript | node | 26 | ts-vitest-eslint | — | — | lint, types, test, coverage, graph |
| typescript | node | 26 | ts7-vitest | — | — | types, test, coverage, graph |
| typescript | node | 22 | js-syntax-22 | — | — | graph |
| typescript | node | 24 | js-syntax-24 | — | — | graph |
| typescript | node | 26 | js-syntax-26 | — | — | graph |

The C# entries name no tool: the .NET SDK brings format, build and test, and `csharp-xunit` pins its test packages itself (xunit.v3.mtp-off 4.0.3, xunit.runner.visualstudio 4.0.1, Microsoft.NET.Test.Sdk 18.11.1, coverlet.collector 10.1.0). That is xunit.v3 without Microsoft.Testing.Platform: the `xunit.v3` package brings MTP v2 from 4.0 on, and from the .NET 10 SDK on `dotnet test` refuses the VSTest route the preset takes (`--collect "XPlat Code Coverage"`). `csharp-xunit-mtp` is the same project as xunit.v3 sets it up by default (xunit.v3 4.0.3, Microsoft.Testing.Extensions.CodeCoverage 18.12.0), with a `global.json` that chooses `Microsoft.Testing.Platform` as the test runner; the run adds its SDK pin beside that choice. A run has the projects target the entry's .NET version (`net10.0`, `net11.0`).

A C++ version is an ISO standard. The C++ entries name no tool either: CMake and GCC are the Linux runner's, clang-tidy 23.1.4, clang-format 23.1.5 and gcovr 8.8 come through `uv tool install` (the clang-tidy wheel names its `run-clang-tidy` `run-clang-tidy.py`; the setup links it under the name the preset calls), and `cpp-cmake-gtest` fetches GoogleTest 1.18.0 itself, pinned by its release URL and SHA-256. A run configures `build/` with the entry's standard (`-DCMAKE_CXX_STANDARD=23`). C++ runs on Linux only: the Windows runner brings MSVC, which takes none of the fixture's GCC coverage flags. `godot4-cpp` holds a GDExtension in the godot-cpp layout (`project/` and `src/`).

A Java version is a feature release of Eclipse Temurin, which `actions/setup-java` installs. The Java entries name no tool either: CI fetches Maven 3.10.0 from Maven Central and checks it against its SHA-512 on both systems, since the runners bring different ones, and `java-gradle-junit6` runs Gradle 9.9.1 through its wrapper, which pins the distribution by SHA-256. The fixtures pin their plugins themselves (maven-compiler-plugin 3.16.0, maven-surefire-plugin 3.6.1, maven-checkstyle-plugin 3.6.0; JaCoCo 0.8.16 measures), and JUnit and Checkstyle by the Java they run on: JUnit 6 and Checkstyle 12 need Java 17, Checkstyle 14 Java 21, so 8 and 11 run JUnit 5.14.4 and Checkstyle 9.3, 17 JUnit 6.1.3 and Checkstyle 12.3.1, 21 and later JUnit 6.1.3 and Checkstyle 14.4.0. A run compiles the fixture at the entry's version: `maven.compiler.release`, `source` and `target` where it also runs on Java 8, whose javac takes no `--release`, and Gradle's toolchain. Java 27 brought no final syntax of its own, so `java-syntax-27` carries the forms of 25.

A PHP version is a release branch of php.net, which `shivammathur/setup-php` 2.40.0 installs with Xdebug 3.5.3 and Composer 2.11.2. The PHP entries name no tool either: each fixture pins PHPUnit, PHPStan 2.3.5 and PHP_CodeSniffer 4.0.4 in its `composer.lock`, which a run installs into each work copy (`composer install --no-interaction --prefer-dist`) before the lanes. PHPUnit sets the floor: 11.5.57 needs PHP 8.2, 12.5.43 PHP 8.3, 13.4.6 PHP 8.4.1, so `php-phpunit11` runs on 8.2, `php-phpunit12` on 8.3 and `php-phpunit13` on 8.4 and 8.5, each lock resolved for the lowest PHP its fixture runs on. The fixtures configure neither PHP_CodeSniffer nor PHPStan, so the lanes run PSR-12 and level `max`.

A Node version is a release line of nodejs.org, which `actions/setup-node` 7.1.0 installs. The Node entries name no tool either: each fixture pins its tools in its `package-lock.json`, which a run installs into each work copy (`npm ci --no-audit --no-fund`) before the lanes. `ts-vitest-eslint` runs ESLint 10.14.0 with typescript-eslint 8.72.2 on TypeScript 6.0.3; typescript-eslint takes no TypeScript 7, so `ts7-vitest` runs TypeScript 7.0.2 without ESLint, on the newest line. Both run Vitest 5.1.0 with @vitest/coverage-v8 5.1.0 on Vite 8.3.8, and in both a JavaScript module imports a TypeScript one. Node 26 brought no JavaScript syntax of its own, so `js-syntax-26` carries the forms of 24, with TypeScript beside them that Node runs by stripping its types.

The list holds every Go, Python, Godot, .NET, Java, PHP and Node version its upstream still maintains, not only the newest. Node's releases come from nodejs.org's dist index (`dist/index.json`), which lists the alpha of the next line (`27.0.0-alpha.0.0.0`) as its pre-release; an entry names a Node line by its major, so a release candidate of a new minor of a listed line is none. Node's end of support comes from endoflife.date's `nodejs`. Java's feature releases come from the Adoptium API (`available_releases`); its tip, while it has early-access builds only, is the pre-release (`28-ea`), and end of support comes from endoflife.date's `eclipse-temurin`. PHP's newest release comes from php.net's release list, its release candidates from php.net's own list of them (a branch whose release is disabled has none); `php-phpunit11` and `php-phpunit12` are held, so only `php-phpunit13` probes the next one. setup-php offers no release candidate, so a probe of `8.6.0RC5` runs the nightly build of the 8.6 branch with the Xdebug setup-php sets up for it: Xdebug 3.6.0alpha1 is not on PECL, where setup-php takes a pinned version from. C++ has no machine-readable source for its standards: the scout carries the next one itself, as the pre-release `26-draft`, until ISO publishes it and it moves into the list by hand; it reports no end of support for C++, since a standard leaves the list by hand too. Every Monday the compatibility scout (`loomux dev compat scout`) asks upstream three things:

- **A new stable version:** it runs the version as a probe and opens an issue with the entry to add. An entry whose fixture stays below newer versions on purpose, because its framework ends there, carries `held` in the list with the reason; the scout asks nothing newer for it. Any other line below the newest version of its runtime keeps reporting until it moves up.
- **An alpha, beta or release candidate of the next version:** it runs it as a probe ahead. A failure only opens an issue and blocks no pull request; a pre-release never enters the list, the release follows once the version is stable.
- **A listed version past end of support:** the answer comes from [endoflife.date](https://endoflife.date) (`isEol`). The version leaves the list. The pull request that drops it carries at least `release:patch` and names in its `## Changelog`, under `Removed`, the last loomux release that still supports it, for example: "Python 3.10 is no longer supported; v1.19.2 is the last release that supports it."

## What the code graph reads

The graph (`loomux graph`, the `graph` lane, `graph_*` over MCP) covers Go,
Python, GDScript, C#, C++, Java, PHP, JavaScript and TypeScript.

| Language | Reads | Resolves | Leaves out |
|---|---|---|---|
| Go | packages, types, functions, methods | calls across packages of the module, imports | type-checked calls (`go/types`) |
| Python | modules, classes, functions, methods | imports, calls, base classes | `.pyi` stubs |
| GDScript | scripts, inner classes, functions, signals; scenes, resources, `project.godot`, `.uid` files | `res://` and `uid://` paths, `class_name`, autoloads (scene autoloads through their root script), inherited calls, signals by name and by value, scene connections, also to C# methods, signals and `[GlobalClass]` classes | VisualScript, scripts inline in a scene, types of locals and fields |
| C# | files, types, methods, constructors, properties, indexers, `[Signal]` delegates; syntax up to C# 14 in full | `using`, bases (`extends`/`implements`), calls, constructors | types of locals, methods in a second file of a `partial` class for scene connections; of C# 15: `union` (no type node), `closed record class` (no node), a type holding an extension indexer (no node), a labelled `break`/`continue` (an error in the body, the other members arrive) |
| C++ | files (`.cpp`, `.h`, `.hpp` and their kin), classes, structs, unions, enums, functions and methods with a body; GDExtension classes (`GDCLASS`) and their bindings; syntax of C++17, 20 and 23 | `#include`, base classes, calls by name or qualified name; GDScript `extends` and `.new()`, scene node types and connections to GDExtension classes | the preprocessor (macros stay unexpanded, every `#if` branch is read), types and overloads (member calls `x.f()` stay unresolved), `.c` files, a signal a base class of a GDExtension class binds (a scene connection finds only the class itself), a member only the C++ class has on a node that runs a script and has a type; of C++23: `if consteval` (the function holding it gets no node, and the definition before the closing brace of the namespace it sits in is lost; what follows arrives without namespace or owner; a file cut this way is read again only from a line at column 0, and a namespace or class open at the cut costs one definition) |
| Java | files, classes, interfaces, enums, records, annotation types, nested types, methods, constructors; syntax up to Java 21 in full | imports (single, `*`, `static`), bases (`extends`/`implements`), calls by name or qualified name, `new` | types and overloads (member calls `obj.m()` stay unresolved), types of locals; of Java 25: `import module` (read, but names no file), a statement before `super(…)` (an error in the constructor body, the node and the members after it arrive) |
| PHP | files, classes, interfaces, traits, enums, methods, functions, also in several `namespace` blocks of one file and between HTML; syntax up to PHP 8.4 in full | `use` of classes and functions (a `use const` is read but gives no edge), `require`/`include` of a string literal, bases (`extends`/`implements`, traits), calls by name or qualified name, `self::`, `static::`, `parent::`, `$this->`, `new` | types of variables (member calls `$obj->m()` stay unresolved); of PHP 8.5: `clone($o, [...])` and the `(void)` cast (an error in the body, the rest arrives); a file heavy with heredocs that hold PHP code can take about 20 s to parse and may lose declarations to parse errors; the graph keeps each file's result until it changes, so the graph waits once, but `check rules` parses every file again on each run, so with `[rules]` set the wait recurs on every edit and commit |
| JavaScript, TypeScript | files (`.js`, `.mjs`, `.cjs`, `.jsx`, `.ts`, `.mts`, `.cts`, `.tsx`), functions, classes, methods, functions bound to a `const`, `let` or `var` at module level, CommonJS `exports.f = …` and `module.exports.f = …`; TS interfaces, types, enums, every declaration of a `namespace` (as `N.f`) | `import`, `export … from`, `require` and `import()` of a string literal, path aliases of `tsconfig.json`/`jsconfig.json`, bases, calls by name, `this.`, `super.`, through an imported namespace, `new`, JSX components (`<X/>`, `<ns.X/>`) as calls | imports of packages, subpath imports (`#x`) from `package.json`'s `imports`, `extends` in `tsconfig.json`, a change to `tsconfig.json` or `jsconfig.json` alone (the graph stays fresh until a source file changes, as with `go.mod`), types (member calls `obj.m()` stay unresolved, as do static `C.s()` and `N.f()` into a TS namespace), declaration files (`.d.ts`, `.d.mts`, `.d.cts`: file node only), nested functions, object methods, ambient declarations (`declare …`); a JSX attribute `class=` in a `.js` file (the definition holding it gets no node, and the rest of the file is read again only from declarations in column 0; `.jsx` reads it) |

The full rules are in the [CLI reference](cli-reference.md#loomux-graph-build---root-path---no-reuse).

## Stacks without a language

`wiki` and `wiki-sources` check the wiki of a project, `project` runs
commands a project names for itself; none of them reads code of a language.
See [Configuration](configuration.md#stacks).
