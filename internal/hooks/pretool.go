package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/config"
)

var readPolicy = config.ReadPolicy

// PreToolUse answers one PreToolUse hook with 0 or 2, never 1: the project's
// policy first, then the global write barrier. Everything that goes wrong on
// the way refuses, because a host reads 1 as "carry on".
func PreToolUse(stdin io.Reader, stdout, stderr io.Writer, root, stateDir string) (code int) {
	defer func() {
		if broke := recover(); broke != nil {
			code = guard.Refuse(stdout, stderr, fmt.Sprintf("the loomux guard broke down, so it refuses: %v", broke))
		}
	}()
	data, err := io.ReadAll(stdin)
	if err != nil {
		return guard.Refuse(stdout, stderr, fmt.Sprintf("loomux cannot read the hook payload, so it refuses: %v", err))
	}
	reasons, err := policyReasons(data, root)
	if err != nil {
		return guard.Refuse(stdout, stderr, fmt.Sprintf("loomux cannot read its policy, so it refuses: %v", err))
	}
	if len(reasons) > 0 {
		return guard.Refuse(stdout, stderr, "loomux policy refused this tool call:\n  - "+strings.Join(reasons, "\n  - "))
	}
	return guard.Run(bytes.NewReader(data), stdout, stderr, stateDir)
}

// policyReasons judges the call against the policy. A payload it cannot
// decode yields no reasons: refusing it is the barrier's job, with the
// barrier's wording, one step later.
func policyReasons(data []byte, root string) ([]string, error) {
	var object map[string]any
	if json.Unmarshal(data, &object) != nil {
		return nil, nil
	}
	tool, input := guard.Call(object)
	if tool == "" {
		return nil, nil
	}
	policy, err := readPolicy(root)
	if err != nil {
		return nil, err
	}
	return checkTool(root, tool, input, policy), nil
}
