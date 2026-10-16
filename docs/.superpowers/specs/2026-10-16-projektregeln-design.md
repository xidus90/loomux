# `loomux check rules`: Projektregeln als Prüfungen

Stand 2026-10-16, Basis `origin/master` 2d6b8597 (v1.6.0), Zweig
`feat/check-rules`.

## Anlass

Teil 4 der Zerlegung aus `2026-10-06-gate-skip-lock-godot-design.md`, Zeile
„Projektregeln als Prüfungen“ der Roadmap. Teil 1 (`check linecov`), Teil 2
(`skip_when_only`, `lock`, `{godot}`) und Teil 3 (`check godot-suite`, die
GDScript-Presets) sind auf master. Von `space/.claude/hooks/godot_quality.py`
bleiben danach vier Projektregeln, die nur das Dateisystem lesen:

| Funktion in space | Regel |
|---|---|
| `missing_test_modules`, `pairing_findings` | ein Testmodul je Quelle |
| `leftover_instrumentation` | kein Rest der Instrumentierung von Nano Coverage im Commit |
| `unchecked_sources` | keine Quelle außerhalb jedes geprüften Verzeichnisses |
| `core_boundary_violations` | der Kern erreicht die UI nicht (Basisklasse, `res://ui/`, `class_name` aus `ui/`, Autoload) |

Diese Spec macht daraus einen eingebauten Befehl, den jedes Projekt über
seine `.loomux/config.toml` einstellt. GDScript ist der erste Nutzer, die
Regeln selbst kennen keine Sprache.

## Was die Recherche in space ergeben hat

Git-Log und Wiki von space, gelesen am 2026-10-16:

- **Die Kerngrenze ist die Regel mit Gewicht.** Am 2026-08-09 rief
  `model/game_state.gd` `MarketLines.has_market` auf, eine Klasse aus
  `ui/panels/`, über ihren `class_name` und ohne `res://ui/`. Das ging durch
  acht Tasks und sieben Reviews.
- **Die eigentliche Regel ist eine Kantentabelle.** `docs/wiki/architecture/
  modulschnitt-fundament.md` führt je Verzeichnis (und für einzelne Dateien
  wie `model/game_state.gd`, `data/upgrade_catalog.gd`), welche anderen es
  kennen darf, mit fünf begründeten Gegenkanten. Geprüft wird davon heute
  nur „kein `ui/`, kein Autoload“. Die Kante `core/ → data/` stand lange im
  Code, bevor sie bemerkt wurde (Commit 78735aa4: „no check has ever
  reported it“).
- **Quellen ohne Rolle:** `harness/probe_driver.gd` lag in einem ungeprüften
  Ordner und wäre bei einem Umbau still zerbrochen (Wiki-Log). Die
  Gegenseite: Am 2026-08-02 erzeugte ein Worktree unter `.claude/` 71 KB
  Fehlalarme je Hook-Lauf, weil die Prüfung die Platte statt den Git-Index
  ablief.
- **Instrumentierung:** Auf dem gdUnit4-Pfad instrumentiert Nano Coverage
  im Speicher („152 Stichproben, null veränderte Quellen“). Auf der Platte
  instrumentiert nur noch `check godot-suite` die Autoloads, und es heilt
  sich selbst (Teil 3, 4.3). Die Regel ist eine Rückfallebene für einen
  abgebrochenen Lauf.

## Ziel und Erfolg

- space kann in Teil 6b den Rest von `godot_quality.py` löschen (Abgleich
  unter „Folgen für space“).
- Die ganze Kantentabelle von space ist geprüft, nicht nur die Kerngrenze.
  Die Kerngrenze ist eine Zeile davon.
- Eine Kante freizuschalten ist eine Entscheidung eines Menschen: Sie steht
  in `.loomux/config.toml`, die kein Agent schreibt.
- Ein Projekt ohne die neue Tabelle merkt nichts: keine Lane, kein Hinweis.

## Leitlinien

1. **Mehrsprachig.** Die Regeln arbeiten auf Dateien, Bereichen und Kanten.
   Was sprachabhängig ist (wie eine Kante entsteht, wo ein Test liegt, wie
   ein Marker aussieht), kommt aus dem Graphen und aus Werkzeugprofilen in
   Daten.
2. **Erkennen statt konfigurieren.** Konfiguriert wird nur, was nur der
   Mensch wissen kann: welche Bereiche es gibt und wer wen kennen darf. Wo
   ein Test liegt, sagt das Testwerkzeug. Woran Instrumentierung zu erkennen
   ist, sagt das Coverage-Profil.
3. **Die Dateien der Werkzeuge sind die Quelle.** Der Testordner von gdUnit4
   kommt aus `project.godot`, nicht aus einer zweiten Angabe in loomux.
4. **Laut statt still.** Eine Karte, die nicht stimmt, ist ein Fehler. Eine
   Regel, die nicht geprüft werden kann, nennt den Grund.

## Entscheidungen

### 1. Befehl

```
loomux check rules [--cached] [--root <dir>]
```

- Ohne `--cached` liest er den Arbeitsbaum, mit `--cached` den Index (den
  Inhalt, der committet wird).
