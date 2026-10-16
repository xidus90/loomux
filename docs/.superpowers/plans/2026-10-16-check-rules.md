# `check rules`: Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux check rules` prüft eine Bereichskarte aus
`[rules.layers.*]` in `.loomux/config.toml`: erlaubte Kanten im Graphen,
Quellen ohne Bereich, ein Testmodul je Quelle und Reste der Instrumentierung
im Commit. Dazu eine Wurzel-Lane `rules` und `gdformat` im GDScript-Preset.

**Architecture:** Ein neues Paket `internal/verify/rules` liest den
Dateibestand aus Git (Arbeitsbaum oder Index), baut den Graphen im Prozess
mit `extract` und `resolve`, ordnet jede Datei einem Bereich zu und wendet
vier Regeln an. Die Extraktoren tragen dafür eine Zeile je Rohkante, und
`resolve` legt unaufgelöste Basen offen. Die Lane `rules` gehört wie `graph`
zur Wurzel und wird nur geplant, wenn `[rules]` in der Konfiguration steht.

**Tech Stack:** Go (Stdlib, `go/token`), gotreesitter über
`internal/code/extract/treesitter`, `internal/pathkey`, `internal/config`,
`internal/verify/linecov`.

**Spec:** `docs/.superpowers/specs/2026-10-16-projektregeln-design.md`. Die
Nummern unten (E1 … E8) verweisen auf deren Entscheidungen.

## Global Constraints

- Code, Kommentare, Meldungen, Commits englisch; dieser Plan und Nachträge zur Spec deutsch.
- 100 % Coverage je Funktion; Ausnahme nur mit `//coverage:exempt <reason>` direkt über `func`.
- Kein `init()`; eingebettete Daten (`presets.toml`, `tools.toml`, Testprofile) beim ersten Gebrauch lesen (`sync.OnceValues`).
- Jede neue Testzeile läuft vor dem Code rot; jede Regel bekommt eine Mutationsrunde mit Kontrollmutant (Muster: `internal/verify/mutation_test.go`).
- Commits nach Conventional Commits, Scope ist ein Code-Bereich (`extract`, `resolve`, `rules`, `verify`, `cli`, `config`), nie ein Arbeitspapier; kein `Co-Authored-By`, Autor ist der Mensch.
- Gate vor jedem Commit: `.githooks/pre-commit` läuft von selbst; von Hand `sh ci/gate.sh`.
- Git in Tests immer über `gitenv.Environ()`; Git-Aufrufe im Produkt mit `-z` und `-c core.quotePath=false`.
- Nie im Checkout von space schreiben; Proben in einer Kopie im Scratchpad.
- Agenten pushen nie.
- Befundtexte wörtlich aus E3.6, E4, E5.4, E6.4.
- Quellen sind `.go`, `.py`, `.gd`; Kantenziele jede Datei des Graphen (E3.3, E4).
- Exit von `check rules`: 0 ohne Befund, 1 mit Befunden, 2 bei Kartenfehler oder Lesefehler (E1).
- Marker: `# __NANO_COVERAGE_INSTRUMENTED__` als erste Zeile nach Trim (E6).
- Label des PRs: `release:minor`.

## Review Focus

1. **Pfad mit Leerzeichen, `#` und Nicht-ASCII** (space liegt unter `#GIT`): `ls-files -z` mit `core.quotePath=false`, nie Zeilen splitten. Test in Task 4 mit einer Datei `a #b/ü.gd`.
2. **CRLF und BOM** in `.gd` und `project.godot`: Marker in Zeile 1 mit `\r` und mit BOM wird erkannt; `test_lookup_folder` mit CRLF gelesen. Tests in Task 9 und Task 10.
3. **Verfolgte Datei im Arbeitsbaum gelöscht** (ohne `--cached`): Sie zählt nicht als Quelle und nicht als Testmodul, statt einen Lesefehler (Exit 2) zu werfen. Test in Task 4.
4. **Gestagte Fassung weicht vom Arbeitsbaum ab** (`--cached`, auch der Index von `git commit -a` über `indexFileFor`): Kanten und Marker kommen aus dem Index. Test in Task 4 und Task 11.
5. **Globs mit Windows-Trennern** in der Karte (`godot\core\**`): Kartenfehler mit klarer Meldung statt eines Globs, der still nichts trifft. Test in Task 5.

---

## Dateien

