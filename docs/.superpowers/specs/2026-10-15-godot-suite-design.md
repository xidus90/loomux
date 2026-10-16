# `loomux check godot-suite` und die GDScript-Presets für `test` und `coverage`

Stand 2026-10-15, Basis `origin/master` 1660b9f8 (v1.5.0), Zweig
`docs/godot-suite`.

## Anlass

Teil 3 der Zerlegung aus `2026-10-06-gate-skip-lock-godot-design.md`. Teil 2
(`skip_when_only`, `lock`, `{godot}`) und Teil 1 (`check linecov`,
`2026-10-14-check-linecov-design.md`) sind auf master. space fährt seine
gdUnit4-Suite noch mit `.claude/hooks/godot_quality.py`, `suite_shards.py`
und `toolchain.py`, dazu zwei GDScript-Dateien im Projekt
(`harness/coverage_shard_hook.gd`, `harness/disk_instrumentation.gd`). Diese
Spec ersetzt den Suite-Teil davon durch einen eingebauten Befehl und gibt dem
Stack `gdscript` Presets für `test` und `coverage`. Die übrigen Prüfungen aus
`godot_quality.py` (Testmodul je Quelle, Grenze Core/UI, ungeprüfte Ordner,
Messreste als Commit-Regel) sind Teil 4.

## Ziel und Erfolg

Ein Godot-Projekt mit gdUnit4 unter `addons/` bekommt ohne eine Zeile
Konfiguration eine `test`-Lane, mit Nano Coverage zusätzlich eine
`coverage`-Lane, die jede Zeile an `check linecov` hält. Erfolg heißt:

- space kann in Teil 6 `suite_shards.py`, den Suite-Teil von
  `godot_quality.py`, `toolchain.py` (soweit es Godot sucht) und beide
  Harness-Skripte löschen.
- Ein Lauf gegen space liefert dasselbe Urteil wie `suite_shards.py` und ist
  nicht langsamer (Referenz: 8 Scherben, rund 63 s für 1837 Fälle).
- Ein veralteter Klassen-Cache wird vor dem Lauf erkannt und mit dem Befehl
  benannt, der ihn erneuert, ohne `project.godot` zu verderben.

## Leitlinien

Wie in Teil 1: mehrsprachig denken, erkennen statt konfigurieren, die Dateien
der Werkzeuge sind die Quelle (`project.godot`, `plugin.cfg`, `results.xml`),
erweiterbar über Daten, laut statt still.

## Was space heute tut (Bestand, 2026-10-15)

Gelesen in `space/.claude/hooks/` und `space/godot/`:

- Engine Godot 4.7.1 mono, gdUnit4 6.2.0, Nano Coverage 1.0 (GDExtension),
  beide vendored unter `godot/addons/`. Das Godot-Projekt liegt in der Area
  `godot/`.
- Je Scherbe ein Prozess:
  `<engine> --path . -s -d --remote-debug tcp://127.0.0.1:0 res://addons/gdUnit4/bin/GdUnitCmdTool.gd -a <suite> … -c --headless --ignoreHeadlessMode -rd .shards/<i>/reports`.
  Port 0 wird nie gebunden; Godot fällt bei einem Parsefehler darum nicht in
  die `debug>`-Schleife. `GdUnitCmdTool.gd` direkt statt `runtest.cmd` spart
  einen zweiten Engine-Start je Scherbe.
- Scherben: `min(8, cpu_count)`, LPT nach den `<testsuite time>` früherer
  Reports, eine Suite-Datei je Einheit. Je Scherbe eigenes `APPDATA`
  (`user://`), eigene Report- und Coverage-Verzeichnisse.
- Urteil aus `results.xml`, nie aus dem Exit-Code: gdUnit4 hat bei
  fehlschlagenden Fällen 0 geliefert (Wiki von space,
  `gdscript-pruefkette.md`), 100 ist ein Testfehler, 105 ein Parsefehler der
  ganzen Suite, 5 ein veralteter Klassen-Cache.
- Nano Coverage instrumentiert im Speicher, übergeht dabei aber Autoloads.
  space instrumentiert darum vor allen Scherben auf der Platte
  (`DiskInstrumenter.instrument_to_disk({"save_project_settings": false})`)
  und stellt danach zurück (`restore_from_disk`). Die Instrumentierung trägt in
  Zeile 1 jeder Datei `# __NANO_COVERAGE_INSTRUMENTED__`.
- Der eigene Session-Hook lenkt je Scherbe `nano_coverage/general/data_store_dir`
  und `report_dir` um, kopiert bei Platten-Instrumentierung `coverage.meta`
  in die Ablage der Scherbe und schreibt am Ende LCOV.
- Zwei registrierte Coverage-Hooks (der eigene und Nanos `gdunit_hook.gd`)
  ergeben `LH:0`. Jeder Editor- oder Importlauf schreibt `project.godot` um
  und trägt Nanos Hook wieder ein.
- Klassen-Cache: `class_name`-Deklarationen gegen
  `.godot/global_script_class_cache.cfg`; der Befund nennt
  `<engine> --path godot --headless --editor --quit-after 2`, danach muss
  `project.godot` von Hand zurückgesetzt werden.
- Kein Timeout, keine Wiederholung; Flackern im vollen Lauf ist bekannt und
  ungeklärt.

## Entscheidungen

### 1. Befehl

```
loomux check godot-suite [--godot <bin>] [--coverage <dir>] [--root <dir>] [--restore]
```

1. `--root`: das Godot-Projekt (Verzeichnis mit `project.godot`). Vorgabe ist
   das Arbeitsverzeichnis, in einer Lane also die Area.
2. `--godot`: das Binary. In einer Lane setzt es `{godot}`. Ohne Flag sucht
   der Befehl wie `{godot}` (Teil 2, 3.2) und prüft gegen `project.godot`
   (3.3); ein Fehlschlag dort ist ein Befund.
3. `--coverage <dir>`: misst mit Nano Coverage und legt je Scherbe
   `<dir>/<i>.info` ab (Entscheidung 5).
4. `--restore`: stellt nur eine auf der Platte gebliebene Instrumentierung
   zurück (Entscheidung 4.4) und läuft sonst nichts.
5. Ausgabe wie `check linecov`: Befunde auf stdout, Hinweise auf stderr als
   `loomux check godot-suite: note: …`. Exit 0 grün, 1 bei Befunden, 2 bei
   falschem Aufruf.
6. `godot-suite` kommt in `Reserved` (`internal/verify/schema.go`).

### 2. Ablauf

