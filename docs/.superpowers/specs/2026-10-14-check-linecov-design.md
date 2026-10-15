# `loomux check linecov`: Zeilenabdeckung für jede Sprache

Stand 2026-10-14, Basis `origin/master` 45a3f121 (v1.4.1), Zweig
`feat/check-linecov`.

## Anlass

Teil 1 der Zerlegung aus `2026-10-06-gate-skip-lock-godot-design.md`: space
prüft die Abdeckung seiner GDScript-Hälfte mit `.claude/hooks/lcov_merge.py`
und `coverage_gate.py`. Diese Spec ersetzt beide durch einen eingebauten
Befehl. Die Roadmap nannte ihn `check lcov`. LCOV allein reicht aber nicht:
PHPUnit, JaCoCo, Kover und `go test` schreiben kein LCOV. Der Befehl heißt
darum nach dem, was er prüft, nicht nach einem Format. `coverage` scheidet
aus, weil es ein Kind-Name ist (`internal/verify/schema.go`, `Reserved`).

## Ziel und Erfolg

`loomux check linecov` hält jede ausführbare Zeile eines Projekts an einer
Regel: gelaufen, oder mit Begründung ausgenommen. Das gilt für jede Sprache,
deren Werkzeug LCOV, Cobertura, JaCoCo-XML oder ein Go-Profil schreibt.
Erfolg heißt:

- space kann in Teil 6 beide Skripte löschen, ohne eine Regel zu verlieren
  (Abgleich unter „Folgen für space“).
- Ein Projekt mit einer der Werkzeugdateien aus Entscheidung 6 braucht keine
  Zeile Konfiguration und kein Flag.
- Ein weiteres Werkzeug ist ein Eintrag in einer Datendatei plus Fixtures,
  kein Go-Code, solange seine Einstellungsdatei eine der bekannten Syntaxen
  hat.

## Leitlinien

1. **Mehrsprachig.** Die Regeln kennen keine Sprache. Was sprachabhängig ist
   (Berichtsformat, Einstellungsdatei, Marker, Endungen), steht in Lesern und
   Profilen, nie in den Regeln.
2. **Erkennen statt konfigurieren.** Was sich aus dem Projekt ablesen lässt,
   wird abgelesen: Format, Berichtsorte, Werkzeug, Bereich, Marker,
   Quellwurzeln. Der Befehl hat zwei Flags, beide optional.
3. **Die Dateien der Werkzeuge sind die Quelle.** Bereich und Marker kommen
   aus deren eigenen Einstellungsdateien, nicht aus einer zweiten Kopie in
   `.loomux/config.toml`.
4. **Erweiterbar in Daten.** Drei Registries, je eine für Berichtsformate,
   Syntaxen von Einstellungsdateien und Werkzeugprofile. Neue Einträge
   erzwingen ihre Fixtures über einen Test.
5. **Laut statt still.** Was nicht gelesen werden kann, ist ein Befund. Was
   nicht geprüft werden kann, nennt ein Hinweis auf stderr.

## Recherche, die die Entscheidungen trägt

Am 2026-10-14 gegen Dokumentation und Quellcode der Werkzeuge. Die Quellen
gehen mit dem Plan nach `docs/{en,de}/cli-reference.md`, soweit sie dort
Verhalten erklären.

- **Ausgeschlossene Zeilen fehlen im Bericht.** coverage.py nimmt sie aus den
  `statements` (`coverage/parser.py`), gcovr schreibt sie nicht
  (`is_reportable`, `formats/lcov/write.py`), lcov/geninfo verwirft sie
  (`lcovrc(5)`). Ein Gate kann ausgeschlossene Zeilen darum nicht aus dem
  Bericht zurückgewinnen; es muss die Quellen lesen.
- **Nano Coverage schreibt Zeilen mit 0 Treffern** und kennt keine Marker
  (`src/reporting/lcov_writer.cpp`). `SF:` ist projektrelativ, `res://`
  abgeschnitten. Einstellungen stehen in `[nano_coverage]` in
  `project.godot`, darunter `general/ignore_paths` (Vorgabe `**/*_test.gd`)
  und `general/ignore_addons` (Vorgabe `true`).
