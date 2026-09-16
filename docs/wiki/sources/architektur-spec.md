---
type: Source
title: Architektur-Design ultra-brain
description: Der Vertrag des Projekts — Zweck, Bausteine, Datenmodell, Scheiben und Entscheidungsprotokoll.
open_conflicts: 0
sources:
  - id: architektur-spec
    resource: brain://project/ultra-brain/docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md
    doc_id: 01M0QGS594F2KCWTWK9XV07M05
    content_hash: "sha256:c82f573ca64a6b8c53d0f158cb83847865cc8b32c99f710d37c6050fb11ac3ff"
    revision: 3
---

Die Architektur-Spec vom 18. August 2026 ist ausdrücklich **der Vertrag**: Taucht
beim Bauen eine Streitfrage auf, gilt, was dort steht — nicht die Erinnerung und
nicht die Auslegung durch ein Modell. Sie wird bei jeder Scheibe fortgeschrieben;
die Fassung, aus der hier verdichtet wurde, trägt Revision 3.

## Die Frage, aus der alles folgt

Wie wird aus gesammeltem Wissen ein System, dem man vertrauen kann, statt ein
wachsender Müllhaufen? Die Frage hat zwei Hälften — **Auffindbarkeit** (alles
wiederfinden, auch ohne das exakte Wort) und **Vertrauen** (Duplikate,
Widersprüche und veraltete Stände fallen auf, statt still herumzuliegen).

## Was das Dokument festlegt

- sieben Fähigkeiten, jede mit genau einem Bauteil, davon drei über die
  Referenzarchitektur hinaus: Aktuell bleiben, Abgrenzen, Wiederverwenden
  → [Grundsätze und Vertrauenskette](../topics/architektur-grundsaetze.md)
- drei Orte für Daten, Manifest, zentrale Registrierung, Identitätsregister und
  geteilte Bereiche → [Datenmodell und Bereiche](../topics/datenmodell-und-bereiche.md)
- Suchprofile, Suchleiter, Latenzbudgets und das Messprogramm
  → [Suche, Profile und Messwerte](../topics/suche-und-profile.md)
- Schema, Ingest, Konfliktform und Lint der Verdichtungsschicht
  → [Die Wiki-Schicht](../topics/wiki-schicht.md)
- zwei Aktualitätsbegriffe, Prüfzentrum, Evidenzbindung und der Merge-Auslöser
  → [Brain Maintenance](../topics/brain-maintenance.md)
- Datenschutzmodi, Kanäle und der Unterschied zwischen `never` und `local_only`
  → [Datenschutz und Kanäle](../topics/datenschutz-und-kanaele.md)
- die Zerlegung in acht Scheiben mit je einem Fertig-Kriterium
  → [Die Scheiben und ihre Abnahmen](../topics/scheiben-und-abnahme.md)

## Was das Dokument über sich selbst sagt

Zwei Eigenschaften sind ungewöhnlich und tragen den Rest:

- **Jede Behauptung hat einen Prüfpunkt; wo keiner formulierbar ist, wird die
  Behauptung gestrichen.** Die Messung läuft vor der Entscheidung.
- **Widerlegte Annahmen bleiben mit ihrer Widerlegung stehen**, statt still
  ersetzt zu werden. §16 führt fünfzehn offene Ränder, darunter mehrere,
  die eine frühere Fassung dieser Spec behauptet und eine Messung entkräftet
  hat — die Mehrsprachigkeit, die Entdopplung, die Profilvorgabe.

Das ist zugleich die Herkunft der Konfliktregel der Wiki-Schicht: Beide Stände
stehen bleiben zu lassen, ist hier bereits die Arbeitsweise des Vertrags selbst.

## Nicht in dieser Seite

`§19` nennt die neun Quellen der Architektur (qmd, Karpathys LLM-Wiki, OKF v0.2,
die Bauanleitung und drei Transkripte, Superpowers, spätere Mockups). Sie sind
Rohquellen und noch nicht verdichtet.
