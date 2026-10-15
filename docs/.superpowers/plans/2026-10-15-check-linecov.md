# `loomux check linecov` — Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ein eingebauter Befehl `loomux check linecov`, der jede ausführbare
Zeile eines Projekts in jeder Sprache an einer Regel hält — gelaufen oder
begründet ausgenommen — und dafür Berichte, Bereich und Marker selbst aus dem
Projekt abliest.

**Architecture:** Neues Paket `internal/verify/linecov` mit drei Registries
(Berichtsformate, Syntaxen von Einstellungsdateien, Musterdialekte) und
Werkzeugprofilen als Daten in `tools.toml`. Leser liefern `Record`s, `Tree`
löst Pfade gegen den Root auf, `Activate` liest die Einstellungen erkannter
Werkzeuge, `ScanSource` findet Marker, `Gate` wendet die Regeln an, `Check`
verbindet alles. `internal/cli/check.go` bekommt `checkLinecov` nach dem
Vorbild von `checkGocover`.

**Tech Stack:** Go 1.26, Standardbibliothek (`encoding/xml`, `encoding/json`,
`regexp`, `io/fs`), `github.com/BurntSushi/toml` (vorhanden),
`internal/verify/gocover.ModulePath` (vorhanden).

**Spec:** `docs/.superpowers/specs/2026-10-14-check-linecov-design.md`
(inklusive Nachtrag 1, der Entscheidung 6 und 8 ergänzt).

## Global Constraints

- Code, Kommentare, Bezeichner, Fehlermeldungen, Befundtexte und Commits
  englisch; dieser Plan und die Spec bleiben deutsch.
- 100 % Abdeckung je Funktion (`check gocover`); eine Ausnahme nur mit
  `//coverage:exempt <reason>` über der Funktion.
- Kein `init()`, keine Paketvariable, die eingebettete Daten parst
  (`AGENTS.md:50`); `tools.toml` lädt über `sync.OnceValues` wie
  `presets.toml`. `regexp.MustCompile` auf Konstanten ist erlaubt (wie
  `godot.go:211`).
- Tests: Tabellen als `for _, row := range []struct{…}{…}` mit `t.Errorf`,
  Testnamen als Sätze. Jede neue Testzeile läuft vor dem Code rot, und zwar
  an einer Assertion, nicht an einem Compilerfehler: wo die Funktion noch
  fehlt, legt Step 1 sie als Stub an, der den Nullwert liefert.
- Je Task eine Mutationsrunde: `go run ./cmd/loomux dev mutants ./internal/verify/linecov`
  (bzw. das geänderte Paket), zuerst mit einem unveränderten
  Kontrollmutanten, Ausgabe nach `$S/t<N>-mut.txt`. Jeder überlebende Mutant
  bekommt einen Test oder eine Begründung im Commit-Rumpf.
- Conventional Commits, kein Bezug auf Plan, Spec oder Task im Text, kein
  `Co-Authored-By`. Commit immer über `git commit -F "$S/msg-t<N>.txt"`, nur
  mit ausdrücklich genannten Pfaden in `git add`.
- Vor jedem Commit: `test "$(git rev-parse --abbrev-ref HEAD)" = feat/check-linecov`
  und `git rev-parse --show-toplevel` endet auf `worktrees/check-linecov`.
- `$S` ist das Scratchpad der ausführenden Sitzung (`export S=<pfad>`); dort
  liegen Commit-Texte, Mutationsausgaben und Coverage-Profile.
- Abdeckung je Task prüfen:
  `go test ./internal/verify/linecov -count=1 -coverprofile=linecov.out && go run ./cmd/loomux check gocover --profile linecov.out; rm -f linecov.out`
- Kein Push. Gepusht wird nur auf ausdrückliche Anweisung des Menschen.

## Review Focus

1. **Groß-/Kleinschreibung im Berichtspfad unter Windows** (`Core/A.gd` im
   Bericht, `core/a.gd` auf der Platte): erwartet wird ein lauter Befund
   „is in the report but not on disk“, kein stilles Bestehen. Test in Task 6.
2. **Dieselbe Datei zweimal in einem Bericht** (coverlet: partielle Klassen;
   gcov: ein Header je Übersetzungseinheit): erwartet werden summierte
   Treffer und ein einziger Befund je Zeile. Test in Task 14.
3. **CRLF-Quellen und ein Marker in der letzten Zeile ohne Zeilenende**:
   erwartet wird, dass der Marker gilt. Test in Task 12.
4. **Leerer Bericht (0 Byte) oder Binärdatei unter `--report`**: erwartet
   wird der Befund „unknown coverage format“, keine Panik. Test in Task 14.
5. **`pyproject.toml` ohne `[tool.coverage.run]`**: die Datei aktiviert
   coverage.py nicht, und eine spätere Kandidatin der Suchreihenfolge wird
   noch geprüft. Test in Task 11.

---

### Task 1: Report, Finding, Textdekodierung

**Files:**
- Create: `internal/verify/linecov/report.go`
- Test: `internal/verify/linecov/report_test.go`

**Interfaces:**
- Produces:
  - `type Report map[string]map[int]uint64`
  - `func (r Report) File(path string) map[int]uint64`
  - `func (r Report) Add(path string, line int, hits uint64)`
  - `func (r Report) Lines() int`
  - `type Finding struct{ Path string; Line int; Text string }`
  - `func (f Finding) String() string`
  - `func SortFindings(fs []Finding) []Finding` (sortiert, entfernt Duplikate)
  - `func addSat(a, b uint64) uint64`, `func text(content []byte) string`

- [ ] **Step 1: Stubs und rote Tests.** `report.go` mit dem Paketkommentar und
  Stubs, die Nullwerte liefern (`Add` tut nichts, `String` liefert `""`,
  `SortFindings` gibt `fs` zurück, `addSat` liefert `0`, `text` liefert `""`).
  Dann `report_test.go`:

```go
package linecov

import (
	"math"
	"slices"
	"testing"
)

func TestReportSumsHitsPerLineAndKeepsAFileWithoutLines(t *testing.T) {
	r := Report{}
	r.Add("a.gd", 3, 2)
	r.Add("a.gd", 3, 5)
	r.Add("b.gd", 1, 0)
	r.File("c.gd")
	if r["a.gd"][3] != 7 || r["b.gd"][1] != 0 {
		t.Fatalf("%v", r)
	}
	if _, ok := r["c.gd"]; !ok {
		t.Fatal("a file without lines was dropped")
	}
	if r.Lines() != 2 {
		t.Fatalf("lines %d", r.Lines())
	}
}

func TestReportSaturatesInsteadOfWrapping(t *testing.T) {
	r := Report{}
	r.Add("a", 1, math.MaxUint64)
	r.Add("a", 1, 1)
	if r["a"][1] != math.MaxUint64 {
		t.Fatal(r["a"][1])
	}
	if addSat(2, 3) != 5 {
		t.Fatal(addSat(2, 3))
	}
}

func TestFindingPrintsLineFileAndBareForms(t *testing.T) {
	for _, row := range []struct {
		f    Finding
		want string
	}{
		{Finding{"a.gd", 3, "not covered"}, "a.gd:3 not covered"},
		{Finding{"a.gd", 0, "is absent from the report"}, "a.gd is absent from the report"},
		{Finding{"", 0, "no coverage report found"}, "no coverage report found"},
	} {
		if got := row.f.String(); got != row.want {
			t.Errorf("%+v: %q", row.f, got)
		}
	}
}

func TestSortFindingsOrdersByPathLineTextAndDropsDuplicates(t *testing.T) {
	got := SortFindings([]Finding{{"b", 1, "x"}, {"a", 2, "y"}, {"a", 1, "z"}, {"a", 1, "y"}, {"a", 1, "y"}})
	want := []Finding{{"a", 1, "y"}, {"a", 1, "z"}, {"a", 2, "y"}, {"b", 1, "x"}}
	if !slices.Equal(got, want) {
		t.Fatalf("%v", got)
	}
}

func TestTextReplacesInvalidUTF8(t *testing.T) {
	if got := text([]byte("a\xffb")); got != "a\uFFFDb" {
		t.Fatalf("%q", got)
	}
}
```

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: FAIL in allen fünf Tests an den Assertions.

- [ ] **Step 3: Implementieren.**

```go
// Package linecov holds every executable line of a project to one rule: it
// ran, or it carries an exemption with a reason. Reports, scope and markers
// are read from the project itself; see docs/en/cli-reference.md,
// "loomux check linecov".
package linecov

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
)

// Report maps a slash path relative to the root to the hits of each line.
type Report map[string]map[int]uint64

// File returns the lines of path, adding the file when the report lacks it:
// a file a report names without lines still counts as reported.
func (r Report) File(path string) map[int]uint64 {
	lines, ok := r[path]
	if !ok {
		lines = map[int]uint64{}
		r[path] = lines
	}
	return lines
}

// Add sums hits into a line; merged reports add up, they never replace.
func (r Report) Add(path string, line int, hits uint64) {
	lines := r.File(path)
	lines[line] = addSat(lines[line], hits)
}

// Lines counts the line records over every file.
func (r Report) Lines() int {
	n := 0
	for _, lines := range r {
		n += len(lines)
	}
	return n
}

// addSat adds without wrapping: a wrapped sum could read as zero hits.
func addSat(a, b uint64) uint64 {
	if a > math.MaxUint64-b {
		return math.MaxUint64
	}
	return a + b
}

// Finding is one line of the gate's output. Line 0 is about a whole file,
// an empty Path about the run.
type Finding struct {
	Path string
	Line int
	Text string
}

func (f Finding) String() string {
	switch {
	case f.Path == "":
		return f.Text
	case f.Line > 0:
		return fmt.Sprintf("%s:%d %s", f.Path, f.Line, f.Text)
	}
	return f.Path + " " + f.Text
}

// SortFindings orders by path, line and text and drops repeats: two profiles
// that know the same marker must not report it twice.
func SortFindings(fs []Finding) []Finding {
	slices.SortFunc(fs, func(a, b Finding) int {
		return cmp.Or(strings.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line), strings.Compare(a.Text, b.Text))
	})
	return slices.Compact(fs)
}

// text decodes leniently: a refused byte would fail the whole gate on one
// mis-encoded file, and a replaced one cannot hide a marker.
func text(content []byte) string { return strings.ToValidUTF8(string(content), "\uFFFD") }
```

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS. Dann die Abdeckungszeile aus den Global Constraints:
  Expected: keine Zeile `not covered:`.

- [ ] **Step 5: Mutationsrunde** (Global Constraints), Ausgabe `$S/t1-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t1.txt`:

```
feat(linecov): add the report model and findings of the line coverage gate
```

```bash
git add internal/verify/linecov/report.go internal/verify/linecov/report_test.go
git commit -F "$S/msg-t1.txt"
```

---

### Task 2: Format-Registry, Erkennung, LCOV

**Files:**
- Create: `internal/verify/linecov/format.go`, `internal/verify/linecov/lcov.go`
- Test: `internal/verify/linecov/lcov_test.go`

**Interfaces:**
- Consumes: `text`, `addSat` (Task 1)
- Produces:
  - `type Record struct{ Path string; Bases []string; Suffix bool; Lines map[int]uint64 }`
  - `type ReadContext struct{ Module string }`
  - `type Format interface{ Name() string; Detect(head []byte) bool; Read(content []byte, ctx ReadContext) ([]Record, []string) }`
  - `func formats() []Format` — Registry; Tasks 3–5 hängen ihre Formate an
  - `func Detect(content []byte) Format`
  - `func head(content []byte) []byte`
  - `type lcovFormat struct{}`

- [ ] **Step 1: Stubs und rote Tests.** `format.go` vollständig wie in Step 3,
  aber `formats()` liefert `nil`; `lcov.go` mit `Detect` → `false` und
  `Read` → `nil, nil`. Dann `lcov_test.go`:

```go
package linecov

import (
	"reflect"
	"testing"
)

func TestDetectFindsLCOVAfterABOMAndBlankLines(t *testing.T) {
	for _, content := range []string{"\xef\xbb\xbf\n  TN:\nSF:a\n", "SF:a\nDA:1,1\n"} {
		if f := Detect([]byte(content)); f == nil || f.Name() != "lcov" {
			t.Errorf("%q: %v", content, f)
		}
	}
	for _, content := range []string{"", "hello\nSF:a\n", "\x00\x01"} {
		if f := Detect([]byte(content)); f != nil {
			t.Errorf("%q detected as %s", content, f.Name())
		}
	}
}

func TestLCOVReadsLinesByPositionAndReportsWhatItCannotRead(t *testing.T) {
	report := "TN:\r\nSF:res://core/a.gd\r\nDA:3,0\r\nDA:4,2,abc123\r\nFN:1,f\r\nLF:2\r\nend_of_record\r\n" +
		"DA:9,1\nSF:core\\b.gd\nDA:x,1\nDA:0,1\nDA:5,1\nDA:5,2\nend_of_record\n"
	recs, problems := lcovFormat{}.Read([]byte(report), ReadContext{})
	want := []Record{
		{Path: "res://core/a.gd", Lines: map[int]uint64{3: 0, 4: 2}},
		{Path: `core\b.gd`, Lines: map[int]uint64{5: 3}},
	}
	if !reflect.DeepEqual(recs, want) {
		t.Fatalf("%+v", recs)
	}
	wantProblems := []string{`core\b.gd: unreadable record "DA:x,1"`, `core\b.gd: unreadable record "DA:0,1"`}
	if !reflect.DeepEqual(problems, wantProblems) {
		t.Fatalf("%q", problems)
	}
}

func TestLCOVRefusesAHitCountTooLargeToHold(t *testing.T) {
	_, problems := lcovFormat{}.Read([]byte("SF:a\nDA:1,99999999999999999999999\n"), ReadContext{})
	if len(problems) != 1 {
		t.Fatalf("%q", problems)
	}
}
```

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run 'LCOV|Detect' -count=1`
  Expected: FAIL an den Assertions.

- [ ] **Step 3: Implementieren.** `format.go`:

```go
package linecov

import "bytes"

// Record is one file's lines as a report spells them, before the path is
// resolved against the root.
type Record struct {
	Path   string
	Bases  []string // Cobertura <source> entries, tried in order
	Suffix bool     // JaCoCo: Path is package/file and is found by its ending
	Lines  map[int]uint64
}

// ReadContext carries what a format needs from outside its report.
type ReadContext struct {
	Module string // module path from go.mod under the root, "" without one
}

// Format reads one kind of coverage report. A new format is one type and one
// entry in formats.
type Format interface {
	Name() string
	Detect(head []byte) bool
	Read(content []byte, ctx ReadContext) ([]Record, []string)
}

func formats() []Format { return []Format{lcovFormat{}} }

// Detect names the format of content by its first bytes, or nil.
func Detect(content []byte) Format {
	h := head(content)
	for _, f := range formats() {
		if f.Detect(h) {
			return f
		}
	}
	return nil
}

func head(content []byte) []byte {
	return bytes.TrimLeft(bytes.TrimPrefix(content, []byte("\xef\xbb\xbf")), " \t\r\n")
}
```

`lcov.go`:

```go
package linecov

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type lcovFormat struct{}

func (lcovFormat) Name() string { return "lcov" }

func (lcovFormat) Detect(h []byte) bool {
	first, _, _ := bytes.Cut(h, []byte("\n"))
	return bytes.HasPrefix(first, []byte("TN:")) || bytes.HasPrefix(first, []byte("SF:"))
}

// daRecord reads DA:<line>,<hits>[,<checksum>] by position: splitting on the
// first comma would read "0,<checksum>" as the hits and lose an uncovered line.
var daRecord = regexp.MustCompile(`^(\d+),(\d+)(?:,|$)`)

// Read keeps only SF and DA; a DA before the first SF or after end_of_record
// belongs to no file and is passed over.
func (lcovFormat) Read(content []byte, _ ReadContext) ([]Record, []string) {
	var recs []Record
	var problems []string
	cur := -1
	for _, raw := range strings.Split(text(content), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "SF:"):
			recs = append(recs, Record{Path: strings.TrimSpace(line[3:]), Lines: map[int]uint64{}})
			cur = len(recs) - 1
		case line == "end_of_record":
			cur = -1
		case strings.HasPrefix(line, "DA:") && cur >= 0:
			n, hits, ok := daFields(line[3:])
			if !ok {
				problems = append(problems, fmt.Sprintf("%s: unreadable record %q", recs[cur].Path, line))
				continue
			}
			recs[cur].Lines[n] = addSat(recs[cur].Lines[n], hits)
		}
	}
	return recs, problems
}

func daFields(s string) (int, uint64, bool) {
	m := daRecord.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 {
		return 0, 0, false
	}
	hits, err := strconv.ParseUint(m[2], 10, 64)
	return n, hits, err == nil
}
```

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`.

- [ ] **Step 5: Mutationsrunde**, `$S/t2-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t2.txt`:

```
feat(linecov): read LCOV reports and detect the format by content
```

```bash
git add internal/verify/linecov/format.go internal/verify/linecov/lcov.go internal/verify/linecov/lcov_test.go
git commit -F "$S/msg-t2.txt"
```

---

### Task 3: Cobertura

**Files:**
- Create: `internal/verify/linecov/cobertura.go`
- Modify: `internal/verify/linecov/format.go` (`formats()`)
- Test: `internal/verify/linecov/cobertura_test.go`

**Interfaces:**
- Consumes: `Record`, `ReadContext`, `formats()` (Task 2)
- Produces: `type coberturaFormat struct{}`, `func xmlRoot(h []byte) string`

- [ ] **Step 1: Stub und rote Tests.** `cobertura.go` mit `xmlRoot` → `""`,
  `Detect` → `false`, `Read` → `nil, nil`; `formats()` liefert
  `[]Format{coberturaFormat{}, lcovFormat{}}`. Dann `cobertura_test.go`:

```go
package linecov

import (
	"reflect"
	"testing"
)

const coberturaReport = `<?xml version="1.0" ?>
<!DOCTYPE coverage SYSTEM "http://cobertura.sourceforge.net/xml/coverage-04.dtd">
<coverage line-rate="0.5" version="7.6.1">
  <sources><source> src </source><source>C:\work\app\lib</source></sources>
  <packages><package name="p"><classes>
    <class name="A" filename="app.py"><methods/><lines>
      <line number="1" hits="3"/><line number="2" hits="0"/>
    </lines></class>
    <class name="A.Inner" filename="app.py"><lines><line number="2" hits="1"/></lines></class>
  </classes></package></packages>
</coverage>`

func TestDetectTellsCoberturaByItsRootElement(t *testing.T) {
	if f := Detect([]byte(coberturaReport)); f == nil || f.Name() != "cobertura" {
		t.Fatalf("%v", f)
	}
	for _, s := range []string{"<report/>", "<coverage", "<!-- only a comment -->", "plain"} {
		if f := Detect([]byte(s)); f != nil && f.Name() == "cobertura" {
			t.Errorf("%q detected as cobertura", s)
		}
	}
}

func TestCoberturaReadsEveryClassWithTheReportSources(t *testing.T) {
	recs, problems := coberturaFormat{}.Read([]byte(coberturaReport), ReadContext{})
	bases := []string{"src", `C:\work\app\lib`}
	want := []Record{
		{Path: "app.py", Bases: bases, Lines: map[int]uint64{1: 3, 2: 0}},
		{Path: "app.py", Bases: bases, Lines: map[int]uint64{2: 1}},
	}
	if len(problems) != 0 || !reflect.DeepEqual(recs, want) {
		t.Fatalf("%+v %q", recs, problems)
	}
}

func TestCoberturaReportsBrokenXMLAndUnreadableLines(t *testing.T) {
	if _, problems := (coberturaFormat{}).Read([]byte("<coverage><packages>"), ReadContext{}); len(problems) != 1 {
		t.Fatalf("%q", problems)
	}
	bad := `<coverage><packages><package><classes><class filename="a.py"><lines><line number="x" hits="1"/><line number="2" hits="-1"/></lines></class></classes></package></packages></coverage>`
	recs, problems := coberturaFormat{}.Read([]byte(bad), ReadContext{})
	if len(problems) != 2 || len(recs) != 1 || len(recs[0].Lines) != 0 {
		t.Fatalf("%+v %q", recs, problems)
	}
}
```

  Review-Focus 2 (dieselbe Datei zweimal) ist hier der Fall `A` und
  `A.Inner`; die Summe pinnt Task 14 (`TestCheckSumsAFileListedTwice`).

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run Cobertura -count=1`
  Expected: FAIL.

- [ ] **Step 3: Implementieren.**

```go
package linecov

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// xmlRoot names the first element of an XML document, or "" for anything
// else. DOCTYPE, comments and the declaration are passed over; nothing is
// fetched.
func xmlRoot(h []byte) string {
	if len(h) == 0 || h[0] != '<' {
		return ""
	}
	d := xml.NewDecoder(bytes.NewReader(h))
	for {
		tok, err := d.Token()
		if err != nil {
			return ""
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local
		}
	}
}

