# Antigravity-Hooks: was gemessen ist und was nicht

**Datum:** 2026-09-10, 14:30–15:05
**Werkzeug:** `agy` 1.1.24 (`C:\Users\micro\AppData\Local\agy\bin\agy.exe`)
**Probe:** `C:\Users\micro\AppData\Local\Temp\agy-probe`, ein Wegwerf-Workspace
außerhalb jedes Repos

## Auftrag

Drei Fragen aus `2026-09-10-go-hooks-drei-hosts-design.md`, Stufe 0:

1. Liest Antigravity die JSON-Hülle auf stdout, oder zählt nur der Exit-Code?
2. Kann `PreInvocation` Kontext in das Modell schreiben?
3. Gibt es eine harte Zeitgrenze für Stop-Handler, und liegt sie unter den
   300 s, die das Gate braucht?

**Keine der drei ist beantwortet.** Was stattdessen gemessen wurde, ist der
Grund dafür, und es sind vier Befunde, die den Entwurf ohnehin berühren.

## Befund 1 — Wo agy Hooks liest

Zwei Orte, beide aus den im Binary eingebetteten Hilfetexten (`strings` auf
`agy.exe`, 2026-09-10):

- `<workspace>/.agents/hooks.json` — pro Projekt. Wörtlich: „respects
  configurations in the `<project-root>/.agents/` folder, loading …"
- `~/.gemini/config/plugins/<name>/hooks.json` — pro Plugin, nutzerweit.
  Wörtlich: „**Hooks** defined in `plugins/<name>/hooks.json` are registered
  and run".

Auf dieser Maschine hat nur das superpowers-Plugin eine solche Datei.
`~/.antigravity` existiert nicht; die Wurzel ist `~/.gemini/`.

Damit ist `.agents/hooks.json` als Ort **bestätigt** — ultralooms Datei liegt
richtig.

## Berichtigung — welche Variable die Doku nennt

Eine Behauptung aus dem Entwurf und aus einer früheren Fassung dieses Textes
war falsch abgeschrieben. `MIGRATION.md:157` und
`skills/migrate-to-antigravity/SKILL.md:100` sagen beide
`${CLAUDE_PLUGIN_ROOT}` — **nicht** `${CLAUDE_PROJECT_DIR}`. Die zweite
Variable kommt in keiner der beiden Dateien vor.

Was daraus folgt: dass Antigravity `CLAUDE_PROJECT_DIR` nicht setzt, ist
**ungemessen**. Es ist naheliegend, weil es eine Variable von Claude Code ist,
aber naheliegend ist nicht gemessen. Was gemessen ist, steht im nächsten
Abschnitt und trägt die Wurzelsuche allein: das Arbeitsverzeichnis eines Hooks
ist nicht die Projektwurzel.

Gefunden hat das der Implementierer von Aufgabe 6, beim Nachrechnen der
Briefbehauptungen.

## Befund 2 — Das Arbeitsverzeichnis eines Hooks

Aus demselben Hilfetext: „The working directory is set to the directory
containing `hooks.json`." Also `.agents/`, **nicht** die Projektwurzel.

Nachgerechnet an der Probe: ein Hook mit `bash ./probe.sh`, dessen Skript
neben `.agents/` lag, fand es nicht. Erst das Skript *in* `.agents/` war
erreichbar.

Das erklärt den Fehler, der in ultralooms `.agents/hooks.json` bis Commit
`3e1c01a` stand, von der anderen Seite: `uv run --directory <ultra-brain>
python .ultra-brain/hooks/wiki_guard.py` war nicht nur wegen `--directory`
falsch, sondern hätte auch ohne es den relativen Pfad nicht getroffen. Ein
Kommando **beim Namen** — `brain guard` — ist die einzige Form, die von hier
aus trägt.

## Befund 3 — Vertrauen ist Voraussetzung

Eine Changelog-Zeile im Binary: „Fixed workspace-local hooks defined in
`<workspace>/.agents/hooks.json` not loading after trusting a folder by
reloading hooks whenever workspaces change."

Projektlokale Hooks laden also nur für einen vertrauten Ordner.
`~/.gemini/antigravity-cli/settings.json` führt hier
`trustedWorkspaces: ["C:\\Users\\micro"]` — das ganze Benutzerverzeichnis,
also auch die Probe und auch ultraloom. Diese Bedingung war während der
Messung erfüllt und ist als Ursache ausgeschlossen.

