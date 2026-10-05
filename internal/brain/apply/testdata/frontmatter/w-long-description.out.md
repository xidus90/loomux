---
type: Synthesis
title: Gegenprüfung vor jeder Designempfehlung
description: Jede Brainstorming-Frage samt eigener Empfehlung erst an einen Fable-Subagenten
  und den Advisor, dann an den Menschen — und die tragenden Zitate danach selbst nachrechnen.
status: draft
open_conflicts: 0
sources:
- id: architektur-spec
  resource: brain://project/loomux/docs/de/architecture.md
  doc_id: 01M0QGS594F2KCWTWK9XV07M05
  content_hash: sha256:0000000000000000000000000000000000000000000000000000000000000000
  revision: 8
generated:
  at: '2026-09-22T08:16:27.936837+00:00'
verified:
- by: human:tester
  at: '2026-09-22T08:16:27.936837+00:00'
---

# Gegenprüfung vor jeder Designempfehlung

**Die Regel.** Im Brainstorming geht jede Frage zusammen mit der eigenen
Empfehlung erst an einen Subagenten mit `model: "fable"` und an den Advisor —
und erst danach an den Menschen. Festgelegt am 2026-08-27 während des
Brainstormings zu [Scheiben und Abnahme](../topics/scheiben-und-abnahme.md).

**Warum.** In derselben Sitzung lagen drei Empfehlungen daneben, alle mit
demselben Muster: Die Architektur-Spec hatte die Frage längst entschieden, und
gelesen worden war nur die Prosa ringsherum statt der Vertragstext selbst.

- Der Ort der Prüffälle stand in §5.2 samt Manifestfeld `layout.review`; der
  Vorschlag hätte sie stattdessen in die Wiki-Bündel gelegt, wo der Lint jeden
  offenen Fall als typlose Seite gemeldet hätte.
- Die Umbenennungserkennung über den Inhaltshash steht wörtlich in §5.6, samt
  Behandlung des harten Falls — es war nie eine Entwurfsfrage, sondern eine
  Lücke zwischen Spec und Code.
- Der Befehlsname `brain maintenance check` war erfunden; der CLI-Vertrag nennt
  `brain reconcile`, `brain cases`, `brain case` und `brain approve` seit §9.1.

**Wie anwenden.** Der Auftrag an den Subagenten enthält die Frage, alle
Optionen, die eigene Empfehlung und die Dateien, gegen die zu prüfen ist — mit
der Auflage, Belege als `Datei:Zeile` zu liefern statt Prosa nachzuerzählen.

**Und danach die tragenden Zitate selbst nachrechnen.** Der Prüfer irrt anders,
nicht seltener: Einmal lag er in der Sache richtig, benannte aber einen falschen
Mechanismus (MCP-Prompt statt Skill), und einmal empfahl er, `brain:wiki-plan`
aus der Scheibe zu nehmen — gegen das Fertig-Kriterium und gegen einen bereits
getroffenen Zerlegungsbeschluss. Ein Zitat, das eine ganze Antwort trägt, wird
gelesen, bevor es festgeschrieben wird.

Verwandt: [Architektur-Grundsätze](../topics/architektur-grundsaetze.md),
[Abnahmen und echte Umgebung](../topics/abnahmen-und-echte-umgebung.md),
[Brain Maintenance](../topics/brain-maintenance.md).
