# Paritätsliste Stufe 1a

**Quelle:** ultraloom `loomux-1a-source` (`9d01a60`), ultra-brain `loomux-1a-source` (`3cc72d2`).
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor Stufe 1a als fertig gilt.

| Fall / Bereich | Alt | Neu | Begründung | Freigabe |
|---|---|---|---|---|
| hook-pre-tool-use/unreadable-payload | ulguard: Exit 1, Host lässt den Aufruf durch | Exit 2 | Wächter scheitert geschlossen (Spec, Fehlerverhalten) | freigegeben 2026-09-15 |
| Policy-Ablehnung, alle Fälle | ulguard: nur stderr | zusätzlich deny-Hülle auf stdout | ein Wächter, eine Antwortform (`guard.Refuse`) | freigegeben 2026-09-15 |
| Befehlsregeln mit Lookahead | ulguard: Kompilierfehler verworfen, Regel greift nie | Konfiguration wird verweigert, Datei und Regel genannt | Befund 2026-09-14 | freigegeben 2026-09-15 |
| Scratchpad der Claude-Sitzung | brain guard: verweigert | erlaubt unter `<temp>/claude/*/*/scratchpad/` | Spec, Abweichungsliste | freigegeben 2026-09-15 |
| Manifestnamen | `.ultra-brain/config.toml`, `.brain.toml` | nur `.loomux/config.toml` | harter Schnitt | freigegeben 2026-09-15 |
| Wiki-Ort | `[wiki] path`, `[area] wiki` in `.brain.toml`; `bundle` in `answers.toml` | nur `[layout] wiki` | eine Stelle für eine Frage | freigegeben 2026-09-15 |
| `[check.lanes]` | brain liest Lanes aus dem Manifest | nicht gelesen | `[verify]` kommt in Stufe 2 | freigegeben 2026-09-15 |
| hook-post-tool-use/broken-wiki-page | `brain lint` als Kindprozess | Lint im Prozess | Spec, Hook-Pfad | freigegeben 2026-09-15 |
| hook-session-start | meldet wartende Läufe aus `.ultraloom/runs` | meldet sie nicht; warnt vor veraltetem Pilot-Binary | Flow-Migration ist Folgeprojekt | freigegeben 2026-09-15 |
| `lint --root` | nicht vorhanden | neues Flag, ohne Flag wie bisher | Fallsuite braucht einen festen Projektort | freigegeben 2026-09-15 |
| Shell-Schreibzugriff auf `.loomux/config.toml` | ulguard/brain guard: Bash und PowerShell ungeprüft, `cat >>`, `sed -i`, `tee`, `Set-Content`, `cp`/`mv` darauf gehen durch | eingebaute Befehlsregel verweigert Umleitung, In-place-Edit, schreibende und löschende Befehle/Cmdlets, Kopie auf das Manifest; Lesen bleibt erlaubt; kein aufgezeichneter Fall ändert sich | Befund 2026-09-19; AGENTS.md: das Manifest schreibt kein Agent | offen |
| Antigravity `run_command` | brain guard: kein Schreibwerkzeug | Policy prüft es nicht | Argumentschlüssel ungemessen; Stufe 2 | freigegeben 2026-09-15 |
| `SRC_UB/pkg/gitenv/gitenv_test.go` | liest `vcs.py` | entfällt | kein Python im Produkt | freigegeben 2026-09-15 |
| status, worktree link/unlink/remove | — | keine Fälle, umgezogene Tests | brauchen echte Worktrees und Junctions | freigegeben 2026-09-15 |
| Mutationsrunde | — | in Stufe 1b | `dev mutants` entsteht dort | freigegeben 2026-09-15 |
| `check`: Usage-Fehler | `ulinit check`: Exit 1 | Exit 2 | Usage-Konvention der Stufe (R4a) | freigegeben 2026-09-15 |
| `check gofmt`: Liste der ungeformten Dateien | stderr, mit Kopfzeile | stdout, eine Datei je Zeile | die Liste ist das Ergebnis, nicht die Meldung (R4a) | freigegeben 2026-09-15 |
| `check`: Meldungen auf stderr | `ulinit check …`, `usage: ulinit …` | `loomux check …` | ein Werkzeugname (R4a) | freigegeben 2026-09-15 |
| `lint` ohne Datei | brain: Exit 1 | Exit 2 | Usage-Konvention der Stufe (R8c) | freigegeben 2026-09-15 |
| `lint` mit mehr als einer Datei | brain lintete `args[0]` und übersah den Rest | Exit 2 | ein stiller Teilbefund ist schlimmer als eine Absage | freigegeben 2026-09-15 |
| `[area] wiki = false` | schaltete das Wiki ab | wirkt nicht mehr | Opt-out entfällt ersatzlos (R8g) | freigegeben 2026-09-15 |
| `hook` ohne Event / unbekanntes Event | Exit 1 | Exit 2 | Usage-Konvention der Stufe (R9a) | freigegeben 2026-09-15 |
| `.loomux/config.toml` ohne `[area]` | Manifest ohne `[area]` war ein Fehler | deklariert nichts, Schranke arbeitet weiter | eine Datei mit Sektionen je Modul; ein Projekt nur mit `[policy]` ist gültig (R7a) | freigegeben 2026-09-15 |
| hook post-tool-use, Wirt | brain/ulguard kannten nur Claude-Nutzlasten | in 1a Claude-only; eine Antigravity-Nutzlast endet still mit 0 | Adapter vollständig in Stufe 2 (R11d) | freigegeben 2026-09-15 |
| Wortlaut der Schreibschranke | „the wiki guard …" | „loomux …" | ein Name in jeder Ablehnung; Nachrichtenklasse, Wortlaut frei | freigegeben 2026-09-15 |
| Git-Umgebung säubern | guard und wiki trugen je eine eigene Liste | eine Liste, 28 Namen aus beiden Repos | ein Ort für eine Frage (`internal/gitenv`) | freigegeben 2026-09-15 |
| Format der Lint-Befunde | durch Fälle belegt | durch `report_test.go` belegt | Lint-Fälle vergleichen Exit und leeres stdout; das Format ist kein Fallgegenstand (R14c) | freigegeben 2026-09-15 |
| CRLF in Aufzeichnungen | `tools/cases.py` normalisierte CRLF zu LF | der Rekorder normalisiert nur Pfade | trägt ein aufgezeichnetes stdout CRLF und der Lauf im Prozess nicht, ist das ein Unterschied der Aufzeichnung, nicht des Verhaltens | freigegeben 2026-09-15 |
| Antigravitys Hook-Vertrag nachmessen (offene Frage der Spec) | — | in Stufe 1b | braucht den Antigravity-Wirt, den 1a nicht hat; kein Task der Stufe 1a deckt sie ab (Controller-Ruling) | freigegeben 2026-09-15 |
| `wiki-drift` für ein Wiki im Projektrepo | zweites `git status` im Wiki; git antwortet repoweit, also war `wikiChanges == codeChanges` und die Meldung unerreichbar | ein `git status`, die Pfade werden am Wiki-Pfad geteilt | die alte Form ist in der ub-Migrationsspec (2026-09-04, Anhang B) als Defekt vermerkt, „eigener Zug"; der Umzug 1a fror sie nur mit um | freigegeben 2026-09-17 |

