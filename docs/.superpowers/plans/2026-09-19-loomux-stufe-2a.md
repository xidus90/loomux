# loomux Stufe 2a — Implementierungsplan: eine Prüfkette für `check` und post-edit

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux check <profil|arten>` fährt je erkanntem Stack die Lanes
`lint`, `types`, `test` und `coverage` aus eingebetteten Presets und
`[verify.<stack>]`. post-edit liest dieselbe Tabelle. `loomux check gocover`
löst `dev covergate` ab, und loomux prüft sich selbst damit.

**Architecture:** `internal/child` startet ein Kind mit Frist und tötet den
ganzen Prozessbaum. `internal/verify` lädt Schema und Presets, legt sie
übereinander, plant einen Graphen aus Lanes, fährt ihn über eine Naht
`func(child.Spec) child.Result` und schreibt Bericht und Urteil. `internal/cli`
und `internal/hooks` sind dünne Aufrufer. Die Parität belegt ein neuer Korpus
`testdata/cases/2a*`, aufgezeichnet an ultraloom und im Prozess abgespielt.

**Tech Stack:** Go (Toolchain 1.27.0, `go.mod` bleibt bei `go 1.25.0`),
Standardbibliothek, `golang.org/x/sys/windows` (schon in `go.mod`, v0.18.0),
`github.com/BurntSushi/toml` über `third_party/toml`. **Keine neue
Abhängigkeit.**

**Spec:** `docs/.superpowers/specs/2026-09-19-loomux-stufe-2a-design.md`. Bei
Widerspruch gilt die Spec, außer an den sieben Stellen, die der Abschnitt
„Nachträge an die Spec" unten nennt; Task 20 trägt sie in die Spec ein.

## Global Constraints

- **Arbeitsort:** Worktree
  `C:\Users\micro\Documents\#GIT\loomux\.claude\worktrees\fusion-migration-teil-2-79865c`,
  Branch `claude/fusion-migration-teil-2-79865c`, Basis `master` `86fa192`.
- **Modulpfad** `github.com/xidus90/loomux`.
- **Sprache:** Code, Bezeichner, Kommentare, Fehlermeldungen und
  Commit-Nachrichten englisch. Plan, Spec und `docs/.superpowers/parity/`
  deutsch. `docs/en/**` englisch, `docs/de/**` deutsch, gleiche Dateinamen.
- **Kein `init()`, keine Paketvariable, die Daten parst.** `presets.toml` wird
  beim ersten Gebrauch geparst (`sync.OnceValues`; die Variable hält eine
  Funktion, keine geparsten Daten).
- **Coverage 100 % je Funktion.** Eine Ausnahme nur mit
  `//coverage:exempt <reason>` als letzte Kommentarzeile direkt über `func`.
- **Das Tor weist ungestagte Eingaben ab.** Vor jedem Commit alles stagen, was
  die Aufgabe angefasst hat. Ab Task 16 gehören `*.toml` und `.loomux/` dazu.
- **`.loomux/config.toml` schreibt kein Agent.** Task 15 ist ein Halt für den
  Menschen.
- **Commits:** der Mensch ist Autor und Committer; kein Modell, kein Agent,
  kein `Co-Authored-By`. Conventional Commits; kein Commit trägt `!`.
  Mehrzeilige Nachrichten über eine Datei und `git commit -F`, nie ein Heredoc.
  **Vor jedem Commit `git branch --show-current` und `git log -1 --format=%h`
  lesen.** Niemand außer einem Menschen pusht.
- **Determinismus:** keine Map-Iteration, deren Reihenfolge in ein Ergebnis
  einfließt; sortiert wird in Byte-Ordnung.
- **Plattform:** Windows zuerst. POSIX baut; plattformgebundene Arme hinter
  Build-Tags (`_windows.go` / `_other.go` mit `//go:build !windows`), ihre Tests
  ebenso.
- **Subagenten**, falls eingesetzt: `model: "opus"` und `effort: "low"`, beides
  ausdrücklich gesetzt. Nach jedem Subagenten
  `git log -1 --format='%an <%ae>'` lesen.

## Nachträge an die Spec

Beim Planen gegen den Code gerechnet; Task 20 trägt sie in die Spec ein.

1. **`shell`, `sql`, `rust` haben doch Presets.** post-edit lintet sie heute
   (`shellcheck`, `sqlfluff`, `cargo clippy`/`cargo fmt --check`). Sie laufen
   aber, wie heute, nur, wenn der Stack erkannt oder konfiguriert ist.
2. **`cpp.types` ist `cmake --build build --parallel`**, wie post-edit heute.
   Altfehler 1 wird damit „alt Traceback, neu der Compiler als Typprüfer",
   nicht `not-applicable`.
3. **`clang-format` prüft, statt umzuschreiben.** post-edit rief
   `clang-format -i <datei>` und schrieb die Datei des Agenten um. Das Preset
   ruft `clang-format --dry-run --Werror {file}`. Neuer Eintrag 17 der
   Abweichungsliste.
4. **Die Wiki-Lane baut `internal/hooks`, nicht `verify`.** `verify.Job` trägt
   dafür ein Feld `Fn`. Die Abhängigkeit lautet damit
   `hooks → verify, brain/wiki` und `verify → child, gitenv, detect,
   shellwords`.
5. **`{file}` ist relativ zum Arbeitsverzeichnis der Lane** (ihrem Bereich),
   mit Schrägstrichen. `{pkgdir}` entfällt: Die Lane läuft ohnehin im Bereich,
   den `detect` für `typescript` liefert.
6. **Die Workspace-Skripte fallen weg.** post-edit rief in einem Bereich
   `npm --prefix D run typecheck`, `run lint` und `run check` — die Skripte des
   Wirts. Die Presets rufen `npx tsc --noEmit`, `npx eslint`, `npx svelte-check`
   direkt. Ein Wirt mit eigenem Skript schreibt es in `[verify.typescript]`.
   Neuer Eintrag 18 der Abweichungsliste.
7. **`{coverprofile}` fehlt zur Laufzeit:** Die Spec verspricht eine eigene
   Meldung der Lane. Gebaut wird sie in Task 9: Vor dem Start der
   Coverage-Befehle prüft `Run` mit `os.Stat` jede Datei, die `Plan` als
   `Job.Reads` einträgt (die aufgelösten Pfade von `{coverprofile}` und
   `{coverdata}` aus `Commands`); fehlt eine, ist die Lane `failed` mit
   `<pfad> is missing: the measuring run did not write it`.

---

## File Structure

| Datei | Verantwortung |
|---|---|
| `internal/shellwords/shellwords.go` | `Split(s string) ([]string, error)` — zieht aus `internal/cases/runner.go` um |
| `internal/child/child.go` | `Spec`, `Result`, `Run`; Fristen, Absaugen, Textnormalisierung |
| `internal/child/child_windows.go` | Job Object, suspendierter Start, `NtResumeProcess`, Baumtötung |
| `internal/child/child_other.go` | Prozessgruppe, `kill(-pgid)` |
| `internal/detect/detect.go` (ändern) | `Facts.Areas map[string][]string` |
| `internal/verify/schema.go` | `Lane`, `Override`, `Config`, `ReadConfig`, `ParseConfig`, `parseLane` |
| `internal/verify/presets.toml` | eingebettete Presets, Endungstabelle |
| `internal/verify/presets.go` | `Presets`, `LoadPresets`, `validatePresets` |
| `internal/verify/tests.go` | `HasTests(root string, patterns []string) bool` |
| `internal/verify/effective.go` | `Resolved`, `Effective`, `Resolve` |
| `internal/verify/plan.go` | `Scope`, `Request`, `Job`, `PlanEnv`, `Plan`, `ExpandProfile` |
| `internal/verify/run.go` | `State`, `Outcome`, `RunOptions`, `Run` |
| `internal/verify/report.go` | `WriteCheck`, `CheckVerdict`, `WriteEdit` |
| `internal/verify/show.go` | `WriteShow` |
| `internal/verify/cover.go` | `NewRunID`, `CoverPaths`, `CleanCover` |
| `internal/verify/gocover/gocover.go` | zieht aus `internal/dev/covergate` um; `ModulePath`, `Total` |
| `internal/cli/check.go` (ändern) | reservierte Namen, Profile, `--root`, `--show`, `-v`, `gocover` |
| `internal/cli/dev.go` (ändern) | `covergate` entfällt |
| `internal/hooks/post_edit.go` (neu geschrieben) | Nutzlast → `verify.Plan(ScopeEdit)` → `verify.Run` |
| `internal/cli/hook.go` (ändern) | `--budget` für `post-tool-use` |
| `internal/dev/faketool/faketool.go`, `_faketool/main.go` | Werkzeug-Attrappe fürs Aufzeichnen und die Naht der Fallsuite |
| `internal/dev/importcases/importcases.go` (ändern) | alte `[verify]` falten |
| `internal/cases/{case,runner,lanes}.go` (ändern/neu) | Vergleichsklasse `lanes` |
| `internal/cli/cases_2a_test.go` | Fallsuite 2a |
| `testdata/cases/2a-worlds/`, `2a-source/`, `2a-map.toml`, `2a/` | Welten, Aufzeichnungen, Übersetzung, übersetzte Fälle |
| `ci/gate.sh`, `.githooks/pre-commit`, `.github/workflows/ci.yml` (ändern) | Selbstnutzung, Eingabeliste, Linux-Test von `child` |
| `docs/.superpowers/parity/stufe-2a.md` | Abweichungsliste, Mutationsrunde |
| `docs/{en,de}/*.md`, `README.md`, `README.de.md`, Fusions-Spec, 2a-Spec (ändern) | Doku |

---

### Task 0: Arbeitsort prüfen

**Files:** keine.

**Interfaces:** keine.

- [ ] **Schritt 1: Ort und Basis lesen**

```sh
git branch --show-current
git log -1 --format='%h %an <%ae>'
git status --porcelain
```

Erwartet: `claude/fusion-migration-teil-2-79865c`, HEAD auf dem letzten
Spec-Commit, Autor der Mensch, Arbeitsbaum sauber.

- [ ] **Schritt 2: Tor einmal leer fahren**

```sh
sh ci/gate.sh
```

Erwartet: grün. Ist es das nicht, liegt der Fehler auf `master` — melden, nicht
reparieren.

---

### Task 1: `internal/shellwords` — Zerlegen nach Shell-Wortregeln

`verify` zerlegt Befehlsstrings; das kann heute nur `internal/cases`, ein
Testpaket. Die Funktion zieht um, `cases` ruft sie weiter.

**Files:**
- Create: `internal/shellwords/shellwords.go`, `internal/shellwords/shellwords_test.go`
- Modify: `internal/cases/runner.go` (`SplitCommand` delegiert)

**Interfaces:**
- Produces: `func Split(s string) ([]string, error)` — Semantik exakt wie
  `cases.SplitCommand` heute (einfache und doppelte Anführung, Backslash außer
  in einfacher Anführung, leeres Token nur aus Anführung, Fehler bei offener
  Anführung).

- [ ] **Schritt 1: Test schreiben**

