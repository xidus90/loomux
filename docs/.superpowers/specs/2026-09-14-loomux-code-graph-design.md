# loomux — Code-Graph & Graph Engineering (Säule 3)

**Datum:** 2026-09-14  
**Stand:** entworfen; **G1 umgesetzt 2026-09-17**, **G2a umgesetzt 2026-09-18**, beide vorgezogen vor
Fusions-Stufe 1b-2. Berichtigt durch [`2026-09-16-loomux-code-g1-delta.md`](2026-09-16-loomux-code-g1-delta.md)
und [`2026-09-17-loomux-code-g2-design.md`](2026-09-17-loomux-code-g2-design.md) — bei Widerspruch
gelten die beiden jüngeren Dokumente. G2b umgesetzt; **G3 umgesetzt 2026-09-19**, verengt durch
[`2026-09-18-loomux-code-g3-delta.md`](2026-09-18-loomux-code-g3-delta.md) — auch hier gilt bei
Widerspruch das jüngere Dokument. G4 und G5 offen  
**Ort:** `docs/.superpowers/specs/2026-09-14-loomux-code-graph-design.md`  
**Ergänzt:** `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` (Säule: *Graph + Loop Engineering*)  
**Referenz-Analyse:** `trailhq/Graft` (TypeScript, Node.js)  

---

## 1. Ziel und Einordnung

Das Loomux-Fusion-Design definiert fünf Säulen für das vereinte Go-Binary:
> **Hooks · Skills · Graph + Loop Engineering · Second Brain / LLM Wiki · LLM OS**

Während die Fusions-Spec die Grundlagen (Hooks, Wächter, Prüfkette, Second Brain/Wiki und MCP-Dienst) abdeckt, spezifiziert dieses Dokument das Subsystem **Graph + Loop Engineering**:

1. **Deterministischer Code- & Aufrufgraph ($0, kein LLM)** über Symbole, Aufrufe, Typ-Hierarchien und Importe.
2. **Personalized PageRank ("GraphRank")** für präzises Code-Retrieval ohne Vektordatenbank („Lexik schlägt vor, der Graph entscheidet“).
3. **Blast-Radius-Analyse** für Refactorings und als Information im Post-Edit-Hook.
4. **Verschachtelte MCP-Architektur (Gateway)** in `loomux serve` mit getrennten Sub-Servern für `brain_*` (Wissen) und `graph_*` (Code).
5. **Verheiratung von Code-Graph und Second Brain**: Code-Symbole verweisen direkt auf lokale Wiki-Konzepte und Architektur-Entscheidungen (ADRs).

Alle Kernprinzipien von Loomux bleiben ausnahmslos gewahrt: **Ein Go-Binary, kein Python, kein Node.js, CGo-frei (`CGO_ENABLED=0`), 100 % Test-Coverage je Funktion, Kaltstart < 35 ms.**

---

## 2. Befunde aus dem Scan von `trailhq/Graft`

Graft (`@nanonets/graft`) löst das Problem des blinden Wieder-Erkundens von Repositories durch Coding-Agenten. Die Analyse des Repositories ergab:

| Komponente in Graft | Funktionsweise | Loomux-Entscheidung |
|---|---|---|
| **Tier 1 Code-Graph** | Tree-sitter AST-Extraktion (23 Sprachen), JSON-Kanten (`wiring.json`) | **Übernehmen**: Nativ in Go (`go/parser` + WASM-Tree-sitter via `wazero`) |
| **Personalized PageRank** | Power-Iteration (Random Walk mit Restart, `alpha=0.25`), Seeds aus Lexik (`src/ask/graphrank.ts`) | **Übernehmen**: Mathematischer Pure-Go-Algorithmus (< 1 ms in Go) |
| **Inkrementelle Frische** | Mtime/Byte-Hash vor jeder Abfrage (~3 ms) (`src/context/check.ts`) | **Übernehmen**: Im Hook nur geänderte Datei aus `stdin` prüfen |
| **Crux-Extraktion** | Signatur + Kernlogik-Spans (5–10 Zeilen) (`src/ask/ask.ts`) | **Übernehmen**: AST-gestützte Span-Extraktion |
| **Laufzeit-Stack** | Node.js (>=20, `package.json:54-56`), native C++-Bindings via `node-gyp` (`package.json:86-99`) | **Verwerfen**: Widerspricht Loomux. Reines Go ohne C-Compiler |
| **Trail Brain Anbindung** | Schickt Symbol-Hashes an Cloud-API (`src/brain/link.ts:19,164`) | **Ersetzen durch Second Brain**: Loomux verknüpft Symbole lokal mit Wiki & ADRs |
| **Telemetrie** | Nutzungsstatistiken an externe Server (`src/telemetry/`, `TELEMETRY.md`) | **Verwerfen**: Loomux hat null Telemetrie und telefoniert niemals nach Hause |

