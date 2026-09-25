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
| 2c | `ultraloom hook stop`, `hook subagent-start` and `hook subagent-stop` at the tag `loomux-1a-source` (`9d01a60`), against fake tools and measured Claude Code payloads | `internal/cli/cases_2c_test.go` | 15 |
| 3a | `brain-mcp reconcile`, `reindex`, `embed` and `init` of ultra-brain at the tag `loomux-3-source` (`3cc72d2`), against a fake qmd | `internal/cli/cases_3a_test.go` | 28 |
| 3b | `brain-mcp cases`, `case` and `approve` of the same reference at the same tag, against a fake qmd | `internal/cli/cases_3b_test.go` | 24 |
| 4a-2 | `brain-mcp hook install`, `status` and `remove` of ultra-brain at the tag `loomux-3-source` (`3cc72d2`) | `internal/cli/cases_4a2_test.go` | 14 |

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
| `2c-worlds/` | The project trees a 2c case runs a session hook in: `stop-*` for the eight stop cases, `subagent-start-*` and `subagent-stop-*` for the two subagent hooks. Every one of the fifteen declares two commits in `git.toml` and carries the old session file under `.ultraloom/hooks/`. Six of the eight stop worlds add a `[worktree]` that dirties a tracked file; `stop-unchanged` adds nothing, and `stop-untracked-only` instead holds a `b.py` that no commit tracks. Every subagent world declares a `[remote.push]` and no stop world does -- the subagent hooks measure refs, the stop gate measures the tree. Only the stop worlds hold a `faketool.json`: the subagent hooks start no checking tool. |
| `2c-payloads/` | The hook payloads a 2c recording feeds on stdin. Measured on 2026-09-20 from a running Claude Code 2.1.276 session, not written by hand; `2c-payloads/README.md` names what was measured, what the measurement proves about `agent_id` and `session_id`, and which two payloads (`subagent-no-agent.json`, `not-json.txt`) are derived from it for the error cases. |
| `2c-source/` | The recordings of ultraloom's session hooks, written by `loomux dev record-case` with the faketool executable first on `PATH` under every tool name the old stop chain calls. |
| `2c-map.toml` | Three rules, one per verb: `ultraloom hook stop` → `loomux hook stop --host claude`, and the same for `subagent-start` and `subagent-stop`. The verbs are unchanged; the host is a flag in loomux, and `loomux init` writes it into the settings. |
| `2c/` | The translated cases. The old session state is folded into loomux's: `.ultraloom/hooks/<id>.json` becomes `.loomux/state/hooks/<id>.json` with `base` and `blocks`, its `snapshots` map -- an agent id against the raw `git ls-remote` text the old hook kept -- becomes one file per subagent under `.loomux/state/hooks/<id>/agents/<agent>.json`, with those lines parsed into a snapshot object, and the old marker `.claude/.no-verify` becomes `.loomux/no-verify`. |
| `3a-worlds/` | The state directories a 3a case runs in, laid out as in 1b-1: the world is the reference's `BRAIN_STATE_DIR` and, in the replay, both `LOOMUX_STATE_DIR` and `LOOMUX_LEGACY_BRAIN_DIR`. `vault-*` hold a writable area `repo-a` (a source, a wiki page citing it, a register) and, for one case each, a read-only area, a second area, a standing case, a broken case file or a manifest left out; `embed-*` a registry and a fixture; `area-*` a repository `repo-new` to onboard. Every world but `embed-no-qmd` carries `qmd-fixture.json`, and qmd's own configuration is written below `xdg/`. Four worlds make `repo-a` a repository: `git.toml` names it with `dir = "repo-a"`, so the repository is the area and not the state directory around it, and every path the declaration names is relative to it. `vault-changed-baseline` commits the source at the state the register names and changes it in `[worktree]`; `vault-changed-stale-head` commits a third state on top, so HEAD is no state the register names. `vault-merge` and `vault-merge-standing` carry a page with `realization: planned` and a merge event in `maintenance/merge-events.tsv`: in `vault-merge` over two commits after the base (`{{COMMIT:1}}`, `{{COMMIT:3}}`), whose paths land in the reverse of their byte order, in `vault-merge-standing` over one (`{{COMMIT:1}}`, `{{COMMIT:2}}`) and with a standing source case on that page. |
| `3a-source/` | The recordings, written by `loomux dev record-case --argv <ultra-brain>/.venv/Scripts/brain-mcp.exe` with the fake qmd first on `PATH`, `LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json`, `XDG_CONFIG_HOME={{WORLD}}/xdg`, and `LOCALAPPDATA`, `XDG_STATE_HOME` and `XDG_CACHE_HOME` pointed at an empty sandbox; `embed/no-qmd` runs with `PATH=C:/Windows/System32` instead. The recorder sets `BRAIN_STATE_DIR` to the staged world itself. `reindex` and `area-add` are recorded with `--compare message`, `reconcile` and `embed` without it. `docs/.superpowers/parity/stufe-3a-orakel/record_all.sh` makes every call, through `record.sh` beside it. |
| `3a-map.toml` | Four command rules (`brain-mcp init ` → `loomux area add `), `manifests = "verbatim"` and one `[[manifest_key]]` rule, `merge_branch` → `branch`. |
| `3a/` | The translated cases. A manifest that nothing else is folded into is moved byte for byte (`manifests = "verbatim"`), because `area add` writes one and the replay holds the two files against each other. `area-add/known-scope/world_after` keeps `repo-new/.ultra-brain/config.toml` under its old name: the import folds only the directories a world's registry names, and the reference wrote that manifest without registering `repo-new`. The replay lists it as missing, which is what loomux, refusing the scope before any write, leaves behind. |
| `3b-worlds/` | The state directories a 3b case runs in, laid out as in 3a: `repo-a` is the vault and, in all but `approve-no-repo`, a repository (`git.toml` with `dir = "repo-a"`), with a register, a wiki holding `page.md`, `audit.md` and `log.md`, and the review centre `review/`. `approve-success` is the base every other world is copied from: its case `a-2026-09-21-5bd8` is the one `reconcile/changed-source-baseline` of 3a opened, moved one day back (a case id dated on the run day would be folded to `{{TODAY}}` on one side only), with its `package.md` as the reference wrote it -- the package 3a holds byte-equal to `maintenance.RenderPackage` -- and a `proposal.md` written by hand. The approve worlds commit the case directory, so its removal lands in the commit; the `cases` and `case` worlds leave it untracked. `approve-rebase` writes `.git/rebase-merge/head-name` through `[worktree]`, the one place a marker in the git directory survives the recording, which skips `.git` when it copies a world. |
| `3b-source/` | The recordings, written by `docs/.superpowers/parity/stufe-3b-orakel/record_all.sh` through `record.sh` beside it, under the environment of 3a. No case carries `compare`: stdout is what `cases` and `case` are for, and the commit id `approve` prints is folded. Every `approve` case with a repository is recorded with `--git-after`, the refusals included, because `git.after` is what proves HEAD did not move. |
| `3b-map.toml` | Three command rules (`brain-mcp cases` before `brain-mcp case`, `brain-mcp approve` without a trailing space, for `approve/empty-args`), `manifests = "verbatim"` and one `[[stdout]]` rule, `brain case --package ` → `loomux case --package `. |
| `3b/` | The translated cases. |
| `4a2-worlds/` | The state directories a 4a-2 case runs in, laid out as in 3a: `registry.toml` names `project/a` at `{{WORLD}}/repo-a`, whose `.brain.toml` consents with `[maintenance] on_merge = true` and `branch = "main"`. `git.toml` makes `repo-a` a repository with one empty commit; the manifest stays untracked, because the import moves it under its new name. `hook-consenting` is that and nothing more; `hook-no-consent` says `on_merge = false`; `hook-no-repository` has no `git.toml`; `hook-empty` registers no area. The hook file lies in `.git/hooks`, which a recording does not copy, so a world that needs one writes it through `[worktree]`: `hook-foreign` a `post-merge` of the user's own without a marker, `hook-own-earlier` the reference's hook as `brain-mcp hook install` renders it, and `hook-installed` the same hook with its record in `maintenance/hooks.tsv`. `hook-orphaned` records a hook for `project/gone`, which the registry does not name. A record carries the reference's five fields, so both tools read it. |
| `4a2-source/` | The recordings of `brain-mcp hook`, written by `docs/.superpowers/parity/stufe-4a-2-orakel/record_all.sh` through `record.sh` beside it, under the environment of 3a. `install`, `status` and `remove` each where they show something; a case whose reference prints its German sentence for "no area, no hook" carries `compare = message`. |
| `4a2-map.toml` | One command rule, `brain-mcp hook` → `loomux merge-hook` (`hook` is the host hooks' namespace in loomux), and `manifests = "verbatim"`. |
| `4a2/` | The translated cases. The suite compares `maintenance/hooks.tsv` over its first three fields: the reference also records the branch and the event path it baked into the hook, loomux bakes nothing in. |

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

### 3a: stdout or the exit code, and a normalized file world

The `reconcile` and `embed` cases are `data` cases: the counts of a pass and
the cases it opened are its result, so stdout is compared byte for byte
(`embed` prints nothing, and that is compared too). The `reindex` and
`area add` cases are `message` cases: their exit code is compared, their
wording is not. What all four commands are for is what they leave on disk, so
every replay also compares the whole world, through `cases.RunCaseWith` and
`cases.NormalizeState`:

- A case **without** a `world_after` is held against its `world`: the recorder
  writes a `world_after` only where the run changed something, so its absence
  says nothing changed, and a refusal that writes after all fails.
- Both sides are normalized before they are compared, and only in three
  places: the pass's stamp (`maintenance/last-run.txt`, and that exact stamp
  wherever a case file or package repeats it) becomes `{{NOW}}` and the day of
  a case id `{{TODAY}}`, on stdout as well; a nanosecond count in the
  modification time column of `maintenance/*/stats.tsv` becomes `{{MTIME}}`;
  a doc id an index run minted, i.e. one no register of the `world` holds and
  one row alone carries, becomes `{{DOCID}}`, and such a register's rows are
  sorted. A stamp of any other shape than the reference's, a time in seconds
  and an id minted twice are left to fail.
  `docs/.superpowers/specs/2026-09-19-loomux-stufe-3-design.md` holds the
  rule and its reasons.

