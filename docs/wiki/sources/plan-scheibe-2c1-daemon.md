---
type: Source
title: Plan Scheibe 2c-1 — Daemon, IPC und gehaltener Unterprozess
description: Der langlebige Prozess auf demselben Kern — und die Spec-Naht, die vor dem Bauen richtiggestellt wurde.
open_conflicts: 0
realization: implemented
implemented_in: 59b9303
sources:
  - id: plan-scheibe-2c1
    resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-21-scheibe-2c1-daemon.md
    doc_id: 01M0QGS5933HZYYJT7JVDGMZ1Y
    content_hash: "sha256:a034db4372598772a762d81b2a12f0df815ce2fed764ae3b3f19c70b89976b10"
    revision: 1
---

Ein langlebiger Daemon hält Graph, Registrierung und einen warmen
Suchmaschinen-Unterprozess; Klienten sprechen ihn über eine Named Pipe
beziehungsweise einen Unix-Socket an. **Der Daemon ist eine Fläche auf dem
bestehenden Kern — dieselben fünf Funktionen, kein zweiter Kern.**

## Die Vorbemerkung, die den Ton setzt

Der Plan beginnt damit, die **Spec zu widerlegen**: Sie benannte als Ort für den
gehaltenen Unterprozess eine Naht, die für Prozessaufrufe mit Argumentliste
gebaut ist — über das gewählte Protokoll gibt es weder eine Argumentliste noch
ein Prozessergebnis. Die Naht, die tatsächlich trägt, ist die Suchschnittstelle
eine Ebene höher.

Die Folge ist eine eigene Aufgabe **vor** dem Bauen: *Ein Plan, der gegen eine
falsche Spec baut, vererbt den Fehler.* Oberhalb der Naht ändert sich nichts —
die Zusage der Spec hält, nur an einer Naht weiter oben
([Der brain-Daemon](../entities/brain-daemon.md)).

## Wie gemessen wird

Der Messtask legt vorab fest, wie mit unbequemen Zahlen umzugehen ist:

- **Verfehlungen werden als Verfehlung eingetragen.** Erwartet wurden rund 70 ms
  gegen ein Budget von 20 ms. *Kein Budget wird in diesem Schritt geweitet.*
- **Weicht das Ergebnis der Rückgratmessung von der Erwartung ab, ist das der
  Befund** — die Vorgabe wird an der neuen Zahl entschieden, nicht an der alten.
- **Der Vorbehalt gehört ins Protokoll:** Der Ausgangswert des Prüfkorpus wurde
  über einen anderen Pfad und ein anderes Rückgrat gemessen; weicht der Wert ab,
  wird erst geklärt, ob Rückgrat oder Pfad — sonst wird ein Rückgratwechsel als
  Kettenverschlechterung verbucht.

Siehe [Suche, Profile und Messwerte](../topics/suche-und-profile.md) für die
Zahlen und [Plan Prüfkorpus v1](plan-pruefkorpus-v1.md) für den Ausgangswert.
