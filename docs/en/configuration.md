# Loomux Configuration Reference

This document provides a comprehensive reference for `.loomux/config.toml`, the primary configuration file for Loomux projects.

---

## 1. Principles of Configuration

1. **Human-Maintained & Agent-Guarded**:
   > [!IMPORTANT]
   > `.loomux/config.toml` is **never modified by an AI agent**. The write barrier strictly forbids agent writes to `.loomux/config.toml`. Propose changes; a human commits them.
2. **Deterministic & Strict**:
   All regular expressions and path globs are compiled on first use. If any rule contains an invalid regex or missing reason, Loomux refuses startup immediately with a clear error naming the exact line.
3. **Separation of Config and State**:
   - `.loomux/config.toml`: Committed human-authored policies and check chains.
   - `.loomux/state/`: Ephemeral, machine-written state (session files, journal, caches). Always git-ignored.

---

## 2. Configuration Sections

### `[project]`
Top-level metadata describing the project.

```toml
[project]
name = "loomux"
version = "0.1.0"
agents = ["claude", "antigravity", "cursor"]
```

| Field | Type | Description |
|---|---|---|
| `name` | string | Project identifier used for scoped collections and registries. |
| `version` | string | Optional project version string. |
| `agents` | array of strings | Active harness targets wired by `loomux init`. |

---

### `[policy.paths]` (Path Protection Rules)
Defines path patterns that AI agents are forbidden from writing to or editing.

```toml
[policy.paths]
rules = [
  { match = [".env", ".env.*"], reason = "Secrets and environment files must not be written by agents" },
  { match = ["*.pem", "*.key", "id_rsa*"], reason = "Private keys and certificates are human-managed" },
  { match = [".loomux/config.toml"], reason = "The guard's own configuration is protected from agent edits" },
  { match = ["package-lock.json", "go.sum", "uv.lock"], reason = "Lockfiles are maintained by package tools, not by hand" }
]
```

| Field | Type | Description |
|---|---|---|
| `rules` | array of tables | List of path inspection rules. |
| `rules[].match` | string or array of strings | Glob patterns supporting `**` (e.g., `.aws/**`, `*.key`). |
| `rules[].reason` | string (**Required**) | Explanatory message displayed to the agent upon refusal. |

---

### `[policy.commands]` (Command Execution Rules)
Defines shell command patterns executed in `Bash` or `PowerShell` tools that must be blocked.

```toml
[policy.commands]
rules = [
  { regex = '(^|\s)git\s+push(\s|$)', reason = "Pushing commits to remote is an exclusive human decision" },
  { regex = '(^|\s)rm\s+-rf\s+/', reason = "Destructive root filesystem commands are prohibited" },
  { regex = '(^|\s)pip\s+install\s+-r', reason = "Dependencies must be managed via lockfiles and declared workflows" }
]
```

| Field | Type | Description |
|---|---|---|
| `rules` | array of tables | List of command inspection rules. |
| `rules[].regex` | string (**Required**) | Go RE2-compatible regular expression. |
| `rules[].reason` | string (**Required**) | Explanatory message displayed to the agent upon refusal. |

---

### `[verify]` (Check Chains & Quality Gates)
Defines the verification table executed at turn completion (`Stop` hook) or before commit.

```toml
[verify]
timeout = 300 # seconds

lanes = [
  { name = "format", command = "gofmt -l cmd internal" },
  { name = "vet", command = "go vet ./..." },
  { name = "test", command = "go test -v ./..." },
  { name = "covergate", command = "loomux dev covergate --profile coverage.out" }
]

[verify.commit]
language = "en"       # Enforces English commit messages
max_first_line = 72   # Maximum subject line length
```

| Field | Type | Description |
|---|---|---|
| `timeout` | integer | Global timeout in seconds for lane execution (default: 300). |
| `lanes` | array of tables | Individual verification steps executed in parallel or sequence. |
| `lanes[].name` | string | Name identifier of the lane. |
| `lanes[].command` | string | Command line executed in the project root. |
| `commit.language` | string | Commit message language validator (`"en"`). |
| `commit.max_first_line` | integer | Max character length of the git commit title line. |

---

