# Migrationsplan

Diese Datei wird mit jeder Änderung nachgeführt: Ein Pull Request, der
Migrationsarbeit beginnt, abschließt, hinzufügt oder streicht, ändert sie hier und
in [`docs/en/migration.md`](../en/migration.md) — den Stand der Stufe, was noch
offen ist, ihre Abhängigkeiten und ihre Priorität, und den Stand jeder Funktion,
die sie berührt. Eine Entscheidung (eine neue
Stufe, ein Wegfall, eine geänderte Reihenfolge) steht zuerst in der
[Fusions-Spec](../.superpowers/specs/2026-09-14-loomux-fusion-design.md), und dieser Plan folgt ihr.

## Stufen

Loomux setzt einen mehrstufigen Fusionsplan um. Eine zweite Spur — der
Code-Graph — läuft **neben** ihm statt hinter ihm, weil keine seiner fertigen
Stufen eine Abhängigkeit einzieht. *Priorität* ordnet die offenen Zeilen nach drei
Regeln der Reihe nach: was die Abhängigkeiten schon zulassen, was loomux an sich
selbst benutzt, dann Größe. Festgelegt ist sie in der
[Fusions-Spec](../.superpowers/specs/2026-09-14-loomux-fusion-design.md) unter „Reihenfolge der offenen Stufen“.

| Stufe | Stand | Herkunft | Was sie gebracht hat | Hängt ab von | Priorität |
|---|---|---|---|---|---|
| **1a** | ✅ | ultraloom + ultra-brain | Der Pilot: Repo-Gerüst, Tore, der vereinte Wächter, die Post-Edit-Lanes. loomux benutzt sich selbst | — | — |
| **1b-1** | ✅ | ultra-brain | Die lesenden Brain-Befehle — `search`, `status`, `catalog`, `read`, `neighbors` — mit Parität zur Python-Referenz | — | — |
| **1b-2** | ✅ | ultra-brain | `serve` mit MCP über Streamable HTTP und die stdio-Brücke, durch einen aufgezeichneten Fallkorpus an die Referenz gemessen. Upkeep ist Stufe 3c | — | — |
| **1b-3** | ✅ | ultraloom + ultra-brain | Wiki und Dokumentation sind umgezogen | — | — |
| **2a** | ✅ | ultraloom | Die Prüfkette: `[verify]` mit Presets je Stack, `loomux check <profil>`, `check gocover`, post-edit auf `[verify]`, Prozessbäume werden ganz beendet. loomux prüft seine eigenen Commits mit `check precommit` | — | — |
| **2b** | ✅ | ultraloom | commit-msg mit `--language`, `--calibrate` und `[commit]` | — | — |
| **2c** | ✅ | ultraloom | Das Stop-Tor (`loomux hook stop`, Profil `stop`, ein Fingerabdruck des Inhalts, 3 Blockaden in Folge, der Marker `.loomux/no-verify`) und `subagent-start`/`subagent-stop`, deren Befunde das Stop-Tor zustellt; das Wiki-Bündel als Lane `lint/wiki` in `loomux check` und im Tor. Antigravity-Nachmessung erfolgt und Host-Adapter für Claude Code und Antigravity implementiert. Dieses Repository fährt das Tor über seine eingecheckte `.claude/settings.json` | 2a ✅ | — |
| **3a** | gebaut, Selbstnutzung offen | ultra-brain | Erkennen: `loomux reindex` und `loomux embed` (Umzug von `ultra-brain/pkg/index`), `loomux reconcile` samt Lese- und Ablageseite des Ereignisprotokolls, `loomux area add`, der Auffangdurchgang vor `reindex`; geschrieben wird nach `LOOMUX_STATE_DIR`, das Altverzeichnis nur noch gelesen. 28 Fälle gegen die Python-Referenz, die vier Git-Fälle über `gitworld` aus 2c. **Offen** ist Bedingung 5, die Selbstnutzung: sie lief auf Entscheidung des Nutzers nur gegen eine Kopie der Registry; der Umstieg folgt nach dem Merge und braucht vorher ein `[index]` in `.loomux/config.toml` (Auflage S3 in `parity/stufe-3a.md`) | 1b-1 ✅ | 1 |
| **3b** | offen | ultra-brain | Entscheiden: `loomux cases`, `loomux case`, `loomux approve`; die Anwendung eines Vorschlags, die Belegbindung und die Schreibseite von `vcs` | 3a (die Fälle) | 1 |
| **3c** | offen | ultra-brain | Pflegen: `loomux check file\|bundle\|all` mit den Regeln für OKF, Haus und Föderation, `loomux lint --scope all`, die Wiki-Typen (`wiki types\|retype\|census\|scaffold`), Upkeep in `serve` | 3a (Upkeep ruft `reconcile`), 1b-2 ✅ (Upkeep läuft in `serve`) | 1 |
| **4** | offen | ultraloom + ultra-brain | Konvertierung und Abruf, das lokale Modell, `loomux migrate`; `loomux init` mit den Skills, der erzeugten `AGENTS.md`, den Git-Hooks, dem Binary auf dem `PATH` und `--detect-only`; die Umstellung der Wirte | 2b ✅ (`migrate` überträgt die Commit-Sprache nach `[commit]`), 2c ✅ (`init` verdrahtet `stop` und `subagent-*`), 3 zu einem Drittel (3a gebaut, 3b und 3c offen; die Brain-Skills rufen `check` aus 3c); die Umstellung der Wirte braucht zudem einen Remote für `brain-knowledge` | 3 |
| **G1** | ✅ | neu | Rang und Blast-Radius als Bibliotheken, an portierten Testvektoren der Referenz belegt | — | — |
| **G2a** | ✅ | neu | Extraktor, Auflösung, Speicher, Frischesonde sowie `graph build` und `graph check` | — | — |
| **G2b** | ✅ | neu | Die Abfrage: lexikalische Saat, die Beiakte, `loomux graph ask` | — | — |
| **G3** | ✅ | neu | `graph_find_code` und `graph_check_freshness` am Gateway von 1b-2; eine Abfrage baut nie einen ersten Graphen | — | — |
| **G4** | offen | neu | Die übrige `graph`-Palette (`callers`, `blast`, `grep`, `skeleton`, `map`) und der Blast-Monitor in post-edit | G3 ✅ | 2 |
| **G5** | offen | neu | Mehrsprachige Extraktion über `wazero` | G4 | 6 |
| **Flow** | offen | ultraloom | Folgeprojekt: die ulflow-Laufzeit, Journal, Resume und Replay, `verify_until_green` als Daten-Flow | ulflow M1 (Zweig `feature/agent-harness`, nicht gemergt) | 4 |
| **W1 – W5** | offen | ultra-brain + neu | Web-OS: Hülle (W1), die Brain-Web-App (W2), Graph-Visualizer (W3), Skill-Suiten und Review (W4), Flow-Editor und Kanban (W5) | W1 an 1b-2 ✅; W2 an W1; W3 an W1 und G4; W4 an G4 und 4; W5 an W1 und Flow | 5 |

