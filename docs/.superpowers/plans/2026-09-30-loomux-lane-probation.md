# loomux: Schonfrist je Lane — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eine Lane, die in einem Projekt noch nie grün war, warnt nur; scharf, mit echtem Exit-Code, wird sie, sobald ein grüner pre-commit-Lauf sie in die versionierte `.loomux/armed.toml` einträgt. Ohne diese Datei bleibt jede Ausgabe von `check` und den Hooks Byte für Byte wie heute.

**Architecture:** Die Datei liest und schreibt ein Baustein in `internal/verify` (`ArmedSet`, Schlüssel `<art>/<sprache>@<bereich>` aus Art, Sprache und Bereich des Jobs). Das Urteil wandert nicht in `verify.Red` (die Funktion kennt nur einen Zustand, zwei ihrer Aufrufer haben gar keine Lane): `verify.Run` bekommt über `RunOptions.Armed` die Frage „ist diese Lane scharf?“ und stempelt `Outcome.Probation`; `verify.Fails` ist das neue Urteil für `check`, Stop-Hook und Post-Edit. Geschrieben wird nur von `loomux check precommit --arm` nach einem insgesamt grünen Lauf und von der Befehlsgruppe `loomux gate` des Menschen. Im Post-Edit-Hook geht der Befund einer Lane in Probe über den vorhandenen Kontextkanal des Wirts (`hosts.WriteContext`), weil bei Exit 0 niemand stderr liest. Der Stop-Hook setzt nach einem nur in Probe roten Lauf den Blockzähler auf 0 und merkt sich zu einem nur in Probe roten Baum einen Stand „gesehen“ in seiner Sitzungsdatei, der Session-Start reicht Lanes in Probe und diesen Stand über den vorhandenen Kontextkanal weiter, `init` legt die leere Datei an, wo vor dem Lauf weder Konfiguration noch Datei stand, das `apply.sh` der Umstellung neben der Konfiguration, die es schreibt, und die Hook-Vorlage für Projekte bleibt ein Einzeiler, der `--arm` ruft: Der Befehl selbst erkennt am Index, den git dem Hook gibt (`gitwork.CommitIndex`, gemessen, per `os.SameFile`), ob der Commit den ganzen Index nimmt, schreibt nur dann und legt die Datei mit `git add` in diesen Index; bei einem Teilcommit tut er nichts davon.

**Tech Stack:** Go (`internal/verify`, `internal/hooks`, `internal/sessions`, `internal/gitwork`, `internal/cli`, `internal/setup`, `internal/switchover`), POSIX-`sh` für die Hook-Vorlage und das Umstellungsskript, echtes git in den Weltentests.

**Spec:** docs/.superpowers/specs/2026-09-30-loomux-lane-probation-design.md

## Global Constraints

- Ausführung: Task 0 führt der Controller mit dem Menschen. Die Tasks 1 bis 12 sind Implementierer-Stoff (subagent-driven, ein frischer Prüfer je Task), in dieser Reihenfolge; jeder Task endet mit einem Commit, der das Tor des Repos besteht. Stand des Codes beim Schreiben des Plans: `47361ab3` auf `feat/lane-probation`; nach dem Rebase steht der Zweig auf `e6c21eed` (`master`, v6.0.0), und Task 0 hat jeden Anker am 2026-10-01 an diesem Stand neu gelesen.
- Nichts wird auf `master` gearbeitet; Zweig `feat/lane-probation`. Spec, Plan und Code liegen in einem Pull Request (kein Plan-PR allein). Nobody but a human pushes: der Agent nennt den Befehl und wartet. Subagenten pushen nie; nach jedem Subagenten `git log -1 --format='%an <%ae>'` lesen.
- Code, Bezeichner, Kommentare, Meldungen und Commit-Texte englisch; Arbeitspapiere unter `docs/.superpowers/` deutsch; `docs/en/**` englisch, `docs/de/**` deutsch, gleiche Dateinamen.
- Coverage 100 % je Funktion; ein Ausschluss nur mit `//coverage:exempt <reason>` direkt über `func`. Kein neuer Ausschluss in diesem Plan: jeder Fehlerzweig hat einen Test oder eine Naht (`var x = …`), die der Task nennt.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst.
- TDD: Jede neue Testzeile läuft zuerst rot gegen den Stand vor dem Code, und zwar als **Assertion**, nicht als Build-Fehler: neue Symbole bekommen zuerst einen Stub, der das alte Verhalten liefert. Der Bericht des Implementierers zeigt den roten Lauf mit der Assertion.
- Mutation je Regel: Die im Task genannten Mutanten werden per `go test -overlay` (oder `go run ./cmd/loomux dev mutants <paket>` bei Paketen, deren Suite unter 20 s läuft) einzeln gesetzt; jeder muss von dem genannten Test getötet werden. `internal/cli` und `internal/hooks` brauchen länger als die feste Grenze von `dev mutants`: dort die Handrunde mit `-overlay` gegen den gezielten Test (`-run`).
- Ein Commit je Task, Conventional Commits, der Scope nennt einen Bereich des Codes. Kein Arbeitspapier im Text (kein „Stufe“, „Plan“, „Task“, „Spec“, „Nachtrag“), KEIN `Co-Authored-By`, kein Modell als Autor, keine Werbezeile. Mehrzeilige Nachrichten per Write in eine Datei und `git commit -F <datei>`.
- Der Agent schreibt nie `.loomux/config.toml` und (ab Task 8) nie `.loomux/armed.toml` eines echten Projekts; Testwelten unter `t.TempDir()` schreibt der Testcode selbst. Er führt weder `loomux init` ohne `--dry-run` noch `loomux gate arm|disarm` aus.
- Die Go-Blöcke dieses Plans sind auf Inhalt geschrieben, nicht auf Spaltenausrichtung: Nach jedem Write/Edit einer `.go`-Datei läuft `gofmt -l <datei>` (leere Ausgabe), sonst fällt die Lane `lint/go` des Tors.
- Code mit Backslashes nie über ein Bash-Heredoc und nie über `sed`: Dateien per Write/Edit, danach mit Read nachlesen. Ein bedingt erlaubter `loomux`-Befehl (`loomux init --dry-run`, `loomux gate status`) läuft als schlichte Einzelzeile ohne Kette, Umleitung, `cd &&`.
- Die Ausgabe langer Läufe (`sh ci/gate.sh`, `go test ./...`) erst ganz in eine Datei im Scratchpad schreiben, dann filtern; bei Rot die Datei nach `--- FAIL` durchsuchen, bevor neu gestartet wird.
- Testwelten mit git setzen `gitenv.Environ()` als Umgebung jedes git-Aufrufs: Die Tests laufen im pre-commit-Tor von loomux, dessen git-Zeiger sonst auf loomux zeigen.
- **Invariante ohne Datei:** Nach jedem Task laufen die aufgezeichneten Fälle grün, ohne dass eine Aufzeichnung oder eine genehmigte Liste angefasst wurde: `go test ./internal/cli -run "TestCases" -count=1`. Das ist der Beweis, dass ohne `.loomux/armed.toml` jede Ausgabe gleich bleibt. Zwei Ausnahmen nennt die Spec: `init` (eine Datei mehr, ein anderer Hook-Text; Task 10) und die Ablehnung von `rm -rf .loomux`, die einen Grund mehr nennt (Task 8). Beide treffen keinen aufgezeichneten Fall.

## Review Focus

Jede Zeile hat ihren Test in dem Task, der in Klammern steht.

- **Eine unlesbare Datei stellt alles scharf, nie etwas stumpf.** Leere Datei (kein `armed`-Schlüssel), Konfliktmarken, falscher Typ, unbekannter Schlüssel: jede Lane scharf, Meldung auf stderr, `--arm` schreibt nichts darüber. Der nächstliegende Fehler ist, die leere Datei wie `armed = []` zu lesen (Task 2 `TestAnUnreadableFileArmsEveryLane`, Task 4 `TestAnUnreadableFileFailsARedLaneAndIsNotWrittenOver`, Task 6 `TestStopSaysAnUnreadableArmedFile`).
- **Eine zweite Area benennt die Lane um, der Schlüssel bleibt.** `lint/go` wird zu `lint/go@.`, sobald es zwei Bereiche gibt; der Schlüssel ist in beiden Welten `lint/go@.` und wird nie aus `Job.Name` gebildet (Task 2 `TestTheKeyOfALaneIsTheSameWithOneAreaAndWithTwo`, Task 7 `TestGateLanesNameTheWikiAndTheProjectLanes`).
- **Ein verweigerter Commit lässt Datei und Index unberührt.** `--arm` schreibt nur bei Exit 0; in einem roten Lauf wird auch keine andere, grüne Lane eingetragen (Task 4 `TestArmWritesNothingWhenTheRunIsRed`, Task 10 Weltentest Fall c).
- **Ein Teilcommit stellt nichts scharf.** Unter `git commit <pfad>` und `--only` gibt git dem Hook einen Index, der nach dem Commit nicht der echte wird; `--arm` erkennt ihn (`os.SameFile` gegen `index` und `index.lock` des Repos, nie ein Vergleich der Schreibweise), schreibt nicht und stagt nicht. Der nächstliegende Fehler ist, am Dateinamen oder an „`GIT_INDEX_FILE` ist gesetzt“ zu entscheiden: Gesetzt ist sie bei jedem Commit (Task 4 Messschritt und `TestCommitIndex…`, `TestArmLeavesAPartialCommitAlone`; Task 10 Weltentest Fälle d und e).
- **Die Tests laufen selbst in einem pre-commit-Hook.** Das Tor von loomux fährt die Suite unter einem gesetzten `GIT_INDEX_FILE`. Ein In-Prozess-Test von `--arm`, der die Umgebung nicht selbst festlegt, wäre allein grün und im Tor rot (Task 4: jede Testwelt setzt `GIT_INDEX_FILE` ausdrücklich).
- **Der Stop-Hook wird nicht still.** Eine scharfe rote Lane blockiert und zählt wie heute, auch wenn daneben Lanes in Probe rot sind und auch wenn ein Stand „gesehen“ vorliegt; „gesehen“ gilt nur für denselben Baum **und** denselben `HEAD`; ändert sich nur `armed.toml`, läuft die Kette neu; ein nur in Probe roter Lauf setzt den Blockzähler auf 0, bewegt aber weder `base` noch den grünen Baum (Task 6, alle Tests).
- **`blocked` folgt der Lane, die blockiert.** Coverage hinter einem roten Test in Probe ist nicht rot, hinter einem scharfen roten Test ist sie rot, auch wenn die Coverage-Lane selbst in Probe ist (Task 3 `TestBlockedIsRedOnlyBehindAnArmedLane`, Task 4 `TestBlockedBehindALaneInProbationDoesNotFail`).
- **Die Post-Edit-Warnung muss den Agenten erreichen.** Bei Exit 0 liest kein Wirt stderr; die Warnung steht im JSON des Wirts auf stdout (Claude Code `hookSpecificOutput.additionalContext`, Antigravity `injectSteps`), und der Test liest sie dort (Task 5 `TestAnEditIsHeldOnlyByAnArmedLane`, `TestAntigravityHearsALaneInProbation`).
- **Ein Projekt, das loomux schon nutzt, fällt nicht in die Probe zurück.** `init` legt die Datei nur an, wo weder Konfiguration noch Datei noch ein pre-commit-Hook von loomux stand; der nächstliegende Fehler ist, nur nach der Konfiguration zu fragen, die ein reines Go-Projekt nie hatte (Task 10 `TestALoomuxHookMeansTheProjectWasSetUp`, `TestInitStartsANewProjectInProbationOnce`).
- **Der gemerkte Stand des Stop-Hooks gilt nur unter den Lanes, die scharf waren.** In einem Projekt, das `.loomux/` ignoriert, ändert `gate arm` keinen Baum; der Stand trägt die scharfe Menge selbst (Task 6 `TestArmingBetweenTwoTurnEndsRunsTheChainAgain`, `TestLosingTheFileRunsTheChainAgain`).
- **Ein fremder oder alter pre-commit-Hook stellt nie scharf.** Nur ein Aufruf mit `--arm` schreibt; `init` ersetzt nur den eigenen älteren Einzeiler und nennt jeden anderen Hook, der nicht scharf stellt (Task 10 Weltentest Fall g, `TestAForeignPreCommitHookIsKeptAndNamed`; Task 9 `TestStatusNamesAHookThatDoesNotArm`).
- **Der Wächter öffnet nichts.** Keine Schreibweise, die vor der Änderung verweigert war, geht danach durch; `gate status` bleibt erlaubt, `gate arm|disarm` in jeder Schreibweise verweigert (Task 8, Differenzprobe als eigener Schritt).
- **Eine ignorierte Datei gilt nur auf diesem Rechner.** Ignoriert das Projekt `.loomux/` oder die Datei, warnen `gate status` und `status`; ein Projekt, das nur `/.loomux/state/` ignoriert, bekommt keine Warnung (Task 7 `TestGateStatusWarnsAboutAnIgnoredFile`, Task 9 `TestStatusWarnsAboutAnIgnoredFile`).

---

## Dateien

| Datei | Verantwortung |
|---|---|
| `internal/verify/armed.go` (neu) | `ArmedFile`, `ArmedSet` (`Arms`, `With`, `Without`, `Missing`, `Text`), `LaneKey`, `ReadArmed`, `WriteArmed`, `HookArms` |
| `internal/verify/run.go` (ändern) | `RunOptions.Armed`, `Outcome.Probation`, Stempel in `Run` und `inherit` |
| `internal/verify/report.go` (ändern) | `Fails`, Zusatz `(probation)` in `WriteCheck`, `CheckVerdict` und `EditReport` über `Fails`, `ProbationLine`, `GreenKeys` |
| `internal/gitwork/commitindex.go` (neu) | `CommitIndex` (ganzer Commit oder Teilcommit), `Stage` (`git add` in den Index des Hooks) |
| `internal/gitwork/gitwork.go` (ändern) | `IgnoredPath` (Task 7), `HooksDir` (Task 9) |
| `internal/cli/check.go` (ändern) | Datei lesen, `--arm` samt Teilcommit-Erkennung und Stagen, `probation:`- und `armed:`-Zeile |
| `internal/hooks/post_edit.go` (ändern) | die scharfe Menge an `verify.Run` reichen |
| `internal/sessions/state.go` (ändern) | `Seen` (Baum, `HEAD`, scharfe Lanes, Bericht, Zeit), `SessionState.Seen`, `LastSeen` |
| `internal/hooks/stop.go` (ändern) | viertes Urteil „nur in Probe rot“, Stand „gesehen“ merken und wiederholen, Blockzähler auf 0 |
| `internal/hooks/lanestates.go` (neu) | `GateLanes`, `LaneStates` (mit `Ignored`), `ReadLaneStates` |
| `internal/cli/gate.go` (neu), `internal/cli/commands.go` (ändern) | `loomux gate status|arm|disarm` |
| `internal/hooks/guard.go` (ändern), `internal/hooks/guardgate.go` (neu) | Pfadregel für `.loomux/armed.toml`, Befehlsregel für `gate arm|disarm` |
| `internal/hooks/testdata/guard-battery-before.tsv` (neu) | Urteil des Wächters vor der Änderung, Zeile für Zeile, samt der benannten Lücken als festgehaltene Durchgänge |
| `internal/hooks/probation.go` (neu), `hook_session_start.go`, `status.go` (ändern) | Kontextzeilen beim Session-Start, Abschnitt im Statusbericht |
| `internal/setup/gitfiles/gitfiles.go` (ändern) | pre-commit-Vorlage mit `--arm`, `Upgrade`, `IsLoomuxPreCommit` |
| `internal/setup/plan.go`, `apply.go`, `write/atomic.go` (ändern) | `armed.toml`, wo weder sie noch eine Konfiguration noch ein pre-commit-Hook von loomux stand; eigener alter Hook wird ersetzt, fremder genannt; Modus nach dem Ersetzen |
| `internal/switchover/apply.sh.tmpl` (ändern) | Schritt 4 legt die leere Datei mit an |
| `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`, `docs/en|de/migration.md`, `docs/.superpowers/parity/stufe-4a-2.md` | Nachtrag #27, Abhängigkeit der Zeile 4e, Papierspur zu `init` |
| `docs/en|de/configuration.md`, `hooks.md`, `cli-reference.md`, `README.md`, `README.de.md` | die Datei, die Probe, die Befehle |

---

## Belege und Messungen (am Code nachgerechnet, Stand `47361ab3`, am 2026-10-01 an `e6c21eed` neu gelesen; Spec in der Fassung von `ba9290cb`)

Diese Punkte binden die Tasks. Die Spec trägt sie inzwischen selbst; hier stehen die Stellen im Code und die Messwerte. Zwischen `47361ab3` und `e6c21eed` hat die letzte Fix-Welle der Umstellung keine der Dateien geändert, die die Punkte 1 bis 10 zitieren; die Zeilennummern gelten unverändert. Die Spec in der Fassung von `ba9290cb` ist die von `5d087f7b` mit den Fixups, die der Plan schon trägt (benannte Lücken des Wächters, der pre-commit-Hook von loomux als drittes Zeichen für `init`, Blockzähler auf 0, die Notiz zu einem fremden Hook nur mit Datei).

1. **Es gibt keine aufgezeichneten `init`-Fälle.** `testdata/cases/4a2/` enthält nur das Verb `hook` (`brain-mcp hook` gegen `loomux merge-hook`, `internal/cli/cases_4a2_test.go:77-80`), und `docs/.superpowers/parity/stufe-4a-2.md:21-23` sagt: „Außer dem post-merge-Hook hat kein Teil eine aufgenommene Referenz“. Keine genehmigte Liste wird erweitert. Task 10 ändert die Einheitstests, die den alten Hook-Text festnageln (`internal/setup/plan_test.go:132` und `:362`, `internal/setup/gitfiles/gitfiles_test.go:36`, `:63`, `:140-148`), und schreibt einen Abschnitt in die Paritätsakte `stufe-4a-2.md`.
2. **Der Kontextkanal für PostToolUse ist da, je Wirt.** `RunPostEdit` gibt bei Exit 0 alle Hinweise über `hosts.WriteContext(env.Host, "PostToolUse", stdout, notices)` aus (`internal/hooks/post_edit.go:120`); bei jedem anderen Code schreibt es kein stdout (`:117-119`).
   - Claude Code: `hosts.WriteContext` (`internal/hosts/hostio.go:128`) ruft `writeClaudeContext` (`internal/hosts/claude.go:83`), das `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":…}}` schreibt; der vorhandene Test `internal/hooks/post_edit_test.go:92-111` liest genau diesen Umschlag.
   - Antigravity: `writeAntigravityContext` (`internal/hosts/antigravity.go:82`) schreibt `{"injectSteps":[{"ephemeralMessage":…}]}`; `hosts.Answer` reicht stdout bei Exit 0 unverändert durch (`internal/hosts/answer.go:59-60`), und agy 1.2.12 zeigt es dem Modell nach einem PostToolUse (gemessen 2026-09-28, Kommentar `answer.go:27-29`).
   - Codex: `writeCodexContext` (`internal/hosts/codex.go:37`) liefert `ErrNoAdapter`; Post-Edit endet dort schon heute bei jedem grünen Lauf mit Exit 1 und blockiert nichts. Es gibt keinen gemessenen Hook-Vertrag, also nichts zu bauen: Der Befund einer Lane in Probe steht dort auf stderr neben der Meldung des fehlenden Adapters, wie jeder andere Hinweis.
   Nichts ist neu zu bauen: Task 5 legt den Befund der Lane in Probe in dieselbe `notices`-Liste, und die Tests lesen ihn im JSON beider Wirte.
3. **`verify.Red` hat keine Lane.** `Red(s State, scope Scope)` (`internal/verify/run.go:38`) wird an sieben Stellen gerufen: `run.go:100` (`inherit`: blockiert den Nachfolger), `run.go:242` (`mergeSteps`: Kopfzeile eines einzelnen Befehls), `report.go:36` (`WriteCheck`: zeigt die Ausgabe), `report.go:64` (`CheckVerdict`), `report.go:98` (`EditReport`), `internal/hooks/stop.go:295` (`stopVerdict`), `internal/hooks/post_edit.go:153` (Blast-Hinweis). Kleinste Änderung: `Red` bleibt, wie es ist; drei Urteilsstellen (`report.go:64`, `report.go:98`, `stop.go:295`) gehen auf `verify.Fails(o, scope)`. Die übrigen vier bleiben beim Zustand: Eine Lane in Probe blockiert ihren Nachfolger weiter, zeigt ihre Ausgabe weiter, und der Blast-Hinweis entfällt weiter, wenn eine Lane rot ist.
4. **Der Index, den git dem pre-commit-Hook gibt — gemessen am 2026-09-30 mit git 2.54.0.vfs.0.4 (Windows), beim Schreiben des Plans.** Der Hook schrieb `GIT_INDEX_FILE` in eine Datei:

   | Aufruf | `GIT_INDEX_FILE` im Hook | wird nach dem Commit der echte Index |
   |---|---|---|
   | `git commit` | `.git/index` (relativ zum Arbeitsverzeichnis des Hooks) | ja, es ist der echte |
   | `git commit --amend` | `.git/index` | ja |
   | `git commit -a` | `<repo>/.git/index.lock` (absolut) | ja |
   | `git commit --amend -a` | `<repo>/.git/index.lock` | ja |
   | `git commit --include <pfad>` | `<repo>/.git/index.lock` | ja |
   | `git commit <pfad>` | `<repo>/.git/next-index-<pid>.lock` | nein |
   | `git commit --only <pfad>` | `<repo>/.git/next-index-<pid>.lock` | nein |
   | `git commit --amend <pfad>` | `<repo>/.git/next-index-<pid>.lock` | nein |

   Die Variable war in keinem Fall ungesetzt, und im gewöhnlichen Fall ist sie **relativ**. Eine zweite Messung ließ den Hook eine Datei schreiben und mit `git add` stagen: bei `git commit`, `-a`, `--include` und `--amend` lag sie im Commit, und `git status --porcelain` war danach für sie leer. Beim Teilcommit (dritte Messung, vor der Entscheidung 12 der Spec) lag sie im Commit, der echte Index zeigte danach `MM .loomux/armed.toml`. Die Regel der Spec stimmt mit der Messung überein. Task 4 wiederholt die Messung als eigenen Schritt auf dem Rechner des Implementierers, bevor Code darauf baut.
5. **Die Tests laufen selbst in einem Hook.** `.githooks/pre-commit` von loomux ruft `ci/gate.sh`, also `go test ./...`, unter dem `GIT_INDEX_FILE` des laufenden Commits. `gitenv.Environ()` nimmt die Variable für Kindprozesse heraus (`internal/gitenv/gitenv.go:50-53`); ein Test, der `os.Getenv("GIT_INDEX_FILE")` im Prozess liest, sieht sie aber. Jede Testwelt von `--arm` setzt sie darum mit `t.Setenv` selbst.
6. **Die Testwelt des Stop-Hooks schließt `.loomux/` aus.** `internal/cases/gitworld.go:60` schreibt `/.loomux/` nach `.git/info/exclude`; in `gitWorld` (`internal/hooks/stop_test.go:55`) liegt `armed.toml` darum **nicht** im Baum. Der Test „nur `armed.toml` geändert“ (Task 6) überschreibt die Ausschlussliste. Im echten Projekt gilt: `gitwork.writeContentTree` (`internal/gitwork/gitwork.go:255-258`) stuft mit `add -A` alles ein, was git nicht ignoriert, und nimmt nur `.loomux/state` wieder heraus; `init` ignoriert nur `/.loomux/state/` (`internal/setup/gitfiles/gitfiles.go:65-67`). `.loomux/armed.toml` ist also Teil des Baums, solange das Projekt sie nicht selbst ignoriert; tut es das, warnen `gate status` und `status` (Tasks 7 und 9).
7. **`internal/hooks` darf nichts aus `internal/setup` importieren** (`internal/cli/imports_test.go:232`, `TestHooksNeverImportTheInstaller`). Die Frage „stellt dieser Hook scharf?“ (`HookArms`) liegt darum in `internal/verify`, und den Hook-Ordner liest `internal/gitwork` (`HooksDir`), nicht `internal/setup`.
8. **Ersetzte Dateien verlieren auf POSIX ihr Ausführungsbit.** `applier.change` (`internal/setup/apply.go:219-222`) ersetzt eine bestehende Datei über `lock.ReplaceText`, das über `os.CreateTemp` (Modus 0600) geht. Task 10 setzt nach dem Ersetzen den Modus über die vorhandene Regel von `internal/setup/write` (`mode`, wird zu `Mode` exportiert) und prüft die Wahl über eine Naht, weil Windows keine Ausführungsbits kennt.
9. **`init` schreibt in einem frischen Go-Projekt selbst keine Konfiguration** (`internal/cli/init_test.go:268`; `area add` schreibt sie, `internal/setup/plan.go:148-166`). Darum hängt die Datei nach Entscheidung 7 der Spec nicht am Schreiben der Konfiguration, sondern daran, dass vor dem Lauf weder `.loomux/config.toml` noch `.loomux/armed.toml` noch ein pre-commit-Hook von loomux stand. Der Hook ist nötig als drittes Zeichen: Ein Go-Projekt kann loomux ohne Konfiguration nutzen, und das `init`, das seinen Hook erneuert, ließe sonst jede Lane in die Probe fallen.
10. **`git restore <pfad>` und `git checkout <pfad>` ohne `--`** gelten dem Wächter nur als Schreiben, wenn der Pfad existiert (`checkedOut`, `internal/hooks/shellwrites.go:1013-1040`). Die Batterie von Task 8 nimmt die Formen mit `--` und prüft die ohne `--` mit und ohne Datei.

---

### Task 0: Vorbedingung — der Zweig steht auf `origin/master`

Der Zweig `feat/lane-probation` steht beim Schreiben dieses Plans auf `docs/stage-4e-spec`. **Die Umsetzung beginnt erst, nachdem `docs/stage-4e-spec` gemergt ist.** Kein Implementierer wird vorher gestartet.

**Files:** keine.