### `[worktree]` (Worktree Mirrors)
Names gitignored directories of the main checkout that `loomux worktree link` makes available in a linked git worktree through a Windows junction. Always read from the main checkout's `.loomux/config.toml`. Junctions exist only on Windows; there are no symlinks on other systems. The mechanism is described in [Hooks](hooks.md#9-worktree-mirroring).

```toml
[worktree]
mirror = [
  "node_modules",
  ".cache",
  "vendor",
  "testdata/large-fixtures"
]
```

| Field | Type | Description |
|---|---|---|
| `mirror` | array of strings | Paths relative to the project root, mirrored so that a worktree does not have to rebuild or re-download them. An entry that is empty, absolute or leaves the project with `..` makes the file count as broken (exit 1). No table, no key or an empty list means nothing to mirror. |

---

### `[graph]` — there is none, and stage G2a settled why

An earlier draft of this manual specified a `[graph]` section with `extensions`,
`exclude` and `freshness_check` fields. None of it exists in the code stage G2a
shipped, and none is planned. The question keeps coming back, so the answer is
here rather than left to be rediscovered:

- **The languages a build parses follow from the extractor, not from a list a
  repository declares.** `internal/code/extract/golang` is a Go extractor; it
  parses `.go` files because that is what it knows how to read. A second
  extractor for another language adds itself the same way — by existing — and
  no config key decides which one runs on a given file.
- **Narrowing a single build to a subset of the tree is a flag on the
  command, not a repository setting.** `loomux graph build` and `loomux graph
  check` already take `--root`; a future narrowing flag on one invocation is
  the right shape for "index only this subtree today", because that choice
  belongs to whoever runs the command, not to the tree being indexed.
- **`freshness_check` never had a decision to make.** `internal/code/freshness`
  always compares size and mtime first and falls back to a content hash only
  when those disagree — the reference implementation's rule, not a strategy a
  project could turn off. There is nothing left to configure.

What narrows or shapes one build belongs to the invocation that runs it, not to
the configuration of the repository being indexed. If this section returns, it
is because a real requirement forced a section-level knob to exist, not because
the manual once sketched one.

---

### `[skills]` (Curated Best-Practice Suites)
Configures language-specific review skills and synchronization destinations.

```toml
[skills]
suites = ["review-go", "review-security", "review-typescript"]
sync = [".claude/skills", ".agents/skills"]
```

| Field | Type | Description |
|---|---|---|
| `suites` | array of strings | Active review skill suites bundled in the Loomux binary. |
| `sync` | array of strings | Directories where `SKILL.md` bundles are provisioned. |

---

### `[privacy]` (Data Protection & Boundaries)
Guarantees sensitive data never leaks to external models or remote telemetry.

```toml
[privacy]
mode = "manual_cloud"
never = [
  "**/.env*",
  "**/credentials.json",
  "**/*.pem",
  "**/*.key"
]
```

| Field | Type | Description |
|---|---|---|
| `mode` | string | `"local_only"`, `"manual_cloud"` (default) or `"automatic_cloud"`; any other value is refused. A `local_only` area does not exist on the cloud channel. |
| `never` | array of strings | Glob patterns of paths that no channel reaches. |

---

## 3. Complete Annotated Example (`config.toml`)

```toml
# ==============================================================================
# Loomux Project Configuration: .loomux/config.toml
# ==============================================================================

[project]
name = "loomux"
version = "0.1.0-fusion"
agents = ["claude", "antigravity", "cursor"]

# --- Path Protection Rules (Exit Code 2 on Violation) -------------------------
[policy.paths]
rules = [
  { match = [".env*", "*.pem", "*.key", "id_rsa*"], reason = "Secrets must not be written by agents" },
  { match = [".loomux/config.toml"], reason = "Guard policy is human-managed and write-protected" },
  { match = [".claude/.no-verify"], reason = "Verification controls cannot be bypassed by agents" },
  { match = ["go.sum", "package-lock.json", "uv.lock"], reason = "Lock files are written by package tools, not by hand" }
]

# --- Command Execution Rules (Exit Code 2 on Violation) ----------------------
[policy.commands]
rules = [
  { regex = '(^|\s)git\s+push(\s|$)', reason = "Whether commits reach the remote is a human decision" },
  { regex = '(^|\s)rm\s+-rf\s+/', reason = "Root filesystem removal is prohibited" }
]

# --- Turn Completion Quality Gates -------------------------------------------
[verify]
timeout = 180

lanes = [
  { name = "gofmt", command = "gofmt -l cmd internal" },
  { name = "vet", command = "go vet ./..." },
  { name = "tests", command = "go test ./..." },
  { name = "covergate", command = "go run ./cmd/loomux dev covergate --profile coverage.out" }
]

[verify.commit]
language = "en"
max_first_line = 72

# --- Worktree Isolation Mirrors ----------------------------------------------
[worktree]
mirror = [
  "node_modules",
  ".cache",
  "bin"
]

# --- Curated Language Review Skills ------------------------------------------
[skills]
suites = ["review-go", "review-security"]
sync = [".claude/skills", ".agents/skills"]

# --- Privacy Boundaries ------------------------------------------------------
[privacy]
mode = "manual_cloud"
never = [".env*", "*.key", "credentials.json"]
```