- Welche Dateien es gibt, sagt immer Git: `git ls-files` (mit `--cached` der
  Index, ohne die verfolgten und die nicht ignorierten unverfolgten Dateien
  des Arbeitsbaums). Ignoriertes, `.godot/`, Worktrees unter `.claude/` und
  `.worktrees/` fallen damit ohne Liste weg.
- Exit 0 ohne Befund, 1 mit Befunden, 2 bei einer kaputten Karte oder einem
  Fehler beim Lesen. Befunde auf stdout, je Zeile einer, sortiert nach Pfad
  und Zeile.
- Fehlt `[rules]` in der Konfiguration, endet er mit 0 und einem Hinweis auf
  stderr.

### 2. Die Bereichskarte in `.loomux/config.toml`

```toml
[rules.layers.core]
paths   = ["godot/core/**", "godot/model/**"]
may_use = ["core", "data"]
bases   = ["RefCounted", "Resource", "Object"]
tests   = true

[rules.layers.game_state]
paths   = ["godot/model/game_state.gd"]
may_use = ["core", "data", "procgen"]
tests   = true

[rules.layers.ui]
paths   = ["godot/ui/**"]
may_use = ["core", "data", "services", "persistence", "autoload"]

[rules.layers.harness]
paths   = ["godot/harness/**"]
checked = false

[rules.layers.addons]
paths   = ["godot/addons/**"]
checked = false
```

1. **Ein Bereich** hat einen Namen und `paths`, Globs relativ zur
   Repowurzel im Dialekt von `[[policy.paths.rules]]` (`*` in einem Segment,
   `**` über Segmente).
2. **Zuordnung:** Jede Datei gehört höchstens einem Bereich. Treffen mehrere
   Globs, gewinnt der spezifischste: ein Glob ohne Platzhalter vor einem mit,
   sonst der mit den meisten festen Segmenten vor dem ersten Platzhalter. Zwei
   gleich spezifische Globs aus verschiedenen Bereichen auf derselben Datei
   sind ein Kartenfehler. So ist eine Ausnahme ein eigener Bereich für eine
   Datei, wie `game_state`.
3. **`may_use`:** die Bereiche, auf die Kanten aus diesem Bereich zeigen
   dürfen. Kanten in den eigenen Bereich sind immer erlaubt. Fehlt der
   Schlüssel, ist jede Kante erlaubt (für Bereiche wie `test`, die alles
   kennen). `may_use = []` erlaubt nur den eigenen Bereich.
4. **`bases`:** Basisklassen, die nirgends im Projekt aufgelöst werden (die
   der Engine oder der Sprache), und die eine Datei des Bereichs erweitern
   darf. Fehlt der Schlüssel, ist jede solche Basis erlaubt. Eine Basis, die
   im Projekt aufgelöst wird, ist eine Kante und fällt unter `may_use`.
5. **`tests = true`:** Jede Quelle des Bereichs braucht ihr Testmodul
   (Entscheidung 5). Vorgabe `false`.
6. **`checked = false`:** Der Bereich ist bekannt, aber keine Regel prüft
   seine Dateien: keine Kanten aus ihm, keine Paarung, kein „ohne Bereich“.
   Kanten **in** ihn hinein prüft `may_use` der Quelle weiterhin; `harness`
   muss also in `may_use` stehen, wer es benutzen darf. `checked = false` mit
   `may_use`, `bases` oder `tests` ist ein Kartenfehler.
7. **Keine Ausnahme im Code.** Es gibt keinen Marker, mit dem eine Zeile eine
   Kante freischaltet. Eine Freigabe geht nur über die Karte und damit über
   `loomux config set … --propose` und einen Menschen (Entscheidung der
   Nutzerin bzw. des Nutzers, 2026-10-16).
8. **Kartenfehler** (Exit 2, alle auf einmal gemeldet): ein unbekannter
   Bereich in `may_use`; ein Glob, der keine Datei trifft; ein leerer Glob
   oder ein leeres `paths`; Gleichstand nach Punkt 2; ein unbekannter
   Schlüssel; Punkt 6. Der Bereichsname folgt den Regeln für Schlüssel in
   `config.toml`.

**Nachtrag 2026-10-16 (Spezifitätsordnung zu 2.2, wie in
`internal/verify/rules/layers.go` umgesetzt und in `configuration.md`
dokumentiert):** Der Glob-Dialekt ist `pathkey.Glob`, derselbe wie bei
`[[policy.paths.rules]]`: führendes `**/`, abschließendes `/**`, sonst
`path.Match`; ein Muster ohne Schrägstrich trifft den Basisnamen in jeder
Tiefe, ein `**` in der Mitte ist nichts Besonderes. Treffen mehrere Globs eine
Datei, gilt in dieser Reihenfolge: (1) ein an der Wurzel verankerter Glob
schlägt einen, der in jeder Tiefe trifft (führendes `**/` oder kein
Schrägstrich); (2) ein Glob ohne Platzhalter (`*`, `?`, `[`) schlägt einen mit;
(3) mehr feste Segmente vor dem ersten Segment mit Platzhalter schlagen
weniger. Gleichstand in allen dreien bei Globs verschiedener Bereiche auf
derselben Datei ist der Kartenfehler aus Punkt 8.

### 3. Kanten

