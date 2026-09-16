# Benchmarks

Chronological performance measurements for loomux, cold and warm.

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
`%LOCALAPPDATA%\brain`. "Before" was run in Task 1 on `219ccb1`, "after" at
`222465b`, both on the same state copy. Machine: AMD Ryzen 7 9800X3D, Go
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
