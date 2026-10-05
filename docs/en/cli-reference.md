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
message on stderr. What `post-tool-use` writes on stdout at exit `0`, the
skipped lanes and the blast monitor's callers, reaches agy as
`injectSteps`, which it shows the model after a PostToolUse (measured with
agy 1.2.12, 2026-09-28).

### Global Flags & Environment
- `--root <path>`: Explicit project root directory. If omitted, Loomux walks upwards from the current working directory until it locates `.loomux/config.toml`.
- `LOOMUX_STATE_DIR`: Overrides the global state directory (defaults to `%LOCALAPPDATA%\loomux` on Windows or `~/.local/state/loomux` on POSIX).
- `loomux version` (also `--version`, `-v`): prints `loomux <version>` on `stdout` and exits `0`; a development build says `0.0.0-dev`.

---

## 2. Policy & Verification (`loomux check`)

`loomux check` takes a **request** first and its flags after it. The request is
a profile (`edit`, `precommit`, `stop` or one of `[verify.profiles]`), `all`, a comma
list of kinds (`lint,types`), or one of the five built-in checks below. What
each kind runs per stack is set by
[`[verify]`](configuration.md#verify-check-chains--quality-gates) and the
presets.

### `loomux check <request> [--root <path>] [--show] [-v] [--arm]`
Runs the lanes of the requested kinds for every active stack and area, and
judges them.

- **Flags** (after the request; `check --show` alone is a usage error):
  - `--root <path>` — project root; found upwards from the working directory
    when empty, the working directory when nothing is found. A project without
    `.loomux/config.toml` is checked by the presets alone.
  - `--show` — run nothing; print the effective lanes as `[verify]` tables.
  - `-v` — print the output of green lanes too.
  - `--arm` — only with the profile `precommit`, else exit `2`. After a run
    that ends green as a whole, it writes every lane that ended `ok` into
    `.loomux/armed.toml` and stages the file with `git add` into the index of
    the commit under way. It writes only where the file exists: without it
    every lane is armed already. In a commit of paths (`git commit <path>`,
    `--only`) it neither writes nor stages and prints `not armed: this commit
    takes only some paths; the next whole commit arms the lanes`. The line
    `armed: <keys>` names what it entered. A file that cannot be written or
    staged is said on `stderr` and does not change the exit code.
- **Lanes in probation**: with a `.loomux/armed.toml` (see
  [Configuration](configuration.md#lane-probation-loomuxarmedtoml)) a lane the
  file does not name still runs. A red one reads `<state> (probation)` and
  fails nothing, a green or skipped one reads as before, and the report of
  every run with such a lane ends with `probation: <keys> (warn only until a
  green commit arms them)`. A file that does not read arms every
  lane, and `stderr` says why.
- **Order**: a lane starts as soon as the lane it waits for (`after`) is done,
  with at most `max_parallel` processes at once. The report comes at the end,
  never interleaved: kinds in request order, within them stacks in byte order,
  then areas. The one exception is `lint/wiki`, the lint over the wiki bundle
  that runs in this process wherever `lint` is requested and the project has a
  wiki: it comes last. It checks the bundle's structure only; the drift rule
  stays with `loomux wiki-gate`.
- **Graph lane**: `check stop` judges the `graph` lane as the stop hook does —
  the working tree, untracked files included, against `HEAD`, through a copy
  of the index in the git directory. It runs the lanes only; the gate around
  them (the marker, a tree already found green, the subagents' findings, the
  block counter) stays the hook's. `check precommit` and a list of kinds read
  the real index.
- **Output** (all on `stdout`): one line per lane,
  `<kind>/<stack>[@<area>]: <state> [<origin>]`, followed by the duration for
  a lane that started, `by <lane>` for a blocked one, or the reason for one
  that never started. `<origin>` is `preset`, `preset, variant <signal>`,
  `config` or `in-process`. A red lane prints its output below its line; a
  green one only with `-v`. A lane with several commands prints one block per
  command, headed `$ <argv>`, with `(failed)` on a red one. A kind that had
  nothing to check ends the report with ``nothing to check for `<kind>` ``
  when the request named it -- a list of kinds, or a profile the project sets
  in `[verify.profiles]` -- or when no lane of the request ran at all; a kind
  of `all` or of a built-in profile beside a lane that ran ends it with
  ``no lane for `<kind>` here, left out`` instead.
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
- **Exit Codes**: `0` (no armed lane red, and every requested kind had a lane
  that ran or is `not-applicable` somewhere), `1` (an armed lane is red, a named kind had
  nothing to check, no lane of the request ran, or `[verify]` or the request cannot be loaded; a load error is one
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
- **Exit Codes**: `0` (Formatted), `1` (Unformatted files listed on `stdout`, or a path that cannot be read, named as `loomux check gofmt: <reason>` on `stderr`).

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
  `graph blast`. `--threshold <n>` (default `3`; the presets pass `5`);
  `--skip-test-callers` (count only callers outside test files: `_test.go` in
  Go; `test_*.py`, `*_test.py`, `tests.py`, `conftest.py` and every file
  under a directory `tests/` or `test/` in Python).
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

Hook entry points are called synchronously by coding agents on tool invocations. Every event but `pre-tool-use` exits `0` without doing anything when `[modules] hooks = false`, and exits `1` when no `--root` is given and no `.loomux/config.toml` is found upwards. An unknown event exits `2` on every host.

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
- **Commands a human runs**: a `Bash` or `PowerShell` line that runs any of
  these is refused:
  - `loomux init`, unless an exempting `--dry-run` or `--detect-only` stands
    beside it;
  - `loomux config`, except `config list …`, `config get …`,
    `config proposals …`, a lone `config --help` or `config -h`, and
    `config set …` or `config unset …` with an exempting `--propose`;
  - `loomux area add`;
  - `loomux merge-hook install` or `remove` (`status` and `record` pass);
  - `loomux dev switchover prune-hooks`, with whatever follows it, `--help`
    included (every other `dev` command passes, `dev switchover render` too);
  - `loomux gate arm` and `loomux gate disarm` (`gate status` passes); the
    refusal says that arm and disarm decide which lanes fail the gate and
    that `loomux gate status` shows them;
  - `loomux convert` or `loomux fetch`, except with a lone `--help` or `-h`.

  The refusal reads ``loomux init, config and area add write
  the configuration the guard reads, merge-hook install and remove write
  executable hooks into repositories, dev switchover prune-hooks removes hook
  entries from a settings file, and convert and fetch write into an
  area's inbox, which the write barrier keeps from agents; a human runs them.
  An agent proposes a change with `loomux config set|unset … --propose`,
  which a human applies``.
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

  These commands write from inside their own process — `.loomux/config.toml`,
  a git hook in another repository, the `settings.json` that holds a
  project's hook entries (the guard's own among them), a file in an area's
  inbox — where no path rule sees the write. The program is
  recognised as `loomux`, `loomux.exe` or a path ending in either (quoted or
  not, `\` or `/`), and as `go` (or `go.exe`) `run` of `cmd/loomux` or
  `cmd/loomux/main.go`
  (with or without `./`, under a module path, at any `@version`, behind build
  flags). It is found behind `VAR=value`, redirections (`2>/dev/null`,
  `> out`, `2>&1`), the reserved words `if`, `then`, `else`, `elif`, `while`,
  `until`, `do`, `!`, `{`, `coproc`, `function <name>`, `try`, `catch`,
  `finally` and PowerShell's dot-source `.` (each reserved word only as the
  word itself: a file of that name, `./do`, is a
  program), after every `{` or `}` on the line, alone or glued to a word
  (the body of a block, function or script block: `try{`, `{loomux …}`),
  and behind the wrappers `sudo`, `command`, `exec`, `nohup`, `env`, `time`,
  `xargs`, `nice`, `ionice`, `stdbuf`, `winpty`, `setsid`, `chronic`,
  `unbuffer`, `timeout <duration>` and `cmd` with every switch up to
  `/c`, `/k` or `/r` (the command may be glued to it, `/cloomux`), a caret
  escape of cmd resolved (`con^fig`), each external one also as `<name>.exe`,
  together with their flags (and `--`, and `env`'s lone `-`).
  The flags are read the way getopt reads them: the separate value of a flag
  of `sudo`, `env`, `xargs`, `nice`, `ionice`, `stdbuf`, `timeout`, `exec`,
  `time` or `unbuffer` that takes
  one is skipped (`sudo -u root`, `xargs -n 1`, `nice -n 10`, `ionice -c 3`,
  `stdbuf -o 0`, `timeout -s KILL`, `exec -a NAME`, GNU `time -o FILE`,
  `unbuffer -ignore HUP`), also when the flag
  is the last of a bundle (`sudo -Hu root`) or a long option cut to a prefix
  (`timeout --sig KILL`), and a redirection between a flag and its value is
  no value. `sudo run` (Sudo for Windows) is read like `sudo`.
  `Start-Process`, `start` or `saps` is refused when loomux is any of its
  arguments, also as the value of a parameter written with a colon
  (`-FilePath:loomux.exe`), whatever the others. Every
  segment of the line counts (`;`, `|`, `&`, `&&`, `||`, a line break, `(`,
  `)`, `$(`, a backtick), and a line continuation (`\` or a backtick at the
  line end) is joined first; a backtick escape inside a word
  (``loomux con`fig``) is read as PowerShell does.
  - **Read inside strings** — a command a shell runs from a string (`sh -c
    "loomux init"`, `pwsh -c …`, `pwsh -EncodedCommand …`, `iex '…'`,
    `eval`, `env -S`) or reads from a pipe or here-string (`echo "…" | sh`,
    `'…' | iex`, `bash <<< '…'`) is judged as a line of its own, up to three
    shells deep (deeper is refused), and no flag exempts there. A variable
    or alias the line sets itself is put in (`M=loomux; $M init`, `alias
    l=loomux; l init`).
  - **Known holes** — the rule reads words, not a shell, so it passes: an
    alias or a program held in a variable set elsewhere than on the line; a
    function the line defines; a command inside `script -c …`; a program a
    known tool runs (`uv run loomux init`, `npx loomux init`, `find … -exec
    loomux …`); `xargs loomux` with its arguments from stdin; `go run .`
    inside `cmd/loomux`; and, after
    an earlier escaped `\"` or `\'` on the same line, a quoted program path
    whose part after its last break character (`(`, `)`, `&`, `;`, `|`)
    holds a blank, such as
    `echo "a \" b"; "C:\Program Files (x86)\My Tools\loomux.exe" init`.
  - **Known false refusals** — it errs toward refusing: behind a program
    it does not know, a later loomux word counts as a call, whatever the
    program does with it (`ssh host loomux init`, `gdb --args loomux init`,
    `zip -r loomux.zip loomux config`; a program that only looks a name up
    or shows its manual, `man`, `tldr`, `which`, is exempt); `echo "x; loomux
    init"`, `start loomux config list`, `Start-Process code -ArgumentList
    loomux`, `command -v loomux init` (which only looks the name up), a
    loomux word right after a brace that opens no block
    (`awk '{ print }' loomux init`, `echo } loomux config set a b`,
    `echo ${X} loomux init`), `loomux init \` followed by
    `--dry-run` on the next line (PowerShell would run the first line alone),
    and `config` with any flag but `--root <dir>`, `--root=<dir>` or
    `--global` before its subcommand (`loomux config --json list`), and
    `convert` or `fetch` with help in any form but a lone `--help` or `-h`
    (`convert -help`, `convert --help=true`, `convert --help x`,
    `fetch --scope x --help`), which only print the help. So is a line that
    carries such text only as data, such as a heredoc holding
    `loomux config set …`.
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
- **Skipped lanes**: a lane whose tool is not on the `PATH`, a Godot project not yet imported, and every lane the budget did not reach are skipped, not failed. Each is named on `stderr`, and at exit 0 on `stdout` in the host's shape, for Claude Code as `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go"}}`, with `<`, `>` and `&` left as they are. A file of a call the budget did not reach is named the same way. At any other exit code nothing is written to `stdout`. Under `--host antigravity` that `stdout` is not passed on, since whether agy reads a PostToolUse's context is unmeasured: there a skip reaches the model only at exit 2, on `stderr`, and the blast monitor's callers not at all.
- **Blast monitor**: after a `.go` edit, when no lane of the call is red, the direct callers in other files of every symbol the edit changed or removed, measured against the graph on disk, follow the skipped lanes in the same `additionalContext`. Silent without a graph and never a finding; see [Hooks](hooks.md#the-blast-monitor).
- **Exit Codes**: `0` (all lanes passed, skipped, or nothing to run), `1` (malformed call, such as a missing or unknown `--host`, a `[verify]` that cannot be loaded, or `--host codex` once the call names a file: Codex has no adapter yet, so the hook refuses rather than answer in another host's shape), `2` (an armed lane failed, timed out or is blocked; its output on `stderr`). A red lane in probation exits 0 and its finding, marked `(probation)`, goes into the host's context like a skipped lane.

### `loomux hook session-start`
Records the commit the session starts on and announces the flow runs waiting for a human.

- **Flags**: `--host <h>` (required; `claude` and `antigravity` have adapters), `--root <r>`.
- **Behavior**:
  - Writes `HEAD` as `base` into `.loomux/state/hooks/<session_id>.json`.
  - Revives a session `worktree unlink` marked ended: removes `<session_id>.ended` and writes the file back with its row of blocks reset, so the session counts again; a session never marked ended only has its file made young. A marker it cannot remove, or a file it cannot write back, is said in the context, with exit 0.
  - Warns in `hookSpecificOutput.additionalContext` when the binary inside the project is older than its Go sources, `go.mod`, `go.sum` or a file under `flows/` whose path has no element starting with `_` or `.` (a flow's `_test/` does not count).
  - Announces every run under `.loomux/state/runs/` that waits at a gate, at every start, a repeated one on Antigravity included: `run <id> (<flow>, <origin>) is waiting at <gate>: <question>`, then `a human answers it with: <binary> flow resume <id> --answer "your answer"`, where `<binary>` is the path the hook runs from, with forward slashes, in double quotes when it holds a blank or another character a shell reads. A runs folder that cannot be listed is named on `stderr` (`.loomux/state/runs cannot be read as a folder of runs: …`), and so is a journal or marker that does not read, which hides only its own run; when the marker says another loomux version wrote the run, the line names both (`run 0001 was written by loomux 0.0.0-dev, this is …`). See [Flows](flows.md#6-gates-are-a-humans).
  - Names each entry under `.loomux/flows/` that carries a bundled flow's name while `[flow] overrides` does not name it: `.loomux/flows/<name> is ignored: [flow] overrides does not name it`. A `.loomux/flows` that cannot be listed gets `.loomux/flows cannot be read as a folder of flows: …`, the words `loomux flow` warns with.
  - Also reads `<state dir>/update.json` and warns when, on Windows, a pass `serve` ran recorded another binary than `<state dir>/bin/loomux.exe` as its own, or when the last self-update pass failed, whoever ran it.
  - At the first start of a session, names the lanes in probation of `.loomux/armed.toml` (one context line each) and the report the stop hook last kept for this `HEAD` under the lanes armed now, cut to 40 lines. A file that does not read is one line `loomux: <error>`.
  - Makes no worktree junctions; that is `loomux worktree link`. See [Hooks](hooks.md#8-session-hooks).
- **Exit Codes**: `0` (Success), `1` (missing or unknown host, no adapter for the host, unreadable payload, failed write).

### `loomux hook stop`
The gate at the end of a turn: delivers what subagents left, then runs the `stop` profile over what changed since the last green run.

- **Flags**: `--host <h>` (required; `claude` and `antigravity` have adapters), `--root <r>`, `--budget <duration>` — how long the lanes may take in all (Go duration, default `270s`, below the 300 s its settings entry grants). Each command gets the smaller of its own `timeout` and what is left of the budget.
- **Standard Input**: the host's `Stop` payload; only `session_id` is read.
- **Behavior**: in this order — the subagents' findings to `stderr`, the block counter (after 3 blocks in a row it gives up for one turn and leaves the findings on disk for the next), the marker `.loomux/no-verify` (it skips the chain, not the findings), the config and its `stop` profile (a `[verify]` that cannot be read ends the gate with exit 1, which holds nothing), the content fingerprint (nothing new since the last green run or the base: no tool starts; with a graph lane only under the same `HEAD`), then the kinds of the `stop` profile (by default `lint`, `types`, `test`, `coverage`, `graph`; the `graph` lane reads a copy of the index holding the whole tree and judges the turn against `HEAD`) in the check scope, plus `lint/wiki` where `lint` is asked for and there is a wiki. A green run moves `base` to `HEAD` and remembers the tree. A run red only in lanes in probation keeps `base`, and remembers its tree, `HEAD`, the armed lanes and its report; a later turn end over the same tree, under the same `HEAD` and with the same lanes armed, starts no tool and says that report again. See [Hooks](hooks.md#stop).
- **Standard Error**: delivered findings as `subagent <agent_id>: <line>`, then only the red lanes, in the format of `loomux check`. At exit 0 with a red lane in probation, the report of those lanes, ending with the `probation: <keys>` line; no host shows `stderr` at exit 0, so the next session start passes it on.
- **Exit Codes**: `0` (the turn ends: green, red only in lanes in probation, nothing new, the marker, or the counter gave up), `2` (the turn is held: an armed red lane, a git failure, or findings delivered — with findings even an exit 1 becomes 2), `1` (the gate could not judge: an unreadable payload or one without `session_id`, the budget ran out, a requested kind had nothing that ran, `[verify]` cannot be loaded — the config is read before the tree, so this exits 1 even when the tree was already found green —, the plan fails, the coverage directory cannot be prepared (`verify.PrepareCover`), or a malformed call; the turn ends).

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
  - Project root path, detected stacks, and whether the wiki bundle is active (with its directory).
  - The lanes the post-edit hook runs per active stack: the `edit` profile as `[verify]` and the presets lay it out, each with its origin, and which of their tools are missing from the `PATH`.
  - The `Stop` entry to wire (`loomux hook stop`, profile `stop`, the wiki bundle as `lint/wiki`), and for each of the six events `PreToolUse`, `PostToolUse`, `SessionStart`, `Stop`, `SubagentStart` and `SubagentStop` whether `.claude/settings.json` calls its `loomux hook` (`[OK]`) or not (`[INFO]`), plus legacy hooks it replaces.
  - A section "Lane Probation" when the project has `.loomux/armed.toml`: each lane as `[ARMED]` or `[PROBATION]`, an entry no lane answers to as `[ORPHAN]`, a `[WARN]` for a pre-commit hook that does not arm lanes and one when there is no pre-commit hook at all (both with the hint to call `loomux check precommit --arm` there, or arm by hand with `loomux gate arm`), and a `[WARN]` when git ignores the file. A file that does not read is one `[WARN]` with its reason. Without the file the section is absent.
- **Exit Codes**: `0` (the report was printed, also when it names a configuration error), `1` (an unknown flag).

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
> **`build`, `check`, `ask`, `callers`, `skeleton`, `grep`, `map`, `stats` and `blast` are wired; `viz` remains specified.** Stage G1 built the packages the graph relies on — `internal/code/model`, `internal/code/pagerank` and `internal/code/blast` — stage G2a added the extractor, the wiring writer, the freshness probe and `graph build` and `check`, stage G2b added the lexicon, lexical scoring, Personalized PageRank blend and `graph ask`, stage G3 put `ask` and `check` behind the MCP tools `graph_find_code` and `graph_check_freshness` (§8), and stage G4a delivered the navigation suite (`callers`, `skeleton`, `grep`, `map`, `stats`) and their four MCP tools; stage G4b added `blast`, the MCP tool `graph_blast`, the checks `graph-fresh` and `blast-audit` (§2) and the post-edit blast monitor (§3). Stage G5a (✅ 2026-09-26) added Python extraction on `gotreesitter`, a Tree-sitter runtime in pure Go, and the extract cache behind `build`.

### `loomux graph build [--root <path>] [--no-reuse]`
Reads and hashes every source file `internal/code/sourceset` finds under the root — `.go` and `.py`, matched exactly and case-sensitively, `.pyi` stubs left out — extracts each with its language's extractor (Go with `go/parser`, Python on `gotreesitter`), resolves them into the deterministic AST graph, and writes it to `.loomux/state/graph/wiring.json`. Each language resolves in a name index of its own, so no edge crosses languages. It also writes the freshness record (`.loomux/state/graph/cache/fingerprint.json`) a later probe reads and the extract cache (`.loomux/state/graph/cache/extract.json`) the next build reads; a failure to write either is announced on `stderr` but does not fail the build, since the graph on disk is already correct.

- **Flags**: `--root <path>` — project root; the working directory when empty. `--no-reuse` — parse every file, whatever the extract cache holds; the way out when an extractor changed without its version.
- **Output**: one line naming files, nodes and edges by relation (`extends`, a Python class's base classes, only when there are any, so a Go graph's line has none), then a line with unresolved import targets, files without a symbol, and the time taken, then one line per language with what the build did with its files, and for a language with parse errors a line naming the first five of those files (`(+N more)` after them). Illustrative shape, not a value to expect — every part of it, including the timing, moves with this repository's own code, and the last three commits each shipped a number here that the next commit falsified: `N files, N nodes, N edges (N contains, N calls, N imports[, N extends])` / `N unresolved import targets, N files without a symbol, Nms` / `  <lang>: N files, P parsed, R reused, E parse errors` / `  <lang> parse errors in: a.py, b.py`. Measured figures with their commands and raw output are in `docs/en/benchmarks.md`.
- **The extract cache** holds, per file, the language's version, the hash of the bytes, the extraction and the body texts. A file whose hash and language version match its entry is not parsed again; the entry of a file that is gone leaves the cache with the next build. A missing cache is a cold start and says nothing. An unreadable cache, or one in another format version, parses every file and says so on `stderr` (`loomux graph build: extract cache ignored, parsing every file: …`); a cache that cannot be written says `extract cache not written: …`. Neither changes the exit code.
- **Parse errors**: a Python file with syntax errors stays in the graph with its file node; a definition the parser had to rebuild around text it rejected gets no node, and the errors are counted in the report: every ERROR and MISSING node of the tree, and one more for a parse that stopped early. The count is a lower bound — a recovered tree can drop text without a node to show for it — so a file with no parse errors counted can still lack a definition. A Go file that does not parse still fails the build.
- **Exit codes**: `0` on success, also when Python files have parse errors; `1` if the root cannot be resolved, a file cannot be read, a Go file cannot be parsed, module resolution fails, or the graph cannot be written; `2` for a usage error.
- **Cost**: `build` never reads the freshness record — it reads and hashes every file, every time, cold or warm alike. The extract cache spares a file only its parse, never its read: an entry counts only against the hash of the bytes just read. `build` is the command that produces the state a probe compares against, so a stale byte in it would be a stale answer, not a saved read. See `docs/en/benchmarks.md` for measured figures.

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
- **Exit codes**: `0` on success (including when no symbols match the query; a `--limit` of zero or below falls back to `8`); `1` on failure (no graph yet, unreadable graph, rebuild failure, syntax error in file when rebuilding); `2` on usage error (no query or more than one, an unknown flag).

### `loomux graph callers <symbol> [--root <path>] [--direction in|out] [-d <depth>] [--in <prefix>] [--no-refresh] [--json]`
Traces who calls, imports, or references a symbol (`--direction in`, default), or what this symbol calls (`--direction out`).

- **Flags**:
  - `--root <path>`: Project root; the working directory when empty.
  - `--no-refresh`: Answer from the graph on disk, never rebuild.
  - `--direction <in|out>`: Trace callers into this symbol (`in`) or callees out of this symbol (`out`).
  - `-d`, `--depth <depth>`: Transitive depth (default `1`; `-d all` or `-d full` for full transitive closure).
  - `--in <prefix>`: Filter symbols by repository path prefix before resolving.
  - Each direct hit (depth 1) carries the first line in the caller's span that names the callee. For `--direction out` that line lies in the start symbol's file and is printed with its path.
  - `--json`: Output machine-readable JSON (`query.CallersAnswer`).
- **Exit codes**: `0` on success; `1` if graph is missing, unreadable, or symbol not found; `2` on usage error.

### `loomux graph skeleton <file> [--root <path>] [--no-refresh] [--json]`
Exports definition signatures, types, and line spans for a file from the graph without function bodies (~10x token reduction).

- **Flags**:
  - `--root <path>`: Project root; the working directory when empty.
  - `--no-refresh`: Answer from the graph on disk, never rebuild.
  - `--json`: Output machine-readable JSON (`skeleton.FileSkeleton`).
- **Exit codes**: `0` on success; `1` if graph is missing or file not found in graph; `2` on usage error.

### `loomux graph grep <pattern> [--root <path>] [-i] [--fixed] [--in <prefix>] [--max-hits <n>] [--no-refresh] [--json]`
Regex search across indexed files, grouped by enclosing symbol and ranked by incoming edge degree (`inDegree`).

- **Flags**:
  - `--root <path>`: Project root; the working directory when empty.
  - `--no-refresh`: Answer from the graph on disk, never rebuild.
  - `-i`, `--ignore-case`: Case-insensitive regex matching.
  - `--fixed`: Treat pattern as a literal string (no regex syntax).
  - `--in <prefix>`: Narrow search to files under path prefix.
  - `--max-hits <n>`: Maximum number of matched lines to return (default `300`); further matches are only counted.
  - `--json`: Output machine-readable JSON (`grep.Result`).
- **Exit codes**: `0` on success (even with 0 hits); `1` on missing/unreadable graph or invalid regex; `2` on usage error.

### `loomux graph map [--root <path>] [--max-dirs <n>] [--hubs-per-dir <n>] [--hotspots <n>] [--no-refresh] [--json]`
Displays token-budgeted directory clusters, local hubs, and global codebase hotspots ranked by in-degree coupling.

- **Flags**:
  - `--root <path>`: Project root; the working directory when empty.
  - `--no-refresh`: Answer from the graph on disk, never rebuild.
  - `--max-dirs <n>`: Maximum number of directory clusters to display (default `16`).
  - `--hubs-per-dir <n>`: Maximum hubs listed per directory (default `3`).
  - `--hotspots <n>`: Maximum repository-wide hotspots (default `12`).
  - `--json`: Output machine-readable JSON (`repomap.RepoMap`).
- **Exit codes**: `0` on success; `1` on missing or unreadable graph; `2` on usage error.

### `loomux graph stats [--root <path>] [--json]`
Prints structural codebase metrics from `.loomux/state/graph/wiring.json`: total node count, edge count grouped by relation, indexed file count, language distribution, and file size.

- **Flags**:
  - `--root <path>`: Project root; the working directory when empty.
  - `--json`: Output machine-readable JSON (`query.StatsAnswer`).
- **Exit codes**: `0` on success; `1` on missing or unreadable graph; `2` on usage error.

### `loomux graph check [--root <path>] [--json]`
Re-extracts the whole tree and diffs it, node by node, against the graph written on disk. It reads the extract cache as `build` does, so a file whose hash and language version match its entry is not parsed again, but never writes it; an unreadable cache only costs it parse time and says nothing.

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

The five data commands read the areas of the one registry (`registry.toml` in `LOOMUX_STATE_DIR` or its platform default) and answer as ultra-brain's `brain-mcp` does; a recorded case corpus (`testdata/cases/1b-1`) holds them to it. A read-only area's artefacts (`index.md`, `graph.json`, `_identities.tsv`) and the reconcile stamp are read from loomux's state directory, under `areas/<flat scope>` and `maintenance/`; an area directory whose `.loomux/config.toml` is missing or has no `[area]` table is not declared.

- **Channel**: every command takes `--channel local|cloud` (default `local`). An area with `[privacy] mode = "local_only"` does not exist on `cloud`; `[privacy] never` globs apply on every channel. Nesting does not lift `local_only`: where the tree of another area holds the wiki or the source tree of a `local_only` area, every path inside it stays hidden on `cloud` through that scope too — `brain read` answers it as a missing file, and search hits, catalog lines, neighbours and `brain status` findings under it are left out. The `local` channel is unchanged.
- **Usage errors** (exit `2`): the usage line, then `loomux brain <command>: error: <reason>` for a missing argument, an invalid choice or `-n` below 1, and `loomux brain: error: <reason>` when the command is missing or unknown or arguments are left over.
- **Runtime errors** (exit `1`): `error: <reason>` on `stderr` and nothing on `stdout` — an unknown scope, a refusal, a missing section, a broken `graph.json` or identity register, a missing or unreadable manifest of any registered area, a missing or broken registry, a search engine that cannot be reached.
- **Advice**: the messages name loomux's own commands, `loomux reindex`, `loomux reconcile` and `loomux embed`; the recorded cases of 1b-1 and 1b-2 carry the reference's wording through rewrite rules of their import maps.

### `loomux brain search <query> [--scope <scope>] [--profile fast|full|keyword] [-n <n>] [--channel local|cloud]`
Searches the visible areas (`--scope all` by default) through the qmd MCP daemon at `http://localhost:8765/mcp`.

- **Profiles**: `fast` (default) vector search without reranking or query expansion; `keyword` BM25 keyword search; `full` the hybrid chain with expansion and reranking. `-n` (default `5`) must be at least 1.
- **Daemon**: when nothing answers there, loomux starts `qmd mcp --http --daemon --port 8765` detached, writes `note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.` on `stderr` and waits up to 60 s for it.
- **Backbone**: what qmd computes on is `[search] backbone` of the machine-wide `<state>/config.toml` (`cuda`, `vulkan` or `cpu`, default `cuda`; `loomux config --global set search.backbone vulkan`). `vulkan` hands qmd `QMD_LLAMA_GPU=vulkan`, `cpu` `QMD_FORCE_CPU=1`, `cuda` nothing. A `QMD_LLAMA_GPU` or `QMD_FORCE_CPU` the user set, even empty, wins and stays untouched. The setting applies to the daemon loomux starts (for `brain search`, the MCP tools and `serve`) and to every qmd command line loomux runs (`brain status`, `reindex`, `embed`, the corpus mode of `dev bench search`). A daemon that is already running keeps the backbone it was started with, whoever started it, until its process ends; `loomux serve stop` ends only loomux's service, not qmd's daemon. To switch, stop the process listening on port 8765 — on Windows `Get-NetTCPConnection -LocalPort 8765 -State Listen | ForEach-Object { Stop-Process -Id $_.OwningProcess }`, on POSIX `lsof -ti tcp:8765 -sTCP:LISTEN | xargs kill` — and the next search starts the daemon with the new backbone. `qmd mcp stop` is not reliable here: a single `qmd status`, which loomux itself runs (`brain status`, `reindex`, `embed`), deletes qmd's PID file, and `qmd mcp stop` then answers `Not running (no PID file).` while the daemon keeps serving. A `[search]` block that does not read is the engine's answer, naming the file and the key: `brain search`, `reindex` and `embed` fail with exit `1`, `brain status` says so on its engine lines, and what asks no engine (`catalog`, `read`, `neighbors`) is not affected.
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
- **Exit codes**: `0`; `1` for a refusal, an area without `graph.json` (``<scope>: never indexed; run `loomux reindex` ``) or another runtime error; `2` for a usage error.

### `loomux brain status [--channel local|cloud]`
Prints what to know before trusting an answer, one line per finding.

- **Lines, in this order**: always the last reconciliation (``last reconcile: never; run `loomux reconcile` ``, ``last reconcile: <iso>; older than 24 h, run `loomux reconcile` `` or `last reconcile: <iso>`); per visible area in registry order, include globs the search engine does not see, a path that does not exist, an area never indexed, fewer than half of its links resolved, and indexed documents the search engine does not know; across all visible areas, the same content under several paths; once, documents indexed but not yet searchable.
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

- **Over areas**: `--scope all` (the default) every area with a wiki path but a workspace that declares no `[area]`, which is passed over without a word; otherwise exactly one. A header line per area, below it `  <path>:<rule>: <message>` or `  no findings`; at the end `no findings` or `<n> findings (<e> errors, <w> warnings)`.
- **Rules**, in the order of the output: `broken-frontmatter`/`missing-type`, `no-sources`, `orphan`, `unlisted-area`, `dead-link`, `outside-area` (warning), `wrong-direction`, `conflict-count`, `untouched` (warning), `stale`, `implemented-without-commit`, `long-planned` (warning). The last two only in areas of the `project/` family; `unlisted-area` only in the signpost.
- **Refusals** (exit `1`, `error: <reason>`, nothing on `stdout`): an unknown scope, a workspace that declares no `[area]` (`… lint sweeps only a brain area`), an area without a wiki path, a wiki path that is not a directory (``… run `loomux wiki init --scope <scope>` first``, in the run over all as well), a registry that cannot be read. A declaration that cannot be read ends the run where it stands.
- **Exit codes**: `0` without an error finding, warnings included; `1` with at least one error finding or a refusal; `2` on a usage error.

#### `loomux wiki-gate [--root <path>]`
The wiki gate of a project whose wiki bundle is active. It reports `wiki-drift` when `git status` shows changed code but nothing changed in the wiki, and every error finding of the bundle lint as `wiki-lint:<rule>`. A project without a wiki bundle passes.

- **Flags**: `--root <path>` — project root; the working directory when empty.
- **Output**: `OK: Wiki Gate passed. …` on `stdout`, or the violations as `  • [<name>] <message>` on `stderr`, followed by `Found <n> violation(s).`
- **Exit codes**: `0` without a violation; `1` with at least one violation, or when the working directory cannot be read; `2` for an unknown flag.

#### `loomux wiki init --scope <scope>`
Lays out the frame of an area's wiki bundle (`_schema.md`, `index.md`, `log.md`, `audit.md`, `_identities.tsv`) and names every file it wrote. An existing file stays as it is. Exit `1` for an unknown scope, an area without a wiki path, or a read-only area.

#### `loomux wiki types`
Counts the page types across every area with a wiki: per type `<type> [<rank>]: <total> (<scope>: <n>, …)`, by total descending, then by name. The rank is `core`, `catalogue`, `origin`, `declared` or `unknown`; an unknown type carries the prefix `? `, a known old name ` -> <catalogue name>`. Across areas the worst rank counts. Exit `0`, unless the registry, a declaration or a page cannot be read (`1`).

#### `loomux wiki retype --scope <scope> --from <old> --to <new>`
Renames one page type in one bundle and names every page it wrote. Only the frontmatter's `type:` line changes; a written page is folded to LF throughout. Skipped are scaffold files, broken frontmatter (a duplicate key included), bytes that are not UTF-8, and a quoted or folded value. A target type no rank knows gives a warning on `stderr`, and the run goes on. Exit `1` for an unknown scope, an area without a wiki path, or a read-only area.

### Upkeep: `loomux reindex`, `loomux embed`, `loomux reconcile`, `loomux area add`

Four commands of ultra-brain's `brain` CLI, top-level commands of loomux since stage 3a; a recorded case corpus (`testdata/cases/3a`) holds them to the Python reference. They read from and write to loomux's state directory alone.

- **Environment**: `LOOMUX_STATE_DIR` holds the registry, the artefacts of read-only areas, `maintenance/` and `qmd-collections.json`. qmd's `index.yml` is found through `XDG_CONFIG_HOME`, else `~/.config`.
- **No `--state-dir`**: the reference accepts it on all four; loomux refuses it like any unknown flag (exit `2`). The state comes from the environment, the one state model of every loomux command.
- **Positional arguments** (exit `2`): none of the four takes one. A word left after the flags is refused with `<command>: unrecognized arguments: <words>` before the environment or qmd is looked at.
- **Messages** of the reconcile pass are German, word for word the reference's.

#### `loomux reindex [--registry <path>]`
Runs a reconcile pass over the registered areas, then rebuilds each area's directory catalogs (`index.md`), link graph (`graph.json`) and identity register (`_identities.tsv`) and enters the areas as collections into qmd's `index.yml`. A writable area keeps its artefacts in its own tree; a read-only area's are written to `<state>/areas/<scope>/` through a staging directory and swapped in whole.

- **`--registry`**: a `registry.toml`, or the directory holding one; default `registry.toml` in the state directory.
- **Catch-up**: the reconcile pass runs first, so that a changed source becomes a case before the index run advances its hash. Cases it opens are listed on `stderr` and the run **goes on**; a vault without a review centre gets a warning and is indexed; any other failure of the pass stops the command before anything is indexed. For a case of a `local_only` area the pass also asks the local model (30 s per case once the model is loaded, see [`loomux reconcile`](#loomux-reconcile)), and a broken `[model]` that ends that pass (see there for when it counts) is such a failure: `reindex` stops.
- **Area lock**: each area is indexed under `<state>/areas/<scope>.lock`, the lock `loomux approve` takes for the same area, so neither rewrites the identity register while the other reads it; a second runner waits until the first releases it. The file stays behind, like `registry.lock`.
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
- **Nested wikis**: a page belongs to the deepest registered area whose wiki holds it. An area whose wiki holds the wiki of another area does not walk into it, so a changed source opens one case for such a page, in the inner area, and never a second one in the enclosing area — whose package would otherwise carry the diff of, say, a `local_only` source into a `manual_cloud` area. The same holds for the pages a merge asks about.
- **Stamp**: the pass writes `maintenance/last-run.txt` in UTC, which `brain status` and `brain search` read.
- **Local model**: for a case of a `local_only` area the pass asks the local model (`[model]` of the machine-wide `config.toml`, see [`loomux config`](#10-configuration-loomux-config)) for a proposal; one whose every claim passes the evidence binding lands as `proposal.md` beside the case, with `prompt_version` in `case.toml`, and anything else leaves a manual case with the note `manual review: the local proposer returned no usable proposal (slice-6 spec §3)`. Each question has 30 s. Before an area's first question in a pass, loomux loads the model into Ollama with the questions' own settings (`num_ctx` included, which Ollama loads a model for) and has it answer one token; that warm-up has no time limit of its own, only cancelling the pass ends it early, so the first case can take as long as loading does. A warm-up that fails (no connection, a status that is not 2xx) is an outage: that area's cases stay manual for the rest of the pass, and no further request is sent for them. With the model off the case stays manual as before. The settings are read only when a `local_only` area is registered; then a `[model]` block that does not read, or an endpoint off the loopback while the model and its role `propose` are on for such an area, ends the pass with exit `1`, after the sources were measured and before any case is written.
- **Exit codes**: `0` for a pass that ran to the end; `1` when a case file cannot be read (`unreadable case: <entry>` on `stderr`), and for a registry that does not read, a vault that declares no review centre or two, or another failure (`error: <reason>`); `2` for a usage error.

#### `loomux area add [--path P] [--scope S] [--wiki W] [--sources S] [--merge-branch B] [--privacy M] [--no-reindex] [-y|--yes]`
Registers a repository as an area and prepares it: the registry entry (written under a lock); `.loomux/config.toml` with `[area]`, `[layout]`, `[index]`, `[privacy]` and `[maintenance]` when the repository has none; the routing rule, appended once to `AGENTS.md`; the wiki bundle frame; then `loomux reindex` with its catch-up.

- **Defaults**: `--path` the working directory; `--scope` `project/<directory name>`; `--sources` `docs` when there is a `docs/` directory, else `.`; `--wiki` `<repo>/docs/wiki` or `<repo>/wiki` (must be absolute); `--merge-branch` the branch git names, `master` without one; `--privacy` `manual_cloud`, one of `automatic_cloud`, `local_only`, `manual_cloud`.
- **Registry first**: a scope already registered is refused before the repository is touched.
- **A kept configuration**: an existing `.loomux/config.toml` is kept byte for byte, with a warning when it declares no `[area]` or another scope. One the declaration reader refuses ends the command with nothing registered.
- **Differences from `brain init`**: no `.mcp.json` and no agent hooks (`loomux init`, stage 4); the index run really happens unless `--no-reindex` is given; the branch is written as `[maintenance] branch`, not `merge_branch`; `--privacy` is checked; the first area of a machine needs no registry file prepared by hand. `-y`/`--yes` is accepted and changes nothing.
- **Exit codes**: `0`, or the exit code of the index run; `1` for a path that is not a directory, an invalid scope, a relative `--wiki`, a refused registry entry, an unreadable file or a failed write; `2` for a usage error, a missing or unknown subcommand (with the usage line) or an unknown `--privacy`.

### Intake: `loomux convert`, `loomux fetch`

Two commands of ultra-brain's `brain` CLI, top-level commands of loomux since stage 4d; a recorded case corpus (`testdata/cases/4d`, 29 cases) holds `convert` to the Python reference, and a recording of Poppler's own output holds the PDF path to the real tool. Both write into an area's inbox, the directory its manifest names as `[layout] inbox`, relative to the area's path.

- **A human's commands**: the guard refuses both to an agent, since a write there is one the write barrier keeps from agents (see [`hook pre-tool-use`](#loomux-hook-pre-tool-use)); only a lone `--help` or `-h` passes.
- **Brain module**: with `[modules] brain = false` in the project found upward from the working directory, both print `loomux <command>: the brain module is off in <file> ([modules] brain = false)` and exit `1` before anything is read. Outside a project nothing is switched off.
- **Environment**: the registry and the area declarations come from `LOOMUX_STATE_DIR`, as for [upkeep](#upkeep-loomux-reindex-loomux-embed-loomux-reconcile-loomux-area-add). `--state-dir` and `--channel`, which the reference accepts and does not use, are unknown flags (exit `2`).
- **External programs**: both are looked up on `PATH` and never installed: `pdftotext` from Poppler (`winget install --id oschwartz10612.Poppler -e`) and `yt-dlp` (`winget install --id yt-dlp.yt-dlp -e`). A missing one is named with that command.

#### `loomux convert [<file>]`
Goes through the inbox of every area in registry order, skipping an area that is read-only, a workspace that declares no `[area]`, an area that declares no inbox or whose inbox is no directory, and converts each regular file in it but `*.md`, in name order (lower case on Windows, as Python sorts paths there). With a `<file>` it converts that file alone, wherever it lies, and never asks the model. Each result is written beside its source as `<name>.<ext>.md` — `doku.pdf` and `doku.txt` become `doku.pdf.md` and `doku.txt.md`.

- **Formats**: a `.pdf`, and a `.txt` whose first 8192 characters hold a transcript mark at a line start: the bracket form `[mm:ss]` or `[hh:mm:ss]`, or the range form `hh:mm:ss - hh:mm:ss` alone on its line. Fragments are joined, without changing a word, into paragraphs of about 1200 characters, each opened by the mark of its first fragment. Anything else is left: `skipped: <name>: no converter knows this format`.
- **PDFs**: through `pdftotext -layout -enc UTF-8 -eol unix <name> -`, run in the inbox, at most 2 minutes per file. Before the first PDF of a run, `pdftotext -v` must name Poppler; xpdf ships a `pdftotext` too and writes another text, so it is refused like a missing program, and every PDF of the run is left with `skipped: <name>: <program> is not Poppler's pdftotext (…); install Poppler with: …`. A run without a PDF never needs the program. Each page is held on its own to the scan threshold: a page with fewer than 100 characters counts as a scan and is left out (`skipped: <name>: <n> page(s) skipped as scanned`, the target still written); a PDF of scans alone writes nothing (`no extractable text, looks like a scan`), one without pages neither (`no pages to extract`), and one `pdftotext` refuses is `cannot be read as a PDF (pdftotext exited <n>: <its first line>)`.
- **The provenance head**: YAML frontmatter, then a blank line and the text:
  ```yaml
  ---
  source_url: https://www.youtube.com/watch?v=<id>
  retrieved: 2026-09-27
  converter: brain-pdf/2
  asr: false
  description: <one German sentence>
  ---
  ```
  `source_url` is read from an eleven-character run in parentheses in the file name (a YouTube id; empty without one); `retrieved` is the UTC date the source was last modified; `converter` is `brain-pdf/2` or `brain-transcript/1`; `asr` is `true` for a transcript, so its text never counts as a verbatim quote; `description` stands only when the model gave a sentence.
- **Second run**: a target is rewritten only when its text would change, so an untouched one keeps its time and is not printed. A target whose head names no converter was written by a person and is never overwritten (`skipped: <target>: not written by us, left untouched`). A sentence a head already carries is kept and never asked for again; a head without one is asked on every run.
- **The local model** (`[model]` of the machine-wide `config.toml`, narrowed by the area's own `[model]`, see [`loomux config`](#10-configuration-loomux-config)): whatever the area's privacy mode, with the model and a role on, `convert` sends the first 1800 characters of the converted text to the local model. Role `describe` asks for the head's one sentence, kept only when it is one German sentence of at most 22 words that no judge refuses (chopped words, measured against an embedded German word-frequency table under CC BY-SA 4.0; a sentence that YAML would not read back); otherwise the head keeps four lines. Role `place` asks, for a file this run wrote, which area it belongs in, offering only the areas that are not read-only and no more open than the inbox's own (`local_only` < `manual_cloud` < `automatic_cloud`); a known answer is a line `suggested: <target>: belongs in <scope>, left in the inbox` on `stdout`, and nothing is moved. No answer, an outage or a refused sentence is no finding. Each role of each inbox warms the model before its first question as [`loomux reconcile`](#loomux-reconcile) does (no time limit for the load, then 30 s per question); a failed warm-up leaves that role without an answer for the rest of the run. The settings are read before the first file: a `[model]` that does not read, or an endpoint off the loopback while a role is on for an inbox, stops the run with nothing converted.
- **Output**: on `stdout` each target written, then the `suggested:` lines; on `stderr` one `skipped: <reason>` line for each file left for a person, and `error: <reason>` for a failure that stops the run (a registry or declaration that does not read, an inbox that cannot be listed — what earlier inboxes wrote is printed first).
- **Exit codes**: `0` when nothing was left; `1` as soon as one `skipped:` line was printed, for an error that stopped the run, and with the brain module off; `2` for a usage error (an unknown flag, more than one file). A scan that stays in an inbox repeats its `skipped:` line and exit `1` on every run, as in the reference.

#### `loomux fetch <url> [--scope <scope>]`
Has `yt-dlp` write a video's subtitles into a fresh temporary directory and files them in the inbox of the area `--scope` (default `knowledge`) as a transcript in bracket form, one fragment per paragraph (`[hh:mm:ss] text`), ready for `convert`. loomux itself never speaks to the network.

- **The call**: `yt-dlp --ignore-config --no-playlist --no-progress --skip-download --write-subs --write-auto-subs --sub-langs de,en --sub-format json3 --write-info-json --ignore-errors -o v <url>`, at most 10 minutes. `--ignore-config` keeps a user configuration from changing name or place, `--no-playlist` fetches only the video of an address with `&list=`, `--ignore-errors` keeps a failing track (a 429 on an auto-translated one) from ending yt-dlp before it writes the info JSON. The exit code of yt-dlp decides nothing; what it wrote does.
- **The track**: manual subtitles before automatic ones, `de` before `en` within each; within a language yt-dlp picks the track. A video without one ends with `<url>: no subtitle track to fetch, and this system does no ASR`.
- **The file**: `<title> (<id>).txt` for an address with a YouTube id, else `<title>.txt`; the title loses what Windows forbids in a name and every control character and is cut to 150 characters, `video` when nothing is left. The inbox is created when missing; a file of the same name is replaced.
- **Refused**: a URL that begins with `-` (yt-dlp would read it as an option, `--` or not: `loomux fetch: a URL does not begin with '-': <url>`, exit `2`); a URL that names only a playlist on `youtube.com`, `www.`, `m.` or `music.youtube.com` (the playlist page `/playlist`, or `/watch` with `list=` but no `v=`, which yt-dlp sends to the playlist page), since `--no-playlist` does not narrow it to one video (`loomux fetch: a URL that names only a playlist is not fetched, give the URL of one video: <url>`, exit `2`; a watch URL with `&list=` is taken); no URL or more than one (exit `2`); a scope the registry does not know, a workspace that declares no `[area]` (`… fetch files only into a brain area`), an area that declares no inbox, a read-only area (exit `1`, `error: …`).
- **Not refused, though yt-dlp reads them as a playlist too**: a bare playlist ID (`PL…`); another `youtube.com` path with `list=` but no `v=` (such as `/embed/videoseries?list=…`); the same pages on another subdomain of `youtube.com` or on `youtubekids.com`. yt-dlp then walks every entry of the playlist; pass the URL of one video instead.
- **Output**: the path of the file written on `stdout`; `error: <reason>` on `stderr`.
- **Exit codes**: `0` for a file written; `1` for a missing `yt-dlp`, a refused area, a fetch that brought no track or an unreadable answer, and with the brain module off; `2` for a usage error.

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
- **An approval** checks first that neither the target page nor a cited source changed since the case was formed, then writes the page with its advanced frontmatter (`generated`, `verified` with the reviewer), advances the identity registers, appends to `log.md` and `audit.md`, removes the case directory and commits exactly those paths onto the vault's current ref through a scratch index (`<state>/maintenance/index`); the user's own index is left alone. It then runs a catch-up pass and an index run, the index run without a catch-up of its own. The catch-up asks the local model for `local_only` cases as `loomux reconcile` does (30 s per case once the model is loaded). When the catch-up fails — a broken `[model]` included — a warning says so and nothing is indexed; when the index run fails, a warning names `loomux reindex`. Neither changes the exit code.
- **A rejection** acknowledges the sources the case was formed over, so the next `loomux reconcile` does not open the same case again: it advances the `revision` and `content_hash` of each matching entry in the page's `sources[]` and the identity registers, appends to `audit.md`, removes the case directory and commits all of it. The page's text, `generated` and `verified` stay as they are, and its hash is not checked; a page without frontmatter, or one deleted or renamed since the case was formed, keeps no rejection from closing the case; a page whose frontmatter does not load (not a mapping, say) refuses the rejection with exit `1` before anything is written. A cited source that changed once more since the case was formed halts the rejection as it halts an approval (note in `case.toml`, exit `1`): that newer state was never under review. The reference does not advance anything on a rejection and reopens the case on every pass; loomux departs from it here on purpose.
- **Area lock**: while an approval or a rejection advances an identity register, it holds `<state>/areas/<scope>.lock` of every area that writes that register — the lock `loomux reindex` takes — and waits for it when an index run holds it. `--defer` takes none.
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
  hours after that for as long as it lives; never `reindex`. The pass asks
  the local model for `local_only` cases as `loomux reconcile` does (30 s
  per case once the model is loaded); a broken `[model]` fails the pass, which is reported like
  any other failure and stops nothing. Every `brain_*`
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

### `loomux upgrade [--beta | --stable | --version <x.y.z>]`

One update pass by hand; `serve` runs the same pass a minute after it
starts and every 24 hours after that. It acts only on the machine-wide
binary, `<state dir>/bin/loomux.exe`, when that is the running binary. A
development build (`0.0.0-dev`) there is replaced by the newest release; one
anywhere else is never touched.

1. Lists the releases through `gh release list` and takes the highest version
   in the machine's channel: with the marker `<state dir>/channel` it takes
   betas and stable releases, without it stable releases only. A binary of the
   count before 1.0.0 without the marker takes the releases of that count and
   the stable ones, never a new beta.
2. Downloads the Windows asset and `SHA256SUMS` through `gh release
   download`, checks the checksum and the new binary's `--version`.
3. Stamps the file with the current time and swaps it in; the old one goes to
   `loomux.old.exe` or the first free numbered slot. The next bridge replaces
   the running `serve`.

The flags choose the release by hand; at most one at a time:

- `--beta` takes the newest release of either kind and sets the marker
  `<state dir>/channel` (the line `beta`), so that `serve` keeps taking betas.
- `--stable` takes the newest stable release and removes the marker. Before the first stable release of the new count
  exists it ends with "no release in channel stable" (exit 1).
- `--version <x.y.z>` takes exactly that version (`v` prefix optional; a beta
  such as `1.1.0-beta.2` is fine). A stable version removes the marker, a beta
  sets it. A downgrade this way does not hold: `serve` lifts the binary again
  within 24 hours.

A problem with the marker alone does not fail the pass: the binary is in
place, the problem goes to stderr, and session start repeats it.

Writes `<state dir>/update.json` (`source` = `serve` | `cli`, `checked_at`,
`executable`, `running`, `result` = `current` | `updated` | `skipped` |
`failed`, `version`, `error`). A pass that finds `update.lock` held steps
aside and writes nothing. A pass by hand that skips writes nothing either,
so the record of `serve`'s last pass stays for session start to read.

| Exit | Meaning |
|---|---|
| 0 | `already current (vX)` or `updated to vX` |
| 1 | the pass failed, or another pass is running |
| 2 | skipped: not on Windows, or not the machine-wide binary (a development build included); or an unrecognized argument, two flags at once, or a version that is none |

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
Mutates the Go decisions of each package and reports which mutants its test suite does not notice.

- **Families**: `a1` the whole `if` condition as `true` and as `false`; `a2` each operand of a top-level `&&` or `||` on its own; `a3` every comparison operator flipped (`==`/`!=`, each ordering against its neighbour), not inside comments or strings; `a4` the condition negated. `for` conditions are never mutated.
- **Mechanism**: the unchanged suite of each package runs first (`-timeout 10m`) and is timed. Each mutant then reaches `go test -overlay <json> -count=1 -failfast -timeout <bound> ./<package>/` through an overlay in a temporary directory; the working tree is never written. The bound is three times the unchanged suite's time, rounded up to whole seconds, and at least 60 s, so a mutant the suite does not notice is not cut off because the suite is slow or several runs share the machine. Each overlay directory is removed after its run, also after an error or Ctrl+C. `--workers` runs that many at once (default: half the processors, at least 1). `--only` keeps files whose name contains the text.
- **Report**: per package first a line with its bound (`internal/cli: each mutant run is bounded at 2m55s (3 × the unchanged suite's 58.3s)`, or `…, the floor (…)` where the 60 s apply), then one line per mutant — `killed`, `timed out`, `SURVIVED` or `no mutant` (does not compile, or changes nothing) — then the sums and the survivors. A run that hits its bound (patience or the test binary's own `-timeout`) is `timed out` and counts as killed; the sums say how many of the killed timed out, since under load a suite that would have passed hits its bound too.
- **Exit codes**: `0` after a complete round, survivors included; `2` for a usage error, a package without source files, or a suite that is not green before the first mutant; `1` when a run cannot be started or the round is interrupted with Ctrl+C.

### `loomux dev swap-binary [--dir <dir>]`
Atomically replaces the running `loomux.exe` binary with `loomux.new.exe` (solving Windows file-locking constraints). The one it replaces is kept as `loomux.old.exe`, or as the first free `loomux.old.<n>.exe` beside it when a process started from an earlier swap -- a `loomux serve` or a bridge -- still holds that name; every slot whose process has ended is removed on the next swap, so at most 16 generations are kept. Two cases still fail, and both leave the binaries where they were: all 16 slots held at once, and a `loomux.exe` that something holds so that it cannot be renamed at all -- a running `loomux.exe` is not that holder, since Windows keeps a running image renamable.

- **Flags**: `--dir <dir>` — directory holding `loomux.new.exe` (default `bin`).
- **Exit codes**: `0` after the swap; `1` when the swap fails (`loomux dev swap-binary: <reason>` on `stderr`); `2` for an unknown flag.

### `loomux dev bench <hooks|repos|search|compare|cases> [flags]`
Three measurements and two helpers under one group. `loomux dev bench` alone prints the subcommands and exits `2`; an unknown subcommand does the same. The group replaces `dev bench-hooks` (now `dev bench hooks`) and `dev bench` (now `dev bench repos`).

- **One report shape**: with `--out <dir>` each of the three measuring subcommands (`hooks`, `repos`, `search`) writes `bench-<stamp>-<command>.md` and `.json` (`dev bench search`: `bench-<stamp>-<profile>`) into that existing directory, both or neither, and never over an existing file, not even one a run of the same minute wrote in the meantime. The stamp is UTC, `YYYY-MM-DD-HHMM`. If either file already exists, the run stops before its first measurement. The markdown of `hooks` and `repos` opens with the command, the stamp, the system, the Go version and the loomux version; that of `search` with the stamp, the profile, qmd, the models, the qmd backbone, the system, loomux, the search path, the measured scope, the number of indexed documents and the question set, then in corpus mode the corpus with a caveat that its numbers measure regression only, and on `fast` a note that the run is purely vectorial. The JSON is one envelope for all three: `schema` (`1`), `command` (`hooks`, `repos` or `search`), `stamp`, `environment` (`os`, `arch`, `cpu`, `go`, `loomux`; `search` adds `qmd`, `models`, `profile`, `port`, which is `daemon` or `cli`, and `backbone`: what a qmd process this run starts computes on (see the backbone of `brain search`), where the user set `QMD_LLAMA_GPU` or `QMD_FORCE_CPU` the value of `QMD_LLAMA_GPU`, `cpu` under a non-empty `QMD_FORCE_CPU` and `default` when both are empty, otherwise the machine's `[search] backbone` (`cuda` when unset); in corpus mode always, since the qmd command line runs on it, and for a run through the search service only when this run started the daemon (it wrote the warming note); `unknown` for a daemon it found running, which keeps the backbone of whichever process started it), `timings[]` (`name`, `cold_ms`, `warm_ms[]`, `median_ms`, `min_ms`, `max_ms`, and where they apply `exit_codes`, `applicable`, `timed_out`), all times in milliseconds, and a `payload` of the command's own (`repos`: the audited repositories; `search`: `question_set`, `corpus` (null outside corpus mode), `documents`, `questions[]` with `id`, `sort`, `rank` (null when not found), `hit` and `elapsed_ms`, `findings[]`, and `latency` (the document and query timed, null without `--latency`); `hooks`: none). `compare` writes the same two files but its JSON is the comparison itself, not this envelope; `cases` writes no report, only a case file and its payloads.

#### `loomux dev bench hooks <case-file> [-n <n>] [--out <dir>]`
Times the hook commands of a case file: every case once cold, then `-n` times warm, and prints a markdown table (case, cold, warm median, min, max, exit codes of the last run) to stdout; with `--out` it writes the two report files as well. The case file is a JSON list of cases, each with `name`, `dir`, `stdin` (a file fed to the steps), `mode` (`single`, the default, and `seq` run the steps in order within one measured span; `par` starts them at once) and `steps[]` (`argv`). The case file may stand before or after the flags. The case files of the chronicle in `docs/en/benchmarks.md` live under `testdata/bench/`.

- **Flags**:
  - `-n <n>`: Warm runs per case, after one cold run (default: `20`, at least `1`).
  - `--out <dir>`: Directory to write the markdown and JSON report to.
- **Exit codes**: `0` after a measurement, whatever the measured commands exit with; `2` for an unknown flag, without a case file, with a second argument or with `-n` below 1; `1` when the case file cannot be read or decoded, a case is invalid, a step cannot be started, or a report file exists or cannot be written.

#### `loomux dev bench repos [--dir <dir>] [--corpus <file>] [--languages <n>] [--tier <tier>] [--warm <n>] [--cache-dir <dir>] [--timeout <d>] [--component-timeout <d>] [--out <dir>] [--save] [--report-dir <dir>]`
Runs latency benchmarks and normalized gap audits on a single repository or against the open-source matrix corpus (1 cold + N warm runs, median/min/max).

- **How hooks are measured**: each hook receives a Claude Code payload for an edit of a sample file in the repository's primary language, so `post-tool-use` runs its real lanes. The Status column lists the exit codes seen across all runs.
- **Single repository mode** (default): measures `pre-tool-use`, `post-tool-use`, and `graph build` (applicable on Go projects), compares against baseline Claude hooks if defined, and audits test/linter coverage gaps against native configuration.
- **Corpus mode** (`--corpus <path>`): clones and benchmarks top-N open-source projects across cataloged languages and frameworks, reporting aggregate matrix latency and tool gaps. A repository that could not be benchmarked is named on stderr as skipped.
- **Output**: without `--out` the markdown report goes to stdout; with `--out` it goes only into the two report files.
- **Flags**:
  - `--dir <path>`: Target project directory (default: `.`).
  - `--corpus <path>`: Path to open-source matrix Markdown document.
  - `--languages <n>`: Number of languages from corpus to benchmark (default: `5`, at least `1` in corpus mode).
  - `--tier <tier>`: Star category tier filter (default: `"Sehr viel"`).
  - `--warm <n>`: Number of warm measurement runs for median calculation (default: `3`, at least `1`).
  - `--cache-dir <dir>`: Directory for cached cloned repositories (default: `.cache/benchcorpus`).
  - `--timeout <d>`: Deadline for the benchmark of one repository (default: `5m`).
  - `--component-timeout <d>`: Deadline for each measured command; a command past it is killed and reported as `timeout` (default: `60s`).
  - `--out <dir>`: Directory to write the markdown and JSON report to (default: markdown on stdout).
  - `--save`: Automatically save benchmark reports into language subdirectories (`docs/{en,de}/benchmarks/<language>/<repo_slug>.md`) and update the central matrix (`docs/{en,de}/benchmarks/matrix.md`).
  - `--report-dir <dir>`: Documentation root directory for saved reports (default: `docs`).
- **Exit codes**: `0` after a benchmark; `2` for a usage error (`--warm` below 1, `--languages` below 1 in corpus mode); `1` when the corpus file cannot be read, a benchmark fails, or a report cannot be written or saved.

#### `loomux dev bench search [--scope <scope>|all] [--profile keyword|fast|full] [--channel local|cloud] [--out <dir>] [--questions <file>] [--corpus v1|<dir>] [--latency] [--latency-query <q>] [--repeat <n>]`
Measures how well the search finds a note: for every question of a question set, the rank of the expected source (a hit at rank ≤ 3) and the time of the answer; with `--latency`, also the latency of catalog, read and the three profiles. The report is always written as the two files; the markdown then goes to stdout.

- **Everyday mode** (default): asks the registered area through the search service (`environment.port` is `daemon`), as loomux searches in use. The question set is `<out>/questions.yaml`; `--out` defaults to `<area>/98 Messung` and must exist. When more than one area is measured (`--scope all` over a registry of several), there is no default and `--out` is required. Without `--scope`, `knowledge` only locates `--out` and with it the question set; what is measured are the areas its `expect` paths lie in: the one area when all lie in one, `all` when they lie in several. An `expect` in no registered area stops the run with exit 1, naming the question and the path; name the area with `--scope` then. A given `--scope`, `knowledge` included, is always measured as given.
- **Corpus mode** (`--corpus`): measures a corpus stand. `v1` names the checked-in `testdata/bench/search/v1` (100 notes, 50 questions, baseline 43/50 on `fast`) and needs a loomux checkout; any other value is a stand's directory. The stand is checked first, then registered in a throwaway state and a qmd index of its own, `loomux-bench-<random>`, and asked through the qmd command line (`environment.port` is `cli`); the shared `index.yml` is never touched. The run copies the `models:` block of `index.yml` into its own index, holds a lock per index name, and removes its index and state on every return path, errors included. Ctrl+C runs no cleanup; a `loomux-bench-*` index that an interrupted or crashed run left and nobody holds is removed by the next corpus run that uses the same state directory. A failed `qmd update` stops the run before `qmd embed`. After the embed, a `fast` or `full` run stops with exit 1 if qmd still reports documents without vectors, naming their count, since the report would measure an unembedded index; `keyword` reads no vectors and does not check. Its latency (seconds per call, since the command line loads the models every time) does not compare with the everyday mode.
- **Question set**: a YAML list of entries with `id`, `sort` (`exakt`, `umschreibung`, `gemischt`, `sprachuebergreifend`), `query`, `expect` (the note, relative to the question file), `beleg` (a passage of that note) and an optional `hinweis`. Every problem is reported at once, before the first query.
- **During the run**: an engine that holds no document for the measured areas stops the run before the first question. What the search chain notes on a query (such as an empty answer twice in a row) is listed as a finding. With `--latency` the latency query is first probed once, untimed; if it finds nothing, the run stops.
- **Flags**:
  - `--scope <scope>`: The area to measure, or `all` (default: the areas the `expect` paths of the question set lie in; `knowledge` only locates the question set).
  - `--profile <p>`: `keyword`, `fast` or `full` (default: `fast`).
  - `--channel <c>`: `local` or `cloud` (default: `local`).
  - `--out <dir>`: Directory for the report (default: `<area>/98 Messung`).
  - `--questions <file>`: Question set (default: `<out>/questions.yaml`).
  - `--corpus <v1|dir>`: `v1` for the checked-in corpus, or a stand's directory. Refuses `--scope`, `--questions` and `--latency`, and requires `--out`.
  - `--latency`: Also time catalog, read and the three profiles, after the quality pass (the chain is warm by then).
  - `--latency-query <q>`: The query the latency searches ask (default: `latenz`).
  - `--repeat <n>`: Warm runs per timed operation, after one cold run (default: `10`, at least `1`).
- **Exit codes**: `0` after a measurement; `2` for an unknown flag, an extra argument, an unknown profile or channel, or `--repeat` below 1; `1` with one `error: <problem>` line on stderr per problem for everything else (a refused flag combination, `--corpus v1` outside a checkout, a broken question set or stand, a missing directory or an existing report file, an empty index, an engine error).

#### `loomux dev bench compare --before <report> --after <report> [--title <t>] [--lang de|en] [--out <dir>]`
Sets two `dev bench hooks` reports side by side: the earlier run (`--before`) against the later (`--after`), with the factor between them (2 is twice as fast, 0.5 half as fast). Cases are paired by name, so the two case files must give the same thing the same name. A case that does not apply on a side (`applicable` false) is left out of that side. The markdown goes to stdout and opens with `# <title>` and one summary line (compared, faster, slower, unclear, new, dropped), then three lists: the cases both runs measured (cold, warm mean, median, the factors and the exit codes of both), the cases only the later run has (new) and those only the earlier one had (dropped). With `--out` it writes `bench-<stamp>-compare.md` and `.json` (the JSON is the comparison itself, indented) as `dev bench hooks` does, both or neither, never over an existing file, and it checks the two names before it reads a report.

- **Flags**:
  - `--before <file>`, `--after <file>`: The JSON reports to compare (required). A report of another `schema` than this loomux writes is refused.
  - `--title <t>`: The heading, for a project's name or an anonymised example (default: `loomux dev bench compare`).
  - `--lang <l>`: `de` or `en`, the language of the headings (default: `de`).
  - `--out <dir>`: Directory to write the markdown and JSON to; it must exist.
- **Exit codes**: `0` after a comparison; `2` for an unknown flag, an extra argument, a missing `--before` or `--after` or an unknown language; `1` when a report cannot be read or decoded, has another schema, names a case twice, or when a report file exists or cannot be written.

#### `loomux dev bench cases [--settings <file>] --root <dir> --file <markdown> --out <dir> [--extras <file>]`
Builds the case file that `dev bench hooks` measures from the hooks of a project's Claude `settings.json`, so the inventory of what runs on an edit comes from the project. It makes one case per event with a command that applies to an edit of `--file` (`SessionStart`, `PreToolUse`, `PostToolUse`, `SubagentStart`, `SubagentStop`, `Stop`, in that order; the tool events only for a matcher that fits `Edit`), named after the event (`PreToolUse (Edit on README.md)` for the tool events), so that the old and the new configuration give the same names for `dev bench compare`. The commands of one event run together (`par`), a single one alone. `${CLAUDE_PROJECT_DIR}` in a command is replaced by `--root`, which is also the directory the cases run in; any other `${NAME}` is replaced by the environment variable of that name (`${LOCALAPPDATA}` in the hook commands loomux installs). The command does not pass through a shell: only the braced form is expanded, and wherever it stands, also inside single quotes, where a shell would leave it; the line is then cut into arguments at blanks and quotes, and an operator, a redirection or a glob stays the text it is. A variable that is not set is an error naming it (the shell would expand it to nothing and leave a broken path); one set to the empty string expands to it. Nothing is printed; the command writes `cases.json` (indented) and, for each case, a payload file `payload-<event>.json` with the input a host hands that hook (an edit of `--file` for the tool events). Every payload carries `session_id` `loomux-bench` and a `transcript_path` to `<root>/.loomux-bench-no-transcript.jsonl`, a file that does not exist; the subagent events also carry `agent_id` `loomux-bench-agent` and `agent_type` `general-purpose`. The hooks that keep state per session refuse a payload without these ids, and measured with them they do their real work, so they may leave state under that session id in the project. `--out` must exist and none of these files may be in it; the files are written all or none, and a write that fails takes back the ones already written.

`--extras` names a JSON file of further cases (the same shape as `cases.json`), for a target without a `settings.json` or with measurements beyond its hooks. In the text of that file `{{ROOT}}` is replaced by `--root` and `{{OUT}}` by `--out` (absolute, without a trailing slash, in slash form) before it is parsed, both as they stand inside a JSON string, so a path with a backslash does not break the file. The extra cases follow the ones of the settings; a case name that occurs twice, between the two or within the extras, is an error. `--settings` may be left out when `--extras` is given; then `cases.json` holds only the extra cases and no payload file is written.

- **Flags**:
  - `--settings <file>`: The Claude `settings.json` to read the hooks from.
  - `--root <dir>`: The project directory (required).
  - `--file <markdown>`: The file the sample edit touches (required).
  - `--out <dir>`: The existing directory for `cases.json` and the payloads (required). A relative one is resolved against the working directory: the cases name their payloads by an absolute path, in slash form, so the measuring run finds them from wherever it starts.
  - `--extras <file>`: Further cases, with `{{ROOT}}` and `{{OUT}}`.
- **Exit codes**: `0` after writing; `2` for an unknown flag, an extra argument, a missing `--root`, `--file` or `--out`, or neither `--settings` nor `--extras`; `1` when a file cannot be read or decoded, a matcher is not a valid expression, a command cannot be split, a command uses a `${NAME}` that is not set, no hook of any event applies to an edit, a case name occurs twice, `--out` is not a directory, or one of the files to write exists or cannot be written.

### `loomux dev switchover <render|prune-hooks> [flags]`
The two pieces of code of the switch-over of a project from the old tools (`ultraloom`, `ultra-brain`) to loomux: an agent prepares, a human runs. `loomux dev switchover` alone prints the subcommands and exits `2`; an unknown subcommand does the same.

#### `loomux dev switchover render --params <json> --out <file>`
Writes the `apply.sh` of one project: a POSIX `sh` script, run in Git Bash, that does every write of the switch-over. The parameters come from a JSON object; a key it does not know, a second document after it, or a value `render` refuses stops it before anything is written, naming the field. The script is written only where no file is (never over one a human may have read already), with LF line endings, and its path is printed.

- **Parameters** (JSON keys; every path absolute, in Windows or POSIX form, but `vault_old` and the entries of `old_files`):
  - `name`, `project`, `loomux`, `config_new` (required): the project's name for the script's head, its repository, the loomux to call (a path, or a name found on `PATH`), and the complete `.loomux/config.toml` to put in place.
  - `registry_new`, with `registry` and `registry_sum` required alongside: the new registry, the live one it replaces, and the SHA-256 of the live one when the new one was made from it (64 lower-case hex digits, as `sha256sum` prints them).
  - `wiki_srcs`, with `wiki_dst` required alongside: the directories the wiki is merged from, in order (a later one wins), and the wiki directory in the project. No entry may be empty or hold `|`, and none may name a directory twice (compared with forward slashes, cleaned, and without regard to case).
  - `vault_old`, with `vault` required alongside: the vault's old wiki folder, relative to the vault; `<vault>/<vault_old>` must be one of `wiki_srcs`, so that the vault only loses what was merged. That folder counts with the files git tracks in it only: empty directories and ignored files are neither copied nor make it a source that is left.
  - `state_area`: the old area's directory in the state directory, renamed to `<state_area>.alt`.
  - `old_files`: files and directories of the old tools, relative to the project, without white space, `..`, a drive or a `.git` at any depth (`sub/.git` is a repository's as well), and none that is, holds or lies in what the script has just put in place: `.loomux`, `.claude` and its `settings.json`, and `wiki_dst`. These names and `.git` are compared without regard to case, since the script removes on a disk that ignores it (`.GIT`, `.Loomux/config.toml` and `Docs/Wiki` are refused as well). An entry with `~` directly before a digit is refused too (`GIT~1`, `sub/LOOMUX~2`, `a~1`): that is the form of the short name NTFS keeps for a file, under which the script's `rm` would find `.git` or `.loomux`; `notes~` and `v1~beta` pass.
  - `old_hooks`: the texts that mark an old hook command, for `prune-hooks`; none may hold `|`.
  - `init_args`: module choices handed to `init` after `--yes`, each of the form `--hooks|brain|graph=all|each|none` (`--brain=none` for a project whose wiki lives in another area; `init` then registers it as a workspace without wiki); anything else is refused.
  - No value may hold a line break; a CR is dropped.
- **The script**: `sh apply.sh --check` prints what it would do (`would run: …`) and writes nothing; `sh apply.sh` does it. Any other argument is refused with exit `2`. It prints one line per step (`registry:`, `wiki:`, `config:`, `probation:`, `init:`, `hooks:`, `files:`, `vault:`, `state:`), saying `already` or `kept` where there is nothing to do, and ends with `done: <project>`. Where a human has to decide it stops with `abort: <reason>` on `stderr` and exit `3`; a command that fails otherwise ends it with that command's code. It puts `/usr/bin` first on `PATH`, since the `find` and `sort` of Windows would otherwise answer. In order:
  1. It stops before any write when the project is no repository or has uncommitted changes; with `vault_old`, when the vault has uncommitted changes, no commit or no remote, or when git tracks nothing under `vault_old` as written but does in another case (write it as git does); when `loomux`, `config_new` (unless the project has a configuration) or `registry_new` is missing; with `old_hooks`, when `loomux` does not know `dev switchover prune-hooks` (a binary older than these commands; `--check` asks it too).
  2. It lays every source of the wiki that holds a file over `<wiki_dst>.staging`, of the vault's folder only the files git tracks, and compares: a file that is at `wiki_dst` already and differs stops it, with one `differs: ./<file>` line per such file on `stderr`; a source that cannot be staged stops it as well. The staging directory is removed however the script ends, also when it is stopped (exit `130`). With no source left that holds a file, the wiki counts as moved when `wiki_dst` exists; otherwise it stops. `--check` builds no staging directory and compares nothing, so a file that differs only stops the real run.
  3. It replaces the registry only when the live one still has `registry_sum`, through a new file that takes its name, so that a session reading it sees the old or the new one whole. The first replacement keeps the old one as `<registry>.bak`; a later one keeps that backup (`registry: backup kept`). A registry equal to `registry_new` counts as replaced, one with another sum stops it.
  4. It copies the files `wiki_dst` lacks and then compares every file of the merge with its copy; the files that are there stay untouched.
  5. It copies `config_new` to `.loomux/config.toml`, never over an existing one. Where it writes the configuration it first writes an empty `.loomux/armed.toml`, so that no lane of a project that never had one fails the gate before a green commit arms it, and prints `probation: started, .loomux/armed.toml written` (`--check`: `probation: would be started, …`). A `.loomux/armed.toml` that stands already — a project `init` set up, or a run stopped after this file — is kept as it is, with `probation: kept, .loomux/armed.toml stands` (`--check`: `probation: would be kept, …`). A standing configuration gets neither the file nor the line. `init` runs after this step, finds the configuration and starts no probation of its own.
  6. It runs `<loomux> init --yes [init_args] --root <project>` (`init: ran (idempotent)`), then `<loomux> dev switchover prune-hooks` with one `--match` per entry of `old_hooks` on the project's `.claude/settings.json`, printing its output and `hooks: already pruned` when no group went, then removes the `old_files` that exist.
  7. After a proven merge, it removes `vault_old` from the vault with `git rm` and one commit; then it renames `state_area`, unless `<state_area>.alt` exists.
  Once the project is committed, a second run changes nothing; before that commit it stops at step 1 (exit `3`), since the first run left changes in the project.
- **Exit codes**: `0` after writing the script; `2` for an unknown flag, an extra argument or a missing `--params` or `--out`; `1` when the parameter file cannot be read or decoded, a parameter is refused, or `--out` exists or cannot be written.

#### `loomux dev switchover prune-hooks --file <settings.json> --match <text> [--match <text>...]`
Removes from a Claude `settings.json` the hook groups a switch-over replaces: a group goes when every one of its commands contains one of the `--match` texts, and an event left without a group goes with it. A group that mixes such a command with others stays. It prints `removed: <event>: <commands>` for each group that went and `kept: <event>: <command>` for each mixed one, or `removed nothing`. Nothing outside `hooks` changes, the new value keeps the file's indent and line endings, and the file is written only when a group goes, so a second run leaves it as it is, down to its modification time. A `--match` text is a plain substring: a short one also takes a group of someone else's that happens to contain it.

A human runs it, as a rule through `apply.sh`: the guard refuses the command to an agent in every form (see "Commands a human runs" under `loomux hook pre-tool-use`), because it rewrites whichever settings file it is pointed at from inside its own process, and would take the guard's own hook entries out of a project the write barrier closes to the agent.

- **Exit codes**: `0` after pruning, also when nothing matched; `2` for an unknown flag, an extra argument, or a missing `--file` or `--match`; `1` when the file cannot be read, is not a JSON object, or cannot be written.

### `loomux dev release <next-version|next-beta|parse-body|changelog-insert|build> [flags]`
The release rules behind `.github/workflows/release.yml` and the `release-pr` check. Without a subcommand, or with an unknown one, it exits `2`. Every error is named as `loomux dev release <subcommand>: <reason>` on `stderr`.

- **`next-version --bump major|minor|patch [--tags <file>]`**: reads one tag per line (default `-`, stdin) and prints the next version. Exit `0`; `2` for an unknown flag, an unreadable tag file or an invalid bump.
- **`next-beta --bump major|minor|patch [--tags <file>]`**: reads tags like `next-version` and prints `X.Y.Z-beta.N` for the version `next-version` would cut, `N` one above the highest beta of that version (`1` without one). Exit `0`; `2` for an unknown flag, an unreadable tag file or an invalid bump.
- **`parse-body [--labels <a,b>] [--body <file>] [--commits <file>]`**: checks the pull request's release label and body (default `-`, stdin) and, with `--commits` (a JSON array of commit messages), that no commit needs a higher label. Prints the parsed body as JSON on `stdout`. Exit `0`; `1` with one line per problem; `2` for an unknown flag or an unreadable body or commit file.
- **`changelog-insert --version <v> --date <YYYY-MM-DD> --link <url> [--notes <file>] [--file <path>]`**: inserts the changelog block (default `-`, stdin) as the release's section into `--file` (default `CHANGELOG.md`, created when missing). `--version` is given without `v`. Exit `0`; `1` when the version is already in the changelog; `2` when a required flag is missing or the file cannot be read or written.
- **`build --version <v> [--channel <name>] [--out <dir>]`**: cross-builds the release binaries into `--out` (default `dist`) and names each file on `stdout`. Exit `0`; `1` when a build fails; `2` for an unknown flag or without `--version`.

### `loomux dev record-case --exe <old-binary>|--argv <program> --cmd <line> --world <dir> --out <dir> [flags]`
Records one case of an old tool under `testdata/cases/`: stages `--world`, runs the command line `--cmd` (with `{{WORLD}}` for the staged directory) and writes what it observed into the case directory `--out`.

- **Flags**:
  - `--exe <path>`: The old binary; `--argv <program and arguments>` puts a program and its leading arguments in place of the command's first token instead. The two exclude each other.
  - `--env KEY=VALUE`: Environment of the recorded process, `{{WORLD}}` allowed; repeatable.
  - `--path-prepend <dir>`: Directory put in front of the recorded process's `PATH`.
  - `--stdin <file>`: File with the payload.
  - `--notes <text>`: Text for `notes.md`.
  - `--compare <mode>`: Empty (compare the data) or `message`.
  - `--git-after`: Pin the commit the run made in `git.after` of the git world's repository.
- **Exit codes**: `0` after the recording; `1` when it fails; `2` for an unknown flag, `--exe` together with `--argv`, or a missing required flag.

### `loomux dev record-mcp-case --argv <program> --tool <name> --world <dir> --out <dir> [flags]`
Records one call of the reference's MCP front as a case: a tool call and its `CallToolResult`, not a command line.

- **Flags**:
  - `--argv <program and arguments>`: The reference's program and its leading arguments.
  - `--tool <name>`: The tool to call; `--arguments <json>` its arguments as a JSON object, `{{WORLD}}` allowed.
  - `--channel <name>`: The channel the case records.
  - `--env KEY=VALUE`, `--path-prepend <dir>`, `--notes <text>`: As for `record-case`.
  - `--compare <mode>`: Empty (compare the text) or `outcome`.
- **Exit codes**: `0` after the recording; `1` when it fails; `2` for an unknown flag, a missing required flag or another `--compare`.

### `loomux dev import-cases --map <file> --from <dir> --to <dir> [--mcp] [--merge-fixture <file>]`
Translates recorded cases from `--from` into `--to` by the `[[command]]` rules (or `[[tool]]` rules with `--mcp`, for recordings of MCP calls) of the TOML file `--map`. With `--mcp`, `[[result]]` rules `{from, to}` replace every occurrence of `from` with `to` in each recorded `result`, byte for byte as recorded, as `[[stdout]]` rules do for a command's output. `--merge-fixture` names a JSON file with two lists and merges it into the `faketool.json` of every translated world, behind what the world recorded: first the `answers`, as they stand, then one copy for every entry `{"prefix": …, "as": …}` of `same` whose `as` the world recorded, which is that recorded answer under the new `prefix` (of two recordings with that prefix the later one, the one the fixture gives). So a command line only loomux asks gets a fixed answer, or the answer the world gave the old command line: a world that recorded a failing `uv run pytest` fails `uv run --with pytest pytest` the same way. A world that did not record `as` gets no copy, and one without a fixture gets one.

- **Exit codes**: `0` after the import; `1` when the map cannot be decoded or the import or merge fails; `2` for an unknown flag or a missing required flag.

### `loomux dev fake-ollama --fixture <file> [--addr <host:port>] [--log <file>]`
Answers every request to an Ollama endpoint with the one answer of the JSON file `--fixture`, until Ctrl+C. It listens on `--addr` (default `127.0.0.1:11435`) and appends the request lines to `--log`, or to `stderr` without it. A request that caps its answer with `num_predict` (loomux's warm-up, which the reference never sends) ends its line with `num_predict=<n>`.

- **Exit codes**: `0` after Ctrl+C; `1` when the fixture or log cannot be opened or the address cannot be served; `2` for an unknown flag or without `--fixture`.

### `loomux dev notices [--out <file>]`
Writes `NOTICE.md` (default `internal/notices/NOTICE.md`) from what the binary built from the checkout in the working directory links: the licence of Go's standard library, of each module and of each tree-sitter grammar whose package is imported, verbatim, and the notice of the embedded word-frequency table. It asks the `go` command for the build graph without cgo, as the release builds, and refuses a copyleft grammar, whose terms would reach the whole binary. A test holds the committed file to what this renders, so a new dependency cannot ship without its notice; the release writes the same text beside the binaries and into `SHA256SUMS`.

- **Output**: the path written on `stdout`.
- **Exit codes**: `0` on success; `1` when `go` fails, a licence file does not read, a grammar is copyleft or the file cannot be written; `2` for an unknown flag.

### `loomux dev record-poppler --exe <pdftotext> --dir <dir> --out <file>`
Records Poppler for the golden test of `convert`: runs `<pdftotext> -v` once and `<pdftotext> -layout -enc UTF-8 -eol unix <name> -` for every `*.pdf` in `--dir`, from that directory and by bare name as `convert` asks, and writes the answers (exit code and output; `-v` with its `stderr` folded into the output) as a fixture of `internal/dev/faketool` to `--out`. A human runs it once with Poppler installed; the fixture has no place for `stderr`, so each PDF's exit code, size and `stderr` go to `stdout` for the parity notes.

- **Exit codes**: `0` on success; `1` when `--dir` cannot be read or the fixture cannot be written; `2` for an unknown flag or a missing `--exe`, `--dir` or `--out`.

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
    instead (`LOOMUX_STATE_DIR`, by default `%LOCALAPPDATA%\loomux`). Its
    keys are the local model's: `model.enabled` (default `false`),
    `model.endpoint` (default `http://127.0.0.1:11434`, loopback only),
    `model.name` (the Ollama model), `model.temperature` (a number from 0
    to 2, default `0.0`) and `model.roles` (a table, edited by hand; once
    set, every role it does not name is off), and the search engine's
    `search.backbone` (`cuda`, `vulkan` or `cpu`, default `cuda`; see
    `brain search`), which exists only here. A new text is checked by the
    reader of `[model]`, by the client's loopback guard and by the reader of
    `[search]`: an endpoint off the loopback or another backbone is refused
    (exit `1`, naming the file and the key) and the file stays as it was.
    After writing `search.backbone`, `set`, `unset` and `apply` print one
    more line: a running qmd daemon keeps its backbone until its process
    ends, so stop the process listening on port 8765 (see the backbone of
    `brain search`); the next search starts it with the new backbone. A file
    that is no TOML is named by its own path. An area's `.loomux/config.toml` knows only
    `model.enabled` and `model.roles`, and can only switch off or narrow.
    `--global` together with `--root` is a usage error.
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
- **Named keys** carry a name the project chooses: `agent.roles.<role>`,
  `agent.models.<name>.provider` and `agent.models.<name>.model` (see
  [`[agent]`](configuration.md#agent-models-for-flow-roles)). `list` shows
  one row per name the file holds, and the family with `*` in the name's
  place as an unset row while it holds none; `set`, `unset` and `get` take
  the key with the name filled in, never the `*`.
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
  commit policy, `[agent]`, `[flow]`, worktree mirrors) and written only when
  all of them accept it. A role binding therefore needs its model first:
  `set agent.roles.reviewer gemini` is refused until
  `agent.models.gemini.provider` is set.
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
4a-2; on 2026-09-28 a human ran it on a fresh clone of this repository and
interactively in a host project (see the [migration plan](migration.md)).

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
  the answers of an earlier run. `--brain=none` still registers the project,
  as a workspace without wiki (the part `workspace`), so that the write
  barrier opens its tree.
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
| base | `config` | `.loomux/config.toml`: `[modules]` where a module is off, `[commit] language` where it is not `en`, and the policy rules of the detected stacks still missing; `[verify]` is left to the presets. The text must pass the configuration's own readers. With it an empty `.loomux/armed.toml`, only where before the run there was neither it, nor a configuration, nor a pre-commit hook of loomux (see below) | on |
| base | `gitignore` | `.gitignore`: `/.loomux/state/` | on |
| base | `agents-md` | `AGENTS.md`, only when the project has none | on, off in a checkout |
| base | `mcp-json` | `.mcp.json` with the server `loomux` (see [The `.mcp.json` of a host](#the-mcpjson-of-a-host)) | on, off in a checkout |
| base | `tools` | looks for `git`, `qmd`, `pdftotext`, `yt-dlp` and `ollama` on the `PATH` and names the install command of a missing one; installs nothing | on |
| hooks | `host-entries` | the hook entries of each host (`.claude/settings.json`, Antigravity's `.agents/hooks.json`) | on |
| hooks | `git-hooks` | `pre-commit` (it runs `check precommit --arm`), `pre-push` (refuses a push to `main` or `master`) and `commit-msg` under `.githooks`, and `git config core.hooksPath .githooks` | on in a repository |
| hooks | `verify-skill` | the skill `verify-until-green` | on, off in a checkout |
| hooks | `workspace` | only while the brain module is off: `workspace-add` writes a registry entry with the scope, `path` and `workspace = true` and no `wiki`, under the registry lock as `area add` does; nothing in `.loomux/config.toml`, no wiki, no routing rule, no merge hook. The brain readers skip such an entry while the project declares no `[area]`. Without an entry the write barrier refuses every write in the project. A scope another path already holds stops the run with exit 1; the action runs before every file, next to `area-add`, so a refused registration leaves nothing written. With the hooks module on and the brain off, the plan notes `workspace: skipped; the registry has an area at this root already` for a registered root, chosen or not, and `workspace: skipped by choice; the write barrier opens no tree in this project until a human registers one` when the part is deselected in a fresh project that declares no `[area]` and is not registered | on, off in a checkout or an area already declared or registered |
| brain | `area` | `loomux area add --scope <scope>`, without `--wiki`, so area add's default wiki applies | on, off in a checkout or an area already declared or registered |
| brain | `merge-hook` | the post-merge hook of `loomux merge-hook install`, for the areas at this root only: a stale area or a foreign hook elsewhere in the registry does not stop init. Planned only when an area of this machine's registry stands here and the declaration here says `[maintenance] on_merge = true`, or `area` runs in the same run on a project without `.loomux/config.toml` (area add writes the consent only into a new one), and only where `${LOCALAPPDATA}/loomux/bin/loomux.exe`, which the hook calls, is installed or `binary-install` runs, a checkout included; without a line for this root the action fails | on in a repository, off in a checkout |
| brain | `brain-skills` | the skills `brain-ingest`, `brain-land`, `brain-research`, `brain-review`, `brain-wiki-plan` | on, off in a checkout |
| brain | `model` | asks Ollama (`GET /api/tags`) for the model `[model] name` of the machine-wide `config.toml` names and, when it is missing, plans `model-pull`: `ollama pull <name>` through `POST /api/pull`, with its progress on stderr and no total time limit; Ctrl+C ends only the download. The one tool `init` installs rather than names; a failure is a note and `init` goes on. When Ollama cannot be reached or the endpoint is off the loopback, the note is all. Only `init` pulls a model, `reconcile` never does | on with `[privacy] mode = "local_only"` in the project or `[model] enabled = true` globally, off otherwise |
| graph | `graph-build` | `loomux graph build` | on for a known stack, off in a checkout |

A checkout of loomux (its `go.mod` declares `github.com/xidus90/loomux`) gets
on by default only what its tracked files already have, so that `init --yes`
on a fresh clone leaves `git status` empty; a human run on 2026-09-28 built
the binary, set `core.hooksPath` and found it so, and a second run had
nothing to change.

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
- **Lane probation.** `init` writes an empty `.loomux/armed.toml` only where
  before the run there was neither that file, nor `.loomux/config.toml`, nor a
  pre-commit hook of loomux, and the part `config` is chosen: no lane of a
  project that loomux was never set up in fails the gate before a green commit
  arms it. A project that is set up already gets the probation only through
  `loomux gate disarm --all`, run by a human; `init` creates nothing there,
  not even when it renews the hook. The pre-commit hook it writes runs `exec
  "<binary>" check precommit --arm`. Its own older pre-commit hook (the
  shebang, the marker line and the one call `check precommit`, nothing else)
  is replaced by today's, with a copy under `.loomux/state/backup/`. A file
  `init` replaces keeps its permission bits, and a script only gains the
  execute bit, so the hook stays executable. A pre-commit hook of the
  project's own is kept as it is; where the project has the file or gets it in
  this run and that hook does not arm, a note names it with the hint to call
  `loomux check precommit --arm` there, or arm by hand with `loomux gate arm`.
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
  `loomux upgrade`), `binary-build` without `bin/loomux.exe`,
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
  refuses the whole file otherwise. A `run_command` is judged by the same
  command rules as `Bash`; what a `send_command_input` or a `manage_task`
  types into a task is judged the same way, line by line, and only as whole
  lines without control characters or a line continuation. A call whose
  command line the guard cannot find is refused; `manage_task`'s `list`,
  `status` and `kill` pass only while they carry no line. An entry from
  before `manage_task` joined the matcher is kept, and init appends a block
  for `manage_task` beside it with the current loomux command and a note; a
  Claude Code entry of ours under ulinit's matcher gets one for `MultiEdit`
  the same way. An entry under a matcher that is no plain list of tool names,
  such as `.*`, is only named, and what it lacks is added by hand. It also gets
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
`written: …`, `skipped: …`, `refused: …` (declined), `failed: …`, and
`note: …` for a failure that stops nothing (a model pull that did not
finish, with the command to run by hand). While `model-pull` runs, its
progress goes to `stderr` as `model-pull: …` lines: one per status, and one
per ten percent of a layer being downloaded.

### The guard
The guard refuses an agent `loomux init` unless it carries `--dry-run` or
`--detect-only` as a word of its own (see
[`hook pre-tool-use`](#loomux-hook-pre-tool-use)). A human runs it.

### Exit codes
`0` success, a dry run, `--detect-only`, a run cancelled with `esc` or
end of input (`loomux init: cancelled; nothing written`), and a run in which
the human declined a change or an action or switched a part off, the binary
included, even where that leaves the host entries, git hooks and merge hook
out, and a run whose model pull failed or was ended with Ctrl+C (a `note:`,
init goes on); `1` the project cannot be read (`go.mod`, git, the registry), the
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

---

## 12. Flows (`loomux flow`)

Runs a flow, pauses it at a gate, resumes, replays, shows and lists flows: the
catalog the binary ships and the project's own under `.loomux/flows/`. The
format, the roles, overlays and the catalog are in [Flows](flows.md). Agent
nodes wait for the model adapters: a flow with one refuses to start.

```bash
loomux flow run [<flow>] [--option name=value]... [--root dir]
loomux flow resume <run> [--answer text] [--root dir]
loomux flow replay <run> [--root dir]
loomux flow show <run|flow> [--root dir]
loomux flow list [--root dir]
```

- **The project** is `--root`, else the nearest `.loomux/config.toml` above
  the working directory, else the working directory itself.
- **Name first, flags after**: the flow or the run number comes before the
  flags. `-h` prints the flags and exits `0`.
- **Every command that loads a flow reads `[agent]` and `[flow]`** before
  it, and stops with exit `1` at a table its reader refuses (see
  [Configuration](configuration.md#agent-models-for-flow-roles)). `resume`
  and `replay` look at the run's journal and marker first; the refusals that
  depend on its open gate come after the flow is found and loaded. `show
  <run>` reads only the journal.
- **No `stdin`**: no command reads it; a run keeps everything it is left
  with in its journal and its marker, so any caller drives it the same way.
- **Warnings** go to `stderr` as `warning: …` and do not change the exit code:
  a project folder a bundled flow ignored, overlays that changed since a run
  started, a node that ran on a definition that has changed since.
- **Refusals** go to `stderr` with exit `1`; a flow that does not load is
  refused with every finding of its load stage, one line each.
- **The guard** refuses an agent `resume … --answer`; every other form is open
  to it (see [Flows](flows.md#6-gates-are-a-humans)).

### Exit codes

| Code | Meaning |
|---|---|
| `0` | the run is done; `show` and `list` succeeded; `-h` |
| `1` | an error or a refusal: a flow that does not load, a run that failed (a node that failed without an error edge, no edge that applies, a visit ceiling), an answer no choice matches, a replay of a run that ended at an exit node |
| `2` | a usage error: no subcommand or an unknown one, a missing name or run number, a run number that is not all digits, an argument left over, an unknown flag, an `--option` without `=` or given twice |
| `3` | the run is paused at a gate |
| other | the code of the exit node the run ended at |

### `loomux flow run [<flow>] [--option name=value]... [--root dir]`
Starts a new run and walks it until it is done, pauses at a gate or fails.

- **Without a name** `[flow] default` runs. Unset, the command refuses with
  `no flow named and [flow] default is unset; known flows: example, ship`; a
  default that names no flow with `[flow] default names "x", which is no flow
  here`.
- **`--option name=value`** sets a parameter; repeatable. The text is read
  as the parameter's type: an `int` in decimal, a `bool` as `true` or `false`,
  a `list[string]` as a JSON list of strings (`'["a","b"]'`), a `string` as
  it stands. Every option that is no parameter or does not read as its type is
  named at once (`option x is no parameter of flow "ship"; known parameters:
  none`).
- **Refused before a run exists**, exit `1`: a name that is no flow name
  (`"Bad" is not a flow name; a flow name is [a-z][a-z0-9-]*`), no flow of
  that name (`no flow named "nope"; known flows: example, ship`), a load
  finding, a provider without an adapter (`no adapter for provider claude
  yet`, today every flow with an agent node), agent nodes that resolve to two
  providers.
- **A run** takes the next number under `.loomux/state/runs/`, writes its
  marker with the options, the origin and, when git answers, `HEAD` and the
  changed files, and journals every step. A run the runner refuses before its
  first step gives its number back; one that fails later keeps its files, and
  the message names the run (`run 0003: …`).
- **Output** on `stdout`: `run <id> (<flow>, <origin>): <status>` with the
  status `done`, `paused` or `error`, then the gate's question or the reason
  the run ended.

```
$ loomux flow run ship
run 0001 (ship, project): paused
Ship it?
$ loomux flow run quick        # a flow whose start is an exit node with code 5
run 0002 (quick, project): error
stopped at once
$ echo $?
5
```

### `loomux flow resume <run> [--answer text] [--root dir]`
Carries on a run that waits at a gate.

- **Without `--answer`** the gate asks again and nothing is written; exit
  `3`.
- **With `--answer`** the gate takes the answer and the run walks on: exit
  `0` when it is done, `3` at the next gate, an exit node's code. An answer is
  a choice, or a choice, a blank or `:` and a reason. `--answer ""` is an
  answer too. An answer no choice matches is refused (`the answer matches none
  of the choices; the choices are yes, no`, exit `1`), and the gate stays
  open.

  ```
  $ loomux flow resume 0001 --answer "no: too thin"   # the bundled example, in its test
  run 0001 (example, bundled): error
  rejected after 2 rounds
  $ echo $?
  4
  ```
- **Refused as a usage error**, exit `2`, before anything is read: a run
  number that is not all digits, as `loomux flow run` hands them out
  (`loomux flow resume: "../x" is no run number; a run number is digits, as
  loomux flow run hands them out`). The number becomes part of a path, so a
  `../`, a separator or a drive would read a journal and marker elsewhere.
- **Refused**, exit `1`: a run that is not there (`no run "0009" under …`,
  with the runs folder in forward slashes), a run without a marker
  (`run "0001" does not say which flow it belongs to`) or with one that does
  not read (an empty one: `…0001.flow: says nothing -- not even which flow it
  belongs to`), a run not
  waiting at a gate (``run 0002 is not waiting at a gate; there is nothing to
  answer. Use `loomux flow replay` to re-derive it, or `loomux flow run` to
  start a new one``), a flow whose `flow.toml` now comes from the other
  source, and a provider without an adapter.
- **The source of the flow** is compared with the marker before the flow is
  loaded. The project folder (`project`, `project (hides bundled)`) and the
  catalog (`bundled`, `bundled+overlay`) are two sources; a change between
  them is refused with both origins (`run 0001 started on project (hides
  bundled) and example now resolves to bundled, another flow.toml; start a
  new run with loomux flow run example`). Within one source the run goes on;
  a different set of overlay files is a warning (`warning: run 0001 started
  with overlays questions/approve.md and now has instructions/draft.md,
  questions/approve.md`).
- **Only a human answers.** The guard refuses `--answer` to an agent in every
  spelling; `resume` without it is open.

### `loomux flow replay <run> [--root dir]`
Re-derives a finished run from its journal. It executes no node and asks no
model, so it works without an adapter.

- **Exit** is that of the run as recorded: `0` for done, `1` for an error.
  The journal keeps an exit node's message but not its code, so a run that
  ended at one replays with its message and exit `1`.
- **Refused as a usage error**, exit `2`: a run number that is not all
  digits (as for `resume`).
- **Refused**, exit `1`: a run waiting at a gate (``run 0001 never finished:
  it is waiting at gate "confirm"; answer it with `loomux flow resume` before
  replaying``), a run that is not there, a run without a marker or with one
  that does not read, a flow whose `flow.toml` now comes from the other source
  (as for `resume`).

```
$ loomux flow replay 0002      # run 0002 of quick ended at its exit node with code 5
run 0002 (quick, project): error
stopped at once
$ echo $?
1
$ loomux flow replay 0001      # run 0001 of ship waits at its gate
run 0001 never finished: it is waiting at gate "confirm"; answer it with `loomux flow resume` before replaying
$ echo $?
1
```

### `loomux flow show <run|flow> [--root dir]`
A run number is digits, a flow name starts with a letter, so the two cannot be
mistaken.

- **A run**: one line per journal entry, in columns: node, kind, outcome,
  tokens, seconds, tool profile (`-` where there is none).

  ```
  $ loomux flow show 0001
  confirm                  gate   paused        0 tok    0.00s -
  ```
- **A flow**, as a run would load it: its origin, its nodes (an agent node
  with its role, the model it resolves to and how) and its edges with the
  condition each is taken on, `[on error]` for an error edge.

  ```
  $ loomux flow show example
  example (bundled)
  nodes:
    draft    agent  writer  claude:cli-default  role writer (node), the CLI's own default
    approve  gate
    stop     exit
  edges:
    draft -> approve [verdict == "done"]
    draft -> draft
    approve -> END [answer == "yes"]
    approve -> stop
    stop -> END
  ```

### `loomux flow list [--root dir]`
Every flow the project can name, sorted: name, origin (`project`, `bundled`,
`bundled+overlay`, `project (hides bundled)`) and `ok`, or the findings that
keep it from loading, one line each; `(default)` marks `[flow] default`. A
flow that does not load is listed with its reason, not left out, and so is a
file directly under `.loomux/flows/` (`… is a file; a flow is a folder with
flow.toml`) and a link there, symbolic or a junction (`… is a link; a flow is
a folder with flow.toml`).

```
$ loomux flow list
broken   project  .loomux/flows/broken/flow.toml: schema_version 7 is unknown; loomux knows version 1
    .loomux/flows/broken/flow.toml: [flow] is missing
example  bundled  ok
ship     project  ok
```

- **Warnings** on `stderr`: a project folder a bundled flow ignored
  (`warning: .loomux/flows/example is ignored: [flow] overrides does not name
  it`), and a name in `[flow] overrides` no bundled flow has (`warning: [flow]
  overrides names "ghost", which no bundled flow has`).
- **Exit** `0`; `1` after the list when `[flow] default` names no flow.

---

## 13. The Gate (`loomux gate`)

What a human says about which lanes fail the gate: the commands for
[`.loomux/armed.toml`](configuration.md#lane-probation-loomuxarmedtoml). A
group of its own and not under `check`, where every first word is a profile or
a kind. A lane is named by its key, `<kind>/<stack>@<area>`, written with `/`
(`lint/go@sub/dir`, `lint/go@.` for the root); a key typed with `\` is read
with `/`. Every command takes `--root <dir>`, the project. Without it the
project is found upwards from the working directory: the directory holding
`.loomux/config.toml`, searched first and without bound as `check` searches
it; else the nearest one holding `.loomux/armed.toml`, a search that stops at
the top level of the git repository; else that top level. Outside a
repository there is no search for `.loomux/armed.toml`, and without a
configuration the working directory is the project. So a project in
probation without a configuration is found from any of its subdirectories,
and the file is never written beside it there. An
agent runs `loomux gate status` only: the guard refuses `arm` and `disarm` to
an agent.

### `loomux gate status [--root <dir>]`
Prints every lane of the project's gate and every entry no lane answers to,
one `<key>: <state>` line each, sorted by key; the state is `armed`,
`probation` or `orphan` (an entry whose stack left or whose area was
renamed). The lanes are those every profile can run: a lane with nothing to
check (`not-applicable`, `unavailable`) is not listed, while `missing-tool`
and `unready` are.

- **The three states of the file.** Without it the command prints `no
  .loomux/armed.toml: every lane is armed` and plans no lane. With it, the
  list below. A file that does not read is an error (exit `1`) that says why
  and that every lane is armed.
- **A file git ignores** is listed as it stands, and `stderr` warns that it
  reaches no commit and holds on this machine only.

```text
$ loomux gate status
coverage/go@.: probation
coverage/python@web: probation
lint/go@.: armed
lint/python@web: probation
lint/typescript@old: orphan
test/go@.: probation
test/python@web: probation
types/python@web: probation
```

### `loomux gate arm <lane>... [--root <dir>]`
Enters the lanes into the file, a red one included: from now on they count.
It prints `armed: <keys>`. A key no lane answers to is an error before
anything else, with the file and without it, and nothing is written. Without
the file nothing is written either, since every lane is armed already: the
command prints the line `no .loomux/armed.toml: every lane is armed`.

### `loomux gate disarm <lane>...|--all [--root <dir>]`
Takes entries out of the file, an orphaned one included, and prints
`probation: <keys>`; a key the file did not hold is named `<key>: was not
armed`. Without the file it creates one that names every other lane as armed
(a key that is no lane is an error first). `--all` writes the file without
an entry, or empties it, and prints `probation: every lane`: **the way to give
a project that is set up already the probation.**

- **Exit codes**: `0`; `1` for a file or the lanes that cannot be read or
  written, and for an unknown key; `2` for a wrong call (no subcommand, an
  unknown one, `arm` without a lane, `disarm` with neither a lane nor `--all`,
  or with both).

