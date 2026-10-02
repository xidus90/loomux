# Benchmarks

Chronological performance measurements for loomux, cold and warm.

Entries before 2026-09-26 carry the command names of their day: `dev bench-hooks` is now `dev bench hooks`, and `dev bench` is now `dev bench repos`.

## 2026-09-14 16:05 — Baseline Measurement (Predecessor Binaries)

Measured on Windows x86_64, warm median over 5 runs. Basis for the Loomux fusion target budgets.

| Case | Predecessor Runtime | Time | Loomux Target Budget |
|---|---|---:|---|
| `ulguard --root` (PreToolUse, Edit) | Go (`ulguard.exe` 6.0 MB) | 33 ms | < 35 ms |
| `brain guard` | Go (`brain.exe` 13.7 MB) | 72 ms | unified in `hook pre-tool-use` |
| Both guards parallel (as seen by edit) | Go + Go | 73 ms | < 35 ms (unified in-process) |
| Both guards sequential (CPU time) | Go + Go | 114 ms | < 35 ms |
| Go cold-start floor (`brain version`, `ulinit --version`) | Go (`ulinit.exe` 8.5 MB) | 32–34 ms | ~32 ms |
| `ulguard hook session-start` | Go | 67 ms | < 100 ms (Stage 2) |
| `ultraloom hook subagent-start` | Python (with/without `uv run`) | 706–730 ms | < 100 ms (Go port) |
| `brain-mcp --help` | Python | 898 ms | < 35 ms (Go port) |
| `brain-mcp status` / `brain status` | Python / Go | 3,113 ms / 36 ms | < 35 ms |
| `brain-mcp search` / `brain search` | Python / Go | 7,773 ms / 242 ms | < 50 ms |

### Key Findings
1. Binary size does not dictate start time: `brain.exe` (13.7 MB) starts in 32 ms, `ulinit.exe` (8.5 MB) in 34 ms, `ulguard.exe` (6.0 MB) in 33 ms.
2. Start-time floor is governed strictly by package `init()` execution and embedded data parsing, not binary bytes.
3. Python hooks and CLI bridges incurred 700–3,000 ms overhead, eliminated by the unified Go binary.

## 2026-09-15 01:20 — The Merged Guard Against the Two It Replaces

Repository `loomux`, worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1a`, branch
`sdd-1a`, commit `8ebdc93` plus the uncommitted tree of Task 15 — this task adds
the measuring tool and the entry, it does not change the hook path.

**Goal.** Show what `loomux hook pre-tool-use` costs against the target value of
72 ms (the parity value of both old guards as an edit met them), and decompose
what the start floor and the write barrier spend.

**Method.** `loomux dev bench-hooks testdata/bench/1a-hooks.json -n 20`: one cold
run per case, then 20 warm ones; the table reports cold, warm median, warm min,
warm max and the exit codes. Payload: an `Edit` on `README.md` of this worktree
(`testdata/bench/edit-readme.json`). Run with
`LOOMUX_STATE_DIR=$TEMP/loomux-bench-state`, holding a `registry.toml` with one
`[[area]]` for this worktree (`workspace = true`); there is no `.loomux/config.toml`
here (a missing file is an empty policy), so the real pilot configuration of Task 16
will add one TOML parse this measurement omits. Binaries measured:
`bin/loomux.exe`, built from this tree with Go 1.27, and the predecessors
`C:/Users/micro/AppData/Local/Temp/loomux-old/{ulguard,brain}.exe`, built from the
tag worktrees `loomux-1a-source` of `ultraloom` and `ultra-brain` — the tagged
versions, not whatever sits on the PATH.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| loomux hook pre-tool-use (Edit on README.md) | 30.4 ms | 24.5 ms | 23.5 ms | 27.5 ms | [0] |
| ulguard --root (old policy guard) | 26.1 ms | 23.7 ms | 23.4 ms | 24.5 ms | [0] |
| brain guard (old write barrier) | 29.0 ms | 27.3 ms | 26.5 ms | 34.5 ms | [2] |
| both old guards in parallel (as an edit meets them) | 37.5 ms | 30.5 ms | 29.5 ms | 32.8 ms | [0 2] |
| both old guards sequentially (CPU time) | 52.6 ms | 51.1 ms | 50.1 ms | 57.1 ms | [0 2] |
| loomux hook session-start | 26.7 ms | 25.0 ms | 24.0 ms | 30.5 ms | [0] |
| loomux version (start floor) | 28.1 ms | 23.6 ms | 23.0 ms | 24.9 ms | [0] |

### Reading

1. **The target value holds with room to spare.** `hook pre-tool-use` is 24.5 ms
   warm against a target of 72 ms, and 0.9 ms above its own start floor
   (23.6 ms): the merged guard decides in under a millisecond, everything else
   is process start.
2. **The comparison is not the 2026-09-14 comparison, and the exit codes say why.**
   `brain guard` ends with 2 here because this repository is not in the old
   registry; it refuses after reading the registry and does less than it did on
   2026-09-14, where the same binary needed 72 ms. Measured on a registered area
   (`ultra-brain/docs/wiki/index.md`, exit 0) the same tagged `brain.exe` needs
   26.7 ms warm today against its own floor of 24.3 ms (`brain version`). The
   72 / 33 / 73 / 114 ms of the baseline were measured over 5 runs on a colder
   machine; against the numbers here the fusion still removes one whole process:
   51.1 ms sequential and 30.5 ms parallel for two binaries against 24.5 ms for one.
3. **Start time: no `init` of loomux code is over 1 ms.** `GODEBUG=inittrace=1
   bin/loomux.exe version` names exactly two lines above the threshold, and
   neither is ours: `github.com/BurntSushi/toml/internal` with 18–22 ms clock
   (1,673 allocations — the package resolves the local time zone at start) and
   `net` with 0.5–1.0 ms. The largest loomux line is
   `github.com/xidus90/loomux/internal/config` with 0.49 ms. The start-time rule
   of the specification therefore holds. Measured against it: an empty Go
   `main` starts on this machine in 4.5 ms warm, so of the ~23.6 ms floor some
   4.5 ms are process start and the remaining ~19 ms are that one dependency's
   `init` — it resolves the local time zone (`time.Now().Zone()` in
   `internal/tz.go`) and loads the operating system's time-zone data for it.
   The same 21 ms appear in the old `brain.exe`.
4. **The barrier's own cost, decomposed.** `LOOMUX_BENCH_REGISTRY=<copy>
   LOOMUX_BENCH_TARGET=<file in a registered area> go test
   ./internal/brain/guard/ -run '^$' -bench
   BenchmarkDecideAgainstTheRealRegistry -benchtime 50x -cpuprofile …`
   against a copy of the real state directory
   (`%LOCALAPPDATA%\brain`, 10 areas) measures **916,728 ns/op** for one
   `Decide`. The profile (`go tool pprof -top`) puts all of it in Windows path
   canonicalisation: `guard.writableRoots` → `finalPath` 66.7 % (one
   `CreateFile` + `GetFinalPathNameByHandle` per registered root), the
   `resolvePath` of the target itself 33.3 %, and `runtime.cgocall` carries
   100 % of the flat samples — the TOML reads of registry and manifests do not
   reach a sample at all. The suspicion noted before the measurement, one
   `git rev-parse --git-common-dir` per target in `askGit`, **did not appear**:
   `sameRepository` reaches it only for a target under a directory that carries
   a manifest and its own `.git` and is not the registered root itself, which
   this payload never is.

No target budget for stage 2 is set here beyond the three items now named;
the lever they point at is the per-root canonicalisation in `writableRoots`.

## 2026-09-15 12:00 — The Exec Hook Against a Function Hook (Claude Mods)

Repository `loomux`, main worktree, commit `311d5b2`, clean tree — this
measurement changes no code in the repository; the probe mod, the probe binaries
and the stand-in daemon live in a scratch directory.

**Goal.** Decide whether the function hooks proposed in
[anthropics/claude-code#91870](https://github.com/anthropics/claude-code/issues/91870)
("Claude Mods") are worth an adapter. They run inside Claude Code's own process,
so they pay no process start. The question is not whether that is faster — it is
by construction — but how much of today's 28 ms is the spawn we would remove,
what the adapter to a long-lived `loomux` would cost in its place, and whether
that spawn can be made cheap in Go instead.

**What the feature is, read from the binary and the published declarations, not
from the issue.** Claude Code 2.1.272 carries it behind
`CLAUDE_CODE_ENABLE_FUNCTION_HOOKS=1` (rollout flag
`tengu_plugin_hooks_modules`). A mod is an ordinary plugin whose
`hooks/hooks.json` names a module exporting `register(on)`; hooks are
`($, e, next)` and nest in registration order. Two facts shape any adapter, both
from `anthropics/claude-code:mods/types/claude-code.d.ts`:

- `tool.check` is the write barrier's seat exactly: it takes `{ tool, input,
  tool_use_id }` and returns `{ decision: 'allow' | 'ask' | 'deny', reason?,
  rule? }` — the same verdict `hook pre-tool-use` writes today, plus `ask`,
  which the exec hook cannot express.
- A hook reaches a long-lived process two ways: `$.http.fetch` (a loopback
  server we would run) or `$.mcp.call(server, tool, args)` (a stdio MCP server
  Claude Code spawns and owns, no port). `$.process.run` is a spawn again.
  Only `$.http.fetch` is measured below.

**Method.** Three measurements, each of one link in the chain.

1. Spawn, as the hook pays it today, and what it is made of:
   `loomux dev bench-hooks -n 20` in one pass over five cases — the hook, the
   start floor, an empty Go `main`, a `main` whose only import is
   `github.com/BurntSushi/toml` v1.6.0, and a `main` whose only statement is
   `time.Now().Zone()`. All five built with Go 1.27, the hook run in the main
   worktree with the pilot `.loomux/config.toml` in place and no
   `LOOMUX_STATE_DIR` override, payload an `Edit` on this repository's
   `README.md`. One cold run per case, then 20 warm.
2. Dispatch through the fold, in situ: a probe mod (`register` hooks
   `tool.check`, asks over `$.http.fetch`, falls through with `next(e)`) run
   under `claude plugin test`, 500 dispatches of `$.tool.check`, the fetch
   answered from memory by a hook the test seats beneath. The test counts the
   fetches and asserts 501, so a silently skipped hook cannot be mistaken for a
   fast one — the first attempt at this measurement did exactly that, and the
   kit reported `no implementation for http.fetch` only once the count was
   asserted.
3. Transport: 200 POSTs from a client to a Go `net/http` server on
   `127.0.0.1:47613` that answers the verdict as the barrier would. Measured
   from Node 24.14.1 outside Claude Code, because a hook's own transport was not
   measurable in the test kit (there is no real `http.fetch` beneath it). Two
   unmeasured factors sit between this number and the real one: Claude Code runs
   Bun, and `$.http.fetch` goes through the engine's noun, not bare `fetch`.

| case | cold (1st run) | warm median | warm min | warm max |
|---|---:|---:|---:|---:|
| `loomux hook pre-tool-use` (Edit on README.md) | 36.1 ms | 28.0 ms | 27.1 ms | 30.0 ms |
| `loomux version` (start floor) | 27.5 ms | 27.5 ms | 26.0 ms | 30.0 ms |
| empty Go `main` (spawn floor) | 10.6 ms | 8.0 ms | 7.7 ms | 10.0 ms |
| Go `main` importing only `BurntSushi/toml` | 93.0 ms | 27.5 ms | 26.0 ms | 32.5 ms |
| Go `main` resolving only the local zone | 78.1 ms | 26.7 ms | 25.6 ms | 29.4 ms |
| mod `tool.check` through the fold, fetch from memory | — | 0.26 ms | 0.20 ms | 2.14 ms |
| loopback round trip to a Go daemon (Node client) | 22.4 ms | 0.34 ms | 0.22 ms | 1.53 ms |

The last two rows come from measurements 2 and 3 and are not part of the
`bench-hooks` pass; the first five are one pass, one machine state.

### Reading

1. **The function hook plus an adapter is on the order of 0.6 ms against
   28.0 ms.** That is the sum of two separately measured halves — dispatch
   0.26 ms in situ, transport 0.34 ms outside — so treat it as an estimate, not
   a measurement of one path. Per guarded tool call it is ~27 ms saved. Read
   item 2 before crediting the mod with it.
2. **19 of those 28 ms are Go resolving the local time zone, and no mod is
   needed to remove them.** A `main` whose only statement is `time.Now().Zone()`
   costs **26.7 ms** against **8.0 ms** for an empty one: 18.7 ms, on the
   Windows path that reads the zone out of the registry. The import of
   `BurntSushi/toml` costs the same 27.5 ms and nothing beyond it, and
   `GODEBUG=inittrace=1 bin/loomux.exe version` names
   `github.com/BurntSushi/toml/internal` at 22 ms clock, 1,673 allocs — that
   package resolves the zone in its `init`. **So the lever is not "swap the TOML
   parser"; it is "no local time on the hook path at all".** Any package, and
   any log line formatting a local timestamp, buys the same 19 ms back, once per
   process. An exec hook that never touches the local zone would land near 9 ms,
   which is ~19 of the ~27 ms gap closed in Go, without a line of TypeScript and
   without giving up the portable path. Only the remaining ~8 ms of Windows
   process creation needs an in-process hook.
3. **The decision itself is still below the noise of the floor.** `hook
   pre-tool-use` at 28.0 ms against `version` at 27.5 ms is 0.5 ms apart, with
   warm ranges that overlap almost completely (27.1–30.0 against 26.0–30.0); in
   a second pass the hook came out *below* the floor. The 24.5 ms of
   2026-09-15 01:20 is not a regression against this: that run was a different
   worktree without `.loomux/config.toml` (one TOML parse fewer) on a quieter
   machine, and `version` moved with it (23.6 → 27.5 ms). The floor is the whole
   story, in both runs.
4. **Cold is where a daemon is worst, and it is a one-off.** The first loopback
   call costs 22.4 ms — client warm-up and the connection. A session pays it
   once; the exec hook pays its cold price on the first edit and ~28 ms on every
   one after.
5. **What the mod buys that milliseconds do not.** `ask` as a verdict, a reason
   rendered in the transcript rather than returned as an error string, and
   `ui.*` for showing the barrier's state. Against that: it is Claude-Code-only
   (the `.githooks/pre-commit` gate and every other host still need the exec
   path), the API may change between releases without notice, and a second
   language enters a tree whose design is one Go binary.

**The lever this points at is item 2, not the mod.** Keeping the local time zone
off the start path is a Go change in this repository, measurable with the tool
already here, and it is worth more than the adapter it would make less
attractive.

## 2026-09-15 15:39 — The Write Barrier in a Linked Worktree

Repository `loomux`, worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1`, branch
`barrier-worktrees`, commit `d8bfad2`.

**Goal.** Show what opening linked worktrees costs: a write in a worktree with no
registry entry of its own, before the change (refused) and after it (allowed), and
a write in the main checkout before and after, which the change must leave untouched.

**Method.** `loomux dev bench-hooks testdata/bench/barrier-worktrees.json -n 20`:
one cold run per case, then 20 warm ones. `LOOMUX_STATE_DIR` points at a copy of the
registry that registers only the main checkout (`workspace = true`). Binaries:
`before.exe` built from `b55ae6d`, `after.exe` built from the commit
above, both with Go `go1.27.0 windows/amd64`.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| before: Write in linked worktree, no registry entry | 180.4 ms | 66.8 ms | 62.0 ms | 84.6 ms | [2] |
| after: Write in linked worktree, no registry entry | 177.3 ms | 65.3 ms | 62.4 ms | 69.8 ms | [0] |
| before: Edit on README.md in main checkout | 33.2 ms | 31.9 ms | 29.0 ms | 40.4 ms | [0] |
| after: Edit on README.md in main checkout | 34.0 ms | 31.0 ms | 27.1 ms | 32.5 ms | [0] |

### Reading

1. **Worktree: opening it costs nothing measurable.** After against before is
   65.3 ms against 66.8 ms warm (1.5 ms lower) and 177.3 ms against 180.4 ms cold
   (3.1 ms lower). The warm range after (62.4–69.8) lies inside the one before
   (62.0–84.6). The two runs do different work: before, the refusal goes all the
   way to the message and looks up the review centre on the way; after, the
   barrier tries one `.git` read per parent directory and reads the three pointer
   files (`.git`, `gitdir`, `commondir`) only at the worktree root.
2. **Main checkout: untouched, as it has to be.** After against before is 31.0 ms
   against 31.9 ms warm (0.9 ms lower) and 34.0 ms against 33.2 ms cold (0.8 ms
   higher), with warm ranges that overlap (27.1–32.5 against 29.0–40.4). That
   difference is noise: `Decide` returns before the worktree lookup, because the
   target lies inside a registered tree.
3. **Against the target of 72 ms, and what the gaps between the rows mean.** Warm,
   both writes stay under the target: the worktree write at 65.3 ms (6.7 ms under),
   the main-checkout write at 31.0 ms (41.0 ms under). The cold values of the two
   groups are not comparable. The worktree cases ran first, so their cold run
   (180.4 and 177.3 ms) is also the first start of each freshly built binary; the
   main-checkout cold runs (33.2 and 34.0 ms) reused the cached binaries. The cold
   figure therefore says nothing about the worktree write alone. The warm gap of
   about 34–35 ms between the worktree and main-checkout rows (66.8 against 31.9
   before, 65.3 against 31.0 after) is present in both binaries, so this change
   does not cause it. By the code path it is consistent with the two `git rev-parse`
   calls that `declaredWikiRoot` → `sameRepository` makes for a worktree whose
   `.loomux/config.toml` names a registered scope; the main checkout skips them
   because its path equals the registered one (`path.go:468`). This pass did not
   measure that attribution.
   The entry of 2026-09-16 19:32 below measures it.

## 2026-09-16 19:10 — The Brain Data Commands on the Real Registry

Repository `loomux`, worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1`, branch
`sdd-1b-1`, commit `9ca6364` plus the uncommitted tree of Task 15 — this task adds
the measurement cases, three benchmarks and this entry; it does not change a command.

**Goal.** Show what `loomux brain search` costs warm and cold per profile against the
target of ≤ 150 ms for `--profile fast` warm end to end (spec: 63–83 ms qmd, 5–17 ms
probe and handshake, ~35 ms Go start floor), with the share for reading every identity
register named on its own; what `loomux brain status` costs; and whether moving the
brain packages in added start time.

**Method.** `loomux dev bench-hooks testdata/bench/1b-1-brain.json -n 20` against the
real state directories (`%LOCALAPPDATA%\loomux` with 11 areas, `%LOCALAPPDATA%\brain`
for read-only artefacts and the reconcile stamp; neither `LOOMUX_STATE_DIR` nor
`LOOMUX_LEGACY_BRAIN_DIR` set), query `latenz`, after one warming call per profile:
the cold column is a cold process against a warm qmd daemon. The cold daemon is
measured separately: the daemon stopped, then one timed `brain search` per profile,
three times; each of the nine runs printed the warming note, so each started the daemon
itself. The register share: `go test ./internal/brain/search/ -run '^$' -bench 'OfTheRealRegistry|WithoutTheEngine' -benchtime 50x -benchmem`
with `LOOMUX_BENCH_REGISTRY`/`LOOMUX_BENCH_LEGACY` on the same directories. Start
time: `GODEBUG=inittrace=1 loomux --version`, three runs each, on the binary of the
commit before Task 3 (`aa945cb^`) and on `bin/loomux.exe`. Machine: AMD Ryzen 7 9800X3D,
Go `windows/amd64`, GOMAXPROCS 16, qmd 2.8.3 on CUDA (loomux's default backbone;
neither `QMD_LLAMA_GPU` nor `QMD_FORCE_CPU` set).

**What makes these numbers non-comparable.** Three things. (1) `qmd mcp stop` could not
stop the daemon: a single `qmd status` deletes `~/.cache/qmd/mcp.pid` while the daemon
keeps running, so the stop answers `Not running (no PID file).` and the next run measures
a warm daemon. The nine cold runs therefore stopped the process on port 8765 directly, and
each was checked for the warming note. (2) Eleven further qmd MCP daemons from earlier
sessions were resident on other ports throughout; one cold `fast` run died with a CUDA
error (`ggml-cuda.cu:106`) and exit 1 and was repeated. (3) Case order: `version` ran
first and was already warm from the three warming calls, `status` last.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| loomux version (start floor) | 45.0 ms | 30.2 ms | 27.7 ms | 44.0 ms | [0] |
| loomux brain search latenz --profile keyword (warm daemon) | 79.0 ms | 67.9 ms | 63.9 ms | 106.6 ms | [0] |
| loomux brain search latenz --profile fast (warm daemon) | 284.8 ms | 260.7 ms | 236.0 ms | 373.7 ms | [0] |
| loomux brain search latenz --profile full (warm daemon) | 805.4 ms | 834.9 ms | 752.8 ms | 940.0 ms | [0] |
| loomux brain status | 2458.3 ms | 2396.7 ms | 2300.9 ms | 2979.8 ms | [0] |

| cold daemon (stopped before each run) | run 1 | run 2 | run 3 |
|---|---:|---:|---:|
| brain search latenz --profile keyword | 1.15 s | 0.90 s | 0.89 s |
| brain search latenz --profile fast | 11.62 s (repeat) | 4.34 s | 7.71 s |
| brain search latenz --profile full | 4.36 s | 4.41 s | 11.92 s |

| benchmark (50 runs) | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| VisibleAreasOfTheRealRegistry | 1,236,552 | 236,531 | 2,213 |
| RegistersOfTheRealRegistry (11 registers) | 813,614 | 610,488 | 1,121 |
| ExecuteSearchWithoutTheEngine | 1,971,952 | 868,143 | 3,574 |

### Reading

1. **The target does not hold, and qmd holds it open.** `--profile fast` warm is
   **260.7 ms** against the target of 150 ms — 110.7 ms over, with a warm range of
   236.0–373.7 ms that never enters it. The post that breaks it is not Go: the start
   floor (`loomux version`) is 30.2 ms, everything loomux does besides process start and
   qmd is 1.97 ms (`ExecuteSearchWithoutTheEngine`), and the same call on the keyword
   path costs 67.9 ms. qmd's share of the answer is therefore about 228 ms — 260.7 less
   the start floor and the 1.97 ms above — against the 63–83 ms the spec budgeted for the
   whole qmd call; about 193 ms of that share is what the vector query adds over the
   keyword path. No rebuild follows in this task.
2. **The registers are 0.3 % of a fast answer.** `RegistersOfTheRealRegistry` reads the
   `_identities.tsv` of all **11** registered areas in **0.81 ms**; against the fast warm
   median of 260.7 ms that is 0.31 %. `VisibleAreasOfTheRealRegistry` — the registry plus
   one manifest per area — is 1.24 ms. The two together (2.05 ms) already account, within
   run-to-run noise, for the whole of `ExecuteSearchWithoutTheEngine` (1.97 ms): reading
   the registry, the manifests and the registers *is* what loomux spends besides process
   start and qmd.
3. **Cold daemon: the model load dominates and it scatters.** Against the spike's
   23 ms / 5.4 s / 11.9 s for a freshly started daemon, the three runs per profile are
   0.89–1.15 s (keyword), 4.34–11.62 s (fast) and 4.36–11.92 s (full). The spread inside
   one profile is wider than the gap between `fast` and `full`, so these nine values order
   nothing: they time a daemon start plus a model load, and that load varied by a factor
   of three on a machine carrying eleven other resident daemons. Keyword is the one clear
   reading: it needs no embedding model, and it still costs ~1 s because the daemon has to
   come up at all. Every value stays under its planned ceiling (6.3 s / 22.4 s / 41.9 s).
   The warm `full` median above is not a new query: the bench repeats `latenz` 21 times,
   which the spike measured at ~250 ms and this pass at 834.9 ms, where a new query on a
   warm daemon took 4.2–7.9 s.
4. **`brain status` is 2.4 s, and none of it is Go.** Warm median 2396.7 ms against a
   start floor of 30.2 ms: the command asks the qmd CLI once per indexed area and once for
   the backlog, so it is bound by qmd process starts.
5. **Moving the brain packages in cost no start time.** Only one `init` line reaches 1 ms,
   before as after: `github.com/BurntSushi/toml/internal` with 21/20/20 ms clock before and
   18/18/22 ms clock after (71,264 bytes, and 71,280 in the last run before; 1,673
   allocs throughout). **No line from
   `github.com/xidus90/loomux/...` reaches 1 ms.** New after the move, none of them above
   0 ms clock except one run of `internal/brain/search` at 0.50 ms:
   `internal/brain/catalog` and `internal/brain/search` (the moved `regexp.MustCompile`
   package variables), `internal/dev/mutants`, and the five `golang.org/x/text` packages
   `unicode/norm`, `internal/language`, `internal/language/compact`, `language` and
   `cases`, which the pre-Task-3 binary does not link at all. Step 10 therefore did not
   trigger; the seven source files holding a package-level `regexp.MustCompile` are exactly
   the ones the plan names.

## 2026-09-16 19:32 — sameRepository Without git rev-parse

Repository `loomux`, branch `claude/cranky-kilby-f5a456`, commit `ee1aadf`;
measured worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1` at `9ca6364`.

**Goal.** Test the attribution the entry of 2026-09-15 15:39 left open: that the
warm gap of about 34 ms between a write in a linked worktree and one in the main
checkout is the two `git rev-parse --git-common-dir` calls `sameRepository` made.
This change reads git's pointer files instead.

**Method.** `loomux dev bench-hooks testdata/bench/barrier-worktrees.json -n 20`:
one cold run per case, then 20 warm ones. `LOOMUX_STATE_DIR` points at a copy of the
registry that registers only the main checkout (`workspace = true`). Binaries:
`before.exe` built from `3855de4` (code identical to `e4e0dc2`), `after.exe` built
from the commit above, both with Go `go1.27.0 windows/amd64`. Both binaries allow
the worktree write; the case names are those of 2026-09-15. The table shows the
second of two runs. The first (19:31) was started together with the build of
`after.exe` and may have overlapped it, so it is not shown; its warm medians lie
within 3.3 ms of these (worktree 73.8 before / 31.3 after, main checkout 30.3
before / 32.0 after). Because both binaries had already started once, all four
cold values here are cached starts and comparable across rows.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| before: Write in linked worktree, no registry entry | 79.4 ms | 72.8 ms | 70.6 ms | 94.4 ms | [0] |
| after: Write in linked worktree, no registry entry | 37.9 ms | 34.6 ms | 31.4 ms | 41.6 ms | [0] |
| before: Edit on README.md in main checkout | 34.2 ms | 31.4 ms | 28.2 ms | 34.4 ms | [0] |
| after: Edit on README.md in main checkout | 32.2 ms | 30.5 ms | 28.6 ms | 35.9 ms | [0] |

