# Go-Graph: Funktionswerte und verkettete Aufrufe

## Absicht

Ein Agent-Benchmark (Spikes zum Graph-Hinweis, 2026-11-11) hat gezeigt, dass
der Code-Graph Aufrufketten genau dort verliert, wo grep sie findet. Deshalb
greift jeder Agent zu Recht auf grep zurück, und ein Hinweis Richtung Graph
lohnt nicht. Drei Fälle aus dem Repo selbst:

- `Deps{Blast: query.Blast}` (`internal/serve/serve.go`): Eine Funktion wird
  als Wert in ein Struct-Feld gelegt; das MCP-Werkzeug ruft sie später über
  `deps.Blast(...)` auf.
- `postToolUse = hooks.PostToolUse` (`internal/cli/hook.go`): eine Funktion
  als Wert einer Paketvariablen.
- `blast.New(g).Reach(...)` (`internal/hooks/blast_monitor.go`): ein
  Methodenaufruf auf dem Ergebnis eines Aufrufs.

Ziel ist, dass `graph callers` und `graph blast` diese Ketten finden.

## Ziele

1. **Funktionswerte:** Wo eine Funktion als Wert steht, entsteht eine Kante
   `references` vom umschließenden Symbol (auf Paketebene: von der Datei)
   zur Funktion. `references` ist schon eine Walk-Relation; `callers`, `blast`
   und der Edit-Hinweis folgen ihr ohne weitere Änderung.
2. **Verkettete Aufrufe:** `f(...).M()` und `pkg.F(...).M()` ergeben eine
   Kante `calls` auf `T.M`, wenn `f` eindeutig auflöst und sein erstes
   Ergebnis ein Typ `T` oder `*T` desselben Pakets ist.
3. **Gebundene Ergebnisse:** `x := pkg.F(...)` gefolgt von `x.M()` löst über
   denselben Weg auf.
4. Messung vorher/nachher: deterministisch über `graph callers`, dann
   dieselben Agentenaufgaben wie im Spike (v0 und v3, je zwei Wiederholungen).

## Nicht-Ziele

- go/types oder gopls: Das Projekt bleibt bei go/parser (siehe Kopf von
  `internal/code/extract/golang/extract.go`).
- Interface-Dispatch, Rückgabetypen aus anderen Paketen (`*other.T`),
  Funktionswerte, die über Kanäle oder Maps weitergereicht werden.
- Der Graph-Hinweis aus Punkt 2 des Handovers: Er ist nach dem Spike
  verworfen.

## Anti-Ziele

- Kanten, die raten, wo vorher keine war: Jede neue Kante folgt denselben
  Eindeutigkeitsregeln wie heute (`one`), und eine Referenz über einen bloßen
  Namen sucht nie repo-weit.

## Entscheidungen (meine; Veto beim Lesen)

- **Wertpositionen, nicht jeder Bezeichner:** Wertpositionen sind die rechte
  Seite von Zuweisung und `var`, der Wert eines Schlüssel-Wert-Paares (nie der
  Schlüssel), die Elemente eines Composite-Literals, Aufrufargumente und
  `return`-Ergebnisse. Ein allgemeiner Lauf über alle Bezeichner würde
  Feldnamen und Typpositionen mitnehmen.
- **Bloßer Name `F`:** Ein Name aus einem inneren Scope ist eine lokale
  Variable und wird übersprungen. Sonst wird er in derselben Datei, dann im
  selben Paketverzeichnis aufgelöst, nie repo-weit: In Go ist ein
  unaufgelöster bloßer Bezeichner im selben Paket oder ein Builtin. Die
  Konfidenz ist `extracted`.
- **`pkg.F` als Wert:** Er läuft über dieselbe Selektor-Auflösung wie ein
  Aufruf. Ziele in Standardbibliothek und Fremdpaketen fallen heraus.
- **Methodenwerte `s.handle`:** Bei bekanntem Empfängertyp laufen sie über
  denselben Owner-Index wie Aufrufe.
- **Verkettung:** Die rohe Kante trägt den inneren Aufruf als `Via`, in
  derselben Form wie eine rohe Kante. Der Resolver löst `Via` mit den
  bestehenden Regeln auf und liest den ersten Ergebnistyp aus der
  `Signature` des Ziels. Die Methode wird nur im Verzeichnis dieses Ziels
  gesucht. Die Konfidenz ist die des schwächsten Glieds.
- **Extraktor-Version:** `go/2` wird `go/3`, weil der Extract-Cache auf ihr
  schlüsselt.
- **Folgen:** `InDegree`, Pagerank, `signal` und `blast-audit` zählen
  `references` mit. Das Delta von `check blast-audit` auf jüngsten Commits
  wird gemessen und berichtet.

## Messung

- Deterministisch: `graph callers --depth 6` für `Radius`, `relInRoot` und
  `Reach`, gegen die grep-Wahrheit; Kantenzahl aus `graph stats`;
  Stichprobe von etwa 20 neuen Kanten auf Fehlkanten; `graph build
  --no-reuse` vorher und nachher.
- Agenten: m1–m3 aus dem Spike mit v0 und v3, zwei Wiederholungen, neues
  Binary über `PATH` und `--mcp-config`. Die Aufgaben stammen aus den
  bekannten Lücken des Graphen; das steht im Bericht.
- Ergebnis in `docs/de|en/benchmarks.md` und `docs/wiki/topics/code-graph.md`.
