# Tor-Infrastruktur: Überspringen, Sperre, `{godot}`

Stand 2026-10-06, Basis `origin/master` ae74a98a (v1.0.2), Zweig
`feat/gate-skip-lock-godot`.

## Anlass

space prüft seine GDScript-Hälfte mit eigenen Skripten
(`.claude/hooks/godot_quality.py` mit `toolchain.py`, `suite_shards.py`,
`lcov_merge.py`, `coverage_gate.py`). Ziel ist, dass der pre-commit von space
nur noch loomux ruft und diese Skripte wegfallen. Die Arbeit ist in sechs
Teilprojekte zerlegt, jedes mit eigener Spec, eigenem Plan und eigenem PR:

| # | Teilprojekt | hängt an |
|---|---|---|
| 1 | LCOV-Gate `check lcov` (ersetzt `lcov_merge.py`, `coverage_gate.py`) | — |
| 2 | **Tor-Infrastruktur: diese Spec** | — |
| 3 | Godot-Suite `check godot-suite` (gdUnit4 in Shards, Nano Coverage, veralteter Klassen-Cache) und Godot-Presets für `test` und `coverage` | 1, 2 |
| 4 | Projektregeln als Prüfungen (Testmodul je Quelle, Messreste, ungeprüfte Ordner, Grenze Core/UI) | — |
| 5 | GDScript im Code-Graph (Roadmap G5c), in einer anderen Sitzung | — |
| 6 | Umstellung in space | 1–4 |

Teil 2 schließt drei Lücken, die nicht an Godot hängen, aber für die
Umstellung nötig sind:

- space überspringt seinen 6-Minuten-Lauf, wenn ein Commit nur inerte Pfade
  trägt (`cannot_affect_gate`). Das loomux-Tor fährt heute jede Lane bei jedem
  Commit.
- Nano Coverage schreibt die `.gd`-Quellen auf der Platte um; zwei Läufe im
  selben Baum hinterlassen ihn instrumentiert. space sperrt mit
  `take_suite_lock`. In loomux serialisiert nichts zwei Läufe: weder pre-commit,
  Stop-Gate noch `check` nehmen eine Sperre (`grep` über `internal/hooks` und
  `internal/verify`, 2026-10-06).
- Die Lane muss das Godot-Binary finden. space sucht es in `toolchain.py`;
  ein Preset kann es heute nicht nennen.

## Ziel und Erfolg

Eine Lane kann für Commits und Turn-Enden aussetzen, die nur Pfade ändern, die
sie für inert erklärt; eine Lane kann sich je Checkout gegen einen zweiten Lauf
sperren; eine `gdscript`-Lane kann `{godot}` schreiben und bekommt ein Binary,
das zu `project.godot` passt, oder einen Klartext, warum nicht. Ohne neue
Schlüssel verhält sich jedes Projekt wie heute.

## Leitlinie: Opt-out

Was loomux kann, ist nach `init` an; abgeschaltet wird per Konfiguration
(`[modules]`, `<kind> = false`). Das Überspringen ist keine Funktion, die man
anschaltet, sondern ein teilweises Abschalten einer Lane, und steht darum auf
der Abschaltseite: ohne Eintrag läuft jede Lane bei jedem Commit. Die Sperre
und `{godot}` sind dort an, wo das Preset sie setzt (Teil 3).

## Messungen, die die Entscheidungen tragen

Alle am 2026-10-06, Skripte im Scratchpad der Sitzung (`mix.sh`,
`skipsim.py`, `missed.py`); sie gehen nach `docs/{en,de}/benchmarks.md`.

- Volles Tor von loomux (`go run ./cmd/loomux check precommit`, Stand
  757d600b, warm): 3 min 50 s, davon `test/go` 226,6 s.
- `git diff --cached --name-only` und `git diff HEAD --name-only` in space,
  Median aus 10 Läufen mit PowerShell `Measure-Command`: 89 ms und 83 ms.
- `Godot_v4.7.1-stable_mono_win64_console.exe --version`, Median aus 5: 57 ms,
  Ausgabe `4.7.1.stable.mono.official.a13da4feb`.
