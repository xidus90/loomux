# `check godot-suite` und GDScript-Presets: Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux check godot-suite` fährt gdUnit4 headless in Scherben, misst
optional mit Nano Coverage, und die GDScript-Presets `test`/`coverage` nutzen
es zusammen mit `{godot}`, `lock` und `check linecov --report {coverdata}`.

**Architecture:** Ein neues Paket `internal/verify/godotsuite` plant und fährt
die Scherben über eingesetzte Prozess- und Zeitfunktionen. Ein eingebetteter
GDScript-Kleber (`run.gd`, `hook.gd`, `disk.gd`) wird je Lauf nach
`<area>/.godot/loomux/` geschrieben. Das Urteil kommt aus `results.xml`, nie
aus dem Exit-Code. `loomux godot rescan` erneuert den Klassen-Cache und
schreibt `project.godot` byteweise zurück. Ein neuer Lane-Schlüssel
`when_found` macht Lanes ohne Addon `unavailable`.

**Tech Stack:** Go 1.x (Stdlib: `encoding/xml`, `embed`, `crypto/sha256`,
`os.Chtimes`), `internal/child`, `internal/verify/linecov`, GDScript für
Godot 4.7 mit gdUnit4 6.2 und Nano Coverage 1.0.

**Spec:** `docs/.superpowers/specs/2026-10-15-godot-suite-design.md`. Die
Entscheidungsnummern unten (E1 … E9) verweisen darauf.

## Global Constraints

- Code, Kommentare, Meldungen, Commits englisch; dieser Plan und Nachträge zur Spec deutsch.
- 100 % Coverage je Funktion; Ausnahme nur mit `//coverage:exempt <reason>` direkt über `func`.
- Kein `init()`; eingebettete Daten werden beim ersten Gebrauch gelesen.
- Jede neue Testzeile läuft vor dem Code rot; jede Regel bekommt eine Mutationsrunde mit Kontrollmutant (Muster: `internal/verify/mutation_test.go`).
- Commits nach Conventional Commits, Scope ist ein Code-Bereich (`verify`, `cli`, `godotsuite`), nie ein Arbeitspapier; kein `Co-Authored-By`, Autor ist der Mensch.
- Gate vor jedem Commit: `.githooks/pre-commit` läuft von selbst; von Hand `sh ci/gate.sh`.
- Nie im Checkout von space (`C:\Users\micro\Documents\#GIT\space`) schreiben; Proben laufen in einer Kopie im Scratchpad.
- Agenten pushen nie; der Mensch pusht.
- Meldungstexte wörtlich aus der Spec (E2, E4, E6, E7, E8).
- `--remote-debug tcp://127.0.0.1:0`, `-c --headless --ignoreHeadlessMode`; Scherben `min(8, runtime.NumCPU(), Suites)`.
- Marker der Platten-Instrumentierung: `# __NANO_COVERAGE_INSTRUMENTED__` in Zeile 1.
- Umgebung je Scherbe: `APPDATA` und `XDG_DATA_HOME` = `<shard>/userdata`; mit Coverage `LOOMUX_COVERAGE_SHARD_DIR=<shard>/coverage`.
- Label des PRs: `release:minor`.

## Review Focus

1. **Pfad mit Leerzeichen oder `#`** (space liegt unter `#GIT`): Argv wird nie als String gebaut, nur als Slice; `res://`-Pfade bleiben relativ. Test in Task 9 mit einem Root unter `t.TempDir()/"a #b"`.
2. **Scherbe stürzt ab, bevor sie einen Report schreibt, während eine andere grün ist**: Der Befund nennt genau die tote Scherbe, die anderen Urteile bleiben, und das Restore läuft trotzdem. Test in Task 9.
3. **CRLF und BOM** in `project.godot`, `.gd` und `results.xml` (Windows-Checkouts): Erkennung von `class_name`, `extends GdUnitTestSuite` und `[autoload]` geht trotzdem. Tests in Task 4 und Task 5.
4. **Zwei Läufe im selben Checkout** (pre-commit und Stop-Gate): Die Laufzeitdatei wird atomar ersetzt (temp + rename), sonst bleibt eine halbe JSON liegen. Test in Task 6 liest nach einer abgebrochenen Schreibung den alten Stand.
5. **Area ist nicht `.`** (space: `godot/`): Schlüssel der Laufzeitdatei, Shard-Verzeichnis und der `--root` im Befund von `rescan` nennen die Area. Tests in Task 6, Task 9 und Task 11.

---

## Dateien

| Datei | Verantwortung |
|---|---|
| `internal/verify/godotsuite/discover.go` | Suites, `class_name`, Cache-Namen, Autoload-Bedarf, Hook-Konflikt, Rest-Instrumentierung |
| `internal/verify/godotsuite/junit.go` | neuester `results.xml` je Scherbe, Fälle, Fehler → Befunde (E7) |
| `internal/verify/godotsuite/shards.go` | LPT, Laufzeitdatei |
| `internal/verify/godotsuite/disk.go` | Hash-/mtime-Schnappschuss, Prüfung nach Restore |
| `internal/verify/godotsuite/glue.go` + `gd/*.gd` | eingebetteter Kleber, Schreiben nach `.godot/loomux/` |
| `internal/verify/godotsuite/suite.go` | `Run`: Ablauf E2, Ablage E5, Aufräumen |
| `internal/verify/godotsuite/rescan.go` | `Rescan` (E6.2) |
| `internal/verify/linecov/export.go` | `GodotCfg`, `NanoMatcher` für godotsuite |
| `internal/verify/schema.go`, `plan.go`, `show.go` | `when_found` |
| `internal/verify/cover.go` | `RemoveAll` |
| `internal/verify/presets.toml` | Presets E8 |
| `internal/cli/check.go`, `internal/cli/godot.go`, `internal/cli/commands.go` | Befehle |
| `testdata/godot/project/` | Fixture-Projekt für den Integrationstest |
| Doku | `docs/{en,de}/cli-reference.md`, `configuration.md`, `benchmarks.md`, `README.md`, `README.de.md`, betroffene Seiten unter `docs/wiki/` |

---

### Task 1: Belege aus der Spec

Fünf Annahmen aus „Zu belegen vor dem Plan“ werden in einer Kopie von space
mit echter Engine geprüft. Die Belege kommen als Nachtrag in die Spec. Die
echten `results.xml` und LCOV werden zu Fixtures. Dieser Task schreibt keinen
Go-Code. Fällt ein Beleg anders aus, ändert der Nachtrag die betroffene
Entscheidung, und der jeweils abhängige Task folgt dem Nachtrag.

