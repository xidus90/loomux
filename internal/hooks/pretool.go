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
	// The barrier reads the bytes itself; a payload that is no object
	// arrives as nil.
	var object map[string]any
	_ = json.Unmarshal(data, &object)
	reasons, err := policyReasons(object, root)
	if err != nil {
		return guard.Refuse(stdout, stderr, fmt.Sprintf("loomux cannot read its policy, so it refuses: %v", err))
	}
	if len(reasons) > 0 {
		return guard.Refuse(stdout, stderr, "loomux policy refused this tool call:\n  - "+strings.Join(reasons, "\n  - "))
	}
	return guard.Run(bytes.NewReader(data), stdout, stderr, stateDir)
}

// policyReasons judges the call against the policy. A payload that did not
// decode arrives as nil and yields no reasons: refusing it is the barrier's
// job, with the barrier's wording, one step later. An object that names no
// tool is refused here, before the policy file is read: no rule can be
// matched against it, and a call nobody judged does not pass.
func policyReasons(object map[string]any, root string) ([]string, error) {
	if object == nil {
		return nil, nil
	}
	tool, input := guard.Call(object)
	if tool == "" {
		return []string{"loomux found no tool name in this call, so it cannot judge it and refuses"}, nil
	}
	policy, err := readPolicy(root)
	if err != nil {
		return nil, err
	}
	return checkTool(root, tool, input, policy), nil
}
