# Loomux Konfigurations-Referenz

Dieses Dokument bietet eine vollständige Referenz für `.loomux/config.toml`, die zentrale Konfigurationsdatei für Loomux-Projekte.

---

## 1. Grundsätze der Konfiguration

1. **Vom Menschen gepflegt, vom Wächter geschützt**:
   > [!IMPORTANT]
   > `.loomux/config.toml` wird **niemals von einem KI-Agenten bearbeitet**. Die Schreibschranke blockiert jeden Schreibversuch eines Agenten auf `.loomux/config.toml`, und der Wächter verweigert einem Agenten die Befehle, die sie schreiben (`loomux init`, ein schreibendes `loomux config`, `loomux area add`). Änderungen werden vorgeschlagen; ein Mensch schreibt und committet sie, von Hand oder mit `loomux config` (siehe die [CLI-Referenz](cli-reference.md#10-konfiguration-loomux-config)).
2. **Deterministisch & Strikt**:
   Alle regulären Ausdrücke und Pfad-Globs werden beim Lesen der Policy geprüft. Eine Regel mit fehlendem oder leerem `match`, einem fehlerhaften Glob, fehlendem oder nicht kompilierbarem `regex` oder fehlender Begründung (`reason`) ist ein Fehler, der die Datei und die Nummer der Regel nennt (`[[policy.commands.rules]] #2 needs a reason`), und der Wächter verweigert dann den Werkzeugaufruf, statt ihn zu beurteilen.
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
| `name` | String | Von loomux nicht gelesen; der Namensraum kommt aus `[area] scope`. |
| `version` | String | Von loomux nicht gelesen. |
| `agents` | Array von Strings | Von loomux nicht gelesen. `loomux init` hält die eingerichteten Wirte in `.loomux/state/answers.toml`, nicht hier. |

---

### `[policy.paths]` (Pfad-Schutzregeln)
Definiert Pfadmuster, die Coding-Agenten weder erstellen noch bearbeiten dürfen.

Eingebaut, ohne Eintrag hier: `.env`, `.env.*`, `*.pem`, `*.key`, `id_rsa*`, `*.p12`, `.npmrc`, `.pypirc`, `credentials.json` und `.aws/**` (Geheimnisse); `.loomux/no-verify` und `.loomux/state/hooks/**` (die Steuerung des Stop-Tors); `uv.lock`, `poetry.lock`, `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, `Cargo.lock` und `go.sum` (Lockfiles). Die Regeln eines Projekts kommen hinzu und können diese nicht aufheben.

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
| `rules[].match` | String oder Array von Strings | Globs relativ zum Projektwurzelverzeichnis, in der Syntax von `path.Match`. Ein Muster ohne `/` wird nur mit dem Dateinamen verglichen (`*.key`, `.env.*`), eines mit `/` mit dem ganzen relativen Pfad, wobei `*` keinen `/` überspringt. `**` steht für beliebige Tiefe als Endung `/**` (`.aws/**`) und als führendes `**/`, das jeden Ordner meint, die Wurzel eingeschlossen (`**/secrets/*.txt`); an jeder anderen Stelle ist es ein einfaches `*`. Eine Regel gilt für das Ziel eines schreibenden Werkzeugs und für jeden Pfad, den eine Shell-Zeile schreibt oder löscht, gleichermaßen. |
| `rules[].reason` | String (**Pflichtfeld**) | Begründung, die dem Agenten bei einer Ablehnung angezeigt wird. |

---

### `[policy.commands]` (Befehlsausführungs-Regeln)
Definiert Muster für Shell-Befehle (`Bash`, `PowerShell`), die blockiert werden müssen.

Eine Befehlsregel ist eingebaut und braucht hier keinen Eintrag: `git push`. Auch `.loomux/config.toml` braucht keinen: sie ist eine eingebaute Pfadregel unter jedem Ordner, und die Pfadregeln prüfen jeden Pfad, den eine Shell-Zeile schreibt oder löscht, ebenso wie das Ziel eines schreibenden Werkzeugs — eine Umleitung hinein, `sed -i`/`perl -i`, `tee`, `Set-Content`/`Add-Content`/`Out-File`, `cp`/`mv`/`Copy-Item` darauf, das Löschen der Datei oder eines Ordners darüber. Lesen (`cat`, `grep`, `Get-Content`) bleibt erlaubt. Der Wächter liest die Wörter des Befehls; im Standardmodus entgeht ihm darum ein Pfad in einer Variablen, der strikte Modus verweigert ein Schreiben, dessen Expansion auf einem geschützten Pfad landen kann. Siehe [Hooks](hooks.md#7-der-entscheidungsweg-von-pre-tool-use).

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

### `[guard]` (Wie genau der Wächter liest)

```toml
[guard]
mode = "default"   # oder "strict"
```

`default` hält gegen einen Agenten, der sich irrt oder bequem ist: die
gängigen Schreib- und Löschbefehle, ihre Wrapper, Globs, Brace-Expansion und
die Ordner über einem geschützten Pfad. `strict` hält zusätzlich gegen einen,
der gezielt umgehen will: jedes Ziel wird vom Dateisystem aufgelöst (Punkte
und Leerzeichen am Ende, 8.3-Kurznamen, Groß-/Kleinschreibung, Junctions),
loomux' eigene Befehle werden an ihren Argumenten erkannt statt am
Programmnamen, ein Programm, das der Wächter nicht kennt, wird auf einem
geschützten Pfad verweigert, ebenso eine Schreibung, deren Pfad eine Expansion
(`$X`, `$(…)`, Backtick, `%X%`) trägt, die dort landen kann. Ohne den
Schlüssel gilt `default`; jeder andere Wert ist ein Fehler, und der Wächter
verweigert dann jeden Aufruf wie bei kaputter Konfiguration. Der strikte Modus
verweigert mehr, als einem Projekt lieb sein mag: in diesem Repo verweigert er
`go build -o bin/loomux.exe`, weil `bin/*` geschützt ist und `go` kein Verb,
das der Wächter kennt. Siehe [Hooks](hooks.md#7-der-entscheidungsweg-von-pre-tool-use).

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

[verify.profiles]       # eingebaut: edit = [lint, types], precommit = alle fünf, stop = alle fünf
edit      = ["lint", "types"]
precommit = ["lint", "types", "test", "coverage", "graph"]
stop      = ["lint", "types", "test", "coverage", "graph"]   # was das Stop-Tor am Rundenende fährt

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

Es gibt fünf Arten, in dieser Reihenfolge: `lint`, `types`, `test`,
`coverage`, `graph`. Eine Anfrage an `loomux check` ist ein Profil, `all` (immer alle
fünf Arten) oder eine Komma-Liste von Arten (`lint,types`). Die Namen `gofmt`,
`commit-msg`, `gocover`, `graph-fresh`, `blast-audit`, `all`, `lint`, `types`, `test`, `coverage` und `graph` sind
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

Eine Stack-Tabelle kennt die Schlüssel `lint`, `types`, `test`, `coverage`, `graph`
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
| python | `uvx ruff check . --output-format=concise` | `uv run --with mypy mypy --no-error-summary --no-pretty --exclude-gitignore .`; mit `pyright`: `uv run pyright`; mit `mypy`: `uv run mypy --no-error-summary --no-pretty` | `uv run --with pytest pytest -q --tb=short --no-header` | `uv run --with coverage coverage report --skip-covered --skip-empty -m` |
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

- Die Python-Lanes bringen ihre Werkzeuge mit `uv run --with` mit: Ein
  Projekt, das mypy, pytest oder coverage nicht als Abhängigkeit führt, fährt
  sie trotzdem, in seiner eigenen Umgebung, und eines, das sie pinnt, behält
  seine Pins. Ohne mypy-Konfiguration prüft mypy die Wurzel; Punktordner wie
  `.venv` und was `.gitignore` nennt, lässt es aus. Das Flag dafür,
  `--exclude-gitignore`, gibt es ab mypy 1.16: Ein Projekt, das ein älteres
  mypy pinnt, setzt `[verify.python.types]` selbst oder konfiguriert mypy und
  bekommt damit die Variante. Die Variante `mypy` greift,
  wo `[tool.mypy]` in `pyproject.toml`, `mypy.ini`, `.mypy.ini` oder `[mypy]`
  in `setup.cfg` steht: Sie nennt kein Ziel, also muss diese Konfiguration
  `files` setzen. Mit `pyright` und `mypy` zugleich prüft pyright.
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
  `uv run --with coverage --with pytest coverage run -m pytest …`); allein bleibt es der schnelle Weg.
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

#### Die Art `graph`

Die fünfte Art prüft, was die Änderung eines Commits im Code-Graphen
erreicht. Ein Preset haben Go und Python, mit denselben Befehlen:

```toml
[stack.go.graph]
commands = ["{loomux} check graph-fresh", "{loomux} check blast-audit --cached --threshold 5"]

[stack.python.graph]
commands = ["{loomux} check graph-fresh", "{loomux} check blast-audit --cached --threshold 5"]
```

`graph-fresh` baut einen gedrifteten Graphen neu; `blast-audit --cached` ist
rot, wenn ein gestagter Bereich einen Seed mit mindestens fünf Aufrufern hat
(Aufrufer in Tests zählen mit) und kein geänderter Test ihn erreicht (siehe
[CLI-Referenz](cli-reference.md#loomux-check-blast-audit---root-pfad---cached----base-ref---threshold-n---skip-test-callers)).
Die Befehle laufen nacheinander, und `blast-audit` läuft auch nach einem roten
`graph-fresh`, gegen den Graphen auf der Platte; die Lane ist dann ohnehin rot.

- **Eine Lane je Lauf, in der Wurzel.** Der Graph gehört der Projektwurzel,
  nicht einem Stack. Ein Stack mit mehreren Bereichen bekommt trotzdem ein
  `graph/go`, nicht einen Neubau je Bereich, und von mehreren Stacks mit
  Graph-Befehl führt ihn nur der erste in Byte-Reihenfolge aus: in einem
  Repository mit Go und Python läuft `graph/go`, und `graph/python` ist
  `not-applicable` („graph covered by graph/go“), das Urteil bleibt grün. Ein
  Stack ohne Graph-Lane (etwa shell) trägt nichts: seine Lane meldet „no
  command“, und der nächste Stack mit Befehl läuft.
- **Nur in einem Check.** Im Edit-Scope plant `graph` gar keine Lane, wie
  eine Art ohne Befehl: ein Profil `edit` mit `graph` baut bei einem Edit nie
  neu.
- **Eine Prüfung entscheidet, bevor die Lane startet.** `loomux check` fragt
  in dieser Reihenfolge, ob `.loomux/state/graph/wiring.json` existiert, ob es
  ein `HEAD` gibt (`git rev-parse --git-path MERGE_HEAD HEAD`), ob kein Merge
  läuft und ob etwas gestagt ist (`git diff --cached --quiet`). Das erste
  „nein“ macht die Lane `not-applicable` mit diesem Grund, und das Urteil
  bleibt grün. Die Lane ist darum lokal: `not-applicable` in CI und in jedem
  frischen Klon (`.loomux/state/` wird nicht eingecheckt), bei einem
  manuellen `loomux check precommit` ohne gestagte Änderungen, bei
  `commit --amend` ohne neue Änderungen, während eines Merges und beim
  Root-Commit. Sie läuft nur, wo jemand den Graphen gebaut hat.
- **Am Stop-Tor urteilt die Lane über die Runde gegen HEAD.** `stop`
  enthält `graph` als Vorgabe. Am Rundenende ist der Index meist leer; der
  Stop-Hook schreibt die Arbeit darum über eine Kopie des Index
  (`loomux-stop-index-<pid>` im Git-Verzeichnis, mit `add -A`, unversionierte
  Dateien eingeschlossen, `.loomux/state` ausgenommen) und gibt nur der
  Graph-Lane diese Kopie als `GIT_INDEX_FILE`; `blast-audit --cached`
  vergleicht dann alles, was git nicht ignoriert, mit `HEAD`. Die Kopie wird
  nach der Kette gelöscht, gleich mit welchem Urteil; eine, die ein während
  der Kette abgebrochener Prozess zurückließ, löscht das nächste Rundenende.
  Ihre Prüfung fragt, ob der Graph existiert, ob es ein `HEAD` gibt, ob kein
  Merge, Rebase, Cherry-Pick oder Revert läuft und ob sich die Arbeit von
  `HEAD` unterscheidet („nothing changed against HEAD“). Eine rote Lane hält
  die Runde (Exit 2) wie jede andere. Ohne Graph entsteht keine Kopie, und
  die Lane ist `not-applicable`. Weil die Lane gegen `HEAD` urteilt, gilt ein
  schon grün befundener Baum nur unter demselben `HEAD` wieder als grün:
  Nach einem Commit innerhalb der Runde läuft die Kette erneut.
  Der Befund trägt die Überschrift `blast audit: index against HEAD`, und
  „index“ ist die Kopie: Der gedruckte Befehl, von Hand gegen den echten,
  meist leeren Index gestartet, findet nichts. `loomux check stop` baut
  dieselbe Kopie und fährt dieselben Lanes; das Tor um sie herum — die Marke
  `.loomux/no-verify`, ein schon grün befundener Baum, die Befunde der
  Subagenten, der Blockadezähler — hat nur der Hook. Ein Projekt, das `stop`
  in `[verify.profiles]` selbst definiert, bekommt die Lane erst, wenn es
  `graph` einträgt. **Grenze:** Das Urteil gegen `HEAD` setzt voraus, dass
  das pre-commit-Tor jeden Commit geprüft hat. Ein Commit an ihm vorbei
  innerhalb der Runde (`git commit --no-verify`, ein Cherry-Pick oder Merge,
  ein Klon ohne scharfe Hooks) gehört am Rundenende zu `HEAD` und wird nie
  geprüft.
- **Den Schwellenwert ändern** heißt, `commands` in der Tabelle des Stacks
  zu ersetzen, der die Lane trägt (`[verify.go.graph]`, in einem reinen
  Python-Repository `[verify.python.graph]`), **beide** Einträge; eine Lane,
  die nur `blast-audit` nennt, verliert den Neubau:

  ```toml
  [verify.go.graph]
  commands = ["{loomux} check graph-fresh", "{loomux} check blast-audit --cached --threshold 10"]
  ```

- **Die Lane abschalten** heißt `graph = false` unter einem beliebigen Stack.
  Der Graph gehört der Wurzel, also gilt der Schalter für das Projekt: in
  einem Repository mit Go und Python macht `graph = false` unter
  `[verify.go]` (oder unter `[verify.python]`) jede Graph-Lane
  `not-applicable` mit „graph switched off under [verify.go]“, benannt nach
  dem ersten solchen Stack in Byte-Reihenfolge, und kein anderer Stack trägt
  den Graphen an seiner Stelle. Bleibt kein Stack mit Graph-Befehl übrig, wie
  in einem Go-Repository mit `graph = false` unter `[verify.go]`, melden die
  Lanes „no command“, wie schon immer. Ein Schalter unter einem Stack, den
  das Projekt nicht hat, ändert nichts. `graph` aus einem Profil wegzulassen
  (`precommit = ["lint", "types", "test", "coverage"]`) hält die Lane nur aus
  diesem Profil heraus: `loomux check all` und `loomux check graph` planen sie
  weiter.

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
| `not-applicable` | Art im Stack nicht definiert oder `false`; bei `graph` sagte die Prüfung nein | neutral, angezeigt | nicht angezeigt |

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
  Ausgabe auf `stderr`. Eine übersprungene Lane blockiert nichts und steht
  ebenfalls auf `stderr`, bei Exit 0 außerdem im Kontext des Hosts auf
  `stdout`, für Claude Code `hookSpecificOutput.additionalContext`; unter
  `--host antigravity` wird dieses `stdout` nicht weitergegeben, denn ob agy
  den Kontext eines PostToolUse liest, ist ungemessen. Eine Datei, deren
  Endung kein aktiver Stack beansprucht, bekommt keine Lanes und endet mit 0;
  unter `--host codex` endet der Aufruf mit 1, sobald die Nutzlast eine Datei
  nennt, denn die Codex-Naht hat keinen Adapter.
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

### `[modules]` (Was in diesem Projekt läuft)

Schaltet die drei Module von loomux für ein Projekt ein oder aus. Fehlt die
Tabelle oder ein Schlüssel darin, ist das Modul **an**; ein unbekannter
Schlüssel oder ein Wert außer `true`/`false` lässt die Datei als kaputt
gelten, damit ein vertipptes `graf = false` nicht eingestellt aussieht, ohne
es zu sein.

```toml
[modules]
hooks = true   # post-edit, stop, session-start, subagent-start/-stop
brain = true   # die Wiki-Lane, die MCP-Werkzeuge brain_*, convert und fetch
graph = false  # die MCP-Werkzeuge graph_*
```

| Schlüssel | Typ | Vorgabe | Aus heißt |
|---|---|---|---|
| `hooks` | Boolean | `true` | `loomux hook post-tool-use`, `stop`, `session-start`, `subagent-start` und `subagent-stop` enden sofort mit 0 und tun nichts. |
| `brain` | Boolean | `true` | Die Lane `lint/wiki` ist überall aus — Edit-Lane, `loomux check` und Stop-Gate —, genau so, wie `[verify.wiki] lint = false` sie abschaltet; `loomux mcp` bietet kein `brain_*`-Werkzeug an; `loomux convert` und `loomux fetch`, in diesem Projekt aufgerufen, verweigern mit Exit 1. |
| `graph` | Boolean | `true` | `loomux mcp` bietet kein `graph_*`-Werkzeug an. |

- **Der Wächter läuft immer.** `hook pre-tool-use` liest `[modules]` nicht:
  Die Schreibschranke ist global und schützt die schreibgeschützten Bereiche
  anderer Repositories; kein Projekt kann sie abschalten.
- **Die Brücke liest es beim Start.** `loomux mcp` nimmt das Projekt aus
  `--root`, sonst aus der ersten `.loomux/config.toml` oberhalb des
  Verzeichnisses, in dem der Wirt sie gestartet hat; außerhalb jedes
  Projekts bietet sie jedes Werkzeug an. Der Wirt hält `tools/list` im
  Cache; eine Änderung wirkt darum, wenn die Brücke neu startet.
- **Die Befehle bleiben.** `loomux brain …` und `loomux graph …` auf der
  Kommandozeile sind keine Module; `[modules]` entscheidet, was von selbst
  läuft und was einem Agenten angeboten wird.
- `loomux config set modules.graph false` schreibt den Schlüssel; `true`
  entfernt ihn wieder, weil eine Vorgabe nie geschrieben wird.

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

- **Welche Sprachen ein Bau parst, ergibt sich aus den Extraktoren, nicht aus
  einer Liste, die ein Repository erklärt.** `internal/code/extract/golang`
  liest `.go`-Dateien und `internal/code/extract/python` liest `.py`-Dateien,
  weil jeder nur die lesen kann; eine feste Liste in
  `internal/code/extract/all` hält sie, und allein die Endung wählt, welcher
  auf eine Datei angewandt wird. Eine weitere Sprache kommt im Code in diese
  Liste, und kein Konfigurationsschlüssel entscheidet, welcher Extraktor auf
  eine Datei angewandt wird.
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

### Die Bereichsdeklaration: `[area]`, `[layout]`, `[wiki]`, `[index]`, `[maintenance]`, `[model]`

Diese Tabellen erklären, zusammen mit `[privacy]` unten, einen
Wissensbereich; das Brain-Modul liest sie. Eine Datei ohne `[area]` erklärt
keinen Bereich. Ein Wert vom falschen Typ ist ein Fehler, der Tabelle und
Schlüssel nennt.

```toml
[area]
scope = "project/loomux"

[layout]
wiki = "wiki"

[wiki]
untouched_days = 180

[maintenance]
on_merge = true
branch = "master"

[model]
enabled = true
roles = { describe = true, place = false, propose = true }
```

| Schlüssel | Typ | Vorgabe | Beschreibung |
|---|---|---|---|
| `area.scope` | String (**Pflichtfeld**) | — | Der Scope, unter dem das Projekt registriert ist, z. B. `project/loomux`. |
| `layout.wiki` | String | — | Wo das Wiki-Bündel liegt, relativ zur Wurzel. |
| `layout.hub` | String | — | Wo die Hub-Seiten liegen. |
| `layout.review` | String | — | Wo Review-Fälle abgelegt werden. |
| `layout.inbox` | String | — | Wo Dateien auf die Umwandlung warten; ein absoluter Pfad wird abgelehnt. |
| `wiki.types` | Array von Strings | — | Seitentypen, die der Bereich über die bekannten hinaus erklärt. |
| `wiki.untouched_days` | Ganzzahl ≥ 1 | `180` | Nach wie vielen Tagen eine Seite als unberührt gilt. |
| `index.include` | Array von Strings | — | Globs der Dateien, die der Index liest. |
| `index.exclude` | Array von Strings | — | Globs, die der Index überspringt. |
| `index.unsearched` | Array von Strings | — | Globs, die registriert, aber nie an qmd gegeben werden. |
| `maintenance.on_merge` | Boolean | `false` | Merges für den Abgleich festhalten. |
| `maintenance.branch` | String | `"main"` | Der Zweig, dessen Merges zählen. |
| `model.enabled` | Boolean | — | `false` schaltet das lokale Modell für diesen Bereich ab. |
| `model.roles` | Tabelle von Booleans | — | Welche von `describe`, `place` und `propose` das Modell hier übernimmt; eine unbekannte Rolle ist ein Fehler. |

`[model]` in einem Bereich kann die maschinenweiten Einstellungen nur
einengen (siehe [unten](#maschinenweite-einstellungen-configtoml-im-zustandsverzeichnis)):
`enabled` schaltet das Modell ab, nie an, und `roles` behält nur die Rollen,
die beide Dateien einschalten.

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
| `mode` | String | `"local_only"`, `"manual_cloud"` (Vorgabe) oder `"automatic_cloud"`; jeder andere Wert wird abgelehnt. Ein `local_only`-Bereich existiert auf dem Cloud-Kanal nicht, und Verschachtelung hebt das nicht auf: Ein Pfad in seinem Wiki oder Quellbaum bleibt dort verborgen, auch wenn der Baum eines anderen Bereichs ihn enthält. |
| `never` | Array von Strings | Glob-Muster der Pfade, die kein Kanal erreicht. |

---

### `[agent]` (Modelle für die Rollen eines Flows)

Bindet die Rollen, die ein [Flow](flows.md#3-rollen-und-modelle) nennt, an
Modelle. Ein Flow nennt selbst nie ein Modell, ein mitgelieferter Flow lädt
also in jedem Projekt; das Projekt entscheidet, wer jede Rolle beantwortet.
Jeder Befehl von `loomux flow`, der einen Flow lädt, liest die Tabelle
vorher.

```toml
[agent]
default     = "sonnet"     # das Modell jeder Rolle ohne Bindung
mcp_servers = ["loomux"]   # was das Werkzeugprofil mcp nutzen darf

[agent.models.sonnet]
provider = "claude"
model    = "sonnet"

[agent.models.gemini]
provider = "agy"           # ohne model: die Vorgabe der CLI des Anbieters

[agent.roles]
reviewer = "gemini"
writer   = "sonnet"
```

| Schlüssel | Typ | Bedeutung |
|---|---|---|
| `[agent] default` | String | Der Modellname, auf dem jede Rolle ohne Bindung läuft, und ein Agentenknoten ohne Rolle. Muss ein Name unter `[agent.models]` sein. |
| `[agent] mcp_servers` | Array von Strings | Die Server, die ein Knoten mit dem Werkzeugprofil `mcp` nutzen darf, je einer als `mcp__<server>`; jeder Eintrag ein nicht leerer String. Vorgabe `[]`. |
| `[agent.models.<name>] provider` | String, **Pflicht** | Wer für diesen Modellnamen antwortet, etwa `claude` oder `agy`. Noch nicht gegen eine Liste geprüft: Die kommt mit den Modelladaptern. |
| `[agent.models.<name>] model` | String | Das Modell des Anbieters. Fehlt es, wählt die CLI des Anbieters ihre Vorgabe; steht es da, darf es nicht leer sein. |
| `[agent.roles] <rolle>` | String | Der Modellname, auf dem die Rolle läuft. Muss ein Name unter `[agent.models]` sein. |

- **Namen** von Modellen und Rollen folgen `[A-Za-z_][A-Za-z0-9_]*`.
- **Jeder Befund auf einmal.** Ein unbekannter Schlüssel unter `[agent]` oder
  `[agent.models.<name>]` wird mit den bekannten abgelehnt, ebenso ein
  Default oder eine Bindung, die kein Modell nennt: `[agent.roles] reviewer
  names "gemini", which is not under [agent.models]; known: none`. Ein Flow
  erfährt also nie erst am ersten bezahlten Knoten, dass eine Rolle nirgends
  läuft. `[agent] settings` ist noch unbekannt; es kommt mit den Adaptern.
- **Die Auflösung** des Modells eines Knotens — Rolle des Knotens, Rolle des
  Flows, Bindung, `[agent] default`, die Vorgabe der claude-CLI — steht in
  [Flows](flows.md#3-rollen-und-modelle); `loomux flow show <flow>` druckt sie
  je Knoten.
- **Mit `loomux config`** heißen die benannten Schlüssel
  `agent.roles.<rolle>`, `agent.models.<name>.provider` und
  `agent.models.<name>.model`; `config list` zeigt eine Zeile je Name, den die
  Datei hält, und `agent.roles.*` (oder `agent.models.*.provider`, `…model`)
  als ungesetzte Zeile, solange sie keinen hält. Das Modell kommt zuerst:
  `loomux config set agent.roles.reviewer gemini` gelingt erst, wenn
  `agent.models.gemini.provider` gesetzt ist, denn der Leser lehnt eine
  Bindung an ein unbekanntes Modell ab (Exit `1`, die Datei bleibt, wie sie
  war). Ein Agent schlägt eine Bindung mit `loomux config set
  agent.roles.reviewer gemini --propose` vor.

**`[agent]` ist nicht `[model]`.** `[model]` ist das lokale Ollama-Modell des
Brains, gesetzt in der maschinenweiten `config.toml` (ein Bereich schaltet es
nur ab oder engt es ein), mit seinen eigenen Rollen `describe`, `place` und
`propose` (siehe
[`loomux config --global`](cli-reference.md#10-konfiguration-loomux-config)).
`[agent]` bindet die Rollen eines Flows an die Modelle der CLI eines
Anbieters.

---

### `[flow]` (Welcher Flow läuft)

```toml
[flow]
default   = "example"      # was `loomux flow run` ohne Namen startet
overrides = ["example"]    # mitgelieferte Flows, die ein Projekt-Flow gleichen Namens verdecken oder überlagern darf
```

| Schlüssel | Typ | Bedeutung |
|---|---|---|
| `default` | String | Der Flow, den `loomux flow run` ohne Namen startet; ein Flow-Name (`[a-z][a-z0-9-]*`). Ohne ihn lehnt `run` ohne Namen ab und nennt die Flows, die es kennt. |
| `overrides` | Array von Strings | Die mitgelieferten Flows, die `.loomux/flows/<name>/` verdecken (mit eigener `flow.toml`) oder überlagern (einzelne Dateien unter `instructions/` und `questions/`) darf; jeder ein Flow-Name, keiner doppelt. Vorgabe `[]`. |

- **Ein unbekannter Schlüssel** wird mit den bekannten abgelehnt.
- **Ob ein Name ein Flow ist,** fragt dieser Leser nicht: Er liest keine
  Flows. Ein `default`, der keinen Flow nennt, lässt `loomux flow list` und
  ein `loomux flow run` ohne Namen scheitern; `list` warnt vor einem Namen in
  `overrides`, den der Katalog nicht ausliefert.
- **Die Freigabe schreibt nur ein Mensch.** Ohne `overrides` wird ein
  Projektordner mit dem Namen eines mitgelieferten Flows mit einer Warnung
  übergangen, und der mitgelieferte Flow fährt; der Wächter verweigert einem
  Agenten jedes Schreiben in den Ordner eines mitgelieferten Flows oder eines
  Namens aus `overrides` (siehe
  [Flows](flows.md#6-tore-gehören-einem-menschen)). Ein Agent schlägt eine
  Freigabe mit `loomux config set flow.overrides example --propose` vor.

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

# --- Wächtermodus: default, oder strict gegen einen, der gezielt umgehen will
# [guard]
# mode = "default"

# --- Prüfkette: loomux check und post-edit; alles Übrige ist Preset ---------
[verify.go.lint]
commands = ["{loomux} check gofmt cmd internal", "go vet ./..."]

[verify.go.test]
measuring = "go test ./... -count=1 -covermode=set -coverpkg=example.com/project/... -coverprofile={coverprofile}"

[verify.go.coverage]
measure = "go test ./... -count=1 -covermode=set -coverpkg=example.com/project/... -coverprofile={coverprofile}"

# --- Module: ein fehlender Schlüssel ist an; der Wächter läuft immer ---------
[modules]
graph = false

# --- Worktree-Isolierungsspiegel ---------------------------------------------
[worktree]
mirror = [
  "node_modules",
  ".cache",
  "bin"
]

# --- Datenschutz-Schranken ---------------------------------------------------
[privacy]
mode = "manual_cloud"
never = [".env*", "*.key", "credentials.json"]

# --- Flows: die Modelle ihrer Rollen und welcher Flow läuft -----------------
[agent]
default = "sonnet"

[agent.models.sonnet]
provider = "claude"
model = "sonnet"

[agent.roles]
reviewer = "sonnet"

[flow]
default = "example"
```

---

## 4. Wo was liegt

| | Ort |
|---|---|
| Zustandsverzeichnis | unter Windows `%LOCALAPPDATA%\loomux`, sonst `~\AppData\Local\loomux`; auf anderen Systemen `$XDG_STATE_HOME/loomux`, sonst `~/.local/state/loomux` |
| Bereichsregistry (Einträge `[[area]]`: `scope` und `path` Pflicht, `wiki` optional, dazu die Wahrheitswerte `readonly`, `signpost`, `shared`, `workspace`; siehe [Erste Schritte](getting-started.md)) | `<Zustandsverzeichnis>\registry.toml` |
| Maschinenweite Einstellungen (lokales Modell) | `<Zustandsverzeichnis>\config.toml` |
| Einzelne Dateien, die die Schreibschranke offen hält | `<Zustandsverzeichnis>\open.toml` |
| Manifest eines beschreibbaren Bereichs | `<Bereichspfad>\.loomux\config.toml` |
| Manifest eines lesenden Bereichs, wie die Schreibschranke es liest | `<Zustandsverzeichnis>\areas\<scope>\.loomux\config.toml` |
| Artefakte eines lesenden Bereichs (`index.md`, `graph.json`, `_identities.tsv`) und sein Manifest, wie `loomux brain` sie liest | `<Zustandsverzeichnis>\areas\<scope>\`; Rückfall zum Lesen auf `%LOCALAPPDATA%\brain\areas\<scope>\` bis Stufe 4e |
| Artefakte eines beschreibbaren Bereichs | sein `path` |
| Stempel des letzten Reconcile | `<Zustandsverzeichnis>\maintenance\last-run.txt`; Rückfall zum Lesen auf `%LOCALAPPDATA%\brain\maintenance\last-run.txt` bis Stufe 4e |
| Sitzungszustand der Hooks (`base`, `blocks`, `green`) | `<projekt>\.loomux\state\hooks\<session_id>.json` |
| Schnappschüsse und Befunde der Subagenten | `<projekt>\.loomux\state\hooks\<session_id>\agents\<agent_id>.json` |
| Der Marker, der das Stop-Tor abschaltet | `<projekt>\.loomux\no-verify` |
| Eigene Flows und Overlays eines Projekts | `<projekt>\.loomux\flows\<name>\` |
| Flow-Läufe: Journal und Marke | `<projekt>\.loomux\state\runs\<id>.jsonl`, `<id>.flow` |

`LOOMUX_STATE_DIR` überschreibt das Zustandsverzeichnis,
`LOOMUX_LEGACY_BRAIN_DIR` das Verzeichnis von ultra-brain; einen
Kommandozeilenschalter gibt es für keines von beiden. Seit Stufe 3a ist das
Altverzeichnis nur noch ein Rückfall zum Lesen für Artefakte, die ultra-brain
geschrieben hat: loomux liest zuerst sein eigenes Zustandsverzeichnis und
schreibt nie hierher. Mit Stufe 4e gleicht ein Mensch den Maschinenzustand von
Hand ab; danach entfallen Rückfall, Verzeichnis und Variable.

### Maschinenweite Einstellungen: `config.toml` im Zustandsverzeichnis

`<Zustandsverzeichnis>\config.toml` hält die Einstellungen des lokalen Modells
für alle Projekte der Maschine. Die Datei schreibt ein Mensch.

```toml
[model]
enabled = true
roles = { describe = true, place = true, propose = false }
```

| Schlüssel | Typ | Vorgabe | Beschreibung |
|---|---|---|---|
| `model.enabled` | Boolean | `false` | Ob das lokale Modell überhaupt gefragt wird; ein Bereich kann es nur abschalten. |
| `model.endpoint` | String | `"http://127.0.0.1:11434"` | Wo Ollama lauscht; die Adresse muss auf dem Loopback bleiben. |
| `model.name` | String | `"hf.co/unsloth/gemma-4-E4B-it-qat-GGUF:UD-Q4_K_XL"` | Das Ollama-Modell, das gefragt wird. |
| `model.roles` | Tabelle von Booleans | `{ describe = true, place = true, propose = true }` | Welche Rollen das Modell übernimmt; ist die Tabelle gesetzt, ist eine nicht genannte Rolle aus. |
| `model.temperature` | Gleitkommazahl | `0.0` | Die Sampling-Temperatur, zwischen 0 und 2. |

Das `[model]` eines Bereichs darf `enabled` und `roles` nur einengen.

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

### Einzelne Dateien: `open.toml`

Eine Datei, die die eigenen Konventionen eines Nutzers außerhalb jedes Baums
ablegen — etwa `~/.claude/AGENT_LEARNINGS.md`, die eine globale `CLAUDE.md` von
den Agenten pflegen lässt —, wird in `<Zustandsverzeichnis>\open.toml` geöffnet.
Die Datei schreibt ein Mensch:

```toml
files = ["C:/Users/ich/.claude/AGENT_LEARNINGS.md"]
```

- Jede genannte Datei ist zu denselben Bedingungen offen wie das Memory oben:
  auch dann, wenn die Registry nicht lesbar ist, und nie für das Manifest.
- Es öffnen nur einzelne Dateien. Ein Eintrag muss ein absoluter Pfad sein,
  darf kein Verzeichnis nennen und nicht im Zustandsverzeichnis liegen, wo
  `registry.toml` und `open.toml` die Grenzen der Schranke selbst halten. Globs
  gibt es nicht.
- `files` ist der einzige Schlüssel. Die Datei gilt ganz oder gar nicht: Ein
  Eintrag, den die Schranke nicht verwenden kann, ein unbekannter Schlüssel
  oder ungültiges TOML, und nichts darin öffnet. Jede Ablehnung endet dann mit
  `open.toml is ignored:` und dem Grund.
- Eine eigene Datei statt einer Tabelle in `registry.toml`, weil
  `loomux area add` die Registry aus ihren `[[area]]`-Einträgen neu schreibt
  und die Tabelle dabei verlöre.
- Kein Agent kann sie schreiben: Das Zustandsverzeichnis liegt außerhalb jedes
  Baums, den die Schranke öffnet.
- Was ein Agent in eine solche Datei schreibt, wirkt überall, wo sie geladen
  wird, wie eine Anweisung; bei `AGENT_LEARNINGS.md` hinter einer globalen
  `CLAUDE.md` ist das jede Sitzung in jedem Projekt. Die Liste entscheidet der
  Nutzer.

Eine Ablehnung mit `lies outside every writable tree` nennt, wo geschrieben
werden darf: die erlaubten Bäume, danach das Scratchpad der Sitzung, die
Dateien, die `open.toml` öffnet, und die Memory-Bäume, sofern sie sich
benennen lassen.
