# ulflow M1, Welle 2 — Befunde und Entscheidungen

**Plan:** [2026-09-11-ulflow-welle-2.md](2026-09-11-ulflow-welle-2.md) ·
**Spec:** [2026-09-11-ulflow-laufzeit-design.md](../specs/2026-09-11-ulflow-laufzeit-design.md) ·
**Vorgänger:** [2026-09-11-ulflow-welle-0-und-1-befunde.md](2026-09-11-ulflow-welle-0-und-1-befunde.md)
**Stand:** zusammengeführt am 2026-09-11, `feature/agent-harness` bis `4c8144f`.
Das Abschlussreview über alle 16 Commits hat drei Befunde „vor dem Merge“
gemeldet. Die Fix-Runde brach am 2026-09-11 mit der Sitzung ab; behoben sind sie
am 2026-09-13 in `7b47bd7` (siehe „Aus dem Abschlussreview“). Geschrieben am
2026-09-13 aus Ledger, Task-0-Bericht, Abschlussreview und Git-Historie.

Was die Umsetzung entschieden, gefunden und für später offen gelassen hat. Die
Arbeitsdateien der Ausführung liegen noch unter
`.superpowers/sdd/2026-09-11-ulflow-welle-2/` (git-ignoriert); sie können weg,
sobald die Fix-Runde durch ist.

## Entscheidungen während der Ausführung

Die fünfzehn Punkte aus Welle 1 und die vier neuen hat der Plan selbst
entschieden (Abschnitt „Entscheidungen, die dieser Plan trifft“); sie stehen
hier nicht noch einmal. Hier steht, was während der Ausführung dazukam.

