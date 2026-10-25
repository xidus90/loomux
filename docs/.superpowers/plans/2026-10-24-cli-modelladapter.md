# CLI-Modelladapter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Agentenknoten laufen echt über die CLIs `claude`, `agy` und `qodercli`; Strata über `claude` mit Basis-URL.

**Architecture:** Ein neues Paket `internal/flow/model/cli` hält einen Adapter je Anbieter hinter `model.Model`, mit einer Prozess-Naht für Tests. Modell-Einstellungen (`agent`, `effort`, `base_url`) kommen aus `[agent.models.<name>]`, der Knoten überschreibt `agent`/`effort`; sie reisen je Aufruf in `model.Request`. `flowProduction` baut den Adapter des Anbieters mit Wurzel und Zeitgrenze.

**Tech Stack:** Go (stdlib `os/exec`, `context`, `encoding/json`), vorhandene Pakete `internal/config`, `internal/flow/...`.

**Spec:** `docs/.superpowers/specs/2026-10-24-cli-modelladapter-design.md`

## Abweichung von der Spec (meine Festlegung, beim Plan getroffen)

Die Spec sieht „ein Adapter je Modell“ über eine geänderte Fabrik vor. Der Lauf
hält aber genau **ein** `model.Model` in `flow.Env` (`flowSession.modelFor`),
und ein Flow darf zwei Modelle desselben Anbieters nutzen (etwa `claude` und
Strata). Deshalb: **ein Adapter je Anbieter und Lauf**, und die modellbezogenen
Werte (`Agent`, `BaseURL`, `Profile`) reisen je Aufruf in `model.Request`.
Wurzel und Zeitgrenze bekommt der Adapter beim Bau. Das Ergebnis für den
Nutzer ist dasselbe; die Fabrik-Signatur ändert sich weniger.

## Global Constraints

- 100 % Coverage je Funktion; Ausnahme nur mit `//coverage:exempt <reason>` direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst.
- Code, Kommentare, Meldungen englisch; Doku zweisprachig (`docs/en`, `docs/de`, README beide).
- Kein Test ruft eine echte CLI auf.
- Kein Adapter setzt `--dangerously-skip-permissions` oder einen Bypass-Modus.
- Anbieter: genau `claude`, `agy`, `qoder`.
- `[agent] timeout`: Vorgabe `30m`, `"0"` schaltet ab.
- `base_url` nur bei `provider = "claude"`.
- Vorrang `agent`/`effort`: Knoten, sonst Modell, sonst leer.
- Tokens = Eingabe + Ausgabe, ohne Cache-Lesen und Denk-Tokens; fehlt die Zahl, `0`.
- Commits: Conventional Commits, Autor der Nutzer, kein Modell als Mitautor.

## Review Focus

1. CLI fehlt im `PATH` → Fehler nennt die CLI und den Anbieter, kein Panic (Task 3).
2. CLI endet mit Exit ≠ 0 oder schreibt kein JSON → Knotenfehler mit den ersten Zeilen von stderr (Task 3).
3. Antwort hat ein zusätzliches oder fehlendes Feld bzw. falschen Typ (`"3"` statt `3`) → abgelehnt, bei `qoder` genau ein neuer Versuch (Task 3, Task 6).
4. Zeitgrenze läuft ab → Kindprozess beendet, Meldung nennt `30m` und `[agent] timeout` (Task 3).
5. Bestehende Goldens unter `flows/catalog/*/_test` und `testdata/cases/` bleiben bytegleich, solange kein neuer Schlüssel gesetzt ist (Task 2).

---

### Task 1: Config — Anbieter, Modell-Schlüssel, Zeitgrenze

**Files:**
- Modify: `internal/config/agent.go`
- Modify: `.loomux/config.example.toml`
- Test: `internal/config/agent_test.go`

**Interfaces:**
- Produces:
  - `type ModelSpec struct { Provider, Model, Agent, Effort, BaseURL string }`
  - `func ModelSpecKeys() []string` → `{"agent", "base_url", "effort", "model", "provider"}`
  - `Agent.Timeout time.Duration` (0 = aus), `AgentKeys()` um `"timeout"` erweitert
  - `func Providers() []string` → `{"agy", "claude", "qoder"}`
  - `const DefaultAgentTimeout = 30 * time.Minute`