- [ ] **Step 1: prüfen, dass der Unterbau gemergt ist.** Run: `git fetch origin`, dann `gh pr list --state merged --head docs/stage-4e-spec`. Expected: eine Zeile mit dem gemergten Pull Request. Ist er nicht gemergt: hier anhalten und dem Menschen sagen, dass der Plan wartet.
- [ ] **Step 2: rebasen.** Run: `git rev-parse --show-toplevel` (muss der Worktree dieses Zweigs sein), dann `git rebase origin/master`. Die Commits von `docs/stage-4e-spec` fallen dabei weg, soweit sie gemergt sind; bei einem umgruppierten Merge bleiben Duplikate als Konflikt stehen: jeden Konflikt zugunsten von `origin/master` auflösen, die zwei Commits dieses Zweigs (Spec und Plan der Schonfrist) bleiben. Danach `git log --oneline origin/master..HEAD`: Expected: nur die Commits `docs(verify): design a probation …` und `docs(verify): plan the probation …` (samt ihrer Fixups).
- [ ] **Step 3: jeden Anker dieses Plans neu lesen.** Die folgenden Stellen zitiert der Plan wörtlich oder mit Zeilennummer; jede wird am neuen Stand geöffnet, und wo Name, Signatur oder Text abweicht, wird der Plan in einem Fixup-Commit berichtigt, **bevor** Task 1 startet:

  | Anker | Stand `47361ab3`, am 2026-10-01 an `e6c21eed` neu gelesen |
  |---|---|
  | `internal/verify/run.go` | `Red` Zeile 38, `inherit` Zeile 99, `Outcome` Zeile 15, `RunOptions` Zeile 25 |
  | `internal/verify/report.go` | `WriteCheck` Zeile 18, `CheckVerdict` Zeile 44, `EditReport` Zeile 95 |
  | `internal/verify/plan.go` | `Job` Zeile 50, `baseJob` Zeile 277 mit `len(eff.Areas[stack]) > 1` |
  | `internal/cli/check.go` | `checkRun` Zeile 155, Flags Zeile 161-166 |
  | `internal/cli/commands.go` | die Tabelle; `gate` ist frei |
  | `internal/hooks/stop.go` | `RunStop` Zeile 73, Schalter Zeile 171-185, `stopVerdict` Zeile 292 |
  | `internal/hooks/post_edit.go` | `checkEdit` Zeile 129 |
  | `internal/hooks/guard.go` | `manifestReason` Zeile 36, `builtinPathRules` Zeile 40, `checkTool` Zeile 272, `wordsWriteConfiguration` Zeile 500 (an `47361ab3` Zeile 499; die Fix-Welle hat den Kommentar von `writesConfiguration` um eine Zeile verlängert und in `wordsWriteConfiguration` den Fall `dev` für `dev switchover prune-hooks` ergänzt) |
  | `internal/hooks/guardflow.go` | `answersAGate` Zeile 25 |
  | `internal/hooks/guardshell_test.go` | `manifestShellWrites` Zeile 64, `manifestShellReads` Zeile 104 |
  | `internal/hooks/guardflow_test.go` | `asGateAnswer` Zeile 76 |
  | `internal/hooks/hook_session_start.go` | `SessionStart` Zeile 39, der Block `if !payload.Repeat` Zeile 74 |
  | `internal/hooks/status.go` | `Status` Zeile 101, `renderLaneTools` Zeile 287 |
  | `internal/sessions/state.go` | `SessionState` Zeile 17, `stateFile` Zeile 36 |
  | `internal/gitwork/gitwork.go` | `writeContentTree` Zeile 233 |
  | `internal/setup/gitfiles/gitfiles.go` | `Hooks` Zeile 29, Kennzeile `# loomux pre-commit hook:` Zeile 36 |
  | `internal/setup/plan.go` | `Build` Zeile 73, Konfigurationsblock Zeile 101-107, `declares` Zeile 151-166, `gitHooks` Zeile 354 |
  | `internal/setup/apply.go` | `change` Zeile 189 |
  | `internal/switchover/apply.sh.tmpl` | Schritt 4 Zeile 208-215 |
  | Fusions-Spec | letzte Zeile der Nachtragstabelle: `| 26 |`; Stufenzeile `**4e** Umstellung` |
  | `docs/en/migration.md`, `docs/de/migration.md` | Zeile `**4e**` |

  Erledigt am 2026-10-01 an `e6c21eed`. Von den Dateien dieser Tabelle und den übrigen, die die Tasks 1 bis 12 mit Zeile, Name oder Text zitieren, hat die Fix-Welle zwischen `47361ab3` und `e6c21eed` an einem Anker nur diese geändert: `internal/hooks/guard.go` (eine Zeile verschoben, siehe oben; neue Regel für `dev switchover prune-hooks`; der Text der Ablehnung von `writesConfiguration`, den der Plan nicht zitiert), `internal/switchover/apply_test.go` (nur `TestApplyCleansUpWhenItIsStopped`, hinter der zitierten Stelle `:692-696`), die Fusions-Spec und beide `migration.md` (Zeile 4e und Nachtrag #26 tragen das Warten der Welle schon; Task 1 ist darauf umgestellt). `internal/switchover/apply.sh.tmpl` ist unverändert, Schritt 4 steht weiter in Zeile 208-215; `internal/switchover/render.go` hat neue Prüfungen für `old_files`, die Task 11 nicht berührt. `README.md`, `README.de.md`, `docs/*/configuration.md` und `docs/*/cli-reference.md` haben in der Welle Text bekommen; die Überschriften und Zeilen, an denen Task 12 ansetzt (`### `[verify]` …`, `### `[modules]` …`, `## 4. Where Things Live`, `## 2.`, `## 4.`, `## 11.` samt „What it writes, and what it leaves“, `### `loomux dev switchover …`` — heute `### `loomux dev switchover <render|prune-hooks> [flags]``, `check precommit`, `## Roadmap`), stehen weiter. Jede andere zitierte Datei ist an beiden Ständen gleich, ihre Anker gelten. Die Go-Blöcke der Tasks 2 bis 11 sind am neuen Stand übersetzt und die Suiten gelaufen (siehe „Task 0, 2026-10-01“ am Ende).

- [ ] **Step 4: das Tor am rebasten Stand.** Run: `sh ci/gate.sh > "<scratchpad>/gate-task0.txt" 2>&1`, danach die Datei lesen. Expected: Exit 0. Erst dann beginnt Task 1.

---

### Task 1: Papiere — Nachtrag #27 und die Abhängigkeit der Stufe 4e

Eine Entscheidung geht zuerst in die Fusions-Spec; der Migrationsplan folgt ihr. Eine Roadmap-Zeile gibt es nicht (Entscheidung 14 der Spec): Plan und Code kommen in einem Pull Request, die Zeile entstünde und verschwände im selben.

**Files:**
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`
- Modify: `docs/en/migration.md`, `docs/de/migration.md`
- Test: `internal/plancheck/plancheck_test.go` (`TestTheRepositoryPlanFollowsTheSpec`, unverändert)

**Interfaces:** keine.

- [ ] **Step 1: die nächste freie Nummer nachlesen.** Run: Grep nach `^\| 2[0-9] \|` in der Fusions-Spec. Expected: die letzte Zeile beginnt mit `| 26 |`. Steht dort inzwischen `| 27 |`, nimmt dieser Task die nächste freie Nummer, und jede `#27` dieses Plans heißt dann so.
- [ ] **Step 2: die Zeile #27 in die Tabelle „Nachgetragen: was bisher keine Stufe hatte“ setzen,** direkt hinter die Zeile `| 26 |` (per Edit, eine Zeile):

  ```
  | 27 | — | Ein frisch eingerichtetes Tor ist vom ersten Tag an scharf über Code, den bisher kein Tor geprüft hat: Pilot `ecoflow` (2026-09-29, der Umstellungs-Commit ging nur mit `--no-verify`), Pilot `space` (2026-09-30, der Stop-Hook hielt jede Sitzung wegen 790 Befunden von `wiki lint`) | Schonfrist je Lane (Spec `2026-09-30-loomux-lane-probation-design.md`): eine versionierte `.loomux/armed.toml` nennt die scharfen Lanes; ohne Datei ist jede Lane scharf und jede Ausgabe wie bisher, mit Datei warnt jede Lane, die nicht darin steht. Scharf stellt nur `loomux check precommit --arm` nach einem insgesamt grünen Lauf oder ein Mensch mit `loomux gate arm`. Keine neue Stufe und kein eigener Rang: **Reihenfolge** erst die vorbereitete Umstellung (#26) mergen und veröffentlichen (geschehen mit v6.0.0 am 2026-10-01), dann die Schonfrist mergen und veröffentlichen, dann die Welle der Stufe 4e | Ohne sie zwingt jede Umstellung zu `--no-verify` oder zum Abschalten des Stop-Hooks, und beides lässt auch die Lanes fallen, die schon grün sind | freigegeben 2026-09-30 (im Gespräch am 2026-09-29 und 2026-09-30 entschieden) |
  ```

  **Was schon steht (Stand `e6c21eed`, geschrieben von der letzten Fix-Welle der Umstellung):** Die Stufenzeile 4e der Fusions-Spec, Nachtrag #26, Punkt 2 des Abschnitts „Umstellung der Wirte nach Stufe 4“ und beide 4e-Zeilen von `migration.md` sagen bereits, dass die Welle auf die Lane-Probation wartet, in einem eigenen Pull Request. Es fehlen nur die Zeile #27 selbst, der Verweis auf sie und die Bedingung „gemergt und veröffentlicht“ samt ihrem Grund. Die Schritte 3 bis 6 ersetzen darum je eine stehende Wendung per Edit (der alte Text als `old_string`, genau einmal in der Datei) und hängen nichts an, was schon dasteht. Stufe, Stand, Herkunft und Rang jeder Zeile bleiben.

- [ ] **Step 3: die Stufenzeile `**4e** Umstellung` der Fusions-Spec.** In der dritten Spalte wird
  ``die Welle über die übrigen sechs Projekte und den Vault wartet auf die Lane-Probation (Spec `2026-09-30-loomux-lane-probation-design.md`, eigener PR);``
  zu
  ``die Welle über die übrigen sechs Projekte und den Vault wartet auf die Lane-Probation (Nachtrag #27, Spec `2026-09-30-loomux-lane-probation-design.md`, eigener PR), bis sie gemergt und veröffentlicht ist;``
- [ ] **Step 4: die übrigen Stellen der Fusions-Spec, die den Stand der Nachträge nennen.**
  - Der Kopf der Spec (Absatz **Fusions-Stufen:**) endet über zwei Zeilen mit „alle sind freigegeben, ebenso die beiden / Nachträge #25 und #26 zur Stufe 4e (2026-09-30).“ Daraus wird „alle sind freigegeben, ebenso die beiden / Nachträge #25 und #26 zur Stufe 4e und #27, die Schonfrist je Lane, auf die ihre Welle wartet (2026-09-30).“ (umbrochen wie der Absatz, rund 80 Zeichen je Zeile).
  - Im Abschnitt „Umstellung der Wirte nach Stufe 4“, Punkt 2, endet der Absatz „Korrigiert am 2026-09-29 (Nachtrag #26)“ mit „die Welle wartet seit dem / 2026-09-30 auf die Lane-Probation (siehe Nachtrag #26).“ Daraus wird „… (siehe Nachträge #26 und #27).“
- [ ] **Step 5: `docs/en/migration.md`, Zeile `**4e**`.**
  - Spalte „What it delivered“: ``The wave over the remaining six projects and the vault waits for the lane probation, a pull request of its own; the clean-up pull request waits for the wave.`` wird ``The wave over the remaining six projects and the vault waits until the lane probation, a pull request of its own, is merged and released (fusion spec #27): a lane that has never been green warns first, so a freshly set-up project is not held by findings no gate ever showed it; the clean-up pull request waits for the wave.``
  - Spalte „Depends on“: ``the wave: the lane probation released (spec `2026-09-30-loomux-lane-probation-design.md`, its own pull request)`` wird ``the wave: the lane probation merged and released (fusion spec #27, spec `2026-09-30-loomux-lane-probation-design.md`, its own pull request)``.
- [ ] **Step 6: `docs/de/migration.md`, Zeile `**4e**`.**
  - Spalte „Was sie gebracht hat“: ``Die Welle über die übrigen sechs Projekte und den Vault wartet auf die Lane-Probation, einen eigenen Pull Request; der Aufräum-Pull-Request wartet auf die Welle.`` wird ``Die Welle über die übrigen sechs Projekte und den Vault wartet, bis die Lane-Probation, ein eigener Pull Request, gemergt und veröffentlicht ist (Fusions-Spec #27): Eine Lane, die noch nie grün war, warnt zuerst, damit ein frisch eingerichtetes Projekt nicht an Befunden hängt, die ihm nie ein Tor gezeigt hat; der Aufräum-Pull-Request wartet auf die Welle.``
  - Spalte „Hängt ab von“: ``die Welle: die Lane-Probation freigegeben (Spec `2026-09-30-loomux-lane-probation-design.md`, eigener Pull Request)`` wird ``die Welle: die Lane-Probation gemergt und veröffentlicht (Fusions-Spec #27, Spec `2026-09-30-loomux-lane-probation-design.md`, eigener Pull Request)``.
- [ ] **Step 7: grün.** Run: `go test ./internal/plancheck -count=1`. Expected: PASS (`TestTheRepositoryPlanFollowsTheSpec` liest Stufe, Stand und Rang; keiner davon hat sich geändert; die Tabelle der Nachträge liest `internal/plancheck` nicht). Grep nach `#27` in den drei Dateien: Expected: in der Spec drei Treffer (Stufenzeile, Kopf, Punkt 2 der Umstellung; die neue Zeile beginnt mit `| 27 |` und nennt `#27` nicht), in jeder `migration.md` zwei. Danach per Python-Skript aus dem Scratchpad (per Write angelegt, nicht per Heredoc) die Bytes `\r` in den drei geänderten Dateien zählen: Expected: 0. Die READMEs bleiben in diesem Task unberührt: Grep nach `probation` und `Schonfrist` im Abschnitt `## Roadmap` beider READMEs: kein Treffer.
- [ ] **Step 8: Commit** `docs(migration): make the switch-over wave wait until the lane probation is released`.

---

### Task 2: `internal/verify/armed.go` — die Datei lesen, schreiben, befragen

**Files:**
- Create: `internal/verify/armed.go`
- Test: `internal/verify/armed_test.go`

**Interfaces:**
- Consumes: `verify.Job` (`Kind`, `Stack`, `Area`), `verify.Plan`, `config.QuoteTOML(value string) string`, `lock.ReplaceText(path, text string) error`, `toml.Decode(data string, v any)`.
- Produces:
  - `const ArmedFile = ".loomux/armed.toml"`
  - `type ArmedSet struct { Exists bool; Keys []string }` — der Nullwert ist ein Projekt ohne Datei: jede Lane scharf.
  - `func LaneKey(j Job) string`
  - `func ReadArmed(root string) (ArmedSet, error)` — fehlt die Datei: `ArmedSet{}, nil`; ist sie unlesbar: `ArmedSet{}` und ein Fehler, dessen Text mit `; every lane is armed` endet.
  - `func (a ArmedSet) Arms(j Job) bool`
  - `func (a ArmedSet) With(keys ...string) ArmedSet`, `func (a ArmedSet) Without(keys ...string) ArmedSet`, `func (a ArmedSet) Missing(keys []string) []string`
  - `func (a ArmedSet) Text() string`
  - `func WriteArmed(root string, a ArmedSet) error`
  - `func HookArms(text string) bool`

- [ ] **Step 1: die Tests schreiben** (`internal/verify/armed_test.go`)

```go
package verify

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/detect"
)

func armedAt(t *testing.T, text string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "armed.toml"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// keyOf is the key of the planned job with this kind, stack and area.
func keyOf(t *testing.T, jobs []Job, kind, stack, area string) (key, name string) {
	t.Helper()
	for _, j := range jobs {
		if j.Kind == kind && j.Stack == stack && j.Area == area {
			return LaneKey(j), j.Name
		}
	}
	t.Fatalf("no %s/%s in %s among %s", kind, stack, area, names(jobs))
	return "", ""
}

// A second area renames the lane (lint/go becomes lint/go@.), and a key built
// from the name would drop the lane back into probation without a word.
func TestTheKeyOfALaneIsTheSameWithOneAreaAndWithTwo(t *testing.T) {
	const src = "[verify.project]\nlint = \"echo hi\"\n"
	one := effFor(t, src, goOnly)
	two := effFor(t, src, detect.Facts{Stacks: []string{"go"}, Areas: map[string][]string{"go": {".", "tools"}}})
	req := Request{Kinds: []string{"lint", "graph"}, Scope: ScopeCheck}
	ready := env(t.TempDir())
	ready.GraphReady = func(string) (bool, string) { return true, "" }
	jobsOne, err := Plan(one, req, ready)
	if err != nil {
		t.Fatal(err)
	}
	jobsTwo, err := Plan(two, req, ready)
	if err != nil {
		t.Fatal(err)
	}
	keyOne, nameOne := keyOf(t, jobsOne, "lint", "go", ".")
	keyTwo, nameTwo := keyOf(t, jobsTwo, "lint", "go", ".")
	if nameOne != "lint/go" || nameTwo != "lint/go@." {
		t.Fatalf("the world must rename the lane: %q, %q", nameOne, nameTwo)
	}
	if keyOne != "lint/go@." || keyTwo != keyOne {
		t.Fatalf("keys %q and %q, want lint/go@. twice", keyOne, keyTwo)
	}
	if key, _ := keyOf(t, jobsTwo, "lint", "go", "tools"); key != "lint/go@tools" {
		t.Errorf("the second area: %q", key)
	}
	if key, _ := keyOf(t, jobsOne, "lint", "project", "."); key != "lint/project@." {
		t.Errorf("the project lane: %q", key)
	}
	// The graph lane's name carries no area at all; its key still does.
	if key, name := keyOf(t, jobsTwo, "graph", "go", "."); key != "graph/go@." || name != "graph/go" {
		t.Errorf("the graph lane: key %q, name %q", key, name)
	}
	if got := LaneKey(Job{Kind: "lint", Stack: "go", Area: `tools\gen`}); got != "lint/go@tools/gen" {
		t.Errorf("an area is spelt with slashes: %q", got)
	}
}

func TestWithoutTheFileEveryLaneIsArmed(t *testing.T) {
	set, err := ReadArmed(t.TempDir())
	if err != nil || set.Exists || len(set.Keys) != 0 {
		t.Fatalf("%+v %v", set, err)
	}
	if !set.Arms(Job{Kind: "lint", Stack: "go", Area: "."}) {
		t.Fatal("no file must arm every lane")
	}
}

func TestTheFileArmsOnlyTheLanesItNames(t *testing.T) {
	set, err := ReadArmed(armedAt(t, "armed = [\n  \"test/go@.\",\n  \"lint/go@.\",\n  \"lint/go@.\",\n]\n"))
	if err != nil || !set.Exists || !slices.Equal(set.Keys, []string{"lint/go@.", "test/go@."}) {
		t.Fatalf("%+v %v", set, err)
	}
	if !set.Arms(Job{Kind: "lint", Stack: "go", Area: "."}) || set.Arms(Job{Kind: "lint", Stack: "go", Area: "tools"}) {
		t.Fatalf("%+v arms the wrong lanes", set)
	}
	empty, err := ReadArmed(armedAt(t, "# a comment\narmed = []\n"))
	if err != nil || !empty.Exists || empty.Arms(Job{Kind: "lint", Stack: "go", Area: "."}) {
		t.Fatalf("an empty list arms nothing: %+v %v", empty, err)
	}
}

// A broken file disarms nothing. The empty file is the nearest wrong
// neighbour of `armed = []`: a file a merge or an editor truncated.
func TestAnUnreadableFileArmsEveryLane(t *testing.T) {
	for name, text := range map[string]string{
		"empty":            "",
		"only a comment":   "# nothing\n",
		"conflict markers": "<<<<<<< HEAD\narmed = []\n=======\narmed = [\"lint/go@.\"]\n>>>>>>> theirs\n",
		"a string":         "armed = \"lint/go@.\"\n",
		"a number inside":  "armed = [1]\n",
		"an unknown key":   "armed = []\nother = true\n",
		"a table":          "[armed]\nlint = true\n",
	} {
		set, err := ReadArmed(armedAt(t, text))
		if err == nil || !strings.HasPrefix(err.Error(), ".loomux/armed.toml ") || !strings.HasSuffix(err.Error(), "; every lane is armed") {
			t.Errorf("%s: err %v", name, err)
		}
		if set.Exists || !set.Arms(Job{Kind: "lint", Stack: "go", Area: "."}) {
			t.Errorf("%s: %+v must arm every lane", name, set)
		}
	}
	// A directory in the file's place cannot be read either.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "armed.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if set, err := ReadArmed(root); err == nil || set.Exists {
		t.Errorf("a directory: %+v %v", set, err)
	}
	// Two unknown keys are named in a fixed order.
	_, err := ReadArmed(armedAt(t, "zeta = 1\nalpha = 2\narmed = []\n"))
	if err == nil || !strings.Contains(err.Error(), "alpha") {
		t.Errorf("unknown keys: %v", err)
	}
}

func TestWithWithoutAndMissingKeepTheKeysSortedAndOnce(t *testing.T) {
	set := ArmedSet{}.With("test/go@.", "lint/go@.", "test/go@.")
	if !set.Exists || !slices.Equal(set.Keys, []string{"lint/go@.", "test/go@."}) {
		t.Fatalf("%+v", set)
	}
	if got := set.Without("lint/go@.", "nope"); !got.Exists || !slices.Equal(got.Keys, []string{"test/go@."}) {
		t.Fatalf("%+v", got)
	}
	if !slices.Equal(set.Keys, []string{"lint/go@.", "test/go@."}) {
		t.Fatalf("Without changed its receiver: %+v", set)
	}
	// Taking a key out of a project without the file leaves a file: the one
	// that arms what is left. It does not stay "no file, every lane armed".
	if got := (ArmedSet{}).Without("lint/go@."); !got.Exists || len(got.Keys) != 0 {
		t.Fatalf("Without on the zero value: %+v", got)
	}
	if got := set.Missing([]string{"test/go@.", "types/go@.", "coverage/go@.", "types/go@."}); !slices.Equal(got, []string{"coverage/go@.", "types/go@."}) {
		t.Fatalf("missing %v", got)
	}
}

func TestTheTextIsOneKeyPerLine(t *testing.T) {
	const head = "# Lanes that are armed: a red run of one of them fails the gate.\n" +
		"# Written by the pre-commit gate; a human edits it through `loomux gate`.\n"
	if got := (ArmedSet{Exists: true}).Text(); got != head+"armed = []\n" {
		t.Fatalf("empty: %q", got)
	}
	got := ArmedSet{}.With("test/go@.", "lint/python@.").Text()
	if got != head+"armed = [\n  \"lint/python@.\",\n  \"test/go@.\",\n]\n" {
		t.Fatalf("two keys: %q", got)
	}
	// What is written reads back as it was, a key with a quote included.
	odd := ArmedSet{}.With(`lint/go@a"b`)
	back, err := ReadArmed(armedAt(t, odd.Text()))
	if err != nil || !slices.Equal(back.Keys, odd.Keys) {
		t.Fatalf("%+v %v", back, err)
	}
}

func TestWriteArmedReplacesTheFileWithLF(t *testing.T) {
	root := t.TempDir()
	if err := WriteArmed(root, ArmedSet{}.With("lint/go@.")); err != nil {
		t.Fatal(err)
	}
	if err := WriteArmed(root, ArmedSet{}.With("lint/go@.", "test/go@.")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".loomux", "armed.toml"))
	if err != nil || bytes.Contains(data, []byte("\r")) || string(data) != (ArmedSet{}).With("lint/go@.", "test/go@.").Text() {
		t.Fatalf("%q %v", data, err)
	}
	entries, _ := os.ReadDir(filepath.Join(root, ".loomux"))
	if len(entries) != 1 {
		t.Fatalf("a temporary file stayed behind: %v", entries)
	}
}

func TestWriteArmedReportsWhatItCannotWrite(t *testing.T) {
	// .loomux is a file: the directory cannot be made.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".loomux"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteArmed(root, ArmedSet{Exists: true}); err == nil {
		t.Fatal("a file in place of .loomux went through")
	}
	// armed.toml is a directory: the swap cannot land.
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "armed.toml", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteArmed(root, ArmedSet{Exists: true}); err == nil {
		t.Fatal("a directory in place of the file went through")
	}
}

func TestHookArmsReadsTheCallNotTheComment(t *testing.T) {
	for text, want := range map[string]bool{
		"#!/bin/sh\n\"/opt/loomux\" check precommit --arm\ncode=$?\nexit \"$code\"\n": true,
		"#!/bin/sh\nloomux check precommit --root . --arm\n":                           true,
		"#!/bin/sh\nexec \"/opt/loomux\" check precommit\n":                            false,
		"#!/bin/sh\n# loomux check precommit --arm\nsh ci/gate.sh\n":                   false,
		"#!/bin/sh\nloomux gate status --arm\n":                                        false,
		"": false,
	} {
		if got := HookArms(text); got != want {
			t.Errorf("HookArms(%q) = %v, want %v", text, got, want)
		}
	}
}
```

- [ ] **Step 2: rot laufen lassen.** Zuerst `internal/verify/armed.go` mit Stubs anlegen, die das alte Verhalten liefern: `LaneKey` gibt `j.Name` zurück, `ReadArmed` gibt `ArmedSet{}, nil`, `Arms` gibt `true`, `With`/`Without` geben den Empfänger zurück, `Missing` gibt `nil`, `Text` gibt `""`, `WriteArmed` gibt `nil`, `HookArms` gibt `false`. Run: `go test ./internal/verify -run "TestTheKeyOfALane|TestWithoutTheFile|TestTheFileArms|TestAnUnreadableFile|TestWithWithout|TestTheTextIs|TestWriteArmed|TestHookArms" -count=1`. Expected: FAIL mit Assertions, darunter `keys "lint/go" and "lint/go@.", want lint/go@. twice`; `TestWithoutTheFileEveryLaneIsArmed` ist schon grün (der Stub ist das alte Verhalten).

- [ ] **Step 3: implementieren** (`internal/verify/armed.go`)

```go
package verify

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// ArmedFile is where a project keeps its armed lanes, relative to its root.
// It is versioned: every contributor and the CI read the same lanes.
const ArmedFile = ".loomux/armed.toml"

// armedHead opens the file; a human who finds it learns who writes it.
const armedHead = "# Lanes that are armed: a red run of one of them fails the gate.\n" +
	"# Written by the pre-commit gate; a human edits it through `loomux gate`.\n"

// ArmedSet is the lanes of a project whose red fails a run. The zero value
// is a project without the file, where every lane is armed; with the file a
// lane it does not name is in probation: it runs and reports, and fails
// nothing.
type ArmedSet struct {
	Exists bool
	Keys   []string // sorted, each once
}

// LaneKey names a lane in the file: kind, stack and area, the area always
// written out. The name a report prints will not do: it carries the area
// only where a stack has several, so a second area would rename the lane.
func LaneKey(j Job) string {
	// Not filepath.ToSlash: that leaves a backslash alone on POSIX, and one
	// project would then have two keys for one lane.
	return j.Kind + "/" + j.Stack + "@" + strings.ReplaceAll(j.Area, `\`, "/")
}

// ReadArmed reads root's file. A missing file is no error and arms every
// lane. A file that does not read arms every lane as well and says why: a
// truncated or conflicted file must not switch a gate off.
func ReadArmed(root string) (ArmedSet, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ArmedFile)))
	if errors.Is(err, fs.ErrNotExist) {
		return ArmedSet{}, nil
	}
	if err != nil {
		return ArmedSet{}, unreadable("cannot be read: " + err.Error())
	}
	doc := map[string]any{}
	if _, err := toml.Decode(string(data), &doc); err != nil {
		return ArmedSet{}, unreadable("is no TOML: " + err.Error())
	}
	for _, key := range slices.Sorted(maps.Keys(doc)) {
		if key != "armed" {
			return ArmedSet{}, unreadable("holds the unknown key " + key)
		}
	}
	list, ok := doc["armed"].([]any)
	if !ok {
		return ArmedSet{}, unreadable("needs `armed` as a list of lane keys")
	}
	keys := make([]string, 0, len(list))
	for _, entry := range list {
		key, ok := entry.(string)
		if !ok {
			return ArmedSet{}, unreadable("needs `armed` as a list of lane keys")
		}
		keys = append(keys, key)
	}
	return ArmedSet{}.With(keys...), nil
}

// unreadable is the one refusal ReadArmed has, with what follows from it.
func unreadable(why string) error {
	return fmt.Errorf("%s %s; every lane is armed", ArmedFile, why)
}

// Arms says whether a red run of j fails the gate.
func (a ArmedSet) Arms(j Job) bool {
	return !a.Exists || slices.Contains(a.Keys, LaneKey(j))
}

// With is the set with keys added; it exists afterwards.
func (a ArmedSet) With(keys ...string) ArmedSet {
	all := slices.Concat(a.Keys, keys)
	slices.Sort(all)
	return ArmedSet{Exists: true, Keys: slices.Compact(all)}
}

// Without is the set with keys taken out; it exists afterwards.
func (a ArmedSet) Without(keys ...string) ArmedSet {
	kept := slices.DeleteFunc(slices.Clone(a.Keys), func(k string) bool { return slices.Contains(keys, k) })
	return ArmedSet{Exists: true, Keys: kept}
}

// Missing are the keys the set does not hold, sorted, each once.
func (a ArmedSet) Missing(keys []string) []string {
	var out []string
	for _, k := range keys {
		if !slices.Contains(a.Keys, k) && !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

// Text is the file: one key a line, so that two branches that armed
// different lanes merge by keeping both sides.
func (a ArmedSet) Text() string {
	if len(a.Keys) == 0 {
		return armedHead + "armed = []\n"
	}
	var b strings.Builder
	b.WriteString(armedHead + "armed = [\n")
	for _, k := range a.Keys {
		b.WriteString("  " + config.QuoteTOML(k) + ",\n")
	}
	b.WriteString("]\n")
	return b.String()
}

// WriteArmed swaps root's file for a's text, with LF on every platform.
func WriteArmed(root string, a ArmedSet) error {
	path := filepath.Join(root, filepath.FromSlash(ArmedFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return lock.ReplaceText(path, a.Text())
}

// HookArms says whether a pre-commit hook's text runs the gate with --arm.
// Comment lines do not count: a hook that mentions the call is not one that
// makes it.
func HookArms(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") && strings.Contains(line, "precommit") && strings.Contains(line, "--arm") {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: grün.** Run: `go test ./internal/verify -count=1`. Expected: PASS. Coverage: `go test ./internal/verify -coverprofile="<scratchpad>/verify.out" -count=1`, dann `go run ./cmd/loomux check gocover --profile "<scratchpad>/verify.out" --dir .`: jede Funktion von `armed.go` bei 100 %.
- [ ] **Step 5: Mutation per `go test -overlay`,** jede einzeln:
  - `LaneKey`: `j.Kind + "/" + j.Stack + "@" + …` durch `j.Name` ersetzen: `TestTheKeyOfALaneIsTheSameWithOneAreaAndWithTwo` wird rot.
  - `ReadArmed`: die Prüfung `if !ok { return ArmedSet{}, unreadable("needs …") }` hinter `doc["armed"].([]any)` entfernen: `TestAnUnreadableFileArmsEveryLane` (Fall `empty`) wird rot.
  - `ReadArmed`: die Schleife über unbekannte Schlüssel entfernen: Fall `an unknown key` wird rot.
  - `Arms`: `!a.Exists ||` durch `a.Exists ||` ersetzen: `TestWithoutTheFileEveryLaneIsArmed` wird rot.
  - `With`: `slices.Sort(all)` entfernen: `TestWithWithoutAndMissingKeepTheKeysSortedAndOnce` wird rot.
  - `Without`: `Exists: true` durch `Exists: a.Exists` ersetzen: derselbe Test („Without on the zero value“) wird rot.
  - `HookArms`: `!strings.HasPrefix(line, "#") &&` entfernen: `TestHookArmsReadsTheCallNotTheComment` wird rot.
- [ ] **Step 6: Commit** `feat(verify): read and write the armed lanes of a project`.

---

### Task 3: Das Urteil — eine Lane in Probe macht den Lauf nicht rot

Kein Aufrufer reicht nach diesem Task eine scharfe Menge; das Verhalten aller Befehle bleibt gleich.

**Files:**
- Modify: `internal/verify/run.go` (`Outcome`, `RunOptions`, `Run`, `inherit`)
- Modify: `internal/verify/report.go` (`WriteCheck`, `CheckVerdict`, `EditReport`; neu `Fails`, `ProbationLine`, `GreenKeys`)
- Test: `internal/verify/run_test.go`, `internal/verify/report_test.go`

**Interfaces:**
- Consumes: `LaneKey` (Task 2).
- Produces:
  - `RunOptions.Armed func(Job) bool` — `nil` heißt: jede Lane scharf.
  - `Outcome.Probation bool` — das Rot dieses Ergebnisses macht den Lauf nicht rot.
  - `func Fails(o Outcome, scope Scope) bool`
  - `func ProbationLine(outs []Outcome, armed func(Job) bool) string` — `""`, wenn keine Lane in Probe etwas zu prüfen hatte.
  - `func GreenKeys(outs []Outcome) []string` — Schlüssel der Lanes mit Zustand `ok`, sortiert, je einmal.
  - Unverändert: `func Red(s State, scope Scope) bool`, `func WriteCheck(w io.Writer, outs []Outcome, verbose bool)`, `func CheckVerdict(kinds []string, outs []Outcome) (code int, notes []string)`, `func EditReport(stderr io.Writer, outs []Outcome, aside string) (red bool, notices []string)`.

- [ ] **Step 1: die Tests schreiben.** In `internal/verify/run_test.go`:

```go
// lane is job with the fields a key is built from.
func lane(kind string, after int, argv string) Job {
	j := job(kind+"/go", after, argv)
	j.Kind, j.Stack, j.Area = kind, "go", "."
	return j
}

// Coverage waits for a test that fails. Whether the blocked lane is red is
// decided by the lane that blocks it, not by its own place in the file.
func TestBlockedIsRedOnlyBehindAnArmedLane(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		if s.Argv[0] == "test" {
			return child.Result{Code: 1}
		}
		return child.Result{}
	}}
	jobs := []Job{lane("test", -1, "test run"), lane("coverage", 0, "cover run")}
	for name, c := range map[string]struct {
		armed               func(Job) bool
		testProb, coverProb bool
	}{
		"no set":                        {nil, false, false},
		"the test is in probation":      {func(j Job) bool { return j.Kind != "test" }, true, true},
		"only coverage is in probation": {func(j Job) bool { return j.Kind == "test" }, false, false},
		"both are in probation":         {func(Job) bool { return false }, true, true},
	} {
		o := opts(f)
		o.Armed = c.armed
		outs := Run(jobs, o)
		if outs[0].State != StateFailed || outs[1].State != StateBlocked || outs[1].BlockedBy != "test/go" {
			t.Fatalf("%s: states %s, %s by %q", name, outs[0].State, outs[1].State, outs[1].BlockedBy)
		}
		if outs[0].Probation != c.testProb || outs[1].Probation != c.coverProb {
			t.Errorf("%s: probation %v, %v; want %v, %v", name, outs[0].Probation, outs[1].Probation, c.testProb, c.coverProb)
		}
		if Fails(outs[1], ScopeCheck) == c.coverProb {
			t.Errorf("%s: blocked fails = %v", name, Fails(outs[1], ScopeCheck))
		}
	}
}

// A lane that ran, and one that inherits "cannot judge", carry the answer for
// their own lane.
func TestEveryOutcomeSaysWhetherItsLaneIsInProbation(t *testing.T) {
	f := &fakeStart{answer: ok}
	pre := lane("test", -1, "test run")
	pre.Pre, pre.Note = StateUnavailable, "no tests found"
	jobs := []Job{pre, lane("coverage", 0, "cover run"), lane("lint", -1, "lint run")}
	o := opts(f)
	o.Armed = func(j Job) bool { return j.Kind == "lint" }
	outs := Run(jobs, o)
	if !outs[0].Probation || !outs[1].Probation || outs[2].Probation {
		t.Fatalf("probation %v %v %v", outs[0].Probation, outs[1].Probation, outs[2].Probation)
	}
	if outs[1].State != StateUnavailable {
		t.Fatalf("coverage %s", outs[1].State)
	}
}
```

  `fakeStart` (Feld `answer`, Methode `start`), `opts`, `job` und `ok` stehen in `run_test.go`.

  In `internal/verify/report_test.go`:

```go
func TestFailsIsRedOutsideProbation(t *testing.T) {
	for _, s := range []State{StateOK, StateFailed, StateTimedOut, StateBudget, StateBlocked, StateMissingTool, StateUnready, StateUnavailable, StateNotApplicable} {
		for _, scope := range []Scope{ScopeCheck, ScopeEdit} {
			if got := Fails(Outcome{State: s}, scope); got != Red(s, scope) {
				t.Errorf("%s armed: fails %v, red %v", s, got, Red(s, scope))
			}
			if Fails(Outcome{State: s, Probation: true}, scope) {
				t.Errorf("%s in probation fails", s)
			}
		}
	}
}

func probing(name, key string, s State, output string) Outcome {
	kind, rest, _ := strings.Cut(key, "/")
	stack, area, _ := strings.Cut(rest, "@")
	return Outcome{Job: Job{Name: name, Kind: kind, Stack: stack, Area: area, Origin: "preset"},
		State: s, Output: output, Duration: 800 * time.Millisecond, Probation: true}
}

// The state stays the real one; the header says that it does not count. A
// green lane in probation reads like any green lane.
func TestWriteCheckMarksARedLaneInProbation(t *testing.T) {
	blocked := probing("coverage/go", "coverage/go@.", StateBlocked, "")
	blocked.BlockedBy = "test/go"
	outs := []Outcome{
		probing("lint/python", "lint/python@.", StateFailed, "E1 bad\n"),
		probing("lint/go", "lint/go@.", StateOK, ""),
		probing("types/python", "types/python@.", StateMissingTool, `"mypy" is not on PATH: mypy .`),
		blocked,
		probing("types/go", "types/go@.", StateNotApplicable, "no command"),
	}
	var b strings.Builder
	WriteCheck(&b, outs, false)
	want := "lint/python: failed (probation) [preset] 0.8s\nE1 bad\n" +
		"lint/go: ok [preset] 0.8s\n" +
		"types/python: missing-tool (probation) [preset] \"mypy\" is not on PATH: mypy .\n" +
		"coverage/go: blocked (probation) [preset] by test/go\n" +
		"types/go: not-applicable [preset] no command\n"
	if b.String() != want {
		t.Fatalf("%q", b.String())
	}
}

func TestCheckVerdictPassesARunRedOnlyInProbation(t *testing.T) {
	red := probing("lint/go", "lint/go@.", StateFailed, "x\n")
	if code, notes := CheckVerdict([]string{"lint"}, []Outcome{red}); code != 0 || len(notes) != 0 {
		t.Fatalf("probation alone: %d %v", code, notes)
	}
	armed := red
	armed.Probation = false
	if code, _ := CheckVerdict([]string{"lint"}, []Outcome{red, armed}); code != 1 {
		t.Fatalf("an armed red lane beside it: %d", code)
	}
	// The second rule is untouched: a kind with nothing to check is red
	// whatever the file says.
	if code, notes := CheckVerdict([]string{"lint", "test"}, []Outcome{red}); code != 1 || len(notes) != 1 {
		t.Fatalf("nothing to check for test: %d %v", code, notes)
	}
}

func TestEditReportSaysARedLaneInProbationWithoutHoldingTheEdit(t *testing.T) {
	var se strings.Builder
	red, notices := EditReport(&se, []Outcome{probing("lint/go", "lint/go@.", StateFailed, "a.go:1: bad\n")}, "aside")
	want := "lint/go: failed (probation)\na.go:1: bad"
	if red || !slices.Equal(notices, []string{want, "aside"}) || se.String() != want+"\n" {
		t.Fatalf("red %v, notices %q, stderr %q", red, notices, se.String())
	}
	se.Reset()
	armed := probing("lint/go", "lint/go@.", StateFailed, "a.go:1: bad\n")
	armed.Probation = false
	red, notices = EditReport(&se, []Outcome{armed}, "aside")
	if !red || len(notices) != 0 || se.String() != "lint/go: failed\na.go:1: bad\n" {
		t.Fatalf("armed: red %v, notices %q, stderr %q", red, notices, se.String())
	}
}

func TestProbationLineNamesTheLanesThatHaveSomethingToCheck(t *testing.T) {
	// The file arms the test lane and nothing else.
	armsOnlyTests := func(j Job) bool { return j.Kind == "test" }
	outs := []Outcome{
		probing("lint/python", "lint/python@.", StateFailed, ""),
		probing("lint/go", "lint/go@.", StateOK, ""),
		probing("types/python", "types/python@.", StateMissingTool, ""),
		probing("types/go", "types/go@.", StateNotApplicable, ""),
		probing("coverage/go", "coverage/go@.", StateUnavailable, ""),
		probing("test/go", "test/go@.", StateFailed, ""),
		probing("lint/go@tools", "lint/go@tools", StateOK, ""),
	}
	want := "probation: lint/go@., lint/go@tools, lint/python@., types/python@. (warn only until a green commit arms them)"
	if got := ProbationLine(outs, armsOnlyTests); got != want {
		t.Fatalf("%q", got)
	}
	if got := ProbationLine(outs, nil); got != "" {
		t.Fatalf("no set: %q", got)
	}
	if got := ProbationLine(outs, func(Job) bool { return true }); got != "" {
		t.Fatalf("every lane armed: %q", got)
	}
}

func TestGreenKeysAreTheLanesThatEndedOK(t *testing.T) {
	outs := []Outcome{
		probing("test/go", "test/go@.", StateOK, ""),
		probing("lint/go", "lint/go@.", StateOK, ""),
		probing("lint/python", "lint/python@.", StateFailed, ""),
		probing("types/go", "types/go@.", StateNotApplicable, ""),
		probing("coverage/go", "coverage/go@.", StateBudget, ""),
		probing("test/go", "test/go@.", StateOK, ""),
	}
	if got := GreenKeys(outs); !slices.Equal(got, []string{"lint/go@.", "test/go@."}) {
		t.Fatalf("%v", got)
	}
}
```

- [ ] **Step 2: rot laufen lassen.** Stubs: `Outcome.Probation` und `RunOptions.Armed` als Felder anlegen, `Fails` gibt `Red(o.State, scope)` zurück, `ProbationLine` gibt `""`, `GreenKeys` gibt `nil`. Run: `go test ./internal/verify -run "TestBlockedIsRed|TestEveryOutcomeSays|TestFailsIsRed|TestWriteCheckMarks|TestCheckVerdictPasses|TestEditReportSays|TestProbationLine|TestGreenKeys" -count=1`. Expected: FAIL mit Assertions, darunter `the test is in probation: probation false, false; want true, true` und `lint/python: failed [preset] 0.8s` statt `failed (probation)`.

- [ ] **Step 3: implementieren.**

  `internal/verify/run.go`, `Outcome` und `RunOptions`:

```go
// Outcome is how one job ended. BlockedBy names the predecessor whose red
// kept it from starting. Probation says its red does not fail the run: the
// lane is not armed, or the lane that blocks it is not.
type Outcome struct {
	Job       Job
	State     State
	Output    string
	Duration  time.Duration
	BlockedBy string
	Probation bool
}
```

```go
// RunOptions is what a run needs from outside. Timeout caps each process, 0
// meaning none; Budget, when positive, caps the whole run from its start.
// Armed says whether a lane's red fails the run; nil arms every lane.
type RunOptions struct {
	Scope           Scope
	MaxParallel     int
	Timeout, Budget time.Duration
	Start           func(child.Spec) child.Result
	Look            func(string) (string, error)
	Now             func() time.Time
	Armed           func(Job) bool
}
```

  In `Run`, im Goroutine-Rumpf, die zwei Zeilen vor `out[i] = o`:

```go
			start := opt.Now()
			o := r.lane(job)
			o.Duration = opt.Now().Sub(start)
			o.Probation = r.probation(job)
			out[i] = o
```

  `inherit` und die neue Methode:

```go
// inherit decides a job by its predecessor: a red one blocks it, and one that
// could not judge leaves it unable to judge as well. The predecessor blocks
// whether or not it is armed -- what it should have written is missing either
// way -- but a block counts as red only where the lane that blocks does.
func (r *runner) inherit(job Job, pred Outcome) (Outcome, bool) {
	if Red(pred.State, r.opt.Scope) {
		return Outcome{Job: job, State: StateBlocked, BlockedBy: pred.Job.Name, Probation: pred.Probation}, true
	}
	switch pred.State {
	case StateUnavailable, StateNotApplicable:
		return Outcome{Job: job, State: pred.State, Probation: r.probation(job)}, true
	case StateBudget, StateMissingTool, StateUnready:
		if r.opt.Scope == ScopeEdit {
			return Outcome{Job: job, State: pred.State, Probation: r.probation(job)}, true
		}
	}
	return Outcome{}, false
}

// probation says whether job's lane is not armed.
func (r *runner) probation(job Job) bool {
	return r.opt.Armed != nil && !r.opt.Armed(job)
}
```

  `internal/verify/report.go`: in `WriteCheck` die Kopfzeile

```go
		state := string(o.State)
		if o.Probation && Red(o.State, ScopeCheck) {
			state += " (probation)"
		}
		head := fmt.Sprintf("%s: %s [%s] ", o.Job.Name, state, origin)
```

  (die Bedingung `o.Output != "" && (Red(o.State, ScopeCheck) || verbose)` darunter bleibt: eine Lane in Probe zeigt ihre Befunde). In `CheckVerdict` die letzte Prüfung:

```go
	if slices.ContainsFunc(outs, func(o Outcome) bool { return Fails(o, ScopeCheck) }) {
		code = 1
	}
```

  In `EditReport` der Schalter:

```go
		switch {
		case Fails(o, ScopeEdit):
			red = true
			fmt.Fprintf(stderr, "%s: %s\n", o.Job.Name, o.State)
			if o.Output != "" {
				fmt.Fprintf(stderr, "%s\n", strings.TrimSuffix(o.Output, "\n"))
			}
		case Red(o.State, ScopeEdit):
			// In probation: said on both streams like a skip, and the edit stands.
			said := fmt.Sprintf("%s: %s (probation)", o.Job.Name, o.State)
			if o.Output != "" {
				said += "\n" + strings.TrimSuffix(o.Output, "\n")
			}
			notices = Skipped(stderr, notices, said)
		case o.State == StateBudget:
			notices = Skipped(stderr, notices, BudgetSkipped(o.Job.Name))
		case o.State == StateMissingTool, o.State == StateUnready:
			notices = Skipped(stderr, notices, skipPrefix+o.Output)
		}
```

  Neu in `report.go`:

```go
// Fails says whether an outcome fails a run: a red state of a lane that is
// not in probation.
func Fails(o Outcome, scope Scope) bool {
	return Red(o.State, scope) && !o.Probation
}

// ProbationLine ends the report of a run with a lane in probation: the keys
// of the lanes armed does not arm, sorted. A lane with nothing to check here
// -- no command, no tests -- is left out: it cannot turn green, so it would
// stand in the line for good. "" without such a lane, and with a nil armed.
func ProbationLine(outs []Outcome, armed func(Job) bool) string {
	if armed == nil {
		return ""
	}
	var keys []string
	for _, o := range outs {
		if o.State != StateNotApplicable && o.State != StateUnavailable && !armed(o.Job) {
			keys = append(keys, LaneKey(o.Job))
		}
	}
	if len(keys) == 0 {
		return ""
	}
	slices.Sort(keys)
	return "probation: " + strings.Join(slices.Compact(keys), ", ") + " (warn only until a green commit arms them)"
}

// GreenKeys are the keys of the lanes that ended ok, sorted, each once: what
// a green run may arm.
func GreenKeys(outs []Outcome) []string {
	var keys []string
	for _, o := range outs {
		if o.State == StateOK {
			keys = append(keys, LaneKey(o.Job))
		}
	}
	slices.Sort(keys)
	return slices.Compact(keys)
}
```

- [ ] **Step 4: grün.** Run: `go test ./internal/verify -count=1`, danach die Invariante `go test ./internal/cli -run "TestCases" -count=1` und `go test ./internal/hooks -count=1`. Expected: PASS, ohne dass ein bestehender Test angefasst wurde (`TestWriteCheckFormatsEveryState` hält die Ausgabe ohne Probe fest).
- [ ] **Step 5: Mutation per `go test -overlay`:**
  - `inherit`: `Probation: pred.Probation` im Blockfall durch `Probation: r.probation(job)` ersetzen: `TestBlockedIsRedOnlyBehindAnArmedLane` (Fälle „the test is in probation“ und „only coverage is in probation“) wird rot.
  - `probation`: `r.opt.Armed != nil &&` entfernen: jeder bestehende Test von `Run` bricht mit einer Nil-Panik ab.
  - `Fails`: `&& !o.Probation` entfernen: `TestFailsIsRedOutsideProbation` wird rot.
  - `WriteCheck`: `&& Red(o.State, ScopeCheck)` entfernen: `TestWriteCheckMarksARedLaneInProbation` wird rot (`lint/go: ok (probation)`).
  - `EditReport`: die Reihenfolge der ersten beiden Fälle vertauschen: `TestEditReportSays…` (zweite Hälfte, scharf) wird rot.
  - `ProbationLine`: `o.State != StateUnavailable &&` entfernen: `TestProbationLineNames…` wird rot.
  - `GreenKeys`: `o.State == StateOK` durch `o.State != StateFailed` ersetzen: `TestGreenKeys…` wird rot.
- [ ] **Step 6: Commit** `feat(verify): let a red lane in probation warn instead of fail`.

---

### Task 4: `loomux check` liest die Datei, `check precommit --arm` schreibt sie

Das Schreiben **und** das Stagen liegen im Befehl, nicht im Shell-Hook: So gibt es genau eine Stelle, die den Teilcommit erkennt und dann weder schreibt noch stagt (Entscheidung 12 der Spec).

**Files:**
- Create: `internal/gitwork/commitindex.go`
- Modify: `internal/cli/check.go` (`checkRun`, die Nahtliste; neu `checkArm`)
- Test: `internal/gitwork/commitindex_test.go` (neu), `internal/cli/check_arming_test.go` (neu)
- Messprotokoll (kein Repo-Inhalt): `<scratchpad>/measure-index.log`

**Interfaces:**
- Consumes: `verify.ReadArmed`, `verify.WriteArmed`, `ArmedSet.Arms|With|Missing|Text`, `verify.RunOptions.Armed`, `verify.ProbationLine`, `verify.GreenKeys`, `verify.ArmedFile` (Tasks 2 und 3); in `internal/gitwork` die vorhandenen `git(root string, arguments ...string) (string, error)` und `gitWith(root string, extra []string, arguments ...string) (string, error)`, die jede git-Ortsvariable der Umgebung herausnehmen.
- Produces:
  - `func CommitIndex(root, inherited string) (index string, whole bool)` in `internal/gitwork` — `inherited` ist das `GIT_INDEX_FILE` des Hooks, `""` wenn ungesetzt. `whole` sagt, dass der Commit den ganzen Index nimmt (der übergebene Index ist der echte oder wird es nach dem Commit); `index` ist dann der Pfad, in den gestagt wird, absolut, `""` für gits eigenen. Bei einem Teilcommit `"", false`.
  - `func Stage(root, index, rel string) error` in `internal/gitwork` — `git add -- <rel>` in diesen Index.
  - das Flag `--arm` von `loomux check precommit`; die Nähte `checkWriteArmed = verify.WriteArmed`, `checkCommitIndex = gitwork.CommitIndex`, `checkStage = gitwork.Stage`; die Ausgabezeilen `armed: <schlüssel>, …`, `not armed: this commit takes only some paths; the next whole commit arms the lanes` und `probation: …` auf stdout, die Meldungen `loomux check: .loomux/armed.toml …; every lane is armed`, `… not written: …` und `… not staged, commit it by hand: …` auf stderr.

- [ ] **Step 0 (Messschritt „Index des Hooks“, vor jedem Code dieses Tasks): messen, welchen Index git dem pre-commit-Hook gibt.** Die Regel der Spec ist Git-Internes und gilt nur, soweit die Messung sie trägt. Per Write ein Skript `<scratchpad>/measure-index.sh` anlegen (kein Heredoc im Bash-Werkzeug; das Skript selbst darf eines enthalten), das
  1. zuerst `unset GIT_DIR GIT_INDEX_FILE GIT_WORK_TREE GIT_PREFIX` ruft (der Lauf kann selbst in einem Hook stehen),
  2. in einem frischen Verzeichnis unter dem Scratchpad ein Repository mit zwei committeten Dateien `a.txt` und `b.txt` anlegt, `user.name`, `user.email`, `commit.gpgsign=false` und `core.hooksPath=hooks` lokal setzt,
  3. als `hooks/pre-commit` einen Hook schreibt, dessen einzige Zeile `printf 'GIT_INDEX_FILE=%s\n' "${GIT_INDEX_FILE-<unset>}" >> measured.log` ist,
  4. `git --version` als erste Zeile nach `measured.log` schreibt und dann nacheinander, jeweils nach einer Änderung an beiden Dateien und mit einer Zeile `--- <aufruf>` davor, ruft: `git add a.txt` und `git commit -m plain`; `git commit -a -m all`; `git commit -m path a.txt`; `git commit --only -m only a.txt`; `git add b.txt` und `git commit --include -m include a.txt`; `git add a.txt` und `git commit --amend -m amend`; `git commit --amend -a -m amend-all`; `git commit --amend -m amend-path a.txt`,
  5. dieselben drei Grundformen (`git commit` nach `git add`, `git commit -a`, `git commit <pfad>`) **aus einem Unterverzeichnis** des Repositorys ruft (`(cd sub && git commit …)`), wobei der Hook zusätzlich sein Arbeitsverzeichnis (`pwd`) und `GIT_DIR` protokolliert,
  6. mit `git worktree add` einen **verlinkten Worktree** anlegt, dort dieselben drei Grundformen ruft und `git rev-parse --absolute-git-dir` protokolliert,
  7. einen **Merge** mit `git merge --no-commit` beginnt und mit `git commit` abschließt,
  8. `measured.log` ausgibt.

  Run: `sh "<scratchpad>/measure-index.sh" "<scratchpad>/mi" > "<scratchpad>/measure-index.log" 2>&1`, danach die Datei lesen. Expected, wie beim Schreiben des Plans mit git 2.54.0.vfs.0.4 gemessen („Belege und Messungen“ Punkt 4): `git commit` und `--amend` zeigen `.git/index` (relativ); `-a`, `--amend -a` und `--include` zeigen `<repo>/.git/index.lock`; `<pfad>`, `--only <pfad>` und `--amend <pfad>` zeigen `<repo>/.git/next-index-<pid>.lock`. Aus einem Unterverzeichnis dieselben drei Werte, der Hook läuft dabei in der Wurzel des Arbeitsbaums (auf die sich das relative `.git/index` bezieht). Im verlinkten Worktree liegen alle drei unter dessen eigenem git-Verzeichnis, das `--absolute-git-dir` nennt: `<repo>/.git/worktrees/<name>/index` (absolut, auch beim gewöhnlichen Commit), `…/index.lock`, `…/next-index-<pid>.lock`. Der mit `git commit` abgeschlossene Merge zeigt `.git/index`. (Die Werte für Unterverzeichnis, Worktree und Merge hat die Durchsicht des Plans am 2026-09-30 mit derselben Git-Version gemessen.) **Nicht gemessen** ist `git commit -p` (`--patch`, `--interactive`): Es braucht eine Konsole; der Implementierer misst es nicht nach und schreibt es als ungemessen in den Kommentar über `CommitIndex`. Die Regel entscheidet dort wie überall am übergebenen Index, nicht an der Aufrufform. Die Git-Version und alle gemessenen Zeilen gehen wörtlich in den Bericht des Tasks und in den Kommentar über `CommitIndex`.

  **Widerspricht die Messung der Regel** — ein Aufruf, der den ganzen Index nimmt, zeigt etwas anderes als `index` oder `index.lock` des Repos; ein Teilcommit zeigt eines von beiden; die Variable ist ungesetzt, wo ein Teilcommit läuft —, dann: **anhalten und berichten**, mit Git-Version und Protokoll. Nicht raten, keine zweite Regel erfinden, keinen Code dieses Tasks schreiben.

- [ ] **Step 0b: die Tests von `CommitIndex` und `Stage` schreiben** (`internal/gitwork/commitindex_test.go`; `repo`, `commit` und `run` stehen in `gitwork_test.go`)

```go
package gitwork

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// aliasOf is another path to dir: a symlink where the platform lets a test
// make one, a junction on Windows. Without either the caller skips.
func aliasOf(t *testing.T, dir string) string {
	t.Helper()
	link := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(dir, link); err == nil {
		return link
	}
	if runtime.GOOS == "windows" {
		if err := exec.Command("cmd", "/c", "mklink", "/J", link, dir).Run(); err == nil {
			return link
		}
	}
	t.Skip("neither a symlink nor a junction can be made here")
	return ""
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// What git hands a pre-commit hook, as measured: the real index, relative,
// for a plain commit; index.lock for -a and --include; next-index-<pid>.lock
// for a commit of paths. Only the first two become the index afterwards.
func TestCommitIndexTellsAWholeCommitFromAPartialOne(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	gitDir := filepath.Join(root, ".git")

	if index, whole := CommitIndex(root, ""); index != "" || !whole {
		t.Fatalf("no index handed in: %q %v", index, whole)
	}
	// Outside a repository too: a run by hand, where nothing is handed in.
	if index, whole := CommitIndex(t.TempDir(), ""); index != "" || !whole {
		t.Fatalf("no index, no repository: %q %v", index, whole)
	}

	realIndex := filepath.Join(gitDir, "index")
	if index, whole := CommitIndex(root, realIndex); index != realIndex || !whole {
		t.Fatalf("the real index: %q %v", index, whole)
	}
	// Relative, as git names it for a plain commit: relative to the directory
	// the hook runs in, and handed on as an absolute path.
	t.Chdir(root)
	index, whole := CommitIndex(root, filepath.Join(".git", "index"))
	if !whole || !filepath.IsAbs(index) || filepath.Base(index) != "index" {
		t.Fatalf("the real index, relative: %q %v", index, whole)
	}

	lock := filepath.Join(gitDir, "index.lock")
	touch(t, lock)
	if index, whole := CommitIndex(root, lock); index != lock || !whole {
		t.Fatalf("index.lock: %q %v", index, whole)
	}
	os.Remove(lock)

	partial := filepath.Join(gitDir, "next-index-4711.lock")
	touch(t, partial)
	if index, whole := CommitIndex(root, partial); index != "" || whole {
		t.Fatalf("a commit of paths: %q %v", index, whole)
	}
	// A copy of the real index is another file, whatever it holds.
	data, _ := os.ReadFile(realIndex)
	os.WriteFile(partial, data, 0o644)
	if _, whole := CommitIndex(root, partial); whole {
		t.Fatal("a copy of the index counts as the index")
	}
	if _, whole := CommitIndex(root, filepath.Join(gitDir, "gone")); whole {
		t.Fatal("an index that is not there counts as whole")
	}
	// The index of another repository is not this one's.
	other := repo(t)
	commit(t, other, "first")
	if _, whole := CommitIndex(root, filepath.Join(other, ".git", "index")); whole {
		t.Fatal("another repository's index counts as whole")
	}
	// No repository to hold it against.
	stray := filepath.Join(t.TempDir(), "index")
	touch(t, stray)
	if _, whole := CommitIndex(filepath.Dir(stray), stray); whole {
		t.Fatal("an index outside every repository counts as whole")
	}
}

// By identity, not by spelling: git answers the long name of its directory,
// the hook's variable may spell the same file through a short name, another
// case or a link.
func TestCommitIndexComparesFilesNotSpellings(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	alias := aliasOf(t, root)
	spelt := filepath.Join(alias, ".git", "index")
	if index, whole := CommitIndex(root, spelt); index != spelt || !whole {
		t.Fatalf("the index through another path: %q %v", index, whole)
	}
}

// A linked worktree has an index of its own, in its own git directory under
// the main one. The main worktree's index is another commit's.
func TestCommitIndexKnowsTheIndexOfALinkedWorktree(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	linked := filepath.Join(t.TempDir(), "linked")
	run(t, root, "worktree", "add", "-q", linked, "-b", "linked")
	own := filepath.Join(root, ".git", "worktrees", "linked", "index")
	if index, whole := CommitIndex(linked, own); index != own || !whole {
		t.Fatalf("the linked worktree's own index: %q %v", index, whole)
	}
	if _, whole := CommitIndex(linked, filepath.Join(root, ".git", "index")); whole {
		t.Fatal("the main worktree's index counts as the linked one's")
	}
	if _, whole := CommitIndex(root, own); whole {
		t.Fatal("the linked worktree's index counts as the main one's")
	}
}

func TestStagePutsAFileIntoTheIndexItIsGiven(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	staged := func(env ...string) string {
		t.Helper()
		cmd := exec.Command("git", "diff", "--cached", "--name-only")
		cmd.Dir = root
		cmd.Env = append(gitenv.Environ(), env...)
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(out))
	}
	touch(t, filepath.Join(root, "f.txt"))
	if err := Stage(root, "", "f.txt"); err != nil || staged() != "f.txt" {
		t.Fatalf("git's own index: %v, staged %q", err, staged())
	}
	// Another index: the file lands there and the real one stays as it was.
	other := filepath.Join(root, ".git", "other-index")
	data, _ := os.ReadFile(filepath.Join(root, ".git", "index"))
	os.WriteFile(other, data, 0o644)
	touch(t, filepath.Join(root, "g.txt"))
	if err := Stage(root, other, "g.txt"); err != nil {
		t.Fatal(err)
	}
	if staged() != "f.txt" || staged("GIT_INDEX_FILE="+other) != "f.txt\ng.txt" {
		t.Fatalf("real %q, other %q", staged(), staged("GIT_INDEX_FILE="+other))
	}
	// A file git ignores is refused, and the refusal comes back.
	touch(t, filepath.Join(root, "h.txt"))
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte("h.txt\n"), 0o644)
	if err := Stage(root, "", "h.txt"); err == nil {
		t.Fatal("an ignored file was staged without a word")
	}
}
```

  (Die Testdatei importiert dazu `github.com/xidus90/loomux/internal/gitenv`.)

- [ ] **Step 1: die Tests des Befehls schreiben** (`internal/cli/check_arming_test.go`)

```go
package cli

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/verify"
)

// armedWorld is goWorld with the file holding text; "" leaves the file out.
// It pins what the process inherits: this suite runs inside loomux's own
// pre-commit gate, whose GIT_INDEX_FILE names loomux's index and would make
// every arming run here a commit of paths. Nothing is staged for real either:
// the world is no repository, and staging is gitwork's to test.
func armedWorld(t *testing.T, text string) string {
	t.Helper()
	t.Setenv("GIT_INDEX_FILE", "")
	oldStage := checkStage
	checkStage = func(string, string, string) error { return nil }
	t.Cleanup(func() { checkStage = oldStage })
	root := goWorld(t)
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if text != "" {
		if err := os.WriteFile(filepath.Join(root, ".loomux", "armed.toml"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func armedText(keys ...string) string { return verify.ArmedSet{}.With(keys...).Text() }

func armedFile(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".loomux", "armed.toml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// failing answers a command that begins with words with 1, every other green.
func failing(words ...string) func(child.Spec) child.Result {
	return func(s child.Spec) child.Result {
		if len(s.Argv) >= len(words) && slices.Equal(s.Argv[:len(words)], words) {
			return child.Result{Code: 1, Stdout: strings.Join(words, " ") + ": bad\n"}
		}
		return green(s)
	}
}

func TestWithoutTheFileARedLaneFailsAndNothingSpeaksOfProbation(t *testing.T) {
	root := armedWorld(t, "")
	stubCheck(t, failing("go", "vet"))
	code, out, errOut := run("check", "precommit", "--root", root)
	if code != 1 || strings.Contains(out, "probation") || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestALaneInProbationWarnsAndAnArmedOneFails(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, failing("go", "vet"))
	code, out, errOut := run("check", "precommit", "--root", root)
	if code != 0 || errOut != "" {
		t.Fatalf("probation: code %d, out %q, err %q", code, out, errOut)
	}
	for _, want := range []string{
		"lint/go: failed (probation) [preset]",
		"go vet: bad",
		"probation: coverage/go@., lint/go@., test/go@. (warn only until a green commit arms them)\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %q", want, out)
		}
	}
	if !strings.HasSuffix(out, "arms them)\n") {
		t.Errorf("the probation line ends the report: %q", out)
	}

	root = armedWorld(t, armedText("lint/go@."))
	code, out, _ = run("check", "precommit", "--root", root)
	if code != 1 || strings.Contains(out, "lint/go: failed (probation)") || !strings.Contains(out, "probation: coverage/go@., test/go@. (") {
		t.Fatalf("armed: code %d, out %q", code, out)
	}
}

func TestBlockedBehindALaneInProbationDoesNotFail(t *testing.T) {
	stubCheck(t, failing("go", "test"))
	root := armedWorld(t, armedText("coverage/go@.", "lint/go@."))
	code, out, _ := run("check", "precommit", "--root", root)
	if code != 0 || !strings.Contains(out, "coverage/go: blocked (probation) [preset] by test/go") {
		t.Fatalf("behind probation: code %d, out %q", code, out)
	}
	root = armedWorld(t, armedText("coverage/go@.", "lint/go@.", "test/go@."))
	code, out, _ = run("check", "precommit", "--root", root)
	if code != 1 || !strings.Contains(out, "coverage/go: blocked [preset] by test/go") {
		t.Fatalf("behind an armed lane: code %d, out %q", code, out)
	}
}

// staging records what --arm stages: root, index and path of each call.
func staging(t *testing.T, answer error) *[]string {
	t.Helper()
	calls := []string{}
	old := checkStage
	checkStage = func(root, index, rel string) error {
		calls = append(calls, root+"|"+index+"|"+rel)
		return answer
	}
	t.Cleanup(func() { checkStage = old })
	return &calls
}

// handed stands in for the index git hands the hook; it records what the
// command passed on from its environment.
func handed(t *testing.T, index string, whole bool) *[]string {
	t.Helper()
	asked := []string{}
	old := checkCommitIndex
	checkCommitIndex = func(root, inherited string) (string, bool) {
		asked = append(asked, root+"|"+inherited)
		return index, whole
	}
	t.Cleanup(func() { checkCommitIndex = old })
	return &asked
}

func TestArmWritesTheGreenLanesOfAGreenRun(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, green)
	staged := staging(t, nil)
	code, out, errOut := run("check", "precommit", "--arm", "--root", root)
	if code != 0 || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	want := armedText("coverage/go@.", "lint/go@.", "test/go@.")
	if got := armedFile(t, root); got != want {
		t.Fatalf("file %q, want %q", got, want)
	}
	// Written, then staged once into git's own index: nothing was handed in.
	if !slices.Equal(*staged, []string{root + "||.loomux/armed.toml"}) {
		t.Fatalf("staged %q", *staged)
	}
	if !strings.Contains(out, "armed: coverage/go@., lint/go@., test/go@.\n") || strings.Contains(out, "probation:") {
		t.Fatalf("out %q", out)
	}
	// A second run has nothing to add and says nothing about arming.
	info, _ := os.Stat(filepath.Join(root, ".loomux", "armed.toml"))
	code, out, _ = run("check", "precommit", "--arm", "--root", root)
	after, _ := os.Stat(filepath.Join(root, ".loomux", "armed.toml"))
	if code != 0 || strings.Contains(out, "armed:") || !after.ModTime().Equal(info.ModTime()) || armedFile(t, root) != want || len(*staged) != 1 {
		t.Fatalf("second run: code %d, out %q, staged %q", code, out, *staged)
	}
	// Without --arm nothing is written, green or not.
	root = armedWorld(t, armedText())
	if code, _, _ = run("check", "precommit", "--root", root); code != 0 || armedFile(t, root) != armedText() {
		t.Fatalf("a run without --arm wrote: %q", armedFile(t, root))
	}
}

// A commit of paths (`git commit <path>`, --only) hands the hook an index
// that does not become the real one: a file staged there would be committed
// and stand in the real index as a staged revert. Such a run arms nothing,
// writes nothing and stages nothing; the next whole commit does.
func TestArmLeavesAPartialCommitAlone(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, green)
	staged := staging(t, nil)
	t.Setenv("GIT_INDEX_FILE", "next-index-4711.lock")
	asked := handed(t, "", false)
	code, out, errOut := run("check", "precommit", "--arm", "--root", root)
	if code != 0 || errOut != "" || armedFile(t, root) != armedText() || len(*staged) != 0 {
		t.Fatalf("code %d, err %q, file %q, staged %q", code, errOut, armedFile(t, root), *staged)
	}
	if !slices.Equal(*asked, []string{root + "|next-index-4711.lock"}) {
		t.Fatalf("the command did not pass its GIT_INDEX_FILE on: %q", *asked)
	}
	want := "not armed: this commit takes only some paths; the next whole commit arms the lanes\n" +
		"probation: coverage/go@., lint/go@., test/go@. (warn only until a green commit arms them)\n"
	if !strings.HasSuffix(out, want) || strings.Contains(out, "\narmed: ") {
		t.Fatalf("out %q", out)
	}
	// With nothing to arm the index is not even asked about.
	root = armedWorld(t, armedText("coverage/go@.", "lint/go@.", "test/go@."))
	*asked = nil
	if code, out, _ = run("check", "precommit", "--arm", "--root", root); code != 0 || len(*asked) != 0 || strings.Contains(out, "not armed") {
		t.Fatalf("nothing to arm: code %d, asked %q, out %q", code, *asked, out)
	}
}

// Under `git commit -a` the index is index.lock, which becomes the real one:
// the file is staged into exactly the index the hook was handed.
func TestArmStagesIntoTheIndexTheHookWasHanded(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, green)
	staged := staging(t, nil)
	handed(t, "/repo/.git/index.lock", true)
	if code, _, errOut := run("check", "precommit", "--arm", "--root", root); code != 0 || errOut != "" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if !slices.Equal(*staged, []string{root + "|/repo/.git/index.lock|.loomux/armed.toml"}) {
		t.Fatalf("staged %q", *staged)
	}
}

// A file that cannot be staged is said; the lanes are armed on disk, and the
// gate's code stands.
func TestArmSaysAFileItCannotStage(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, green)
	staging(t, errors.New("the path is ignored"))
	code, out, errOut := run("check", "precommit", "--arm", "--root", root)
	if code != 0 || errOut != "loomux check: .loomux/armed.toml not staged, commit it by hand: the path is ignored\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if armedFile(t, root) != armedText("coverage/go@.", "lint/go@.", "test/go@.") || !strings.Contains(out, "armed: coverage/go@., lint/go@., test/go@.\n") {
		t.Fatalf("file %q, out %q", armedFile(t, root), out)
	}
}

