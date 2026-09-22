# Abweichungsliste Stufe 2c

**Quelle:** ultraloom am Tag `loomux-1a-source` (`9d01a60`): `ultraloom hook
stop`, `hook subagent-start` und `hook subagent-stop` mit `hooks/stop.py`,
`hooks/subagent_start.py`, `hooks/subagent_stop.py`, `hooks/state.py`,
`worktree.py` und der Prüfkette aus `checks.py`, die Werkzeuge beantwortet von
`faketool`.
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor Stufe 2c als
fertig gilt. „Alt" ist die Quelle oben, „Neu" ist `loomux hook stop`,
`loomux hook subagent-start` und `loomux hook subagent-stop`.
**Stand:** 2026-09-20. 15 Fälle über 15 Welten; 13 bestehen, 2 stehen als
genehmigte Abweichung in `approved2c` (`internal/cli/cases_2c_test.go`):
`hook-stop/gave-up` (Eintrag 2) und `hook-subagent-stop/no-snapshot`
(Eintrag 9).

**Selbstnutzung, über die lokale Datei.** Am 2026-09-22 liefen die drei
Sitzungshooks in dieser Sitzung selbst: der Controller hat sie in die nicht
eingecheckte `.claude/settings.local.json` eingetragen —
`hook stop --host claude --root … --budget 270s` mit Frist 300,
`hook subagent-start` und `hook subagent-stop` mit Frist 30. Seit dem
2026-09-22 stehen dieselben drei Einträge in der eingecheckten
`.claude/settings.json` (Task 16, auf Anweisung des Menschen vom Controller
geschrieben), und die lokale Datei ist wieder entfernt. Belegt ist:

- **Gehalten.** Der Controller legte einen absichtlichen `vet`-Befund an
  (`internal/verify/zz_probe_test.go` mit `fmt.Printf("%d", "x")`) und ließ
  die Runde enden. Das Tor hielt sie mit Exit 2. Sein stderr, das Claude Code
  in die Sitzung zurückgab, trug in dieser Reihenfolge: zuerst den Befund eines
  fortgesetzten Subagenten, vom Tor zugestellt —
  `subagent a65afa06307224871: branch sdd-2c moved 96cab907c6fd40cb8a3829e0727c48f03412b585 -> 69a9a7eb481e0b2700c20978d34980c44eb7d886`
  und
  `subagent a65afa06307224871: new commit 69a9a7e docs(parity): correct the struck deviation's citation and record the self-usage`
  —, danach `lint/go: failed [config] 0.9s` mit der Zeile von `go vet` für die
  angelegte Datei, `test/go: failed [config] 51.9s` (das Paket baute nicht) und
  `coverage/go: blocked [config] by test/go`. Danach stand in der
  Sitzungsdatei `{"base":"96cab90…","blocks":1}`, und die Befunddatei des
  Subagenten war weg: zugestellt und weggeräumt. Die Branch-Zeile ist die Form
  `moved`, nicht `is new at` — der Subagent hatte auf einem bestehenden Branch
  committet; die Form für einen neuen Branch hält
  `TestSubagentStopSeesALocalBranch` fest.
- **Grün.** Der Controller nahm die Datei wieder heraus und ließ die nächste
  Runde enden. Um 08:02 hatte das Tor die Kette grün gefahren und
  `{"base":"69a9a7eb…","blocks":0,"green":"5a4023a4bc187bd1a76a583b47fa250f53386ae6"}`
  gespeichert; `git rev-parse HEAD^{tree}` ist genau `5a4023a4…`. Der grüne
  Baum ist der Inhalt von HEAD, der Zähler steht wieder auf 0.
- **Der tote Reviewer.** Ein Reviewer starb an einem Rate-Limit der API.
  `hook subagent-start` hatte seinen Schnappschuss geschrieben (Remote `ok`,
  26 Refs von GitHub, 18 lokale Branches); ohne `SubagentStop` bleibt er
  liegen, bis `sessions.Forget` ihn mit der Sitzung wegräumt. **Bekannte
  Grenze:** für einen Agenten, den der Wirt selbst beendet, schickt der Wirt
  kein Stop. Die zweite Agent-Datei unter der Sitzung gehört dem fortgesetzten
  Subagenten: eine Fortsetzung löst `SubagentStart` erneut aus, und deshalb hat
  sein Befund die Fortsetzung überlebt — `subagent-start` trägt ihn weiter
  (Eintrag 19).

**Offener Punkt für den Bericht des Stop-Tores**, keine Abweichung gegen
Python: eine rote Test-Lane schreibt ihre ganze Ausgabe in den Kontext des
Agenten — in der gehaltenen Runde jede `ok <paket>`-Zeile von `go test ./...`,
rund sechzig Zeilen, wo nur die `FAIL`-Zeile zählt.

**Abweichung 16 der Spec ist falsch und entfällt.** Sie lautete: „Eine neue,
nicht ignorierte Datei ist eine Änderung. Python fragte
`git diff --name-only <base>` und sah untracked Dateien nicht; eine Runde, die
nur neue Dateien anlegte, lief ungeprüft durch." Am Quelltext nachgerechnet
stimmt das nicht. `worktree.changed_since` vereinigt den Diff mit dem Status:
`src/ultraloom/worktree.py:129` ruft `git diff --name-only -z --no-renames
<base>`, `:131` `_parse_status(_status(root))`, und `:134` gibt
`tuple(sorted(set(_relocate(root, committed + reported))))` zurück; `_status` (`:216-217`) ist
`git status --porcelain -z -uall`, also samt aller untracked Dateien. Eine
neue Datei war auf beiden Seiten eine Änderung. Der Fall
`hook-stop/untracked-only` belegt die Übereinstimmung von der Aufzeichnung wie
von der Wiedergabe her: alt Exit 2 mit `lint: faketool: uvx ruff check`, neu
ebenfalls rot; er steht nicht in `approved2c` und besteht. Die Einträge danach
sind deshalb um eins heruntergezählt — die Spec-Zeile 17 (Antigravity) ist
hier Eintrag 16. (Der Aufgabenbrief nennt für diese Stelle
`worktree.py:128-133`; gemessen sind es 129, 131 und 134.)

Die Spalte „Fall" nennt nur Fälle aus `testdata/cases/2c/`, die in `approved2c`
stehen — das sind zwei. Wo kein genehmigter Fall die Abweichung zeigt, steht
der Unit-Test, der das neue Verhalten festhält; ein bestandener Fall, dessen
aufgezeichnete Ausgabe den alten Wortlaut belegt, wird in der Begründung
genannt. Die Einträge 1 bis 15 sind die Zeilen 1 bis 15 der Spec, 16 ist deren
Zeile 17, und 17 bis 24 sind die Nachträge und Verfügungen dieses Plans.

