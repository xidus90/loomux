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
| cpp | `CMakeLists.txt`, `CMakePresets.json`, `meson.build`, `*.cpp`, `*.hpp` | clang-tidy; clang-format on the edited file | the CMake build | ctest, where `CMakeLists.txt` enables testing | gcovr, 100 % of lines | — | — | needs a configured `build/` (`CMakeCache.txt`); no compiler version pinned |
| shell | `*.sh`, `.shellcheckrc` | shellcheck, on the edited file only | — | — | — | — | — | no version pinned |
| sql | `.sqlfluff` | sqlfluff | — | — | — | — | — | no version pinned |
| rust | `Cargo.toml` | clippy, `cargo fmt` | — | — | — | — | — | no version pinned |

"No version pinned" means loomux calls whatever version of the tool the
project or the machine provides and has not checked one against another.

## Verified combinations

Each row is a combination loomux runs for real before a major release and whenever the list changes: the runtime, the addons and the tools at the versions named, over a fixture project, on Linux and Windows. The table is `internal/compat/compat.toml`; a test holds it against this page cell by cell, and against the `proven` flag of every framework row ([GDScript](stacks/gdscript.md)). How the runs work is in the [CLI reference](cli-reference.md#loomux-dev-compat-matrixrunscout-flags), the hold they put on a release in [Releases](releases.md#how-a-pull-request-becomes-a-release).

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
| python | python | 3.15 | python-mypy | — | coverage 7.16.*, mypy 2.4.*, pytest 9.1.*, ruff 0.17.* | lint, types, test, coverage, graph |
| python | python | 3.15 | python-pyright | — | coverage 7.16.*, pytest 9.1.*, ruff 0.17.* | lint, types, test, coverage, graph |
| go | go | 1.26 | go-module | — | — | lint, test, coverage, graph |
| go | go | 1.27 | go-module | — | — | lint, test, coverage, graph |

## What the code graph reads

The graph (`loomux graph`, the `graph` lane, `graph_*` over MCP) covers Go,
Python and GDScript. TypeScript and C++ are on the
[roadmap](../../README.md#roadmap).

| Language | Reads | Resolves | Leaves out |
|---|---|---|---|
| Go | packages, types, functions, methods | calls across packages of the module, imports | type-checked calls (`go/types`) |
| Python | modules, classes, functions, methods | imports, calls, base classes | `.pyi` stubs |
| GDScript | scripts, inner classes, functions, signals; scenes, resources, `project.godot`, `.uid` files | `res://` and `uid://` paths, `class_name`, autoloads (scene autoloads through their root script), inherited calls, signals by name and by value, scene connections | C#, VisualScript, scripts inline in a scene, types of locals and fields |

The full rules are in the [CLI reference](cli-reference.md#loomux-graph-build---root-path---no-reuse).

## Stacks without a language

`wiki` and `wiki-sources` check the wiki of a project, `project` runs
commands a project names for itself; none of them reads code of a language.
See [Configuration](configuration.md#stacks).