// What stands in the repository comes from a commit that went through: a
// red run arms no lane, not even one that was green in it.
func TestArmWritesNothingWhenTheRunIsRed(t *testing.T) {
	before := armedText("lint/go@.")
	root := armedWorld(t, before)
	stubCheck(t, failing("go", "vet"))
	staged := staging(t, nil)
	code, out, _ := run("check", "precommit", "--arm", "--root", root)
	if code != 1 || armedFile(t, root) != before || strings.Contains(out, "armed:") || len(*staged) != 0 {
		t.Fatalf("code %d, file %q, out %q, staged %q", code, armedFile(t, root), out, *staged)
	}
	if !strings.Contains(out, "test/go: ok") {
		t.Fatalf("the world must have a green lane the run did not arm: %q", out)
	}
}

func TestArmLeavesARedLaneInProbation(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, failing("go", "vet"))
	code, out, _ := run("check", "precommit", "--arm", "--root", root)
	if code != 0 || armedFile(t, root) != armedText("coverage/go@.", "test/go@.") {
		t.Fatalf("code %d, file %q", code, armedFile(t, root))
	}
	if !strings.Contains(out, "armed: coverage/go@., test/go@.\n") || !strings.Contains(out, "probation: lint/go@. (") {
		t.Fatalf("out %q", out)
	}
}

// The line names what this run armed, not what the file holds afterwards: a
// lane armed before is no news.
func TestArmNamesOnlyTheLanesItAdded(t *testing.T) {
	root := armedWorld(t, armedText("lint/go@."))
	stubCheck(t, green)
	code, out, _ := run("check", "precommit", "--arm", "--root", root)
	if code != 0 || armedFile(t, root) != armedText("coverage/go@.", "lint/go@.", "test/go@.") {
		t.Fatalf("code %d, file %q", code, armedFile(t, root))
	}
	if !strings.Contains(out, "\narmed: coverage/go@., test/go@.\n") || strings.Contains(out, "lint/go@.,") {
		t.Fatalf("out %q", out)
	}
}

func TestArmWithoutTheFileWritesNone(t *testing.T) {
	root := armedWorld(t, "")
	stubCheck(t, green)
	code, out, _ := run("check", "precommit", "--arm", "--root", root)
	if _, err := os.Stat(filepath.Join(root, ".loomux", "armed.toml")); code != 0 || err == nil || strings.Contains(out, "armed:") {
		t.Fatalf("code %d, stat %v, out %q", code, err, out)
	}
}

func TestArmBelongsToThePrecommitProfile(t *testing.T) {
	root := armedWorld(t, armedText())
	seen := stubCheck(t, green)
	for _, request := range []string{"all", "lint", "stop"} {
		code, _, errOut := run("check", request, "--arm", "--root", root)
		if code != 2 || errOut != "loomux check: --arm belongs to the precommit profile\n" {
			t.Errorf("%s: code %d, err %q", request, code, errOut)
		}
	}
	if len(*seen) != 0 || armedFile(t, root) != armedText() {
		t.Fatalf("a refused call ran or wrote: %v", *seen)
	}
}

func TestAnUnreadableFileFailsARedLaneAndIsNotWrittenOver(t *testing.T) {
	const broken = "<<<<<<< HEAD\narmed = []\n=======\n"
	root := armedWorld(t, broken)
	stubCheck(t, failing("go", "vet"))
	code, out, errOut := run("check", "precommit", "--root", root)
	if code != 1 || strings.Contains(out, "probation") ||
		!strings.HasPrefix(errOut, "loomux check: .loomux/armed.toml is no TOML") || !strings.HasSuffix(errOut, "; every lane is armed\n") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	stubCheck(t, green)
	if code, _, _ = run("check", "precommit", "--arm", "--root", root); code != 0 || armedFile(t, root) != broken {
		t.Fatalf("a green --arm wrote over a broken file: code %d, %q", code, armedFile(t, root))
	}
}

// The commit does not hang on the file: a write that fails is said, and the
// gate's code stands.
func TestArmThatCannotWriteKeepsTheGatesCode(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, green)
	old := checkWriteArmed
	checkWriteArmed = func(string, verify.ArmedSet) error { return errors.New("disk full") }
	t.Cleanup(func() { checkWriteArmed = old })
	code, out, errOut := run("check", "precommit", "--arm", "--root", root)
	if code != 0 || errOut != "loomux check: .loomux/armed.toml not written: disk full\n" || strings.Contains(out, "armed:") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if !strings.Contains(out, "probation: coverage/go@., lint/go@., test/go@. (") {
		t.Fatalf("the lanes stay in probation: %q", out)
	}
}
```

- [ ] **Step 2: rot laufen lassen.** Stubs: in `internal/gitwork/commitindex.go` gibt `CommitIndex` immer `"", true` zurück und `Stage` immer `nil`; in `checkRun` das Flag `arm := fs.Bool("arm", false, "…")` anlegen und nicht benutzen (`_ = arm`), die Nähte `checkWriteArmed = verify.WriteArmed`, `checkCommitIndex = gitwork.CommitIndex`, `checkStage = gitwork.Stage` anlegen. Run: `go test ./internal/gitwork -run "TestCommitIndex|TestStagePuts" -count=1` und `go test ./internal/cli -run "TestWithoutTheFileARedLane|TestALaneInProbation|TestBlockedBehind|TestArm|TestAnUnreadableFileFails" -count=1`. Expected: FAIL mit Assertions, darunter `a commit of paths: "" true` in `TestCommitIndexTellsAWholeCommitFromAPartialOne`, `probation: code 1` in `TestALaneInProbationWarnsAndAnArmedOneFails` und `all: code 0` in `TestArmBelongsToThePrecommitProfile`; `TestWithoutTheFileARedLaneFails…` und `TestArmWithoutTheFileWritesNone` sind schon grün.

- [ ] **Step 3: implementieren.** `internal/gitwork/commitindex.go` (die gemessenen acht Zeilen und die Git-Version aus Step 0 gehen in den Kommentar, an die Stelle der hier stehenden):

```go
package gitwork

import (
	"os"
	"path/filepath"
	"strings"
)

// CommitIndex judges the index git handed a pre-commit hook at root.
// inherited is the hook's GIT_INDEX_FILE, "" when it has none. whole says
// that the commit takes the whole index: the index handed in is the real one
// or becomes it once the commit is made. index is then where a hook stages
// into, absolute, and "" for git's own. For a commit of paths both are zero:
// what a hook stages there is committed, while the real index keeps the old
// entry as a staged revert.
//
// This is git's inner working, so it is measured and not derived. With git
// 2.54.0.vfs.0.4 a pre-commit hook sees:
//
//	git commit                   .git/index (relative to the hook's directory)
//	git commit --amend           .git/index
//	git commit -a                <git dir>/index.lock
//	git commit --amend -a        <git dir>/index.lock
//	git commit --include <path>  <git dir>/index.lock
//	git commit <path>            <git dir>/next-index-<pid>.lock
//	git commit --only <path>     <git dir>/next-index-<pid>.lock
//	git commit --amend <path>    <git dir>/next-index-<pid>.lock
//
// From a subdirectory the three are the same, and the hook runs at the top of
// the working tree. In a linked worktree <git dir> is that worktree's own,
// <main git dir>/worktrees/<name>, and its plain commit names the index by
// an absolute path. A merge concluded by git commit hands in .git/index.
// Not measured: git commit -p, which needs a console; it is judged by the
// index it hands in, as every form is.
//
// Only index and index.lock of root's own git directory count, and they are
// compared as files, not as spellings: git answers the long name of its
// directory, the variable may carry a short name, another case or a link. An
// index that is not there, one of another repository, and a root no
// repository covers are all a commit of paths to the caller: nothing is
// written that could land in the wrong place.
func CommitIndex(root, inherited string) (index string, whole bool) {
	if inherited == "" {
		return "", true
	}
	index = inherited
	if !filepath.IsAbs(index) {
		// git names it relative to the directory it runs the hook in.
		wd, _ := os.Getwd()
		index = filepath.Join(wd, index)
	}
	given, err := os.Stat(index)
	if err != nil {
		return "", false
	}
	out, err := git(root, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", false
	}
	dir := strings.TrimSpace(out)
	for _, name := range []string{"index", "index.lock"} {
		if own, err := os.Stat(filepath.Join(dir, name)); err == nil && os.SameFile(given, own) {
			return index, true
		}
	}
	return "", false
}

// Stage puts rel of root into the index a commit hook was handed: index as
// CommitIndex answered it, "" for git's own. git strips GIT_INDEX_FILE from
// every call this package makes, so the index is handed in, not inherited.
func Stage(root, index, rel string) error {
	var extra []string
	if index != "" {
		extra = []string{"GIT_INDEX_FILE=" + index}
	}
	_, err := gitWith(root, extra, "add", "--", rel)
	return err
}
```

  `internal/cli/check.go`: Die Nahtliste bekommt drei Zeilen — `checkWriteArmed = verify.WriteArmed`, `checkCommitIndex = gitwork.CommitIndex`, `checkStage = gitwork.Stage` —, mit dem Satz im Kommentar darüber: „… a file that cannot be written, and the index a commit hook is handed, which no test process has“.

  In `checkRun` hinter `show := …`:

```go
	arm := fs.Bool("arm", false, "after a green run, write every lane that ended ok into .loomux/armed.toml and stage it")
```

  hinter der Prüfung `fs.NArg() > 0`:

```go
	// Only the pre-commit run arms: what the file holds comes from a commit
	// that went through, and no other profile is one.
	if *arm && request != "precommit" {
		fmt.Fprintln(stderr, "loomux check: --arm belongs to the precommit profile")
		return 2
	}
```

  Der Schluss der Funktion, ab `jobs = append(jobs, hooks.WikiGateJobs(…)…)`:

```go
	jobs = append(jobs, hooks.WikiGateJobs(eff, facts, root, kinds)...)
	// A file that does not read arms every lane, and every run says so.
	armed, err := verify.ReadArmed(root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux check: %v\n", err)
	}
	outs := verify.Run(jobs, verify.RunOptions{
		Scope: verify.ScopeCheck, MaxParallel: eff.Config.MaxParallel, Timeout: eff.Config.Timeout,
		Start: checkStart, Look: checkLook, Now: checkNow, Armed: armed.Arms,
	})
	verify.WriteCheck(stdout, outs, *verbose)
	code, notes := verify.CheckVerdict(kinds, outs)
	for _, note := range notes {
		fmt.Fprintln(stdout, note)
	}
	// Armed only by a run that ends green as a whole, and only where the
	// project has the file: without it every lane is armed already.
	if added := armed.Missing(verify.GreenKeys(outs)); *arm && code == 0 && armed.Exists && len(added) > 0 {
		armed = checkArm(root, armed, added, stdout, stderr)
	}
	if line := verify.ProbationLine(outs, armed.Arms); line != "" {
		fmt.Fprintln(stdout, line)
	}
	// A file left behind costs disk, not correctness: the verdict stands.
	if err := verify.CleanCover(root, runID, code == 0); err != nil {
		fmt.Fprintf(stderr, "loomux check: cleaning coverage files: %v\n", err)
	}
	return code
```

  und die neue Funktion:

```go
// checkArm enters the lanes a green pre-commit run found ok, and lays the
// file into the commit under way; it answers the set that holds afterwards.
// Writing and staging sit here and not in the hook's script, so that one
// place knows a commit of paths and then does neither: staged into the index
// such a commit hands its hook, the file would be committed while the real
// index kept the old entry as a staged revert. The commit hangs on nothing
// here: a file that cannot be written or staged is said, and the verdict
// stands.
func checkArm(root string, armed verify.ArmedSet, added []string, stdout, stderr io.Writer) verify.ArmedSet {
	index, whole := checkCommitIndex(root, os.Getenv("GIT_INDEX_FILE"))
	if !whole {
		fmt.Fprintln(stdout, "not armed: this commit takes only some paths; the next whole commit arms the lanes")
		return armed
	}
	next := armed.With(added...)
	if err := checkWriteArmed(root, next); err != nil {
		fmt.Fprintf(stderr, "loomux check: %s not written: %v\n", verify.ArmedFile, err)
		return armed
	}
	fmt.Fprintf(stdout, "armed: %s\n", strings.Join(added, ", "))
	if err := checkStage(root, index, verify.ArmedFile); err != nil {
		fmt.Fprintf(stderr, "loomux check: %s not staged, commit it by hand: %v\n", verify.ArmedFile, err)
	}
	return next
}
```

  Ohne Datei ist `armed` der Nullwert: `Arms` antwortet immer `true`, `ProbationLine` also `""`, `armed.Exists` ist `false`, und weder die Umgebung noch git wird gefragt. Kein Byte der Ausgabe ändert sich.

- [ ] **Step 4: grün.** Run: `go test ./internal/gitwork -count=1` und `go test ./internal/cli -run "TestCheck|TestWithoutTheFileARedLane|TestALaneInProbation|TestBlockedBehind|TestArm|TestAnUnreadableFileFails|TestCases" -count=1`. Expected: PASS; die aufgezeichneten `check`-Fälle (2a, 3c) unverändert. **Dazu einmal unter der Umgebung eines Hooks:** denselben `internal/cli`-Lauf mit gesetztem `GIT_INDEX_FILE` wiederholen (der Wert ist der absolute Pfad einer Datei, die es nicht gibt, etwa `<scratchpad>/no-index`; in Git Bash als `GIT_INDEX_FILE="<scratchpad>/no-index" go test ./internal/cli -run "TestArm" -count=1`). Expected: PASS. Wird dieser Lauf rot, liest ein Test die geerbte Variable, statt sie selbst zu setzen.
- [ ] **Step 5: Mutation per `go test -overlay`.** In `internal/gitwork` gegen `-run "TestCommitIndex|TestStagePuts"`:
  - `os.SameFile(given, own)` durch `filepath.Clean(index) == filepath.Join(dir, name)` ersetzen: `TestCommitIndexComparesFilesNotSpellings` wird rot.
  - `"index.lock"` aus der Liste entfernen: `TestCommitIndexTells…` (`index.lock`) wird rot.
  - Den Block `if !filepath.IsAbs(index) { … }` entfernen: `TestCommitIndexTells…` („the real index, relative“) wird rot: der Pfad käme relativ zurück, und `Stage` reichte ihn an ein git, das in der Wurzel läuft, nicht im Verzeichnis des Hooks.
  - `if inherited == "" { return "", true }` durch `return "", false` ersetzen: der erste Fall wird rot.
  - In `Stage` `extra` nie setzen: `TestStagePuts…` (zweiter Teil: `g.txt` läge im echten Index) wird rot.

  In `internal/cli` gegen `-run "TestArm|TestALaneInProbation|TestAnUnreadableFileFails"`:
  - `code == 0 &&` aus der Bedingung entfernen: `TestArmWritesNothingWhenTheRunIsRed` wird rot.
  - `armed.Exists &&` entfernen: `TestArmWithoutTheFileWritesNone` wird rot (die Datei entsteht); `TestAnUnreadableFileFails…` wird rot (die kaputte Datei wird überschrieben).
  - `*arm &&` entfernen: der letzte Teil von `TestArmWritesTheGreenLanesOfAGreenRun` („a run without --arm wrote“) wird rot.
  - `request != "precommit"` durch `request == "precommit"` ersetzen: `TestArmBelongsToThePrecommitProfile` wird rot.
  - In `checkArm` den Zweig `if !whole { … }` entfernen: `TestArmLeavesAPartialCommitAlone` wird rot (Datei geschrieben, gestagt).
  - In `checkArm` `index` beim Aufruf von `checkStage` durch `""` ersetzen: `TestArmStagesIntoTheIndexTheHookWasHanded` wird rot.
  - In `checkArm` das `return armed` nach dem Schreibfehler durch ein Weiterlaufen ersetzen: `TestArmThatCannotWriteKeepsTheGatesCode` wird rot (`armed:`-Zeile, fehlende Probe-Zeile).
  - `armed = checkArm(…)` durch den bloßen Aufruf ersetzen: `TestArmWritesTheGreenLanesOfAGreenRun` wird rot (die `probation:`-Zeile nennt gerade scharf gestellte Lanes).
  - `Armed: armed.Arms` aus den `RunOptions` entfernen: `TestALaneInProbationWarnsAndAnArmedOneFails` wird rot.
  - In `checkArm` `strings.Join(added, ", ")` durch `strings.Join(next.Keys, ", ")` ersetzen: `TestArmNamesOnlyTheLanesItAdded` wird rot.
  - In `gitwork.CommitIndex` `git(root, …)` durch einen Vergleich gegen `<root>/.git` ersetzen: `TestCommitIndexKnowsTheIndexOfALinkedWorktree` wird rot.
- [ ] **Step 6: Commit** `feat(cli): arm the green lanes of a green precommit run`.

---

### Task 5: Post-Edit — eine Lane in Probe hält die Bearbeitung nicht

Der Hook blockiert heute bei jeder roten Lane mit Exit 2 (`internal/hooks/post_edit.go:69`, `:161-164`); mit Datei blockiert nur noch eine scharfe Lane (Entscheidung 11 der Spec). Ein PostToolUse-Hook, der mit 0 endet, wird vom Agenten nur über den Kontextkanal des Wirts gehört; stderr liest bei Exit 0 niemand.

**Der Kanal je Wirt — vorhanden, nichts zu bauen** (siehe „Belege und Messungen“ Punkt 2): `RunPostEdit` ruft bei Exit 0 `hosts.WriteContext(env.Host, "PostToolUse", stdout, notices)` (`internal/hooks/post_edit.go:120`).
- Claude Code: `writeClaudeContext` (`internal/hosts/claude.go:83`) schreibt `hookSpecificOutput.additionalContext` mit `hookEventName: "PostToolUse"`.
- Antigravity: `writeAntigravityContext` (`internal/hosts/antigravity.go:82`) schreibt `injectSteps[].ephemeralMessage`; `hosts.Answer` (`internal/hosts/answer.go:59-60`) reicht es bei Exit 0 durch.
- Codex: `writeCodexContext` (`internal/hosts/codex.go:37`) liefert `ErrNoAdapter`, Post-Edit endet dort heute schon mit Exit 1; kein gemessener Vertrag, kein Bau in diesem Plan.

Der Befund einer Lane in Probe kommt in die `notices`, die dieser Kanal trägt: `verify.EditReport` legt ihn seit Task 3 über `verify.Skipped` hinein. Dieser Task reicht nur die scharfe Menge an den Lauf und beweist im Test, dass der Warntext **im ausgegebenen JSON** beider Wirte steht.

**Files:**
- Modify: `internal/hooks/post_edit.go` (`RunPostEdit`, `checkEdit`)
- Test: `internal/hooks/post_edit_test.go`

**Interfaces:**
- Consumes: `verify.ReadArmed`, `ArmedSet.Arms`, `verify.RunOptions.Armed`, `verify.EditReport` (Task 3); `hosts.WriteContext(host Host, event string, w io.Writer, lines []string) error`, `hosts.Answer(host Host, event string, w io.Writer, code int, out []byte, reason string) int`, `hosts.HostAntigravity` (vorhanden).
- Produces: `checkEdit(stderr io.Writer, root, raw, runID string, eff verify.Effective, facts detect.Facts, env EditEnv, armed verify.ArmedSet) (int, []string)` (ein Parameter mehr; einziger Aufrufer ist `RunPostEdit`).

- [ ] **Step 1: die Tests schreiben** (ans Ende von `internal/hooks/post_edit_test.go`)

```go
// redTool answers every tool with a finding.
func redTool(child.Spec) child.Result { return child.Result{Code: 1, Stdout: "a.go:1: bad\n"} }

