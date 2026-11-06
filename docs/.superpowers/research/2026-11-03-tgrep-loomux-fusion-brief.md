# Brief: Fusion aus Microsoft tgrep und dem loomux-Code-Graph

Stand: 2026-11-03. Arbeitspapier, deutsch. Grundlage für Design und Plan, die
`claude-hb` daraus erstellt.

## Auftrag

Ein DeepSeek-Harness-Plugin (DSH-Plugin), das **Microsoft tgrep** und den
**Code-Graph-Teil von loomux** zu einer Werkzeugfläche fusioniert: schnelle
Textsuche über einen Trigramm-Index, deren Treffer sofort Graph-Knoten sind,
mit Kanten, Rang und Blast-Radius statt bloßer Datei:Zeile-Listen.

## Heutiger Stand (belegt)

### DSH und das bestehende Plugin

- DSH lädt Plugins als npm-Pakete über `cordis.patch.yml` in einem Profil
  (`~/.dsh/profiles/<profil>/`). Ein `insert`-Eintrag bindet Name und Config.
- `dsh-tgrep` v0.1.13 (`https://github.com/huuthuan-nguyen/dsh-tgrep`, MIT,
  Node >= 22.19) schattet das eingebaute `grep`-Tool: es registriert sich in
  der Tool-Ebene des Agenten und ersetzt `@deepseek-ai/dsh-tool-fs-search`.
  Es startet bei Bedarf `tgrep serve` als Daemon, beendet ihn bei Idle oder
  Harness-Ende und liefert `matches` plus `SearchMeta` für die Suchkarte.
  Maße: 30 s Timeout, 300 Zeilen Voreinstellung, 20 MiB stdout-Deckel,
  5000 Treffer Maximum.