1. Der Graph wird im Prozess gebaut, aus den Dateien nach Entscheidung 1,
   mit denselben Extraktoren und Resolvern wie `loomux graph`
   (`internal/code/extract`, `resolve.Graph`). Ein gespeicherter Graph wird
   nicht gelesen. Damit läuft die Regel auch in CI und braucht keinen
   frischen Index. Mit `--cached` werden die Inhalte aus dem Index gelesen.
2. Es zählen alle aufgelösten Kanten außer `contains`: `extends`, Importe
   (`preload`/`load` mit Literal, Go- und Python-Importe), Verweise auf
   `class_name` und Autoloads, Aufrufe. Autoloads löst der Graph über
   `project.godot` auf eine Datei auf; sie fallen damit in den Bereich ihrer
   Datei (in space `autoload`). Kommentare ergeben keine Kanten.
3. **Quellen einer Kante** sind nur Dateien mit `.go`, `.py`, `.gd`.
   **Ziele** dürfen jede Datei des Graphen sein, auch `.tscn`, `.tres` oder
   eine Datei einer anderen Sprache: Ein `preload("res://ui/x.tscn")` im Kern
   ist eine Verletzung, wenn `ui/**` ein Bereich ist.
4. **Befund:** eine Kante von einer Datei in Bereich A (geprüft) auf eine
   Datei in Bereich B ≠ A, wenn B nicht in `may_use` von A steht. Ein Ziel
   ohne Bereich ist kein Befund dieser Regel; ist es eine Quelle, meldet es
   Regel 4. Ein Befund je Quelldatei und Zieldatei, an der ersten Zeile, an
   der die Quelle das Ziel benutzt.
5. **Unaufgelöste Basen** (meine Entscheidung nach dem Builder-Check):
   `resolve` legt die `extends` offen, die es nicht auflöst, und sagt
   warum. Heute verwirft es sie still (`resolveGDScript`, `resolvePython`).
   - **Kein Kandidat** (eine Klasse der Engine oder der Sprache, etwa
     `Node`, `object`): erlaubt, wenn `bases` fehlt oder den Namen enthält,
     sonst ein Befund. Gilt auch für innere Klassen (`class X extends Node`)
     und für jede einzelne Basis einer Python-Klasse.
   - **Mehrere Kandidaten** (ein doppelter `class_name`): immer ein Befund
     „ambiguous base“, unabhängig von `bases`. Sonst ginge eine Kante still
     durch.
   - Ein `extends` auf einen Pfad, der nicht auflöst, ist kein Befund dieser
     Regel.
6. **Befundtext:** `<datei>:<zeile> <layer A> uses <layer B> (<symbol>) —
   not in may_use of <A>; allow it with: loomux config set
   rules.layers.<A>.may_use … --propose`.
7. **Zeile je Kante** (Entscheidung der Nutzerin bzw. des Nutzers,
   2026-10-16): Die Extraktoren für Go, Python und GDScript bekommen eine
   Zeile je Rohkante (`extract.RawEdge`), die Zeile der Verwendung, nicht
   der Definition. Heute kennt eine Kante nur den Umfang ihres
   Quellknotens; beim Vorfall vom 2026-08-09 wäre das der Anfang der
   Methode, bei einem Verweis auf Skriptebene Zeile 1. Folgen, die in den
   Plan gehören: die Version jedes Extraktors steigt, der Extrakt-Cache wird
   ungültig, und ob `model.Edge` und das Schema von `wiring.json` die Zeile
   mitnehmen, entscheidet der Bauende. Wo der Resolver Kanten je (Quelle,
   Ziel, Relation) zusammenfasst, bleibt die kleinste Zeile.
6. **Bewusst nicht erfasst:** ein `res://`-Pfad in einem beliebigen
   String, der an keinem `preload`/`load`/`extends` steht (etwa
   `change_scene_to_file("res://ui/x.tscn")`). space meldet heute jedes
   `res://ui/` textuell. Der Graph erfasst das nicht; der Fall ist in space
   nach Lage der Kantentabelle nicht im Kern belegt. Wird er gebraucht, ist
   das eine Erweiterung des GDScript-Extraktors, nicht dieser Regel.

### 4. Quellen ohne Bereich

Eine Quelle (`.go`, `.py`, `.gd`; Szenen, Ressourcen und `project.godot`
zählen nicht, obwohl der GDScript-Extraktor sie liest), die in keinem
Bereich liegt, ist ein Befund:
`<datei> is in no layer; add it to a layer in [rules.layers], or to one with
checked = false`. Testdateien zählen mit; space braucht also einen Bereich
`test`.

### 5. Testpaarung

1. Für jede Datei eines Bereichs mit `tests = true`, die selbst kein Test
   ist, muss die erwartete Testdatei im Dateibestand nach Entscheidung 1
   stehen.
