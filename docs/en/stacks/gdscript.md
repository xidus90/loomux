# GDScript: frameworks loomux drives

For GDScript projects loomux drives third-party frameworks for tests, coverage and lint. This page lists which ones, how loomux recognises each, and which Godot line each carries.

The table comes from `internal/verify/frameworks.toml`. A test compares every table row with the data, cell by cell, in English and in German, so neither can drift from the other.

A framework missing from the table is not forbidden. loomux drives what it finds and says what it does not know about a version it has not seen.

"Found by" gives the detection pattern as `file:text`: the file must contain the text, as a substring. Where the row is proven it is the pattern the lane's `when_found` carries; on an unproven row it is what a detection would use, since no lane detects that framework yet — loomux drives only what it has run. A test holds the data and the presets against each other both ways, so neither a proven pattern outside its lane nor an unproven one inside a lane passes.

The cell is empty where the lane needs no detection: the lint lane runs on any GDScript project, so neither gdtoolkit row names a pattern.

On Godot 3, gdformat (gdtoolkit 3) reads no config and cannot skip `addons/`, so a project with addons in the tree fails `gdformat --check`; the gdtoolkit of Godot 4 honours `excluded_directories` in `.gdformatrc`.

| Framework | Kind | Found by | Carries | Source |
|---|---|---|---|---|
| gdUnit4 | test | `plugin.cfg:name="gdUnit4"` | Godot 4.3 to 4.8 | https://github.com/godot-gdunit-labs/gdUnit4 |
| NanoCoverage | coverage | `plugin.cfg:name="NanoCoverage"` | Godot 4 | https://github.com/IgorBayerl/nano-coverage-godot |
| gdtoolkit4 | lint |  | Godot 4 | https://github.com/scony/godot-gdscript-toolkit |
| GUT | test | `plugin.cfg:name="Gut"` | GUT 7.x carries Godot 3, 9.x carries Godot 4; the pattern does not tell them apart | https://github.com/bitwes/Gut |
| gdtoolkit3 | lint |  | Godot 3 | https://github.com/scony/godot-gdscript-toolkit |
| godot-code-coverage (unproven) | coverage | `plugin.cfg:name="Coverage"` | documented for Godot 3.5, unconfirmed on 3.6 | https://github.com/jamie-pate/godot-code-coverage |

## Unproven rows

A row marked unproven has not been run by loomux against a real installation of the line it names. Still to do for each:

- godot-code-coverage (unproven): run a coverage lane through loomux on Godot 3.5, and one on 3.6.

## Rust extensions (godot-rust)

`loomux check godot-suite` and `loomux godot rescan` build a godot-rust extension of the project before they start Godot, because the suites and the import load the compiled library. The `rust` stack's own lanes do not order against a GDScript lane (an `after` stays inside one stack and one area), so the build lives in these two commands.

- **What counts**: every `.gdextension` in the project whose `[configuration]` has `entry_symbol = "gdext_rust_init"`. A C++ extension or a third party's addon has another symbol and is not touched.
- **What is built**: loomux does not pick a platform key. Godot chooses a library by feature tags, and real projects write `linux.editor.x86_64`, keys without an architecture and the like. Instead, every `[libraries]` path that points into `<dir>/target/<profile>/` (a `res://` path resolves against the directory of `project.godot`) builds that profile once for the host: `debug` as `cargo build`, `release` as `cargo build --release`, any other as `cargo build --profile <profile>`. `linux.debug`, `linux.editor` and `windows.debug` on `target/debug` are one build. A path that goes through a target triple (`target/wasm32-unknown-emscripten/debug/`) is not a host build and is skipped. An entry whose key carries the `release` feature tag (a dot-separated key segment equal to `release` or `template_release`: `linux.release.x86_64`, `windows.release`) is skipped as well, because the headless suite and the editor import load the debug or editor library and a release build would only double the cold cost. Keys with `debug`, `editor` or no build tag count, and the profile is still the one in the path. So the usual godot-rust template, with a `debug` and a `release` path per platform, builds only `debug`. If every Rust entry is release-only, nothing is built and the run says so in a note.
- **Where**: the nearest directory above `<dir>/target` that holds a `Cargo.toml`, searched up to the repository root and no further (the nearest directory with a `.git`, also a worktree's `.git` file; outside a git checkout the directory around the Godot project, so `godot/` and `rust/` side by side are found); for a workspace that is its root, so cargo builds every member. A `target/` outside the repository is not built (the run says so in a note), and a `Cargo.toml` above the repository is never used.
- **Known gap**: a custom target directory sends cargo's output somewhere the `.gdextension` does not point, so Godot loads the old library at the listed path without a warning. With `CARGO_TARGET_DIR` set in the environment the run says so in a note. A `build.target-dir` in `.cargo/config.toml` has the same effect and no note: loomux does not read cargo's configuration.
- **Failure**: a failing `cargo build` fails the run with cargo's output before any engine starts. Without `cargo` on `PATH` the run reports `cargo is not on PATH` as a finding, the way a missing Godot binary is reported by the same command.

### Time budget

The build runs inside the `loomux check godot-suite` process, so it counts against the lane's timeout, `[verify].timeout`, 600 s by default. Measured on 2026-11-15 with a crate that depends only on `godot = "=0.5.5"` (cargo 1.99.1, 16 logical cores, cold `target/`): `cargo build` 42 s (`cargo build --release` took 59 s, but a release entry is not built). That fits the default with room for the suite on a machine like that. A project with a larger crate or a machine with few cores can exceed it; then raise `[verify].timeout`, or run `cargo build` once beforehand so the lane only compiles what changed. Cargo builds incrementally, so a warm `target/` costs seconds.

### Windows: the library is in use

Windows refuses to overwrite a DLL a running process has loaded. Measured on Windows 11 with the headless editor (`--headless --editor`) and a headless game (`--headless --path`) on Godot 4.7.1 (mono console build) with godot-rust 0.5.5 and gdUnit4 6.2.2. Not measured: the editor with a window, other Godot versions, hot reload.

- **The headless editor did not hold the DLL.** `godot --headless --editor` loaded a copy (`~demo.dll` beside `demo.dll`), so `cargo build` and `loomux check godot-suite` built and tested while the editor had the project open. The loaded file did not depend on `reloadable`: the same with `true` and with `false`. What the key does for hot reload is untested.
- **A headless game did hold it.** `godot --headless --path <project>` (no `--editor`) loaded `demo.dll` itself, and `cargo build` then failed with `failed to remove file ... demo.dll` (`os error 5`), again with either value of `reloadable`. Stop a running game before the check.
