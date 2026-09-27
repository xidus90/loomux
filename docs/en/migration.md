# Migration plan

This file is kept current with every change: a pull request that starts,
finishes, adds or drops migration work updates it here and in
[`docs/de/migration.md`](../de/migration.md) — the status of the stage, what is
left, its dependencies and its priority, and the status of each capability it
touches. A decision (a new stage, a removal, a
changed order) goes into the [fusion spec](../.superpowers/specs/2026-09-14-loomux-fusion-design.md) first,
and this plan follows it.

## Stages

Loomux is executing a staged fusion plan. A second track — the code graph —
runs **alongside** it rather than after it, because none of its stages waits on
a fusion stage (G5a added the one dependency so far, the pure-Go tree-sitter
runtime `gotreesitter`). *Priority* orders the open rows by three rules in turn:
what its dependencies already allow, what loomux uses on itself, then size. The
[fusion spec](../.superpowers/specs/2026-09-14-loomux-fusion-design.md) sets it under "Reihenfolge der offenen Stufen".

This plan holds migration only. Open work the fusion spec counts as a
follow-up project or leaves optional — the open code-graph stages G4c and G5b to
G5d, Flow, the Web OS and the Claude Mods adapter — is on the
[roadmap](../../README.md#roadmap), ranked by the same priorities.

| Stage | Status | Origin | What it delivered | Depends on | Priority |
|---|---|---|---|---|---|
| **1a** | ✅ | ultraloom + ultra-brain | The pilot: repo scaffolding, gates, the unified guard, the post-edit lanes. loomux uses itself | — | — |
| **1b-1** | ✅ | ultra-brain | The brain read commands — `search`, `status`, `catalog`, `read`, `neighbors` — at parity with the Python reference | — | — |
| **1b-2** | ✅ | ultra-brain | `serve` with MCP over Streamable HTTP and the stdio bridge, held to the reference by a recorded case corpus. Upkeep is Stage 3c | — | — |
| **1b-3** | ✅ | ultraloom + ultra-brain | The wiki and the documentation moved in | — | — |
| **2a** | ✅ | ultraloom | The check chain: `[verify]` with presets per stack, `loomux check <profile>`, `check gocover`, post-edit on `[verify]`, process trees killed whole. loomux gates its own commits with `check precommit` | — | — |
| **2b** | ✅ | ultraloom | commit-msg with `--language`, `--calibrate` and `[commit]` | — | — |
| **2c** | ✅ | ultraloom | The stop gate (`loomux hook stop`, profile `stop`, a content fingerprint, 3 blocks in a row, the marker `.loomux/no-verify`) and `subagent-start`/`subagent-stop`, whose findings the stop gate delivers; the wiki bundle as the lane `lint/wiki` in `loomux check` and the gate. Antigravity measured and host adapters implemented for Claude Code and Antigravity. This repository runs the gate through its tracked `.claude/settings.json` | 2a ✅ | — |
| **3a** | ✅ | ultra-brain | Detect: `loomux reindex` and `loomux embed` (moved from `ultra-brain/pkg/index`), `loomux reconcile` with the read and drop side of the event log, `loomux area add`, the catch-up pass before `reindex`; state is written to `LOOMUX_STATE_DIR`, the legacy directory only read. 28 cases against the Python reference, the four git cases through `gitworld` from 2c. Since 2026-09-22 this machine runs `reindex` and `embed` over the real registry; `[index]` in `.loomux/config.toml` limits `project/loomux` to `docs/wiki` and `docs/.superpowers` (requirement S3 in `parity/stufe-3a.md`) | 1b-1 ✅ | — |
| **3b** | ✅ | ultra-brain | Decide: `loomux cases`, `loomux case`, `loomux approve`; applying a proposal (rewritten from `apply.py`), evidence binding (`evidence`) and the write side of `vcs`, which commits through an index of its own onto the vault's current ref; after a written approval, a catch-up pass and an index run. 24 cases against the Python reference, 19 without a difference after normalization (stderr not compared), 5 with signed-off differences (`parity/stufe-3b.md`). Self-use ran on 2026-09-23 against the real registry: `cases` and `case` identical to the Python reference, `approve` only with `--defer` by the user's decision (`parity/stufe-3b.md`, "Selbstnutzung") | 3a ✅ (the cases) | — |
| **3c** | ✅ | ultra-brain | Maintain: `loomux brain check file\|bundle\|all` (moved from `pkg/check/{okf,house,run}`, the Go binary as the reference), `loomux lint --scope all\|<scope>` with the twelve rules of `lint.py` as a rule set of its own beside the Go form, which `lint <file>`, `wiki-gate` and the lane keep, `loomux wiki init\|types\|retype`, the daily catch-up in `serve` (`reconcile` only, a gate before the first answer, note lines on the answers). `brain check code` is dropped. 35 cases against the reference, 34 without a difference, one signed off (`check code`, `parity/stufe-3c.md`). The reading commands ran against the real registry on 2026-09-23 equal to the reference; the catch-up ran on 2026-09-24 against a copy of the state and the vault (gate, note line and a fresh stamp as specified), `wiki retype` and `wiki init` wait there for the human | 3a ✅ (upkeep calls `reconcile`), 1b-2 ✅ (upkeep runs in `serve`) | — |
| **4a-1** | 🚧 built; human steps open (MCP working-directory measurement, three-terminal check, human config set) | new | Schema and `config`: `internal/config/schema` names every key the readers of `.loomux/config.toml` accept and checks a new text through those same readers; `internal/config/edit` changes one line and keeps every comment, and refuses what it cannot place without guessing; `internal/tui`, a full-screen list and input on `x/term`. `loomux config list\|get\|set\|unset` and the interactive form, each key with its origin (set, default, preset, unset); a default is never written. `[modules]` (`hooks`, `brain`, `graph`) acts at runtime: `hooks = false` silences every hook but the guard, `brain = false` turns off the wiki lane and the `brain_*` tools, `graph = false` the `graph_*` tools. `loomux mcp --root` and the upward search pick the project whose modules count. The guard refuses an agent `loomux init`, a writing `config` and `area add`; an agent proposes a change with `config set`/`unset --propose`, which a human reviews with `config proposals` and applies with `config apply`. `--global` is wired; since 4c-1 it knows the local model's `[model]` keys. The bridge searches upward from the MCP working directory, which a human still measures (`parity/stufe-4a-1.md`) | 3 ✅ | 3 |
| **4a-2** | 🚧 built; human steps open (`init --yes` on a fresh clone with an empty `git status`, one host set up interactively, a project `.mcp.json` approved once in Claude Code, a fresh probe of the flat `.agents/hooks.json` with a running agy) | ultraloom + ultra-brain | `loomux init`: ulinit rewritten onto the schema and `tui` (`write` and the host-file merge moved with their tests), modules with all/each/none, `--yes`, `--dry-run`, `--detect-only`, `--hosts`; every change as a diff, nothing overwritten, a backup before the first change except `.loomux/config.toml`, which is replaced whole once its readers accept the new text, a host file or `.mcp.json` it cannot read stopping the run with nothing written, `answers.toml` and `installed.toml` under `.loomux/state/`. Parts: the newest release at `${LOCALAPPDATA}/loomux/bin/loomux.exe` (`selfupdate.Install`, a first install) or `bin/loomux.exe` built in a checkout; `.loomux/config.toml`, `.gitignore`, `AGENTS.md`, `.mcp.json`; host entries without a marker; git hooks; `verify-until-green` and the five brain skills, in English; `area add`, the merge hook, `graph build`; tools checked, never installed, except the local model, which the part `model` pulls into Ollama on confirmation when it is missing (since 4c-1, 2026-09-26). `loomux merge-hook install\|status\|remove\|record`: the post-merge hook calls the binary instead of baking paths in; 14 cases against `brain-mcp hook`, eleven without a difference. The guard refuses an agent `merge-hook install` and `remove`. Antigravity (built 2026-09-25, fusion spec #21, and #23, which awaits sign-off): four entries (`PreInvocation`, `PreToolUse`, `PostToolUse`, `Stop`) in the group `loomux` of `.agents/hooks.json` that call the installed binary through `cmd.exe` as `%LOCALAPPDATA%/loomux/bin/loomux.exe … --root ..`, none while `LOCALAPPDATA` contains whitespace or a character `cmd.exe` reads as whitespace or syntax; `PreInvocation` and `Stop` as a flat list of handlers, which agy 1.2.11 requires; the answers measured with agy 1.2.11 on 2026-09-25: `pre-tool-use` refuses with exit 2, `post-tool-use` warns with exit 2 without aborting, a held stop continues with `{"decision":"continue"}` on stdout, context goes out as `injectSteps` at the first `PreInvocation`, and at a later one only to say the session is left uncounted; `run_command` (`CommandLine`, measured) and `send_command_input` (`Input`, not measured) and `manage_task` with `send_input` (`Input`, measured) go through the command rules; post-edit reads its files from the `toolCall` agy's PostToolUse carries; other groups carried over, one running our command named; the skills under `.agents/skills/`; the rules of ultraloom's `internal/agenthooks` built into the host-file merge instead of moving the package (`parity/stufe-4a-2.md`). Meant to replace the two manual commands of a fresh clone | 4a-1 🚧 (built, human steps open), 2b ✅ (commit language), 2c ✅ (`stop`, `subagent-*`), `feat/self-update` ✅ (canonical location, `internal/swap`) | 3 |
| **4c-1** | 🚧 built 2026-09-26; self-use open (human steps: a wiki page in obsidian-ai, the `pktmon` capture, the measurement against the real model) | ultra-brain | The local model: Ollama client on loopback only (no proxy, no redirect, 2 s to connect, 30 s in all for a question; the pull has no total limit), gate, the prompt `vorschlag-v4`, `[model]` in the machine-wide `config.toml` through `config --global` and per area only `enabled` and `roles`, proposals for `local_only` cases in `reconcile`, checked by the evidence binding; `loomux dev fake-ollama` and seven recorded cases against the reference (`parity/stufe-4c-1.md`). Fixes two bugs inherited in 3b: `approve --reject` advances the page's sources and the register, and halts on a source that moved again; `reindex` and `approve` share a lock per area (`<state>/areas/<scope>.lock`) | 4a-1 🚧 (built; the per-area `[model]` keys are in the schema), 3 ✅ | 3 |
| **4c-2** | 🚧 built 2026-09-26; parity ✅ 2026-09-27 (50/50 ranks equal to the reference on `keyword`); self-use done 2026-09-27 (corpus `fast` 40/50 against the baseline 43/50, everyday latency through the service, `hooks` and `repos` with `--out`), everyday quality open (qmd backbone: the index is embedded under CUDA, which crashes on this machine, and the user decided on one machine-wide `[search] backbone` in `<state>/config.toml`, which the daemon and every qmd command line take; left is to embed again under the chosen backbone and measure) | ultra-brain | The search bench: `loomux dev bench search` measures the rank of the expected source per question (a hit at rank ≤ 3) and, with `--latency`, the latency of catalog, read and the three profiles, over an area's question set through the search service or over the checked-in corpus `v1` (`testdata/bench/search/v1`, 100 notes, 50 questions, baseline 43/50) through the qmd command line in a throwaway state and a named index `loomux-bench-<random>`, so the shared `index.yml` is never touched; a lock per name, stale indexes of a crashed run swept. The question set and the corpus are checked before anything is written. The group `dev bench hooks\|repos\|search`, renamed from `dev bench-hooks` and `dev bench` (a major release); all three write one report shape with `--out <dir>`, `.md` and `.json` both or neither, times in milliseconds, and `docs/benchmarks.json` moved to milliseconds. `dev bench repos` loses `--json-out`; its `--out` takes a directory. Fixes a bug already on master: `--timeout` per repository was read and never used. Parity file `parity/stufe-4c-2.md` | 3 ✅ | 3 |
| **4d** | open | ultra-brain | `loomux convert` and `loomux fetch` over `pdftotext` and `yt-dlp`, with the model roles `describe` and `place` and the judges they need | 4c-1 🚧 (built, self-use open) | 3 |
| **4e** | open | — | Host switch-over, a checklist a human ticks off in `parity/stufe-4e.md`, and one clean-up pull request: reconcile the machine state by hand, after which `LegacyBrainDirUntilStage3`, `ReadAreaManifestUntilStage4` with the old manifest names and `Manifest.Lanes` go in a pull request of their own; four obligations `parity/stufe-3a.md` had put on the dropped `migrate` (carrying `merge-events.done.tsv` and `qmd-collections.json` over, the read-only areas' declarations, the asides of a killed swap, the `brain reindex`/`brain reconcile` advice) still need a carrier; per host `loomux init` and a smoke test, remove old entries by hand (`ulguard` and `brain guard` entries, which `init` leaves and names). A post-merge hook `brain-mcp` set up shows as `unrecorded`, since the old `hooks.tsv` is not read: `loomux merge-hook install` per host takes it over. `loomux migrate` is dropped (fusion spec, addendum #19) | 4a-2 🚧 (built, human steps open), 4c-1 🚧 (built, self-use open), 4d; a remote for `brain-knowledge`; ultra-brain's branch `feature/artefakte-nach-lebensdauer` read against loomux (fusion spec #20; read on 2026-09-27, no fix needed, six decisions open, `parity/artefakte-nach-lebensdauer.md`) | 3 |
| **4f** | proposed (fusion spec #24, awaiting sign-off) | — | No reference to the predecessor projects is left in loomux, as the user set on 2026-09-27: the detection of old host entries goes once every host is switched over, comments, package docs and messages are reworded without the names, and whatever stays — the recorded evidence under `testdata/cases/*-source`, the changelog, the working papers under `docs/.superpowers/` — stays only by an explicit decision. The fusion spec lists the classes of references with a proposal each | 4e | — (proposed; the fusion spec ranks it once signed off) |
| **G1** | ✅ | new | Ranking and blast radius as libraries, held to the reference by ported test vectors | — | — |
| **G2a** | ✅ | new | The extractor, the resolver, the store, the freshness probe, and `graph build` / `graph check` | — | — |
| **G2b** | ✅ | new | The query: the lexical seed, the ask sidecar, `loomux graph ask` | — | — |
| **G3** | ✅ | new | `graph_find_code` and `graph_check_freshness` on the 1b-2 gateway; a query never builds a first graph | — | — |
| **G4a** | ✅ | new | Navigation palette (`callers`, `skeleton`, `grep`, `map`, `stats`) and 4 MCP tools (`graph_file_api`, `graph_trace_calls`, `graph_find_all`, `graph_repo_map`) with fail-closed privacy | G3 ✅ | — |
| **G4b** | ✅ 2026-09-23 | new | Git-diff blast radius (`blast.Radius`, `graph blast`, MCP `graph_blast`), `check graph-fresh` and `check blast-audit`, the verify kind `graph` (in the profile default `precommit`, `not-applicable` without a graph), and the post-edit blast monitor | G4a ✅ | — |
| **G5a** | ✅ 2026-09-26 (accepted on `iam_backend` and `ultra-brain`, `docs/.superpowers/parity/code-g5.md`; pulled forward by the user on 2026-09-26) | new | The extractor interface (`extract.Language`, the fixed list in `extract/all`) and a shared tree-sitter core on `gotreesitter`, a Tree-sitter runtime in pure Go, so the binary stays CGo-free (`CGO_ENABLED=0` build in the gate); Python: modules, classes, functions and methods, imports, base classes, calls through `self`/`cls` and the bases, constructor calls to `__init__`, in a name index of its own; an extract cache per file and `graph build --no-reuse`; parse errors counted per language instead of failing the build; Python test paths for `blast`; one graph lane per run, with a Python preset | G4b ✅ | — |

The stages and what each waits for, drawn from the "Depends on" column above
(`P` is the priority of an open stage, 1 first). A pull request that changes a
row changes the node with it:

```mermaid
flowchart TD
    subgraph Fusion["Fusion track"]
        s1a["1a pilot, guard, post-edit"]:::done
        s1b1["1b-1 brain read commands"]:::done
        s1b2["1b-2 serve, MCP, stdio bridge"]:::done
        s1b3["1b-3 wiki and docs moved in"]:::done
        s2a["2a check chain"]:::done
        s2b["2b commit-msg"]:::done
        s2c["2c stop gate, subagent hooks"]:::done
        s3a["3a detect: reindex, embed, reconcile"]:::done
        s3b["3b decide: cases, case, approve"]:::done
        s3c["3c maintain: brain check, wiki types"]:::done
        s4a1["4a-1 schema and config · P3"]:::partial
        s4a2["4a-2 init · P3"]:::partial
        s4c1["4c-1 local model · P3"]:::partial
        s4c2["4c-2 search bench · P3"]:::partial
        s4d["4d convert, fetch · P3"]:::planned
        s4e["4e host switch-over · P3"]:::planned
        s4f["4f no predecessor references · proposed"]:::planned
    end

    subgraph Code["Code graph track"]
        g1["G1 ranking and blast libraries"]:::done
        g2a["G2a extractor, graph build and check"]:::done
        g2b["G2b graph ask"]:::done
        g3["G3 graph MCP tools"]:::done
        g4a["G4a navigation palette"]:::done
        g4b["G4b diff blast, graph kind, edit monitor"]:::done
        g5a["G5a extractor interface, Python on gotreesitter"]:::done
    end

    s2a --> s2c
    s1b1 --> s3a
    s3a --> s3b
    s3a --> s3c
    s1b2 --> s3c
    s3a --> s4a1
    s3b --> s4a1
    s3c --> s4a1
    s4a1 --> s4a2
    s2b --> s4a2
    s2c --> s4a2
    s4a1 --> s4c1
    s3a --> s4c1
    s3b --> s4c1
    s3c --> s4c1
    s3a --> s4c2
    s3b --> s4c2
    s3c --> s4c2
    s4c1 --> s4d
    s4a2 --> s4e
    s4c1 --> s4e
    s4d --> s4e
    s4e --> s4f
    g3 --> g4a
    g4a --> g4b
    g4b --> g5a

    classDef done fill:#d4edda,stroke:#28a745,color:#155724
    classDef partial fill:#fff3cd,stroke:#d39e00,color:#664d03
    classDef planned fill:#f1f3f5,stroke:#868e96,stroke-dasharray:5 5,color:#495057
```

What the two source repositories can do and no stage had taken on yet is listed
in the [fusion spec](../.superpowers/specs/2026-09-14-loomux-fusion-design.md) under "Nachgetragen", each item with a
proposed stage or a proposed removal; the assignment holds once it is signed off.
Signed off are items 1 and 2 (Stage 3c) and 3 and 17 (Stage 3a) on 2026-09-19;
5, 7, 8, 10, 11, 13 and 16 (Stage 4a-2), 12 (dropped) and 18 (dropping
`brain check code`) on 2026-09-23; 9 (dropped from `init`), 15 and 21 (Stage
4a-2), 19 (dropping `loomux migrate`), 20 (before 4e) and 22 (the follow-up
project Flow) on 2026-09-24. Items 1 and 2 were built with 3c, 3 and 17 — `reindex`
and `embed` as commands, which had no stage until 2026-09-19 — with 3a. Items 5
(the post-merge hook, as `loomux merge-hook`), 7 (the brain skills), 8
(`verify-until-green`), 10 (the generated `AGENTS.md`), 11 (the `.gitignore`
entries), 15 (the binary at its canonical place) and 16 (`init --detect-only`)
were built with 4a-2 on 2026-09-24, item 21 (the rules for `.agents/hooks.json`)
on 2026-09-25. Items 4, 6, 14 and 23 await sign-off, and so does item 24 (Stage
4f), whose goal the user set on 2026-09-27.

Each stage ends green and is handed over on its own, with its own plan and — once
it is done — its own parity file recording every ruling it made.

## Capabilities

Where each capability stands:

| Pillar / Capability | Origin | Description | Status |
|---|---|---|---|
| **1. Hooks & Guard** | | | |
| Unified Pre-Tool Guard | ultraloom + ultra-brain | Single-pass validation of write barriers, path protections, and forbidden commands (<35ms budget; 32–34ms measured on predecessor; a write in a linked worktree measured 34.6 ms warm (2026-09-16)). Linked git worktrees of a registered workspace are writable without a registry entry of their own. Registry and area declarations are read through the same checks as the brain commands; a broken entry refuses every write. | ✅ **Implemented** (Stage 1a) |
| Post-Tool Check Lanes | ultraloom | The `edit` profile of `[verify]` on the file that was just edited, for its stack and in its area — `go vet` and `gofmt` of the one file, the in-process wiki lint, ruff/mypy, eslint/tsc, stylelint and the rest, from the same presets `loomux check` runs. Commands start as argv, without a shell. A failing lane exits 2; lanes skipped for a missing tool or a spent budget (`--budget`, default 50 s) are named back to the model. | ✅ **Implemented** (Stage 1a; lanes from `[verify]` since 2a) |
| Post-Tool Blast Monitor | new | After a `.go` edit with no red lane, the post-edit hook re-extracts the file, compares each symbol's body hash with the graph on disk, and names the direct callers in other files of what changed or went missing (at most ten), plus a note for a changed type, whose coupling the graph does not wire. Silent without a graph, on a graph of another schema or one that no build with this binary's Go extractor wrote, or on a parse error; never blocks. Costs 24.7 ms on this repository's 7.07 MiB graph (2026-09-23). | ✅ **Implemented** (Stage G4b) |
| Session Start | ultraloom | Records the commit a session starts on and warns when the binary in the project is older than `go.mod`, `go.sum`, a `.go` file under `cmd/` or `internal/`, or a file under `flows/` whose path has no element starting with `_` or `.`. Since the flow runtime moved in (a follow-up project beside the migration, see the [roadmap](../../README.md#roadmap)) it also announces every flow run waiting at a gate, on every start, with the command a human answers it with, and names a project folder a bundled flow ignored. Announces only; never blocks a turn. | ✅ **Implemented** (Stage 1a; flow runs since Flow A) |
| Subagent Drift & Stop Gate | ultraloom | `loomux hook stop` runs the `stop` profile (by default `lint`, `types`, `test` and `coverage`, within a 270 s budget; `graph` stays out, see Stage G4c on the [roadmap](../../README.md#roadmap)) at every turn end with new content, holds the turn with exit 2 on a red lane and gives up after 3 blocks in a row; a turn end with nothing new costs 169.5 ms on this repository. `subagent-start` and `subagent-stop` snapshot `origin`, the local branches and `HEAD`, and park what moved for the main agent's stop gate. Host adapters implemented for Claude Code and Antigravity. | ✅ **Implemented** (Stage 2c) |
| Check Commands | ultraloom | `loomux check commit-msg` (the language of every line, the Conventional Commits header, `--language`, `--calibrate` and `[commit]`), `check gofmt` (formatting, with the exit code `gofmt -l` does not give), `check gocover` (100% per function against a profile, or a total with `--floor`), and the two halves of the graph lane, `check graph-fresh` (rebuild on drift, red only on a failed rebuild, a probe error or a held lock) and `check blast-audit` (red when a changed area with enough callers has no changed test). `dev covergate` is gone. | ✅ **Implemented** (Stage 1a; `gocover` 2a; `commit-msg` 2b; `graph-fresh`, `blast-audit` G4b) |
| Check Chain Table | ultraloom | One `[verify]` table drives `loomux check <profile>` and the post-edit hook: presets per stack that work without any config, one lane per kind, stack and area (the `graph` kind runs once per run, in the root, carried by the first stack with a graph command), `after` edges instead of stages, a verdict per kind, and `--show` to print what runs. A lane can name files it `needs`: the C++ lanes on the build tree wait for `build/CMakeCache.txt` and are `unready` until the build is configured, while an edit still runs `clang-format` on the file. Child process trees are killed whole on a timeout (a Job Object on Windows). | ✅ **Implemented** (Stage 2a) |
| Worktree Mirroring | ultraloom | `loomux worktree link\|unlink` mirrors the main checkout's ignored directories into a subagent's git worktree as NTFS junctions (Windows only; elsewhere it refuses rather than fall back to a symlink) and tracks the sessions sharing it; `worktree remove <path>` takes the junctions out before git removes the worktree. | ✅ **Implemented** (Stage 1a) |
| Setup Report | ultraloom | `loomux status` (also `doctor` and `explain`, one code path) reports a project's detected stacks, its wiki directory, its verification lanes and its hook setup, and names old hook entries a newer command supersedes. | ✅ **Implemented** (Stage 1a) |
| Configuration and Setup | ultraloom + new | `loomux config` shows every setting of `.loomux/config.toml` with its origin (set, default, preset) and changes it line by line after confirmation; an agent stores a change with `--propose` instead, which a human reviews with `config proposals` (with its diff) and applies or rejects with `config apply\|reject`; `--global` addresses the machine-wide file. `loomux init` sets a project up in modules (hooks, wiki, graph) and writes the choice to `[modules]`, which applies at runtime. Replaces ulinit, `install.ps1` and the hook half of `brain init`. Built with 4a-1: `loomux config` and `[modules]` with its runtime effect; built with 4a-2: `loomux init` with its parts (binary, configuration, `.gitignore`, `AGENTS.md`, `.mcp.json`, a check of the tools, host entries, git hooks, skills, area, merge hook, graph), `--dry-run`, `--detect-only` and `--yes`; since 4c-1 the part `model`, which pulls the local model into Ollama when it is missing. Its runs by a human on a fresh clone and a host are open | 🚧 **In migration** (4a-1 and 4a-2 built, human steps open) |
| Zone-Free Start Path | new | Go's local time zone stays off the hook path: the TOML parser builds its local zones on first use (`third_party/toml`), and a gate test fails any package init over 500 allocations. `hook pre-tool-use` 7.5 ms warm against 26.5 ms before (measured 2026-09-17). | ✅ **Implemented** (no stage) |
| **2. Skills & Best Practices** | | | |
| Brain and Verify Skills | ultra-brain + ultraloom | `loomux init` lays the five brain skills (`brain-ingest`, `brain-land`, `brain-research`, `brain-review`, `brain-wiki-plan`), translated into English and calling loomux commands, and `verify-until-green` (`loomux check all` instead of `uv run ultraloom check all`) into `.claude/skills/` of a host, and for Antigravity into `.agents/skills/`; an existing skill stays. `session-handover` is dropped (fusion spec #9). Not yet in use on a host | 🚧 **In migration** (Stage 4a-2 built, human steps open) |
| **3. Code Graph & Loop** | | | |
| Go Native AST Extractor | new | Deterministic symbol & call extraction via `go/parser` and `go/ast` alone — no `go/types`, no build ($0, zero dependencies). Wired behind `loomux graph build`; takes on the order of a tenth of a second on this repository, measured with its command and raw output in `docs/en/benchmarks.md`. | ✅ **Implemented** (Stage G2a) |
| Personalized PageRank | new | Power-iteration random-walk ranking over call and dependency graphs, undirected over five relations, max-normalized with a deterministic tie order. Blended with BM25 lexical candidate scoring in `loomux graph ask` (~48 ms warm retrieval). | ✅ **Implemented** (Stage G2b) |
| Blast Radius Engine | new | Transitive closure and impact analysis (`In`/`Out`, depth limits, smallest depth wins). `EdgeWalk`, `Resolve`, and `InDegree` power graph navigation; `blast.Radius` takes a git diff to the symbols its hunks touch, what reaches them, a test signal (`changed`, `stale`, `none`, `na`) and quoted evidence, behind `loomux graph blast` and the MCP tool `graph_blast`. | ✅ **Implemented** (Stage G1, extended in G4a and G4b) |
| Symbol-Coupled Grep | new | Regex search grouped by enclosing symbol and ranked by incoming edge degree (`inDegree`). Provided via CLI `loomux graph grep` and MCP tool `graph_find_all` with fail-closed privacy. | ✅ **Implemented** (Stage G4a) |
| Graph Navigation Palette | new | Complete structural and caller navigation over the deterministic AST graph: `loomux graph callers`, `skeleton`, `map`, and `stats`, plus MCP tools `graph_file_api`, `graph_trace_calls`, and `graph_repo_map` with fail-closed privacy on the cloud channel. | ✅ **Implemented** (Stage G4a) |
| Multi-Language AST | new | CGo-free Tree-sitter extraction on `gotreesitter`, a Tree-sitter runtime in pure Go — no WebAssembly, no C toolchain. Python since G5a: modules, classes, functions and methods with their imports, base classes and calls, resolved in a name index of their own, so no edge crosses languages; a file with syntax errors keeps its file node and is counted in the build report instead of failing the build. `graph build` reuses the extraction of unchanged files from `.loomux/state/graph/cache/extract.json` (`--no-reuse` parses everything). TypeScript/TSX, GDScript and C++ follow in G5b–d; the post-edit blast monitor stays Go-only. | ✅ **Implemented for Python** (Stage G5a); TypeScript/TSX, GDScript and C++ on the [roadmap](../../README.md#roadmap) (G5b–d) |
| MCP Service & stdio Bridge | ultra-brain | `loomux serve` holds two loopback listeners, one per channel, each with its own token, and answers twelve tools over Streamable HTTP — the five `brain_*` tools and, since Stages G3, G4a and G4b, the seven `graph_*` tools (`graph_find_code`, `graph_check_freshness`, `graph_file_api`, `graph_trace_calls`, `graph_find_all`, `graph_repo_map`, `graph_blast`); `loomux serve status` and `stop [--force]` control it, and `loomux mcp` is the stdio bridge a host starts, which starts and replaces the service itself. Since stage 3c the service catches up on a due `reconcile` itself at start and daily after that, holds every `brain_*` tool for the first pass and appends what it found to the answers. `internal/hooks` links none of it: a gate test reads the import graph. The front is held to the Python reference's own MCP front by a recorded case corpus, which compares the text of each `CallToolResult` and `isError` rather than the envelope two different SDKs negotiate. | ✅ **Implemented** (Stages 1b-2, G3, G4a, 3c, G4b) |
| **4. Second Brain & Wiki** | | | |
| Local Markdown Wiki | ultra-brain | The bundle itself lives in `docs/wiki/` (area `project/loomux`, moved page by page on 2026-09-16 and released line by line). `loomux lint <file>` checks one page's links and frontmatter, `loomux wiki-gate` checks the bundle's freshness and structure, and the lane `lint/wiki` checks its structure in `loomux check lint` and at every turn end. Identity registers and the topic graph are written by `loomux reindex` from stage 3a, not by the move; the page rules of `brain check` (OKF, house, federation) and the lint over every bundle came with stage 3c. | ✅ **Implemented** (Stages 1b-3, 3a, 3c) |
| Semantic QMD Index | ultra-brain | Embedding and neural search integration with local caching in `~/.cache/qmd`. `loomux reindex` enters the collections into qmd's `index.yml`, and `loomux embed` generates the pending vectors (row "Reindex and Embedding"). | ✅ **Implemented** (Stage 3a) |
| Reindex and Embedding | ultra-brain | `loomux reindex` rebuilds each area's identity register, catalogs, link graph and qmd collections, running the catch-up pass (`reconcile`) first so that no change of knowledge slips past the review gate; `loomux embed` generates the vectors `reindex` leaves pending, through `QmdMcpPort.Embed`. Moved from `ultra-brain/pkg/index` with its tests. Neither had a stage until 2026-09-19 (item 17 of the fusion spec's gap list). | ✅ **Implemented** (Stage 3a) |
| Reconciliation | ultra-brain | `loomux reconcile` measures every source of the registered areas against its identity register and opens a case in the review centre for a changed source or a merge event, with a package carrying the diff and the evidence. Rewritten from `src/brain/maintenance/` (`reconcile`, `case`, `package`, `derive`, the read side of `vcs`, the read and drop side of the event log). Since 4c-1 (built, self-use open) a case of a `local_only` area carries the local model's proposal when `[model]` is on and the proposal passes the evidence binding; otherwise it opens without one. The cases are decided with stage 3b (row "Deciding Review Cases"). | ✅ **Implemented** (Stage 3a) |
| Deciding Review Cases | ultra-brain | `loomux cases` lists the waiting cases, `loomux case <id>` shows the head, package and proposal (withheld for `local_only` until `--package`), `loomux approve <id>` decides: approve, approve a proposal of one's own with `--amend`, discard with `--reject` or put off with `--defer`. Every claim of a proposal needs a verbatim quote from the package; an approval writes the page, frontmatter, registers, `log.md` and `audit.md`, removes the case and commits exactly those paths. Rewritten from `apply.py` and `vcs.commit_paths`; `evidence`, `patch`, `frontmatter`, `format`, `lookup` and `cases` moved. 26–27 ms warm against 841–852 ms for the Python form for `cases`, `case` and `approve --defer`; a writing approval is not measured (`docs/en/benchmarks.md`, 2026-09-23). | ✅ **Implemented** (Stage 3b) |
| Area Onboarding | ultra-brain | `loomux area add` registers an area, writes `[area]` and `[layout]` into `.loomux/config.toml`, lays out the wiki scaffold and the routing rule in the project instructions, then runs `reindex` (off with `--no-reindex`). From `src/brain/init.py`, **without** the hook half: `.mcp.json` and the hook files are written by `loomux init` (stage 4a-2), which also runs `area add` as its part `area`. | ✅ **Implemented** (Stage 3a) |
| Merge Hook | ultra-brain | `loomux merge-hook install\|status\|remove` puts the post-merge hook into every repository of an area whose manifest says `[maintenance] on_merge = true`, reports it in seven states and takes it back; the hook calls `loomux merge-hook record`, which notes the merge for `reconcile` from the registry, never prints and never fails a merge. `brain-mcp hook` of the reference without the baked-in paths; 14 cases, eleven without a difference. `loomux init` installs it as its part `merge-hook`. Not yet in use on a host | 🚧 **In migration** (Stage 4a-2 built, human steps open) |
| Wiki Upkeep | ultra-brain | `loomux brain check file\|bundle\|all` checks pages, bundles and the federation against OKF and the house rules (Go reference); `loomux lint --scope` lints every registered bundle by the twelve rules of `lint.py`; `loomux wiki init` lays out a bundle, `wiki types` counts the page types across every area with their rank, `wiki retype` renames one type in one bundle. `brain check code` is dropped (fusion spec #18). | ✅ **Implemented** (Stage 3c) |
| Brain Data Commands | ultra-brain | `loomux brain search`, `catalog`, `read`, `neighbors` and `status` over the one registry, held to the Python reference by a recorded case corpus. A registry or area declaration loomux cannot use refuses the call and names the file, entry and reason. | ✅ **Implemented** (Stage 1b-1) |
| Self-Update | new | The machine-wide binary the MCP bridge and `serve` run from lives in `%LOCALAPPDATA%\loomux\bin\loomux.exe` and comes from a release, never from a checkout. `serve` checks the newest release of its own channel through `gh` once a day, verifies `SHA256SUMS` and swaps the file; the next bridge then replaces `serve`. `loomux self-update` does the same by hand, and session start warns when `serve` runs from elsewhere or the update failed. Windows only. Installed by hand on 2026-09-23 after the bridge had run for days from a stale checkout. | ✅ **Implemented** (no stage, `feat/self-update`) |
| Local Model | ultra-brain | Ollama on loopback only writes proposals for `local_only` cases, checked by the same evidence binding as every approval; an outage leads to a case without a proposal, never to the cloud. Plus `loomux dev bench search`, which measures the rank of search hits | 🚧 **In migration** (Stage 4c-1 built for `propose`, self-use open; `dev bench search` built with 4c-2, parity ✅, everyday quality open; `describe` and `place` with 4d) |
| Inbox | ultra-brain | `loomux convert` turns PDFs and transcripts in an area's inbox into markdown with a provenance head; `loomux fetch` pulls a video's subtitles | 📋 **Specified** (stage 4d) |
| **5. Developer Tools** | | | |
| Mutation Tests | ultra-brain | `loomux dev mutants <pkg>` mutates the Go decisions of a package and reports which mutants its test suite does not notice; each stage's mutation round runs on it. | ✅ **Implemented** (Stage 1b-1) |
| Release Rules | new | `loomux dev release next-version\|parse-body\|changelog-insert\|build`: the next version from the tags and a bump, the check of a pull request body, the changelog entry and the cross build, called by `ci/` and the release workflow. | ✅ **Implemented** (no stage) |
| Hook and Repository Benchmarks | new | `loomux dev bench hooks` times the hook commands of a case file; `dev bench repos` times the hooks and `graph build` on a repository or on the repositories of the open-source matrix and audits their lanes. Both write the report shape they share with `dev bench search` through `--out`. | ✅ **Implemented** (`hooks` Stage 1a, `repos` no stage; grouped under `dev bench` with 4c-2) |

*Origin: `ultraloom` or `ultra-brain` — migrated from that repository; `new` — built for loomux, not carried over.*

*Legend: ✅ Implemented & Verified in Binary · 🚧 In Active Migration / Fusion, or built and not yet in loomux's own use · 📋 Fully Specified & Ready for Build*