| Entscheidung | Grund | Kosten, falls falsch |
|---|---|---|
| Die vier Lanes liefen parallel, jede in `.worktrees/ulflow-<lane>`, entgegen der Skill-Regel „nie mehrere Implementierer parallel“ | Die Regel schützt vor Kollisionen in einem Arbeitsbaum; hier eigener Worktree, eigener Zweig ab `6f26a77`, Dateimengen im Vorab-Scan disjunkt. Welle 1 lief mit sieben Lanes so | ein Merge-Konflikt, eine Lane neu |
| Implementierer und Reviews auf Opus, die eingegrenzten Nach-Reviews auf Sonnet | — | nur Kosten |
| Vorab-Scan F1: die Variable in `catalog_test.go` heißt `holds`, nicht `always` | `always` ist im selben Testpaket schon ein Typ (`contract_test.go`) | ein Name |
| Vorab-Scan F2: der Gate-Test erwartet `"Approve after 2 rounds?\n"` mit Zeilenende | `Render` kopiert die Datei byteweise; der Plantext konnte nicht bestehen. Eine Textdatei ohne Schlusszeilenumbruch wäre die schlechtere Richtung | eine Assertion |
| Vorab-Scan F3: `agent_test.go` bekommt `agentState()` und `agentEnv(m, c)` | Fünf Tests des Plans renderten `draft.md` ohne `topic` und `max_rounds`; `tmpl.Render` bricht dann ab | — |
| `toInt` bekommt einen Helfer `fromFloat`, der erst den `int64`-Bereich und dann die Ganzzahligkeit prüft — gegen den Code des Plans | `float64(1)` galt als Zahl, `json.Number("1.0")` als Bruch; `1e3` hieß „not a whole number“; `float64(1e19)` lief beim Umwandeln über und nannte sich ebenfalls Bruch. Ein `Coerce` ist genau dafür da, dass ein Flow geladen und nachgegangen dasselbe bedeutet | ein Helfer, zwei Testzeilen |
| `var _ Registry = (*Catalog)(nil)`, und `toList` kopiert ein `[]string` | Vier Lanes zweigen von diesem Commit ab: ein `Catalog`, der `Registry` nicht mehr erfüllt, bräche alle vier und `internal/flow` bliebe grün. Eine geteilte Scheibe ließe den Schreibvorgang eines Knotens einen gespeicherten Wert erreichen | zwei Zeilen |
| Eine zusätzliche Testzeile `json.Number("x")` in Task 0 | sonst 98,x % statt 100 % | eine Zeile |
| `agent`: ein privates `agentDefine` liefert `(Definition, Resolved, error)`; `Run` löst das Modell nicht ein zweites Mal auf — gegen die Form des Plans | Der zweite `Resolve` lief erst nach einem erfolgreichen ersten mit denselben Argumenten; sein Fehlerzweig war tot, 100 % unerreichbar | eine private Funktion |
| Bausteine stellen ihren Fehlern nichts voran; der Aufrufer ordnet zu | Der Journaleintrag trägt `node` als eigenes Feld, der Lader präfixt selbst (sonst `node "draft": node "draft": model …`), und `gate`/`exit` präfixten ohnehin nicht | vier Wrapper und ihre Assertions |
| Ein unbekannter Antworttyp ist in `agent` ein Fehler | Er fiel auf `ReplyType("")` durch und hätte eine bezahlte Anfrage mit leerem Typ geschickt | ein Guard |
| Abgelehnt: der Review-Befund, `Definition` habe keinen Platz für das Antwortschema | Der Runner hasht `node.Raw`, den ganzen TOML-Eintrag samt `reply`-Tabelle; eine geänderte Antwort ändert den `definition_hash` | — |
| `exit`: `raw, _ := flow.ReadText(…)` mit Kommentar statt eines Fehlerzweigs | `Exit.Texts` setzt nie einen `Path`, `ReadText` kann dann nicht scheitern; der Zweig ließ 97,7 % | stumm, falls `exit` je eine Datei bekommt |
| Runner: ein Kantenziel ohne Knoten ist ein Go-Fehler (`the flow has no node %q`) | Kein zweiter Prüfdurchgang, sondern die Abfrage, die den Namen gerade benutzt. Vorher: unter Replay `exceeded max_visits=0`, sonst Panik in fremdem `Define`. Handgebaute Graphen (Runner-Tests, Golden-Journal) laufen nicht durch den Lader | zwei Zeilen und ein Test |
| Runner: `Run` setzt mit dem Nachgeh-Flag auch den Journal-Schnappschuss eines früheren `Resume` zurück | Der Code behauptete den Rücksetzer und setzte nur die Hälfte; die andere Hälfte verlöre still einen Pausen-Eintrag. Kein heutiger Aufrufer erreicht es | eine Zeile und ein Test |
| Lader: ein vorhandener Schlüssel mit falschem Werttyp ist ein Befund, über die Stufe-1-Liste des Briefs hinaus. Vier Leser `text`, `flag`, `table`, `entries` | `on_error = "true"` machte aus einer Fehlerkante eine unbedingte, `when = 3` ließ die Bedingung verschwinden — im Paket, das einen Flow vor dem Lauf ablehnen soll. Ein fehlender Schlüssel bleibt stumm (die Leser fragen `present`, nicht `ok`) | fünf Befunde, fünf Testzeilen |
| Der Flow-Name fällt unter die reservierten Namen | Knoten, Felder und Parameter taten es schon; ein Flow namens `END` oder `true` lud | eine Zeile |
| Die doppelte Meldung für einen falsch typisierten `[flow] name` bleibt | Beide Zeilen nennen Ursache und Folge und zeigen auf dieselbe Zeile | — |
| Abschlussreview: das als `ok` journalisierte undeklarierte Delta wird von „minor, zurückgestellt“ auf „vor dem Merge“ hochgestuft | Die Zurückstellung nach dem Task-4-Review übersah, dass der Knoten auch seine Fehlerkante nicht angeboten bekommt | — |
| Abschlussreview: für Replay über `exit` und Fehlerkanten die gründliche Reparatur, nicht nur eine bessere Meldung | Die Spec benutzt einen Flow, der über `exit` endet, als eigenes Beispiel; eine Meldung allein ließe ihn unreproduzierbar | ein Nachgehzweig und zwei Tests |

## Aus dem Abschlussreview