| Datei | Verantwortung |
|---|---|
| `internal/code/extract/extract.go`, `golang/`, `python/`, `gdscript/` | `RawEdge.Line`, Versionen |
| `internal/code/model/graph.go`, `decode.go`, `internal/code/resolve/*.go` | `Edge.Line`, `GraphWith`, `Unresolved`, Schema 3 |
| `internal/code/resolve/gomod.go` | `ModulesFrom` aus Inhalten |
| `internal/config/rules.go`, `internal/config/schema/schema.go`, `schema/validate.go` | `[rules]` lesen, Schlüssel, Validierung |
| `internal/verify/rules/files.go` | Dateibestand aus Git |
| `internal/verify/rules/layers.go` | Zuordnung, Kartenfehler mit Dateien |
| `internal/verify/rules/edges.go` | Kanten und Basen (E3) |
| `internal/verify/rules/unlayered.go` | Quellen ohne Bereich (E4) |
| `internal/verify/rules/pairing.go`, `tests.toml` | Testprofile, Paarung (E5) |
| `internal/verify/rules/instrumented.go` | Rest der Instrumentierung (E6) |
| `internal/verify/rules/check.go` | `Check` |
| `internal/verify/linecov/tools.go`, `tools.toml`, `internal/verify/godotsuite/discover.go` | Marker als Profilschlüssel |
| `internal/cli/check.go` | `check rules` |
| `internal/verify/schema.go`, `plan.go`, `presets.toml`, `internal/hooks/post_edit.go`, `internal/hooks/stopgraph.go` | Lane `rules`, `gdformat` |
| Doku | `docs/{en,de}/cli-reference.md`, `configuration.md`, `benchmarks.md`, `README.md`, `README.de.md`, betroffene Seiten unter `docs/wiki/` |

---

### Task 1: Belege

**Files:**
- Modify: `docs/.superpowers/specs/2026-10-16-projektregeln-design.md` (Nachtrag am Ende von E5 und E7)

- [ ] **Step 1:** In `C:\Users\micro\Documents\#GIT\space\godot\addons\gdUnit4\src\core\GdUnitTestSuiteScanner.gd` `resolve_test_suite_path` und `_to_naming_convention` wörtlich lesen; in `GdUnitSettings.gd` die Schlüssel `settings/test/test_lookup_folder` und `settings/test/test_suite_naming_convention` samt Vorgaben und wie die Enum in `project.godot` steht.
- [ ] **Step 2:** Drei Abbildungen von Hand nachrechnen und als Tabelle in den Nachtrag schreiben: `res://core/seeded_rng.gd`, `res://ui/main_menu/main_menu.gd`, `res://addons/p/x/foo.gd`.
- [ ] **Step 3:** Klären, was `blast-audit --cached` bei leerem Index und in CI tut (`query.GraphReady`, `internal/cli/check.go:136`), und was `hooks.OpenStopIndex` in die Kopie legt. Ergebnis: Satz im Nachtrag zu E7.2 („`--cached` ohne Gestagtes liest …“).
- [ ] **Step 4:** Commit `docs(verify): record the gdUnit4 test path rule and the cached index semantics`.

### Task 2: Zeile je Rohkante in den Extraktoren

**Files:**
- Modify: `internal/code/extract/extract.go:158` (`RawEdge`)
- Modify: `internal/code/extract/golang/extract.go` (`importEdges` :252, `callEdges` :273, `collect` :349, `callEdge` :402, `Version` :41)
- Modify: `internal/code/extract/python/python.go` (`base` :207, `version` :57), `python/imports.go` (`importEdge` :83, Aufrufer :15, :33), `python/calls.go` (`call` :76)
- Modify: `internal/code/extract/gdscript/edges.go` (`extendsOf` :16, `call` :104, `attribute` :135, `reference` :172), `gdscript/resource.go` (:32), `gdscript/gdscript.go` (`version` :64)
- Modify: `internal/code/extract/all/all_test.go` (`TestVersionIsSortedJoin`)
- Test: je Extraktor das bestehende `*_test.go`

**Interfaces:**
- Produces: `RawEdge.Line int \`json:"line,omitempty"\``: die 1-basierte Zeile der Verwendung; 0 nur für `contains`. Versionen `go/2`, `python/2`, `gdscript/2`.