Für einen fremden Rechner heißt es: `ulinit` kann eine `.agents/hooks.json`
schreiben, und sie tut nichts, bis jemand den Ordner vertraut. Das gehört in
die Meldung, die `ulinit` beim Schreiben ausgibt.

## Befund 4 — Im Print-Modus feuert kein Lebenszyklus-Hook

Der eigentliche Grund, warum die drei Fragen offen sind.

Drei Anläufe, jeder mit `agy -p "…"` in der vertrauten Probe:

| Aufbau | Hook | Ergebnis |
|---|---|---|
| `bash ./probe.sh`, Skript daneben | `Stop`, `PreInvocation` | keine Nutzlast |
| `bash probe.sh`, Skript in `.agents/` | `Stop`, `PreInvocation` | keine Nutzlast |
| dasselbe, Nutzlast auf absoluten Pfad | `Stop`, `PreInvocation` | keine Nutzlast |

Gegenprobe, dass das Skript selbst trägt: `echo '{"probe":"self test"}' | bash
.agents/probe.sh` schreibt die Nutzlastdatei. Das Skript, der Pfad, das
Schema und das Vertrauen sind damit als Ursache ausgeschlossen.

`PreToolUse` ließ sich nicht provozieren, weil im Print-Modus **kein**
Werkzeug läuft, das eine Freigabe braucht. agys eigene Meldung:

> a tool required the "read_file" permission that headless mode cannot prompt
> for, so it was auto-denied. Add an allow-rule under permissions.allow in
> settings.json … Alternatively, re-run with
> `--dangerously-skip-permissions`.

Das gilt auch für `read_file` — nicht nur für die Shell.

## Was daraus folgt

**Die drei Fragen brauchen eine interaktive Sitzung**, oder eine der beiden
Lockerungen, die agy selbst nennt: Freigaberegeln in
`~/.gemini/antigravity-cli/settings.json`, oder `--dangerously-skip-permissions`.
Beides ist eine Entscheidung des Nutzers: das eine ändert die globale
Konfiguration eines anderen Werkzeugs, das andere lässt einen Agenten mit
abgeschalteter Freigabe laufen.

**Und ein Befund, der über die drei Fragen hinausgeht**, in drei Stufen, weil
eine einzige Formulierung hier zu stark oder zu schwach wäre:

1. **`Stop` und `PreInvocation` feuern im Print-Modus belegt nicht.** Nicht
   „unbelegt" — drei Aufbauten, keine Nutzlast, Gegenprobe von Hand
   erfolgreich. Für diese zwei Ereignisse ist die Aussage positiv gemessen.
2. **`PreToolUse` ist ungemessen.** Es ließ sich nicht provozieren, weil im
   Print-Modus kein freigabepflichtiges Werkzeug läuft. Ob es feuern würde,
   wenn eines liefe, sagt diese Messung nicht. Der eine Aufbau, der es hätte
   zeigen können — mit `permissions.allow` —, endete im 5-Minuten-Timeout von
   `agy -p` mit laufender Runde.
3. **`brain guard` ist im `agy -p`-Pfad gegenstandslos**, und zwar aus dem
   zweiten Grund und nicht aus dem ersten: dort läuft ohne Freigaberegeln
   überhaupt kein Werkzeug, das eine Schreibschranke prüfen könnte.

Commit `3e1c01a` hat den Inhalt der Datei richtiggestellt — dass eine
`PreToolUse`-Schranke auf Antigravity tatsächlich läuft, ist damit weiter
nicht gezeigt. Der Satz „auf beiden Hosts" in jener Commit-Nachricht ist
stärker als die Belege.

Die Dreiteilung geht auf einen Widerspruch der Nachbarsitzung zurück, die
zu Recht einwandte, dass „unbelegt" für Punkt 1 zu schwach ist — und deren
Verallgemeinerung auf `brain guard` eine Stufe weiter griff als die Messung
trägt.

Eine weitere Changelog-Zeile setzt außerdem eine Versionsschwelle: „Improved
hook ordering so hooks defined in `hooks.json` run before the built-in
termination checks, which … lets `Stop` hooks run at all instead of sitting
unreachable behind the built-ins." Vor jener Version liefen Stop-Hooks
überhaupt nicht. Welche Version das war, sagt der Text nicht; 1.1.24 trägt
den Fix.