Kein Critical. Die Befunde 1–4 waren am 2026-09-13 gegen `4c8144f` nachgeprüft
noch alle im Code. Die Fix-Runde ging an den ursprünglichen Runner-Implementierer,
auf den zusammengeführten Baum umgelenkt, weil sein Lane-Worktree schon abgebaut
war. Sie begann um 22:47, und ihre letzte Nachricht um 22:49 kündigte nur an, den
Plan prüfen zu lassen. Danach endete die Sitzung ohne Commit und ohne Änderung im
Baum.

**Behoben in `7b47bd7`** (2026-09-13), jeweils mit Tests, die vorher rot waren:

- 1 und 2: `walk` baut den nächsten Zustand vor dem Schreiben; ein Delta, das der
  Flow nicht halten kann, geht in den Fehlerzweig und schreibt Tokens und Modell
  mit. Beide Tests prüfen jetzt das Journal; neu ist
  `TestADeltaTheFlowCannotHoldTakesTheErrorEdge`.
- 3: Replay liest einen `error`-Eintrag unter dem Schlüssel als erreichtes Ende
  (`recordedFailure`). Geht das Journal danach weiter, folgt Replay der
  Fehlerkante, sonst endet es mit dem `detail` des Eintrags und ohne Exit-Code.
  Neu sind `TestReplayReproducesAnExit`, `TestReplayFollowsTheErrorEdgeTheRunTook`,
  `TestReplayEndsAtAFailureTheJournalDoesNotGoPast` und
  `TestReplayOfAFailureWithoutADetailEndsWithoutOne`.
- 4: `paramChecks` in `startChecks` lehnt einen deklarierten Parameter ab, der
  fehlt oder einen anderen Go-Typ hat als den, den `flow.Coerce` daraus macht.
  Aufgefüllt wird nicht, weil `flow.Params` laut `types.go:40` schon mit Vorgaben
  ankommt.

Befund 5 ist am 2026-09-13 behoben: `Agent.Check` meldet ein vorhandenes `tools`
oder `effort`, das keine Zeichenkette ist (`tools is 3, not text`), im Wortlaut
der Typbefunde des Laders.

**Nachreview** über `68c7107..d2dd57f` (2026-09-13, Opus, nur gelesen): kein
Critical, zwei Important, sieben Minor. Beide Important wurden mit einem roten
Test nachgemessen, nicht nur nachgerechnet.

- **Behoben in `e057172`:**
  - Ein `Resume(nil)` an einem Tor, das als Fallback hinter einem gescheiterten
    Knoten steht, begrub die Pause. Das Nachgehen findet für den Knoten kein
    `ok` und führt ihn erneut aus. Sein zweiter `error`-Eintrag landet hinter
    der Pause, und das Tor schrieb keine neue, weil dieser Besuch schon als
    pausiert verzeichnet war. `journal.Pending` liest nur den letzten Eintrag
    (`gate.go:28`). Also war das Tor weg: Replay war nicht mehr gesperrt, und
    eine Antwort fand „no gate is waiting“. Die Ursache liegt vor diesem Range.
    Die Tests sahen es nicht, weil `memJournal.Pending` anders arbeitet als
    `journal.Pending`. Jetzt wird die Pause erneut geschrieben, sobald der Gang
    etwas angehängt hat.
  - Replay warnt, wenn der Lauf über eine Fehlerkante weiterging, die der Flow
    nicht mehr hat.
  - Drei Kommentare behaupteten mehr, als der Code tut, und sind korrigiert.
  - Zwei Assertions, die eine Mutation überlebten, prüfen jetzt die Meldungen
    der Parameterablehnung und das Modell im Fehlereintrag.