```go
package shellwords

import (
	"reflect"
	"testing"
)

func TestSplit(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`go vet ./...`, []string{"go", "vet", "./..."}},
		{`npx stylelint "**/*.{css,scss}"`, []string{"npx", "stylelint", "**/*.{css,scss}"}},
		{`a 'b c' "d\"e"`, []string{"a", "b c", `d"e`}},
		{`x ""`, []string{"x", ""}},
		{`  spaced   out  `, []string{"spaced", "out"}},
		{`a\ b`, []string{"a b"}},
	}
	for _, c := range cases {
		got, err := Split(c.in)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("Split(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestSplitRefusesAnOpenQuote(t *testing.T) {
	for _, in := range []string{`a "b`, `a 'b`, `a \`} {
		if _, err := Split(in); err == nil {
			t.Errorf("Split(%q): want error", in)
		}
	}
}
```

- [ ] **Schritt 2: Test laufen lassen, er scheitert**

Run: `go test ./internal/shellwords/`
Erwartet: FAIL, `Split` undefiniert.

- [ ] **Schritt 3: Umziehen**

Den Rumpf von `SplitCommand` aus `internal/cases/runner.go` wörtlich nach
`internal/shellwords/shellwords.go` als `Split` übernehmen (Paketkommentar:
`// Package shellwords splits a command line by shell word rules, without a
shell.`). Wirft der alte Rumpf bei einem Backslash am Ende keinen Fehler, einen
ergänzen: `if escaped { return nil, errors.New("trailing backslash") }`. In
`runner.go` bleibt:

```go
// SplitCommand splits a recorded command line; the rules live in shellwords
// because verify splits configured commands the same way.
func SplitCommand(s string) ([]string, error) { return shellwords.Split(s) }
```

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/shellwords/ ./internal/cases/ ./internal/cli/`
Erwartet: PASS. Die Fallsuiten 1a und 1b-1 benutzen `SplitCommand` und
müssen unverändert grün sein.

- [ ] **Schritt 5: Commit**

```sh
git add internal/shellwords internal/cases/runner.go
git commit -m "refactor(cases): move command splitting to internal/shellwords"
```

---

### Task 2: `internal/dev/faketool` und die Aufzeichnung an ultraloom

Der Korpus braucht Werkzeuge, die antworten, ohne zu prüfen. faketool ist eine
Exe, die unter jedem Werkzeugnamen auf dem PATH liegt und aus einer
Fixture-Datei antwortet. Dieselbe Fixture bedient später die Naht der
Fallsuite (Task 14).

**Files:**
- Create: `internal/dev/faketool/faketool.go`, `internal/dev/faketool/faketool_test.go`
- Create: `internal/dev/faketool/_faketool/main.go`
- Create: `testdata/cases/2a-worlds/<welt>/…` (siehe Schritt 6)
- Create: `testdata/cases/2a-source/…` (von `loomux dev record-case` geschrieben)

**Interfaces:**
- Produces:
  - `const FixtureName = "faketool.json"`, `const FixtureEnv = "LOOMUX_FAKE_TOOL_FIXTURE"`
  - `type Write struct { Path string \`json:"path"\`; Content string \`json:"content"\` }`
  - `type Answer struct { Prefix string \`json:"prefix"\`; Exit int \`json:"exit"\`; Stdout string \`json:"stdout"\`; SleepMS int \`json:"sleep_ms"\`; Writes []Write \`json:"writes"\` }`
  - `type Fixture struct { Answers []Answer \`json:"answers"\` }`
  - `func Load(path string) (*Fixture, error)` — fehlende Datei ⇒ leere Fixture
  - `func (f *Fixture) Match(argv []string) (Answer, bool)` — längster `Prefix`, der auf `strings.Join(argv, " ")` passt; `argv[0]` wird vorher auf seinen Basisnamen ohne `.exe` gekürzt
  - `func (f *Fixture) Apply(a Answer, dir string) error` — schreibt `Writes` relativ zu `dir`, `{{DIR}}` im Inhalt wird `filepath.ToSlash(dir)`
  - `func Main(args []string, getenv func(string) string, cwd string, stdout, stderr io.Writer) int` — die Exe: `argv = [basename(os.Args[0])] + args`; kein Treffer ⇒ stderr `faketool: no answer for "<argv>"`, Exit 127

- [ ] **Schritt 1: Tests schreiben**

```go
package faketool

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), FixtureName)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMatchTakesTheLongestPrefixAndStripsExe(t *testing.T) {
	f, err := Load(fixture(t, `{"answers":[
		{"prefix":"uv run","exit":3},
		{"prefix":"uv run pytest","exit":0,"stdout":"1 passed\n"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	a, ok := f.Match([]string{`C:\bin\uv.exe`, "run", "pytest", "-q"})
	if !ok || a.Exit != 0 || a.Stdout != "1 passed\n" {
		t.Fatalf("got %+v %v", a, ok)
	}
	if _, ok := f.Match([]string{"ruff"}); ok {
		t.Fatal("ruff has no answer")
	}
}

func TestLoadOfAMissingFileIsEmpty(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil || len(f.Answers) != 0 {
		t.Fatalf("%+v %v", f, err)
	}
}

func TestLoadRefusesBrokenJSON(t *testing.T) {
	if _, err := Load(fixture(t, `{`)); err == nil {
		t.Fatal("want error")
	}
}

func TestApplyWritesFilesWithTheDirToken(t *testing.T) {
	dir := t.TempDir()
	f := &Fixture{}
	a := Answer{Writes: []Write{{Path: "out/cover.out", Content: "mode: set\n{{DIR}}/a.go:1.1,2.2 1 1\n"}}}
	if err := f.Apply(a, dir); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "out", "cover.out"))
	if !bytes.Contains(got, []byte(filepath.ToSlash(dir)+"/a.go")) {
		t.Fatalf("%s", got)
	}
}

func TestMainAnswersAndFailsLoudly(t *testing.T) {
	p := fixture(t, `{"answers":[{"prefix":"ruff check","exit":1,"stdout":"E1 x\n"}]}`)
	getenv := func(k string) string {
		if k == FixtureEnv {
			return p
		}
		return ""
	}
	var out, errb bytes.Buffer
	if code := Main([]string{"ruff.exe", "check", "."}, getenv, t.TempDir(), &out, &errb); code != 1 || out.String() != "E1 x\n" {
		t.Fatalf("code %d out %q", code, out.String())
	}
	out.Reset()
	if code := Main([]string{"mypy"}, getenv, t.TempDir(), &out, &errb); code != 127 {
		t.Fatalf("code %d", code)
	}
}
```

Dazu je ein Test für: kaputte Fixture in `Main` (Exit 127, stderr nennt den
Pfad), `SleepMS` (mit einer Naht `var sleep = time.Sleep`, im Test ersetzt und
aufgezeichnet), `Apply`-Fehler (Ziel ist ein bestehendes Verzeichnis).

- [ ] **Schritt 2: Test laufen lassen, er scheitert**

Run: `go test ./internal/dev/faketool/`
Erwartet: FAIL, Paket leer.

- [ ] **Schritt 3: Implementieren**

```go
// Package faketool answers for a checking tool from a fixture, so that a
// recording or a replay of `check` exercises the chain and not the tools.
package faketool

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	FixtureName = "faketool.json"
	FixtureEnv  = "LOOMUX_FAKE_TOOL_FIXTURE"
)

type Write struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type Answer struct {
	Prefix  string  `json:"prefix"`
	Exit    int     `json:"exit"`
	Stdout  string  `json:"stdout"`
	SleepMS int     `json:"sleep_ms"`
	Writes  []Write `json:"writes"`
}

type Fixture struct {
	Answers []Answer `json:"answers"`
}

var sleep = time.Sleep

func Load(path string) (*Fixture, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Fixture{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f Fixture
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &f, nil
}

// The tool is named by its base name without .exe: the recorder finds it on
// PATH as uv.exe, the replay seam sees whatever argv the plan built.
func line(argv []string) string {
	head := strings.TrimSuffix(filepath.Base(argv[0]), ".exe")
	return strings.Join(append([]string{head}, argv[1:]...), " ")
}

func (f *Fixture) Match(argv []string) (Answer, bool) {
	l := line(argv)
	best, found := Answer{}, false
	for _, a := range f.Answers {
		if (l == a.Prefix || strings.HasPrefix(l, a.Prefix+" ")) && len(a.Prefix) >= len(best.Prefix) {
			best, found = a, true
		}
	}
	return best, found
}

func (f *Fixture) Apply(a Answer, dir string) error {
	for _, w := range a.Writes {
		target := filepath.Join(dir, filepath.FromSlash(w.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		body := strings.ReplaceAll(w.Content, "{{DIR}}", filepath.ToSlash(dir))
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func Main(args []string, getenv func(string) string, cwd string, stdout, stderr io.Writer) int {
	f, err := Load(getenv(FixtureEnv))
	if err != nil {
		fmt.Fprintf(stderr, "faketool: %v\n", err)
		return 127
	}
	a, ok := f.Match(args)
	if !ok {
		fmt.Fprintf(stderr, "faketool: no answer for %q\n", line(args))
		return 127
	}
	sleep(time.Duration(a.SleepMS) * time.Millisecond)
	if err := f.Apply(a, cwd); err != nil {
		fmt.Fprintf(stderr, "faketool: %v\n", err)
		return 127
	}
	io.WriteString(stdout, a.Stdout)
	return a.Exit
}
```

`_faketool/main.go` (der Unterstrich hält das Verzeichnis aus `./...`, wie bei
`fakeqmd`):

```go
package main

import (
	"os"

	"github.com/xidus90/loomux/internal/dev/faketool"
)

func main() {
	cwd, _ := os.Getwd()
	os.Exit(faketool.Main(os.Args, os.Getenv, cwd, os.Stdout, os.Stderr))
}
```

Achtung: `Main` bekommt `os.Args` **samt** Programmnamen; die Tests oben reichen
ebenso `argv[0]` mit.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/dev/faketool/ -cover`
Erwartet: PASS, 100.0 %.

- [ ] **Schritt 5: Werkzeuge für die Aufzeichnung bauen**

```sh
mkdir -p "$TMP/faketools"
go build -o "$TMP/faketools/uv.exe" ./internal/dev/faketool/_faketool
```

Dann dieselbe Exe unter jedem Namen kopieren, den die alte Kette ruft:
`uvx.exe`, `ctest.exe`, `gcovr.exe`, `clang-tidy.exe`, `eslint.exe`, `tsc.exe`,
`vitest.exe`, `godot.exe`, `go.exe`.

**Falle:** Liegt die Attrappe `uv.exe` zuerst auf dem PATH, fängt sie auch den
Aufruf der Referenz selbst. Der Rekorder ruft deshalb das echte `uv` über
seinen absoluten Pfad (`(Get-Command uv).Source` bzw. `command -v uv` vor dem
Umstellen des PATH lesen); nur die Kinder der Referenz lösen über den PATH auf.

- [ ] **Schritt 6: Welten anlegen**

Unter `testdata/cases/2a-worlds/`, je Welt die Markerdateien, eine
`faketool.json` und, wo nötig, eine alte `.ultraloom/config.toml`:

| Welt | Inhalt | zeigt |
|---|---|---|
| `python-green` | `pyproject.toml`, `tests/test_a.py`; Antworten `uvx ruff check` 0, `uv run mypy` 0, `uv run pytest` 0, `uv run coverage run` 0, `uv run coverage report` 0 | alles grün |
| `python-red-lint` | wie oben, `uvx ruff check` Exit 1 mit Befund | `lint` rot, Rest grün |
| `python-red-test` | `uv run coverage run` Exit 1 | `test` rot, `coverage` blockiert |
| `cpp` | `CMakeLists.txt` mit `enable_testing()`; `clang-tidy` 0, `ctest` 0, `gcovr` 0 | `types` alt Traceback |
| `node` | `package.json` mit `vitest`, `tsconfig.json`; `eslint` 0, `tsc` 0, `vitest run` 0 | vier Arten |
| `godot-unready` | `project.godot` ohne `.godot/global_script_class_cache.cfg` | `test` `unready` |
| `godot-no-coverage` | `project.godot` mit Cache-Datei | `coverage` `unavailable` |
| `go-only` | `go.mod`, `a.go`, `a_test.go` | alt: alles `unavailable` |
| `mixed-config` | `pyproject.toml`, `go.mod`, `.ultraloom/config.toml` mit `[verify.lint]` (ruff, `go vet`) und `[verify.test]` | Konfiguration ersetzt Preset |
| `config-empty-list` | `.ultraloom/config.toml` mit `[verify] lint = []` | Ladefehler |
| `config-after-cycle` | `[verify.after] test = "coverage"` und `coverage = "test"` | Ladefehler |
| `config-profile-unknown` | `[verify.profiles] x = ["lint", "style"]` | Ladefehler |
| `no-marker` | leer | alles `unavailable` |

Die Antworten der Fixture tragen die Kommandozeile so, wie die alte Kette sie
baut (Tabelle in der Spec, Abschnitt Presets, und `checks.py:69-121` am Tag
`loomux-1a-source`).

- [ ] **Schritt 7: Aufzeichnen**

Je Welt und je Aufruf `check lint`, `check types`, `check test`,
`check coverage`, `check all` (wo die Welt es trägt):

```sh
uvreal="<absoluter Pfad des echten uv>"
ul="C:/Users/micro/Documents/#GIT/ultraloom"
go run ./cmd/loomux dev record-case \
  --argv "$uvreal run --no-sync --project $ul ultraloom" \
  --path-prepend "$TMP/faketools" \
  --env "LOOMUX_FAKE_TOOL_FIXTURE={{WORLD}}/faketool.json" \
  --cmd "ultraloom check all --root {{WORLD}}" \
  --world testdata/cases/2a-worlds/python-green \
  --out testdata/cases/2a-source/check/python-green-all \
  --notes "ultraloom at tag loomux-1a-source (9d01a60); tools answered by faketool" \
  --compare lanes
```

`--compare lanes` setzt die Klasse, die Task 14 im Paket `cases` einführt; bis
dahin steht sie nur als Datei im Fall. Ausnahme: Fälle mit Ladefehler bekommen
`--compare message` (nur Exit-Code). Vor dem Aufzeichnen einmal
`git -C "$ul" rev-parse HEAD` lesen: `9d01a60…`, sonst Halt und den Menschen
fragen.

- [ ] **Schritt 8: Policy-Schutz vormerken und committen**

Die Aufzeichnungen sind Beleg und werden nie editiert. Die Regel dafür trägt der
Mensch in Task 15 ein; bis dahin werden `2a-source/` nur durch den Rekorder
geschrieben.

```sh
git add internal/dev/faketool testdata/cases/2a-worlds testdata/cases/2a-source
git commit -m "test(cases): record the stage 2a check chain from ultraloom"
```

---

### Task 3: `internal/child` — ein Kind mit Frist, der Baum stirbt mit

**Files:**
- Create: `internal/child/child.go`, `internal/child/child_windows.go`, `internal/child/child_other.go`
- Test: `internal/child/child_test.go`, `internal/child/child_windows_test.go`, `internal/child/child_other_test.go`

**Interfaces:**
- Consumes: `gitenv.Clean(parent []string) []string`
- Produces:
  - `type Spec struct { Argv []string; Dir string; Env []string; Timeout time.Duration }` — `Env` sind zusätzliche `KEY=VALUE`; `Timeout` 0 heißt keine Frist
  - `type Result struct { Code int; Stdout string; Stderr string; TimedOut bool; OutputAbandoned bool; Err error }` — `Code` ist `-1`, wenn es keinen gibt; `Err` nur bei Startfehler
  - `func Run(s Spec) Result`
  - `const DrainGrace = 5 * time.Second`

Vertrag, übernommen aus `process.py` (Spec, Abschnitt Pakete):

- Umgebung: `gitenv.Clean(os.Environ())`, dann `s.Env`, dann
  `PYTHONIOENCODING=utf-8`. Der erzwungene Wert steht **zuletzt**, damit der
  Aufrufer ihn nicht überschreibt (ein späteres `KEY=VALUE` gewinnt in `exec`).
- stdout und stderr getrennt, je ein Leser; Text über
  `strings.ToValidUTF8(…, "�")`, `\r\n` und `\r` werden `\n`.
- Frist: nach `Timeout` wird der Baum getötet, `TimedOut = true`. Tötung,
  Abholen und Leser teilen **eine** Nachfrist `DrainGrace`.
- Sauberes Ende, aber ein Leser wird binnen `DrainGrace` nicht fertig (ein
  Enkel hält die Pipe): Baum töten, `OutputAbandoned = true`.

- [ ] **Schritt 1: Plattformunabhängige Tests schreiben (`child_test.go`)**

Die Tests starten das Test-Binary selbst als Kind, damit kein fremdes Programm
vorausgesetzt ist:

```go
package child

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// TestHelperProcess is the child every test here starts: the test binary
// itself, told by environment what to do.
func TestHelperProcess(t *testing.T) {
	switch os.Getenv("CHILD_HELPER") {
	case "":
		return
	case "echo":
		fmt.Print("out\r\nline\r")
		fmt.Fprint(os.Stderr, "err\n")
		os.Exit(3)
	case "env":
		fmt.Print(os.Getenv("PYTHONIOENCODING"), "|", os.Getenv("GIT_DIR"), "|", os.Getenv("EXTRA"))
	case "sleep":
		time.Sleep(time.Minute)
	case "badutf8":
		os.Stdout.Write([]byte{'a', 0xff, 'b'})
	}
	os.Exit(0)
}

func helper(mode string, extra ...string) Spec {
	return Spec{
		Argv: []string{os.Args[0], "-test.run=^TestHelperProcess$"},
		Env:  append([]string{"CHILD_HELPER=" + mode}, extra...),
	}
}

func TestRunSeparatesStreamsAndNormalisesNewlines(t *testing.T) {
	r := Run(helper("echo"))
	if r.Code != 3 || r.Stdout != "out\nline\n" || r.Stderr != "err\n" || r.TimedOut || r.Err != nil {
		t.Fatalf("%+v", r)
	}
}

func TestRunForcesTheEnvironmentAndStripsGit(t *testing.T) {
	t.Setenv("GIT_DIR", "/elsewhere")
	r := Run(helper("env", "EXTRA=1", "PYTHONIOENCODING=latin-1"))
	if r.Stdout != "utf-8||1" {
		t.Fatalf("%q", r.Stdout)
	}
}

func TestRunKillsAtTheDeadline(t *testing.T) {
	s := helper("sleep")
	s.Timeout = 300 * time.Millisecond
	start := time.Now()
	r := Run(s)
	if !r.TimedOut || r.Code == 0 || time.Since(start) > 10*time.Second {
		t.Fatalf("%+v after %v", r, time.Since(start))
	}
}

func TestRunReplacesInvalidUTF8(t *testing.T) {
	if r := Run(helper("badutf8")); r.Stdout != "a�b" {
		t.Fatalf("%q", r.Stdout)
	}
}

func TestRunReportsAStartFailure(t *testing.T) {
	r := Run(Spec{Argv: []string{"loomux-no-such-tool-xyz"}})
	if r.Err == nil || r.Code != -1 || !strings.Contains(r.Err.Error(), "loomux-no-such-tool-xyz") {
		t.Fatalf("%+v", r)
	}
}
```

- [ ] **Schritt 2: Baum-Tests schreiben (`child_windows_test.go`)**

Der Beweis ist die Zählung im Job selbst, nicht eine Prozessliste der Maschine
(ein fremdes `ping` ließe die flackern). `x/sys` v0.18.0 kennt
`JOBOBJECT_BASIC_ACCOUNTING_INFORMATION` nicht (am 2026-09-18 nachgemessen),
die Struktur steht deshalb im Test:

```go
//go:build windows

package child

import (
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type accounting struct {
	TotalUserTime, TotalKernelTime, ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime int64
	TotalPageFaultCount, TotalProcesses, ActiveProcesses, TotalTerminatedProcesses    uint32
}

// cmd starts ping in the background and a second one in front: without the
// job, the background ping would outlive the kill of cmd.
func TestRunKillsTheGrandchildAtTheDeadline(t *testing.T) {
	var active, total uint32 = 99, 0
	old := afterKill
	afterKill = func(job windows.Handle) {
		var a accounting
		windows.QueryInformationJobObject(job, 1, uintptr(unsafe.Pointer(&a)), uint32(unsafe.Sizeof(a)), nil)
		active, total = a.ActiveProcesses, a.TotalProcesses
	}
	t.Cleanup(func() { afterKill = old })
	r := Run(Spec{
		Argv:    []string{"cmd", "/c", "start /b ping -n 60 127.0.0.1 >nul & ping -n 60 127.0.0.1 >nul"},
		Timeout: time.Second,
	})
	if !r.TimedOut || total < 3 || active != 0 {
		t.Fatalf("%+v total %d active %d", r, total, active)
	}
}

func TestRunAbandonsOutputAGrandchildHolds(t *testing.T) {
	start := time.Now()
	r := Run(Spec{Argv: []string{"cmd", "/c", "start /b ping -n 60 127.0.0.1"}})
	if !r.OutputAbandoned || time.Since(start) > DrainGrace+5*time.Second {
		t.Fatalf("%+v after %v", r, time.Since(start))
	}
}
```

Dazu je ein Test pro Fehlerzweig in `start` und `adopt` (Schritt 6): Er ersetzt
die Paketvariable des Systemaufrufs durch einen, der einen Fehler liefert, und
erwartet `Result.Err != nil` bzw. keinen hängenden Prozess.

Und ein Test für den Fall, dass das Kind die Tötung überlebt, denn dort liegt
die Gefahr, zweimal auf dieselbe Frist zu warten: Er ersetzt `resume` durch
eine Funktion, die nichts tut (das Kind bleibt suspendiert und endet nie von
selbst), und `killTree` — die Paketvariable, über die `tree.kill` den Job
beendet — durch eine, die nichts tötet. Erwartet: `Run` kehrt nach etwa
`DrainGrace` mit `TimedOut` und `OutputAbandoned` zurück, statt zu hängen
(Testfrist 3 × `DrainGrace`). Im Aufräumen des Tests den Job wirklich beenden.
`tree.kill` ruft dafür `killTree(t.job)` mit `var killTree = func(job
windows.Handle) { windows.TerminateJobObject(job, 1) }`.

- [ ] **Schritt 3: POSIX-Test schreiben (`child_other_test.go`)**

```go
//go:build !windows

package child

import (
	"testing"
	"time"
)

func TestRunKillsTheProcessGroup(t *testing.T) {
	r := Run(Spec{Argv: []string{"sh", "-c", "sleep 60 & sleep 60"}, Timeout: 300 * time.Millisecond})
	if !r.TimedOut {
		t.Fatalf("%+v", r)
	}
}
```

- [ ] **Schritt 4: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/child/`
Erwartet: FAIL, Paket leer.

- [ ] **Schritt 5: `child.go` implementieren**

```go
// Package child runs one checking tool: a deadline, both streams, and the
// whole process tree dead afterwards -- not only the process it started.
package child

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/gitenv"
)

// DrainGrace is shared by the kill, the reap and the readers after a
// deadline or a clean exit; it is one budget, not one per step.
const DrainGrace = 5 * time.Second

// pipe is a seam: only exhausted handles make os.Pipe fail.
var pipe = os.Pipe

type Spec struct {
	Argv    []string
	Dir     string
	Env     []string
	Timeout time.Duration
}

type Result struct {
	Code            int
	Stdout          string
	Stderr          string
	TimedOut        bool
	OutputAbandoned bool
	Err             error
}

func environ(extra []string) []string {
	env := append(gitenv.Clean(os.Environ()), extra...)
	return append(env, "PYTHONIOENCODING=utf-8")
}

func text(b []byte) string {
	s := strings.ToValidUTF8(string(b), "�")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// lockedBuffer: Run may read a stream a grandchild still writes to, after it
// gave up on the reader.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}

type reader struct {
	buf  lockedBuffer
	done chan struct{}
}

func drain(r io.Reader) *reader {
	rd := &reader{done: make(chan struct{})}
	go func() {
		io.Copy(&rd.buf, r)
		close(rd.done)
	}()
	return rd
}

func errStart(argv0 string, err error) error {
	return errors.Join(errors.New(argv0), err)
}

// Run uses os.Pipe and Process.Wait, not StdoutPipe and cmd.Wait: cmd.Wait
// also waits for the pipes to close, so a grandchild holding one would hide
// that the child itself had ended, and OutputAbandoned could not be seen.
func Run(s Spec) Result {
	cmd := exec.Command(s.Argv[0], s.Argv[1:]...)
	cmd.Dir = s.Dir
	cmd.Env = environ(s.Env)
	outR, outW, err := pipe()
	if err != nil {
		return Result{Code: -1, Err: err}
	}
	errR, errW, err := pipe()
	if err != nil {
		outR.Close()
		outW.Close()
		return Result{Code: -1, Err: err}
	}
	cmd.Stdout, cmd.Stderr = outW, errW
	t, err := start(cmd)
	// The parent's write ends close either way: only the child's copies may
	// keep a reader waiting.
	outW.Close()
	errW.Close()
	if err != nil {
		outR.Close()
		errR.Close()
		return Result{Code: -1, Err: errStart(s.Argv[0], err)}
	}
	out, errOut := drain(outR), drain(errR)
	defer outR.Close()
	defer errR.Close()

	var state *os.ProcessState
	exited := make(chan struct{})
	go func() { state, _ = cmd.Process.Wait(); close(exited) }()

	var deadline <-chan time.Time
	if s.Timeout > 0 {
		timer := time.NewTimer(s.Timeout)
		defer timer.Stop()
		deadline = timer.C
	}
	res := Result{Code: -1}
	// One point in time, not one channel: a time.After channel fires once,
	// and a second wait on it after it fired would block for ever.
	var until time.Time
	select {
	case <-exited:
		until = time.Now().Add(DrainGrace)
	case <-deadline:
		res.TimedOut = true
		t.kill()
		until = time.Now().Add(DrainGrace)
		select {
		case <-exited:
		case <-time.After(time.Until(until)):
		}
	}
	for _, rd := range []*reader{out, errOut} {
		select {
		case <-rd.done:
		case <-time.After(time.Until(until)):
			res.OutputAbandoned = true
		}
	}
	if res.OutputAbandoned {
		t.kill()
	}
	t.release()
	select {
	case <-exited:
		res.Code = state.ExitCode()
	default:
	}
	res.Stdout, res.Stderr = text(out.buf.bytes()), text(errOut.buf.bytes())
	return res
}
```

Die beiden `os.Pipe`-Fehlerzweige erreicht kein Test ohne erschöpfte Handles:
`os.Pipe` wird über `var pipe = os.Pipe` gerufen und im Test durch eine
Funktion ersetzt, die beim ersten bzw. zweiten Aufruf scheitert.
`go test -race ./internal/child/` muss sauber sein.

- [ ] **Schritt 6: `child_windows.go` implementieren**

```go
//go:build windows

package child

import (
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Seams for the error arms, and afterKill for the test that counts the job.
var (
	createJob = windows.CreateJobObject
	setJob    = windows.SetInformationJobObject
	openProc  = windows.OpenProcess
	assignJob = windows.AssignProcessToJobObject
	resume    = ntResume
	afterKill = func(job windows.Handle) {}
)

type tree struct{ job windows.Handle }

// NtResumeProcess because exec closes the thread handle ResumeThread would
// need; measured on 2026-09-18 with cmd and two ping grandchildren.
func ntResume(h windows.Handle) error {
	status, _, _ := windows.NewLazySystemDLL("ntdll.dll").NewProc("NtResumeProcess").Call(uintptr(h))
	if status != 0 {
		return windows.NTStatus(status)
	}
	return nil
}

// start creates the child suspended, puts it into a job that dies with its
// handle, and only then lets it run: no grandchild is born outside the job.
func start(cmd *exec.Cmd) (*tree, error) {
	job, err := createJob(nil, nil)
	if err != nil {
		return nil, err
	}
	t := &tree{job: job}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := setJob(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		t.release()
		return nil, err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_SUSPENDED}
	if err := cmd.Start(); err != nil {
		t.release()
		return nil, err
	}
	if err := t.adopt(cmd.Process.Pid); err != nil {
		cmd.Process.Kill()
		t.release()
		return nil, err
	}
	return t, nil
}

func (t *tree) adopt(pid int) error {
	h, err := openProc(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_SUSPEND_RESUME, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	if err := assignJob(t.job, h); err != nil {
		return err
	}
	return resume(h)
}

var killTree = func(job windows.Handle) { windows.TerminateJobObject(job, 1) }

func (t *tree) kill() {
	killTree(t.job)
	afterKill(t.job)
}

func (t *tree) release() { windows.CloseHandle(t.job) }
```

- [ ] **Schritt 7: `child_other.go` implementieren**

```go
//go:build !windows

package child

import (
	"os/exec"
	"syscall"
)

type tree struct{ pid int }

//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func start(cmd *exec.Cmd) (*tree, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &tree{pid: cmd.Process.Pid}, nil
}

//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func (t *tree) kill() { syscall.Kill(-t.pid, syscall.SIGKILL) }

//coverage:exempt POSIX-only; there is no job to release
func (t *tree) release() {}
```

- [ ] **Schritt 8: Tests laufen lassen**

Run: `go test -race -cover ./internal/child/`
Erwartet: PASS, 100.0 % unter Windows.

- [ ] **Schritt 9: Commit**

```sh
git add internal/child
git commit -m "feat(child): run a tool with a deadline and kill its whole tree"
```

---

### Task 4: `internal/detect` — Bereiche je Stack

**Files:**
- Modify: `internal/detect/detect.go:20-70`
- Test: `internal/detect/detect_test.go`

**Interfaces:**
- Produces: `Facts.Areas map[string][]string` — je Stack die Bereiche, in
  denen ein Signal traf; `"."` für das Wurzelverzeichnis, sonst der Name des
  Unterverzeichnisses; sortiert, ohne Dubletten. `Stacks` und `GodotDir`
  bleiben unverändert.

- [ ] **Schritt 1: Test schreiben**

```go
func TestDetectRecordsEveryAreaAStackIsFoundIn(t *testing.T) {
	root := fstest.MapFS{
		"go.mod":              {Data: []byte("module x\n")},
		"web/package.json":    {Data: []byte("{}")},
		"web/tsconfig.json":   {Data: []byte("{}")},
		"admin/package.json":  {Data: []byte("{}")},
		"admin/tsconfig.json": {Data: []byte("{}")},
		"game/project.godot":  {Data: []byte("")},
	}
	f := Detect(root)
	want := map[string][]string{
		"go":         {"."},
		"typescript": {"admin", "web"},
		"godot":      {"game"},
		"gdscript":   {"game"},
	}
	for stack, areas := range want {
		if !reflect.DeepEqual(f.Areas[stack], areas) {
			t.Errorf("%s: %v, want %v", stack, f.Areas[stack], areas)
		}
	}
	if f.GodotDir != "game" {
		t.Errorf("GodotDir %q", f.GodotDir)
	}
}
```

- [ ] **Schritt 2: Test laufen lassen, er scheitert**

Run: `go test ./internal/detect/ -run TestDetectRecordsEveryArea`
Erwartet: FAIL, `f.Areas` undefiniert.

- [ ] **Schritt 3: Implementieren**

In `Facts`:

```go
	// Areas names, per stack, every area a signal of that stack matched in:
	// "." for the root, else the top-level directory. A check runs one lane
	// per area, so two web apps under one root are linted where each lives.
	Areas map[string][]string
```

In `Detect` die Schleife ersetzen — kein `break` mehr, jede Fläche zählt:

```go
	found := map[string]bool{}
	areasOf := map[string]map[string]bool{}
	for _, sig := range signals {
		for _, area := range areas {
			if !matches(root, area, sig) {
				continue
			}
			for _, stack := range sig.stacks {
				found[stack] = true
				if areasOf[stack] == nil {
					areasOf[stack] = map[string]bool{}
				}
				areasOf[stack][area] = true
			}
		}
	}
	facts := Facts{Stacks: sorted(found), Areas: map[string][]string{}, Ambiguous: doubts(root, areas), GodotDir: godotArea(root, areas)}
	for stack, set := range areasOf {
		facts.Areas[stack] = sorted(set)
	}
```

Vorher prüfen, wie `searchAreas` das Wurzelverzeichnis nennt (laut Digest
`"."`); `sorted` sortiert Byte-geordnet, `"."` steht damit vor jedem Namen.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/detect/ ./internal/hooks/ -cover`
Erwartet: PASS, `detect` 100.0 %.

- [ ] **Schritt 5: Commit**

```sh
git add internal/detect
git commit -m "feat(detect): record every area a stack is found in"
```

---

### Task 5: `internal/verify/schema.go` — `[verify]` streng laden

**Files:**
- Create: `internal/verify/schema.go`, `internal/verify/schema_test.go`

**Interfaces:**
- Consumes: `config.ManifestPath(root string) string`, `shellwords.Split`
- Produces:
  - `func Kinds() []string` → `[]string{"lint", "types", "test", "coverage"}`
  - `func StackNames() []string` → `go, python, typescript, vue, svelte, css, html, gdscript, cpp, shell, sql, rust, wiki, project` (in dieser Reihenfolge; Aufrufer sortieren selbst)
  - `func Reserved(name string) bool` — `gofmt`, `commit-msg`, `gocover`, `all` und die vier Arten
  - `type Lane struct { Commands []string; OnFile []string; Threaded bool; Measuring string; Measure string; After string; Off bool }`
  - `type Override struct { Lane Lane; Replace bool; Set map[string]bool }` — `Replace`: String- oder Listenform, die Preset-Lane fällt ganz weg; `Set`: welche Schlüssel eine Tabellenform nennt; `Lane.Off` bei `false`
  - `type Config struct { MaxParallel int; Timeout time.Duration; Profiles map[string][]string; Stacks map[string]map[string]Override; ImportCheck bool }`
  - `func ReadConfig(root string) (Config, error)` — fehlende Datei oder fehlende `[verify]` ⇒ Vorgaben
  - `func ParseConfig(path string, doc map[string]any) (Config, error)`
  - `func parseLane(owner, kind string, value any) (Override, error)` — auch für `presets.toml` (Task 6)
  - `const placeholderFile = "{file}"`

Vorgaben: `MaxParallel = runtime.NumCPU()`, `Timeout = 600 * time.Second`,
`Profiles = {"edit": ["lint","types"], "precommit": ["lint","types","test","coverage"]}`,
`ImportCheck = true`.

Fehlertexte, alle mit `<pfad>: ` davor (Stil von `ReadPolicy`):

| Fall | Text |
|---|---|
| unbekannter Schlüssel unter `[verify]` | `[verify] has unknown key "x"` |
| alte Top-Level-Art | `[verify].types is the old form; name the stack: [verify.<stack>].types` |
| unbekannter Stack | `[verify.x] is not a stack; stacks are: go, python, …` |
| unbekannter Schlüssel im Stack | `[verify.go] has unknown key "x"` |
| `import_check` außerhalb `gdscript` | `[verify.go] has unknown key "import_check"` |
| `wiki` mit anderem Wert als `lint = false` | `[verify.wiki].lint can only be false` |
| `true` | `[verify.go].lint = true is not a command; leave the key out to keep the preset` |
| leere Liste / leerer Befehl | `[verify.go].lint is empty` / `[verify.go].lint #2 is empty` |
| offene Anführung | `[verify.go].lint #1: <fehler von shellwords>` |
| `{file}` außerhalb `on_file` | `[verify.go].lint uses {file}, which only on_file knows` |
| unbekannter Tabellenschlüssel | `[verify.go.lint] has unknown key "x"` |
| `measuring`/`measure`/`after` bei `lint`/`types` | `[verify.go.lint] cannot have measuring` |
| `after` auf eine unbekannte Art | `[verify.go.coverage].after names unknown kind "x"` |
| `after`-Zyklus | `[verify.go] after forms a cycle: test -> coverage -> test` |
| `max_parallel` < 1 oder kein Integer | `[verify].max_parallel must be a positive integer, found …` |
| `timeout` < 1 oder kein Integer | `[verify].timeout must be a positive number of seconds, found …` |
| Profil mit reserviertem Namen | `[verify.profiles].lint collides with a reserved name` |
| Profil leer / unbekannte Art | `[verify.profiles].x is empty` / `[verify.profiles].x names unknown kind "style"` |
| `all` überschrieben | `[verify.profiles].all collides with a reserved name` |

- [ ] **Schritt 1: Tests schreiben**

Tabellengetrieben: je Zeile der Fehlertabelle ein TOML-Schnipsel und der
erwartete Textteil; dazu die Gutfälle.

```go
package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
)

func parse(t *testing.T, src string) (Config, error) {
	t.Helper()
	doc := map[string]any{}
	if _, err := toml.Decode(src, &doc); err != nil {
		t.Fatalf("fixture is not TOML: %v", err)
	}
	return ParseConfig("cfg.toml", doc)
}

func TestParseConfigRefuses(t *testing.T) {
	cases := []struct{ src, want string }{
		{"[verify]\nparallelism = 4", `[verify] has unknown key "parallelism"`},
		{"[verify]\ntypes = \"mypy\"", "[verify].types is the old form"},
		{"[verify.cobol]\nlint = \"x\"", "[verify.cobol] is not a stack"},
		{"[verify.go]\nstyle = \"x\"", `[verify.go] has unknown key "style"`},
		{"[verify.go]\nimport_check = true", `[verify.go] has unknown key "import_check"`},
		{"[verify.wiki]\nlint = \"x\"", "[verify.wiki].lint can only be false"},
		{"[verify.go]\nlint = true", "is not a command"},
		{"[verify.go]\nlint = []", "[verify.go].lint is empty"},
		{"[verify.go]\nlint = [\"a\", \" \"]", "[verify.go].lint #2 is empty"},
		{"[verify.go]\nlint = \"a 'b\"", "[verify.go].lint #1:"},
		{"[verify.go]\nlint = \"x {file}\"", "only on_file knows"},
		{"[verify.go.lint]\ncommands = [\"x\"]\nfoo = 1", `[verify.go.lint] has unknown key "foo"`},
		{"[verify.go.lint]\ncommands = [\"x\"]\nmeasuring = \"y\"", "[verify.go.lint] cannot have measuring"},
		{"[verify.go.coverage]\ncommands = [\"x\"]\nafter = \"style\"", `names unknown kind "style"`},
		{"[verify.go.test]\ncommands = [\"x\"]\nafter = \"coverage\"\n[verify.go.coverage]\ncommands = [\"y\"]\nafter = \"test\"", "after forms a cycle"},
		{"[verify]\nmax_parallel = 0", "[verify].max_parallel must be a positive integer"},
		{"[verify]\ntimeout = \"10s\"", "[verify].timeout must be a positive number of seconds"},
		{"[verify.profiles]\nlint = [\"lint\"]", "collides with a reserved name"},
		{"[verify.profiles]\nall = [\"lint\"]", "collides with a reserved name"},
		{"[verify.profiles]\nx = []", "[verify.profiles].x is empty"},
		{"[verify.profiles]\nx = [\"style\"]", `names unknown kind "style"`},
	}
	for _, c := range cases {
		_, err := parse(t, c.src)
		if err == nil || !strings.HasPrefix(err.Error(), "cfg.toml: ") || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: got %v, want %q", c.src, err, c.want)
		}
	}
}

func TestParseConfigDefaults(t *testing.T) {
	cfg, err := parse(t, "")
	if err != nil || cfg.MaxParallel < 1 || cfg.Timeout != 600*time.Second || !cfg.ImportCheck {
		t.Fatalf("%+v %v", cfg, err)
	}
	if strings.Join(cfg.Profiles["edit"], ",") != "lint,types" || len(cfg.Profiles["precommit"]) != 4 {
		t.Fatalf("%v", cfg.Profiles)
	}
}

func TestParseConfigForms(t *testing.T) {
	cfg, err := parse(t, `
[verify]
max_parallel = 3
timeout = 90
[verify.profiles]
edit = ["lint"]
[verify.go]
lint = "go vet ./..."
types = false
[verify.go.test]
measuring = "go test -coverprofile={coverprofile} ./..."
[verify.typescript.lint]
commands = ["npx eslint ."]
on_file = ["npx eslint --cache {file}"]
threaded = true
[verify.gdscript]
import_check = false
[verify.wiki]
lint = false
`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxParallel != 3 || cfg.Timeout != 90*time.Second || cfg.ImportCheck {
		t.Fatalf("%+v", cfg)
	}
	golint := cfg.Stacks["go"]["lint"]
	if !golint.Replace || golint.Lane.Commands[0] != "go vet ./..." {
		t.Fatalf("%+v", golint)
	}
	if !cfg.Stacks["go"]["types"].Lane.Off {
		t.Fatal("types = false")
	}
	gotest := cfg.Stacks["go"]["test"]
	if gotest.Replace || !gotest.Set["measuring"] || gotest.Set["commands"] {
		t.Fatalf("%+v", gotest)
	}
	ts := cfg.Stacks["typescript"]["lint"]
	if !ts.Lane.Threaded || ts.Lane.OnFile[0] != "npx eslint --cache {file}" {
		t.Fatalf("%+v", ts)
	}
	if !cfg.Stacks["wiki"]["lint"].Lane.Off {
		t.Fatal("wiki lint = false")
	}
}

func TestReadConfigOfAMissingFileIsTheDefault(t *testing.T) {
	cfg, err := ReadConfig(t.TempDir())
	if err != nil || cfg.Timeout != 600*time.Second {
		t.Fatalf("%+v %v", cfg, err)
	}
}

func TestReadConfigNamesTheFileOfBrokenTOML(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify"), 0o644)
	if _, err := ReadConfig(root); err == nil || !strings.Contains(err.Error(), "config.toml") {
		t.Fatalf("%v", err)
	}
}
```

Zusätzlich je ein Test für: `[verify]` ist kein Table, `[verify.go]` ist kein
Table, `[verify.profiles]` ist kein Table, ein Profilwert ist keine Liste, ein
Listenelement ist kein String, `threaded` ist kein Bool, `commands` fehlt in
einer Tabelle **ohne** weitere Schlüssel (leere Tabelle ⇒ `is empty`),
`measuring` ist kein String, `import_check` ist kein Bool, Lesefehler der Datei
(ein Verzeichnis statt der Datei). Jede Zeile aus `ParseConfig` muss von einem
Test erreicht werden, sonst verfehlt das Tor die 100 %.

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/verify/ -run 'ParseConfig|ReadConfig'`
Erwartet: FAIL, `ParseConfig` undefiniert.

- [ ] **Schritt 3: Implementieren**

```go
package verify

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/shellwords"
)

const placeholderFile = "{file}"

func Kinds() []string { return []string{"lint", "types", "test", "coverage"} }

func StackNames() []string {
	return []string{"go", "python", "typescript", "vue", "svelte", "css", "html",
		"gdscript", "cpp", "shell", "sql", "rust", "wiki", "project"}
}

func Reserved(name string) bool {
	return slices.Contains([]string{"gofmt", "commit-msg", "gocover", "all"}, name) || slices.Contains(Kinds(), name)
}

type Lane struct {
	Commands  []string
	OnFile    []string
	Threaded  bool
	Measuring string
	Measure   string
	After     string
	Off       bool
}

// Override is one [verify.<stack>].<kind> entry. A string or a list stands
// for the whole lane: whoever writes a command means it as written, which is
// also what the old chain did when a project configured a kind. A table only
// changes the keys it names.
type Override struct {
	Lane    Lane
	Replace bool
	Set     map[string]bool
}

type Config struct {
	MaxParallel int
	Timeout     time.Duration
	Profiles    map[string][]string
	Stacks      map[string]map[string]Override
	ImportCheck bool
}

func defaults() Config {
	return Config{
		MaxParallel: runtime.NumCPU(),
		Timeout:     600 * time.Second,
		Profiles: map[string][]string{
			"edit":      {"lint", "types"},
			"precommit": {"lint", "types", "test", "coverage"},
		},
		Stacks:      map[string]map[string]Override{},
		ImportCheck: true,
	}
}

func ReadConfig(root string) (Config, error) {
	path := config.ManifestPath(root)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return defaults(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	doc := map[string]any{}
	if _, err := toml.Decode(string(data), &doc); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return ParseConfig(path, doc)
}

func ParseConfig(path string, doc map[string]any) (Config, error) {
	cfg := defaults()
	raw, ok := doc["verify"]
	if !ok {
		return cfg, nil
	}
	if err := parseVerify(&cfg, raw); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func parseVerify(cfg *Config, raw any) error {
	table, ok := raw.(map[string]any)
	if !ok {
		return errors.New("[verify] must be a table")
	}
	for _, key := range sortedKeys(table) {
		value := table[key]
		switch {
		case key == "max_parallel":
			n, ok := value.(int64)
			if !ok || n < 1 {
				return fmt.Errorf("[verify].max_parallel must be a positive integer, found %v", value)
			}
			cfg.MaxParallel = int(n)
		case key == "timeout":
			n, ok := value.(int64)
			if !ok || n < 1 {
				return fmt.Errorf("[verify].timeout must be a positive number of seconds, found %v", value)
			}
			cfg.Timeout = time.Duration(n) * time.Second
		case key == "profiles":
			if err := parseProfiles(cfg, value); err != nil {
				return err
			}
		case slices.Contains(Kinds(), key):
			return fmt.Errorf("[verify].%s is the old form; name the stack: [verify.<stack>].%s", key, key)
		case slices.Contains(StackNames(), key):
			if err := parseStack(cfg, key, value); err != nil {
				return err
			}
		case isTable(value):
			return fmt.Errorf("[verify.%s] is not a stack; stacks are: %s", key, strings.Join(StackNames(), ", "))
		default:
			return fmt.Errorf("[verify] has unknown key %q", key)
		}
	}
	return nil
}

func isTable(v any) bool { _, ok := v.(map[string]any); return ok }
```

`parseProfiles`, `parseStack` und `parseLane` nach der Fehlertabelle oben:

```go
func parseProfiles(cfg *Config, value any) error {
	table, ok := value.(map[string]any)
	if !ok {
		return errors.New("[verify.profiles] must be a table")
	}
	for _, name := range sortedKeys(table) {
		if Reserved(name) {
			return fmt.Errorf("[verify.profiles].%s collides with a reserved name", name)
		}
		list, ok := table[name].([]any)
		if !ok {
			return fmt.Errorf("[verify.profiles].%s must be a list of kinds", name)
		}
		if len(list) == 0 {
			return fmt.Errorf("[verify.profiles].%s is empty", name)
		}
		kinds := []string{}
		for _, item := range list {
			kind, ok := item.(string)
			if !ok || !slices.Contains(Kinds(), kind) {
				return fmt.Errorf("[verify.profiles].%s names unknown kind %q", name, fmt.Sprint(item))
			}
			kinds = append(kinds, kind)
		}
		cfg.Profiles[name] = kinds
	}
	return nil
}

func parseStack(cfg *Config, stack string, value any) error {
	table, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("[verify.%s] must be a table", stack)
	}
	lanes := map[string]Override{}
	for _, key := range sortedKeys(table) {
		v := table[key]
		switch {
		case key == "import_check" && stack == "gdscript":
			b, ok := v.(bool)
			if !ok {
				return fmt.Errorf("[verify.gdscript].import_check must be a boolean")
			}
			cfg.ImportCheck = b
		case slices.Contains(Kinds(), key):
			if stack == "wiki" && (key != "lint" || v != false) {
				return errors.New("[verify.wiki].lint can only be false")
			}
			o, err := parseLane("verify."+stack, key, v)
			if err != nil {
				return err
			}
			lanes[key] = o
		default:
			return fmt.Errorf("[verify.%s] has unknown key %q", stack, key)
		}
	}
	if err := checkCycle("verify."+stack, lanes); err != nil {
		return err
	}
	cfg.Stacks[stack] = lanes
	return nil
}
```

`parseLane(owner, kind, value)`:
- `false` ⇒ `Override{Lane: Lane{Off: true}, Replace: true}`.
- `true` ⇒ Fehler „is not a command".
- String ⇒ `commands(owner+"]."+kind, []any{v}, false)`, `Replace: true`.
- Liste ⇒ ebenso mit der Liste.
- Tabelle ⇒ je Schlüssel: `commands` und `on_file` über `commands(…)`
  (`on_file` mit `allowFile = true`), `threaded` Bool, `measuring`/`measure`
  String mit `commands(…)`-Prüfung eines Elements, `after` String aus
  `Kinds()`; `measuring`/`measure`/`after` nur bei `test` und `coverage`;
  sonst „has unknown key". Eine Tabelle ohne jeden Schlüssel ⇒ „is empty".
  `Set[key] = true` je genanntem Schlüssel.

`commands(label string, items []any, allowFile bool) ([]string, error)` prüft:
nicht leer, jedes Element ein nicht-leerer String (nach `TrimSpace`),
`shellwords.Split` gelingt, `{file}` nur mit `allowFile`. Die Beschriftung
`label` ist `[verify.go].lint` bzw. `[verify.go.lint] on_file`, damit die Texte
der Tabelle entstehen.

`checkCycle(owner string, lanes map[string]Override) error` folgt `After` von
jeder Art in `Kinds()`-Reihenfolge; trifft der Weg eine schon besuchte Art,
nennt der Fehler den Ring `test -> coverage -> test`.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/verify/ -run 'ParseConfig|ReadConfig' -cover`
Erwartet: PASS. Die Coverage von `schema.go` prüft das Tor am Ende der
Stufe; hier genügt, dass `go tool cover -func` für jede Funktion der Datei
100 % zeigt:

```sh
go test ./internal/verify/ -coverprofile=/tmp/v.out && go tool cover -func=/tmp/v.out | grep schema.go
```

- [ ] **Schritt 5: Commit**

```sh
git add internal/verify/schema.go internal/verify/schema_test.go
git commit -m "feat(verify): load [verify] strictly, one table per stack"
```

---

### Task 6: `internal/verify/presets.toml` — die eingebetteten Presets

**Files:**
- Create: `internal/verify/presets.toml`, `internal/verify/presets.go`, `internal/verify/presets_test.go`
- Modify: `internal/detect/signals.go` (eine exportierte Funktion)

**Interfaces:**
- Consumes: `parseLane`, `checkCycle`, `StackNames`, `Kinds`
- Produces:
  - `detect.SignalNames() []string` — jeder Name, der in `signals` als Stack vorkommt, sortiert
  - `type Variant struct { When string; Lanes map[string]Lane }`
  - `type PresetStack struct { TestsWhen []string; Lanes map[string]Lane; Variants []Variant }`
  - `type Presets struct { Ignored []string; Extensions map[string]string; Stacks map[string]PresetStack }`
  - `func LoadPresets() (*Presets, error)` — einmal geparst (`sync.OnceValues`)
  - `func parsePresets(data string) (*Presets, error)` — für Tests mit eigenem Text

**Format von `presets.toml`** — dasselbe Lane-Schema wie `[verify.<stack>]`,
darüber die Endungstabelle:

```toml
# Presets of `loomux check` and of post-edit. The lanes use the schema of
# [verify.<stack>] in .loomux/config.toml, so whatever a preset can say, a
# project can say too. Parsed on first use, never at start.

ignored = [".txt", ".json", ".yaml", ".yml", ".toml", ".svg", ".png", ".jpg", ".jpeg", ".import", ".lock"]

[extensions]
".go" = "go"
".py" = "python"
".gd" = "gdscript"
".cpp" = "cpp"
".hpp" = "cpp"
".cc" = "cpp"
".cxx" = "cpp"
".c" = "cpp"
".h" = "cpp"
".ts" = "typescript"
".tsx" = "typescript"
".js" = "typescript"
".jsx" = "typescript"
".vue" = "vue"
".svelte" = "svelte"
".css" = "css"
".scss" = "css"
".sass" = "css"
".less" = "css"
".html" = "html"
".htm" = "html"
".sh" = "shell"
".bash" = "shell"
".zsh" = "shell"
".sql" = "sql"
".rs" = "rust"
".md" = "wiki"

[stack.go]
tests_when = ["*_test.go"]

[stack.go.lint]
commands = ["go vet ./...", "{loomux} check gofmt ."]
threaded = true

[stack.go.test]
commands = ["go test ./... -count=1"]
measuring = "go test ./... -count=1 -covermode=set -coverprofile={coverprofile}"

[stack.go.coverage]
commands = ["{loomux} check gocover --profile {coverprofile}"]
measure = "go test ./... -count=1 -covermode=set -coverprofile={coverprofile}"
after = "test"

[stack.python]
tests_when = ["tests/", "test_*.py", "pyproject.toml:[tool.pytest"]

[stack.python.lint]
commands = ["uvx ruff check . --output-format=concise"]

# `uv run`, not `uvx`: a type checker has to see the project's dependencies.
[stack.python.types]
commands = ["uv run mypy --no-error-summary --no-pretty"]

[[stack.python.variant]]
when = "pyright"
[stack.python.variant.types]
commands = ["uv run pyright"]

[stack.python.test]
commands = ["uv run pytest -q --tb=short --no-header"]
measuring = "uv run coverage run -m pytest -q --tb=short --no-header"

# --skip-covered --skip-empty: the files at 100 % are the ones nobody reads;
# -m names the missing lines, which saves the repairer a round.
[stack.python.coverage]
commands = ["uv run coverage report --skip-covered --skip-empty -m"]
measure = "uv run coverage run -m pytest -q --tb=short --no-header"
after = "test"

[stack.gdscript.lint]
commands = ["uvx gdlint ."]
on_file = ["uvx gdlint {file}"]

[stack.gdscript.test]
commands = ["godot --headless --quit"]

[stack.cpp]
tests_when = ["CMakeLists.txt:enable_testing("]

[stack.cpp.lint]
commands = ["clang-tidy -p build"]
on_file = ["clang-format --dry-run --Werror {file}"]

# The compiler is the type checker of C++; post-edit ran the build before.
[stack.cpp.types]
commands = ["cmake --build build --parallel"]

[stack.cpp.test]
commands = ["ctest --test-dir build --output-on-failure"]

# --fail-under-line makes a gate of it: the old preset only wrote a report.
[stack.cpp.coverage]
commands = ["gcovr --root . --object-directory build --fail-under-line 100 --txt"]
after = "test"

[stack.typescript]
tests_when = ["package.json:vitest"]

[stack.typescript.lint]
commands = ["npx eslint ."]
on_file = ["npx eslint --cache {file}"]

[[stack.typescript.variant]]
when = "biome"
[stack.typescript.variant.lint]
commands = ["npx biome check ."]
on_file = ["npx biome check {file}"]

[stack.typescript.types]
commands = ["npx tsc --noEmit"]

[stack.typescript.test]
commands = ["npx vitest run"]

# One stage: vitest measures and reports in the same run.
[stack.typescript.coverage]
commands = ["npx vitest run --coverage"]

[stack.vue.types]
commands = ["npx vue-tsc --noEmit"]

[stack.svelte.types]
commands = ["npx svelte-check"]

[stack.css.lint]
commands = ["npx stylelint **/*.{css,scss}"]
on_file = ["npx stylelint {file}"]

[stack.html.lint]
commands = ["npx htmlhint **/*.html"]
on_file = ["npx htmlhint {file}"]

[stack.shell.lint]
on_file = ["shellcheck {file}"]

[stack.sql.lint]
commands = ["sqlfluff lint ."]
on_file = ["sqlfluff lint {file}"]

[stack.rust.lint]
commands = ["cargo clippy -- -D warnings", "cargo fmt --check"]
```

Kein Shell-Glob: `stylelint` und `htmlhint` expandieren `**/*.{css,scss}` bzw.
`**/*.html` selbst; ohne Shell kommt das Muster wörtlich bei ihnen an.
`shell` hat keine `commands`, weil `shellcheck` Globs nicht selbst auflöst; im
`check`-Scope ist `lint/shell` damit `not-applicable`.

**Validierung beim Laden** (`parsePresets`), jeder Verstoß ist ein Fehler mit
dem Präfix `presets.toml: `:
- jeder Stack-Name aus `StackNames()`, jede Art aus `Kinds()`;
- jede Lane über `parseLane` (Tabellenform) — dieselben Regeln wie für die
  Konfiguration;
- jedes `when` aus `detect.SignalNames()`;
- jede Endung beginnt mit `.` und zeigt auf einen Stack aus `StackNames()`;
- `checkCycle` je Stack;
- `{coverprofile}`-Regel (Task 7) je Stack.

- [ ] **Schritt 1: `detect.SignalNames` mit Test**

```go
func TestSignalNamesAreSortedAndUnique(t *testing.T) {
	names := SignalNames()
	if !sort.StringsAreSorted(names) || !slices.Contains(names, "pyright") || !slices.Contains(names, "biome") {
		t.Fatalf("%v", names)
	}
	for i := 1; i < len(names); i++ {
		if names[i] == names[i-1] {
			t.Fatalf("duplicate %q", names[i])
		}
	}
}
```

```go
// SignalNames lists every stack name a signal can report; the presets of
// verify select their variants by these names and are checked against them.
func SignalNames() []string {
	set := map[string]bool{}
	for _, sig := range signals {
		for _, s := range sig.stacks {
			set[s] = true
		}
	}
	return sorted(set)
}
```

- [ ] **Schritt 2: Preset-Tests schreiben**

```go
func TestTheEmbeddedPresetsLoad(t *testing.T) {
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	if p.Extensions[".go"] != "go" || !slices.Contains(p.Ignored, ".lock") {
		t.Fatalf("%+v", p.Extensions)
	}
	if p.Stacks["python"].Variants[0].When != "pyright" {
		t.Fatal("pyright variant")
	}
	if p.Stacks["go"].Lanes["coverage"].After != "test" {
		t.Fatal("go coverage after test")
	}
	again, _ := LoadPresets()
	if again != p {
		t.Fatal("parsed twice")
	}
}

func TestParsePresetsRefuses(t *testing.T) {
	cases := []struct{ src, want string }{
		{"[stack.cobol.lint]\ncommands=[\"x\"]", "cobol"},
		{"[stack.go.style]\ncommands=[\"x\"]", "style"},
		{"[[stack.go.variant]]\nwhen = \"nothing\"", `"nothing"`},
		{"[extensions]\ngo = \"go\"", `"go"`},
		{"[extensions]\n\".x\" = \"cobol\"", "cobol"},
		{"[stack.go.lint]\ncommands=[]", "empty"},
		{"[stack.go.coverage]\ncommands=[\"{loomux} check gocover --profile {coverprofile}\"]", "coverprofile"},
		{"x = [", "presets.toml"},
	}
	for _, c := range cases {
		if _, err := parsePresets(c.src); err == nil || !strings.Contains(err.Error(), c.want) || !strings.HasPrefix(err.Error(), "presets.toml: ") {
			t.Errorf("%q: %v", c.src, err)
		}
	}
}
```

Dazu Tests für: `ignored` ist keine Liste, `tests_when` ist keine Liste,
`variant` ist keine Liste von Tabellen, `when` fehlt, `stack` ist kein Table.

- [ ] **Schritt 3: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/verify/ ./internal/detect/ -run 'Presets|SignalNames'`
Erwartet: FAIL.

- [ ] **Schritt 4: Implementieren**

```go
package verify

import (
	_ "embed"
	"sync"
)

//go:embed presets.toml
var presetsText string

// loadPresets holds a function, not parsed data: the start rule forbids a
// package variable that parses embedded data, and OnceValues parses on the
// first call only.
var loadPresets = sync.OnceValues(func() (*Presets, error) { return parsePresets(presetsText) })

func LoadPresets() (*Presets, error) { return loadPresets() }
```

`//go:embed` in einem `string` ist keine Parse-Arbeit beim Start (der Linker
legt die Bytes ab); die Startzeit-Regel betrifft das Parsen.

`parsePresets(data)` dekodiert in `map[string]any`, liest `ignored`,
`extensions` und `stack`, und für jeden Stack in sortierter Reihenfolge
`tests_when`, die Lanes (über `parseLane("stack."+name, kind, v)`; eine
`Override` mit `Replace` ist hier kein Fehler, sie ist nur die Tabellenform)
und `variant` (eine Liste von Tabellen mit `when` und Lanes). Jeder Fehler wird
mit `fmt.Errorf("presets.toml: %w", err)` umhüllt.

- [ ] **Schritt 5: Tests laufen lassen**

Run: `go test ./internal/verify/ ./internal/detect/ -cover`
Erwartet: PASS.

- [ ] **Schritt 6: Commit**

```sh
git add internal/verify/presets.toml internal/verify/presets.go internal/verify/presets_test.go internal/detect/signals.go internal/detect/signals_test.go
git commit -m "feat(verify): embed presets per stack in the schema of [verify]"
```

---

### Task 7: Testerkennung und die wirksame Tabelle

**Files:**
- Create: `internal/verify/tests.go`, `internal/verify/effective.go`
- Test: `internal/verify/tests_test.go`, `internal/verify/effective_test.go`

**Interfaces:**
- Consumes: `Config`, `Presets`, `detect.Facts` (mit `Areas` aus Task 4)
- Produces:
  - `func HasTests(root string, patterns []string) bool`
  - `type Resolved struct { Lane Lane; Origin string; Defined bool }` — `Origin` ist `preset`, `preset, variant <when>` oder `config`; `Defined` heißt: nicht `Off` und mindestens ein Befehl in `Commands` oder `OnFile`
  - `type Effective struct { Stacks map[string]map[string]Resolved; Areas map[string][]string; Active []string; Configured map[string]bool; TestsWhen map[string][]string; Extensions map[string]string; Ignored []string; Config Config }` — `Extensions` und `Ignored` übernimmt `Resolve` aus den Presets, damit Plan und post-edit nur `Effective` brauchen
  - `func Resolve(cfg Config, p *Presets, facts detect.Facts) (Effective, error)`

**`HasTests`:** geht rekursiv mit `filepath.WalkDir` durch `root` und
überspringt `.git`, `vendor`, `node_modules`, `third_party` und `.loomux`.
Muster:
- endet auf `/` ⇒ ein Verzeichnis dieses Namens irgendwo (`tests/`);
- enthält `:` ⇒ `datei:text` — eine Datei dieses Namens irgendwo, deren Inhalt
  `text` enthält;
- sonst ein Glob auf den Basisnamen (`*_test.go`).
Der erste Treffer beendet den Gang (`filepath.SkipAll`).

**`Resolve`:**
1. Je Stack aus `StackNames()` außer `wiki`: die Preset-Lanes; dann die
   **erste** Variante, deren `When` in `facts.Stacks` steht — sie ersetzt die
   Arten, die sie nennt, ganz; `Origin` wird `preset, variant <when>`.
2. Darüber je Art die `Override` aus `cfg.Stacks[stack]`: `Replace` ⇒ die Lane
   wird die der Konfiguration (`After` bleibt aus dem Preset, wenn die
   Konfiguration keins nennt); Tabellenform ⇒ nur die Schlüssel aus `Set`
   ersetzen; `Origin` wird `config`.
3. **Aktiv** ist ein Stack, wenn er in `facts.Stacks` steht oder in
   `cfg.Stacks` mindestens eine Lane mit Befehl trägt. `project` ist nur aktiv,
   wenn konfiguriert. `Active` ist sortiert.
4. **Bereiche:** `facts.Areas[stack]`; für einen nur konfigurierten Stack und
   für `project` `["."]`.
5. **Prüfung:** Nennt `coverage` `{coverprofile}` (in `Commands`), aber keins
   von `test.Commands`, `test.Measuring`, `coverage.Measure` desselben Stacks,
   ist das ein Fehler: `[verify.<stack>].coverage reads {coverprofile}, but
   nothing in test or coverage.measure writes it`. Dieselbe Funktion
   (`checkCoverProfile(stack string, lanes map[string]Lane) error`) prüft in
   Task 6 die Presets.
6. `TestsWhen[stack]` aus dem Preset; `Configured[stack]` ist wahr, wenn
   `cfg.Stacks[stack]["test"]` gesetzt ist — dann gilt die Erkennung nicht.

- [ ] **Schritt 1: Tests schreiben**

```go
func TestHasTests(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	write("internal/cli/a_test.go", "package cli")
	write("node_modules/x/b_test.go", "")
	write("CMakeLists.txt", "project(x)\nenable_testing()\n")
	if !HasTests(root, []string{"*_test.go"}) {
		t.Error("two levels deep must count")
	}
	if !HasTests(root, []string{"CMakeLists.txt:enable_testing("}) {
		t.Error("content marker")
	}
	if HasTests(root, []string{"tests/", "test_*.py"}) {
		t.Error("no python tests here")
	}
	only := t.TempDir()
	os.MkdirAll(filepath.Join(only, "node_modules", "y"), 0o755)
	os.WriteFile(filepath.Join(only, "node_modules", "y", "c_test.go"), nil, 0o644)
	if HasTests(only, []string{"*_test.go"}) {
		t.Error("node_modules is skipped")
	}
}
```

```go
func presetsFor(t *testing.T) *Presets {
	t.Helper()
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveLayersPresetVariantAndConfig(t *testing.T) {
	cfg, _ := parse(t, `
[verify.python]
lint = "ruff check ."
[verify.go.test]
measuring = "go test -coverpkg=x/... -coverprofile={coverprofile} ./..."
[verify.sql]
lint = "sqlfluff lint ."
`)
	facts := detect.Facts{
		Stacks: []string{"go", "pyright", "python"},
		Areas:  map[string][]string{"go": {"."}, "python": {"."}, "pyright": {"."}},
	}
	eff, err := Resolve(cfg, presetsFor(t), facts)
	if err != nil {
		t.Fatal(err)
	}
	if got := eff.Stacks["python"]["types"]; got.Origin != "preset, variant pyright" || got.Lane.Commands[0] != "uv run pyright" {
		t.Fatalf("%+v", got)
	}
	if got := eff.Stacks["python"]["lint"]; got.Origin != "config" || got.Lane.Commands[0] != "ruff check ." {
		t.Fatalf("%+v", got)
	}
	gotest := eff.Stacks["go"]["test"]
	if gotest.Lane.Commands[0] != "go test ./... -count=1" || !strings.Contains(gotest.Lane.Measuring, "coverpkg") {
		t.Fatalf("table merges: %+v", gotest)
	}
	if !slices.Equal(eff.Active, []string{"go", "python", "sql"}) || !slices.Equal(eff.Areas["sql"], []string{"."}) {
		t.Fatalf("%v %v", eff.Active, eff.Areas)
	}
	if eff.Stacks["go"]["types"].Defined {
		t.Fatal("go has no types")
	}
}

func TestResolveRefusesACoverProfileNobodyWrites(t *testing.T) {
	cfg, _ := parse(t, "[verify.go]\ntest = \"go test ./...\"\n[verify.go.coverage]\nmeasure = \"go test ./...\"\n")
	_, err := Resolve(cfg, presetsFor(t), detect.Facts{Stacks: []string{"go"}})
	if err == nil || !strings.Contains(err.Error(), "nothing in test or coverage.measure writes it") {
		t.Fatalf("%v", err)
	}
}
```

Dazu: Konfiguration schaltet eine Lane mit `false` ab (`Defined == false`,
`Origin == "config"`); ein String ersetzt, `After` bleibt aus dem Preset;
`project` ohne Konfiguration nicht aktiv; `Configured["go"]` bei
`[verify.go] test = …`.

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/verify/ -run 'HasTests|Resolve'`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren** — nach der Beschreibung oben. Die
Überlagerung der Tabellenform:

```go
func merge(base Lane, o Override) Lane {
	if o.Replace {
		if o.Lane.After == "" {
			o.Lane.After = base.After
		}
		return o.Lane
	}
	if o.Set["commands"] {
		base.Commands = o.Lane.Commands
	}
	if o.Set["on_file"] {
		base.OnFile = o.Lane.OnFile
	}
	if o.Set["threaded"] {
		base.Threaded = o.Lane.Threaded
	}
	if o.Set["measuring"] {
		base.Measuring = o.Lane.Measuring
	}
	if o.Set["measure"] {
		base.Measure = o.Lane.Measure
	}
	if o.Set["after"] {
		base.After = o.Lane.After
	}
	return base
}
```

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/verify/ -cover`
Erwartet: PASS.

- [ ] **Schritt 5: Commit**

```sh
git add internal/verify/tests.go internal/verify/tests_test.go internal/verify/effective.go internal/verify/effective_test.go internal/verify/presets.go
git commit -m "feat(verify): layer presets, variants and config, and find tests"
```

---

### Task 8: `internal/verify/plan.go` — Lanes und ihr Graph

**Files:**
- Create: `internal/verify/plan.go`, `internal/verify/cover.go`
- Test: `internal/verify/plan_test.go`, `internal/verify/cover_test.go`

**Interfaces:**
- Consumes: `Effective`, `HasTests`, `shellwords.Split`
- Produces:
  - `type Scope int`; `const ( ScopeCheck Scope = iota; ScopeEdit )`
  - `type Request struct { Kinds []string; Scope Scope; File string }` — `File` repowurzel-relativ mit Schrägstrichen, nur im Edit-Scope
  - `type State string` mit den Konstanten `StateOK = "ok"`, `StateFailed = "failed"`, `StateTimedOut = "timed-out"`, `StateBudget = "budget"`, `StateBlocked = "blocked"`, `StateMissingTool = "missing-tool"`, `StateUnready = "unready"`, `StateUnavailable = "unavailable"`, `StateNotApplicable = "not-applicable"`
  - `type Job struct { Name, Kind, Stack, Area, Origin string; Dir string; Argvs [][]string; Threaded bool; Measure []string; After int; Env []string; Pre State; Note string; Fn func() (string, error); Reads []string }` — `Reads` sind die aufgelösten Pfade von `{coverprofile}` und `{coverdata}` aus `Commands`, die vorher existieren müssen; `After` ist der Index des Vorgängers oder `-1`; `Pre` ist ein Zustand, den schon der Plan entscheidet
  - `type PlanEnv struct { Root, Loomux, RunID string; HasTests func(root string, patterns []string) bool; ImportReady func(dir string) bool }`
  - `func ExpandProfile(cfg Config, request string) ([]string, error)`
  - `func Plan(eff Effective, req Request, env PlanEnv) ([]Job, error)`
  - `func NewRunID(now time.Time, pid int) string` → `20260919T101500-1234`
  - `func CoverPaths(root, runID, stack, area string) (profile, data string)` → `<root>/.loomux/state/cover/<runID>-<stack>-<area>.out` bzw. `.data`; `area` `"."` wird `root`
  - `func CleanCover(root, runID string, green bool) error`

**`ExpandProfile`:** `all` ⇒ die vier Arten. Ein Name aus `cfg.Profiles` ⇒
dessen Arten. Sonst eine Komma-Liste von Arten (Leerraum um die Teile wird
entfernt); ein unbekanntes Teil ⇒ Fehler `unknown check "x"; kinds: lint,
types, test, coverage; profiles: edit, precommit` (Profile sortiert); eine
leere Liste ⇒ `"" names no check`. Doppelte Arten fallen weg, die Reihenfolge
der ersten Nennung bleibt.

**`Plan`** (die Regeln der Spec, Abschnitte Lauf und Zustände):
1. Jobs in fester Reihenfolge: Arten wie angefragt, darin `eff.Active`
   (sortiert), darin die Bereiche. Im Edit-Scope nur der Stack der Datei
   (`eff.Extensions` nach der kleingeschriebenen Endung) und darin nur der
   **eine** Bereich, der die Datei enthält (längstes Präfix; `"."` sonst). Ist
   der Stack der Datei nicht aktiv oder die Endung unbekannt: keine Jobs.
2. `Name` ist `<art>/<stack>`, bei mehr als einem Bereich des Stacks
   `<art>/<stack>@<bereich>`. `Dir` ist `filepath.Join(root, area)`.
3. **Befehle:** Edit-Scope ⇒ `OnFile`, wenn gesetzt, sonst `Commands`; `project`
   im Edit-Scope nur mit `OnFile`, sonst kein Job. Check-Scope ⇒ `Commands`.
4. **Nicht definiert** (`!Defined` oder im Scope keine Befehle): Check-Scope ⇒
   Job mit `Pre = StateNotApplicable`, Note `no command`; Edit-Scope ⇒ kein Job.
5. **Keine Tests:** Art `test` oder `coverage`, `TestsWhen[stack]` nicht leer,
   `!Configured[stack]`, und `env.HasTests(dir, patterns)` falsch ⇒
   `Pre = StateUnavailable`, Note `no tests found`.
6. **Godot:** `gdscript`, Art `test` oder `coverage`, `Config.ImportCheck`, und
   `env.ImportReady(dir)` falsch ⇒ `Pre = StateUnready`, Note
   `run the Godot editor once to import the project`.
7. **Messen:** Art `test` mit `Measuring` und `coverage` in der Anfrage ⇒
   `Argvs = [Measuring]`.
8. **Vorgänger:** Nach dem Bau aller Jobs: `After` jeder Lane mit `Lane.After`
   zeigt auf den Job derselben Art `Lane.After`, desselben Stacks und
   Bereichs, **wenn** er in der Anfrage steht. Steht er nicht darin: `Measure`
   gesetzt ⇒ `Job.Measure = split(Measure)`; sonst, wenn die Lane `{coverprofile}`
   oder `{coverdata}` liest, `Pre = StateFailed`, Note
   `` `test` did not run and there is no measure step ``.
9. **Platzhalter** je Argument nach dem Zerlegen: `{file}` ⇒ die Datei relativ
   zu `Dir` mit Schrägstrichen; `{area}` ⇒ `Dir`; `{coverprofile}`,
   `{coverdata}` ⇒ `CoverPaths(root, RunID, stack, area)`; `{loomux}` ⇒
   `env.Loomux`. Andere Klammern bleiben stehen.
10. **Umgebung:** Stack `python` bekommt `COVERAGE_FILE=<coverdata>`, damit zwei
    Läufe sich kein `.coverage` teilen.

11. **`Reads`:** Jeder Pfad, den `{coverprofile}` oder `{coverdata}` in `Commands` einer Lane ergibt, kommt nach `Job.Reads` — aber nur, wenn die Lane einen Vorgänger (`After`) oder ein `Measure` hat; ein Befehl, der die Datei selbst schreibt, liest sie nicht vorher.

**`CleanCover`:** löscht in `<root>/.loomux/state/cover/` jede Datei, deren
Name nicht mit `runID+"-"` beginnt; die eigenen nur, wenn `green`. Ein
fehlendes Verzeichnis ist kein Fehler.

- [ ] **Schritt 1: Tests schreiben**

```go
func env(root string) PlanEnv {
	return PlanEnv{
		Root: root, Loomux: `C:\bin\loomux.exe`, RunID: "R",
		HasTests:    func(string, []string) bool { return true },
		ImportReady: func(string) bool { return true },
	}
}

func effFor(t *testing.T, src string, facts detect.Facts) Effective {
	t.Helper()
	cfg, err := parse(t, src)
	if err != nil {
		t.Fatal(err)
	}
	eff, err := Resolve(cfg, presetsFor(t), facts)
	if err != nil {
		t.Fatal(err)
	}
	return eff
}

var goOnly = detect.Facts{Stacks: []string{"go"}, Areas: map[string][]string{"go": {"."}}}

func TestPlanLinksCoverageToTestAndMeasuresInTest(t *testing.T) {
	root := t.TempDir()
	jobs, err := Plan(effFor(t, "", goOnly), Request{Kinds: []string{"lint", "types", "test", "coverage"}}, env(root))
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, j := range jobs {
		names = append(names, j.Name)
	}
	if strings.Join(names, " ") != "lint/go types/go test/go coverage/go" {
		t.Fatalf("%v", names)
	}
	if jobs[1].Pre != StateNotApplicable {
		t.Fatalf("types/go: %+v", jobs[1])
	}
	profile, _ := CoverPaths(root, "R", "go", ".")
	if !slices.Contains(jobs[2].Argvs[0], "-coverprofile="+profile) {
		t.Fatalf("test measures: %v", jobs[2].Argvs)
	}
	if jobs[3].After != 2 || jobs[3].Measure != nil {
		t.Fatalf("coverage: %+v", jobs[3])
	}
	if jobs[3].Argvs[0][0] != `C:\bin\loomux.exe` {
		t.Fatalf("{loomux}: %v", jobs[3].Argvs[0])
	}
}

func TestPlanLetsCoverageMeasureAlone(t *testing.T) {
	jobs, _ := Plan(effFor(t, "", goOnly), Request{Kinds: []string{"coverage"}}, env(t.TempDir()))
	if jobs[0].After != -1 || jobs[0].Measure == nil {
		t.Fatalf("%+v", jobs[0])
	}
}

func TestPlanFailsCoverageThatCannotMeasure(t *testing.T) {
	src := "[verify.go]\ncoverage = \"{loomux} check gocover --profile {coverprofile}\"\n"
	jobs, _ := Plan(effFor(t, src, goOnly), Request{Kinds: []string{"coverage"}}, env(t.TempDir()))
	if jobs[0].Pre != StateFailed || !strings.Contains(jobs[0].Note, "no measure step") {
		t.Fatalf("%+v", jobs[0])
	}
}

func TestPlanMarksMissingTestsUnavailable(t *testing.T) {
	e := env(t.TempDir())
	e.HasTests = func(string, []string) bool { return false }
	jobs, _ := Plan(effFor(t, "", goOnly), Request{Kinds: []string{"test", "coverage"}}, e)
	if jobs[0].Pre != StateUnavailable || jobs[1].Pre != StateUnavailable {
		t.Fatalf("%+v", jobs)
	}
}

func TestPlanEditScopeRunsOnFileInTheFilesArea(t *testing.T) {
	facts := detect.Facts{Stacks: []string{"typescript"}, Areas: map[string][]string{"typescript": {"admin", "web"}}}
	root := t.TempDir()
	jobs, _ := Plan(effFor(t, "", facts), Request{Kinds: []string{"lint", "types"}, Scope: ScopeEdit, File: "web/src/a.ts"}, env(root))
	if len(jobs) != 2 || jobs[0].Name != "lint/typescript@web" || jobs[0].Dir != filepath.Join(root, "web") {
		t.Fatalf("%+v", jobs)
	}
	if strings.Join(jobs[0].Argvs[0], " ") != "npx eslint --cache src/a.ts" {
		t.Fatalf("%v", jobs[0].Argvs)
	}
}
```

Dazu: `ExpandProfile` (all, Profil, Komma-Liste mit Leerraum, Dublette,
unbekannt, leer); Edit-Scope für eine Datei eines nicht aktiven Stacks (keine
Jobs); `project` im Edit-Scope ohne `on_file` (kein Job); Godot `unready`;
`COVERAGE_FILE` bei `python`; ein unbekannter Platzhalter `{x}` bleibt stehen;
`NewRunID`; `CoverPaths` für `"."` und `"web"`; `CleanCover` (fremde Dateien
weg, eigene nur bei grün, fehlendes Verzeichnis). Ein Befehl, den
`shellwords.Split` nicht zerlegt, kann hier nicht mehr vorkommen (Task 5 prüft
beim Laden); `Plan` gibt den Fehler trotzdem zurück, getestet mit einer von
Hand gebauten `Effective`.

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/verify/ -run 'Plan|Expand|Cover|RunID'`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren** — nach den Regeln oben.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/verify/ -cover`
Erwartet: PASS.

- [ ] **Schritt 5: Commit**

```sh
git add internal/verify
git commit -m "feat(verify): plan lanes per stack, area and kind"
```

---

### Task 9: `internal/verify/run.go` — den Graphen fahren

**Files:**
- Create: `internal/verify/run.go`
- Test: `internal/verify/run_test.go`

**Interfaces:**
- Consumes: `Job`, `State`, `child.Spec`, `child.Result`
- Produces:
  - `type Outcome struct { Job Job; State State; Output string; Duration time.Duration; BlockedBy string }`
  - `type RunOptions struct { Scope Scope; MaxParallel int; Timeout, Budget time.Duration; Start func(child.Spec) child.Result; Look func(string) (string, error); Now func() time.Time }`
  - `func Run(jobs []Job, opt RunOptions) []Outcome` — `Outcome[i]` gehört zu `jobs[i]`
  - `func Red(s State, scope Scope) bool`

**Regeln:**
- Jeder Job läuft in einer eigenen Goroutine und wartet auf den Kanal seines
  Vorgängers. Ein Semaphor der Größe `MaxParallel` umschließt **jeden**
  `Start`-Aufruf, nicht die Goroutine.
- `Red`: `failed`, `timed-out`, `blocked` immer; `missing-tool` und `unready`
  nur im Check-Scope.
- **Vorgänger:** rot ⇒ `blocked`, `BlockedBy` = sein Name. Sein Zustand ist
  `unavailable` oder `not-applicable` ⇒ derselbe Zustand (Erbregel). Im
  Edit-Scope zusätzlich `budget`, `missing-tool`, `unready` ⇒ derselbe Zustand.
- `Pre` gesetzt ⇒ dieser Zustand, `Output` = `Note`.
- `Fn` gesetzt ⇒ `out, err := Fn()`; `err` ⇒ `failed`.
- Sonst: `Measure` zuerst, dann `Argvs` — mit `Threaded` parallel, sonst der
  Reihe nach, **alle**, auch nach einem roten. Ein roter `Measure` beendet die
  Lane als `failed`.
- Vor jedem Befehl: `Look(argv[0])` scheitert ⇒ die Lane ist `missing-tool`,
  Output `"<tool>" is not on PATH: <befehl>`; kein Befehl der Lane startet.
- **Budget:** Hat `opt.Budget > 0`, gilt eine Gesamtfrist ab Aufruf von `Run`.
  Jeder Start bekommt `min(opt.Timeout, Rest)`; ist der Rest ≤ 0, startet nichts
  mehr, die Lane ist `budget`. Endet ein Befehl mit `TimedOut`, und seine Frist
  war der Rest (nicht `opt.Timeout`), ist die Lane `budget`, sonst `timed-out`.
- Ergebnis eines Befehls: `Err` ⇒ rot mit `Err.Error()`; `TimedOut` wie oben;
  `OutputAbandoned` ⇒ `failed` mit dem Hinweis
  `output abandoned: a process the tool started kept its pipe open`;
  `Code != 0` ⇒ `failed`.
- **Vorbedingung `Reads`:** Vor dem ersten Befehl der Lane (nach einem `Measure`) prüft `os.Stat` jeden Pfad aus `Job.Reads`; fehlt einer, ist die Lane `failed` mit `<pfad> is missing: the measuring run did not write it`, und kein Befehl startet. Test: `Reads` auf eine fehlende Datei, erwartet `failed` und kein Start.
- **Ausgabe:** stdout, dann stderr. Ein Befehl: so, wie er kam. Mehrere:
  je Block `$ <argv mit Leerzeichen>` (plus ` (failed)` bei rot), dann die
  Ausgabe ohne Zeilenende am Schluss, Blöcke durch eine Leerzeile getrennt, ein
  `\n` am Ende — wie `checks.py:574-616`.
- `Duration` misst `Now()` vor und nach der Lane.

- [ ] **Schritt 1: Tests schreiben**

```go
type fakeStart struct {
	mu      sync.Mutex
	started []string
	answer  func(child.Spec) child.Result
	running, peak int
}

func (f *fakeStart) start(s child.Spec) child.Result {
	f.mu.Lock()
	f.started = append(f.started, strings.Join(s.Argv, " "))
	f.running++
	if f.running > f.peak {
		f.peak = f.running
	}
	f.mu.Unlock()
	r := f.answer(s)
	f.mu.Lock()
	f.running--
	f.mu.Unlock()
	return r
}

func ok(child.Spec) child.Result { return child.Result{Code: 0} }

func opts(f *fakeStart) RunOptions {
	return RunOptions{Scope: ScopeCheck, MaxParallel: 4, Timeout: time.Minute, Start: f.start,
		Look: func(s string) (string, error) { return s, nil }, Now: time.Now}
}

func job(name string, after int, argvs ...string) Job {
	j := Job{Name: name, After: after}
	for _, a := range argvs {
		j.Argvs = append(j.Argvs, strings.Fields(a))
	}
	return j
}

func TestRunBlocksTheSuccessorOfARedLane(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		if s.Argv[0] == "t" {
			return child.Result{Code: 1, Stdout: "boom\n"}
		}
		return child.Result{}
	}}
	out := Run([]Job{job("test/go", -1, "t"), job("coverage/go", 0, "c")}, opts(f))
	if out[0].State != StateFailed || out[1].State != StateBlocked || out[1].BlockedBy != "test/go" {
		t.Fatalf("%+v", out)
	}
	if len(f.started) != 1 {
		t.Fatalf("coverage must not start: %v", f.started)
	}
}

func TestRunInheritsAPredecessorThatCouldNotRun(t *testing.T) {
	pre := job("test/go", -1)
	pre.Pre = StateUnavailable
	out := Run([]Job{pre, job("coverage/go", 0, "c")}, opts(&fakeStart{answer: ok}))
	if out[1].State != StateUnavailable {
		t.Fatalf("%+v", out[1])
	}
}

func TestRunNamesAMissingTool(t *testing.T) {
	o := opts(&fakeStart{answer: ok})
	o.Look = func(string) (string, error) { return "", errors.New("not found") }
	out := Run([]Job{job("lint/python", -1, "ruff check .")}, o)
	if out[0].State != StateMissingTool || !strings.Contains(out[0].Output, `"ruff" is not on PATH`) {
		t.Fatalf("%+v", out[0])
	}
}

func TestRunCapsProcessesAtMaxParallel(t *testing.T) {
	f := &fakeStart{answer: func(child.Spec) child.Result { time.Sleep(20 * time.Millisecond); return child.Result{} }}
	o := opts(f)
	o.MaxParallel = 2
	jobs := []Job{}
	for i := 0; i < 6; i++ {
		jobs = append(jobs, job(fmt.Sprintf("lint/s%d", i), -1, "x"))
	}
	Run(jobs, o)
	if f.peak > 2 {
		t.Fatalf("peak %d", f.peak)
	}
}

func TestRunRunsEveryCommandAndBlocksTheOutput(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		if s.Argv[0] == "b" {
			return child.Result{Code: 1, Stdout: "bad\n"}
		}
		return child.Result{Stdout: "fine\n"}
	}}
	j := job("lint/go", -1, "a 1", "b 2", "c 3")
	out := Run([]Job{j}, opts(f))
	want := "$ a 1\nfine\n\n$ b 2 (failed)\nbad\n\n$ c 3\nfine\n"
	if out[0].State != StateFailed || out[0].Output != want || len(f.started) != 3 {
		t.Fatalf("%q %v", out[0].Output, f.started)
	}
}

func TestRunTurnsABudgetDeadlineIntoBudget(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		if s.Timeout >= time.Minute {
			t.Errorf("the budget must cap the timeout, got %v", s.Timeout)
		}
		return child.Result{Code: -1, TimedOut: true}
	}}
	o := opts(f)
	o.Scope, o.Budget = ScopeEdit, 50*time.Millisecond
	out := Run([]Job{job("lint/go", -1, "go vet ./...")}, o)
	if out[0].State != StateBudget {
		t.Fatalf("%+v", out[0])
	}
}
```

Dazu: `Pre`, `Fn` grün und rot, `Measure` rot beendet die Lane, `Threaded`
startet alle Befehle, `TimedOut` mit eigener Frist ⇒ `timed-out`, `Err`,
`OutputAbandoned`, Budget schon aufgebraucht ⇒ kein Start, Erbregel im
Edit-Scope für `missing-tool`, `Red` für jeden Zustand in beiden Scopes.

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/verify/ -run Run`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren** — nach den Regeln oben.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test -race ./internal/verify/ -cover`
Erwartet: PASS, ohne Datenrennen.

- [ ] **Schritt 5: Commit**

```sh
git add internal/verify/run.go internal/verify/run_test.go
git commit -m "feat(verify): run the lane graph under one process cap and budget"
```

---

### Task 10: Bericht, Urteil und `--show`

**Files:**
- Create: `internal/verify/report.go`, `internal/verify/show.go`
- Test: `internal/verify/report_test.go`, `internal/verify/show_test.go`

**Interfaces:**
- Consumes: `Outcome`, `State`, `Red`, `Effective`, `PlanEnv`, `HasTests`
- Produces:
  - `func WriteCheck(w io.Writer, outs []Outcome, verbose bool)`
  - `func CheckVerdict(kinds []string, outs []Outcome) (code int, notes []string)`
  - `func WriteEdit(stdout, stderr io.Writer, outs []Outcome) int` — gibt `0` oder `2` zurück
  - `func WriteShow(w io.Writer, eff Effective, kinds []string, env PlanEnv)`

**`WriteCheck`** — eine Kopfzeile je Outcome, in Job-Reihenfolge:

| Zustand | Kopfzeile |
|---|---|
| ok, failed, timed-out, budget | `<name>: <state> [<origin>] <sekunden mit einer Nachkommastelle>s` |
| blocked | `<name>: blocked [<origin>] by <BlockedBy>` |
| alle übrigen | `<name>: <state> [<origin>] <Output>` (Output ist die Note, eine Zeile) |

Nach der Kopfzeile folgt die Ausgabe (ohne abschließendes `\n`, dann ein `\n`),
wenn `Red(state, ScopeCheck)` gilt oder `verbose` gesetzt ist und die Lane lief.
`<origin>` ist `Job.Origin`; eine Lane mit `Fn` hat `in-process`.

**`CheckVerdict`** — je angeforderter Art:
- `ran`: eine Lane dieser Art hat einen Zustand außer `unavailable` und
  `not-applicable` — auch `blocked`, `missing-tool` und `unready` zählen, denn
  dort gab es etwas zu prüfen, und die Lane ist ohnehin rot; die Note „nothing
  to check" wäre dort falsch;
- `na`: eine Lane dieser Art ist `not-applicable`;
- nicht `ran` und nicht `na` ⇒ Note `nothing to check for ` + "`<art>`", Code 1.
Dazu Code 1, wenn irgendeine Lane `Red(…, ScopeCheck)` ist. Sonst 0. Die Notes
schreibt der Aufrufer nach dem Bericht auf stdout.

**`WriteEdit`**:
- rote Lanes (`Red(…, ScopeEdit)`): je Lane `<name>: <state>` und ihre
  Ausgabe auf stderr; Rückgabe 2.
- übersprungene (`budget`, `missing-tool`, `unready`): je eine Zeile
  - `missing-tool`: `loomux hook post-tool-use: lane skipped, <Output>` —
    `Output` ist schon `"<tool>" is not on PATH: <befehl>`, wie der alte Text;
  - `budget`: `loomux hook post-tool-use: lane skipped, the edit budget ran out: <name>`;
  - `unready`: `loomux hook post-tool-use: lane skipped, <Output>`.
  Die Zeilen gehen gesammelt als eine JSON-Zeile auf stdout, exakt wie
  `reportSkipped` heute:
  `{"hookSpecificOutput":{"additionalContext":"<zeilen ohne letztes \n>","hookEventName":"PostToolUse"}}`.
  `reportSkipped` zieht aus `post_edit.go` hierher (`writeSkipped`), samt Test.
- Rückgabe 0, wenn keine Lane rot ist.

**`WriteShow`** — je aktivem Stack, sortiert:

```
# go: areas ., tests found
[verify.go.lint]
commands = ["go vet ./...", "{loomux} check gofmt ."]  # preset
threaded = true  # preset

[verify.go.types]  # not defined
```

Ein Stack mit `TestsWhen` bekommt `tests found` / `no tests found`, einer ohne
`tests not detected`. Jede Art aus `kinds`; eine nicht definierte als
Kommentarzeile. Werte in TOML-Syntax (`strconv.Quote` für Strings). Budget und
`max_parallel`/`timeout` als erste Zeilen: `# max_parallel = 8, timeout = 600s`.

- [ ] **Schritt 1: Tests schreiben**

```go
func out(name string, s State, origin string) Outcome {
	return Outcome{Job: Job{Name: name, Kind: strings.Split(name, "/")[0], Origin: origin}, State: s, Duration: 1200 * time.Millisecond}
}

func TestWriteCheckFormatsEveryState(t *testing.T) {
	outs := []Outcome{
		out("lint/go", StateOK, "config"),
		{Job: Job{Name: "lint/python", Kind: "lint", Origin: "preset"}, State: StateFailed, Output: "E1 bad\n", Duration: 800 * time.Millisecond},
		{Job: Job{Name: "types/go", Kind: "types", Origin: "preset"}, State: StateNotApplicable, Output: "no command"},
		{Job: Job{Name: "coverage/go", Kind: "coverage", Origin: "preset"}, State: StateBlocked, BlockedBy: "test/go"},
	}
	var b strings.Builder
	WriteCheck(&b, outs, false)
	want := "lint/go: ok [config] 1.2s\n" +
		"lint/python: failed [preset] 0.8s\nE1 bad\n" +
		"types/go: not-applicable [preset] no command\n" +
		"coverage/go: blocked [preset] by test/go\n"
	if b.String() != want {
		t.Fatalf("%q", b.String())
	}
}

func TestCheckVerdictPerKind(t *testing.T) {
	cases := []struct {
		name  string
		kinds []string
		outs  []Outcome
		code  int
		note  string
	}{
		{"green", []string{"lint", "types"}, []Outcome{out("lint/go", StateOK, "p"), out("types/go", StateNotApplicable, "p")}, 0, ""},
		{"a kind with nothing run", []string{"test"}, []Outcome{out("test/go", StateUnavailable, "p")}, 1, "nothing to check for `test`"},
		{"red", []string{"lint"}, []Outcome{out("lint/go", StateMissingTool, "p")}, 1, ""},
		{"unready is red, not nothing", []string{"test"}, []Outcome{out("test/gdscript", StateUnready, "p")}, 1, ""},
		{"unavailable beside a run lane", []string{"test"}, []Outcome{out("test/go", StateUnavailable, "p"), out("test/python", StateOK, "p")}, 0, ""},
	}
	for _, c := range cases {
		code, notes := CheckVerdict(c.kinds, c.outs)
		if code != c.code || (c.note == "" && len(notes) != 0) || (c.note != "" && (len(notes) != 1 || notes[0] != c.note)) {
			t.Errorf("%s: %d %v", c.name, code, notes)
		}
	}
}

func TestWriteEditSkipsQuietlyAndBlocksOnRed(t *testing.T) {
	var so, se strings.Builder
	skip := Outcome{Job: Job{Name: "lint/python"}, State: StateMissingTool, Output: `"ruff" is not on PATH: ruff check .`}
	if code := WriteEdit(&so, &se, []Outcome{skip}); code != 0 || se.Len() != 0 {
		t.Fatalf("%d %q", code, se.String())
	}
	if !strings.Contains(so.String(), `lane skipped, \"ruff\" is not on PATH`) || !strings.Contains(so.String(), `"hookEventName":"PostToolUse"`) {
		t.Fatalf("%q", so.String())
	}
	so.Reset()
	red := Outcome{Job: Job{Name: "lint/go"}, State: StateFailed, Output: "vet: x\n"}
	if code := WriteEdit(&so, &se, []Outcome{red}); code != 2 || !strings.Contains(se.String(), "vet: x") {
		t.Fatalf("%d %q", code, se.String())
	}
}
```

Dazu Tests für `verbose`, `budget` und `unready` in `WriteEdit`, eine
`Fn`-Lane (`in-process`), und `WriteShow` gegen eine feste Erwartung für
`goOnly` mit `HasTests` wahr und falsch.

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/verify/ -run 'Write|Verdict'`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren** — nach den Regeln oben.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/verify/ -cover`
Erwartet: PASS.

- [ ] **Schritt 5: Commit**

```sh
git add internal/verify
git commit -m "feat(verify): report lanes, judge each requested kind, show the table"
```

---

### Task 11: `loomux check gocover` — `dev covergate` wird Produkt

**Files:**
- Create: `internal/verify/gocover/gocover.go`, `internal/verify/gocover/gocover_test.go`
- Delete: `internal/dev/covergate/` (Inhalt zieht um)
- Modify: `internal/cli/check.go`, `internal/cli/dev.go`, `internal/cli/check_test.go`, `internal/cli/dev_test.go`, `ci/gate.sh`

**Interfaces:**
- Produces:
  - `gocover.Line`, `gocover.Parse`, `gocover.Gate` — unverändert aus `covergate`
  - `func ModulePath(gomod []byte) (string, error)` — die `module`-Zeile; fehlt sie ⇒ Fehler `go.mod has no module line`
  - `func Total(out []byte) (float64, error)` — die Zeile `total:` aus `go tool cover -func`; fehlt sie ⇒ Fehler `no total line`
  - in `internal/cli/check.go`: `var coverFunc = runCoverFunc` (zieht aus `dev.go` um), `func checkGocover(args []string, stdout, stderr io.Writer) int`

**Verhalten** `loomux check gocover --profile <p> [--floor N] [--dir D]`:
- `--profile` ist Pflicht (kein stilles `coverage.out` als Vorgabe mehr —
  Aufrufer nennen ihr Profil; `ci/gate.sh` und das Go-Preset tun es).
- `--dir` (Vorgabe: Arbeitsverzeichnis) ist das Verzeichnis mit `go.mod`; ein
  relatives `--profile` gilt relativ dazu, die Quelldateien für die
  Ausschlussprüfung ebenso, und `go tool cover` läuft dort. Das Flag gibt es,
  damit die Fallsuite (Task 14) `gocover` im Prozess rufen kann, ohne das
  Arbeitsverzeichnis des ganzen Prozesses zu wechseln, während andere Lanes
  parallel laufen. Das Preset braucht es nicht: Seine Lane läuft ohnehin im
  Bereich.
- `go.mod` in `--dir` lesen ⇒ Modulpfad; fehlt die Datei ⇒ Exit 1,
  `loomux check gocover: <fehler>`.
- `coverFunc` bekommt dafür die Signatur `func(dir, profile string) ([]byte, error)`
  und setzt `cmd.Dir = dir`; `Gate` liest die Quellen über
  `func(p string) ([]byte, error) { return os.ReadFile(filepath.Join(dir, p)) }`.
- `coverFunc(profile)`; Fehler ⇒ Exit 1.
- Ohne `--floor`: `Parse`, leer ⇒ Exit 1 `no functions in <p>`, sonst
  `Gate(lines, module, os.ReadFile, stdout)`.
- Mit `--floor N`: `Total`; unter `N` ⇒ Exit 1,
  `coverage <t>% is below the floor of <N>%`; sonst `coverage <t>%` auf stdout,
  Exit 0.

- [ ] **Schritt 1: Paket umziehen**

```sh
git mv internal/dev/covergate internal/verify/gocover
```

`git mv` stagt die Löschung des alten Pfads schon; der Commit-Schritt nennt ihn
deshalb nicht mehr (ein `git add` auf einen verschwundenen Pfad bricht mit
„pathspec did not match").

In beiden Dateien `package covergate` → `package gocover`, Imports in
`internal/cli/dev.go` nachziehen. Tests laufen lassen:
`go test ./internal/verify/gocover/` — PASS, unverändert.

- [ ] **Schritt 2: Tests für `ModulePath` und `Total` schreiben**

```go
func TestModulePath(t *testing.T) {
	got, err := ModulePath([]byte("// x\nmodule github.com/a/b\n\ngo 1.25.0\n"))
	if err != nil || got != "github.com/a/b" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := ModulePath([]byte("go 1.25.0\n")); err == nil {
		t.Fatal("want error")
	}
}

func TestTotal(t *testing.T) {
	out := []byte("github.com/a/b/x.go:3:\tF\t100.0%\ntotal:\t(statements)\t97.5%\n")
	if got, err := Total(out); err != nil || got != 97.5 {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := Total([]byte("x.go:1:\tF\t1%\n")); err == nil {
		t.Fatal("want error")
	}
}
```

Dazu ein Test für eine `total:`-Zeile mit unlesbarer Zahl.

- [ ] **Schritt 3: CLI-Tests schreiben** (`internal/cli/check_test.go`)

```go
func stubCoverFunc(t *testing.T, out string, err error) {
	t.Helper()
	old := coverFunc
	coverFunc = func(string, string) ([]byte, error) { return []byte(out), err }
	t.Cleanup(func() { coverFunc = old })
}

func TestCheckGocoverPerFunctionAndFloor(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/m\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package m\n\nfunc F() {}\n"), 0o644)
	t.Chdir(dir)
	stubCoverFunc(t, "example.com/m/a.go:3:\tF\t50.0%\ntotal:\t(statements)\t50.0%\n", nil)
	var so, se bytes.Buffer
	if code := Run([]string{"check", "gocover", "--profile", "p.out"}, nil, &so, &se); code != 1 || !strings.Contains(so.String(), "not covered: a.go:3 F 50.0%") {
		t.Fatalf("%d %q %q", code, so.String(), se.String())
	}
	so.Reset()
	if code := Run([]string{"check", "gocover", "--profile", "p.out", "--floor", "40"}, nil, &so, &se); code != 0 {
		t.Fatalf("%d %q", code, se.String())
	}
	if code := Run([]string{"check", "gocover", "--profile", "p.out", "--floor", "60"}, nil, &so, &se); code != 1 {
		t.Fatalf("%d", code)
	}
}
```

Dazu: ohne `--profile` (Exit 2), unbekanntes Flag (Exit 2), kein `go.mod`
(Exit 1), `coverFunc` scheitert (Exit 1), leeres Profil (Exit 1, `no functions`),
`--floor` ohne `total:` (Exit 1). Die bisherigen Tests von `dev covergate` in
`dev_test.go` werden zu diesen Tests umgeschrieben und dort gelöscht.

- [ ] **Schritt 4: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/cli/ ./internal/verify/gocover/ -run 'Gocover|ModulePath|Total'`
Erwartet: FAIL.

- [ ] **Schritt 5: Implementieren**

`checkCommand` verteilt jetzt `case "gocover": return checkGocover(args[1:], stdout, stderr)`.
`runCoverFunc` und `coverFunc` ziehen von `dev.go` nach `check.go`; `const
module` und `devCovergate` samt Eintrag `"covergate"` in `devCommands` entfallen.

- [ ] **Schritt 6: `ci/gate.sh` umstellen**

Die letzte Zeile wird:

```sh
go run ./cmd/loomux check gocover --profile coverage.out
```

- [ ] **Schritt 7: Tor fahren**

Run: `sh ci/gate.sh`
Erwartet: grün; die Ausgabe von `gocover` ist leer (alles gedeckt).

- [ ] **Schritt 8: Commit**

Ein Commit: Umzug und Wegfall lassen sich nicht trennen, ohne dass einer der
beiden Stände nicht baut. Der Wegfall steht im Rumpf und später im Changelog
unter `Removed`; kein `!`, denn `dev` ist keine Produktschnittstelle.

Nachricht in `$TMP/msg.txt`:

```
feat(check): add loomux check gocover and drop dev covergate

The per-function gate moves to internal/verify/gocover, reads the module
from go.mod, and gains --floor. dev covergate is removed; ci/gate.sh calls
check gocover.
```

```sh
git add internal/verify/gocover internal/cli/check.go internal/cli/check_test.go internal/cli/dev.go internal/cli/dev_test.go ci/gate.sh
git commit -F "$TMP/msg.txt"
```

---

### Task 12: `loomux check <profil|arten>` in der CLI

**Files:**
- Modify: `internal/cli/check.go`, `internal/cli/check_test.go`

**Interfaces:**
- Consumes: `verify.ReadConfig`, `verify.LoadPresets`, `verify.Resolve`,
  `verify.ExpandProfile`, `verify.Plan`, `verify.Run`, `verify.WriteCheck`,
  `verify.CheckVerdict`, `verify.WriteShow`, `verify.NewRunID`,
  `verify.CleanCover`, `verify.HasTests`, `detect.Detect`, `hosts.FindRoot`,
  `child.Run`
- Produces (Nähte für Task 14):
  - `var checkStart = child.Run`
  - `var checkLook = exec.LookPath`
  - `var checkExecutable = os.Executable`
  - `var checkNow = time.Now`
  - `func checkRun(args []string, stdout, stderr io.Writer) int`

**Verhalten:**
- `loomux check` ohne Argument: Exit 2,
  `loomux check: name a profile or kinds (lint,types,test,coverage), or one of: commit-msg, gofmt, gocover`.
- `commit-msg`, `gofmt`, `gocover`: wie bisher bzw. Task 11.
- Sonst **zuerst genau ein** Positionsargument (Profil, Arten oder `all`), **danach** die Flags `--root <dir>`, `-v`, `--show` — wie `hook <event> --host … --root …`: Gos `flag` hört am ersten Nicht-Flag auf, deshalb `request := args[0]; fs.Parse(args[1:])`, und übrig gebliebene Argumente nach den Flags sind ein Aufruffehler. Die Aufzeichnungen haben dieselbe Form (`check all --root {{WORLD}}`).
  Mehr oder keins ⇒ Exit 2. Leeres `--root` ⇒ `hosts.FindRoot(cwd)`; findet es
  nichts, gilt `cwd` (ein Projekt ohne `.loomux/config.toml` hat trotzdem
  Presets).
- Lade- und Profilfehler ⇒ stderr `loomux check: <fehler>`, Exit 1.
- `--show` ⇒ `WriteShow`, Exit 0.
- Sonst: `os.MkdirAll(<root>/.loomux/state/cover)`, Plan, Run mit
  `RunOptions{Scope: ScopeCheck, MaxParallel, Timeout aus der Konfiguration,
  Start: checkStart, Look: checkLook, Now: checkNow}`, `WriteCheck`, die Notes
  aus `CheckVerdict` je Zeile auf stdout, `CleanCover(root, runID, code == 0)`,
  Exit `code`.
- `PlanEnv.ImportReady` ist `verify.ImportReady` (angelegt in Task 12 in
  `internal/verify/plan.go`, mit Test; Task 13 benutzt dieselbe Funktion):
  `.godot/global_script_class_cache.cfg` existiert unter `dir`.

- [ ] **Schritt 1: Tests schreiben**

Welten in `t.TempDir()`, Naht `checkStart` durch eine Tabelle ersetzt:

```go
func stubCheck(t *testing.T, answer func(child.Spec) child.Result) *[]string {
	t.Helper()
	var mu sync.Mutex
	seen := []string{}
	oldS, oldL, oldE := checkStart, checkLook, checkExecutable
	checkStart = func(s child.Spec) child.Result {
		mu.Lock()
		seen = append(seen, strings.Join(s.Argv, " "))
		mu.Unlock()
		return answer(s)
	}
	checkLook = func(s string) (string, error) { return s, nil }
	checkExecutable = func() (string, error) { return "loomux", nil }
	t.Cleanup(func() { checkStart, checkLook, checkExecutable = oldS, oldL, oldE })
	return &seen
}

func goWorld(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module m\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "a_test.go"), []byte("package m\n"), 0o644)
	return dir
}

func TestCheckRunsTheGoPresetsInTheirOrder(t *testing.T) {
	root := goWorld(t)
	seen := stubCheck(t, func(child.Spec) child.Result { return child.Result{} })
	var so, se bytes.Buffer
	code := Run([]string{"check", "all", "--root", root}, nil, &so, &se)
	if code != 0 {
		t.Fatalf("%d %q %q", code, so.String(), se.String())
	}
	for _, want := range []string{"lint/go: ok [preset]", "types/go: not-applicable [preset] no command", "test/go: ok [preset]", "coverage/go: ok [preset]"} {
		if !strings.Contains(so.String(), want) {
			t.Errorf("missing %q in %q", want, so.String())
		}
	}
	if !strings.Contains(strings.Join(*seen, "\n"), "go test ./... -count=1 -covermode=set -coverprofile=") {
		t.Errorf("test must measure: %v", *seen)
	}
}

func TestCheckFailsAKindWithNothingToRun(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module m\n"), 0o644)
	stubCheck(t, func(child.Spec) child.Result { return child.Result{} })
	var so, se bytes.Buffer
	if code := Run([]string{"check", "test", "--root", root}, nil, &so, &se); code != 1 || !strings.Contains(so.String(), "nothing to check for `test`") {
		t.Fatalf("%d %q", code, so.String())
	}
}
```

Dazu: ohne Argument (2), zwei Positionsargumente (2), unbekanntes Flag (2),
kaputte Konfiguration (1, stderr nennt die Datei), unbekannte Art (1),
`--show` (0, enthält `[verify.go.lint]`), `-v` zeigt die Ausgabe einer grünen
Lane, eine rote Lane (1), `--root` leer mit `t.Chdir` in eine Welt mit
`.loomux/config.toml` (findet sie), und ohne (nimmt `cwd`).
`checkExecutable` scheitert ⇒ `{loomux}` wird `loomux` (der Name, den der PATH
auflöst) — getestet.

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/cli/ -run Check`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren** — nach dem Verhalten oben.

- [ ] **Schritt 4: Tests laufen lassen**

Run: `go test ./internal/cli/ -cover`
Erwartet: PASS.

- [ ] **Schritt 5: Von Hand gegen dieses Repo**

```sh
go run ./cmd/loomux check all --show
go run ./cmd/loomux check lint
```

Erwartet: `--show` zeigt `go` mit `tests found`; `check lint` meldet
`lint/go: ok [preset]` — oder rot, weil `{loomux} check gofmt .` auch
`third_party/` prüft. Ist es rot, ist das der Grund für die eigene Zeile im
Kopierblock von Task 15 (`gofmt cmd internal`), kein Fehler dieser Aufgabe.

- [ ] **Schritt 6: Commit**

```sh
git add internal/cli/check.go internal/cli/check_test.go
git commit -m "feat(check): run a profile or a list of kinds per stack"
```

---

### Task 13: post-edit auf `[verify]`

**Files:**
- Modify: `internal/hooks/post_edit.go` (neu geschrieben), `internal/hooks/post_edit_test.go` (neu geschrieben), `internal/cli/hook.go`, `internal/cli/hook_test.go`

**Interfaces:**
- Consumes: alles aus `verify`, `detect.Detect`, `wiki.LintReport`, `wiki.Root`, `config.ReadManifest`
- Produces:
  - `func PostToolUse(stdin io.Reader, stdout, stderr io.Writer, root string, budget time.Duration) int`
  - `type EditEnv struct { Start func(child.Spec) child.Result; Look func(string) (string, error); Loomux string; Budget time.Duration; Now func() time.Time; ImportReady func(dir string) bool }`
  - `func ImportReady(dir string) bool` — `.godot/global_script_class_cache.cfg` existiert unter `dir`; zieht als exportierte Funktion nach `verify` (`verify.ImportReady`), damit `check` (Task 12) und post-edit dieselbe benutzen
  - `func RunPostEdit(stdin io.Reader, stdout, stderr io.Writer, root string, env EditEnv) int`
  - `const DefaultBudget = 50 * time.Second`

**Ablauf von `RunPostEdit`:**
1. Nutzlast dekodieren wie bisher; unlesbar oder ohne `file_path`/
   `notebook_path` ⇒ `ExitOK`.
2. Konfiguration, Presets, `Resolve`. Ein Fehler ⇒ stderr
   `loomux hook post-tool-use: <fehler>`, `ExitInternal` (1; blockiert keinen
   Edit, zeigt aber den Fehler).
3. Endung in `eff.Ignored` ⇒ `ExitOK`.
4. Stack `wiki` (`.md`): wie bisher nur, wenn `declaresWikiLayout(root)` und
   `isWikiPath(...)`; dann **ein** Job mit `Fn`, Name `lint/wiki`, `Origin`
   `in-process`, der den alten Wiki-Lauf ruft. `[verify.wiki] lint = false` ⇒
   kein Job. Sonst `ExitOK`.
5. Alle anderen: `kinds, _ := ExpandProfile(cfg, "edit")` und

   ```go
   jobs, err := verify.Plan(eff, verify.Request{Kinds: kinds, Scope: verify.ScopeEdit, File: rel}, verify.PlanEnv{
       Root:        root,
       Loomux:      env.Loomux,
       RunID:       verify.NewRunID(env.Now(), os.Getpid()),
       HasTests:    verify.HasTests,
       ImportReady: env.ImportReady,
   })
   ```

   `rel` ist die Datei relativ zu `root`, mit Schrägstrichen; eine Datei
   außerhalb von `root` ⇒ `ExitOK`. Das Profil `edit` fordert weder `test` noch
   `coverage`; `HasTests` und `ImportReady` werden dort nie gerufen, stehen
   aber da, damit ein Profil `edit`, das ein Wirt um `test` erweitert, richtig
   plant.
6. `Run(jobs, RunOptions{Scope: ScopeEdit, Budget: env.Budget, …})`,
   `WriteEdit(stdout, stderr, outs)`.

Aus dem alten `post_edit.go` bleiben `declaresWikiLayout`, `isWikiPath`,
`wikiDirFor`, `relativeToRoot` und der Wiki-Lauf; `getCommandsForStacks`,
`extensionStackMap`, `explicitIgnoredExtensions`, `commandRunner`,
`laneTool`, `laneAvailable`, `runLane`, `getWorkspaceDir`, `relativeToArea`
und `reportSkipped` entfallen (Task 10 hat `writeSkipped`).

`hook.go`: Flag `--budget` (Typ `time.Duration`, Vorgabe
`hooks.DefaultBudget`); `PostToolUse(stdin, stdout, stderr, resolved, *budget)`.

- [ ] **Schritt 1: Tests neu schreiben**

Die alten Tests prüften die verdrahteten Befehle je Stack; das tun jetzt die
Presets. Die neuen prüfen den Ablauf:

```go
func editEnv(t *testing.T, answer func(child.Spec) child.Result, seen *[]string) EditEnv {
	t.Helper()
	var mu sync.Mutex
	return EditEnv{
		Start: func(s child.Spec) child.Result {
			mu.Lock()
			*seen = append(*seen, s.Dir+"|"+strings.Join(s.Argv, " "))
			mu.Unlock()
			return answer(s)
		},
		Look:   func(s string) (string, error) { return s, nil },
		Loomux: "loomux",
		Budget: DefaultBudget,
		Now:    time.Now,
	}
}

func TestPostEditRunsTheEditProfileOfTheFilesStack(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module m\n"), 0o644)
	seen := []string{}
	payload := `{"tool_input":{"file_path":"` + filepath.ToSlash(filepath.Join(root, "a.go")) + `"}}`
	var so, se bytes.Buffer
	code := RunPostEdit(strings.NewReader(payload), &so, &se, root, editEnv(t, func(child.Spec) child.Result { return child.Result{} }, &seen))
	if code != ExitOK {
		t.Fatalf("%d %q", code, se.String())
	}
	got := strings.Join(seen, "\n")
	if !strings.Contains(got, "go vet ./...") || strings.Contains(got, "go test") {
		t.Fatalf("edit runs lint and types only: %v", seen)
	}
}

func TestPostEditBlocksOnARedLane(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module m\n"), 0o644)
	seen := []string{}
	payload := `{"tool_input":{"file_path":"a.go"}}`
	var so, se bytes.Buffer
	code := RunPostEdit(strings.NewReader(payload), &so, &se, root, editEnv(t, func(s child.Spec) child.Result {
		return child.Result{Code: 1, Stdout: "vet: bad\n"}
	}, &seen))
	if code != ExitDenied || !strings.Contains(se.String(), "vet: bad") {
		t.Fatalf("%d %q", code, se.String())
	}
}

func TestPostEditSkipsAMissingToolOutLoud(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module m\n"), 0o644)
	seen := []string{}
	env := editEnv(t, func(child.Spec) child.Result { return child.Result{} }, &seen)
	env.Look = func(string) (string, error) { return "", errors.New("not found") }
	var so, se bytes.Buffer
	code := RunPostEdit(strings.NewReader(`{"tool_input":{"file_path":"a.go"}}`), &so, &se, root, env)
	if code != ExitOK || !strings.Contains(so.String(), "lane skipped") {
		t.Fatalf("%d %q", code, so.String())
	}
}
```

Dazu: kaputte Nutzlast, leere Pfade, `notebook_path`, ignorierte Endung,
unbekannte Endung, Datei außerhalb der Wurzel, Konfigurationsfehler (Exit 1),
Wiki-Seite grün und rot, `.md` außerhalb des Wikis, `[verify.wiki] lint =
false`, ein TypeScript-Bereich `web/` (Lane läuft in `web`, `{file}` relativ
dazu), Budget erschöpft (Meldung, Exit 0), und `PostToolUse` einmal echt mit
`go.mod` und `sample.go` (übernommen aus
`TestPostToolUseRunsTheGoLaneThroughItsOwnRunner`). In `hook_test.go`: das
Flag `--budget` wird durchgereicht, ein ungültiger Wert ⇒ Exit 2.

- [ ] **Schritt 2: Tests laufen lassen, sie scheitern**

Run: `go test ./internal/hooks/ ./internal/cli/ -run 'PostEdit|PostToolUse|Hook'`
Erwartet: FAIL.

- [ ] **Schritt 3: Implementieren** — nach dem Ablauf oben.

- [ ] **Schritt 4: Tests und Fallsuiten laufen lassen**

Run: `go test ./internal/hooks/ ./internal/cli/ -cover`
Erwartet: PASS. Die 1a-Fälle `hook-post-tool-use/broken-wiki-page` (Exit 2),
`ignored-extension` (0) und `markdown-outside-wiki` (0) bleiben grün — sie
berühren nur Wiki-Lane und Endungstabelle.

- [ ] **Schritt 5: Commit**

```sh
git add internal/hooks internal/cli/hook.go internal/cli/hook_test.go
git commit -m "feat(hooks): run post-edit lanes from [verify] and the presets"
```

---

### Task 14: Fallsuite 2a — Import, Vergleichsklasse `lanes`, Abspielen

**Files:**
- Modify: `internal/cases/case.go` (`LoadCase` kennt `lanes`), `internal/cases/runner.go` (`RunCase` vergleicht sie)
- Create: `internal/cases/lanes.go`, `internal/cases/lanes_test.go`
- Modify: `internal/dev/importcases/importcases.go`, `internal/dev/importcases/importcases_test.go`
- Create: `testdata/cases/2a-map.toml`, `testdata/cases/2a/…` (von `dev import-cases` geschrieben), `internal/cli/cases_2a_test.go`
- Modify: `testdata/cases/README.md`

**Interfaces:**
- Produces:
  - `func KindVerdicts(stdout []byte) map[string]string` — je Art `red`, `ok` oder `neutral`
  - `importcases`: `func foldVerify(dir string, old map[string]any) (map[string]any, error)`

**`KindVerdicts`:** liest jede Zeile nach `^(lint|types|test|coverage)(/[^:]*)?: ([a-z-]+)`.
Rot sind `failed`, `timed-out`, `blocked`, `missing-tool`, `unready` und
`error`. Die alte Form schreibt `unavailable` nicht als Zustand, sondern als
Quelle in Klammern (`types: failed [unavailable]`); das Muster liest dort
`failed` und damit rot, wie Python urteilte. Eine Art ist `red`, wenn eine ihrer Zeilen rot ist; sonst `ok`, wenn eine `ok`
ist; sonst `neutral`. Zeilen, die das Muster nicht treffen (Befehlsausgabe),
zählen nicht.

**`RunCase` mit `Compare == "lanes"`:** Exit-Code gleich **und**
`KindVerdicts(expected) == KindVerdicts(actual)`; stdout wird nicht Byte für
Byte verglichen.

**Falten der alten Konfiguration** (`foldVerify`, gerufen in `translateDir`,
wenn `.ultraloom/config.toml` eine `[verify]` trägt):
- `[verify].<art>` (String oder Liste) ⇒ `verify.project.<art>` (dieselbe
  Form).
- `[verify.<art>]` mit `commands`/`threaded` ⇒ `verify.project.<art>` als
  Tabelle.
- `[verify.coverage].report` ⇒ `verify.project.coverage`.
- `[verify.after]` ⇒ `after` an der jeweiligen Lane unter `verify.project`.
- `[verify.profiles]`, `max_parallel`, `timeout` ⇒ gleichnamig.
- `godot_import` ⇒ `verify.gdscript.import_check`.
- **Für jede so gefaltete Art** und jeden Stack aus
  `detect.Detect(os.DirFS(dir)).Stacks`, der in `verify.StackNames()` steht:
  `verify.<stack>.<art> = false` — die alte Konfiguration ersetzte das Preset.
- `tests` entfällt (Abweichung 12).

**`2a-map.toml`:**

```toml
# Stage 2a translates one command prefix: the old check lives under the
# same verb in loomux.

[[command]]
from = "ultraloom check "
to   = "loomux check "
```

**Suite `cases_2a_test.go`** — nach dem Muster von `cases_1b1_test.go`:

```go
func TestCases2a(t *testing.T) {
	all, err := cases.DiscoverCases(filepath.Join("..", "..", "testdata", "cases", "2a"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != wantCases2a {
		t.Fatalf("found %d cases, want %d", len(all), wantCases2a)
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			outcome, err := cases.RunCase(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				useFakeTools(t, dir)
				return Run(args, stdin, stdout, stderr)
			})
			if err != nil {
				t.Fatal(err)
			}
			reason, approved := approved2a[c.Verb+"/"+c.Name]
			switch {
			case approved && outcome.Passed:
				t.Fatalf("listed as a deviation (%s) but passes; remove it from approved2a and parity/stufe-2a.md", reason)
			case !approved && !outcome.Passed:
				t.Fatalf("%v", outcome.Mismatches)
			}
		})
	}
}
```

- `useFakeTools(t, dir)` lädt `faketool.Load(filepath.Join(dir,
  faketool.FixtureName))` und ersetzt `checkStart` durch einen Runner:
  `argv[0]` gleich dem Wert von `checkExecutable` ⇒ im Prozess
  `Run(append(argv[1:], "--dir", spec.Dir), …)` in einen eigenen Puffer, **ohne** `Chdir` — die Lanes laufen parallel, ein Wechsel des Arbeitsverzeichnisses träfe alle (so läuft `gocover` echt auf dem
  Profil, das die Fixture geschrieben hat); sonst `fixture.Match(argv)` —
  Treffer ⇒ `fixture.Apply(answer, spec.Dir)` und
  `child.Result{Code: answer.Exit, Stdout: answer.Stdout}`, kein Treffer ⇒
  `child.Result{Code: 127, Stderr: "faketool: no answer"}`. `checkLook` meldet
  jeden Namen als gefunden, für den die Fixture ein Präfix hat, und `{loomux}`.
- `approved2a map[string]string` nennt je abweichendem Fall die Nummer in
  `parity/stufe-2a.md`. Die Suite prüft beide Richtungen: Ein genehmigter Fall,
  der plötzlich passt, ist ein Fehler — die Liste bleibt ehrlich.
- `wantCases2a` ist die Zahl der Fälle nach dem Import; in `README.md` die
  Tabelle um die Zeile 2a ergänzen.

**Antworten für die neue Seite:** Die Presets von loomux rufen teils andere
Kommandozeilen als die alte Kette (`npx eslint .` statt `eslint .`,
`go vet ./...` für Go). Die Fixture in `2a-source/<fall>/world/` ist Beleg und
bleibt unberührt. Die zusätzlichen Antworten stehen deshalb einmal in
`testdata/cases/2a-extra-answers.json`, und `dev import-cases` mischt sie mit
einem neuen Flag `--merge-fixture` in jede übersetzte Welt
`2a/<fall>/world/faketool.json`. Ohne sie endet jede neue Lane als
„faketool: no answer".

- [ ] **Schritt 1: `KindVerdicts` mit Tests**

```go
func TestKindVerdicts(t *testing.T) {
	old := []byte("lint: ok [config]\ntypes: failed [unavailable]\nno preset\ntest: failed [preset]\nE1\ncoverage: failed [blocked]\n")
	neu := []byte("lint/go: ok [config] 1.2s\nlint/python: ok [preset] 0.1s\ntypes/go: not-applicable [preset] no command\ntest/go: failed [preset] 3.0s\ncoverage/go: blocked [preset] by test/go\n")
	o, n := KindVerdicts(old), KindVerdicts(neu)
	if o["lint"] != "ok" || o["test"] != "red" || o["coverage"] != "red" {
		t.Fatalf("old %v", o)
	}
	if n["types"] != "neutral" || n["test"] != "red" || n["coverage"] != "red" {
		t.Fatalf("new %v", n)
	}
}
```

Vor dem Schreiben die echte alte Kopfzeile in einer Aufzeichnung aus Task 2
nachlesen (`cat testdata/cases/2a-source/check/*/stdout | head`): Die Form
`<kind>: ok|failed [<source>]` stammt aus `cli.py:270-280`; wie `unavailable`
dort aussieht (`types: failed [unavailable]`), ist aus der Aufzeichnung zu
bestätigen, nicht zu raten. Der Test übernimmt die echte Form.

- [ ] **Schritt 2: `LoadCase` und `RunCase` erweitern, mit Tests** —
`LoadCase` akzeptiert `lanes`; `RunCase` vergleicht wie oben. Tests in
`internal/cases/runner_test.go`: gleicher Exit und gleiche Urteile ⇒ bestanden;
anderes Urteil ⇒ Mismatch `lanes: test ok != red`.

- [ ] **Schritt 3: `foldVerify` mit Tests** — eine Welt mit `pyproject.toml`
und `go.mod` und einer alten `[verify]` wie `mixed-config`; erwartet wird die
gefaltete Tabelle wörtlich (als `map[string]any` verglichen).

- [ ] **Schritt 4: `--merge-fixture` in `dev import-cases`, mit Test** —
liest eine Fixture und hängt ihre Antworten an jede `world/faketool.json` der
importierten Fälle an (Datei fehlt ⇒ angelegt).

- [ ] **Schritt 5: Importieren**

```sh
go run ./cmd/loomux dev import-cases --from testdata/cases/2a-source --to testdata/cases/2a --map testdata/cases/2a-map.toml --merge-fixture testdata/cases/2a-extra-answers.json
```

(Die Flagnamen von `import-cases` vorher mit `go run ./cmd/loomux dev
import-cases -h` bestätigen und die Zeile anpassen.)

- [ ] **Schritt 6: Suite schreiben und laufen lassen**

Run: `go test ./internal/cli/ -run TestCases2a -v`
Erwartet: Jeder Fall besteht oder steht in `approved2a`. Für jeden roten Fall:
Ist die Abweichung eine der Spec (Liste in `parity/stufe-2a.md`, Task 17), in
`approved2a` eintragen; sonst ist es ein Fehler in `verify` — beheben, nicht
eintragen. Erwartete Einträge: `no-marker` (alt `unavailable` rot, neu Exit 1
mit „nothing to check" — gleiches Urteil, möglicherweise sogar grün),
`go-only-*` (Abweichung 16), `cpp-types` (Abweichung 1),
`godot-no-coverage-all` (Abweichung 10).

- [ ] **Schritt 7: Commit**

```sh
git add internal/cases internal/dev/importcases internal/cli/cases_2a_test.go testdata/cases/2a testdata/cases/2a-map.toml testdata/cases/2a-extra-answers.json testdata/cases/README.md
git commit -m "test(cases): replay the stage 2a check cases against loomux"
```

---

### Task 15: **Halt** — der Mensch trägt die Konfiguration ein

`.loomux/config.toml` schreibt kein Agent. Hier hält die Ausführung an.

- [ ] **Schritt 1: Dem Menschen den Block nennen**

Den Block aus der Spec, Abschnitt Selbstnutzung, wörtlich vorlegen:

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

Und warten. Weiter erst, wenn der Mensch sagt, dass er eingetragen ist.

- [ ] **Schritt 2: Prüfen, dass es wirkt**

```sh
go run ./cmd/loomux check precommit --show
```

Erwartet: `[verify.go.lint]` mit `# config`, `measuring` von `test` mit
`-coverpkg` und `# config`.

---

### Task 16: loomux prüft sich mit `check precommit`

**Files:**
- Modify: `ci/gate.sh`, `.githooks/pre-commit`, `.github/workflows/ci.yml`

- [ ] **Schritt 1: `ci/gate.sh` umstellen**

```sh
#!/bin/sh
# The gate of loomux: `loomux check precommit` over the [verify] table in
# .loomux/config.toml -- gofmt and vet, the tests measuring coverage, and the
# per-function coverage gate. The pre-commit hook and every CI forge run
# exactly this script.
set -eu
cd "$(git rev-parse --show-toplevel)"
go run ./cmd/loomux check precommit
```

Warum `-count=1` und das Modulmuster in `-coverpkg` nötig sind, steht jetzt im
Kommentar des Kopierblocks in `.loomux/config.toml`; der alte Kommentar in
`gate.sh` dazu entfällt.

- [ ] **Schritt 2: `.githooks/pre-commit` — neue Eingaben**

```sh
unstaged=$(git diff --name-only -- '*.go' '*.toml' go.mod go.sum testdata .githooks ci .loomux)
untracked=$(git ls-files --others --exclude-standard -- '*.go' '*.toml' go.mod go.sum testdata .githooks ci .loomux)
```

`.loomux/state/` ist git-ignoriert und taucht damit in keiner der beiden
Listen auf.

- [ ] **Schritt 3: CI — `child` unter Linux**

In `.github/workflows/ci.yml`, Job `build-linux`, nach `go vet ./...`:

```yaml
      - run: go test ./internal/child/...
```

und den Kommentar über dem Job um einen Satz ergänzen: `child`s
Prozessgruppen-Arm ist nur hier lauffähig und wird deshalb hier getestet.

- [ ] **Schritt 4: Tor fahren**

Run: `sh ci/gate.sh`
Erwartet: grün; die Ausgabe zeigt `lint/go`, `types/go: not-applicable`,
`test/go`, `coverage/go`, alle `ok`.

- [ ] **Schritt 5: Gegenprobe — ein Tor, das rot werden kann**

Eine Funktion ohne Test anlegen (`internal/verify/zz_probe.go` mit
`func probe() int { return 1 }`), `sh ci/gate.sh` ⇒ rot mit
`not covered: internal/verify/zz_probe.go:… probe 0.0%`. Datei löschen.

- [ ] **Schritt 6: Commit**

```sh
git add ci/gate.sh .githooks/pre-commit .github/workflows/ci.yml
git commit -m "ci: gate loomux with its own check precommit"
```

---

### Task 17: Abweichungsliste `parity/stufe-2a.md`

**Files:**
- Create: `docs/.superpowers/parity/stufe-2a.md`

- [ ] **Schritt 1: Schreiben**

Aufbau wie `parity/stufe-1b-1.md`: Kopf (Stand, Quelle `loomux-1a-source`
`9d01a60`, Zahl der Fälle), dann die Liste 1–16 aus der Spec plus Nachtrag 17 (`clang-format --dry-run --Werror` statt `-i`) und Nachtrag 18 (Workspace-Skripte `npm run typecheck|lint|check` durch direkte `npx`-Aufrufe ersetzt). Je Eintrag: Fall (Name aus
`testdata/cases/2a/`, oder „kein Fall — Unit-Test `<name>`"), altes Verhalten,
neues Verhalten, Begründung, Freigabe (leer; der Mensch trägt sie ein).
Jeder Name aus `approved2a` (Task 14) muss hier stehen und umgekehrt.

- [ ] **Schritt 2: Abgleich prüfen**

```sh
grep -o '"check/[^"]*"' internal/cli/cases_2a_test.go | sort
```

Jeder dieser Namen steht in `stufe-2a.md`.

- [ ] **Schritt 3: Commit**

```sh
git add docs/.superpowers/parity/stufe-2a.md
git commit -m "docs(parity): list the stage 2a deviations"
```

---

### Task 18: Mutationsrunde über `child` und `verify`

- [ ] **Schritt 1: Laufzeit schätzen**

```sh
go test ./internal/child/ -count=1
go test ./internal/verify/... -count=1
```

Die Zeiten notieren. Eine Runde kostet je Mutant etwa einen Testlauf des
Pakets; `child` startet echte Prozessbäume (die Windows-Baumtests warten bis zu
`DrainGrace`). Liegt die Schätzung für `child` über 30 Minuten mit
`--workers` = Kernzahl, die Runde für `child` auf `--family a1,a2` begrenzen
und das in der Akte begründen.

- [ ] **Schritt 2: Runde fahren**

```sh
go run ./cmd/loomux dev mutants ./internal/verify ./internal/verify/gocover
go run ./cmd/loomux dev mutants ./internal/child
```

- [ ] **Schritt 3: Überlebende verfügen**

Je Überlebendem: Test nachreichen (dann neuer Test, neuer Lauf nur dieses
Mutanten mit `--only`), oder als gleichwertig begründen. Beides in
`parity/stufe-2a.md`, Abschnitt „Mutationsrunde", mit Befehl, Commit und
Worker-Zahl, wie in `stufe-1b-1.md`.

- [ ] **Schritt 4: Commit**

```sh
git add internal docs/.superpowers/parity/stufe-2a.md
git commit -m "test(verify): kill the survivors of the stage 2a mutation round"
```

---

### Task 19: Messen

**Files:**
- Create: `testdata/bench/2a-check.json`
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`

- [ ] **Schritt 1: Bench-Fälle anlegen**

`testdata/bench/2a-check.json` mit drei Fällen, `dir` auf eine Welt unter
`testdata/cases/2a-worlds/go-only`, deren Werkzeuge faketool sind (PATH wie in
Task 2):
1. `check-precommit` — `bin/loomux.exe check precommit`;
2. `check-show` — `bin/loomux.exe check all --show` (Laden + Presets + Plan,
   kein Kind: das ist die Eigenzeit);
3. `post-edit-go` — `bin/loomux.exe hook post-tool-use --host claude --root <welt>`
   mit einer Nutzlast für `a.go`.

- [ ] **Schritt 2: Messen**

```sh
go build -o bin/loomux.exe ./cmd/loomux
go run ./cmd/loomux dev bench-hooks -n 20 testdata/bench/2a-check.json
```

Zusätzlich den Parse der Presets allein: ein Benchmark
`BenchmarkLoadPresets` in `presets_test.go`, der `parsePresets(presetsText)`
misst (`go test ./internal/verify/ -bench LoadPresets -run ^$`).

Und der Startzeit-Nachweis:

```sh
GODEBUG=inittrace=1 bin/loomux.exe version 2>&1 | grep -E 'verify|child'
```

Erwartet: Eine Zeile für `verify` darf erscheinen — `sync.OnceValues` als
Paketvariable legt beim Init eine Closure an. Die Regel verbietet Parsen, nicht
Existenz: Die Zeit nach `@…ms` liegt unter 0,05 ms und die Zahl der
Allokationen ist einstellig. Ein Wert darüber heißt, dass beim Start doch
geparst wird.

- [ ] **Schritt 3: Eintragen**

In beide `benchmarks.md` chronologisch: Datum und Uhrzeit, was gemessen wurde,
Basis (`master` `86fa192`, post-edit mit fest verdrahteten Lanes) gegen die
Änderung, kalt und warm; post-edit gegen den 1a-Zielwert 72 ms, der Parse der
Presets einzeln.

- [ ] **Schritt 4: Commit**

```sh
git add testdata/bench/2a-check.json docs/en/benchmarks.md docs/de/benchmarks.md internal/verify/presets_test.go
git commit -m "docs(bench): measure check and post-edit of stage 2a"
```

---

### Task 20: Doku und Specs

**Files:**
- Modify: `README.md`, `README.de.md`, `docs/en/cli-reference.md`, `docs/de/cli-reference.md`, `docs/en/configuration.md`, `docs/de/configuration.md`, `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`, `docs/.superpowers/specs/2026-09-19-loomux-stufe-2a-design.md`

Vorher `ls docs/en docs/de` lesen und die tatsächlichen Dateinamen nehmen.

- [ ] **Schritt 1: Befehlsdoku** — `check <profil|arten>` mit `--root`,
`--show`, `-v`; `check gocover`; `hook post-tool-use --budget`; `dev
covergate` entfernt. Englisch und deutsch, gleiche Struktur.

- [ ] **Schritt 2: Konfigurationsdoku** — `[verify]`, `[verify.profiles]`,
`[verify.<stack>]` mit allen Formen, Platzhaltern, Stacks, Testerkennung,
Zusammenführungsregel, die Zustände und das Urteil je Art. Beispiele aus der
Spec.

- [ ] **Schritt 3: READMEs** — Stand und Fahrplan: Stufe 2 in 2a/2b/2c
zerlegt, 2a fertig.

- [ ] **Schritt 4: Fusions-Spec** — Stufentabelle: Zeile 2 „in drei
Teilstufen zerfallen", darunter eine Tabelle 2a/2b/2c wie bei 1b;
Abhängigkeitsregeln: `verify → child, gitenv, detect, shellwords`,
`hooks → verify, brain/wiki`; die Regel aus `docs/stufe-2-stop-notes`
(Fristen, Fingerabdruck) unter 2c vormerken. Kopfzeile „Stand" nachziehen.

- [ ] **Schritt 5: 2a-Spec** — die sieben Nachträge aus diesem Plan eintragen,
Abweichung 1 anpassen, Eintrag 17 ergänzen, „Stand" auf umgesetzt.

- [ ] **Schritt 6: Commit**

```sh
git add README.md README.de.md docs
git commit -m "docs: document check profiles, [verify] and stage 2a"
```

---

### Task 21: Abschluss

- [ ] **Schritt 1: Tor und Fallsuiten**

```sh
sh ci/gate.sh
go test ./internal/cli/ -run 'TestCases' -count=1
```

Erwartet: grün.

- [ ] **Schritt 2: Fertig-Kriterien der Fusions-Spec abhaken**

1. Fälle grün oder freigegeben in `stufe-2a.md` (Freigaben trägt der Mensch).
2. 100 % Coverage — Tor grün.
3. Mutationsrunde dokumentiert — Task 18.
4. Zielwerte gemessen — Task 19.
5. loomux nutzt `check` selbst — Task 16.

- [ ] **Schritt 3: Übergabe**

Commits nach Thema gruppieren und den Pull Request öffnen: das übernimmt der
Skill `release-pr` (Label `release:minor`, Changelog mit `Added`: `check
<profile>`, `check gocover`, `[verify]`; `Changed`: post-edit liest
`[verify]`; `Removed`: `dev covergate`). Pushen tut der Mensch.
