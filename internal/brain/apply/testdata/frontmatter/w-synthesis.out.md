---
type: Synthesis
title: Warum `fast` die Vorgabe bleibt, obwohl `full` mehr findet
description: These, Belege und offene Fragen zur Profilvorgabe — eine Entscheidung
  über Latenz, nicht über Trefferqualität.
status: draft
open_conflicts: 0
sources:
- id: entscheidung-46
  resource: brain://project/loomux/docs/.superpowers/bench-ub/entscheidung-46.md
  doc_id: 01M0QGS58WEG5TSWKWEJM89E5T
  content_hash: sha256:0000000000000000000000000000000000000000000000000000000000000000
  revision: 8
- id: architektur-spec
  resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
  doc_id: 01M0QGS594F2KCWTWK9XV07M05
  content_hash: sha256:0000000000000000000000000000000000000000000000000000000000000000
  revision: 8
generated:
  at: '2026-09-22T08:16:27.936837+00:00'
verified:
- by: human:tester
  at: '2026-09-22T08:16:27.936837+00:00'
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

Beleg ist das Messprotokoll zur Entscheidung 46, seit dem Umzug am 2026-09-16
im Bereich `project/loomux` unter
`docs/.superpowers/bench-ub/entscheidung-46.md`; die Einordnung in
den Vertrag steht in
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

- **Keine gelesene Stelle im Code oder in der Spec schreibt die Vorgabe fest.**
  Der Beleg ist ein Messprotokoll, kein Regelwerk; dass `fast` der eingebaute
  Standard ist, wurde nicht an einer Codestelle nachgewiesen.
- **Ob ein Nutzer die Vorgabe dauerhaft auf `full` umstellen kann**, ist nicht
  belegt.
- **Ob die 24 von `fast` verfehlten Fragen im Alltag überhaupt gestellt werden**,
  ist ungemessen — das Protokoll benennt diesen Vorbehalt selbst.
- **Ob stärkere Hardware die Latenz unter die Nutzbarkeitsschwelle drückt**, ist
  offen; andere Rückgrate als die zwei gemessenen wurden nicht gesucht.
