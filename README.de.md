# loomux

🌐 **[English](README.md)** | **[Deutsch](README.de.md)** · 📖 **[Docs (EN)](docs/en/)** | **[Handbuch (DE)](docs/de/)**

**Die vereinte autonome Entwickler-Plattform in einem einzigen Go-Binary: Hooks, Skills, Code-Graph, Second Brain & LLM OS.**

Loomux gibt KI-Coding-Agenten (Claude Code, Antigravity, Cursor, Codex) tiefes Codebase-Verständnis, deterministisches Graph-Retrieval, undurchdringliche Schreibschranken und automatisierte Prüfketten — vollständig autark und ohne externe Laufzeit-Abhängigkeiten.

- **Kein Python. Kein Node.js.** Ein einziges, in sich geschlossenes Go-Binary (`loomux.exe` / `loomux`).
- **Kaltstart unter 35 ms.** Federleichte Ausführung, die sich strikt in die Latenz-Budgets von Agenten-Toolcalls einfügt.
- **100 % Test-Coverage je Funktion.** Kompromisslose Qualität mit Mutationstests und null ungetesteten Codepfaden.
- **Windows-First, POSIX-nativ.** Native Betriebssystem-Primitive (Job Objects, Junctions) mit voller Linux/macOS-Unterstützung.

> **Woher kommt der Name?**  
> **Loomux** verbindet den **Loom** (den Webstuhl, der die Fäden aus Agenten-Loops, Code-Graphen und Team-Wissen zu einem dichten, nahtlosen Gewebe verwebt) mit der Endung **-ux** (inspiriert von der Unix-Philosophie robuster, modularer Betriebssystem-Werkzeuge sowie einer kompromisslosen Developer & Agent Experience).

---

## Die 5 Säulen von Loomux

```mermaid
flowchart TD
    subgraph Core["loomux (Einziges Go-Binary)"]
        P1["1. Hooks & Wächter<br/><b>Policy & Schreibschranke</b>"]
        P2["2. Skills & Review<br/><b>Best-Practice-Suiten</b>"]
        P3["3. Graph & Loop<br/><b>AST, PageRank, Blast Radius</b>"]
        P4["4. Second Brain<br/><b>Wiki, ADRs, Semantisches QMD</b>"]
        P5["5. LLM OS & UI<br/><b>Eingebettetes Web-Dashboard</b>"]
    end

    Agents["Coding-Agenten<br/>(Claude Code / Antigravity / Cursor)"] <--> |Hooks| P1
    Agents <--> |MCP / Prompts| P2
    Agents <--> |MCP Tools| P3
    Agents <--> |MCP Tools| P4
    Human["Entwickler / Team"] <--> |Browser / localhost| P5
```

---

## Kern-Workflows

### 1. Der autonome Wächter- & Prüfkreislauf

Jede Interaktion eines Coding-Agenten wird in Echtzeit überwacht und validiert, ohne den Entwicklungsfluss zu blockieren.

```mermaid
sequenceDiagram
    autonumber
    actor Agent as Coding-Agent
    participant Hook as loomux hook
    participant Policy as Policy & Schranke
    participant Graph as Code-Graph (State)
    participant Verify as Prüfkette

    Agent->>Hook: PreToolUse (Tool-Call auf stdin)
    Hook->>Policy: Pfade, Befehle & Schreibschranke prüfen (<35ms)
    alt Verboten
        Policy-->>Agent: Exit 2 (Abgelehnt mit exakter Zeilenbegründung)
    else Erlaubt
        Hook-->>Agent: Exit 0 (Ausführung gestattet)
    end

    Agent->>Agent: Führt Datei-Änderung / Befehl aus

    Agent->>Hook: PostToolUse (stdin)
    Hook->>Graph: Geänderte Datei hashen & Blast Radius berechnen (Ziel <5ms, G4)
    Hook-->>Agent: Betroffene Aufrufer & Blast-Warnungen inline ausgeben

    Agent->>Hook: Stop (Rundenende, Stufe 2c)
    Hook->>Verify: Profil stop über neuen Inhalt fahren (Lanes, Tests, Coverage-Tor)
    Verify-->>Agent: Grün (Exit 0), Halt mit Feedback (Exit 2) oder kein Urteil (Exit 1)
```