- **Pfadformen.** Absolut (coverlet-LCOV, geninfo), relativ zum
  Arbeitsverzeichnis oder Root (coverage.py, istanbul, gcovr), relativ zu
  `<sources>` (Cobertura), Paket plus Dateiname ohne Wurzel (JaCoCo, Kover),
  Importpfad (Go-Profil).
- **Statisch lesbar** sind `.coveragerc`, `setup.cfg`, `tox.ini`,
  `pyproject.toml`, `gcovr.cfg`, `.lcovrc`, `phpunit.xml`, `*.runsettings`,
  `.nycrc`, `.c8rc`, `package.json`, `project.godot`. **Nicht** statisch
  lesbar sind `vitest.config.ts`, `nyc.config.js`, Gradle- und Kover-DSL.
- **Schwellen** sind fast überall Summen (`fail_under`, `fail-under-line`,
  `Threshold`); das Werkzeug prüft sie selbst.

## Entscheidungen

### 1. Befehl

```
loomux check linecov [--report <pfad|glob>]… [--root <dir>]
```

1. `--root`: Bezugsverzeichnis für Pfade, Einstellungen und Bereich.
   Vorgabe ist das Arbeitsverzeichnis, in einer Lane also das der Area.
2. `--report`: wiederholbar. Ein Verzeichnis steht für jede reguläre Datei
   darin, ein Glob für jeden Treffer. Ohne `--report` gelten die
   Standardorte aller Profile (Entscheidung 6), relativ zu `--root`; gelesen
   wird jeder, der existiert.
3. Ausgabe wie `check gocover`: ein Befund je Zeile auf stdout, sortiert nach
   Pfad und Zeile, Form `<pfad>:<zeile> <text>` oder `<pfad> <text>`.
   Hinweise auf stderr als `loomux check linecov: note: …`.
4. Exit 0 grün, 1 bei Befunden, 2 bei falschem Aufruf.
5. `linecov` kommt in `Reserved` (`internal/verify/schema.go`), damit kein
   Profil den Namen verdeckt; Abschnitt in `docs/{en,de}/cli-reference.md`.

### 2. Berichtsformate

Ein Interface und eine Registry (Slice) in `internal/verify/linecov`:

```go
type Format interface {
    Name() string
    Detect(head []byte) bool
    Read(content []byte, ctx PathContext) (Report, []Finding)
}
```

Erkennung am Inhalt, nach BOM und Leerraum, in Registry-Reihenfolge. Passt
kein Format, ist die Datei ein Befund (`unknown coverage format`).

| Format | Erkennung | Gelesen | Pfad |
|---|---|---|---|
| Go-Profil | erste Zeile `mode: ` | `datei:sl.sc,el.ec n count`; jede Zeile `sl…el` gehört zum Block | Modulpfad aus `go.mod` unter `--root` durch `--root` ersetzt; außerhalb des Moduls ein Befund |
| JaCoCo-XML (auch Kover) | Wurzelelement `report` | `package@name` + `sourcefile@name`, `line@nr`, Treffer `ci` | Endungssuche unter `--root` (Entscheidung 4) |
| Cobertura | Wurzelelement `coverage` | `class@filename`, `line@number`, `line@hits` | absolut, sonst je `<source>` die erste existierende Verbindung, sonst relativ zu `--root` |
| LCOV | eine Zeile beginnt mit `TN:` oder `SF:` | `SF:`, `DA:<zeile>,<treffer>[,<prüfsumme>]` positionsweise; `FN`, `BRDA`, `LF`, `LH` übergangen | wie Cobertura ohne `<source>` |

- Ein Go-Block markiert jede Zeile von `sl` bis `el`. Eine Zeile gilt als
  getroffen, wenn irgendein Block über ihr getroffen ist; ihre Trefferzahl
  ist die größte der Blöcke über ihr.
- Ein unlesbarer Datensatz ist ein Befund mit Datei und Text des Datensatzes;
  ein `DA:` vor dem ersten `SF:` gehört zu keiner Datei und wird übergangen.
  Kaputtes XML ist ein Befund für die ganze Datei.
- Ungültiges UTF-8 wird ersetzt, nicht abgelehnt.

### 3. Zusammenführen

