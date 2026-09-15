# Getting Started with Loomux

This guide walks you through installing Loomux, initializing a repository, and wiring Loomux hooks into your AI coding agents (**Claude Code**, **Google Antigravity**, and **Cursor**) in under 3 minutes.

---

## 1. Prerequisites

- **Go**: Version 1.25 or newer (for building from source or using `go install`).
- **Git**: Version 2.30 or newer.
- **Operating System**:
  - **Windows (x64)**: First-class native support (NTFS junctions, Windows Job Objects).
  - **Linux / macOS**: Fully supported (symlinks, POSIX process groups).

---

## 2. Installation

### Option A: Install via Go (Recommended)
```bash
go install github.com/xidus90/loomux/cmd/loomux@latest
```
Make sure your `$GOPATH/bin` (or `%USERPROFILE%\go\bin`) is in your system `PATH`.

### Option B: Build from Source
```bash
git clone https://github.com/xidus90/loomux.git
cd loomux
go build -o bin/loomux.exe ./cmd/loomux
```

Verify your installation:
```bash
loomux --version
# Output: loomux 0.1.0-fusion
```

---

## 3. Quickstart in 3 Steps

### Step 1: Initialize Your Repository
Stand in the root of your project and run:
```bash
loomux init
```
This command:
1. Detects active coding agent harnesses in your workspace (`.claude/`, `.agents/`, `.cursor/`).
2. Creates the configuration folder `.loomux/`.
3. Creates a starter `.loomux/config.toml` (declaring write boundaries and verify lanes).
4. Configures agent hook files to invoke the `loomux hook` entry points.

> [!NOTE]
> Pass `--dry-run` to see what files would be created without writing to disk:
> `loomux init --dry-run`

### Step 2: Review Your Configuration
Open `.loomux/config.toml`. A minimal starter configuration looks like this:

```toml
# .loomux/config.toml

[project]
name = "my-project"

[policy.paths]
# Protect sensitive files from accidental agent modification
rules = [
  { match = [".env*", "*.pem", "*.key"], reason = "Secrets must not be written by an agent" },
  { match = ["package-lock.json", "go.sum"], reason = "Lock files are managed by package tools" }
]

[policy.commands]
# Block unapproved destructive commands
rules = [
  { regex = '(^|\s)git\s+push(\s|$)', reason = "Pushing to remote requires explicit human decision" }
]

[verify]
# Multi-lane check chain executed at turn completion
lanes = [
  { name = "lint", command = "golangci-lint run" },
  { name = "test", command = "go test -v ./..." }
]
```

### Step 3: Verify the Setup
Run the diagnostic doctor:
```bash
loomux status
```
You should see:
```text
=== loomux Hook Inspection ===
Project root:     C:\Projects\my-project
Harnesses found:  Claude Code, Antigravity
Hook status:      PreToolUse (Active), PostToolUse (Active), Stop (Active)
Verify lanes:     lint (golangci-lint), test (go test)
Write barrier:    Enforced (global registry + project policy)
Overall status:   READY (Green)
```

> [!NOTE]
> The write barrier opens only the trees listed in the global registry
> (`%LOCALAPPDATA%\loomux\registry.toml`). A linked git worktree of a repository
> registered with `workspace = true` counts as part of that repository and needs no
> entry of its own; the barrier reads git's worktree files and starts no `git`
> process. A worktree moved without `git worktree repair` stays shut.

---

## 4. Agent Harness Integrations

Loomux works seamlessly alongside your favorite agent harnesses:

### Claude Code
`loomux init` registers Loomux in `.claude/settings.json`:
```json
{
  "hooks": {
    "PreToolUse": "loomux hook pre-tool-use --host claude",
    "PostToolUse": "loomux hook post-tool-use --host claude",
    "SessionStart": "loomux hook session-start --host claude",
    "Stop": "loomux hook stop --host claude"
  }
}
```
Whenever Claude Code attempts a `Write`, `Edit`, or shell command, Loomux validates the policy in **<35ms**. If a violation occurs, the tool call is blocked with exit code 2 and an explanatory message is displayed in Claude's turn.

### Google Antigravity
For Antigravity, hooks are registered in `.agents/hooks.json`:
```json
{
  "hooks": [
    {
      "event": "PreToolUse",
      "command": "loomux hook pre-tool-use --host antigravity"
    },
    {
      "event": "PostToolUse",
      "command": "loomux hook post-tool-use --host antigravity"
    }
  ]
}
```

### Cursor & MCP Clients
Start the local MCP service:
```bash
loomux serve --port 8080
```
Or use the stdio bridge directly in your Cursor MCP configuration:
```json
{
  "mcpServers": {
    "loomux": {
      "command": "loomux",
      "args": ["mcp"]
    }
  }
}
```

---

## 5. Next Steps

- **[Configuration Reference](configuration.md)**: Deep dive into all configuration sections (`[policy]`, `[verify]`, `[worktree]`, `[graph]`).
- **[CLI Reference](cli-reference.md)**: Explore the complete command manual with all flags, options, and exit codes.
- **[Architecture & Concepts](architecture.md)**: Learn about Karpathy's LLM OS, Google Knowledge Items, and Graft's AST GraphRank.
- **[Hook Lifecycle](hooks.md)**: Understand the pre-tool write barrier, post-tool blast monitor, and event stream.