### 2. Deterministisches Code-Graph-Retrieval ("GraphRank")

Agenten erkunden Codebasen oft bei jeder Sitzung mühsam von Neuem und verbrennen dabei Zeit und Token. Loomux baut einmalig einen lokalen, deterministischen AST-Code-Graphen auf und beantwortet Abfragen daraus via **Personalized PageRank**.

> **Stand (Stufe G4a).** Stufe G2b hat den Abfragepfad vollendet: `loomux graph ask` sucht Code-Symbole gerankt nach BM25-artigem lexikalischen Matching verschmolzen mit Personalized PageRank (alpha=0.25). Das Retrieval benötigt ~48 ms warm (~38 ms bei Namens-Matching ohne die 1-MB-Rumpfbeiakte auf diesem ~3.000-Knoten-Repo; die Beiakte dient der Skalierung auf 30.000+ Knoten). Quelltext-Spans werden bei Bedarf via `--source` inline eingeblendet. Bei Abweichung wird der Graph automatisch im Hintergrund neu gebaut, sofern `--no-refresh` fehlt; einen ersten Graphen baut keine Abfrage. Stufe G3 stellt die Abfrage und die Driftprüfung als `graph_find_code` und `graph_check_freshness` über MCP bereit (siehe §3). Stufe G4a liefert die vollständige Graph-Navigationspalette (`callers`, `skeleton`, `grep`, `map`, `stats`) und vier neue MCP-Werkzeuge (`graph_file_api`, `graph_trace_calls`, `graph_find_all`, `graph_repo_map`). Stufe G4b ergänzt Git-Diff-Blast-Radius-Analyse und den Post-Edit-Blast-Monitor.

```mermaid
flowchart LR
    Q["Anfrage / Task"] --> Lex["Lexikalischer Treffer<br/>(Tokens / Symbole)"]
    Lex --> |Seeds| PR["Personalized PageRank<br/>(Power-Iteration, alpha=0.25)"]
    Graph[".loomux/state/graph/<br/>AST Wiring Graph"] --> PR
    PR --> Ranked["Gerankte Symbole<br/>(Strukturelle Hubs oben)"]
    Ranked --> Crux["Crux-Inliner<br/>(5-10 Zeilen Kernlogik, $0)"]
    Crux --> Context["Injektierter Kontext<br/>(Volle Antwort ohne Dateilesen)"]
```

> **„Lexik schlägt vor, der Graph entscheidet“**: Keywords finden potenzielle Kandidaten; der strukturelle Aufrufgraph konzentriert die Masse auf die tatsächlich relevanten Kernkomponenten und filtert isolierten oder toten Code heraus.

### 3. Verschachteltes MCP-Composite-Gateway

`loomux serve` fungiert als modularer **MCP-Gateway-Router** über Streamable HTTP und stellt saubere Namensräume für Agenten bereit:

```mermaid
flowchart TD
    Host["Agenten-Host (Claude / Antigravity / Cursor)"] <--> |stdio| Bridge["loomux mcp"]
    Bridge <--> |localhost HTTP| Root["loomux serve (Root MCP Gateway)"]
    
    subgraph Namespaces["Sub-Server Module"]
        Root <--> Brain["brain_*<br/>(search, catalog, read, neighbors, status)"]
        Root <--> Graph["graph_*<br/>(find_code, check_freshness, file_api,<br/>trace_calls, find_all, repo_map)"]
        Root <--> Upstreams["Upstream Proxies<br/>(Sprachserver, qmd mcp)"]
    end
```

