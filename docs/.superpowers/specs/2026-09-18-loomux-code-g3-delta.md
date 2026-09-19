# loomux G3 — Delta zur Säule-3-Spec

**Datum:** 2026-09-18
**Stand:** umgesetzt auf Branch `code-g3` am 2026-09-19; Plan
[`2026-09-18-loomux-code-g3.md`](../plans/2026-09-18-loomux-code-g3.md); die Paritätsliste
[`parity/code-g3.md`](../parity/code-g3.md) ist vom Nutzer freigegeben (2026-09-19)
**Ergänzt:** [`2026-09-14-loomux-code-graph-design.md`](2026-09-14-loomux-code-graph-design.md),
[`2026-09-16-loomux-code-g1-delta.md`](2026-09-16-loomux-code-g1-delta.md) und
[`2026-09-17-loomux-code-g2-design.md`](2026-09-17-loomux-code-g2-design.md). Diese Datei ersetzt die
Säule-3-Spec nicht, sie berichtigt und verengt sie für die Stufe G3.
**Referenz:** `trailhq/Graft`, Commit `1e352a3`, MIT. Gelesen: `src/mcp/tools.ts`,
`src/mcp/tool-names.ts`, `src/graph/refresh.ts`, `src/context/check.ts`.
**Arbeitsort:** Worktree `C:\Users\micro\Documents\#GIT\loomux-code-g3`, Branch `code-g3`,
abgezweigt von `sdd-1b-2` `11222c8`.

## 1. Warum es dieses Delta gibt

§10 der Säule-3-Spec gibt G3 das Gateway in `internal/serve`, die Kanaltrennung und sechs
MCP-Werkzeuge. Zwei Dinge haben sich seither verschoben:

- **Das Gateway steht schon.** Stufe 1b-2 hat `loomux serve` mit zwei Listenern (Kanal = Adresse,
  zwei Token), die stdio-Brücke `loomux mcp` und die geteilte Werkzeugliste `internal/mcptools`
  gebaut. Die „Server-Profile `local`/`cloud`" aus §4.2 der Säule-3-Spec sind damit dieselbe Sache
  unter anderem Namen, und es gilt die Lösung von 1b-2. G3 hängt sich an, es baut kein zweites
  Gateway.
- **Nur zwei der sechs Werkzeuge haben eine Abfrageschicht.** `graph_find_code` steht auf
  `internal/code/ask` (G2b), `graph_check_freshness` auf der Neuextraktion hinter `graph check`
  (G2a). Für `graph_trace_calls` fehlt die Symbolauflösung — `internal/code/resolve` löst Kanten auf,
  nicht Namen —, für `graph_file_api`, `graph_find_all` und `graph_repo_map` gibt es noch nichts.

**Es gilt:** G3 liefert die zwei Werkzeuge, die G2 trägt, und die Verdrahtung dazu. Die übrigen vier
kommen in G4, jedes zusammen mit seinem CLI-Geschwister (`callers`, `skeleton`, `grep`, `map`). Das
ist die Reihenfolge, die 1b-2 vorgemacht hat: erst die Abfrage, dann MCP als dünne Schicht darüber.

**Branchbasis.** `internal/serve` und das MCP-SDK gibt es nur auf `sdd-1b-2`; `sdd-1b-2` enthält seit
dem 2026-09-18 den Stand von `master` mit G2b. `code-g3` zweigt deshalb von `sdd-1b-2` ab, und die
Reihenfolge ist wie bei G1 festgelegt: erst geht 1b-2 nach `master`, dann `code-g3` hinterher.
Schreibt 1b-2 seine Historie vor dem Merge um, wird `code-g3` darauf umgesetzt.

## 2. Zuschnitt

```
internal/code/query/     Build, Ask, Check und die Textberichte, herausgelöst aus internal/cli/graph.go
internal/code/ask/       ein Prädikat Keep in Options, neben In
internal/serve/graph/    Register: zwei Handler, so dünn wie internal/serve/brain
internal/mcptools/       zwei Werkzeugdefinitionen mehr, sieben insgesamt
internal/serve/serve.go  graph.Register neben servebrain.Register
internal/cli/graph.go    Flags und Exit-Codes über query
```

**Nicht in G3:** die vier übrigen Werkzeuge, Föderation über mehrere Bereiche, Upstream-Proxies
(§4.3 der Säule-3-Spec), Hook-Anbindung, `check`-Lanes.

## 3. Berichtigungen

### 3.1 Einen ersten Graphen baut keine Abfrage

