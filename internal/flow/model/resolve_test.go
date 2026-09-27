package model

import (
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

func TestResolveWalksTheRoleChain(t *testing.T) {
	agent := config.Agent{
		Default: "writer",
		Models: map[string]config.ModelSpec{
			"writer": {Provider: "claude", Model: "claude-opus-5-5"},
			"gemini": {Provider: "agy"},
		},
		Roles: map[string]string{"reviewer": "gemini"},
	}
	for name, tc := range map[string]struct {
		agent          config.Agent
		node, flowRole string
		want           Resolved
		label, via     string
	}{
		"node role bound":   {agent, "reviewer", "writer_role", Resolved{Role: "reviewer", RoleFrom: "node", Name: "gemini", ModelFrom: "binding", Provider: "agy"}, "agy:cli-default", "role reviewer (node), bound to gemini"},
		"flow role unbound": {agent, "", "planner", Resolved{Role: "planner", RoleFrom: "flow", Name: "writer", ModelFrom: "default", Provider: "claude", Model: "claude-opus-5-5"}, "claude:claude-opus-5-5", "role planner (flow), [agent] default writer"},
		"no role, default":  {agent, "", "", Resolved{Name: "writer", ModelFrom: "default", Provider: "claude", Model: "claude-opus-5-5"}, "claude:claude-opus-5-5", "no role, [agent] default writer"},
		"nothing at all":    {config.Agent{}, "reviewer", "", Resolved{Role: "reviewer", RoleFrom: "node", ModelFrom: "cli-default", Provider: "claude"}, "claude:cli-default", "role reviewer (node), the CLI's own default"},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := Resolve(tc.agent, tc.node, tc.flowRole)
			if err != nil || got != tc.want || got.Label() != tc.label || got.Via() != tc.via {
				t.Fatalf("got %+v (%q, %q), %v", got, got.Label(), got.Via(), err)
			}
		})
	}
}

func TestResolveRefusesAHandBuiltBindingToNothing(t *testing.T) {
	_, err := Resolve(config.Agent{Roles: map[string]string{"r": "x"}}, "r", "")
	if err == nil || err.Error() != `model "x" is not under [agent.models]` {
		t.Fatalf("err = %v", err)
	}
}
