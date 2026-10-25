package config

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// ModelSpec is one entry under [agent.models]: who answers, and with which of
// its models. An empty Model is the provider CLI's own default.
type ModelSpec struct {
	Provider string
	Model    string
	Agent    string // the provider's own agent definition; "" when unset
	Effort   string // reasoning effort; "" when unset
	BaseURL  string // endpoint override, provider claude only; "" when unset
}

// DefaultAgentTimeout bounds one agent node when [agent] sets no timeout.
const DefaultAgentTimeout = 30 * time.Minute

// Providers are the CLIs a model can be served by, sorted.
func Providers() []string { return []string{"agy", "claude", "qoder"} }

// Agent is the [agent] table: the models a flow's roles can be bound to.
type Agent struct {
	Default    string               // model name for every role without a binding; "" when unset
	MCPServers []string             // the servers the mcp tool profile opens
	Models     map[string]ModelSpec // by model name
	Roles      map[string]string    // role name to model name
	Timeout    time.Duration        // bound of one agent node; 0 switches it off
}

// AgentKeys are the keys [agent] knows, sorted. settings arrives with the
// model adapters and is unknown until then.
func AgentKeys() []string { return []string{"default", "mcp_servers", "models", "roles", "timeout"} }

// ModelSpecKeys are the keys one model under [agent.models] knows, sorted.
func ModelSpecKeys() []string { return []string{"agent", "base_url", "effort", "model", "provider"} }

// ReadAgent reads [agent] of the project at root. A project without the file
// or the table has an empty Agent: every role then runs on the claude CLI's
// own default.
func ReadAgent(root string) (Agent, error) {
	path := ManifestPath(root)
	doc, err := manifestDocument(path)
	if err != nil || doc == nil {
		return Agent{Timeout: DefaultAgentTimeout}, err
	}
	return ParseAgent(path, doc)
}

// manifestDocument decodes the manifest at path. An absent file is (nil, nil).
func manifestDocument(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	doc := map[string]any{}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: not valid TOML: %w", path, err)
	}
	return doc, nil
}

// ParseAgent reads [agent] from a decoded document and reports every finding
// at once. A binding or a default that names no model is refused here, so a
// flow never learns at its first paid node that its role runs nowhere.
func ParseAgent(path string, doc map[string]any) (Agent, error) {
	raw, present := doc["agent"]
	if !present {
		return Agent{Timeout: DefaultAgentTimeout}, nil
	}
	table, ok := raw.(map[string]any)
	if !ok {
		return Agent{}, fmt.Errorf("%s: [agent] must be a table, found %s", path, tomlType(raw))
	}
	agent := Agent{Timeout: DefaultAgentTimeout}
	var findings []string
	for _, key := range slices.Sorted(maps.Keys(table)) {
		if !slices.Contains(AgentKeys(), key) {
			findings = append(findings, fmt.Sprintf("[agent] does not know %q; known: %s", key, strings.Join(AgentKeys(), ", ")))
		}
	}
	if value, present := table["models"]; present {
		agent.Models = parseModels(value, &findings)
	}
	if value, present := table["default"]; present {
		agent.Default = modelName("[agent] default", value, agent.Models, &findings)
	}
	if value, present := table["roles"]; present {
		agent.Roles = parseRoles(value, agent.Models, &findings)
	}
	if value, present := table["mcp_servers"]; present {
		agent.MCPServers = parseServers(value, &findings)
	}
	if value, present := table["timeout"]; present {
		agent.Timeout = parseTimeout(value, &findings)
	}
	if len(findings) > 0 {
		return Agent{}, fmt.Errorf("%s: %s", path, strings.Join(findings, "; "))
	}
	return agent, nil
}