- [ ] **Step 1: Write the failing tests** in `agent_test.go`, im Tabellenstil der vorhandenen Fehlerfälle:
  - `TestParseAgentModelKeys`: `[agent.models.q]\nprovider="claude"\nagent="reviewer"\neffort="high"\nbase_url="http://127.0.0.1:8080"` → `ModelSpec{Provider:"claude", Agent:"reviewer", Effort:"high", BaseURL:"http://127.0.0.1:8080"}`.
  - Fehlerfälle: `provider="gpt"` → `[agent.models.w] provider "gpt" is unknown; known: agy, claude, qoder`; `provider="agy"` + `base_url="x"` → `[agent.models.w] base_url only works with provider claude`; `agent=""` / `effort=""` / `base_url=""` → `… must be a non-empty string`.
  - `TestParseAgentTimeout`: fehlt → `30*time.Minute`; `"45m"` → `45*time.Minute`; `"0"` → `0`; `"soon"` → `[agent] timeout "soon" is not a duration such as "30m", or "0" to switch it off`; `-1m` → derselbe Fehler.
- [ ] **Step 2:** `go test ./internal/config -run 'TestParseAgent'` → FAIL.
- [ ] **Step 3:** In `parseModels` die drei Schlüssel lesen (gleiche Prüfung wie `model`), Anbieter gegen `Providers()` prüfen, `base_url` nur bei `claude`. `timeout` in `ParseAgent` mit `time.ParseDuration`; `"0"` → 0; negativ abgelehnt. `config.example.toml`: auskommentierte Beispiele für `timeout`, `agent`, `effort`, `base_url` (Strata).
- [ ] **Step 4:** `go test ./internal/config` → PASS.
- [ ] **Step 5:** Commit `feat(config): add agent, effort, base_url and timeout to the agent models`.

### Task 2: Auflösung, Definition, Request, Hash

**Files:**
- Modify: `internal/flow/model/resolve.go`, `internal/flow/model/model.go`, `internal/flow/model/tools.go`
- Modify: `internal/flow/block.go`, `internal/flow/blocks/agent.go`, `internal/flow/journal/hash.go`, `internal/flow/runner/walk.go`
- Test: `internal/flow/model/resolve_test.go`, `internal/flow/model/tools_test.go`, `internal/flow/blocks/agent_test.go`, `internal/flow/journal/hash_test.go`

**Interfaces:**
- Consumes: `config.ModelSpec` aus Task 1.
- Produces:
  - `model.Resolved` + `Agent, Effort, BaseURL string` (aus dem `ModelSpec`; leer bei `cli-default`)
  - `model.Request` + `Profile, Agent, BaseURL string`
  - `func model.EnforcesTools(provider string) bool` → `false` nur für `"agy"`
  - `flow.Definition` + `Agent, BaseURL string`, `ToolsEnforced bool`
  - `func journal.DefinitionHash(node any, text []byte, model, effort, agent, baseURL string, tools []string) (string, error)`

- [ ] **Step 1: Write the failing tests:**
  - `TestResolveCarriesModelSettings`: Rolle gebunden an `ModelSpec{Provider:"claude", Agent:"a", Effort:"high", BaseURL:"u"}` → `Resolved.Agent=="a"`, `Effort=="high"`, `BaseURL=="u"`.
  - `TestAgentNodeOverridesAgentAndEffort` (blocks): Knoten mit `agent="n"`, `effort="low"` über Modell mit `a`/`high` → `Definition.Agent=="n"`, `Effort=="low"`; ohne Knotenschlüssel → `a`/`high`; ohne beides → leer. Der Fake sieht dieselben Werte und `Profile` in `Seen()[0]`.
  - `TestAgentCheckAcceptsAgentKey`: Knoten mit `agent = "x"` → keine Finding; `agent = 3` → `agent is 3, not text`.
  - `TestToolsEnforcedFollowsProvider`: `claude`/`qoder` → `true`, `agy` → `false`.
  - `TestDefinitionHashUnchangedWithoutNewKeys`: `DefinitionHash(n, t, m, e, "", "", tools)` gleich dem heute festgehaltenen Wert (vor der Änderung einmal ausrechnen und als Konstante in den Test schreiben); mit `agent="x"` anders; mit `baseURL="u"` anders.
