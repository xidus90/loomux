# Kompatibilitätsmatrix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Jede Kombination aus Laufzeit-Version (Godot, Python, Go) und Framework/Werkzeug, die loomux zu unterstützen behauptet, steht als eine Zeile Daten in einer Liste und wird von zwei Schichten belegt: schnell in `go test` bei jedem PR, echt in einem GitHub-Workflow auf Linux und Windows; ein Wochenjob meldet neue Upstream-Versionen als Issue mit Probelauf.

**Architecture:** Ein neues Paket `internal/compat` hält die Liste (`compat.toml`, eingebettet), die Beispielprojekte (`testdata/<fixture>/` mit `expect.toml`), die Tests der Schicht 1 und die Logik der drei Unterbefehle `loomux dev compat matrix|run|scout`. Zwei Workflows (`compat.yml`, `compat-scout.yml`) rufen nur diese Unterbefehle und installieren Laufzeiten und Addons. Die Support-Seite und `frameworks.toml` werden per Test gegen die Liste gehalten.

**Tech Stack:** Go, GitHub Actions (`fromJSON`-Matrix, `workflow_call`), uv, actions/setup-go, offizielle Godot-Builds von `godotengine/godot-builds`.

**Spec:** `docs/.superpowers/specs/2026-10-19-kompatibilitaetsmatrix-design.md` (Abschnitt „Umfang der Matrix (PR 3)“ und folgende).

## Global Constraints

- 100 % Coverage je Funktion; kein `init()`; eingebettete Daten erst beim ersten Gebrauch parsen.
- Schicht 1 braucht kein fremdes Werkzeug und läuft im bestehenden Tor (`gate-windows`).
- Schicht 2 und Späher laden fremde Binaries und Pakete nur aus offiziellen Quellen, mit Prüfsumme, wo die Quelle eine veröffentlicht (Godot: `SHA512-SUMS.txt`).
- Ein Lauf zählt nur als grün, wenn jede Art des Eintrags im Zustand `ok` endete; `not-applicable`, `unavailable`, `unready`, `skipped`, `missing-tool` sind rot.
- Der Späher hat nur `contents: read` und `issues: write`; er pusht nichts und öffnet keinen PR.
- Godot 3 unter Linux läuft mit dem offiziellen `linux_headless`-Build; unter Windows mit dem normalen Build (geprüft, siehe GUT-Plan).
- Doku Englisch und Deutsch; der PR trägt `release:minor` (neue `dev`-Unterbefehle und Workflows; kein Nutzerverhalten ändert sich).

## Review Focus

1. **Ein Eintrag, dessen Lane übersprungen wird** (Werkzeug fehlt, Addon nicht installiert): rot mit Lane und Zustand, nie grün. → Test in Task 4.
2. **Ein Bruch, der die falsche Art rot macht** (ein Lint-Fehler bricht auch `test`): rot, denn genau die genannte Art muss rot werden und die anderen grün bleiben. → Test in Task 4.
3. **Ein PR ohne `release:major` und ohne Änderung an der Liste:** die Pflichtprüfung `compat` meldet sofort Erfolg, ohne eine Engine zu laden. → Test in Task 3.
4. **Zwei Späher-Läufe für dieselbe neue Version:** ein Issue, das aktualisiert wird, kein zweites. → Test in Task 6.
5. **Eine neue Patch-Version** (Godot 4.7.3): löst nichts aus. → Test in Task 6.

---

### Task 1: Die Liste und ihr Lader

**Files:**
- Create: `internal/compat/compat.go`, `internal/compat/compat.toml`, `internal/compat/compat_test.go`

**Interfaces:**
- Produces:
  - `type Addon struct { Name, Version string }` (Godot-Addons aus `frameworks.toml`: gdUnit4, NanoCoverage, GUT)
  - `type Entry struct { Stack, Runtime, Version, Fixture string; Kinds []string; Addons []Addon; Tools map[string]string; Line int }`
  - `func (e Entry) Key() string` → `"<stack>/<runtime>-<version>/<fixture>"`
  - `func Load() ([]Entry, error)` (eingebettet, `sync.OnceValues`), `func parse(data []byte, path string) ([]Entry, error)`

