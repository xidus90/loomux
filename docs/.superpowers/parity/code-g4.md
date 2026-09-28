# Paritätsliste Code-Graph G4a und G4b

**Quelle:** `trailhq/Graft` @ `1e352a3` (MIT), `src/graph/traverse.ts`, `src/graph/blast.ts`, `src/grep.ts`, `src/context/skeleton.ts`, `src/context/repomap.ts`, `src/mcp/tools.ts`, `src/cli.ts`.  
**Spec:** [2026-09-22-loomux-code-g4-delta.md](../specs/2026-09-22-loomux-code-g4-delta.md), für G4b ergänzt und berichtigt durch [2026-09-23-loomux-code-g4b-delta.md](../specs/2026-09-23-loomux-code-g4b-delta.md)  
**Pläne:** [2026-09-22-loomux-code-g4a.md](../plans/2026-09-22-loomux-code-g4a.md), [2026-09-23-loomux-code-g4b.md](../plans/2026-09-23-loomux-code-g4b.md)  
**Regel:** Jede Zeile dokumentiert die Entscheidungen und Übereinstimmungen gegen die Referenz.

---

## 1. Übersicht & Architektur-Entscheidungen

Stufe G4 wurde in zwei Phasen unterteilt:
- **G4a:** Vollständige Navigationspalette, Algorithmen, Orchestrierung in `query`, 4 neue MCP-Werkzeuge und 5 neue CLI-Subcommands.
- **G4b:** Diff-Parsing (`internal/code/diff`), Git-Diff Blast Radius (`blast.Radius`), `graph blast` und MCP `graph_blast`, `check graph-fresh` und `check blast-audit`, die Art `graph` in `[verify]` und der Blast-Monitor im Post-Edit-Hook (Abschnitt 4).

### Wichtige Architektur-Festlegungen
1. **Kein doppelter Walk (`traverse` vs `blast`):**
   Grafts `traverse.ts` (edgeWalk, reach, resolveSymbol) und `blast.ts` teilen denselben Walk. Statt eines redundanten zweiten Pakets wurden `Resolve`, `EdgeWalk`, `InDegree` und `QuoteLine` direkt in `internal/code/blast` implementiert.
2. **Reine Algorithmen-Pakete (Kein I/O):**
   `skeleton`, `grep` und `repomap` führen keine eigenen Dateisystem-Operationen durch. Sie arbeiten rein auf `*model.Graph` und injizierten Lesefunktionen (`io.Reader` / `func(path string) ([]byte, error)`).
3. **Fail-Closed Privacy auf dem Cloud-Kanal:**
   Treffer in Dateien unter geschützten Bereichen (`never`-Globs laut Manifest) werden auf dem Cloud-Kanal strikt ausgeblendet:
   - `graph_trace_calls`: Zählt ausgeblendete Treffer im `Hidden`-Zähler, schützt Symbolpfade.
   - `graph_find_all`: Injizierte Lesefunktion liefert `os.ErrPermission`; Datei wird als unlesbar gezählt, kein Quelltext gelangt nach außen.
   - `graph_file_api`: Verhält sich bei geschützten Dateien exakt wie bei unbekannten Dateien (`NotFound`).
   - `graph_repo_map`: Filtert geschützte Verzeichnisse und Knoten vor der Map-Erstellung.
4. **Namenskollision `query.Stats`:**
   In `internal/code/query/build.go` existierte bereits `type Stats struct`. Die neue Metrik-Funktion heißt daher `GraphStats(root string) (StatsAnswer, error)` mit Formatter `StatsReport`.
5. **Gemeinsamer Ladehelfer `loadGraph`:**
   In `internal/code/query/load.go` zentralisiert: Frischeprüfung via `ask.EnsureFresh`, Existenzcheck (`ErrNoGraph`) und Einlesen via `store.Read`.

---

## 2. Verfügungen gegen die Referenz (Graft @ 1e352a3)

