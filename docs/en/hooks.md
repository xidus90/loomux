# Loomux Hook Lifecycle & Host Integration

This document details the hook execution lifecycle, payload specifications for different agent harnesses, and the decoupled event stream architecture.

---

## 1. The 4-Phase Hook Lifecycle

Loomux hooks intercept agent interactions at four critical phases:

```mermaid
sequenceDiagram
    autonumber
    actor Dev as Developer / Prompt
    participant Host as Agent Harness (Claude / Antigravity)
    participant Pre as loomux hook pre-tool-use
    participant Post as loomux hook post-tool-use
    participant Stop as loomux hook stop
    participant Journal as events.jsonl (Append-Only)

    Dev->>Host: User Prompt
    Host->>Host: Plans tool action

    rect rgb(240, 248, 255)
    Note over Host,Pre: Phase 1: Pre-Tool Inspection (<35ms)
    Host->>Pre: Tool Call Payload (stdin)
    Pre->>Pre: Evaluate Policy & Global Write Barrier
    alt Forbidden Path or Disallowed Command
        Pre-->>Host: Exit 2 + Deny Envelope (stderr explanation)
    else Permitted
        Pre->>Journal: Append Event (<0.2ms)
        Pre-->>Host: Exit 0 (Proceed)
    end
    end

    Host->>Host: Executes Tool (File Edit / Shell Command)

    rect rgb(255, 250, 240)
    Note over Host,Post: Phase 2: Post-Tool Blast Analysis (<5ms)
    Host->>Post: Tool Output & Modified Paths (stdin)
    Post->>Post: Hash Dirty Files & Calculate Blast Radius
    Post->>Journal: Append Event (<0.2ms)
    Post-->>Host: Exit 0 (Inline caller warnings)
    end

    rect rgb(245, 255, 245)
    Note over Host,Stop: Phase 3: Turn Completion Verification
    Host->>Stop: Turn Finished Payload (stdin)
    Stop->>Stop: Run [verify] Check Chain (Linter, Tests, Coverage)
    alt Quality Gate Fails
        Stop-->>Host: Exit 1 + Failure Feedback (Halt turn)
    else All Lanes Pass
        Stop-->>Host: Exit 0 (Turn Green)
    end
    end
```

---

## 2. Host Payload Formats

Loomux automatically normalizes payloads from supported coding agent hosts.

### A. Claude Code
Claude Code pipes standard tool call objects to `stdin`:

```json
{
  "tool_name": "Write",
  "tool_input": {
    "file_path": "C:/Projects/loomux/.env"
  }
}
```

For shell commands (`Bash`):
```json
{
  "tool_name": "Bash",
  "tool_input": {
    "command": "git push origin main"
  }
}
```

### B. Google Antigravity
Antigravity packages tool invocations in camelCase structures:

```json
{
  "toolCall": {
    "name": "write_to_file",
    "args": {
      "TargetFile": "C:/Projects/loomux/.env",
      "CodeContent": "SECRET_KEY=12345"
    }
  }
}
```

For Antigravity shell execution (`run_command`):
```json
{
  "toolCall": {
    "name": "run_command",
    "args": {
      "CommandLine": "git push origin main"
    }
  }
}
```

Loomux parses both representations natively into a unified internal representation (`tool`, `input`).

---

## 3. The Decoupled Event Stream Architecture

A common architectural trap in agent tooling is having hooks make synchronous HTTP calls or IPC requests to a background daemon. This introduces significant latency (>10ms) and creates a catastrophic failure mode if the background service crashes.

Loomux guarantees total isolation through an **append-only file journal**:

```
Hook Process (loomux hook pre/post-tool-use)
   │
   └── Appends single JSON line (O_APPEND, <0.2ms)
       │
       ▼
.loomux/state/journal/events.jsonl
       │
       ▲
       └── Background Tailing Routine (inotify / stat polling)
           │
   ┌───────┴────────────────────────┐
   ▼                                ▼
loomux serve                   Connected Browsers
(Root Gateway)                 (SSE /api/events)
```

1. **Zero-Overhead Write**: Hooks append events using raw file descriptors in `<0.2ms`.
2. **Crash Resilience**: If `loomux serve` is not running, hooks continue working with zero degradation.
3. **Live Sync**: When `loomux serve` runs, it tails `events.jsonl` and broadcasts updates via Server-Sent Events (SSE) to the Web OS dashboard and Kanban boards in real time.

---

## 4. Refusal Envelopes & Diagnostics

When `loomux hook pre-tool-use` blocks an operation, it outputs:

### 1. `stdout` Envelope (for the Agent Harness)
```json
{
  "hookSpecificOutput": {
    "permissionDecision": "deny",
    "permissionDecisionReason": "secrets are not written by an agent: .env"
  }
}
```

### 2. `stderr` Message (for Terminal & Logs)
```text
loomux policy refused this Write:
  - secrets are not written by an agent
```

### 3. Exit Code: `2`
Exit code 2 informs the agent harness that the tool call was explicitly rejected and must not be retried with the same parameters.
