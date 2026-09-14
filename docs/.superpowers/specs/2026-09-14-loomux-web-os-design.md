# loomux — LLM OS Web-Interface & Skill-System (Säule 2 & 5)

**Datum:** 2026-09-14  
**Stand:** entworfen, zur Umsetzung im Anschluss an die Fusions- und Graph-Stufen  
**Ort:** `docs/.superpowers/specs/2026-09-14-loomux-web-os-design.md`  
**Ergänzt:**  
- `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` (Säule 5: *LLM OS*, Folgeprojekte 2 & 3)  
- `docs/.superpowers/specs/2026-09-14-loomux-code-graph-design.md` (Säule 3: *Graph + Loop Engineering*)  
**Referenzen:** `ultra-brain/web` (React/Vite Wiki-App), `trailhq/Graft` (`viewer/` & `src/viz/`)  

---

## 1. Vision & Zielsetzung

Loomux vereint fünf Säulen in einem einzigen Go-Binary:
> **Hooks · Skills · Graph + Loop Engineering · Second Brain / LLM Wiki · LLM OS**

Bisher drohte die Gefahr isolierter Teil-Oberflächen (ein Wiki-Viewer aus `ultra-brain`, ein separater Graph-Viewer aus Graft, ein späterer Flow-Editor). Diese Spezifikation definiert das **ganzheitliche Loomux Web OS**:

1. **Eine zentrale, voll integrierte Web-Oberfläche (`loomux/web`)**:
   Kompiliert als React/Vite-Single-Page-Application und per `go:embed` direkt in das Go-Binary `loomux` eingebettet. Keine Node.js-Laufzeit auf dem Zielsystem, kein separater Webserver — `loomux serve` liefert die UI autark unter `http://127.0.0.1:<port>` aus.
2. **Säule 2: Vorkonfigurierte Skills & Sprachspezifische Code-Reviews**:
   Loomux liefert kuratierte Best-Practice- und Code-Review-Skills (z. B. für Go, Python, TypeScript, Rust) direkt im Binary aus und provisioniert sie automatisch für alle unterstützten Agenten-Hosts (Claude Code, Antigravity, Cursor).
3. **Graph- und Brain-gestützte Governance**:
   Code-Reviews und Agenten-Aktionen finden nicht im luftleeren Raum statt: Skills nutzen den Code-Graphen (`graph_blast`) und das Second Brain (Wiki-ADRs), um Änderungen architektonisch fundiert zu bewerten.

---

## 2. Das OS-Desktop-Konzept