| Komponente | Graft (`1e352a3`) | loomux (G4a) | Typ | Begründung |
|---|---|---|---|---|
| **Paketgrenzen Walk** | `traverse.ts` + `blast.ts` (parallele Traversierungslogik) | `internal/code/blast` vereint `Resolve`, `EdgeWalk`, `Reach` | **Bereinigung** | Verhindert Auseinanderlaufen von Kantenauflösung und BFS-Traversierung. |
| **EdgeWalk Tiefe 1** | Direkte Kantenabfrage (`traverse.ts:edgeWalk`), Duplikate erhalten, Rekursion als Self-Loop erhalten | `(x *Index) EdgeWalk(start, dir, depth)` mit identischer Depth-1-Semantik | **Parität** | Exakte Übereinstimmung mit Grafts Kantenbehandlung. |
| **Grep Lookarounds** | TypeScript RegExp unterstützt Lookahead/Lookbehind | Go RE2 verbietet Lookaround/Backreferences; grep prüft vorab und meldet sauberen Fehler | **Sicherheit** | Schutz vor DoS und RE2-Inkompatibilitäten. |
| **Grep Zeilenlimit** | Kürzt bei 160 Zeichen (`grep.ts:133`) | Kürzt bei 160 Runen mit `…` | **Parität** | Exakt identische Darstellung für LLM-Kontextfenster. |
| **Repomap 60% Monolith** | Verfeinert Verzeichnisse mit >60% der Gesamtknoten in Subcluster | `repomap.Build` teilt bei >60% in Unterverzeichnisse auf | **Parität** | Identisches Verhalten bei großen Monorepos. |
| **MCP-Werkzeugnamen** | `graft_trace_calls`, `graft_file_api`, `graft_find_all`, `graft_repo_map` | `graph_trace_calls`, `graph_file_api`, `graph_find_all`, `graph_repo_map` | **Abweichung** | Folgt loomux-Namenskonvention (`graph_*`), analog zu `brain_*`. |
| **MCP-Scope** | Kein `scope`-Parameter (Server bindet genau 1 Repo) | Pflichtfeld `scope` für alle Werkzeuge | **Zusatz** | Notwendig für Multi-Area-Föderation in loomux. |
| **Schema-Defaults** | Keine Defaults im Schema deklariert (nur in Beschreibungen) | Keine `default`-Felder im JSON-Schema; Defaults greifen im Handler | **Parität** | Exakte Parität mit Graft `tools.ts` und Vermeidung von Client-Caching-Drift. |
| **CLI Parameter** | Flags wie `-d`, `-i`, `--json`, `--max-hits` | Unterstützt dieselben Flags; Flags können vor oder nach Positionsargument stehen | **Parität** | Robuste POSIX/Go-Kommandozeilenführung. |

---

## 3. Schnittstellen & Testabdeckung

Alle Pakete dieser Stufe erfüllen strikt die 100%-Coverage-Vorgabe pro Funktion:

- `internal/code/model/spans.go`: 100.0%
- `internal/code/blast`: 100.0%
- `internal/code/skeleton`: 100.0%
- `internal/code/grep`: 100.0%
- `internal/code/repomap`: 100.0%
- `internal/code/query` (neue Funktionen): 100.0%
- `internal/mcptools`: 100.0%
- `internal/serve/graph`: 100.0%
- `internal/cli` (neue Subcommands): 100.0% (JSON-Fehlerpfade mit `//coverage:exempt`)
- `testdata/cases/graph`: Deterministische Golden-Tests für `callers`, `skeleton`, `grep`, `map` und `stats`.

---

## 4. G4b

Spec: [2026-09-23-loomux-code-g4b-delta.md](../specs/2026-09-23-loomux-code-g4b-delta.md), das
G4-Delta dort, wo der Nachtrag nichts sagt. Graft-Stellen nach `src/blast/blast.ts`,
`src/blast/diff.ts` und `src/blast/evidence.ts` @ `1e352a3`, wie das G4-Delta sie gelesen hat.