Alle Berichte, auch verschiedener Formate, ergeben einen `Report`
(`map[pfad]map[zeile]treffer`). Dateien: Vereinigung, nie Schnitt (ein
Schnitt wäre eine still schrumpfende Basis). Treffer: Summe je Datei und
Zeile, gesättigt bei `MaxUint64`. Ein Zusammenführungsbericht wird nicht
geschrieben.

### 4. Pfade

Jeder Pfad wird zu einem Pfad mit `/` relativ zu `--root`:

1. `\` wird `/`; `res://` wird abgeschnitten.
2. Ein absoluter Pfad wird relativ gemacht; unter Windows ohne Rücksicht auf
   Groß-/Kleinschreibung beim Abschneiden des Roots.
3. Ein Pfad **außerhalb** von `--root` (C++-Systemheader, Abhängigkeiten)
   gehört nicht zum Projekt: übergangen, gezählt in einem Hinweis.
4. Ein Pfad in einem **anderen Checkout** unter `--root` (ein Verzeichnis mit
   einem `.git`-Eintrag zwischen Root und Datei, etwa
   `.claude/worktrees/<name>`) wird übergangen. Das ersetzt
   `IGNORED_PREFIXES` aus space ohne Liste.
5. **JaCoCo-Endungssuche:** `com/x/Foo.java` wird als Pfadendung unter
   `--root` gesucht, einmal je Lauf über einen Baumdurchlauf, der andere
   Checkouts überspringt. Ein Treffer gilt; keiner ist der Befund „nicht auf
   der Platte“; mehrere sind ein Befund `ambiguous source`, der die Kandidaten
   nennt.

### 5. Einstellungsdateien: Leser je Syntax

Eine Registry von Syntax-Lesern. Jeder liest aus einer Datei die Werte zu
Schlüsseln, die ein Profil benennt:

| Syntax | Für | Umsetzung |
|---|---|---|
| `ini` | `.coveragerc`, `setup.cfg`, `tox.ini`, `gcovr.cfg`, `.lcovrc` | eigener kleiner Leser: Abschnitte optional, `=`/`:`, mehrzeilige Werte und Komma-Listen wie configparser, `#`/`;`-Kommentare |
| `toml` | `pyproject.toml`, `.coveragerc.toml` | `github.com/BurntSushi/toml` (vorhanden) |
| `xml` | `phpunit.xml`, `*.runsettings` | `encoding/xml`, einfache Pfade `a/b/c` und `@attr` |
| `json` | `.nycrc`, `.c8rc`, `package.json` | `encoding/json`, Punktpfade |
| `godotcfg` | `project.godot` | Abschnitt und Schlüssel wie der Leser für `{godot}` (`internal/verify/godot.go`), dazu `PackedStringArray(…)`, Strings, Booleans |

Eine Einstellungsdatei, die ihr Leser nicht versteht, ist ein Befund, kein
stilles „keine Einstellung“. Eine neue Syntax (etwa YAML, `yaml.v3` ist
vorhanden) ist ein Leser und ein Registry-Eintrag.

### 6. Werkzeugprofile: `tools.toml`

Eingebettet in `internal/verify/linecov/tools.toml`, geparst beim ersten
Gebrauch wie `presets.toml`, nie beim Start. Ein Eintrag je Werkzeug:

```toml
[tool.coveragepy]
extensions = [".py"]
reports    = ["coverage.xml", "coverage.lcov", "lcov.info"]
settings   = [  # coverage.py's own lookup order; the first file holding the section wins
  { file = ".coveragerc",      syntax = "ini",  section = "run" },
  { file = ".coveragerc.toml", syntax = "toml", table = "run" },
  { file = "setup.cfg",        syntax = "ini",  section = "coverage:run" },
  { file = "tox.ini",          syntax = "ini",  section = "coverage:run" },
  { file = "pyproject.toml",   syntax = "toml", table = "tool.coverage.run" },
]
include  = { dirs = ["source", "source_dirs"], globs = ["include"] }
exclude  = { globs = ["omit"] }
patterns = "glob"
markers  = [ { pattern = 'pragma:\s*no\s*cover' } ]

[tool.nanocoverage]
extensions = [".gd"]
reports    = ["coverage-report/lcov.info"]
settings   = [ { file = "project.godot", syntax = "godotcfg", section = "nano_coverage" } ]
scope      = "all"   # no include key: every file with the extensions is expected
exclude    = { globs = ["general/ignore_paths"], defaults = ["**/*_test.gd"] }
switches   = [ { key = "general/ignore_addons", default = true, exclude = ["addons/**"] } ]
patterns   = "glob"
```