Die UI ist wie ein Entwickler-Betriebssystem für Agenten aufgebaut:

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│  loomux OS ── [● Daemon: aktiv] [qmd: warm] [Freshness: ok]        [ ⌘K Suchen ] │
├──────────────┬──────────────┬──────────────┬──────────────┬──────────────────────┤
│ 1. Brain     │ 2. CodeGraph │ 3. Skills    │ 4. Flows     │ 5. Governance        │
│ (Wiki / ADR) │ (Graft-Viz)  │ (Review & BP)│ (Loop Editor)│ (Wächter & Verify)   │
├──────────────┴──────────────┴──────────────┴──────────────┴──────────────────────┤
│                                                                                  │
│  [ Multi-Panel Workspace / Geteilte Ansichten ]                                 │
│                                                                                  │
│  ┌─────────────────────────────────┬──────────────────────────────────────────┐  │
│  │ Code Graph / Visualizer         │ Detail- & Verknüpfungs-Drawer            │  │
│  │ (D3-Force / WebGL)              │                                          │  │
│  │                                 │ - Symbol: internal/hooks/guard.go        │  │
│  │ - Zoom, Pan, Type-Filter        │ - Crux: Zeilen L45-L62 (inline)          │  │
│  │ - Blast-Radius-Overlay (Rot)    │ - Aufrufer: 8 | Aufgerufene: 3           │  │
│  │ - Kanten: calls, imports, uses  ├──────────────────────────────────────────┤  │
│  │                                 │ Verknüpfte Wiki-Dokumente & ADRs:        │  │
│  │                                 │ 📄 docs/wiki/Policy-Schutz.md            │  │
│  │                                 │ 📄 ADR-003: Schreibschranke-Regeln       │  │
│  └─────────────────────────────────┴──────────────────────────────────────────┘  │
│                                                                                  │
│  [ Statusleiste: Aktive Sitzung: Claude Code | Diff-Blast: 3 Dateien | Mode: Dev]│
└──────────────────────────────────────────────────────────────────────────────────┘
```

### 2.1 Globale OS-Komponenten
- **System-Statusleiste (Header)**: Zeigt Daemon-Status, qmd-Modell-Zustand, Arbeitsbaum-Frische (`Fresh`/`Stale`) und aktive Agenten-Sitzungen in Echtzeit.
- **Universal Command Center (`Cmd/Ctrl+K`)**:
  Schnellsuche über alle Subsysteme hinweg:
  - Springe zu Wiki-Seite oder ADR (`brain:...`)
  - Springe zu Code-Symbol oder Datei (`graph:...`)
  - Führe Skill oder Code-Review aus (`skill:review-...`)
  - Öffne Flow-Definition (`flow:...`)
- **Live-Event-Bus via SSE (`/api/events`) & Entkoppeltes Journal**:
  Der Wächter (`loomux hook pre-tool-use`, `post-tool-use`) darf **niemals** synchrone HTTP-, TCP- oder IPC-Calls an `loomux serve` absetzen — das würde Latenzen (>10ms) und Single-Point-of-Failure-Risiken einführen (was, wenn `serve` nicht läuft?).
  Stattdessen gilt strikte Entkopplung:
  1. Hooks hängen Ereignisse im atomaren Append-Modus (`O_APPEND`) in `<0.2ms` an `.loomux/state/journal/events.jsonl` an.
  2. Läuft `loomux serve`, überwacht ein leichtgewichtiger File-Watcher (Inotify/Read-Tail) das Journal und spiegelt neue Zeilen sofort in den SSE-Kanal `/api/events`.
  3. Läuft `loomux serve` nicht, arbeitet der Hook-Pfad völlig unbeeinträchtigt und lautlos weiter.

---

## 3. Die 5 Kern-Module im Detail

### Modul 1: Second Brain & LLM Wiki (aus `ultra-brain/web`)
- **Wiki-Reader & -Editor**: Zweispaltige Ansicht (Markdown / gerendertes HTML) mit bidirektionalen Wiki-Links (`[[Seitenname]]`).
- **Wissens-Graph**: Visualisierung der Beziehungen zwischen Themen, Konzepten, RFCs und Identitäten.
- **ADR-Register**: Zentrale Übersicht aller Architekturentscheidungen mit Status (*Proposed*, *Accepted*, *Deprecated*).

### Modul 2: Code & Architecture Graph (Erbe aus `trailhq/Graft` `viewer/`)
- **D3-Force / WebGL Graph**:
  Portierung der visuellen Mechanik aus Grafts `viewer/`:
  - Kanten-Chips zum Filtern nach Relation (`part of`, `uses`, `implements`, `extends`, `calls`).
  - Knoten-Typen-Legende (Functions, Methods, Classes, Interfaces, Files).
  - Outline-Baum der Verzeichnisse und Symbole.
- **Interaktiver Blast-Radius**:
  Klick auf ein Symbol markiert alle transitiven Aufrufer rot (optische Darstellung der Auswirkung vor einem Refactoring).
- **Die Brain-Brücke**:
  Jeder Code-Knoten zeigt automatisch verknüpfte Wiki-Konzepte und ADRs an, die dieses Symbol oder Paket dokumentieren.

### Modul 3: Skills & Code-Review Engine (Säule 2)
- **Das 3-Kanal-Distributionsmodell**:
  Loomux bedient drei komplementäre Kanäle zur Bereitstellung von Fähigkeiten und Leitplanken:
  1. *Host-Native Filesystem Skills*: Autonomes Schreiben standardisierter `SKILL.md`-Dateien in `.claude/skills/<name>/SKILL.md`, `.agents/skills/<name>/SKILL.md` für native Invocation durch Claude Code und Antigravity.
  2. *System Prompt / Instruction Injection*: Injizieren prägnanter Leitplanken in `.cursorrules`, `CLAUDE.md` oder `AGENTS.md`.
  3. *Native MCP Prompts*: Loomux registriert standardisierte Prompts über das Model Context Protocol (z. B. `mcp://prompts/code-review`, `mcp://prompts/blast-analysis`), die Hosts dynamisch und ohne Filesystem-Verschmutzung abrufen können.
- **Vorkonfigurierte Best-Practice-Suiten**:
  Loomux bringt kuratierte, erprobte Review-Regeln direkt im Binary mit:
  - `review-go`: Idiomatisches Fehlerhandling, Goroutine-Leaks, Zero-Allocation-Pfade, Coverage-Garantie (100% per Function), Vermeidung von Paket-`init()`.
  - `review-python`: Type-Annotations (Strict Typing), Vermeidung globaler Zustände, API-Sicherheit.
  - `review-typescript`: Strict Null-Checks, sauberes Exception- und Promise-Handling, Dependency-Hygiene.
  - `review-security`: Input-Validierung, Pfad-Traversierung, Secrets-Erkennung, Permission-Eskalation.
