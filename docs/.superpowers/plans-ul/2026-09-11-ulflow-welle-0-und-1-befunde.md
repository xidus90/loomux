# ulflow M1, Welle 0 und 1 — Befunde und Entscheidungen

**Plan:** [2026-09-11-ulflow-welle-0-und-1.md](2026-09-11-ulflow-welle-0-und-1.md) ·
**Spec:** [2026-09-11-ulflow-laufzeit-design.md](../specs/2026-09-11-ulflow-laufzeit-design.md)
**Stand:** umgesetzt und zusammengeführt am 2026-09-11, `feature/agent-harness` bis `2580803`

Was die Umsetzung entschieden, gefunden und für spätere Wellen offen gelassen
hat. Die Arbeitsdateien der Ausführung (Ledger, Briefs, Reviews) sind danach
gelöscht worden; dies ist ihr dauerhafter Rest.

## Entscheidungen während der Ausführung

| Entscheidung | Grund | Kosten, falls falsch |
|---|---|---|
| Die sieben Lanes liefen parallel, jede in `.worktrees/ulflow-<lane>` | Dateimengen paarweise getrennt; so gewünscht | ein Merge-Konflikt, eine Lane neu |
| Alle Subagenten auf Opus | Vorgabe des Nutzers | nur Kosten |
| Kein Effort gesetzt | Der Agent-Aufruf nimmt keinen an | Subagenten laufen mit dem Effort der Sitzung |
| Paketkommentare von `model` und `flowcfg` bleiben in `model.go` und `flowcfg.go` | Ein Kommentar je Paket ist gewahrt | zwei Kommentare verschieben |
| `parseStatus` erkennt R/C in beiden Status-Spalten | `git add -N` nach `mv` ergab ` R neu\0alt\0`; bei altem Pfad unter 3 Bytes Panik, sonst abgeschnittener Pfad | — |
| `worktree.py` bleibt unverändert, obwohl dort derselbe Fehler steckt | M1 ändert kein Python | Python schneidet solche Pfade weiter still ab, bis M4/M6 |
| Testfall mit einer Datei namens nur `.jsonl` in `NextID` | Sonst waren die geforderten 100 % nicht erreichbar | eine Testzeile |
| `InputHash` weicht von Pythons `input_hash` ab | Spec: keine Seite setzt Läufe der anderen fort | — |
| `nil` gegen leer (`null` gegen `[]`/`{}`) normalisiert der Runner | Die Hashes unterscheiden beides | ein Resume findet Ergebnisse nicht wieder |
| Der Lader bekommt ein eigenes Paket | `flow → flow/expr → flow` wäre ein Importzyklus | — |
| `internal/flow/doc.go` sagt schon jetzt „lives in a package of its own" | Die Welle-2-Aufgabe legt das Paket an und macht den Satz wahr | der Satz ist bis dahin verfrüht |

## Messungen vom 2026-09-11

Auf den ersten drei steht Code dieser Wellen; die vierte betrifft M2.

- Unter Windows meldet `os.ReadDir` auf einer Datei `fs.ErrNotExist`.
- `github.com/BurntSushi/toml` v1.6.0 verschluckt `agent = 3` beim Decodieren in ein Struct-Feld ohne Fehler.
- Mit beschädigtem Index: `git check-ignore -q .` endet mit 128, `git rev-parse --show-prefix` mit 0, `git status` mit 128.
- Claude Code 2.1.267 meldet sich bei MCP-Servern mit `initialize` und `protocolVersion "2025-11-25"`, nicht mit 2026-07-28.

## Für den Plan von Welle 2

1. `journal.Entries` liest Zahlen im `delta` als `float64` und Listen als `[]any`. Ganze Zahlen über 2^53 verlieren beim Nachgehen an Genauigkeit, und die Eingabe-Hashes späterer Knoten ändern sich. `UseNumber` benutzen oder die Grenze dokumentieren.
2. In `DefinitionHash` den Knoteneintrag geben, wie TOML ihn decodiert hat, nicht `flow.Node` (keine JSON-Tags, Go-Feldnamen landen im Hash). Nicht-endliche Zahlen (`nan`, `inf`) in Knotenschlüsseln beim Laden ablehnen, sonst scheitert `Canonical` erst zur Laufzeit.
3. Einen Namen, der in `[params]` und `[state]` zugleich steht, ablehnen.
4. `true`, `false` und `END` als Bezeichner reservieren. `true`/`false` liest `expr` vor den Parametern als Literal.
5. Der Runner füllt jedes deklarierte Feld und jeden Parameter, bevor eine Kante ausgewertet wird. `comparison.Holds` prüft Typen per Assertion und bricht sonst ab.
6. Entscheiden und testen, ob der Eingabe-Hash die Besuchszähler abdeckt. Python hasht nur die Daten; das ändert, wie ein Tor auf einem Zyklus beim Nachgehen wirkt.
7. `model.Reply.Model` ist ein nackter Modellname; `flow.Result.Model` braucht `<anbieter>:<modell>`. Der Baustein `agent` setzt den Anbieter davor.
8. Der Baustein `exit` entscheidet, ob er die Codes 0, 1 und 3 der Laufzeit erlaubt.
9. Deckel über Parameter können zur Laufzeit 0 oder negativ werden, und Graphregel 5 lässt sich für sie beim Laden nicht prüfen. Lader oder Runner behandelt beides.
10. Für eine Kante ohne `when` nicht `ParseCondition` rufen; eine leere Bedingung ist ein Fehler.
11. Feld- und Parameternamen auf `[A-Za-z_][A-Za-z0-9_]*` beschränken, sonst kann keine Bedingung sie nennen.
12. `tmpl.Render` nimmt nur `string`, `int`, `bool`, `[]string`; TOML-`int64`, JSON-`float64` und `[]any` vorher umwandeln.
13. Prüfstufe 7 löst Knoten, Flow und `[agent] default` einzeln auf (`Resolve(node, "")`, `Resolve("", flow)`, `Resolve("", "")`); `Resolve` selbst prüft nur den ersten gesetzten Namen und nennt die Stufe nicht.
14. Von Hand gebaute Testknoten setzen `MaxVisits`; `Cap{}` bedeutet null Besuche.
15. Die Spec-Tabelle „Pakete" nennt das Lader-Paket noch nicht.

