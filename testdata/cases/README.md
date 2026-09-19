# The case corpora

The cases here are the parity evidence of loomux against the tools it
replaces: recordings of the old tools, translated into loomux's own command
lines and replayed in process. Each stage keeps its own worlds, recordings,
translation table and suite.

| Stage | Recorded from | Suite | Cases |
|---|---|---|---:|
| 1a | `ulguard` and `ulinit` from ultraloom, `brain` from ultra-brain, both at the tag `loomux-1a-source` | `internal/cli/cases_test.go` | 19 |
| 1b-1 | `brain-mcp`, the Python reference of ultra-brain at the tag `loomux-1a-source` (`3cc72d2`), against a fake qmd | `internal/cli/cases_1b1_test.go` | 71 |
| 1b-2 | `brain-mcp mcp`, the same reference's MCP front over its own daemon, against a fake qmd | `internal/cli/cases_1b2_test.go` | 54 |
| 2a | `ultraloom check` at the tag `loomux-1a-source` (`9d01a60`), against fake tools | `internal/cli/cases_2a_test.go` | 53 |
| 2b | `ultraloom commit-msg` at the tag `loomux-1a-source` (`9d01a60`), against staged worlds and, for `--calibrate`, a fake git | `internal/cli/cases_2b_test.go` | 19 |

## Layout

