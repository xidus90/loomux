# loomux Stufe 4e: die vorbereitete Umstellung — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die Werkzeuge bauen, mit denen ein LLM die Umstellung eines Projekts auf loomux vorbereitet und ein Mensch sie mit einer Zeile ausführt, und die Baseline und den Vergleich alt gegen neu messen; danach zwei Piloten und eine Welle durchführen.

**Architecture:** Der Code hängt an vorhandenen Bausteinen: `loomux dev bench hooks` misst beliebige Programme (ein Kaltlauf, n Warmläufe) und schreibt `benchreport.Report`; dazu kommen der Mittelwert (`benchreport.Mean`), ein Vergleich zweier Berichte nach Fallnamen (`internal/dev/benchcompare`), ein Fallgenerator aus der Hook-Konfiguration eines Projekts (`internal/dev/benchcases`) und das Paket `internal/switchover` mit der Vorlage für das `apply.sh` und dem Entfernen abgelöster Hook-Einträge. Die Durchführung (Baseline, Piloten, Welle) sind Abläufe mit Befehlen, keine Go-Aufgaben.

**Tech Stack:** Go (Pakete unter `internal/dev` und `internal/switchover`, Befehle unter `internal/cli`), POSIX-`sh` in Git Bash für das Skript.

**Spec:** `docs/.superpowers/specs/2026-09-29-loomux-stufe-4e-umstellung-vorbereitet-design.md` (verbindlich); davor `2026-09-28-loomux-stufe-4e-design.md`; Fusions-Spec Nachtrag #26.

**Ausführung:** Die Tasks 1 bis 6 sind Implementierer-Stoff (subagent-driven, ein frischer Prüfer je Task). Die Tasks 7 bis 12 führt der Controller mit dem Menschen Schritt für Schritt: sie messen an den echten Projekten und warten mitten im Task auf einen Befehl des Menschen (Task 9 Schritt 3), das verträgt sich nicht mit einem Ablauf, der zwischen Tasks nie anhält.

## Global Constraints

- Nichts wird auf `master` gearbeitet; Zweig `docs/stage-4e-spec`. Spec, Plan und Code liegen in einem Pull Request (kein Plan-PR allein). Nobody but a human pushes.
- Code, Bezeichner, Kommentare, Meldungen und Commit-Texte englisch; Arbeitspapiere unter `docs/.superpowers/` deutsch; die Berichte in `docs/en/benchmarks.md` und `docs/de/benchmarks.md` in der Sprache der Datei.
- Coverage 100 % je Funktion; ein Ausschluss nur mit `//coverage:exempt <reason>` direkt über `func`.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst (die Skriptvorlage wird beim ersten Gebrauch geladen, nicht beim Start).
- Ein Commit je Änderung, Conventional Commits, kein Arbeitspapier im Text (kein „Stufe“, „Plan“, „Task“), KEIN Co-Authored-By, kein Modell als Autor.
- Der Agent schreibt nie `.loomux/config.toml`, führt nie `init`, `config apply`, `merge-hook install` aus und schreibt nichts in `brain-knowledge`; er liest, bereitet im Ablageort `.superpowers/switchover/<projekt>/` vor und prüft danach nur lesend. Jede Schreibaktion in Projekt, Registry oder Vault steht im `apply.sh`.
- Messungen: ein Kaltlauf, danach **fünf** Warmläufe (`-n 5`); berichtet werden Mittelwert, Median, Min und Max; Zeitpunkt, Werkzeugversion und Umgebung stehen im Kopf (`benchHead`).
- Schreibende Shell-Zeilen mit Backslashes nie im Bash-Heredoc; Dateien per Write/Edit. Ein bedingt erlaubter Befehl (`loomux init --dry-run`, `loomux area check`) läuft als schlichte Einzelzeile ohne Kette, Umleitung, `cd &&`.
- Pfade mit `#` (`C:/Users/micro/Documents/#GIT/…`) stehen in Skripten immer in Anführungszeichen.

## Review Focus

- `Compare`: zwei Fälle gleichen Namens in **einer** Messung sind ein Fehler; ein Fall mit `applicable: false` zählt weder als verglichen noch als neu oder weggefallen.
- `Factor`: eine Seite mit Zeit 0 ergibt kein Verhältnis (`–`), keine Division durch null und kein `+Inf`.
- `split` (Befehlszeilen der Hook-Konfiguration): ein Pfad mit `#` und Leerzeichen in Anführungszeichen bleibt ein Argument; ein nicht geschlossenes Anführungszeichen ist ein Fehler.
- `Build` (Fälle aus `settings.json`): eine `settings.json` ohne `hooks`, ein Matcher, der kein gültiger regulärer Ausdruck ist, und eine leere Befehlszeile sind Fehler, kein Absturz.
- `PruneHooks`: eine Gruppe, die alte **und** fremde Befehle mischt, bleibt stehen und wird gemeldet; die Reihenfolge der Ereignisse und aller anderen Schlüssel der Datei bleibt.
- `apply.sh`: ein ungesichertes Projekt oder ein Vault ohne Remote bricht **vor** dem ersten Schreiben ab; ein zweiter Lauf ändert nichts; `--check` schreibt nichts.
- `apply.sh`: hat sich die Registry seit der Vorbereitung geändert, bricht das Skript ab, statt sie zu überschreiben.

---

## Dateien

| Datei | Verantwortung |
|---|---|
| `internal/dev/benchreport/stats.go` (ändern) | `Mean` |
| `internal/dev/benchcompare/compare.go` (neu) | `Compare`, `Factor`, `Markdown` |
| `internal/dev/benchcases/cases.go` (neu) | `Build`, `split` |
| `internal/switchover/render.go`, `apply.sh.tmpl` (neu) | `Params`, `Render` |
| `internal/switchover/prune.go` (neu) | `PruneHooks` |
| `internal/cli/switchover.go` (neu) | `dev switchover render`, `dev switchover prune-hooks` |
| `internal/cli/benchcompare.go` (neu) | `dev bench compare`, `dev bench cases` |
| `internal/cli/dev.go` (ändern) | die zwei Tabellen der Untergruppen und `benchUsage` |
| `docs/en/cli-reference.md`, `docs/de/cli-reference.md`, `README.md`, `README.de.md` | die neuen Befehle |
| `.superpowers/switchover/<projekt>/` (git-ignoriert) | Ablageort je Ziel: `baseline/`, `after/`, `apply.sh`, `config.toml.new`, `registry.toml.new`, `PLAN.md` |
| `docs/.superpowers/parity/bench-4e-<projekt>.md` (neu, je Ziel) | Bericht je Projekt, deutsch, echter Name |
| `docs/en/benchmarks.md`, `docs/de/benchmarks.md` (ändern) | ein anonymisierter Bericht „Beispielprojekt 1“ bis „9“ |

---

## Stück 1: Messwerkzeuge (Go)

### Task 1: `benchreport.Mean`

**Files:**
- Modify: `internal/dev/benchreport/stats.go`
- Test: `internal/dev/benchreport/stats_test.go`

**Interfaces:**
- Produces: `func Mean(ms []float64) float64` — Mittelwert; 0 bei leerer Liste.

- [ ] **Step 1: den Test schreiben**

```go
func TestMeanAveragesEveryValueAndTakesZeroForNone(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{"none", nil, 0},
		{"one", []float64{7}, 7},
		// Unsorted with an outlier: a mean that took the middle value or
		// dropped the extremes would not give 5.
		{"spread", []float64{10, 0, 5}, 5},
		{"even", []float64{1, 2, 3, 4}, 2.5},
	}
	for _, c := range cases {
		if got := Mean(c.in); got != c.want {
			t.Errorf("%s: Mean(%v) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: rot laufen lassen.** Run: `go test ./internal/dev/benchreport -run TestMean -count=1`. Expected: FAIL (`undefined: Mean`). Zuerst einen Stub `func Mean([]float64) float64 { return -1 }` anlegen, damit der rote Lauf eine Assertion zeigt, nicht einen Build-Fehler.

- [ ] **Step 3: implementieren**

```go
// Mean is the average of the warm runs. The median stays the figure the
// tables of earlier measurements carry; the mean is what a series of a few
// runs is compared by.
func Mean(ms []float64) float64 {
	if len(ms) == 0 {
		return 0
	}
	var sum float64
	for _, v := range ms {
		sum += v
	}
	return sum / float64(len(ms))
}
```

- [ ] **Step 4: grün.** Run: `go test ./internal/dev/benchreport -count=1`. Expected: PASS.
- [ ] **Step 5: Mutation per `go test -overlay`:** `sum / float64(len(ms))` durch `sum / float64(len(ms)-1)` ersetzen: `TestMean…` wird rot. Coverage 100 %.
- [ ] **Step 6: Commit** `feat(bench): add the mean of the warm runs`.

### Task 2: `internal/dev/benchcompare`

**Files:**
- Create: `internal/dev/benchcompare/compare.go`
- Test: `internal/dev/benchcompare/compare_test.go`

**Interfaces:**
- Consumes: `benchreport.Timing`, `benchreport.Mean`, `benchreport.FormatMS`.
- Produces:
  - `type Row struct { Name string; Before, After benchreport.Timing }`
  - `type Result struct { Compared []Row; Added []benchreport.Timing; Dropped []benchreport.Timing }`
  - `func Compare(before, after []benchreport.Timing) (Result, error)`
  - `func Factor(before, after float64) float64`
  - `func Markdown(r Result, title, lang string) (string, error)` (`lang` ist `"de"` oder `"en"`, sonst Fehler)

- [ ] **Step 1: die Tests schreiben**

```go
package benchcompare

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

func timing(name string, cold float64, warm ...float64) benchreport.Timing {
	return benchreport.Summarize(name, cold, warm)
}

func TestCompareSortsCasesByWhoMeasuredThem(t *testing.T) {
	before := []benchreport.Timing{timing("guard", 40, 30, 32), timing("wiki lint", 90, 80, 82), timing("both", 5, 5)}
	after := []benchreport.Timing{timing("guard", 20, 10, 12), timing("graph", 15, 14, 16), timing("both", 6, 6)}
	r, err := Compare(before, after)
	if err != nil {
		t.Fatal(err)
	}
	names := func(ts []benchreport.Timing) []string {
		var out []string
		for _, x := range ts {
			out = append(out, x.Name)
		}
		return out
	}
	if len(r.Compared) != 2 || r.Compared[0].Name != "guard" || r.Compared[1].Name != "both" {
		t.Fatalf("compared = %+v, want guard then both in the order of the earlier run", r.Compared)
	}
	if got := names(r.Added); len(got) != 1 || got[0] != "graph" {
		t.Errorf("added = %v, want [graph]", got)
	}
	if got := names(r.Dropped); len(got) != 1 || got[0] != "wiki lint" {
		t.Errorf("dropped = %v, want [wiki lint]", got)
	}
}