2. Wo der Test liegt, sagt ein **Testprofil** in Daten
   (`internal/verify/rules/tests.toml` o. ä., Ablage dem Bauenden
   überlassen), gewählt nach Endung der Quelle und erkannt am Projekt:
   - **gdUnit4** (`.gd`, aktiv bei `plugin.cfg:gdUnit4`): Wurzel ist das
     Verzeichnis des nächsten `project.godot`. Testordner aus
     `[gdunit4] settings/test/test_lookup_folder` in `project.godot`,
     Vorgabe `test`; leer heißt neben der Quelle. Name nach
     `test_suite_naming_convention`: `foo.gd → foo_test.gd`,
     `Foo.gd → FooTest.gd` (Vorgabe AUTO_DETECT). Abbildung exakt wie
     `GdUnitTestSuiteScanner.resolve_test_suite_path` des vendored gdUnit4
     (in space 6.2.0), auch wo sie überrascht: Das abschließende
     `.replace(file_name, suite_name)` läuft über den ganzen Pfad, also
     `ui/main_menu/main_menu.gd → test/ui/main_menu_test/main_menu_test.gd`.
     Das Werkzeug ist die Quelle (meine Entscheidung); der Unterschied zur
     Spiegelung von space trifft dort nur Bereiche ohne `tests`. Eine Datei
     ist ein Test, wenn sie unter dem Testordner liegt.
   - **Go** (`.go`): `foo.go → foo_test.go` im selben Verzeichnis; eine Datei
     auf `_test.go` ist ein Test.
   - Für Python gibt es kein Profil, weil pytest keine Abbildung von Quelle
     auf Testdatei kennt (nur `python_files`, Vorgabe `test_*.py`,
     `*_test.py`). Ein Bereich mit `tests = true` und `.py`-Quellen ist ein
     Kartenfehler, der das fehlende Profil nennt. Ein Profil nachzutragen ist
     ein Dateneintrag plus Fixture.
3. Die Recherche dazu (gdUnit4 v6.2.2 `GdUnitSettings.gd`,
   `GdUnitTestSuiteScanner.gd`; Go `cmd/go`; pytest `reference.rst`) geht mit
   ihren URLs nach `docs/{en,de}/cli-reference.md`. Vor dem Plan gegen das
   gdUnit4 aus `space/godot/addons/` zu belegen: die Abbildung für
   `addons/` und für `/src/`, und wie die Namenskonvention in
   `project.godot` steht (Ganzzahl).
4. **Befund:** `<quelle> has no test module; expected <pfad>`.

**Nachtrag 2026-10-16 (Belege zu 5.2 und 5.3, gegen das vendored gdUnit4 6.2.0
in `space/godot/addons/gdUnit4/`):**

- Schlüssel (`GdUnitSettings.gd:19-20`): `gdunit4/settings/test/test_lookup_folder`
  (Vorgabe `DEFAULT_TEST_LOOKUP_FOLDER = "test"`, Z. 109) und
  `gdunit4/settings/test/test_suite_naming_convention` (Vorgabe
  `NAMING_CONVENTIONS.AUTO_DETECT`, Z. 129). Die Enum (Z. 114-118) zählt
  `AUTO_DETECT = 0`, `SNAKE_CASE = 1`, `PASCAL_CASE = 2`; in `project.godot`
  steht sie als Ganzzahl in der Sektion `[gdunit4]` als
  `settings/test/test_suite_naming_convention=<0|1|2>`, der Ordner als
  `settings/test/test_lookup_folder="<ordner>"`. `get_setting` (Z. 203)
  liefert die Vorgabe, wenn der Schlüssel fehlt; das `project.godot` von
  `space` setzt beide nicht (nur `hooks/session_hooks`), es gilt also
  `test` und AUTO_DETECT. Ein leerer Ordner oder `/` heißt „neben der
  Quelle“ (`resolve_test_suite_path`, Z. 293).
- Namensbildung (`_to_naming_convention`, `GdUnitTestSuiteScanner.gd:275`):
  AUTO_DETECT nimmt `to_snake_case(name + "Test")`, wenn `is_snake_case(name)`
  (kein Großbuchstabe außer `_`), sonst `to_pascal_case(name + "Test")`
  (`value.capitalize().replace(" ", "")`). `to_snake_case` setzt vor jedem
  Großbuchstaben ein `_`, aber erst ab dem zweiten bereits geschriebenen
  Zeichen (`result.size() > 1`, `GdObjects.gd:329`).
- Pfadbildung (`resolve_test_suite_path`, Z. 290-317), für `res://`-Quellen
  ohne `/src/`: `split("/", false)` verwirft leere Teile, `paths[0]` ist
  `res:`. Liegt `/src/` im Pfad, wird es durch `/<ordner>/` ersetzt. Ist
  `paths[1] == "addons"`: `res://addons/<paths[2]>/<ordner>/<Rest>`. Sonst
  `res://<ordner>/<paths[1..]>`. Zuletzt `normalize_path` und
  `.replace(file_name, suite_name)` über den **ganzen** Pfad.

| Quelle | Zweig | Zwischenpfad | Testdatei |
|---|---|---|---|
| `res://core/seeded_rng.gd` | kein `/src/`, `paths[1]=core` | `res://test/core/seeded_rng.gd` | `res://test/core/seeded_rng_test.gd` |
| `res://ui/main_menu/main_menu.gd` | kein `/src/`, `paths[1]=ui` | `res://test/ui/main_menu/main_menu.gd` | `res://test/ui/main_menu_test/main_menu_test.gd` |
| `res://addons/p/x/foo.gd` | `paths[1]=addons` | `res://addons/p/test/x/foo.gd` | `res://addons/p/test/x/foo_test.gd` |

Die Zeile 2 bestätigt die Überraschung aus 5.2: `main_menu` kommt im Ordner-
und im Dateinamen vor, beide werden ersetzt. Die Tabelle ist von Hand
nachgerechnet; der Bauende macht sie zu Fixtures.

