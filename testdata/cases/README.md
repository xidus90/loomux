# The case corpora

The cases here are the parity evidence of loomux against the tools it
replaces: recordings of the old tools, translated into loomux's own command
lines and replayed in process. Each stage keeps its own worlds, recordings,
translation table and suite.

| Stage | Recorded from | Suite | Cases |
|---|---|---|---:|
| 1a | `ulguard` and `ulinit` from ultraloom, `brain` from ultra-brain, both at the tag `loomux-1a-source` | `internal/cli/cases_test.go` | 19 |
| 1b-1 | `brain-mcp`, the Python reference of ultra-brain at the tag `loomux-1a-source` (`3cc72d2`), against a fake qmd | `internal/cli/cases_1b1_test.go` | 71 |

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

## What a case compares

`compare` in a case directory selects the comparison; a missing file means
`data`.

- **`compare = data`** — the exit code **and** the recorded `stdout`, byte for
  byte.
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

## A recorded `stdout` is evidence, not always an expectation

The `stdout` of a `message` case is kept as it was recorded — it still names
`.brain.toml`, absolute machine paths and the old tools' wording. That text is
the record of what the old binary printed on that run, not a claim about what
loomux prints. Nothing compares it.

A 1b-1 recording folds the `\r\n` Python writes into a pipe to `\n` and runs
Python with `PYTHONUTF8=1`; nothing else in its stdout is changed.

## Rules for working with the corpus

- **A recording is evidence.** Files under `1a-source/` and `1b-1-source/` are
  never edited by hand. If a case is wrong, it is *re-recorded*, never patched:
  for 1a with the old binaries (build them from the tag worktrees, put them
  first on `PATH`, run `loomux dev record-case`); for 1b-1 with the fake qmd
  rebuilt from `internal/dev/fakeqmd/qmd` and the recording command of the
  stage plan.
- **A translated case may deviate from its recording only where the parity
  list says so.** Every such deviation has a line in
  `docs/.superpowers/parity/stufe-1a.md` or `stufe-1b-1.md`. Today there is
  one, in 1a: the exit of `hook-pre-tool-use/unreadable-payload` is 2 (the
  guard fails closed) where the old tool gave 1. 1b-1 has none.
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
  the 1b-1 corpus does not hold exactly 71, so a partial import cannot pass as
  parity. Adding a case means raising that number.