## Pilot

**Reihenfolge, vor Schritt 1:** `.loomux/config.toml` (Schritt 1 der Aufgabe 16)
und `%LOCALAPPDATA%\loomux\registry.toml` (Schritt 2) müssen **vor** der ersten
Sitzung mit den eingetragenen Hooks stehen. Eine fehlende `.loomux/config.toml`
ist harmlos — fehlende Datei heißt leere Policy. Eine fehlende Registry ist es
nicht: die Schreibschranke behandelt „keine Registry" wie „kaputte Registry" und
**verweigert jeden Write** (`internal/brain/guard/registry.go`, Fall
`no registry` in `run_test.go`). Wer die Sitzung vorher startet, steht still.

Der Rauchtest (Step 4 der Aufgabe 16) ist am 2026-09-15 auf Weisung des Nutzers
vom Controller gefahren und hier abgehakt worden. Gemessen wurde gegen die echte
Registry (`%LOCALAPPDATA%\loomux\registry.toml`) und die echte
`.loomux/config.toml`: je Schritt wurde `bin/loomux.exe` direkt mit der
Hook-Nutzlast gerufen und der Exit-Code samt Begründung geprüft. Was damit
**nicht** belegt ist: dass Claude Code die Hooks aus `.claude/settings.json`
selbst auslöst — das zeigt sich beim nächsten Sitzungsstart im Repo.

- [x] Session-Start: kein Fehler. Nach einer Änderung an einer `.go`-Datei ohne
      Neubau (`touch internal/cli/cli.go`) erscheint die Warnung beim nächsten
      Session-Start; nach `sh .githooks/pre-commit` nicht mehr — auch nicht nach
      dem Commit, der auf das Tor folgt.
- [x] Edit an `internal/cli/cli.go` (Kommentarzeile) → erlaubt; post-edit läuft
      ohne Meldung.
- [x] Write an `.env` → verweigert, Grund `secrets`.
- [x] Edit an `.loomux/config.toml` → verweigert.
- [x] Write an `C:/Users/micro/Documents/#GIT/ultraloom/x.md` → verweigert.
- [x] Write in das Scratchpad der Sitzung → erlaubt.
- [x] Bash `git push` → verweigert.
- [x] Commit mit englischer Nachricht → Tore grün; mit deutscher →
      commit-msg verweigert.

Gemessen am 2026-09-15, Binary `bin/loomux.exe` aus dem Tor von `b55d3e2`:
Session-Start ohne Befund 0; nach `touch internal/cli/cli.go` meldet er
`loomux binary bin/loomux.exe is older than internal/cli/cli.go` und nach
`sh .githooks/pre-commit` schweigt er wieder. Edit an `internal/cli/cli.go` 0,
post-tool-use auf dieselbe Datei 0 ohne Meldung. Verweigert mit 2: `.env`
(„secrets are not written by an agent"), `.loomux/config.toml` (Manifest),
`bin/x` (Pfadregel dieses Projekts), `#GIT/ultraloom/x.md` („lies outside every
writable tree"), Bash `git push` („a human's decision"). Erlaubt mit 0: Bash
`go test ./...` und ein Write ins Scratchpad der Sitzung. `check commit-msg`
englisch 0, deutsch 1 mit Umlautbegründung.

## Was die Fälle der Stufe 1a decken

`testdata/cases/1a-source/` hält die Aufzeichnungen der alten Binaries (Beweis,
nie nachbearbeitet), `testdata/cases/1a/` die Übersetzung, an der loomux
gemessen wird. Neunzehn Fälle, gefahren von
`internal/cli/cases_test.go`. Angepasst wurde genau eine Erwartung:
`hook-pre-tool-use/unreadable-payload` von Exit 1 auf 2 — die erste Zeile
dieser Liste.