**Was heute steht (Stufen 1b-2, G3 und G4a):** der Wirt, die Brücke und die Wurzel über zwei
Loopback-Listener — einer je Kanal, jeder mit eigenem Token — und elf Werkzeuge:
die fünf `brain_*`-Werkzeuge und, seit den Stufen G3 und G4a, sechs `graph_*`-Werkzeuge
(`graph_find_code`, `graph_check_freshness`, `graph_file_api`, `graph_trace_calls`,
`graph_find_all`, `graph_repo_map`). Die Upstream-Proxies sind spezifiziert, nicht gebaut.
`loomux mcp` fällt auf `--channel local` zurück, startet und ersetzt den Dienst selbst,
und der Pro-Edit-Hook-Pfad verlinkt nichts davon, was ein Test über den Importgraphen
festhält. Siehe [`docs/de/cli-reference.md`](docs/de/cli-reference.md) §8.

---

## Migrationsplan

Wo jede Stufe und jede Funktion steht — Herkunft, Stand, Abhängigkeiten und
Priorität —, steht im **[Migrationsplan](docs/de/migration.md)**. Stufe 2c
(das Stop-Tor und die Subagenten-Hooks) ist für Claude Code und Antigravity
fertig. Stufe 3a (`reindex`, `embed`, `reconcile`,
`area add`) ist fertig; diese Maschine fährt sie über ihre eigene Registry.
Stufe 3b (`cases`, `case`, `approve`) ist fertig, samt ihrer Selbstnutzung gegen die
echte Registry. Stufe 3c (`brain check`, `lint --scope`, `wiki init|types|retype`,
die tägliche Aufholung in `serve`) ist fertig; ihre lesenden Befehle sind gegen die
echte Registry gelaufen.
Stufe G4a (`callers`, `skeleton`, `grep`, `map`, `stats` und 4 MCP-Werkzeuge) ist fertig.

---

## CLI-Referenz

Aktive Befehle nach den Stufen 1a, 1b-1, 1b-2, 2a, 2b, 2c, 3a, 3b und 3c im Vergleich zu spezifizierten Befehlen der Folge- und Graph-Stufen:

