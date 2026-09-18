# Paritätsliste Code-Graph G2b

**Quelle:** `trailhq/Graft` @ `1e352a3` (MIT), `src/ask/ask.ts`, `src/ask/lexicon.ts`,
`src/ask/graphrank.ts`, `src/ask/source.ts`, `src/graph/refresh.ts`.
**Spec:** [2026-09-17-loomux-code-g2-design.md](../specs/2026-09-17-loomux-code-g2-design.md) (Abschnitte 10 und 12.3)
**Plan:** [2026-09-17-loomux-code-g2b.md](../plans/2026-09-17-loomux-code-g2b.md)
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor die Stufe als
fertig gilt.

**Stand 2026-09-18, freigegeben.** Der Nutzer hat am 2026-09-18 alle 21
Verfügungen (§2: 5, §3: 15, §5: 1) freigegeben, nachdem sie gegen den Code auf
`fee7313` nachgerechnet und fünf Begründungen berichtigt wurden. Runde 4 (Code-Review-Korrekturen, siehe §3) bringt eine 22. Verfügung, `refresh.go:122`; auch sie ist am 2026-09-18 freigegeben. Das Tor ist grün auf `HEAD` dieses
Zweigs, Worktree `C:\Users\micro\Documents\#GIT\loomux-code-g2`, Branch
`code-g2`:

```
gofmt -l cmd internal
go vet ./...
go test ./... -count=1 -covermode=set "-coverpkg=github.com/xidus90/loomux/..." -coverprofile=coverage.out
go run ./cmd/loomux dev covergate --profile coverage.out
go build -o bin/loomux.exe ./cmd/loomux
```

`gofmt` und `vet` schweigen, alle 44 Pakete grün, `covergate` endet mit 0 — keine
Funktion außerhalb von `third_party/toml` liegt unter 100 %. Ausgenommen mit
`//coverage:exempt <grund>` auf der Zeile unmittelbar über `func` sind in dieser
Stufe genau zwei Funktionen, beide wegen eines unerreichbaren
`MarshalIndent`-Zweigs: `graphAsk` in `internal/cli/graph.go` und `Write` in
`internal/code/lexicon/index.go`. `writeEverything` trägt keine Ausnahme mehr;
sein Schreibzweig für den Fingerabdruck ist getestet.

## 1. Portierte Vektoren der Referenz

Abschnitt 12.3 der Spezifikation nennt die portierten Vektoren aus `trailhq/Graft`
namentlich. Jeder Vektor ist durch automatisierte Tests in dieser Stufe abgedeckt:

| Quelle in Graft | Test dieser Stufe | Eigenschaft / Verhalten |
|---|---|---|
| `test/ask.test.ts:112` | `TestGraphAskAnswersWithLocationsAndNoCode` | Ohne `--source` liefert `ask` nur Fundort (Pfad, Zeilen), ID, Signatur und Score; kein Quelltext |
| `test/ask.test.ts:154` | `TestGraphAskSourceInlinesTheSpan`, `TestInlineCutsTheSpanOutOfTheFile` | Span-Rückfall: Mit `--source` wird der exakte Zeilenschnitt aus der Datei eingeblendet (gedeckelt bei 80 Zeilen) |
| `test/ask.test.ts:470` | `TestGraphAskFindsASymbolByAWordOnlyInItsBody` | Ein Symbol wird auch dann gefunden und gerankt, wenn der Suchbegriff nur im Funktionsrumpf vorkommt |
| `test/ask.test.ts:491` | `TestGraphAskFindsAFileByAWordInItsImportHeader` | Eine Datei wird als Dokument über Begriffe in ihrem Import-Header/Modulkopf gefunden |
| `test/ask.test.ts:40`, `:72` | `TestRunDeRanksATestFile`, `TestRunDeRanksATestEvenWhenItIsTheStrongestRawMatch`, `TestRunLiftsThePenaltyWhenTheQueryAsksForTests` | Testdateien werden mit Faktor 0,35 abgewertet (`testRankPenalty` in `internal/code/ask/ask.go`), selbst als stärkster Rohtreffer; die Strafe entfällt, wenn die Abfrage nach Tests fragt |
| `test/ask.test.ts:890` | `TestRunFiltersBeforeScoringAndSegmentAware`, `TestFilterIsSegmentAware` | `--in <prefix>` ist segmentbewusst (`lib` trifft `lib/foo`, aber nicht `liberty`) und filtert vor dem Scoring |
| `test/ask.test.ts:928` | `TestFilterRecomputesDocumentFrequencyOverTheRemainder` | Die Dokumenthäufigkeit (DF) wird über der Restmenge unterhalb von `--in` neu berechnet, nicht global übernommen |
| `test/ask-index.test.ts` | `TestWriteThenReadRoundTrips`, `TestBuildCountsDocumentFrequencyAcrossFields` | Die Beiakte `ask-index.json` ist ein exakter Cache für DF, Dokumentanzahl und Rumpftokens |