Felder:

- `extensions`: welche Dateien das Profil erwartet und auf Marker liest.
- `reports`: Standardorte für den Aufruf ohne `--report`.
- `settings`: Kandidaten in Suchreihenfolge; ein Glob im Namen
  (`*.runsettings`) gilt nur, wenn er genau eine Datei trifft, sonst ein
  Hinweis.
- `include` / `exclude`: Schlüssel, deren Werte Verzeichnisse oder Muster
  sind; `defaults` gelten, wenn die Datei den Schlüssel nicht setzt.
- `scope = "all"`: ohne Include-Schlüssel ist jede Datei mit den Endungen
  erwartet (Nano Coverage instrumentiert alles).
- `switches`: Boolesche Schlüssel, die Muster zuschalten.
- `patterns`: `glob` (Syntax von `internal/pathkey`) oder `regex`
  (gcovr, gegen den Pfad relativ zur Einstellungsdatei).
- `markers`: native Marker als Regex, optional mit Gruppe `reason`, optional
  `reason = false` (etwa `LCOV_EXCL_STOP`); `marker_keys` nennt Schlüssel,
  deren Werte weitere Marker sind (`.lcovrc` `lcov_excl_line`, gcovr
  `exclude-pattern-prefix`).

Erstbestand: coverage.py, coverlet, gcovr, lcov/geninfo, llvm-cov und
cargo-llvm-cov (nur Berichtsorte und `#[coverage(off)]`), PHPUnit, nyc, c8,
istanbul/vitest (Marker und Berichtsorte; Bereich nicht lesbar), JaCoCo,
Kover, Go, Nano Coverage.

Ein Profil ist **erkannt**, wenn eine seiner Einstellungsdateien unter
`--root` liegt oder einer seiner Standardberichte gelesen wurde. Mehrere
Profile gelten nebeneinander (Python und GDScript in einem Repo); jedes
erwartet nur Dateien seiner Endungen.

### 7. Bereich und erwartete Quellen

1. **Erwartet** ist jede Datei unter `--root` mit einer Endung eines
   erkannten Profils, die im Include liegt (oder `scope = "all"`), nicht im
   Exclude, nicht in einem anderen Checkout und ohne `coverage:exempt-file`.
2. Eine erwartete Datei, die in keinem Bericht vorkommt, ist ein Befund
   `absent from the report`.
3. Ein Datensatz für eine Datei im Exclude wird übergangen: das Werkzeug hätte
   ihn nicht schreiben sollen, und er beschreibt nichts, das zum Bereich
   gehört.
4. Liefert kein erkanntes Profil einen Bereich (coverlet nur mit
   `ExcludeByFile`, vitest), entfällt allein Regel 2, und ein Hinweis nennt
   das Profil.

### 8. Regeln je Zeile

Geprüft wird jede Datei des zusammengeführten Berichts, die auf der Platte
liegt.

| Fall | Befund |
|---|---|
| Datei im Bericht, nicht auf der Platte | `<pfad> is in the report but not on disk` |
| Zeile mit 0 Treffern ohne Marker | `<pfad>:<n> not covered` |
| `coverage:exempt` ohne Grund | `<pfad>:<n> coverage:exempt without a reason` |
| `coverage:exempt` auf getroffener Zeile | `<pfad>:<n> coverage:exempt but hit <k> times` |
| `coverage:exempt-file` ohne Grund | `<pfad> coverage:exempt-file without a reason` |
| `coverage:exempt-file` in einer Datei mit Treffern | `<pfad> coverage:exempt-file but <m> lines hit` |
| nativer Marker ohne Grund | `<pfad>:<n> <marker> without a reason` |

- **`coverage:exempt <grund>`** gilt für die Zeile, auf der er steht, oder,
  steht er auf einer reinen Kommentarzeile, für die Zeile direkt darunter.
  Reine Kommentarzeile heißt: nach Leerraum beginnt sie mit `#`, `//`, `/*`,
  `*`, `--`, `;`, `'`, `%` oder `<!--`. So passt der Marker auch an Zeilen,
  an die kein Kommentar zu hängen ist (Zeile in einem mehrzeiligen String,
  Klammerzeilen aus coverlet und gcov).