| Nr. | Fall | Alt | Neu | Begründung | Freigabe |
|---|---|---|---|---|---|
| 1 | kein Fall — Unit-Test `TestStopHoldsAGitFailure` | `stop.py:141-146` fing jeden `WorktreeError` ab, schrieb ihn auf stderr und endete mit `EXIT_INTERNAL` (1). Exit 1 hält keine Runde — der Kommentar im selben Arm (`:142-144`) wollte das Gegenteil: „a question git could not answer must not end a turn as if the answer had been ‚clean'" | ein Git-Fehler in einem Repo ist Exit 2 und zählt: `RunStop` hält die Runde und erhöht `state.Blocks` (`internal/hooks/stop.go`, Arm `case err != nil` nach `stopTrees`) | Spec-Zeile 1. Der Kommentar der Quelle nennt die Regel, der Code führte sie nicht aus; ein Tor, das bei einer unbeantworteten Frage durchlässt, ist an genau der Stelle blind, an der es messen sollte. Kein Quellfall zeigt es: der Rekorder kann git im Fall nicht scheitern lassen | |
| 2 | `hook-stop/gave-up` | Exit 0, auf stderr: „gave up after 3 blocks in this session; the chain was still red the last time it ran. Run `ultraloom check all` by hand, or remove .claude/.no-verify once the gate should count again." Der Zähler zählte je Sitzung und blieb stehen — `_advance` ließ ihn ausdrücklich unberührt („The counter is deliberately *not* cleared"), also war das Tor nach drei roten Rundenenden für den Rest der Sitzung aus | der Zähler zählt Blockaden **in Folge**: ein grüner Lauf setzt ihn auf 0, und der Aufgeben-Arm setzt ihn ebenfalls auf 0 und meldet `gave up after 3 consecutive blocks; base stays at <kurz-sha>. Fix the lanes or set .loomux/no-verify.`, Exit 0 | Spec-Zeile 2. Eine Sitzung, die zwischendurch grün war, ist keine Sitzung in der Schleife; die Obergrenze soll einen Streit zwischen Agent und Tor beenden, nicht das Tor. Unit-Tests `TestStopGivesUpAfterThreeInARow`, `TestStopResetsTheCounterWhenGreen`. Der Fall weicht in `blocks` ab (Klasse `state`), nicht im Exit | |
| 3 | kein Fall — Unit-Tests `TestDefaultsHoldAStopProfile`, `TestAStopProfileCanBeNarrowed`, `TestStopPassesAndMovesTheBase` | `--checks <profil|liste>` schnitt zu, was das Tor fuhr; ein bestandener Lauf unter `--checks` rückte die Basis **nie** vor (`_advance` lief nur bei `checks is None`) | `hook stop` kennt kein `--checks`: die Unterbefehle haben `--host`, `--root` und — nur `stop` und `post-edit` — `--budget` (`internal/cli/hook.go:47-53`), alles andere ist ein Usage-Fehler. Gefahren wird das eingebaute Profil `stop` = `lint, types, test, coverage` (`internal/verify/schema.go:83`), überschreibbar über `[verify.profiles] stop`, und jeder grüne Lauf rückt `base` und `green` vor | Spec-Zeile 3. Die Basis ist das Wort des Tores für „bis hierher geprüft"; wenn ein Zuschnitt sie nie bewegt, wächst die Spanne ewig, und wer zuschneidet, tut es genau deshalb. Mit dem Profil ist der Zuschnitt Sache der Konfiguration, und was das Profil prüft, darf die Basis auch bewegen | |
| 4 | kein Fall — Unit-Tests `TestStopSkipsAGreenTree`, `TestStopSkipsATreeEqualToTheBase` | die Abkürzung fragte `changed_since` bzw. `changed_files` nach Pfaden; eine Runde, die nur las und antwortete, war frei, aber jede Runde, die irgendeine Datei berührte, fuhr die ganze Kette | verglichen wird ein Fingerabdruck des Inhalts: `gitwork.ContentTree` gegen `state.Green` (der Baum des letzten grünen Laufs) und gegen `base^{tree}`. Sind sie gleich, startet kein einziges Werkzeug | Spec-Zeile 4. Ein Inhalt, der schon grün war, ist wieder grün, gleich wie oft er geschrieben wurde; ein Rundenende ohne neuen Inhalt kostet 121 ms in einer kleinen Welt und 176 ms in diesem Arbeitsbaum mit 7.341 Dateien, davon rund 110 ms der Fingerabdruck selbst (`docs/de/benchmarks.md`), gegen die Sekunden einer ganzen Kette. Der Kopiervorgang des Index geht nach `os.TempDir()`, damit ein unschreibbares Zustandsverzeichnis kein Git-Fehler wird, der jede Runde hält | |
| 5 | kein Fall — Unit-Test `TestStopOutsideARepositoryRunsTheChain` | `changed_since`/`changed_files` warfen außerhalb eines Repos `WorktreeError`; die Runde endete über den Arm aus Eintrag 1 mit Exit 1, ohne dass irgendetwas geprüft wurde | kein Repo ist kein Fehler: `stopTrees` gibt `errNoRepository` zurück, es gibt keinen Fingerabdruck, und die Kette läuft jedes Mal. Ein grüner Lauf schreibt `base` und `green` leer und wiederholt sich beim nächsten Mal | Spec-Zeile 5. Ein Projekt ohne Git soll geprüft werden, nur eben ohne Abkürzung; die Alternative wäre ein Tor, das genau dort schweigt, wo es keine Historie zum Nachsehen gibt | |
| 6 | kein Fall — Unit-Tests `TestStopInAnUnbornRepository`, `TestStopMeasuresFromTheEmptyTreeWhenThereIsNoHead` | ein Repo ohne Commit: `head_commit` warf, `_advance` schluckte es (`except WorktreeError: return`), und die Basis blieb still stehen, was sie war | ohne HEAD ist der leere Baum (`gitwork.EmptyTree`) die Basis; gemessen wird alles, was im Arbeitsbaum steht. Weder „no base commit" noch „base … is gone" wird gesagt, denn beides träfe nicht zu | Spec-Zeile 6. Ein Repo ohne Commit hat keine Vorgeschichte, also ist alles neu; still die alte Basis zu behalten hieße, eine Messung vorzutäuschen | |
| 7 | kein Fall — Unit-Test `TestStopReportsABudgetThatRanOut` | es gab kein Budget: `timeout` galt je Befehl (600 s), und die Kette konnte länger laufen, als der Wirt auf den Hook wartete | ist das Budget (`--budget`, Vorgabe `DefaultStopBudget` = 270 s unter der Frist von 300 s) aufgebraucht, sind die offenen Lanes `budget`, und `stopVerdict` meldet `not everything was verified; raise --budget or shrink the stop profile` mit Exit 1: kein Urteil, keine Blockade, keine neue Basis | Spec-Zeile 7. Ein Tor, das über seine Frist läuft, wird vom Wirt abgeschnitten und hat dann weder geurteilt noch es gesagt. Exit 1 hält nichts, sagt aber, dass nichts geprüft wurde | |
| 8 | kein Fall — Unit-Tests `TestStopDeliversFindingsAndHolds`, `TestStopDeliversFindingsEvenWithTheMarker` | `subagent-stop` schrieb seine Zeilen auf **sein eigenes** stdout, etwa `subagent a1: new commit e635fe8 second` (`hook-subagent-stop/new-commit`, stdout). Das stdout eines Subagenten erreicht den Hauptagenten nicht | `subagent-stop` legt die Zeilen in `.loomux/state/hooks/<sitzung>/agents/<agent>.json` ab; `stop` stellt sie als Erstes zu — vor dem Marker, vor dem Zähler — mit dem Präfix `subagent <id>: ` und hält die Runde dafür an | Spec-Zeile 8, dazu Nachtrag 7: die Datei trägt die Zeilen **ohne** Präfix, `stop` setzt es beim Zustellen, die Vergleichsklasse `finding` beim Vergleich. Was ein Subagent an `origin` getan hat, muss der lesen, der ihn gestartet hat. Vor dem Marker, weil der Marker eine Aussage über das Prüfen ist und nicht über das Berichten | |
| 9 | `hook-subagent-stop/no-snapshot` | Exit 0 und auf stdout `subagent a1: no snapshot for this subagent; nothing to compare` — eine Zeile in einem Strom, den niemand liest (siehe Eintrag 8) | ohne Agent-Datei oder ohne Schnappschuss darin schweigt der Hook und endet mit 0; die Datei bleibt liegen, falls sie eine geparkte Erkenntnis trägt | Spec-Zeile 9. Die Zeile kostete einen Vergleich und erreichte niemanden; wo sie hinmüsste (die Agent-Datei), stünde sie als Befund über einen Subagenten, über den nichts bekannt ist. Unit-Tests `TestSubagentStopWithoutSnapshotIsSilent`, `TestSubagentStopWithAFindingAndNoSnapshotIsSilent`. Der Fall weicht in den `finding`-Zeilen ab, nicht im Exit | |
| 10 | kein Fall — Unit-Tests `TestSubagentStopWithARemoteMissingAtStart`, `TestSubagentStopWithARemoteGoneAtStop` | `remote_refs` gab bei jedem Fehlschlag — kein Remote, kein Netz, Zeitüberlauf, kein git — den leeren String zurück. Ein Schnappschuss mit Refs gegen eine leere Antwort meldete danach **jede** Ref als verschwunden, und umgekehrt jede als neu | der Schnappschuss merkt sich, **ob** der Remote geantwortet hat (`sessions.RemoteOK` / `RemoteUnavailable`). War er an einem der beiden Enden nicht lesbar, steht genau eine Zeile da — `remote could not be read at start` bzw. `… at stop` — und kein Vergleich der Refs von `origin`; die Zeilen der lokalen Branches und der neuen Commits kommen trotzdem | Spec-Zeile 10. Ein unerreichbarer Remote ist eine Tatsache über die Maschine, keine über den Subagenten; die alte Form machte aus einem WLAN-Aussetzer einen Bericht, der jede Ref des Remotes als gelöscht meldete | |
| 11 | kein Fall — Unit-Tests `TestSubagentStopSeesALocalBranch`, `TestSubagentStopWithATranslatedSnapshot` | der Schnappschuss kannte nur `ls-remote origin` und den lokalen HEAD (`_HEAD_MARKER`). Ein Subagent, der einen lokalen Branch anlegte, verschob oder löschte, ohne zu pushen, blieb unsichtbar | der Schnappschuss trägt zusätzlich `Heads`, die lokalen Branches aus `gitwork.LocalBranches`; verglichen wird mit dem Präfix `branch ` und ohne `refs/heads/`, und ein bewegter Branch liefert außerdem seine neuen Commits | Spec-Zeile 11, dazu Nachtrag 3: ein aus Python übersetzter Schnappschuss hat `heads: null` und vergleicht keine Branches; `takeSnapshot` schreibt nie `null`, sondern mindestens `{}`. Das ist der Unterschied zwischen „hatte keine Branches" und „wusste nichts von Branches". Seit Nachtrag 24 fällt ein Branch, der am Start oder am Stop in einem anderen Worktree ausgecheckt ist, ganz aus dem Vergleich (Feld `elsewhere`, Unit-Tests `TestSubagentStopLeavesOutTheBranchOfAnotherWorktree` und Geschwister) | |
| 12 | kein Fall — Unit-Tests `TestSubagentStartRecordsTheSnapshot`, `TestSubagentStopWithNothingChangedRemovesTheFile` | alle Schnappschüsse standen als `snapshots`-Abbildung in **einer** Sitzungsdatei; zwei Subagenten, die gleichzeitig starteten, schrieben dieselbe Datei, und ein Schnappschuss blieb bis zum Ende der Sitzung liegen | eine Datei je Agent unter `.loomux/state/hooks/<sitzung>/agents/<agent>.json`, atomar geschrieben; sie wird nach dem Vergleich entfernt, sobald nichts mehr darin steht, und `sessions.Forget` nimmt beim Sitzungsende das ganze Verzeichnis mit | Spec-Zeile 12. Zwei Prozesse, die eine Datei überschreiben, verlieren einander; zwei Prozesse mit je einer Datei nicht. Die Übersetzung der alten Fälle faltet `snapshots` in dieses Layout (`dev import-cases`) | |
| 13 | kein Fall — Unit-Test `TestStopMarkerLetsTheTurnEnd` | der Marker war `.claude/.no-verify` — ein Pfad im Verzeichnis des Wirts, den ein Agent selbst anlegen konnte | der Marker ist `.loomux/no-verify`; die Schreibsperre (`internal/hooks/guard.go`) hält ihn außerhalb der Reichweite eines Agenten | Spec-Zeile 13. Ein Tor, dessen Aus-Schalter der Geprüfte umlegen kann, ist keines; außerdem gehört `.claude/` dem Wirt und nicht loomux. Die Übersetzung der Fälle benennt den Marker mit um (`testdata/cases/README.md`, Zeile `2c/`) | |
| 14 | kein Fall — Unit-Tests `TestWikiGateJobsRunTheGateInProcess`, `TestWikiGateJobsOnlyForLintAndAWiki` | das Wiki-Tor war ein eigener Stop-Eintrag neben der Kette, mit eigenem Urteil und eigenem Zähler | `WikiGateJobs` hängt eine Lane `lint/wiki` (Art `lint`, Stack `wiki`, in diesem Prozess) an die geplanten Jobs — in `loomux check` wie im `stop`-Tor. Ein Lauf, ein Urteil, ein Zähler | Spec-Zeile 14, dazu Nachtrag 5: die Wiki-Lane steht **am Ende**, weil `After` Indizes sind und ein Einfügen sie verschöbe; die Reihenfolgeregel der 2a-Spec („Arten in Anfrage-Reihenfolge") gilt für `lint/wiki` deshalb nicht | |
| 15 | kein Fall — Unit-Tests `TestStopShowsOnlyRedLanes`, `TestStopPassesAndMovesTheBase` | eine Zeile je roter Art in der Form `<art>: <ausgabe>`, etwa `lint: faketool: uvx ruff check` gefolgt von `tests/test_a.py:1:1: F401 unused import` (`hook-stop/red`, stderr), dazu die Sätze `gave up after …`, `no base commit for this session, so only the working tree is measured -- anything this session committed is invisible here. Start a session with the session-start hook wired up to close that gap.` (`hook-stop/no-base`, stderr) und `the chain could not run; nothing was verified` | Lane-Zeilen aus `verify.WriteCheck`, und nur die roten: `<art>/<stack>` mit der Ausgabe darunter. Die Meldungstexte sind neu gefasst (`no base commit for this session; measuring from HEAD, so what this session committed stays unseen`, `nothing was verified for these kinds; the base stays`, `gave up after 3 consecutive blocks; base stays at <kurz-sha>. …`) | Spec-Zeile 15. Was in den Kontext des Agenten läuft, ist das, was er beheben soll — grüne Lanes sind dort Rauschen. Die Fallsuite vergleicht deshalb keine Meldungstexte: die Klasse `state` prüft Exit, `base` und `blocks`, die Klasse `finding` Exit und Befundzeilen. Der alte Wortlaut steht in den aufgenommenen `stderr`-Dateien | |
| 16 | kein Fall, kein Unit-Test — offen | — | — | Spec-Zeile 17 („Was die Antigravity-Messung ergibt"). **Offen:** Tasks 14 und 15 sind Halte für den Menschen und am 2026-09-20 nicht gelaufen; in `internal/hooks` gibt es bis dahin weder einen Antigravity-Adapter noch das `ErrNoAdapter`, das die Spec für das vorsieht, was die Messung nicht belegt. Die Zeile bleibt stehen und wird gefüllt, wenn gemessen ist | |
| 17 | kein Fall — Unit-Tests `TestWikiGateJobsPassWhenOnlyCodeChanged`, `TestLintBundleReportsALintThatFails` | `loomux wiki-gate` (und davor ultralooms Wiki-Tor) prüft zweierlei: die Struktur des Bündels (`wiki-lint:*`) **und** die Drift-Regel — Code geändert, Doku nicht | die Lane `lint/wiki` prüft **nur das Bündel** (`wiki.LintBundle`, `Severity == check.Error`, dieselben Befunde, die `GateReport` als `wiki-lint:*` meldet). Die Drift-Regel läuft dort nicht; `loomux wiki-gate` behält sie für den, der sie will | Verfügung zu Task 9. Ein Tor, das jeden reinen Code-Commit abweist, ist kein Tor: es hielte jedes Rundenende von loomux selbst. Drift ist ein Urteil darüber, ob Doku den Code begleitet, und ultraloom hat es nur dort eingebaut, wo `[wiki] mode = "brain"` das sagte. **Kosten, wenn falsch:** Drift wird nirgends mehr automatisch erzwungen und bleibt ein Befehl | |
| 18 | kein Fall — Unit-Tests `TestStopReportsAFindingFileItCannotRemove`, `TestStopGivesUpOnAFindingItCannotClearAway` | die Frage stellte sich nicht: Befunde gingen über stdout (Eintrag 8) und hinterließen nichts, was liegenbleiben könnte | ein Befund, dessen Datei nach dem Zustellen nicht wegzuräumen ist, **zählt als Blockade**: `state.Blocks` steigt, und die Aufgeben-Regel aus Eintrag 2 begrenzt die Reihe: drei Runden angehalten, die vierte durchgelassen, mit dem Befund geschrieben und weiter auf der Platte (Nachtrag 23). Ein Halt allein aus Befunden zählt sonst nichts | Verfügung zu Task 10. Sonst käme derselbe Befund an jedem Rundenende wieder, hielte jedes Mal, und nichts begrenzte die Schleife — auch der Marker nicht, denn er überspringt keine Befunde. Verworfen wurde: gar nicht halten, denn dann liefe der Befund auf ein stderr, das bei Exit 0 niemand liest. **Kosten:** nach drei solchen Runden lässt das Tor eine Runde durch, während der Befund noch auf der Platte liegt — der Mensch sieht das in der Meldung | |
| 19 | kein Fall — Unit-Tests `TestSubagentStartKeepsAParkedFinding`, `TestSubagentStopAddsToACarriedFinding`, `TestStopKeepsAFindingThatArrivesWhileItDelivers`, `TestStopClearsAFindingFileThatShrankWhileItDelivered` | der Schnappschuss eines Agenten wurde bei jedem `subagent-start` überschrieben; ein Bericht, den niemand gelesen hatte, war danach weg | ein Befund wird nie fallengelassen, nur ergänzt. (a) `subagent-start` schreibt den neuen Schnappschuss **neben** den geparkten Befund. (b) `subagent-stop` hängt seine Zeilen hinten an, ältester Lauf zuerst, ohne Entdoppelung. (c) `stop` räumt genau die Zeilen weg, die es zugestellt hat, entschieden gegen ein erneutes Lesen der Datei unmittelbar vor dem Schreiben, und lässt eine Datei stehen, die noch einen Schnappschuss trägt | Verfügungen zu Task 11. Ein Controller, der einen Subagenten unter derselben Kennung fortsetzt — diese Sitzung tut das —, darf nicht löschen, was dessen letzter Lauf an `origin` getan hat; dieselbe Zeile zweimal ist Geschichte, nicht Rauschen. **Offen und benannt:** ein Schreibvorgang, der genau zwischen das erneute Lesen und das Umbenennen fällt, verliert eine Zeile. Eine Sperre über eine Datei, die zwei Prozesse anfassen, baut diese Stufe nicht | |
| 20 | kein Fall — Unit-Tests `TestStopMeasuresFromHeadWhenTheBaseIsGone`, `TestStopMeasuresFromTheTreeHeadHoldsWhenTheBaseIsGone` | die Frage stellte sich anders: `changed_since` warf bei einer Basis, die nicht mehr auflöst, `WorktreeError`, und das lief in den Arm aus Eintrag 1 — Exit 1, nichts geprüft | eine Basis, die nach `--amend`, Rebase und `gc` nicht mehr auflöst, ist kein Git-Fehler, der hält: `stopTrees` meldet `base <kurz-sha> is gone; measuring from HEAD`, misst gegen HEADs Baum, und der nächste grüne Lauf setzt die Basis neu | Nachtrag 8. Sonst hielte das Tor drei Runden je Nutzerrunde mit einem Hinweis auf Lanes, die damit nichts zu tun haben, und die Reihe endete erst über die Aufgeben-Regel. Mit HEAD als Basis ist die Messung eng und stimmt wieder, sobald einmal grün war | |
| 21 | kein Fall — Unit-Test `TestLsRemoteReadsTheRemote` (`internal/gitwork`) | `subagent_stop.py` rief `git ls-remote origin` über `process.run` mit der Umgebung, die der Hook geerbt hatte, und einer Frist von 10 s | `gitwork.LsRemote` setzt genau eine Variable: `GIT_TERMINAL_PROMPT=0` (`internal/gitwork/remote.go:29`), Frist weiterhin `RemoteTimeout` = 10 s. Ein `GIT_SSH_COMMAND` wird **nicht** gesetzt | Nachtrag 1. `GIT_SSH_COMMAND` schlüge `core.sshCommand` und die SSH-Einstellung des Nutzers (etwa einen Agenten über ein eigenes `ssh`); ein SSH ohne TTY fragt ohnehin nicht, sondern scheitert, und die Frist fängt den Rest. `gitenv` streicht `GIT_TERMINAL_PROMPT` nicht (`internal/gitenv/gitenv.go:43`) | |
| 22 | kein Fall — Unit-Test `TestHeadRefusesAnIgnoredRoot` (`internal/gitwork`); auf der Stop-Seite hält kein Test das ignorierte Wurzelverzeichnis fest, nur `TestStopOutsideARepositoryRunsTheChain` den Zwilling „kein Repo" | `_refuse_if_ignored(root)` warf für ein ignoriertes Wurzelverzeichnis `WorktreeError` — wieder der Arm aus Eintrag 1, Exit 1 | ein ignoriertes Wurzelverzeichnis (`gitwork.ErrIgnoredRoot`) zählt für `stop` wie „kein Repo": `stopTrees` gibt `errNoRepository` zurück, es gibt keinen Fingerabdruck, und die Kette läuft jedes Mal | Nachtrag 4. Beide Fälle bedeuten dasselbe — es gibt keinen Baum, gegen den sich messen ließe —, und beide dürfen nicht dazu führen, dass gar nicht geprüft wird. **Bekannt und offen:** dadurch sagt das Tor zu einem ignorierten Wurzelverzeichnis gar nichts; der Text von `ErrIgnoredRoot` ist hier unerreichbar (Nachtrag zu Task 10) | |
| 23 | kein Fall — Unit-Test `TestStopReportsNothingVerified` | „alle roten sind `UNAVAILABLE`" ergab `the chain could not run; nothing was verified` mit Exit 1 — dieselbe Absicht, aber über den Zustand der roten Lanes | eine **angefragte Art ohne Lane, die lief**, ist kein Grün: `verify.CheckVerdict` liefert Notizen, `stopVerdict` schreibt sie und danach `nothing was verified for these kinds; the base stays`, Exit 1, keine neue Basis | Nachtrag 6. Pythons Regel war „alle roten sind unavailable"; loomux fragt stattdessen, was von den angefragten Arten überhaupt ein Urteil hat — das ist dieselbe Aussage, aber unabhängig davon, ob es rote Lanes gab. Ein echter Befund neben einer Art ohne Lane bleibt rot, weil `stopVerdict` die roten Lanes vor den Notizen liest | |
| 24 | kein Fall — Unit-Test `TestStopHoldsALoadErrorWhenItDeliveredFindings` | die Frage stellte sich nicht: Befunde gingen über stdout (Eintrag 8), und ein Ladefehler endete schlicht mit 1 | wurden im selben Aufruf Befunde zugestellt, wird auch Exit 1 zu Exit 2: ein Ladefehler der Konfiguration, ein Planfehler, ein aufgebrauchtes Budget halten die Runde dann an (`end(code)` in `RunStop`), gleich was der Zähler sagt und auch mit Marker. Ohne Befunde bleibt es bei 1. Das zählt **nicht** als Blockade. Die Runde, in der der Zähler aufgibt, endet mit 0 und lässt die Befunde deshalb liegen; das nächste Rundenende stellt sie zu (Nachtrag 23, Unit-Test `TestStopKeepsFindingsThroughTheGiveUpTurn`) | Nachtrag 9. Die Befunddateien sind nach dem Zustellen weg; nur ein Halt sorgt dafür, dass der Hauptagent sie in seinem Kontext liest. Ein Fehler des Tores ist keine Aussage über die Arbeit, darum zählt er nicht — die Zeile des Subagenten muss trotzdem ankommen | |
| 25 | kein Fall — Unit-Test `TestHookSessionStartKeepsABaseTheSessionAlreadyHas` | die Basis wurde bei **jedem** `SessionStart` auf HEAD gesetzt. Am Tag `loomux-1a-source` gibt es kein `session_start.py` mehr (entfernt in `6a7037a`); der Hook ist dort Go, `cmd/guard/hook_session_start.go`, und `recordBase` liest den Zustand und schreibt `state.Base = commit` ohne Bedingung. Die letzte Python-Form (`session_start.py` in `fa3dd38`, `_record_base`) tat dasselbe: `write_state(root, session_id, replace(state, base=commit))` | `recordBase` (`internal/hooks/hook_session_start.go`) schreibt `base` nur, wenn die Sitzung noch keine hat; eine fortgesetzte, geleerte oder kompaktierte Sitzung behält ihre Basis, und nur ein grüner Lauf rückt sie vor | Nachtrag 22. `SessionStart` feuert ohne Matcher auch bei `resume`, `clear` und `compact` unter derselben `session_id`. Ein roter Commit, danach eine Kompaktierung, und die Basis stand auf diesem Commit: der nächste `stop` fand den Baum gleich dem der Basis und ließ die Runde enden, ohne eine Lane zu fahren. Die Quelle hatte dieselbe Lücke: `stop.py` fragte `changed_since(root, base)` und übersprang die Kette, wenn nichts seit der mitgewanderten Basis geändert war | |

## Was die Fälle der Stufe 2c decken

`testdata/cases/2c-source/` hält die Aufzeichnungen der drei Sitzungshooks von
ultraloom (Beweis, nie nachbearbeitet), `testdata/cases/2c/` die Übersetzung,
an der loomux gemessen wird: `ultraloom hook <verb>` wird
`loomux hook <verb> --host claude` (`2c-map.toml`), `.ultraloom/hooks/<id>.json`
wird `.loomux/state/hooks/<id>.json` mit `base` und `blocks`, jeder Eintrag aus
`snapshots` wird eine Agent-Datei mit geparsten Refs, und `.claude/.no-verify`
wird `.loomux/no-verify`. 15 Fälle über 15 Welten (acht `stop-*`, zwei
`subagent-start-*`, fünf `subagent-stop-*`), gefahren von
`internal/cli/cases_2c_test.go`; `wantCases2c` ist auf 15 festgenagelt, damit
ein unvollständiger Import nicht als Parität durchgeht.

Die Klasse `state` vergleicht für `stop` den Exit-Code und `base` und `blocks`
der Sitzungsdatei, die Klasse `finding` für `subagent-stop` den Exit-Code und
die Befundzeilen aller Agent-Dateien, die Klasse `message` nur den Exit-Code.
Meldungstexte werden nirgends verglichen; die aufgenommenen `stderr`-Dateien
belegen den alten Wortlaut in der Liste oben. Ein Fall in `approved2c` muss
scheitern, jeder andere bestehen; ein genehmigter Fall, der besteht, lässt die
Suite rot werden — genau daran ist die gestrichene Spec-Abweichung 16
aufgefallen. Angepasst wurde keine Aufzeichnung.

Zwei Dinge, die die Fälle **nicht** zeigen, weil beide Seiten sie gleich tun:
ein bewegter Standard-Branch meldet zwei Zeilen (`origin HEAD moved …` und
`origin refs/heads/master moved …`), weil `ls-remote` die `HEAD`-Zeile
mitliefert und Python sie mitverglich (Nachtrag 2, Fälle
`hook-subagent-stop/ref-moved`, `-new`, `-gone`); und eine neue untracked Datei
ist auf beiden Seiten eine Änderung (`hook-stop/untracked-only`, oben).

Kein aufgezeichneter Fall hält fest, was `subagent-start` schreibt:
`hook-subagent-start/records` vergleicht in der Klasse `state` nur `base` und
`blocks` der Sitzungsdateien, und `no-agent` in der Klasse `message` nur den
Exit-Code. Den Schnappschuss in der Agent-Datei hält allein der Unit-Test
`TestSubagentStartRecordsTheSnapshot` fest.

## Absichten der Python-Tests

Je Test am Tag `loomux-1a-source` eine Zeile: der Go-Test, der dieselbe Absicht
prüft, oder die Nummer der Abweichung, die erklärt, warum es ihn nicht gibt.
Gezählt aus der Quelle: `tests/hooks/test_stop.py` 26,
`tests/hooks/test_subagent_start.py` 4, `tests/hooks/test_subagent_stop.py` 16
— zusammen 46, wie der Plan es sagt. Go-Tests ohne Paketangabe stehen in
`internal/hooks`.

### `test_stop.py` (26)

| Python-Test | Absicht getragen von |
|---|---|
| `test_the_marker_switches_the_gate_off` | `TestStopMarkerLetsTheTurnEnd`; der Pfad ist Abweichung 13 |
| `test_a_project_without_a_claude_directory_does_not_raise` | `TestStopPassesAndMovesTheBase` und jeder andere Lauf ohne Marker: `os.Stat` auf einem fehlenden Pfad ist kein Fehler, der irgendwohin durchschlägt |
| `test_a_session_that_changed_nothing_is_not_checked` | `TestStopSkipsAGreenTree`, `TestStopSkipsATreeEqualToTheBase`; gemessen wird jetzt der Inhalt (Abweichung 4) |
| `test_the_counter_stops_blocking_after_three_rounds` | `TestStopGivesUpAfterThreeInARow`; die Zählweise ist Abweichung 2 |
| `test_a_green_chain_lets_the_turn_end` | `TestStopPassesAndMovesTheBase` |
| `test_a_red_chain_blocks_and_names_every_finding` | `TestStopHoldsARedChainAndCounts`; „jeden Befund nennen" ist auf die roten Lanes eingeschränkt, Abweichung 15 (`TestStopShowsOnlyRedLanes`) |
| `test_a_green_turn_does_not_clear_the_counter` | Abweichung 2 — loomux tut das Gegenteil, festgehalten von `TestStopResetsTheCounterWhenGreen` |
| `test_a_chain_that_could_not_run_is_not_a_block` | `TestStopReportsNothingVerified`; die Regel ist neu gefasst, Abweichung 23 |
| `test_a_chain_that_refuses_to_be_scheduled_is_an_internal_error` | `TestStopReportsAPlanItCannotMake` |
| `test_a_broken_config_is_an_internal_error` | `TestStopReportsABadConfig` |
| `test_git_refusing_to_answer_is_never_read_as_nothing_changed` | `TestStopHoldsAGitFailure`; jetzt Exit 2 statt 1, Abweichung 1 |
| `test_an_unreadable_payload_is_an_internal_error` | `TestStopRefusesAPayloadThatIsNoJSON`, dazu der Fall `hook-stop/bad-payload` |
| `test_a_payload_without_a_session_id_is_an_internal_error` | `TestStopRefusesAPayloadWithoutSession` |
| `test_a_red_turn_leaves_the_next_one_with_the_same_question` | `TestStopHoldsARedChainAndCounts`: die Basis bleibt, wo sie war |
| `test_a_committed_change_is_still_visible_to_the_gate` | `TestStopPassesAndMovesTheBase` und `TestContentTreeSeesChangesAndNewFilesButNotState` (`internal/gitwork`): gemessen wird der Inhalt gegen `base^{tree}`, also versteckt ein Commit nichts |
| `test_a_green_turn_moves_the_base_so_the_next_one_is_free` | `TestStopPassesAndMovesTheBase`, `TestStopSkipsAGreenTree` |
| `test_without_a_base_the_blind_spot_is_said_out_loud` | `TestStopWithoutBaseMeasuresFromHead`; der Wortlaut ist neu, Abweichung 15 |
| `test_a_green_run_where_the_base_cannot_be_moved_still_ends_the_turn` | `TestStopOutsideARepositoryRunsTheChain` (Abweichung 5) und `TestStopReportsAStateItCannotWrite`: ein Buchhaltungsfehler hält keine Runde |
| `test_a_filesystem_that_refuses_the_marker_leaves_the_gate_on` | **kein Go-Test.** Die Regel liegt in `os.Stat(filepath.Join(root, NoVerifyMarker)); err == nil` (`stop.go`): jeder Fehler lässt das Tor an, wie Pythons `except OSError: return False`. Kein Test hält es fest — Lücke, keine Abweichung |
| `test_a_real_finding_blocks_even_beside_a_check_that_cannot_run` | **kein eigener Go-Test.** `stopVerdict` liest die roten Lanes vor den Notizen, ein echter Befund kann also nicht von einer Art ohne Lane geschluckt werden (Abweichung 23). Lücke |
| `test_a_profile_narrows_what_the_gate_runs` | Abweichung 3 — `--checks` entfällt; das Profil `stop` tritt an seine Stelle (`TestDefaultsHoldAStopProfile`) |
| `test_a_comma_separated_list_narrows_it_too` | Abweichung 3 |
| `test_without_the_argument_every_kind_still_runs` | Abweichung 3 — ohne Konfiguration fährt das eingebaute Profil alle vier Arten (`TestDefaultsHoldAStopProfile`) |
| `test_an_unknown_profile_never_holds_the_turn` | Abweichung 3 — ein Profil lässt sich nur noch in `[verify.profiles]` nennen, und `ExpandProfile` hat `stop` eingebaut; ein unbekannter Name ist damit kein Zustand, den der Hook erreicht |
| `test_a_narrowed_pass_leaves_the_base_where_it_was` | Abweichung 3 — die Basis rückt jetzt nach jedem grünen Lauf des Profils vor (`TestAStopProfileCanBeNarrowed`, `TestStopPassesAndMovesTheBase`) |
| `test_a_tool_that_is_not_installed_holds_the_turn` | `TestRedDependsOnTheScope` (`internal/verify`): `StateMissingTool` ist im Prüfumfang rot, und `stop` fährt `ScopeCheck` |

### `test_subagent_start.py` (4)

| Python-Test | Absicht getragen von |
|---|---|
| `test_the_snapshot_lands_under_the_agent_id` | `TestSubagentStartRecordsTheSnapshot`; das Layout ist Abweichung 12 |
| `test_a_second_subagent_does_not_overwrite_the_first` | Abweichung 12 — je Agent eine eigene Datei, also gibt es nichts zu überschreiben; `TestSubagentStartKeepsAParkedFinding` hält dazu fest, dass ein zweiter Start desselben Agenten den geparkten Befund behält (Abweichung 19) |
| `test_a_payload_without_an_agent_id_is_an_internal_error` | `TestSubagentStartRefusesWithoutAgent`, dazu `TestSubagentStartRefusesWithoutASession` und der Fall `hook-subagent-start/no-agent` |
| `test_an_unreadable_payload_is_an_internal_error` | **kein Go-Test, keine Abweichung.** `subagentPayload` meldet den Host-Fehler und den JSON-Fehler über denselben Arm, und kein Test füttert eine kaputte Nutzlast; `TestSubagentHooksWithAnUnknownHost` erreicht den Arm über den anderen Weg. Lücke, in `progress.md` als Minor von Task 11 notiert |

### `test_subagent_stop.py` (16)

| Python-Test | Absicht getragen von |
|---|---|
| `test_no_snapshot_says_so_rather_than_claiming_nothing_happened` | Abweichung 9 (Fall `hook-subagent-stop/no-snapshot`); `TestSubagentStopWithoutSnapshotIsSilent` |
| `test_an_unchanged_remote_is_reported_as_nothing` | `TestSubagentStopWithNothingChangedRemovesTheFile` |
| `test_a_push_is_reported` | `TestSubagentStopSeesAPush`, dazu der Fall `hook-subagent-stop/ref-moved` |
| `test_a_commit_that_stayed_local_is_reported_too` | `TestSubagentStopSeesACommitOffEveryBranch`, `TestSubagentStopCountsACommitOnce` |
| `test_a_snapshot_without_a_head_line_compares_the_remote_only` | `TestSubagentStopWithATranslatedSnapshot`: ein übersetzter Schnappschuss ohne `head` und mit `heads: null` vergleicht nur den Remote (Abweichung 11, Nachtrag 3) |
| `test_an_unmoved_head_is_reported_as_nothing` | `TestSubagentStopWithNothingChangedRemovesTheFile` |
| `test_differences_names_both_sides` | `TestSubagentStopSeesANewAndAGoneRef` — `refLines` nennt beide Richtungen |
| `test_a_line_that_is_no_ref_at_all_is_passed_over` | `TestLsRemoteReadsTheRemote` (`internal/gitwork`): `parseRefs` überspringt jede Zeile ohne Tabulator |
| `test_differences_notices_a_ref_that_disappeared` | `TestSubagentStopSeesANewAndAGoneRef`, dazu der Fall `hook-subagent-stop/ref-gone` |
| `test_a_repository_without_a_remote_is_not_an_error` | `TestLsRemoteWithoutRemote` (`internal/gitwork`) und `TestSubagentStopWithARemoteMissingAtStart`; was daraus folgt, ist Abweichung 10 |
| `test_remote_refs_is_empty_when_git_cannot_be_run` | `TestLsRemoteReportsAStartThatFailed` (`internal/gitwork`); der leere String wird zu `RemoteUnavailable`, Abweichung 10 |
| `test_remote_refs_is_empty_when_the_remote_does_not_answer` | `TestLsRemoteGivesUpAtItsDeadline` (`internal/gitwork`) und `TestSubagentStopWithARemoteGoneAtStop`, Abweichung 10 |
| `test_head_is_empty_outside_a_repository` | `TestHeadOutsideARepository` (`internal/gitwork`) |
| `test_new_commits_is_empty_when_the_range_makes_no_sense` | `TestLocalHeadsAndLogOutsideARepository` (`internal/gitwork`) und `TestSubagentStopLogsNothingForABranchThatIsGone` |
| `test_an_unreadable_payload_is_an_internal_error` | **kein Go-Test, keine Abweichung** — dieselbe Lücke wie bei `subagent-start`, beide Hooks gehen durch `subagentPayload` |
| `test_a_payload_without_an_agent_id_is_an_internal_error` | `TestSubagentStartRefusesWithoutAgent` über das gemeinsame `subagentPayload`; einen eigenen Test auf der Stop-Seite gibt es nicht |

## Mutationsrunde

Gelaufen am 2026-09-20 mit `go run ./cmd/loomux dev mutants` (Task 18), gegen
Commit `e130c2d` auf `sdd-2c`, 8 Arbeiter (die Voreinstellung
`DefaultWorkers()` = halbe Kernzahl, 16 Kerne), Go 1.27.0 unter Windows.
`dev mutants` kennt `-family`, `-only` und `-workers`; ein `--files` gibt es
nicht, der Aufgabenbrief riet daneben. Ein Zeitüberlauf zählt als getötet.
Protokolle unter `$TEMP/…/scratchpad/round-*.txt` und `confirm-*.txt`.

### Schätzung und Zuschnitt

Schritt 1, gemessen warm:

| Lauf | Zeit |
|---|---:|
| `go test ./internal/hooks/ -count=1` | 39,6 s |
| `go test ./internal/sessions/ -count=1` | 0,5 s |
| `go test ./internal/gitwork/ -count=1` | 4,2 s |

Die Mutantenzahl je Datei ist vorab mit einem Wegwerftest über
`mutants.Generate` gezählt worden, nicht geraten:

| Paket | Mutanten |
|---|---:|
| `internal/sessions` | 127 |
| `internal/gitwork` | 64 |
| `internal/hooks` (ganz) | 814 |
| davon `stop.go` / `subagent.go` / `wikigate.go` | 141 / 90 / 20 |
| `internal/cases` | 429 |
| `internal/dev/importcases` | 328 |

Daraus: `sessions` und `gitwork` kosten zusammen rund 3 min und laufen
vollständig. `internal/hooks` ganz wären 814 × rund 45 s ÷ 8 Arbeiter ≈ 76
min, weit über der Grenze von 30 min — **die Runde ist deshalb mit `-only` auf
die drei Dateien beschränkt, die diese Stufe geschrieben hat**: `stop.go`,
`subagent.go`, `wikigate.go`, zusammen 251 Mutanten, geschätzt ≈ 24 min. Die
übrigen sechs Dateien des Pakets (`worktree.go`, `post_edit.go`, `status.go`,
`hook_session_start.go`, `guard.go`, `pretool.go`) stammen aus früheren Stufen
und sind hier nicht angefasst worden.

`internal/cases` und `internal/dev/importcases` sind Testinfrastruktur und
bleiben aus: 757 weitere Mutanten, und `internal/cases` baut selbst Git-Welten,
also liegt auch dort ein Lauf im Minutenbereich. Sie kämen zusammen auf ein
Vielfaches der Grenze.

Tatsächlich gebraucht: `wikigate.go` 2 min 11 s, `stop.go` 5 min 19 s,
`subagent.go` 5 min 35 s. Die erste Runde über `sessions` und `gitwork` ist
nicht gestoppt worden; die zweite lief warm in 40 s.
Die Schätzung lag hoch, weil `-failfast` einen getöteten Mutanten lange vor den
40 s der vollen Suite beendet; nur Überlebende kosten den ganzen Lauf. Die
zweite Runde brauchte 40 s, 4 min 28 s, 5 min 14 s und 2 min 50 s.

### Befehle

```sh
go run ./cmd/loomux dev mutants ./internal/sessions ./internal/gitwork
go run ./cmd/loomux dev mutants -only wikigate.go ./internal/hooks
go run ./cmd/loomux dev mutants -only stop.go     ./internal/hooks
go run ./cmd/loomux dev mutants -only subagent.go ./internal/hooks
```

### Erste Runde

| Paket bzw. Datei | Mutanten | getötet | überlebt | nicht kompiliert |
|---|---:|---:|---:|---:|
| `internal/sessions` | 127 | 69 | 9 | 49 |
| `internal/gitwork` | 64 | 47 | 4 | 13 |
| `internal/hooks` `wikigate.go` | 20 | 20 | 0 | 0 |
| `internal/hooks` `stop.go` | 141 | 86 | 27 | 28 |
| `internal/hooks` `subagent.go` | 90 | 42 | 23 | 25 |

Zusammen 442 Mutanten, 264 getötet, 63 überlebt, 115 keine Mutanten. Ein
vierundsechzigster Überlebender kam erst in der dritten Runde zum Vorschein,
siehe unten.

### Nachprüfung

`-only` ist ein Dateifilter und kein Mutantenfilter — den einen Mutanten
nachzufahren, wie der Aufgabenbrief es beschreibt, geht damit nicht. Jeder
nachgereichte Test ist deshalb zuerst einzeln gegen genau seinen Mutanten
gestellt worden, über dasselbe `go test -overlay`, das `dev mutants` benutzt
(Skript `killcheck.sh` im Scratchpad, eine Zeile ersetzt, nur der eine Test
gefahren). Danach lief je Datei die volle zweite Runde mit denselben Befehlen:

Über die drei Dateien in `internal/hooks` lief danach noch eine dritte Runde
mit `-workers 4`, um die Zeitüberläufe unten auszuschließen.

| Paket bzw. Datei | 1. Runde | 2. Runde | 3. Runde (4 Arbeiter) |
|---|---:|---:|---:|
| `internal/sessions` | 9 | 4 | — |
| `internal/gitwork` | 4 | 0 | — |
| `internal/hooks` `wikigate.go` | 0 | 0 | 0 |
| `internal/hooks` `stop.go` | 27 | 13 | 10 |
| `internal/hooks` `subagent.go` | 23 | 9 | 8 |

47 Überlebende sind mit nachgereichten Tests getötet, 17 bleiben begründet
stehen: 4 in `internal/sessions`, 6 an `stop.go`, 7 an `subagent.go`, keiner in
`internal/gitwork` und keiner an `wikigate.go`. Die Grundgesamtheit ist 64 und
nicht 63: `subagent.go:54` (a2) zeigte sich erst in der dritten Runde als
Überlebender, in den ersten beiden war er scheingetötet. Er ist der dritte
echte Befund dieser Runde (unten).

Die dritte Runde ist nicht der letzte Stand. Sie lief, bevor die Tests für
`subagent.go:54` und für `stop.go:157` geschrieben waren; die acht bzw. zehn
Zahlen der Spalte sind also die *vor* diesen Tests. Jeder dieser Mutanten ist
einzeln über dasselbe Overlay nachgeprüft: `subagent.go:54` fällt, und die vier
an `stop.go:157` fallen alle vier. Die Tests an
`headTree` und an den Branch-Spannen dagegen waren in der dritten Runde schon
dabei, und deren Zahlen bestätigen sie.

**Scheintötungen durch Zeitüberlauf.** Die Suite von `internal/hooks` braucht
allein 40 s, die Schranke eines Mutantenlaufs steht bei 60 s, und
`dev mutants` zählt einen Zeitüberlauf wie einen roten Lauf (`round.go`, der
`default`-Zweig: „Failed and TimedOut are both killed“). Unter acht
gleichzeitigen Läufen reicht die Luft nicht immer. Drei Mutanten sind so
falsch als getötet gemeldet worden: `stop.go:62` (a4) und `stop.go:119` (a2) in
der zweiten Runde, `subagent.go:54` (a2) in der ersten *und* der zweiten. Alle
drei sind einzeln mit dem Overlay und der ganzen Paketsuite nachgefahren und
überleben; `:62` (a4) ist obendrein wortgleich mit `:62` (a3), der jede Runde
überlebt hat.

**Wieviel davon ist damit gedeckt?** Nur die Überlebenden. Eine Runde kann
einen Mutanten, der in jeder Runde in den Zeitüberlauf lief, grundsätzlich
nicht sehen — genau das hat `subagent.go:54` zwei Runden lang getan, und
gefunden wurde er nur, weil eine dritte mit weniger Arbeitern lief. **Die 148
Tötungen der ersten Runde in `internal/hooks` (20 + 86 + 42) sind deshalb eine
Obergrenze und sind nie nachvalidiert worden**; wieviele davon Zeitüberläufe
waren, steht nirgends. Die übrigen 116 der 264 gehören zu `internal/sessions`
und `internal/gitwork`.
`-failfast` spricht dafür, dass es wenige sind — ein wirklich getöteter Mutant
endet beim ersten roten Test, lange vor der Schranke —, ist aber kein Beweis.
Unberührt davon sind `internal/sessions` und `internal/gitwork`: deren Suiten
brauchen 0,5 s und 4,2 s und erreichen die 60 s unter keiner Last, ihre Zahlen
stimmen so, wie sie dastehen.

**Was das Werkzeug reparieren würde**, nicht der Zuschnitt einer Runde: ein
Zeitüberlauf darf nicht als Tötung zählen, sondern muss als eigener,
unentschiedener Ausgang gemeldet werden — wie ein Mutant, der nicht kompiliert,
eine Zeile im Bericht bekommt statt stillschweigend in eine Spalte zu fallen.
Solange das so bleibt, muss `goTimeout` (heute 60 s, `mutants.go`) über der
Laufzeit der Suite liegen, die mutiert wird, und nicht knapp daneben.

### Getötet

`internal/sessions`

- `sessions.go:97` `if builder.Len() == 0` → `if false`. **Echtes Loch.** Ohne
  den gemeinsamen Namen zeigt `Forget` sein `RemoveAll` auf das
  Zustandsverzeichnis selbst und nimmt die Dateien aller anderen Sitzungen mit.
  Der alte Test sah das nicht, weil er nur prüfte, dass *seine* Datei weg ist —
  und die war sie, aus dem falschen Grund.
  `TestAnIdWithNothingKeepableInItBecomesUnnamed` legt jetzt eine
  Nachbardatei daneben und verlangt, dass sie stehen bleibt.
- `state.go:54` (a2) `err != nil || file.Blocks == nil` → `err != nil`. Neuer
  `TestReadStateWithoutBlocksIsEmpty`: eine Datei ohne `blocks` ist Schaden und
  liest als leer, nicht als Sitzung mit Zähler 0 und gültiger Basis.
- `agents.go:95` `if werr == nil` → `if true`, `if werr != nil`,
  `if !(werr == nil)`. Neuer
  `TestWriteAgentKeepsAFailedWriteOverASuccessfulClose`: die Naht `createTemp`
  gibt eine nur zum Lesen geöffnete Datei zurück, der Schreibversuch scheitert,
  das Schließen nicht. Der alte Test reichte eine bereits geschlossene Datei,
  bei der beides scheitert — daran ist kein Unterschied zu sehen.

`internal/gitwork`

- `gitwork.go:108` `if detail != ""` → `if false`. `TestHeadCommitOutsideARepository`
  prüft jetzt, dass gits eigene Worte nicht ohne Trenner auf den Exitstatus
  laufen (`exit status 128fatal: …`).
- `gitwork.go:158` `if err != nil` → `if false`.
  `TestContentTreeOutsideARepository` verlangt, dass die Meldung den Aufruf
  nennt, der abgelehnt hat (`--git-path`); sonst wird die leere Antwort als
  relativer Pfad weitergetragen und ein ganz anderer Fehler gemeldet.
- `gitwork.go:162` `if !filepath.IsAbs(real)` → `if true`. Neuer
  `TestContentTreeInALinkedWorktree`: in einem verknüpften Arbeitsbaum
  antwortet `rev-parse --git-path index` absolut, und an die Wurzel gehängt
  ist dieser Pfad keiner.
- `gitwork.go:165` `if err := os.MkdirAll(…); err != nil` → `if false`. Im
  Unterfall „scratch is a file“ darf die Meldung nicht die Kopie
  (`index-<pid>`) beschuldigen, sondern das Verzeichnis.

`internal/hooks` `stop.go`

- `:77` und `:82` je `if false`: Host und Nutzlast werden dort gemeldet, wo sie
  scheitern. `TestStopWithAnUnknownHost` verlangt den Namen im Text,
  `TestStopRefusesAPayloadThatIsNoJSON` verlangt, dass nicht die Sitzung
  beschuldigt wird.
- `:177` `if true` und (a3) `code != ExitOK`: `TestStopPassesAndMovesTheBase`
  verlangt Schweigen bei einem sauberen Lauf; neuer
  `TestStopKeepsThisRunsCoverageOnlyWhenItHolds` hält die Uhr an, benennt so
  die Lauf-ID vorweg und pinnt die Regel — grün räumt die Profile weg, ein
  Halt lässt sie liegen, weil der Agent sie lesen können muss.
- `:157` alle vier, der Rückfall `ready = verify.ImportReady`. Neuer
  `TestStopHandsThePlanAnImportReady`. Der erste Anlauf hielt die vier für
  unerreichbar ohne Godot-Welt — das war falsch gerechnet: `stopPlan` ist eine
  Naht (`stop.go:50`), die die Suite ohnehin schon stellt, und die `PlanEnv`,
  die dort ankommt, trägt das `ImportReady`. Ein Stub, der es abgreift, reicht.
  Zwei Fälle: ohne eigenes `ImportReady` muss der Plan trotzdem eines bekommen
  (tötet `if false` und beide Formen von `ready != nil`, die `ready` nil lassen),
  und mit einem eigenen muss er genau dieses bekommen (tötet `if true`, das es
  jedes Mal überschreibt).
- `:209` (a2) `base == "" && head != ""` → `base == ""` und `:214` `if true`:
  `TestStopInAnUnbornRepository` verlangt, dass ein Repository ohne Commit
  weder „no base commit“ noch „is gone“ zu hören bekommt.
- `:215` `if true`: `TestStopPassesAndMovesTheBase` verlangt, dass eine Basis,
  die auflöst, nicht als verschwunden gemeldet wird.
- `:335` `if len(file.Finding) > delivered` → `if true`. Neuer
  `TestStopClearsAFindingFileThatShrankWhileItDelivered`: die Naht `readAgent`
  gibt eine kürzere Liste zurück als zugestellt wurde; ohne die Schranke
  schneidet der Slice über das Ende hinaus.
- `:242` (a1 `true`, a3, a4) und `:246` (a3, a4), also fünf der sechs an
  `headTree`. Neuer `TestStopMeasuresFromTheTreeHeadHoldsWhenTheBaseIsGone`:
  eine verschwundene Basis in einer Welt, deren Arbeitsbaum genau das hält,
  was HEAD hält. Dann sind die beiden Bäume gleich, es ist nichts seit der
  Basis, und keine Bahn läuft. Alle fünf Mutanten geben statt HEADs Baum den
  leeren zurück, damit unterscheiden sich die Bäume und die Kette läuft. Die
  beiden alten Tests erreichten die Funktion zwar, konnten sie aber nicht
  prüfen: in `stopWorld` weicht der Arbeitsbaum ohnehin von HEAD ab.
- `:346` alle vier von `shortSHA`: `TestStopGivesUpAfterThreeInARow` verlangt
  „base stays at no commit“ ohne Basis,
  `TestStopGivesUpOnAFindingItCannotClearAway` den kurzen SHA mit Basis.

`internal/hooks` `subagent.go`

- `:79` `if err == nil` → `if true`: `TestSubagentHooksWithAnUnknownHost`
  verlangt den Hostnamen im Text.
- `:54` (a2) `!found || before.Snapshot == nil` → nur `!found`. **Echtes
  Loch:** `*before.Snapshot` auf einer Datei, die nur eine geparkte
  Erkenntnis trägt, ist eine nil-Dereferenzierung. Neuer
  `TestSubagentStopWithAFindingAndNoSnapshotIsSilent`: so eine Datei gehört
  einem Subagenten, dessen Start dieser Hook nie gesehen hat; er schweigt und
  lässt sie stehen.
- `:82` (a2) `p.SessionID == "" || p.AgentID == ""` → nur `p.AgentID == ""`.
  **Echtes Loch.** Neuer `TestSubagentStartRefusesWithoutASession`: ein
  Schnappschuss ohne Sitzung landet unter dem gemeinsamen Ausweichnamen, wo
  ihn das Stop-Tor der nächsten Sitzung als seinen liest.
- `:133` fünf Mutanten (a1 `false`, a4, drei a3): neuer
  `TestSubagentStopSeesACommitOffEveryBranch`. Ein Commit auf losgelöstem HEAD
  bewegt keinen Branch, also ist HEADs eigene Spanne der einzige Bericht
  darüber.
- `:136` drei Mutanten (a1 `false`, a4, a3) und `:139` vier (a1 `false`, a4,
  zwei a3): neuer `TestSubagentStopSeesACommitOnABranchItIsNotOn`. Über
  `git commit-tree` entsteht ein Commit, auf den `git branch -f` einen Branch
  setzt, ohne dass HEAD sich rührt — damit sind die Branch-Spannen das
  Einzige, was den Commit meldet.
- `:139` (a1 `true`, a2 `was != now`): neuer
  `TestSubagentStopLogsNothingForABranchThatIsGone`. `LogOneline` schreibt
  seine Spanne als `from..to`; bei einem gelöschten Branch ist `to` leer, und
  `from..` liest git als `from..HEAD`. Ein Branch, der hinter HEAD stand und
  gelöscht wird, meldete damit jeden Commit, um den er zurück war, als Neuheit
  dieses Subagenten.

### Überlebende mit Begründung

Alle 17 Überlebenden, jeder mit Verfügung.

`internal/sessions` (4)

1. `agents.go:80` (a1 `false`) — der `json.Marshal`-Fehlerarm von
   `WriteAgent`, ausdrücklich `//coverage:exempt`: Strings, Maps von Strings
   und Slices von Strings kodieren immer. Unerreichbar, also gleichwertig.
2. `agents.go:95` (a1 `false`) — verlangt einen Schreibvorgang, der gelingt,
   und ein Schließen, das scheitert. Durch die Naht `createTemp` (sie gibt ein
   `*os.File` zurück) ist das auf keiner Plattform portabel herzustellen; ein
   gültiger Handle schließt sich, ein ungültiger scheitert schon beim
   Schreiben. Die drei anderen Mutanten derselben Zeile sind getötet.
3. `agents.go:135` (a2, `entry.IsDir()` gestrichen) — gleichwertig:
   `readAgentFile` liest ein Verzeichnis nicht, antwortet `false`, und der
   Eintrag wird so oder so übersprungen.
4. `state.go:78` (a1 `false`) — der `json.Marshal`-Fehlerarm von
   `WriteState`, ausdrücklich `//coverage:exempt`. Unerreichbar.

`internal/hooks` `stop.go` (6)

5. und 6. `:62` (a3, a4 — wortgleich) — der Rückfall auf `"loomux"` in `Stop`.
   Beide Arme
   sind von `TestStopFindsTheBinaryItNames` und
   `TestStopFallsBackToTheCommandName` gefahren, aber beide Läufe enden an der
   leeren Nutzlast, ehe der Name irgendwohin gelangt. Auseinanderhalten könnte
   sie nur eine ganze Kette mit echtem `child.Run` — das ist kein Test dieses
   Tores, sondern ein Ende-zu-Ende-Lauf. **Lücke, bewusst offen.**
7. `:93` (a3, `filepath.Abs` invertiert) — gleichwertig in der Praxis: `Abs`
   scheitert nur, wenn das Arbeitsverzeichnis nicht zu ermitteln ist, und die
   Welten übergeben ohnehin absolute Wurzeln, sodass `root` in beiden Zweigen
   dasselbe bleibt.
8. `:119` (a2, `holdForFindings && code != ExitDenied` → `holdForFindings`) —
   **echt gleichwertig**: im einzigen abweichenden Fall ist `code` schon
   `ExitDenied`, und beide Zweige liefern `ExitDenied`.
9. `:242` (a1 `false`) — **echt gleichwertig**: fällt der Arm für `head == ""`
   weg, so scheitert `TreeOf(root, "")` gleich darauf und `:246` gibt denselben
   `EmptyTree` zurück. Die fünf übrigen Mutanten an `headTree` sind getötet;
   der `//coverage:exempt`-Vermerk deckt nur den `TreeOf`-Fehlerarm an `:246`,
   nicht die Funktion als Ganzes, und die Funktion wird sehr wohl erreicht.
10. `:335` (a3, `>` → `>=`) — **echt gleichwertig**: bei Gleichheit ist
    `file.Finding[delivered:]` leer, `len(rest) == 0` trifft, und es geht
    denselben Weg wie `rest == nil`.

`internal/hooks` `subagent.go` (7)

11. `:86` (a3, `filepath.Abs` invertiert) — wie 7.
12. bis 15. `:133` (a1 `true` und die drei a2) — gleichwertig, aber nicht aus
    dem naheliegenden Grund. Eine Spanne, die dazukommt, obwohl die Bedingung
    sie nicht will, ist entweder `(x, x)` — dann meldet `LogOneline` nichts —
    oder hat eine leere Seite. Eine leere Seite ist hier **kein Fehler**:
    `LogOneline` schreibt `from..to` (`gitwork.go:209`), und git liest die
    fehlende Seite als `HEAD`. Erreichbar ist nur `before.Head == ""`, also
    ein Repository, das beim Start noch keinen Commit hatte; `..after` ist dann
    `HEAD..after`, und HEAD *ist* `after`. Die Spanne ist leer, keine Zeile
    entsteht. Am gelöschten Branch (`:139`) ist genau dieselbe Schreibweise
    sehr wohl aufgefallen und getötet — dort steht HEAD woanders.
16. `:136` (a1 `true`) — **echt gleichwertig**: die Schleife läuft dann auch
    über `before.Heads == nil`, und über eine nil-Map zu iterieren tut nichts.
17. `:139` (a2, `now != ""` allein) — gleichwertig: die Bedingung fällt nur
    weg, wo `was == now`, und `LogOneline(x, x)` meldet nichts. Die beiden
    anderen Mutanten dieser Zeile sind über den gelöschten Branch getötet.