### Aktive Befehle (Stufen 1a, 1b-1, 1b-2, 2a, 2b, 2c, 3a, 3b und 3c)
```bash
loomux check <profil|arten>         # Fährt die [verify]-Lanes: edit, precommit, all oder lint,types,... (--root, --show, -v)
loomux check gocover --profile <p>  # 100 % je Funktion, oder eine Gesamtgrenze mit --floor N
loomux check commit-msg <datei>     # Prüft eine Commit-Nachricht: Kopf nach Conventional Commits und Sprache ([commit], --language, --calibrate N)
loomux check gofmt [pfade...]       # Prüft Go-Formatierung ohne Dateiänderungen
loomux hook pre-tool-use            # Prüft Policy und globale Schreibschranke gegen stdin
loomux hook post-tool-use           # Fährt die Lanes des Profils edit gegen die eben geänderte Datei (--budget, Vorgabe 50s)
loomux hook session-start           # Hält den Basis-Commit der Sitzung fest und warnt vor veraltetem Binary
loomux hook stop                    # Tor am Rundenende: Profil stop über neuen Inhalt, Befunde der Subagenten (--budget, Vorgabe 270s)
loomux hook subagent-start|subagent-stop  # Schnappschuss von origin, Branches und HEAD um einen Subagenten; parkt, was sich bewegt hat, für stop
loomux status|doctor|explain        # Zeigt Hook-Status, Prüfketten und erkannte Host-Harnesses (drei Namen, ein Codeweg)
loomux worktree link|unlink|remove  # Verwaltet isolierte Arbeitsbaum-Spiegel und Junction-Pfade
loomux dev swap-binary              # Tauscht laufendes Binary atomar gegen Neubau aus
loomux version                      # Gibt die Version dieses Binaries aus
loomux lint <datei>                 # Prüft Links und Frontmatter einer Wiki-Seite
loomux lint [--scope all|S]         # Lintet jedes registrierte Bündel nach den zwölf Regeln der Referenz
loomux brain check file|bundle|all  # Die Regeln für OKF, Haus und Föderation über eine Seite, ein Bündel oder alle Bereiche (--notes)
loomux wiki init --scope S          # Legt das Gerüst des Wiki-Bündels eines Bereichs an
loomux wiki types                   # Zählt die Seitentypen über alle Bereiche, mit Rang und Altnamen
loomux wiki retype --scope S --from A --to B  # Benennt einen Seitentyp in einem Bündel um
loomux wiki-gate                    # Erzwingt Frische und strukturelle Schranken des Wikis
loomux brain search "<anfrage>"     # Durchsucht die sichtbaren Bereiche über den qmd-Daemon (--profile fast|full|keyword)
loomux brain catalog [--scope S]    # Wurzelkatalog der sichtbaren Bereiche oder das index.md eines Bereichs
loomux brain read <pfad> --scope S  # Eine Datei eines Bereichs oder einen Abschnitt daraus (--section)
loomux brain neighbors <pfad> --scope S  # Eingehende und ausgehende Links einer Seite
loomux brain status                 # Was man wissen muss, bevor man einer Antwort traut
loomux serve [--foreground]         # Startet den langlebigen localhost-MCP-Dienst, abgekoppelt oder hier; holt einen fälligen reconcile täglich nach
loomux serve status                 # Was serve.json sagt und ob der Listener antwortet
loomux serve stop [--force]         # Beendet den Dienst über seinen Endpunkt oder über seine PID
loomux mcp [--channel local|cloud]  # stdio-Brücke, die ein MCP-Wirt startet; sie startet den Dienst selbst
loomux reindex [--registry P]       # Erst abgleichen, dann Kataloge, Linkgraph, Identitätsregister und qmd-Sammlungen jedes Bereichs neu bauen
loomux embed [--registry P]         # Erzeugt die Vektoren, die reindex offen lässt (braucht qmd auf dem PATH)
loomux reconcile                    # Eröffnet Prüffälle für geänderte Quellen und gelandete Merges; ein Fall ist kein Fehlschlag
loomux area add [--path P] [--scope S]  # Meldet ein Repository als Bereich an, legt sein Wiki an und indiziert es (--wiki, --sources, --merge-branch, --privacy, --no-reindex)
loomux cases                        # Listet die Fälle, die im Prüfzentrum warten; ein Fall ist kein Fehlschlag
loomux case <id> [--package]        # Zeigt einen Fall mit Paket und Vorschlag; bei local_only zurückgehalten bis --package
loomux approve <id>                 # Entscheidet einen Fall: wendet den belegten Vorschlag an und committet ihn (--amend D, --reject, --defer)
```

### Implementierte Befehle (Code-Graph — Stufen G2a–G4a)
```bash
loomux graph build [--root <pfad>]  # Extrahiert, löst auf und schreibt .loomux/state/graph/wiring.json
loomux graph check [--root <pfad>]  # Extrahiert neu und vergleicht mit Graph auf Platte (Exit 1 bei Drift)
loomux graph ask "<anfrage>" [flags] # Sucht Symbole gerankt nach lexikalischem Score und Personalized PageRank; baut nie einen ersten Graphen
loomux graph callers <symbol>       # Zeigt Aufrufer, Aufgerufene (--direction out) oder transitive Hülle (-d all)
loomux graph skeleton <datei>       # Gibt Signaturen und Zeilenspans einer Datei aus (~10x Token-Ersparnis)
loomux graph grep "<regex>"         # Regex-Suche gruppiert nach Symbol und sortiert nach Kopplung
loomux graph map                    # Gibt token-budgetierte Verzeichnis-Cluster, Hubs und Hotspots aus
loomux graph stats                  # Gibt Graph-Kennzahlen aus (Knoten, Kanten je Relation, Dateien, Sprachen, Größe)
```

### Spezifizierte Befehle (Code-Graph — Stufen G4b–G5)

Stufe G4a hat Navigation und Retrieval geliefert (`callers`, `skeleton`, `grep`, `map`, `stats`); Stufe G4b ergänzt `blast` und den Post-Tool-Blast-Monitor; Stufe G5 ergänzt mehrsprachige Extraktion über `wazero`.
```bash
loomux graph blast [dir]            # Berechnet den Blast-Radius eines Git-Diffs gegen Working Tree oder Merge-Base
loomux graph viz                    # Öffnet den interaktiven Graph-Viewer im Browser
```

