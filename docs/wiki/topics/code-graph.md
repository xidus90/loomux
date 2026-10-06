---
type: Topic
title: Der Code-Graph
description: Wie loomux Symbole und Kanten ohne Modell aus dem Quelltext zieht, Treffer mit Personalized PageRank ordnet und den Blast-Radius einer Änderung berechnet.
open_conflicts: 0
realization: in_progress
sources:
  - id: architektur
    resource: brain://project/loomux/docs/de/architecture.md
    doc_id: 01M47W91R5WKHDNYTXW8ZXTCSW
    content_hash: "sha256:7e34bcbe26fc6c0977631769395ca6f97d8bea9be643fbe4641809bb35266fbf"
    revision: 2
  - id: cli-referenz
    resource: brain://project/loomux/docs/de/cli-reference.md
    doc_id: 01M47W91R5FRSPN6BE643656ZE
    content_hash: "sha256:1b731affd83910cd4e8dd6ca0b05a7b375ecba6d7a4790c6106c4e85f6b158f3"
    revision: 3
---

## Wofür

Ein Coding-Agent erkundet ein Repository sonst in jeder Sitzung blind von
vorn. Der Code-Graph gibt ihm stattdessen einen **deterministischen Symbol-
und Aufrufgraphen, ohne Modell und ohne Kosten**: Symbole, Aufrufe,
Typ-Hierarchien und Importe. Leitsatz: **„Lexik schlägt vor, der Graph
entscheidet."** Eine Vektordatenbank braucht es dafür nicht.

Das Vorbild steht in `NOTICE.md` unter „Ported sources“. Übernommen sind Extraktion, Ranking und die
Frischeprüfung; statt des Kernlogik-Exzerpts (Crux) des Vorbilds hängt `graph ask
--source` den eigenen Span des Symbols an, höchstens 80 Zeilen. Verworfen sind Laufzeit, Cloud und
Telemetrie:

| Vorbild | loomux |
|---|---|
| Node.js mit nativen C++-Bindings | ein Go-Binary, CGo-frei |
| Symbol-Hashes an eine Cloud-API | Verknüpfung nur lokal, mit dem eigenen Wiki |
| Nutzungsstatistik an fremde Server | keine Telemetrie |

## Wie der Graph entsteht

`loomux graph build` zieht Symbole und Kanten aus dem Quelltext und schreibt
sie nach `.loomux/state/graph/wiring.json` — Maschinenzustand, git-ignoriert.

- **Go** liest er allein mit `go/parser`. Die Spec sah für den Offline-Bau
  noch `go/types` vor; gebaut ist der Extraktor ohne `go/types` und ohne
  Build (`internal/code/extract/golang`).
- **Python** liest er seit G5a auf `gotreesitter`, einer Tree-sitter-Laufzeit
  in reinem Go (`internal/code/extract/python` auf dem gemeinsamen Kern
  `extract/treesitter`). Das Binary bleibt CGo-frei; ein Build mit
  `CGO_ENABLED=0` im Tor hält das fest. Eine Datei mit Syntaxfehlern bricht
  den Build nicht ab: sie behält ihren Dateiknoten, und der Bericht zählt die
  Fehler.

Jede Sprache löst in einem eigenen Namensindex auf; Kanten über
Sprachgrenzen gibt es nicht. Unveränderte Dateien nimmt der Build aus dem
Extraktions-Cache (`.loomux/state/graph/cache/extract.json`), statt sie neu zu
parsen; gelesen und gehasht wird trotzdem jede, und `--no-reuse` parst alle.
`loomux graph check` extrahiert erneut, vergleicht mit dem Graphen auf der
Platte und endet mit 0, wenn er frisch ist, mit 1 bei Drift.

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
  schweigt er; **einen Edit blockiert er nie.** Eine `.py`-Datei bekommt
  keinen: jeder solche Edit zahlte das Laden einer Grammatik und das Parsen
  der Datei, gegen ein Ziel von unter 100 ms Eigenzeit des Monitors.
  `internal/hooks` importiert die Tree-sitter-Laufzeit darum nicht.
- **Prüfart `graph`:** `check graph-fresh` baut bei Drift neu und ist nur rot,
  wenn der Neubau scheitert; `check blast-audit` ist rot, wenn ein geänderter
  Bereich mit genug Aufrufern keinen geänderten Test hat. Die Lane-Namen der
  Spec (`graph-freshness`) wurden dabei ersetzt. Presets haben Go und
  Python; die Lane läuft je Lauf einmal, in der Wurzel, getragen vom ersten
  Stack mit Graph-Befehl, und `graph = false` unter einem Stack schaltet sie
  für das ganze Projekt ab. Testdateien sind `_test.go` und für Python
  `test_*.py`, `*_test.py`, `tests.py`, `conftest.py` und alles unter
  `tests/` oder `test/`.
- **Stop-Hook mit Blast-Logik:** `graph` steht auch im Profil `stop`. Am
  Rundenende prüft `blast-audit --cached` über eine Kopie des Index, die den
  ganzen Arbeitsbaum samt unversionierter Dateien trägt, alles gegen `HEAD`;
  ein Befund hält die Runde mit Exit 2 an wie ein roter Test.

## Was bewusst fehlt

Kein Gegenstück zum Cloud-Brain des Vorbilds, keine Telemetrie, kein `upgrade`-Befehl: das
Wissen ist lokal ([Architektur-Grundsätze](architektur-grundsaetze.md)), und
das Binary kommt über die üblichen Wege.

## Was offen ist

- **Mehrsprachige Extraktion** (G5): G5a — die Schnittstelle, der Kern auf
  `gotreesitter`, Python und der Cache — ist fertig und an zwei Python-Repos
  abgenommen (`docs/.superpowers/parity/code-g5.md` in den Arbeitspapieren des
  Archiv-Release `archive/parity-recordings`). Offen sind G5b (TypeScript/TSX), G5c (GDScript) und
  G5d (C++, erst nach einer Recall-Prüfung gegen die C-Laufzeit). Der erste
  Plan, Tree-sitter als WebAssembly über `wazero`, ist verworfen.
- **Eine Python-Klasse als Saat** gibt im Blast das Testsignal `na`: als
  Verhalten zählen nur Funktionen und Methoden, eine Änderung an einer Klasse
  ohne `__init__` läuft darum still durch `blast-audit`.
- **Die Konfiguration `[graph]`** mit `languages`, `exclude` und `max_depth`
  aus der Spec: das Schema kennt heute nur den Schalter `graph` unter
  `[modules]`.
- Die Brücke von Code-Symbolen zu Wiki-Seiten und ADRs und der Viewer
  `graph viz` ordnete die Spec G4 und G5 zu; die Roadmap im README führt
  beide heute unter dem Web-OS (W3).

Quelle: die Architektur und die CLI-Referenz der Nutzerdoku; offene Stufen
führt die Roadmap im README.