Regeln von `parse` (Ladefehler mit Datei und Zeile, wie `parseFrameworks`): unbekannter Schlüssel; `stack` nicht in `gdscript, python, go`; `runtime` nicht `godot` (gdscript), `python` (python), `go` (go); `version` nicht `MAJOR.MINOR`; eine Art nicht in `verify.Kinds()`; doppelter Schlüssel; ein Addon ohne Zeile in `verify.LoadFrameworks()` desselben Stacks; ein Werkzeug, das `verify.ToolsOf(stack)` nicht kennt; ein `fixture` ohne Ordner unter `testdata/`.

Erste Liste: je Linie die älteste und die neueste noch gepflegte Minor-Version, am Tag des Baus aus den Upstream-Quellen bestimmt (Godot: Minor-Versionen mit einer Patch-Version in den letzten zwölf Monaten; Python: die Zyklen, die python.org als gepflegt führt; Go: die zwei neuesten Minor-Versionen). Einträge mindestens:

| Fixture | Laufzeit | Addons / Werkzeuge | Arten |
|---|---|---|---|
| `godot3-gut` | älteste und neueste 3.x | GUT 7.x, gdtoolkit 3.* | lint, test, graph |
| `godot4-plain` | älteste 4.x | gdtoolkit 4.* | lint, graph |
| `godot4-gdunit4` | älteste 4.x, die gdUnit4 trägt, und neueste | gdUnit4, NanoCoverage, gdtoolkit 4.* | lint, test, coverage, graph |
| `godot4-gut` | neueste 4.x | GUT 9.x, gdtoolkit 4.* | lint, test, graph |
| `python-mypy` | jede gepflegte 3.x | ruff, mypy, pytest, coverage | lint, types, test, coverage, graph |
| `python-pyright` | neueste 3.x | ruff, pytest, coverage (pyright aus dem Projekt) | lint, types, test, coverage, graph |
| `go-module` | die zwei gepflegten | — | lint, test, coverage, graph |

- [ ] **Step 1: Tests** — `TestTheListLoads` (jede Zeile hat Stack, Laufzeit, Version, Fixture, mindestens eine Art); `TestTheListRefuses` als Tabelle mit je einem Fall pro Regel oben und der erwarteten Meldung samt `compat.toml:<zeile>`.
- [ ] **Step 2–4:** rot, implementieren, grün (`go test ./internal/compat`).
- [ ] **Step 5:** Commit in den `feat`-Commit falten (der Branch endet in einem Commit `feat(dev): verify every supported combination of runtime and framework`).

### Task 2: Beispielprojekte und Schicht 1

**Files:**
- Create: `internal/compat/testdata/<fixture>/…` je Fixture der Tabelle, mit `expect.toml`; `internal/compat/expect.go`; `internal/compat/layer1_test.go`

**Interfaces:**
- Produces: `type Expect struct { Stacks, Variants []string; Lanes map[string]string; Edges [][3]string; Breaks []Break }`, `type Break struct { Kind, File, Content string }`, `func LoadExpect(fixture string) (Expect, error)`.

Jedes Beispielprojekt ist grün und übt jede Art seines Eintrags wirklich aus: mindestens ein Test, eine gemessene Zeile, eine Graph-Kante je Sprachmerkmal der Linie (Godot 3: `connect("s", self, "m")` und `.f()`; Godot 4: `s.connect(m)`, `super.f()`, eine Szene mit `[connection]`). Addons liegen im Projekt nur als `plugin.cfg` (für die Erkennung); Schicht 2 legt das echte Addon darüber. `Lanes` nennt je Art den Anfang des geplanten Befehls (`test: "{loomux} check gut-suite"`). Je Art ein `Break`: eine Datei, die überschrieben oder angelegt wird (lint: ein Stilfehler; types: ein Typfehler; test: eine fehlschlagende Assertion; coverage: eine ungetestete Funktion; graph: wird von Schicht 2 direkt geprüft und braucht keinen Bruch).

