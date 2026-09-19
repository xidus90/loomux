# Stufe 2a: `child`, `[verify]`, `loomux check <profil>`, Coverage-Tor

**Stand:** 2026-09-19, umgesetzt. Was bei Planung und Umsetzung vom Entwurf
abwich, steht unter „Nachträge" am Ende; wo Text und Nachtrag sich
widersprechen, gilt der Nachtrag. Rahmen: `2026-09-14-loomux-fusion-design.md`,
Stufe 2. Nachfolger: 2b (commit-msg vollständig), 2c (`stop`,
`subagent-start`/`-stop`, Antigravity-Adapter).

Stufe 2 der Fusion ist zu groß für einen Plan: rund 4.500 Zeilen Python
(`checks.py`, `config.py`, `process.py`, `commit/*`, `hooks/*`,
`hooks/coverage-check.py`) und 900 Zeilen Go aus ultraloom. Sie zerfällt wie
1b in drei Teilstufen:

| Teilstufe | Inhalt | hängt an |
|---|---|---|
| **2a** | `child`, das Schema `[verify]`, Presets je Stack, `loomux check <profil>`, `loomux check gocover`, post-edit auf `[verify]` | – |
| **2b** | commit-msg mit `--language`, `--calibrate`, `[commit]` | – |
| **2c** | Hooks `stop`, `subagent-start`, `subagent-stop`, Antigravity-Adapter vollständig | 2a |

Diese Spec beschreibt 2a.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| post-edit | Liest seine Lanes **in 2a** aus `[verify]`; die fest verdrahteten Lanes in `internal/hooks/post_edit.go` entfallen |
| Befehlsform | `loomux check <profil\|arten>`; reservierte Namen daneben (siehe unten) |
| Mehrsprachigkeit | **Eine Lane je Stack und Art.** Presets je Stack; Überschreiben mit `[verify.<stack>]` |
| Presets | **Eingebettetes TOML** im selben Schema wie `[verify.<stack>]`, geladen beim ersten Gebrauch |
| Defaults | Presets gelten **ohne Konfiguration**; `test` und `coverage` nur, wenn Tests erkannt sind. `loomux check --show` zeigt die wirksame Tabelle. Den auskommentierten Kommentarblock schreibt `loomux init` in Stufe 4 |
| Coverage-Tor | `dev covergate` wird zu `loomux check gocover` (Modul aus `go.mod`, 100 % je Funktion als Vorgabe, `--floor N`). `hooks/coverage-check.py` wird nicht nachgebaut; die Zwei-Arm-Form ist der Normalfall zweier Stacks |
| `dev covergate` | Entfällt; `ci/gate.sh` stellt um |
| Befehle | **argv, keine Shell.** Zerlegt nach Shell-Wortregeln, Platzhalter je Argument ersetzt |
| Fristen | `timeout` je Befehl, ohne Obergrenze beim Laden. Das **Budget** gehört dem Scope |
| Altfehler | Werden behoben, nicht übernommen; jeder ist ein Eintrag der Abweichungsliste |

## Befunde, die den Entwurf formen

Gelesen und gemessen am 2026-09-18/19.

- **Es gibt keine aufgezeichneten Fälle der Prüfkette.** Die „parity"-Treffer in
  ultraloom gehören zu den Sitzungshooks. 2a beginnt deshalb mit einem
  Aufzeichnungsdurchgang.
- **Die Go-Seite von ultraloom portiert die Kette nicht.** `internal/verify`
  dort enthält die Werkzeuge, die eine Konfiguration aufruft (`gofmt`, `types`,
  Coverage-Untergrenze), nichts liest `[verify]` und nichts plant Prüfungen.
- **Aufzeichnungsquelle:** Der Tag `loomux-1a-source` zeigt auf ultraloom-HEAD
  `9d01a60`; `checks.py`, `config.py`, `process.py`, `cli.py` und
  `hooks/coverage-check.py` sind seitdem unverändert.
- **Ein Marker gewinnt, heute.** `checks.py` wählt genau ein Preset in der
  Reihenfolge `pyproject.toml` → `CMakeLists.txt` → `package.json` →
  `project.godot`; Go hat keins. post-edit in loomux dagegen erkennt alle
  Stacks. Die beiden Seiten lesen die Welt schon heute verschieden.
- **argv an `.cmd`-Dateien:** `exec.Command("npx", …)` reicht unter go1.27
  `a b.ts`, `q"uote` und `x&y` unverändert durch (Wegwerf-Spike, 2026-09-18).
- **Prozessbaum unter Windows:** Suspendierter Start, Zuordnung zu einem Job
  Object mit `KILL_ON_JOB_CLOSE`, dann `NtResumeProcess` aus ntdll: Drei
  Prozesse (cmd und zwei Enkel) standen im Job, nach `TerminateJobObject` null
  (Wegwerf-Spike, 2026-09-18). Die Toolhelp-Suche nach dem Thread-Handle, die
  `process.py` braucht, entfällt.
- **Hook-Fristen in loomux:** post-edit 60 s, pre-tool-use 15 s,
  session-start 20 s (`.claude/settings.json`). Der Git-Hook hat keine Frist.

## Pakete

- **`internal/child`** (neu): `Run(Spec) Result`. `Spec` trägt argv,
  Verzeichnis, Zusatzumgebung und Frist; `Result` Exit-Code, stdout, stderr,
  `TimedOut` und `OutputAbandoned`.
  - Windows: suspendiert starten, Job Object, `NtResumeProcess`; die Tötung
    beendet den Job. Ein Kind, das mit `CREATE_BREAKAWAY_FROM_JOB` ausbricht,
    ist die dokumentierte Ausnahme.
  - POSIX: eigene Prozessgruppe, `kill(-pgid, SIGKILL)`.
  - Beide Arme hinter Build-Tags, die Tests ebenso (`_windows_test.go`,
    `_other_test.go`).
  - Umgebung: Git-Variablen über `internal/gitenv` entfernt,
    `PYTHONIOENCODING=utf-8` gesetzt.
  - Ausgabe als UTF-8 mit Ersetzung, `\r\n` und `\r` werden `\n`. Eine
    gemeinsame Nachfrist von 5 s deckt Tötung, Abholen und Absaugen der Pipes.
    Ein Leser, der nach sauberem Ende nicht fertig wird, ergibt
    `OutputAbandoned` und tötet den Baum.
