# Prüfliste: Wiki- und Dokuumzug nach loomux

**Quellen:** `loomux-src/ub` (ultra-brain) und `loomux-src/ul` (ultraloom),
jeweils der versionierte `docs/`-Baum.
**Spec:** [2026-09-14-loomux-fusion-design.md](../specs/2026-09-14-loomux-fusion-design.md),
Abschnitt „Datenumzug", Punkt 1 — Stufe 1b, vor dem Kopieren.
**Regel:** Jede Zeile braucht eine Freigabe. Drei Werte: **behalten** (zieht um),
**aktualisieren** (zieht um, Inhalt wird vorher gerichtet), **Archiv** (bleibt im
alten Repo liegen, wird nicht kopiert).
**Vorschlag** ist mein Vorschlag, nicht die Entscheidung.
**Stand:** Alle 183 Zeilen sind am 2026-09-16 freigegeben — 42 behalten,
22 aktualisieren, 119 Archiv.
Umgesetzt am 2026-09-17 auf dem Zweig `umzug-wiki`, gezählt mit `git ls-files`:
`docs/wiki` trägt 33 Dateien — 24 Inhaltsseiten, `_schema.md`, fünf `index.md`,
`log.md`, `audit.md` und `_identities.tsv` (nur Kopfzeile), die letzten acht neu
angelegt; `docs/.superpowers/plans-ub` 11 (10 Pläne und `HASHES.txt`),
`specs-ub` 8, `specs-ul` 10, `bench-ub` 1. Die Nutzerdoku ist in
`getting-started`, `configuration`, `hooks` und `cli-reference` beider Sprachen
eingearbeitet; die drei Flows `policy`, `session-hooks` und `worktree-mirror`
(je beide Sprachen) kamen am 2026-09-17 nach: in `hooks` Abschnitt 4 und 7–9,
in `configuration` unter `[worktree]` und in `cli-reference` Abschnitt 3 und 5.

**Ziel:** `docs/wiki` im loomux-Repo — so steht es im Manifest (`[layout] wiki`)
und in der Registry (`wiki = ".../loomux/docs/wiki"`). Das Verzeichnis besteht
seit dem Umzug; der post-edit-Hook lintet die bearbeitete Seite, und
`loomux wiki-gate` prüft das Bündel, wenn man es aufruft — kein Hook und kein
Tor ruft es heute.

Nach der Freigabe wird kopiert: die Inhaltsseiten, Bereich `project/loomux`,
Links `brain://project/ultra-brain/…` und `brain://project/ultraloom/…`
umgeschrieben. Der Suchindex folgt erst mit `reindex` — der Befehl entsteht in
Stufe 3.

**Entscheidung des Nutzers (2026-09-16):** Was neu erzeugt werden kann, zieht
nicht mit, und was nur die alten Systeme beschreibt, ebenso wenig. Das trifft
die fünf `index.md` und `_identities.tsv` (Artefakte: `render_catalog` und
`reindex` schreiben sie), `audit.md` (leerer Platzhalter) und `log.md`
(Protokoll der alten Bundle-Arbeit). Folge: das Protokoll beginnt in loomux
leer, und der erste `reindex` prägt neue `doc_id`s.

