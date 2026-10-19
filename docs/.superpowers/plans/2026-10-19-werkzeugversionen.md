# Werkzeugversionen (`[verify.<stack>].tools`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ein Projekt legt die Version der Werkzeuge fest, die die Presets selbst mitbringen (ruff, mypy, pytest, coverage, gdtoolkit); ohne Angabe bleibt alles wie heute.

**Architecture:** `[verify.<stack>]` bekommt den Schlüssel `tools` (Werkzeug → Versionsangabe). Die Preset-Befehle nennen ihre Werkzeuge über zwei Platzhalter, die `expand` je Argument auflöst: `{tool:NAME}` wird zur Anforderung (`NAME` oder `NAME==SPEC`), `{from:NAME}` fällt ohne Angabe weg und wird mit Angabe zu `--from=NAME==SPEC`. Ohne `tools` sind die Argumente Zeichen für Zeichen die heutigen.

**Tech Stack:** Go, das eigene TOML-Schema in `internal/verify/schema.go`, `internal/config/schema` für `loomux config`.

**Spec:** `docs/.superpowers/specs/2026-10-19-kompatibilitaetsmatrix-design.md`, Abschnitt „Entscheidungen des Nutzers“, Punkt 2.

## Global Constraints

- 100 % Coverage je Funktion; Ausnahme nur mit `//coverage:exempt <reason>` direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst.
- Ohne `tools` erzeugt jedes Preset dieselben Argumente wie vor diesem PR (die Replay-Fälle unter `testdata/cases/` bleiben unverändert).
- `tools` gibt es nur für Stacks, deren Presets einen Platzhalter `{tool:…}` oder `{from:…}` nennen: in diesem PR `python` und `gdscript`.
- Doku Englisch und Deutsch, Code und Commits Englisch, Conventional Commits; der PR trägt `release:minor`.

## Review Focus

1. **Tippfehler im Werkzeugnamen** (`tools = { ruf = "0.6" }`): Ladefehler, der die bekannten Werkzeuge des Stacks nennt; kein stilles Ignorieren. → Test in Task 1.
2. **Versionsangabe mit Operator** (`">=1.16,<2"`): bleibt wie geschrieben hinter dem Namen, kein vorangestelltes `==`. → Test in Task 2.
3. **Versionsangabe mit Leerzeichen oder Anführungszeichen** (`"1.16 "`, `"'1.16'"`): Ladefehler, denn ein Argument ohne Shell würde sie wörtlich an uv geben. → Test in Task 1.
4. **Eine Projekt-Lane, die selbst `{tool:x}` nennt**, ohne dass `x` in `tools` steht: wird zu `x`, kein Fehler. → Test in Task 2.
5. **`[verify.go].tools`**: Ladefehler „go has no tools to pin“, weil die Go-Lanes nur die Toolchain nutzen. → Test in Task 1.

---

### Task 1: `tools` lesen und prüfen

**Files:**
- Modify: `internal/verify/schema.go` (`Config`, `parseStack`)
- Test: `internal/verify/schema_test.go`

**Interfaces:**
- Produces: `Config.Tools map[string]map[string]string` (Stack → Werkzeug → Versionsangabe, wie geschrieben); `func toolsOf(stack string) ([]string, error)` in `internal/verify/presets.go`: die Werkzeugnamen, die die Preset-Lanes des Stacks (Varianten eingeschlossen) in `{tool:…}`/`{from:…}` nennen, sortiert.

- [ ] **Step 1: Tests schreiben**

`TestToolsPinsAPresetTool`: `[verify.python]\ntools = { ruff = "0.6.*", mypy = ">=1.16" }` lädt, `cfg.Tools["python"]` ist `{ruff: "0.6.*", mypy: ">=1.16"}`.
`TestToolsRefusesUnknownNamesAndBadSpecs`, Tabelle mit erwarteter Fehlermeldung (Teilstring):
- `tools = { ruf = "1" }` unter python → `[verify.python].tools has unknown tool "ruf"; known: coverage, mypy, pytest, ruff`
- `tools = { ruff = "" }` → `[verify.python].tools.ruff must be a version`
- `tools = { ruff = "1 .2" }` und `tools = { ruff = "'1'" }` → `[verify.python].tools.ruff must be a version`
- `tools = { ruff = 1 }` → `[verify.python].tools.ruff must be a version`
- `tools = "x"` → `[verify.python].tools must be a table`
- `[verify.go]\ntools = { x = "1" }` → `[verify.go].tools: go has no tools to pin`

- [ ] **Step 2: Laufen lassen, rot**

Run: `go test ./internal/verify -run 'Tools' -v` — Expected: FAIL (`unknown key "tools"`).

- [ ] **Step 3: Implementieren**

In `parseStack` ein `case key == "tools":`. Gültige Versionsangabe: nicht leer, nur `[0-9A-Za-z.*=<>!~,+-]`. Bekannte Namen aus `toolsOf(stack)`; leer → der Go-Fehler oben mit dem Stacknamen.

- [ ] **Step 4: Laufen lassen, grün** — `go test ./internal/verify -run 'Tools'`, PASS. Danach `go test ./internal/verify`.

- [ ] **Step 5: Commit** — `feat(verify): read pinned tool versions from [verify.<stack>].tools`

### Task 2: Platzhalter `{tool:…}` und `{from:…}` auflösen

**Files:**
- Modify: `internal/verify/plan.go` (`expand`, Aufrufer, die die Werkzeuge des Stacks durchreichen)
- Test: `internal/verify/plan_test.go`

