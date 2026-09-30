# Abweichungsliste Stufe 2a

**Quelle:** ultraloom am Tag `loomux-1a-source` (`9d01a60`): `ultraloom check`
mit `checks.py`, `config.py`, `process.py`, `cli.py` und
`hooks/coverage-check.py`, die Werkzeuge beantwortet von `faketool`. Die
Einträge 17 und 18 vergleichen dagegen mit dem post-edit von loomux selbst auf
`master` `86fa192` (`internal/hooks/post_edit.go`), denn diese Lanes gab es in
ultraloom nicht.
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor Stufe 2a als
fertig gilt. „Alt" ist die Quelle oben, „Neu" ist `loomux check` bzw. der
post-edit-Hook über `[verify]` und die Presets.
**Stand:** 2026-09-30. 53 Fälle über 13 Welten; 38 bestehen, 15 stehen als
genehmigte Abweichung in `approved2a` (`internal/cli/cases_2a_test.go`):
Eintrag 1 zweimal, Eintrag 10 sechsmal, Eintrag 16 fünfmal, Eintrag 20
viermal (zwei davon, die beiden `-all`-Fälle, stehen auch unter Eintrag 10).
Bis 2026-09-19 waren es 13; Eintrag 20 kam am 2026-09-30 dazu. `mixed-config-all`
und `no-marker-all` standen zwischenzeitlich dort und bestehen seit `7fe061d`
bzw. `749acd8` ohne Eintrag.

Die Spalte „Fall" nennt nur Fälle aus `testdata/cases/2a/`, die in
`approved2a` stehen. Wo kein aufgezeichneter Fall die Abweichung zeigt, steht
der Unit-Test, der das neue Verhalten festhält.

