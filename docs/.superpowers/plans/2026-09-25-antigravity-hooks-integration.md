# Antigravity Hooks Integration Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Vollständige Implementierung und Aktivierung der Antigravity-Lebenszyklus-Hooks (`PreInvocation`, `PreToolUse`, `PostToolUse`, `Stop`) im Loomux-Go-Binary inklusive 100 % Testabdeckung und Bereitstellung von `.agents/hooks.json`.

**Architecture:** Antigravity verlangt Exit-Code 0 bei geordneter Signalisierung und JSON auf `stdout` (`injectSteps` für Kontext, `decision: continue` für Stop-Tor-Blockaden). Da `PostToolUse` bei Antigravity keine Tool-Argumente liefert, puffert `PreToolUse` erlaubte Dateiänderungen temporär im Sitzungs-State ab. `internal/setup/hostfile` stellt alle 4 Einträge im Namensraum `loomux` bereit.

**Tech Stack:** Go (Standard-Bibliothek `encoding/json`, `io`, `os`), Antigravity ProtoJSON Hook-Protokoll, Loomux Verifikationskette.

**Spec:** [`docs/.superpowers/specs/2026-09-25-antigravity-hooks-integration-design.md`](file:///c:/Users/micro/Documents/#GIT/loomux/docs/.superpowers/specs/2026-09-25-antigravity-hooks-integration-design.md)

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

- [ ] **Step 1: Tests für `writeAntigravityContext` anlegen**
- [ ] **Step 2: Tests ausführen und Fehlschlag verifizieren**
- [ ] **Step 3: `writeAntigravityContext` in `internal/hosts/antigravity.go` implementieren**
- [ ] **Step 4: Tests ausführen und 100 % Coverage prüfen**
- [ ] **Step 5: Commit**

---

### Task 2: Stop-Tor-Signalisierung für Antigravity (`internal/hooks`)

**Files:**
- Modify: `internal/hooks/stop.go`
- Test: `internal/hooks/stop_test.go`
- Modify: `internal/cli/hook.go`

**Interfaces:**
- Consumes: `hosts.HostAntigravity`
- Produces: `RunStop` mit Antigravity-JSON-Ausgabe (`decision: continue`) bei Exit 0

- [ ] **Step 1: Test für Antigravity Stop schreiben (`decision: continue` bei Exit 0)**
- [ ] **Step 2: Test ausführen und Fehlschlag verifizieren**
- [ ] **Step 3: Antigravity-Zweig in `internal/hooks/stop.go` implementieren**
- [ ] **Step 4: Tests ausführen und 100 % Coverage prüfen**
- [ ] **Step 5: Commit**

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

- [ ] **Step 1: Tests für `RecordPendingEdit` & `TakePendingEdit` in `internal/sessions` schreiben**
- [ ] **Step 2: `internal/sessions/pending.go` implementieren und testen (100 % Coverage)**
- [ ] **Step 3: Pufferung in `PreToolUse` und Abruf in `PostToolUse` verdrahten**
- [ ] **Step 4: Tests ausführen und 100 % Coverage prüfen**
- [ ] **Step 5: Commit**

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

- [ ] **Step 1: Tests für alle 4 Antigravity-Einträge schreiben**
- [ ] **Step 2: `internal/setup/hostfile/table.go` um `PreInvocation` und `Stop` erweitern**
- [ ] **Step 3: Tests in `hostfile` und `plan_test.go` anpassen und ausführen (100 % Coverage)**
- [ ] **Step 4: `loomux init` ausführen, um `.agents/hooks.json` im Workspace zu aktualisieren**
- [ ] **Step 5: Pre-Commit Gate (`ci/gate.sh`) und kanonisches Binary bauen**
- [ ] **Step 6: Commit**
