# Changelog

All notable changes to loomux are listed here, newest first. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [Semantic Versioning](https://semver.org/).

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
