# Paritätsliste Stufe 1b-1

**Quelle:** ultra-brain `loomux-1a-source` (`3cc72d2`): Python-Referenz `src/brain/` unter Python 3.14.7 (`brain-mcp`), qmd 2.8.3 für den Spike vom 2026-09-15.
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor Stufe 1b-1 als fertig gilt. „Alt" ist die Python-Referenz, „Neu" ist `loomux brain`.
**Stand:** Alle 54 Zeilen sind am 2026-09-16 im Durchgang mit dem Nutzer
freigegeben. Drei davon tragen einen Nachtrag: die Prüfungen aus den Zeilen
18, 20 und 22 sind am 2026-09-16 nachgerüstet, siehe
`../specs/2026-09-16-loomux-registry-manifest-pruefungen-design.md`.

| Fall / Bereich | Alt | Neu | Begründung | Freigabe |
|---|---|---|---|---|
| `search --profile fast`, Rangfolge (Fall `brain-search/fast`) | `qmd vsearch`: ein LLM erweitert die Anfrage, Suche je Variante, Zusammenführung per Maximum, Schnitt unter 0,3 | qmd-MCP-Daemon mit `searches:[{type:"vec"}]`, `rerank:false`, ohne Erweiterung; Score 1/Rang | Spec-Zeile 1, Task 8: Spike 2026-09-15, `qmd query "vec: <q>" --no-rerank` liefert die MCP-Liste; der Fall vergleicht nur den Exit | freigegeben 2026-09-16 |
| `search --profile keyword` (Fall `brain-search/keyword`) | BM25-Score, erste Fundstelle | gleiche Reihenfolge, Score 1/Rang, Zeile des besten Abschnitts | Spec-Zeile 2, Task 8; der Fall vergleicht nur den Exit | freigegeben 2026-09-16 |
| Zustand und Manifestnamen, befristet | Registry, Artefakte schreibgeschützter Bereiche und Stempel unter `BRAIN_STATE_DIR` bzw. `--state-dir`; Manifest `.ultra-brain/config.toml`, sonst `.brain.toml` | Registry unter `LOOMUX_STATE_DIR` (`%LOCALAPPDATA%\loomux`); Artefakte und Stempel unter `LOOMUX_LEGACY_BRAIN_DIR` (sonst `%LOCALAPPDATA%\brain`) bis Stufe 3; `.loomux/config.toml` vor beiden Altnamen bis Stufe 4 — steht sie mit einer `[area]`-Tabelle neben einem Altmanifest, gilt sie; ohne `[area]` deklariert sie für `brain` nichts, und das Altmanifest gilt | Spec-Zeile 3, befristete Ausnahme; Task 1 (1, 2) | freigegeben 2026-09-16 |
| Aufzeichnungen der Stufe 1b-1 | Python schreibt in eine Pipe `\r\n`, ohne `PYTHONUTF8` in cp1252 | der Rekorder faltet `\r\n` im stdout zu `\n` und setzt `PYTHONUTF8=1` in jedem aufgezeichneten Prozess; aufgezeichnet mit `uv run --no-sync` und `PYTHONDONTWRITEBYTECODE=1` | Spec-Zeile 4, Task 11 (1): ein Unterschied der Aufzeichnung, nicht des Verhaltens. Die Faltung gilt ab Task 11 für jede Aufzeichnung; die Zeile „CRLF in Aufzeichnungen" der Stufe 1a beschreibt den Rekorder davor | freigegeben 2026-09-16 |
| Aufwärm-Hinweis | kein Hinweis | `note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.` auf stderr, einmal je erzeugter `ConnectFunc` -- also einmal je Port in einem Lauf --, wenn loomux den Daemon selbst gestartet hat | Spec-Zeile 5, Task 8 (3) | freigegeben 2026-09-16 |
| Backbone beim Daemon-Start | `qmd_mcp.py` schreibt `{**os.environ, **env}` und überschreibt `QMD_LLAMA_GPU`/`QMD_FORCE_CPU` des Nutzers | eine vom Nutzer gesetzte Variable, auch leer, gewinnt; sonst CUDA-Vorgabe ohne Variable | Spec-Zeile 6, Task 8 (4) | freigegeben 2026-09-16 |
| CLI-Port (`qmd ls`, `qmd status`), Umgebung | `_default_runner` pinnt `QMD_LLAMA_GPU=vulkan` | Umgebung des Nutzers, CUDA-Vorgabe, nichts angehängt | Task 8 (5) | freigegeben 2026-09-16 |
| Wartefrist auf den Daemon, Wortlaut | `… within 60s` | `no qmd daemon answered on http://localhost:8765/mcp within 1m0s` | Task 8 (6) | freigegeben 2026-09-16 |
| Parser der CLI-Suche (`QmdPort.Search`, in 1b-1 ungenutzt) | `expected a list of hits, found dict`, `the hit is missing its 'file': {…}`; verlangt ganzzahliges `line` und eine Zahl als `score` | ub-Go-Wortlaute `expected a list of hits`, `the hit is missing 'file'`; fehlendes `line`/`score` wird 0 | Task 8 (7) | freigegeben 2026-09-16 |
| MCP-Antwortübersetzung | `qmd_mcp.py`: `col/` ohne Rest behält den ganzen Pfad; negative Zeilen bleiben | `col/` wird (`col`, `""`); eine Zeile unter 1 wird 1 | Task 8 (8); im Korpus unsichtbar, die Python-CLI nutzt den MCP-Port nie | freigegeben 2026-09-16 |
| Unlesbarer npm-Shim | `cannot run qmd: [Errno …] …` | `cannot read {shim}: …` | Task 8 (9) | freigegeben 2026-09-16 |
| `Pending`-Zeile von `qmd status` | Pythons `re` liest `\s` und `\d` als Unicode-Klassen | `pendingLinePattern` mit ASCII-`\s`/`\d` (ub-Go) | Task 8 (11) | freigegeben 2026-09-16 |
| Registerpfad mit `..` in Fehlermeldungen | `Path` behält `..` | `filepath.Join` löst `..` auf | Task 8 (12) | freigegeben 2026-09-16 |
| `n < 1` an `ExecuteSearch` | `ordered[:n]` schneidet von hinten | leere Liste | Task 8 (13); die CLI verweigert `-n` unter 1 vorher (Fall `brain-search/count-below-one`) | freigegeben 2026-09-16 |
| Betriebssystemfehler im Wortlaut (Fälle `brain-catalog/missing-registry`, `brain-search/missing-registry`, `brain-catalog/missing-index`, `brain-read/missing-file`) | `[Errno 2] No such file or directory: '<pfad>'` bzw. `[Errno 13] Permission denied: '<pfad>'`, Pfad in `repr` | Go-Wortlaut von `os.ReadFile` (`open <pfad>: <Systemtext>` bzw. `read <pfad>: <Systemtext>`), bei der Registry `open <pfad>: …` (seit registry-manifest-pruefungen.md) | Task 3 (3), Task 4 (4), Task 5 (8), Task 6 (2, 3), Task 7 (1), Task 8 (10), Task 9. Betroffen: fehlende Registry, fehlendes `index.md`, fehlende Datei bei `read`, `read` von `.` oder leerem Pfad (Python liest das Verzeichnis), `_identities.tsv`, `graph.json` oder Stempel, die existieren und nicht lesbar sind. Beide Exit 1. **Der Systemtext ist sprachabhängig und steht hier als Platzhalter, nicht als Sollwert** (Ruling R13 und R14): hier in der englischen Windows-Fassung notiert, `read <pfad>: Incorrect function.` für ein Verzeichnis und `open <pfad>: The system cannot find the file specified.` für eine fehlende Datei; gemessen wurde auf dieser Maschine (Windows 11, deutsches System) derselbe Fehler in deutscher Sprache. Kein Test nagelt einen dieser Sätze fest. **Plattformabhängig für das unlesbare Register** (Ruling R11): Python meldet unter Windows `PermissionError: [Errno 13] Permission denied: '<pfad>'` (Task 12 gemessen 2026-09-16), unter POSIX `IsADirectoryError: [Errno 21]` (Angabe des Controllers, auf dieser Maschine nicht messbar) | freigegeben 2026-09-16 |
| Fehlendes Manifest eines registrierten Bereichs (Fälle `brain-catalog/missing-manifest`, `brain-read/missing-manifest`, `brain-status/missing-manifest`) | `[Errno 2] No such file or directory: '<bereich>\\.brain.toml'` | `<bereich>: no manifest found (.loomux\config.toml, .ultra-brain\config.toml, .brain.toml)` | Task 3 (1), Wortlaut aus Task 1; beide Exit 1 und brechen den ganzen Aufruf ab | freigegeben 2026-09-16 |
| `[layout] review` ohne Pfadteile (`.`, `./`, `./.`) | `ValueError`, von `cli.main` nicht gefangen: Traceback, Exit 1 | `error: [layout] review must not resolve to the area root, found '.'`, Exit 1 | Task 3 (2) | freigegeben 2026-09-16 |
| Registry-Prüfungen | `RegistryError` bei doppeltem Scope, fehlendem `scope`/`path`, Scope ohne brauchbare Zeichen, gleichem Zustandsverzeichnis zweier Scopes, zwei `signpost` | `config.ReadRegistry` überspringt Einträge ohne `scope`/`path` und prüft den Rest nicht; bei doppeltem Scope liefert `VisibleAreas` beide, `Single` den ersten | Task 3 (4) | freigegeben 2026-09-16 (Nachtrag: Registry- und Manifestprüfungen nachrüsten; nachgerüstet, siehe registry-manifest-pruefungen.md) |
| Manifest mit BOM | `not valid TOML: Invalid statement (at line 1, column 1)` | gelesen (`loomux brain catalog --scope x` druckt die `index.md`) | Task 1 (3); beide Seiten am 2026-09-16 gemessen (Ruling R9) | freigegeben 2026-09-16 |
| `[area] scope` als Nicht-Zeichenkette (`scope = 3`, Fall `brain-catalog/manifest-wrong-type`) | `[area] scope is required and must be a non-empty string` | `not valid TOML: toml: line 2 (last key "area.scope"): incompatible types: TOML value has type int64; destination has type string` | Task 1 (4) | freigegeben 2026-09-16 (Nachtrag: Registry- und Manifestprüfungen nachrüsten; nachgerüstet, siehe registry-manifest-pruefungen.md) |
| Wortlaut nach `not valid TOML: ` bei einem kaputten Manifest oder einer kaputten Registry (Fall `brain-catalog/broken-registry`) | der von `tomllib`, für `[[area` gemessen `Expected ']]' at the end of an array declaration (at line 1, column 7)` | der von BurntSushi toml v1.6.0, für `[[area` gemessen `toml: line 2: expected '.' or ']' to end table name, but got '\n' instead` | Task 1 (5); die Registry liest `config.ReadRegistry` mit derselben Form `<pfad>: not valid TOML: <grund>` (`internal/config/registry.go:106`); beide Exit 1. Beide Wortlaute am 2026-09-16 gemessen (Ruling R9), der loomux-Wortlaut am Manifest und an der Registry gleich; nicht sprachabhängig | freigegeben 2026-09-16 |
| Weitere Prüfungen von `read_manifest` | `[wiki] types` als Liste von Zeichenketten, `[wiki] untouched_days` ≥ 1, `[maintenance] on_merge`/`branch`, `[model]`; Glob-Listen `must be an array of strings` bzw. `contains a non-string` | nicht geprüft; Glob-Listen mit falschem Typ scheitern als TOML-Fehler, gemessen `not valid TOML: toml: line 5 (last key "privacy.never"): incompatible types: TOML value has type string; destination has type slice` und `… (last key "index.include"): … has type int64; destination has type string` | Task 1 (7); die Python-Wortlaute `[privacy] never must be an array of strings` und `[index] include contains a non-string: 3` am 2026-09-16 gemessen (Ruling R9) | freigegeben 2026-09-16 (Nachtrag: Registry- und Manifestprüfungen nachrüsten; nachgerüstet, siehe registry-manifest-pruefungen.md; auch der Wiki-Lint hängt nicht am Manifest: er fragt eine eigene Typliste statt `config.Manifest.KnowsType`, das die Herkunftstypen und `[wiki] types` bereits kennt, und meldet darum für jedes umgezogene Bündel mit eigenen Typen Warnungen) |
| Wortlaut der Modusmeldung | `found 'bogus'` (`repr`) | `found "bogus"` (Go-`%q`) | Task 1 (8): `config` darf `brain/pytext` nicht importieren | freigegeben 2026-09-16 |
| `.brain.toml` als Verzeichnis, kein anderer Name | beim Lesen `PermissionError: [Errno 13] Permission denied: '<pfad>'` (Windows; unter POSIX wäre es `IsADirectoryError: [Errno 21]`, hier nicht messbar) | `no manifest found (.loomux\config.toml, .ultra-brain\config.toml, .brain.toml)` | Task 1 (9); beide Seiten am 2026-09-16 unter Windows gemessen (Ruling R9, Plattformhinweis wie R11) | freigegeben 2026-09-16 |
| Leeres `XDG_STATE_HOME` (POSIX) | relativer Pfad `brain` | `home/.local/state/brain` | Task 1 (2) | freigegeben 2026-09-16 |
| Ungültiges UTF-8 in einer gelesenen Datei | Traceback (`UnicodeDecodeError`) | `error: {pfad}: not valid UTF-8` für `_identities.tsv`, `graph.json`, `index.md` und Dokumente; beim Manifest gemessen `not valid TOML: toml: line 2 (last key "area.scope"): invalid UTF-8 byte: 0xff` | Task 2 (3), Task 4 (3), Task 5 (9), Task 1 (6); die Manifestseite beider Sprachen am 2026-09-16 gemessen (Ruling R9): Python endet mit `UnicodeDecodeError: 'utf-8' codec can't decode byte 0xff in position 16: invalid start byte`, ungefangen in `read_manifest`; Task 6 und 7 führen die Zeile mit. Der Stempel ist ausgenommen: `read_last_run` fängt den `UnicodeDecodeError` (eine `ValueError`), und `ReadLastRun` antwortet ebenso „kein Stempel" (Task 8) | freigegeben 2026-09-16 |
| `repr` von Zeichen, die Unicode 17 neu vergibt (4803 Codepunkte, 47 Bereiche, darunter U+088F) | `\uXXXX`/`\UXXXXXXXX` (Python 3.14 kennt Unicode 16) | unverändert | Task 2 (1) | freigegeben 2026-09-16 |
| `casefold` von 28 Unicode-17-Buchstaben (U+A7CE, U+A7D2, U+A7D4, U+16EA0…U+16EB8) | nicht gefaltet | gefaltet; betrifft `[privacy] never`-Globs mit diesen Zeichen | Task 2 (2) | freigegeben 2026-09-16 |
| `Repr` eines Bytes, das kein UTF-8 ist | — (Python kennt keinen solchen `str`) | `\xhh` | Task 2 (4); nur über Argumente oder Dateinamen erreichbar, die nicht aus `ReadText` stammen | freigegeben 2026-09-16 |
| Stempel-Formen außerhalb der gemessenen Teilmenge | `datetime.fromisoformat` liest 14 Formen zonenbehaftet: kleines `t` oder ein anderes Trennzeichen als `T`/Leerzeichen, `HH:MM` und `HH`, Grundform `20000101T000000+0000`, Offset `+0200` und `+02`, Bruch im Offset, Wochendatum `2000-W01-1`, `24:00:00`, Offset-Minute oder -Sekunde 60, Leerzeichen vor dem Offset | „kein Stempel": `status` meldet `last reconcile: never; …`, `search` keinen Stempelbefund | Task 2 (5) | freigegeben 2026-09-16 |
| `IsoFormat` mit Nanosekunden | — | schneidet unter der Mikrosekunde ab | Task 2 (6); über einen Stempel nicht erreichbar | freigegeben 2026-09-16 |
| Revision aus Nicht-ASCII-Ziffern | `٣` wird 3; `²` endet im Traceback (`ValueError`) | `{pfad}: line {n}: revision '…' is not a number`, Exit 1 | Task 4 (1) | freigegeben 2026-09-16 |
| Revision jenseits von `int` (`99999999999999999999`) | gelesen | verweigert mit demselben Wortlaut | Task 4 (2) | freigegeben 2026-09-16 |
| Wahrheitswert als Zählwert in `links` (`"total": true`) | angenommen (`bool` ist `int`) | `the link counts are not numbers` | Task 5 (1) | freigegeben 2026-09-16 |
| Nicht-Zeichenketten in `from`/`to` | per `str()` gewandelt (`1` → `1`, `null` → `None`) | `{pfad}: json: cannot unmarshal … of type string`; `null` wird leere Zeichenkette | Task 5 (2) | freigegeben 2026-09-16 |
| Wortlaut des JSON-Fehlers in Klammern | `json.JSONDecodeError` (`Expecting value: line 1 column 1 (char 0)`, `Extra data: …`) | `encoding/json` (`invalid character 'o' in literal null (expecting 'u')`) | Task 5 (3) | freigegeben 2026-09-16 |
| `NaN`, `Infinity`, `-Infinity` in `graph.json` | gelesen; als Zählwert `the link counts are not numbers` | `graph is not valid JSON (invalid character 'N' …)` | Task 5 (4) | freigegeben 2026-09-16 |
| Wurzelwert von `graph.json`, der kein Objekt ist | `[]` und `"x"` wie loomux; `"edges"` → `graph is missing links`; `["edges","links"]`, eine Zahl oder `null` → Traceback (`TypeError`) | immer `graph is missing edges, links` | Task 5 (5) | freigegeben 2026-09-16 |
| Ganzzahl jenseits von `int` als Zählwert | angenommen | `{pfad}: json: cannot unmarshal number 99999999999999999999 into Go struct field … of type int` | Task 5 (6) | freigegeben 2026-09-16 |
| `nodes` oder `scope` von falschem Typ | nie gelesen | `{pfad}: json: cannot unmarshal …` (ub-Verhalten) | Task 5 (7) | freigegeben 2026-09-16 |
| `read --section ""` | sucht eine leere Überschrift (`_section('# \nx', '')` → `'# \nx\n'`) | gibt die ganze Datei aus | Task 6 (1) | freigegeben 2026-09-16 |
| Abkürzungen langer Optionen | `--prof` wird `--profile`; `--s` → `ambiguous option: --s could match --scope, --state-dir` | nur ganze Namen: `unrecognized arguments: --prof x` bzw. `--s x`, Exit 2 | Task 10 (1) | freigegeben 2026-09-16 |
| `--state-dir` | an allen fünf Befehlen | `unrecognized arguments: --state-dir …`, Exit 2; der Zustand kommt aus `LOOMUX_STATE_DIR` und `LOOMUX_LEGACY_BRAIN_DIR` | Task 10 (2) | freigegeben 2026-09-16 |
| `-h`/`--help` | Hilfe auf stdout, Exit 0, auch angeklebt (`search q -hx`) | keine Hilfe; das Wort endet bei `unrecognized arguments` oder hinter dem Fehler über fehlende Argumente, Exit 2 | Task 10 (3) | freigegeben 2026-09-16 |
| `-n` mit Nicht-ASCII-Ziffern (`٣`, `３`) | 3 | `invalid _at_least_one value`, Exit 2 | Task 10 (4) | freigegeben 2026-09-16 |
| `-n` über `math.MaxInt` (`99999999999999999999999`) | die Zahl bleibt | auf `math.MaxInt` gekappt | Task 10 (5) | freigegeben 2026-09-16 |
| Usage-Zeilen | nach Terminalbreite umgebrochen, mit `[-h]` und `[--state-dir STATE_DIR]`; die oberste Usage und `argument command: invalid choice` zählen 23 Unterbefehle | eine Zeile ohne beide Optionen, fünf Unterbefehle | Task 10 (6); die Fälle `brain-search/count-below-one`, `brain-search/invalid-profile`, `brain-search/missing-query`, `brain-catalog/invalid-channel`, `brain-read/missing-scope`, `brain-neighbors/missing-scope`, `brain-status/invalid-channel` vergleichen Exit und leeres stdout | freigegeben 2026-09-16 |
| Wörter vor dem Unterbefehl | `brain-mcp -- search q` angenommen; `brain-mcp -x search q` → `unrecognized arguments: -x` | das erste Wort ist der Unterbefehl: `argument command: invalid choice: '--'` bzw. `'-x'`, Exit 2 | Task 10 (7) | freigegeben 2026-09-16 |
| Brain-Daemon | ein laufender brain-Daemon beantwortet die fünf Befehle in seinem eigenen Format (`cli.py` `_through_daemon`) | fragt nie einen brain-Daemon | Task 10 (8); aufgezeichnet ohne Daemon, dessen Pipe-Name vom Zustandsverzeichnis abhängt | freigegeben 2026-09-16 |
| qmd in den Aufzeichnungen | echtes qmd 2.8.3 | `internal/dev/fakeqmd`: `qmd ls` mit fester Größe und Zeit, `qmd status` nur `Documents` und die `Pending`-Zeile, Treffer synthetisch in Fixture-Reihenfolge; eine scheiternde Suche ist die Störung `search_error` (CLI Exit 1 mit dem Wert auf stderr, MCP JSON-RPC-Fehler `-32603`), kein gemessener qmd-Wortlaut | Task 11 (2); kein Modell und keine GPU in Aufzeichnung und Tor; `search_error` trägt den Fall `brain-search/engine-fails` | freigegeben 2026-09-16 |
| Snippets auf beiden Wegen | CLI `--json` liefert den Snippet ohne Zeilennummern | echtes qmd 2.8.3 stellt im MCP-`query` jeder Snippet-Zeile `N: ` voran (`dist/mcp/server.js:301`), bei gleichem Ausschnittfenster (`server.js:293` und `cli/qmd.js:2140`); `translateReply` entfernt genau dieses Präfix, stdout ist darum gleich | Task 11 (3), berichtigt nach F69; Task 8 (Ruling R4). Der Fake nummeriert auf dem MCP-Weg wie qmd; `TestLoomuxsPortsReadTheFake` und die `full`-Fälle des Korpus belegen die Entfernung, der Live-Vergleich von Task 15 bestätigt sie am echten qmd | freigegeben 2026-09-16 |
| Befunde von `search` und alle Fehlerwortlaute | auf stderr | auf stderr, eigener Wortlaut, wo die Zeilen oben es sagen | Spec, Vergleichsklassen: der Korpus vergleicht stderr nie; die exakten Zeichenketten stehen in den Unit-Tests der Tasks 3–10 | freigegeben 2026-09-16 |
| Treffer aus einer nicht gefragten Sammlung | `_split` schlägt auf `collections[0]` zurück, wenn der Pfad mit keiner gefragten Sammlung beginnt | gleich: `splitPath` schlägt ebenso auf die erste gefragte Sammlung zurück, der Treffer wird also unter diesem Bereich geführt und gegen dessen `never`-Globs geprüft | Ruling R25, Abschlussdurchsicht F2: kein Leck, solange qmd den `collections`-Filter einhält; der `!known`-Zweig in `assemble` ist darum auf dem MCP-Weg unerreichbar und nur für Ports da, die nicht umetikettieren | freigegeben 2026-09-16 |
| Live-Vergleich `status` | `brain-mcp` liest `%LOCALAPPDATA%\brain\registry.toml` (10 Bereiche) | `loomux brain` liest `%LOCALAPPDATA%\loomux\registry.toml` (dieselben 10 plus `project/loomux`) und meldet für diesen `` never indexed; run `brain reindex` `` | eine Registry für Schranke und Datenbefehle (Spec, Entscheidungen „Zustand“) | freigegeben 2026-09-16 |