### Reading

1. **Worktree: the write drops by more than half.** After against before is
   34.6 ms against 72.8 ms warm (38.2 ms lower) and 37.9 ms against 79.4 ms cold
   (41.5 ms lower). The warm ranges do not overlap (31.4–41.6 against 70.6–94.4).
   The gap to the main checkout was 41.4 ms before (72.8 against 31.4) and is
   4.1 ms after (34.6 against 30.5), with overlapping warm ranges (31.4–41.6
   against 28.6–35.9). Those 4.1 ms are within run-to-run variation: in the first
   run the worktree write after (31.3 ms) was below the main-checkout write after
   (32.0 ms).
2. **Main checkout: noise.** After against before is 30.5 ms against 31.4 ms warm
   (0.9 ms lower) and 32.2 ms against 34.2 ms cold (2.0 ms lower), with warm ranges
   that overlap (28.6–35.9 against 28.2–34.4). `sameRepository` returns there
   before it reads any of git's files, because the checkout's path equals the
   registered one.
3. **The attribution: confirmed.** The worktree write falls by 38.2 ms warm, about
   the gap it had to the main checkout (41.4 ms in this run, about 34–35 ms on
   2026-09-15), and what remains of the gap is within noise. The gap was the two
   `git rev-parse` calls. Against the target of 72 ms: the worktree write after is
   37.4 ms under it. `before.exe` in this run is 0.8 ms over it (72.8 ms) and
   7.5 ms above the 65.3 ms of the 2026-09-15 `after.exe`, built from `d8bfad2`,
   which allowed the worktree write as well. That row ran the same verdict and the
   same two `git rev-parse` calls: between `d8bfad2` and `e4e0dc2` the only code
   commit is `c2e172d`, which adds an `Lstat` and one path comparison to
   `linkedCommon`. The 7.5 ms is unexplained. The main-checkout rows show no
   general slowdown: they are slightly faster than on 2026-09-15 (31.4 against
   31.9 before, 30.5 against 31.0 after). Another session in the measured worktree
   would fit a slowdown that hits only the worktree, but that remains an open
   guess. The attribution rests on the gap within this run, 41.4 ms before and
   4.1 ms after.

## 2026-09-16 22:13 — Registry and Declaration Checks in One Place

Repository `loomux`, worktree
`C:/Users/micro/Documents/#GIT/loomux/.claude/worktrees/recursing-bartik-b2d7a1`,
branch `claude/recursing-bartik-b2d7a1`, commit `222465b` (Task 5 of the plan
`2026-09-16-loomux-registry-manifest-pruefungen`).

**Goal.** Show what the strict readers of the registry and the area declarations
cost at the write barrier and at `brain catalog`.

**Method.** `loomux dev bench-hooks testdata/bench/registry-checks.json -n 20`: one
cold run per case, then 20 warm ones, against the real state directory
(`%LOCALAPPDATA%\loomux` with 11 areas, no `LOOMUX_STATE_DIR`); the barrier case
edits `C:/Users/micro/Documents/#GIT/loomux/README.md` with `--root` on the main
checkout. Binaries: `loomux-before.exe` built from `219ccb1` (the plan commit before
Task 2), `bin/loomux.exe` built by the pre-commit gate at `222465b`. Go benchmarks,
five runs each, median of the five:
`go test ./internal/brain/guard/ -run '^$' -bench DecideAgainstTheRealRegistry -benchtime 50x -benchmem -count 5`
with `LOOMUX_BENCH_REGISTRY` on a copy of `%LOCALAPPDATA%\loomux` and
`LOOMUX_BENCH_TARGET=C:/Users/micro/Documents/#GIT/loomux/README.md`, and
`go test ./internal/brain/search/ -run '^$' -bench VisibleAreasOfTheRealRegistry -benchtime 50x -benchmem -count 5`
with `LOOMUX_BENCH_REGISTRY` on the same copy and `LOOMUX_BENCH_LEGACY` on
`%LOCALAPPDATA%\brain`. For the Go benchmarks, "before" was run in Task 1 on
`219ccb1` and "after" at `222465b`, both on the same state copy; the bench-hooks
before/after rows ran in one run against the real state directory. Machine: AMD Ryzen 7 9800X3D, Go
`go1.27.0 windows/amd64`, GOMAXPROCS 16.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| before: loomux hook pre-tool-use (Edit on README.md, real registry) | 73.7 ms | 35.5 ms | 33.1 ms | 47.7 ms | [0] |
| after: loomux hook pre-tool-use (Edit on README.md, real registry) | 36.0 ms | 35.7 ms | 32.6 ms | 44.4 ms | [0] |
| before: loomux brain catalog (real registry) | 39.5 ms | 35.6 ms | 33.0 ms | 44.5 ms | [0] |
| after: loomux brain catalog (real registry) | 35.0 ms | 37.0 ms | 31.9 ms | 42.5 ms | [0] |

| benchmark | ns/op before | ns/op after | B/op before | B/op after | allocs/op before | allocs/op after |
|---|---:|---:|---:|---:|---:|---:|
| DecideAgainstTheRealRegistry | 1,596,244 | 1,384,310 | 137,694 | 140,757 | 1,277 | 1,327 |
| VisibleAreasOfTheRealRegistry | 1,217,696 | 1,432,590 | 237,694 | 226,964 | 2,213 | 2,079 |

### Reading

1. **Barrier, end to end.** After against before is 35.7 ms against 35.5 ms warm
   (0.2 ms higher), with overlapping warm ranges (32.6–44.4 against 33.1–47.7).
   Cold, after is 36.0 ms against 73.7 ms (37.7 ms lower).
2. **`brain catalog`, end to end.** After against before is 37.0 ms against 35.6 ms
   warm (1.4 ms higher), with overlapping warm ranges (31.9–42.5 against
   33.0–44.5). Cold, after is 35.0 ms against 39.5 ms (4.5 ms lower).
3. **`DecideAgainstTheRealRegistry`.** 1,384,310 against 1,596,244 ns/op (211,934 ns
   or 13.3 % lower); the five runs do not overlap (1,361,546–1,452,418 against
   1,502,812–1,720,166). 140,757 against 137,694 B/op (2.2 % more), 1,327 against
   1,277 allocs/op (50 or 3.9 % more).
4. **`VisibleAreasOfTheRealRegistry`.** 1,432,590 against 1,217,696 ns/op
   (214,894 ns or 17.6 % higher); the five runs do not overlap (1,395,594–1,503,752
   against 1,146,422–1,388,878). 226,964 against 237,694 B/op (4.5 % less), 2,079
   against 2,213 allocs/op (134 or 6.1 % fewer).
5. **Against the plan's limits** (barrier warm median at most 3 ms more, Go
   benchmarks at most 20 % more): barrier +0.2 ms, `DecideAgainstTheRealRegistry`
   −13.3 %, `VisibleAreasOfTheRealRegistry` +17.6 %.

### Cause

Investigated on 2026-09-16 at 22:20 after the branch review. A worktree of
`219ccb1` and the branch at `e01977b` (no code change since `222465b`) ran the
`VisibleAreasOfTheRealRegistry` command above in alternation, three rounds of five
runs per side, with the same environment. Before: round medians 1,147,168,
1,163,598 and 1,421,438 ns/op; median of all 15 runs 1,178,826 (1,096,538–1,614,158).
After: 1,127,922, 1,123,730 and 1,120,828 ns/op; median of all 15 runs 1,123,730
(1,076,292–1,323,442). The +17.6 % does not reproduce: the third before round
jumps the same way on unchanged code, so the machine's state during the run
produced it, not the change. CPU profiles at 3,000 iterations, two runs per side
merged, before against after: `ReadAreaManifestUntilStage4` 6.41 s against 6.42 s
cumulative, all syscalls 5.13 s against 4.99 s, TOML decoding 1.70 s against 1.71 s,
`ReadRegistry` 0.73 s against 0.67 s. The stat-before-read order moves time from
failed opens of the missing names (`os.Open` 2.50 s → 1.07 s) to stats (`os.Stat`
1.20 s → 2.55 s) and leaves the sum unchanged. No code was changed.

## 2026-09-17 14:16 — The Start Path Without the Local Time Zone

Repository `loomux`, worktree `.claude/worktrees/recursing-bartik-b2d7a1`, branch
`perf-lazy-local-zone` on `6cafdd8`. The change: `go.mod` replaces
`github.com/BurntSushi/toml` v1.6.0 with a pruned copy under `third_party/toml`
whose `internal/tz.go` builds the three local zones on first use instead of in a
package variable.

**Goal.** Take the 18.7 ms the entry of 2026-09-15 12:00 attributed to resolving
the local time zone off the hook path, and check that nothing else on the path
resolves it at run time.

**Method.** `loomux dev bench-hooks testdata/bench/lazy-local-zone.json -n 20`, one
pass, one cold run per case and 20 warm. `before.exe` is built from `master` at
`6cafdd8`, `after.exe` from the branch, both with Go 1.27, copied to
`%TEMP%\loomux-zone-bench`. The hook runs in the main checkout with its pilot
`.loomux/config.toml` and the real registry, payload
`testdata/bench/edit-readme-main.json`. The floors are an empty `main` and a
`main` whose only statement is `time.Now().Zone()`. Neither `.loomux/config.toml`
nor `registry.toml` holds a date or time value, so no parse on this path asks for
the lazy zones.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| before: loomux hook pre-tool-use (Edit on README.md) | 51.9 ms | 26.5 ms | 25.5 ms | 32.5 ms | [0] |
| after: loomux hook pre-tool-use (Edit on README.md) | 9.5 ms | 7.5 ms | 7.0 ms | 8.0 ms | [0] |
| before: loomux version (start floor) | 25.5 ms | 24.1 ms | 23.1 ms | 25.0 ms | [0] |
| after: loomux version (start floor) | 7.0 ms | 5.5 ms | 5.5 ms | 6.0 ms | [0] |
| empty Go main (spawn floor) | 48.0 ms | 4.5 ms | 4.5 ms | 5.7 ms | [0] |
| Go main resolving only the local zone | 75.6 ms | 23.9 ms | 22.4 ms | 32.1 ms | [0] |

### Reading

1. **The hook, end to end.** After against before is 7.5 ms against 26.5 ms warm
   (19.0 ms less), with disjoint warm ranges (7.0–8.0 against 25.5–32.5). Cold it
   is 9.5 ms against 51.9 ms.
2. **The start floor moved by the same amount.** `version` is 5.5 ms against
   24.1 ms warm (18.6 ms less), 1.0 ms above the empty `main`. The zone-only
   `main` costs 19.4 ms above the empty one in this pass, against 18.7 ms on
   2026-09-15; the saving is that cost and no more.
3. **Nothing on the path resolves the zone later.** The hook is 2.0 ms above its
   floor after the change, where before it was 2.4 ms; a run-time resolution
   would show up as the 19 ms again.
4. **The init trace, allocations rather than clock.**
   `GODEBUG=inittrace=1 loomux version`: `github.com/BurntSushi/toml/internal`
   drops from 20 ms clock, 1,673 allocs to 0 ms, 2 allocs. The test
   `TestStartDoesNoWorkInPackageInit` in `cmd/loomux` now builds the binary and
   fails if any package init makes more than 500 allocations, so a dependency
   that brings the zone back is caught by the gate, not by the next measurement.

## 2026-09-17 20:30 — The Start Floor With the MCP SDK Linked

Repository `loomux`, worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b-2`,
branch `sdd-1b-2`, before `48e5d38`. Stage 1b-2, Task 1: `go-sdk` v1.8.0 comes
in, `x/sys` rises to v0.48.0 and `x/text` to v0.42.0, and the `go` directive to
1.26.0.

**Goal.** Fix the start floor the MCP service will be measured against, and say
what of it is the dependency bump and what is the SDK itself. The SDK is the
first dependency of this size the binary carries, and every hook pays its start.

**Method.** PowerShell, one `Measure-Command` per run, twelve runs per binary:
the first is the cold value, the median of the other eleven the warm one.
`loomux --version` in every case, Go `go1.27.0 windows/amd64`. Three binaries:
`before` is `bin/loomux.exe` as `48e5d38` built it; `bump only` is a scratch
build after the dependency bump, with nothing importing the SDK, so the linker
drops it; `after` is `bin/loomux.exe` as the gate rebuilt it, with `cmd/loomux`
importing `internal/mcptools` and the SDK therefore linked in. The cold values
are first runs in an already warm shell and are not comparable across rows; they
are shown, not read.

| case | size | cold (1st run) | warm median | warm min | warm max |
|---|---:|---:|---:|---:|---:|
| before: `48e5d38` | 13,736,448 B | 18.9 ms | 7.4 ms | 7.3 ms | 7.8 ms |
| bump only: SDK dropped by the linker | 13,755,904 B | 46.8 ms | 7.6 ms | 7.3 ms | 8.3 ms |
| after: SDK linked | 16,004,608 B | 45.7 ms | 7.6 ms | 7.4 ms | 8.0 ms |

Package inits over 100 allocations, from `GODEBUG=inittrace=1 loomux version`
on `after`:

| package | allocations |
|---|---:|
| `encoding/gob` | 369 |
| `github.com/google/jsonschema-go/jsonschema` | 298 |
| `gopkg.in/yaml.v3` | 274 |
| `github.com/modelcontextprotocol/go-sdk/mcp` | 183 |
| `github.com/xidus90/loomux/internal/brain/wiki` | 131 |
| `github.com/xidus90/loomux/internal/brain/search` | 103 |

### Reading

1. **The start time does not move.** After against before is 7.6 ms against
   7.4 ms warm (0.2 ms more), with warm ranges that overlap almost entirely
   (7.4–8.0 against 7.3–7.8). The bump-only binary sits at the same 7.6 ms, so
   even that 0.2 ms is not the SDK. The SDK's own init is 183 allocations and
   costs no measurable clock.
2. **The size does move, and the SDK is all of it.** After against before is
   16,004,608 B against 13,736,448 B, 2,268,160 B more, 16.5 %. Of that, 19,456 B
   belong to the dependency bump and 2,248,704 B to linking the SDK: `mcp`,
   `jsonschema-go`, `segmentio/encoding`, `uritemplate`, `x/oauth2` and
   `encoding/gob`.
3. **The allocation ceiling holds, with room.** The largest init is
   `encoding/gob` at 369, under the 500 that `TestStartDoesNoWorkInPackageInit`
   allows and close to the 367 the plan recorded. `gob` is new on this path: it
   comes in with the SDK, as does `jsonschema-go` at 298. Both are below the
   line, but the line now has two packages under it that no loomux code asked
   for directly, and a further SDK dependency could push one over.
## 2026-09-18 11:10 — The Graph Commands, Cold and Warm, and the G1 Debt

Repository `loomux`, worktree `C:/Users/micro/Documents/#GIT/loomux-code-g2`,
branch `code-g2`, commit `75c8136`. This task adds two benchmark files and
this entry; it changes no production code.

**Goal.** Four numbers task 10 owed. First, the cold figure for the dangling-mass
optimisation in `internal/code/pagerank` that the entry of 2026-09-16 left
open (only the warm ~9 ms pooled against ~4.5 s per-node existed). Second and
third, `loomux graph build` on this repository, cold and warm, against the
~27–32 ms parsing floor of the spec's §11 (254 files; see the correction note
under "Reading," point 2, below — the figure I first had for this floor was
measured over a doubled tree). Fourth, `internal/code/freshness.Probe`
alone, against the reference implementation's ~3 ms for 280 files — the number
§7.1 of the design spec makes its file-set decision conditional on.

**Method.**

*Dangling mass, cold.* `internal/code/pagerank/dangling_bench_test.go` did not
exist before this task; it now holds `BenchmarkDanglingPooled` (calls the
production `Rank`) and `BenchmarkDanglingPerNode` (a copy of `Rank`'s loop with
the one line the optimisation replaced: the dangling mass is handed back once
per dangling node instead of pooled and applied in one pass). Both use the
same fixture as `TestRankBroadSeedsOnMostlyDanglingGraph`: 20,000 nodes, a
100-node chain, the rest dangling, every node seeded. Cold means one process
per measurement, not several rounds inside one:

```
$ go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDanglingPooled -benchtime 1x -count 1
BenchmarkDanglingPooled-16    	       1	   5875500 ns/op

$ go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDanglingPerNode -benchtime 1x -count 1
BenchmarkDanglingPerNode-16    	       1	3299862800 ns/op
```

*`graph build`, cold and warm.* A binary built from this commit
(`go build -o /tmp/loomux-bench.exe ./cmd/loomux`), run against this
repository. Cold means the graph and the freshness record removed first, so
the reported time includes writing both to a tree that had neither:

```
$ rm -rf .loomux/state/graph && time /tmp/loomux-bench.exe graph build --root .
254 files, 2801 nodes, 8980 edges (2547 contains, 5081 calls, 1352 imports)
1123 unresolved import targets, 3 files without a symbol, 196ms

real	0m0.219s
```

Warm, three repeats immediately after, graph and freshness record left in
place:

```
$ time /tmp/loomux-bench.exe graph build --root .
254 files, 2801 nodes, 8980 edges (2547 contains, 5081 calls, 1352 imports)
1123 unresolved import targets, 3 files without a symbol, 191ms   real 0m0.213s
254 files, 2801 nodes, 8980 edges (2547 contains, 5081 calls, 1352 imports)
1123 unresolved import targets, 3 files without a symbol, 190ms   real 0m0.214s
254 files, 2801 nodes, 8980 edges (2547 contains, 5081 calls, 1352 imports)
1123 unresolved import targets, 3 files without a symbol, 212ms   real 0m0.236s
```

*`graph check`, warm.* Same binary, same tree, graph already written by the
run above:

```
$ time /tmp/loomux-bench.exe graph check --root .
loomux graph check: OK
real	0m0.213s

$ time /tmp/loomux-bench.exe graph check --root .
loomux graph check: OK
real	0m0.223s
```

*The probe alone.* `internal/code/freshness/probe_bench_test.go` did not exist
before this task. `BenchmarkProbe` writes a freshness record for this
repository's real file set — 254 Go files, read and hashed once, outside the
timer — then times `Probe` alone, repeatedly, against that record:

```
$ go test ./internal/code/freshness/ -bench BenchmarkProbe -benchtime 10x -v
BenchmarkProbe
    probe_bench_test.go:61: probing 254 files
    probe_bench_test.go:61: probing 254 files
BenchmarkProbe-16    	      10	  57477360 ns/op
PASS

$ go test ./internal/code/freshness/ -bench BenchmarkProbe -benchtime 10x
BenchmarkProbe-16    	      10	  59840350 ns/op
```

Two separate runs, shown as two separate blocks rather than one assembled
line: 57.48 ms/op with `-v` (which is also where the "probing 254 files" log
line comes from), 59.84 ms/op on a plain repeat immediately after. Both are
used below; the "reported" figure this entry's table and Reading carry is
59.84 ms/op, the more recent of the two identical-method runs.

Machine: AMD Ryzen 7 9800X3D, Go `go1.27.0 windows/amd64`, GOMAXPROCS 16 — the
same machine the 2026-09-16 pooled/per-node figures and the Graft reference
comparisons were made on.

| case | cold | warm | reference / floor |
|---|---:|---:|---|
| pagerank dangling mass, pooled (20k nodes, 19,900 dangling) | 5.88 ms | — | ~9 ms pooled, 2026-09-16 |
| pagerank dangling mass, per node (same graph) | 3.30 s | — | ~4.5 s per node, 2026-09-16 |
| `graph build --root .` (this repository, pre-`SkipDir`, superseded) | 219 ms wall / 196 ms self-reported | 213–236 ms wall / 190–212 ms self-reported | superseded — see the 2026-09-18 11:49 entry below for the current figure and its own parsing-floor benchmark |
| `graph check --root .` (this repository, pre-`SkipDir`, superseded) | — | 213–223 ms | n/a |
| `freshness.Probe` alone (254 files, this repository) | — | 59.8 ms/op (10 reps) | ~3 ms for 280 files (Graft) |

### Reading

1. **The G1 debt is paid, and the number confirms the design comment word for
   word.** Cold, pooled is 5.88 ms against per-node's 3.30 s — a factor of
   about 561, on a single cold process each, not an average over many warm
   ones. Both are close to the 2026-09-16 warm figures (~9 ms, ~4.5 s) on the
   same machine, which is what a cold-vs-warm gap this small should look like
   for a computation with no I/O and no cache to warm: the cost is arithmetic,
   not process state.
2. **`graph build` does not distinguish cold from warm, and the code explains
   why before the number does.** Cold (219 ms) and warm (213–236 ms) overlap
   completely. `buildGraph`'s own comment says it: the command "reads and
   hashes every file, every time — never the probe's stat fast path", because
   a stat may decide whether a *query* rebuilds, never what a rebuild looks
   at. There is no cache for `build` itself to warm.
   **Correction (2026-09-18, later the same day).** This entry originally read
   "against the 44–46 ms parsing floor of §11, 190–219 ms is 4.3–4.8x", and a
   same-day fix then wrote "`build`'s 190–219 ms self-reported" — also wrong,
   since 219 ms in this entry's own table above is the *cold wall-clock*
   figure, not self-reported (self-reported cold is 196 ms). **Both the floor
   and the `build` figures in this entry are superseded, and neither belongs
   here any further**: `SkipDir` shipped after this entry was written and made
   `build` itself faster (see the 2026-09-18 11:49 entry below, which carries
   its own command and raw output, measured once on the final tree, after
   every code change this task made). This paragraph is left in place, struck
   through in spirit rather than deleted, because the journal is chronological
   and a wrong number that vanishes teaches the next reader nothing about why
   it was wrong.
3. **`graph check`, warm, costs about what `build` costs, which is the design,
   not a defect.** 213–223 ms against `build`'s 190–212 ms warm: `check`
   re-extracts the whole tree to compare body hashes, so its cost is a second
   `buildGraph` plus a diff, minus the write. The two numbers being close is
   the CLI reference's claim made visible — `check` does not read the
   freshness record, so a `touch` is not a finding, but it is also not a
   cheap one.
4. **The probe is 20x the reference, and §7.1's premise does not hold as
   confidently as it reads.** 59.8 ms/op against Graft's ~3 ms for 280 files
   is far above what a hook budget of a few tens of milliseconds can absorb
   once alongside everything else on that path. It is not extraction cost —
   `Probe` never opens a file when size and mtime match, which they do here by
   construction. A separate timing of `sourceset.Stat` alone against this
   repository's root reproduced the same ~57–70 ms, so the cost is the
   directory walk `internal/code/sourceset` performs, not the comparison
   after it. This repository's tree has 1,910 directories in total, of which
   1,826 sit below one of the 3 directories named `testdata` (mostly recorded
   case fixtures with no `.go` files); Graft's 280-file reference tree is not
   this shape. §7.1's argument — that a
   `git ls-files` subprocess per probe call would cost more than the probe —
   is still true against a *subprocess's* floor (tens of milliseconds), but
   the probe's own floor on a tree with this many non-source directories is
   not the ~3 ms the section assumes; it is closer to the subprocess cost it
   was avoiding. This entry does not decide the design question; it hands the
   next one a number that says the assumption needs rechecking on a
   directory-heavy tree, not only a file-heavy one.

## 2026-09-18 11:35 — The Probe's Missing Skip Entry

Repository `loomux`, worktree `C:/Users/micro/Documents/#GIT/loomux-code-g2`,
branch `code-g2`. This entry closes the question the previous one left open:
whether §7.1's design decision (walk the filesystem rather than shell out to
`git ls-files`) should be reversed. It should not — the cause was a missing
name on the skip list, not the walk itself.

**The check.** `internal/code/sourceset.skipDirs` did not include `testdata`,
so `sourceset.Stat`'s `filepath.WalkDir` descended into every directory under
`testdata/` looking for `.go` files it never found there — none of this
repository's fixtures under `testdata/` are `.go` (they are `.go.txt` or
recorded case corpora). `go/build` itself already ignores `testdata/`; Graft's
own skip list lacks the name only because Graft is not Go-specific. Adding
`"testdata"` to `skipDirs`, in the list's existing order, is the whole change
(`internal/code/sourceset/sourceset.go`); a new test,
`TestListSkipsTestdata`, asserts a `.go` file under `testdata/` is never
listed.

**Before and after, same command, same tree:**

```
$ go test ./internal/code/freshness/ -bench BenchmarkProbe -benchtime 10x
```

| when | ns/op | files probed | directories total | directories below a `testdata` root |
|---|---:|---:|---:|---:|
| before (as committed in the first round) | 59,840,350 | 254 | 1,910 | 1,826 |
| after (`testdata` added to `skipDirs`) | 4,293,460 and 2,719,400 (two runs) | 254 | 1,910 (unchanged — the entries themselves are not removed, only not walked) | 1,826 (unchanged) |

"Directories below a `testdata` root" is the predicate that matches what the
skip buys: 3 directories are named `testdata` in this tree, and
`filepath.WalkDir` still visits each of those 3 once to decide to skip it —
only the 1,826 directories beneath them stop being walked. (The count of
"3 `testdata` roots plus everything below them" is 1,829; that is a different,
also-true number, but not the one the fix removes from the walk.)

Raw output of the two "after" runs:

```
BenchmarkProbe-16    	      10	   4293460 ns/op
BenchmarkProbe-16    	      10	   2719400 ns/op
```

**`graph build` and `graph check` afterward, to confirm nothing indexable
moved:**

```
$ rm -rf .loomux/state/graph && time /tmp/loomux-bench2.exe graph build --root .
254 files, 2802 nodes, 8983 edges (2548 contains, 5083 calls, 1352 imports)
1123 unresolved import targets, 3 files without a symbol, 157ms
real	0m0.217s

$ time /tmp/loomux-bench2.exe graph build --root .
254 files, 2802 nodes, 8983 edges (2548 contains, 5083 calls, 1352 imports)
1123 unresolved import targets, 3 files without a symbol, 152ms
real	0m0.176s

$ time /tmp/loomux-bench2.exe graph check --root .
loomux graph check: OK
real	0m0.177s
```