## 2. Verfügungen dieser Stufe, die keine Mutation betreffen

- **Kein Crux.** Der Zeilenschnitt (`source.go:slice`) schneidet den AST-Span
  `from..to` aus der Datei. Das ist Grafts eigener Rückfall für Sprachen und
  Knoten ohne Crux-Extraktor (Abschnitt 3.1 der Spec). Ein heuristischer Crux
  ohne Sprachwissen existiert nicht; die Crux-Erzeugung wandert mit den
  Sprach-Plugins in spätere Stufen.
- **Die Beiakte ist ein Cache.** Ihr Verlust bricht die Abfrage nicht: Fehlt
  `ask-index.json` (oder ist sie unlesbar), meldet `loomux graph ask` dies auf
  `stderr` und fällt auf das Ranking über Symbolnamen und Pfade zurück. Das
  Ergebnis bleibt valide, verliert jedoch Rumpftreffer.
- **Die Messung zur Beiakte.** Auf diesem Repository (~3.000 Knoten, 267 Dateien)
  dauert `graph ask` warm mit Beiakte ~48 ms (Median 47,6 ms), ohne Beiakte
  ~37 ms (Median 37,5 ms). Das Deserialisieren des 1-MB-JSON-Index kostet ~10–12 ms.
  Bei 3.000 Knoten ist Live-Matching im RAM schneller als das Laden des Index;
  die Beiakte existiert, um auf 30.000+ Knoten zu skalieren (wo Graft ~45 %
  Ersparnis maß), und wird hier ehrlich als repo-spezifischer Messwert
  festgehalten (`docs/{en,de}/benchmarks.md`).
- **Der Testfaktor wirkt zweimal.** Der Dämpfungsfaktor heißt
  `testRankPenalty` und ist **0,35**, aus der Referenz übernommen; `testFactor`
  ist die Funktion, die ihn je Pfad ausliefert. Er wirkt auf den lexikalischen
  Rohwert (`ask.go`, `lexical(...) * factor(n.Path)`) und bei der finalen
  Verschmelzung nochmals (`ask.go`, `(lexNorm + graphWeight*graph) *
  factor(n.Path)`), erkannt über `isTestPath` (`_test.go`, `.test.*`,
  `.spec.*`, Verzeichnisse `test/`, `tests/`, `__tests__/`, `spec/`), es sei
  denn, die Abfrage enthält ein Test-Schlüsselwort. Die Liste ist
  `testSeeking` (`ask.go`) und genau diese, als ganze Wörter und ohne
  Rücksicht auf Groß- und Kleinschreibung: `test`, `tests`, `spec`, `specs`,
  `coverage`, `assert`, `asserts`, `assertion`, `assertions`, `fixture`,
  `fixtures`, `mock`, `mocks`. Eine
  einmalige Dämpfung würde zulassen, dass ein stark vernetzter Test durch
  PageRank-Masse wieder an die Spitze geschwemmt wird.
