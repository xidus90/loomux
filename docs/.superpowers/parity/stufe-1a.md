# Paritätsliste Stufe 1a

**Quelle:** ultraloom `loomux-1a-source` (`9d01a60`), ultra-brain `loomux-1a-source` (`3cc72d2`).
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor Stufe 1a als fertig gilt.

| Fall / Bereich | Alt | Neu | Begründung | Freigabe |
|---|---|---|---|---|
| hook-pre-tool-use/unreadable-payload | ulguard: Exit 1, Host lässt den Aufruf durch | Exit 2 | Wächter scheitert geschlossen (Spec, Fehlerverhalten) | |
| Policy-Ablehnung, alle Fälle | ulguard: nur stderr | zusätzlich deny-Hülle auf stdout | ein Wächter, eine Antwortform (`guard.Refuse`) | |
| Befehlsregeln mit Lookahead | ulguard: Kompilierfehler verworfen, Regel greift nie | Konfiguration wird verweigert, Datei und Regel genannt | Befund 2026-09-14 | |
| Scratchpad der Claude-Sitzung | brain guard: verweigert | erlaubt unter `<temp>/claude/*/*/scratchpad/` | Spec, Abweichungsliste | |
| Manifestnamen | `.ultra-brain/config.toml`, `.brain.toml` | nur `.loomux/config.toml` | harter Schnitt | |
| Wiki-Ort | `[wiki] path`, `[area] wiki` in `.brain.toml`; `bundle` in `answers.toml` | nur `[layout] wiki` | eine Stelle für eine Frage | |
| `[check.lanes]` | brain liest Lanes aus dem Manifest | nicht gelesen | `[verify]` kommt in Stufe 2 | |
| hook-post-tool-use/broken-wiki-page | `brain lint` als Kindprozess | Lint im Prozess | Spec, Hook-Pfad | |
| hook-session-start | meldet wartende Läufe aus `.ultraloom/runs` | meldet sie nicht; warnt vor veraltetem Pilot-Binary | Flow-Migration ist Folgeprojekt | |
| `lint --root` | nicht vorhanden | neues Flag, ohne Flag wie bisher | Fallsuite braucht einen festen Projektort | |
| Antigravity `run_command` | brain guard: kein Schreibwerkzeug | Policy prüft es nicht | Argumentschlüssel ungemessen; Stufe 2 | |
| `SRC_UB/pkg/gitenv/gitenv_test.go` | liest `vcs.py` | entfällt | kein Python im Produkt | |
| status, worktree link/unlink/remove | — | keine Fälle, umgezogene Tests | brauchen echte Worktrees und Junctions | |
| Mutationsrunde | — | in Stufe 1b | `dev mutants` entsteht dort | |
| `check`: Usage-Fehler | `ulinit check`: Exit 1 | Exit 2 | Usage-Konvention der Stufe (R4a) | |
| `check gofmt`: Liste der ungeformten Dateien | stderr, mit Kopfzeile | stdout, eine Datei je Zeile | die Liste ist das Ergebnis, nicht die Meldung (R4a) | |
| `check`: Meldungen auf stderr | `ulinit check …`, `usage: ulinit …` | `loomux check …` | ein Werkzeugname (R4a) | |
| `lint` ohne Datei | brain: Exit 1 | Exit 2 | Usage-Konvention der Stufe (R8c) | |
| `lint` mit mehr als einer Datei | brain lintete `args[0]` und übersah den Rest | Exit 2 | ein stiller Teilbefund ist schlimmer als eine Absage | |
| `[area] wiki = false` | schaltete das Wiki ab | wirkt nicht mehr | Opt-out entfällt ersatzlos (R8g) | |
| `hook` ohne Event / unbekanntes Event | Exit 1 | Exit 2 | Usage-Konvention der Stufe (R9a) | |
| `.loomux/config.toml` ohne `[area]` | Manifest ohne `[area]` war ein Fehler | deklariert nichts, Schranke arbeitet weiter | eine Datei mit Sektionen je Modul; ein Projekt nur mit `[policy]` ist gültig (R7a) | |
| hook post-tool-use, Wirt | brain/ulguard kannten nur Claude-Nutzlasten | in 1a Claude-only; eine Antigravity-Nutzlast endet still mit 0 | Adapter vollständig in Stufe 2 (R11d) | |
| Wortlaut der Schreibschranke | „the wiki guard …" | „loomux …" | ein Name in jeder Ablehnung; Nachrichtenklasse, Wortlaut frei | |
| Git-Umgebung säubern | guard und wiki trugen je eine eigene Liste | eine Liste, 28 Namen aus beiden Repos | ein Ort für eine Frage (`internal/gitenv`) | |
| Format der Lint-Befunde | durch Fälle belegt | durch `report_test.go` belegt | Lint-Fälle vergleichen Exit und leeres stdout; das Format ist kein Fallgegenstand (R14c) | |
| CRLF in Aufzeichnungen | `tools/cases.py` normalisierte CRLF zu LF | der Rekorder normalisiert nur Pfade | trägt ein aufgezeichnetes stdout CRLF und der Lauf im Prozess nicht, ist das ein Unterschied der Aufzeichnung, nicht des Verhaltens | |

## Was die Fälle der Stufe 1a decken

`testdata/cases/1a-source/` hält die Aufzeichnungen der alten Binaries (Beweis,
nie nachbearbeitet), `testdata/cases/1a/` die Übersetzung, an der loomux
gemessen wird. Neunzehn Fälle, gefahren von
`internal/cli/cases_test.go`. Angepasst wurde genau eine Erwartung:
`hook-pre-tool-use/unreadable-payload` von Exit 1 auf 2 — die erste Zeile
dieser Liste.