### 6. Rest der Instrumentierung

1. Keine Konfiguration. Das Coverage-Profil in
   `internal/verify/linecov/tools.toml` bekommt einen Schlüssel für die
   Zeile, die das Werkzeug beim Instrumentieren auf der Platte als erste
   Zeile schreibt. Für `nanocoverage`:
   `# __NANO_COVERAGE_INSTRUMENTED__` (Konstante `MARKER` in
   `src/instrumentation/disk_instrumenter.cpp`).
2. Aktiv, wenn `linecov.Activate` das Profil am Projekt erkennt (für Nano
   Coverage die Sektion `[nano_coverage]` in `project.godot` oder ein
   Bericht), unabhängig von `[rules.layers]`. Aber nur
   mit `[rules]` in der Konfiguration, damit ein Projekt ohne Tabelle nichts
   merkt (Ziel).
3. Geprüft werden alle Dateien mit den Endungen des Profils, auch in
   Bereichen mit `checked = false` (space instrumentiert `autoload/` und
   prüft dort keine Kanten). Ein Befund nur, wenn die erste Zeile nach Trim
   gleich dem Marker ist.
4. Nur mit `--cached`, also beim Commit: Nach einem abgebrochenen Lauf soll
   die Bearbeitung nicht bei jeder Datei warnen, aber nichts davon soll
   committet werden. Befund: `<datei> still holds <tool> instrumentation;
   restore the sources before committing`. Diese Befunde kommen vor allen
   anderen, weil eine instrumentierte Datei nicht die geschriebene ist.

### 7. Lane und Presets

1. Neue Lane-Art `rules`. Sie gehört wie `graph` zur Wurzel: höchstens ein
   Job je Lauf, getragen vom ranghöchsten aktiven Stack mit einem
   `rules`-Befehl (die Trägerlogik der Graph-Lane, auf beide Arten
   verallgemeinert).
2. **Ohne `[rules]` wird die Lane nicht geplant** (meine Entscheidung nach
   dem Builder-Check). `when_found` reicht dafür nicht: Es vergleicht nur
   Basisnamen, überspringt Punktverzeichnisse wie `.loomux/`, und ein
   fehlender Treffer erscheint in `--show` als „none found“. Die Planung
   liest stattdessen aus der wirksamen Konfiguration, ob `[rules]` steht;
   fehlt es, gibt es weder Zeile noch Hinweis.
3. Presets in `presets.toml` für `go`, `python` und `gdscript`:
   ```toml
   [stack.gdscript.rules]
   commands = ["{loomux} check rules --cached"]
   on_file  = ["{loomux} check rules"]
   ```
   Wie `--cached` in einem Lauf ohne Commit (`check precommit` mit leerem
   Index, CI) zu verstehen ist, entscheidet der Bauende nach dem Muster der
   Graph-Lane (`blast-audit --cached`).
4. **Profile** (meine Entscheidung): `rules` kommt in die
   Vorgabeprofile `edit`, `precommit` und `stop` (`verify/schema.go`). Ein
   Projekt mit eigenen `profiles` nimmt es selbst auf; `configuration.md`
   sagt das.
5. **Bei einer Bearbeitung** plant die Graph-Lane heute nichts, und eine
   Datei ohne Stack ergibt keine Ziele. `rules` braucht darum eine eigene
   Regel: Eine Bearbeitung einer Quelle (`.go`, `.py`, `.gd`) oder von
   `.loomux/config.toml` plant einen `rules`-Job an der Wurzel, einmal je
   Bearbeitung, unabhängig vom Stack der Datei.
6. Nach einer Bearbeitung warnt die Lane (post-edit blockiert nie). Beim
   Commit blockiert sie. Das ist das Verhalten von space heute (`--warn` und
   `--precommit`).

**Nachtrag 2026-10-16 (Beleg zu 7.3, `--cached` ohne Gestagtes):**

- `check blast-audit --cached` selbst liest den Index gegen HEAD
  (`internal/code/query/blast.go:70`, `changedFiles(root, ["--cached"], "HEAD")`)
  und fällt bei leerem Index nicht auf den letzten Commit zurück (Z. 89 gilt
  nur ohne `--cached`); ein leerer Index ergibt keine Dateien und keinen
  Befund. Es ist die **Planung**, die ihn gar nicht erst startet:
  `query.GraphReady` (`internal/code/query/refresh.go:74`) prüft erst
  `GraphPrereq` (Graph da, HEAD da, keine Merge-/Rebase-Marker, Z. 44) und
  dann `git diff --cached --quiet`; Exit 0 heißt `nothing staged`, die Lane
  wird `not applicable` mit dieser Notiz (`internal/verify/plan.go:380-388`).
  Das gilt für `check precommit` mit leerem Index und für CI, wo nichts
  gestagt ist. `checkBlastAudit` (`internal/cli/check.go:136`) kennt diese
  Bedingung nicht.