| Komponente | Graft (`1e352a3`) | loomux (G4b) | Typ | Begründung |
|---|---|---|---|---|
| **Dateiknoten in `Radius`** | Dateiknoten nur, wenn kein Symbol getroffen wurde (Code von `blast.ts`, gegen seinen Kopfkommentar) | ebenso; gewalkt mit `Reach` **ohne** Expansion auf die Symbole der Datei | **Grenze** | Ein Dateiknoten erreicht nur seine Importeure, und Importkanten enden auf einer Repräsentanten-Datei je Paket (`resolve.importTarget`). Eine Änderung nur auf Paketebene (Import, Konstante) in einer anderen Datei des Pakets erreicht darum nichts. |
| **Dateiknoten im Monitor** | — (kein Monitor) | der Dateiknoten seedet im Monitor nie; nur Symbole mit geändertem oder fehlendem `BodyHash` | **Abweichung vom G4-Delta §3.5.2** | Aus demselben Grund: der Hinweis hinge davon ab, welche Datei eines Pakets bearbeitet wurde. Nur Paketebene geändert ⇒ Schweigen. |
| **Testsignal** | `changed \| stale \| none \| na`; Testdateien aus dem Walk | dieselben vier Werte; die Testdateien kommen **immer aus der Hülle** `Reach(…, In, All)`, `--depth` begrenzt nur die gemeldeten Treffer | **Präzisierung** | Ein Test, der über einen Helfer aufruft, erreicht den Bereich auch; sonst hinge das Signal an der Anzeigetiefe. |
| **Testregel** | je Sprache | `_test.go` ist die einzige Regel (`blast.IsTestPath`) | **Grenze** | Go-only, solange es nur den Go-Extraktor gibt; G5 bringt die Regeln der übrigen Sprachen. |
| **Testdatei als Bereich** | `na` | `na`, ebenso ein Bereich, dessen Seeds weder `function` noch `method` sind | **Parität** | Eine Testdatei hat keinen Test über sich; ein struct, interface oder type hat keinen Test, der ihn ruft (G4-Delta §3.6). |
| **inDegree im Hub-Ranking** | `changedAreas` zählt alle eingehenden Kanten, `contains` eingeschlossen | eingehende Walk-Kanten ohne `contains`, eine Definition überall (`Index.InDegree`) | **Abweichung** | G4-Delta §3.4: `contains` hebt fast jedes Symbol um genau 1 und ändert die Reihenfolge kaum; zwei Definitionen liefen auseinander. |
| **`--skip-test-callers`** | — | zählt für die rote Bedingung von `check blast-audit` nur Aufrufer außerhalb von Testdateien (`Index.InDegreeWhere`); das Preset setzt ihn **nicht** | **Zusatz, entschieden nach Messung** | E2′: auf den letzten 50 Commits änderte der Schalter in der Sicht der Lane nichts (die roten Bereiche waren `none`, dort gibt es keine Testaufrufer). Entschieden am 2026-09-23: Schwelle 5, alle Aufrufer. Der Schalter bleibt als Wahl für ein Projekt. |
| **E2′-Ergebnis** | — | auf dem Graphen von `c` rot bei N = 3: 3 von 50 Commits, N = 5: 1, N = 10: 0; mit und ohne Testaufrufer gleich. Alle Befunde waren Rauschen (Aufruf über einen Funktionswert, Methode auf einer lokalen Variable, ein Refactoring, das seine unveränderten Tests schon abdecken) | **Messung** | `docs/*/benchmarks.md`, Eintrag 2026-09-23 21:48. Die Basis ist der Graph von `c`, nicht der Elternstand: der Elternstand kennt neue Dateien nicht und erzeugte den einen Befund, den die Lane nicht hätte. |
| **Arbeitsbaum-Graph gegen Index-Diff** | — | `graph-fresh` baut den Graphen aus dem Arbeitsbaum, `blast-audit --cached` liest den Diff des Index | **Grenze** | Bei teilweise gestagten Dateien können einzelne Seeds daneben liegen: die Spans stammen aus dem Arbeitsbaum, die Hunks aus dem Index. Ein Graph des Index wäre ein zweiter Extraktorlauf über einen temporären Baum. |
| **Belege: Hunk und Span** | `hunksIn(file, span)`: Hunks innerhalb der Spanne | Hunks, die die Spanne **überlappen** (`h.To >= from && h.From <= to`) | **Absicht** | Ein Hunk, der über den Rand eines Symbols reicht, ändert es trotzdem; die Seeds kommen über `Innermost` ebenfalls aus der Überlappung, und ein Seed ohne Beleg wäre widersprüchlich. |
| **Kappungen** | `MAX_EVIDENCE = 4`, `MAX_LINES = 6` | 4 und 6 wie Graft; dazu 24 Zeilen je Hunk und 200 je Datei beim Einlesen, die Bereiche bleiben vollständig | **Parität + Zusatz** | Die Einlesekappung hält den Speicher bei großen Diffs klein; die weggelassenen Zeilen werden gezählt (`Omitted`). |
| **git-Aufruf** | — | immer `git -c core.quotePath=false diff --relative -M`, dazu `--src-prefix=a/ --dst-prefix=b/`, für den Patch `--unified=0 --inter-hunk-context=0 --no-color --no-ext-diff --no-textconv`; gestartet in der loomux-Wurzel | **Härtung** | Der Parser liest `a/`/`b/`-Köpfe; `diff.noprefix`, `diff.mnemonicPrefix`, Farbe ein externes Diff-Werkzeug aus der Konfiguration des Nutzers oder ein `textconv`-Filter aus seinen Attributen zerbrächen ihn; ein `diff.interHunkContext` verschmölze nahe Hunks über unveränderte Zeilen hinweg und machte deren Symbole zu Seeds. `--relative` macht die Pfade zu Graphpfaden, auch in einem Unterverzeichnis-Bereich. |
| **`--base`** | — | eine Basis, die mit `-` beginnt, ist ein Fehler (`--base must name a revision`); die Revision steht hinter `--end-of-options` | **Sicherheit** | `graph_blast` nimmt `base` von einem MCP-Aufrufer; eine Basis wie `--output=…` schriebe sonst Dateien. |
| **Status `U`** | — | ein nicht zusammengeführter Eintrag in `--name-status` ist ein eigener Fehler, `unresolved conflict in <pfad>` (`diff.ErrUnmerged`); andere unbekannte Buchstaben bleiben ein Parsefehler | **Absicht** | Ein ungelöster Konflikt hat keinen eindeutigen Stand, dessen Zeilen ein Radius lesen könnte. `U` entsteht nicht nur im Merge: auch ein angehaltener Cherry-Pick, Revert oder Rebase lässt Konflikte im Index (`blast --cached` sieht sie). Die Lane ist in allen vier Fällen `not-applicable`: `GraphReady` prüft `MERGE_HEAD`, `CHERRY_PICK_HEAD`, `REVERT_HEAD`, `rebase-merge` und `rebase-apply` über `git rev-parse --git-path` in einem Aufruf. |
| **`GIT_INDEX_FILE`** | — | die git-Zeiger eines umgebenden Hooks werden entfernt (`gitenv`); `GIT_INDEX_FILE` wird nur durchgereicht, wenn es absolut ist und im absoluten git-Verzeichnis des Repos der Wurzel liegt, nicht dieses Verzeichnis selbst ist und im Hauptarbeitsbaum nicht unter `worktrees/` liegt; die Kindprozesse der Lane (`graph-fresh`, `blast-audit`) bekommen es über `PlanEnv.GraphEnv` zurück, weil `child` ihre Umgebung mit `gitenv.Clean` baut | **Absicht** | Unter `git commit -a` oder `git commit <pfad>` gibt git dem Hook einen temporären Index, und nur der hält, was committet wird; ein fremder Index, der eines verknüpften Arbeitsbaums oder ein relativer Pfad bleibt draußen. Ohne die Rückgabe an die Kinder las `blast-audit` `.git/index` und die Lane war unter `git commit -a` grün (Test mit echtem Hook in `internal/cli/check_hook_test.go`). |
| **Zählungen unter `never`** | — | `Seed.InDegree` und das Testsignal zählen Aufrufer und Tests auch in verweigerten Pfaden mit; eine verweigerte geänderte Datei bleibt für das Signal im Diff (`withheld` in `blast.Radius`), ein verweigerter Test, der mit seinem Bereich geändert wurde, ergibt also `changed` | **Grenze** | Wie `Hidden`: eine Zahl verrät höchstens, dass es dort etwas gibt, nie einen Pfad. |
| **Stop-Hook mit Blast** | — | `graph` steht in der Profilvorgabe `stop`; `hooks/stop.go` schreibt den Baum über eine behaltene Indexkopie `loomux-stop-index-<pid>` im Git-Verzeichnis (`add -A`, ohne `.loomux/state`) und gibt sie nur der Graph-Lane als `GIT_INDEX_FILE`; die Prüfung nennt fehlenden Graphen, fehlendes HEAD, laufende Operation und „nothing changed against HEAD“; `check stop` baut dieselbe Kopie | **Gebaut (Stufe G4c, 2026-09-28)** | G4c-Delta §2: `blast-audit` ohne `--cached` sähe keine unversionierten Tests und fiele bei sauberem Baum auf `HEAD~1` zurück. Selbstnutzung siehe unten. |
| **Art `graph` im Edit-Scope** | — | im `ScopeEdit` plant `Plan` für `graph` keinen Job, wie für eine Lane ohne Befehl | **Absicht** | Ein Neubau des Graphen gehört nicht in einen Edit. |
| **`graph_blast` als Text** | — | derselbe Textbericht wie `graph blast` ohne `--json` | **Festlegung** | Wie die übrigen Graph-Werkzeuge (`CallersReport` & Co.); ein Modell liest Text, `--json` bleibt der CLI. |
| **`graph_blast`, Privacy** | — | eine geänderte Datei, ein Treffer oder ein Testpfad unter `never` wird gezählt (`Hidden`), weder gewalkt noch genannt; auf dem Cloud-Kanal keine Refresh-Hinweise | **Sicherheit** | Belege sind rohe Diff-Zeilen, dasselbe Risiko wie `graph_find_all`; auch ein Testpfad kann einen verborgenen Pfad nennen. |
| **`parseDepth`** | — | `"all"`/`"full"` ist die Hülle, eine Zahl wird abgerundet, was danach unter 1 liegt oder nicht zahlig ist, ist 1; `graph_blast` und `graph_trace_calls` teilen die Funktion | **Berichtigung** | Auf `master` machte `parseDepth` aus `0.5` eine `0`, einen leeren Walk, gegen §3.2 des G4-Deltas; behoben in einem eigenen `fix`-Commit. |