1. **Vorprüfungen.** Jede Vorprüfung ist ein Befund; ist eine rot, startet
   keine Engine.
   - gdUnit4 fehlt (`addons/gdUnit4/bin/GdUnitCmdTool.gd`).
   - **Klassen-Cache** (Entscheidung 6).
   - **Hook-Konflikt**, nur mit `--coverage`: `[gdunit4] hooks/session_hooks`
     in `project.godot` nennt einen Hook, dessen Quelle `CoverageApi` oder
     `NanoCoverage` enthält. Der Befund nennt den Eintrag und
     `loomux godot rescan` als Weg, ein vom Editor zurückgeschriebenes
     `project.godot` künftig zu vermeiden.
   - Rest einer Instrumentierung auf der Platte: wird zurückgestellt
     (Entscheidung 4.3), kein Befund.
2. **Suites erkennen.** Jede `.gd` unter `--root`, deren Quelle eine Zeile
   `extends GdUnitTestSuite` trägt; übergangen werden `addons/`, `.godot/`
   und andere Checkouts (Verzeichnis mit `.git`-Eintrag, wie linecov 4.4).
   Keine Namensannahme (`*_test.gd`). Keine Suite ist der Befund
   `no gdUnit4 test suite under <root>`.
3. **Scherben planen.** Anzahl `min(8, runtime.NumCPU(), Suites)`. Gewichte
   sind die Laufzeiten je Suite aus `.loomux/state/godot-suite/<area>.json`
   im Root des Checkouts (Area wie bei `CoverPaths` geschrieben); eine Suite
   ohne Zeit bekommt den Mittelwert, ohne jede Zeit 1. LPT: absteigend nach
   (Gewicht, Pfad), jede Suite in die leichteste Scherbe. Deterministisch,
   eine flackernde Suite bleibt in ihrer Scherbe.
4. **Kleber schreiben** (Entscheidung 3) nach `<root>/.godot/loomux/`.
5. **Scherben fahren**, alle zugleich, je Scherbe:
   ```
   <godot> --path <root> -s -d --remote-debug tcp://127.0.0.1:0 res://.godot/loomux/run.gd
           -a res://<suite> … -c --headless --ignoreHeadlessMode -rd <shard>/reports
   ```
   `<shard>` ist `.loomux/state/godot-suite/<runid>-<area>/<i>/` im Root des
   Checkouts. Umgebung: `APPDATA` (Windows) und `XDG_DATA_HOME` (sonst) auf
   `<shard>/userdata`, damit `user://` je Scherbe getrennt ist; mit
   `--coverage` zusätzlich `LOOMUX_COVERAGE_SHARD_DIR=<shard>/coverage`.
   Gestartet wird über `internal/child` wie jede Lane.
6. **Urteil** aus dem neuesten `results.xml` je Scherbe (Entscheidung 7).
7. **Laufzeiten zurückschreiben**, nur aus lesbaren Reports, nur für
   gelaufene Suites; andere Einträge bleiben.
8. **Aufräumen:** `<shard>`-Verzeichnisse eines grünen Laufs werden gelöscht,
   die eines roten bleiben für die Untersuchung; Reste älter als 24 h räumt
   jeder Lauf weg (wie `CleanCover`).

Keine Wiederholungen. Kein eigenes Timeout: die Lane hat `[verify].timeout`,
und ein abgebrochener Lauf heilt beim nächsten Start (4.3).

### 3. Eingebetteter GDScript-Kleber

Drei Dateien unter `internal/verify/godotsuite/gd/`, eingebettet mit
`embed`, geschrieben bei jedem Lauf nach `<root>/.godot/loomux/`. `.godot/`
liegt unter `res://`, ist von Godots Vorlage für `.gitignore` erfasst und
wird von Nano Coverage als Teil des Projekts nicht instrumentiert (zu
belegen, Abschnitt „Zu belegen“). Das Projekt braucht keine eigene Zeile.

- **`run.gd`** erbt von `res://addons/gdUnit4/bin/GdUnitCmdTool.gd`. Ist
  `LOOMUX_COVERAGE_SHARD_DIR` gesetzt, hängt es in `_initialize` vor
  `super._initialize()` eine Instanz von `hook.gd` an die Hooks des
  `GdUnitTestSessionHookService` an, ohne `register()`: `register()` speichert
  die Hook-Liste in die Projekteinstellungen und schriebe `project.godot` um.
  Fehlt der Dienst oder seine Liste (andere gdUnit4-Version), endet der
  Lauf mit einer Meldung auf stderr und Exit ≠ 0, die die gdUnit4-Version aus
  `plugin.cfg` nennt.
- **`hook.gd`** tut, was space' `coverage_shard_hook.gd` tut: Ablage und
  Reportverzeichnis auf `LOOMUX_COVERAGE_SHARD_DIR`, Workspace
  `loomux-<i>`, `clear_coverage_data`; bei Platten-Instrumentierung
  `coverage.meta` aus der konfigurierten Ablage kopieren, sonst
  `ProjectBootstrapper.instrument_all_scripts()`; am Ende `save_session`,
  `reset`, `generate_coverage_report`. Ein fehlgeschlagener Schritt ist eine
  Meldung im Session-Log; das Fehlen des LCOV macht Entscheidung 7 zum
  Befund.
- **`disk.gd`** ist space' `disk_instrumentation.gd`: `-- instrument` mit
  `save_project_settings: false`, `-- restore` mit `is_instrumented()` als
  Kurzschluss; JSON-Ergebnis auf stdout, Exit 0 nur bei `status == "ok"`.

### 4. Instrumentierung auf der Platte

1. **Wann.** Mit `--coverage`, wenn `[autoload]` in `project.godot` ein
   Skript nennt, das unter `--root` liegt und nicht unter
   `[nano_coverage] general/ignore_paths` fällt (Dialekt `nano` aus linecov,
   Nachtrag 1.1, mit Vorgabe `**/*_test.gd` und `addons/` bei
   `general/ignore_addons`). Sonst instrumentiert `hook.gd` je Scherbe im
   Speicher, und der Baum bleibt unberührt.
2. **Ablauf.** Vor dem Instrumentieren je `.gd` unter `--root` (ohne
   `addons/`, `.godot/`, andere Checkouts) SHA-256 und mtime merken;
   `disk.gd -- instrument`; Scherben; `disk.gd -- restore` in jedem Fall,
   auch wenn eine Scherbe nicht starten konnte.
3. **Selbstheilung.** Trägt vor dem Lauf eine Quelle die Markerzeile
   `# __NANO_COVERAGE_INSTRUMENTED__`, wird zuerst `disk.gd -- restore`
   gefahren, mit Hinweis auf stderr. So heilt ein Lauf, den ein Timeout oder
   ein Absturz beendet hat.
4. **Nach dem Restore.** Jede gemerkte Datei wird gegen ihren Hash geprüft;
   eine Abweichung ist der Befund `<pfad> differs after restoring the
   instrumentation`. Stimmt der Hash, wird die mtime mit `os.Chtimes`
   zurückgesetzt: sonst wäre jede Quelle neuer als der Report und linecov 9.2
   meldete sie. Scheitert das Restore, ist das der einzige Befund:
   `sources are still instrumented; run loomux check godot-suite --restore`.
   Die Ergebnisse der Scherben werden dann nicht gemeldet.