func editedGoFile(t *testing.T, armed string) (root, payload string) {
	t.Helper()
	root = goProject(t)
	writeWorldFile(t, root, "a.go", "package m\n")
	if armed != "" {
		writeWorldFile(t, root, ".loomux/armed.toml", armed)
	}
	return root, filePayload(t, filepath.Join(root, "a.go"))
}

// The key an edit's lane has is the key the check's lane has: what a green
// commit armed holds for the edit of a file in that area too.
func TestAnEditIsHeldOnlyByAnArmedLane(t *testing.T) {
	root, payload := editedGoFile(t, "")
	if code, _, se, _ := postEdit(t, root, payload, redTool); code != ExitDenied || strings.Contains(se, "probation") {
		t.Fatalf("no file: %d %q", code, se)
	}

	root, payload = editedGoFile(t, "armed = []\n")
	code, so, se, _ := postEdit(t, root, payload, redTool)
	if code != ExitOK || !strings.Contains(se, "lint/go: failed (probation)\n") || !strings.Contains(se, "a.go:1: bad") {
		t.Fatalf("probation: %d %q", code, se)
	}
	if said := editContextOf(t, so); !strings.Contains(said, "lint/go: failed (probation)") || !strings.Contains(said, "a.go:1: bad") {
		t.Fatalf("the model is told nothing: %q", said)
	}

	root, payload = editedGoFile(t, "armed = [\"lint/go@.\"]\n")
	code, so, se, _ = postEdit(t, root, payload, redTool)
	if code != ExitDenied || !strings.Contains(se, "lint/go: failed\n") || so != "" {
		t.Fatalf("armed: %d %q %q", code, so, se)
	}
}

func TestPostEditSaysAnUnreadableArmedFileAndHolds(t *testing.T) {
	root, payload := editedGoFile(t, "armed = 1\n")
	code, _, se, _ := postEdit(t, root, payload, redTool)
	if code != ExitDenied || !strings.Contains(se, "loomux hook post-tool-use: .loomux/armed.toml") || !strings.Contains(se, "every lane is armed") {
		t.Fatalf("%d %q", code, se)
	}
}

// Antigravity reads a hook's stdout at exit 0 as injectSteps: the warning is
// in the message it shows the model, and the adapter hands it on as it is.
func TestAntigravityHearsALaneInProbation(t *testing.T) {
	root := goProject(t)
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = []\n")
	seen := []string{}
	env := editEnv(t, redTool, &seen)
	env.Host = hosts.HostAntigravity
	var so, se bytes.Buffer
	code := RunPostEdit(strings.NewReader(agyCall(t, root, "a.go")), &so, &se, root, env)
	if code != ExitOK {
		t.Fatalf("%d %q", code, se.String())
	}
	var said struct {
		InjectSteps []struct {
			EphemeralMessage string `json:"ephemeralMessage"`
		} `json:"injectSteps"`
	}
	if err := json.Unmarshal(so.Bytes(), &said); err != nil || len(said.InjectSteps) != 1 {
		t.Fatalf("stdout is no injectSteps document: %q %v", so.String(), err)
	}
	if msg := said.InjectSteps[0].EphemeralMessage; !strings.Contains(msg, "lint/go: failed (probation)") || !strings.Contains(msg, "a.go:1: bad") {
		t.Fatalf("the warning is not in the message: %q", msg)
	}
	// What `loomux hook` does with it: exit 0, stdout untouched.
	var out bytes.Buffer
	if got := hosts.Answer(hosts.HostAntigravity, "post-tool-use", &out, code, so.Bytes(), se.String()); got != 0 || out.String() != so.String() {
		t.Fatalf("the adapter changed the answer: %d %q", got, out.String())
	}
}
```

  `editContextOf` (Zeile 961 der Datei) liest `hookSpecificOutput.additionalContext` aus dem stdout-Umschlag von Claude Code, nach den genauen Schlüsseln; `editEnv`, `agyCall`, `goProject`, `postEdit`, `filePayload` stehen in derselben Datei, `writeWorldFile` in `stop_test.go`.

- [ ] **Step 2: rot laufen lassen.** Kein neues Symbol, also kein Stub. Run: `go test ./internal/hooks -run "TestAnEditIsHeldOnly|TestPostEditSaysAnUnreadable|TestAntigravityHears" -count=1`. Expected: FAIL mit `probation: 2 "lint/go: failed\n…"`, im zweiten Test ohne die Meldung zur Datei, im dritten mit Exit 2.
- [ ] **Step 3: implementieren.** In `RunPostEdit` hinter dem Block `eff, err := editLoad(root, facts)`:

```go
	// A file that does not read arms every lane; stderr is where a host
	// looks when the edit is then held.
	armed, err := verify.ReadArmed(root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook post-tool-use: %v\n", err)
	}
```

  Der Aufruf wird `checkEdit(stderr, root, raw, id, eff, facts, fileEnv, armed)`, die Signatur bekommt `armed verify.ArmedSet` als letzten Parameter, und die `RunOptions` in `checkEdit` bekommen `Armed: armed.Arms`. Die Zeile mit dem Blast-Hinweis (`verify.Red(o.State, verify.ScopeEdit)`, Zeile 153) bleibt beim Zustand: Wo eine Lane rot ist, zählt der Befund mehr, in Probe oder nicht.
- [ ] **Step 4: grün.** Run: `go test ./internal/hooks -run "TestPostEdit|TestAnEditIsHeldOnly|TestAntigravityHears" -count=1` und `go test ./internal/cli -run "TestCases" -count=1`. Expected: PASS (die aufgezeichneten `hook-post-tool-use`-Fälle unverändert).
- [ ] **Step 5: Mutation per `go test -overlay`:** `Armed: armed.Arms` in `checkEdit` entfernen: `TestAnEditIsHeldOnlyByAnArmedLane` (Mitte) und `TestAntigravityHearsALaneInProbation` werden rot. Die Meldung `fmt.Fprintf(stderr, …)` im Fehlerzweig entfernen: `TestPostEditSaysAnUnreadableArmedFileAndHolds` wird rot. In `verify.EditReport` (Task 3) im Probe-Fall `notices = Skipped(stderr, notices, said)` durch ein bloßes `fmt.Fprintln(stderr, said)` ersetzen (der Befund stünde nur auf stderr, das bei Exit 0 niemand liest): beide Wirtstests werden rot.
- [ ] **Step 6: Commit** `feat(hooks): let a post-edit lane in probation warn instead of block`.

---

### Task 6: Stop-Hook — nur in Probe rot: nicht blockieren, merken, wiederholen

**Files:**
- Modify: `internal/sessions/state.go` (`Seen`, `SessionState`, `stateFile`, `ReadState`, `WriteState`, neu `LastSeen`)
- Modify: `internal/hooks/stop.go` (`RunStop`, `stopVerdict`)
- Test: `internal/sessions/state_test.go`, `internal/hooks/stop_test.go`

**Interfaces:**
- Consumes: `verify.ReadArmed`, `ArmedSet.Arms`, `verify.Fails`, `verify.ProbationLine`, `verify.WriteCheck` (Tasks 2 und 3); `StopEnv.Now`.
- Produces:
  - `type Seen struct { Tree, Head string; Armed []string; Report string; At time.Time }` in `internal/sessions` (JSON `tree`, `head`, `armed`, `report`, `at`) — `Armed` sind die Schlüssel der Datei, als die Kette lief: In einem Projekt, das `.loomux/` ignoriert, liegt die Datei nicht im Baum, und ein `gate arm` muss den Stand trotzdem entwerten.
  - `SessionState.Seen *Seen` (JSON-Schlüssel `seen`, `omitempty`: eine Zustandsdatei ohne den Stand bleibt Byte für Byte wie heute)
  - `func LastSeen(root string) (Seen, bool)` — der jüngste Stand „gesehen“ aller Sitzungen des Projekts, nach `At`.
  - `func stopVerdict(stderr io.Writer, kinds []string, outs []verify.Outcome, armed func(verify.Job) bool) (code int, warned string)` — `warned` ist der Bericht der nur in Probe roten Lanes, `""` sonst.

- [ ] **Step 1: die Tests der Zustandsdatei schreiben** (`internal/sessions/state_test.go`)

```go
func TestTheSeenStateGoesThroughTheFile(t *testing.T) {
	root := t.TempDir()
	at := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	in := SessionState{Blocks: 2, Base: "abc", Green: "g", Seen: &Seen{Tree: "t1", Head: "h1", Armed: []string{"test/go@."}, Report: "lint/go: failed (probation)\n", At: at}}
	if err := WriteState(root, "s1", in); err != nil {
		t.Fatal(err)
	}
	out := ReadState(root, "s1")
	if out.Blocks != 2 || out.Base != "abc" || out.Green != "g" || out.Seen == nil {
		t.Fatalf("%+v %+v", out, out.Seen)
	}
	got, want := *out.Seen, *in.Seen
	if got.Tree != want.Tree || got.Head != want.Head || got.Report != want.Report || !got.At.Equal(want.At) || !slices.Equal(got.Armed, want.Armed) {
		t.Fatalf("seen %+v, want %+v", got, want)
	}
	raw, _ := os.ReadFile(filepath.Join(root, ".loomux", "state", "hooks", "s1.json"))
	if !strings.HasPrefix(string(raw), `{"base":"abc","blocks":2,"green":"g","seen":{"tree":"t1","head":"h1",`) {
		t.Fatalf("key order: %s", raw)
	}
}

// A state without the seen stand is the file of today, byte for byte, and a
// file written before the stand existed still reads.
func TestAStateWithoutSeenIsTheFileOfToday(t *testing.T) {
	root := t.TempDir()
	if err := WriteState(root, "s1", SessionState{Blocks: 1, Base: "abc", Green: "g"}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(root, ".loomux", "state", "hooks", "s1.json"))
	if string(raw) != `{"base":"abc","blocks":1,"green":"g"}` {
		t.Fatalf("%s", raw)
	}
	if got := ReadState(root, "s1"); got.Seen != nil {
		t.Fatalf("%+v", got.Seen)
	}
}