---

## 3. Architektur & Paketstruktur

### 3.1 Pakete unter `internal/`

```
internal/
├── graph/                  # Kern-Subsystem (Algorithmen & Parsing)
│   ├── model/              # Node, Edge, Span, WiringGraph, FileCard
│   ├── pagerank/           # Reiner Algorithmus: Personalized PageRank (Power-Iteration)
│   ├── blast/              # BFS/DFS, In/Out-Kanten, transitive Hülle (Blast Radius)
│   ├── freshness/          # Mtime/Hash-Check für Arbeitsbaum und Einzeldateien
│   ├── crux/               # Crux- & Signatur-Extraktion ($0)
│   └── extract/            # AST-Extraktoren Interface
│       ├── golang/         # Go-Parser (go/parser, go/types)
│       └── wasm/           # Multi-Language Tree-sitter (wazero, wasm32-wasi)
├── serve/
│   └── graph/              # MCP-Handler für den Code-Graphen (trennt Hooks von MCP-SDK)
```

### 3.2 Strikte Startzeit- und Abhängigkeitsisolierung

Da in einem einzelnen Go-Binary die `init()`-Funktionen aller gelinkten Pakete vor `main()` laufen, muss verhindert werden, dass `wazero` oder `go-sdk` den Kaltstartboden (~32 ms) von `loomux hook pre-tool-use` belasten:

1. **Lazy Loading von wazero**: Das Paket `extract/wasm` initialisiert die `wazero`-Runtime erst beim ersten tatsächlichen Parse-Aufruf (`sync.Once`), niemals auf Paketebene oder in `init()`.
2. **Architektur-Trennung**: `hooks` importiert nur `graph/blast` und `graph/freshness` (reine Datenstrukturen und Hashes, null Fremdabhängigkeiten). Die MCP-Tool-Definitionen liegen isoliert in `internal/serve/graph`.
3. **Zweistufiges Qualitäts- und Startzeittor**:
   - *Abhängigkeitstor:* `go list -deps ./internal/hooks | grep -E 'go-sdk|wazero'` muss leer bleiben.
   - *Inittrace-Boden:* Vor Releases und im Pre-Commit misst `GODEBUG=inittrace=1 bin/loomux.exe --version` die Gesamtzeit aller `init()`-Aufrufe. Der Wert muss unter 1 ms bleiben und wird in `docs/benchmarks.md` protokolliert.

---

## 4. Verschachtelte MCP-Architektur (MCP Gateway)

`loomux serve` fungiert als modularer **MCP Composite Router / Gateway**.

```
                           Coding Agent (Claude / Antigravity / Codex)
                                             │ (stdio)
                                     loomux mcp (Brücke)
                                             │ (Streamable HTTP, localhost)
                                  ┌──────────┴──────────┐
                                  │    loomux serve     │
                                  │ (MCP Composite Root)│
                                  └──────────┬──────────┘
                         ┌───────────────────┼───────────────────┐
                         ▼                   ▼                   ▼
                 Sub-Server: brain    Sub-Server: graph   Upstream-Proxies
                 (Wissen, Wiki, qmd)  (Code-Graph, AST)   (z. B. qmd mcp, LSP)
```

### 4.1 Namensraum und Validität
Alle Tool-Namen genügen der MCP-Spezifikation `^[a-zA-Z0-9_-]{1,64}$`:

- **Wissens-Tools (`brain`):** `brain_search`, `brain_catalog`, `brain_read`, `brain_neighbors`, `brain_status`.
- **Code-Graph-Tools (`graph`):** Die 6 Kern-Tools des Code-Graphen:
  1. **`graph_find_code`**:
     - *Parameter:* `query` (string, Pflicht), `limit` (int, Standard 5), `full` (bool: ganzes Definitions-Span statt $\le$ 8 Zeilen Crux-Exzerpt), `in` (string: Pfad-Präfix-Filter).
     - *Funktion:* Plain-Text-Suche über Symbole, gerankt via Personalized PageRank ("GraphRank"), mit Dateipfaden, Zeilen-Spans und inline Kerncode.
  2. **`graph_file_api`**:
     - *Parameter:* `file` (string, Pflicht: relativer Pfad oder eindeutiger Basename).
     - *Funktion:* Reine Signatur- und Typ-Übersicht einer Datei ohne Rümpfe (~10× günstiger als Dateilesen, $0, kein LLM).
  3. **`graph_trace_calls`**:
     - *Parameter:* `symbol` (string, Pflicht: Name, `Class.method`, `pkg.Fn` oder Dateipfad), `direction` (`in` für Aufrufer/Abhängige [Standard], `out` für Aufgerufene/Abhängigkeiten), `depth` (int oder `"all"` für transitive Hülle), `in` (string: Pfad-Präfix).
     - *Funktion:* Strukturelle Kanten über Aufrufe, Referenzen, Vererbung und Importe. Dient der Blast-Radius-Ermittlung vor Refactorings.
  4. **`graph_find_all`**:
     - *Parameter:* `pattern` (string, Pflicht), `in` (string: Pfad-Präfix), `ignore_case` (bool), `fixed` (bool: Literal statt Regex).
     - *Funktion:* Volltext-/Regex-Suche über alle indizierten Dateien, Treffer **gruppiert nach umschließendem Symbol** und **gerankt nach eingehendem Kanten-Grad (`inDegree` / Kopplung)**. Beantwortet dem Agenten sofort, welcher Treffer architektonisch relevant ist und welcher in totem/isoliertem Code liegt.
  5. **`graph_repo_map`**:
     - *Parameter:* `max_dirs` (int, Standard 16).
     - *Funktion:* Token-budgetierte Repo-Orientierung für neue Sitzungen: Verzeichnis-Cluster, Hubs je Verzeichnis und globale Hotspots aus dem Graphen.
  6. **`graph_check_freshness`**:
     - *Parameter:* keine.
     - *Funktion:* Schneller Abweichungsbericht (Drift) zwischen lokalem Arbeitsbaum und generiertem Graphen.

### 4.2 Server-Profile statt Request-Filterung
Das offizielle Go-MCP-SDK bietet keine dynamischen Filter im `tools/list`-Request. Daher instanziiert `loomux serve` zwei Server-Instanzen mit geteiltem Datenbestand:
1. **Profil `local`**: Beinhaltet alle internen Tools sowie lokale/vertrauliche Wiki-Bereiche.
2. **Profil `cloud`**: Filtert Werkzeuge und maskiert Pfade, die laut `internal/brain/privacy` als `local_only` deklariert sind. Quellcode-Spans aus geschützten Bereichen werden von `graph_find_code` und `graph_file_api` ausgeblendet.

### 4.3 Kaskadierung nachgelagerter MCP-Server (Upstreams)
`loomux serve` kann externe MCP-Server (z. B. Sprachserver oder dedizierte Prüftools) als Upstream-Client anbinden:
- Werkzeuge werden mit Namensraum-Präfix registriert.
- `tools/list_changed`-Events von Upstreams lösen ein Neuregistrieren und Weiterleiten an den Host aus.
- Timeouts und Context-Cancellation werden strikt von oben nach unten durchgereicht.
- `isError: true` auf Tool-Ebene wird transparent durchgereicht und führt nie zum Absturz des Gateway-RPCs.

---

## 5. Kern-Algorithmen