### 5. Ablage der Coverage

1. `--coverage <dir>`: godot-suite legt `<dir>` als Verzeichnis an und kopiert
   das LCOV jeder Scherbe als `<dir>/<i>.info` hinein. Eine Scherbe ohne LCOV
   ist der Befund `shard <i> wrote no coverage report`.
2. In der Lane ist `<dir>` = `{coverdata}`. Der Platzhalter bleibt ein Pfad
   ohne neue Bedeutung; dass er hier ein Verzeichnis wird, ist Sache des
   Befehls. `check linecov --report {coverdata}` liest jede Datei darin
   (linecov 1.2) und summiert die Scherben (linecov 3). Frische entsteht über
   die Run-ID im Pfad (linecov 9.1).
3. `CleanCover` (`internal/verify/cover.go`) entfernt Einträge mit
   `os.RemoveAll` statt `os.Remove`, damit ein Verzeichnis eines früheren
   Laufs nicht liegen bleibt und der Aufräumlauf nicht scheitert.

### 6. Klassen-Cache und `loomux godot rescan`

1. **Erkennung.** Deklariert: jede Zeile `class_name <Name>` in einer `.gd`
   unter `--root` (ohne `addons/`, `.godot/`, andere Checkouts). Im Cache:
   `"class": &"<Name>"` bzw. `"class": "<Name>"` in
   `.godot/global_script_class_cache.cfg`.
   - Datei fehlt: `.godot/global_script_class_cache.cfg does not exist; run
     loomux godot rescan` (mit `--root`, wenn die Area nicht `.` ist).
   - Namen fehlen: `class_name not in the class cache: <Name> (<pfad>), …;
     run loomux godot rescan`.
   - Addons werden nicht verglichen: ein Addon-Update ohne Scan bleibt
     unerkannt (wie in space, `godot_quality.py`).
2. **`loomux godot rescan [--root <dir>] [--godot <bin>]`.** Neuer
   Befehlsbaum `godot` unter `internal/cli`.
   1. Binary wie `{godot}` finden und prüfen (Teil 2, 3.2–3.3).
   2. `project.godot` als Bytes lesen.
   3. `<godot> --path <root> --headless --import` fahren (falls nicht
      belegbar, `--editor --quit-after <n>`; siehe „Zu belegen“).
   4. `project.godot` byteweise zurückschreiben, wenn es sich geändert hat,
      und das als Hinweis melden.
   5. Weitere getrackte Dateien unter `--root`, die sich geändert haben
      (`git status --porcelain -- <root>`, vor und nach dem Lauf verglichen),
      als Hinweis nennen, nicht zurücksetzen.
   6. Exit 0, wenn der Lauf gelang und der Cache danach existiert, sonst 1
      mit der Ausgabe der Engine.
3. **`import_check`.** Die Notiz einer `unready`-Lane („run the Godot editor
   once to import the project“) nennt künftig `loomux godot rescan`.

### 7. Urteil je Scherbe

Gelesen wird der neueste `results.xml` unter `<shard>/reports/` (nach mtime,
bei Gleichstand nach der Nummer von `report_<n>`). Der Exit-Code der Engine
zählt nur, wenn der Report nichts Falsches zeigt.

| Fall | Befund |
|---|---|
| kein Report | `shard <i> wrote no test report; it ran <n> suites` |
| Report unlesbar | `shard <i> wrote an unreadable test report <pfad>: <fehler>` |
| keine Fälle im Report | `shard <i> ran no test cases; none of its <n> suites loaded` |
| fehlschlagender Fall | `<suite-pfad> <testfall>: <erste Zeile der Meldung>` je Fall |
| geplante Suite fehlt im Report | `shard <i> did not run: <suite>, …` |
| Exit ≠ 0 bei sauberem Report | `shard <i> exited with <code> while its report shows nothing wrong:` und die Ausgabe der Engine |

Befunde sortiert nach Scherbe, dann nach Suite und Fall. Anders als space
nennt godot-suite jeden fehlschlagenden Fall mit Namen und Meldung, damit ein
Agent nicht in den Report schauen muss.

### 8. Presets

In `internal/verify/presets.toml`; der Kommentar „No test lane“ fällt weg.

```toml
# gdUnit4 decides by the report, never by its exit code; see the cli reference.
[stack.gdscript.test]
commands   = ["{loomux} check godot-suite --godot {godot}"]
measuring  = "{loomux} check godot-suite --godot {godot} --coverage {coverdata}"
lock       = true
when_found = ["plugin.cfg:gdUnit4"]

# The lane's own command does not name {godot}: a commit that skips `test`
# needs no binary for this lane either.
[stack.gdscript.coverage]
commands   = ["{loomux} check linecov --report {coverdata}"]
measure    = "{loomux} check godot-suite --godot {godot} --coverage {coverdata}"
after      = "test"
lock       = true
when_found = ["plugin.cfg:gdUnit4", "plugin.cfg:NanoCoverage"]
```

1. **`when_found`**, neuer Lane-Schlüssel in der Tabellenform: Muster in der
   Form von `tests_when` (Verzeichnis mit `/`, `datei:text`, Glob auf den
   Namen), gesucht unter der Area mit dem Lauf von `HasTests`. **Jedes**
   Muster muss gefunden werden. Fehlt eines, ist die Lane `unavailable` mit
   der Notiz `none found: <muster>`. Eine leere Liste ist ein Ladefehler.
   Geprüft zur Planzeit vor `needs` und `{godot}`, damit ein Projekt ohne
   Addon kein Binary braucht. Kein Preset eines anderen Stacks setzt ihn.
2. **Nachtrag 5 aus Teil 2.** Die Coverage-Lane nennt `{godot}` nur in
   `measure`. Ein Commit, der `test` per `skip_when_only` überspringt, braucht
   für `coverage` darum kein Binary. Für Lanes, die ein Projekt selbst so
   schreibt, dass ihre eigenen Befehle `{godot}` nennen, bleibt der Punkt
   offen (siehe „Grenzen“).
3. **`lock`** an beiden Lanes, unabhängig von der Art der Instrumentierung:
   auch zwei Läufe im Speicher teilen sich `.godot/loomux/` und die
   Laufzeitdatei.
4. Die Lanes haben kein `on_file`: die Suite ist kein Werkzeug für eine
   Datei, und im Edit-Scope laufen `test` und `coverage` nicht.

### 9. Aufbau

```
internal/verify/godotsuite/
  suite.go      Run: preflight, plan, glue, shards, verdict, timings
  discover.go   suites, class_name, autoload, hook conflict
  shards.go     LPT, timings file
  junit.go      results.xml → cases, failures
  disk.go       hash/mtime snapshot, self-healing restore
  gd/run.gd  gd/hook.gd  gd/disk.gd
internal/cli/check.go   checkGodotSuite, modelled on checkLinecov
internal/cli/godot.go   godot rescan
internal/verify/schema.go, plan.go   when_found
internal/verify/cover.go             RemoveAll
```

