# Wächter: einmaliger Hinweis auf den Code-Graph

## Absicht

Agenten in loomux-Projekten suchen Code mit grep, obwohl der Code-Graph
Aufrufer und Ketten in einem Schritt liefert. In 34 Transkripten dieses Repos
gab es keinen einzigen Graph-Aufruf, aber 1202 Shell-Greps. Vier
Agentenrunden vom 11. November 2026 (Wiki:
[`syntheses/graph-hinweis-im-waechter.md`](../../wiki/syntheses/graph-hinweis-im-waechter.md),
Zahlen: `docs/de/benchmarks.md`, Eintrag „Funktionswerte und Ketten im
Go-Graph“) zeigen: Seit der Graph Funktionswerte und Ketten kennt (`go/3`,
v1.24.0), kostet eine einmalige Ablehnung des ersten grep über Code nichts
extra und macht Antworten vollständiger, wo nur der Graph die Kette kennt.

Der Wächter soll genau diese Form ausliefern.

## Ziele

1. **Erste Code-Suche einmal ablehnen:** Sucht ein Agent zum ersten Mal in
   einer Session mit grep nach einem Bezeichner im Code, lehnt der
   PreToolUse-Hook den Aufruf ab. Die Begründung nennt
   `loomux graph callers <Name>` und sagt, dass derselbe Befehl beim zweiten
   Mal durchgeht.
2. **Erfasste Werkzeuge:** das Grep-Werkzeug von Claude Code und grep, `rg`,
   `git grep` und `Select-String` in den Shell-Werkzeugen, die der Wächter
   schon liest (Bash, PowerShell, `run_command`).
3. **Nur wo der Graph hilft:** Es gibt einen gebauten Graph für das
   Verzeichnis, in dem der Befehl läuft, und die Suche zielt auf Code einer
   Sprache, die der Graph liest (`.go`, `.py`, `.gd`), nicht auf Doku, Logs
   oder Konfiguration.
4. **Höchstens einmal pro Session:** Danach schweigt der Hinweis. Abgeschaltet
   wird er pro Projekt über einen Schlüssel unter `[guard]`.
5. **Abnahme:** Die Bestätigungsrunde (Runde 4) wird mit dem echten Binary
   statt mit dem Spike-Skript wiederholt; der Hinweis bleibt im Zeitbudget
   des Hooks (35 ms).

## Nicht-Ziele

- Read und Glob: Read ist vor jedem Edit Pflicht und meist gezielt.
- Hinweise nach dem grep (PostToolUse): Sie kamen im Spike zu spät.
- Verweis auf die MCP-Werkzeuge: Sie sind in Claude Code zurückgestellt und
  kosten einen ToolSearch-Umweg.
- Eine Graph-Abfrage im Hook: Sie dauerte im Spike 270–505 ms.
- Ein strenger Modus, der grep dauerhaft sperrt.

## Anti-Ziele

- Eine Ablehnung, die wie eine Policy-Verweigerung klingt: Ein Agent liest
  „loomux policy refused“ als „nie tun“ und weicht aus. Der Hinweis hat
  eigenen Wortlaut.
- Eine Schleife: Wird die Markierung nicht geschrieben, darf der Wächter
  nicht ablehnen.
- Ein Hinweis, der einen kalten Graph-Build auslöst: ohne gebauten Graph kein
  Hinweis.

## Randbedingungen

- PreToolUse-Budget 35 ms (`docs/de/benchmarks.md`). Der Detektor liest keine
  Datei außer der Markierung und prüft den Graph nur auf Existenz.
- Der Wächter verweigert alles, was schiefgeht (`guard.Refuse`). Für den
  Hinweis gilt das Gegenteil: Jeder Fehler auf seinem Weg lässt den Aufruf
  durch.
- Shell-Zeilen zerlegt der Wächter schon (`commandTools`, `typedLines` in
  `internal/hooks/guard.go`); der Detektor nutzt dieselbe Zerlegung.

