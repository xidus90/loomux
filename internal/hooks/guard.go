// Package hooks holds what loomux runs from a harness lifecycle event: the
// pre-tool-use guard that answers with the project's policy and the global
// write barrier, the post-tool-use lanes, the session start, the status report
// and the worktree mirror.
package hooks

import (
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/config"
)

const (
	ExitOK       = 0
	ExitInternal = 1
	ExitDenied   = 2
)

type HookPayload struct {
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
}

// Built-in rules that protect secrets, stop gate controls, and lock files.
var builtinPathRules = []config.PathRule{
	{Match: []string{".env"}, Reason: "secrets are not written by an agent"},
	{Match: []string{".env.*"}, Reason: "secrets are not written by an agent"},
	{Match: []string{"*.pem"}, Reason: "secrets are not written by an agent"},
	{Match: []string{"*.key"}, Reason: "secrets are not written by an agent"},
	{Match: []string{"id_rsa*"}, Reason: "secrets are not written by an agent"},
	{Match: []string{"*.p12"}, Reason: "secrets are not written by an agent"},
	{Match: []string{".npmrc"}, Reason: "secrets are not written by an agent"},
	{Match: []string{".pypirc"}, Reason: "secrets are not written by an agent"},
	{Match: []string{"credentials.json"}, Reason: "secrets are not written by an agent"},
	{Match: []string{".aws/**"}, Reason: "secrets are not written by an agent"},
	{Match: []string{".claude/.no-verify"}, Reason: "the stop gate's own controls are not written by the party it gates"},
	{Match: []string{".loomux/state/hooks/**"}, Reason: "the stop gate's own controls are not written by the party it gates"},
	{Match: []string{"uv.lock"}, Reason: "lock files are written by their package manager, not by hand"},
	{Match: []string{"poetry.lock"}, Reason: "lock files are written by their package manager, not by hand"},
	{Match: []string{"package-lock.json"}, Reason: "lock files are written by their package manager, not by hand"},
	{Match: []string{"pnpm-lock.yaml"}, Reason: "lock files are written by their package manager, not by hand"},
	{Match: []string{"yarn.lock"}, Reason: "lock files are written by their package manager, not by hand"},
	{Match: []string{"Cargo.lock"}, Reason: "lock files are written by their package manager, not by hand"},
	{Match: []string{"go.sum"}, Reason: "lock files are written by their package manager, not by hand"},
}

// builtinCommands compiles on first use rather than at load: this binary hangs
// on every tool call, and a package variable would pay for the expression in
// runs that never look at a command line.
var builtinCommands = sync.OnceValue(func() []config.CommandRule {
	return []config.CommandRule{{
		Regex:  regexp.MustCompile(`(^|\s)git\s+push(\s|$)`),
		Source: `(^|\s)git\s+push(\s|$)`,
		Reason: "Whether commits reach the remote is a human's decision.",
	}}
})

// commandTools are the tools whose "command" argument is a shell line.
var commandTools = map[string]bool{"Bash": true, "PowerShell": true}

// matchGlob matches a slash-separated path against a glob pattern supporting `**`.
func matchGlob(pattern, path string) bool {
	if pattern == path {
		return true
	}
	// Direct wildcard suffix like .aws/**
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	if strings.Contains(pattern, "/") {
		matched, _ := filepath.Match(pattern, path)
		return matched
	}
	// For patterns without slashes (e.g. *.pem or .env.* or uv.lock)
	// they match either at the root or base name depending on rule semantics
	base := filepath.Base(path)
	matched, _ := filepath.Match(pattern, base)
	return matched
}

func relativePath(raw, root string) string {
	raw = filepath.Clean(raw)
	if !filepath.IsAbs(raw) {
		return filepath.ToSlash(raw)
	}
	rel, err := filepath.Rel(root, raw)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(raw)
	}
	return filepath.ToSlash(rel)
}

// checkTool judges one tool call against the built-in rules and the project's
// own, and answers every reason it found: a caller that wants to say why it
// refuses needs all of them, not the first.
func checkTool(root, tool string, input map[string]any, policy config.Policy) []string {
	var reasons []string
	if guard.IsWritingTool(tool) {
		for _, target := range guard.WriteTargets(input) {
			rel := relativePath(target, root)
			for _, rule := range append(builtinPathRules, policy.Paths...) {
				for _, glob := range rule.Match {
					if matchGlob(glob, rel) {
						reasons = append(reasons, rule.Reason)
						break
					}
				}
			}
		}
	}
	if commandTools[tool] {
		if line, ok := input["command"].(string); ok {
			for _, rule := range append(builtinCommands(), policy.Commands...) {
				if rule.Regex.MatchString(line) {
					reasons = append(reasons, rule.Reason)
				}
			}
		}
	}
	return reasons
}
