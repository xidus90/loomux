# Paritätsliste Code-Graph G3

**Quelle:** `trailhq/Graft` @ `1e352a3` (MIT), `src/mcp/tools.ts`, `src/mcp/tool-names.ts`,
`src/graph/refresh.ts`, `src/context/check.ts`, `src/cli.ts`.
**Spec:** [2026-09-18-loomux-code-g3-delta.md](../specs/2026-09-18-loomux-code-g3-delta.md)
**Plan:** [2026-09-18-loomux-code-g3.md](../plans/2026-09-18-loomux-code-g3.md)
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor die Stufe als
fertig gilt.

**Stand 2026-09-19: freigegeben.** Der Nutzer hat am 2026-09-19 alle offenen
Zeilen freigegeben, in drei Blöcken: §1 (17 Verfügungen gegen die Referenz),
die Änderungen der Korrekturrunden 1 und 2 (§3, §4) und die zwei offenen
Überlebenden von Runde 5 (§5). Zeilen, die „abgelöst durch §5" oder
„überholt" tragen, stehen nicht zur Freigabe; die mit „freigegeben in G2b"
sind unverändert aus G2b übernommen.

**Commits.** Der Branch ist nach Korrekturrunde 2 auf `master` umgesetzt und in
vier Commits gruppiert worden (PR #8): `feat(ask)`, `feat(graph)!`,
`feat(serve)` und `docs(graph)`. Die
Commit-Hashes in §2 bis §4 stammen aus der Entwicklung davor und existieren
nicht mehr; was sie änderten, steckt in diesen vier. Beim Umsetzen kam aus
`master` die Korrektur „ein Bau steht, wenn nur der Frische-Record nicht
geschrieben werden kann" dazu, die jetzt in `query.Build` sitzt
(`notice`-Argument). Alle Zeilenangaben in §1 und §5 gelten für den Code dieses Stands; der
Test, den Runde 5 nachgetragen hat (§5), ist in `feat(graph)!` gefaltet.

Das Tor ist grün auf dem Kopf von PR #8 (in jedem der vier Commits vom
Pre-Commit-Hook gefahren), Worktree
`C:\Users\micro\Documents\#GIT\loomux-code-g3`, Branch `code-g3`:

```
gofmt -l cmd internal
go vet ./...
go test ./... -count=1 -covermode=set "-coverpkg=github.com/xidus90/loomux/..." -coverprofile=coverage.out
go run ./cmd/loomux dev covergate --profile coverage.out
go build -o bin/loomux.exe ./cmd/loomux
```

`gofmt` und `vet` schweigen, alle Pakete mit Tests grün (`internal/testlock`
hat keine eigenen), `covergate` endet mit 0. In den Paketen dieser Stufe —
`internal/code/query`, `internal/serve/graph`, `internal/mcptools` — steht
kein `//coverage:exempt`; die Ausnahmen in `internal/serve/serve.go` und
`internal/serve/state.go` stammen aus 1b-2 und sind unverändert.

Die Graft-Stellen unten sind gegen die Referenzdateien auf `1e352a3`
nachgelesen: `src/mcp/tools.ts`, `src/mcp/tool-names.ts`,
`src/graph/refresh.ts`, `src/context/check.ts` und `src/cli.ts`. Der
Kommandozeilen-Standard 8 steht in `cli.ts:595`
(`.option("-n, --limit <n>", "max results", "8")`), nicht in `:596`, wie G2 §9 sagt;
die Spec §3.3 ist berichtigt.

## 1. Verfügungen gegen die Referenz