`ask.EnsureFresh` baut heute neu, sobald der Frische-Record fehlt — auch dann, wenn es gar keinen
Graphen gibt. G2 §8.1 begründet „ohne Record unbekannt, nicht frisch", den Fall „kein Graph"
entscheidet es nicht ausdrücklich. Graft entscheidet ihn: `ensureFreshGraph`
(`src/graph/refresh.ts`, Z. 156–170) kehrt ohne Bau zurück, wenn `wiring.json` fehlt, und das
Werkzeug meldet „no graph found". Die Begründung dort: ein Vollbau unter einer Abfrage ist eine
Überraschung, und es ist der eine Fall, in dem der Nutzer das Werkzeug noch gar nicht eingeschaltet
hat.

Über MCP wiegt das schwerer als auf der Kommandozeile: jedes Modell, auch eines auf dem
Cloud-Kanal, könnte über einen beliebigen sichtbaren `scope` einen Vollbau auf der Maschine des
Nutzers auslösen.

**Es gilt:** `query.Ask` prüft vor `EnsureFresh`, ob der Graph existiert. Fehlt er, gibt es
`ErrNoGraph` zurück und baut nicht. Auffrischen ja, Erstbau nie — für MCP **und** CLI, eine Regel.
Für `loomux graph ask` ist das eine Verhaltensänderung: ohne Graph Exit 1 mit „no graph. Run
`loomux graph build` first." und keine neu entstandene `wiring.json`. Fehlt nur der Record, bleibt
es bei G2: unbekannt, also neu bauen.

### 3.2 `graph_check_freshness` kennt keinen Fehlerarm für Drift

Graft gibt aus `graft_check_freshness` immer `isError: false` zurück; „NO GRAPH" ist ein Berichtstext
(`src/context/check.ts`, Z. 156–158). loomux stimmt damit schon überein: `Missing` ist ein Feld des
Ergebnisses, kein Fehler, und `graph check` exitet 1 wegen `!OK`, nicht wegen eines Fehlers.

**Es gilt:** Drift und fehlender Graph sind Text. `isError` nur bei einem echten Lesefehler.

Grafts Werkzeug hängt zwei Berichte aneinander: den der `.context/`-Schicht und den des Graphen.
loomux hat keine `.context/`-Schicht; es bleibt der Graphbericht.

### 3.3 Die Werkzeugparameter

Gegen `src/mcp/tools.ts` (Z. 44–80) gelesen, nicht aus §4.1 der Säule-3-Spec übernommen:

| Werkzeug | Parameter |
|---|---|
| `graph_find_code` | `scope` (Pflicht), `query` (Pflicht), `limit` (Standard **5**), `full`, `in` |
| `graph_check_freshness` | `scope` (Pflicht) |

- `limit` ist über MCP 5, auf der Kommandozeile 8 — beides Grafts Werte (MCP `tools.ts`, CLI
  `cli.ts:595`; G2 §9 nennt `:596`).
- Der Quelltext ist über MCP **immer** eingefügt (`source: true` in Grafts Handler); `full` wählt
  den ganzen Span statt des gekappten Auszugs.
- **`scope` ist loomux' Zusatz.** Graft bedient ein Repo je Server; `loomux serve` bedient alle
  registrierten Bereiche. Ein Graph gehört zu genau einem Repo, also ist `scope` Pflicht.
  `scope = "all"` wäre Föderation und ist nicht in G3.

### 3.4 Der Privacy-Filter wirkt vor dem Ranking

§4.2 der Säule-3-Spec sagt, der Cloud-Kanal „maskiere" Pfade aus `local_only`-Bereichen, ohne
Mechanismus. **Es gilt dieselbe Regel wie für `brain_*`:**

- Ein Bereich, dessen Manifest `local_only` erklärt, existiert auf dem Cloud-Kanal nicht:
  `privacy.VisibleAreas` löst ihn gar nicht erst auf, und die Antwort ist „unknown scope" wie bei
  `brain_*`. Das geschieht vor jedem Zugriff auf den Graphen — dieselbe Begründung, die
  `VisibleAreas` schon trägt: schon die Abfrage wäre eine Preisgabe.
- Auf beiden Kanälen fallen Knoten weg, deren Pfad unter `NeverGlobs` des Manifests fällt
  (`privacy.IsReadable`). Das geschieht **vor dem Scoring**, über ein Prädikat `Keep` in
  `ask.Options` an derselben Stelle, an der `In` greift. Ein Filter nach dem Ranking ließe den
  Knoten mitranken, Rangmasse sammeln und einen Platz des `limit` belegen. Der Driftbericht von
  `graph_check_freshness` lässt dieselben Pfade weg. Auf dem lokalen Kanal nennt er die Zahl
  der weggelassenen Einträge, auf dem Cloud-Kanal nicht: schon die Zahl sagte einem entfernten
  Modell, dass sich unter den `never`-Globs etwas geändert hat. Der Bericht bleibt in beiden
  Fällen DRIFT und wird nie OK (entschieden vom Nutzer am 2026-09-19).

