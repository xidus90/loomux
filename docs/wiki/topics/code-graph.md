---
type: Topic
title: Der Code-Graph
description: Wie loomux Symbole und Kanten ohne Modell aus dem Quelltext zieht, Treffer mit Personalized PageRank ordnet und den Blast-Radius einer Änderung berechnet.
open_conflicts: 0
realization: in_progress
sources:
  - id: code-graph-spec
    resource: brain://project/loomux/docs/.superpowers/specs/2026-09-14-loomux-code-graph-design.md
    doc_id: 01M39G4J1415Z8ESCSS7FY00SE
    content_hash: "sha256:83365e0af4c04080d587fe9381f7eb6d6f046a7ad223b7a6c719f43533d3afda"
    revision: 1
---

## Wofür

Ein Coding-Agent erkundet ein Repository sonst in jeder Sitzung blind von
vorn. Der Code-Graph gibt ihm stattdessen einen **deterministischen Symbol-
und Aufrufgraphen, ohne Modell und ohne Kosten**: Symbole, Aufrufe,
Typ-Hierarchien und Importe. Leitsatz: **„Lexik schlägt vor, der Graph
entscheidet."** Eine Vektordatenbank braucht es dafür nicht.

Vorbild ist `trailhq/Graft`. Übernommen sind Extraktion, Ranking, die
Frischeprüfung und das Kernlogik-Exzerpt; verworfen sind Laufzeit, Cloud und
Telemetrie:

| Graft | loomux |
|---|---|
| Node.js mit nativen C++-Bindings | ein Go-Binary, CGo-frei |
| Symbol-Hashes an eine Cloud-API | Verknüpfung nur lokal, mit dem eigenen Wiki |
| Nutzungsstatistik an fremde Server | keine Telemetrie |

## Wie der Graph entsteht

`loomux graph build` zieht Symbole und Kanten aus dem Go-Quelltext, allein mit
`go/parser`, und schreibt sie nach `.loomux/state/graph/wiring.json` —
Maschinenzustand, git-ignoriert. Die Spec sah für den Offline-Bau noch
`go/types` vor; gebaut ist der Extraktor ohne `go/types` und ohne Build
(`internal/code/extract/golang`). `loomux graph check` extrahiert erneut,
vergleicht mit dem Graphen auf der Platte und endet mit 0, wenn er frisch
ist, mit 1 bei Drift.

**Die Pakete liegen unter `internal/code/`, nicht unter dem `internal/graph/`
der Spec.** Die Trennung, die die Spec verlangt, gilt trotzdem: der
Hook-Pfad bindet kein MCP-SDK, und ein Tor-Test liest dafür den Importgraphen.

## Wie er ordnet

**Personalized PageRank („GraphRank"):** Lexikalische Treffer (BM25 auf
Symbolnamen und Doku) sind die Saat, dann eine Power-Iteration über den
ungerichteten Graphen mit Neustart-Wahrscheinlichkeit 0,25 und 25 Schritten.
Die Masse isolierter Knoten fließt an die Saat zurück, die Summe bleibt 1; bei
Gleichstand entscheidet die Symbol-ID alphabetisch. `loomux graph ask` fährt
diese Kette.

**Blast-Radius:** Gerichtete Kanten (`calls`, `imports`, `extends`,
`implements`, `references`), in Richtung `in` (wer hängt von mir ab?) oder
`out` (wovon hänge ich ab?), transitiv per Breitensuche mit Tiefengrenze.
`loomux graph blast` führt einen git-Diff auf die Symbole, die seine Hunks
berühren, und auf das, was sie erreicht.

## Die Oberflächen

| Befehl | MCP-Werkzeug | Zweck |
|---|---|---|
| `graph ask` | `graph_find_code` | Suche über Symbole, nach GraphRank geordnet |
| `graph check` | `graph_check_freshness` | Drift zwischen Arbeitsbaum und Graph |
| `graph skeleton` | `graph_file_api` | Signaturen einer Datei ohne Rümpfe |
| `graph callers` | `graph_trace_calls` | Aufrufer oder Aufgerufene, bis zur Hülle |
| `graph grep` | `graph_find_all` | Regex, gruppiert nach Symbol, nach Kopplung geordnet |
| `graph map` | `graph_repo_map` | Repo-Karte mit Hubs und Hotspots |
| `graph blast` | `graph_blast` | Blast-Radius eines git-Diffs |

Dazu `graph build` und `graph stats`. Die Spec nannte sechs MCP-Werkzeuge;
`graph_blast` kam als siebtes hinzu. `loomux serve` beantwortet sie neben den
fünf `brain_*`-Werkzeugen. **Auf dem Cloud-Kanal schließen sie im Zweifel:**
was ein `local_only`-Bereich verbirgt, verraten auch die Graph-Werkzeuge
nicht ([Datenschutz und Kanäle](datenschutz-und-kanaele.md)).

## In Hook und Prüfkette

- **Post-Edit-Hook, rein informativ:** Nach einem Edit an einer `.go`-Datei
  ohne rote Lane nennt er die direkten Aufrufer in anderen Dateien dessen, was
  sich geändert hat (höchstens zehn). Ohne Graph oder bei einem Parsefehler
  schweigt er; **einen Edit blockiert er nie.**
- **Prüfart `graph`:** `check graph-fresh` baut bei Drift neu und ist nur rot,
  wenn der Neubau scheitert; `check blast-audit` ist rot, wenn ein geänderter
  Bereich mit genug Aufrufern keinen geänderten Test hat. Die Lane-Namen der
  Spec (`graph-freshness`) wurden dabei ersetzt.

## Was bewusst fehlt

Kein `graft brain`-Pendant, keine Telemetrie, kein `upgrade`-Befehl: das
Wissen ist lokal ([Architektur-Grundsätze](architektur-grundsaetze.md)), und
das Binary kommt über die üblichen Wege.

## Was offen ist

- **Stop-Hook mit Blast-Logik** (G4c): der Audit am Rundenende, Arbeitsbaum
  gegen `HEAD`; bis dahin bleibt `graph` im Profil `stop` außen vor.
- **Mehrsprachige Extraktion** (G5): Tree-sitter als WebAssembly über
  `wazero`, zuerst TypeScript und Python.
- **Die Konfiguration `[graph]`** mit `languages`, `exclude` und `max_depth`
  aus der Spec: das Schema kennt heute nur den Schalter `graph` unter
  `[modules]`.
- Die Brücke von Code-Symbolen zu Wiki-Seiten und ADRs und der Viewer
  `graph viz` ordnete die Spec G4 und G5 zu; die Migrationstabelle führt
  beide heute unter dem Web-OS (W3).

Quelle: Code-Graph-Design; den Stand führt `docs/de/migration.md`
(G1 bis G4b fertig).
