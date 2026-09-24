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
---

Diese Seite beschrieb Scheibe 5 als Absicht. Seit dem Umzug am 2026-09-16
heißt die Scheibe **Stufe 3**, und seit dem 2026-09-23 ist sie in loomux
gebaut (Stufen 3a bis 3c). `loomux reconcile` misst jede registrierte Quelle
gegen ihr Identitätsregister und öffnet je betroffener Seite einen Fall samt
Analysepaket. `loomux cases`, `loomux case` und `loomux approve` entscheiden
ihn: freigeben, mit `--amend` einen eigenen Vorschlag freigeben, mit
`--reject` verwerfen oder mit `--defer` zurückstellen. Die Freigabe prüft die
Evidenzbindung, schreibt Seite, Register, `log.md` und `audit.md` und
committet genau diese Pfade. `loomux serve` holt einen fälligen Abgleich beim
Start und danach täglich nach, und vor jedem `loomux reindex` läuft er
ebenfalls.

**Offen sind zwei Teile, beide Stufe 4:** der Prüfvorschlag des lokalen
Modells — bis dahin kommt ein Fall in `project/loomux` ohne Vorschlag an, und
freigegeben wird ein eigener mit `--amend` — und der `post-merge`-Hook, der
den zweiten Auslöser schreibt; sein Protokoll liest `reconcile` schon. Einen
Wächter gibt es nicht.

## Zwei Arten von Aktualität

| | Technische Aktualität | Fachliche Aktualität |
|---|---|---|
| Frage | Kennt der Index die aktuelle Datei? | Passt das abgeleitete Wissen noch zur Quelle? |
| Zuständig | Indexlauf | Brain Maintenance |

Beide Stände sind auffindbar, und **der veraltete sieht besser aus** —
verdichtet, verlinkt, mit Quellenangabe. Ein Bearbeitungsdatum beweist keine
inhaltliche Aktualität; ein Inhalts-Hash macht Änderungen beweisbar sichtbar.

## Erkennung

Ein **Wächter** meldet Änderungen im Betrieb, entprellt bis der Schreibvorgang
stabil ist — bequem, aber prinzipiell unzuverlässig. Der **tägliche
Sicherheitsabgleich** ist die Wahrheit und fängt, was der Wächter verpasst hat;
zweistufig, mit Zeitstempel und Größe als Vorfilter und Hash nur für Verdächtige.

Wer „täglich" auslöst, ist festgelegt statt geraten: **Der Daemon holt beim
Start nach.** Liegt der letzte vollständige Abgleich länger als 24 Stunden
zurück, läuft er, bevor die erste Anfrage beantwortet wird. Damit hängt die
Aktualität an der Benutzung statt an einem Zeitplan, den niemand überwacht.

**Der Abgleich erzeugt keine Änderungen, sondern Fälle.**

## Drei Gleichzeitigkeiten, festgelegt statt geraten

- **Die Quelle ändert sich erneut, während ein Fall in Prüfung liegt** → der
  offene Fall wird verworfen und neu gebildet. Ein Vorschlag zu einem überholten
  Quellstand ist wertlos.
- **Die Wiki-Seite wird von Hand geändert, während ein Fall offen ist** → beim
  Schreiben prüft der Code den Hash der Zielseite gegen den Stand, auf dem der
  Vorschlag beruht. Weicht er ab, wird **nicht geschrieben**.
- **Mehrere Quellen einer Seite ändern sich** → ein Fall je **Seite**, nicht je
  Quelle; drei Fälle zu derselben Seite erzeugten einander widersprechende
  Vorschläge.

## Der Ablauf

Der technische Weg läuft **immer und sofort** — Index, Graph und Katalog werden
nachgeführt, damit die Suche den aktuellen Dateistand kennt, auch während eine
fachliche Entscheidung offen ist. Danach: Eignungsprüfung (Code) →
Datenschutzprüfung (Code) → Analysepaket (Code) → Prüfvorschlag (Modell,
schreibt nichts) → Evidenzbindung (Code) → Prüfzentrum → **erst nach Freigabe**
Wiki schreiben, Hash und Revision fortschreiben, protokollieren und in einem
isolierten Commit im Vault ablegen. Das Code-Repo bleibt dabei unberührt.

## Die Evidenzbindung als prüfbares Verfahren

„Jede Aussage zeigt auf das Paket" muss deterministisch entscheidbar sein, sonst
ist die Prüfung nicht schreibbar. Deshalb wird das Analysepaket in nummerierte
Segmente zerlegt, das Ausgabeschema verlangt je Behauptung Segmentnummer **und
wörtliches Zitat**, und der Prüfer ist reine Textarbeit: Existiert das Segment?
Kommt das Zitat wortgleich darin vor?

Fällt eine einzelne Behauptung durch, wird sie entfernt und im Fall vermerkt.
Fallen alle durch, wird der Vorschlag verworfen und der Fall manuell geführt —
**kein zweiter Versuch**, denn ein Modell, das beim ersten Mal unbelegt
schreibt, tut es beim zweiten meist auch, nur teurer.

## Zweiter Auslöser: Merge nach `main`

Ein post-merge-Hook erkennt, dass Beschlossenes gelandet ist, und legt einen
Fall „Realisierung prüfen" an. **Das Paket enthält Dateipfade und
Commit-Nachrichten, keinen Quelltext** — damit bleibt der Grundsatz gewahrt,
nicht gegen Codeinhalte abzugleichen, und die Datenschutzprüfung ist einfach.
Die Kandidatenmenge ist klein und deterministisch: nur Seiten, die noch nicht
`implemented` sind. Aktiviert wird das je Repo, nie global — ein Werkzeug, das
ungefragt in fremde Git-Hooks schreibt, wird einmal benutzt.

## Protokolle

`log.md` hält fest, **was** sich geändert hat. `audit.md` hält fest, was
**vorgeschlagen**, was **entschieden** und was **tatsächlich geändert** wurde —
nur so bleibt ein abgelehnter Vorschlag nachvollziehbar. `loomux approve`
schreibt seine Zeile in `log.md` unter die Überschrift des Tages, neueste
zuerst, wie OKF §9 es verlangt; `audit.md` wächst nach unten.

Siehe [Die Wiki-Schicht](wiki-schicht.md) für den anderen Schreibweg; Quelle ist
[Architektur-Design ultra-brain](../sources/architektur-spec.md).
