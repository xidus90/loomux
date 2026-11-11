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
| typescript | `tsconfig.json` beside `package.json`, `biome.json` | eslint; biome where configured | `tsc` | vitest, where `package.json` names it | vitest with coverage | — | — | `.js` and `.jsx` count as TypeScript; no version pinned |
| vue | `*.vue`, `vue.config.js` | — | `vue-tsc` | — | — | — | — | no version pinned |
| svelte | `*.svelte`, `svelte.config.js` | — | `svelte-check` | — | — | — | — | no version pinned |
| css | `*.css`, `*.scss`, `.stylelintrc*` | stylelint | — | — | — | — | — | no version pinned |
| html | `*.html`, `.htmlhintrc` | htmlhint | — | — | — | — | — | no version pinned |
| gdscript | `project.godot`, `.gdlintrc` | gdlint, gdformat (gdtoolkit) | — | gdUnit4 through `check godot-suite`, or GUT through `check gut-suite`, where installed | NanoCoverage through `check linecov`, where installed | code graph | layer map | graph: Godot 3.x and 4.x, verified on 3.x, 4.2 and 4.7; test: gdUnit4 on Godot 4.3 to 4.8, GUT 7 on Godot 3 and GUT 9 on Godot 4 (verified on 3.6 and 4.7); coverage: Godot 4.3 to 4.8; lint: Godot 3 and 4 (on Godot 3 a project with addons in the tree fails `gdformat --check`, which cannot skip `addons/`), gdtoolkit pinnable with [`tools`](configuration.md#stacks); verified on Godot 3.6, 4.5 and 4.7 ([Verified combinations](#verified-combinations), [details](stacks/gdscript.md)) |
| csharp | `*.sln`, `*.slnx`, `*.csproj`, `global.json`, `Directory.Build.props` | `dotnet format` | `dotnet build -warnaserror` | `dotnet test`, where a `.csproj` names `Microsoft.NET.Test.Sdk` | coverlet (`XPlat Code Coverage`) through `check linecov` | code graph | layer map | the .NET SDK of the machine; .NET 10 and 11 verified ([Verified combinations](#verified-combinations)) |
| cpp | `CMakeLists.txt`, `CMakePresets.json`, `meson.build`, `*.cpp`, `*.hpp` | clang-tidy over the compile commands (`run-clang-tidy`); clang-format on the edited file | the CMake build | ctest, after the build, where `CMakeLists.txt` enables testing | gcovr, 100 % of lines | code graph | layer map | needs a configured `build/` (`CMakeCache.txt`); a root `CMakeLists.txt`, `CMakePresets.json` or `meson.build` makes the root the only area; C++17, 20 and 23 verified with GCC on Linux ([Verified combinations](#verified-combinations)); no compiler version pinned |
| java | `pom.xml` (Maven); `build.gradle`, `settings.gradle`, and their `.kts` forms where a `.java` file lies below (Gradle) | Maven: Checkstyle, where `pom.xml` configures `maven-checkstyle-plugin`; Gradle: `checkstyleMain checkstyleTest`, where a `build.gradle*` names `checkstyle`; a compile error shows here too | Maven: `compile test-compile`; Gradle: `compileJava compileTestJava` | Maven: `mvn test`; Gradle: `test` | JaCoCo through `check linecov`: Maven through the plugin 0.8.16 without configuration, Gradle where the build applies `jacoco` | code graph | layer map | the JDK, Maven and Gradle of the machine, Gradle through the project's wrapper where it has one; a root `pom.xml` or Gradle build makes the root the only area; with both in one area, Maven checks, and each area chooses its own build; not yet verified |
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

The C# entries name no tool: the .NET SDK brings format, build and test, and `csharp-xunit` pins its test packages itself (xunit.v3.mtp-off 4.0.3, xunit.runner.visualstudio 4.0.1, Microsoft.NET.Test.Sdk 18.11.1, coverlet.collector 10.1.0). That is xunit.v3 without Microsoft.Testing.Platform: the `xunit.v3` package brings MTP v2 from 4.0 on, and from the .NET 10 SDK on `dotnet test` refuses the VSTest route the preset takes (`--collect "XPlat Code Coverage"`). A run has the projects target the entry's .NET version (`net10.0`, `net11.0`).

A C++ version is an ISO standard. The C++ entries name no tool either: CMake and GCC are the Linux runner's, clang-tidy 23.1.4, clang-format 23.1.5 and gcovr 8.8 come through `uv tool install` (the clang-tidy wheel names its `run-clang-tidy` `run-clang-tidy.py`; the setup links it under the name the preset calls), and `cpp-cmake-gtest` fetches GoogleTest 1.18.0 itself, pinned by its release URL and SHA-256. A run configures `build/` with the entry's standard (`-DCMAKE_CXX_STANDARD=23`). C++ runs on Linux only: the Windows runner brings MSVC, which takes none of the fixture's GCC coverage flags. `godot4-cpp` holds a GDExtension in the godot-cpp layout (`project/` and `src/`).

The list holds every Go, Python, Godot and .NET version its upstream still maintains, not only the newest. C++ has no machine-readable source for its standards: the scout carries the next one itself, as the pre-release `26-draft`, until ISO publishes it and it moves into the list by hand; it reports no end of support for C++, since a standard leaves the list by hand too. Every Monday the compatibility scout (`loomux dev compat scout`) asks upstream three things:

- **A new stable version:** it runs the version as a probe and opens an issue with the entry to add.
- **An alpha, beta or release candidate of the next version:** it runs it as a probe ahead. A failure only opens an issue and blocks no pull request; a pre-release never enters the list, the release follows once the version is stable.
- **A listed version past end of support:** the answer comes from [endoflife.date](https://endoflife.date) (`isEol`). The version leaves the list. The pull request that drops it carries at least `release:patch` and names in its `## Changelog`, under `Removed`, the last loomux release that still supports it, for example: "Python 3.10 is no longer supported; v1.19.2 is the last release that supports it."

## What the code graph reads

The graph (`loomux graph`, the `graph` lane, `graph_*` over MCP) covers Go,
Python, GDScript, C#, C++ and Java. TypeScript is on the
[roadmap](../../README.md#roadmap).

| Language | Reads | Resolves | Leaves out |
|---|---|---|---|
| Go | packages, types, functions, methods | calls across packages of the module, imports | type-checked calls (`go/types`) |
| Python | modules, classes, functions, methods | imports, calls, base classes | `.pyi` stubs |
| GDScript | scripts, inner classes, functions, signals; scenes, resources, `project.godot`, `.uid` files | `res://` and `uid://` paths, `class_name`, autoloads (scene autoloads through their root script), inherited calls, signals by name and by value, scene connections, also to C# methods, signals and `[GlobalClass]` classes | VisualScript, scripts inline in a scene, types of locals and fields |
| C# | files, types, methods, constructors, properties, indexers, `[Signal]` delegates; syntax up to C# 14 in full | `using`, bases (`extends`/`implements`), calls, constructors | types of locals, methods in a second file of a `partial` class for scene connections; of C# 15: `union` (no type node), `closed record class` (no node), a type holding an extension indexer (no node), a labelled `break`/`continue` (an error in the body, the other members arrive) |
| C++ | files (`.cpp`, `.h`, `.hpp` and their kin), classes, structs, unions, enums, functions and methods with a body; GDExtension classes (`GDCLASS`) and their bindings; syntax of C++17, 20 and 23 | `#include`, base classes, calls by name or qualified name; GDScript `extends` and `.new()`, scene node types and connections to GDExtension classes | the preprocessor (macros stay unexpanded, every `#if` branch is read), types and overloads (member calls `x.f()` stay unresolved), `.c` files, a signal a base class of a GDExtension class binds (a scene connection finds only the class itself), a member only the C++ class has on a node that runs a script and has a type; of C++23: `if consteval` (the function holding it gets no node, and the definition before the closing brace of the namespace it sits in is lost; what follows arrives without namespace or owner; a file cut this way is read again only from a line at column 0, and a namespace or class open at the cut costs one definition) |
| Java | files, classes, interfaces, enums, records, annotation types, nested types, methods, constructors; syntax up to Java 21 in full | imports (single, `*`, `static`), bases (`extends`/`implements`), calls by name or qualified name, `new` | types and overloads (member calls `obj.m()` stay unresolved), types of locals; of Java 25: `import module` (read, but names no file), a statement before `super(…)` (an error in the constructor body, the node and the members after it arrive) |

The full rules are in the [CLI reference](cli-reference.md#loomux-graph-build---root-path---no-reuse).

## Stacks without a language

`wiki` and `wiki-sources` check the wiki of a project, `project` runs
commands a project names for itself; none of them reads code of a language.
See [Configuration](configuration.md#stacks).
