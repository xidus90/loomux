package cli

// commands is the whole command table. Each stage-1a task adds its line here.
var commands = map[string]command{
	"approve":     approveCommand,
	"area":        areaCommand,
	"brain":       brainCommand,
	"case":        caseCommand,
	"cases":       casesCommand,
	"check":       checkCommand,
	"config":      configCommand,
	"dev":         devCommand,
	"doctor":      statusCommand,
	"embed":       embedCommand,
	"explain":     statusCommand,
	"graph":       graphCommand,
	"hook":        hookCommand,
	"lint":        lintCommand,
	"mcp":         mcpCommand,
	"merge-hook":  mergeHookCommand,
	"reconcile":   reconcileCommand,
	"reindex":     reindexCommand,
	"self-update": selfUpdateCommand,
	"serve":       serveCommand,
	"status":      statusCommand,
	"wiki":        wikiCommand,
	"wiki-gate":   wikiGateCommand,
	"worktree":    worktreeCommand,
}