- [ ] **Step 2:** `go test ./internal/flow/...` → FAIL.
- [ ] **Step 3:** Umsetzen. Hash: `agent` und `base_url` kommen nur in die Map, wenn nicht leer, damit bestehende Hashes und Goldens gleich bleiben. `walk.go` reicht die neuen Felder an `DefinitionHash`. `agentDefine` setzt `ToolsEnforced = model.EnforcesTools(resolved.Provider)`; `Run` füllt `Request.Profile/Agent/BaseURL`. `KeyFindings` um `"agent"` erweitern.
- [ ] **Step 4:** `go test ./internal/flow/... ./flows/...` → PASS, und `git diff --stat -- flows testdata` leer.
- [ ] **Step 5:** Commit `feat(flow): let models and nodes set agent and effort for agent nodes`.

### Task 3: Gemeinsamer CLI-Kern

**Files:**
- Create: `internal/flow/model/cli/cli.go`, `internal/flow/model/cli/reply.go`
- Test: `internal/flow/model/cli/cli_test.go`, `internal/flow/model/cli/reply_test.go`

**Interfaces:**
- Consumes: `model.Request`, `model.Reply`, `model.ReplyType` aus Task 2.
- Produces:
  - `type Command struct { Name string; Args []string; Env []string; Dir string }` (Env: nur Zusätze zu `os.Environ()`)
  - `type Exec func(ctx context.Context, c Command) (stdout, stderr []byte, err error)`
  - `func System(ctx context.Context, c Command) ([]byte, []byte, error)` — die echte Naht über `exec.CommandContext`
  - `type Options struct { Root string; Timeout time.Duration; Exec Exec }`
  - `func run(ctx context.Context, o Options, c Command) ([]byte, error)` — Zeitgrenze, Dir = Root, Fehlertexte
  - `func Schema(reply map[string]model.ReplyType) string` — JSON-Schema: `type: object`, alle Felder `required`, `additionalProperties: false`, Typen `string`/`integer`/`boolean`; Schlüssel sortiert
  - `func Fields(raw []byte, reply map[string]model.ReplyType) (map[string]any, error)` — prüft genau die Felder und Typen; `int` aus JSON-Zahl ohne Nachkommastellen

- [ ] **Step 1: Write the failing tests:**
  - `TestSchemaIsSortedAndClosed`: `{"verdict":string,"n":int,"ok":bool}` → exakt `{"additionalProperties":false,"properties":{"n":{"type":"integer"},"ok":{"type":"boolean"},"verdict":{"type":"string"}},"required":["n","ok","verdict"],"type":"object"}`.
  - `TestFieldsRejects`: Zusatzfeld → `reply carries "x", which the node did not ask for`; fehlend → `reply lacks "n"`; `"3"` für int → `reply field "n" is "3", not int`; `1.5` → dieselbe Art Fehler.
  - `TestRunTimesOut`: Fake-`Exec`, die auf `ctx.Done()` wartet; `Timeout: 10*time.Millisecond` → Fehler enthält `timed out after 10ms` und `[agent] timeout`.
  - `TestRunWithoutTimeout`: `Timeout: 0` → kein Deadline im ctx (Fake prüft `ctx.Deadline()`).
  - `TestRunNamesMissingCLI`: Fake gibt `exec.ErrNotFound` → `claude CLI not found on PATH`.
  - `TestRunExitCodeShowsStderr`: Fehler + stderr mit 12 Zeilen → Meldung enthält die ersten 5.
  - `TestRunSetsDir`: Fake sieht `Dir == Options.Root`.
  - `TestSystemRunsAProcess`: über `TestHelperProcess`-Muster (`os.Args[0]` mit `-test.run=TestHelperProcess`, Env `GO_HELPER=1`) → stdout zurück; das ist der einzige echte Prozess, keine Modell-CLI.
- [ ] **Step 2:** `go test ./internal/flow/model/cli` → FAIL.
- [ ] **Step 3:** Umsetzen.
- [ ] **Step 4:** PASS, Coverage 100 % (`go run ./cmd/loomux check coverage`).
- [ ] **Step 5:** Commit `feat(flow): add the shared core of the CLI model adapters`.

### Task 4: Adapter `claude` (auch Strata)

**Files:**
- Create: `internal/flow/model/cli/claude.go`
- Test: `internal/flow/model/cli/claude_test.go`

