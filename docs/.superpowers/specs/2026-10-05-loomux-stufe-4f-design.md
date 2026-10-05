# loomux Stufe 4f: keine Verweise auf die Vorgänger — Design

**Stand:** Entwurf 2026-10-05, gegen `origin/master` 80db6bb0 (4e ✅).
**Bezug:** Fusions-Spec, Nachtrag #24 und „#24 im Einzelnen“ (Klassen,
Ausnahmen, Fertig-Bedingung), Nachtrag #31 (Release-Neustart als Abschluss von
4f); Schwester-Spec `2026-10-05-loomux-release-neustart-design.md`.

## Ziel

In loomux verweist nichts mehr auf `ultraloom` und `ultra-brain`, außer in den
benannten Ausnahmen aus #24. Vorher ist belegt, dass loomux alles trägt, was
die Ursprungsrepos auf ihrem letzten Stand können. Mit dem Abschluss von 4f
endet die Beta: der Release-Neustart (Schwester-Spec) veröffentlicht v1.0.0
als stabiles Release.

## Ausgangslage, gemessen am 2026-10-05

- Beide Ursprungsrepos haben nach `git fetch --all` seit dem 2026-09-14 keinen
  Commit mehr, auf keinem Zweig.
- Jede Aufnahme unter `testdata/cases/*-source` stammt von einem von zwei
  Tags: ultraloom `loomux-1a-source` = `9d01a60`, ultra-brain
  `loomux-1a-source` = `loomux-3-source` = `3cc72d2` (Quellenzeilen der Akten
  `parity/stufe-*.md`). Beide Tags sind der Kopf des jeweiligen `master`.