- **Deterministische Term-Reihenfolge.** Bei Mehrwort-Anfragen werden die
  Abfrage-Begriffe deterministisch alphabetisch sortiert summiert
  (`score.go:newQuestion`, einmal je Abfrage; der Typ `question` ist nur über
  diesen Konstruktor zu bekommen, also nie unsortiert). Dies verhindert, dass
  Go-Map-Iterationsschwankungen gleiche Punktwerte um winzige
  Rundungsdifferenzen kippen.

## 3. Mutationsrunde

Befehl:
```sh
go run ./cmd/loomux dev mutants internal/code/lexicon internal/code/ask
```

**Gesamtzahlen (Runde 4, Stand 2026-09-18):**
- Untersuchte Mutanten: **198** (55 über `internal/code/lexicon`, 143 über `internal/code/ask`)
- Nicht kompiliert (keine Mutante): **35** (11 in `lexicon`, 24 in `ask`)
- Getötet: **147** (41 in `lexicon`, 106 in `ask`)
- **Überlebt: 16** (3 in `lexicon`, 13 in `ask`)

**Runde 4** folgt auf zwei Korrekturen aus dem Code-Review vom 2026-09-18:
`lexicon.Write` schreibt die Beiakte jetzt über eine temporäre Datei und ein
Rename (ein Lauf, der die Rebuild-Sperre verloren hat, las sonst womöglich eine
halb geschriebene Beiakte), und eine verwaiste Rebuild-Sperre wird über
`takeOver` gebrochen — Rename auf einen eigenen Namen, dann Alter der
umbenannten Datei prüfen —, statt per Pfad gelöscht (zwei Läufe konnten sonst
beide bauen). Die neuen Zweige sind getestet
(`TestWriteKeepsTheOldSidecarWhenTheReplacementCannotBeWritten`,
`TestWriteReportsATargetItCannotReplace`, `TestTakeOverLeavesALiveLock`,
`TestTakeOverFailsWhenTheLockVanished`). Es überleben dieselben 15 wie in
Runde 3, nur mit verschobenen Zeilennummern, und **ein neuer**:
`refresh.go:122`, dieselbe Grenze `< lockStale` in `takeOver`. Die
Zeilennummern der Tabelle sind die aus Runde 4.

Die Runde wurde nach der Zusammenlegung der Präfix-Vergleiche und dem Umzug der
Term-Reihenfolge (`newQuestion`) erneut gefahren: **dieselben 15 Überlebenden**,
kein neuer. Es gibt eine Mutante weniger als in Runde 2 (133 statt 134 in `ask`),
weil `ask.underPrefix` entfallen ist und sein Vergleich nun nur noch einmal — in
`lexicon` — existiert. Die Zeilennummern in der Tabelle sind die nach dieser
Änderung; die Nummern im Text der Korrekturen unten sind die aus Runde 2, unter
denen die Mutanten gemeldet wurden.

Die Runde wird **je Paket** gefahren, und `dev mutants` startet zu jeder Mutante
nur `go test ./<paket>/`. Ein Fehlerzweig, den erst ein Test eines **anderen**
Pakets erreicht, kann hier also überleben, ohne ungetestet zu sein — diese
Unterscheidung steht unten in den Verfügungen, wo sie zutrifft.

In Runde 1 überlebten noch 21 Mutanten. Vier davon deckten fehlende Tests auf und
wurden in Runde 2 getötet:
1. `source.go:55` (a3, `from > to -> from >= to`): Ein-Zeilen-Spans (`L3-L3`) wurden
   von keinem Test geprüft. Behoben durch `TestInlineSingleLineSpan`.
2. `source.go:59` (a3, `len > 80 -> len >= 80`): Ein exakt 80 Zeilen langer Span
   erhielt bei `>= 80` fälschlich einen Auslassungs-Marker. Behoben durch
   `TestInlineExactlyEightyLinesHasNoMarker`.