File count: 254, unchanged from the 2026-09-18 11:10 entry. Node count moved
by +1 (2801 → 2802) and edges by +3 (8980 → 8983) — both explained by this
round's own new test code inside `internal/code` (the added
`TestListSkipsTestdata` function and its call sites), not by the `skipDirs`
change: a `.go` file under `testdata/` was never a member of the file set
either way, so skipping the directory could not remove a node that was never
there. Build and check times (152–217 ms) are within the noise of the
11:10 entry's 190–236 ms.

### Reading

1. **The 20x gap closes to about at-or-under the reference, on one changed
   line.** 59.8 ms/op before, 2.7–4.3 ms/op after, against Graft's ~3 ms for
   280 files: the second run lands under the reference, the first just over
   it — both are the right order of magnitude, where the previous entry's
   number was 20x off. The file count did not move (254 either way), which is
   the check that this was a walk-cost fix and not a silent narrowing of what
   gets indexed.
2. **§7.1's conditional resolves to a third answer, not either of the two it
   named.** The section asked whether the probe's cost would force a switch
   to `git ls-files`. It did not: the walk itself was never the problem, and
   neither was the absence of Git. The problem was one missing name on a list
   that already existed for exactly this purpose. The spec's §7.1, §13 and
   §3.7 deviation table are updated to record this rather than leaving the
   conditional looking like it was never opened.
3. **`graph build` and `graph check` did not move for the reason that
   mattered.** Their file, node and edge counts are unchanged apart from this
   round's own test additions — confirming that `testdata/` held nothing the
   graph needs, and that the fix cost the probe its walk time without costing
   the graph anything.

The `graph build`/`graph check` figures above (152–157 ms self-reported,
176–217 ms wall) are this round's own, honestly measured and left as-is; a
later change (`goModPaths` sharing `sourceset.SkipDir`, see the commit after
this one) made `build` faster still. The current figure, measured once after
every code change this task made, is in the 2026-09-18 11:49 entry below.

## 2026-09-18 11:49 — Final Numbers, Measured Once After the Last Code Change

Repository `loomux`, worktree `C:/Users/micro/Documents/#GIT/loomux-code-g2`,
branch `code-g2`, on top of commit `cc8f7f0` (`goModPaths` now shares
`sourceset.SkipDir`) — no further code change follows this entry. Every
number below was measured in one sitting, after all of task 10's code changes
had landed, specifically to close two problems the previous two entries had:
the `build` figures in the 2026-09-18 11:10 entry were overtaken by the very
commit that entry described (`SkipDir` stopped `goModPaths` from walking
`testdata/`, so `build` got faster the same day it was measured slower), and
the "~27–32 ms parsing floor" both benchmarks files asserted had no command
or raw output of its own — it pointed at the design spec, which pointed back
here. Every figure below carries its own command and its own raw output; none
of them cites the spec, and none of them reuses a number this task did not
measure itself on this machine.

**The parsing floor, measured here for the first time as a real benchmark**
(`internal/code/extract/golang/parsefloor_bench_test.go`, new in this
commit): `go/parser` alone over this repository's real file set, with and
without `parser.SkipObjectResolution`, mirroring the choice `extract.go`
documents (`ast.Object` is deprecated, so the extractor keeps its own scope
stack instead of asking `go/parser` to resolve identifiers).

```
$ go test ./internal/code/extract/golang/ -bench BenchmarkParse -benchtime 10x -v
goos: windows
goarch: amd64
pkg: github.com/xidus90/loomux/internal/code/extract/golang
cpu: AMD Ryzen 7 9800X3D 8-Core Processor
BenchmarkParseSkipObjectResolution
    parsefloor_bench_test.go:57: parsing 255 files
    parsefloor_bench_test.go:57: parsing 255 files
BenchmarkParseSkipObjectResolution-16    	      10	  18755780 ns/op
BenchmarkParseWithObjectResolution
    parsefloor_bench_test.go:74: parsing 255 files
    parsefloor_bench_test.go:74: parsing 255 files
BenchmarkParseWithObjectResolution-16    	      10	  26208830 ns/op
PASS
ok  	github.com/xidus90/loomux/internal/code/extract/golang	0.735s
```

