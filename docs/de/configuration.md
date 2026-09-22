# Loomux Konfigurations-Referenz

Dieses Dokument bietet eine vollständige Referenz für `.loomux/config.toml`, die zentrale Konfigurationsdatei für Loomux-Projekte.

---

## 1. Grundsätze der Konfiguration

1. **Vom Menschen gepflegt, vom Wächter geschützt**:
   > [!IMPORTANT]
   > `.loomux/config.toml` wird **niemals von einem KI-Agenten bearbeitet**. Die Schreibschranke blockiert jeden Schreibversuch eines Agenten auf `.loomux/config.toml`. Änderungen werden vorgeschlagen; ein Mensch schreibt und committet sie.
2. **Deterministisch & Strikt**:
   Alle regulären Ausdrücke und Pfad-Globs werden beim ersten Gebrauch kompiliert. Enthält eine Regel einen ungültigen Regex oder fehlt eine Begründung (`reason`), bricht Loomux sofort mit einer präzisen Fehlermeldung unter Nennung der exakten Zeile ab.
3. **Trennung von Konfiguration und Zustand**:
   - `.loomux/config.toml`: Versionierte, von Menschen definierte Richtlinien und Prüfketten.
   - `.loomux/state/`: Flüchtiger, maschinengeschriebener Zustand (Sitzungsdaten, Journal, Caches). Stets in `.gitignore`.

---

## 2. Die Konfigurations-Sektionen

### `[project]`
Metadaten zur Beschreibung des Projekts.

```toml
[project]
name = "loomux"
version = "0.1.0"
agents = ["claude", "antigravity", "cursor"]
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `name` | String | Projekt-Bezeichner für Namensräume und Registrierungen. |
| `version` | String | Optionale Versionsnummer. |
| `agents` | Array von Strings | Aktive Agenten-Harnesses, die durch `loomux init` angebunden werden. |

---

### `[policy.paths]` (Pfad-Schutzregeln)
Definiert Pfadmuster, die Coding-Agenten weder erstellen noch bearbeiten dürfen.

```toml
[policy.paths]
rules = [
  { match = [".env", ".env.*"], reason = "Secrets und Umgebungsdateien dürfen nicht von Agenten bearbeitet werden" },
  { match = ["*.pem", "*.key", "id_rsa*"], reason = "Private Schlüssel und Zertifikate werden vom Menschen verwaltet" },
  { match = [".loomux/config.toml"], reason = "Die Wächter-Konfiguration ist vor Agenten-Schreibzugriffen geschützt" },
  { match = ["package-lock.json", "go.sum", "uv.lock"], reason = "Lockfiles werden durch Paketmanager verwaltet, nicht manuell" }
]
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `rules` | Array von Tabellen | Liste der Pfad-Inspektionsregeln. |
| `rules[].match` | String oder Array von Strings | Glob-Muster mit `**`-Unterstützung (z. B. `.aws/**`, `*.key`). |
| `rules[].reason` | String (**Pflichtfeld**) | Begründung, die dem Agenten bei einer Ablehnung angezeigt wird. |

---

### `[policy.commands]` (Befehlsausführungs-Regeln)
Definiert Muster für Shell-Befehle (`Bash`, `PowerShell`), die blockiert werden müssen.

Zwei Befehlsregeln sind eingebaut und brauchen hier keinen Eintrag: `git push` und jede Shell-Zeile, die `.loomux/config.toml` schreibt — eine Umleitung hinein, `sed -i`/`perl -i`, `tee`, `Set-Content`/`Add-Content`/`Out-File`, `cp`/`mv`/`Copy-Item` darauf oder das Löschen der Datei. Lesen (`cat`, `grep`, `Get-Content`) bleibt erlaubt. Die Regel liest den Befehlstext; ein Pfad in einer Variablen entgeht ihr.

```toml
[policy.commands]
rules = [
  { regex = '(^|\s)git\s+push(\s|$)', reason = "Commits zum Remote zu pushen erfordert eine menschliche Entscheidung" },
  { regex = '(^|\s)rm\s+-rf\s+/', reason = "Zerstörerische Wurzelverzeichnis-Löschbefehle sind verboten" },
  { regex = '(^|\s)pip\s+install\s+-r', reason = "Abhängigkeiten müssen über Lockfiles und Workflows verwaltet werden" }
]
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `rules` | Array von Tabellen | Liste der Befehls-Inspektionsregeln. |
| `rules[].regex` | String (**Pflichtfeld**) | Go RE2-kompatibler regulärer Ausdruck. |
| `rules[].reason` | String (**Pflichtfeld**) | Begründung, die dem Agenten bei einer Ablehnung angezeigt wird. |

---

### `[verify]` (Prüfketten & Quality-Gates)
Sagt, was `loomux check` und der post-edit-Hook fahren. **Ohne jedes
`[verify]` gelten die eingebauten Presets** für jeden Stack, den die Erkennung
findet; der Abschnitt ändert nur, was abweicht. `loomux check <profil> --show`
druckt die wirksame Tabelle (siehe
[CLI-Referenz](cli-reference.md#loomux-check-anfrage---root-pfad---show--v)).

```toml
[verify]
max_parallel = 8        # Prozesse gleichzeitig; Vorgabe: Zahl der CPUs
timeout      = 600      # Sekunden je Befehl; Vorgabe 600, keine Obergrenze