3. `ask.go:168` (a1, `len(hits) == 0 -> true`): Wenn Treffer existieren, durfte `Note`
   nicht gesetzt werden. Behoben durch explizite `Note == ""` Prüfung in
   `TestRunRanksByBlendedLexicalAndPPR`.
4. `index.go:125` (a1, `out.DocCount > 0 -> false`): `AvgBodyLen` auf gefilterten
   Teilbäumen wurde nicht validiert. Behoben in
   `TestFilterRecomputesDocumentFrequencyOverTheRemainder`.

**Runde 3 (2026-09-18) korrigiert die Runde-2-Tabelle an vier Stellen.** Beim
Nachrechnen der Verfügungen gegen den Code — nicht gegen die Tests — hielten drei
von ihnen nicht: sie begründeten einen Überlebenden mit einer Aussage über die
*Tests* („äquivalent bei Einzeltreffern“, „äquivalent im Normalpfad“), und das
ist nach der Regel des Plans ein fehlender Test und keine Äquivalenz. Dazu
benannten zwei Zeilen die falsche Anweisung. Drei Mutanten sind daraufhin
getötet:

1. `ask.go:113` (a1, `if score > maxLex -> if true`): Ohne die Vergleichswache
   ist `maxLex` der Wert des **letzten** Dokuments in ID-Reihenfolge; das
   skaliert jedes `lexNorm` um, während die Graph-Achse fest bleibt. Kein Test
   fiel darauf, weil jeder von ihnen seinen stärksten Treffer zufällig zuletzt
   sortiert hat (`overlay/…` nach `core/…`) — dort sind Maximum und letzter Wert
   dieselbe Zahl. Behoben durch
   `TestRunNormalizesTheLexicalAxisByTheMaximumAndNotTheLastDocument`: `a.go`
   trifft per Name, `z.go` nur im Rumpf. Beleg: unter der Mutante liefert der
   Test `lexical = 2.3045…` statt exakt 1.
2. `ask.go:153` (a1, `if hits[i].Score != hits[j].Score -> if true`): Die
   Mutante macht den ID-Tiebreak unerreichbar, und eine echte Punktgleichheit
   sortiert dann in der Reihenfolge einer Map-Iteration. Das ist pro Lauf ein
   Münzwurf — `TestRunIsDeterministicOnATie` verglich fünf Läufe und prüfte die
   ID **einmal**, was mit ~3 % Wahrscheinlichkeit grün bleibt, und genau dieser
   Fall trat in Runde 2 ein. Behoben durch 50 Läufe mit der ID-Zusicherung
   **innerhalb** der Schleife; Beleg: die Mutante fällt im Lauf 1.
3. `lexicon/index.go:172` (a1, `if err != nil -> if false`): Das ist die
   Fehlerprüfung von `os.ReadFile` in `Read` — die Runde-2-Zeile schrieb sie
   `os.WriteFile` in `Write` zu. Ohne sie läuft `Read` mit `b == nil` weiter,
   `json.Unmarshal` scheitert ebenfalls, und „keine Beiakte“ kommt beim Aufrufer
   als „halbe Beiakte“ an. Behoben, indem
   `TestReadReportsAMissingSidecar` jetzt `errors.Is(err, fs.ErrNotExist)`
   verlangt; Beleg: unter der Mutante meldet der Test
   `read ask index: unexpected end of JSON input`.

Die vierte Korrektur ist eine reine Richtigstellung: `lexicon/index.go:159` ist
die Fehlerprüfung von `json.MarshalIndent`, nicht die von `os.MkdirAll`. Zu
`os.MkdirAll` (Zeile 163) existiert gar kein a1-Mutant — `err` wäre dann
deklariert und unbenutzt, der Mutant kompiliert nicht —, und der Zweig ist durch
`TestWriteReportsAnUnusableCacheDirectory` im eigenen Paket abgedeckt. Denselben
Weg erreicht zusätzlich `internal/cli`, wo ein Test eine reguläre Datei dort
anlegt, wo das `cache`-Verzeichnis hingehört.