- **Eine Fehlerkante nach `END`: am 2026-09-13 für Weg 3 entschieden und
  umgesetzt.** Die Restlücke steht im Kommentar in `walk.go`: Ein `exit`-Knoten
  mit eigener Fehlerkante nach `END` wird beim Replay als `done` gelesen. Der Lader erlaubt
  `END` als Ziel jeder Kante (`check.go:157`). Scheitert ein Knoten mit
  `on_error → END`, endet der Lauf `done`, und das Journal ist ein einzelner
  `error`-Eintrag. Replay meldet `error` (gemessen). Dasselbe Journal schreibt
  ein `exit`, und der Runner kann beides nicht unterscheiden. Drei Wege:
  1. Der Lader lehnt `on_error → END` ab: eine Prüfung in Stufe 6, eine
     Fixture, eine Ausnahme in der Spec. Kostet Flows ein „scheitert egal“.
  2. Das Journal unterscheidet `exit` von Fehler: Entscheidung 18 revidieren
     und die Verträglichkeit mit `ulguard` und dem Python-Journal prüfen.
  3. Replay wertet eine vorhandene Fehlerkante nach `END` als genommen: eine
     Zeile. Falsch bleibt dann nur ein `exit`-Knoten, der selbst eine
     Fehlerkante nach `END` hat.
- **Notiert:** Scheitert der Merge einer Tor-Antwort (`given != nil`), fehlt die
  Antwort im Fehlereintrag. Über den Lader ist das nicht erreichbar, weil er
  `Writes()` gegen `[state]` prüft; nur handgebaute Graphen kommen dahin.

Wie das Review sie gemeldet hat:

**Vor dem Merge:**

1. **Ein undeklariertes oder falsch typisiertes Delta wird als `ok` journalisiert**
   (`internal/runner/walk.go`: `write` in Z. 148, `advance` → `merge` erst in Z. 151).
   Der Lauf nennt denselben Knoten als Fehlerstelle, bietet ihm aber keine
   Fehlerkante an. Das verstößt gegen Planentscheidung 5 und die Knotenfehler-Regel
   der Spec (`outcome = error`, Fehlerkante, wenn vorhanden). `show` und jedes
   spätere Resume lesen einen erfolgreichen Schritt.
   *Reparatur:* zuerst `merge`, dann schreiben. Scheitert `merge`, geht es in den
   Zweig `runErr != nil`: Fehlereintrag mit der Meldung als `detail`, Fehlerkante
   anbieten. Nur die Kantenwahl bleibt nach dem Schreiben, dort ist `ok` richtig.
2. **Die beiden Tests dazu können nicht scheitern** (`runner_test.go:371`
   `TestADeltaTheFlowDoesNotDeclareIsANodeError`, `:391`
   `TestADeltaOfTheWrongTypeIsANodeError`). Sie prüfen nur `Status` bzw. `Node`,
   nicht `log.lines`, und keiner der beiden Graphen hat eine Fehlerkante.
   *Reparatur:* `log.lines[0].Outcome == "error"` prüfen und einen Fall mit
   Fehlerkante ergänzen, in dem der Lauf dort weitergeht.
3. **Replay reproduziert keinen Lauf, der an einem `exit` endete oder eine
   Fehlerkante nahm** (`walk.go:61–81`). Beim Nachgehen zählen nur `ok`-Einträge;
   `exit` und Knotenfehler schreiben `error` (Planentscheidung 18), also meldet
   Replay `node "stop" is not in the journal` über einen Knoten, der einen Eintrag
   hat. Kein Test deckt das ab. Das ist eine Folge des Plans: Entscheidung 18 hat
   den Exit-Code aus dem Journal genommen.
   *Reparatur (entschieden, die gründliche):* Findet die `ok`-Suche nichts, einen
   `error`-Eintrag unter demselben Schlüssel suchen und dessen Ende reproduzieren.
   Geht das Journal danach weiter, der Fehlerkante folgen, sonst `Status: "error"`
   mit dem `detail` dieses Eintrags. Keinen Exit-Code, weil das Journal keinen
   trägt; `doc.go` sagt das. Das Review meinte, damit erledige sich auch, dass
   `Resume(nil)` unter Replay auf einem Journal mit offenem Tor
   `node "ask" is not in the journal` meldet. Das stimmt nicht: Ein offenes Tor
   hat einen `paused`-Eintrag und keinen `error`-Eintrag, die neue Suche findet
   ihn also nicht. Der Punkt steht unter „Für den Plan von Welle 3“.