- `check stop` ersetzt die Probe (`check.go:234-240`): `hooks.OpenStopIndex`
  (`internal/hooks/stopgraph.go:25`) legt im Git-Verzeichnis eine **Kopie des
  echten Index** ab (`gitwork.KeptContentTree`, `gitwork.go:269`), darauf
  `git add -A` (alles, was git nicht ignoriert) und `git rm --cached
  .loomux/state`, und schreibt daraus den Baum (`writeContentTree`,
  `gitwork.go:301-327`). Die Lane liest über `GIT_INDEX_FILE` diese Kopie
  (`StopIndex.Env`, `stopgraph.go:71`) und gilt als bereit, wenn Baum und
  HEAD-Baum sich unterscheiden (`StopIndex.Ready`, Z. 52-69); der echte Index
  bleibt unberührt, am Turnende ist er meist leer.
- Folge für `check rules --cached` (**Korrektur** des früheren Wortlauts, der
  die Lane bei nichts Gestagtem als `not applicable: nothing staged` plante):
  `check rules --cached` läuft immer, wenn die Lane geplant ist. Bei nichts
  Gestagtem hält der Index trotzdem jede verfolgte Datei (gleich HEAD), der
  Befehl prüft also den ausgecheckten Baum, auch in CI. Am Turnende (`stop`)
  liest er über `GIT_INDEX_FILE` die Kopie von `OpenStopIndex`.

### 8. Rest von `godot_quality.py`, der nicht Teil 4 ist

Damit Teil 6b die Datei ganz löschen kann, kommt in dieses PR:

- `gdformat --check` ins GDScript-Preset `lint`, neben `gdlint`, mit
  `on_file` für die bearbeitete Datei.
- Kein Lint-Bereich in loomux. space ersetzt `LINTED_DIRS` durch
  `excluded_directories` in seinem `.gdlintrc` und einem neuen `.gdformatrc`
  (beide YAML, Schlüssel seit gdtoolkit 3.2.8 bzw. 4.3.0, Abgleich nach
  Namen in jeder Tiefe, Vorgabe nur `.git`). Das ist Teil 6b und
  Konfiguration in space.

## Folgen für space (Teil 6b)

| space heute | danach |
|---|---|
| `CORE_DIRS`, `TESTED_DIRS`, `KNOWN_DIRS`, `UNCHECKED_DIRS` | `[rules.layers.*]` mit `tests` und `checked` |
| Kantentabelle in `modulschnitt-fundament.md` (ungeprüft) | `may_use` je Bereich; die Wikiseite behält die Begründungen und verweist auf `config.toml` |
| `CORE_BASE_CLASSES` | `bases` |
| `AUTOLOADS`, `_AUTOLOAD_USE` | Graph über `[autoload]` in `project.godot`; Bereich `autoload` |
| `_ui_class_names`, `res://ui/` | Kanten im Graphen (Entscheidung 3.6 für Strings ohne `load`) |
| `missing_test_modules`, `pairing_findings` | Entscheidung 5, Profil gdUnit4 |
| `unchecked_sources`, `_sources_outside` | Entscheidung 4 über `git ls-files` |
| `leftover_instrumentation`, `INSTRUMENTATION_MARKER` | Entscheidung 6 |
| `stale_class_cache`, `_project_scan_command` | Teil 3, `godot rescan` |
| `lint_checks`, `linted_target`, `LINTED_DIRS`, `GDTOOLKIT` | Preset `lint` mit `gdformat`; `excluded_directories` in `.gdlintrc`/`.gdformatrc` |
| `run_checks`, `suite_findings`, Sperre | Teil 3 |
| `INERT_PREFIXES`, `GATE_FILES`, `cannot_affect_gate` | Teil 2, `skip_when_only` |
| `main --warn` / `--precommit` | post-edit / `check precommit` |

## Nicht-Ziele

- Keine Kanten auf Ebene von Symbolen (eine Ausnahme für drei Aufrufe von
  `ResourceCatalog.default()` ist ein Bereich für die Datei oder gar keine).
- Keine Regeln für Szenen und Ressourcen (`.tscn`, `.tres`) als Quellen.
- Kein Marker im Code, kein Schreiben in `config.toml` durch den Befehl.
- Kein Lesen der Wikiseite.

## Tests

Jede neue Testzeile läuft vor dem Code rot; jede Regel bekommt die
Mutationsrunde mit einem Kontrollmutanten. 100 % je Funktion.

- **Karte:** jede Art Kartenfehler aus 2.8; Spezifität mit Datei-Glob, mit
  tieferem Verzeichnis, Gleichstand; `may_use` fehlt, leer, gesetzt.
- **Kanten:** je Relation ein Fall in GDScript, ein Go-Import, ein
  Python-Import; Autoload; `class_name` aus einem anderen Bereich ohne Pfad
  (der Fall vom 2026-08-09); Kommentar ergibt nichts; innere Klasse mit
  verbotener Basis; Basis aufgelöst, ohne Kandidat, mit zwei Kandidaten;
  Python-Klasse mit zwei Basen; `preload` einer `.tscn`; Ziel ohne Bereich
  ergibt nichts; Kante in einen Bereich mit `checked = false`.
- **Zeilen:** je Extraktor die Zeile der Verwendung für jede Relation; zwei
  Verwendungen desselben Ziels ergeben einen Befund an der kleineren Zeile;
  Verweis auf Skriptebene trägt seine Zeile, nicht 1.
