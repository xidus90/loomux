# loomux G1 — Delta zur Säule-3-Spec

**Datum:** 2026-09-16
**Stand:** entworfen, freigegeben zur Planung
**Ergänzt:** [`2026-09-14-loomux-code-graph-design.md`](2026-09-14-loomux-code-graph-design.md) — diese Datei
ersetzt die Säule-3-Spec nicht, sie berichtigt und verengt sie für die Stufe G1.
**Referenz:** `trailhq/Graft`, Commit `1e352a3` vom 2026-09-16, MIT.
**Arbeitsort:** Worktree `C:\Users\micro\Documents\#GIT\loomux-code-g1`, Branch `code-g1`,
abgezweigt von `sdd-1b-1` `636138b`.

## 1. Warum es dieses Delta gibt

Die Säule-3-Spec sagt in ihrem Kopf „zur Umsetzung nach Abschluss der
Fusions-Stufen 1a–4". G1 — die drei reinen Rechenpakete — wird vorgezogen und
parallel zur laufenden Stufe 1b-1 gebaut. Der Grund, den die Spec für die
Reihenfolge nennt (§11: Inittrace mit `go-sdk`), bindet G1 nicht: G1 zieht keine
Abhängigkeit ein. Die Messung gehört zu G3, wo `internal/serve` das MCP-SDK
tatsächlich einzieht.

Beim Abgleich gegen Graft hat sich außerdem gezeigt, dass §5.1 und §5.2 der
Säule-3-Spec an sechs Stellen etwas anderes beschreiben als die Referenz tut.
Abschnitt 3 hält jede Abweichung fest; gilt jeweils dieses Delta.

## 2. Zuschnitt

G1 sind drei neue Pakete, ihre Tests und ihre Golden-Files. Sonst nichts.

```
internal/code/model/      Node, Edge, Span, Graph — Lesemodell des Schemas
internal/code/pagerank/   Personalized PageRank, Prepare + Rank
internal/code/blast/      Gerichtete Kantenläufe, New + Reach
```

**Nicht in G1:** `internal/cli/commands.go`, `.loomux/config.toml`, `docs/**`,
die READMEs, `go.mod`. Kein Extraktor, keine lexikalische Saat, kein Schreiber
für `wiring.json`. Das Modell liest das Schema, geschrieben wird nichts.

Der Zuschnitt ist so gewählt, dass G1 **keine Datei anfasst, die `sdd-1b-1`
ebenfalls ändert**. Die Konfliktfläche der laufenden Stufe ist
`internal/cli/commands.go`, `internal/cli/dev.go`, `go.mod`, `go.sum`,
`docs/{en,de}/benchmarks.md` und `docs/{en,de}/cli-reference.md`; G1 berührt
keine davon. CLI-Verdrahtung und Dokumentation holt G2 nach, nicht G1.

**Elternordner `code/` statt `graph/`.** Die Säule-3-Spec legt die Pakete unter
`internal/graph/` an. Ein Verzeichnis dieses Namens neben dem vorhandenen
`internal/brain/graph` (dem Linkgraphen der Wiki-Dokumente) ist für einen Leser
verwechselbar, auch wenn der Compiler nichts dagegen hat: die Pakete heißen
`model`, `pagerank` und `blast`, `internal/brain/graph` heißt `graph`, eine
Namenskollision gibt es nicht. `internal/code/` benennt die Domäne, so wie
`internal/brain/` seine benennt, und kein Unterverzeichnis heißt `graph` — der
Gegenstand *ist* der Graph. `internal/brain/graph` bleibt unangetastet; ein
sprechenderer Name (`linkgraph`) wäre möglich, gehört aber nicht in diese Stufe.

## 3. Sechs Berichtigungen an §5.1 und §5.2

Gemessen an `src/ask/graphrank.ts`, `src/graph/relations.ts`,
`src/graph/types.ts` und `src/graph/traverse.ts` in Graft `1e352a3`.

### 3.1 Knoten-IDs tragen keinen Span

§5.1 legt die Tie-Ordnung auf `path:span#name` fest. Graft bildet IDs als
`path#Qualified.Name` — `src/cache.ts#Cache.get` — und hält den Span bewusst
heraus: IDs können ein Dedup-Ordinal tragen (`Cache.get~2`), und das Feld
`owner` existiert eigens, damit niemand die ID zerschneiden muss. Mit dem Span
in der ID verschlüsselte jede Zeilenverschiebung den Knoten neu und entwertete
die Pro-Datei-Karten aus §5.3.