- **Graph-gestützter Code-Review**:
  Der Review-Skill liest nicht nur das Git-Diff, sondern ruft `graph_trace_calls` und `loomux graph blast` auf:
  - *„Diese Änderung an Funktion `X` bricht 4 externe Aufrufer in Modul `Y`, für die keine Test-Anpassungen im PR vorliegen.“*
  - *„Die Änderung widerspricht Architekturentscheidung ADR-002 aus dem Second Brain.“*
- **Skill-Katalog & Provisionierung**:
  Im Web OS können Skills per Schalter aktiviert/deaktiviert und mit einem Klick synchronisiert werden.

### Modul 4: Flow, Loop Engineering & Kanban Board (Folgeprojekt 3)
- **Visuelles Kanban Board für Subagent- & Loop-Tracking**:
  Echtzeit-Tracking von mehrstufigen Agenten-Workflows, Subagenten-Loops und Prüfketten:
  - **Spalten**:
    - *Backlog / Plan*: Geplante Phasen, Tasks aus `writing-plans` und User-Prompts.
    - *In Progress*: Aktive Loops (`verify_until_green`, Subagent-PIDs, aktueller Tool-Call, Token-Budget-Verbrauch).
    - *Verification*: Laufende Prüflanes (Pre-Commit-Gating, Test-Suiten, Covergate-Prüfung, Diff-Blast-Impact).
    - *Done / Blocked*: Erfolgreich abgeschlossene Arbeiten mit Commit-Referenz bzw. durch Wächter/Policy blockierte Schritte mit Fehlerursache.
  - **Interaktiver Task-Drilldown**: Klick auf eine Kanban-Karte öffnet das Tool-Call-Journal, den Diff-Inspector und die zugehörigen Crux-Karten.
- **Visueller DAG-Editor**:
  Agenten-Workflows (Daten-Flows aus `ulflow`: `verify_until_green`, Loop-Iterationen) als interaktiver Knotengraph.
- **Journal-Replay & Time-Travel**:
  Schritt-für-Schritt-Wiedergabe vergangener Agenten-Läufe mit Anzeige von Inputs, Tool-Calls, Token-Verbrauch und Prüfergebnissen.

### Modul 5: Governance & Live Monitor
- **Echtzeit-Audit-Stream**:
  Protokoll aller Wächter-Aufrufe (`PreToolUse`, `PostToolUse`). Blockierte Aktionen (Policy-Verstöße) werden rot hervorgehoben.
- **Prüfketten-Dashboard (`[verify]`)**:
  Status aller konfigurierten Lanes (Pre-Commit, Linter, Tests, Coverage-Tor).
- **Token- & Kosten-Dashboard**:
  Aufsummierte Token-Ersparnis durch Graph-Retrieval (Crux vs. Full File Reads) und Cache-Hits.

---

## 4. Backend-Architektur & APIs (`loomux serve`)

Alle UI-Funktionen werden über einheitliche JSON-APIs und SSE vom Go-Binary bedient:

```
loomux serve (Port dynamisch/konfiguriert, Token-Bootstrap via Launch-URL)
├── GET  /                           -> Statische SPA (index.html, assets via go:embed)
├── GET  /api/events                 -> SSE Event-Stream (Live-Updates aus events.jsonl)
│
├── REST /api/brain/                 -> Wiki-Seiten, ADRs, semantischer Graph
│   ├── GET  /pages
│   ├── GET  /page/:id
│   └── GET  /graph
│
├── REST /api/graph/                 -> Code-Graph & Blast Radius
│   ├── GET  /wiring                 -> Vollständiger Symbol- und Kantengraph
│   ├── GET  /cards/:path_hash       -> Lazy-geladene Datei-Karte mit Crux-Spans
│   ├── GET  /blast?symbol=X&depth=N -> Transitive Hülle für Symbol X
│   └── GET  /diff-blast?base=HEAD~1 -> Blast-Radius des aktuellen Git-Diffs
│
├── REST /api/skills/                -> Skill-Registry & Review-Engine
│   ├── GET  /catalog                -> Verfügbare Best-Practice- & Review-Skills
│   ├── POST /install                -> Schreibt Skill in Host-Konfigurationen
│   └── POST /run-review             -> Führt graph-gestützten Review auf Diff aus
│
├── REST /api/kanban/                -> Workflow- & Subagent-Zustand
│   ├── GET  /boards                 -> Aktuelle Spalten & Karten
│   └── POST /tasks/:id/move         -> Manuelle oder agentische Statusänderung
│
└── REST /api/governance/            -> Live-Zustand, Wächter-Logs, Verify-Lanes
```

