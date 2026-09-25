# Changelog

All notable changes to loomux are listed here, newest first. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [Semantic Versioning](https://semver.org/).

## [2.13.1] - 2026-09-25

<https://github.com/xidus90/loomux/pull/38>

### Fixed
- A session resumed in a linked worktree with `[worktree] mirror` keeps its stop-gate base, so the next turn end still checks every commit since the last green run instead of none.
- A session resumed more than a day after its last turn end counts again for `worktree unlink`, so another session ending in the same worktree no longer removes its junctions.

## [2.13.0] - 2026-09-25

<https://github.com/xidus90/loomux/pull/36>

### Added
- `loomux init`: set up a project interactively or with `--yes`, preview with `--dry-run`, inspect with `--detect-only`; choose modules and parts with `--hooks`, `--brain`, `--graph` and hosts with `--hosts`.
- `loomux merge-hook install|status|remove|record`: record merges on an area's branch for `reconcile`, through a post-merge hook that works in every clone; `install` and `remove` are for humans, the guard refuses them to agents.
- A multi-select in the full-screen terminal forms.
### Fixed
- A `core.hooksPath` starting with `~` is now read the way git expands it.

## [2.12.1] - 2026-09-24

<https://github.com/xidus90/loomux/pull/28>

### Fixed
- `loomux approve` writes its line in `log.md` newest first, under one `## YYYY-MM-DD` heading per day as OKF §9 requires, instead of appending it at the end of the file.

## [2.12.0] - 2026-09-24

<https://github.com/xidus90/loomux/pull/27>

### Added
- `loomux config` with `list`, `get`, `set`, `unset` and an interactive full-screen form, plus `--global` for the machine-wide file.
- `[modules]` in `.loomux/config.toml`: `hooks`, `brain` and `graph` switch the session hooks, the wiki lane and the matching MCP tools off per project; a missing key means on.
- `loomux mcp --root <dir>`; without it the bridge looks for the project upwards from where the host starts it.
- Configuration proposals for agents: `loomux config set|unset … --propose`, `loomux config proposals`, `loomux config apply` and `loomux config reject`.

### Changed
- The guard refuses an agent `loomux init`, `loomux config set`, `loomux config unset`, `loomux config apply`, `loomux config reject`, the interactive `loomux config` and `loomux area add`; `loomux config list`, `get`, `proposals` and a direct `set|unset … --propose` stay allowed.
- `loomux mcp` offers only the tools of the modules a project leaves on; a broken `[modules]` table makes it exit 1.

## [2.11.1] - 2026-09-24

<https://github.com/xidus90/loomux/pull/25>

### Fixed
- `graph blast`, `graph_blast` and `check blast-audit` no longer mix up the hunks of two files when git's `diff.interHunkContext` is set.
- The blast radius no longer counts unchanged symbols between two nearby changes as changed.
- `graph_blast` reports `changed` instead of `stale` when a test file hidden by a `never` rule changed together with the code it covers.

## [2.11.0] - 2026-09-24

<https://github.com/xidus90/loomux/pull/21>

### Added
- The write barrier reads `open.toml` in the state directory (`files = ["C:/Users/me/.claude/AGENT_LEARNINGS.md"]`) and keeps each single file listed there open to every agent, like the agents' memory. An unusable `open.toml` opens nothing, and the refusal names the reason.

## [2.10.0] - 2026-09-24

<https://github.com/xidus90/loomux/pull/24>

### Added
- `loomux graph blast [--cached|--base X] [-d N|all] [--json]`: the blast radius of a git diff with a test signal per changed file.
- MCP tool `graph_blast` (scope, base, depth), with refused paths counted and never quoted on the cloud channel.
- `loomux check graph-fresh`: rebuilds a drifted, foreign or outdated graph for a gate and waits up to 30 s for another rebuild.
- `loomux check blast-audit [--threshold N] [--skip-test-callers]`: fails when a changed, well-connected symbol has no changed test reaching it.
- Verify kind `graph` in the default precommit profile, with a Go lane at threshold 5 that is not-applicable where it cannot mean anything.
- post-tool-use names the callers an edit to a Go symbol may break, and notes a changed type the graph cannot follow yet.

### Fixed
- `graph_trace_calls` treated a depth like 0.5 as zero and returned no callers; it now floors to at least one.

## [2.9.1] - 2026-09-24

<https://github.com/xidus90/loomux/pull/23>

### Fixed
- On Windows a `*` in a `[policy]` path glob no longer matches across `/`; use `/**` to guard a whole tree. A glob ending in a lone backslash is now refused when the config loads.
- `graph grep` no longer rejects valid patterns such as `\\1` (a literal backslash before a digit) or `\(?=` as unsupported.
- On Windows, git variables such as `git_dir` spelled in lower or mixed case are no longer passed to loomux's git child processes.

## [2.9.0] - 2026-09-24

<https://github.com/xidus90/loomux/pull/22>

### Added
- `loomux self-update` replaces the machine-wide binary with the newest release of its channel.
- `loomux serve` checks for a newer release a minute after it starts and daily after that, and installs it (Windows only; needs the GitHub CLI, logged in).
- Session start warns when `serve` runs from another binary than `%LOCALAPPDATA%\loomux\bin\loomux.exe`, or when its last self-update failed.
### Fixed
- `loomux dev swap-binary` puts the previous binary back when the new one cannot take its place, instead of leaving no `loomux.exe` behind.

## [2.8.0] - 2026-09-23

<https://github.com/xidus90/loomux/pull/20>

### Added
- `loomux brain check file|bundle|all [--notes]` checks pages, bundles and the federation against the Open Knowledge Format and the house rules.
- `loomux lint --scope all|<scope>` lints every registered bundle, or one, by the rules of the reference lint.
- `loomux wiki init`, `loomux wiki types` and `loomux wiki retype` lay out a bundle, count page types across every area, and rename a page type.
- `loomux serve` catches up on the daily reconciliation before its first answer when the last pass is more than a day old, and reports open cases and failures of the pass with its answers.

### Changed
- `loomux lint` without a file no longer stops with a usage error; it lints every registered bundle.

## [2.7.0] - 2026-09-23

<https://github.com/xidus90/loomux/pull/19>

### Added
- `loomux cases` lists the review cases waiting for a decision.
- `loomux case <id> [--package]` shows a case with its package and proposal; local-only material is withheld unless `--package` is given.
- `loomux approve <id>` applies a proposal whose every claim quotes its evidence, and commits the change; `--amend PATH` approves a corrected proposal, `--reject` rejects it, `--defer` leaves the case waiting.

## [2.6.0] - 2026-09-23

<https://github.com/xidus90/loomux/pull/18>

### Added
- Code graph navigation commands: `loomux graph callers`, `loomux graph skeleton`, `loomux graph grep`, `loomux graph map`, and `loomux graph stats`
- Four new MCP tools: `graph_file_api`, `graph_trace_calls`, `graph_find_all`, and `graph_repo_map` with fail-closed privacy redaction on cloud channel
- Query orchestration and pure algorithmic packages for AST symbol spans, blast resolution/walks, skeleton extraction, symbol-coupled grep, and token-budgeted repo orientation maps

## [2.5.0] - 2026-09-22

<https://github.com/xidus90/loomux/pull/16>

### Added
- `loomux reindex [--registry P]` rebuilds the identity register, directory catalogs, link graph and qmd collections of every registered area, running a reconcile pass first so no knowledge change slips past the review gate; open cases are listed and the run goes on.
- `loomux embed [--registry P]` generates the vectors qmd has pending and names the install command when qmd is not on PATH.
- `loomux reconcile` checks every registered source against its identity register and opens a case with a diff package in the review centre for each changed source and each landed merge; open cases are not a failure, an unreadable case file is.
- `loomux area add` registers a repository as an area, writes its `.loomux/config.toml` when there is none, adds the routing rule to `AGENTS.md`, scaffolds the wiki bundle and indexes it (skip with `--no-reindex`).
### Changed
- `loomux brain` commands and `loomux serve` read the artefacts of read-only areas and the reconcile stamp from loomux's state directory first and fall back to ultra-brain's directory while nothing lies there.

## [2.4.0] - 2026-09-22

<https://github.com/xidus90/loomux/pull/15>

### Added
- Antigravity host adapter in internal/hosts reading conversationId payloads
- Normalized hook payload fixtures for Antigravity stop, pre-invocation, pre-tool, and post-tool events

## [2.3.0] - 2026-09-22

<https://github.com/xidus90/loomux/pull/14>

### Added
- `loomux hook stop`: a stop gate that runs the `stop` profile at every turn end and holds the turn while a lane is red.
- `loomux hook subagent-start` and `loomux hook subagent-stop`: report pushes, branch moves and new commits a subagent made to the main agent.
- A `stop` profile in `[verify.profiles]`, defaulting to lint, types, test and coverage.
- The lane `lint/wiki` in `loomux check` and in the stop gate, checking the wiki bundle's structure.
### Changed
- The no-verify marker is `.loomux/no-verify` instead of `.claude/.no-verify`.
- `loomux hook session-start` keeps an existing session base on resume and compact instead of moving it to HEAD.
- `loomux hook status` reports each hook event loomux serves and flags the `ultraloom hook stop` and subagent hooks as superseded.

## [2.2.0] - 2026-09-20

<https://github.com/xidus90/loomux/pull/13>

### Added
- `[commit]` in `.loomux/config.toml`: `language` (`en` or `de`), `threshold`, `conventional`, and `[[commit.allow]]` rules of `regex` and `reason` that skip a line.
- `loomux check commit-msg --calibrate N` prints, for the thresholds 1 to 4, how many of the last N commits each would have refused and which ones.
- `loomux check commit-msg --language en|de` picks the language to calibrate against; it is refused when a message file is checked.

### Changed
- `loomux check commit-msg` reads every line of a message instead of the subject alone, and a line is refused at two hits instead of one. Code spans, quotes, paths, git trailers, the scissors line and `[[commit.allow]]` matches are exempt.
- A refusal now names each refused line with its line number and the words that count against it, and exits 1 as before.

## [2.1.0] - 2026-09-19

<https://github.com/xidus90/loomux/pull/10>

### Added
- `loomux check <profile|kinds|all>` with `--root`, `-v` and `--show`, and the profiles `edit`, `precommit` and `all`.
- `[verify]` configuration per stack: presets for twelve languages, overrides as a string, a list, a table or `false`, test detection, and one lane per project area.
- `loomux check gocover --profile <file>` with `--floor N` and `--dir D`.
- `loomux hook post-tool-use --budget <duration>` (default 50 s).
- A lane can name files it `needs`; the C++ lanes that read the build tree need `build/CMakeCache.txt` and are skipped on an edit (and reported as not ready by `check`) until the build is configured.
### Changed
- post-edit takes its lanes from `[verify]` and the presets instead of a fixed list; a Go edit checks the formatting of the edited file only, a C++ edit checks formatting without rewriting the file, and TypeScript lanes call the tools through `npx`.
- `loomux status` lists the lanes post-edit actually runs.
- A post-edit lane stopped by the edit budget is reported and skipped instead of holding the edit until the hook times out.
### Removed
- `loomux dev covergate`; use `loomux check gocover`.

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
