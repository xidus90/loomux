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

### `[worktree]` (Subagent Isolation Mirrors)
Configures paths automatically mirrored into isolated subagent git worktrees via NTFS junctions (Windows) or symlinks (POSIX).

```toml
[worktree]
mirrors = [
  "node_modules",
  ".cache",
  "vendor",
  "testdata/large-fixtures"
]
```

| Field | Type | Description |
|---|---|---|
| `mirrors` | array of strings | Directory paths from the main checkout to junction-link into worktrees, avoiding expensive re-downloads. |

---

### `[graph]` (Code Graph Engine)
Configures AST extraction, indexing boundaries, and freshness detection.

```toml
[graph]
extensions = [".go", ".ts", ".tsx", ".py", ".rs"]
exclude = ["vendor/**", "dist/**", "node_modules/**", "**/*_test.go"]
freshness_check = "hash" # "mtime" or "hash"
```

| Field | Type | Description |
|---|---|---|
| `extensions` | array of strings | File extensions to parse into the AST code graph. |
| `exclude` | array of strings | Glob patterns to ignore during graph construction. |
| `freshness_check` | string | Strategy for working tree drift detection: `"mtime"` (<1ms) or `"hash"` (<3ms, bit-exact). |

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
mode = "strict"
never = [
  "**/.env*",
  "**/credentials.json",
  "**/*.pem",
  "**/*.key"
]
```

| Field | Type | Description |
|---|---|---|
| `mode` | string | Privacy enforcement mode (`"strict"` or `"standard"`). |
| `never` | array of strings | Glob patterns guaranteed to be stripped from all graph inlining and external API payloads. |

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
mirrors = [
  "node_modules",
  ".cache",
  "bin"
]

# --- Code Graph & AST Settings -----------------------------------------------
[graph]
extensions = [".go", ".ts", ".py"]
exclude = ["vendor/**", "dist/**"]
freshness_check = "hash"

# --- Curated Language Review Skills ------------------------------------------
[skills]
suites = ["review-go", "review-security"]
sync = [".claude/skills", ".agents/skills"]

# --- Privacy Boundaries ------------------------------------------------------
[privacy]
mode = "strict"
never = [".env*", "*.key", "credentials.json"]
```
