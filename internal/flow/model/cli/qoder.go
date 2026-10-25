package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/xidus90/loomux/internal/flow/model"
)

type qoder struct{ o Options }

// NewQoder returns the adapter for the Qoder CLI (qodercli). qodercli has no
// schema flag, so the prompt carries the schema and a bad reply gets one retry.
func NewQoder(o Options) model.Model { return qoder{o} }

// qoderToolNames maps profile tool names to qodercli's built-in names. The
// names are equal, measured with qodercli 1.1.78: --tools restricts hard and
// the CLI silently accepts unknown names.
func qoderToolNames() map[string]string {
	return map[string]string{
		"Read": "Read", "Glob": "Glob", "Grep": "Grep",
		"Edit": "Edit", "Write": "Write", "Bash": "Bash",
	}
}

// qoderTools translates a profile's tools; those without a qodercli
// equivalent are returned in dropped so the caller can warn.
func qoderTools(tools []string) (kept, dropped []string) {
	names := qoderToolNames()
	for _, t := range tools {
		if n, ok := names[t]; ok {
			kept = append(kept, n)
		} else {
			dropped = append(dropped, t)
		}
	}
	return kept, dropped
}

// qoderOutput is the part of `qodercli -p -o json` that is read.
type qoderOutput struct {
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
	Usage   struct {
		Input  int `json:"input_tokens"`
		Output int `json:"output_tokens"`
	} `json:"usage"`
	ModelUsage        map[string]json.RawMessage `json:"modelUsage"`
	PermissionDenials []denial                   `json:"permission_denials"`
}

// badReply marks a model answer that does not fit the schema: the only
// failure worth asking again about.
type badReply struct{ error }

func (q qoder) Ask(ctx context.Context, r model.Request) (model.Reply, error) {
	// The prompt travels on stdin, so --tools may be the last flag.
	args := []string{"-p", "-o", "json"}
	for _, f := range [][2]string{{"-m", r.Model}, {"--reasoning-effort", r.Effort}, {"--agent", r.Agent}} {
		if f[1] != "" {
			args = append(args, f[0], f[1])
		}
	}
	kept, _ := qoderTools(r.Tools)
	if len(kept) == 0 && len(r.Tools) > 0 {
		return model.Reply{}, errors.New("qoder has none of the tools the profile allows")
	}
	args = append(args, "--tools", strings.Join(kept, ","))
	prompt := r.Prompt + "\n\nAnswer with one JSON object and nothing else, matching this JSON Schema: " + Schema(r.Reply)
	var lastErr error
	for range 2 {
		reply, err := q.once(ctx, Command{Name: "qodercli", Args: args, Stdin: []byte(prompt)}, r)
		if err == nil {
			return reply, nil
		}
		var bad badReply
		if !errors.As(err, &bad) {
			return model.Reply{}, err
		}
		lastErr = err
		prompt = r.Prompt + "\n\nYour previous answer was rejected: " + err.Error() +
			"\nAnswer with one JSON object and nothing else, matching this JSON Schema: " + Schema(r.Reply)
	}
	return model.Reply{}, lastErr
}

func (q qoder) once(ctx context.Context, c Command, r model.Request) (model.Reply, error) {
	stdout, err := run(ctx, q.o, c)
	if err != nil {
		return model.Reply{}, err
	}
	var out qoderOutput
	if err := json.Unmarshal(stdout, &out); err != nil {
		return model.Reply{}, fmt.Errorf("qodercli output is not JSON: %w", err)
	}
	if out.IsError {
		return model.Reply{}, fmt.Errorf("qodercli reported an error: %s", out.Result)
	}
	text := strings.TrimSpace(out.Result)
	text = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(text, "```json"), "```"), "```"))
	fields, err := Fields([]byte(text), r.Reply)
	if err != nil {
		if denied := deniedError("qoder", "qodercli's settings", out.PermissionDenials); denied != nil {
			return model.Reply{}, denied
		}
		return model.Reply{}, badReply{err}
	}
	reply := model.Reply{Fields: fields, Tokens: out.Usage.Input + out.Usage.Output}
	if len(out.ModelUsage) == 1 { // several models: the answering one is unknown
		for name := range out.ModelUsage {
			reply.Model = name
		}
	}
	return reply, nil
}