func TestLastSeenIsTheNewestStandOfAnySession(t *testing.T) {
	root := t.TempDir()
	if _, found := LastSeen(root); found {
		t.Fatal("found a stand where no session wrote")
	}
	t0 := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	write := func(id string, seen *Seen) {
		t.Helper()
		if err := WriteState(root, id, SessionState{Seen: seen}); err != nil {
			t.Fatal(err)
		}
	}
	write("old", &Seen{Tree: "t-old", Head: "h", Report: "old\n", At: t0})
	write("new", &Seen{Tree: "t-new", Head: "h", Report: "new\n", At: t0.Add(time.Hour)})
	write("green", nil)
	// What is no session file is passed by: an end marker, a session's agent
	// directory, a directory named like a file, and a file that is no JSON.
	dir := filepath.Join(root, ".loomux", "state", "hooks")
	os.WriteFile(filepath.Join(dir, "old.ended"), nil, 0o644)
	os.MkdirAll(filepath.Join(dir, "old", "agents"), 0o755)
	os.MkdirAll(filepath.Join(dir, "dir.json"), 0o755)
	os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{"), 0o644)
	got, found := LastSeen(root)
	if !found || got.Tree != "t-new" || got.Report != "new\n" {
		t.Fatalf("%+v %v", got, found)
	}
}
```

  (Imports der Testdatei um `slices`, `strings`, `time`, `os`, `path/filepath` ergänzen, soweit sie fehlen.)

- [ ] **Step 2: die Tests des Hooks schreiben** (ans Ende von `internal/hooks/stop_test.go`)

```go
const probing = "armed = []\n"

// probationWorld is stopWorld with the file; blocks is the counter the
// session starts with.
func probationWorld(t *testing.T, armed string, blocks string) string {
	t.Helper()
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":`+blocks+`}`)
	writeWorldFile(t, root, ".loomux/armed.toml", armed)
	return root
}

// A chain red only in lanes in probation is not green and not a block: the
// turn ends, the base and the green tree stay where they are, the tree is
// remembered, and the row of blocks is over, because the turn end went
// through.
func TestStopEndsTheTurnOnAChainRedOnlyInProbation(t *testing.T) {
	root := probationWorld(t, probing, "2")
	base := stateOf(t, root).Base
	at := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	env := redVet()
	env.Now = func() time.Time { return at }
	code, se := runStop(t, root, s1, env)
	if code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	for _, want := range []string{"lint/go: failed (probation) [preset]", "go vet", "probation: lint/go@., test/go@. (warn only until a green commit arms them)\n"} {
		if !strings.Contains(se, want) {
			t.Errorf("missing %q in %q", want, se)
		}
	}
	tree, err := gitwork.ContentTree(root, os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	state := stateOf(t, root)
	if state.Base != base || state.Green != "" || state.Blocks != 0 {
		t.Fatalf("the run counted as green, or the row of blocks went on: %+v", state)
	}
	if state.Seen == nil || state.Seen.Tree != tree || state.Seen.Head != headOf(t, root) || state.Seen.Report != se {
		t.Fatalf("seen %+v, want tree %s head %s", state.Seen, tree, headOf(t, root))
	}
	// When the chain ran, and which lanes were armed then: none.
	if !state.Seen.At.Equal(at) || len(state.Seen.Armed) != 0 {
		t.Fatalf("seen at %v with %v armed, want %v and none", state.Seen.At, state.Seen.Armed, at)
	}
}

// A row of two blocks, then a turn that ends red only in probation, then a
// block: the counter reads 1, not 3. The gate gives up after three blocks in
// a row, and a turn end that went through is what ends a row.
func TestAProbationOnlyTurnEndsTheRowOfBlocks(t *testing.T) {
	root := probationWorld(t, probing, "2")
	if code, se := runStop(t, root, s1, redVet()); code != ExitOK || stateOf(t, root).Blocks != 0 {
		t.Fatalf("%d %q %+v", code, se, stateOf(t, root))
	}
	// The lane is armed now and the tree has changed: the next turn end is held.
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = [\"lint/go@.\"]\n")
	writeWorldFile(t, root, "a.go", "package a\n\nfunc A() int { return 4444 }\n")
	code, se := runStop(t, root, s1, redVet())
	if code != ExitDenied || stateOf(t, root).Blocks != 1 {
		t.Fatalf("%d %q: blocks %d, want 1", code, se, stateOf(t, root).Blocks)
	}
}

// The same holds for a turn end that only says again what it has seen: a
// counter left over from before is back at 0 afterwards.
func TestSayingTheSeenStandAgainEndsTheRowOfBlocks(t *testing.T) {
	root := probationWorld(t, probing, "0")
	runStop(t, root, s1, redVet())
	state := stateOf(t, root)
	state.Blocks = 2
	if err := sessions.WriteState(root, "s1", state); err != nil {
		t.Fatal(err)
	}
	env, started := countTools(redVet())
	if code, se := runStop(t, root, s1, env); code != ExitOK || started.Load() != 0 || stateOf(t, root).Blocks != 0 || stateOf(t, root).Seen == nil {
		t.Fatalf("%d %q, %d tools, %+v", code, se, started.Load(), stateOf(t, root))
	}
}

// A project that ignores .loomux keeps the file out of the tree: arming a
// lane there changes no tree. The stand remembers which lanes were armed,
// so the turn end after a `gate arm` runs the chain and holds. The bench's
// repository is such a project: it excludes all of .loomux.
func TestArmingBetweenTwoTurnEndsRunsTheChainAgain(t *testing.T) {
	root := probationWorld(t, probing, "0")
	before, err := gitwork.ContentTree(root, os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if code, se := runStop(t, root, s1, redVet()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = [\"lint/go@.\"]\n")
	if after, _ := gitwork.ContentTree(root, os.TempDir()); after != before {
		t.Fatalf("the world must keep the file out of the tree: %s %s", before, after)
	}
	env, started := countTools(redVet())
	code, se := runStop(t, root, s1, env)
	if code != ExitDenied || started.Load() == 0 || !strings.Contains(se, "lint/go: failed [preset]") {
		t.Fatalf("%d, %d tools, %q", code, started.Load(), se)
	}
}

// A file that is gone arms every lane, and one that does not read does too:
// neither lets the turn end on what was seen while lanes were in probation.
func TestLosingTheFileRunsTheChainAgain(t *testing.T) {
	for name, write := range map[string]func(root string){
		"gone":       func(root string) { os.Remove(filepath.Join(root, ".loomux", "armed.toml")) },
		"unreadable": func(root string) { writeWorldFile(t, root, ".loomux/armed.toml", "armed = 1\n") },
	} {
		root := probationWorld(t, probing, "0")
		runStop(t, root, s1, redVet())
		write(root)
		env, started := countTools(redVet())
		if code, se := runStop(t, root, s1, env); code != ExitDenied || started.Load() == 0 {
			t.Errorf("%s: %d, %d tools, %q", name, code, started.Load(), se)
		}
	}
}

// The nearest wrong neighbour: the same red lane, armed. It holds the turn
// and counts, with or without other lanes in probation beside it.
func TestStopStillHoldsAnArmedRedLane(t *testing.T) {
	root := probationWorld(t, "armed = [\"lint/go@.\"]\n", "1")
	code, se := runStop(t, root, s1, redVet())
	if code != ExitDenied || !strings.Contains(se, "lint/go: failed [preset]") || strings.Contains(se, "probation") {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Blocks != 2 || state.Seen != nil {
		t.Fatalf("%+v", state)
	}
}

func TestStopStartsNoToolOnATreeItHasSeen(t *testing.T) {
	root := probationWorld(t, probing, "0")
	_, first := runStop(t, root, s1, redVet())
	env, started := countTools(redVet())
	code, again := runStop(t, root, s1, env)
	if code != ExitOK || started.Load() != 0 || again != first {
		t.Fatalf("%d, %d tools, %q against %q", code, started.Load(), again, first)
	}
	// Still not green: the third turn end says it again.
	if code, third := runStop(t, root, s1, env); code != ExitOK || third != first || stateOf(t, root).Green != "" {
		t.Fatalf("%d %q", code, third)
	}
	// The same with a lane armed beside the one in probation: the stand holds
	// the lanes it was taken under, and they are the ones the file names now.
	root = probationWorld(t, "armed = [\"test/go@.\"]\n", "0")
	runStop(t, root, s1, redVet())
	if seen := stateOf(t, root).Seen; seen == nil || !slices.Equal(seen.Armed, []string{"test/go@."}) {
		t.Fatalf("seen %+v", seen)
	}
	env, started = countTools(redVet())
	if code, se := runStop(t, root, s1, env); code != ExitOK || started.Load() != 0 {
		t.Fatalf("with a lane armed: %d, %d tools, %q", code, started.Load(), se)
	}
}

func TestStopRunsTheChainAgainWhenTheTreeChanged(t *testing.T) {
	root := probationWorld(t, probing, "0")
	runStop(t, root, s1, redVet())
	writeWorldFile(t, root, "a.go", "package a\n\nfunc A() int { return 4444 }\n")
	env, started := countTools(redVet())
	if code, _ := runStop(t, root, s1, env); code != ExitOK || started.Load() == 0 {
		t.Fatalf("%d, %d tools", code, started.Load())
	}
}

// The tree alone is not the key: the graph lane judges against HEAD, so a
// commit inside the session that leaves the tree alone runs the chain again.
func TestStopRunsTheChainAgainWhenHeadMoved(t *testing.T) {
	root := probationWorld(t, probing, "0")
	runStop(t, root, s1, redVet())
	seen := stateOf(t, root).Seen
	git(t, root, "add", "a.go")
	// The identity on the command line: the bench's repository carries none.
	git(t, root, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "--no-verify", "-m", "third")
	if tree, _ := gitwork.ContentTree(root, os.TempDir()); tree != seen.Tree || headOf(t, root) == seen.Head {
		t.Fatalf("the world must move HEAD and keep the tree: %s %s", tree, headOf(t, root))
	}
	env, started := countTools(redVet())
	if code, _ := runStop(t, root, s1, env); code != ExitOK || started.Load() == 0 {
		t.Fatalf("%d, %d tools", code, started.Load())
	}
}

// .loomux/armed.toml is part of the tree the gate measures -- everything git
// does not ignore, less .loomux/state. A colleague's pull that arms the red
// lane changes the tree, so the remembered stand does not let the turn end.
func TestStopRunsTheChainAgainWhenOnlyTheArmedFileChanged(t *testing.T) {
	root := probationWorld(t, probing, "0")
	// The test bench excludes all of .loomux; a project ignores only its state.
	writeWorldFile(t, root, ".git/info/exclude", "/git.toml\n/faketool.json\n/.loomux/state/\n")
	before, err := gitwork.ContentTree(root, os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if code, se := runStop(t, root, s1, redVet()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = [\"lint/go@.\"]\n")
	after, err := gitwork.ContentTree(root, os.TempDir())
	if err != nil || after == before {
		t.Fatalf("the file is not part of the tree: %s %s %v", before, after, err)
	}
	env, started := countTools(redVet())
	code, se := runStop(t, root, s1, env)
	if code != ExitDenied || started.Load() == 0 || !strings.Contains(se, "lint/go: failed [preset]") {
		t.Fatalf("%d, %d tools, %q", code, started.Load(), se)
	}
}

func TestAGreenRunForgetsWhatWasSeen(t *testing.T) {
	root := probationWorld(t, probing, "0")
	runStop(t, root, s1, redVet())
	writeWorldFile(t, root, "a.go", "package a\n\nfunc A() int { return 4444 }\n")
	code, se := runStop(t, root, s1, greenTools())
	state := stateOf(t, root)
	if code != ExitOK || state.Seen != nil || state.Base != headOf(t, root) || state.Green == "" || strings.Contains(se, "probation") {
		t.Fatalf("%d %q %+v", code, se, state)
	}
}

// Without the file the state file gains no key.
func TestStopWithoutTheFileWritesTheStateOfToday(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".loomux", "state", "hooks", "s1.json"))
	if err != nil || strings.Contains(string(raw), "seen") {
		t.Fatalf("%s %v", raw, err)
	}
}

func TestStopSaysAnUnreadableArmedFile(t *testing.T) {
	root := probationWorld(t, "<<<<<<< HEAD\n", "0")
	code, se := runStop(t, root, s1, redVet())
	if code != ExitDenied || !strings.Contains(se, "loomux hook stop: .loomux/armed.toml is no TOML") || !strings.Contains(se, "every lane is armed") {
		t.Fatalf("%d %q", code, se)
	}
}

// Outside a repository there is no tree to remember: the chain runs at every
// turn end, as it does today.
func TestStopRemembersNothingOutsideARepository(t *testing.T) {
	root := gitWorld(t, "", `{"blocks":0}`)
	writeWorldFile(t, root, ".loomux/armed.toml", probing)
	if code, se := runStop(t, root, s1, redVet()); code != ExitOK || stateOf(t, root).Seen != nil {
		t.Fatalf("%d %q %+v", code, se, stateOf(t, root).Seen)
	}
	env, started := countTools(redVet())
	if code, _ := runStop(t, root, s1, env); code != ExitOK || started.Load() == 0 {
		t.Fatalf("%d, %d tools", code, started.Load())
	}
}

// The verdict's order, on outcomes laid out by hand: an armed red lane holds
// whatever else happened; a budget that ran out and a kind with nothing to
// check leave the turn unjudged, and then nothing is remembered, whatever
// stands in probation beside them.
func TestTheVerdictRemembersOnlyAChainItJudgedWhole(t *testing.T) {
	lane := func(kind string, s verify.State, probation bool) verify.Outcome {
		return verify.Outcome{Job: verify.Job{Name: kind + "/go", Kind: kind, Stack: "go", Area: ".", Origin: "preset"},
			State: s, Output: kind + " said\n", Probation: probation}
	}
	none := func(verify.Job) bool { return false }
	for name, c := range map[string]struct {
		kinds  []string
		outs   []verify.Outcome
		code   int
		warned bool
	}{
		"red only in probation":          {[]string{"lint"}, []verify.Outcome{lane("lint", verify.StateFailed, true)}, ExitOK, true},
		"an armed red lane beside it":    {[]string{"lint", "test"}, []verify.Outcome{lane("lint", verify.StateFailed, true), lane("test", verify.StateFailed, false)}, ExitDenied, false},
		"a budget that ran out beside it": {[]string{"lint", "test"}, []verify.Outcome{lane("lint", verify.StateFailed, true), lane("test", verify.StateBudget, true)}, ExitInternal, false},
		"a kind with nothing to check":   {[]string{"lint", "test"}, []verify.Outcome{lane("lint", verify.StateFailed, true)}, ExitInternal, false},
		"all green in probation":         {[]string{"lint"}, []verify.Outcome{lane("lint", verify.StateOK, true)}, ExitOK, false},
	} {
		var se strings.Builder
		code, warned := stopVerdict(&se, c.kinds, c.outs, none)
		if code != c.code || (warned != "") != c.warned {
			t.Errorf("%s: code %d, warned %q, stderr %q", name, code, warned, se.String())
		}
		// Held by an armed lane: only that lane is said. What stands in
		// probation beside it is not the agent's to fix before the turn ends.
		if c.code == ExitDenied && (!strings.Contains(se.String(), "test/go: failed [preset]") || strings.Contains(se.String(), "lint/go") || strings.Contains(se.String(), "probation")) {
			t.Errorf("%s: stderr %q", name, se.String())
		}
		if c.warned && (warned != se.String() || !strings.Contains(warned, "lint/go: failed (probation) [preset]") || !strings.Contains(warned, "probation: lint/go@. (")) {
			t.Errorf("%s: warned %q, stderr %q", name, warned, se.String())
		}
	}
}
```

  `git(t, root, …)` steht in `worktree_test.go`, `countTools`, `redVet`, `greenTools`, `stateOf`, `headOf`, `runStop`, `s1` in `stop_test.go`.

- [ ] **Step 3: rot laufen lassen.** Stubs: `sessions.Seen` als Typ und `SessionState.Seen` als Feld anlegen, ohne sie zu lesen oder zu schreiben; `LastSeen` gibt `Seen{}, false`; `stopVerdict` bekommt den vierten Parameter und das zweite Ergebnis (`""`), ohne sie zu benutzen. Run: `go test ./internal/sessions -run "TestTheSeenState|TestAStateWithoutSeen|TestLastSeen" -count=1` und `go test ./internal/hooks -run "TestStopEndsTheTurnOnAChainRedOnlyInProbation|TestStopStillHolds|TestStopStartsNoTool|TestStopRunsTheChainAgain|TestAGreenRunForgets|TestStopWithoutTheFileWrites|TestStopSaysAnUnreadable|TestStopRemembersNothing|TestTheVerdictRemembers|TestAProbationOnlyTurn|TestSayingTheSeenStand|TestArmingBetween|TestLosingTheFile" -count=1`. Expected: FAIL mit Assertions, darunter `2 "lint/go: failed …"` in `TestStopEndsTheTurn…` (der Hook blockiert noch); `TestStopWithoutTheFileWritesTheStateOfToday` und `TestAStateWithoutSeenIsTheFileOfToday` sind schon grün.

- [ ] **Step 4: implementieren.**

  `internal/sessions/state.go`:

```go
// Seen is a tree the stop gate found red only in lanes in probation: not
// green, so the base stays, and no block. It is kept so that the same tree
// under the same HEAD starts no tool again and only says Report again. HEAD
// is part of it because the graph lane judges against HEAD. Armed is the
// armed lanes the chain ran under: a project that ignores .loomux keeps the
// file out of the tree, so a lane armed by hand changes no tree and has to
// end the stand by itself. At is when the chain ran, which is how the newest
// stand of several sessions is told.
type Seen struct {
	Tree   string    `json:"tree"`
	Head   string    `json:"head"`
	Armed  []string  `json:"armed"`
	Report string    `json:"report"`
	At     time.Time `json:"at"`
}
```

  `SessionState` bekommt `Seen *Seen` (Kommentar des Typs um einen Satz ergänzen: „Seen is the last tree found red only in lanes in probation, nil without one.“), `stateFile` bekommt als letztes Feld `Seen *Seen \`json:"seen,omitempty"\``; die Felder bleiben alphabetisch. `ReadState` setzt `state.Seen = file.Seen`, `WriteState` setzt `file.Seen = state.Seen`. Dazu:

```go
// LastSeen is the newest stand any session of root left behind, by the time
// its chain ran. Session start reads it for a session that has no state of
// its own yet. A file that does not read is passed by, as ReadState passes it.
func LastSeen(root string) (Seen, bool) {
	dir := filepath.Join(root, filepath.FromSlash(StateDir))
	entries, _ := os.ReadDir(dir)
	var newest Seen
	found := false
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		// A directory named like a session file fails here and is passed by.
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var file stateFile
		if json.Unmarshal(raw, &file) != nil || file.Seen == nil {
			continue
		}
		if !found || file.Seen.At.After(newest.At) {
			newest, found = *file.Seen, true
		}
	}
	return newest, found
}
```

  `internal/hooks/stop.go`, `stopVerdict`:

```go
// stopVerdict writes the lanes that fail the run and says what they mean for
// the turn: red holds it, a budget that ran out or a kind with nothing to
// check leaves it unjudged, anything else passes. A chain red only in lanes
// in probation passes too, and warned is what those lanes reported: the
// caller neither moves the base over it nor counts a block.
func stopVerdict(stderr io.Writer, kinds []string, outs []verify.Outcome, armed func(verify.Job) bool) (code int, warned string) {
	var red, held []verify.Outcome
	for _, o := range outs {
		switch {
		case verify.Fails(o, verify.ScopeCheck):
			red = append(red, o)
		case verify.Red(o.State, verify.ScopeCheck):
			held = append(held, o)
		}
	}
	if len(red) > 0 {
		// Only the lanes that fail: what reaches the agent's context is what
		// it has to fix.
		verify.WriteCheck(stderr, red, false)
		return ExitDenied, ""
	}
	if slices.ContainsFunc(outs, func(o verify.Outcome) bool { return o.State == verify.StateBudget }) {
		fmt.Fprintln(stderr, "loomux hook stop: not everything was verified; raise --budget or shrink the stop profile")
		return ExitInternal, ""
	}
	if _, notes := verify.CheckVerdict(kinds, outs); len(notes) > 0 {
		for _, note := range notes {
			fmt.Fprintf(stderr, "loomux hook stop: %s\n", note)
		}
		fmt.Fprintln(stderr, "loomux hook stop: nothing was verified for these kinds; the base stays")
		return ExitInternal, ""
	}
	if len(held) > 0 {
		var report strings.Builder
		verify.WriteCheck(&report, held, false)
		fmt.Fprintln(&report, verify.ProbationLine(outs, armed))
		fmt.Fprint(stderr, report.String())
		return ExitOK, report.String()
	}
	return ExitOK, ""
}
```

  (`strings` in die Importe.) In `RunStop` bekommt der Schalter hinter `stopTrees` einen Fall, direkt hinter dem Fall `tree == state.Green && !headMoved`:

```go
	case state.Seen != nil && tree == state.Seen.Tree && head == state.Seen.Head &&
		armed.Exists && slices.Equal(state.Seen.Armed, armed.Keys):
		// The same tree under the same HEAD, with the same lanes armed, was
		// red only in lanes in probation: no tool starts, and what they found
		// is said again. The armed lanes are asked by themselves, because a
		// project that ignores .loomux keeps the file out of the tree; a file
		// that is gone or does not read arms every lane and ends the stand. Not a
		// green pass -- the base stays -- but the turn end goes through, and
		// that ends a row of blocks. Written only when that changes something,
		// as the green arm above.
		fmt.Fprint(stderr, state.Seen.Report)
		if !stuck && state.Blocks != 0 {
			passed()
		}
		return end(ExitOK)
```

  Die Datei wird **vor** dem Schalter gelesen, direkt hinter der Zeile `headMoved := …`, weil der neue Fall sie braucht:

```go
	// Read before the tree is judged: the stand below holds only under the
	// lanes that were armed when it was taken. What is wrong with the file
	// is said where a chain runs, not at a turn end that starts none.
	armed, armedErr := verify.ReadArmed(root)
```

  Hinter dem Schalter, vor `runID := …`:

```go
	// A file that does not read arms every lane, and every chain says so.
	if armedErr != nil {
		say("%v", armedErr)
	}
```

  Die `RunOptions` bekommen `Armed: armed.Arms`. Der Schluss:

```go
	code, warned := stopVerdict(stderr, kinds, outs, armed.Arms)
	// A chain with findings keeps its coverage files for whoever looks into
	// it, in probation or not.
	if err := verify.CleanCover(root, runID, code == ExitOK && warned == ""); err != nil {
		say("cleaning coverage files: %v", err)
	}
	switch {
	case code == ExitDenied:
		countBlock()
		return ExitDenied
	case code == ExitOK && warned != "":
		// Not green: the base and the green tree stay. The turn end goes
		// through all the same, which ends a row of blocks as a green pass
		// does. Outside a repository there is no tree to remember it by.
		if tree != "" {
			state.Seen = &sessions.Seen{Tree: tree, Head: head, Armed: slices.Clone(armed.Keys), Report: warned, At: env.Now()}
		}
		passed()
	case code == ExitOK:
		state.Base, state.Green, state.Seen = head, tree, nil
		passed()
	}
	return end(code)
```

  `passed()` ist die vorhandene Hilfsfunktion von `RunStop` (`internal/hooks/stop.go:135-140`): Sie setzt `state.Blocks = 0`, außer ein nicht räumbarer Befund eines Subagenten hält die Runde (`stuck`), und speichert. Den Kommentar über `RunStop` um einen Satz ergänzen: „A chain red only in lanes in probation ends the turn with 0 as well and ends a row of blocks, without moving the base, and is remembered by its tree, its HEAD and the lanes that were armed.“

- [ ] **Step 5: grün.** Run: `go test ./internal/sessions ./internal/hooks -count=1` und `go test ./internal/cli -run "TestCases" -count=1`. Expected: PASS; die aufgezeichneten `hook-stop`-Fälle (2c) unverändert. Coverage von `internal/sessions` und `stop.go` bei 100 % je Funktion.
- [ ] **Step 6: Mutation per `go test -overlay`** gegen `-run "TestStop|TestAGreenRun|TestAProbationOnlyTurn|TestSayingTheSeenStand|TestTheVerdictRemembers|TestArmingBetween|TestLosingTheFile"`:
  - Im neuen Fall `&& head == state.Seen.Head` entfernen: `TestStopRunsTheChainAgainWhenHeadMoved` wird rot.
  - Im neuen Fall `tree == state.Seen.Tree &&` entfernen: `TestStopRunsTheChainAgainWhenTheTreeChanged` wird rot.
  - Im neuen Fall `slices.Equal(state.Seen.Armed, armed.Keys)` durch `true` ersetzen: `TestArmingBetweenTwoTurnEndsRunsTheChainAgain` wird rot.
  - Im neuen Fall `armed.Exists &&` entfernen: `TestLosingTheFileRunsTheChainAgain` wird rot (beide Fälle: ohne Datei und unlesbar sind die Schlüssel leer wie beim gemerkten Stand).
  - Beim Merken `Armed: slices.Clone(armed.Keys)` weglassen: `TestSayingTheSeenStandAgain…` und `TestStopStartsNoToolOnATreeItHasSeen` bleiben grün (leer gleich leer); rot wird ein Lauf, der mit einer scharfen Lane merkt — dafür steht der folgende Fall in `TestStopStartsNoToolOnATreeItHasSeen`.
  - Beim Merken `At: env.Now()` weglassen: `TestStopEndsTheTurnOnAChainRedOnlyInProbation` wird rot.
  - In `stopVerdict` im Zweig `len(red) > 0` `verify.WriteCheck(stderr, append(red, held...), false)` schreiben: `TestTheVerdictRemembersOnlyAChainItJudgedWhole` („an armed red lane beside it“) wird rot.
  - `case code == ExitOK && warned != "":` entfernen (der Lauf zählt als grün): `TestStopEndsTheTurnOnAChainRedOnlyInProbation` wird rot (`state.Green != ""`).
  - In diesem Fall `passed()` durch `save()` ersetzen (der Zähler bleibt stehen): `TestAProbationOnlyTurnEndsTheRowOfBlocks` wird rot (Zähler 3 statt 1).
  - Im Fall „gesehen“ die drei Zeilen `if !stuck && state.Blocks != 0 { passed() }` entfernen: `TestSayingTheSeenStandAgainEndsTheRowOfBlocks` wird rot.
  - `if tree != ""` entfernen: `TestStopRemembersNothingOutsideARepository` wird rot.
  - In `stopVerdict` `verify.Fails` durch `verify.Red(o.State, verify.ScopeCheck)` ersetzen: `TestStopEndsTheTurn…` wird rot (Exit 2).
  - In `stopVerdict` den Block `if len(held) > 0 { … }` vor die Budget-Prüfung ziehen: `TestTheVerdictRemembersOnlyAChainItJudgedWhole` („a budget that ran out beside it“) wird rot.
  - `state.Seen = nil` aus dem grünen Fall entfernen: `TestAGreenRunForgetsWhatWasSeen` wird rot.
  - In `LastSeen` `file.Seen.At.After(newest.At)` durch `file.Seen.At.Before(newest.At)` ersetzen: `TestLastSeenIsTheNewestStandOfAnySession` wird rot.
- [ ] **Step 7: Commit** `feat(hooks): end the turn on a chain red only in probation and remember the tree`.

---

### Task 7: `loomux gate status|arm|disarm`

**Files:**
- Create: `internal/hooks/lanestates.go`, `internal/cli/gate.go`
- Modify: `internal/gitwork/gitwork.go` (neu `IgnoredPath`)
- Modify: `internal/cli/commands.go` (eine Zeile `"gate": gateCommand,`, alphabetisch zwischen `flow` und `graph`)
- Test: `internal/gitwork/gitwork_test.go`, `internal/hooks/lanestates_test.go`, `internal/cli/gate_test.go`

**Interfaces:**
- Consumes: `verify.ReadArmed`, `verify.WriteArmed`, `ArmedSet.With|Without`, `verify.LaneKey`, `verify.Plan` über die vorhandene Naht `stopPlan`, `hooks.WikiGateJobs`, `editLoad` (Paket `hooks`), `store.WiringPath`, `hosts.FindRoot`, `parseInterspersed` (`internal/cli/interspersed.go`).
- Produces:
  - `func GateLanes(root string) ([]string, error)` in `internal/hooks` — die Schlüssel aller Lanes, die das Tor des Projekts fahren kann, sortiert.
  - `func IgnoredPath(root, rel string) bool` in `internal/gitwork` — ob git `rel` in `root` ignoriert; `false` außerhalb eines Repositorys und für eine versionierte Datei.
  - `type LaneStates struct { Armed, Probation, Orphans []string; Ignored bool }` — `Ignored` sagt, dass git die Datei ignoriert: Sie erreicht dann nie einen Commit und gilt nur auf diesem Rechner.
  - `func ReadLaneStates(root string) (states LaneStates, exists bool, err error)` — ohne Datei `exists == false`, und die Lanes werden gar nicht erst geplant.
  - `func gateCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int` und die Naht `var gateWrite = verify.WriteArmed`
  - die Warnung von `gate status` auf stderr: `loomux gate status: .loomux/armed.toml is ignored by git: it reaches no commit and holds on this machine only`
  - Befehle: `loomux gate status [--root <dir>]`, `loomux gate arm <lane>… [--root <dir>]`, `loomux gate disarm <lane>…|--all [--root <dir>]`.

**Wie die Spec es festlegt** (Abschnitt „Die Befehle des Menschen“):
- `gate arm <lane>` ohne Datei schreibt nichts und sagt `no .loomux/armed.toml: every lane is armed` (Exit 0): Ohne Datei ist ohnehin alles scharf, und die Datei mit nur dieser Lane anzulegen, würfe jede andere in die Probe.
- Ein unbekannter Schlüssel ist bei `gate arm` zuerst ein Fehler (Exit 1), mit und ohne Datei: Wer sich vertippt, hört es, statt „alles scharf“ zu lesen.
- `gate disarm <lane>` ohne Datei legt die Datei an, mit jeder anderen Lane als scharf eingetragen: Vorher waren alle scharf, danach sind es alle bis auf die genannte. Hier muss die Lane existieren.

**Festlegungen, die die Spec offen lässt:**
- `gate disarm <lane>` mit Datei entfernt den Eintrag, auch einen verwaisten; ein Schlüssel, der nicht drinsteht, wird genannt (`<lane>: was not armed`) und ist kein Fehler.
- Eine unlesbare Datei ist für `arm` und `disarm <lane>` ein Fehler (Exit 1, nichts geschrieben): Der Mensch löst erst den Konflikt. `disarm --all` schreibt sie neu.

- [ ] **Step 1: die Tests der Lanes schreiben** (`internal/hooks/lanestates_test.go`)

```go
package hooks

import (
	"errors"
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/verify"
)

func goLanesWorld(t *testing.T) string {
	t.Helper()
	root := goProject(t)
	writeWorldFile(t, root, "a_test.go", "package m\n")
	return root
}

// A lane with nothing to run -- types/go has no command -- is no lane of the
// gate, one with nothing to check -- a module without tests -- is none
// either, and the graph lane is one only where a graph was built.
func TestGateLanesAreTheLanesWithSomethingToRun(t *testing.T) {
	root := goLanesWorld(t)
	got, err := GateLanes(root)
	if err != nil || !slices.Equal(got, []string{"coverage/go@.", "lint/go@.", "test/go@."}) {
		t.Fatalf("%v %v", got, err)
	}
	writeWorldFile(t, root, ".loomux/state/graph/wiring.json", "{}")
	got, err = GateLanes(root)
	if err != nil || !slices.Equal(got, []string{"coverage/go@.", "graph/go@.", "lint/go@.", "test/go@."}) {
		t.Fatalf("with a graph: %v %v", got, err)
	}
	// No test file: the test lane finds no tests and coverage has nothing to
	// measure. Neither would ever leave the list.
	got, err = GateLanes(goProject(t))
	if err != nil || !slices.Equal(got, []string{"lint/go@."}) {
		t.Fatalf("without tests: %v %v", got, err)
	}
}

// The wiki's lane and a project lane are built from kind, stack and area as
// every other: lint/wiki@. and lint/project@., whatever their names print.
func TestGateLanesNameTheWikiAndTheProjectLanes(t *testing.T) {
	root, _ := wikiProject(t, "[verify.project]\nlint = \"echo hi\"\n", cleanPage)
	got, err := GateLanes(root)
	if err != nil || !slices.Equal(got, []string{"lint/project@.", "lint/wiki@."}) {
		t.Fatalf("%v %v", got, err)
	}
}

func TestGateLanesReportAConfigAndAPlanThatFail(t *testing.T) {
	root := goLanesWorld(t)
	writeWorldFile(t, root, ".loomux/config.toml", "[verify\n")
	if _, err := GateLanes(root); err == nil {
		t.Fatal("a config that does not parse went through")
	}
	root = goLanesWorld(t)
	old := stopPlan
	stopPlan = func(verify.Effective, verify.Request, verify.PlanEnv) ([]verify.Job, error) {
		return nil, errors.New("no plan")
	}
	t.Cleanup(func() { stopPlan = old })
	if _, err := GateLanes(root); err == nil || err.Error() != "no plan" {
		t.Fatalf("%v", err)
	}
}

func TestLaneStatesSortEveryLaneAndEveryEntry(t *testing.T) {
	root := goLanesWorld(t)
	if states, exists, err := ReadLaneStates(root); exists || err != nil || len(states.Probation) != 0 {
		t.Fatalf("no file: %+v %v %v", states, exists, err)
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = [\"lint/go@.\", \"lint/python@.\"]\n")
	states, exists, err := ReadLaneStates(root)
	if !exists || err != nil ||
		!slices.Equal(states.Armed, []string{"lint/go@."}) ||
		!slices.Equal(states.Probation, []string{"coverage/go@.", "test/go@."}) ||
		!slices.Equal(states.Orphans, []string{"lint/python@."}) {
		t.Fatalf("%+v %v %v", states, exists, err)
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = 1\n")
	if _, _, err := ReadLaneStates(root); err == nil {
		t.Fatal("an unreadable file went through")
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = []\n")
	writeWorldFile(t, root, ".loomux/config.toml", "[verify\n")
	if _, _, err := ReadLaneStates(root); err == nil {
		t.Fatal("lanes that cannot be planned went through")
	}
}

// A project that ignores .loomux, or the file, never gets it into a commit:
// what it says holds on this machine only, and the states say so. Ignoring
// the state directory alone, as init sets a project up, is no such case.
func TestLaneStatesSayWhenGitIgnoresTheFile(t *testing.T) {
	for ignore, want := range map[string]bool{
		".loomux/\n":            true,
		".loomux/armed.toml\n":  true,
		"*.toml\n":              true,
		"/.loomux/state/\n":     false,
		"":                      false,
	} {
		root := goLanesWorld(t)
		gitInit(t, root)
		writeWorldFile(t, root, ".gitignore", ignore)
		writeWorldFile(t, root, ".loomux/armed.toml", "armed = []\n")
		states, exists, err := ReadLaneStates(root)
		if err != nil || !exists || states.Ignored != want {
			t.Errorf("%q: ignored %v, want %v (%v)", ignore, states.Ignored, want, err)
		}
	}
	// No repository: nothing ignores anything.
	root := goLanesWorld(t)
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = []\n")
	if states, _, err := ReadLaneStates(root); err != nil || states.Ignored {
		t.Fatalf("outside a repository: %+v %v", states, err)
	}
}
```

  In `internal/gitwork/gitwork_test.go` (`repo`, `commit` und `run` stehen dort):

```go
func TestIgnoredPathAsksGitAboutOnePath(t *testing.T) {
	root := repo(t)
	os.MkdirAll(filepath.Join(root, ".loomux", "state"), 0o755)
	os.WriteFile(filepath.Join(root, ".loomux", "armed.toml"), []byte("armed = []\n"), 0o644)
	if IgnoredPath(root, ".loomux/armed.toml") {
		t.Fatal("nothing is ignored yet")
	}
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/.loomux/state/\n"), 0o644)
	if IgnoredPath(root, ".loomux/armed.toml") || !IgnoredPath(root, ".loomux/state/x") {
		t.Fatal("only the state directory is ignored")
	}
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".loomux/\n"), 0o644)
	if !IgnoredPath(root, ".loomux/armed.toml") {
		t.Fatal("an ignored folder takes the file with it")
	}
	// A file git already holds is not ignored, whatever .gitignore says: it
	// reaches every commit.
	run(t, root, "add", "-f", ".loomux/armed.toml")
	run(t, root, "commit", "-m", "hold the file")
	if IgnoredPath(root, ".loomux/armed.toml") {
		t.Fatal("a tracked file counts as ignored")
	}
	if IgnoredPath(t.TempDir(), ".loomux/armed.toml") {
		t.Fatal("outside a repository a path counts as ignored")
	}
}
```

  Die Signatur von `stopPlan` liest der Implementierer aus `stop.go` (`stopPlan = verify.Plan`) und übernimmt sie für den Ersatz.

- [ ] **Step 2: die Tests des Befehls schreiben** (`internal/cli/gate_test.go`)

```go
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const gateUsageWant = "usage: loomux gate status [--root <dir>]\n" +
	"       loomux gate arm <lane>... [--root <dir>]\n" +
	"       loomux gate disarm <lane>...|--all [--root <dir>]\n"

func TestGateNeedsASubcommandItKnows(t *testing.T) {
	root := armedWorld(t, armedText())
	for _, args := range [][]string{
		{"gate"}, {"gate", "list"}, {"gate", "status", "extra", "--root", root},
		{"gate", "arm", "--root", root}, {"gate", "disarm", "--root", root},
		{"gate", "disarm", "--all", "lint/go@.", "--root", root}, {"gate", "arm", "--all", "--root", root},
	} {
		if code, out, _ := run(args...); code != 2 || out != "" {
			t.Errorf("%v: code %d, out %q", args, code, out)
		}
	}
	if code, _, errOut := run("gate"); code != 2 || errOut != gateUsageWant {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if armedFile(t, root) != armedText() {
		t.Fatal("a refused call wrote")
	}
}

func TestGateStatusNamesEveryLaneAndEveryOrphan(t *testing.T) {
	root := armedWorld(t, "")
	if code, out, errOut := run("gate", "status", "--root", root); code != 0 || out != "no .loomux/armed.toml: every lane is armed\n" || errOut != "" {
		t.Fatalf("no file: code %d, out %q, err %q", code, out, errOut)
	}
	root = armedWorld(t, armedText("lint/go@.", "lint/python@."))
	want := "coverage/go@.: probation\nlint/go@.: armed\nlint/python@.: orphan\ntest/go@.: probation\n"
	if code, out, _ := run("gate", "status", "--root", root); code != 0 || out != want {
		t.Fatalf("code %d, out %q", code, out)
	}
	root = armedWorld(t, "armed = 1\n")
	if code, out, errOut := run("gate", "status", "--root", root); code != 1 || out != "" || !strings.Contains(errOut, "every lane is armed") {
		t.Fatalf("unreadable: code %d, out %q, err %q", code, out, errOut)
	}
}

// The list stands, and beside it the warning: a file git ignores reaches no
// commit, so nobody else and no CI ever reads it.
func TestGateStatusWarnsAboutAnIgnoredFile(t *testing.T) {
	const warning = "loomux gate status: .loomux/armed.toml is ignored by git: it reaches no commit and holds on this machine only\n"
	root := armedWorld(t, armedText("lint/go@."))
	mustRunGit(t, root, "init", "-q")
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".loomux/\n"), 0o644)
	code, out, errOut := run("gate", "status", "--root", root)
	if code != 0 || !strings.Contains(out, "lint/go@.: armed\n") || errOut != warning {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	// Ignoring the state directory alone is how init sets a project up.
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/.loomux/state/\n"), 0o644)
	if code, _, errOut = run("gate", "status", "--root", root); code != 0 || errOut != "" {
		t.Fatalf("only the state ignored: code %d, err %q", code, errOut)
	}
}

func TestGateArmEntersALaneWhateverItsState(t *testing.T) {
	root := armedWorld(t, armedText("lint/python@."))
	code, out, errOut := run("gate", "arm", "test/go@.", "lint/go@.", "--root", root)
	if code != 0 || out != "armed: lint/go@., test/go@.\n" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if got := armedFile(t, root); got != armedText("lint/go@.", "lint/python@.", "test/go@.") {
		t.Fatalf("%q", got)
	}
	// A key no lane answers to is a mistake, and nothing of the call is written.
	before := armedFile(t, root)
	code, out, errOut = run("gate", "arm", "coverage/go@.", "lint/go", "--root", root)
	if code != 1 || out != "" || errOut != "loomux gate arm: no lane \"lint/go\"; lanes: coverage/go@., lint/go@., test/go@.\n" || armedFile(t, root) != before {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestGateArmWithoutTheFileWritesNone(t *testing.T) {
	root := armedWorld(t, "")
	code, out, _ := run("gate", "arm", "lint/go@.", "--root", root)
	if _, err := os.Stat(filepath.Join(root, ".loomux", "armed.toml")); code != 0 || out != "no .loomux/armed.toml: every lane is armed\n" || err == nil {
		t.Fatalf("code %d, out %q, stat %v", code, out, err)
	}
	// A key no lane answers to is an error all the same, before the file is
	// even asked about.
	code, out, errOut := run("gate", "arm", "lint/go", "--root", root)
	if _, err := os.Stat(filepath.Join(root, ".loomux", "armed.toml")); code != 1 || out != "" || !strings.HasPrefix(errOut, "loomux gate arm: no lane \"lint/go\"; lanes: ") || err == nil {
		t.Fatalf("an unknown key without the file: code %d, out %q, err %q, stat %v", code, out, errOut, err)
	}
}

func TestGateDisarmTakesAnEntryOut(t *testing.T) {
	root := armedWorld(t, armedText("lint/go@.", "lint/python@.", "test/go@."))
	code, out, _ := run("gate", "disarm", "lint/python@.", "lint/go@.", "types/go@.", "--root", root)
	if code != 0 || out != "probation: lint/go@., lint/python@.\ntypes/go@.: was not armed\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
	if got := armedFile(t, root); got != armedText("test/go@.") {
		t.Fatalf("%q", got)
	}
}

// Without the file every lane is armed; taking one out of that leaves every
// other armed, which is a file naming them.
func TestGateDisarmWithoutTheFileArmsEveryOtherLane(t *testing.T) {
	root := armedWorld(t, "")
	code, out, _ := run("gate", "disarm", "lint/go@.", "--root", root)
	if code != 0 || out != "probation: lint/go@.\n" || armedFile(t, root) != armedText("coverage/go@.", "test/go@.") {
		t.Fatalf("code %d, out %q, file %q", code, out, armedFile(t, root))
	}
	root = armedWorld(t, "")
	code, _, errOut := run("gate", "disarm", "lint/python@.", "--root", root)
	if _, err := os.Stat(filepath.Join(root, ".loomux", "armed.toml")); code != 1 || !strings.HasPrefix(errOut, "loomux gate disarm: no lane \"lint/python@.\"") || err == nil {
		t.Fatalf("code %d, err %q, stat %v", code, errOut, err)
	}
}

func TestGateDisarmAllStartsTheProbation(t *testing.T) {
	for name, text := range map[string]string{"no file": "", "a full file": armedText("lint/go@."), "a broken file": "<<<<<<<\n"} {
		root := armedWorld(t, text)
		code, out, errOut := run("gate", "disarm", "--all", "--root", root)
		if code != 0 || out != "probation: every lane\n" || errOut != "" || armedFile(t, root) != armedText() {
			t.Errorf("%s: code %d, out %q, err %q, file %q", name, code, out, errOut, armedFile(t, root))
		}
	}
}

func TestGateReportsWhatItCannotReadOrWrite(t *testing.T) {
	root := armedWorld(t, "armed = 1\n")
	for _, args := range [][]string{{"gate", "arm", "lint/go@.", "--root", root}, {"gate", "disarm", "lint/go@.", "--root", root}} {
		if code, _, errOut := run(args...); code != 1 || !strings.Contains(errOut, "every lane is armed") || armedFile(t, root) != "armed = 1\n" {
			t.Errorf("%v: code %d, err %q", args, code, errOut)
		}
	}
	// Lanes that cannot be planned: a config that does not parse. status and
	// arm need the lanes beside a file; disarm needs them only without one.
	root = armedWorld(t, armedText())
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify\n"), 0o644)
	for _, args := range [][]string{{"gate", "status", "--root", root}, {"gate", "arm", "lint/go@.", "--root", root}} {
		if code, _, errOut := run(args...); code != 1 || !strings.HasPrefix(errOut, "loomux gate") || armedFile(t, root) != armedText() {
			t.Errorf("%v: code %d, err %q", args, code, errOut)
		}
	}
	root = armedWorld(t, "")
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify\n"), 0o644)
	if code, _, errOut := run("gate", "disarm", "lint/go@.", "--root", root); code != 1 || !strings.HasPrefix(errOut, "loomux gate disarm: ") {
		t.Errorf("disarm without a file: code %d, err %q", code, errOut)
	}
	// A file that cannot be written: a directory stands in its place.
	root = armedWorld(t, "")
	os.MkdirAll(filepath.Join(root, ".loomux", "armed.toml", "x"), 0o755)
	if code, _, errOut := run("gate", "disarm", "--all", "--root", root); code != 1 || !strings.HasPrefix(errOut, "loomux gate disarm: ") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	// A write that fails after the file was read: the seam stands in for a
	// disk that is full.
	root = armedWorld(t, armedText("lint/go@."))
	old := gateWrite
	gateWrite = func(string, verify.ArmedSet) error { return errors.New("disk full") }
	t.Cleanup(func() { gateWrite = old })
	for sub, args := range map[string][]string{"arm": {"gate", "arm", "test/go@.", "--root", root}, "disarm": {"gate", "disarm", "lint/go@.", "--root", root}} {
		if code, out, errOut := run(args...); code != 1 || out != "" || errOut != "loomux gate "+sub+": disk full\n" {
			t.Errorf("%s: code %d, out %q, err %q", sub, code, out, errOut)
		}
	}
}

// Without --root the project is found upwards, as check finds it.
func TestGateFindsTheRootUpwards(t *testing.T) {
	root := armedWorld(t, armedText())
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644)
	sub := filepath.Join(root, "deep", "er")
	os.MkdirAll(sub, 0o755)
	t.Chdir(sub)
	if code, out, _ := run("gate", "status"); code != 0 || !strings.Contains(out, "lint/go@.: probation") {
		t.Fatalf("code %d, out %q", code, out)
	}
}
```

  Die Testdatei importiert dazu `errors` und `github.com/xidus90/loomux/internal/verify`. `armedWorld`, `armedText` und `armedFile` stehen in `check_arming_test.go` (Task 4), `run` in `cli_test.go`.

- [ ] **Step 3: rot laufen lassen.** Stubs: `gitwork.IgnoredPath` gibt `false`, `GateLanes` gibt `nil, nil`, `ReadLaneStates` gibt `LaneStates{}, false, nil`, `gateCommand` druckt die Usage und gibt 2 zurück; die Zeile in `commands.go` steht schon. Run: `go test ./internal/gitwork -run TestIgnoredPath -count=1`, `go test ./internal/hooks -run "TestGateLanes|TestLaneStates" -count=1` und `go test ./internal/cli -run "TestGate" -count=1`. Expected: FAIL mit Assertions, darunter `[] <nil>` in `TestGateLanesAreTheLanesWithSomethingToRun` und `code 2` in `TestGateStatusNamesEveryLaneAndEveryOrphan`.

- [ ] **Step 4: implementieren.** `internal/gitwork/gitwork.go`, neben dem vorhandenen `ignored(root)`, das dieselbe Frage für die Wurzel stellt:

```go
// IgnoredPath says whether git ignores rel in root. Only exit 0 of
// check-ignore means ignored, as for ignored above: exit 1 is a path git does
// not ignore -- a file it already holds among them, whatever .gitignore says
// -- and everything else (no repository, no git) is no answer, read as not
// ignored. The caller warns about a file that reaches no commit, and a wrong
// warning is worse than none.
func IgnoredPath(root, rel string) bool {
	command := exec.Command("git", "check-ignore", "-q", "--", rel)
	command.Dir = root
	command.Env = gitenv.Environ()
	return command.Run() == nil
}
```

  `internal/hooks/lanestates.go`:

```go
package hooks

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/verify"
)

// GateLanes are the keys of every lane root's gate can run, sorted: the lanes
// of every kind as a check plans them, and the wiki's. A lane with nothing to
// run here -- no command, a graph lane another stack carries -- is none, and
// so is one with nothing to check: a stack without tests has no test and no
// coverage lane. Neither can ever turn green, so neither is armed by a
// commit, and listing them would list them for good. The graph lane is one
// only where a graph was built. A tool that is missing and an import that
// was not made do not take a lane away: it is there, and red.
func GateLanes(root string) ([]string, error) {
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	facts := detect.Detect(os.DirFS(root))
	eff, err := editLoad(root, facts)
	if err != nil {
		return nil, err
	}
	kinds := verify.Kinds()
	jobs, err := stopPlan(eff, verify.Request{Kinds: kinds, Scope: verify.ScopeCheck}, verify.PlanEnv{
		Root: root, Loomux: "loomux", RunID: "gate",
		HasTests: verify.HasTests, ImportReady: verify.ImportReady,
		GraphReady: func(root string) (bool, string) {
			_, err := os.Stat(store.WiringPath(root))
			return err == nil, "no graph"
		},
	})
	if err != nil {
		return nil, err
	}
	jobs = append(jobs, WikiGateJobs(eff, facts, root, kinds)...)
	var keys []string
	for _, j := range jobs {
		if j.Pre != verify.StateNotApplicable && j.Pre != verify.StateUnavailable {
			keys = append(keys, verify.LaneKey(j))
		}
	}
	slices.Sort(keys)
	return slices.Compact(keys), nil
}

// LaneStates sorts the lanes of a gate by what the file says about them, and
// names the entries no lane answers to: a stack that left, an area renamed.
type LaneStates struct {
	Armed, Probation, Orphans []string
	// Ignored says git ignores the file: it reaches no commit then, and
	// what it says holds on this machine only.
	Ignored bool
}

// ReadLaneStates reads root's file and lays it over the lanes. Without the
// file every lane is armed, exists is false, and the lanes are not even
// planned: a project without probation pays nothing for the question.
func ReadLaneStates(root string) (states LaneStates, exists bool, err error) {
	armed, err := verify.ReadArmed(root)
	if err != nil || !armed.Exists {
		return LaneStates{}, false, err
	}
	lanes, err := GateLanes(root)
	if err != nil {
		return LaneStates{}, true, err
	}
	states.Ignored = gitwork.IgnoredPath(root, verify.ArmedFile)
	for _, key := range lanes {
		if slices.Contains(armed.Keys, key) {
			states.Armed = append(states.Armed, key)
		} else {
			states.Probation = append(states.Probation, key)
		}
	}
	for _, key := range armed.Keys {
		if !slices.Contains(lanes, key) {
			states.Orphans = append(states.Orphans, key)
		}
	}
	return states, true, nil
}
```

  `internal/cli/gate.go`:

```go
package cli

import (
	"cmp"
	"flag"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/hooks"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/verify"
)

const gateUsage = "usage: loomux gate status [--root <dir>]\n" +
	"       loomux gate arm <lane>... [--root <dir>]\n" +
	"       loomux gate disarm <lane>...|--all [--root <dir>]"

// gateWrite is the seam for a file that cannot be written after it was read,
// which no world provokes: what stands in the file's way stops the read first.
var gateWrite = verify.WriteArmed

// gateCommand is `loomux gate`: what a human says about which lanes fail the
// gate. A group of its own and not under check, where every first word is a
// profile or a kind. status reads; arm and disarm write .loomux/armed.toml,
// which the guard keeps from agents.
func gateCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	usage := func() int {
		fmt.Fprintln(stderr, gateUsage)
		return 2
	}
	if len(args) == 0 || !slices.Contains([]string{"status", "arm", "disarm"}, args[0]) {
		return usage()
	}
	sub := args[0]
	flags := flag.NewFlagSet("loomux gate "+sub, flag.ContinueOnError)
	flags.SetOutput(stderr)
	rootFlag := flags.String("root", "", "path to the project root; found upwards when empty")
	all := new(bool)
	if sub == "disarm" {
		all = flags.Bool("all", false, "put every lane into probation: write the file without an entry")
	}
	keys, err := parseInterspersed(flags, args[1:])
	if err != nil {
		return 2
	}
	root := *rootFlag
	if root == "" {
		root, _ = hosts.FindRoot(".")
	}
	if abs, err := filepath.Abs(cmp.Or(root, ".")); err == nil {
		root = abs
	}
	fail := func(err error) int {
		fmt.Fprintf(stderr, "loomux gate %s: %v\n", sub, err)
		return 1
	}
	switch {
	case sub == "status" && len(keys) == 0:
		return gateStatus(root, stdout, stderr, fail)
	case sub == "arm" && len(keys) > 0:
		return gateArm(root, keys, stdout, fail)
	case sub == "disarm" && *all && len(keys) == 0:
		if err := gateWrite(root, verify.ArmedSet{Exists: true}); err != nil {
			return fail(err)
		}
		fmt.Fprintln(stdout, "probation: every lane")
		return 0
	case sub == "disarm" && !*all && len(keys) > 0:
		return gateDisarm(root, keys, stdout, fail)
	}
	return usage()
}

const noArmedFile = "no " + verify.ArmedFile + ": every lane is armed"

// gateStatus prints every lane and every entry without a lane, by key.
func gateStatus(root string, stdout, stderr io.Writer, fail func(error) int) int {
	states, exists, err := hooks.ReadLaneStates(root)
	if err != nil {
		return fail(err)
	}
	if !exists {
		fmt.Fprintln(stdout, noArmedFile)
		return 0
	}
	said := map[string]string{}
	for state, keys := range map[string][]string{"armed": states.Armed, "probation": states.Probation, "orphan": states.Orphans} {
		for _, key := range keys {
			said[key] = state
		}
	}
	for _, key := range slices.Sorted(maps.Keys(said)) {
		fmt.Fprintf(stdout, "%s: %s\n", key, said[key])
	}
	if states.Ignored {
		// A warning, not a refusal: the list above holds, on this machine.
		fmt.Fprintf(stderr, "loomux gate status: %s is ignored by git: it reaches no commit and holds on this machine only\n", verify.ArmedFile)
	}
	return 0
}

// unknownLane is the refusal for the first key no lane answers to.
func unknownLane(keys, lanes []string) error {
	for _, key := range keys {
		if !slices.Contains(lanes, key) {
			return fmt.Errorf("no lane %q; lanes: %s", key, strings.Join(lanes, ", "))
		}
	}
	return nil
}

// gateArm enters lanes whatever their state: from now on they count. A key
// no lane answers to refuses the whole call.
func gateArm(root string, keys []string, stdout io.Writer, fail func(error) int) int {
	armed, err := verify.ReadArmed(root)
	var lanes []string
	if err == nil {
		lanes, err = hooks.GateLanes(root)
	}
	// A key no lane answers to is a mistake before anything else, with the
	// file and without it: a typo must not read as "every lane is armed".
	if err == nil {
		err = unknownLane(keys, lanes)
	}
	if err != nil {
		return fail(err)
	}
	if !armed.Exists {
		// Every lane is armed already; a file naming only these would put
		// every other lane into probation.
		fmt.Fprintln(stdout, noArmedFile)
		return 0
	}
	if err := gateWrite(root, armed.With(keys...)); err != nil {
		return fail(err)
	}
	fmt.Fprintf(stdout, "armed: %s\n", strings.Join(verify.ArmedSet{}.With(keys...).Keys, ", "))
	return 0
}

// gateDisarm takes entries out. Without the file every lane is armed, so
// taking one out leaves a file that names every other lane.
func gateDisarm(root string, keys []string, stdout io.Writer, fail func(error) int) int {
	armed, err := verify.ReadArmed(root)
	if err != nil {
		return fail(err)
	}
	if !armed.Exists {
		lanes, err := hooks.GateLanes(root)
		if err == nil {
			err = unknownLane(keys, lanes)
		}
		if err != nil {
			return fail(err)
		}
		armed = verify.ArmedSet{}.With(lanes...)
	}
	var gone, absent []string
	for _, key := range (verify.ArmedSet{}).With(keys...).Keys {
		if slices.Contains(armed.Keys, key) {
			gone = append(gone, key)
		} else {
			absent = append(absent, key)
		}
	}
	if err := gateWrite(root, armed.Without(keys...)); err != nil {
		return fail(err)
	}
	if len(gone) > 0 {
		fmt.Fprintf(stdout, "probation: %s\n", strings.Join(gone, ", "))
	}
	for _, key := range absent {
		fmt.Fprintf(stdout, "%s: was not armed\n", key)
	}
	return 0
}
```

- [ ] **Step 5: grün.** Run: `go test ./internal/gitwork -count=1`, `go test ./internal/hooks -run "TestGateLanes|TestLaneStates" -count=1` und `go test ./internal/cli -run "TestGate|TestCases|TestUsage|TestRun" -count=1`. Expected: PASS. Ein Test, der die Befehlsliste der Usage wörtlich festhält (`internal/cli/cli_test.go`), bekommt die Zeile `  gate` an ihrer alphabetischen Stelle. `go test ./internal/cli -run "TestHooksNever" -count=1` (die Importgrenzen) bleibt grün.
- [ ] **Step 6: Mutation per `go test -overlay`:**
  - `GateLanes`: `j.Pre != verify.StateNotApplicable &&` entfernen: `TestGateLanesAreTheLanesWithSomethingToRun` wird rot (`types/go@.`).
  - `GateLanes`: `&& j.Pre != verify.StateUnavailable` entfernen: derselbe Test („without tests“) wird rot.
  - `GateLanes`: `HasTests: verify.HasTests` durch eine Funktion ersetzen, die immer `true` antwortet: derselbe Test („without tests“) wird rot.
  - `GateLanes`: `GraphReady` immer `true`: derselbe Test, erste Hälfte, wird rot.
  - `ReadLaneStates`: `|| !armed.Exists` entfernen: `TestLaneStatesSortEveryLaneAndEveryEntry` (erste Assertion) wird rot.
  - `ReadLaneStates`: die Zeile `states.Ignored = …` entfernen: `TestLaneStatesSayWhenGitIgnoresTheFile` und `TestGateStatusWarnsAboutAnIgnoredFile` werden rot.
  - `IgnoredPath`: `command.Run() == nil` durch `command.Run() != nil` ersetzen: `TestIgnoredPathAsksGitAboutOnePath` wird rot. `rel` durch `"."` ersetzen: derselbe Test („only the state directory is ignored“) wird rot.
  - `gateStatus`: `if states.Ignored` durch `if !states.Ignored` ersetzen: `TestGateStatusWarnsAboutAnIgnoredFile` wird rot, und jeder andere Statustest, der ein leeres stderr verlangt.
  - `gateArm`: den Zweig `if !armed.Exists` entfernen: `TestGateArmWithoutTheFileWritesNone` wird rot.
  - `gateArm`: `err = unknownLane(keys, lanes)` entfernen: `TestGateArmEntersALaneWhateverItsState` (zweite Hälfte) wird rot.
  - `gateArm`: den Zweig `if !armed.Exists` vor die Prüfung des Schlüssels ziehen: `TestGateArmWithoutTheFileWritesNone` (zweite Hälfte) wird rot.
  - `gateDisarm`: `armed = verify.ArmedSet{}.With(lanes...)` durch `armed = verify.ArmedSet{Exists: true}` ersetzen: `TestGateDisarmWithoutTheFileArmsEveryOtherLane` wird rot.
  - Im Schalter `&& len(keys) == 0` bei `status` entfernen: `TestGateNeedsASubcommandItKnows` wird rot.
- [ ] **Step 7: Commit** `feat(cli): add gate status, arm and disarm`.

---

### Task 8: Der Wächter — die Datei und `gate arm|disarm` gehören dem Menschen

**Files:**
- Modify: `internal/hooks/guard.go` (`builtinPathRules`, `checkTool`; neu `armedReason`)
- Create: `internal/hooks/guardgate.go`
- Create: `internal/hooks/testdata/guard-battery-before.tsv`
- Test: `internal/hooks/guardgate_test.go` (neu), `internal/hooks/guardflow_test.go` (`asGateAnswer` wird auf einen gemeinsamen Helfer umgestellt), `internal/hooks/guardshell_test.go` (`runFileShellWrites`: die Ablehnung von `rm -rf .loomux` nennt einen Grund mehr)

**Interfaces:**
- Consumes: `lineVariants`, `segments`, `readings`, `dropPrefixes`, `programArgs`, `checkTool`, `manifestShellWrites`, `manifestShellReads`, `loomuxSpellings`, `shellReasons` (alle vorhanden).
- Produces:
  - `const armedReason = ".loomux/armed.toml: which lanes fail the gate is a human's decision; an agent arms a lane only by a green commit"`
  - `const gateReason = "loomux gate arm and disarm decide which lanes fail the gate; a human runs them. `loomux gate status` shows the lanes"` (der Backtick-Teil im Go-Quelltext als verketteter String, siehe unten)
  - `func armsOrDisarms(line string, anyProgram bool) bool`

**Warum die Pfadregel reicht:** Jede Shell-Form, die `.loomux/config.toml` schützt, läuft über `shellWrites` und dieselben Pfadregeln (`judge.judgePath`, `internal/hooks/guardjudge.go:146`); es gibt keinen auf `config.toml` verdrahteten Sonderweg außer der Regel selbst. Eine zweite Zeile in `builtinPathRules` deckt also Write/Edit, Umleitung, `cp`, `mv`, `sed -i`, `tee`, `git checkout --`, `git restore`, die PowerShell-Schreiber und das Löschen über den Eltern-Ordner (`ancestorReasons`). Die Differenzprobe beweist es, statt es zu behaupten.

- [ ] **Step 1: die Batterie und ihren Stand vor der Änderung festhalten — eigener Schritt, vor jeder Änderung am Wächter.** In `internal/hooks/guardgate_test.go` zuerst nur die Batterie und den Aufzeichner anlegen:

```go
package hooks

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

const batteryFile = "testdata/guard-battery-before.tsv"

// inPlaceOfConfig is a line about the manifest, turned into the same line
// about the armed lanes.
func inPlaceOfConfig(line string) string {
	return strings.NewReplacer("config.toml", "armed.toml", "Config.toml", "Armed.toml").Replace(line)
}

// armedShellWrites are the manifest's write spellings aimed at the armed
// lanes, and the ways that reach the file through its folder.
func armedShellWrites() []string {
	var lines []string
	for _, line := range manifestShellWrites() {
		lines = append(lines, inPlaceOfConfig(line))
	}
	return append(lines,
		"git restore -- .loomux/armed.toml",
		"git restore --source=HEAD~1 -- .loomux/armed.toml",
		"git checkout HEAD~1 -- .loomux/armed.toml",
		"sed -i '/lint/d' .loomux/armed.toml",
		"rm -f ./.loomux/armed.toml",
		"Remove-Item -Force .loomux/armed.toml",
		"echo 'armed = []' > .loomux/armed.toml",
	)
}

// folderRemovals take the file with its folder; they were refused for the
// manifest's sake before the armed lanes existed.
func folderRemovals() []string {
	return []string{
		"rm -rf .loomux",
		"rm -r .loomux/",
		"Remove-Item -Recurse -Force .loomux",
		"git clean -fdx",
		"mv .loomux elsewhere",
	}
}

// knownGaps are spellings that can disarm a lane and pass, before this rule
// and after it: the armed lanes share them with the manifest, and this
// change does not close them. They stand in the battery as recorded passes,
// so that nobody takes them for closed and a later fix flips them on purpose.
//   - git checkout <rev> -- <folder> and git restore -s <rev> <folder> bring
//     back an older file through the folder above it; the guard judges a
//     folder as a write target only for a removal.
//   - git stash, git reset --hard and git switch rewrite the working tree
//     without naming a path.
//   - sh -c "…" hands loomux to a second shell inside a string the words of
//     the line do not open.
func knownGaps() []string {
	return []string{
		"git checkout HEAD~1 -- .loomux",
		"git checkout HEAD~1 -- .",
		"git restore -s HEAD~1 .",
		"git stash",
		"git reset --hard",
		"git switch other-branch",
		`sh -c "loomux gate disarm --all"`,
	}
}

// gateLines are calls of the gate group, with what the guard says after the
// change: true for a refusal.
func gateLines() map[string]bool {
	lines := map[string]bool{
		"loomux gate status":                        false,
		"loomux gate status --root .":               false,
		"bin/loomux.exe gate status":                false,
		"loomux gate":                               false,
		"echo \"loomux gate arm lint/go@.\" > notes": false,
		"loomux gate arm lint/go@.":                 true,
		"loomux gate disarm lint/go@.":              true,
		"loomux gate disarm --all":                  true,
		"loomux gate --root . arm lint/go@.":        true,
		"loomux gate status; loomux gate arm x":     true,
		"loomux gate status && loomux gate disarm --all": true,
		"cd x && loomux gate arm lint/go@.":         true,
	}
	for _, row := range loomuxSpellings() {
		lines[inPlaceOfCommand(nil, row, "gate arm lint/go@.", `ga\te arm lint/go@.`, "ga`te arm lint/go@.")] = true
	}
	return lines
}

// battery is every line the guard is asked about, each once, in a fixed order.
func battery() []string {
	lines := slices.Concat(manifestShellWrites(), manifestShellReads(), armedShellWrites(), folderRemovals(), knownGaps())
	for _, line := range manifestShellReads() {
		lines = append(lines, inPlaceOfConfig(line))
	}
	for line := range gateLines() {
		lines = append(lines, line)
	}
	slices.Sort(lines)
	return slices.Compact(lines)
}

// refusedBy says whether the guard refuses line from Bash.
func refusedBy(root, line string) bool {
	return len(checkTool(root, "Bash", map[string]any{"command": line}, config.Policy{})) > 0
}

// TestRecordTheGuardBattery writes what the guard says about every line of
// the battery. It runs only when asked, once, at the state before a change to
// the guard; the file it writes is the "before" the differential test reads.
func TestRecordTheGuardBattery(t *testing.T) {
	if os.Getenv("LOOMUX_RECORD_GUARD_BATTERY") != "1" {
		t.Skip("set LOOMUX_RECORD_GUARD_BATTERY=1 to record the guard's verdicts")
	}
	root := t.TempDir()
	var b strings.Builder
	for _, line := range battery() {
		verdict := "pass"
		if refusedBy(root, line) {
			verdict = "refused"
		}
		b.WriteString(verdict + "\t" + strconv.Quote(line) + "\n")
	}
	if err := os.MkdirAll(filepath.Dir(batteryFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(batteryFile, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}
```

  Dazu in `internal/hooks/guardflow_test.go` den Helfer verallgemeinern, ohne ein Verhalten zu ändern: `asGateAnswer` wird

```go
func asGateAnswer(t *testing.T, row string) string {
	t.Helper()
	tick := "`"
	return inPlaceOfCommand(t, row, "flow resume 0001 --answer yes", `flow res\ume 0001 --answer yes`, "flow res"+tick+"ume 0001 --answer yes")
}

// inPlaceOfCommand puts a command where a spelling row has its own, the first
// init, config, area add or merge-hook install or remove that stands as a
// word: plain, or slashed and ticked where the row spells its command with an
// escape, since the escape is what such a row spells. A row that names no
// command must be a Start-Process spelling, refused whatever it runs; the
// command goes after it. Any other such row is a new spelling this helper
// cannot place a command in, and fails the test rather than pass untested;
// with a nil t it panics, for a caller that builds a list outside a test.
func inPlaceOfCommand(t *testing.T, row, plain, slashed, ticked string) string {
	if t != nil {
		t.Helper()
	}
	tick := "`"
	command := regexp.MustCompile(`(^|[\s"'{(])(con\\fig|con` + tick + `fig|merge-hook (?:install|remove)|area add|config|init)($|[\s"'` + tick + `;})])`)
	at := command.FindStringSubmatchIndex(row)
	if at == nil {
		if head := strings.ToLower(strings.Fields(row)[0]); head != "start-process" && head != "start" && head != "saps" {
			if t == nil {
				panic(row + " names no command and is no Start-Process spelling")
			}
			t.Fatalf("%q names no command to put another in place of, and is no Start-Process spelling", row)
		}
		return row + " " + plain
	}
	switch row[at[4]:at[5]] {
	case `con\fig`:
		plain = slashed
	case "con" + tick + "fig":
		plain = ticked
	}
	return row[:at[4]] + plain + row[at[5]:]
}
```

  Run (am Stand **vor** jeder Änderung an `guard.go`): `go test ./internal/hooks -run "TestAnAgentMayNotAnswerAGate" -count=1` (der umgestellte Helfer, Expected: PASS), dann mit gesetzter Umgebungsvariable `LOOMUX_RECORD_GUARD_BATTERY=1` den Lauf `go test ./internal/hooks -run TestRecordTheGuardBattery -count=1`. Expected: PASS, und `internal/hooks/testdata/guard-battery-before.tsv` existiert. Die Datei lesen und gegenprüfen: jede Zeile aus `manifestShellWrites()` und `folderRemovals()` trägt `refused`; jede Zeile mit `armed.toml` als Schreibziel, jede `gate`-Zeile und jede Zeile aus `knownGaps()` (die benannten Lücken der Spec) trägt `pass` (das ist der Stand vor der Änderung, den die Probe als Ausgangspunkt braucht). Zeilen zählen: Expected: so viele wie `len(battery())`. Task 0 hat die Aufzeichnung am 2026-10-01 an `e6c21eed` in einer Kopie des Baums gefahren: 278 Zeilen, 55 `refused`, 223 `pass`, Byte für Byte gleich der Aufzeichnung der Nachrechnung vom 2026-09-30 an `47361ab3`. Die Regel der Fix-Welle für `dev switchover prune-hooks` (neuer Fall `dev` in `wordsWriteConfiguration`) trifft keine Zeile der Batterie; der Implementierer zeichnet trotzdem selbst auf, an seinem Stand vor der Änderung, und übernimmt keine Datei aus dem Scratchpad.

- [ ] **Step 2: die Tests der neuen Regeln schreiben** (weiter in `guardgate_test.go`)

```go
// readBattery is the recorded verdict of every line, by line.
func readBattery(t *testing.T) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(batteryFile)
	if err != nil {
		t.Fatal(err)
	}
	before := map[string]bool{}
	for _, row := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		verdict, quoted, _ := strings.Cut(row, "\t")
		line, err := strconv.Unquote(quoted)
		if err != nil {
			t.Fatalf("%q: %v", row, err)
		}
		before[line] = verdict == "refused"
	}
	return before
}

// The differential probe: no line the guard refused before passes now, and
// the lines that flipped to a refusal are exactly the writes of the armed
// lanes and the calls that arm or disarm.
func TestTheGuardOpensNothingItRefusedBefore(t *testing.T) {
	before := readBattery(t)
	if lines := battery(); len(before) != len(lines) {
		t.Fatalf("the battery has %d lines, the record %d: record it again at the state before the change", len(lines), len(before))
	}
	root := t.TempDir()
	flips := map[string]bool{}
	for _, line := range armedShellWrites() {
		flips[line] = true
	}
	for line, refused := range gateLines() {
		if refused {
			flips[line] = true
		}
	}
	for line, was := range before {
		now := refusedBy(root, line)
		switch {
		case was && !now:
			t.Errorf("%q was refused and passes now", line)
		case !was && now && !flips[line]:
			t.Errorf("%q passed and is refused now, and is no write of the armed lanes", line)
		case !was && !now && flips[line]:
			t.Errorf("%q still passes", line)
		}
	}
}

// The gaps the rule names are gaps: each passed before and passes now. One
// that is refused here was closed, and then it leaves knownGaps, the comment
// of armedReason and the docs together.
func TestTheNamedGapsAreStillOpen(t *testing.T) {
	before := readBattery(t)
	root := t.TempDir()
	for _, line := range knownGaps() {
		was, recorded := before[line]
		if !recorded || was || refusedBy(root, line) {
			t.Errorf("%q: recorded %v, refused before %v, refused now %v", line, recorded, was, refusedBy(root, line))
		}
	}
}

func TestTheArmedLanesAreKeptFromEveryWritingTool(t *testing.T) {
	root := t.TempDir()
	for _, tool := range []string{"Write", "Edit"} {
		for _, path := range []string{".loomux/armed.toml", ".LOOMUX/Armed.toml", filepath.Join(root, ".loomux", "armed.toml"), "../other/.loomux/armed.toml"} {
			if got := checkTool(root, tool, map[string]any{"file_path": path}, config.Policy{}); !slices.Equal(got, []string{armedReason}) {
				t.Errorf("%s %s: reasons %q", tool, path, got)
			}
		}
	}
	for _, path := range []string{".loomux/armed.toml.bak", "docs/armed.toml", ".loomux/armedx.toml"} {
		if got := checkTool(root, "Write", map[string]any{"file_path": path}, config.Policy{}); len(got) != 0 {
			t.Errorf("%s: reasons %q, want none", path, got)
		}
	}
}

func TestTheShellPathCheckKeepsTheArmedLanes(t *testing.T) {
	root := t.TempDir()
	for _, line := range armedShellWrites() {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Equal(got, []string{armedReason}) {
			t.Errorf("%q: reasons %q, want the armed lanes'", line, got)
		}
	}
	for _, line := range manifestShellReads() {
		if got := shellReasons(t, root, inPlaceOfConfig(line), config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", inPlaceOfConfig(line), got)
		}
	}
	// A removal of the folder takes both files with it and names both.
	if got := shellReasons(t, root, "rm -rf .loomux", config.Policy{}); !slices.Contains(got, armedReason) || !slices.Contains(got, manifestReason) {
		t.Errorf("rm -rf .loomux: reasons %q", got)
	}
	// Without its --, git restore and git checkout take a word for a path only
	// where one stands (checkedOut): the file has to be there to be named.
	for _, line := range []string{"git restore .loomux/armed.toml", "git checkout .loomux/armed.toml"} {
		if got := shellReasons(t, root, line, config.Policy{}); len(got) != 0 {
			t.Errorf("%q without the file: reasons %q", line, got)
		}
	}
	mkfile(t, root, ".loomux/armed.toml")
	for _, line := range []string{"git restore .loomux/armed.toml", "git checkout .loomux/armed.toml", "git restore --worktree --staged .loomux/armed.toml"} {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Equal(got, []string{armedReason}) {
			t.Errorf("%q: reasons %q", line, got)
		}
	}
	// The index alone is not the file.
	if got := shellReasons(t, root, "git restore --staged .loomux/armed.toml", config.Policy{}); len(got) != 0 {
		t.Errorf("restore --staged: reasons %q", got)
	}
}

func TestAnAgentMayNotArmOrDisarm(t *testing.T) {
	for line, refused := range gateLines() {
		if got := armsOrDisarms(line, false); got != refused && !strings.HasPrefix(strings.ToLower(line), "start") && !strings.HasPrefix(strings.ToLower(line), "saps") {
			t.Errorf("%q: armsOrDisarms = %v, want %v", line, got, refused)
		}
	}
	// In strict mode a renamed binary is loomux by what it is told.
	if !armsOrDisarms("doc.exe gate disarm --all", true) || armsOrDisarms("doc.exe gate disarm --all", false) {
		t.Error("a renamed binary: strict must refuse, the default mode must not")
	}
}

// The refusal names the way a human does it, and status stays open.
func TestCheckToolNamesTheGateReason(t *testing.T) {
	for _, tool := range []string{"Bash", "PowerShell"} {
		got := checkTool(t.TempDir(), tool, map[string]any{"command": "loomux gate arm lint/go@."}, config.Policy{})
		if !slices.Equal(got, []string{gateReason}) {
			t.Errorf("[%s] reasons %q", tool, got)
		}
		if got := checkTool(t.TempDir(), tool, map[string]any{"command": "loomux gate status"}, config.Policy{}); len(got) != 0 {
			t.Errorf("[%s] gate status: reasons %q", tool, got)
		}
	}
	// A Start-Process of loomux keeps the reasons it had: the words cannot see
	// its arguments, and the call is refused without this rule.
	line := "Start-Process loomux -ArgumentList 'gate','arm','lint/go@.'"
	got := checkTool(t.TempDir(), "PowerShell", map[string]any{"command": line}, config.Policy{})
	if len(got) == 0 || slices.Contains(got, gateReason) {
		t.Errorf("Start-Process: reasons %q", got)
	}
}
```

- [ ] **Step 3: rot laufen lassen.** Stubs: die Konstanten `armedReason` und `gateReason` anlegen, `armsOrDisarms` gibt `false`. Run: `go test ./internal/hooks -run "TestTheGuardOpensNothing|TestTheArmedLanesAreKept|TestTheShellPathCheckKeepsTheArmedLanes|TestAnAgentMayNotArmOrDisarm|TestCheckToolNamesTheGateReason|TestTheNamedGaps" -count=1`. Expected: FAIL mit Assertions, darunter `"echo x >.loomux/armed.toml" still passes` und `"loomux gate arm lint/go@.": armsOrDisarms = false, want true`.

- [ ] **Step 4: implementieren.** `internal/hooks/guard.go`: hinter `manifestReason`

```go
// armedReason refuses an agent the armed lanes, to a writing tool and to a
// shell line alike. A line taken out disarms, and a file removed arms every
// lane for everybody: both are a human's to decide.
//
// Named gaps, shared with the manifest and not closed here; each can disarm
// a lane and passes: git checkout <rev> -- .loomux, git checkout <rev> -- .,
// git restore -s <rev> ., git stash, git reset --hard, git switch, and
// sh -c "loomux gate disarm --all". The battery of the guard's tests holds
// them as recorded passes (knownGaps).
const armedReason = ".loomux/armed.toml: which lanes fail the gate is a human's decision; an agent arms a lane only by a green commit"
```

  in `builtinPathRules`, direkt hinter der Zeile mit `manifestReason`:

```go
	// The second copy of verify.ArmedFile, kept in step by hand like the
	// state directories below: a rule is a verbatim glob here.
	{Match: []string{"**/.loomux/armed.toml"}, Reason: armedReason},
```

  in `checkTool`, hinter dem Block `if answersAGate(line, policy.Strict) { … }`:

```go
					if armsOrDisarms(line, policy.Strict) {
						reasons = append(reasons, gateReason)
					}
```

  `internal/hooks/guardgate.go`:

```go
package hooks

// gateReason refuses an agent the two commands that write the armed lanes.
const gateReason = "loomux gate arm and disarm decide which lanes fail the gate; a human runs them. " +
	"`loomux gate status` shows the lanes"

// armsOrDisarms says whether a shell line runs `loomux gate` with anything
// but status. An agent arms a lane only by a commit that goes through; the
// command would let it disarm one as well. The line is read the way
// writesConfiguration reads it, and has the same holes; with anyProgram, in
// strict mode, every program knownProgram does not name counts as loomux.
//
// What reads is named and everything else is refused, so a subcommand added
// later is refused until it is listed here. A bare `loomux gate` prints the
// usage and passes. A Start-Process of loomux is none of this rule's: the
// configuration rule refuses it already, whatever it runs, and a second
// reason would change what that refusal says.
func armsOrDisarms(line string, anyProgram bool) bool {
	for _, variant := range lineVariants(line) {
		for _, segment := range segments(variant) {
			for _, words := range readings(segment) {
				if readingArms(words, anyProgram) {
					return true
				}
			}
		}
	}
	return false
}

// readingArms judges one reading from its head and from every word after a
// lone { or }, for readingWrites' reason: a block opens a command.
func readingArms(words []string, anyProgram bool) bool {
	for i, w := range words {
		if (w == "{" || w == "}") && wordsArm(words[i+1:], anyProgram) {
			return true
		}
	}
	return wordsArm(words, anyProgram)
}

// wordsArm says whether one reading, past what runs in front of the program,
// is loomux gate with a subcommand other than status.
func wordsArm(words []string, anyProgram bool) bool {
	words = dropPrefixes(words)
	if len(words) == 0 {
		return false
	}
	args, ok := programArgs(words, anyProgram)
	return ok && len(args) > 1 && args[0] == "gate" && args[1] != "status"
}
```

  Den Kommentar über `builtinPathRules` („protect secrets, the manifest, the stop gate's controls, …“) um „the armed lanes“ ergänzen.

  **Ein vorhandener Test ändert sich, und mit ihm eine Ausgabe ohne Datei.** Wer `.loomux` löscht, nimmt jetzt auch die scharfen Lanes mit; die Ablehnung nennt darum einen Grund mehr, gleich, ob das Projekt eine `armed.toml` hat. Das ist nach der Spec (Abschnitt „Der Wächter“, „Eine Folge ohne Datei“) neben `init` die zweite Ausnahme von der Byte-gleich-Invariante; sie trifft nur den Text einer Ablehnung, nicht, ob abgelehnt wird. In `internal/hooks/guardshell_test.go`, `runFileShellWrites` (Zeile 153), wird die Erwartung für `rm -rf .loomux`:

```go
		// Flip: removing .loomux was a hole. The armed lanes go with the
		// folder, so their reason stands beside the manifest's.
		"rm -rf .loomux": {stopGateWant, manifestReason, armedReason, runFilesWant},
```

  Die Reihenfolge ist die der Regeln in `builtinPathRules`. Die aufgezeichneten Fälle unter `testdata/cases/` enthalten diese Zeile nicht (der Lauf `TestCases` in Step 5 belegt es); stünde sie doch in einem, wäre das ein Befund für den Menschen, keine Aufzeichnung zum Umschreiben.

- [ ] **Step 5: grün.** Run: `go test ./internal/hooks -count=1` und `go test ./internal/cli -run "TestCases" -count=1`. Expected: PASS; die aufgezeichneten `hook-pre-tool-use`-Fälle unverändert. Schlägt `TestTheGuardOpensNothingItRefusedBefore` mit „passed and is refused now, and is no write of the armed lanes“ fehl, ist das ein Befund an der Regel (eine Lesezeile wird jetzt verweigert), nicht an der Erwartung: die Regel enger fassen, die Aufzeichnung **nicht** neu schreiben.
- [ ] **Step 6: Mutation per `go test -overlay`:**
  - Die neue Zeile in `builtinPathRules` entfernen: `TestTheArmedLanesAreKeptFromEveryWritingTool` und `TestTheGuardOpensNothingItRefusedBefore` („still passes“) werden rot.
  - Das Glob auf `.loomux/armed.toml` (ohne `**/`) kürzen: `TestTheArmedLanesAreKept…` (absoluter Pfad, Nachbarprojekt) wird rot.
  - `wordsArm`: `args[1] != "status"` durch `args[1] == "arm"` ersetzen: `TestAnAgentMayNotArmOrDisarm` (`gate disarm --all`, `gate --root . arm`) wird rot.
  - `wordsArm`: `len(args) > 1 &&` durch `len(args) > 0 &&` ersetzen: Nil-/Index-Panik in `TestAnAgentMayNotArmOrDisarm` (`loomux gate`).
  - In `checkTool` den neuen Block entfernen: `TestCheckToolNamesTheGateReason` wird rot.
- [ ] **Step 7: Commit** `feat(hooks): keep the armed lanes from agents`. Ab diesem Commit verweigert der Wächter des Piloten (`bin/loomux.exe`, vom pre-commit-Tor neu gebaut) dem Agenten jedes Schreiben einer `.loomux/armed.toml`, auch in einem anderen Checkout; Testwelten schreiben sie weiter im Testprozess.

---

### Task 9: Sichtbarkeit — Session-Start und Statusbericht

**Files:**
- Modify: `internal/gitwork/gitwork.go` (neu `HooksDir`)
- Create: `internal/hooks/probation.go`
- Modify: `internal/hooks/hook_session_start.go` (`SessionStart`), `internal/hooks/status.go` (`Status`)
- Test: `internal/gitwork/gitwork_test.go`, `internal/hooks/probation_test.go` (neu)

**Interfaces:**
- Consumes: `ReadLaneStates` und `LaneStates.Ignored` (Task 7), `sessions.LastSeen` (Task 6), `verify.HookArms` (Task 2), `gitwork.Head`, `hosts.WriteContext` (vorhanden).
- Produces:
  - `func HooksDir(root string) (string, error)` in `internal/gitwork` — der Ordner, aus dem git die Hooks von `root` ruft, absolut: `core.hooksPath`, wo gesetzt, sonst gits eigener.
  - `func probationLines(root string) []string` — die Kontextzeilen des Session-Starts.
  - `func renderProbation(w io.Writer, root string)` — der Abschnitt des Statusberichts.
  - `const maxSeenLines = 40`

- [ ] **Step 1: die Tests schreiben.** In `internal/gitwork/gitwork_test.go` (`repo` und `run` stehen dort, Zeile 130 und 166):

```go
func TestHooksDirIsWhereGitRunsHooksFrom(t *testing.T) {
	root := repo(t)
	dir, err := HooksDir(root)
	if err != nil || !filepath.IsAbs(dir) || filepath.Base(dir) != "hooks" || filepath.Base(filepath.Dir(dir)) != ".git" {
		t.Fatalf("%q %v", dir, err)
	}
	run(t, root, "config", "core.hooksPath", ".githooks")
	dir, err = HooksDir(root)
	if err != nil || !filepath.IsAbs(dir) || filepath.Base(dir) != ".githooks" {
		t.Fatalf("with core.hooksPath: %q %v", dir, err)
	}
	if _, err := HooksDir(t.TempDir()); err == nil {
		t.Fatal("a directory that is no repository went through")
	}
}
```

  `internal/hooks/probation_test.go`:

```go
package hooks

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/sessions"
)

// probationProject is a Go project loomux recognises as a root, committed,
// with the file holding text.
func probationProject(t *testing.T, armed string) string {
	t.Helper()
	root := project(t)
	writeWorldFile(t, root, "go.mod", "module m\n")
	writeWorldFile(t, root, "a_test.go", "package m\n")
	gitInit(t, root)
	if armed != "" {
		writeWorldFile(t, root, ".loomux/armed.toml", armed)
	}
	return root
}

func TestSessionStartNamesTheLanesInProbation(t *testing.T) {
	root := probationProject(t, "armed = [\"lint/go@.\"]\n")
	lines := probationLines(root)
	want := []string{
		"loomux: lane coverage/go@. is in probation: what it finds warns and fails no gate until a green commit arms it",
		"loomux: lane test/go@. is in probation: what it finds warns and fails no gate until a green commit arms it",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("%q", lines)
	}
	// Through the hook, in the channel session start already answers in.
	var out, errOut bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1","hook_event_name":"SessionStart"}`), &out, &errOut, root, "claude", "1.0.0")
	if code != ExitOK || !strings.Contains(out.String(), "additionalContext") || !strings.Contains(out.String(), "lane test/go@. is in probation") {
		t.Fatalf("%d %q %q", code, out.String(), errOut.String())
	}
}

// Without the file, and with every lane armed, session start says what it
// says today: nothing.
func TestSessionStartSaysNothingWithoutALaneInProbation(t *testing.T) {
	for name, armed := range map[string]string{"no file": "", "every lane armed": "armed = [\"coverage/go@.\", \"lint/go@.\", \"test/go@.\"]\n"} {
		root := probationProject(t, armed)
		// A stand some session left while lanes were in probation says nothing
		// once none is.
		stale := &sessions.Seen{Tree: "t", Head: headOf(t, root), Report: "lint/go: failed (probation)\n", At: time.Now()}
		if err := sessions.WriteState(root, "earlier", sessions.SessionState{Seen: stale}); err != nil {
			t.Fatal(err)
		}
		var out, errOut bytes.Buffer
		code := SessionStart(strings.NewReader(`{"session_id":"s1","hook_event_name":"SessionStart"}`), &out, &errOut, root, "claude", "1.0.0")
		if code != ExitOK || out.Len() != 0 {
			t.Errorf("%s: %d %q %q", name, code, out.String(), errOut.String())
		}
	}
}

func TestSessionStartSaysAnUnreadableArmedFile(t *testing.T) {
	root := probationProject(t, "armed = 1\n")
	lines := probationLines(root)
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "loomux: .loomux/armed.toml ") || !strings.HasSuffix(lines[0], "every lane is armed") {
		t.Fatalf("%q", lines)
	}
}

// What the stop gate last saw is passed on while HEAD is where it saw it,
// from whichever session saw it, and cut where it runs long.
func TestSessionStartPassesOnWhatTheStopGateLastSaw(t *testing.T) {
	root := probationProject(t, "armed = []\n")
	head := headOf(t, root)
	report := "lint/go: failed (probation) [preset] 0.1s\na.go:1: bad\nprobation: lint/go@. (warn only until a green commit arms them)\n"
	if err := sessions.WriteState(root, "earlier", sessions.SessionState{Seen: &sessions.Seen{Tree: "t", Head: head, Report: report, At: time.Now()}}); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(probationLines(root), "\n")
	for _, want := range []string{"loomux: at the last turn end that checked this commit, the lanes in probation reported:", "lint/go: failed (probation) [preset] 0.1s", "a.go:1: bad"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in %q", want, joined)
		}
	}
	// A long report is cut, and says how much is left out.
	var long strings.Builder
	for i := 1; i <= maxSeenLines+5; i++ {
		fmt.Fprintf(&long, "finding %d\n", i)
	}
	sessions.WriteState(root, "earlier", sessions.SessionState{Seen: &sessions.Seen{Tree: "t", Head: head, Report: long.String(), At: time.Now()}})
	joined = strings.Join(probationLines(root), "\n")
	if !strings.Contains(joined, fmt.Sprintf("finding %d\n", maxSeenLines)) || strings.Contains(joined, fmt.Sprintf("finding %d", maxSeenLines+1)) ||
		!strings.Contains(joined, "loomux: 5 more lines; `loomux check stop` shows all") {
		t.Errorf("cut: %q", joined)
	}
	// A report of exactly the limit is passed on whole, with nothing cut.
	long.Reset()
	for i := 1; i <= maxSeenLines; i++ {
		fmt.Fprintf(&long, "finding %d\n", i)
	}
	sessions.WriteState(root, "earlier", sessions.SessionState{Seen: &sessions.Seen{Tree: "t", Head: head, Report: long.String(), At: time.Now()}})
	if joined = strings.Join(probationLines(root), "\n"); !strings.HasSuffix(joined, fmt.Sprintf("finding %d", maxSeenLines)) || strings.Contains(joined, "more lines") {
		t.Errorf("at the limit: %q", joined)
	}
	// A commit since then: the stand is about another HEAD and stays unsaid.
	writeWorldFile(t, root, "b.txt", "x")
	git(t, root, "add", "b.txt")
	git(t, root, "commit", "-q", "-m", "second")
	if joined = strings.Join(probationLines(root), "\n"); strings.Contains(joined, "finding") || !strings.Contains(joined, "lane lint/go@. is in probation") {
		t.Errorf("after a commit: %q", joined)
	}
}

// agy fires the start before every model call; the lanes are named at the
// first and not again, as the stale binary is.
func TestARepeatedStartDoesNotNameTheProbationAgain(t *testing.T) {
	root := probationProject(t, "armed = []\n")
	for payload, named := range map[string]bool{
		`{"conversationId":"s1","invocationNum":0}`: true,
		`{"conversationId":"s1","invocationNum":2}`: false,
	} {
		var out, errOut bytes.Buffer
		code := SessionStart(strings.NewReader(payload), &out, &errOut, root, "antigravity", "1.0.0")
		if code != ExitOK || strings.Contains(out.String(), "is in probation") != named {
			t.Errorf("%s: %d %q %q", payload, code, out.String(), errOut.String())
		}
	}
}

func TestStatusShowsTheProbation(t *testing.T) {
	root := probationProject(t, "")
	var b bytes.Buffer
	if renderProbation(&b, root); b.Len() != 0 {
		t.Fatalf("no file: %q", b.String())
	}
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = [\"lint/go@.\", \"lint/python@.\"]\n")
	renderProbation(&b, root)
	for _, want := range []string{
		" Lane Probation (.loomux/armed.toml)\n",
		" [ARMED] lint/go@.\n",
		" [PROBATION] test/go@.: warns only until a green commit arms it\n",
		" [ORPHAN] lint/python@.: no lane answers to this entry\n",
		" [WARN] no pre-commit hook arms lanes: ",
	} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("missing %q in %q", want, b.String())
		}
	}
	b.Reset()
	writeWorldFile(t, root, ".loomux/armed.toml", "armed = 1\n")
	if renderProbation(&b, root); !strings.Contains(b.String(), " [WARN] .loomux/armed.toml ") {
		t.Errorf("unreadable: %q", b.String())
	}
}