[verify.profiles]       # eingebaut: edit = [lint, types], precommit = stop = alle vier
edit      = ["lint", "types"]
precommit = ["lint", "types", "test", "coverage"]
stop      = ["lint", "types", "test", "coverage"]   # was das Stop-Tor am Rundenende fährt

[verify.go]             # je Stack; nicht genannte Stacks behalten ihr Preset
lint     = ["go vet ./...", "{loomux} check gofmt cmd internal"]
coverage = "{loomux} check gocover --profile {coverprofile} --floor 90"

[verify.typescript.lint]            # Tabellenform
commands = ["npx eslint ."]
on_file  = ["npx eslint --cache {file}"] # was post-edit fährt; fehlt es, gilt commands
threaded = true

[verify.cpp]
types = false           # Lane abschalten, auch gegen ein Preset

[verify.project]        # projektweit, ohne Stack, im Wurzelverzeichnis
lint = "make lint"
```

| Schlüssel | Typ | Beschreibung |
|---|---|---|
| `max_parallel` | positive Ganzzahl | Obergrenze gleichzeitig laufender Kindprozesse über alle Lanes. Vorgabe: Zahl der CPUs. |
| `timeout` | positive Ganzzahl | Sekunden, die jeder Befehl laufen darf. Vorgabe 600, keine Obergrenze. Der post-edit-Hook und das Stop-Tor haben zusätzlich ein Budget für den ganzen Lauf (`--budget`, Vorgabe 50 s bzw. 270 s); `loomux check` hat keins. |
| `profiles.<name>` | Liste von Arten | Eine benannte Menge von Arten. `edit` (post-edit), `precommit` (das Pre-Commit-Tor) und `stop` (das Stop-Tor am Rundenende) sind eingebaut und überschreibbar, aber nicht zu entfernen. Eine leere Liste, eine unbekannte Art oder ein reservierter Name ist ein Ladefehler. |
| `<stack>.<art>` | String, Liste, `false` oder Tabelle | Wie eine Art für einen Stack läuft; siehe unten. |
| `gdscript.import_check` | Boolean | Vorgabe `true`: `test` und `coverage` von GDScript sind `unready`, bis der Godot-Editor das Projekt importiert hat (`.godot/global_script_class_cache.cfg`). |

Jeder andere Schlüssel ist ein Ladefehler, der Datei und Schlüssel nennt, auf
jeder Ebene: `[verify]`, `[verify.<stack>]` und `[verify.<stack>.<art>]`.
Ebenso die alte Form auf oberster Ebene, `[verify].types = "…"`; die Meldung
verweist auf `[verify.<stack>].types`.
Regeln für Commit-Nachrichten gehören nicht zu `[verify]`: `[verify.commit]` wird
wie jeder unbekannte Stack abgewiesen. Die Commit-Policy wird in der
eigenen Top-Level-Tabelle `[commit]` konfiguriert (siehe unten).

#### Arten, Profile und reservierte Namen

Es gibt vier Arten, in dieser Reihenfolge: `lint`, `types`, `test`,
`coverage`. Eine Anfrage an `loomux check` ist ein Profil, `all` (immer alle
vier Arten) oder eine Komma-Liste von Arten (`lint,types`). Die Namen `gofmt`,
`commit-msg`, `gocover`, `all`, `lint`, `types`, `test` und `coverage` sind
reserviert; ein Profil mit einem davon ist ein Ladefehler. `stop` ist nicht
reserviert: es ist ein eingebautes Profil wie `edit` und `precommit`, und ein
Projekt, dessen Suite für jedes Rundenende zu langsam ist, engt es ein
(`stop = ["lint", "types"]`).

#### Stacks

`[verify.<stack>]` gibt es für `go`, `python`, `typescript`, `vue`, `svelte`,
`css`, `html`, `gdscript`, `cpp`, `shell`, `sql`, `rust`, dazu `wiki` und
`project`. Jeder andere Name ist ein Ladefehler.

- **Wo ein Stack läuft.** Die Erkennung liest das Wurzelverzeichnis und eine
  Ebene darunter (`go.mod`, `pyproject.toml`, `CMakeLists.txt`,
  `tsconfig.json` neben `package.json`, `project.godot`, `Cargo.toml`,
  `*.sh`, …). Jedes Verzeichnis, in dem ein Stack gefunden wurde, ist ein
  **Bereich**: `.` für die Wurzel, sonst das Verzeichnis der obersten Ebene.
  Zwei Bereiche ergeben zwei Lanes (`lint/typescript@admin`,
  `lint/typescript@web`), jede läuft in ihrem Bereich. Einen Override je
  Bereich gibt es nicht.
- **Ein konfigurierter Stack gilt als erkannt.** Ein `[verify.<stack>]`, das
  mindestens einer Art einen Befehl gibt, läuft im Wurzelverzeichnis, auch wo
  die Erkennung nichts fand.
- `.js` und `.jsx` gehören zu `typescript`; `javascript` gibt es nicht.
  Godot-Code heißt `gdscript`.
- `wiki` kennt nur `lint = false`; das schaltet beide Wiki-Lanes ab, die im
  eigenen Prozess von loomux laufen: den Lint der bearbeiteten Seite im
  post-edit-Hook und `lint/wiki`, den Lint über das ganze Bündel, den
  `loomux check` und das Stop-Tor dort anhängen, wo `lint` angefragt ist und
  das Projekt ein Wiki hat. `lint/wiki` prüft nur die Struktur des Bündels und
  steht im Bericht hinter jeder anderen Lane. Die Drift-Regel — Code geändert,
  Doku nicht — bleibt bei `loomux wiki-gate`, einem eigenen Befehl.
- `project` hat kein Preset. Seine Lanes laufen für `loomux check` im
  Wurzelverzeichnis; der post-edit-Hook fährt sie nur mit `on_file` und nur
  neben den Lanes einer bearbeiteten Datei, deren Stack aktiv ist.

#### Die Formen einer Art

Eine Stack-Tabelle kennt die Schlüssel `lint`, `types`, `test`, `coverage`
(und `import_check` in `[verify.gdscript]`). Jede Art ist eins von:

| Form | Beispiel | Bedeutung |
|---|---|---|
| String | `lint = "make lint"` | ein Befehl; **ersetzt die ganze Lane** |
| Liste | `lint = ["go vet ./...", "{loomux} check gofmt ."]` | mehrere Befehle; **ersetzt die ganze Lane** |
| `false` | `types = false` | schaltet die Lane ab, auch gegen ein Preset |
| Tabelle | `[verify.go.test]` mit `measuring = "…"` | **führt Schlüssel für Schlüssel** mit dem Preset zusammen |

Die Schlüssel der Tabellenform:

| Schlüssel | Typ | Beschreibung |
|---|---|---|
| `commands` | Liste | Was `loomux check` fährt. |
| `on_file` | Liste | Was der post-edit-Hook für eine Datei fährt; fehlt es, fährt er `commands`. |
| `threaded` | Boolean | Die Befehle nebeneinander statt nacheinander fahren. |
| `measuring` | String | Nur `test`/`coverage`. Der eine Befehl, den `test` statt `commands` fährt, wenn `coverage` im selben Lauf steht. |
| `measure` | String | Nur `test`/`coverage`. Der eine Befehl, den `coverage` zuerst fährt, wenn sein Vorgänger nicht im Lauf steht. |
| `after` | Art | Nur `test`/`coverage`. Die Art desselben Stacks, auf die diese Lane wartet. Zyklen sind Ladefehler, die Meldung nennt den Ring. |
| `needs` | Liste | Dateien, relativ zum Verzeichnis der Lane und darin, ohne die ihre `commands` nichts bedeuten. Fehlt eine, ist die Lane `unready`: im Edit übersprungen und genannt, im Check rot. `on_file` bewachen sie nicht: ein Edit, der die Form einer Lane für eine Datei ausführt, führt sie in jedem Fall aus. |

- **Ersetzen oder zusammenführen.** Ein String oder eine Liste steht für die
  Lane, wie sie dasteht: `measuring`, `measure`, `on_file` und `needs` des
  Presets gelten nicht mehr, nur `after` bleibt. Eine Tabelle ändert nur die
  Schlüssel, die sie nennt: `[verify.go.test] measuring = "…"` behält
  `commands` aus dem Preset.
- Ein Schlüssel ist in TOML entweder Wert oder Tabelle: `test = "…"` und
  `test.measuring = "…"` in derselben Tabelle sind ungültig. Dann die
  Tabellenform mit `commands` nehmen.
- `true`, eine leere Liste, ein leerer Befehl und eine offene Anführung sind
  Ladefehler.
- **Befehle sind argv, keine Shell.** Jeder Befehl wird nach Shell-Wortregeln
  zerlegt und direkt gestartet; es gibt kein `cmd /c` und kein `sh -c`. Pipes,
  `&&` und Globbing durch eine Shell gibt es nicht.
- Bevor ein Befehl startet, muss sein Werkzeug auf dem `PATH` liegen, auch bei
  konfigurierten Befehlen; sonst ist die Lane `missing-tool`.

#### Platzhalter

Ersetzt werden nur diese Namen, jeweils innerhalb eines Arguments; alle
anderen Klammern bleiben wörtlich (`-run 'Test{A,B}'`).

| Platzhalter | Wert |
|---|---|
| `{file}` | Die bearbeitete Datei, relativ zum Bereich der Lane, mit Schrägstrichen. **Nur in `on_file`**; anderswo ein Ladefehler. |
| `{area}` | Das absolute Verzeichnis des Bereichs der Lane. |
| `{coverprofile}` | `.loomux/state/cover/<lauf-id>-<stack>-<bereich>.out`, eine je Lauf und Lane. |
| `{coverdata}` | Dasselbe mit `.data`. Python-Lanes bekommen zusätzlich `COVERAGE_FILE={coverdata}` in ihre Umgebung. |
| `{loomux}` | Das laufende Binary. So findet das Go-Preset `gocover` auch dort, wo loomux nicht auf dem `PATH` liegt. |

- Eine `coverage`-Lane, die `{coverprofile}` liest, während weder `test`,
  `test.measuring` noch `coverage.measure` desselben Stacks es schreibt, ist
  ein Ladefehler.
- Fehlt eine Coverage-Datei, wenn die Lane startet, ist sie `failed`:
  `<path> is missing: the measuring run did not write it`.
- **Aufräumen.** `loomux check` und der post-edit-Hook legen beide
  `.loomux/state/cover/` an, bevor eine Lane startet, und räumen danach gleich
  auf. Ein grüner Lauf löscht am Ende seine eigenen
  Coverage-Dateien; ein roter lässt sie zum Nachsehen liegen. Dateien anderer
  Läufe werden gelöscht, sobald sie 24 Stunden alt sind, damit zwei Läufe
  nebeneinander einander nie die Profile löschen. Einen Pfad außerhalb von
  `.loomux/state/cover/` berührt loomux nie.

#### Presets

Die Presets sind ins Binary eingebettet (`internal/verify/presets.toml`) und
benutzen das Schema oben; ein Preset kann also nichts sagen, was ein Projekt
nicht auch sagen könnte. Die Schichten sind: Preset, dann die erste
**Variante**, deren Signal die Erkennung fand, dann `[verify.<stack>]`.

| Stack | `lint` | `types` | `test` | `coverage` |
|---|---|---|---|---|
| go | `go vet ./...`, `{loomux} check gofmt .` (parallel) | — | `go test ./... -count=1` | `{loomux} check gocover --profile {coverprofile}` |
| python | `uvx ruff check . --output-format=concise` | `uv run mypy --no-error-summary --no-pretty`; mit `pyright`: `uv run pyright` | `uv run pytest -q --tb=short --no-header` | `uv run coverage report --skip-covered --skip-empty -m` |
| typescript | `npx eslint .`; mit `biome`: `npx biome check .` | `npx tsc --noEmit` | `npx vitest run` | `npx vitest run --coverage` |
| vue | — | `npx vue-tsc --noEmit` | — | — |
| svelte | — | `npx svelte-check` | — | — |
| css | `npx stylelint **/*.{css,scss}` | — | — | — |
| html | `npx htmlhint **/*.html` | — | — | — |
| gdscript | `uvx gdlint .` | — | `godot --headless --quit` | — |
| cpp | `clang-tidy -p build` | `cmake --build build --parallel` | `ctest --test-dir build --output-on-failure` | `gcovr --root . --object-directory build --fail-under-line 100 --txt` |
| shell | nur `on_file` | — | — | — |
| sql | `sqlfluff lint .` | — | — | — |
| rust | `cargo clippy -- -D warnings`, `cargo fmt --check` | — | — | — |

- Die `on_file`-Formen: go `go vet ./...` und `{loomux} check gofmt {file}`
  (ein Edit formatiert nur seine eigene Datei), gdscript `uvx gdlint {file}`,
  cpp `clang-format --dry-run --Werror {file}` (prüft, schreibt nie um),
  typescript `npx eslint --cache {file}` (biome: `npx biome check {file}`),
  css `npx stylelint {file}`, html `npx htmlhint {file}`, shell
  `shellcheck {file}`, sql `sqlfluff lint {file}`.
- `lint`, `types`, `test` und `coverage` von cpp tragen `needs =
  ["build/CMakeCache.txt"]`: solange der Build-Baum nicht konfiguriert ist,
  sind sie `unready` mit `build/CMakeCache.txt is missing: configure the build
  first`. Ein Edit führt das `clang-format` der Lint-Lane auf der Datei
  trotzdem aus; es braucht keinen Build-Baum. Konfigurieren ist Sache des Projekts (Generator, Optionen,
  Toolchain), loomux rät keinen Konfigurationsschritt.
- **Messen.** `test` misst nur, wenn `coverage` im selben Lauf steht (go:
  `-covermode=set -coverprofile={coverprofile}`, python:
  `uv run coverage run -m pytest …`); allein bleibt es der schnelle Weg.
  `coverage` läuft `after = "test"`, wartet aber nur dann auf `test`, wenn die
  Test-Lane, wie sie für diesen Lauf geplant ist, eine Datei schreibt, die
  `coverage` liest: Sie läuft in ihrer Form `measuring`, oder ihr Befehl nennt
  das `{coverprofile}` oder `{coverdata}`, das `coverage` liest. Ein
  Python-`coverage`, das mit coverage.py berichtet (`coverage report`, `xml`,
  `json`, `html` oder `lcov`), liest `{coverdata}` über `COVERAGE_FILE`, auch
  wenn sein Befehl es nicht nennt; da jede Python-Lane diese Variable
  bekommt, sagt die Umgebung nichts darüber, wer schreibt. Ein `coverage`,
  das keine solche Datei liest, wartet in jedem Fall auf `test`. Sonst, und
  allein angefragt, misst `coverage` selbst mit `measure`; so bekommt auch
  `[verify.go] test = "go test ./..."`, das `measuring` des Presets verwirft,
  sein Profil. Ein `coverage` ohne `measure`, das eine Coverage-Datei liest,
  die im Lauf niemand schreibt, ist rot: „`test` did not run and there is no
  measure step" oder „`test` does not write what this lane reads and there is
  no measure step". Eins, das keine Coverage-Datei liest, läuft einfach.
- **Die Schwelle steht im Tor:** `--floor` bei `gocover`, `fail_under` in
  `pyproject.toml` für Python, `--fail-under-line` bei gcovr. Nur Go prüft je
  Funktion (100 %, außer `//coverage:exempt <grund>` steht über `func`).