**Files:**
- Modify: `docs/.superpowers/specs/2026-10-15-godot-suite-design.md` (Abschnitt `## Nachtrag 1 (<datum>): Belege`)
- Create: `internal/verify/godotsuite/testdata/junit/{green,failing,errors,empty}.xml`, `internal/verify/godotsuite/testdata/lcov/shard.info`

- [ ] **Step 1: Kopie anlegen**

```bash
S="$SCRATCH/space-copy"; rm -rf "$S"; cp -r "/c/Users/micro/Documents/#GIT/space/godot" "$S"
cd "$S" && git init -q && git add -A && git commit -qm base
```
(`$SCRATCH` ist das Scratchpad der Sitzung; `git init` nur, um später `git status` zu lesen.)

- [ ] **Step 2: Läufer aus `.godot/` laden (Beleg 1) und Hook einhängen (Beleg 2)**

`$S/.godot/loomux/run.gd`:
```gdscript
extends "res://addons/gdUnit4/bin/GdUnitCmdTool.gd"

class ProbeHook extends GdUnitTestSessionHook:
	func _init() -> void:
		super("LoomuxProbe", "probe")
	func startup(_s: GdUnitTestSession) -> GdUnitResult:
		print("LOOMUX_PROBE_STARTUP")
		return GdUnitResult.success()
	func shutdown(_s: GdUnitTestSession) -> GdUnitResult:
		print("LOOMUX_PROBE_SHUTDOWN")
		return GdUnitResult.success()

func _initialize() -> void:
	var service := GdUnitTestSessionHookService.instance()
	var hook := ProbeHook.new()
	hook.set_meta("enabled", true)
	service.enigne_hooks = hook  # setter appends; register() would save project settings
	super._initialize()
```
Ausführen:
```bash
"$GODOT_BIN" --path "$S" -s -d --remote-debug tcp://127.0.0.1:0 res://.godot/loomux/run.gd -a res://test/smoke_test.gd -c --headless --ignoreHeadlessMode -rd "$SCRATCH/rd"
cd "$S" && git status --porcelain project.godot
```
Erwartet: Beide `LOOMUX_PROBE_*`-Zeilen erscheinen, der Report liegt unter `$SCRATCH/rd`, und `git status` ist für `project.godot` leer. Prüfen, ob `_enable_hook` die Meta `enabled` liest. Liest es sie anders, wird die Zeile angepasst, und der Nachtrag hält die genaue Form fest.

- [ ] **Step 3: Nano lässt `.godot/` aus (Beleg 3), absolute Ablage (Zusatz)**

Ein Hook wie in Step 2, aber mit dem Inhalt von space' `harness/coverage_shard_hook.gd`. Die Ablage setzt er aus `LOOMUX_COVERAGE_SHARD_DIR`, ein absoluter Pfad außerhalb von `res://`. Vorher den Eintrag `hooks/session_hooks` aus `$S/project.godot` entfernen.
Erwartet: Der Lauf stürzt nicht ab („Bad address index“), und unter `$LOOMUX_COVERAGE_SHARD_DIR` liegt ein `lcov.info`. Falls Nano keine absolute Ablage annimmt: Der Nachtrag legt die Ablage nach `<root>/.godot/loomux/<run>/<i>/coverage` (unter `res://`), und Task 8 und 9 folgen ihm.

- [ ] **Step 4: Klassen-Cache per `--import` (Beleg 4)**

```bash
printf 'class_name LoomuxProbeClass\nextends RefCounted\n' > "$S/core/loomux_probe.gd"
"$GODOT_BIN" --path "$S" --headless --import; grep -c LoomuxProbeClass "$S/.godot/global_script_class_cache.cfg"
cd "$S" && git diff --stat project.godot
```
Erwartet: Zähler ≥ 1. Bei 0 dasselbe mit `--headless --editor --quit-after 2` und danach mit `--quit-after 300` probieren. Den kleinsten Befehl, der den Cache schreibt, hält der Nachtrag fest. Die Änderung an `project.godot` wird notiert, das ist das Material für den Test in Task 11.

- [ ] **Step 5: Restore ist byte-gleich (Beleg 5)**

```bash
cd "$S" && git checkout -q -- . && git clean -qfd -e .godot   # back to the copy's base commit
"$GODOT_BIN" --path "$S" --headless -s res://harness/disk_instrumentation.gd -- instrument
"$GODOT_BIN" --path "$S" --headless -s res://harness/disk_instrumentation.gd -- restore
git status --porcelain
```
Erwartet: leer. Ist es nicht leer: Diff ansehen (Zeilenenden? BOM?). Der Nachtrag hält fest, ob E4.4 die Hashes nach Normalisierung von Zeilenenden vergleichen muss.

- [ ] **Step 6: Fixtures sichern**

Aus den Läufen in Step 2 und 3 und aus einem Lauf mit absichtlich rotem Test (eine Assertion in einer Kopie der Suite umdrehen) und einem mit leerem `-a` kopieren:
`green.xml`, `failing.xml` (mit `<failure message=…>`), `errors.xml` (Suite mit Parsefehler, falls gdUnit4 dafür `errors` zählt, sonst weglassen und im Nachtrag nennen), `empty.xml` (0 Fälle) sowie `shard.info`.

- [ ] **Step 7: Nachtrag schreiben und committen**

Je Beleg: Befehl, Ergebnis, Folge für die Spec (keine oder welche Entscheidung sich ändert).
```bash
git add docs/.superpowers/specs/2026-10-15-godot-suite-design.md internal/verify/godotsuite/testdata
git commit -m "docs: record what Godot, gdUnit4 and Nano Coverage do for check godot-suite"
```

---

### Task 2: Lane-Schlüssel `when_found`

**Files:**
- Modify: `internal/verify/schema.go` (`Lane`, `parseLaneTable`, Layering wie bei `Needs`)
- Modify: `internal/verify/plan.go:390-394` (nach der `tests_when`-Prüfung, vor `import_check`)
- Modify: `internal/verify/show.go` (Schlüssel in `--show` ausgeben, wie `needs`)
- Test: `internal/verify/schema_test.go`, `internal/verify/plan_test.go`, `internal/verify/show_test.go`
- Modify: `docs/en/configuration.md`, `docs/de/configuration.md` (Lane-Schlüssel-Tabelle)

**Interfaces:**
- Produces: `Lane.WhenFound []string`; Planzeit-Zustand `StateUnavailable` mit Notiz `none found: <muster>`.

- [ ] **Step 1: Failing tests**

