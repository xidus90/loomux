---
type: Topic
title: Brain Maintenance
description: Wie das System merkt, dass eine Quelle sich geändert hat — Erkennung, Prüfzentrum, Evidenzbindung, Merge-Auslöser.
open_conflicts: 0
realization: in_progress
sources:
  - id: architektur-spec
    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
    doc_id: 01M39G4J14CK311B66GRSAK7HQ
    content_hash: "sha256:c82f573ca64a6b8c53d0f158cb83847865cc8b32c99f710d37c6050fb11ac3ff"
    revision: 3
  - id: stufe-3-spec
    resource: brain://project/loomux/docs/.superpowers/specs/2026-09-19-loomux-stufe-3-design.md
    doc_id: 01M39G4J140NNQD5Q2HRVGKBG2
    content_hash: "sha256:06779f84695b1540346b2f2f73e9f5a0aa48f15ce835048b1efd3e209ea7faa9"
    revision: 1
  - id: stufe-4-spec
    resource: brain://project/loomux/docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md
    doc_id: 01M39NDHWJ80C90J6YB9B9AAQ9
    content_hash: "sha256:a9c28b567436dca62d44c1eab1fb4d39749e9193c1f3bb70709f1dd69e349cec"
    revision: 1
---

Die Pflegeschicht von ultra-brain ist mit **Stufe 3** nach loomux gezogen,
geschnitten in drei Teilstufen: **3a Erkennen** (`loomux reindex`,
`loomux embed`, `loomux reconcile`, `loomux area add`), **3b Entscheiden**
(`loomux cases`, `loomux case`, `loomux approve`) und **3c Pflegen** (unter
anderem die Aufholung in `loomux serve`). Alle drei sind gebaut. Das Ziel der
Stufe: Es gibt keinen Pfad mehr, auf dem loomux Wissen indiziert, ohne den Fall
vorher zu öffnen — oder laut zu sagen, dass keiner aufgehen konnte.

**Was Stufe 3 an Stufe 4 abgab, ist gebaut:** das lokale Modell (4c-1), das
für einen `local_only`-Bereich den Prüfvorschlag schreibt, und der
`post-merge`-Hook (4a-2, `loomux merge-hook`), der den zweiten Auslöser über
`merge-hook record` protokolliert. Offen sind nur ihre Läufe durch einen
Menschen. Einen Wächter
gibt es nicht.

## Zwei Arten von Aktualität

| | Technische Aktualität | Fachliche Aktualität |
|---|---|---|
| Frage | Kennt der Index die aktuelle Datei? | Passt das abgeleitete Wissen noch zur Quelle? |
| Zuständig | Indexlauf | Brain Maintenance |

Beide Stände sind auffindbar, und **der veraltete sieht besser aus** —
verdichtet, verlinkt, mit Quellenangabe. Ein Bearbeitungsdatum beweist keine
inhaltliche Aktualität; ein Inhalts-Hash macht Änderungen beweisbar sichtbar.

## Erkennung

Die Architektur sah zwei Wege vor: einen **Wächter**, der Änderungen im
Betrieb meldet — bequem, aber prinzipiell unzuverlässig —, und den **täglichen
Sicherheitsabgleich**, der die Wahrheit ist. loomux hat nur den zweiten.
`loomux reconcile` scannt je registrierten Bereich zweistufig: Zeitstempel und
Größe als Vorfilter, der Hash nur für Verdächtige.

Wer „täglich" auslöst, ist festgelegt statt geraten: **`loomux serve` holt
nach.** Ist der letzte `reconcile` älter als 24 Stunden, holt der Dienst ihn
nach und hängt den Hinweis an die Antwort. Damit hängt die Aktualität an der
Benutzung statt an einem Zeitplan, den niemand überwacht.

Der zweite Anschluss ist der **Auffangdurchgang**: `loomux reindex` fährt
zuerst `reconcile` über die Registry. Ohne ihn schriebe `reindex` für jede
bewegte Quelle eine neue Identität, und der nächste Abgleich fände die Quelle
identisch zu ihrem Register — die Änderung wäre am Prüftor vorbeigelaufen.
Ein Tor ist der Durchgang trotzdem nicht:

| Zustand | Verhalten |
|---|---|
| Durchgang grün, Fälle eröffnet | Fälle auf stderr, dann wird indiziert |
| Kein Bereich erklärt ein Prüfzentrum | lange Warnung, dann wird indiziert |
| Durchgang scheitert | Exit 1, **nicht** indiziert |

Die Regel dahinter: kein stiller Fortschritt am Tor vorbei. Jeder Zustand, in
dem das Register ohne Abgleich vorrückt, ist laut.

**Der Abgleich erzeugt keine Änderungen, sondern Fälle.** Sie liegen im
Prüfzentrum, dem einen Ort, den `[layout].review` eines Bereichs erklärt.
Erklären zwei Bereiche verschiedene Orte, ist das ein Fehler und kein Vorrang;
erklärt keiner einen, endet `reconcile` mit einem Fehler, der den Schlüssel
nennt. Für loomux ist das `95 Prüfzentrum/` in `brain-knowledge`.

## Drei Gleichzeitigkeiten, festgelegt statt geraten

- **Die Quelle ändert sich erneut, während ein Fall in Prüfung liegt** → der
  stehende Fall wird abgelöst und neu gebildet. Ein Vorschlag zu einem
  überholten Quellstand ist wertlos.
- **Die Wiki-Seite wird von Hand geändert, während ein Fall offen ist** → beim
  Schreiben prüft der Code den Hash der Zielseite gegen den Stand, auf dem der
  Vorschlag beruht. Weicht er ab, wird **nicht geschrieben**.
