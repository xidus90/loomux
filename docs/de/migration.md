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
| **3a** | ✅ | ultra-brain | Erkennen: `loomux reindex` und `loomux embed` (Umzug von `ultra-brain/pkg/index`), `loomux reconcile` samt Lese- und Ablageseite des Ereignisprotokolls, `loomux area add`, der Auffangdurchgang vor `reindex`; geschrieben wird nach `LOOMUX_STATE_DIR`, das Altverzeichnis nur noch gelesen. 28 Fälle gegen die Python-Referenz, die vier Git-Fälle über `gitworld` aus 2c. Seit dem 2026-09-22 fährt dieser Rechner `reindex` und `embed` über die echte Registry; `[index]` in `.loomux/config.toml` beschränkt `project/loomux` auf `docs/wiki` (Auflage S3 in `parity/stufe-3a.md`) | 1b-1 ✅ | — |
| **3b** | ✅ | ultra-brain | Entscheiden: `loomux cases`, `loomux case`, `loomux approve`; die Anwendung eines Vorschlags (Neuschrift von `apply.py`), die Belegbindung (`evidence`) und die Schreibseite von `vcs`, die über einen eigenen Index auf den aktuellen Ref des Tresors committet; nach einer geschriebenen Freigabe Aufholung und Indexlauf. 24 Fälle gegen die Python-Referenz, 19 ohne Unterschied nach der Normalisierung (stderr nicht verglichen), 5 mit freigegebenen Unterschieden (`parity/stufe-3b.md`). Die Selbstnutzung lief am 2026-09-23 gegen die echte Registry: `cases` und `case` gleich der Python-Referenz, `approve` auf Entscheidung des Nutzers nur mit `--defer` (`parity/stufe-3b.md`, „Selbstnutzung“) | 3a ✅ (die Fälle) | — |
| **3c** | ✅ | ultra-brain | Pflegen: `loomux brain check file\|bundle\|all` (Umzug von `pkg/check/{okf,house,run}`, Referenz das Go-Binär), `loomux lint --scope all\|<scope>` mit den zwölf Regeln von `lint.py` als eigenem Regelsatz neben der Go-Form, die `lint <datei>`, `wiki-gate` und die Lane behalten, `loomux wiki init\|types\|retype`, die tägliche Aufholung in `serve` (nur `reconcile`, Tor vor der ersten Antwort, Hinweiszeilen an den Antworten). `brain check code` fällt weg. 35 Fälle gegen die Referenz, 34 ohne Unterschied, einer freigegeben (`check code`, `parity/stufe-3c.md`). Die lesenden Befehle liefen am 2026-09-23 gegen die echte Registry gleich der Referenz; die Aufholung lief am 2026-09-24 gegen eine Kopie von Zustand und Vault (Tor, Hinweiszeile und neuer Stempel wie spezifiziert), `wiki retype` und `wiki init` warten dort auf den Menschen | 3a ✅ (Upkeep ruft `reconcile`), 1b-2 ✅ (Upkeep läuft in `serve`) | — |
| **4a-1** | 🚧 gebaut; Schritte des Menschen offen (Messung des MCP-Arbeitsverzeichnisses, Prüfung in drei Terminals, ein `config set` durch den Menschen) | neu | Schema und `config`: `internal/config/schema` nennt jeden Schlüssel, den die Leser von `.loomux/config.toml` annehmen, und prüft einen neuen Text über eben diese Leser; `internal/config/edit` ändert eine Zeile, erhält jeden Kommentar und verweigert, was es nicht ohne Raten setzen kann; `internal/tui`, Vollbild-Liste und -Eingabe auf `x/term`. `loomux config list\|get\|set\|unset` und die interaktive Form, jeder Schlüssel mit Herkunft (gesetzt, Vorgabe, Preset, ungesetzt); eine Vorgabe wird nie geschrieben. `[modules]` (`hooks`, `brain`, `graph`) wirkt zur Laufzeit: `hooks = false` schaltet jeden Hook außer dem Wächter ab, `brain = false` die Wiki-Lane und die `brain_*`-Werkzeuge, `graph = false` die `graph_*`-Werkzeuge. `loomux mcp --root` und die Suche nach oben wählen das Projekt, dessen Module gelten. Der Wächter verweigert einem Agenten `loomux init`, ein schreibendes `config` und `area add`; ein Agent schlägt eine Änderung mit `config set`/`unset --propose` vor, die ein Mensch mit `config proposals` durchsieht und mit `config apply` anwendet. `--global` ist verdrahtet, kennt aber noch keinen Schlüssel. Die Brücke sucht vom MCP-Arbeitsverzeichnis aus nach oben, das ein Mensch noch misst (`parity/stufe-4a-1.md`) | 3 ✅ | — |
| **4a-2** | 🚧 gebaut; Schritte des Menschen offen (`init --yes` auf einem frischen Klon mit leerem `git status`, ein Wirt interaktiv eingerichtet, eine Projekt-`.mcp.json` einmal in Claude Code freigegeben) | ultraloom + ultra-brain | `loomux init`: ulinit auf Schema und `tui` neu geschrieben (`write` und das Zusammenführen der Host-Datei samt Tests umgezogen), Module mit alles/einzeln/nichts, `--yes`, `--dry-run`, `--detect-only`, `--hosts`; jede Änderung als Diff, nichts überschrieben, eine Sicherung vor der ersten Änderung außer für `.loomux/config.toml`, die im Ganzen ersetzt wird, sobald ihre Leser den neuen Text annehmen, eine unlesbare Host-Datei oder `.mcp.json` beendet den Lauf, ohne etwas zu schreiben, `answers.toml` und `installed.toml` unter `.loomux/state/`. Teile: das neueste Release nach `${LOCALAPPDATA}/loomux/bin/loomux.exe` (`selfupdate.Install`, eine Erstinstallation) oder in einem Checkout gebautes `bin/loomux.exe`; `.loomux/config.toml`, `.gitignore`, `AGENTS.md`, `.mcp.json`; Host-Einträge ohne Marke; Git-Hooks; `verify-until-green` und die fünf Brain-Skills, englisch; `area add`, der Merge-Hook, `graph build`; Werkzeuge geprüft, nie installiert. `loomux merge-hook install\|status\|remove\|record`: Der post-merge-Hook ruft das Binary, statt Pfade einzubacken; 14 Fälle gegen `brain-mcp hook`, elf ohne Unterschied. Der Wächter verweigert einem Agenten `merge-hook install` und `remove`. Antigravity (gebaut 2026-09-25, Fusions-Spec #21 und #23): vier Einträge (`PreInvocation`, `PreToolUse`, `PostToolUse`, `Stop`) in der Gruppe `loomux` von `.agents/hooks.json`, die das installierte Binary über `cmd.exe` als `%LOCALAPPDATA%/loomux/bin/loomux.exe … --root ..` rufen, keine, solange `LOCALAPPDATA` Leerraum oder ein Zeichen enthält, das `cmd.exe` als Syntax liest; `PreInvocation` und `Stop` als flache Liste von Handlern, wie agy 1.2.11 sie verlangt; die Antworten gemessen mit agy 1.2.11 am 2026-09-25: `pre-tool-use` verweigert mit Exit 2, `post-tool-use` warnt mit Exit 2, ohne abzubrechen, ein gehaltener Stop läuft mit `{"decision":"continue"}` auf stdout weiter, Kontext geht als `injectSteps` beim ersten `PreInvocation` hinaus; `run_command` (`CommandLine`, gemessen) und `send_command_input` (`Input`, ungemessen) laufen durch die Befehlsregeln; Post-Edit liest seine Dateien aus dem `toolCall`, den das PostToolUse von agy trägt; andere Gruppen übernommen, eine mit unserem Befehl genannt; die Skills unter `.agents/skills/`; die Regeln von ultraloom `internal/agenthooks` ins Zusammenführen der Host-Datei eingebaut statt das Paket umzuziehen (`parity/stufe-4a-2.md`). Soll die zwei Handbefehle eines frischen Klons ersetzen | 4a-1 🚧 (gebaut, Schritte des Menschen offen), 2b ✅ (Commit-Sprache), 2c ✅ (`stop`, `subagent-*`), `feat/self-update` ✅ (kanonischer Ort, `internal/swap`) | — |
| **4c** | offen | ultra-brain | Das lokale Modell: Ollama-Client nur auf Loopback, Tor, Richter, Prompts, `[model]`, Vorschläge für `local_only`-Fälle in `reconcile`; Heilung zweier geerbter Fehler aus 3b; `loomux dev bench search` und ein Berichtsschema für `dev bench` | 4a-1 🚧 (gebaut; `[model]` steht im Schema), 3 ✅ | 3 |
| **4d** | offen | ultra-brain | `loomux convert` und `loomux fetch` über `pdftotext` und `yt-dlp`, mit den Modellrollen `describe` und `place` | 4c | 3 |
| **4e** | offen | — | Umstellung der Wirte, eine Checkliste ohne Code: Abgleich des Maschinenzustands, je Wirt `loomux init` und Rauchtest, alte Einträge von Hand entfernen (Einträge von `ulguard` und `brain guard`, die `init` stehen lässt und nennt). Ein von `brain-mcp` eingerichteter post-merge-Hook heißt `unrecorded`, weil die alte `hooks.tsv` nicht gelesen wird: `loomux merge-hook install` je Wirt übernimmt ihn. `loomux migrate` fällt weg (Fusions-Spec, Nachtrag #19) | 4a-2 🚧 (gebaut, Schritte des Menschen offen), 4c, 4d; ein Remote für `brain-knowledge`; der ultra-brain-Zweig `feature/artefakte-nach-lebensdauer` gegen loomux gelesen (Fusions-Spec #20) | 3 |
| **G1** | ✅ | neu | Rang und Blast-Radius als Bibliotheken, an portierten Testvektoren der Referenz belegt | — | — |
| **G2a** | ✅ | neu | Extraktor, Auflösung, Speicher, Frischesonde sowie `graph build` und `graph check` | — | — |
| **G2b** | ✅ | neu | Die Abfrage: lexikalische Saat, die Beiakte, `loomux graph ask` | — | — |
| **G3** | ✅ | neu | `graph_find_code` und `graph_check_freshness` am Gateway von 1b-2; eine Abfrage baut nie einen ersten Graphen | — | — |
| **G4a** | ✅ | neu | Navigationspalette (`callers`, `skeleton`, `grep`, `map`, `stats`) und 4 MCP-Werkzeuge (`graph_file_api`, `graph_trace_calls`, `graph_find_all`, `graph_repo_map`) mit Fail-Closed Privacy | G3 ✅ | — |
| **G4b** | ✅ 2026-09-23 | neu | Git-Diff-Blast-Radius (`blast.Radius`, `graph blast`, MCP `graph_blast`), `check graph-fresh` und `check blast-audit`, die Prüfart `graph` (in der Profilvorgabe `precommit`, ohne Graph `not-applicable`) und der Blast-Monitor im Post-Edit-Hook | G4a ✅ | — |
| **G4c** | offen | neu | Der Stop-Hook mit Blast-Logik: eine Form Arbeitsbaum gegen HEAD, die weiß, dass sie am Rundenende läuft; bis dahin ist `graph` in einem Profil `stop` `not-applicable` | G4b ✅ | 2 |
| **G5** | offen | neu | Mehrsprachige Extraktion über `wazero` | G4b ✅ | 6 |
| **Flow** | offen | ultraloom | Folgeprojekt: die ulflow-Laufzeit, Journal, Resume und Replay, `verify_until_green` als Daten-Flow; Agenten-Flows über Gemini und Claude nach der Multi-Provider-Spec aus ultraloom (Fusions-Spec #22) | ulflow M1 (Zweig `feature/agent-harness`, nicht gemergt) | 4 |
| **W1 – W5** | offen | ultra-brain + neu | Web-OS: Hülle (W1), die Brain-Web-App (W2), Graph-Visualizer (W3), Skill-Suiten und Review (W4), Flow-Editor und Kanban (W5) | W1 an 1b-2 ✅; W2 an W1; W3 an W1 und G4a ✅; W4 an G4b ✅ und 4; W5 an W1 und Flow | 5 |

Die Stufen und worauf jede wartet, gezeichnet aus der Spalte „Hängt ab von“
oben (`P` ist die Priorität einer offenen Stufe, 1 zuerst). Ein Pull Request,
der eine Zeile ändert, ändert den Knoten mit:

```mermaid
flowchart TD
    subgraph Fusion["Fusionsspur"]
        s1a["1a Pilot, Wächter, Post-Edit"]:::done
        s1b1["1b-1 Brain-Lesebefehle"]:::done
        s1b2["1b-2 serve, MCP, stdio-Brücke"]:::done
        s1b3["1b-3 Wiki und Doku umgezogen"]:::done
        s2a["2a Prüfkette"]:::done
        s2b["2b commit-msg"]:::done
        s2c["2c Stop-Tor, Subagenten-Hooks"]:::done
        s3a["3a Erkennen: reindex, embed, reconcile"]:::done
        s3b["3b Entscheiden: cases, case, approve"]:::done
        s3c["3c Pflegen: brain check, Wiki-Typen"]:::done
        s4["4 init, migrate, lokales Modell, Host-Umstellung · P3"]:::planned
    end

    subgraph Code["Code-Graph-Spur"]
        g1["G1 Ranking- und Blast-Bibliotheken"]:::done
        g2a["G2a Extraktor, graph build und check"]:::done
        g2b["G2b graph ask"]:::done
        g3["G3 Graph-MCP-Werkzeuge"]:::done
        g4a["G4a Navigationspalette"]:::done
        g4b["G4b Diff-Blast, Prüfart graph, Edit-Monitor"]:::done
        g4c["G4c Stop-Hook mit Blast-Logik · P2"]:::planned
        g5["G5 Mehrsprachig über wazero · P6"]:::planned
    end

    subgraph WebOS["Web OS · P5"]
        w1["W1 Shell"]:::planned
        w2["W2 Brain-Web-App"]:::planned
        w3["W3 Graph-Visualisierung"]:::planned
        w4["W4 Skill-Suiten und Review"]:::planned
        w5["W5 Flow-Editor und Kanban"]:::planned
    end

    ulflow["ulflow M1<br/>(Branch feature/agent-harness, nicht gemergt)"]:::external
    flow["Flow: ulflow-Laufzeit, Journal, Replay · P4"]:::planned

    s2a --> s2c
    s1b1 --> s3a
    s3a --> s3b
    s3a --> s3c
    s1b2 --> s3c
    s2b --> s4
    s2c --> s4
    s3b --> s4
    s3c --> s4
    g3 --> g4a
    g4a --> g4b
    g4b --> g4c
    g4b --> g5
    ulflow --> flow
    s1b2 --> w1
    w1 --> w2
    w1 --> w3
    g4a --> w3
    g4b --> w4
    s4 --> w4
    w1 --> w5
    flow --> w5

    classDef done fill:#d4edda,stroke:#28a745,color:#155724
    classDef partial fill:#fff3cd,stroke:#d39e00,color:#664d03
    classDef planned fill:#f1f3f5,stroke:#868e96,stroke-dasharray:5 5,color:#495057
    classDef external fill:#ffffff,stroke:#6f42c1,stroke-dasharray:2 2,color:#6f42c1
```

Was die beiden Quellrepos können und bisher keine Stufe übernommen hatte, steht
in der [Fusions-Spec](../.superpowers/specs/2026-09-14-loomux-fusion-design.md) unter „Nachgetragen“, je mit einer
vorgeschlagenen Stufe oder einem vorgeschlagenen Wegfall; die Zuordnung gilt
nach der Freigabe. Freigegeben sind #1 und #2 (Stufe 3c, gebaut mit 3c), #3 und #17 (Stufe 3a),
alle am 2026-09-19, und #18 (Wegfall von `brain check code`) am 2026-09-23. #17 — `reindex` und `embed` als Befehle — war bis dahin
keiner Stufe zugeordnet; die Lücke ist seit dem 2026-09-19 geschlossen, gebaut
mit 3a. #5 (der post-merge-Hook, als `loomux merge-hook`), #7 (die
Brain-Skills), #8 (`verify-until-green`), #10 (die erzeugte `AGENTS.md`),
#11 (die `.gitignore`-Einträge), #15 (das Binary am kanonischen Ort) und #16
(`init --detect-only`) wurden am 2026-09-24 mit 4a-2 gebaut.

Jede Stufe endet grün und wird einzeln übergeben, mit eigenem Plan und — sobald
sie fertig ist — eigener Paritätsakte, die jede ihrer Verfügungen festhält.

## Funktionen

Wo die einzelnen Funktionen stehen:

| Säule / Funktion | Herkunft | Beschreibung | Status |
|---|---|---|---|
| **1. Hooks & Wächter** | | | |
| Einheitlicher Pre-Tool Wächter | ultraloom + ultra-brain | Prüfung von Schreibschranken, Pfadregeln und verbotenen Befehlen (< 35 ms Zielbudget; 32–34 ms gemessen am Vorgänger; ein Write in einem verknüpften Worktree gemessen 34,6 ms warm (2026-09-16)). Verknüpfte Git-Worktrees eines registrierten Workspace sind ohne eigenen Registry-Eintrag beschreibbar. Registry und Bereichsdeklarationen laufen durch dieselben Prüfungen wie die Brain-Befehle; ein kaputter Eintrag verweigert jeden Write. | ✅ **Implementiert** (Stufe 1a) |
| Post-Tool Prüf-Lanes | ultraloom | Das Profil `edit` aus `[verify]` auf der eben geänderten Datei, für ihren Stack und in ihrem Bereich — `go vet` und `gofmt` der einen Datei, der Wiki-Lint im eigenen Prozess, ruff/mypy, eslint/tsc, stylelint und die übrigen, aus denselben Presets, die `loomux check` fährt. Befehle starten als argv, ohne Shell. Eine gescheiterte Lane endet mit 2; Lanes, die wegen eines fehlenden Werkzeugs oder eines aufgebrauchten Budgets (`--budget`, Vorgabe 50 s) ausfallen, werden dem Modell namentlich zurückgemeldet. | ✅ **Implementiert** (Stufe 1a; Lanes aus `[verify]` seit 2a) |
| Post-Tool Blast Monitor | neu | Nach einem Edit an einer `.go`-Datei ohne rote Lane extrahiert der Post-Edit-Hook die Datei neu, vergleicht den Rumpf-Hash jedes Symbols mit dem Graphen auf der Platte und nennt die direkten Aufrufer in anderen Dateien dessen, was sich geändert hat oder fehlt (höchstens zehn), dazu einen Hinweis bei einem geänderten Typ, dessen Kopplung der Graph nicht verdrahtet. Schweigt ohne Graph, bei einem fremden Graphen oder einem Parsefehler; blockiert nie. Kostet 24,7 ms auf dem 7,07-MiB-Graphen dieses Repositorys (2026-09-23). | ✅ **Implementiert** (Stufe G4b) |
| Blast-Audit im Stop-Hook | neu | Der Blast-Audit am Rundenende, Arbeitsbaum gegen `HEAD`, für das Stop-Tor. | 💡 **Geplant** (Stufe G4c) |
| Sitzungsstart | ultraloom | Hält den Commit fest, auf dem eine Sitzung beginnt, und warnt, wenn das Binary im Projekt älter ist als `go.mod`, `go.sum` oder eine `.go`-Datei unter `cmd/` oder `internal/`. Kündigt nur an; blockiert nie einen Zug. | ✅ **Implementiert** (Stufe 1a) |
| Subagent-Drift & Stop-Tor | ultraloom | `loomux hook stop` fährt an jedem Rundenende mit neuem Inhalt das Profil `stop` (vorgegeben `lint`, `types`, `test` und `coverage`, in einem Budget von 270 s; `graph` bleibt draußen, siehe Stufe G4c), hält die Runde bei einer roten Lane mit Exit 2 an und gibt nach 3 Blockaden in Folge auf; ein Rundenende ohne Neues kostet auf diesem Repository 169,5 ms. `subagent-start` und `subagent-stop` halten `origin`, die lokalen Branches und `HEAD` fest und parken, was sich bewegt hat, für das Stop-Tor des Hauptagenten. Host-Adapter für Claude Code und Antigravity implementiert. | ✅ **Implementiert** (Stufe 2c) |
| Prüfbefehle | ultraloom | `loomux check commit-msg` (die Sprache jeder Zeile, der Kopf nach Conventional Commits, `--language`, `--calibrate` und `[commit]`), `check gofmt` (Formatierung, mit dem Exit-Code, den `gofmt -l` nicht gibt), `check gocover` (100 % je Funktion gegen ein Profil, oder eine Gesamtgrenze mit `--floor`) und die beiden Hälften der Graph-Lane, `check graph-fresh` (Neubau bei Drift, rot nur bei fehlgeschlagenem Neubau, einem Fehler der Probe oder einer gehaltenen Sperre) und `check blast-audit` (rot, wenn ein geänderter Bereich mit genug Aufrufern keinen geänderten Test hat). `dev covergate` gibt es nicht mehr. | ✅ **Implementiert** (Stufe 1a; `gocover` 2a; `commit-msg` 2b; `graph-fresh`, `blast-audit` G4b) |
| Prüfketten-Tabelle | ultraloom | Eine Tabelle `[verify]` treibt `loomux check <profil>` und den post-edit-Hook: Presets je Stack, die ohne jede Konfiguration gelten, eine Lane je Art, Stack und Bereich, `after`-Kanten statt Stufen, ein Urteil je Art und `--show`, das zeigt, was läuft. Eine Lane kann Dateien nennen, die sie braucht (`needs`): die C++-Lanes auf dem Build-Baum warten auf `build/CMakeCache.txt` und sind `unready`, bis der Build konfiguriert ist; ein Edit führt `clang-format` auf der Datei trotzdem aus. Kindprozessbäume werden bei einer Frist ganz beendet (unter Windows über ein Job Object). | ✅ **Implementiert** (Stufe 2a) |
| Worktree-Spiegelung | ultraloom | Isolierte Subagent-Git-Worktrees mit NTFS-Junctions und Sitzungsverfolgung. | ✅ **Implementiert** (Stufe 1a) |
| Konfiguration und Einrichtung | ultraloom + neu | `loomux config` zeigt jede Einstellung von `.loomux/config.toml` mit Herkunft (gesetzt, Vorgabe, Preset) und ändert sie zeilengenau mit Bestätigung; `loomux init` richtet ein Projekt in Modulen ein (Hooks, Wiki, Graph) und schreibt die Wahl nach `[modules]`, das zur Laufzeit gilt. Ersetzt ulinit, `install.ps1` und die Hook-Hälfte von `brain init`. Gebaut mit 4a-1: `loomux config` und `[modules]` mit Laufzeitwirkung; gebaut mit 4a-2: `loomux init` mit seinen Teilen (Binary, Konfiguration, `.gitignore`, `AGENTS.md`, `.mcp.json`, Host-Einträge, Git-Hooks, Skills, Bereich, Merge-Hook, Graph), `--dry-run`, `--detect-only` und `--yes`. Seine Läufe durch einen Menschen auf einem frischen Klon und in einem Wirt stehen aus | 🚧 **In Migration** (4a-1 und 4a-2 gebaut, Schritte des Menschen offen) |
| Zonenfreier Startpfad | neu | Gos lokale Zeitzone bleibt vom Hook-Pfad fern: Der TOML-Parser baut seine lokalen Zonen beim ersten Gebrauch (`third_party/toml`), und ein Test im Tor lässt jedes Paket-Init über 500 Allokationen scheitern. `hook pre-tool-use` 7,5 ms warm gegen 26,5 ms vorher (gemessen 2026-09-17). | ✅ **Implementiert** (ohne Stufe) |
| Claude-Mods-Adapter | neu | Die Schreibschranke in einen `tool.check`-Function-Hook setzen ([claude-code#91870](https://github.com/anthropics/claude-code/issues/91870)), der über `$.mcp.call` mit einem langlebigen loomux spricht — entfernt den Spawn, bringt ein `ask`-Urteil und eine gerenderte Begründung. Nur für Claude Code; der Exec-Hook bleibt der portable Pfad. | 💡 **Optional** (ohne Stufe) |
| **2. Skills & Best Practices** | | | |
| Brain- und Verify-Skills | ultra-brain + ultraloom | `loomux init` legt die fünf Brain-Skills (`brain-ingest`, `brain-land`, `brain-research`, `brain-review`, `brain-wiki-plan`), ins Englische übersetzt und mit loomux-Befehlen, und `verify-until-green` (`loomux check all` statt `uv run ultraloom check all`) nach `.claude/skills/` eines Wirts, für Antigravity nach `.agents/skills/`; ein vorhandener Skill bleibt. `session-handover` fällt weg (Fusions-Spec #9). Noch in keinem Wirt in Gebrauch | 🚧 **In Migration** (Stufe 4a-2 gebaut, Schritte des Menschen offen) |
| Kuratierte Sprach-Suiten | neu | Eingebettete Best-Practice-Regeln für Go (Zero-Alloc, Error-Handling, no-init), Python, TS, Rust. | 📋 **Spezifiziert** (Stufe W4) |
| Graph-gestützter Code-Review | neu | Review-Skills, die via `graph_blast` Aufrufer-Auswirkungen prüfen und ADRs abgleichen. | 📋 **Spezifiziert** (Stufe W4) |
| 3-Kanal-Distribution | neu | Konfiguriert via `.loomux/config.toml`, synchronisiert in Host-Ordner, via MCP-Prompts oder Web OS. | 📋 **Spezifiziert** (Stufe W4) |
| **3. Code-Graph & Loop** | | | |
| Nativer Go-AST-Extraktor | neu | Deterministische Symbol- & Kantenextraktion allein via `go/parser` und `go/ast` — kein `go/types`, kein Build ($0, 0 Deps). Hinter `loomux graph build` verdrahtet; braucht auf diesem Repository in der Größenordnung eines Zehntel einer Sekunde, mit Befehl und Rohausgabe gemessen in `docs/de/benchmarks.md`. | ✅ **Implementiert** (Stufe G2a) |
| Personalized PageRank | neu | Power-Iteration Random-Walk-Ranking über Aufruf- und Abhängigkeitsgraphen, ungerichtet über fünf Relationen, max-normiert mit deterministischer Gleichstandsordnung. Verschmolzen mit BM25-Kandidaten-Scoring in `loomux graph ask` (~48 ms warmes Retrieval). | ✅ **Implementiert** (Stufe G2b) |
| Blast-Radius-Engine | neu | Transitive Hülle und Impact-Analyse (`In`/`Out`, Tiefenbegrenzung, kleinste Tiefe gewinnt). `EdgeWalk`, `Resolve` und `InDegree` treiben die Graph-Navigation; `blast.Radius` führt einen git-Diff auf die Symbole, die seine Hunks berühren, auf das, was sie erreicht, auf ein Testsignal (`changed`, `stale`, `none`, `na`) und auf zitierte Belege, hinter `loomux graph blast` und dem MCP-Werkzeug `graph_blast`. | ✅ **Implementiert** (Stufe G1, erweitert in G4a und G4b) |
| Symbol-gekoppelter Grep | neu | Regex-Suche, gruppiert nach umschließendem Symbol und gerankt nach Kanten-Grad (`inDegree`). Als CLI-Befehl `loomux graph grep` und MCP-Werkzeug `graph_find_all` mit Fail-Closed Privacy. | ✅ **Implementiert** (Stufe G4a) |
| Graph-Navigationspalette | neu | Vollständige Struktur- und Aufrufer-Navigation über den deterministischen AST-Graphen: `loomux graph callers`, `skeleton`, `map` und `stats` sowie MCP-Werkzeuge `graph_file_api`, `graph_trace_calls` und `graph_repo_map` mit Fail-Closed Privacy auf dem Cloud-Kanal. | ✅ **Implementiert** (Stufe G4a) |
| Multi-Language AST | neu | CGo-freier Tree-sitter über WebAssembly (`wazero`) mit persistentem AOT-Kompilierungs-Cache. | 💡 **Geplant** (Stufe G5) |
| MCP-Dienst & stdio-Brücke | ultra-brain | `loomux serve` hält zwei Loopback-Listener, je einen pro Kanal und jeden mit eigenem Token, und beantwortet zwölf Werkzeuge über Streamable HTTP — die fünf `brain_*`-Werkzeuge und, seit den Stufen G3, G4a und G4b, die sieben `graph_*`-Werkzeuge (`graph_find_code`, `graph_check_freshness`, `graph_file_api`, `graph_trace_calls`, `graph_find_all`, `graph_repo_map`, `graph_blast`); `loomux serve status` und `stop [--force]` steuern ihn, und `loomux mcp` ist die stdio-Brücke, die ein Wirt startet und die den Dienst selbst startet und ersetzt. Seit Stufe 3c holt der Dienst einen fälligen `reconcile` beim Start und danach täglich selbst nach, lässt jedes `brain_*`-Werkzeug auf den ersten Durchgang warten und hängt dessen Befund an die Antworten. `internal/hooks` bindet nichts davon: ein Tor-Test liest den Importgraphen. Die Front ist durch einen aufgezeichneten Fallkorpus an die MCP-Front der Python-Referenz gemessen; verglichen wird der Text jeder `CallToolResult` und `isError`, nicht der Umschlag, den zwei verschiedene SDKs aushandeln. | ✅ **Implementiert** ((Stufen 1b-2, G3, G4a, 3c, G4b)) |
| **4. Second Brain & Wiki** | | | |
| Lokales Markdown-Wiki | ultra-brain | Das Bündel selbst liegt in `docs/wiki/` (Bereich `project/loomux`, am 2026-09-16 Seite für Seite umgezogen und zeilenweise freigegeben). `loomux lint <datei>` prüft Links und Frontmatter einer Seite, `loomux wiki-gate` Frische und Struktur des Bündels, und die Lane `lint/wiki` seine Struktur in `loomux check lint` und an jedem Rundenende. Identitätsregister und Themen-Graph schreibt `loomux reindex` aus Stufe 3a, nicht der Umzug; die Seitenregeln von `brain check` (OKF, Haus, Föderation) und der Lint über alle Bündel kamen mit Stufe 3c. | ✅ **Implementiert** (Stufen 1b-3, 3a, 3c) |
| Semantischer QMD-Index | ultra-brain | Einbettung lokaler Vektoren und neuronaler Suche mit Caching in `~/.cache/qmd`. Die Sammlungen trägt `loomux reindex` in qmds `index.yml` ein, die offenen Vektoren erzeugt `loomux embed` (Zeile „Index-Neubau und Einbettung“). | ✅ **Implementiert** (Stufe 3a) |
| Index-Neubau und Einbettung | ultra-brain | `loomux reindex` baut je Bereich Identitätsregister, Kataloge, Linkgraph und die qmd-Sammlungen neu und fährt vorher den Auffangdurchgang (`reconcile`), damit keine Wissensänderung am Prüftor vorbeigeht; `loomux embed` erzeugt über `QmdMcpPort.Embed` die Vektoren, die `reindex` offen lässt. Umzug von `ultra-brain/pkg/index` samt Tests. Bis zum 2026-09-19 war beides keiner Stufe zugeordnet (Nachtrag #17 der Fusions-Spec). | ✅ **Implementiert** (Stufe 3a) |
| Abgleich | ultra-brain | `loomux reconcile` misst jede Quelle der registrierten Bereiche gegen ihr Identitätsregister und öffnet im Prüfzentrum einen Fall für eine veränderte Quelle oder ein Merge-Ereignis, samt Paket mit Diff und Belegen. Neuschrift von `src/brain/maintenance/` (`reconcile`, `case`, `package`, `derive`, die Leseseite von `vcs`, Lese- und Ablageseite des Ereignisprotokolls). Einen Vorschlag des lokalen Modells gibt es erst mit Stufe 4; bis dahin öffnet ein `local_only`-Bereich den Fall ohne Vorschlag. Entschieden werden die Fälle mit Stufe 3b (Zeile „Prüfzentrum entscheiden“). | ✅ **Implementiert** (Stufe 3a) |
| Prüfzentrum entscheiden | ultra-brain | `loomux cases` listet die wartenden Fälle, `loomux case <id>` zeigt Kopf, Paket und Vorschlag (bei `local_only` zurückgehalten bis `--package`), `loomux approve <id>` entscheidet: freigeben, mit `--amend` einen eigenen Vorschlag freigeben, mit `--reject` verwerfen oder mit `--defer` zurückstellen. Jede Behauptung des Vorschlags braucht ein wörtliches Zitat aus dem Paket; die Freigabe schreibt Seite, Frontmatter, Register, `log.md` und `audit.md`, entfernt den Fall und committet genau diese Pfade. Neuschrift von `apply.py` und `vcs.commit_paths`, Umzug von `evidence`, `patch`, `frontmatter`, `format`, `lookup`, `cases`. Warm 26–27 ms gegen 841–852 ms der Python-Form für `cases`, `case` und `approve --defer`; eine schreibende Freigabe ist nicht gemessen (`docs/de/benchmarks.md`, 2026-09-23). | ✅ **Implementiert** (Stufe 3b) |
| Bereichs-Onboarding | ultra-brain | `loomux area add` meldet einen Bereich in der Registry an, schreibt `[area]` und `[layout]` in `.loomux/config.toml`, legt das Wiki-Gerüst an und die Weichenregel in die Projektanweisung, dann `reindex` (abschaltbar mit `--no-reindex`). Aus `src/brain/init.py`, **ohne** die Hook-Hälfte: `.mcp.json` und die Hook-Dateien schreibt `loomux init` (Stufe 4a-2), das `area add` auch als seinen Teil `area` ruft. | ✅ **Implementiert** (Stufe 3a) |
| Merge-Hook | ultra-brain | `loomux merge-hook install\|status\|remove` legt den post-merge-Hook in jedes Repository eines Bereichs, dessen Manifest `[maintenance] on_merge = true` sagt, meldet ihn in sieben Zuständen und nimmt ihn zurück; der Hook ruft `loomux merge-hook record`, das den Merge aus der Registry für `reconcile` vormerkt, nie etwas ausgibt und nie einen Merge scheitern lässt. `brain-mcp hook` der Referenz ohne eingebackene Pfade; 14 Fälle, elf ohne Unterschied. `loomux init` richtet ihn als Teil `merge-hook` ein. Noch in keinem Wirt in Gebrauch | 🚧 **In Migration** (Stufe 4a-2 gebaut, Schritte des Menschen offen) |
| Wiki-Pflege | ultra-brain | `loomux brain check file\|bundle\|all` prüft Seiten, Bündel und Föderation nach OKF und Hausregeln (Go-Referenz); `loomux lint --scope` lintet jedes registrierte Bündel nach den zwölf Regeln von `lint.py`; `loomux wiki init` legt ein Bündel an, `wiki types` zählt die Seitentypen über alle Bereiche mit Rang, `wiki retype` benennt einen Typ in einem Bündel um. `brain check code` ist entfallen (Fusions-Spec #18). | ✅ **Implementiert** (Stufe 3c) |
| Brain-Datenbefehle | ultra-brain | `loomux brain search`, `catalog`, `read`, `neighbors` und `status` über die eine Registry, an der Python-Referenz durch einen aufgezeichneten Fallkorpus gemessen. Eine Registry oder Bereichsdeklaration, die loomux nicht verwenden kann, verweigert den Aufruf und nennt Datei, Eintrag und Grund. | ✅ **Implementiert** (Stufe 1b-1) |
| Self-Update | neu | Das maschinenweite Binary, aus dem MCP-Brücke und `serve` laufen, liegt in `%LOCALAPPDATA%\loomux\bin\loomux.exe` und kommt aus einem Release, nie aus einem Checkout. `serve` prüft einmal am Tag über `gh` das neueste Release seines eigenen Kanals, prüft `SHA256SUMS` und tauscht die Datei; die nächste Brücke ersetzt dann `serve`. `loomux self-update` tut dasselbe von Hand, und der Sitzungsstart warnt, wenn `serve` woanders läuft oder das Update gescheitert ist. Nur Windows. Am 2026-09-23 von Hand eingerichtet, nachdem die Brücke tagelang aus einem veralteten Checkout lief. | ✅ **Implementiert** |
| Brain-zu-Graph Brücke | neu | Code-Symbole verweisen direkt auf Architekturentscheidungen (ADRs) und Dokumentation. | 📋 **Spezifiziert** (Stufe W3) |
| Lokales Modell | ultra-brain | Ollama nur auf Loopback schreibt Vorschläge für `local_only`-Fälle, geprüft mit derselben Belegbindung wie jede Freigabe; ein Ausfall führt zu einem Fall ohne Vorschlag, nie in die Cloud. Dazu `loomux dev bench search`, das den Trefferrang der Suche misst | 📋 **Spezifiziert** (Stufe 4c) |
| Eingang | ultra-brain | `loomux convert` wandelt PDFs und Transkripte im Eingang eines Bereichs mit Herkunftskopf; `loomux fetch` holt die Untertitel eines Videos | 📋 **Spezifiziert** (Stufe 4d) |
| **5. LLM OS & Web-Interface** | | | |
| Eingebettetes Web-OS | ultra-brain | Autarke React/Vite-SPA, per `go:embed` ausgeliefert über `loomux serve` auf `http://127.0.0.1` mit `embed_stub.go`-Fallback. | 📋 **Spezifiziert** (Stufe W1) |
| Interaktiver Graph-Visualizer | neu | D3-Force / WebGL Graph mit Kanten-Chips, Typen-Filterung und Blast-Radius-Overlays. | 📋 **Spezifiziert** (Stufe W3) |
| Kanban Board & Loop Tracker | neu | Echtzeit-Tracking von mehrstufigen Agenten-Workflows, Subagenten-Loops und Prüfketten. | 📋 **Spezifiziert** (Stufe W5) |
| Grafischer Flow-Editor | neu | Visueller DAG-Canvas zum Entwerfen, Abspielen und Debuggen von Agenten-Prüfschleifen. | 💡 **Zukunft** (Stufe W5) |

*Herkunft: `ultraloom` oder `ultra-brain` — aus diesem Repo migriert; `neu` — für loomux gebaut, nicht übernommen.*

*Legende: ✅ Im Go-Binary implementiert & verifiziert · 🧩 Bibliothek gebaut, noch an keinen Befehl verdrahtet · 🚧 In aktiver Migration / Fusion, oder gebaut und noch nicht im eigenen Betrieb von loomux · 📋 Spezifiziert & Bau-Bereit · 💡 Geplant / Zukunftsvision*