Prozessstart, Dateisystem-Zeit und die Engine-Antworten gehen als
eingesetzte Funktionen hinein, wie bei `GodotFor`.

## Abgrenzung

- Keine C#-Tests: `runtest.cmd` baut bei einem Mono-Binary mit
  `dotnet build`; godot-suite tut das nicht. Ein Projekt mit C#-Suites ist
  ein eigener Schritt.
- Keine Wiederholung flackernder Fälle.
- Keine Regeln aus Teil 4 (Testmodul je Quelle, Grenze, ungeprüfte Ordner,
  Messreste als Commit-Regel).
- Kein fester Debug-Port; Port 0 wie im Gate von space.

## Grenzen

- Der Kleber greift in die Hook-Liste von gdUnit4 ein, ohne deren
  öffentliche Registrierung. Eine neue gdUnit4-Version kann das brechen; der
  Lauf scheitert dann laut (Entscheidung 3) und nennt die Version.
- macOS: `user://` liegt unter `~/Library/Application Support`; weder
  `APPDATA` noch `XDG_DATA_HOME` trennen es dort. Scherben teilen sich dort
  `user://`. Ein Hinweis auf stderr nennt das, wenn mehr als eine Scherbe
  läuft.
- Nachtrag 5 aus Teil 2 bleibt für selbst geschriebene Lanes offen.
- Die Sperre schützt nicht vor einem Godot-Editor, der dieselben Quellen
  während einer Platten-Instrumentierung öffnet.

## Folgen für space (Teil 6)

| space heute | danach |
|---|---|
| `suite_shards.py` | `check godot-suite` |
| `harness/coverage_shard_hook.gd`, `disk_instrumentation.gd` | eingebetteter Kleber; Eintrag in `hooks/session_hooks` entfernen |
| `toolchain.godot_binary` | `{godot}` mit `GODOT_BIN` |
| `take_suite_lock` | `lock` |
| Klassen-Cache-Befund, Editorbefehl | Entscheidung 6, `loomux godot rescan` |
| `session_hook_findings` | Hook-Konflikt (2.1) |
| `leftover_instrumentation` | Selbstheilung (4.3); als Commit-Regel Teil 4 |
| `SPACE_COVERAGE_SHARD` | `LOOMUX_COVERAGE_SHARD_DIR` |

## Tests

Jede neue Testzeile läuft vor dem Code rot; jede Regel bekommt die
Mutationsrunde mit einem Kontrollmutanten. 100 % je Funktion.

- **Erkennung:** Suites mit und ohne `extends GdUnitTestSuite`, in
  `addons/`, `.godot/`, anderem Checkout; keine Suite.
- **Scherben:** LPT mit, ohne und mit teilweisen Zeiten; Anzahl bei weniger
  Suites als Kernen; kaputte Laufzeitdatei (Hinweis, Mittelwert 1).
- **Urteil:** jede Zeile der Tabelle in Entscheidung 7 mit echten
  `results.xml` aus space als Fixtures; neuester Report bei mtime-Gleichstand.
- **Vorprüfungen:** gdUnit4 fehlt; Cache fehlt, Name fehlt, `&"…"` und
  `"…"`; Hook-Konflikt mit Nanos Hook und mit einem eigenen Coverage-Hook;
  kein Konflikt ohne `--coverage`.
- **Platte:** Autoload erkannt, Autoload in `ignore_paths`, Autoload unter
  `addons/` bei `ignore_addons`; Selbstheilung; Restore scheitert; Hash
  abweichend; mtime nach Restore gleich der vorherigen.
- **Ablage:** `<dir>/<i>.info` je Scherbe; Scherbe ohne LCOV;
  `CleanCover` mit Verzeichnis.
- **`when_found`:** ein Muster fehlt, alle da, leere Liste beim Laden; Lane
  ohne Addon fragt kein Binary.
- **Presets:** `--show` für ein Godot-Projekt mit beiden Addons; Kette
  `test → coverage` mit `measuring`; Doku-Commit hinter übersprungenem `test`
  ohne Binary bleibt grün.
- **`rescan`:** `project.godot` wird zurückgeschrieben; unverändert bleibt
  unberührt; weitere geänderte Datei als Hinweis; Engine scheitert.
- **Integration**, nur mit `GODOT_BIN`: ein Fixture-Projekt unter
  `testdata/godot/` mit gdUnit4, einer grünen und einer roten Suite; in CI
  übersprungen, weil dort keine Engine liegt.
- **Abnahme gegen space** (Plan, nicht CI): Urteil und Laufzeit gegen
  `suite_shards.py`, kalt und warm, nach `docs/{en,de}/benchmarks.md`.

## Doku und Roadmap im selben PR

- `docs/{en,de}/cli-reference.md`: `check godot-suite`, `godot rescan`, die
  Befunde, die Exit-Code-Regel von gdUnit4.
- `docs/{en,de}/configuration.md`: die GDScript-Presets, `when_found`; der
  Absatz „A project names its own test command“ fällt weg.
- `docs/{en,de}/benchmarks.md`: die Abnahme gegen space.
- `README.md`, `README.de.md`: die Roadmap-Zeile „Godot test suite“ fällt
  weg, die Fähigkeit steht in der Liste der Prüfungen; die Zeile für Teil 6
  hängt danach nur noch an Teil 4.
- Label `release:minor` (neue Befehle, neuer Schlüssel, neue Presets).

## Zu belegen vor dem Plan

Mit Godot 4.7.1 und den Addons aus space, in einer Kopie des Projekts
(nie im Checkout von space):

1. Lädt `godot -s res://.godot/loomux/run.gd` ein Skript aus `.godot/`, und
   erbt es von `GdUnitCmdTool.gd`?
2. Läuft ein in die Hook-Liste eingehängter Hook (`startup`, `shutdown`), und
   bleibt `project.godot` dabei unverändert?
3. Instrumentiert Nano Coverage `.godot/loomux/*.gd` nicht (sonst die
   Absturzlage aus space, „Bad address index“)?
4. Schreibt `--headless --import` `.godot/global_script_class_cache.cfg` mit
   neuen `class_name`, oder braucht es `--editor --quit-after <n>`?
5. Setzt `restore_from_disk` die Quellen byte-gleich zurück?

## Nachtrag 1 (2026-10-15): Belege

Geprüft mit Godot 4.7.1 mono (`--headless`, `timeout 300`), gdUnit4 6.2.0 und
Nano Coverage 1.0 in einer Kopie von `space/godot` (`.godot/` mitkopiert, kein
`--import` vorab; `override.cfg` blieb erhalten). Der Nachtrag ändert die
Entscheidungen 2, 3, 4, 6 und 7 wie unten je Beleg und in „Korrekturen“
beschrieben (Entscheidung 5 gilt unverändert); die Aufgaben
des Plans folgen ihm, wo er der Spec widerspricht.