`expected3a` in `internal/cli/cases_3a_test.go` lists every case whose replay
differs, with the exact list of mismatches and the row of
`docs/.superpowers/parity/stufe-3a.md` it belongs to. A
case not listed must pass; a listed one must report exactly its list. A file
listed as `formatOnly` differs in its bytes and is decoded on both sides and
held equal, so a format difference cannot cover a change of content.

### 3b: stdout, the file world and the commit

Every 3b case is a `data` case, replayed as 3a is, through
`cases.RunCaseWith` and `cases.NormalizeState`. `NormalizeState` folds four
things more for an approval: the reviewer `human:<account>` in `audit.md` and
on a page's `by:` lines, the stamp of the new audit block wherever it stands
in a file the run changed, its day in the new `log.md` line, and the commit id
after `committet als` on stdout. An `approve` case also compares
`repo-a/git.after`, written after the run on both sides: the subject of HEAD,
its author and committer, the paths HEAD tracks, the paths whose working-tree
content differs from HEAD (`git diff --name-status HEAD`) and the status
lines. The diff lines are the proof of what the commit holds: an approval
commits through a scratch index and leaves the user's index at the old
state, so the status reads `MM` whether HEAD holds the working tree or not.

`expected3b` in `internal/cli/cases_3b_test.go` lists five cases, none for a
difference in behaviour: the scratch index `maintenance/index` (a git index
with the stat data of its run), the index formats of 3a, and the register the
technical update rewrites. That register hashes the page and `audit.md` the
approval stamped, so its bytes cannot agree; the suite holds each side's
`content_hash` against that side's own file and compares the rows without it
(`rehashed`). `docs/.superpowers/parity/stufe-3b.md` names the rows.

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