Die verbliebenen 16 Überlebenden sind semantisch äquivalent; jede Verfügung
argumentiert unten aus dem Code, nicht aus der Abdeckung:

| Ort | Familie | Mutation | Ausgang | Verfügung | Freigabe |
|---|---|---|---|---|---|
| `lexicon/index.go:163` | a3 | `if i := strings.Index(id, "#"); i >= 0 {` $\to$ `... i > 0 {` | Überlebt | **Äquivalent:** Symbol-IDs in loomux folgen `path/file.go#Symbol`. Der Pfad steht immer vor `#`. Bei `i == 0` (z. B. `#Symbol`) ergäbe `id[:0]` einen leeren Pfad `""`; bei `i > 0` bliebe `#Symbol`. Da keine ID mit `#` beginnt, verhält sich beides identisch. | freigegeben 2026-09-18 |
| `lexicon/index.go:181` | a1 | `if err != nil {` $\to$ `if false {` | Überlebt | **Äquivalent, weil unerreichbar:** Das ist die Fehlerprüfung von `json.MarshalIndent` in `Write` (nicht die von `os.MkdirAll`, Zeile 185). `Write` trägt bereits ein `//coverage:exempt` auf der Zeile über `func` — die Ausnahme gilt der ganzen Funktion, nicht diesem Zweig allein —, und seine Begründung argumentiert aus den Typen: ein `Index` ist ein `int`, ein endlicher `float64`, eine `map[string]int` und eine Scheibe davon — kein `Index`, den dieses Programm baut, lässt `MarshalIndent` scheitern. Ein Zweig, den kein Wert erreicht, ist von keinem Test zu töten. | freigegeben 2026-09-18 |
| `lexicon/index.go:219` | a3 | `return docs[i].ID < docs[j].ID` $\to$ `... docs[i].ID <= docs[j].ID` | Überlebt | **Äquivalent:** `sort.Slice` über eindeutige IDs. Da alle Dokument-IDs paarweise verschieden sind (`docs[i].ID != docs[j].ID`), sind `<` und `<=` identisch; ein Element wird nie mit sich selbst verglichen. | freigegeben 2026-09-18 |
| `ask/ask.go:86` | a1 | `if in != "" {` $\to$ `if true {` | Überlebt | **Äquivalent:** Bei leerem Präfix `in == ""` normalisiert `NormalizePrefix("")` zu `""`. `Filter` normalisiert selbst und gibt bei leerem Präfix sofort den unveränderten Index `ix` zurück (`index.go:107`); die Mutante ruft es also nur umsonst. Die Knotenseite filtert `walk` getrennt, hinter seiner eigenen Wache `prefix != ""`, die von dieser Mutante unberührt bleibt. Das Ergebnis ist exakt dasselbe. | freigegeben 2026-09-18 |
| `ask/ask.go:116` | a3 | `if score > maxLex {` $\to$ `if score >= maxLex {` | Überlebt | **Äquivalent:** Wenn `score == maxLex`, weist die Mutante `maxLex` denselben Wert erneut zu. Das Maximum bleibt unverändert. | freigegeben 2026-09-18 |
| `ask/ask.go:128` | a3 | `if s.Score >= rescueFloor {` $\to$ `... s.Score > rescueFloor {` | Überlebt | **Äquivalent bei `pagerank.defaultIterations = 25`, nicht grundsätzlich:** Bei Konvergenz läge der Fall n = 5 exakt auf 0,15 (§4). Nach 25 Iterationen hat er die Masse `0.150197654`, der Nachbarfall `0.125164712` (`TestRunRescuesJustAboveTheFloorAndNotJustBelowIt`); keiner trifft die Schwelle genau, also ergeben `>=` und `>` dieselbe Entscheidung. Ändert sich die Iterationszahl, ist diese Verfügung neu zu prüfen. | freigegeben 2026-09-18 |
| `ask/ask.go:157` | a3 | `return hits[i].Score > hits[j].Score` $\to$ `... hits[i].Score >= hits[j].Score` | Überlebt | **Äquivalent:** Wird nur erreicht, wenn Zeile 156 (`hits[i].Score != hits[j].Score`) bereits feststellte, dass die Scores ungleich sind. Für ungleiche Floats sind `>` und `>=` identisch. | freigegeben 2026-09-18 |
| `ask/ask.go:159` | a3 | `return hits[i].ID < hits[j].ID` $\to$ `... hits[i].ID <= hits[j].ID` | Überlebt | **Äquivalent:** Tie-Breaker bei identischem Score. Da jede Node-ID im Graphen eindeutig ist, sind zwei verschiedene Treffer niemals identisch (`ID_i != ID_j`). Für ungleiche Strings sind `<` und `<=` identisch. | freigegeben 2026-09-18 |
| `ask/ask.go:166` | a3 | `if len(hits) > limit {` $\to$ `if len(hits) >= limit {` | Überlebt | **Äquivalent:** Wenn `len(hits) == limit`, schneidet `hits[:limit]` die Scheibe auf `hits[:len(hits)]` (No-Op). Für `>` schneiden beide auf `limit` ab. | freigegeben 2026-09-18 |
| `ask/ask.go:191` | a1 | `if len(seed) == 0 {` $\to$ `if false {` | Überlebt | **Äquivalent aus dem Code von `pagerank.Rank`:** `Rank` summiert die Startgewichte und kehrt bei `total <= 0` mit `nil` zurück (`rank.go`, „a seed set that is empty after that yields no result at all“). Eine leere Seed-Map summiert zu 0, also liefern beide Formen dieselbe leere Scheibe; die Wache spart nur `Prepare` und die Iteration. Kein Beobachter kann die beiden unterscheiden. | freigegeben 2026-09-18 |
| `ask/refresh.go:94` | a3 | `if statErr != nil || time.Since(info.ModTime()) < lockStale {` $\to$ `... <= lockStale {` | Überlebt | **Äquivalent bis auf die Uhrauflösung:** Die Grenze ist erreichbar, nur nicht beobachtbar. Uhr und NTFS-Zeitstempel ticken unter Windows in 100 ns; `<` und `<=` unterscheiden sich nur, wenn das Alter der Sperre genau `lockStale` (eine Stunde) beträgt — dann gilt eine Sperre um einen Takt früher als verwaist. Kein Test kann diesen Takt treffen, und das Verhalten beiderseits der Grenze ist dasselbe. | freigegeben 2026-09-18 |
| `ask/refresh.go:122` | a3 | `if err != nil \|\| time.Since(got.ModTime()) < lockStale {` $	o$ `... <= lockStale {` | Überlebt | **Äquivalent bis auf die Uhrauflösung, aus demselben Grund wie die Zeile darüber:** `takeOver` prüft die umbenannte Sperre gegen dieselbe Grenze `lockStale`; `<` und `<=` unterscheiden sich nur bei einem Alter von genau einer Stunde, auf den Takt von 100 ns genau. | freigegeben 2026-09-18 |
| `ask/source.go:30` | a1 | `if !ok {` $\to$ `if false {` | Überlebt | **Äquivalent:** Wenn `Span.Lines()` `ok == false` meldet, sind `from = 0, to = 0`. `slice` erhält `0, 0`, klemmt `from = 1`, stellt `from > to` (1 > 0) fest und gibt `"", false` zurück. `Inline` weist `h.Code` nur zu, wenn `slice` `ok == true` meldet. Die Wache in Zeile 30 spart nur den Aufruf von `slice`. | freigegeben 2026-09-18 |
| `ask/source.go:34` | a1 | `if !ok {` $\to$ `if false {` | Überlebt | **Äquivalent:** Wenn `slice` `ok == false` liefert, ist `code == ""`. Die Mutante weist `h.Code = ""` zu, was dem Initialwert von `h.Code` entspricht. | freigegeben 2026-09-18 |
| `ask/source.go:48` | a3 | `if from < 1 {` $\to$ `if from <= 1 {` | Überlebt | **Äquivalent:** Bei `from == 1` weist die Mutante `from = 1` zu (identischer Wert). | freigegeben 2026-09-18 |
| `ask/source.go:52` | a3 | `if to > len(lines) {` $\to$ `if to >= len(lines) {` | Überlebt | **Äquivalent:** Bei `to == len(lines)` weist die Mutante `to = len(lines)` zu (identischer Wert). | freigegeben 2026-09-18 |