- **`internal/verify`** (wächst):
  - `schema.go` lädt `[verify]` streng, Fehlertexte im Stil von `[policy]`
    (`<pfad>: [verify.go].lint …`).
  - `presets.toml` (eingebettet) und `presets.go`; geparst beim ersten
    Gebrauch, nie in `init()` oder einer Paketvariable.
  - `plan.go` bildet aus Stacks, Bereichen und Anfrage die Lanes und ihren
    Graphen.
  - `run.go` fährt den Graphen. **Naht:** `Run` nimmt
    `func(child.Spec) child.Result`; Produktion reicht `child.Run`, Tests und
    Fallsuite einen Fake.
  - `report.go` schreibt die Ausgabe.
- **`internal/verify/gocover`**: `internal/dev/covergate` zieht hierher. Das
  Modul kommt aus `go.mod` statt aus einer Konstante.
- **`internal/detect`**: bekommt die Bereiche je Stack (`Areas map[string][]string`),
  gelesen wie bisher im Wurzelverzeichnis und eine Ebene darunter.
- **`internal/cli/check.go`**: verteilt auf die reservierten Namen, alles
  andere ist Profil oder Komma-Liste.
- **`internal/hooks/post_edit.go`**: schrumpft auf Nutzlast → Datei → Stack →
  `verify.Plan(scope=edit, file)` → Lauf → Bericht.

**Abhängigkeiten:** `hooks → verify → child, gitenv, detect, brain/wiki`.
`verify` importiert nie `hooks`. Die Fusions-Spec nennt heute
`verify → child, brain/check`; post-edit ruft aber `wiki.LintReport` aus
`brain/wiki`. Die Fusions-Spec wird entsprechend korrigiert.

**Datenfluss:** `config` liest `.loomux/config.toml` → `detect` liefert Stacks
und Bereiche → `verify.Load` legt Preset, Variante und `[verify.<stack>]`
übereinander → `Plan(anfrage, scope)` → `Run` → `Report` → Exit-Code.

## Schema `[verify]`

```toml
[verify]
max_parallel = 8        # Vorgabe: Zahl der CPUs
timeout      = 600      # Sekunden je Befehl; Vorgabe 600, keine Obergrenze

[verify.profiles]       # eingebaut: edit = [lint, types], precommit = [alle vier]
edit      = ["lint", "types"]
precommit = ["lint", "types", "test", "coverage"]

[verify.go]             # je Stack; nicht genannte Stacks behalten ihr Preset
lint     = ["go vet ./...", "{loomux} check gofmt cmd internal"]
coverage = "{loomux} check gocover --profile {coverprofile} --floor 90"

[verify.typescript.lint]            # Tabellenform
commands = ["npx eslint ."]
on_file  = ["npx eslint --cache {file}"] # Form für post-edit; fehlt sie, gilt commands
threaded = true

[verify.cpp]
types = false           # Lane abschalten, auch gegen ein Preset

[verify.project]        # projektweit, ohne Stack, im Wurzelverzeichnis
lint = "make lint"
```

### Stacks

Tabellen `[verify.<stack>]` gibt es nur für Sprachen: `go`, `python`,
`typescript`, `vue`, `svelte`, `css`, `html`, `gdscript`, `cpp`, `shell`, `sql`,
`rust`; dazu `wiki` und `project`. Ein anderer Name ist ein Ladefehler.

- `detect` mischt Sprachen und Werkzeuge (`pyright`, `uv`, `cmake`, `meson`,
  `clang-tidy`, `biome`, `golangci-lint`, `stylelint`, `htmlhint`,
  `shellcheck`). Die Werkzeugsignale wählen in `presets.toml` eine **Variante**
  (`when = "pyright"` → pyright statt mypy). Jedes `when` muss ein Signalname
  aus `detect` sein; ein Test prüft das an `presets.toml`. Treffen mehrere
  Varianten zu, gilt die erste in Dateireihenfolge.
- Schichtung: Preset → Variante → `[verify.<stack>]`.
- `javascript` gibt es nicht; `.js` läuft als `typescript`, wie heute in
  post-edit. `godot` heißt als Sprache `gdscript`.
- `shell`, `sql` und `rust` haben kein Preset. **Ein konfigurierter Stack gilt
  als erkannt:** trägt `[verify.<stack>]` mindestens einen Befehl, läuft er
  mit dem Wurzelverzeichnis als Bereich, auch wenn `detect` ihn nicht findet.
- `wiki` kennt nur `lint = false`; die Lane ist ein Funktionsaufruf im Prozess
  und läuft in 2a nur im Edit-Scope. `loomux wiki-gate` bleibt eigener Befehl.
- `project` läuft nur im `check`-Scope, im Edit-Scope nur mit `on_file`.

### Formen je Art

Eine Stack-Tabelle kennt die Schlüssel `lint`, `types`, `test`, `coverage`;
nur `[verify.gdscript]` zusätzlich `import_check`. Jede Art ist ein String, eine
Liste, `false` oder eine Tabelle mit `commands`, `on_file`, `threaded`,
`measuring`, `measure`, `after`.

- Andere Schlüssel sind ein Ladefehler, ebenso unbekannte Schlüssel unter
  `[verify]`.
- `measuring` und `measure` sind je **ein** Befehl (String). `measuring`
  ersetzt die ganze `commands`-Liste für diesen Lauf, wie in Python; eine
  messende Lane hat also genau einen Befehl.