| Path | What it holds |
|---|---|
| `1a-worlds/` | The staged project trees a case runs in: `project-writable`, `guard-allow-writable`, `guard-deny-readonly`, `plain`. Each recording copies the whole world into its own `world/`. |
| `1a-payloads/` | The hook payloads fed to a run on stdin. `{{WORLD}}` in a payload is replaced by the staged world directory. |
| `1a-source/` | The recordings of the old binaries, written by `loomux dev record-case`. `verb/name/` holds `cmd`, `exit`, `stdout`, `notes.md`, the staged `world/`, and optionally `stdin` and `compare`. |
| `1a-map.toml` | The translation table: `[[command]]` rules rewriting the head of a recorded command line (`ulguard post-edit --root …` → `loomux hook post-tool-use --host claude --root …`). Longer prefixes stand before shorter ones. |
| `1a/` | The translated cases, written by `loomux dev import-cases` from `1a-source/` and `1a-map.toml`. This is the directory the test suite runs. The old tools' configuration files (`.brain.toml`, `.ultraloom/policy.toml`, `.ultra-brain/config.toml`) are folded into one `.loomux/config.toml` in every staged world. |
| `1b-1-worlds/` | The state directories a 1b-1 case runs in. A world is the reference's `BRAIN_STATE_DIR` and, in the replay, both `LOOMUX_STATE_DIR` and `LOOMUX_LEGACY_BRAIN_DIR`: `registry.toml` with `{{WORLD}}/<path>` paths, writable areas under `repo-*` with `.brain.toml` or `.ultra-brain/config.toml`, the artefacts of read-only areas under `areas/<flat scope>/`, the reconcile stamp under `maintenance/last-run.txt` (year 2000 is stale, year 2999 fresh), and `qmd-fixture.json` for the fake qmd. `vault` serves search, catalog, read and neighbors; each `status-*` world shows one status line kind or, in `status-broken-register`, the register error that aborts status; `no-registry`, `broken-registry`, `missing-manifest`, `manifest-without-scope`, `manifest-wrong-type`, `damaged` and `engine-fails` each carry one broken input. |
| `1b-1-source/` | The recordings of `brain-mcp`, written by `loomux dev record-case --argv "uv run --no-sync --project <ub> brain-mcp"` with the fake qmd first on `PATH`. |
| `1b-1-map.toml` | One rule: `brain-mcp ` → `loomux brain `. |
| `1b-1/` | The translated cases. Besides the root and `areas/<name>`, the old manifests are folded in every directory the world's registry names as `{{WORLD}}/<path>`. A registry the import cannot read names no directory; the replay reports it. |
| `1b-2-worlds/` | The state directories a 1b-2 case runs in: copies of the 1b-1 worlds a case needs, each with `maintenance/last-run.txt` set to `2999-01-01T00:00:00+00:00`. The reference's daemon runs a real reconciliation before its first answer whenever that stamp is missing or older than a day, and it rewrites the stamp doing so; a stamp in the year 2999 means nothing is due, so no pass runs, the world is not touched and the daemon adds no maintenance lines to the answer. It also means `status-stamp-stale` and `status-stamp-missing` have no 1b-2 case: the daemon destroys their subject before it answers. |
| `1b-2-source/` | The recordings of `brain-mcp mcp`, written by `loomux dev record-mcp-case`. `verb/name/` holds `call`, `result`, `notes.md`, the staged `world/`, and optionally `compare`. The whole invocation, because every part of it was paid for once: `--argv "uv run --no-sync --project <ub> brain-mcp"`, `--path-prepend <dir holding qmd.exe>`, `--env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json"`, `--channel cloud`, and then `--tool`, `--arguments`, `--world`, `--out`, `--notes` and optionally `--compare outcome`. **Without the `--env` the fake answers out of an empty fixture** and every `status` case records `Collection not found`, which is word for word what the real qmd says -- a recording round was lost to exactly that. The report of task 13 in `.superpowers/sdd/` carries the full list of 54 invocations. |
| `1b-2-map.toml` | The tool renaming: `[[tool]]` rules putting the reference's five bare names into loomux's `brain_` family. |
| `1b-2/` | The translated cases, written by `loomux dev import-cases --mcp`. This is the directory the 1b-2 suite runs. |
| `2a-worlds/` | The project trees a 2a case runs `check` in: one per stack (`python-*`, `node`, `cpp`, `go-only`, `godot-*`), `mixed-config` with a Python and a Go marker under one old `[verify]`, `no-marker` with none, and the three `config-*` worlds whose old `[verify]` the old schema already refused. Each world but those three and `no-marker` carries `faketool.json`. |
| `2a-source/` | The recordings of `ultraloom check`, written by `loomux dev record-case` with the faketool executable first on `PATH` under every tool name the old chain calls. |
| `2a-map.toml` | One rule: `ultraloom check ` → `loomux check `. |
| `2a-extra-answers.json` | The answers only loomux asks for: its presets call `npx eslint`, `npx tsc`, `npx vitest`, `cmake --build` and `go test`, which the old chain did not. `loomux dev import-cases --merge-fixture` appends them to every translated world's `faketool.json` (and writes one where the world had none); the recorded fixture stays as it was. |
| `2a/` | The translated cases. The old `[verify]` of `.ultraloom/config.toml` is folded: every kind it named moves to `[verify.project]`, and every stack the world holds gets that kind switched off, because the old configuration replaced the preset. `[verify.after]` becomes `after` on the project lane, `godot_import` becomes `[verify.gdscript] import_check`; `tests` and `threshold` are dropped. |
| `2b-worlds/` | The worlds a 2b case runs in: a `.ultraloom/config.toml` carrying the old `[commit]` section, the message file the case checks, and for the calibrate cases a `faketool.json` the replay answers `git log` from. |
| `2b-source/` | The recordings of `ultraloom commit-msg`, written with `uv run ultraloom commit-msg --root <world> <world>/msg.txt`. A refusal prints to stderr only, so these cases compare the exit code (`compare = message`). |
| `2b-map.toml` | One command rule, `ultraloom commit-msg ` → `loomux check commit-msg `, and one `[[exit]]` rule mapping a refusal's exit 2 onto loomux' 1 (deviation 5 of `parity/stufe-2b.md`). |
| `2b/` | The translated cases. The old `[commit]` section moves into `.loomux/config.toml` unchanged. |

## What a case compares

### 1a and 1b-1: a command line and its stdout

`compare` in a case directory selects the comparison; a missing file means
`data`.

- **`compare = data`** — the exit code **and** the recorded `stdout`, byte for
  byte.
- **`compare = lanes`** — the exit code **and** the verdict per kind: every
  line `<kind>[/<lane>]: <state>` is read, a kind is red when one of its
  lines is red (`failed`, `timed-out`, `blocked`, `missing-tool`, `unready`,
  `error`) or loomux notes ``nothing to check for `<kind>` ``, else ok when
  one is ok, else neutral. The old chain printed one
  line per kind and wrote `unavailable` as the source of a failed kind
  (`types: failed [unavailable]`), loomux prints one line per lane; the tool
  output below the lines is not compared. Stage 2a uses it wherever the
  recording printed a report.