## 4. Schnittstellen

### `internal/code/query`

Der Kern dessen, was heute in `internal/cli/graph.go` rechnet, damit `serve` es erreicht, ohne
`internal/cli` zu importieren — dasselbe Muster wie `internal/brain/answer`.

- `Build(root)` ist das heutige `writeEverything`: Graph, Beiakte, Frische-Record. Die Sequenz gibt es
  danach genau einmal; der Kommentar an `writeEverything` verlangt das, weil eine vergessene
  Beiakte eine stumm schlechtere Antwort ist.
- `Ask(root, question, AskOptions) (ask.Answer, []string, error)` — `AskOptions` sind `Limit`,
  `In`, `Source`, `Full`, `NoRefresh` und `Keep`. Ablauf: Graph vorhanden? sonst `ErrNoGraph`
  (§3.1) → `EnsureFresh` mit `Build` → `store.Read` → `lexicon.Read`, bei Fehlschlag
  `lexicon.Build` mit Hinweis → `ask.Run` → bei `Source` `ask.Inline`. Hinweise kommen als
  Liste zurück, nicht nach stderr.
- `Check(root) (Drift, error)` ist das heutige `check`: Neuextraktion gegen den geschriebenen
  Graphen, **ohne** den Frische-Record zu lesen (G2 §8.2). `Drift` trägt `Missing`.
- `AskReport` und `CheckReport` sind die heutigen Textformatierer; CLI und MCP liefern denselben
  Text.

### `internal/serve/graph`

`Register(server, channel, Deps)` mit `Deps{RegistryDir, LegacyDir, Ask, Check}`. `Ask` und `Check`
sind Funktionsfelder, damit ein Test ohne Baum auskommt; der Kanal ist der des Listeners, nie ein
Argument — wie in `serve/brain`.

### `internal/mcptools`

Sieben Werkzeuge in fester Reihenfolge: die fünf `brain_*`, dann `graph_find_code`,
`graph_check_freshness`. Die Brücke beantwortet `tools/list` selbst; eine Liste, die von der von
`serve` abwiche, wäre der schlimmste Fehler dieser Schicht (Paketkommentar), also stehen beide
Werkzeuge hier und nirgends sonst.

## 5. Datenfluss und Fehlerverhalten

**`graph_find_code`:**

1. `scope` oder `query` leer ⇒ `isError` „graph_find_code requires a scope" bzw. „… a query".
2. `VisibleAreas(…, channel)` → `Single` ⇒ `Area.Path` (die Repo-Wurzel) und Manifest; unsichtbar
   oder unbekannt ⇒ `isError` „unknown scope".
3. `query.Ask(root, q, {Limit: 5, In, Source: true, Full, Keep})`.
4. `ErrNoGraph` ⇒ `isError` mit „no graph. Run `loomux graph build` first."
5. Auffrisch-Hinweise stehen **vor** der Antwort (Graft `callTool`: `${note}\n${res.text}`) und
   gehen zusätzlich als Fortschritt hinauf. Grund für den Platz: ein Auffrisch-Hinweis erklärt die
   Antwort, also steht er vorn. `withFindings` in `serve/brain` hängt seine Befunde hinten an, weil
   ein Befund die Antwort entwertet. Zwei Arten Hinweis, zwei Plätze. **Auf dem Cloud-Kanal
   gehen keine Hinweise hinaus**, weder im Text noch als Fortschritt: ein Hinweis zählt auch
   Dateien unter den `never`-Globs, und der Hinweis eines gescheiterten Neubaus zitiert den
   Parse-Fehler einer verborgenen Datei (Korrektur nach dem Abschlussreview, 2026-09-19).
6. Jeder andere Fehler aus `query.Ask` oder `query.Check` ist auf dem Cloud-Kanal der feste Text
   „the graph could not be read on this channel; ask on the local channel for details": ein
   Parse-Fehler nennt Datei, Zeile und Token, ein Lesefehler einen absoluten lokalen Pfad.
   `ErrNoGraph` behält seinen Text. Der lokale Kanal sieht den Fehler selbst.

**`graph_check_freshness`:** Schritte 1–2 wie oben, dann `query.Check(root)` **ohne** Auffrischen —
Grafts `NO_REFRESH_TOOLS` (`tools.ts`, Z. 201–203) mit demselben Grund: sonst meldete das Werkzeug
über einen Graphen, den es eben selbst repariert hat, also immer „OK". Drift ist Text (§3.2).

**Paniken:** jeder Handler fängt sie und gibt `isError` mit der Meldung zurück. Graft verspricht
dasselbe (`callTool` wirft nie).