### Spezifizierte Befehle (Second Brain & Dienste — Stufen 4 & W1–W5)
```bash
loomux serve                        # Das eingebettete Web OS neben den MCP-Listenern
loomux init [--detect-only]         # Richtet Hooks, Einstellungen, Skills und AGENTS.md in erkannten Agenten ein
```

### Entwickler- & Worktree-Werkzeuge
```bash
loomux dev bench-hooks <fall>       # Misst die Latenz der Hook-Ausführung gegen die Grundlinie von < 35 ms
loomux dev bench [--dir <dir>] [--save] # Benchmark für Einzel-Repo oder Open-Source-Matrix-Korpus mit Lücken-Audit; --save sichert in docs/
loomux dev mutants <paket>          # Führt Mutationstests über kritische Entscheidungspakete aus
loomux dev record-case --out <dir>  # Zeichnet einen Lauf eines Referenz-Binaries als Fall auf
loomux dev import-cases --map <f>   # Übersetzt ein Verzeichnis aufgezeichneter Fälle in loomux-Fälle
loomux dev record-mcp-case --out <dir> # Zeichnet einen MCP-Werkzeugaufruf eines Referenzdienstes als Fall auf
loomux dev release <unterbefehl>    # Release-Regeln für die CI: next-version, parse-body, changelog-insert, build
```

---

## Architekturentscheidungen: Was wir bauen, ablösen und weglassen

| Komponente / Idee | Herkunft / Inspiration | Entscheidung in Loomux | Begründung |
|---|---|---|---|
| **Einziges Go-Binary** | Grundarchitektur | ✅ **Kernmandat** | 0 Python, 0 Node.js. 7,5 ms warmer Hook, autarke Auslieferung, 100 % Testabdeckung. |
| **AST-Code-Graph & PageRank** | `trailhq/Graft` | ✅ **Nativ übernommen** | $0 deterministischer Code-Graph. Personalized PageRank filtert strukturelle Kern-Hubs statt naiver Keyword-Listen. |
| **Blast Radius & Crux-Inlining** | `trailhq/Graft` | ✅ **Nativ übernommen** | Auswirkungsanalyse bei Edits (Ziel < 5 ms); liefert 5-10 Zeilen Kernlogik oder Spans ($0 Token-Lesekosten, ~48 ms warmes Retrieval). |
| **Symbol-gekoppelter Grep** | `trailhq/Graft` | ✅ **Nativ übernommen** | Regex-Treffer gruppiert nach umschließendem Symbol und gerankt nach Kanten-Kopplung (`inDegree`). |
| **Lokales Second Brain & Wiki** | Grundarchitektur | ✅ **Kernmandat** | Markdown-Wiki, ADRs und Identitätsregister direkt im Repo. Code-Symbole verlinken direkt auf Architektur-Entscheidungen. |
| **Node.js & C++ Toolchain** | `trailhq/Graft` | ❌ **Abgelehnt** | Graft setzt Node.js >=20, `node-gyp` und MSVC voraus. Loomux bleibt 100 % Pure Go ohne C-Compiler-Zwang. |
| **Cloud-Brain-Synchronisation** | `trailhq/Graft` | ❌ **Abgelehnt** | Graft synchronisiert Symbol-Hashes mit Cloud-APIs. Loomux hält alles Wissen, alle Regeln und Graphen 100 % lokal und offline. |
| **Telemetrie & Tracking** | `trailhq/Graft` | ❌ **Abgelehnt** | Graft sendet Nutzungsstatistiken an externe Server. Loomux hat null Telemetrie und telefoniert niemals nach Hause. |

---

## Architektur-Prinzipien