- **Grund** ist der Text nach dem Marker, ohne Kommentarschluss (`*/`,
  `-->`) und Leerraum, oder der Inhalt der Gruppe `reason`. Er muss
  nicht leer sein.
- **`coverage:exempt-file <grund>`** irgendwo in der Datei nimmt sie ganz aus:
  keine Zeilenregel, nicht erwartet. Für Dateien ohne ausführbaren Code, für
  die kein Werkzeug einen Datensatz schreibt (leeres `__init__.py`,
  Interface-Datei, GDScript nur mit `_init`).
- **Native Marker** werden in jeder Datei gesucht, die im Bericht steht oder
  erwartet ist: die Marker aller Profile aus `tools.toml` (ein
  `LCOV_EXCL_LINE` braucht kein `gcovr.cfg`, um als Marker zu gelten), dazu
  die Marker aus `marker_keys` erkannter Einstellungsdateien. Code-Muster wie
  `exclude_lines` von coverage.py (`if TYPE_CHECKING:`) sind keine Marker:
  sie sind eine Entscheidung für das ganze Projekt und stehen begründbar in
  dessen Einstellungsdatei.
- **Ein nativer Marker mit Grund nimmt seine Zeile aus**, wie
  `coverage:exempt`. Meist ist sie ohnehin nicht im Bericht; bei einem
  Werkzeug, das sie mit 0 Treffern stehen lässt, wäre sie sonst ein Befund,
  obwohl das Projekt sie begründet ausgenommen hat. Block-Marker
  (`block = "start"` / `"stop"` im Profil, etwa `LCOV_EXCL_START`/`STOP`)
  nehmen jede Zeile dazwischen aus; der Grund steht am Start. Ein Start ohne
  Stop reicht bis zum Dateiende und ist ein Befund `<marker> without a stop`.
  Ein nativer Marker auf einer getroffenen Zeile ist kein Befund: das
  Werkzeug selbst hat ihn dann nicht angewandt, und seine Semantik
  (`ignore next` gilt für einen Ausdruck, nicht für eine Zeile) ist nicht
  die von `coverage:exempt`.
- Ob eine Zeile ausführbar ist, entscheidet das Werkzeug. Bewertet wird nur,
  was im Bericht steht.

### 9. Frische und Leere

1. **Frische über den Ablageort.** In einer Lane schreibt die Test-Lane nach
   `{coverdata}` (Pfad mit Run-ID, `internal/verify/cover.go`), und die
   Coverage-Lane ruft `{loomux} check linecov --report {coverdata}`. Ein
   Bericht eines früheren Laufs kann dort nicht liegen; „veraltet“ wird
   „fehlt“. Kein neuer Platzhalter.
2. **mtime gegen Quellen**, auch von Hand: Ist eine erwartete oder geprüfte
   Quelle neuer als der neueste gelesene Bericht (2 s Spiel für FAT), ist das
   der Befund `<pfad> is newer than the report`.
3. **Leer ist rot:** kein Bericht gefunden (`no coverage report found;
   looked in …` mit den Orten) oder kein Zeilendatensatz nach dem Übergehen
   (`the reports hold no line records`). Wie `check gocover`: ein Tor, das
   nichts zu prüfen findet, besteht nicht.

### 10. Aufbau

```
internal/verify/linecov/
  report.go     Report, merge
  format.go     Format interface, registry, detection
  lcov.go  cobertura.go  jacoco.go  goprofile.go
  syntax.go     settings-reader interface, registry
  ini.go  tomlset.go  xmlset.go  jsonset.go  godotcfg.go
  tools.go      tools.toml schema, lazy load, detection of profiles
  tools.toml
  paths.go      normalisation, nested checkouts, suffix search
  scope.go      expected sources
  markers.go    coverage:exempt, exempt-file, native markers, reason
  gate.go       rules of decisions 7–9 → []Finding
internal/cli/check.go   checkLinecov, modelled on checkGocover
```

`gate.go` sieht `Report`, Bereich und Marker, nie ein Format oder Werkzeug.

## Erweitern

