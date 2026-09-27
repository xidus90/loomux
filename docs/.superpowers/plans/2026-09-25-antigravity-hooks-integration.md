# Antigravity Hooks Integration Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Vollständige Implementierung und Aktivierung der Antigravity-Lebenszyklus-Hooks (`PreInvocation`, `PreToolUse`, `PostToolUse`, `Stop`) im Loomux-Go-Binary inklusive 100 % Testabdeckung und Bereitstellung von `.agents/hooks.json`.

**Architecture:** Antigravity verlangt Exit-Code 0 bei geordneter Signalisierung und JSON auf `stdout` (`injectSteps` für Kontext, `decision: continue` für Stop-Tor-Blockaden); nur `pre-tool-use` behält Exit 2. Die Abbildung leistet ein Adapter `hosts.Answer`, den `loomux hook` einmal für jeden Austrittspfad ruft (Task 5); Post-Edit liest das Ziel aus dem `toolCall` des PostToolUse (Task 6). Da `PostToolUse` bei Antigravity keine Tool-Argumente liefert, puffert `PreToolUse` erlaubte Dateiänderungen temporär im Sitzungs-State ab. `internal/setup/hostfile` stellt alle 4 Einträge im Namensraum `loomux` bereit.

**Tech Stack:** Go (Standard-Bibliothek `encoding/json`, `io`, `os`), Antigravity ProtoJSON Hook-Protokoll, Loomux Verifikationskette.

**Spec:** [`docs/.superpowers/specs/2026-09-25-antigravity-hooks-integration-design.md`](../specs/2026-09-25-antigravity-hooks-integration-design.md)

## Global Constraints

- Coverage ist 100 % je Funktion (Ausnahmen nur mit `//coverage:exempt <reason>` direkt über `func`).
- Kein `init()` und keine Paket-Variablen, die eingebettete Daten parsen.
- Commit-Messages nach Conventional Commits: `<type>[(<scope>)][!]: <description>`.
- Arbeitsunterlagen unter `docs/.superpowers/` sind deutsch; Code, Kommentare, Fehlermeldungen und Commit-Nachrichten sind englisch.
- Branch: `feat/init-antigravity`.

---

### Task 1: Kontext-Emission für Antigravity (`internal/hosts`)

**Files:**
- Modify: `internal/hosts/antigravity.go:46-55`
- Test: `internal/hosts/antigravity_test.go`

**Interfaces:**
- Consumes: `hosts.HostAntigravity`, `WriteContext(host, event, w, lines)`
- Produces: `writeAntigravityContext(w io.Writer, event string, lines []string) error`

- [x] **Step 1: Tests für `writeAntigravityContext` anlegen**
- [x] **Step 2: Tests ausführen und Fehlschlag verifizieren**
- [x] **Step 3: `writeAntigravityContext` in `internal/hosts/antigravity.go` implementieren**
- [x] **Step 4: Tests ausführen und 100 % Coverage prüfen**
- [x] **Step 5: Commit**

---

### Task 2: Stop-Tor-Signalisierung für Antigravity (`internal/hooks`)

**Files:**
- Modify: `internal/hooks/stop.go`
- Test: `internal/hooks/stop_test.go`
- Modify: `internal/cli/hook.go`

**Interfaces:**
- Consumes: `hosts.HostAntigravity`
- Produces: `RunStop` mit Antigravity-JSON-Ausgabe (`decision: continue`) bei Exit 0 — ersetzt durch Task 5

- [x] **Step 1: Test für Antigravity Stop schreiben (`decision: continue` bei Exit 0)**
- [x] **Step 2: Test ausführen und Fehlschlag verifizieren**
- [x] **Step 3: Antigravity-Zweig in `internal/hooks/stop.go` implementieren**
- [x] **Step 4: Tests ausführen und 100 % Coverage prüfen**
- [x] **Step 5: Commit**

---

### Task 3: PostToolUse Pfad-Pufferung (`internal/sessions` & `internal/hooks`)

**Files:**
- Create: `internal/sessions/pending.go`
- Test: `internal/sessions/pending_test.go`
- Modify: `internal/hooks/pretool.go`
- Modify: `internal/hooks/post_edit.go`
- Test: `internal/hooks/post_edit_test.go`

**Interfaces:**
- Consumes: `sessions.RecordPendingEdit(root, sessionID string, stepIdx int, path string) error`
- Produces: `sessions.TakePendingEdit(root, sessionID string, stepIdx int) (string, error)`

- [x] **Step 1: Tests für `RecordPendingEdit` & `TakePendingEdit` in `internal/sessions` schreiben**
- [x] **Step 2: `internal/sessions/pending.go` implementieren und testen (100 % Coverage)**
- [x] **Step 3: Pufferung in `PreToolUse` und Abruf in `PostToolUse` verdrahten**
- [x] **Step 4: Tests ausführen und 100 % Coverage prüfen**
- [x] **Step 5: Commit**

---

### Task 4: Setup-Tabelle & Workspace-Aktivierung (`internal/setup/hostfile`)

**Files:**
- Modify: `internal/setup/hostfile/table.go`
- Test: `internal/setup/hostfile/table_test.go`
- Test: `internal/setup/hostfile/merge_test.go`
- Modify: `internal/setup/plan_test.go`
- Create / Update: `.agents/hooks.json`

**Interfaces:**
- Consumes: `table.Entries(hosts.HostAntigravity, binary)`
- Produces: 4 Einträge (`PreInvocation`, `PreToolUse`, `PostToolUse`, `Stop`) in `.agents/hooks.json`