### 2c: what the hook decided, not what it said

A session hook writes no stdout worth comparing: the stop gate speaks on
stderr, and the subagent hooks speak through the files the next turn end
reads. Two comparison classes pin what they decided instead.

- **`compare = state`** — the exit code **and**, for every session file the
  recording's `world_after` holds, the `base` the gate measures from and the
  `blocks` it counted. Nothing else in the tree is compared: the green tree,
  the lane output and every other file loomux writes are its own.
- **`compare = finding`** — the exit code **and** the findings of the
  subagent files after the run: every `finding` line of every
  `.loomux/state/hooks/<id>/agents/<agent>.json`, each with the
  `subagent <id>: ` prefix Python printed, against the recorded `stdout`, both
  sides sorted. The file carries the line without that prefix; the stop gate
  puts it there when it delivers, and the comparison puts it there when it
  compares.

Both read only loomux's state, so both always need a `world_after`: a git
world has its `{{COMMIT:<n>}}` tokens replaced at every staging, and the world
itself still holds the tokens where the SHAs belong. The two error cases
(`hook-stop/bad-payload`, `hook-subagent-start/no-agent`) compare `message`
instead -- the exit code alone, plus the whole staged tree against
`world_after`, because a hook that refuses its payload must leave the world
untouched.

The stop gate's tools answer from the world's `faketool.json` through the
`stopHook` seam, which the suite points at `hooks.RunStop` with the same fake
`Start` and `Look` that `check` gets. A world without a fixture is a subagent
world: it starts no tool, and the seam stays as it is.

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

