# loomux G4b — Implementierungsplan: Git-Diff-Blast, Art `graph`, Blast-Monitor

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Den Blast-Radius eines git-Diffs berechnen und zeigen (`loomux graph blast`, MCP `graph_blast`),
ihn als Art `graph` im Pre-Commit prüfen (`check graph-fresh`, `check blast-audit`) und nach einem
Edit die Aufrufer einer geänderten Go-Funktion melden (Blast-Monitor im Post-Edit-Hook).

**Architecture:** `internal/code/diff` parst die git-Ausgabe rein, `blast.Radius` rechnet Seeds,
Treffer und Testsignal auf dem Graphen, `query` ruft git, lädt, filtert Privacy und schreibt Berichte.
`ask.Refresh` gibt `EnsureFresh` einen Rückgabewert, auf dem `check graph-fresh` steht. `verify`
bekommt die fünfte Art `graph` mit einer Prüfung im Plan (`PlanEnv.GraphReady`); der Hook rechnet
den Monitor aus `store.Read` und `golang.File` ohne Frischeprüfung.

**Tech Stack:** Go (Standardbibliothek), `git` als Kindprozess, `github.com/modelcontextprotocol/go-sdk`
wie vorhanden. **Keine neuen Abhängigkeiten.**

**Spec:** `docs/.superpowers/specs/2026-09-23-loomux-code-g4b-delta.md` (Nachtrag, gilt zuerst) und
`docs/.superpowers/specs/2026-09-22-loomux-code-g4-delta.md`. Bei Widerspruch gilt der Nachtrag, dann
das G4-Delta, dann diese Datei.

---

## Global Constraints

- **Arbeitsort:** Checkout `C:\Users\micro\Documents\#GIT\loomux`, Branch `feat/code-g4b`, abgezweigt
  von `master` `e3cab29c`. Vor jedem Commit `git branch --show-current` und `git rev-parse --short HEAD`
  lesen.
- **Commits:** Conventional Commits, Scope = Code-Bereich (`diff`, `blast`, `query`, `cli`,
  `serve/graph`, `ask`, `verify`, `hooks`, `bench`). Keine Arbeitspapiere im Text (kein „Task 3",
  „G4b", „Plan"). Mehrzeilige Nachrichten über eine Datei und `git commit -F`. Autor ist der Nutzer;
  **kein `Co-Authored-By`**, keine Werbezeile.
- **Niemand außer dem Nutzer pusht.**
- **Coverage 100 % je Funktion**; darunter nur mit `//coverage:exempt <grund>` direkt über `func`.
- **Kein `init()`**, keine Paketvariable, die eingebettete Daten parst.
- **Sprachen:** Code, Bezeichner, Kommentare, Meldungen englisch; Plan, Parität, Spec deutsch.
- **`.loomux/config.toml` schreibt kein Agent.**
- **Tor:** `sh ci/gate.sh` (läuft auch im Pre-Commit). Einzelne Pakete: `go test ./internal/<pkg>/ -count=1`.
- **Nach Änderungen unter `cmd/` oder `internal/`** baut der Pre-Commit `bin/loomux.exe` neu; vor einer
  Messung mit dem Binary selbst bauen: `go build -o bin/loomux.new.exe ./cmd/loomux`, dann
  `go run ./cmd/loomux dev swap-binary --dir bin`.
- **Werte aus der Spec, wörtlich:** `MAX_EVIDENCE = 4`, `MAX_LINES = 6`, Diff-Kappung 24 Zeilen je
  Hunk und 200 je Datei, Schwellenwert-Vorgabe `N = 3`, Sperrwartezeit `30 s`, Monitor höchstens
  zehn Trefferzeilen, E1-Hinweis wörtlich:
  `[graph] Modified struct/interface/type: type coupling not wired in graph v1 (check references via grep)`.
- **git immer als** `git -c core.quotePath=false …`, im Verzeichnis der loomux-Wurzel, `diff` mit
  `--relative -M`.

---

## File Structure

| Datei | Verantwortung |
|---|---|
| `testdata/bench/g4b-post-edit.json`, `testdata/bench/edit-go.json` | Messfall Post-Edit auf eine `.go`-Datei |
| `internal/serve/graph/tools.go` | `parseDepth` berichtigt (Task 1), Handler `graphBlast` (Task 5) |
| `internal/code/diff/diff.go` | `Status`, `File`, `Hunk`, `ParseNameStatus`, `ApplyHunks` |
| `internal/code/blast/radius.go` | `Radius`, `Report`, `Area`, `Seed`, `RadiusHit`, `Evidence`, `Signal`, `IsTestPath` |
| `internal/code/blast/index.go` | `InDegreeWhere` |
| `internal/code/query/git.go` | `gitOutput`-Naht, `runGit` |
| `internal/code/query/blast.go` | `Blast`, `BlastOptions`, `BlastAnswer`, `BlastReport` |
| `internal/code/query/audit.go` | `Audit`, `AuditOptions`, `AuditAnswer`, `AuditReport` |
| `internal/code/query/refresh.go` | `RefreshGraph`, `GraphReady` |
| `internal/code/ask/refresh.go` | `Refresh`, `Status`, `RefreshOptions`, `LockedError`, `ProbeError`, `RebuildError`; `EnsureFresh` als Hülle |
| `internal/cli/graph.go` | `graph blast` |
| `internal/cli/check.go` | `check graph-fresh`, `check blast-audit`, `GraphReady` im `PlanEnv`, Usage |
| `internal/mcptools/tools.go` | Werkzeug `graph_blast` (zwölf) |
| `internal/serve/serve.go` | `Blast: query.Blast` in `Deps` |
| `internal/verify/schema.go`, `plan.go`, `presets.toml`, `report.go` | Art `graph`, `GraphReady`, Preset, `WriteEdit(…, aside)` |
| `internal/hooks/blast_monitor.go` | `blastAside` |
| `internal/hooks/post_edit.go` | Monitor nach den Lanes, `relInRoot` |
| `testdata/cases/graph/blast.golden`, `audit.golden` | Golden-Files |
| Doku | `README*.md`, `docs/{en,de}/{cli-reference,configuration,hooks,migration,benchmarks}.md`, Fusion-Spec, `parity/code-g4.md` |

---

### Task 0: Messung vorher

Der Post-Edit-Hook ändert sich in Task 10; danach gibt es kein „vorher".

**Files:**
- Create: `testdata/bench/edit-go.json`, `testdata/bench/g4b-post-edit.json`
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`

- [ ] **Step 1: Payload anlegen.** `testdata/bench/edit-go.json` (Pfad absolut auf diesen Checkout):

```json
{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"C:/Users/micro/Documents/#GIT/loomux/internal/code/blast/reach.go","old_string":"x","new_string":"x"}}
```

- [ ] **Step 2: Messfall anlegen.** `testdata/bench/g4b-post-edit.json` nach dem Muster von
  `testdata/bench/1a-hooks.json`:

```json
[
  {
    "name": "loomux hook post-tool-use (Edit on a .go file)",
    "dir": "C:/Users/micro/Documents/#GIT/loomux",
    "stdin": "C:/Users/micro/Documents/#GIT/loomux/testdata/bench/edit-go.json",
    "mode": "single",
    "steps": [{"argv": ["C:/Users/micro/Documents/#GIT/loomux/bin/loomux.exe", "hook", "post-tool-use", "--host", "claude", "--root", "C:/Users/micro/Documents/#GIT/loomux"]}]
  }
]
```

- [ ] **Step 3: Binary auf den Stand des Branches bringen und Graph bauen.**

```sh
go build -o bin/loomux.new.exe ./cmd/loomux
go run ./cmd/loomux dev swap-binary --dir bin
bin/loomux.exe graph build
```

- [ ] **Step 4: Messen.** `bin/loomux.exe dev bench-hooks testdata/bench/g4b-post-edit.json -n 10`.
  Ausgabe (Tabelle cold / warm median / min / max / exit codes) sichern. Größe von
  `.loomux/state/graph/wiring.json` notieren (`ls -l`).
- [ ] **Step 5: Eintrag schreiben.** In beiden `benchmarks.md` einen datierten Abschnitt
  `## 2026-09-23 HH:MM — Post-edit on a .go file, before the blast monitor` (de: „… vor dem
  Blast-Monitor"): Methode (Befehl oben, Payload), Tabelle, Graphgröße, Commit `git rev-parse --short HEAD`.
  Der Abschnitt bekommt in Task 11 den Nachher-Teil.
- [ ] **Step 6: Commit.** `docs(bench): record the post-edit hook on a Go file before the blast monitor`
  mit beiden JSON-Dateien und beiden `benchmarks.md`.

---

### Task 1: `parseDepth` rundet auf mindestens 1

