package cli

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xidus90/loomux/internal/flow/model"
)

type agy struct{ o Options }

// NewAgy returns the adapter for the Antigravity CLI (agy). agy has no tool
// list flag, so the tool profile only picks the execution mode.
func NewAgy(o Options) model.Model { return agy{o} }

// agyMaxPrompt bounds the prompt: agy takes it as a command-line argument and
// Windows limits the whole command line to about 32000 characters.
const agyMaxPrompt = 30000

// agyOutput is the part of `agy -p --output-format json` that is read.
type agyOutput struct {
	Status           string          `json:"status"`
	Response         string          `json:"response"`
	StructuredOutput json.RawMessage `json:"structured_output"`
	Denied           []struct {
		DisplayName string `json:"display_name"`
		Action      string `json:"action"`
	} `json:"denied_actions"`
	Usage struct {
		Input  int `json:"input_tokens"`
		Output int `json:"output_tokens"`
	} `json:"usage"`
}

func (a agy) Ask(ctx context.Context, r model.Request) (model.Reply, error) {
	if len(r.Prompt) > agyMaxPrompt {
		return model.Reply{}, fmt.Errorf("agy takes the prompt as an argument: %d bytes exceed the limit of %d", len(r.Prompt), agyMaxPrompt)
	}
	mode := "accept-edits"
	if r.Profile == "read_only" {
		mode = "plan"
	}
	args := []string{"--output-format", "json", "--json-schema", Schema(r.Reply)}
	for _, f := range [][2]string{{"--model", r.Model}, {"--effort", r.Effort}, {"--agent", r.Agent}} {
		if f[1] != "" {
			args = append(args, f[0], f[1])
		}
	}
	if a.o.Timeout > 0 {
		args = append(args, "--print-timeout", a.o.Timeout.String())
	}
	// -p takes the next word as the prompt, so it comes last.
	args = append(args, "--mode", mode, "-p", r.Prompt)
	stdout, err := run(ctx, a.o, Command{Name: "agy", Args: args})
	if err != nil {
		return model.Reply{}, err
	}
	var out agyOutput
	if err := json.Unmarshal(stdout, &out); err != nil {
		return model.Reply{}, fmt.Errorf("agy output is not JSON: %w", err)
	}
	if out.Status != "SUCCESS" {
		return model.Reply{}, fmt.Errorf("agy reported %s: %s", out.Status, out.Response)
	}
	if len(out.StructuredOutput) == 0 || string(out.StructuredOutput) == "null" {
		if len(out.Denied) > 0 {
			names := make([]string, len(out.Denied))
			for i, d := range out.Denied {
				names[i] = cmp.Or(d.DisplayName, d.Action)
			}
			return model.Reply{}, fmt.Errorf("agy denied %s in headless mode and returned no reply; allow it under permissions.allow in agy's settings.json", strings.Join(names, ", "))
		}
		return model.Reply{}, fmt.Errorf("agy returned no structured_output")
	}
	fields, err := Fields(out.StructuredOutput, r.Reply)
	if err != nil {
		return model.Reply{}, err
	}
	return model.Reply{Fields: fields, Tokens: out.Usage.Input + out.Usage.Output}, nil
}
