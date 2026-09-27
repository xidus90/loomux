package model

import (
	"fmt"

	"github.com/xidus90/loomux/internal/config"
)

// DefaultProvider answers a role nothing binds and no default covers.
const DefaultProvider = "claude"

// Resolved is the model a node runs on, and how it was found.
type Resolved struct {
	Role      string // "" when neither the node nor the flow names one
	RoleFrom  string // "node", "flow" or ""
	Name      string // the model's name under [agent.models]; "" when nothing named one
	ModelFrom string // "binding", "default" or "cli-default"
	Provider  string
	Model     string // "" is the provider CLI's own default
}

// Resolve is the role chain: the node's role, else the flow's; that role's
// binding in [agent.roles], else [agent] default; else the claude CLI's own
// default. The config reader refuses a binding to a model it does not know,
// so only an Agent built by hand reaches the error.
func Resolve(agent config.Agent, nodeRole, flowRole string) (Resolved, error) {
	var r Resolved
	switch {
	case nodeRole != "":
		r.Role, r.RoleFrom = nodeRole, "node"
	case flowRole != "":
		r.Role, r.RoleFrom = flowRole, "flow"
	}
	name, from := "", "cli-default"
	if bound, ok := agent.Roles[r.Role]; ok && r.Role != "" {
		name, from = bound, "binding"
	} else if agent.Default != "" {
		name, from = agent.Default, "default"
	}
	r.ModelFrom = from
	if name == "" {
		r.Provider = DefaultProvider
		return r, nil
	}
	spec, ok := agent.Models[name]
	if !ok {
		return Resolved{}, fmt.Errorf("model %q is not under [agent.models]", name)
	}
	r.Name, r.Provider, r.Model = name, spec.Provider, spec.Model
	return r, nil
}

// Label is how the journal records the model.
func (r Resolved) Label() string {
	if r.Model == "" {
		return r.Provider + ":cli-default"
	}
	return r.Provider + ":" + r.Model
}

// Via says in one phrase where the model came from, as `loomux flow show`
// prints it next to the node.
func (r Resolved) Via() string {
	role := "no role"
	if r.Role != "" {
		role = "role " + r.Role + " (" + r.RoleFrom + ")"
	}
	switch r.ModelFrom {
	case "binding":
		return role + ", bound to " + r.Name
	case "default":
		return role + ", [agent] default " + r.Name
	default:
		return role + ", the CLI's own default"
	}
}