## Was die Fälle der Stufe 1b-1 decken

`testdata/cases/1b-1-source/` hält die Aufzeichnungen von `brain-mcp` gegen den
Fake-qmd (Beweis, nie nachbearbeitet), `testdata/cases/1b-1/` die Übersetzung,
an der loomux gemessen wird. 71 Fälle über 23 Welten, gefahren von
`internal/cli/cases_1b1_test.go`: `search` 15 (davon 2 nur Exit), `catalog` 14,
`read` 14, `neighbors` 9, `status` 19 mit einer Welt je Zeilenart L1a–L8b.
Von den 71 Fällen vergleichen nur 34 den Text, den loomux schreibt: 35 prüfen
den Exit-Code und ein leeres stdout -- jede Verweigerung, jeder Usage-,
Registry- und Manifestfehler --, und die zwei `message`-Fälle
(`brain-search/fast`, `brain-search/keyword`) prüfen den Exit-Code allein, denn
ihr aufgezeichnetes stdout wird nie verglichen.
Jeder Befehl hat seinen Erfolg, jede Verweigerung, einen Usage-Fehler und die
Registry- und Manifestfehler; dazu die kaputte Registry, das Manifest mit
falschem Typ und die scheiternde Suche.
Angepasst wurde keine Erwartung.

