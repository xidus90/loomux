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
| go | `go.mod` | `go vet`, `gofmt` | — | `go test` | `gocover`, 100 % per function | code graph | layer map | Go syntax up to 1.27, the `go/parser` of the toolchain loomux is built with; a Go file that does not parse fails the graph build |
| python | `pyproject.toml`, `requirements.txt`, `uv.lock`, `manage.py` | ruff | mypy; pyright or the project's mypy where configured | pytest | coverage.py | code graph | layer map | Python 3, through `uv`; the preset's mypy needs 1.16 or later; no minor version checked one by one |
| typescript | `tsconfig.json` beside `package.json`, `biome.json` | eslint; biome where configured | `tsc` | vitest, where `package.json` names it | vitest with coverage | — | — | `.js` and `.jsx` count as TypeScript; no version pinned |
| vue | `*.vue`, `vue.config.js` | — | `vue-tsc` | — | — | — | — | no version pinned |
| svelte | `*.svelte`, `svelte.config.js` | — | `svelte-check` | — | — | — | — | no version pinned |
| css | `*.css`, `*.scss`, `.stylelintrc*` | stylelint | — | — | — | — | — | no version pinned |
| html | `*.html`, `.htmlhintrc` | htmlhint | — | — | — | — | — | no version pinned |
| gdscript | `project.godot`, `.gdlintrc` | gdlint, gdformat (gdtoolkit) | — | gdUnit4 through `check godot-suite`, where installed | NanoCoverage through `check linecov`, where installed | code graph | layer map | graph: Godot 3.x and 4.x, verified on 3.x, 4.2 and 4.7; test and coverage: Godot 4.3 to 4.8; lint: Godot 4, Godot 3 unproven ([details](stacks/gdscript.md)) |
| cpp | `CMakeLists.txt`, `CMakePresets.json`, `meson.build`, `*.cpp`, `*.hpp` | clang-tidy; clang-format on the edited file | the CMake build | ctest, where `CMakeLists.txt` enables testing | gcovr, 100 % of lines | — | — | needs a configured `build/` (`CMakeCache.txt`); no compiler version pinned |
| shell | `*.sh`, `.shellcheckrc` | shellcheck, on the edited file only | — | — | — | — | — | no version pinned |
| sql | `.sqlfluff` | sqlfluff | — | — | — | — | — | no version pinned |
| rust | `Cargo.toml` | clippy, `cargo fmt` | — | — | — | — | — | no version pinned |

"No version pinned" means loomux calls whatever version of the tool the
project or the machine provides and has not checked one against another.

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