### 5.1 Personalized PageRank (GraphRank)
Implementiert in `internal/graph/pagerank`:
- **Eingabe:** Adjazenzmatrix $A$ des ungerichteten Aufrufgraphen, Knotengrade $d_j = \sum_i A_{ij}$, Seed-Vektor $s$ aus lexikalischen Treffern (BM25 / Substring-Score auf Symbolnamen und Dokumentations-Tokens).
- **Übergangsmatrix:** Spalten-gradnormierte stochastische Matrix $P$ mit $P_{ij} = \frac{A_{ij}}{d_j}$ für $d_j > 0$.
- **Iteration:** Power-Iteration mit Teleportations-Wahrscheinlichkeit $\alpha = 0.25$ und $N = 25$ Schritten:
  $$p^{(k+1)} = \alpha \cdot s + (1 - \alpha) \cdot P \cdot p^{(k)} + \frac{(1 - \alpha) \cdot m_{\text{dangling}}}{|S|} \cdot s$$
  wobei $m_{\text{dangling}} = \sum_{j: d_j=0} p_j^{(k)}$ die Masse isolierter Knoten sammelt und gepoolt an die Seed-Menge $S$ zurückführt. Damit bleibt die Gesamtwahrscheinlichkeitsmasse exakt 1.
- **Deterministische Tie-Ordnung:** Bei identischem Score entscheidet die alphabetische Sortierung nach Symbol-ID (`path:span#name`).
- **Laufzeit:** < 1 ms für typische Repositories (1.000–10.000 Symbole) in Go.

### 5.2 Blast-Radius & Aufruf-Traversierung
Implementiert in `internal/graph/blast`:
- Gerichtete Kanten: `calls`, `imports`, `extends`, `implements`, `references`.
- `DirectionIn` (wer hängt von mir ab?) und `DirectionOut` (wovon hänge ich ab?).
- Transitive Hülle via BFS mit Tiefenlimit (`depth: 1` für direkte Aufrufer, `depth: "all"` für vollständige Abhängigkeitskaskade).
- Ermittelt die exakte Menge betroffener Quelldateien vor Refactorings.

### 5.3 Frische-Prüfung & Speicherung
- **Ablageort:** Der Graph ist Maschinenzustand und liegt ausschließlich in **`.loomux/state/graph/`** (git-ignoriert):
  - `wiring.json`: Gesamter Symbol- und Kantengraph.
  - `cards/<path_hash>.json`: Lazy-geladene Pro-Datei-Karten.
- **Konfiguration:** Vom Menschen gepflegt in `.loomux/config.toml` unter `[graph]` (`languages`, `exclude`, `max_depth`).
- **Hook-Optimierung:** Der `post-tool-use`-Hook prüft **nicht** das gesamte Dateisystem per `os.Stat`. Er liest den Pfad der editierten Datei direkt aus der Hook-Nutzlast auf `stdin` und aktualisiert nur den Hash dieser spezifischen Datei.
- **CI-Verhalten:** In CI-Pipelines ohne gecachten Graphen baut ein vorgelagerter Schritt `loomux graph build` den Graphen auf; existiert kein Graph, meldet `loomux graph check` den fehlenden Index sauber als nicht-initialisiert.

---

## 6. AST- und Sprach-Strategie

### 6.1 Go nativ (Phase 1)
- Extraktion via Standardbibliothek: `go/parser` und `go/token`.
- Extrem schnell (< 10 ms für das gesamte Loomux-Repo), 0 externe Abhängigkeiten.
- Liefert Funktionen, Methoden, Interfaces, Structs und paketinterne Aufrufe.
- Dateiübergreifende Type-Resolution via `go/types` läuft ausschließlich im Offline-Build (`loomux graph build`), **niemals im Hook-Pfad**.