type coberturaFormat struct{}

func (coberturaFormat) Name() string           { return "cobertura" }
func (coberturaFormat) Detect(h []byte) bool    { return xmlRoot(h) == "coverage" }

type coberturaDoc struct {
	Sources []string `xml:"sources>source"`
	Classes []struct {
		Filename string `xml:"filename,attr"`
		Lines    []struct {
			Number string `xml:"number,attr"`
			Hits   string `xml:"hits,attr"`
		} `xml:"lines>line"`
	} `xml:"packages>package>classes>class"`
}

// Read keeps one record per class; a file split over classes is summed when
// the records are merged.
func (coberturaFormat) Read(content []byte, _ ReadContext) ([]Record, []string) {
	var doc coberturaDoc
	if err := xml.Unmarshal([]byte(text(content)), &doc); err != nil {
		return nil, []string{"unreadable XML: " + err.Error()}
	}
	var bases []string
	for _, s := range doc.Sources {
		bases = append(bases, strings.TrimSpace(s))
	}
	var recs []Record
	var problems []string
	for _, c := range doc.Classes {
		rec := Record{Path: c.Filename, Bases: bases, Lines: map[int]uint64{}}
		for _, l := range c.Lines {
			n, err1 := strconv.Atoi(l.Number)
			hits, err2 := strconv.ParseUint(l.Hits, 10, 64)
			if err1 != nil || err2 != nil || n < 1 {
				problems = append(problems, fmt.Sprintf("%s: unreadable line number=%q hits=%q", c.Filename, l.Number, l.Hits))
				continue
			}
			rec.Lines[n] = addSat(rec.Lines[n], hits)
		}
		recs = append(recs, rec)
	}
	return recs, problems
}
```

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`.

- [ ] **Step 5: Mutationsrunde**, `$S/t3-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t3.txt`:

```
feat(linecov): read Cobertura reports
```

```bash
git add internal/verify/linecov/cobertura.go internal/verify/linecov/cobertura_test.go internal/verify/linecov/format.go
git commit -F "$S/msg-t3.txt"
```

---

### Task 4: JaCoCo (und Kover)

**Files:**
- Create: `internal/verify/linecov/jacoco.go`
- Modify: `internal/verify/linecov/format.go` (`formats()`)
- Test: `internal/verify/linecov/jacoco_test.go`

**Interfaces:**
- Consumes: `xmlRoot` (Task 3)
- Produces: `type jacocoFormat struct{}`

- [ ] **Step 1: Stub und rote Tests.** `jacoco.go` mit `Detect` → `false`,
  `Read` → `nil, nil`; `formats()` →
  `[]Format{jacocoFormat{}, coberturaFormat{}, lcovFormat{}}`.

```go
package linecov

import (
	"reflect"
	"testing"
)

const jacocoReport = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<!DOCTYPE report PUBLIC "-//JACOCO//DTD Report 1.1//EN" "report.dtd">
<report name="app">
  <package name="com/x">
    <class name="com/x/Calc" sourcefilename="Calc.java"/>
    <sourcefile name="Calc.java">
      <line nr="3" mi="0" ci="2" mb="0" cb="0"/>
      <line nr="4" mi="3" ci="0" mb="0" cb="0"/>
    </sourcefile>
  </package>
  <package name="">
    <sourcefile name="Main.kt"><line nr="1" mi="0" ci="1"/></sourcefile>
  </package>
</report>`

func TestJaCoCoReadsCoveredInstructionsPerLineAndAsksForASuffixSearch(t *testing.T) {
	if f := Detect([]byte(jacocoReport)); f == nil || f.Name() != "jacoco" {
		t.Fatalf("%v", f)
	}
	recs, problems := jacocoFormat{}.Read([]byte(jacocoReport), ReadContext{})
	want := []Record{
		{Path: "com/x/Calc.java", Suffix: true, Lines: map[int]uint64{3: 2, 4: 0}},
		{Path: "Main.kt", Suffix: true, Lines: map[int]uint64{1: 1}},
	}
	if len(problems) != 0 || !reflect.DeepEqual(recs, want) {
		t.Fatalf("%+v %q", recs, problems)
	}
}

func TestJaCoCoReportsBrokenXMLAndUnreadableLines(t *testing.T) {
	if _, problems := (jacocoFormat{}).Read([]byte("<report><package>"), ReadContext{}); len(problems) != 1 {
		t.Fatalf("%q", problems)
	}
	bad := `<report><package name="p"><sourcefile name="A.java"><line nr="0" ci="1"/><line nr="2" ci="x"/></sourcefile></package></report>`
	recs, problems := jacocoFormat{}.Read([]byte(bad), ReadContext{})
	if len(problems) != 2 || len(recs) != 1 || len(recs[0].Lines) != 0 {
		t.Fatalf("%+v %q", recs, problems)
	}
}
```

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run JaCoCo -count=1`
  Expected: FAIL.

- [ ] **Step 3: Implementieren.**

```go
package linecov

import (
	"encoding/xml"
	"fmt"
	"path"
	"strconv"
)

// jacocoFormat reads JaCoCo XML, which Kover writes too. A source file is
// named by package and file name only, so its record asks for a suffix search.
type jacocoFormat struct{}

func (jacocoFormat) Name() string        { return "jacoco" }
func (jacocoFormat) Detect(h []byte) bool { return xmlRoot(h) == "report" }

type jacocoDoc struct {
	Packages []struct {
		Name  string `xml:"name,attr"`
		Files []struct {
			Name  string `xml:"name,attr"`
			Lines []struct {
				Nr string `xml:"nr,attr"`
				CI string `xml:"ci,attr"`
			} `xml:"line"`
		} `xml:"sourcefile"`
	} `xml:"package"`
}

// Read counts a line as hit when one of its instructions ran (ci > 0).
func (jacocoFormat) Read(content []byte, _ ReadContext) ([]Record, []string) {
	var doc jacocoDoc
	if err := xml.Unmarshal([]byte(text(content)), &doc); err != nil {
		return nil, []string{"unreadable XML: " + err.Error()}
	}
	var recs []Record
	var problems []string
	for _, p := range doc.Packages {
		for _, f := range p.Files {
			rec := Record{Path: path.Join(p.Name, f.Name), Suffix: true, Lines: map[int]uint64{}}
			for _, l := range f.Lines {
				n, err1 := strconv.Atoi(l.Nr)
				ci, err2 := strconv.ParseUint(l.CI, 10, 64)
				if err1 != nil || err2 != nil || n < 1 {
					problems = append(problems, fmt.Sprintf("%s: unreadable line nr=%q ci=%q", rec.Path, l.Nr, l.CI))
					continue
				}
				rec.Lines[n] = addSat(rec.Lines[n], ci)
			}
			recs = append(recs, rec)
		}
	}
	return recs, problems
}
```

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`.

- [ ] **Step 5: Mutationsrunde**, `$S/t4-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t4.txt`:

```
feat(linecov): read JaCoCo and Kover XML reports
```

```bash
git add internal/verify/linecov/jacoco.go internal/verify/linecov/jacoco_test.go internal/verify/linecov/format.go
git commit -F "$S/msg-t4.txt"
```

---

### Task 5: Go-Profil

**Files:**
- Create: `internal/verify/linecov/goprofile.go`
- Modify: `internal/verify/linecov/format.go` (`formats()`)
- Test: `internal/verify/linecov/goprofile_test.go`

**Interfaces:**
- Consumes: `ReadContext.Module`
- Produces: `type goProfileFormat struct{}`

- [ ] **Step 1: Stub und rote Tests.** `goprofile.go` mit `Detect` → `false`,
  `Read` → `nil, nil`; `formats()` →
  `[]Format{goProfileFormat{}, jacocoFormat{}, coberturaFormat{}, lcovFormat{}}`.

```go
package linecov

import (
	"reflect"
	"testing"
)

const goReport = "mode: set\n" +
	"example.com/m/a.go:3.10,5.2 2 1\n" +
	"example.com/m/a.go:5.2,7.3 1 0\n" +
	"example.com/m/sub/b.go:1.1,1.9 1 0\n" +
	"other.org/x/c.go:1.1,2.2 1 1\n" +
	"other.org/x/c.go:4.1,4.2 1 1\n" +
	"garbage\n" +
	"example.com/m/a.go:9.1,8.1 1 1\n"

func TestGoProfileSpreadsBlocksOverLinesAndKeepsTheHighestCount(t *testing.T) {
	if f := Detect([]byte(goReport)); f == nil || f.Name() != "goprofile" {
		t.Fatalf("%v", f)
	}
	recs, problems := goProfileFormat{}.Read([]byte(goReport), ReadContext{Module: "example.com/m"})
	want := []Record{
		{Path: "a.go", Lines: map[int]uint64{3: 1, 4: 1, 5: 1, 6: 0, 7: 0}},
		{Path: "sub/b.go", Lines: map[int]uint64{1: 0}},
	}
	if !reflect.DeepEqual(recs, want) {
		t.Fatalf("%+v", recs)
	}
	wantProblems := []string{
		"other.org/x/c.go: outside module example.com/m",
		`unreadable record "garbage"`,
		`unreadable record "example.com/m/a.go:9.1,8.1 1 1"`,
	}
	if !reflect.DeepEqual(problems, wantProblems) {
		t.Fatalf("%q", problems)
	}
}

func TestGoProfileNeedsAModule(t *testing.T) {
	recs, problems := goProfileFormat{}.Read([]byte(goReport), ReadContext{})
	if recs != nil || len(problems) != 1 {
		t.Fatalf("%+v %q", recs, problems)
	}
}
```

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run GoProfile -count=1`
  Expected: FAIL.

- [ ] **Step 3: Implementieren.**

```go
package linecov

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type goProfileFormat struct{}

func (goProfileFormat) Name() string        { return "goprofile" }
func (goProfileFormat) Detect(h []byte) bool { return bytes.HasPrefix(h, []byte("mode: ")) }

// goBlock is <import path>:<line>.<col>,<line>.<col> <statements> <count>.
var goBlock = regexp.MustCompile(`^(.+):(\d+)\.\d+,(\d+)\.\d+ \d+ (\d+)$`)

// Read gives every line of a block the block's count; a line under several
// blocks keeps the highest, so one run block marks it as hit.
func (goProfileFormat) Read(content []byte, ctx ReadContext) ([]Record, []string) {
	if ctx.Module == "" {
		return nil, []string{"no go.mod under the root: Go profile paths cannot be resolved"}
	}
	var recs []Record
	var problems []string
	index := map[string]int{}
	outside := map[string]bool{}
	for _, raw := range strings.Split(text(content), "\n")[1:] {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		file, from, to, count, ok := goFields(line)
		if !ok {
			problems = append(problems, fmt.Sprintf("unreadable record %q", line))
			continue
		}
		rel, inside := strings.CutPrefix(file, ctx.Module+"/")
		if !inside {
			if !outside[file] {
				outside[file] = true
				problems = append(problems, fmt.Sprintf("%s: outside module %s", file, ctx.Module))
			}
			continue
		}
		i, seen := index[rel]
		if !seen {
			i = len(recs)
			index[rel] = i
			recs = append(recs, Record{Path: rel, Lines: map[int]uint64{}})
		}
		for n := from; n <= to; n++ {
			if cur, ok := recs[i].Lines[n]; !ok || count > cur {
				recs[i].Lines[n] = count
			}
		}
	}
	return recs, problems
}

func goFields(line string) (file string, from, to int, count uint64, ok bool) {
	m := goBlock.FindStringSubmatch(line)
	if m == nil {
		return "", 0, 0, 0, false
	}
	from, err1 := strconv.Atoi(m[2])
	to, err2 := strconv.Atoi(m[3])
	count, err3 := strconv.ParseUint(m[4], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || from < 1 || to < from {
		return "", 0, 0, 0, false
	}
	return m[1], from, to, count, true
}
```

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`.

- [ ] **Step 5: Mutationsrunde**, `$S/t5-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t5.txt`:

```
feat(linecov): read Go coverage profiles line by line
```

```bash
git add internal/verify/linecov/goprofile.go internal/verify/linecov/goprofile_test.go internal/verify/linecov/format.go
git commit -F "$S/msg-t5.txt"
```

---

### Task 6: Baum und Pfadauflösung

**Files:**
- Create: `internal/verify/linecov/tree.go`, `internal/verify/linecov/helpers_test.go`
- Test: `internal/verify/linecov/tree_test.go`

**Interfaces:**
- Consumes: `Record` (Task 2)
- Produces:
  - `type Tree struct{ Root string; Files []string; has map[string]bool; nested []string }`
  - `func ScanTree(root string) (*Tree, error)`
  - `func (t *Tree) Has(rel string) bool`, `func (t *Tree) InNested(rel string) bool`
  - `type Outcome int` mit `Resolved, Outside, Nested, Missing, Ambiguous`
  - `type Resolution struct{ Rel string; Outcome Outcome; Candidates []string }`
  - `func (t *Tree) Resolve(rec Record) Resolution`
  - `var foldCase = runtime.GOOS == "windows"` (Naht für Tests)
  - Testhelfer `writeTree(t *testing.T, files map[string]string) string`

- [ ] **Step 1: Helfer, Stubs, rote Tests.** `helpers_test.go`:

```go
package linecov

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTree makes a directory holding files (slash paths to contents) and
// returns its path.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
```

  `tree.go` mit Stubs (`ScanTree` → `&Tree{}, nil`, `Resolve` →
  `Resolution{}`, `Has`/`InNested` → `false`). Dann `tree_test.go`:

```go
package linecov

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestScanTreeSkipsGitAndOtherCheckouts(t *testing.T) {
	root := writeTree(t, map[string]string{
		"core/a.gd":                       "",
		".git/config":                     "",
		".claude/worktrees/w/.git":        "gitdir: elsewhere",
		".claude/worktrees/w/core/a.gd":   "",
		"vendor/lib/.git/HEAD":            "",
		"vendor/lib/x.c":                  "",
		"z.txt":                           "",
	})
	tree, err := ScanTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"core/a.gd", "z.txt"}; !reflect.DeepEqual(tree.Files, want) {
		t.Fatalf("%q", tree.Files)
	}
	if !tree.InNested(".claude/worktrees/w/core/a.gd") || !tree.InNested("vendor/lib") || tree.InNested("vendor/libx/a.c") {
		t.Fatal("nested checkouts not recognised")
	}
	if !tree.Has("core/a.gd") || tree.Has("core/b.gd") {
		t.Fatal("Has")
	}
}

func TestScanTreeFailsOnAMissingRoot(t *testing.T) {
	if _, err := ScanTree(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("no error")
	}
}

func TestResolveMapsEveryPathFormToTheRoot(t *testing.T) {
	root := writeTree(t, map[string]string{
		"core/a.gd": "", "src/app.py": "", "src/main/java/com/x/Calc.java": "",
		"a/com/y/D.java": "", "b/com/y/D.java": "",
		"sub/.git": "", "sub/s.c": "",
	})
	tree, err := ScanTree(root)
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.ToSlash(tree.Root)
	for _, row := range []struct {
		rec  Record
		want Resolution
	}{
		{Record{Path: "res://core/a.gd"}, Resolution{Rel: "core/a.gd", Outcome: Resolved}},
		{Record{Path: `core\a.gd`}, Resolution{Rel: "core/a.gd", Outcome: Resolved}},
		{Record{Path: abs + "/core/a.gd"}, Resolution{Rel: "core/a.gd", Outcome: Resolved}},
		{Record{Path: "/usr/include/stdio.h"}, Resolution{Rel: "/usr/include/stdio.h", Outcome: Outside}},
		{Record{Path: "../elsewhere/x.c"}, Resolution{Rel: "../elsewhere/x.c", Outcome: Outside}},
		{Record{Path: "app.py", Bases: []string{"lib", "src"}}, Resolution{Rel: "src/app.py", Outcome: Resolved}},
		{Record{Path: "app.py", Bases: []string{abs + "/src"}}, Resolution{Rel: "src/app.py", Outcome: Resolved}},
		{Record{Path: "gone.py", Bases: []string{"src"}}, Resolution{Rel: "gone.py", Outcome: Missing}},
		{Record{Path: "com/x/Calc.java", Suffix: true}, Resolution{Rel: "src/main/java/com/x/Calc.java", Outcome: Resolved}},
		{Record{Path: "com/x/Gone.java", Suffix: true}, Resolution{Rel: "com/x/Gone.java", Outcome: Missing}},
		{Record{Path: "com/y/D.java", Suffix: true}, Resolution{Rel: "com/y/D.java", Outcome: Ambiguous, Candidates: []string{"a/com/y/D.java", "b/com/y/D.java"}}},
		{Record{Path: "sub/s.c"}, Resolution{Rel: "sub/s.c", Outcome: Nested}},
	} {
		if got := tree.Resolve(row.rec); !reflect.DeepEqual(got, row.want) {
			t.Errorf("%+v: %+v", row.rec, got)
		}
	}
}

// Review focus 1: a report spelling the path in another case finds no file
// and says so; only the root prefix of an absolute path is compared without
// case on Windows.
func TestResolveComparesTheRootPrefixWithoutCaseOnlyWhereTheSystemDoes(t *testing.T) {
	root := writeTree(t, map[string]string{"core/a.gd": ""})
	tree, err := ScanTree(root)
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(filepath.ToSlash(tree.Root)) + "/core/a.gd"
	old := foldCase
	t.Cleanup(func() { foldCase = old })
	foldCase = true
	if got := tree.Resolve(Record{Path: upper}); got.Outcome != Resolved {
		t.Fatalf("fold: %+v", got)
	}
	foldCase = false
	if got := tree.Resolve(Record{Path: upper}); got.Outcome != Outside {
		t.Fatalf("no fold: %+v", got)
	}
	if got := tree.Resolve(Record{Path: "Core/A.gd"}); got.Outcome != Missing {
		t.Fatalf("case of the relative part: %+v", got)
	}
}

