# First-time setup

← [Installing loomux](install.md) · [Overview](README.md) · [Configuration](configuration.md) →

This page takes a project from an installed loomux to a working setup: it initializes the repository, reviews the configuration, checks the result, wires your agent and shows how to register an area by hand. It assumes loomux is installed as [Installing loomux](install.md) describes, and the examples call it as `loomux`; if the folder is not on your `PATH`, call the binary by its full path.

---

## 1. Initialize your repository

Stand in the root of your project and run:
```bash
loomux init
```
This command:
1. Detects the coding agent harnesses in your workspace (`.claude/` → Claude Code, `.agents/hooks.json`, `.agents/skills/` or `GEMINI.md` → Antigravity).
2. Asks per module (`hooks`, `brain`, `graph`) `all`, `each` or `none`, then the commit language.
3. Shows every change as a diff — `.loomux/config.toml` (modules, commit language, the policy rules of your stack; verify lanes come from the presets), `.gitignore`, agent hook entries, git hooks, skills — and writes only what you approve.
4. Puts the newest release at `%LOCALAPPDATA%\loomux\bin\loomux.exe`, which the hook entries call, when none stands there yet (Windows only; see [Installing loomux](install.md)).

Every part, flag and exit code is in the [CLI reference](cli-reference.md#11-project-setup-loomux-init).

> [!NOTE]
> Pass `--dry-run` to see what files would be created without writing to disk:
> `loomux init --dry-run`

---

## 2. Review your configuration

Open `.loomux/config.toml`. `loomux init` wrote only what departs from the defaults (see [Your first day](configuration.md#your-first-day)). A small configuration with rules of your own looks like this:

```toml
# .loomux/config.toml

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

[verify.go.test]
# Lanes run from built-in presets per detected stack; a table changes only
# the keys it names. `loomux check precommit --show` prints what runs.
measuring = "go test ./... -count=1 -covermode=set -coverpkg=example.com/my-project/... -coverprofile={coverprofile}"
```

---

## 3. Verify the setup

Run the diagnostic doctor:
```bash
loomux status
```
You should see (excerpt):
```text
================================================================================
 loomux Hook Inspection
================================================================================
Project Root:    C:\Projects\my-project
Detected Stacks: [go]
...
[PostToolUse] (Matcher: Write|Edit|NotebookEdit)
  -> loomux hook post-tool-use (profile `edit`: lint, types):
     * go (*.go) lint: go vet ./... ; {loomux} check gofmt {file} [preset, parallel]
...
--------------------------------------------------------------------------------
 Lane Tools On This Machine
--------------------------------------------------------------------------------
 [OK] Every configured lane's tool is on PATH.
```

> [!NOTE]
> The write barrier opens the trees listed in the global registry
> (`%LOCALAPPDATA%\loomux\registry.toml`), plus a few other places such as the
> agents' memory and the session scratchpad. A linked git worktree of a repository
> registered with `workspace = true` counts as part of that repository and needs no
> entry of its own; the barrier recognises the worktree from git's own worktree
> files, without starting a `git` process for that. A worktree moved without
> `git worktree repair` stays shut.

---

## 4. Wire your agent

`loomux init` writes the hook entries of the hosts it detects (step 1). This section shows them, and how to connect an MCP client.

### Claude Code
`loomux init` writes these entries into `.claude/settings.json`, calling the binary by its path (`"${LOCALAPPDATA}/loomux/bin/loomux.exe"`); they are shown here with `loomux`. `loomux status` names what is missing. The six entries, with the timeouts `init` writes:
```json
{
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "timeout": 20,
      "command": "loomux hook session-start --host claude --root \"${CLAUDE_PROJECT_DIR}\""}]}],
    "PreToolUse": [{"matcher": "Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell",
      "hooks": [{"type": "command", "timeout": 15,
      "command": "loomux hook pre-tool-use --host claude --root \"${CLAUDE_PROJECT_DIR}\""}]}],
    "PostToolUse": [{"matcher": "Write|Edit|MultiEdit|NotebookEdit",
      "hooks": [{"type": "command", "timeout": 60,
      "command": "loomux hook post-tool-use --host claude --root \"${CLAUDE_PROJECT_DIR}\""}]}],
    "Stop": [{"hooks": [{"type": "command", "timeout": 300,
      "command": "loomux hook stop --host claude --root \"${CLAUDE_PROJECT_DIR}\" --budget 270s"}]}],
    "SubagentStart": [{"hooks": [{"type": "command", "timeout": 30,
      "command": "loomux hook subagent-start --host claude --root \"${CLAUDE_PROJECT_DIR}\""}]}],
    "SubagentStop": [{"hooks": [{"type": "command", "timeout": 30,
      "command": "loomux hook subagent-stop --host claude --root \"${CLAUDE_PROJECT_DIR}\""}]}]
  }
}
```
When Claude Code attempts a `Write`, `Edit` or shell command, the guard checks it against the policy within a budget of **35 ms**. If a rule refuses it, the tool call is blocked with exit code 2 and an explanatory message is displayed in Claude's turn.

### Google Antigravity
For Antigravity, hooks are registered in `.agents/hooks.json`:
```json
{
  "loomux": {
    "PreInvocation": [{"type": "command", "command": "%LOCALAPPDATA%/loomux/bin/loomux.exe hook session-start --host antigravity --root ..", "timeout": 20}],
    "PreToolUse": [{"matcher": "write_to_file|replace_file_content|multi_replace_file_content|write_file|edit_file|run_command|send_command_input|manage_task", "hooks": [{"type": "command", "command": "%LOCALAPPDATA%/loomux/bin/loomux.exe hook pre-tool-use --host antigravity --root ..", "timeout": 15}]}],
    "PostToolUse": [{"matcher": "write_to_file|replace_file_content|multi_replace_file_content|write_file|edit_file", "hooks": [{"type": "command", "command": "%LOCALAPPDATA%/loomux/bin/loomux.exe hook post-tool-use --host antigravity --root ..", "timeout": 60}]}],
    "Stop": [{"type": "command", "command": "%LOCALAPPDATA%/loomux/bin/loomux.exe hook stop --host antigravity --root .. --budget 270s", "timeout": 300}]
  }
}
```
`loomux init` writes this group; agy runs it through `cmd.exe` from `.agents/`, hence `%LOCALAPPDATA%` and `--root ..`. Run `loomux init` again after an upgrade: an entry of ours under an older matcher stays as it is, and init appends a block for the tools it lacks, such as `manage_task`. The CLI reference explains how the hooks answer Antigravity.

### Cursor & MCP clients
Start the local MCP service:
```bash
loomux serve
```
Or use the stdio bridge directly in your Cursor MCP configuration:
```json
{
  "mcpServers": {
    "loomux": {
      "command": "C:/Users/<you>/AppData/Local/loomux/bin/loomux.exe",
      "args": ["mcp", "--channel", "local"]
    }
  }
}
```
For Claude Code, `loomux init` writes this entry into `.mcp.json` as
`${LOCALAPPDATA}/loomux/bin/loomux.exe`, which Claude Code expands; for another
client, give the path in the form that client reads.

---

## 5. Registering an area by hand

An area is two declarations: an entry in the machine-wide registry that says
where the tree lies and what may be written in it, and a manifest inside the
tree that says what it is. `loomux area add` writes both, and `loomux init`
runs it as its part `area`; this section is for a human writing them by hand.
The write barrier refuses an agent `.loomux/config.toml`. It keeps the host's
file tools out of the registry only while no registered tree encloses the state
directory, and it does not check shell lines or the file tools of MCP servers
against it. All of this holds within the limits listed in
[Hooks](hooks.md#7-the-decision-path-of-pre-tool-use). Where both files live is
listed in
[Where Things Live](configuration.md#4-where-things-live).

### The registry entry
`%LOCALAPPDATA%\loomux\registry.toml`, one `[[area]]` table per area:

```toml
[[area]]
scope     = "project/my-project"
path      = "C:/Users/me/Documents/GIT/my-project"
wiki      = "C:/Users/me/Documents/GIT/my-project/docs/wiki"
workspace = true
```

- `scope` and `path` are required and must be non-empty strings.
- `workspace = true` opens the whole `path` to writing tools.
- `wiki` opens the bundle it names. A writable area whose manifest declares
  `[layout] wiki` gets that bundle opened without this key, provided the
  manifest lies in the registered `path` (or a linked worktree of it) and its
  scope matches.
- `readonly = true` turns the area's `wiki` into a forbidden zone: the barrier
  refuses every write there, even where a writable area encloses the
  directory. It does not close a `workspace` tree, which is a separate
  question. The manifest of a read-only area is read from the state directory,
  not from its tree.
- At most one area may set `signpost = true`.

### The manifest
`.loomux/config.toml` in the area's root, next to the policy rules:

```toml
[area]
scope = "project/my-project"

[layout]
wiki = "docs/wiki"

[index]
include    = ["**/*.md"]
exclude    = [".venv/**", "node_modules/**"]
unsearched = ["docs/.superpowers/**"]

[privacy]
mode  = "manual_cloud"
never = ["private/**"]
```

- **`[area] scope`** names the registry entry. A `.loomux/config.toml` without
  an `[area]` table is policy only and declares no area.
- **`[layout] wiki`** is relative to the repository root, with forward slashes;
  it may neither leave the repository nor name its root. It is one of several
  ways to start the post-edit hook's wiki lane: the lane also starts when the
  manifest has a `[wiki]` table or `wiki = true`, or when `index.md` or
  `bundle.toml` in `wiki/` (or, where that is missing or empty, in
  `docs/wiki/`) carries `okf_version`. The hook takes the bundle from this key
  when it names an existing directory, else `docs/wiki`, else `wiki`, else a
  wiki beside the repository.
- **`[index] include`** — the search engine knows one pattern per collection
  and sees only the first glob; `loomux brain status` names the others.
- **`[index] exclude`** names paths `loomux reindex` leaves out of the index.
- **`[index] unsearched`** declares what is readable but never searched. qmd
  does not enter dot directories, so `docs/.superpowers/**` is reachable
  through `brain read` and `brain neighbors` but not through `brain search`.
  Declared, `brain status` stops counting those files as missing.
- **`[privacy] mode`** is `local_only`, `manual_cloud` (the default) or
  `automatic_cloud`; any other value is refused. On the cloud channel a
  `local_only` area does not exist: no hits, no contents, and its scope is
  answered as unknown. Nesting does not lift that: its wiki inside another
  area's tree stays hidden through that area's scope as well.
- **`[privacy] never`** names paths no channel reaches.

### What goes wrong
- **Every `loomux brain` command fails with `no manifest found`.** The commands
  read the manifest of every registered area before they look at `--scope`, so
  one area without a declaration fails calls about all the others; only an
  entry with `workspace = true` and no declaration is skipped silently. A
  `.loomux/config.toml` without `[area]` counts as none, and so does an area
  whose directory holds no `.loomux/config.toml`, whatever other files lie
  there. A `.loomux/config.toml` the system refuses to inspect is no absence
  either: it fails the call, a workspace included. The write
  barrier does not mind: registering an area before its manifest exists is
  normal there.
- **Every write is refused with `loomux cannot read the registry, so it
  refuses`.** The barrier reads the registry strictly: a missing `scope` or
  `path`, a duplicate scope, two scopes that flatten to the same state
  directory name, a second `signpost`, `[area]` written instead of `[[area]]`,
  or a registered area's manifest that fails its own checks closes every tree.
  Only the agents' memory, the session scratchpad and the files `open.toml`
  lists stay open. The `brain`
  commands are laxer and skip an entry without `scope` or `path` without a
  word.
- **Pages under a dot directory are never found.** That is the limit of the
  search engine, not an error; declare them in `unsearched`.

---

## 6. Next steps

- **[Installing loomux](install.md)**: Platforms, updates and channels, and building from source.
- **[Configuration Reference](configuration.md)**: Deep dive into all configuration sections (`[modules]`, `[policy]`, `[verify]`, `[worktree]`).
- **[CLI Reference](cli-reference.md)**: Explore the complete command manual with all flags, options, and exit codes.
- **[Architecture & Concepts](architecture.md)**: Learn about Karpathy's LLM OS, Google Knowledge Items, and AST GraphRank.
- **[Hook Lifecycle](hooks.md)**: Understand the pre-tool write barrier, post-tool blast monitor, and event stream.

← [Installing loomux](install.md) · [Overview](README.md) · [Configuration](configuration.md) →