**These figures — 18.8 ms / 26.2 ms warm over 255 files — differ from the
~27–32 ms / ~38–45 ms the coordinator measured with a separate scratch
harness on 2026-09-17.** Mine are lower on both sides and are the ones this
document now carries, because they come with the command above, are
reproducible with `go test`, and were taken on the same tree as every other
number in this entry. I did not investigate the gap (different point in time,
different process, a possibly different set of files — this benchmark counts
255, the coordinator's harness counted 254) beyond noting it plainly rather
than silently picking whichever number was more convenient.

**`graph build` and `graph check`, cold and warm, on a binary built from this
commit:**

```
$ go build -o /tmp/loomux-final.exe ./cmd/loomux
$ rm -rf .loomux/state/graph && time /tmp/loomux-final.exe graph build --root .
255 files, 2808 nodes, 9003 edges (2553 contains, 5092 calls, 1358 imports)
1128 unresolved import targets, 3 files without a symbol, 94ms
real	0m0.151s

$ time /tmp/loomux-final.exe graph build --root .
255 files, 2808 nodes, 9003 edges (2553 contains, 5092 calls, 1358 imports)
1128 unresolved import targets, 3 files without a symbol, 106ms
real	0m0.131s

$ time /tmp/loomux-final.exe graph build --root .
255 files, 2808 nodes, 9003 edges (2553 contains, 5092 calls, 1358 imports)
1128 unresolved import targets, 3 files without a symbol, 98ms
real	0m0.127s

$ time /tmp/loomux-final.exe graph check --root .
loomux graph check: OK
real	0m0.136s
```

**`freshness.Probe` alone:**

```
$ go test ./internal/code/freshness/ -bench BenchmarkProbe -benchtime 10x -v
BenchmarkProbe
    probe_bench_test.go:61: probing 255 files
    probe_bench_test.go:61: probing 255 files
BenchmarkProbe-16    	      10	   4245920 ns/op
PASS
```

**The pagerank dangling-mass pair, cold, one process each, re-confirmed on
this tree** (no code in `internal/code/pagerank` changed since the
2026-09-18 11:10 entry, but this entry's rule is "measured in this sitting or
it does not go in this entry"):

```
$ go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDanglingPooled -benchtime 1x -count 1
BenchmarkDanglingPooled-16    	       1	   6352900 ns/op

$ go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDanglingPerNode -benchtime 1x -count 1
BenchmarkDanglingPerNode-16    	       1	3865115600 ns/op
```

| case | figure | command |
|---|---:|---|
| parsing floor, `SkipObjectResolution` (255 files, warm) | 18.8 ms | `go test ./internal/code/extract/golang/ -bench BenchmarkParseSkipObjectResolution -benchtime 10x` |
| parsing floor, with object resolution (255 files, warm) | 26.2 ms | `go test ./internal/code/extract/golang/ -bench BenchmarkParseWithObjectResolution -benchtime 10x` |
| `graph build --root .`, cold (255 files) | 151 ms wall / 94 ms self-reported | `rm -rf .loomux/state/graph && time /tmp/loomux-final.exe graph build --root .` |
| `graph build --root .`, warm (255 files, two repeats) | 127–131 ms wall / 98–106 ms self-reported | `time /tmp/loomux-final.exe graph build --root .` |
| `graph check --root .`, warm | 136 ms wall, exit 0 | `time /tmp/loomux-final.exe graph check --root .` |
| `freshness.Probe` alone (255 files, 10 reps) | 4.25 ms/op | `go test ./internal/code/freshness/ -bench BenchmarkProbe -benchtime 10x` |
| pagerank dangling mass, pooled (20k nodes, cold, one process) | 6.35 ms | `go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDanglingPooled -benchtime 1x -count 1` |
| pagerank dangling mass, per node (20k nodes, cold, one process) | 3.87 s | `go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDanglingPerNode -benchtime 1x -count 1` |

### Reading

1. **`build` against its own floor: about 5–7×, on numbers that share a
   command each.** 94–106 ms self-reported (255 files) against an 18.8 ms
   parsing-only floor is 5.0–5.6×; against the 26.2 ms floor with object
   resolution, 3.6–4.0×. Both are in the same range the earlier, doubled-tree
   comparison landed on by accident (4.3–4.8× and, corrected, 5–7×) — the
   ratio held up across a wrong floor, a corrected-but-uncommitted floor, and
   now a floor with its own reproducible benchmark, which is some evidence
   the extraction-and-resolution cost above the parse floor is a stable
   multiple of it rather than an artifact of any one measurement's error.
2. **`build` got faster than every previous entry in this file reported, and
   the reason is a change this task made, not noise.** 94–106 ms self-reported
   here against 190–212 ms in the 11:10 entry and 152–157 ms in the 11:35
   entry, on the same 254–255 files: `goModPaths` no longer walks 1,826
   `testdata/` directories looking for a `go.mod` it also never finds there.
   This entry is the only one of the three where the number and the code that
   produced it are the same commit.
3. **The dangling-mass ratio is stable across re-measurement.** 6.35 ms
   pooled against 3.87 s per node here, against 5.88 ms / 3.30 s in the 11:10
   entry and ~9 ms / ~4.5 s on 2026-09-16 — all cold, single-process, same
   machine, three different sessions. The absolute numbers move by run-to-run
   noise (a factor of ~1.1–1.2×); the ~500–600× ratio between pooled and
   per-node does not.

## 2026-09-18 15:30 — Stage 1b-2 Closed: Size, Hook Path, Handshake and the Two Hops

Repository `loomux`, worktree `C:/Users/micro/Documents/#GIT/loomux-sdd-1b-2`,
branch `sdd-1b-2`, commit `0a4786c` — the four measurements the stage is signed
off against. Machine: AMD Ryzen 7 9800X3D, GOMAXPROCS 16, Go
`go1.27.0 windows/amd64`.

**One correction to the brief this task carried.** It named the pre-build
figures as "13.7 MB / 8.1 ms without the SDK, 16.0 MB / 8.2 ms with it, measured
2026-09-17 with `Measure-Command` over 20 runs". The entry of 2026-09-17 20:30
above says 12 runs (one cold, median of eleven warm) and 7.4 / 7.6 ms. The
method reproduced here is the file's, and the file's numbers are the baseline.

### 1. Binary size and start floor, before and after the SDK

**Method.** PowerShell, one `Measure-Command` per run, twelve runs per binary:
the first is the cold value, the median of the other eleven the warm one;
`loomux --version` in every case. `after` is `bin/loomux.exe` at `0a4786c`.
`before` is `48e5d38` — the commit before the SDK came in, the same one the
2026-09-17 entry measured — rebuilt here from a `git archive` export into a
scratch directory, which is why it is 13,746,688 B against the 13,736,448 B
recorded then: the export is not a git repository, so the build carries no VCS
stamp. Cold values are first runs in an already warm shell and are not
comparable across rows; they are shown, not read.

**This section uses no fixture.** It times `loomux --version` with
`Measure-Command` directly; the JSON fixture belongs to section 2 alone. The two
therefore report two different start floors for the same binary — 8.4 ms here,
6.4 ms there — and that is the instrument, not the binary. Section 4 says which
of the two it subtracts and why it gives a range.

Building the `before` binary, in PowerShell, since nothing in the tree does it:

```powershell
git archive -o "$env:TEMP\loomux-1b-2-src.tar" 48e5d38
New-Item -ItemType Directory -Force "$env:TEMP\loomux-1b-2-src" | Out-Null
tar -xf "$env:TEMP\loomux-1b-2-src.tar" -C "$env:TEMP\loomux-1b-2-src"
Push-Location "$env:TEMP\loomux-1b-2-src"
go build -o "$env:TEMP\loomux-1b-2\before.exe" ./cmd/loomux
Pop-Location
```

| case | size | cold (1st run) | warm median | warm min | warm max |
|---|---:|---:|---:|---:|---:|
| before `48e5d38`: no SDK | 13,746,688 B | 53.2 ms | 8.3 ms | 7.9 ms | 10.5 ms |
| after `0a4786c`: the whole stage | 17,559,552 B | 46.3 ms | 8.4 ms | 8.1 ms | 10.7 ms |

Package inits over 100 allocations, `GODEBUG=inittrace=1 loomux version` on
`after`:

| package | allocations |
|---|---:|
| `encoding/gob` | 369 |
| `github.com/google/jsonschema-go/jsonschema` | 298 |
| `gopkg.in/yaml.v3` | 274 |
| `github.com/modelcontextprotocol/go-sdk/mcp` | 183 |
| `github.com/xidus90/loomux/internal/brain/wiki` | 131 |
| `github.com/xidus90/loomux/internal/brain/search` | 103 |

**Reading.** The start floor did not move across the whole stage: 8.4 ms against
8.3 ms warm, with warm ranges that overlap almost entirely (8.1–10.7 against
7.9–10.5). The size did: 17,559,552 B against 13,746,688 B, 3,812,864 B more,
27.7 %. Of that, 2,268,160 B were the SDK arriving in Task 1; the remaining
~1,545,000 B are the code this stage added and the SDK surface it reaches — the
Streamable HTTP server and client, `x/oauth2`, `uritemplate`. That is more than
the five new packages: `git diff --stat 48e5d38..0a4786c -- cmd/ internal/`
names the case recorder, the case importer and the corpus loader among them. The
allocation ceiling is unmoved: the largest init is still `encoding/gob` at 369,
under the 500 `TestStartDoesNoWorkInPackageInit` allows, and no package of
`internal/serve`, `internal/bridge`, `internal/lock`, `internal/mcptools` or
`internal/brain/answer` appears in the list at all.

### 2. The hook path, unchanged and measured

**Method.** `loomux dev bench-hooks testdata/bench/1b-2-hooks.json -n 20`, one
pass, one cold run per case and 20 warm. Payload an `Edit` on this worktree's
`README.md` (`testdata/bench/edit-readme-1b-2.json`) against the real state
directory — this worktree is a linked worktree of a registered workspace, so the
barrier takes the worktree path. Same two binaries as above, in one pass with
their own floors so the hook can be read against them.

**Before this command runs anywhere else, the fixture has to be re-pathed.**
`1b-2-hooks.json` and `edit-readme-1b-2.json` carry absolute paths of the
worktree they were recorded in — every `dir`, every `stdin` and two `argv`
entries in the first file, three more in the second — exactly as
`1a-hooks.json` does for stage 1a. The `after` binary is `bin/loomux.exe` of
that worktree; the `before` binary is built by the PowerShell above. The fixture
names `%TEMP%\loomux-1b-2\before.exe`, a stable location rather than a session
directory, so only the worktree paths need replacing.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| before: `loomux version` (start floor, no SDK) | 10.2 ms | 7.6 ms | 6.0 ms | 14.1 ms | [0] |
| after: `loomux version` (start floor, SDK linked) | 7.5 ms | 6.4 ms | 5.7 ms | 7.0 ms | [0] |
| before: `loomux hook pre-tool-use` (Edit on README.md) | 12.0 ms | 9.9 ms | 9.0 ms | 12.0 ms | [0] |
| after: `loomux hook pre-tool-use` (Edit on README.md) | 11.5 ms | 9.0 ms | 8.1 ms | 11.0 ms | [0] |

**Reading.** The hook is 9.0 ms warm after the stage against 9.9 ms before it,
and cold 11.5 against 12.0 ms; the warm ranges overlap (8.1–11.0 against
9.0–12.0). Above its own floor the hook costs 2.6 ms after and 2.3 ms before —
the same decision, and the difference is inside the noise both floors show.
That the MCP stack is not on this path is structural and held by
`TestHooksNeverImportServeOrBridge` in `internal/cli/imports_test.go`, which
reads `go list -deps` of `internal/hooks` and fails on `internal/serve`,
`internal/bridge` or `.../go-sdk/mcp`. This table is the measured half of the
same claim. Against the 7.5 ms of 2026-09-17 14:16: that run was the main
checkout, where the barrier's path comparison ends before the worktree lookup;
the two rows are not comparable and both are far under the target of 72 ms.

#### 2026-09-18 19:40 — the same four cases after the fixture was re-pathed

The committed fixture is not the one that produced the table above: the `before`
binary was moved out of a session directory to `%TEMP%\loomux-1b-2\` and the
fixture rewritten to name it. Same binaries, same payload, fewer runs — `-n 5`
instead of `-n 20`, one cold and five warm:

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| before: `loomux version` (start floor, no SDK) | 39.5 ms | 6.5 ms | 6.0 ms | 6.6 ms | [0] |
| after: `loomux version` (start floor, SDK linked) | 8.0 ms | 6.5 ms | 5.5 ms | 6.6 ms | [0] |
| before: `loomux hook pre-tool-use` (Edit on README.md) | 14.9 ms | 9.2 ms | 9.0 ms | 9.5 ms | [0] |
| after: `loomux hook pre-tool-use` (Edit on README.md) | 10.0 ms | 9.0 ms | 8.5 ms | 9.5 ms | [0] |

**Reading.** The fixture still works and the conclusion is unchanged — but the
numbers are not the same numbers, and calling this a reproduction would be too
strong. The before/after gap on the hook is **0.2 ms here against 0.9 ms above**,
and the two floors are equal here where they differed by 1.2 ms above. Both
passes say the same thing — the gap is inside the noise and the hook did not
move — and this one says it more plainly, on five warm runs instead of twenty
and on a busier machine (the cold column shows it). Where the two disagree, the
twenty-run pass above is the measurement; this one is the proof that the
re-pathed fixture runs.

### 3. The bridge handshake, with a running service and with a cold one

**Method.** A stdlib-only Go probe in a scratch directory starts
`loomux mcp --channel local`, writes `initialize` as one line of JSON on its
stdin and stamps the moment the matching reply arrives on its stdout; the clock
starts before the process does, so the number includes the bridge's own start.
`LOOMUX_STATE_DIR` points at a scratch directory holding a copy of the real
`registry.toml` (11 areas), so nothing here touches the real service. Warm:
21 runs against a service that was already up. Cold: `loomux serve stop`, then
one run, three times.

| case | first run | warm median | warm min | warm max |
|---|---:|---:|---:|---:|
| `initialize`, service already running (21 runs) | 10.4 ms | 8.5 ms | 7.5 ms | 10.5 ms |
| `initialize`, no service (3 runs: 9.0 / 9.1 / 10.0 ms) | — | 9.1 ms | 9.0 ms | 10.0 ms |

**Reading.** The two are the same number, and that is the design and not an
accident: `bridge.Run` offers the five tools out of `internal/mcptools` itself
and nudges the service in a goroutine (`go func() { ensure(...) }()` in
`bridge.go`), so no start ever sits inside the host's handshake. A host sees an
answering server in about 9 ms whether a service exists or not, which is the
bridge's own start floor plus one round trip over its pipes. The cold price is
paid by the first `tools/call` instead — see 4.

### 4. One `brain_search` through the bridge against the command line

**Method.** Same probe, same scratch state directory. After the handshake it
sends one `tools/call` for `brain_search` and stamps the reply; the command line
is `loomux brain search latenz --profile keyword -n 5 --channel local` timed with
`Measure-Command`. Both sides pin `profile`, `n` and `channel`, because the two
fronts default differently (`n` is 10 over MCP and 5 on the command line) and
because `keyword` is the one profile whose warm spread does not bury a
millisecond: the entry of 2026-09-16 19:10 shows `fast` scattering 236–373 ms.
21 runs each, the first shown separately. The qmd daemon on port 8765 was up and
warm throughout; five connections were resident on it.

| case | first run | warm median | warm min | warm max |
|---|---:|---:|---:|---:|
| `brain_search` through the bridge, service running | 55.5 ms | 49.5 ms | 44.5 ms | 61.0 ms |
| `loomux brain search`, directly | 61.5 ms | 44.5 ms | 37.5 ms | 60.7 ms |
| `brain_search` through the bridge, no service (3 runs) | — | 310.1 ms | 310.0 ms | 310.2 ms |
| the service announcing itself after a detached start (3 runs) | — | 41.9 ms | 41.3 ms | 43.1 ms |

**Reading.**

1. **The two hops cost 5 ms end to end, and 11–13 ms of work.** The call through
   the bridge is 49.5 ms against 44.5 ms for the command line, 5.0 ms more, with
   overlapping warm ranges (44.5–61.0 against 37.5–60.7). But the command line
   pays a process start the bridge call does not, so the bare answer is the
   44.5 ms less the start floor, and the two hops — JSON-RPC over the host's
   pipes, then HTTP with a bearer token to the service — are the difference.
   **Which floor to subtract is not free of an instrument.** Taken with the same
   `Measure-Command` as the 44.5 ms, the floor is 8.4 ms (measurement 1) and the
   hops are 13.4 ms; taken with `dev bench-hooks`, which starts the process
   itself, it is 6.4 ms (measurement 2) and the hops are 11.4 ms. The honest
   figure is the range: **11–13 ms**, and no subtraction across two instruments
   is tighter than that. Either way it is a millisecond figure against an answer
   that qmd dominates.
2. **A cold service costs 260 ms, and 250 of them are a poll interval, not a
   start.** The three cold runs are 310.0, 310.2 and 310.1 ms — a spread of
   0.2 ms, which is the signature of a quantised wait rather than of work. The
   service itself is up and has written `serve.json` 41.3–43.1 ms after a
   detached start. What the bridge waits for is `waitForService`, which polls at
   `StartTick = 250 ms` (`internal/bridge/connect.go`), so a service ready after
   42 ms is noticed at 250 ms. **This is the cheapest open lever in the stage:**
   a shorter first tick, or a backing-off one, would take about 200 ms off the
   first call after a cold start without touching anything else. It is not a
   defect — 250 ms was taken from qmd's handshake on purpose, to have one waiting
   rhythm in the project — and it is not changed here.
3. **The machine has to settle first.** An earlier batch of ten warm runs, taken
   immediately after the very first cold start, gave handshakes of 20–118 ms and
   calls of 101–161 ms. The tables above are the settled state, measured after
   qmd had answered once. The earlier batch is reported because it is what a
   session's first minute looks like, not because it measures anything.
## 2026-09-18 19:30 - graph ask Response Time and the Sidecar

Query response time of `loomux graph ask "write barrier refuse" --limit 8 --no-refresh`
on this repository (267 files, 2,982 nodes, 9,637 edges), comparing the warm path
with the `ask-index.json` sidecar against querying without the sidecar (names and
paths only), plus a cold start.

### Measurements

| case | figure | command |
|---|---:|---|
| `graph ask`, warm with sidecar (10 runs, median) | 47.6 ms (range 46.5 - 59.2 ms) | `./bin/loomux.exe graph ask "write barrier refuse" --limit 8 --no-refresh` |
| `graph ask`, warm without sidecar (10 runs, median) | 37.5 ms (range 35.6 - 44.1 ms) | (with `ask-index.json` moved away) `./bin/loomux.exe graph ask "write barrier refuse" --limit 8 --no-refresh` |
| `graph ask`, cold (fresh process, graph and sidecar cached) | 52.7 ms | `./bin/loomux.exe graph ask "write barrier refuse" --limit 8 --no-refresh` |

### Reading

1. **The sidecar cost vs. benefit on a 3,000-node repository.**
   Warm queries with the sidecar take ~48 ms; without the sidecar (evaluating names
   and paths without body inverted index), queries take ~37 ms. Deserializing the
   1MB `ask-index.json` payload takes ~10-12 ms of JSON decode time. On a repository
   of this size (~3,000 nodes), parsing symbol names in memory is fast enough that
   the sidecar load time exceeds the token matching savings.
2. **Why the sidecar exists nonetheless.**
   As documented in the design spec and Graft reference, at 30,000+ nodes
   live extraction of all function bodies on every query is prohibitively slow
   (~45% query time reduction observed in Graft at scale). The sidecar scales
   to large multi-package codebases where full body rescanning is impossible
   within interactive query budgets. The difference here is recorded honestly as
   a property of repository size rather than rounded down or hidden.


## 2026-09-18 21:20 - One Sorted Term Order Per Query Instead of Three Per Document

### What was measured

`ask.Run` over a synthetic graph of 3,000 nodes and 2,999 edges with a five-word
query, before and after the query's term order moved out of the per-document
score (`lexical` called `overlap` twice and `bm25` once, each sorting the same
terms again) into one `newQuestion` per query. Throwaway benchmark, not kept in
the tree, `-benchmem -count=5` on the reference machine (AMD Ryzen 7 9800X3D,
Windows x86_64).

### Measurements

| case | ns/op (median of 5) | B/op (median of 5) | allocs/op |
|---|---:|---:|---:|
| baseline, order built per document | 8,318,985 (range 7.49 - 8.64 ms) | 3,881,243 | 18,167 |
| change, one order per query | 8,510,965 (range 7.98 - 9.58 ms) | 3,148,879 | 9,167 |

The CLI path was measured too, 10 warm runs of
`graph ask "how does the write barrier decide what to refuse and why" --limit 5`
on this repository: 77 ms median before, 79.5 ms after. Process start dominates
there and the difference is noise.

### Reading

1. **The allocation count halves and the time does not move.** 18,167 -> 9,167
   allocations per query and 19% fewer bytes, while the wall clock stays inside
   the run-to-run spread of +-1 ms. Sorting four strings is cheap; doing it 9,000
   times is measurable only in the allocator, not in the clock.
2. **The change is recorded as removed redundant work, not as a speed-up.** At
   3,000 nodes there is no time win to claim. Whether the allocation saving
   becomes a time saving at 30,000 nodes is untested and must not be assumed.


## 2026-09-19 00:15 - The Sidecar Version Check the Freshness Probe Now Does

### What was measured

`loomux graph ask` on a clean tree, where the freshness record says nothing has
drifted and the probe returns without rebuilding. That is the only path the
change touches: `EnsureFresh` now calls `lexicon.Usable`, one open and one
partial read of `ask-index.json` decoded as far as its `version` field, before
it may return early.

Both binaries built from source to separate paths, `86fa192` (v1.1.0) as the
baseline and `f160d73` plus this branch's working tree as the change, measured
against the same root and the same graph state: a checkout of this repository at
`86fa192`, 282 files, 3,073 nodes, 9,884 edges, sidecar on disk. Query
`"retry backoff" --limit 1`, three warm-up runs discarded, then 10 runs timed
with `Measure-Command` on the reference machine (AMD Ryzen 7 9800X3D, Windows
x86_64).

### Measurements

| case | median | min | max |
|---|---:|---:|---:|
| baseline, record clean is enough | 51.2 ms | 49.1 ms | 53.6 ms |
| change, record clean plus sidecar version | 49.9 ms | 48.9 ms | 53.1 ms |

### Reading

1. **The added read does not show.** The change measures 1.3 ms *below* the
   baseline, which is inside the run-to-run spread of both (the ranges overlap
   almost entirely) and is therefore noise, not a speed-up. The cost is one
   `open` plus a few hundred buffered bytes against a 50 ms process whose time
   is dominated by start-up and by reading the graph.
2. **There is no cold case to measure.** A cold ask has no usable graph state
   and rebuilds; the build reads and hashes every file and took 283 ms on this
   root. The one extra read is on the warm path by construction, and on the
   cold path it is not reached.
3. **`Usable` is not `Read`.** It decodes to the `version` field and stops.
   Measuring the whole-file `Read` instead would have been a different and
   larger number; that is why the check exists as its own function.


## 2026-09-19 00:31 — `loomux dev bench` on loomux and the open-source corpus

Repo `loomux`, branch `open-source-matrix`, `loomux dev bench --dir . --warm 3`
after the corpus run `--corpus docs/de/open-source-matrix.md --languages 25
--warm 3`. Each hook receives a Claude Code payload for an edit of
`cmd/loomux/main.go`, so post-tool-use runs its real lanes. Reference machine
(AMD Ryzen 7 9800X3D, Windows x86_64).

### loomux

| Component | Cold | Warm median | Warm min | Warm max | Exit codes |
|---|---:|---:|---:|---:|---|
| pre-tool-use | 11.1 ms | 9.0 ms | 8.5 ms | 11.0 ms | 0 |
| post-tool-use | 557.6 ms | 509.4 ms | 507.1 ms | 530.0 ms | 0 |
| graph build | 259.8 ms | 252.7 ms | 250.1 ms | 293.4 ms | 0 |

### Corpus

243 repositories across 25 languages, one skipped
(`membraneframework/membrane_core`: a `|` in a path cannot be checked out on
Windows). post-tool-use exited 0 in 137 repositories and 2 in 95; 11 had no
sample file. Details per repository: [Open-Source Benchmark
Matrix](benchmarks/matrix.md).

### Reading

1. **post-tool-use is the edit's cost, and it is the lanes.** About 0.5 s on
   loomux is `go vet ./...` on a warm build cache; the hook itself stays in
   the pre-tool-use range. An earlier pilot of this tool reported 11.5 ms for
   post-tool-use; that run sent no payload, so no lane ran, and its numbers
   are not kept.
2. **Exit 2 is loomux blocking, not the benchmark failing.** Sampled by hand:
   `go vet` finds issues in gorm, `ruff` in celery. Whether ruff applies the
   repository's own configuration there was not checked.
3. **The baseline speedup of 1.1x on loomux compares loomux with itself**:
   this repository's `.claude/settings.json` calls the same binary.

## 2026-09-19 00:40 - Stage G3: the Start Floor, and graph_find_code Through the Bridge Against graph ask

### What was measured

Two things. **The control:** the start floor (`loomux --version`) and the binary
size of the base `11222c8` against the change `c25575f` (HEAD of `code-g3`), both
built with `go build -o … ./cmd/loomux` into a scratch directory, the base from a
detached worktree removed afterwards. G3 links nothing new — `extract/golang`
has been in the binary through `internal/cli` since G2a — so the expectation was
equal within the noise. `Measure-Command` in PowerShell, each pass one first
run and 20 warm runs, over three passes: pass 1 the base alone, right after its
build; pass 2 the change alone, right after its build; pass 3 both binaries back
to back. **Cold and warm in the first table come from different passes:** cold
is the first run after the build (pass 1 for the base, pass 2 for the change),
warm is pass 3. **The new figure:** the answer
time of `graph_find_code` and `graph_check_freshness` through `loomux mcp`
against `loomux graph ask` and `graph check` on the command line.

The area was the `code-g3` worktree at HEAD, scope `project/loomux`, its graph
built once with `graph build --root <worktree>` (339 files, 3,719 nodes, 11,833
edges). Every command ran with `LOOMUX_STATE_DIR` pointing at an isolated scratch
state directory whose `registry.toml` holds only that area, so the bridge
started its own service and never met the user's (a newer bridge replaces an
older running service). The MCP side is a throwaway Go client outside the tree
(`mcp.CommandTransport` over `loomux mcp`), which times each `tools/call` with
`time.Since` after the handshake: 11 calls per session, the first shown
separately, the median of the other ten as warm. The command line is
`Measure-Command`, one first run and ten warm runs. Afterwards `serve stop` with
the same state directory; no loomux process was left running.

### Measurements

| case | cold (1st run) | warm median | warm min | warm max |
|---|---:|---:|---:|---:|
| base `11222c8`: `loomux --version` (20 runs) | 48.5 ms | 8.0 ms | 7.7 ms | 8.6 ms |
| change `c25575f`: `loomux --version` (20 runs) | 48.6 ms | 8.0 ms | 7.7 ms | 11.4 ms |

The warm medians of the other passes: base 7.9 ms in pass 1 (range 7.6–10.1 ms),
change 8.5 ms in pass 2 (range 7.9–9.5 ms). Pass 3's first runs were 15.2 ms
(base) and 9.6 ms (change); they are not cold, since each binary had run
before. Pass 3 is the one reported warm because it is the only pass that ran
both binaries in one sitting, on the same machine state; passes 1 and 2 were
taken separately, at different moments.

Binary size: 17,839,616 bytes before, 17,863,680 bytes after (+24,064 bytes,
+0.13 %).

| case | first run / call | warm median (10) | warm min | warm max |
|---|---:|---:|---:|---:|
| `graph ask "run" --root <worktree>` (limit 8, no source) | 69.6 ms | 57.6 ms | 55.9 ms | 62.4 ms |
| `graph ask "run" --limit 5 --source --root <worktree>` (the MCP defaults) | 58.4 ms | 58.6 ms | 56.9 ms | 62.6 ms |
| `graph_find_code` `{"scope": "project/loomux", "query": "run"}`, service started by this bridge | 342.6 ms | 62.2 ms | 59.4 ms | 80.4 ms |
| `graph_find_code`, same, second session against the running service | 88.6 ms | 67.8 ms | 63.2 ms | 79.4 ms |
| `graph check --root <worktree>` | 104.3 ms | 104.1 ms | 99.6 ms | 109.3 ms |
| `graph_check_freshness` `{"scope": "project/loomux"}`, running service | 122.7 ms | 96.3 ms | 93.6 ms | 117.2 ms |

The handshake (`Connect`, including the bridge's process start) took 9.5, 8.5
and 8.0 ms in the three sessions. All answers had `isError: false`; the check
said `OK` every time.

### Reading

1. **The control holds; the run-to-run spread it rests on is 0.1–0.5 ms.**
   The same binary's warm median moved between passes by 0.1 ms (base,
   7.9 -> 8.0 ms) and 0.5 ms (change, 8.5 -> 8.0 ms). Base and change are
   0.0 ms apart in pass 3, where both ran in one sitting, and 0.6 ms apart
   across passes 1 and 2 — no more than those two drifts added up. The cold
   runs are 0.1 ms apart, and the binary is 24 KB larger for two handlers and
   two tool definitions. G3 did not move the start floor beyond what the passes
   move by themselves.
2. **Through the bridge `graph_find_code` takes 11.6 ms more than its bare
   answer, and this entry does not know why.** The two columns do not measure
   the same thing: the command line includes a process start, the MCP client
   times a call on an open session. Taking the 8.0 ms floor off the equal-work
   command line (58.6 ms) leaves 50.6 ms; the bridge's 62.2 ms is 11.6 ms above
   it. The second session's 67.8 ms has a range overlapping the first's. The
   service reads the graph and sidecar for each call and keeps nothing between
   calls, so a warm service is not a cached answer.
3. **The same subtraction for the drift check gives 0.2 ms, so the two
   differences disagree.** 96.3 ms through the bridge against 104.1 − 8.0 =
   96.1 ms bare. Both calls take the same two hops (JSON-RPC over the pipes,
   HTTP to the service), so the hops alone cannot be both 11.6 ms and 0.2 ms.
   The cause of the 11.6 ms is unmeasured. Unconfirmed candidates: the reply of
   `graph_find_code` carries the inlined source of five hits through both hops
   where the check's reply is one line, and the subtraction mixes two
   instruments (`Measure-Command` against `time.Since`). The check's own
   spread (93.6–117.2 ms) is also wider than either difference.
4. **The first call after a cold start carries the poll tick again.** 342.6 ms
   holds the same 250 ms wait the entry of 2026-09-18 15:30 found (310 ms there,
   its call included): the bridge polls for the service at `StartTick = 250 ms`.
   250 ms plus one warm call (62.2 ms) accounts for about 312 ms; the remaining
   ~30 ms is one unrepeated sample and is not taken apart here. A fresh session
   against a running service pays 88.6 ms on its first call, about 20 ms above
   warm.

## 2026-09-19 08:31 — check and post-edit of Stage 2a, Lanes From Presets

Repository `loomux`, worktree `.claude/worktrees/fusion-migration-teil-2-79865c`,
branch `claude/fusion-migration-teil-2-79865c` on `c2e640c`. Stage 2a replaces
the hard-wired post-edit lanes with lanes that `[verify]` and the embedded
presets lay out, and adds `loomux check`. `c2e640c` names runs in UTC (see
reading 2); a first pass at 08:26 on `2aceea1`, before that commit, found the
cost it removes.

**Goal.** Show what `check` and post-edit cost now, post-edit against its
baseline (`master` `86fa192`, v1.1.0, hard-wired lanes) and against the 1a
target of 72 ms; and show that the presets are parsed on first use, not at start.

**Method.** `bin/loomux.exe dev bench-hooks -n 20 testdata/bench/2a-check.json`,
one pass, one cold run per case and 20 warm. `change.exe` is built from `c2e640c`,
`base.exe` from `86fa192` in a throwaway worktree, both with Go 1.27, copied to
`%TEMP%\loomux-2a-bench`. The world is a copy of
`testdata/cases/2a-worlds/go-only` in the same directory (`check precommit`
creates `.loomux/state/cover` in it). Every tool is the faketool
(`internal/dev/faketool/_faketool`, built as `go.exe`, `gofmt.exe`, `uv.exe`,
`uvx.exe`) at the front of `PATH`, answering from the world's `faketool.json`
through `LOOMUX_FAKE_TOOL_FIXTURE`; the lanes therefore measure loomux and
a process start, not `go vet`. Payload: an `Edit` on the world's `a.go`, named
absolutely as Claude names it. The case file names this machine's paths
(`C:/Users/micro/AppData/Local/Temp/loomux-2a-bench/…`), as `1a-hooks.json`
did. Before measuring, both binaries were run once against a fixture whose
`go` answers exit 1: both blocked with 2, so neither skips its lanes silently.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| check-precommit (change) | 45.2 ms | 16.5 ms | 15.1 ms | 18.5 ms | [1] |
| check-show (change: load, presets, plan, no child) | 9.2 ms | 7.9 ms | 7.0 ms | 9.0 ms | [0] |
| post-edit-go (change: lanes from presets) | 17.0 ms | 15.9 ms | 15.0 ms | 18.4 ms | [0] |
| post-edit-go (base 86fa192: hard-wired lanes) | 31.5 ms | 29.2 ms | 27.0 ms | 37.3 ms | [0] |
| loomux version (change, start floor) | 7.7 ms | 6.5 ms | 6.0 ms | 7.0 ms | [0] |
| loomux version (base 86fa192, start floor) | 8.5 ms | 6.3 ms | 6.0 ms | 7.0 ms | [0] |

The first pass, 08:26, same case file and setup, `change.exe` built from
`2aceea1` (run ID in local time):

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| check-precommit (2aceea1) | 63.8 ms | 34.1 ms | 33.1 ms | 39.5 ms | [1] |
| check-show (2aceea1) | 26.5 ms | 25.1 ms | 24.7 ms | 27.0 ms | [0] |
| post-edit-go (2aceea1) | 36.5 ms | 33.0 ms | 32.0 ms | 35.9 ms | [0] |
| post-edit-go (base 86fa192) | 32.0 ms | 27.5 ms | 26.0 ms | 39.5 ms | [0] |
| loomux version (2aceea1) | 7.5 ms | 6.0 ms | 6.0 ms | 6.5 ms | [0] |
| loomux version (base 86fa192) | 7.9 ms | 6.0 ms | 5.6 ms | 7.5 ms | [0] |

The presets' parse alone (`BenchmarkLoadPresets` in
`internal/verify/presets_test.go`, `parsePresets(presetsText)`):

```
$ go test ./internal/verify/ -bench LoadPresets -run '^$' -benchmem -count 5
BenchmarkLoadPresets-16    	    7468	    135381 ns/op	  159596 B/op	    1757 allocs/op
BenchmarkLoadPresets-16    	    9241	    135440 ns/op	  159593 B/op	    1757 allocs/op
BenchmarkLoadPresets-16    	    9342	    139665 ns/op	  159593 B/op	    1757 allocs/op
BenchmarkLoadPresets-16    	    7710	    146760 ns/op	  159593 B/op	    1757 allocs/op
BenchmarkLoadPresets-16    	    8348	    147377 ns/op	  159593 B/op	    1757 allocs/op
```

The start trace, three runs on `c2e640c`:

```
$ GODEBUG=inittrace=1 bin/loomux.exe version 2>&1 | grep -E 'verify|child'
init github.com/xidus90/loomux/internal/verify/commit @1.5 ms, 0 ms clock, 4416 bytes, 16 allocs
init github.com/xidus90/loomux/internal/verify @2.5 ms, 0 ms clock, 88 bytes, 2 allocs
init github.com/xidus90/loomux/internal/verify/commit @1.4 ms, 0 ms clock, 4416 bytes, 16 allocs
init github.com/xidus90/loomux/internal/verify @2.4 ms, 0 ms clock, 88 bytes, 2 allocs
init github.com/xidus90/loomux/internal/verify/commit @2.0 ms, 0 ms clock, 4416 bytes, 16 allocs
init github.com/xidus90/loomux/internal/verify @3.0 ms, 0 ms clock, 88 bytes, 2 allocs
```

### Reading

1. **post-edit is under its baseline and far under the 1a target.** 15.9 ms
   warm against 29.2 ms for the hard-wired lanes of `86fa192` (13.3 ms less,
   disjoint warm ranges 15.0–18.4 against 27.0–37.3) and against 72 ms. Cold it
   is 17.0 ms against 31.5 ms. Both start floors are 6.3–6.5 ms.
2. **The local time zone was most of the own time, and the run ID loaded it.**
   In the first pass `check all --show`, which starts no child, stood 19.1 ms
   above its floor, and post-edit was 5.5 ms slower than the baseline.
   `verify.NewRunID` formatted `time.Now()` in local time, and Windows loads the
   zone on first use — the ~18 ms the 2026-09-17 entry removed from the start
   path, paid at run time instead. `c2e640c` formats `now.UTC()`:
   `check all --show` falls from 25.1 to 7.9 ms (1.4 ms above the floor),
   post-edit from 33.0 to 15.9 ms, `check precommit` from 34.1 to 16.5 ms. The
   cover files' names now carry the UTC time.
3. **The presets cost 0.14 ms.** 135–147 µs and 1,757 allocations per parse,
   paid once per process on first use; that is a small part of what
   `check all --show` still spends above the floor, and not a lever.
4. **Nothing parses at start.** The `verify` line is the `sync.OnceValues`
   closure: 0 ms clock, 88 bytes, 2 allocations. `@2.4–3.0 ms` is the offset
   from the process start at which that init ran, not its duration; the
   duration is the clock column. `verify/commit` (16 allocations, 0 ms) is not
   the presets.
5. **`check precommit` exits 1 by construction of the world.** The coverage lane
   reads a profile the measuring `go test` should write at a path that carries
   the run ID; the standalone faketool cannot know that path, so the lane fails
   at once with "the measuring run did not write it". Lint and test ran
   (`ok`), types is not applicable for Go; the 16.5 ms is loomux's path with
   every lane started, not a failing tool's runtime.

## 2026-09-20 02:20 — The Stop Gate and the Subagent Hooks of Stage 2c

Repository `loomux`, worktree `.claude/worktrees/github-versioning-releases-cli-ca4f58`,
branch `sdd-2c` on `745e301`. Windows 11, AMD Ryzen 7 9800X3D (16 threads),
Go 1.27, `bin/loomux.exe` built from that tree.

**Goal.** Show what `hook stop` costs at a turn end that finds nothing new —
the path that runs at every turn end and starts no tool — and what the two
subagent hooks cost, which are `git ls-remote` and little else. The plan's
target for the no-op path is about 300 ms on this repository.

**Method.** `bin/loomux.exe dev bench-hooks -n 20 testdata/bench/2c-hooks.json`,
one cold run per case and 20 warm ones. Two worlds are staged into
`%TEMP%\loomux-2c-bench` and built by `cases.BuildGitWorld` from the `git.toml`
declarations in `testdata/bench/2c-worlds/` — `dev bench-hooks` builds no world,
so a throwaway `main` under `internal/dev/_benchworld/` called
`cases.StageWorld(src, dst)` and then `cases.BuildGitWorld(dst)`, and was
deleted again: `clean`, a three-file Go module
whose working tree is its last commit and whose `origin` is the bare
`.origin.git` beside it, and `dirty`, the same module with one uncommitted
`b.go`. Payloads are `stop.json` (`hook_event_name` `Stop` and `session_id`)
and `subagent.json` (`hook_event_name` `SubagentStart`, `session_id`,
`agent_id` and `agent_type`) in the same directory; `subagent-stop` was fed the
same `subagent.json`, which it may be, because no hook reads the event name.
Every tool is the faketool
(`internal/dev/faketool/_faketool`, built as `go.exe`, `gofmt.exe`, `uv.exe`,
`uvx.exe`) at the front of `PATH`, answering from `dirty/faketool.json` through
`LOOMUX_FAKE_TOOL_FIXTURE`. The chain case runs in `seq` mode with a step that
deletes the session's state file before the gate: after one pass the gate
remembers the green tree and every warm run would take the no-op path instead,
and after a block the give-up rule would let every fourth run through. The
reset is measured on its own so it can be subtracted.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| stop-unchanged (small world, clean tree) | 121.3 ms | 121.3 ms | 114.7 ms | 169.4 ms | [0] |
| stop-fake-chain (small world, changed tree, faketool) incl. state reset | 159.1 ms | 143.8 ms | 139.2 ms | 154.1 ms | [0 2] |
| state reset only (control for the chain case) | 11.0 ms | 10.0 ms | 9.5 ms | 11.0 ms | [0] |
| subagent-start (small world, local bare remote) | 105.9 ms | 101.4 ms | 97.0 ms | 218.0 ms | [0] |
| subagent-start + subagent-stop (small world, local bare remote) | 200.6 ms | 201.6 ms | 193.5 ms | 215.1 ms | [0 0] |
| git rev-parse HEAD (one git process, small world) | 17.0 ms | 13.0 ms | 12.0 ms | 16.3 ms | [0] |
| git ls-remote origin (local bare remote) | 37.1 ms | 36.8 ms | 35.5 ms | 39.7 ms | [0] |
| git ls-remote origin (GitHub, this worktree) | 991.5 ms | 1017.1 ms | 946.6 ms | 1506.5 ms | [0] |
| subagent-start (this worktree, GitHub remote) | 1008.0 ms | 1038.7 ms | 997.0 ms | 1098.8 ms | [0] |
| loomux version (start floor) | 8.5 ms | 6.5 ms | 6.0 ms | 12.5 ms | [0] |

The gate on **this** repository (7,341 tracked files), measured in a first pass
at 02:05 before this task had written a file, so the working tree was clean and
the gate took its no-op path — the case is in `2c-hooks.json` and repeats once
this entry is committed:

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| stop-unchanged (this worktree, clean tree, 7,341 files) | 176.1 ms | 169.5 ms | 164.4 ms | 178.0 ms | [0] |
| git ls-remote origin (GitHub, this worktree) | 992.7 ms | 951.2 ms | 912.4 ms | 1048.3 ms | [0] |
| subagent-start (this worktree, GitHub remote) | 1072.8 ms | 1035.2 ms | 997.4 ms | 1085.9 ms | [0] |
| subagent-start + subagent-stop (this worktree, GitHub remote) | 2059.0 ms | 2068.6 ms | 2027.5 ms | 2206.0 ms | [0 0] |
| loomux version (start floor) | 8.5 ms | 6.5 ms | 6.0 ms | 7.3 ms | [0] |

It does repeat: the whole case file, run at 02:52 against the committed tree,
reads 172.4 ms warm (167.7–217.1, cold 190.0) for that case, and every other
case within its earlier range.

The fingerprint alone (`BenchmarkContentTree` in
`internal/gitwork/gitwork_test.go`, over this repository):

```
$ go test ./internal/gitwork/ -bench ContentTree -run '^$' -benchmem -count 5
BenchmarkContentTree-16    	      10	 110482790 ns/op	 1440883 B/op	    2091 allocs/op
BenchmarkContentTree-16    	      10	 112377110 ns/op	 1439894 B/op	    2090 allocs/op
BenchmarkContentTree-16    	      10	 114720320 ns/op	 1439653 B/op	    2088 allocs/op
BenchmarkContentTree-16    	      10	 107556430 ns/op	 1438593 B/op	    2086 allocs/op
BenchmarkContentTree-16    	      10	 109411100 ns/op	 1439019 B/op	    2087 allocs/op
```

### Reading

1. **The no-op path is 169.5 ms on this repository, well under the 300 ms the
   plan budgets.** It starts no tool: 6.5 ms process start, 107–115 ms for the
   content fingerprint, and the rest is `Head` (`check-ignore`, `rev-parse
   --is-inside-work-tree`, `rev-parse HEAD^{commit}`) and `TreeOf` for the
   base's tree — eight git processes in all, four of them inside `ContentTree`
   (`rev-parse --git-path index`, `add -A`, `rm --cached`, `write-tree`).
2. **What the gate pays for is eight git starts plus a content term that
   scales.** One git process in the small world is 13.0 ms warm, measured as a
   control case above. Eight of them are 104 ms; with the 6.5 ms process start
   that is 110.5 ms of the small world's 121.3 ms, and the remaining ~11 ms is
   the index work over three files. This repository costs 169.5 ms: the same
   ~110 ms floor, ~11 ms of index work and about 48 ms more for scanning and
   hashing 7,341 files. A first version of this entry divided that residual by
   eight and called it 14 ms per process; the control case measures 13.0 ms
   and the arithmetic no longer has to stand in for it. The floor is the same
   for every repository, the content term is not: ten times the files would add
   roughly 480 ms to it, and a machine with a slower `git` start moves the
   floor instead.
3. **The 224–245 ms the function's comment records is a different measurement,
   not a lost one.** It is in
   `docs/.superpowers/specs/2026-09-19-loomux-stufe-2c-design.md`, in the table
   "Kosten, gemessen auf diesem Repo (7.322 Dateien, warm, Windows 11)": the
   index copied and `git add -A`, `git rm --cached` and `git write-tree` each
   started from a shell, timed around the whole sequence, with no run count
   given. Re-run today exactly that way, five warm runs on this worktree, it
   reads 150, 137, 139, 140 and 141 ms — about 30 ms above
   `BenchmarkContentTree`, which times the same four steps in-process. So the
   shell protocol explains 30 ms of the gap; the remaining ~85 ms between then
   and now is not explained by anything measured here — same machine, same
   repository, 19 files more. The number to carry forward is 107–115 ms
   in-process, 137–150 ms shell-driven, 176–190 ms for the whole hook cold.
4. **The subagent hooks are `git ls-remote`, and nothing else is worth naming.**
   Against a local bare remote `subagent-start` is 101.4 ms, of which
   `ls-remote` is 36.8 ms; against GitHub over the network it is 1,038.7 ms, of
   which `ls-remote` is 1,017.1 ms. A subagent's whole lifecycle — start plus
   stop, two snapshots — is 201.6 ms locally and 2,068.6 ms against GitHub. The
   1.30–1.45 s the design spec's table records for `ls-remote origin` on
   2026-09-19 — the same table as in reading 3 — were the same remote and the
   same 24 refs; between 0.95 s and 1.45 s is the network, not the code, and
   the 10 s deadline is what bounds it.
5. **The chain adds little of its own.** The gate with a changed tree and four
   fake lanes is 143.8 ms including a 10.0 ms state reset, so about 134 ms
   against the 121.3 ms of the no-op path in the same world: some 13 ms for
   `detect`, the config, the plan and four child processes. It exits 2 because
   the coverage lane sees `b.go` uncovered — by construction of the world, not
   because a lane is slow. What a real turn end costs is the real tools: with
   the real Go toolchain on `PATH` instead of the faketool, `bin/loomux.exe
   check precommit` through the same harness is 627.0 ms warm median of 5
   (cold 641.8 ms) in the `dirty` world and 622.9 ms warm median of 10 (cold
   719.0 ms) in a copy of `testdata/cases/2a-worlds/go-only`.
6. **A faketool that is not really on `PATH` is not visible in the numbers.**
   A first pass put `C:/Users/...` on `PATH` from a POSIX shell, where the colon
   is the separator: the entry fell apart, the real Go toolchain answered, and
   the chain read 768.9 ms with no sign that anything was wrong. Only
   `exec.LookPath` in a probe said which `go.exe` was being started. Every
   measurement that uses the faketool should check the resolved path once
   before it counts.

## 2026-09-22 11:42 — Stage 3a: reconcile, reindex and area add Against the Python Reference

Repository `loomux`, worktree `.claude/worktrees/planung-von-3-c56c81`, branch
`claude/planung-von-3-c56c81` on `4ca3fc2` (code as in `01c2a2a`; task 19
adds tests only). Reference: ultra-brain `loomux-3-source` (`3cc72d2`), run as
`ultra-brain/.venv/Scripts/brain-mcp.exe`, Python 3.14.7.

**Goal.** The three measurements of the stage: `reconcile` warm, `reindex`
cold and warm, `area add` on an empty repository, each against the Python
form. No target was set (spec 3a, "Messen").

**Method.** Neither command has an area filter; both walk the whole registry.
So the runs did **not** touch the real state but copies: the five repositories
of the writable areas, `.git` included, under `%TEMP%\lx3a\<world>\repos`; the
three read-only areas in place (read only); a copy of the registry with the ten
areas both tools know (without `project/loomux`, which the reference skips for
lack of a `.brain.toml`); `%LOCALAPPDATA%\brain` copied. One world per tool,
both from the same template. **With qmd** (2.8.3), but with `QMD_CONFIG_DIR`,
`INDEX_PATH` and `XDG_CACHE_HOME` pointed into the world — every world starts
with an empty qmd index, and `reindex` runs `qmd update` against it. Timing: a
PEP 723 script, `time.perf_counter_ns` around `subprocess.run`, one first run
and ten warm runs per case, median of the warm ones. Order per world: `reindex`
(its first run is the cold one), then `reconcile`. `area add`: a fresh
`git init` repository and a fresh state directory with an empty
`registry.toml` for every run (the reference fails without it, ruling B5),
only the command timed. Machine: AMD Ryzen 7 9800X3D, Go 1.27.0
`windows/amd64`.

**What "cold" means here.** The first `reindex` of a fresh world: an empty qmd
index and, for loomux, an empty state directory — loomux reads the stat cache
from its own directory only (parity record, finding S5), so the catch-up pass
ahead of the index run hashes every source. The reference finds its cache in
the copied `%LOCALAPPDATA%\brain`. Not cold in the file-cache sense: the copies
had just been made.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| loomux reindex (10 areas) | 14983.0 ms | 4105.9 ms | 3770.6 ms | 4803.7 ms | [0] |
| brain-mcp reindex (10 areas) | 37490.6 ms | 27822.7 ms | 25086.0 ms | 32346.4 ms | [0] |
| loomux reconcile (10 areas) | 1083.0 ms | 962.2 ms | 924.2 ms | 1006.1 ms | [0] |
| brain-mcp reconcile (10 areas) | 13087.1 ms | 12717.3 ms | 11775.6 ms | 17791.9 ms | [0] |
| loomux area add -y | 286.9 ms | 274.6 ms | 264.3 ms | 290.2 ms | [0] |
| loomux area add -y -no-reindex | 32.6 ms | 33.5 ms | 30.0 ms | 40.3 ms | [0] |
| brain-mcp init -y | 947.9 ms | 922.9 ms | 900.4 ms | 973.6 ms | [0] |

And loomux alone over all eleven areas (the world with `project/loomux`,
whose register holds 2147 sources after the first run — 3064 checked over eleven areas less 917 over the ten):

| case | first run | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| loomux reindex (11 areas) | 12381.4 ms | 6837.7 ms | 6460.2 ms | 7893.9 ms | [0] |
| loomux reconcile (11 areas) | 1552.3 ms | 1385.0 ms | 1375.5 ms | 1446.8 ms | [0] |
| loomux --version (start floor, 20 warm) | 11.0 ms | 7.6 ms | 7.3 ms | 13.3 ms | [0] |

### Reading

1. **`reindex` is 6.8 times as fast warm, 2.5 times cold.** 4105.9 ms against
   27822.7 ms, with separate warm ranges (3770.6–4803.7 against
   25086.0–32346.4). Cold it is 14983.0 ms against 37490.6 ms; the loomux
   figure carries the catch-up pass hashing every source for want of a cache,
   and the first `qmd update` against an empty index.
2. **`reconcile` is 13.2 times as fast warm.** 962.2 ms against 12717.3 ms.
   Neither side hashes anything in the warm runs (`0 davon gehasht`), so the
   difference is the walk and the stat calls, not hashing. loomux counted 917
   sources, the reference 910; the seven are findings S1, S2 and S6 of the
   parity record (+14 packages in the review centre, −4 in `ultraloom`, −3
   through the junction in `space`).
3. **`area add` is 3.4 times as fast although it does more.** 274.6 ms against
   922.9 ms — and `area add` indexes at the end (ruling of task 15), `brain init`
   does not. Without the index run it is 33.5 ms, 27.5 times as fast; the index
   run over the empty repository therefore costs 241.1 ms, nearly all of it
   qmd's start.
4. **`project/loomux` costs 2731.8 ms per `reindex` and 422.8 ms per
   `reconcile`.** That is the area without `[index]`, which walks the whole
   repository including `testdata/` (parity record, finding S3). An `[index]`
   taking only `docs/wiki` would remove the item.
5. **The start floor has not moved.** 7.6 ms warm against 5.5–7.6 ms in the
   entries of 2026-09-17 and 2026-09-19.

**Start time.** `GODEBUG=inittrace=1 loomux --version`, three runs. The new
packages of the stage:

| package | clock | bytes | allocations |
|---|---:|---:|---:|
| `internal/brain/index` | 0 ms (3/3) | 11,512 | 94 |
| `internal/brain/maintenance` | 0 ms (3/3) | 3,576 | 36 |
| `internal/brain/vcs`, `internal/lock` | no init | — | — |

No `init()` and no `//go:embed` in the four packages; what runs at start is
package variables: in `index` three `regexp.MustCompile` over literals
(`document.go:16`, `:21`, `:22`) and the exclusion lists, in `maintenance` one
(`package.go:38`) and small tables. None parses embedded data. The largest
init of loomux is still `internal/cases` with 438 allocations, below the 500
of `TestStartDoesNoWorkInPackageInit`; above 300 there are otherwise only
`encoding/gob` (366–370) and `internal/verify/commit` (322), both present
before 3a.

## 2026-09-23 00:06 — Stage 3b: `apply` and `evidence` on the Start Path

Worktree `.claude/worktrees/stuffe-4-brainstorming-74e215`, branch
`feat/review-cases` on `d3a5303` plus the fixups before it. `loomux approve` makes
`internal/cli` import `internal/brain/apply` for the first time, and with it
`internal/brain/evidence`. Both declared their regular expressions as package
variables (`regexp.MustCompile`), and `TestStartDoesNoWorkInPackageInit` failed.
The change compiles each of them on first use through `sync.OnceValue`, which is
the pattern `internal/cases/state.go` already follows.

**Method.** `before.exe` is `d3a5303` with the four files carrying the
expressions (`evidence/evidence.go`, `apply/patch.go`, `apply/frontmatter.go`,
`apply/pyyaml.go`) taken from `04de33c`. `after.exe` is the branch. Both were built
with Go 1.27 into the session scratchpad. `loomux dev bench-hooks <fixture> -n 20`,
one pass, one cold run per case and 20 warm, in this worktree. The hook payload is
`testdata/bench/edit-readme.json`: its path lies outside the worktree, so both
binaries refuse with exit 2 on the same path. The init counts come from
`GODEBUG=inittrace=1 loomux version`.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| before: loomux version (regexps compiled at start) | 40.5 ms | 7.5 ms | 7.0 ms | 13.0 ms | [0] |
| after: loomux version (regexps compiled on first use) | 8.5 ms | 6.5 ms | 6.0 ms | 7.5 ms | [0] |
| before: loomux hook pre-tool-use (Edit on README.md) | 13.5 ms | 10.0 ms | 9.5 ms | 17.5 ms | [2] |
| after: loomux hook pre-tool-use (Edit on README.md) | 12.0 ms | 9.1 ms | 8.6 ms | 15.0 ms | [2] |

| package init | before | after |
|---|---:|---:|
| `internal/brain/evidence` | 646 allocs, 74,544 bytes | 20 allocs, 720 bytes |
| `internal/brain/apply` | 1,299 allocs, 148,096 bytes | 26 allocs, 1,896 bytes |

### Reading

The allocation counts are the finding: both packages are back far below the 500 of
the start rule. The warm medians move by about 1 ms, which is within the spread of
a single pass. The cold `before` row is the first start of a freshly written
binary and carries the file cache, so it is not the price of the expressions.

## 2026-09-23 01:54 — Stage 3b: cases, case and approve --defer Against the Python Reference

Worktree `.claude/worktrees/stuffe-4-brainstorming-74e215`, branch
`feat/review-cases` on `07d0881` (the code as in `a882946`; the commits between
add tests only). Reference: ultra-brain `master` on `3cc72d2` (the tag
`loomux-3-source`), run as `ultra-brain/.venv/Scripts/brain-mcp.exe`, Python
3.14.7 — the venv's console script rather than `uv run brain`, so that uv's own
start is not in the figure. loomux built with Go 1.27.0 `windows/amd64`.
Machine: AMD Ryzen 7 9800X3D.

**Goal.** The three commands of the stage that decide nothing or only look:
`cases`, `case <id>` and `approve --defer <id>`, each against its Python form.
No target was set.

**Method.** A throwaway world, never the real state: the world
`testdata/cases/3b-worlds/cases-three` copied into the session scratchpad,
`{{WORLD}}` replaced, and the repository its `git.toml` declares built by hand
(one commit `base`, the source changed in the working tree). Three cases in two
scopes, the first with package and proposal. Both tools point at it —
`LOOMUX_STATE_DIR` and `BRAIN_STATE_DIR` at the world, `LOOMUX_LEGACY_BRAIN_DIR`
at an empty directory, `QMD_CONFIG_DIR`, `INDEX_PATH` and `XDG_CACHE_HOME` into
the world. Both tools printed the same stdout for all three commands, the
same (empty) stderr and exit 0, once the CRLF that Python's text-mode stdout
writes on Windows is folded to LF; loomux writes LF. Timing: a PEP 723
script, `time.perf_counter_ns` around `subprocess.run`, per case one first run
and ten warm runs, median of the warm ones. The world's files hashed the same
before and after the series (`b0d3974bcb2db8b6`), and nothing under
`%LOCALAPPDATA%\brain` or `%LOCALAPPDATA%\loomux` was newer than a marker set
before it.

**What "cold" means here.** For loomux, the first start of a freshly written
copy of the binary, one copy per command. For brain-mcp, the first run of the
series, not a cold start of the file cache: brain-mcp had run in the same
session about an hour before. Only the very first row (`cases`) stands out.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| loomux cases | 66.1 ms | 26.9 ms | 26.2 ms | 28.4 ms | [0] |
| brain-mcp cases | 1879.7 ms | 852.2 ms | 835.3 ms | 873.8 ms | [0] |
| loomux case \<id\> | 67.3 ms | 26.9 ms | 25.8 ms | 28.2 ms | [0] |
| brain-mcp case \<id\> | 848.5 ms | 841.1 ms | 827.7 ms | 879.9 ms | [0] |
| loomux approve --defer \<id\> | 72.9 ms | 26.5 ms | 25.8 ms | 33.0 ms | [0] |
| brain-mcp approve --defer \<id\> | 847.2 ms | 842.3 ms | 821.3 ms | 917.6 ms | [0] |
| loomux --version (start floor, 20 warm) | 9.6 ms | 6.9 ms | 6.8 ms | 12.8 ms | [0] |

**Start time.** `GODEBUG=inittrace=1 loomux --version`, three runs, the same
in each. The packages of the stage:

| package | clock | bytes | allocations |
|---|---:|---:|---:|
| `internal/brain/apply` | 0 ms (3/3) | 1,896 | 26 |
| `internal/brain/evidence` | 0 ms (3/3) | 720 | 20 |
| `internal/brain/vcs` | no init | — | — |
| `internal/cases` | 0–0.5 ms | 43,216 | 446 |
| `internal/cli` | 0 ms (3/3) | 1,752 | 10 |

### Reading

1. **All three are 31 to 32 times as fast warm.** 26.9 ms against 852.2 ms for
   `cases`, 26.9 against 841.1 for `case`, 26.5 against 842.3 for
   `approve --defer`, with separate ranges. The Python figures are nearly the
   same for all three although the work differs, which points at what they
   share: the interpreter's start and the imports of `brain` (not measured
   apart).
2. **loomux spends about 20 ms above its start floor.** 26.5–26.9 ms against
   6.9 ms for `--version`, alike for three commands of different work — listing,
   reading one case with its package, and deciding nothing. What they share is
   reading the registry and the areas' declarations and walking the review
   centre; which of those costs the 20 ms was not measured.
3. **Cold, loomux is 66–73 ms.** A freshly written binary, first start;
   what the 39–46 ms above the warm figure consist of was not measured.
4. **The start rule holds.** `TestStartDoesNoWorkInPackageInit` stays green,
   and no package of the stage comes near its 500 allocations. The largest init
   of loomux is still `internal/cases`, at 446 (438 at stage 3a): the headroom
   below 500 is shrinking. Above 300 there are otherwise only `encoding/gob`
   (367–373) and `internal/verify/commit` (322), as before.

## 2026-09-23 20:21 — Stage 3c: brain check, lint and wiki types Against the Reference

Worktree `.claude/worktrees/recursing-bartik-b2d7a1`, branch
`claude/mit-3c-fortsetzen-94e9a0` at `c081369`. Reference: ultra-brain at
`3cc72d2` (the tag `loomux-3-source`); `brain check` against `brain-3c.exe`,
which the user built from the tag on 2026-09-23, `lint` and `types` against
`ultra-brain/.venv/Scripts/brain-mcp.exe`, Python 3.14.7. loomux built with
Go 1.27.0 `windows/amd64`. Machine: AMD Ryzen 7 9800X3D.

**Goal.** The stage's three reading commands against this machine's real
registry, each against its reference. No target was set.

**Method.** Read only. Both sides on the same registry: `LOOMUX_STATE_DIR` and
`LOOMUX_LEGACY_BRAIN_DIR` on `%LOCALAPPDATA%\brain`, the reference's state
directory, ten areas. Outputs compared (record `parity/stufe-3c.md`,
"Selbstnutzung"): `check all --notes` byte for byte equal, `lint --scope all`
and `types` equal but for the CRLF Python's text-mode stdout writes on Windows.
Timing as in 3b: a PEP 723 script, `time.perf_counter_ns` around
`subprocess.run`, per case one first and ten warm runs, the median of the warm
ones. Cold means, for loomux, the first start of a freshly written copy of the
binary, one copy per command; for the reference, the first run of the series.

| Case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| brain.exe check all | 50.7 ms | 44.0 ms | 38.7 ms | 46.4 ms | [0] |
| loomux brain check all | 79.9 ms | 38.2 ms | 36.2 ms | 66.7 ms | [0] |
| brain-mcp lint --scope all | 936.5 ms | 930.2 ms | 914.6 ms | 1025.9 ms | [0] |
| loomux lint --scope all | 88.9 ms | 44.8 ms | 43.9 ms | 60.7 ms | [0] |
| brain-mcp types | 908.5 ms | 919.5 ms | 903.6 ms | 932.1 ms | [0] |
| loomux wiki types | 59.3 ms | 20.7 ms | 20.0 ms | 28.5 ms | [0] |
| loomux --version (start floor) | 47.1 ms | 11.9 ms | 11.4 ms | 12.3 ms | [0] |

**Start time.** `GODEBUG=inittrace=1 loomux --version`. The stage's packages:

| Package | clock | bytes | allocations |
|---|---:|---:|---:|
| `internal/brain/check/okf` | 0 ms | 72 | 2 |
| `internal/brain/check/run` | 0 ms | 256 | 2 |
| `internal/brain/check/house` | no init | — | — |
| `internal/brain/wiki` | 0 ms | 15,680 | 131 |
| `internal/cli` | 0 ms | 1,752 | 10 |

### Reading

1. **`brain check` is level with the Go binary.** 38.2 ms against 44.0 ms warm,
   both over ten areas and the same rules; the spans touch at the edge (loomux
   up to 66.7 ms in one outlier).
2. **`lint` and `types` are 21 and 44 times as fast as Python.** 44.8 ms against
   930.2 ms, 20.7 ms against 919.5 ms. The Python figures are nearly equal
   although `types` does less; as in 3b that points at the start of the
   interpreter and its imports (not measured apart).
3. **The start floor is 11.9 ms**, against 6.9 ms in the 3b measurement. The
   binary measures 20.2 MB today; what the five milliseconds consist of is not
   measured.
4. **The start rule holds.** `okf` compiled its date-heading regex at package
   start (68 allocations, 0.5 ms) and since this stage does so on first use
   (2 allocations). `wiki` stands at 131 with its three regexes from stage 1a;
   the lint's new reader builds its patterns inside the call.

## 2026-09-23 20:21 — Post-edit on a .go file, before the blast monitor

Repository `loomux`, the main checkout, branch `feat/code-g4b` on `219dd0f`
(the tree clean apart from the two new case files). `bin/loomux.exe` built
from that tree with Go 1.27.0 `windows/amd64`. Machine: AMD Ryzen 7 9800X3D.
The baseline for the post-edit hook on a Go file, before the hook gains a
monitor; the after part is the entry
[2026-09-23 22:55](#2026-09-23-2255--post-edit-on-a-go-file-with-the-blast-monitor).

**Goal.** What the post-edit hook costs today for an `Edit` on a `.go` file in
this repository, with the real tools and the real graph. No target was set.

**Method.** The protocol, which the after part repeats:

1. `go build -o bin/loomux.new.exe ./cmd/loomux`, then
   `go run ./cmd/loomux dev swap-binary --dir bin`.
2. `bin/loomux.exe graph build`: 555 files, 6551 nodes, 21706 edges (5996
   contains, 12330 calls, 3380 imports), 897 ms;
   `.loomux/state/graph/wiring.json` is 7,161,354 bytes.
3. One pass that is discarded (pass 1 below).
4. The recorded pass:
   `bin/loomux.exe dev bench-hooks testdata/bench/g4b-post-edit.json -n 10`,
   one cold run and ten warm.

Payload `testdata/bench/edit-go.json`: a `PostToolUse` `Edit` on
`internal/code/blast/reach.go`, named absolutely as Claude names it, with
`old_string` equal to `new_string`; the file is not touched (`git status`
showed no modification after the series). The case file names this machine's
paths. The edit profile runs, per `bin/loomux.exe check edit --show`, the Go
lint lane `on_file`: `go vet ./...` over the whole repository and
`{loomux} check gofmt {file}`, threaded. The shell lane does not apply to a
`.go` file. No faketool: these are the real `go vet` and the real graph.

| pass (time) | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| 1 (20:20, right after build, swap and graph build; discarded) | 2090.0 ms | 1796.7 ms | 988.1 ms | 1965.7 ms | [0] |
| 2 (20:20) | 685.9 ms | 657.0 ms | 622.8 ms | 969.0 ms | [0] |
| 3 (20:21, recorded) | 644.2 ms | 638.8 ms | 623.6 ms | 964.0 ms | [0] |

The two lanes alone, at 20:22, through the same bench from a throwaway case
file (not committed):

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| go vet ./... (bare) | 628.2 ms | 612.9 ms | 595.8 ms | 647.1 ms | [0] |
| loomux check gofmt reach.go (bare) | 11.0 ms | 7.5 ms | 7.0 ms | 8.5 ms | [0] |

### Reading

1. **The baseline is 638.8 ms warm.** The first run of the recorded pass 3
   read 644.2 ms; the cold figure after a rebuild is the first run of pass 1,
   2090.0 ms. Passes 2 and 3 agree to
   18 ms in the median (657.0 against 638.8), and each has one run near
   965 ms; that is the resolution of this setup. A monitor that costs a few
   tens of milliseconds sits near it.
2. **The first pass after a rebuild is not the baseline.** Pass 1, straight
   after `go build`, `swap-binary` and `graph build`, read 1796.7 ms warm and
   still settled within the pass (min 988.1 ms). What refilled was not
   measured apart. The after part discards its first pass the same way.
3. **`go vet ./...` is the hook's cost.** 612.9 ms bare against 638.8 ms for
   the whole hook; gofmt on one file is 7.5 ms and runs beside vet. About
   26 ms remain for loomux's own path. The whole hook, 638.8 ms, is 40 times
   the 15.9 ms of the 2026-09-19 entry, which measured the same hook against
   the faketool in the `go-only` world and therefore not `go vet`.

## 2026-09-23 21:48 — blast-audit over the last 50 commits

Repository `loomux`, branch `feat/code-g4b` on `cad8255`; `bin/loomux.exe`
built from that tree with Go 1.27.0 `windows/amd64` (`go build -o
bin/loomux.new.exe ./cmd/loomux`, then `go run ./cmd/loomux dev swap-binary
--dir bin`). Machine: AMD Ryzen 7 9800X3D. Replayed history: the last 50
first-parent commits of `master` up to `e3cab29c` (v2.7.0), `d902e8fa` to
`e3cab29c`. All 50 have one parent: the history is linear, so each replayed
diff is one commit, the unit the pre-commit lane audits.

**Goal.** How often `loomux check blast-audit` would have been red on this
repository's recent history, for the thresholds 3, 5 and 10, with and without
`--skip-test-callers`. The numbers decide the default threshold and the
counting mode of the pre-commit lane. No target was set.

**Method.** One detached worktree under the session scratchpad, walked oldest
first. For each commit `c` the diff is `c~1...c` (`--base c~1` with the
worktree on `c`), audited against two graphs:

- **graph of `c~1`** (the parent state), still on disk from the step before;
  the design text asks for this one;
- **graph of `c`**, rebuilt after the checkout. This is what the lane sees:
  `check graph-fresh` runs first and `query.Build` reads the tree on disk,
  which holds the staged change.

The verdict is read from stdout, not from the exit code: `check blast-audit`
also exits 1 on an error (`internal/cli/check.go`), so the exit code cannot
tell a finding from a failure. `R` is a report that starts with
`blast audit:`, `G` one that starts with `no area at or above`, `E` anything
else or anything on stderr. The area count comes from one extra call with
`--threshold 999999`, since only the clean report prints it. The script,
verbatim:

```sh
#!/bin/sh
# Replays blast-audit over the last 50 first-parent commits of master, oldest
# first, in one detached worktree. For each commit c and the diff c~1...c it
# audits twice: once against the graph of c~1 (the parent state, still on disk
# from the previous step) and once against the graph of c (what the lane sees
# after graph-fresh has rebuilt from the tree). Each for N = 3, 5, 10, with and
# without test callers. The verdict comes from stdout, not the exit code:
# check blast-audit exits 1 both for findings and for errors.
# R = red, G = green, E = error. areas comes from a call whose threshold no
# seed reaches, since only the clean report prints the area count.
set -u
repo="C:/Users/micro/Documents/#GIT/loomux"
bin="$repo/bin/loomux.exe"
out="C:/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux/e346e4ea-8ea8-4af2-b360-1800a79cfdec/scratchpad/e2"
wt="$out/wt"
mkdir -p "$out/n3"
: > "$out/times.txt"
: > "$out/builds.txt"

# audit <n> <skip> <tag>: prints R, G or E; saves the N=3 reports.
audit() {
  t0=$(date +%s%N)
  "$bin" check blast-audit --root "$wt" --base "$c~1" --threshold "$1" $2 >"$out/stdout" 2>"$out/stderr"
  t1=$(date +%s%N)
  echo $(( (t1 - t0) / 1000000 )) >> "$out/times.txt"
  if [ "$1" = 3 ]; then cat "$out/stdout" "$out/stderr" > "$out/n3/$c-$3${2:+-skip}.txt"; fi
  if [ -s "$out/stderr" ]; then echo E
  elif head -1 "$out/stdout" | grep -q '^blast audit:'; then echo R
  elif head -1 "$out/stdout" | grep -q '^no area at or above'; then echo G
  else echo E; fi
}

areas() {
  "$bin" check blast-audit --root "$wt" --base "$c~1" --threshold 999999 2>/dev/null \
    | sed -n 's/.*(\([0-9]*\) areas)$/\1/p'
}

build() {
  t0=$(date +%s%N)
  "$bin" graph build --root "$wt" >/dev/null 2>"$out/build-stderr" || echo "build failed at $1" >&2
  t1=$(date +%s%N)
  echo $(( (t1 - t0) / 1000000 )) >> "$out/builds.txt"
}

commits=$(git -C "$repo" rev-list --first-parent --reverse -n 50 master)
first=$(echo "$commits" | head -1)
git -C "$repo" worktree add -q --detach "$wt" "$first~1" || { echo "worktree add failed" >&2; exit 2; }
build "$first~1"
echo "commit,areas_parent,p3,p5,p10,p3s,p5s,p10s,areas_c,n3,n5,n10,n3s,n5s,n10s"
for c in $commits; do
  if ! git -C "$wt" checkout -q --detach "$c"; then echo "$c,ERR_CHECKOUT"; continue; fi
  line="$c,$(areas)"
  for skip in "" "--skip-test-callers"; do
    for n in 3 5 10; do line="$line,$(audit "$n" "$skip" parent)"; done
  done
  build "$c"
  line="$line,$(areas)"
  for skip in "" "--skip-test-callers"; do
    for n in 3 5 10; do line="$line,$(audit "$n" "$skip" own)"; done
  done
  echo "$line"
done
git -C "$repo" worktree remove --force "$wt"
git -C "$repo" worktree prune
```

The run went from 21:48 to 21:53. No call ended in `E`, no checkout failed,
and every graph build succeeded.

**Red commits out of 50.** "all callers" is the default count, "non-test" is
`--skip-test-callers`. In brackets the share of the commits whose diff had at
least one area: 34 for the graph of `c`, 18 for the graph of `c~1` (a new
file is not in the parent graph and so is no area there).

| graph | counting | N = 3 | N = 5 | N = 10 |
|---|---|---:|---:|---:|
| of `c` (what the lane sees) | all callers | 3/50 = 6 % (3/34) | 1/50 = 2 % (1/34) | 0/50 |
| of `c` (what the lane sees) | non-test | 3/50 = 6 % (3/34) | 1/50 = 2 % (1/34) | 0/50 |
| of `c~1` (parent) | all callers | 2/50 = 4 % (2/18) | 1/50 = 2 % (1/18) | 1/50 = 2 % (1/18) |
| of `c~1` (parent) | non-test | 1/50 = 2 % (1/18) | 0/50 | 0/50 |

Areas per commit (graph of `c`): 16 commits with 0 areas (docs, releases,
recorded cases), 14 with 1–3, 15 with 4–9, 5 with 10 or more; median 2, 175
in total. Red at N = 3 by that size: 0 of 16, 1 of 14, 1 of 15, 1 of 5.

**Runtime.** One `check blast-audit --base c~1` call: median 129 ms over 600
calls (min 98 ms, max 663 ms), both graphs together. One `graph build` of a
commit: median 655 ms over 51 builds (min 510 ms, max 2520 ms, the first
build in the fresh worktree).

**The red commits, read by hand.** Four commits appear in the table; the
reports at N = 3:

1. `245be87d` feat(cli): add reindex, embed, reconcile and area add — graph
   of `c`, `internal/cli/index.go [none]: refusesArguments in-degree 4` and
   `internal/cli/maintenance.go [none]: reportReconcileError in-degree 3,
   germanCount in-degree 3`. **Noise.** The commit brings `index_test.go`,
   `index_catchup_test.go` and `maintenance_test.go` along; they drive the
   commands through the CLI's command table, a call through a function value
   the graph does not draw, so no test file "reaches" the area.
2. `d2c45688` feat(vcs): commit named paths onto the current ref — both
   graphs, `internal/brain/vcs/vcs.go [stale]: run in-degree 3`. **Noise,
   with a correct signal.** `run` became a one-line wrapper around the new
   `runWith`; the unchanged `vcs` tests that reach it still cover the old
   behaviour, and there is nothing a changed test would add.
3. `a0de5983` feat(apply): write only inside the vault and name every touched
   file — graph of `c`, `internal/brain/apply/place.go [none]: refuse
   in-degree 7, touch 3, gate 6, isScaffoldName 3`. **Noise.** The new file
   arrives with a 774-line `place_test.go` that calls these methods on 34
   lines, but through a `*place` from a helper; the graph does not resolve
   method calls on a local variable, so the test reaches nothing.
4. `df3fb48c` feat(blast): add Resolve with Go package filter, EdgeWalk and
   Quote — only the graph of `c~1`, at every N,
   `internal/code/blast/index.go [stale]: New in-degree 11`. **Noise, from
   the parent graph.** The new `edgewalk_test.go` and `quote_test.go` call
   `New`, but do not exist in the parent graph; in the graph of `c` the area
   is green.

No red commit was a real finding. Three of four are blind spots of the call
graph (a call through a function value, a method call on a local variable,
a test missing from the parent graph); the fourth is a refactor that its
unchanged tests already cover.

### Reading

1. **The red rate is low at every setting.** In the lane's view (graph of
   `c`) N = 3 is red on 3 of 50 commits, N = 5 on 1, N = 10 on none. The fear
   "almost every commit red" did not materialise on this history.
2. **`--skip-test-callers` changes nothing in the lane's view.** The three
   red areas at N = 3 are `none` areas, and a `none` area has no test caller
   by definition; the in-degree is the same in both counts. The switch
   matters only for `stale` areas, and the one `stale` red in the lane's view
   (`d2c45688`, `run`) has three non-test callers.
3. **The parent graph is the wrong baseline for the lane.** It misses new
   files as areas (18 against 34 commits with areas) and produces the one red
   the lane would not (`df3fb48c`), because the tests of the same commit are
   not in it. The numbers for the decision are the graph-of-`c` rows.
4. **Every red here is noise.** The finding the lane is meant to catch did
   not occur in these 50 commits; the reds come from edges the graph does not
   draw. At N = 5 one red remains (`a0de5983`), at N = 10 none.
5. **The cost is small.** 129 ms per audit call plus a graph rebuild of about
   650 ms when the tree drifted; that is the price per commit.

## 2026-09-23 22:55 — Post-edit on a .go file, with the blast monitor

The after part of the baseline in [2026-09-23 20:21 — Post-edit on a .go file, before the blast monitor](#2026-09-23-2021--post-edit-on-a-go-file-before-the-blast-monitor), measured by its protocol.

Measured at 22:55–22:58 on `6dcbc00e`, the same branch, after the monitor was
built into the hook (`internal/hooks/blast_monitor.go`); the tree clean. Same
machine, same Go, same case file and payload. The protocol of that entry:
`go build -o bin/loomux.new.exe ./cmd/loomux`, `go run ./cmd/loomux dev
swap-binary --dir bin`, `bin/loomux.exe graph build` (571 files, 6777 nodes,
22506 edges: 6206 contains, 12795 calls, 3505 imports; 2.527 s;
`wiring.json` 7,415,396 bytes = 7.07 MiB), one discarded pass, then the
recorded pass.

The monitor runs after the lanes, only when none is red, and only for a `.go`
file. With the file unchanged its symbols hash as the graph has them, so it is
silent: the hook wrote nothing on `stdout`. For the second series the body of
`Reach` in `reach.go` was changed (`var hits []Hit` → `var hits []Hit //
edited`, the edit the Go benchmark below replays); `go vet` and gofmt stay
green, and the hook wrote once by hand:

```text
{"hookSpecificOutput":{"additionalContext":"[graph] internal/code/blast/reach.go: changed Reach; callers in other files:\n  EdgeWalk (internal/code/blast/edgewalk.go)\n  Radius (internal/code/blast/radius.go)\n  signal (internal/code/blast/radius.go)","hookEventName":"PostToolUse"}}
```

The change was reverted afterwards and the graph rebuilt from the clean tree.

| series | pass (time) | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---|---:|---:|---:|---:|---|
| before (20:21, `219dd0f`) | 3, recorded | 644.2 ms | 638.8 ms | 623.6 ms | 964.0 ms | [0] |
| after, file unchanged (monitor silent) | 1 (22:55, right after build, swap and graph build; discarded) | 1088.6 ms | 720.0 ms | 707.7 ms | 1639.7 ms | [0] |
| after, file unchanged (monitor silent) | 2 (22:55, recorded) | 716.2 ms | 722.7 ms | 694.8 ms | 798.2 ms | [0] |
| after, file unchanged (monitor silent) | 3 (22:55) | 724.3 ms | 762.1 ms | 703.5 ms | 961.8 ms | [0] |
| after, body of `Reach` changed (monitor speaks) | 1 (22:56; discarded) | 717.2 ms | 713.1 ms | 678.8 ms | 770.6 ms | [0] |
| after, body of `Reach` changed (monitor speaks) | 2 (22:56, recorded) | 836.0 ms | 738.1 ms | 691.4 ms | 859.7 ms | [0] |

The two lanes alone, at 22:55, through the same bench from a throwaway case
file (not committed), as before:

| case | before (20:22) warm median | after (22:55) cold (1st run) | after warm median | after warm min | after warm max |
|---|---:|---:|---:|---:|---:|
| go vet ./... (bare) | 612.9 ms | 672.8 ms | 678.0 ms | 643.1 ms | 848.3 ms |
| loomux check gofmt reach.go (bare) | 7.5 ms | 10.7 ms | 8.0 ms | 7.5 ms | 19.0 ms |

**The monitor alone**, `go test ./internal/hooks/ -run '^$' -bench
BlastAside -benchtime 20x -count=3` at 22:58 (read and decode `wiring.json`,
parse the edited file, compare the hashes, walk one level in). The median of
the three counts; `BenchmarkBlastAsideScaled` writes this repository's graph
`k` times over, every id and path under `copyN/`, and edits the body of
`Reach` in `copy0/`:

| case | `wiring.json` | ms per call (median of 3) | the three |
|---|---:|---:|---|
| unchanged (silent) | 7.07 MiB | 24.7 | 24.1, 24.7, 25.1 |
| body of `Reach` changed | 7.07 MiB | 27.6 | 25.0, 28.3, 27.6 |
| k = 1 | 7.41 MiB | 30.4 | 30.6, 28.8, 30.4 |
| k = 2 | 14.81 MiB | 58.0 | 58.5, 57.4, 58.0 |
| k = 5 | 37.03 MiB | 148.5 | 148.0, 148.5, 153.2 |
| k = 10 | 74.07 MiB | 298.8 | 323.3, 298.8, 295.1 |
| k = 20 | 148.7 MiB | 625.2 | 625.2, 681.8, 589.6 |

A single `-count=1` run just before, at 22:57, read 23.3 and 22.3 ms for the
first two rows but 86.4 ms for k = 2 and 312.4 ms for k = 5; it is left out of
the table as the outlier it was against the three that followed. (k = 1 is
the same graph as the unchanged row with every id and path prefixed by
`copy0/`; hence 7.41 against 7.07 MiB.)

**The graph lane's commands**, each 11 calls in a shell loop timed with
`date +%s%N`, the first call as cold and the other ten as warm:

| command | state | cold | warm median | warm min | warm max | exit |
|---|---|---:|---:|---:|---:|---|
| `check graph-fresh` | drift before every call (a comment in `reach.go` toggled), so every call rebuilt | 582 ms | 597.5 ms | 578 ms | 647 ms | 0, `graph rebuilt` |
| `check graph-fresh` | no drift | 88 ms | 73.5 ms | 68 ms | 105 ms | 0, `graph is fresh` |
| `check blast-audit --cached` | the change to `reach.go` staged; pass 1, 22:56 (discarded) | 190 ms | 164.5 ms | 106 ms | 362 ms | 1 |
| `check blast-audit --cached` | the same, pass 2, 22:57 (recorded) | 105 ms | 110.5 ms | 104 ms | 125 ms | 1 |
| `git rev-parse --git-path MERGE_HEAD HEAD` | first probe call of `GraphReady` | 65 ms | 46.5 ms | 43 ms | 57 ms | 0 |
| `git diff --cached --quiet` | second probe call, something staged | 58 ms | 52.5 ms | 47 ms | 111 ms | 1 |

`blast-audit --cached` at the default threshold 3 found
`internal/code/blast/reach.go [stale]: Reach in-degree 3`; with the preset's
`--threshold 5` the same index is green (`no area at or above 5 callers lacks a
changed test (1 areas)`, exit 0). The index was unstaged and the file reverted
afterwards; no commit ran while anything was staged.

### Reading

1. **The hook went from 638.8 to 722.7 ms warm, and `go vet` took most of
   it.** `go vet ./...` alone rose from 612.9 to 678.0 ms: the tree grew from
   555 to 571 files between the two measurements. What is left for loomux's
   own path rose from about 26 to about 45 ms (722.7 − 678.0). The difference,
   about 19 ms, fits the 24.7 ms the monitor costs on its own; the monitor
   runs after the lanes, not beside them, so it adds in full.
2. **A monitor that speaks costs no more than one that is silent, as far as
   the hook can tell.** 738.1 against 722.7 ms warm is inside the 40 ms two
   passes of the same series differ by (722.7 and 762.1). In the Go benchmark
   the changed body costs 2.9 ms more (27.6 against 24.7): the parse and the
   comparison run either way, only the walk and the text are extra.
3. **The cost is the decode of `wiring.json`, linear in its size.** 58.0,
   148.5, 298.8 and 625.2 ms for 14.81, 37.03, 74.07 and 148.7 MiB are 3.9 to
   4.2 ms per MiB. The design text assumed 12 ms for `store.Read` on a graph
   of 3 MB; at about 4 ms per MiB that holds, but this repository's graph is
   7.07 MiB today.
4. **The monitor passes 100 ms at about 25 MiB of wiring.json (k ≈ 3.4).**
   Interpolated linearly between k = 2 (14.81 MiB, 58.0 ms) and k = 5 (37.03
   MiB, 148.5 ms), 4.07 ms per MiB; no measured point lies at 100 ms. That is
   about 3.5 times this repository's graph.
5. **The graph lane costs a rebuild when the tree drifted, and about 180 ms
   when it did not.** `graph-fresh` is 597.5 ms with a rebuild and 73.5 ms
   without; `blast-audit --cached` is 110.5 ms warm, its first pass spread
   more (106 to 362 ms). The plan-time probe `GraphReady` adds its two git
   calls, 46.5 and 52.5 ms, to every check that plans a graph lane, also
   when the lane ends `not-applicable` — except the first check, a missing
   `wiring.json`, which costs no git call.

## 2026-09-24 03:55 — Stage 4a-1: the Hook Path With `[modules]`, and `config list`

Worktree `.claude/worktrees/fusion-migration-teil-2-79865c`, branch
`feat/stage-4-config` at `58b4a8d`. `before.exe` is `master` at `1d42d2f`
(the branch's merge base), `after.exe` the branch; both built with Go 1.27.0
`windows/amd64` into the session scratchpad. Machine: AMD Ryzen 7 9800X3D.

**Goal.** `hook pre-tool-use` does not read `[modules]` and must not move;
`hook post-tool-use` reads it once per call and may rise by the noise at
most. `config list`, a command a human types, is measured on its own. The
baseline the plan names, 7.5 ms warm for `pre-tool-use` (2026-09-17 14:16),
was taken in the main checkout on a 5.5 ms start floor; the before/after pair
below is the comparison that holds, since both run in the same pass.

**Method.** `bin/loomux.exe dev bench-hooks <cases> -n 20`, one pass, one
cold run per case and 20 warm, in this worktree with its own
`.loomux/config.toml` and the machine's registry. Payloads: an `Edit` of
`README.md` (pre-tool-use, allowed), a `PostToolUse` `Edit` of `README.md`
(no lane for Markdown outside the wiki) and one of `docs/wiki/index.md`. For
`config list`: `dev bench-hooks <case> -n 10` on a freshly copied binary, so
the cold run is the first start of that file. The init counts come from
`GODEBUG=inittrace=1 loomux version`. Two mutation rounds ran before this
pass, none during it.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| before: loomux hook pre-tool-use (Edit on README.md) | 16.5 ms | 13.0 ms | 12.5 ms | 13.5 ms | [0] |
| after: loomux hook pre-tool-use (Edit on README.md) | 15.0 ms | 12.7 ms | 12.0 ms | 26.7 ms | [0] |
| before: loomux hook post-tool-use (Edit on README.md) | 17.9 ms | 17.0 ms | 16.0 ms | 17.8 ms | [0] |
| after: loomux hook post-tool-use (Edit on README.md) | 17.4 ms | 17.5 ms | 16.4 ms | 33.0 ms | [0] |
| before: loomux hook post-tool-use (Edit on docs/wiki/index.md) | 17.4 ms | 16.6 ms | 16.5 ms | 18.8 ms | [0] |
| after: loomux hook post-tool-use (Edit on docs/wiki/index.md) | 18.8 ms | 17.8 ms | 16.5 ms | 32.0 ms | [0] |
| before: loomux version (start floor) | 10.6 ms | 10.7 ms | 10.0 ms | 13.0 ms | [0] |
| after: loomux version (start floor) | 11.0 ms | 10.5 ms | 10.0 ms | 24.9 ms | [0] |
| loomux config list (this repository, n = 10) | 24.0 ms | 11.0 ms | 10.0 ms | 11.8 ms | [0] |

| package init (after) | clock | bytes | allocations |
|---|---:|---:|---:|
| `internal/config/edit` | 0 ms | 104 | 2 |
| `internal/tui` | 0 ms | 72 | 2 |
| `internal/config/schema` | no init | — | — |
| `internal/cli` | 0 ms | 1,752 | 10 |

### Reading

1. **`pre-tool-use` did not move.** 12.7 ms against 13.0 ms warm, the ranges
   overlap (12.0–26.7 against 12.5–13.5). The 26.7 ms maximum is one outlier;
   the floor rows carry one of the same size in the same pass.
2. **`post-tool-use` rose by 0.5 and 1.2 ms**, within the spread: the
   minimums are 16.4 against 16.0 and 16.5 against 16.5 ms. Reading
   `[modules]` is one more small file read and TOML parse per call.
3. **Against the 7.5 ms of 2026-09-17 the hook is 5.2 ms slower**, and the
   start floor 5.0 ms (10.5 against 5.5 ms). The stage did not cause it:
   `before`, without this stage, stands at the same place. The 3c entry
   already measured the floor at 11.9 ms; the binary is 20.3 MB now.
4. **`config list` costs about what `version` costs** (11.0 against 10.5 ms
   warm): `list` reads the file once and decodes it once; the readers that
   `set` runs through `Validate` are not on its path.
5. **The start rule holds for the new packages**: `edit` and `tui` compile
   their expressions on first use (2 allocations each), `schema` has no
   package init.

## 2026-09-24 10:45 — The write barrier reading open.toml

Measured on `eb83426e` with `internal/brain/guard/open.go` uncommitted on top,
go1.27.0 windows/amd64, this machine. `Decide` now calls `openFiles` on every
writing tool call, before the memory fast path, so its cost lands on the
hook the host runs most often. A throwaway Go benchmark (not committed) ran
`openFiles` over a fresh temporary state directory, `-count 5`; `b.Loop`
warms up on its own, so there is no separate cold figure.

| case | ns/op, 5 runs | median |
|---|---|---:|
| no `open.toml` | 90293, 80033, 74248, 82883, 101659 | 82883 |
| `open.toml` with one entry | 222814, 196172, 203734, 253601, 216453 | 216453 |

### Reading

1. **Without the file the barrier pays about 0.08 ms per call**: one path
   resolution of the state directory and one failed read. With one entry it
   is about 0.22 ms: the read, the TOML decode, and a second resolution for
   the entry.
2. **Against the hook this is noise.** A `pre-tool-use` call spawns a process
   in the tens of milliseconds; 0.2 ms is below the spread of two passes of
   the same series in the entries above.

## 2026-09-24 20:06 — Stage 4a-2: `init --dry-run`, and the Hook Path Beside `internal/setup`

Worktree `.claude/worktrees/beautiful-noether-local-c6dd1a`, branch
`feat/stage-4a-2-init` at `309f592b`. `before.exe` is `master` at
`540ea728`, `after.exe` the branch; both built with Go 1.27.0
`windows/amd64` into the session scratchpad. Since the merge base `3453354e`
`master` has moved by one docs commit; nothing under `cmd`, `internal`,
`third_party`, `go.mod` and `go.sum` changed, so before is the merge base's
code.
Machine: AMD Ryzen 7 9800X3D.

**Goal.** `hook pre-tool-use` must not rise measurably warm: the stage brings
`internal/setup/...` into the binary, and the guard learns `merge-hook`; the
hook path does not reach the installer (`TestHooksNeverImportTheInstaller`).
`init --dry-run`, a command a human types, is measured on its own.

**Method.** `after.exe dev bench-hooks <cases> -n 10`, two passes back to
back and a third at 20:28, one cold run per case and 10 warm, in this worktree with its
`.loomux/config.toml` and the machine's registry. Payload for
`pre-tool-use`: an `Edit` of this worktree's `README.md` (allowed),
`--host claude --root <worktree>`. `init --dry-run --root <worktree>` runs on
a freshly copied binary, so its cold run in the first pass is that file's
first start; for `before.exe` and `after.exe` it is not (both ran once
before for the init counts). `master` has no `init`, so that row exists only
after. The init counts come from `GODEBUG=inittrace=1 loomux --version`. The
mutation round ran before this pass, none during it.

First pass:

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| before: loomux hook pre-tool-use (Edit on README.md) | 16.5 ms | 10.4 ms | 10.0 ms | 11.0 ms | [0] |
| after: loomux hook pre-tool-use (Edit on README.md) | 36.5 ms | 11.8 ms | 10.0 ms | 24.9 ms | [0] |
| before: loomux version (start floor) | 12.0 ms | 9.0 ms | 7.0 ms | 11.2 ms | [0] |
| after: loomux version (start floor) | 9.5 ms | 8.5 ms | 7.5 ms | 9.5 ms | [0] |
| after: loomux init --dry-run (this worktree) | 119.7 ms | 63.7 ms | 58.6 ms | 84.9 ms | [0] |

Second pass:

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| before: loomux hook pre-tool-use (Edit on README.md) | 15.0 ms | 10.5 ms | 10.0 ms | 11.5 ms | [0] |
| after: loomux hook pre-tool-use (Edit on README.md) | 12.5 ms | 10.5 ms | 9.5 ms | 42.0 ms | [0] |
| before: loomux version (start floor) | 9.5 ms | 7.8 ms | 7.5 ms | 14.5 ms | [0] |
| after: loomux version (start floor) | 9.0 ms | 7.0 ms | 7.0 ms | 7.5 ms | [0] |
| after: loomux init --dry-run (this worktree) | 60.1 ms | 58.5 ms | 57.0 ms | 76.5 ms | [0] |

Third pass, 20:28:

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| before: loomux hook pre-tool-use (Edit on README.md) | 16.0 ms | 10.3 ms | 10.0 ms | 19.0 ms | [0] |
| after: loomux hook pre-tool-use (Edit on README.md) | 13.0 ms | 9.8 ms | 9.0 ms | 14.5 ms | [0] |
| before: loomux version (start floor) | 10.0 ms | 7.5 ms | 7.5 ms | 8.0 ms | [0] |
| after: loomux version (start floor) | 8.5 ms | 7.0 ms | 6.5 ms | 27.5 ms | [0] |
| after: loomux init --dry-run (this worktree) | 61.5 ms | 54.7 ms | 52.3 ms | 65.5 ms | [0] |

| package init | before: bytes / allocations | after: bytes / allocations |
|---|---:|---:|
| `internal/setup` | — | 256 / 2 |
| `internal/setup/hostfile`, `gitfiles`, `templates`, `write` | — | no init |
| `internal/brain/maintenance` | 3,576 / 36 | 3,624 / 38 |
| `internal/cli` | 1,752 / 10 | 2,008 / 12 |

### Reading

1. **`pre-tool-use` did not move.** The warm medians after against before:
   11.8 against 10.4 ms in the first pass, 10.5 against 10.5 in the second,
   9.8 against 10.3 in the third. In the first, more than half of the after
   runs were slower than the slowest before run (11.0 ms); the two later
   passes do not show that, and the third reverses it. The minima are equal
   or lower after in all three (10.0/9.5/9.0 against 10.0/10.0/10.0 ms).
   The first pass was the first after the mutation round; a shift the stage
   caused would stand in all three. The maxima of 24.9 and 42.0 ms and the
   36.5 ms cold run are outliers of single runs.
2. **Neither did the start floor**, though the binary grew from 20.7 to
   24.2 MB (8.5/7.0/7.0 against 9.0/7.8/7.5 ms warm).
3. **`init --dry-run` costs 55 to 64 ms warm**, 120 ms cold on the file's
   first start. It reads git's configuration through `git`, looks up five
   tools on the `PATH`, reads the registry and plans every file; for a
   command a human types once per project that is no concern.
4. **The start rule holds**: of the new packages only `internal/setup` has a
   package init (2 allocations); `maintenance` and `cli` grow by 2 each. No
   new init comes near 500.

## 2026-09-26 17:25 — Python Extraction on gotreesitter: the Hook Path, the Binary, and `graph build`

Worktree `.claude/worktrees/g5-python`, branch `feat/graph-python` at
`69ce9fde`. `before.exe` is `master` at `b5c99cf1`, `after.exe` the branch;
both built with Go 1.27.0 `windows/amd64`, `CGO_ENABLED=0`, into the session
scratchpad. gotreesitter v0.55.0. Machine: AMD Ryzen 7 9800X3D.

**Goal.** The branch links the pure-Go tree-sitter runtime
`github.com/odvcencio/gotreesitter` into the binary. Its package `init`
runs on every start, hooks included; design decision E1 accepts about 2 ms
of it and names a trigger: a warm `hook pre-tool-use` median that rises by
3 ms or more on a quiet machine brings back the patched vendored copy as an
option. The post-edit blast monitor stays Go-only, so `post-tool-use` must
not move.

**Method.** `dev bench-hooks <cases> -n 30`, three passes, before and after
alternating, one cold run per case and 30 warm, over a worktree at
`b5c99cf1` (Go only). Each binary ran against a graph it had built itself,
so the blast monitor saw its own extractor stamp. Payloads: an `Edit` of
that worktree's `README.md` for `pre-tool-use`, an `Edit` of
`internal/code/blast/reach.go` for `post-tool-use`. The machine was quiet:
a check run right before gave `before` a warm `pre-tool-use` median of
9.1 ms. Init sums come from `GODEBUG=inittrace=1 loomux --version`, five
runs each.

| case | pass 1 warm median | pass 2 | pass 3 | warm min (1/2/3) |
|---|---:|---:|---:|---:|
| before: hook pre-tool-use | 9.3 ms | 9.7 ms | 10.4 ms | 7.4 / 7.7 / 8.0 ms |
| after: hook pre-tool-use | 11.2 ms | 10.3 ms | 9.4 ms | 8.2 / 8.4 / 8.0 ms |
| before: hook post-tool-use (.go) | 1033.8 ms | 1016.4 ms | 1012.9 ms | 965.8 / 937.0 / 930.4 ms |
| after: hook post-tool-use (.go) | 1017.1 ms | 1012.0 ms | 1028.8 ms | 955.7 / 952.1 / 967.9 ms |
| before: --version | 7.9 ms | 8.9 ms | 8.3 ms | 7.0 / 6.7 / 6.6 ms |
| after: --version | 9.1 ms | 9.2 ms | 9.6 ms | 7.2 / 7.4 / 7.2 ms |

| | before | after |
|---|---:|---:|
| binary size | 24,256,000 B | 36,276,736 B |
| init sum, five runs | 4.03 / 3.25 / 2.94 / 2.55 / 2.00 ms | 7.27 / 4.56 / 5.57 / 3.02 / 4.09 ms |
| packages with init | 141 | 145 |

`graph build` on scratch clones of two Python repositories (never on the
originals):

| repository | files | cold | warm (cache) | `--no-reuse` | `extract.json` |
|---|---|---:|---:|---:|---:|
| `iam_backend` (Django) | 399 Python | 3.61 s | 446 ms (0 parsed) | 1.30 s | 11.0 MB |
| `ultra-brain` | 174 Go + 182 Python | 3.95 s | 478 ms (0 parsed) | 1.36 s | 10.3 MB |

`graph check` on the `ultra-brain` clone: 232 / 187 / 185 ms with the
extract cache, 823 / 762 / 817 ms with it moved aside.

`graph build` after one changed file, measured later the same evening with
the branch's final binary (`c51378ca` before regrouping) while a review ran
on the machine: `iam_backend` 326 / 327 / 325 ms (1 parsed, 398 reused),
`ultra-brain` 371 / 378 / 447 ms (1 parsed, 355 reused); a warm build without
a change took 342–467 ms and 377–381 ms there.

The probe that chose gotreesitter over `wazero` (2026-09-25/26, same
machine, gotreesitter v0.55.0 against `web-tree-sitter` 0.25.10 in node with
`tree-sitter-cpp` 0.23.4 and `tree-sitter-python` 0.25.0): Python parses
clean on 8,556 files, at 0.36–0.68 ms per KB, about twice the C runtime;
C++ reports error nodes in 193 of 397 PrusaSlicer files against 55 for the
C runtime, 4.7 times slower. Details in the design
(`docs/.superpowers/specs/2026-09-26-loomux-code-g5-design.md` §2).

### Reading

1. **`pre-tool-use` moved by half a millisecond, not three.** The warm
   medians average 9.8 ms before and 10.3 ms after; the passes disagree in
   sign (+1.9, +0.6, −1.0 ms). The minima sit 0.5 ms higher. E1's trigger
   is not reached; the patched copy stays off the table.
2. **The start floor carries the init cost.** `--version` rose by 0.9 ms
   warm, and the init sum by about 1.6 ms (median 2.94 → 4.56 ms). That is
   the price E1 accepted: gotreesitter's `init` runs on every start, and no
   import boundary can keep it out of one binary.
3. **`post-tool-use` did not move.** Its second is spent in the edit lanes;
   the blast monitor stays on `go/parser`, and the new combined extractor
   stamp still lets it read the graph.
4. **The binary grew by 12.0 MB**, almost all of it the tree-sitter runtime
   itself; a copy pruned to three grammars saved 0.5 MB in the probe.
5. **The cache pays for itself on Python.** A warm build of `iam_backend`
   takes a third of a `--no-reuse` build, which parses every file as well.
   The cold run was the first start of a freshly built binary on a fresh
   clone; why it took almost three times as long as `--no-reuse` was not
   measured apart. On pure Go the cache saves little — decoding 17 MB of JSON costs nearly
   what `go/parser` does — but `graph check` gets four times faster.

## 2026-09-27 00:47 — Stage 4c-2: search quality on the corpus `v1` and everyday latency through the service

Worktree `.claude/worktrees/4c-planung-bd1521`, branch `feat/bench-search`,
a binary built from the branch; qmd 2.8.3 (`facd35e`) with the three models
of the machine's `index.yml` (embeddinggemma-300M, qmd-query-expansion-1.7B,
Qwen3-Reranker-0.6B). Machine: `windows/amd64`, AMD64 Family 26 Model 68
(as the report names it). Backbone of qmd for both runs:
`QMD_LLAMA_GPU=vulkan`; under the default backbone (CUDA) neither run
completed on this machine (the parity file `stufe-4c-2.md` has the details).

**Method.** Quality: `loomux dev bench search --corpus v1 --profile fast
--out <scratch>`, 00:47–00:57, through the qmd command line in a named index
of its own, 100 documents, 50 questions; against the baseline of
2026-08-21 (`testdata/bench/search/v1/baseline/`). Latency: `loomux dev bench
search --scope all --latency` over a scratch copy of the real question set,
through the qmd service (`port: daemon`), 511 indexed documents, report stamp
`2026-09-26-2304` UTC (01:04 local); one cold and 10 warm runs per
operation, after the quality pass, so the chain is already warm when the
cold run starts. Read document `engineering/craft/_schema.md`, query
`latenz`.

Quality on the corpus, `fast`:

| sort | 2026-09-27 | baseline 2026-08-21 |
|---|---:|---:|
| exakt | 12/13 | 13/13 |
| umschreibung | 11/13 | 11/13 |
| gemischt | 7/10 | 8/10 |
| sprachuebergreifend | 10/14 | 11/14 |
| total | 40/50 | 43/50 |

Latency through the service:

| operation | cold | warm median | warm min | warm max |
|---|---:|---:|---:|---:|
| catalog | 1.0 ms | 1.0 ms | 1.0 ms | 1.5 ms |
| read | 1.0 ms | 1.0 ms | 1.0 ms | 1.5 ms |
| keyword | 13.5 ms | 12.6 ms | 12.0 ms | 14.5 ms |
| fast | 185.7 ms | 185.5 ms | 179.1 ms | 209.7 ms |
| full | 9,293.0 ms | 706.7 ms | 653.0 ms | 730.6 ms |

### Reading

1. **The corpus lost three hits against the baseline.** The misses shared
   with the baseline are `c16`, `c21`, `c27`, `c40`, `c44`, `c49`; new are
   `c07`, `c32`, `c35` (not found) and `c50` (rank 4); `c33` hits now. The
   run does not explain the gap.
2. **The corpus latency is not a number to keep.** The median per question
   is about 10.8 s (hits 10,804 ms, misses 11,132 ms) because the command
   line loads the models on every call; the service does not.
3. **The everyday quality could not be measured cleanly.** The index was
   embedded under CUDA; under Vulkan the query embeddings do not match it,
   and `fast` scored 0/50 with the same page for every query. `keyword`
   scored 8/50, which shows only that loomux maps hits to their area and
   path. The reference scored 26/50 with `fast` in everyday use on
   2026-08-22; that run indexed 276 documents against 511 now, with the
   question set from before `space` moved its wiki, so the two numbers do not
   compare directly. A clean number needs a working CUDA path or an index
   embedded under the backbone that searches it.
4. **`full` pays once.** Its cold run, the first `full` search of the run
   (the quality pass asked `fast`), takes 9.3 s; warm it is 0.7 s, `fast`
   0.19 s, `keyword` 13 ms.

## 2026-09-27 01:14 — Stage 4c-2: `dev bench hooks` and `dev bench repos` with `--out`

Same worktree, branch and binary as the entry above; Go 1.27.0. Report
stamp `2026-09-26-2314` UTC. Both runs wrote both report files.

**Method.** `loomux dev bench hooks <case> -n 10 --out <scratch>` over one
case written for the run: the `pre-tool-use` guard on an `Edit` in this
worktree. `loomux dev bench repos --dir . --warm 3 --out <scratch>` on this
worktree, sample file `cmd/loomux/main.go`; one cold and three warm runs.

| case | cold (1st run) | warm median | warm min | warm max | exit codes |
|---|---:|---:|---:|---:|---|
| hooks: pre-tool-use guard (this worktree) | 14.5 ms | 9.0 ms | 8.5 ms | 9.5 ms | [0] |
| repos: pre-tool-use | 11.5 ms | 10.5 ms | 9.7 ms | 10.5 ms | [0] |
| repos: post-tool-use | 787.2 ms | 791.2 ms | 783.6 ms | 844.4 ms | [0] |
| repos: graph build | 675.5 ms | 679.2 ms | 677.0 ms | 687.6 ms | [0] |
| repos: total | 1474.2 ms | 1489.3 ms | 1472.5 ms | 1531.9 ms | [0] |

### Reading

1. **The guard stays under the 35 ms budget**, cold as warm, in both
   commands.
2. **The renamed commands write the shared envelope**: the `repos` JSON
   carries `schema` 1, `command` `repos` and a payload with `repos` and
   `skipped`.

## 2026-09-27 16:18 — The Flow Runtime: `session-start` With Waiting Runs, and the Guard's Flow Rules

Worktree `.worktrees/flow-a`, branch `feat/flow-runtime`. `before.exe` is
`fa1e83dc`, the branch's merge base at the time, built from `git archive` into
the scratchpad; `after.exe` is the branch head `0e39521d`, before the branch
was rebased onto `a7805df8`. Both built with Go 1.27.0 `windows/amd64`.
Machine: AMD Ryzen 7 9800X3D, Windows 11 Pro.

**Goal.** The flow runtime adds two things to the hook path: `session-start`
reads `.loomux/state/runs/` and announces a run waiting at a gate, and
`pre-tool-use` gains the built-in rules for run files, bundled flow folders
and gate answers. Neither may cost measurable time. ultraloom's flow runtime
(ulflow M1) measured +2.1 ms warm for its session start, within the noise.

**Method.** `after.exe dev bench-hooks <cases> -n 30` (the command was then
spelled with a hyphen), three passes from 16:18:28 to 16:18:40, before and
after alternating, the binary in `argv` swapped per pass. Case file
`testdata/bench/flow-hooks.json`; its stdin payloads lay under `%TEMP%` and
are not committed. Five cases: `pre-tool-use` with an `Edit` of
`internal/hooks/guard.go` (outside `.loomux/`), with an `Edit` of
`.loomux/flows/mine/flow.toml` (the guard reads `[flow]` of this worktree's
`.loomux/config.toml` for it), and with a `Bash` `git status`; `session-start`
against a small git world without runs, and against one whose single run
waits at a gate (marker and the first two journal lines of the `example`
flow's golden journal; `after` announces it, `before` knows no runs). Both
worlds hold one commit and no `.loomux/config.toml`. The real state directory
and registry. "Cold" is the first run of a case, not a cold file cache. The
table gives the median of the three pass medians, warm over 30 runs each, the
smallest minimum of the three passes, and cold as the median of the three
first runs. Every run ended with exit 0. The case file runs unchanged under
today's `loomux dev bench hooks` (probed 2026-09-27 16:52, exit 0).

| case | before cold | after cold | before warm median | after warm median | before warm min | after warm min |
|---|---:|---:|---:|---:|---:|---:|
| pre-tool-use Edit outside `.loomux/` | 16.5 ms | 14.5 ms | 13.8 ms | 12.5 ms | 11.5 ms | 10.5 ms |
| pre-tool-use Edit under `.loomux/flows/mine/` | 14.5 ms | 13.0 ms | 13.5 ms | 12.5 ms | 11.5 ms | 10.9 ms |
| pre-tool-use Bash `git status` | 11.0 ms | 10.0 ms | 10.5 ms | 9.7 ms | 9.0 ms | 8.0 ms |
| session-start, a project without runs | 12.0 ms | 11.5 ms | 11.0 ms | 9.3 ms | 9.0 ms | 8.0 ms |
| session-start, one run waiting at a gate | 12.5 ms | 11.0 ms | 11.0 ms | 10.0 ms | 9.5 ms | 8.1 ms |

| binary | size |
|---|---:|
| before (`fa1e83dc`) | 36,457,472 bytes |
| after (`0e39521d`) | 37,006,336 bytes (+548,864, +1.5 %; the whole branch, not only the rules) |

### Reading

1. **The new rules cost no measurable time.** `after` is about 1 ms ahead of
   `before` in every case, also in those the new rules barely touch (`Bash`
   `git status`, the session start without runs). The lead therefore does not
   come from this change; the build (`before` from `git archive`, without
   `.git` and so without a VCS stamp) or the order in each pass (`before` ran
   first) are the likely causes. It lies within the spread (warm maxima up to
   26 ms).
2. **Reading `[flow]` for an edit under `.loomux/flows/` costs nothing
   visible**: after, that case runs level with the edit outside `.loomux/`.
3. **The waiting run** costs `after` 0.7 ms warm against the project without
   runs, less than the spread of either case.

## 2026-09-27 21:04 — Stage 4d: the start cost of the embedded table, and `convert` over a real inbox

Windows 11 Pro 10.0.26200, Go 1.27.0.

**Start cost.** `GODEBUG=inittrace=1 <binary> --version`, six runs in a row
of each binary, all twelve within 21:04:43 +0200. Base: `origin/master`
`9428f0af`, built in a scratch worktree with `go build -o <scratch>/base.exe
./cmd/loomux`. Change: `feat/convert-fetch` at `c5477e58`, built the same
way. The first run of each follows its build, so the OS cache is warm: there
is no true cold run. The table sums every `init` line of a run.

| run | packages with init | allocations | bytes | clock (sum) |
|---|---:|---:|---:|---:|
| base, first run | 146 | 14,538 | 1,521,336 | 3.48 ms |
| base, warm (5) | 146 | 14,534 to 14,541 | 1,520,696 to 1,521,816 | median 2.48 ms |
| change, first run | 149 | 14,573 | 1,523,096 | 2.98 ms |
| change, warm (5) | 149 | 14,569 to 14,582 | 1,522,456 to 1,524,536 | median 2.98 ms |

Binary size: base 36,687,872 bytes, change 37,185,024 bytes (+497,152; the
gzip Zipf table is 288,299 of it, `NOTICE.md` 48,428, the table's own notice
6,191).

**`convert` over a real inbox.** Measured by the human after the self-use on
the same evening (the time of day was not recorded), in PowerShell with
`1..6 | ForEach-Object { (Measure-Command { bin\loomux.exe convert }).TotalMilliseconds }`,
over the inbox of the area `knowledge`: two PDFs and the one target an
earlier run wrote, nothing new to convert (one PDF is all scans and one
keeps seven scan pages, so each run prints two `skipped:` lines and exits 1).
Model off, Poppler 25.07.0. `convert` is new; there is no base to set
against it.

| command | cold (1st run) | warm median | warm min | warm max |
|---|---:|---:|---:|---:|
| `loomux convert` (2 PDFs, 1 target, nothing new) | 556.8 ms | 437.1 ms | 413.6 ms | 526.1 ms |

Warm runs: 452.0, 526.1, 413.6, 437.1, 434.1 ms.

### Reading

1. **The embedded table costs nothing at start.** Three packages gain an
   `init` — `internal/brain/model` (10 allocations, 360 bytes),
   `internal/brain/convert` (18, 832) and `internal/dev/notices` (4, 144) —
   and `internal/cli` grows from 14 to 16 allocations: about 35 allocations
   and 1.6 KB in all, 0.24 %. The Zipf table (`zipf/de.txt.gz`, unpacked on
   first use behind `sync.OnceValue`) and both notice files are `//go:embed`
   byte slices and strings and allocate nothing at start. The largest init
   stays `gotreesitter/grammars/runtime` (784,832 bytes, 10,180 allocations)
   in both builds.
2. **The clock column is noise.** Windows reports each package init as 0 or
   about 0.5 ms, timer ticks, so both sums move in 0.5 ms steps alike; the
   allocation count is the measure.
3. **A run with nothing new still extracts every PDF.** It runs `pdftotext`
   over both PDFs (46 pages) to learn that nothing changed, since the
   target's text is compared, not the source's time. How much of the
   0.44 s that takes, this run does not separate.

## 2026-09-28 09:25 — Everyday search quality: the qmd backbone, and `fast` across areas

What: `loomux dev bench search` over the everyday question set of
`brain-knowledge` (50 questions), loomux 4.2.0 against the fix on this branch,
qmd 2.8.3, `--repeat 10`. The question set was used from a copy whose 27
`expect` paths were moved from `space/wiki/` to `space/docs/wiki/` (the wiki
moved on 2026-09-07). Every index was embedded again in full (`qmd embed -f
--timeout 0`) under the backbone it was searched with.

| run | Vulkan | CUDA |
|---|---:|---:|
| full embed, 587 documents, 2,986 chunks | 2 min 51 s, nothing pending | 3 min 0 s, nothing pending |
| `keyword`, `--scope all` | 8/50, median 15 ms | 8/50, median 13 ms |
| `fast`, `--scope project/space` | 19/50, median 93 ms | 20/50, median 117 ms |
| `fast`, `--scope all`, loomux 4.2.0 | 0/50 | 0/50 |
| `fast`, `--scope all`, this fix | 31/50 (exakt 9/13, umschreibung 5/13, gemischt 7/10), median 92 ms | — |

### Reading

1. **CUDA ran cleanly this time.** On 2026-09-27 the embed under CUDA ran
   40 min and left 94 % of the documents without a vector, and the daemon
   crashed on its first question. Today, same machine, it ran through in
   3 min. One clean run against one crash; the difference between the two
   backbones (19 against 20, 93 against 117 ms) is within the spread.
2. **`fast` across several areas found nothing, whatever the backbone.** qmd
   fuses one ranked list per named collection and weighs the first list
   double, so each collection's best hit comes back in the order the
   collections were named (scores 1, ½, ⅓ …). A search over no named
   collection ranks the whole index at once; loomux now asks that way and
   keeps the hits of the areas it was asked about.
3. **`keyword` stays as it is.** Asked the same way it scored 7/50 against
   8/50; that is noise, not a gain.
## 2026-09-28 11:58 — The stop gate with the graph lane

Binary built from `feat/g4c-stop-blast` at `2ffa19ac`, before the branch was
rebased onto `849ac5eb`, in a detached scratch
worktree of this repository (711 files, graph built with `loomux graph build`
in 968 ms). Each run: a one-line edit inside `gitenv.Environ` (in-degree 58,
no test changed), then `loomux hook stop --host claude` with a fresh session
id (no base, so it measures against `HEAD`), timed from the shell. "Cold" is
the first hook after the edit, "warm" a second hook on the same tree. Five
edits per case.

"Five kinds" is the new default `stop` (lint, types, test, coverage, graph).
"Four kinds" is the same hook with `.loomux/state/graph` moved aside: without
`wiring.json` no index copy is kept and the graph lane is `not-applicable`, so
the chain is the old four kinds. (A `[verify.profiles] stop` override in the
scratch clone was refused by the write barrier, which guards every
`.loomux/config.toml`.)

| case | cold runs (ms) | cold median | warm runs (ms) | warm median | exit |
|---|---|---:|---|---:|---|
| four kinds (no graph), 12:17 | 110523, 103169, 108144, 154151, 123440 | 110.5 s | 111804, 122729, 100235, 122478, 115371 | 115.4 s | 0 |
| five kinds, 11:58 | 119908, 130246, 93726, 94839, 108384 | 108.4 s | 141636, 137273, 92550, 111495, 109087 | 111.5 s | 2 |

The `graph/go` lane as the hook reported it (always red, `gitenv.go [stale]:
Environ in-degree 58`): cold 1.4, 1.3, 1.5, 1.7, 1.0 s (rebuild after the
edit), warm 0.7, 0.2, 0.2, 0.3, 0.3 s (no drift).

### Reading

1. **The lane costs 1.0–1.7 s cold and 0.2–0.7 s warm**, run in parallel
   with `test` and `coverage`, which take the rest of the ~100 s.
2. **The whole hook does not measurably move.** The spread within one case
   (93–154 s) is far wider than the lane; the five-kind medians are even
   lower than the four-kind ones, which is noise of the test lanes.
3. The red five-kind runs and the green four-kind runs end on different
   paths (exit 2 against a green pass that moves the base); both run the
   whole chain first.

## 2026-09-28 15:53 — The Guard Reads Shell Write Targets as Paths

Worktree `.worktrees/guard-shell`, branch `fix/guard-shell-paths`.
`before.exe` is `24f12d2c`, the branch's merge base with `origin/master`,
built from `git archive` into the scratchpad; `after.exe` is the branch's
code at `45902725`. Both built with Go 1.27.0 `windows/amd64`. Machine: AMD Ryzen 7
9800X3D, Windows 11 Pro.

**Goal.** `pre-tool-use` now reads a shell line by a table of verbs and
wrappers, unfolds braces, matches globs against the disk and, for a removal,
asks what lies below the removed folder; every target then goes through the
path rules a writing tool's target goes through. The verb table, brace and
glob expansion and the ancestor check may not lift `pre-tool-use` above 35 ms
warm.

**Method.** `after.exe dev bench hooks <cases> -n 30 --out <dir>`, three
passes from 15:53:51 to 15:54:09, before and after alternating, each binary
copied onto the binary `argv` names before each pass. Case file
`testdata/bench/guard-shell-hooks.json`:
the five cases of `testdata/bench/flow-hooks.json` against this worktree and
the same two worlds, and a sixth, a `Bash` line with wrappers, braces and
globs:

```sh
cd docs && sudo -u root env X=1 timeout -s KILL 60 cp -r notes/{a,b,c}/*.md build/out/ ; tee -a build/log.txt < in.txt | xargs -n 1 echo && rm -rf build/tmp/* 2>&1 ; git status
```

`after` reads `cd docs` and judges `docs/build/out/`, `docs/build/log.txt`
and `docs/build/tmp/*` (which match nothing on disk and stay as written);
neither binary refuses it. The stdin payloads and the worlds lay in the
session's scratchpad and are not committed; the case file names that path,
so it runs only on this machine, like the one before it. The real state
directory and registry. "Cold" is the first run of a case, not a cold file
cache. The table gives the median of the three pass medians, warm over 30
runs each, the smallest minimum of the three passes, and cold as the median
of the three first runs. Every run ended with exit 0.

| case | before cold | after cold | before warm median | after warm median | before warm min | after warm min |
|---|---:|---:|---:|---:|---:|---:|
| pre-tool-use Edit outside `.loomux/` | 74.0 ms | 83.9 ms | 14.0 ms | 14.0 ms | 11.5 ms | 12.0 ms |
| pre-tool-use Edit under `.loomux/flows/mine/` | 15.5 ms | 15.0 ms | 14.7 ms | 15.0 ms | 12.0 ms | 12.3 ms |
| pre-tool-use Bash `git status` | 10.0 ms | 11.0 ms | 12.0 ms | 11.5 ms | 10.0 ms | 9.5 ms |
| pre-tool-use Bash, the long line above | 11.0 ms | 12.0 ms | 12.3 ms | 12.0 ms | 10.0 ms | 10.0 ms |
| session-start, a project without runs | 21.5 ms | 13.0 ms | 12.2 ms | 11.5 ms | 9.8 ms | 9.5 ms |
| session-start, one run waiting at a gate | 18.5 ms | 14.5 ms | 12.2 ms | 12.1 ms | 10.5 ms | 10.5 ms |

| binary | size |
|---|---:|
| before (`24f12d2c`) | 37,279,744 bytes |
| after (`45902725`) | 37,410,304 bytes (+130,560, +0.35 %) |

**Strict mode** (`[guard] mode = "strict"`) is measured in process, not as a
hook: a world for it needs a `.loomux/config.toml`, which the guard of the
measuring session rightly refuses an agent under any directory.
`BenchmarkCheckTool` (`internal/hooks/guardstrict_bench_test.go`) judges one
long line in a temporary project with a `bin/*` path rule, in both modes:

```sh
sudo -u root env X=1 timeout -s KILL 60 frob --out bin/app.exe src/notes/{a,b,c}/*.md ; tee -a build/log.txt < in.txt | xargs -n 1 echo && rm -rf build/tmp/* 2>&1 ; git status
```

`frob` is no program the guard knows and names the protected
`bin/app.exe`: strict mode refuses the line after resolving every target
through the file system, the default mode lets it pass. `go test
./internal/hooks/ -run '^$' -bench CheckTool -benchtime 200x -count 3`,
`45902725` plus the benchmark:

| `checkTool`, the line above | run 1 | run 2 | run 3 |
|---|---:|---:|---:|
| default mode | 0.76 ms | 0.81 ms | 1.35 ms |
| strict mode | 5.36 ms | 4.28 ms | 4.62 ms |

### Reading

1. **The new reading costs no measurable time.** Every warm median of
   `after` lies within 0.7 ms of `before`, in both directions, and far below
   the 35 ms budget; the long line runs level with `git status`. The warm
   maxima reach 50 ms (before) and 44 ms (after), so a difference below 1 ms
   is noise.
2. **Strict mode costs about 4 ms more per line** than the default mode,
   in process: resolving every target, its brace and glob variants included,
   through the file system. Added to the 12 ms warm of the default hook this
   stays below 35 ms, as an estimate from two measurements, not a hook run.
3. **The cold run of the first case** (74 and 84 ms) is the first start of a
   freshly copied binary in each pass, most likely, not the guard: the second case, which
   runs the same binary right after it, is cold at 15 ms.
4. **A glob costs a directory read only where a target holds one**: the long
   line's globbed target `build/tmp/*` matches nothing and costs nothing
   visible against `git status` (the globs in `cp`'s sources are read, not
   written, and are not expanded at all).

## 2026-09-28 16:09 — The stop gate's no-op path with the graph lane

A detached scratch worktree of this repository at the branch head
`feat/g4c-stop-blast` (15,138 tracked files). `before.exe` is the branch's
merge base `849ac5eb`, `after.exe` the branch with its review fixes; both
built with Go 1.27.0 `windows/amd64` into the scratch directory. Machine:
AMD Ryzen 7 9800X3D, Windows 11 Pro.

**Goal.** The branch loads the config before the content fingerprint and,
with a graph, writes the fingerprint through a copy kept in the git
directory. Neither may make the turn end with nothing new noticeably dearer.

**Method.** `after.exe dev bench hooks -n 20` over a case file with both
binaries on one stdin payload (`{"session_id":"bench","hook_event_name":"Stop"}`)
and one session state whose `base` is `HEAD` and whose `green` is
`HEAD^{tree}`, so both take the no-op path. Two passes without a graph, then
`graph build` in the worktree and two passes with one. Every run exited 0.

| case | cold (1st run) | warm median | warm min | warm max |
|---|---:|---:|---:|---:|
| before, no graph, pass 1 | 252.6 ms | 245.2 ms | 221.0 ms | 295.4 ms |
| after, no graph, pass 1 | 265.2 ms | 266.6 ms | 238.5 ms | 464.0 ms |
| before, no graph, pass 2 | 320.9 ms | 238.5 ms | 211.6 ms | 633.3 ms |
| after, no graph, pass 2 | 225.5 ms | 238.6 ms | 218.0 ms | 291.9 ms |
| before, graph, pass 1 | 229.9 ms | 235.7 ms | 215.5 ms | 298.0 ms |
| after, graph, pass 1 | 241.6 ms | 256.1 ms | 238.3 ms | 407.6 ms |
| before, graph, pass 2 | 237.6 ms | 236.9 ms | 216.3 ms | 276.2 ms |
| after, graph, pass 2 | 250.9 ms | 256.4 ms | 239.8 ms | 274.8 ms |

### Reading

1. **Without a graph the no-op path does not move.** The two passes disagree
   (+21 ms, then +0.1 ms); the config that now loads first costs less than
   the spread.
2. **With a graph it costs about 20 ms more**, in both passes: the copy is
   written into the git directory and kept, and `HEAD`'s tree is asked once
   more for the lane's probe. `before` has no graph kind in `stop` and takes
   the plain path.
3. **The repository doubled since 2026-09-20** (7,341 to 15,138 files); the
   no-op path grew from 169.5 ms to about 237 ms with it, most of it the
   fingerprint over the larger index.

## 2026-09-28 17:29 — The Guard Walks a `find` With a Name Filter

A `find` that deletes under a name filter is now judged by what it takes on
disk: the guard walks the start paths once per call (up to 50000 entries)
and judges each entry that matches. Measured as a hook in this repository's
worktree (21392 entries, 19163 of them under `testdata/`), six runs in a row
each, the first counted cold, with the binary built from `4f385ebd` (before)
and from `dbd96590` (after):

```sh
find . -type d -name __pycache__ -exec rm -rf {} +
```

| `hook pre-tool-use` | cold | warm, runs 2–6 | verdict |
|---|---:|---|---|
| before, the line above | 99 ms | 35, 35, 35, 42, 36 ms | refused (the start path `.` taken for the whole project) |
| after, the line above | 348 ms | 351, 335, 351, 343, 340 ms | allowed |
| before, `ls src` | 35 ms | 34, 35, 34, 36, 56 ms | allowed |
| after, `ls src` | 35 ms | 42, 50, 41, 58, 36 ms | allowed |

### Reading

1. **The walk costs about 300 ms in a tree of 21000 entries**, far above the
   35 ms budget, and only for a `find` or `Get-ChildItem` that deletes under a
   name filter; every other line pays nothing (`ls src` stays level). The
   line was refused before, so the price buys a line that passes.
2. **The limit decides over a large tree.** With 20000 entries as the limit,
   a first build took this worktree for too large and refused the line in
   330 ms; 50000 lets it through. A tree past the limit is refused as
   before, after walking up to the limit.

## 2026-09-28 20:20 — `reconcile` With the Real Local Model Against Without

**Goal.** The time per proposal of the local model (the reference's
criterion: under two seconds, 563 to 1,314 ms there), as step 2 of the 4c-1
plan asks: `reconcile` with the model against without, cold and warm.

**Method.** `bin/loomux.exe` from `origin/master` (`85f06f66`), Ollama 0.34.0
on `127.0.0.1:11434` with CUDA (22.8 GiB free), model
`hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q4_K_XL` (the default,
`temperature` 0). Each run on a fresh copy of the world
`testdata/cases/4c1/reconcile/proposal-kept` (one `local_only` area, one
changed source, one page citing it) in the scratchpad, with
`LOOMUX_STATE_DIR`, `LOOMUX_LEGACY_BRAIN_DIR` and `XDG_CONFIG_HOME` inside the
world and `[model]` there set to `enabled = true` or `false`. Wall clock
around the whole process.

| Case | Runs | Time | Result |
|---|---|---:|---|
| Model off, cold | 1 | 239 ms | manual case |
| Model off, warm | 5 | median 209 ms (184–217) | manual case |
| Model on, cold (model not loaded) | 1 | 30,348 ms | manual case: the 30 s limit ran out |
| Model on, first question after the load | 1 | 28,307 ms | proposal, `vorschlag-v4` |
| Model on, warm | 4 | median 793 ms (584–1,036) | proposal, `vorschlag-v4` |

Ollama itself reported 350 to 798 ms for the four warm questions.

### Key Findings
1. **Warm, the criterion holds.** Warm with the model minus warm without
   comes to about 580 ms per proposal, within the reference's band.
2. **Cold, the first question sits at the 30 s limit.** On the first run
   Ollama was still loading the model when loomux gave up after 30 s; Ollama
   then dropped the half-loaded model (`client connection closed before
   llama-server finished loading, aborting load`), and the case opened
   without a proposal. Preloading the model by hand did not help: loomux asks
   with a `num_ctx` of its own (`internal/brain/model/client.go`), Ollama loaded the model again for them (3.4 s),
   and the first prompt evaluation took 20.4 s for 1,289 tokens, the ones
   after it under a second. The limit is a decision of the spec ("30 s in all
   for a question"); whether the first question after a load gets more time
   is the user's call.

## 2026-09-29 08:56 — `reconcile` With the Local Model Warmed Up

**Goal.** Check whether warming the model before a client's first question
solves the 30 s limit found on 2026-09-28 20:20.

**Method.** As on 2026-09-28: a fresh copy of the world
`testdata/cases/4c1/reconcile/proposal-kept` per run, `[model] enabled =
true`, the default model, Ollama 0.34.0 with CUDA, freshly started with no
model loaded (`/api/ps` empty). Binary from the branch with
`fix(model): load and warm the local model before the first question`.
Times per request from Ollama's log.

| Case | Time | Result |
|---|---:|---|
| cold, whole run | 17,297 ms | proposal, `vorschlag-v4` |
| cold, warm-up | 14,643 ms | of it 12.6 s starting the llama-server; prompt evaluation 0.37 s for 13 tokens |
| cold, the question | 2,298 ms | prompt evaluation 0.55 s for 1,289 tokens |
| warm, whole run (5) | 1,867–3,053 ms | a proposal in every run |
| warm, warm-up | 104–211 ms | |
| warm, the question | 1,230–2,319 ms | |

### Key Findings
1. **The limit holds again.** The first prompt evaluation, 20.4 s
   yesterday, takes 0.55 s after the warm-up; cold, the proposal comes after
   17 s instead of not at all.
2. **Warm, the warm-up costs 0.1–0.2 s per client.**
3. **The questions themselves were slower than yesterday** (54 instead of
   107 tokens per second generating): a foreign process kept the GPU at
   100 % load and 20 GB of memory during the run, and after Ollama was
   stopped too. The time per proposal here is therefore no comparison with
   the reference's criterion (under 2 s); the run of 2026-09-28 20:20
   recorded that.

## 2026-09-30 08:10 — Two Projects Switched Over to loomux, Before Against After

**Goal.** What the hooks and the knowledge commands of a project cost before
and after its switch-over from the old tools to loomux: everything old,
everything new and everything dropped, for the two pilot projects of the
switch-over.

**Method.** Two projects of the user, anonymised. Example project 1 is a
Python project that had no `.claude/settings.json` and therefore no old
hooks; example project 2 is a Godot project that carried the hooks of both
old tools. Per project and side, `loomux dev bench cases` built the cases
from the project's `.claude/settings.json` and a file of extra cases for the
commands beside the hooks, `loomux dev bench hooks -n 5` measured them (one
cold run, then five warm ones; earlier entries use 20, five is the user's
choice for the switch-over), and `loomux dev bench compare` paired the two
reports by case name. The tables are its output: cold, the mean and the
median of the warm runs, a factor above 1 where the new side is faster, and
the exit codes a case ended with.

The old side is the old tools as installed, measured on 2026-09-29 about
21:00 before either project was touched. For example project 2 the three
session hooks (`SubagentStart`, `SubagentStop`, `Stop`) come from a second
run half an hour later: the first payloads carried no `session_id`, and
those hooks refused them before doing any work. The new side is the
installed loomux the project's hooks call after the switch-over: 5.3.0
(beta) for example project 1, measured on 2026-09-29 about 22:40, and 5.3.1
(beta) for example project 2, measured on 2026-09-30 about 08:09. On the
second day the qmd service was not running, which it had been throughout
the baseline; a `loomux serve --foreground` and the `loomux mcp` of five
open sessions ran instead.

**Example project 1, compared**

| Case | cold old | cold new | × | warm mean old | warm mean new | × | median old | median new | exit old | exit new |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| status | 49.5 ms | 6,308.6 ms | 0.01× | 67.6 ms | 5,196.1 ms | 0.01× | 43.0 ms | 5,238.4 ms | [0] | [0] |
| search (fast) | 206.4 ms | 2,739.9 ms | 0.08× | 250.7 ms | 145.6 ms | 1.72× | 236.0 ms | 121.7 ms | [0] | [0] |
| wiki lint | 47.0 ms | 156.4 ms | 0.30× | 49.5 ms | 162.4 ms | 0.30× | 48.5 ms | 158.3 ms | [0] | [0] |

**Example project 1, new**

| Case | cold | warm mean | median |
|---|---:|---:|---:|
| SessionStart | 213.5 ms | 13.3 ms | 13.0 ms |
| PreToolUse (Edit on README.md) | 27.5 ms | 45.2 ms | 19.5 ms |
| PostToolUse (Edit on README.md) | 22.0 ms | 45.6 ms | 21.8 ms |
| SubagentStart | 292.5 ms | 678.4 ms | 568.9 ms |
| SubagentStop | 292.5 ms | 13.0 ms | 13.0 ms |
| Stop | 806.3 ms | 1,042.5 ms | 827.1 ms |
| graph stats | 123.3 ms | 218.0 ms | 134.0 ms |
| graph check | 28.0 ms | 24.0 ms | 23.5 ms |

**Example project 2, compared**

| Case | cold old | cold new | × | warm mean old | warm mean new | × | median old | median new | exit old | exit new |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| SessionStart | 683.9 ms | 553.7 ms | 1.24× | 552.6 ms | 253.0 ms | 2.18× | 456.8 ms | 251.4 ms | [0] | [0 0] |
| PreToolUse (Edit on AGENTS.md) | 54.0 ms | 12.4 ms | 4.35× | 48.8 ms | 9.8 ms | 4.98× | 48.4 ms | 9.6 ms | [0 0] | [0] |
| PostToolUse (Edit on AGENTS.md) | 52.4 ms | 18.8 ms | 2.79× | 67.5 ms | 13.7 ms | 4.92× | 52.2 ms | 13.2 ms | [0] | [0] |
| SubagentStart | 2,093.3 ms | 1,735.5 ms | 1.21× | 1,837.0 ms | 1,127.5 ms | 1.63× | 1,765.4 ms | 1,112.6 ms | [0] | [0] |
| SubagentStop | 1,935.8 ms | 1,088.9 ms | 1.78× | 1,866.0 ms | 8.1 ms | 230.60× | 1,820.6 ms | 8.0 ms | [0] | [0] |
| Stop | 618.9 ms | 1,099.8 ms | 0.56× | 664.2 ms | 336.7 ms | 1.97× | 608.8 ms | 408.3 ms | [1 1] | [2] |
| commit-msg | 603.2 ms | 355.8 ms | 1.70× | 587.6 ms | 261.8 ms | 2.24× | 488.1 ms | 262.1 ms | [0] | [0] |
| status | 201.6 ms | 2,336.0 ms | 0.09× | 206.7 ms | 2,143.4 ms | 0.10× | 208.1 ms | 2,113.9 ms | [0] | [0] |
| search (fast) | 267.7 ms | 11,506.8 ms | 0.02× | 232.4 ms | 76.1 ms | 3.05× | 197.6 ms | 76.1 ms | [0] | [0] |
| wiki lint | 45.5 ms | 173.6 ms | 0.26× | 47.4 ms | 169.8 ms | 0.28× | 47.8 ms | 169.0 ms | [0] | [1] |

**Example project 2, new**

| Case | cold | warm mean | median |
|---|---:|---:|---:|
| graph stats | 10.0 ms | 7.5 ms | 7.5 ms |
| graph check | 84.5 ms | 37.0 ms | 36.5 ms |

**Example project 2, dropped**

| Case | cold | warm mean | median |
|---|---:|---:|---:|
| python entry (ultraloom --help) | 247.1 ms | 119.1 ms | 116.9 ms |

### Key Findings
1. **Two measuring days are not comparable without a control.** `commit-msg`
   of example project 2 is the same hook of the project on both sides and
   still ran faster on the second day (median 488.1 → 262.1 ms, 2.24× on the
   warm mean). A factor of up to about 2 therefore proves nothing here;
   `SessionStart` (2.18×) lies inside that noise, and two commands run side
   by side there now instead of one.
2. **The hooks at every edit are faster beyond that noise.** In example
   project 2 `PreToolUse` went from 48.4 to 9.6 ms and `PostToolUse` from
   52.2 to 13.2 ms (warm median): one loomux process in place of the two old
   guards.
3. **Regression: `status`.** Warm median 43.0 → 5,238.4 ms in example
   project 1 and 208.1 → 2,113.9 ms in example project 2. The new command
   asks qmd, the old one did not. Open; nothing on this branch changes it.
4. **Regression: `wiki lint`.** Warm median 48.5 → 158.3 ms and 47.8 →
   169.0 ms (0.30× and 0.28× on the warm mean). In example project 2 it also
   ends with exit 1 instead of 0: the new check reports findings the old one
   passed in silence.
5. **`search (fast)` is faster warm and slower cold.** Warm median 236.0 →
   121.7 ms and 197.6 → 76.1 ms; the cold run takes 2,739.9 ms in example
   project 1, where the first call starts the search service, and
   11,506.8 ms in example project 2, measured with no qmd service running.
6. **The session hooks compare two ways of failing.** `Stop` of example
   project 2 ended with `[1 1]` before and `[2]` after; neither side ran a
   check chain to its end, and the new stop gate gives up once after three
   blocks in a row, so its five warm runs are mixed. `SubagentStop` fell from
   1,820.6 to 8.0 ms warm, but cold it takes 1,088.9 ms: only the first run
   does visible work, so the warm value is no measure of the hook. `Stop` of
   example project 1, new there, ends with exit 2 as well: it runs the
   project's gate, which falls on lint findings that stood before the
   switch-over.
7. **Not measured: the pre-commit gate of the projects**, on neither side. A
   probe ran into the time limit of a project's test suite, and other
   projects' gates need tools that were not at hand (Docker, Godot); the gate
   is therefore left out before and after.
8. **Example project 1 compares only the knowledge commands.** It had no old
   hooks, so all six hooks appear as new; what a switch-over does to old
   hooks shows in example project 2 alone.

## 2026-10-01 14:45 — The Guard After Reading Strings, Folders, Variables and Patches

**Goal.** Check what reading the lines a shell runs from a string or a
pipe, the variables a line sets, the folder a copy lands in, patch files
and 8.3 names costs the guard per call.

**Method.** `go test -bench BenchmarkCheckTool -count=10 ./internal/hooks`
(`internal/hooks/guardstrict_bench_test.go`: a long line with wrappers,
braces and globs, in a project with a manifest), Go 1.27.0 on Windows 11,
the same machine back to back: baseline master at `e6c21eed`, change the
branch `fix/guard-wrapper-holes` at `ac9a4a73`. Warm, in-process; the start
of the binary is not part of it. Median, minimum and maximum of ten runs.

| Mode | Baseline | Change |
|---|---:|---:|
| default | 483 µs (453–530) | 681 µs (549–731) |
| strict | 4.68 ms (3.87–5.17) | 4.59 ms (4.44–4.93) |

### Key Findings
1. **The default mode costs 0.2 ms more per call** (+41 %): every line is
   read once more for the lines it runs from strings and with the variables
   it sets, and every target gets its lexical spellings.
2. **Strict mode does not change measurably**: the file system resolution
   dominates it, and the new readings fall within its spread.
3. **The protected elements are worked out once per call**: an earlier
   draft did it per target, and the hooks test suite took 135 s instead of
   80 s (not measured with the benchmark).

## 2026-10-02 12:35 — Five More Targets Switched Over to loomux, Before Against After

**Goal.** As in the entry of 2026-09-30: what the hooks and the knowledge
commands cost before and after the switch-over from the old tools to loomux,
now for the wave after the two pilots. The numbers continue those of the entry
of 2026-09-30 (example projects 1 and 2 are the pilots).

**Method.** Five targets of the user, anonymised, in the order of the wave:
example projects 3, 4 and 5 are code projects that carried the hooks of both
old tools, switched over without a knowledge bundle of their own; example
project 6 is a wiki-only repository without hooks, switched over as a wiki
area only; example project 7 is a knowledge vault that had no hooks before.
Per project and side, `loomux dev bench cases` built the cases from the
`.claude/settings.json` and a file of extra cases (by hand for example
project 6, which has no hook), `loomux dev bench hooks -n 5` measured them
(one cold run, then five warm runs; reported are the mean and the median of
the warm runs), and `loomux dev bench compare` paired the reports by case
name. A factor above 1 means the new side is faster.

The old side is the baseline of 2026-09-29 around 21:00, with the qmd service
running. For example projects 3, 4 and 5 the session hooks (`SubagentStart`,
`SubagentStop`, `Stop`, where present) come from the re-measure without load
three quarters of an hour later, as for pilot 2. The new side is the
installed loomux 6.1.0 (beta), measured on 2026-10-02 between 12:35 and
12:40, serially, under Windows 11 Pro on the same machine. The qmd service
was running here too; beside it ran a `loomux serve --foreground`, the
`loomux mcp` of ten open sessions and an unrelated language-model server
holding about 45 GB of memory (CPU load 24 % before the first run). As on
2026-09-30, the pre-commit gate is measured on neither side.

**Example project 3, compared**

| Case | cold old | cold new | × | warm mean old | warm mean new | × | median old | median new | exit old | exit new |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| PreToolUse (Edit on README.md) | 53.5 ms | 16.8 ms | 3.18× | 42.1 ms | 13.3 ms | 3.16× | 41.5 ms | 13.0 ms | [0] | [2] |
| PostToolUse (Edit on README.md) | 80.0 ms | 33.0 ms | 2.42× | 79.6 ms | 29.2 ms | 2.72× | 64.5 ms | 30.0 ms | [0] | [0] |
| Stop | 216.7 ms | 201.2 ms | 1.08× | 273.0 ms | 164.8 ms | 1.66× | 260.8 ms | 156.2 ms | [1] | [0] |
| commit-msg | 569.4 ms | 28.5 ms | 19.95× | 103.0 ms | 24.5 ms | 4.21× | 98.9 ms | 24.1 ms | [0] | [0] |
| status | 90.1 ms | 5,293.5 ms | 0.02× | 109.0 ms | 3,109.2 ms | 0.04× | 105.1 ms | 3,191.8 ms | [0] | [0] |
| search (fast) | 226.2 ms | 5,182.6 ms | 0.04× | 236.1 ms | 97.1 ms | 2.43× | 208.0 ms | 97.6 ms | [0] | [0] |

**Example project 3, new**

| Case | cold | warm mean | median |
|---|---:|---:|---:|
| SessionStart | 147.2 ms | 103.0 ms | 99.7 ms |
| SubagentStart | 1,234.0 ms | 1,168.1 ms | 1,178.1 ms |
| SubagentStop | 1,121.7 ms | 9.7 ms | 9.5 ms |
| graph stats | 34.5 ms | 33.5 ms | 34.5 ms |
| graph check | 1,620.3 ms | 117.2 ms | 113.5 ms |

**Example project 4, compared**

| Case | cold old | cold new | × | warm mean old | warm mean new | × | median old | median new | exit old | exit new |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| SessionStart | 558.0 ms | 135.6 ms | 4.12× | 411.0 ms | 82.3 ms | 4.99× | 221.4 ms | 81.0 ms | [0] | [0] |
| PreToolUse (Edit on AGENTS.md) | 136.1 ms | 16.9 ms | 8.04× | 41.4 ms | 13.3 ms | 3.11× | 41.0 ms | 13.6 ms | [0] | [2] |
| PostToolUse (Edit on AGENTS.md) | 58.0 ms | 19.0 ms | 3.05× | 47.1 ms | 14.9 ms | 3.16× | 46.9 ms | 15.0 ms | [0] | [0] |
| SubagentStart | 1,705.2 ms | 1,378.0 ms | 1.24× | 2,033.0 ms | 1,101.4 ms | 1.85× | 2,135.1 ms | 1,098.1 ms | [0] | [0] |
| SubagentStop | 1,989.5 ms | 1,036.6 ms | 1.92× | 1,909.6 ms | 9.6 ms | 198.48× | 1,898.3 ms | 9.6 ms | [0] | [0] |
| Stop | 346.9 ms | 27,559.8 ms | 0.01× | 239.8 ms | 182.0 ms | 1.32× | 232.8 ms | 178.7 ms | [1] | [0] |
| commit-msg | 105.9 ms | 39.0 ms | 2.72× | 120.6 ms | 30.2 ms | 3.99× | 103.7 ms | 29.6 ms | [0] | [0] |
| status | 119.9 ms | 3,173.5 ms | 0.04× | 118.1 ms | 3,177.2 ms | 0.04× | 101.0 ms | 3,135.5 ms | [0] | [0] |
| search (fast) | 241.3 ms | 168.0 ms | 1.44× | 253.3 ms | 67.8 ms | 3.74× | 249.0 ms | 67.6 ms | [0] | [0] |

**Example project 4, new**

| Case | cold | warm mean | median |
|---|---:|---:|---:|
| graph stats | 10.6 ms | 8.0 ms | 8.0 ms |
| graph check | 15.0 ms | 13.6 ms | 13.5 ms |

**Example project 4, dropped**

| Case | cold | warm mean | median |
|---|---:|---:|---:|
| python entry (ultraloom --help) | 391.3 ms | 230.3 ms | 223.4 ms |

**Example project 5, compared**

| Case | cold old | cold new | × | warm mean old | warm mean new | × | median old | median new | exit old | exit new |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| SessionStart | 433.3 ms | 113.0 ms | 3.83× | 160.6 ms | 75.0 ms | 2.14× | 147.6 ms | 75.7 ms | [0] | [0] |
| PreToolUse (Edit on AGENTS.md) | 46.5 ms | 13.5 ms | 3.44× | 65.6 ms | 12.1 ms | 5.42× | 43.0 ms | 12.0 ms | [0] | [2] |
| PostToolUse (Edit on AGENTS.md) | 49.5 ms | 16.5 ms | 3.00× | 68.3 ms | 12.2 ms | 5.58× | 45.5 ms | 12.0 ms | [0] | [0] |
| SubagentStart | 2,007.9 ms | 1,218.9 ms | 1.65× | 1,829.0 ms | 1,167.3 ms | 1.57× | 1,844.9 ms | 1,182.1 ms | [0] | [0] |
| SubagentStop | 1,794.4 ms | 1,072.1 ms | 1.67× | 1,840.9 ms | 8.3 ms | 222.76× | 1,745.2 ms | 8.5 ms | [0] | [0] |
| commit-msg | 103.9 ms | 31.6 ms | 3.29× | 150.8 ms | 26.3 ms | 5.74× | 101.5 ms | 25.7 ms | [0] | [0] |
| status | 225.4 ms | 3,602.2 ms | 0.06× | 120.6 ms | 3,266.2 ms | 0.04× | 108.0 ms | 3,307.8 ms | [0] | [0] |
| search (fast) | 259.6 ms | 92.8 ms | 2.80× | 251.4 ms | 73.9 ms | 3.40× | 206.6 ms | 70.7 ms | [0] | [0] |

**Example project 5, new**

| Case | cold | warm mean | median |
|---|---:|---:|---:|
| Stop | 143.8 ms | 160.0 ms | 164.6 ms |
| graph stats | 11.6 ms | 8.8 ms | 9.0 ms |
| graph check | 103.4 ms | 13.8 ms | 13.5 ms |

**Example project 5, dropped**

| Case | cold | warm mean | median |
|---|---:|---:|---:|
| python entry (ultraloom --help) | 255.8 ms | 151.6 ms | 135.5 ms |

**Example project 6, compared**

| Case | cold old | cold new | × | warm mean old | warm mean new | × | median old | median new | exit old | exit new |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| status | 51.4 ms | 3,481.7 ms | 0.01× | 68.3 ms | 3,206.6 ms | 0.02× | 42.5 ms | 3,119.7 ms | [0] | [0] |
| search (fast) | 233.7 ms | 70.8 ms | 3.30× | 198.2 ms | 70.0 ms | 2.83× | 196.7 ms | 68.1 ms | [0] | [0] |
| wiki lint | 43.1 ms | 675.3 ms | 0.06× | 59.8 ms | 81.5 ms | 0.73× | 47.5 ms | 81.1 ms | [0] | [1] |

**Example project 7, compared**

| Case | cold old | cold new | × | warm mean old | warm mean new | × | median old | median new | exit old | exit new |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| status | 71.4 ms | 3,514.4 ms | 0.02× | 43.8 ms | 3,133.6 ms | 0.01× | 43.0 ms | 3,207.6 ms | [0] | [0] |
| search (fast) | 213.3 ms | 74.5 ms | 2.86× | 204.8 ms | 81.6 ms | 2.51× | 202.6 ms | 82.4 ms | [0] | [0] |
| wiki lint | 137.1 ms | 41.4 ms | 3.31× | 46.2 ms | 40.9 ms | 1.13× | 47.0 ms | 41.4 ms | [0] | [1] |

**Example project 7, new**

| Case | cold | warm mean | median |
|---|---:|---:|---:|
| SessionStart | 43.0 ms | 9.0 ms | 9.0 ms |
| PreToolUse (Edit on AGENTS.md) | 13.5 ms | 12.7 ms | 12.4 ms |
| PostToolUse (Edit on AGENTS.md) | 22.7 ms | 20.5 ms | 20.4 ms |
| SubagentStart | 1,219.5 ms | 1,170.2 ms | 1,163.6 ms |
| SubagentStop | 1,131.2 ms | 10.8 ms | 11.0 ms |
| Stop | 222.9 ms | 188.4 ms | 189.0 ms |
| commit-msg | 30.0 ms | 26.2 ms | 25.9 ms |

### Key Findings
1. **The hooks on every edit are three to five times faster.** Warm median
   `PreToolUse` 41.5 → 13.0 ms, 41.0 → 13.6 ms and 43.0 → 12.0 ms,
   `PostToolUse` 64.5 → 30.0 ms, 46.9 → 15.0 ms and 45.5 → 12.0 ms (example
   projects 3, 4, 5): one loomux process instead of the two old guards.
2. **But `PreToolUse` ends with exit 2 instead of 0 in example projects 3, 4
   and 5:** the guard refuses the edit, because a project without a knowledge
   bundle of its own gets no area and so lies in no writable tree. The faster
   time is that of a refusal; this is a finding of the switch-over, not a
   measuring error. In example project 7 the same refusal hits a file beside
   the writable folders and is expected.
3. **`SessionStart` 221.4 → 81.0 ms and 147.6 → 75.7 ms** (example projects
   4 and 5), above the noise of about a factor of 2: the old Python is gone.
   The Python entry point itself (median 223.4 and 135.5 ms) is dropped.
4. **`SubagentStop` warm 1,898.3 → 9.6 ms and 1,745.2 → 8.5 ms, but cold
   1,036.6 and 1,072.1 ms**, as for pilot 2: only the first run does visible
   work. `SubagentStart` now takes about 1.1–1.2 s and is 1.6 to 1.9 times
   faster in example projects 4 and 5.
5. **`commit-msg` 98.9 → 24.1, 103.7 → 29.6 and 101.5 → 25.7 ms**: the hook is
   now loomux's instead of a Python script. Since it is not the same hook,
   there is no control case this time, and three days lie between the sides;
   factors up to about 2 prove nothing on their own.
6. **Regression: `status`** in all five targets, warm median about 100 ms
   (vault and wiki: 43 ms) against 3.1–3.3 s, cold up to 5.3 s: the new
   command queries qmd. Still open, as on 2026-09-30.
7. **`search (fast)` is 2.4 to 3.7 times faster warm** (about 200–250 ms
   against 68–98 ms); cold 5,182.6 ms in example project 3, as the first
   search call of the measurement.
8. **`wiki lint` ends with exit 1 instead of 0 in example projects 6 and 7**:
   the new check reports house rules the old one let pass silently. In example
   project 6 it is slower (47.5 → 81.1 ms), in example project 7 a little
   faster (47.0 → 41.4 ms).
9. **`Stop`**: in example projects 3 and 4 exit 1 before, exit 0 after in
   156–179 ms warm (median); the cold run of example project 4 ran lint and
   types over the project (27.6 s), the warm runs met an unchanged base. In
   example project 7 the new stop hook ends with exit 1, because its profile
   holds check kinds that have nothing to check in a vault without code.
10. **New:** `graph stats` and `graph check` in the code projects (warm
    8–118 ms), in example project 3 also the session hooks, which did not run
    before, and in example project 7 all six hooks and `commit-msg`.
    **Dropped:** the Python entry point of the old tools.
11. **Not measured:** two further targets of the original plan; they are not
    switched over.