- [ ] **Step 1: Failing tests.** Je Sprache ein Test `TestEdgeLines` mit einer Quelle, in der jede Relation auf einer anderen Zeile steht, Assertion auf `(Relation, Name|Specifier, Line)`:
  - Go: Import auf Zeile 3, Aufruf `f()` auf 7, `x.g()` auf 8.
  - Python: `import a` auf 1, `from b import c` auf 2 (Modul und Name je Zeile 2), `class K(Base):` auf 4, Aufruf auf 5.
  - GDScript: `extends RefCounted` auf 1, `const X = preload("res://x.gd")` auf 3, Verweis `MarketLines.has_market()` in einer Methode auf 9 (Zeile 9, nicht der Methodenanfang), `class Inner extends Node:` auf 12; in einer `.tscn` ein `ext_resource` auf Zeile 2; in `project.godot` ein Autoload auf seiner Zeile.
- [ ] **Step 2:** `go test ./internal/code/extract/...` → FAIL (Feld fehlt).
- [ ] **Step 3:** Feld setzen. Go: das `*token.FileSet` aus `File` (:62) an `importEdges` und `callEdges` reichen, `fset.Position(n.Pos()).Line`; `importsOf` gibt die `*ast.ImportSpec` mit. Tree-sitter: `int(n.StartPoint().Row)+1`; Python `importEdge` bekommt den Knoten. `reference` dedupliziert schon je (Quelle, Name), die erste Verwendung ist die kleinste Zeile.
- [ ] **Step 4:** Versionen anheben, `TestVersionIsSortedJoin` nachziehen; `go test ./internal/code/...` → PASS. Goldens, die Rohkanten enthalten, regenerieren und den Diff lesen: nur `line` kommt hinzu.
- [ ] **Step 5:** Commit `feat(extract): record the line of every raw edge`.

### Task 3: `Edge.Line`, unaufgelöste Basen, Module aus Inhalten

**Files:**
- Modify: `internal/code/model/graph.go:151` (`Edge`), `model/decode.go:12`, `internal/code/resolve/resolve.go` (:19, :57), `resolve/gdscript.go` (:41-76, `bind` :203, `chain` :232, `base` :246), `resolve/python.go` (:27-77, `classNamed` :327, `onlyClass` :603), `resolve/gomod.go:39`
- Test: `resolve/gdscript_test.go`, `resolve/python_test.go`, `resolve/resolve_test.go`, `resolve/gomod_test.go`, Goldens unter `testdata/`

**Interfaces:**
- Consumes: `RawEdge.Line` (Task 2).
- Produces:
  ```go
  // model
  type Edge struct { Source, Target NodeID; Relation Relation; Confidence Confidence; Line int `json:"line,omitempty"` }
  // resolve
  type Reason string
  const ReasonNone Reason = "none"; const ReasonAmbiguous Reason = "ambiguous"
  type Unresolved struct { Source model.NodeID; File string; Line int; Name string; Reason Reason }
  func GraphWith(files []extract.Result, mods []Module, extractor string) (*model.Graph, []Unresolved)
  func Graph(files []extract.Result, mods []Module, extractor string) *model.Graph // = GraphWith ohne zweiten Wert
  func ModulesFrom(gomods map[string]string) ([]Module, error) // rel path of go.mod -> content
  ```

- [ ] **Step 1: Failing tests.**
  - `TestEdgeCarriesTheSmallestLine`: zwei Verweise derselben GDScript-Methode auf dieselbe Klasse (Zeilen 5 und 9) ergeben eine Kante mit `Line == 5`; dasselbe für zwei Python-Importe desselben Moduls.
  - `TestUnresolvedBases`: GDScript `extends Node` → `Unresolved{Name:"Node", Reason: ReasonNone, Line: 1}`; zwei Dateien mit `class_name Dup`, eine dritte `extends Dup` → `ReasonAmbiguous`; Python `class K(object, Missing)` → zwei Einträge `ReasonNone`; ein aufgelöstes `extends` ergibt keinen Eintrag; ein `extends "res://missing.gd"` ergibt keinen Eintrag.
  - `TestModulesFromMatchesModules`: für die Fixtures von `gomod_test.go` liefern `ModulesFrom` (Inhalte) und `Modules` (Platte) dasselbe.
  - `TestSchemaVersion3`: `Validate` nimmt 3 und lehnt 2 ab.
- [ ] **Step 2:** `go test ./internal/code/resolve/... ./internal/code/model/...` → FAIL.
- [ ] **Step 3:** Implementieren. `bind`/`chain`/`base` geben statt `bool` einen `Reason` zurück (leer = aufgelöst), `only`/`len(ids) != 1` unterscheiden 0 von ≥ 2; ebenso `classNamed`/`onlyClass`. Der `seen`-Schlüssel in `resolveGDScript` wird `(Source, Target, Relation)`, der Wert hält die kleinste Zeile; dieselbe Regel für die Python-Importe. `schemaVersion` in `resolve.go` und `model/decode.go` auf 3.
- [ ] **Step 4:** `go test ./internal/code/...` → PASS; Goldens (`wiring.json`) regenerieren, Diff lesen: `line` und die Version, sonst nichts.
- [ ] **Step 5:** Commit `feat(resolve): carry edge lines and report unresolved bases`.

