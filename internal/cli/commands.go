package cli

// commands is the whole command table. Each stage-1a task adds its line here.
var commands = map[string]command{
	"area":      areaCommand,
	"brain":     brainCommand,
	"case":      caseCommand,
	"cases":     casesCommand,
	"check":     checkCommand,
	"dev":       devCommand,
	"doctor":    statusCommand,
	"embed":     embedCommand,
	"explain":   statusCommand,
	"graph":     graphCommand,
	"hook":      hookCommand,
	"lint":      lintCommand,
	"mcp":       mcpCommand,
	"reconcile": reconcileCommand,
	"reindex":   reindexCommand,
	"serve":     serveCommand,
	"status":    statusCommand,
	"wiki-gate": wikiGateCommand,
	"worktree":  worktreeCommand,
}
