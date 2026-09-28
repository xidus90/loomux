# Plan: `dev bench search` ohne `--scope` misst, wo der Fragensatz liegt; Messung 4a-1 Task 1 in die Akte

Zweig: `fix/bench-search-scope` von `origin/master` (24f12d2c). Spec: keine
eigene; Grundlage sind die Befunde vom 2026-09-28 in
`docs/.superpowers/parity/stufe-4c-2.md` (auf dem Zweig von PR #52) und die
Entscheidung des Nutzers vom 2026-09-28 („Vorschlag nehmen“).

## Global Constraints

- Coverage 100 % je Funktion; Ausnahme nur mit `//coverage:exempt <Grund>`
  direkt über `func` (AGENTS.md).
- Code, Kommentare, Fehlermeldungen englisch; Arbeitspapiere deutsch.
- Commits: Conventional Commits, nennen kein Arbeitspapier; Autor ist der
  Nutzer, keine Mitautor-Zeile. Agenten pushen nie.
- Mehrzeilige Commit-Nachrichten per Datei und `git commit -F`.
- Das Tor ist `sh ci/gate.sh`; der Pre-Commit-Hook führt es aus.

## Task 1: `--scope` fällt auf die Bereiche des Fragensatzes

**Befund:** `loomux dev bench search` ohne `--scope` misst nur den Bereich
`knowledge` (`internal/dev/benchsearch/run.go`, `c.scope = "knowledge"`). Der
Alltags-Fragensatz liegt in `knowledge/98 Messung/questions.yaml`, seine
`expect`-Pfade zeigen aber in andere Bereiche (`project/space` u. a.). Ergebnis
war still 0/50, auch bei `keyword`.

**Verhalten danach:**

- `knowledge` bleibt die Vorgabe, um `--out` und damit den Fragensatz zu
  finden (unverändert).
- Ist `--scope` nicht gesetzt und läuft kein `--corpus`, wird der gemessene
  Bereich nach dem Laden des Fragensatzes aus den `expect`-Pfaden
  abgeleitet: Liegen alle in genau einem registrierten Bereich (Pfad des
  Bereichs ist Präfix, Vergleich wie sonst im Paket für Pfade üblich), wird
  dieser gemessen; liegen sie in mehreren, wird `all` gemessen.
- Liegt ein `expect`-Pfad in keinem registrierten Bereich, bricht der Lauf
  mit Exit 1 ab und nennt die Frage-ID und den Pfad, mit dem Hinweis, einen
  Bereich mit `--scope` zu nennen. Ein ausdrücklich gesetztes `--scope`
  (auch `knowledge`) wird nie überschrieben.
- Der Bericht nennt den gemessenen Bereich (falls er es heute nicht schon
  tut, dort ergänzen, wo der Kopf des Berichts gebaut wird).

**Tests (TDD, erst rot):**

1. Fragensatz mit allen `expect` in Bereich A, ohne `--scope` → gemessen
   wird A (nicht `knowledge`).
2. `expect` in A und B, ohne `--scope` → gemessen wird `all`.
3. Ein `expect` außerhalb aller Bereiche, ohne `--scope` → Exit 1, Meldung
   enthält Frage-ID und Pfad.
4. `--scope knowledge` ausdrücklich gesetzt, `expect` in A → gemessen wird
   `knowledge` (keine Ableitung).
Jeder Test muss rot werden, wenn die Ableitung entfernt wird (einmal per
`go test -overlay` prüfen und im Bericht festhalten).

**Doku:** Die Beschreibung von `--scope` in `docs/en/cli-reference.md` und
`docs/de/cli-reference.md` (falls `dev bench search` dort steht, sonst die
Stelle, die per `grep -rn "bench search" docs/en docs/de README*.md` die
Flags erklärt) und der Hilfetext des Flags (`scope` in der Flag-Definition)
sagen: ohne Angabe die Bereiche der `expect`-Pfade.

**Commit:** `fix(dev): measure the areas a question set points at when no scope is given`

## Task 2: Messung 4a-1 Task 1 in die Akte

`docs/.superpowers/parity/stufe-4a-1.md`: Unter „Offene Punkte“ im Punkt
„Menschenschritte“ den Teil „Task 1 (Arbeitsverzeichnis)“ als erledigt
ausweisen und darunter (oder im Abschnitt „Messungen vor dem Bau“, falls es
ihn gibt) diesen Absatz einfügen:

> **Task 1, gemessen am 2026-09-28 (Claude Code Desktop, CLI 2.1.x):** In der
> Desktop-App startet jeder `loomux mcp`-Prozess im Verzeichnis seiner
> Sitzung, bei einer Sitzung auf einem Worktree also im Worktree (gelesen per
> `psutil` an sieben laufenden Servern; auch agy startet ihn im
> Projektverzeichnis). `claude -p` aus `internal/` startet einen stdio-Server
> in `…\internal`. Die CLI-Probe lief über `--mcp-config
> --strict-mcp-config`, nicht über den Nutzerbereich; die
> Nutzerkonfiguration blieb unberührt. Die Annahme der Spec trägt: die
> Brücke sucht vom Startverzeichnis aufwärts.

In `docs/en/migration.md` und `docs/de/migration.md` in der Zeile 4a-1 den
Menschenschritt „MCP working-directory measurement“ bzw. dessen deutsche
Entsprechung streichen, die beiden übrigen bleiben.

**Commit:** `docs: record where Claude Code starts a stdio server`