- [ ] **Step 1: Tests** in `layer1_test.go`, je Eintrag ein Untertest:
  - `detect.Detect(os.DirFS(fixture))` enthält `Stacks` und `Variants`;
  - `verify.Resolve` + `verify.Plan` mit einer Umgebung ohne Werkzeuge (wie `env()` in `internal/verify/plan_test.go`) plant für jede Art die Lane mit dem erwarteten Befehlsanfang;
  - `query.Build` des Beispielprojekts enthält jede Kante aus `Edges` (`source`, `relation`, `target`);
  - jeder `Break` nennt eine Art des Eintrags, und seine Datei liegt im Projekt oder in einem existierenden Ordner;
  - jedes Fixture unter `testdata/` wird von einem Eintrag benutzt.
- [ ] **Step 2–4:** rot, Beispielprojekte und `expect.go` schreiben, grün.
- [ ] **Step 5:** in den `feat`-Commit falten.

### Task 3: `loomux dev compat matrix`

**Files:**
- Create: `internal/compat/matrix.go`, `internal/compat/matrix_test.go`; Modify: `internal/cli/dev.go` (Unterbefehl `compat`), Test `internal/cli/dev_compat_test.go`

**Interfaces:**
- Produces: `type Job struct { Entry string \`json:"entry"\`; OS string \`json:"os"\` }`, `func Matrix(entries []Entry, sel Selection) []Job`, `type Selection struct { Full bool; Keys []string; ChangedFixtures []string; ChangedEntries []string }`, `func ChangedSince(repo, ref string) (fixtures, entries []string, err error)` (Fixtures aus `git diff --name-only <ref>` unter `internal/compat/testdata/`, Einträge aus dem Vergleich der Liste bei `<ref>` mit der jetzigen: neu oder geändert).

CLI: `loomux dev compat matrix [--full] [--entry <key>]... [--changed-since <ref>]` schreibt `{"include":[{"entry":…,"os":"linux"},{"entry":…,"os":"windows"}]}` auf stdout; ohne Auswahl `{"include":[]}`.

- [ ] **Step 1: Tests** — volle Matrix = jeder Eintrag × `linux`, `windows`; `--entry` wählt genau den; geänderte Fixtures wählen jeden Eintrag, der sie nutzt; eine nur in der Doku geänderte Datei wählt nichts (Review Focus 3); unbekannter `--entry` ist Exit 2.
- [ ] **Step 2–4:** rot, implementieren, grün.

### Task 4: `loomux dev compat run`

**Files:**
- Create: `internal/compat/run.go`, `internal/compat/run_test.go`; Modify: `internal/cli/dev.go`, `internal/cli/check.go` (den Teil von `check`, der plant, ausführt und Zustände je Lane liefert, als Funktion herauslösen, die `run` in-process aufruft: `func runCheck(root string, kinds []string, env …) ([]LaneResult, error)` mit `LaneResult{Lane, State, Output string}`)

**Interfaces:**
- Consumes: `Load`, `LoadExpect`, die herausgelöste Prüf-Funktion, `godotsuite.Rescan`, `query.Build`.
- Produces: `func Run(o RunOptions) (Report, error)`, `type RunOptions struct { Key, Work, Addons string; Check func(root string, kinds []string) ([]LaneResult, error) }`, `type Report struct { Entry string; Green bool; Lines []string }`.

Ablauf: Beispielprojekt nach `Work` kopieren; die Addons des Eintrags aus `Addons/<name>` nach `addons/<dir>` des Projekts kopieren; die `tools` des Eintrags als `[verify.<stack>.tools]` in `.loomux/config.toml` der Arbeitskopie schreiben; für gdscript `godot rescan`; alle Arten außer `graph` prüfen und `ok` verlangen; `graph` durch `query.Build` und die Kanten aus `expect.toml`; dann je `Break`: frische Kopie, Bruch anwenden, nur seine Art prüfen und `failed` verlangen, die übrigen Arten des Eintrags weiter `ok`. Jede Abweichung ist eine Zeile im Report mit Lane, Zustand und den letzten Zeilen der Ausgabe.