`schema_test.go`:
```go
func TestParseLaneTableReadsWhenFound(t *testing.T) {
	o, err := parseLaneTable("[verify.gdscript.test]", "test", map[string]any{"when_found": []any{"plugin.cfg:gdUnit4"}})
	if err != nil || !slices.Equal(o.Lane.WhenFound, []string{"plugin.cfg:gdUnit4"}) || !o.Set["when_found"] {
		t.Fatalf("got %+v, %v", o, err)
	}
}

func TestParseLaneTableRefusesAnEmptyWhenFound(t *testing.T) {
	_, err := parseLaneTable("[verify.gdscript.test]", "test", map[string]any{"when_found": []any{}})
	if err == nil || !strings.Contains(err.Error(), "when_found") {
		t.Fatalf("err = %v", err)
	}
}
```
`plan_test.go`: zwei Fälle mit einem Preset, dessen `test`-Lane `WhenFound: []string{"plugin.cfg:gdUnit4", "plugin.cfg:NanoCoverage"}` trägt. `HasTests` wird eingesetzt und antwortet nur für `gdUnit4` mit `true`. Erwartet: `Pre == StateUnavailable`, `Note == "none found: plugin.cfg:NanoCoverage"`. Ein zweiter Fall setzt `Godot` als Funktion ein, die `t.Fatal` ruft: Ohne Addon darf die Lane kein Binary suchen (der Befehl nennt `{godot}`). Ein dritter Fall: Beide Muster werden gefunden, die Lane läuft weiter.

- [ ] **Step 2: Rot laufen lassen**

Run: `go test ./internal/verify/ -run 'WhenFound|PlanWhenFound' -count=1`
Expected: FAIL (`unknown key "when_found"` bzw. Feld fehlt)

- [ ] **Step 3: Implementieren**

`schema.go`, Feld:
```go
	// WhenFound are tests_when patterns that must each match under the
	// lane's directory; one that finds nothing plans the lane unavailable.
	WhenFound []string
```
In `parseLaneTable`:
```go
		case "when_found":
			o.Lane.WhenFound, err = laneNeeds(table+".when_found", value)
```
`laneNeeds` lehnt eine leere Liste schon ab; ist die Meldung dort nicht schlüsselgenau, einen eigenen `laneStrings(table, key, value)` aus `laneNeeds` herauslösen. Das Layering (Preset → Config) nimmt `WhenFound` wie `Needs`, im selben `switch`/`if o.Set[...]`-Block.

`plan.go`, direkt nach dem `tests_when`-Block:
```go
	for _, p := range r.Lane.WhenFound {
		if !env.HasTests(dir, []string{p}) {
			job.Pre, job.Note = StateUnavailable, "none found: "+p
			return job, link{}, true, nil
		}
	}
```
`show.go`: `when_found = [...]` ausgeben, wo `needs` ausgegeben wird.

- [ ] **Step 4: Grün und Mutationsrunde**

Run: `go test ./internal/verify/ -count=1`
Mutanten: `!env.HasTests` → `env.HasTests`; `return` entfernt. Beide müssen rot werden.

- [ ] **Step 5: Doku**

In beiden `configuration.md` eine Zeile in der Tabelle der Lane-Schlüssel (neben `needs`): „`when_found` | list | Patterns in the form of `tests_when`; each must be found under the lane's directory, or the lane is `unavailable` (`none found: <pattern>`). Checked before `needs` and `{godot}`.“ Deutsch entsprechend.

- [ ] **Step 6: Commit**

```bash
git add internal/verify docs/en/configuration.md docs/de/configuration.md
git commit -m "feat(verify): add when_found, a lane key that needs files to be present"
```

---

### Task 3: `CleanCover` räumt Verzeichnisse

**Files:**
- Modify: `internal/verify/cover.go` (`cleanCover`, `os.Remove` → `os.RemoveAll`)
- Test: `internal/verify/cover_test.go`

- [ ] **Step 1: Failing test**

```go
func TestCleanCoverRemovesAStaleDirectory(t *testing.T) {
	root := t.TempDir()
	if err := PrepareCover(root); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(coverDir(root), "old-gdscript-root.data")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "0.info"), []byte("SF:a.gd\n"), 0o644)
	old := time.Now().Add(-48 * time.Hour)
	os.Chtimes(dir, old, old)
	if err := cleanCover(root, "new", true, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("stat = %v", err)
	}
}
```
Dazu derselbe Fall für das eigene Verzeichnis eines grünen Laufs (`<runID>-…data/`).

- [ ] **Step 2: Rot** — `go test ./internal/verify/ -run CleanCover -count=1` → FAIL (`directory not empty`)
- [ ] **Step 3: `os.RemoveAll`** an der Stelle `os.Remove(filepath.Join(dir, e.Name()))`.
- [ ] **Step 4: Grün** — dieselbe Zeile → PASS
- [ ] **Step 5: Commit**

```bash
git add internal/verify/cover.go internal/verify/cover_test.go
git commit -m "refactor(verify): let CleanCover remove a run's coverage directory"
```

---

### Task 4: Erkennung im Projekt (`discover.go`)

**Files:**
- Create: `internal/verify/linecov/export.go`
- Create: `internal/verify/godotsuite/discover.go`
- Test: `internal/verify/godotsuite/discover_test.go`, `internal/verify/linecov/export_test.go`

**Interfaces:**
- Produces (linecov):
  - `func GodotCfg(content []byte, section string) (Values, bool, error)` ruft `readGodotCfg(content, Where{Section: section})` (Feldname aus `syntax.go` übernehmen).
  - `func NanoMatcher(pattern string) (Matcher, error)` ruft `nanoGlob(pattern, "")`.
- Produces (godotsuite):
  - `func Suites(root string) ([]string, error)`: `res://`-Pfade, sortiert.
  - `func ClassCacheFindings(root, rootArg string) []string`
  - `func NeedsDisk(root string) (bool, error)`
  - `func HookConflict(root string) (string, error)`: Befund oder `""`.
  - `func Instrumented(root string) ([]string, error)`: Dateien mit Marker.
  - `func walkGD(root string, fn func(rel string, data []byte) error) error`: gemeinsamer Lauf, überspringt `addons/`, `.godot/`, andere Checkouts.

- [ ] **Step 1: Failing tests**

Hilfsfunktion im Test:
```go
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
```
Fälle (je ein `Test…`):
- `Suites`:
  - `test/a_test.gd` mit `extends GdUnitTestSuite` und `test/helper.gd` ohne → nur `res://test/a_test.gd`.
  - `x/b.gd` mit `extends GdUnitTestSuite\r\n` hinter einer BOM → gefunden.
  - Je eine Suite unter `addons/gdUnit4/`, unter `.godot/` und unter `wt/` mit einer Datei `wt/.git` → nicht gefunden.
