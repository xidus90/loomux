---
type: Synthesis
title: Wann ein Hinweis auf den Code-Graph lohnt
description: Agenten-Spikes vom 11. November 2026 — ein Hinweis Richtung Graph lohnt erst, wenn der Graph die Ketten kennt, die grep findet; von fünf Formen trägt nur die einmalige Ablehnung des ersten grep.
status: draft
open_conflicts: 0
realization: implemented
sources:
  - id: benchmarks
    resource: brain://project/loomux/docs/de/benchmarks.md
    doc_id: 01M47W91R5MPENKN7N34H6QTNT
    content_hash: "sha256:d10df343fb61fb76ea53dfdf88a3f2511e6c17849aa11f01be74cdf87ac0d952"
    revision: 22
  - id: code-graph
    resource: brain://project/loomux/docs/wiki/topics/code-graph.md
    doc_id: 01M39NDHWJPTMR7NKS5RZ78AWK
    content_hash: "sha256:d3a36f526faa25de7c50a62f218568f30d1e5e57e777ab9f3b94403b4f5fbae2"
    revision: 17
---

## Frage

Soll der Wächter einen Agenten, der Code mit grep durchsucht, auf den
[Code-Graph](../topics/code-graph.md) lenken, und wenn ja, in welcher Form?
Angeregt hat das graphify, dessen PreToolUse-Hook Read und Grep Richtung Graph
schiebt.

## Was die Transkripte zeigten

34 Transkripte von Haupt- und Subagenten in diesem Repo, Stand 11. November
2026: kein einziger Aufruf eines `graph_*`-Werkzeugs. 1202 Suchen liefen über
grep, rg oder Select-String in Bash oder PowerShell, nur 109 über das
Grep-Werkzeug. Etwa 155 suchten Definitionen, etwa 410 suchten sonst im Code,
der Rest in Doku und Logs. Daraus folgt:

- Ein Hinweis nur auf `Read|Grep|Glob` verfehlt rund 90 % der Suchen; er
  muss im Shell-Befehl greifen.
- Read ist kein Ziel: Claude Code verlangt vor jedem Edit ein Read, und die
  Reads sind meist gezielt.
- Die MCP-Werkzeuge von loomux sind in Claude Code zurückgestellt; der Agent
  sieht ihr Schema erst nach einem ToolSearch.

## Die fünf Formen (Runde 1, alter Graph)

Headless-Läufe mit Opus, sechs Aufgaben (vier Codefragen, zwei zu Doku und
Changelog), je eine Wiederholung. Alle 30 Antworten richtig.

| Form | Ergebnis |
|---|---|
| v0 kein Hinweis | Grep in ein bis zwei Aufrufen, richtig |
| v1 Hinweis auf `loomux graph …` nach dem grep | zu spät: der Agent hat die Antwort schon und ignoriert ihn |
| v2 Hinweis auf das MCP-Werkzeug nach dem grep | +53 % Tokens: ToolSearch-Umweg, dann doch grep |
| v3 erstes grep über Code einmal ablehnen | +37 % Tokens: der Graph lieferte Lücken, grep musste nachbessern |
| v4 Graph-Antwort im Hook anhängen | 270–505 ms im Hook, das Budget ist 35 ms; sonst wie v1 |

Bei Doku-Aufgaben schlug keine Form an.

## Der eigentliche Engpass

Runde 2 mit Fragen über mehrere Aufrufstufen zeigte, warum: Der Graph verlor
Ketten an Funktionswerten (`Deps{Blast: query.Blast}`,
`postToolUse = hooks.PostToolUse`) und an verketteten Aufrufen
(`blast.New(g).Reach()`). grep fand dort mehr. Ein Hinweis Richtung Graph
lenkte also auf das schwächere Werkzeug. Der Go-Extraktor `go/3` schließt
diese Lücken (`references`-Kanten, Auflösung über den Ergebnistyp); die
Zahlen stehen in [`docs/de/benchmarks.md`](../../de/benchmarks.md), Eintrag
„Funktionswerte und Ketten im Go-Graph“.

## Nach dem Fix

- Runde 3 (Aufgaben aus den geschlossenen Lücken, n = 6 je Form): v3 braucht
  15 % weniger Tool-Aufrufe und 9 % weniger Tokens als v0, beide immer
  richtig.
- Runde 4 (acht neue Aufgaben aus anderen Teilen des Repos, n = 16 je Form):
  gleiche Kosten (5,2 gegen 5,3 Aufrufe, 238 k gegen 236 k Tokens), v3
  16 von 16 vollständig, v0 15 von 16. Der eine Fehlgriff übersah Aufrufer,
  die über `readGraph = store.Read` gehen.

## Schluss

- **Erst der Graph, dann der Hinweis.** Solange grep Ketten findet, die der
  Graph nicht kennt, kostet jeder Hinweis Richtung Graph.
- **Die Form ist v3:** das erste grep über Code einmal ablehnen und den
  Shell-Befehl `loomux graph callers <Name>` nennen. Ein Hinweis nach dem
  grep kommt zu spät, ein Verweis auf das MCP-Werkzeug kostet den
  ToolSearch-Umweg, und eine Graph-Abfrage im Hook sprengt das Zeitbudget.
- **Belegstärke:** 16 Läufe je Form, eine Maschine, ein Modell. Das ist eine
  Richtung, kein Beweis.

## Mit dem echten Binary (12. November 2026)

Gebaut nach `docs/.superpowers/specs/2026-11-12-graph-hinweis-design.md`,
standardmäßig an, abschaltbar mit `[guard] graph_hint = false`. Die erste
Fassung kostete in der Abnahme (32 Läufe je Variante) +14 % Tokens, machte
aber `store.Read` vollständig (4/4 gegen 0/4). Drei Verluste fielen auf und
sind behoben: ein Hinweis nach einer schon gestellten Graph-Abfrage, ein
nackter Name, der ein gleichnamiges falsches Symbol traf, und MCP-Aufrufe mit
`scope: "loomux"`. Danach kostet der Hinweis +5 % im Mittel bei gleichem
Median. Die Hook-Zeit bleibt mit 14–17 ms warm im Budget. Zahlen:
[`docs/de/benchmarks.md`](../../de/benchmarks.md), Eintrag „Der
Graph-Hinweis im Wächter“.