### Task 4: Dateibestand aus Git (`files.go`)

**Files:**
- Create: `internal/verify/rules/files.go`, `internal/verify/rules/files_test.go`

**Interfaces:**
- Produces:
  ```go
  type File struct { Rel string; Content []byte }
  // Files lists the files git knows under root: with cached the index
  // (contents from the index), else tracked plus untracked-not-ignored
  // (contents from disk; a tracked file missing on disk is left out).
  func Files(root string, cached bool) ([]File, error)
  ```

- [ ] **Step 1: Failing tests** in einem Repo aus `t.TempDir()` mit `gitenv.Environ()`:
  - `TestFilesSkipsIgnoredAndWorktrees`: `.gitignore` mit `.godot/`; Dateien `.godot/x.gd`, `.claude/worktrees/w/core/a.gd` (eigenes `git init` darin); Ergebnis enthält keine von beiden.
  - `TestFilesWorkingTree`: verfolgte `a.gd`, unverfolgte `b.gd`, gelöschte verfolgte `c.gd` → `a.gd`, `b.gd`.
  - `TestFilesCached`: `a.gd` gestagt mit Inhalt `x`, im Baum `y` → Inhalt `x`; unverfolgte `b.gd` fehlt.
  - `TestFilesCachedHonoursHookIndex`: mit `hookIndex` (Muster `internal/code/query/git_test.go:17`) liest `Files(root, true)` den Index von `git commit -a`.
  - `TestFilesOddNames`: `a #b/ü.gd` kommt unverändert an.
- [ ] **Step 2:** `go test ./internal/verify/rules/` → FAIL.
- [ ] **Step 3:** Implementieren über `query.runGit` (exportieren als `query.Git(dir string, args ...string) ([]byte, error)`, falls nötig, damit der Hook-Index gilt): `ls-files -z` bzw. `ls-files -z --cached --others --exclude-standard`; Index-Inhalte gesammelt über `cat-file --batch` mit den Blob-IDs aus `ls-files -z --stage`.
- [ ] **Step 4:** `go test ./internal/verify/rules/` → PASS.
- [ ] **Step 5:** Commit `feat(rules): list files from the git index or working tree`.

### Task 5: `[rules]` in der Konfiguration

**Files:**
- Create: `internal/config/rules.go`, `internal/config/rules_test.go`
- Modify: `internal/config/schema/schema.go` (Schlüssel), `schema/validate.go:51` (`readAll`), `schema/schema_test.go` (Leser-Abgleich)

**Interfaces:**
- Produces:
  ```go
  type Layer struct { Name string; Paths, MayUse, Bases []string; HasMayUse, HasBases, Tests, Checked bool }
  type Rules struct { Present bool; Layers []Layer } // sorted by Name
  func ReadRules(root string) (Rules, error)
  func ParseRules(path string, doc map[string]any) (Rules, error)
  func RulesKeys() []string
  ```
  `Checked` ist `true`, außer `checked = false` steht. Schlüssel in `schema`: `rules.layers.*` mit `paths`, `may_use`, `bases` (StringList), `tests`, `checked` (Bool), Modul `Base`.

