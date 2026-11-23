# Native DSH Hooks

loomux guards a DeepSeek Harness (DSH) session natively: the plugin
`dsh-loomux-hooks` registers the harness's own seams and runs the project's
loomux hooks as real hook processes in them — the same commands, the same
Claude-dialect protocol, and harness-native refusals, blocks, steerings and
context instead of a bridge.

---

## 1. Install

Two pieces belong together: the hook file loomux writes and the plugin the
harness loads.

**The hook file.** From the project root:

```sh
go run ./cmd/loomux init --yes --hosts=dsh
```

`init` writes `.loomux/dsh/hooks.json`: one entry per event with the matcher
`.*` and an absolute path to the loomux binary, in the Claude `settings.json`
dialect. Once the file exists, a plain `loomux init` maintains it like every
other host entry.

**The plugin.** Install the package `dsh-loomux-hooks` into a profile,
either through the harness's `plugin_manager` (install bundle
`dsh-loomux-hooks`) or by adding it to the profile's bundle list with its
patch. Its configuration is all optional:

```yaml
- id: loomux-hooks
  name: dsh-loomux-hooks
  config: {}
  # config:
  #   configPath: C:/work/proj/.loomux/dsh/hooks.json  # skip the chain
  #   defaultTimeoutMs: 60000                          # for entries without a timeout
```

A restart is needed after installing, and for a changed hook file to be
re-read: what a directory resolves to is remembered until the harness
process restarts. A session without hook entries for its directory runs
unhooked.

## 2. How the hook file is found

The plugin resolves one file per session directory — the harness process's
own directory says nothing about the project under way — and remembers what
it found, in this order:

1. `configPath` from the plugin configuration, if set;
2. `.loomux/dsh/hooks.json` in the agent's working directory;
3. `.claude/settings.json`;
4. `hooks/hooks.json`, the position documented for the codex host.

Nothing found: the seams stay silent for that directory. A file that exists
but cannot be read or parsed: the plugin warns through its logger and that
directory runs unhooked — it does not fall through to the next position, so
a broken file is never silently bypassed. A file with an empty `hooks`
object suppresses hooks on purpose. The whole file is rejected when any
matcher is not a valid regular expression.

## 3. Seams and decisions

The plugin registers six seams. Each hook command is `loomux hook <event>
--host dsh`; the payload arrives as JSON on stdin exactly as in the other
hosts, and `transcript_path` may be empty.

| Harness seam | Hook event | What the verdict does |
|---|---|---|
| `agent/created` | SessionStart | awaited before the first input is released; `additionalContext` → injected as a user message |
| `tools/pre-execute` | PreToolUse | exit 2 or `permissionDecision: "deny"` → the call is refused with the reason; `"ask"` → the harness asks |
| `tools/post-execute` | PostToolUse | exit 2 or deny → the result is blocked and the reason fed back; `additionalContext` → injected after the call |
| `agent/turn-stopping` | Stop | exit 2 or `decision: "block"` → the agent is steered on with the reason |
| `subagent/start` | SubagentStart | `additionalContext` → injected into the child |
| `subagent/end` | SubagentStop | the hooks run and are audited; the verdict is ignored |

Entries run in file order; the first decisive answer of a seam wins, and
several `additionalContext` answers accumulate. Only a `hookSpecificOutput`
whose `hookEventName` matches the event is honoured — a wrong name makes the
JSON a no-op, exactly as in Claude Code.

## 4. When something goes wrong

- **Internal failures never lock.** Exit 1, a timeout, a missing binary and
  unparseable output let the call through, at every seam; only exit 2 and a
  correct JSON verdict decide.
- **A timeout kills the whole process tree**, on Windows through
  `taskkill /T /F`, so a hook that started grandchildren leaves none behind.
  The entry's `timeoutMs` wins over the plugin's `defaultTimeoutMs`, whose
  default is 60000.
- **Loop guard.** The first Stop evaluation per agent run reports
  `stop_hook_active: false`, every later one `true` — hook authors can stop
  re-entering. Independently of that, the plugin ignores the steering of a
  Stop hook after at most eight steerings per agent run.
- **Hooks are outside the harness sandbox.** A hook process runs with your
  user's rights and sees the workspace through `CLAUDE_PROJECT_DIR`; it is
  not confined by the harness's file policy. Write-barrier refusals of the
  agent's own writes are unaffected by this.
- **A warning never breaks a session.** A broken hook file, a dead audit log
  and a misbehaving entry are logged; the session continues.

## 5. Audit

Every hook run appends two records of the known session-log format:
`hook/invoked` (turn, seam, dialect `claude`, handler id
`loomux:<Event>:<index>`, matcher) and `hook/result` (verdict, duration,
exit code, stderr summary of at most 500 characters). When the session log
is unreachable, the plugin logger writes them instead; the audit never
changes a verdict.

## 6. Verified against the live harness

The behaviour above was proven against an installed DSH before it was coded:
exit 2 refuses at `PreToolUse` and blocks at `PostToolUse` with the stderr
text as the visible reason; a `Stop` hook's steering comes back exactly once
per exit-2; `SessionStart` context lands as the first user message of the
new agent; the block feedback and the injected context need no factory
import. The JSON verdicts are only honoured with a matching
`hookEventName`, and a hook call costs roughly half a second — matcher and
scope are the tuning knobs. Per-hook details: the package README in
`dsh-loomux-hooks/`.