- Überspringen nach Stack-Endung, letzte 400 Commits ohne Merges in 12 Repos.
  „Hybrid“ überspringt nur, wenn jeder Pfad die Endung eines *anderen* Stacks
  hat; Dateien ohne Endung, `.cfg`, `.tscn` und die `ignored`-Endungen zählen
  als unbekannt und lassen laufen. „Verpasst“ zählt übersprungene Commits, die
  eine Datei änderten, die ein Test des Stacks nachweislich aus dem Repo liest:

  | Projekt, Lane | automatisch | hybrid | verpasst | gelesen von |
  |---|---|---|---|---|
  | loomux, go | 39 % | 35 % | 0 | `internal/dev/benchcorpus/committed_test.go` liest `docs/benchmarks*.json` (unbekannt, läuft) |
  | space, python | 97 % | 77 % | 83 | `.claude/hooks/tests/test_conventions.py` liest `*_test.gd` |
  | odysseus, python | 31 % | 29 % | 65 | Tests lesen `static/js/*.js` über node |
  | iam_backend, python | 33 % | 25 % | 16 | `tests/test_claude_rules.py` liest `.claude/rules/*.md` |
  | open-design, typescript | 24 % | 15 % | 13 | e2e liest `skills/*/SKILL.md` |
  | Strata, python | 66 % | 34 % | 1 | `serve/test_*.py` liest `serve/web/index.html`, `app.js` |

  Eine Automatik nach Stack schaltet also in fünf von sechs geprüften
  Projekten Tests still ab, die betroffen waren. Das trägt Entscheidung 1.

## Entscheidungen

### 1. Überspringen: `skip_when_only`

1. **Schlüssel.** `skip_when_only = ["<glob>", …]` in der Tabellenform jeder
   Lane, `[verify.<stack>.<kind>]`. Eine leere Liste ist ein Ladefehler wie bei
   `needs`. Kein Preset setzt ihn. Die String- und Listenform einer Lane
   bleibt, was sie ist (ganze Lane); wer den Schlüssel will, schreibt die
   Tabelle.
2. **Muster.** Relativ zum Repo-Root, mit Vorwärtsschrägstrichen, in der
   Syntax von `[policy.paths]`. Der Matcher `matchGlob` aus
   `internal/hooks/guard.go` zieht dafür als exportierte Funktion in
   `internal/pathkey` um (`hooks` importiert `verify`, umgekehrt ginge es
   nicht); `hooks` ruft ihn dort. Ein Muster, das `path.Match` nicht lesen
   kann, ist ein Ladefehler; die Prüfung nimmt `path.Match(glob, "")` wie
   `config.ReadPolicy`, und ein Fehler beim Abgleich zur Laufzeit lässt die
   Lane laufen.
3. **Wann.** Zur Planzeit, wenn eine Liste geänderter Pfade vorliegt, sie
   nicht leer ist und jeder Pfad auf mindestens ein Muster passt. Eine leere
   Liste (etwa `git commit --amend` ohne neue Änderung im Index) lässt
   laufen, ebenso jeder Edit-Scope. Beim Abschluss eines Merges ist die Liste
   der Unterschied zum ersten Elternteil, also die hereinkommenden
   Änderungen; das ist richtig so und kein Sonderfall.
4. **Woher die Liste kommt.**
   - *Commit:* `loomux check` erkennt den pre-commit an `GIT_INDEX_FILE`
     (Variable gesetzt). Es fragt `git diff --cached --name-only` unter genau
     dieser Variable, aufgelöst gegen das Arbeitsverzeichnis des Hooks wie in
     `gitwork.CommitIndex`, damit `git commit <pfad>` mit
     `next-index-<pid>.lock` die Pfade dieses Commits liefert. Die Erkennung
     liest die Variable selbst: `CommitIndex` antwortet bei leerer Variable
     „ganzer Index“, für das Überspringen heißt leer aber „von Hand“. Ohne die
     Variable lief `check` von Hand, und alles läuft. Bestehende Git-Hooks
     brauchen keine Änderung.
   - *Turn-Ende:* `git diff-tree -r --name-only <green> <tree>` zwischen dem
     zuletzt grünen Baum der Sitzung (`sessions.SessionState.Green`) und dem
     Baum, den `stopTrees` gerade geschrieben hat. Ohne grünen Baum gilt der
     Baum der Basis. Gegen die Basis allein würde nach dem ersten Code-Turn
     nie mehr übersprungen, weil dessen Änderung immer mitliefe.
   - Scheitert der git-Aufruf, liegt keine Liste vor, und alles läuft.