- Vor den Tags liegen nur Seitenzweige: ultraloom
  `claude/ultra-loom-brain-fusion-a5bb17` (+4), `claude/wiki-stufe-2` (+2),
  `feature/agent-harness` (+61), `fix-audit-scheibe5` (+1); ultra-brain
  `claude/eager-engelbart-46d10e` (+2), `claude/exciting-sanderson-efc25b`
  (+2), `claude/scheibe-9b` (+1), `docs/artefakte-nach-lebensdauer` (+3),
  `feature/artefakte-nach-lebensdauer` (+44). Einige sind schon Nachträge
  (#20, #21, #29).
- Altverweise nach der Suchliste aus #24 (`git grep -l`): 1.079 Dateien unter
  `testdata/cases` (681 in `*-source`, 383 in den übersetzten Fällen, meist
  `notes.md`), 134 Arbeitspapiere, rund 130 Dateien in Code und Doku.
- Gelesen werden `*-source`, `*-map.toml`, `*-payloads`, `*-worlds` nur von
  `internal/dev/importcases` (`loomux dev import-cases`); die Wiedergabetests
  lesen die übersetzten Fälle unter `testdata/cases/<stufe>/`.
- Aus Klasse (a) lebt noch `loomux area check` samt der Altmanifest-Meldung in
  `internal/config/areadeclaration.go`.

## Schnitt: drei Pull Requests in fester Reihenfolge

| PR | Inhalt | Wartet auf |
|---|---|---|
| **A** Gleichheitsprüfung | Inventur und Gleichheitsbeleg, Akte `parity/stufe-4f.md`; löscht nichts | nichts |
| **B** Code und Doku | Klassen (a) Rest, (b), (c), (d), (f) Referenzdoku, (g), (i) als Vorschlag | A, alle Unterschiede entschieden |
| **C** Aufzeichnungen und Tor | Klasse (e), `dev import-cases` fällt, Tor-Test | B |

Die Brücke aus der Schwester-Spec läuft daneben und wartet auf nichts hiervon.

## PR A: Gleichheitsprüfung

**Inventur.** Dieselbe Durchsicht wie am 2026-09-19, gegen den Stand vom Tag
der Prüfung und über alle Zweige beider Repos, lokal und remote: jede Stelle
(Befehl, Flag, Konfigurationsschlüssel, Hook, Skill, Vorlage) ist in loomux
gebaut, einer Stufe oder einem Folgeprojekt zugeordnet oder freigegeben
weggefallen. Für die Seitenzweige gilt die Durchsicht je Commit vor dem Tag;
was dort liegt und weder Nachtrag noch Zuordnung hat, wird ein neuer
Nachtrag. Die Tabelle der Akte führt je Zweig die Commits und je Stelle die
Zuordnung.

**Gleichheitsbeleg der Aufnahmen.** Eine Aufnahme gilt als aktuell, wenn ihr
Tag gleich dem Kopf von `master` seines Repos ist (`git rev-parse <tag>^{}`
gegen `git rev-parse master` und `origin/master`) und der Arbeitsbaum, aus dem
aufgenommen wurde, sauber war (steht in den Akten). Der Vergleich geht über den
ganzen Baum, nicht über die aufgezeichneten Pfade: ein Helfer außerhalb ändert
das Verhalten genauso. Ist der Kopf ein anderer, wird die Stufe neu
aufgenommen und verglichen.

**Stichprobe der Reproduzierbarkeit.** Wo ein Aufnahmeskript liegt
(`parity/stufe-{3a,3b,3c,4a-2,4c-1,4d}-orakel/record.sh`), wird die Stufe
zusätzlich neu aufgenommen und Datei für Datei mit `*-source` verglichen. Ein
Unterschied heißt hier nicht „die Referenz hat sich bewegt“ (das schließt der
Beleg oben aus), sondern „die Aufnahme ist nicht reproduzierbar“ — Zeitstempel,
Pfade, Reihenfolge. Jeder Unterschied wird in der Akte erklärt und dem Nutzer
vorgelegt, bevor PR C die Aufzeichnungen aus dem Baum nimmt. Für 1a bis 2c gibt
es kein Skript; dort trägt der Beleg allein.

**Ergebnis.** `parity/stufe-4f.md` mit Inventur, Beleg und Stichprobe; neue
Nachträge in der Fusions-Spec. Kein Code.

## PR B: Code und Doku

- **(a) Rest.** `loomux area check` (`internal/cli/areacheck.go`, der Zweig in
  `internal/cli/area.go`) entfällt, ebenso die Altmanifest-Erkennung in
  `internal/config/areadeclaration.go`: ein Ordner mit nur `.brain.toml` ist
  danach ein Ordner ohne Manifest und bekommt die Meldung jedes anderen
  (`ErrNoManifest`). Der Aufrufer `IsUndeclared` und jeder tolerante Leser
  wird mit genau diesem Fall gegen die neue Meldung geprüft.
- **(b)** Die Tabelle der abgelösten Hooks in `internal/hooks/status.go`, der
  Schutz von `.ultraloom/vendor` in `internal/hooks/worktree.go` und der
  Ausschluss `/.ultraloom/` in `internal/cases/gitworld.go` werden gelöscht.
- **(c)** Kommentare, Paketdoku und Meldungen ohne die Namen: Herkunft wird
  gestrichen, ein Verweis auf die Python-Referenz wird „the reference“ oder
  fällt, wo nur Herkunft steht. `loomux status` druckt „Wiki:“ statt
  „UltraBrain Wiki:“.
- **(d)** Tests mit Altnamen als Eingabe fallen mit ihrem Code oder bekommen
  neutrale Namen. Übersetzte Fälle, deren `stdout`/`stderr` eine geänderte
  Meldung trägt, ziehen hier mit.
- **(f)** Referenzdoku in `docs/en`, `docs/de` und `docs/wiki`; `benchmarks.md`
  bleibt unverändert (Ausnahme).
- **(g)** Der Kopf von `AGENTS.md` wird neu gefasst; der Required Notice in
  `LICENSE.md` zeigt auf `https://github.com/xidus90/loomux`; die zwei Zeilen im
  Changelog bleiben (Ausnahme).
- **(i)** Den Kommentar in `.loomux/config.toml` ändert der Agent nicht; er
  legt `loomux config … --propose` vor, ein Mensch wendet an.

Release-Stufe: `release:major`, weil `loomux area check` als Befehl wegfällt.

Vorbedingung vor dem Merge: Kein Bereich der Registry hat nur ein Altmanifest.
Geprüft wird das je Eintrag mit einem Blick auf `.loomux/config.toml` neben
`.brain.toml`/`.ultra-brain/config.toml` und in der Akte festgehalten. Sonst
würde ein solcher Bereich still manifestlos.

## PR C: Aufzeichnungen und Tor

1. **Sichern.** Ein Tag `archive/parity-recordings` auf dem letzten Commit,
   der die Originale noch trägt; er beginnt nicht mit `v`, `NextVersion`
   übergeht ihn (`tagPattern`), und der Neustart löscht ihn nicht. Gepusht
   wird er vom Menschen.
2. **Entfernen.** `testdata/cases/*-source`, `*-map.toml`, `*-payloads`,
   `*-worlds`, `internal/dev/importcases` und `loomux dev import-cases`.
3. **Eigene Erwartungen.** Die übersetzten Fälle bleiben, wie sie sind; nur ihre
   `notes.md` werden ohne die Altnamen neu gefasst. Die Wiedergabetests ändern
   sich nicht.
4. **Tor-Test.** Ein Go-Test (Paket `internal/plancheck`, er hält schon Doku
   gegen die Spec) liest `git ls-files`, greppt nach der Suchliste aus #24 und
   lässt nur eine feste Liste zu: `docs/{en,de}/benchmarks.md`, die zwei
   Changelog-Zeilen, `docs/.superpowers/` außer den Archiven, die Archive
   `specs-ul/`, `specs-ub/`, `plans-ub/`, `plans-ul/`, `bench-ub/` sowie die
   Roadmap- und Planzeilen der laufenden Folgeprojekte (`ulflow`,
   `ultra-brain/web`) bis zu deren Abschluss, und die Testdatei selbst. Jedes
   Folgeprojekt kürzt die Liste bei seinem Abschluss.
5. **Release-Stufe.** `release:major`, weil `loomux dev import-cases` wegfällt.
6. **Plan.** Die Zeile 4f in beiden `migration.md` wird ✅, wenn auch der
   Neustart der Schwester-Spec durch ist (#31).

## Fertig ist 4f, wenn

- der Tor-Test grün ist und ein Grep nach der Suchliste nur in seinen
  Ausnahmen trifft,
- `parity/stufe-4f.md` jeden Unterschied der Stichprobe mit Entscheidung führt,
- und v1.0.0 nach der Schwester-Spec als stabiles Release veröffentlicht ist.