**Interfaces:**
- Consumes: Task 3.
- Produces: `func NewClaude(o Options) model.Model`

- [ ] **Step 1: Write the failing tests** mit Fake-`Exec`, die den `Command` festhält und eine gemessene JSON-Ausgabe von `claude -p --output-format json` zurückgibt (vor dem Test einmal mit der echten CLI aufnehmen und als Testdatei unter `internal/flow/model/cli/testdata/claude-reply.json` ablegen):
  - `TestClaudeArgs`: Request mit Model `m`, Effort `high`, Agent `a`, Tools `{Glob,Grep,Read,mcp__qmd}` → `Name=="claude"`, Args enthalten `-p`, `--output-format json`, `--json-schema <Schema>`, `--model m`, `--effort high`, `--agent a`, `--tools Glob,Grep,Read`, `--allowedTools mcp__qmd`; keine `--dangerously-skip-permissions`; Prompt als letztes Argument bzw. über stdin, wie die CLI es verlangt.
  - Leeres Model/Effort/Agent → die Flags fehlen.
  - `TestClaudeBaseURL`: `BaseURL "http://127.0.0.1:8080"` → `Env` enthält `ANTHROPIC_BASE_URL=http://127.0.0.1:8080`; ohne → `Env` leer.
  - `TestClaudeReply`: aus `claude-reply.json` → `Fields` gemäß Schema, `Tokens` = input+output, `Model` aus der Ausgabe.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Umsetzen. Prüfen, ob Strata zusätzlich einen Token verlangt (README/Doku von Strata); wenn ja, `ANTHROPIC_AUTH_TOKEN` nur setzen, wenn `BaseURL` gesetzt ist, und das im Test festhalten.
- [ ] **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(flow): run agent nodes through the claude CLI`.

### Task 5: Adapter `agy`

**Files:**
- Create: `internal/flow/model/cli/agy.go`
- Test: `internal/flow/model/cli/agy_test.go`, `internal/flow/model/cli/testdata/agy-reply.json`

**Interfaces:**
- Produces: `func NewAgy(o Options) model.Model`

- [ ] **Step 1: Write the failing tests** (Antwort gemessen: `{"status":"SUCCESS","response":…,"usage":{"input_tokens":…,"output_tokens":…}}`):
  - `TestAgyArgs`: `Profile "read_only"` → `--mode plan`; jedes andere Profil → `--mode accept-edits`; dazu `--output-format json`, `--json-schema`, `--model`, `--effort`, `--agent` wie bei claude; `-p` steht als **letztes** Flag direkt vor dem Prompt (agy nimmt das nächste Wort als Prompt); kein `--dangerously-skip-permissions`.
  - `Options.Timeout > 0` → zusätzlich `--print-timeout <dauer>`.
  - `TestAgyReply`: `Tokens` = input+output (ohne `cache_read_tokens`, `thinking_tokens`).
  - `TestAgyStatusNotSuccess`: `status:"ERROR"` → Fehler mit `response`-Text.
- [ ] **Step 2:** FAIL. **Step 3:** Umsetzen. **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(flow): run agent nodes through the agy CLI`.

### Task 6: Adapter `qoder`

**Files:**
- Create: `internal/flow/model/cli/qoder.go`
- Test: `internal/flow/model/cli/qoder_test.go`, `internal/flow/model/cli/testdata/qoder-reply.json`

**Interfaces:**
- Produces: `func NewQoder(o Options) model.Model`; `func qoderTools(tools []string) (kept []string, dropped []string)`

- [ ] **Step 1: Vorab messen** (kein Test): `qodercli --help`, `qodercli -p -o json "say ok"` aufnehmen; die eingebauten Werkzeugnamen ermitteln. Tabelle Profilname → qoder-Name in `qoder.go` als Funktion, die eine frische Map zurückgibt (keine Paketvariable).
- [ ] **Step 2: Write the failing tests:**
  - `TestQoderArgs`: `qodercli`, `-p`, `-o json`, `-m <model>`, `--reasoning-effort <effort>`, `--tools <übersetzt>`; kein Bypass.
  - `TestQoderToolsDropsUnknown`: Werkzeug ohne Entsprechung → in `dropped`, nicht in den Args.
  - `TestQoderPromptCarriesSchema`: Prompt endet mit einer Anweisung, die `Schema(reply)` enthält.
  - `TestQoderRetriesOnce`: erste Antwort kein gültiges JSON, zweite gültig → Erfolg, Fake zweimal aufgerufen, zweiter Prompt nennt den Fehler; zwei ungültige → Fehler, genau zwei Aufrufe.
  - `TestQoderAgent`: `Agent != ""` → je nach Messung `--agents`-Form oder Fehler `qoder takes no agent by name yet`.
