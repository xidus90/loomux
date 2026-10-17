package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

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
	// The switch is read from the registry at most once, and only by a call
	// that needs it; an area it cannot find or read keeps it off. It opens
	// the manifest only where no link leads it to another file, and the
	// note says so only then.
	open := sync.OnceValue(func() bool {
		_, on, err := guard.AreaAt(root, stateDir)
		return err == nil && on && guard.ManifestUnlinked(root)
	})
	note := func() string {
		if open() {
			return llmConfigNote
		}
		return ""
	}
	// The hook is installed for one root, while a shell keeps whatever
	// directory it moved to; the host says which in cwd, and a call that
	// does not say counts as standing elsewhere.
	cwd, _ := object["cwd"].(string)
	here := sync.OnceValue(func() bool { return guard.AtRoot(cwd, root) })
	reasons, err := policyReasons(object, root, open, here)
	if err != nil {
		return guard.Refuse(stdout, stderr, fmt.Sprintf("loomux cannot read its policy, so it refuses: %v", err))
	}
	if len(reasons) > 0 {
		reason := "loomux policy refused this tool call:\n  - " + strings.Join(reasons, "\n  - ")
		if n := note(); n != "" {
			reason += "\n\n" + n
		}
		return guard.Refuse(stdout, stderr, reason)
	}
	return guard.Run(bytes.NewReader(data), stdout, stderr, stateDir, root, note)
}

// llmConfigNote ends every refusal in a project whose llm_config switch
// opens its manifest, so that a human reading it knows the agent may change
// the limits.
const llmConfigNote = "llm_config is on for this project: agents may edit .loomux/config.toml (turn it off with loomux config llm off)"

// policyReasons judges the call against the policy. A payload that did not
// decode arrives as nil and yields no reasons: refusing it is the barrier's
// job, with the barrier's wording, one step later. An object that names no
// tool is refused here, before the policy file is read: no rule can be
// matched against it, and a call nobody judged does not pass.
func policyReasons(object map[string]any, root string, open, here func() bool) ([]string, error) {
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
	return checkToolOpen(root, tool, input, policy, open, here), nil
}