- Installationskopie: `C:\Users\micro\.dsh\profiles\desktop\node_modules\dsh-tgrep\`
  (`index.js`, `cordis.patch.yml`, `README.md`, `package.json`).
- **tgrep ist auf diesem Rechner derzeit nicht auf PATH.** Das DSH-`grep`-Tool
  schlägt mit `spawn tgrep ENOENT` fehl. Die externe Binary-Abhängigkeit ist
  damit praktisch schon gebrochen — ein Argument für die Fusion.

### Microsoft tgrep (github.com/microsoft/tgrep)

- Rust, Trigramm-Index plus Client/Server; Regex-Suche in großen Repos;
  in GitHub Copilot CLI integriert.
- Index unter `.tgrep/`: `lookup.bin` (sortierte 16-Byte-Einträge:
  Trigramm u32 + Offset u64 + Länge u32), `index.bin` (6-Byte-Postings:
  file_id u32 + loc_mask u8 + next_mask u8), `files.bin`, `files-extra.bin`,
  `meta.json`, `filestamps.json`, `serve.json`, `serve.lock`.
- Suchpfad: Trigramme grenzen Kandidatendateien ein, dann verifiziert der
  Regex-Engine parallel in den Kandidaten; ohne nutzbare Trigramme wird jede
  indizierte Datei geprüft. Benchmark (24.08.2026): schneller als ripgrep in
  17 von 18 Repo/Plattform-Kombinationen, 0,93x bis 51,9x (Client/Server-
  Latenz bei fertigem Index, ohne Index-Bauzeit).
- CLI: `-i/-s/-S`, `-F`, `-w`, `-v`, `-o`, `-e`, `-f`, `-U`, `--multiline-dotall`,
  `-n/-N`, `-c`, `-l`, `--files-without-match`, `-q`, `-m`, `-g`, `--iglob`,
  `-t/-T`, `--type-add`, `--files`, `-A/-B/-C`, `--heading`, `-H/-I`,
  `--json` (ripgrep-kompatibler JSON-Strom), `--vimgrep`, `--stats`,
  `--no-index`, `--hidden`, `--no-ignore`, `-a/--text`, `--max-filesize`
  (Vorgabe 64M), `-L/--follow`, `--no-messages`, `--index-path`,
  `--exclude` (nur `index`/`serve`), `-r/--replace`.
- Ignorierregeln: `.gitignore` nur im Git-Repo (Warnung sonst), `--no-require-git`
  für Repos ohne `.git`; Nested-Ignores; `core.ignorecase`-Sonderpass.
- Index-Lebenszyklus: `serve` aktualisiert asynchron; ohne Server ist der
  Index nach Änderungen veraltet (`tgrep index .` erneut laufen lassen);
  `--no-index` liest den aktuellen Stand. Journal, Lock, Staging, externe
  Merge-Sort-Strategie, Speicher-/Platten-Vorabprüfungen.
- Format, Server, Shared Worktrees und RPC sind tgrep-Produktumfang — für
  eine Fusion ist nur der Kern interessant: Trigramme, Postings,
  Kandidatenfilter, Verifikation, Speicherstrategie.

### loomux-Code-Graph

- Ein Go-Binary, CGo-frei. Graph-Engine unter `internal/code/…`, Zustand in
  `.loomux/state/graph/wiring.json`, Freshness-Record in
  `.loomux/state/graph/cache/fingerprint.json`, Extract-Cache daneben.
- `loomux graph` (Doku: `docs/en/cli-reference.md` §6, deutsch
  `docs/de/cli-reference.md` §6): `build`, `check`, `ask`, `callers`,
  `skeleton`, `grep`, `map`, `stats`, `blast`; `viz` ist nur spezifiziert.
  Alle außer `build`/`check`/`stats` haben `--json`.
- `graph grep <pattern>` existiert bereits: Regex über die indizierten
  Dateien, **gruppiert nach umschließendem Symbol**, sortiert nach
  `inDegree`; Flags `-i`, `--fixed`, `--in`, `--max-hits` (Vorgabe 300),
  `--no-refresh`, `--json` (`grep.Result`).
- `graph ask` rankt lexikalisch (BM25-artig) plus Personalized PageRank;
  `callers` läuft Kanten in/out mit Tiefe; `skeleton` liefert Signaturen
  ohne Rümpfe; `map` liefert Verzeichnis-Cluster/Hubs/Hotspots; `blast`
  verbindet einen Git-Diff mit dem Graphen.
- Freshness: alle lesenden Graph-Befehle prüfen den Freshness-Record und
  bauen bei Drift (oder fehlendem Sidecar) im Hintergrund neu; `--no-refresh`
  überspringt das.
- Sprachen: Go (`go/parser`), Python und GDScript (`gotreesitter`, v0.55.1);
  TypeScript/C++ stehen als Roadmap-Zeilen in `README.md`/`README.de.md`.
- MCP-Dienst (Doku §8): u. a. `graph_find_code`, `graph_check_freshness`,
  `graph_blast` und vier Navigations-Tools. **Wichtig:** Die Graph-CLI
  filtert `[privacy] never`-Pfade nicht, die MCP-Tools tun es. Ein Plugin,
  das die CLI aufruft, muss diese Schranke selbst beachten.
- Quelle: `internal/code/sourceset` bestimmt die Dateiliste (was git kennt);
  der Graph wird bei jedem `build` aus denselben Bytes gebaut, der
  Extract-Cache spart nur das Parsen.

### Projektregeln (aus `AGENTS.md`, gelten für die Umsetzung)

- Ein Go-Binary, CGo-frei, keine Abhängigkeitspakete ohne Not.
- 100 % Coverage je Funktion; Ausnahme nur mit `//coverage:exempt <Grund>`
  direkt über `func`. Kein `init()`; keine Paketvariable parst eingebettete
  Daten, Laden beim ersten Zugriff.
- Code, Kommentare, Fehlermeldungen, Commit-Messages: Englisch. Arbeitspapiere
  unter `docs/.superpowers/` sind Deutsch und werden nicht übersetzt.
- Conventional Commits, ein Commit je Thema, kein Commit/Push durch Agenten,
  jede Änderung über Branch und Pull Request. Label `release:minor` für ein
  neues Feature, `Release:`-Zeile und `## Changelog` im PR-Body.
- README.md und README.de.md bei Fähigkeitsänderungen mitpflegen; Roadmap in
  beiden READMEs aktualisieren. Performance-Zahlen chronologisch in
  `docs/en/benchmarks.md` und `docs/de/benchmarks.md`.

## Entscheidung des Leads (überschreibbar, aber begründet)

**Der Trigramm-Index wird in Go in loomux gebaut, nicht als tgrep eingebettet
und nicht als tgrep-Port.** Gründe:

1. Ein Binary, CGo-frei, deterministisch — kein zweiter Prozess, kein Port,
   kein Lock, keine `.tgrep/`-Nebenstruktur.
2. Der Index entsteht im selben Durchlauf wie der Graph aus derselben
   Dateiliste und denselben Bytes. Index und Graph können nicht auseinander-
   laufen; der Freshness-Record gilt für beide.
3. Der Index kann **Symbole** mitindizieren (Name, qualifizierter Name,
   Signatur), die der Graph ohnehin extrahiert. Treffer sind dann Graph-Knoten
   mit Kanten und Rang, nicht nur Textzeilen — das ist die eigentliche Fusion.