// A root at the top of the file system already ends in a separator.
func TestResolveTreatsARootAtTheTopOfTheFileSystem(t *testing.T) {
	root, p := "/", "/x/y.c"
	if runtime.GOOS == "windows" {
		root, p = `C:\`, "C:/x/y.c"
	}
	tree := &Tree{Root: root, has: map[string]bool{"x/y.c": true}}
	if got := tree.Resolve(Record{Path: p}); got.Outcome != Resolved || got.Rel != "x/y.c" {
		t.Fatalf("%+v", got)
	}
}
```

  Imports des Tests: `path/filepath`, `reflect`, `runtime`, `strings`,
  `testing` (kein `os`).

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run 'Tree|Resolve' -count=1`
  Expected: FAIL.

- [ ] **Step 3: Implementieren.**

```go
package linecov

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Tree is the root's files, walked once and shared by path resolution, the
// suffix search and the expected sources.
type Tree struct {
	Root   string   // absolute and clean
	Files  []string // slash paths below Root, sorted; .git and other checkouts left out
	has    map[string]bool
	nested []string
}

// foldCase compares the root prefix of an absolute path without case, as the
// file system does on Windows.
var foldCase = runtime.GOOS == "windows"

// ScanTree walks root. A directory holding a .git entry is another checkout
// (a linked worktree, a submodule): its files belong to another project.
// ponytail: walks node_modules and build output too; skip them when a large
// JavaScript tree makes the walk measurably slow.
func ScanTree(root string) (*Tree, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	t := &Tree{Root: abs, has: map[string]bool{}}
	err = filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == abs {
			return err
		}
		rel, _ := filepath.Rel(abs, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			if _, err := os.Lstat(filepath.Join(p, ".git")); err == nil {
				t.nested = append(t.nested, rel)
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && d.Name() != ".git" {
			t.Files = append(t.Files, rel)
			t.has[rel] = true
		}
		return nil
	})
	slices.Sort(t.Files)
	return t, err
}

func (t *Tree) Has(rel string) bool { return t.has[rel] }

func (t *Tree) InNested(rel string) bool {
	return slices.ContainsFunc(t.nested, func(n string) bool { return rel == n || strings.HasPrefix(rel, n+"/") })
}

type Outcome int

const (
	Resolved Outcome = iota
	Outside          // not below the root: a system header, a dependency
	Nested           // inside another checkout below the root
	Missing          // below the root but not on disk
	Ambiguous        // a suffix search found more than one file
)

type Resolution struct {
	Rel        string
	Outcome    Outcome
	Candidates []string
}

// Resolve turns a report's spelling of a path into a slash path relative to
// the root: backslashes and res:// first, then a suffix search, an absolute
// path, the report's bases in order, and last the path as it stands.
func (t *Tree) Resolve(rec Record) Resolution {
	p := strings.TrimPrefix(strings.ReplaceAll(rec.Path, `\`, "/"), "res://")
	if rec.Suffix {
		return t.bySuffix(p)
	}
	if isAbs(p) {
		rel, ok := t.relTo(p)
		if !ok {
			return Resolution{Rel: p, Outcome: Outside}
		}
		return t.check(rel)
	}
	for _, b := range rec.Bases {
		joined := path.Join(strings.ReplaceAll(b, `\`, "/"), p)
		rel, ok := cleanRel(joined)
		if isAbs(joined) {
			rel, ok = t.relTo(joined)
		}
		if ok && t.has[rel] {
			return t.check(rel)
		}
	}
	rel, ok := cleanRel(p)
	if !ok {
		return Resolution{Rel: p, Outcome: Outside}
	}
	return t.check(rel)
}

func (t *Tree) check(rel string) Resolution {
	switch {
	case t.InNested(rel):
		return Resolution{Rel: rel, Outcome: Nested}
	case !t.has[rel]:
		return Resolution{Rel: rel, Outcome: Missing}
	}
	return Resolution{Rel: rel, Outcome: Resolved}
}

func (t *Tree) bySuffix(p string) Resolution {
	var found []string
	for _, f := range t.Files {
		if f == p || strings.HasSuffix(f, "/"+p) {
			found = append(found, f)
		}
	}
	switch len(found) {
	case 0:
		return Resolution{Rel: p, Outcome: Missing}
	case 1:
		return Resolution{Rel: found[0], Outcome: Resolved}
	}
	return Resolution{Rel: p, Outcome: Ambiguous, Candidates: found}
}

// relTo cuts the root off an absolute slash path.
func (t *Tree) relTo(p string) (string, bool) {
	prefix := strings.TrimSuffix(filepath.ToSlash(t.Root), "/") + "/"
	if len(p) < len(prefix) {
		return "", false
	}
	got := p[:len(prefix)]
	if got != prefix && !(foldCase && strings.EqualFold(got, prefix)) {
		return "", false
	}
	return cleanRel(p[len(prefix):])
}

// cleanRel cleans a relative slash path and refuses one that leaves the root.
func cleanRel(p string) (string, bool) {
	c := path.Clean(p)
	return c, c != "." && c != ".." && !strings.HasPrefix(c, "../")
}

func isAbs(p string) bool {
	return strings.HasPrefix(p, "/") || (len(p) >= 3 && p[1] == ':' && p[2] == '/')
}
```

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`. Fehlt Abdeckung für den
  Fehlerzweig von `WalkDir` (nicht lesbares Unterverzeichnis), deckt ihn
  `TestScanTreeFailsOnAMissingRoot` (`err != nil` beim ersten Aufruf).

- [ ] **Step 5: Mutationsrunde**, `$S/t6-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t6.txt`:

```
feat(linecov): resolve report paths against the root and skip other checkouts
```

```bash
git add internal/verify/linecov/tree.go internal/verify/linecov/tree_test.go internal/verify/linecov/helpers_test.go
git commit -F "$S/msg-t6.txt"
```

---

### Task 7: Syntax-Registry, INI, Godot-cfg

**Files:**
- Create: `internal/verify/linecov/syntax.go`, `internal/verify/linecov/ini.go`, `internal/verify/linecov/godotcfg.go`
- Test: `internal/verify/linecov/syntax_test.go`

**Interfaces:**
- Consumes: `text` (Task 1)
- Produces:
  - `type Values map[string][]string`, `func (v Values) add(key string, vals ...string)`
  - `type Where struct{ Section, Table, Path string }`
  - `type Syntax func(content []byte, w Where) (Values, bool, error)` — `bool`: Abschnitt/Tabelle/Pfad vorhanden
  - `func syntaxes() map[string]Syntax` (Task 8 ergänzt `toml`, `json`, `xml`)
  - `func readINI`, `func readGodotCfg`, `func splitList(s string) []string`

- [ ] **Step 1: Stubs und rote Tests.** Stubs liefern `nil, false, nil`;
  `syntaxes()` liefert `{"ini": readINI, "godotcfg": readGodotCfg}`.

```go
package linecov

import (
	"reflect"
	"testing"
)

func TestINIReadsASectionWithListsAndContinuations(t *testing.T) {
	cfg := "# top\nfilter = src/\nfilter = lib/\n[run]\nSource = src,\n    pkg\nomit =\n    src/gen/*\n    */tests/*\n; note\nbranch: true\n[report]\nomit = other\n"
	vals, found, err := readINI([]byte(cfg), Where{Section: "run"})
	want := Values{"source": {"src", "pkg"}, "omit": {"src/gen/*", "*/tests/*"}, "branch": {"true"}}
	if err != nil || !found || !reflect.DeepEqual(vals, want) {
		t.Fatalf("%v %v %v", vals, found, err)
	}
	top, found, err := readINI([]byte(cfg), Where{})
	if err != nil || !found || !reflect.DeepEqual(top, Values{"filter": {"src/", "lib/"}}) {
		t.Fatalf("%v %v %v", top, found, err)
	}
	if _, found, _ := readINI([]byte(cfg), Where{Section: "coverage:run"}); found {
		t.Fatal("found a section that is not there")
	}
}

func TestINIRefusesWhatConfigparserRefuses(t *testing.T) {
	for _, cfg := range []string{"[run\nx = 1\n", "  orphan\n", "no assignment\n", "=value\n"} {
		if _, _, err := readINI([]byte(cfg), Where{Section: "run"}); err == nil {
			t.Errorf("%q: no error", cfg)
		}
	}
}

func TestGodotCfgReadsPackedArraysStringsAndBooleans(t *testing.T) {
	cfg := "config_version=5\n\n[application]\nconfig/name=\"space\"\n\n[nano_coverage]\n; comment\n" +
		"general/ignore_paths=PackedStringArray(\"**/*_test.gd\", \"res://harness/(old)/**\",\n\"res://x\\\"y\")\n" +
		"general/ignore_addons=false\ngeneral/report_dir=\"res://coverage-report\"\ngeneral/odd=\"a\\q\"\n"
	vals, found, err := readGodotCfg([]byte(cfg), Where{Section: "nano_coverage"})
	want := Values{
		"general/ignore_paths":  {"**/*_test.gd", "res://harness/(old)/**", `res://x"y`},
		"general/ignore_addons": {"false"},
		"general/report_dir":    {"res://coverage-report"},
		"general/odd":           {`a\q`}, // an escape Go cannot unquote stays as written
	}
	if err != nil || !found || !reflect.DeepEqual(vals, want) {
		t.Fatalf("%q %v %v", vals, found, err)
	}
}

func TestGodotCfgRefusesBrokenFiles(t *testing.T) {
	for _, cfg := range []string{"[a\n", "[a]\nkey\n", "[a]\nk=PackedStringArray(\"x\"\n"} {
		if _, _, err := readGodotCfg([]byte(cfg), Where{Section: "a"}); err == nil {
			t.Errorf("%q: no error", cfg)
		}
	}
}
```

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run 'INI|GodotCfg' -count=1`
  Expected: FAIL.

- [ ] **Step 3: Implementieren.** `syntax.go`:

```go
package linecov

import "strings"

// Values maps a settings key to its values; a scalar is a list of one.
type Values map[string][]string

func (v Values) add(key string, vals ...string) {
	for _, s := range vals {
		if s = strings.TrimSpace(s); s != "" {
			v[key] = append(v[key], s)
		}
	}
}

// Where names the part of a settings file a profile reads: a section (ini,
// godotcfg), a dotted table (toml), or a path (xml: slash elements from the
// root element, json: dotted keys).
type Where struct {
	Section string
	Table   string
	Path    string
}

// Syntax reads the values under w; found says whether w exists in the file,
// so a lookup order can pass over a file that does not configure the tool.
// A new syntax is one function and one entry in syntaxes.
type Syntax func(content []byte, w Where) (vals Values, found bool, err error)

func syntaxes() map[string]Syntax {
	return map[string]Syntax{"ini": readINI, "godotcfg": readGodotCfg}
}

// splitList splits a list value on commas and newlines.
func splitList(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '\n' })
}
```

`ini.go`:

```go
package linecov

import (
	"fmt"
	"strings"
)

// readINI reads configparser-style files (.coveragerc, setup.cfg, tox.ini,
// gcovr.cfg, .lcovrc): keys before the first section belong to section "",
// keys are lower-cased, an indented line continues the value above, and a
// repeated key adds to its list.
func readINI(content []byte, w Where) (Values, bool, error) {
	vals := Values{}
	section, key := "", ""
	found := w.Section == ""
	for i, raw := range strings.Split(text(content), "\n") {
		line := strings.TrimRight(raw, " \t\r")
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "" || trimmed[0] == '#' || trimmed[0] == ';':
		case line[0] == '[':
			end := strings.IndexByte(trimmed, ']')
			if end < 0 {
				return nil, false, fmt.Errorf("line %d: unclosed section header", i+1)
			}
			section, key = strings.TrimSpace(trimmed[1:end]), ""
			found = found || section == w.Section
		case line[0] == ' ' || line[0] == '\t':
			if key == "" {
				return nil, false, fmt.Errorf("line %d: continuation without a key", i+1)
			}
			if section == w.Section {
				vals.add(key, splitList(trimmed)...)
			}
		default:
			k, v, ok := cutAssign(trimmed)
			if !ok {
				return nil, false, fmt.Errorf("line %d: neither = nor :", i+1)
			}
			key = strings.ToLower(k)
			if section == w.Section {
				vals.add(key, splitList(v)...)
			}
		}
	}
	return vals, found, nil
}

func cutAssign(s string) (string, string, bool) {
	i := strings.IndexAny(s, "=:")
	if i <= 0 {
		return "", "", false
	}
	return strings.TrimSpace(s[:i]), s[i+1:], true
}
```

`godotcfg.go`:

```go
package linecov

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var godotString = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)

// readGodotCfg reads project.godot: [section], key=value, values that are
// strings, booleans or PackedStringArray(...), which may run over lines.
func readGodotCfg(content []byte, w Where) (Values, bool, error) {
	vals := Values{}
	section, pending := "", ""
	found := false
	for i, raw := range strings.Split(text(content), "\n") {
		line := strings.TrimSpace(raw)
		if pending != "" {
			if pending += "\n" + line; open(pending) {
				continue
			}
			line, pending = pending, ""
		} else if line == "" || line[0] == ';' {
			continue
		} else if line[0] == '[' {
			end := strings.IndexByte(line, ']')
			if end < 0 {
				return nil, false, fmt.Errorf("line %d: unclosed section header", i+1)
			}
			section = line[1:end]
			found = found || section == w.Section
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return nil, false, fmt.Errorf("line %d: no =", i+1)
		}
		if open(v) {
			pending = line
			continue
		}
		if section == w.Section {
			vals.add(strings.TrimSpace(k), godotValue(strings.TrimSpace(v))...)
		}
	}
	if pending != "" {
		return nil, false, fmt.Errorf("unclosed value: %q", pending)
	}
	return vals, found, nil
}

// open says whether a value still has a parenthesis to close, outside strings.
func open(v string) bool {
	depth, quoted := 0, false
	for i := 0; i < len(v); i++ {
		switch c := v[i]; {
		case quoted && c == '\\':
			i++
		case c == '"':
			quoted = !quoted
		case !quoted && c == '(':
			depth++
		case !quoted && c == ')':
			depth--
		}
	}
	return depth > 0
}

func godotValue(v string) []string {
	if !strings.Contains(v, `"`) {
		return []string{v}
	}
	var out []string
	for _, m := range godotString.FindAllStringSubmatch(v, -1) {
		s, err := strconv.Unquote(`"` + m[1] + `"`)
		if err != nil {
			s = m[1]
		}
		out = append(out, s)
	}
	return out
}
```

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`.

- [ ] **Step 5: Mutationsrunde**, `$S/t7-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t7.txt`:

```
feat(linecov): read INI and Godot project settings
```

```bash
git add internal/verify/linecov/syntax.go internal/verify/linecov/ini.go internal/verify/linecov/godotcfg.go internal/verify/linecov/syntax_test.go
git commit -F "$S/msg-t7.txt"
```

---

### Task 8: TOML, JSON, XML

**Files:**
- Create: `internal/verify/linecov/structured.go`
- Modify: `internal/verify/linecov/syntax.go` (`syntaxes()`)
- Test: `internal/verify/linecov/structured_test.go`

**Interfaces:**
- Consumes: `Values`, `Where`, `splitList` (Task 7)
- Produces: `readTOML`, `readJSON`, `readXML`; `syntaxes()` mit allen fünf

- [ ] **Step 1: Stubs und rote Tests.** Stubs liefern `nil, false, nil`;
  `syntaxes()` um `"toml"`, `"json"`, `"xml"` ergänzt.

```go
package linecov

import (
	"reflect"
	"testing"
)

func TestTOMLReadsADottedTable(t *testing.T) {
	doc := "[tool.coverage.run]\nsource = [\"src\", \"pkg\"]\nbranch = true\nparallel = 4\nmixed = [\"a\", [\"b\"]]\n[tool.coverage.run.paths]\nx = [\"a\"]\n"
	vals, found, err := readTOML([]byte(doc), Where{Table: "tool.coverage.run"})
	want := Values{"source": {"src", "pkg"}, "branch": {"true"}, "parallel": {"4"}, "mixed": {"a", "b"}}
	if err != nil || !found || !reflect.DeepEqual(vals, want) {
		t.Fatalf("%v %v %v", vals, found, err)
	}
	if _, found, err := readTOML([]byte("[project]\nname = \"x\"\n"), Where{Table: "tool.coverage.run"}); found || err != nil {
		t.Fatal("found a table that is not there")
	}
	if _, _, err := readTOML([]byte("[broken"), Where{Table: "run"}); err == nil {
		t.Fatal("no error")
	}
}

func TestJSONReadsTheRootOrAKeyPath(t *testing.T) {
	doc := `{"name":"app","nyc":{"include":["src/**"],"all":true,"n":1.5}}`
	vals, found, err := readJSON([]byte(doc), Where{Path: "nyc"})
	if err != nil || !found || !reflect.DeepEqual(vals, Values{"include": {"src/**"}, "all": {"true"}, "n": {"1.5"}}) {
		t.Fatalf("%v %v %v", vals, found, err)
	}
	root, found, err := readJSON([]byte(`{"exclude":"x"}`), Where{})
	if err != nil || !found || !reflect.DeepEqual(root, Values{"exclude": {"x"}}) {
		t.Fatalf("%v %v %v", root, found, err)
	}
	if _, found, _ := readJSON([]byte(`{"nyc":3}`), Where{Path: "nyc"}); found {
		t.Fatal("a scalar is not a section")
	}
	if _, _, err := readJSON([]byte(`{`), Where{}); err == nil {
		t.Fatal("no error")
	}
}

func TestXMLReadsElementsAndAttributesBelowAPath(t *testing.T) {
	doc := `<?xml version="1.0"?><phpunit><source>
	  <include><directory suffix=".php">src</directory><file>bootstrap.php</file></include>
	  <exclude><directory>src/Legacy</directory></exclude>
	</source></phpunit>`
	vals, found, err := readXML([]byte(doc), Where{Path: "phpunit/source"})
	want := Values{
		"include/directory":         {"src"},
		"include/directory@suffix":  {".php"},
		"include/file":              {"bootstrap.php"},
		"exclude/directory":         {"src/Legacy"},
	}
	if err != nil || !found || !reflect.DeepEqual(vals, want) {
		t.Fatalf("%v %v %v", vals, found, err)
	}
	for _, path := range []string{"other/source", "phpunit/coverage"} {
		if _, found, _ := readXML([]byte(doc), Where{Path: path}); found {
			t.Errorf("%s found", path)
		}
	}
	if _, _, err := readXML([]byte("<a><b></a>"), Where{Path: "a"}); err == nil {
		t.Fatal("no error")
	}
}

func TestXMLMergesEveryElementOnThePathAndSplitsLists(t *testing.T) {
	doc := `<RunSettings><DataCollectionRunSettings><DataCollectors>
	  <DataCollector><Configuration><ExcludeByFile>**/gen/*.cs,**/obj/**</ExcludeByFile></Configuration></DataCollector>
	  <DataCollector><Configuration><ExcludeByFile>x.cs</ExcludeByFile></Configuration></DataCollector>
	</DataCollectors></DataCollectionRunSettings></RunSettings>`
	vals, found, err := readXML([]byte(doc), Where{Path: "RunSettings/DataCollectionRunSettings/DataCollectors/DataCollector/Configuration"})
	if err != nil || !found || !reflect.DeepEqual(vals, Values{"ExcludeByFile": {"**/gen/*.cs", "**/obj/**", "x.cs"}}) {
		t.Fatalf("%v %v %v", vals, found, err)
	}
}
```

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run 'TOML|JSON|XML' -count=1`
  Expected: FAIL.

- [ ] **Step 3: Implementieren.**

```go
package linecov

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/BurntSushi/toml"
)

func readTOML(content []byte, w Where) (Values, bool, error) {
	var doc map[string]any
	if _, err := toml.Decode(text(content), &doc); err != nil {
		return nil, false, err
	}
	return section(doc, splitDots(w.Table))
}

func readJSON(content []byte, w Where) (Values, bool, error) {
	var doc any
	if err := json.Unmarshal([]byte(text(content)), &doc); err != nil {
		return nil, false, err
	}
	return section(doc, splitDots(w.Path))
}

func splitDots(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ".")
}

// section walks keys down nested maps and flattens the scalars and scalar
// lists it finds there; nested tables are not values.
func section(doc any, keys []string) (Values, bool, error) {
	cur, ok := doc.(map[string]any)
	for _, k := range keys {
		if !ok {
			break
		}
		cur, ok = cur[k].(map[string]any)
	}
	if !ok {
		return Values{}, false, nil
	}
	vals := Values{}
	for k, v := range cur {
		vals.add(k, scalars(v)...)
	}
	return vals, true, nil
}

func scalars(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case bool, int64, float64:
		return []string{fmt.Sprint(x)}
	case []any:
		var out []string
		for _, e := range x {
			out = append(out, scalars(e)...)
		}
		return out
	}
	return nil
}

type xmlNode struct {
	name  string
	attrs []xml.Attr
	text  strings.Builder
	kids  []*xmlNode
}

func parseXML(content []byte) (*xmlNode, error) {
	d := xml.NewDecoder(bytes.NewReader([]byte(text(content))))
	top := &xmlNode{}
	stack := []*xmlNode{top}
	for {
		tok, err := d.Token()
		if err != nil {
			// A document that ends inside an element leaves it on the stack.
			if errors.Is(err, io.EOF) && len(stack) == 1 && len(top.kids) == 1 {
				return top.kids[0], nil
			}
			return nil, fmt.Errorf("unreadable XML: %w", err)
		}
		switch x := tok.(type) {
		case xml.StartElement:
			n := &xmlNode{name: x.Name.Local, attrs: x.Attr}
			parent := stack[len(stack)-1]
			parent.kids = append(parent.kids, n)
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			stack[len(stack)-1].text.Write(x)
		}
	}
}

// readXML selects every element on w.Path (slash names from the root
// element) and reads what lies below: a leaf's text under its relative path,
// an attribute as <path>@<name>.
func readXML(content []byte, w Where) (Values, bool, error) {
	root, err := parseXML(content)
	if err != nil {
		return nil, false, err
	}
	steps := strings.Split(w.Path, "/")
	if steps[0] != root.name {
		return Values{}, false, nil
	}
	nodes := []*xmlNode{root}
	for _, step := range steps[1:] {
		var next []*xmlNode
		for _, n := range nodes {
			for _, k := range n.kids {
				if k.name == step {
					next = append(next, k)
				}
			}
		}
		nodes = next
	}
	if len(nodes) == 0 {
		return Values{}, false, nil
	}
	vals := Values{}
	for _, n := range nodes {
		collect(n, "", vals)
	}
	return vals, true, nil
}

func collect(n *xmlNode, prefix string, vals Values) {
	for _, k := range n.kids {
		key := strings.TrimPrefix(prefix+"/"+k.name, "/")
		if len(k.kids) == 0 {
			vals.add(key, splitList(k.text.String())...)
		}
		for _, a := range k.attrs {
			vals.add(key+"@"+a.Name.Local, a.Value)
		}
		collect(k, key, vals)
	}
}
```

  `TestXMLReadsElementsAndAttributesBelowAPath` braucht noch die Zeile für
  ein leeres Dokument (EOF ohne Element):

```go
	if _, _, err := readXML([]byte("  "), Where{Path: "a"}); err == nil {
		t.Fatal("an empty document was read")
	}
```

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`.

- [ ] **Step 5: Mutationsrunde**, `$S/t8-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t8.txt`:

```
feat(linecov): read TOML, JSON and XML settings
```

```bash
git add internal/verify/linecov/structured.go internal/verify/linecov/structured_test.go internal/verify/linecov/syntax.go
git commit -F "$S/msg-t8.txt"
```

---

### Task 9: Musterdialekte

**Files:**
- Create: `internal/verify/linecov/dialect.go`
- Test: `internal/verify/linecov/dialect_test.go`

**Interfaces:**
- Produces:
  - `type Matcher func(rel string) bool`
  - `type Dialect func(pattern, root string) (Matcher, error)` — `root` absolut, mit `/`
  - `func dialects() map[string]Dialect` — `coveragepy`, `minimatch`, `nano`, `lcov`, `regex`, `dir`
  - `func globRegexp(p string, starCrosses bool) string`
  - `func dirDialect(pattern, root string) (Matcher, error)` (Task 11 nutzt ihn direkt)

- [ ] **Step 1: Stubs und rote Tests.** Jeder Dialekt liefert einen Matcher,
  der immer `false` sagt.

```go
package linecov

import "testing"

func TestDialectsMatchTheWayTheirToolsDo(t *testing.T) {
	const root = "/w/app"
	for _, row := range []struct {
		dialect, pattern, rel string
		want                  bool
	}{
		{"nano", "**/*_test.gd", "test/a_test.gd", true},
		{"nano", "**/*_test.gd", "a_test.gd", true},
		{"nano", "res://harness/**", "harness/p.gd", true},
		{"nano", "res://harness/**", "core/harness/p.gd", false},
		{"nano", "addons/**", "addons/x/plugin.gd", false}, // Nano matches full res:// paths
		{"nano", "res://addons/**", "addons/x/plugin.gd", true},
		{"lcov", "*/third_party/*", "src/third_party/x.c", true},
		{"lcov", "third", "src/third_party/x.c", true},
		{"lcov", "/w/app/src/*", "src/a/b.c", true},
		{"lcov", "/other/*", "src/a.c", false},
		{"coveragepy", "skip_*.py", "src/skip_a.py", true},
		{"coveragepy", "src/gen/*", "src/gen/a.py", true},
		{"coveragepy", "src/gen/*", "src/gen/b/c.py", false},
		{"coveragepy", "/w/app/src/*", "src/a.py", true},
		{"coveragepy", "src\\gen\\*", "src/gen/a.py", true},
		{"minimatch", "src/**/*.js", "src/a.js", true},
		{"minimatch", "src/**/*.js", "src/x/y.js", true},
		{"minimatch", "./src/*.js", "src/a.js", true},
		{"minimatch", "src/*.js", "src/x/a.js", false},
		{"minimatch", "a?.js", "ab.js", true},
		{"regex", "src/", "src/a.cpp", true},
		{"regex", "^/w/app/lib/", "lib/a.cpp", true},
		{"regex", "^lib/", "src/lib/a.cpp", false},
		{"dir", "src", "src/a.php", true},
		{"dir", "src", "src", true},
		{"dir", "./src/", "src/a.php", true},
		{"dir", "src", "srcx/a.php", false},
		{"dir", ".", "a.py", true}, // coverage.py's source = . names the whole root
	} {
		m, err := dialects()[row.dialect](row.pattern, root)
		if err != nil {
			t.Errorf("%s %q: %v", row.dialect, row.pattern, err)
			continue
		}
		if got := m(row.rel); got != row.want {
			t.Errorf("%s %q on %q: %v", row.dialect, row.pattern, row.rel, got)
		}
	}
}

func TestDialectsRefuseWhatTheyCannotRead(t *testing.T) {
	for _, row := range []struct{ dialect, pattern string }{
		{"regex", "("},
		{"minimatch", "!src/**"},
	} {
		if _, err := dialects()[row.dialect](row.pattern, "/w"); err == nil {
			t.Errorf("%s %q: no error", row.dialect, row.pattern)
		}
	}
}
```

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run Dialect -count=1`
  Expected: FAIL.

- [ ] **Step 3: Implementieren.**

```go
package linecov

import (
	"errors"
	"path"
	"regexp"
	"strings"
)

// Matcher answers for a slash path relative to the root.
type Matcher func(rel string) bool

// Dialect turns one of a tool's patterns into a Matcher, read the way that
// tool reads it. root is absolute, with slashes. A new dialect is one
// function and one entry in dialects.
type Dialect func(pattern, root string) (Matcher, error)

func dialects() map[string]Dialect {
	return map[string]Dialect{
		"coveragepy": coveragePy,
		"minimatch":  miniMatch,
		"nano":       nanoGlob,
		"lcov":       lcovGlob,
		"regex":      regexDialect,
		"dir":        dirDialect,
	}
}

// globRegexp translates ** * ? into a regular expression; starCrosses lets
// * and ? match a slash as well.
func globRegexp(p string, starCrosses bool) string {
	star, one := "[^/]*", "[^/]"
	if starCrosses {
		star, one = ".*", "."
	}
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		switch {
		case strings.HasPrefix(p[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 2
		case strings.HasPrefix(p[i:], "**"):
			b.WriteString(".*")
			i++
		case p[i] == '*':
			b.WriteString(star)
		case p[i] == '?':
			b.WriteString(one)
		default:
			b.WriteString(regexp.QuoteMeta(p[i : i+1]))
		}
	}
	return b.String()
}

func anchored(p string, starCrosses bool) (*regexp.Regexp, error) {
	return regexp.Compile("^" + globRegexp(p, starCrosses) + "$")
}

// relPattern makes an absolute pattern relative to root.
func relPattern(p, root string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	return strings.TrimPrefix(strings.TrimPrefix(p, root+"/"), "./")
}

// coveragePy: a pattern without a slash matches the file name in any
// directory; any other is relative to the root.
func coveragePy(p, root string) (Matcher, error) {
	p = relPattern(p, root)
	re, err := anchored(p, false)
	if !strings.Contains(p, "/") {
		return func(rel string) bool { return re.MatchString(path.Base(rel)) }, err
	}
	return func(rel string) bool { return re.MatchString(rel) }, err
}

func miniMatch(p, root string) (Matcher, error) {
	if strings.HasPrefix(p, "!") {
		return nil, errors.New("negated patterns are not supported")
	}
	re, err := anchored(relPattern(p, root), false)
	return func(rel string) bool { return re.MatchString(rel) }, err
}

// nanoGlob matches the full res:// path, as Nano Coverage's script scanner does.
func nanoGlob(p, _ string) (Matcher, error) {
	re, err := anchored(p, false)
	return func(rel string) bool { return re.MatchString("res://" + rel) }, err
}

// lcovGlob is lcov's transform_pattern: * crosses slashes, no anchors, and
// the pattern is matched against the absolute path.
func lcovGlob(p, root string) (Matcher, error) {
	re, err := regexp.Compile(globRegexp(p, true))
	return func(rel string) bool { return re.MatchString(root + "/" + rel) }, err
}

// regexDialect (gcovr) tries the path relative to the root and the absolute one.
func regexDialect(p, root string) (Matcher, error) {
	re, err := regexp.Compile(p)
	if err != nil {
		return nil, err
	}
	return func(rel string) bool { return re.MatchString(rel) || re.MatchString(root+"/"+rel) }, nil
}

// dirDialect (PHPUnit, coverage.py's source) names a directory or a file,
// never a pattern; "." is the whole root.
func dirDialect(p, root string) (Matcher, error) {
	d := path.Clean(relPattern(p, root))
	if d == "." {
		return func(string) bool { return true }, nil
	}
	return func(rel string) bool { return rel == d || strings.HasPrefix(rel, d+"/") }, nil
}
```

  Die Glob-Dialekte reichen den Fehler von `anchored` ohne eigenen Zweig
  weiter: der Aufrufer (`Active.build`, Task 11) verwirft den Matcher, sobald
  ein Fehler kommt. Erreichbar ist der Fehler dort praktisch nicht, weil
  `globRegexp` jedes andere Zeichen mit `QuoteMeta` schützt; ein eigener
  Zweig wäre ungetestet.

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`.

- [ ] **Step 5: Mutationsrunde**, `$S/t9-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t9.txt`:

```
feat(linecov): match scope patterns the way each coverage tool reads them
```

```bash
git add internal/verify/linecov/dialect.go internal/verify/linecov/dialect_test.go
git commit -F "$S/msg-t9.txt"
```

---

### Task 10: Werkzeugprofile (`tools.toml`)

**Files:**
- Create: `internal/verify/linecov/tools.go`, `internal/verify/linecov/tools.toml`
- Test: `internal/verify/linecov/tools_test.go`

**Interfaces:**
- Consumes: `syntaxes()` (Task 7/8), `dialects()` (Task 9)
- Produces:
  - `type Profile struct{ Name string; Extensions, Reports []string; Settings []SettingsFile; Include, Exclude KeySet; Scope string; Switches []Switch; Patterns string; Markers, MarkerKeys []MarkerSpec }`
  - `type SettingsFile struct{ File, Syntax, Section, Table, Path string }`
  - `type KeySet struct{ Dirs, Globs, Defaults []string }`
  - `type Switch struct{ Key string; Default bool; Exclude []string }`
  - `type MarkerSpec struct{ Label, Pattern, Key, Template, Block string; NoReason bool }`
  - `type Marker struct{ Label string; Re *regexp.Regexp; NoReason bool; Block string }`
  - `func (s MarkerSpec) compile(value string) (Marker, error)`
  - `func LoadProfiles() ([]Profile, error)`, `func parseProfiles(data string) ([]Profile, error)`

- [ ] **Step 1: Stubs und rote Tests.** `tools.go` mit den Typen,
  `parseProfiles` → `nil, nil`, `compile` → `Marker{}, nil`; `tools.toml`
  leer. Dann `tools_test.go`:

```go
package linecov

import (
	"slices"
	"strings"
	"testing"
)

func TestLoadProfilesReadsTheBuiltInTools(t *testing.T) {
	ps, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range ps {
		names = append(names, p.Name)
	}
	want := []string{"c8", "coveragepy", "coverlet", "gcovr", "jacoco", "kover", "lcov", "nanocoverage", "nyc", "phpunit", "rust", "vitest"}
	if !slices.Equal(names, want) {
		t.Fatalf("%q", names)
	}
}

func TestParseProfilesRefusesWhatWouldMisleadTheGate(t *testing.T) {
	for _, row := range []struct{ doc, want string }{
		{"[tool.a]\nextension = [\".x\"]\n", "unknown key"},
		{"[tool.a]\n", "no extensions"},
		{"[tool.a]\nextensions=[\".x\"]\nscope=\"some\"\n", "unknown scope"},
		{"[tool.a]\nextensions=[\".x\"]\npatterns=\"fnmatch\"\n", "unknown patterns dialect"},
		{"[tool.a]\nextensions=[\".x\"]\nexclude={globs=[\"omit\"]}\n", "globs need a patterns dialect"},
		{"[tool.a]\nextensions=[\".x\"]\nsettings=[{file=\"f\",syntax=\"yaml\"}]\n", "unknown syntax"},
		{"[tool.a]\nextensions=[\".x\"]\nsettings=[{file=\"f\",syntax=\"xml\"}]\n", "xml needs a path"},
		{"[tool.a]\nextensions=[\".x\"]\nmarkers=[{pattern=\"x\"}]\n", "needs a label and a pattern"},
		{"[tool.a]\nextensions=[\".x\"]\nmarkers=[{label=\"x\",pattern=\"(\"}]\n", "marker x"},
		{"[tool.a]\nextensions=[\".x\"]\nmarkers=[{label=\"x\",pattern=\"x\",block=\"middle\"}]\n", "unknown block"},
		{"[tool.a]\nextensions=[\".x\"]\nmarker_keys=[{template=\"{}\"}]\n", "needs a key"},
		{"[tool.a]\nextensions=[\".x\"]\nmarker_keys=[{key=\"k\",block=\"middle\"}]\n", "unknown block"},
		{"[tool.a\n", "tools.toml"},
	} {
		if _, err := parseProfiles(row.doc); err == nil || !strings.Contains(err.Error(), row.want) {
			t.Errorf("%q: %v", row.doc, err)
		}
	}
}

func TestMarkerSpecCompilesALiteralAKeyValueAndATemplate(t *testing.T) {
	for _, row := range []struct {
		spec       MarkerSpec
		value      string
		label, re  string
	}{
		{MarkerSpec{Label: "pragma: no cover", Pattern: `pragma:\s*no cover`}, "", "pragma: no cover", `pragma:\s*no cover`},
		{MarkerSpec{Key: "lcov_excl_line"}, "MY_EXCL", "MY_EXCL", "MY_EXCL"},
		{MarkerSpec{Key: "exclude-pattern-prefix", Template: `\b{}_EXCL_LINE\b`}, "P.X", "P.X_EXCL_LINE", `\bP\.X_EXCL_LINE\b`},
	} {
		m, err := row.spec.compile(row.value)
		if err != nil || m.Label != row.label || m.Re.String() != row.re {
			t.Errorf("%+v %q: %+v %v", row.spec, row.value, m, err)
		}
	}
	if _, err := (MarkerSpec{Key: "k"}).compile("("); err == nil {
		t.Fatal("a broken configured marker compiled")
	}
}
```

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run 'Profiles|MarkerSpec' -count=1`
  Expected: FAIL.

- [ ] **Step 3: Implementieren.** `tools.go`:

```go
package linecov

import (
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

//go:embed tools.toml
var toolsText string

// loadProfiles holds a function, not parsed data: no package variable may
// parse embedded data, and OnceValues parses on the first call only.
var loadProfiles = sync.OnceValues(func() ([]Profile, error) { return parseProfiles(toolsText) })

// LoadProfiles returns the built-in tool profiles, sorted by name.
func LoadProfiles() ([]Profile, error) { return loadProfiles() }

// Profile describes one coverage tool as data: what it measures, where it
// writes, which settings files it reads and what they mean. A tool whose
// settings use a known syntax needs nothing but an entry in tools.toml.
type Profile struct {
	Name       string         `toml:"-"`
	Extensions []string       `toml:"extensions"`
	Reports    []string       `toml:"reports"`
	Settings   []SettingsFile `toml:"settings"`
	Include    KeySet         `toml:"include"`
	Exclude    KeySet         `toml:"exclude"`
	Scope      string         `toml:"scope"`
	Switches   []Switch       `toml:"switches"`
	Patterns   string         `toml:"patterns"`
	Markers    []MarkerSpec   `toml:"markers"`
	MarkerKeys []MarkerSpec   `toml:"marker_keys"`
}

// SettingsFile is one candidate in the tool's own lookup order.
type SettingsFile struct {
	File    string `toml:"file"`
	Syntax  string `toml:"syntax"`
	Section string `toml:"section"`
	Table   string `toml:"table"`
	Path    string `toml:"path"`
}

// KeySet names settings keys: Dirs hold directories or files, Globs hold
// patterns in the profile's dialect, Defaults apply when no glob key is set.
type KeySet struct {
	Dirs     []string `toml:"dirs"`
	Globs    []string `toml:"globs"`
	Defaults []string `toml:"defaults"`
}

// Switch is a boolean key that adds exclusions while it is on.
type Switch struct {
	Key     string   `toml:"key"`
	Default bool     `toml:"default"`
	Exclude []string `toml:"exclude"`
}

// MarkerSpec is a native exclusion marker: a fixed Pattern, or the values of
// a settings Key, optionally put into a Template at {}. Block is "", "start",
// "stop" or "file".
type MarkerSpec struct {
	Label    string `toml:"label"`
	Pattern  string `toml:"pattern"`
	Key      string `toml:"key"`
	Template string `toml:"template"`
	Block    string `toml:"block"`
	NoReason bool   `toml:"no_reason"`
}

// Marker is a compiled MarkerSpec. A group named reason in Re holds the
// reason; without it the text after the match does.
type Marker struct {
	Label    string
	Re       *regexp.Regexp
	NoReason bool
	Block    string
}

func (s MarkerSpec) compile(value string) (Marker, error) {
	pattern, label := s.Pattern, s.Label
	if s.Key != "" {
		pattern, label = value, value
		if s.Template != "" {
			pattern = strings.ReplaceAll(s.Template, "{}", regexp.QuoteMeta(value))
			label = strings.ReplaceAll(strings.ReplaceAll(s.Template, `\b`, ""), "{}", value)
		}
	}
	re, err := regexp.Compile(pattern)
	return Marker{Label: label, Re: re, NoReason: s.NoReason, Block: s.Block}, err
}

func parseProfiles(data string) ([]Profile, error) {
	var doc struct {
		Tool map[string]Profile `toml:"tool"`
	}
	md, err := toml.Decode(data, &doc)
	if err != nil {
		return nil, fmt.Errorf("tools.toml: %w", err)
	}
	if un := md.Undecoded(); len(un) > 0 {
		return nil, fmt.Errorf("tools.toml: unknown key %s", un[0])
	}
	var out []Profile
	for _, name := range slices.Sorted(maps.Keys(doc.Tool)) {
		p := doc.Tool[name]
		p.Name = name
		if err := p.validate(); err != nil {
			return nil, fmt.Errorf("tools.toml: tool %s: %w", name, err)
		}
		out = append(out, p)
	}
	return out, nil
}

func (p Profile) validate() error {
	if len(p.Extensions) == 0 {
		return errors.New("no extensions")
	}
	if p.Scope != "" && p.Scope != "all" {
		return fmt.Errorf("unknown scope %q", p.Scope)
	}
	if _, ok := dialects()[p.Patterns]; p.Patterns != "" && !ok {
		return fmt.Errorf("unknown patterns dialect %q", p.Patterns)
	}
	globs := len(p.Include.Globs) + len(p.Include.Defaults) + len(p.Exclude.Globs) + len(p.Exclude.Defaults) + len(p.Switches)
	if globs > 0 && p.Patterns == "" {
		return errors.New("globs need a patterns dialect")
	}
	for _, s := range p.Settings {
		if _, ok := syntaxes()[s.Syntax]; !ok {
			return fmt.Errorf("settings %s: unknown syntax %q", s.File, s.Syntax)
		}
		if s.Syntax == "xml" && s.Path == "" {
			return fmt.Errorf("settings %s: xml needs a path", s.File)
		}
	}
	for _, s := range p.Markers {
		if s.Label == "" || s.Pattern == "" {
			return errors.New("a marker needs a label and a pattern")
		}
		if _, err := s.compile(""); err != nil {
			return fmt.Errorf("marker %s: %w", s.Label, err)
		}
	}
	for _, s := range append(slices.Clone(p.MarkerKeys), p.Markers...) {
		if !slices.Contains([]string{"", "start", "stop", "file"}, s.Block) {
			return fmt.Errorf("unknown block %q", s.Block)
		}
	}
	for _, s := range p.MarkerKeys {
		if s.Key == "" {
			return errors.New("a marker key needs a key")
		}
	}
	return nil
}
```

  Die Reihenfolge der Prüfungen in `validate` muss die Testtabelle tragen:
  die Block-Prüfung über `Markers` läuft nach der Compile-Prüfung, aber vor
  der Key-Prüfung; die Tabellenzeile `marker_keys=[{template="{}"}]` erwartet
  „needs a key“ und hat einen gültigen leeren Block.

`tools.toml`:

```toml
# Coverage tools as data. Each entry says which files the tool measures,
# where it writes its report, which settings files it reads (in its own
# lookup order) and which keys there define its scope and markers. The
# lint test in tools_fixtures_test.go demands a fixture project for every
# entry under testdata/tools/<name>/.

[tool.coveragepy]
extensions = [".py"]
reports = ["coverage.xml", "coverage.lcov"]
settings = [
  { file = ".coveragerc", syntax = "ini", section = "run" },
  { file = ".coveragerc.toml", syntax = "toml", table = "run" },
  { file = ".coveragerc.toml", syntax = "toml", table = "tool.coverage.run" },
  { file = "setup.cfg", syntax = "ini", section = "coverage:run" },
  { file = "tox.ini", syntax = "ini", section = "coverage:run" },
  { file = "pyproject.toml", syntax = "toml", table = "tool.coverage.run" },
]
include = { dirs = ["source", "source_dirs"], globs = ["include"] }
exclude = { globs = ["omit"] }
patterns = "coveragepy"
markers = [
  { label = "pragma: no cover", pattern = '#\s*(?:pragma|PRAGMA)[:\s]?\s*(?:no|NO)\s*(?:cover|COVER)' },
]

[tool.coverlet]
extensions = [".cs"]
reports = ["coverage.cobertura.xml", "coverage.info"]
settings = [
  { file = "*.runsettings", syntax = "xml", path = "RunSettings/DataCollectionRunSettings/DataCollectors/DataCollector/Configuration" },
]
exclude = { globs = ["ExcludeByFile"] }
patterns = "minimatch"
markers = [
  { label = "ExcludeFromCodeCoverage", pattern = '\[\s*(?:assembly:\s*)?ExcludeFrom(?:Code)?Coverage(?:Attribute)?\s*(?:\(\s*(?:Justification\s*=\s*"(?P<reason>[^"]*)")?\s*\))?\s*\]' },
]

[tool.gcovr]
extensions = [".c", ".cc", ".cpp", ".cxx", ".h", ".hh", ".hpp", ".hxx"]
settings = [{ file = "gcovr.cfg", syntax = "ini" }]
include = { globs = ["filter"] }
exclude = { globs = ["exclude"] }
patterns = "regex"
markers = [
  { label = "GCOVR_EXCL_LINE", pattern = '\b(?:GCOVR|GCOV)_EXCL_LINE\b' },
  { label = "GCOVR_EXCL_START", pattern = '\b(?:GCOVR|GCOV)_EXCL_START\b', block = "start" },
  { label = "GCOVR_EXCL_STOP", pattern = '\b(?:GCOVR|GCOV)_EXCL_STOP\b', block = "stop", no_reason = true },
]
marker_keys = [
  { key = "exclude-pattern-prefix", template = '\b{}_EXCL_LINE\b' },
  { key = "exclude-pattern-prefix", template = '\b{}_EXCL_START\b', block = "start" },
  { key = "exclude-pattern-prefix", template = '\b{}_EXCL_STOP\b', block = "stop", no_reason = true },
]

[tool.lcov]
extensions = [".c", ".cc", ".cpp", ".cxx", ".h", ".hh", ".hpp", ".hxx"]
settings = [{ file = ".lcovrc", syntax = "ini" }]
include = { globs = ["include"] }
exclude = { globs = ["exclude"] }
patterns = "lcov"
markers = [
  { label = "LCOV_EXCL_LINE", pattern = '\bLCOV_EXCL_LINE\b' },
  { label = "LCOV_EXCL_START", pattern = '\bLCOV_EXCL_START\b', block = "start" },
  { label = "LCOV_EXCL_STOP", pattern = '\bLCOV_EXCL_STOP\b', block = "stop", no_reason = true },
]
marker_keys = [
  { key = "lcov_excl_line" },
  { key = "lcov_excl_start", block = "start" },
  { key = "lcov_excl_stop", block = "stop", no_reason = true },
]

[tool.rust]
extensions = [".rs"]
markers = [
  { label = "coverage(off)", pattern = '#\[\s*(?:cfg_attr\s*\([^,]*,\s*)?coverage\s*\(\s*off\s*\)\s*\)?\s*\]' },
]

[tool.phpunit]
extensions = [".php"]
settings = [
  { file = "phpunit.xml", syntax = "xml", path = "phpunit/source" },
  { file = "phpunit.dist.xml", syntax = "xml", path = "phpunit/source" },
  { file = "phpunit.xml.dist", syntax = "xml", path = "phpunit/source" },
]
include = { dirs = ["include/directory", "include/file"] }
exclude = { dirs = ["exclude/directory", "exclude/file"] }
markers = [
  { label = "@codeCoverageIgnore", pattern = '@codeCoverageIgnore(?:[^A-Za-z]|$)' },
  { label = "@codeCoverageIgnoreStart", pattern = '@codeCoverageIgnoreStart\b', block = "start" },
  { label = "@codeCoverageIgnoreEnd", pattern = '@codeCoverageIgnoreEnd\b', block = "stop", no_reason = true },
]

[tool.nyc]
extensions = [".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx"]
reports = ["coverage/lcov.info", "coverage/cobertura-coverage.xml"]
settings = [
  { file = ".nycrc", syntax = "json" },
  { file = ".nycrc.json", syntax = "json" },
  { file = "package.json", syntax = "json", path = "nyc" },
]
include = { globs = ["include"] }
exclude = { globs = ["exclude"] }
patterns = "minimatch"
markers = [
  { label = "istanbul ignore", pattern = 'istanbul\s+ignore\s+(?:if|else|next)\b' },
  { label = "istanbul ignore file", pattern = 'istanbul\s+ignore\s+file\b', block = "file" },
]

[tool.c8]
extensions = [".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx"]
reports = ["coverage/lcov.info", "coverage/cobertura-coverage.xml"]
settings = [
  { file = ".c8rc", syntax = "json" },
  { file = ".c8rc.json", syntax = "json" },
  { file = "package.json", syntax = "json", path = "c8" },
]
include = { globs = ["include"] }
exclude = { globs = ["exclude"] }
patterns = "minimatch"
markers = [
  { label = "c8 ignore next", pattern = 'c8\s+ignore\s+next\b' },
  { label = "c8 ignore start", pattern = 'c8\s+ignore\s+start\b', block = "start" },
  { label = "c8 ignore stop", pattern = 'c8\s+ignore\s+stop\b', block = "stop", no_reason = true },
  { label = "c8 ignore file", pattern = 'c8\s+ignore\s+file\b', block = "file" },
]

# vitest keeps its settings in code (vitest.config.ts), which is not read:
# the profile brings report locations and the v8 markers only.
[tool.vitest]
extensions = [".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".vue", ".svelte"]
reports = ["coverage/lcov.info", "coverage/cobertura-coverage.xml"]
markers = [
  { label = "v8 ignore", pattern = 'v8\s+ignore\s+(?:next|if|else)\b' },
  { label = "v8 ignore start", pattern = 'v8\s+ignore\s+start\b', block = "start" },
  { label = "v8 ignore stop", pattern = 'v8\s+ignore\s+stop\b', block = "stop", no_reason = true },
  { label = "v8 ignore file", pattern = 'v8\s+ignore\s+file\b', block = "file" },
]

# Gradle and Maven keep JaCoCo's settings in build code; the profile brings
# report locations only. Both Gradle locations are best known, not documented.
[tool.jacoco]
extensions = [".java", ".kt"]
reports = ["target/site/jacoco/jacoco.xml", "build/reports/jacoco/test/jacocoTestReport.xml"]

[tool.kover]
extensions = [".kt", ".java"]
reports = ["build/reports/kover/report.xml"]

[tool.nanocoverage]
extensions = [".gd"]
reports = ["coverage-report/lcov.info"]
settings = [{ file = "project.godot", syntax = "godotcfg", section = "nano_coverage" }]
scope = "all"
exclude = { globs = ["general/ignore_paths"], defaults = ["**/*_test.gd"] }
switches = [{ key = "general/ignore_addons", default = true, exclude = ["res://addons/**"] }]
patterns = "nano"
```

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`.

- [ ] **Step 5: Mutationsrunde**, `$S/t10-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t10.txt`:

```
feat(linecov): describe coverage tools as data in tools.toml
```

```bash
git add internal/verify/linecov/tools.go internal/verify/linecov/tools.toml internal/verify/linecov/tools_test.go
git commit -F "$S/msg-t10.txt"
```

---

### Task 11: Profile aktivieren, Bereich, native Marker sammeln

**Files:**
- Create: `internal/verify/linecov/activate.go`
- Test: `internal/verify/linecov/activate_test.go`

**Interfaces:**
- Consumes: `Tree` (6), `Values`/`syntaxes` (7, 8), `dialects`/`dirDialect` (9), `Profile`/`MarkerSpec.compile` (10)
- Produces:
  - `type Active struct{ Profile Profile; Settings string; include, exclude []Matcher; markers []Marker }`
  - `func (a Active) Owns(rel string) bool`, `Scoped() bool`, `Excludes(rel string) bool`, `Expects(rel string) bool`
  - `func Activate(t *Tree, reports []string, ps []Profile) ([]Active, []Finding, []string)` — `reports`: gelesene Berichte relativ zum Root; Rückgabe: aktive Profile, Befunde, Hinweise
  - `func Natives(ps []Profile, act []Active) []Marker`

- [ ] **Step 1: Stubs und rote Tests.** Stubs liefern Nullwerte.

```go
package linecov

import (
	"slices"
	"strings"
	"testing"
)

func activeIn(t *testing.T, files map[string]string, reports ...string) ([]Active, []Finding, []string, *Tree) {
	t.Helper()
	tree, err := ScanTree(writeTree(t, files))
	if err != nil {
		t.Fatal(err)
	}
	ps, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	act, finds, notes := Activate(tree, reports, ps)
	return act, finds, notes, tree
}

func names(act []Active) []string {
	var out []string
	for _, a := range act {
		out = append(out, a.Profile.Name)
	}
	return out
}

func TestActivateReadsCoveragePyScopeFromTheFirstFileThatConfiguresIt(t *testing.T) {
	act, finds, _, _ := activeIn(t, map[string]string{
		// Review focus 5: setup.cfg comes first in the order but holds no
		// coverage section, so the later pyproject.toml is read.
		"setup.cfg":      "[metadata]\nname = x\n",
		"pyproject.toml": "[tool.coverage.run]\nsource = [\"src\"]\nomit = [\"src/gen/*\"]\n",
		"src/a.py":       "", "src/gen/b.py": "", "tools/c.py": "",
	})
	if len(finds) != 0 || !slices.Equal(names(act), []string{"coveragepy"}) || act[0].Settings != "pyproject.toml" {
		t.Fatalf("%v %v", names(act), finds)
	}
	a := act[0]
	for rel, want := range map[string]bool{"src/a.py": true, "src/gen/b.py": false, "tools/c.py": false, "src/a.txt": false} {
		if a.Expects(rel) != want {
			t.Errorf("%s: %v", rel, !want)
		}
	}
}

func TestActivateTakesNanoCoverageDefaultsAndSwitches(t *testing.T) {
	act, _, _, _ := activeIn(t, map[string]string{
		"project.godot": "[nano_coverage]\ngeneral/ignore_paths=PackedStringArray(\"**/*_test.gd\", \"res://harness/**\")\n",
		"core/a.gd": "", "harness/p.gd": "", "addons/x/y.gd": "", "test/a_test.gd": "",
	})
	a := act[0]
	if !a.Scoped() || a.Settings != "project.godot" {
		t.Fatalf("%+v", a)
	}
	for rel, want := range map[string]bool{"core/a.gd": true, "harness/p.gd": false, "addons/x/y.gd": false, "test/a_test.gd": false} {
		if a.Expects(rel) != want {
			t.Errorf("%s: %v", rel, !want)
		}
	}
	off, _, _, _ := activeIn(t, map[string]string{"project.godot": "[nano_coverage]\ngeneral/ignore_addons=false\n", "addons/x/y.gd": "", "a_test.gd": ""})
	if !off[0].Expects("addons/x/y.gd") || off[0].Expects("a_test.gd") {
		t.Fatal("switch off or defaults not applied")
	}
}

func TestActivateFindsAToolByItsReportAloneAndSaysWhenItHasNoScope(t *testing.T) {
	act, _, _, _ := activeIn(t, map[string]string{"coverage/lcov.info": "", "src/a.ts": ""}, "coverage/lcov.info")
	if !slices.Equal(names(act), []string{"c8", "nyc", "vitest"}) {
		t.Fatalf("%v", names(act))
	}
	for _, a := range act {
		if a.Scoped() || a.Expects("src/a.ts") || !a.Owns("src/a.ts") {
			t.Errorf("%s: scoped without settings", a.Profile.Name)
		}
	}
}

func TestActivateReportsUnreadableSettingsPatternsAndAmbiguousFiles(t *testing.T) {
	_, finds, _, _ := activeIn(t, map[string]string{".coveragerc": "[run\n"})
	if len(finds) != 1 || finds[0].Path != ".coveragerc" || !strings.HasPrefix(finds[0].Text, "unreadable settings:") {
		t.Fatalf("%v", finds)
	}
	_, finds, _, _ = activeIn(t, map[string]string{"gcovr.cfg": "filter = (\n"})
	if len(finds) != 1 || !strings.Contains(finds[0].Text, `unreadable pattern "("`) {
		t.Fatalf("%v", finds)
	}
	_, finds, _, _ = activeIn(t, map[string]string{".lcovrc": "lcov_excl_line = (\n"})
	if len(finds) != 1 || !strings.Contains(finds[0].Text, "unreadable marker") {
		t.Fatalf("%v", finds)
	}
	act, _, notes, _ := activeIn(t, map[string]string{"a.runsettings": "", "b.runsettings": ""})
	if len(act) != 0 || len(notes) != 1 || !strings.Contains(notes[0], "2 files match *.runsettings") {
		t.Fatalf("%v %q", names(act), notes)
	}
}

func TestActivateReadsDirectoriesAndConfiguredMarkers(t *testing.T) {
	act, finds, _, _ := activeIn(t, map[string]string{
		"phpunit.xml": `<phpunit><source><include><directory suffix=".php">src</directory></include><exclude><directory>src/Legacy</directory></exclude></source></phpunit>`,
		"gcovr.cfg":   "exclude-pattern-prefix = MYPROJ\n",
	})
	if len(finds) != 0 {
		t.Fatal(finds)
	}
	var php, gcovr Active
	for _, a := range act {
		switch a.Profile.Name {
		case "phpunit":
			php = a
		case "gcovr":
			gcovr = a
		}
	}
	if !php.Expects("src/A.php") || php.Expects("src/Legacy/B.php") || php.Expects("lib/C.php") {
		t.Fatal("phpunit scope")
	}
	ps, _ := LoadProfiles()
	var labels []string
	for _, m := range Natives(ps, []Active{gcovr, gcovr}) {
		labels = append(labels, m.Label)
	}
	for _, want := range []string{"MYPROJ_EXCL_LINE", "MYPROJ_EXCL_START", "MYPROJ_EXCL_STOP", "LCOV_EXCL_LINE", "pragma: no cover"} {
		if !slices.Contains(labels, want) {
			t.Errorf("missing %s in %q", want, labels)
		}
	}
	if len(labels) != len(slices.Compact(slices.Sorted(slices.Values(labels)))) {
		t.Errorf("duplicate markers: %q", labels)
	}
}
```

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run 'Activate' -count=1`
  Expected: FAIL.

- [ ] **Step 3: Implementieren.**

```go
package linecov

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
)

// Active is a profile the project uses, with its scope and configured
// markers read from its settings file.
type Active struct {
	Profile  Profile
	Settings string // settings file read, relative to the root; "" without one
	include  []Matcher
	exclude  []Matcher
	markers  []Marker
}

func (a Active) Owns(rel string) bool { return slices.Contains(a.Profile.Extensions, path.Ext(rel)) }

// Scoped says whether the profile knows which files must be measured.
func (a Active) Scoped() bool { return a.Profile.Scope == "all" || len(a.include) > 0 }

func (a Active) Excludes(rel string) bool {
	return slices.ContainsFunc(a.exclude, func(m Matcher) bool { return m(rel) })
}

func (a Active) Expects(rel string) bool {
	if !a.Owns(rel) || !a.Scoped() || a.Excludes(rel) {
		return false
	}
	return a.Profile.Scope == "all" || slices.ContainsFunc(a.include, func(m Matcher) bool { return m(rel) })
}

// Activate finds the profiles the project uses: by a settings file in the
// root, or by a report read from one of the profile's default locations.
func Activate(t *Tree, reports []string, ps []Profile) ([]Active, []Finding, []string) {
	var act []Active
	var finds []Finding
	var notes []string
	root := filepath.ToSlash(t.Root)
	for _, p := range ps {
		vals, file, f, n := readSettings(t.Root, p)
		finds, notes = append(finds, f...), append(notes, n...)
		byReport := slices.ContainsFunc(p.Reports, func(r string) bool { return slices.Contains(reports, r) })
		if file == "" && !byReport {
			continue
		}
		a := Active{Profile: p, Settings: file}
		finds = append(finds, a.build(vals, root)...)
		act = append(act, a)
	}
	return act, finds, notes
}

// readSettings follows the profile's lookup order; the first file holding
// the profile's section wins.
func readSettings(root string, p Profile) (Values, string, []Finding, []string) {
	for _, sf := range p.Settings {
		matches, _ := filepath.Glob(filepath.Join(root, sf.File))
		switch {
		case len(matches) == 0:
			continue
		case len(matches) > 1:
			return Values{}, "", nil, []string{fmt.Sprintf("%s: %d files match %s; none is read", p.Name, len(matches), sf.File)}
		}
		rel := filepath.Base(matches[0])
		content, err := os.ReadFile(matches[0])
		if err != nil {
			return Values{}, "", []Finding{{rel, 0, "unreadable settings: " + err.Error()}}, nil
		}
		vals, found, err := syntaxes()[sf.Syntax](content, Where{Section: sf.Section, Table: sf.Table, Path: sf.Path})
		if err != nil {
			return Values{}, "", []Finding{{rel, 0, "unreadable settings: " + err.Error()}}, nil
		}
		if found {
			return vals, rel, nil, nil
		}
	}
	return Values{}, "", nil, nil
}

func (a *Active) build(vals Values, root string) []Finding {
	p := a.Profile
	var finds []Finding
	add := func(list *[]Matcher, d Dialect, pats []string) {
		for _, pat := range pats {
			m, err := d(pat, root)
			if err != nil {
				finds = append(finds, Finding{a.Settings, 0, fmt.Sprintf("unreadable pattern %q: %v", pat, err)})
				continue
			}
			*list = append(*list, m)
		}
	}
	dialect := dialects()[p.Patterns]
	for _, k := range p.Include.Dirs {
		add(&a.include, dirDialect, vals[k])
	}
	add(&a.include, dialect, pick(vals, p.Include.Globs, p.Include.Defaults))
	for _, k := range p.Exclude.Dirs {
		add(&a.exclude, dirDialect, vals[k])
	}
	add(&a.exclude, dialect, pick(vals, p.Exclude.Globs, p.Exclude.Defaults))
	for _, s := range p.Switches {
		on := s.Default
		if v := vals[s.Key]; len(v) > 0 {
			on = v[len(v)-1] == "true"
		}
		if on {
			add(&a.exclude, dialect, s.Exclude)
		}
	}
	for _, spec := range p.MarkerKeys {
		for _, v := range vals[spec.Key] {
			m, err := spec.compile(v)
			if err != nil {
				finds = append(finds, Finding{a.Settings, 0, fmt.Sprintf("unreadable marker %q: %v", v, err)})
				continue
			}
			a.markers = append(a.markers, m)
		}
	}
	return finds
}

// pick returns the values of keys, or defaults when none of them is set.
func pick(vals Values, keys, defaults []string) []string {
	var out []string
	for _, k := range keys {
		out = append(out, vals[k]...)
	}
	if len(out) == 0 {
		return defaults
	}
	return out
}

// Natives is every built-in marker of every profile, so an LCOV_EXCL_LINE
// needs no .lcovrc to count, plus the markers active settings configure.
func Natives(ps []Profile, act []Active) []Marker {
	var out []Marker
	seen := map[string]bool{}
	add := func(m Marker) {
		if k := m.Block + "\x00" + m.Re.String(); !seen[k] {
			seen[k] = true
			out = append(out, m)
		}
	}
	for _, p := range ps {
		for _, s := range p.Markers {
			// validate compiled every built-in marker already.
			m, _ := s.compile("")
			add(m)
		}
	}
	for _, a := range act {
		for _, m := range a.markers {
			add(m)
		}
	}
	return out
}
```

  `dialect` ist `nil` für ein Profil ohne `patterns`; `validate` stellt
  sicher, dass es dann keine Globs gibt, und `add` ruft `d` nur in der
  Schleife über eine leere Liste. Im Test der gcovr-Datei mit einem leeren
  `filter` greift das ebenfalls.

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`. Den Zweig, in dem
  `os.ReadFile` scheitert, deckt dieser Test in `activate_test.go`:

```go
func TestActivateReportsASettingsFileItCannotRead(t *testing.T) {
	// project.godot is a directory here: it matches the name, it cannot be read.
	_, finds, _, _ := activeIn(t, map[string]string{"project.godot/x": ""})
	if len(finds) != 1 || finds[0].Path != "project.godot" || !strings.HasPrefix(finds[0].Text, "unreadable settings:") {
		t.Fatalf("%v", finds)
	}
}
```

- [ ] **Step 5: Mutationsrunde**, `$S/t11-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t11.txt`:

```
feat(linecov): read scope and markers from the coverage tools' own settings
```

```bash
git add internal/verify/linecov/activate.go internal/verify/linecov/activate_test.go
git commit -F "$S/msg-t11.txt"
```

---

### Task 12: Marker in Quellen

**Files:**
- Create: `internal/verify/linecov/markers.go`
- Test: `internal/verify/linecov/markers_test.go`

**Interfaces:**
- Consumes: `Marker` (10), `Finding`, `text` (1)
- Produces:
  - `type exemption struct{ own, reason bool }`
  - `type Scan struct{ File, FileReason, NativeFile bool; Lines map[int]exemption; Findings []Finding }`
  - `func ScanSource(rel string, src []byte, natives []Marker) Scan`
  - `func cleanReason(s string) string`, `func commentOnly(line string) bool`

- [ ] **Step 1: Stubs und rote Tests.** `ScanSource` liefert `Scan{}`,
  `cleanReason` `""`, `commentOnly` `false`.

```go
package linecov

import (
	"reflect"
	"testing"
)

func natives(t *testing.T) []Marker {
	t.Helper()
	ps, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	return Natives(ps, nil)
}

func TestScanFindsOwnMarkersOnTheLineAndAboveIt(t *testing.T) {
	src := "a = 1  # coverage:exempt only on Windows\n" +
		"# coverage:exempt the line below is generated\n" +
		"b = 2\n" +
		"c = 3 // coverage:exempt\n" +
		"d = 4 /* coverage:exempt  */\n" +
		"e = 5 # coverage:exemption is not a marker\n"
	s := ScanSource("a.py", []byte(src), nil)
	want := map[int]exemption{1: {true, true}, 3: {true, true}, 4: {true, false}, 5: {true, false}}
	if !reflect.DeepEqual(s.Lines, want) || s.File || len(s.Findings) != 0 {
		t.Fatalf("%+v", s)
	}
}

func TestScanFindsTheFileMarker(t *testing.T) {
	if s := ScanSource("a.gd", []byte("extends Node\n# coverage:exempt-file only _init, never instrumented\n"), nil); !s.File || !s.FileReason {
		t.Fatalf("%+v", s)
	}
	s := ScanSource("a.gd", []byte("# coverage:exempt-file\n"), nil)
	if !s.File || s.FileReason || !reflect.DeepEqual(s.Findings, []Finding{{"a.gd", 0, "coverage:exempt-file without a reason"}}) {
		t.Fatalf("%+v", s)
	}
}

func TestScanAsksNativeMarkersForAReasonAfterOrAbove(t *testing.T) {
	src := "x = 1  # pragma: no cover\n" + // 1: no reason
		"y = 2  # pragma: no cover -- debug only\n" + // 2
		"// the extension is tested upstream\n" + // 3
		"function two() { return 2; } // @codeCoverageIgnore\n" + // 4: reason above
		"[ExcludeFromCodeCoverage(Justification = \"generated\")]\n" + // 5: reason in the group
		"[ExcludeFromCodeCoverage]  // debugging aid\n" + // 6: reason after
		"/* LCOV_EXCL_START legacy */\n" + // 7
		"old();\n" + // 8
		"/* LCOV_EXCL_STOP */\n" + // 9
		"/* istanbul ignore file: generated */\n" // 10
	s := ScanSource("m.x", []byte(src), natives(t))
	if !reflect.DeepEqual(s.Findings, []Finding{{"m.x", 1, "pragma: no cover without a reason"}}) {
		t.Fatalf("%v", s.Findings)
	}
	for _, n := range []int{1, 2, 4, 5, 6, 7, 8, 9} {
		if e, ok := s.Lines[n]; !ok || e.own {
			t.Errorf("line %d: %+v %v", n, e, ok)
		}
	}
	if !s.NativeFile || s.File {
		t.Fatalf("%+v", s)
	}
}

func TestScanCoversTheLineBelowANativeMarkerOnAPureCommentLine(t *testing.T) {
	s := ScanSource("a.js", []byte("/* istanbul ignore next: flag */\nexport const two = () => 2;\n"), natives(t))
	if _, ok := s.Lines[2]; !ok || len(s.Findings) != 0 {
		t.Fatalf("%+v", s)
	}
}

func TestScanReportsAStartWithoutAStopAndRunsItToTheEnd(t *testing.T) {
	s := ScanSource("a.c", []byte("// LCOV_EXCL_START legacy\nx();\ny();"), natives(t))
	if !reflect.DeepEqual(s.Findings, []Finding{{"a.c", 1, "LCOV_EXCL_START without a stop"}}) {
		t.Fatalf("%v", s.Findings)
	}
	if _, ok := s.Lines[3]; !ok {
		t.Fatal("the range did not run to the end")
	}
	if s := ScanSource("a.c", []byte("x();\n// LCOV_EXCL_STOP\n"), natives(t)); len(s.Findings) != 0 || len(s.Lines) != 0 {
		t.Fatalf("a stop without a start did something: %+v", s)
	}
}

// Review focus 3: CRLF and a marker on the last line without a newline.
func TestScanReadsCRLFAndALastLineWithoutANewline(t *testing.T) {
	s := ScanSource("a.py", []byte("a = 1\r\nb = 2  # coverage:exempt windows only"), nil)
	if e, ok := s.Lines[2]; !ok || !e.reason {
		t.Fatalf("%+v", s)
	}
}

func TestCleanReasonStripsCommentSyntax(t *testing.T) {
	for in, want := range map[string]string{
		"  -- debug only */": "debug only",
		": flag":             "flag",
		"*/":                 "",
		"// reason":          "reason",
		" * docblock line":   "docblock line",
		"<!-- html -->":      "html",
		"":                   "",
	} {
		if got := cleanReason(in); got != want {
			t.Errorf("%q: %q", in, got)
		}
	}
}

func TestCommentOnlyKnowsTheCommonCommentTokens(t *testing.T) {
	for _, line := range []string{"# a", "  // a", "/* a", " * a", "-- a", "; a", "' a", "% a", "<!-- a"} {
		if !commentOnly(line) {
			t.Errorf("%q", line)
		}
	}
	for _, line := range []string{"x = 1 # a", "", "f()"} {
		if commentOnly(line) {
			t.Errorf("%q", line)
		}
	}
}
```

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run 'Scan|CleanReason|CommentOnly' -count=1`
  Expected: FAIL.

- [ ] **Step 3: Implementieren.**

```go
package linecov

import (
	"regexp"
	"strings"
)

// exemption is what makes a line exempt: loomux's own marker (own), or a
// tool's native one; reason says whether the own marker carries one.
type exemption struct {
	own    bool
	reason bool
}

// Scan is what one source file says about its exemptions.
type Scan struct {
	File       bool // coverage:exempt-file
	FileReason bool
	NativeFile bool // a tool's whole-file marker such as "istanbul ignore file"
	Lines      map[int]exemption
	Findings   []Finding
}

var ownMarker = regexp.MustCompile(`coverage:exempt(-file)?([^\w-].*)?$`)

// commentTokens start a comment in the languages the profiles cover;
// longer tokens come before their prefixes.
var commentTokens = []string{"<!--", "//", "/*", "#", "--", ";", "'", "%", "*"}

// ScanSource reads the exemptions of one file. A marker on a line that holds
// nothing but a comment covers the line below it as well.
func ScanSource(rel string, src []byte, natives []Marker) Scan {
	lines := strings.Split(text(src), "\n")
	s := Scan{Lines: map[int]exemption{}}
	openAt, openLabel := 0, ""
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		n := i + 1
		if m := ownMarker.FindStringSubmatch(line); m != nil {
			reason := cleanReason(m[2]) != ""
			if m[1] != "" {
				s.File, s.FileReason = true, s.FileReason || reason
				if !reason {
					s.Findings = append(s.Findings, Finding{rel, 0, "coverage:exempt-file without a reason"})
				}
			} else {
				s.Lines[target(line, n)] = exemption{own: true, reason: reason}
			}
		}
		for _, mk := range natives {
			loc := mk.Re.FindStringSubmatchIndex(line)
			if loc == nil {
				continue
			}
			if !mk.NoReason && reasonOf(mk, line, loc) == "" && !commentAbove(lines, i) {
				s.Findings = append(s.Findings, Finding{rel, n, mk.Label + " without a reason"})
			}
			switch mk.Block {
			case "":
				s.native(n)
				s.native(target(line, n))
			case "file":
				s.NativeFile = true
			case "start":
				if openAt == 0 {
					openAt, openLabel = n, mk.Label
				}
			case "stop":
				for l := openAt; openAt > 0 && l <= n; l++ {
					s.native(l)
				}
				openAt = 0
			}
		}
	}
	if openAt > 0 {
		s.Findings = append(s.Findings, Finding{rel, openAt, openLabel + " without a stop"})
		for l := openAt; l <= len(lines); l++ {
			s.native(l)
		}
	}
	return s
}

// native marks a line exempt by a tool's marker unless loomux's own marker
// already rules it: only the own marker is refuted by a hit.
func (s Scan) native(n int) {
	if _, ok := s.Lines[n]; !ok {
		s.Lines[n] = exemption{}
	}
}

func target(line string, n int) int {
	if commentOnly(line) {
		return n + 1
	}
	return n
}

func reasonOf(mk Marker, line string, loc []int) string {
	if i := mk.Re.SubexpIndex("reason"); i > 0 && loc[2*i] >= 0 {
		if r := cleanReason(line[loc[2*i]:loc[2*i+1]]); r != "" {
			return r
		}
	}
	return cleanReason(line[loc[1]:])
}

// commentAbove says whether the line above holds a comment with text, which
// counts as the reason of a native marker that cannot carry one itself.
func commentAbove(lines []string, i int) bool {
	return i > 0 && commentOnly(lines[i-1]) && cleanReason(strings.TrimRight(lines[i-1], "\r")) != ""
}

func commentOnly(line string) bool {
	t := strings.TrimSpace(line)
	for _, tok := range commentTokens {
		if strings.HasPrefix(t, tok) {
			return true
		}
	}
	return false
}

// cleanReason strips comment syntax and separators around a reason.
func cleanReason(s string) string {
	s = strings.TrimSpace(s)
	for _, end := range []string{"*/", "-->"} {
		s = strings.TrimSpace(strings.TrimSuffix(s, end))
	}
	for _, tok := range commentTokens {
		if strings.HasPrefix(s, tok) {
			s = s[len(tok):]
			break
		}
	}
	return strings.TrimSpace(strings.TrimLeft(s, ":-– \t"))
}
```

  Die erste Erwartung in
  `TestScanAsksNativeMarkersForAReasonAfterOrAbove` prüft die Zeilen 1–9;
  Zeile 10 ist nur der Dateimarker. Zeile 3 (`// the extension …`) ist eine
  reine Kommentarzeile ohne Marker und erzeugt keinen Eintrag.

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`.

- [ ] **Step 5: Mutationsrunde**, `$S/t12-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t12.txt`:

```
feat(linecov): find exemption markers and their reasons in source files
```

```bash
git add internal/verify/linecov/markers.go internal/verify/linecov/markers_test.go
git commit -F "$S/msg-t12.txt"
```

---

### Task 13: Regeln (`Gate`)

**Files:**
- Create: `internal/verify/linecov/gate.go`
- Test: `internal/verify/linecov/gate_test.go`

**Interfaces:**
- Consumes: `Report`, `Finding`, `SortFindings` (1), `Tree` (6), `Active` (11), `Marker` (10), `ScanSource` (12)
- Produces:
  - `type Input struct{ Report Report; Tree *Tree; Active []Active; Natives []Marker; Newest time.Time; Read func(rel string) ([]byte, error); ModTime func(rel string) (time.Time, error) }`
  - `func Gate(in Input) []Finding`
  - `const mtimeSlack = 2 * time.Second`

- [ ] **Step 1: Stub und rote Tests.** `Gate` liefert `nil`.

```go
package linecov

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func gateWorld(t *testing.T, files map[string]string, report Report, settingsReports ...string) Input {
	t.Helper()
	act, finds, _, tree := activeIn(t, files, settingsReports...)
	if len(finds) != 0 {
		t.Fatal(finds)
	}
	ps, _ := LoadProfiles()
	now := time.Now()
	return Input{
		Report: report, Tree: tree, Active: act, Natives: Natives(ps, act), Newest: now,
		Read: func(rel string) ([]byte, error) {
			if c, ok := files[rel]; ok {
				return []byte(c), nil
			}
			return nil, errors.New("gone")
		},
		ModTime: func(string) (time.Time, error) { return now, nil },
	}
}

func TestGateHoldsEveryLineToTheRule(t *testing.T) {
	src := "def f():\n" + // 1
		"    return 1\n" + // 2 hit
		"    x = 1  # coverage:exempt windows only\n" + // 3 not hit, exempt
		"    y = 2  # coverage:exempt\n" + // 4 not hit, no reason
		"    z = 3  # coverage:exempt dead\n" + // 5 hit: refuted
		"    w = 4\n" + // 6 not hit
		"    v = 5  # pragma: no cover -- debug\n" // 7 not hit, native
	in := gateWorld(t, map[string]string{".coveragerc": "[run]\nsource = .\n", "a.py": src},
		Report{"a.py": {2: 1, 3: 0, 4: 0, 5: 7, 6: 0, 7: 0}})
	want := []Finding{
		{"a.py", 4, "coverage:exempt without a reason"},
		{"a.py", 5, "coverage:exempt but hit 7 times"},
		{"a.py", 6, "not covered"},
	}
	if got := Gate(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
}

func TestGateTreatsFileMarkersAndAbsence(t *testing.T) {
	files := map[string]string{
		"project.godot": "[nano_coverage]\n",
		"core/a.gd":     "func f():\n\tpass\n",
		"core/b.gd":     "# coverage:exempt-file only _init\n",
		"core/c.gd":     "# coverage:exempt-file never run\nfunc g():\n\tpass\n",
		"core/d.gd":     "func h():\n\tpass\n",
	}
	in := gateWorld(t, files, Report{"core/a.gd": {2: 1}, "core/c.gd": {3: 4}})
	want := []Finding{
		{"core/c.gd", 0, "coverage:exempt-file but 1 lines hit"},
		{"core/d.gd", 0, "is absent from the report"},
	}
	if got := Gate(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
}

func TestGateSkipsExcludedRecordsAndNativeFileMarkers(t *testing.T) {
	files := map[string]string{
		".nycrc":     `{"include":["src/**"],"exclude":["src/vendor/**"]}`,
		"src/a.js":   "x();\n",
		"src/gen.js": "/* istanbul ignore file: generated */\nx();\n",
		"src/vendor/v.js": "x();\n",
	}
	in := gateWorld(t, files, Report{"src/a.js": {1: 1}, "src/gen.js": {2: 1}, "src/vendor/v.js": {1: 0}})
	if got := Gate(in); len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestGateReportsASourceNewerThanTheReportAndOneItCannotRead(t *testing.T) {
	in := gateWorld(t, map[string]string{"a.py": "x = 1\n"}, Report{"a.py": {1: 1}, "b.py": {1: 1}})
	in.ModTime = func(rel string) (time.Time, error) {
		if rel == "a.py" {
			return in.Newest.Add(3 * time.Second), nil
		}
		return time.Time{}, errors.New("gone")
	}
	want := []Finding{{"a.py", 0, "is newer than the report"}, {"b.py", 0, "unreadable: gone"}}
	if got := Gate(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
	in.ModTime = func(string) (time.Time, error) { return in.Newest.Add(time.Second), nil }
	if got := Gate(in); len(got) != 1 {
		t.Fatalf("slack not granted: %v", got)
	}
}
```

  `b.py` steht im Bericht, aber nicht in `files`: `Read` scheitert, der
  Befund nennt das. (Dass `b.py` nicht auf der Platte liegt, meldet sonst
  schon `Check`; `Gate` sieht nur aufgelöste Pfade.)

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run Gate -count=1`
  Expected: FAIL.

- [ ] **Step 3: Implementieren.**

```go
package linecov

import (
	"fmt"
	"maps"
	"slices"
	"time"
)

// Input is everything the rules need; Check builds it.
type Input struct {
	Report  Report // resolved paths only
	Tree    *Tree
	Active  []Active
	Natives []Marker
	Newest  time.Time // modification time of the newest report read
	Read    func(rel string) ([]byte, error)
	ModTime func(rel string) (time.Time, error)
}

// mtimeSlack absorbs FAT's two-second mtime: a report written moments after
// a source can carry a time slightly before it.
const mtimeSlack = 2 * time.Second

// Gate applies the rules to every reported file and every expected one.
func Gate(in Input) []Finding {
	var out []Finding
	graded := map[string]bool{}
	for _, rel := range slices.Sorted(maps.Keys(in.Report)) {
		if excluded(in.Active, rel) {
			continue
		}
		graded[rel] = true
		src, err := in.Read(rel)
		if err != nil {
			out = append(out, Finding{rel, 0, "unreadable: " + err.Error()})
			continue
		}
		s := ScanSource(rel, src, in.Natives)
		out = append(out, s.Findings...)
		out = append(out, grade(rel, in.Report[rel], s)...)
		if mt, err := in.ModTime(rel); err == nil && mt.After(in.Newest.Add(mtimeSlack)) {
			out = append(out, Finding{rel, 0, "is newer than the report"})
		}
	}
	for _, rel := range in.Tree.Files {
		if graded[rel] || !slices.ContainsFunc(in.Active, func(a Active) bool { return a.Expects(rel) }) {
			continue
		}
		src, err := in.Read(rel)
		if err != nil {
			out = append(out, Finding{rel, 0, "unreadable: " + err.Error()})
			continue
		}
		s := ScanSource(rel, src, in.Natives)
		out = append(out, s.Findings...)
		if !s.File && !s.NativeFile {
			out = append(out, Finding{rel, 0, "is absent from the report"})
		}
	}
	return SortFindings(out)
}

// excluded: a record for a file the tool's settings exclude describes
// nothing that belongs to the scope.
func excluded(act []Active, rel string) bool {
	return slices.ContainsFunc(act, func(a Active) bool { return a.Owns(rel) && a.Excludes(rel) })
}

func grade(rel string, lines map[int]uint64, s Scan) []Finding {
	if s.NativeFile {
		return nil
	}
	if s.File {
		hit := 0
		for _, h := range lines {
			if h > 0 {
				hit++
			}
		}
		if hit > 0 {
			return []Finding{{rel, 0, fmt.Sprintf("coverage:exempt-file but %d lines hit", hit)}}
		}
		return nil
	}
	var out []Finding
	for _, n := range slices.Sorted(maps.Keys(lines)) {
		e, exempt := s.Lines[n]
		switch hits := lines[n]; {
		case hits > 0 && exempt && e.own:
			out = append(out, Finding{rel, n, fmt.Sprintf("coverage:exempt but hit %d times", hits)})
		case hits == 0 && !exempt:
			out = append(out, Finding{rel, n, "not covered"})
		case hits == 0 && e.own && !e.reason:
			out = append(out, Finding{rel, n, "coverage:exempt without a reason"})
		}
	}
	return out
}
```

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`. Den Zweig von `grade` für
  eine Datei mit `coverage:exempt-file`, die im Bericht steht, aber keinen
  Treffer hat, deckt:

```go
func TestGateAcceptsAnExemptFileWhoseLinesDidNotRun(t *testing.T) {
	in := gateWorld(t, map[string]string{"a.py": "# coverage:exempt-file only on Windows\nx = 1\n"}, Report{"a.py": {2: 0}})
	if got := Gate(in); len(got) != 0 {
		t.Fatalf("%v", got)
	}
}
```

- [ ] **Step 5: Mutationsrunde**, `$S/t13-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t13.txt`:

```
feat(linecov): apply the line rules, file markers, absence and freshness
```

```bash
git add internal/verify/linecov/gate.go internal/verify/linecov/gate_test.go
git commit -F "$S/msg-t13.txt"
```

---

### Task 14: `Check` — Berichte finden, lesen, auflösen

**Files:**
- Create: `internal/verify/linecov/check.go`
- Test: `internal/verify/linecov/check_test.go`

**Interfaces:**
- Consumes: alles aus Tasks 1–13; `gocover.ModulePath(gomod []byte) (string, error)` aus `internal/verify/gocover`
- Produces:
  - `type Options struct{ Root string; Reports []string }`
  - `type Result struct{ Findings []Finding; Notes []string }`
  - `func Check(o Options) (Result, error)`

- [ ] **Step 1: Stub und rote Tests.** `Check` liefert `Result{}, nil`.

```go
package linecov

import (
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func check(t *testing.T, root string, reports ...string) Result {
	t.Helper()
	res, err := Check(Options{Root: root, Reports: reports})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestCheckRunsWithoutFlagsFromTheDefaultLocations(t *testing.T) {
	root := writeTree(t, map[string]string{
		".coveragerc":  "[run]\nsource = src\n",
		"src/app.py":   "def f():\n    return 1\n    x = 2\n",
		"coverage.xml": `<coverage><sources><source>src</source></sources><packages><package><classes><class filename="app.py"><lines><line number="2" hits="1"/><line number="3" hits="0"/></lines></class></classes></package></packages></coverage>`,
	})
	res := check(t, root)
	if want := []Finding{{"src/app.py", 3, "not covered"}}; !reflect.DeepEqual(res.Findings, want) {
		t.Fatalf("%v %q", res.Findings, res.Notes)
	}
}

func TestCheckSaysWhereItLookedWhenNoReportExists(t *testing.T) {
	// Profiles are visited by name, so c8's locations come first; each
	// location is named once although three profiles share it.
	res := check(t, writeTree(t, map[string]string{"a.py": ""}))
	if len(res.Findings) != 1 ||
		!strings.HasPrefix(res.Findings[0].Text, "no coverage report found; looked in coverage/lcov.info, coverage/cobertura-coverage.xml, coverage.cobertura.xml") ||
		strings.Count(res.Findings[0].Text, "coverage/lcov.info") != 1 {
		t.Fatalf("%v", res.Findings)
	}
	root := writeTree(t, nil)
	res = check(t, root, filepath.Join(root, "nothing", "*.info"))
	if len(res.Findings) != 1 || !strings.Contains(res.Findings[0].Text, "nothing") {
		t.Fatalf("%v", res.Findings)
	}
}

// Review focus 4: an empty or binary report is a finding, never a panic.
func TestCheckReportsUnreadableUnknownAndEmptyReports(t *testing.T) {
	root := writeTree(t, map[string]string{"r/empty.info": "", "r/bin.dat": "\x00\x01\x02", "r/only.info": "SF:a.py\nend_of_record\n", "a.py": "x\n"})
	res := check(t, root, filepath.Join(root, "r"))
	want := []Finding{
		{"", 0, "the reports hold no line records"},
		{"r/bin.dat", 0, "unknown coverage format"},
		{"r/empty.info", 0, "unknown coverage format"},
	}
	if !reflect.DeepEqual(res.Findings, want) {
		t.Fatalf("%v", res.Findings)
	}
}

// Review focus 2: one file listed twice is one file with summed hits.
func TestCheckSumsAFileListedTwiceAndAcrossReports(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a.c": "x();\ny();\n",
		"one.info": "SF:a.c\nDA:1,0\nDA:2,1\nend_of_record\nSF:a.c\nDA:1,0\nend_of_record\n",
		"two.info": "SF:a.c\nDA:1,3\nend_of_record\n",
	})
	if res := check(t, root, filepath.Join(root, "*.info")); len(res.Findings) != 0 {
		t.Fatalf("%v", res.Findings)
	}
	if res := check(t, root, filepath.Join(root, "one.info")); !reflect.DeepEqual(res.Findings, []Finding{{"a.c", 1, "not covered"}}) {
		t.Fatalf("%v", res.Findings)
	}
}

func TestCheckResolvesEveryOutcomeAndNotesWhatItSkipped(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod":  "module example.com/m\n",
		"a.go":    "package m\n",
		"w/.git":  "gitdir: x",
		"w/b.c":   "",
		"x/com/y/D.java": "", "z/com/y/D.java": "",
		"r.info":  "SF:/usr/include/stdio.h\nDA:1,0\nSF:w/b.c\nDA:1,0\nSF:gone.c\nDA:1,0\nend_of_record\n",
		"g.out":   "mode: set\nexample.com/m/a.go:1.1,1.9 1 1\n",
		"j.xml":   `<report><package name="com/y"><sourcefile name="D.java"><line nr="1" ci="1"/></sourcefile></package></report>`,
	})
	res := check(t, root, filepath.Join(root, "r.info"), filepath.Join(root, "g.out"), filepath.Join(root, "j.xml"))
	want := []Finding{
		{"com/y/D.java", 0, "ambiguous source: x/com/y/D.java, z/com/y/D.java"},
		{"gone.c", 0, "is in the report but not on disk"},
	}
	if !reflect.DeepEqual(res.Findings, want) {
		t.Fatalf("%v", res.Findings)
	}
	for _, note := range []string{"1 records outside the root were skipped", "1 records in other checkouts were skipped"} {
		if !slices.Contains(res.Notes, note) {
			t.Errorf("missing note %q in %q", note, res.Notes)
		}
	}
}

func TestCheckNotesAProfileWithoutScopeAndReportsReadProblems(t *testing.T) {
	root := writeTree(t, map[string]string{
		"coverage/lcov.info": "SF:src/a.js\nDA:1,1\nDA:x,1\nend_of_record\n",
		"src/a.js":           "x();\n",
	})
	res := check(t, root)
	if !reflect.DeepEqual(res.Findings, []Finding{{"coverage/lcov.info", 0, `src/a.js: unreadable record "DA:x,1"`}}) {
		t.Fatalf("%v", res.Findings)
	}
	if !slices.Contains(res.Notes, "vitest: no scope in its settings; files absent from the report are not checked") {
		t.Fatalf("%q", res.Notes)
	}
}

func TestCheckFailsOnAMissingRootAndABadGlob(t *testing.T) {
	if _, err := Check(Options{Root: filepath.Join(t.TempDir(), "absent")}); err == nil {
		t.Fatal("no error for a missing root")
	}
	if _, err := Check(Options{Root: t.TempDir(), Reports: []string{"["}}); err == nil {
		t.Fatal("no error for a bad glob")
	}
}

func TestCheckReadsEveryFileOfAReportDirectoryButNotItsSubdirectories(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a.c": "x();\n", "r/one.info": "SF:a.c\nDA:1,1\nend_of_record\n", "r/deeper/two.info": "SF:a.c\nDA:1,0\nend_of_record\n",
	})
	if res := check(t, root, filepath.Join(root, "r")); len(res.Findings) != 0 {
		t.Fatalf("%v", res.Findings)
	}
}

func TestCheckReportsAReportItCannotRead(t *testing.T) {
	root := writeTree(t, map[string]string{"a.c": "x();\n", "r.info": "SF:a.c\nDA:1,1\nend_of_record\n"})
	old := readFile
	t.Cleanup(func() { readFile = old })
	readFile = func(string) ([]byte, error) { return nil, errors.New("locked") }
	res := check(t, root, filepath.Join(root, "r.info"))
	want := []Finding{{"", 0, "the reports hold no line records"}, {"r.info", 0, "unreadable: locked"}}
	if !reflect.DeepEqual(res.Findings, want) {
		t.Fatalf("%v", res.Findings)
	}
}

func TestCheckNamesAReportOutsideTheRootAsGiven(t *testing.T) {
	outside := writeTree(t, map[string]string{"r.info": ""})
	res := check(t, writeTree(t, map[string]string{"a.c": ""}), filepath.Join(outside, "r.info"))
	if len(res.Findings) == 0 || res.Findings[len(res.Findings)-1].Path != filepath.ToSlash(filepath.Join(outside, "r.info")) {
		t.Fatalf("%v", res.Findings)
	}
}
```

  Imports des Tests: `errors`, `path/filepath`, `reflect`, `slices`,
  `strings`, `testing`.

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run Check -count=1`
  Expected: FAIL.

- [ ] **Step 3: Implementieren.**

```go
package linecov

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/verify/gocover"
)

// Options of one run. Reports are files, directories or globs relative to
// the working directory; none means the profiles' default locations.
type Options struct {
	Root    string
	Reports []string
}

// Result is what the run found and what it could not check.
type Result struct {
	Findings []Finding
	Notes    []string
}

// readFile is a seam for the one read error a test cannot make on disk.
var readFile = os.ReadFile

// Check reads the reports, resolves their paths, activates the profiles and
// applies the rules. An error means the run could not start at all.
func Check(o Options) (Result, error) {
	ps, err := LoadProfiles()
	if err != nil {
		return Result{}, err
	}
	tree, err := ScanTree(o.Root)
	if err != nil {
		return Result{}, err
	}
	files, looked, err := reportFiles(o.Reports, tree.Root, ps)
	if err != nil {
		return Result{}, err
	}
	if len(files) == 0 {
		return Result{Findings: []Finding{{"", 0, "no coverage report found; looked in " + strings.Join(looked, ", ")}}}, nil
	}
	var module string
	if gomod, err := os.ReadFile(filepath.Join(tree.Root, "go.mod")); err == nil {
		module, _ = gocover.ModulePath(gomod)
	}
	rep := Report{}
	var finds []Finding
	var notes, readRel []string
	var newest time.Time
	outside, nested := 0, 0
	for _, f := range files {
		shown := display(tree.Root, f)
		content, err := readFile(f)
		if err != nil {
			finds = append(finds, Finding{shown, 0, "unreadable: " + err.Error()})
			continue
		}
		if st, err := os.Stat(f); err == nil && st.ModTime().After(newest) {
			newest = st.ModTime()
		}
		format := Detect(content)
		if format == nil {
			finds = append(finds, Finding{shown, 0, "unknown coverage format"})
			continue
		}
		readRel = append(readRel, shown)
		recs, problems := format.Read(content, ReadContext{Module: module})
		for _, p := range problems {
			finds = append(finds, Finding{shown, 0, p})
		}
		for _, rec := range recs {
			switch res := tree.Resolve(rec); res.Outcome {
			case Resolved:
				rep.File(res.Rel)
				for n, h := range rec.Lines {
					rep.Add(res.Rel, n, h)
				}
			case Outside:
				outside++
			case Nested:
				nested++
			case Missing:
				finds = append(finds, Finding{res.Rel, 0, "is in the report but not on disk"})
			case Ambiguous:
				finds = append(finds, Finding{res.Rel, 0, "ambiguous source: " + strings.Join(res.Candidates, ", ")})
			}
		}
	}
	act, actFinds, actNotes := Activate(tree, readRel, ps)
	finds, notes = append(finds, actFinds...), append(notes, actNotes...)
	if outside > 0 {
		notes = append(notes, fmt.Sprintf("%d records outside the root were skipped", outside))
	}
	if nested > 0 {
		notes = append(notes, fmt.Sprintf("%d records in other checkouts were skipped", nested))
	}
	for _, a := range act {
		if !a.Scoped() {
			notes = append(notes, a.Profile.Name+": no scope in its settings; files absent from the report are not checked")
		}
	}
	if rep.Lines() == 0 {
		finds = append(finds, Finding{"", 0, "the reports hold no line records"})
		return Result{Findings: SortFindings(finds), Notes: notes}, nil
	}
	finds = append(finds, Gate(Input{
		Report: rep, Tree: tree, Active: act, Natives: Natives(ps, act), Newest: newest,
		Read: func(rel string) ([]byte, error) { return os.ReadFile(filepath.Join(tree.Root, filepath.FromSlash(rel))) },
		ModTime: func(rel string) (time.Time, error) {
			st, err := os.Stat(filepath.Join(tree.Root, filepath.FromSlash(rel)))
			if err != nil {
				return time.Time{}, err
			}
			return st.ModTime(), nil
		},
	})...)
	return Result{Findings: SortFindings(finds), Notes: notes}, nil
}

// reportFiles expands the given reports, or the default locations of every
// profile; looked lists what was tried, for the finding that nothing was found.
func reportFiles(given []string, root string, ps []Profile) (files, looked []string, err error) {
	if len(given) == 0 {
		for _, p := range ps {
			for _, r := range p.Reports {
				if slices.Contains(looked, r) {
					continue
				}
				looked = append(looked, r)
				if st, err := os.Stat(filepath.Join(root, filepath.FromSlash(r))); err == nil && st.Mode().IsRegular() {
					files = append(files, filepath.Join(root, filepath.FromSlash(r)))
				}
			}
		}
		return files, looked, nil
	}
	for _, g := range given {
		looked = append(looked, g)
		if st, err := os.Stat(g); err == nil && st.IsDir() {
			entries, _ := os.ReadDir(g)
			for _, e := range entries {
				if e.Type().IsRegular() {
					files = append(files, filepath.Join(g, e.Name()))
				}
			}
			continue
		}
		matches, err := filepath.Glob(g)
		if err != nil {
			return nil, nil, fmt.Errorf("--report %q: %w", g, err)
		}
		files = append(files, matches...)
	}
	return files, looked, nil
}

// display names a report relative to the root when it lies below it.
func display(root, f string) string {
	abs, _ := filepath.Abs(f)
	if rel, err := filepath.Rel(root, abs); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(f)
}
```

- [ ] **Step 4: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS; Abdeckung ohne `not covered:`.

- [ ] **Step 5: Mutationsrunde**, `$S/t14-mut.txt`.

- [ ] **Step 6: Commit.** `$S/msg-t14.txt`:

```
feat(linecov): find, read and resolve coverage reports in one check
```

```bash
git add internal/verify/linecov/check.go internal/verify/linecov/check_test.go
git commit -F "$S/msg-t14.txt"
```

---

### Task 15: Fixture-Pflicht für Profile und Formate

**Files:**
- Create: `internal/verify/linecov/fixtures_test.go`
- Create: `internal/verify/linecov/testdata/formats/{lcov,cobertura,jacoco,goprofile}/…`
- Create: `internal/verify/linecov/testdata/tools/<name>/…` für alle zwölf Profile

**Interfaces:**
- Consumes: `LoadProfiles`, `formats`, `Check`, `Activate`, `ScanTree`, `Natives`

- [ ] **Step 1: Der Test.** Er läuft zuerst rot, weil `testdata/` fehlt.

```go
package linecov

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Every format and every profile brings a fixture, so an entry cannot be
// added untested. A tool fixture is a small project: its settings file, a
// source carrying each of the profile's markers with a reason, and a report
// at a default location or as report.* beside it. want.txt lists the
// findings the fixture must produce, one per line; absent means none.
func TestEveryFormatHasAFixtureItReadsCleanly(t *testing.T) {
	for _, f := range formats() {
		matches, _ := filepath.Glob(filepath.Join("testdata", "formats", f.Name(), "report.*"))
		if len(matches) != 1 {
			t.Errorf("%s: want one testdata/formats/%s/report.*", f.Name(), f.Name())
			continue
		}
		content, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatal(err)
		}
		if got := Detect(content); got == nil || got.Name() != f.Name() {
			t.Errorf("%s: detected as %v", f.Name(), got)
		}
		recs, problems := f.Read(content, ReadContext{Module: "example.com/m"})
		if len(recs) == 0 || len(problems) != 0 {
			t.Errorf("%s: %d records, problems %q", f.Name(), len(recs), problems)
		}
	}
}

func TestEveryProfileHasAFixtureProjectThatPasses(t *testing.T) {
	ps, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		src := filepath.Join("testdata", "tools", p.Name)
		if _, err := os.Stat(src); err != nil {
			t.Errorf("%s: no fixture project at %s", p.Name, src)
			continue
		}
		// A copy, written in one go: a checkout writes the fixture's files in
		// any order and at any pace, and the freshness rule would see that.
		dir := t.TempDir()
		if err := os.CopyFS(dir, os.DirFS(src)); err != nil {
			t.Fatal(err)
		}
		reports, _ := filepath.Glob(filepath.Join(dir, "report.*"))
		res, err := Check(Options{Root: dir, Reports: reports})
		if err != nil {
			t.Fatalf("%s: %v", p.Name, err)
		}
		var got, want []string
		for _, f := range res.Findings {
			got = append(got, f.String())
		}
		if w, _ := os.ReadFile(filepath.Join(dir, "want.txt")); strings.TrimSpace(string(w)) != "" {
			want = strings.Split(strings.TrimSpace(string(w)), "\n")
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: findings %q, want %q", p.Name, got, want)
		}
		tree, err := ScanTree(dir)
		if err != nil {
			t.Fatal(err)
		}
		act, _, _ := Activate(tree, nil, []Profile{p})
		if len(p.Settings) > 0 && (len(act) != 1 || act[0].Settings == "") {
			t.Errorf("%s: the fixture's settings file was not read", p.Name)
		}
		for _, spec := range p.Markers {
			m, _ := spec.compile("")
			if !fixtureHas(t, tree, p, m) {
				t.Errorf("%s: no fixture source carries %s", p.Name, spec.Label)
			}
		}
	}
}

func fixtureHas(t *testing.T, tree *Tree, p Profile, m Marker) bool {
	for _, rel := range tree.Files {
		if !slices.Contains(p.Extensions, filepath.Ext(rel)) {
			continue
		}
		src, err := os.ReadFile(filepath.Join(tree.Root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if m.Re.Match(src) {
			return true
		}
	}
	return false
}
```

  `want.txt` fehlt bei allen zwölf Fixtures: jede muss sauber durchlaufen.
  Die Datei ist für spätere Fixtures da, die einen Befund zeigen sollen.

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/verify/linecov -run Every -count=1`
  Expected: FAIL mit „no fixture project“ und „want one testdata/formats“.

- [ ] **Step 3: Format-Fixtures.**
  - `testdata/formats/lcov/report.info`:
    ```
    TN:
    SF:src/a.c
    DA:1,1
    DA:2,0
    end_of_record
    ```
  - `testdata/formats/cobertura/report.xml`:
    ```xml
    <?xml version="1.0" ?>
    <coverage version="7.6.1"><sources><source>src</source></sources><packages><package name="."><classes><class name="a.py" filename="a.py"><lines><line number="1" hits="1"/></lines></class></classes></package></packages></coverage>
    ```
  - `testdata/formats/jacoco/report.xml`:
    ```xml
    <?xml version="1.0" encoding="UTF-8"?><!DOCTYPE report PUBLIC "-//JACOCO//DTD Report 1.1//EN" "report.dtd"><report name="m"><package name="com/x"><sourcefile name="Calc.java"><line nr="3" mi="0" ci="1" mb="0" cb="0"/></sourcefile></package></report>
    ```
  - `testdata/formats/goprofile/report.out`:
    ```
    mode: set
    example.com/m/a.go:3.10,5.2 2 1
    ```

- [ ] **Step 4: Werkzeug-Fixtures.** Je Profil ein Verzeichnis
  `testdata/tools/<name>/` mit genau diesen Dateien:

  **coveragepy**
  - `.coveragerc`: `[run]\nsource = src\nomit = src/skip_*.py\n`
  - `src/app.py`:
    ```python
    def one():
        return 1
    def debug():  # pragma: no cover -- only for manual runs
        return 2
    ```
  - `coverage.xml`:
    ```xml
    <?xml version="1.0" ?>
    <coverage version="7.6.1"><sources><source>src</source></sources><packages><package name="."><classes><class name="app.py" filename="app.py"><lines><line number="1" hits="1"/><line number="2" hits="1"/></lines></class></classes></package></packages></coverage>
    ```

  **coverlet**
  - `coverlet.runsettings`:
    ```xml
    <?xml version="1.0" encoding="utf-8"?>
    <RunSettings><DataCollectionRunSettings><DataCollectors><DataCollector friendlyName="XPlat code coverage"><Configuration><Format>cobertura</Format><ExcludeByFile>**/Generated/*.cs</ExcludeByFile></Configuration></DataCollector></DataCollectors></DataCollectionRunSettings></RunSettings>
    ```
  - `Calc.cs`:
    ```csharp
    public static class Calc
    {
        public static int One() => 1;
        [ExcludeFromCodeCoverage(Justification = "debug helper")]
        public static int Two() => 2;
    }
    ```
  - `coverage.cobertura.xml`:
    ```xml
    <?xml version="1.0" encoding="utf-8"?>
    <coverage line-rate="1" version="1.9"><sources><source>.</source></sources><packages><package name="app"><classes><class name="Calc" filename="Calc.cs"><lines><line number="3" hits="1"/></lines></class></classes></package></packages></coverage>
    ```

  **gcovr**
  - `gcovr.cfg`: `filter = src/\nexclude = src/third_party/\nexclude-pattern-prefix = MYPROJ\n`
  - `src/main.cpp`:
    ```cpp
    int one() { return 1; }
    int two() { return 2; } // GCOVR_EXCL_LINE generated by the build
    int three() { return 3; } // MYPROJ_EXCL_LINE kept for the ABI
    // GCOVR_EXCL_START debug-only helpers
    int four() { return 4; }
    // GCOVR_EXCL_STOP
    ```
  - `report.info`: `SF:src/main.cpp\nDA:1,1\nend_of_record\n`

  **lcov**
  - `.lcovrc`: `lcov_excl_line = LCOV_EXCL_LINE\nexclude = */third_party/*\n`
  - `src/lib.c`:
    ```c
    int one(void) { return 1; }
    int two(void) { return 2; } /* LCOV_EXCL_LINE never called on this platform */
    /* LCOV_EXCL_START legacy API */
    int three(void) { return 3; }
    /* LCOV_EXCL_STOP */
    ```
  - `report.info`: `SF:src/lib.c\nDA:1,1\nend_of_record\n`

  **rust**
  - `src/lib.rs`:
    ```rust
    pub fn one() -> i32 { 1 }
    // only reachable from the CLI binary
    #[cfg_attr(coverage_nightly, coverage(off))]
    pub fn two() -> i32 { 2 }
    ```
  - `report.info`: `SF:src/lib.rs\nDA:1,1\nend_of_record\n`

  **phpunit**
  - `phpunit.xml`:
    ```xml
    <?xml version="1.0"?>
    <phpunit><source><include><directory suffix=".php">src</directory></include><exclude><directory>src/Legacy</directory></exclude></source></phpunit>
    ```
  - `src/Calc.php`:
    ```php
    <?php
    function one() { return 1; }
    // thin wrapper around the extension, tested upstream
    function two() { return 2; } // @codeCoverageIgnore
    // platform shim, tested upstream
    // @codeCoverageIgnoreStart
    function three() { return 3; }
    // @codeCoverageIgnoreEnd
    ```
  - `report.xml`:
    ```xml
    <?xml version="1.0"?>
    <coverage><sources><source>src</source></sources><packages><package name="app"><classes><class name="Calc" filename="Calc.php"><lines><line number="2" hits="1"/></lines></class></classes></package></packages></coverage>
    ```

  **nyc**
  - `.nycrc`: `{ "include": ["src/**/*.js"], "exclude": ["src/vendor/**"] }`
  - `src/app.js`:
    ```js
    export const one = () => 1;
    /* istanbul ignore next: guarded by a feature flag */
    export const two = () => 2;
    ```
  - `src/gen.js`:
    ```js
    /* istanbul ignore file: generated by the build */
    export const x = 1;
    ```
  - `coverage/lcov.info`: `SF:src/app.js\nDA:1,1\nend_of_record\n`

  **c8**
  - `.c8rc.json`: `{ "include": ["src/**"] }`
  - `src/app.js`:
    ```js
    export const one = () => 1;
    /* c8 ignore next -- only in the browser */
    export const two = () => 2;
    /* c8 ignore start -- debug output */
    console.log("debug");
    /* c8 ignore stop */
    ```
  - `src/gen.js`:
    ```js
    /* c8 ignore file -- generated by the build */
    export const x = 1;
    ```
  - `coverage/lcov.info`: `SF:src/app.js\nDA:1,1\nend_of_record\n`

  **vitest**
  - `src/app.ts`:
    ```ts
    export const one = () => 1;
    /* v8 ignore next -- browser only */
    export const two = () => 2;
    /* v8 ignore start -- debug output */
    console.log("debug");
    /* v8 ignore stop */
    ```
  - `src/gen.ts`: `/* v8 ignore file -- generated by the build */\nexport const x = 1;\n`
  - `coverage/lcov.info`: `SF:src/app.ts\nDA:1,1\nDA:3,1\nend_of_record\n`
    (v8 schreibt die ignorierte Zeile 3 als getroffen; der native Marker ist
    darauf kein Befund.)

  **jacoco**
  - `src/main/java/com/x/Calc.java`:
    ```java
    package com.x;
    class Calc {
        int one() { return 1; }
    }
    ```
  - `target/site/jacoco/jacoco.xml`: wie `testdata/formats/jacoco/report.xml`.

  **kover**
  - `src/main/kotlin/com/x/Calc.kt`:
    ```kotlin
    package com.x
    class Calc {
        fun one() = 1
    }
    ```
  - `build/reports/kover/report.xml`:
    ```xml
    <?xml version="1.0" encoding="UTF-8"?><report name="m"><package name="com/x"><sourcefile name="Calc.kt"><line nr="3" mi="0" ci="1" mb="0" cb="0"/></sourcefile></package></report>
    ```

  **nanocoverage**
  - `project.godot`:
    ```
    config_version=5

    [nano_coverage]

    general/ignore_paths=PackedStringArray("**/*_test.gd", "res://harness/**")
    general/ignore_addons=true
    ```
  - `core/a.gd`:
    ```gdscript
    extends RefCounted
    func one() -> int:
    	return 1
    ```
  - `harness/probe.gd`: `extends Node\nfunc run():\n\tpass\n`
  - `addons/x/plugin.gd`: `extends Node\nfunc run():\n\tpass\n`
  - `test/a_test.gd`: `extends Node\nfunc test_one():\n\tpass\n`
  - `coverage-report/lcov.info`: `SF:core/a.gd\nDA:3,1\nend_of_record\n`

  Die Fixtures von nyc, c8 und vitest teilen den Standardort
  `coverage/lcov.info`; in jeder Fixture werden darum auch die beiden anderen
  JavaScript-Profile aktiv. Das ist gewollt und erzeugt nur Hinweise.

- [ ] **Step 5: Grün prüfen.** Run: `go test ./internal/verify/linecov -count=1`
  Expected: PASS. Scheitert eine Fixture mit einem Befund, ist zuerst zu
  klären, ob die Fixture oder der Code falsch ist; der Befund ist dann im
  Commit-Rumpf zu nennen.

- [ ] **Step 6: Commit.** `$S/msg-t15.txt`:

```
test(linecov): demand a fixture project for every tool and format
```

```bash
git add internal/verify/linecov/fixtures_test.go internal/verify/linecov/testdata
git commit -F "$S/msg-t15.txt"
```

---

### Task 16: CLI, `Reserved`, Doku

**Files:**
- Modify: `internal/cli/check.go:63` (Usage), `:66-92` (Switch), neu `checkLinecov` nach `checkGocover` (`:334`)
- Modify: `internal/cli/check_test.go:26` (`TestCheckNeedsARequest`), neue Tests am Ende
- Modify: `internal/verify/schema.go:34-38` (`Reserved`), `internal/verify/schema_test.go` (`TestNames`, Liste bei :189)
- Modify: `docs/en/cli-reference.md` (neuer Abschnitt nach :172), `docs/de/cli-reference.md` (nach dem gocover-Abschnitt, ~:182)
- Modify: `README.md:148-149`, `README.de.md:148-149`

**Interfaces:**
- Consumes: `linecov.Check`, `linecov.Options`, `linecov.Result`
- Produces: `func checkLinecov(args []string, stdout, stderr io.Writer) int`, `type stringList []string`

- [ ] **Step 1: Rote Tests.** `checkLinecov` als Stub, der `0` liefert, und
  ein Fall `"linecov"` im Switch. In `check_test.go`:

```go
func TestCheckLinecovPassesACleanProject(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.c"), "x();\n")
	writeFile(t, filepath.Join(root, "r.info"), "SF:a.c\nDA:1,1\nend_of_record\n")
	code, out, errOut := run("check", "linecov", "--root", root, "--report", filepath.Join(root, "r.info"))
	if code != 0 || out != "" {
		t.Fatalf("code %d out %q err %q", code, out, errOut)
	}
}

func TestCheckLinecovPrintsFindingsAndNotes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "a.js"), "x();\ny();\n")
	writeFile(t, filepath.Join(root, "coverage", "lcov.info"), "SF:src/a.js\nDA:1,1\nDA:2,0\nend_of_record\n")
	code, out, errOut := run("check", "linecov", "--root", root)
	if code != 1 || out != "src/a.js:2 not covered\n" || !strings.Contains(errOut, "loomux check linecov: note: vitest: no scope") {
		t.Fatalf("code %d out %q err %q", code, out, errOut)
	}
}