## Mutationsrunde

Gelaufen am 2026-09-16 mit `bin/loomux.exe dev mutants <paket>` (Task 13),
Commit `d50c2cd`, 8 Arbeiter, Protokolle unter `$TEMP/mutants-1b1/`. Ein
Zeitüberlauf zählt als getötet. Baseline von `internal/hooks` während seiner
Runde: `real 0m22,271s`, also unter der Grenze von 60 s -- ein Vierer-Lauf war
nicht nötig. Mutantenzahlen gleich `_mutants_of` aus `go_mutants.py`; die
Gegenprobe über `internal/config` mit einem Arbeiter urteilt über jeden
Mutanten gleich wie der Achter-Lauf (`diff` leer).

Die festen Zahlen der Stufe 1a sind neu geeicht (Ruling R7):
`internal/brain/guard` zählt 771 statt 712, seit die Schreibschranke
`worktree.go` mitgebracht hat, und `internal/config` 173 statt 156, seit Task 1
`legacy.go` angelegt hat. `internal/cases` (231), `internal/brain/wiki` (200)
und `internal/hooks` (706) stehen unverändert.

| Paket | Mutanten | getötet | überlebt | nicht kompiliert | kein Mutant |
|---|---:|---:|---:|---:|---:|
| internal/config | 173 | 137 | 2 | 34 | 0 |
| internal/cases | 231 | 181 | 21 | 29 | 0 |
| internal/brain/wiki | 200 | 137 | 24 | 39 | 0 |
| internal/brain/pytext | 225 | 177 | 23 | 25 | 0 |
| internal/brain/identity | 55 | 41 | 7 | 7 | 0 |
| internal/brain/graph | 89 | 62 | 1 | 26 | 0 |
| internal/brain/reader | 45 | 38 | 1 | 6 | 0 |
| internal/brain/catalog | 8 | 7 | 1 | 0 | 0 |
| internal/brain/privacy | 158 | 126 | 14 | 18 | 0 |
| internal/brain/status | 96 | 84 | 0 | 12 | 0 |
| internal/brain/search | 444 | 314 | 60 | 70 | 0 |
| internal/brain/guard | 771 | 549 | 17 | 205 | 0 |
| internal/hooks | 706 | 475 | 109 | 122 | 0 |