### 6.2 Multi-Language via WebAssembly Tree-sitter (Phase 2)
Um CGo (`gcc`/MSVC-Zwang unter Windows) und native Shared Libraries vollständig zu vermeiden:
- Laufzeit: **`wazero`** (reiner Go-WebAssembly-Interpreter und AOT-Compiler).
- Sprachmodule: Vorkompilierte `tree-sitter-<lang>.wasm`-Module, gebaut mit `wasi-sdk` als `wasm32-wasi`.
- **AOT-Cache:** Kompilierte WASM-Module werden unter `%LOCALAPPDATA%\loomux\cache\wazero\` gecacht (vermeidet 50–500 ms Neukompilierung beim Start).
- Kernsprachen in Phase 2: TypeScript/JavaScript, Python, Rust.

---

## 7. CLI-Schnittstelle (`loomux graph ...`)

Alle Fähigkeiten des Code-Graphen sind nicht nur über MCP, sondern auch direkt als Go-Kommandozeilenwerkzeuge unter `loomux graph` verfügbar — für Menschen, Skripte, CI-Pipelines und die Loomux-Prüfkette:

| Befehl | Flags / Argumente | Beschreibung |
|---|---|---|
| `loomux graph build` | `[dir]`, `--no-reuse`, `--languages <exts>` | Analysiert den Quelltext und schreibt den Graphen nach `.loomux/state/graph/wiring.json`. |
| `loomux graph ask` | `"<anfrage>"`, `--limit <n>`, `--full`, `--in <pfad>`, `--json` | Führt PageRank-Retrieval aus der Konsole aus. Gibt Treffer mit Datei, Zeilenspan und Crux-Exzerpt aus. |
| `loomux graph callers` | `<symbol>`, `--direction in\|out`, `-d <tiefe>`, `--in <pfad>` | Ermittelt Aufrufer (`in`), Aufgerufene (`out`) oder die transitive Aufrufhülle eines Symbols. |
| `loomux graph blast` | `[dir]`, `--base <ref>`, `--format text\|markdown\|json` | Berechnet den Blast Radius eines Git-Diffs (Working Tree oder gegenüber z. B. `origin/main`). Zeigt alle betroffenen Aufrufer und Dateien. |
| `loomux graph grep` | `"<muster>"`, `-i`, `--fixed`, `--in <pfad>`, `--max-hits <n>` | Regex-Suche über alle indizierten Dateien, gruppiert nach Symbol und sortiert nach Kanten-Kopplung (`inDegree`). |
| `loomux graph skeleton`| `<datei>` | Gibt alle Signaturen, Typen und Zeilenspans einer Datei aus ($0, kein LLM). |
| `loomux graph map` | `--max-dirs <n>` | Gibt die token-budgetierte Repo-Karte (Verzeichnis-Cluster, Hubs, Hotspots) im Terminal aus. |
| `loomux graph check` | `--json` | Prüft die Frische des Graphen gegenüber dem Arbeitsbaum. Exitet 0 wenn frisch, 1 wenn veraltet/Drift vorhanden. |
| `loomux graph stats` | `--json` | Gibt Metriken aus: Anzahl Knoten, Kanten, Verzeichnisse, unterstützte Sprachen, Speichergröße. |
| `loomux graph viz` | `--port <p>`, `--no-open`, `--export <verzeichnis>` | Startet den interaktiven Graph-Viewer im Browser oder exportiert eine autarke HTML/SVG-Visualisierung (integriert in `loomux/web`). |

### 7.1 Integration in die Loomux-Prüfkette (`[verify]`)
Zwei CLI-Befehle werden direkt als Lanes in `loomux check` (Pre-Commit und Pre-Push) verfügbar gemacht:
1. **Lane `graph-freshness`**: Führt `loomux graph check` aus. Verhindert Commits mit veraltetem Code-Graphen.
2. **Lane `blast-audit`**: Führt `loomux graph blast --base HEAD~1` aus. Erkennt, ob Änderungen an hochgradig vernetzten Symbolen vorgenommen wurden, für die keine Tests ausgeführt wurden.

### 7.2 Was aus Graft entfällt (Begründung)
- `graft brain connect/push/pull/status`: Entfällt ersatzlos. Loomux besitzt sein Second Brain lokal (`internal/brain`), keine Cloud-Anbindung nötig.
- `graft telemetry`: Entfällt ersatzlos. Loomux ist 100 % lokal und sendet keine Telemetrie.
- `graft upgrade`: Entfällt. Loomux ist ein einzelnes Go-Binary, das über Standard-Verteilungswege (z. B. Pre-Commit Pilot-Build oder `go install`) aktualisiert wird.

---

## 8. Hook-Integration & Verhalten

| Hook | Aufgabe | Budget | Fehlerverhalten |
|---|---|---|---|
| `post-tool-use` | Prüft editierte Datei aus `stdin`, aktualisiert deren Kanten, loggt Blast-Radius (betroffene Aufrufer) | < 100 ms Eigenzeit (~40–50 ms warm inkl. Startboden) | **Rein informativ**: Index fehlt oder veraltet ⇒ Hook schweigt, blockiert niemals einen Edit. |
| `stop` | Warnt, falls Dateien mit hohem Blast-Radius geändert wurden, deren Tests nicht ausgeführt wurden | Teil der Prüfkette | Verhält sich nach Exit-1/Exit-2-Disziplin der Prüfkette. |

---

## 9. Tests und Qualitätstore

1. **100 % Coverage je Funktion**:
   - `internal/graph/*` und `internal/serve/graph/*` unterliegen der 100 %-Pflicht.
   - WASM-Traps und Plattform-Kleber in `extract/wasm` erhalten begründete `//coverage:exempt <grund>`-Zeilen.
2. **CGo-Freiheitstor**:
   - Das Pre-Commit-Tor baut mit:
     ```bash
     CGO_ENABLED=0 go build ./cmd/loomux
     ```
3. **Mutationstests**:
   - Mutationsläufe mit `loomux dev mutants internal/graph/pagerank` und `internal/graph/blast`. Überlebende Mutanten werden in der Paritätsdatei dokumentiert.
4. **Deterministische Golden-Files**:
   - Testfälle für PageRank und Blast-Radius vergleichen Ausgaben gegen feste Golden-Files (`testdata/cases/graph/...`).

---

## 10. Stufenplan (Nach Abschluss der Fusion)

Diese Reihenfolge galt ursprünglich „nach Stufe 4 der Fusions-Spec"
(`2026-09-14-loomux-fusion-design.md`). **So ist es nicht gekommen, und das war
Absicht:** G1 und G2a wurden vorgezogen und parallel zu den Fusions-Stufen 1b-1
und 1b-2 gebaut. Der Grund steht in §1 des G1-Deltas — die Begründung für die
alte Reihenfolge (§11: Inittrace mit dem MCP-SDK) bindet G1 und G2a nicht, weil
beide keine Abhängigkeit einziehen; die Messung gehört zu G3, wo
`internal/serve` das SDK tatsächlich holt.

| Stufe | Stand | Inhalt |
|---|---|---|
| **G1** | ✅ 2026-09-17 | Paket `internal/graph/model`, `pagerank`, `blast` in Pure Go. 100 % Unit-Tests mit deterministischer Tie-Ordnung. Inittrace-Messung mit `go-sdk`. |
| **G2** | 🔶 G2a ✅ 2026-09-18, G2b offen | Nativer Go-Extraktor (`extract/golang`), Frische-Check (`freshness`), Speicherung unter `.loomux/state/graph/`. CLI-Befehle `loomux graph build` und `loomux graph check`. |
| **G3** | offen | MCP-Integration: Verschachteltes Gateway in `internal/serve`, Handler `internal/serve/graph/`, Kanaltrennung mit Privacy-Schutz. Bereitstellung der 6 MCP-Tools: `graph_find_code`, `graph_file_api`, `graph_trace_calls`, `graph_find_all`, `graph_repo_map`, `graph_check_freshness`. |
| **G4** | offen | Vollständige CLI-Palette: `loomux graph ask`, `callers`, `blast`, `grep`, `skeleton`, `map`, `stats`. Post-Edit-Hook-Anbindung (informativer Blast-Radius). Verknüpfung von Code-Symbolen mit Second-Brain-Seiten (`internal/brain/wiki`). |
| **G5** | offen | Multi-Language-Support: WASM-Tree-sitter via `wazero` für TypeScript und Python. Graph-Visualisierung `loomux graph viz` angebunden an `loomux/web`. |

---

## 11. Offene Messpunkte vor dem Bau

- **Inittrace mit `go-sdk` und `wazero`**: Sobald Stufe 1b das Go-MCP-SDK einzieht, misst `GODEBUG=inittrace=1 loomux --version` die Auswirkungen auf den Startboden (32 ms).
- **WASI-Kompilierung von Tree-sitter**: Verifikation, dass `tree-sitter-typescript` unter `wasi-sdk` ohne JS-Host-Imports als sauberes Standalone-WASM läuft.
- **Speicherbedarf von `wiring.json`**: Vermessung der Graphgröße auf echten Repositories und Validierung des Pro-Datei-Karten-Ansatzes.