func parseModels(value any, findings *[]string) map[string]ModelSpec {
	table, ok := value.(map[string]any)
	if !ok {
		*findings = append(*findings, fmt.Sprintf("[agent.models] must be a table of models, found %s", tomlType(value)))
		return nil
	}
	models := make(map[string]ModelSpec, len(table))
	for _, name := range slices.Sorted(maps.Keys(table)) {
		where := "[agent.models." + name + "]"
		if !IsIdentifier(name) {
			*findings = append(*findings, fmt.Sprintf("%s: %q is not a model name; a name is %s", where, name, IdentifierRule))
		}
		entry, ok := table[name].(map[string]any)
		if !ok {
			*findings = append(*findings, fmt.Sprintf("%s must be a table, found %s", where, tomlType(table[name])))
			continue
		}
		for _, key := range slices.Sorted(maps.Keys(entry)) {
			if !slices.Contains(ModelSpecKeys(), key) {
				*findings = append(*findings, fmt.Sprintf("%s does not know %q; known: %s", where, key, strings.Join(ModelSpecKeys(), ", ")))
			}
		}
		provider, _ := entry["provider"].(string)
		if provider == "" {
			*findings = append(*findings, where+" needs provider as a non-empty string")
		}
		if provider != "" && !slices.Contains(Providers(), provider) {
			*findings = append(*findings, fmt.Sprintf("%s provider %q is unknown; known: %s", where, provider, strings.Join(Providers(), ", ")))
		}
		spec := ModelSpec{Provider: provider}
		for _, field := range []struct {
			key string
			dst *string
		}{{"model", &spec.Model}, {"agent", &spec.Agent}, {"effort", &spec.Effort}, {"base_url", &spec.BaseURL}} {
			text, isText := entry[field.key].(string)
			if _, has := entry[field.key]; has && (!isText || text == "") {
				*findings = append(*findings, where+" "+field.key+" must be a non-empty string")
			}
			*field.dst = text
		}
		if spec.BaseURL != "" && provider != "claude" {
			*findings = append(*findings, where+" base_url only works with provider claude")
		}
		models[name] = spec
	}
	return models
}

// modelName reads a value that has to name a model under [agent.models].
func modelName(where string, value any, models map[string]ModelSpec, findings *[]string) string {
	name, ok := value.(string)
	if !ok || name == "" {
		*findings = append(*findings, where+" must be a non-empty string")
		return ""
	}
	if _, known := models[name]; !known {
		*findings = append(*findings, fmt.Sprintf("%s names %q, which is not under [agent.models]; known: %s", where, name, knownModels(models)))
	}
	return name
}

func knownModels(models map[string]ModelSpec) string {
	if len(models) == 0 {
		return "none"
	}
	return strings.Join(slices.Sorted(maps.Keys(models)), ", ")
}

func parseRoles(value any, models map[string]ModelSpec, findings *[]string) map[string]string {
	table, ok := value.(map[string]any)
	if !ok {
		*findings = append(*findings, fmt.Sprintf("[agent.roles] must be a table of role = model, found %s", tomlType(value)))
		return nil
	}
	roles := make(map[string]string, len(table))
	for _, role := range slices.Sorted(maps.Keys(table)) {
		if !IsIdentifier(role) {
			*findings = append(*findings, fmt.Sprintf("[agent.roles]: %q is not a role name; a name is %s", role, IdentifierRule))
			continue
		}
		roles[role] = modelName("[agent.roles] "+role, table[role], models, findings)
	}
	return roles
}

func parseServers(value any, findings *[]string) []string {
	items, ok := value.([]any)
	if !ok {
		*findings = append(*findings, fmt.Sprintf("[agent] mcp_servers must be a list of names, found %s", tomlType(value)))
		return nil
	}
	servers := make([]string, 0, len(items))
	for i, item := range items {
		name, ok := item.(string)
		if !ok || name == "" {
			*findings = append(*findings, fmt.Sprintf("[agent] mcp_servers #%d must be a non-empty string", i+1))
			continue
		}
		servers = append(servers, name)
	}
	return servers
}

func parseTimeout(value any, findings *[]string) time.Duration {
	text, ok := value.(string)
	if !ok {
		*findings = append(*findings, `[agent] timeout must be a string such as "30m", or "0" to switch it off`)
		return DefaultAgentTimeout
	}
	d, err := time.ParseDuration(text)
	if text == "0" {
		d, err = 0, nil
	}
	if err != nil || d < 0 {
		*findings = append(*findings, fmt.Sprintf(`[agent] timeout %q is not a duration such as "30m", or "0" to switch it off`, text))
		return DefaultAgentTimeout
	}
	return d
}