- [x] **Step 1: Tests für alle 4 Antigravity-Einträge schreiben**
- [x] **Step 2: `internal/setup/hostfile/table.go` um `PreInvocation` und `Stop` erweitern**
- [x] **Step 3: Tests in `hostfile` und `plan_test.go` anpassen und ausführen (100 % Coverage)**
- [x] **Step 4: `loomux init` ausführen, um `.agents/hooks.json` im Workspace zu aktualisieren**
- [x] **Step 5: Pre-Commit Gate (`ci/gate.sh`) und kanonisches Binary bauen**
- [x] **Step 6: Commit**

---

### Task 5: Nacharbeit nach dem Review vom 2026-09-25

Das Review fand an Task 2 und 3 Lücken; Task 2 ist dabei ersetzt worden.

**Files:**
- Create: `internal/hosts/answer.go`, `internal/hosts/answer_test.go`
- Modify: `internal/cli/hook.go`, `internal/cli/hook_test.go`
- Revert: `internal/hooks/stop.go`, `internal/hooks/stop_test.go`, `internal/cli/cases_2c_test.go` auf den Stand von `master` (der Antigravity-Zweig in `RunStop` entfällt)
- Modify: `internal/hooks/guard.go`, `internal/hooks/guard_test.go`
- Modify: `internal/sessions/pending.go`, `internal/sessions/pending_test.go`
- Modify: `internal/hooks/pretool.go`, `internal/hooks/post_edit.go` und ihre Tests
- Docs: Fusions-Spec #23, diese Spec, `docs/{en,de}/{migration,cli-reference,getting-started}.md`, beide READMEs, `parity/stufe-4a-2.md`

**Interfaces:**
- Produces: `hosts.Answer(host, event string, w io.Writer, code int, out []byte, reason string) int`
- Produces: `sessions.TakePendingEdits(root, sessionID string, stepIdx int) ([]string, error)` statt `TakePendingEdit`

- [x] **Step 1: `run_command` in `commandTools` mit den drei Schreibweisen, Verweigerung ohne Befehlszeile**
- [x] **Step 2: `hosts.Answer` und der Aufruf in `cli/hook.go`; `RunStop` wieder wirtsneutral**
- [x] **Step 3: Pufferung je Aufruf, Abruf aller Ziele eines Schritts, auch bei `error`**
- [x] **Step 4: Tests, 100 % je Funktion, Gate**
- [x] **Step 5: Doku**
- [x] **Step 6 (Mensch): die Probe aus `parity/stufe-4a-2.md` mit einem laufenden agy** (lief 2026-09-25 mit agy 1.2.11)

---

### Task 6: Umbau nach der Probe mit agy 1.2.11

**Files:**
- Modify: `internal/setup/hostfile/{table,merge}.go` und Tests, `.agents/hooks.json` (flache `Stop`/`PreInvocation`, `send_command_input` im Matcher)
- Delete: `internal/sessions/pending.go`, `internal/sessions/pending_test.go`
- Modify: `internal/hooks/{pretool,post_edit,guard,hook_session_start}.go` und Tests
- Modify: `internal/hosts/{answer,antigravity,hostio}.go` und Tests, `internal/cli/hook.go` und Tests
- Docs: Fusions-Spec #23, diese Spec, `docs/{en,de}/{migration,cli-reference,getting-started,hooks}.md`, beide READMEs, `parity/stufe-4a-2.md`

- [x] **Step 1: flache Form für `Stop` und `PreInvocation` (`Entry.Flat`)**
- [x] **Step 2: Post-Edit liest `toolCall`; die Ablage entfällt**
- [x] **Step 3: `post-tool-use` behält Exit 2 unter agy; unbekanntes Ereignis bleibt 2; Panik über den Adapter**
- [x] **Step 4: `send_command_input` durch die Befehlsregeln**
- [x] **Step 5: `session-start` meldet sich nur beim ersten `invocationNum`**
- [x] **Step 6: Budget 0 bleibt unbegrenzt**
- [x] **Step 7: Doku, Gate**

**Nachtrag 2026-09-27 (Code-Review):** Step 4 und Step 5 beschreiben den Stand
vom 2026-09-25. Seither steht auch `manage_task` im Matcher von `PreToolUse`
(`writers + "|run_command|send_command_input|manage_task"`) und in den
Befehlsregeln; jede gefundene Zeile wird geprüft, gleich welche `Action` der
Aufruf nennt, die Argumentnamen ohne Rücksicht auf die Schreibung, und was
`send_command_input` und `manage_task` tippen, geht nur als ganze Zeilen ohne
Steuerzeichen und ohne Zeilenfortsetzung durch; ein Wert unter einem
Zeilenschlüssel, der kein String ist, macht den Aufruf unprüfbar und wird
verweigert; ein leerer String trägt keine Zeile; ein Aufruf ohne
Werkzeugnamen wird verweigert.
`session-start` sagt bei einem späteren `invocationNum` (Zahl oder
Dezimal-String) nichts und belebt die Sitzung nicht wieder. Gemessen am
2026-09-27 mit agy 1.2.11: `invocationNum` kommt als JSON-Zahl und beginnt bei
0 (0, 1, 2, 3 je Modellaufruf), ein späterer Aufruf ist also
`invocationNum > 0`; ungemessen bleiben nur die String-Form eines
64-Bit-Zählers und die Breite des Felds. post-edit schreibt
seinen Kontext bei Exit 0 über `hosts.WriteContext`, für agy als
`injectSteps`, das `hosts.Answer` verwirft (ob agy es auf PostToolUse liest,
ist ungemessen); bei Exit ≠ 0 schreibt es nichts auf stdout. Die Aussage unter
„Architecture“, Antigravity bekomme Kontext als `injectSteps` auf stdout, gilt
damit nur für `session-start`. Stand der Regeln: `parity/stufe-4a-2.md`,
„Nachtrag nach dem Code-Review, 2026-09-27“.