1. **Deterministisch als Standard**: Code-Graph, Blast-Radius und Schreibschranken laufen lokal, deterministisch und kosten $0.
2. **Startzeit-Disziplin**: `loomux` misst seinen Startboden (5,5 ms warm, 2026-09-17) kontinuierlich. Keine Paketvariable und kein `init()` darf eingebettete Daten parsen oder I/O durchführen.
3. **Strikte Isolierung**: Hook-Pfade laufen im Prozess und hängen niemals von einem laufenden `serve`-Daemon ab.
4. **Agentensichere Konfiguration**: `.loomux/config.toml` deklariert Schutzbereiche und Policies; sie wird vom Menschen gepflegt und ist für Agenten schreibgeschützt — für Schreibwerkzeuge wie für Shell-Befehle (`>`, `sed -i`, `tee`, `Set-Content`, `cp`/`mv` darauf). Maschinenzustand liegt in `.loomux/state/` (git-ignoriert).

---

## Dokumentations-Suite

Vollständige Handbücher und technische Leitfäden sind unter [`docs/de/`](docs/de/) gegliedert:

| Handbuch | Beschreibung |
|---|---|
| 🚀 **[Erste Schritte](docs/de/getting-started.md)** | Installation, 3-Minuten-Schnellstart und Anbindung an Agenten-Harnesses (Claude Code, Antigravity, Cursor). |
| 🏛️ **[Architektur & Konzepte](docs/de/architecture.md)** | Das theoretische Fundament: Andrej Karpathys LLM OS, Googles Knowledge Items (KI), Grafts AST-GraphRank und der Schreibschranken-Kernel. |
| ⚙️ **[Konfigurations-Referenz](docs/de/configuration.md)** | Vollständige Referenz für `.loomux/config.toml` (`[verify]`, `[policy]`, `[worktree]`, `[graph]`, `[skills]`, `[privacy]`). |
| 📖 **[CLI-Referenzhandbuch](docs/de/cli-reference.md)** | Detailliertes Handbuch aller Befehle, Flags, stdin-JSON-Nutzlasten und Exit-Codes. |
| 🪝 **[Hook-Lebenszyklus & Integration](docs/de/hooks.md)** | Technische Spezifikation des 4-Phasen-Hook-Zyklus, der Host-Formate und des entkoppelten SSE-Ereignisstroms. |
| 🗺️ **[Migrationsplan](docs/de/migration.md)** | Jede Stufe und jede Funktion der Fusion und des Code-Graphen: Herkunft, Stand, Abhängigkeiten und Priorität. |
| ⏱️ **[Leistungs-Benchmarks](docs/de/benchmarks.md)** | Chronologische Messungen gegenüber den Vorläufer-Programmen und verbindliche Latenzbudgets. |
| 📊 **[Benchmark-Matrix](docs/de/benchmarks/matrix.md)** | Open-Source-Matrix über Top-Sprachen hinweg mit Detailberichten pro Sprache und Repository. |

---

## Spezifikationen & Interne Arbeitspapiere

Detailentwürfe und interne Arbeitspapiere liegen unter `docs/.superpowers/specs/`:
- [Fusions-Design: Ultraloom & Ultra-Brain](docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md)
- [Code-Graph Subsystem-Spezifikation](docs/.superpowers/specs/2026-09-14-loomux-code-graph-design.md)
- [Web OS & Skill-System-Spezifikation](docs/.superpowers/specs/2026-09-14-loomux-web-os-design.md)

---

## Releases

