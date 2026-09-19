# Changelog

All notable changes to loomux are listed here, newest first. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [Semantic Versioning](https://semver.org/).

## [2.0.1] - 2026-09-19

<https://github.com/xidus90/loomux/pull/9>

### Fixed
- The pre-tool-use guard refuses shell commands (Bash and PowerShell) that write, move or delete `.loomux/config.toml`, as it already refused writing tools; reading it stays allowed.

## [2.0.0] - 2026-09-19

<https://github.com/xidus90/loomux/pull/8>

### Added
- `graph_find_code` and `graph_check_freshness` tools in `loomux serve` and the `loomux mcp` bridge, on the local and the cloud channel.
- Paths under an area's `[privacy] never` globs are kept out of code-graph answers and drift reports.
### Changed
- `loomux graph ask` no longer builds a graph that was never built: without one it exits 1 and points at `loomux graph build`.

## [1.3.0] - 2026-09-19

<https://github.com/xidus90/loomux/pull/7>

### Added
- `loomux serve` answers MCP over Streamable HTTP in stateless mode, one listener per channel, each guarded by its own bearer token. `loomux serve status` and `loomux serve stop` inspect and end it.
- `loomux mcp` bridges one stdio host to that service, starting it on demand and reconnecting when it restarts.
- `loomux dev record-mcp-case` records a call of the reference's MCP front, and the corpus under `testdata/cases/1b-2-*` replays those calls against this one.

### Fixed
- `loomux dev swap-binary` keeps up to sixteen numbered `loomux.old.<n>.exe` slots and sweeps the ones whose process has ended, so a swap no longer fails when a process started from an earlier swap still holds the old name.

## [1.2.1] - 2026-09-19

<https://github.com/xidus90/loomux/pull/5>

### Fixed
- `graph ask` rebuilds the graph when the ask index is missing or was written by a binary with another index version. Before, the freshness record stayed clean, no rebuild was triggered, and every later question ranked on names and paths only — until some source file happened to change.
- `graph build` reports a freshness record it could not write and exits 0, instead of exiting 1 for a build whose graph and sidecar are on disk and correct.
- `graph ask` restores the rebuild lock by hard link and falls back to a rename, instead of deleting its claim unconditionally. On a network share, exFAT or a container bind mount the live lock was removed and two runs rebuilt at the same time.

## [1.2.0] - 2026-09-18

<https://github.com/xidus90/loomux/pull/6>

### Added
- `loomux dev bench` measures pre-tool-use, post-tool-use and graph build latency on a repository or on the open-source corpus (`--corpus`, `--languages`, `--tier`), compares them with the project's own Claude hooks, reports uncovered native tools, and saves reports with `--save`.

### Fixed
- post-tool-use skips the `cmake --build build` lane with a notice when `build/CMakeCache.txt` is missing, instead of blocking the edit.

## [1.1.0] - 2026-09-18

<https://github.com/xidus90/loomux/pull/3>

### Added
- `loomux graph ask "<question>"` answers from the code graph with ranked definitions and their locations; `--in`, `--limit`, `--source`, `--full`, `--json` and `--no-refresh` shape the answer.
- `loomux graph build` writes an ask index next to the graph, so a question can match words in function bodies.

## [1.0.1] - 2026-09-18

<https://github.com/xidus90/loomux/pull/4>

### Fixed
- The `pr-label` check refuses a pull request containing a commit whose header is not a Conventional Commit.

## [1.0.0] - 2026-09-18

<https://github.com/xidus90/loomux/pull/1>

### Added
- `loomux version` names the release channel of a pre-release build, e.g. `loomux 1.0.0 (beta)`.
- `loomux dev release` computes the next version, checks a pull request's label, changelog and commits, writes the changelog entry and cross-builds the release binaries.
- `loomux check commit-msg` requires Conventional Commits headers.
- Releases with checksums for windows/amd64, linux/amd64, linux/arm64, darwin/amd64 and darwin/arm64.