// A project that ignores .loomux keeps the file out of every commit; the
// report says so, and says nothing where only the state is ignored.
func TestStatusWarnsAboutAnIgnoredFile(t *testing.T) {
	const warning = " [WARN] .loomux/armed.toml is ignored by git: it reaches no commit and holds on this machine only\n"
	root := probationProject(t, "armed = []\n")
	var b bytes.Buffer
	if renderProbation(&b, root); strings.Contains(b.String(), "ignored by git") {
		t.Fatalf("nothing is ignored: %q", b.String())
	}
	writeWorldFile(t, root, ".gitignore", "/.loomux/state/\n")
	b.Reset()
	if renderProbation(&b, root); strings.Contains(b.String(), "ignored by git") {
		t.Fatalf("only the state is ignored: %q", b.String())
	}
	writeWorldFile(t, root, ".gitignore", ".loomux/\n")
	b.Reset()
	if renderProbation(&b, root); !strings.Contains(b.String(), warning) {
		t.Fatalf("%q", b.String())
	}
}

func TestStatusNamesAHookThatDoesNotArm(t *testing.T) {
	root := probationProject(t, "armed = []\n")
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexec loomux check precommit\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	renderProbation(&b, root)
	// By its tail: git spells the directory its own way, and a runner's TEMP
	// is a short name.
	if !strings.Contains(b.String(), "/.git/hooks/pre-commit does not arm lanes: ") || strings.Contains(b.String(), "no pre-commit hook") {
		t.Fatalf("%q", b.String())
	}
	b.Reset()
	os.WriteFile(hook, []byte("#!/bin/sh\nloomux check precommit --arm\n"), 0o755)
	if renderProbation(&b, root); strings.Contains(b.String(), "[WARN]") {
		t.Fatalf("a hook that arms is warned about: %q", b.String())
	}
	// Outside a repository there is no hook to name.
	outside := project(t)
	writeWorldFile(t, outside, ".loomux/armed.toml", "armed = []\n")
	b.Reset()
	if renderProbation(&b, outside); !strings.Contains(b.String(), " [WARN] no pre-commit hook arms lanes: ") {
		t.Fatalf("no repository: %q", b.String())
	}
}
```

  `project` und `gitInit` stehen in `hook_session_start_test.go`, `headOf` und `writeWorldFile` in `stop_test.go`, `git` in `worktree_test.go`.

- [ ] **Step 2: rot laufen lassen.** Stubs: `HooksDir` gibt `"", nil`, `probationLines` gibt `nil`, `renderProbation` schreibt nichts, `maxSeenLines = 40`. Run: `go test ./internal/gitwork -run TestHooksDir -count=1` und `go test ./internal/hooks -run "TestSessionStartNamesTheLanes|TestSessionStartSaysNothingWithoutALane|TestSessionStartSaysAnUnreadable|TestSessionStartPassesOn|TestARepeatedStart|TestStatusShows|TestStatusNames|TestStatusWarns" -count=1`. Expected: FAIL mit Assertions; `TestSessionStartSaysNothingWithoutALaneInProbation` ist schon grün.

- [ ] **Step 3: implementieren.** `internal/gitwork/gitwork.go`:

```go
// HooksDir is the directory git runs root's hooks from, absolute:
// core.hooksPath where it is set, expanded as git expands it, and git's own
// hook directory otherwise.
func HooksDir(root string) (string, error) {
	out, err := git(root, "rev-parse", "--path-format=absolute", "--git-path", "hooks")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
```

  `internal/hooks/probation.go`:

```go
package hooks

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/sessions"
	"github.com/xidus90/loomux/internal/verify"
)

// maxSeenLines is how much of the stop gate's last report session start
// passes on: enough to say what is wrong, not a context of findings.
const maxSeenLines = 40

// howToArm is what both reports tell a project whose hook arms nothing.
const howToArm = "call `loomux check precommit --arm` there, or arm by hand with `loomux gate arm`"

// probationLines are what session start tells an agent about the gate: one
// line per lane in probation, and what the stop gate last found in them while
// HEAD is where it found it. The stop gate ends a turn over such findings
// with 0, where no host shows its stderr, so this is where they are heard.
// Nothing without the file or without a lane in probation.
func probationLines(root string) []string {
	states, exists, err := ReadLaneStates(root)
	if err != nil {
		return []string{"loomux: " + err.Error()}
	}
	if !exists || len(states.Probation) == 0 {
		return nil
	}
	var lines []string
	for _, key := range states.Probation {
		lines = append(lines, "loomux: lane "+key+" is in probation: what it finds warns and fails no gate until a green commit arms it")
	}
	seen, found := sessions.LastSeen(root)
	if head, err := gitwork.Head(root); !found || err != nil || head != seen.Head {
		return lines
	}
	report := strings.Split(strings.TrimSuffix(seen.Report, "\n"), "\n")
	lines = append(lines, "loomux: at the last turn end that checked this commit, the lanes in probation reported:")
	lines = append(lines, report[:min(len(report), maxSeenLines)]...)
	if rest := len(report) - maxSeenLines; rest > 0 {
		lines = append(lines, fmt.Sprintf("loomux: %d more lines; `loomux check stop` shows all", rest))
	}
	return lines
}

// preCommitHook is the hook git runs before a commit of root, by path and
// text; false where there is none or no repository to hold one.
func preCommitHook(root string) (path, text string, found bool) {
	dir, err := gitwork.HooksDir(root)
	if err != nil {
		return "", "", false
	}
	path = filepath.Join(dir, "pre-commit")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false
	}
	return filepath.ToSlash(path), string(data), true
}

// renderProbation is the status report's section on the armed lanes. A
// project without the file gets none: its report reads as it always has.
func renderProbation(w io.Writer, root string) {
	states, exists, err := ReadLaneStates(root)
	if err == nil && !exists {
		return
	}
	fmt.Fprintln(w, "\n--------------------------------------------------------------------------------")
	fmt.Fprintf(w, " Lane Probation (%s)\n", verify.ArmedFile)
	fmt.Fprintln(w, "--------------------------------------------------------------------------------")
	if err != nil {
		fmt.Fprintf(w, " [WARN] %v\n", err)
		return
	}
	for _, key := range states.Armed {
		fmt.Fprintf(w, " [ARMED] %s\n", key)
	}
	for _, key := range states.Probation {
		fmt.Fprintf(w, " [PROBATION] %s: warns only until a green commit arms it\n", key)
	}
	for _, key := range states.Orphans {
		fmt.Fprintf(w, " [ORPHAN] %s: no lane answers to this entry\n", key)
	}
	switch path, text, found := preCommitHook(root); {
	case !found:
		fmt.Fprintf(w, " [WARN] no pre-commit hook arms lanes: %s\n", howToArm)
	case !verify.HookArms(text):
		fmt.Fprintf(w, " [WARN] %s does not arm lanes: %s\n", path, howToArm)
	}
	if states.Ignored {
		fmt.Fprintf(w, " [WARN] %s is ignored by git: it reaches no commit and holds on this machine only\n", verify.ArmedFile)
	}
}
```

  `internal/hooks/hook_session_start.go`, im Block `if !payload.Repeat`, hinter `updateWarnings`:

```go
		lines = append(lines, probationLines(root)...)
```

  mit dem Kommentar darüber ergänzt: „… and the lanes in probation are named once: agy's PreInvocation comes before every model call, and planning the lanes each time would cost every one of them.“

  `internal/hooks/status.go`, in `Status` direkt hinter `renderLaneTools(stdout, unavailableLanes(eff, exec.LookPath))`:

```go
	renderProbation(stdout, root)
```

- [ ] **Step 4: grün.** Run: `go test ./internal/gitwork ./internal/hooks -count=1` und `go test ./internal/cli -run "TestCases|TestHooksNever" -count=1`. Expected: PASS; die aufgezeichneten `hook-session-start`-Fälle unverändert, die Importgrenzen halten (`probation.go` importiert nichts aus `internal/setup`).
- [ ] **Step 5: Mutation per `go test -overlay`:**
  - `probationLines`: `|| len(states.Probation) == 0` entfernen: `TestSessionStartSaysNothingWithoutALaneInProbation` (Fall „every lane armed“, der gemerkte Stand würde gesagt) wird rot.
  - `probationLines`: `|| head != seen.Head` entfernen: `TestSessionStartPassesOn…` (letzter Teil) wird rot.
  - `probationLines`: `rest > 0` durch `rest >= 0` ersetzen: `TestSessionStartPassesOn…` („at the limit“) wird rot.
  - `renderProbation`: `!verify.HookArms(text)` durch `verify.HookArms(text)` ersetzen: `TestStatusNamesAHookThatDoesNotArm` wird rot.
  - `renderProbation`: `if states.Ignored` durch `if !states.Ignored` ersetzen: `TestStatusWarnsAboutAnIgnoredFile` wird rot.
  - Die Zeile in `SessionStart` aus dem Block `if !payload.Repeat` heraus hinter ihn verschieben: `TestARepeatedStartDoesNotNameTheProbationAgain` wird rot.
- [ ] **Step 6: Commit** `feat(hooks): show the lanes in probation at session start and in status`.

---

### Task 10: `init` und die Hook-Vorlage — neue Projekte starten in Probe, der Hook stellt scharf

**Files:**
- Modify: `internal/setup/gitfiles/gitfiles.go` (`Hooks`; neu `preCommit`, `Upgrade`, `IsLoomuxPreCommit`, `preCommitMarker`)
- Modify: `internal/setup/write/atomic.go` (`mode` wird zu `Mode` exportiert, Aufrufer und Tests folgen)
- Modify: `internal/setup/plan.go` (`Build`, `gitHooks`, `builder`), `internal/setup/apply.go` (`change`)
- Modify: `docs/.superpowers/parity/stufe-4a-2.md`
- Test: `internal/setup/gitfiles/gitfiles_test.go`, `internal/setup/plan_test.go`, `internal/setup/apply_test.go`, `internal/cli/init_test.go`, `internal/cli/check_arm_hook_test.go` (neu)

**Interfaces:**
- Consumes: `verify.ArmedFile`, `verify.ArmedSet.Text`, `verify.HookArms` (Task 2); `loomux check precommit --arm`, das selbst schreibt, den Teilcommit erkennt und stagt (Task 4); `hookBinary` (`internal/cli/check_hook_test.go`), `repo` (`internal/cli/graph_test.go`).
- Produces:
  - `gitfiles.Hooks(binary)["pre-commit"]` in der neuen Form (unten wörtlich).
  - `func Upgrade(text string) (string, bool)` in `gitfiles` — der eigene ältere Einzeiler in der neuen Form, für dasselbe Binary; `false` für jeden anderen Text.
  - `func IsLoomuxPreCommit(text string) bool` in `gitfiles` — ob ein Hook-Text die Kennzeile `# loomux pre-commit hook:` trägt, in alter oder neuer Form, auch mit eigenen Zeilen eines Menschen dahinter.
  - `func Mode(name, body string) fs.FileMode` in `internal/setup/write`.
  - `const armedPath = verify.ArmedFile` in `internal/setup`; die Naht `var chmod = os.Chmod` in `apply.go`.

**Die neue Vorlage, Byte für Byte** (`<b>` ist das Binary ohne die umgebenden Anführungszeichen): derselbe Einzeiler wie bisher, mit `--arm`, und `exec` bleibt.

```sh
#!/bin/sh
# loomux pre-commit hook: the check chain of .loomux/config.toml.
exec "<b>" check precommit --arm
```

Der Hook stagt nichts: Das tut der Befehl (Task 4), der als Einziger weiß, ob der Commit den ganzen Index nimmt.

**Wann `init` die Datei anlegt (Entscheidung 7 der Spec):** genau dann, wenn vor dem Lauf weder `.loomux/config.toml` noch `.loomux/armed.toml` stand **und im Hook-Verzeichnis kein pre-commit-Hook von loomux** (Kennzeile `# loomux pre-commit hook:`) — gleich, ob der Lauf selbst eine Konfiguration schreibt (in einem reinen Go-Projekt schreibt sie erst `area add`). Der Hook ist das Zeichen, dass das Projekt loomux schon eingerichtet hatte: Ein Go-Projekt, das loomux seit Monaten ohne Konfiguration nutzt, behält sein scharfes Tor, auch wenn dieses `init` seinen Hook erneuert; ohne diese Bedingung fiele dort mit `init --yes` jede Lane in die Probe, ohne dass jemand den Diff sähe. Ein fremder Hook zählt nicht. Das Hook-Verzeichnis ist das, in dem `init` die Hooks sucht und schreibt: `core.hooksPath`, wo gesetzt; gits eigenes, wo dort schon Hooks leben; sonst `.githooks`. Die Datei gehört zum Teil `config`: Wer diesen Teil abwählt, bekommt auch sie nicht.

- [ ] **Step 1: die Tests der Vorlage schreiben** (`internal/setup/gitfiles/gitfiles_test.go`)