CLI: `loomux dev compat run <key> --work <dir> --addons <dir>`, Exit 0 grün, 1 rot, 2 falscher Aufruf; der Report als Markdown auf stdout (für die Job-Zusammenfassung) und als JSON nach `--json <file>`.

- [ ] **Step 1: Tests** mit einer falschen `Check`: alle `ok` → grün; eine Lane `missing-tool` → rot mit Lane und Zustand (Review Focus 1); ein Bruch, der seine Art nicht rot macht → rot; ein Bruch, der zusätzlich eine andere Art rot macht → rot (Review Focus 2); fehlende Kante → rot; die geschriebene `config.toml` enthält die Pins.
- [ ] **Step 2–4:** rot, implementieren (die Herauslösung aus `check.go` verändert die Ausgabe von `loomux check` nicht: die vorhandenen Tests und Replay-Fälle bleiben grün), grün.

### Task 5: Workflow `compat.yml`

**Files:**
- Create: `.github/workflows/compat.yml`, `ci/compat-setup.sh` (installiert eine Laufzeit und die Addons eines Eintrags; liest die Versionen über `loomux dev compat run --describe <key>` als JSON)
- Modify: `.github/rulesets/master.json` (Pflichtprüfung `compat`), `internal/compat/run.go` (`--describe`)

Aufbau: `on: pull_request` (Typen `opened, synchronize, reopened, labeled, unlabeled`), `workflow_dispatch` (Eingabe `entries`, leer = alle), `workflow_call` (Eingabe `entries` als JSON-Liste, für den Späher). Jobs: `pick` (wie `ci.yml`); `plan` baut loomux und ruft `dev compat matrix` (`--full` bei Label `release:major` oder leerer Dispatch-Eingabe, `--entry` bei Eingaben, sonst `--changed-since origin/master`); `run` mit `strategy.matrix: ${{ fromJSON(needs.plan.outputs.matrix) }}`, `fail-fast: false`, `if: matrix nicht leer`; `compat` (der Pflichtname) `needs: [plan, run]`, `if: always()`, grün, wenn `run` übersprungen wurde oder erfolgreich war. `compat-setup.sh` lädt Godot (Linux: `linux.x86_64` für 4.x, `linux_headless.64` für 3.x; Windows: `win64.exe` und für 4.x die `_console.exe`) samt `SHA512-SUMS.txt` und prüft die Summe; Python über `uv python install <version>`; Go über `actions/setup-go` mit `go-version: <version>` (im Workflow, nicht im Skript); Addons per `git clone --depth 1 --branch <tag>`.

- [ ] **Step 1:** `--describe` mit Test (JSON mit Laufzeit, Version, Addons mit Repo und Tag).
- [ ] **Step 2:** Workflow und Skript; `actionlint` lokal (`go run github.com/rhysd/actionlint/cmd/actionlint@latest`) ohne Befund; `sh -n ci/compat-setup.sh`.
- [ ] **Step 3:** Ruleset ergänzen; der Mensch wendet es auf GitHub an (im PR-Body genannt).
- [ ] **Step 4:** Abnahme nach dem Push: ein `workflow_dispatch` mit einem Godot- und einem Python-Eintrag läuft auf beiden Plattformen grün; das Ergebnis kommt in den PR-Body.

### Task 6: Späher `loomux dev compat scout` und `compat-scout.yml`

**Files:**
- Create: `internal/compat/scout.go`, `internal/compat/scout_test.go`, `.github/workflows/compat-scout.yml`; Modify: `internal/cli/dev.go`

**Interfaces:**
- Produces: `type Upstream interface { Releases(source string) ([]string, error) }` (eine echte Umsetzung über HTTP, eine falsche im Test), `type Finding struct { Line, Version string; Probe Entry; TOML string }`, `func Scout(entries []Entry, up Upstream) ([]Finding, error)`, `func IssueTitle(f Finding) string` → `"compat: <line> <version> is out"`.