- Das Go-Preset misst ohne `-coverpkg`, weil es den Modulpfad nicht kennt; es
  untertreibt deshalb bei Tests über Paketgrenzen. Ein Projekt nennt
  `-coverpkg` in seinem `test.measuring` und `coverage.measure`, wie loomux
  selbst.
- GDScript hat kein Coverage-Preset.

#### Testerkennung

`test` und `coverage` laufen nur, wo es Tests gibt. Gesucht wird rekursiv in
jedem Bereich, ausgenommen jedes Verzeichnis, dessen Name mit einem Punkt
beginnt (`.git`, `.loomux`, `.venv`, `.tox`, …), sowie `vendor`,
`node_modules`, `third_party`, `venv`, `build`, `target` und `dist`: Die Tests
installierter Pakete machen ein Projekt nicht getestet. Gesucht wird nur, wenn `test` oder `coverage` angefragt ist, das
Profil `edit` läuft also nie durch den Baum.

| Stack | Signal |
|---|---|
| go | eine Datei `*_test.go` |
| python | ein Verzeichnis `tests/`, eine Datei `test_*.py` oder `[tool.pytest` in `pyproject.toml` |
| cpp | `enable_testing(` in `CMakeLists.txt` |
| typescript | `vitest` in `package.json` |

- Keine Tests: `test` und `coverage` sind `unavailable`.
- Ein `[verify.<stack>].test`, das den Befehl hinschreibt, übergeht die Suche:
  die String- oder Listenform oder eine Tabelle mit `commands`. Wer den Befehl
  hinschreibt, hat Tests. `test = false` und eine Tabelle, die nur
  `measuring`, `threaded` oder einen anderen Schlüssel ändert, behalten die
  Suche.