```go
func TestThePreCommitHookArms(t *testing.T) {
	want := "#!/bin/sh\n" +
		"# loomux pre-commit hook: the check chain of .loomux/config.toml.\n" +
		"exec \"/opt/loomux\" check precommit --arm\n"
	if got := Hooks("/opt/loomux")["pre-commit"]; got != want {
		t.Fatalf("pre-commit = %q", got)
	}
	if got := Hooks(`"/opt/loomux"`)["pre-commit"]; got != want {
		t.Fatalf("a quoted binary: %q", got)
	}
	if !RunsAGate(want) {
		t.Fatal("our own hook no longer counts as a gate")
	}
	// The command stages the file itself, into the index git handed the hook;
	// a `git add` in the script would pull it into a commit of paths.
	if strings.Contains(want, "git add") {
		t.Fatal("the hook stages")
	}
}

// Only the hook init wrote before is ours to replace: the marker line and
// the one call, nothing else. Anything a human added makes it theirs.
func TestUpgradeKnowsOnlyTheHookInitWroteBefore(t *testing.T) {
	const old = "#!/bin/sh\n# loomux pre-commit hook: the check chain of .loomux/config.toml.\nexec \"${LOCALAPPDATA}/loomux/bin/loomux.exe\" check precommit\n"
	got, ok := Upgrade(old)
	if !ok || got != Hooks("${LOCALAPPDATA}/loomux/bin/loomux.exe")["pre-commit"] {
		t.Fatalf("%v %q", ok, got)
	}
	if got, ok = Upgrade(strings.ReplaceAll(old, "\n", "\r\n")); !ok || strings.Contains(got, "\r") {
		t.Fatalf("CRLF: %v %q", ok, got)
	}
	if got, ok = Upgrade(strings.Replace(old, "${LOCALAPPDATA}/loomux/bin/loomux.exe", "./bin/loomux.exe", 1)); !ok || !strings.Contains(got, "exec \"./bin/loomux.exe\" check precommit --arm\n") {
		t.Fatalf("the binary the old hook called is kept: %v %q", ok, got)
	}
	for name, text := range map[string]string{
		"the new form":    Hooks("/opt/loomux")["pre-commit"],
		"no marker":       "#!/bin/sh\n# our gate\nexec \"/opt/loomux\" check precommit\n",
		"a line added":    old + "echo done\n",
		"another call":    strings.Replace(old, "check precommit", "check all", 1),
		"no exec":         strings.Replace(old, "exec \"", "\"", 1),
		"another shebang": strings.Replace(old, "#!/bin/sh", "#!/bin/bash", 1),
		"a foreign gate":  "#!/bin/sh\nsh ci/gate.sh\n",
		"empty":           "",
	} {
		if got, ok := Upgrade(text); ok || got != "" {
			t.Errorf("%s: %v %q", name, ok, got)
		}
	}
}

// The marker line says loomux wrote the hook, whatever form it has and
// whatever a human added since; a hook that only mentions the words does not.
func TestIsLoomuxPreCommitGoesByTheMarkerLine(t *testing.T) {
	const old = "#!/bin/sh\n# loomux pre-commit hook: the check chain of .loomux/config.toml.\nexec \"/opt/loomux\" check precommit\n"
	for name, c := range map[string]struct {
		text string
		want bool
	}{
		"the old form":       {old, true},
		"the new form":       {Hooks("/opt/loomux")["pre-commit"], true},
		"a line added":       {old + "echo done\n", true},
		"CRLF":               {strings.ReplaceAll(old, "\n", "\r\n"), true},
		"a foreign gate":     {"#!/bin/sh\nsh ci/gate.sh\n", false},
		"the words mid-line": {"#!/bin/sh\necho '# loomux pre-commit hook: no'\n", false},
		"the other hook":     {Hooks("/opt/loomux")["commit-msg"], false},
		"empty":              {"", false},
	} {
		if got := IsLoomuxPreCommit(c.text); got != c.want {
			t.Errorf("%s: %v, want %v", name, got, c.want)
		}
	}
}
```

  Den alten Text hält nur `gitfiles_test.go:36` genau (`exec "` + inner + `" check precommit` + "\n"): Die Zeile wird rot und bekommt `--arm`. `:63` und die Tabelle `:140-148` sind `strings.Contains`-Prüfungen, die grün bleiben; sie bekommen `--arm` ebenfalls, damit sie die neue Form halten.

- [ ] **Step 2: die Tests von `init` schreiben.** In `internal/setup/plan_test.go`:

```go
const oldPreCommit = "#!/bin/sh\n# loomux pre-commit hook: the check chain of .loomux/config.toml.\nexec \"${LOCALAPPDATA}/loomux/bin/loomux.exe\" check precommit\n"

// withoutArea is the default choice with the area part off: a run that
// writes no configuration at all into a plain Go project.
func withoutArea(t *testing.T, root string) Plan {
	t.Helper()
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	c.Parts["area"] = false
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// A project that had neither a configuration, nor the file, nor a pre-commit
// hook of loomux before the run starts in probation -- whether or not the
// run writes a configuration. In a plain Go project init writes none
// itself; area add does, or nobody.
func TestAProjectWithoutAConfigurationStartsInProbation(t *testing.T) {
	empty := verify.ArmedSet{Exists: true}.Text()
	// area add writes the configuration.
	root := world(t, map[string]string{"go.mod": goMod, ".git/": ""})
	p := plan(t, gather(t, root, ""))
	if _, ok := changeOf(p, configPath); ok || !slices.Contains(actions(p), "area-add") {
		t.Fatalf("the world must leave the configuration to area add: %v %v", paths(p), actions(p))
	}
	if ch, ok := changeOf(p, armedPath); !ok || ch.After != empty || ch.Exists || ch.Part != "config" {
		t.Fatalf("beside area add: %+v %v", ch, ok)
	}
	// Nobody writes one: the file is planned all the same.
	p = withoutArea(t, root)
	if _, ok := changeOf(p, configPath); ok || slices.Contains(actions(p), "area-add") {
		t.Fatalf("the world must write no configuration: %v %v", paths(p), actions(p))
	}
	if ch, ok := changeOf(p, armedPath); !ok || ch.After != empty {
		t.Fatalf("without any configuration: %+v %v", ch, ok)
	}
	// init writes one itself: a uv project gets a rule.
	root = world(t, map[string]string{"uv.lock": "", ".git/": ""})
	p = plan(t, gather(t, root, ""))
	if _, ok := changeOf(p, configPath); !ok {
		t.Fatalf("the world must plan a configuration: %v", paths(p))
	}
	if ch, ok := changeOf(p, armedPath); !ok || ch.After != empty {
		t.Fatalf("beside init's own configuration: %+v %v", ch, ok)
	}
}

// A pre-commit hook of loomux says the project was set up before: a Go
// project that has run loomux for months without a configuration keeps its
// armed gate when init renews its hook. The old one-liner counts, the new
// form counts, and so does a hook a human added a line to; a foreign hook
// does not, and neither does a loomux hook where git does not look.
func TestALoomuxHookMeansTheProjectWasSetUp(t *testing.T) {
	newForm := gitfiles.Hooks("/opt/loomux")["pre-commit"]
	for name, c := range map[string]struct {
		files     map[string]string
		hooksPath string
		starts    bool
	}{
		"nothing":                       {map[string]string{}, "", true},
		"the old loomux hook":           {map[string]string{".githooks/pre-commit": oldPreCommit}, ".githooks", false},
		"the new loomux hook":           {map[string]string{".githooks/pre-commit": newForm}, ".githooks", false},
		"a loomux hook with a line more": {map[string]string{".githooks/pre-commit": oldPreCommit + "echo done\n"}, ".githooks", false},
		"a fresh clone, no hooksPath yet": {map[string]string{".githooks/pre-commit": oldPreCommit}, "", false},
		"a loomux hook under core.hooksPath": {map[string]string{"tools/hooks/pre-commit": oldPreCommit}, "tools/hooks", false},
		"a foreign hook":                {map[string]string{".githooks/pre-commit": "#!/bin/sh\nsh ci/gate.sh\n"}, ".githooks", true},
		"a loomux hook git does not run": {map[string]string{"old-hooks/pre-commit": oldPreCommit}, ".githooks", true},
	} {
		files := map[string]string{"go.mod": goMod, ".git/": ""}
		for path, text := range c.files {
			files[path] = text
		}
		root := world(t, files)
		if _, ok := changeOf(plan(t, gather(t, root, c.hooksPath)), armedPath); ok != c.starts {
			t.Errorf("%s: armed.toml planned %v, want %v", name, ok, c.starts)
		}
	}
	// core.hooksPath outside the project: init writes no hook there and reads
	// none, so nothing says the project was set up.
	outside := t.TempDir()
	writeFile(t, outside, "pre-commit", oldPreCommit)
	root := world(t, map[string]string{"go.mod": goMod, ".git/": ""})
	if _, ok := changeOf(plan(t, gather(t, root, outside)), armedPath); !ok {
		t.Error("a hook directory outside the project kept the project out of probation")
	}
}

// A project that has a configuration is not put into probation behind its
// back, a file that stands is left as it is, and the file goes with the
// config part.
func TestAProjectThatIsSetUpGetsNoProbation(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod, ".git/": "", configPath: "[commit]\nlanguage = \"de\"\n"})
	if _, ok := changeOf(plan(t, gather(t, root, "")), armedPath); ok {
		t.Fatal("armed.toml planned over a standing configuration")
	}
	// The file without a configuration: a human's `gate disarm --all`, or a
	// run of init before area add was chosen. It is not written over.
	root = world(t, map[string]string{"go.mod": goMod, ".git/": "", armedPath: "armed = [\"lint/go@.\"]\n"})
	for name, p := range map[string]Plan{"with area add": plan(t, gather(t, root, "")), "without": withoutArea(t, root)} {
		if _, ok := changeOf(p, armedPath); ok {
			t.Errorf("%s: a standing armed.toml is planned over: %v", name, paths(p))
		}
	}
	// The config part deselected: nothing of it is written, the file included.
	root = world(t, map[string]string{"go.mod": goMod, ".git/": ""})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	c.Parts["config"] = false
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := changeOf(p, armedPath); ok {
		t.Fatalf("armed.toml planned without the config part: %v", paths(p))
	}
}

func TestOurOwnOlderPreCommitHookIsReplaced(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod, ".git/": "", ".githooks/pre-commit": oldPreCommit})
	p := plan(t, gather(t, root, ".githooks"))
	ch, ok := changeOf(p, ".githooks/pre-commit")
	if !ok || !ch.Exists || ch.Before != oldPreCommit || !verify.HookArms(ch.After) || ch.Binary != "" {
		t.Fatalf("%+v %v", ch, ok)
	}
	if hasNote(p, ".githooks/pre-commit: kept") || hasNote(p, "does not arm lanes") {
		t.Fatalf("notes %v", p.Notes)
	}
	// Renewing the hook starts no probation: the hook says loomux was here.
	if _, ok := changeOf(p, armedPath); ok {
		t.Fatalf("armed.toml planned beside the renewed hook: %v", paths(p))
	}
	// Replaced once: the new form plans nothing and is named as kept.
	root = world(t, map[string]string{"go.mod": goMod, ".git/": "", ".githooks/pre-commit": ch.After, configPath: "[commit]\nlanguage = \"de\"\n"})
	p = plan(t, gather(t, root, ".githooks"))
	if _, ok := changeOf(p, ".githooks/pre-commit"); ok || !hasNote(p, ".githooks/pre-commit: kept; it runs a gate already") || hasNote(p, "does not arm lanes") {
		t.Fatalf("%v %v", paths(p), p.Notes)
	}
}

// A hook init did not write stays, and where the project has or gets the
// file, the plan says that this hook arms nothing.
func TestAForeignPreCommitHookIsKeptAndNamed(t *testing.T) {
	const foreign = "#!/bin/sh\nsh ci/gate.sh\n"
	const note = ".githooks/pre-commit: does not arm lanes; call `loomux check precommit --arm` there, or arm by hand with `loomux gate arm`"
	root := world(t, map[string]string{"go.mod": goMod, ".git/": "", ".githooks/pre-commit": foreign})
	p := plan(t, gather(t, root, ".githooks"))
	if _, ok := changeOf(p, ".githooks/pre-commit"); ok || !hasNote(p, ".githooks/pre-commit: kept; it runs a gate already") || !hasNote(p, note) {
		t.Fatalf("a new project: %v %v", paths(p), p.Notes)
	}
	// Without the file, now or after this run, there is nothing to arm.
	root = world(t, map[string]string{"go.mod": goMod, ".git/": "", ".githooks/pre-commit": foreign, configPath: "[commit]\nlanguage = \"de\"\n"})
	if p = plan(t, gather(t, root, ".githooks")); hasNote(p, "does not arm lanes") {
		t.Fatalf("no file: %v", p.Notes)
	}
	root = world(t, map[string]string{"go.mod": goMod, ".git/": "", ".githooks/pre-commit": foreign, configPath: "[commit]\nlanguage = \"de\"\n", armedPath: "armed = []\n"})
	if p = plan(t, gather(t, root, ".githooks")); !hasNote(p, note) {
		t.Fatalf("a project in probation: %v", p.Notes)
	}
	// A foreign hook that arms is not named.
	root = world(t, map[string]string{"go.mod": goMod, ".git/": "", ".githooks/pre-commit": "#!/bin/sh\nloomux check precommit --arm\n", armedPath: "armed = []\n", configPath: "[commit]\nlanguage = \"de\"\n"})
	if p = plan(t, gather(t, root, ".githooks")); hasNote(p, "does not arm lanes") {
		t.Fatalf("a hook that arms: %v", p.Notes)
	}
}
```

  Die Welt mit `uv.lock` löst die Katalogregel aus, wie `TestTheRuleCatalogFollowsTheStacks` (`configtext_test.go:78`) es zeigt. Die Testdateien importieren dazu `github.com/xidus90/loomux/internal/verify`, `apply_test.go` außerdem `io/fs`. `goMod`, `world`, `gather`, `plan`, `changeOf`, `hasNote`, `paths`, `actions`, `reader` stehen in `helpers_test.go`. Die vorhandenen Assertions `plan_test.go:132` und `:362` sind `strings.Contains`-Prüfungen, die auch mit der neuen Vorlage grün bleiben; sie bekommen trotzdem `--arm` ans Ende (`exec "…" check precommit --arm`), damit sie die neue Form halten. **Rot wird `TestAFreshRepositoryGetsEveryPart` (`plan_test.go:106-115`):** Seine Liste der geplanten Pfade ist genau und bekommt `.loomux/armed.toml` als erstes Element (die Datei wird direkt hinter dem Konfigurationsblock geplant, vor `.gitignore`):

```go
	want := []string{
		".loomux/armed.toml",
		".gitignore", "AGENTS.md", ".mcp.json", ".claude/settings.json",
		".githooks/commit-msg", ".githooks/pre-commit", ".githooks/pre-push",
		".claude/skills/verify-until-green/SKILL.md",
		".claude/skills/brain-ingest/SKILL.md", ".claude/skills/brain-land/SKILL.md",
		".claude/skills/brain-research/SKILL.md", ".claude/skills/brain-review/SKILL.md",
		".claude/skills/brain-wiki-plan/SKILL.md",
	}
```

  `plan_test.go` importiert dazu `github.com/xidus90/loomux/internal/setup/gitfiles`, soweit es fehlt. `TestSelfUseChangesNothing` bleibt unverändert grün (der Hook von loomux ist fremd, es gibt eine Konfiguration und keine Datei, also weder Änderung noch neue Notiz).

  In `internal/setup/apply_test.go`:

```go
// A replaced script stays a script: the swap goes through a temporary file
// whose mode is not the hook's.
func TestAReplacedFileGetsTheModeOfANewOne(t *testing.T) {
	root := world(t, map[string]string{".githooks/pre-commit": "#!/bin/sh\nold\n", "data.json": "{}\n"})
	var got []string
	old := chmod
	chmod = func(path string, mode fs.FileMode) error {
		got = append(got, filepath.ToSlash(strings.TrimPrefix(path, root))+" "+mode.String())
		return nil
	}
	t.Cleanup(func() { chmod = old })
	p := Plan{Changes: []Change{
		{Part: "git-hooks", Path: ".githooks/pre-commit", Before: "#!/bin/sh\nold\n", After: "#!/bin/sh\nnew\n", Exists: true},
		{Part: "mcp-json", Path: "data.json", Before: "{}\n", After: "{\"a\":1}\n", Exists: true},
	}}
	if _, err := Apply(root, p, Choice{}, all, func(Action) error { return nil }, there, "1.0.0", time.Now()); err != nil {
		t.Fatal(err)
	}
	want := []string{"/.githooks/pre-commit -rwxr-xr-x", "/data.json -rw-r--r--"}
	if !slices.Equal(got, want) {
		t.Fatalf("chmod %v, want %v", got, want)
	}
	chmod = func(string, fs.FileMode) error { return errors.New("no chmod") }
	p.Changes[0].Before, p.Changes[0].After = "#!/bin/sh\nnew\n", "#!/bin/sh\nnewer\n"
	if _, err := Apply(root, Plan{Changes: p.Changes[:1]}, Choice{}, all, func(Action) error { return nil }, there, "1.0.0", time.Now()); err == nil || !strings.Contains(err.Error(), ".githooks/pre-commit") {
		t.Fatalf("a chmod that fails: %v", err)
	}
}
```

  In `internal/cli/init_test.go` (Import `github.com/xidus90/loomux/internal/verify` dazu). In `TestInitYesSetsUpAFreshRepository` bekommt die Liste der Dateien, die nach dem Lauf da sind, eine Zeile; der Kommentar darüber bleibt, denn die Konfiguration schreibt weiter `area add`:

```go
		".gitignore", "AGENTS.md", ".mcp.json", ".claude/settings.json",
		".loomux/armed.toml",
```

  In `TestInitDryRunShowsEveryChangeAndWritesNothing` wird die Liste der erwarteten Ausschnitte:

```go
		for _, want := range []string{"--- .claude/settings.json", "--- .githooks/pre-commit", "--- .loomux/armed.toml", "binary-install", "+"} {
```

  Und ein neuer Test ans Ende der Datei:

```go
// A project loomux is set up in for the first time starts in probation, and
// only then: the second run finds a configuration, the file and its own
// hook, and leaves what a commit or a human armed since.
func TestInitStartsANewProjectInProbationOnce(t *testing.T) {
	root, _ := initWorld(t)
	if code, out, errOut := run("init", "--root", root, "--yes"); code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if got := readAt(t, root, ".loomux/armed.toml"); got != (verify.ArmedSet{Exists: true}).Text() {
		t.Fatalf("armed.toml %q", got)
	}
	if !verify.HookArms(readAt(t, root, ".githooks/pre-commit")) {
		t.Fatalf("the hook init wrote does not arm:\n%s", readAt(t, root, ".githooks/pre-commit"))
	}
	const armed = "armed = [\"lint/go@.\"]\n"
	writeAt(t, filepath.Join(root, ".loomux", "armed.toml"), armed)
	if code, out, errOut := run("init", "--root", root, "--yes"); code != 0 || readAt(t, root, ".loomux/armed.toml") != armed {
		t.Fatalf("second run: code %d, armed.toml %q: %s\n%s", code, readAt(t, root, ".loomux/armed.toml"), errOut, out)
	}
	// Taking the file away does not bring the probation back: the project
	// is set up, and without the file every lane is armed.
	if err := os.Remove(filepath.Join(root, ".loomux", "armed.toml")); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := run("init", "--root", root, "--yes"); code != 0 || there(root, ".loomux/armed.toml") {
		t.Fatalf("third run: code %d, the file is back: %s", code, errOut)
	}
}
```

- [ ] **Step 3: den Weltentest mit echtem git und echtem Hook schreiben** (`internal/cli/check_arm_hook_test.go`). Der Hook ist der, den `init` schreibt; alles Schreiben und Stagen tut das echte Binary unter dem Index, den git ihm gibt.

```go
package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/setup/gitfiles"
)

const (
	formatted   = "package x\n\nfunc F() {\n}\n"
	unformatted = "package x\n\nfunc F() {\nreturn\n}\n"
)

// armWorld is a committed repository with two project lanes that the loomux
// binary itself judges -- lint is red when a/ holds an unformatted file,
// types when b/ does -- the file holding armed, and as its pre-commit hook
// hook, or the one init writes when hook is "". The project ignores only its
// state, as init sets it up.
func armWorld(t *testing.T, binary, armed, hook string) (root string, gitIn func(args ...string) (string, error)) {
	t.Helper()
	root = repo(t, map[string]string{
		".gitignore":          "/.loomux/state/\n",
		".loomux/config.toml": "[verify.project]\nlint = \"{loomux} check gofmt a\"\ntypes = \"{loomux} check gofmt b\"\n",
		".loomux/armed.toml":  armed,
		"a/x.go":              formatted,
		"b/y.go":              formatted,
		"c.txt":               "c\n",
	})
	if hook == "" {
		hook = gitfiles.Hooks(filepath.ToSlash(binary))["pre-commit"]
	}
	if err := os.MkdirAll(filepath.Join(root, ".githooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".githooks", "pre-commit"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn = func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = gitenv.Environ()
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"}, {"config", "commit.gpgsign", "false"},
		{"add", "."}, {"commit", "-qm", "init", "--no-verify"},
		{"config", "core.hooksPath", ".githooks"},
	} {
		if out, err := gitIn(args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root, gitIn
}

func put(t *testing.T, root, rel, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func must(t *testing.T, gitIn func(...string) (string, error), args ...string) string {
	t.Helper()
	out, err := gitIn(args...)
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(out)
}

func armedAtHead(t *testing.T, gitIn func(...string) (string, error)) string {
	t.Helper()
	return must(t, gitIn, "show", "HEAD:.loomux/armed.toml")
}

func armedOnDisk(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".loomux", "armed.toml"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(data))
}

func TestTheHookArmsTheGreenLanesIntoTheSameCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	binary := hookBinary(t)
	none := strings.TrimSpace(armedText())
	both := strings.TrimSpace(armedText("lint/project@.", "types/project@."))

	// (a) and (b)
	t.Run("a commit arms the green lanes, and the file is in it", func(t *testing.T) {
		root, gitIn := armWorld(t, binary, armedText(), "")
		put(t, root, "c.txt", "c2\n")
		must(t, gitIn, "add", "c.txt")
		out := must(t, gitIn, "commit", "-m", "change c")
		if !strings.Contains(out, "armed: lint/project@., types/project@.") {
			t.Fatalf("the hook said %q", out)
		}
		if names := must(t, gitIn, "show", "--name-only", "--format=", "HEAD"); !strings.Contains(names, ".loomux/armed.toml") || !strings.Contains(names, "c.txt") {
			t.Fatalf("the commit holds %q", names)
		}
		if got := armedAtHead(t, gitIn); got != both {
			t.Fatalf("armed at HEAD: %q", got)
		}
		if status := must(t, gitIn, "status", "--porcelain"); status != "" {
			t.Fatalf("left dirty: %q", status)
		}
		// A second commit has nothing to arm and does not touch the file.
		put(t, root, "c.txt", "c3\n")
		must(t, gitIn, "add", "c.txt")
		must(t, gitIn, "commit", "-qm", "change c again")
		if names := must(t, gitIn, "show", "--name-only", "--format=", "HEAD"); names != "c.txt" {
			t.Fatalf("the second commit holds %q", names)
		}
	})

	// (c)
	t.Run("a red lane in probation lets the commit through, an armed one holds it", func(t *testing.T) {
		armed := armedText("lint/project@.")
		root, gitIn := armWorld(t, binary, armed, "")
		put(t, root, "b/y.go", unformatted)
		out := must(t, gitIn, "commit", "-am", "break the lane in probation")
		if !strings.Contains(out, "types/project: failed (probation)") || armedAtHead(t, gitIn) != strings.TrimSpace(armed) {
			t.Fatalf("out %q, armed %q", out, armedAtHead(t, gitIn))
		}
		// Now the armed lane is red and the other green: refused, and the
		// green lane is not armed by a run that did not go through.
		put(t, root, "b/y.go", formatted)
		put(t, root, "a/x.go", unformatted)
		commits := must(t, gitIn, "rev-list", "--count", "HEAD")
		out, err := gitIn("commit", "-am", "break the armed lane")
		if err == nil || !strings.Contains(out, "lint/project: failed [config]") {
			t.Fatalf("the commit went through: %v\n%s", err, out)
		}
		if armedOnDisk(t, root) != strings.TrimSpace(armed) {
			t.Fatalf("a refused commit wrote the file: %q", armedOnDisk(t, root))
		}
		if staged := must(t, gitIn, "diff", "--cached", "--name-only"); strings.Contains(staged, "armed.toml") {
			t.Fatalf("a refused commit left the file in the index: %q", staged)
		}
		if status := must(t, gitIn, "status", "--porcelain", "--", ".loomux/armed.toml"); status != "" || must(t, gitIn, "rev-list", "--count", "HEAD") != commits {
			t.Fatalf("status %q", status)
		}
	})

	// (d)
	for name, commit := range map[string][]string{
		"git commit <path>":        {"commit", "-m", "only a", "a/x.go"},
		"git commit --only <path>": {"commit", "--only", "-m", "only a", "a/x.go"},
	} {
		t.Run("a partial commit arms nothing: "+name, func(t *testing.T) {
			root, gitIn := armWorld(t, binary, armedText(), "")
			put(t, root, "a/x.go", formatted+"\n// more\n")
			put(t, root, "c.txt", "c2\n")
			out := must(t, gitIn, commit...)
			if !strings.Contains(out, "not armed: this commit takes only some paths") {
				t.Fatalf("the hook said %q", out)
			}
			if names := must(t, gitIn, "show", "--name-only", "--format=", "HEAD"); names != "a/x.go" {
				t.Fatalf("the commit holds %q", names)
			}
			// File and index as they were: no staged revert, nothing to see.
			if armedOnDisk(t, root) != none || armedAtHead(t, gitIn) != none {
				t.Fatalf("armed on disk %q, at HEAD %q", armedOnDisk(t, root), armedAtHead(t, gitIn))
			}
			if status := must(t, gitIn, "status", "--porcelain", "--", ".loomux/armed.toml"); status != "" {
				t.Fatalf("after the partial commit: %q", status)
			}
			// The next ordinary commit arms.
			must(t, gitIn, "add", "c.txt")
			must(t, gitIn, "commit", "-qm", "c")
			if status := must(t, gitIn, "status", "--porcelain"); status != "" || armedAtHead(t, gitIn) != both {
				t.Fatalf("after the next commit: status %q, armed %q", status, armedAtHead(t, gitIn))
			}
		})
	}

	// (e)
	t.Run("git commit -a arms like a plain commit", func(t *testing.T) {
		root, gitIn := armWorld(t, binary, armedText(), "")
		put(t, root, "c.txt", "c2\n")
		must(t, gitIn, "commit", "-qam", "change c")
		if names := must(t, gitIn, "show", "--name-only", "--format=", "HEAD"); !strings.Contains(names, ".loomux/armed.toml") || !strings.Contains(names, "c.txt") {
			t.Fatalf("the commit holds %q", names)
		}
		if status := must(t, gitIn, "status", "--porcelain"); status != "" || armedAtHead(t, gitIn) != both {
			t.Fatalf("status %q, armed %q", status, armedAtHead(t, gitIn))
		}
	})

	// (f)
	t.Run("--no-verify arms nothing", func(t *testing.T) {
		root, gitIn := armWorld(t, binary, armedText(), "")
		put(t, root, "c.txt", "c2\n")
		must(t, gitIn, "commit", "-qam", "unchecked", "--no-verify")
		if armedAtHead(t, gitIn) != none || armedOnDisk(t, root) != none {
			t.Fatalf("armed at HEAD %q, on disk %q", armedAtHead(t, gitIn), armedOnDisk(t, root))
		}
	})

	// (g)
	t.Run("a hook without --arm never arms", func(t *testing.T) {
		old := "#!/bin/sh\n# loomux pre-commit hook: the check chain of .loomux/config.toml.\nexec \"" + filepath.ToSlash(binary) + "\" check precommit\n"
		root, gitIn := armWorld(t, binary, armedText(), old)
		put(t, root, "c.txt", "c2\n")
		out := must(t, gitIn, "commit", "-am", "through the old hook")
		if !strings.Contains(out, "probation: lint/project@., types/project@.") || armedAtHead(t, gitIn) != none || armedOnDisk(t, root) != none {
			t.Fatalf("out %q, armed %q", out, armedAtHead(t, gitIn))
		}
	})
}
```

  `armedText` steht in `check_arming_test.go` (Task 4). Zeigt ein Fall etwas anderes, als hier steht — der Teilcommit stellt scharf, `-a` stellt nicht scharf, `git status` ist nach einem Fall nicht leer —, ist das ein Befund an `gitwork.CommitIndex` oder an der Messung von Task 4, keine Zeile zum Anpassen: Der Implementierer meldet die Ausgabe samt `git --version` und hält an.

- [ ] **Step 4: rot laufen lassen.** Stubs: `Upgrade` gibt `"", false`, `IsLoomuxPreCommit` gibt `false`; `write.Mode` ist die umbenannte `mode`; `armedPath` und `chmod` sind angelegt und unbenutzt. Run: `go test ./internal/setup/... -run "TestThePreCommitHookArms|TestUpgradeKnows|TestAProjectWithoutAConfiguration|TestALoomuxHookMeans|TestAProjectThatIsSetUp|TestOurOwnOlder|TestAForeignPreCommit|TestAReplacedFile|TestAFreshRepositoryGetsEveryPart" -count=1` und `go test ./internal/cli -run "TestTheHookArmsTheGreenLanes|TestInitYesSetsUp|TestInitDryRunShows|TestInitStartsANewProject" -count=1`. Expected: FAIL mit Assertions, darunter `pre-commit = "#!/bin/sh\n# loomux pre-commit hook: …\nexec \"/opt/loomux\" check precommit\n"` und im Weltentest `the hook said "…probation: lint/project@., types/project@. …"` (die alte Vorlage stellt nicht scharf); `TestAProjectThatIsSetUpGetsNoProbation` ist schon grün.

- [ ] **Step 5: implementieren.**

  `internal/setup/gitfiles/gitfiles.go`: in `Hooks` wird der Eintrag zu `"pre-commit": preCommit(b),` und dazu

```go
// preCommitMarker opens the comment line of the pre-commit hook init writes;
// Upgrade knows its own older hook by it.
const preCommitMarker = "# loomux pre-commit hook:"

// preCommit is the pre-commit hook of a host project for the binary b: the
// gate with --arm, which enters the lanes a green run found ok and stages the
// file itself. The script stages nothing: only the command knows whether the
// commit takes the whole index, and a `git add` here would pull the file
// into a commit of paths.
func preCommit(b string) string {
	return "#!/bin/sh\n" +
		preCommitMarker + " the check chain of .loomux/config.toml.\n" +
		`exec "` + b + `" check precommit --arm` + "\n"
}

// Upgrade is the pre-commit hook init wrote before lanes could be armed --
// the shebang, the marker line and the one call `exec "<binary>" check
// precommit`, nothing else -- in today's form for the same binary. False for
// every other text: a hook with a line of its own is the project's.
func Upgrade(text string) (string, bool) {
	lines := strings.Split(strings.TrimSuffix(strings.ReplaceAll(text, "\r\n", "\n"), "\n"), "\n")
	if len(lines) != 3 || lines[0] != "#!/bin/sh" || !strings.HasPrefix(lines[1], preCommitMarker) {
		return "", false
	}
	b, ok := strings.CutPrefix(lines[2], `exec "`)
	if !ok {
		return "", false
	}
	b, ok = strings.CutSuffix(b, `" check precommit`)
	if !ok {
		return "", false
	}
	return preCommit(b), true
}

// IsLoomuxPreCommit says whether a hook's text carries the marker line of the
// pre-commit hook init writes, in the old form or today's, with or without
// lines a human added: the sign that loomux was set up in this project.
func IsLoomuxPreCommit(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, preCommitMarker) {
			return true
		}
	}
	return false
}
```

  `internal/setup/plan.go`: `builder` bekommt das Feld `unarming string` („the pre-commit hook that stays and does not arm, by path“); `armedPath` steht neben `configPath` (`const armedPath = verify.ArmedFile`). In `Build`, direkt hinter dem Block `if on("config") { … }`:

```go
	// A project loomux was not set up in before this run starts in
	// probation: no lane fails its gate before a green commit armed it. Set
	// up means a configuration, the armed lanes, or a pre-commit hook of
	// loomux: a Go project can run loomux for months without a configuration,
	// and the run that renews its hook must not drop its armed gate into
	// probation. Whether this run writes a configuration does not matter --
	// in a plain Go project area add does, or nobody. A file that stands is
	// left as it is.
	_, configThere := b.file(configPath)
	_, armedThere := b.file(armedPath)
	starts := on("config") && !configThere && !armedThere && !b.loomuxHookThere()
	if starts {
		b.change("config", armedPath, nil, false, verify.ArmedSet{Exists: true}.Text())
	}
```

  und direkt vor `if b.err != nil`:

```go
	if b.unarming != "" && (starts || armedThere) {
		b.note(b.unarming + ": does not arm lanes; call `loomux check precommit --arm` there, or arm by hand with `loomux gate arm`")
	}
```

  In `gitHooks` wird die Schleife:

```go
	for _, name := range names {
		path := dir + "/" + name
		existing, exists := b.file(path)
		upgraded, ours := gitfiles.Upgrade(string(existing))
		switch {
		case exists && name == "pre-commit" && ours:
			// Our own hook from before lanes could be armed: the same call
			// for the same binary, in today's form. It calls no binary it
			// did not call before, so nothing about it can be missing.
			b.add(Change{Part: "git-hooks", Path: path, Before: string(existing), After: upgraded, Exists: true})
		case exists && gitfiles.RunsAGate(string(existing)):
			b.note(path + ": kept; it runs a gate already")
			b.keptHook(name, path, string(existing))
		case exists:
			b.note(path + ": kept; a hook of the project is already there")
			b.keptHook(name, path, string(existing))
		default:
			b.add(Change{Part: "git-hooks", Path: path, After: hooks[name], Binary: b.f.Binary})
		}
	}
```

  und

```go
// keptHook remembers a pre-commit hook that stays as it is and does not arm;
// Build names it once it knows whether the project has lanes to arm.
func (b *builder) keptHook(name, path, text string) {
	if name == "pre-commit" && !verify.HookArms(text) {
		b.unarming = path
	}
}

// standingHookDir is hookDir's answer without its notes and its action: the
// directory, relative to the root, that init looks for the project's hooks
// in, "" where that lies outside the project. It is asked whether or not
// the git-hooks part is chosen, and may plan nothing.
func (b *builder) standingHookDir() string {
	switch {
	case b.f.HooksPath != "":
		rel, _ := within(b.f.Root, b.f.HooksPath)
		return rel
	case b.f.GitHooksLive:
		rel, _ := within(b.f.Root, b.f.GitHooksDir)
		return rel
	}
	return ".githooks"
}

// loomuxHookThere says whether the project has a pre-commit hook loomux
// wrote, in the old form or today's: the sign that loomux was set up here
// before this run, configuration or not.
func (b *builder) loomuxHookThere() bool {
	dir := b.standingHookDir()
	if dir == "" {
		return false
	}
	text, _ := b.file(dir + "/pre-commit")
	return gitfiles.IsLoomuxPreCommit(string(text))
}
```

  `internal/setup/apply.go`: der Import `os` kommt dazu (die Datei hat ihn heute nicht), die Naht `var chmod = os.Chmod` („replaced by tests: Windows has no execute bit to read back“) und in `change`, im Ersetzungszweig, der jede bestehende Datei trifft, die `init` ersetzt, nicht nur Hooks: Eine ersetzte `.claude/settings.json` oder `.mcp.json` bekommt 0644 statt der 0600 der temporären Datei, wie eine neu angelegte; das ist gewollt und steht in der Paritätsakte.

```go
	err = write.CheckParents(a.root, ch.Path)
	full := filepath.Join(a.root, filepath.FromSlash(ch.Path))
	if err == nil {
		err = lock.ReplaceText(full, ch.After)
	}
	if err == nil {
		// The swap goes through a temporary file of its own mode; the file
		// gets the mode a new one with this text would have, so a replaced
		// hook stays executable.
		err = chmod(full, write.Mode(ch.Path, ch.After))
	}
	if err != nil {
		return fmt.Errorf("%s: %w", ch.Path, err)
	}
```

  `internal/setup/write/atomic.go`: `mode` heißt `Mode`, der Kommentar beginnt mit „Mode is …“, Aufrufer und Tests im Paket folgen.

- [ ] **Step 6: die Paritätsakte ergänzen.** In `docs/.superpowers/parity/stufe-4a-2.md` direkt vor `## Offen` ein Abschnitt (deutsch):

  ```
  ## Schonfrist je Lane, 2026-09-30

  `init` hat keine aufgezeichnete Referenz (siehe „Referenzen“); die folgenden
  Änderungen halten darum Einheits- und Weltentests, keine genehmigte Liste.

  | Was sich ändert | Vorher | Nachher | Test |
  |---|---|---|---|
  | pre-commit-Hook eines Wirts | `exec "<binary>" check precommit` | `exec "<binary>" check precommit --arm`; der Befehl schreibt und stagt `.loomux/armed.toml` selbst, außer bei einem Teilcommit | `TestThePreCommitHookArms`, `TestTheHookArmsTheGreenLanesIntoTheSameCommit` |
  | eigener älterer Hook | blieb stehen („kept; it runs a gate already“) | wird ersetzt, mit Diff und Sicherung unter `.loomux/state/backup/` | `TestOurOwnOlderPreCommitHookIsReplaced`, `TestUpgradeKnowsOnlyTheHookInitWroteBefore` |
  | fremder Hook | blieb stehen | bleibt stehen; wo das Projekt `.loomux/armed.toml` hat oder bekommt, nennt der Plan, dass er nicht scharf stellt | `TestAForeignPreCommitHookIsKeptAndNamed` |
  | `.loomux/armed.toml` | gab es nicht | entsteht leer, wenn vor dem Lauf weder sie noch `.loomux/config.toml` noch ein pre-commit-Hook von loomux stand und der Teil `config` gewählt ist | `TestAProjectWithoutAConfigurationStartsInProbation`, `TestALoomuxHookMeansTheProjectWasSetUp`, `TestAProjectThatIsSetUpGetsNoProbation`, `TestInitStartsANewProjectInProbationOnce` |
  | Modus einer ersetzten Datei, jeder, nicht nur eines Hooks | der der temporären Datei (0600 auf POSIX) | der einer neu angelegten (0755 für ein Skript, sonst 0644) | `TestAReplacedFileGetsTheModeOfANewOne` |
  ```