---

## 4. Where Things Live

| | Location |
|---|---|
| State directory | on Windows `%LOCALAPPDATA%\loomux`, else `~\AppData\Local\loomux`; elsewhere `$XDG_STATE_HOME/loomux`, else `~/.local/state/loomux` |
| Area registry | `<state directory>\registry.toml` |
| Manifest of a writable area | `<area path>\.loomux\config.toml` |
| Manifest of a read-only area, as the write barrier reads it | `<state directory>\areas\<scope>\.loomux\config.toml` |
| Artefacts of a read-only area (`index.md`, `graph.json`, `_identities.tsv`) and its manifest, as `loomux brain` reads them | `%LOCALAPPDATA%\brain\areas\<scope>\` until stage 3 |
| Artefacts of a writable area | its `path` |
| Last reconcile stamp | `%LOCALAPPDATA%\brain\maintenance\last-run.txt` until stage 3 |

`LOOMUX_STATE_DIR` overrides the state directory and
`LOOMUX_LEGACY_BRAIN_DIR` the ultra-brain directory; there is no command-line
flag for either. The legacy directory exists because ultra-brain still writes
those artefacts: loomux has no indexer of its own before stage 3.

`<scope>` is the scope flattened into one directory name: every run of
characters other than `A-Z a-z 0-9 _ . -` becomes one `-`, and dashes at both
ends come off, so `project/loomux` becomes `project-loomux`.

**The knowledge is in none of these places.** It lies wherever an area's
`path` and `wiki` point — a repository, or an ordinary Obsidian vault under
git. loomux keeps no copy of it.

---

## 5. The Write Barrier and the Agents' Memory

The write barrier (`loomux hook pre-tool-use`) lets tools write where the
registry declares an area. A linked git worktree of an area registered with
`workspace = true` belongs to it and needs no entry of its own: it is the same
repository checked out a second time, and the barrier recognises it from git's
own worktree files. Beyond that, three places are always open, for every
user and without a registry entry:

| Where | Open below |
|---|---|
| Claude Code memory | `$CLAUDE_CONFIG_DIR/projects/<project>/memory/`, else `~/.claude/projects/<project>/memory/` |
| Claude Code session scratchpad | `<temp>/claude/<project>/<session>/scratchpad/` |
| Antigravity memory | `~/.gemini/{antigravity,antigravity-cli,antigravity-ide}/knowledge/` and `…/brain/<conversation-id>/` |

- A call that writes only there passes even when the registry cannot be read.
- `.loomux/config.toml` stays locked there too; the manifest is decided before
  anything else.
- Every path is resolved before it is compared, so a link from memory into a
  closed tree is judged at its target. In a call that also writes elsewhere, a
  memory target inside a read-only zone is still refused.
- The rest of `~/.claude` and `~/.gemini` stays shut, `antigravity-backup`
  included: `settings.json`, hooks and plugins steer the agent itself, and an
  agent that rewrites them switches off its own barriers.
- Claude Code loads `MEMORY.md` into later sessions of the same project, so
  what an agent writes there acts like an instruction to future sessions;
  whether Antigravity loads `knowledge/` the same way has not been measured.
  That was accepted knowingly with the decision of 2026-09-13.

A refusal that reads `lies outside every writable tree` lists where writing is
allowed: the permitted trees, then the session scratchpad and the memory trees,
wherever they can be named.