4. **`startChecks` sichert Deckel gegen fehlende Parameter, Bedingungen nicht**
   (`runner.go:187–210`). `expr.comparison.Holds` macht `right = params[c.param]`
   und dann `right.(int)` (`condition.go:272, 276`). Fehlt der Parameter in
   `Env.Params`, gibt es eine Panik mitten in der Kantenauswertung, ohne
   Journaleintrag und ohne Meldung. Welle 3 baut als erster Code `flow.Params`.
   Der Kommentar `runner.go:180` („startChecks are the three things a run is
   refused for“) und `doc.go:24` sind damit unvollständig.
   *Reparatur:* in `startChecks` eine Schleife über `r.graph.Params`, die einen
   deklarierten Parameter ablehnt, der in `r.env.Params` fehlt oder den falschen
   Typ hat, oder ihn aus `Field.Default` füllt. Dieselbe Form wie der Deckelcheck
   daneben.

**Bald:**

5. **`agent`: ein vorhandenes, aber nicht als Zeichenkette gesetztes `tools` oder
   `effort` wird ohne Befund verworfen.** `tools = 3` wird still zu `read_only`,
   `effort = 3` ändert still eine Eingabe des `definition_hash` — des einen Werts,
   auf dem die Nachgeh-Warnungen beruhen. *Reparatur:* in `Check` eine Typprüfung
   je Schlüssel, wie `exit.go` sie für `code` macht.

## Messungen

- Die Diffs der vier Lanes gegen die Dateilisten ihrer Tasks: `loader` 38 Dateien,
  nur `internal/flowload`; `gate-exit` 6; `agent` 3; `runner` 6. Keine Datei
  außerhalb der Liste. Nachgerechnet am 2026-09-13 über `M^1..M` der vier Merges.
- Alle vier Merges waren konfliktfrei, auch `gate-exit` und `agent`, die sich
  `internal/blocks` teilen.
- `go test -cover ./...` am 2026-09-13: alles grün. `internal/flow`,
  `internal/flowload`, `internal/blocks` (zum ersten Mal als Ganzes),
  `internal/runner` und `internal/journal` stehen bei 100,0 %. `gofmt -l internal
  cmd` gibt nichts aus.
- Der Verbindungstest `TestThePlanningFlowLoadsAgainstTheRealBlocks` (12 Knoten,
  alle sieben Stufen, echter Katalog) bestand beim ersten Lauf.
- `git ls-remote origin` für `feature/agent-harness` und die vier `ulflow/*`: leer.
  Nichts gepusht (am 2026-09-11 und erneut am 2026-09-13 gelesen).
- RED-Nachweis für `UseNumber`: ohne ihn kam `2^53+1` als
  `9.007199254740992e+15` zurück, also auf `2^53` gerundet — genau der Wert, den
  ein Resume gehasht hätte.
- `ulguard post-edit` brach beim Bearbeiten von `.gitattributes` mit
  `**/*.sh: openBinaryFile: invalid argument` ab: ein Glob, der als Dateiname
  geöffnet wird. Die Bearbeitung selbst gelang. Nicht untersucht, nicht behoben.
- Jeder `exit`-Knoten in den Fixtures des Laders lässt `code` weg, das der echte
  `Exit.Check` verlangt (`exit.go:26–29`); nur `stage2_kinds` setzt einen, und
  zwar den verbotenen Code 3. Kein Testflow mit `exit` ist daher mit dem echten
  Katalog ladbar, und der Verbindungstest berührt `Exit` nie (`planning.toml` hat
  keinen `exit`-Knoten).

## Für den Plan von Welle 3

Die Befunde 1–5 oben sind behoben.

Vorab: **`Resume(nil)` unter Replay auf einem Journal mit offenem Tor** meldet
`node "ask" is not in the journal`. `runner.py` tut dasselbe. `Run` lehnt
denselben Fall unter Replay mit „this run waits at a gate“ ab; `Resume` sollte es
ebenso tun, oder `cmd/flow` ruft `Resume` nie mit Replay auf.

Danach:

1. **Die vier Punkte aus Welle 1 sind unverändert offen.** Kein Commit dieser Welle
   hat `internal/runs` oder `cmd/guard` angefasst:
   1. `ulguard hook session-start` gibt für jeden wartenden Lauf
      `ultraloom resume <id>` aus, auch für Go-Läufe; er braucht die Zeile
      `runtime` der Laufmarke.
   2. `runs.NextID` kann eine vergebene Nummer erneut ausgeben, und
      `journal.Append` schreibt dann still in ein fremdes Journal. Die Marke mit
      `O_EXCL` anlegen.
   3. Die Kommandozeile übergibt `WriteMarker` immer `Runtime: "go"` und die
      Version.
   4. Eine Marke ohne Basis zu schreiben ist ungetestet.
2. **Die Naht zwischen Runner und echtem Journal ist ungetestet.** Die Runner-Tests
   laufen nur gegen das Double `memJournal`. Dessen `Pending` durchsucht alle
   Einträge und hebt eine Pause bei passendem `ok` auf; `journal.Pending` liest nur
   den letzten Eintrag und braucht `Detail != nil`. Für jedes Journal, das dieser
   Runner erzeugt, stimmen beide überein. Das Golden-Journal ist das Erste, was die
   Naht schließt.
3. **Der Verbindungstest sollte den Beispiel-Flow der Spec benutzen** (Abschnitt
   „Das Flow-Format“). Er hat alle drei Knotenarten; `planning.toml` hat keinen
   `exit`.
4. **`ulguard` neu bauen, in beiden Ständen.** Ein Journal aus Welle 2 setzt alle
   zwölf Schlüssel, und ein altes `ulguard` lehnt `definition_hash` als unbekannten
   Schlüssel ab. `cmd/guard/hook_session_start.go:117` liest nur `journal.Pending`,
   nie `Entry.Delta`; `UseNumber` ist dort unsichtbar.
5. **`NewCatalog` speichert ein nil-Prädikat** und findet es wieder:
   `Predicate(name)` liefert `(nil, true)`, und `predicateCondition.Holds` gerät in
   Panik. Der einzige Aufrufer ist `cmd/flow`. Eine Zeile in der Prädikatschleife
   lehnt es ab.
6. **Die Meldung für einen unbekannten Platzhalter nennt nur `known fields:`**
   (`flowload/check.go:80`), obwohl ein Platzhalter auch einen Parameter nennen
   darf. Beide Mengen zusammenführen und `known names:` schreiben.
7. **Der Kommentar zu Graphregel 5** (`flowload/check.go:187`) behauptet, der
   Runner prüfe jeden Deckel gegen die aufgelösten Parameter. Er prüft `≥ 1`,
   nicht `> 1` auf einem Zyklus. `--option rounds=1` auf einem parametrisierten
   Zyklusdeckel besteht beide Prüfungen und endet zur Laufzeit mit einem
   Deckeleintrag. Sicher, aber die Begründung ist falsch; Kommentar korrigieren
   oder die Kommandozeile prüft es.
8. **Der Lader nimmt unbekannte Tabellen auf oberster Ebene an.** `[stat]` statt
   `[state]` wird still ignoriert und zeigt sich erst als Kaskade von Befunden in
   Stufe 4 und 5. Knotenschlüssel werden geprüft (`Node.UnknownKeys`), das Dokument
   nicht.
9. **`result.Tokens` fällt auf dem Pausen- und dem Exit-Pfad weg**
   (`walk.go:137–159`, Stand `7b47bd7`). Kein echter Baustein liefert dort Tokens;
   eine Lücke im Vertrag, die noch nicht zuschlägt. Verwandt: `agent.Run` gibt
   `flow.Result{}` zurück, wenn die Anfrage scheitert (`agent.go:145`) und wenn
   die Antwort nicht zu den deklarierten Feldern passt (`agent.go:150`). Der
   Fehlerzweig des Runners schreibt seit `7b47bd7` Tokens und Modell mit, bekommt
   von `agent` aber keine. Die Tokens einer bezahlten, falschen Antwort fehlen
   also weiter im Journal.
10. Die Ablehnung „no adapter for provider claude yet“ vor dem Start gehört an die
    Kommandozeile (steht schon unter „Außerhalb dieses Plans“).