Zusammen 3.201 Mutanten, 2.328 getötet, 280 überlebt, 593 keine Mutanten.

### Erledigte Überlebende

Neun Pakete sind im ersten Durchgang abgearbeitet: 53 Überlebende haben einen nachgereichten Test,
der sie tötet, 27 eine Begründung, warum der Code den Unterschied nicht sehen
kann. Die Nachprüfung ist je Paket eine vollständige zweite Runde statt
`--only`/`--family`; ihre Protokolle liegen unter `$TEMP/mutants-1b1/recheck/`
und belegen zugleich, dass kein Mutant neu überlebt.

| Datei:Zeile | Familie | war -> jetzt | Erledigung |
|---|---|---|---|
| internal/config/manifest.go:214 | a3 | `<` -> `<=` | Kein Unterschied: die Lanes entstehen aus den Schlüsseln der Tabelle `map[string]string` (manifest.go:207-211), und Schlüssel einer Map sind verschieden; bei verschiedenen Namen antworten beide Formen gleich |
| internal/config/manifest.go:337 | a3 | `size > 0` -> `size >= 0` | Kein Unterschied: `size == 0` liefert `utf8.DecodeRuneInString` nur für den leeren String, und dann ist `len(value) >= size+2` die Aussage `0 >= 2` und falsch -- die Bedingung ist in beiden Formen falsch |
| internal/cases/case.go:38 | a1 | `err != nil` -> `false` | Test `TestAMissingCmdIsNotAnEmptyOne`, danach `killed` |
| internal/cases/case.go:47 | a1 | `err != nil` -> `false` | Test `TestAMissingExitIsNotAnUnreadableOne`, danach `killed` |
| internal/cases/case.go:73 | a2 | beide Wörter -> nur `message` | Test `TestDataIsAComparisonTheCorpusWrites`, danach `killed` |
| internal/cases/case.go:107 | a1 | `!d.IsDir()` -> `false` | Kein Unterschied unter erreichbaren Eingaben: für einen Pfad, der kein Verzeichnis ist, kann `<pfad>/cmd` nicht existieren, `os.Stat` scheitert, und beide Formen antworten `nil`. Ein Symlink auf ein Fallverzeichnis träfe den Unterschied; ihn anzulegen braucht unter Windows ein Recht, das die Suite nicht hat |
| internal/cases/case.go:127 | a1, a3, a4 | `Verb != Verb` -> `true`, `false`, `!(...)`, `==` | Test `TestCasesAreOrderedByVerbAndThenByName` (vier Mutanten), danach `killed` |
| internal/cases/case.go:128 | a3 | `<` -> `<=` | Kein Unterschied: Zeile 127 lässt diese Zeile nur für verschiedene Verben laufen, und für verschiedene Werte sind `<` und `<=` dasselbe |
| internal/cases/case.go:130 | a3 | `<` -> `<=` | Kein Unterschied, der zu prüfen wäre: gleiche Namen unter gleichem Verb gibt es nur unter verschiedenen Elternpfaden, und für gleiche Schlüssel sagt `sort.Slice` bei keiner der beiden Formen eine Reihenfolge zu |
| internal/cases/runner.go:62 | a1 | `err != nil` -> `false` | Kein Unterschied unter erreichbaren Eingaben: der Arm trägt seit 1a `//coverage:exempt` (runner.go:52) mit dem Grund, dass `filepath.Rel` einen von `WalkDir` unter `src` gefundenen Pfad braucht, der nicht unter `src` liegt |
| internal/cases/runner.go:65 | a1 | `rel == "."` -> `false` | Kein Unterschied: für die Wurzel ist `rel` gleich `.`, `filepath.Join(dst, ".")` ist `dst`, und `os.MkdirAll` auf das eben angelegte Verzeichnis antwortet `nil` |
| internal/cases/runner.go:119 | a1 | `os.IsNotExist(err)` -> `true` | Kein Unterschied unter erreichbaren Eingaben: der verbleibende Fehlerarm trägt `//coverage:exempt` (runner.go:113) mit dem Grund, dass er ein Verzeichnis braucht, das das Betriebssystem zu listen verweigert, während sein Elternverzeichnis liest |
| internal/cases/runner.go:128 | a1 | `err != nil` -> `false` | Kein Unterschied unter erreichbaren Eingaben: derselbe `//coverage:exempt` (runner.go:113) nennt den `filepath.Rel`-Arm |
| internal/cases/runner.go:160 | a2 | `c == Backslash && !inSingle` -> `c == Backslash` | Test `TestABackslashInsideSingleQuotesIsACharacter`, danach `killed` |
| internal/cases/runner.go:188 und :200 | a1, a3 | `cur.Len() > 0 oder hadQuotes` -> `true`, `>= 0` | Test `TestRunsOfSeparatorsOpenNoEmptyTokens` (vier Mutanten), danach `killed` |
| internal/cases/runner.go:209 | a1 | `err != nil` -> `false` | Test `TestTheTempDirectoryFailureIsTheOneReported`, danach `killed` |
| internal/cases/runner.go:221 | a1 | `err != nil` -> `false` | Test `TestACommandThatCannotBeSplitIsNotAnEmptyCommand`, danach `killed` |
| internal/brain/wiki/gate.go:26 | a1 | `HasSuffix(...)` -> `true` | Kein Unterschied: ohne passendes Suffix lässt `strings.TrimSuffix` den Namen stehen, der Kandidat ist damit genau `parent/<name>_wiki` -- der Rückfall aus Zeile 35, den der echte Code sowieso als nächstes prüft |
| internal/brain/wiki/gate.go:69 | a3 | `len(l) > 3` -> `>= 3` | Kein Unterschied unter erreichbaren Eingaben: eine Zeile von `git status --porcelain` ist zwei Statusspalten, ein Leerzeichen und mindestens ein Zeichen Pfad; eine Zeile von genau drei Zeichen gibt git nicht aus, und die leere letzte Zeile des Splits scheitert an beiden Formen |
| internal/brain/wiki/gate.go:80 | a1 | `wikiPath == ""` -> `false` | Test `TestAProjectWithoutAWikiIsNotGated`, danach `killed` |
| internal/brain/wiki/lint.go:79 | a1 | `DeclaredConflicts == nil` -> `true` | Test `TestAPageThatDeclaresTheWrongNumberOfConflictsIsReported`, danach `killed` |
| internal/brain/wiki/lint.go:108 | a1 und a2 (`d.IsDir()`) | ganze Bedingung -> `false`, bzw. -> `d.IsDir()` | Test `TestABundleThatIsNotThereIsNoBundle` (zwei Mutanten), danach `killed`: `WalkDir` reicht für eine unlesbare Wurzel `nil` als Eintrag herein, und wer den Fehler nicht zuerst liest, fragt diesen nil-Eintrag |
| internal/brain/wiki/lint.go:108 | a2 (`err != nil`) | ganze Bedingung -> `err != nil` | Kein Unterschied: ein Verzeichnis, das den weggefallenen `d.IsDir()` übersteht, muss auf `.md` enden, und `ReadPage` scheitert an einem Verzeichnis -- der Rückruf antwortet so oder so `nil` |
| internal/brain/wiki/lint.go:123 | a1 | `err != nil` -> `false` | Kein Unterschied unter erreichbaren Eingaben: `ReadPage` scheitert nur, wenn `os.ReadFile` scheitert, und der Pfad kommt eben von `WalkDir`; Verzeichnisse sind in Zeile 108 schon heraus |
| internal/brain/wiki/lint.go:144 | a1 | `judged` -> `true` | Kein Unterschied: `LintSingleFile` beginnt selbst mit `if IsScaffoldFile(filePath) { return nil, nil }` (lint.go:36-38), der Aufruf für eine Gerüstseite trägt also nichts bei |
| internal/brain/wiki/lint.go:158 | a1 | `targetPath == ""` -> `false` | Kein Unterschied: ein leeres Ziel löst sich auf das eigene Verzeichnis der Seite auf, das es gibt und das kein `page.Relative` ist -- weder ein Befund noch die Waisenzählung ändern sich |
| internal/brain/wiki/lint.go:163 | a1, a4 | `HasPrefix(targetPath, "/")` -> `true`, `false`, `!(...)` | Test `TestAnAbsoluteLinkResolvesFromTheWikiRootNotFromThePage` (drei Mutanten), danach `killed` |
| internal/brain/wiki/lint.go:201 und :204 | a1 | `IsScaffoldFile(...)` -> `true`, `inbound == 0` -> `false` | Test `TestAPageNobodyLinksIsAnOrphan` (zwei Mutanten), danach `killed` |
| internal/brain/wiki/parse.go:119 | a1 | `node.Kind != MappingNode` -> `false` | Kein Unterschied: der Knotenbaum wird nur im `else`-Zweig gelesen, in dem das getypte `yaml.Unmarshal` (parse.go:89) gelungen ist -- ein Frontmatter, das keine Abbildung ist, kommt dort nie an |
| internal/brain/wiki/parse.go:124 | a3 | `i+1 < len` -> `i+1 <= len` | Kein Unterschied: ein Abbildungsknoten hält Schlüssel und Wert paarweise, `len(node.Content)` ist gerade, und für gerades `i` tritt `i+1 == len` nie ein |
| internal/brain/wiki/parse.go:133 | a1, a2 | vier Klauseln -> `true` und je eine Klausel | Test `TestLinksWithASchemeAreNotWikiLinks` (fünf Mutanten), danach `killed` |
| internal/brain/wiki/parse.go:173 | a3 | `from < len(body)` -> `<=` | Kein Unterschied: für `from == len(body)` ist `body[from:]` leer, der Ausdruck findet nichts, und die Schleife bricht in beiden Formen ab |
| internal/brain/wiki/root.go:20 | a2 | `err == nil && layout != ""` -> `err == nil` | Test `TestAnUnsaidLayoutIsNoLayout`, danach `killed` |
| internal/brain/pytext/isotime.go:62 | a3 | `>= '0'` -> `> '0'` | Test `TestAZeroInsideTheFractionStaysInTheFraction`, danach `killed` |
| internal/brain/pytext/isotime.go:79 | a3 | `seconds > 59` -> `>= 59` | Test `TestAnOffsetSecondOfFiftyNineIsStillAnOffset`, danach `killed` |
| internal/brain/pytext/isotime.go:90 | a3 | `year < 1` -> `<= 1` | Test `TestTheFirstYearOfTheCalendarParses`, danach `killed` |
| internal/brain/pytext/isotime.go:90 | a3 | `month > 12` -> `>= 12` | Test `TestDecemberParses`, danach `killed` |
| internal/brain/pytext/isotime.go:105 | a1, a2 | Ziffernbereich -> `false` und je eine Klausel | Test `TestANonDigitWhereTheShapeWantsADigitIsRefused` (drei Mutanten), danach `killed`. Das schlechte Zeichen steht im Jahr, dem einzigen Feld ohne eigene Bereichsprüfung: in der Sekunde fängt `second > 59` es ein zweites Mal ab und verdeckt die Form |
| internal/brain/pytext/isotime.go:110 | a1 | `s[i] != pattern[i]` -> `false` | Test `TestTheFixedCharactersOfTheShapeMustMatch`, danach `killed` |
| internal/brain/pytext/lines.go:19 | a3 | `i < len(s)` -> `<=` | Test `TestACarriageReturnAtTheVeryEndOpensNoEmptyLine`, danach `killed` |
| internal/brain/pytext/path.go:44 | a2 | drei Klauseln -> nur die letzte | Kein Unterschied: ist `root` nicht leer, ist er der Backslash, und der Block kann ihn nur auf den Backslash setzen; beginnt `drive` nicht mit einem Backslash, hält er keinen und hat damit nie vier oder sechs Teile |
| internal/brain/pytext/path.go:48 | a2 | zwei Klauseln -> ohne `len == 6` | Test `TestASixPartUNCDriveGainsARoot`, danach `killed` |
| internal/brain/pytext/path.go:53 | a2 | drei Klauseln -> `len(tail) > 0` | Test `TestATailThatReadsAsADriveIsPrefixedOnlyWithoutADrive`, danach `killed` |
| internal/brain/pytext/path.go:70 | a1, a2, a3 | `len(p) >= 8 && EqualFold(...)` -> `false`, `len(p) >= 8`, `len(p) > 8` | Tests `TestASixPartUNCDriveGainsARoot`, `TestAPlainUNCPathIsNotReadAsAnExtendedOne` und `TestAnExtendedUNCPrefixWithNothingBehindItIsAllDrive` (drei Mutanten), danach `killed` |
| internal/brain/pytext/path.go:74 | a1 (`true`), a3, a4 | `index < 0` -> `true`, `<= 0`, `!(...)` | Tests `TestAPlainUNCPathIsNotReadAsAnExtendedOne` und `TestAThirdSlashBelongsToTheShareNotToTheSearch` (drei Mutanten), danach `killed` |
| internal/brain/pytext/path.go:74 | a1 (`false`) | `index < 0` -> `false` | Kein Unterschied: ist `index` kleiner null, gibt es hinter dem Anfang keinen weiteren Backslash, die zweite Suche scheitert genauso, und Zeile 79 gibt dasselbe Tripel zurück |
| internal/brain/pytext/path.go:79 | a1, a3 | `index2 < 0` -> `true`, `<= 0` | Tests `TestAPlainUNCPathIsNotReadAsAnExtendedOne` und `TestAnEmptyShareStillEndsTheDrive` (zwei Mutanten), danach `killed` |
| internal/brain/pytext/repr.go:56 | a3 | `r < 0x7f` -> `r <= 0x7f` | Kein Unterschied: `r == 0x7f` fängt schon die Zeile darüber ab (`case r < ' ' \|\| r == 0x7f`), der Wert erreicht diesen Zweig nie |
| internal/brain/pytext/repr.go:58 | a3 | `r <= 0xff` -> `r < 0xff` | Kein Unterschied: der einzige Wert 0xff ist `ÿ`, und `unicode.IsPrint` nimmt ihn, also nimmt ihn schon Zeile 56 |
| internal/brain/identity/identity.go:140 | a1 | `s == ""` -> `false` | Kein Unterschied: über den leeren String läuft die Schleife nicht, `strconv.Atoi("")` scheitert, und beide Formen antworten `(0, false)` |
| internal/brain/identity/identity.go:144 | a2 (`r < '0'`) | zwei Klauseln -> `r < '0'` | Kein Unterschied: jede Rune über `'9'`, die die weggefallene Klausel abwiese, weist auch `strconv.Atoi` ab -- es nimmt nur ASCII-Ziffern mit einem Vorzeichen, und die Vorzeichen stehen unter `'0'` |
| internal/brain/identity/identity.go:144 | a3 | `<= '0'`, `>= '9'` | Test `TestTheOuterDigitsAreRevisionsLikeAnyOther` (zwei Mutanten), danach `killed` |
| internal/brain/identity/identity.go:159 | a1 | `!ok` -> `true` | Test `TestAPathThatStayedIsNotAlsoAPathThatWent`, danach `killed` |
| internal/brain/identity/identity.go:166 | a1 | `!ok` -> `true` | Test `TestAPathThatWasAlreadyThereIsNotFresh`, danach `killed` |
| internal/brain/identity/identity.go:206 | a3 | `<` -> `<=` | Kein Unterschied, der zu prüfen wäre: bei verschiedenen `DocID` antworten beide Formen gleich, bei gleichen sagt `sort.Slice` für keine der beiden eine Reihenfolge zu |
| internal/brain/graph/read.go:112 | a1 | `!ok` -> `false` | Kein Unterschied: der Fehlertext von Zeile 113 ist wortgleich der von Zeile 121, und eine nil-Map gibt keine Schlüssel her -- ein `links`, das keine Tabelle ist, fällt durch und meldet denselben Satz |
| internal/brain/reader/section.go:21 | a1 | `HasPrefix(line, "#")` -> `true` | Test `TestOnlyAHeadingLineCanOpenASection`, danach `killed` |
| internal/brain/catalog/root.go:23 | a3 | `<` -> `<=` | Kein Unterschied: die gerenderte Zeile trägt nur `Scope`, zwei Bereiche mit gleichem Scope rendern also gleich, und bei verschiedenen Scopes sind `<` und `<=` dasselbe |

