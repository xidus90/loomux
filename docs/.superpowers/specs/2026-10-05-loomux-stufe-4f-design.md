# loomux Stufe 4f: keine Verweise auf die Vorgänger — Design

**Stand:** Entwurf 2026-10-05, gegen `origin/master` 80db6bb0 (4e ✅); vom Nutzer freigegeben am 2026-10-05.
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
  wird mit genau diesem Fall gegen die neue Meldung geprüft, einmal als Probe,
  festgehalten in der Akte (#35).
- **(b)** Die Tabelle der abgelösten Hooks in `internal/hooks/status.go` und
  der Ausschluss `/.ultraloom/` in `internal/cases/gitworld.go` werden
  gelöscht. In `internal/hooks/worktree.go` bekommen die Kommentare und
  Fixturen zu `.ultraloom/vendor` neutrale Beispiele; die Schutzlogik bleibt
  (#35).
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
  `LICENSE.md` zeigt auf `https://github.com/xidus90/loomux`; die fünf Zeilen im
  Changelog bleiben (Ausnahme, #34 Weg (a)), benannt nach ihrem Eintrag statt
  nach Zeilennummer, weil jedes Release die Nummern verschiebt: je eine Zeile
  in 7.0.1, 7.0.0, 4.2.2, 2.5.0 und 2.3.0.
- **Aus #32 und #34 (freigegeben 2026-10-05):**
  - `loomux dev switchover` fällt weg, samt `internal/switchover`, der
    Wächterregel zu `dev switchover prune-hooks` und den Abschnitten der
    CLI-Referenz.
  - Die Marker `ultraloom`/`ulguard` in `gitfiles.RunsAGate` fallen, ebenso
    die Ausschlüsse `**/.brain.toml` und `**/.ultra-brain/**` in
    `index.AlwaysExcludes`. Die übersetzten Fälle mit `index.yml` ziehen mit.
  - Die Vorlage `brain-review/SKILL.md:86` nennt kein `ultra-brain-…` mehr.
  - `NeighbourWiki` (`internal/detect/edges.go`) wird samt Tests gelöscht.
  - #15 nennt die Installationsskripte von ultra-brain.
  - In der Roadmap beider READMEs:
    - Die Zeile „Register revisions only from review“ bekommt die Regel aus
      `cdf5dfd`.
    - Die Zeile W2 nennt Web 7a-2 (`edges-cross.json`) und 7b
      (Seitenleser, Volltextsuche, Prüfzentrum).
    - Die Flow-Zeile nennt `run --no-model` als Eingang.
  - Vor dem Wegfall des Korpus von `claude/scheibe-9b` wird `reconcile` mit
    einem unerwarteten Argument einmal geprobt (Exit 2).
- **(i)** Den Kommentar in `.loomux/config.toml` ändert der Agent nicht; der
  Plan nennt die neuen Zeilen, ein Mensch ändert sie von Hand, weil
  `loomux config … --propose` keine Kommentare kennt (#35).
- **Aus #35:** Die Roadmap-Zeile „The hint beside a declaration without
  `[area]`“ fällt aus beiden READMEs; die Prosa in `docs/wiki` wird neu gefasst,
  `sources:`-Zeilen in die Archive und `docs/wiki/log.md` bleiben.

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
   `*-worlds`, `internal/dev/importcases` und `loomux dev import-cases`,
   dazu `internal/dev/recordcase`, `loomux dev record-case` und `record-mcp-case` (#34), samt
   der Aufnahmeskripte unter `parity/stufe-*-orakel/`.
2a. **Sicherung der Ursprungsrepos (#33).** Je Repo ein `git bundle --all`
   samt einem Archiv der ungetrackten Dateien, als Anhang am selben Archiv
   wie die Aufzeichnungen. Vorher kopiert PR C die drei ulflow-Pläne nach
   `plans-ul/` (#32), und die Wurzelquellen `Bauanleitung_Second-Brain.pdf`
   und `NoteGPT_Transcript…txt` von ultra-brain werden am Inhalt gegen
   brain-knowledge gehalten; was dort fehlt, legt der Mensch in dessen
   Eingang. Das Bündel bereitet der Agent vor, das Hochladen führt der
   Mensch aus.
3. **Eigene Erwartungen (#24 (e), präzisiert 2026-10-05).** Die übersetzten
   Fälle unter `testdata/cases/<stufe>/` werden zu loomux' eigenen
   Erwartungen. Was die Wiedergabe vergleicht, steht in `parity/stufe-4f.md`
   (Abschnitt 4, je Suite nachgelesen). Danach gilt:
   - **Nicht verglichen**, die Altnamen werden per Ersetzungstabelle neutral
     gefasst: `notes.md`, `README.md`, jede `stderr` (LoadCase liest sie
     nie), `stdout` von Meldungs- und Lanes-Fällen, die `.mcp.json` unter
     `3a/area-add/*/world_after/repo-new/` (nur als „fehlt“ toleriert).
   - **Byte-genau verglichen**, die Kommentare in `4a2/hook/*/git.toml`
     ändern sich in `world/` und `world_after/` gleich.
   - **Weltdateien mit Altpfad:**
     `3a/area-add/known-scope/world_after/repo-new/.ultra-brain/config.toml`
     fällt samt ihrer Toleranzzeile (`cases_3a_test.go`).
     `4d/convert/no-registry/world/vault/.brain.toml` fällt, wenn der Fall
     sie laut `notes.md` nicht braucht; sonst wird sie umbenannt.
   - `world_after/xdg/qmd/index.yml` zieht PR B mit (#34).
   - Ein Wiedergabeschalter entsteht nicht. Jede Datei ändert sich nur in
     Altnamen; das Ersetzungsskript prüft, dass sich der Diff nach dem
     Maskieren allein durch sie erklärt.
3a. **Umzug.** `testdata/cases/2c-payloads/agy-*.json` liest
   `internal/hosts/antigravity_test.go:27`. Die Dateien ziehen zu den
   Testdaten von `internal/hosts`, statt zu fallen.
3b. **Policy.** Vier Pfadregeln in `.loomux/config.toml` schützen
   `1a-source`, `1b-1-source`, `2a-source` und `2b-source`. Ihr Entfernen
   und der Kommentar aus (i) laufen als `loomux config … --propose`; ein
   Mensch wendet an, bevor gelöscht wird.
4. **Tor-Test.** Ein Go-Test im Paket `internal/plancheck` (es hält schon
   Doku gegen die Spec).
   - **Bauweise:** eine reine Funktion über Dateiliste und Leser, dazu ein
     Läufer, der `git ls-files` liest. Die Suchliste ist die aus #24,
     case-insensitive.
   - **Ausnahmen:** eine Tabelle aus Pfad oder Muster, erlaubter Zeilenregel
     und Eigentümer (Flow, Web, Plan-Ende, Geschichte). Sie enthält:
     - `docs/{en,de}/benchmarks.md` und die Messchronik
       `testdata/bench/1a-hooks.json` und `testdata/bench/search/v1/baseline/*`,
     - die fünf Changelog-Zeilen: je genau ein Treffer in den Abschnitten
       7.0.1, 7.0.0, 4.2.2, 2.5.0 und 2.3.0. Die Überschrift wird auch in der
       Form `[7.0.1-beta]` erkannt, die Neustart-Schritt 3b einführt,
     - die Herkunftsspalte von `docs/*/migration.md` bis zum Ende des Plans,
     - `docs/.superpowers/` außer den Archiven, und die Archive `specs-ul/`,
       `specs-ub/`, `plans-ub/`, `plans-ul/`, `bench-ub/` eigens gelistet,
     - die Roadmap-Zeilen der laufenden Folgeprojekte (`ulflow`,
       `ultra-brain/web`),
     - die Ausnahmen aus #35 von PR B,
     - die Testdatei selbst.

     Jedes Folgeprojekt kürzt die Liste bei seinem Abschluss.
   - **Commit:** Der Tor-Test kommt als letzter Commit. Vorher wäre er rot,
     und jeder Commit läuft durch das Tor. Er löst die Verbotslisten in
     `internal/setup/templates/templates_test.go` ab.
5. **Reihenfolge.**
   - **Ab #81, ohne B:** Policy, Sichern, Löschen, Umzug, Erwartungen.
   - **Nach dem Rebase auf `master` mit B:** Doku (CLI-Referenz,
     READMEs, `AGENTS.md`-Zeile zu `testdata/cases/`) und der Tor-Test.
6. **Release-Stufe.** `release:major`, weil `loomux dev import-cases`
   wegfällt. Der Merge wird ein weiteres Pre-Release der alten Zählung.
7. **Plan.** Die Zeile 4f in beiden `migration.md` wird ✅, wenn auch der
   Neustart der Schwester-Spec durch ist (#31).

## PR D: Abschluss (Nachtrag #36, vor dem Neustart)

1. **Migrationsplan heraus.** Entfernt werden:
   - `docs/{en,de}/migration.md`,
   - `internal/plancheck/plancheck.go` samt Test,
   - die Regeln zum Migrationsplan in `AGENTS.md`,
   - die Links aus den READMEs.

   Der Tor-Test aus PR C bleibt. Er zieht in ein Paket, dessen Name nicht
   „plan“ sagt, oder bleibt in `internal/plancheck`, wenn das Paket sonst
   leer wird; das entscheidet der Plan.
2. **Arbeitspapiere ins Archiv.** `docs/.superpowers/specs`, `plans`,
   `parity`, die fünf Archive und alles Übrige unter `docs/.superpowers/`
   werden als Tarball mit `SHA256SUMS` ein weiterer Anhang des Releases
   `archive/parity-recordings`. Den lädt der Mensch hoch, danach verlassen
   die Dateien den Baum. Ausgenommen sind die Papiere des Neustarts, die
   noch gebraucht werden und keinen Altnamen tragen:
   `specs/2026-10-05-loomux-release-neustart-design.md`,
   `plans/2026-10-05-release-bruecke.md` und
   `plans/2026-10-05-release-neustart.md`. Den Beleg aus Schritt 2 des
   Neustarts hält dann der PR-Text des Neustarts fest, nicht mehr diese
   Akte.
   - **Wiki:** Die `sources:`-Zeilen, die auf sie zeigen, werden auf die
     Wiki-Seite selbst umgehängt oder fallen. `reindex` bringt
     `_identities.tsv` nach; geprüft wird das mit `lint` über das Wiki.
   - **Ablage künftiger Papiere:** Neue Arbeitspapiere (Specs, Pläne)
     entstehen weiter unter `docs/.superpowers/`. Nur die der Migration
     gehen.
3. **Roadmap-Zeile** der Web-App ohne `ultra-brain/web`.
4. **Tor-Test.** Die Ausnahmen schrumpfen auf `docs/{en,de}/benchmarks.md`,
   `testdata/bench/1a-hooks.json`, `testdata/bench/search/v1/baseline/*` und
   die fünf Changelog-Zeilen.
5. **Graft nur als Idee (Nachtrag #37).** `trailhq/Graft` bleibt an zwei
   Stellen:
   - **Lizenz:** Für den portierten Code steht der MIT-Hinweis von Graft
     (Copyright und Lizenztext, Stand `1e352a3`) in `NOTICE.md`. Der
     Generator `loomux dev notices` bekommt dafür einen Abschnitt für
     portierte Quellen.
   - **Idee:** Je ein Satz in README und `architecture.md` (en/de) sagt, dass
     Code-Graph, Ranking und Blast-Radius von Graft angeregt sind.

   Überall sonst fällt der Name:
   - in den Code-Kommentaren von `internal/code/**` und den übrigen
     Paketen, gezählt 35 Dateien mit 58 Zeilen (aus „Ported from trailhq/Graft …“
     wird ein Satz zur Herkunft der Testvektoren ohne Namen, oder er
     entfällt),
   - auf der Wiki-Seite `code-graph.md`,
   - in der Vergleichstabelle der READMEs,
   - in `cli-reference.md` und `getting-started.md`.

   `benchmarks.md` bleibt Chronik. Die Pläne und Specs gehen mit Punkt 2
   ins Archiv. Der Tor-Test bekommt `graft` mit diesen Ausnahmen:
   `NOTICE.md` und den Generator, die beiden Ideensätze und
   `benchmarks.md`.
6. **Release-Stufe.** `release:none`, falls kein Befehl entfällt. Dabei
   zählt `internal/plancheck` nicht als Befehl.

## Fertig ist 4f, wenn

- der Tor-Test grün ist und ein Grep nach der Suchliste nur in seinen
  Ausnahmen trifft, nach PR D nur noch in Changelog und Benchmarks,
- `parity/stufe-4f.md` jeden Unterschied der Stichprobe mit Entscheidung führt
  (nach PR D im Archiv),
- und v1.0.0 nach der Schwester-Spec als stabiles Release veröffentlicht ist.