func TestCompareRefusesTheSameNameTwiceInOneRun(t *testing.T) {
	dup := []benchreport.Timing{timing("guard", 1, 1), timing("guard", 2, 2)}
	if _, err := Compare(dup, nil); err == nil || !strings.Contains(err.Error(), `"guard"`) {
		t.Errorf("before: err = %v, want one naming guard", err)
	}
	if _, err := Compare(nil, dup); err == nil {
		t.Errorf("after: no error for a duplicated name")
	}
}

func TestCompareIgnoresACaseThatDoesNotApply(t *testing.T) {
	no := false
	gone := timing("x", 1, 1)
	gone.Applicable = &no
	r, err := Compare([]benchreport.Timing{gone}, []benchreport.Timing{timing("x", 1, 1)})
	if err != nil {
		t.Fatal(err)
	}
	// The earlier side does not apply: the later one is new, nothing is compared.
	if len(r.Compared) != 0 || len(r.Added) != 1 || len(r.Dropped) != 0 {
		t.Errorf("result = %+v", r)
	}
}

func TestFactorIsHowManyTimesFasterTheLaterRunIs(t *testing.T) {
	cases := []struct {
		before, after, want float64
	}{
		{30, 10, 3},
		{10, 30, 1.0 / 3},
		{0, 10, 0},
		{10, 0, 0},
		{-1, 10, 0},
	}
	for _, c := range cases {
		if got := Factor(c.before, c.after); got != c.want {
			t.Errorf("Factor(%v, %v) = %v, want %v", c.before, c.after, got, c.want)
		}
	}
}

func TestMarkdownCarriesTitleValuesAndTheThreeLists(t *testing.T) {
	r, _ := Compare(
		[]benchreport.Timing{timing("guard", 40, 30, 30, 30, 30, 30), timing("wiki lint", 90, 80, 80, 80, 80, 80)},
		[]benchreport.Timing{timing("guard", 20, 10, 10, 10, 10, 10), timing("graph", 15, 14, 14, 14, 14, 14)})
	de, err := Markdown(r, "Beispielprojekt 3", "de")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Beispielprojekt 3", "## Verglichen", "## Neu", "## Weggefallen",
		"guard", "3.00×", "graph", "wiki lint", "1 verglichen", "1 schneller"} {
		if !strings.Contains(de, want) {
			t.Errorf("de: %q missing in\n%s", want, de)
		}
	}
	en, _ := Markdown(r, "Example project 3", "en")
	for _, want := range []string{"## Compared", "## New", "## Dropped", "1 compared", "1 faster"} {
		if !strings.Contains(en, want) {
			t.Errorf("en: %q missing in\n%s", want, en)
		}
	}
	if _, err := Markdown(r, "x", "fr"); err == nil {
		t.Errorf("an unknown language must be refused")
	}
}

func TestMarkdownCountsSlowerAndUnclearApart(t *testing.T) {
	r, _ := Compare(
		[]benchreport.Timing{timing("a", 1, 10), timing("b", 1, 0)},
		[]benchreport.Timing{timing("a", 1, 30), timing("b", 1, 5)})
	out, _ := Markdown(r, "t", "en")
	if !strings.Contains(out, "1 slower") || !strings.Contains(out, "1 unclear") {
		t.Errorf("summary wrong:\n%s", out)
	}
}
```

- [ ] **Step 2: rot laufen lassen** (mit Stubs, die `Result{}`/`0`/`""` liefern): `go test ./internal/dev/benchcompare -count=1`, Expected: FAIL mit Assertions.
- [ ] **Step 3: implementieren**

```go
// Package benchcompare sets two runs of the same cases side by side: what
// both measured with the factor between them, what only the later run has
// (new) and what only the earlier one had (dropped). Cases are paired by
// name, so the two case files give the same thing the same name.
package benchcompare

