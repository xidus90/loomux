---
type: Synthesis
title: Warum `fast` die Vorgabe bleibt, obwohl `full` mehr findet
description: These, Belege und offene Fragen zur Profilvorgabe — eine Entscheidung über Latenz, nicht über Trefferqualität.
status: draft
open_conflicts: 0
sources:
  - id: benchmarks
    resource: brain://project/loomux/docs/de/benchmarks.md
    doc_id: 01M47W91R5MPENKN7N34H6QTNT
    content_hash: "sha256:73cf99840e93d14ff9d4cc7a385fb33fe7f8733867d1ee8523a43faeb943c1d8"
    revision: 1
  - id: suche-und-profile
    resource: brain://project/loomux/docs/wiki/topics/suche-und-profile.md
    doc_id: 01M3534TGBT5D7K9XM7A1TM09Z
    content_hash: "sha256:9f07e477ed3905c2cd8d4bce0c6f282e65de1187776189cb5b8a8f591232f6c0"
    revision: 8
---

**Entwurf.** Ergebnis einer Recherche vom 24. August 2026, noch nicht geprüft.

## These

`fast` ist die Vorgabe **nicht, weil es besser findet, sondern weil es benutzbar
ist.** Die frühere Begründung — `fast` finde besser als `full` — ist gefallen;
was die Vorgabe heute trägt, ist allein die Latenz.

## Belege

- **Der Trefferunterschied ist real und wiederholbar:** 26/50 gegen 41/50, je
  dreimal dieselbe Zahl. *„Fünfzehn Fragen Unterschied ist kein Rauschen."*
- **Der Preis ist das Sechzigfache:** 93–97 ms je Frage gegen 5351–5429 ms.
- **Die alte Begründung ist ausdrücklich zurückgenommen:** *„`fast` verliert
  seine Begründung als ‚findet besser'"* — und ebenso ausdrücklich ersetzt:
  *„`fast` behält seine Begründung als ‚ist benutzbar'."*
- **Wofür genau gezahlt wird:** *„Der Reranker holt die Treffer. Die
  Frageerweiterung kostet nur Zeit."*

Beleg ist das Messprotokoll zur Entscheidung 46
(`entscheidung-46.md` in den Arbeitspapieren des Archiv-Release
`archive/parity-recordings`); die spätere Messung über den Dienst steht in
`docs/de/benchmarks.md`, die Einordnung in
den Vertrag in
[Suche, Profile und Messwerte](../topics/suche-und-profile.md).

## Zwei Spannungen, zitiert statt aufgelöst

- **Die Rohmessung und die Kettenmessung widersprechen sich.** Roh sah
  Vektorsuche plus Reranking wie das Optimum aus („40 gegen 39 Treffer bei 1,4 s
  weniger"); durch die volle Kette gemessen war es *„fünf Fragen schlechter,
  nicht einer besser."* Der Unterschied liegt daran, dass die Rohmessung
  Dateinamen verglich und die Kette Pfade.
- **Die frühere Zahl aus Scheibe 2b steht gegen die heutige** (24/30 gegen 22/30
  für `fast`). Das Protokoll erklärt die Differenz mit Umfang und Bestand — ein
  Bereich und dreißig Fragen gegen vier Bereiche und fünfzig.

## Was diese Recherche nicht belegt

- ~~**Keine gelesene Stelle im Code oder in der Spec schreibt die Vorgabe fest.**~~
  Beantwortet: **Die Vorgabe steht im Code.** `internal/cli/brainargs.go` setzt
  `fast` als Rückfall für `--profile`, und das MCP-Werkzeug übernimmt ihn
  (`internal/serve/brain/tools.go`).
- **Ob ein Nutzer die Vorgabe dauerhaft auf `full` umstellen kann**, ist nicht
  belegt.
- **Ob die 24 von `fast` verfehlten Fragen im Alltag überhaupt gestellt werden**,
  ist ungemessen — das Protokoll benennt diesen Vorbehalt selbst.
- **Ob stärkere Hardware die Latenz unter die Nutzbarkeitsschwelle drückt**, ist
  offen; andere Rückgrate als die zwei gemessenen wurden nicht gesucht.
