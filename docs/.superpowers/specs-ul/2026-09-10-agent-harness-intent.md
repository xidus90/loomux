# Intent: Agent-Harness für den lokalen Entwicklungszyklus

| | |
|---|---|
| **Status** | Arbeitsnotiz. Geht in der Spec auf und wird danach gelöscht: auf dem Ideenweg gibt es kein eigenes `intent.md` |
| **Eingang** | Idee, kein Auftrag. Deshalb gab es kein Annahmetor |
| **Autor** | Christoph Wübbels |
| **Klärung** | 2026-09-10 bis 2026-09-11 |
| **Rohverlauf** | Claude-Code-Sitzung `80f5e1a7-f0a2-4d07-931c-c339c67d900d` unter `~/.claude/projects/C--Users-micro-Documents--GIT-ultraloom/`. Nur zur Nachvollziehbarkeit; kein Knoten liest ihn |
| **Zweig** | `feature/agent-harness`, Worktree `.worktrees/agent-harness` |
| **Stand des Entwurfs** | [Graph](2026-09-10-agent-harness-graph.md) · [Knotenkatalog](2026-09-10-agent-harness-knotenkatalog.md) |

## Problem

Die Arbeit läuft über mehrere Modelle: Claude ist das Hauptmodell, daneben
kommen Gemini und Qwen zum Einsatz. Einen gemeinsamen Harness für den ganzen
Entwicklungszyklus gibt es nicht. Die Verfahren, die heute tragen
(`superpowers`), hängen an Claude Code und laufen auf einem Gemini- oder
Qwen-Knoten nicht. Nach der Planung bleibt die Arbeit an einen Menschen
gebunden, der jeden Schritt anstößt.

## Gewünschtes Ergebnis

Ein Loop- und Graph-Harness für den lokalen Entwicklungszyklus, von der Idee
bis zum Commit:

- Nach der Freigabe des Plans arbeitet der Harness **autonom bis zum Ende**.
- Aufgaben sind **gekapselt**: ein Knoten hat Modell, Effort und eine
  vollständige Beschreibung dessen, was er tut.
- Rollen laufen **auf verschiedenen Anbietern** und bleiben untereinander
  verträglich, z. B. Recherche auf Gemini, Implementierung auf Qwen,
  Orchestrierung auf Claude.
- Die Aufgaben liegen als markierte Karten auf einem **Kanban-Board**; das Board
  erzwingt Pflichtübergänge wie „nach der Implementierung immer durch die
  Review".
- Ein Mensch antwortet an festen Stellen **immer** und an weiteren nur,
  **wenn das Modell nicht weiterkommt**.

## Betroffene Systeme

- Laufzeit in `ultraloom`: `graph.py`, `runner.py`, `state.py`, `journal.py`,
  `model/port.py`, `model/agent_sdk.py`, `tools.py`, `config.py`,
  `discovery.py`, `cli.py`
- Prüfkette: `checks.py`, deren Thread-Pool der Fächerknoten mitnutzen soll
- Bestehender Flow: `flows/verify_until_green.py`
- Dokumentationsprüfung: `tests/test_flow_docs.py`
- Außerhalb: Claude Agent SDK, `agy` (Gemini-CLI)

## Randbedingungen

- **Das Vorhaben lebt in `ultraloom`**, nicht in einem eigenen Projekt.
- **Die Prüfkette bleibt LLM-frei.** `dependencies = []`; der Agentpfad hängt am
  Extra `agent`.
- **Zuschnitt:** nur der lokale Entwicklungszyklus. Auslieferung und Betrieb sind
  optional.
- **Kein Lauf pusht.** Ein Lauf darf committen; über das Remote entscheidet ein
  Mensch.
- **Arbeitsweise:** TDD, 100 % Coverage, statische Typen ohne unbegründetes
  `Any` oder `type: ignore`; Commits nur unter Nutzeridentität.
- **Grenzen aus dem Code**, nur mit Kernänderung verschiebbar:
  - Das Antwortschema eines Agentenknotens erlaubt nur flache Skalare
    (`graph.py:38–44`). Kein Agent gibt eine Liste oder ein Board zurück.
  - Der Graphrunner läuft sequenziell.
  - `validate()` weist jeden Zyklus ab, dessen Knoten nur einen Besuch erlauben.
  - `FlowContext` trägt kein Modell; das entsteht in `cli.py:423` und geht an den
    `Runner`, nicht an `build()`.
  - `discovery.py` kennt genau zwei Flow-Quellen und keine Entry-Points.
- **Anbieterneutrale Anweisungen.** Ein Gemini- oder Qwen-Knoten liest
  `.claude/skills` nicht.

## Offene Fragen

1. **Einordnung einfalten?** Seit der Plan die Marken selbst vergibt, überschreibt
   er die Ausgabe der Einordnung.
2. **Picker:** Code- oder Agentenknoten.
3. **Zugangsbahn zu fremden Modellen:** CLI, direkte API oder gemischt. `agy` kennt
   kein `--allowed-tools`; die Werkzeugdecke aus `tools.py` wäre dort nicht
   erzwingbar.