import (
	"fmt"
	"strings"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// Row is one case both runs measured.
type Row struct {
	Name          string
	Before, After benchreport.Timing
}

// Result sorts the cases of two runs by who measured them.
type Result struct {
	Compared []Row
	Added    []benchreport.Timing
	Dropped  []benchreport.Timing
}

func applies(t benchreport.Timing) bool { return t.Applicable == nil || *t.Applicable }

// index keys a run by case name and refuses a name that occurs twice: two
// lines of one name make the pairing a guess.
func index(run []benchreport.Timing) (map[string]benchreport.Timing, error) {
	byName := map[string]benchreport.Timing{}
	for _, t := range run {
		if _, dup := byName[t.Name]; dup {
			return nil, fmt.Errorf("case %q occurs twice in one run", t.Name)
		}
		byName[t.Name] = t
	}
	return byName, nil
}

// Compare pairs the cases of two runs by name. A case that does not apply
// on a side is left out of that side.
func Compare(before, after []benchreport.Timing) (Result, error) {
	if _, err := index(before); err != nil {
		return Result{}, err
	}
	later, err := index(after)
	if err != nil {
		return Result{}, err
	}
	var r Result
	paired := map[string]bool{}
	for _, b := range before {
		if !applies(b) {
			continue
		}
		a, ok := later[b.Name]
		if !ok || !applies(a) {
			r.Dropped = append(r.Dropped, b)
			continue
		}
		paired[b.Name] = true
		r.Compared = append(r.Compared, Row{Name: b.Name, Before: b, After: a})
	}
	for _, a := range after {
		if applies(a) && !paired[a.Name] {
			r.Added = append(r.Added, a)
		}
	}
	return r, nil
}

// Factor is how many times faster the later value is: 2 is twice as fast,
// 0.5 half as fast. A side with no time gives none (0), not a division.
func Factor(before, after float64) float64 {
	if before <= 0 || after <= 0 {
		return 0
	}
	return before / after
}

func formatFactor(f float64) string {
	if f == 0 {
		return "–"
	}
	return fmt.Sprintf("%.2f×", f)
}

type words struct {
	compared, added, dropped, name, before, after, cold, warmMean, median, exit, factor, none string
	summary                                                                                   string // verglichen, schneller, langsamer, unklar, neu, weggefallen
}

var languages = map[string]words{
	"de": {compared: "Verglichen", added: "Neu", dropped: "Weggefallen", name: "Fall", before: "alt", after: "neu",
		cold: "kalt", warmMean: "warm Ø", median: "Median", exit: "Exit", factor: "×", none: "keine",
		summary: "%d verglichen, %d schneller, %d langsamer, %d unklar, %d neu, %d weggefallen"},
	"en": {compared: "Compared", added: "New", dropped: "Dropped", name: "case", before: "before", after: "after",
		cold: "cold", warmMean: "warm mean", median: "median", exit: "exit", factor: "×", none: "none",
		summary: "%d compared, %d faster, %d slower, %d unclear, %d new, %d dropped"},
}

// Markdown is the comparison as a report: a heading, one summary line and
// the three lists. title names the project, or the anonymised example.
func Markdown(r Result, title, lang string) (string, error) {
	w, ok := languages[lang]
	if !ok {
		return "", fmt.Errorf("unknown language %q; known are de and en", lang)
	}
	faster, slower, unclear := 0, 0, 0
	for _, row := range r.Compared {
		switch f := Factor(benchreport.Mean(row.Before.WarmMS), benchreport.Mean(row.After.WarmMS)); {
		case f > 1:
			faster++
		case f > 0 && f < 1:
			slower++
		default:
			unclear++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, w.summary+"\n\n", len(r.Compared), faster, slower, unclear, len(r.Added), len(r.Dropped))
	fmt.Fprintf(&b, "## %s\n\n", w.compared)
	fmt.Fprintf(&b, "| %s | %s %s | %s %s | %s | %s %s | %s %s | %s | %s %s | %s %s | %s %s | %s %s |\n", w.name,
		w.cold, w.before, w.cold, w.after, w.factor, w.warmMean, w.before, w.warmMean, w.after, w.factor,
		w.median, w.before, w.median, w.after, w.exit, w.before, w.exit, w.after)
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|\n")
	for _, row := range r.Compared {
		mb, ma := benchreport.Mean(row.Before.WarmMS), benchreport.Mean(row.After.WarmMS)
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %v | %v |\n", row.Name,
			benchreport.FormatMS(row.Before.ColdMS), benchreport.FormatMS(row.After.ColdMS),
			formatFactor(Factor(row.Before.ColdMS, row.After.ColdMS)),
			benchreport.FormatMS(mb), benchreport.FormatMS(ma), formatFactor(Factor(mb, ma)),
			benchreport.FormatMS(row.Before.MedianMS), benchreport.FormatMS(row.After.MedianMS),
			row.Before.ExitCodes, row.After.ExitCodes)
	}
	for _, list := range []struct {
		title string
		rows  []benchreport.Timing
	}{{w.added, r.Added}, {w.dropped, r.Dropped}} {
		fmt.Fprintf(&b, "\n## %s\n\n", list.title)
		if len(list.rows) == 0 {
			fmt.Fprintf(&b, "%s\n", w.none)
			continue
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n|---|---:|---:|---:|\n", w.name, w.cold, w.warmMean, w.median)
		for _, t := range list.rows {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", t.Name, benchreport.FormatMS(t.ColdMS),
				benchreport.FormatMS(benchreport.Mean(t.WarmMS)), benchreport.FormatMS(t.MedianMS))
		}
	}
	return b.String(), nil
}
```

- [ ] **Step 4: grün.** Run: `go test ./internal/dev/benchcompare -count=1`. Expected: PASS, 100 % Coverage je Funktion (`go test -coverprofile`, `go tool cover -func`).
- [ ] **Step 5: Mutationsrunde** (die Suite dieses Pakets ist kurz, `loomux dev mutants` ist hier brauchbar; sonst per `-overlay`): `f > 1` zu `f >= 1`, die zwei `<= 0` in `Factor`, die Doppelt-Prüfung in `index`, die Bedingung `applies(a)` in `Compare`. Kein Mutant darf überleben; Überlebende begründet in die Akte.
- [ ] **Step 6: Commit** `feat(bench): compare two runs of the same cases`.

### Task 3: `internal/dev/benchcases` — Fälle aus der Hook-Konfiguration

**Files:**
- Create: `internal/dev/benchcases/cases.go`
- Test: `internal/dev/benchcases/cases_test.go`

**Interfaces:**
- Consumes: `benchhooks.Case`, `benchhooks.Step`.
- Produces:
  - `type Payload struct { Name string; Data []byte }` — Dateiname im Ausgabeordner und Inhalt
  - `func Build(settings []byte, root, file, dir string) ([]benchhooks.Case, []Payload, error)` — je Ereignis mit passenden Befehlen ein Fall; `dir` ist der Ausgabeordner, in dem die Nutzlasten liegen werden; `file` ist eine Markdown-Datei des Projekts (absolut)
  - `func split(command string) ([]string, error)`

Regeln: Reihenfolge der Ereignisse `SessionStart`, `PreToolUse`, `PostToolUse`, `SubagentStart`, `SubagentStop`, `Stop`. Bei `PreToolUse` und `PostToolUse` zählt eine Gruppe, wenn ihr `matcher` (leer = alles) als ganzer regulärer Ausdruck auf `Edit` passt; bei den anderen Ereignissen zählen alle Gruppen (ein Matcher gilt dort nicht). `${CLAUDE_PROJECT_DIR}` wird durch `root` ersetzt, dann in Argumente geteilt. Mehr als ein Befehl je Ereignis: `Mode` `par` (der Host startet sie zugleich), sonst `single`. `Name`: `<Ereignis>`, bei den zwei Werkzeug-Ereignissen `<Ereignis> (Edit on <Basisname von file>)`. `Stdin`: `<dir>/payload-<Ereignis>.json`.

- [ ] **Step 1: die Tests schreiben**

```go
package benchcases

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const oldSettings = `{
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "command": "uv run --project \"${CLAUDE_PROJECT_DIR}/.ultraloom/vendor/ultraloom\" ultraloom hook session-start --root \"${CLAUDE_PROJECT_DIR}\""}]}],
    "PreToolUse": [
      {"matcher": "Write|Edit|Bash", "hooks": [{"type": "command", "command": "ulguard --root \"${CLAUDE_PROJECT_DIR}\""}]},
      {"matcher": "NotebookEdit", "hooks": [{"type": "command", "command": "never-matches"}]},
      {"matcher": "", "hooks": [{"type": "command", "command": "brain guard"}]}
    ],
    "Stop": [{"matcher": "wiki", "hooks": [{"type": "command", "command": "brain wiki-gate --root \"${CLAUDE_PROJECT_DIR}\""}]}]
  }
}`

func TestBuildMakesOneCasePerEventInFixedOrder(t *testing.T) {
	root := "C:/Users/me/Documents/#GIT/my project"
	cases, payloads, err := Build([]byte(oldSettings), root, root+"/README.md", "C:/out")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, c := range cases {
		names = append(names, c.Name)
	}
	want := []string{"SessionStart", "PreToolUse (Edit on README.md)", "Stop"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	pre := cases[1]
	// Two groups match Edit (the plain one and the empty matcher), the
	// NotebookEdit one does not: two steps, started together.
	if pre.Mode != "par" || len(pre.Steps) != 2 {
		t.Fatalf("PreToolUse = mode %q with %d steps, want par with 2", pre.Mode, len(pre.Steps))
	}
	if got := pre.Steps[0].Argv; !reflect.DeepEqual(got, []string{"ulguard", "--root", root}) {
		t.Errorf("argv = %q, want the root, with its space and #, as one argument", got)
	}
	if cases[0].Mode != "single" || cases[0].Dir != root || cases[0].Stdin != "C:/out/payload-SessionStart.json" {
		t.Errorf("SessionStart case = %+v", cases[0])
	}
	if len(payloads) != 3 || payloads[0].Name != "payload-SessionStart.json" {
		t.Errorf("payloads = %d, first %q", len(payloads), payloads[0].Name)
	}
	var edit struct {
		Tool  string `json:"tool_name"`
		Input struct {
			Path string `json:"file_path"`
		} `json:"tool_input"`
		Cwd string `json:"cwd"`
	}
	if err := json.Unmarshal(payloads[1].Data, &edit); err != nil {
		t.Fatal(err)
	}
	if edit.Tool != "Edit" || edit.Input.Path != root+"/README.md" || edit.Cwd != root {
		t.Errorf("edit payload = %+v", edit)
	}
}

func TestBuildIgnoresTheMatcherOfAnEventWithoutTools(t *testing.T) {
	// The Stop group carries the matcher "wiki", which matches no tool.
	cases, _, err := Build([]byte(oldSettings), "/r", "/r/a.md", "/o")
	if err != nil {
		t.Fatal(err)
	}
	if last := cases[len(cases)-1]; last.Name != "Stop" || len(last.Steps) != 1 {
		t.Errorf("last case = %+v, want Stop with its one step", last)
	}
}

func TestMatchesToolAnchorsTheWholeNameAndKnowsTheTwoWildcards(t *testing.T) {
	cases := []struct {
		matcher string
		want    bool
	}{
		{"", true}, {"*", true}, {"Edit", true}, {"Write|Edit", true},
		{"NotebookEdit", false}, {"Edi", false}, {"Edit|", true},
	}
	for _, c := range cases {
		got, err := matchesTool(c.matcher, "Edit")
		if err != nil || got != c.want {
			t.Errorf("matchesTool(%q) = %v, %v; want %v", c.matcher, got, err, c.want)
		}
	}
	if _, err := matchesTool("(", "Edit"); err == nil {
		t.Errorf("a broken expression must be an error")
	}
}

func TestBuildCountsAStarMatcherAsEveryTool(t *testing.T) {
	settings := `{"hooks": {"PreToolUse": [{"matcher": "*", "hooks": [{"command": "a"}]}, {"hooks": [{"command": "b"}]}]}}`
	cases, _, err := Build([]byte(settings), "/r", "/r/a.md", "/o")
	if err != nil || len(cases) != 1 || len(cases[0].Steps) != 2 {
		t.Fatalf("cases = %+v, err %v; want one case with both steps", cases, err)
	}
}

func TestPayloadCarriesWhatEachEventNeeds(t *testing.T) {
	decode := func(event string) map[string]any {
		data, err := payload(event, "/r", "/r/a.md")
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		return doc
	}
	if d := decode("SessionStart"); d["source"] != "startup" || d["cwd"] != "/r" {
		t.Errorf("SessionStart = %v", d)
	}
	if d := decode("Stop"); d["stop_hook_active"] != false {
		t.Errorf("Stop = %v", d)
	}
	if d := decode("PostToolUse"); d["tool_response"] == nil || d["tool_name"] != "Edit" {
		t.Errorf("PostToolUse = %v", d)
	}
	if d := decode("PreToolUse"); d["tool_response"] != nil || d["tool_name"] != "Edit" {
		t.Errorf("PreToolUse = %v", d)
	}
	if d := decode("SubagentStop"); d["tool_name"] != nil || d["source"] != nil || d["hook_event_name"] != "SubagentStop" {
		t.Errorf("SubagentStop = %v", d)
	}
}

func TestBuildRefusesWhatItCannotMeasure(t *testing.T) {
	cases := map[string]string{
		"no hooks":        `{}`,
		"empty hooks":     `{"hooks": {}}`,
		"bad matcher":     `{"hooks": {"PreToolUse": [{"matcher": "(", "hooks": [{"command": "x"}]}]}}`,
		"empty command":   `{"hooks": {"Stop": [{"hooks": [{"command": ""}]}]}}`,
		"open quote":      `{"hooks": {"Stop": [{"hooks": [{"command": "x \"y"}]}]}}`,
		"not json":        `{`,
	}
	for name, settings := range cases {
		if _, _, err := Build([]byte(settings), "/r", "/r/a.md", "/o"); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestSplitKeepsQuotedWordsWhole(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`a b  c`, []string{"a", "b", "c"}},
		{`a "b c" d`, []string{"a", "b c", "d"}},
		{`a 'b "c"' d`, []string{"a", `b "c"`, "d"}},
		{`a "b \"c\" d"`, []string{"a", `b "c" d`}},
		{`x"y z"w`, []string{"xy zw"}},
		{`""`, []string{""}},
	}
	for _, c := range cases {
		got, err := split(c.in)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("split(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	if _, err := split(`a 'b`); err == nil || !strings.Contains(err.Error(), "unterminated") {
		t.Errorf("an open quote must be an error, got %v", err)
	}
}
```

- [ ] **Step 2: rot laufen lassen** (Stubs: `Build` gibt `nil, nil, nil`, `split` gibt `nil, nil`): `go test ./internal/dev/benchcases -count=1`, Expected: FAIL mit Assertions.
- [ ] **Step 3: implementieren**

```go
// Package benchcases turns the hook configuration of a project into the
// case file `loomux dev bench hooks` measures, so the inventory of what runs
// on an edit comes from the project and not from a list written by hand. It
// reads the old configuration before a project is switched over and the new
// one after, and both sides name a case after its event, which is what lets
// `loomux dev bench compare` pair them.
package benchcases

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xidus90/loomux/internal/dev/benchhooks"
)

// Payload is one stdin file a case reads: its name within the output
// directory and its content.
type Payload struct {
	Name string
	Data []byte
}

// events are the hook events of a Claude settings file, in the order a
// session meets them.
var events = []string{"SessionStart", "PreToolUse", "PostToolUse", "SubagentStart", "SubagentStop", "Stop"}

// toolEvents are the events whose matcher names a tool.
func toolEvent(event string) bool { return event == "PreToolUse" || event == "PostToolUse" }

type settingsFile struct {
	Hooks map[string][]struct {
		Matcher string `json:"matcher"`
		Hooks   []struct {
			Command string `json:"command"`
		} `json:"hooks"`
	} `json:"hooks"`
}

// Build makes one case per event that has a command for the sample edit.
// Commands of one event run together (mode par), as a host starts them.
func Build(settings []byte, root, file, dir string) ([]benchhooks.Case, []Payload, error) {
	var parsed settingsFile
	if err := json.Unmarshal(settings, &parsed); err != nil {
		return nil, nil, fmt.Errorf("settings: not valid JSON: %w", err)
	}
	var cases []benchhooks.Case
	var payloads []Payload
	for _, event := range events {
		var steps []benchhooks.Step
		for _, group := range parsed.Hooks[event] {
			if toolEvent(event) {
				ok, err := matchesTool(group.Matcher, "Edit")
				if err != nil {
					return nil, nil, fmt.Errorf("%s: matcher %q: %w", event, group.Matcher, err)
				}
				if !ok {
					continue
				}
			}
			for _, h := range group.Hooks {
				argv, err := split(strings.ReplaceAll(h.Command, "${CLAUDE_PROJECT_DIR}", root))
				if err != nil {
					return nil, nil, fmt.Errorf("%s: %w", event, err)
				}
				if len(argv) == 0 {
					return nil, nil, fmt.Errorf("%s: a hook names no command", event)
				}
				steps = append(steps, benchhooks.Step{Argv: argv})
			}
		}
		if len(steps) == 0 {
			continue
		}
		name := event
		if toolEvent(event) {
			name = fmt.Sprintf("%s (Edit on %s)", event, filepath.Base(file))
		}
		mode := "single"
		if len(steps) > 1 {
			mode = "par"
		}
		payloadName := "payload-" + event + ".json"
		cases = append(cases, benchhooks.Case{Name: name, Dir: root, Stdin: dir + "/" + payloadName, Mode: mode, Steps: steps})
		data, err := payload(event, root, file)
		if err != nil {
			return nil, nil, err
		}
		payloads = append(payloads, Payload{Name: payloadName, Data: data})
	}
	if len(cases) == 0 {
		return nil, nil, fmt.Errorf("settings: no hook of any event applies to an edit")
	}
	return cases, payloads, nil
}

// matchesTool applies a hook matcher to a tool name: an empty matcher and "*"
// match every tool, anything else is a regular expression that must match the
// whole name.
func matchesTool(matcher, tool string) (bool, error) {
	if matcher == "" || matcher == "*" {
		return true, nil
	}
	re, err := regexp.Compile("^(?:" + matcher + ")$")
	if err != nil {
		return false, err
	}
	return re.MatchString(tool), nil
}

// payload is the stdin a host hands the hook of an event: the event, the
// working directory and, for the tool events, an edit of file.
func payload(event, root, file string) ([]byte, error) {
	doc := map[string]any{"hook_event_name": event, "cwd": root}
	edit := map[string]any{"file_path": file, "old_string": "a", "new_string": "b"}
	switch event {
	case "PreToolUse":
		doc["tool_name"], doc["tool_input"] = "Edit", edit
	case "PostToolUse":
		doc["tool_name"], doc["tool_input"], doc["tool_response"] = "Edit", edit, map[string]any{"success": true}
	case "SessionStart":
		doc["source"] = "startup"
	case "Stop":
		doc["stop_hook_active"] = false
	}
	return json.Marshal(doc)
}
```

Die Nutzlasten der übrigen Ereignisse (`SubagentStart`, `SubagentStop`) tragen nur `hook_event_name` und `cwd`; der Test der Nutzlast in `TestBuildMakesOneCasePerEventInFixedOrder` deckt `PreToolUse`, ein zusätzlicher Test (`TestPayloadCarriesWhatEachEventNeeds`) je Ereignis: `SessionStart` hat `source`, `Stop` hat `stop_hook_active` gleich `false`, `PostToolUse` hat `tool_response`, `SubagentStop` hat weder `tool_name` noch `source`.

```go
// split cuts a command line into arguments the way a shell does for the
// forms hook configurations use: words separated by blanks, single and
// double quotes, and \" or \\ inside double quotes.
func split(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inWord := false
	var quote rune
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case quote != 0:
			switch {
			case r == quote:
				quote = 0
			case r == '\\' && quote == '"' && i+1 < len(rs) && (rs[i+1] == '"' || rs[i+1] == '\\'):
				i++
				cur.WriteRune(rs[i])
			default:
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote, inWord = r, true
		case r == ' ' || r == '\t':
			if inWord {
				args = append(args, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated %c quote in %q", quote, s)
	}
	if inWord {
		args = append(args, cur.String())
	}
	return args, nil
}
```

- [ ] **Step 4: grün.** Run: `go test ./internal/dev/benchcases -count=1`. Expected: PASS; Coverage 100 % je Funktion.
- [ ] **Step 5: Mutationsrunde** (kurze Suite): der Matcher-Anker `^(?:`/`)$` (Test: ein Matcher `Edit` darf nicht auf `NotebookEdit` passen — ergänze einen Fall dafür, falls kein Test ihn trennt), `len(steps) > 1` zu `>= 1`, die `toolEvent`-Bedingung, das `\\`-Zeichen in `split`. Kein Mutant überlebt unbegründet.
- [ ] **Step 6: Commit** `feat(bench): build the cases of a project's hooks from its settings`.

### Task 4: die Befehle `dev bench compare` und `dev bench cases`

**Files:**
- Create: `internal/cli/benchcompare.go`
- Modify: `internal/cli/dev.go` (`benchCommands`, `benchUsage`)
- Test: `internal/cli/benchcompare_test.go`

**Interfaces:**
- Consumes: `benchcompare.Compare`, `benchcompare.Markdown`, `benchcases.Build`, `benchreport.Report`, `benchTargets`, `benchWriteBoth`, `benchClock`, `benchreport.Stamp`.
- Produces: `devBenchCompare` und `devBenchCases`, beide `func(args []string, stdin io.Reader, stdout, stderr io.Writer) int`.

`dev bench compare --before <json> --after <json> --title <t> [--lang de|en] [--out <dir>]`: liest zwei `benchreport.Report`-Dateien (`--before` und `--after` sind Pflicht), ruft `Compare` und `Markdown`, druckt das Markdown; mit `--out` schreibt es `bench-<stamp>-compare.md` und `.json` (der JSON-Teil ist das eingerückte `Result`), vor der Ausgabe geprüft wie bei `bench hooks` (`benchTargets`). Exit 2 bei Aufruffehlern, 1 bei Lese-, Vergleichs- oder Schreibfehlern.

`dev bench cases --settings <settings.json> --root <dir> --file <markdown> --out <dir> [--extras <json>]`: `--settings` darf fehlen, wenn `--extras` gegeben ist (ein Ziel ohne `settings.json`, etwa `ecoflow`, hat nur Zusatzfälle); fehlen beide, Exit 2. Sonst liest der Befehl die Konfiguration, ruft `Build`, schreibt `cases.json` (eingerückt) und je Nutzlast `payload-<Ereignis>.json` in `--out` (der Ordner muss existieren; keine der Dateien darf schon liegen, alle oder keine). `--extras` ist eine JSON-Datei mit weiteren Fällen (`[]benchhooks.Case`); in ihrem Text ersetzt der Befehl `{{ROOT}}` durch `--root` und `{{OUT}}` durch `--out`, und hängt die Fälle an; ein Fallname, der schon vorkommt, ist ein Fehler.

- [ ] **Step 1: die Tests schreiben.** Für `compare`: zwei Berichtsdateien in `t.TempDir()` (mit `benchreport.Report{Schema: benchreport.Schema, Command: "hooks", Timings: …}.JSON()` erzeugt), Aufruf mit `--title "Beispielprojekt 1" --lang de`, erwartet Exit 0 und das Markdown mit `# Beispielprojekt 1`; ein Test für fehlendes `--before` (Exit 2); einer für eine Datei, die kein JSON ist (Exit 1); einer für `--out`, der prüft, dass beide Dateien entstehen und ein zweiter Lauf im selben Minute scheitert, ohne die ersten zu ändern (Vorbild: die Tests von `devBenchHooks`). Für `cases`: eine `settings.json` wie `oldSettings` aus Task 3 im Temp-Ordner, `--out` auf einen Unterordner; erwartet `cases.json` mit den drei Fällen und drei Nutzlastdateien; ein Test mit `--extras` (Fall mit `{{ROOT}}` im argv, wird ersetzt); einer, der einen doppelten Fallnamen zwischen Konfiguration und Extras verweigert (Exit 1) und danach **keine** Datei hinterlässt; einer für einen `--out`, der nicht existiert (Exit 1); einer ohne `--settings` mit `--extras` (nur die Zusatzfälle in `cases.json`, keine Nutzlastdateien außer denen, die die Zusatzfälle nennen) und einer ohne beides (Exit 2).
- [ ] **Step 2: rot laufen lassen** mit Stubs, die `0` geben.
- [ ] **Step 3: implementieren** nach dem Muster von `devBenchHooks` (`internal/cli/dev.go`): eigenes `flag.NewFlagSet`, `fs.SetOutput(stderr)`, Pflichtflags prüfen, Fehler als `loomux dev bench compare: …` auf stderr. In `dev.go`: `"compare": devBenchCompare, "cases": devBenchCases` in `benchCommands`, und in `benchUsage` die Zeilen `  compare set two hook runs side by side (faster, new, dropped)` und `  cases   build the case file of a project's hooks from its settings`, `<hooks|repos|search|compare|cases>` im Kopf.
- [ ] **Step 4: grün**, `go test ./internal/cli -run 'TestDevBench' -count=1`; Coverage 100 % je Funktion der neuen Datei.
- [ ] **Step 5: Mutationen von Hand per `-overlay`** (die Suite von `internal/cli` braucht ca. 90 s, über der festen Grenze von `dev mutants`: nur gezielte Tests, sequenziell): Pflichtflag-Prüfung, die Reihenfolge „erst Ziele prüfen, dann messen“, die Ersetzung von `{{ROOT}}`, die Prüfung auf doppelte Namen. Jeder Mutant muss einen Test rot machen.
- [ ] **Step 6: Doku** `docs/en/cli-reference.md` und `docs/de/cli-reference.md`: beide Befehle mit Flags, Ausgabe und Exit-Codes, gleiche Struktur wie `bench hooks`; `README.md` und `README.de.md` nur, wo sie die `bench`-Befehle aufzählen.
- [ ] **Step 7: Commit** `feat(cli): add `dev bench compare` and `dev bench cases``, Doku im selben oder einem eigenen Commit `docs(cli): document …`.

---

## Stück 2: das Umstellungsskript (Go + Shell)

### Task 5: `PruneHooks` — abgelöste Hook-Einträge entfernen

**Files:**
- Create: `internal/switchover/prune.go`
- Test: `internal/switchover/prune_test.go`

**Interfaces:**
- Produces: `func PruneHooks(settings []byte, needles []string) (out []byte, removed []string, kept []string, err error)`.

Regeln: In `hooks.<Ereignis>[]` (Gruppen) fällt eine Gruppe weg, wenn **jeder** ihrer Befehle mindestens eines der `needles` als Teilstring enthält (eine Gruppe ohne Befehle fällt nicht weg). Eine Gruppe, in der manche Befehle passen und andere nicht, **bleibt** und wird in `kept` gemeldet (`"<Ereignis>: <erster passender Befehl>"`). `removed` nennt je entfernte Gruppe `"<Ereignis>: <Befehle, mit „; “ getrennt>"`. Ein Ereignis, dessen Liste dabei leer wird, verschwindet ganz. **Alles außerhalb des `hooks`-Objekts bleibt Byte für Byte, die Reihenfolge der Ereignisse bleibt;** wird nichts entfernt, ist `out` gleich `settings`. Ohne `hooks` oder mit leeren `needles`: `out` gleich `settings`, keine Fehler. Kein gültiges JSON: Fehler.

- [ ] **Step 1: die Tests schreiben**

```go
package switchover

import (
	"reflect"
	"strings"
	"testing"
)

const settings = `{
  "permissions": {"allow": ["Bash(git status:*)"]},
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "uv run ultraloom hook session-start"}], "ultraLoomOwned": true}
    ],
    "PreToolUse": [
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "ulguard --root x"}], "ultraLoomOwned": true},
      {"matcher": "Edit", "hooks": [{"type": "command", "command": "loomux hook pre-tool-use --host claude"}]}
    ],
    "Stop": [
      {"hooks": [{"type": "command", "command": "brain wiki-gate"}, {"type": "command", "command": "my-own-check"}]}
    ]
  },
  "enabledPlugins": {"pyright-lsp@claude-plugins-official": true}
}`

var needles = []string{"ulguard", "brain wiki-gate", "ultraloom hook"}

func TestPruneHooksDropsWholeOldGroupsAndKeepsTheRest(t *testing.T) {
	out, removed, kept, err := PruneHooks([]byte(settings), needles)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if strings.Contains(text, "ulguard") || strings.Contains(text, "ultraloom hook") {
		t.Errorf("an old group is still there:\n%s", text)
	}
	if !strings.Contains(text, "loomux hook pre-tool-use") {
		t.Errorf("the new hook is gone:\n%s", text)
	}
	wantRemoved := []string{"SessionStart: uv run ultraloom hook session-start", "PreToolUse: ulguard --root x"}
	if !reflect.DeepEqual(removed, wantRemoved) {
		t.Errorf("removed = %q, want %q", removed, wantRemoved)
	}
	// The Stop group mixes an old command with the user's own: it stays.
	if !strings.Contains(text, "my-own-check") || len(kept) != 1 || !strings.HasPrefix(kept[0], "Stop: brain wiki-gate") {
		t.Errorf("mixed group: kept = %q, text has own check = %v", kept, strings.Contains(text, "my-own-check"))
	}
	// SessionStart lost its only group: the event goes with it.
	if strings.Contains(text, `"SessionStart"`) {
		t.Errorf("an empty event is still there:\n%s", text)
	}
}

func TestPruneHooksLeavesEverythingOutsideHooksByteForByte(t *testing.T) {
	out, _, _, err := PruneHooks([]byte(settings), needles)
	if err != nil {
		t.Fatal(err)
	}
	head := settings[:strings.Index(settings, `"hooks"`)]
	tail := settings[strings.LastIndex(settings, `"enabledPlugins"`):]
	if !strings.HasPrefix(string(out), head) || !strings.HasSuffix(string(out), tail) {
		t.Errorf("the bytes around hooks changed:\n%s", out)
	}
	// Events keep their order.
	if strings.Index(string(out), `"PreToolUse"`) > strings.Index(string(out), `"Stop"`) {
		t.Errorf("event order changed:\n%s", out)
	}
}

func TestPruneHooksChangesNothingWhenNothingMatches(t *testing.T) {
	for name, n := range map[string][]string{"no needles": nil, "no match": {"zzz"}} {
		out, removed, kept, err := PruneHooks([]byte(settings), n)
		if err != nil || string(out) != settings || len(removed) != 0 || len(kept) != 0 {
			t.Errorf("%s: err %v, changed %v, removed %q, kept %q", name, err, string(out) != settings, removed, kept)
		}
	}
	plain := `{"permissions": {}}`
	if out, _, _, err := PruneHooks([]byte(plain), needles); err != nil || string(out) != plain {
		t.Errorf("a file without hooks changed: %q, %v", out, err)
	}
}

func TestPruneHooksRefusesWhatIsNotJSON(t *testing.T) {
	if _, _, _, err := PruneHooks([]byte(`{`), needles); err == nil {
		t.Errorf("no error for broken JSON")
	}
	if _, _, _, err := PruneHooks([]byte(`{"hooks": []}`), needles); err == nil {
		t.Errorf("no error for hooks that is not an object")
	}
}
```

- [ ] **Step 2: rot laufen lassen** mit einem Stub, der `settings, nil, nil, nil` gibt.
- [ ] **Step 3: implementieren.** Vorgehen (Byte-Erhalt außerhalb von `hooks`): mit `json.Decoder` die Schlüssel der obersten Ebene lesen; beim Schlüssel `hooks` den Wert per `dec.Decode(&raw)` (ein `json.RawMessage` mit den genauen Bytes) holen und aus `dec.InputOffset()` und `len(raw)` seinen Anfang und sein Ende in `settings` berechnen. Das `hooks`-Objekt in seine Ereignisse zerlegen, **in Dateireihenfolge** (die Schlüssel des Objekts mit einem zweiten Decoder in Reihenfolge lesen, die Werte als `map[string][]json.RawMessage` halten); jede Gruppe (`json.RawMessage`) wird nur zur Entscheidung entpackt (`struct{ Hooks []struct{ Command string `json:"command"` } `json:"hooks"` }`), die **Rohbytes der behaltenen Gruppen bleiben unverändert**. Danach den neuen `hooks`-Wert aus den behaltenen Rohbytes zusammensetzen (`{`, `"Ereignis": [` Gruppen, durch `,` getrennt, `]`, `}`) und mit `json.Indent(&buf, compact, "  ", "  ")` einrücken (der Wert steht in der Datei eine Ebene tief); `out` = Bytes vor dem Wert + neuer Wert + Bytes nach dem Wert. Wird nichts entfernt, `settings` unverändert zurückgeben (kein Umformatieren). `needles` leer: sofort zurück.
- [ ] **Step 4: grün**, 100 % Coverage je Funktion.
- [ ] **Step 5: Mutationen per `-overlay`/`dev mutants`** (kurze Suite): `jeder Befehl` zu `irgendein Befehl` (Test: die gemischte Gruppe), die Regel „leeres Ereignis verschwindet“, die Byte-Erhaltung (statt `settings[:start]` neu formatieren).
- [ ] **Step 6: Commit** `feat(switchover): remove the hook entries a switch-over replaces`.

### Task 6: die Vorlage `apply.sh` und ihre Befehle

**Files:**
- Create: `internal/switchover/apply.sh.tmpl`, `internal/switchover/render.go`, `internal/cli/switchover.go`
- Modify: `internal/cli/dev.go` (die Tabelle der `dev`-Untergruppen: dort steht `bench`; dazu `switchover` mit `render` und `prune-hooks`)
- Test: `internal/switchover/render_test.go`, `internal/cli/switchover_test.go`

**Interfaces:**
- Consumes: `PruneHooks` (Task 5).
- Produces:
  - `type Params struct { Name, Project, Loomux, ConfigNew, Registry, RegistryNew, RegistrySum, WikiDst, Vault, VaultOld, StateArea string; WikiSrcs, OldFiles, OldHooks []string }` (JSON-Schlüssel in `snake_case`)
  - `func Render(p Params) (string, error)` — Pflicht: `Name`, `Project`, `Loomux`, `ConfigNew`; gehört `RegistryNew` gesetzt, sind `Registry` und `RegistrySum` Pflicht; gehört `WikiSrcs` (eine oder mehrere Ordner, in der Reihenfolge, in der sie zusammengelegt werden) gesetzt, ist `WikiDst` Pflicht; gehört `VaultOld` gesetzt, ist `Vault` Pflicht
  - `loomux dev switchover render --params <json> --out <file>` schreibt das Skript (verweigert eine vorhandene Datei) und Exit 0; `loomux dev switchover prune-hooks --file <settings.json> --match <s> [--match <s>…]` schreibt die Datei (nur wenn sich etwas ändert) und listet die entfernten und behaltenen Gruppen

Das Skript (POSIX-`sh`, Git Bash; Werte werden von `Render` in einfache Anführungszeichen gesetzt, ein `'` darin als `'\''`; Token `@NAME@` in der Vorlage):

```sh
#!/bin/sh
# apply.sh for @NAME@: written by the agent, run by the human.
#   sh apply.sh --check   shows what would happen and writes nothing
#   sh apply.sh           does it; running it again changes nothing
set -eu

PROJECT=@PROJECT@
LOOMUX=@LOOMUX@
CONFIG_NEW=@CONFIG_NEW@
REGISTRY=@REGISTRY@
REGISTRY_NEW=@REGISTRY_NEW@
REGISTRY_SUM=@REGISTRY_SUM@
WIKI_SRCS=@WIKI_SRCS@
WIKI_DST=@WIKI_DST@
VAULT=@VAULT@
VAULT_OLD=@VAULT_OLD@
STATE_AREA=@STATE_AREA@
OLD_FILES=@OLD_FILES@
OLD_HOOKS=@OLD_HOOKS@

CHECK=0
[ "${1:-}" = "--check" ] && CHECK=1

say() { printf '%s\n' "$*"; }
die() { printf 'abort: %s\n' "$*" >&2; exit 3; }
run() { if [ "$CHECK" = 1 ]; then say "would run: $*"; else "$@"; fi; }
clean() { [ -z "$(git -C "$1" status --porcelain)" ]; }
tree_sum() { (cd "$1" && find . -type f | LC_ALL=C sort | while IFS= read -r f; do sha256sum "$f"; done); }

# 1. Nothing is written before these hold.
clean "$PROJECT" || die "$PROJECT has uncommitted changes"
if [ -n "$VAULT_OLD" ]; then
  clean "$VAULT" || die "$VAULT has uncommitted changes"
  git -C "$VAULT" rev-parse --verify HEAD >/dev/null 2>&1 || die "$VAULT has no commit"
  [ -n "$(git -C "$VAULT" remote)" ] || die "$VAULT has no remote"
fi

# 2. The registry, only if it is still the one this script was prepared against.
if [ -n "$REGISTRY_NEW" ]; then
  if cmp -s "$REGISTRY" "$REGISTRY_NEW"; then
    say "registry: already replaced"
  elif [ "$(sha256sum "$REGISTRY" | cut -d' ' -f1)" = "$REGISTRY_SUM" ]; then
    run cp "$REGISTRY" "$REGISTRY.bak"
    run cp "$REGISTRY_NEW" "$REGISTRY"
  else
    die "the registry changed since this script was prepared"
  fi
fi

# 3. The wiki. Every source is laid over one staging directory in the order
#    given (a later one wins); the destination only gains the files it lacks.
#    A file that is there and differs stops the script: a human decides.
MOVED=0
if [ -n "$WIKI_SRCS" ]; then
  have=0
  OLD_IFS=$IFS; IFS='|'
  for s in $WIKI_SRCS; do [ -e "$s" ] && have=1; done
  IFS=$OLD_IFS
  if [ "$have" = 0 ]; then
    [ -e "$WIKI_DST" ] || die "no source of the wiki is left and $WIKI_DST does not exist"
    say "wiki: already moved"
  else
    STAGING="$WIKI_DST.staging"
    run rm -rf "$STAGING"
    run mkdir -p "$STAGING"
    OLD_IFS=$IFS; IFS='|'
    for s in $WIKI_SRCS; do [ -e "$s" ] && run cp -R "$s/." "$STAGING/"; done
    IFS=$OLD_IFS
    if [ "$CHECK" = 0 ]; then
      (cd "$STAGING" && find . -type f | LC_ALL=C sort | while IFS= read -r f; do
        if [ -f "$WIKI_DST/$f" ]; then cmp -s "$f" "$WIKI_DST/$f" || { printf 'differs: %s\n' "$f" >&2; exit 1; }; fi
      done) || die "a file of the wiki differs from the one at $WIKI_DST"
      (cd "$STAGING" && find . -type f | LC_ALL=C sort | while IFS= read -r f; do
        [ -e "$WIKI_DST/$f" ] || { mkdir -p "$(dirname "$WIKI_DST/$f")"; cp "$f" "$WIKI_DST/$f"; }
      done)
      rm -rf "$STAGING"
      MOVED=1
    fi
  fi
fi

# 4. The configuration, never over an existing one.
if [ ! -e "$PROJECT/.loomux/config.toml" ]; then
  run mkdir -p "$PROJECT/.loomux"
  run cp "$CONFIG_NEW" "$PROJECT/.loomux/config.toml"
else
  say "config: $PROJECT/.loomux/config.toml exists, kept"
fi

# 5. Hooks, guards and the area, as init does them.
run "$LOOMUX" init --yes --root "$PROJECT"

# 6. What init leaves standing: the old hook entries, then the old files.
if [ -n "$OLD_HOOKS" ] && [ -f "$PROJECT/.claude/settings.json" ]; then
  set --
  OLD_IFS=$IFS; IFS='|'
  for m in $OLD_HOOKS; do set -- "$@" --match "$m"; done
  IFS=$OLD_IFS
  run "$LOOMUX" dev switchover prune-hooks --file "$PROJECT/.claude/settings.json" "$@"
fi
for f in $OLD_FILES; do
  [ -e "$PROJECT/$f" ] && run rm -rf "$PROJECT/$f"
done

# 7. Only after the copy was proven: the old wiki leaves the vault, the old
#    state directory is renamed, not deleted.
if [ "$MOVED" = 1 ] && [ -n "$VAULT_OLD" ] && [ -e "$VAULT/$VAULT_OLD" ]; then
  run git -C "$VAULT" rm -r -q -- "$VAULT_OLD"
  run git -C "$VAULT" commit -q -m "chore: move the wiki $VAULT_OLD into its project"
fi
if [ -n "$STATE_AREA" ] && [ -e "$STATE_AREA" ] && [ ! -e "$STATE_AREA.alt" ]; then
  run mv "$STATE_AREA" "$STATE_AREA.alt"
fi
say "done: $PROJECT"
```

- [ ] **Step 1: die Tests schreiben.** `render_test.go`: ein Pflichtfeld fehlt je Fall (Fehler nennt das Feld); ein Wert mit `'` wird als `'\''` gesetzt (Teststring `it's`); kein `@X@`-Token bleibt im Ergebnis; `sh -n` (wenn `sh` im Pfad, sonst `t.Skip`) akzeptiert das Ergebnis. Der **Weltentest** (überspringt sich ohne `sh` und `git`; Git-Bash-Syntax; alle Pfade mit `filepath.ToSlash`) stellt die **echte Topologie** nach: in `t.TempDir()` ein Projekt (`git init`, ein Commit, Benutzername und -adresse per `-c`), das **schon ein Wiki** hat (`docs/wiki` mit `seite.md` und einer `index.md`), ein Vault (`git init`, ein Commit, `git remote add origin <bare>`), darin der Ordner `91 Projekte/x` (`VaultOld`) mit `_schema.md`, `audit.md` und einer `index.md` **gleichen** Inhalts wie die des Projekts; `WikiSrcs` ist genau dieser Vault-Ordner, `WikiDst` das `docs/wiki` des Projekts. Dazu eine Registry-Datei und eine neue, eine `config.toml.new`, eine im Projekt **committete** `settings.json` (Muster wie `settings` in Task 5), ein Attrappen-`loomux`, das **sein `argv` in eine Datei schreibt** und mit 0 endet, und `OldFiles: [".ultraloom"]` (Ordner im Projekt). Erwartungen: (a) `sh apply.sh --check`: Exit 0, `would run:` in der Ausgabe, Schnappschuss (Pfad, Größe) aller Dateien vor und nach gleich; (b) mit einem ungesicherten Projekt (`echo x >> …`, nicht committet): Exit 3, `abort:` auf stderr, Schnappschuss gleich; (c) ohne Remote im Vault: Exit 3; (d) Registry verändert: Exit 3 mit „registry changed“, die Datei bleibt; (e) der volle Lauf: Exit 0, `.loomux/config.toml` liegt im Projekt, `docs/wiki` hat `seite.md`, `index.md` **und die neuen** `_schema.md` und `audit.md`, `.ultraloom` ist weg, `91 Projekte/x` ist aus dem Vault entfernt und der Vault hat einen neuen Commit, und die Datei des Attrappen-`loomux` zeigt die Aufrufe `init --yes --root <projekt>` und `dev switchover prune-hooks --file <projekt>/.claude/settings.json --match ulguard …` mit den erwarteten `--match`-Werten; (f) ein **zweiter** Lauf, nachdem (e) das Projekt committet wurde (`git add -A && git commit` im Test), endet mit Exit 0, sagt `wiki: already moved` und ändert nichts (Schnappschuss gleich, kein neuer Commit im Vault); (g) eine `index.md` im Vault-Ordner, die von der des Projekts **abweicht**: Exit 3, `differs: ./index.md` auf stderr, das Projekt und der Vault unverändert (auch kein `.staging`-Ordner bleibt liegen); (h) zwei Quellen, die dieselbe Datei mit verschiedenem Inhalt tragen: die spätere gewinnt in `docs/wiki`.
  `switchover_test.go`: `render` mit fehlender Parameterdatei (Exit 2), ungültigem JSON (Exit 1), vorhandener `--out` (Exit 1, Datei unverändert), Erfolg (Exit 0, Datei beginnt mit `#!/bin/sh`); `prune-hooks` ohne `--match` (Exit 2), auf eine Datei ohne Treffer (Exit 0, Datei unverändert, Datei-Änderungszeit gleich), auf die `settings`-Konstante (Exit 0, entfernte Gruppen gelistet, die Datei geschrieben).
- [ ] **Step 2: rot laufen lassen** mit Stubs (`Render` gibt `""`, die Befehle `0`).
- [ ] **Step 3: implementieren.** `render.go` lädt die Vorlage beim ersten Gebrauch (`//go:embed apply.sh.tmpl` in einer `string`-Variablen ist erlaubt, weil es nichts parst; die Ersetzung geschieht in `Render`), prüft die Pflichtfelder, setzt die Werte mit `quote(s string) string` (`'` + `strings.ReplaceAll(s, "'", `'\''`)` + `'`; eine leere Zeichenkette wird `''`), verbindet `OldFiles` mit Leerzeichen und `OldHooks` mit `|`, ersetzt jeden `@X@` und prüft, dass keiner übrig bleibt (`regexp.MustCompile("@[A-Z_]+@")` innerhalb der Funktion kompilieren, nicht als Paketvariable). `internal/cli/switchover.go` folgt dem Muster von `devBenchHooks`. In `dev.go` die Untergruppe `switchover` anmelden.
- [ ] **Step 4: grün**, 100 % Coverage je Funktion der Go-Dateien.
- [ ] **Step 5: Mutationen per `-overlay`:** eine Pflichtfeldprüfung, das Anführungszeichen-Ersetzen, die Prüfung „kein Token übrig“, und **am Skript** (Kopie im Scratchpad, in den Weltentest als Vorlage eingespeist): `clean "$PROJECT" ||` weglassen (Test b rot), den `cmp`-Vergleich in Schritt 3 weglassen (der Test (g) wird rot), den Existenzwächter in Schritt 4 weglassen (Test f rot), `MOVED=1` immer setzen (ein Test mit `--check`, der das Löschen im Vault verlangt, wäre rot).
- [ ] **Step 6: Doku** `docs/en/cli-reference.md` und `docs/de/cli-reference.md` (`dev switchover render`, `prune-hooks`), `README.md`/`README.de.md` wo sie `dev` aufzählen.
- [ ] **Step 7: Commit** `feat(switchover): add the apply script and the hook pruner` und `docs(cli): document the switch-over commands`.

---

## Stück 3: Durchführung (Abläufe)

Jede Aufgabe hier hat einen Ablageort `.superpowers/switchover/<projekt>/` und endet mit einer Zeile im Ledger. Der Agent liest und bereitet vor; der Mensch führt aus.

### Task 7: die drei offenen Messungen vor der Baseline (nur lesend)

- [ ] **Step 1 (`init --dry-run` zeigt die ganze `settings.json`?):** auf `ecoflow`, das keine `settings.json` hat, und auf `iam_backend`, das eine hat, je `./bin/loomux.exe init --dry-run --yes --root "<pfad>"` als schlichte Einzelzeile. Festhalten: nennt die Ausgabe die ganze Ziel-Datei oder nur Änderungen? Das Ergebnis entscheidet, ob Schritt 6 des Skripts (Entfernen aus der lebenden Datei) genügt (Spec, offener Punkt). In die Akte `docs/.superpowers/parity/stufe-4e.md`, Abschnitt „Messungen“, mit Datum.
- [ ] **Step 2 (welche Module und welcher Guard-Modus setzt `init --yes`?):** aus dem Code (`internal/setup`, `internal/cli/init.go`) und dem Trockenlauf: Hooks, Brain, Graph an? `[guard] mode`? Erwartet ist `default`; weicht es ab, steht das im Ledger als Abweichung von der Vorgabe des Nutzers.
- [ ] **Step 3 (kann `area add` einen vorhandenen Bereich ändern?):** aus `internal/cli/area.go` (`planArea`): ändert es `readonly` oder `wiki` eines vorhandenen Scopes? Das Ergebnis entscheidet, ob die Registry-Datei nötig ist (Spec, offener Punkt).
- [ ] **Step 4 (die Session-Hooks der `iam_*`):** `uv` und das vendorte Python (`<projekt>/.ultraloom/vendor/ultraloom`) laufen? `uv --version` und `uv run --project "<projekt>/.ultraloom/vendor/ultraloom" ultraloom --help` je als Einzelzeile. Läuft es nicht, ist der Fall der Baseline „nicht messbar“ und steht so im Bericht.
- [ ] **Step 4a (Inhalt und Überschneidung je Ziel, nur lesend):** Die erste Fassung der Spec ging davon aus, dass die Wikis im Vault liegen; gemessen am 2026-09-29 liegen sie bei `space` schon im Projekt (`docs/wiki`, 204 Markdown-Dateien). Je Ziel (`ecoflow`, `space`, `iam_wiki`, `ultra-brain`, `ultraloom`, `brain-knowledge`) die Dateizahl (`find … -type f | wc -l`, dazu Markdown) an vier Orten: im Vault `91 Projekte/<name>`, im Zustandsverzeichnis `%LOCALAPPDATA%/loomux/areas/project-<name>`, im Projekt unter `docs/wiki` und unter `wiki`. Je Paar von Orten die Dateinamen, die auf beiden Seiten vorkommen, und ob ihr Inhalt gleich ist (`cmp -s`). Das Ergebnis ist eine Tabelle in der Akte und legt je Ziel fest: `WikiDst` (der Ort im Projekt), `WikiSrcs` (nur Ordner, deren Dateien im Ziel fehlen; **nicht** der Bestand des Zustandsverzeichnisses als Ganzes, der auch `.agents`, `.ultraloom` und `docs` des Projekts kopiert trägt), und die Liste der Dateien gleichen Namens mit abweichendem Inhalt, die der Nutzer je Datei entscheidet (behalten oder ersetzen). Solange die Tabelle fehlt, wird kein `apply.sh` für ein Ziel mit Wiki gerendert.
- [ ] **Step 4b (die Git-Hooks des Vaults):** gemessen am 2026-09-29: `core.hooksPath` ist dort nicht gesetzt, ein Ordner `.githooks` fehlt und in `.git/hooks` liegen nur Beispiele; der Commit des Skripts (`chore: move the wiki …`) trifft also keinen `commit-msg`. Nach einem `init` im Vault (Task 11) ist das neu zu prüfen, bevor ein späteres Skript dort committet.
- [ ] **Step 5: Commit der Akte** `docs(migration): record what the switch-over still had to measure`.

### Task 8: die Baseline aller Ziele (Phase 0, nur lesend, an einem Tag, vor dem ersten `apply.sh`)

Ziele: `ecoflow`, `space`, `ultra-brain`, `ultraloom`, `iam_backend`, `iam_frontend`, `iam_workers`, `iam_wiki`, `brain-knowledge`.

- [ ] **Step 1 (je Ziel): Fälle bauen.** `./bin/loomux.exe dev bench cases --settings "<projekt>/.claude/settings.json" --root "<projekt>" --file "<projekt>/README.md" --out ".superpowers/switchover/<projekt>/baseline" [--extras <datei>]` als schlichte Einzelzeile (den Ordner vorher mit Write anlegen: eine Datei `.gitkeep`). Hat ein Ziel keine `settings.json` (`ecoflow`), gibt es nur die Zusatzfälle. Die **Zusatzfälle** (`extras-old.json`, je Ziel, mit `{{ROOT}}`): jedes Werkzeug, das der Alt-Stand hat und das kein Hook-Ereignis ist, mit einem Namen, der auf der Neu-Seite **gleich** lautet: `commit-msg` (`.githooks/commit-msg` oder der alte Aufruf), `pre-commit-gate`, `search (fast)`, `status`, `reindex`, `reconcile`, `wiki lint`, `graph query` — nur was es dort gibt; die Argumente lesen wir aus `<werkzeug> --help` und aus den Fällen in `testdata/bench/1b-1-brain.json` (Neu-Seite), nicht aus dem Gedächtnis. Was die Neu-Seite später zusätzlich kann (Graph, Blast, `area check`, `convert`, `fetch`, `config`, `init --dry-run`), steht nur in `extras-new.json` und erscheint im Vergleich als „neu“; was der Alt-Stand hatte und das Neue nicht mehr braucht (`hooks.tsv`, `[relevance]`, der Python-Einstieg), steht nur in `extras-old.json` und erscheint als „weggefallen“. Das Inventar der alten Einträge prüft `./bin/loomux.exe init --detect-only --root "<projekt>"` (Einzelzeile) gegen die Ablösetabelle in `internal/hooks/status.go`.
- [ ] **Step 2 (je Ziel): messen.** `./bin/loomux.exe dev bench hooks ".superpowers/switchover/<projekt>/baseline/cases.json" -n 5 --out ".superpowers/switchover/<projekt>/baseline"` (Einzelzeile). Das erste Ergebnis eines Falls ist der Kaltlauf. Vor jedem Ziel den qmd-Dienst auf denselben Zustand bringen (läuft der Dienst, läuft er bei Baseline und Nachher; steht in der Akte).
- [ ] **Step 3: Kontrolle.** Kein Ausgang wird verworfen: ein Fall mit Exit ≠ 0 bleibt in der Tabelle (Spalte `exit codes`) und wird im Bericht erklärt (der alte Wächter gibt auf einem ihm unbekannten Repo Exit 2). Ein Fall, dessen Programm nicht startet, bricht die Messung ab: dann fehlt das Werkzeug; das Ziel bekommt in der Akte den Vermerk „nicht messbar: <Werkzeug>“ und **keinen** erfundenen Wert.
- [ ] **Step 4: Ledger.** Je Ziel eine Zeile mit Pfad der Baseline-JSON, Werkzeugversionen und Zeitpunkt.

### Task 9: Pilot 1 `ecoflow` (der saubere Pfad)

- [ ] **Step 1: Diagnose (nur lesend).** `area check "<ecoflow>"`, `init --dry-run --root`, Inhalt und Überschneidung nach Task 7 Schritt 4a (im Vault liegt nur ein Gerüst aus fünf Dateien), Git-Zustand beider Repos.
- [ ] **Step 2: vorbereiten** in `.superpowers/switchover/ecoflow/`: `config.toml.new` (die Übersetzung der `.brain.toml`: `[area] scope`, `[layout]`, `[index]`, `[privacy]`, `[maintenance]` mit `on_merge = true`, `[model]` nach dem Hinweis von `area check`; `readonly` und `wiki = true` entfallen; dazu `[modules]` so, dass Hooks, Brain und Graph an sind, und `[guard] mode = "default"`), `registry.toml.new` (aus der **lebenden** Registry erzeugt: `project/ecoflow` mit `wiki = "<ecoflow>/docs/wiki"`, sonst unverändert) und seine SHA-256 (`sha256sum`), `params.json` für `dev switchover render`, dann das Skript mit `./bin/loomux.exe dev switchover render --params ".superpowers/switchover/ecoflow/params.json" --out ".superpowers/switchover/ecoflow/apply.sh"`.
- [ ] **Step 3: der Mensch führt aus.** Der Agent nennt genau eine Zeile: erst `! sh "…/apply.sh" --check`, danach `! sh "…/apply.sh"`. Wartet.
- [ ] **Step 4: Prüfung (nur lesend).** `area check "<ecoflow>"` Exit 0; `status`/`doctor`; ein erlaubter Edit und ein verweigerter Edit als Nutzlast durch `./bin/loomux.exe hook pre-tool-use --host claude --root "<ecoflow>" < <payload>`; Wiki-Dateizahl und Prüfsummen gegen die Quelle; `git -C "<vault>" log -1` (der Löschcommit); ein zweiter Lauf `apply.sh --check` zeigt nur „already“/„kept“-Zeilen.
- [ ] **Step 5: Phase 2** (Task 8 Schritt 1 und 2 mit der neuen `settings.json`, Ausgabe in `after/`), dann `./bin/loomux.exe dev bench compare --before <baseline-json> --after <after-json> --title "ecoflow" --lang de --out ".superpowers/switchover/ecoflow"` (der Befehl legt neben dem Markdown auch die JSON-Datei ab; sie gehört nicht ins Repo), dann das Markdown nach `docs/.superpowers/parity/bench-4e-ecoflow.md` kopieren.
- [ ] **Step 6: Was `ecoflow` nicht beweist,** in die Akte: Entfernen alter Hooks, Bereinigung der `settings.json`, Alt/Neu-Vergleich der Hooks.
- [ ] **Step 7: Ledger.**

### Task 10: Pilot 2 `space` (der volle Durchgang)

Wie Task 9, mit den Unterschieden aus der Spec: alle drei Altdateien (`.ultraloom/config.toml`, `.claude/settings.json`, `.brain.toml`), `readonly` fällt aus der Registry, `workspace` bleibt; die Seiten liegen **schon im Projekt** (`docs/wiki`, das ist `WikiDst`); `WikiSrcs` sind nach Task 7 Schritt 4a die Ordner, deren Dateien dort fehlen (das Gerüst im Vault `91 Projekte/space` und, was die Messung als Artefakt des Wikis ausweist, aus dem Zustandsverzeichnis — nicht dessen Kopie von `.agents`, `.ultraloom` und `docs`); Dateien gleichen Namens mit abweichendem Inhalt entscheidet der Nutzer je Datei; `VaultOld` ist `91 Projekte/space`; `StateArea` ist `%LOCALAPPDATA%\loomux\areas\project-space` (wird `.alt`); `OldHooks`: `ulguard`, `brain guard`, `brain wiki-gate`, `ultraloom hook`; `OldFiles`: `.ultraloom`, `.brain.toml`. Zusätzlich nach dem Umzug ein `reindex` des Bereichs `hub` (`91 Projekte/.brain.toml` indiziert `**/*.md`; Löschungen sind erwartet) und die Baseline-Kontrolle, dass der Alt-Vergleich der Hooks jetzt echte Werte hat (der Hook-Vergleich `PreToolUse` alt gegen neu).

- [ ] **Steps 1–7** wie Task 9, mit diesen Unterschieden; danach im Ledger: was `space` gelehrt hat (Punkte, die das Skript ändern muss). **Änderungen an Vorlage oder Befehlen sind eigene Tasks mit Test** (Stück 2), bevor die Welle beginnt.

### Task 11: die Welle

**Wartet (Entscheidung des Nutzers, 2026-09-30):** Die Welle beginnt erst, wenn dieser Zweig und danach die Schonfrist je Lane (`specs/2026-09-30-loomux-lane-probation-design.md`) gemergt, veröffentlicht und installiert sind. Je Ziel gelten außerdem die Entscheidungen der Spec vom 2026-09-29 und 2026-09-30 vor dem Text unten: `WikiSrcs` überall leer, kein `VaultOld`, `iam_wiki` mit dem Wiki im Wurzelverzeichnis, `iam_backend`/`iam_frontend`/`iam_workers` mit `init_args = ["--brain=none"]`.

Reihenfolge: `ultra-brain`, `ultraloom`, `iam_backend`, `iam_frontend`, `iam_workers` (ohne Vault-Wiki), dann `iam_wiki` (mit Vault-Wiki, selbst ein Wiki: `docs/wiki` oder die Wurzel entscheidet der Nutzer nach dem Ergebnis der Diagnose), zuletzt `brain-knowledge`.

- [ ] **Je Ziel** die Schritte von Task 9. **Unterschiede:** `iam_backend`/`iam_frontend`/`iam_workers` haben kein Wiki: `WikiSrcs` leer, `init` legt das Bündel an und registriert den Bereich (`RegistryNew` leer); der Agent darf dort nicht schreiben, alles steht im Skript; die alten Hooks (`ulguard`, `brain wiki-gate`, `uv run … ultraloom hook …`) und `.ultraloom/` samt dem vendorten Python entfallen. `ultra-brain` und `ultraloom` haben ihr Wiki schon im Projekt (`docs/wiki`): kein Wiki-Umzug, nur Übersetzung, `init`, Bereinigung. `brain-knowledge` ist der Vault mit eigener Form: `init` (Hooks, Guards), die Übersetzung seiner vier Manifeste (Wurzel, `92 Engineering/python`, `92 Engineering/craft`, `91 Projekte`) in vier `.loomux/config.toml`, kein Wiki-Umzug; ein `apply.sh` je Manifest ist unnötig, ein Skript mit mehreren `cp`-Schritten reicht.
- [ ] **Nach dem letzten Ziel:** der Mensch führt einmal `! ./bin/loomux.exe merge-hook install` aus (Abschlussskript; wirkt auf die ganze Registry). Danach `merge-hook status` und je Ziel ein Test: ein Merge in einem Wegwerfzweig legt ein Ereignis ab.
- [ ] **Ledger** je Ziel; ein Ziel, dessen Baseline oder Nachmessung fehlt, ist offen und steht so im Bericht.

### Task 12: die Berichte und die Doku

- [ ] **Step 1: je Ziel** `bench-4e-<projekt>.md` (Task 9 Schritt 5) mit echtem Namen, deutsch, unter `docs/.superpowers/parity/`; oben Datum, Werkzeugversionen (`loomux --version`, `ulguard`, `brain`), Umgebung, die Ausgabe von `benchHead`.
- [ ] **Step 2: der anonymisierte Bericht.** Für alle neun Ziele `dev bench compare --title "Beispielprojekt N" --lang de` (Nummern in der Reihenfolge der Welle, die Zuordnung Name zu Nummer steht **nur** in den Arbeitspapieren der Ledger-Datei `docs/.superpowers/parity/stufe-4e.md`), und dasselbe mit `--lang en`. Aus den neun Ausgaben entsteht in `docs/en/benchmarks.md` und `docs/de/benchmarks.md` ein Abschnitt mit Datum und Uhrzeit, Beschreibung der Methode (ein Kaltlauf, fünf Warmläufe, Mittelwert, Umgebung), je einer Tabelle je Beispielprojekt und einer Zusammenfassung: was wie viel schneller ist, was hinzukam, was wegfiel; Projektnamen, Pfade und Dateizahlen stehen nicht darin (Dateizahlen sind Quasi-Kennungen, der Nutzer nimmt das in Kauf; sie stehen trotzdem nicht in der Doku).
- [ ] **Step 3: Migrationsplan und READMEs.** `docs/en/migration.md` und `docs/de/migration.md` (die Zeile 4e: Stand nachführen; Status und Priorität ändern sich nur, wenn die Spec es sagt; `internal/plancheck` hält sie), Roadmap und beide READMEs.
- [ ] **Step 4: Mutationsrunde und Akte.** Die Runde der neuen Pakete (`benchcompare`, `benchcases`, `switchover`; Suiten kurz, `loomux dev mutants` ist brauchbar) und die Handrunde der neuen CLI-Funktionen (Task 4 Schritt 5, Task 6 Schritt 5): Abschnitt „## Überlebende Mutanten“ in `docs/.superpowers/parity/stufe-4e.md` fortschreiben.
- [ ] **Step 5: der Pull Request** mit dem Skill `release-pr` (Label `release:minor`: neue `dev`-Befehle; Changelog-Block nennt `dev bench compare`, `dev bench cases`, `dev switchover`), `parse-body` über **alle** Commit-Nachrichten des Zweigs. Der Mensch pusht.

---

## Stück 4: der Aufräum-PR (nach der Welle, eigener Zweig von `master`)

Wie im Plan `2026-09-28-loomux-stufe-4e.md`, Stück C (Tasks 6–10), mit diesen Änderungen aus der Spec vom 2026-09-29:

- Die Aside-Auflösung in `ResolvedAreaDir` (Task 6) und die Leser, die daran vorbeigehen (Task 7), **bleiben**, weil `#Obsidian/AI` als read-only-Bereich bestehen bleibt; der Testfall ist dieser eine Bereich (`project/obsidian-ai`).
- Der Bereich `hub` und sein Wiki-Pfad bleiben wie sie sind (die Überschneidung mit `obsidian-ai` besteht weiter).
- Das Entfernen der Rückfälle (`LegacyBrainDirUntilStage3`, `ReadAreaManifestUntilStage4`, alte Manifestnamen) beginnt erst, wenn `area check` für **jedes** umgestellte Ziel Exit 0 gab und `#Obsidian/AI` seine Deklaration unter `.loomux/config.toml` im Zustandsverzeichnis hat oder ausdrücklich beim Rückfall bleibt (Entscheidung des Nutzers, im Ledger).

---

## Self-Review

**Spec-Abdeckung:** Zielzustand und Ablauf (Stück 2 und Tasks 9–11), Ablageort `.superpowers/switchover/<projekt>/` (Dateien-Tabelle, Task 9), ganze Datei statt `config set --propose` (Task 9 Schritt 2), Registry nur für bereits registrierte Projekte mit Prüfsumme (Vorlage Schritt 2), Inhalt und Überschneidung der Wiki-Orte je Ziel (Task 7 Schritt 4a, Task 10), `settings.json` nach `init` bereinigen (Task 5, Vorlage Schritt 6), Existenzwächter und `--check` (Task 6), `merge-hook install` im Abschlussskript (Task 11), zwei Piloten und Welle (Tasks 9–11), Benchmarks mit fünf Warmläufen, Mittelwert und Median, alt/neu/nur-neu/weggefallen (Tasks 1–4, 8, 9), Berichte je Projekt und anonymisiert (Task 12), Stück B/C der Vorspec (Stück 4), `brain-knowledge` als neuntes Ziel (Task 11), `#Obsidian/AI` bleibt (Stück 4).

**Offene Punkte der Spec und wo sie fallen:** Wiki-Ort der `iam_*` (keiner: `init` legt an, Task 11), Module und Guard-Modus von `init --yes` (Task 7 Schritt 2), `area add` ändert vorhandenen Bereich (Task 7 Schritt 3), ganze Ziel-`settings.json` im Trockenlauf (Task 7 Schritt 1), Session-Hooks der `iam_*` (Task 7 Schritt 4), Sicherung vor dem Löschen im Vault (der Commit im Skript; ein Tag ist nicht vorgesehen: bei Bedarf Entscheidung des Nutzers nach Pilot 1).

**Typkonsistenz:** `Compare(before, after []benchreport.Timing) (Result, error)`, `Result{Compared []Row; Added, Dropped []benchreport.Timing}`, `Markdown(Result, title, lang string) (string, error)` in Task 2 und 4; `Build(settings []byte, root, file, dir string) ([]benchhooks.Case, []Payload, error)` in Task 3 und 4; `PruneHooks(settings []byte, needles []string) ([]byte, []string, []string, error)` in Task 5 und 6; `Params` in Task 6 und `params.json` in Task 9.

**Bekannte Lücken, die der Ausführende kennen muss:** Die Zusatzfälle je Ziel (Task 8) entstehen erst, wenn die alten Werkzeuge ihre Argumente nennen (`<werkzeug> --help`); die Namen müssen auf beiden Seiten gleich lauten, sonst erscheint ein vergleichbarer Fall als „neu“ und als „weggefallen“. Die Code-Skizzen (Tasks 2, 3, 5, 6) sind nicht kompiliert; der rote Lauf jedes Tasks zeigt das zuerst, und der Implementierer behebt Tippfehler, ohne das Verhalten zu ändern, das die Tests festlegen.
