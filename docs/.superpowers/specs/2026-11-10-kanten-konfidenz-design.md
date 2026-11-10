# Kanten-Konfidenz sichtbar machen: Design

Stand 2026-11-10. Anlass: Vergleich mit graphify (v0.9.105), das jede Kante
als `EXTRACTED` oder `INFERRED` ausweist. Zweiter Punkt aus demselben Vergleich
(Guard-Hinweis auf den Graphen bei `Read`/`Grep`) ist ein eigenes Design und
ein eigener PR.

## Anlass und Ziel

Der Graph kennt die Sicherheit jeder Kante schon: `model.Edge.Confidence`
(`internal/code/model/graph.go`) mit `lsp_resolved`, `lsp_dispatch`,
`extracted`, `inferred`. `inferred` setzt der Resolver nur, wenn er einen
Aufruf allein über einen global eindeutigen Namen zuordnet
(`internal/code/resolve/resolve.go`, `python.go`). Im Graphen dieses Repos sind
das 7700 von 27733 `calls`-Kanten (28 %); `contains` und `imports` sind immer
`extracted`. Weil der Walk nach dem schwächsten Glied markiert, trägt in
tieferen Ebenen grob jeder zweite Treffer die Markierung oder mehr.

Keine Ausgabe zeigt das Feld. `blast.Index` wirft es beim Aufbau weg. Ein
Agent, der `graph callers` oder `graph_blast` liest, hält deshalb eine
geratene Kante für so sicher wie eine belegte. Ziel: Er sieht, welche Treffer
auf einer geratenen Kante stehen, und prüft gezielt nur diese nach.

## Ziele

1. `graph callers` (CLI) und `graph_trace_calls` (MCP) markieren jeden Treffer,
   dessen Weg eine `inferred`-Kante enthält.
2. `graph blast` (CLI) und `graph_blast` (MCP) markieren ebenso jeden
   erreichten Treffer, dessen Weg eine `inferred`-Kante enthält (schwächstes
   Glied des Weges).
3. Die JSON-Ausgabe (`--json`) trägt das Feld für jeden Treffer.
4. Der Hook nach einem Edit (`internal/hooks/blast_monitor.go`, „callers in
   other files“) markiert geratene Aufrufer ebenso, z. B.
   `  Run (cmd/x.go, inferred)` (Entscheidung des Menschen).

## Nicht-Ziele

- `graph grep` / `graph_find_all`: Das ist eine Textsuche ohne Kanten; dort
  gibt es nichts zu markieren. (Die Übergabe nannte es irrtümlich mit.)
- `graph ask`: Ranking bleibt unverändert.
- `lsp_resolved` / `lsp_dispatch` bekommen keine eigene Darstellung; sie gelten
  wie `extracted` als sicher.
- Kein Flag, das unsichere Treffer ausblendet (Entscheidung des Menschen:
  nur markieren).
- Keine neue Konfidenzstufe, kein Score.
- Unverändert bleiben alle Zählungen und Signale, die geratene Kanten heute
  mitzählen: das Test-Signal je Bereich in `blast` (`signal` in
  `internal/code/blast/radius.go`), `InDegree`/`InDegreeWhere`, `audit`,
  `pagerank` und `ask`. Eine geratene Kante zählt dort weiter voll.

## Anti-Ziele

- Rauschen: Sichere Treffer bekommen keinen Zusatz. Wer heute die Ausgabe
  liest, sieht sie unverändert, solange keine `inferred`-Kante beteiligt ist.
- Kein Bruch der JSON-Leser: Das Feld kommt hinzu, nichts wird umbenannt.

## Vom Menschen entschieden

- Markiert wird nur `inferred`; `extracted` bleibt still.
- `blast` markiert nach dem schwächsten Glied des Weges und blendet nichts aus.
- `ask` und `lsp_*` bleiben unberührt.
- Eigener PR, getrennt vom Guard-Hinweis. Dieser PR (Konfidenz) kommt zuerst
  und zweigt von `master` ab; der Guard-PR baut danach auf dem Stand nach dem
  Merge auf.

