# The case corpora

Each stage directory `<stage>/` holds the cases of one stage, replayed in
process by that stage's suite: `internal/cli/cases_test.go` for `1a`,
`internal/cli/cases_<stage>_test.go` for the others (the hyphen is dropped:
`1b-1` is `cases_1b1_test.go`). `graph/` is no stage; it
holds the golden outputs of the code graph commands.

The expectations are loomux's own. Where they came from -- recordings of the
tools loomux replaced, with the scripts and translation tables that made them
-- is kept in the archive release `archive/parity-recordings`; nothing in this
tree is regenerated from it.

## A command case

A case is a directory `<verb>/<name>/` with

- `cmd`: a loomux command line; `{{WORLD}}` stands for the staged world and is
  replaced in the command, in `stdin` and in every file of `world/`,
- `exit` and `stdout`: the expected exit code and standard output,
- `world/`: the tree the command runs in,
- optionally `world_after/`: the tree the run must leave behind,
- optionally `compare`: which comparison applies (below),
- optionally `stdin`: the payload fed to the command,
- optionally `notes.md`: why the case exists. It is never compared, nor is
  `stderr`, which a failing case only shows.

## What `compare` selects

A missing file means `data`.

| `compare` | What is held against the case |
|---|---|
| `data` | the exit code and `stdout`, byte for byte |
| `lanes` | the exit code and the verdict per kind of `check` (red, ok or neutral), not the tool output below the lines |
| `message` | the exit code, not `stdout`: the wording of a message is loomux's own |
| `state` | the exit code and the `base` and `blocks` of every session file the case's `world_after` holds; it needs a `world_after` |
| `finding` | the exit code and the findings of the subagent files after the run, against `stdout` |

Besides that, the staged tree is held against `world_after` when the case has
one, and against its own `world/` when it has none -- but the second only in a
suite that passes a normalizer (the suites that compare file worlds); in the
others a case without `world_after` has no tree comparison. `state` and
`finding` cases skip the tree comparison; `message` cases keep it.

## An MCP case

An MCP case (`1b-2`) holds no command line and no `stdout`. It is a directory
with `call` (a JSON object: `tool`, `arguments` -- an object, empty if the tool
takes none -- and `channel`), `result` (`isError`, `text`, and `rpcError` for
a call that never became a result), `world/`, optionally `compare` and
`notes.md`. Here `compare` is `text` (the default: the text and `isError`) or
`outcome` (`isError` alone). The protocol envelope is never compared.

## Helpers the replay needs

- `internal/dev/fakeqmd` answers for qmd from the world's
  `qmd-fixture.json`; no replay starts qmd.
- `internal/dev/faketool` answers for the checking tools from the world's
  `faketool.json`, by the longest command-line prefix.

The case count of a stage is pinned in its suite. Adding a case means raising
that number.