- `ClassCacheFindings`:
  - Cache fehlt → `[".godot/global_script_class_cache.cfg does not exist; run loomux godot rescan"]` bei `rootArg == "."`, mit ` --root godot` bei `rootArg == "godot"`.
  - Cache mit `"class": &"A"` und Quellen `A`, `B` (in `core/b.gd`) → `["class_name not in the class cache: B (core/b.gd); run loomux godot rescan"]`.
  - Die Form `"class": "A"` ohne `&` gilt als vorhanden.
  - Ein `class_name C` unter `addons/` wird nicht verlangt.
- `NeedsDisk`:
  - `[autoload]\nClock="*res://autoload/clock.gd"` mit vorhandener Datei → true.
  - Dasselbe mit `[nano_coverage]\ngeneral/ignore_paths=PackedStringArray("res://autoload/**")` → false.
  - Autoload unter `res://addons/x/a.gd`: ohne `general/ignore_addons` → false (Vorgabe true), mit `general/ignore_addons=false` → true.
  - Ohne `[autoload]` → false.
- `HookConflict`:
  - `[gdunit4]\nhooks/session_hooks=Dictionary[String, bool]({\n"res://h.gd": true\n})` und `h.gd` enthält `CoverageApi` → `"project.godot registers a coverage session hook, res://h.gd; remove it from [gdunit4] hooks/session_hooks"`.
  - Ein Hook ohne diese Wörter → `""`.
  - Kein Abschnitt → `""`.
- `Instrumented`: Eine Datei beginnt mit `# __NANO_COVERAGE_INSTRUMENTED__\n` → genannt; eine Datei mit dem Marker in Zeile 2 → nicht.

`export_test.go`: `GodotCfg` liest `PackedStringArray` als Liste; `NanoMatcher("**/*_test.gd")` trifft `test/a_test.gd`, aber nicht `test/a.gd`.

- [ ] **Step 2: Rot** — `go test ./internal/verify/godotsuite/ ./internal/verify/linecov/ -count=1` → FAIL (Paket/Funktionen fehlen)

- [ ] **Step 3: Implementieren**

`export.go`:
```go
package linecov

// GodotCfg reads one section of project.godot for callers outside the gate.
func GodotCfg(content []byte, section string) (Values, bool, error) {
	return readGodotCfg(content, Where{Section: section})
}

// NanoMatcher reads a pattern the way Nano Coverage's script scanner does.
func NanoMatcher(pattern string) (Matcher, error) { return nanoGlob(pattern, "") }
```
`discover.go` (Kern):
```go
package godotsuite

const instrumentedMarker = "# __NANO_COVERAGE_INSTRUMENTED__"

var (
	suiteLine = regexp.MustCompile(`(?m)^\s*extends\s+GdUnitTestSuite\b`)
	classLine = regexp.MustCompile(`(?m)^\s*class_name\s+([A-Za-z_][A-Za-z0-9_]*)`)
	cacheName = regexp.MustCompile(`"class"\s*:\s*&?"([^"]+)"`)
	resPath   = regexp.MustCompile(`"(res://[^"]+\.gd)"`)
)

// walkGD visits the project's own GDScript: addons, the import cache and
// nested checkouts belong to someone else.
func walkGD(root string, fn func(rel string, data []byte) error) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if p != root && (rel == "addons" || d.Name() == ".godot" || isCheckout(p)) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(p) != ".gd" {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return fn(rel, bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")))
	})
}

func isCheckout(dir string) bool {
	_, err := os.Lstat(filepath.Join(dir, ".git"))
	return err == nil
}