## 4. Vorbehalt: die Plan-Tests pinnten keine einzige Rangkonstante

Diese Runde ist erst in **Runde 3** so sauber. Wer die Tabelle oben liest, darf
daraus **nicht** schließen, dass die Tests des Plans ausgereicht hätten: von den
fünf Rangkonstanten dieser Stufe war zum Zeitpunkt der Plan-Tests **keine
einzige** gepinnt. Gemessen während der Ausführung, nicht geschätzt:

- **`k1 = 1.2` und `b = 0.75`:** Eine Implementierung mit `k1 = 1.5, b = 0.5`
  bestand alle neun Tests von Aufgabe 4. Die Sättigungsprüfungen sind
  Ungleichungen, und bei einem Längenverhältnis von exakt 1 kollabiert
  `(1 - b + b*ratio)` zu 1 — `b` war unsichtbar.
- **Der Abfragezähler in `overlap`:** ungeprüft; das Weglassen des Faktors
  `float64(query[term])` fiel keinem Test auf.
- **`graphWeight = 0.5`:** `TestRunDoesNotLetConnectivityOverruleAClearLexicalWinner`
  liefert genau **einen** Treffer — das Hub-Cluster wird nie geseedet, weil
  „Hub“ zu `hub` tokenisiert und `X`/`Y` am Einzelzeichen-Gate scheitern. Der
  Test hält damit bei jedem `graphWeight`, 50 eingeschlossen.