## Entscheidungen (meine; Veto beim Lesen)

1. **Standardmäßig an.** Das ändert das Verhalten der Agenten in jedem
   loomux-Projekt, nicht nur hier. Begründung: Ohne Hinweis nutzt kein Agent
   den Graph, und die Messung zeigt keine Mehrkosten. Abschalten mit
   `[guard] graph_hint = false`.
2. **Wann der Detektor anschlägt:** Das Suchmuster nennt einen Bezeichner in
   Definitions- oder Aufrufform (`func X`, `type X`, `X(`, `.X(`) oder ist
   ein bloßer Bezeichner in CamelCase oder mixedCase mit mindestens vier
   Zeichen, und der Suchbereich ist Code: eine der Endungen oben, ein
   `--include`/`-t`/`--type`, ein `glob` oder `type` des Grep-Werkzeugs auf
   diese Endungen, oder ein Pfad ohne Doku-Endung. Stammt der Bereich
   erkennbar aus Doku, Logs oder Konfiguration (`.md`, `.log`, `.json`,
   `.toml`, `.yaml`, `docs/`), schweigt er. Vorlage ist der Detektor aus dem
   Spike; seine Fehlgriffe aus dem Transkript-Replay werden Testfälle.
3. **Einmal pro Session und Agent:** Die Markierung ist eine eigene Datei im
   Zustandsordner der Session, damit das bestehende Aufräumen sie mitnimmt.
   Sie wird vor der Ablehnung geschrieben; scheitert das Schreiben, geht der
   Aufruf durch. Trägt die Nutzlast eine `agent_id`, bekommt jeder Subagent
   seine eigene Markierung, sonst teilt er sie mit der Session.
4. **Graph am Arbeitsverzeichnis:** Gesucht wird der Graph des Bereichs, in
   dem `cwd` liegt, nicht der der Hook-Wurzel. In Worktrees ohne Graph
   schweigt der Hinweis.
5. **Wortlaut:** eigener Text, kein „policy refused“. Er nennt das Symbol aus
   dem Muster, `loomux graph callers <Name>` (Aufrufer),
   `--direction out` (Aufgerufene) und `loomux graph blast` (Folgen einer
   Änderung) und sagt, dass derselbe Befehl beim zweiten Mal durchgeht.
6. **Parallele Greps:** Schickt ein Agent zwei Greps in einer Nachricht, kann
   der Wächter beide ablehnen, bevor die Markierung greift. Das ist hingenommen.
7. **Matcher:** `Grep` kommt in den PreToolUse-Matcher der Host-Tabelle
   (`internal/setup/hostfile/table.go`) und in die `.claude/settings.json`
   dieses Repos.
8. **Andere Hosts:** Shell-Greps fängt der Hinweis bei jedem Host, dessen
   Shell-Werkzeug der Wächter liest. Ein eigenes Suchwerkzeug eines anderen
   Hosts als Claude Code bleibt außen vor.

## Bekannte Grenze

Der Hinweis nennt `loomux` vom `PATH`. Läuft der Hook mit einem anderen
Binary als dem auf dem `PATH` (in diesem Repo der Entwicklungs-Build unter
`%LOCALAPPDATA%\loomux\dev\`), kann `graph callers` den Graph mit einer
älteren Extraktor-Version neu bauen. Für Nutzer mit einer Installation gilt
das nicht.

## Dem Bauer überlassen

- Die genauen Muster des Detektors, solange die Testfälle aus dem Replay
  halten.
- Der Name der Markierungsdatei und die Form, in der der Schlüssel unter
  `[guard]` gelesen wird.

## Doku

`docs/de|en/hooks.md` (Entscheidungsweg von pre-tool-use),
`docs/de|en/configuration.md` (`[guard]` und das kommentierte Muster),
`docs/de|en/benchmarks.md` (Hook-Zeit kalt und warm mit Grep-Fällen, Runde 4
mit dem echten Binary), die Wiki-Synthese und das Wiki-Log.