- **Dateibestand:** ignorierte Datei, Worktree unter `.claude/`,
  unverfolgte nicht ignorierte Datei ohne `--cached`; mit `--cached` eine
  gestagte Fassung, die vom Arbeitsbaum abweicht.
- **Paarung:** gdUnit4 mit Vorgabe, mit eigenem `test_lookup_folder`, mit
  leerem; PascalCase; Go; `.py` mit `tests = true` als Kartenfehler.
- **Instrumentierung:** Marker in Zeile 1, mit Leerraum, in Zeile 2, nur
  erwähnt; ohne `--cached` nicht geprüft; ohne erkanntes Profil nicht
  geprüft; in `checked = false`.
- **Lane:** `--show` mit `[rules]` und ohne (keine Zeile, kein Hinweis);
  ein Träger bei Go und GDScript zusammen; post-edit einer `.gd`-Datei, von
  `.loomux/config.toml`, einer Datei ohne Stack (kein Job); eigene
  `profiles` ohne `rules`.
- **Abnahme gegen space** (Plan, nicht CI): die Kantentabelle der Wikiseite
  als Karte, Befunde gegen `godot_quality.py` auf demselben Stand, und
  Laufzeit kalt und warm nach `docs/{en,de}/benchmarks.md`.

## Doku und Roadmap im selben PR

- `docs/{en,de}/cli-reference.md`: `check rules`, die Befunde, die
  Testprofile mit Quellen.
- `docs/{en,de}/configuration.md`: `[rules.layers]`, die Lane `rules`,
  `gdformat` im Preset.
- `docs/{en,de}/benchmarks.md`: die Abnahme gegen space.
- `README.md`, `README.de.md`: Die Roadmap-Zeile „Projektregeln als
  Prüfungen“ fällt weg, die Fähigkeit steht in der Liste der Prüfungen; die
  Zeile für Teil 6 hängt an nichts mehr.
- Die deutschen Dateien unter `docs/de/` scheitern an `wiki-sources`, bis ein
  Mensch neu pinnt; sie gehen als Patch in den Notizordner.
- Label `release:minor` (neuer Befehl, neuer Konfigurationsschlüssel, neue
  Lane-Art, Preset-Erweiterung).

## Dem Bauenden überlassen

- Paket und Dateinamen unter `internal/verify/` und die Ablage der
  Testprofile.
- Wie `config set` Listen in `rules.layers.<name>.may_use` vorschlägt, falls
  der heutige Befehl das nicht kann; der Befundtext nennt dann die Form, die
  geht.
- Die genaue Spezifitätsordnung, solange 2.2 gilt und sie dokumentiert ist.

## Nachtrag Abnahme (2026-10-16)

`check rules` gegen die vier Regeln von `space` (`missing_test_modules` über
`pairing_findings`, `leftover_instrumentation`, `unchecked_sources`,
`core_boundary_violations`) auf demselben Stand: `space` an `HEAD`, mit
`git archive` in ein Scratch-Verzeichnis entpackt (ohne `.git` und `.tools`),
dort neu eingecheckt, damit `rules.Files` Git befragen kann. Die Karte ist die
Kantentabelle aus `modulschnitt-fundament.md`, eine Zeile je Bereich, die
Zeilen einzelner Dateien (`game_state.gd`, `generation_params.gd`,
`upgrade_entry.gd`, `upgrade_catalog.gd`) als eigene Bereiche, dazu `test`
(ohne `may_use`), `autoload`, `harness` und `addons` (`checked = false`);
`bases` aus `CORE_BASE_CLASSES`, `tests = true` auf den sechs Ordnern aus
`TESTED_DIRS`. Eine Zeile, die einen Ordner nennt (`model/`, `data/`), erlaubt
alle Bereiche dieses Ordners. Der Vorschlag steht als
`space-rules-proposal.toml` im Notizordner (Teil 6b); zusätzlich hat die
Probe den Bereich `hooks` (`.claude/**`, `checked = false`) gebraucht, weil
sonst 23 `.py`-Dateien der Hooks „ohne Bereich“ wären. Eine Abweichung von
der Tabelle: Der Vorschlag trägt `model` zusätzlich in das `may_use` von
`game_state` ein, das die Tabelle dort nicht nennt; ohne den Eintrag meldete
die Karte jede Kante von `game_state.gd` in die übrigen Dateien von
`godot/model/`. Gelaufen sind die Schritte von
`check rules --cached` aus einem Wegwerf-Test; eine `.loomux/config.toml`
wurde nirgends geschrieben. Die Zeiten stehen in `docs/en/benchmarks.md`.

**Ergebnis.** Die vier Regeln von `space` melden auf der Kopie nichts (die
fünfte Regel von `fast_findings`, der fehlende Klassen-Cache, ist ein
Merkmal der frischen Kopie). `check rules` meldet 76 Befunde: 71 Kanten,
1 Testmodul, 4 Dateien ohne Bereich, 0 Instrumentierung, 0 Kartenfehler.

**1. Zusätzliche Kantenbefunde (71), erwartet.** Das Tor von `space` prüft
nur die Richtung Kern nach `ui`/Autoload; die übrigen Zeilen der Tabelle hat
nichts geprüft. Jetzt zeigen sie, dass die Tabelle dem Code hinterherhinkt,
nicht dass der Code falsch sein müsste:

| Kante | Zahl | Was die Tabelle sagt |
|---|---:|---|
| `ui` nach `core` (`DisplaySettings`, `BodyAppearance`, `CellRelief` …) | 54 | die Zeile `ui/` nennt `core/` nicht |
| `model` nach `data` (`ResourceEntry`, `ResourceCatalog`) | 4 | `model/` nennt nur `model/`, `core/` |
| `data` nach `model` (`Counters`, `RunRules`) | 4 | für `data/` außer drei Dateien steht keine Zeile |
| `ui` nach `procgen` (`UniverseGenerator`) | 2 | `ui/` nennt `procgen/` nicht |
| `game_state` nach `core` (`StartColony`, `MarketPricing`) | 2 | die Zeile nennt `data/`, `procgen/`; die Spec-Karte erlaubt `core` |
| `data` nach `data_generation_params` (`param_catalog`, `param_field`) | 2 | Folge der Bereichskarte: die Datei ist eigener Bereich, die Tabelle kennt keine Kante aus den übrigen `data/`-Dateien |
| `data_upgrade_catalog` nach `data_upgrade_entry` | 1 | Folge der Bereichskarte, dieselbe Art; die Zeile nennt nur `procgen/`, `services/` |
| `services` nach `persistence` (`SaveStore` in `world_build.gd`) | 1 | `services/` nennt `persistence/` nicht |
| `model` nach `procgen` (`GalaxyGenerator` in `galaxy.gd`) | 1 | nur `game_state.gd` darf nach `procgen/` |

(54 + 4 + 4 + 2 + 2 + 2 + 1 + 1 + 1 = 71, verteilt auf 41 Dateien.) Die Entscheidung, ob Tabelle oder Code
nachgezogen wird, ist die der Nutzerin bzw. des Nutzers in Teil 6b; der
Vorschlag lässt die Tabelle wörtlich stehen und zeigt so die Lücke. Die
beiden Befunde „Folge der Bereichskarte“ verschwinden, sobald die übrigen
`data/`-Dateien auf die Dateibereiche verweisen dürfen.

**2. E3.6, Zeichenketten ohne `load`.** Zwei Proben in einer zweiten Kopie
(gestagt, danach verworfen): `Game.something()`, eine Klasse aus `ui/` bei
ihrem Namen und `extends Node` in einer Kerndatei melden beide Seiten auf
derselben Zeile (`check rules` mit den Texten „core uses autoload (…)“,
„core uses ui (MarketLines)“, „core extends Node — not in bases of core“).
Die Zeichenkette `"res://ui/panels/market_lines.gd"` ohne `load` meldet nur
`space` („core module references ui/“); `check rules` sieht sie nicht, wie
E3.6 es festlegt. Zwei weitere Unterschiede, ohne Probe: `space` meldet jede
Zeile, `check rules` je Quelldatei und Zieldatei nur die erste; `space`
meldet je Zeile höchstens einen Autoload, `check rules` eine Kante je
Zieldatei.

**3. gdUnit4-Abbildung (1 Befund).** `godot/core/cie_1931.gd`: erwartet
`godot/test/core/Cie1931Test.gd`, vorhanden ist `cie_1931_test.gd`. Die
Abbildung folgt `GdUnitTestSuiteScanner.resolve_test_suite_path` mit
`GdObjects.is_snake_case` aus dem gevendorten gdUnit4 6.2.0: Ziffern zählen
dort als „nicht klein“, `cie_1931` ist also nicht Snake-Case, die
Auto-Erkennung wählt PascalCase und der Pfad, den gdUnit4 beim *Anlegen* der
Suite vorschlüge, ist `Cie1931Test.gd`. `space` fragt nur nach
`<name>_test.gd` und ist zufrieden. Gefunden wird die Datei von gdUnit4 beim
Ausführen über die Vererbung, nicht über den Namen; der Befund ist also die
getreue Abbildung des Werkzeugs, keine Lücke von `space`. Für 6b: die Datei
umbenennen oder `test_suite_naming_convention` in `project.godot` auf Snake
(1) setzen; beides ist eine Entscheidung der Nutzerin bzw. des Nutzers.

**4. Dateien ohne Bereich (4).** `docs/.superpowers/specs/messungen/*.py`
(`2026-08-27-bahnvergabe`, `2026-08-27-zonen`, `2026-08-28-hoehenfeld`,
`2026-08-28-sonne-im-bild`). `unchecked_sources` sieht nur `.gd` unterhalb
von `godot/`; `check rules` kennt alle drei Quellsprachen im ganzen Repo
(Entscheidung 4); die 4 liegen außerhalb von `godot/` und `.claude/`.
Für 6b: ein Bereich mit `checked = false` für `docs/**`.

**5. Instrumentierung (0).** Beide Seiten melden nichts; die Kopie
enthält keine instrumentierte Datei, ein Unterschied der Regel zeigt sich
hier nicht.

**6. Zeit.** Etwa 1,5 s für die Schritte von `check rules` gegen etwa 0,8 s
für `fast_findings`, kalt wie warm (kein Cache über Prozesse; dort
„Kalt“ und „Warm“ unterscheiden sich um das Rauschen). Das Mehr ist der
Symbolgraph für die Kanten (etwa 1,2 s); Wert und Läufe in
`docs/en/benchmarks.md`.