### Zweiter Durchgang: privacy und search (Ruling R19)

Dieselbe Runde, ein zweites Mal ausgewertet. `internal/brain/privacy` und
`internal/brain/search` sind Code dieser Stufe und darum hier bewertet;
`internal/brain/guard` und `internal/hooks` sind Pakete der Stufe 1a und
bleiben nach Ruling R19 unangetastet.

| Paket | überlebt (erste Runde) | nach den Tests | durch Test getötet | Begründung 4b |
|---|---:|---:|---:|---:|
| internal/brain/privacy | 14 | 8 | 6 | 8 |
| internal/brain/search | 60 | 16 | 44 | 16 |

24 neue Tests: `internal/brain/privacy/mutation_test.go` (5) und
`internal/brain/search/mutation_test.go` (19). Kein Produktionscode geändert,
kein 4c-Befund.

`internal/brain/search` hat zwei Hälften mit zwei Referenzen: `qmd.go` bildet
`src/brain/search/qmd.py` nach und wird dagegen gemessen; der MCP-Weg in
`http.go` und `mcp.go` ist loomux' eigener -- die Referenz spricht mit qmd nur
über die Kommandozeile --, dort sind JSON-RPC 2.0, das SSE-Format und die
eigenen Konstanten die Quelle.

| Datei:Zeile | Familie | war -> jetzt | Erledigung |
|---|---|---|---|
| internal/brain/privacy/containment.go:44 | a3 | `size > 0` -> `size >= 0` | Kein Unterschied: `size == 0` liefert `utf8.DecodeRuneInString` nur für den leeren String, und dann ist `len(p) > size` die Aussage `0 > 0` und falsch |
| internal/brain/privacy/glob.go:66 | a1 | `idx == last` -> `true` | Test `TestATwoStarPartInTheMiddleStillNeedsASeparator`, danach `killed` |
| internal/brain/privacy/glob.go:72 | a1 | `part != ""` -> `true` | Kein Unterschied: `translateSegment("")` läuft über null Runen (glob.go:96) und schreibt nichts, beide Formen hängen also denselben leeren String an |
| internal/brain/privacy/glob.go:109 | a2, a3 | `j < n && pat[j] == '!'` -> ohne die Schranke, `j <= n` | Test `TestAnUnclosedBracketAtTheVeryEndIsALiteral` (zwei Mutanten), danach `killed`: ohne die Schranke liest der Mutant eine Rune hinter das Segment und stürzt ab |
| internal/brain/privacy/glob.go:112 | a1, a2 | `j < n && pat[j] == ']'` -> `true`, `j < n` | Kein Unterschied: der Schritt, den diese Zeile tut, ist genau der erste Schritt der Schleife in Zeile 115 -- steht dort kein `]`, sucht die Schleife von `j` oder von `j+1` aus dasselbe `]`; steht dort eines, geht auch die echte Form darüber; und ab `j >= n` antwortet Zeile 118 für beide |
| internal/brain/privacy/glob.go:145 | a1 | `!Contains(pat[i:j], '-')` -> `false` | Kein Unterschied: ohne `-` im Rumpf antwortet `indexHyphen` sofort -1, die Stückliste bekommt den ganzen Rumpf als einziges Stück, die Zusammenlegung läuft nicht, und das Escaping des Stückwegs (glob.go:176) ist ohne Bindestrich dieselbe Verdopplung der Backslashes wie in Zeile 146 |
| internal/brain/privacy/glob.go:150 | a1 | `pat[i] == '!'` -> `false` | Test `TestTheHyphenRightAfterANegationIsNotARangeSeparator`, danach `killed` |
| internal/brain/privacy/glob.go:155 | a3 | `k < 0` -> `k <= 0` | Kein Unterschied: `bracketBody` bekommt als `i` den Index hinter einem `[`, also `i >= 1`; die Suche beginnt bei `i+1 >= 2`, und `indexHyphen` antwortet entweder -1 oder einen Index von mindestens 2 |
| internal/brain/privacy/glob.go:169 | a3 | `>` -> `>=` | Test `TestARangeOfOneCharacterIsNotEmpty`, danach `killed` |
| internal/brain/privacy/glob.go:184 | a3 | `k < j` -> `k <= j` | Kein Unterschied: `bracketBody` wird nur gerufen, nachdem Zeile 115 an einem `]` stehengeblieben ist und Zeile 118 es durchgelassen hat -- `pat[j]` ist also `]` und nie der Bindestrich, den der zusätzliche Schritt fände |
| internal/brain/privacy/glob.go:185 | a1 | `pat[k] == '-'` -> `true` | Test `TestARangeIsFoundWhereItStandsNotAtTheFirstCharacter`, danach `killed` |
| internal/brain/privacy/readable.go:42 | a2 | zwei Klauseln -> `manifest == nil` | Kein Unterschied: `MatchesGlobs` über eine leere Musterliste fällt durch seine Schleife und antwortet `false` (readable.go:27-32), und `!false` ist dasselbe `true` |
| internal/brain/search/fake.go:92 | a1 | `err != nil` -> `true` | Kein Unterschied: `err` ist der gescriptete Wert, und ein `return err` mit `err == nil` gibt denselben leeren Fehler zurück wie das `return nil` in Zeile 96 |
| internal/brain/search/http.go:45 | a1, a3 | `port <= 0` -> `false`, `port < 0` | Test `TestASessionAskedForNoPortGetsTheDaemonsOwn` (zwei Mutanten), danach `killed` |
| internal/brain/search/http.go:57 | a1 | `!s.portOpen(...)` -> `false` | Test `TestAClosedProbeAddressMeansUnreachable`, danach `killed`. Erst als 4b abgelegt (»ein geschlossener Port lässt auch keinen Handshake zu«) -- falsch, sobald `Host:Port` und `URL` verschiedene Orte nennen, und genau so ist der Test gebaut |
| internal/brain/search/http.go:68 | a1 (`true`), a3, a4 | `host == ""` -> `true`, `host != ""`, `!(...)` | Test `TestAClosedProbeAddressMeansUnreachable` (drei Mutanten), danach `killed` |
| internal/brain/search/http.go:68 | a1 (`false`) | `host == ""` -> `false` | Kein Unterschied, der zu prüfen wäre: bei gesetztem `Host` läuft der Rumpf ohnehin nicht, und bei leerem `Host` bleibt die Wähladresse `:<port>`, die Go als die des eigenen Rechners liest (`net.Dial`: »If the host is empty ... the local system is assumed«) -- also derselbe Rechner, auf den `DefaultHost` zeigt |
| internal/brain/search/http.go:72 | a1 | `port == 0` -> `false` | Kein Unterschied, der zu prüfen wäre: trennen ließe sich das nur mit einem Lauscher auf dem festen `DefaultPort` 8765, und `NewHTTPSession` (http.go:45) lässt `Port` nie auf 0 stehen -- die Wache antwortet nur einer von Hand gebauten Sitzung |
| internal/brain/search/http.go:84 | a1 | `s.URL != ""` -> `true` | Test `TestASessionWithoutAURLBuildsOneFromHostAndPort`, danach `killed` |
| internal/brain/search/http.go:88 und :92 | a1, a3, a4 | `host == ""` und `port <= 0` -> `true`, `false`, `!(...)`, `port < 0` | Test `TestTheWaitingErrorNamesTheAddressItWaitedOn` (acht Mutanten), danach `killed`: die Fehlerzeile von `WaitUntilReachable` trägt die gebaute Adresse |
| internal/brain/search/http.go:101 | a1, a3, a4 | `pollInterval <= 0` -> `true`, `false`, `!(...)`, `< 0` | Kein Unterschied, der zu prüfen wäre: der Wert geht einzig an `time.Sleep` zwischen zwei Proben einer schon scheiternden Schleife; alle Formen schlafen eine nicht-negative Zeit, und 0 von 250 ms zu trennen hieße messen, wie lange ein Fehlschlag gedauert hat |
| internal/brain/search/http.go:109 | a4 | `!now.Before(deadline)` -> `!(!...)` | Test `TestTheWaitingErrorNamesTheAddressItWaitedOn`, danach `killed` -- durch Zeitüberlauf: mit umgedrehter Frist bricht die Schleife nie ab und läuft in die 60-s-Grenze der Runde |
| internal/brain/search/http.go:159 | a1 (`false`), a4 | `timeout > 0` -> `false`, `!(...)` | Test `TestACallCarriesItsTimeoutOnTheRequest` (zwei Mutanten), danach `killed` |
| internal/brain/search/http.go:159 | a1 (`true`), a3 | `timeout > 0` -> `true`, `timeout >= 0` | Kein Unterschied: `postWithTimeout` ist unexportiert und hat zwei Rufer, `Handshake` mit `HandshakeTimeout` (http.go:126) und `Call` mit `QueryTimeout` (http.go:135) -- beide positiv, und für einen positiven Wert sind die drei Formen dieselbe |
| internal/brain/search/http.go:173 | a1 | `client == nil` -> `true` | Test `TestTheSessionAsksThroughTheClientItWasGiven`, danach `killed` |
| internal/brain/search/http.go:183 | a2 | `< 200 \|\| >= 300` -> `>= 300` | Kein Unterschied, der zu prüfen wäre: Go's Transport reicht genau einen Status unter 200 weiter, denn er behandelt eine 1xx-Antwort nur dann als Zwischenmeldung, wenn sie nicht 101 ist (`is1xxNonTerminal := is1xx && resCode != StatusSwitchingProtocols`, `net/http/transport.go`). Eine 101 erreicht die Zeile also. Sie herbeizuführen bräuchte einen rohen Lauscher, der von Hand ein Protokoll-Upgrade antwortet -- ein schlechterer Test als gar keiner |
| internal/brain/search/http.go:183 | a3 | `>= 300` -> `> 300` | Test `TestThreeHundredIsNotASuccessfulStatus`, danach `killed` |
| internal/brain/search/http.go:194 | a2 | `ok && rpcErr != nil` -> `ok` | Test `TestANullErrorMemberIsNoError`, danach `killed` |
| internal/brain/search/http.go:207 | a3 | `i >= 0` -> `i > 0` | Test `TestAStreamWhoseOnlyDataLineIsItsFirstOne`, danach `killed` |
| internal/brain/search/http.go:209 | a1 | `HasPrefix(line, "data:")` -> `true` | Test `TestALineWithoutTheDataFieldIsNotTheReply`, danach `killed` |
| internal/brain/search/http.go:211 | a1 | `data == ""` -> `false` | Kein Unterschied: ein leeres `data` ist kein gültiges JSON, `json.Unmarshal` scheitert daran, und Zeile 216 macht mit demselben `continue` weiter |
| internal/brain/search/mcp.go:128 | a1 | `p.cli == nil` -> `false` | Test `TestAPortBuiltWithoutACLIStillHasOne`, danach `killed`. Erst als nicht prüfbar abgelegt (»das hieße qmd starten«) -- falsch: ein leerer PATH lässt den Launcher den Namen nachschlagen und ablehnen, bevor ein Prozess entsteht, wie es `TestNewQmdMcpPort_DefaultConnectRefusesAMissingQmdWithoutANotice` seit 1b-1 tut |
| internal/brain/search/mcp.go:151 | a1 | `session == nil` -> `true` | Test `TestAnOpenSessionIsNotConnectedTwice`, danach `killed` |
| internal/brain/search/mcp.go:174 | a1 | `p.session != nil` -> `false` | Test `TestASessionThatFailedIsClosedAndLetGo`, danach `killed` |
| internal/brain/search/mcp.go:237 | a1 | `!ok` -> `false` | Kein Unterschied: der Index in eine nil-Map ist erlaubt und gibt den Nullwert, `sc["results"]` ist dann nicht `[]any`, und Zeile 241 antwortet für beide Formen dasselbe `nil` |
| internal/brain/search/mcp.go:241 | a1 | `!ok` -> `false` | Test `TestAReplyWithoutAResultsListIsNotAnEmptyList`, danach `killed`. Der Test hält `nil` gegen die leere Liste fest -- ein Unterschied, den heute kein Rufer liest, aber der einzige, den die Signatur weitergibt |
| internal/brain/search/mcp.go:256 | a2 | `ok && lineVal != nil` -> `ok` | Kein Unterschied: ein `nil` trifft keinen Zweig der Typprüfung darunter, `line` bleibt 1, und `if line < 1` ändert daran nichts |
| internal/brain/search/mcp.go:265 | a3 | `line < 1` -> `line <= 1` | Kein Unterschied: der Rumpf setzt `line` auf 1, für `line == 1` also auf den Wert, den es schon hat |
| internal/brain/search/mcp.go:320 | a1, a3 | `len(collections) > 0` -> `true`, `>= 0` | Test `TestAHitIsSplitEvenWhenNoCollectionWasAsked` (zwei Mutanten), danach `killed`: ohne die Prüfung greift der Mutant in eine leere Liste |
| internal/brain/search/qmd.go:54 | a1 | `err != nil` -> `true` | Kein Unterschied: ist `err` nil, scheitert die Typzusicherung auf `*exec.ExitError` darunter, und der Rumpf tut nichts |
| internal/brain/search/qmd.go:113 | a1 | `!isArray` -> `true` | Test `TestAJSONListThatHoldsNoHitsIsAReadingFailure`, danach `killed` |
| internal/brain/search/qmd.go:136, :140, :144, :148 | a1 | `raw.X != nil` -> `true` | Test `TestAHitMayCarryNothingButItsFileAndItsDocID` (vier Mutanten), danach `killed`: ohne die Prüfung liest der Mutant durch einen nil-Zeiger |
| internal/brain/search/qmd.go:195 | a3 | `start < 0` -> `start <= 0` | Test `TestAListingLineMayBeginWithTheLocation`, danach `killed` |
| internal/brain/search/qmd.go:210, :220, :238 | a1, a3, a4 | `exe == ""` -> `true`, `false`, `!(...)`, `exe != ""` | Test `TestEveryCommandFallsBackToTheQmdExecutable` (neun Mutanten), danach `killed` |
| internal/brain/search/search.go:156 | a3 | `len(ordered) > n` -> `>= n` | Kein Unterschied: bei `len(ordered) == n` schneidet der Mutant `ordered[:n]` ab, was die ganze Liste ist; und für die leere Liste ist `ordered[:0]` wieder dieselbe |

