@AGENTS.md

# For Claude Code only

- Subagents never push; after a subagent run read `git log -1 --format='%an <%ae>'`.
- The hooks in `.claude/settings.json` call `bin/loomux.exe`, which the
  pre-commit gate rebuilds. If session-start warns that the binary is older
  than a Go source under `cmd/` or `internal/` (or `go.mod`/`go.sum`), rebuild
  before trusting a refusal.
