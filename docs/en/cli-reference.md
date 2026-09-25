# Loomux CLI Reference Manual

This manual documents the syntax, flags, standard I/O contracts, and exit codes for all Loomux command-line interfaces.

---

## 1. Global Conventions

### Exit Codes
Loomux uses strict exit code semantics aligned with AI coding agent harnesses:

| Exit Code | Meaning | Behavior in Harness (Claude / Antigravity) |
|---|---|---|
| **`0`** | **Success / Allowed** | Tool execution proceeds; turn passes. |
| **`1`** | **Announcement / Warning** | Read by harnesses as "informational" or "carry-on". |
| **`2`** | **Hard Refusal / Denied** | The write barrier or policy blocked the action. Tool execution is aborted. |

Under `--host antigravity` (measured with agy 1.2.8 and 1.2.11, 2026-09-25):
`pre-tool-use`'s `2` refuses the call as in Claude Code, and
`post-tool-use`'s `2` hands stderr to the model as a warning without
aborting. A held stop becomes `{"decision":"continue","reason":"…"}` on
stdout with exit `0`, the reason being what the gate wrote to stderr; agy
then re-enters its loop. Every other non-zero code ends with `0`, its
message on stderr.

### Global Flags & Environment
- `--root <path>`: Explicit project root directory. If omitted, Loomux walks upwards from the current working directory until it locates `.loomux/config.toml`.
- `LOOMUX_STATE_DIR`: Overrides the global state directory (defaults to `%LOCALAPPDATA%\loomux` on Windows or `~/.local/state/loomux` on POSIX).
- `LOOMUX_LEGACY_BRAIN_DIR`: ultra-brain's state directory, read as the fallback for brain artefacts and never written (see section 7).

---

## 2. Policy & Verification (`loomux check`)