### Geparkt (Ruling R19)

Zwei Pakete bleiben unbewertet: `internal/brain/guard` (17 Überlebende) und
`internal/hooks` (109), zusammen 126. Beide sind Pakete der Stufe 1a, die diese
Stufe nicht geschrieben hat -- guard ist nur durch die eingegliederte
Schreibschranken-Stufe gewachsen. Eine Stufe, die im Vorbeigehen die Tests von
1a umschreibt, kann niemand mehr prüfen; darum wird hier nichts angefasst. Die
`SURVIVED`-Zeilen stehen vollständig in `$TEMP/mutants-1b1/brain-guard.log` und
`hooks.log`; wer weitermacht, braucht die Runde nicht zu wiederholen.

Damit sind von den 280 Überlebenden der Runde 103 durch einen nachgereichten
Test getötet, 51 mit einer Begründung abgelegt und 126 geparkt.

## Live-Vergleich

Python (`brain-mcp` im Tag-Worktree `loomux-1a-source`, ohne brain-Daemon) und loomux
(`bin/loomux.exe` aus dem Tor von `9ca6364`) gegen den echten Bestand, von
2026-09-16T17:24:29Z bis 2026-09-16T17:25:38Z (UTC), ohne `LOOMUX_STATE_DIR` und
`LOOMUX_LEGACY_BRAIN_DIR`, der qmd-Daemon vor dem ersten Befehl gestoppt. Die Registry
hält 11 Bereiche: `project/loomux` und die zehn aus `%LOCALAPPDATA%\brain\registry.toml`;
einen eigenen Block für den Worktree gibt es nicht. `qmd mcp stop` konnte den Daemon nicht
beenden -- die PID-Datei fehlte (Befund in Task 15) --, darum wurde der Prozess auf
Port 8765 unmittelbar gestoppt.