- [ ] **Step 7: grün.** Run: `go test ./internal/setup/... -count=1` und `go test ./internal/cli -run "TestInit|TestTheHookArmsTheGreenLanes|TestCases|TestHooksNever" -count=1`. Expected: PASS. `TestSelfUseChangesNothing` ist unverändert grün: ein frischer Klon von loomux hat eine Konfiguration und bekommt weder die Datei noch eine neue Notiz.
- [ ] **Step 8: Mutation per `go test -overlay`:**
  - `preCommit`: `--arm` entfernen: `TestThePreCommitHookArms` und Weltentest Fall a werden rot.
  - `Upgrade`: `len(lines) != 3 ||` entfernen: `TestUpgradeKnows…` („a line added“) wird rot.
  - `Build`: `!configThere &&` entfernen: `TestAProjectThatIsSetUpGetsNoProbation` (erster Fall) wird rot.
  - `Build`: `&& !armedThere` entfernen: derselbe Test (zweiter Fall) wird rot.
  - `Build`: `on("config") &&` entfernen: derselbe Test (dritter Fall) wird rot.
  - `Build`: `&& !b.loomuxHookThere()` entfernen: `TestALoomuxHookMeansTheProjectWasSetUp` (vier Fälle) und `TestOurOwnOlderPreCommitHookIsReplaced` werden rot.
  - `standingHookDir`: immer `.githooks` antworten: derselbe Test („a loomux hook under core.hooksPath“) wird rot.
  - `loomuxHookThere`: den Zweig `if dir == ""` entfernen: bleibt grün (unter dem leeren Namen findet `b.file` nichts); als äquivalent in den Bericht.
  - `loomuxHookThere`: statt `gitfiles.IsLoomuxPreCommit` das `ok` von `gitfiles.Upgrade` fragen: derselbe Test („the new loomux hook“, „a loomux hook with a line more“) wird rot.
  - `Build`: `starts` zusätzlich an eine geplante Konfiguration binden (`&& slices.ContainsFunc(b.plan.Changes, …configPath…)`): `TestAProjectWithoutAConfigurationStartsInProbation` (erste zwei Fälle) wird rot.
  - `Build`: `(starts || armedThere)` durch `true` ersetzen: `TestAForeignPreCommitHookIsKeptAndNamed` („no file“) wird rot.
  - `gitHooks`: `name == "pre-commit" &&` entfernen: bleibt grün (nur der pre-commit-Text passt auf `Upgrade`); als äquivalent in den Bericht.
  - `change`: die `chmod`-Zeile entfernen: `TestAReplacedFileGetsTheModeOfANewOne` wird rot.
  - In `gitwork.CommitIndex` (Task 4) den Schluss `return "", false` durch `return index, true` ersetzen: Weltentest Fall d wird rot, beide Formen. Das ist die Gegenprobe, dass der Weltentest die Regel hält und nicht nur die Naht.
- [ ] **Step 9: Commit** `feat(setup): start a new project in probation and arm from the pre-commit hook`.

---

### Task 11: `apply.sh` — Schritt 4 legt die leere Datei mit an

Schritt 5 des Skripts ruft `init` erst, nachdem Schritt 4 die Konfiguration geschrieben hat; `init` sieht dann eine stehende Konfiguration und legt nichts an. Darum schreibt das Skript die Datei selbst, mit demselben Text wie `init`.

**Files:**
- Modify: `internal/switchover/apply.sh.tmpl` (Schritt 4)
- Test: `internal/switchover/apply_test.go`

**Interfaces:**
- Consumes: `verify.ArmedSet{Exists: true}.Text()` (Task 2) als der Text, dem das Skript gleichen muss.
- Produces: die Ausgabezeile `probation: started, .loomux/armed.toml written` (unter `--check`: `probation: would be started, .loomux/armed.toml written`).

- [ ] **Step 1: die Tests schreiben** (`internal/switchover/apply_test.go`)

```go
// Step 4 starts the probation where it writes the configuration: init runs
// after it, finds a configuration and would start none.
func TestApplyStartsTheProbationWhereItWritesTheConfiguration(t *testing.T) {
	w := newWorld(t)
	out := w.full()
	if got := w.read(w.project + "/.loomux/armed.toml"); got != (verify.ArmedSet{Exists: true}).Text() {
		t.Fatalf("armed.toml %q, want the text init writes", got)
	}
	if !strings.Contains(out, "config: written\nprobation: started, .loomux/armed.toml written\n") {
		t.Fatalf("stdout:\n%s", out)
	}
	// A second run finds the configuration and leaves the lanes a human or a
	// commit armed since.
	w.write(w.project+"/.loomux/armed.toml", "armed = [\"lint/go@.\"]\n")
	w.commitAll(w.project, "switch over")
	before := w.snapshot()
	code, out, errOut := w.run()
	if code != 0 || strings.Contains(out, "probation:") {
		t.Fatalf("code %d:\n%s\n%s", code, out, errOut)
	}
	w.sameAs(before)
}

// A project that has a configuration is not put into probation by the script.
func TestApplyStartsNoProbationOverAStandingConfiguration(t *testing.T) {
	w := newWorld(t)
	w.write(w.project+"/.loomux/config.toml", "[area]\nname = \"mine\"\n")
	w.commitAll(w.project, "own config")
	out := w.full()
	if exists(w.project+"/.loomux/armed.toml") || strings.Contains(out, "probation:") {
		t.Fatalf("armed.toml written over a standing configuration:\n%s", out)
	}
}
```

  `TestApplyCheckWritesNothing` bekommt die Assertion, dass stdout `probation: would be started, .loomux/armed.toml written\n` enthält und die Datei danach nicht existiert (die Schnappschuss-Prüfung des Tests deckt das zweite schon ab). `TestApplyMovesTheWikiIntoTheProject` bleibt unverändert grün: Seine Liste erwarteter Zeilen ist eine Teilmenge. **Rot wird `TestApplyWithOnlyTheRequiredParts` (`apply_test.go:692-696`):** Er vergleicht den ganzen Baum und nimmt nur die Konfiguration aus. Die Datei kommt dazu:

```go
	after := w.snapshot()
	delete(after, "/proj's/.loomux")
	delete(after, "/proj's/.loomux/config.toml")
	delete(after, "/proj's/.loomux/armed.toml")
	if !reflect.DeepEqual(after, before) {
		t.Error("more than the configuration and the armed lanes changed")
	}
```

- [ ] **Step 2: rot laufen lassen.** Kein neues Go-Symbol. Run: `go test ./internal/switchover -run "TestApplyStartsTheProbation|TestApplyStartsNoProbation|TestApplyCheckWritesNothing" -count=1`. Expected: FAIL mit `armed.toml` nicht lesbar (der Helfer `w.read` bricht mit der fehlenden Datei ab) und der fehlenden `probation:`-Zeile; `TestApplyStartsNoProbationOverAStandingConfiguration` ist schon grün.
- [ ] **Step 3: implementieren.** Schritt 4 der Vorlage wird:

```sh
# 4. The configuration, never over an existing one. With it the armed lanes,
#    empty: no lane fails the gate of a project that never had one before a
#    green commit armed it. init runs after this step, finds a configuration
#    and starts no probation of its own; the text is the one init writes.
if [ -e "$PROJECT/.loomux/config.toml" ]; then
  say "config: kept, $PROJECT/.loomux/config.toml exists"
else
  run mkdir -p "$PROJECT/.loomux"
  run cp "$CONFIG_NEW" "$PROJECT/.loomux/config.toml"
  step config "written"
  if [ "$CHECK" = 0 ]; then
    printf '%s\n' \
      '# Lanes that are armed: a red run of one of them fails the gate.' \
      '# Written by the pre-commit gate; a human edits it through `loomux gate`.' \
      'armed = []' > "$PROJECT/.loomux/armed.toml"
  fi
  step probation "started, .loomux/armed.toml written"
fi
```

  Die Backticks stehen in einfachen Anführungszeichen und werden von `sh` nicht ausgeführt. Der Kopfkommentar des Skripts und die Platzhalter bleiben unverändert; `Render` prüft keine dieser Zeilen.
- [ ] **Step 4: grün.** Run: `go test ./internal/switchover -count=1`. Expected: PASS, auch `TestApplyASecondTimeChangesNothing` (der zweite Lauf nimmt den Zweig „kept“).
- [ ] **Step 5: Mutation (von Hand an der Vorlage, über `LOOMUX_SWITCHOVER_TEMPLATE` auf eine Kopie im Scratchpad, siehe `templateUnderTest`):**
  - Den `printf`-Block aus dem `else`-Zweig hinter das `fi` verschieben (er läuft dann auch über einer stehenden Konfiguration): `TestApplyStartsNoProbationOverAStandingConfiguration` und der zweite Teil von `TestApplyStartsTheProbation…` werden rot.
  - `if [ "$CHECK" = 0 ]` um den `printf` entfernen: `TestApplyCheckWritesNothing` wird rot.
  - Eine der drei Textzeilen ändern (`armed = [ ]`): `TestApplyStartsTheProbation…` wird rot (der Text weicht von `init` ab).
- [ ] **Step 6: Commit** `feat(switchover): start the probation where the script writes the configuration`.

---

### Task 12: Doku — die Datei, die Probe, die Befehle

**Files:**
- Modify: `docs/en/configuration.md`, `docs/de/configuration.md`
- Modify: `docs/en/hooks.md`, `docs/de/hooks.md`
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md`
- Modify: `README.md`, `README.de.md` (die Fähigkeit; **keine** Roadmap-Zeile, Entscheidung 14 der Spec)

**Interfaces:** keine.

- [ ] **Step 1: `docs/en/configuration.md`.** Ein neuer Abschnitt `### Lane probation: `.loomux/armed.toml`` direkt hinter dem Abschnitt `### `[verify]` (Check Chains & Quality Gates)` und vor `### `[modules]``, mit diesem Inhalt (englisch, in dieser Reihenfolge):
  - Was es ist: „A lane that has never been green in a project warns instead of failing. `.loomux/armed.toml` names the lanes that are armed: a red run of one of them fails the gate. It is versioned, so every contributor and the CI read the same lanes.“ Darunter das Beispiel der Datei aus der Spec (zwei Schlüssel).
  - Die drei Zustände der Datei als Tabelle: keine Datei — jede Lane scharf, jede Ausgabe wie ohne das Feature; Datei — jede Lane, die nicht darin steht, ist in Probe, auch eine, die später dazukommt; unlesbare Datei (kein TOML, Konfliktmarken, leere Datei, unbekannter Schlüssel, falscher Typ) — jede Lane scharf, jeder Lauf meldet es auf stderr.
  - Der Schlüssel: `<kind>/<stack>@<area>`, Bereich mit Vorwärtsschrägstrichen, `.` für die Wurzel; er trägt den Bereich immer, anders als der Name im Bericht (`lint/go` wird zu `lint/go@.`, sobald ein zweiter Bereich dazukommt, der Schlüssel bleibt `lint/go@.`); `lint/wiki@.` und `<kind>/project@.` für die projektweiten Lanes.
  - Was „in Probe“ heißt: die Lane läuft und zeigt ihre Befunde, ihr Zustand trägt `(probation)`, der Lauf endet mit `probation: <keys> (warn only until a green commit arms them)`; `blocked` hinter einer Lane in Probe ist nicht rot, hinter einer scharfen schon; Lanes ohne etwas zu prüfen (`not-applicable`, `unavailable`) stehen weder in der Zeile noch in `gate status`, `missing-tool` und `unready` stehen dort; das gilt gleich für `loomux check`, das pre-commit-Tor, den Stop-Hook, den Post-Edit-Hook und das CI.
  - Wie eine Lane scharf wird: nur `loomux check precommit --arm`, und nur, wenn der Lauf insgesamt grün endet; der Befehl schreibt die Datei und legt sie selbst in den Index des Commits; der Hook, den `init` schreibt, ruft ihn. Ein Commit mit `--no-verify`, ein eigener Hook ohne den Aufruf und das CI stellen nichts scharf; von Hand `loomux gate arm`.
  - **Teilcommit:** Bei `git commit <path>` und `git commit --only <path>` stellt der Lauf nichts scharf, schreibt nichts und sagt `not armed: this commit takes only some paths; the next whole commit arms the lanes`. Grund in zwei Sätzen: git gibt dem Hook dort einen Index, der nach dem Commit nicht der echte wird; eine dort gestagte Datei käme in den Commit, stünde im echten Index aber als gestagte Rücknahme. `git commit`, `git commit -a`, `--include` und `--amend` stellen scharf.
  - Wer die Datei anlegt: `init`, wenn vor dem Lauf weder `.loomux/config.toml` noch `.loomux/armed.toml` noch ein pre-commit-Hook von loomux stand (und der Teil `config` gewählt ist); das Umstellungsskript in seinem Konfigurationsschritt. **Ein Projekt, in dem loomux schon eingerichtet ist — es hat eine Konfiguration oder den pre-commit-Hook von loomux —, bekommt die Schonfrist nur durch `loomux gate disarm --all`, von einem Menschen ausgeführt;** `init` legt dort nichts an, auch nicht, wenn es den Hook erneuert.
  - Merge-Konflikt: ein Schlüssel je Zeile, sortiert; beide Seiten behalten; solange Konfliktmarken stehen, gilt „unlesbar“, also alles scharf. `merge=union` wird nicht gesetzt, weil es eine mit `gate disarm` entfernte Zeile zurückholte.
  - Zwei Warnhinweise: Die Datei darf nicht ignoriert sein (`init` ignoriert nur `/.loomux/state/`): Ignoriert das Projekt `.loomux/` oder die Datei, erreicht sie nie einen Commit und gilt nur auf diesem Rechner; `loomux gate status` und `loomux status` warnen davor. Ein Agent schreibt die Datei nie selbst: Der Wächter verweigert es, scharf stellt er nur über einen grünen Commit.
  - In `## 4. Where Things Live` eine Zeile für `.loomux/armed.toml` (versioniert, vom pre-commit-Tor und von `loomux gate` geschrieben).
- [ ] **Step 2: `docs/de/configuration.md`.** Derselbe Abschnitt an derselben Stelle (`### Schonfrist je Lane: `.loomux/armed.toml`` hinter `### `[verify]` (Prüfketten & Quality-Gates)`), Satz für Satz deutsch, gleiche Reihenfolge, gleiche Tabelle, gleiches Beispiel; die Zeile in `## 4. Wo was liegt`.
- [ ] **Step 3: `docs/en/hooks.md` und `docs/de/hooks.md`.** Fünf Stellen, in beiden Sprachen an der gleichnamigen Stelle:
  - `## 5. The Post-Edit Lanes by Stack` / `## 5. Die post-edit-Lanes je Sprachstack`: ein Absatz: mit `.loomux/armed.toml` hält eine rote Lane in Probe die Bearbeitung nicht (Exit 0); ihr Befund geht mit `(probation)` über den Kontextkanal des Wirts an den Agenten (Claude Code `hookSpecificOutput.additionalContext`, Antigravity `injectSteps`), weil stderr bei Exit 0 niemand liest; eine scharfe rote Lane blockiert wie bisher.
  - `### `session-start``: ein Listenpunkt „**Names the lanes in probation.**“: eine Kontextzeile je Lane, dazu der Bericht, den der Stop-Hook zuletzt für diesen `HEAD` gemerkt hat, auf 40 Zeilen gekürzt; nur beim ersten Start; nichts ohne Datei oder ohne Lane in Probe.
  - `### `stop``: ein Absatz „A chain red only in lanes in probation“: Exit 0; der Blockzähler geht auf 0, weil das Rundenende durchging; `base` und grüner Baum bleiben, der Lauf ist nicht grün; der Hook merkt sich Baum, `HEAD` und Bericht in der Sitzungsdatei (`seen`); derselbe Baum unter demselben `HEAD` startet kein Werkzeug und wiederholt den Bericht; ein geänderter Baum, ein bewegter `HEAD` oder eine geänderte `armed.toml` (sie gehört zum Baum) lassen die Kette neu laufen. Der Satz, dass stderr bei Exit 0 den Agenten nicht erreicht und darum der Session-Start der Kanal ist. In `### The session state` / `### Der Sitzungszustand` der Schlüssel `seen` mit seinen fünf Feldern (`tree`, `head`, `armed`, `report`, `at`) und der Satz, warum die scharfen Lanes dazugehören: In einem Projekt, das `.loomux/` ignoriert, liegt die Datei nicht im Baum.
  - `## 7. The Decision Path of `pre-tool-use`` / `## 7. Der Entscheidungsweg von `pre-tool-use``: `.loomux/armed.toml` in der Liste der eingebauten Pfadregeln neben `.loomux/config.toml`, und `loomux gate arm|disarm` in der Liste der Befehle, die nur ein Mensch ausführt (`gate status` bleibt erlaubt). Dazu die **benannten Lücken**, wörtlich wie im Kommentar von `armedReason`: `git checkout <rev> -- .loomux`, `git checkout <rev> -- .`, `git restore -s <rev> .`, `git stash`, `git reset --hard`, `git switch` und `sh -c "loomux gate disarm --all"` gehen durch und können Lanes entschärfen; die Datei teilt sie mit der Konfiguration. Und der Satz, dass die Ablehnung von `rm -rf .loomux` einen Grund mehr nennt, auch in einem Projekt ohne die Datei.
  - `## 10. Git Repository Hooks`, Unterabschnitt pre-commit: ein Absatz, der den Hook eines Wirts beschreibt (`exec "<binary>" check precommit --arm`, das Schreiben und Stagen tut der Befehl, der Teilcommit stellt nichts scharf), und der Satz, dass `.githooks/pre-commit` und `ci/gate.sh` von loomux selbst unverändert sind: loomux hat keine `armed.toml`.
- [ ] **Step 4: `docs/en/cli-reference.md` und `docs/de/cli-reference.md`.**
  - In `## 2. …(`loomux check`)`: die Überschrift `### `loomux check <request> [--root <path>] [--show] [-v]`` bekommt `[--arm]` (deutsch entsprechend), darunter ein Absatz zu `--arm` (nur beim Profil `precommit`, sonst Exit 2; schreibt nur nach einem insgesamt grünen Lauf und nur, wenn die Datei existiert; stagt die Datei mit `git add` in den Index des Commits; bei einem Teilcommit weder Schreiben noch Stagen, Zeile `not armed: …`; Ausgabezeile `armed: …`; ein Schreib- oder Stage-Fehler steht auf stderr und ändert den Exit-Code nicht) und zu `(probation)` und der `probation:`-Zeile.
  - Ein neuer Abschnitt `## 13. The Gate (`loomux gate`)` / `## 13. Das Tor (`loomux gate`)` am Ende, mit je einem Unterabschnitt für `loomux gate status [--root <dir>]` (die drei Zustände, die Zeile ohne Datei, ein Beispiel der Ausgabe, die Warnung auf stderr bei einer von git ignorierten Datei), `loomux gate arm <lane>... [--root <dir>]` (trägt ein, auch eine rote Lane; unbekannter Schlüssel ist zuerst ein Fehler, mit und ohne Datei, und nichts wird geschrieben; ohne Datei wird nichts geschrieben, weil ohnehin alles scharf ist) und `loomux gate disarm <lane>...|--all [--root <dir>]` (entfernt Einträge, auch verwaiste; ohne Datei entsteht sie mit jeder anderen Lane als scharf; `--all` legt sie leer an oder leert sie: **der Weg, einem schon eingerichteten Projekt die Schonfrist zu geben**), den Exit-Codes (0, 1 für eine Datei oder Lanes, die sich nicht lesen oder schreiben lassen, und für einen unbekannten Schlüssel, 2 für einen falschen Aufruf) und dem Satz zum Wächter (`arm` und `disarm` führt nur ein Mensch aus).
  - In `## 4. …(`loomux status`)`: ein Satz zum Abschnitt „Lane Probation“ des Berichts (nur mit Datei; verwaiste Einträge; der Hinweis auf einen pre-commit-Hook, der nicht scharf stellt; die Warnung bei einer ignorierten Datei).
  - In `## 11. …(`loomux init`)`, „What it writes, and what it leaves“ / „Was es schreibt und was es stehen lässt“: `.loomux/armed.toml` (leer, nur wo vor dem Lauf weder sie noch eine Konfiguration noch ein pre-commit-Hook von loomux stand; ein eingerichtetes Projekt: `gate disarm --all`), der pre-commit-Hook mit `--arm`, das Ersetzen des eigenen älteren Hooks mit Sicherung und erhaltenem Ausführungsbit, die Notiz zu einem fremden Hook.
  - In `### `loomux dev switchover …``: Schritt 4 des Skripts legt die leere Datei mit an, wenn er die Konfiguration schreibt.
- [ ] **Step 5: `README.md` und `README.de.md`.** Wo die READMEs das Tor und `loomux check` beschreiben (Grep nach `check precommit` und nach dem Abschnitt, der die Befehle auflistet), ein Absatz von drei Sätzen zur Schonfrist mit Verweis auf `docs/en/configuration.md` bzw. `docs/de/configuration.md`, und `loomux gate` in der Befehlsliste. Der Abschnitt `## Roadmap` bleibt unberührt.
- [ ] **Step 6: prüfen.** Run: `go test ./internal/plancheck -count=1` (PASS). Grep in `docs/` und den READMEs nach `check precommit` ohne `--arm` im Zusammenhang mit dem Hook eines Wirts und nach Sätzen, die das alte Verhalten beschreiben („a red lane blocks the edit“, „blockiert die Bearbeitung“, „holds the turn“): jede Stelle lesen und, wo sie ohne den Zusatz zur Probe jetzt falsch ist, ergänzen. Überschriften in `docs/en` und `docs/de` gegeneinander halten: gleiche Anzahl, gleiche Reihenfolge. Grep nach `probation` und `Schonfrist` im Abschnitt `## Roadmap` beider READMEs: kein Treffer. Bytes `\r` in den acht Dateien zählen (Python-Skript per Write): 0.
- [ ] **Step 7: das ganze Tor.** Run: `sh ci/gate.sh > "<scratchpad>/gate-task12.txt" 2>&1`, danach lesen. Expected: Exit 0, Coverage 100 % je Funktion.
- [ ] **Step 8: Commit** `docs(verify): describe the lane probation`.
- [ ] **Step 9: Übergabe.** Der Controller ruft den Skill `release-pr`: Commits nach Thema gruppieren (Fixups falten), Label `release:minor` (neues Feature; ohne `.loomux/armed.toml` ändert sich kein Verhalten), im Rumpf `Release: minor — a lane that has never been green warns until a green commit arms it` und ein `## Changelog` mit `### Added` (`.loomux/armed.toml` und die Probe; `loomux check precommit --arm`; `loomux gate status|arm|disarm`; Lanes in Probe beim Session-Start und in `loomux status`) und `### Changed` (der pre-commit-Hook, den `init` in ein Projekt schreibt, ruft `check precommit --arm`; `init` ersetzt seinen älteren Hook; eine ersetzte Datei bekommt den Modus einer neu angelegten). Der Mensch pusht.

---

## Selbstprüfung

Zweiter Durchgang am 2026-09-30, gegen die Spec in der Fassung von `5d087f7b`; die Fixups bis `ba9290cb` sind eingearbeitet (siehe „Belege und Messungen“), und Task 0 hat die Zuordnung am 2026-10-01 nicht geändert.

**Entscheidungen der Spec → Task**

| Entscheidung | Task |
|---|---|
| 1 je Lane, 2 versioniert, 3 das Tor schreibt, 5 ohne Datei alles scharf | 2, 3, 4 |
| 4 nur der pre-commit-Lauf, nur insgesamt grün, die Datei im selben Commit | 4 (Schreiben und Stagen), 10 (Vorlage, Weltentest) |
| 6 sichtbar, von Hand scharf und zurück | 7, 9 |
| 7 `init` legt an, wenn weder Konfiguration noch Datei noch ein pre-commit-Hook von loomux stand; `apply.sh` | 10, 11 |
| 8 Stop-Hook merkt sich den Stand | 6 |
| 9 Session-Start | 9 |
| 10 Reihenfolge | 0, 1 |
| 11 Post-Edit warnt | 3 (`EditReport`), 5 (Kanal je Wirt, Test am JSON) |
| 12 Teilcommit stellt nicht scharf | 4 (Messschritt, `CommitIndex`, `checkArm`), 10 (Weltentest d) |
| 13 Blockzähler auf 0 | 6 |
| 14 keine Roadmap-Zeile | 1 und 12 (beide prüfen, dass keine entsteht) |

**Abschnitt der Spec → Task**

| Abschnitt der Spec | Task |
|---|---|
| Die Zustandsdatei (Schlüssel, Form, fehlt, unlesbar, verwaist) | 2; verwaist: 7 |
| Was „in Probe“ heißt (`Red`, `blocked`, `CheckVerdict`, Post-Edit, `probation:`-Zeile ohne `not-applicable`/`unavailable`) | 3, 4, 5; `gate status`: 7 |
| Scharf stellen: `--arm`, Hook von loomux unverändert | 4; unverändert: kein Task fasst `.githooks/pre-commit` oder `ci/gate.sh` an |
| Scharf stellen: Hook-Vorlage als Einzeiler mit `--arm`, Stagen im Befehl, `git add` scheitert | 10 (Vorlage), 4 (`checkArm`, `gitwork.Stage`) |
| Scharf stellen: Teilcommit, `os.SameFile`, Messung für sechs Aufrufformen | 4 (Step 0, `TestCommitIndex…`), 10 (Weltentest d, beide Formen) |
| Scharf stellen: Bestandsinstallationen, `--no-verify` | 10 |
| Die Befehle des Menschen (samt `disarm` ohne Datei, `arm` ohne Datei, unbekannter Schlüssel zuerst) | 7 |
| Der Wächter (Pfadregel, Shell-Formen, Befehlsregel, Differenzprobe, benannte Lücken, die Folge ohne Datei) | 8 |
| Der Stop-Hook (samt scharfer Menge im gemerkten Stand) | 6 |
| Sichtbarkeit: Session-Start, `status`/`doctor`, jeder Torlauf | 9, 4 |
| Sichtbarkeit: ignorierte Datei | 7 (`gate status`), 9 (`status`) |
| `init` und das Umstellungsskript, Ausführungsbit | 10, 11 |
| Fehlerfälle, Zeile für Zeile | fehlt/unlesbar: 2, 4; `--arm` ohne Datei, nicht schreibbar: 4; `git add` scheitert: 4 (`TestArmSaysAFileItCannotStage`); Merge-Konflikt: 2, 12; `gate arm` unbekannt: 7; Werkzeug fehlt: 3 (`missing-tool` bleibt in der Zeile), 7 |
| Tests und Abnahme: Urteil, Schlüssel | 3, 2 |
| Tests und Abnahme: Weltentest a–f | 10 (a, b, c, d in zwei Formen, e, f; dazu g: alter Hook) |
| Tests und Abnahme: Stop-Hook, Wächter, Invariante, `init`, `apply.sh` | 6, 8, jede „grün“-Stufe, 10, 11 |
| Doku und Papiere | 1, 10 (Paritätsakte), 12 |
| Nicht Teil dieses Entwurfs | kein Task |

**Namen und Signaturen über die Tasks:** `ArmedSet`, `LaneKey`, `ReadArmed`, `WriteArmed`, `HookArms`, `Missing`, `With`, `Without`, `Text` (Task 2) heißen in den Tasks 4, 5, 6, 7, 9, 10 und 11 gleich. `RunOptions.Armed`, `Outcome.Probation`, `Fails`, `ProbationLine(outs, armed)`, `GreenKeys` (Task 3) werden in 4, 5 und 6 mit derselben Signatur gerufen. `gitwork.CommitIndex(root, inherited) (index string, whole bool)` und `gitwork.Stage(root, index, rel) error` (Task 4) stehen in Task 4 hinter den Nähten `checkCommitIndex` und `checkStage` und in Task 10 (Gegenprobe im Weltentest) unter denselben Namen. `sessions.Seen` (`Tree`, `Head`, `Armed`, `Report`, `At`), `LastSeen` (Task 6) in 9. `gitfiles.IsLoomuxPreCommit`, `builder.standingHookDir`, `builder.loomuxHookThere` nur in Task 10. `GateLanes`, `LaneStates` (mit `Ignored`), `ReadLaneStates`, `gitwork.IgnoredPath(root, rel) bool` (Task 7) in 9. `gitwork.HooksDir` nur in Task 9. `armedReason`, `gateReason`, `armsOrDisarms` nur in Task 8. Der Hinweistext für einen Hook, der nicht scharf stellt, lautet in Task 9 (`howToArm`) und Task 10 (Notiz von `init`) gleich: „call `loomux check precommit --arm` there, or arm by hand with `loomux gate arm`“.

**Platzhalter:** keine Stelle mit „TBD“, „später“ oder „wie in Task N“. Jeder Testhelfer, den ein Schritt benutzt, steht entweder im Schritt selbst oder ist mit seiner Datei genannt und am Stand `47361ab3` nachgelesen, am 2026-10-01 an `e6c21eed` erneut (keine dieser Dateien hat sich dazwischen geändert außer `switchover/apply_test.go`, dort nur `TestApplyCleansUpWhenItIsStopped`) (`fakeStart`, `opts`, `job` in `verify/run_test.go`; `gitWorld`, `fakeTools`, `countTools`, `runStop`, `stateOf`, `headOf`, `writeWorldFile` in `hooks/stop_test.go`; `goProject`, `postEdit`, `filePayload`, `editEnv`, `agyCall`, `editContextOf`, `wikiProject`, `cleanPage` in `hooks/post_edit_test.go`; `project`, `gitInit` in `hooks/hook_session_start_test.go`; `git` in `hooks/worktree_test.go`; `mkfile` in `hooks/pathspell_test.go`; `repo`, `commit`, `run` in `gitwork/gitwork_test.go`; `stubCheck`, `goWorld` und die Variable `green` in `cli/check_test.go`; `run` in `cli/cli_test.go`; `repo` in `cli/graph_test.go`; `mustRunGit` in `cli/mergehook_test.go`; `hookBinary` in `cli/check_hook_test.go`; `world`, `gather`, `plan`, `changeOf`, `hasNote`, `paths`, `actions`, `reader`, `goMod` in `setup/helpers_test.go`; `all`, `there` in `setup/apply_test.go`; `newWorld` und die Methoden von `world` in `switchover/apply_test.go`). Task 0 liest sie nach dem Rebase noch einmal.

**Review Focus → Test:** jede der dreizehn Zeilen nennt ihren Test und Task in Klammern; die Tests stehen unter diesen Namen in den genannten Tasks.

## Entschieden am 2026-09-30

Die sieben offenen Fragen der ersten Fassung hat der Nutzer entschieden; die Spec trägt sie als Entscheidungen 7 und 11 bis 14 und in den Abschnitten „Die Befehle des Menschen“ und „Sichtbarkeit“.

1. Keine Roadmap-Zeile (Tasks 1 und 12).
2. `init` legt die Datei an, wenn vor dem Lauf weder Konfiguration noch Datei noch ein pre-commit-Hook von loomux stand, gleich, ob der Lauf eine Konfiguration schreibt (Task 10; die dritte Bedingung kam mit der Durchsicht des Plans dazu).
3. Post-Edit warnt; die Warnung geht über den Kontextkanal des Wirts (Task 5).
4. Ein Teilcommit stellt nichts scharf; Erkennen, Schreiben und Stagen liegen im Befehl, die Regel ist gemessen (Tasks 4 und 10).
5. Eine von git ignorierte Datei: `gate status` und `status` warnen (Tasks 7 und 9).
6. `gate arm` ohne Datei schreibt nichts; `gate disarm <lane>` ohne Datei legt sie mit jeder anderen Lane als scharf an (Task 7).
7. Ein nur in Probe roter Stop-Lauf setzt den Blockzähler auf 0 (Task 6).

## Durchsicht und Nachrechnung, 2026-09-30

Eine Durchsicht hat jeden Go-Block der Tasks 2 bis 11 übersetzt, die Suiten gefahren und 90 Mutanten gesetzt (Bericht: `.superpowers/sdd/2026-09-29-loomux-stufe-4e-umstellung/probation-plan-review.md`). Ihre Befunde sind eingearbeitet:

| Befund | Wo im Plan |
|---|---|
| B1 `check_arm_test.go` wird nur für `GOARCH=arm` gebaut | die Datei heißt `check_arming_test.go` (Tasks 4, 7, 10) |
| B2 Composite-Literale im Kopf von `if` und `range` | geklammert (Tasks 2 und 7) |
| B3, I4 `GateLanes`: nie gerufene Hülle, Lanes ohne Tests gelistet | `verify.HasTests`, `verify.ImportReady`, Filter auf `not-applicable` und `unavailable` (Task 7) |
| I1 `rm -rf .loomux` nennt einen Grund mehr | `runFileShellWrites` angepasst, als zweite Ausnahme von der Invariante benannt (Task 8, Global Constraints) |
| I2, I3 zwei vorhandene Tests mit genauen Listen | `TestAFreshRepositoryGetsEveryPart` (Task 10), `TestApplyWithOnlyTheRequiredParts` (Task 11) |
| I5 Schlüssel mit Backslash auf POSIX | `strings.ReplaceAll` statt `filepath.ToSlash` (Task 2) |
| I6 der Fall „no marker“ prüfte die Kennzeile nicht | drei Zeilen ohne Kennzeile (Task 10) |
| I7 ein Projekt, das loomux ohne Konfiguration nutzt, fiele in die Probe | dritte Bedingung: kein pre-commit-Hook von loomux (Task 10, Entscheidung 7 der Spec) |
| M5 vier überlebende Mutanten | vier Tests (Tasks 2, 4, 6) |
| M6 der gemerkte Stand kannte die scharfe Menge nicht | `Seen.Armed`, Vergleich beim Wiederholen (Task 6) |
| M7 sieben Schreibweisen, die entschärfen und durchgehen | `knownGaps` in der Batterie, im Kommentar von `armedReason` und in der Doku benannt, nicht geschlossen (Tasks 8 und 12) |
| M10 Messung ohne Worktree, Unterverzeichnis, Merge | Task 4 Step 0 erweitert, `commit -p` als ungemessen benannt |
| M11 unbekannter Schlüssel ohne Datei | zuerst ein Fehler (Task 7) |
| M3, M4, M9, M12, M13 | Fundstellen berichtigt, `os`-Import genannt, `chmod` für jede ersetzte Datei gesagt, `init_test.go` als Code, `realIndex` |

**Nachgerechnet nach dem Einarbeiten** (git 2.54.0.vfs.0.4, Windows, in einer Kopie des Baums unter dem Scratchpad, nie im Arbeitsbaum): Alle Go-Blöcke der Tasks 2 bis 11 und die Änderungen an den vorhandenen Tests wurden über die Kopie gelegt; `go build ./...` und `go vet` der betroffenen Pakete sind fehlerfrei. Die Batterie des Wächters wurde am Stand vor der Regel neu aufgezeichnet (278 Zeilen). Grün liefen, nacheinander: `internal/verify`, `internal/gitwork`, `internal/sessions`, `internal/setup/...`, `internal/switchover`, `internal/hooks` und `internal/cli` jeweils ganz, darin die aufgezeichneten Fälle (`TestCases…`) und der Weltentest mit dem echten Hook in allen sieben Fällen; dazu die Tests von `--arm` und `gate` unter einem gesetzten `GIT_INDEX_FILE`. Jede neue oder geänderte Funktion steht bei 100 % (je Paket gemessen). Von 16 Mutanten für die Regeln dieser Überarbeitung starben 15; der sechzehnte (`filepath.ToSlash` im Schlüssel) ist nur auf POSIX sichtbar.

**Nicht nachgerechnet:** die Läufe auf Linux und macOS (I5 und das Ausführungsbit sind dort entscheidend), die Mutanten der Skriptvorlage von Task 11, die Verschiebung der Zeile in `SessionStart` (Task 9), die übrigen 90 Mutanten der Durchsicht am geänderten Stand, die Doku-Schritte der Tasks 1 und 12 und `gofmt`: Elf der übergelegten Dateien würde `gofmt` noch an der Spaltenausrichtung ändern (siehe Global Constraints).

**Früher eine Abweichung vom Wortlaut der Spec, inzwischen in der Spec:** Die Spec in der Fassung von `5d087f7b` ließ `init` einen fremden pre-commit-Hook in den Hinweisen immer nennen; seit `ba9290cb` (Abschnitt „Scharf stellen“, Bestandsinstallationen) trägt sie dieselbe Einschränkung wie der Plan. Der Plan nennt ihn nur, wo das Projekt die Datei hat oder in diesem Lauf bekommt (Task 10, `TestAForeignPreCommitHookIsKeptAndNamed`): Ohne Datei gibt es nichts scharf zu stellen, und ein frischer Klon von loomux selbst bekäme sonst bei jedem `init` einen Hinweis zu seinem eigenen Hook.

## Task 0, 2026-10-01

Der Unterbau `docs/stage-4e-spec` ist nach einer letzten Fix-Welle per Rebase nach `master` gegangen; `master` steht auf `e6c21eed` (v6.0.0), und `feat/lane-probation` trägt darauf genau die zwei Commits dieses Zweigs (Spec, Plan).

**Anker:** Jede Zeile, jeder Name und jeder zitierte Text des Plans in einer Datei, die sich zwischen `47361ab3` und `e6c21eed` geändert hat, ist neu gelesen; die übrigen Dateien sind an beiden Ständen gleich. Berichtigt: `wordsWriteConfiguration` steht in Zeile 500 (Task 0, Step 3); Task 1 baut auf dem Text auf, den die Fix-Welle in Fusions-Spec und `migration.md` schon geschrieben hat, statt ihn ein zweites Mal anzuhängen; der Stand der Spec heißt `ba9290cb`. Kein Anker ist verschwunden, keine Signatur hat sich geändert, und keine Änderung der Welle widerspricht einer Entscheidung der Spec.

**Nachgerechnet** (git 2.54.0.vfs.0.4, Windows, in einer Kopie von `git archive HEAD` unter dem Scratchpad, nie im Arbeitsbaum): Alle Go-Blöcke der Tasks 2 bis 11 und die Änderungen an den vorhandenen Tests liegen über `e6c21eed`, jeder Anker der Überlagerung trifft genau eine Stelle. `go build ./...` und `go vet ./...` fehlerfrei. Die Batterie des Wächters am Stand vor der Regel neu aufgezeichnet: 278 Zeilen, gleich der vom 2026-09-30. Grün liefen, nacheinander und jeweils ganz: `internal/verify`, `internal/gitwork`, `internal/sessions`, `internal/setup/...`, `internal/switchover`, `internal/hooks` (mit der Batterie), `internal/cli` (darin `TestCases…` und der Weltentest mit dem echten Hook) und alle übrigen Pakete. Nicht neu gefahren: Coverage je Funktion, die Mutanten, die Läufe auf Linux und macOS; `gofmt` steht wie in Global Constraints beschrieben (die Kopie wurde vor den Läufen mit `gofmt -w` ausgerichtet).