**Berichtigung (2026-09-17), an der Referenz nachgelesen:** Die fünf
`index.md` unter `docs/wiki` schreibt `reindex` nicht. `_write_catalogs`
überspringt jedes Verzeichnis im eigenen Bündel („A bundle owns its catalog",
`loomux-src/ub/src/brain/cli.py:203-206`), und `render_catalog` hat keinen
anderen Aufrufer. Die fünf Kataloge wurden beim Umzug einmal von Hand in der
Form von `render_catalog` geschrieben; wer sie ab Stufe 3 pflegt, ist eine
offene Entscheidung des Nutzers. `_identities.tsv` wiederum schreibt `reindex`
in das Artefaktverzeichnis des Bereichs (`cli.py:142,164`), und das ist bei
einem beschreibbaren Bereich dessen `path` (`registry.py:129`), für
`project/loomux` also die Repo-Wurzel — nicht `docs/wiki`
(`docs/wiki/_schema.md`, „Wo das Identitätsregister liegt").

## Vor den Zeilen: zwei Entscheidungen

**A — die Quellenkette.** Jede verdichtete Seite trägt in der Frontmatter
`resource`, `content_hash` und `revision` ihrer Quelle. Zwölf Ziele werden
zitiert: die alte Architektur-Spec, zehn Pläne und `bench/2c1/entscheidung-46.md`
(außerhalb von `docs/`, damit außerhalb der Ausgangsliste). Elf der zwölf
Prüfsummen stimmen heute noch, nur die der Architektur-Spec nicht.

**Entschieden (Nutzer, 2026-09-16):** Die zitierten Quellen ziehen mit, damit
die Kette prüfbar bleibt — byte-gleich, ohne Linkumschreibung in ihnen selbst;
umgeschrieben wird nur die `resource`-Zeile der zitierenden Seite. Der
`content_hash` läuft über den Inhalt, nicht über den Pfad, und überlebt den
Umzug darum. Zielort (Nutzer): `docs/.superpowers/plans-ub/` und `specs-ub/`, also dieselbe
Ebene mit einem Suffix am Verzeichnis; die zwölfte Quelle aus `bench/` landet
in `docs/.superpowers/bench-ub/`. Loomux' eigene Pläne bleiben unvermischt.

**B — die Kataloge.** Entschieden: Kataloge, Register, Audit und Protokoll
ziehen nicht mit (siehe oben). Damit kollidiert nichts mehr, und vom
ul-Bundle bleibt nichts übrig — es besteht nur aus diesen fünf Dateien.
~~Offen bleibt allein `_schema.md`: es ist keine erzeugte Datei, sondern das
Regelwerk des Bundles, und loomux braucht eines.~~ Erledigt:
`docs/wiki/_schema.md` ist das Regelwerk des Bundles.

**Berichtigung der Spec:** sie nennt „`_identities.tsv` (10 und 1)". Beide
Register enthalten nur die Kopfzeile. Die `doc_id`s in der Frontmatter der
Seiten gehören den zitierten Quellen (`sources[]`), nicht den Seiten; eigene
`doc_id`s der Seiten prägt erst `reindex` in Stufe 3. Die Zeile ist in der
Fusions-Spec richtiggestellt.

## 1. Wiki-Bundle ultra-brain (33 Dateien)

| Datei | Zeilen | Stand | Vorschlag | Begründung | Freigabe |
|---|---:|---|---|---|---|
| `ub/docs/wiki/topics/wiki-schicht.md` | 113 | 2026-08-30 | **aktualisieren** | Lint-Regeln sind in loomux andere; die Schlusszeile nennt „Architektur-Design ultra-brain" und bleibt, weil sie das Dokument benennt (Task 3) | freigegeben 2026-09-16 |
| `ub/docs/wiki/topics/datenmodell-und-bereiche.md` | 114 | 2026-08-30 | **aktualisieren** | Manifestnamen, Orte und die Regel `wrong-direction` sind in loomux andere; die Schlusszeile nennt „Architektur-Design ultra-brain" und bleibt, weil sie das Dokument benennt (Task 3) | freigegeben 2026-09-16 |
| `ub/docs/wiki/topics/suche-und-profile.md` | 140 | 2026-08-30 | **aktualisieren** | Ketten und Messwerte sind in loomux andere; die Schlusszeile nennt „Architektur-Design ultra-brain" und bleibt, weil sie das Dokument benennt (Task 3) | freigegeben 2026-09-16 |
| `ub/docs/wiki/topics/index.md` | 14 | 2026-08-30 | **Archiv** | Artefakt in der Form von `render_catalog`; `reindex` schreibt die Kataloge des eigenen Bündels nicht (`cli.py:203-206`), sie wurden beim Umzug einmal von Hand neu geschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/index.md` | 16 | 2026-08-30 | **Archiv** | Artefakt in der Form von `render_catalog`; `reindex` schreibt die Kataloge des eigenen Bündels nicht (`cli.py:203-206`), sie wurden beim Umzug einmal von Hand neu geschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/sources/index.md` | 17 | 2026-08-30 | **Archiv** | Artefakt in der Form von `render_catalog`; `reindex` schreibt die Kataloge des eigenen Bündels nicht (`cli.py:203-206`), sie wurden beim Umzug einmal von Hand neu geschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/_identities.tsv` | 1 | 2026-08-30 | **Archiv** | Artefakt: `reindex` führt das Register des Bereichs in dessen Artefaktverzeichnis, bei `project/loomux` die Repo-Wurzel, nicht im Bündel (`cli.py:142,164`, `_schema.md`); enthält ohnehin nur die Kopfzeile | freigegeben 2026-09-16 |
| `ub/docs/wiki/entities/okf.md` | 34 | 2026-08-30 | **behalten** | Begriffsseite ohne Bezug auf die Werkzeugteilung | freigegeben 2026-09-16 |
| `ub/docs/wiki/_schema.md` | 42 | 2026-08-30 | **aktualisieren** | Zeile 42 nennt `project/ultra-brain` als Bereich | freigegeben 2026-09-16 |
| `ub/docs/wiki/log.md` | 46 | 2026-08-30 | **Archiv** | Protokoll der alten Bundle-Arbeit: beschreibt Scheiben und Quellen, die nicht mitziehen | freigegeben 2026-09-16 |
| `ub/docs/wiki/sources/plan-scheibe-2b-messwerk.md` | 46 | 2026-08-30 | **behalten** | Verdichtete Quelle; nur die `brain://`-Zeile wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/sources/plan-scheibe-2a-nacharbeit.md` | 47 | 2026-08-30 | **behalten** | Verdichtete Quelle; nur die `brain://`-Zeile wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/sources/plan-scheibe-2c1-daemon.md` | 48 | 2026-08-30 | **behalten** | Verdichtete Quelle; nur die `brain://`-Zeile wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/syntheses/gegenpruefung-vor-jeder-designempfehlung.md` | 48 | 2026-08-30 | **behalten** | Seite zieht um; nur die `brain://`-Zeile wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/audit.md` | 4 | 2026-08-30 | **Archiv** | Leerer Platzhalter („gefüllt ab Scheibe 5"); die Wartungsschicht schreibt ihn ab Stufe 3 selbst | freigegeben 2026-09-16 |
| `ub/docs/wiki/entities/qmd.md` | 56 | 2026-08-30 | **aktualisieren** | Paritätszeilen 1–2: Profile und Rangfolge sind in loomux andere | freigegeben 2026-09-16 |
| `ub/docs/wiki/sources/entscheidungen-scheibe-2a.md` | 58 | 2026-08-30 | **behalten** | Verdichtete Quelle; nur die `brain://`-Zeile wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/sources/plan-scheibe-2a-suchkette.md` | 58 | 2026-08-30 | **behalten** | Verdichtete Quelle; nur die `brain://`-Zeile wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/entities/brain-daemon.md` | 61 | 2026-08-30 | **aktualisieren** | Paritätszeile 49: loomux fragt nie einen brain-Daemon | freigegeben 2026-09-16 |
| `ub/docs/wiki/sources/architektur-spec.md` | 62 | 2026-08-30 | **aktualisieren** | Die Quelle ist die einzige, deren `content_hash` nicht mehr stimmt — dafür steht ein Konfliktkasten; der Titel „Architektur-Design ultra-brain" bleibt, weil er das Dokument benennt (Task 3) | freigegeben 2026-09-16 |
| `ub/docs/wiki/topics/datenschutz-und-kanaele.md` | 62 | 2026-08-30 | **aktualisieren** | Nur die `resource`-Zeile wurde gerichtet; die Schlusszeile nennt „Architektur-Design ultra-brain" und bleibt, weil sie das Dokument benennt (Task 3) | freigegeben 2026-09-16 |
| `ub/docs/wiki/syntheses/warum-fast-die-vorgabe-bleibt.md` | 65 | 2026-08-30 | **aktualisieren** | Zeile 40 nennt `project/ultra-brain` und die Datei `bench/2c1/entscheidung-46.md` außerhalb von `docs/` | freigegeben 2026-09-16 |
| `ub/docs/wiki/sources/plan-pruefkorpus-v1.md` | 66 | 2026-08-30 | **behalten** | Verdichtete Quelle; nur die `brain://`-Zeile wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/topics/abnahmen-und-echte-umgebung.md` | 70 | 2026-08-30 | **behalten** | Keine Prosa-Nennung der alten Namen, nur `brain://`-Quellen | freigegeben 2026-09-16 |
| `ub/docs/wiki/topics/architektur-grundsaetze.md` | 70 | 2026-08-30 | **aktualisieren** | Nur die `resource`-Zeile wurde gerichtet; die Schlusszeile nennt „Architektur-Design ultra-brain" und bleibt, weil sie das Dokument benennt (Task 3) | freigegeben 2026-09-16 |
| `ub/docs/wiki/sources/plan-scheibe-2c2-mcp-fronten.md` | 71 | 2026-08-30 | **behalten** | Verdichtete Quelle; nur die `brain://`-Zeile wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/topics/scheiben-und-abnahme.md` | 72 | 2026-08-30 | **aktualisieren** | Scheiben heißen in loomux Stufen, die Abnahme ist enger; die Schlusszeile nennt „Architektur-Design ultra-brain" und bleibt, weil sie das Dokument benennt (Task 3) | freigegeben 2026-09-16 |
| `ub/docs/wiki/sources/plan-scheibe-1-indexer.md` | 74 | 2026-08-30 | **behalten** | Verdichtete Quelle; nur die `brain://`-Zeile wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/sources/abnahme-scheibe-2a.md` | 78 | 2026-08-30 | **behalten** | Verdichtete Quelle; nur die `brain://`-Zeile wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/syntheses/index.md` | 8 | 2026-08-30 | **Archiv** | Artefakt in der Form von `render_catalog`; `reindex` schreibt die Kataloge des eigenen Bündels nicht (`cli.py:203-206`), sie wurden beim Umzug einmal von Hand neu geschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/sources/plan-scheibe-0-fundament.md` | 95 | 2026-08-30 | **behalten** | Verdichtete Quelle; nur die `brain://`-Zeile wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/wiki/topics/brain-maintenance.md` | 95 | 2026-08-30 | **aktualisieren** | Die Pflege ist in loomux Stufe 3 und nicht gebaut; die Schlusszeile nennt „Architektur-Design ultra-brain" und bleibt, weil sie das Dokument benennt (Task 3) | freigegeben 2026-09-16 |
| `ub/docs/wiki/entities/index.md` | 9 | 2026-08-30 | **Archiv** | Artefakt in der Form von `render_catalog`; `reindex` schreibt die Kataloge des eigenen Bündels nicht (`cli.py:203-206`), sie wurden beim Umzug einmal von Hand neu geschrieben | freigegeben 2026-09-16 |

## 2. Wiki-Bundle ultraloom (5 Dateien)

| Datei | Zeilen | Stand | Vorschlag | Begründung | Freigabe |
|---|---:|---|---|---|---|
| `ul/docs/wiki/_identities.tsv` | 1 | 2026-09-06 | **Archiv** | Gerüst ohne Seiten (Katalog drei Zeilen, Log und Audit je vier, Register nur Kopfzeile); die Kataloge des ub-Bundles werden übernommen | freigegeben 2026-09-16 |
| `ul/docs/wiki/_schema.md` | 34 | 2026-09-06 | **Archiv** | Gerüst ohne Seiten (Katalog drei Zeilen, Log und Audit je vier, Register nur Kopfzeile); die Kataloge des ub-Bundles werden übernommen | freigegeben 2026-09-16 |
| `ul/docs/wiki/index.md` | 3 | 2026-09-06 | **Archiv** | Gerüst ohne Seiten (Katalog drei Zeilen, Log und Audit je vier, Register nur Kopfzeile); die Kataloge des ub-Bundles werden übernommen | freigegeben 2026-09-16 |
| `ul/docs/wiki/audit.md` | 4 | 2026-09-06 | **Archiv** | Gerüst ohne Seiten (Katalog drei Zeilen, Log und Audit je vier, Register nur Kopfzeile); die Kataloge des ub-Bundles werden übernommen | freigegeben 2026-09-16 |
| `ul/docs/wiki/log.md` | 4 | 2026-09-06 | **Archiv** | Gerüst ohne Seiten (Katalog drei Zeilen, Log und Audit je vier, Register nur Kopfzeile); die Kataloge des ub-Bundles werden übernommen | freigegeben 2026-09-16 |

## 3. Nutzerdoku ultra-brain (3 Dateien)

| Datei | Zeilen | Stand | Vorschlag | Begründung | Freigabe |
|---|---:|---|---|---|---|
| `ub/docs/produkt-design.md` | 132 | 2026-09-06 | **behalten** | Entscheidungen der Selbsteinrichtung; Rohquelle für `loomux init` (Stufe 4), zieht nach `specs-ub/` | freigegeben 2026-09-16 |
| `ub/docs/installation.md` | 385 | 2026-09-06 | **aktualisieren** | Einrichtung des alten Werkzeugs (uv, PATH, `brain init`); brauchbar sind „Einen Bereich einrichten", „Wo was liegt" und „Was dabei schiefging" — einarbeiten in `getting-started` und `configuration` | freigegeben 2026-09-16 |
| `ub/docs/hooks.md` | 207 | 2026-09-14 | **aktualisieren** | Prüfkette des alten Repos; CRLF-Abschnitt und „Die Schreibschranke und das Memory der Agenten" gehören in loomux' `hooks` bzw. `configuration` | freigegeben 2026-09-16 |

## 4. Nutzerdoku ultraloom (12 Dateien)

| Datei | Zeilen | Stand | Vorschlag | Begründung | Freigabe |
|---|---:|---|---|---|---|
| `ul/docs/flows/verify-until-green.md` | 1013 | 2026-08-25 | **Archiv** | `verify_until_green` gehört laut Fusions-Spec zum Flow-Folgeprojekt (ulflow M2/M3), nicht zu loomux | freigegeben 2026-09-16 |
| `ul/docs/flows/verify-until-green.de.md` | 1031 | 2026-08-25 | **Archiv** | Deutsche Fassung, gleiche Begründung | freigegeben 2026-09-16 |
| `ul/docs/flows/policy.md` | 79 | 2026-08-31 | **aktualisieren** | Beschreibt den Entscheidungsweg der Policy, den loomux heute geht — Pfade und Dateinamen sind andere | freigegeben 2026-09-16 |
| `ul/docs/flows/policy.de.md` | 83 | 2026-08-31 | **aktualisieren** | Deutsche Fassung, gleiche Behandlung | freigegeben 2026-09-16 |
| `ul/docs/flows/session-hooks.md` | 227 | 2026-09-10 | **aktualisieren** | Die fünf Sitzungshooks kommen erst mit Stufe 2; Text beschreibt sie am alten Binary | freigegeben 2026-09-16 |
| `ul/docs/flows/session-hooks.de.md` | 244 | 2026-09-10 | **aktualisieren** | Deutsche Fassung, gleiche Behandlung | freigegeben 2026-09-16 |
| `ul/docs/flows/worktree-mirror.md` | 384 | 2026-09-10 | **aktualisieren** | Worktree-Spiegelung läuft in loomux (Stufe 1a); Befehlsnamen und Pfade richten | freigegeben 2026-09-16 |
| `ul/docs/flows/worktree-mirror.de.md` | 400 | 2026-09-10 | **aktualisieren** | Deutsche Fassung, gleiche Behandlung | freigegeben 2026-09-16 |
| `ul/docs/hooks.de.md` | 118 | 2026-09-11 | **aktualisieren** | Deutsche Fassung von `ul/docs/hooks.md`, gleiche Behandlung | freigegeben 2026-09-16 |
| `ul/docs/hooks.md` | 118 | 2026-09-11 | **aktualisieren** | Hook-Architektur des alten Binaries; neu ist nur die Tabelle der Sprachstacks und Werkzeugketten — der Rest steht in loomux' `hooks` | freigegeben 2026-09-16 |
| `ul/docs/benchmarks.md` | 198 | 2026-09-11 | **Archiv** | Messungen des alten Binaries; loomux führt eine eigene chronologische Reihe und zitiert die alten Werte, wo sie die Grundlinie sind | freigegeben 2026-09-16 |
| `ul/docs/benchmarks.de.md` | 204 | 2026-09-11 | **Archiv** | Deutsche Fassung, gleiche Begründung | freigegeben 2026-09-16 |

## 5. Arbeitspapiere ultra-brain (71 Dateien)

| Datei | Zeilen | Stand | Vorschlag | Begründung | Freigabe |
|---|---:|---|---|---|---|
| `ub/docs/.superpowers/plans/2026-08-18-scheibe-0-fundament.md` | 672 | 2026-08-19 | **behalten** | Zitierte Rohquelle einer Wiki-Seite: zieht byte-gleich mit, damit der `content_hash` stimmt; nur die `resource`-Zeile der zitierenden Seite wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-20-scheibe-2a-abnahme.md` | 1058 | 2026-08-20 | **behalten** | Zitierte Rohquelle einer Wiki-Seite: zieht byte-gleich mit, damit der `content_hash` stimmt; nur die `resource`-Zeile der zitierenden Seite wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-19-scheibe-1-indexer.md` | 1542 | 2026-08-20 | **behalten** | Zitierte Rohquelle einer Wiki-Seite: zieht byte-gleich mit, damit der `content_hash` stimmt; nur die `resource`-Zeile der zitierenden Seite wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-20-scheibe-2a-suchkette.md` | 2234 | 2026-08-20 | **behalten** | Zitierte Rohquelle einer Wiki-Seite: zieht byte-gleich mit, damit der `content_hash` stimmt; nur die `resource`-Zeile der zitierenden Seite wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-20-scheibe-2a-entscheidungen.md` | 65 | 2026-08-20 | **behalten** | Zitierte Rohquelle einer Wiki-Seite: zieht byte-gleich mit, damit der `content_hash` stimmt; nur die `resource`-Zeile der zitierenden Seite wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-20-scheibe-2a-nacharbeit.md` | 713 | 2026-08-20 | **behalten** | Zitierte Rohquelle einer Wiki-Seite: zieht byte-gleich mit, damit der `content_hash` stimmt; nur die `resource`-Zeile der zitierenden Seite wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-21-scheibe-2c1-daemon.md` | 1385 | 2026-08-21 | **behalten** | Zitierte Rohquelle einer Wiki-Seite: zieht byte-gleich mit, damit der `content_hash` stimmt; nur die `resource`-Zeile der zitierenden Seite wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-21-scheibe-2b-messwerk.md` | 1693 | 2026-08-21 | **behalten** | Zitierte Rohquelle einer Wiki-Seite: zieht byte-gleich mit, damit der `content_hash` stimmt; nur die `resource`-Zeile der zitierenden Seite wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-21-pruefkorpus-design.md` | 209 | 2026-08-21 | **behalten** | Methode des Fallkorpus, den 1b-1 benutzt | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-21-scheibe-2b-messwerk-design.md` | 217 | 2026-08-21 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-21-pruefkorpus-v1.md` | 984 | 2026-08-21 | **behalten** | Zitierte Rohquelle einer Wiki-Seite: zieht byte-gleich mit, damit der `content_hash` stimmt; nur die `resource`-Zeile der zitierenden Seite wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-22-scheibe-2c2-mcp-fronten.md` | 1232 | 2026-08-22 | **behalten** | Zitierte Rohquelle einer Wiki-Seite: zieht byte-gleich mit, damit der `content_hash` stimmt; nur die `resource`-Zeile der zitierenden Seite wird umgeschrieben | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-21-scheibe-2c1-daemon-design.md` | 451 | 2026-08-22 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-23-briefing-scheibe-3.md` | 108 | 2026-08-23 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-23-pre-commit-kette.md` | 143 | 2026-08-23 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-22-scheibe-2c2-mcp-fronten-design.md` | 397 | 2026-08-23 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-23-scheibe-3-wiki-schicht.md` | 1284 | 2026-08-24 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-24-wiki-verbund.md` | 897 | 2026-08-24 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-25-scheibe-4-import.md` | 1629 | 2026-08-25 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-24-scheibe-4-import-design.md` | 367 | 2026-08-25 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-25-typkatalog-code.md` | 986 | 2026-08-25 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-25-typkatalog-und-migration-design.md` | 386 | 2026-08-26 | **behalten** | Typkatalog gehört zu Stufe 3 | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-27-scheibe-5-brain-maintenance.md` | 1382 | 2026-08-27 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-28-bereichsfremdes-register.md` | 298 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-27-scheibe-5-brain-maintenance-design.md` | 398 | 2026-08-28 | **behalten** | Brain-Pflege ist Stufe 3 | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-28-brain-init.md` | 60 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-29-ultra-brain-projektordner-und-doku.md` | 39 | 2026-08-29 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-29-ultra-brain-projektordner-und-doku-design.md` | 54 | 2026-08-29 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-08-29-projekt-wiki-im-repo.md` | 945 | 2026-08-29 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-28-brain-init-design.md` | 126 | 2026-08-30 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-24-wiki-verbund-design.md` | 328 | 2026-08-30 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-29-projekt-wiki-im-repo-design.md` | 634 | 2026-08-30 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-28-bereichsfremdes-register-design.md` | 137 | 2026-09-01 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-23-scheibe-3-wiki-schicht-design.md` | 342 | 2026-09-01 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-01-pruefkatalog-in-go.md` | 850 | 2026-09-03 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-04-scheibe-0-pruefbestand.md` | 1546 | 2026-09-04 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-04-python-nach-go-migration-design.md` | 251 | 2026-09-04 | **behalten** | Begründet den Portierweg, den 1b fortsetzt | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-01-pruefkatalog-in-go-design.md` | 417 | 2026-09-04 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-04-scheibe-1-code-achse.md` | 640 | 2026-09-04 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-05-scheibe-7-zielbilder/realdata.json` | 0 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-05-scheibe-7-zielbilder/zwei-gegen-drei.html` | 107 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-05-scheibe-7-zielbilder/renderer-benchmark.html` | 113 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-05-scheibe-7-zielbilder/uebersicht-layout.html` | 118 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-05-scheibe-7-zielbilder/zoom-prototyp.html` | 122 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/design/2026-09-05-scheibe-7-canvas/Pruefzentrum.dc.html` | 171 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/design/2026-09-05-scheibe-7-canvas/Scanner.dc.html` | 184 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/design/2026-09-05-scheibe-7-canvas/Seite.dc.html` | 184 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/design/2026-09-05-scheibe-7-canvas/Main.dc.html` | 201 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/design/2026-09-05-scheibe-7-canvas/Fall.dc.html` | 226 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/design/2026-09-05-scheibe-7-canvas/canvas.json` | 34 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/design/2026-09-05-scheibe-7-canvas/Galaxie.dc.html` | 458 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-05-scheibe-2-messscheibe.md` | 478 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-05-scheibe-7-zielbilder/naehe-echt.html` | 78 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-05-scheibe-7-zielbilder/drei-schaumodi.html` | 96 | 2026-09-05 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-06-pruefung-scheibe-2-messscheibe.md` | 194 | 2026-09-06 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-06-scheibe-5-suche.md` | 197 | 2026-09-06 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-06-scheibe-4-neighbors-graph.md` | 225 | 2026-09-06 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-05-scheibe-3-catalog-read.md` | 274 | 2026-09-06 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-06-scheibe-7a-1-datenweg-und-uebersicht.md` | 3913 | 2026-09-06 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-07-scheibe-8-cases.md` | 113 | 2026-09-07 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-07-scheibe-7-mcp.md` | 317 | 2026-09-07 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-07-scheibe-6-index-embed.md` | 332 | 2026-09-07 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-07-scheibe-6-lokales-modell.md` | 1798 | 2026-09-08 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md` | 2051 | 2026-09-08 | **behalten** | Referenzarchitektur der Parität bis Stufe 3 | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-05-scheibe-7-web-app-design.md` | 1181 | 2026-09-11 | **Archiv** | Ist in loomux' eigener Web-OS-Spec (`2026-09-14-loomux-web-os-design.md`) aufgegangen | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-11-brain-install-folgt-master.md` | 1187 | 2026-09-11 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-11-scheibe-7a-dritte-tiefe.md` | 1708 | 2026-09-11 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-11-brain-install-folgt-master-design.md` | 287 | 2026-09-11 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-05-scheibe-6-lokales-modell-design.md` | 460 | 2026-09-11 | **behalten** | Lokales Modell ist Stufe 4 | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/plans/2026-09-13-schranke-memory-offen.md` | 784 | 2026-09-13 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ub/docs/.superpowers/specs/2026-09-13-schranke-memory-offen-design.md` | 210 | 2026-09-14 | **behalten** | Schranke, in loomux weiter gültig | freigegeben 2026-09-16 |

| `ub/bench/2c1/entscheidung-46.md` | 97 | 2026-08-22 | **behalten** | Zitierte Rohquelle von `syntheses/warum-fast-die-vorgabe-bleibt.md`; liegt außerhalb von `docs/` und stand darum nicht im Ausgangsbestand | freigegeben 2026-09-16 |
## 6. Arbeitspapiere ultraloom (58 Dateien)

| Datei | Zeilen | Stand | Vorschlag | Begründung | Freigabe |
|---|---:|---|---|---|---|
| `ul/docs/.superpowers/plans/2026-08-21-teilprojekt-1-kern.md` | 4364 | 2026-08-21 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-22-pruefkette-reihenfolge.md` | 2164 | 2026-08-22 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-22-teilprojekt-2-verify-until-green.md` | 2222 | 2026-08-22 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-22-teilprojekt-2-verify-until-green-design.md` | 507 | 2026-08-23 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-22-pruefkette-reihenfolge-design.md` | 510 | 2026-08-23 | **behalten** | Prüfkette `[verify]`, Stufe 2 | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-21-ultraloom-kern-design.md` | 654 | 2026-08-23 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-23-guard-basis-commit.md` | 918 | 2026-08-23 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-23-guard-basis-commit-design.md` | 157 | 2026-08-24 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-24-review-defects.md` | 240 | 2026-08-24 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-24-agent-settings-sources-design.md` | 253 | 2026-08-24 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-24-agent-settings-sources.md` | 522 | 2026-08-24 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-25-sitzungs-hooks.md` | 1289 | 2026-08-25 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-25-policy-baukasten.md` | 1373 | 2026-08-25 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-25-sitzungs-hooks-payloads.md` | 144 | 2026-08-25 | **behalten** | Payload-Tatsachen, host-unabhängig | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-25-sitzungs-hooks-design.md` | 291 | 2026-08-25 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-25-policy-baukasten-design.md` | 293 | 2026-08-25 | **behalten** | Policy-Baukasten des Wächters | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-25-commit-sprachpruefung.md` | 816 | 2026-08-25 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-27-sprachen-erweitern-design.md` | 137 | 2026-08-27 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-27-toolchain-design.md` | 155 | 2026-08-27 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-27-docs-check-design.md` | 187 | 2026-08-27 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-25-commit-sprachpruefung-design.md` | 226 | 2026-08-27 | **behalten** | `commit-msg --language`, Stufe 2 | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-27-sprachen-erweitern.md` | 230 | 2026-08-27 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-27-generierte-dateien-design.md` | 134 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-28-installer-kern.md` | 1869 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-28-dokumentvorlagen-design.md` | 38 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-28-installer-kern-design.md` | 410 | 2026-08-28 | **behalten** | `loomux init`, Stufe 4 | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-28-agents-and-lifecycle-hooks-design.md` | 46 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-28-agent-ausstattung.md` | 50 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-28-guard-nativ.md` | 51 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-28-agent-ausstattung-design.md` | 54 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-28-dokumentvorlagen.md` | 58 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-28-agents-and-lifecycle-hooks.md` | 66 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-28-guard-nativ-design.md` | 66 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-28-hook-bestand-design.md` | 71 | 2026-08-28 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-29-legacy-bereinigung-und-entflechtung-design.md` | 104 | 2026-08-29 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-29-ecosystem-integration-and-native-tooling-design.md` | 131 | 2026-08-29 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-28-tooling-check-and-install-design.md` | 43 | 2026-08-29 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-28-tooling-check-and-install.md` | 55 | 2026-08-29 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-29-ecosystem-integration-and-native-tooling.md` | 596 | 2026-08-29 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-28-hook-bestand.md` | 60 | 2026-08-29 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-29-git-repos-hook-and-tooling-inventory.md` | 93 | 2026-08-29 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-29-legacy-bereinigung-und-entflechtung.md` | 95 | 2026-08-29 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-30-cpp-stack-and-multi-tooling-design.md` | 137 | 2026-08-30 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-30-selective-post-edit-hook-dispatch.md` | 392 | 2026-08-30 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-08-30-cpp-stack-and-multi-tooling.md` | 473 | 2026-08-30 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-30-selective-post-edit-hook-dispatch-design.md` | 77 | 2026-08-30 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-09-01-audit-nacharbeit-design.md` | 394 | 2026-09-01 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-09-02-codebase-cleanup.md` | 128 | 2026-09-02 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-09-07-worktree-mirror-design.md` | 225 | 2026-09-08 | **behalten** | Worktree-Spiegelung läuft in loomux | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-09-07-worktree-mirror.md` | 2812 | 2026-09-08 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-09-10-antigravity-hook-messung.md` | 151 | 2026-09-10 | **behalten** | Messgrundlage der Hook-Zielwerte | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-09-10-go-hooks-stufe-0-und-1.md` | 2321 | 2026-09-10 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-09-10-go-hooks-drei-hosts-design.md` | 291 | 2026-09-10 | **behalten** | Stufen 2–5 laufen in loomux weiter | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-09-10-wiki-flottenstandard-design.md` | 351 | 2026-09-11 | **behalten** | Stufen 3–5 gehen in `loomux init` auf | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/specs/2026-08-21-teilprojekt-2-backlog.md` | 492 | 2026-09-11 | **behalten** | Offener Backlog, noch nicht abgearbeitet | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-09-11-wiki-flottenstandard-stufe-0.md` | 553 | 2026-09-11 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-09-11-wiki-flottenstandard-stufe-1.md` | 888 | 2026-09-11 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
| `ul/docs/.superpowers/plans/2026-09-13-wiki-flottenstandard-stufe-2.md` | 922 | 2026-09-13 | **Archiv** | Arbeitspapier der alten Repos; loomux hat eigene Specs und Pläne | freigegeben 2026-09-16 |