5. **Ergebnis.** `not-applicable` mit der Notiz
   `only skip_when_only paths changed (<erster pfad>, +N)`. Eine solche Lane
   wird nicht scharf geschaltet (`GreenKeys` nimmt nur `ok`) und löst im
   strengen Befund kein „nothing to check“ aus (`CheckVerdict` zählt
   `not-applicable` als behandelt).
6. **Nachfolger.** `settle` überspringt heute einen Vorgänger mit
   `Pre == not-applicable` als „nicht angefordert“; eine Lane mit `measure`
   (das Go-Preset hat eines an `coverage`) fährt dann die volle Suite selbst,
   und die Ersparnis wäre still null. Eine wegen `skip_when_only`
   übersprungene Lane bekommt darum eine eigene Markierung am `Job`; `settle`
   hängt Nachfolger an sie, und `inherit` gibt ihnen ihr `not-applicable`. Der
   Eintrag an `test` deckt so `coverage` mit ab. Eine mit `= false`
   abgeschaltete Lane verhält sich wie bisher.

### 2. Sperre: `lock`

1. **Schlüssel.** `lock = true|false` in der Tabellenform einer Lane. Ohne
   Schlüssel keine Sperre. Das Preset setzt sie mit Teil 3 an die
   Godot-Lanes; ein Projekt schaltet sie mit `lock = false` ab. Bis dahin
   kann space sie an seine eigene Test-Lane hängen.
2. **Einheit.** Je Stack und Area, nicht je Kind: Test-Lane und eine
   Coverage-Lane mit eigenem `measure` fahren dieselbe Suite über dieselben
   Quellen. Gehalten für die Prozesse einer Lane, `measure` eingeschlossen;
   frei, sobald die Lane endet.
3. **Ort.** `.loomux/state/locks/<stack>-<area>.lock` im Root des Checkouts,
   `/` in der Area als `_` wie bei `CoverPaths`, die Root-Area als `root`. Ein
   verlinkter Worktree hat sein eigenes `.loomux/state` und damit seine eigene
   Sperre. Der Ordner wird angelegt wie `PrepareCover` den Coverage-Ordner
   anlegt, weil der Wächter Agenten unter `.loomux/state` nicht schreiben
   lässt.
4. **Mechanik.** `internal/lock`: das Betriebssystem hält die Sperre, ein
   gestorbener Halter gibt sie frei, eine Alters-Heuristik wie in space
   (30 min) entfällt. Die Sperrdatei bleibt leer, weil Windows den gesperrten
   Bereich auch fürs Lesen sperrt. Wer hält, steht in `<…>.lock.who`: PID,
   Aufrufer (`check precommit`, `hook stop`, `check <profil>`), Startzeit in
   UTC. Geschrieben nach dem Nehmen, gelöscht vor der Freigabe; bei freier
   Sperre wird eine übrig gebliebene `.who` ignoriert.
5. **Warten.** Alle 250 ms `TryAcquire`, wie `WaitFree`. Dabei wird kein
   `max_parallel`-Platz gehalten (dieselbe Regel, die `Run` für `After`
   befolgt). Der erste Fehlversuch schreibt eine Zeile auf stderr:
   `<lane>: waiting for the lock (held by loomux pid <pid>, <aufrufer>, since <n>s)`.
   - Turn-Ende und Edit-Scope: die Wartezeit zählt gegen das Budget; ist es
     aufgebraucht, endet die Lane als `budget`.
   - pre-commit und `check`: höchstens `[verify].timeout`, danach
     `timed-out` (rot) mit dem Halter in der Ausgabe.