- **`compare = message`** — the exit code **alone**. The wording of a message
  is loomux's own: it is allowed to differ from the old tool's.

Independently of `compare`, a case that carries a `world_after/` directory also
has the staged tree compared against it after the run. A recording only gets a
`world_after/` when the run actually changed the world; no stage-1a and no
stage-1b-1 case does.

In 1b-1 every case compares data except `brain-search/fast` and
`brain-search/keyword`: there the reference runs qmd's command line and loomux
the MCP daemon, and the two rank differently. A usage or runtime error compares
data too — its stdout is empty on both sides, and that emptiness is part of the
contract. stderr is never compared; the findings of `search` and every error
wording are pinned by unit tests.

### 1b-2: a tool call and its CallToolResult

A 1b-2 case holds no command line and no stdout. `call` is one tool call --
`tool`, `arguments`, `channel` -- and `result` is what the reference answered:
the `text` of the CallToolResult, `isError`, and `rpcError` for a call the
reference never turned into a result at all. `compare` selects one of two:

- **`compare = text`** (the default) -- the text **and** `isError`.
- **`compare = outcome`** -- `isError` alone. The wording of a refusal is
  loomux's own, exactly as a `message` case's stdout is. 15 of the 54 cases
  carry it; `docs/.superpowers/parity/stufe-1b-2.md` names every one and why.

**The envelope is never compared.** The reference speaks through the Python MCP
SDK and loomux through the Go one, so `initialize` alone differs in
capabilities, in `serverInfo` and in the revision the two negotiate; comparing
that would compare two libraries and would grow again at every SDK update. The
envelope is held against the specification, in the unit tests of
`internal/serve` and `internal/bridge`. For the same reason `tools/list` is no
case: the Go SDK sorts it alphabetically (`go-sdk@v1.8.0/mcp/features.go`),
the reference lists in registration order, and a case there would measure the
SDK rather than us.

A 1b-2 case carries no `world_after`. What a run leaves in the state directory
belongs to the daemon -- its lock, its pid file, its reconciliation -- not to
the tool whose answer the case pins.

Every 1b-2 case runs on a state directory of its own (`LOOMUX_STATE_DIR`), gets
its own `loomux serve`, and ends that service through `serve.Stop` when the case
is over, so a `go test` run leaves nothing behind.

## The fake qmd (1b-1)