- [ ] **Step 1: Failing tests** `TestParseRules` als Tabelle: keine Sektion → `Present == false`; `[rules]` leer → `Present`, keine Bereiche; vollständiges Beispiel aus E2; Fehler (alle in einer Meldung, `; `-getrennt, Datei genannt): unbekannter Schlüssel, Bereichsname kein Identifier, leeres `paths`, leerer Glob, Glob mit `\`, `may_use` nennt unbekannten Bereich, `checked = false` mit `may_use`/`bases`/`tests`. Dazu `TestValidateRejectsABrokenRulesMap` über `schema.Validate`.
- [ ] **Step 2:** `go test ./internal/config/...` → FAIL.
- [ ] **Step 3:** Implementieren nach dem Muster `ParseFlowSettings` (`flowsettings.go:38`); Glob-Prüfung wie `globList` (`policy.go:131`) plus Ablehnung von `\`.
- [ ] **Step 4:** `go test ./internal/config/...` → PASS.
- [ ] **Step 5:** Commit `feat(config): read the layer map under [rules.layers]`.

### Task 6: Zuordnung zu Bereichen (`layers.go`)

**Files:**
- Create: `internal/verify/rules/layers.go`, `layers_test.go`

**Interfaces:**
- Consumes: `config.Rules` (Task 5), `pathkey.Glob`.
- Produces:
  ```go
  type Map struct{ /* layers by name, assignment by rel */ }
  func Assign(r config.Rules, rels []string) (*Map, []string) // map errors as text
  func (m *Map) Of(rel string) (config.Layer, bool)
  ```

- [ ] **Step 1: Failing tests** `TestAssign`: `core/**` und `core/a.gd` → `core/a.gd` im Datei-Bereich, `core/b.gd` im Verzeichnis-Bereich; `godot/**` gegen `godot/core/**` → der tiefere gewinnt; zwei Bereiche mit `core/*.gd` und `core/a.*` auf `core/a.gd` → Fehler `core/a.gd matches layers a and b equally`; ein Glob ohne Treffer → Fehler `layer x: godot/sim/** matches no file`; Datei außerhalb → `Of` false.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Spezifität: (a) Glob ohne `*`/`?`/`[` vor einem mit; (b) mehr feste Segmente vor dem ersten Platzhalter; (c) Gleichstand zwischen Bereichen ist ein Fehler, im selben Bereich egal. Die Ordnung steht als Kommentar über der Vergleichsfunktion und in `configuration.md` (Task 13).
- [ ] **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(rules): assign every file to the most specific layer`.

### Task 7: Kanten und Basen (`edges.go`)

**Files:**
- Create: `internal/verify/rules/edges.go`, `edges_test.go`

**Interfaces:**
- Consumes: `File` (Task 4), `*Map` (Task 6), `resolve.GraphWith`, `resolve.ModulesFrom`, `all.For`, `all.Version`.
- Produces:
  ```go
  type Finding struct { Path string; Line int; Text string } // String(): "path:line text" or "path text" when Line == 0
  func Edges(files []File, m *Map) ([]Finding, error)
  ```

- [ ] **Step 1: Failing tests** `TestEdges` auf kleinen Godot-Projekten im Speicher (`project.godot` mit `[autoload]`):
  - der Fall vom 2026-08-09: `model/game_state.gd` ruft `MarketLines.has_market()` auf Zeile 9, `MarketLines` in `ui/panels/market_lines.gd` → `model/game_state.gd:9 core uses ui (MarketLines) — not in may_use of core; allow it with: loomux config set rules.layers.core.may_use … --propose`;
  - Autoload `Game.x` aus `core` ohne `autoload` in `may_use` → Befund; mit → keiner;
  - `preload("res://ui/x.tscn")` aus `core` → Befund mit Symbol `res://ui/x.tscn`;
  - `# MarketLines` im Kommentar → kein Befund;
  - Kante in den eigenen Bereich, Bereich ohne `may_use`, Quelle in `checked = false` → kein Befund; Kante aus `core` in `harness` (`checked = false`) ohne `harness` in `may_use` → Befund;
  - Ziel ohne Bereich → kein Befund;
  - `class Inner extends Node:` in `core` mit `bases = ["RefCounted"]` → `core/a.gd:12 core extends Node — not in bases of core`; ohne `bases` → keiner;
  - doppelter `class_name Dup` und `extends Dup` → `… ambiguous base Dup` unabhängig von `bases`;
  - ein Go-Import über Bereiche und ein Python-Import über Bereiche ergeben je einen Befund;
  - zwei Verwendungen desselben Ziels in einer Datei → ein Befund an der kleineren Zeile.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Graph aus `files` bauen: je Datei `all.For(rel)` → `lang.File(rel, string(content))`; `go.mod`-Inhalte an `ModulesFrom`; `GraphWith`. Kanten mit `Relation != contains`, deren Quelldatei (Knoten → `Path`) eine Quelle (`.go`/`.py`/`.gd`) in einem geprüften Bereich ist. Symbol = `Name` des Zielknotens, für Datei-Ziele der Pfad wie im Import geschrieben. Befundtext für Basen: `<path>:<line> <layer> extends <Name> — not in bases of <layer>` bzw. `<path>:<line> ambiguous base <Name>: more than one class declares it`.
- [ ] **Step 4:** PASS; Mutationsrunde (Vergleich `may_use`, Spezifität, `checked`) mit Kontrollmutant.
- [ ] **Step 5:** Commit `feat(rules): report edges and bases a layer may not use`.

### Task 8: Quellen ohne Bereich (`unlayered.go`)

**Files:**
- Create: `internal/verify/rules/unlayered.go`, `unlayered_test.go`

**Interfaces:**
- Produces: `func Unlayered(rels []string, m *Map) []Finding`

- [ ] **Step 1: Failing test** `TestUnlayered`: `godot/sim/a.gd` → `godot/sim/a.gd is in no layer; add it to a layer in [rules.layers], or to one with checked = false`; `.tscn`, `.tres`, `project.godot`, `README.md` ohne Bereich → nichts; Datei in `checked = false` → nichts; `.go` und `.py` ohne Bereich → je ein Befund.
- [ ] **Step 2:** FAIL. **Step 3:** Implementieren. **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(rules): report sources outside every layer`.

### Task 9: Testprofile und Paarung (`pairing.go`, `tests.toml`)

**Files:**
- Create: `internal/verify/rules/tests.toml`, `internal/verify/rules/pairing.go`, `pairing_test.go`

**Interfaces:**
- Consumes: `linecov.GodotCfg` (`linecov/export.go:4`), Task 1 (Abbildung von gdUnit4).
- Produces:
  ```go
  func Pairing(files []File, m *Map) ([]Finding, []string, error) // findings, map errors
  ```
  `tests.toml` beschreibt je Profil Endung, Erkennung (`plugin.cfg:gdUnit4` bzw. immer für Go), wie ein Test erkannt wird und welche Abbildung gilt (`gdunit4` oder `beside`). Die Abbildung `gdunit4` ist Go-Code, weil sie die Logik von `resolve_test_suite_path` nachbildet; der Eintrag wählt sie nur aus.

- [ ] **Step 1: Failing tests** `TestPairing`:
  - Godot-Projekt unter `godot/` mit `addons/gdUnit4/plugin.cfg`; `godot/core/seeded_rng.gd` ohne Test → `godot/core/seeded_rng.gd has no test module; expected godot/test/core/seeded_rng_test.gd`; mit Test → nichts;
  - `[gdunit4] settings/test/test_lookup_folder="spec"` (mit CRLF) → erwartet `godot/spec/core/…`; leerer Wert → neben der Quelle;
  - `Foo.gd` → `FooTest.gd`; die drei Abbildungen aus Task 1 als Tabelle;
  - Dateien unter dem Testordner sind selbst keine Quellen der Paarung;
  - Go: `a.go` ohne `a_test.go` → Befund, `a_test.go` selbst nicht;
  - `.py` in einem Bereich mit `tests = true` → Kartenfehler `layer core: tests = true, but no test profile knows .py files`;
  - `.gd` mit `tests = true` ohne gdUnit4 → Kartenfehler, der das fehlende Addon nennt;
  - ein Eintrag in `tests.toml` ohne Fixture lässt einen Lint-Test scheitern (Muster `linecov/fixtures_test.go`).
- [ ] **Step 2:** FAIL. **Step 3:** Implementieren; Profile über `sync.OnceValues`. **Step 4:** PASS; Mutationsrunde.
- [ ] **Step 5:** Commit `feat(rules): require a test module per source from the test tool's own layout`.

### Task 10: Marker der Instrumentierung als Profilschlüssel (`instrumented.go`)

**Files:**
- Modify: `internal/verify/linecov/tools.go` (`Profile`), `tools.toml:172` (`[tool.nanocoverage]`), `tools_test.go`
- Modify: `internal/verify/godotsuite/discover.go:18,199` (Konstante durch das Profil ersetzen)
- Create: `internal/verify/rules/instrumented.go`, `instrumented_test.go`

**Interfaces:**
- Produces: `Profile.Instrumented string \`toml:"instrumented"\`` (erste Zeile einer auf der Platte instrumentierten Datei); `func Instrumented(files []File, root string) ([]Finding, error)`.

- [ ] **Step 1: Failing tests**:
  - `TestNanoProfileNamesItsMarker` in linecov: `instrumented == "# __NANO_COVERAGE_INSTRUMENTED__"`.
  - `TestInstrumented`: Projekt mit `[nano_coverage]` in `project.godot`; Marker in Zeile 1 → `godot/core/a.gd still holds nanocoverage instrumentation; restore the sources before committing`; mit `\r`, mit BOM, mit führendem Leerraum → Befund; in Zeile 2 oder nur erwähnt → nichts; ohne erkanntes Profil (`linecov.Activate`) → nichts; Datei in `checked = false` → Befund.
  - die bestehenden Tests von `godotsuite.Instrumented` bleiben grün.
- [ ] **Step 2:** FAIL. **Step 3:** Implementieren; `godotsuite` liest den Marker über `linecov.LoadProfiles()`. **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(rules): report coverage instrumentation left in committed sources`.

### Task 11: `Check` und `loomux check rules`

**Files:**
- Create: `internal/verify/rules/check.go`, `check_test.go`
- Modify: `internal/cli/check.go` (Switch :71-101, Usage :68), `internal/verify/schema.go:36` (`Reserved` um `rules`), `internal/cli/check_test.go`

**Interfaces:**
- Consumes: Tasks 4–10.
- Produces:
  ```go
  type Options struct { Root string; Cached bool }
  type Result struct { Findings []Finding; MapErrors []string; Notes []string }
  func Check(o Options) (Result, error)
  ```
  Reihenfolge: Instrumentierung (nur `Cached`) zuerst, dann Kanten, Paarung, ohne Bereich; innerhalb nach Pfad und Zeile.

- [ ] **Step 1: Failing tests** in `internal/cli/check_test.go` über `checkCommand`: ohne `[rules]` → Exit 0, stderr `loomux check rules: note: no [rules] in .loomux/config.toml`; Kartenfehler → Exit 2, alle Fehler auf stderr; Befunde → Exit 1, einer je Zeile auf stdout; sauber → 0; `--cached` mit abweichendem Index; unbekanntes Flag → 2.
- [ ] **Step 2:** FAIL. **Step 3:** Implementieren nach `checkLinecov` (:497) und `checkBlastAudit` (:136, `--root`, `--cached`). **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(cli): add check rules`.

### Task 12: Lane `rules`, Planung, Presets, `gdformat`

**Files:**
- Modify: `internal/verify/schema.go` (`Kinds` :26, `Config` :76, `defaults` :88, `ParseConfig` :129), `internal/config/schema/schema.go:88` (Profil-Vorgabe), `internal/verify/plan.go` (`Plan` :183, `graphCarrier` :244, `graphOff` :272, `targets` :292, `planJob` :359), `internal/hooks/post_edit.go` (`checkEdit` :145, `editJobs` :223), `internal/hooks/stopgraph.go:88` (`wantsStopIndex`), `internal/verify/presets.toml`
- Test: `plan_test.go`, `schema_test.go`, `presets_test.go`, `show_test.go`, `internal/hooks/post_edit_test.go`, `internal/config/schema/schema_test.go`

**Interfaces:**
- Consumes: `config.Rules.Present` (Task 5).
- Produces: `Config.Rules bool` (aus `doc["rules"]`), Lane-Art `rules`, Preset:
  ```toml
  [stack.<go|python|gdscript>.rules]
  commands = ["{loomux} check rules --cached"]
  on_file  = ["{loomux} check rules"]
  ```
  und in `[stack.gdscript.lint]` zusätzlich `uvx --from gdtoolkit gdformat --check .` bzw. `… --check {file}`.

- [ ] **Step 1: Failing tests:**
  - `TestRulesLaneNeedsARulesTable`: `--show` für ein Go-Projekt ohne `[rules]` hat keine Zeile `rules`; mit `[rules]` genau eine `rules` an `.`.
  - `TestRulesCarrierIsOne`: Go und GDScript aktiv → ein Job `rules/go`, die andere Zeile „rules covered by rules/go“.
  - `TestRulesOnEdit`: post-edit einer `.gd`-Datei plant `{loomux} check rules` an der Wurzel; von `.loomux/config.toml` ebenso; einer `.md`-Datei keinen `rules`-Job; ohne `[rules]` keinen.
  - `TestDefaultProfilesCarryRules`: `edit` = `lint, types, rules`, `precommit` und `stop` enthalten `rules`; `TestProfilesDefaultFollowsTheGates` nachgezogen; eigene `profiles` ohne `rules` planen keinen.
  - `TestStopIndexForRules`: `wantsStopIndex` ist wahr für `rules`.
  - `TestGDScriptLintFormats`: Preset `lint` von `gdscript` enthält `gdformat --check`.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Trägerlogik von `graph` auf eine Menge von Wurzel-Arten `{graph, rules}` verallgemeinern (`graphCarrier`/`graphOff` nehmen die Art). Ohne `Config.Rules` überspringt `Plan` die Art `rules` ganz. Für `ScopeEdit` plant `Plan` den `rules`-Job an `.` vor `targets()`, wenn die Datei eine Quelle oder `.loomux/config.toml` ist; `checkEdit` lässt `.loomux/config.toml` dafür nicht an `eff.Ignored` scheitern.
- [ ] **Step 4:** `go test ./internal/verify/... ./internal/hooks/... ./internal/config/...` → PASS; `go run ./cmd/loomux check precommit --show` in loomux selbst zeigt keine `rules`-Zeile.
- [ ] **Step 5:** Commit `feat(verify): add the rules lane and gdformat to the gdscript preset`.

### Task 13: Doku, README, Wiki

**Files:**
- Modify: `docs/en/cli-reference.md`, `docs/en/configuration.md`, `README.md`, betroffene Seiten unter `docs/wiki/` (nach `AGENTS.md` und `docs/wiki/index.md` suchen: Prüfkette, Graph, Konfiguration)
- Create im Notizordner: Patch für `docs/de/cli-reference.md`, `docs/de/configuration.md`, `README.de.md` (Muster `side-session-notes/local_eddcbe46-…/godot-suite-de-patches`; `docs/de` scheitert an `wiki-sources`, bis ein Mensch neu pinnt)

- [ ] **Step 1:** `cli-reference`: `check rules`, Flags, Exit-Codes, alle Befundtexte, Testprofile mit den Quellen (gdUnit4 `GdUnitSettings.gd`/`GdUnitTestSuiteScanner.gd`, Go `cmd/go`, pytest `python_files`), Marker mit Quelle `disk_instrumenter.cpp`.
- [ ] **Step 2:** `configuration`: `[rules.layers.*]` mit allen Schlüsseln, Spezifitätsordnung (Task 6), „keine Ausnahme im Code“, `config set rules.layers.<x>.may_use … --propose`; Lane `rules` und Profile; `gdformat` im Preset und `excluded_directories` in `.gdlintrc`/`.gdformatrc`.
- [ ] **Step 3:** README: Roadmap-Zeile „Project rules as checks“ entfernen, Fähigkeit in die Liste der Prüfungen; die Zeile für Teil 6 hängt an nichts mehr. Deutsche Fassungen als Patch.
- [ ] **Step 4:** `sh ci/gate.sh` → grün.
- [ ] **Step 5:** Commit `docs: document check rules and the layer map`.

### Task 14: Abnahme gegen space

**Files:**
- Modify: `docs/en/benchmarks.md` (deutsche Fassung als Patch)
- Modify: Spec, Nachtrag „Abnahme“

- [ ] **Step 1:** space in den Scratchpad kopieren (ohne `.tools`, mit `.git`). Dort eine `.loomux/config.toml` mit der Kantentabelle aus `docs/wiki/architecture/modulschnitt-fundament.md` als `[rules.layers.*]` schreiben (je Zeile der Tabelle ein Bereich; `test`, `autoload`, `harness`, `addons` dazu).
- [ ] **Step 2:** `loomux check rules` und `uv run --script .claude/hooks/godot_quality.py` (bzw. die vier Funktionen über einen kleinen Aufruf) auf demselben Stand; jede Abweichung erklären und im Nachtrag festhalten: erwartet sind zusätzliche Kantenbefunde (die bisher ungeprüften Zeilen der Tabelle) und Unterschiede aus E3.6 und der gdUnit4-Abbildung.
- [ ] **Step 3:** Laufzeit kalt (Extrakt-Cache leer) und warm, je drei Läufe, gegen `fast_findings` von space; Eintrag in `benchmarks.md` mit Datum und Uhrzeit.
- [ ] **Step 4:** Die Karte aus Step 1 als Vorschlag für Teil 6b in den Notizordner legen.
- [ ] **Step 5:** Commit `docs: measure check rules against the gate it replaces`.

---

## Abdeckung der Spec

| Spec | Task |
|---|---|
| E1 Befehl, Dateibestand | 4, 11 |
| E2 Karte, Kartenfehler | 5, 6 |
| E3 Kanten, Basen, Zeile je Kante | 2, 3, 7 |
| E4 Quellen ohne Bereich | 8 |
| E5 Testpaarung | 1, 9 |
| E6 Instrumentierung | 10 |
| E7 Lane, Profile, Bearbeitung | 1, 12 |
| E8 `gdformat`, Lint-Bereich | 12, 13 |
| Doku, Roadmap, Label | 13 |
| Abnahme | 14 |