4. **Wie Qwen erreicht wird.** `agy models` listete am 2026-09-10 kein Qwen.
5. **Artefaktnamen** des Playbooks übernehmen (`intent.md`, `spec.md`,
   `plan.md`)?
6. **Katalog als Daten oder Code:** TOML-Instanzen oder Python-Module.
7. **Abhängigkeiten auf Karten:** `blocked_by` als Zeichenkette oder ein Aufruf je
   Abhängigkeit, wegen der flachen Schemas.
8. **Persistenz des Boards** über mehrere Läufe: wer schreibt, wann, was gilt
   beim Abbruch.
9. **Metriken:** als Abfrage über das Journal oder als eigener Sammler.
10. **Verfahren für Recherche und Dokumentation:** dafür gibt es heute keins.
11. **Nebenläufigkeit** paralleler Karten, getrennt vom Fächer.

## Verworfen, weil

Damit ein späterer Knoten nicht wieder vorschlägt, was schon mit Grund
abgelehnt ist.

- **Eigenes Repo für den Harness:** die Laufzeit steht hier, und ein Fremdpaket
  wäre für `ultraloom run` unsichtbar.
- **Auslieferung und Betrieb im Kern:** zuerst nur das lokale Entwickeln; Detektor
  und Kontrollbänder bleiben als Anhang festgehalten.
- **Brainstorming als eigene Phase:** es ist der erste Teil der Planung; welches
  Verfahren läuft, ist eine Einstellung am Knoten.
- **Ein Prüfer je Artefakt:** allgemeine Fehler, Spec-Treue und Sicherheit sind
  Blicke, die sich nicht gegenseitig ersetzen.
- **`intent.md` und Annahme als Pflicht auf jedem Weg:** wer selbst eine Idee hat,
  hat schon entschieden.
- **Verlaufsprotokoll als `intent.md`:** es enthielte überholte Stände, die ein
  späterer Knoten für gültig halten könnte.
- **Entscheidungsprotokoll mit Quellenangabe:** es wiederholte Graph und Katalog
  und liefe mit ihnen auseinander.
- **Einordnung vor der Klärung:** vorher steht nicht fest, worum es geht.
- **Der Mensch als erster Prüfer:** er bekommt nur ein Artefakt vorgelegt, das die
  Validierung sauber gemeldet hat.
- **Tore mit nur einem Ausgang:** eine abgelehnte Spec geht zurück ins
  Brainstorming, ein abgelehnter Plan in eine optionale Klärung.
- **Eigene Entwurfsphase mit `design.md`:** Anforderungen und Entwurf entstehen im
  selben Gespräch; die Linsen beider Artefakte überschnitten sich.
- **Pflichtübergänge im Prompt:** das Board soll sie erzwingen, und Daten sind
  testbar.
- **Kartenmarken von einem Tagger nach dem Plan:** wer zerlegt, weiß, welcher Art
  die Teile sind.
- **Nur Pflichttore:** ein festgefahrener Lauf braucht Notausgänge statt endloser
  Reparatur.
- **HTML-Seite für den Graphen:** Markdown mit Mermaid lässt sich ohne
  eingebundene Bibliothek lesen.

## Zu bestätigen

Vorgeschlagen, aber nie ausdrücklich bestätigt. Nach der Durchsicht wandert jeder
Punkt entweder in „Verworfen, weil" oder in die Spec, und dieser Abschnitt
entfällt.

| # | Vorschlag | verworfen wäre |
|---|---|---|
| 1 | `feature/multi-provider-llm` ist keine Basis. Von 155 Commits berühren neun den Strang, der Diff in `model/` ist gegenüber `master` negativ | den Zweig rebasen und weiterbauen |
| 2 | Die Anweisung lebt im Flow-Modul und verweist auf ein Verfahrensdokument unter `docs/` | Rollen als `.claude/skills` oder `.claude/agents` |
| 3 | Die Linsen laufen in einem Fächerknoten, der intern parallel ruft; jede Linse mit eigenem Modell | ein Agent mit mehreren Durchgängen, oder ein nebenläufiger Runner |
| 4 | Wer den Code schreibt, darf ihn nicht abnehmen; über den Modellnamen im Journal geprüft | die Regel nur im Prompt |
| 5 | Nacharbeit hat 5 Runden, ab Runde 4 ein frischer Arbeiter auf einem stärkeren Modell | fester Wiederholungszähler ohne Eskalation |
| 6 | Die Kernlinsen `grounding`, `references`, `consistency` laufen parallel zu den übrigen, mit Vorrang beim Zusammenführen | `grounding` als eigener Knoten vor dem Fächer |
| 7 | Eine abgelehnte Annahme beendet den Lauf | zurück in die Aufnahme |
| 8 | Vom Plantor gibt es keinen gezeichneten Rückweg zur Spec; ein Verwurf ist ein neuer Lauf | Kante „Plan verworfen → Spec" |
| 9 | Die Marke `area` wählt den Prüfsatz der Karte | je Bereich eigene Kanten im Graphen |
| 10 | Die Linse `cards` prüft, ob das Board aus dem Plan Karten schneiden kann | keine eigene Prüfung des Schnitts |