- `measuring`, `measure` und `after` nur bei `test` und `coverage`.
- `true` ist ein Ladefehler („Schlüssel weglassen, dann gilt das Preset").
- Eine leere Liste und ein leerer Befehl sind Ladefehler.
- Die alten Top-Level-Arten (`[verify].types = "…"`) sind ein Ladefehler mit
  Hinweis auf `[verify.<stack>]`.
- `[verify.gdscript] import_check = true|false` (Vorgabe `true`) steuert den
  Zustand `unready`.

**Zusammenführen mit dem Preset.** Ein String oder eine Liste **ersetzt die
ganze Lane**: `measuring`, `measure` und `on_file` des Presets gelten dann
nicht mehr, nur `after` bleibt. Wer einen Befehl hinschreibt, meint ihn so,
wie er dasteht. Das entspricht dem alten Verhalten, in dem eine konfigurierte
Art das Preset ersetzte. Eine **Tabelle führt Schlüssel für Schlüssel
zusammen**: `[verify.go.test] measuring = "…"` ändert nur die messende Form
und behält `commands` aus dem Preset. `false` schaltet die Lane ab.

Ein Schlüssel ist in TOML entweder Wert oder Tabelle. `test = "…"` und
`test.measuring = "…"` in derselben Tabelle sind deshalb ungültig. Wer beides
braucht, nimmt die Tabellenform mit `commands`.

### Platzhalter

`{file}` und `{pkgdir}` (beide nur in `on_file`; `{pkgdir}` ist das nächste
Verzeichnis mit `package.json` über der Datei), `{area}`, `{coverprofile}`,
`{coverdata}`, `{loomux}`. Ersetzt werden nur diese Namen; alle anderen
Klammern bleiben wörtlich (`-run 'Test{A,B}'`).

- `{loomux}` ist das laufende Binary (`os.Executable()`). Nur so findet das
  Go-Preset `gocover` auch bei loomux selbst, wo das Binary unter `bin/` und
  nicht auf dem PATH liegt.
- `{coverprofile}` und `{coverdata}` zeigen auf
  `.loomux/state/cover/<lauf-id>.out` bzw. `.data`. Bei grünem Ergebnis werden
  sie am Laufende gelöscht; bei rotem bleiben sie zum Nachsehen liegen. Jeder
  Lauf räumt, was nicht seine `<lauf-id>` trägt. Pfade außerhalb von
  `.loomux/state/cover/`, etwa ein `coverage.out`, das ein fremder Aufrufer
  `gocover` direkt reicht, berührt loomux nie.
- Python bekommt `COVERAGE_FILE={coverdata}` in der Lane-Umgebung statt eines
  festen `.coverage`.
- **Ladeprüfung:** Steht `{coverprofile}` in `coverage`, aber in keinem von
  `test`, `test.measuring` und `coverage.measure` desselben Stacks, ist das ein
  Ladefehler. Fehlt die Datei zur Laufzeit, ist die Coverage-Lane rot mit
  eigener Meldung. Die Pfade je Lauf schließen die Python-Lücke „Coverage über
  alten Daten" ohnehin; die Ladeprüfung ist Komfort.

### `after`

Nur innerhalb eines Stacks; im Preset `coverage` → `test`. Zyklen sind
Ladefehler, die Meldung nennt den Ring. Ein Sammelbericht über mehrere Stacks
ist bewusst nicht ausdrückbar.

### Profile und reservierte Namen

- `edit` und `precommit` sind eingebaut und überschreibbar; `all` ist reserviert
  und heißt immer alle vier Arten. post-edit fährt das Profil `edit`.
- Reserviert sind, an einer Stelle: `gofmt`, `commit-msg`, `gocover`, `all`,
  `lint`, `types`, `test`, `coverage`. Ein Profil mit einem dieser Namen ist ein
  Ladefehler; die alte Verdeckung einer Art durch ein Profil entfällt.
- Ein leeres Profil ist ein Ladefehler.

### Testerkennung

`tests_when` je Stack in `presets.toml`, rekursiv gesucht, ausgenommen `.git`,
`vendor`, `node_modules`, `third_party`:

| Stack | Signal |
|---|---|
| go | `*_test.go` |
| python | `tests/`, `test_*.py`, `[tool.pytest` in `pyproject.toml` |
| cpp | `enable_testing(` in `CMakeLists.txt` |
| typescript | `vitest` in `package.json` |

- Gesucht wird nur, wenn `test` oder `coverage` angefordert ist. Das Profil
  `edit` fordert beide nicht; post-edit läuft also keinen rekursiven Walk.
- Keine Tests ⇒ `test` und `coverage` sind `unavailable`.
- Ein ausdrücklich konfiguriertes `[verify.<stack>].test` übergeht die
  Erkennung: Wer den Befehl hinschreibt, hat Tests.
- Ein Stack ohne `tests_when` (`gdscript`, `project`) hat keine Erkennung:
  Seine Lane läuft, sobald sie definiert ist.

### Presets

Übernommen aus `checks.py`, ergänzt um Go und die post-edit-Lanes, mit einer
Änderung: Die JS/TS-Presets rufen `npx eslint`, `npx tsc --noEmit` und
`npx vitest run` statt der nackten Namen aus `checks.py`. Mit `LookPath` vor
jedem Start wäre ein nicht global installiertes Werkzeug sonst `missing-tool`;
post-edit ruft heute schon `npx`. Die Coverage-Zeilen:

```toml
[go.test]
commands  = ["go test ./... -count=1"]
measuring = "go test ./... -count=1 -covermode=set -coverprofile={coverprofile}"

[go.coverage]
commands  = ["{loomux} check gocover --profile {coverprofile}"]
measure   = "go test ./... -count=1 -covermode=set -coverprofile={coverprofile}"
after     = "test"

[python.test]
commands  = ["uv run pytest -q --tb=short --no-header"]
measuring = "uv run coverage run -m pytest -q --tb=short --no-header"

[python.coverage]
commands  = ["uv run coverage report --skip-covered --skip-empty -m"]
measure   = "uv run coverage run -m pytest -q --tb=short --no-header"
after     = "test"

[cpp.test]
commands  = ["ctest --test-dir build --output-on-failure"]

[cpp.coverage]
commands  = ["gcovr --root . --object-directory build --fail-under-line 100 --txt"]
after     = "test"
```

- `test` misst nur mit, wenn `coverage` im selben Lauf steht; allein bleibt es
  der schnelle Weg. `coverage` allein misst selbst (`measure`).
- Läuft `coverage` ohne seinen Vorgänger und hat es kein `measure`, etwa weil
  eine String-Form die Lane ersetzt hat, ist die Lane rot mit
  „`test` did not run and there is no measure step". Pythons Warnung wird so
  zum Befund. Wer `coverage` als String überschreibt und es auch allein laufen
  lassen will, nimmt die Tabellenform und behält `measure`.
- Python: die Schwelle ist `fail_under` in `pyproject.toml`.
- C++: Das alte Preset `gcovr --cobertura coverage.xml` schrieb nur einen
  Bericht und war immer grün. `--fail-under-line` macht ein Tor daraus.
  Einen Build mit Coverage-Flags erzwingt kein Preset; ohne `.gcda`-Dateien
  meldet gcovr 0 % und das Tor ist zu Recht rot.
- Go prüft als einzige Sprache ohne Zusatzwerkzeug **je Funktion**
  (`go tool cover -func`). Python und C++ prüfen eine Gesamtgrenze.
- Das Go-Preset misst ohne `-coverpkg`, weil der Modulpfad nicht bekannt ist.
  Es untertreibt deshalb bei Tests über Paketgrenzen; ein Wirt wie loomux
  nennt `-coverpkg` in seinem `test.measuring`.
- Godot bekommt weiterhin kein Coverage-Preset (Begründung in `checks.py`).

### Wegfall aus dem alten Schema

| Alt | Neu |
|---|---|
| `[verify].tests` (Schutzliste des Reparatur-Flows) | Flow-Folgeprojekt bzw. `[policy]` |
| `.ultraloom/checks/<art>.*` | `[verify.project]` |
| `[verify.coverage].threshold`, `--threshold` | Schwelle im Tor (`--floor`, `fail_under`, `--fail-under-line`) |
| `[verify].godot_import` | `[verify.gdscript] import_check` |
| `[verify.coverage].report` | `[verify.<stack>].coverage` |
| `[exec].prefix` | entfällt; kein Wirt nutzt ihn (geprüft am 2026-09-19: `iam_backend`, `iam_frontend`, `iam_workers`, `space`, `ultra-brain`, `ultraloom`) |
| `[agent].cli_path`, `ULTRALOOM_CLI_PATH` | nicht Teil der Prüfkette; entfällt hier |
| `ULTRALOOM_ALONGSIDE` | entfällt; einziger Verbraucher war `coverage-check.py` |

## Lauf, Urteil, Bericht

### Lauf

- Der Plan ist ein Graph aus Lanes (Art × Stack × Bereich). Kanten entstehen
  nur aus `after`. Eine Lane startet, sobald ihr Vorgänger fertig ist;
  `max_parallel` begrenzt die Kindprozesse. Globale Stufen gibt es nicht mehr.
- Zwei Bereiche eines Stacks ergeben zwei Lanes (`lint/typescript@web`,
  `lint/typescript@admin`). Einen Override je Bereich gibt es bewusst nicht.
- Hat eine Lane mehrere Befehle, laufen alle, auch nach einem roten; mit
  `threaded` parallel, sonst nacheinander. Ein roter `measure`-Schritt beendet
  die Lane.
- Vor jedem Start prüft `LookPath` das Werkzeug, **auch bei konfigurierten
  Befehlen**. Sonst würde ein vom PATH gefallenes `~/go/bin` jeden Edit
  blockieren (Vorfall vom 2026-09-12).
- **Budget:** Ein Lauf hat eine Gesamtfrist; jedes Kind bekommt
  `min(eigene Frist, Restbudget)`. post-edit: 50 s (Hook-Frist 60 s minus
  10 s), über `--budget` einstellbar. `check`: kein Budget. `stop` bekommt
  seins in 2c.

### Zustände

| Zustand | wann | `check` | post-edit |
|---|---|---|---|
| ok | alle Befehle Exit 0 | grün | grün |
| failed | Exit ≠ 0, Startfehler, Ausgabe abgebrochen | rot | rot, Exit 2 |
| timed-out | eigene Frist | rot | rot |
| budget | Laufbudget erschöpft | – | übersprungen, Meldung |
| blocked | Vorgänger rot | rot | rot |
| missing-tool | Werkzeug nicht auf dem PATH | rot | übersprungen, Meldung |
| unready | Godot-Import fehlt | rot | übersprungen, Meldung |
| unavailable | Art definiert, kann nicht laufen (keine Tests erkannt) | zählt als „nichts lief" | nicht angezeigt |
| not-applicable | Art im Stack nicht definiert oder `false` | neutral, angezeigt | nicht angezeigt |

- **Erbregel:** Konnte ein Vorgänger nicht laufen, übernimmt der Nachfolger
  dessen Zustand (`unavailable`, `not-applicable` oder übersprungen), nicht
  `blocked`. Damit ist Altfehler 5 schon in 2a behoben.
- **Urteil `check`, je angeforderter Art:** Hat eine angeforderte Art keine
  Lane, die lief, und ist sie in keinem Stack `not-applicable`, ist das
  Exit 1 mit „nothing to check for `<art>`". Ein Tor, das nichts prüft, ist
  nicht grün. Sonst Exit 0, wenn keine Lane rot ist, und Exit 1, wenn eine rot
  ist. Ladefehler: Exit 1. Aufruffehler: Exit 2.
- **Urteil post-edit:** Rote Lane ⇒ Exit 2, Ausgabe auf stderr. Sonst Exit 0,
  Übersprungenes auf stdout. Eine Datei ohne erkannten oder konfigurierten
  Stack hat keine Lanes: Exit 0.

**Gegen loomux selbst durchgerechnet** (Go-only, Tests zwei Ebenen tief,
kein `types`): `lint/go` läuft; `types/go` ist `not-applicable`; `test/go`
findet `*_test.go` rekursiv und läuft; `coverage/go` läuft nach `test/go`.
Alles grün ⇒ Exit 0. Ein frisches Go-Repo ohne Tests: `test/go` und
`coverage/go` sind `unavailable` ⇒ Exit 1 für `test`.

### Ausgabe `check`

Alles auf stdout, Ladefehler auf stderr:

```
lint/go: ok [config] 1.2s
lint/python: failed [preset] 0.8s
<Ausgabe>
types/go: not-applicable [preset]
coverage/go: blocked [preset] by test/go
```

- Feste Reihenfolge: Arten in Anfrage-Reihenfolge, darin Stacks alphabetisch,
  dann Bereich. Ausgegeben wird am Ende, nie verzahnt.
- Grüne Lanes zeigen nur ihre Kopfzeile; `-v` zeigt alles.
- Mehrere Befehle: je Block `$ <argv>[ (failed)]`, wie in Python.
- Die Dauer maskiert eine Vergleichsklasse der Fallsuite (`\d+\.\ds`).

### `check --show [profil|arten]`

Führt nichts aus. Druckt je Stack die wirksame Tabelle als TOML, jede Zeile mit
Herkunft (`# preset`, `# preset, variant pyright`, `# config`), dazu Bereiche,
erkannte Tests und Budget. Die Ausgabe ist nach `.loomux/config.toml`
kopierbar; `loomux init` benutzt sie in Stufe 4 für den Kommentarblock.

### post-edit

Die Endung bestimmt den Stack über eine Tabelle in `presets.toml` (heute
`extensionStackMap` in Go). Der Bereich ist der nächste über der Datei. Es
laufen die Lanes des Profils
`edit`, mit `on_file`, wo vorhanden, sonst `commands`. `cmd /c` entfällt.

## `loomux check gocover`

`loomux check gocover --profile <pfad> [--floor N]`. Ohne `--floor` gilt
100 % je Funktion mit `//coverage:exempt <grund>` direkt über `func`, wie
`dev covergate` heute; mit `--floor` die Gesamtgrenze aus `go tool cover -func`.
Das Modul kommt aus `go.mod` im Arbeitsverzeichnis. Ein Profil ohne Funktionen
ist rot. `ci/gate.sh` ruft es nicht direkt, sondern über `check precommit`
und die Coverage-Lane des Go-Presets (siehe Selbstnutzung).

## Parität

### Aufzeichnen

- Aus ultraloom am Tag `loomux-1a-source` mit
  `loomux dev record-case --argv "<abs. Pfad zu uv> run --no-sync --project <ul> ultraloom"`.
  Die alte Oberfläche kennt nur `check {lint|types|test|coverage|all}`.
- **`internal/dev/faketool`** (wie `fakeqmd`), gebaut als `uv.exe`, `uvx.exe`,
  `go.exe`, `gcovr.exe` … und zuerst auf den PATH gelegt.
  - Schlüssel ist der Anfang der Kommandozeile (`uv run pytest`, `uvx ruff`),
    nicht der Name der Exe.
  - Antwort: Exit-Code, stdout, optional eine Pause für Fristfälle, optional
    `writes = [{path, content}]` für Coverage-Artefakte.
  - Der Rekorder ruft das echte `uv` über seinen absoluten Pfad; nur die Kinder
    lösen über den PATH auf (`_located` in `checks.py` nimmt `shutil.which`).
  - In der Fallsuite liest der Fake-Runner an der Naht von `verify.Run`
    dieselbe `faketool.json` der Welt; ein Prozess startet dort nur für
    `{loomux} check gocover`, das echt auf dem geschriebenen Fake-Profil läuft.
- Welten unter `testdata/cases/2a-worlds/`: python, go, gemischt, cpp,
  godot-unready. Welten mit Ladefehlern nur, wo das **alte** Schema denselben
  Fehler kannte (leere Liste, `after`-Zyklus, unbekannte Art im Profil). Fehler
  des neuen Schemas sind Unit-Tests.

### Übersetzen

`2a-map.toml`: `ultraloom check ` → `loomux check `. Die alte Konfiguration
wird beim Import gefaltet, so dass sie das alte Verhalten exakt reproduziert:
`[verify].<art>` → `[verify.project].<art>`, und jeder erkannte Stack der Welt
bekommt `<art> = false`, denn die alte Konfiguration ersetzte das Preset.

| Alt | Neu |
|---|---|
| `ultraloom check <art>\|all` | `loomux check <art>\|all` |
| `[verify].<art> = …` | `[verify.<stack>].<art>` bzw. `[verify.project].<art>` |
| `[verify.<art>] commands/threaded` | `[verify.<stack>.<art>] commands/threaded` |
| `[verify.coverage].report = "…coverage-check.py 98.0"` | `[verify.go] coverage = "{loomux} check gocover --profile {coverprofile} --floor 98"` und `[verify.python] coverage` aus dem Preset |
| `.ultraloom/checks/<art>.*` | `[verify.project].<art>` |
| `ulinit check gofmt` | `loomux check gofmt` |
| `loomux dev covergate --profile p` | `loomux check gocover --profile p` |

### Vergleichsklasse `lanes`

Neu. Sie zieht aus alter und neuer Ausgabe je Art das Urteil, fasst die neuen
Lanes je Art zusammen (Art rot, wenn eine Lane rot ist) und vergleicht Urteil
und Exit-Code, nicht die Befehlsausgabe. Sie lebt in `cases_2a_test.go`.

### Abweichungsliste `parity/stufe-2a.md`

1. CMake-Projekt fragt `types`: alt `KeyError`/Traceback, neu
   `cmake --build build --parallel`, der Compiler als Typprüfer (Nachtrag 2).
2. `threshold`/`--threshold` setzten nichts durch: entfallen.
3. Unbekannte Schlüssel unter `[verify]` wurden ignoriert: Ladefehler.
4. pre-commit rief `check all` statt des Profils: ruft `check precommit`.
5. `test` `unavailable` machte `coverage` `blocked`: Erbregel. Ein
   abgeschaltetes `test` (`not-applicable`) zählt als nicht angefordert,
   `coverage` misst dann selbst mit `measure` (Nachtrag 11).
6. `check <art>` umging Reihenfolge, Blockieren und Zyklusprüfung und endete
   bei Fehlern im Traceback: jeder Aufruf läuft durch denselben Planer.
7. Eine falsche `ULTRALOOM_CLI_PATH` brach jedes `load_config`: entfällt.
8. Fristen: `timeout` 600 s gegen Host-Frist 300 s (aus
   `docs/stufe-2-stop-notes`): `timeout` bleibt je Befehl, ohne Obergrenze
   beim Laden; das Budget gehört dem Scope (post-edit 50 s über `--budget`,
   `check` keins). Ist es aufgebraucht, startet keine Lane mehr, die offenen
   sind `budget`, kein Befund. Die Regel für `stop` gilt in 2c.
9. C++-Coverage war kein Tor: `--fail-under-line`.
10. `unavailable` war rot: neutral, solange die Art anderswo lief. Eine Art,
    die für einen erkannten Stack nicht definiert ist (GDScript `types` und
    `coverage`), ist `not-applicable`. Hat eine angefragte Art nirgends eine
    Lane, die lief, bleibt sie rot („nothing to check for `<art>`").
11. Ausgabe: eine Zeile je Lane statt je Art; grüne Lanes nur mit Kopfzeile.
12. Weggefallene Schlüssel (Tabelle oben).
13. `coverage-check.py` → `gocover` und Presets je Stack.
14. `dev covergate` entfällt.
15. post-edit: argv statt `cmd /c`, Lanes aus Presets; die 1a-Fälle
    `hook-post-tool-use` bleiben grün oder stehen hier.
16. **Go-Welten, gruppiert:** Alt gibt es kein Go-Preset, also ist dort alles
    `unavailable`; neu laufen die Go-Lanes.
17. post-edit rief `clang-format -i <datei>` und schrieb die Datei des Agenten
    um: Das Preset ruft `clang-format --dry-run --Werror {file}`, es prüft
    statt umzuschreiben (Nachtrag 3).
18. post-edit rief in einem Bereich die Workspace-Skripte des Wirts
    (`npm --prefix <bereich> run typecheck`, `run lint`, `run check`): Die
    Presets rufen `npx tsc --noEmit`, `npx eslint`, `npx svelte-check`,
    `npx vue-tsc --noEmit` direkt; ein Wirt mit eigenem Skript schreibt es in
    `[verify.typescript]` (Nachtrag 6).

## Tests und Tore

- TDD, 100 % je Funktion, jeder Ausschluss begründet.
- `child`: unter Windows mit echtem Prozessbaum (der Enkel überlebt nicht, die
  Frist greift, abgebrochene Ausgabe). Der POSIX-Arm trägt `//coverage:exempt`
  wie `runLane` heute; der Linux-Lauf in `.github/workflows/ci.yml` fährt
  zusätzlich `go test ./internal/child/...`, damit `kill(-pgid)` wirklich
  läuft.
- `verify`: jeder Ladefehler, Gültigkeit der Presets (jedes `when` und jeder
  Platzhalter bekannt, jedes Preset lädt), Plan, Erbregel, Budget, Urteil je
  Art.
- Startzeit-Regel: `presets.toml` wird beim ersten Gebrauch geparst; Nachweis
  mit `GODEBUG=inittrace=1`.
- `.githooks/pre-commit` prüft zusätzlich `*.toml` und `.loomux/` auf
  ungestagte Änderungen; die Konfiguration und `presets.toml` steuern jetzt das
  Tor.

## Selbstnutzung

`.loomux/config.toml` schreibt kein Agent. Der Plan **hält an**, bis der Mensch
diesen Block eingetragen hat:

```toml
[verify.go]
lint = ["{loomux} check gofmt cmd internal", "go vet ./..."]

# Tables merge onto the preset: commands stay, only the measuring forms change.
# -coverpkg because a function exercised only by another package's tests would
# otherwise read 0 %.
[verify.go.test]
measuring = "go test ./... -count=1 -covermode=set -coverpkg=github.com/xidus90/loomux/... -coverprofile={coverprofile}"

[verify.go.coverage]
measure = "go test ./... -count=1 -covermode=set -coverpkg=github.com/xidus90/loomux/... -coverprofile={coverprofile}"

[[policy.paths.rules]]
match  = [".loomux/state/**"]
reason = "Machine state is written by loomux, not by an edit."

[[policy.paths.rules]]
match  = ["testdata/cases/2a-source/**"]
reason = "Recordings of the old tools are evidence; re-record them, never edit them."
```

`.gitignore` trägt `/.loomux/state/` bereits. Danach ruft `ci/gate.sh` —
der eine Einstieg aller Forges — `go run ./cmd/loomux check precommit`, und
post-edit von loomux läuft über `[verify]`.

## Messen

In `docs/en/benchmarks.md` und `docs/de/benchmarks.md`, kalt und warm:

- Eigenzeit von `check precommit` ohne Werkzeuge (Fake-Runner, Exit 0 sofort).
- post-edit gegen den 1a-Zielwert von 72 ms; der Parse von `presets.toml` wird
  einzeln ausgewiesen.
- Vor der Mutationsrunde wird ihre Laufzeit geschätzt, weil jeder Mutant von
  `child` echte Prozessbäume startet.

## Fertig, wenn

Die fünf Kriterien der Fusions-Spec gelten: übersetzte Fälle grün oder
freigegeben, 100 % Coverage, Mutationsrunde über `child` und `verify` mit
dokumentierten Überlebenden, Zielwerte gemessen, loomux nutzt `check` selbst.
Dazu:

- READMEs (en, de) und die Befehlsdoku unter `docs/en`/`docs/de` zu `check`.
- Fusions-Spec: Teilstufen-Tabelle 2a/2b/2c, Abhängigkeitsregel, die Regel aus
  `docs/stufe-2-stop-notes` unter 2c.
- Release `release:minor`: `feat(check)` für Befehl und Schema; der Wegfall von
  `dev covergate` als `refactor(dev)`, denn `dev` ist Entwicklerwerkzeug und
  keine Produktschnittstelle. Kein Commit trägt `!`.

## Reihenfolge für den Plan

0. faketool, Welten, Aufzeichnung
1. `child`
2. Schema
3. Presets, Bereiche in `detect`, `tests_when`
4. Plan, Lauf, Bericht
5. `check` und `--show`
6. `gocover`; `dev covergate` entfällt
7. post-edit auf `[verify]`
8. **Halt:** der Mensch trägt `[verify.go]` und die Policy-Regeln ein
9. Selbstnutzung, `ci/gate.sh`, `.githooks/pre-commit`
10. Import und Vergleichsklasse `lanes`
11. Abweichungsliste
12. Mutanten
13. Messen
14. Doku

## Nachträge

Eingetragen am 2026-09-19. 1–7 hat der Plan beim Rechnen gegen den Code
gefunden (Abschnitt „Nachträge an die Spec" in
`plans/2026-09-19-loomux-stufe-2a.md`), 8–16 hat die Umsetzung entschieden,
17–22 hat der Abgleich der Doku mit dem Code gefunden, 23–27 die
Abschlussprüfung des Zweigs. Maßgeblich ist der Code
in `internal/verify`, `internal/cli/check.go` und `internal/hooks/post_edit.go`.

1. **`shell`, `sql`, `rust` haben doch Presets.** post-edit lintete sie schon
   (`shellcheck`, `sqlfluff`, `cargo clippy`/`cargo fmt --check`). Sie laufen
   aber nur, wenn der Stack erkannt oder konfiguriert ist. `shell` kennt nur
   `on_file`; im `check`-Scope ist `lint/shell` deshalb `not-applicable`.
2. **`cpp.types` ist `cmake --build build --parallel`**, wie post-edit es fuhr.
   Altfehler 1 wird damit „alt Traceback, neu der Compiler als Typprüfer",
   nicht `not-applicable`.
3. **`clang-format` prüft, statt umzuschreiben.** post-edit rief
   `clang-format -i <datei>` und schrieb die Datei des Agenten um. Das Preset
   ruft `clang-format --dry-run --Werror {file}`. Eintrag 17 der
   Abweichungsliste.
4. **Die Wiki-Lane baut `internal/hooks`, nicht `verify`.** `verify.Job`
   trägt dafür ein Feld `Fn`, eine Funktion, die im Prozess läuft. Die
   Abhängigkeit lautet damit `hooks → verify, brain/wiki`; zu `verify` siehe
   Nachtrag 22.
5. **`{file}` ist relativ zum Arbeitsverzeichnis der Lane** (ihrem Bereich),
   mit Schrägstrichen. `{pkgdir}` entfällt: Die Lane läuft ohnehin im Bereich,
   den `detect` für `typescript` liefert.
6. **Die Workspace-Skripte fallen weg.** post-edit rief in einem Bereich
   `npm --prefix D run typecheck`, `run lint` und `run check`. Die Presets
   rufen `npx tsc --noEmit`, `npx eslint`, `npx svelte-check` direkt. Ein Wirt
   mit eigenem Skript schreibt es in `[verify.typescript]`. Eintrag 18 der
   Abweichungsliste.
7. **`{coverprofile}` fehlt zur Laufzeit:** Vor dem Start der
   Coverage-Befehle prüft `Run` mit `os.Stat` jede Datei, die `Plan` als
   `Job.Reads` einträgt (die aufgelösten Pfade von `{coverprofile}` und
   `{coverdata}`); fehlt eine, ist die Lane `failed` mit
   `<pfad> is missing: the measuring run did not write it`.
8. **Aufräumen ohne Wettlauf.** „Jeder Lauf räumt, was nicht seine
   `<lauf-id>` trägt" hätte zwei gleichzeitige Läufe (ein `check` neben dem
   Pre-Commit-Tor) einander die Profile löschen lassen. `CleanCover` löscht
   fremde Dateien erst, wenn sie 24 h alt sind; die eigenen nur bei grünem
   Lauf. Aufgeräumt wird nur von `check`; post-edit misst im Profil `edit`
   nicht.
9. **`check --show` druckt kein Budget.** `check` hat keins; das Budget von
   post-edit ist das Hook-Flag `--budget` (Vorgabe 50 s,
   `hooks.DefaultBudget`). Die Kopfzeile nennt `max_parallel` und `timeout`.
10. **`--show` markiert die Herkunft je Lane, nicht je Schlüssel.** `Resolved`
    trägt eine Herkunft je Lane: Eine Tabelle, die einen Schlüssel ändert,
    markiert alle Schlüssel der Lane als `# config`.
11. **Erbregel verfeinert.** Eine Coverage-Lane, deren `test` abgeschaltet ist
    (`not-applicable`), zählt den Vorgänger als nicht angefordert und misst
    selbst über `measure`; ohne `measure` ist sie rot mit
    „`test` did not run and there is no measure step". Geerbt wird nur
    `unavailable` (keine Tests), im Edit-Scope zusätzlich `budget`,
    `missing-tool` und `unready`. Wer `test` abschaltet, sagt nichts darüber,
    ob es Tests gibt.
12. **Edit-Scope.** `project` fährt sein `on_file` nur neben einem aktiven
    Stack der bearbeiteten Datei. Eine unbekannte Endung oder eine, deren Stack
    nicht aktiv ist, bekommt gar keine Jobs, auch keine von `project`. Die
    weite Kette von 1a („jede andere Endung startet alle Lanes") entfällt.
13. **Go-Lint im Edit-Scope.** Das Go-Preset hat
    `on_file = ["go vet ./...", "{loomux} check gofmt {file}"]`: Ein Edit
    formatiert nur die bearbeitete Datei; eine unformatierte Datei anderswo
    ist nicht Sache dieses Edits. `commands` prüft mit `check gofmt .` alles.
14. **Lauf-IDs sind UTC.** `NewRunID` formatiert die Startzeit in UTC; die
    lokale Zone zu laden kostete unter Windows rund 18 ms je Lauf.
15. **Anfrage zuerst, Flags danach.** `loomux check all --show`,
    `loomux check precommit --root DIR`. `flag` hört beim ersten Argument auf,
    das keins ist; `check --show` allein ist ein Aufruffehler (Exit 2). Die
    Form `check --show [profil|arten]` aus dem Abschnitt oben gilt nicht.
16. **`loomux status` leitet die Lanes ab.** Es listet die Lanes des Profils
    `edit` je aktivem Stack aus `[verify]` und den Presets, mit Herkunft, und
    prüft deren Werkzeuge gegen den `PATH`; eine fest verdrahtete Liste gibt es
    nicht mehr.
17. **`check gocover` hat `--dir`.** `--dir <verzeichnis>` (Vorgabe `.`) nennt
    das Verzeichnis mit `go.mod`; das Profil und die Quelldateien werden
    relativ dazu gelesen, `go tool cover` läuft dort. Die Ausnahme
    `//coverage:exempt <grund>` muss die letzte Kommentarzeile direkt über
    `func` sein.
18. **`--show` ist nur teilweise kopierbar.** Die definierten Tabellen lassen
    sich nach `.loomux/config.toml` übernehmen. Eine nicht definierte Lane
    druckt `[verify.<stack>.<art>]  # not defined`; eingefügt ist das eine
    leere Tabelle und ein Ladefehler (`[verify.go].types is empty`, gemessen am
    2026-09-19). Wer eine Lane abschalten will, schreibt `<art> = false`.
    `loomux init` muss in Stufe 4 diese Zeilen auskommentiert schreiben.
19. **Die Testsuche überspringt auch `.loomux`**, neben `.git`, `vendor`,
    `node_modules` und `third_party`, und sucht je Bereich, nicht nur im
    Wurzelverzeichnis.
20. **Form von `presets.toml`.** Die Presets stehen unter `[stack.<name>]`,
    Varianten unter `[[stack.<name>.variant]]`, dazu `ignored` und
    `[extensions]` (die Endungstabelle). Das Go-Preset lintet mit
    `{loomux} check gofmt .`, nicht `cmd internal`. Die Varianten sind
    `pyright` (Python-`types`) und `biome` (TypeScript-`lint`).
21. **Selbstnutzung in Tabellenform.** Der Mensch hat
    `[verify.go.lint] commands = […]` eingetragen, nicht
    `[verify.go] lint = […]`: Die Tabelle behält `on_file` und `threaded` des
    Presets, die String-Form hätte beides verworfen.
22. **Abhängigkeiten, gegen `go list` gerechnet:** `verify → child, detect,
    shellwords` (dazu `config`, das jedes Paket benutzen darf), `child →
    gitenv`. Plan-Nachtrag 4 nannte `verify → child, gitenv, detect,
    shellwords`; `gitenv` erreicht `verify` aber nur über `child`.
23. **Das Cover-Verzeichnis legt `verify.PrepareCover` an, für `check` und
    post-edit.** Nachtrag 8 („post-edit misst im Profil `edit` nicht") hielt
    nur für die Vorgabe: Ein Profil `edit = ["lint", "test", "coverage"]`
    scheiterte an jedem Edit mit „Pfad nicht gefunden", und der Agent konnte
    das Verzeichnis nicht anlegen, weil die Policy Schreiben unter
    `.loomux/state/**` verweigert. Beide Aufrufer rufen `PrepareCover` vor
    `Run`; scheitert es, endet `check` mit 1 und post-edit mit 1
    (`ExitInternal`), ehe ein Werkzeug startet. post-edit räumt danach wie
    `check` mit `CleanCover` auf: eigene Dateien nur bei grünem Edit, fremde
    nach 24 h; ein Fehler dabei ist eine Warnung auf stderr, kein Urteil.
24. **`coverage` wartet nur auf ein `test`, das schreibt, was es liest.**
    `[verify.go] test = "go test ./..."` verwirft `measuring` des Presets; die
    Ladeprüfung lässt das durch, weil `coverage.measure` das Profil schreibt.
    `settle` band `coverage` trotzdem an `test`, `measure` lief nie, und die
    Lane war rot mit „… is missing". Jetzt bindet `settle` nur, wenn der
    Vorgänger, wie geplant, einen der Pfade aus `Reads` in seinem argv oder
    seiner Umgebung nennt (Python schreibt über `COVERAGE_FILE`), wenn er
    einen geplanten Zustand trägt (`unavailable`, `unready`: der wird geerbt)
    oder wenn `coverage` gar nichts liest (dann zählt nur die Reihenfolge).
    Sonst misst `coverage` selbst mit `measure`; ohne `measure` ist es rot mit
    „`test` does not write what this lane reads and there is no measure step".
    Die Rot-Regel aus Nachtrag 11 gilt nur für ein `coverage`, das eine
    Coverage-Datei liest.
25. **Die Testsuche überspringt jedes Punktverzeichnis**, wie `detect` es
    tut, dazu `vendor`, `node_modules`, `third_party`, `venv`, `build`,
    `target` und `dist`; das ersetzt die Liste aus Nachtrag 19. Anlass:
    `.venv/Lib/site-packages/*/tests` ließ ein Python-Projekt ohne eigene
    Tests getestet aussehen.
26. **Als konfiguriert gilt nur ein `test`, das den Befehl schreibt.**
    `Effective.Configured` übersprang die Testsuche für jede
    `[verify.<stack>].test`-Angabe, auch für eine Tabelle, die nur `measuring`
    ändert (so die Selbstnutzung). Jetzt zählt nur die String- oder
    Listenform oder eine Tabelle mit `commands`, und nie `false`.
27. **`--show` lädt zurück.** Das ersetzt Nachtrag 18. Eine Lane ohne Befehl
    druckt `# [verify.<stack>.<art>] not defined`, als ganze Kommentarzeile;
    eine abgeschaltete Lane, die vorher ebenfalls „not defined" hieß, steht
    als `<art> = false  # <herkunft>` in einer Tabelle `[verify.<stack>]` vor
    den Tabellen der übrigen Lanes. Die Ausgabe lädt über `ParseConfig`
    ohne Fehler, und beide Lanes kommen unverändert zurück
    (`TestWriteShowLoadsBack`). `loomux init` kann sie in Stufe 4 so
    schreiben.
28. **Schreiber ist nur ein Test-Job in messender Form oder dessen argv den
    gelesenen Pfad nennt.** Das verfeinert Nachtrag 24. `planJob` gibt jedem
    Python-Job `COVERAGE_FILE=<coverdata>` in die Umgebung, messend oder
    nicht; der Abgleich über die Umgebung band `coverage` deshalb immer an
    `test`. Mit `[verify.python] test = "uv run pytest"` lief `measure` nie,
    und die Lane war bei jedem Lauf rot („No data to report"). Die Umgebung
    zählt nicht mehr; `COVERAGE_FILE` bleibt, `coverage run` braucht es.
    Weil das Preset-`coverage` von Python die Datendatei nur über die
    Umgebung liest und keinen Pfad nennt, zählt `<coverdata>` für eine
    Python-Coverage-Lane beim Binden als gelesen, sofern ein argv mit
    coverage.py berichtet (das Wort `coverage`, dahinter `report`, `xml`,
    `json`, `html` oder `lcov`); ein Override wie `uv run pytest --cov=src`
    liest nichts und läuft wie vorher. Das landet aber nicht in
    `Reads`: `Run` prüft es nicht mit `os.Stat`, das Verhalten eines Laufs,
    der misst, bleibt gleich. Ein `coverage` ohne jeden gelesenen Pfad
    wartet weiter in jedem Fall (so C++, `gcovr` nach `ctest`).
29. **Eine Lane kann Dateien verlangen: `needs`.** Portiert aus master
    a24b17b („skip the cmake lane when no build tree is configured"), dort
    fest am cmake-Befehl von post-edit. Hier ist es ein Lane-Schlüssel, nur
    in Tabellenform, in Presets wie in `[verify]`: eine Liste von Pfaden
    relativ zum Verzeichnis der Lane (`{area}`), jeder innerhalb davon. Die
    Presets geben ihn den C++-Lanes auf dem Build-Baum (`types` =
    `cmake --build`, `test` = `ctest`, `coverage` = `gcovr`) als
    `needs = ["build/CMakeCache.txt"]`; `lint` braucht keinen. Fehlt eine
    verlangte Datei, markiert `Plan` die Lane `unready` mit der Notiz
    `<datei> is missing: configure the build first` — wie Godot: im Edit
    laut übersprungen, im Check rot. Die String- und Listenform ersetzen die
    Lane ganz und tragen kein `needs`; eine Tabelle ändert es wie jeden
    anderen Schlüssel. `--show` druckt es, und es lädt zurück.
30. **Die Bench-Suite von master liest ihre Lanes aus den Presets.**
    `internal/dev/benchcorpus` rief `hooks.TargetCommandsForStacks` und
    `hooks.StackForExtension` über die feste Tabelle von post-edit, die
    diese Stufe entfernt. `hooks.StackForExtension` bleibt mit derselben
    Signatur und liest `Presets.Extensions` (Ladefehler: `"", false`);
    statt `TargetCommandsForStacks` liefert `hooks.EditLaneCommands(facts)`
    die Befehle, die post-edit im Profil `edit` (`lint`, `types`) ausführt:
    die Presets über `verify.Resolve` mit den erkannten Fakten aufgelöst, so
    dass eine Variante greift (pyright, biome), je Lane die Form für eine
    Datei, wo es sie gibt, sonst `commands`, Stack für Stack in Byte-Folge.
    `{loomux}` wird zu `loomux`, `{file}` bleibt stehen; die Lückenprüfung
    liest nur das Werkzeug. Ein `[verify]` des vermessenen Projekts zählt
    nicht; sie vergleicht also mit dem, was loomux ohne Konfiguration beim
    Edit ausführt. Die erste Fassung las die ganzprojektweiten `commands`
    ohne Varianten und prüfte C++ damit auf clang-tidy statt clang-format,
    Shell auf nichts und ein pyright-Projekt auf mypy.