**Interfaces:**
- Consumes: `Config.Tools` aus Task 1.
- Produces: `func toolArg(arg string, tools map[string]string) (string, bool)`: gibt das ersetzte Argument zurück und `false`, wenn das Argument entfällt.

- [ ] **Step 1: Tests schreiben**

`TestToolPlaceholders`, Tabelle `arg, tools → want, keep`:
- `{tool:mypy}`, `{}` → `mypy`, true
- `{tool:mypy}`, `{mypy: "1.16.*"}` → `mypy==1.16.*`, true
- `{tool:mypy}`, `{mypy: ">=1.16,<2"}` → `mypy>=1.16,<2`, true
- `{tool:x}`, `{}` → `x`, true (Projekt-Lane, nicht gepinnt)
- `{from:ruff}`, `{}` → `""`, false
- `{from:ruff}`, `{ruff: "0.6.*"}` → `--from=ruff==0.6.*`, true
- `check`, `{ruff: "1"}` → `check`, true (kein Platzhalter)

`TestPlanPinsTheToolsOfItsStack`: ein Plan für ein Python-Projekt mit `tools = { ruff = "0.6.9" }` hat für `lint` die Argumente `["uvx", "--from=ruff==0.6.9", "ruff", "check", ".", "--output-format=concise"]`; ohne `tools` `["uvx", "ruff", "check", ".", "--output-format=concise"]`.

- [ ] **Step 2: Laufen lassen, rot** — `go test ./internal/verify -run 'ToolPlaceholders|PinsTheTools'`, FAIL.

- [ ] **Step 3: Implementieren** — `toolArg` nur für ein Argument, das als Ganzes `{tool:NAME}` oder `{from:NAME}` ist; `expand` lässt ein entfallendes Argument aus. Die Werkzeuge des Stacks reichen über den Plan bis `expand` (die übrigen Ersetzungen bleiben im `strings.Replacer`).

- [ ] **Step 4: Laufen lassen, grün.**

- [ ] **Step 5: Commit** — fold into Task 1's commit if both land together; sonst `feat(verify): expand {tool:} and {from:} into pinned requirements`.

### Task 3: Presets auf die Platzhalter umstellen

**Files:**
- Modify: `internal/verify/presets.toml` (python: lint `commands`/`on_file`, types, test `commands`/`measuring`, coverage `commands`/`measure`; gdscript: lint `commands`/`on_file`)
- Test: `internal/verify/presets_test.go`

**Interfaces:**
- Consumes: Task 2.

Umstellung, Beispiele (die übrigen nach demselben Muster):
- `uvx ruff check . --output-format=concise` → `uvx {from:ruff} ruff check . --output-format=concise`
- `uv run --with mypy mypy …` → `uv run --with {tool:mypy} mypy …`
- `uv run --with coverage --with pytest coverage run -m pytest …` → `uv run --with {tool:coverage} --with {tool:pytest} coverage run -m pytest …`
- `uvx --from gdtoolkit gdlint .` → `uvx --from {tool:gdtoolkit} gdlint .`

Die Varianten (`uv run pyright`, `uv run mypy …`) bleiben unverändert: dort nutzt das Projekt sein eigenes Werkzeug, und das pinnt es selbst.

- [ ] **Step 1: Test schreiben** — `TestUnpinnedPresetsKeepTheirArguments`: für jede Lane von python und gdscript sind die expandierten Argumente ohne `tools` gleich den Argumenten der heutigen Befehle (die heutigen Befehle als Literale im Test).
- [ ] **Step 2: rot**, **Step 3: `presets.toml` umstellen**, **Step 4: grün**, dazu `go test ./...` (die Replay-Fälle unter `testdata/cases/` laufen unverändert durch).
- [ ] **Step 5: Commit** — in den Commit von Task 1 falten.

### Task 4: `loomux config` und Doku

**Files:**
- Modify: `internal/config/schema/schema.go` (Schlüssel `{Section: "verify.python.tools", Name: "*", Kind: String, Module: Hooks, …}` und dasselbe für `verify.gdscript.tools`)
- Modify: `docs/en/configuration.md`, `docs/de/configuration.md` (Abschnitt zu `[verify.<stack>]`: Schlüssel `tools`, Platzhalter `{tool:}`/`{from:}` für eigene Lanes)
- Modify: `docs/en/support.md`, `docs/de/support.md` (Versionsspalte von python und gdscript: „version pinnable with `tools`“)
- Test: `internal/config/schema/schema_test.go`, `internal/cli/config_test.go`

- [ ] **Step 1: Tests** — `Lookup("verify.python.tools.ruff")` findet den Schlüssel; `loomux config set verify.python.tools.ruff "0.6.*" --propose` legt einen Vorschlag an, dessen Diff `[verify.python.tools]` mit `ruff = "0.6.*"` zeigt.
- [ ] **Step 2: rot**, **Step 3: Schlüssel eintragen**, **Step 4: grün.**
- [ ] **Step 5: Doku** — beide Sprachen; Support-Seite: `go test ./internal/verify -run Support` bleibt grün.
- [ ] **Step 6: Commit** — in den Commit von Task 1 falten; Gate `sh ci/gate.sh` grün (bei `lint/wiki-sources` register-stale: `docs/de` als Patch ablegen, Repin durch den Menschen).

## Abschluss

`release-pr`-Skill: ein Commit `feat(verify): pin the versions of the tools presets bring`, Label `release:minor`, Changelog `### Added – [verify.<stack>].tools pins the version of ruff, mypy, pytest, coverage or gdtoolkit; unset, the newest runs as before.`