For stage 3a the fake also answers `qmd update` and `qmd embed`, and appends
each such call it accepted to `qmd-calls.log` beside the fixture. The two
calls change the engine and print nothing either side reads, so without the
log a command that never made them would replay exactly like one that did;
with it, the call is part of the world both sides are compared on. Reads leave
no trace.

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

- **A recording is evidence.** Files under `1a-source/`, `1b-1-source/`,
  `2a-source/`, `2c-source/`, `3a-source/`, `3b-source/` and `4a2-source/` are never edited by hand. If a case is wrong, it is *re-recorded*, never patched:
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
  the suite. 2c does the same with `approved2c` in
  `internal/cli/cases_2c_test.go` and `stufe-2c.md`; it names two cases, and a
  `-v` run logs what each one actually differs in, so a deviation that starts
  to differ for another reason does not hide behind its entry.
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
  when the 2a corpus does not hold exactly 53, `internal/cli/cases_2c_test.go`
  when the 2c corpus does not hold exactly 15, `internal/cli/cases_3a_test.go`
  when the 3a corpus does not hold exactly 28, `internal/cli/cases_3b_test.go`
  when the 3b corpus does not hold exactly 24, `internal/cli/cases_4a2_test.go`
  when the 4a-2 corpus does not hold exactly 14, so a partial import cannot pass
  as parity. Adding a case means raising that number.
- **A 3a recording never touches the machine's state.** The recorder sets
  `BRAIN_STATE_DIR` to the staged world and loomux's own two variables to an
  empty sandbox, and `XDG_CONFIG_HOME` keeps qmd's configuration inside the
  world: without it, `reindex` on either side rewrites the `index.yml` of the
  machine's real qmd. The replay sets `LOOMUX_STATE_DIR`,
  `LOOMUX_LEGACY_BRAIN_DIR` and `XDG_CONFIG_HOME` to the staged world before
  every run.
- **A recording and its replay run git under one environment.**
  `cases.GitEnv` sets `GIT_CONFIG_NOSYSTEM=1` and points `HOME` at
  `{{WORLD}}/.no-git-home`, which nothing creates. The recorder adds both to
  every recorded process, and `TestCases3a` and `TestCases2c` set them before
  every run, so neither side reads the machine's system or user
  configuration. `XDG_CONFIG_HOME` lies in the world in 3a; 2c sets it empty.
  `GIT_CONFIG_GLOBAL` would not do: the reference and `gitenv` both strip it
  before git starts. On Windows `HOME` reaches git alone; on POSIX it is also
  the home both tools read.
- **A git world may put its repository below the world.** `dir` in `git.toml`
  names the directory, and `InfraPath` keeps a `.git` or `.origin.git` at any
  depth out of the tree a replay compares and out of the corpus. The SHAs of
  a git world are the same on every build -- fixed identity, dates,
  configuration and object format (`--object-format=sha1`, whatever
  `GIT_DEFAULT_HASH` says) -- so a `world_after` may hold them written out:
  the recorder does not turn them back into `{{COMMIT:<n>}}`.
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