`loomux check` takes a **request** first and its flags after it. The request is
a profile (`edit`, `precommit`, `stop` or one of `[verify.profiles]`), `all`, a comma
list of kinds (`lint,types`), or one of the five built-in checks below. What
each kind runs per stack is set by
[`[verify]`](configuration.md#verify-check-chains--quality-gates) and the
presets.

### `loomux check <request> [--root <path>] [--show] [-v]`
Runs the lanes of the requested kinds for every active stack and area, and
judges them.

- **Flags** (after the request; `check --show` alone is a usage error):
  - `--root <path>` — project root; found upwards from the working directory
    when empty, the working directory when nothing is found. A project without
    `.loomux/config.toml` is checked by the presets alone.
  - `--show` — run nothing; print the effective lanes as `[verify]` tables.
  - `-v` — print the output of green lanes too.
- **Order**: a lane starts as soon as the lane it waits for (`after`) is done,
  with at most `max_parallel` processes at once. The report comes at the end,
  never interleaved: kinds in request order, within them stacks in byte order,
  then areas. The one exception is `lint/wiki`, the lint over the wiki bundle
  that runs in this process wherever `lint` is requested and the project has a
  wiki: it comes last. It checks the bundle's structure only; the drift rule
  stays with `loomux wiki-gate`.
- **Output** (all on `stdout`): one line per lane,
  `<kind>/<stack>[@<area>]: <state> [<origin>]`, followed by the duration for
  a lane that started, `by <lane>` for a blocked one, or the reason for one
  that never started. `<origin>` is `preset`, `preset, variant <signal>`,
  `config` or `in-process`. A red lane prints its output below its line; a
  green one only with `-v`. A lane with several commands prints one block per
  command, headed `$ <argv>`, with `(failed)` on a red one. A kind that had
  nothing to check ends the report with ``nothing to check for `<kind>` ``.
  ```text
  lint/go: ok [preset] 0.2s
  types/go: not-applicable [preset] no command
  test/go: ok [preset] 0.5s
  coverage/go: failed [preset] 0.1s
  not covered: w.go:3 A 66.7%
  ```
- **Coverage files**: measuring lanes write to `.loomux/state/cover/`, which
  `check` and the post-edit hook create themselves. A green
  run deletes its own files, a red one keeps them; other runs' files go once
  they are 24 hours old.
- **Exit Codes**: `0` (no lane red, and every requested kind had a lane that
  ran or is `not-applicable` somewhere), `1` (a lane is red, a kind had nothing
  to check, or `[verify]` or the request cannot be loaded; a load error is one
  line on `stderr`), `2` (malformed call: no request, a flag before the
  request, an unknown flag, a second request).

### `loomux check <request> --show`
Prints what the request would run, as TOML, and runs nothing.

```text
# max_parallel = 16, timeout = 600s

# go: areas ., tests found
[verify.go.lint]
commands = ["go vet ./...", "{loomux} check gofmt ."]  # preset
on_file = ["go vet ./...", "{loomux} check gofmt {file}"]  # preset
threaded = true  # preset

# [verify.go.types] not defined
```

- The first line holds the run's limits; each active stack gets a comment
  with its areas and its test state (`tests found`, `no tests found`, or
  `tests not detected` for a stack without a test signal).
- Every key carries the origin of its **lane**: a table that changes one key
  of a preset marks the whole lane `config`.
- There is no budget line: `loomux check` has no budget. The post-edit budget
  is the hook flag `--budget`.
- The output loads as it stands and can be copied into
  `.loomux/config.toml`. A lane with nothing to run is a comment,
  `# [verify.<stack>.<kind>] not defined`; a lane switched off is printed as
  `<kind> = false` under `[verify.<stack>]`, ahead of the stack's other
  tables. Pasted back, both stay as they were.

### `loomux check gocover --profile <path> [--floor <n>] [--dir <dir>]`
Judges a Go coverage profile of the module in `--dir` through
`go tool cover -func`. It is what the Go preset's `coverage` lane runs.

- **Flags**: `--profile` (required; relative to `--dir`), `--floor <n>` (a
  minimum total in percent instead of the per-function gate), `--dir`
  (directory holding `go.mod`; default `.`). The module path comes from
  `go.mod`.
- **Without `--floor`**: every function at 100 %, unless
  `//coverage:exempt <reason>` is the last comment line directly above its
  `func`. Each other function is one line on `stdout`:
  `not covered: <file>:<line> <func> <percent>%`. A profile without functions
  fails.
- **With `--floor`**: prints `coverage <total>%`, or fails with
  `coverage <total>% is below the floor of <n>%` on `stderr`.
- **Exit Codes**: `0` (passed), `1` (a function or the total below the gate, no
  `go.mod`, an unreadable profile), `2` (no `--profile`, an unknown flag).

### `loomux check commit-msg [flags] [<file>]`
Validates a git commit message file against language and formatting rules, or measures thresholds against the git history.

- **Arguments**:
  - `<file>`: Path to the commit message file (e.g., `.git/COMMIT_EDITMSG`). Cannot be passed together with `--calibrate`.
- **Flags**:
  - `--root <path>`: Path to project root (defaults to walking up to find `.loomux/config.toml` or `.git`).
  - `--calibrate <N>`: Measure refusal rates across the last `N` commits instead of checking a single file.
  - `--language <en|de>`: The language to calibrate against (defaults to `[commit].language` or `"en"`). Cannot be passed when checking a file.
- **Behavior**:
  - **Vocabulary & Language**: Scans all lines for foreign-language stop words (Variant B). In English mode (`en`), a word carrying an umlaut counts as a hit, and 82 German developer words (`fehler`, `datei`, `behebe`, `aktualisiere` …) are a fourth word source beside the German, English and Romance ones.
  - **Non-Latin Scripts**: Detects runs of non-Latin script characters (CJK ideographs, Hiragana, Katakana, Cyrillic, etc.) and treats each run as a foreign-language hit.
  - **Exemptions & Spans**: Skips text in multi-line backtick code blocks (``` `...` ```), single-line quotes (`"..."`; an apostrophe is not a delimiter), git comment lines (`#`), scissors lines (`# ------------------------ >8 ------------------------`) and trailing diffs, and git trailers (`Signed-off-by:`, `Co-authored-by:`, etc.) in the body. Also exempts file paths, name particles (`van`, `von`), hyphenated/underscored identifiers, and every line a `[[commit.allow]]` pattern matches, which is skipped whole.
  - **Conventional Commits**: When `[commit].conventional = true` (default), validates that the subject line conforms to `<type>[(<scope>)][!]: <description>`.
- **Exit Codes**:
  - `0`: Valid commit message, or calibration completed successfully.
  - `1`: Commit message refused (reason and offending lines on `stderr`), or configuration/git error.
  - `2`: Command line usage error (invalid flags or arguments).

### `loomux check gofmt [paths...]`
Inspects Go source files for formatting compliance without modifying them.

- **Arguments**: Optional directory or file paths (defaults to working directory).
- **Exit Codes**: `0` (Formatted), `1` (Unformatted files listed on `stdout`).

### `loomux check graph-fresh [--root <path>] [--wait <duration>]`
The first half of the graph lane: makes the graph on disk describe the tree
before `blast-audit` reads it.

- **Flags**: `--root <path>` (project root; found upwards when empty), `--wait
  <duration>` (how long to wait for another run's rebuild to release the lock;
  default `30s`).
- **Behavior**: drift, a missing freshness record, a graph from another
  extractor version and an outdated schema all rebuild the graph under the
  cross-process lock, and are green. A lock older than one hour is taken over.
  Progress notes go to `stderr`.
- **Output**: `graph rebuilt` or `graph is fresh` on `stdout`.
- **Exit Codes**: `0` (fresh, or rebuilt), `1` (no project root, no graph at
  all — it never builds a first one —, a failed rebuild, a failed probe, or a
  lock still held after `--wait`, named with its path and age), `2` (an
  unknown flag).

### `loomux check blast-audit [--root <path>] [--cached | --base <ref>] [--threshold <n>] [--skip-test-callers]`
The second half of the graph lane: red when a changed area with enough
callers has no changed test that reaches it.

- **Flags**: `--root <path>`; `--cached` (the index against `HEAD`, what the
  lane uses); `--base <ref>` (`<ref>...HEAD`); neither is the working tree
  against `HEAD`, or the last commit when the tree is clean, as for
  `graph blast`. `--threshold <n>` (default `3`; the Go preset passes `5`);
  `--skip-test-callers` (count only callers outside `_test.go` files).
- **The finding**: an area whose test signal is `none` or `stale` and that has
  a seed with at least `<n>` incoming walk edges. It reads the graph as it is
  and never rebuilds it; that is `graph-fresh`'s job.
- **Output**: on a finding `blast audit: <range>, threshold <n>` and one line
  per red area, `<path> [<signal>]: <seed> in-degree <k>, …`; otherwise
  `no area at or above <n> callers lacks a changed test (<m> areas)`.
- **Exit Codes**: `0` (no finding), `1` (a finding, or an error on `stderr`:
  no project root or graph, `--base` together with `--cached`, a base that
  starts with `-`, a threshold below 1, a git failure), `2` (an unknown flag).
  The exit code alone does not tell a finding from an error; `stdout` does.

---

## 3. Agent Harness Hooks (`loomux hook`)

Hook entry points are called synchronously by coding agents on tool invocations.

```bash
loomux hook <event> --host <claude|antigravity|codex> [--root <path>]
```

### `loomux hook pre-tool-use`
Evaluates the project policy and global write barrier before an agent executes a tool call.

- **Standard Input**: JSON payload emitted by the agent harness:
  ```json
  {
    "tool_name": "Write",
    "tool_input": {
      "file_path": "C:/Projects/repo/.env"
    }
  }
  ```
- **Execution Budget**: `<35ms` cold start floor.
- **Not a module**: `[modules]` is not read here. The write barrier is global
  and protects other repositories' read-only areas, so `hooks = false` leaves
  the guard running.
- **Commands that write the configuration**: a `Bash` or `PowerShell` line
  that runs `loomux init` (without an exempting `--dry-run` or
  `--detect-only`), any
  `loomux config` but `config list …`, `config get …`, `config proposals …`,
  a lone `config --help` or `config -h`, and `config set …` or
  `config unset …` with an exempting `--propose`, or
  `loomux area add`, or `loomux merge-hook install` or `remove` (`status`
  and `record` pass) is refused with ``loomux init, config and area add write
  the configuration the guard reads, and merge-hook install and remove write
  executable hooks into repositories; a human runs them. An agent proposes a
  change with `loomux config set|unset … --propose`, which a human applies``.
  `config apply` and `config reject` stay refused.
  - **When a flag exempts** — an allowlist, judged on the line as written
    before any rewriting. The flag exempts only when all three hold:
    - The call is direct: the loomux program (`loomux`, a path to it, or
      `go run` of `cmd/loomux`) is the first word of a command that the
      line itself starts — at the line's start or right after a `;`, `|`,
      `&&` or line break that stands outside quotes. A loomux inside a
      quoted string (`cmd /c 'x; loomux init --dry-run'`,
      `sh -c 'true` + line break + `loomux …'`) is still found and refused,
      but never exempt. Behind any prefix — `cmd /c`, `sudo`, `env`, `nice`,
      `timeout`, `xargs`, `exec`, `command`, `Start-Process`, `VAR=value`, a
      redirection, a keyword such as `then` or `!`, a `{` — the flag exempts
      nothing either, because a wrapper may read the words a second time:
      `cmd /c` resolves `^`, `%X%`, `!X!` and `"` even inside what the shell
      passed on as one single-quoted word.
    - The whole line is plain. Outside quotes it holds only letters,
      digits, blanks, `. _ / : = , + -`, the breaks `;` `|` `&&` and line
      breaks, the redirections `<` `>` (with `&` inside one, `2>&1`, `&>`),
      and a `#` that starts a word (a comment; the rest of its line is not
      judged). A lone `&` — PowerShell's call operator or a background job —
      is not plain. Inside double quotes only the first of these sets may
      stand, so no `$`, backtick or `\`. Inside single quotes any ASCII
      character may stand, since neither bash nor PowerShell expands
      anything there, except the breaks `; | & ( ) < >` and cmd's `^ % !`,
      which a wrapper handed the string could read again, and `#`, since a
      quoted `#x` reaches the program as a word the guard would read as a
      comment; and none beyond
      ASCII, because PowerShell also ends a single-quoted string at a
      typographic quote. An unclosed quote is not
      plain. Anything else — `$`, a backtick, `\`, `( ) { } [ ]`,
      `* ? ~ ! @`, a `#` inside a word — makes the line not plain. `%` is
      not plain either: after PowerShell's stop-parsing token `--%` the rest
      of the line goes to the program with `%X%` expanded.
    - Among the arguments, the flag (`--propose` or `-propose`; `--dry-run`
      or `--detect-only`) stands as a word of its own before any `--`, with
      no `-name=…` of the same flag beside it (the last one wins, so
      `--dry-run --dry-run=false` is refused). A `#` word ends the
      arguments. From the first redirection on, only redirections and their
      targets may follow (`> out`, `>out`, `2>&1`): `> out --propose=false`
      still hands the program `--propose=false`.

    So a value that holds `$`, `*`, `{…}`, `?`, `[…]` or a backslash must be
    single-quoted (`loomux config set index.include 'docs/**/*.md'
    --propose`), and an exempt command must stand on its own, not in a
    block (`try { … }`) or behind a program path that expands (`${X}/loomux`).

  These commands write `.loomux/config.toml` from inside
  their own process, where no path rule sees the write. The program is
  recognised as `loomux`, `loomux.exe` or a path ending in either (quoted or
  not, `\` or `/`), and as `go run` of `cmd/loomux` or `cmd/loomux/main.go`
  (with or without `./`, under a module path, at any `@version`, behind build
  flags). It is found behind `VAR=value`, redirections (`2>/dev/null`,
  `> out`, `2>&1`), the reserved words `if`, `then`, `else`, `elif`, `while`,
  `until`, `do`, `!`, `{`, `coproc`, `function <name>`, `try`, `catch` and
  `finally`, after every `{` or `}` on the line, alone or glued to a word
  (the body of a block, function or script block: `try{`, `{loomux …}`),
  and behind the wrappers `sudo`, `command`, `exec`, `nohup`, `env`, `time`,
  `xargs`, `nice` (also `nice -n N`), `timeout <duration>` and `cmd` with
  every switch up to `/c` or `/k`, together with their flags that take no
  separate value (and `--`). `Start-Process`, `start` or `saps` is refused
  when loomux is any of its arguments, also as the value of a parameter
  written with a colon (`-FilePath:loomux.exe`), whatever the others. Every
  segment of the line counts (`;`, `|`, `&`, `&&`, `||`, a line break, `(`,
  `)`, `$(`, a backtick), and a line continuation (`\` or a backtick at the
  line end) is joined first; a backtick escape inside a word
  (``loomux con`fig``) is read as PowerShell does.
  - **Known holes** — the rule reads words, not a shell, so it passes: an
    alias; a program held in a variable; a wrapper flag with a separate value
    (`sudo -u root loomux init`, `xargs -n 1 …`, `timeout -s KILL 60 …`); a
    command inside a string (`sh -c "loomux init"`, `pwsh -c …`); `go run .`
    inside `cmd/loomux`; and, after an earlier escaped `\"` or `\'` on the
    same line, a quoted program path whose part after its last break
    character (`(`, `)`, `&`, `;`, `|`) holds a blank, such as
    `echo "a \" b"; "C:\Program Files (x86)\My Tools\loomux.exe" init`.
  - **Known false refusals** — it errs toward refusing: `echo "x; loomux
    init"`, `start loomux config list`, `Start-Process code -ArgumentList
    loomux`, `command -v loomux init` (which only looks the name up), a
    loomux word right after a brace that opens no block
    (`awk '{ print }' loomux init`, `echo } loomux config set a b`,
    `echo ${X} loomux init`), `loomux init \` followed by
    `--dry-run` on the next line (PowerShell would run the first line alone),
    and `config` with any flag but `--root <dir>`, `--root=<dir>` or
    `--global` before its subcommand (`loomux config --json list`). So is a line that carries such text only
    as data, such as a heredoc holding `loomux config set …`.
- **Standard Output / Error**:
  - On Refusal: JSON refusal envelope on `stdout`, human-readable reason on `stderr`.
- **Exit Codes**:
  - `0`: Permitted.
  - `2`: Refused (policy violation or write outside registered workspace).

### `loomux hook post-tool-use`
Fires after an agent has edited a file.

- **Standard Input**: Tool name and input payload; the edited path comes from `file_path`, else `notebook_path`.
- **Flags**: `--host <h>` (required), `--root <r>`, `--budget <duration>` — how long the lanes may take in all (Go duration, default `50s`, below the host's hook timeout of 60 s). Each command gets the smaller of its own `timeout` and what is left of the budget.
- **Behavior**: Runs the lanes of the `edit` profile (by default `lint` and `types`) for the edited file's stack, in the area that holds the file, as [`[verify]`](configuration.md#verify-check-chains--quality-gates) and the presets lay them out, with `on_file` where a lane has it; see [Hooks](hooks.md#5-the-post-edit-lanes-by-stack).
- **Skipped lanes**: a lane whose tool is not on the `PATH`, a Godot project not yet imported, and every lane the budget did not reach are skipped, not failed. They are named on `stdout` as `{"hookSpecificOutput":{"additionalContext":"loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go","hookEventName":"PostToolUse"}}`.
- **Blast monitor**: after a `.go` edit with no red lane, the direct callers in other files of every symbol the edit changed or removed, measured against the graph on disk, follow the skipped lanes in the same `additionalContext`. Silent without a graph and never a finding; see [Hooks](hooks.md#the-blast-monitor).
- **Exit Codes**: `0` (all lanes passed, skipped, or nothing to run), `1` (malformed call, such as a missing `--host`, or a `[verify]` that cannot be loaded), `2` (a lane failed, timed out or is blocked; its output on `stderr`).

### `loomux hook session-start`
Records the commit the session starts on.

- **Flags**: `--host <h>` (required; `claude` and `antigravity` have adapters), `--root <r>`.
- **Behavior**:
  - Writes `HEAD` as `base` into `.loomux/state/hooks/<session_id>.json`.
  - Revives a session `worktree unlink` marked ended: removes `<session_id>.ended` and writes the file back with its row of blocks reset, so the session counts again; a session never marked ended only has its file made young. A marker it cannot remove, or a file it cannot write back, is said in the context, with exit 0.
  - Warns in `hookSpecificOutput.additionalContext` when the binary inside the project is older than its Go sources.
  - Also reads `<state dir>/update.json` and warns when, on Windows, a pass `serve` ran recorded another binary than `<state dir>/bin/loomux.exe` as its own, or when the last self-update pass failed, whoever ran it.
  - Makes no worktree junctions; that is `loomux worktree link`. See [Hooks](hooks.md#8-session-hooks).
- **Exit Codes**: `0` (Success), `1` (missing or unknown host, no adapter for the host, unreadable payload, failed write).

### `loomux hook stop`
The gate at the end of a turn: delivers what subagents left, then runs the `stop` profile over what changed since the last green run.

- **Flags**: `--host <h>` (required; `claude` and `antigravity` have adapters), `--root <r>`, `--budget <duration>` — how long the lanes may take in all (Go duration, default `270s`, below the 300 s its settings entry grants). Each command gets the smaller of its own `timeout` and what is left of the budget.
- **Standard Input**: the host's `Stop` payload; only `session_id` is read.
- **Behavior**: in this order — the subagents' findings to `stderr`, the block counter (after 3 blocks in a row it gives up for one turn and leaves the findings on disk for the next), the marker `.loomux/no-verify` (it skips the chain, not the findings), the content fingerprint (nothing new since the last green run or the base: no tool starts), then the kinds of the `stop` profile (by default `lint`, `types`, `test`, `coverage`) in the check scope, plus `lint/wiki` where `lint` is asked for and there is a wiki. A green run moves `base` to `HEAD` and remembers the tree. See [Hooks](hooks.md#stop).
- **Standard Error**: delivered findings as `subagent <agent_id>: <line>`, then only the red lanes, in the format of `loomux check`.
- **Exit Codes**: `0` (the turn ends: green, nothing new, the marker, or the counter gave up), `2` (the turn is held: a red lane, a git failure, or findings delivered — with findings even an exit 1 becomes 2), `1` (the gate could not judge: an unreadable payload or one without `session_id`, the budget ran out, a requested kind had nothing that ran, `[verify]` cannot be loaded, the plan fails, the coverage directory cannot be prepared (`verify.PrepareCover`), or a malformed call; the turn ends).

### `loomux hook subagent-start`
Writes down where `origin`, the local branches and `HEAD` stand before a subagent runs.

- **Flags**: `--host <h>` (required; `claude` and `antigravity` have adapters), `--root <r>`.
- **Standard Input**: the host's `SubagentStart` payload; `session_id` and `agent_id` are read.
- **Behavior**: `git ls-remote origin` (10 s deadline, `GIT_TERMINAL_PROMPT=0`), the local branches and `HEAD` go into `.loomux/state/hooks/<session_id>/agents/<agent_id>.json`; a remote that does not answer is recorded as `unavailable`. A finding still parked in that file is kept. See [Hooks](hooks.md#subagent-start-and-subagent-stop).
- **Exit Codes**: `0` (written), `1` (missing or unknown host, no `session_id` or `agent_id`, unreadable payload, failed write). Never 2.

### `loomux hook subagent-stop`
Compares against the snapshot and parks what moved for the main agent's `stop`.

- **Flags**: `--host <h>` (required; `claude` and `antigravity` have adapters), `--root <r>`.
- **Standard Input**: the host's `SubagentStop` payload; `session_id` and `agent_id` are read.
- **Behavior**: one line per ref of `origin` or local branch that is new, gone or moved, and one `new commit <oneline>` per commit `HEAD` and the moved branches gained, appended to the agent's file; nothing to report removes the file. Without a file or a snapshot it is silent. It writes nothing for the model: its own output would reach the subagent, not the main agent.
- **Exit Codes**: `0` (compared, or nothing to compare), `1` (missing or unknown host, no `session_id` or `agent_id`, unreadable payload, failed write). Never 2.

---

## 4. Diagnostic Doctor (`loomux status`)

### `loomux status [--root <path>]`
Inspects the active repository setup, verification lanes, detected harnesses, and hook registration.

```bash
loomux status
```
- **Aliases**: `loomux explain`, `loomux doctor`.
- **Output Details**:
  - Project root path and declared areas.
  - Active harnesses (`.claude/`, `.agents/`, `.cursor/`).
  - Write barrier status and policy rule count.
  - The lanes the post-edit hook runs per active stack: the `edit` profile as `[verify]` and the presets lay it out, each with its origin, and which of their tools are missing from the `PATH`.
  - The `Stop` entry to wire (`loomux hook stop`, profile `stop`, the wiki bundle as `lint/wiki`), and for each of the six events `PreToolUse`, `PostToolUse`, `SessionStart`, `Stop`, `SubagentStart` and `SubagentStop` whether `.claude/settings.json` calls its `loomux hook` (`[OK]`) or not (`[INFO]`), plus legacy hooks it replaces.
- **Exit Codes**: `0` (Ready), `1` (Configuration error).

---

## 5. Worktree Mirroring (`loomux worktree`)

Puts the directories named in `[worktree] mirror` into linked git worktrees as Windows junctions, and takes them out again. Junctions exist only on Windows. The full decision path is in [Hooks](hooks.md#9-worktree-mirroring).

### `loomux worktree link [--root <path>]`
In a linked worktree, makes a junction into the main checkout for every configured path that is missing there; then, wherever it runs, sweeps our junctions out of directories under `.worktrees/` and `.claude/worktrees/` that git no longer holds.

### `loomux worktree unlink [--root <path>]`
Reads `session_id` from the payload on `stdin`, marks this session ended with a `<id>.ended` file beside its state under `.loomux/state/hooks/` (the state stays, for a resume under the same id; `hook session-start` takes the marker away again), and removes the junctions only when no other session younger than 24 hours and not marked ended remains.

### `loomux worktree remove <worktree-path>`
Refuses the main checkout and any directory git holds no worktree at, removes the junctions, runs `git worktree remove --force`, checks that the directory is gone, and prints `removed <path>`.

- **Exit Codes**: `0` (in order, or nothing to do), `1` (a fault, named on `stderr`), `2` (no subcommand, or an unknown one).

---

## 6. Code Graph Engine (`loomux graph`)

> [!NOTE]
> **`build`, `check`, `ask`, `callers`, `skeleton`, `grep`, `map`, `stats` and `blast` are wired; `viz` remains specified.** Stage G1 built the packages the graph relies on — `internal/code/model`, `internal/code/pagerank` and `internal/code/blast` — stage G2a added the extractor, the wiring writer, the freshness probe and `graph build` and `check`, stage G2b added the lexicon, lexical scoring, Personalized PageRank blend and `graph ask`, stage G3 put `ask` and `check` behind the MCP tools `graph_find_code` and `graph_check_freshness` (§8), and stage G4a delivered the navigation suite (`callers`, `skeleton`, `grep`, `map`, `stats`) and their four MCP tools; stage G4b added `blast`, the MCP tool `graph_blast`, the checks `graph-fresh` and `blast-audit` (§2) and the post-edit blast monitor (§3).

### `loomux graph build [--root <path>]`
Reads and hashes every Go source file `internal/code/sourceset` finds under the root, extracts and resolves them into the deterministic AST graph, and writes it to `.loomux/state/graph/wiring.json`. It also writes the freshness record (`.loomux/state/graph/cache/fingerprint.json`) a later probe reads; a failure to write that record is announced on `stderr` but does not fail the build, since the graph on disk is already correct.

- **Flags**: `--root <path>` — project root; the working directory when empty.
- **Output**: one line naming files, nodes and edges by relation, then a line with unresolved import targets, files without a symbol, and the time taken. Illustrative shape, not a value to expect — every part of it, including the timing, moves with this repository's own code, and the last three commits each shipped a number here that the next commit falsified: `N files, N nodes, N edges (N contains, N calls, N imports)` / `N unresolved import targets, N files without a symbol, Nms`. Measured figures with their commands and raw output are in `docs/en/benchmarks.md`.
- **Exit codes**: `0` on success; `1` if the root cannot be resolved, a file cannot be read or parsed, module resolution fails, or the graph cannot be written; `2` for a usage error.
- **Cost**: `build` never reads the freshness record — it reads and hashes every file, every time, cold or warm alike. It is the command that produces the state a probe compares against, so a stale byte in it would be a stale answer, not a saved read. See `docs/en/benchmarks.md` for measured figures.

### `loomux graph ask "<query>" [flags]`
Retrieves code symbols ranked by BM25-style lexical matching blended with **Personalized PageRank** over the AST call graph.

- **Flags**:
  - `--root <path>` — project root; the working directory when empty.
  - `--limit <n>` — maximum number of hits to report (default `8`).
  - `--in <prefix>` — filter candidate nodes by path prefix before scoring and PageRank walk; recomputes document frequency over the remainder.
  - `--source` — inline the source code span for each hit (capped at 80 lines unless `--full`). Without `--source`, only locations and signatures are reported.
  - `--full` — when `--source` is enabled, inlines the full source code span without the 80-line cap.
  - `--json` — emit machine-readable JSON matching the `ask.Answer` struct (`hits`, `query`, `note`, `stats`).
  - `--no-refresh` — skip the freshness probe and automatic background rebuild on drift.
- **The two things this otherwise gets asked twice**:
  - Without `--source`, no code is shown — only location (path, line span), symbol id, signature, and composite score along with its lexical and graph components.
  - By default, `ask` probes the graph for freshness before answering. If the working tree has drifted, the freshness record is missing, or the ask sidecar is missing or carries another index version, it rebuilds the graph and sidecar under a cross-process lock before answering, logging rebuild progress to `stderr`. The sidecar is part of the probe because the freshness record knows about source files only: without that check, a deleted `ask-index.json` would leave every later question ranking on names and paths until some source file happened to change. To query the existing graph without rebuilding, pass `--no-refresh` — which skips the sidecar check as well, so the answer may fall back to names and paths, and says so on `stderr`.
  - Without a graph, exits 1 and points at `loomux graph build`; a query never builds a first graph.
- **Output**: Ranked list of hits in the format:
  `N. <id>  <path>:<span-or-line>  (<score> lex <lexical> graph <graph>)`
  followed by signature and, if `--source` is requested, the inlined code block prefixed with `|`. If no symbols match the query, outputs an empty answer note and exits 0.
- **Exit codes**: `0` on success (including when no symbols match the query); `1` on failure (no graph yet, unreadable graph, rebuild failure, syntax error in file when rebuilding); `2` on usage error (missing query, negative limit).

### `loomux graph callers <symbol> [--direction in|out] [-d <depth>] [--in <prefix>] [--json]`
Traces who calls, imports, or references a symbol (`--direction in`, default), or what this symbol calls (`--direction out`).

- **Flags**:
  - `--direction <in|out>`: Trace callers into this symbol (`in`) or callees out of this symbol (`out`).
  - `-d <depth>`: Transitive depth (default `1`; `-d all` or `-d full` for full transitive closure).
  - `--in <prefix>`: Filter symbols by repository path prefix before resolving.
  - Each direct hit (depth 1) carries the first line in the caller's span that names the callee. For `--direction out` that line lies in the start symbol's file and is printed with its path.
  - `--json`: Output machine-readable JSON (`query.CallersAnswer`).
- **Exit codes**: `0` on success; `1` if graph is missing, unreadable, or symbol not found; `2` on usage error.

### `loomux graph skeleton <file> [--json]`
Exports definition signatures, types, and line spans for a file from the graph without function bodies (~10x token reduction).

- **Flags**:
  - `--json`: Output machine-readable JSON (`skeleton.FileSkeleton`).
- **Exit codes**: `0` on success; `1` if graph is missing or file not found in graph; `2` on usage error.

### `loomux graph grep <pattern> [-i] [--fixed] [--in <prefix>] [--max-hits <n>] [--json]`
Regex search across indexed files, grouped by enclosing symbol and ranked by incoming edge degree (`inDegree`).

- **Flags**:
  - `-i`: Case-insensitive regex matching.
  - `--fixed`: Treat pattern as a literal string (no regex syntax).
  - `--in <prefix>`: Narrow search to files under path prefix.
  - `--max-hits <n>`: Maximum number of matched lines to return (default `300`); further matches are only counted.
  - `--json`: Output machine-readable JSON (`grep.Result`).
- **Exit codes**: `0` on success (even with 0 hits); `1` on missing/unreadable graph or invalid regex; `2` on usage error.

### `loomux graph map [--max-dirs <n>] [--hubs-per-dir <n>] [--hotspots <n>] [--json]`
Displays token-budgeted directory clusters, local hubs, and global codebase hotspots ranked by in-degree coupling.

- **Flags**:
  - `--max-dirs <n>`: Maximum number of directory clusters to display (default `16`).
  - `--hubs-per-dir <n>`: Maximum hubs listed per directory (default `3`).
  - `--hotspots <n>`: Maximum repository-wide hotspots (default `12`).
  - `--json`: Output machine-readable JSON (`repomap.RepoMap`).
- **Exit codes**: `0` on success; `1` on missing or unreadable graph; `2` on usage error.

### `loomux graph stats [--json]`
Prints structural codebase metrics from `.loomux/state/graph/wiring.json`: total node count, edge count grouped by relation, indexed file count, language distribution, and file size.

- **Flags**:
  - `--json`: Output machine-readable JSON (`query.StatsAnswer`).
- **Exit codes**: `0` on success; `1` on missing or unreadable graph; `2` on usage error.

### `loomux graph check [--root <path>] [--json]`
Re-extracts the whole tree and diffs it, node by node, against the graph written on disk.

- **Flags**: `--root <path>` — project root; the working directory when empty. `--json` — write the drift as JSON (`checkResult`: `ok`, `missing`, `foreign`, `added`, `removed`, `changed`) instead of the human report.
- **The one thing this otherwise gets asked twice**: `check` does not read the freshness record. That sidecar answers "should a query bother rebuilding"; `check` answers "does the graph still describe the code", and the only honest way to answer that is to extract again and compare body hashes. A `touch` that changes a file's mtime but not its bytes is therefore not a finding here, same as it is not one for the probe — but for a different reason: the probe never gets past its stat comparison, `check` gets all the way to a hash and finds it unchanged.
- **Output**: `NO GRAPH` when nothing has been built yet; `FOREIGN GRAPH` when the graph on disk names an extractor version other than this binary's; `OK` when nothing has drifted; otherwise `DRIFT` with one line per added, removed or changed node id.
- **Exit codes**: `0` — fresh (`OK`); `1` — no graph yet, a foreign graph, drift found, or a fault while re-extracting; `2` — usage error.

### `loomux graph blast [--root <path>] [--cached | --base <ref>] [-d N|all] [--no-refresh] [--json]`
Shows what a change reaches, from git's diff: per changed file the symbols its hunks touch, what reaches them, and whether a test that reaches them changed too. Deleted files close the report.

- **Flags**:
  - `--root <path>`: project root; the working directory when empty.
  - `--cached`: compare the index against `HEAD`.
  - `--base <ref>`: compare `<ref>...HEAD` instead of the working tree. Not together with `--cached`.
  - `-d`, `--depth`: depth limit, a positive integer (default `1`), or `all`/`full` for the closure.
  - `--no-refresh`: answer from the graph on disk, never rebuild.
  - `--json`: write the answer as JSON.
- **Range**: without `--cached` and `--base`, the working tree against `HEAD`, and `HEAD~1...HEAD` when the tree is clean. git runs in the project root with `--relative`, so the paths are the graph's also in a subdirectory area.
- **Output**: `blast radius: <range>`, then per changed file `<path> [<signal>]` with its seeds (`seed <name> (<kind>) <span> in-degree <n>`) and the tests that reach it; `reached:` with every hit's depth, relation and the changed files it came from; `not indexed:` files the graph does not know; `evidence:` the diff lines inside at most four seeds, six lines each; `deleted:` last. The signal is `changed` (a test file that reaches the area is in the diff too), `stale` (tests reach it, none changed), `none` (no test reaches it) or `na` (the file is a test, or no seed is a function or method); the tests always come from the full closure, whatever `-d` says.
- **Exit codes**: `0` on success; `1` without a project root or graph, a base that starts with `-`, or when git or the graph fails; `2` on usage error (an argument, `--base` with `--cached`, a bad depth).

### `loomux graph viz [dir]` *(specified, Stage W3)*
Starts the local D3-Force / WebGL interactive graph visualizer.
- **Flags**: `--port <p>`, `--no-open`.

---

## 7. Second Brain & Wiki (`loomux brain`)

The five data commands read the areas of the one registry (`registry.toml` in `LOOMUX_STATE_DIR` or its platform default) and answer as ultra-brain's `brain-mcp` does; a recorded case corpus (`testdata/cases/1b-1`) holds them to it. A read-only area's artefacts (`index.md`, `graph.json`, `_identities.tsv`) and the reconcile stamp are read from loomux's state directory first, where stage 3a writes them, and from ultra-brain's state directory as long as nothing lies in the new place: `LOOMUX_LEGACY_BRAIN_DIR`, defaulting to `%LOCALAPPDATA%\brain` on Windows and to `$XDG_STATE_HOME/brain` or `~/.local/state/brain` on POSIX. The whole area directory decides, never a single file; `loomux migrate` (stage 4) moves the rest. Until stage 4, an area directory whose `.loomux/config.toml` is missing or has no `[area]` table is read through `.ultra-brain/config.toml` or `.brain.toml`.

- **Channel**: every command takes `--channel local|cloud` (default `local`). An area with `[privacy] mode = "local_only"` does not exist on `cloud`; `[privacy] never` globs apply on every channel.
- **Usage errors** (exit `2`): the usage line, then `loomux brain <command>: error: <reason>` for a missing argument, an invalid choice or `-n` below 1, and `loomux brain: error: <reason>` when the command is missing or unknown or arguments are left over.
- **Runtime errors** (exit `1`): `error: <reason>` on `stderr` and nothing on `stdout` — an unknown scope, a refusal, a missing section, a broken `graph.json` or identity register, a missing or unreadable manifest of any registered area, a missing or broken registry, a search engine that cannot be reached.
- **Advice**: the messages still name `brain reindex`, `brain reconcile` and `brain embed`, the commands of ultra-brain, because the recorded cases of 1b-1 and 1b-2 hold that wording; they move to `loomux reindex`, `loomux reconcile` and `loomux embed` with the switch-over.

### `loomux brain search <query> [--scope <scope>] [--profile fast|full|keyword] [-n <n>] [--channel local|cloud]`
Searches the visible areas (`--scope all` by default) through the qmd MCP daemon at `http://localhost:8765/mcp`.

- **Profiles**: `fast` (default) vector search without reranking or query expansion; `keyword` BM25 keyword search; `full` the hybrid chain with expansion and reranking. `-n` (default `5`) must be at least 1.
- **Daemon**: when nothing answers there, loomux starts `qmd mcp --http --daemon --port 8765` detached, writes `note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.` on `stderr` and waits up to 60 s for it. The backbone defaults to CUDA; a `QMD_LLAMA_GPU` or `QMD_FORCE_CPU` the user set stays untouched.
- **Output**: per hit `brain://<scope>/<path>:<line>  <score>%  <title>`, each snippet line indented by four spaces, then an empty line; exactly `no matches` without hits.
- **Notes**: after the hits, `note: <finding>` lines on `stderr`, in this order — the engine answered empty twice, a hit is missing from its area's identity register, how many hits were withheld, the reconcile stamp is 24 hours old or older.
- **Exit codes**: `0` with hits or `no matches`; `1` for a runtime error, including an engine that does not answer — never an empty answer instead; `2` for a usage error.

### `loomux brain catalog [--scope <scope>] [--channel local|cloud]`
Prints the catalog of the visible areas, or of one area.

- **Output**: with `--scope all` (default) `# brain`, an empty line and `* [<scope>](brain://<scope>/)` per visible area, sorted by scope; with a named scope, that area's `index.md`, strict UTF-8 with line ends folded to `\n`.
- **Exit codes**: `0`; `1` for an unknown scope, a missing `index.md` or another runtime error; `2` for a usage error.

### `loomux brain read <path> --scope <scope> [--section <title>] [--channel local|cloud]`
Prints one file of an area, or one section of it.

- **Reading**: strict UTF-8 with line ends folded to `\n`. `--section` prints from the heading with that title to the next heading of the same or a higher level.
- **Refusals** (exit `1`): `<scope>/<path> leaves the area`; `<scope>/<path> is excluded by [privacy] never`; `<scope>/<path> is the review centre; refused on the cloud channel`; `no section titled '<title>'`.
- **Exit codes**: `0`; `1` for a refusal or another runtime error; `2` for a usage error.

### `loomux brain neighbors <path> --scope <scope> [--channel local|cloud]`
Prints the links into and out of one page, read from the area's `graph.json`.

- **Output**: `incoming: <a>, <b>` and `outgoing: <c>`, each list sorted, `-` for a direction without links.
- **Exit codes**: `0`; `1` for a refusal, an area without `graph.json` (``<scope>: never indexed; run `brain reindex` ``) or another runtime error; `2` for a usage error.

### `loomux brain status [--channel local|cloud]`
Prints what to know before trusting an answer, one line per finding.

- **Lines, in this order**: always the last reconciliation (``last reconcile: never; run `brain reconcile` ``, ``last reconcile: <iso>; older than 24 h, run `brain reconcile` `` or `last reconcile: <iso>`); per visible area in registry order, include globs the search engine does not see, a path that does not exist, an area never indexed, fewer than half of its links resolved, and indexed documents the search engine does not know; across all visible areas, the same content under several paths; once, documents indexed but not yet searchable.
- **Search engine**: two lines ask the qmd CLI (`qmd ls <collection>`, `qmd status`); when it does not answer, the line says so and the command goes on.
- **Exit codes**: `0`; `1` for a runtime error (registry, manifest, stamp, `graph.json`, identity register); `2` for a usage error.

### Wiki upkeep: `loomux brain check`, `loomux lint`, `loomux wiki`

Since stage 3c. A recorded case corpus (`testdata/cases/3c`) holds them to the reference: `brain check` to ultra-brain's Go binary, since there is no Python form, the others to `brain-mcp`. Registry and declarations come from `LOOMUX_STATE_DIR` as for the maintenance commands; until stage 4 an area's declaration is also read under `.ultra-brain/config.toml` and `.brain.toml`. None takes `--state-dir`.

#### `loomux brain check file <path> | bundle --scope <scope> | all [--notes]`
Checks pages along the `okf` axis (what a foreign reader of the Open Knowledge Format requires) and the `house` axis (the stricter house rules including the federation: `wrong-direction`, `unlisted-area`).

- **Widths**: `file` one page without neighbours; `bundle` one registered area, against the whole registry; `all` every area with a wiki. The reference's fourth width, `code`, does not exist: the code lanes belong to `loomux check`.
- **Output** on `stdout`: one line per finding, `[<severity>] <axis>/<rule> <scope>/<path>: <message>`, sorted by scope, path, axis and rule. Notes appear only with `--notes`, which may stand anywhere after the width.
- **Exit codes**: `0` checked and without error; `1` at least one error finding; `2` the run did not happen — no or an unknown width, a path that is missing or not a file, `bundle` without `--scope` or with `--scope all` (that is the width `all`), an unknown scope, a registry that cannot be read (`error: <reason>` on `stderr`).

#### `loomux lint [<file> | --file <file>] [--scope all|<scope>] [--root <path>]`
Without a file, the lint over the registered areas by the twelve rules of the reference's `lint.py`; with a file (a path that is a file, or a name ending in `.md`), the single page of stage 1a, whose rules `loomux wiki-gate`, the `lint/wiki` lane and the post-edit hook run as well.

- **Over areas**: `--scope all` (the default) every area with a wiki path, otherwise exactly one. A header line per area, below it `  <path>:<rule>: <message>` or `  no findings`; at the end `no findings` or `<n> findings (<e> errors, <w> warnings)`.
- **Rules**, in the order of the output: `broken-frontmatter`/`missing-type`, `no-sources`, `orphan`, `unlisted-area`, `dead-link`, `outside-area` (warning), `wrong-direction`, `conflict-count`, `untouched` (warning), `stale`, `implemented-without-commit`, `long-planned` (warning). The last two only in areas of the `project/` family; `unlisted-area` only in the signpost.
- **Refusals** (exit `1`, `error: <reason>`, nothing on `stdout`): an unknown scope, an area without a wiki path, a wiki path that is not a directory (``… run `loomux wiki init --scope <scope>` first``, in the run over all as well), a registry that cannot be read. A declaration that cannot be read ends the run where it stands.
- **Exit codes**: `0` without an error finding, warnings included; `1` with at least one error finding or a refusal; `2` on a usage error.

#### `loomux wiki init --scope <scope>`
Lays out the frame of an area's wiki bundle (`_schema.md`, `index.md`, `log.md`, `audit.md`, `_identities.tsv`) and names every file it wrote. An existing file stays as it is. Exit `1` for an unknown scope, an area without a wiki path, or a read-only area.

#### `loomux wiki types`
Counts the page types across every area with a wiki: per type `<type> [<rank>]: <total> (<scope>: <n>, …)`, by total descending, then by name. The rank is `core`, `catalogue`, `origin`, `declared` or `unknown`; an unknown type carries the prefix `? `, a known old name ` -> <catalogue name>`. Across areas the worst rank counts. Exit `0`, unless the registry, a declaration or a page cannot be read (`1`).

#### `loomux wiki retype --scope <scope> --from <old> --to <new>`
Renames one page type in one bundle and names every page it wrote. Only the frontmatter's `type:` line changes; a written page is folded to LF throughout. Skipped are scaffold files, broken frontmatter (a duplicate key included), bytes that are not UTF-8, and a quoted or folded value. A target type no rank knows gives a warning on `stderr`, and the run goes on. Exit `1` for an unknown scope, an area without a wiki path, or a read-only area.

### Upkeep: `loomux reindex`, `loomux embed`, `loomux reconcile`, `loomux area add`

Four commands of ultra-brain's `brain` CLI, top-level commands of loomux since stage 3a; a recorded case corpus (`testdata/cases/3a`) holds them to the Python reference. They write only to loomux's state directory and read the legacy one as the fallback described above.

- **Environment**: `LOOMUX_STATE_DIR` holds the registry, the artefacts of read-only areas, `maintenance/` and `qmd-collections.json`; `LOOMUX_LEGACY_BRAIN_DIR` is the fallback and is never written. qmd's `index.yml` is found through `XDG_CONFIG_HOME`, else `~/.config`.
- **No `--state-dir`**: the reference accepts it on all four; loomux refuses it like any unknown flag (exit `2`). The state comes from the environment, the one state model of every loomux command.
- **Positional arguments** (exit `2`): none of the four takes one. A word left after the flags is refused with `<command>: unrecognized arguments: <words>` before the environment or qmd is looked at.
- **Messages** of the reconcile pass are German, word for word the reference's.

#### `loomux reindex [--registry <path>]`
Runs a reconcile pass over the registered areas, then rebuilds each area's directory catalogs (`index.md`), link graph (`graph.json`) and identity register (`_identities.tsv`) and enters the areas as collections into qmd's `index.yml`. A writable area keeps its artefacts in its own tree; a read-only area's are written to `<state>/areas/<scope>/` through a staging directory and swapped in whole.

- **`--registry`**: a `registry.toml`, or the directory holding one; default `registry.toml` in the state directory.
- **Catch-up**: the reconcile pass runs first, so that a changed source becomes a case before the index run advances its hash. Cases it opens are listed on `stderr` and the run **goes on**; a vault without a review centre gets a warning and is indexed; any other failure of the pass stops the command before anything is indexed.
- **Output**: `indexed the areas of <path>` on `stdout`; on `stderr` the collections updated or dropped, and each collection refused because qmd already holds one of that name that brain did not create.
- **Exit codes**: `0` on success, and also when there is no registry in the state directory (`no areas registered in <path>; nothing to index` on `stdout`); `1` for a registry named with `--registry` that does not exist, a registry that does not read, a failed catch-up, a failed index run or a refused collection; `2` for a usage error.

#### `loomux embed [--registry <path>]`
Asks qmd to generate the vectors the index run leaves pending, for every registered area.

- **qmd first**: without `qmd` on `PATH` it prints `loomux embed: qmd is not on PATH; install it with: npm install -g @tobilu/qmd` and exits `1` before it reads the registry.
- **Output**: `embedded <n> area(s)` on `stderr`.
- **Exit codes**: `0` on success, and also when there is no registry in the state directory (`no areas registered in <path>; nothing to embed` on `stdout`); `1` for missing qmd, a named registry that does not exist, a registry that does not read, or an engine that refuses; `2` for a usage error.

#### `loomux reconcile`
Measures every source of the registered areas against its identity register and, for each wiki page derived from a changed source, opens a case (`case.toml` and a package with the diff) in the review centre, the one directory an area declares under `[layout] review`. A merge recorded in `maintenance/merge-events.tsv` opens a case with the merge's evidence, after the source cases.

- **Output** on `stdout`: `<n> Quellen geprüft, <m> davon gehasht`, one line per case (directory, area, target, state and `manuell` for a case that asks for a manual decision, tab-separated, indented by two spaces), then `<k> Fälle`.
- **A case is not a failure**: open cases leave the exit code at `0`.
- **Stamp**: the pass writes `maintenance/last-run.txt` in UTC, which `brain status` and `brain search` read.
- **Exit codes**: `0` for a pass that ran to the end; `1` when a case file cannot be read (`unreadable case: <entry>` on `stderr`), and for a registry that does not read, a vault that declares no review centre or two, or another failure (`error: <reason>`); `2` for a usage error.

#### `loomux area add [--path P] [--scope S] [--wiki W] [--sources S] [--merge-branch B] [--privacy M] [--no-reindex] [-y|--yes]`
Registers a repository as an area and prepares it: the registry entry (written under a lock); `.loomux/config.toml` with `[area]`, `[layout]`, `[index]`, `[privacy]` and `[maintenance]` when the repository has none; the routing rule, appended once to `AGENTS.md`; the wiki bundle frame; then `loomux reindex` with its catch-up.

- **Defaults**: `--path` the working directory; `--scope` `project/<directory name>`; `--sources` `docs` when there is a `docs/` directory, else `.`; `--wiki` `<repo>/docs/wiki` or `<repo>/wiki` (must be absolute); `--merge-branch` the branch git names, `master` without one; `--privacy` `manual_cloud`, one of `automatic_cloud`, `local_only`, `manual_cloud`.
- **Registry first**: a scope already registered is refused before the repository is touched.
- **A kept configuration**: an existing `.loomux/config.toml` is kept byte for byte, with a warning when it declares no `[area]` or another scope. One the declaration reader refuses ends the command with nothing registered.
- **Differences from `brain init`**: no `.mcp.json` and no agent hooks (`loomux init`, stage 4); the index run really happens unless `--no-reindex` is given; the branch is written as `[maintenance] branch`, not `merge_branch`; `--privacy` is checked; the first area of a machine needs no registry file prepared by hand. `-y`/`--yes` is accepted and changes nothing.
- **Exit codes**: `0`, or the exit code of the index run; `1` for a path that is not a directory, an invalid scope, a relative `--wiki`, a refused registry entry, an unreadable file or a failed write; `2` for a usage error, a missing or unknown subcommand (with the usage line) or an unknown `--privacy`.

### The post-merge hook: `loomux merge-hook install|status|remove|record`
The hook that tells `reconcile` a merge has landed, in every repository of an area whose manifest says `[maintenance] on_merge = true`. ultra-brain's `brain-mcp hook` under a new name, since `hook` is the namespace of the host hooks here; a recorded case corpus (`testdata/cases/4a2`, 14 cases, eleven without a difference) holds it to the reference. `loomux init` runs `merge-hook install` as its part `merge-hook` (off in a checkout of loomux, whose hook directory is the tracked `.githooks`).

- **The hook bakes nothing in**: the file is the same text everywhere, a short `sh` that runs `"${LOCALAPPDATA}/loomux/bin/loomux.exe" merge-hook record` and discards its output. Which repository and branch count is decided at merge time from the registry, so nothing in the file can go stale. It lands where git looks for hooks, `core.hooksPath` included. The installations are remembered in `maintenance/hooks.tsv` under `LOOMUX_STATE_DIR` (three fields, `scope`, `repo`, `hook`; a line of the reference with more is read by its first three).
- **`install`** writes the hook into each consenting repository and remembers it; a file with the reference's marker `# brain post-merge hook` is the same hook's predecessor and is replaced. A foreign `post-merge` is `refused` and left alone.
- **`status`** names each repository's state: `installed`, `missing` (remembered, file gone), `moved` (remembered, the file stands, but `core.hooksPath` changed and git looks elsewhere; `install` writes it where git looks now), `not installed` (with `: another hook` after the path when a file of someone else's stands there), `unrecorded` (the file is ours, the record is lost — also a hook `brain-mcp` set up, whose records under the old state directory are not read), `orphaned` (remembered for an area that no longer consents or a repository that moved) and `refused`; every command says `no repository` for a consenting area whose path lies in none.
- **`remove`** takes back every hook of ours it remembers, an unrecorded one included; one somebody replaced with their own is `refused` and its record kept.
- **Output** on `stdout`: one line per repository, `<state>: <scope> — <repo>`, followed by ` [<hook file>]` when there is one. Without a line: `no area consents with [maintenance] on_merge = true, and no hook is installed`.
- **`record`** is what the hook calls: one event for the merge that just landed in the working directory, if an area wants it. It runs inside the user's `git merge`, so it never prints and always exits `0` — no registry, a broken one, no repository, a state directory it cannot write and extra arguments included.
- **The guard** refuses an agent `install` and `remove`, which write executable files into repositories; `status` and `record` pass (see [`hook pre-tool-use`](#loomux-hook-pre-tool-use)).
- **Exit codes**: `0`, also for an orphan, a missing, a moved or an unrecorded hook, which are findings; `1` when a line says `refused` or `no repository`, or the registry does not read or a hook file cannot be written (`loomux merge-hook <sub>: <reason>` on `stderr`); `2` for a usage error (`usage: loomux merge-hook install|status|remove|record`).

### Review: `loomux cases`, `loomux case`, `loomux approve`

Three commands of ultra-brain's `brain` CLI, top-level commands of loomux since stage 3b; a recorded case corpus (`testdata/cases/3b`) holds them to the Python reference, `approve` together with the files it writes and the commit it makes. They decide the cases `loomux reconcile` leaves in the review centre. A case is addressed by the name of its directory in the review centre, the first column of `loomux cases`.

- **Environment**: the registry and the review centre come from `LOOMUX_STATE_DIR`, as for the upkeep commands; the review centre is the one directory an area declares under `[layout] review`.
- **No `--state-dir`**: the reference accepts it on all three; loomux refuses it like any unknown flag (exit `2`).
- **Flags and the id** may stand in any order, as with argparse. A missing id is `<command>: the following arguments are required: id`, a second word `<command>: unrecognized arguments: <words>`, both exit `2`.
- **Messages** on `stdout` are German, word for word the reference's.
- **Runtime errors** (exit `1`): `error: <reason>` on `stderr` — a registry that does not read, a vault that declares no review centre or two, an id no case answers to (`no case named '<id>' …`) or that two directories carry, a `case.toml` that does not read.

#### `loomux cases`
Lists every case standing in the review centre, one line each: directory, area, target and state, tab-separated, and `manuell` for a case that asks for a manual decision; `keine offenen Fälle` when there is none.

- **A case is not a failure**: waiting cases leave the exit code at `0`.
- **Renamed case**: when the `id` in `case.toml` and the directory name differ, a warning on `stderr` says so; the directory name counts.
- **Exit codes**: `0`; `1` when a case file cannot be read (`unreadable case: <entry>` on `stderr`, the other cases are still listed) or for a runtime error; `2` for any positional argument.

#### `loomux case <id> [--package]`
Prints what the human gate has to see before it decides: `Fall <id> (<state>, ausgelöst durch <trigger>, Gewicht <weight>)`, the area, the target, one `Quelle:` line per source with revision and hash, then `package.md`, `proposal.md` and, where one was superseded, `superseded-proposal.md`, each under a `===== <file> =====` heading, or `(nicht vorhanden: <path>)`.

- **Withheld**: a case of a `local_only` area, or of an area whose declaration cannot be read, prints the halt line instead of its files, with the place in the review centre and `Bewusst ausgeben: loomux case --package <id>`. The package carries the source diff, and a proposal quotes it verbatim.
- **`--package`**: prints the withheld files anyway. The halt line still stands above them.
- **Further lines**: `manuell: für diesen Fall wird kein Skill-Pfad angeboten` for a manual case, `Vermerk: <note>` when a refused approval left one.
- **Exit codes**: `0`; `1` for a runtime error; `2` for a usage error.

#### `loomux approve <id> [--amend <file> | --reject | --defer]`
Decides one case. Without a flag the case's own proposal is approved; `--amend` approves the named file instead; `--reject` discards the proposal; `--defer` leaves the case in the queue.

- **Evidence binding**: every claim of the proposal needs a verbatim quote from a segment of the package. A claim without one is dropped and named on `stdout` (`  verworfene Behauptung: <claim>`); when none survives, or the surviving claims carry no diff that applies, nothing is written to the page.
- **An approval** checks first that neither the target page nor a cited source changed since the case was formed, then writes the page with its advanced frontmatter (`generated`, `verified` with the reviewer), advances the identity registers, appends to `log.md` and `audit.md`, removes the case directory and commits exactly those paths onto the vault's current ref through a scratch index (`<state>/maintenance/index`); the user's own index is left alone. It then runs a catch-up pass and an index run, the index run without a catch-up of its own. When the catch-up fails, a warning says so and nothing is indexed; when the index run fails, a warning names `loomux reindex`. Neither changes the exit code.
- **A rejection** appends to `audit.md`, removes the case directory and commits both. It does **not** advance the page's revision and hash, so the next `loomux reconcile` opens the same case again — inherited from the reference.
- **`--defer`** writes nothing: `Fall <id> zurückgestellt; er bleibt unverändert in der Warteschlange.`
- **The reviewer** is `human:<account>`, the account running the command with its domain cut off. No flag names it.
- **Output**: `Fall <id>: approve` or `Fall <id>: reject`, then `committet als <sha>`. A decision written but not committed — the vault is no git repository, or a rebase or merge is in progress — is `geschrieben, aber nicht committet: <reason>` (or `entschieden, …` for a rejection) on `stderr`, with exit `0`: the vault changed, and running the command again would not improve matters.
- **Refusals** (exit `1`): a moved target, a moved source or a proposal refused by the evidence check write a note into `case.toml` and, for an outcome not recorded before, a block into `audit.md`; a refused proposal of the case itself, not an `--amend` file, also marks the case `manuell`. Then `error: <reason>`. Every file already changed is named after a `Hinweis:` line on `stderr`; none of it is committed.
- **`--amend=`** with an empty value names `.`, as Python reads `Path("")`, and is refused as a directory; it never falls back to the case's own proposal.
- **Exit codes**: `0` for a decision taken, committed or not, and for `--defer`; `1` for a refusal or a runtime error; `2` for a usage error, including two decisions at once (`loomux approve: argument --<later>: not allowed with argument --<earlier>`) and `--amend` followed by a word argparse reads as an option (`argument --amend: expected one argument`).

---

## 8. MCP Service & stdio Bridge (`loomux serve` / `loomux mcp`)

The service answers twelve tools over Streamable HTTP — the five `brain_*` tools
and the seven `graph_*` tools of Stages G3, G4a and G4b; the bridge is what an MCP host starts
and all it does is pass calls on. Both were built in Stage 1b-2. The Web OS of
Stage W1 is not here yet, and neither are the upstream proxies.

**The channel is the address, not a field of the request.** `serve` binds two
loopback listeners, one for `local` and one for `cloud`, each with its own
random token. A caller holding the cloud token cannot reach the local view at
all. Everything the service writes — `serve.json`, `serve.lock` and
`logs/serve.log` — lives under the state directory (`LOOMUX_STATE_DIR`, by
default `%LOCALAPPDATA%\loomux`), never under a hard-wired path.

### `loomux serve [--foreground]`
Starts the long-lived localhost MCP service. Without `--foreground` the start is
detached: the service is spawned beside the caller, asked to break away from the
caller's job object, and the command returns at once naming the log file. With
`--foreground` the service runs in this process, which is the only way a human
sees a failed start — and it is also what the detached child runs.

- **One instance**: `serve.lock` plus a liveness check. A second `loomux serve`
  says so instead of starting a second service.
- **Breakaway**: if the request to leave the host's job object is refused, the
  start is retried without it and the command says `note: the breakaway was
  refused, so this service dies with its host`.
- **Daily catch-up** (since stage 3c): when the last `reconcile` is older than
  24 hours, or there was none, the service runs it itself at start and every 24
  hours after that for as long as it lives; never `reindex`. Every `brain_*`
  tool waits for the first pass and says so as progress. What it found rides
  on the answers: `brain_status` the opened cases, the unreadable case files
  and a failure, the other four one line with `! ` (the failure or the number
  of cases), every one but `brain_search` also the line about an aged stamp.
  The `cloud` channel sees only its own areas and not the cause of a failure.
- **Exit codes**: `0` started; `1` already running, or the spawn or the run
  failed; `2` an unrecognized argument or an unknown subcommand.

### `loomux serve status`
Asks the local listener whether it answers — a state file is a hint, a listener
that answers is the truth — and then prints what `serve.json` says: PID, both
endpoint URLs, the executable, its size and build time, whether the lock is held
and whether the breakaway held. The tokens stay out of the report. A service
that is not there is a state and not a failure: the report says so and the exit
code is `0`.

- **Exit codes**: `0` reported; `1` the state could not be read; `2` an
  unrecognized argument.

### `loomux serve stop [--force]`
Ends the service through its own endpoint. `--force` kills it by the PID in
`serve.json` when the endpoint no longer answers. Success is silent.

- **Exit codes**: `0` stopped, and also when nothing was running; `1` the stop
  failed; `2` an unrecognized argument.

### `loomux self-update`

One self-update pass by hand; `serve` runs the same pass a minute after it
starts and every 24 hours after that. It acts only on the machine-wide
binary, `<state dir>/bin/loomux.exe`, when that is the running binary. A
development build (`0.0.0-dev`) there is replaced by the newest release; one
anywhere else is never touched.

1. Lists the releases through `gh release list` and takes the highest version
   of the running binary's channel (`beta` takes pre-releases, `stable` does
   not).
2. Downloads the Windows asset and `SHA256SUMS` through `gh release
   download`, checks the checksum and the new binary's `--version`.
3. Stamps the file with the current time and swaps it in; the old one goes to
   `loomux.old.exe` or the first free numbered slot. The next bridge replaces
   the running `serve`.

Writes `<state dir>/update.json` (`source` = `serve` | `cli`, `checked_at`,
`executable`, `running`, `result` = `current` | `updated` | `skipped` |
`failed`, `version`, `error`). A pass that finds `update.lock` held steps
aside and writes nothing. A pass by hand that skips writes nothing either,
so the record of `serve`'s last pass stays for session start to read.

| Exit | Meaning |
|---|---|
| 0 | `already current (vX)` or `updated to vX` |
| 1 | the pass failed, or another pass is running |
| 2 | skipped: not on Windows, or not the machine-wide binary (a development build included); or an unrecognized argument |

### `loomux mcp [--channel local|cloud] [--root <dir>]`
The stdio bridge an MCP host starts. It offers the twelve tools itself — the
descriptions are static, so a cold service never sits inside the host's
handshake — and forwards every `tools/call` to the service over the channel's
address, name to name and arguments to arguments.

- **Default channel**: `local`, exactly as `loomux brain` falls back. The narrow
  channel is the only safe default.
- **Only the project's modules**: the bridge offers the `brain_*` tools only
  with `[modules] brain` on and the `graph_*` tools only with `graph` on (see
  [`[modules]`](configuration.md#modules-what-runs-in-this-project)). The
  project is the one `--root` names, else the first `.loomux/config.toml`
  above the directory the host started the bridge in; outside any project
  every tool is offered. A `[modules]` the reader refuses ends the bridge with
  exit `1` before the handshake.
- **A change of `[modules]` needs a restart**: the bridge reads it once at
  start, and the host caches `tools/list`. The change takes effect when the
  host starts the bridge again.
- **It starts the service itself**: in the background, beside the handshake. A
  recorded build older than the bridge's own is stopped and replaced (newer
  wins); an equal or newer one whose lock nobody holds is started again.
- **One retry**: a call whose session died — a commit rebuilt the binary and
  another bridge replaced the service — is renegotiated once. Two retries would
  hide a real outage.
- **Nothing is ever written to stdout**: stdout is the host's MCP pipe. Refusals
  and failures go to stderr.
- **Exit codes**: `0` the host hung up, or Ctrl+C; `1` the bridge failed; `2` an
  unrecognized argument or an invalid `--channel`.

### The twelve tools

| Tool | Arguments |
|---|---|
| `brain_search` | `query` (required), `scope` → `all`, `profile` ∈ {`fast`, `full`, `keyword`} → `fast`, `n` → 10 |
| `brain_catalog` | `scope` → `all` |
| `brain_read` | `scope` and `relative` (both required), `section` |
| `brain_neighbors` | `scope` and `relative` (both required) |
| `brain_status` | none |
| `graph_find_code` | `scope` and `query` (both required), `limit` → 5, `full`, `in` |
| `graph_check_freshness` | `scope` (required) |
| `graph_file_api` | `scope` and `file_path` (both required) |
| `graph_trace_calls` | `scope` and `symbol` (both required), `direction` ∈ {`in`, `out`} → `in`, `depth` → 1 |
| `graph_find_all` | `scope` and `pattern` (both required), `ignore_case`, `fixed` |
| `graph_repo_map` | `scope` (required), `max_dirs` → 16 |
| `graph_blast` | `scope` (required), `base`, `depth` → 1 |

`n` is 10 here and 5 on the command line; that is parity with the Python
reference, which does the same, not an inconsistency. `limit` is 5 here and 8
for `loomux graph ask`, both Graft's values.

The seven `graph_*` tools operate over the repository of one registered area:

- **`graph_find_code`** always inlines the source at each hit; `full` takes
  the whole span instead of the capped excerpt, and `in` narrows to a path
  prefix before scoring. A refresh note stands before the answer. Without a
  graph the call is an error pointing at `loomux graph build` — a query never
  builds a first graph.
- **`graph_check_freshness`** never refreshes, so it reports on the graph as it
  was found. Drift and a missing graph are text, not errors. `isError` marks
  a refused call — a missing, unknown or hidden scope, or for
  `graph_find_code` a missing query — and a real read failure.
- **`graph_file_api`** exports definitions and types for `file_path` from the
  cached AST graph without function bodies.
- **`graph_trace_calls`** traces incoming callers (`direction: in`) or outgoing
  callees (`direction: out`) of `symbol`, either direct (`depth: 1`) or
  transitive (`depth: "all"`).
- **`graph_find_all`** performs symbol-coupled regex search across indexed files,
  grouped by enclosing symbol and ranked by incoming edge degree (`inDegree`).
- **`graph_repo_map`** generates a token-budgeted structural overview of directory
  clusters, hubs, and hotspots.
- **`graph_blast`** is `loomux graph blast` in the area's root: `base` compares
  `base...HEAD`, empty compares the working tree with `HEAD`, or the last commit
  when the tree is clean; `depth` as for `graph_trace_calls`. A changed file or
  hit under the `never` globs is counted, not named, and the refresh notes are
  left out on the cloud channel.

**Visibility:** an area whose manifest sets `[privacy] mode = "local_only"` does not exist on
the cloud channel (`unknown scope`, as for `brain_*`), and on both channels
paths under the manifest's `[privacy] never` globs are dropped before scoring,
from file listings, and from the drift report — which names how many it left out on the local
channel and not on the cloud one. On the cloud channel:
- `graph_file_api` on a file under `never` globs reports not found (`isError`).
- `graph_trace_calls` suppresses callers and callees located in files under `never` globs, incrementing `hidden`.
- `graph_find_all` uses an injected reader that refuses files under `never` globs (`os.ErrPermission`), reporting them in `unreadable_files` without returning any matching lines.
- `graph_repo_map` strips directories and nodes under `never` globs before constructing the map.
- No refresh note goes out either, neither before the answer nor as progress: a note counts files
under the `never` globs too. Every read or query error becomes the fixed text
"the graph could not be read on this channel; ask on the local channel for
details" there, because an error message can name a hidden file or a local
path. A missing graph is the exception and keeps its own text. An internal
error reads "internal error; ask on the local channel for details" there,
without the value it carried.

### The `.mcp.json` of a host

`loomux init` writes it as its part `mcp-json` (see [`loomux init`](#11-project-setup-loomux-init)),
only when the user scope (`~/.claude.json`) names no server `loomux`, and
keeps every other server of an existing file:

```json
{ "mcpServers": { "loomux": { "command": "${LOCALAPPDATA}/loomux/bin/loomux.exe", "args": ["mcp", "--channel", "local"] } } }
```

It calls the machine-wide binary by its fixed place, not `loomux` on the
`PATH`. That Claude Code resolves `${LOCALAPPDATA}` there is not yet
measured; until it is, a host can still name `loomux` on the `PATH` by hand.

---

## 9. Developer Quality Gates (`loomux dev`)

### `loomux dev mutants <package>... [--only <name>] [--family a1|a2|a3|a4] [--workers <n>]`
Mutates the Go decisions of each package and reports which mutants its test suite does not notice — a port of ultra-brain's `tools/go_mutants.py`.

- **Families**: `a1` the whole `if` condition as `true` and as `false`; `a2` each operand of a top-level `&&` or `||` on its own; `a3` every comparison operator flipped (`==`/`!=`, each ordering against its neighbour), not inside comments or strings; `a4` the condition negated. `for` conditions are never mutated.
- **Mechanism**: each mutant reaches `go test -overlay <json> -count=1 -failfast -timeout 60s ./<package>/` through an overlay in a temporary directory; the working tree is never written. Each overlay directory is removed after its run, also after an error or Ctrl+C. `--workers` runs that many at once (default: half the processors, at least 1). `--only` keeps files whose name contains the text.
- **Report**: one line per mutant — `killed`, `SURVIVED` or `no mutant` (does not compile, or changes nothing) — then the sums and the survivors. A run that hits the time limit counts as killed.
- **Exit codes**: `0` after a complete round, survivors included; `2` for a usage error, a package without source files, or a suite that is not green before the first mutant; `1` when a run cannot be started or the round is interrupted with Ctrl+C.

### `loomux dev swap-binary --dir <bin>`
Atomically replaces the running `loomux.exe` binary with `loomux.new.exe` (solving Windows file-locking constraints). The one it replaces is kept as `loomux.old.exe`, or as the first free `loomux.old.<n>.exe` beside it when a process started from an earlier swap -- a `loomux serve` or a bridge -- still holds that name; every slot whose process has ended is removed on the next swap, so at most 16 generations are kept. Two cases still fail, and both leave the binaries where they were: all 16 slots held at once, and a `loomux.exe` that something holds so that it cannot be renamed at all -- a running `loomux.exe` is not that holder, since Windows keeps a running image renamable.

### `loomux dev bench [--dir <dir>] [--corpus <file>] [--languages <n>] [--tier <tier>] [--warm <n>] [--cache-dir <dir>] [--out <file>] [--json-out <file>] [--component-timeout <d>] [--save] [--report-dir <dir>]`
Runs comprehensive latency benchmarks and normalized gap audits on a single repository or against the open-source matrix corpus (1 cold + N warm runs, median/min/max).

- **How hooks are measured**: each hook receives a Claude Code payload for an edit of a sample file in the repository's primary language, so `post-tool-use` runs its real lanes. The Status column lists the exit codes seen across all runs.
- **Single repository mode** (default): measures `pre-tool-use`, `post-tool-use`, and `graph build` (applicable on Go projects), compares against baseline Claude hooks if defined, and audits test/linter coverage gaps against native configuration.
- **Corpus mode** (`--corpus <path>`): clones and benchmarks top-N open-source projects across cataloged languages and frameworks, reporting aggregate matrix latency and tool gaps.
- **Flags**:
  - `--dir <path>`: Target project directory (default: `.`).
  - `--corpus <path>`: Path to open-source matrix Markdown document.
  - `--languages <n>`: Number of languages from corpus to benchmark (default: `5`).
  - `--tier <tier>`: Star category tier filter (default: `"Sehr viel"`).
  - `--warm <n>`: Number of warm measurement runs for median calculation (default: `3`).
  - `--component-timeout <d>`: Deadline for each measured command; a command past it is killed and reported as `timeout` (default: `60s`).
  - `--cache-dir <dir>`: Directory for cached cloned repositories (default: `.cache/benchcorpus`).
  - `--out <path>`: Write Markdown report to file (default: stdout).
  - `--json-out <path>`: Write detailed machine-readable JSON report to file.
  - `--save`: Automatically save benchmark reports into language subdirectories (`docs/{en,de}/benchmarks/<language>/<repo_slug>.md`) and update the central matrix (`docs/{en,de}/benchmarks/matrix.md`).
  - `--report-dir <dir>`: Documentation root directory for saved reports (default: `docs`).

---

## 10. Configuration (`loomux config`)

Shows every key of `.loomux/config.toml` with where its value comes from, and
changes one key at a time as a line edit that keeps every comment and every
line it does not touch.

```bash
loomux config                                   # the interactive form
loomux config list [--json]
loomux config get <key>
loomux config set <key> <value> [--yes | --propose]
loomux config unset <key> [--yes | --propose]
loomux config proposals [--json]
loomux config apply <id>|--all [--yes]
loomux config reject <id>|--all
# each form also takes --root <dir> or --global, before or after the subcommand
```

- **The guard refuses an agent** every form but `list`, `get`, `proposals`,
  a lone `--help` or `-h`, and `set` or `unset` with `--propose`: `set` and
  `unset` without it, `apply`, `reject` and the interactive form (see
  [`hook pre-tool-use`](#loomux-hook-pre-tool-use)). A human runs them.
- **For agents**: an agent changes nothing itself. It proposes a change with
  `loomux config set <key> <value> --propose` or
  `loomux config unset <key> --propose`, which goes through every check of
  `set` and is stored instead of written; the human reviews it with
  `loomux config proposals` and applies it with `loomux config apply <id>`
  (or drops it with `loomux config reject <id>`). `config list` names how
  many proposals are open.
- **Flags** (before or after the subcommand; a flag the subcommand does not take is a usage error):
  - `--root <dir>` — the project; found upwards from the working directory
    when empty.
  - `--global` — the machine-wide `config.toml` in the state directory
    instead. It knows no key yet: `list` prints nothing (`[]` with
    `--json`), and `get`, `set` and `unset`
    refuse every key as unknown. `--global` together with `--root` is a usage
    error.
  - `--yes` — `set`, `unset` and `apply` write without asking.
  - `--propose` — `set` and `unset` store a proposal instead of writing;
    together with `--yes` it is a usage error.
  - `--all` — `apply` and `reject` act on every open proposal instead of one
    `<id>`; an `<id>` together with `--all` is a usage error.
  - `--json` — `list` prints a JSON array of `{key, module, value, origin,
    count}`; `proposals` prints a JSON array of proposals (`[]` when none is
    open).
- **Keys** are `<section>.<name>` (`commit.threshold`, `modules.graph`,
  `verify.timeout`), grouped by module: `base`, `hooks`, `brain`. Every key
  the readers of the file accept is listed, and no other, except the
  per-stack tables `[verify.<stack>.<kind>]` (see `list`).
- **Origins**: `set` (in the file), `default` (the reader's default, shown as
  its value), `preset` (`verify.profiles`, filled by the presets; the value
  shown is the built-in one) and `unset` (no value and no default).

### `loomux config list [--json]`
One line per key: module, key, value, origin. A list of tables
(`commit.allow`, `policy.paths.rules`, `policy.commands.rules`) shows its
number of entries instead of a value. A value longer than 60 characters is
cut there and ends in `…`; `get` and `--json` give it whole. When proposals are open, a last line
`N proposals open — loomux config proposals` follows (`1 proposal open …`
for one); `--json` prints the
rows alone. The per-stack tables
`[verify.<stack>.<kind>]` are not keys: they are changed by hand, and
`loomux check precommit --show` prints what they add up to with the presets.

### `loomux config get <key>`
Prints the key's value, the default where none is set. For a list of tables
it prints one line per entry, the table as Go prints it
(`map[reason:… regex:…]`), and an empty line when there is none; `list` shows the
number of entries (`list --json` as `count`).

### `loomux config set <key> <value> [--yes]`
Computes the new file, prints the change as a line diff on `stderr`, asks
`write these changes? [y/N]` and writes only on `y` or `yes`.

- **Values** are typed as a human writes them: a string without quotes, a
  number, `true`/`false`, a list as `a, b`. A comma inside a `{…}` group
  belongs to the item, so `docs/**/*.{md,txt}, src` is two globs. An item
  with a comma, an empty item or one with blanks at its ends stands in double
  quotes, as a TOML string with its escapes: `"a,b", c` is two items, `""`
  is an empty one. An empty item without quotes is none, so `a,` is the list
  of `a`. The interactive form shows a list in this same form. An enum
  key takes only the values its reader accepts. `verify.timeout` is whole
  seconds (`600`, not `10m`). A number is written in its plain form: `+600`
  and `0600` are `600`. The word `default` is a value like any other.
- **Defaults are never written**: `set` to the default value removes the
  line, and a section it leaves empty goes too, unless a comment is left in it. A file that repeated a
  default would pin it against a later change of the default.
- **Checked by the real readers**: the new text is handed to every reader
  that runs in operation (area declaration, `[modules]`, policy, `[verify]`,
  commit policy, worktree mirrors) and written only when all of them accept
  it.
- **Brain keys need `[area]`**: without it the brain reads nothing, so every
  brain key but `area.scope` is refused with `set area.scope first`.
- **Refused as tables**: `model.roles`, `verify.profiles` and the lists of
  tables are edited by hand.
- **Refused as a guess**: a file that holds the section's keys as dotted or
  quoted keys or an inline table, a multi-line string, a key or section
  written twice, or a line of no known form. The message names the line.
- **Kept as written**: a replaced line keeps its indentation, the spacing
  around `=` and its trailing comment; a leading UTF-8 byte order mark stays.
- **Output**: `loomux config: wrote <path>`, `already so; nothing written`
  (the file would not change) or `declined; nothing written`, all on
  `stderr`.

### `loomux config unset <key> [--yes]`
Takes the key's line out, so the key falls back to its default (or to unset
where it has none), and a section left empty goes too. Diff, question,
readers, refusals and output are those of `set`; a brain key needs no
`[area]` here. A key that is not in the file writes nothing (`already so`).

### `loomux config set|unset … --propose`
Computes the change exactly as `set` or `unset` does, with the same
refusals (exit `1`), but writes no `config.toml`. It stores a proposal as
`.loomux/state/config/proposals/<id>.json` in the project (for `--global`:
`config/proposals/<id>.json` in the state directory) with `id`, `op`
(`set`/`unset`), `key`, `input` (the value as typed), `created` (UTC),
and `target` (`project`/`global`), and prints the diff and the id on
`stdout`. No diff is stored: every form that shows one computes it anew
against the file as it is then. The id is the UTC time and a counter
(`20260924T101530Z-001`), so ids sort in the order they were made. A
proposal that would change nothing is not stored (`already so; nothing
proposed`, exit `0`).

### `loomux config proposals [--json]`
Lists the open proposals, oldest first: id, time, the change asked for, and
the diff it would make to the current file, computed now. Each diff is
against the current file on its own, not against what the proposals before
it would leave behind under `apply --all`. A proposal that no longer holds
shows `refused now: <reason>` instead of a diff, one that is already so
`already so; apply removes it`. With `--json` each entry carries the
recomputed `diff` and, when it no longer holds, an `error`. Nothing open
prints nothing (`[]` with `--json`). A proposal file that does not read is
named on `stderr`, the others are still listed, and the run ends in `1`;
`reject` still removes it. A `config.toml` that does not read is `1`.

Key, value and diff come from an agent. In the output of `--propose`,
`proposals` (text and `--json`) and `apply`, every control character but
the line break and the tab (an escape sequence, a bell, a carriage return,
DEL, the C1 range) and every byte that is no UTF-8 is printed escaped as
`\xNN`; in `--json` that escape stands in the string's value (`"\\x1b"`).
In the header lines of the text forms and in `apply`'s messages, a key or
value is shown as typed when it is one word of printable characters, and
quoted with Go escapes when it is empty or holds a blank, a `"` or a
character that is not printable.

### `loomux config apply <id>|--all [--yes]`
For each proposal in id order: computes the change again from `op`, `key`
and `input` against the file **as it is now**, prints that diff, asks (or
takes `--yes`), writes the way `set` writes and removes the proposal. A
proposal that is already so is removed with a note; a declined one stays;
one that now fails a reader stays with its error, the others go on, and the
run ends in `1`. When the change is written but the proposal file cannot be
removed, it says so and the run ends in `1`; `reject` removes the file. An
unknown id is `1`.

### `loomux config reject <id>|--all`
Removes the proposal, or every open one, without writing anything.

### `loomux config` (interactive)
A full-screen list of the keys, grouped by module: `↑`/`↓` move, `/`
filters, `enter` changes the key under the cursor, `q` or `esc` ends. A text
key is typed; an enum or a boolean cycles its choices with `tab`, and when
it has a default one more choice, `(default)`, takes its line out the way
`unset` does. A typed key goes back to its default with `unset`. The diff
follows with a confirmation, and each change is written on its own, the way
`set` writes it. A table is shown, not edited; a list of tables shows its
number of entries and each entry. The title shows how many proposals are
open. Without a terminal the form
exits `2` and names `list`, `get`, `set` and `unset`.

### Exit codes
`0` success, also a declined confirmation, a change that changes nothing, a
stored proposal and `apply`/`reject` with nothing open; `1` a reader or the
editor refuses (unknown key, invalid value, a guess, a file that does not
read), the write fails, a proposal cannot be stored or read, an unknown
proposal id, or `apply` left a proposal that no longer holds; `2` a usage
error (also `--propose` with `--yes`, a `set`/`unset` that names the
propose flag but ends with it off — `--propose --propose=false`,
`-propose=0`, or `--propose` after `--` where it is a value — which says
`--propose given and switched off; say what you mean`, `--propose` outside `set`/`unset`,
`--all` outside `apply`/`reject`, `apply`/`reject` without exactly one of
`<id>` and `--all`), and the interactive form without a terminal.

---

## 11. Project Setup (`loomux init`)

Sets a project up for loomux: it reads what the project is, asks per module
what to set up, shows every change as a diff and every action by name, and
writes only what a human approves. It replaces ultraloom's `ulinit`,
`scripts/install.ps1` and the hook half of `brain init`. Built with stage
4a-2; running it on a fresh clone of this repository and on a host project is
still to be done by a human (see the [migration plan](migration.md)).

```bash
loomux init [--root DIR] [--dry-run] [--detect-only] [--yes]
            [--hooks=all|each|none] [--brain=all|each|none] [--graph=all|each|none]
            [--hosts=claude,antigravity]
```

- **`--root DIR`**: the project; the working directory when empty.
- **`--dry-run`**: show the plan and write nothing. Without a terminal it
  keeps the defaults instead of asking.
- **`--detect-only`**: print what the project is as JSON (stacks, hosts,
  hook directory, configuration, binary, merge hook, graph) and stop; it
  takes no other flag than `--root`.
- **`--yes`**: take the defaults and approve every change and action,
  without a terminal.
- **`--hooks`, `--brain`, `--graph`**: `all` turns on every part of the
  module, `none` turns the module off, `each` asks part by part. A flag beats
  the answers of an earlier run.
- **`--hosts`**: comma-separated, `claude`, `antigravity` or `codex`; by
  default the hosts the project has (`.claude/` → Claude Code;
  `.agents/hooks.json`, `.agents/skills/` or `GEMINI.md` → Antigravity, a
  bare `.agents/` is not enough; neither → Claude Code).
- `--dry-run=…` and `--detect-only=…` are a usage error, and no flag takes a
  following `--dry-run` as its value: the guard lets a line with the word
  `--dry-run` through, and a value could take that back.

### Modules and parts
The interview asks per module `all`, `each` or `none` (for `each`, a
full-screen list of its parts), then the commit language (`en` or `de`) and,
when the project becomes an area, its scope. What the parts default to
follows the project; the answers of an earlier run come first. `none` always
switches the module off; a module that runs with none of its parts set up
now (the graph of a loomux checkout) is offered as `each`, so taking every
offer keeps it on. The default scope is `project/<directory name>` with
blanks joined by `-`, and `project/root` where the name leaves nothing.

| Module | Part | What it does | Default |
|---|---|---|---|
| base | `binary` | the loomux binary the entries call: `binary-install` puts the newest release at `${LOCALAPPDATA}/loomux/bin/loomux.exe` (through `gh`, checked against `SHA256SUMS` and its `--version`); in a checkout of loomux `binary-build` builds `bin/loomux.exe` | on |
| base | `config` | `.loomux/config.toml`: `[modules]` where a module is off, `[commit] language` where it is not `en`, and the policy rules of the detected stacks still missing; `[verify]` is left to the presets. The text must pass the configuration's own readers | on |
| base | `gitignore` | `.gitignore`: `/.loomux/state/` | on |
| base | `agents-md` | `AGENTS.md`, only when the project has none | on, off in a checkout |
| base | `mcp-json` | `.mcp.json` with the server `loomux` (see [The `.mcp.json` of a host](#the-mcpjson-of-a-host)) | on, off in a checkout |
| base | `tools` | looks for `git`, `qmd`, `pdftotext`, `yt-dlp` and `ollama` on the `PATH` and names the install command of a missing one; installs nothing | on |
| hooks | `host-entries` | the hook entries of each host (`.claude/settings.json`, Antigravity's `.agents/hooks.json`) | on |
| hooks | `git-hooks` | `pre-commit`, `pre-push` (refuses a push to `main` or `master`) and `commit-msg` under `.githooks`, and `git config core.hooksPath .githooks` | on in a repository |
| hooks | `verify-skill` | the skill `verify-until-green` | on, off in a checkout |
| brain | `area` | `loomux area add --scope <scope>`, without `--wiki`, so area add's default wiki applies | on, off in a checkout or an area already declared or registered |
| brain | `merge-hook` | the post-merge hook of `loomux merge-hook install`, for the areas at this root only: a stale area or a foreign hook elsewhere in the registry does not stop init. Planned only when an area of this machine's registry stands here and the declaration here says `[maintenance] on_merge = true`, or `area` runs in the same run on a project without `.loomux/config.toml` (area add writes the consent only into a new one), and only where `${LOCALAPPDATA}/loomux/bin/loomux.exe`, which the hook calls, is installed or `binary-install` runs, a checkout included; without a line for this root the action fails | on in a repository, off in a checkout |
| brain | `brain-skills` | the skills `brain-ingest`, `brain-land`, `brain-research`, `brain-review`, `brain-wiki-plan` | on, off in a checkout |
| graph | `graph-build` | `loomux graph build` | on for a known stack, off in a checkout |

A checkout of loomux (its `go.mod` declares `github.com/xidus90/loomux`) gets
on by default only what its tracked files already have, so that `init --yes`
on a fresh clone is meant to leave `git status` empty; a human run that
checks this is still pending.

### What it writes, and what it leaves
- **Never overwrite, report instead.** A new file is created exclusively; a
  file that is there is changed only where init owns it. A host entry is
  init's when its command calls a loomux binary; a foreign entry stays and is
  named. A host file or `.mcp.json` that is no JSON, whose root is `null`,
  or whose `hooks` or `mcpServers` is no object, is not repaired: the plan
  fails and the run stops with nothing written. An existing `AGENTS.md` or
  skill stays as it is. Before a file changes for the first time, a copy goes to
  `.loomux/state/backup/<path>.bak` — except `.loomux/config.toml`, which is
  replaced whole once its readers accept the new text.
- **Entries call the binary by its place**: Claude Code's entries in a host
  project call `"${LOCALAPPDATA}/loomux/bin/loomux.exe"`, in a checkout
  `"${CLAUDE_PROJECT_DIR}/bin/loomux.exe"`; the git hooks call the same
  binary, a checkout's as `./bin/loomux.exe`. When no binary stands there —
  the part `binary` is off, declined or failed — these entries and git hooks
  are left out and init says so. Antigravity's entries call the installed
  binary in every project through `cmd.exe`, unquoted, as
  `%LOCALAPPDATA%/loomux/bin/loomux.exe`, and like the merge hook they are
  planned only where it is installed or `binary-install` runs (see below).
  `binary-install` fails when `LOOMUX_STATE_DIR` moves the state directory
  away from `${LOCALAPPDATA}/loomux`, because the entries would call
  nothing.
- **Order**: the binary, `area add`, the files, `core.hooksPath`, the merge
  hook, the graph. `area add` writes `[area]`, `[layout]`, `[index]`,
  `[privacy]` and `[maintenance]` only into a configuration that is not there
  yet, so it goes first; init's own change to `.loomux/config.toml` and a
  missing `AGENTS.md` are then made over what it left (the template before
  its routing rule). The plan shows the diff against the file as it stands
  and says so in a note.
- **An action is planned only while its result is missing**: `binary-install`
  without the installed binary (updates stay with `serve` and
  `loomux self-update`), `binary-build` without `bin/loomux.exe`,
  `merge-hook` without our `post-merge`, `graph-build` without a graph.
- **Live hooks keep their directory**: with no `core.hooksPath` and a live
  `pre-commit`, `pre-push` or `commit-msg` in `.git/hooks`, init writes only
  the missing hooks there. A hook directory outside the root (a linked
  worktree, a submodule) is left alone with a note.
- **Antigravity** gets four entries in the group `loomux` of
  `.agents/hooks.json` — `PreInvocation` running `session-start` (timeout
  20 s), `PreToolUse` on
  `write_to_file|replace_file_content|multi_replace_file_content|run_command`
  (15 s), `PostToolUse` on the three writing tools (60 s) and `Stop` with
  `--budget 270s` (300 s); `PreToolUse` also matches `send_command_input`
  and `manage_task`. `PreInvocation` and `Stop` are written as a flat list of
  handlers, the tool events as a block with `matcher` and `hooks`: agy 1.2.11
  refuses the whole file otherwise. A `run_command`, a `send_command_input`
  and a `manage_task` that sends input to a task are judged by the same
  command rules as `Bash`; one whose command line the guard cannot find is
  refused. An entry from before `manage_task` joined the matcher is kept and
  named in a note; add `|manage_task` to it by hand. It also gets
  the skills under `.agents/skills/<name>/SKILL.md`, the same texts Claude
  Code gets. agy runs a hook through `cmd.exe` from `.agents/`: it expands
  `%LOCALAPPDATA%` but leaves `${LOCALAPPDATA}` as it stands, and it breaks a
  quoted program path. So the entries always call the installed binary,
  unquoted, in a checkout too:
  `%LOCALAPPDATA%/loomux/bin/loomux.exe hook pre-tool-use --host antigravity --root ..`
  (and the other three alike). Without that binary, and without a
  `binary-install` in the same run, the plan leaves the entries out with a
  note, as for the merge hook, and a run writes the file only while the
  binary stands; Claude Code's entries keep their own binary. An installed
  binary older than the running init may not know these hooks, and agy
  aborts on a hook that fails: init asks it for its `--version` and plans the
  entries only when it is at least the running init's, or when
  `binary-install` runs in the same run. A development build of init
  (`0.0.0-dev`) has no version to compare and plans none. Each case is named
  in a note. The merge hook does not wait for a version: its call is silent
  and exits 0, so an older binary records nothing until it is updated. When
  `LOCALAPPDATA` contains whitespace or one of `, ; = & | < > ^ ( ) "`, `cmd.exe` would split the unquoted
  path: init then writes no Antigravity entries, says so in a note and does
  not read `.agents/hooks.json`; the skills still come. Every
  other group of the file is carried over token for token (key order,
  escapes and numbers as they were; only the indentation becomes two
  spaces), and a group in which a loomux binary already runs one of these
  hooks, in any form (another path, quoted, another `--root`), is named
  (`the group X already runs loomux hook …; it now fires twice`), never
  repaired. agy
  loads a project's hooks only in a folder it trusts (`trustedWorkspaces` in
  `~/.gemini/antigravity-cli/settings.json`); the plan reminds of that in a
  note. Codex has no hook file.
- **State**: `.loomux/state/answers.toml` keeps the chosen hosts and parts
  (everything else is in `.loomux/config.toml`); `.loomux/state/installed.toml`
  lists what the last run wrote and ran, and is written last, so an
  interrupted run shows its open changes again in the next plan.

### Output
The plan on `stdout`: each change as `--- <path>` and a diff, then
`actions:` with one line per action, then `notes:`; `nothing to change` when
there is nothing. After a run, one line per path or action:
`written: …`, `skipped: …`, `refused: …` (declined), `failed: …`.

### The guard
The guard refuses an agent `loomux init` unless it carries `--dry-run` or
`--detect-only` as a word of its own (see
[`hook pre-tool-use`](#loomux-hook-pre-tool-use)). A human runs it.

### Exit codes
`0` success, a dry run, `--detect-only`, a run cancelled with `esc` or
end of input (`loomux init: cancelled; nothing written`), and a run in which
the human declined a change or an action or switched a part off, the binary
included, even where that leaves the host entries, git hooks and merge hook
out; `1` the project cannot be read (`go.mod`, git, the registry), the
binary step, a change or an action failed, the terminal failed during the
interview or the approval, or it could not be restored; `2`, before
anything is written: a usage error, a root that is no directory, a
`.loomux/config.toml` or `.claude/settings.json` that cannot be read, an
`answers.toml` that does not read, a plan that fails (a configuration its
readers refuse, a host file or `.mcp.json` that is no JSON, whose root is
`null` or whose `hooks` or `mcpServers` is no object, any file the plan
cannot read), and a run
that has to ask without a terminal (`init asks questions; run it
in a terminal, or pass --yes or --dry-run`).


