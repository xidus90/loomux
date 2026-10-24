---
type: Synthesis
title: Modelladapter über die CLI, die API später
description: Entscheidung vom 24. Oktober 2026 — Agentenknoten erreichen Claude und Gemini zuerst über deren CLIs; API-Adapter folgen danach hinter derselben Schnittstelle.
status: accepted
open_conflicts: 0
sources:
  - id: flows
    resource: brain://project/loomux/docs/de/flows.md
    doc_id: 01M47W91R5WGVSD1M1VC73VBW5
    content_hash: "sha256:f2415ee6515f2a723b089b2a36b8508d93c4ad68b58aa8bf390b1f62740569d2"
    revision: 1
  - id: konfiguration
    resource: brain://project/loomux/docs/de/configuration.md
    doc_id: 01M47W91R50RT15H2ES8RV7TJT
    content_hash: "sha256:0b9bf911c08d8ebb31dda0b47da8e0a658f643039ab34ceeca9777d01f05a3eb"
    revision: 14
---

**Entschieden am 24. Oktober 2026:** Option A, nur CLI, zuerst. API-Adapter
(Option B) kommen später als eigene Arbeit.

## Frage

Agentenknoten eines [Flows](../../de/flows.md) laufen nicht, weil loomux keinen
Modelladapter hat. Offen war, ob die Adapter Claude und Gemini über deren CLIs,
deren APIs oder beides erreichen.

## Was jeder Adapter erfüllen muss

Fest durch den Code (`internal/flow/model`):

- Eingang `Request{Prompt, Tools, Effort, Provider, Model, Reply}`, Antwort
  flach: Felder `string`, `int` oder `bool`, dazu Tokens und das Modell, das
  tatsächlich antwortete.
- Die Werkzeugprofile `read_only`, `edit`, `shell` und `mcp` sind die einzige
  Grenze zwischen einem Knoten und Bash oder Write.
- Ein Lauf hält einen Anbieter; ein Flow, dessen Rollen auf zwei auflösen, wird
  abgelehnt.
- Ein leeres `model` heißt: die CLI wählt ihr Standardmodell — die
  [Rollenkette](../../de/configuration.md) setzt die CLI also schon voraus.

## Abwägung

| | A: nur CLI | B: nur API | C: beides |
|---|---|---|---|
| Anmeldung | Abo bzw. Login der CLI, loomux fasst keine Schlüssel an | loomux verwaltet API-Schlüssel | beides |
| Werkzeuge | führt die CLI aus; das Profil wird ihre Freigabeliste | loomux baut die Werkzeugschleife selbst | beides |
| Schreibschranke und Wächter | greifen, weil die CLI-Sitzung die loomux-Hooks auslöst | im eigenen Werkzeugloop nachzubauen | doppelt |
| Strukturierte Antwort, Tokens | hängen an der JSON-Ausgabe der CLI und ihrer Version | genau | — |
| Aufwand | ein Prozessstart und ein Parser je Anbieter | groß | Summe, doppelte Testmatrix |

## Begründung

1. Die [Schreibschranke](../topics/schreibschranke.md) wirkt in einer
   CLI-Sitzung ohnehin; B müsste diese Sicherheitsgrenze ein zweites Mal bauen.
2. loomux verwaltet keine Zugangsdaten.
3. `Model` ist ein Interface: ein API-Adapter kommt später dazu, ohne dass sich
   das Flow-Format ändert. C jetzt wäre Arbeit auf Vorrat.

## Später: API-Adapter

B bleibt gewollt, nach A. Anlässe: Läufe ohne installierte CLI (etwa in der
CI), genaue Kostenerfassung, strukturierte Antworten ohne Parser. Vorher zu
klären: wie die Werkzeugschleife die Schreibschranke und den Wächter
einhält und wo Schlüssel liegen.

## Offen für A

- Ruft der Anbieter `agy` die `agy`-CLI oder die `gemini`-CLI auf?
- Antwort auslesen: JSON-Schema-Flag der CLI, falls vorhanden, oder
  Prompt-Vorgabe mit Parser und genau einem neuen Versuch.
- Werkzeuge außerhalb des Profils ausdrücklich sperren, nicht nur das Profil
  freigeben; sonst hängt die Grenze an den Voreinstellungen der CLI.
- Zeitgrenze, Abbruch, und was ins Journal kommt, wenn die CLI keine
  Token-Zahl liefert.
