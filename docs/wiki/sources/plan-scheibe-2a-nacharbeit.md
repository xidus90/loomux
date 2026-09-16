---
type: Source
title: Plan Scheibe 2a — Nacharbeit nach der Abnahme
description: Eine zurückgenommene Zusage entfernen, drei Präzisierungen umsetzen, einen Fehler der Vorgabe beheben.
open_conflicts: 0
realization: implemented
implemented_in: 370b6c4
sources:
  - id: plan-scheibe-2a-nacharbeit
    resource: brain://project/ultra-brain/docs/.superpowers/plans/2026-08-20-scheibe-2a-nacharbeit.md
    doc_id: 01M0QGS593MGR37J6BQ1W21ZWY
    content_hash: "sha256:ebcb1159c3da4e93e645839802bc486b279cd57749d0346394a1cc24137d0edf"
    revision: 1
---

Fünf kleine Änderungen, mit denen die Suchkette an den Vertrag angeglichen
wird, wie ihn die [Abnahme](abnahme-scheibe-2a.md) korrigiert hat.

## Die Falle dieses Plans

**Aufgabe 1 entfernt eine Zusage, nicht einen Fehler.** Der Code funktionierte;
er hielt bloß eine Zusage, die die Spec zurückgenommen hatte. Die Versuchung
ist, ihn „für später" stehenzulassen — und das wäre falsch: *eine Gruppierung,
die nie greift, sieht im Code aus wie eine, die greift, und der nächste Leser
baut darauf auf.* Was bleibt, ist eine Meldung, und die kostet drei Zeilen.

## Was offen blieb

- **Die Ablagefrage.** Specs und Pläne liegen in einem Punktverzeichnis, das die
  Suchmaschine nicht betritt. Die Manifest-Angabe **erklärt** den Zustand,
  statt ihn zu beheben — richtig, solange der Zustand gewollt ist; ob er das für
  alle Punktverzeichnisse ist, war nicht entschieden.
- **Die Rücknahme der Entdopplung ist keine Aussage über qmds Verhalten.** Die
  Suchmaschine führt dieselbe Datei weiterhin unter jedem ihrer Pfade; nur ist
  der Fall in den registrierten Bereichen nicht vorhanden, und das eigene
  Register kann ihn nicht ausdrücken.
- **Ein skalares `graph.json` bricht `status` mit einem Traceback** — eine Zeile
  plus ein Testfall würden es schließen. Bewusst geparkt, weil der Fall schwer
  zu erzeugen ist, und **eingecheckt festgehalten, damit er nicht mit dem
  Arbeitsverzeichnis stirbt.**
- **Die Wiederholung bei leerem Ergebnis ist falsch verortet.** Sie ist die
  gemessene Eigenart *einer* Suchmaschine, wohnt aber oberhalb der Naht, die
  zusagt, dass nichts über ihr die Suchmaschine kennt. Die saubere Form ist eine
  Vertragsänderung mit Testfolge, keine Reparatur.

Siehe [Suche, Profile und Messwerte](../topics/suche-und-profile.md) und
[Entscheidungen der Ausführung, Scheibe 2a](entscheidungen-scheibe-2a.md).