- Ein Stack ohne Signal (`gdscript`, `project`) fährt seine Lane, sobald sie
  definiert ist.

#### Zustände und Urteil

| Zustand | wann | `loomux check` | post-edit |
|---|---|---|---|
| `ok` | alle Befehle Exit 0 | grün | grün |
| `failed` | Exit ≠ 0, Startfehler, Ausgabe abgebrochen, Coverage-Datei fehlt | rot | rot, Exit 2 |
| `timed-out` | eigenes `timeout` | rot | rot, Exit 2 |
| `budget` | Laufbudget erschöpft | – | übersprungen, genannt |
| `blocked` | die Lane, auf die sie wartet (`after`), ist rot | rot | rot, Exit 2 |
| `missing-tool` | ein Werkzeug liegt nicht auf dem `PATH` | rot | übersprungen, genannt |
| `unready` | Godot hat das Projekt nicht importiert, oder eine Datei aus `needs` fehlt | rot | übersprungen, genannt |
| `unavailable` | Art definiert, kann nicht laufen (keine Tests gefunden) | neutral, zählt als „nichts lief" | nicht angezeigt |
| `not-applicable` | Art im Stack nicht definiert oder `false` | neutral, angezeigt | nicht angezeigt |

- **Was eine Lane erbt.** Konnte die Lane, auf die sie wartet, nicht laufen,
  übernimmt eine Lane deren Zustand (`unavailable`, `not-applicable`, im
  Edit-Scope auch `budget`, `missing-tool`, `unready`), statt `blocked` zu
  werden. Ein abgeschaltetes `test` zählt als nicht angefragt: `coverage` misst
  dann selbst mit `measure`.