| Erweiterung | Aufwand |
|---|---|
| Werkzeug mit bekannter Syntax | Eintrag in `tools.toml`, Fixtures |
| neue Syntax einer Einstellungsdatei | ein Leser, ein Registry-Eintrag, Fixtures |
| neues Berichtsformat (OpenCover, Clover) | ein Leser, ein Registry-Eintrag, Fixtures |

Ein Test läuft über jedes Profil in `tools.toml` und verlangt
`testdata/tools/<name>/` mit mindestens einer Einstellungsdatei (sofern das
Profil `settings` hat), einer Quelle mit jedem nativen Marker samt Grund und
einem Bericht. Er prüft, dass Bereich und Marker gelesen werden. Ein zweiter
Test verlangt für jedes Format `testdata/formats/<name>/`. Ein Eintrag ohne
Fixtures lässt die Suite scheitern.

## Abgrenzung

- **Keine Presets.** Die Godot-Lanes `test` und `coverage` kommen mit Teil 3;
  Presets anderer Sprachen sind eigene Schritte. `check gocover` und das
  Go-Preset bleiben unverändert: `gocover` prüft je Funktion, `linecov` je
  Zeile; ein Go-Projekt kann `linecov` zusätzlich in eine Lane schreiben.
- **Keine Zweige, keine Untergrenze.** Jede Zeile wird gehalten. Eine
  Untergrenze kommt, wenn ein Projekt weniger als jede Zeile will.
- **Keine Schwellen der Werkzeuge.** Das Werkzeug prüft seine Summe selbst.
- **Keine Konfiguration in Code.** `vitest.config.ts`, `nyc.config.js`,
  Gradle- und Kover-DSL werden nicht ausgeführt; dort entfällt Regel 7.2 mit
  Hinweis.
- **Keine Profile in `.loomux/config.toml`.** Ein neues Werkzeug kommt in
  `tools.toml`.

## Grenzen

- Marker werden zeilenweise im Text gesucht, ohne Sprachparser. Ein
  `coverage:exempt` in einem String-Literal gilt als Marker; der Fall ist
  selten, und der Befund „but hit“ fängt ihn, sobald die Zeile läuft.
- Die Endungssuche für JaCoCo hängt an eindeutigen Paketpfaden. Ein Projekt
  mit derselben Klasse in zwei Quellwurzeln bekommt `ambiguous source`
  statt einer Annahme.
- Die mtime-Regel kann eine Quelle melden, die während des Laufs geschrieben
  wurde. Das ist gewollt: der Bericht beschreibt sie nicht nachweisbar.

## Folgen für space (Teil 6)

| space heute | danach |
|---|---|
| `lcov_merge.py` (Scherben summieren) | Entscheidung 3 |
| `DA_RECORD` positionsweise, Prüfsumme | Entscheidung 2, LCOV |
| `# no-cover: <grund>` | `# coverage:exempt <grund>`; alle Marker umschreiben |
| Marker auf getroffener Zeile | Entscheidung 8 |
| `MEASURED_DIRS`, „fehlt im Bericht“ | Profil `nanocoverage`, `scope = "all"` minus `ignore_paths` und `addons/`; die Differenz zu `MEASURED_DIRS` (etwa Helfer unter `test/`, die nicht auf `_test.gd` enden) geht nach `ignore_paths` |
| `UNMEASURED_DIRS = ("harness",)` | `harness/**` in `ignore_paths`: nicht instrumentiert, nicht erwartet |
| `IGNORED_PREFIXES` | Entscheidung 4.4, ohne Liste |
| `_has_instrumentable_code` (nur `_init`) | `# coverage:exempt-file <grund>` in diesen Dateien |
| `_staleness` | Entscheidung 9.2 |
| `_not_regenerated` (`measured_after`) | Entscheidung 9.1 |
| keine Datensätze | Entscheidung 9.3 |

## Tests

Jede neue Testzeile läuft vor dem Code rot; jede Regel bekommt die
Mutationsrunde mit einem Kontrollmutanten. 100 % je Funktion.

- **Formate:** je Format Fixtures in der Form jedes Werkzeugs aus der
  Recherche; echte Ausgaben für das Go-Profil (loomux selbst) und Nano
  Coverage (space). LCOV mit Prüfsumme, `DA:` vor `SF:`, unlesbarer
  Datensatz, CRLF, BOM; Cobertura mit und ohne `<sources>`, absolut und
  relativ; JaCoCo mit null, einem und zwei Treffern der Endungssuche;
  Go-Profil mit überlappenden Blöcken und einem Pfad außerhalb des Moduls.
