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
   of the specification therefore holds, and the floor of ~23 ms is almost
   entirely one dependency's `init` — the same 21 ms appear in the old
   `brain.exe`.
4. **The barrier's own cost, decomposed.** `go test ./internal/brain/guard/
   -run '^$' -bench BenchmarkDecideAgainstTheRealRegistry -benchtime 50x
   -cpuprofile …` against a copy of the real state directory
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