| Befehl | Exit | Zeilen stdout |
|---|---:|---:|
| `brain-mcp status` | 0 | 113 |
| `brain-mcp search latenz --profile full` | 0 | 34 |
| `loomux brain search latenz --profile full` | 0 | 34 |
| `loomux brain status` | 0 | 114 |

### `status`

```text
1a2
> project/loomux: never indexed; run `brain reindex`
```

### `search --profile full`, Trefferzeilen

```text
keine Ausgabe
```

### `search --profile full`, stdout

```text
keine Ausgabe
```

### `search --profile full`, stderr

```text
0a1
> note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.
```

Vorhergesagt waren genau die eingefügte `never indexed`-Zeile in `status` -- eine statt
zweier, weil die Registry keinen Worktree-Block trägt -- und der Aufwärm-Hinweis als erste
stderr-Zeile von loomux. Nichts darüber hinaus.

## Pilot

Der Rauchtest (Task 16) ist am 2026-09-16 auf Weisung des Nutzers vom
Implementierer dieses Tasks gefahren und hier abgehakt worden. Gemessen wurde
gegen die echte Registry (`%LOCALAPPDATA%\loomux\registry.toml`, nach Task 15
Schritt 1 mit den zehn Bereichen von ultra-brain neben `project/loomux`, ohne
Worktree-Block) und ultra-brains Zustandsverzeichnis (`%LOCALAPPDATA%\brain`),
mit `bin/loomux.exe` aus dem Tor von `74e9d54`, ohne `LOOMUX_STATE_DIR` und
`LOOMUX_LEGACY_BRAIN_DIR`. Was damit **nicht** belegt ist: der Kanal `cloud` auf
echten Daten und ein Bereich mit `local_only` — beides deckt der Korpus.