**Es gilt:** Die ID ist für loomux eine undurchsichtige Zeichenkette. Die
alphabetische Tie-Ordnung bleibt, sie sortiert nach dieser ID.

### 3.2 Die Dangling-Masse geht proportional zur Saat zurück

§5.1 schreibt den Rückfluss als `(1-α)·m_dangling/|S| · s` und behauptet
daneben, die Gesamtmasse bleibe exakt 1. Beides zusammen ist falsch: `s` ist
bereits auf Summe 1 normiert, die zusätzliche Division durch `|S|` verliert
Masse. Graft verteilt `(1-α)·m_dangling` proportional zu den
Restart-Gewichten, ohne weiteren Teiler.

**Es gilt:** `next[i] += (1-α)·m_dangling·r[i]` mit `Σr = 1`.

### 3.3 Sechs Relationen, fünf davon laufbar

§5.2 zählt `calls`, `imports`, `extends`, `implements`, `references`. Im Schema
gibt es zusätzlich `contains` (Datei→Symbol, Klasse→Methode). Vom Lauf ist es
ausgeschlossen, und Graft begründet das: eine Datei „enthält" jedes Symbol in
ihr, als Kante machte sie jedes Geschwistersymbol zum Nachbarn und wirkte als
falscher Hub.

**Es gilt:** Das Modell kennt sechs Relationen. Laufbar sind die fünf; die
Frage beantwortet `Relation.IsWalk()`. `contains` wird dekodiert, weil das Schema es führt und jeder Lauf es
ausschließen muss — nicht, weil ein Lauf darauf liefe.

Der Blast-Radius eines Dateiknotens ist zwar die Vereinigung über die Symbole
der Datei, aber Graft findet sie **über Pfadgleichheit**, nicht über
`contains`: `symbolsInFile` nimmt jeden Knoten mit `kind != "file"` und
demselben `path`. So gilt es auch hier.

### 3.4 Kanten tragen eine Konfidenz, und ihr Ziel ist nicht immer ein Knoten

Das Schema führt auf jeder Kante `confidence`
(`lsp_resolved | lsp_dispatch | extracted | inferred`, stärkste zuerst), und
bei `imports` darf `target` eine unaufgelöste Modulzeichenkette statt einer
Knoten-ID sein.

**Es gilt:** Das Lesemodell dekodiert beides. Die beiden Rechner gehen damit
aber **verschieden** um, und das ist kein Versehen der Referenz:

- `pagerank` verwirft eine Kante, deren Enden nicht beide Knoten sind — sonst
  sammelte eine Modulzeichenkette Rangmasse und erschiene im Ergebnis.
- `blast` behält sie: ein unaufgelöstes Importziel ist ein Treffer mit `id`,
  Relation und Tiefe, aber ohne Knoten. Wer fragt „wovon hängt das ab",
  will `npm:lodash` sehen.

`confidence` wird in G1 gelesen und nicht ausgewertet — wer danach rangiert,
kommt später.

### 3.5 Die Ausgabe ist max-normiert

§5.1 nennt keine Ausgabenormierung und behauptet Gesamtmasse 1. Graft teilt am
Ende durch den größten Wert, gibt nur erreichte Knoten zurück und bei leerer
oder durchweg nicht-positiver Saat eine leere Menge.

**Es gilt:** Spitzenknoten 1,0; unerreichte Knoten fehlen; leere Saat ⇒ leeres
Ergebnis. „Masse exakt 1" ist die Invariante *innerhalb* der Iteration.

### 3.6 Was die Läufe über die Treffer sagen

§5.2 schweigt dazu. Graft schließt den Saatknoten aus seinen eigenen Treffern
aus, meldet jeden Knoten genau einmal bei der zuerst erreichten Tiefe (ein
Diamant konvergiert also auf einen Treffer), und `direction: in` schlüsselt die
Adjazenz auf `edge.target`.

**Es gilt:** ebenso.