6. **Ein `budget` erreicht die Nachfolger in jedem Scope.** Heute reicht
   `inherit` ein `budget` nur im Edit-Scope weiter. Am Turn-Ende startet der
   Nachfolger (`coverage` nach `test`), findet den Report nicht („is missing:
   the measuring run did not write it“) und endet als `failed`; der Turn
   blockiert, nach `MaxBlocks` gibt das Gate auf. Das ist ein Fehler, den es
   auf master schon gibt (ein `test/go` über 270 s trifft ihn), und die
   Sperre machte ihn zur Regel: jeder Turn im Fenster eines laufenden
   pre-commit. Behoben, indem `inherit` ein `budget` im Check-Scope an einen
   Nachfolger gibt, der Dateien des Vorgängers liest (`Job.Reads` nicht
   leer); ein abgebrochener Vorgänger hat sie nicht oder nur halb
   geschrieben. Ein Nachfolger ohne `Reads` läuft weiter wie bisher: das legt
   `TestRunLetsACheckRunPastABudgetPredecessor` seit dem ersten Verify-Commit
   bewusst fest, `after` ordnet dort nur. Danach meldet `stopVerdict` wie bei jedem `budget`
   „not everything was verified“ und antwortet `ExitInternal`: der Turn geht
   durch, Basis und grüner Baum bleiben stehen (nachgelesen in
   `internal/hooks/stop.go`, `stopVerdict`). Der Fix ist ein eigener Commit
   vor den übrigen, weil der Fehler vor dem Zweig bestand.

### 3. Platzhalter `{godot}`

1. **Geltung.** In jedem Befehl einer `gdscript`-Lane, auch in `measure` und
   `measuring`. Aufgelöst zur Planzeit, wie `ImportReady`, und nur für Lanes,
   die ihn nennen. In einer Lane eines anderen Stacks ist er ein Ladefehler.
2. **Suche**, das erste Gefundene gilt:
   1. `GODOT_BIN` (Konvention von gdUnit4);
   2. `[verify.gdscript] godot = "<pfad>"`, relativ zum Root oder absolut;
      neuer Schlüssel neben `import_check`;
   3. PATH: `godot`, dann `godot4`.

   Unter Windows wird statt einer gefundenen `X.exe` die `X_console.exe`
   daneben genommen, wenn es sie gibt: die Fenster-Variante schreibt nichts
   auf stdout.
3. **Prüfung gegen `project.godot` der Area.** Pflichtquelle, kein Raten
   (Memory „Projektdateien sind die Quelle“). `<binary> --version` einmal je
   Binary und Lauf; daraus Haupt- und Nebenversion und das Token `mono`.
   `config/features` liefert die Version (das erste Element der Form
   `<zahl>.<zahl>`), ein `[dotnet]`-Abschnitt verlangt Mono. Haupt- und
   Nebenversion müssen gleich sein, die Patch-Version ist frei: eine neuere
   Nebenversion würde das Projekt beim Öffnen hochstufen und Projektdateien
   umschreiben.
4. **Ergebnisse.**

   | Fall | Zustand | Text nennt |
   |---|---|---|
   | nichts gefunden | `missing-tool` | die drei Quellen |
   | `--version` ohne Ausgabe oder Exit ≠ 0 | `unready` | unter Windows die `_console.exe` |
   | Version oder Mono passt nicht | `unready` | gefundene Version und Quelle, verlangte Version und Mono, die beiden Stellschrauben |
   | `project.godot` fehlt in der Area | `unready` | dass keine Version zu prüfen ist |

   `unready` ist wie heute rot bei `check` und pre-commit und nicht rot im
   Edit-Scope.

## Grenzen

- `config set … --propose` erreicht keinen Lane-Schlüssel: die Schlüsseltabelle
  in `internal/config/schema/schema.go` kennt unter `verify` nur
  `max_parallel`, `timeout` und `profiles`, nicht einmal `import_check`. Die
  neuen Schlüssel schreibt ein Mensch, wie heute alle Lane-Schlüssel; die
  Roadmap-Zeile „`verify.profiles` through `config set`“ bleibt der Ort dafür.
- Die Muster sind wie `[policy.paths]` groß-/kleinschreibungsgenau. Ein Pfad,
  der nur in der Schreibweise abweicht, passt nicht, und die Lane läuft: die
  Abweichung geht zur sicheren Seite.
- `git commit --amend` vergleicht den Index mit HEAD, also mit dem Commit, der
  ersetzt wird: es zählt nur, was das Amend hinzufügt. Fügt es nur Doku hinzu,
  setzt die Lane aus, obwohl der ersetzte Commit Code trägt. Der wurde beim
  Anlegen schon geprüft; war das mit `--no-verify`, prüft ihn auch dieses Tor
  nicht. Ein Amend ist im Hook nicht von einem gewöhnlichen Commit zu
  unterscheiden (beide reichen `.git/index`, `commitindex.go`).
