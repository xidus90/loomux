# Changelog

All notable changes to loomux are listed here, newest first. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions
follow [Semantic Versioning](https://semver.org/).

## [5.2.0] - 2026-09-28

<https://github.com/xidus90/loomux/pull/58>

### Added
- The stop gate runs the `graph` lane by default: at a turn end the blast audit checks the whole working tree against `HEAD` and holds the turn when a widely called function changed without a changed test.
- `loomux check stop` judges the `graph` lane the way the stop gate does.
### Fixed
- A graph rebuild lock left by a process that is no longer running is broken at once instead of blocking graph refreshes for up to an hour.
- `loomux config get` and `loomux config list` show `graph` in the default `precommit` profile.

## [5.1.0] - 2026-09-28

<https://github.com/xidus90/loomux/pull/55>

### Added
- `loomux convert` converts the PDFs (through Poppler's `pdftotext`) and transcripts in every writable area's inbox, or one named file, into Markdown with a provenance head, and reports what it skipped.
- `loomux convert` asks the local model, when it is enabled, for one sentence for the head and for a suggested target area.
- `loomux fetch <url>` puts a video's subtitles into an area's inbox through `yt-dlp`, and refuses a URL that names only a playlist.
- Releases ship `NOTICE.md` with the licenses of every third-party piece in the binary; `loomux dev notices` writes it.
- `loomux dev record-poppler` records Poppler's output for the test PDFs.
- The guard refuses `loomux convert` and `loomux fetch` to an agent, and `[modules] brain = false` refuses both commands.

## [5.0.0] - 2026-09-28

<https://github.com/xidus90/loomux/pull/56>

### Changed
- `loomux self-update` is renamed to `loomux upgrade`; the old name is gone.
- The session-start warning about a failed update of the machine-wide binary now says "updating loomux failed at …" and names `loomux upgrade` to retry.

## [4.2.3] - 2026-09-28

<https://github.com/xidus90/loomux/pull/54>

### Fixed
- `loomux dev bench search` without `--scope` measures the areas its question set points at instead of only `knowledge`, and refuses an expected page that lies in no registered area.

## [4.2.2] - 2026-09-28

<https://github.com/xidus90/loomux/pull/53>

### Security
- Antigravity: `loomux hook pre-tool-use` judges every argument name of a command tool regardless of case, judges a line a `manage_task` call carries whatever its `Action` says, and refuses a value under such a name that is no string.
- Antigravity: what an agent types into a task with `manage_task` or `send_command_input` passes only as whole lines without control characters, and a line ending in a backslash or a backtick, which bash and PowerShell continue on the next line, is refused, so a command split across calls or lines, edited with a backspace or completed at a tab no longer slips past the command rules. A single keystroke without Enter, Ctrl-C and arrow keys can no longer be sent to a task; `kill` still ends one.
- `loomux init` appends a block for the tools an own hook entry under an older matcher lacks, `manage_task` for Antigravity and `MultiEdit` for a Claude Code entry of ours under ulinit's matcher, so an upgrade guards them without a hand edit.
- `loomux hook pre-tool-use` refuses a call that names no tool instead of letting it pass.
### Fixed
- `loomux hook post-tool-use` names every lane it skipped, and every file of a call the shared budget did not reach, on stderr as well, so the model hears them when the edit is blocked. This corrects the v2.14.2 entry: the notices of an edit naming several files reached no host until now, because Antigravity does not read post-tool-use's stdout and Claude Code's edits name one file; they now arrive on stderr.
- `loomux hook post-tool-use` writes nothing to stdout unless it exits 0, so the callers of a green file no longer follow a red file of the same call, and with `--host claude` its context keeps `<`, `>` and `&` as they are. With `--host codex` it ends with exit 1 once the call names a file, as the Codex seam does elsewhere, instead of answering in Claude Code's shape.
- Antigravity: `loomux hook session-start` treats only the first model call of a conversation as its start. agy counts `invocationNum` from 0, so the second call was taken for a first one and repeated the binary and update warnings; and a session is revived only at the first call, so a marker it cannot remove is no longer repeated before every later one.
- Antigravity: `invocationNum` is also read as a decimal string, protojson's spelling of a 64-bit integer.

## [4.2.1] - 2026-09-28

<https://github.com/xidus90/loomux/pull/52>

### Fixed
- A `fast` search across several areas returned the best hit of each area in the order the areas were named, whatever its similarity; it now ranks all areas together.

## [4.2.0] - 2026-09-28

<https://github.com/xidus90/loomux/pull/50>

### Added
- `loomux flow run`, `resume`, `replay`, `show` and `list` start a flow, answer a paused run's gate, replay a finished run, show a run or a flow's graph, and list every flow a project can name.
- A catalog of flows shipped with the binary, with an `example` flow; a project can overlay a bundled flow's instructions and questions when `[flow] overrides` names it.
- `[agent]` configuration keys bind a flow's roles to models (`agent.roles.<role>`, `agent.models.<name>.provider|model`, `agent.default`), and `[flow]` keys set the default flow and the overrides; `loomux config get|set|unset` and the interactive form reach them.
- Session start names every flow run waiting at a gate, with the command a human answers it with.
- The guard refuses an agent's answer to a flow gate and its writes to run files and to bundled flow folders.

### Fixed
- The guard's built-in path rules match a path in any case, so `.ENV`, `GO.SUM` or `.LOOMUX/No-Verify` are protected on Windows and macOS too.

## [4.1.0] - 2026-09-27

<https://github.com/xidus90/loomux/pull/48>

### Added
- `loomux config set --global search.backbone cuda|vulkan|cpu` chooses the compute backbone qmd runs on, for the search daemon and the qmd command line; `QMD_LLAMA_GPU` and `QMD_FORCE_CPU` still take precedence.
### Changed
- `dev bench search` reports now name the qmd backbone for everyday runs that started the search daemon.

## [4.0.0] - 2026-09-27

<https://github.com/xidus90/loomux/pull/47>

### Added
- `loomux dev bench search` measures the rank of search hits and the search chain's latency over a question set or the checked-in corpus `v1`.
- `dev bench hooks` and `dev bench repos` write a markdown and a JSON report into a directory with `--out <dir>`.
### Changed
- **Breaking:** `loomux dev bench-hooks` is now `loomux dev bench hooks` and `loomux dev bench` is now `loomux dev bench repos`; `loomux dev bench` alone prints the group's help.
- **Breaking:** `dev bench repos` takes `--out <dir>` instead of `--out <file>` and `--json-out <file>`, and its JSON report and `docs/benchmarks.json` carry milliseconds instead of nanoseconds.
### Fixed
- `dev bench repos --timeout` now limits the time spent on each repository; it was ignored.

## [3.3.0] - 2026-09-26

<https://github.com/xidus90/loomux/pull/44>

### Added
- The code graph reads Python: modules, classes, functions and methods, with their imports, base classes, calls through self and cls and the base classes, and constructor calls; graph ask, callers, blast and the graph MCP tools answer for Python code.
- `graph build --no-reuse` parses every file; without it, graph build reuses the extraction of unchanged files from `.loomux/state/graph/cache/extract.json`.
- graph build reports files, reused files and parse errors per language and counts extends edges; a Python file with syntax errors keeps its file node instead of failing the build.
- graph blast and check blast-audit treat Python test files as tests: test_*.py, *_test.py, tests.py, conftest.py and files under tests/ or test/.
- A graph lane for Python projects in [verify].
### Changed
- A project with several stacks runs one graph lane instead of one per stack; graph = false under any active stack switches it off.
- A graph built by an earlier loomux is rebuilt once on the next query, because the extractor stamp now names every language.
- The binary is about 12 MB larger and its start about 1–2 ms slower, the cost of the tree-sitter runtime.

## [3.2.0] - 2026-09-26

<https://github.com/xidus90/loomux/pull/45>

### Added
- `loomux init` can pull the local model into Ollama when it is missing (part `model` of the brain module, on by default for `local_only` projects or with `[model] enabled = true`), showing its progress; Ctrl+C ends only the download.

## [3.1.0] - 2026-09-26

<https://github.com/xidus90/loomux/pull/43>

### Added
- `loomux reconcile` asks a local Ollama model, on the loopback only, for a proposal on cases of `local_only` areas and keeps it only when every claim passes the evidence check; without a usable answer the case stays manual.
- `loomux config --global` lists and edits the local model's settings (`[model]`: `enabled`, `endpoint`, `name`, `temperature`, `roles`) and refuses an endpoint off the loopback; an area's `[model]` can switch the model off or narrow its roles.
- `loomux dev fake-ollama` serves a fixed answer to Ollama requests, for recording and replaying cases.
### Fixed
- `loomux approve --reject` advances the page's sources and the identity register, so the next `reconcile` no longer reopens the rejected case; it halts, like an approval, when a source changed again since the case was opened.
- `loomux reindex` and `loomux approve` share a lock per area, so a register advanced by an approval is no longer lost to a reindex running at the same time.

## [3.0.0] - 2026-09-25

<https://github.com/xidus90/loomux/pull/42>

### Added
- `loomux config set` and the interactive form take a list item in double quotes, as a TOML string: `"a,b", c` is two items and `""` an empty one.
### Changed
- `loomux config` refuses a flag its subcommand does not take with exit code 2, where it used to ignore it: `--yes` on `get`, `list`, `proposals` and `reject`, `--json` outside `list` and `proposals`, `--propose` outside `set` and `unset`, `--all` outside `apply` and `reject`.
- `loomux config` takes `--root` and `--global` before the subcommand as well as after it.
- `loomux config list` cuts a value longer than 60 characters and ends it with `…`; `get` and `--json` still give it whole.
- `loomux merge-hook status` adds `: another hook` to the path when someone else's post-merge hook stands where loomux's would go.
### Fixed
- `loomux config set`, `unset` and `apply` take `yes` as a confirmation, not only `y`.
- A stray comma in a typed list no longer adds an empty item.
- `loomux config unset` no longer leaves a section's comments behind under the section above when it removes the section's last key.
- `loomux init` no longer keeps a module on when a human takes the offered `none`; a module that runs without a part set up now is offered as `each`.
- `loomux init --yes` no longer registers an area under a scope with blanks from the directory name; blanks become `-`.
- `loomux init --yes` in a directory whose name leaves nothing (blanks alone, a volume root) uses the scope `project/root` instead of an empty segment.
- Full-screen lists no longer wrap lines of wide characters, and their columns stay aligned.
- `loomux merge-hook` tells two repositories apart on Linux whose paths differ only in case.
- `loomux init` on Linux no longer takes a state directory that differs from `$LOCALAPPDATA/loomux` only in case for the place of the installed binary.
- The guard lets an agent run `loomux config --root <dir> list` and other reads or `--propose` calls that name `--root` or `--global` before the subcommand; writes stay refused.

## [2.14.2] - 2026-09-25

<https://github.com/xidus90/loomux/pull/41>

### Security
- On Antigravity, input sent to a running command with `manage_task` is now judged by the guard's command rules; before, such a line reached an open shell unchecked. `.agents/hooks.json` written by an earlier init keeps its matcher and is named in a note: add `|manage_task` to its `PreToolUse` matcher by hand.
### Fixed
- `loomux init` plans no Antigravity entries when `LOCALAPPDATA` contains `,`, `;` or `=`; `cmd.exe` split the unquoted path there and every Antigravity hook, the guard included, failed.
- On Antigravity, a later model call now tells the session when it could not be counted again for worktree unlink; before, only the first call could say so.
- `loomux hook post-tool-use` writes a single JSON document for an edit naming several files, so the host no longer reports a hook error, and a file the shared budget did not reach is named there.
- `loomux init` recognises its entry inside a hook block that holds both a `command` and a `hooks` list, instead of adding a second entry that made the hook fire twice.

## [2.14.1] - 2026-09-25

<https://github.com/xidus90/loomux/pull/40>

### Fixed
- `loomux self-update` and serve's update pass replace a development build (`0.0.0-dev`) at `%LOCALAPPDATA%\loomux\bin\loomux.exe` with the newest release instead of skipping it on every pass.

## [2.14.0] - 2026-09-25

<https://github.com/xidus90/loomux/pull/39>

### Added
- `loomux init` sets up Antigravity: hook entries in `.agents/hooks.json` and skills in `.agents/skills/`, only where the installed loomux can serve them.
- loomux's hooks answer Antigravity's PreToolUse, PostToolUse, Stop and PreInvocation in the form agy reads, and the guard judges `run_command` and `send_command_input`.
### Changed
- `loomux init` refuses a hook file whose root is `null` instead of reading it as empty.

## [2.13.2] - 2026-09-25

<https://github.com/xidus90/loomux/pull/37>

### Fixed
- The guard no longer refuses `loomux init --dry-run` and similar exempt calls when another part of the line holds a `#` inside double quotes, such as a path.
- `loomux init` refuses to write over a file changed after the plan was made, lists a failed write under failures, and records no `installed.toml` after a failed step.
- `loomux init` no longer writes `.mcp.json` when the installed binary it calls is missing.
- `loomux init` refuses a hook file whose content is `null` instead of writing over it.
- `loomux init` makes a git hook executable wherever it writes it, including a custom `core.hooksPath` and `.git/hooks`.
- `loomux init` names an own hook entry that runs an outdated command instead of reporting it as kept.
- `loomux config set`, `unset`, `apply` and the interactive form refuse to write over a file changed after it was read.
- `loomux config` keeps every key and each line's ending in a file with mixed line endings, including a CRLF file without a final line break.
- `loomux merge-hook install` keeps the records of the hooks it wrote before a failure.
- `loomux merge-hook status` reports a hook git no longer runs after `core.hooksPath` changed as `moved`.

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