- **`rescueFloor = 0.15`:** Der gerettete Helfer des Plan-Tests sitzt bei Masse
  0,750988 — dem Fünffachen der Schwelle. Die im Plan selbst genannte Mutation
  0.15 → 0.16 überlebte ihn.

Für alle fünf wurden **während der Ausführung** Pins nachgezogen (`k1`, `b`, der
Abfragezähler, `graphWeight`, `rescueFloor`), jeder mit einer absichtlichen
Mutation als Beleg, dass der neue Test unterscheidet.

**Die ehrliche Grenze des Rescue-Pins.** Das Paar aus
`TestRunRescuesJustAboveTheFloorAndNotJustBelowIt` klammert die Konstante nur
als `0.125164712 < rescueFloor <= 0.150197654`. Eine Mutation **nach unten** auf
0.14 überlebt also. Und die 2e-4 Luft über der Schwelle existieren einzig,
weil `pagerank.defaultIterations` 25 ist: bei Konvergenz liegt der Fall n = 5
exakt auf 0,15. Dieser Test hält damit auch die Iterationszahl fest, nicht nur
den Boden.

## 5. Akzeptanzbeobachtung (Aufgabe 8, Schritt 5)

Kein Test leistet diese Beobachtung; der Plan verlangt sie ausdrücklich als
Eintrag hier und nicht als stille Korrektur an einer Konstante.