- [x] `brain catalog` → Exit 0, so viele Bereiche wie `[[area]]`-Blöcke in der Registry, byte-sortiert.
- [x] `brain catalog --scope engineering/python` → Exit 0, byte-gleich zu `index.md`.
- [x] `brain read index.md --scope engineering/python --section Dateien` → Exit 0, der an der Referenz gemessene Abschnitt.
- [x] `brain neighbors index.md --scope engineering/python` → Exit 0, `incoming: -`, `outgoing: _schema.md, audit.md, log.md`.
- [x] `brain read ../x.md --scope engineering/python` → Exit 1, `error: engineering/python/../x.md leaves the area`.
- [x] `brain read index.md` → Exit 2, `loomux brain read: error: the following arguments are required: --scope` (Unterbefehl).
- [x] `brain catalog extra` → Exit 2, `loomux brain: error: unrecognized arguments: extra` (oberster Parser).
- [x] `brain search latenz --profile fast` → Exit 0, Stempelbefund als letzte `note:`-Zeile.
- [x] `brain status` → Exit 0, erste Zeile der Stempel, zweite `project/loomux: never indexed`.

Gemessen am 2026-09-16, Binary `bin/loomux.exe` aus dem Tor von `74e9d54`:
`catalog` 0 mit 11 Bereichen; `catalog --scope engineering/python` 0, `cmp` 0;
`read index.md --section Dateien` 0, `cmp` 0; `neighbors index.md` 0, `cmp` 0.
Verweigert mit 1: `read ../x.md` („leaves the area"). Usage-Fehler mit 2:
`read index.md` beim Unterbefehl, `catalog extra` beim obersten Parser, `cmp` je
0. `search latenz --profile fast` 0 mit 5 Treffern und 1 `note:`-Zeile;
`status` 0 mit 114 Zeilen.