Fehler auf `master`: `0.5` wird `Depth(0)`, `EdgeWalk` liefert nichts (§3.2 des G4-Deltas: „alles unter 1 … ist 1").

**Files:**
- Modify: `internal/serve/graph/tools.go` (`parseDepth`)
- Test: `internal/serve/graph/tools_test.go`

- [ ] **Step 1: Failing test**

```go
func TestParseDepthFloorsAndNeverGoesBelowOne(t *testing.T) {
	cases := []struct {
		in   any
		want blast.Depth
	}{
		{0.5, 1}, {2.7, 2}, {3.0, 3}, {-4.0, 1}, {"all", blast.All}, {"FULL", blast.All},
		{"2", 2}, {"0", 1}, {"x", 1}, {nil, 1}, {true, 1},
	}
	for _, c := range cases {
		if got := parseDepth(c.in); got != c.want {
			t.Errorf("parseDepth(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2:** `go test ./internal/serve/graph/ -run TestParseDepth -count=1` → FAIL bei `0.5`.
- [ ] **Step 3: Fix**

```go
	case float64:
		if n := int(math.Floor(val)); n >= 1 {
			return blast.Depth(n)
		}
```

  (Import `math`.)
- [ ] **Step 4:** Test grün, dann `go test ./internal/serve/graph/ -count=1`.
- [ ] **Step 5: Commit** `fix(serve/graph): floor a fractional depth to at least one`

---

### Task 2: Diff-Parser (`internal/code/diff`)

**Files:**
- Create: `internal/code/diff/diff.go`, `internal/code/diff/diff_test.go`

**Interfaces:**
- Produces:

```go
package diff

type Status byte

const (
	Added       Status = 'A'
	Modified    Status = 'M'
	Deleted     Status = 'D'
	Renamed     Status = 'R'
	Copied      Status = 'C'
	TypeChanged Status = 'T'
)

const (
	MaxHunkLines = 24
	MaxFileLines = 200
)

// Hunk is one change range in the post-image. A pure deletion (+c,0) is the
// one line before the gap, and line 1 when the gap is at the top.
type Hunk struct {
	From    int      `json:"from"`
	To      int      `json:"to"`
	Lines   []string `json:"lines,omitempty"`   // '+' and '-' lines, capped
	Omitted int      `json:"omitted,omitempty"` // lines beyond the caps
}

type File struct {
	Status  Status `json:"status"`
	Path    string `json:"path"`
	OldPath string `json:"old_path,omitempty"`
	Hunks   []Hunk `json:"hunks,omitempty"`
}

func ParseNameStatus(r io.Reader) ([]File, error)
func ApplyHunks(files []File, r io.Reader) error
```

`Status` serialisiert als Zahl; `MarshalText` gibt den Buchstaben:

```go
func (s Status) MarshalText() ([]byte, error) { return []byte{byte(s)}, nil }
```

- [ ] **Step 1: Failing tests** (Tabelle, je Fall ein Name):

```go
func TestParseNameStatus(t *testing.T) {
	in := "M\x00a.go\x00R087\x00old name.go\x00neu/ä.go\x00D\x00gone.go\x00A\x00new.go\x00C100\x00x.go\x00y.go\x00T\x00link\x00"
	got, err := ParseNameStatus(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	want := []File{
		{Status: Modified, Path: "a.go"},
		{Status: Renamed, OldPath: "old name.go", Path: "neu/ä.go"},
		{Status: Deleted, Path: "gone.go"},
		{Status: Added, Path: "new.go"},
		{Status: Copied, OldPath: "x.go", Path: "y.go"},
		{Status: TypeChanged, Path: "link"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestParseNameStatusRejects(t *testing.T) {
	for name, in := range map[string]string{
		"unknown status": "X\x00a.go\x00",
		"missing path":   "M\x00",
		"rename one path": "R100\x00a.go\x00",
		"empty status":   "\x00a.go\x00",
	} {
		if _, err := ParseNameStatus(strings.NewReader(in)); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestParseNameStatusEmpty(t *testing.T) {
	got, err := ParseNameStatus(strings.NewReader(""))
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestApplyHunks(t *testing.T) {
	files := []File{{Status: Modified, Path: "a.go"}, {Status: Deleted, Path: "gone.go"}, {Status: Modified, Path: "my file.go"}, {Status: Modified, Path: "ä.go"}, {Status: Modified, Path: "untouched.go"}}
	patch := strings.Join([]string{
		"diff --git a/a.go b/a.go",
		"index 1..2 100644",
		"--- a/a.go",
		"+++ b/a.go",
		"@@ -3 +3 @@ func A() {",
		"-\told()",
		"+\tnew()",
		"@@ -10,2 +9,0 @@",
		"--- a removed line that looks like a header",
		"-x",
		"@@ -1,0 +1,2 @@",
		"+package a",
		"+",
		"diff --git a/gone.go b/gone.go",
		"deleted file mode 100644",
		"--- a/gone.go",
		"+++ /dev/null",
		"@@ -1,2 +0,0 @@",
		"-package gone",
		"-",
		"diff --git a/my file.go b/my file.go",
		"--- a/my file.go\t",
		"+++ b/my file.go\t",
		"@@ -5 +5 @@",
		"-a",
		"+b",
		`diff --git "a/\303\244.go" "b/\303\244.go"`,
		`--- "a/\303\244.go"`,
		`+++ "b/\303\244.go"`,
		"@@ -2 +2 @@",
		"-a",
		"+b",
		"\\ No newline at end of file",
		"",
	}, "\n")
	if err := ApplyHunks(files, strings.NewReader(patch)); err != nil {
		t.Fatal(err)
	}
	a := files[0].Hunks
	if len(a) != 3 || a[0].From != 3 || a[0].To != 3 || a[1].From != 9 || a[1].To != 9 || a[2].From != 1 || a[2].To != 2 {
		t.Fatalf("a.go hunks %+v", a)
	}
	if a[1].Lines[0] != "--- a removed line that looks like a header" {
		t.Fatalf("header-like removed line lost: %q", a[1].Lines)
	}
	if g := files[1].Hunks; len(g) != 1 || g[0].From != 1 || g[0].To != 1 {
		t.Fatalf("deleted file hunk %+v", g)
	}
	if len(files[2].Hunks) != 1 || len(files[3].Hunks) != 1 || files[4].Hunks != nil {
		t.Fatalf("space/umlaut/untouched: %+v", files[2:])
	}
}

func TestApplyHunksCaps(t *testing.T) {
	var b strings.Builder
	b.WriteString("--- a/big.go\n+++ b/big.go\n")
	for h := 0; h < 10; h++ {
		fmt.Fprintf(&b, "@@ -%d,30 +%d,30 @@\n", h*100+1, h*100+1)
		for i := 0; i < 30; i++ {
			b.WriteString("-o\n")
		}
		for i := 0; i < 30; i++ {
			b.WriteString("+n\n")
		}
	}
	files := []File{{Status: Modified, Path: "big.go"}}
	if err := ApplyHunks(files, strings.NewReader(b.String())); err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, h := range files[0].Hunks {
		if len(h.Lines) > MaxHunkLines {
			t.Fatalf("hunk over cap: %d", len(h.Lines))
		}
		if len(h.Lines)+h.Omitted != 60 {
			t.Fatalf("lines %d + omitted %d != 60", len(h.Lines), h.Omitted)
		}
		total += len(h.Lines)
	}
	if total != MaxFileLines || len(files[0].Hunks) != 10 || files[0].Hunks[9].To != 930 {
		t.Fatalf("file cap: total %d, hunks %d", total, len(files[0].Hunks))
	}
}

func TestApplyHunksRejectsABrokenHeader(t *testing.T) {
	files := []File{{Status: Modified, Path: "a.go"}}
	for _, h := range []string{"@@ -1 +x @@", "@@ nothing @@", "@@ -1 +2,y @@"} {
		if err := ApplyHunks(files, strings.NewReader("--- a/a.go\n+++ b/a.go\n"+h+"\n")); err == nil {
			t.Errorf("%q: want error", h)
		}
	}
	if err := ApplyHunks(files, strings.NewReader(`+++ "b/\x.go"`+"\n")); err == nil {
		t.Error("bad quoting: want error")
	}
}
```

  Außerdem ein Test, dass ein Lesefehler des Readers durchgereicht wird (`iotest.ErrReader`), je für
  beide Funktionen.
- [ ] **Step 2:** `go test ./internal/code/diff/ -count=1` → FAIL (Paket fehlt).
- [ ] **Step 3: Implementierung**

```go
// Package diff reads what git says about a change: which files, and which
// lines of their post-image. It parses and never runs git; the caller hands in
// the output of `git diff --name-status -z` and of `git diff --unified=0`.
package diff

func ParseNameStatus(r io.Reader) ([]File, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	fields := strings.Split(string(raw), "\x00")
	var out []File
	for i := 0; i < len(fields); i++ {
		tok := fields[i]
		if tok == "" && i == len(fields)-1 {
			break
		}
		if tok == "" {
			return nil, fmt.Errorf("name-status: empty status at field %d", i)
		}
		st := Status(tok[0])
		paths := 1
		switch st {
		case Renamed, Copied:
			paths = 2
		case Added, Modified, Deleted, TypeChanged:
		default:
			return nil, fmt.Errorf("name-status: unknown status %q", tok)
		}
		if i+paths >= len(fields) || fields[i+paths] == "" {
			return nil, fmt.Errorf("name-status: %q without its path", tok)
		}
		f := File{Status: st, Path: fields[i+paths]}
		if paths == 2 {
			f.OldPath = fields[i+1]
		}
		out = append(out, f)
		i += paths
	}
	return out, nil
}

func ApplyHunks(files []File, r io.Reader) error {
	byPath := make(map[string]*File, len(files))
	for i := range files {
		byPath[files[i].Path] = &files[i]
	}
	br := bufio.NewReader(r)
	var cur *File
	minus := ""
	fileLines := 0
	oldLeft, newLeft := 0, 0
	for {
		line, err := br.ReadString('\n')
		if line == "" && err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if oldLeft > 0 || newLeft > 0 {
			switch {
			case strings.HasPrefix(line, "-"):
				oldLeft--
			case strings.HasPrefix(line, "+"):
				newLeft--
			default: // "\ No newline at end of file"
				continue
			}
			if cur != nil {
				h := &cur.Hunks[len(cur.Hunks)-1]
				if len(h.Lines) < MaxHunkLines && fileLines < MaxFileLines {
					h.Lines = append(h.Lines, line)
					fileLines++
				} else {
					h.Omitted++
				}
			}
			continue
		}
		switch {
		case strings.HasPrefix(line, "diff --git "):
			cur, minus, fileLines = nil, "", 0
		case strings.HasPrefix(line, "--- "):
			p, err := headerPath(line[4:])
			if err != nil {
				return err
			}
			minus = p
		case strings.HasPrefix(line, "+++ "):
			p, err := headerPath(line[4:])
			if err != nil {
				return err
			}
			if p == "" {
				p = minus
			}
			cur, fileLines = byPath[p], 0
		case strings.HasPrefix(line, "@@ "):
			h, o, n, err := parseHunkHeader(line)
			if err != nil {
				return err
			}
			oldLeft, newLeft = o, n
			if cur != nil {
				cur.Hunks = append(cur.Hunks, h)
			}
		}
		if err == io.EOF {
			return nil
		}
	}
}

// headerPath is the path of a ---/+++ line without its a/ or b/ prefix, ""
// for /dev/null. Git ends a name holding a space with a tab and C-quotes a
// name holding a quote, a backslash or a control character.
func headerPath(s string) (string, error) {
	s = strings.TrimSuffix(s, "\t")
	if s == "/dev/null" {
		return "", nil
	}
	if strings.HasPrefix(s, `"`) {
		u, err := strconv.Unquote(s)
		if err != nil {
			return "", fmt.Errorf("diff: cannot unquote %s: %w", s, err)
		}
		s = u
	}
	if len(s) > 2 && (s[:2] == "a/" || s[:2] == "b/") {
		s = s[2:]
	}
	return s, nil
}

// parseHunkHeader reads "@@ -a[,b] +c[,d] @@".
func parseHunkHeader(line string) (Hunk, int, int, error) {
	parts := strings.Fields(line)
	if len(parts) < 3 || !strings.HasPrefix(parts[1], "-") || !strings.HasPrefix(parts[2], "+") {
		return Hunk{}, 0, 0, fmt.Errorf("diff: bad hunk header %q", line)
	}
	_, oldCount, err := rangeOf(parts[1][1:])
	if err != nil {
		return Hunk{}, 0, 0, fmt.Errorf("diff: bad hunk header %q: %w", line, err)
	}
	start, newCount, err := rangeOf(parts[2][1:])
	if err != nil {
		return Hunk{}, 0, 0, fmt.Errorf("diff: bad hunk header %q: %w", line, err)
	}
	if newCount == 0 {
		at := max(start, 1)
		return Hunk{From: at, To: at}, oldCount, 0, nil
	}
	return Hunk{From: start, To: start + newCount - 1}, oldCount, newCount, nil
}

func rangeOf(s string) (start, count int, err error) {
	a, b, found := strings.Cut(s, ",")
	if start, err = strconv.Atoi(a); err != nil {
		return 0, 0, err
	}
	if !found {
		return start, 1, nil
	}
	count, err = strconv.Atoi(b)
	return start, count, err
}
```

- [ ] **Step 4:** Tests grün; `go test ./internal/code/diff/ -count=1 -coverprofile=c.out && go tool cover -func=c.out`
  → jede Funktion 100 %. Lücken mit weiteren Fällen schließen, nicht mit `exempt`.
- [ ] **Step 5: Commit** `feat(diff): parse git name-status and zero-context hunks`

---

### Task 3: `blast.Radius` und `InDegreeWhere`

**Files:**
- Create: `internal/code/blast/radius.go`, `internal/code/blast/radius_test.go`
- Modify: `internal/code/blast/index.go`, `internal/code/blast/index_test.go`

**Interfaces:**
- Consumes: `diff.File`, `diff.Hunk`, `model.FileSpans`, `model.Innermost`, `(*Index).Reach`, `(*Index).InDegree`
- Produces:

```go
func (x *Index) InDegreeWhere(id model.NodeID, keep func(*model.Node) bool) int
func IsTestPath(path string) bool // Go: strings.HasSuffix(path, "_test.go")

type Signal string

const (
	SignalChanged Signal = "changed"
	SignalStale   Signal = "stale"
	SignalNone    Signal = "none"
	SignalNA      Signal = "na"
)

const (
	MaxEvidence      = 4
	MaxEvidenceLines = 6
)

type Seed struct {
	Node     *model.Node `json:"node"`
	InDegree int         `json:"in_degree"`
}

type Area struct {
	Path   string      `json:"path"`
	Status diff.Status `json:"status"`
	Seeds  []Seed      `json:"seeds"`
	Signal Signal      `json:"signal"`
	Tests  []string    `json:"tests,omitempty"`
}

type RadiusHit struct {
	Hit
	From []string `json:"from"` // changed files whose walk reached it
}

type Evidence struct {
	Node  *model.Node `json:"node"`
	Lines []string    `json:"lines"`
	More  int         `json:"more,omitempty"`
}

type Report struct {
	Areas        []Area      `json:"areas"`
	Hits         []RadiusHit `json:"hits"`
	Deleted      []string    `json:"deleted,omitempty"`
	Unindexed    []string    `json:"unindexed,omitempty"`
	Evidence     []Evidence  `json:"evidence,omitempty"`
	MoreEvidence int         `json:"more_evidence,omitempty"`
}

func Radius(g *model.Graph, x *Index, changed []diff.File, depth Depth) Report
```

`Hit` bekommt JSON-Tags (`id`, `node`, `relation`, `depth`), falls es keine hat.

- [ ] **Step 1: Failing tests.** Ein Testgraph im Code (kein Extraktor): Dateien `lib.go` (`Add` L3-L5,
  `Sub` L7-L9, `T` struct L11-L13), `main.go` (`main` L3-L6, ruft `Add`), `lib_test.go` (`TestAdd` L3-L6,
  ruft `helper`), `helper_test.go` (`helper` L3-L5, ruft `Add`), `other.go` (keine Aufrufer), Dateiknoten
  für jede Datei, `contains`-Kanten Datei→Symbol, eine `imports`-Kante `main.go`→`lib.go`. Fälle:

  1. Hunk L4 in `lib.go` ⇒ Seed `Add`, Treffer `main` (d1, from `lib.go`), `helper` (d1),
     `TestAdd` nur mit `depth=All` (d2); Signal `stale`, Tests `[helper_test.go, lib_test.go]`.
  2. Dasselbe plus `lib_test.go` im Diff ⇒ Signal `changed`.
  3. Hunk L8 (`Sub`, keine Aufrufer) ⇒ Signal `none`, keine Treffer.
  4. Hunk L12 (`T`) ⇒ Signal `na` (kein behaviouraler Seed).
  5. Hunk L1 (Paketzeile, kein Symbol) ⇒ Seed ist der Dateiknoten `lib.go`, Treffer `main.go` über
     `imports`, **keine** Expansion auf `Add`/`Sub`.
  6. `lib_test.go` geändert ⇒ Bereich mit Signal `na`.
  7. `Deleted` ⇒ nur in `Report.Deleted`, nicht gewalkt; unbekannte Datei ⇒ `Unindexed`.
  8. Zwei geänderte Dateien, die denselben Knoten erreichen ⇒ ein Treffer, flachste Tiefe, `From` mit
     beiden Pfaden in Diff-Reihenfolge.
  9. Belege: fünf geänderte Symbole ⇒ vier `Evidence`, `MoreEvidence == 1`; ein Hunk mit neun Zeilen
     im Span ⇒ sechs Zeilen und `More == 3`; der Dateiknoten bekommt keinen Beleg.
  10. `InDegreeWhere(Add, func(n) !IsTestPath(n.Path))` == 1, `InDegree(Add)` == 2; `keep` bekommt
      `nil` für eine Kante aus einem unbekannten Knoten.
  11. Ein Treffer ohne Knoten (Kante auf eine Id, die kein Knoten ist) fällt weg.

- [ ] **Step 2:** `go test ./internal/code/blast/ -count=1` → FAIL.
- [ ] **Step 3: Implementierung**

```go
// InDegreeWhere counts the incoming walk edges of id whose source keep
// accepts. keep sees nil for a source that is not a node of the graph.
func (x *Index) InDegreeWhere(id model.NodeID, keep func(*model.Node) bool) int {
	n := 0
	for _, l := range x.in[id] {
		if keep(x.nodes[l.other]) {
			n++
		}
	}
	return n
}
```

```go
// Radius is the blast radius of a change: per changed file the symbols its
// hunks touch, what reaches them, and whether a test that reaches them
// changed too.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/blast/blast.ts, with the
// corrections of the G4 delta: the file node is a seed only when no symbol
// was hit, and it is walked as itself, not expanded.
func Radius(g *model.Graph, x *Index, changed []diff.File, depth Depth) Report {
	spans := model.FileSpans(g)
	files := map[string]*model.Node{}
	for i := range g.Nodes {
		if g.Nodes[i].Kind == model.KindFile {
			files[g.Nodes[i].Path] = &g.Nodes[i]
		}
	}
	inDiff := map[string]bool{}
	for _, f := range changed {
		inDiff[f.Path] = true
	}
	rep := Report{}
	hitAt := map[model.NodeID]int{}
	var evidenceSeeds []evidenceSeed
	for _, f := range changed {
		if f.Status == diff.Deleted {
			rep.Deleted = append(rep.Deleted, f.Path)
			continue
		}
		fileNode := files[f.Path]
		if fileNode == nil {
			rep.Unindexed = append(rep.Unindexed, f.Path)
			continue
		}
		seeds := seedsOf(spans[f.Path], f.Hunks)
		if len(seeds) == 0 {
			seeds = []*model.Node{fileNode}
		} else {
			for _, s := range seeds {
				evidenceSeeds = append(evidenceSeeds, evidenceSeed{s, f.Hunks})
			}
		}
		area := Area{Path: f.Path, Status: f.Status}
		ids := make([]model.NodeID, len(seeds))
		for i, s := range seeds {
			ids[i] = s.ID
			area.Seeds = append(area.Seeds, Seed{Node: s, InDegree: x.InDegree(s.ID)})
		}
		area.Signal, area.Tests = signal(x, f.Path, seeds, inDiff)
		rep.Areas = append(rep.Areas, area)
		for _, h := range x.Reach(ids, In, depth) {
			if h.Node == nil {
				continue
			}
			if i, seen := hitAt[h.ID]; seen {
				if h.Depth < rep.Hits[i].Depth {
					rep.Hits[i].Depth, rep.Hits[i].Relation = h.Depth, h.Relation
				}
				if !slices.Contains(rep.Hits[i].From, f.Path) {
					rep.Hits[i].From = append(rep.Hits[i].From, f.Path)
				}
				continue
			}
			hitAt[h.ID] = len(rep.Hits)
			rep.Hits = append(rep.Hits, RadiusHit{Hit: h, From: []string{f.Path}})
		}
	}
	sort.SliceStable(rep.Hits, func(i, j int) bool { return rep.Hits[i].Depth < rep.Hits[j].Depth })
	rep.Evidence, rep.MoreEvidence = evidence(evidenceSeeds)
	return rep
}

// seedsOf is the innermost symbols every hunk touches, each once, in the
// order the hunks meet them.
func seedsOf(spans []model.SymbolSpan, hunks []diff.Hunk) []*model.Node {
	var out []*model.Node
	seen := map[model.NodeID]bool{}
	for _, h := range hunks {
		for _, n := range model.Innermost(spans, h.From, h.To) {
			if !seen[n.ID] {
				seen[n.ID] = true
				out = append(out, n)
			}
		}
	}
	return out
}

// signal says whether a test covers the area and changed with it. The walk
// to the tests is always the closure: a test that reaches through a helper
// still reaches.
func signal(x *Index, path string, seeds []*model.Node, inDiff map[string]bool) (Signal, []string) {
	if IsTestPath(path) {
		return SignalNA, nil
	}
	var ids []model.NodeID
	for _, s := range seeds {
		if s.Kind == "function" || s.Kind == "method" {
			ids = append(ids, s.ID)
		}
	}
	if len(ids) == 0 {
		return SignalNA, nil
	}
	set := map[string]bool{}
	for _, h := range x.Reach(ids, In, All) {
		if h.Node != nil && IsTestPath(h.Node.Path) {
			set[h.Node.Path] = true
		}
	}
	tests := slices.Sorted(maps.Keys(set))
	switch {
	case len(tests) == 0:
		return SignalNone, nil
	case slices.ContainsFunc(tests, func(t string) bool { return inDiff[t] }):
		return SignalChanged, tests
	}
	return SignalStale, tests
}

type evidenceSeed struct {
	node  *model.Node
	hunks []diff.Hunk
}

// evidence quotes the hunks inside the first MaxEvidence seeds' spans.
func evidence(seeds []evidenceSeed) ([]Evidence, int) {
	var out []Evidence
	for i, s := range seeds {
		if i == MaxEvidence {
			return out, len(seeds) - MaxEvidence
		}
		from, to, _ := s.node.Span.Lines()
		var lines []string
		for _, h := range s.hunks {
			if h.To >= from && h.From <= to {
				lines = append(lines, h.Lines...)
			}
		}
		e := Evidence{Node: s.node, Lines: lines}
		if len(lines) > MaxEvidenceLines {
			e.Lines, e.More = lines[:MaxEvidenceLines], len(lines)-MaxEvidenceLines
		}
		out = append(out, e)
	}
	return out, 0
}

func IsTestPath(path string) bool { return strings.HasSuffix(path, "_test.go") }
```

  `Innermost` liefert nur Knoten mit lesbarem Span (FileSpans filtert), daher ist das `_` in
  `evidence` sicher; das steht als Kommentar dort.
- [ ] **Step 4:** Tests grün, Coverage 100 % je Funktion in `blast`.
- [ ] **Step 5: Commit** `feat(blast): compute the blast radius of a diff with a test signal`

---

### Task 4: `query.Blast`, `graph blast`, Golden

**Files:**
- Create: `internal/code/query/git.go`, `internal/code/query/git_test.go`, `internal/code/query/blast.go`, `internal/code/query/blast_test.go`, `testdata/cases/graph/blast.golden`
- Modify: `internal/code/query/golden_test.go`, `internal/cli/graph.go`, `internal/cli/graph_test.go`

**Interfaces:**
- Consumes: `diff.ParseNameStatus`, `diff.ApplyHunks`, `blast.Radius`, `loadGraph`
- Produces:

```go
// git.go
var gitOutput = runGit // seam: tests replace it to reach git's failure arms
func runGit(dir string, args ...string) ([]byte, error)

// blast.go
type BlastOptions struct {
	Base      string
	Cached    bool
	Depth     blast.Depth // <= 0 and not All means 1
	NoRefresh bool
	Keep      func(path string) bool
}
type BlastAnswer struct {
	Range string `json:"range"`
	blast.Report
	Hidden int `json:"hidden,omitempty"`
}
var ErrBaseAndCached = errors.New("--base and --cached exclude each other")
func Blast(root string, opts BlastOptions) (BlastAnswer, []string, error)
func BlastReport(a BlastAnswer) string
```

- [ ] **Step 1: Test-Helfer** in `blast_test.go` (Paket `query`):

```go
func gitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyTree(t, filepath.Join("..", "..", "..", "testdata", "cases", "graph", "repo"), root)
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"}, {"add", "."}, {"commit", "-qm", "init"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, _, err := Build(root, func(string) {}); err != nil {
		t.Fatal(err)
	}
	return root
}
```

  `copyTree` ist die Kopie aus `golden_test.go` (dort Paket `query_test`) — im internen Test neu
  geschrieben, 15 Zeilen, oder `golden_test.go` bekommt einen Blast-Fall im externen Paket; der
  Helfer wird dann dort eingefügt. Entscheidung: **Blast-Golden im externen `golden_test.go`, die
  Fehlerarme im internen `blast_test.go`** (dort ohne Repo, `gitOutput` ersetzt).
- [ ] **Step 2: Failing tests**
  - Golden (`golden_test.go`): Repo wie oben, `calc/calc.go` Zeile 5 `return a + b` → `return b + a`,
    `query.Blast(root, query.BlastOptions{NoRefresh: true})` ⇒ `BlastReport` gegen
    `testdata/cases/graph/blast.golden`. Erwartung (von Hand prüfen nach `-update`): Bereich
    `calc/calc.go` mit Seed `Add`, Signal `stale`, Treffer `TestAdd` und `main` (d1), ein Beleg mit
    `-\treturn a + b` / `+\treturn b + a`.
  - Sauberer Baum ⇒ `Range == "HEAD~1...HEAD"`; im Repo mit einem einzigen Commit ist das ein
    git-Fehler ⇒ `err != nil` mit „HEAD~1" im Text. Zweiter Commit dazu ⇒ Blast über ihn.
  - `--cached`: Änderung gestagt ⇒ Treffer; ungestagt ⇒ leerer Bericht (keine Bereiche).
  - `Base: "HEAD~1"` nach einem zweiten Commit ⇒ `Range == "HEAD~1...HEAD"` und die Bereiche dieses
    Commits.
  - `Base` und `Cached` zusammen ⇒ `ErrBaseAndCached`.
  - Kein Graph ⇒ `ErrNoGraph`.
  - Privacy: `Keep` lehnt `main.go` ab ⇒ `main` fehlt in den Treffern, `Hidden == 1`; lehnt
    `calc/calc.go` ab ⇒ keine Bereiche, `Hidden == 1`.
  - Intern: `gitOutput` gibt einen Fehler beim ersten bzw. zweiten Aufruf ⇒ Fehler; gibt kaputtes
    name-status (`"X\x00a\x00"`) ⇒ Fehler; kaputten Hunk-Header ⇒ Fehler.
  - `runGit` in einem Nicht-Repo ⇒ Fehler, der die stderr-Zeile von git enthält.
  - `BlastReport` mit `Deleted`, `Unindexed`, `MoreEvidence`, `Evidence.More` und `Hidden` (Tabelle,
    erwarteter Text wörtlich).
  - CLI (`graph_test.go`): `graph blast --root <repo> --no-refresh` ⇒ 0 und Bericht;
    `--json` ⇒ gültiges JSON mit `range`; `--base X --cached` ⇒ 2; unbekanntes Flag ⇒ 2;
    Positionsargument ⇒ 2; `--depth 0` ⇒ 2; fehlender Graph ⇒ 1.
- [ ] **Step 3: Implementierung**

```go
// runGit runs git in dir with paths left unquoted, and keeps git's own
// complaint in the error: an exit status alone does not say what was wrong.
func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-c", "core.quotePath=false"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out, fmt.Errorf("git %s: %w: %s", args[0], err, bytes.TrimSpace(exit.Stderr))
	}
	return out, err
}
```

```go
// Blast is the blast radius of a change as git sees it: the index against
// HEAD with Cached, Base...HEAD with Base, otherwise the working tree against
// HEAD and, when that is clean, the last commit.
func Blast(root string, opts BlastOptions) (BlastAnswer, []string, error) {
	if opts.Base != "" && opts.Cached {
		return BlastAnswer{}, nil, ErrBaseAndCached
	}
	g, notes, err := loadGraph(root, opts.NoRefresh)
	if err != nil {
		return BlastAnswer{}, notes, err
	}
	rng, rev := "working tree against HEAD", []string{"HEAD"}
	switch {
	case opts.Cached:
		rng, rev = "index against HEAD", []string{"--cached", "HEAD"}
	case opts.Base != "":
		rng = opts.Base + "...HEAD"
		rev = []string{rng}
	}
	files, err := changedFiles(root, rev)
	if err == nil && len(files) == 0 && opts.Base == "" && !opts.Cached {
		rng, rev = "HEAD~1...HEAD", []string{"HEAD~1...HEAD"}
		files, err = changedFiles(root, rev)
	}
	if err != nil {
		return BlastAnswer{}, notes, err
	}
	ans := BlastAnswer{Range: rng}
	if opts.Keep != nil {
		kept := files[:0]
		for _, f := range files {
			if opts.Keep(f.Path) {
				kept = append(kept, f)
			} else {
				ans.Hidden++
			}
		}
		files = kept
	}
	depth := opts.Depth
	if depth <= 0 && depth != blast.All {
		depth = 1
	}
	ans.Report = blast.Radius(g, blast.New(g), files, depth)
	if opts.Keep != nil {
		hits := ans.Hits[:0]
		for _, h := range ans.Hits {
			if opts.Keep(h.Node.Path) {
				hits = append(hits, h)
			} else {
				ans.Hidden++
			}
		}
		ans.Hits = hits
	}
	return ans, notes, nil
}

// changedFiles asks git twice: which files, then which lines.
func changedFiles(root string, rev []string) ([]diff.File, error) {
	names, err := gitOutput(root, append([]string{"diff", "--relative", "-M", "--name-status", "-z"}, rev...)...)
	if err != nil {
		return nil, err
	}
	files, err := diff.ParseNameStatus(bytes.NewReader(names))
	if err != nil || len(files) == 0 {
		return files, err
	}
	patch, err := gitOutput(root, append([]string{"diff", "--relative", "-M", "--unified=0", "--no-color", "--no-ext-diff"}, rev...)...)
	if err != nil {
		return nil, err
	}
	return files, diff.ApplyHunks(files, bytes.NewReader(patch))
}
```

  `BlastReport` (Text, eine Zeile je Fakt, stabil für Golden):

```
blast radius: <Range>
<Path> [<Signal>]            ← je Bereich; Status D/A/R vorangestellt wie "R old -> new" nicht nötig
  seed <Name> (<Kind>) <Span> in-degree <n>
  tests: <t1>, <t2>          ← nur wenn Tests
reached:
  <Name> (<Kind>, depth <d>, <Relation>) in <Path>:<Span> from <f1>, <f2>
deleted: <path>
not indexed: <path>
evidence:
  <Name> in <Path>
    <line>
    … +<More> lines
  … <MoreEvidence> more symbols
hidden: <n>
```

  Leere Abschnitte fallen weg; ohne Bereiche und ohne gelöschte Dateien eine Zeile `no change`.
  Ein Dateiknoten als Seed wird `seed <Path> (file)` ohne Span.
- CLI `graphBlast` nach dem Muster von `graphCallers` (Flags `--root`, `--base`, `--cached`,
  `-d`/`--depth` mit `1`, `--no-refresh`, `--json`; keine Positionsargumente; Notes auf stderr mit
  Präfix `loomux graph blast: `), eingetragen in `graphCommand` und `graphUsage`
  (`  blast     show what a change reaches, from git's diff`). Der `MarshalIndent`-Arm bekommt
  dasselbe `//coverage:exempt` wie `graphSkeleton`, mit dem Grund „BlastAnswer is built of strings,
  ints, slices and model types".
- [ ] **Step 4:** `go test ./internal/code/query/ -run Golden -update -count=1`, `blast.golden` lesen
  und gegen die Erwartung aus Step 2 prüfen; dann ohne `-update` und alle Tests grün, Coverage 100 %.
- [ ] **Step 5: Commit** `feat(query): report the blast radius of a git diff` (Body: der Bereich je
  Modus und der Rückfall auf `HEAD~1...HEAD`), zusammen mit `graph blast` und Golden.

---

### Task 5: MCP-Werkzeug `graph_blast`

**Files:**
- Modify: `internal/mcptools/tools.go`, `internal/mcptools/tools_test.go`, `internal/serve/graph/tools.go`, `internal/serve/graph/tools_test.go`, `internal/serve/serve.go`

**Interfaces:**
- Consumes: `query.Blast`, `query.BlastOptions`, `query.BlastReport`, `parseDepth`, `readable`, `resolve`
- Produces: `Deps.Blast func(root string, opts query.BlastOptions) (query.BlastAnswer, []string, error)`; Werkzeug Nr. 12.

- [ ] **Step 1: Failing tests**
  - `mcptools`: zwölf Werkzeuge, `graph_blast` mit `required: ["scope"]`, Properties `scope`, `base`
    (string), `depth` (ohne `type`, wie `graph_trace_calls`), kein `default`; der Kommentar
    „the brain's five, then the graph's six" wird „… seven".
  - `serve/graph`: ohne `scope` ⇒ Failure `graph_blast requires a scope`; lokal ⇒ Text von
    `BlastReport`, Notes vorn; Cloud ⇒ keine Notes, Fehlertext `cloudFailure` bei einem git-Fehler,
    `ErrNoGraph` bleibt lesbar; `Keep` ist gesetzt (Fake prüft `opts.Keep("secret/x.go") == false`
    für eine `never`-Regel); `depth: 0.5` kommt als `1` an; `base` wird durchgereicht.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implementierung.** Definition in `mcptools.build()` hinter `graph_repo_map`, vor
  `graph_check_freshness`:

```go
		{
			Name:        "graph_blast",
			Description: "Show what a change reaches: the symbols a git diff touches, their callers, and whether a test that reaches them changed too.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope": areaScope,
					"base":  map[string]any{"type": "string", "description": "compare base...HEAD; empty compares the working tree with HEAD, or the last commit when the tree is clean"},
					"depth": map[string]any{"description": "depth limit as integer or 'all' for full transitive closure"},
				},
				"required": []string{"scope"},
			},
		},
```

  Handler nach dem Muster von `traceCalls`:

```go
func graphBlast(channel privacy.Channel, deps Deps) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := arguments(req)
		area, refusal := resolve("graph_blast", channel, deps, str(args, "scope"))
		if refusal != nil {
			return refusal, nil
		}
		answer, notes, err := deps.Blast(area.Area.Path, query.BlastOptions{
			Base:  str(args, "base"),
			Depth: parseDepth(args["depth"]),
			Keep:  readable(area.Manifest),
		})
		if channel == privacy.ChannelCloud {
			notes = nil
		}
		for _, note := range notes {
			report(ctx, req, note)
		}
		if err != nil {
			return failure(withNotes(notes, errorText(channel, err))), nil
		}
		return success(withNotes(notes, strings.TrimSuffix(query.BlastReport(answer), "\n"))), nil
	}
}
```

  In `Register` `"graph_blast": graphBlast(channel, deps)`; in `serve.go` `Blast: query.Blast`.
  Prüfen, ob `resolve` einen leeren `scope` schon mit `graph_blast requires a scope` abweist (ja,
  `resolve` Zeile 1).
- [ ] **Step 4:** Tests grün, Coverage.
- [ ] **Step 5: Commit** `feat(serve/graph): serve the blast radius as graph_blast`

---

### Task 6: `ask.Refresh`, `query.RefreshGraph`, `check graph-fresh`

**Files:**
- Modify: `internal/code/ask/refresh.go`, `internal/code/ask/refresh_test.go`, `internal/cli/check.go`, `internal/cli/check_test.go`
- Create: `internal/code/query/refresh.go`, `internal/code/query/refresh_test.go`

**Interfaces:**
- Produces:

```go
// ask
type Status int

const (
	StatusClean Status = iota
	StatusRebuilt
)

type RefreshOptions struct {
	Force  string        // a reason to rebuild although the probe is clean; "" probes
	Wait   time.Duration // how long to wait for a held lock; 0 does not wait
	Notice func(string)
}

type ProbeError struct{ Err error }   // Error(): "freshness probe failed: <err>"; Unwrap
type RebuildError struct{ Err error } // Error(): "rebuild failed: <err>"; Unwrap
type LockedError struct {
	Path string
	Age  time.Duration
}                                       // Error(): "another run holds the rebuild lock <path> (<age> old)"

func Refresh(root, extractor string, rebuild Rebuild, opts RefreshOptions) (Status, error)

// query
func RefreshGraph(root string, wait time.Duration, notice func(string)) (ask.Status, error)
```

- [ ] **Step 1: Failing tests (ask)**
  - Sauber ⇒ `StatusClean, nil`, kein Neubau.
  - Drift ⇒ Notice `N files moved, rebuilding the graph`, `StatusRebuilt`.
  - `Force: "x"` bei sauberem Baum ⇒ Notice `x`, `StatusRebuilt`.
  - Probe-Fehler ⇒ `*ProbeError`, kein Neubau.
  - Neubau-Fehler ⇒ `*RebuildError`, Sperre danach frei.
  - Gehaltene, frische Sperre, `Wait: 0` ⇒ `*LockedError` mit `Path == LockPath(root)`;
    `Wait: 300ms` und die Sperre wird nach 100 ms frei (Goroutine löscht sie) ⇒ `StatusRebuilt`.
  - Alle bestehenden `EnsureFresh`-Tests bleiben unverändert grün (dieselben Notice-Texte:
    `freshness probe failed, answering from the graph on disk: …`,
    `another run is rebuilding, answering from the graph on disk`,
    `rebuild failed, answering from the graph on disk: …`).
- [ ] **Step 2: Failing tests (query)** `RefreshGraph`:
  - kein `wiring.json` ⇒ `ErrNoGraph`;
  - `wiring.json` mit `meta.version` 1 ⇒ Force-Grund `graph schema is outdated, rebuilding` ⇒
    `StatusRebuilt`, danach liest `store.Read` den Graphen;
  - `meta.extractor` `"go/0"` ⇒ Grund `graph was built by extractor "go/0", rebuilding`;
  - kaputtes JSON ⇒ Grund `graph is unreadable (…), rebuilding`;
  - frischer Graph ⇒ `StatusClean`.
- [ ] **Step 3: Failing tests (CLI)** `check graph-fresh --root <repo>`: frisch ⇒ 0, Meldung
  `graph is fresh`; Drift ⇒ 0 und `graph rebuilt`; Sperre gehalten mit `--wait 0s` ⇒ 1 und Pfad der
  Sperre auf stderr; kein Graph ⇒ 1 mit `ErrNoGraph`-Text; `--wait x` ⇒ 2.
- [ ] **Step 4: Implementierung (ask).** Der Rumpf von `EnsureFresh` wandert nach `Refresh`:

```go
func Refresh(root, extractor string, rebuild Rebuild, opts RefreshOptions) (Status, error) {
	say := func(format string, args ...any) {
		if opts.Notice != nil {
			opts.Notice(fmt.Sprintf(format, args...))
		}
	}
	if opts.Force != "" {
		say("%s", opts.Force)
	} else {
		drift, err := freshness.Probe(root, extractor)
		if err != nil {
			return StatusClean, &ProbeError{err}
		}
		switch {
		case drift == nil:
			say("no freshness record, building the graph")
		case !drift.Clean():
			say("%d files moved, rebuilding the graph", drift.Count())
		case !lexicon.Usable(root):
			say("no ask index, rebuilding the graph")
		default:
			return StatusClean, nil
		}
	}
	deadline := time.Now().Add(opts.Wait)
	for !lock(root) {
		if !time.Now().Before(deadline) {
			locked := &LockedError{Path: LockPath(root)}
			if info, err := os.Stat(locked.Path); err == nil {
				locked.Age = time.Since(info.ModTime()).Round(time.Second)
			}
			return StatusClean, locked
		}
		time.Sleep(lockPoll)
	}
	defer unlock(root)
	if err := rebuild(); err != nil {
		return StatusClean, &RebuildError{err}
	}
	return StatusRebuilt, nil
}

// lockPoll is how often a waiting run looks at the lock again.
const lockPoll = 100 * time.Millisecond

func EnsureFresh(root, extractor string, rebuild Rebuild, notice func(string)) {
	_, err := Refresh(root, extractor, rebuild, RefreshOptions{Notice: notice})
	if err == nil || notice == nil {
		return
	}
	var probe *ProbeError
	var locked *LockedError
	var failed *RebuildError
	switch {
	case errors.As(err, &probe):
		notice(fmt.Sprintf("freshness probe failed, answering from the graph on disk: %v", probe.Err))
	case errors.As(err, &locked):
		notice("another run is rebuilding, answering from the graph on disk")
	case errors.As(err, &failed):
		notice(fmt.Sprintf("rebuild failed, answering from the graph on disk: %v", failed.Err))
	}
}
```

  Der lange Kommentar über `EnsureFresh` zieht zu `Refresh`; `EnsureFresh` behält einen Satz:
  „EnsureFresh is Refresh for a question: it never fails, and says why it answers from the graph on
  disk." Die Reihenfolge Sperre-dann-„no freshness record"-Notice bleibt wie heute (die Notice kommt
  vor dem Sperrversuch).
- [ ] **Step 5: Implementierung (query)**

```go
// RefreshGraph drives the graph at root to fresh for a gate: drift, a missing
// record, an outdated schema and a foreign extractor all rebuild. Only a
// missing graph, a failed probe or rebuild and a lock held past wait fail.
func RefreshGraph(root string, wait time.Duration, notice func(string)) (ask.Status, error) {
	if _, err := os.Stat(store.WiringPath(root)); errors.Is(err, os.ErrNotExist) {
		return ask.StatusClean, ErrNoGraph
	}
	force := ""
	g, err := store.Read(root)
	switch {
	case errors.Is(err, model.ErrSchemaVersion):
		force = "graph schema is outdated, rebuilding"
	case err != nil:
		force = fmt.Sprintf("graph is unreadable (%v), rebuilding", err)
	case g.Meta.Extractor != golang.Version:
		force = fmt.Sprintf("graph was built by extractor %q, rebuilding", g.Meta.Extractor)
	}
	return ask.Refresh(root, golang.Version,
		func() error { _, _, err := Build(root, notice); return err },
		ask.RefreshOptions{Force: force, Wait: wait, Notice: notice})
}
```

  Vorher prüfen: gibt `store.Read` `model.ErrSchemaVersion` gewrappt zurück (`errors.Is`)? Falls nicht,
  ist der Test in Step 2 rot und `store.Read` wrappt ihn mit `%w` — kleiner Eingriff, im selben Commit.
- [ ] **Step 6: Implementierung (CLI).** In `checkCommand` `case "graph-fresh": return checkGraphFresh(args[1:], stdout, stderr)`:

```go
// checkGraphFresh is the first half of the graph lane: the graph must match
// the tree before blast-audit reads it.
func checkGraphFresh(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check graph-fresh", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; found upwards when empty")
	wait := fs.Duration("wait", 30*time.Second, "how long to wait for another run's rebuild")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux check graph-fresh: %v\n", err)
		return 1
	}
	status, err := query.RefreshGraph(project, *wait,
		func(s string) { fmt.Fprintf(stderr, "loomux check graph-fresh: %s\n", s) })
	if err != nil {
		fmt.Fprintf(stderr, "loomux check graph-fresh: %v\n", err)
		return 1
	}
	if status == ask.StatusRebuilt {
		fmt.Fprintln(stdout, "graph rebuilt")
	} else {
		fmt.Fprintln(stdout, "graph is fresh")
	}
	return 0
}
```

  `projectRoot` ist der Helfer aus `graph.go`; er liegt im selben Paket.
- [ ] **Step 7:** Tests grün, Coverage 100 % in `ask`, `query`, `cli`.
- [ ] **Step 8: Commit** `feat(ask): report how a refresh ended and let a gate wait for the lock`,
  dann `feat(cli): add check graph-fresh` (zwei Commits: der erste enthält `ask` und
  `query.RefreshGraph`).

---

### Task 7: `query.Audit` und `check blast-audit`

**Files:**
- Create: `internal/code/query/audit.go`, `internal/code/query/audit_test.go`, `testdata/cases/graph/audit.golden`
- Modify: `internal/cli/check.go`, `internal/cli/check_test.go`, `internal/code/query/golden_test.go`

**Interfaces:**
- Consumes: `Blast`, `blast.IsTestPath`, `(*blast.Index).InDegreeWhere`
- Produces:

```go
type AuditOptions struct {
	Base            string
	Cached          bool
	Threshold       int
	SkipTestCallers bool
}
type AuditFinding struct {
	Path   string       `json:"path"`
	Signal blast.Signal `json:"signal"`
	Seeds  []blast.Seed `json:"seeds"` // only the seeds at or above the threshold
}
type AuditAnswer struct {
	Range    string         `json:"range"`
	Areas    int            `json:"areas"`
	Findings []AuditFinding `json:"findings"`
}
func Audit(root string, opts AuditOptions) (AuditAnswer, []string, error)
func AuditReport(a AuditAnswer, threshold int) string
```

- [ ] **Step 1: Failing tests**
  - Mini-Repo (Helfer aus Task 4), `Add` geändert und gestagt, `Cached: true, Threshold: 2` ⇒ ein
    Befund `calc/calc.go` (`stale`, Seed `Add` in-degree 2) ⇒ Golden `audit.golden`.
  - `Threshold: 3` ⇒ kein Befund.
  - `Threshold: 2, SkipTestCallers: true` ⇒ kein Befund (nur `main` zählt).
  - `calc/calc_test.go` mitgeändert und gestagt ⇒ Signal `changed` ⇒ kein Befund.
  - `Audit` ruft `Blast` mit `NoRefresh: true` (Test: veralteter Fingerprint, `Audit` baut nicht neu —
    `wiring.json`-mtime unverändert).
  - `Threshold < 1` ⇒ Fehler `threshold must be at least 1`.
  - CLI: `check blast-audit --cached --threshold 2 --root <repo>` ⇒ 1 und Bericht auf stdout;
    `--threshold 3` ⇒ 0 und `no area at or above 3 callers lacks a changed test`; `--base X --cached`
    ⇒ 1 mit `ErrBaseAndCached`; `--threshold x` ⇒ 2.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implementierung**

```go
// Audit is blast-audit: an area is a finding when no changed test reaches it
// (none or stale) and one of its seeds has at least Threshold callers. It never
// rebuilds the graph; that is graph-fresh's job, and two rebuilds would meet
// at the same lock.
func Audit(root string, opts AuditOptions) (AuditAnswer, []string, error) {
	if opts.Threshold < 1 {
		return AuditAnswer{}, nil, errors.New("threshold must be at least 1")
	}
	ans, notes, err := Blast(root, BlastOptions{Base: opts.Base, Cached: opts.Cached, NoRefresh: true})
	if err != nil {
		return AuditAnswer{}, notes, err
	}
	out := AuditAnswer{Range: ans.Range, Areas: len(ans.Areas)}
	var x *blast.Index
	if opts.SkipTestCallers {
		g, _, err := loadGraph(root, true)
		if err != nil {
			return AuditAnswer{}, notes, err
		}
		x = blast.New(g)
	}
	for _, a := range ans.Areas {
		if a.Signal != blast.SignalNone && a.Signal != blast.SignalStale {
			continue
		}
		f := AuditFinding{Path: a.Path, Signal: a.Signal}
		for _, s := range a.Seeds {
			n := s.InDegree
			if x != nil {
				n = x.InDegreeWhere(s.Node.ID, func(m *model.Node) bool { return m != nil && !blast.IsTestPath(m.Path) })
			}
			if n >= opts.Threshold {
				f.Seeds = append(f.Seeds, blast.Seed{Node: s.Node, InDegree: n})
			}
		}
		if len(f.Seeds) > 0 {
			out.Findings = append(out.Findings, f)
		}
	}
	return out, notes, nil
}
```

  Den doppelten Ladevorgang vermeiden: `Blast` bekommt ein unexportiertes Gegenstück
  `blastWith(root, opts) (BlastAnswer, *model.Graph, []string, error)`, das `Blast` und `Audit`
  teilen; `Audit` baut den Index aus dem zurückgegebenen Graphen. (So im Code, der Block oben zeigt
  nur die Logik.)
- `AuditReport`:

```
blast audit: <Range>, threshold <n>
<Path> [<Signal>]: <Name> in-degree <k>, <Name> in-degree <k>
```

  ohne Befund: `no area at or above <n> callers lacks a changed test (<Areas> areas)`.
- CLI `checkBlastAudit` (Flags `--root`, `--base`, `--cached`, `--threshold` mit 3,
  `--skip-test-callers`), Exit 1 bei Befund, `case "blast-audit"` in `checkCommand`.
- [ ] **Step 4:** Golden erzeugen, prüfen, Tests grün, Coverage.
- [ ] **Step 5: Commit** `feat(cli): add check blast-audit`

---

### Task 8: Messung E2′ — Rot-Quote auf der Historie

**Kein Produktcode.** Ergebnis ist eine Zahl für die Entscheidung des Nutzers.

**Files:**
- Create (Scratchpad, nicht im Repo): `e2-replay.sh`
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`

- [ ] **Step 1: Binary bauen** (Task 7 muss drin sein): `go build -o bin/loomux.new.exe ./cmd/loomux`,
  `go run ./cmd/loomux dev swap-binary --dir bin`.
- [ ] **Step 2: Skript** (Git Bash):

```sh
#!/bin/sh
# Replays blast-audit over the last 50 first-parent commits of master: for each
# commit c, the graph of c and the diff c~1...c, for N = 3, 5, 10, with and
# without test callers. Prints one CSV line per commit.
set -u
repo="C:/Users/micro/Documents/#GIT/loomux"
bin="$repo/bin/loomux.exe"
wt="$TMPDIR/loomux-e2"
echo "commit,n3,n5,n10,n3s,n5s,n10s"
for c in $(git -C "$repo" rev-list --first-parent -n 50 master); do
  git -C "$repo" worktree add -q --detach "$wt" "$c" || continue
  "$bin" graph build --root "$wt" >/dev/null 2>&1
  line="$c"
  for skip in "" "--skip-test-callers"; do
    for n in 3 5 10; do
      "$bin" check blast-audit --root "$wt" --base "$c~1" --threshold "$n" $skip >/dev/null 2>&1
      line="$line,$?"
    done
  done
  echo "$line"
  git -C "$repo" worktree remove --force "$wt"
done
```

  Exit 1 = rot, 0 = grün, alles andere als Fehler zählen und nennen (z. B. der Root-Commit).
- [ ] **Step 3: Laufen lassen** (`run_in_background`, dauert Minuten), CSV sichern.
- [ ] **Step 4: Auswerten:** je Spalte Anteil rot von 50; die fünf Commits mit Rot bei N = 3 von Hand
  ansehen (`bin/loomux.exe check blast-audit --root <wt> --base c~1 --threshold 3`) und je Commit einen
  Satz: echter Befund oder Lärm.
- [ ] **Step 5: Eintrag** in beiden `benchmarks.md`: `## 2026-09-23 HH:MM — blast-audit over the
  last 50 commits` mit Methode (Skript im Wortlaut), Tabelle Rot-Quote je N × Zählweise, Laufzeit je
  Commit (Median von `check blast-audit`), die Handdurchsicht.
- [ ] **Step 6: Commit** `docs(bench): measure how often blast-audit would fail on recent history`
- [ ] **Step 7: STOPP.** Dem Nutzer die Tabelle vorlegen und fragen: Schwelle N und
  `--skip-test-callers` ja/nein für das Go-Preset. **Task 9 beginnt erst mit dieser Antwort.** Die
  Antwort geht als Satz in den Nachtrag (§4, E2′ „entschieden am …: N = …, Zählweise …") und in die
  Paritätsliste.

---

### Task 9: Art `graph` in der Prüfkette

**Files:**
- Modify: `internal/verify/schema.go`, `schema_test.go`, `plan.go`, `plan_test.go`, `presets.toml`, `presets_test.go`, `report_test.go`, `internal/cli/check.go`, `check_test.go`, `internal/code/query/refresh.go`, `refresh_test.go`

**Interfaces:**
- Produces:

```go
// verify
type PlanEnv struct {
	Root, Loomux, RunID string
	HasTests            func(root string, patterns []string) bool
	ImportReady         func(dir string) bool
	// GraphReady says whether the graph lanes can mean anything at root, and
	// why not. nil means they cannot.
	GraphReady func(root string) (bool, string)
}

// query
func GraphReady(root string) (bool, string)
```

- [ ] **Step 1: Failing tests (verify)**
  - `Kinds()` == `[lint types test coverage graph]`; `Reserved("graph-fresh")`, `Reserved("blast-audit")`,
    `Reserved("graph")` wahr; `[verify.profiles] graph = ["lint"]` ⇒ Fehler „collides".
  - Profilvorgabe `precommit` == `[lint types test coverage graph]`; `edit` und `stop` unverändert.
  - `Plan` mit Go-Stack in zwei Bereichen (`.` und `tools`) und `Kinds: ["graph"]`, Scope Check,
    `GraphReady` ⇒ `true` ⇒ **ein** Job `graph/go`, `Area "."`, ohne `@`-Suffix.
  - `GraphReady` ⇒ `false, "no graph"` ⇒ Job `Pre == StateNotApplicable`, `Note == "no graph"`;
    `GraphReady == nil` ⇒ `StateNotApplicable`, Note `graph lanes need a graph probe`.
  - Scope Edit mit `Kinds: ["graph"]` ⇒ kein Job.
  - Ein Stack ohne `graph`-Lane (python) ⇒ `not-applicable` `no command`, und `GraphReady` wird dafür
    nicht aufgerufen (Zähler im Fake).
  - `CheckVerdict(["graph"], …)` mit nur `not-applicable` ⇒ Code 0.
  - Preset: `[stack.go.graph]` hat genau die Befehle aus der Entscheidung von Task 8.
- [ ] **Step 2: Failing tests (query)** `GraphReady` im Mini-Repo:
  - kein Graph ⇒ `false`, `no graph at .loomux/state/graph/wiring.json` — und `gitOutput` wurde
    **nicht** gerufen;
  - Repo ohne Commit ⇒ `false`, Text beginnt mit `no HEAD`;
  - `MERGE_HEAD` angelegt (`git rev-parse --git-path MERGE_HEAD` beschreiben) ⇒ `false`,
    `a merge is in progress`;
  - nichts gestagt ⇒ `false`, `nothing staged`;
  - gestagt ⇒ `true, ""`;
  - `gitOutput` für `diff --cached --quiet` mit Exit 128 (Fake: `*exec.ExitError` lässt sich nicht
    bauen — der Fake gibt einen Fehler, der kein Exit 1 ist) ⇒ `false` mit dem Fehlertext.
- [ ] **Step 3: Failing tests (CLI)** `check graph --root <repo>` im Mini-Repo mit gestagter Änderung
  ⇒ Lane `graph/go` läuft (Fake-`checkStart` sieht beide Befehle); ohne Graph ⇒ `not-applicable` und
  Exit 0; Usage-Text nennt `graph` und `graph-fresh, blast-audit`.
- [ ] **Step 4: Implementierung (verify)**

```go
func Kinds() []string { return []string{"lint", "types", "test", "coverage", "graph"} }

func Reserved(name string) bool {
	return slices.Contains([]string{"gofmt", "commit-msg", "gocover", "graph-fresh", "blast-audit", "all"}, name) || slices.Contains(Kinds(), name)
}
```

  Profilvorgaben: `"precommit": {"lint", "types", "test", "coverage", "graph"}`; der Kommentar an
  `stop` wird „The four kinds that check code, without graph: its lane reads the index, which is
  empty at a turn end."

  In `Plan`:

```go
			areas := t.areas
			if kind == "graph" {
				// The graph belongs to the root: one job, not one rebuild per area.
				areas = []string{"."}
			}
			for _, area := range areas {
```

  In `planJob` direkt nach dem Aufbau von `job`:

```go
	if kind == "graph" {
		if req.Scope == ScopeEdit {
			return Job{}, link{}, false, nil
		}
		job.Name = kind + "/" + stack
	}
```

  und nach dem Block `if !r.Defined || len(cmds) == 0 { … }`:

```go
	if kind == "graph" {
		ready, note := false, "graph lanes need a graph probe"
		if env.GraphReady != nil {
			ready, note = env.GraphReady(env.Root)
		}
		if !ready {
			job.Pre, job.Note = StateNotApplicable, note
			return job, link{}, true, nil
		}
	}
```

  Preset (Werte aus Task 8; Vorgabe der Spec):

```toml
# The graph lane is local by nature: it runs only where a graph was built,
# and a plan-time probe makes it not-applicable in CI and with nothing staged.
[stack.go.graph]
commands = ["{loomux} check graph-fresh", "{loomux} check blast-audit --cached --threshold 3"]
```

- [ ] **Step 5: Implementierung (query)**

```go
// GraphReady is the plan-time probe of the graph lane: a graph to read, a HEAD
// to diff against, no merge in progress, and something staged.
func GraphReady(root string) (bool, string) {
	if _, err := os.Stat(store.WiringPath(root)); err != nil {
		return false, "no graph at .loomux/state/graph/wiring.json"
	}
	out, err := gitOutput(root, "rev-parse", "--git-path", "MERGE_HEAD", "HEAD")
	if err != nil {
		return false, "no HEAD to compare with: " + err.Error()
	}
	mergeHead, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	if !filepath.IsAbs(mergeHead) {
		mergeHead = filepath.Join(root, mergeHead)
	}
	if _, err := os.Stat(mergeHead); err == nil {
		return false, "a merge is in progress"
	}
	_, err = gitOutput(root, "diff", "--cached", "--quiet")
	var exit *exec.ExitError
	switch {
	case err == nil:
		return false, "nothing staged"
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		return true, ""
	}
	return false, err.Error()
}
```

  `runGit` wrappt `*exec.ExitError` mit `%w`, daher greift `errors.As`. Für den Fake-Test braucht es
  einen Weg zu Exit 1 ohne echten Prozess: der Test ruft für diesen Arm das echte git im Mini-Repo;
  der Fake deckt nur den Fehler-Arm ab.
- [ ] **Step 6: CLI.** `env := verify.PlanEnv{…, GraphReady: query.GraphReady}` in `checkRun`; die
  Usage-Zeile wird
  `loomux check: name a profile or kinds (lint,types,test,coverage,graph), or one of: commit-msg, gofmt, gocover, graph-fresh, blast-audit`.
  Tests, die die alte Zeile vergleichen, nachziehen.
- [ ] **Step 7:** `sh ci/gate.sh` — **dieser Commit schaltet die Lane im eigenen Pre-Commit ein**. Der
  Gate-Lauf zeigt `graph/go: ok` oder `not-applicable`; ist sie rot, Befund lesen, nicht umgehen.
- [ ] **Step 8: Commit** `feat(verify): add the graph kind with a plan-time probe`

---

### Task 10: Blast-Monitor im Post-Edit-Hook

**Files:**
- Create: `internal/hooks/blast_monitor.go`, `internal/hooks/blast_monitor_test.go`, `internal/hooks/blast_monitor_bench_test.go`
- Modify: `internal/verify/report.go`, `report_test.go`, `internal/hooks/post_edit.go`, `post_edit_test.go`

**Interfaces:**
- Produces:

```go
// verify
func WriteEdit(stdout, stderr io.Writer, outs []Outcome, aside string) int

// hooks
func blastAside(root, rel string, read func(string) ([]byte, error)) string
func relInRoot(root, raw string) (string, bool)
```

- [ ] **Step 1: Failing tests (verify)** `WriteEdit` mit `aside`:
  - keine roten Lanes, nichts übersprungen, `aside != ""` ⇒ genau ein JSON-Objekt, `additionalContext == aside`;
  - übersprungene Lane und `aside` ⇒ ein Objekt, Kontext = die Skip-Zeilen, dann `aside`, getrennt
    durch `\n`;
  - rote Lane und `aside` ⇒ Exit 2, stdout enthält kein `[graph]` (Skip-Meldungen einer anderen Lane
    bleiben wie heute);
  - `aside == ""` ⇒ Verhalten wie heute (bestehende Tests mit `""` aufrufen).
- [ ] **Step 2: Failing tests (hooks)** `blastAside` auf einem Temp-Repo aus
  `testdata/cases/graph/repo` mit `query.Build`:
  - `calc/calc.go`: Rumpf von `Add` geändert ⇒ Text beginnt mit
    `[graph] calc/calc.go: changed Add; callers in other files:` und enthält `main (main.go)` und
    `TestAdd (calc/calc_test.go)`;
  - `Add` entfernt ⇒ `removed Add` vor den Aufrufern;
  - eine Funktion ohne Aufrufer geändert ⇒ `""`;
  - neue Funktion hinzugefügt, sonst nichts ⇒ `""`;
  - nur ein Kommentar außerhalb jedes Symbols ⇒ `""` (kein Dateiknoten-Seed);
  - ein struct geändert (Testdatei um `type T struct{ A int }` ergänzen, bauen, dann Feld ändern) ⇒
    Text enthält die E1-Zeile wörtlich, auch ohne Aufrufer;
  - kein Graph ⇒ `""`; `wiring.json` mit `meta.extractor` `"go/0"` ⇒ `""`; veraltetes Schema ⇒ `""`;
  - Datei nicht im Graphen ⇒ `""`; `read` gibt Fehler ⇒ `""`; Parsefehler (`func (`) ⇒ `""`;
  - zwölf Aufrufer (Fixture mit zwölf Aufrufer-Dateien) ⇒ zehn Zeilen und `… and 2 more`.
- [ ] **Step 3: Failing tests (post_edit)** mit `editEnv`:
  - Edit an `calc/calc.go` im Temp-Repo (Graph gebaut, Rumpf geändert), Lanes grün ⇒ stdout ist ein
    gültiges JSON-Objekt mit dem Blast-Hinweis;
  - dieselbe Lage, Lane rot ⇒ stdout leer, Exit 2;
  - Edit an einer `.py`-Datei ⇒ kein Hinweis;
  - Edit außerhalb der Wurzel ⇒ kein Hinweis, Exit 0.
- [ ] **Step 4: Implementierung (verify)**

```go
// WriteEdit reports a post-edit run: red lanes on stderr, which blocks the
// edit with 2, and as an aside for the model on stdout the lanes it had to
// skip and whatever else the hook has to say. The aside is dropped when a
// lane is red: the finding matters more, and stderr stays the finding's.
func WriteEdit(stdout, stderr io.Writer, outs []Outcome, aside string) int {
	…Schleife wie heute…
	notices := skipped.String()
	if aside != "" && code == 0 {
		notices += aside + "\n"
	}
	writeSkipped(stdout, notices)
	return code
}
```

  Heute schreibt `WriteEdit` die Skip-Meldungen auch bei roter Lane; das bleibt so. Nur der
  Blast-Hinweis entfällt bei Rot (§3.5.4 des G4-Deltas).
- [ ] **Step 5: Implementierung (hooks)**

```go
// Seams a test swaps: reading the graph from disk.
var monitorRead = store.Read

// maxAsideCallers is how many callers the aside names before it counts.
const maxAsideCallers = 10

// typeNote is the aside for a changed type: the Go graph has no edges to
// types, so silence would read as "nothing depends on this".
const typeNote = "[graph] Modified struct/interface/type: type coupling not wired in graph v1 (check references via grep)"

// blastAside tells the model who calls what an edit just changed: the
// symbols of rel whose body hash differs from the graph's, or that the graph
// has and the file no longer does, and their direct callers in other files.
// It never fails and never blocks; whatever it cannot know is silence.
func blastAside(root, rel string, read func(string) ([]byte, error)) string {
	g, err := monitorRead(root)
	if err != nil || g.Meta.Extractor != golang.Version {
		return ""
	}
	src, err := read(rel)
	if err != nil {
		return ""
	}
	now, err := golang.File(rel, string(src))
	if err != nil {
		return ""
	}
	hashes := map[model.NodeID]string{}
	for _, n := range now.Nodes {
		if n.Kind != model.KindFile {
			hashes[n.ID] = n.BodyHash
		}
	}
	var removed, changed []*model.Node
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if n.Path != rel || n.Kind == model.KindFile {
			continue
		}
		h, ok := hashes[n.ID]
		switch {
		case !ok:
			removed = append(removed, n)
		case h != n.BodyHash:
			changed = append(changed, n)
		}
	}
	seeds := append(removed, changed...)
	if len(seeds) == 0 {
		return ""
	}
	ids := make([]model.NodeID, len(seeds))
	typed := false
	for i, s := range seeds {
		ids[i] = s.ID
		typed = typed || s.Kind == "struct" || s.Kind == "interface" || s.Kind == "type"
	}
	var callers []string
	for _, h := range blast.New(g).Reach(ids, blast.In, 1) {
		if h.Node != nil && h.Node.Path != rel {
			callers = append(callers, fmt.Sprintf("  %s (%s)", h.Node.Name, h.Node.Path))
		}
	}
	var b strings.Builder
	if len(callers) > 0 {
		fmt.Fprintf(&b, "[graph] %s: %s; callers in other files:\n", rel, seedList(removed, changed))
		for i, c := range callers {
			if i == maxAsideCallers {
				fmt.Fprintf(&b, "  … and %d more\n", len(callers)-maxAsideCallers)
				break
			}
			b.WriteString(c + "\n")
		}
	}
	if typed {
		b.WriteString(typeNote + "\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// seedList names the removed symbols first: their callers break for sure.
func seedList(removed, changed []*model.Node) string {
	var parts []string
	if len(removed) > 0 {
		parts = append(parts, "removed "+names(removed))
	}
	if len(changed) > 0 {
		parts = append(parts, "changed "+names(changed))
	}
	return strings.Join(parts, ", ")
}

func names(nodes []*model.Node) string {
	out := make([]string, len(nodes))
	for i, n := range nodes {
		out[i] = n.Name
	}
	return strings.Join(out, ", ")
}
```

  Vorher am Code prüfen: welche `Kind`-Strings der Extraktor für Typen vergibt (`extract.go:270-273`
  zeigt `struct`, `interface`; der Rest heißt `type`?) und ob `golang.File` dieselben Ids wie der Build
  vergibt (Build reicht die Bytes an `golang.File`, `query/build.go`). Weicht etwas ab, ist der
  Test in Step 2 rot, und die Konstante folgt dem Code.

  In `RunPostEdit` (Ausschnitt):

```go
	outs := verify.Run(jobs, …)
	aside := ""
	if eff.Extensions[ext] == "go" && !slices.ContainsFunc(outs, func(o verify.Outcome) bool { return verify.Red(o.State, verify.ScopeEdit) }) {
		if rel, ok := relInRoot(root, raw); ok {
			aside = blastAside(root, rel, func(p string) ([]byte, error) {
				return os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
			})
		}
	}
	code := ExitOK
	if verify.WriteEdit(stdout, stderr, outs, aside) != 0 {
		code = ExitDenied
	}
```

  `relInRoot` ist der Anfang von `editJobs` (absolut machen, `filepath.Rel`, `filepath.IsLocal`,
  `ToSlash`); `editJobs` ruft ihn ebenfalls, damit die Regel an einer Stelle steht.
- [ ] **Step 6: Benchmark** `blast_monitor_bench_test.go`:
  - `BenchmarkBlastAside` auf dem Graphen dieses Repos (`../../` als Wurzel; `b.Skip`, wenn kein
    `wiring.json` da ist), Datei `internal/code/blast/reach.go`.
  - `BenchmarkBlastAsideScaled` mit `k = 1, 2, 5, 10, 20`: der Graph dieses Repos `k`-fach kopiert
    (Ids und Pfade mit Präfix `copyN/`), in ein Temp-Verzeichnis geschrieben (`store.Write`), dann
    `blastAside` darauf. Ausgabe je `k`: ns/op und Größe von `wiring.json`.
  - Benchmark-Dateien zählen nicht zur Coverage (sie enthalten keine Produktfunktion).
- [ ] **Step 7:** Tests grün, Coverage 100 % in `hooks` und `verify`; `sh ci/gate.sh`.
- [ ] **Step 8: Commit** `feat(hooks): name the callers an edit may break` (Body: Seeds über den
  Body-Hash, kein Dateiknoten, E1-Hinweis, Schweigen bei allem Unbekannten, Hinweis entfällt bei roter
  Lane). `WriteEdit` im selben Commit — die Signatur ändert sich nur für diesen Zweck.

---

### Task 11: Messung nachher, Mutanten, Parität, Doku

**Files:**
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`, `docs/.superpowers/parity/code-g4.md`, `README.md`, `README.de.md`, `docs/{en,de}/cli-reference.md`, `docs/{en,de}/configuration.md`, `docs/{en,de}/hooks.md`, `docs/{en,de}/migration.md`, `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`, `docs/.superpowers/specs/2026-09-22-loomux-code-g4-delta.md` (nur Kopf: Verweis auf den Nachtrag)

- [ ] **Step 1: Messen.** Binary bauen (Global Constraints), `bin/loomux.exe graph build`, dann
  `bin/loomux.exe dev bench-hooks testdata/bench/g4b-post-edit.json -n 10`. Einmal mit gebautem Graphen
  (Monitor aktiv, Payload-Datei unverändert ⇒ Schweigen), einmal nach einer echten Rumpfänderung an
  `reach.go` (Monitor meldet), danach die Änderung zurücknehmen. Dazu
  `go test ./internal/hooks/ -run '^$' -bench BlastAside -benchtime 20x`, und
  `bin/loomux.exe check graph-fresh` mit und ohne Drift, `bin/loomux.exe check blast-audit --cached`
  mit einer gestagten Änderung (je 10 Läufe, Median; Messmethode wie Task 0).
- [ ] **Step 2: Eintrag.** Der Abschnitt aus Task 0 bekommt „after": Tabelle vorher/nachher kalt und
  warm, die Benchmark-Tabelle je `k` mit Größe von `wiring.json`, und der Satz „the monitor passes
  100 ms at about <Größe> MB of wiring.json (k ≈ …)". Englisch und deutsch.
- [ ] **Step 3: Mutanten.** `go run ./cmd/loomux dev mutants internal/code/blast internal/code/diff internal/code/grep`.
  Überlebende töten (Test ergänzen) oder mit Begründung in die Paritätsliste. Tests als eigener Commit
  `test(blast): kill the mutants that survived` (bzw. `diff`).
- [ ] **Step 4: Paritätsliste** `docs/.superpowers/parity/code-g4.md`: Titel „G4a und G4b", Abschnitt
  `## 4. G4b` mit Tabelle (Komponente | Graft | loomux | Typ | Begründung) mindestens für:
  Dateiknoten nicht expandiert; Dateiknoten nicht im Monitor; Testsignal mit Hülle; `_test.go` als
  einzige Testregel; Testdatei als eigener Bereich ⇒ `na`; inDegree ohne `contains` im Hub-Ranking;
  `--skip-test-callers` und die Entscheidung aus Task 8; Arbeitsbaum-Graph gegen Index-Diff; kein
  Stop-Hook; `graph_blast` als Text; `parseDepth`; Mutanten-Überlebende.
- [ ] **Step 5: Doku.**
  - `cli-reference` (en/de): `graph blast`, `check graph-fresh`, `check blast-audit` mit Flags und
    Exit-Codes; `graph_blast` bei den MCP-Werkzeugen; der Satz „neither are git-diff blast radius
    analysis (Stage G4b)" (en Zeile ~492, de ~498) fällt weg.
  - `configuration` (en/de): Art `graph` in der Liste der Arten, Profilvorgabe `precommit`, das Preset
    und wie man den Schwellenwert ändert (beide Befehle ersetzen), wo die Lane `not-applicable` ist.
  - `hooks` (en/de): Blast-Monitor — wann er spricht, wann er schweigt, Wiederholung bis zum Neubau,
    E1-Hinweis, Vorrang roter Lanes.
  - `migration` (en/de): Zeile G4b ✅ mit Datum, Beschreibung „Art `graph`" statt der zwei
    Lane-Namen; Zeile G5 abhängig von „G4b ✅"; W4 „on G4b ✅ and 4"; „Post-Tool Blast Monitor" und
    „Blast Radius Engine" auf ✅ mit Stufe G4b; „elf Werkzeuge" → zwölf; neue Zeile **G4c** „Stop-Hook
    mit Blast-Logik" (offen, hängt an G4b ✅, Prio 2) samt Fähigkeitszeile.
  - Fusion-Spec: Zeile `G4–G5` in der Stufentabelle teilt sich in `G4a ✅`, `G4b ✅ 2026-09-…`, `G4c`
    offen (Stop-Hook mit Blast-Logik, Form Arbeitsbaum gegen HEAD) und `G5`; Prio-Tabelle entsprechend.
    **Diese Änderung zuerst**, dann `migration.md` (AGENTS.md).
  - `README.md`/`README.de.md`: `graph blast` in der Befehlsübersicht, die Art `graph`, der Monitor
    in einem Satz bei den Hooks, zwölf MCP-Werkzeuge.
  - G4-Delta, Kopf: „Berichtigt und für G4b ergänzt durch `2026-09-23-loomux-code-g4b-delta.md`."
- [ ] **Step 6:** `sh ci/gate.sh` grün.
- [ ] **Step 7: Commits** `docs(bench): record the post-edit hook with the blast monitor`,
  `docs: document the blast radius, the graph kind and the edit monitor` (Parität, Referenz, README,
  Migration, Fusion-Spec).
- [ ] **Step 8: Übergabe an `release-pr`.** Fixup-Commits einfalten, Label `release:minor` (feat,
  kein Bruch — die Signaturänderung von `WriteEdit` ist intern; eine neue Art im Standardprofil ist
  kompatibel, weil sie ohne Graph `not-applicable` ist), `## Changelog` mit `Added` (graph blast,
  graph_blast, check graph-fresh, check blast-audit, graph kind, blast monitor) und `Fixed`
  (graph_trace_calls depth below 1). Push-Befehl dem Nutzer nennen.

---

## Selbstprüfung gegen die Spec

| Spec-Stelle | Task |
|---|---|
| Nachtrag §2 Baustein 0 | 0 |
| §2 Baustein 1, G4 §4 `diff` | 2 |
| §2 Baustein 2, §3 Testsignal, Dateiknoten | 3 |
| §3 CLI, git-Aufruf, `--relative` | 4 |
| §3 MCP `graph_blast`, Privacy, `parseDepth` | 1, 5 |
| §2 Baustein 3a, E2′ | 8 |
| §4 `ask.Refresh`, `graph-fresh`, Sperre 30 s | 6 |
| §4 `blast-audit`, `--skip-test-callers`, kein Neubau | 7 |
| §4 Schema, ein Job je Stack, Edit-Scope, `GraphReady`, Preset | 9 |
| §5 Monitor, `WriteEdit(…, aside)` | 10 |
| §6 Nachweise, Messung, Doku; G4 §7 Mutanten | 11 |
| E4′ Stop-Hook als neue Stufe (Fusion-Spec zuerst) | 11 |
