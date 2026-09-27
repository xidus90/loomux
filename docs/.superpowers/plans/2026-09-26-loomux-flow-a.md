# Flow A — die Flow-Laufzeit zieht nach loomux: Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux flow run|resume|replay|show|list` fährt Flows aus Ordnern, aus einem mitgelieferten Katalog und aus Projekt-Overlays, mit Rollen, die ein Projekt in `.loomux/config.toml` an Modelle bindet; Session-Start meldet wartende Läufe, der Wächter hält Torantworten, Laufdateien und mitgelieferte Flows beim Menschen.

**Architecture:** Die Pakete von ulflow M1 (ultraloom `feature/agent-harness`, Commit `d7041e5`) ziehen unter `internal/flow/…` um, `flowcfg` geht in `internal/config` und `internal/flow/model` auf, `cmd/flow` wird `internal/cli/flow*.go`. Neu sind Ordnerformat, Rollen, Overlay, `[flow] default|overrides`, benannte Schema-Schlüssel, der Katalog `flows/catalog/` und drei Wächterregeln.

**Tech Stack:** Go 1.26 (`go.mod`), `third_party/toml` (BurntSushi v1.6.0), `embed`, `io/fs`, `testing/fstest`.

**Spec:** [`docs/.superpowers/specs/2026-09-26-loomux-flow-a-design.md`](../specs/2026-09-26-loomux-flow-a-design.md). Verhaltensvertrag, Bedingungssprache und Platzhalter stehen unverändert in der [M1-Spec](../specs-ul/2026-09-11-ulflow-laufzeit-design.md).

## Global Constraints

- Modul `github.com/xidus90/loomux`, `go 1.26.0`; **keine neue Abhängigkeit**, `go.mod`/`go.sum` bleiben unverändert.
- Coverage 100 % je Funktion; eine Ausnahme nur mit `//coverage:exempt <grund>` direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst; lesen beim ersten Gebrauch.
- Code, Kommentare, Fehlermeldungen, Commit-Texte englisch; dieser Plan deutsch.
- Commits: Conventional Commits; der Text nennt kein Arbeitspapier (kein „Task 3“, kein „Flow A“); Autor der Mensch, kein `Co-Authored-By`. Mehrzeilige Nachricht per Write in eine Datei und `git commit -F <datei>`.
- Kein Agent pusht, kein Agent schreibt `.loomux/config.toml`.
- Flow-Namen `[a-z][a-z0-9-]*`; Knoten-, Feld-, Parameter-, Rollen- und Modellnamen `[A-Za-z_][A-Za-z0-9_]*`.
- Exit-Codes: 0 fertig, 1 Fehler, 2 Aufruffehler, 3 pausiert, sonst der Code eines Bausteins.
- Läufe unter `.loomux/state/runs/`; Projekt-Flows unter `.loomux/flows/<name>/`; Katalog unter `flows/catalog/<name>/`, Tests darin unter `_test/`.
- Das Tor ist `sh ci/gate.sh`; `.githooks/pre-commit` fährt es bei jedem Commit. Seine Ausgabe erst ganz in eine Datei im Scratchpad schreiben, dann filtern.
- Code mit Backslashes (`"\r\n"`, Regex) nur per Write/Edit schreiben, nie per Heredoc oder `printf`; danach mit `grep` nachlesen.
- Vor jedem Commit `git rev-parse --show-toplevel --abbrev-ref HEAD` lesen: erwartet `…/.worktrees/flow-a` und `feat/flow-runtime`.

## Review Focus

1. **`loomux flow …` aus einem Unterverzeichnis des Projekts.** Wer in `internal/` steht, erwartet dieselben Flows wie an der Wurzel. Ohne `--root` sucht die Kommandozeile die Wurzel wie `loomux config` mit `hosts.FindRoot(".")` (Task 12, Test `TestFlowFindsTheRootFromASubdirectory`).
2. **Ein Pfad mit Backslash in `flow.toml`** (`instruction = "instructions\\draft.md"`, unter Windows naheliegend). Erwartet: Ladefehler, der `/` verlangt, statt „file not found“ (Task 10, Test `TestLoadRefusesABackslashInATextPath`).
3. **Ein Projektordner `.loomux/flows/<name>/` mit nur einer `README.md` oder einer Streudatei.** Erwartet: ein Ladefehler, der die Datei nennt und sagt, was erlaubt ist (Task 10, Test `TestFindRefusesAStrayFileInAProjectFolder`).
4. **Resume, nachdem sich die Herkunft geändert hat** (`[flow] overrides` gesetzt oder entfernt zwischen `run` und `resume`). Liegt jetzt eine andere `flow.toml` darunter (Wechsel zwischen `project`/`project (hides bundled)` und `bundled`/`bundled+overlay`), lehnt `resume` ab und nennt beide Herkünfte und `flow run`; ändert sich nur die Menge der Overlay-Dateien, läuft er weiter und warnt (Task 12, Tests `TestResumeRefusesAnotherFlowFile`, `TestResumeWarnsWhenTheOverlaysChanged`).
5. **Eine Frage mit CRLF** landet in Terminal und Session-Start ohne `\r` (Task 7 `TestReadTextTurnsCRLFIntoLF`, Task 12 `TestAPausedRunPrintsTheQuestionWithoutCarriageReturns`).

## Arbeitsumgebung

- Worktree `C:/Users/micro/Documents/#GIT/loomux/.worktrees/flow-a`, Zweig `feat/flow-runtime` auf `origin/master` (`f3cf731a`, v3.3.0).
- Quelle: `UL="/c/Users/micro/Documents/#GIT/ultraloom"`, Commit `d7041e5`. Gelesen wird nur mit `git -C "$UL" show d7041e5:<pfad>`; in ultraloom wird nichts geschrieben.
- Scratchpad: `S="/c/Users/micro/AppData/Local/Temp/claude/C--Users-micro-Documents--GIT-loomux/456f2afc-7b6c-48bb-94ad-5195d734a31c/scratchpad"`.

**Portier-Sed-Datei.** Einmal vor Task 3 per Write nach `$S/port.sed` (keine Backslashes, darum Heredoc-frei unkritisch, trotzdem per Write):

```
s#github.com/xidus90/ultra-loom/internal/flowload#github.com/xidus90/loomux/internal/flow/load#g
s#github.com/xidus90/ultra-loom/internal/blocks#github.com/xidus90/loomux/internal/flow/blocks#g
s#github.com/xidus90/ultra-loom/internal/runner#github.com/xidus90/loomux/internal/flow/runner#g
s#github.com/xidus90/ultra-loom/internal/journal#github.com/xidus90/loomux/internal/flow/journal#g
s#github.com/xidus90/ultra-loom/internal/runs#github.com/xidus90/loomux/internal/flow/runs#g
s#github.com/xidus90/ultra-loom/internal/model#github.com/xidus90/loomux/internal/flow/model#g
s#github.com/xidus90/ultra-loom/internal/#github.com/xidus90/loomux/internal/#g
s#package flowload#package load#
s#package flowload_test#package load_test#
s#flowload[.]#load.#g
s#[.]ultraloom/runs#.loomux/state/runs#g
s#[.]ultraloom/flows#.loomux/flows#g
```

Eine Datei portieren (ein Shell-Aufruf je Datei oder je Paket):

```bash
git -C "$UL" show "d7041e5:internal/flow/graph.go" | sed -f "$S/port.sed" > internal/flow/graph.go
```

**Testimporte vor dem Port lesen.** Das Tor läuft bei jedem Commit; ein portierter Test, der ein Paket eines späteren Tasks importiert, macht den Commit rot. Vor jedem Port-Task für jede Testdatei: `git -C "$UL" show d7041e5:<pfad>_test.go | sed -n '/^import/,/^)/p'`. Ein Test, der ein späteres Paket braucht, wandert in dessen Task. Stand der Prüfung beim Planen: nur `internal/runs/marker_test.go` greift vor (`flow.Baseline`); Task 4 stellt ihn auf `runs.Baseline` um. `flowcfg` in Tests von `blocks` und `flowload` wird `config` bzw. `model`, beide liegen früher.

