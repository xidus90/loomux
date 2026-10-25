package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xidus90/loomux/internal/flow/model"
)

type claude struct{ o Options }

// NewClaude returns the adapter for the claude CLI, which also serves any
// Anthropic-compatible endpoint named by Request.BaseURL.
func NewClaude(o Options) model.Model { return claude{o} }

// claudeOutput is the part of `claude -p --output-format json` that is read.
type claudeOutput struct {
	IsError          bool            `json:"is_error"`
	Result           string          `json:"result"`
	StructuredOutput json.RawMessage `json:"structured_output"`
	Usage            struct {
		Input  int `json:"input_tokens"`
		Output int `json:"output_tokens"`
	} `json:"usage"`
	ModelUsage        map[string]json.RawMessage `json:"modelUsage"`
	PermissionDenials []denial                   `json:"permission_denials"`
}

func (c claude) Ask(ctx context.Context, r model.Request) (model.Reply, error) {
	// Tool lists are variadic, so the prompt travels on stdin. MCP tools are
	// only allowed, built-ins are the available set; none asked means none.
	var builtin, mcp []string
	for _, t := range r.Tools {
		if strings.HasPrefix(t, "mcp__") {
			mcp = append(mcp, t)
		} else {
			builtin = append(builtin, t)
		}
	}
	args := []string{"-p", "--output-format", "json", "--json-schema", Schema(r.Reply)}
	for _, f := range [][2]string{{"--model", r.Model}, {"--effort", r.Effort}, {"--agent", r.Agent}} {
		if f[1] != "" {
			args = append(args, f[0], f[1])
		}
	}
	args = append(args, "--tools", strings.Join(builtin, ","))
	if len(mcp) > 0 {
		args = append(args, "--allowedTools", strings.Join(mcp, ","))
	}
	cmd := Command{Name: "claude", Args: args, Stdin: []byte(r.Prompt)}
	if r.BaseURL != "" {
		cmd.Env = []string{"ANTHROPIC_BASE_URL=" + r.BaseURL}
	}
	stdout, err := run(ctx, c.o, cmd)
	if err != nil {
		return model.Reply{}, err
	}
	var out claudeOutput
	if err := json.Unmarshal(stdout, &out); err != nil {
		return model.Reply{}, fmt.Errorf("claude output is not JSON: %w", err)
	}
	if out.IsError {
		return model.Reply{}, fmt.Errorf("claude reported an error: %s", out.Result)
	}
	if len(out.StructuredOutput) == 0 {
		if err := deniedError("claude", "claude's settings", out.PermissionDenials); err != nil {
			return model.Reply{}, err
		}
		return model.Reply{}, fmt.Errorf("claude returned no structured_output")
	}
	fields, err := Fields(out.StructuredOutput, r.Reply)
	if err != nil {
		return model.Reply{}, err
	}
	reply := model.Reply{Fields: fields, Tokens: out.Usage.Input + out.Usage.Output}
	if len(out.ModelUsage) == 1 { // several models: the answering one is unknown
		for name := range out.ModelUsage {
			reply.Model = name
		}
	}
	return reply, nil
}