- Die Sperre schützt vor loomux-Läufen. Ein Godot-Editor, der dieselben Quellen
  zur selben Zeit öffnet, sieht sie nicht.

## Tests

Jede neue Testzeile läuft vor dem Code rot; jede Regel bekommt die
Mutationsrunde mit einem Kontrollmutanten.

- Überspringen: jede Form aus der gemessenen Tabelle in
  `internal/gitwork/commitindex.go` (`git commit`, `-a`, `<pfad>`, `--amend`,
  Abschluss eines Merges mit hereinkommendem Code), jeweils einmal mit nur inerten und einmal mit einem
  zusätzlichen nicht inerten Pfad; `check` von Hand ohne `GIT_INDEX_FILE`;
  Stop-Gate mit und ohne grünen Baum, und ein Turn, dessen Code-Änderung schon
  grün war und der jetzt nur Doku ändert; leere Liste; Kette `test → coverage`
  mit `measure`, die vor der Änderung nachweislich die Suite in `coverage`
  fährt; `= false` bleibt unverändert; ungültiges Muster beim Laden.
- Sperre: zwei Handles auf dieselbe Sperre in einem Prozess (`LockFileEx`
  sperrt je Handle); Budget, Timeout, Halter-Text; `max_parallel = 1`, eine
  fremde Lane läuft, während eine andere wartet; eine übrig gebliebene `.who`
  bei freier Sperre; zwei Areas sperren getrennt.
- `budget` an Nachfolger: Kette `test → coverage` im Check-Scope, `coverage`
  mit `Reads`, `test` endet als `budget`; vor dem Fix wird `coverage`
  `failed` (rot), danach `budget`, und `stopVerdict` antwortet
  `ExitInternal` ohne grünen Baum. Derselbe Fall ohne `Reads` läuft weiter
  (`TestRunLetsACheckRunPastABudgetPredecessor` bleibt grün).
- `{godot}`: Fake-Binary über den Fake-Tool-Mechanismus der Testwelten; alle
  drei Quellen zugleich gesetzt, damit die Reihenfolge unterschieden wird;
  Wahl der `_console.exe`; Parser für `4.7.1.stable.mono.official.<hash>` und
  `4.7.1.stable.official.<hash>`; Mono verlangt und fehlt; Nebenversion
  abweichend; gleiche Nebenversion mit anderem Patch; `project.godot` fehlt;
  `{godot}` in einer `go`-Lane ist ein Ladefehler.
- Bestehende Aufnahmen unter `testdata/cases/` behalten ihre Ausgabe, weil
  kein Preset die Schlüssel setzt; vor dem Plan per `grep` prüfen, ob eine
  Aufnahme `--show` mit allen Lane-Schlüsseln druckt.

## Doku und Roadmap im selben PR

- `docs/{en,de}/configuration.md`: `skip_when_only`, `lock`,
  `[verify.gdscript] godot`, `{godot}`.
- `docs/{en,de}/benchmarks.md`: die Messungen oben, mit Datum.
- `README.md` und `README.de.md`, Abschnitt Roadmap: Zeilen für Teil 1, 3, 4
  und 6 unter „Coming“; Teil 2 erscheint dort nicht, weil er in diesem PR
  fertig wird. Die Priorität der neuen Zeilen setzt der Nutzer im PR.
- Label `release:minor` (neue Schlüssel, kompatibel).

## Nachträge

Aus der Vorabprüfung des Plans am 2026-10-06, vor der Umsetzung.

1. **Zu 2.6, welcher Nachfolger ein `budget` übernimmt.** Nicht „`Job.Reads`
   nicht leer“, sondern ein neues Feld `Job.Consumes`: die Lane liest, was ihr
   Vorgänger schreibt, per Dateiname (`Reads`) oder, bei Python, über
   `COVERAGE_FILE` (die versteckten Pfade der Verknüpfung). Pythons
   Coverage-Lane hat `Reads == nil` und liest trotzdem die Datendatei des
   Test-Laufs; mit `Reads` allein bliebe der Fehler dort bestehen.