Danach bleiben Verweise auf `flowcfg`, `ulflow`, Python und `cli.py` stehen, die die Tasks einzeln auflösen: `grep -rn "flowcfg\|ulflow\|Python\|cli.py\|ultraloom" <paket>` am Ende jedes Port-Tasks muss leer sein oder nur Sätze zeigen, die als Herkunftsangabe stimmen („journal.py's entries“ bleibt, „a Python journal stays readable“ nicht).

## Dateistruktur

| Pfad | Aufgabe | Task |
|---|---|---|
| `internal/config/names.go` | Namensregeln für Bezeichner und Flows | 1 |
| `internal/config/agent.go` | `[agent]`: Modelle, Default, Rollen, MCP-Server | 1 |
| `internal/config/flowsettings.go` | `[flow]`: `default`, `overrides` | 1 |
| `internal/config/schema/{schema,current,validate}.go` | neue Schlüssel, benannte Schlüssel, Leser in `readAll` | 1, 2 |
| `internal/cli/config.go`, `config_ui.go` | `config` versteht benannte Schlüssel | 2 |
| `internal/flow/journal/` | Journal lesen, schreiben, Hashes | 3 |
| `internal/flow/runs/` | Laufnummer, Marke, Basis-Typ | 4 |
| `internal/gitwork/gitwork.go` | `ChangedFiles` | 5 |
| `internal/flow/model/` | Port, Fake, Rollenauflösung, Werkzeugprofile | 6 |
| `internal/flow/`, `flow/expr/`, `flow/tmpl/` | Verträge, Bedingungen, Platzhalter | 7 |
| `internal/flow/blocks/` | `agent`, `gate`, `exit` | 8 |
| `internal/flow/runner/` | Gehen, Nachgehen, Tore, Deckel | 9 |
| `internal/flow/load/` | Format, Ladeprüfung, Auffindung, Overlay | 10 |
| `flows/embed.go`, `flows/catalog/example/` | Katalog und Beitragstest | 11 |
| `internal/cli/flow*.go` | `loomux flow …` | 12 |
| `internal/hooks/hook_session_start.go`, `flowruns.go` | wartende Läufe melden | 13 |
| `internal/hooks/guard.go`, `guardflow.go` | drei Wächterregeln | 14 |
| `docs/{en,de}/…`, `README*.md`, `AGENTS.md`, `.gitattributes`, `testdata/bench/flow-a-hooks.json` | Doku, Messung | 15 |

Abhängigkeiten zwischen Paketen (ohne Zyklus): `config` ← `flow/model` ← `flow` ← `flow/expr`, `flow/tmpl`, `flow/blocks`, `flow/runner`, `flow/load` ← `internal/cli`; `flow/runs` und `flow/journal` sind Blätter (nur Standardbibliothek), `flow` importiert `flow/runs` für den Basis-Typ. `internal/hooks` bindet nur `flow/runs`, `flow/journal`, `flows` und `config`.

---

### Task 1: `[agent]` und `[flow]` lesen

**Files:**
- Create: `internal/config/names.go`, `internal/config/names_test.go`
- Create: `internal/config/agent.go`, `internal/config/agent_test.go`
- Create: `internal/config/flowsettings.go`, `internal/config/flowsettings_test.go`
- Modify: `internal/config/schema/validate.go` (`readAll`), `internal/config/schema/validate_test.go`

**Interfaces:**
- Produces: `config.IsIdentifier(string) bool`, `config.IsFlowName(string) bool`, `config.IdentifierRule`, `config.FlowNameRule`; `config.ModelSpec{Provider, Model string}`; `config.Agent{Default string; MCPServers []string; Models map[string]ModelSpec; Roles map[string]string}`; `config.ReadAgent(root) (Agent, error)`, `config.ParseAgent(path, doc) (Agent, error)`, `config.AgentKeys() []string`; `config.FlowSettings{Default string; Overrides []string}`, `(FlowSettings).Allows(name) bool`, `config.ReadFlowSettings(root)`, `config.ParseFlowSettings(path, doc)`, `config.FlowKeys() []string`.

- [ ] **Step 1: Failing tests für die Namensregeln**

`internal/config/names_test.go`:

```go
package config

import "testing"

func TestIsIdentifier(t *testing.T) {
	for name, want := range map[string]bool{
		"reviewer": true, "_x": true, "Gemini_Pro2": true,
		"": false, "2fast": false, "dev-cycle": false, "a.b": false, "ü": false,
	} {
		if got := IsIdentifier(name); got != want {
			t.Errorf("IsIdentifier(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestIsFlowName(t *testing.T) {
	for name, want := range map[string]bool{
		"example": true, "dev-cycle": true, "strict-security-review2": true, "a": true,
		"": false, "Review": false, "_x": false, ".x": false, "-x": false, "9x": false, "dev_cycle": false, "a/b": false,
	} {
		if got := IsFlowName(name); got != want {
			t.Errorf("IsFlowName(%q) = %v, want %v", name, got, want)
		}
	}
}
```

- [ ] **Step 2: Rot sehen** — `go test ./internal/config/ -run 'TestIsIdentifier|TestIsFlowName'` → FAIL (`undefined: IsIdentifier`).

- [ ] **Step 3: `internal/config/names.go`**

```go
package config

// The two name rules, as a message states them.
const (
	IdentifierRule = "[A-Za-z_][A-Za-z0-9_]*"
	FlowNameRule   = "[a-z][a-z0-9-]*"
)

// IsIdentifier reports whether name follows the rule a role, a model, a node,
// a field and a parameter name share. It is the rule a flow's conditions can
// scan, and a key segment of .loomux/config.toml can hold without quotes.
func IsIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
		case i > 0 && c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return true
}

// IsFlowName reports whether name follows the rule of a flow's name. The name
// is the flow's folder, so it is lower case -- Windows does not tell Review
// from review -- and it starts with a letter, because go:embed leaves out a
// name that starts with "_" or ".".
func IsFlowName(name string) bool {
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for i := 1; i < len(name); i++ {
		c := name[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Grün** — derselbe Befehl → PASS.

- [ ] **Step 5: Failing tests für `[agent]`**

`internal/config/agent_test.go` (Tabellentest; `doc` aus TOML-Text über `toml.Unmarshal`):

```go
package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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
		Roles: map[string]string{"reviewer": "gemini"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseAgentWithoutTheTableIsEmpty(t *testing.T) {
	got, err := ParseAgent("c.toml", agentDoc(t, "[modules]\nhooks = true\n"))
	if err != nil || !reflect.DeepEqual(got, Agent{}) {
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
	if got, err := ReadAgent(root); err != nil || !reflect.DeepEqual(got, Agent{}) {
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
```

Ein Lesefehler außer „nicht da“ (Verzeichnis an Stelle der Datei) bekommt einen eigenen Fall: `os.MkdirAll(path, 0o755)` statt Datei, `ReadAgent` → Fehler mit dem Pfad.

- [ ] **Step 6: Rot sehen** — `go test ./internal/config/ -run 'Agent'` → FAIL.

- [ ] **Step 7: `internal/config/agent.go`**

Vorher `grep -n "func tomlType" internal/config/*.go` — die Hilfe gibt es (in `modules.go` benutzt); sie wird mitbenutzt.

```go
package config

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

// ModelSpec is one entry under [agent.models]: who answers, and with which of
// its models. An empty Model is the provider CLI's own default.
type ModelSpec struct {
	Provider string
	Model    string
}

// Agent is the [agent] table: the models a flow's roles can be bound to.
type Agent struct {
	Default    string               // model name for every role without a binding; "" when unset
	MCPServers []string             // the servers the mcp tool profile opens
	Models     map[string]ModelSpec // by model name
	Roles      map[string]string    // role name to model name
}

// AgentKeys are the keys [agent] knows, sorted. settings arrives with the
// model adapters and is unknown until then.
func AgentKeys() []string { return []string{"default", "mcp_servers", "models", "roles"} }

// ReadAgent reads [agent] of the project at root. A project without the file
// or the table has an empty Agent: every role then runs on the claude CLI's
// own default.
func ReadAgent(root string) (Agent, error) {
	path := ManifestPath(root)
	doc, err := manifestDocument(path)
	if err != nil || doc == nil {
		return Agent{}, err
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
		return Agent{}, nil
	}
	table, ok := raw.(map[string]any)
	if !ok {
		return Agent{}, fmt.Errorf("%s: [agent] must be a table, found %s", path, tomlType(raw))
	}
	var agent Agent
	var found []string
	for _, key := range slices.Sorted(maps.Keys(table)) {
		if !slices.Contains(AgentKeys(), key) {
			found = append(found, fmt.Sprintf("[agent] does not know %q; known: %s", key, strings.Join(AgentKeys(), ", ")))
		}
	}
	if value, present := table["models"]; present {
		agent.Models = parseModels(value, &found)
	}
	if value, present := table["default"]; present {
		agent.Default = modelName("[agent] default", value, agent.Models, &found)
	}
	if value, present := table["roles"]; present {
		agent.Roles = parseRoles(value, agent.Models, &found)
	}
	if value, present := table["mcp_servers"]; present {
		agent.MCPServers = parseServers(value, &found)
	}
	if len(found) > 0 {
		return Agent{}, fmt.Errorf("%s: %s", path, strings.Join(found, "; "))
	}
	return agent, nil
}

func parseModels(value any, found *[]string) map[string]ModelSpec {
	table, ok := value.(map[string]any)
	if !ok {
		*found = append(*found, fmt.Sprintf("[agent.models] must be a table of models, found %s", tomlType(value)))
		return nil
	}
	models := make(map[string]ModelSpec, len(table))
	for _, name := range slices.Sorted(maps.Keys(table)) {
		where := "[agent.models." + name + "]"
		if !IsIdentifier(name) {
			*found = append(*found, fmt.Sprintf("%s: %q is not a model name; a name is %s", where, name, IdentifierRule))
		}
		entry, ok := table[name].(map[string]any)
		if !ok {
			*found = append(*found, fmt.Sprintf("%s must be a table, found %s", where, tomlType(table[name])))
			continue
		}
		for _, key := range slices.Sorted(maps.Keys(entry)) {
			if key != "provider" && key != "model" {
				*found = append(*found, fmt.Sprintf("%s does not know %q; known: model, provider", where, key))
			}
		}
		provider, _ := entry["provider"].(string)
		if provider == "" {
			*found = append(*found, where+" needs provider as a non-empty string")
		}
		model, isText := entry["model"].(string)
		if _, has := entry["model"]; has && (!isText || model == "") {
			*found = append(*found, where+" model must be a non-empty string")
		}
		models[name] = ModelSpec{Provider: provider, Model: model}
	}
	return models
}

// modelName reads a value that has to name a model under [agent.models].
func modelName(where string, value any, models map[string]ModelSpec, found *[]string) string {
	name, ok := value.(string)
	if !ok || name == "" {
		*found = append(*found, where+" must be a non-empty string")
		return ""
	}
	if _, known := models[name]; !known {
		*found = append(*found, fmt.Sprintf("%s names %q, which is not under [agent.models]; known: %s", where, name, knownModels(models)))
	}
	return name
}

func knownModels(models map[string]ModelSpec) string {
	if len(models) == 0 {
		return "none"
	}
	return strings.Join(slices.Sorted(maps.Keys(models)), ", ")
}

func parseRoles(value any, models map[string]ModelSpec, found *[]string) map[string]string {
	table, ok := value.(map[string]any)
	if !ok {
		*found = append(*found, fmt.Sprintf("[agent.roles] must be a table of role = model, found %s", tomlType(value)))
		return nil
	}
	roles := make(map[string]string, len(table))
	for _, role := range slices.Sorted(maps.Keys(table)) {
		if !IsIdentifier(role) {
			*found = append(*found, fmt.Sprintf("[agent.roles]: %q is not a role name; a name is %s", role, IdentifierRule))
			continue
		}
		roles[role] = modelName("[agent.roles] "+role, table[role], models, found)
	}
	return roles
}

func parseServers(value any, found *[]string) []string {
	items, ok := value.([]any)
	if !ok {
		*found = append(*found, fmt.Sprintf("[agent] mcp_servers must be a list of names, found %s", tomlType(value)))
		return nil
	}
	servers := make([]string, 0, len(items))
	for i, item := range items {
		name, ok := item.(string)
		if !ok || name == "" {
			*found = append(*found, fmt.Sprintf("[agent] mcp_servers #%d must be a non-empty string", i+1))
			continue
		}
		servers = append(servers, name)
	}
	return servers
}
```

- [ ] **Step 8: Grün** — `go test ./internal/config/ -run 'Agent' -cover` → PASS.

- [ ] **Step 9: Failing tests für `[flow]`** — `internal/config/flowsettings_test.go`, gleiche Form: liest `default = "example"` und `overrides = ["dev-cycle", "review"]`; ohne Tabelle leer; lehnt ab: `flow = 3` („[flow] must be a table“), unbekannter Schlüssel („[flow] does not know \"name\"“), `default = "Dev"` („[flow] default must be a flow name“), `overrides = "x"` („[flow] overrides must be a list of flow names“), `overrides = ["a", 3]` („#2 must be a flow name“), `overrides = ["a", "a"]` („names \"a\" twice“); `Allows("review")` true, `Allows("x")` false; `ReadFlowSettings` ohne Datei, mit Datei, mit kaputtem TOML (wie `TestReadAgent`).

- [ ] **Step 10: Rot sehen**, dann **`internal/config/flowsettings.go`**:

```go
package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// FlowSettings is the [flow] table: which flow runs without a name, and
// which bundled flows a project flow of the same name may hide or overlay.
type FlowSettings struct {
	Default   string
	Overrides []string
}

// FlowKeys are the keys [flow] knows, sorted.
func FlowKeys() []string { return []string{"default", "overrides"} }

// Allows reports whether a project flow may hide or overlay the bundled flow
// name. The answer lives in this file because only a human writes it: a
// bundled flow's gates are not the party's to remove that they ask.
func (s FlowSettings) Allows(name string) bool { return slices.Contains(s.Overrides, name) }

// ReadFlowSettings reads [flow] of the project at root.
func ReadFlowSettings(root string) (FlowSettings, error) {
	path := ManifestPath(root)
	doc, err := manifestDocument(path)
	if err != nil || doc == nil {
		return FlowSettings{}, err
	}
	return ParseFlowSettings(path, doc)
}

// ParseFlowSettings reads [flow] from a decoded document, every finding at
// once. Whether default names a flow that exists is not asked here: this
// reader reads no flows.
func ParseFlowSettings(path string, doc map[string]any) (FlowSettings, error) {
	raw, present := doc["flow"]
	if !present {
		return FlowSettings{}, nil
	}
	table, ok := raw.(map[string]any)
	if !ok {
		return FlowSettings{}, fmt.Errorf("%s: [flow] must be a table, found %s", path, tomlType(raw))
	}
	var settings FlowSettings
	var found []string
	for _, key := range slices.Sorted(maps.Keys(table)) {
		if !slices.Contains(FlowKeys(), key) {
			found = append(found, fmt.Sprintf("[flow] does not know %q; known: %s", key, strings.Join(FlowKeys(), ", ")))
		}
	}
	if value, present := table["default"]; present {
		name, ok := value.(string)
		if !ok || !IsFlowName(name) {
			found = append(found, "[flow] default must be a flow name; a flow name is "+FlowNameRule)
		}
		settings.Default = name
	}
	if value, present := table["overrides"]; present {
		items, ok := value.([]any)
		if !ok {
			found = append(found, fmt.Sprintf("[flow] overrides must be a list of flow names, found %s", tomlType(value)))
		}
		for i, item := range items {
			name, ok := item.(string)
			switch {
			case !ok || !IsFlowName(name):
				found = append(found, fmt.Sprintf("[flow] overrides #%d must be a flow name; a flow name is %s", i+1, FlowNameRule))
			case slices.Contains(settings.Overrides, name):
				found = append(found, fmt.Sprintf("[flow] overrides names %q twice", name))
			default:
				settings.Overrides = append(settings.Overrides, name)
			}
		}
	}
	if len(found) > 0 {
		return FlowSettings{}, fmt.Errorf("%s: %s", path, strings.Join(found, "; "))
	}
	return settings, nil
}
```

- [ ] **Step 11: `schema.Validate` fragt beide Leser.** Failing test in `internal/config/schema/validate_test.go`:

```go
func TestValidateAsksTheAgentAndFlowReaders(t *testing.T) {
	for text, want := range map[string]string{
		"[agent]\ndefault = \"w\"\n": "not under [agent.models]",
		"[flow]\ndefault = \"Dev\"\n":  "[flow] default must be a flow name",
	} {
		err := Validate(text)
		if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), ".loomux/config.toml") {
			t.Errorf("Validate(%q) = %v, want %q named against .loomux/config.toml", text, err, want)
		}
	}
	if err := Validate("[agent.models.w]\nprovider = \"claude\"\n[agent.roles]\nreviewer = \"w\"\n[flow]\noverrides = [\"example\"]\n"); err != nil {
		t.Fatalf("a sound file: %v", err)
	}
}
```

Dann in `readAll` (nach `commit.ReadPolicy`, vor `mirror.Mirror`):

```go
	if _, err := config.ReadAgent(root); err != nil {
		return err
	}
	if _, err := config.ReadFlowSettings(root); err != nil {
		return err
	}
```

- [ ] **Step 12: Grün und Tor** — `go test ./internal/config/... -count=1` → PASS. Commit:

```
feat(config): read the [agent] and [flow] tables

[agent] names the models a flow's roles run on, binds roles to them and
sets a default; [flow] picks the flow `loomux flow run` starts without a
name and names the bundled flows a project flow may hide or overlay.
Both readers report every finding at once and refuse a binding or a
default that names no model, and schema.Validate asks them, so
`config set` cannot write a file the flow commands would refuse.
```

---

### Task 2: benannte Schlüssel in Schema und `loomux config`

**Files:**
- Modify: `internal/config/schema/schema.go` (neue Keys, `Wildcard`, `Named`, `withName`, `Match`, `Lookup`)
- Modify: `internal/config/schema/current.go` (`CurrentOf` expandiert benannte Schlüssel; `lookup` über `lookupPath`)
- Modify: `internal/cli/config.go` (`configTarget.lookup` über `schema.Match`), `internal/cli/config_ui.go` (`changeOne` verweigert einen benannten Platzhalter)
- Test: `internal/config/schema/schema_test.go`, `current_test.go`, `internal/cli/config_test.go`, `internal/config/edit/edit_test.go`

**Interfaces:**
- Consumes: `config.IsIdentifier` (Task 1).
- Produces: `schema.Wildcard = "*"`, `(Key).Named() bool`, `schema.Match(keys []Key, id string) (Key, bool)`; `schema.Lookup(id)` = `Match(Keys(), id)`. Neue IDs: `agent.default`, `agent.mcp_servers`, `agent.models.*.provider`, `agent.models.*.model`, `agent.roles.*`, `flow.default`, `flow.overrides`, alle `Module: Base`.

- [ ] **Step 1: Beleg für `edit.Set` mit fehlender benannter Tabelle** (Spec, „Konfiguration“). Test in `internal/config/edit/edit_test.go`:

```go
func TestSetAddsANamedTableThatIsNotThereYet(t *testing.T) {
	text := "[agent]\ndefault = \"w\"\n\n[agent.models.w]\nprovider = \"claude\"\n"
	got, err := Set(text, "agent.models.gemini", "provider", `"agy"`)
	if err != nil {
		t.Fatal(err)
	}
	want := text + "\n[agent.models.gemini]\nprovider = \"agy\"\n"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	got, err = Set(got, "agent.roles", "reviewer", `"gemini"`)
	if err != nil || !strings.HasSuffix(got, "[agent.roles]\nreviewer = \"gemini\"\n") {
		t.Fatalf("got %q, %v", got, err)
	}
}
```

`go test ./internal/config/edit/ -run TestSetAddsANamedTable`. Ist er grün, braucht `edit` nichts. Ist er rot, ist das ein Befund: den Grund in `edit.go` lesen, die kleinste Änderung machen, die ihn grün macht, und im Task-Bericht nennen. Weiter erst mit grünem Test.

- [ ] **Step 2: Failing tests für `Match`** (`schema_test.go`):

```go
func TestMatchFillsANamedKey(t *testing.T) {
	keys := Keys()
	for id, want := range map[string][2]string{
		"agent.roles.reviewer":          {"agent.roles", "reviewer"},
		"agent.models.gemini.provider":  {"agent.models.gemini", "provider"},
		"agent.models.gemini.model":     {"agent.models.gemini", "model"},
		"agent.default":                 {"agent", "default"},
		"flow.overrides":                {"flow", "overrides"},
	} {
		k, ok := Match(keys, id)
		if !ok || k.Section != want[0] || k.Name != want[1] || k.Named() {
			t.Errorf("Match(%q) = %+v, %v", id, k, ok)
		}
	}
	for _, id := range []string{"agent.roles.*", "agent.roles.a-b", "agent.roles", "agent.models.gemini", "agent.models.gemini.size", "agent.roles.x.y"} {
		if k, ok := Match(keys, id); ok {
			t.Errorf("Match(%q) = %+v, want no key", id, k)
		}
	}
}

func TestEveryNewKeyIsBase(t *testing.T) {
	for _, id := range []string{"agent.default", "agent.mcp_servers", "agent.models.*.provider", "agent.models.*.model", "agent.roles.*", "flow.default", "flow.overrides"} {
		found := false
		for _, k := range Keys() {
			if k.ID() == id {
				found = true
				if k.Module != Base {
					t.Errorf("%s is in module %s", id, k.Module)
				}
			}
		}
		if !found {
			t.Errorf("no key %s", id)
		}
	}
}
```

Bestehende Tests, die die ganze Schlüsselliste oder „jede ID ist eindeutig und auffindbar per Lookup“ prüfen, werden an benannte Schlüssel angepasst: ein benannter Schlüssel ist nicht per eigener ID auffindbar (sie enthält `*`), nur per gefüllter.

- [ ] **Step 3: Rot sehen**, dann in `schema.go`:

In `Keys()` nach den `worktree`-Zeilen:

```go
		{Section: "agent", Name: "default", Kind: String, Module: Base, Doc: "The model every flow role without a binding runs on; a name under agent.models."},
		{Section: "agent", Name: "mcp_servers", Kind: StringList, Default: "[]", Module: Base, Doc: "The MCP servers a flow node with the mcp tool profile may use."},
		{Section: "agent.models.*", Name: "provider", Kind: String, Module: Base, Doc: "Who answers for this model name, e.g. claude or agy."},
		{Section: "agent.models.*", Name: "model", Kind: String, Module: Base, Doc: "The provider's model; unset, the provider CLI's own default."},
		{Section: "agent.roles", Name: "*", Kind: String, Module: Base, Doc: "The model name a flow role runs on."},
		{Section: "flow", Name: "default", Kind: String, Module: Base, Doc: "The flow `loomux flow run` starts without a name."},
		{Section: "flow", Name: "overrides", Kind: StringList, Default: "[]", Module: Base, Doc: "Bundled flows a project flow of the same name may hide or overlay."},
```

Und neu:

```go
// Wildcard stands for a name in a key: agent.roles.* is every role,
// agent.models.*.provider every model's provider.
const Wildcard = "*"

// Named reports whether the key stands for a family of keys, one per name.
func (k Key) Named() bool {
	return strings.Contains(k.Section, Wildcard) || k.Name == Wildcard
}

// withName is the member of a named key's family that name picks.
func (k Key) withName(name string) Key {
	k.Section = strings.Replace(k.Section, Wildcard, name, 1)
	if k.Name == Wildcard {
		k.Name = name
	}
	return k
}

// Match finds the key id names among keys: an exact ID first, then a named key
// whose wildcard id fills with a name. The key comes back as id spells it, so
// an editor writes [agent.roles] reviewer and never a star.
func Match(keys []Key, id string) (Key, bool) {
	for _, k := range keys {
		if !k.Named() && k.ID() == id {
			return k, true
		}
	}
	parts := strings.Split(id, ".")
	for _, k := range keys {
		if !k.Named() {
			continue
		}
		pattern := strings.Split(k.ID(), ".")
		if len(pattern) != len(parts) {
			continue
		}
		if name, ok := fill(pattern, parts); ok {
			return k.withName(name), true
		}
	}
	return Key{}, false
}

// fill is the name that turns pattern into parts, when one does: every other
// segment equal, and the name a valid one.
func fill(pattern, parts []string) (string, bool) {
	name := ""
	for i, segment := range pattern {
		if segment == Wildcard {
			if !config.IsIdentifier(parts[i]) {
				return "", false
			}
			name = parts[i]
			continue
		}
		if segment != parts[i] {
			return "", false
		}
	}
	return name, name != ""
}
```

`Lookup` wird `return Match(Keys(), id)`. `strings` in die Imports.

- [ ] **Step 4: Grün** für `schema_test.go`.

- [ ] **Step 5: Failing test für die Anzeige** (`current_test.go`):

```go
func TestCurrentListsEveryMemberOfANamedKey(t *testing.T) {
	entries, err := Current("[agent.models.w]\nprovider = \"claude\"\n\n[agent.models.g]\nprovider = \"agy\"\nmodel = \"gemini-3\"\n\n[agent.roles]\nreviewer = \"g\"\n")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, e := range entries {
		if strings.HasPrefix(e.Key.ID(), "agent.") {
			got[e.Key.ID()] = string(e.Origin) + " " + e.Value
		}
	}
	want := map[string]string{
		"agent.default":            "unset ",
		"agent.mcp_servers":        "default []",
		"agent.models.g.provider":  `set "agy"`,
		"agent.models.g.model":     `set "gemini-3"`,
		"agent.models.w.provider":  `set "claude"`,
		"agent.models.w.model":     "unset ",
		"agent.roles.reviewer":     `set "g"`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}

func TestCurrentShowsAnEmptyNamedKeyOnceAsUnset(t *testing.T) {
	entries, err := Current("")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"agent.roles.*", "agent.models.*.provider", "agent.models.*.model"} {
		n := 0
		for _, e := range entries {
			if e.Key.ID() == id {
				n++
				if e.Origin != Unset {
					t.Errorf("%s is %s", id, e.Origin)
				}
			}
		}
		if n != 1 {
			t.Errorf("%s shown %d times", id, n)
		}
	}
}
```

- [ ] **Step 6: Rot sehen**, dann `current.go`: den Rumpf der Schleife in `CurrentOf` in `entryOf(doc map[string]any, k Key) Entry` ziehen (unverändert), und die Schleife wird:

```go
	for _, k := range keys {
		if k.Named() {
			out = append(out, family(doc, k)...)
			continue
		}
		out = append(out, entryOf(doc, k))
	}
```

Neu:

```go
// family expands a named key into one entry per name the document holds, in
// name order, or into the key itself, unset, when it holds none: `config
// list` still shows that the family exists and how its IDs are spelled.
func family(doc map[string]any, k Key) []Entry {
	holder, _ := lookupPath(doc, k.parent())
	table, _ := holder.(map[string]any)
	var out []Entry
	for _, name := range slices.Sorted(maps.Keys(table)) {
		if config.IsIdentifier(name) {
			out = append(out, entryOf(doc, k.withName(name)))
		}
	}
	if len(out) == 0 {
		out = append(out, Entry{Key: k, Origin: Unset})
	}
	return out
}
```

`(Key).parent()` in `schema.go`: die Segmente der ID vor dem ersten `*` (`agent.roles.*` → `["agent","roles"]`, `agent.models.*.provider` → `["agent","models"]`). `lookup(doc, k)` ruft künftig `lookupPath(doc, path)` mit den Segmenten aus Section und Name; `lookupPath` ist der bisherige Rumpf von `lookup`. Imports: `maps`, `slices`, `github.com/xidus90/loomux/internal/config` (schema importiert `config` schon).

- [ ] **Step 7: Grün.** Dann `go test ./internal/setup/... ./internal/cli/... -count=1`: `setup` liest `schema.Current`; ein Aufrufer, der jede Zeile als schreibbaren Schlüssel nimmt, muss benannte Platzhalter (`e.Key.Named()`) überspringen. Jeden roten Test so lesen und einzeln begründet anpassen.

  **Aufzeichnungen nie von Hand ändern.** Die sieben neuen Schlüssel und die Platzhalterzeilen ändern die Ausgabe von `config list` und womöglich der `init`-Vorlage. Vorher `grep -rln "worktree.mirror" testdata internal --include=*golden* --include=*.txt --include=*.json --include=*.md` — jede Datei, die eine Schlüsselliste aufzeichnet, zeigt sich so. Ein Treffer unter `testdata/cases/` wird mit dem Recorder neu aufgezeichnet (`loomux dev record-case`, Aufruf wie in der Akte der Stufe, die ihn aufgenommen hat), ein Golden eines Go-Tests mit dessen `-update`; beides steht im Task-Bericht mit Datei und Grund.

- [ ] **Step 8: Failing tests für `loomux config`** (`internal/cli/config_test.go`, im Stil der bestehenden Tests dort, gegen ein Projekt im TempDir):

1. `config set agent.models.gemini.provider agy --yes`, dann `config set agent.roles.reviewer gemini --yes` → Datei hält `[agent.models.gemini]\nprovider = "agy"` und `[agent.roles]\nreviewer = "gemini"`.
2. `config set agent.roles.reviewer nothing --yes` → Exit 1, stderr enthält `not under [agent.models]` (Validate aus Task 1), Datei unverändert.
3. `config get agent.roles.reviewer` → `"gemini"`.
4. `config set agent.roles.* x` → Exit 1, `unknown key "agent.roles.*"`.
5. `config set agent.roles.reviewer gemini --propose` legt einen Vorschlag an (bestehender Mechanismus) und ändert die Datei nicht.
6. `config list` zeigt `agent.roles.reviewer` und bei leerer Datei eine Zeile `agent.roles.*  unset`.

Wie die bestehenden Tests `--yes`, Bestätigung und Vorschläge ansteuern, steht in `config_test.go` und `config_proposals_test.go`; dieselben Helfer nehmen.

- [ ] **Step 9: Rot sehen**, dann in `internal/cli/config.go`:

```go
func (t configTarget) lookup(id string) (schema.Key, bool) {
	return schema.Match(t.keys, id)
}
```

und in `config_ui.go` am Anfang von `changeOne`:

```go
	if e.Key.Named() {
		return fmt.Errorf("%s stands for one key per name; set one with `loomux config set %s <value>`", e.Key.ID(), strings.Replace(e.Key.ID(), schema.Wildcard, "<name>", 1))
	}
```

Mit einem Test in `config_ui_test.go` im Stil der bestehenden, der einen benannten Platzhalter wählt und die Meldung erwartet.

- [ ] **Step 10: Grün, Tor, Commit**

```
feat(config): address named keys such as agent.roles.<role>

A role binding and a model's provider sit in tables whose keys a project
chooses. The schema now knows keys with a name segment, `config list`
shows one row per name the file holds and one unset row for a family it
holds none of, and `config get|set|unset` and `--propose` reach
agent.roles.<role> and agent.models.<name>.provider|model like any other
key. The interactive form sends a family row to `config set`.
```

---

### Task 3: das Journal

**Files:**
- Create (Port): `internal/flow/journal/{journal,hash,write,gate,lookup}.go` und ihre `_test.go` aus `d7041e5:internal/journal/`.

**Interfaces:**
- Produces: `journal.Entry` mit den Feldern `Node, Kind, InputHash, Delta, Outcome, Tools, Effort, Tokens, Seconds, Detail, Model, Role, DefinitionHash`; `Entries(path) ([]Entry, error)`, `Append(path, Entry) error`, `Pending(path) (*PendingGate, error)`, `Lookup(entries, node, hash, outcome) (Entry, bool)`, `Canonical(any) ([]byte, error)`, `InputHash(node, data) (string, error)`, `DefinitionHash(node any, text []byte, model, effort string, tools []string) (string, error)`.

- [ ] **Step 1: Port** — jede Datei per `git show … | sed -f "$S/port.sed"` nach `internal/flow/journal/`. Das Paket importiert nur die Standardbibliothek.

- [ ] **Step 2: Failing tests für die Änderungen** in `journal_test.go`:

```go
func TestEntriesRefusesALineWithoutRole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.jsonl")
	line := `{"node":"a","kind":"gate","input_hash":"h","delta":{},"outcome":"ok","tools":null,"effort":null,"tokens":0,"seconds":0,"detail":null,"model":null,"definition_hash":"d"}`
	if err := os.WriteFile(path, []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Entries(path); err == nil || !strings.Contains(err.Error(), "no value for role") {
		t.Fatalf("err = %v", err)
	}
}

func TestAppendWritesRoleEvenWhenThereIsNone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.jsonl")
	if err := Append(path, Entry{Node: "a", Kind: "gate", InputHash: "h", Delta: map[string]any{}, Outcome: "ok"}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), `"role":null`) {
		t.Fatalf("line %s", raw)
	}
	role := "reviewer"
	if err := Append(path, Entry{Node: "b", Kind: "agent", InputHash: "h", Delta: map[string]any{}, Outcome: "ok", Role: &role}); err != nil {
		t.Fatal(err)
	}
	entries, err := Entries(path)
	if err != nil || entries[1].Role == nil || *entries[1].Role != "reviewer" {
		t.Fatalf("entries %+v, %v", entries, err)
	}
}
```

Die Tests, die belegen, dass eine Zeile **ohne** `model` oder `definition_hash` gelesen wird (Python-Journale), werden umgedreht: dieselbe Zeile ist jetzt ein Fehler „no value for model, role, definition_hash“ (die Reihenfolge von `entryKeys`, also der Felder). `grep -n "Python\|optional" internal/flow/journal/*_test.go` findet sie.

- [ ] **Step 3: Rot sehen**, dann `journal.go`:
  - `Entry` bekommt nach `Model` das Feld ``Role *string `json:"role"` ``. Der Kommentar über `Model` und `DefinitionHash` wird: „Model, Role and DefinitionHash are null on a line that has none of them -- a gate asks no model and plays no role -- and never absent.“
  - `entryKeys` wird die Liste aller dreizehn Schlüssel in Feldreihenfolge; `optionalKeys` und seine Prüfung in `decode` entfallen. Der Kommentar über `entryKeys` erklärt nur noch, warum die Liste ausgeschrieben ist (eine abgeleitete Liste machte jedes neue Feld stillschweigend Pflicht).
  - Paketkommentar: „Package journal is a flow run's journal: one JSONL line per step. The same file is the log a person reads and the only source a resume reads from; loomux hook session-start only reads it.“
  - In `hash.go` fällt der Satz „Neither runtime resumes the other's runs.“ weg; der Satz über die abweichenden Bytes gegenüber `json.dumps` bleibt als Herkunft.

- [ ] **Step 4: Grün** — `go test ./internal/flow/journal/ -cover -count=1` → PASS, 100 %.

- [ ] **Step 5: Commit** — `feat(flow): add the run journal with its writer and hashes` (Rumpf: ein JSONL-Eintrag je Schritt, kanonisches JSON, Eingabe- und Definitions-Hash; jede Zeile trägt `model`, `role` und `definition_hash`, `null` wo sie nichts sagen).

---

### Task 4: Laufnummern und Marken

**Files:**
- Create (Port): `internal/flow/runs/{doc,id,claim,marker}.go` und Tests aus `d7041e5:internal/runs/`.
- Create: `internal/flow/runs/baseline.go`.

**Interfaces:**
- Produces: `runs.Dir = ".loomux/state/runs"`; `runs.Baseline{Commit string; Dirty []string}`; `runs.Marker{Flow, Origin string; Overlays []string; Options map[string]string; Baseline *Baseline; Version string}`; `NextID(root)`, `JournalPath(root,id)`, `MarkerPath(root,id)`, `Files(id)`, `Claim(root, Marker) (string, error)`, `WriteMarker(path, Marker)`, `ReadMarker(path) (*Marker, error)`.

- [ ] **Step 1: Port** der vier Dateien und Tests. `marker.go` importiert danach `internal/flow` nicht mehr (Schritt 3), damit `internal/hooks` dieses Paket ohne die Laufzeit binden kann.

- [ ] **Step 2: Failing tests** in `marker_test.go`:

```go
func TestAMarkerCarriesOriginOverlaysAndVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.flow")
	in := Marker{
		Flow: "dev-cycle", Origin: "bundled+overlay",
		Overlays: []string{"questions/approve.md", "instructions/review.md"},
		Options:  map[string]string{"max_rounds": "3"},
		Baseline: &Baseline{Commit: "abc", Dirty: []string{"b.go", "a.go"}},
		Version:  "3.3.0",
	}
	if err := WriteMarker(path, in); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	want := "dev-cycle\n" +
		"baseline=\"a.go\\nb.go\"\n" +
		"baseline_commit=\"abc\"\n" +
		"loomux_version=\"3.3.0\"\n" +
		"max_rounds=\"3\"\n" +
		"origin=\"bundled+overlay\"\n" +
		"overlays=\"instructions/review.md\\nquestions/approve.md\"\n"
	if string(raw) != want {
		t.Fatalf("got %q\nwant %q", raw, want)
	}
	out, err := ReadMarker(path)
	if err != nil {
		t.Fatal(err)
	}
	if out.Origin != "bundled+overlay" || out.Version != "3.3.0" ||
		!reflect.DeepEqual(out.Overlays, []string{"instructions/review.md", "questions/approve.md"}) ||
		!reflect.DeepEqual(out.Options, map[string]string{"max_rounds": "3"}) {
		t.Fatalf("read back %+v", out)
	}
}

func TestAnOptionMayNotTakeAMarkerName(t *testing.T) {
	for _, name := range []string{"baseline", "baseline_commit", "origin", "overlays", "loomux_version"} {
		err := WriteMarker(filepath.Join(t.TempDir(), "x.flow"), Marker{Flow: "f", Options: map[string]string{name: "1"}})
		if err == nil || !strings.Contains(err.Error(), "reserved") {
			t.Errorf("%s: %v", name, err)
		}
	}
}
```

(Der erwartete Text enthält Backslashes: diesen Test per Write schreiben und danach mit `grep -n 'a.go' internal/flow/runs/marker_test.go` nachlesen.) Die M1-Tests zu `Runtime` und `ulflow_version` entfallen; der Test von `Files` erwartet `.loomux/state/runs/0001.flow` und `.loomux/state/runs/0001.jsonl`.

- [ ] **Step 3: Rot sehen**, dann:
  - `baseline.go`:
    ```go
    package runs

    // Baseline is what a run starts from: the HEAD commit and the paths that
    // were already changed then, relative to the project root. It lives here,
    // with the marker that carries it, so that reading a marker links no part
    // of the runtime.
    type Baseline struct {
    	Commit string
    	Dirty  []string
    }
    ```
  - `id.go`: `const Dir = ".loomux/state/runs"`; der Kommentar über Windows bleibt.
  - `marker.go`: `Runtime` entfällt; `Origin string` und `Overlays []string` kommen dazu; die Schlüssel werden `keyOrigin = "origin"`, `keyOverlays = "overlays"`, `keyVersion = "loomux_version"`; `reserved` hält alle fünf. `WriteMarker` schreibt `origin`, wenn gesetzt, und `overlays` sortiert mit `\n` verbunden, wenn nicht leer (wie `baseline`); `ReadMarker` liest beide zurück (`decodeDirty` taugt für beide Listen, darum umbenennen in `decodeList`). `*flow.Baseline` wird `*Baseline`; der Import von `internal/flow` entfällt.
  - `claim.go`: der Absatz „A Python run is not held back …“ entfällt; der Kommentar endet mit „The journal comes after, and so never belongs to two runs.“
  - `marker.go` behält den benannten kleinen Befund als Kommentar an `WriteMarker`: „A write that fails after the create leaves half a marker that NextID counts and ReadMarker refuses; it costs one number and needs a full disk.“

- [ ] **Step 4: Grün, 100 %, Commit** — `feat(flow): number runs and write their markers` (Rumpf: Läufe unter `.loomux/state/runs/`, Marke mit Flow, Herkunft, ersetzten Dateien, Optionen, Basis und `loomux_version`; der Claim ist exklusiv).

---

### Task 5: geänderte Dateien zu Laufbeginn

**Files:**
- Modify: `internal/gitwork/gitwork.go` (hinzu: `ChangedFiles`, `parseStatus`)
- Create (Port): `internal/gitwork/changed_test.go` aus `d7041e5:internal/gitwork/changed_test.go`
- Modify: `internal/gitwork/gitwork_test.go` (`repo()` pinnt `status.renames`)

**Interfaces:**
- Produces: `gitwork.ChangedFiles(root string) ([]string, error)`.

- [ ] **Step 1: Test portieren** (`changed_test.go` per sed) und in `repo()` von `gitwork_test.go` nach `user.name` einfügen:

```go
	// Pinned locally: under a global status.renames=false, git status reports a
	// rename as an addition and a deletion, and the rename tests would pass
	// without ever reaching parseStatus's rename branch.
	run(t, root, "config", "status.renames", "true")
```

Passen Helfernamen im portierten Test nicht zu `gitwork_test.go` in loomux (`repo`, `commit`, `run`), den portierten Test angleichen, nicht die bestehenden Helfer.

- [ ] **Step 2: Rot sehen** — `go test ./internal/gitwork/ -run Changed` → FAIL (`undefined: ChangedFiles`).

- [ ] **Step 3: Implementieren** — `ChangedFiles` und `parseStatus` wörtlich aus `git -C "$UL" diff 471eab4 d7041e5 -- internal/gitwork/gitwork.go` ans Ende von `gitwork.go`. Die Meldung wird `"%s: %w -- run loomux in a working tree of its own"`. `parseStatus` enthält `"\x00"`: per Edit einfügen und mit `grep -n 'x00' internal/gitwork/gitwork.go` nachlesen.

- [ ] **Step 4: Grün, Commit** — `feat(gitwork): list the files changed at a run's start`.

---

### Task 6: Modell-Port, Fake, Rollenauflösung, Werkzeugprofile

**Files:**
- Create (Port): `internal/flow/model/{model,fake}.go` und Tests aus `d7041e5:internal/model/`
- Create: `internal/flow/model/resolve.go`, `resolve_test.go`, `tools.go`, `tools_test.go`

**Interfaces:**
- Consumes: `config.Agent`, `config.ModelSpec` (Task 1).
- Produces: `model.Model`, `model.Request`, `model.Reply`, `model.ReplyType` (+ Konstanten), `model.Fake`, `model.NewFake(...Answer)`, `model.Answer`; `model.DefaultProvider = "claude"`; `model.Resolved{Role, RoleFrom, Name, ModelFrom, Provider, Model string}`, `(Resolved).Label() string`, `(Resolved).Via() string`; `model.Resolve(agent config.Agent, nodeRole, flowRole string) (Resolved, error)`; `model.Tools(profile string, servers []string) ([]string, error)`.

- [ ] **Step 1: Port** von `model.go`, `fake.go` und Tests. Paketkommentar: „Package model is the port every agent node reaches a model through. Tests hand the runner the Fake; the adapters for the providers' CLIs arrive later.“

- [ ] **Step 2: Failing tests** `resolve_test.go`:

```go
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
```

`tools_test.go`: die Tests aus `d7041e5:internal/flowcfg/tools_test.go`, umgestellt auf `Tools`; dazu ein Test, dass zwei Aufrufe verschiedene Slices liefern (Änderung am einen ändert den anderen nicht).

- [ ] **Step 3: Rot sehen**, dann `resolve.go`:

```go
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
```

`tools.go` ist `d7041e5:internal/flowcfg/tools.go` mit zwei Änderungen: `package model`, und `profiles` wird eine Funktion `func profiles() map[string][]string { return map[string][]string{…} }` statt einer Paketvariable, damit kein Init auf dem Hook-Pfad eine Karte baut; `joinKeys` wird lokal mit `slices.Sorted(maps.Keys(…))` geschrieben.

- [ ] **Step 4: Grün, 100 %, Commit** — `feat(flow): add the model port, the fake and role resolution`.

---

### Task 7: Verträge, Bedingungen, Platzhalter

**Files:**
- Create (Port): `internal/flow/{doc,types,block,catalog,coerce,graph,text}.go`, `internal/flow/expr/*.go`, `internal/flow/tmpl/*.go` samt Tests aus `d7041e5`.

**Interfaces:**
- Consumes: `config.Agent` (1), `model.Model` (6), `runs.Baseline` (4).
- Produces (Änderungen gegenüber M1):
  - `flow.Baseline = runs.Baseline` (Alias).
  - `flow.Node{Name, Kind, Role string; MaxVisits Cap; Keys, Raw map[string]any}` — `Model` heißt jetzt `Role`.
  - `flow.Edge{From, To string; When Condition; Text string; OnError bool}` — `Text` ist das `when`, wie es in der Datei steht.
  - `flow.Graph{Name, File string; Texts fs.FS; Origin string; Overlays []string; Start, Role string; Params, State map[string]Field; Nodes []Node; Edges []Edge}` — `Dir` entfällt, `Texts` hält `flow.toml`, `instructions/`, `questions/`; `Model` heißt `Role`.
  - `flow.Env{Root string; Params Params; Answer *string; Baseline *Baseline; Model model.Model; Agent config.Agent}`.
  - `flow.Definition{Text []byte; Role, Model, Effort, Profile string; Tools []string}`.
  - `flow.ReadText(texts fs.FS, t Text) ([]byte, error)` — liest per `fs.ReadFile`, wandelt CRLF in LF.

- [ ] **Step 1: Port** aller Dateien der drei Pakete per sed.

- [ ] **Step 2: Failing tests** in `text_test.go`:

```go
func TestReadTextTurnsCRLFIntoLF(t *testing.T) {
	texts := fstest.MapFS{"questions/q.md": {Data: []byte("Ship it?\r\nReally?\r\n")}}
	got, err := ReadText(texts, Text{Key: "question", Path: "questions/q.md"})
	if err != nil || string(got) != "Ship it?\nReally?\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestReadTextNamesTheMissingFile(t *testing.T) {
	_, err := ReadText(fstest.MapFS{}, Text{Key: "instruction", Path: "instructions/x.md"})
	if err == nil || !strings.Contains(err.Error(), "reading instruction instructions/x.md") {
		t.Fatalf("err = %v", err)
	}
}
```

Beide per Write (Backslashes), danach `grep -n 'r.n' internal/flow/text_test.go`. Die übrigen M1-Tests von `ReadText` werden von Verzeichnis auf `fstest.MapFS` umgestellt.

- [ ] **Step 3: Rot sehen**, dann:
  - `types.go`: `Baseline` wird `type Baseline = runs.Baseline` mit dem Kommentar „Baseline is the run's starting point; it is defined beside the marker that carries it (internal/flow/runs).“
  - `graph.go`: Felder wie oben; der Kommentar an `Texts`: „the flow's files as the run reads them: its folder, or a bundled folder with a project's overlay laid over it“.
  - `block.go`: Imports `internal/config` statt `flowcfg`, `internal/flow/model`; `Env.Agent config.Agent`; `Definition.Role string` mit Kommentar „the role the node plays; empty without one“.
  - `text.go`:
    ```go
    // ReadText returns a text's bytes: the file among the flow's texts, or the
    // text itself when it is inline. A carriage return before a line feed is
    // dropped on the way in, so the bytes -- and the definition hash taken over
    // them -- do not depend on the editor or on core.autocrlf.
    func ReadText(texts fs.FS, t Text) ([]byte, error) {
    	if t.Path == "" {
    		return []byte(t.Inline), nil
    	}
    	raw, err := fs.ReadFile(texts, t.Path)
    	if err != nil {
    		return nil, fmt.Errorf("reading %s %s: %w", t.Key, t.Path, err)
    	}
    	return bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n")), nil
    }
    ```
  - `doc.go`: „The loader that fills a Graph from a flow's folder lives in internal/flow/load …“.
  - Tests, die `Graph{Dir: …}` oder `Node{Model: …}` bauen, auf `Texts`/`Role` umstellen.

- [ ] **Step 4: Grün** — `go test ./internal/flow/ ./internal/flow/expr/ ./internal/flow/tmpl/ -cover -count=1` → PASS, 100 %.

- [ ] **Step 5: Commit** — `feat(flow): add the flow contracts, conditions and placeholders`.

---

### Task 8: Bausteine `agent`, `gate`, `exit`

**Files:**
- Create (Port): `internal/flow/blocks/{doc,agent,gate,exit}.go` und Tests aus `d7041e5:internal/blocks/`.

**Interfaces:**
- Consumes: `flow.*` (7), `model.Resolve`, `model.Tools` (6).
- Produces: `blocks.Agent{}`, `blocks.Gate{}`, `blocks.Exit{}` (unverändert `flow.Block`).

- [ ] **Step 1: Port** per sed.

- [ ] **Step 2: Failing test** in `agent_test.go`, der die Rolle in die Definition bringt:

```go
func TestAgentDefinitionCarriesTheRoleAndTheBoundModel(t *testing.T) {
	graph := &flow.Graph{Texts: fstest.MapFS{"instructions/d.md": {Data: []byte("Draft.")}}, Role: "planner"}
	node := flow.Node{Name: "draft", Kind: "agent", Role: "reviewer",
		Keys: map[string]any{"instruction": "instructions/d.md", "reply": map[string]any{"ok": "bool"}}}
	env := flow.Env{Agent: config.Agent{
		Models: map[string]config.ModelSpec{"gemini": {Provider: "agy", Model: "gemini-3"}},
		Roles:  map[string]string{"reviewer": "gemini"},
	}}
	got, err := Agent{}.Define(graph, node, env)
	if err != nil || got.Role != "reviewer" || got.Model != "agy:gemini-3" {
		t.Fatalf("got %+v, %v", got, err)
	}
}
```

- [ ] **Step 3: Rot sehen**, dann in `agent.go`:
  - `flowcfg.Resolved` → `model.Resolved`; `env.Agent.Resolve(node.Model, graph.Model)` → `model.Resolve(env.Agent, node.Role, graph.Role)`; `flowcfg.Tools` → `model.Tools`.
  - `Definition` bekommt `Role: resolved.Role`.
  - `flow.ReadText(graph.Dir, …)` → `flow.ReadText(graph.Texts, …)` (auch in `gate.go`, `exit.go`).
  - Meldungen: `"instruction must name a file under instructions/"`, in `gate.go` `"question must name a file under questions/"`; `"node %q needs a model; M1 registers no adapter yet"` → `"node %q needs a model, and this run has none"`.
  - `doc.go`: „Package blocks holds the node kinds the runtime ships: agent, gate and exit.“
  - Die Tests auf `Texts: fstest.MapFS{…}` und `Role` umstellen.

- [ ] **Step 4: Grün, 100 %, Commit** — `feat(flow): add the agent, gate and exit blocks`.

---

### Task 9: der Runner

**Files:**
- Create (Port): `internal/flow/runner/{doc,runner,walk}.go` und Tests (`runner_test.go`, `gate_test.go`, `retrace_test.go`) aus `d7041e5:internal/runner/`.

**Interfaces:**
- Consumes: `flow.*`, `journal.*`.
- Produces: unverändert `runner.New(Options) *Runner`, `(*Runner).Run(ctx)`, `(*Runner).Resume(ctx, answer *string)`, `runner.Options`, `runner.Result`, `runner.Clock`, `runner.Journal`.

- [ ] **Step 1: Port** per sed.

- [ ] **Step 2: Failing test** in `runner_test.go`: ein Graph mit einem Agentenknoten `role = "reviewer"` gegen den Fake schreibt `"role":"reviewer"` ins Journal, ein Torknoten `"role":null`. Den Test nach dem Muster der vorhandenen Journal-Assertions in `runner_test.go` schreiben (ein Test-Journal im Speicher, dessen Einträge verglichen werden).

- [ ] **Step 3: Rot sehen**, dann in `walk.go`:
  - `write` setzt `Role: optional(definition.Role)`.
  - `warnOnChangedDefinition`: der Zweig `if cached.DefinitionHash == nil { return nil }` samt Python-Kommentar entfällt; jede Zeile trägt den Hash. Ein Journal, dessen Zeile ihn als `null` trägt, ist damit „geändert“ und wird gewarnt; dafür einen Test (eine Zeile mit `definition_hash: null` im Test-Journal → Warnung).
  - `optional` behält seinen Zweck; der Kommentar wird: „optional is the journal's spelling of nothing here: null, not an empty string that would claim a value was recorded.“
  - `doc.go`: `internal/flowload` → `internal/flow/load`.
  - Tests: handgebaute Graphen mit `Texts: fstest.MapFS{…}` statt `Dir`, `Node{Role: …}` statt `Model`.

- [ ] **Step 4: Grün, 100 %, Commit** — `feat(flow): add the runner with resume and replay`.

---

### Task 10: Flows laden — Ordner, Katalog, Overlay

**Files:**
- Create (Port): `internal/flow/load/{doc,decl,check,load,params,discover}.go` und Tests aus `d7041e5:internal/flowload/`; Testdaten aus `d7041e5:internal/flowload/testdata/` nach `internal/flow/load/testdata/`, umgebaut ins Ordnerformat.
- Create: `internal/flow/load/overlay.go`, `overlay_test.go`, `find_test.go`.

**Interfaces:**
- Consumes: `config.FlowSettings`, `config.IsFlowName`, `config.IsIdentifier` (1); `flow.*`, `flow/expr`, `flow/tmpl` (7).
- Produces:
  - `load.Dir = ".loomux/flows"`.
  - `load.OriginProject = "project"`, `load.OriginBundled = "bundled"`, `load.OriginOverlay = "bundled+overlay"`, `load.OriginHides = "project (hides bundled)"`.
  - `load.Found{Name, Origin, File string; Files fs.FS; Overlays []string; Warnings []string}`.
  - `load.Find(root string, bundled fs.FS, settings config.FlowSettings, name string) (Found, error)`.
  - `load.Load(found Found, catalog *flow.Catalog) (*flow.Graph, error)`.
  - `load.Entry{Name, Origin, Problem string; Default bool; Warnings []string}`.
  - `load.List(root string, bundled fs.FS, settings config.FlowSettings, catalog *flow.Catalog) []Entry`.
  - `load.SettingsWarnings(bundled fs.FS, settings config.FlowSettings) []string`.
  - `load.Params(graph, options) (flow.Params, error)` (unverändert), `load.Findings`.

- [ ] **Step 1: Testdaten umbauen.** Für jeden Ordner unter `d7041e5:internal/flowload/testdata/<fall>/`:
  - die TOML-Datei (`f.toml`, `example.toml`, `planning.toml`) wird `testdata/<fall>/flow.toml`;
  - `[flow] name` wird gestrichen (der Name ist der Ordner);
  - Fragen ziehen nach `questions/` (im Fall `example`: `instructions/approve-question.md` → `questions/approve.md`, der Pfad in `flow.toml` mit);
  - `model = …` an Knoten und in `[flow]` wird `role = …`; der Fall `stage7_model` entfällt, denn Stufe 7 gibt es nicht mehr;
  - Ordnernamen mit `_` (`stage1_edges` …) werden `stage1-edges` usw., weil der Ordnername die Flow-Namensregel erfüllen muss.
  
  Die Tests greifen auf einen Fall über `testFlow(t, "stage1-edges")` zu:
  ```go
  func testFlow(t *testing.T, name string) Found {
  	t.Helper()
  	return Found{Name: name, Origin: OriginProject, File: name + "/flow.toml", Files: os.DirFS(filepath.Join("testdata", name))}
  }
  ```

- [ ] **Step 2: Port** der Quellen per sed; die M1-Tests auf `testFlow`, `Load(found, catalog)` und die neuen Ordnernamen umstellen.

- [ ] **Step 3: Failing tests für das Format** (`load_test.go`), je ein Flow im TempDir oder als `fstest.MapFS`:

```go
func TestLoadRefusesUnknownKeys(t *testing.T) {
	for name, tc := range map[string]struct{ toml, want string }{
		"top level": {"schema_version = 1\ncolour = 1\n[flow]\nstart = \"a\"\n", `unknown key "colour"`},
		"flow":      {"schema_version = 1\n[flow]\nstart = \"a\"\nmodel = \"w\"\n", `[flow] model: a flow names a role, write role = ...`},
		"flow name": {"schema_version = 1\n[flow]\nstart = \"a\"\nname = \"x\"\n", `[flow] name: a flow's name is its folder's name`},
		"node":      {"schema_version = 1\n[flow]\nstart = \"a\"\n[[node]]\nname = \"a\"\nkind = \"exit\"\ncode = 4\nmessage = \"m\"\nmodel = \"w\"\n", `node "a": model: a node names a role, write role = ...`},
		"edge":      {"schema_version = 1\n[flow]\nstart = \"a\"\n[[node]]\nname = \"a\"\nkind = \"exit\"\ncode = 4\nmessage = \"m\"\n[[edge]]\nfrom = \"a\"\nto = \"END\"\nif = \"x\"\n", `edge 1: unknown key "if"`},
		"role name": {"schema_version = 1\n[flow]\nstart = \"a\"\nrole = \"a-b\"\n", `role "a-b" is not a name`},
	} {
		t.Run(name, func(t *testing.T) {
			found := Found{Name: "f", File: "f/flow.toml", Files: fstest.MapFS{"flow.toml": {Data: []byte(tc.toml)}}}
			_, err := Load(found, testCatalog(t))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestLoadRefusesABackslashInATextPath(t *testing.T) {
	flowFile := "schema_version = 1\n[flow]\nstart = \"a\"\n[state]\nanswer = { type = \"string\", default = \"\" }\nanswer_text = { type = \"string\", default = \"\" }\n" +
		"[[node]]\nname = \"a\"\nkind = \"gate\"\nquestion = 'questions\\q.md'\nchoices = [\"yes\", \"no\"]\nanswer = \"answer\"\n[[edge]]\nfrom = \"a\"\nto = \"END\"\n"
	found := Found{Name: "f", File: "f/flow.toml", Files: fstest.MapFS{"flow.toml": {Data: []byte(flowFile)}, "questions/q.md": {Data: []byte("?")}}}
	_, err := Load(found, testCatalog(t))
	if err == nil || !strings.Contains(err.Error(), `question "questions\\q.md" must use / between folders`) {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadHoldsEveryTextToItsFolder(t *testing.T) {
	// An instruction under questions/, a question under instructions/, a path
	// that leaves the folder and an absolute one: all four refused.
	…
}
```

(Alle drei per Write, sie enthalten Backslashes; danach `grep -n 'q.md' internal/flow/load/load_test.go`.) Den vierten Test mit den vier Fällen ausschreiben; die erwarteten Meldungen: `instruction "questions/d.md" must be a file under instructions/`, `question "instructions/q.md" must be a file under questions/`, `instruction "../x.md" must be a file under instructions/`, `instruction "/abs.md" must be a file under instructions/`. `testCatalog(t)` baut `flow.NewCatalog([]flow.Block{blocks.Agent{}, blocks.Gate{}, blocks.Exit{}}, nil)`; steht er im M1-Test schon unter anderem Namen, diesen nehmen.

- [ ] **Step 4: Rot sehen**, dann in `decl.go`:
  - `nameRule` bleibt für Knoten, Felder, Parameter und Rollen; die Flow-Namensregel prüft `Find` (Schritt 7).
  - neue Listen:
    ```go
    // The keys each part of a flow file may hold. A key the reader does not
    // know is a finding: `instructon` would otherwise be an instruction nobody
    // reads, and a later format would find its key taken by a typo.
    var (
    	topKeys    = []string{"edge", "flow", "node", "params", "schema_version", "state"}
    	flowKeys   = []string{"role", "start"}
    	sharedKeys = []string{"kind", "max_visits", "name", "role"}
    	edgeKeys   = []string{"from", "on_error", "to", "when"}
    )
    ```
  - `declarations` prüft `topKeys`; `head` prüft `flowKeys` und meldet `name` und `model` mit eigener Meldung (siehe Tests), liest `graph.Role` statt `graph.Model` und prüft die Rolle mit `checkName(file, "role", …)`; `nodes` liest `role` statt `model` und meldet `model` am Knoten eigens; `edges` prüft `edgeKeys` und setzt `flow.Edge.Text = when`.
  - `version`: die Meldung wird `"schema_version %v is unknown; loomux knows version %d"` (und ebenso bei fehlender Version).
  - `check.go`: `stage` verliert den Parameter `agent`; `stage7` entfällt samt Aufruf; `stage3` prüft vor dem Lesen jeden Text mit Pfad:
    ```go
    // textFolders is where each kind of text lives. A fixed place per kind is
    // what lets a project overlay a bundled flow's text file by file.
    var textFolders = map[string]string{"instruction": "instructions/", "question": "questions/"}

    // textPath says what is wrong with a text's path, or "" when nothing is.
    func textPath(t flow.Text) string {
    	if strings.Contains(t.Path, `\`) {
    		return fmt.Sprintf("%s %q must use / between folders", t.Key, t.Path)
    	}
    	folder, known := textFolders[t.Key]
    	if !known {
    		folder = "instructions/"
    	}
    	if !fs.ValidPath(t.Path) || !strings.HasPrefix(t.Path, folder) {
    		return fmt.Sprintf("%s %q must be a file under %s", t.Key, t.Path, folder)
    	}
    	return ""
    }
    ```
    (per Write: die Zeile mit `` `\` `` hat einen Backslash). `stage3` meldet `node %q: <textPath>` und liest nur Pfade ohne Befund, mit `flow.ReadText(made.graph.Texts, text)`; `stage4` liest ebenfalls über `made.graph.Texts`.
  - `load.go`:
    ```go
    // Load reads a found flow and returns the graph it stands for, or the
    // findings of the first stage that had any.
    func Load(found Found, catalog *flow.Catalog) (*flow.Graph, error) {
    	raw, err := fs.ReadFile(found.Files, "flow.toml")
    	if err != nil {
    		return nil, Findings{found.File + ": " + err.Error()}
    	}
    	var doc map[string]any
    	if _, err := toml.Decode(string(raw), &doc); err != nil {
    		return nil, Findings{found.File + ": " + err.Error()}
    	}
    	made, findings := declarations(found.File, doc)
    	made.graph.Name, made.graph.File, made.graph.Texts = found.Name, found.File, found.Files
    	made.graph.Origin, made.graph.Overlays = found.Origin, found.Overlays
    	if len(findings) > 0 {
    		return nil, findings
    	}
    	for _, check := range []stage{stage2, stage3, stage4, stage5, stage6} {
    		if findings := check(found.File, made, catalog); len(findings) > 0 {
    			return nil, findings
    		}
    	}
    	return made.graph, nil
    }
    ```
    Die Stelle in `head`, die den Flow-Namen prüfte, entfällt.
  - `doc.go`: sechs Stufen statt sieben, „finds flows by name in the project and in the catalog it is handed“.

- [ ] **Step 5: Grün** für die Formattests und alle portierten Stufentests.

- [ ] **Step 6: Failing tests für Auffindung** (`find_test.go`). Ein Katalog als `fstest.MapFS` mit `example/flow.toml`, `example/instructions/draft.md`, `example/questions/approve.md` (Inhalt aus `testdata/example`), Projekte im TempDir:

| Test | Aufbau | Erwartung |
|---|---|---|
| `TestFindABundledFlow` | kein Projektordner | `Origin == "bundled"`, `File == "bundled:example/flow.toml"`, `fs.ReadFile(found.Files, "questions/approve.md")` liest das Original |
| `TestFindAProjectFlow` | `.loomux/flows/mine/flow.toml` | `Origin == "project"`, `File == ".loomux/flows/mine/flow.toml"` |
| `TestFindHidesABundledFlowOnlyWithAnOverride` | `.loomux/flows/example/flow.toml` | ohne Override: `Origin "bundled"`, `Warnings == [".loomux/flows/example is ignored: [flow] overrides does not name it"]`; mit `Overrides: ["example"]`: `Origin "project (hides bundled)"` |
| `TestFindOverlaysSingleFiles` | `.loomux/flows/example/questions/approve.md` = „Really?“, Override gesetzt | `Origin "bundled+overlay"`, `Overlays == ["questions/approve.md"]`, `questions/approve.md` liest „Really?“, `instructions/draft.md` das Original |
| `TestFindIgnoresAnOverlayWithoutAnOverride` | wie oben, ohne Override | `Origin "bundled"`, Warnung wie oben |
| `TestFindRefusesAnOverlayOfNothing` | `.loomux/flows/example/questions/nope.md`, Override | Fehler `.loomux/flows/example overlays nothing: bundled flow "example" has no questions/nope.md` |
| `TestFindRefusesAnOverlayWithoutABundledFlow` | `.loomux/flows/ghost/questions/q.md` | Fehler `.loomux/flows/ghost overlays nothing: there is no bundled flow "ghost"` |
| `TestFindRefusesAStrayFileInAProjectFolder` | `.loomux/flows/example/README.md` (+ Override) | Fehler `.loomux/flows/example/README.md is neither flow.toml nor a file under instructions/ or questions/` |
| `TestFindRefusesAnEmptyProjectFolder` | leerer Ordner `.loomux/flows/mine/` | Fehler `.loomux/flows/mine holds no flow.toml and nothing to overlay` |
| `TestFindRefusesABadName` | `Find(…, "Dev")` | Fehler `"Dev" is not a flow name; a flow name is [a-z][a-z0-9-]*` |
| `TestFindNamesTheKnownFlows` | `Find(…, "nope")` | Fehler `no flow named "nope"; known flows: example, mine` |

- [ ] **Step 7: Rot sehen**, dann `discover.go` (ersetzt die M1-Fassung) und `overlay.go`:

```go
package load

// Dir is where a project keeps its flows, one folder each, below the root.
const Dir = ".loomux/flows"

// Where a found flow came from, as list, show and the marker spell it.
const (
	OriginProject = "project"
	OriginBundled = "bundled"
	OriginOverlay = "bundled+overlay"
	OriginHides   = "project (hides bundled)"
)

// Found is a flow located by name and ready to load.
type Found struct {
	Name     string
	Origin   string
	File     string   // how messages name its flow.toml
	Files    fs.FS    // flow.toml, instructions/ and questions/ as a run reads them
	Overlays []string // the project's files laid over a bundled flow, sorted
	Warnings []string // a project folder that was ignored, and why
}

// Find locates a flow: a project folder under Dir, a bundled one, or a
// bundled one with the project's files laid over it. A project may hide or
// overlay a bundled flow only when [flow] overrides names it; otherwise the
// bundled flow runs and the project folder is named in a warning. That is the
// guard's second line: a folder an agent wrote cannot take a bundled flow's
// gates away, whichever binary judged the write.
func Find(root string, bundled fs.FS, settings config.FlowSettings, name string) (Found, error) {
	if !config.IsFlowName(name) {
		return Found{}, fmt.Errorf("%q is not a flow name; a flow name is %s", name, config.FlowNameRule)
	}
	isBundled := exists(bundled, name+"/flow.toml")
	folder := filepath.Join(root, filepath.FromSlash(Dir), name)
	shown := Dir + "/" + name
	kind, overlays, err := classify(folder, shown)
	switch {
	case err != nil:
		return Found{}, err
	case kind == noFolder && isBundled:
		return bundledFound(bundled, name), nil
	case kind == noFolder:
		return Found{}, fmt.Errorf("no flow named %q; known flows: %s", name, offer(Names(root, bundled)))
	case kind == fullFolder && !isBundled:
		return Found{Name: name, Origin: OriginProject, File: shown + "/flow.toml", Files: os.DirFS(folder)}, nil
	case !isBundled:
		return Found{}, fmt.Errorf("%s overlays nothing: there is no bundled flow %q", shown, name)
	case !settings.Allows(name):
		found := bundledFound(bundled, name)
		found.Warnings = []string{shown + " is ignored: [flow] overrides does not name it"}
		return found, nil
	case kind == fullFolder:
		return Found{Name: name, Origin: OriginHides, File: shown + "/flow.toml", Files: os.DirFS(folder)}, nil
	}
	for _, file := range overlays {
		if !exists(bundled, name+"/"+file) {
			return Found{}, fmt.Errorf("%s overlays nothing: bundled flow %q has no %s", shown, name, file)
		}
	}
	base := bundledFound(bundled, name)
	base.Origin, base.Overlays = OriginOverlay, overlays
	base.Files = overlayFS{base: base.Files, top: os.DirFS(folder), files: overlays}
	return base, nil
}
```

Dazu in `discover.go`: `bundledFound(bundled, name)` (`fs.Sub(bundled, name)`; `Sub` scheitert nur an einem ungültigen Namen, den `IsFlowName` ausschließt — der Fehler wird mit `//` begründet verworfen), `exists(fsys, path) bool` (`fs.Stat`), `Names(root, bundled) []string` (Vereinigung der Ordner unter `Dir` und im Katalog, sortiert, ohne Doppel), `offer` (wie M1), und `classify`:

```go
type folderKind int

const (
	noFolder folderKind = iota
	fullFolder
	overlayFolder
)

// classify says what a project folder holds: nothing (absent), a flow of its
// own (it has a flow.toml), or files to lay over a bundled flow. Anything
// else in it is refused by name, so a stray README is a message and not a
// silent overlay.
func classify(folder, shown string) (folderKind, []string, error) {
	if _, err := os.Stat(folder); errors.Is(err, fs.ErrNotExist) {
		return noFolder, nil, nil
	}
	var files []string
	full := false
	err := fs.WalkDir(os.DirFS(folder), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		switch {
		case path == "flow.toml":
			full = true
		case strings.HasPrefix(path, "instructions/") || strings.HasPrefix(path, "questions/"):
			files = append(files, path)
		default:
			return fmt.Errorf("%s/%s is neither flow.toml nor a file under instructions/ or questions/", shown, path)
		}
		return nil
	})
	switch {
	case err != nil:
		return noFolder, nil, err
	case full:
		return fullFolder, nil, nil
	case len(files) == 0:
		return noFolder, nil, fmt.Errorf("%s holds no flow.toml and nothing to overlay", shown)
	}
	slices.Sort(files)
	return overlayFolder, files, nil
}
```

Ein Projekt-Flow mit eigenem `flow.toml` darf daneben beliebige Dateien unter `instructions/` und `questions/` haben; eine Streudatei neben `flow.toml` wird ebenso abgelehnt (die Walk-Funktion unterscheidet nicht). `overlay.go`:

```go
package load

import (
	"io/fs"
	"slices"
)

// overlayFS reads a bundled flow with some of its files taken from a project
// folder instead: the files Find checked the bundled flow has.
type overlayFS struct {
	base, top fs.FS
	files     []string
}

func (o overlayFS) Open(name string) (fs.File, error) {
	if slices.Contains(o.files, name) {
		return o.top.Open(name)
	}
	return o.base.Open(name)
}
```

- [ ] **Step 8: Failing tests für `List` und `SettingsWarnings`**: `List` nennt jeden Flow einmal, sortiert, mit Herkunft, `Default` bei `settings.Default`, einem Flow, der nicht lädt, mit `Problem`, und einem ignorierten Ordner mit seiner Warnung; `SettingsWarnings` meldet `[flow] overrides names "ghost", which no bundled flow has` und `[flow] default names "ghost", which is no flow here` (der zweite braucht `root`, darum `SettingsWarnings(root, bundled, settings)`).

- [ ] **Step 9: Rot sehen, implementieren, Grün** — `List` ruft je Name `Find` und dann `Load`; ein Fehler von `Find` oder `Load` wird `Problem`.

- [ ] **Step 10: 100 %, Commit** — `feat(flow): load flows from folders, the catalog and overlays` (Rumpf: ein Flow ist ein Ordner mit festen Plätzen für Anweisungen und Fragen; unbekannte Schlüssel, Backslashes und Pfade außerhalb der Ordner sind Ladefehler; Projekt-Flows verdecken oder überlagern einen mitgelieferten nur, wenn `[flow] overrides` ihn nennt; CRLF-Texte laden als LF).

---

### Task 11: der Katalog

**Files:**
- Create: `flows/embed.go`, `flows/catalog_test.go` (Paket `flows_test`), `flows/embed_test.go` (Paket `flows`)
- Create: `flows/catalog/example/{flow.toml,README.md,instructions/draft.md,questions/approve.md,_test/script.toml,_test/journal.jsonl}`
- Modify: `.gitattributes`, `AGENTS.md` („Where things live“)

**Interfaces:**
- Consumes: `load.*` (10), `blocks.*` (8), `runner.*` (9), `model.Fake` (6), `journal.Append` (3).
- Produces: `flows.FS() fs.FS` (eine Ebene tiefer als `catalog/`: `example/flow.toml`), `flows.Names() []string`.

- [ ] **Step 1: Failing test** `flows/embed_test.go`:

```go
package flows

import (
	"io/fs"
	"reflect"
	"strings"
	"testing"
)

func TestEveryCatalogFolderIsEmbeddedWithoutItsTests(t *testing.T) {
	if got := Names(); !reflect.DeepEqual(got, []string{"example"}) {
		t.Fatalf("Names() = %v", got)
	}
	err := fs.WalkDir(FS(), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(path, "_test") {
			t.Errorf("%s is embedded", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(FS(), "example/questions/approve.md"); err != nil {
		t.Fatal(err)
	}
}
```

`Names()` muss später jeden Ordner unter `flows/catalog/` liefern; der Test nennt die Liste wörtlich, damit ein neuer Katalog-Flow ihn bewusst ändert.

- [ ] **Step 2: Rot sehen**, dann `flows/embed.go`:

```go
// Package flows is the catalog of flows loomux ships: one folder per flow
// under catalog/, contributed by pull request, loaded like a project's own.
//
// go:embed walks catalog/ and leaves out every name that starts with "_" or
// ".", so a flow's _test/ folder -- its script and golden journal -- stays
// out of the binary without a pattern per folder.
package flows

import (
	"embed"
	"io/fs"
	"slices"
)

//go:embed catalog
var catalog embed.FS

// FS is the catalog as the loader reads it: one folder per flow at the top.
func FS() fs.FS {
	// Sub fails only for a name that is not a valid path, and "catalog" is one.
	sub, _ := fs.Sub(catalog, "catalog")
	return sub
}

// Names are the bundled flows, sorted.
func Names() []string {
	// The catalog is compiled in, so reading its top never fails.
	entries, _ := fs.ReadDir(FS(), ".")
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	return names
}
```

(`embed.FS` als Paketvariable parst nichts; die Regel „keine Paketvariable parst eingebettete Daten“ hält.)

- [ ] **Step 3: Der Beispiel-Flow.** `flows/catalog/example/flow.toml` ist `testdata/example/flow.toml` aus Task 10 (ohne `[flow] name`, Frage unter `questions/approve.md`), dazu `role = "writer"` am Knoten `draft`, damit das Beispiel Rollen zeigt. `instructions/draft.md`: „Write a draft plan. You have {{max_rounds}} rounds.“; `questions/approve.md`: „Approve the plan after {{count}} rounds?“. `README.md`:

```markdown
# example

The template to copy for a flow of your own: an agent drafts, a human
approves, a rejection ends the run with code 4.

- **Roles:** `writer` (the node `draft`). Bind it in `.loomux/config.toml`
  with `[agent.roles] writer = "<model>"`; unbound, it runs on `[agent] default`
  or the claude CLI's own default.
- **Parameters:** `max_rounds` (int, default 5).
- **Gate:** `approve`, choices `yes` and `no`.

Until loomux has a model adapter, `loomux flow run example` refuses to start;
the flow runs in its test, `flows/catalog/example/_test/`.
```

- [ ] **Step 4: Failing test** `flows/catalog_test.go` — das Skriptformat und die Fahrt:

```go
package flows_test

// script is a catalog flow's _test/script.toml: the run's options, then one
// step per visit that needs an answer from outside, in the order the run
// asks. An agent step carries the reply fields, or an error the fake model
// raises instead; a gate step carries the answer and names its node, which
// the run must be paused at.
type script struct {
	Options map[string]string `toml:"options"`
	Steps   []step            `toml:"step"`
}

type step struct {
	Node   string         `toml:"node"`
	Reply  map[string]any `toml:"reply"`
	Tokens int            `toml:"tokens"`
	Error  string         `toml:"error"`
	Answer *string        `toml:"answer"`
}
```

Der Test `TestEveryCatalogFlowRunsToItsGoldenJournal`:
1. geht über `flows.Names()`; je Flow liest er `catalog/<name>/_test/script.toml` vom Datenträger (nicht aus dem Binary) und verlangt `catalog/<name>/README.md`;
2. baut `load.Find(t.TempDir(), flows.FS(), config.FlowSettings{}, name)` und `load.Load`;
3. reiht alle Agentenschritte in `model.NewFake(...)` ein (Reply mit `Tokens`, oder `Err: errors.New(step.Error)`), in Skriptreihenfolge;
4. fährt `runner.New(runner.Options{Graph: g, Catalog: c, Journal: fileJournal{path}, Env: flow.Env{Params: params, Model: fake}, Clock: halfSecondClock()})` mit `Run`, und solange das Ergebnis `paused` ist, den nächsten Torschritt: dessen `Node` muss `result.Node` sein, dann `Resume(ctx, step.Answer)`;
5. verlangt am Ende, dass jeder Schritt verbraucht ist (der Fake hat keine Antwort übrig, kein Torschritt bleibt);
6. vergleicht das Journal Byte für Byte mit `catalog/<name>/_test/journal.jsonl`; mit `-update` schreibt er es.

`halfSecondClock` und `fileJournal` stehen im Testpaket (die Uhr wie im M1-Harness: jede Abfrage 500 ms weiter ab `time.Unix(0, 0)`). `-update` als `var update = flag.Bool("update", false, "write the golden journals instead of comparing")`.

`flows/catalog/example/_test/script.toml`:

```toml
# The run drafts once, asks, and is rejected.
[[step]]
node = "draft"
reply = { verdict = "done", count = 2 }
tokens = 120

[[step]]
node = "approve"
answer = "no: too thin"
```

- [ ] **Step 5: Rot sehen** (kein Journal), `go test ./flows -update` einmal, dann `go test ./flows -count=1` → PASS. Das erzeugte `journal.jsonl` lesen: vier Zeilen (draft ok, approve paused, approve ok, stop error mit „rejected after 2 rounds“), `"role":"writer"` an `draft`, `"model":"claude:cli-default"`.

- [ ] **Step 6: `.gitattributes` und `AGENTS.md`.** In `.gitattributes`:

```
# A catalog flow's golden journal is compared byte for byte.
flows/catalog/*/_test/** text eol=lf
```

In `AGENTS.md` unter „Where things live“:

```markdown
- `flows/catalog/<name>/` is the catalog of flows loomux ships. A
  contribution is data only: `flow.toml`, `instructions/`, `questions/`, a
  `README.md` and `_test/` with `script.toml` and the golden
  `journal.jsonl` that `go test ./flows` replays. A flow that needs a block
  the runtime lacks is a separate pull request with Go code.
```

- [ ] **Step 7: Commit** — `feat(flow): ship a catalog of flows with an example`.

---

### Task 12: `loomux flow`

**Files:**
- Create (Port): `internal/cli/flow.go` (aus `cmd/flow/main.go`), `flow_session.go` (aus `session.go`), `flow_run.go` (aus `run.go`), `flow_resume.go` (aus `resume.go`), `flow_show.go` (aus `show.go`), `flow_journal.go` (aus `journal.go`) und Tests `flow_*_test.go` aus `cmd/flow/*_test.go`.
- Modify: `internal/cli/commands.go` (`"flow": flowCommand`), `internal/cli/imports_test.go` (nichts, aber prüfen, dass `cli` weiter alles erreicht, was es muss).

**Interfaces:**
- Consumes: alles aus 1–11; `hosts.FindRoot`; `Version`, `Channel` aus `internal/cli`.
- Produces: `flowCommand(args, stdin, stdout, stderr) int`; intern `flowDeps{Stdout, Stderr io.Writer; Clock runner.Clock; Models func(provider string) (model.Model, error); Blocks []flow.Block; Bundled fs.FS}`, `flowCLI(args []string, deps flowDeps) int`.

- [ ] **Step 1: Port und Umbenennen.** Die sechs Quelldateien per sed nach `internal/cli/flow*.go`; `package main` → `package cli`; `Deps` → `flowDeps`, `cli(` → `flowCLI(`, `production(` → `flowProduction(`, `runCommand` → `flowRunCommand` usw. (alle Funktionsnamen mit Präfix `flow`, weil `internal/cli` schon `runCommand`-ähnliche Namen haben kann: `grep -n "^func run\|^func show\|^func list\|^func resume\|^func replay\|^func arguments\|^func refuse\|^func finish" internal/cli/*.go` vorher). `main()` entfällt; `flowCommand` ist:

```go
// flowCommand is `loomux flow`: the edge between a command line and the
// runtime under internal/flow.
func flowCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	return flowCLI(args, flowProduction(stdout, stderr))
}
```

`flowProduction` hält `Bundled: flows.FS()` und `Models: func(provider string) (model.Model, error) { return nil, fmt.Errorf("no adapter for provider %s yet", provider) }`. `version` (M1: `"0.1.0"`) entfällt; die Marke bekommt `strings.TrimPrefix(versionLine(), "loomux ")`. `--version` als Unterbefehl entfällt (`loomux version` gibt es). Usage-Text mit `loomux flow run [<flow>] …`, Meldungspräfix `loomux flow <befehl>:` statt `ulflow`.

- [ ] **Step 2: Tests portieren** (`helpers_test.go` → `flow_helpers_test.go` usw.). Der Harness bekommt `Bundled` (ein `fstest.MapFS` oder `flows.FS()`), `exampleProject` kopiert nicht mehr, sondern legt ein leeres Projekt an (der Beispiel-Flow kommt aus dem Katalog), und ruft `flowCLI`. Der Golden-Test aus `cmd/flow/golden_test.go` entfällt: der Katalog-Test aus Task 11 ersetzt ihn (Spec, „Nachweis“). Tests zu `runtime`/Python (`was started by the Python runtime`) entfallen.

- [ ] **Step 3: Failing tests für das Neue** (`flow_test.go`):

| Test | Erwartung |
|---|---|
| `TestFlowRunWithoutANameStartsTheDefault` | `[flow] default = "example"` in `.loomux/config.toml`, `flow run` ohne Namen → pausiert am Tor wie `flow run example` |
| `TestFlowRunWithoutANameOrADefaultIsRefused` | Exit 1, stderr `no flow named and [flow] default is unset; known flows: example` |
| `TestFlowRunNamesAMissingDefault` | `[flow] default = "ghost"` → Exit 1, `[flow] default names "ghost", which is no flow here` |
| `TestFlowFindsTheRootFromASubdirectory` | Projekt mit `.loomux/config.toml` (danach sucht `hosts.FindRoot`), ohne `--root`, Arbeitsverzeichnis `root/internal` (per `t.Chdir`) → dieselbe Auffindung wie an der Wurzel |
| `TestFlowWithoutAConfigUsesTheWorkingDirectory` | Projekt ohne `.loomux/config.toml`, nur `.loomux/flows/mine/` → `flow run mine` aus der Wurzel findet den Flow (`FindRoot` scheitert mit `ErrNoRoot`, dann gilt `.`) |
| `TestFlowRunRefusesAnAgentFlowWithoutAnAdapter` | Produktions-`Models` → Exit 1, `no adapter for provider claude yet`, kein Lauf unter `.loomux/state/runs/` |
| `TestAPausedRunNamesItsOrigin` | Overlay gesetzt → stdout `run 0001 (example, bundled+overlay: questions/approve.md): paused` |
| `TestAPausedRunPrintsTheQuestionWithoutCarriageReturns` | Overlay `questions/approve.md` mit CRLF → stdout ohne `\r` |
| `TestResumeRefusesAnotherFlowFile` | Lauf auf einem Projekt-Flow `example` mit Override starten, Override aus der Config nehmen, `resume --answer yes` → Exit 1, `run 0001 started on project (hides bundled) and example now resolves to bundled, another flow.toml; start a new run with loomux flow run example`, Journal unverändert |
| `TestResumeWarnsWhenTheOverlaysChanged` | Lauf mit Overlay `questions/approve.md` starten, ein zweites Overlay `instructions/draft.md` dazulegen, `resume --answer yes` → läuft, stderr `warning: run 0001 started with overlays questions/approve.md and now has instructions/draft.md, questions/approve.md` |
| `TestFlowShowPrintsAFlow` | `flow show example` → Kopf `example (bundled)`, je Knoten Name, Art, Rolle, Modell-Label und `Via()`, je Kante `from -> to [when]` |
| `TestFlowListMarksTheDefaultAndWarns` | `flow list` → `example  bundled  ok  (default)` und die Warnungen aus `SettingsWarnings` und `Found.Warnings` auf stderr |
| `TestFlowRunReadsTheAgentTable` | `[agent]` mit Bindung `writer = "w"` und `[agent.models.w] provider = "agy"` → Fake bekommt `Provider: "agy"` |
| `TestAConfigTheReadersRefuseStopsEveryFlowCommand` | `[agent] default = "x"` ohne Modell → `flow list` Exit 1 mit der Meldung des Lesers |

Die zwei Tests mit `\r` per Write, danach nachlesen.

- [ ] **Step 4: Rot sehen**, dann umsetzen:
  - `flow_session.go`: `project(root, deps)` liest `config.ReadAgent(root)` und `config.ReadFlowSettings(root)` und baut den Katalog; `openFlow(root, name, options, deps)` ruft `load.Find(root, deps.Bundled, settings, name)`, gibt `found.Warnings` als `warning: …` auf stderr aus, lädt mit `load.Load(found, catalog)`; `session` hält `agent config.Agent`, `settings config.FlowSettings`, `found load.Found`. `modelFor` löst über `model.Resolve(s.agent, node.Role, s.graph.Role)`.
  - Wurzel: `--root` hat keinen Vorgabewert mehr; leer → `hosts.FindRoot(".")` (wie `resolveConfigTarget`). `FindRoot` sucht nach `.loomux/config.toml`; ein Projekt ohne Config hat trotzdem Flows, darum gilt bei `errors.Is(err, hosts.ErrNoRoot)` das Arbeitsverzeichnis `.`, jeder andere Fehler ist Exit 1.
  - `flow run [<flow>]`: ohne Namen `settings.Default`; leer → die Ablehnung aus der Tabelle; ein `default`, den `Find` nicht findet → die eigene Meldung.
  - Marke: `runs.Marker{Flow: name, Origin: found.Origin, Overlays: found.Overlays, Options: …, Baseline: …, Version: strings.TrimPrefix(versionLine(), "loomux ")}`.
  - `finish` druckt `run <id> (<flow>, <origin>[: <overlays, comma>]): <status>`; die Frage ohne `\r` (sie kommt schon ohne aus `ReadText`, und `strings.TrimRight(text, "\n")` bleibt).
  - `recorded` (resume/replay): nach `openFlow` vergleicht es `marker.Origin` mit `s.found.Origin`. Gehören beide zu verschiedenen Quellen der `flow.toml` — `project`/`project (hides bundled)` lesen den Projektordner, `bundled`/`bundled+overlay` den Katalog —, lehnt es ab wie in der Tabelle: das offene Tor gehört womöglich zu einem anderen Graphen. Ändert sich nur `marker.Overlays` gegenüber `s.found.Overlays`, warnt es und läuft weiter; die Definitions-Hashes melden die betroffenen Knoten ohnehin. Die Python-Prüfung (`marker.Runtime != "go"`) entfällt.
  - `flow show <lauf|flow>`: ein Argument aus Ziffern ist eine Laufnummer (bisheriges Verhalten), sonst ein Flow-Name → neue Funktion `flowShowFlow`.
  - `flow list`: `load.List` plus `load.SettingsWarnings`; Spalten `Name`, `Origin`, `ok|Problem`, `(default)`.
  - Die offenen Punkte aus den Wave-3-Befunden, die diese Dateien berühren: `session.go:135` (`TrimRight` am `Detail`) bleibt, weil `ReadText` das `\r` schon nimmt; `resume.go:52`-Test („nothing is written“ nur über Zeilenzahl) wird auf Byte-Vergleich des Journals umgestellt; `forgetUnstarted` ignoriert einen Fehler von `os.Remove` weiter und sagt das im Kommentar.

- [ ] **Step 5: Grün, 100 %** — `go test ./internal/cli/ -run Flow -count=1 -cover`.

- [ ] **Step 6: Commit** — `feat(cli): add loomux flow run, resume, replay, show and list`.

---

### Task 13: Session-Start meldet wartende Läufe

**Files:**
- Create: `internal/hooks/flowruns.go`, `internal/hooks/flowruns_test.go`
- Modify: `internal/hooks/hook_session_start.go` (Signatur um `version string`, Aufruf von `waitingRuns`), `internal/cli/hook.go` (übergibt `strings.TrimPrefix(versionLine(), "loomux ")`), Tests beider
- Modify: `internal/cli/imports_test.go` (neue Liste `forbiddenFlowForHooks`)

**Interfaces:**
- Consumes: `runs.*` (4), `journal.Pending` (3), `flows.Names` (11), `config.ReadFlowSettings` (1), `hooks.executable` (bestehende Naht).
- Produces: `hooks.SessionStart(stdin, stdout, stderr, root, hostName, version string) int`; intern `waitingRuns(root, version string, stderr io.Writer) []string`, `ignoredFlowFolders(root string) []string`.

- [ ] **Step 1: Failing tests** `flowruns_test.go`:

```go
func TestWaitingRunsAnnouncesEachPausedRun(t *testing.T) {
	root := t.TempDir()
	writeRun(t, root, "0001", runs.Marker{Flow: "example", Origin: "bundled", Version: "3.3.0"}, pausedLine("approve", "Ship it?"))
	writeRun(t, root, "0002", runs.Marker{Flow: "dev-cycle", Origin: "bundled+overlay", Overlays: []string{"instructions/review.md"}, Version: "3.3.0"}, pausedLine("approve_plan", "Plan ok?"))
	writeRun(t, root, "0003", runs.Marker{Flow: "example", Origin: "bundled", Version: "3.3.0"}, okLine("draft"))
	executable = func() (string, error) { return filepath.FromSlash("C:/x/loomux.exe"), nil }
	t.Cleanup(func() { executable = os.Executable })
	var stderr bytes.Buffer
	got := waitingRuns(root, "3.3.0", &stderr)
	want := []string{
		"run 0001 (example, bundled) is waiting at approve: Ship it?\n  a human answers it with: C:/x/loomux.exe flow resume 0001 --answer \"your answer\"",
		"run 0002 (dev-cycle, bundled+overlay: instructions/review.md) is waiting at approve_plan: Plan ok?\n  a human answers it with: C:/x/loomux.exe flow resume 0002 --answer \"your answer\"",
	}
	if !reflect.DeepEqual(got, want) || stderr.Len() != 0 {
		t.Fatalf("got %q\nstderr %q", got, stderr.String())
	}
}

func TestWaitingRunsNamesAJournalAnotherBinaryWrote(t *testing.T) {
	// A line with a key this reader does not know, from a marker with another
	// version: the hook names both versions instead of calling it damaged.
	…
	// want on stderr: "loomux hook session-start: run 0001 was written by loomux 9.9.9, this is 3.3.0"
}

func TestWaitingRunsNamesADamagedJournalAndGoesOn(t *testing.T) { … }

func TestIgnoredFlowFoldersWarns(t *testing.T) {
	// .loomux/flows/example/flow.toml without [flow] overrides → one line
	// ".loomux/flows/example is ignored: [flow] overrides does not name it";
	// with overrides = ["example"] → none; .loomux/flows/mine/ → none.
	…
}
```

`writeRun`, `pausedLine`, `okLine` schreiben Marke und Journal über `runs.WriteMarker` und `journal.Append`. Die Pünktchen-Tests mit ihren Erwartungen ausschreiben (Aufbau wie im ersten). Die Tests mit `\n` in den Erwartungen per Write.

In `hook_session_start_test.go`: ein Test, dass `SessionStart` die Zeilen in den Kontext schreibt, auch bei `payload.Repeat` (ein wartender Lauf ist nach einem Compact wieder wichtig).

- [ ] **Step 2: Rot sehen**, dann `flowruns.go`:

```go
package hooks

// waitingRuns announces every run of the project that waits at a gate, one
// context line each, by run number. The answer is a human's -- the guard
// refuses it to an agent -- so the line says who answers and names the binary
// this hook runs from: loomux is not on every PATH, and a project runs its
// hooks from bin/ or from the machine-wide install.
//
// A journal that will not read is named on stderr and the other runs are still
// announced. When its marker says another loomux wrote the run, the line names
// both versions instead: a newer binary's journal is not damaged, only newer.
func waitingRuns(root, version string, stderr io.Writer) []string { … }
```

Rumpf nach dem Muster des M1-Hooks (`d7041e5:cmd/guard/hook_session_start.go` Zeilen 90–170): Verzeichnis `runs.Dir` lesen, `*.jsonl` sortiert, `journal.Pending`, `runs.ReadMarker`; Herkunft `(<flow>, <origin>[: <overlays>])`; Binärpfad `filepath.ToSlash` von `executable()`, bei Fehler `loomux`. `ignoredFlowFolders` schweigt ohne `.loomux/flows/` und liest `[flow]` erst, wenn der Ordner einen Unterordner mit einem Katalognamen hat: ein Projekt ohne Flows zahlt keinen zweiten Decode, und die aufgezeichneten Session-Start-Fälle mit absichtlich kaputter Config bleiben unverändert. Dann `flows.Names()` und `config.ReadFlowSettings(root)`; ein Lesefehler der Config wird eine Zeile „loomux: [flow] cannot be read: …“. Ein Test belegt das Schweigen: kaputte Config, kein `.loomux/flows/` → keine Zeile.

`SessionStart` bekommt den Parameter `version` und hängt `waitingRuns` und `ignoredFlowFolders` an `lines` an, **außerhalb** des `!payload.Repeat`-Zweigs. Der Kommentar „Waiting flow runs are not announced here …“ entfällt.

- [ ] **Step 3: Importgraph.** In `internal/cli/imports_test.go` nach `forbiddenMaintenanceForHooks`:

```go
// forbiddenFlowForHooks is the flow runtime. The session-start hook reads run
// markers and journals and the names of the bundled flows, nothing more: a
// hook that linked the loader or the runner would pay for them on every start.
func forbiddenFlowForHooks() []string {
	return []string{
		"github.com/xidus90/loomux/internal/flow",
		"github.com/xidus90/loomux/internal/flow/blocks",
		"github.com/xidus90/loomux/internal/flow/expr",
		"github.com/xidus90/loomux/internal/flow/load",
		"github.com/xidus90/loomux/internal/flow/model",
		"github.com/xidus90/loomux/internal/flow/runner",
		"github.com/xidus90/loomux/internal/flow/tmpl",
	}
}
```

und ein Test, der sie wie `forbiddenMaintenanceForHooks` prüft (Muster steht in derselben Datei). Der Test vergleicht exakte Paketpfade (`seen[forbidden]`), darum verbietet `internal/flow` nicht zugleich `internal/flow/runs` und `internal/flow/journal`, die der Hook bindet.

- [ ] **Step 4: Grün, 100 %** — `go test ./internal/hooks/ ./internal/cli/ -count=1`.

- [ ] **Step 5: Commit** — `feat(hooks): announce waiting flow runs at session start`.

---

### Task 14: Wächterregeln für Tore, Laufdateien und mitgelieferte Flows

**Files:**
- Modify: `internal/hooks/guard.go` (`builtinPathRules`, `builtinCommands`, `manifestWriteSource` → `writeSource(name)`, `checkTool`)
- Create: `internal/hooks/guardflow.go` (`answersAGate`, `flowFolderRules`), `internal/hooks/guardflow_test.go`
- Modify: `internal/hooks/guard_test.go` (bestehende Tests der Manifest-Regel laufen über `writeSource`)

**Interfaces:**
- Consumes: `flows.Names()` (11), `config.ReadFlowSettings` (1), `loomuxArgs`, `lineVariants`, `segments`, `readings`, `splitSegments`, `plainLine` (bestehend in `guard.go`).
- Produces: intern `answersAGate(line string) bool`, `writeSource(name string) string`, `flowFolderReasons(root, rel string) []string`, `flowFolderCommand(root string) (config.CommandRule, bool)`.

- [ ] **Step 1: Failing tests** `guardflow_test.go`:

```go
func TestAnAgentMayNotAnswerAGate(t *testing.T) {
	for _, line := range []string{
		`loomux flow resume 0001 --answer yes`,
		`loomux flow resume 0001 --answer=yes`,
		`loomux flow resume 0001 -answer "no: thin"`,
		`bin/loomux.exe flow resume 0001 --answer yes`,
		`echo ok && loomux flow resume 0001 --answer yes`,
		`"C:/Users/x/AppData/Local/loomux/bin/loomux.exe" flow resume 0001 --answer yes`,
	} {
		if !answersAGate(line) {
			t.Errorf("%q passes", line)
		}
	}
	for _, line := range []string{
		`loomux flow resume 0001`,
		`loomux flow run example`,
		`loomux flow show 0001`,
		`echo "loomux flow resume 0001 --answer yes"`,
	} {
		if answersAGate(line) {
			t.Errorf("%q is refused", line)
		}
	}
}
```

(Die vierte Zeile der erlaubten Liste — der Befehl nur als Text in `echo` — folgt dem, was `writesConfiguration` für zitierte Befehle heute tut; zeigt der bestehende Test von `writesConfiguration` für diesen Fall „refused“, übernimmt dieser Test dieselbe Erwartung, und der Bericht sagt es.)

Weiter: ein Write/Edit auf `.loomux/state/runs/0001.jsonl` wird verweigert (Grund „a flow's journal and marker are written by loomux, not by the party the gates ask“); Shell-Zeilen `echo x >> .loomux/state/runs/0001.jsonl`, `rm .loomux/state/runs/0001.flow`, `Remove-Item .loomux\state\runs\0001.jsonl`, `sed -i s/a/b/ .loomux/state/runs/0001.jsonl` werden verweigert, `cat .loomux/state/runs/0001.jsonl` nicht; ein Write auf `.loomux/flows/example/flow.toml` (Katalogname) und auf `.loomux/flows/review/questions/q.md` bei `[flow] overrides = ["review"]` wird verweigert, auf `.loomux/flows/mine/flow.toml` nicht; `rm -r .loomux/flows/example` und `git rm .loomux/flows/review/questions/q.md` werden verweigert. Diese Tests laufen über `checkTool(root, tool, input, policy)` wie die bestehenden Tests in `guard_test.go`; Zeilen mit Backslash per Write.

- [ ] **Step 2: Rot sehen**, dann:
  - `manifestWriteSource()` wird `writeSource(name string) string` (Rumpf unverändert, `name` statt der Konstante), und `builtinCommands` baut die Manifest-Regel mit `writeSource(manifestName)`, wobei `manifestName` die bisherige Konstante `name` ist. Dazu eine zweite Regel:
    ```go
    	runsName := `['"]?(?:[^\s;|&'"<>]*[/\\=:])?\.loomux[/\\]+state[/\\]+runs(?:[/\\][^\s;|&'"<>]*)?['"]?`
    ```
    mit dem Grund „a flow's journal and marker are written by loomux, not by the party the gates ask“. (Per Edit: Backslashes.)
  - `builtinPathRules` bekommt `{Match: []string{".loomux/state/runs/**"}, Reason: "a flow's journal and marker are written by loomux, not by the party the gates ask"}`.
  - `guardflow.go`:
    ```go
    // answersAGate says whether a shell line runs `loomux flow resume` with an
    // answer. A gate asks a human; an agent that answered its own gates would
    // approve its own plan and its own push. The line is read the way
    // writesConfiguration reads it, and has the same holes.
    func answersAGate(line string) bool {
    	for _, variant := range lineVariants(line) {
    		for _, segment := range segments(variant) {
    			for _, words := range readings(segment) {
    				args, ok := loomuxArgs(words)
    				if ok && len(args) > 1 && args[0] == "flow" && args[1] == "resume" && namesAnswer(args[2:]) {
    					return true
    				}
    			}
    		}
    	}
    	return false
    }

    // namesAnswer is any spelling of --answer the flag package takes.
    func namesAnswer(args []string) bool {
    	for _, arg := range args {
    		name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
    		if name != arg && (name == "answer" || strings.HasPrefix(name, "answer=")) {
    			return true
    		}
    	}
    	return false
    }
    ```
    Nutzt `writesConfiguration` die Hilfen mit anderen Signaturen (`readings(segment)` usw.), passt dieser Code sich an sie an, nicht umgekehrt.
  - Mitgelieferte Flows: `protectedFlows(root) ([]string, error)` = `flows.Names()` ∪ `settings.Overrides`; `flowFolderReasons(root, rel)` antwortet für `rel` unter `.loomux/flows/<name>/` mit dem Grund aus der Spec, wenn `<name>` geschützt ist, und mit „loomux cannot read [flow] of .loomux/config.toml, so it refuses writes under .loomux/flows: …“, wenn die Config nicht lesbar ist. Für die Shell baut `flowFolderCommand(root)` eine Regel mit `writeSource` über `\.loomux[/\\]+flows[/\\]+(?:<namen, regexp.QuoteMeta>)(?:[/\\][^\s;|&'"<>]*)?`, nur wenn die Zeile `flows` enthält (Kleinschreibung verglichen) — sonst kostet die Regel nichts.
  - `checkTool`: nach den Pfadregeln `reasons = append(reasons, flowFolderReasons(root, rel)...)`; nach den Befehlsregeln die Flow-Regel und `if answersAGate(line) { reasons = append(reasons, "a flow's gate asks a human; the answer is theirs. Ask the user to answer it with `flow resume <run> --answer \"…\"` themselves") }`.

- [ ] **Step 3: Grün, 100 %** — `go test ./internal/hooks/ -count=1 -cover`.

- [ ] **Step 4: Messen** (Spec, „Session-Start“ und Wächter). `testdata/bench/flow-a-hooks.json` im Format von `testdata/bench/2c-hooks.json` mit vier Fällen: `hook pre-tool-use` mit einem `Edit` außerhalb von `.loomux/`, mit einem `Edit` unter `.loomux/flows/mine/`, mit einem `Bash`-Aufruf `git status`, und `hook session-start --host claude` gegen ein Projekt mit einem wartenden Lauf. Zwei Binaries in den Scratchpad bauen: `before.exe` von `origin/master`, `after.exe` von diesem Zweig (`go build -o "$S/after.exe" ./cmd/loomux`; `before` aus einem Worktree auf `origin/master`). `"$S/after.exe" dev bench-hooks testdata/bench/flow-a-hooks.json -n 30` und dasselbe mit `before.exe`, drei Durchgänge abwechselnd, die Ausgabe jeweils ganz in eine Datei. Die Tabelle mit Datum, Uhrzeit, Commit, Maschine, kalt/warm, Median und Minimum sowie die Binärgrößen gehen in Task 15 in die Benchmark-Seiten.

- [ ] **Step 5: Commit** — `feat(guard): keep gate answers, run files and bundled flows with a human`.

---

### Task 15: Doku

**Files:**
- Create: `docs/en/flows.md`, `docs/de/flows.md`
- Modify: `docs/{en,de}/cli-reference.md` (Abschnitt `loomux flow`), `docs/{en,de}/configuration.md` (`[agent]`, `[flow]`, benannte Schlüssel), `docs/{en,de}/hooks.md` (Session-Start), `docs/{en,de}/benchmarks.md` (Messung aus Task 14), `docs/{en,de}/README.md` (Zeile „Flows“), `README.md`, `README.de.md` (Befehlsliste, Roadmap-Zeilen „Flow-Laufzeit“ und neu „Flows über MCP“)

- [ ] **Step 1: `docs/en/flows.md`** mit den Abschnitten: What a flow is (Ordner, Knotenarten, Kanten, Tore), Running a flow (`loomux flow …`, Exit-Codes, wo Läufe liegen), Roles and models (`[agent]`, Kette, `flow show`), The catalog and overrides (Herkunft, Overlay, `[flow] overrides`, Warnungen), Contributing a flow (Ordner unter `flows/catalog/`, `_test/script.toml` mit Beispiel, `go test ./flows -update`, PR mit `release:minor`), Gates are a human's (Wächterregeln, `!`-Weg, der Pfad aus dem Session-Start-Hinweis). `docs/de/flows.md` trägt dasselbe auf Deutsch mit gleichen Abschnitten.

- [ ] **Step 2: Referenzen.** CLI-Referenz: je Unterbefehl Aufruf, Flags, Exit-Codes, Beispielausgabe. Konfigurationsreferenz: die sieben Schlüssel aus Task 2 mit Beispiel, der Hinweis, dass `config set agent.roles.<rolle>` erst nach `agent.models.<name>.provider` gelingt, und die Abgrenzung zu `[model]`. Hooks: Session-Start meldet wartende Läufe und ignorierte Projektordner.

- [ ] **Step 3: READMEs.** Befehlsliste um `loomux flow run|resume|replay|show|list`; Roadmap: Zeile „Flow-Laufzeit“ sagt, was gebaut ist (Ordnerformat, Katalog, Rollen, Overlay, Wächter) und dass Agentenknoten auf die Adapter warten; neue Zeile „Flows über MCP“ (A2) unter „Kommt“, Priorität 4, hängt ab von der Flow-Laufzeit. Liegt die Roadmap (PR #46) noch nicht auf `origin/master`, rebased dieser Zweig vorher darauf (Task 16).

- [ ] **Step 4: Benchmarks.** Der Eintrag aus Task 14 Schritt 4 chronologisch ans Ende beider Seiten, im Stil des letzten Eintrags (Kopf mit Datum und Uhrzeit, Ziel, Methode, Tabelle).

- [ ] **Step 5: Tor, Commit** — `docs(flow): document flows, the catalog and the new keys`.

---

### Task 16: Pull Request vorbereiten

- [ ] **Step 1:** `git fetch origin`; ist PR #46 gemergt, `git rebase origin/master`; sonst bleibt die Roadmap-Änderung aus Task 15 und kollidiert beim Merge nur mit sich selbst — dann vor dem Push erneut prüfen.
- [ ] **Step 2:** Skill `release-pr`: Commits nach Thema gruppieren (Spec, Plan und Papiere als ein `docs`-Commit; je Paketgruppe ein `feat`), Label `release:minor`, Rumpf mit `## Changelog` (`### Added`: `loomux flow`, Katalog, `[agent]`/`[flow]`, benannte Schlüssel, Session-Start-Hinweis, Wächterregeln), `parse-body` grün.
- [ ] **Step 3:** Push-Befehl für den Menschen nennen, danach `git ls-remote` gegen `HEAD`, PR mit `--assignee @me`.

---

## Selbstprüfung gegen die Spec

| Spec-Abschnitt | Task |
|---|---|
| Ablage (Paketliste, Katalog, AGENTS.md) | 3–12, 11 Step 6 |
| Flow-Format (Ordner, Pfade, CRLF, Namensregeln, unbekannte Schlüssel, Rollen) | 1, 7, 10 |
| Rollen und Modelle (Kette, Journal-Feld `role`, `show`) | 6, 3, 9, 12 |
| Auffindung, Overlay, Auswahl, `[flow] overrides` | 1, 10, 12 |
| Konfiguration (Schlüssel, Schema, benannte Schlüssel, `edit.Set`-Beleg) | 1, 2 |
| Katalog und Beitrag (`flows/catalog`, `_test/`, Test im Tor, `example`) | 11 |
| Läufe (Verzeichnis, Marke, CLI, Aufruf aus der Sitzung) | 4, 12 |
| Session-Start (Hinweis, Binärpfad, Versionen, ignorierte Ordner, Importgraph, Messung) | 13, 14 Step 4 |
| Wächter (Antwort, Laufdateien, mitgelieferte Flows, Löschen, Grenze) | 14 |
| Übernahme aus M1 (entfällt, bleibt offen) | 3, 4, 9, 12 (je Step „Rot sehen, dann“) |
| Doku im selben PR | 15 |
| Nachweis | jede Task, 11 |

Abweichungen von der Spec, beim Planen festgelegt und dort nachgetragen: Katalog unter `flows/catalog/` mit `_test/` statt `flows/<name>/test/` (Grund: `go:embed`); der Lader bekommt den Katalog als `fs.FS` und importiert `flows` nicht; nach einer Namenskollision durch ein Release gilt `[flow] overrides` statt „Projekt verdeckt weiter“; `resume` lehnt einen Lauf ab, dessen `flow.toml` inzwischen aus einer anderen Quelle kommt; ohne `.loomux/config.toml` gilt das Arbeitsverzeichnis als Wurzel.