No recording and no replay starts qmd. `internal/dev/fakeqmd` answers from the
world's `qmd-fixture.json`: built as `qmd.exe` and put first on `PATH`, it
serves the reference's `qmd ls`, `qmd status` and `qmd query|vsearch|search
--json`; in the suite, the same fixture serves loomux's MCP port over
`httptest` and its command line port through the `Runner` seam. A world
without a fixture has an engine that knows nothing. A fixture with
`search_error` fails every search: `search`, `vsearch` and `query` exit 1 with
that stderr, and the MCP `query` answers with a JSON-RPC error
(`brain-search/engine-fails`). The MCP `query` numbers snippet lines as qmd
does (`N: `); loomux removes that prefix, so the recorded stdout still holds.
A daemon that cannot be started is covered by unit tests, not by a case: no
recording may start qmd.

## The fake tools (2a)

No recording and no replay starts a checking tool. `internal/dev/faketool`
answers from the world's `faketool.json` by the longest command-line prefix:
built as `uv.exe`, `uvx.exe`, `go.exe` … and put first on `PATH` for the
recording, and through the `checkStart` seam in the suite. There, `{loomux}`
runs in process: `check gocover` gets `--dir` of its lane and really reads
the profile the fake `go test` wrote, through a real `go tool cover -func`.
A write whose path is `{{COVERPROFILE}}` lands where the command line's
`-coverprofile=` points, because that path carries the run's ID; without the
flag nothing is written. A tool the fixture has no prefix for is not on the
`PATH` (`missing-tool`), a command line it has no answer for exits 127.

## A recorded `stdout` is evidence, not always an expectation

The `stdout` of a `message` case is kept as it was recorded — it still names
`.brain.toml`, absolute machine paths and the old tools' wording. That text is
the record of what the old binary printed on that run, not a claim about what
loomux prints. Nothing compares it.

A 1b-1 recording folds the `\r\n` Python writes into a pipe to `\n` and runs
Python with `PYTHONUTF8=1`; nothing else in its stdout is changed.

## Rules for working with the corpus

- **A recording is evidence.** Files under `1a-source/`, `1b-1-source/` and
  `2a-source/` are never edited by hand. If a case is wrong, it is *re-recorded*, never patched:
  for 1a with the old binaries (build them from the tag worktrees, put them
  first on `PATH`, run `loomux dev record-case`); for 1b-1 with the fake qmd
  rebuilt from `internal/dev/fakeqmd/_qmd` and the recording command of the
  stage plan.
- **A translated case may deviate from its recording only where the parity
  list says so.** Every such deviation has a line in
  `docs/.superpowers/parity/stufe-1a.md` or `stufe-1b-1.md`. Today there is
  one, in 1a: the exit of `hook-pre-tool-use/unreadable-payload` is 2 (the
  guard fails closed) where the old tool gave 1. 1b-1 has none. 2a keeps its
  deviations out of the corpus: `approved2a` in `internal/cli/cases_2a_test.go`
  names each case with its number in `docs/.superpowers/parity/stufe-2a.md`,
  a re-import cannot lose them, and a listed case that starts to pass fails
  the suite.
- **A re-import throws that deviation away, and it has to be re-applied by
  hand.** The deviation lives only in `1a/`; `1a-source/` still holds the
  recorded 1, and `loomux dev import-cases` copies the recording over the
  translated case. After every re-import, set
  `1a/hook-pre-tool-use/unreadable-payload/exit` back to `2`. Nothing is silent
  about it: until it is set, `internal/cli/cases_test.go` fails on that case
  with `exit code: expected 1, got 2`.
- **An import removes what no recording backs.** `loomux dev import-cases`
  prunes every case in the target that the recordings no longer hold (and the
  verb directory that loses its last case), so a case dropped from the
  recordings cannot survive behind an unchanged case count.
- **An import reads every world's manifests.** `TranslateWorld` decodes the
  old manifests of each staged world; a manifest that is not valid TOML stops
  the import, so such a manifest is no case. `brain-catalog/manifest-wrong-type`
  records valid TOML of the wrong type instead. A registry the import cannot
  read names no directory to fold and is left to the replay
  (`brain-catalog/broken-registry`).
- **The case count is pinned.** `internal/cli/cases_test.go` fails when the 1a
  corpus does not hold exactly 19 cases, `internal/cli/cases_1b1_test.go` when
  the 1b-1 corpus does not hold exactly 71, `internal/cli/cases_1b2_test.go`
  when the 1b-2 corpus does not hold exactly 54, `internal/cli/cases_2a_test.go`
  when the 2a corpus does not hold exactly 53, so a partial import cannot pass
  as parity. Adding a case means raising that number.
- **A 1b-2 recording starts the reference's daemon as `daemon run`, never as
  `daemon start`.** `daemon start` reaches `client._start_outside_job`, which
  creates the daemon through WMI on Windows; a process created that way gets
  the user's default environment, so neither the directory the recording puts
  in front of PATH nor `LOOMUX_FAKE_QMD_FIXTURE` arrives, and the daemon
  answers out of the machine's real qmd instead of the world's fixture. The
  recorder therefore starts `daemon run` as its own child and waits for
  `daemon status`. Readiness is the **absence of the phrase `no daemon`** in
  that command's stdout: it exits 0 whether or not anything is running -- it
  reports, it does not judge -- so its words are the only answer there is.
  Measured on 2026-09-18; the parity list has the numbers.
- **`search` with hits has no 1b-2 case, and cannot have one.**
  `brain/search/qmd_mcp.py` pins the engine to `localhost:8765` with no way to
  override it, so the reference's search reaches whatever qmd daemon holds that
  port on the machine -- never the world's fixture. Only the five `search`
  calls that fail or answer *before* the engine is asked are recorded. The
  parity list says which, and what stays unproven because of it.