Was die beiden Quellrepos können und bisher keine Stufe übernommen hatte, steht
in der [Fusions-Spec](../.superpowers/specs/2026-09-14-loomux-fusion-design.md) unter „Nachgetragen“, je mit einer
vorgeschlagenen Stufe oder einem vorgeschlagenen Wegfall; die Zuordnung gilt
nach der Freigabe. Freigegeben sind #1 und #2 (Stufe 3c), #3 und #17 (Stufe 3a),
alle am 2026-09-19. #17 — `reindex` und `embed` als Befehle — war bis dahin
keiner Stufe zugeordnet; die Lücke ist seit dem 2026-09-19 geschlossen, gebaut
mit 3a.

Jede Stufe endet grün und wird einzeln übergeben, mit eigenem Plan und — sobald
sie fertig ist — eigener Paritätsakte, die jede ihrer Verfügungen festhält.

## Funktionen

Wo die einzelnen Funktionen stehen:

| Säule / Funktion | Herkunft | Beschreibung | Status |
|---|---|---|---|
| **1. Hooks & Wächter** | | | |
| Einheitlicher Pre-Tool Wächter | ultraloom + ultra-brain | Prüfung von Schreibschranken, Pfadregeln und verbotenen Befehlen (< 35 ms Zielbudget; 32–34 ms gemessen am Vorgänger; ein Write in einem verknüpften Worktree gemessen 34,6 ms warm (2026-09-16)). Verknüpfte Git-Worktrees eines registrierten Workspace sind ohne eigenen Registry-Eintrag beschreibbar. Registry und Bereichsdeklarationen laufen durch dieselben Prüfungen wie die Brain-Befehle; ein kaputter Eintrag verweigert jeden Write. | ✅ **Implementiert** (Stufe 1a) |
| Post-Tool Prüf-Lanes | ultraloom | Das Profil `edit` aus `[verify]` auf der eben geänderten Datei, für ihren Stack und in ihrem Bereich — `go vet` und `gofmt` der einen Datei, der Wiki-Lint im eigenen Prozess, ruff/mypy, eslint/tsc, stylelint und die übrigen, aus denselben Presets, die `loomux check` fährt. Befehle starten als argv, ohne Shell. Eine gescheiterte Lane endet mit 2; Lanes, die wegen eines fehlenden Werkzeugs oder eines aufgebrauchten Budgets (`--budget`, Vorgabe 50 s) ausfallen, werden dem Modell namentlich zurückgemeldet. | ✅ **Implementiert** (Stufe 1a; Lanes aus `[verify]` seit 2a) |
| Post-Tool Blast Monitor | neu | Hashing geänderter Dateien und Warnung bei berührten Aufrufern. Den Wiring-Graphen, den es braucht, schreibt jetzt `loomux graph build`, aber der Edit-Hook liest ihn noch nicht: heute kein Hash und keine Warnung. | 📋 **Spezifiziert** (Stufe G4) |
| Sitzungsstart | ultraloom | Hält den Commit fest, auf dem eine Sitzung beginnt, und warnt, wenn das Binary im Projekt älter ist als `go.mod`, `go.sum` oder eine `.go`-Datei unter `cmd/` oder `internal/`. Kündigt nur an; blockiert nie einen Zug. | ✅ **Implementiert** (Stufe 1a) |
| Subagent-Drift & Stop-Tor | ultraloom | `loomux hook stop` fährt an jedem Rundenende mit neuem Inhalt das Profil `stop` (vorgegeben die vier Arten von `precommit`, in einem Budget von 270 s), hält die Runde bei einer roten Lane mit Exit 2 an und gibt nach 3 Blockaden in Folge auf; ein Rundenende ohne Neues kostet auf diesem Repository 169,5 ms. `subagent-start` und `subagent-stop` halten `origin`, die lokalen Branches und `HEAD` fest und parken, was sich bewegt hat, für das Stop-Tor des Hauptagenten. Host-Adapter für Claude Code und Antigravity implementiert. | ✅ **Implementiert** (Stufe 2c) |
| Prüfbefehle | ultraloom | `loomux check commit-msg` (die Sprache jeder Zeile, der Kopf nach Conventional Commits, `--language`, `--calibrate` und `[commit]`), `check gofmt` (Formatierung, mit dem Exit-Code, den `gofmt -l` nicht gibt) und `check gocover` (100 % je Funktion gegen ein Profil, oder eine Gesamtgrenze mit `--floor`). `dev covergate` gibt es nicht mehr. | ✅ **Implementiert** (Stufe 1a; `gocover` 2a; `commit-msg` 2b) |
| Prüfketten-Tabelle | ultraloom | Eine Tabelle `[verify]` treibt `loomux check <profil>` und den post-edit-Hook: Presets je Stack, die ohne jede Konfiguration gelten, eine Lane je Art, Stack und Bereich, `after`-Kanten statt Stufen, ein Urteil je Art und `--show`, das zeigt, was läuft. Eine Lane kann Dateien nennen, die sie braucht (`needs`): die C++-Lanes auf dem Build-Baum warten auf `build/CMakeCache.txt` und sind `unready`, bis der Build konfiguriert ist; ein Edit führt `clang-format` auf der Datei trotzdem aus. Kindprozessbäume werden bei einer Frist ganz beendet (unter Windows über ein Job Object). | ✅ **Implementiert** (Stufe 2a) |
| Worktree-Spiegelung | ultraloom | Isolierte Subagent-Git-Worktrees mit NTFS-Junctions und Sitzungsverfolgung. | ✅ **Implementiert** (Stufe 1a) |
| Zonenfreier Startpfad | neu | Gos lokale Zeitzone bleibt vom Hook-Pfad fern: Der TOML-Parser baut seine lokalen Zonen beim ersten Gebrauch (`third_party/toml`), und ein Test im Tor lässt jedes Paket-Init über 500 Allokationen scheitern. `hook pre-tool-use` 7,5 ms warm gegen 26,5 ms vorher (gemessen 2026-09-17). | ✅ **Implementiert** (ohne Stufe) |
| Claude-Mods-Adapter | neu | Die Schreibschranke in einen `tool.check`-Function-Hook setzen ([claude-code#91870](https://github.com/anthropics/claude-code/issues/91870)), der über `$.mcp.call` mit einem langlebigen loomux spricht — entfernt den Spawn, bringt ein `ask`-Urteil und eine gerenderte Begründung. Nur für Claude Code; der Exec-Hook bleibt der portable Pfad. | 💡 **Optional** (ohne Stufe) |
| **2. Skills & Best Practices** | | | |
| Kuratierte Sprach-Suiten | neu | Eingebettete Best-Practice-Regeln für Go (Zero-Alloc, Error-Handling, no-init), Python, TS, Rust. | 📋 **Spezifiziert** (Stufe W4) |
| Graph-gestützter Code-Review | neu | Review-Skills, die via `graph_blast` Aufrufer-Auswirkungen prüfen und ADRs abgleichen. | 📋 **Spezifiziert** (Stufe W4) |
| 3-Kanal-Distribution | neu | Konfiguriert via `.loomux/config.toml`, synchronisiert in Host-Ordner, via MCP-Prompts oder Web OS. | 📋 **Spezifiziert** (Stufe W4) |
| **3. Code-Graph & Loop** | | | |
| Nativer Go-AST-Extraktor | neu | Deterministische Symbol- & Kantenextraktion allein via `go/parser` und `go/ast` — kein `go/types`, kein Build ($0, 0 Deps). Hinter `loomux graph build` verdrahtet; braucht auf diesem Repository in der Größenordnung eines Zehntel einer Sekunde, mit Befehl und Rohausgabe gemessen in `docs/de/benchmarks.md`. | ✅ **Implementiert** (Stufe G2a) |
| Personalized PageRank | neu | Power-Iteration Random-Walk-Ranking über Aufruf- und Abhängigkeitsgraphen, ungerichtet über fünf Relationen, max-normiert mit deterministischer Gleichstandsordnung. Verschmolzen mit BM25-Kandidaten-Scoring in `loomux graph ask` (~48 ms warmes Retrieval). | ✅ **Implementiert** (Stufe G2b) |
| Blast-Radius-Engine | neu | Transitive Hülle und Impact-Analyse (`In`/`Out`, Tiefenbegrenzung, kleinste Tiefe gewinnt). Noch fragt kein Befehl etwas. | 🧩 **Bibliothek** (Stufe G1) |
| Symbol-gekoppelter Grep | neu | Regex-Suche, gruppiert nach umschließendem Symbol und gerankt nach Kanten-Grad (`inDegree`). | 📋 **Spezifiziert** (Stufe G4) |
| Multi-Language AST | neu | CGo-freier Tree-sitter über WebAssembly (`wazero`) mit persistentem AOT-Kompilierungs-Cache. | 💡 **Geplant** (Stufe G5) |
| MCP-Dienst & stdio-Brücke | ultra-brain | `loomux serve` hält zwei Loopback-Listener, je einen pro Kanal und jeden mit eigenem Token, und beantwortet sieben Werkzeuge über Streamable HTTP — die fünf `brain_*`-Werkzeuge und, seit Stufe G3, `graph_find_code` und `graph_check_freshness`; `loomux serve status` und `stop [--force]` steuern ihn, und `loomux mcp` ist die stdio-Brücke, die ein Wirt startet und die den Dienst selbst startet und ersetzt. `internal/hooks` bindet nichts davon: ein Tor-Test liest den Importgraphen. Die Front ist durch einen aufgezeichneten Fallkorpus an die MCP-Front der Python-Referenz gemessen; verglichen wird der Text jeder `CallToolResult` und `isError`, nicht der Umschlag, den zwei verschiedene SDKs aushandeln. | ✅ **Implementiert** (Stufen 1b-2, G3) |
| **4. Second Brain & Wiki** | | | |
| Lokales Markdown-Wiki | ultra-brain | Das Bündel selbst liegt in `docs/wiki/` (Bereich `project/loomux`, am 2026-09-16 Seite für Seite umgezogen und zeilenweise freigegeben). `loomux lint <datei>` prüft Links und Frontmatter einer Seite, `loomux wiki-gate` Frische und Struktur des Bündels, und die Lane `lint/wiki` seine Struktur in `loomux check lint` und an jedem Rundenende. Identitätsregister und Themen-Graph schreibt `loomux reindex` aus Stufe 3a, nicht der Umzug; die Seitenregeln von `brain check` (OKF, Haus, Föderation) kommen mit Stufe 3c. | 🚧 **In Migration** (Stufen 3a, 3c) |
| Semantischer QMD-Index | ultra-brain | Einbettung lokaler Vektoren und neuronaler Suche mit Caching in `~/.cache/qmd`. Die Sammlungen trägt `loomux reindex` in qmds `index.yml` ein, die offenen Vektoren erzeugt `loomux embed` (Zeile „Index-Neubau und Einbettung“). | 🚧 **Gebaut, Selbstnutzung offen** (Stufe 3a) |
| Index-Neubau und Einbettung | ultra-brain | `loomux reindex` baut je Bereich Identitätsregister, Kataloge, Linkgraph und die qmd-Sammlungen neu und fährt vorher den Auffangdurchgang (`reconcile`), damit keine Wissensänderung am Prüftor vorbeigeht; `loomux embed` erzeugt über `QmdMcpPort.Embed` die Vektoren, die `reindex` offen lässt. Umzug von `ultra-brain/pkg/index` samt Tests. Bis zum 2026-09-19 war beides keiner Stufe zugeordnet (Nachtrag #17 der Fusions-Spec). | 🚧 **Gebaut, Selbstnutzung offen** (Stufe 3a) |
| Abgleich | ultra-brain | `loomux reconcile` misst jede Quelle der registrierten Bereiche gegen ihr Identitätsregister und öffnet im Prüfzentrum einen Fall für eine veränderte Quelle oder ein Merge-Ereignis, samt Paket mit Diff und Belegen. Neuschrift von `src/brain/maintenance/` (`reconcile`, `case`, `package`, `derive`, die Leseseite von `vcs`, Lese- und Ablageseite des Ereignisprotokolls). Einen Vorschlag des lokalen Modells gibt es erst mit Stufe 4; bis dahin öffnet ein `local_only`-Bereich den Fall ohne Vorschlag. | 🚧 **Gebaut, Selbstnutzung offen** (Stufe 3a) |
| Bereichs-Onboarding | ultra-brain | `loomux area add` meldet einen Bereich in der Registry an, schreibt `[area]` und `[layout]` in `.loomux/config.toml`, legt das Wiki-Gerüst an und die Weichenregel in die Projektanweisung, dann `reindex` (abschaltbar mit `--no-reindex`). Aus `src/brain/init.py`, **ohne** die Hook-Hälfte: `.mcp.json` und die Hook-Dateien schreibt `loomux init` in Stufe 4. | 🚧 **Gebaut, Selbstnutzung offen** (Stufe 3a) |
| Brain-Datenbefehle | ultra-brain | `loomux brain search`, `catalog`, `read`, `neighbors` und `status` über die eine Registry, an der Python-Referenz durch einen aufgezeichneten Fallkorpus gemessen. Eine Registry oder Bereichsdeklaration, die loomux nicht verwenden kann, verweigert den Aufruf und nennt Datei, Eintrag und Grund. | ✅ **Implementiert** (Stufe 1b-1) |
| Brain-zu-Graph Brücke | neu | Code-Symbole verweisen direkt auf Architekturentscheidungen (ADRs) und Dokumentation. | 📋 **Spezifiziert** (Stufe W3) |
| **5. LLM OS & Web-Interface** | | | |
| Eingebettetes Web-OS | ultra-brain | Autarke React/Vite-SPA, per `go:embed` ausgeliefert über `loomux serve` auf `http://127.0.0.1` mit `embed_stub.go`-Fallback. | 📋 **Spezifiziert** (Stufe W1) |
| Interaktiver Graph-Visualizer | neu | D3-Force / WebGL Graph mit Kanten-Chips, Typen-Filterung und Blast-Radius-Overlays. | 📋 **Spezifiziert** (Stufe W3) |
| Kanban Board & Loop Tracker | neu | Echtzeit-Tracking von mehrstufigen Agenten-Workflows, Subagenten-Loops und Prüfketten. | 📋 **Spezifiziert** (Stufe W5) |
| Grafischer Flow-Editor | neu | Visueller DAG-Canvas zum Entwerfen, Abspielen und Debuggen von Agenten-Prüfschleifen. | 💡 **Zukunft** (Stufe W5) |

*Herkunft: `ultraloom` oder `ultra-brain` — aus diesem Repo migriert; `neu` — für loomux gebaut, nicht übernommen.*

*Legende: ✅ Im Go-Binary implementiert & verifiziert · 🧩 Bibliothek gebaut, noch an keinen Befehl verdrahtet · 🚧 In aktiver Migration / Fusion, oder gebaut und noch nicht im eigenen Betrieb von loomux · 📋 Spezifiziert & Bau-Bereit · 💡 Geplant / Zukunftsvision*
