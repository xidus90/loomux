package cli

// commands is the whole command table. Each stage-1a task adds its line here.
var commands = map[string]command{
	"brain":     brainCommand,
	"check":     checkCommand,
	"dev":       devCommand,
	"doctor":    statusCommand,
	"explain":   statusCommand,
	"graph":     graphCommand,
	"hook":      hookCommand,
	"lint":      lintCommand,
	"mcp":       mcpCommand,
	"serve":     serveCommand,
	"status":    statusCommand,
	"wiki-gate": wikiGateCommand,
	"worktree":  worktreeCommand,
}
