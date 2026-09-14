# The stage-1a case corpus

The cases here are the parity evidence of stage 1a: recordings of the two old
binaries (`ulguard` and `ulinit` from ultraloom, `brain` from ultra-brain, both
at the tag `loomux-1a-source`), translated into loomux's own command lines and
replayed in process by `internal/cli/cases_test.go`.

## Layout

| Path | What it holds |
|---|---|
| `1a-worlds/` | The staged project trees a case runs in: `project-writable`, `guard-allow-writable`, `guard-deny-readonly`, `plain`. Each recording copies the whole world into its own `world/`. |
| `1a-payloads/` | The hook payloads fed to a run on stdin. `{{WORLD}}` in a payload is replaced by the staged world directory. |
| `1a-source/` | The recordings of the old binaries, written by `loomux dev record-case`. `verb/name/` holds `cmd`, `exit`, `stdout`, `notes.md`, the staged `world/`, and optionally `stdin` and `compare`. |
| `1a-map.toml` | The translation table: `[[command]]` rules rewriting the head of a recorded command line (`ulguard post-edit --root …` → `loomux hook post-tool-use --host claude --root …`). Longer prefixes stand before shorter ones. |
| `1a/` | The translated cases, written by `loomux dev import-cases` from `1a-source/` and `1a-map.toml`. This is the directory the test suite runs. The old tools' configuration files (`.brain.toml`, `.ultraloom/policy.toml`, `.ultra-brain/config.toml`) are folded into one `.loomux/config.toml` in every staged world. |

## What a case compares

`compare` in a case directory selects the comparison; a missing file means
`data`.

- **`compare = data`** — the exit code **and** the recorded `stdout`, byte for
  byte.
- **`compare = message`** — the exit code **alone**. The wording of a message
  is loomux's own: it is allowed to differ from the old tool's.

Independently of `compare`, a case that carries a `world_after/` directory also
has the staged tree compared against it after the run. A recording only gets a
`world_after/` when the run actually changed the world; no stage-1a case does.

## A recorded `stdout` is evidence, not always an expectation

The `stdout` of a `message` case is kept as it was recorded — it still names
`.brain.toml`, absolute machine paths and the old tools' wording. That text is
the record of what the old binary printed on that run, not a claim about what
loomux prints. Nothing compares it.

## Rules for working with the corpus

- **A recording is evidence.** Files under `1a-source/` are never edited by
  hand. If a case is wrong, it is *re-recorded* with the old binaries (build
  them from the tag worktrees, put them first on `PATH`, run
  `loomux dev record-case`), never patched.
- **A translated case under `1a/` may deviate from its recording only where the
  parity list says so.** Every such deviation has a line in
  `docs/.superpowers/parity/stufe-1a.md`. Today there is one: the exit of
  `hook-pre-tool-use/unreadable-payload` is 2 (the guard fails closed) where
  the old tool gave 1.
- **The case count is pinned.** `internal/cli/cases_test.go` fails when the
  corpus does not hold exactly 19 cases, so a partial import cannot pass as
  parity. Adding a case means raising that number.