4. Die bestehende `loomux graph grep`-Semantik (Gruppierung nach Symbol,
   inDegree-Rang) bleibt; der Index beschleunigt nur die Kandidatenauswahl.

Das DSH-Plugin bleibt die dünne Fassade: es schattet weiter `grep`, ruft aber
`loomux graph grep --json` (bzw. die MCP-Tools) statt der tgrep-Binary. tgrep
bleibt optionaler Fallback für Repos/Sprachen, die der Graph nicht liest.

## Minimalumfang des Index (Vorschlag, im Design zu schärfen)

- Trigramme aus dekodierten Bytes; Posting-Listen (Trigramm → Datei +
  Positionen); Kandidatenmenge; Verifikation mit Go-`regexp` (RE2) parallel.
- Speicherstrategie: Posting-Puffer mit Spill auf Platte (externes Merge-Sort)
  oder bewusst In-Memory mit dokumentierter Grenze — Entscheidung im Design.
- Persistenz unter `.loomux/state/graph/` (eigener Unterordner), Version im
  Header, Format-Änderung erzwingt Neuaufbau wie beim Graph (Schema-Version).
- Aufbau im `graph build`; `graph grep` nutzt den Index, wenn er zum
  Freshness-Record passt, sonst Scan wie heute. Kein Server.
- Symbol-Tabelle: Knoten-ID → Name, Qualifizierter Name, Signatur, Datei,
  Zeilenspanne; Suche kann Text- und Symbol-Trigramme mischen.
- Determinismus: gleiche Eingabe → gleiche Indexbytes; keine Zeitstempel im
  Inhalt außer im Meta-Record.

## Plugin-Seite (Vorschlag)

- Weiter `grep` schatten, Tool-Schema kompatibel halten
  (`pattern`, `path`, `include`, `case_insensitive`, `max_results`), damit
  bestehende Prompts und Karten weiter funktionieren.
- Ergebnis: `matches` wie heute, zusätzlich je Treffer optional
  `symbol`, `kind`, `signature`, `inDegree`, `edges` — nur wenn der Graph den
  Pfad kennt; sonst reiner Texttreffer.
- Neues Tool nur, wenn es die Fusion wirklich trägt (z. B. `graph_context`:
  Treffer → Caller/Blast in einem Aufruf). Vorhandene loomux-MCP-Tools nicht
  duplizieren.
- Fehlerverhalten: fehlt `loomux` oder der Graph, Textsuche als Fallback und
  ein einmaliger Hinweis; niemals ein leerer Treffer als Antwort.
- Privacy: Pfade unter `[privacy] never` müssen auch auf dem CLI-Weg
  ausgefiltert werden (MCP-Tools tun es bereits).

## Offene Fragen (vom Menschen zu entscheiden)

1. **Aktualität:** Index nur beim Graph-Refresh neu bauen (Empfehlung, passt
   zum Freshness-Modell) oder während einer Session live nachführen?
2. **Ort des Plugin-Codes:** eigenes Repo/Fork von `dsh-tgrep`, oder ein neues
   Paket `dsh-loomux-graph`? Das loomux-Repo enthält nur die Go-Seite.
3. **tgrep-Fallback:** soll die tgrep-Binary optional bleiben (für Sprachen
   ohne Graph-Extraktor) oder fliegt sie ganz raus?

## Zu liefern (Aufgabe für `claude-hb`)

1. Design: `docs/.superpowers/specs/2026-11-03-tgrep-loomux-fusion-design.md`
   (Deutsch) — Intent, Ziele/Nicht-Ziele/Anti-Ziele, Architektur, Datenmodell
   und Dateiformat, Schnittstellen (Go-Pakete, CLI-Flags, JSON, Plugin-Tool),
   Fehlerfälle, Privacy/Sicherheit, Performance-Ziele und Messplan,
   Teststrategie, Rollout und Rückbau, offene Fragen.
2. Plan: `docs/.superpowers/plans/2026-11-03-tgrep-loomux-fusion-plan.md`
   (Deutsch) — TDD-Schritte in Reihenfolge, je Schritt Dateien, Tests,
   Akzeptanzkriterien, Abhängigkeiten, Risiken; getrennt in loomux-Go-Teil
   und Plugin-Teil.

Stil: Wie die vorhandenen Papiere unter `docs/.superpowers/specs/` und
`docs/.superpowers/plans/` (z. B. `2026-10-24-regeln-als-pruefungen-design.md`).
Keine Commits, keine Pushes, nur die beiden Dateien schreiben.