### Beleg 0 (Zusatz): Engine-Binary

Das Fenster-Exe (`Godot_v4.7.1-stable_mono_win64.exe`) schreibt unter Windows
nichts nach stdout/stderr; die Konsolen-Exe (`..._console.exe` im selben
Verzeichnis) schon. space' `godot4-headless.cmd` ruft das Fenster-Exe. Die
Proben liefen mit der Konsolen-Exe, `DOTNET_ROOT` und `PATH` auf das
mitgelieferte dotnet. `PATH`, `DOTNET_ROOT` und alles Weitere dieser Art sind
die Umgebung des Anwenders (oder das Binary, das `{godot}` nennt, etwa ein
Wrapper); godot-suite setzt davon nichts.

Folge: Wo die Binary-Suche (Teil 2, 3.2) unter Windows ein `Godot_*.exe`
findet, bevorzugt godot-suite das Geschwister `*_console.exe`, wenn es
existiert. Sonst fehlt die Ausgabe der Scherbe im Befund. Entscheidung 2
ändert sich nicht; der Punkt gehört in die Binary-Suche (Task 3).

### Beleg 1: `.godot/loomux/…` lädt und erbt von `GdUnitCmdTool.gd`

Befehl (Arbeitsverzeichnis die Kopie, `$RD` ein Berichtsverzeichnis):

    godot --path . -s -d --remote-debug tcp://127.0.0.1:0 res://.godot/loomux/run_GdUnitCmdTool.gd -a res://test/smoke_test.gd -c --headless --ignoreHeadlessMode -rd $RD

Ergebnis: Das Skript lädt aus `.godot/`, und `extends
"res://addons/gdUnit4/bin/GdUnitCmdTool.gd"` greift. **Aber:** gdUnit4s
`CmdArgumentParser.parse` überspringt alle Argumente bis zum ersten, das die
Zeichenfolge `GdUnitCmdTool.gd` enthält. Mit dem Pfad
`res://.godot/loomux/run.gd` findet er sie nie, liest keine Option und druckt
die Hilfe (Exit 0, kein Test läuft).

Folge (Entscheidung 3): Die eingebettete Datei heißt **`run_GdUnitCmdTool.gd`**
(der Name muss `GdUnitCmdTool.gd` enthalten); überall, wo die Spec `run.gd`
schreibt, gilt dieser Name. Der Befehl der Scherbe (Entscheidung 2) ist der
oben stehende. Port 0 erzeugt zwei harmlose `ERROR:`-Zeilen des
Remote-Debuggers und verhindert die `debug>`-Schleife bei Parsefehlern (wie
in space). Exit 0 bei grün, 100 bei fehlgeschlagenen Tests (Beleg 6).

### Beleg 2: Hook einhängen, `project.godot` bleibt unverändert

Die Zeile der Spec (`service.enigne_hooks = hook`) **parst nicht**: der Setter
nimmt den Typ der Eigenschaft, und `Value of type "ProbeHook" cannot be
assigned to a variable of type "Array[GdUnitTestSessionHook]"` bricht das
Skript ab. Richtig ist:

    var service := GdUnitTestSessionHookService.instance()
    var hook: GdUnitTestSessionHook = load("res://.godot/loomux/hook.gd").new()
    hook.set_meta("enabled", true)
    service.enigne_hooks.append(hook)  # no register(): it saves project.godot
    super._initialize()