## Bau

### Datenfluss

- `blast.link` nimmt die Konfidenz der Kante auf; `blast.New` füllt sie.
- `blast.Hit` bekommt ein Feld `Inferred bool` mit JSON-Namen `inferred`,
  `omitempty`. *Meine Wahl:* ein bool statt des vollen Konfidenzwerts, weil
  nur „geraten ja/nein“ eine Aussage trägt und der Pfadwert ohnehin ein
  Aggregat ist, kein Kantenwert.
- `EdgeWalk` Tiefe 1: `Inferred` = die eine Kante ist `inferred`.
- `Reach` (BFS): `Inferred` = die Kante zum Treffer ist `inferred` oder der
  Vorgänger auf dem Weg war schon `Inferred`. *Meine Wahl:* Erreicht die BFS
  einen Knoten auf derselben Tiefe über mehrere Wege, gilt er als sicher,
  sobald einer dieser Wege sicher ist (ein bereits gesetzter Treffer derselben
  Tiefe wird auf sicher hochgestuft und vererbt das an die nächste Ebene,
  sofern diese noch nicht verarbeitet ist — die BFS arbeitet ebenenweise, also
  ist sie es nicht). Ein sicherer Weg größerer Tiefe zählt nicht; dass die BFS
  nur kürzeste Wege kennt, ist bekannte Grenze. *Meine Wahl:* Beim Hochstufen
  übernimmt der Treffer auch die `Relation` der sicheren Kante. Sonst stünde
  z. B. `calls` ohne Markierung da, obwohl gerade die `calls`-Kante geraten
  war (parallele Kanten A→B mit `calls`/inferred und `references`/extracted
  kommen vor).
- `Radius` führt Treffer mehrerer geänderter Dateien zusammen und behält die
  kleinste Tiefe. *Meine Wahl:* Bei gleicher Tiefe gewinnt sicher; bei
  kleinerer Tiefe übernimmt der Treffer `Relation` und `Inferred` des
  kürzeren Weges.
- `query.CallerHit` übernimmt `Inferred` aus dem `blast.Hit`.

### Darstellung (Text, CLI und MCP)

MCP liefert dieselben Textberichte wie die CLI (`CallersReport`,
`BlastReport`); eine Änderung dort deckt beide Kanäle ab.

*Meine Wahl:* Das Wort `inferred` als letztes Element in der Klammer, die schon
Art, Tiefe und Relation trägt:

```
  - Run (function, depth 1, calls, inferred) in cmd/x.go:12-30
  Run (function, depth 2, calls, inferred) in cmd/x.go:12-30 from a.go
```

Kein Symbol, keine Legende: Das Wort erklärt sich selbst und kostet einen
Agenten weniger als ein Zeichen, das er deuten müsste.

### Wiki

`docs/wiki/topics/code-graph.md` im selben PR: ein Absatz, was `inferred`
bedeutet (Zuordnung nur über eindeutigen Namen im ganzen Repo), dass Walks es
nach dem schwächsten Glied weitergeben und wo es erscheint.

## Dem Bauenden überlassen

- Testaufbau (TDD, 100 % Coverage nach Projektregel): kleine Graphen mit
  gemischten Kanten für `EdgeWalk`, `Reach` (Diamant mit einem sicheren und
  einem geratenen Arm, Kette mit geratener Mitte), `Radius` (zwei Dateien
  erreichen denselben Knoten) und die beiden Reports.
- Ob bestehende Golden-Tests (`internal/code/query/golden_test.go`) neue
  Erwartungen brauchen.
- Ob die Werkzeugbeschreibungen in `internal/mcptools/tools.go` einen
  Halbsatz zu `inferred` bekommen; empfohlen, damit ein Agent das Wort ohne
  Wiki versteht.