| Nr. | Fall | Alt | Neu | Begründung | Freigabe |
|---|---|---|---|---|---|
| 1 | `check/cpp-all`, `check/cpp-types` | `check types` in einem CMake-Projekt (`cpp-types`): stdout leer, auf stderr ein Traceback `KeyError: 'CMakeLists.txt'` (der Marker fehlt in `_LANGUAGE_NAMES`), Exit 1. `check all` (`cpp-all`) läuft durch: `lint`, `test` und `coverage` ok, nur `types: failed [error]` mit der Zeile `KeyError: 'CMakeLists.txt'`, Exit 1 | `types/cpp` ist `cmake --build build --parallel` (`internal/verify/presets.toml`, `[stack.cpp.types]`), der Compiler als Typprüfer; grün mit Exit 0 | Spec-Zeile 1, berichtigt durch Plan-Nachtrag 2: nicht `not-applicable`, sondern dieselbe Zeile, die post-edit schon fuhr. Ein Altfehler wird behoben, nicht übernommen. `cpp-types` vergleicht nur den Exit (Klasse `message`), `cpp-all` die Urteile je Art (`lanes`) | |
| 2 | kein Fall — Unit-Test `TestFoldVerifyMovesTheKindsToTheProjectAndSwitchesThePresetsOff`, `TestParseConfigRefuses`, `TestCheckRefusesMalformedCalls` | `[verify.coverage].threshold` und `--threshold` wurden gelesen, aber nirgends durchgesetzt | beide entfallen: `threshold` in einer Lane-Tabelle ist ein Ladefehler (`has unknown key`), `--threshold` ein Usage-Fehler mit Exit 2. Die Schwelle steht im Tor selbst: `--floor` bei `gocover`, `fail_under` in `pyproject.toml`, `--fail-under-line` bei gcovr | Spec-Zeile 2 und Tabelle „Wegfall aus dem alten Schema". Der Import der Fälle wirft `threshold` weg, weil die alte Kette es ohnehin nicht wirken ließ. Kein Test nennt `--threshold`; `TestCheckRefusesMalformedCalls` zeigt nur, dass unbekannte Flags allgemein abgewiesen werden | |
| 3 | kein Fall — Unit-Test `TestParseConfigRefuses` | unbekannte Schlüssel unter `[verify]` wurden stillschweigend ignoriert | Ladefehler mit Datei und Schlüssel, etwa `cfg.toml: [verify] has unknown key "parallelism"`, auf jeder Ebene (`[verify]`, `[verify.<stack>]`, `[verify.<stack>.<art>]`) | Spec-Zeile 3: ein Tippfehler in der Konfiguration darf kein stilles Grün ergeben | |
| 4 | kein Fall — Unit-Test `TestExpandProfile` | der pre-commit-Hook rief `check all` statt eines Profils | das Tor ruft `loomux check precommit` (`ci/gate.sh`, Zeile 8, seit `25db0b2`); `precommit` ist eingebaut, heißt die vier Arten und ist überschreibbar | Spec-Zeile 4: der Hook soll das Profil fahren, das der Wirt für den Commit festlegt, nicht alles | |
| 5 | kein Fall — Unit-Test `TestPlanCoverageMeasuresItselfWhenTestIsSwitchedOff`, `TestSettleStillLinksATestLaneWithoutTests`, `TestRunInheritsAPredecessorThatCouldNotRun` | war `test` `unavailable` (keine Tests), wurde `coverage` `blocked` und damit rot | Erbregel: `coverage` erbt den Zustand eines Vorgängers, der nicht laufen konnte, nur bei `unavailable` (keine Tests); ist `test` abgeschaltet (`test = false`, also `not-applicable`), zählt der Vorgänger als nicht angefordert, und die Coverage-Lane misst mit ihrem `measure` selbst. Ohne `measure` ist sie rot mit „`test` did not run and there is no measure step" | Spec-Zeile 5, verfeinert in Task 14 (`7fe061d`): wer `test` abschaltet, sagt nichts darüber, ob es Tests gibt. `mixed-config-all` und `mixed-config-coverage` bestehen damit ohne Eintrag | |
| 6 | kein Fall — Unit-Test `TestRunBlocksTheSuccessorOfARedLane`, `TestParseConfigRefuses` (Zyklus), `TestCheckReportsAPlanError`, `TestCheckReportsLoadErrors` | `check <art>` umging Reihenfolge, Blockieren und Zyklusprüfung; Fehler endeten im Traceback | jeder Aufruf, ob Profil, Art oder Liste von Arten, läuft durch denselben Planer: `after`, Blockieren nach Rot und Zyklusprüfung beim Laden gelten überall; Lade- und Planfehler sind eine Zeile auf stderr mit Exit 1 | Spec-Zeile 6. Die Welten `config-after-cycle`, `config-empty-list` und `config-profile-unknown` wurden nur mit `check all` aufgezeichnet, wo alt und neu übereinstimmen; ihre Fälle bestehen | |
| 7 | kein Fall, kein Unit-Test: der Bezeichner kommt in `cmd/` und `internal/` nicht vor | eine falsche `ULTRALOOM_CLI_PATH` (bzw. `[agent].cli_path`) brach jedes `load_config`, also auch jede Prüfung | entfällt; loomux liest weder die Variable noch den Schlüssel | Spec-Zeile 7 und Tabelle „Wegfall aus dem alten Schema": nicht Teil der Prüfkette | |
| 8 | kein Fall — Unit-Test `TestParseConfigDefaults`, `TestHookPostToolUsePassesTheBudgetOn`, `TestRunStartsNothingOnceTheBudgetIsSpent`, `TestRunGivesTheRestWhenThereIsNoTimeout`, `TestRedDependsOnTheScope` | `timeout` 600 s je Befehl gegen eine Host-Frist von 300 s: ein Befehl konnte länger laufen, als der Host wartete | `timeout` bleibt 600 s je Befehl als Vorgabe, ohne Obergrenze beim Laden. Das Budget gehört dem Scope: post-edit hat 50 s (`hooks.DefaultBudget`, `--budget`), unter der Hook-Frist von 60 s; `loomux check` hat keins. Ist das Budget aufgebraucht, startet keine Lane mehr, und die offenen sind `budget`, kein Befund | Spec-Zeile 8 (aus `docs/stufe-2-stop-notes`). Die Regel für den `stop`-Hook folgt in Stufe 2c | |
| 9 | kein Fall, kein Unit-Test: der `coverage`-Fall der Welt `cpp` besteht, weil die Klasse `lanes` Urteile vergleicht, nicht die Befehlszeile; kein Test hält `--fail-under-line` fest | das C++-Preset `gcovr --cobertura coverage.xml` schrieb nur einen Bericht und war immer grün | `gcovr --root . --object-directory build --fail-under-line 100 --txt` (`internal/verify/presets.toml`, `[stack.cpp.coverage]`): unter 100 % Zeilen ist die Lane rot | Spec-Zeile 9: ein Coverage-Schritt, der nie rot wird, ist kein Tor | |
| 10 | `check/godot-no-coverage-all`, `check/godot-no-coverage-coverage`, `check/godot-no-coverage-types`, `check/godot-unready-all`, `check/godot-unready-coverage`, `check/godot-unready-types` | `unavailable` war rot: GDScript meldete `types: failed [unavailable]` und `coverage: failed [unavailable]` („GDScript has no types tool — a known limitation, not a passed check"), Exit 1 | zwei Regeln. (a) Eine Art, die für einen erkannten Stack nicht definiert ist (GDScript `types` und `coverage`), ist `not-applicable`: neutral, Exit 0. (b) `unavailable` ist neutral, solange die Art in einem anderen Stack lief. Hat eine angeforderte Art nirgends eine Lane, die lief, bleibt sie rot: „nothing to check for `<art>`", Exit 1. In `godot-unready-all` blieb der Exit bis 2026-09-30 bei 1, weil `test/gdscript` `unready` und damit rot war; seit Eintrag 20 hat GDScript auch keine `test`-Lane mehr, und der Fall endet mit 0 | Spec-Zeile 10, erweitert um die Regel (a). Eine bekannte Lücke eines Stacks ist kein Befund über den Code; ohne jede Lane aber wäre Grün eine Lüge. `no-marker-all` (kein Stack) besteht seit `749acd8` ohne Eintrag. Unit-Tests: `TestCheckVerdictPerKind`, `TestCheckFailsAKindWithNothingToRun`, `TestKindVerdictsReadsTheNothingToCheckNoteAsRed` | |
| 11 | kein Fall — Unit-Test `TestWriteCheckFormatsEveryState`, `TestWriteCheckVerboseAndInProcess`, `TestCheckVerboseShowsAGreenLanesOutput` | eine Zeile je Art, grüne Arten mit ihrer ganzen Ausgabe | eine Zeile je Lane (`<art>/<stack>[@<bereich>]`); grüne Lanes zeigen nur ihre Kopfzeile, `-v` zeigt alles; mehrere Befehle je Block `$ <argv>[ (failed)]` | Spec-Zeile 11: eine Lane je Stack und Art. Die Fallsuite vergleicht mit der Klasse `lanes` nur Urteile je Art und Exit, der Wortlaut fällt dort nicht auf | |
| 12 | kein Fall — Unit-Test `TestParseConfigRefuses`, `TestFoldVerifyMovesTheKindsToTheProjectAndSwitchesThePresetsOff` | `[verify].<art>`, `[verify.<art>]`, `[verify].tests`, `[verify].godot_import`, `[verify.coverage].report`, `[exec].prefix`, `.ultraloom/checks/<art>.*`, `ULTRALOOM_ALONGSIDE` | die Tabelle „Wegfall aus dem alten Schema" der Spec: `[verify.<stack>].<art>` bzw. `[verify.project].<art>`, `[verify.gdscript] import_check`, `[verify.<stack>].coverage`; die alte Form ist ein Ladefehler mit Hinweis (`[verify].types is the old form; name the stack: [verify.<stack>].types`), `import_check` außerhalb von `gdscript` ein unbekannter Schlüssel | Spec-Zeile 12. Die `config-*`-Welten bestehen, weil `dev import-cases` ihre alte Konfiguration faltet | |
| 13 | kein Fall — Unit-Test `TestCheckGocoverPerFunctionAndFloor`, `TestCheckGocoverNeedsAGoMod`, `TestCheckRunsTheGoPresetsInTheirOrder` | `hooks/coverage-check.py` als `[verify.coverage].report`, ein Skript für alle Sprachen | `loomux check gocover --profile <p> [--floor N]` im Go-Preset (100 % je Funktion, `//coverage:exempt` wie bisher), Coverage-Presets je Stack für Python und C++ | Spec-Zeile 13. Die Go-Seite zeigt Eintrag 16 (`check/go-only-coverage`) | |
| 14 | kein Fall — Unit-Test `TestDevCovergateIsGone` | `loomux dev covergate --profile p` | entfällt; `loomux check gocover --profile p` tut dieselbe Arbeit, `ci/gate.sh` ruft es über `check precommit` | Spec-Zeile 14 | |
| 15 | kein Fall — Unit-Test `TestPostEditRunsTheEditProfileOfTheFilesStack`, `TestPostToolUseRunsTheGoLaneThroughRealTools`, `TestRunStatusAllStacksAndNoLegacy` | post-edit startete jeden Befehl über `cmd /c` mit fest verdrahteten Lanes in `internal/hooks/post_edit.go` | argv ohne Shell, über `internal/child`; die Lanes kommen aus dem Profil `edit` der Presets und aus `[verify]`, mit `on_file`, wo vorhanden | Spec-Zeile 15. Die drei 1a-Fälle unter `testdata/cases/1a/hook-post-tool-use/` bestehen unverändert: `go test ./internal/cli -run TestRecordedCasesOfStage1a -count=1` ergab am 2026-09-19 `ok`; die 1a-Suite kennt keine Genehmigungsliste | |
| 16 | `check/go-only-all`, `check/go-only-coverage`, `check/go-only-lint`, `check/go-only-test`, `check/go-only-types` | kein Go-Preset. `check all` (`go-only-all`): jede Art `failed [unavailable]` mit „could not tell what kind of project … is; set [verify].<art> …", Exit 1. Die Einzelarten (`go-only-lint`, `-test`, `-coverage`, `-types`): stdout leer, dieselbe Meldung auf stderr, Exit 1; diese Fälle vergleichen nur den Exit (Klasse `message`) | die Go-Lanes laufen: `lint` (`go vet`, `check gofmt`), `test` (`go test`), `coverage` (`check gocover` auf dem geschriebenen Profil), alle grün mit Exit 0. `types` hat bei Go keine Lane und ist `not-applicable`, Exit 0 (Regel 10 a) | Spec-Zeile 16, gruppiert: die alte Kette wählte genau ein Preset aus einem Marker, und Go hatte keins | |
| 17 | kein Fall — Unit-Test `TestRunStatusAllStacksAndNoLegacy` | post-edit rief `clang-format -i <datei>` und schrieb die Datei des Agenten um | das C++-Preset ruft `clang-format --dry-run --Werror {file}` (`[stack.cpp.lint] on_file`): es prüft, statt umzuschreiben; eine Abweichung ist ein roter Befund | Plan-Nachtrag 3: ein Hook, der nach dem Edit die Datei ändert, schreibt hinter dem Rücken des Agenten. Der Test verlangt die neue Zeile und dass `clang-format -i` nicht mehr vorkommt | |
| 18 | kein Fall — Unit-Test `TestPostEditRunsATypeScriptLaneInItsArea`, `TestPlanEditScopeRunsOnFileInTheFilesArea`, `TestRunStatusAllStacksAndNoLegacy` | post-edit rief in einem Bereich die Workspace-Skripte des Wirts: `npm --prefix <bereich> run typecheck`, `run lint`, `run check` | die Presets rufen die Werkzeuge direkt: `npx tsc --noEmit`, `npx eslint`, `npx svelte-check`, `npx vue-tsc --noEmit`; die JS/TS-Presets aus `checks.py` ebenso `npx eslint`, `npx tsc --noEmit`, `npx vitest run` statt der nackten Namen. Ein Wirt mit eigenem Skript schreibt es in `[verify.typescript]` | Plan-Nachtrag 6: ein Preset kann nicht wissen, welche Skripte ein Wirt hat; mit `LookPath` vor jedem Start wäre ein nicht global installiertes Werkzeug sonst `missing-tool`. Die `node-*`-Fälle bestehen, weil `lanes` Urteile vergleicht; `testdata/cases/2a-extra-answers.json` liefert die Antworten für die `npx`-Zeilen | |
| 19 | kein Fall — Unit-Test `TestRunStatusAllStacksAndNoLegacy` (verlangt, dass `dmypy` nicht mehr vorkommt) | post-edit wählte für Python ohne `pyright`, aber mit `uv` auf dem `PATH`, `dmypy run -- --no-error-summary --no-pretty` (`internal/hooks/post_edit.go` vor `6e68ed6`, Zeile 359): den mypy-Daemon, warm nach dem ersten Edit | das Preset ruft `uv run mypy --no-error-summary --no-pretty` (`[stack.python.types]`), bei jedem Edit kalt; `pyright` bleibt die Variante. Wer den Daemon will, schreibt `[verify.python] types = "uv run dmypy run -- --no-error-summary --no-pretty"`. Unter Windows stirbt ein Daemon, den eine Lane startet, aber mit ihr: `internal/child` legt jeden Prozessbaum in ein Job-Objekt mit `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` (`child_windows.go`, Zeile 44) und startet ohne `CREATE_BREAKAWAY_FROM_JOB`; der Daemon wird also jedes Mal neu gestartet und ist nie warm | Ein Preset gilt für `check` und post-edit gleich und darf keinen Prozess zurücklassen, den niemand beendet; `dmypy` braucht außerdem eine Statusdatei, die ihren Daemon überleben kann. Die Kosten (ein kalter mypy-Lauf je Python-Edit) sind nicht gemessen | |
| 20 | `check/godot-no-coverage-test`, `check/godot-unready-test`, `check/godot-no-coverage-all`, `check/godot-unready-all` | das GDScript-Preset fuhr als `test` `godot --headless --quit`: in der importierten Welt `test: ok [preset]`, Exit 0 (`godot-no-coverage-test`); in der nicht importierten `test: failed [unready]`, Exit 1 (`godot-unready-test`, `godot-unready-all`) | GDScript hat kein `test`-Preset (`internal/verify/presets.toml`). Ohne `[verify.gdscript.test]` ist `test/gdscript` `not-applicable` („no command"): neutral, Exit 0, und `import_check` wird gar nicht gefragt, weil es keine Lane gibt, für die der Import zählte. Nennt ein Projekt einen Befehl, läuft die Lane, und `import_check` gilt wie bisher (`unready`, rot) | Nutzerentscheidung vom 2026-09-30. `godot` liegt unter genau diesem Namen selten auf dem PATH (ein Download heißt `Godot_v4.7.1-stable_mono_win64.exe`); die Lane war dann `missing-tool` und hielt Stop-Hook und `loomux check` rot. Und der Befehl prüft nichts: gemessen mit Godot 4.7.1 endet er mit 0, auch wenn ein Skript nicht parst. Die Aufzeichnungen bleiben, wie sie sind; die `godot`-Antwort in `faketool.json` wird nur nicht mehr abgerufen. Unit-Tests: `TestPlanHasNoGodotTestLaneUntilAProjectNamesOne`, `TestTheGdscriptPresetHasOnlyALintLane`, `TestPlanMarksGodotUnready`, `TestPlanRunsTheGodotBinaryAProjectNames` | Nutzer, 2026-09-30 |

## Was die Fälle der Stufe 2a decken

`testdata/cases/2a-source/` hält die Aufzeichnungen von `ultraloom check` gegen
`faketool` (Beweis, nie nachbearbeitet), `testdata/cases/2a/` die Übersetzung,
an der loomux gemessen wird: `ultraloom check ` wird `loomux check `, die alte
`[verify]` wird beim Import gefaltet, und `2a-extra-answers.json` ergänzt die
Antworten für Befehle, die nur loomux startet. 53 Fälle über 13 Welten
(`config-after-cycle`, `config-empty-list`, `config-profile-unknown`, `cpp`,
`go-only`, `godot-no-coverage`, `godot-unready`, `mixed-config`, `no-marker`,
`node`, `python-green`, `python-red-lint`, `python-red-test`), gefahren von
`internal/cli/cases_2a_test.go`. Die Klasse `lanes` vergleicht Urteil je Art
und Exit-Code, nicht die Befehlsausgabe; Fälle mit leerem stdout (der alte
Traceback, die alten Fehlerzeilen auf stderr) vergleichen mit `message` nur den
Exit. Ein Fall in `approved2a` muss scheitern, jeder andere bestehen; ein
genehmigter Fall, der besteht, lässt die Suite rot werden. Angepasst wurde
keine Aufzeichnung.

## Mutationsrunde

Gelaufen am 2026-09-19 mit `bin/loomux.exe dev mutants --workers 8 <paket>`
(Task 18), gebaut aus und gefahren gegen Commit `e6b596a`, 8 Arbeiter auf 16
Kernen, Go 1.27.0 unter Windows, Protokolle unter `$TEMP/mutants-2a/`. Ein
Zeitüberlauf zählt als getötet. Die Schätzung vorab (Schritt 1): ein Lauf von
`go test ./internal/child/ -count=1` dauert 12,3 s, von `internal/verify` und
`internal/verify/gocover` zusammen 2,1 s. `internal/child` hat 59 Mutanten,
geschätzt also rund 1,5 min bei 8 Arbeitern, weit unter der Grenze von 30 min:
die Runde über `child` lief vollständig, über alle vier Familien. Gemessen:
`verify` und `gocover` zusammen 3 min 9 s, `child` 1 min 57 s.

| Paket | Mutanten | getötet | überlebt | nicht kompiliert | kein Mutant |
|---|---:|---:|---:|---:|---:|
| internal/verify | 760 | 570 | 31 | 159 | 0 |
| internal/verify/gocover | 59 | 51 | 6 | 2 | 0 |
| internal/child | 59 | 35 | 10 | 14 | 0 |

Zusammen 878 Mutanten, 656 getötet, 47 überlebt, 175 keine Mutanten.

Die Nachprüfung ist wie in Stufe 1b-1 eine vollständige zweite Runde je Paket
gegen `e6b596a` mit den neuen Tests (der Arbeitsbaum vor der Berichtigung von
`TestALaneThatIsNotThreadedRunsOneCommandAtATime`, sonst gleich `1855389`),
Protokolle unter
`$TEMP/mutants-2a/recheck/`: `verify` 10 Überlebende, `gocover` 0, `child` 4.
Kein Mutant ist neu hinzugekommen: jeder Überlebende der zweiten Runde hat
schon die erste überlebt.
Neun der zehn in `verify` und die vier in `child` sind unten begründet; der
zehnte, `run.go:168`, überlebte die zweite Runde, weil die erste Fassung seines
Tests von der Startreihenfolge zweier Goroutinen abhing. Der Test ist
berichtigt (jeder Befehl wartet auf den anderen), der Mutant fällt seitdem 20
von 20 Mal, und `dev mutants --only run.go --family a1 ./internal/verify`
meldet ihn `killed`, 0 Überlebende.

Ein Urteil unterscheidet sich zwischen den Runden: `run.go:232`, a3,
`(state == StateOK || state != StateBudget)` war in der ersten Runde „nicht
kompiliert", in der zweiten `killed`. Er kompiliert; gegen die Suite ohne die
neuen Tests (`go test -overlay`, `mutation_test.go` ausgeblendet) scheitert er
an den alten Tests. Die erste Runde hat ihn also falsch eingeordnet. Die
wahrscheinliche Ursache ist ein Fehler bei der Durchführung: während der
ersten Runde stand ein Entwurf von `internal/verify/mutation_test.go` etwa
30 s lang im Baum und wurde dann herausgenommen. Der Entwurf hielt nur die
vier ersten Tests (`cleanCover`, `merge`, `editArea`) und rief `Run` nicht;
die endgültige Datei ruft es. Welche Mutanten in dem Fenster liefen, lässt sich
aus dem Protokoll nicht ablesen: es druckt in Erzeugungsreihenfolge, als
Mutant 498 stand, die Arbeiter laufen aber voraus, sobald ein Platz frei ist.
Ein `go test`, das die Datei noch im Verzeichnis fand und beim Übersetzen nicht
mehr, meldet einen Build-Fehler -- genau das Urteil „nicht kompiliert". Die
zweite Runde und die Schlussrunde unten liefen gegen einen ruhenden Baum;
`--only run.go --family a3` meldet alle vier Mutanten der Zeile `killed`.

Schlussrunde über `internal/verify` nach der Berichtigung von `plan.go:174`
(siehe unten): `bin/loomux.exe dev mutants --workers 8 ./internal/verify`
gegen Commit `670cc4b`, Protokoll unter `$TEMP/mutants-2a/final/`.
3 min 4 s, 764 Mutanten (vier mehr als vorher, aus der neuen Zeile
`if best == ""`, alle vier `killed`), 597 getötet, 158 nicht kompiliert,
9 überlebt: die acht unten begründeten und `plan.go:174` a3, der am
berichtigten Code gleichwertig ist. Nach der
Berichtigung stehen `plan.go:205`, `:227`, `:229`, `:274` und `:277` drei
Zeilen tiefer. Kein Mutant überlebt, der nicht begründet ist.

### Erledigte Überlebende

34 Überlebende haben einen nachgereichten Test, der sie tötet, einer deckte
einen echten Fehler auf (unten), 8 haben eine
Begründung, warum der Code den Unterschied nicht sehen kann, und 4 sind unter
Windows nicht zu beurteilen (`child_other.go`, letzte Zeile der Tabelle: drei
fielen unter Linux an vorhandenen Tests, der vierte kompiliert nicht). Die
neuen Tests stehen in `internal/verify/mutation_test.go` (16),
`internal/verify/gocover/mutation_test.go` (5), `internal/child/child_test.go`
(1) und `internal/child/child_windows_test.go` (1, dazu die geschärfte
`failStart` und die geschärfte `TestRunAbandonsOutputAGrandchildHolds`). Ein
Überlebender war ein echter Fehler: `plan.go:174` (a3) wählte nie einen
Bereich aus einem Zeichen, weil `best` mit `.` (Länge 1) begann; behoben in
`670cc4b` („fix(verify): let a one-letter area own the files under it"). Sonst
ist kein Produktionscode geändert.

| Datei:Zeile | Familie | war -> jetzt | Erledigung |
|---|---|---|---|
| internal/verify/cover.go:58 | a1 | `!own` -> `true` | Test `TestCleanCoverRemovesTheOwnFreshFilesOfAGreenRun`, danach `killed` |
| internal/verify/cover.go:60 | a3 | `< staleAfter` -> `<=` | Test `TestCleanCoverRemovesAFileExactlyADayOld`, danach `killed` |
| internal/verify/effective.go:64 | a2 | `detected && len(...) > 0` -> `len(...) > 0` | Kein Unterschied unter erreichbaren Eingaben: `Resolve` bekommt seine Fakten nur aus `detect.Detect` (`internal/cli/check.go:177`, `internal/hooks/post_edit.go:73`, `internal/hooks/status.go:95`), und dort hat `Areas` nur Schlüssel für gefundene Stacks (`detect.go:64-66`); `project` nennt kein Signal. Ein nicht erkannter Stack hat also keine Bereiche, die der Mutant übernehmen könnte |
| internal/verify/effective.go:106 | a1 | `o.Lane.After == ""` -> `true` | Kein Unterschied: `Replace` setzen nur die Formen `false`, String und Liste von `parseLane` (`schema.go:243-249`), und keine davon setzt `After`; der Wert ist dort immer leer |
| internal/verify/effective.go:114, :117, :120, :123, :126 | a1 | `o.Set[...]` -> `true` | Test `TestATableOverrideKeepsEveryKeyItDoesNotName` (fünf Mutanten), danach `killed` |
| internal/verify/plan.go:174 | a2 | drei Klauseln -> nur `HasPrefix` | Test `TestTheDeepestAreaWinsWhateverTheOrder`, danach `killed` |
| internal/verify/plan.go:174 | a3 | `len(a) > len(best)` -> `>=` | Echter Fehler, kein gleichwertiger Mutant: `best` begann mit `"."`, also Länge 1, und `len(a) > len(best)` ließ einen Bereich aus einem Zeichen nie gewinnen -- `editArea([]string{"a", "."}, "a/x.go")` gab `.` zurück, `ab` ging. Der Mutant verhielt sich richtig. `best` beginnt jetzt leer und fällt am Ende auf `.` zurück (`670cc4b`); Test `TestPlanEditScopeLetsAOneLetterAreaOwnItsFiles`, der am alten Code scheitert. Am berichtigten Code ist derselbe Mutant gleichwertig und überlebt die Schlussrunde: `best` beginnt leer, `.` ist ausgenommen, und zwei Bereiche gleicher Länge, die beide Präfix von `file` samt `/` sind, sind gleich und ergeben dasselbe `best` |
| internal/verify/plan.go:205 | a2 | `!r.Defined \|\| len(cmds) == 0` -> `len(cmds) == 0` | Kein Unterschied, solange das eingebettete `presets.toml` keine Lane mit `= false` hat -- und nur deshalb: `Defined` ist falsch, wenn die Lane aus ist oder weder `commands` noch `on_file` hat; eine ausgeschaltete Lane entsteht nur aus `= false` als `Lane{Off: true}` ohne Befehle (`schema.go:243`, die Tabellenform kennt kein `off`). Schaltete ein Preset eine Lane mit `= false` aus und gäbe die Konfiguration ihr per Tabelle `commands`, bliebe `Off` stehen (`merge` übernimmt nur die genannten Schlüssel), die Lane hätte Befehle, und der Mutant führte sie aus, wo der echte Code `not-applicable` meldet |
| internal/verify/plan.go:214 | a3 | `len(patterns) > 0` -> `>= 0` | Test `TestAStackWithoutTestPatternsIsNotSearchedForTests`, danach `killed` |
| internal/verify/plan.go:227 | a1 | `req.Scope == ScopeEdit` -> `true` | Kein Unterschied: im Prüfumfang ist `File` leer, und `{file}` kommt in keinem Befehl vor, der dort läuft -- `commands` weist es außerhalb von `on_file` ab (`schema.go:348`), für `commands`, `measuring` und `measure`, auch in den Presets |
| internal/verify/plan.go:229 | a1 | `area != "."` -> `true` | Kein Unterschied: für `.` entfernt der Mutant ein Präfix `./`, und `Request.File` ist relativ mit Schrägstrichen (`plan.go:24`), gebaut aus `filepath.Rel` (`post_edit.go:143`), das nie mit `./` beginnt |
| internal/verify/plan.go:274 | a1, a2 (`job.Pre != ""`) | ganze Bedingung -> `false`, bzw. -> `job.Pre != ""` | Test `TestAMeasureWithoutAnAfterIsNeverRun` (zwei Mutanten), danach `killed` |
| internal/verify/plan.go:274 | a2 (`l.after == ""`) | ganze Bedingung -> `l.after == ""` | Kein Unterschied: jeder Ausgang von `planJob`, der `Pre` setzt, gibt `link{}` mit leerem `after` zurück; ein Job mit `Pre` geht also auch über die verbleibende Klausel |
| internal/verify/plan.go:277 | a1 | `slices.Contains(req.Kinds, l.after)` -> `true` | Kein Unterschied: die Schleife sucht einen Job der Art `l.after`, und `Plan` legt nur Jobs der angeforderten Arten an; ist `after` nicht angefordert, findet sie keinen und fällt durch wie der echte Code |
| internal/verify/presets.go:204 | a1 | `!slices.Contains(Kinds(), key)` -> `false` | Test `TestAVariantRefusesAKeyThatIsNoKindEvenAsACommand`, danach `killed`: der alte Fall `style = 1` scheiterte auch am Mutanten, nur an einer späteren Prüfung |
| internal/verify/run.go:144 | a1 | `len(job.Measure) > 0` -> `false` | Test `TestAMeasureWhoseToolIsMissingStartsNothing`, danach `killed` |
| internal/verify/run.go:156 | a1 | `s.state != StateBudget` -> `false` | Test `TestAMeasureThatTimedOutFailsTheLane`, danach `killed` |
| internal/verify/run.go:168 | a1 | `job.Threaded` -> `true` | Test `TestALaneThatIsNotThreadedRunsOneCommandAtATime`, danach `killed` (siehe oben: erst in der berichtigten Fassung) |
| internal/verify/run.go:191 | a3 | `rest <= 0` -> `< 0` | Test `TestABudgetSpentToTheNanosecondStartsNothing`, danach `killed`: der Mutant startet bei genau verbrauchtem Budget einen Befehl mit Timeout 0, also ohne Frist |
| internal/verify/run.go:194 | a3 | `rest < timeout` -> `<=` | Test `TestATimeoutAsLongAsTheRestIsTheLanesOwn`, danach `killed` |
| internal/verify/run.go:230 | a2 | zwei Klauseln -> `s.state == StateBudget` | Test `TestABudgetAfterAFindingDoesNotHideIt`, danach `killed` |
| internal/verify/schema.go:139, :145 | a3 | `n < 1` -> `n <= 1` | Test `TestTheSmallestCapAndTimeoutAreAccepted` (zwei Mutanten), danach `killed` |
| internal/verify/schema.go:310 | a2 | `!ok \|\| !Contains(...)` -> `!Contains(...)` | Kein Unterschied: ist der Wert kein String, ist `s` leer, und keine Art heißt `""`; die verbleibende Klausel weist ab, mit demselben Text, denn er nennt `value` |
| internal/verify/show.go:55 | a1, a3 | `len(l.Commands) > 0` -> `true`, `>= 0` | Test `TestALaneWithOnlyAnOnFileFormShowsNoCommands` (zwei Mutanten), danach `killed` |
| internal/verify/gocover/gocover.go:30 | a2 | zwei Klauseln -> nur `total:` | Test `TestParseSkipsABlankLine`, danach `killed` |
| internal/verify/gocover/gocover.go:34 | a1 | `len(fields) != 3` -> `false` | Test `TestParseRefusesALineWithAFieldTooMany`, danach `killed` |
| internal/verify/gocover/gocover.go:38 | a1 | `len(location) < 2` -> `false` | Test `TestParseRefusesALocationThatIsOnlyANumber`, danach `killed` |
| internal/verify/gocover/gocover.go:83 | a2, a3 (`>`) | ohne `line < 2`, `line-2 > len(rows)` | Test `TestALineOutsideTheFileIsNeverExempt` (zwei Mutanten), danach `killed`: beide Mutanten lesen außerhalb der Zeilen und stürzen ab |
| internal/verify/gocover/gocover.go:83 | a3 (`<=`) | `line < 2` -> `line <= 2` | Test `TestAnExemptionOnTheFirstLineCoversTheSecond`, danach `killed` |
| internal/child/child.go:103 | a1 | `err != nil` -> `false` | Test `TestRunStopsAtTheFirstPipeThatFails`, danach `killed`: der alte Test ließ auch das zweite `os.Pipe` scheitern, und der Mutant meldete denselben Text |
| internal/child/child.go:160 | a1, a4 | `res.OutputAbandoned` -> `true`, `false`, `!(...)` | Tests `TestRunAbandonsOutputAGrandchildHolds` (geschärft: genau ein Kill, danach kein Prozess mehr im Job) und `TestRunKillsNothingAfterACleanExit` (drei Mutanten), danach `killed`. Unter Windows beendet auch `release` den Job (`JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`); gezählt wird darum über den Haken `afterKill`, und unter POSIX, wo `release` nichts tut, ist dieser Kill der einzige |
| internal/child/child_windows.go:39 | a1 | `err != nil` -> `false` | `failStart` verlangt jetzt `errors.Is(r.Err, errDenied)`; `TestRunReportsAFailedJob`, danach `killed`: der Mutant läuft mit Handle 0 weiter und scheitert erst an `SetInformationJobObject` |
| internal/child/child_windows.go:67 | a1 | `err != nil` -> `false` | Dieselbe geschärfte `failStart`; `TestRunReportsAFailedOpen`, danach `killed`: der Mutant scheitert erst an `AssignProcessToJobObject` mit Handle 0 |
| internal/child/child_other.go:18 | a1, a3, a4 | `err != nil` -> `true`, `false`, `err == nil`, `!(...)` | Unter Windows nicht zu beurteilen: die Datei trägt `//go:build !windows`, die Runde lief unter Windows, der Mutant wird dort nie übersetzt und überlebt jede Suite. Der a4-Mutant `if !(err := cmd.Start(); err != nil)` ist gar kein Go (dieselbe Form an `child_windows.go:54` und `:71` meldet die Runde als „nicht kompiliert") und wäre auch unter Linux kein Mutant. Die übrigen drei fielen unter Linux an vorhandenen Tests: `true` und `err == nil` geben nach einem gelungenen Start `(nil, nil)` zurück, und an der Frist dereferenziert `t.kill()` den leeren Baum (`TestRunKillsTheProcessGroup` stürzt ab); `false` und `err == nil` lesen nach einem gescheiterten Start `cmd.Process.Pid` von `nil` (`TestRunReportsAStartFailure` stürzt ab). Nachgerechnet, nicht gemessen; eine Runde auf dem Linux-Zweig von `ci.yml` stünde aus |

Die bekannte Schwachstelle der Leseschleife in `child.go` (Zeilen 153-159:
ist `until` vorbei, wählt `select` zufällig zwischen einem geschlossenen
`rd.done` und dem sofort feuernden Timer, ein fertiger Leser kann also als
`OutputAbandoned` gelten) hat weder ein Mutant noch ein Test dieser Runde
sichtbar gemacht. Behoben nach der Abschlussprüfung: `abandonedBy` fragt einen
beendeten Leser zuerst ohne zu warten; `TestAFinishedReaderIsNeverAbandoned`
hält es fest.