`[agent] default` wird auch für einen Flow aufgelöst, dessen Knoten ihn nie
erreichen. Das ist **kein** Befund: Stufe 7 der Spec verlangt, alle drei Stufen der
Kette zu prüfen.

## Kleinere Befunde, bewusst offen

- `flow`: `int(value)` schneidet auf 32 Bit ab (älter als diese Welle);
  `Cap.Limit` setzt voraus, dass `Params` Go-`int` hält, und sagt es nur in der
  Fehlermeldung.
- `blocks/gate`: eine dreimal gelistete Wahl wird zweimal als „listed twice“
  gemeldet. Kein Test legt fest, wie eine Wahl behandelt wird, die Präfix einer
  anderen ist (`no`/`no_go`); das Verhalten stimmt, weil `gateMatch` ein
  Trennzeichen verlangt.
- `blocks/exit`: `Check` nimmt eine leere `message` an, `gate` lehnt eine leere
  `question` ab (so vom Plan vorgegeben). `Define` und `Run` wiederholen das Lesen
  von `Texts(node)[0]`; ein Helfer entfernte nebenbei das `raw, _ :=`. Ein Test
  zählt für die Codes 0/1/3 nur die Befunde, ohne ihren Text (auch vom Plan);
  `TestExitCheckFindsEveryProblemAtOnce` legt den Text fest.
- `blocks/agent`: `sortedNames`/`sortedTypes` sind unpräfixiert in einem Paket, in
  das zwei Lanes schreiben. Kein RED-Nachweis für die drei
  Charakterisierungstests. Dass das Flow-Modell vor `[agent] default` gewinnt, wird
  hier nicht geprüft, nur in `flowcfg`. Die älteren Fehlertests prüfen mit
  `strings.Contains` auf den Kern der Meldung.
- `flowload`: `list` (`decl.go:348`) und `offer` (`discover.go:93`) sind dieselbe
  Funktion zweimal. Stufe 4 liest einen Text erneut über `flow.ReadText` und
  verwirft den Fehler; unerreichbar, weil Stufe 3 jeden Text schon geöffnet hat.
  Die Fixtures legen den Wortlaut von `expr`, `flowcfg`, `flow.Coerce` und `tmpl`
  fest, die ihn selbst auch festlegen.
- `runner`: `gate_test.go:91` prüft `Tokens == 0` auf einem Pfad, der keine Tokens
  setzen kann (vom Plan vorgegeben). `gate_test.go:176` vergleicht zwei
  `InputHash` verschiedener Besuche, die Assertion kann nicht scheitern. `Define`
  läuft auf dem Pausenpfad zweimal; der Kommentar sollte sagen, dass `Define`
  billig und rein bleiben muss.

## Betrieb

- Der Abbau der Lane-Worktrees nach Welle 1 funktionierte wie beschrieben: zuerst
  die `dmypy`-Daemons beenden, dann `ulguard worktree-remove`. Alle vier Worktrees
  gingen sauber weg. Die Zweige `ulflow/*` sind am 2026-09-13 mit `git branch -d`
  gelöscht, und die beiden unversionierten `agent-harness-graph*.html` liegen im
  Papierkorb.
- Die Lane-Worktrees wurden abgebaut, während das Abschlussreview noch lief. Die
  Fix-Runde musste deshalb auf den zusammengeführten Baum umgelenkt werden. Mit
  offenen Befunden aus einem Review lohnt es, die Lane-Worktrees stehen zu lassen,
  bis es durch ist.
- Die Orchestrator-Sitzung lief aus `C:/Users/micro/orca/workspaces/ultraloom/wentletrap`.
  Ihr Transkript und die der Subagenten liegen unter
  `~/.claude/projects/C--Users-micro-orca-workspaces-ultraloom-wentletrap/040fc8fd-…`.
  Sie endete mitten in der Fix-Runde. Das Ledger meldete die Fix-Runde als
  „unterwegs“ — ein Vorsatz, kein Ergebnis. Den Stand eines Zweigs liest man aus
  `git reflog`, nicht aus dem Ledger.
- Die Checkboxen von Task 5 im Plan sind nicht abgehakt, obwohl die Schritte 1–8
  erledigt sind.