func TestCheckLinecovRefusesABadCall(t *testing.T) {
	if code, _, _ := run("check", "linecov", "--nope"); code != 2 {
		t.Fatalf("code %d", code)
	}
	if code, _, errOut := run("check", "linecov", "extra"); code != 2 || !strings.Contains(errOut, `unexpected argument "extra"`) {
		t.Fatalf("code %d err %q", code, errOut)
	}
	if code, _, errOut := run("check", "linecov", "--root", filepath.Join(t.TempDir(), "absent")); code != 1 || !strings.HasPrefix(errOut, "loomux check linecov: ") {
		t.Fatalf("code %d err %q", code, errOut)
	}
}
```

  Vorher mit `grep -n "func writeFile" internal/cli/*_test.go` nachsehen;
  fehlt der Helfer, kommt er nach `check_test.go`:

```go
// writeFile writes content to p, making its directory first.
func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
```

  Gibt es ihn mit anderer Signatur, passen die drei Tests sich an ihn an.
  In `TestCheckNeedsARequest` den erwarteten Text um
  `, linecov` erweitern (nach `gocover`). In `schema_test.go` `"linecov"` in
  die Liste bei :189 aufnehmen.

- [ ] **Step 2: Rot prüfen.** Run: `go test ./internal/cli ./internal/verify -count=1`
  Expected: FAIL in den drei neuen Tests, in `TestCheckNeedsARequest` und in
  `TestNames`.

- [ ] **Step 3: Implementieren.** `schema.go`: `"linecov"` in die Liste von
  `Reserved`. `check.go`: Usage-Text
  `"loomux check: name a profile or kinds (lint,types,test,coverage,graph), or one of: commit-msg, gofmt, gocover, linecov, graph-fresh, blast-audit"`,
  im Switch `case "linecov": return checkLinecov(args[1:], stdout, stderr)`,
  und:

```go
// stringList collects a flag that may be given more than once.
type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

// checkLinecov holds every executable line to one rule, in any language
// whose tool writes LCOV, Cobertura, JaCoCo or a Go profile.
func checkLinecov(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check linecov", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var reports stringList
	fs.Var(&reports, "report", "coverage report: file, directory or glob (repeatable; default: the tools' own locations)")
	root := fs.String("root", ".", "directory paths, settings and scope refer to")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "loomux check linecov: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	res, err := linecov.Check(linecov.Options{Root: *root, Reports: reports})
	if err != nil {
		fmt.Fprintf(stderr, "loomux check linecov: %v\n", err)
		return 1
	}
	for _, n := range res.Notes {
		fmt.Fprintf(stderr, "loomux check linecov: note: %s\n", n)
	}
	for _, f := range res.Findings {
		fmt.Fprintln(stdout, f)
	}
	if len(res.Findings) > 0 {
		return 1
	}
	return 0
}
```

  Import `github.com/xidus90/loomux/internal/verify/linecov`.

- [ ] **Step 4: Doku.** `docs/en/cli-reference.md` nach dem gocover-Abschnitt:

```markdown
### `loomux check linecov [--report <path|glob>]… [--root <dir>]`

Holds every executable line to one rule: it ran, or it carries an exemption
with a reason. Works for any tool that writes LCOV, Cobertura, JaCoCo XML
(also Kover) or a Go coverage profile, and reads scope and markers from the
tool's own settings file, so a project needs no configuration.

- **Flags:** `--report` (repeatable) names a file, a directory (every file in
  it) or a glob; without it, the tools' default report locations below
  `--root` are read. `--root` (default `.`) is what paths, settings and scope
  refer to.
- **Tools it knows:** coverage.py, coverlet, gcovr, lcov, cargo-llvm-cov,
  PHPUnit, nyc, c8, vitest, JaCoCo, Kover, Nano Coverage. Each is found by
  its settings file in `--root` (`.coveragerc`, `pyproject.toml`,
  `*.runsettings`, `gcovr.cfg`, `.lcovrc`, `phpunit.xml`, `.nycrc`,
  `.c8rc.json`, `package.json`, `project.godot`, …) or by a report at its
  default location.
- **Rules:** a line with no hits needs `coverage:exempt <reason>` on it, or
  on a comment-only line directly above it; a marked line that ran is a
  finding; `coverage:exempt-file <reason>` exempts a whole file. A tool's own
  marker (`pragma: no cover`, `LCOV_EXCL_LINE`, `@codeCoverageIgnore`,
  `istanbul ignore`, `[ExcludeFromCodeCoverage]`, …) needs a reason after it
  or in a comment line above it. A file in the tool's scope that is missing
  from every report, a source newer than the report, and a report path that
  is not on disk are findings. No report, or no line record, is red.
- **Skipped, with a note:** records outside `--root` (system headers,
  dependencies), records inside another checkout below it (a linked
  worktree), and the absence rule for a tool whose scope lives in code
  (`vitest.config.ts`, Gradle).
- **Exit codes:** 0 clean, 1 findings (one per line on stdout), 2 bad call.
```

  `docs/de/cli-reference.md` an der entsprechenden Stelle:

```markdown
### `loomux check linecov [--report <pfad|glob>]… [--root <verz>]`

Hält jede ausführbare Zeile an einer Regel: gelaufen, oder mit Begründung
ausgenommen. Gilt für jedes Werkzeug, das LCOV, Cobertura, JaCoCo-XML (auch
Kover) oder ein Go-Profil schreibt, und liest Bereich und Marker aus dessen
eigener Einstellungsdatei; ein Projekt braucht keine Konfiguration.

- **Flags:** `--report` (wiederholbar) nennt eine Datei, ein Verzeichnis
  (jede Datei darin) oder ein Glob; ohne werden die Standardorte der
  Werkzeuge unter `--root` gelesen. `--root` (Vorgabe `.`) ist der Bezug für
  Pfade, Einstellungen und Bereich.
- **Bekannte Werkzeuge:** coverage.py, coverlet, gcovr, lcov, cargo-llvm-cov,
  PHPUnit, nyc, c8, vitest, JaCoCo, Kover, Nano Coverage. Erkannt an der
  Einstellungsdatei in `--root` (`.coveragerc`, `pyproject.toml`,
  `*.runsettings`, `gcovr.cfg`, `.lcovrc`, `phpunit.xml`, `.nycrc`,
  `.c8rc.json`, `package.json`, `project.godot`, …) oder an einem Bericht am
  Standardort.
- **Regeln:** eine Zeile ohne Treffer braucht `coverage:exempt <grund>` auf
  ihr oder auf einer reinen Kommentarzeile direkt darüber; eine markierte
  Zeile, die lief, ist ein Befund; `coverage:exempt-file <grund>` nimmt eine
  ganze Datei aus. Ein Marker des Werkzeugs (`pragma: no cover`,
  `LCOV_EXCL_LINE`, `@codeCoverageIgnore`, `istanbul ignore`,
  `[ExcludeFromCodeCoverage]`, …) braucht einen Grund dahinter oder in einer
  Kommentarzeile darüber. Eine Datei im Bereich des Werkzeugs, die in keinem
  Bericht steht, eine Quelle, die neuer ist als der Bericht, und ein
  Berichtspfad, der nicht auf der Platte liegt, sind Befunde. Kein Bericht
  oder kein Zeilendatensatz ist rot.
- **Übergangen, mit Hinweis:** Datensätze außerhalb von `--root`
  (Systemheader, Abhängigkeiten), Datensätze in einem anderen Checkout
  darunter (verlinkter Worktree) und die Regel „fehlt im Bericht“ für ein
  Werkzeug, dessen Bereich in Code steht (`vitest.config.ts`, Gradle).
- **Exit-Codes:** 0 sauber, 1 Befunde (einer je Zeile auf stdout), 2
  falscher Aufruf.
```

  `README.md`: Zeile 148 („LCOV coverage gate“) löschen; in Zeile 149
  `| LCOV coverage gate |` durch `| \`check linecov\` |` ersetzen. In der
  Liste der Prüfungen (dort, wo `check gocover` genannt ist) einen Punkt
  ergänzen: `` `loomux check linecov`: line coverage for any language, from the coverage tool's own report and settings ``.
  `README.de.md` entsprechend: Zeile „LCOV-Coverage-Tor“ löschen, in
  „Godot-Testsuite“ die Abhängigkeit `` `check linecov` `` nennen, und in der
  Liste: `` `loomux check linecov`: Zeilenabdeckung für jede Sprache, aus Bericht und Einstellungen des Coverage-Werkzeugs ``.

- [ ] **Step 5: Grün prüfen.** Run: `go test ./internal/cli ./internal/verify/... -count=1`
  Expected: PASS. Abdeckung für `internal/cli`:
  `go test ./internal/cli -count=1 -coverprofile=cli.out && go run ./cmd/loomux check gocover --profile cli.out; rm -f cli.out`
  Expected: keine Zeile `not covered:` für `checkLinecov` und `stringList`.

- [ ] **Step 6: Mutationsrunde** über `./internal/cli` (nur die neuen
  Funktionen sind neu), `$S/t16-mut.txt`.

- [ ] **Step 7: Commit.** `$S/msg-t16.txt`:

```
feat(check): add check linecov, a line coverage gate for every language

Reads LCOV, Cobertura, JaCoCo and Go profiles, takes scope and exclusion
markers from the coverage tools' own settings files, and needs no flags in
a project whose tool writes to its default location.
```

```bash
git add internal/cli/check.go internal/cli/check_test.go internal/verify/schema.go internal/verify/schema_test.go docs/en/cli-reference.md docs/de/cli-reference.md README.md README.de.md
git commit -F "$S/msg-t16.txt"
```

---

### Task 17: Abgleich mit echten Daten und volles Tor

**Files:** keine neuen; nur Messung. Ein Fehler, den sie findet, wird ein
eigener Commit mit Test in dem Task, dem der Code gehört.

- [ ] **Step 1: loomux selbst.**
  `go test ./internal/verify/linecov -count=1 -coverprofile="$S/self.out"`,
  dann `go run ./cmd/loomux check linecov --report "$S/self.out"`.
  Expected: Exit 0 oder nur Befunde, die `check gocover` für dieselbe Datei
  ebenfalls meldet. Jede Abweichung zwischen beiden Toren ist ein Befund an
  `goprofile.go` oder `gate.go`.

- [ ] **Step 2: space mit Nano Coverage.** Im space-Checkout einen
  Suite-Lauf mit Coverage auslösen (`uv run --script .claude/hooks/godot_quality.py --precommit`
  schreibt `godot/coverage-report/lcov.info`), dann
  `go run ./cmd/loomux check linecov --root <space>/godot`.
  Expected: Befunde für jede `# no-cover:`-Zeile (noch nicht auf
  `coverage:exempt` umgestellt — das ist Teil 6) und für `harness/` (noch
  nicht in `ignore_paths`), sonst dieselben Befunde wie
  `coverage_gate.findings`. Die Liste der Unterschiede geht in den
  Commit-Rumpf von Step 4 und ist die Eingabe für Teil 6.

- [ ] **Step 3: Volles Tor.** Run: `bash ci/gate.sh`
  Expected: alle Lanes `ok`, keine Zeile `not covered:`.

- [ ] **Step 4: Abschluss-Commit nur, wenn Step 1–3 etwas geändert haben;**
  sonst kein Commit. Die Messwerte aus Step 2 (Zahl der Befunde je Art)
  bekommt die Beschreibung des Pull Requests.
