---
type: Entity
title: qmd
description: Die lokale Hybrid-Suchmaschine hinter der austauschbaren Suchschnittstelle.
open_conflicts: 0
sources:
  - id: cli-referenz
    resource: brain://project/loomux/docs/de/cli-reference.md
    doc_id: 01M47W91R5FRSPN6BE643656ZE
    content_hash: "sha256:68797887e50379828f958338511f2c03b36e31a01d9f813304bab6489922995a"
    revision: 2
  - id: benchmarks
    resource: brain://project/loomux/docs/de/benchmarks.md
    doc_id: 01M47W91R5MPENKN7N34H6QTNT
    content_hash: "sha256:73cf99840e93d14ff9d4cc7a385fb33fe7f8733867d1ee8523a43faeb943c1d8"
    revision: 1
  - id: erste-schritte
    resource: brain://project/loomux/docs/de/getting-started.md
    doc_id: 01M47W91R51JAGYP0JXJFDFAR4
    content_hash: "sha256:eca8772d56e6ec73c51e161a693c590ff8b790d06a09cfd7e86ff6d3d6158d3c"
    revision: 1
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

Bis zum Umzug am 2026-09-16 wurden diese Befehle über die Kommandozeile
gerufen; loomux spricht stattdessen qmds MCP-Daemon an — `fast` als
`searches:[{type:"vec"}]` mit `rerank:false` und ohne Erweiterung, `keyword`
ebenso mit `{type:"lex"}`, `full` mit `rerank:true` —, und auf den beiden
rerankerfreien Wegen trägt die Reihenfolge der Score der Antwort (1/Rang).
Der gehaltene Unterprozess ist damit
dieser Daemon: loomux startet ihn, wenn auf dem Port keiner antwortet, und sagt
dann einmal je Port an, dass der erste Aufruf den Modellstart zahlt.

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