Binaries gibt es auf der [Releases-Seite](https://github.com/xidus90/loomux/releases):
`loomux_<version>_<os>_<arch>` für `windows/amd64` (`.exe`), `linux/amd64`,
`linux/arm64`, `darwin/amd64` und `darwin/arm64`. Geprüft wird gegen
`SHA256SUMS` aus demselben Release:

```sh
sha256sum --check --ignore-missing SHA256SUMS
```

Jeder gemergte Pull Request nach `master` wird nach seinem Label veröffentlicht:

| Label | Bedeutung | Version |
|---|---|---|
| `release:major` | Inkompatible Änderung an Befehl, Flag, Hook-Protokoll, Konfigurationsformat oder Exit-Code | `X+1.0.0` |
| `release:minor` | Neue Funktion, kompatibel | `X.Y+1.0` |
| `release:patch` | Fehlerbehebung oder Abhängigkeits-Update, kompatibel | `X.Y.Z+1` |
| `release:none` | Nur Doku, CI oder Tests | kein Release |

Jedes Release ist ein Beta-Pre-Release, bis `RELEASE_CHANNEL` auf `stable`
steht. Was sich geändert hat, steht in [`CHANGELOG.md`](CHANGELOG.md).

### Einen Pull Request öffnen

Auf `master` wird nie direkt committet; jede Änderung läuft über einen Pull
Request. Die Hooks in `.githooks` lehnen einen Commit auf `master` und einen
Push dorthin ab, sobald `git config core.hooksPath .githooks` gesetzt ist. Mit
einem LLM erledigt der Skill `release-pr` die Schritte unten; von Hand:

1. Commits thematisch gruppieren, ein Commit pro Änderung. Eine spätere
   Korrektur an etwas, das dieser Branch eingeführt hat, gehört in den
   Commit, der es eingeführt hat; nur die Behebung eines Fehlers, der schon
   auf `master` war, behält einen eigenen Commit. Einen bereits gepushten
   Branch neu zu schreiben braucht `git push --force-with-lease`. Jede
   Nachricht ist ein Conventional Commit (`feat: …`, `fix: …`, `docs: …`;
   `!` für eine inkompatible Änderung). Der Hook `commit-msg` prüft das
   lokal, und der Check `pr-label` lehnt einen Pull Request ab, sobald ein
   Commit-Titel die Form verletzt.
2. Ein Label aus der Tabelle oben wählen; zwischen zwei Stufen die höhere.
   Es liegt nie unter den Commits: `!` verlangt major, `feat` minor, `fix`
   patch.
3. Den Rumpf schreiben:
   ```
   Release: <level> — <one-sentence reason>

   ## Summary
   - <what changes for a user>

   ## Changelog
   ### Added
   - <entry>
   ```
   Changelog-Überschriften sind nur `Added`, `Changed`, `Deprecated`,
   `Removed`, `Fixed`, `Security`; bei `release:none` entfällt der Abschnitt.
4. Prüfen: `go run ./cmd/loomux dev release parse-body --labels release:<level> --body <file>`.
5. Branch pushen, dann `gh pr create --base master --label release:<level> --body-file <file>`.

### Releases einrichten (Maintainer)

1. Labels:
   ```sh
   gh label create release:major --color B60205 --description "Breaking change"
   gh label create release:minor --color 0E8A16 --description "New feature, compatible"
   gh label create release:patch --color 1D76DB --description "Bug fix or dependency update"
   gh label create release:none --color CCCCCC --description "No release"
   ```
2. Kanal: `gh variable set RELEASE_CHANNEL --body beta`
3. GitHub App (Settings → Developer settings → GitHub Apps → New): Name
   `loomux-release`, Webhook aus, Repository-Rechte `Contents: Read and
   write`, `Pull requests: Read-only`, `Metadata: Read-only`, „Only on this
   account“. Private Key erzeugen, App nur in `xidus90/loomux` installieren,
   dann:
   ```sh
   gh secret set RELEASE_APP_CLIENT_ID --body <client-id>
   gh secret set RELEASE_APP_PRIVATE_KEY < loomux-release.private-key.pem
   ```
4. Rulesets (erst wenn das Repository öffentlich ist): eines für `master`
   und eines für Tags `v*`, jeweils mit der App `loomux-release` als einzigem
   Bypass-Akteur. Das Ruleset für `master` verlangt außerdem die
   Statuschecks `gate-windows` und `build-linux` (Workflow `ci`) sowie
   `check` (Workflow `pr-label`).

Ist ein Release ausgefallen, den Workflow `release` von Hand starten
(`gh workflow run release.yml -f pr=<nummer>`). Er verweigert einen Pull
Request, der nicht nach `master` gemergt ist, und ein erneuter Lauf nach dem
`chore(release): v*`-Commit verwendet diesen Commit wieder. Gibt es für den Pull
Request schon einen Tag oder ein Release, nicht erneut starten, sondern das
vorhandene Release von Hand reparieren.

---

## Lizenz

PolyForm Noncommercial License 1.0 (`LICENSE.md`).