Quellen: Godot über `https://api.github.com/repos/godotengine/godot/releases` (nur `-stable`); Python über `https://www.python.org/api/v2/downloads/release/?is_published=true`; Go über `https://go.dev/dl/?mode=json`; gdUnit4, GUT, NanoCoverage über ihre GitHub-Releases; ruff, mypy, pytest, coverage, gdtoolkit über `https://pypi.org/pypi/<name>/json`. Neu ist eine Minor- oder Major-Version, die neuer ist als jeder Eintrag ihrer Linie (Review Focus 5: Patch zählt nicht). Die Probe ist eine Kopie des neuesten Eintrags der Linie mit der neuen Version.

CLI: `loomux dev compat scout --json <file>` schreibt die Findings samt Probe-Einträgen; `compat.yml` nimmt Probe-Einträge über `workflow_call` als ganze Einträge an (nicht nur Schlüssel), `dev compat run --entry-json <file>` führt einen Eintrag aus, der nicht in der Liste steht. Workflow: wöchentlich (`schedule: cron: '0 5 * * 1'`) und `workflow_dispatch`; Job `scout` → `probe` (ruft `compat.yml` per `workflow_call`) → `report`: je Finding `gh issue list --search "<Titel> in:title" --state open`, dann `gh issue edit` (vorhanden, Review Focus 4) oder `gh issue create`, Rumpf mit Ergebnis je Plattform und dem TOML-Block; ein Issue, dessen Version inzwischen in der Liste steht, schließt `gh issue close`.

- [ ] **Step 1: Tests** mit falscher `Upstream`: neue Minor → ein Finding mit Probe und TOML; neue Patch → keins; neue Major → eins; ein schon eingetragener Stand → keins; `IssueTitle` stabil; eine Quelle, die nicht antwortet, ist ein Fehler, der sie nennt.
- [ ] **Step 2–4:** rot, implementieren, grün.
- [ ] **Step 5:** `actionlint` ohne Befund; Abnahme per `workflow_dispatch` nach dem Push (mindestens ein Issue oder die Meldung „nothing new“).

### Task 7: Rückkopplung zur Doku

**Files:**
- Modify: `docs/en/support.md`, `docs/de/support.md` (Abschnitt „Verified combinations“ / „Geprüfte Kombinationen“: eine Zeile je Eintrag mit Stack, Laufzeit und Version, Addons und Werkzeuge mit Versionen, Arten; die Versionsspalte der Übersicht verweist für gdscript, python und go darauf), `docs/en/cli-reference.md`, `docs/de/cli-reference.md` (`loomux dev compat`), `docs/en/releases.md`, `docs/de/releases.md` (die Matrix hält ein Major-Release auf; neue Version eintragen = eine Zeile)
- Test: `internal/compat/docs_test.go`; Modify: `internal/verify/frameworksfixtures_test.go`

- [ ] **Step 1: Tests** — `TestTheSupportPageListsEveryEntry`: die Tabelle beider Sprachen hat genau die Einträge der Liste, Zelle für Zelle; `TestProvenFollowsTheList` (in `internal/compat`, um keinen Import-Zyklus zu bauen): eine `frameworks.toml`-Zeile ist genau dann `proven = true`, wenn ein Eintrag ihr Addon führt, oder (für gdtoolkit3/4) das Werkzeug `gdtoolkit` in der passenden Hauptversion.
- [ ] **Step 2–4:** rot, Doku schreiben, grün. (`proven` bleibt als Feld in `frameworks.toml`; der Test macht es zur Ableitung, statt es zu entfernen — gleiche Zusicherung, kleinerer Umbau.)
- [ ] **Step 5:** Gate; bei register-stale `docs/de` als Patch für den Menschen.

## Abschluss

`release-pr`: ein Commit `feat(dev): verify every supported combination of runtime and framework` (mit diesem Plan), Label `release:minor`, Changelog `### Added – loomux dev compat matrix|run|scout and the compat workflows: every supported combination of Godot, Python and Go with its frameworks runs for real on Linux and Windows before a major release and when the list changes; a weekly job reports new upstream versions as issues.` Im Body: das Ruleset `compat` anwenden; Ergebnis der Abnahme-Läufe.