- **Mehrere Quellen einer Seite ändern sich** → ein Fall je **Seite**, nicht je
  Quelle; drei Fälle zu derselben Seite erzeugten einander widersprechende
  Vorschläge.

## Der Ablauf

`reconcile` lädt die Manifeste aller registrierten Bereiche — ein Bereich ohne
Manifest wird übersprungen, ein unlesbares bricht laut ab —, bestimmt das
Prüfzentrum, scannt, bildet **erst** Quellfälle und **dann**
Zusammenführungsfälle, entdoppelt nach `id` und schreibt `last-run.txt`. Neben
jede Fallakte legt es ein Analysepaket. Eine unlesbare Quelle bricht nichts
ab; sie steht im Bericht.

Die Architektur sah danach einen Prüfvorschlag des Modells vor, der nichts
schreibt. In Stufe 3 **fragte `reconcile` niemanden**; seit 4c-1 bekommt ein
`local_only`-Bereich seinen Vorschlag vom lokalen Modell, wenn es
eingeschaltet ist. Trifft
`reconcile` einen solchen Bereich, öffnet es den Fall mit `manual = true` und
einer Notiz, statt zu scheitern — ein Fall ohne Vorschlag bleibt ein Fall, den
ein Mensch entscheiden kann. Auch außerhalb von `local_only` legt `reconcile`
keinen Vorschlag an: Wer einen Fall freigeben will, reicht den Vorschlag heute
selbst mit `--amend` ein.

Entschieden wird im Prüfzentrum: `loomux cases` listet, was wartet,
`loomux case <id>` zeigt Paket und Vorschlag, `loomux approve` entscheidet —
freigeben, mit `--amend` einen eigenen Vorschlag freigeben, mit `--reject`
verwerfen oder mit `--defer` zurückstellen. Die Reihenfolge einer Freigabe
ist fest: Hash der Zielseite, Stand der Quellen, Evidenzbindung, Seite,
Frontmatter und Register, `log.md` und `audit.md`, Fallakte entfernen, **ein**
Commit. Er geht über einen eigenen Index auf den aktuellen Ref des Tresors;
einen Zweig legt `approve` nicht an, und der Index des Nutzers bleibt, wie er
war. Nach einer geschriebenen Freigabe laufen `reconcile` und `reindex`.

Ein geerbter Fehler ist mitgezogen: `--reject` rückt Revision und Hash der
Seite nicht vor, also eröffnet der nächste Abgleich denselben Fall wieder.
Eine Heilung ist ein Nachtrag der Fusions-Spec; Stufe 4
(`docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`) sieht sie für
das Modell (4c) vor.

## Die Evidenzbindung als prüfbares Verfahren

„Jede Aussage zeigt auf das Paket" muss deterministisch entscheidbar sein, sonst
ist die Prüfung nicht schreibbar. Deshalb wird das Analysepaket in nummerierte
Segmente zerlegt, jede Behauptung des Vorschlags braucht ein **wörtliches
Zitat** aus einem Segment, und der Prüfer ist reine Textarbeit: Existiert das
Segment? Kommt das Zitat wortgleich darin vor? Ohne Beleg wird nicht
angewandt. Ein Belegzaun, der selbst einen Segmentrumpf zitiert, darf nicht
als zweite Behauptung gelesen werden; die Zaunerkennung folgt CommonMark.

Fällt eine einzelne Behauptung durch, wird sie entfernt und im Fall vermerkt.
Fallen alle durch, wird der Vorschlag verworfen und der Fall manuell geführt —
**kein zweiter Versuch**, denn ein Modell, das beim ersten Mal unbelegt
schreibt, tut es beim zweiten meist auch, nur teurer. `apply`, `evidence` und
das Paket rufen dabei weder ein Modell noch das Netz.

## Zweiter Auslöser: Merge

Ein post-merge-Hook erkennt, dass Beschlossenes gelandet ist, und legt einen
Fall „Realisierung prüfen" an. **Das Paket enthält geänderte Dateipfade und
Commit-Betreffe, keinen Quelltext** — damit bleibt der Grundsatz gewahrt,
nicht gegen Codeinhalte abzugleichen, und die Datenschutzprüfung ist einfach.
Die Kandidaten sind klein und deterministisch: nur Seiten, deren
`realization` `planned` oder `in_progress` ist. Ein Merge-Fall tritt hinter
einen stehenden Fall derselben Seite zurück; das Ereignis wartet, bis jener
entschieden ist.

Die Leseseite gibt es seit 3a: `reconcile` liest das Ereignisprotokoll unter
`<state>/maintenance/` und legt verarbeitete Ereignisse ab. Den Schreiber, den
Git-Hook, bringt seit 4a-2 `loomux merge-hook install`. Aktiviert wird er je
Repo, nie global — ein Werkzeug, das ungefragt in fremde Git-Hooks schreibt,
wird einmal benutzt. Die Einwilligung steht als `[maintenance]` (`on_merge`,
`branch`) in der Erklärung des Bereichs; `loomux merge-hook` liest sie.

## Protokolle

`log.md` hält fest, **was** sich geändert hat. `audit.md` hält fest, was
**vorgeschlagen**, was **entschieden** und was **tatsächlich geändert** wurde —
nur so bleibt ein abgelehnter Vorschlag nachvollziehbar; auch `--reject`
schreibt seinen Auditblock. `loomux approve` schreibt seine Zeile in `log.md`
unter die Überschrift des Tages, neueste zuerst, wie OKF §9 es verlangt;
`audit.md` wächst nach unten.

Siehe [Die Wiki-Schicht](wiki-schicht.md) für den anderen Schreibweg. Quellen:
Architektur-Design ultra-brain; Design von Stufe 3 in loomux.
