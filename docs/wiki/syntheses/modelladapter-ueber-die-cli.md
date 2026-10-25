---
type: Synthesis
title: Modelladapter über die CLI, die API später
description: Entscheidung vom 24. Oktober 2026 — Agentenknoten erreichen Claude und Gemini zuerst über deren CLIs; API-Adapter folgen danach hinter derselben Schnittstelle.
status: accepted
open_conflicts: 0
realization: implemented
implemented_in: 3e680c05
sources:
  - id: flows
    resource: brain://project/loomux/docs/de/flows.md
    doc_id: 01M47W91R5WGVSD1M1VC73VBW5
    content_hash: "sha256:491752cf01d4a39afa98d1f9883a7f073b8d6d6537ed06363c381e6fcabf3801"
    revision: 3
  - id: konfiguration
    resource: brain://project/loomux/docs/de/configuration.md
    doc_id: 01M47W91R50RT15H2ES8RV7TJT
    content_hash: "sha256:6eb1b4a440fda08eb8b8700da49ed2127019e0b32f0abdc207e427c3c5bf8c43"
    revision: 23
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

## Umgesetzt

Seit dem 25. Oktober 2026 laufen Agentenknoten über drei CLIs: `claude`
(Claude Code), `agy` (Antigravity) und `qoder` (`qodercli`). Strata, ein
lokales Qwen mit Anthropic-kompatibler API, läuft über `claude`; seine
`base_url` kommt als `ANTHROPIC_BASE_URL` in die Umgebung des Kindprozesses.
Kein Adapter umgeht je eine Berechtigung. Nutzerseitig steht alles in
[Flows](../../de/flows.md) und in der
[Konfigurationsreferenz](../../de/configuration.md).

Gemessen, nicht angenommen:

- **agy** liefert die Antwort über `--json-schema` als `structured_output`.
  Headless verweigert es ein Werkzeug automatisch, das eine Berechtigung
  braucht; der Knoten scheitert dann mit einem Fehler, der die Werkzeuge nennt,
  und der Nutzer erlaubt sie unter `permissions.allow` in der `settings.json`
  von agy. agy kann Werkzeuge nicht beschränken: `read_only` läuft in
  `--mode plan`, jedes andere Profil in `--mode accept-edits`, und loomux
  warnt vor dem Lauf. Die Schreibschranke beurteilt nur Werkzeuge, die
  Dateien schreiben, keine Shell-Befehle.
- **qoder** beschränkt mit `--tools` hart; verifiziert, die Namen sind `Bash`,
  `Edit`, `Glob`, `Grep`, `Read` und `Write`. Die Antwort kommt über eine
  Schema-Anweisung im Prompt mit genau einem neuen Versuch; Tokens kann es
  als 0 melden.
- **claude** beschränkt mit `--tools` (MCP-Werkzeuge über `--allowedTools`) und
  antwortet über `--json-schema`.

Damit sind die ersten drei Punkte unter „Offen für A“ beantwortet; die
Zeitgrenze ist `[agent] timeout` (Vorgabe `30m`, `"0"` schaltet sie ab).

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
