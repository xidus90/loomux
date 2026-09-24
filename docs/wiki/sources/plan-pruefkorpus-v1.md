---
type: Source
title: Plan Prüfkorpus v1
description: Ein eingecheckter Prüfbestand aus 100 Notizen und 50 Fragen — und die Regel, was seine Zahlen nie beantworten dürfen.
open_conflicts: 0
realization: implemented
implemented_in: c3e0661
sources:
  - id: plan-pruefkorpus-v1
    resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-21-pruefkorpus-v1.md
    doc_id: 01M39G4J14AGHVYCB236STY4DY
    content_hash: "sha256:2a9358ddace4085a0ce389fda921b968fa17964e5688a741f194b50be41c301d"
    revision: 1
---

Ein versionierter, eingecheckter Prüfbestand: **100 Notizen, 50 Fragen, 10
Themen** — damit eine Verschlechterung der Suchkette auffällt, bevor sie im
Alltag weh tut. Der Korpus ist Daten, kein Code; dazu kommen nur ein Prüfer,
der die Regeln des Standes maschinell durchsetzt, und ein Schalter, der ihn in
einen Wegwerf-Zustandsraum einhängt. **Am Messwerk selbst ändert sich nichts.**

## Die Regel, die über allem steht

**Korpuszahlen tragen nie eine Architekturentscheidung.** Sie beantworten „ist
es schlechter geworden", nie „was ist gut".

## Der feste Zuschnitt

Genau 100 Notizen, genau 50 Fragen, genau 10 Themen — **keine Ausnahme**.
Verteilung 13 exakt / 13 Umschreibung / 10 gemischt / 14 sprachübergreifend,
davon mindestens fünf in der Gegenrichtung. Jedes Thema hat zehn Notizen und
**ein benanntes Nachbarthema, mit dem es Vokabular teilt** — sonst prüft der
Bestand nur, ob die Suche zehn unverwechselbare Inseln auseinanderhält.

Ein neuer Stand ist **additiv**: aus `v1` wird nie etwas geändert oder entfernt,
und ein späterer Stand bringt neue Themen. Zu jeder Notiz stehen Quelle,
Abrufdatum und Lizenz fest — fremder Text ohne Herkunftsangabe kommt nicht ins
Repository.

Bemerkenswert ist auch, was der Plan **nicht** tut: Er führt keinen zweiten Weg
neben der bestehenden Verteilung ein. Steht die Vorgabe noch auf der alten Zahl,
ist das die erste Zeile, die zu ändern ist — nicht ein Schalter daneben.

## Der Ausgangswert

**43/50 auf dem Vorgabeprofil**, gemessen am 21. August 2026 (13/13 exakt,
11/13 Umschreibung, 8/10 gemischt, 11/14 sprachübergreifend).

Der eigentliche Inhalt des Ausgangswerts ist aber nicht die Zahl, sondern die
mit abgelegte **Fehlliste**: Ohne sie ließe sich beim nächsten Lauf sagen
„schlechter", aber nicht „diese sieben waren es damals, diese neun sind es
jetzt".

Zwei Beobachtungen dazu:

- **Exakt 13/13 ist keine Leistung der Suchmaschine**, sondern die Gegenprobe
  auf die Eindeutigkeitsprüfung jedes Bezeichners vor der Aufnahme.
- **Sprachübergreifend ist der Korpus schwerer als der echte Bestand** (11/14
  gegen 13/14) — und das mit Absicht: Seine Fragen teilen mit ihrer Zielnotiz
  kein Inhaltswort. *Ein künstlicher Bestand, der leichter wäre als der echte,
  würde eine Verschlechterung erst bemerken, wenn sie im Alltag längst weh tut.*
- **Das Hybridprofil war auf diesem Rechner nicht messbar** — drei Versuche,
  drei Abstürze im Reranking. Damit bestätigt der Korpus an einem völlig anderen
  Bestand, was die Messung am echten Vault zeigte, und löst die Beobachtung von
  der Eigenart einer bestimmten Notizsammlung
  ([qmd](../entities/qmd.md), [Suche, Profile und Messwerte](../topics/suche-und-profile.md)).