### Mutanten

`go run ./cmd/loomux dev mutants internal/code/blast internal/code/diff internal/code/grep` am
2026-09-23. Erster Lauf: `blast` 256 Mutanten, davon 37 nicht übersetzbar, 30 von 219 überlebten;
`diff` 125, 5, 9 von 120; `grep` 97, 10, 13 von 87. Die Tests der Commits `test(blast)`,
`test(diff)` und `test(grep)` töten 40 der 52. Zweiter Lauf: 12 überleben, alle äquivalent:

| Stelle | Mutante | Warum äquivalent |
|---|---|---|
| `blast/edgewalk.go:42` | `if depth < 1 && depth != All` → `if false` | `Reach` mit einer Tiefe unter 1 außer `All` läuft keine Ebene und gibt `nil` zurück, wie die Abkürzung. |
| `blast/edgewalk.go:42` | `depth < 1` → `depth <= 1` | Tiefe 1 ist oben schon behandelt und kommt hier nie an. |
| `blast/index.go:75` | `if n.Kind != model.KindFile` → `if true` | Der Dateiknoten stünde in `fileSymbols` seines eigenen Pfads; `EdgeWalk` nimmt ihn ohnehin als Start, und `Reach` entdoppelt Starts und meldet sie nie als Treffer. |
| `blast/radius.go:215` | `len(lines) > MaxEvidenceLines` → `>=` | Bei genau sechs Zeilen ist `lines[:6]` die ganze Liste und `More` 0. |
| `blast/reach.go:20` | `if visited[id]` → `if false` | Ein doppelter Start käme zweimal in die erste Front; sein zweiter Lauf findet nur schon besuchte Knoten. Mehrarbeit, kein anderes Ergebnis. |
| `blast/resolve.go:108` | `strings.Contains(cleanQ, ".")` → `true` | Ohne Punkt ist `LastIndex` −1 und der bloße Name die ganze Anfrage; Stufe 1 fand dazu kein Symbol, also treffen nur Dateiknoten mit diesem Namen, und die liefert `matchFiles` in Stufe 3 genauso. |
| `diff/diff.go:108` | `line == "" && err != nil` → `line == ""` | `ReadString` liefert eine leere Zeile nur zusammen mit einem Fehler. |
| `diff/diff.go:163` | `if err == io.EOF` → `if false` | Die letzte Zeile ohne Zeilenende wird verarbeitet; der nächste Lesevorgang liefert `""` mit `io.EOF` und beendet die Schleife oben. |
| `grep/grep.go:163` | `len(runes) > 160` → `>=` | Eine Zeile aus genau 160 Runen, auf 160 gekürzt, ist dieselbe Zeile. |
| `grep/grep.go:173` | `if x != nil` → `if true` | `Index.InDegree` gibt für einen `nil`-Index 0 zurück. |
| `grep/grep.go:199` | `>` → `>=` | Der Vergleich läuft nur, wenn die beiden inDegrees verschieden sind. |
| `grep/grep.go:202` | `<` → `<=` | Ebenso nur bei verschiedenen Pfaden. |

