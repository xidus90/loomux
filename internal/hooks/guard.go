// Package hooks holds what loomux runs from a harness lifecycle event: the
// post-tool-use lanes, the session start, the status report and the worktree
// mirror. The pre-tool-use barrier joins them in Task 10.
package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	ExitOK       = 0
	ExitInternal = 1
	ExitDenied   = 2
)

type PolicyFile struct {
	Policy struct {
		Paths struct {
			Rules []PathRule `toml:"rules"`
		} `toml:"paths"`
		Commands struct {
			Rules []CommandRule `toml:"rules"`
		} `toml:"commands"`
	} `toml:"policy"`
}

type PathRule struct {
	Match  string `toml:"match"`
	Reason string `toml:"reason"`
}

type CommandRule struct {
	Regex  string `toml:"regex"`
	Reason string `toml:"reason"`
}

type HookPayload struct {
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
}

// Built-in rules that protect secrets, stop gate controls, and lock files.
var builtinPathRules = []PathRule{
	{Match: ".env", Reason: "secrets are not written by an agent"},
	{Match: ".env.*", Reason: "secrets are not written by an agent"},
	{Match: "*.pem", Reason: "secrets are not written by an agent"},
	{Match: "*.key", Reason: "secrets are not written by an agent"},
	{Match: "id_rsa*", Reason: "secrets are not written by an agent"},
	{Match: "*.p12", Reason: "secrets are not written by an agent"},
	{Match: ".npmrc", Reason: "secrets are not written by an agent"},
	{Match: ".pypirc", Reason: "secrets are not written by an agent"},
	{Match: "credentials.json", Reason: "secrets are not written by an agent"},
	{Match: ".aws/**", Reason: "secrets are not written by an agent"},
	{Match: ".claude/.no-verify", Reason: "the stop gate's own controls are not written by the party it gates"},
	{Match: ".ultraloom/hooks/**", Reason: "the stop gate's own controls are not written by the party it gates"},
	{Match: "uv.lock", Reason: "lock files are written by their package manager, not by hand"},
	{Match: "poetry.lock", Reason: "lock files are written by their package manager, not by hand"},
	{Match: "package-lock.json", Reason: "lock files are written by their package manager, not by hand"},
	{Match: "pnpm-lock.yaml", Reason: "lock files are written by their package manager, not by hand"},
	{Match: "yarn.lock", Reason: "lock files are written by their package manager, not by hand"},
	{Match: "Cargo.lock", Reason: "lock files are written by their package manager, not by hand"},
	{Match: "go.sum", Reason: "lock files are written by their package manager, not by hand"},
}

var builtinCommandRules = []CommandRule{
	{Regex: `(^|\s)git\s+push(\s|$)`, Reason: "Whether commits reach the remote is a human's decision."},
}

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

func loadPolicy(root string) (PolicyFile, error) {
	policyPath := filepath.Join(root, ".ultraloom", "policy.toml")
	var p PolicyFile
	data, err := os.ReadFile(policyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return p, err
	}
	if err := toml.Unmarshal(data, &p); err != nil {
		return p, fmt.Errorf("parsing %s: %w", policyPath, err)
	}
	return p, nil
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

func checkTool(root string, payload HookPayload, policy PolicyFile) []string {
	var reasons []string
	tool := payload.ToolName
	input := payload.ToolInput

	// File tools
	var path string
	switch tool {
	case "Write", "Edit", "MultiEdit":
		if p, ok := input["file_path"].(string); ok {
			path = p
		}
	case "NotebookEdit":
		if p, ok := input["notebook_path"].(string); ok {
			path = p
		}
	}

	if path != "" {
		rel := relativePath(path, root)
		// Check built-in path rules
		for _, rule := range builtinPathRules {
			if matchGlob(rule.Match, rel) {
				reasons = append(reasons, rule.Reason)
			}
		}
		// Check configured path rules
		for _, rule := range policy.Policy.Paths.Rules {
			if matchGlob(rule.Match, rel) {
				reasons = append(reasons, rule.Reason)
			}
		}
	}

	// Command tools
	if tool == "Bash" || tool == "PowerShell" {
		if cmd, ok := input["command"].(string); ok {
			// Check built-in command rules
			for _, rule := range builtinCommandRules {
				matched, _ := regexp.MatchString(rule.Regex, cmd)
				if matched {
					reasons = append(reasons, rule.Reason)
				}
			}
			// Check configured command rules
			for _, rule := range policy.Policy.Commands.Rules {
				matched, _ := regexp.MatchString(rule.Regex, cmd)
				if matched {
					reasons = append(reasons, rule.Reason)
				}
			}
		}
	}

	return reasons
}

func runGuard(stdin io.Reader, stderr io.Writer, root string) int {
	data, err := io.ReadAll(stdin)
	if err != nil || len(data) == 0 {
		fmt.Fprintf(stderr, "loomux: failed to read hook payload: %v\n", err)
		return ExitInternal
	}

	var payload HookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		fmt.Fprintf(stderr, "loomux: unreadable hook payload: %v\n", err)
		return ExitInternal
	}
	if payload.ToolName == "" {
		fmt.Fprintf(stderr, "loomux: tool_name is required in payload\n")
		return ExitInternal
	}

	policy, err := loadPolicy(root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux: %v\n", err)
		return ExitDenied
	}

	reasons := checkTool(root, payload, policy)
	if len(reasons) > 0 {
		fmt.Fprintf(stderr, "ultraloom policy refused this %s:\n", payload.ToolName)
		for _, r := range reasons {
			fmt.Fprintf(stderr, "  - %s\n", r)
		}
		return ExitDenied
	}

	return ExitOK
}
