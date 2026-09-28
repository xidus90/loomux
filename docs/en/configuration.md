# Loomux Configuration Reference

This document provides a comprehensive reference for `.loomux/config.toml`, the primary configuration file for Loomux projects.

---

## 1. Principles of Configuration

1. **Human-Maintained & Agent-Guarded**:
   > [!IMPORTANT]
   > `.loomux/config.toml` is **never modified by an AI agent**. The write barrier strictly forbids agent writes to `.loomux/config.toml`, and the guard refuses an agent the commands that write it (`loomux init`, a writing `loomux config`, `loomux area add`). Propose changes; a human writes and commits them, by hand or with `loomux config` (see the [CLI reference](cli-reference.md#10-configuration-loomux-config)).
2. **Deterministic & Strict**:
   All regular expressions and path globs are checked when the policy is read. A rule with a missing or empty `match`, a malformed glob, a missing or non-compiling `regex` or a missing `reason` is an error naming the file and the rule's number (`[[policy.commands.rules]] #2 needs a reason`), and the guard refuses the tool call instead of judging it.
3. **Separation of Config and State**:
   - `.loomux/config.toml`: Committed human-authored policies and check chains.
   - `.loomux/state/`: Ephemeral, machine-written state (session files, journal, caches). Always git-ignored.

---

## 2. Configuration Sections

### `[project]`
Top-level metadata describing the project.

```toml
[project]
name = "loomux"
version = "0.1.0"
agents = ["claude", "antigravity", "cursor"]
```

| Field | Type | Description |
|---|---|---|
| `name` | string | Not read by loomux; the scope comes from `[area] scope`. |
| `version` | string | Not read by loomux. |
| `agents` | array of strings | Not read by loomux. `loomux init` keeps the hosts it set up in `.loomux/state/answers.toml`, not here. |

---

### `[policy.paths]` (Path Protection Rules)
Defines path patterns that AI agents are forbidden from writing to or editing.

Built in, with no entry here: `.env`, `.env.*`, `*.pem`, `*.key`, `id_rsa*`, `*.p12`, `.npmrc`, `.pypirc`, `credentials.json` and `.aws/**` (secrets); `.loomux/no-verify` and `.loomux/state/hooks/**` (the stop gate's own controls); `uv.lock`, `poetry.lock`, `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, `Cargo.lock` and `go.sum` (lock files). A project's rules come on top of these and cannot remove them.

```toml
[policy.paths]
rules = [
  { match = [".env", ".env.*"], reason = "Secrets and environment files must not be written by agents" },
  { match = ["*.pem", "*.key", "id_rsa*"], reason = "Private keys and certificates are human-managed" },
  { match = [".loomux/config.toml"], reason = "The guard's own configuration is protected from agent edits" },
  { match = ["package-lock.json", "go.sum", "uv.lock"], reason = "Lockfiles are maintained by package tools, not by hand" }
]
```

| Field | Type | Description |
|---|---|---|
| `rules` | array of tables | List of path inspection rules. |
| `rules[].match` | string or array of strings | Globs relative to the project root, in `path.Match` syntax. A pattern without `/` is matched against the file name alone (`*.key`, `.env.*`); a pattern with `/` against the whole relative path, where `*` does not cross a `/`. `**` means any depth only as the ending `/**` (`.aws/**`); anywhere else it is a plain `*`. |
| `rules[].reason` | string (**Required**) | Explanatory message displayed to the agent upon refusal. |

---

### `[policy.commands]` (Command Execution Rules)
Defines shell command patterns executed in `Bash` or `PowerShell` tools that must be blocked.

Two command rules are built in and need no entry here: `git push`, and any shell line that writes `.loomux/config.toml` — a redirect into it, `sed -i`/`perl -i`, `tee`, `Set-Content`/`Add-Content`/`Out-File`, `cp`/`mv`/`Copy-Item` onto it, or removing it. Reading it (`cat`, `grep`, `Get-Content`) stays allowed. The rule reads command text, so a path held in a variable is not caught.

```toml
[policy.commands]
rules = [
  { regex = '(^|\s)git\s+push(\s|$)', reason = "Pushing commits to remote is an exclusive human decision" },
  { regex = '(^|\s)rm\s+-rf\s+/', reason = "Destructive root filesystem commands are prohibited" },
  { regex = '(^|\s)pip\s+install\s+-r', reason = "Dependencies must be managed via lockfiles and declared workflows" }
]
```

| Field | Type | Description |
|---|---|---|
| `rules` | array of tables | List of command inspection rules. |
| `rules[].regex` | string (**Required**) | Go RE2-compatible regular expression. |
| `rules[].reason` | string (**Required**) | Explanatory message displayed to the agent upon refusal. |

---

### `[verify]` (Check Chains & Quality Gates)
Says what `loomux check` and the post-edit hook run. **Without any `[verify]`
the built-in presets apply** to every stack detection finds; the section only
changes what differs. `loomux check <profile> --show` prints the table that is
in effect (see [CLI Reference](cli-reference.md#loomux-check-request---root-path---show--v)).

```toml
[verify]
max_parallel = 8        # processes at once; default: the number of CPUs
timeout      = 600      # seconds per command; default 600, no upper limit

[verify.profiles]       # built in: edit = [lint, types], precommit = all five, stop = all five
edit      = ["lint", "types"]
precommit = ["lint", "types", "test", "coverage", "graph"]
stop      = ["lint", "types", "test", "coverage", "graph"]   # what the stop gate runs at a turn end

[verify.go]             # per stack; stacks not named keep their preset
lint     = ["go vet ./...", "{loomux} check gofmt cmd internal"]
coverage = "{loomux} check gocover --profile {coverprofile} --floor 90"

[verify.typescript.lint]            # table form
commands = ["npx eslint ."]
on_file  = ["npx eslint --cache {file}"] # what post-edit runs; without it, commands
threaded = true

[verify.cpp]
types = false           # switch a lane off, also against a preset

[verify.project]        # project-wide, no stack, in the root
lint = "make lint"
```

| Key | Type | Description |
|---|---|---|
| `max_parallel` | positive integer | Cap on child processes running at once, across all lanes. Default: the number of CPUs. |
| `timeout` | positive integer | Seconds each command may run. Default 600; no upper limit. The post-edit hook and the stop gate also have a budget for the whole run (`--budget`, default 50 s and 270 s); `loomux check` has none. |
| `profiles.<name>` | list of kinds | A named set of kinds. `edit` (post-edit), `precommit` (the pre-commit gate) and `stop` (the stop gate at a turn end) are built in and can be overridden, but not removed. An empty list, an unknown kind or a reserved name is a load error. |
| `<stack>.<kind>` | string, list, `false` or table | How one kind runs for one stack; see below. |
| `gdscript.import_check` | boolean | Default `true`: `test` and `coverage` of GDScript are `unready` until the Godot editor has imported the project (`.godot/global_script_class_cache.cfg`). |

Every other key is a load error that names the file and the key, on every
level: `[verify]`, `[verify.<stack>]` and `[verify.<stack>.<kind>]`. So is
the old top-level form `[verify].types = "…"`; the message points to
`[verify.<stack>].types`.
Commit message rules are not part of `[verify]`: `[verify.commit]` is refused
like any other unknown stack. Commit message policy is configured in the dedicated
top-level `[commit]` table (see below).

#### Kinds, profiles and reserved names

There are five kinds, in the order they are listed: `lint`, `types`, `test`,
`coverage`, `graph`. A request to `loomux check` is a profile, `all` (always all five
kinds) or a comma list of kinds (`lint,types`). The names `gofmt`,
`commit-msg`, `gocover`, `graph-fresh`, `blast-audit`, `all`, `lint`, `types`, `test`, `coverage` and `graph` are
reserved; a profile with one of them is a load error. `stop` is not reserved:
it is a built-in profile like `edit` and `precommit`, and a project whose suite
is too slow for every turn end narrows it (`stop = ["lint", "types"]`).

#### Stacks

`[verify.<stack>]` exists for `go`, `python`, `typescript`, `vue`, `svelte`,
`css`, `html`, `gdscript`, `cpp`, `shell`, `sql`, `rust`, plus `wiki` and
`project`. Any other name is a load error.

- **Where a stack runs.** Detection reads the root and one level below it
  (`go.mod`, `pyproject.toml`, `CMakeLists.txt`, `tsconfig.json` beside
  `package.json`, `project.godot`, `Cargo.toml`, `*.sh`, …). Each directory in
  which a stack was found is an **area**: `.` for the root, else the
  top-level directory. Two areas give two lanes (`lint/typescript@admin`,
  `lint/typescript@web`), each run in its area. There is no override per area.
- **A configured stack counts as detected.** A `[verify.<stack>]` that gives
  at least one kind a command runs in the root, even where detection found
  nothing.
- `.js` and `.jsx` belong to `typescript`; there is no `javascript`. Godot
  code is `gdscript`.
- `wiki` knows only `lint = false`, which switches off both wiki lanes that
  run in loomux's own process: the lint of the edited page in the post-edit
  hook, and `lint/wiki`, the lint over the whole bundle that `loomux check`
  and the stop gate add wherever `lint` is requested and the project has a
  wiki. `lint/wiki` checks the bundle's structure only and is reported after
  every other lane. The drift rule — code changed, documentation not — stays
  with `loomux wiki-gate`, a command of its own.
- `project` has no preset. Its lanes run in the root for `loomux check`; the
  post-edit hook runs them only with `on_file`, and only beside the lanes of
  an edited file whose stack is active.

#### The forms of a kind

A stack table knows the keys `lint`, `types`, `test`, `coverage`, `graph` (and
`import_check` in `[verify.gdscript]`). Each kind is one of:

| Form | Example | Meaning |
|---|---|---|
| string | `lint = "make lint"` | one command; **replaces the whole lane** |
| list | `lint = ["go vet ./...", "{loomux} check gofmt ."]` | several commands; **replaces the whole lane** |
| `false` | `types = false` | switches the lane off, also against a preset |
| table | `[verify.go.test]` with `measuring = "…"` | **merges key by key** onto the preset |

The keys of the table form:

| Key | Type | Description |
|---|---|---|
| `commands` | list | What `loomux check` runs. |
| `on_file` | list | What the post-edit hook runs for one file; without it the hook runs `commands`. |
| `threaded` | boolean | Run the commands side by side instead of one after another. |
| `measuring` | string | `test`/`coverage` only. The one command `test` runs instead of `commands` when `coverage` is in the same run. |
| `measure` | string | `test`/`coverage` only. The one command `coverage` runs first when its predecessor is not in the run. |
| `after` | kind | `test`/`coverage` only. The kind of the same stack this lane waits for. Cycles are load errors that name the ring. |
| `needs` | list | Files, relative to the lane's directory and inside it, the lane's `commands` cannot mean anything without. A missing one makes the lane `unready`: skipped and named in an edit, red in a check. They do not guard `on_file`: an edit that runs a lane's form for one file runs it either way. |

- **Replace or merge.** A string or a list stands for the lane as written:
  `measuring`, `measure`, `on_file` and `needs` of the preset no longer
  apply; only `after` stays. A table changes only the keys it names:
  `[verify.go.test] measuring = "…"` keeps the preset's `commands`.
- A key is either a value or a table in TOML: `test = "…"` and
  `test.measuring = "…"` in the same table are invalid. Use the table form with
  `commands`.
- `true`, an empty list, an empty command and an unclosed quote are load
  errors.
- **Commands are argv, not shell.** Each command is split by shell word rules
  and started directly; there is no `cmd /c` and no `sh -c`. Pipes, `&&` and
  globbing by a shell do not exist.
- Before any command starts, its tool must be on the `PATH` (also for
  configured commands); otherwise the lane is `missing-tool`.

#### Placeholders

Only these names are replaced, each inside one argument; every other brace
stays literal (`-run 'Test{A,B}'`).

| Placeholder | Value |
|---|---|
| `{file}` | The edited file, relative to the lane's area, with forward slashes. **Only in `on_file`**; anywhere else a load error. |
| `{area}` | The absolute directory of the lane's area. |
| `{coverprofile}` | `.loomux/state/cover/<run-id>-<stack>-<area>.out`, one per run and lane. |
| `{coverdata}` | The same with `.data`. Python lanes also get `COVERAGE_FILE={coverdata}` in their environment. |
| `{loomux}` | The running binary. The Go preset finds `gocover` this way even where loomux is not on the `PATH`. |

- A `coverage` lane that reads `{coverprofile}` while neither `test`,
  `test.measuring` nor `coverage.measure` of the same stack writes it is a
  load error.
- A coverage file that is missing when the lane starts makes it `failed`:
  `<path> is missing: the measuring run did not write it`.
- **Cleaning up.** `loomux check` and the post-edit hook both create
  `.loomux/state/cover/` before any lane starts and clean it the same way
  afterwards. A green run deletes its own coverage files at the end; a red
  one leaves them for inspection. Files of other runs are deleted once they are
  24 hours old, so two runs side by side never delete each other's profiles.
  A path outside `.loomux/state/cover/` is never touched.

#### Presets

The presets are embedded in the binary (`internal/verify/presets.toml`) and
use the schema above, so a preset can say nothing a project could not. The
layers are: preset, then the first **variant** whose signal detection found,
then `[verify.<stack>]`.

| Stack | `lint` | `types` | `test` | `coverage` |
|---|---|---|---|---|
| go | `go vet ./...`, `{loomux} check gofmt .` (threaded) | — | `go test ./... -count=1` | `{loomux} check gocover --profile {coverprofile}` |
| python | `uvx ruff check . --output-format=concise` | `uv run mypy --no-error-summary --no-pretty`; with `pyright`: `uv run pyright` | `uv run pytest -q --tb=short --no-header` | `uv run coverage report --skip-covered --skip-empty -m` |
| typescript | `npx eslint .`; with `biome`: `npx biome check .` | `npx tsc --noEmit` | `npx vitest run` | `npx vitest run --coverage` |
| vue | — | `npx vue-tsc --noEmit` | — | — |
| svelte | — | `npx svelte-check` | — | — |
| css | `npx stylelint **/*.{css,scss}` | — | — | — |
| html | `npx htmlhint **/*.html` | — | — | — |
| gdscript | `uvx gdlint .` | — | `godot --headless --quit` | — |
| cpp | `clang-tidy -p build` | `cmake --build build --parallel` | `ctest --test-dir build --output-on-failure` | `gcovr --root . --object-directory build --fail-under-line 100 --txt` |
| shell | only `on_file` | — | — | — |
| sql | `sqlfluff lint .` | — | — | — |
| rust | `cargo clippy -- -D warnings`, `cargo fmt --check` | — | — | — |

- The `on_file` forms: go `go vet ./...` and `{loomux} check gofmt {file}`
  (an edit formats only its own file), gdscript `uvx gdlint {file}`, cpp
  `clang-format --dry-run --Werror {file}` (checks, never rewrites),
  typescript `npx eslint --cache {file}` (biome: `npx biome check {file}`),
  css `npx stylelint {file}`, html `npx htmlhint {file}`, shell
  `shellcheck {file}`, sql `sqlfluff lint {file}`.
- The cpp `lint`, `types`, `test` and `coverage` carry `needs =
  ["build/CMakeCache.txt"]`: until the build tree is configured they are
  `unready` with `build/CMakeCache.txt is missing: configure the build first`.
  An edit still runs the lint's `clang-format` on the file, which needs no
  build tree.
  Configuring is the project's decision (generator, options, toolchain), so
  loomux does not guess a configure step.
- **Measuring.** `test` measures only when `coverage` is in the same run
  (go: `-covermode=set -coverprofile={coverprofile}`, python:
  `uv run coverage run -m pytest …`); on its own it stays the fast path.
  `coverage` runs `after = "test"`, but waits for `test` only when the test
  lane, as planned for this run, writes a file `coverage` reads: it runs its
  `measuring` form, or its command names the `{coverprofile}` or `{coverdata}`
  `coverage` reads. A Python `coverage` that reports with coverage.py
  (`coverage report`, `xml`, `json`, `html` or `lcov`) reads `{coverdata}`
  through `COVERAGE_FILE` even when its command does not name it; since every
  Python lane gets that variable, the environment says nothing about who
  writes. A `coverage` that reads no such file waits for `test` in any case.
  Otherwise, and when asked for on its own,
  `coverage` measures itself with `measure`; so `[verify.go] test = "go test
  ./..."`, which drops the preset's `measuring`, still gets its profile. A
  `coverage` without `measure` that reads a coverage file nobody in the run
  writes is red: "`test` did not run and there is no measure step", or
  "`test` does not write what this lane reads and there is no measure step".
  One that reads no coverage file just runs.
- **Thresholds live in the gate:** `--floor` for `gocover`, `fail_under` in
  `pyproject.toml` for Python, `--fail-under-line` for gcovr. Only Go checks
  per function (100 % unless `//coverage:exempt <reason>` stands above `func`).
- The Go preset measures without `-coverpkg`, because it does not know the
  module path; it under-reports tests that cross packages. A project names
  `-coverpkg` in its `test.measuring` and `coverage.measure`, as loomux does.
- GDScript has no coverage preset.

#### The `graph` kind

The fifth kind checks what a commit's change reaches in the code graph. Go and
Python have a preset for it, with the same commands:

```toml
[stack.go.graph]
commands = ["{loomux} check graph-fresh", "{loomux} check blast-audit --cached --threshold 5"]

[stack.python.graph]
commands = ["{loomux} check graph-fresh", "{loomux} check blast-audit --cached --threshold 5"]
```

`graph-fresh` rebuilds a drifted graph; `blast-audit --cached` is red when a
staged area has a seed with at least five callers (callers in tests count
too) and no changed test reaches it (see
[CLI Reference](cli-reference.md#loomux-check-blast-audit---root-path---cached----base-ref---threshold-n---skip-test-callers)).
The commands run one after the other, and `blast-audit` runs even after a red
`graph-fresh`, against the graph on disk; the lane is red then anyway.

- **One lane per run, in the root.** The graph belongs to the project root,
  not to a stack. A stack with several areas still gets one `graph/go`, not
  one rebuild per area, and of several stacks with a graph command only the
  first in byte order runs it: in a repository with Go and Python, `graph/go`
  runs and `graph/python` is `not-applicable` ("graph covered by graph/go"),
  which leaves the verdict green. A stack without a graph lane (shell, say)
  carries nothing: its lane says "no command", and the next stack with one
  runs.
- **Only in a check.** The edit scope plans no `graph` lane at all, as for a
  kind without a command: an `edit` profile that names `graph` never rebuilds
  on an edit.
- **A probe decides before the lane starts.** `loomux check` asks, in this
  order, whether `.loomux/state/graph/wiring.json` exists, whether there is a
  `HEAD` (`git rev-parse --git-path MERGE_HEAD HEAD`), whether no merge is in
  progress, and whether anything is staged (`git diff --cached --quiet`). The
  first "no" makes the lane `not-applicable` with that reason, which leaves
  the verdict green. So the lane is local by nature: it is `not-applicable`
  in CI and in every fresh clone (`.loomux/state/` is not committed), in a
  manual `loomux check precommit` with nothing staged, in a `commit --amend`
  without new changes, during a merge and at the root commit. It runs only
  where someone built the graph.
- **At the stop gate the lane judges the turn against HEAD.** `stop`
  includes `graph` by default. At a turn end the index is usually empty, so
  the stop hook writes the work through a copy of the index
  (`loomux-stop-index-<pid>` in the git directory, with `add -A`, untracked
  files included, `.loomux/state` left out) and hands only the graph lane
  that copy as `GIT_INDEX_FILE`; `blast-audit --cached` then compares
  everything git does not ignore against `HEAD`. The copy is removed after
  the chain, whatever its verdict; one left by a process killed during the
  chain is removed at the next turn end. Its probe asks whether the graph
  exists, whether there is a `HEAD`, whether no merge, rebase, cherry-pick
  or revert is in progress, and whether the work differs from `HEAD`
  ("nothing changed against HEAD"). A red lane holds the turn (exit 2) like
  any other. Without a graph no copy is kept and the lane is
  `not-applicable`. Because the lane judges against `HEAD`, a tree found
  green before counts as green again only under the same `HEAD`: after a
  commit inside the turn the chain runs again.
  The finding is headed `blast audit: index against HEAD`, and "index" is
  the copy: the printed command, run by hand against the real index, which
  is usually empty, finds nothing. `loomux check stop` builds the same copy
  and runs the same lanes; the gate around them — the marker
  `.loomux/no-verify`, a tree already found green, the subagents' findings,
  the block counter — is the hook's alone. A project that defines `stop` in
  `[verify.profiles]` itself gets the lane only once it lists `graph`.
  **Limit:** judging against `HEAD` presumes the pre-commit gate audited
  every commit. A commit made past it inside the turn (`git commit
  --no-verify`, a cherry-pick or merge, a clone without armed hooks) is part
  of `HEAD` by the turn end and is never audited.
- **Changing the threshold** means replacing `commands` in the table of the
  stack that carries the lane (`[verify.go.graph]`, in a Python-only
  repository `[verify.python.graph]`), **both** entries; a lane that names
  only `blast-audit` loses the rebuild:

  ```toml
  [verify.go.graph]
  commands = ["{loomux} check graph-fresh", "{loomux} check blast-audit --cached --threshold 10"]
  ```

- **Switching the lane off** takes `graph = false` under any one stack. The
  graph belongs to the root, so the switch is the project's: in a repository
  with Go and Python, `graph = false` under `[verify.go]` (or under
  `[verify.python]`) makes every graph lane `not-applicable` with "graph
  switched off under [verify.go]", naming the first such stack in byte
  order, and no other stack carries the graph in its place. Where no stack
  is left with a graph command, as in a Go repository with `graph = false`
  under `[verify.go]`, the lanes say "no command", as they always did. A
  switch under a stack the project does not have changes nothing. Leaving
  `graph` out of a profile (`precommit = ["lint", "types", "test",
  "coverage"]`) keeps it out of that profile only: `loomux check all` and
  `loomux check graph` still plan it.

#### Test detection

`test` and `coverage` run only where tests exist. The search runs through each
area recursively and skips every directory whose name starts with a dot
(`.git`, `.loomux`, `.venv`, `.tox`, …) as well as `vendor`, `node_modules`,
`third_party`, `venv`, `build`, `target` and `dist`: the tests of installed
packages do not make a project tested. It runs only when `test` or `coverage` is requested, so the `edit`
profile never walks the tree.

| Stack | Signal |
|---|---|
| go | a file `*_test.go` |
| python | a directory `tests/`, a file `test_*.py`, or `[tool.pytest` in `pyproject.toml` |
| cpp | `enable_testing(` in `CMakeLists.txt` |
| typescript | `vitest` in `package.json` |

- No tests: `test` and `coverage` are `unavailable`.
- A `[verify.<stack>].test` that writes the command skips the search: the
  string or list form, or a table with `commands`. Whoever writes the command
  has tests. `test = false` and a table that only changes `measuring`,
  `threaded` or another key keep the search.
- A stack without a signal (`gdscript`, `project`) runs its lane as soon as it
  is defined.

#### States and verdict

| State | When | `loomux check` | post-edit |
|---|---|---|---|
| `ok` | every command exited 0 | green | green |
| `failed` | exit ≠ 0, start error, output abandoned, coverage file missing | red | red, exit 2 |
| `timed-out` | its own `timeout` | red | red, exit 2 |
| `budget` | the run's budget was spent | — | skipped, named |
| `blocked` | the lane it waits for (`after`) is red | red | red, exit 2 |
| `missing-tool` | a tool is not on the `PATH` | red | skipped, named |
| `unready` | Godot has not imported the project, or a file in `needs` is missing | red | skipped, named |
| `unavailable` | the kind is defined but cannot run (no tests found) | neutral, counts as "nothing ran" | not shown |
| `not-applicable` | the kind is not defined for the stack, or `false`; for `graph`, the probe said no | neutral, shown | not shown |

- **What a lane inherits.** When the lane it waits for could not run, a lane
  takes over that state (`unavailable`, `not-applicable`, and in the edit scope
  also `budget`, `missing-tool`, `unready`) instead of turning `blocked`.
  A `test` that is switched off counts as not requested: `coverage` then
  measures itself with `measure`.
- **Verdict of `loomux check`, per requested kind:** if no lane of a kind ran
  and the kind is `not-applicable` nowhere, the check prints
  ``nothing to check for `<kind>` `` and exits 1: a gate that checks nothing is
  not green. Otherwise exit 0 when no lane is red, 1 when one is. A load error
  exits 1, a malformed call 2.
- **Verdict of the post-edit hook:** a red lane exits 2 with its output on
  `stderr`. A lane it skipped blocks nothing and is named on `stderr` as well,
  and at exit 0 in the host's context on `stdout`, for Claude Code
  `hookSpecificOutput.additionalContext`; under `--host antigravity` that
  `stdout` is not passed on, since whether agy reads a PostToolUse's context
  is unmeasured. A file whose ending no active stack claims gets no lanes and
  exits 0; under `--host codex` the call ends with 1 as soon as the payload
  names a file, since the Codex seam has no adapter.
- **Verdict of the stop gate:** the profile `stop` in the check scope, so the
  states read as in the `loomux check` column. A red lane exits 2 and holds the
  turn, with only the red lanes on `stderr`; a lane the budget did not reach,
  or a requested kind with nothing that ran, exits 1 — the gate could not
  judge, and the turn ends. See [Hooks](hooks.md#stop).
- **The marker `.loomux/no-verify`.** While it exists, the stop gate lets
  every turn end without running the chain. Findings of subagents are still
  delivered, and still hold the turn. A human creates and removes it; the policy refuses the path to
  an agent.

---

### `[modules]` (What Runs in This Project)

Switches the three modules of loomux on or off for one project. A missing
table or a missing key means **on**; an unknown key or a value other than
`true`/`false` makes the file count as broken, so a misspelt `graf = false`
cannot look configured and not be.

```toml
[modules]
hooks = true   # post-edit, stop, session-start, subagent-start/-stop
brain = true   # the wiki lane, the brain_* MCP tools, convert and fetch
graph = false  # the graph_* MCP tools
```

| Key | Type | Default | Off means |
|---|---|---|---|
| `hooks` | boolean | `true` | `loomux hook post-tool-use`, `stop`, `session-start`, `subagent-start` and `subagent-stop` exit 0 at once and do nothing. |
| `brain` | boolean | `true` | The lane `lint/wiki` is off everywhere — edit lane, `loomux check` and the stop gate — exactly as `[verify.wiki] lint = false` would turn it off; `loomux mcp` offers no `brain_*` tool; `loomux convert` and `loomux fetch`, run in this project, refuse with exit 1. |
| `graph` | boolean | `true` | `loomux mcp` offers no `graph_*` tool. |

- **The guard always runs.** `hook pre-tool-use` does not read `[modules]`:
  the write barrier is global and protects other repositories' read-only
  areas, so no project can switch it off.
- **The bridge reads it at start.** `loomux mcp` takes the project from
  `--root`, else from the first `.loomux/config.toml` above the directory the
  host started it in; outside any project it offers every tool. A host caches
  `tools/list`, so a change takes effect when the bridge restarts.
- **The commands stay.** `loomux brain …` and `loomux graph …` on the command
  line are not modules; `[modules]` decides what runs by itself and what an
  agent is offered.
- `loomux config set modules.graph false` writes the key; `true` removes it
  again, since a default is never written.

---

### `[commit]` (Commit Message Validation)

Configures the validation rules enforced by `loomux check commit-msg` and the `.githooks/commit-msg` hook.
If `[commit]` is omitted from `.loomux/config.toml`, standard defaults apply: `language = "en"`, `threshold = 2`, `conventional = true`.

```toml
[commit]
language     = "en"       # "en" (default) or "de"
threshold    = 2          # hits in one line that refuse it (default: 2)
conventional = true       # require Conventional Commits subject line (default: true)

# A line one of these matches is skipped whole
[[commit.allow]]
regex  = '(Müller|Zürich|Löwis)'
reason = "Proper names with umlauts: two of them in one line would refuse it"
```

| Key | Type | Default | Description |
|---|---|---|---|
| `language` | string | `"en"` | Target language for the commit message: `"en"` (English) or `"de"` (German). |
| `threshold` | integer | `2` | A line carrying this many hits or more is refused. Must be `> 0`. |
| `conventional` | boolean | `true` | When `true`, validates that the first line matches the Conventional Commits format `<type>[(<scope>)][!]: <description>`. |
| `allow` | list of tables | `[]` | Exception rules. Each entry must specify `regex` (valid RE2 pattern) and `reason` (non-empty string). A line the pattern matches is skipped whole, hits and all. |

### `[worktree]` (Worktree Mirrors)
Names gitignored directories of the main checkout that `loomux worktree link` makes available in a linked git worktree through a Windows junction. Always read from the main checkout's `.loomux/config.toml`. Junctions exist only on Windows; there are no symlinks on other systems. The mechanism is described in [Hooks](hooks.md#9-worktree-mirroring).

```toml
[worktree]
mirror = [
  "node_modules",
  ".cache",
  "vendor",
  "testdata/large-fixtures"
]
```

| Field | Type | Description |
|---|---|---|
| `mirror` | array of strings | Paths relative to the project root, mirrored so that a worktree does not have to rebuild or re-download them. An entry that is empty, absolute or leaves the project with `..` makes the file count as broken (exit 1). No table, no key or an empty list means nothing to mirror. |

---

### `[graph]` — there is none, and stage G2a settled why

An earlier draft of this manual specified a `[graph]` section with `extensions`,
`exclude` and `freshness_check` fields. None of it exists in the code stage G2a
shipped, and none is planned. The question keeps coming back, so the answer is
here rather than left to be rediscovered:

- **The languages a build parses follow from the extractors, not from a list a
  repository declares.** `internal/code/extract/golang` reads `.go` files and
  `internal/code/extract/python` reads `.py` files, because that is what each
  knows how to read; a fixed list in `internal/code/extract/all` holds them,
  and the extension alone picks the one that runs on a file. A further
  language joins that list in code, and no config key decides which extractor
  runs on a given file.
- **Narrowing a single build to a subset of the tree is a flag on the
  command, not a repository setting.** `loomux graph build` and `loomux graph
  check` already take `--root`; a future narrowing flag on one invocation is
  the right shape for "index only this subtree today", because that choice
  belongs to whoever runs the command, not to the tree being indexed.
- **`freshness_check` never had a decision to make.** `internal/code/freshness`
  always compares size and mtime first and falls back to a content hash only
  when those disagree — the reference implementation's rule, not a strategy a
  project could turn off. There is nothing left to configure.

What narrows or shapes one build belongs to the invocation that runs it, not to
the configuration of the repository being indexed. If this section returns, it
is because a real requirement forced a section-level knob to exist, not because
the manual once sketched one.

---

### The Area Declaration: `[area]`, `[layout]`, `[wiki]`, `[index]`, `[maintenance]`, `[model]`

These tables, with `[privacy]` below, declare a knowledge area; the brain
module reads them. A file without `[area]` declares no area. A value of the
wrong type is an error that names the table and the key.

```toml
[area]
scope = "project/loomux"

[layout]
wiki = "wiki"

[wiki]
untouched_days = 180

[maintenance]
on_merge = true
branch = "master"

[model]
enabled = true
roles = { describe = true, place = false, propose = true }
```

| Key | Type | Default | Description |
|---|---|---|---|
| `area.scope` | string (**Required**) | — | The scope this project is registered under, e.g. `project/loomux`. |
| `layout.wiki` | string | — | Where the wiki bundle lives, relative to the root. |
| `layout.hub` | string | — | Where the hub pages live. |
| `layout.review` | string | — | Where review cases are filed. |
| `layout.inbox` | string | — | Where files wait to be converted; an absolute path is refused. |
| `wiki.types` | array of strings | — | Page types this area declares beyond the known ones. |
| `wiki.untouched_days` | integer ≥ 1 | `180` | After how many days a page counts as untouched. |
| `index.include` | array of strings | — | Globs of the files the index reads. |
| `index.exclude` | array of strings | — | Globs the index skips. |
| `index.unsearched` | array of strings | — | Globs that are registered but never given to qmd. |
| `maintenance.on_merge` | boolean | `false` | Record merges for reconciliation. |
| `maintenance.branch` | string | `"main"` | The branch whose merges count. |
| `model.enabled` | boolean | — | `false` switches the local model off for this area. |
| `model.roles` | table of booleans | — | Which of `describe`, `place` and `propose` the model takes here; an unknown role is an error. |

`[model]` in an area can only narrow the machine-wide settings (see
[below](#machine-wide-settings-configtoml-in-the-state-directory)): `enabled`
switches the model off, never on, and `roles` keeps only the roles both
files switch on.

---

### `[privacy]` (Data Protection & Boundaries)
Guarantees sensitive data never leaks to external models or remote telemetry.

```toml
[privacy]
mode = "manual_cloud"
never = [
  "**/.env*",
  "**/credentials.json",
  "**/*.pem",
  "**/*.key"
]
```

| Field | Type | Description |
|---|---|---|
| `mode` | string | `"local_only"`, `"manual_cloud"` (default) or `"automatic_cloud"`; any other value is refused. A `local_only` area does not exist on the cloud channel. |
| `never` | array of strings | Glob patterns of paths that no channel reaches. |

---

### `[agent]` (Models for Flow Roles)

Binds the roles a [flow](flows.md#3-roles-and-models) names to models. A flow
never names a model itself, so a bundled flow loads in every project; the
project decides who answers each role. Every `loomux flow` command that
loads a flow reads the table first.

```toml
[agent]
default     = "sonnet"     # the model of every role without a binding
mcp_servers = ["loomux"]   # what the mcp tool profile may use

[agent.models.sonnet]
provider = "claude"
model    = "sonnet"

[agent.models.gemini]
provider = "agy"           # no model: the provider CLI's own default

[agent.roles]
reviewer = "gemini"
writer   = "sonnet"
```

| Key | Type | Meaning |
|---|---|---|
| `[agent] default` | string | The model name every role without a binding runs on, and an agent node without a role. Must be a name under `[agent.models]`. |
| `[agent] mcp_servers` | array of strings | The servers a node with the tool profile `mcp` may use, one `mcp__<server>` each; every entry a non-empty string. Default `[]`. |
| `[agent.models.<name>] provider` | string, **required** | Who answers for this model name, e.g. `claude` or `agy`. Not yet checked against a list: that comes with the model adapters. |
| `[agent.models.<name>] model` | string | The provider's model. Absent, the provider's CLI picks its own default; present, it may not be empty. |
| `[agent.roles] <role>` | string | The model name the role runs on. Must be a name under `[agent.models]`. |

- **Names** of models and roles follow `[A-Za-z_][A-Za-z0-9_]*`.
- **Every finding at once.** An unknown key under `[agent]` or
  `[agent.models.<name>]` is refused with the known ones, and so is a default
  or a binding that names no model: `[agent.roles] reviewer names "gemini",
  which is not under [agent.models]; known: none`. A flow therefore never
  learns at its first paid node that a role runs nowhere. `[agent] settings`
  is not known yet; it arrives with the adapters.
- **Resolution** of a node's model — node role, flow role, binding,
  `[agent] default`, the claude CLI's own default — is in
  [Flows](flows.md#3-roles-and-models); `loomux flow show <flow>` prints it
  per node.
- **With `loomux config`** the named keys are `agent.roles.<role>`,
  `agent.models.<name>.provider` and `agent.models.<name>.model`; `config
  list` shows one row per name the file holds, and `agent.roles.*` (or
  `agent.models.*.provider`, `…model`) as an unset row while it holds none. The
  model comes first: `loomux config set agent.roles.reviewer gemini` succeeds
  only once `agent.models.gemini.provider` is set, because the reader refuses
  a binding to an unknown model (exit `1`, the file unchanged). An agent
  proposes a binding with `loomux config set agent.roles.reviewer gemini
  --propose`.

**`[agent]` is not `[model]`.** `[model]` is the brain's local Ollama model,
set in the machine-wide `config.toml` (an area only switches it off or narrows
it), with its own roles `describe`, `place` and `propose` (see
[`loomux config --global`](cli-reference.md#10-configuration-loomux-config)).
`[agent]` binds the roles of a flow to the models of a provider's CLI.

---

### `[flow]` (Which Flow Runs)

```toml
[flow]
default   = "example"      # what `loomux flow run` starts without a name
overrides = ["example"]    # bundled flows a project flow of the same name may hide or overlay
```

| Key | Type | Meaning |
|---|---|---|
| `default` | string | The flow `loomux flow run` starts without a name; a flow name (`[a-z][a-z0-9-]*`). Unset, `run` without a name refuses and names the flows it knows. |
| `overrides` | array of strings | The bundled flows that `.loomux/flows/<name>/` may hide (a `flow.toml` of its own) or overlay (single `instructions/` and `questions/` files); each a flow name, none twice. Default `[]`. |

- **An unknown key** is refused with the known ones.
- **Whether a name is a flow** is not asked here: this reader reads no flows.
  A `default` that names no flow fails `loomux flow list` and a `loomux flow
  run` without a name; `list` warns about a name in `overrides` the catalog
  does not ship.
- **Only a human grants an override.** Without `overrides`, a project folder
  named like a bundled flow is ignored with a warning and the bundled flow
  runs; the guard refuses an agent every write into the folder of a bundled
  flow or of a name in `overrides` (see
  [Flows](flows.md#6-gates-are-a-humans)). An agent proposes one with
  `loomux config set flow.overrides example --propose`.

---

## 3. Complete Annotated Example (`config.toml`)

```toml
# ==============================================================================
# Loomux Project Configuration: .loomux/config.toml
# ==============================================================================

[project]
name = "loomux"
version = "0.1.0-fusion"
agents = ["claude", "antigravity", "cursor"]

# --- Path Protection Rules (Exit Code 2 on Violation) -------------------------
[policy.paths]
rules = [
  { match = [".env*", "*.pem", "*.key", "id_rsa*"], reason = "Secrets must not be written by agents" },
  { match = [".loomux/config.toml"], reason = "Guard policy is human-managed and write-protected" },
  { match = [".loomux/no-verify"], reason = "Verification controls cannot be bypassed by agents" },
  { match = ["go.sum", "package-lock.json", "uv.lock"], reason = "Lock files are written by package tools, not by hand" }
]

# --- Command Execution Rules (Exit Code 2 on Violation) ----------------------
[policy.commands]
rules = [
  { regex = '(^|\s)git\s+push(\s|$)', reason = "Whether commits reach the remote is a human decision" },
  { regex = '(^|\s)rm\s+-rf\s+/', reason = "Root filesystem removal is prohibited" }
]

# --- Check Chain: loomux check and post-edit; everything else is preset ------
[verify.go.lint]
commands = ["{loomux} check gofmt cmd internal", "go vet ./..."]

[verify.go.test]
measuring = "go test ./... -count=1 -covermode=set -coverpkg=example.com/project/... -coverprofile={coverprofile}"

[verify.go.coverage]
measure = "go test ./... -count=1 -covermode=set -coverpkg=example.com/project/... -coverprofile={coverprofile}"

# --- Modules: a missing key is on; the guard always runs ---------------------
[modules]
graph = false

# --- Worktree Isolation Mirrors ----------------------------------------------
[worktree]
mirror = [
  "node_modules",
  ".cache",
  "bin"
]

# --- Privacy Boundaries ------------------------------------------------------
[privacy]
mode = "manual_cloud"
never = [".env*", "*.key", "credentials.json"]

# --- Flows: the models of their roles, and which flow runs ------------------
[agent]
default = "sonnet"

[agent.models.sonnet]
provider = "claude"
model = "sonnet"

[agent.roles]
reviewer = "sonnet"

[flow]
default = "example"
```

---

## 4. Where Things Live

| | Location |
|---|---|
| State directory | on Windows `%LOCALAPPDATA%\loomux`, else `~\AppData\Local\loomux`; elsewhere `$XDG_STATE_HOME/loomux`, else `~/.local/state/loomux` |
| Area registry (`[[area]]` entries: `scope` and `path` required, `wiki` optional, and the booleans `readonly`, `signpost`, `shared`, `workspace`; see [Getting Started](getting-started.md)) | `<state directory>\registry.toml` |
| Machine-wide settings (local model) | `<state directory>\config.toml` |
| Single files the write barrier keeps open | `<state directory>\open.toml` |
| Manifest of a writable area | `<area path>\.loomux\config.toml` |
| Manifest of a read-only area, as the write barrier reads it | `<state directory>\areas\<scope>\.loomux\config.toml` |
| Artefacts of a read-only area (`index.md`, `graph.json`, `_identities.tsv`) and its manifest, as `loomux brain` reads them | `<state directory>\areas\<scope>\`; falls back to `%LOCALAPPDATA%\brain\areas\<scope>\` for reading until stage 4e |
| Artefacts of a writable area | its `path` |
| Last reconcile stamp | `<state directory>\maintenance\last-run.txt`; falls back to `%LOCALAPPDATA%\brain\maintenance\last-run.txt` for reading until stage 4e |
| Session state of the hooks (`base`, `blocks`, `green`) | `<project>\.loomux\state\hooks\<session_id>.json` |
| Snapshots and findings of subagents | `<project>\.loomux\state\hooks\<session_id>\agents\<agent_id>.json` |
| The marker that switches the stop gate off | `<project>\.loomux\no-verify` |
| A project's own flows and overlays | `<project>\.loomux\flows\<name>\` |
| Flow runs: journal and marker | `<project>\.loomux\state\runs\<id>.jsonl`, `<id>.flow` |

`LOOMUX_STATE_DIR` overrides the state directory and
`LOOMUX_LEGACY_BRAIN_DIR` the ultra-brain directory; there is no command-line
flag for either. Since stage 3a the legacy directory is only a read fallback
for artefacts ultra-brain wrote: loomux reads its own state directory first
and never writes here. With stage 4e a human reconciles the machine state by
hand; after that the fallback, the directory and the variable go away.

### Machine-wide settings: `config.toml` in the state directory

`<state directory>\config.toml` holds the local model's settings for every
project on the machine. A human writes it.

```toml
[model]
enabled = true
roles = { describe = true, place = true, propose = false }
```

| Key | Type | Default | Description |
|---|---|---|---|
| `model.enabled` | boolean | `false` | Let the local model be asked at all; an area can only switch it off. |
| `model.endpoint` | string | `"http://127.0.0.1:11434"` | Where Ollama listens; it must stay on the loopback. |
| `model.name` | string | `"hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q4_K_XL"` | The Ollama model that is asked. |
| `model.roles` | table of booleans | `{ describe = true, place = true, propose = true }` | Which roles the model takes; once set, an unnamed role is off. |
| `model.temperature` | float | `0.0` | The sampling temperature, between 0 and 2. |

An area's `[model]` may only narrow `enabled` and `roles`.

`<scope>` is the scope flattened into one directory name: every run of
characters other than `A-Z a-z 0-9 _ . -` becomes one `-`, and dashes at both
ends come off, so `project/loomux` becomes `project-loomux`.

**The knowledge is in none of these places.** It lies wherever an area's
`path` and `wiki` point — a repository, or an ordinary Obsidian vault under
git. loomux keeps no copy of it.

---

## 5. The Write Barrier and the Agents' Memory

The write barrier (`loomux hook pre-tool-use`) lets tools write where the
registry declares an area. A linked git worktree of an area registered with
`workspace = true` belongs to it and needs no entry of its own: it is the same
repository checked out a second time, and the barrier recognises it from git's
own worktree files. Beyond that, three places are always open, for every
user and without a registry entry:

| Where | Open below |
|---|---|
| Claude Code memory | `$CLAUDE_CONFIG_DIR/projects/<project>/memory/`, else `~/.claude/projects/<project>/memory/` |
| Claude Code session scratchpad | `<temp>/claude/<project>/<session>/scratchpad/` |
| Antigravity memory | `~/.gemini/{antigravity,antigravity-cli,antigravity-ide}/knowledge/` and `…/brain/<conversation-id>/` |

- A call that writes only there passes even when the registry cannot be read.
- `.loomux/config.toml` stays locked there too; the manifest is decided before
  anything else.
- Every path is resolved before it is compared, so a link from memory into a
  closed tree is judged at its target. In a call that also writes elsewhere, a
  memory target inside a read-only zone is still refused.
- The rest of `~/.claude` and `~/.gemini` stays shut, `antigravity-backup`
  included: `settings.json`, hooks and plugins steer the agent itself, and an
  agent that rewrites them switches off its own barriers.
- Claude Code loads `MEMORY.md` into later sessions of the same project, so
  what an agent writes there acts like an instruction to future sessions;
  whether Antigravity loads `knowledge/` the same way has not been measured.
  That was accepted knowingly with the decision of 2026-09-13.

### Single files: `open.toml`

A file a user's own conventions put outside every tree — a global `CLAUDE.md`
that has agents keep `~/.claude/AGENT_LEARNINGS.md`, say — is opened in
`<state directory>\open.toml`, which a human writes:

```toml
files = ["C:/Users/me/.claude/AGENT_LEARNINGS.md"]
```

- Each listed file is open on the terms memory is, above: also when the
  registry cannot be read, never for the manifest.
- Only single files open. An entry must be an absolute path, may not name a
  directory and may not lie in the state directory, where `registry.toml` and
  `open.toml` keep the barrier's own limits. There are no globs.
- `files` is the only key. The file is read whole or not at all: one entry the
  barrier cannot use, an unknown key or invalid TOML, and nothing in it opens.
  Every refusal then ends with `open.toml is ignored:` and the reason.
- A file of its own rather than a table in `registry.toml`, because
  `loomux area add` rewrites the registry from its `[[area]]` entries and
  would drop the table.
- No agent can write it: the state directory lies outside every tree the
  barrier opens.
- What an agent writes into such a file acts like an instruction wherever it
  is loaded; for `AGENT_LEARNINGS.md` behind a global `CLAUDE.md` that is every
  session in every project. The list is the user's to decide.

A refusal that reads `lies outside every writable tree` lists where writing is
allowed: the permitted trees, then the session scratchpad, the files
`open.toml` opens and the memory trees, wherever they can be named.