Bestätigt hat der Abgleich `alpha = 0.25`, `iters = 25`, den ungerichteten Walk
(für „verstehe diese Gegend" ist der Aufgerufene so wichtig wie der Aufrufer),
die Spalten-Gradnormierung und den Dateinamen `wiring.json`.

## 4. Schnittstellen

### `internal/code/model`

Reine Daten mit JSON-Tags, ein eigenes Lesemodell statt Dekodierung in eine
Map — dieselbe Begründung, die `internal/brain/graph` schon trägt: ein vom
Erzeuger umbenanntes Feld darf nicht stumm bis zum Aufrufer durchrutschen.

`NodeID`, `Node` (ID, Name, Kind, Pfad, Span, Signatur, Owner), `Span`, `Edge`
(Quelle, Ziel, Relation, Konfidenz), `Relation` mit den sechs Werten,
`Relation.IsWalk()` für die fünf laufbaren, `Graph` als Bündel aus Knoten und
Kanten.

Dazu eine Frage, die keinen Lauf braucht und deshalb hier steht:
`SymbolsInFile(g, path)` gibt die Symbole einer Datei in Graphreihenfolge —
über Pfadgleichheit, nicht über `contains` (§3.3). Sie baut die Startmenge für
den Blast-Radius eines Dateiknotens.

`FileCard` gehört **nicht** zu G1: die Pro-Datei-Karten entstehen mit dem
Extraktor, vorher wäre der Typ Vorrat ohne Erzeuger.

### `internal/code/pagerank`

```go
func Prepare(g *model.Graph, keep func(model.NodeID) bool) Topology
func Rank(t Topology, seed map[model.NodeID]float64, opts Options) []Scored
```

`Prepare` scannt Knoten und Kanten einmal und baut die ungerichtete Adjazenz
über die laufbaren Relationen; `Rank` rechnet darauf. Die Trennung ist Grafts
`preparePageRankTopology` nachgebildet und zahlt sich aus, sobald eine Abfrage
mehrere Saaten über demselben Graphen fährt. `keep` ist der Teilgraph-Filter:
eine Kante zählt nur, wenn beide Enden ihn passieren.

`Options` sind `Alpha` (Standard 0,25) und `Iterations` (Standard 25).
`Scored` ist ID und Wert, absteigend sortiert, bei Gleichstand aufsteigend nach
ID.

### `internal/code/blast`

```go
func New(g *model.Graph) *Index
func (x *Index) Reach(start []model.NodeID, dir Direction, depth Depth) []Hit
```

`New` baut die gerichtete Adjazenz einmal. `Direction` ist `In` (wer hängt von
mir ab) oder `Out`; `Depth` ist eine Zahl oder „alle". Einen Pfadpräfix-Filter
gibt es hier **nicht**: in Graft verengt `--in` die Symbolauflösung, nicht die
Treffer (`traverse-cli.ts` reicht ihn an `resolveSymbol`, `edgeWalk` kennt ihn
nicht). Er gehört damit zur Abfrageschicht. `Hit` trägt ID, Relation und die Tiefe des ersten Erreichens, dazu
den Knoten — der fehlt, wenn die ID ein unaufgelöstes Importziel ist (§3.4).

Ein Dateiknoten als Startpunkt läuft über sich selbst **und** jedes Symbol
derselben Datei; das ist Sache des Aufrufers, der die Startmenge bildet, und
`Reach` nimmt deshalb eine Liste.

Dass PageRank denselben Kanten ungerichtet begegnet und Blast gerichtet, ist
Absicht und steht in beiden Paketkommentaren.

Symbolauflösung — aus `Cache.get` oder `pkg.Fn` einen Knoten machen — ist
**nicht** in G1. Sie gehört zur Abfrageschicht; `Reach` bekommt IDs.

## 5. Nachweise

**Portiert werden Grafts Testvektoren auf handgebauten Graphen**, gelesen, nie
ausgeführt; MIT erlaubt das, jede Golden-Datei trägt
`trailhq/Graft @ 1e352a3 (MIT)` als Herkunftszeile.

| Quelle | Fälle | Ziel |
|---|---:|---|
| `test/graphrank.test.ts`, Z. 54–169 | 9 | `internal/code/pagerank` |
| `test/graph-traverse.test.ts`, ab Z. 190 | 12 | `internal/code/blast` |

Nicht portiert werden Grafts zwei Partitionstests (Z. 182–265,
`preparePageRankPartitions`): das ist eine Beschleunigung für Abfragen über
mehrere Geltungsbereiche desselben Graphen, die G1 nicht hat. `Prepare` mit
seinem Filter deckt dieselbe Rechnung ab; kommt die Mehrbereichsabfrage, kommt
der Test mit ihr.

Darunter: Cluster schlägt Isoliertes, Nachbarn ohne eigenes Restart-Gewicht
sammeln Masse, leere und nicht-positive Saat, Kanten auf Nicht-Knoten,
Dangling-Pooling algebraisch gleich der Verteilung je Knoten, Teilgraph-Filter;
sowie `callersOf`/`calleesOf` ohne `contains`, Diamant auf kleinste Tiefe
dedupliziert, Tiefenkappe, Saat aus den eigenen Treffern, Dateiknoten als
Vereinigung über seine Symbole.

**Draußen bleiben** die vier `ask`-Integrationstests (sie brauchen den
Extraktor, also G2) und die zwölf `resolveSymbol`-Tests (Abfrageschicht).

**Kreuzprobe, ausgeführt am 2026-09-16:** die vier Konstanten des
Dangling-Falls (`a = 0.9577162737326514`, `b = 1`, `c = 0.37462976423958994`,
`d = 0.29154325474653076`) unabhängig nachgerechnet, im Sitzungs-Scratchpad,
Wegwerf. Drei Werte stimmen exakt, `c` weicht um 1 ULP ab — Summationsreihenfolge,
nicht Verfahren. Die Masse vor der Normierung ist exakt 1,0, was §3.2
bestätigt. Der Go-Test vergleicht deshalb mit Toleranz 1e-9, wie Graft es tut,
und nicht auf Gleichheit.

**Wo die Nachweise liegen.** Die Vektoren sind handgebaute Graphen mit wenigen
Knoten; sie stehen als Go-Literale im Testquelltext, wie sie in Graft als
TypeScript-Literale stehen, jeweils mit ihrer Herkunftszeile. Eine JSON-Datei
daraus zu machen hieße, für `pagerank` und `blast` einen Dekoder in den
Testpfad zu ziehen, den beide Pakete gar nicht kennen — sie nehmen ein
`model.Graph`. `testdata/` bekommt deshalb nur `internal/code/model`: dort ist
der Dekoder der Gegenstand, und die Datei zeigt ein `wiring.json` mit allen
sechs Relationen, einer Konfidenz je Wert, einem unaufgelösten Importziel und
einer ID mit Dedup-Ordinal.

Beides liegt jedenfalls unter `internal/code/`, nicht unter
`testdata/cases/`. §9.4 der Säule-3-Spec sagt das andere; `testdata/cases/`
ist laut seiner eigenen README ausschließlich die Paritätsakte gegen die
ersetzten Werkzeuge — Aufzeichnungen alter Programme, über eine
Übersetzungstabelle als loomux-Kommandozeilen nachgespielt. G1 hat kein
ersetztes Werkzeug und keine Kommandozeile.

## 6. Tor

`.githooks/pre-commit` bleibt unverändert und gilt: `gofmt`, `go vet`,
`go test ./... -count=1 -covermode=set -coverpkg=./...`, `dev covergate` mit
100 % je Funktion, dann das Pilot-Binary. Die Fehlerarme des JSON-Dekoders
gehören damit von Anfang an in die Tests.

Danach die Mutationsrunde der Stufe:
`loomux dev mutants internal/code/pagerank internal/code/blast`. Überlebende
bekommen ihre Verfügung in `docs/.superpowers/parity/code-g1.md`.

## 7. Weg zurück

`code-g1` zweigt von `sdd-1b-1` ab, weil `loomux dev mutants` dort entstanden
ist und auf `master` fehlt. Die Reihenfolge ist damit festgelegt: erst geht
1b-1 nach `master`, dann `code-g1` hinterher. Schreibt 1b-1 seine Historie vor
dem Merge um, wird `code-g1` darauf umgesetzt.

Die Schreibschranke braucht für den Worktree keinen eigenen
Registry-Eintrag — ein verknüpfter Worktree eines `workspace`-Bereichs ist
seit dem 2026-09-15 freigegeben (`docs/.superpowers/parity/schranke-worktrees.md`).
Im Worktree ist einmal `go build -o bin/loomux.exe ./cmd/loomux` nötig, damit
die Hooks dort ein Binary finden.

## 8. Was G1 offen lässt

- Lexikalische Saat (BM25 / Substring auf Namen und Dokumentations-Tokens):
  §5.1 nennt den Saatvektor eine Eingabe, sie ist Textarbeit und braucht
  Namen aus dem Extraktor. G2 oder später.
- Extraktor, `wiring.json`-Schreiber, Frischeprüfung, CLI, Konfiguration: G2.
- Die Inittrace-Messung mit `go-sdk`: G3, nicht G1 — dort zieht
  `internal/serve` das SDK ein. Im Stufenplan der Säule-3-Spec (§10) steht sie
  in der Zeile G1 und ist dort fehl am Platz.
- `FileCard`, `confidence`-Rangfolge, LSP-Anreicherung: mit ihren Erzeugern.