- **Urteil von `loomux check`, je angefragter Art:** Lief keine Lane einer Art
  und ist die Art nirgends `not-applicable`, druckt der Check
  ``nothing to check for `<art>` `` und endet mit Exit 1: Ein Tor, das nichts
  prüft, ist nicht grün. Sonst Exit 0, wenn keine Lane rot ist, und 1, wenn
  eine rot ist. Ein Ladefehler endet mit 1, ein fehlerhafter Aufruf mit 2.
- **Urteil des post-edit-Hooks:** Eine rote Lane endet mit Exit 2 und ihrer
  Ausgabe auf `stderr`; übersprungene Lanes nennt er in
  `hookSpecificOutput.additionalContext` auf `stdout`, Exit 0. Eine Datei,
  deren Endung kein aktiver Stack beansprucht, bekommt keine Lanes und endet
  mit 0.
- **Urteil des Stop-Tors:** das Profil `stop` im Check-Scope, die Zustände
  gelten also wie in der Spalte `loomux check`. Eine rote Lane endet mit Exit 2
  und hält die Runde an, nur die roten Lanes auf `stderr`; eine Lane, die das
  Budget nicht mehr erreichte, oder eine angefragte Art ohne etwas, das lief,
  endet mit Exit 1 — das Tor konnte nicht urteilen, und die Runde endet. Siehe
  [Hooks](hooks.md#stop).
- **Der Marker `.loomux/no-verify`.** Solange er existiert, lässt das Stop-Tor
  jede Runde enden, ohne die Kette zu fahren. Befunde von Subagenten werden
  trotzdem zugestellt und halten die Runde weiter an. Ein Mensch legt ihn an und entfernt ihn; die Policy
  verweigert einem Agenten den Pfad.

---

### `[commit]` (Commit-Nachrichten-Validierung)

Konfiguriert die Prüfregeln für `loomux check commit-msg` und den `.githooks/commit-msg`-Hook.
Fehlt `[commit]` in `.loomux/config.toml`, gelten die Standardwerte: `language = "en"`, `threshold = 2`, `conventional = true`.

```toml
[commit]
language     = "en"       # "en" (Standard) oder "de"
threshold    = 2          # Treffer in einer Zeile, ab denen sie abgelehnt wird (Standard: 2)
conventional = true       # Erzwingt Conventional-Commits-Format für Betreffzeile (Standard: true)

# Eine Zeile, auf die eines dieser Muster passt, wird ganz übersprungen
[[commit.allow]]
regex  = '(Müller|Zürich|Löwis)'
reason = "Eigennamen mit Umlaut: zwei davon in einer Zeile würden sie ablehnen"
```

| Schlüssel | Typ | Standard | Beschreibung |
|---|---|---|---|
| `language` | String | `"en"` | Zielsprache der Commit-Nachricht: `"en"` (Englisch) oder `"de"` (Deutsch). |
| `threshold` | Integer | `2` | Eine Zeile mit so vielen Treffern oder mehr wird abgelehnt. Muss `> 0` sein. |
| `conventional` | Boolean | `true` | Wenn `true`, muss die erste Zeile dem Conventional-Commits-Format `<type>[(<scope>)][!]: <description>` entsprechen. |
| `allow` | Liste von Tabellen | `[]` | Ausnahmeregeln. Jeder Eintrag muss `regex` (gültiges RE2-Muster) und `reason` (nicht-leerer String) enthalten. Eine Zeile, auf die das Muster passt, wird ganz übersprungen, samt ihrer Treffer. |

### `[worktree]` (Worktree-Spiegel)
Nennt git-ignorierte Verzeichnisse des Haupt-Checkouts, die `loomux worktree link` in einem verknüpften Git-Worktree über eine Windows-Junction bereitstellt. Gelesen wird immer die `.loomux/config.toml` des Haupt-Checkouts. Junctions gibt es nur unter Windows; Symlinks auf anderen Systemen gibt es nicht. Der Mechanismus steht unter [Hooks](hooks.md#9-worktree-spiegelung).

```toml
[worktree]
mirror = [
  "node_modules",
  ".cache",
  "vendor",
  "testdata/large-fixtures"
]
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `mirror` | Array von Strings | Pfade relativ zur Projektwurzel, gespiegelt, damit ein Worktree sie nicht neu bauen oder herunterladen muss. Ein Eintrag, der leer oder absolut ist oder das Projekt mit `..` verlässt, macht die Datei kaputt (Exit 1). Keine Tabelle, kein Schlüssel oder eine leere Liste heißt: nichts zu spiegeln. |

---

### `[graph]` — den gibt es nicht, und Stufe G2a hat geklärt, warum

Ein früherer Entwurf dieses Handbuchs spezifizierte einen Abschnitt `[graph]`
mit den Feldern `extensions`, `exclude` und `freshness_check`. Nichts davon
existiert im Code, den Stufe G2a ausgeliefert hat, und nichts davon ist
geplant. Die Frage kommt immer wieder, deshalb steht die Antwort hier, statt
neu entdeckt zu werden:

- **Welche Sprachen ein Bau parst, ergibt sich aus dem Extraktor, nicht aus
  einer Liste, die ein Repository erklärt.** `internal/code/extract/golang`
  ist ein Go-Extraktor; er parst `.go`-Dateien, weil er nur die lesen kann.
  Ein zweiter Extraktor für eine andere Sprache reiht sich auf dieselbe Art
  ein — indem er existiert —, und kein Konfigurationsschlüssel entscheidet,
  welcher auf eine Datei angewandt wird.
- **Einen einzelnen Bau auf einen Teilbaum einzuschränken ist eine Flagge des
  Aufrufs, keine Repository-Einstellung.** `loomux graph build` und `loomux
  graph check` nehmen bereits `--root`; eine künftige Einschränkungs-Flagge
  auf einem Aufruf ist die richtige Form für „heute nur diesen Teilbaum
  indizieren", weil diese Wahl zu dem gehört, der den Befehl ausführt, nicht
  zu dem Baum, der indiziert wird.
- **`freshness_check` hatte nie etwas zu entscheiden.** `internal/code/freshness`
  vergleicht immer zuerst Größe und Änderungszeit und greift erst auf einen
  Inhaltshash zurück, wenn die beiden nicht übereinstimmen — die Regel der
  Referenzimplementierung, keine Strategie, die ein Projekt abschalten könnte.
  Da bleibt nichts mehr zu konfigurieren.

Was einen Bau einschränkt oder formt, gehört zum Aufruf, der ihn ausführt,
nicht zur Konfiguration des Repositories, das indiziert wird. Kehrt dieser
Abschnitt zurück, dann weil eine reale Anforderung eine Einstellung auf
Abschnittsebene erzwungen hat — nicht weil das Handbuch einst eine skizziert
hatte.

---

### `[skills]` (Kuratierte Best-Practice-Suiten)
Konfiguriert sprachspezifische Review-Regeln und Synchronisationsziele.

```toml
[skills]
suites = ["review-go", "review-security", "review-typescript"]
sync = [".claude/skills", ".agents/skills"]
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `suites` | Array von Strings | Aktive Review-Suiten, die im Loomux-Binary mitgeliefert werden. |
| `sync` | Array von Strings | Zielordner, in die `SKILL.md`-Dateien abgelegt werden. |

---

### `[privacy]` (Datenschutz & Geheimnisschutz)
Garantiert, dass sensible Daten unter keinen Umständen an externe Modelle oder Logs weitergegeben werden.

```toml
[privacy]
mode = "manual_cloud"
never = [
  "**/.env*",
  "**/credentials.json",
  "**/*.pem",
  "**/*.key"
]
```

| Feld | Typ | Beschreibung |
|---|---|---|
| `mode` | String | `"local_only"`, `"manual_cloud"` (Vorgabe) oder `"automatic_cloud"`; jeder andere Wert wird abgelehnt. Ein `local_only`-Bereich existiert auf dem Cloud-Kanal nicht. |
| `never` | Array von Strings | Glob-Muster der Pfade, die kein Kanal erreicht. |

---

## 3. Vollständiges kommentiertes Muster (`config.toml`)

```toml
# ==============================================================================
# Loomux Projektkonfiguration: .loomux/config.toml
# ==============================================================================

[project]
name = "loomux"
version = "0.1.0-fusion"
agents = ["claude", "antigravity", "cursor"]

# --- Pfad-Schutzregeln (Exit-Code 2 bei Verstoß) -----------------------------
[policy.paths]
rules = [
  { match = [".env*", "*.pem", "*.key", "id_rsa*"], reason = "Secrets dürfen nicht von Agenten bearbeitet werden" },
  { match = [".loomux/config.toml"], reason = "Wächter-Regeln werden vom Menschen verwaltet und sind schreibgeschützt" },
  { match = [".loomux/no-verify"], reason = "Prüfschranken dürfen nicht durch Agenten umgangen werden" },
  { match = ["go.sum", "package-lock.json", "uv.lock"], reason = "Lockfiles werden durch Paketmanager verwaltet, nicht manuell" }
]

# --- Befehlsausführungs-Regeln (Exit-Code 2 bei Verstoß) --------------------
[policy.commands]
rules = [
  { regex = '(^|\s)git\s+push(\s|$)', reason = "Pushen zum Remote erfordert eine menschliche Entscheidung" },
  { regex = '(^|\s)rm\s+-rf\s+/', reason = "Löschung des System-Wurzelverzeichnisses ist verboten" }
]

# --- Prüfkette: loomux check und post-edit; alles Übrige ist Preset ---------
[verify.go.lint]
commands = ["{loomux} check gofmt cmd internal", "go vet ./..."]

[verify.go.test]
measuring = "go test ./... -count=1 -covermode=set -coverpkg=example.com/project/... -coverprofile={coverprofile}"

[verify.go.coverage]
measure = "go test ./... -count=1 -covermode=set -coverpkg=example.com/project/... -coverprofile={coverprofile}"

# --- Worktree-Isolierungsspiegel ---------------------------------------------
[worktree]
mirror = [
  "node_modules",
  ".cache",
  "bin"
]

# --- Kuratierte Sprach-Review-Suiten -----------------------------------------
[skills]
suites = ["review-go", "review-security"]
sync = [".claude/skills", ".agents/skills"]

# --- Datenschutz-Schranken ---------------------------------------------------
[privacy]
mode = "manual_cloud"
never = [".env*", "*.key", "credentials.json"]
```

---

## 4. Wo was liegt

| | Ort |
|---|---|
| Zustandsverzeichnis | unter Windows `%LOCALAPPDATA%\loomux`, sonst `~\AppData\Local\loomux`; auf anderen Systemen `$XDG_STATE_HOME/loomux`, sonst `~/.local/state/loomux` |
| Bereichsregistry | `<Zustandsverzeichnis>\registry.toml` |
| Manifest eines beschreibbaren Bereichs | `<Bereichspfad>\.loomux\config.toml` |
| Manifest eines lesenden Bereichs, wie die Schreibschranke es liest | `<Zustandsverzeichnis>\areas\<scope>\.loomux\config.toml` |
| Artefakte eines lesenden Bereichs (`index.md`, `graph.json`, `_identities.tsv`) und sein Manifest, wie `loomux brain` sie liest | `%LOCALAPPDATA%\brain\areas\<scope>\` bis Stufe 3 |
| Artefakte eines beschreibbaren Bereichs | sein `path` |
| Stempel des letzten Reconcile | `%LOCALAPPDATA%\brain\maintenance\last-run.txt` bis Stufe 3 |
| Sitzungszustand der Hooks (`base`, `blocks`, `green`) | `<projekt>\.loomux\state\hooks\<session_id>.json` |
| Schnappschüsse und Befunde der Subagenten | `<projekt>\.loomux\state\hooks\<session_id>\agents\<agent_id>.json` |
| Der Marker, der das Stop-Tor abschaltet | `<projekt>\.loomux\no-verify` |

`LOOMUX_STATE_DIR` überschreibt das Zustandsverzeichnis,
`LOOMUX_LEGACY_BRAIN_DIR` das Verzeichnis von ultra-brain; einen
Kommandozeilenschalter gibt es für keines von beiden. Das Altverzeichnis gibt
es, weil ultra-brain diese Artefakte noch schreibt: vor Stufe 3 hat loomux
keinen eigenen Indexer.

`<scope>` ist der Scope, zu einem Verzeichnisnamen geplättet: jede Folge von
Zeichen außer `A-Z a-z 0-9 _ . -` wird zu einem `-`, und Striche an beiden Enden
fallen weg; aus `project/loomux` wird `project-loomux`.

**Das Wissen liegt an keinem dieser Orte.** Es liegt dort, wohin `path` und
`wiki` eines Bereichs zeigen — in einem Repo oder in einem gewöhnlichen
Obsidian-Vault unter Git. loomux hält keine Kopie davon.

---

## 5. Die Schreibschranke und das Memory der Agenten

Die Schreibschranke (`loomux hook pre-tool-use`) lässt Werkzeuge schreiben, wo
die Registry einen Bereich erklärt. Ein verknüpfter Git-Worktree eines Bereichs
mit `workspace = true` gehört dazu und braucht keinen eigenen Eintrag: Er ist
dasselbe Repo ein zweites Mal ausgecheckt, und die Schranke erkennt ihn an Gits
eigenen Worktree-Dateien. Darüber hinaus ist immer an drei Orten offen —
für jeden Nutzer, ohne Eintrag in der Registry:

| Wo | offen ist, unterhalb von |
|---|---|
| Memory von Claude Code | `$CLAUDE_CONFIG_DIR/projects/<projekt>/memory/`, sonst `~/.claude/projects/<projekt>/memory/` |
| Scratchpad einer Claude-Code-Sitzung | `<temp>/claude/<projekt>/<sitzung>/scratchpad/` |
| Memory von Antigravity | `~/.gemini/{antigravity,antigravity-cli,antigravity-ide}/knowledge/` und `…/brain/<gesprächs-id>/` |

- Ein Aufruf, der nur dorthin schreibt, geht auch durch, wenn die Registry
  nicht lesbar ist.
- `.loomux/config.toml` bleibt auch dort gesperrt; über das Manifest wird vor
  allem anderen entschieden.
- Jeder Pfad wird vor dem Vergleich aufgelöst; ein Link aus dem Memory in einen
  gesperrten Baum wird also am Ziel beurteilt. In einem Aufruf, der auch
  anderswohin schreibt, bleibt ein Memory-Ziel in einer Verbotszone gesperrt.
- Den Rest von `~/.claude` und `~/.gemini` öffnet die Ausnahme nicht,
  `antigravity-backup` eingeschlossen: `settings.json`, Hooks und Plugins
  steuern den Agenten selbst, und wer sie umschreibt, schaltet seine eigenen
  Schranken ab.
- Claude Code lädt `MEMORY.md` in spätere Sitzungen desselben Projekts — was
  ein Agent dort schreibt, wirkt also wie eine Anweisung an künftige Sitzungen;
  ob Antigravity `knowledge/` ebenso lädt, ist nicht gemessen. Das ist mit der
  Entscheidung vom 2026-09-13 bewusst in Kauf genommen.

Eine Ablehnung mit `lies outside every writable tree` nennt, wo geschrieben
werden darf: die erlaubten Bäume, danach das Scratchpad der Sitzung und die
Memory-Bäume, sofern sie sich benennen lassen.
