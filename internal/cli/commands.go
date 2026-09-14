package cli

// commands is the whole command table. Each stage-1a task adds its line here.
var commands = map[string]command{
	"check":     checkCommand,
	"dev":       devCommand,
	"doctor":    statusCommand,
	"explain":   statusCommand,
	"hook":      hookCommand,
	"lint":      lintCommand,
	"status":    statusCommand,
	"wiki-gate": wikiGateCommand,
	"worktree":  worktreeCommand,
}