- [ ] **Step 3:** FAIL. **Step 4:** Umsetzen. **Step 5:** PASS.
- [ ] **Step 6:** Commit `feat(flow): run agent nodes through the qoder CLI`.

### Task 7: Verdrahtung in `loomux flow`

**Files:**
- Modify: `internal/cli/flow.go`, `internal/cli/flow_session.go`
- Create: `internal/flow/model/cli/new.go` mit `func New(provider string, o Options) (model.Model, error)`
- Test: `internal/cli/flow_test.go`, `internal/flow/model/cli/new_test.go`

**Interfaces:**
- Consumes: Tasks 1–6.
- Produces: `flowDeps.Models func(provider string, run model.Run) (model.Model, error)` mit `type Run struct { Root string; Timeout time.Duration }` in `internal/flow/model`.

- [ ] **Step 1: Write the failing tests:**
  - `TestNewPicksAdapter`: `claude`/`agy`/`qoder` → jeweiliger Typ; `gpt` → `no adapter for provider gpt`.
  - `TestFlowRunWarnsWhenToolsNotEnforced` (cli): Flow mit Agentenknoten auf `agy` und Fake-Model → stderr enthält einmal je Knoten `warning: node "draft" runs on agy, which cannot enforce the tool profile read_only; the barrier and the guard are the only limit`; Lauf geht weiter.
  - `TestFlowProductionUsesConfigTimeout`: `[agent] timeout = "5m"` → `Run.Timeout == 5*time.Minute`.
  - Der bisherige Test, der `no adapter for provider claude yet` erwartet, wird auf den neuen Weg umgestellt.
- [ ] **Step 2:** FAIL. **Step 3:** Umsetzen (`flowProduction.Models` → `cli.New(provider, cli.Options{Root, Timeout, Exec: cli.System})`). **Step 4:** `sh ci/gate.sh` → grün.
- [ ] **Step 5:** Commit `feat(flow): wire the CLI model adapters into loomux flow`.

### Task 8: Doku, Roadmap, Wiki

**Files:**
- Modify: `docs/en/flows.md`, `docs/de/flows.md`, `docs/en/configuration.md`, `docs/de/configuration.md`, `README.md`, `README.de.md`, `docs/wiki/syntheses/modelladapter-ueber-die-cli.md`, `docs/wiki/log.md`

- [ ] **Step 1:** `flows.md` (beide): „What runs today“ — Agentenknoten laufen über `claude`, `agy`, `qoder`; Abschnitt 3 um `agent`, `effort`, `base_url` (Strata-Beispiel), `timeout`, Vorrang, agy-Warnung.
- [ ] **Step 2:** `configuration.md` (beide): neue Schlüssel mit Vorgaben und Fehlermeldungen aus Task 1.
- [ ] **Step 3:** README beide: Zeile Flow-Laufzeit — Adapter gebaut; neue Zeile **Parallel agent nodes** / **Parallele Agentenknoten**: Fan-out/Join als Knotenart, Grenze der gleichzeitigen Agenten mit Vorgabe, konfigurierbar, ausschaltbar; hängt ab von Flow-Laufzeit; Priorität `—`.
- [ ] **Step 4:** Wiki-Seite: `status: accepted` → Abschnitt „Umgesetzt“ mit den drei Adaptern und der Warnung; `realization: implemented`; Log-Eintrag; nach dem Repin `sources[]`-Hashes über `loomux` nachziehen (Memory: Agent darf repinnen, zitierende Seiten vorher prüfen).
- [ ] **Step 5:** `sh ci/gate.sh` → grün (inkl. `lint/wiki`, `lint/wiki-sources`).
- [ ] **Step 6:** Commit `docs(flow): document the CLI model adapters`; danach Commits nach `release-pr` gruppieren (der Doku-Commit `aee042bc` geht in diesen auf), Label `release:minor`.