Ergebnis mit dieser Form: `LOOMUX_PROBE_STARTUP` vor den Tests,
`LOOMUX_PROBE_SHUTDOWN` danach (Shutdown läuft in umgekehrter Reihenfolge),
Bericht unter `-rd`, `git status --porcelain project.godot` leer. Ein in
`project.godot` eingetragener Hook (space' `NanoCoverageShard`) wird von
`instance()` weiter geladen und steht vor dem angehängten. `is_enabled` liest
die Meta `enabled`; fehlt sie, ist der Hook an.

Folge (Entscheidung 3): `service.enigne_hooks.append(hook)` statt der
Zuweisung; der Hook ist eine eigene Datei (`hook.gd`, per `load` geholt) statt
einer inneren Klasse. Der Dienst, die Liste `enigne_hooks` (sic, so
geschrieben in 6.2.0) und `instance()` sind die Abhängigkeit, die die
Versionsmeldung der Spec absichert.

### Beleg 3: Nano Coverage instrumentiert `.godot/loomux/*.gd`: ja

Hook wie space' `coverage_shard_hook.gd` unter `.godot/loomux/hook.gd`,
Eintrag `hooks/session_hooks` aus `project.godot` entfernt, `ignore_paths`
unverändert. Nano zählt 203 statt 202 Skripte, instrumentiert `hook.gd` im
Speicher, und der Lauf scheitert mit

    ERROR: Condition ' (ip + 5) > _code_size ' is true. Breaking..
    SCRIPT ERROR: Internal script error! Opcode: 1 (please report).
    Hook call 'startup' failed with error: 'Result is null! ...'

also die Absturzlage aus space. Die Annahme der Spec („`.godot/` wird nicht
instrumentiert“) ist **falsch**.

Gegenmittel, belegt: `run_GdUnitCmdTool.gd` hängt vor `super._initialize()`
das Muster `**/.godot/loomux/**` an `nano_coverage/general/ignore_paths` an,
nur im Speicher:

    var key := "nano_coverage/general/ignore_paths"
    var ignored: PackedStringArray = ProjectSettings.get_setting(key, PackedStringArray())
    ignored.append("**/.godot/loomux/**")
    ProjectSettings.set_setting(key, ignored)

Danach zählt Nano 201 Skripte, kein Fehler, Exit 0, LCOV mit 189 `SF:`-Blöcken
(keiner unter `.godot/`). `set_setting` speichert nichts, `project.godot`
bleibt unverändert, und die Vorgabe des Projekts (`**/*_test.gd`, …) bleibt,
weil an die gelesene Liste angehängt wird.

Folge (Entscheidung 3, Absatz zu `.godot/`): Der Satz „wird von Nano Coverage
nicht instrumentiert“ entfällt; stattdessen hängt `run_GdUnitCmdTool.gd` das
Muster an, bevor der Hook startet. Ohne es ist ein Hook unter `.godot/` nicht
lauffähig. Der `nano`-Dialekt aus linecov (Entscheidung 4.1) bleibt
unberührt: das Muster steht nur im Lauf, nicht in der Projektkonfiguration.

### Beleg 3 (Zusatz): absolute Ablage außerhalb von `res://`

Der Hook setzt `nano_coverage/general/data_store_dir` und `report_dir` per
`ProjectSettings.set_setting` auf `$DIR/data` und `$DIR/report`, mit `$DIR`
aus `LOOMUX_COVERAGE_SHARD_DIR` (absolut, außerhalb des Projekts). Nano nimmt
sie an und legt dort `data/coverage.meta`,
`data/<workspace>/runs/session_data.covdata` und `report/lcov.info` an, nichts
unter `res://`. `generate_coverage_report` liefert `{ "status": "ok",
"report_path": "<dir>/report" }`; `report_path` ist das **Verzeichnis**, die
Datei heißt `lcov.info`. Die `SF:`-Pfade im LCOV sind relativ zur
Projektwurzel (`core/…`), weder `res://` noch absolut.

Folge: Entscheidung 5 gilt unverändert, die Ausweichlösung unter
`res://.godot/loomux/…` ist nicht nötig. Das LCOV einer Scherbe liegt unter
`<LOOMUX_COVERAGE_SHARD_DIR>/report/lcov.info`. Fixture:
`testdata/lcov/shard.info`, vier Blöcke eines echten Laufs
(`core/animation_throttle.gd`, `core/game_clock.gd`,
`ui/panels/time_controls.gd` mit Treffern, `ui/starmap/orbit_camera.gd` mit
Trefferzahl 0); es kommen nur `TN:`, `SF:`, `DA:`, `LF:`, `LH:` und
`end_of_record` vor, keine `FN`- oder `BRDA`-Zeilen.

### Beleg 4: `--headless --import` schreibt den Klassen-Cache

Befehl: `godot --path . --headless --import`, nachdem `core/loomux_probe.gd`
mit `class_name LoomuxProbeClass` angelegt war.

Ergebnis: Exit 0; `grep -c LoomuxProbeClass
.godot/global_script_class_cache.cfg` springt von 0 auf 1 (`"class":
&"LoomuxProbeClass",`). `--editor --quit-after` ist nicht nötig:
**`--headless --import`** ist der kleinste Befehl, Entscheidung 6.2.3 gilt wie
geschrieben. Die Ausgabe ist nach der Editor-Sprache lokalisiert (hier
Deutsch): nur Exit-Code und Cache auswerten, keinen Text.

Nebenwirkungen (Material für Task 11):

- `project.godot` wird umgeschrieben (rund 30 Zeilen neu, 65 entfernt):
  alle Kommentare fallen weg, Abschnitte werden umsortiert (`[autoload]`
  hinter `[application]`; `[display]`, `renderer/…` und `textures/…` fehlen
  danach), die `[input]`-Ereignisse stehen mit allen Standardfeldern, und
  `[gdunit4] hooks/session_hooks` bekommt zusätzlich den Eintrag des Addons
  (`res://addons/nano_coverage_godot/integrations/gdunit_hook.gd`), den der
  Plugin-Start des Editors registriert. Byteweises Zurückschreiben
  (6.2.2/6.2.4) ist nötig und genügt.
- Getrackte Dateien ändern sich: `ui/theme/**/*.import` (15 Dateien) und
  `.godot/` (`editor/`, `imported/`, der Cache).
- Neue, ungetrackte Dateien entstehen: `*.gd.uid` für Skripte ohne UID
  (`core/loomux_probe.gd.uid`, `harness/*.uid`, `test/**/*.uid`) und
  `reports/report_*/css/logo.png.import`.

Folge (6.2.5): Der Hinweis auf weitere geänderte Dateien trifft in der Praxis
fast immer zu. Er nennt auch neue ungetrackte Dateien (`??` in
`git status --porcelain -- <root>`) und vermerkt, dass `*.uid` und `*.import`
Erzeugnisse der Engine sind. Zurückgesetzt wird nur `project.godot`.

### Beleg 5: Restore ist byte-gleich

SHA-256 aller `*.gd`, `*.uid` und `project.godot` außerhalb von `.godot/`
(1305 Dateien) vor und nach:

    godot --path . --headless -s res://harness/disk_instrumentation.gd -- instrument
    godot --path . --headless -s res://harness/disk_instrumentation.gd -- restore

`instrument` liefert `{"failed_count":0,"ignored_count":10,
"instrumented_count":191,"status":"ok"}`; danach tragen die Quellen die
Markerzeile `# __NANO_COVERAGE_INSTRUMENTED__` (Hashes weichen ab). `restore`
liefert `{"restored_count":191,"status":"ok","warnings":[]}`, und der
Hash-Vergleich gegen den Stand vor `instrument` ist **leer**: alle Quellen sind
byte-gleich, auch bei Zeilenenden.

Folge (4.4): Der Hash-Vergleich braucht **keine** Normalisierung der
Zeilenenden. Nebenwirkung: Der Prozess schreibt beim Beenden („Flushing
execution hits“) nach `coverage-data/default/runs/session_data.covdata` im
konfigurierten Datenverzeichnis des Projekts, auch mit
`save_project_settings: false`. Das sind keine Quellen und kein Befund nach
4.4; der Test in Task 11 zählt `coverage-data/` nicht.

### Beleg 6: Berichte, Fixtures und `user://`

**Berichtsverzeichnis.** `-rd <dir>` legt `<dir>/report_<n>/results.xml`
(JUnit) und `index.html` an; `<n>` zählt hoch (1 bei leerem `<dir>`). Wer
`<dir>` je Scherbe frisch anlegt, liest `report_1/results.xml`. `-rd` nahm
einen absoluten Pfad an, dessen Ordner (auch zwei Ebenen tief) vorher nicht
existierte; die Engine legt ihn an. Ein relativer Pfad ist nicht geprüft:
godot-suite übergibt immer einen absoluten.

**`user://` und `APPDATA`.** space legt das Benutzerverzeichnis über
`override.cfg` fest (`use_custom_user_dir=true`,
`custom_user_dir_name="Godot/app_userdata/Kontari-space"`). Neben dem Exe
liegt außerdem die Datei `._sc_` (selbstenthaltender Modus: `editor_data`
neben dem Exe); sie betrifft Editor-Daten, nicht `user://`, und „selbstenthalten“
beschreibt nur diese Editor-Daten, nicht das Benutzerverzeichnis.
Ein `-s`-Skript, das `OS.get_user_data_dir()` druckt:

    APPDATA unverändert:  C:/Users/micro/AppData/Roaming/Godot/app_userdata/Kontari-space
    APPDATA=<frisch>:     <frisch>/Godot/app_userdata/Kontari-space

`APPDATA` verschiebt `user://` also, auch mit eigenem `custom_user_dir_name`
(der Name hängt unter `APPDATA`). Godot 4.7.1 hat keinen Schalter für das
Benutzerverzeichnis (`--help` geprüft). Folge: godot-suite setzt im
Prozessumfeld jeder Scherbe `APPDATA` auf ein frisches Verzeichnis (unter
Unix `XDG_DATA_HOME`); das trennt die Prozesse beim `user://`-Zugriff. Belegt
ist nur die `APPDATA`-Hälfte (der Lauf war Windows-only); die
`XDG_DATA_HOME`-Hälfte ist ungeprüft und folgt der Godot-Dokumentation.
Editor-Einstellungen liegen bei der Exe (`editor_data`) und hängen nicht
davon ab.

**Fixtures `results.xml`** (echte Läufe, unverändert):

- `green.xml`: zwei Suiten, 9 Fälle, grün; Exit 0.
- `failing.xml`: ein Fall besteht, einer schlägt fehl: `<failure
  message="FAILED: res://test/lx/failing_test.gd:9" type="FAILURE">` mit
  `CDATA`-Text; Exit 100.
- `errors.xml`: Laufzeitfehler in einem Fall: `<error message="ERROR:
  res://test/lxrt/runtime_error_test.gd:10" type="ABORT">`, `errors="1"` an
  der `testsuite`; Exit 100. `errors` steht nur an `testsuite`, nicht an
  `testsuites` (dort `tests`, `failures`, `skipped`, `flaky`).
- **`empty.xml` entfällt.** Ohne Fälle schreibt gdUnit4 weder `results.xml`
  noch `report_<n>`: `No test cases found, abort test run!`, Exit 0. Das gilt
  für ein leeres Verzeichnis, eine Suite ohne `test_*`-Funktion, ein `-i`,
  das alles ausschließt, und einen Lauf ohne `-a`. Folge (Entscheidung 7): Eine
  Scherbe mit Exit 0 ohne `results.xml` ist ein Befund (`shard <i> wrote no
  test report`), kein Erfolg. Der Test braucht keine Fixture.
- **Parsefehler.** Eine Suite mit Parsefehler allein: Exit 105, keine
  `results.xml`, nur `No test cases found`. In derselben Scherbe mit gesunden
  Suiten (mehrere `-a`) **stürzt der Prozess ab** (Exit 139; nur gesehen bei
  aktivem Nano-Hook, ohne Hook nicht geprüft), und die Ergebnisse der ganzen Scherbe fehlen. Folge: Exit
  ≠ 0 ohne `results.xml` ist der Befund „Scherbe abgestürzt“ und trägt die
  Prozessausgabe. Eine `errors`-Fixture für Parsefehler gibt es nicht, weil
  gdUnit4 sie nicht als `errors` zählt.

### Zusammenfassung

1. Die Datei heißt `run_GdUnitCmdTool.gd`, nicht `run.gd` (Beleg 1).
2. Der Hook hängt per `service.enigne_hooks.append(hook)` ein, nicht per
   Zuweisung (Beleg 2).
3. `run_GdUnitCmdTool.gd` hängt `**/.godot/loomux/**` an `ignore_paths` an
   (im Speicher); „Nano lässt `.godot/` aus“ gilt nicht (Beleg 3).
4. Die absolute Ablage funktioniert; LCOV in `<dir>/report/lcov.info`, `SF:`
   relativ zur Wurzel (Beleg 3, Zusatz).
5. `--headless --import` genügt; `project.godot`, `.import`, `.uid` ändern
   sich (Beleg 4).
6. Restore ist byte-gleich ohne Normalisierung (Beleg 5).
7. `APPDATA` je Scherbe verschiebt `user://`; kein CLI-Schalter (Beleg 6).
8. Leerer Lauf: Exit 0 ohne `results.xml` ist ein Befund; Parsefehler neben
   gesunden Suiten: Absturz, Exit 139 (Beleg 6).
9. Unter Windows die Konsolen-Exe nehmen (Beleg 0).

### Korrekturen

**Überholte Stellen.** Diese Stellen im Text oben im Dokument gelten nicht
mehr:

- Entscheidung 2, Befehlszeile (`… res://.godot/loomux/run.gd`) und
  Entscheidung 3, Aufzählung („**`run.gd`** erbt von …“): der Name ist
  `run_GdUnitCmdTool.gd`.
- Abschnitt 9 (Aufbau), Zeile `gd/run.gd  gd/hook.gd  gd/disk.gd`:
  `gd/run_GdUnitCmdTool.gd  gd/hook.gd  gd/disk.gd`. Die eingebettete
  Quelldatei trägt denselben Namen wie die geschriebene Datei unter
  `<root>/.godot/loomux/`: `internal/verify/godotsuite/gd/run_GdUnitCmdTool.gd`.
- Entscheidung 3, Einleitung: „… wird von Nano Coverage als Teil des Projekts
  nicht instrumentiert (zu belegen …)“ ist falsch; Nano instrumentiert
  `.godot/loomux/*.gd`, solange `run_GdUnitCmdTool.gd` (und `disk.gd`, siehe
  unten) das Muster `**/.godot/loomux/**` nicht an `ignore_paths` hängt. „Das
  Projekt braucht keine eigene Zeile“ bleibt richtig.
- Entscheidung 3, `run.gd`-Absatz: „hängt … eine Instanz von `hook.gd` an die
  Hooks … an“ heißt konkret `service.enigne_hooks.append(hook)`; die Zuweisung
  (Setter) ist nicht gemeint.
- Entscheidung 3, `hook.gd`-Absatz: bei Platten-Instrumentierung wird
  `coverage.meta` aus der **in `project.godot` konfigurierten** Ablage kopiert
  (K4 unten), nicht aus einem festen Pfad.

**K1. `ignore_paths` beim Anhängen.** Mit entferntem Schlüssel
`[nano_coverage] general/ignore_paths` in `project.godot` liefert
`ProjectSettings.get_setting("nano_coverage/general/ignore_paths")` in
`run_GdUnitCmdTool.gd` die Vorgabe von Nano, `["**/*_test.gd"]`; nach dem
Anhängen steht `["**/*_test.gd", "**/.godot/loomux/**"]`. Nano zählt dann 202
Skripte, patcht 190, Exit 0, LCOV ok. Die Vorgaben bleiben also erhalten, und
die Rückfallregel (Liste `["**/*_test.gd"]`, wenn der Schlüssel fehlt) ist
**nicht nötig**: „bestehende Einstellung lesen, Muster anhängen“ genügt.
`disk.gd` braucht dieselben Zeilen: ohne sie instrumentiert
`instrument_to_disk` auch `.godot/loomux/hook.gd` und `run_GdUnitCmdTool.gd`
(Markerzeile gefunden; 194 statt 192 Skripte), das Restore stellt sie wieder
her. Mit dem Muster bleiben beide unberührt.

**K2. Entscheidung 7, Tabelle (Reihenfolge der Prüfung).**

| # | Fall | Befund |
|---|---|---|
| 1 | kein Report (gleich welcher Exit-Code; deckt leeren Lauf, Parsefehler, Absturz ab) | `shard <i> wrote no test report (exit <code>); it ran <n> suites:` und die Ausgabe der Engine |
| 2 | Report unlesbar | `shard <i> wrote an unreadable test report <pfad>: <fehler>` |
| 3 | keine Fälle im Report | `shard <i> ran no test cases; none of its <n> suites loaded` |
| 4 | fehlschlagender Fall (`failure` oder `error`) | `<suite-pfad> <testfall>: <Meldung>` je Fall |
| 5 | geplante Suite fehlt im Report | `shard <i> did not run: <suite>, …` |
| 6 | Exit ≠ 0 bei sauberem Report | `shard <i> exited with <code> while its report shows nothing wrong:` und die Ausgabe der Engine |

Die Zeile „kein Report“ ersetzt die der Spec (`… wrote no test report; it ran
<n> suites`). Die Zeile 3 („keine Fälle im Report“) bleibt, ist aber nur durch
einen Report mit null `testcase`-Elementen erreichbar; **keine Probe hat einen
solchen Report erzeugt** (ein Lauf ohne Fälle schreibt gar keinen, Beleg 6).
Sie ist Vorsorge für eine andere gdUnit4-Version. Die `ERROR:`-Zeilen des
Remote-Debuggers (Port 0) werden in der Ausgabe von Zeile 1 und 6 **nicht
gefiltert**.

**K3. Meldungstext aus `results.xml`.** Der Text eines fehlschlagenden Falls ist
die erste nicht leere Zeile der Zeichendaten des Elements (`<failure>` und
`<error>` zählen gleich). Nur wenn die Zeichendaten leer sind, gilt das
Attribut `message`. Grund: `message` enthält nur die Fundstelle
(`testdata/junit/failing.xml:7`: `message="FAILED: res://test/lx/failing_test.gd:9"`,
`testdata/junit/errors.xml:7`: `message="ERROR: res://test/lxrt/runtime_error_test.gd:10"`),
die Meldung steht im `CDATA`-Block (`failing.xml:9`: `Expecting:`;
`errors.xml:9`: `Godot Runtime Error !`). Die Fundstelle steht außerdem im
`at '…'`-Teil des Blocks; wer sie braucht, nimmt zusätzlich das Attribut.

**K4. Platten-Modus zusammen mit dem Scherben-Hook.** Ablauf in der
Kopie (Projekt ohne `coverage-data/`): `disk.gd -- instrument`, eine Scherbe
(`run_GdUnitCmdTool.gd`, `-a res://test/autoload/game_test.gd`, absolutes
`LOOMUX_COVERAGE_SHARD_DIR`), `disk.gd -- restore`.

- Nach `instrument` liegt `coverage.meta` in der konfigurierten Ablage:
  `<root>/coverage-data/coverage.meta` (zusätzlich `coverage-data/default/runs/`);
  `coverage-report/` entsteht nicht. Der Hook liest die Ablage vor dem
  Überschreiben mit `ProjectSettings.get_setting("nano_coverage/general/data_store_dir")`
  (hier `res://coverage-data`; Vorgabe von Nano bei fehlendem Schlüssel).
- Der Hook kopiert aus `<konfigurierte Ablage>/coverage.meta` (also
  `res://coverage-data/coverage.meta`) nach `<shard>/data/coverage.meta`
  (`DirAccess.copy_absolute` lieferte 0), nachdem er die Ablage der Scherbe auf
  `<shard>/data` gesetzt und `clear_coverage_data` gerufen hat.
  `DiskInstrumenter.new().is_instrumented()` wählt den Zweig;
  `instrument_all_scripts()` entfällt dann.
- Ergebnis: Exit 0, 35 Fälle grün, `report/lcov.info` liegt in der Scherbe, und
  `SF:autoload/game.gd` hat Treffer (`LH:75`, z. B. `DA:87,30`): der Autoload
  ist gemessen. Eine Zeile `ERROR: Parse JSON failed. Error at line 0:
  Expected key` kam einmal in der Ausgabe, ohne Folge für Exit und Report.
- `restore` meldet `restored_count 192`, die Markerzeile ist weg.

**K5. E4.2, Ergänzung: `coverage-data/`.** `instrument_to_disk` legt
`<root>/coverage-data/` an, wenn es fehlt, und `restore` entfernt es nicht.
godot-suite löscht `<root>/coverage-data/` nach dem Restore nur dann, wenn das
Verzeichnis vor dem Lauf nicht existierte; sonst bleibt es. Die Prüfung
geschieht vor `disk.gd -- instrument`. Unverändert gilt: E4.2 hasht nur `.gd`
unter `--root` (nicht `.uid` und nicht `project.godot`); die Hashprobe in
Beleg 5 war weiter gefasst, als die Spec es verlangt.

## Nachtrag 2 (2026-10-16): Entscheidungen der Abschlussprüfung

Die Prüfung des ganzen Zweigs hat drei Befunde ergeben; die Entscheidungen
ändern die Spec an drei Stellen.

**F1. `.godot/` gehört nicht zum Quellbaum von `check linecov`.** Das Profil
Nano hat `scope = "all"`; der Baumlauf von `ScanTree` fand die Glue-Dateien
`.godot/loomux/*.gd` und meldete sie in jedem Godot-Projekt als „absent from
the report“. `ScanTree` überspringt jetzt ein Verzeichnis namens `.godot` in
jeder Tiefe, wie `.git`: Es ist Godots Importcache und der Ort, an den
`check godot-suite` seine Glue schreibt; keine Quelle eines Werkzeugs liegt
dort.

**F2. Eigene Befehle in `[verify.gdscript.test]` erben `when_found` nicht.**
`when_found` schützt die Befehle des Presets. Setzt ein Projekt in der
Tabellenform `commands` und nicht `when_found`, löscht `merge` das
`when_found` des Presets; setzt es beides, gilt seine Liste. Ohne diese
Regel wäre die Lane in einem GUT-Projekt still nicht verfügbar. Die Semantik
von `measuring` bleibt wie für alle Stacks: Wer eigene Befehle schreibt,
setzt auch `measuring` oder nutzt die String-Form.

**F3. E2.1 gilt in beiden Läufen.** Die Spec sah den Hook-Konflikt nur mit
`--coverage` vor. Der Editor-Plugin von Nano trägt seinen gdUnit4-Hook aber
wieder ein (Beleg 4: `integrations/gdunit4.gd`, `enable()` ergänzt
`hooks/session_hooks`, solange `nano_coverage/integrations/gdunit4` wahr ist
und der Editor startet). Ein solcher Hook instrumentiert auch im einfachen
Lauf im Speicher und lässt jede Scherbe gleichzeitig `res://coverage-data`
und `res://coverage-report` in den Arbeitsbaum schreiben. Die Prüfung läuft
deshalb immer; der Befund nennt die dauerhafte Abhilfe:
`project.godot registers a coverage session hook, <res-Pfad>; remove it from
[gdunit4] hooks/session_hooks and set [nano_coverage] integrations/gdunit4=false,
or the Godot editor registers it again`.