```
$ ./bin/loomux.exe graph ask "how does the write barrier decide what to refuse" --limit 5
1. internal/brain/guard/run.go#Refuse    internal/brain/guard/run.go:L61-L69      (1.121 lex 1.000 graph 0.242)
2. internal/cli/brain.go#brainRefuse     internal/cli/brain.go:L72-L74            (1.008 lex 0.938 graph 0.139)
3. internal/brain/guard/guard.go#Decide  internal/brain/guard/guard.go:L398-L546  (0.775 lex 0.695 graph 0.159)
4. internal/hosts/hostio.go#WriteContext internal/hosts/hostio.go:L100-L113       (0.503 lex 0.364 graph 0.278)
5. internal/cli/brain.go#brainCommand    internal/cli/brain.go:L36-L68            (0.451 lex 0.363 graph 0.176)
```

Die Zahlen stammen aus einem Lauf vor dem Rebase auf `master`; die dort genannten
Commits gibt es nicht mehr. Nachgeprüft auf `HEAD fee7313` (nach dem Rebase auf
`master` `e1d3f8d`, 2026-09-18): 1.120 / 1.007 / 0.779 / 0.504 / 0.450 bei
unveränderter Reihenfolge. Der Korpus dieser Abfrage ist dieses Repository selbst,
also wandern die Werte in der dritten Dezimale mit jeder Änderung an ihm. Die Beobachtung ist die Rangfolge, nicht die
Zahl.

Der eigentliche Policy-Entscheider ist `guard.Decide` — er steht auf Platz 3.
Davor stehen die beiden `Refuse`-Funktionen, weil das stärkste Wort der Frage
„refuse“ ist und `nameWeight = 3` einen Namenstreffer dreifach zählt: `Refuse`
trifft mit dem Namen, `Decide` nur mit Pfad und Rumpf. Das ist keine Panne,
sondern die Referenzgewichtung bei einer Frage, die den Mechanismus („decide“)
nur nebenbei nennt und das Ergebnis („refuse“) betont.

**Verfügung: keine Änderung.** Weder eine Konstante noch ein Feldgewicht wird
darauf hin gestimmt. Die Antwort ist vertretbar — alle drei Treffer gehören zur
Schreibbarriere, der Leser landet in der richtigen Ecke —, und eine Anpassung
der Feldgewichte gegen ein einzelnes Beispiel würde die Parität zur Referenz
aufgeben, ohne dass ein zweites Beispiel sie stützt. Wer die Gewichtung später
bewegen will, braucht eine Beispielsammlung, nicht diese eine Abfrage.

## 6. Was G2 ganz offen lässt

Mit G2a und G2b ist das Fundament des deterministischen Code-Graphen
(Extraktion, Frischeprüfung, Wiring-Speicherung, Lexik-Indizierung, BM25-Scoring,
Personalized PageRank und `loomux graph ask`) vollständig implementiert, gemessen
und durch 100 % Coverage sowie Paritätsprüfungen abgesichert.

Folgende Bestandteile verbleiben für die Folge-Stufen:

- **`--lsp` gegen `gopls`:** Die optionale Folgestufe zur Anreicherung semantischer Typkanten (Abschnitt 14 der Spec).
- **Graph-Navigationsbefehle (Stufe G4):** `loomux graph callers`, `blast`, `grep`, `skeleton`, `map`, `stats`, `viz` — samt Normalisierung von `Depth` und `Direction` an der CLI-Grenze.
- **MCP-Gateway-Anbindung (Stufe G3):** Die sechs `graph_*`-Werkzeuge (`graph_ask`, `graph_callers`, `graph_blast`, `graph_grep`, `graph_skeleton`, `graph_map`) über den MCP-Server `loomux serve`.
- **Check-Lanes & Post-Edit-Hook (Stufe G4):** Dirty-File-Hashing und Blast-Radius-Warnung im `post-tool-use`-Hook.
- **FileCard & semantische Crux-Extraktion:** Vollwertige Crux-Extraktoren mit syntaktischer Verdichtung für weitere Sprachen.
