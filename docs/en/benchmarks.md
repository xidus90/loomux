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