## 5. G4c: Selbstnutzung

Am 2026-09-28 in einem abgelösten Scratch-Worktree dieses Repositorys bei
`2ffa19ac` (vor dem Rebase auf `849ac5eb`), Binary `bin/loomux.exe` aus
demselben Stand, Graph gebaut. Hub:
`gitenv.Environ`, Eingangsgrad 58 (`graph callers`, Tiefe 1). Payload
`{"session_id":"…","hook_event_name":"Stop"}` über `< datei`, neue
Sitzung je Lauf (keine Basis, gegen `HEAD`).

| Runde | Hook | Ergebnis |
|---|---|---|
| Kommentarzeile in `Environ`, kein Test | `bin/loomux.exe hook stop --host claude` | **Exit 2**; stderr: `graph/go: failed [preset] 1.5s`, `graph rebuilt`, `blast audit: index against HEAD, threshold 5`, `internal/gitenv/gitenv.go [stale]: Environ in-degree 58` |
| dieselbe Änderung, dazu neuer **unversionierter** `internal/gitenv/selftest_new_test.go`, der `Environ` ruft | derselbe | **Exit 0**, nur die Notiz „no base commit for this session“; danach keine `loomux-stop-index-*` im Git-Verzeichnis |

Die zehn Messläufe mit fünf Arten (siehe `benchmarks.md`, 2026-09-28 11:58)
endeten ebenfalls alle mit Exit 2 und demselben Befund.