**Graph und Beiakte aus zwei Bauten.** `wiring.json` und `ask-index.json` werden je für sich atomar
geschrieben (temp + rename), aber nicht gemeinsam. In einem langlebigen Dienst ist der Leser
während eines Neubaus kein Randfall mehr: er kann einen neuen Graphen mit alter Beiakte lesen.
`ask.Run` fängt das schon ab (`ask.go`, Schleife über `ix.Docs`): ein Dokument ohne Knoten fällt
weg, ein neuer Knoten ohne Dokument bekommt keinen lexikalischen Score. Die Antwort ist schlechter,
nie falsch, und die nächste Frage ist wieder frisch. Das bleibt so.

## 6. Nachweise

TDD, 100 % je Funktion.

- **`query`:** Die Tests der heutigen `cli/graph.go`-Logik ziehen mit um. Neu: `Ask` ohne Graph gibt
  `ErrNoGraph` und legt keine `wiring.json` an; `Check` ohne Graph gibt `Missing` und keinen
  Fehler; `Keep` wirkt vor dem Scoring — ein gefilterter Knoten fehlt, und `limit` wird trotzdem
  voll.
- **`ask`:** ein Test für `Keep` neben dem für `In`.
- **`serve/graph`** mit ersetzten `Ask`/`Check`: Pflichtparameter; „unknown scope" für einen
  `local_only`-Bereich auf dem Cloud-Kanal, sichtbar auf dem lokalen; Hinweise vor dem Text; Panik
  ⇒ `isError`; `check_freshness` bei Drift und bei fehlendem Graphen ohne `isError`.
- **`mcptools`:** genau sieben Werkzeuge in fester Reihenfolge, jeder Name auf
  `^[a-zA-Z0-9_-]{1,64}$`, Pflichtfelder gesetzt; die Brücke beantwortet `tools/list` identisch zu
  `serve`.
- **Ende zu Ende über HTTP:** ein `serve` im Test beantwortet `graph_find_code` auf einem im Test
  registrierten Mini-Repo — das Gerüst, das 1b-2 für `brain_*` hat.
- **CLI:** `graph ask` ohne Graph exitet 1 mit der Meldung und baut nicht (§3.1); die übrigen
  Befehle verhalten sich unverändert, die bestehenden Tests halten das fest.

## 7. Tor, Mutationen, Messung, Doku

**Tor:** `.githooks/pre-commit` unverändert.

**Mutationen:** `loomux dev mutants internal/code/query internal/serve/graph`. Überlebende und die
Abweichungen gegen Graft bekommen ihre Verfügung in `docs/.superpowers/parity/code-g3.md`.

**Messung.** G1 und G2 haben die Inittrace-Messung mit dem MCP-SDK an G3 verwiesen. 1b-2 hat das SDK
gelinkt und `cmd/loomux/start_test.go` hält die Regel (keine Paketinitialisierung über 500
Allokationen). Damit ist sie erledigt, und G3 ändert daran nichts: `extract/golang` und
`go/parser` sind seit G2a in jedem `loomux`-Prozess gelinkt, weil `internal/cli/graph.go` sie
importiert und `cmd/loomux` `internal/cli` linkt. `query` ist verschobener Code, `serve/graph`
hat kein `init`. Nachzuweisen sind deshalb nur:

1. `start_test.go` bleibt grün.
2. Ein datierter Eintrag in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`: `loomux --version`
   kalt und warm vor und nach G3 als **Kontrolle** (erwartet: gleich innerhalb des Rauschens), und
   als neue Zahl die Antwortzeit von `graph_find_code` über die Brücke gegen `graph ask` auf
   demselben Repo.

**Doku:** `README.md` und `README.de.md` (Werkzeugliste, Stufenstand);
`docs/en/cli-reference.md` und `docs/de/cli-reference.md` (`serve` mit zwei Werkzeugen mehr,
`graph ask` baut keinen ersten Graphen); der Kopf der Säule-3-Spec bekommt den Stand „G3 umgesetzt"
mit Verweis auf dieses Delta.

## 8. Was G3 offen lässt

- `graph_trace_calls`, `graph_file_api`, `graph_find_all`, `graph_repo_map` mit ihren
  CLI-Geschwistern und der Symbolauflösung: G4.
- Föderation über mehrere Bereiche (`scope = "all"`).
- **Worktrees.** Die Registry kennt den Haupt-Checkout; ein Graph unter `.loomux/state/graph/` eines
  verknüpften Worktrees ist über `scope` nicht erreichbar. Graft kopiert dafür den Graphen des
  Haupt-Checkouts in den Worktree (`seedUnderLock`, `refresh.ts`). Später.
- Upstream-Proxies (§4.3 der Säule-3-Spec).