- **Zusammenführen:** Vereinigung bei disjunkten Dateien, Summe, Sättigung,
  zwei Formate in einem Lauf.
- **Pfade:** `\`, `res://`, absolut innerhalb und außerhalb, Windows-Groß-
  und Kleinschreibung, anderer Checkout mit `.git`-Datei und mit
  `.git`-Verzeichnis.
- **Einstellungen:** je Syntax gültig, leer, kaputt; coverage.py-Suchreihenfolge
  (die erste Datei mit Abschnitt gewinnt, eine Datei ohne Abschnitt nicht);
  `*.runsettings` mit null, einer, zwei Dateien; `ignore_addons` an und aus.
- **Regeln:** jede Zeile der Tabelle in Entscheidung 8 positiv und negativ;
  Marker auf der Zeile und darüber, darüber aber keine reine Kommentarzeile;
  Grund leer nach Kommentarschluss; Gruppe `reason` (`Justification`);
  `reason = false`; Marker aus `marker_keys`.
- **Frische und Leere:** kein Bericht, leerer Bericht, nur übergangene
  Datensätze, Quelle neuer als Bericht innerhalb und außerhalb des Spiels.
- **Registries:** die beiden Fixture-Pflicht-Tests aus „Erweitern“, dazu
  einer, der mit einem Profil ohne Fixtures rot wird.
- **CLI:** Exit-Codes 0, 1, 2; Aufruf ohne Flags in einer Testwelt mit
  `.coveragerc` und `coverage.xml`; `linecov` in `Reserved`.

## Doku

- `docs/{en,de}/cli-reference.md`: Abschnitt `check linecov` mit den Regeln,
  Markern, der Profilliste und den Quellen der Recherche.
- `README.md`, `README.de.md`: die Roadmap-Zeile „LCOV-Coverage-Tor“ fällt
  weg; die Fähigkeit steht in der Liste der Prüfungen. In der Zeile
  „Godot-Testsuite“ heißt die Abhängigkeit `check linecov`.

## Zu belegen vor dem Plan

Offen aus der Recherche, mit Einfluss auf Profile, nicht auf die Regeln:

- `.lcovrc` `include`/`exclude`: Semantik (die Manpage war abgeschnitten).
- coverlet, PHPUnit, istanbul/v8, grcov, `#[coverage(off)]`: ob ausgenommene
  Zeilen aus dem Bericht fallen oder mit 0 Treffern bleiben. Für die Regeln
  gleichgültig (Entscheidung 8 nimmt die Zeile in beiden Fällen aus), für die
  Fixtures nicht: sie sollen zeigen, was das Werkzeug wirklich schreibt.
- `.coveragerc.toml`: Tabellennamen (`[run]` oder `[tool.coverage.run]`).
- PHPUnit: Suchreihenfolge `phpunit.xml` / `phpunit.xml.dist`.
- Nano Coverage: Glob-Semantik von `ignore_paths`.
- vitest: ob `cobertura` ein eingebauter Reporter ist (nur Berichtsorte).

## Nachtrag 1 (2026-10-15): Belege und was sie ändern

Die offenen Punkte oben sind belegt (Quellcode der Werkzeuge, Stand
2026-10-15). Fünf davon ändern die Spec:

1. **Musterdialekte statt `glob`/`regex`.** Die Werkzeuge lesen ihre Muster
   verschieden, und ein falscher Dialekt verschiebt still den Bereich:
   - Nano Coverage (`script_scanner.cpp`): `**` wird `.*`, `*` wird `[^/]*`,
     verankert, abgeglichen gegen den vollen `res://…`-Pfad. `addons/**`
     trifft dort nie, `res://addons/**` und `**/*_test.gd` schon.
   - lcov (`lcovutil.pm`, `transform_pattern`): `*` wird `.*` und kreuzt `/`,
     keine Verankerung (Teilstring), gegen den absoluten Pfad.
   - gcovr: Regex. coverage.py: fnmatch, ein Muster ohne `/` trifft den
     Dateinamen in jedem Verzeichnis. nyc, c8, coverlet `ExcludeByFile`:
     minimatch-artig. PHPUnit: Verzeichnisse und Dateien, keine Muster.

   `patterns` im Profil nennt darum einen Dialekt: `coveragepy`,
   `minimatch`, `nano`, `lcov`, `regex`, `dir`. Jeder übersetzt in einen
   Matcher über den Pfad relativ zu `--root`, mit dem Bezug, den das
   Werkzeug hat (Verzeichnis der Einstellungsdatei, `res://`, absoluter
   Pfad). Ein neuer Dialekt ist eine Funktion und ein Eintrag in einer
   Tabelle, wie Syntaxen und Formate.
