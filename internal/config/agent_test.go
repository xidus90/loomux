package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
)

func agentDoc(t *testing.T, text string) map[string]any {
	t.Helper()
	doc := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestParseAgentReadsModelsRolesDefaultAndServers(t *testing.T) {
	doc := agentDoc(t, `
[agent]
default = "writer"
mcp_servers = ["docs", "search"]

[agent.models.writer]
provider = "claude"
model = "claude-opus-5-5"

[agent.models.gemini]
provider = "agy"

[agent.roles]
reviewer = "gemini"
`)
	got, err := ParseAgent("c.toml", doc)
	if err != nil {
		t.Fatal(err)
	}
	want := Agent{
		Default:    "writer",
		MCPServers: []string{"docs", "search"},
		Models: map[string]ModelSpec{
			"writer": {Provider: "claude", Model: "claude-opus-5-5"},
			"gemini": {Provider: "agy"},
		},
		Roles:   map[string]string{"reviewer": "gemini"},
		Timeout: DefaultAgentTimeout,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseAgentWithoutTheTableIsEmpty(t *testing.T) {
	got, err := ParseAgent("c.toml", agentDoc(t, "[modules]\nhooks = true\n"))
	if err != nil || !reflect.DeepEqual(got, Agent{Timeout: DefaultAgentTimeout}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestParseAgentRefuses(t *testing.T) {
	for name, tc := range map[string]struct{ text, want string }{
		"not a table":        {"agent = 3", "[agent] must be a table"},
		"unknown key":        {"[agent]\nsettings = 1", `[agent] does not know "settings"`},
		"models not a table": {"[agent]\nmodels = 3", "[agent.models] must be a table of models"},
		"model not a table":  {"[agent.models]\nw = 3", "[agent.models.w] must be a table"},
		"model name":         {"[agent.models.\"a-b\"]\nprovider = \"claude\"", `"a-b" is not a model name`},
		"model unknown key":  {"[agent.models.w]\nprovider = \"claude\"\nsize = 1", `[agent.models.w] does not know "size"`},
		"no provider":        {"[agent.models.w]\nmodel = \"x\"", "[agent.models.w] needs provider"},
		"empty model":        {"[agent.models.w]\nprovider = \"claude\"\nmodel = \"\"", "[agent.models.w] model must be a non-empty string"},
		"default unknown":    {"[agent]\ndefault = \"w\"", `[agent] default names "w", which is not under [agent.models]; known: none`},
		"default not text":   {"[agent]\ndefault = 3", "[agent] default must be a non-empty string"},
		"roles not a table":  {"[agent]\nroles = 3", "[agent.roles] must be a table"},
		"role name":          {"[agent.models.w]\nprovider = \"claude\"\n[agent.roles]\n\"a-b\" = \"w\"", `"a-b" is not a role name`},
		"role unknown model": {"[agent.models.w]\nprovider = \"claude\"\n[agent.roles]\nreviewer = \"x\"", `[agent.roles] reviewer names "x", which is not under [agent.models]; known: w`},
		"unknown provider":   {"[agent.models.w]\nprovider = \"gpt\"", `[agent.models.w] provider "gpt" is unknown; known: agy, claude, qoder`},
		"base_url on agy":    {"[agent.models.w]\nprovider = \"agy\"\nbase_url = \"x\"", "[agent.models.w] base_url only works with provider claude"},
		"empty agent":        {"[agent.models.w]\nprovider = \"claude\"\nagent = \"\"", "[agent.models.w] agent must be a non-empty string"},
		"empty effort":       {"[agent.models.w]\nprovider = \"claude\"\neffort = \"\"", "[agent.models.w] effort must be a non-empty string"},
		"empty base_url":     {"[agent.models.w]\nprovider = \"claude\"\nbase_url = \"\"", "[agent.models.w] base_url must be a non-empty string"},
		"timeout text":       {"[agent]\ntimeout = \"soon\"", `[agent] timeout "soon" is not a duration such as "30m", or "0" to switch it off`},
		"timeout negative":   {"[agent]\ntimeout = \"-1m\"", `[agent] timeout "-1m" is not a duration`},
		"timeout not text":   {"[agent]\ntimeout = 3", `[agent] timeout must be a string such as "30m", or "0" to switch it off`},
		"servers not a list": {"[agent]\nmcp_servers = \"docs\"", "[agent] mcp_servers must be a list of names"},
		"empty server":       {"[agent]\nmcp_servers = [\"\"]", "[agent] mcp_servers #1 must be a non-empty string"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseAgent("c.toml", agentDoc(t, tc.text))
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.HasPrefix(err.Error(), "c.toml: ") {
				t.Fatalf("err = %v, want it to start with c.toml and hold %q", err, tc.want)
			}
		})
	}
}

func TestParseAgentReportsEveryFindingAtOnce(t *testing.T) {
	_, err := ParseAgent("c.toml", agentDoc(t, "[agent]\nsettings = 1\ndefault = 3\n"))
	if err == nil || !strings.Contains(err.Error(), "settings") || !strings.Contains(err.Error(), "default must be") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadAgent(t *testing.T) {
	root := t.TempDir()
	if got, err := ReadAgent(root); err != nil || !reflect.DeepEqual(got, Agent{Timeout: DefaultAgentTimeout}) {
		t.Fatalf("without a file: %+v, %v", got, err)
	}
	path := ManifestPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[agent.models.w]\nprovider = \"claude\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadAgent(root)
	if err != nil || got.Models["w"].Provider != "claude" {
		t.Fatalf("got %+v, %v", got, err)
	}
	if err := os.WriteFile(path, []byte("[agent"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadAgent(root); err == nil || !strings.Contains(err.Error(), "not valid TOML") {
		t.Fatalf("broken TOML: %v", err)
	}
}

func TestReadAgentReportsAnUnreadableFile(t *testing.T) {
	root := t.TempDir()
	path := ManifestPath(root)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadAgent(root); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("a directory where the file should be: %v, want an error naming %s", err, path)
	}
}

func TestParseAgentModelKeys(t *testing.T) {
	doc := agentDoc(t, "[agent.models.q]\nprovider=\"claude\"\nagent=\"reviewer\"\neffort=\"high\"\nbase_url=\"http://127.0.0.1:8080\"\n")
	got, err := ParseAgent("c.toml", doc)
	want := ModelSpec{Provider: "claude", Agent: "reviewer", Effort: "high", BaseURL: "http://127.0.0.1:8080"}
	if err != nil || got.Models["q"] != want {
		t.Fatalf("got %+v, %v", got.Models["q"], err)
	}
}

func TestParseAgentTimeout(t *testing.T) {
	for text, want := range map[string]time.Duration{
		"":                         DefaultAgentTimeout,
		"[agent]":                  DefaultAgentTimeout,
		"[agent]\ntimeout=\"45m\"": 45 * time.Minute,
		"[agent]\ntimeout=\"0\"":   0,
	} {
		got, err := ParseAgent("c.toml", agentDoc(t, text))
		if err != nil || got.Timeout != want {
			t.Fatalf("%q: got %v, %v, want %v", text, got.Timeout, err, want)
		}
	}
}
