---
type: Entity
title: qmd
description: Die lokale Hybrid-Suchmaschine hinter der austauschbaren Suchschnittstelle.
open_conflicts: 0
sources:
  - id: architektur-spec
    resource: brain://project/ultra-brain/docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md
    doc_id: 01M0QGS594F2KCWTWK9XV07M05
    content_hash: "sha256:c82f573ca64a6b8c53d0f158cb83847865cc8b32c99f710d37c6050fb11ac3ff"
    revision: 3
---

Fremdes Werkzeug (github.com/tobi/qmd), vollständig lokal, hier in Version
2.8.3 im Einsatz. Es liefert BM25, Vektorsuche und Reranking und steht
ausdrücklich **hinter einer eigenen Schnittstelle**, damit es austauschbar
bleibt — einen eigenen Suchkern zu bauen ist ausdrücklich nicht vorgesehen,
bis eine Messung dagegen spricht.

## Was es kennt

Drei Befehle, und **nichts dazwischen**: Erweiterung samt Reranker, rein
vektoriell, BM25. Die Annahme, es gebe eine Kette „Erweiterung ohne Reranker",
war eine ungeprüfte Vermutung über ein fremdes Werkzeug und hat ein ganzes
Suchprofil erfunden, das es nicht gibt
([Suche, Profile und Messwerte](../topics/suche-und-profile.md)).

## Gemessene Eigenheiten

- **Der Prozessstart kostet 278–310 ms** — gemessen mit einem Aufruf, der
  nichts tut außer zu starten. Bei einem Budget von 30 ms für die
  Stichwortsuche ist ein Prozessstart je Anfrage damit ausgeschlossen; qmd wird
  als gehaltener Unterprozess gefahren.
- **Es bricht gelegentlich still ab** und liefert einen leeren Treffersatz:
  Speicherzugriffsfehler ohne Fehlertext und ohne verratenden Rückgabecode, drei
  von sechzig Abfragen. Deshalb muss die Suchschnittstelle leeres Ergebnis von
  fehlgeschlagener Suche unterscheiden.
- **Es betritt Punktverzeichnisse grundsätzlich nicht**, unabhängig von jeder
  Ausschlussliste. Was dort herausfällt, ist keine Divergenz der Indizes,
  sondern eine Eigenschaft — erklärt über `unsearched` statt gemeldet.
- **Der dokumentierte Weg, das Einbettungsmodell über eine Umgebungsvariable zu
  wählen, ist wirkungslos**: Die Konfigurationsdatei hat Vorrang, und das
  Anlegen einer Sammlung schreibt dort die Standardwerte hinein. Wäre der
  Modellvergleich dem README gefolgt, hätte er zweimal dasselbe Modell gemessen,
  „keinen Unterschied" ergeben und zur Ablösung des Suchkerns geführt — wo in
  Wahrheit ein Konfigurationseintrag genügt.
- **Es bettet je eindeutigem Inhaltshash ein, führt dieselbe Datei aber unter
  jedem ihrer Pfade** in der Trefferliste. Bei einer Frage standen Rang 1 und
  Rang 3 auf demselben Dokument.
- **`qmd update` schreibt nur den Volltextindex fort, nicht die Vektoren.**
  Deshalb ist das Einbetten ein eigener Befehl; ein Dokument ohne Vektoren ist
  sprachübergreifend unauffindbar.

Siehe [Datenmodell und Bereiche](../topics/datenmodell-und-bereiche.md) für die
Ausschlusslisten und [Der brain-Daemon](brain-daemon.md) für die Haltung des
Prozesses.