| Punkt | Graft | loomux | Art | Freigabe |
|---|---|---|---|---|
| Werkzeugnamen | `graft_find_code`, `graft_check_freshness` (`tools.ts:44`, `:77`) und die Aliase `graft_ask`, `graft_check` (`tool-names.ts:23–30`, aufgelöst über `canonicalToolName` in `callTool`, `tools.ts:223`) | `graph_find_code`, `graph_check_freshness`, keine Aliase (`internal/mcptools/tools.go:136`, `:152`) | **Abweichung:** loomux hat keine Altnamen zu bedienen; kein Client hat je einen `graft_*`-Namen gegen loomux gerufen. Das Präfix folgt `brain_*`: Familie, Unterstrich, Verb. | freigegeben 2026-09-19 |
| `scope` | fehlt, ein Repo je Server; `graft_check_freshness` hat gar keine Parameter (`tools.ts:79`) | Pflichtfeld beider Werkzeuge (`mcptools/tools.go:142`, `:156`); leer ⇒ `isError` „… requires a scope" (`serve/graph/tools.go:52`, `:121`) | **Zusatz (Spec §3.3):** `loomux serve` bedient alle registrierten Bereiche, ein Graph gehört zu genau einem Repo. `scope = "all"` wäre Föderation und ist nicht in G3; es ist ein unbekannter Scope wie jeder andere (`TestAllIsNotAScope`). | freigegeben 2026-09-19 |
| `limit`, Typ | `number` (`tools.ts:51`) | `integer` (`mcptools/tools.go:144`) | **Abweichung:** Alle Schemas in loomux deklarieren Zählwerte als `integer` (vgl. `n` bei `brain_*`, `mcptools/tools.go:89`). Ein Bruch kommt als JSON-Zahl trotzdem an; `limit` schneidet ihn mit `int(n)` ab. | freigegeben 2026-09-19 |
| `limit` ≤ 0 | übernommen wie gegeben: `typeof args.limit === 'number' ? args.limit : 5` (`tools.ts:163`, `:257`) reicht `0` und `-2` durch | fällt auf 5 (`serve/graph/tools.go:205–210`: `n >= 1`, sonst `mcpLimit`) | **Abweichung:** `ask.Run` behandelt `Limit <= 0` als „kein Wert" und nimmt seinen eigenen Standard 8 (`defaultLimit` in `internal/code/ask/ask.go`), den der Kommandozeile — über MCP wäre das ein anderer Standard als der, den das Schema verspricht („default 5"). Die Grenze liegt bei 1: `limit: 1` kommt als 1 an (`TestALimitOfOneIsTakenAsGiven`), `0` und `-2` als 5 (`TestFindCodePassesLimitFullAndIn`). | freigegeben 2026-09-19 |
| `limit` keine Zahl | fällt auf 5 (dieselbe Zeile, `tools.ts:163`, `:257`) | fällt auf 5 (`serve/graph/tools.go:206`, die Typprüfung `.(float64)`) | **gleich** (`"seven"` in `TestFindCodePassesLimitFullAndIn`). | freigegeben 2026-09-19 |
| Standardwerte | `limit` 5, Quelltext immer eingefügt (`source: true`), `full` wählt den ganzen Span (`tools.ts:257–260`) | `Limit: 5`, `Source: true`, `Full` aus dem Argument (`serve/graph/tools.go:62–64`) | **gleich** (`TestFindCodeReachesTheAreaRootWithGraftsDefaults`). Die Kommandozeile bleibt bei 8, auch das Grafts Wert (`cli.ts:595`). | freigegeben 2026-09-19 |
| Hinweis vor Text | `callTool`: `${note}\n${res.text}` (`tools.ts:240`) | `withNotes` (`serve/graph/tools.go:164–169`), zusätzlich als Fortschrittsmeldung (`report`, `:233`) | **gleich auf dem lokalen Kanal.** Auf dem Cloud-Kanal geht kein Hinweis hinaus (`:66–72`), siehe §4 „Auffrisch-Hinweise auf dem Cloud-Kanal". Ein Auffrisch-Hinweis erklärt die Antwort, also steht er vorn; `serve/brain` hängt Befunde hinten an, weil ein Befund die Antwort entwertet. Ohne Hinweis ist der Text der Bericht allein, ohne führenden Zeilenumbruch (`TestAnAnswerWithoutNotesIsTheReportAlone`). | freigegeben 2026-09-19 |
| Kein Erstbau | `ensureFreshGraph` kehrt ohne Bau zurück, wenn `wiring.json` fehlt und keine Worktree-Saat greift (`refresh.ts:156–172`, die Begründung Z. 163–167) | `query.Ask` gibt `ErrNoGraph` vor jedem Auffrischen (`internal/code/query/ask.go:41–43`); MCP antwortet mit `isError`, `loomux graph ask` exitet 1 | **gleich**, für MCP **und** CLI. Für die CLI ist das eine **Berichtigung an G2**: `graph ask` baute bisher einen ersten Graphen, wenn keiner da war. Fehlt nur der Frische-Record, bleibt es bei G2: unbekannt, also neu bauen (Spec §3.1). | freigegeben 2026-09-19 |
| `check_freshness` frischt nicht auf | `NO_REFRESH_TOOLS` (`tools.ts:201–203`, geprüft in `callTool` Z. 232) | `checkFreshness` ruft nur `query.Check` (`serve/graph/tools.go:86–110`) | **gleich:** ein Werkzeug, das vorher auffrischt, meldete über einen Graphen, den es eben repariert hat, also immer OK. | freigegeben 2026-09-19 |
| Drift und fehlender Graph | Text, `isError: false` (`tools.ts:275`). Fehlt der Graph, lässt Graft den Graphbericht einfach weg (`if (!g.missing) parts.push(…)`, `tools.ts:269–276`); das „NO GRAPH" in `check.ts:156–158` gehört zum Bericht über das `.context/`-Manifest, nicht zum Graphen | Text ohne `isError`; `isError` nur bei einem Lesefehler (`serve/graph/tools.go:92–95`). Ein fehlender Graph ergibt den Text „NO GRAPH … Run `loomux graph build` first." (`CheckReport` in `query/check.go`) | **gleich** beim `isError`; **Abweichung** im Text: loomux hat nur den Graphbericht und sagt deshalb ausdrücklich, dass keiner da ist, statt zu schweigen (Spec §3.2; `TestCheckFreshnessRunsWithoutARefreshAndDriftIsNoError`, `TestCheckFreshnessOnAMissingGraphIsText`). | freigegeben 2026-09-19 |
| `.context/`-Bericht in `check_freshness` | ja, vor dem Graphbericht (`formatCheckReport`, `tools.ts:273`) | entfällt | **Entfällt:** loomux hat keine `.context/`-Schicht; es bleibt der Graphbericht. | freigegeben 2026-09-19 |
| Paniken | `callTool` wirft nie (`tools.ts:3`, `catch` Z. 241–243) | `guarded` (`serve/graph/tools.go:180–192`) macht aus einer Panik `isError`; lokal mit der Meldung, auf dem Cloud-Kanal mit einem festen Text | **gleich auf dem lokalen Kanal** (`TestAPanicBecomesAnErrorResult`). Auf dem Cloud-Kanal **Abweichung**, siehe §4 „Panik auf dem Cloud-Kanal". | freigegeben 2026-09-19 |
| Worktree-Saat | `seedUnderLock` (`refresh.ts:124`, gerufen Z. 161) kopiert den Graphen des Haupt-Checkouts in einen Worktree | nein | **Offen (Spec §8):** Die Registry kennt den Haupt-Checkout; ein Graph eines verknüpften Worktrees ist über `scope` nicht erreichbar. Später. | freigegeben 2026-09-19 |
| `local_only` | — | Ein `local_only`-Bereich existiert auf dem Cloud-Kanal nicht: `privacy.VisibleAreas` löst ihn nicht auf, die Antwort ist „unknown scope" wie bei `brain_*` (`resolve`, `serve/graph/tools.go:120–133`) | **Zusatz (Spec §3.4):** vor jedem Zugriff auf den Graphen, weil schon die Abfrage eine Preisgabe wäre (`TestALocalOnlyAreaDoesNotExistOnTheCloudChannel`, `TestALocalOnlyAreaIsVisibleOnTheLocalChannel`). Ein kaputtes Register bleibt ein Registerfehler und wird nicht als „unknown scope" ausgegeben (`TestABrokenRegistryIsNotCalledAnUnknownScope`); auf dem Cloud-Kanal wird er zum festen Text, siehe §4. | freigegeben 2026-09-19 |
| `NeverGlobs` in `find_code` | — | `Keep` in `ask.Options`, vor dem Scoring (`internal/code/ask/ask.go:91`, `admission` `:214`); der Index wird über den `Path` des Graphknotens gefiltert (`ask.go:101`; `lexicon.(*Index).Where` nimmt ein Prädikat über die Id); der Handler reicht `privacy.IsReadable` des Manifests (`readable`, `serve/graph/tools.go:156–158`) | **Zusatz (Spec §3.4):** ein Filter nach dem Ranking ließe den Knoten mitranken, Rangmasse sammeln und einen Platz des `limit` belegen. Auf beiden Kanälen. Ein Datenschutzpfad wird **nie** aus einer Id geschnitten (Korrekturrunde 1, §3). | freigegeben 2026-09-19 |
| `NeverGlobs` in `check_freshness` | — | `Drift.Only` lässt die Pfade weg und zählt sie in `Hidden` (`internal/code/query/check.go:111`). Der Pfad ist der, den `Check` aus dem Knoten gelesen hat (`record` `:89`, `pathFor` `:99`; ein nicht exportiertes Feld, das JSON bleibt gleich); eine Id ohne gemerkten Pfad gilt als verborgen. `OK` bleibt, was es war | **Zusatz (Spec §3.4):** Drift in einer verborgenen Datei bleibt Drift; OK wäre eine Lüge über den Graphen. | freigegeben 2026-09-19 |
| Zahl der verborgenen Einträge | — | lokal: „(N more under the area's never globs)" (`check.go:158–159`); Cloud-Kanal: keine Zahl, keine Zeile (`serve/graph/tools.go:97–106` setzt `Hidden = 0`) | **Entscheidung des Nutzers vom 2026-09-19:** schon die Zahl sagte einem entfernten Modell, dass sich unter den `never`-Globs etwas geändert hat. Der Bericht bleibt DRIFT und wird nie OK (`TestTheCloudChannelHearsNoCountOfHiddenDrift`, `TestCheckReportWithoutHiddenDriftSaysNothingOfIt`). | freigegeben 2026-09-19 |

## 2. Mutationsrunde

**Geschichte.** §2 bis §4 zeichnen die Runden 1 bis 4 und die beiden
Korrekturrunden auf, wie sie vor dem Umsetzen auf `master` liefen; Zeilen und
Commits dort gelten für jenen Stand. Die Überlebenden, die zur Freigabe stehen,
sind in Runde 5 (§5) am Code von PR #8 neu gerechnet.

Befehl:
```sh
go run ./cmd/loomux dev mutants internal/code/query internal/serve/graph
```

Runde 3 fährt dazu `internal/code/ask` und `internal/code/lexicon` mit, weil
die Korrekturrunde beide ändert.

Die Runde wird **je Paket** gefahren, und `dev mutants` startet zu jeder
Mutante nur `go test ./<paket>/`. Ein Test in einem anderen Paket tötet hier
nichts.

**Runde 1 (2026-09-19, auf `007c5ec`):**

| Paket | Mutanten | nicht kompiliert | getötet | überlebt |
|---|---|---|---|---|
| `internal/code/query` | 125 | 17 | 63 | **45** |
| `internal/serve/graph` | 58 | 10 | 45 | **3** |

Die 45 in `query` hatten eine gemeinsame Ursache: Spec §6 sagt, die Tests der
alten `cli/graph.go`-Logik „ziehen mit um". Sie sind nicht umgezogen. `Check`
hatte in `query` keinen einzigen Test, `Build`s Rückgabewerte und die Zählung
`NoSymbol` prüfte niemand, und `AskReport` wurde nie Zeile für Zeile verglichen.
Die Tests in `internal/cli` erreichen diesen Code, aber nach der Regel aus G2b
Runde 3 ist „von einem Test anderswo erreicht" keine Äquivalenz, sondern ein
fehlender Test in diesem Paket. 43 der 45 waren fehlende Tests, dazu alle drei
in `serve/graph`:

| Ort (Runde 1) | Familie | Mutanten | Fehlte | Getötet durch |
|---|---|---|---|---|
| `query/check.go:30`, `:31`, `:34`, `:45`, `:50`, `:66`, `:71`, `:78` | a1, a3, a4 | 21 | jeder Test von `Check` im eigenen Paket | `TestCheckIsOKRightAfterABuild`, `TestCheckWithoutAGraphIsMissingAndNoError`, `TestCheckOnASchemaOneGraphIsOutdated`, `TestCheckReportsACorruptGraphAsAnError`, `TestCheckCallsAGraphFromAnotherExtractorForeignBeforeDiffing`, `TestCheckReportsAFailedReExtraction`, `TestCheckNamesAddedRemovedAndChangedSymbols` |
| `query/check.go:140` | a1, a3 | 2 | ein Bericht mit Drift und `Hidden == 0` | `TestCheckReportWithoutHiddenDriftSaysNothingOfIt` — dieser Test hält zugleich die Nutzerentscheidung zum Cloud-Kanal fest |
| `query/ask.go:63` | a1 | 1 | `Source` aus ⇒ kein Code | `TestAskWithoutSourceCarriesNoCode` |
| `query/ask.go:78`, `:81` | a1, a3, a4 | 8 | der genaue Wortlaut von `AskReport` mit und ohne Signatur und Code | `TestAskReportPrintsSignatureAndCodeOnlyWhenThere` |
| `query/build.go:43`, `:55`, `:87` | a1, a3, a4 | 6 | Graph und Statistik, die `Build` zurückgibt; `Build` auf einer fehlenden Wurzel | `TestBuildReturnsTheGraphAndWhatItLearned`, `TestBuildFailsWhenTheRootDoesNotExist` |
| `query/build.go:77` | a1 | 1 | Der Lesefehler selbst: unter der Mutante lief `golang.File` auf leerem Inhalt und meldete einen Parse-Fehler, den der alte Test ebenso als Fehler annahm | `TestExtractReportsTheReadFailureNotAParseError` (`errors.As` auf `*fs.PathError`) |
| `query/build.go:84` | a1 | 1 | eine Datei, die nicht parst | `TestExtractRefusesAFileItCannotParse` |
| `query/build.go:109`, `:113` | a1, a2 | 3 | `goModPaths` auf einer fehlenden Wurzel, ein Modul in einem Unterverzeichnis, eine Wurzel namens `testdata` | `TestGoModPathsOnAMissingRootIsEmpty`, `TestGoModPathsFindsANestedModule`, `TestGoModPathsWalksARootThatWouldBeSkippedBelowIt` |
| `serve/graph/tools.go:104` | a1 | 1 | Unter der Mutante kam ein kaputtes Register als „unknown scope" heraus; der alte Test prüfte nur, dass Text da ist | `TestABrokenRegistryIsNotCalledAnUnknownScope` |
| `serve/graph/tools.go:124` | a1 | 1 | eine Antwort ohne Hinweis, Wort für Wort | `TestAnAnswerWithoutNotesIsTheReportAlone` |
| `serve/graph/tools.go:154` | a3 | 1 | `limit: 1` — genau die Grenze der Verfügung „≤ 0 fällt auf 5" | `TestALimitOfOneIsTakenAsGiven` |

Commits: `2eb4181` `test(query): …`, `05fb11c` `test(graph): …`.

**Runde 2 (2026-09-19, auf `05fb11c`):**

| Paket | Mutanten | nicht kompiliert | getötet | überlebt |
|---|---|---|---|---|
| `internal/code/query` | 125 | 17 | 106 | **2** |
| `internal/serve/graph` | 58 | 10 | 48 | **0** |

Die Verfügungen zu den zwei Überlebenden, wie Runde 2 sie gab. Die zweite ist in Korrekturrunde 1 zurückgezogen (§3):

| Ort | Familie | Mutation | Ausgang | Verfügung | Freigabe |
|---|---|---|---|---|---|
| `query/build.go:122` | a1 | `if err == nil {` $\to$ `if true {` | Überlebt | **Äquivalent, weil unerreichbar:** `filepath.Rel(root, p)` in `goModPaths` scheitert nur, wenn `p` nicht relativ zu `root` ausgedrückt werden kann. `WalkDir` liefert jedes `p` als `filepath.Join(root, …)`, also mit `root` als lexikalischem Präfix; `Rel` gibt dafür nie einen Fehler. Derselbe Schluss trägt die Ausnahme an `sourceset.go:87` und `cases/runner.go:52`. | abgelöst durch §5 |
| `query/check.go:109` | a3 | `if i := strings.Index(id, "#"); i >= 0 {` $\to$ `... i > 0 {` | Überlebt | ~~Äquivalent~~ — **zurückgezogen in Korrekturrunde 1.** Die Begründung „keine ID beginnt mit `#`" ist falsch: eine Dateiknoten-Id ist der Pfad ohne Zusatz (`extract/golang/extract.go:145`), und `sourceset` nimmt jede `*.go`-Datei, also auch `#gen.go`. Der Befund war ein Datenschutzfehler, siehe §3. | zurückgezogen |

**Nicht kompiliert (27), keine Mutanten.** Das Werkzeug erzeugt für ein `if`
mit Init-Anweisung Formen, die der Compiler ablehnt (die Variable wäre
deklariert und unbenutzt, oder `!( … ; … )` ist kein Ausdruck). Betroffen sind
unter anderem `query/ask.go:41` — die Wache „kein Erstbau" — und
`serve/graph/tools.go:135` (`recover`), `:154` (`limit`), `:170`
(`json.Unmarshal`). Auf diesen Zeilen hat die Runde also **nicht alle**
Mutanten geprüft — ein kompilierter a3-Mutant auf `serve/graph/tools.go:154`
(`n >= 1` $\to$ `n > 1`) ist gelaufen und getötet. Die Tests stehen im eigenen
Paket:
`TestAskWithoutAGraphRefusesAndBuildsNothing`, `TestAPanicBecomesAnErrorResult`,
`TestFindCodePassesLimitFullAndIn` mit `TestALimitOfOneIsTakenAsGiven`,
`TestArgumentsThatAreNotAnObjectEndEmpty`.

## 3. Korrekturrunde 1 (2026-09-19): Datenschutzpfade nicht mehr aus Ids

**Befund.** Das Review hat die Äquivalenz von `query/check.go:109` widerlegt, und
die Nachrechnung gegen den Code bestätigt es. `pathOf` schnitt eine Knoten-Id
am **ersten** `#`. Eine Dateiknoten-Id ist aber der Pfad selbst, ohne Zusatz
(`extract/golang/extract.go:145`), und `sourceset` nimmt jede Datei auf `.go`.
Eine Datei `#gen.go` im Wurzelverzeichnis hat also die Id `#gen.go`, ihr Symbol
die Id `#gen.go#Gen`, und `pathOf` las für beide `""`. Eine Datei unter `dir#1/`
las sich als `dir`. Keiner dieser Pfade fällt unter einen `never`-Glob wie `#*`
oder `dir#1/**`. Solche Dateien kamen deshalb an zwei Stellen am Filter vorbei:

- **`graph_find_code`:** `lexicon.(*Index).Where` urteilte über `pathOf(d.ID)`.
  Der Walk urteilte schon über den Knotenpfad, der Index aber nicht, und ein
  lexikalischer Treffer wird auch ohne den Walk Kandidat. Beleg vor der
  Korrektur: `TestRunKeepJudgesTheNodesPathAndNeverAPathCutFromTheID` fand
  `#gen.go`, `#gen.go#Generate` und `dir#1/a.go#Generate`. Dasselbe galt für
  `In`: `--in dir` ließ `dir#1/a.go` zu (`TestRunInJudgesTheNodesPathToo`).
- **`graph_check_freshness`:** `Drift.Only` urteilte über `pathOf(id)`. Beleg
  vor der Korrektur: `TestOnlyAfterCheckJudgesTheNodesPathAndNeverAPathCutFromTheID`
  meldete sechs Ids unter den `never`-Globs mit `Hidden = 0`.

Ende zu Ende über MCP zeigte
`TestANeverGlobHidesAFileWhoseNameStartsWithAHashEndToEnd` dasselbe: ein
registrierter Bereich mit `never = ["#*"]`, echte `query.Ask`/`query.Check`.
Vor der Korrektur nannte `graph_find_code` den Code aus `#gen.go`, und
`graph_check_freshness` listete `changed #gen.go`.

**Regel (Entscheidung des Koordinators, bindend):** Ein Datenschutzpfad wird
nie aus einer Id geparst.

| Ort | Änderung | Freigabe |
|---|---|---|
| `lexicon.(*Index).Where` (`internal/code/lexicon/index.go:124`) | nimmt ein Prädikat `func(model.NodeID) bool` über die Dokument-Id statt eines Pfads aus `pathOf` | freigegeben 2026-09-19 |
| `ask.Run` (`internal/code/ask/ask.go:95–102`) | baut `byID` vor dem Filtern und reicht `func(id) bool { return admit(byID[id].Path) }`. Ein Dokument ohne Knoten hat dort den Pfad `""`. Unter `In` fällt es damit weg, unter `Keep` allein kommt es durch und prägt DF und `AvgBodyLen` mit. Die Schleife danach verwirft es in jedem Fall, weil es keinen Knoten hat, also nie als Treffer. Das ist der Fall „neuer Graph, alte Beiakte" aus Spec §5, und er wird schlechter, nie falsch. | freigegeben 2026-09-19 |
| `query.Check`/`Drift` (`internal/code/query/check.go:30`, `:89`, `:100`) | `Check` merkt sich zu jeder gemeldeten Id den Pfad des Knotens, in einem **nicht exportierten** Feld `paths`, damit das JSON unverändert bleibt (`TestTheRecordedPathsStayOutOfTheJSON`). `Drift.Only` urteilt über diesen Pfad. ~~Nur für eine Id ohne gemerkten Pfad fällt es auf `pathOf` zurück.~~ **Überholt durch Korrekturrunde 2 (§4):** eine Id ohne gemerkten Pfad gilt als verborgen, `pathOf` ist aus `query` entfernt. | überholt, siehe §4 |
| `lexicon.(*Index).Filter` (`index.go:106`) | ~~behält sein Verhalten und seine Präfixregel über `pathOf(id)`. Bleibt stehen.~~ **Überholt durch Korrekturrunde 2 (§4):** `Filter` und `pathOf` sind aus `lexicon` entfernt. | überholt, siehe §4 |

Commit: `4780c81` `fix(graph): judge never globs by the node's path, not one cut from its id`.
Derselbe Commit schärft `TestABrokenRegistryIsNotCalledAnUnknownScope`: der Text
muss `registry.toml` und „not valid TOML" nennen.

**G2b wieder geöffnet.** Die Freigabe der G2b-Zeile
`lexicon/index.go:163` (a3, `i >= 0` $\to$ `i > 0` in `pathOf`) vom
2026-09-18 ruht auf derselben falschen Prämisse „keine Id beginnt mit `#`".
Diese Zeile ist damit **wieder offen**. G3 hat Folgendes geändert: `pathOf` in
`lexicon` dient nur noch `Filter`. `Where` und damit `ask.Run` urteilen über den
Knotenpfad. Die Mutante lebt weiter (jetzt `index.go:177`, siehe Runde 3), ist
aber nicht mehr äquivalent. **Korrekturrunde 2 (§4):** `pathOf` und `Filter` sind aus
`lexicon` entfernt; die Mutante gibt es nicht mehr, und die G2b-Zeile hat keinen Gegenstand mehr.

**Runde 3 (2026-09-19, auf `4780c81`), vier Pakete:**

```sh
go run ./cmd/loomux dev mutants internal/code/query internal/serve/graph internal/code/ask internal/code/lexicon
```

| Paket | Mutanten | nicht kompiliert | getötet | überlebt |
|---|---|---|---|---|
| `internal/code/query` | 132 | 22 | 108 | **2** |
| `internal/serve/graph` | 58 | 10 | 48 | **0** |
| `internal/code/ask` | 147 | 24 | 111 | **12** |
| `internal/code/lexicon` | 55 | 11 | 41 | **3** |

Die Überlebenden, jeder gegen den Code dieses Stands nachgerechnet. Zehn in
`ask` und zwei in `lexicon` sind dieselben wie in G2b (dort freigegeben), nur
mit verschobenen Zeilen. Der G2b-Überlebende `ask.go:86` (`in != ""`) existiert
nicht mehr, weil `admission` die Wache übernommen hat.

| Ort | Familie | Mutation | Verfügung | Freigabe |
|---|---|---|---|---|
| `query/build.go:122` | a1 | `if err == nil {` $\to$ `if true {` | **Äquivalent, weil unerreichbar:** siehe Runde 2; unverändert. | abgelöst durch §5 |
| `query/check.go:136` | a3 | `pathOf`: `i >= 0` $\to$ `i > 0` | **Entfällt in Runde 4:** `pathOf` ist aus `query` entfernt (§4). Die Verfügung dieser Runde lautete: **Nicht äquivalent, bewusst nicht getötet.** `pathOf` dient nur noch der Rückfallebene für einen von Hand gebauten `Drift`. Bei einer Id, die mit `#` beginnt, liefert das Original `""` und die Mutante die ganze Id. Die Mutante ist dort sogar die richtigere Lesart. Ein Test, der sie tötet, müsste das bekannte falsche Verhalten der Rückfallebene festschreiben. `Check` erreicht diesen Zweig für keine Id, die es selbst meldet. | abgelöst durch §5 |
| `lexicon/index.go:177` | a3 | `pathOf`: `i >= 0` $\to$ `i > 0` | **Entfällt in Runde 4:** `pathOf` und `Filter` sind aus `lexicon` entfernt (§4). Die Verfügung dieser Runde lautete: **Nicht äquivalent, bewusst nicht getötet**, aus demselben Grund. `Filter("#gen.go")` behält unter der Mutante den Dateiknoten `#gen.go`, das Original verwirft ihn. Ein tötender Test schriebe die `#`-Grenze von `Filter` fest. Ersetzt die G2b-Zeile `index.go:163`, die oben wieder geöffnet ist. | abgelöst durch §5 |
| `lexicon/index.go:195` | a1 | `if err != nil {` $\to$ `if false {` | **Äquivalent, weil unerreichbar:** die `MarshalIndent`-Prüfung in `Write`, G2b `index.go:181`. | freigegeben in G2b |
| `lexicon/index.go:233` | a3 | `docs[i].ID < docs[j].ID` $\to$ `<=` | **Äquivalent:** eindeutige Ids, G2b `index.go:219`. | freigegeben in G2b |
| `ask/ask.go:125` | a3 | `score > maxLex` $\to$ `>=` | **Äquivalent:** weist bei Gleichheit denselben Wert zu, G2b `ask.go:116`. | freigegeben in G2b |
| `ask/ask.go:137` | a3 | `s.Score >= rescueFloor` $\to$ `>` | **Äquivalent bei 25 Iterationen**, G2b `ask.go:128`. | freigegeben in G2b |
| `ask/ask.go:166` | a3 | `hits[i].Score > hits[j].Score` $\to$ `>=` | **Äquivalent:** nur bei ungleichen Scores erreicht, G2b `ask.go:157`. | freigegeben in G2b |
| `ask/ask.go:168` | a3 | `hits[i].ID < hits[j].ID` $\to$ `<=` | **Äquivalent:** eindeutige Ids, G2b `ask.go:159`. | freigegeben in G2b |
| `ask/ask.go:175` | a3 | `len(hits) > limit` $\to$ `>=` | **Äquivalent:** bei Gleichheit ein Schnitt ohne Wirkung, G2b `ask.go:166`. | freigegeben in G2b |
| `ask/ask.go:200` | a1 | `len(seed) == 0` $\to$ `false` | **Äquivalent:** `pagerank.Rank` gibt bei leerer Saat `nil`, G2b `ask.go:191`. | freigegeben in G2b |
| `ask/refresh.go:94` | a3 | `< lockStale` $\to$ `<=` | **Äquivalent bis auf die Uhrauflösung**, G2b. | freigegeben in G2b |
| `ask/refresh.go:122` | a3 | `< lockStale` $\to$ `<=` | wie oben, G2b. | freigegeben in G2b |
| `ask/source.go:30` | a1 | `if !ok {` $\to$ `if false {` | **Äquivalent**, G2b. | freigegeben in G2b |
| `ask/source.go:34` | a1 | `if !ok {` $\to$ `if false {` | **Äquivalent**, G2b. | freigegeben in G2b |
| `ask/source.go:48` | a3 | `from < 1` $\to$ `<=` | **Äquivalent:** bei `from == 1` dieselbe Zuweisung, G2b. | freigegeben in G2b |
| `ask/source.go:52` | a3 | `to > len(lines)` $\to$ `>=` | **Äquivalent:** bei Gleichheit dieselbe Zuweisung, G2b. | freigegeben in G2b |

Die G2b-Verfügungen sind am Code dieses Stands nachgerechnet: `ask.go` hat
sich nur oberhalb der Zeilen verschoben (`byID` steht jetzt vor dem Filter),
die Ausdrücke sind dieselben.

## 4. Korrekturrunde 2 (2026-09-19): Abschlussreview

Das Abschlussreview über den ganzen Branch fand zwei weitere Preisgaben an den
Cloud-Kanal und zwei Stellen, an denen noch ein Pfad aus einer Id geschnitten
wurde. Die Entscheidungen des Koordinators sind bindend; jede Zeile ist vom
Nutzer freigegeben (2026-09-19).

| Punkt | Änderung | Tests | Freigabe |
|---|---|---|---|
| Auffrisch-Hinweise auf dem Cloud-Kanal | `graph_find_code` schickt auf dem Cloud-Kanal **keine** Hinweise, weder im Text noch als Fortschritt (`serve/graph/tools.go:66–72`). Grund: `ask.EnsureFresh` zählt auch Dateien unter den `never`-Globs („%d files moved, rebuilding the graph"), und der Hinweis eines gescheiterten Neubaus zitiert den Parse-Fehler einer verborgenen Datei. Der lokale Kanal ist unverändert. | `TestTheCloudChannelHearsNoRefreshNote`; lokal weiter `TestNotesComeBeforeTheAnswerAndAsProgress`, `TestWithoutAProgressTokenTheNotesLapseButStayInTheText` | freigegeben 2026-09-19 |
| Fehlertexte auf dem Cloud-Kanal | Jeder Fehler aus `Ask` oder `Check` wird auf dem Cloud-Kanal zu „the graph could not be read on this channel; ask on the local channel for details" (`errorText`, `serve/graph/tools.go:139`). Ausnahme: `query.ErrNoGraph` behält seinen eigenen Text, und zwar ohne das, worin er eingewickelt kam. Die Argument-Verweigerungen und „unknown scope" tragen nur, was der Aufrufer selbst gab, und bleiben; ein Registerfehler dagegen nicht, siehe die nächste Zeile. Lokal unverändert. | `TestACloudFailureOfFindCodeIsTheFixedText`, `TestACloudFailureOfCheckFreshnessIsTheFixedText`, `TestNoGraphKeepsItsOwnTextOnTheCloudChannel`, `TestALocalFailureKeepsItsDetail` | freigegeben 2026-09-19 |
| Registerfehler auf dem Cloud-Kanal | Nachtrag vom Code-Review (2026-09-19): `resolve` gab die Fehler von `privacy.VisibleAreas` unverändert weiter; ein kaputtes `registry.toml` nannte dem Cloud-Kanal den absoluten Pfad auf dieser Maschine samt Parse-Fehler. `resolve` fragt die sichtbaren Bereiche jetzt unter „all" ab, sodass jeder verbleibende Fehler von `VisibleAreas` ein Konfigurationsfehler ist und durch `errorText` geht; „unknown scope" kommt aus `privacy.Single` über dieselben sichtbaren Bereiche. Lokal unverändert. Für `brain_*` gilt das nicht, dort bleibt der Registerfehler wörtlich. | `TestABrokenRegistryIsTheFixedTextOnTheCloudChannel`; lokal weiter `TestABrokenRegistryIsNotCalledAnUnknownScope`; „unknown scope" weiter `TestALocalOnlyAreaDoesNotExistOnTheCloudChannel`, `TestAllIsNotAScope` | freigegeben 2026-09-19 |
| Panik auf dem Cloud-Kanal | Nachtrag vom Code-Review (2026-09-19): `guarded` gab den Wert einer Panik auf jedem Kanal als „internal error: %v" weiter. Der Wert ist, was der fehlschlagende Code hielt, und kann ein Pfad oder eine Zeile unter den `never`-Globs sein. Auf dem Cloud-Kanal lautet der Fehler jetzt fest „internal error; ask on the local channel for details"; lokal bleibt der Wert. Graft gibt die Meldung der Ausnahme wörtlich zurück (`tools.ts`, `catch` in `callTool`); Graft kennt keinen Cloud-Kanal. | `TestAPanicTellsTheCloudChannelNothingOfItsValue`; lokal weiter `TestAPanicBecomesAnErrorResult` | freigegeben 2026-09-19 |
| `Drift.pathFor` schließt | Eine Id ohne gemerkten Pfad gilt als verborgen: `Only` verwirft sie und zählt sie in `Hidden`, ohne `keep` zu fragen; sie wird nie geparst (`query/check.go:99`, `:111`). `pathOf` ist aus `query` entfernt. Die zwei `serve/graph`-Tests, die einen `Drift` von Hand bauten, holen ihn jetzt aus einem echten `query.Check` (`drifted` in `tools_test.go`); ein `export_test.go` in `query` hätte `serve/graph` nicht erreicht, weil es nur in `query`s eigenes Testbinary kompiliert wird. | `TestAnIDWithNoRecordedPathIsHiddenAndNeverParsed` (ersetzt `TestADriftBuiltByHandFallsBackToTheIDsPath`), `TestCheckRecordsThePathOfEveryIDItReports`, `TestCheckFreshnessRunsWithoutARefreshAndDriftIsNoError`, `TestTheCloudChannelHearsNoCountOfHiddenDrift` | freigegeben 2026-09-19 |
| `lexicon.(*Index).Filter` entfernt | `Filter` und `lexicon`s `pathOf` sind gelöscht: kein Aufrufer im Produktionscode, und die Präfixregel über `pathOf(id)` war die bekannt falsche Lesart. Die Filter-Tests, die etwas Echtes prüften, laufen jetzt über `Where` mit einem Prädikat aus den Knotenpfaden (`under` in `index_test.go`). Gelöscht, weil sie nur `Filter`s eigene Präfixbehandlung prüften (die Tabelle in `TestNormalizePrefix` deckt die Normalisierung): `TestFilterOnAnEmptyPrefixIsTheWholeIndex`, `TestFilterNormalizesTheShellCompletedPrefix`, `TestFilterNormalizesAWindowsShapedSubtree`, `TestFilterOnASlashAloneIsTheWholeIndex`, `TestFilterStaysSegmentAwareAfterNormalizing`. | `TestWhereOnASubtreeRecomputesDocumentFrequencyOverTheRemainder`, `TestUnderPrefixIsSegmentAware`, `TestWhereThatAdmitsNothingIsEmpty` | freigegeben 2026-09-19 |

Dazu, ohne eigene Zeile: `graph_find_code` prüft `scope` und `query`, bevor es
den Scope auflöst (Spec §5 Schritt 1; `TestFindCodeChecksTheQueryBeforeTheScope`,
`TestFindCodeWithNeitherArgumentAsksForTheScope`); der Ende-zu-Ende-Test in
`internal/serve` fragt den Cloud-Listener zuerst nach dem offenen Bereich
`project/code`; Kommentare in `ask.go` (`walk`), `cli/graph.go`
(Ausnahme an `graphCheck`), `model/graph.go` und am Cloud-Zweig von
`checkFreshness` sind am Code nachgerechnet. Die Zeilenangaben in §1 sind auf
den Code von PR #8 nachgezogen.

Commits: `a8596ff` `fix(query): …`, `6738037` `refactor(lexicon): …`,
`510b0da` `docs(ask): …`, `c42db5f` `fix(serve): …`, `bbea9e5` `test(query,graph): …`.

**Runde 4 (2026-09-19, auf `c42db5f`), vier Pakete:**

```sh
go run ./cmd/loomux dev mutants internal/code/query internal/serve/graph internal/code/ask internal/code/lexicon
```

| Paket | Mutanten | nicht kompiliert | getötet | überlebt |
|---|---|---|---|---|
| `internal/code/query` | 127 | 19 | 106 | **2** |
| `internal/serve/graph` | 67 | 10 | 56 | **1** |
| `internal/code/ask` | 147 | 24 | 111 | **12** |
| `internal/code/lexicon` | 47 | 8 | 37 | **2** |

Zwei Überlebende waren neu und fehlende Tests; beide sind in `bbea9e5`
getötet. Nachgeprüft mit `dev mutants --only check.go internal/code/query`
(41 Mutanten, 7 nicht kompiliert, 0 überlebt) und
`dev mutants --only tools.go internal/serve/graph` (67, 10, 0 überlebt):

| Ort (Runde 4) | Familie | Mutation | Fehlte | Getötet durch |
|---|---|---|---|---|
| `query/check.go:90` | a1 | `if d.paths == nil {` $\to$ `if true {` | Unter der Mutante legt jedes `record` die Karte neu an; nur die zuletzt gemeldete Id behält ihren Pfad, und das schließende `Only` verbirgt alle anderen. Kein Test ließ mehrere Ids unter einem `keep` durch, das alles zulässt. | `TestCheckRecordsThePathOfEveryIDItReports` |
| `serve/graph/tools.go:52` | a1 | `if scope == "" {` $\to$ `if false {` | Ohne diese Wache fängt `resolve` einen leeren Scope mit demselben Text ab — außer wenn auch `query` fehlt: dann käme „requires a query". | `TestFindCodeWithNeitherArgumentAsksForTheScope` |

Die übrigen Überlebenden, gegen den Code von `bbea9e5` nachgerechnet:

| Ort | Familie | Mutation | Verfügung | Freigabe |
|---|---|---|---|---|
| `query/build.go:122` | a1 | `if err == nil {` $\to$ `if true {` | **Äquivalent, weil unerreichbar:** siehe Runde 2; `build.go` ist unverändert. | abgelöst durch §5 |
| `lexicon/index.go:165` | a1 | `if err != nil {` $\to$ `if false {` | **Äquivalent, weil unerreichbar:** die `MarshalIndent`-Prüfung in `Write`, G2b `index.go:181`, in Runde 3 `:195`; die Zeile ist nur durch das Löschen von `Filter` und `pathOf` gewandert. | freigegeben in G2b |
| `lexicon/index.go:203` | a3 | `docs[i].ID < docs[j].ID` $\to$ `<=` | **Äquivalent:** eindeutige Ids, G2b `index.go:219`, Runde 3 `:233`. | freigegeben in G2b |
| `ask/ask.go:125`, `:137`, `:166`, `:168`, `:175` | a3 | wie Runde 3 | **Äquivalent**, dieselben Ausdrücke auf denselben Zeilen wie in Runde 3. | freigegeben in G2b |
| `ask/ask.go:201` | a1 | `len(seed) == 0` $\to$ `false` | **Äquivalent:** `pagerank.Rank` gibt bei leerer Saat `nil`; Runde 3 `:200`, eine Zeile tiefer durch den berichtigten `walk`-Kommentar. | freigegeben in G2b |
| `ask/refresh.go:94`, `:122` | a3 | `< lockStale` $\to$ `<=` | **Äquivalent bis auf die Uhrauflösung**, unverändert. | freigegeben in G2b |
| `ask/source.go:30`, `:34`, `:48`, `:52` | a1, a3 | wie Runde 3 | **Äquivalent**, unverändert. | freigegeben in G2b |

Die Überlebenden `query/check.go:136` und `lexicon/index.go:177` aus Runde 3
gibt es nicht mehr: `pathOf` ist in beiden Paketen entfernt. Damit hat auch die
wieder geöffnete G2b-Zeile `lexicon/index.go:163` keinen Gegenstand mehr.

## 5. Runde 5 (2026-09-19, auf dem umgesetzten Branch): nach dem Umsetzen auf `master`

Die Runden 1 bis 4 liefen auf Code, der so nicht mehr im Branch steht:
`master` hatte zwischenzeitlich `ask/refresh.go` (`takeOver` ohne harte Links),
`lexicon/index.go` (`Usable`) und den Bau (Frische-Record als Hinweis statt
Fehler) geändert. Runde 5 rechnet deshalb alle vier Pakete neu.

```sh
go run ./cmd/loomux dev mutants internal/code/query internal/serve/graph internal/code/ask internal/code/lexicon
```

| Paket | Mutanten | nicht kompiliert | getötet | überlebt |
|---|---|---|---|---|
| `internal/code/query` | 127 | 19 | 106 | **2** |
| `internal/serve/graph` | 71 | 10 | 61 | **0** |
| `internal/code/ask` | 144 | 28 | 104 | **12** |
| `internal/code/lexicon` | 75 | 21 | 52 | **2** |

Ein Überlebender war neu und ein fehlender Test:

| Ort | Familie | Mutation | Fehlte | Getötet durch |
|---|---|---|---|---|
| `query/build.go:62` | a3 | `err != nil` $\to$ `err == nil` beim Schreiben des Frische-Records | Die aus `master` übernommene Regel „ein Bau steht, wenn nur der Record nicht geschrieben werden kann" hatte ihren Test nur in `internal/cli` (`TestGraphBuildWarnsButSucceedsWhenTheFingerprintCannotBeWritten`); im eigenen Paket prüfte niemand, dass `Build` genau dann einen Hinweis gibt. Unter der Mutante meldete jeder gelungene Bau „freshness record not written: <nil>". | `TestBuildAnnouncesOnlyARecordItCouldNotWrite`; nachgeprüft mit `dev mutants --only build.go internal/code/query` (57 Mutanten, 9 nicht kompiliert, nur `build.go:129` überlebt) |

Die übrigen Überlebenden, jeder am Code von PR #8 nachgerechnet. Die
Ausdrücke sind dieselben wie in Runde 4; nur die Zeilen haben sich durch den
Code aus `master` verschoben.

| Ort | Familie | Mutation | Verfügung | Freigabe |
|---|---|---|---|---|
| `query/build.go:129` | a1 | `if err == nil {` $\to$ `if true {` | **Äquivalent, weil unerreichbar:** `filepath.Rel(root, p)` in `goModPaths`; `WalkDir` liefert jedes `p` als `filepath.Join(root, …)`, `Rel` scheitert dafür nie (Runde 2, dort `:122`). | freigegeben 2026-09-19 |
| `ask/ask.go:125` | a3 | `score > maxLex` $\to$ `>=` | **Äquivalent:** bei Gleichheit dieselbe Zuweisung. | freigegeben in G2b |
| `ask/ask.go:137` | a3 | `s.Score >= rescueFloor` $\to$ `>` | **Äquivalent bei 25 Iterationen.** | freigegeben in G2b |
| `ask/ask.go:166` | a3 | `hits[i].Score > hits[j].Score` $\to$ `>=` | **Äquivalent:** nur bei ungleichen Scores erreicht. | freigegeben in G2b |
| `ask/ask.go:168` | a3 | `hits[i].ID < hits[j].ID` $\to$ `<=` | **Äquivalent:** eindeutige Ids. | freigegeben in G2b |
| `ask/ask.go:175` | a3 | `len(hits) > limit` $\to$ `>=` | **Äquivalent:** bei Gleichheit ein Schnitt ohne Wirkung. | freigegeben in G2b |
| `ask/ask.go:201` | a1 | `len(seed) == 0` $\to$ `false` | **Äquivalent:** `pagerank.Rank` gibt bei leerer Saat `nil`. | freigegeben in G2b |
| `ask/refresh.go:114` | a3 | `< lockStale` $\to$ `<=` in `lock` | **Äquivalent bis auf die Uhrauflösung:** der Unterschied liegt auf genau einer Nanosekunde an der Stundengrenze. `master` hat `lock` nicht geändert, nur verschoben (G2b `:94`). | freigegeben in G2b |
| `ask/refresh.go:156` | a3 | `< lockStale` $\to$ `<=` in `takeOver` | **Äquivalent bis auf die Uhrauflösung**, derselbe Vergleich. `master` hat in `takeOver` nur geändert, was **nach** dem Vergleich geschieht (Rückgabe per Link oder Umbenennen); der Ausdruck ist derselbe wie in G2b `:122`. | freigegeben 2026-09-19 |
| `ask/source.go:30`, `:34` | a1 | `if !ok {` $\to$ `if false {` | **Äquivalent**, unverändert seit G2b. | freigegeben in G2b |
| `ask/source.go:48`, `:52` | a3 | `from < 1` $\to$ `<=`, `to > len(lines)` $\to$ `>=` | **Äquivalent:** bei Gleichheit dieselbe Zuweisung, unverändert seit G2b. | freigegeben in G2b |
| `lexicon/index.go:165` | a1 | `if err != nil {` $\to$ `if false {` | **Äquivalent, weil unerreichbar:** die `MarshalIndent`-Prüfung in `Write`, mit `//coverage:exempt` und derselben Begründung. | freigegeben in G2b |
| `lexicon/index.go:241` | a3 | `docs[i].ID < docs[j].ID` $\to$ `<=` | **Äquivalent:** eindeutige Ids. | freigegeben in G2b |