2. **Grund eines nativen Markers auch darüber.** php-code-coverage erkennt
   `@codeCoverageIgnore` nur exakt (`=== '@codeCoverageIgnore'`); Text
   dahinter schaltet den Marker ab. Ein nativer Marker gilt darum auch als
   begründet, wenn die Zeile direkt darüber eine reine Kommentarzeile mit
   Text ist. Für `coverage:exempt` bleibt der Grund hinter dem Marker
   Pflicht: loomux erkennt ihn mit Text.
3. **v8 lässt ignorierte Zeilen stehen**, als `DA:<n>,1` (v8-to-istanbul,
   `s[i] = line.ignore ? 1 : line.count`). Das trägt Entscheidung 8: ein
   nativer Marker auf einer getroffenen Zeile ist kein Befund.
4. **Suchreihenfolgen.** PHPUnit: `phpunit.xml`, `phpunit.dist.xml`,
   `phpunit.xml.dist` (`XmlConfigurationFileFinder`). coverage.py:
   `.coveragerc`, `.coveragerc.toml` (`[run]` oder `[tool.coverage.run]`),
   `setup.cfg`, `tox.ini`, `pyproject.toml` (`[tool.coverage.run]`).
   Einstellungsdateien werden direkt in `--root` gesucht, nicht darunter.
5. **Standardorte der Berichte** im Erstbestand: `coverage.xml`,
   `coverage.lcov` (coverage.py); `coverage.cobertura.xml`, `coverage.info`
   (coverlet msbuild); `coverage/lcov.info`,
   `coverage/cobertura-coverage.xml` (istanbul, vitest, nyc, c8);
   `target/site/jacoco/jacoco.xml` (JaCoCo Maven);
   `build/reports/jacoco/test/jacocoTestReport.xml` (JaCoCo Gradle) und
   `build/reports/kover/report.xml` (Kover), beide nicht primär belegt;
   `coverage-report/lcov.info` (Nano Coverage). gcovr, cargo-llvm-cov,
   llvm-cov und PHPUnit schreiben ohne Ziel nach stdout oder gar nicht und
   haben keinen Standardort. Ein falscher Standardort kostet nichts: er wird
   nicht gefunden.

6. **Native Dateimarker.** `istanbul ignore file`, `c8 ignore file` und
   `v8 ignore file` nehmen eine ganze Datei aus. Ohne eigene Form wäre die
   Datei der Befund „absent from the report“. Ein Profil-Marker mit
   `block = "file"` wirkt darum wie `coverage:exempt-file` (Grund Pflicht,
   Datei nicht erwartet, keine Zeilenregel), nur ohne den Befund „but lines
   hit“: v8 schreibt die Zeilen einer ignorierten Datei als getroffen.
7. **Kein Profil ohne Wirkung.** llvm-cov und `go test` haben weder
   Einstellungsdatei noch Standardort noch Marker; ein Profil für sie würde
   nie erkannt. Ihre Berichte liest `linecov` über die Formate LCOV und
   Go-Profil, ein Profil bekommen sie nicht. cargo-llvm-cov bleibt als Profil
   `rust` wegen `#[coverage(off)]`.

Klarstellungen ohne Änderung:

- `--report`-Pfade sind relativ zum Arbeitsverzeichnis wie jedes Argument;
  in einer Lane ist das `--root`.
- Ein Befund ohne Datei (kein Bericht, keine Datensätze) wird als blanker
  Text ausgegeben.
- Nano Coverage behandelt `_init` wie jede Funktion (`instrumenter.cpp`);
  die Beobachtung in space, dass `_init`-Rümpfe nie im Bericht stehen, ist
  damit in Teil 6 neu zu messen. `coverage:exempt-file` deckt beide Fälle.