func Suites(root string) ([]string, error) {
	var out []string
	err := walkGD(root, func(rel string, data []byte) error {
		if suiteLine.Match(data) {
			out = append(out, "res://"+rel)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}
```
`ClassCacheFindings`, `NeedsDisk`, `HookConflict` und `Instrumented` folgen derselben Form:
- `NeedsDisk` liest `[autoload]` über `linecov.GodotCfg`. Jeder Wert verliert sein führendes `*` und muss mit `res://` beginnen und nicht mit `res://addons/`, es sei denn, `ignore_addons` ist `false`. Er muss existieren und darf auf kein `NanoMatcher` aus `general/ignore_paths` passen; ohne Schlüssel gilt die Vorgabe `**/*_test.gd`.
- `HookConflict` nimmt den Rohtext ab `hooks/session_hooks=` bis zur schließenden `)` und zieht die Pfade mit `resPath` heraus. Er liest jede Datei und prüft auf `CoverageApi` oder `NanoCoverage`. Den Text des Befunds liefert er für den ersten Treffer.
- Ein unlesbares `project.godot` ist ein Fehler, den der Aufrufer als Befund ausgibt.

- [ ] **Step 4: Grün und Mutationsrunde** (`isCheckout` → `false`; `&?` → `&`; `ignore_addons`-Vorgabe umdrehen)

Run: `go test ./internal/verify/godotsuite/ ./internal/verify/linecov/ -count=1 -cover`

- [ ] **Step 5: Commit**

```bash
git add internal/verify/linecov/export.go internal/verify/linecov/export_test.go internal/verify/godotsuite
git commit -m "feat(godotsuite): find suites, class names, autoloads and coverage hooks in a Godot project"
```

---

### Task 5: Urteil aus `results.xml` (`junit.go`)

**Files:**
- Create: `internal/verify/godotsuite/junit.go`
- Test: `internal/verify/godotsuite/junit_test.go` (Fixtures aus Task 1)

**Interfaces:**
- Produces:
```go
type Shard struct {
	Index  int
	Suites []string // res:// paths planned for this shard
	Dir    string   // <shard>
	Code   int      // engine exit code
	Output string   // engine stdout+stderr
}
// Verdict judges one shard by its newest report and returns the findings
// (E7) and the seconds each suite took, keyed by res:// path.
func Verdict(s Shard) (findings []string, times map[string]float64)
```

- [ ] **Step 1: Failing tests**, ein Test je Zeile der Tabelle E7:
  - Kein Report → `shard 2 wrote no test report; it ran 3 suites`.
  - Unlesbares XML → `shard 0 wrote an unreadable test report <pfad>: …` (Präfix prüfen).
  - `empty.xml` → `shard 0 ran no test cases; none of its 1 suites loaded`.
  - `failing.xml` → `res://test/x_test.gd test_y: <erste Zeile>` (Zeile aus der Fixture).
  - Geplante Suite fehlt → `shard 1 did not run: res://test/z_test.gd`.
  - `green.xml` mit `Code: 1` → `shard 0 exited with 1 while its report shows nothing wrong:\n<Output>`.
  - `green.xml` mit `Code: 0` → keine Befunde, und `times` enthält die Suite mit ihrer `time`.
  - Zwei Reports `report_9` und `report_10` mit gleicher mtime → `report_10` gilt.
  - Ein Report mit BOM und CRLF wird gelesen.
  - Ein Report, in dem `<testsuite>` den Pfad nur als Namen trägt: Zuordnung über den Dateistamm der geplanten Suites (so tut es space). Erwartet wird der volle `res://`-Pfad im Befund.

- [ ] **Step 2: Rot** — `go test ./internal/verify/godotsuite/ -run Verdict -count=1`

- [ ] **Step 3: Implementieren**

```go
type junitSuites struct {
	Suites []struct {
		Name  string  `xml:"name,attr"`
		Time  float64 `xml:"time,attr"`
		Cases []struct {
			Name     string `xml:"name,attr"`
			Failures []struct {
				Message string `xml:"message,attr"`
				Text    string `xml:",chardata"`
			} `xml:"failure"`
			Errors []struct {
				Message string `xml:"message,attr"`
				Text    string `xml:",chardata"`
			} `xml:"error"`
		} `xml:"testcase"`
	} `xml:"testsuite"`
}
```
- `newestReport(dir)` sucht `reports/report_*/results.xml` und sortiert nach (mtime, Nummer).
- Der Suite-Name wird zum `res://`-Pfad über eine Map `stem → res://…` aus `s.Suites` aufgelöst.
- Die erste nicht leere Zeile aus `Message`, sonst aus `Text`, ist der Meldungstext.

- [ ] **Step 4: Grün und Mutationsrunde** (Sortierung umdrehen; Exit-Code-Regel invertieren)
- [ ] **Step 5: Commit**

```bash
git add internal/verify/godotsuite/junit.go internal/verify/godotsuite/junit_test.go
git commit -m "feat(godotsuite): judge a gdUnit4 shard by its report, not its exit code"
```

---

### Task 6: Scherben und Laufzeiten (`shards.go`)

**Files:**
- Create: `internal/verify/godotsuite/shards.go`
- Test: `internal/verify/godotsuite/shards_test.go`

**Interfaces:**
- Produces:
```go
func Plan(suites []string, times map[string]float64, n int) [][]string
func ShardCount(suites, cpus int) int // min(8, cpus, suites), at least 1
func LoadTimes(path string) (map[string]float64, string) // times, note ("" or why unreadable)
func SaveTimes(path string, update map[string]float64) error // merge, write temp + rename
func TimesPath(checkoutRoot, area string) string // .loomux/state/godot-suite/<area>.json, area "." → "root", "/" → "_"
```

- [ ] **Step 1: Failing tests**
  - `Plan` mit Zeiten `{a:5,b:4,c:3,d:2}` und n=2 → `[[a,d],[b,c]]`.
  - Ohne jede Zeit sind alle Gewichte 1, und die Verteilung geht nach Pfad.
  - Mit teilweisen Zeiten bekommt die fehlende Suite den Mittelwert.
  - Bei Gleichstand entscheidet der Pfad.
  - `ShardCount(3, 16) == 3`, `ShardCount(100, 4) == 4`, `ShardCount(100, 32) == 8`, `ShardCount(0, 4) == 1`.
  - `LoadTimes` mit fehlender Datei → leer, keine Notiz; mit kaputter JSON → leer und Notiz `.loomux/state/godot-suite/root.json is unreadable; planning without timings`.
  - `SaveTimes` behält fremde Einträge und überschreibt nur die gelaufenen.
  - Eine liegen gebliebene `*.tmp` ändert den gelesenen Stand nicht (Review Focus 4).
  - `TimesPath(r, "godot")` endet auf `godot-suite/godot.json`, `TimesPath(r, "a/b")` auf `a_b.json` (Review Focus 5).
- [ ] **Step 2: Rot**
- [ ] **Step 3: Implementieren**: LPT. Die Suites werden absteigend nach Gewicht sortiert, bei Gleichstand aufsteigend nach Pfad; jede Suite geht in die Scherbe mit der kleinsten Last, bei Gleichstand in die mit dem kleinsten Index. Leere Scherben fallen weg. `SaveTimes` schreibt mit `os.CreateTemp` im Zielordner und danach `os.Rename`.
- [ ] **Step 4: Grün und Mutationsrunde**
- [ ] **Step 5: Commit** — `feat(godotsuite): plan shards by the last run's timings`

---

### Task 7: Schnappschuss der Quellen (`disk.go`)

**Files:**
- Create: `internal/verify/godotsuite/disk.go`
- Test: `internal/verify/godotsuite/disk_test.go`

**Interfaces:**
- Produces:
```go
type Snapshot map[string]struct {
	Sum   [32]byte
	MTime time.Time
}
func Take(root string) (Snapshot, error) // every .gd walkGD visits
func (s Snapshot) Settle(root string) []string // E4.4: findings for changed files; resets mtime of unchanged ones
```

- [ ] **Step 1: Failing tests**
  - Nach `Take` wird eine Datei umgeschrieben und mit demselben Inhalt zurückgeschrieben. `Settle` liefert keinen Befund, und `os.Stat(...).ModTime()` ist gleich der gemerkten Zeit (auf eine Sekunde gerundet vergleichen, wegen FAT).
  - Eine Datei mit anderem Inhalt → `core/a.gd differs after restoring the instrumentation`.
  - Eine gelöschte Datei → derselbe Befund.
  - Liefert Task 1 Beleg 5 „Zeilenenden ändern sich“, kommt ein Test, der CRLF gegen LF als gleich wertet, und `Take` normalisiert vor dem Hash.
- [ ] **Step 2: Rot**
- [ ] **Step 3: Implementieren** mit `sha256.Sum256` und `os.Chtimes(p, mtime, mtime)`.
- [ ] **Step 4: Grün und Mutationsrunde** (`Chtimes` entfernen → Test rot)
- [ ] **Step 5: Commit** — `feat(godotsuite): restore source timestamps after disk instrumentation`

---

### Task 8: Eingebetteter Kleber (`glue.go`, `gd/`)

**Files:**
- Create: `internal/verify/godotsuite/gd/run.gd`, `gd/hook.gd`, `gd/disk.gd`
- Create: `internal/verify/godotsuite/glue.go`
- Test: `internal/verify/godotsuite/glue_test.go`, `internal/verify/godotsuite/integration_test.go`
- Create: `testdata/godot/project/` (Fixture-Projekt: `project.godot` mit `config/features=PackedStringArray("4.7")`, ohne `[dotnet]`; gdUnit4 wird im Test aus `$LOOMUX_GDUNIT4_DIR` kopiert und nicht vendored; `test/green_test.gd`, `test/red_test.gd`, `src/a.gd` mit `class_name A`; dazu `autoload/clock.gd` für den Plattenweg)

**Interfaces:**
- Produces:
```go
const GlueDir = ".godot/loomux"
func WriteGlue(root string) error // writes run.gd, hook.gd, disk.gd; overwrites
```

- [ ] **Step 1: Die drei Skripte nach den Belegen aus Task 1**

`gd/run.gd`:
```gdscript
extends "res://addons/gdUnit4/bin/GdUnitCmdTool.gd"
## loomux's gdUnit4 entry point: GdUnitCmdTool with a coverage hook added in
## memory when LOOMUX_COVERAGE_SHARD_DIR is set. register() would save the
## hook list into project.godot, so the hook is appended directly.

const SHARD_ENV = "LOOMUX_COVERAGE_SHARD_DIR"


func _initialize() -> void:
	if OS.get_environment(SHARD_ENV) != "":
		var service := GdUnitTestSessionHookService.instance()
		if service == null or not ("enigne_hooks" in service):
			printerr("loomux: this gdUnit4 (%s) has no session hook list to join" % _gdunit_version())
			quit(70)
			return
		var hook: GdUnitTestSessionHook = load("res://.godot/loomux/hook.gd").new()
		hook.set_meta("enabled", true)
		service.enigne_hooks = hook
	super._initialize()


func _gdunit_version() -> String:
	var cfg := ConfigFile.new()
	cfg.load("res://addons/gdUnit4/plugin.cfg")
	return cfg.get_value("plugin", "version", "?")
```
(Die Form des Einhängens folgt Beleg 2.)

`gd/hook.gd`: space' `harness/coverage_shard_hook.gd` mit folgenden Änderungen:
- `SHARD_ENV = "LOOMUX_COVERAGE_SHARD_DIR"`;
- Ablage = `<dir>/data`, Report = `<dir>/report`, Workspace = `"loomux-" + <dir>.get_file()`;
- `DISK_METADATA` liest die konfigurierte `data_store_dir` vor dem Umlenken plus `coverage.meta`;
- keine Verweise auf space; Kommentare nach Spec E3.

`gd/disk.gd`: space' `harness/disk_instrumentation.gd`, unverändert bis auf Kommentare.

- [ ] **Step 2: Failing tests**

`glue_test.go`: `WriteGlue(root)` legt die drei Dateien unter `root/.godot/loomux/` an, mit dem Inhalt der eingebetteten Dateien. Ein zweiter Aufruf überschreibt eine veränderte Datei. Ein `root`, in dem `.godot` eine Datei ist, ergibt einen Fehler.

`integration_test.go`:
```go
func TestGlueAgainstARealEngine(t *testing.T) {
	bin, gdunit := os.Getenv("GODOT_BIN"), os.Getenv("LOOMUX_GDUNIT4_DIR")
	if bin == "" || gdunit == "" {
		t.Skip("needs GODOT_BIN and LOOMUX_GDUNIT4_DIR")
	}
	// copy testdata/godot/project and gdUnit4 into t.TempDir(), run
	// `bin --path <p> --headless --import`, then Run(...) from Task 9 and
	// expect exactly the red_test finding.
}
```
Der Rumpf wird in Task 9 vervollständigt, sobald `Run` existiert. Bis dahin skippt der Test immer.

- [ ] **Step 3: Rot, dann `glue.go`**

```go
//go:embed gd/*.gd
var glue embed.FS

func WriteGlue(root string) error {
	dir := filepath.Join(root, filepath.FromSlash(GlueDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"run.gd", "hook.gd", "disk.gd"} {
		data, _ := glue.ReadFile("gd/" + name) // embedded at build time; cannot fail
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
```
(`embed.FS` als Paketvariable ist keine Parse-Arbeit beim Start; der Inhalt wird erst bei `WriteGlue` gelesen.)

- [ ] **Step 4: Grün; gdlint über `gd/`**

Run: `go test ./internal/verify/godotsuite/ -run Glue -count=1` und `uvx --from gdtoolkit gdlint internal/verify/godotsuite/gd`

- [ ] **Step 5: Commit** — `feat(godotsuite): embed the gdUnit4 runner, coverage hook and disk instrumentation scripts`

---

### Task 9: Ablauf (`suite.go`)

**Files:**
- Create: `internal/verify/godotsuite/suite.go`
- Test: `internal/verify/godotsuite/suite_test.go`; Rumpf von `integration_test.go`

**Interfaces:**
- Consumes: Task 4–8.
- Produces:
```go
type Options struct {
	Root         string // the Godot project (area), absolute
	CheckoutRoot string // where .loomux/state lives
	Area         string // slash path of Root under CheckoutRoot, "." for the root
	Godot        string // binary
	Coverage     string // "" or the directory for <i>.info
	RestoreOnly  bool
	RunID        string
	CPUs         int
	Start        func(child.Spec) child.Result
	Now          func() time.Time
}
type Result struct{ Findings, Notes []string }
func Run(o Options) (Result, error)
```

- [ ] **Step 1: Failing tests** mit einem eingesetzten `Start`. Dieser erkennt am Argv, ob `disk.gd -- instrument|restore` oder eine Scherbe gemeint ist, und schreibt für eine Scherbe die passende Fixture nach `<-rd>/report_1/results.xml` und mit Coverage `<LOOMUX_COVERAGE_SHARD_DIR>/report/lcov.info`. Fälle:
  1. Grünes Projekt mit 2 Suites und `CPUs: 2`: zwei Scherben, keine Befunde, und die Laufzeitdatei ist geschrieben. Die Argv jeder Scherbe enthält wörtlich `--remote-debug`, `tcp://127.0.0.1:0`, `res://.godot/loomux/run.gd`, `-c`, `--headless`, `--ignoreHeadlessMode`, `-rd`. Die Env enthält `APPDATA=` und `XDG_DATA_HOME=`.
  2. Vorprüfungen: Ohne `addons/gdUnit4/bin/GdUnitCmdTool.gd` kommt der Befund `gdUnit4 is missing: addons/gdUnit4/bin/GdUnitCmdTool.gd`. Ein veralteter Cache führt dazu, dass `Start` nie gerufen wird. Ein Hook-Konflikt zählt nur mit `Coverage`.
  3. Keine Suite → `no gdUnit4 test suite under <Root>`.
  4. Scherbe 1 stirbt ohne Report, Scherbe 0 ist grün: genau der Befund zu Scherbe 1 (Review Focus 2).
  5. Coverage im Speicher (kein Autoload): `disk.gd` wird nie gestartet, und `<Coverage>/0.info`, `<Coverage>/1.info` existieren. Fehlt einer Scherbe das LCOV, kommt der Befund `shard 1 wrote no coverage report`.
  6. Coverage auf der Platte (Autoload): Reihenfolge `instrument`, Scherben, `restore`. Der `restore` kommt auch dann, wenn `Start` für eine Scherbe `Err` liefert. `Settle` läuft danach.
  7. Ein Rest-Marker vor dem Lauf führt zuerst zu `restore` und zur Notiz `restored a leftover disk instrumentation`.
  8. Ein `restore` mit Exit 1 → der einzige Befund ist `sources are still instrumented; run loomux check godot-suite --restore`.
  9. `RestoreOnly` → nur `restore`, keine Scherbe.
  10. Aufräumen: Nach Grün ist `<CheckoutRoot>/.loomux/state/godot-suite/<RunID>-<area>/` weg, nach Rot bleibt es. Ein Verzeichnis, das älter als 24 h ist (mtime über `Now`), wird entfernt.
  11. Root unter `t.TempDir()/"a #b"` (Review Focus 1): Die Argv-Elemente sind unverändert, `--path` ist genau ein Element.
  12. Macht eine Gabelung über `runtime.GOOS == "darwin"` die Notiz `on macOS the shards share user://` nötig, wird die Plattform eingesetzt (`GOOS string` in `Options`) statt `runtime.GOOS` direkt gelesen.
- [ ] **Step 2: Rot**
- [ ] **Step 3: Implementieren** in der Reihenfolge von E2:
  - Vorprüfungen;
  - `Suites`;
  - `LoadTimes` und `Plan`;
  - `WriteGlue`;
  - Ablage vorbereiten;
  - optional `Take` mit Instrumentierung;
  - Scherben parallel (`sync.WaitGroup`, Ergebnisse in einem Slice nach Index);
  - `defer` für das Restore, nur auf dem Plattenweg, mit `Settle`;
  - `Verdict` je Scherbe;
  - LCOV kopieren;
  - `SaveTimes`;
  - aufräumen.

  Befunde nach Scherbe sortiert; die Fall-Befunde behalten die Reihenfolge des Reports. Der Engine-Befehl als Slice:
```go
argv := []string{o.Godot, "--path", o.Root, "-s", "-d", "--remote-debug", "tcp://127.0.0.1:0", "res://.godot/loomux/run.gd"}
for _, s := range suites {
	argv = append(argv, "-a", s)
}
argv = append(argv, "-c", "--headless", "--ignoreHeadlessMode", "-rd", filepath.Join(shardDir, "reports"))
```
- [ ] **Step 4: Integrationstest-Rumpf** ausfüllen (Kopie, `--import`, `Run` mit `child.Run`, Erwartung: genau ein Befund aus `red_test.gd`; mit `Coverage` existiert mindestens ein `.info`, und `src/a.gd` ist nach dem Lauf byte-gleich).
- [ ] **Step 5: Grün, auch mit Engine**

```bash
go test ./internal/verify/godotsuite/ -count=1 -cover
GODOT_BIN=<pfad> LOOMUX_GDUNIT4_DIR="/c/Users/micro/Documents/#GIT/space/godot/addons/gdUnit4" go test ./internal/verify/godotsuite/ -run Integration -count=1 -v
```
- [ ] **Step 6: Mutationsrunde** (`defer restore` entfernen; Sortierung; Aufräumen bei Rot)
- [ ] **Step 7: Commit** — `feat(godotsuite): run gdUnit4 in shards with optional Nano Coverage`

---

### Task 10: CLI `check godot-suite`

**Files:**
- Modify: `internal/cli/check.go` (Fall `godot-suite`, Usage-Zeile, `checkGodotSuite`)
- Modify: `internal/verify/schema.go:37` (`Reserved`)
- Test: `internal/cli/check_test.go`, `internal/verify/schema_test.go`
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md`

**Interfaces:**
- Consumes: `godotsuite.Run`, `verify.GodotFor`, `verify.ReadConfig`, `hosts.FindRoot`, `verify.NewRunID`.

- [ ] **Step 1: Failing tests**
  - `Reserved("godot-suite")` ist true.
  - Ein falscher Aufruf (`--bogus`, ein loses Argument) ergibt Exit 2.
  - Mit eingesetztem Lauf (Paketvariable `checkGodotRun = godotsuite.Run`, im Test ersetzt):
    - Befunde → Exit 1 und Befunde auf stdout, Notizen als `loomux check godot-suite: note: …` auf stderr;
    - keine Befunde → Exit 0;
    - ein Fehler von `Run` → Exit 1 mit `loomux check godot-suite: <err>`.
  - Ohne `--godot`: Der Finder wird mit `ReadConfig(checkoutRoot).Godot` gerufen. Liefert er einen Zustand, ist dessen Notiz ein Befund und Exit 1.
- [ ] **Step 2: Rot**
- [ ] **Step 3: Implementieren** nach dem Muster von `checkLinecov`:
  - Flags `--godot`, `--coverage`, `--root` (Vorgabe `.`), `--restore`.
  - `Root` und `CheckoutRoot` werden absolut (`hosts.FindRoot(root)`), `Area` relativ in Slash-Form.
  - `RunID` kommt aus `verify.NewRunID(checkNow(), os.Getpid())`, `CPUs` aus `runtime.NumCPU()`, `Start` ist `checkStart`.
  - Ein relatives `--coverage` gilt relativ zum Arbeitsverzeichnis.
- [ ] **Step 4: Doku** — Abschnitt `check godot-suite` in beiden `cli-reference.md`:
  - Flags und Ablauf;
  - die Befundtabelle E7;
  - die Regel „gdUnit4's exit code is not the verdict“ mit dem Beleg aus space;
  - `--restore`;
  - Umgebung je Scherbe;
  - die Grenzen (macOS, C#).
- [ ] **Step 5: Grün, Gate, Commit** — `feat(cli): add check godot-suite`

---

### Task 11: `loomux godot rescan` und die `import_check`-Notiz

**Files:**
- Create: `internal/verify/godotsuite/rescan.go`, Test `rescan_test.go`
- Create: `internal/cli/godot.go`, Test `internal/cli/godot_test.go`
- Modify: `internal/cli/commands.go` (`"godot": godotCommand`)
- Modify: `internal/verify/plan.go` (Notiz von `import_check`), `plan_test.go`
- Modify: `docs/{en,de}/cli-reference.md`, `docs/{en,de}/configuration.md` (Zeile `gdscript.import_check`)

**Interfaces:**
- Produces:
```go
type RescanOptions struct {
	Root, Godot string
	Start       func(child.Spec) child.Result
	Status      func(root string) (string, error) // git status --porcelain -- <root>
}
func Rescan(o RescanOptions) (notes []string, err error)
```

- [ ] **Step 1: Failing tests** (`Start` eingesetzt; er schreibt `project.godot` um wie in Task 1 Beleg 4 beobachtet und legt den Cache an):
  - `project.godot` ist danach byte-gleich, Notiz `restored project.godot, which the engine rewrote`.
  - Unverändert → keine Notiz.
  - `Status` meldet vorher `""` und nachher ` M ui/theme/x.tres` → Notiz `the engine also changed: ui/theme/x.tres`.
  - Die Engine liefert Exit 1 → Fehler mit ihrer Ausgabe, `project.godot` trotzdem zurückgeschrieben.
  - Cache danach nicht vorhanden → Fehler `the engine ran but wrote no .godot/global_script_class_cache.cfg`.
  - Der Argv ist der Befehl aus Beleg 4 (`--path <Root> --headless --import` oder das belegte `--editor --quit-after <n>`).
  - CLI: `loomux godot` ohne Unterbefehl → Exit 2; `loomux godot rescan --root godot` mit eingesetztem `Rescan` → Exit 0 und Notizen auf stderr.
  - `plan_test.go`: Die Notiz bei `import_check` lautet `run loomux godot rescan to import the project` (Area ≠ `.`: `run loomux godot rescan --root <area> to import the project`).
- [ ] **Step 2: Rot**
- [ ] **Step 3: Implementieren.**
  - Das Zurückschreiben steht in einem `defer`, damit es auch bei einem Fehler der Engine läuft.
  - `Status` ist in der CLI ein `git status --porcelain -- <root>` über `child.Run` aus dem Checkout-Root.
  - Das Binary kommt aus `--godot` oder aus `verify.GodotFor` wie in Task 10.
- [ ] **Step 4: Doku**:
  - Abschnitt `godot rescan` in `cli-reference.md`: was es zurückschreibt, was es nur meldet.
  - `configuration.md`: Text bei `import_check`.
- [ ] **Step 5: Grün, Mutationsrunde (`defer` entfernen), Commit** — `feat(cli): add godot rescan, which refreshes the class cache and keeps project.godot`

---

### Task 12: Presets

**Files:**
- Modify: `internal/verify/presets.toml` (Block E8 statt des Kommentars „No test lane“)
- Test: `internal/verify/presets_test.go`, `internal/verify/plan_test.go`, `internal/cli/check_test.go` (`--show`)
- Modify: `docs/{en,de}/configuration.md` (Abschnitt GDScript-Presets; der Absatz „A project names its own test command“ fällt weg), `README.md`, `README.de.md`
- Prüfen: `testdata/cases/` per `grep -rl gdscript testdata/cases` — eine Aufnahme mit `--show` für ein Godot-Projekt bekommt die neue Ausgabe; der Diff wird im Commit begründet.

- [ ] **Step 1: Failing tests**
  - Die Presets laden, und `gdscript.test` hat `Lock`, `WhenFound == ["plugin.cfg:gdUnit4"]` und `Measuring` mit `--coverage {coverdata}`. `gdscript.coverage` hat `After == "test"`, `Measure` mit `{godot}` und `Commands`, die `{godot}` **nicht** enthalten.
  - Plan-Test, Kette mit `measuring`: Bei `kinds = test,coverage` fährt `test` die `measuring`-Form, und `coverage` liest `{coverdata}` (`Job.Reads`).
  - Plan-Test, Doku-Commit (Teil 2, Nachtrag 5): `test` trägt `skip_when_only = ["docs/**"]`, geändert ist nur `docs/a.md`, und der eingesetzte `Godot`-Finder antwortet `missing-tool`. Erwartet: `test` und `coverage` sind beide `not-applicable`, und der Finder wurde nie gefragt.
  - Plan-Test: Ohne Nano-`plugin.cfg` ist `coverage` `unavailable` (`none found: plugin.cfg:NanoCoverage`), `test` läuft.
  - `--show` in einer Testwelt mit beiden `plugin.cfg` zeigt beide Lanes mit `lock = true` und `when_found`.
- [ ] **Step 2: Rot**
- [ ] **Step 3: Preset eintragen** (wörtlich E8, mit den beiden Kommentaren)
- [ ] **Step 4: Doku**:
  - `configuration.md`: Abschnitt „GDScript“ mit den Presets, was sie erkennen und wie man sie abschaltet (`test = false`, `lock = false`).
  - README und README.de: Die Roadmap-Zeile „Godot test suite“ fällt weg. Die Liste der Prüfungen bekommt `check godot-suite` und `godot rescan`. Die Zeile für Teil 6 (Umstellung in space) hängt nur noch an Teil 4.
  - Unter `docs/wiki/` per `grep -rl -i "godot\|gdscript" docs/wiki` die Seiten zu Verify und Presets finden und nachziehen. Wo eine Seite widerspricht, beides festhalten und markieren (globale Regel).
- [ ] **Step 5: Grün, Gate, Commit** — `feat(verify): add gdscript test and coverage presets on check godot-suite`

---

### Task 13: Abnahme gegen space

**Files:**
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`

- [ ] **Step 1: Binary bauen**: `go build -o "$SCRATCH/loomux.exe" ./cmd/loomux`
- [ ] **Step 2: Kopie von space** wie in Task 1. In der Kopie den Eintrag `hooks/session_hooks` mit `coverage_shard_hook.gd` aus `project.godot` entfernen (Teil 6 tut das in space).
- [ ] **Step 3: Referenz**: In einer zweiten, unveränderten Kopie läuft `uv run --script .claude/hooks/godot_quality.py`, mit Zeitmessung über `Measure-Command`, je einmal kalt und warm. Fälle, Fehler und Laufzeit notieren.
- [ ] **Step 4: loomux**:
```powershell
Measure-Command { & "$env:SCRATCH\loomux.exe" check godot-suite --root godot --godot $env:GODOT_BIN --coverage "$env:SCRATCH\cov" }
& "$env:SCRATCH\loomux.exe" check linecov --root godot --report "$env:SCRATCH\cov"
```
Kalt (ohne Laufzeitdatei) und warm.
Erwartet:
- dasselbe Urteil wie die Referenz;
- Laufzeit nicht über der Referenz plus 10 %;
- danach `git status --porcelain` in der Kopie leer, bis auf die bekannte Entfernung des Hooks.

Weicht etwas ab: `superpowers:systematic-debugging`, kein Weiterschreiben.
- [ ] **Step 5: Benchmarks** in beiden Sprachen chronologisch eintragen: Datum und Uhrzeit, was gemessen wurde, Referenz gegen loomux, kalt und warm.
- [ ] **Step 6: Commit** — `docs: measure check godot-suite against the gdUnit4 gate it replaces`

---

## Nach dem letzten Task

- `release-pr`-Skill: Commits nach Thema gruppieren (Task 1 ist `docs`; Korrekturen falten), Label `release:minor`, Body mit `Release: minor — …` und `## Changelog` (`### Added`: `check godot-suite`, `godot rescan`, `when_found`, GDScript-Presets; `### Changed`: Notiz von `import_check`).
- Den Push-Befehl dem Menschen nennen; nicht pushen.