### 4.1 Token-Bootstrap & Authentifizierung
1. Beim Start von `loomux serve` generiert das Binary ein kryptografisches Einmal-Token (32 Byte Hex).
2. Auf der Konsole wird die Start-URL ausgegeben: `http://127.0.0.1:<port>/?token=<hex>`.
3. Der erste Browser-Aufruf validiert das Query-Token, setzt ein sicheres Session-Cookie (`HttpOnly; SameSite=Strict; Path=/`) und leitet per `302 Found` auf `/` um.
4. Folge-Requests und der SSE-Kanal nutzen das Cookie automatisch — keine Token-Lecks in der Browser-Adressleiste oder im Referrer-Header.
5. Automatisierte Clients oder MCP-Hosts können alternativ den Header `Authorization: Bearer <token>` mitsenden.

---

## 5. Frontend-Technologie-Stack & Zero-Node Go-Builds

1. **Framework**: React 19 + Vite (TypeScript strict).
2. **Styling**: Modernes, komponentenbasiertes Dark/Light-Designsystem (Tailwind CSS oder CSS-Variablen, konsistent mit Loomux-Aesthetik).
3. **Graph-Engine**: D3-Force für 2D-Layouts (portiert und modularisiert aus Grafts `viewer/`), optional PixiJS/WebGL bei Graphen mit > 5.000 Knoten.
4. **Markdown & Code**: Shiki oder Prism für Syntax-Highlighting der Crux-Spans und Markdown-Vorschau.
5. **Build-Artefakt & Zero-Node Fallback (`embed_stub.go`)**:
   - Produktion: Vite baut nach `internal/web/dist/`. `internal/web/embed.go` mit Build-Tag `//go:build prod` bettet die echten Assets via `//go:embed all:dist` ein.
   - Entwicklungs- & CI-Standard: `internal/web/embed_stub.go` mit Build-Tag `//go:build !prod` liefert ein minimales In-Memory-Dateisystem mit einer informativen Platzhalter-HTML-Seite (*„Loomux Web OS UI not compiled. Run npm run build in internal/web to generate bundle.“*).
   - **Garantie**: `go test ./...`, `go build ./...` und der Git-Pre-Commit-Hook funktionieren auf einem frischen Clone sofort und ohne Node.js/npm-Abhängigkeit!

---

## 6. Stufenplan für das Web OS & Skill-System

Die Umsetzung schließt an die Fusions-Spec und die Code-Graph-Spec an:

| Stufe | Meilenstein | Inhalt |
|---|---|---|
| **W1** | **OS-Shell & Basis** | React/Vite-Gerüst unter `internal/web`, `go:embed`-Auslieferung mit `embed_stub.go`-Fallback in `loomux serve`, SSE-Eventbus `/api/events`, Theme- und Layout-System, Command-Palette (`⌘K`), Token-Bootstrap. |
| **W2** | **Second Brain Integration** | Umzug der React-Komponenten aus `ultra-brain/web`, Markdown-Editor, ADR-Katalog, Wissens-Graph. |
| **W3** | **Code-Graph-Visualizer** | Portierung der D3-Visualisierung aus Graft (`viewer/`): Kanten-Chips, Knoten-Legende, Crux-Inspector, Blast-Radius-Overlay. Verknüpfung von Code-Knoten mit Brain-ADRs. |
| **W4** | **Skill-System & Code-Review** | Eingebettete Best-Practice- und Review-Skills (Go, Python, TS, Security) über 3 Kanäle (Filesystem, Prompts, MCP Prompts). Graph-gestützte PR-/Diff-Review-Engine. |
| **W5** | **Flow-Editor, Kanban & Governance** | Grafischer DAG-Editor für Agenten-Loops (ulflow M1–M3), interaktives Kanban Board für Subagenten-Tracking, Live-Wächter-Log, Verify-Dashboard. |

---

## 7. Offene Punkte & Verifikationen vor der Umsetzung

- **Bundle-Größe & Startzeit**: Die kompilierte SPA darf die Binärgröße von `loomux.exe` um nicht mehr als ~5–8 MB erhöhen. Der Kaltstart des CLI-Kerns darf durch das `go:embed`-Dateisystem nicht beeinträchtigt werden.
- **Graph-Rendering-Performance**: Benchmarking der D3-Force-Engine bei Repositories mit > 10.000 Symbolen (Evaluierung des Wechsels zu WebGL/Canvas ab Schwellenwert).
- **Skill-Host-Kompatibilität**: Testen der generierten Skill-Markdown-Dateien auf allen Ziel-Hosts (Claude Code, Antigravity, Cursor) auf identische Regelauslegung.
- **Kanban-Sync-Robustheit**: Sicherstellen, dass das Tailing des Journals `.loomux/state/journal/events.jsonl` auch bei intensiven Dateioperationen und parallelen Subagenten absolut verzugsfrei bleibt.