## Für den Plan von Welle 3

1. `ulguard hook session-start` gibt für jeden wartenden Lauf `ultraloom resume <id>` aus, auch für Go-Läufe. Das Kriterium der Spec für Welle 3 („meldet einen wartenden Go-Lauf") würde mit dem falschen Befehl bestehen. Der Hook braucht die Zeile `runtime` der Laufmarke. `internal/runs` importiert `internal/flow` und damit `flowcfg`, TOML und `model`; entweder `Baseline` verlegen oder die Zeile im Hook selbst lesen.
2. `runs.NextID` zählt nur Journale und kann eine vergebene Nummer erneut ausgeben. `journal.Append` schreibt dann mit `O_APPEND|O_CREATE` still in das Journal eines anderen Laufs. Die Kommandozeile legt die Marke mit `O_EXCL` an und hinterlässt bei einer Ablehnung keine.
3. Die Kommandozeile übergibt `WriteMarker` immer `Runtime: "go"` und die Version; das Paket kann es nicht erzwingen.
4. Eine Marke ohne Basis zu schreiben ist ungetestet; vor Welle 3 nachholen.

## Kleinere Befunde, bewusst offen

- `expr`: nur Leerzeichen und Tab gelten als Leerraum; Spalten zählen Bytes; `"max_rounds + +1"` wird angenommen; Fehlertexte mischen Fragmente und Sätze; ein negativer Überlauf heißt „too large"; `kind` bezeichnet zwei Dinge; `t.Fatalf` in einer Tabellenschleife.
- `tmpl`: die Meldung „not a placeholder" nennt keine Position; Großbuchstaben in Namen ungetestet; „holds a int64"; die Meldung verschweigt, dass ein Name nicht mit einer Ziffer beginnt.
- `model`: kein Test, dass eine gescheiterte Anfrage mitgeschrieben wird; `Seen` kopiert nur die äußere Liste; `NewFake` übernimmt die Liste des Aufrufers.
- `flowcfg`: der Fall `[agent` prüft nur das Pfadpräfix; die Schlüsselliste eines Modelleintrags steht doppelt; `joinKeys`/`sortedKeys` liegen in fremden Dateien.
- `runs`: Weglassen leerer `Runtime`/`Version` und `slices.Clone` von `Dirty` ungetestet; `NextID` läuft bei einem Journal namens `9223372036854775807` über.
- `flow`: `Block` hat kein `Reads`; eine spätere Knotenart `command` braucht eine Vertragsänderung.
- `worktree.py`: `_parse_status` prüft nur die erste Status-Spalte, und das Beispiel im Docstring ist falsch (`ts/test_cli.py`, nicht `s/test_cli.py`).

## Betrieb der Lane-Worktrees

- Vor dem ersten Commit in einem frischen Worktree `env -u VIRTUAL_ENV uv sync`, sonst scheitert der Pre-Commit-Hook an `ultraloom.exe`: Das `#` im Pfad lässt uv bei jedem `uv run` neu installieren.
- Jeder Commit hinterlässt einen `dmypy`-Daemon aus dem `.venv` des Worktrees. Vor `ulguard worktree-remove` beenden, sonst meldet git beim Löschen „Invalid argument" und das Verzeichnis bleibt liegen.
- `ulguard` prüft Pfadregeln relativ zu `--root` der Sitzung; Dateien in Lane-Worktrees trifft keine Pfadregel. Der Diff jeder Lane wurde deshalb vor dem Merge gegen ihre Dateiliste geprüft.
- Beim Schreiben von Go-Dateien per Bash-Heredoc wurde einmal `\\` zu `\`; Dateien mit Backslashes mit Write oder Edit schreiben.
