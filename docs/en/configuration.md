# Loomux Configuration Reference

This document provides a comprehensive reference for `.loomux/config.toml`, the primary configuration file for Loomux projects.

---

## 1. Principles of Configuration

1. **Human-Maintained & Agent-Guarded**:
   > [!IMPORTANT]
   > `.loomux/config.toml` is **never modified by an AI agent**. The write barrier strictly forbids agent writes to `.loomux/config.toml`. Propose changes; a human commits them.
2. **Deterministic & Strict**:
   All regular expressions and path globs are compiled on first use. If any rule contains an invalid regex or missing reason, Loomux refuses startup immediately with a clear error naming the exact line.
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
| `name` | string | Project identifier used for scoped collections and registries. |
| `version` | string | Optional project version string. |
| `agents` | array of strings | Active harness targets wired by `loomux init`. |

---

### `[policy.paths]` (Path Protection Rules)
Defines path patterns that AI agents are forbidden from writing to or editing.

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
| `rules[].match` | string or array of strings | Glob patterns supporting `**` (e.g., `.aws/**`, `*.key`). |
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

[verify.profiles]       # built in: edit = [lint, types], precommit = all four
edit      = ["lint", "types"]
precommit = ["lint", "types", "test", "coverage"]

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
| `timeout` | positive integer | Seconds each command may run. Default 600; no upper limit. The post-edit hook also has a budget for the whole run (`--budget`, default 50 s); `loomux check` has none. |
| `profiles.<name>` | list of kinds | A named set of kinds. `edit` (post-edit) and `precommit` (the gate) are built in and can be overridden. An empty list, an unknown kind or a reserved name is a load error. |
| `<stack>.<kind>` | string, list, `false` or table | How one kind runs for one stack; see below. |
| `gdscript.import_check` | boolean | Default `true`: `test` and `coverage` of GDScript are `unready` until the Godot editor has imported the project (`.godot/global_script_class_cache.cfg`). |

Every other key is a load error that names the file and the key, on every
level: `[verify]`, `[verify.<stack>]` and `[verify.<stack>.<kind>]`. So is
the old top-level form `[verify].types = "…"`; the message points to
`[verify.<stack>].types`.
Commit message rules are not part of `[verify]`: `[verify.commit]` is refused
like any other unknown stack, and the `[commit]` section arrives with stage 2b.

#### Kinds, profiles and reserved names

There are four kinds, in the order they are listed: `lint`, `types`, `test`,
`coverage`. A request to `loomux check` is a profile, `all` (always all four
kinds) or a comma list of kinds (`lint,types`). The names `gofmt`,
`commit-msg`, `gocover`, `all`, `lint`, `types`, `test` and `coverage` are
reserved; a profile with one of them is a load error.

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
- `wiki` knows only `lint = false`, which switches off the in-process wiki
  lint of the post-edit hook. `loomux wiki-gate` stays a command of its own.
- `project` has no preset. Its lanes run in the root for `loomux check`; the
  post-edit hook runs them only with `on_file`, and only beside the lanes of
  an edited file whose stack is active.

#### The forms of a kind

A stack table knows the keys `lint`, `types`, `test`, `coverage` (and
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
| `not-applicable` | the kind is not defined for the stack, or `false` | neutral, shown | not shown |

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
  `stderr`; lanes it skipped are named in `hookSpecificOutput.additionalContext`
  on `stdout`, exit 0. A file whose ending no active stack claims gets no
  lanes and exits 0.

---

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

- **The languages a build parses follow from the extractor, not from a list a
  repository declares.** `internal/code/extract/golang` is a Go extractor; it
  parses `.go` files because that is what it knows how to read. A second
  extractor for another language adds itself the same way — by existing — and
  no config key decides which one runs on a given file.
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

### `[skills]` (Curated Best-Practice Suites)
Configures language-specific review skills and synchronization destinations.

```toml
[skills]
suites = ["review-go", "review-security", "review-typescript"]
sync = [".claude/skills", ".agents/skills"]
```

| Field | Type | Description |
|---|---|---|
| `suites` | array of strings | Active review skill suites bundled in the Loomux binary. |
| `sync` | array of strings | Directories where `SKILL.md` bundles are provisioned. |

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
  { match = [".claude/.no-verify"], reason = "Verification controls cannot be bypassed by agents" },
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

# --- Worktree Isolation Mirrors ----------------------------------------------
[worktree]
mirror = [
  "node_modules",
  ".cache",
  "bin"
]

# --- Curated Language Review Skills ------------------------------------------
[skills]
suites = ["review-go", "review-security"]
sync = [".claude/skills", ".agents/skills"]

# --- Privacy Boundaries ------------------------------------------------------
[privacy]
mode = "manual_cloud"
never = [".env*", "*.key", "credentials.json"]
```

---

## 4. Where Things Live

| | Location |
|---|---|
| State directory | on Windows `%LOCALAPPDATA%\loomux`, else `~\AppData\Local\loomux`; elsewhere `$XDG_STATE_HOME/loomux`, else `~/.local/state/loomux` |
| Area registry | `<state directory>\registry.toml` |
| Manifest of a writable area | `<area path>\.loomux\config.toml` |
| Manifest of a read-only area, as the write barrier reads it | `<state directory>\areas\<scope>\.loomux\config.toml` |
| Artefacts of a read-only area (`index.md`, `graph.json`, `_identities.tsv`) and its manifest, as `loomux brain` reads them | `%LOCALAPPDATA%\brain\areas\<scope>\` until stage 3 |
| Artefacts of a writable area | its `path` |
| Last reconcile stamp | `%LOCALAPPDATA%\brain\maintenance\last-run.txt` until stage 3 |

`LOOMUX_STATE_DIR` overrides the state directory and
`LOOMUX_LEGACY_BRAIN_DIR` the ultra-brain directory; there is no command-line
flag for either. The legacy directory exists because ultra-brain still writes
those artefacts: loomux has no indexer of its own before stage 3.

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

A refusal that reads `lies outside every writable tree` lists where writing is
allowed: the permitted trees, then the session scratchpad and the memory trees,
wherever they can be named.