2. **Zu 1.6, welcher Nachfolger an einer übersprungenen Lane hängt.** Nur
   einer, der ihre Dateien liest (`Consumes`). Ein Nachfolger, der mit
   `after` nur ordnet, läuft wie hinter einer mit `= false` abgeschalteten
   Lane: er hat mit dem Überspringen nichts zu tun, und ohne eigenen
   `skip_when_only` gilt für ihn die Leitlinie „ohne Eintrag läuft jede
   Lane“. `TestRunLetsACheckRunPastABudgetPredecessor` hält dasselbe für
   `budget` fest.
3. **Zu „Tests“, die Commit-Formen.** `StagedPaths` wird an jedem Index
   geprüft, den die gemessene Tabelle nennt: `.git/index` (absolut und
   relativ zum Hook-Verzeichnis), `index.lock` (`git commit -a`),
   `next-index-<pid>.lock` (`git commit <pfad>`) und der Abschluss eines
   Merges (Index gegen den ersten Elternteil). `--amend` reicht `.git/index`
   wie ein gewöhnlicher Commit (siehe „Grenzen“) und hat keinen eigenen Weg.
   Die Paarung „nur inerte Pfade / ein nicht inerter dazu“ prüfen die
   Plan-Tests von `skippedBy`, nicht jede Commit-Form einzeln.
4. **Zu „Tests“, `{godot}`.** Statt eines Fake-Binarys über die Testwelten
   bekommt die Suche ihre Antworten als eingesetzte Funktionen
   (`godotSources`: Variable, PATH, vorhandene Dateien, `--version`), und
   `GodotFor` wird mit einem eingesetzten Prozessstart geprüft. Das trennt
   die Reihenfolge der drei Quellen ohne echte Dateien auf jedem Betriebssystem.

Aus dem Abschluss-Review, nach der Umsetzung.

5. **Zu 3.1, wann `{godot}` in `measure` gefragt wird.** Nur wenn der
   Messschritt tatsächlich genommen wird, also die Lane an keinem Vorgänger
   hängt. Eine lesende Lane hinter einer übersprungenen oder laufenden Lane
   fragt nicht; sonst wäre ein Doku-Commit ohne Godot-Binary rot, obwohl der
   Messschritt nie liefe (1.6). Befehle, die `{godot}` nennen, fragen weiter in
   jedem Fall; eine lesende Lane, deren eigene Befehle `{godot}` nennen, bleibt
   hinter einer übersprungenen Lane darum `missing-tool`, wenn kein Binary da
   ist. Kein Preset hat diese Form; offen für Teil 3.

Aus einem `/code-review` nach dem Abschluss-Review; jeder Befund wurde vor
dem Fix mit einem roten Test bestätigt.

6. **Zu 1.4, Turn-Ende mit Commits im Turn.** Ist HEAD im Turn gewandert,
   zählen zu den geänderten Pfaden auch die zwischen dem Basis-Commit und
   HEAD (`diff-tree <base> <head>`), vereinigt mit dem Baumvergleich gegen
   den grünen Baum. Die Graph-Lane urteilt gegen HEAD; ohne die Vereinigung
   setzte sie nach einem Commit aus, dessen Code schon grün war. Scheitert
   einer der beiden git-Aufrufe, gibt es keine Liste, und alles läuft.
7. **Zu 1.6, Graph-Träger.** Eine übersprungene Graph-Lane trägt den Graphen
   nicht; die Graph-Lane des nächsten Stacks baut ihn.
8. **Zu 3.3 und 3.4, die Probe zählt gegen das Budget.** Am Turn-Ende und im
   Edit-Scope ist das `--version` der Godot-Suche durch das Restbudget
   begrenzt (höchstens 30 s), und `Run` bekommt nur, was nach dem Planen vom
   Budget übrig ist; ein aufgebrauchtes Budget heißt `budget` für jede Lane.
   Schneidet das Budget die Probe ab, ist die Lane `budget`, nicht `unready`,
   und das Ergebnis wird nicht zwischengespeichert. In `check` und pre-commit
   gilt weiter die feste Grenze von 30 s. Der Post-Edit-Hook baut den Finder
   einmal je Aufruf, nicht je Datei.