### Mutanten (G4c)

Am 2026-09-28 über den Endstand des Zweigs, auf die geänderten Dateien beschränkt:
`loomux dev mutants -only alive internal/child`, `-only refresh internal/code/ask`,
`-only refresh internal/code/query` und `-only gitwork.go internal/gitwork`. Über
`internal/hooks` und `internal/cli` läuft `dev mutants` nicht: Ihre Suiten brauchen länger als
die 60 s, die es einem Lauf gibt (`goTimeout`), und es verweigert die Runde, weil schon der
Grundlauf als rot gilt. Dort lief eine Handrunde über jede geänderte Bedingung in
`hooks/stopgraph.go`, `hooks/stop.go` und `cli/check.go`, je Mutante ein `-overlay`, die Tests
auf `-run 'Stop|OpenStopIndex'` bzw. `'CheckStop|CheckPrecommit|CheckGraph'` begrenzt.

Erster Lauf: `child` 8 Mutanten, 3 überlebten; `ask` 69, davon 23 nicht übersetzbar, 3 von 46;
`query` 25, 6, keiner von 19; `gitwork` 125, 27, 5 von 98; Handrunde 19, davon 2 ungültig, einer
von 17. Drei neue Tests töten die echten: `TestAliveForgetsAnEndedProcessSomeoneStillHolds`
(ein beendeter Prozess, den ein offenes Handle hält), `TestEnsureFreshBreaksAStaleLock` mit einem
lebenden Halter (vorher brach die tote PID 1 den Lock, das Alter wurde nie geprüft) und
`TestStopWithoutABaseAndNothingNewRunsNothing`. Es überleben:

| Stelle | Mutante | Warum sie bleibt |
|---|---|---|
| `child/alive.go:9` | `pid > 0` → `pid >= 0` | Unter Windows gleichwertig: PID 0 lässt sich nicht öffnen. Unter POSIX erreichte ein Signal an 0 die Prozessgruppe; dort tötet `TestAliveRefusesNumbersThatAreNoProcess` im Linux-Lauf von `ci.yml`. |
| `child/alive_other.go:18` | `err == nil` → `err != nil` | Die Datei wird unter Windows nicht gebaut; der Linux-Lauf deckt sie. |
| `ask/refresh.go:205` | `>= lockStale` → `> lockStale` | Die Uhr trifft die Stunde nicht auf die Nanosekunde. |
| `ask/refresh.go:157` | `err == nil \|\| notice == nil` → `notice == nil` | Stand auf master; nicht Teil der Stufe. |
| `gitwork/gitwork.go:203` | `if err != nil` → `if false` (nach `--absolute-git-dir`) | Außerhalb eines Repositorys scheitert dann `add -A` mit Git's eigenem Fehler; Ergebnis, Pfad und Aufräumen sind dieselben. |
| `gitwork/gitwork.go:277, 373, 415` (vier) | Fehlerzweige in `LocalBranches` und `ChangedFiles`, eine Grenze in `parseStatus` | Stand auf master; nicht Teil der Stufe. |
