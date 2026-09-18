# loomux G2a — Implementierungsplan: der Graph entsteht

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux graph build` schreibt einen Wiring-Graphen dieses Repos nach
`.loomux/state/graph/wiring.json`, und `loomux graph check` sagt, ob er noch zum
Code passt.

**Architecture:** Fünf neue Pakete und eine Schemaänderung. `sourceset` zählt
die Dateien auf, die ein Bau ansieht. `extract/golang` parst jede davon mit
`go/parser` und liefert Knoten und **Rohkanten** — Kanten, deren Ziel noch ein
Name ist. `resolve` macht daraus Kanten mit Knoten-IDs und einer Konfidenz und
verwirft, was mehrdeutig bleibt. `store` schreibt sortiert und atomar.
`freshness` hält je Datei einen Abdruck `[größe, mtime, hash]` und sagt ohne
Parsen, ob sich etwas bewegt hat. `model` bekommt Schema 2.

**Tech Stack:** Go (Toolchain 1.27.0 im Einsatz, `go.mod` bleibt bei
`go 1.25.0`), Standardbibliothek allein: `go/parser`, `go/token`, `go/ast`,
`crypto/sha256`, `encoding/json`, `os/exec` für `git ls-files`, `sort`,
`path/filepath`. **Keine neue Abhängigkeit**, `go.mod` und `go.sum` werden nicht
angefasst.

**Spec:** `docs/.superpowers/specs/2026-09-17-loomux-code-g2-design.md`
(berichtigt und verengt `2026-09-14-loomux-code-graph-design.md` und ergänzt
`2026-09-16-loomux-code-g1-delta.md`; bei Widerspruch gilt die G2-Spec, danach
diese Datei).

**Referenz:** `trailhq/Graft` @ `1e352a3` (MIT). Alles, was aus ihr gebraucht
wird, steht als Literal oder als Zahl in diesem Plan. **Wer den Klon nicht hat,
braucht ihn nicht.**

## Global Constraints

- **Arbeitsort:** Worktree `C:\Users\micro\Documents\#GIT\loomux-code-g2`,
  Branch `code-g2`, abgezweigt von `master` `2e06c68`. Die Schreibschranke
  braucht keinen eigenen Registry-Eintrag
  (`docs/.superpowers/parity/schranke-worktrees.md`, freigegeben 2026-09-15);
  ein `go build -o bin/loomux.exe ./cmd/loomux` im Worktree ist einmal nötig,
  damit die Hooks dort ein Binary finden.
- **Modulpfad** `github.com/xidus90/loomux`. Neue Pakete liegen unter
  `internal/code/`.
- **Sprache:** Code, Bezeichner, Kommentare, Fehlermeldungen und
  Commit-Nachrichten englisch. Dieser Plan, die Spec und
  `docs/.superpowers/parity/` deutsch. `docs/en/**` englisch, `docs/de/**`
  deutsch, gleiche Dateinamen.
- **Kein `init()`, keine Paketvariable, die Daten parst.** Geladen wird beim
  ersten Gebrauch.
- **Coverage 100 % je Funktion.** Eine Ausnahme nur mit
  `//coverage:exempt <reason>` direkt über `func`. Das Tor
  (`.githooks/pre-commit`) fährt `gofmt -l cmd internal`, `go vet ./...`,
  `go test ./... -count=1 -covermode=set -coverpkg=github.com/xidus90/loomux/...`, `dev covergate` und
  baut danach das Pilot-Binary.
- **Das Tor weist unstaged Eingaben ab.** Vor jedem Commit alles stagen, was
  die Aufgabe angefasst hat — `*.go`, `go.mod`, `go.sum`, `testdata`,
  `.githooks`. Ein vergessenes `git add` bricht den Commit mit
  „inputs differ from the index".
- **Commits:** der Mensch ist Autor und Committer, kein Modell und kein Agent
  wird genannt. Mehrzeilige Nachrichten über eine Datei und `git commit -F`,
  nie über ein Heredoc. **Vor jedem Commit Zweig und HEAD lesen** — eine fremde
  Sitzung im selben Checkout leert den Index. Niemand außer einem Menschen
  pusht.
- **Determinismus vor Bequemlichkeit.** Keine Map-Iteration, deren Reihenfolge
  in ein Ergebnis einfließt. Sortiert wird in Byte-Ordnung (`sort.Strings`,
  `strings.Compare`), nie über eine Kollation.
- **Jeder Pfad im Graphen ist repowurzel-relativ und geht durch
  `filepath.ToSlash`.** Ein Backslash in einer Knoten-ID ist ein Fehler.
- **Subagenten**, falls eingesetzt: `model: "opus"` und `effort: "low"`, beides
  ausdrücklich gesetzt.

---

## File Structure

| Datei | Verantwortung |
|---|---|
| `internal/code/model/graph.go` (ändern) | `Node.BodyHash`, `Node.BodyText` (`json:"-"`), `Meta.Extractor`, `Span.Lines` |
| `internal/code/model/decode.go` (ändern) | `schemaVersion = 2`; vier neue Regeln in `Validate` |
| `internal/code/model/testdata/wiring.json` (ändern) | auf Schema 2, mit `body_hash` je Knoten und `extractor` in `meta` |
| `internal/code/sourceset/sourceset.go` | `List`, `Stat`, `SourceFile`; Git-Menge, Sperrliste, 1-MB-Grenze |
| `internal/code/extract/golang/extract.go` | `Version`, `File`, `Result`, `RawEdge`; Knoten und Rohkanten einer Datei |
| `internal/code/extract/golang/scope.go` | Gültigkeitsbereich-Stapel: ist ein Selektorname beschattet? |
| `internal/code/extract/golang/ident.go` | `mintID`, `qualify`, `exported`, `receiverType`, `signature` |
| `internal/code/resolve/resolve.go` | `Graph(files []golang.Result, mods []Module) *model.Graph` |
| `internal/code/resolve/gomod.go` | `Modules`, `resolveImport` |
| `internal/code/store/store.go` | `Write`, `Read`, `WiringPath`, `CachePath` |
| `internal/code/freshness/fingerprint.go` | `Print`, `Fingerprint`, `Read`, `Write`, `Probe`, `Drift` |
| `internal/cli/graph.go` | `graphCommand`: `build`, `check` |
| `internal/cli/commands.go` (ändern) | eine Zeile: `"graph": graphCommand` |
| `docs/{en,de}/cli-reference.md`, `architecture.md`, `configuration.md`, `benchmarks.md` (ändern) | die Dokuschuld aus Abschnitt 13 der Spec |
| `README.md`, `README.de.md` (ändern) | der Absatz zum Stand und die Fahrplanzeilen |
| `docs/.superpowers/parity/code-g2a.md` | Mutationsrunde: jeder Überlebende mit Verfügung |

Die Vorlagen des Extraktors liegen unter
`internal/code/extract/golang/testdata/` mit der Endung **`.go.txt`**. Grund:
`go build` und `go vet ./...` ignorieren `testdata/` ohnehin, aber die erste
Bahn des Tors — `gofmt -l cmd internal` — steigt hinein und meldet eine krumme
Datei dort wie jede andere (am 2026-09-17 nachgeprüft). Eine absichtlich
unformatierte Vorlage bräche also das Tor.

---

### Task 0: Arbeitsort prüfen

**Files:** keine.

**Interfaces:**
- Consumes: nichts.
- Produces: nichts.

- [ ] **Schritt 1: Ort und Basis lesen**

```sh
cd "C:/Users/micro/Documents/#GIT/loomux-code-g2"
git rev-parse --abbrev-ref HEAD
git log -1 --format='%h %an <%ae>'
git status --porcelain
```

Erwartet: `code-g2`, HEAD auf dem Spec-Commit, Autor der Mensch, Arbeitsbaum
sauber.

- [ ] **Schritt 2: Binary für die Hooks bauen**

```sh
go build -o bin/loomux.exe ./cmd/loomux
```

- [ ] **Schritt 3: Tor einmal leer fahren**

```sh
go test ./... -count=1
```

Erwartet: alles grün, bevor eine Zeile G2a dazukommt. Ist es das nicht, liegt
der Fehler auf `master` und nicht in dieser Stufe — melden, nicht reparieren.

---

### Task 1: `internal/code/model` — Schema 2

**Files:**
- Modify: `internal/code/model/graph.go`
- Modify: `internal/code/model/decode.go`
- Modify: `internal/code/model/testdata/wiring.json`
- Test: `internal/code/model/decode_test.go`, `internal/code/model/graph_test.go`

**Interfaces:**
- Consumes: das vorhandene `model` aus G1.
- Produces: `Node.BodyHash string`, `Node.BodyText string` (nicht
  serialisiert), `Meta.Extractor string`, `schemaVersion = 2`, und ein
  `Validate`, das doppelte IDs, eine Kantenquelle ohne Knoten, ein leeres
  `body_hash` und ein leeres `meta.extractor` abweist.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

An `internal/code/model/decode_test.go` anhängen:

```go
func TestValidateRejectsDuplicateNodeIDs(t *testing.T) {
	// pagerank.Prepare keeps the first node of an id, blast.New overwrites --
	// two answers to the same question out of one graph.
	g := &model.Graph{
		Meta: model.Meta{Version: 2, Extractor: "go/1"},
		Nodes: []model.Node{
			{ID: "a.go#F", Kind: "function", Path: "a.go", BodyHash: "h1"},
			{ID: "a.go#F", Kind: "function", Path: "a.go", BodyHash: "h2"},
		},
	}
	err := g.Validate()
	if err == nil || !strings.Contains(err.Error(), "a.go#F") {
		t.Fatalf("got %v, want an error naming the duplicated id", err)
	}
}

func TestValidateRejectsAnEdgeSourceThatIsNoNode(t *testing.T) {
	// A loose end is foreseen on the target side of an import only. An invented
	// source shows up in blast as a hit without a node and is then
	// indistinguishable from a genuinely unresolved import.
	g := &model.Graph{
		Meta:  model.Meta{Version: 2, Extractor: "go/1"},
		Nodes: []model.Node{{ID: "a.go", Kind: model.KindFile, Path: "a.go", BodyHash: "h"}},
		Edges: []model.Edge{{
			Source: "a.go#Ghost", Target: "a.go", Relation: model.RelationCalls,
			Confidence: model.ConfidenceExtracted,
		}},
	}
	err := g.Validate()
	if err == nil || !strings.Contains(err.Error(), "a.go#Ghost") {
		t.Fatalf("got %v, want an error naming the invented source", err)
	}
}

func TestValidateRejectsAnEmptyBodyHash(t *testing.T) {
	// Without it `graph check` cannot judge the node and would have to call it
	// fresh in silence.
	g := &model.Graph{
		Meta:  model.Meta{Version: 2, Extractor: "go/1"},
		Nodes: []model.Node{{ID: "a.go", Kind: model.KindFile, Path: "a.go"}},
	}
	err := g.Validate()
	if err == nil || !strings.Contains(err.Error(), "body hash") {
		t.Fatalf("got %v, want an error about the missing body hash", err)
	}
}

func TestValidateRejectsAnEmptyExtractorStamp(t *testing.T) {
	// `check` compares the stamp before it diffs nodes: a graph from another
	// extractor is not stale, it is foreign.
	g := &model.Graph{
		Meta:  model.Meta{Version: 2},
		Nodes: []model.Node{{ID: "a.go", Kind: model.KindFile, Path: "a.go", BodyHash: "h"}},
	}
	err := g.Validate()
	if err == nil || !strings.Contains(err.Error(), "extractor") {
		t.Fatalf("got %v, want an error about the missing extractor stamp", err)
	}
}

func TestDecodeRefusesSchemaOne(t *testing.T) {
	// A writer that bumped the version changed something; failing loudly beats
	// ranking a stale shape.
	_, err := model.Decode(strings.NewReader(`{"meta":{"version":1},"nodes":[],"edges":[]}`))
	if err == nil || !strings.Contains(err.Error(), "version 1") {
		t.Fatalf("got %v, want a refusal naming version 1", err)
	}
}

func TestBodyTextNeverReachesTheWire(t *testing.T) {
	// It is ~65% of wiring.json's bytes and lives tokenized in the ask sidecar
	// instead. The json:"-" tag is how that decision is enforced.
	out, err := json.Marshal(model.Node{ID: "a.go#F", BodyText: "secret body"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "secret body") {
		t.Fatalf("body text reached the wire: %s", out)
	}
}
```

Der Import-Block von `decode_test.go` braucht `encoding/json` und `strings`,
falls noch nicht vorhanden.

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/model/ -run 'TestValidateRejects|TestDecodeRefusesSchemaOne|TestBodyTextNever' -v
```

Erwartet: Übersetzungsfehler, `unknown field BodyHash in struct literal` und
`unknown field Extractor`.

- [ ] **Schritt 3: Die Typen erweitern**

In `internal/code/model/graph.go`, `Node` um zwei Felder ergänzen:

```go
	// BodyHash is sha256 (full hex) over the text of the whole declaration. It
	// is what `graph check` diffs on: a body that changed changes the hash,
	// and a doc comment that changed does not -- go/ast's Pos()..End() leaves
	// the comment out, matching tree-sitter, where the comment is a sibling.
	BodyHash string `json:"body_hash"`

	// BodyText is the searchable body: whitespace collapsed, capped. It never
	// reaches disk -- the ask sidecar holds it tokenized, and in wiring.json it
	// would be ~65% of the bytes. The tag is the enforcement, not a
	// convenience.
	BodyText string `json:"-"`
```

und `Meta` um eines:

```go
	// Extractor identifies the extractor that produced this graph. `check`
	// compares it before diffing a single node: a graph from another extractor
	// is not stale, it is foreign, and its nodes say nothing about this code.
	Extractor string `json:"extractor"`
```

- [ ] **Schritt 4: Version und Prüfung nachziehen**

In `internal/code/model/decode.go` die Konstante auf 2 setzen und `Validate`
erweitern. Die vorhandenen Schleifen bleiben; dazu kommen der Mengen-Aufbau und
die Quellprüfung:

```go
const schemaVersion = 2
```

In `Validate`, nach der Versionsprüfung:

```go
	if g.Meta.Extractor == "" {
		return fmt.Errorf("graph has no extractor stamp")
	}
	seen := make(map[NodeID]struct{}, len(g.Nodes))
```

In der Knotenschleife, als weitere Fälle des `switch`:

```go
		case n.BodyHash == "":
			return fmt.Errorf("node %q has no body hash", n.ID)
```

und direkt nach dem `switch`, noch in der Schleife:

```go
		if _, dup := seen[n.ID]; dup {
			return fmt.Errorf("node %q appears twice; the two computers would read it differently", n.ID)
		}
		seen[n.ID] = struct{}{}
```

In der Kantenschleife, als weiterer Fall:

```go
		case !isNode(seen, e.Source):
			return fmt.Errorf("edge %d has source %q, which is no node of this graph", i, e.Source)
```

Dazu der Helfer, damit die Absicht am Namen steht:

```go
// isNode reports whether id belongs to a node of the graph.
//
// Only edge sources are held to this. A target may be a loose end: an
// unresolved import names its module ("fmt"), and that is a fact about the
// code, not a defect.
func isNode(nodes map[NodeID]struct{}, id NodeID) bool {
	_, ok := nodes[id]
	return ok
}
```

- [ ] **Schritt 5: `Span.Lines` anlegen**

Die Form `L12-L40` ist Schemagut und hat zwei Leser: der Extraktor markiert
damit die Zeilen eines Symbols, die Abfrage schneidet damit den Quelltext. Sie
gehört deshalb als Methode an den Typ und nicht als Helfer in beide Pakete.

Korrigiert in Task 11: Die ursprüngliche Fassung dieses Schnipsels endete auf
`return from, to, okA && okB`, was `0, 4, false` für `"Lx-L4"` und `12, 0,
false` für `"L12-L"` geliefert hätte — im Widerspruch zum Test direkt darunter,
der für beide Zeilen `0, 0, false` verlangt. Der tatsächlich implementierte
Code zeigt beide Hälften auf 0, wenn eine von beiden nicht parst; die Ruling
dazu steht im Ledger unter Task 1.

```go
// Lines reads "L12-L40" back into 12 and 40, reporting whether the span had
// that shape at all.
func (s Span) Lines() (int, int, bool) {
	str := string(s)
	dash := strings.Index(str, "-L")
	if !strings.HasPrefix(str, "L") || dash < 0 {
		return 0, 0, false
	}
	from, okA := atoi(str[1:dash])
	to, okB := atoi(str[dash+2:])
	if !okA || !okB {
		return 0, 0, false
	}
	return from, to, true
}

// atoi is strconv.Atoi reporting failure as a bool, because a malformed span is
// not an error any caller would handle differently from an unusable one.
func atoi(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}
```

Der Test, in `internal/code/model/graph_test.go`:

```go
func TestSpanLines(t *testing.T) {
	cases := []struct {
		span     model.Span
		from, to int
		ok       bool
	}{
		{"L12-L40", 12, 40, true},
		{"L1-L1", 1, 1, true},
		{"", 0, 0, false},
		{"12-40", 0, 0, false},
		{"L12", 0, 0, false},
		{"Lx-L4", 0, 0, false},
		{"L12-L", 0, 0, false},
	}
	for _, c := range cases {
		from, to, ok := c.span.Lines()
		if from != c.from || to != c.to || ok != c.ok {
			t.Errorf("Span(%q).Lines() = %d, %d, %v; want %d, %d, %v",
				c.span, from, to, ok, c.from, c.to, c.ok)
		}
	}
}
```

- [ ] **Schritt 6: Das Testdatum auf Schema 2 heben**

`internal/code/model/testdata/wiring.json`: `"version": 2` und
`"extractor": "test/1"` in `meta`, je Knoten ein `"body_hash"` (irgendein
Hex-Literal, es wird nicht nachgerechnet), und beim Dateiknoten
`"exported": true` statt `false` — so schreibt der Extraktor ihn. Prüfen, ob ein
vorhandener Test die Knotenzahl oder ein Feld dieses Datums festnagelt, und ihn
mitziehen:

```sh
grep -rn "testdata/wiring.json\|nodeCount" internal/code/model/
```

- [ ] **Schritt 7: Lauf, der grün sein muss**

```sh
go test ./internal/code/... -count=1
```

Erwartet: PASS, auch `pagerank` und `blast` — sie lesen `Node` und bleiben
unberührt.

- [ ] **Schritt 8: Commit**

```sh
git add internal/code/model
git commit -F ../msg.txt
```

Nachricht (`../msg.txt`, danach löschen):

```
Raise the code graph to schema 2

A node carries body_hash, the hash of its whole declaration, because
`graph check` diffs on it rather than on a file timestamp. It carries
body_text too, but never on the wire: it is most of wiring.json's bytes
and the ask sidecar holds it tokenized.

Meta carries the extractor that wrote the graph. Without it a changed
extractor reports fresh against nodes it never built.

Validate closes the two gaps G1 left -- a duplicated node id, an edge
source that is no node -- now that an extractor is about to write
graphs no test wrote.
```

---

### Task 2: `internal/code/sourceset` — die Dateimenge

**Files:**
- Create: `internal/code/sourceset/sourceset.go`
- Test: `internal/code/sourceset/sourceset_test.go`

**Interfaces:**
- Consumes: nichts aus G2a.
- Produces:

```go
type SourceFile struct {
	Abs   string // absolute path
	Rel   string // repo-relative, forward slashes
	Size  int64
	MTime int64 // UnixNano
}

func List(root string) ([]string, error)          // Rel paths, sorted
func Stat(root string) ([]SourceFile, error)      // List plus size and mtime
```

Beide zählen dieselbe Menge auf. `freshness` und `build` rufen `Stat`, `check`
ruft `List`.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package sourceset_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/code/sourceset"
)

// tree writes files into a fresh directory. The map's keys are slash paths
// relative to the root.
func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestListTakesGoFilesAndNothingElse(t *testing.T) {
	root := tree(t, map[string]string{
		"main.go":         "package main\n",
		"pkg/helper.go":   "package pkg\n",
		"pkg/helper_test.go": "package pkg\n",
		"README.md":       "# no\n",
		"web/app.ts":      "export {}\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	// _test.go stays in: Graft does not exclude tests, it de-ranks them at
	// query time, and "where are the tests" is a fair question of the graph.
	want := []string{"main.go", "pkg/helper.go", "pkg/helper_test.go"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestListSkipsDependencyAndBuildDirectories(t *testing.T) {
	root := tree(t, map[string]string{
		"keep.go":              "package a\n",
		"vendor/dep/dep.go":    "package dep\n",
		"node_modules/x/x.go":  "package x\n",
		"dist/out.go":          "package out\n",
		"_build/gen.go":        "package gen\n",
		".hidden/secret.go":    "package secret\n",
		"internal/bin/keep.go": "package keep\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	// A dot directory is skipped wholesale; the skip list matches a single path
	// segment, so a directory merely NAMED like an output dir deeper in the
	// tree is not exempt -- but "internal/bin" is not on the list at all.
	want := map[string]bool{"keep.go": true, "internal/bin/keep.go": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want the two keepers", got)
	}
	for _, rel := range got {
		if !want[rel] {
			t.Errorf("unexpected file %q", rel)
		}
	}
}

func TestListDropsAFileOverTheSizeLimit(t *testing.T) {
	big := "package big\n" + string(make([]byte, 1_000_001))
	root := tree(t, map[string]string{"big.go": big, "small.go": "package small\n"})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	// Above a megabyte a file is generated or vendored in practice, not written
	// by hand.
	if len(got) != 1 || got[0] != "small.go" {
		t.Fatalf("got %v, want only small.go", got)
	}
}

func TestStatCarriesSizeAndMTimeAndSlashPaths(t *testing.T) {
	root := tree(t, map[string]string{"pkg/a.go": "package pkg\n"})

	got, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d files, want 1", len(got))
	}
	f := got[0]
	if f.Rel != "pkg/a.go" {
		t.Errorf("Rel = %q, want forward slashes and no leading dot", f.Rel)
	}
	if f.Size != int64(len("package pkg\n")) {
		t.Errorf("Size = %d, want %d", f.Size, len("package pkg\n"))
	}
	// mtime is nanoseconds as an int64 -- never a float, because equality of
	// this field decides whether a rebuild happens.
	if f.MTime == 0 {
		t.Error("MTime = 0, want the file's modification time")
	}
	if !filepath.IsAbs(f.Abs) {
		t.Errorf("Abs = %q, want an absolute path", f.Abs)
	}
}

func TestListSkipsTheOutputDirectory(t *testing.T) {
	root := tree(t, map[string]string{
		"a.go": "package a\n",
		".loomux/state/graph/leftover.go": "package leftover\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "a.go" {
		t.Fatalf("got %v, want only a.go -- the graph must not index itself", got)
	}
}

func TestListRefusesAMissingRoot(t *testing.T) {
	_, err := sourceset.List(filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("got nil, want an error for a root that is not there")
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/sourceset/ -v
```

Erwartet: Übersetzungsfehler, `no required module provides package .../sourceset`.

- [ ] **Schritt 3: Die Implementierung schreiben**

```go
// Package sourceset is the file set a graph build looks at, and the stat
// metadata a freshness probe needs.
//
// It is a package of its own and not a part of the extractor, for the reason
// Graft gives (src/graph/source-files.ts): the probe has to enumerate exactly
// the same files as the build, and importing the builder to learn them would
// tie the hook path to the parser. Two enumerations that can drift are two
// answers to "did anything change".
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/source-files.ts and
// src/ingest/fs.ts.
package sourceset

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// maxFileBytes is where a hand-written file stops. Above it a file is
// generated or vendored in practice; one repository indexed 4,484 files of
// vendored code against 121 written by hand, and queries slowed enough that a
// hook's budget could no longer hold them.
const maxFileBytes = 1_000_000

// skipDirs are dependency and build output, never source. The comparison is
// against a single path segment.
var skipDirs = map[string]bool{
	"node_modules": true,
	"dist":         true,
	"build":        true,
	"_build":       true,
	"out":          true,
	"target":       true,
	"vendor":       true,
	"coverage":     true,
	"__pycache__":  true,
	"venv":         true,
}

// SourceFile is one file of the set, with what a freshness probe compares.
type SourceFile struct {
	Abs   string
	Rel   string
	Size  int64
	MTime int64
}

// List returns the repo-relative paths of the Go files a build looks at,
// sorted in byte order.
func List(root string) ([]string, error) {
	files, err := Stat(root)
	if err != nil {
		return nil, err
	}
	rels := make([]string, 0, len(files))
	for _, f := range files {
		rels = append(rels, f.Rel)
	}
	return rels, nil
}

// Stat returns the file set with each file's size and modification time.
//
// A file that vanishes between the walk and the stat is dropped rather than
// reported: the next probe will see the same thing and call it removed.
func Stat(root string) ([]SourceFile, error) {
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}
	var out []SourceFile
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxFileBytes {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		out = append(out, SourceFile{
			Abs:   path,
			Rel:   filepath.ToSlash(rel),
			Size:  info.Size(),
			MTime: info.ModTime().UnixNano(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, nil
}

// skipDir reports whether a directory of this name is walked.
//
// Every dot directory is skipped wholesale -- .git, .github, .loomux and the
// state the graph itself writes into it. That last one matters: a graph that
// indexed its own output would grow on every build.
func skipDir(name string) bool {
	return strings.HasPrefix(name, ".") || skipDirs[name]
}
```

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/code/sourceset/ -count=1 -cover
```

Erwartet: PASS. Bleibt eine Funktion unter 100 %, fehlt ein Fall im Test — den
Test ergänzen, nicht die Funktion ausnehmen.

- [ ] **Schritt 5: Commit**

```sh
git add internal/code/sourceset
git commit -F ../msg.txt
```

Nachricht:

```
Enumerate the file set a graph build looks at

Its own package, not a part of the extractor: the freshness probe must
enumerate exactly the same files, and importing the parser to learn them
would tie the hook path to it. Two enumerations that can drift are two
answers to "did anything change".

Test files stay in the set. Graft does not exclude them either; it
de-ranks them when answering, and "where are the tests for X" is a fair
question to put to the graph.
```

---

### Task 3: `internal/code/extract/golang` — Knoten

**Files:**
- Create: `internal/code/extract/golang/extract.go`
- Create: `internal/code/extract/golang/ident.go`
- Create: `internal/code/extract/golang/testdata/shapes.go.txt`
- Test: `internal/code/extract/golang/extract_test.go`

**Interfaces:**
- Consumes: `model.Node`, `model.KindFile` aus Task 1.
- Produces:

```go
// Version is the extractor's identity. Bump it by hand whenever the nodes or
// edges this package produces change shape or meaning.
const Version = "go/1"

type RawEdge struct {
	Source    model.NodeID
	Relation  model.Relation
	TargetID  model.NodeID // set for contains: already resolved
	Name      string       // a bare call target, or the selected name
	Owner     string       // a receiver type, for a member call on a known local
	Receiver  string       // a selector's receiver identifier, unresolved
	Specifier string       // an import path, for an imports edge
	File      string
}

type Import struct {
	Alias string // as written, or "" when the import has no alias
	Path  string
}

type Result struct {
	Path    string
	Package string   // the file's package clause
	Imports []Import // in source order
	Nodes   []model.Node
	Edges   []RawEdge
}

func File(rel, source string) (Result, error)
```

`File` bekommt den Quelltext als Zeichenkette, nicht einen Pfad: der Aufrufer
hat ihn schon gelesen, um zu hashen, und zweimal lesen wäre zweimal I/O.

- [ ] **Schritt 1: Die Vorlage schreiben**

`internal/code/extract/golang/testdata/shapes.go.txt`:

```go
// Package shapes carries one of every definition the extractor tells apart.
package shapes

import "fmt"

// Doc comments must not reach the body hash: go/ast's Pos() starts at `func`.
func Exported() {}

func unexported() {}

type User struct {
	Name string
}

type Reader interface {
	Read() string
}

type ID int

type (
	First  int
	Second string
)

// Save is a method on a pointer receiver; the id qualifies by the receiver.
func (u *User) Save() error { return nil }

// name is a method on a value receiver of the same type.
func (u User) name() string { return u.Name }

func useFmt() { fmt.Println("x") }
```

- [ ] **Schritt 2: Den fehlschlagenden Test schreiben**

```go
package golang_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/model"
)

// fixture reads a .go.txt template. The templates do not end in .go because
// the gate's first lane, `gofmt -l cmd internal`, walks into testdata and
// would report a deliberately crooked one like any other file.
func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name+".go.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func nodeByID(r golang.Result, id model.NodeID) *model.Node {
	for i := range r.Nodes {
		if r.Nodes[i].ID == id {
			return &r.Nodes[i]
		}
	}
	return nil
}

func TestFileEmitsOneNodePerDefinitionShape(t *testing.T) {
	r, err := golang.File("pkg/shapes.go", fixture(t, "shapes"))
	if err != nil {
		t.Fatal(err)
	}

	want := map[model.NodeID]model.Kind{
		"pkg/shapes.go":             model.KindFile,
		"pkg/shapes.go#Exported":    "function",
		"pkg/shapes.go#unexported":  "function",
		"pkg/shapes.go#useFmt":      "function",
		"pkg/shapes.go#User":        "struct",
		"pkg/shapes.go#Reader":      "interface",
		"pkg/shapes.go#ID":          "type",
		"pkg/shapes.go#First":       "type",
		"pkg/shapes.go#Second":      "type",
		"pkg/shapes.go#User.Save":   "method",
		"pkg/shapes.go#User.name":   "method",
	}
	if len(r.Nodes) != len(want) {
		var got []model.NodeID
		for _, n := range r.Nodes {
			got = append(got, n.ID)
		}
		t.Fatalf("got %d nodes %v, want %d", len(r.Nodes), got, len(want))
	}
	for id, kind := range want {
		n := nodeByID(r, id)
		if n == nil {
			t.Errorf("node %q missing", id)
			continue
		}
		if n.Kind != kind {
			t.Errorf("node %q has kind %q, want %q", id, n.Kind, kind)
		}
	}
	// A grouped `type ( ... )` yields one node per spec, never one for the
	// declaration: otherwise a change to one type would change the hash of
	// every other in the group and `check` would report them all.
	if nodeByID(r, "pkg/shapes.go#First") == nil || nodeByID(r, "pkg/shapes.go#Second") == nil {
		t.Error("a grouped type declaration must yield one node per name")
	}
	// No constants, no variables. Graft emits none for Go, and a const block
	// reaches a query through the file node's residual text instead.
	for _, n := range r.Nodes {
		if n.Kind == "const" || n.Kind == "var" {
			t.Errorf("node %q: this extractor emits no %s nodes", n.ID, n.Kind)
		}
	}
}

func TestFileQualifiesAMethodByItsReceiverAndKeepsTheBareName(t *testing.T) {
	r, err := golang.File("pkg/shapes.go", fixture(t, "shapes"))
	if err != nil {
		t.Fatal(err)
	}

	save := nodeByID(r, "pkg/shapes.go#User.Save")
	if save == nil {
		t.Fatal("method node missing")
	}
	// The id qualifies, because Go methods do not nest and the id would collide
	// per receiver. The name stays bare so a call `u.Save()` can hit it.
	if save.Name != "Save" {
		t.Errorf("Name = %q, want the bare name", save.Name)
	}
	// The pointer is unwrapped: `func (u *User)` owns `User`.
	if save.Owner != "User" {
		t.Errorf("Owner = %q, want User", save.Owner)
	}
	if !save.Exported {
		t.Error("Save must be exported: its own name starts uppercase")
	}
	lower := nodeByID(r, "pkg/shapes.go#User.name")
	if lower == nil || lower.Exported {
		t.Error("name must be unexported: the own name is the part after the dot")
	}
}

func TestFileCutsASignatureAtTheHeader(t *testing.T) {
	r, err := golang.File("pkg/shapes.go", fixture(t, "shapes"))
	if err != nil {
		t.Fatal(err)
	}

	cases := map[model.NodeID]string{
		"pkg/shapes.go#Exported":  "func Exported()",
		"pkg/shapes.go#User.Save": "func (u *User) Save() error",
		// Graft's own code would cut a struct down to the bare name here: its
		// type_spec starts at the name and the header ends at the `struct`
		// keyword. Its comment says "where the body opens", which is what this
		// port writes instead -- see 5.2.1 of the spec.
		"pkg/shapes.go#User":   "type User struct",
		"pkg/shapes.go#Reader": "type Reader interface",
		"pkg/shapes.go#ID":     "type ID int",
		"pkg/shapes.go#First":  "type First int",
	}
	for id, want := range cases {
		n := nodeByID(r, id)
		if n == nil {
			t.Errorf("node %q missing", id)
			continue
		}
		if n.Signature != want {
			t.Errorf("node %q signature = %q, want %q", id, n.Signature, want)
		}
	}
}

func TestFileLeavesTheDocCommentOutOfTheBodyHash(t *testing.T) {
	const withDoc = "package p\n\n// Doc says something.\nfunc F() { return }\n"
	const without = "package p\n\nfunc F() { return }\n"

	a, err := golang.File("p.go", withDoc)
	if err != nil {
		t.Fatal(err)
	}
	b, err := golang.File("p.go", without)
	if err != nil {
		t.Fatal(err)
	}
	// go/ast's Pos() starts at `func`, so the comment is outside the range --
	// the same as tree-sitter, where a comment is a sibling of the
	// declaration. Rewording a doc comment must not make `check` report drift.
	if nodeByID(a, "p.go#F").BodyHash != nodeByID(b, "p.go#F").BodyHash {
		t.Error("a doc comment must not change the body hash")
	}
	// The file node hashes the whole file, so there it must differ.
	if nodeByID(a, "p.go").BodyHash == nodeByID(b, "p.go").BodyHash {
		t.Error("the file node hashes the whole file, comment included")
	}
}

func TestFileNodeCarriesPathAsItsID(t *testing.T) {
	r, err := golang.File("pkg/shapes.go", fixture(t, "shapes"))
	if err != nil {
		t.Fatal(err)
	}

	f := nodeByID(r, "pkg/shapes.go")
	if f == nil {
		t.Fatal("file node missing")
	}
	// The id is the path itself, without a '#'. Validate's rule that an edge
	// source must be a node depends on it: an import edge's source is the file.
	if f.Name != "shapes.go" || f.Kind != model.KindFile {
		t.Errorf("file node = %+v, want name shapes.go and kind file", f)
	}
	if f.Signature != "" || !f.Exported {
		t.Errorf("file node signature = %q, exported = %v; want empty and true", f.Signature, f.Exported)
	}
	if !strings.HasPrefix(string(f.Span), "L1-L") {
		t.Errorf("file span = %q, want L1-L<lines>", f.Span)
	}
}

func TestFileMintsAnOrdinalForACollidingID(t *testing.T) {
	// Two definitions of one name in one file: invalid Go, but the extractor
	// must not lose one. It parses far enough for the walk.
	const src = "package p\n\nfunc F() {}\n\nfunc F() {}\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if nodeByID(r, "p.go#F") == nil || nodeByID(r, "p.go#F~2") == nil {
		var got []model.NodeID
		for _, n := range r.Nodes {
			got = append(got, n.ID)
		}
		t.Fatalf("got %v, want p.go#F and p.go#F~2", got)
	}
}

func TestFileMintsPastASourceNameEndingInAnOrdinal(t *testing.T) {
	// The loop, not a single `~2` guess, is what makes this collision-proof: a
	// source name may itself end in ~N-looking text after qualification.
	const src = "package p\n\nfunc F() {}\n\nfunc F() {}\n\nfunc F() {}\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if nodeByID(r, "p.go#F~3") == nil {
		t.Error("a third collision must mint ~3")
	}
}

func TestFileRefusesSourceItCannotParse(t *testing.T) {
	_, err := golang.File("p.go", "package ???\n")
	if err == nil {
		t.Fatal("got nil, want a parse error")
	}
}

func TestFileTrimsBodyTextAndCapsIt(t *testing.T) {
	long := "package p\n\nfunc F() {\n" + strings.Repeat("\tprintln(\"x\")\n", 2000) + "}\n"
	r, err := golang.File("p.go", long)
	if err != nil {
		t.Fatal(err)
	}
	n := nodeByID(r, "p.go#F")
	if len(n.BodyText) > 5000 {
		t.Errorf("body text is %d chars, want at most 5000", len(n.BodyText))
	}
	if strings.Contains(n.BodyText, "\n") || strings.Contains(n.BodyText, "\t") {
		t.Error("body text must be whitespace-collapsed")
	}
}

func TestFileNodeBodyTextIsTheResidualOutsideEverySymbol(t *testing.T) {
	const src = "package p\n\nimport \"strings\"\n\nconst marker = \"needle\"\n\nfunc F() { _ = strings.TrimSpace(\"inside\") }\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	f := nodeByID(r, "p.go")
	// The residual is what no symbol span covers -- the import header, package
	// constants -- so a file is findable by a word that lives in no function,
	// without storing any symbol body twice.
	if !strings.Contains(f.BodyText, "needle") {
		t.Errorf("file residual = %q, want the package constant in it", f.BodyText)
	}
	if strings.Contains(f.BodyText, "inside") {
		t.Errorf("file residual = %q, must not repeat a symbol body", f.BodyText)
	}
}
```

- [ ] **Schritt 3: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/extract/golang/ -v
```

Erwartet: Übersetzungsfehler, das Paket gibt es nicht.

- [ ] **Schritt 4: Die Hilfsfunktionen schreiben**

`internal/code/extract/golang/ident.go`:

```go
package golang

import (
	"go/ast"
	"strings"
	"unicode"
)

// maxBodyChars caps the searchable body. Graft's figure; a definition longer
// than this is findable by its first 5000 characters or not at all.
const maxBodyChars = 5000

// mintID returns base, or base with the lowest free ordinal appended.
//
// A loop and not a single "~2" guess: a qualified source name may itself end
// in "~2", and only the loop is tight against that.
func mintID(base string, minted map[string]bool) string {
	id := base
	for k := 2; minted[id]; k++ {
		id = base + "~" + itoa(k)
	}
	minted[id] = true
	return id
}

// itoa is strconv.Itoa without the import, so this file stays free of
// anything but the AST.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// exported reports Go visibility: the first letter of a symbol's OWN name is
// uppercase. For a receiver-qualified name the own name is the part after the
// last dot.
func exported(name string) bool {
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}

// receiverType is the base type name of a method's receiver, with a pointer
// unwrapped: `func (u *User)` owns "User". Empty when it cannot be read.
func receiverType(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	expr := recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	// A generic receiver: `func (c *Cache[K]) Get()` -- the name is the index
	// expression's own, not its type argument.
	switch t := expr.(type) {
	case *ast.IndexExpr:
		expr = t.X
	case *ast.IndexListExpr:
		expr = t.X
	}
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// receiverVar is the name a method binds its receiver to -- the `u` in
// `func (u *User) Save()`. Empty for an unnamed receiver.
func receiverVar(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 || len(recv.List[0].Names) == 0 {
		return ""
	}
	return recv.List[0].Names[0].Name
}

// collapse turns a definition's text into one searchable line: every run of
// whitespace becomes a single space, and the result is capped.
func collapse(text string) string {
	out := strings.Join(strings.Fields(text), " ")
	if len(out) > maxBodyChars {
		return out[:maxBodyChars]
	}
	return out
}
```

- [ ] **Schritt 5: Den Extraktor schreiben**

`internal/code/extract/golang/extract.go`:

```go
// Package golang extracts the nodes and raw edges of one Go file.
//
// Raw, because a call's target is a name here and not yet a node: resolving it
// needs every file of the repository, and that is internal/code/resolve's job.
// The split is Graft's (src/graph/extract.ts against src/graph/resolve.ts) and
// it is what keeps this package free of any knowledge about the repository.
//
// go/parser alone, no go/types: cross-package resolution belongs to the
// optional --lsp stage against gopls. See section 3.5 of the G2 spec.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/extract.ts (describeGo).
package golang

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strings"

	"github.com/xidus90/loomux/internal/code/model"
)

// Version is this extractor's identity. Bump it by hand whenever the nodes or
// edges below change shape or meaning.
//
// It reaches the graph's meta and the freshness record, and `check` compares it
// before diffing a single node. Without it a changed extractor matches the tree
// byte for byte, reports fresh, and keeps answering from nodes the old
// extractor built. cli.Version cannot serve: every development build says
// 0.0.0-dev.
const Version = "go/1"

// RawEdge is an edge whose target is not resolved yet.
//
// Three shapes of call reach the resolver, and the difference is exactly what
// the resolver is allowed to assume:
//
//   - Name alone      -- a bare call; the same file first, then a unique match
//   - Name and Owner  -- a member call on a local whose type is known
//   - Name and Receiver -- a selector whose receiver is declared nowhere in the
//     file; the resolver decides whether that receiver names a package, because
//     only it can see the target's package clause
type RawEdge struct {
	Source    model.NodeID
	Relation  model.Relation
	TargetID  model.NodeID
	Name      string
	Owner     string
	Receiver  string
	Specifier string
	File      string
}

// Import is one import of a file, with the alias exactly as written.
//
// The alias is kept raw and not resolved to a name here, because resolving it
// needs the TARGET package's clause -- `import "gopkg.in/yaml.v3"` binds `yaml`
// and not `v3` -- and this package parses one file at a time. Guessing the last
// path segment here would put the guess where nothing can correct it.
type Import struct {
	Alias string `json:"alias,omitempty"`
	Path  string `json:"path"`
}

// Result is what one file contributes to the graph.
type Result struct {
	Path    string
	Package string
	Imports []Import
	Nodes   []model.Node
	Edges   []RawEdge
}

// File extracts one Go file. rel is its repo-relative, slash-separated path;
// source is its contents, which the caller has already read in order to hash
// it.
func File(rel, source string) (Result, error) {
	fset := token.NewFileSet()
	// SkipObjectResolution: ast.Object and File.Unresolved are deprecated as of
	// Go 1.22, and this package tracks scopes itself (scope.go). Skipping also
	// costs less -- 44-46ms against 58-59ms over this repository.
	file, err := parser.ParseFile(fset, rel, source, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return Result{}, fmt.Errorf("parse %s: %w", rel, err)
	}

	r := Result{Path: rel, Package: file.Name.Name, Imports: importsOf(file)}
	minted := map[string]bool{}
	fileID := model.NodeID(rel)

	covered := map[int]bool{} // 1-based lines a symbol span covers
	// owners maps a function declaration to the id its node was MINTED with.
	// The call walk must not recompute that id: the mint may have appended an
	// ordinal, and a second computation would produce an id no node has.
	owners := map[*ast.FuncDecl]model.NodeID{}
	for _, decl := range file.Decls {
		nodes := declNodes(fset, rel, source, decl, minted)
		if fn, ok := decl.(*ast.FuncDecl); ok && len(nodes) == 1 {
			owners[fn] = nodes[0].ID
		}
		for _, n := range nodes {
			r.Nodes = append(r.Nodes, n)
			r.Edges = append(r.Edges, RawEdge{
				Source: fileID, Relation: model.RelationContains,
				TargetID: n.ID, File: rel,
			})
			markCovered(covered, n.Span)
		}
	}

	r.Nodes = append([]model.Node{fileNode(fset, rel, source, file, covered)}, r.Nodes...)
	r.Edges = append(r.Edges, importEdges(rel, file)...)
	r.Edges = append(r.Edges, callEdges(rel, file, owners)...)
	return r, nil
}

// fileNode is the node that stands for the whole file.
//
// Its id is the path itself, with no '#'. That is not cosmetic: an import edge
// has the file as its source, and model.Validate requires an edge's source to
// be a node of the graph.
func fileNode(fset *token.FileSet, rel, source string, file *ast.File, covered map[int]bool) model.Node {
	sum := sha256.Sum256([]byte(source))
	lines := strings.Count(source, "\n") + 1
	return model.Node{
		ID:       model.NodeID(rel),
		Name:     path.Base(rel),
		Kind:     model.KindFile,
		Path:     rel,
		Span:     model.Span(fmt.Sprintf("L1-L%d", lines)),
		Exported: true,
		BodyHash: hex.EncodeToString(sum[:]),
		BodyText: collapse(residual(source, covered)),
	}
}

// residual is the file's text outside every symbol span: the import header,
// package constants, the package comment.
//
// Symbol bodies are indexed on their own nodes, so repeating them here would
// store most of the repository twice. What is left is exactly what makes a file
// findable by a word that lives in no function.
func residual(source string, covered map[int]bool) string {
	var keep []string
	for i, line := range strings.Split(source, "\n") {
		if !covered[i+1] {
			keep = append(keep, line)
		}
	}
	return strings.Join(keep, "\n")
}

// markCovered records the lines of a span as belonging to a symbol.
//
// The span is parsed by model.Span.Lines and not by a copy here: the extractor
// writes that form and the query reads it, and two parsers for one string are
// one too many.
func markCovered(covered map[int]bool, span model.Span) {
	from, to, ok := span.Lines()
	if !ok {
		return
	}
	for i := from; i <= to; i++ {
		covered[i] = true
	}
}

```

Und der Teil, der eine Deklaration in Knoten übersetzt — in derselben Datei:

```go
// declNodes turns one top-level declaration into its nodes.
//
// A grouped `type ( ... )` yields one node per spec and never one for the
// declaration: hashing the GenDecl would make a change to one type change the
// hash of every other in the group, and `check` would report them all.
func declNodes(fset *token.FileSet, rel, source string, decl ast.Decl, minted map[string]bool) []model.Node {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		return []model.Node{funcNode(fset, rel, source, d, minted)}
	case *ast.GenDecl:
		if d.Tok != token.TYPE {
			// No const and no var nodes: Graft emits none for Go, and a const
			// block reaches a query through the file node's residual instead.
			return nil
		}
		var out []model.Node
		for _, spec := range d.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			out = append(out, typeNode(fset, rel, source, ts, minted))
		}
		return out
	default:
		return nil
	}
}

// funcNode is a function or a method.
func funcNode(fset *token.FileSet, rel, source string, d *ast.FuncDecl, minted map[string]bool) model.Node {
	name := d.Name.Name
	kind := model.Kind("function")
	owner := ""
	qualified := name
	if d.Recv != nil {
		kind = "method"
		owner = receiverType(d.Recv)
		if owner != "" {
			// Methods do not nest in Go, so the id qualifies by the receiver or
			// two receivers' same-named methods would collide. The name stays
			// bare, so a call `u.Save()` can hit it.
			qualified = owner + "." + name
		}
	}
	// The header ends where the body opens; a body-less declaration is its own
	// header.
	headerEnd := d.End()
	if d.Body != nil {
		headerEnd = d.Body.Pos()
	}
	return model.Node{
		ID:        model.NodeID(mintID(rel+"#"+qualified, minted)),
		Name:      name,
		Kind:      kind,
		Owner:     owner,
		Path:      rel,
		Span:      span(fset, d.Pos(), d.End()),
		Signature: collapse(slice(fset, source, d.Pos(), headerEnd)),
		Exported:  exported(name),
		BodyHash:  hash(slice(fset, source, d.Pos(), d.End())),
		BodyText:  collapse(slice(fset, source, d.Pos(), d.End())),
	}
}

// typeNode is one TypeSpec: a struct, an interface or a named type.
//
// The signature deviates from Graft's code and follows Graft's comment. Its
// type_spec starts at the NAME and its header ends at the `struct` keyword, so
// its signature for `type Cache struct { ... }` is the bare "Cache" -- a value
// no test of the reference pins. See 5.2.1 of the G2 spec.
func typeNode(fset *token.FileSet, rel, source string, ts *ast.TypeSpec, minted map[string]bool) model.Node {
	kind := model.Kind("type")
	sig := "type " + collapse(slice(fset, source, ts.Pos(), ts.End()))
	switch ts.Type.(type) {
	case *ast.StructType:
		kind = "struct"
		sig = "type " + ts.Name.Name + " struct"
	case *ast.InterfaceType:
		kind = "interface"
		sig = "type " + ts.Name.Name + " interface"
	}
	return model.Node{
		ID:        model.NodeID(mintID(rel+"#"+ts.Name.Name, minted)),
		Name:      ts.Name.Name,
		Kind:      kind,
		Path:      rel,
		Span:      span(fset, ts.Pos(), ts.End()),
		Signature: sig,
		Exported:  exported(ts.Name.Name),
		BodyHash:  hash(slice(fset, source, ts.Pos(), ts.End())),
		BodyText:  collapse(slice(fset, source, ts.Pos(), ts.End())),
	}
}

// span is "L<from>-L<to>", 1-based, over a node's whole extent.
func span(fset *token.FileSet, from, to token.Pos) model.Span {
	return model.Span(fmt.Sprintf("L%d-L%d", fset.Position(from).Line, fset.Position(to).Line))
}

// slice is the source text between two positions.
func slice(fset *token.FileSet, source string, from, to token.Pos) string {
	a, b := fset.Position(from).Offset, fset.Position(to).Offset
	if a < 0 || b > len(source) || a > b {
		return ""
	}
	return source[a:b]
}

// hash is sha256 as full hex. Truncating would save bytes in wiring.json and
// buy a collision nobody would debug.
func hash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
```

Die beiden Kantenbauer `importEdges` und `callEdges` kommen in Task 4. Damit
dieser Task übersetzt, zunächst zwei Rümpfe, die nichts liefern:

```go
// importEdges is Task 4.
func importEdges(rel string, file *ast.File) []RawEdge { return nil }

// callEdges is Task 4.
func callEdges(rel string, file *ast.File, owners map[*ast.FuncDecl]model.NodeID) []RawEdge {
	return nil
}
```

- [ ] **Schritt 6: Lauf, der grün sein muss**

```sh
go test ./internal/code/extract/golang/ -count=1
```

Erwartet: PASS. Der `~3`-Test verlangt, dass `go/parser` eine Datei mit drei
gleichnamigen Funktionen noch bis zum Ende parst — tut er, doppelte
Deklarationen sind ein Typprüfer-Fehler und keiner des Parsers.

- [ ] **Schritt 7: Commit**

```sh
git add internal/code/extract/golang
git commit -F ../msg.txt
```

Nachricht:

```
Extract the nodes of a Go file

Five kinds and the file itself: function, method, struct, interface,
named type. No constants and no variables -- Graft emits none for Go,
and a const block reaches a query through the file node's residual text.

A method's id qualifies by its receiver, because methods do not nest in
Go and two receivers' same-named methods would otherwise collide; the
name stays bare so a call on a receiver can hit it. The body hash runs
over Pos()..End(), which leaves the doc comment out and so matches
tree-sitter, where a comment is a sibling: rewording a comment must not
read as drift.

The signature of a struct follows Graft's comment rather than Graft's
code, which cuts it down to the bare type name. No test of the reference
pins that value; see 5.2.1 of the spec.
```

---

### Task 4: `internal/code/extract/golang` — Rohkanten

**Files:**
- Create: `internal/code/extract/golang/scope.go`
- Modify: `internal/code/extract/golang/extract.go` (die beiden Rümpfe aus Task 3)
- Create: `internal/code/extract/golang/testdata/edges.go.txt`
- Test: `internal/code/extract/golang/edges_test.go`

**Interfaces:**
- Consumes: `RawEdge`, `Result`, `File` aus Task 3.
- Produces: `importEdges` und `callEdges` liefern echte Kanten; `RawEdge.Owner`
  trägt bei einem Memberaufruf den Empfängertyp, `RawEdge.Specifier` bei einem
  Paketselektor den Importpfad des Pakets.

Drei Relationen und nicht mehr: `contains`, `imports`, `calls`. **Kein
`extends`, kein `implements`** — Go hat kein explizites `implements`, und Grafts
Erbschaftskanten entstehen nur für `kind: class` und die JVM-/Swift-Typen.
**Kein `references`** — den Sammler für importierte Symbole gibt es in Graft nur
für TypeScript und PHP. Das ist Zuschnitt und keine Auslassung.

- [ ] **Schritt 1: Die Vorlage schreiben**

`internal/code/extract/golang/testdata/edges.go.txt`:

```go
package edges

import (
	"fmt"
	b "example.com/repo/blast"
	"example.com/repo/store"
	_ "example.com/repo/driver"
)

type Cache struct{ n int }

func (c *Cache) Get() int { return c.n }

func (c *Cache) Warm() int {
	// A call on the method's own receiver variable.
	return c.Get()
}

func local() int { return 1 }

func caller() int {
	// A call inside the same file.
	n := local()
	// A package selector, aliased.
	_ = b.New(nil)
	// A package selector, plain.
	_ = store.Open("x")
	// A shadowed package: from here `store` is a variable, not a package.
	store := Cache{}
	_ = store.Get()
	// A call into the standard library: the import stays an external string.
	fmt.Println(n)
	return n
}

func generic() {
	// A type argument list must be peeled before the selector is read.
	_ = b.Of[int](3)
}
```

- [ ] **Schritt 2: Den fehlschlagenden Test schreiben**

```go
package golang_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/model"
)

func edgesOf(t *testing.T, r golang.Result, rel model.Relation) []golang.RawEdge {
	t.Helper()
	var out []golang.RawEdge
	for _, e := range r.Edges {
		if e.Relation == rel {
			out = append(out, e)
		}
	}
	return out
}

func hasEdge(edges []golang.RawEdge, want golang.RawEdge) bool {
	for _, e := range edges {
		if e.Source == want.Source && e.Name == want.Name &&
			e.Owner == want.Owner && e.Receiver == want.Receiver {
			return true
		}
	}
	return false
}

func TestFileEmitsContainsFromTheFileToEverySymbol(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	contains := edgesOf(t, r, model.RelationContains)
	// One per symbol node, never one for the file itself.
	if len(contains) != len(r.Nodes)-1 {
		t.Fatalf("got %d contains edges for %d symbol nodes", len(contains), len(r.Nodes)-1)
	}
	for _, e := range contains {
		if e.Source != "pkg/edges.go" {
			t.Errorf("contains edge from %q, want the file node", e.Source)
		}
		if e.TargetID == "" {
			t.Error("a contains edge carries a resolved target id, not a name")
		}
	}
}

func TestFileEmitsOneImportEdgePerSpecifier(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	imports := edgesOf(t, r, model.RelationImports)
	want := map[string]bool{
		"fmt":                     false,
		"example.com/repo/blast":  false,
		"example.com/repo/store":  false,
		"example.com/repo/driver": false,
	}
	for _, e := range imports {
		if e.Source != "pkg/edges.go" {
			t.Errorf("import edge from %q, want the file node", e.Source)
		}
		if _, ok := want[e.Specifier]; !ok {
			t.Errorf("unexpected import %q", e.Specifier)
			continue
		}
		want[e.Specifier] = true
	}
	for spec, seen := range want {
		if !seen {
			// A blank import binds no selector but is still a dependency of the
			// file, so it keeps its edge.
			t.Errorf("import %q missing", spec)
		}
	}
}

func TestFileResolvesACallOnTheMethodsOwnReceiver(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	// `c.Get()` inside a method of *Cache: the receiver variable is known, so
	// the edge carries the owner and resolve can find the right method.
	if !hasEdge(calls, golang.RawEdge{Source: "pkg/edges.go#Cache.Warm", Name: "Get", Owner: "Cache"}) {
		t.Errorf("a call on the own receiver must carry its owner; got %+v", calls)
	}
}

func TestFileEmitsABareCallWithoutAnOwner(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	if !hasEdge(calls, golang.RawEdge{Source: "pkg/edges.go#caller", Name: "local"}) {
		t.Errorf("a bare call must be a raw edge with a name alone; got %+v", calls)
	}
}

func TestFileCarriesAnUnshadowedSelectorReceiverUnresolved(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	// The extractor does NOT decide that `b` is a package: the name a plain
	// import binds is the TARGET's package clause, and one file cannot see it.
	// `import "gopkg.in/yaml.v3"` binds `yaml`, not `v3`, and guessing here
	// would put the guess where nothing can correct it.
	for _, want := range []golang.RawEdge{
		{Source: "pkg/edges.go#caller", Name: "New", Receiver: "b"},
		{Source: "pkg/edges.go#caller", Name: "Open", Receiver: "store"},
		{Source: "pkg/edges.go#caller", Name: "Println", Receiver: "fmt"},
	} {
		if !hasEdge(calls, want) {
			t.Errorf("selector %s.%s must reach resolve unresolved; got %+v",
				want.Receiver, want.Name, calls)
		}
	}
	for _, e := range calls {
		if e.Specifier != "" {
			t.Errorf("a call edge carries no specifier; that is the resolver's job: %+v", e)
		}
	}
}

func TestFileRecordsThePackageClauseAndEveryImport(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Package != "edges" {
		t.Errorf("Package = %q, want the file's clause", r.Package)
	}
	want := map[string]string{
		"fmt":                     "",
		"example.com/repo/blast":  "b",
		"example.com/repo/store":  "",
		"example.com/repo/driver": "_",
	}
	if len(r.Imports) != len(want) {
		t.Fatalf("got %+v, want %d imports", r.Imports, len(want))
	}
	for _, imp := range r.Imports {
		alias, ok := want[imp.Path]
		if !ok {
			t.Errorf("unexpected import %+v", imp)
			continue
		}
		// The alias is raw, "_" included: the resolver has to tell "binds
		// nothing" from "binds its package clause".
		if imp.Alias != alias {
			t.Errorf("import %q alias = %q, want %q", imp.Path, imp.Alias, alias)
		}
	}
}

func TestFileTreatsAShadowedPackageNameAsAVariable(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	// After `store := Cache{}` the name is a local variable. Go code does this
	// constantly (`model := model.Decode(r)`), and letting the selector reach
	// resolve as a receiver-with-no-owner would wire a call into a package the
	// line never touches.
	for _, e := range calls {
		if e.Name == "Get" && e.Receiver != "" {
			t.Errorf("a shadowed name is a value, not a receiver to resolve; got %+v", e)
		}
	}
	// It is a member call on a known local type instead.
	if !hasEdge(calls, golang.RawEdge{Source: "pkg/edges.go#caller", Name: "Get", Owner: "Cache"}) {
		t.Errorf("the shadowed call must resolve against the local binding; got %+v", calls)
	}
}

func TestFilePeelsATypeArgumentList(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	// `b.Of[int](3)` must take the same path as `b.Of(3)`.
	if !hasEdge(calls, golang.RawEdge{
		Source: "pkg/edges.go#generic", Name: "Of", Receiver: "b",
	}) {
		t.Errorf("a generic call must peel its type arguments; got %+v", calls)
	}
}

func TestFileBindsALocalVariableOnlyInTheFourFormsGraftKnows(t *testing.T) {
	// var x T, x := T{}, x := &T{}, x := NewT(...) -- and nothing else. The
	// fourth is a convention, not a resolution. Widening this set widens the
	// set of call edges and owes its own reason.
	const src = `package p

type T struct{}

func (t T) M() {}

func NewT() T { return T{} }

func other() T { return T{} }

func f() {
	var a T
	a.M()
	b := T{}
	b.M()
	c := &T{}
	c.M()
	d := NewT()
	d.M()
	e := other()
	e.M()
}
`
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	owned := 0
	for _, e := range calls {
		if e.Name == "M" && e.Owner == "T" {
			owned++
		}
	}
	// a, b, c, d carry the owner; e does not -- a plain function's return type
	// is not something this extractor knows.
	if owned != 4 {
		t.Errorf("got %d owned calls of M, want 4; edges %+v", owned, calls)
	}
}

func TestFileAttributesACallToTheEnclosingSymbol(t *testing.T) {
	const src = "package p\n\nfunc a() {}\n\nfunc b() { a() }\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	if len(calls) != 1 || calls[0].Source != "p.go#b" {
		t.Fatalf("got %+v, want one call whose source is the calling function", calls)
	}
}

func TestFileAttributesAPackageLevelCallToTheFile(t *testing.T) {
	// A call in a var initialiser sits inside no function, so the file owns it.
	const src = "package p\n\nfunc a() int { return 1 }\n\nvar x = a()\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	if len(calls) != 1 || calls[0].Source != "p.go" {
		t.Fatalf("got %+v, want one call owned by the file node", calls)
	}
}

func TestFileEmitsNeitherHeritageNorReferenceEdges(t *testing.T) {
	const src = `package p

type Reader interface{ Read() string }

type Impl struct{ Reader }

func (i Impl) Read() string { return "" }
`
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	// Go has no explicit implements, and an embedded interface is not one
	// either. Graft emits heritage edges for kind:class and the JVM/Swift
	// types alone, and reference edges only where an import binds a symbol --
	// a collector it has for TypeScript and PHP, not for Go.
	for _, e := range r.Edges {
		switch e.Relation {
		case model.RelationExtends, model.RelationImplements, model.RelationReferences:
			t.Errorf("this extractor emits no %q edges; got %+v", e.Relation, e)
		}
	}
}
```

- [ ] **Schritt 3: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/extract/golang/ -run 'TestFileEmits|TestFileResolves|TestFileReads|TestFileTreats|TestFilePeels|TestFileBinds|TestFileAttributes' -v
```

Erwartet: FAIL — `importEdges` und `callEdges` liefern noch `nil`, also
„import "fmt" missing" und leere Aufruflisten.

- [ ] **Schritt 4: Den Gültigkeitsbereich-Stapel schreiben**

`internal/code/extract/golang/scope.go`:

```go
package golang

import "go/ast"

// scope is the set of names a position has in view, plus the local type binding
// of each.
//
// Function-scoped and deliberately over-shadowing: a frame is pushed per
// function and not per block, so a name declared inside an `if` shadows for the
// rest of the function. That direction is the safe one -- it drops call edges
// rather than inventing them -- and a per-block scope is a later refinement,
// not a correctness fix.
//
// It exists because `ast.File.Unresolved` would answer the one question this
// package asks of it -- is this name declared in the file? -- and is
// deprecated together with ast.Object as of Go 1.22. A new package does not
// build on a field on its way out.
//
// The question matters: Go shadows package names constantly.
//
//	graph := graph.New(g)   // from here `graph` is a variable
//
// Reading the second selector as a package would wire a call into a package the
// line never touches.
type scope struct {
	frames []map[string]string // name -> bound type, "" when unknown
}

func newScope() *scope {
	return &scope{frames: []map[string]string{{}}}
}

func (s *scope) push() { s.frames = append(s.frames, map[string]string{}) }

func (s *scope) pop() { s.frames = s.frames[:len(s.frames)-1] }

// declare records a name in the innermost frame, with the type it is bound to
// ("" when the form is one this package does not read).
func (s *scope) declare(name, typ string) {
	if name == "" || name == "_" {
		return
	}
	s.frames[len(s.frames)-1][name] = typ
}

// declared reports whether the name is in view at all -- the shadowing
// question.
func (s *scope) declared(name string) bool {
	for i := len(s.frames) - 1; i >= 0; i-- {
		if _, ok := s.frames[i][name]; ok {
			return true
		}
	}
	return false
}

// lookup returns the type a name is bound to, or "" when the name is unknown or
// its form was not one of the four below.
func (s *scope) lookup(name string) string {
	for i := len(s.frames) - 1; i >= 0; i-- {
		if typ, ok := s.frames[i][name]; ok {
			return typ
		}
	}
	return ""
}

// boundType is the type a right-hand side binds, in exactly the four forms
// Graft's own Go collector reads (bindings.ts handleGo):
//
//	var x T   /  var x *T   -> T
//	x := T{}                -> T
//	x := &T{}               -> T
//	x := NewT(...)          -> T   (a convention, not a resolution)
//
// Everything else binds nothing: a plain function's return type, a field
// access, a range variable, a channel receive. Widening this set widens the set
// of call edges, and that owes its own reason.
func boundType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.CompositeLit:
		return typeName(e.Type)
	case *ast.UnaryExpr:
		if lit, ok := e.X.(*ast.CompositeLit); ok {
			return typeName(lit.Type)
		}
	case *ast.CallExpr:
		if id, ok := peelIndex(e.Fun).(*ast.Ident); ok {
			if name, ok := constructorType(id.Name); ok {
				return name
			}
		}
	}
	return ""
}

// constructorType reads "NewCache" as "Cache". Go's own convention, and the
// only heuristic in this file: it is what lets `c := NewCache()` bind.
func constructorType(fn string) (string, bool) {
	const prefix = "New"
	if len(fn) <= len(prefix) || fn[:len(prefix)] != prefix {
		return "", false
	}
	rest := fn[len(prefix):]
	if !exported(rest) {
		return "", false
	}
	return rest, true
}

// typeName is the bare name of a type expression, pointer and type arguments
// peeled.
func typeName(expr ast.Expr) string {
	switch t := peelIndex(expr).(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return typeName(t.X)
	}
	return ""
}

// peelIndex strips a type argument list, so `Of[int]` reads as `Of`.
func peelIndex(expr ast.Expr) ast.Expr {
	switch e := expr.(type) {
	case *ast.IndexExpr:
		return e.X
	case *ast.IndexListExpr:
		return e.X
	}
	return expr
}
```

- [ ] **Schritt 5: Die beiden Kantenbauer schreiben**

Die Rümpfe aus Task 3 in `extract.go` ersetzen:

```go
// importsOf is every import of a file, alias exactly as written.
//
// "_" and "." are recorded with their alias as written: they bind no selector,
// and the resolver has to be able to tell "this import binds nothing" from
// "this import binds its package clause".
func importsOf(file *ast.File) []Import {
	var out []Import
	for _, spec := range file.Imports {
		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		imp := Import{Path: p}
		if spec.Name != nil {
			imp.Alias = spec.Name.Name
		}
		out = append(out, imp)
	}
	return out
}

// importEdges is one edge per import specifier, from the file node.
//
// A blank import binds no selector and is still a dependency of the file, so it
// keeps its edge. A dot import binds no selector either; the same holds.
func importEdges(rel string, file *ast.File) []RawEdge {
	var out []RawEdge
	for _, imp := range importsOf(file) {
		out = append(out, RawEdge{
			Source: model.NodeID(rel), Relation: model.RelationImports,
			Specifier: imp.Path, File: rel,
		})
	}
	return out
}

// callEdges walks every function body and the package-level initialisers and
// emits one raw edge per call site.
//
// Three shapes reach resolve, and the difference is what resolve is allowed to
// assume:
//
//   - a bare name          -- resolve tries the same file, then a unique match
//   - a name plus an owner -- a member call on a known local type
//   - a name plus a specifier -- a package selector; resolve looks in that
//     package alone
func callEdges(rel string, file *ast.File, owners map[*ast.FuncDecl]model.NodeID) []RawEdge {
	var out []RawEdge
	// A call outside every function -- in a var initialiser -- is owned by the
	// file, the same node that owns the imports.
	walkCalls(file, rel, model.NodeID(rel), owners, &out)
	return out
}
```

Und der Läufer selbst, ebenfalls in `extract.go`:

```go
// walkCalls descends the file, keeping a scope stack and the symbol a call
// belongs to.
func walkCalls(file *ast.File, rel string, fileID model.NodeID, owners map[*ast.FuncDecl]model.NodeID, out *[]RawEdge) {
	sc := newScope()
	// The file's own package-level names: a call may target one of them, and a
	// local of the same name must shadow it.
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil {
				sc.declare(d.Name.Name, "")
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					sc.declare(s.Name.Name, "")
				case *ast.ValueSpec:
					for _, n := range s.Names {
						sc.declare(n.Name, typeName(s.Type))
					}
				}
			}
		}
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			// A package-level initialiser: its calls belong to the file.
			collect(decl, rel, fileID, sc, out)
			continue
		}
		owner, ok := owners[fn]
		if !ok {
			// No node was minted for this declaration, so nothing can own its
			// calls. Attributing them to the file would invent a caller.
			continue
		}
		sc.push()
		// The receiver variable is bound to its own type: that is how `c.Get()`
		// inside a method of *Cache finds Cache.
		if fn.Recv != nil {
			sc.declare(receiverVar(fn.Recv), receiverType(fn.Recv))
		}
		declareParams(sc, fn.Type)
		collect(fn, rel, owner, sc, out)
		sc.pop()
	}
}

// declareParams puts a function's parameters and named results in view. Their
// types are recorded, so a member call on a parameter resolves.
func declareParams(sc *scope, ft *ast.FuncType) {
	for _, list := range []*ast.FieldList{ft.Params, ft.Results, ft.TypeParams} {
		if list == nil {
			continue
		}
		for _, field := range list.List {
			for _, name := range field.Names {
				sc.declare(name.Name, typeName(field.Type))
			}
		}
	}
}

// collect walks one declaration's statements, tracking declarations as it goes
// and emitting a raw edge per call.
func collect(node ast.Node, rel string, owner model.NodeID, sc *scope, out *[]RawEdge) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			if s.Tok == token.DEFINE {
				for i, lhs := range s.Lhs {
					id, ok := lhs.(*ast.Ident)
					if !ok {
						continue
					}
					typ := ""
					if i < len(s.Rhs) {
						typ = boundType(s.Rhs[i])
					}
					sc.declare(id.Name, typ)
				}
			}
		case *ast.ValueSpec:
			for _, name := range s.Names {
				sc.declare(name.Name, typeName(s.Type))
			}
		case *ast.RangeStmt:
			// A range variable binds no type this package can read, but it must
			// shadow: `for store := range m` hides the package `store`.
			for _, e := range []ast.Expr{s.Key, s.Value} {
				if id, ok := e.(*ast.Ident); ok {
					sc.declare(id.Name, "")
				}
			}
		case *ast.CallExpr:
			if e, ok := callEdge(s, rel, owner, sc); ok {
				*out = append(*out, e)
			}
		}
		return true
	})
}

// callEdge reads one call site.
//
// Shadowing is decided HERE, because only this package sees the file's scopes;
// whether an unshadowed receiver names a package is decided in resolve, because
// only that sees the target's package clause. Splitting the question along that
// line is what keeps `import "gopkg.in/yaml.v3"` from binding `v3`.
func callEdge(call *ast.CallExpr, rel string, owner model.NodeID, sc *scope) (RawEdge, bool) {
	switch fn := peelIndex(call.Fun).(type) {
	case *ast.Ident:
		return RawEdge{
			Source: owner, Relation: model.RelationCalls,
			Name: fn.Name, File: rel,
		}, true
	case *ast.SelectorExpr:
		recv, ok := peelIndex(fn.X).(*ast.Ident)
		if !ok {
			// A chained or computed receiver -- `a.b().c()`, `m[k].c()`. Graft
			// drops these too: without a receiver type a bare method name says
			// nothing about what it belongs to.
			return RawEdge{}, false
		}
		if !sc.declared(recv.Name) {
			// Declared nowhere in view. It may be a package, and resolve is the
			// only side that can say so.
			return RawEdge{
				Source: owner, Relation: model.RelationCalls,
				Name: fn.Sel.Name, Receiver: recv.Name, File: rel,
			}, true
		}
		typ := sc.lookup(recv.Name)
		if typ == "" {
			// Declared, but bound to nothing this package reads. Dropping beats
			// guessing: a unique bare method name says nothing about its
			// receiver.
			return RawEdge{}, false
		}
		return RawEdge{
			Source: owner, Relation: model.RelationCalls,
			Name: fn.Sel.Name, Owner: typ, File: rel,
		}, true
	default:
		return RawEdge{}, false
	}
}
```

Der Import-Block von `extract.go` braucht dazu `strconv`.

- [ ] **Schritt 6: Lauf, der grün sein muss**

```sh
go test ./internal/code/extract/golang/ -count=1 -cover
```

Erwartet: PASS bei 100 % je Funktion. Der Fall „chained receiver" in
`callEdge` braucht einen eigenen Test, falls die Abdeckung ihn nicht erreicht:

```go
func TestFileDropsACallOnAComputedReceiver(t *testing.T) {
	const src = "package p\n\nfunc f(m map[string]T) { m[\"k\"].M() }\n\ntype T struct{}\n\nfunc (T) M() {}\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	// Without a receiver type a bare method name says nothing about what it
	// belongs to. Graft drops it; so does this.
	for _, e := range r.Edges {
		if e.Relation == model.RelationCalls && e.Name == "M" {
			t.Errorf("a computed receiver must yield no call edge; got %+v", e)
		}
	}
}
```

- [ ] **Schritt 7: Commit**

```sh
git add internal/code/extract/golang
git commit -F ../msg.txt
```

Nachricht:

```
Extract the raw edges of a Go file

Three relations and no more: contains, imports, calls. Go has no
explicit implements, and Graft emits heritage edges for kind:class and
the JVM/Swift types alone; reference edges need an import that binds a
symbol, a collector Graft has for TypeScript and PHP and not for Go.
That is scope, not an omission.

A call site asks two questions in this order: is the name shadowed
anywhere in view, and only then does a file header bind it to a package.
Go shadows package names constantly -- `graph := graph.New(g)` -- and
reading the second selector as a package would wire a call into a
package the line never touches. ast.File.Unresolved would answer the
first question and is deprecated with ast.Object as of Go 1.22, so this
package keeps its own scope stack.

A local binds a type in the four forms Graft's own Go collector reads
and in no others: var x T, x := T{}, x := &T{}, x := NewT(). A plain
function's return type is not among them.
```

---

### Task 5: `internal/code/resolve` — Rohkanten werden Kanten

**Files:**
- Create: `internal/code/resolve/gomod.go`
- Create: `internal/code/resolve/resolve.go`
- Test: `internal/code/resolve/gomod_test.go`, `internal/code/resolve/resolve_test.go`

**Interfaces:**
- Consumes: `golang.Result`, `golang.RawEdge`, `golang.Version`, `model.*`.
- Produces:

```go
type Module struct {
	Dir  string // repo-relative, slash-separated; "" for a go.mod at the root
	Path string // the module directive
}

func Modules(root string, files []string) ([]Module, error)
func Graph(files []golang.Result, mods []Module) *model.Graph
```

`Graph` liefert einen fertigen `*model.Graph` mit `Meta.Extractor` gesetzt und
sortiert; `Validate` darauf muss grün sein.

- [ ] **Schritt 1: Den fehlschlagenden Test für `go.mod` schreiben**

```go
package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/code/resolve"
)

func TestModulesReadsOnlyTheModuleDirective(t *testing.T) {
	root := t.TempDir()
	body := "// a comment\nmodule example.com/repo\n\ngo 1.25.0\n\nrequire github.com/x/y v1.2.3\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	mods, err := resolve.Modules(root, []string{"go.mod"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Path != "example.com/repo" || mods[0].Dir != "" {
		t.Fatalf("got %+v, want one module example.com/repo at the root", mods)
	}
}

func TestModulesFindsANestedGoMod(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(rel, body string) {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/repo\n")
	write("tools/go.mod", "module example.com/repo/tools\n")

	mods, err := resolve.Modules(root, []string{"go.mod", "tools/go.mod"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 2 {
		t.Fatalf("got %+v, want both modules", mods)
	}
	var tools resolve.Module
	for _, m := range mods {
		if m.Dir == "tools" {
			tools = m
		}
	}
	if tools.Path != "example.com/repo/tools" {
		t.Fatalf("got %+v, want the nested module at tools", mods)
	}
}

func TestModulesIgnoresAGoModWithoutADirective(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("go 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mods, err := resolve.Modules(root, []string{"go.mod"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 0 {
		t.Fatalf("got %+v, want none: a file without a module directive declares nothing", mods)
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/resolve/ -run TestModules -v
```

Erwartet: Übersetzungsfehler, das Paket gibt es nicht.

- [ ] **Schritt 3: `gomod.go` schreiben**

```go
// Package resolve turns the raw edges of every extracted file into edges with
// node ids and a confidence, and drops what stays ambiguous.
//
// The two-tier provenance is Graft's (src/graph/resolve.ts):
//
//   - extracted -- the target is certain: a hit in the same file, an import
//     specifier, structural containment, or a package selector resolved
//     against the one package it can mean
//   - inferred  -- a bare name resolved through exactly one match across files
//
// Ambiguous means dropped, never guessed. Graft's own header says why: name
// guessing "halved precision", and one same-named symbol once collected 1040
// in-edges across 476 files, so every pull request touching it dragged a whole
// backend into its blast radius.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/resolve.ts.
package resolve

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Module is one go.mod of the repository.
type Module struct {
	Dir  string
	Path string
}

// Modules reads the module directive of every go.mod among files.
//
// Only that directive, by hand: golang.org/x/mod would parse the whole file and
// is a dependency this stage does not take. A go.mod without a module
// directive declares nothing and is skipped.
func Modules(root string, files []string) ([]Module, error) {
	var out []Module
	for _, rel := range files {
		if path.Base(rel) != "go.mod" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		mod, ok := moduleDirective(string(b))
		if !ok {
			continue
		}
		dir := path.Dir(rel)
		if dir == "." {
			dir = ""
		}
		out = append(out, Module{Dir: dir, Path: mod})
	}
	// Longest path first, so resolveImport can take the first match and have
	// the most specific module.
	sort.Slice(out, func(i, j int) bool { return len(out[i].Path) > len(out[j].Path) })
	return out, nil
}

// moduleDirective is the argument of the first `module` line.
func moduleDirective(body string) (string, bool) {
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1], true
		}
	}
	return "", false
}

// importDir is the repo-relative directory an import path names.
//
// The bool is not decoration: the root package of a root module legitimately
// lives at "", and "" is also what a caller would use for "not found". Returning
// both apart is what keeps `import "example.com/repo"` from reading as the
// standard library. Graft has the same distinction and spells it with a
// sentinel; in Go the pair is the honest form.
//
// The longest module path wins, so a nested module in a monorepo beats its
// parent.
func importDir(spec string, mods []Module) (string, bool) {
	for _, m := range mods {
		var sub string
		switch {
		case spec == m.Path:
			sub = ""
		case strings.HasPrefix(spec, m.Path+"/"):
			sub = spec[len(m.Path)+1:]
		default:
			continue
		}
		switch {
		case m.Dir == "":
			return sub, true
		case sub == "":
			return m.Dir, true
		default:
			return m.Dir + "/" + sub, true
		}
	}
	return "", false
}

// dirOf is the directory a node's path lies in, with path.Dir's "." for a file
// at the repository root spelled as "" -- the same form importDir returns, so
// the two can be compared at all.
func dirOf(p string) string {
	dir := path.Dir(p)
	if dir == "." {
		return ""
	}
	return dir
}
```

- [ ] **Schritt 4: Den fehlschlagenden Test für `Graph` schreiben**

```go
package resolve_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// result is a hand-built extraction of one file, so these tests exercise
// resolve alone and never the parser. The package clause defaults to the last
// segment of the directory, which is the ordinary case.
func result(rel string, nodes []model.Node, edges []golang.RawEdge) golang.Result {
	return golang.Result{Path: rel, Package: path.Base(path.Dir(rel)), Nodes: nodes, Edges: edges}
}

// importing is result plus the imports the file wrote, which is what a selector
// resolves through.
func importing(rel string, imports []golang.Import, nodes []model.Node, edges []golang.RawEdge) golang.Result {
	r := result(rel, nodes, edges)
	r.Imports = imports
	return r
}

func fileNode(rel string) model.Node {
	return model.Node{ID: model.NodeID(rel), Name: rel, Kind: model.KindFile, Path: rel, BodyHash: "h", Exported: true}
}

func fn(rel, name string, exported bool) model.Node {
	return model.Node{
		ID: model.NodeID(rel + "#" + name), Name: name, Kind: "function",
		Path: rel, BodyHash: "h", Exported: exported,
	}
}

func edgeBetween(g *model.Graph, from, to model.NodeID, rel model.Relation) *model.Edge {
	for i := range g.Edges {
		e := &g.Edges[i]
		if e.Source == from && e.Target == to && e.Relation == rel {
			return e
		}
	}
	return nil
}

func TestGraphResolvesASameFileCallAsExtracted(t *testing.T) {
	files := []golang.Result{result("a.go",
		[]model.Node{fileNode("a.go"), fn("a.go", "caller", false), fn("a.go", "local", false)},
		[]golang.RawEdge{{Source: "a.go#caller", Relation: model.RelationCalls, Name: "local", File: "a.go"}},
	)}

	g := resolve.Graph(files, nil)
	e := edgeBetween(g, "a.go#caller", "a.go#local", model.RelationCalls)
	if e == nil {
		t.Fatalf("edge missing; got %+v", g.Edges)
	}
	// Same file, so the target is certain.
	if e.Confidence != model.ConfidenceExtracted {
		t.Errorf("confidence = %q, want extracted", e.Confidence)
	}
}

func TestGraphResolvesAUniqueCrossFileCallAsInferred(t *testing.T) {
	files := []golang.Result{
		result("a.go",
			[]model.Node{fileNode("a.go"), fn("a.go", "caller", false)},
			[]golang.RawEdge{{Source: "a.go#caller", Relation: model.RelationCalls, Name: "helper", File: "a.go"}},
		),
		result("b.go", []model.Node{fileNode("b.go"), fn("b.go", "helper", false)}, nil),
	}

	g := resolve.Graph(files, nil)
	e := edgeBetween(g, "a.go#caller", "b.go#helper", model.RelationCalls)
	if e == nil {
		t.Fatalf("edge missing; got %+v", g.Edges)
	}
	// One match across files: shadowing could in principle fool this, so it is
	// inferred and not extracted.
	if e.Confidence != model.ConfidenceInferred {
		t.Errorf("confidence = %q, want inferred", e.Confidence)
	}
}

func TestGraphDropsAnAmbiguousCall(t *testing.T) {
	files := []golang.Result{
		result("a.go",
			[]model.Node{fileNode("a.go"), fn("a.go", "caller", false)},
			[]golang.RawEdge{{Source: "a.go#caller", Relation: model.RelationCalls, Name: "helper", File: "a.go"}},
		),
		result("b_windows.go", []model.Node{fileNode("b_windows.go"), fn("b_windows.go", "helper", false)}, nil),
		result("b_other.go", []model.Node{fileNode("b_other.go"), fn("b_other.go", "helper", false)}, nil),
	}

	g := resolve.Graph(files, nil)
	// Build constraints are not evaluated: a platform pair is two definitions,
	// the call is ambiguous, and the edge falls. Sound by the rule, and stated
	// in 6.1 of the spec so nobody files it as a bug.
	for _, e := range g.Edges {
		if e.Relation == model.RelationCalls {
			t.Errorf("an ambiguous call must yield no edge; got %+v", e)
		}
	}
}

func TestGraphResolvesAMemberCallThroughTheOwner(t *testing.T) {
	method := model.Node{
		ID: "a.go#Cache.Get", Name: "Get", Kind: "method", Owner: "Cache",
		Path: "a.go", BodyHash: "h", Exported: true,
	}
	files := []golang.Result{result("a.go",
		[]model.Node{fileNode("a.go"), fn("a.go", "caller", false), method},
		[]golang.RawEdge{{
			Source: "a.go#caller", Relation: model.RelationCalls,
			Name: "Get", Owner: "Cache", File: "a.go",
		}},
	)}

	g := resolve.Graph(files, nil)
	if edgeBetween(g, "a.go#caller", "a.go#Cache.Get", model.RelationCalls) == nil {
		t.Fatalf("a member call must resolve against the owner-qualified index; got %+v", g.Edges)
	}
}

func TestGraphResolvesAPackageSelectorInsideTheTargetPackageOnly(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []golang.Result{
		importing("cli/run.go",
			[]golang.Import{{Path: "example.com/repo/blast"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]golang.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "New", Receiver: "blast", File: "cli/run.go",
			}},
		),
		result("blast/index.go", []model.Node{fileNode("blast/index.go"), fn("blast/index.go", "New", true)}, nil),
		// A same-named function elsewhere must not be reachable this way.
		result("other/thing.go", []model.Node{fileNode("other/thing.go"), fn("other/thing.go", "New", true)}, nil),
	}

	g := resolve.Graph(files, mods)
	e := edgeBetween(g, "cli/run.go#run", "blast/index.go#New", model.RelationCalls)
	if e == nil {
		t.Fatalf("a package selector must resolve inside the target package; got %+v", g.Edges)
	}
	if e.Confidence != model.ConfidenceExtracted {
		t.Errorf("confidence = %q, want extracted: the package names the target", e.Confidence)
	}
	if edgeBetween(g, "cli/run.go#run", "other/thing.go#New", model.RelationCalls) != nil {
		t.Error("a same-named symbol outside the target package must never be reached")
	}
}

func TestGraphSelectorSkipsMethodsAndTestFiles(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	method := model.Node{
		ID: "blast/index.go#T.New", Name: "New", Kind: "method", Owner: "T",
		Path: "blast/index.go", BodyHash: "h", Exported: true,
	}
	files := []golang.Result{
		importing("cli/run.go",
			[]golang.Import{{Path: "example.com/repo/blast"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]golang.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "New", Receiver: "blast", File: "cli/run.go",
			}},
		),
		// Only a method of that name, plus a function of that name in a test
		// file. An importer sees neither.
		result("blast/index.go", []model.Node{fileNode("blast/index.go"), method}, nil),
		result("blast/index_test.go", []model.Node{fileNode("blast/index_test.go"), fn("blast/index_test.go", "New", true)}, nil),
	}

	g := resolve.Graph(files, mods)
	for _, e := range g.Edges {
		if e.Relation == model.RelationCalls {
			t.Errorf("neither a method nor a test-file symbol is a selector candidate; got %+v", e)
		}
	}
}

func TestGraphResolvesAnInRepoImportToANonTestRepresentative(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []golang.Result{
		result("cli/run.go",
			[]model.Node{fileNode("cli/run.go")},
			[]golang.RawEdge{{
				Source: "cli/run.go", Relation: model.RelationImports,
				Specifier: "example.com/repo/blast", File: "cli/run.go",
			}},
		),
		// Sorted first by id, and invisible to an importer.
		result("blast/a_test.go", []model.Node{fileNode("blast/a_test.go")}, nil),
		result("blast/index.go", []model.Node{fileNode("blast/index.go")}, nil),
	}

	g := resolve.Graph(files, mods)
	if edgeBetween(g, "cli/run.go", "blast/index.go", model.RelationImports) == nil {
		t.Fatalf("the representative must skip test files; got %+v", g.Edges)
	}
}

func TestGraphKeepsAnExternalImportAsAString(t *testing.T) {
	files := []golang.Result{result("a.go",
		[]model.Node{fileNode("a.go")},
		[]golang.RawEdge{{
			Source: "a.go", Relation: model.RelationImports, Specifier: "fmt", File: "a.go",
		}},
	)}

	g := resolve.Graph(files, nil)
	e := edgeBetween(g, "a.go", "fmt", model.RelationImports)
	if e == nil {
		t.Fatalf("an external import keeps its package path as the target; got %+v", g.Edges)
	}
	// blast keeps this as a hit without a node, pagerank drops the edge. That
	// divergence is G1's ruling and stays.
	if e.Confidence != model.ConfidenceExtracted {
		t.Errorf("confidence = %q, want extracted: pointing outward is a fact, not a doubt", e.Confidence)
	}
}

func TestGraphEmitsNoCallEdgeForAnExternalSelector(t *testing.T) {
	files := []golang.Result{result("a.go",
		[]model.Node{fileNode("a.go"), fn("a.go", "f", false)},
		[]golang.RawEdge{{
			Source: "a.go#f", Relation: model.RelationCalls,
			Name: "Println", Receiver: "fmt", File: "a.go",
		}},
	)}

	g := resolve.Graph(files, nil)
	for _, e := range g.Edges {
		if e.Relation == model.RelationCalls {
			t.Errorf("a call into a package outside the repository has no node to point at; got %+v", e)
		}
	}
}

func TestGraphResolvesASelectorIntoTheRootPackage(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []golang.Result{
		importing("cli/run.go",
			[]golang.Import{{Path: "example.com/repo"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]golang.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "Root", Receiver: "repo", File: "cli/run.go",
			}},
		),
		// A file at the repository root: its directory is "" and not ".", or it
		// could never be found by the import path of the root module.
		golang.Result{
			Path: "root.go", Package: "repo",
			Nodes: []model.Node{fileNode("root.go"), fn("root.go", "Root", true)},
		},
	}

	g := resolve.Graph(files, mods)
	if edgeBetween(g, "cli/run.go#run", "root.go#Root", model.RelationCalls) == nil {
		t.Fatalf("the root package must be reachable; got %+v", g.Edges)
	}
}

func TestGraphBindsAPlainImportByThePackageClauseAndNotThePathTail(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []golang.Result{
		importing("cli/run.go",
			[]golang.Import{{Path: "example.com/repo/yaml.v3"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]golang.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "Marshal", Receiver: "yaml", File: "cli/run.go",
			}},
		),
		// The directory's last segment is "yaml.v3", the clause is "yaml", and
		// Go binds the clause. Guessing the path tail would drop this call --
		// and versioned module paths make the case ordinary, not exotic.
		golang.Result{
			Path: "yaml.v3/marshal.go", Package: "yaml",
			Nodes: []model.Node{fileNode("yaml.v3/marshal.go"), fn("yaml.v3/marshal.go", "Marshal", true)},
		},
	}

	g := resolve.Graph(files, mods)
	if edgeBetween(g, "cli/run.go#run", "yaml.v3/marshal.go#Marshal", model.RelationCalls) == nil {
		t.Fatalf("a plain import binds the target's package clause; got %+v", g.Edges)
	}
}

func TestGraphIgnoresABlankAndADotImportForASelector(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []golang.Result{
		importing("cli/run.go",
			[]golang.Import{
				{Alias: "_", Path: "example.com/repo/driver"},
				{Alias: ".", Path: "example.com/repo/dsl"},
			},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]golang.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "Open", Receiver: "driver", File: "cli/run.go",
			}},
		),
		golang.Result{
			Path: "driver/driver.go", Package: "driver",
			Nodes: []model.Node{fileNode("driver/driver.go"), fn("driver/driver.go", "Open", true)},
		},
	}

	g := resolve.Graph(files, mods)
	// Neither binds a selector name. A `driver.Open` in a file that only
	// blank-imports driver is some other driver entirely.
	for _, e := range g.Edges {
		if e.Relation == model.RelationCalls {
			t.Errorf("a blank import binds no selector; got %+v", e)
		}
	}
}

func TestGraphCarriesContainsThroughAndStampsTheExtractor(t *testing.T) {
	files := []golang.Result{result("a.go",
		[]model.Node{fileNode("a.go"), fn("a.go", "f", false)},
		[]golang.RawEdge{{
			Source: "a.go", Relation: model.RelationContains, TargetID: "a.go#f", File: "a.go",
		}},
	)}

	g := resolve.Graph(files, nil)
	e := edgeBetween(g, "a.go", "a.go#f", model.RelationContains)
	if e == nil || e.Confidence != model.ConfidenceExtracted {
		t.Fatalf("contains is already resolved and certain; got %+v", g.Edges)
	}
	if g.Meta.Extractor != golang.Version {
		t.Errorf("Meta.Extractor = %q, want %q", g.Meta.Extractor, golang.Version)
	}
	if g.Meta.Version == 0 || g.Meta.NodeCount != len(g.Nodes) || g.Meta.EdgeCount != len(g.Edges) {
		t.Errorf("meta = %+v, want the counts of the graph it describes", g.Meta)
	}
	if err := g.Validate(); err != nil {
		t.Errorf("a resolved graph must validate: %v", err)
	}
}

func TestGraphSortsNodesAndEdges(t *testing.T) {
	files := []golang.Result{
		result("b.go", []model.Node{fileNode("b.go")}, nil),
		result("a.go", []model.Node{fileNode("a.go")}, nil),
	}

	g := resolve.Graph(files, nil)
	// Byte order, so a rebuild of an unchanged tree is byte-identical and the
	// golden files do not wander between platforms.
	if g.Nodes[0].ID != "a.go" || g.Nodes[1].ID != "b.go" {
		t.Fatalf("nodes unsorted: %+v", g.Nodes)
	}
}
```

- [ ] **Schritt 5: `resolve.go` schreiben**

```go
// Graph resolves every raw edge and returns a whole wiring graph.
func Graph(files []golang.Result, mods []Module) *model.Graph {
	idx := index(files)
	g := &model.Graph{Meta: model.Meta{
		Version: schemaVersion, Extractor: golang.Version, Languages: []string{"go"},
	}}
	for _, f := range files {
		g.Nodes = append(g.Nodes, f.Nodes...)
	}
	for _, f := range files {
		for _, raw := range f.Edges {
			if e, ok := idx.resolveEdge(raw, mods); ok {
				g.Edges = append(g.Edges, e)
			}
		}
	}
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	sort.Slice(g.Edges, func(i, j int) bool {
		a, b := g.Edges[i], g.Edges[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Relation != b.Relation {
			return a.Relation < b.Relation
		}
		return a.Target < b.Target
	})
	g.Meta.NodeCount = len(g.Nodes)
	g.Meta.EdgeCount = len(g.Edges)
	return g
}

// schemaVersion is the version Graph writes. model refuses anything else, and
// the two are bumped together.
const schemaVersion = 2

// repoIndex is every lookup the resolver makes, built once.
type repoIndex struct {
	nodes      map[model.NodeID]model.Node
	perFile    map[string]map[string][]model.Node // path -> name -> functions
	global     map[string][]model.Node            // name -> functions, repo-wide
	byOwner    map[string][]model.Node            // "Owner.name" -> methods
	perDir     map[string]map[string][]model.Node // dir -> name -> functions, no tests
	filesInDir map[string][]string                // dir -> file node paths, no tests
	clauseOf   map[string]string                  // dir -> package clause, no tests
	importsOf  map[string][]golang.Import         // file path -> its imports
}

func index(files []golang.Result) *repoIndex {
	x := &repoIndex{
		nodes: map[model.NodeID]model.Node{}, perFile: map[string]map[string][]model.Node{},
		global: map[string][]model.Node{}, byOwner: map[string][]model.Node{},
		perDir: map[string]map[string][]model.Node{}, filesInDir: map[string][]string{},
		clauseOf: map[string]string{}, importsOf: map[string][]golang.Import{},
	}
	for _, f := range files {
		x.importsOf[f.Path] = f.Imports
		// The clause of a package is what an importer binds without an alias.
		// Test files are skipped: `package X_test` is a different clause for the
		// same directory, and no importer ever sees it.
		if !isTestFile(f.Path) && f.Package != "" {
			x.clauseOf[dirOf(f.Path)] = f.Package
		}
		for _, n := range f.Nodes {
			x.nodes[n.ID] = n
			switch {
			case n.Kind == model.KindFile:
				if !isTestFile(n.Path) {
					dir := dirOf(n.Path)
					x.filesInDir[dir] = append(x.filesInDir[dir], n.Path)
				}
			case n.Kind == "method":
				x.byOwner[n.Owner+"."+n.Name] = append(x.byOwner[n.Owner+"."+n.Name], n)
			case n.Kind == "function":
				if x.perFile[n.Path] == nil {
					x.perFile[n.Path] = map[string][]model.Node{}
				}
				x.perFile[n.Path][n.Name] = append(x.perFile[n.Path][n.Name], n)
				x.global[n.Name] = append(x.global[n.Name], n)
				// A selector's candidates: functions of the target package,
				// test files excluded -- an importer never sees them.
				if !isTestFile(n.Path) {
					dir := dirOf(n.Path)
					if x.perDir[dir] == nil {
						x.perDir[dir] = map[string][]model.Node{}
					}
					x.perDir[dir][n.Name] = append(x.perDir[dir][n.Name], n)
				}
			}
		}
	}
	for dir := range x.filesInDir {
		sort.Strings(x.filesInDir[dir])
	}
	return x
}

// isTestFile reports whether an importer would ever see this file. It would
// not: neither the `package X` tests nor the `package X_test` ones.
func isTestFile(p string) bool {
	return strings.HasSuffix(p, "_test.go")
}

// resolveEdge turns one raw edge into an edge, or reports that it is not sound
// enough to keep.
func (x *repoIndex) resolveEdge(raw golang.RawEdge, mods []Module) (model.Edge, bool) {
	switch raw.Relation {
	case model.RelationContains:
		// Already resolved by the extractor, and structural containment is
		// certain by construction.
		return model.Edge{
			Source: raw.Source, Target: raw.TargetID,
			Relation: model.RelationContains, Confidence: model.ConfidenceExtracted,
		}, true
	case model.RelationImports:
		return model.Edge{
			Source: raw.Source, Target: x.importTarget(raw.Specifier, mods),
			Relation: model.RelationImports, Confidence: model.ConfidenceExtracted,
		}, true
	case model.RelationCalls:
		return x.resolveCall(raw, mods)
	default:
		return model.Edge{}, false
	}
}

// importTarget is a representative file node of the imported package, or the
// raw package path when the import points outside every module of the
// repository.
//
// The representative is the lowest id among the package's NON-test files.
// Graft takes the lowest id outright; in Go that could point an import at
// `index_test.go`, a file the importer never sees.
func (x *repoIndex) importTarget(spec string, mods []Module) model.NodeID {
	dir, ok := importDir(spec, mods)
	if !ok {
		return model.NodeID(spec)
	}
	files := x.filesInDir[dir]
	if len(files) == 0 {
		// Inside a module of the repository, but nothing indexed there: an
		// empty package directory, or one holding only tests.
		return model.NodeID(spec)
	}
	return model.NodeID(files[0])
}

// resolveCall applies the three shapes a raw call may have.
func (x *repoIndex) resolveCall(raw golang.RawEdge, mods []Module) (model.Edge, bool) {
	switch {
	case raw.Receiver != "":
		return x.resolveSelector(raw, mods)
	case raw.Owner != "":
		// A member call on a known local type: the owner-qualified index is the
		// only sound answer, because a unique bare method name says nothing
		// about its receiver.
		return one(raw, x.byOwner[raw.Owner+"."+raw.Name], model.ConfidenceExtracted)
	default:
		if e, ok := one(raw, x.perFile[raw.File][raw.Name], model.ConfidenceExtracted); ok {
			return e, true
		}
		// One match across files. Shadowing could in principle fool this, so it
		// is inferred; several matches are dropped rather than guessed.
		return one(raw, x.global[raw.Name], model.ConfidenceInferred)
	}
}

// resolveSelector answers `pkg.Fn()` against the one package it can mean.
//
// This is where the port goes past Graft, which drops the case for Go. The rule
// is Graft's own, for a reference with a specifier (resolve.ts:229-237):
// resolve inside the named target alone, and only when exactly one candidate
// is there, "so a same-named symbol elsewhere in the repo cannot become a false
// edge".
//
// Candidates are functions. A call `pkg.T(x)` on a type is a conversion, and
// methods are excluded because a package may hold `func New()` and
// `func (x *T) New()` at once -- only the first can be what the selector means.
func (x *repoIndex) resolveSelector(raw golang.RawEdge, mods []Module) (model.Edge, bool) {
	dir, ok := x.packageDir(raw.File, raw.Receiver, mods)
	if !ok {
		// The receiver names no package of this repository: the standard
		// library, a third-party package, or something this extractor cannot
		// see at all. Nothing to point at.
		return model.Edge{}, false
	}
	return one(raw, x.perDir[dir][raw.Name], model.ConfidenceExtracted)
}

// packageDir answers which directory of the repository a selector's receiver
// names, for the file the selector stands in.
//
// Two rounds, and the order is Go's own:
//
//  1. an ALIAS binds outright -- `import b ".../blast"` makes `b.New` that
//     package, whatever the target calls itself
//  2. otherwise the bound name is the TARGET's package clause, which is why
//     this lives here and not in the extractor: `import "gopkg.in/yaml.v3"`
//     binds `yaml`, and only a side that has read the target directory knows
//     that
//
// "_" and "." bind no selector and are skipped in both rounds.
func (x *repoIndex) packageDir(file, receiver string, mods []Module) (string, bool) {
	var plain []golang.Import
	for _, imp := range x.importsOf[file] {
		switch imp.Alias {
		case "":
			plain = append(plain, imp)
		case "_", ".":
			// Binds nothing.
		case receiver:
			return importDir(imp.Path, mods)
		}
	}
	for _, imp := range plain {
		dir, ok := importDir(imp.Path, mods)
		if !ok {
			continue
		}
		if x.clauseOf[dir] == receiver {
			return dir, true
		}
	}
	return "", false
}

// one keeps an edge when the candidate set names exactly one node.
func one(raw golang.RawEdge, candidates []model.Node, conf model.Confidence) (model.Edge, bool) {
	if len(candidates) != 1 {
		return model.Edge{}, false
	}
	return model.Edge{
		Source: raw.Source, Target: candidates[0].ID,
		Relation: raw.Relation, Confidence: conf,
	}, true
}
```

Der Import-Block von `resolve.go` braucht `path`, `sort`, `strings`, dazu
`golang` und `model`.

- [ ] **Schritt 6: Lauf, der grün sein muss**

```sh
go test ./internal/code/resolve/ -count=1 -cover
```

Erwartet: PASS bei 100 % je Funktion. Der Import-Block der Testdatei braucht
`path` für den Helfer `result`.

- [ ] **Schritt 7: Commit**

```sh
git add internal/code/resolve
git commit -F ../msg.txt
```

Nachricht:

```
Resolve raw edges into edges, and drop what stays ambiguous

Two tiers of provenance, Graft's: extracted when the target is certain
-- same file, import specifier, containment, or a package selector
resolved against the one package it can mean -- and inferred when a bare
name matched exactly once across files. Several matches are dropped, not
guessed: Graft's header records that name guessing halved precision, and
one same-named symbol collecting 1040 in-edges dragged a whole backend
into every blast radius.

A package selector is the one place this goes past the reference, which
drops the case for Go and leaves cross-package structure to import edges
alone. The rule applied is Graft's own for a specifier-bearing
reference: resolve inside the named target alone, and only when exactly
one candidate is there. Candidates are functions -- a call on a type is a
conversion -- and test files are excluded, because an importer never
sees them.

Build constraints are not evaluated, so a platform pair reads as two
definitions and its call falls. Sound by the rule, and stated so nobody
files it as a bug.
```

---

### Task 6: `internal/code/store` — schreiben und lesen

**Files:**
- Create: `internal/code/store/store.go`
- Test: `internal/code/store/store_test.go`

**Interfaces:**
- Consumes: `model.Graph`, `model.Decode`.
- Produces:

```go
func Dir(root string) string        // <root>/.loomux/state/graph
func WiringPath(root string) string // <root>/.loomux/state/graph/wiring.json
func CachePath(root, name string) string // <root>/.loomux/state/graph/cache/<name>
func Write(root string, g *model.Graph) error
func Read(root string) (*model.Graph, error) // os.ErrNotExist when absent
```

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
)

func graph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 2, Extractor: "go/1", Languages: []string{"go"}, NodeCount: 2, EdgeCount: 1},
		Nodes: []model.Node{
			{ID: "a.go", Kind: model.KindFile, Path: "a.go", BodyHash: "h1", Exported: true, BodyText: "residual text"},
			{ID: "a.go#F", Kind: "function", Path: "a.go", Name: "F", BodyHash: "h2", BodyText: "body text"},
		},
		Edges: []model.Edge{{
			Source: "a.go", Target: "a.go#F",
			Relation: model.RelationContains, Confidence: model.ConfidenceExtracted,
		}},
	}
}

func TestWriteThenReadRoundTrips(t *testing.T) {
	root := t.TempDir()
	if err := store.Write(root, graph()); err != nil {
		t.Fatal(err)
	}
	got, err := store.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Nodes) != 2 || got.Meta.Extractor != "go/1" {
		t.Fatalf("got %+v", got.Meta)
	}
}

func TestWriteIsByteIdenticalOnAnUnchangedGraph(t *testing.T) {
	root := t.TempDir()
	if err := store.Write(root, graph()); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(store.WiringPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Write(root, graph()); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(store.WiringPath(root))
	if err != nil {
		t.Fatal(err)
	}
	// No timestamps anywhere: a rebuild of an unchanged tree must produce the
	// same bytes, or every build would show up as a diff.
	if string(first) != string(second) {
		t.Error("two writes of one graph produced different bytes")
	}
}

func TestWriteLeavesBodyTextOnTheFloor(t *testing.T) {
	root := t.TempDir()
	if err := store.Write(root, graph()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(store.WiringPath(root))
	if err != nil {
		t.Fatal(err)
	}
	// It is most of the file's bytes on a real repository, and the ask sidecar
	// holds it tokenized. The model's json:"-" is what enforces this; the test
	// is here so a future field rename cannot quietly undo it.
	if strings.Contains(string(b), "body text") || strings.Contains(string(b), "residual text") {
		t.Errorf("body text reached wiring.json:\n%s", b)
	}
}

func TestWriteLeavesNoTemporaryFileBehind(t *testing.T) {
	root := t.TempDir()
	if err := store.Write(root, graph()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(store.Dir(root))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("temporary file left behind: %s", e.Name())
		}
	}
}

func TestReadReportsAMissingGraphAsNotExist(t *testing.T) {
	_, err := store.Read(t.TempDir())
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v, want a wrapped os.ErrNotExist so a caller can tell absent from broken", err)
	}
}

func TestReadRefusesABrokenGraph(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.WiringPath(root), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := store.Read(root)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v, want an error that is not ErrNotExist: broken is not absent", err)
	}
}

func TestPathsSitUnderTheStateDirectory(t *testing.T) {
	root := filepath.FromSlash("/repo")
	if got := store.WiringPath(root); !strings.HasSuffix(filepath.ToSlash(got), ".loomux/state/graph/wiring.json") {
		t.Errorf("WiringPath = %q", got)
	}
	if got := store.CachePath(root, "fingerprint.json"); !strings.HasSuffix(filepath.ToSlash(got), ".loomux/state/graph/cache/fingerprint.json") {
		t.Errorf("CachePath = %q", got)
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/store/ -v
```

Erwartet: Übersetzungsfehler, das Paket gibt es nicht.

- [ ] **Schritt 3: Die Implementierung schreiben**

```go
// Package store holds the wiring graph on disk.
//
// The graph is machine state and lives under .loomux/state/graph/, which
// .gitignore already excludes. The derived sidecars sit beside it in cache/:
// they are regenerable at any time, the graph is the result.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/write.ts.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/xidus90/loomux/internal/code/model"
)

// Dir is where a repository's graph state lives.
func Dir(root string) string {
	return filepath.Join(root, ".loomux", "state", "graph")
}

// WiringPath is the graph itself.
func WiringPath(root string) string {
	return filepath.Join(Dir(root), "wiring.json")
}

// CachePath is a derived sidecar beside the graph.
func CachePath(root, name string) string {
	return filepath.Join(Dir(root), "cache", name)
}

// Write serializes the graph, atomically.
//
// Temp plus rename, with the pid in the temp name and the temp removed when the
// write fails. Graft's reasoning holds: a fixed temp name lets a concurrent run
// write the same scratch file and hand the loser a truncated graph, and a
// failed rename would leave a full-size orphan nothing ever cleans up.
//
// The caller has sorted the graph; this function adds no order of its own, so
// there is exactly one place where order is decided.
func Write(root string, g *model.Graph) error {
	if err := os.MkdirAll(Dir(root), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')

	final := WiringPath(root)
	tmp := final + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, final); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// Read loads the graph and validates it.
//
// A missing graph wraps os.ErrNotExist, so a caller can tell "never built" from
// "built and broken" -- the two want different answers, and conflating them is
// how a query starts reporting a clean tree it never looked at.
func Read(root string) (*model.Graph, error) {
	f, err := os.Open(WiringPath(root))
	if err != nil {
		return nil, fmt.Errorf("read graph: %w", err)
	}
	defer f.Close()
	return model.Decode(f)
}
```

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/code/store/ -count=1 -cover
```

Erwartet: PASS bei 100 % je Funktion. Die drei Fehlerarme von `Write`
(`MkdirAll`, `WriteFile`, `Rename`) sind nur über ein schreibgeschütztes
Verzeichnis erreichbar; `internal/testlock` baut genau diese Welt und ist dafür
gedacht — `testlock.LockDir(t, dir)`. Was dort auf einer Plattform nicht
herstellbar ist, bekommt ein `//coverage:exempt` mit Begründung.

- [ ] **Schritt 5: Commit**

```sh
git add internal/code/store
git commit -F ../msg.txt
```

Nachricht:

```
Write the wiring graph, sorted and atomically

No timestamps in the file, so a rebuild of an unchanged tree produces
the same bytes and a build is not a diff. Temp plus rename with the pid
in the temp name: a fixed name lets a concurrent run write the same
scratch file and hand the loser a truncated graph.

Read distinguishes absent from broken by wrapping os.ErrNotExist. The
two want different answers, and conflating them is how a query starts
reporting a tree it never looked at.
```

---

### Task 7: `internal/code/freshness` — die Sonde

**Files:**
- Create: `internal/code/freshness/fingerprint.go`
- Test: `internal/code/freshness/fingerprint_test.go`

**Interfaces:**
- Consumes: `sourceset.Stat`, `store.CachePath`.
- Produces:

```go
type Drift struct {
	Changed []string
	Added   []string
	Removed []string
}

func (d Drift) Clean() bool
func (d Drift) Count() int

func Write(root, extractor string, files []sourceset.SourceFile, hashes map[string]string) error
func Probe(root, extractor string) (*Drift, error) // nil Drift, nil error == unknown
```

**Dieses Paket importiert den Extraktor nicht.** Das ist die
Architekturbedingung, die G4 erlaubt, die Sonde im Hook-Pfad zu benutzen, ohne
das Abhängigkeitstor aus §3.2 der Säule-3-Spec zu brechen. Der Stempel kommt
deshalb als Zeichenkette herein und wird nicht aus `golang.Version` gelesen.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package freshness_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/sourceset"
)

// build writes a tree, records its fingerprint, and returns the root -- the
// state right after a `graph build`.
func build(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	record(t, root)
	return root
}

// record writes the fingerprint of the tree as it is now.
func record(t *testing.T, root string) {
	t.Helper()
	stat, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{}
	for _, f := range stat {
		b, err := os.ReadFile(f.Abs)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(b)
		hashes[f.Rel] = hex.EncodeToString(sum[:])
	}
	if err := freshness.Write(root, "go/1", stat, hashes); err != nil {
		t.Fatal(err)
	}
}

func TestProbeIsCleanRightAfterABuild(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || !d.Clean() {
		t.Fatalf("got %+v, want clean", d)
	}
}

func TestProbeReportsNoRecordAsUnknownAndNotAsClean(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	// Never built, or built by a version that wrote no record. A caller must
	// rebuild -- treating this as clean is how a query answers from a graph
	// that does not exist.
	if d != nil {
		t.Fatalf("got %+v, want nil for unknown", d)
	}
}

func TestProbeTreatsAForeignExtractorAsUnknown(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})

	d, err := freshness.Probe(root, "go/2")
	if err != nil {
		t.Fatal(err)
	}
	// A changed extractor matches the tree byte for byte. Without this check it
	// reports clean and queries keep answering from nodes the old extractor
	// built.
	if d != nil {
		t.Fatalf("got %+v, want nil: the record belongs to another extractor", d)
	}
}

func TestProbeSeesAChangedFile(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || len(d.Changed) != 1 || d.Changed[0] != "a.go" {
		t.Fatalf("got %+v, want a.go changed", d)
	}
	if d.Count() != 1 || d.Clean() {
		t.Errorf("Count = %d, Clean = %v", d.Count(), d.Clean())
	}
}

func TestProbeIgnoresATouchThatChangedNoBytes(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	abs := filepath.Join(root, "a.go")
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(abs, later, later); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	// The stat fast path suspects the file, the hash clears it. Without this a
	// `touch` or a checkout restoring identical bytes would cost a rebuild.
	if d == nil || !d.Clean() {
		t.Fatalf("got %+v, want clean after a touch", d)
	}
}

func TestProbeSeesAnAddedAndARemovedFile(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n", "b.go": "package a\n"})
	if err := os.Remove(filepath.Join(root, "b.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "c.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || len(d.Added) != 1 || d.Added[0] != "c.go" {
		t.Fatalf("got %+v, want c.go added", d)
	}
	if len(d.Removed) != 1 || d.Removed[0] != "b.go" {
		t.Fatalf("got %+v, want b.go removed", d)
	}
	if d.Count() != 2 {
		t.Errorf("Count = %d, want 2", d.Count())
	}
}

func TestProbeSortsEachCategory(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	for _, rel := range []string{"z.go", "m.go", "b.go"} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte("package a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"b.go", "m.go", "z.go"}
	if len(d.Added) != 3 {
		t.Fatalf("got %+v", d.Added)
	}
	for i := range want {
		if d.Added[i] != want[i] {
			t.Fatalf("got %v, want %v: a report a human reads is sorted", d.Added, want)
		}
	}
}

func TestProbeReportsAMissingRoot(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	// The record survives, the tree does not.
	if err := os.Remove(filepath.Join(root, "a.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := freshness.Probe(filepath.Join(root, "gone"), "go/1"); err == nil {
		t.Fatal("got nil, want an error for a root that is not there")
	}
}

func TestProbeTreatsABrokenRecordAsUnknown(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	if err := os.WriteFile(freshness.Path(root), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	// A sidecar is a cache. A broken one costs the next probe its fast path and
	// nothing else -- it must never be an error a query dies on.
	if d != nil {
		t.Fatalf("got %+v, want nil for a record that cannot be read", d)
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/freshness/ -v
```

Erwartet: Übersetzungsfehler, das Paket gibt es nicht.

- [ ] **Schritt 3: Die Implementierung schreiben**

```go
// Package freshness is the cheap "has the working tree moved?" probe.
//
// Every retrieval runs it, so the unchanged path has to be almost free: one
// stat per source file, no reads and no parsing. Only a file whose size or
// mtime disagrees with the record gets read and hashed, which is what keeps a
// touch -- or a checkout that restores identical bytes -- from costing a
// rebuild.
//
// It does NOT import the extractor. That is deliberate and it is what lets the
// hook path use this package under the dependency gate: the stamp arrives as a
// string. The same separation is why the file set lives in
// internal/code/sourceset -- the probe and the build must enumerate identical
// files, and two enumerations that can drift are two answers.
//
// `graph check` is the other mechanism and does something else entirely: it
// re-extracts and diffs node ids and body hashes. A stat may decide whether a
// query bothers rebuilding; it may not decide what the rebuild looks at.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/fingerprint.ts.
package freshness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/xidus90/loomux/internal/code/sourceset"
	"github.com/xidus90/loomux/internal/code/store"
)

// recordVersion is the shape of the sidecar. A record of another version is a
// record this code cannot read, which is the same as no record.
const recordVersion = 1

// print is one file's fingerprint: size, modification time in nanoseconds, and
// the hash of its bytes.
//
// mtime is an int64 and stays one all the way through JSON. Graft's field is a
// JavaScript number and so a double; here that would be a silent loss of
// precision on a field whose equality decides whether a rebuild happens.
type print struct {
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"`
	Hash  string `json:"hash"`
}

// record is the sidecar.
type record struct {
	Version   int              `json:"version"`
	Extractor string           `json:"extractor"`
	Files     map[string]print `json:"files"`
}

// Drift is what moved since the last build. Three empty slices mean nothing to
// do.
type Drift struct {
	Changed []string
	Added   []string
	Removed []string
}

// Clean reports whether nothing moved.
func (d Drift) Clean() bool { return d.Count() == 0 }

// Count is how many files moved, in any category.
func (d Drift) Count() int { return len(d.Changed) + len(d.Added) + len(d.Removed) }

// Path is where the record lives.
func Path(root string) string { return store.CachePath(root, "fingerprint.json") }

// Write records the tree as the build just saw it.
//
// hashes is keyed by the same repo-relative path sourceset uses. A file the
// build could not read is recorded with an empty hash, which puts it back on
// the slow path every time -- the only way to learn it is readable again is to
// try.
func Write(root, extractor string, files []sourceset.SourceFile, hashes map[string]string) error {
	rec := record{Version: recordVersion, Extractor: extractor, Files: map[string]print{}}
	for _, f := range files {
		rec.Files[f.Rel] = print{Size: f.Size, MTime: f.MTime, Hash: hashes[f.Rel]}
	}
	body, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(Path(root), body, 0o644)
}

// Probe diffs the working tree against the last build's record.
//
// A nil Drift with a nil error means UNKNOWN: never built, built by a version
// that wrote no record, a record that cannot be read, or a record from another
// extractor. A caller treats that as "rebuild", never as "clean".
func Probe(root, extractor string) (*Drift, error) {
	files, err := sourceset.Stat(root)
	if err != nil {
		return nil, err
	}
	rec, ok := read(root, extractor)
	if !ok {
		return nil, nil
	}

	d := &Drift{}
	seen := make(map[string]bool, len(files))
	for _, f := range files {
		seen[f.Rel] = true
		p, known := rec.Files[f.Rel]
		if !known {
			d.Added = append(d.Added, f.Rel)
			continue
		}
		if trusted(p, f) {
			continue
		}
		// Suspect. Confirm by bytes, so a touch does not cost a rebuild.
		sum, err := hashFile(f.Abs)
		if err != nil {
			// Unreadable right now: leave it to the next probe rather than
			// reporting drift a rebuild could not repair either.
			continue
		}
		if sum != p.Hash {
			d.Changed = append(d.Changed, f.Rel)
		}
	}
	for rel := range rec.Files {
		if !seen[rel] {
			d.Removed = append(d.Removed, rel)
		}
	}
	sort.Strings(d.Changed)
	sort.Strings(d.Added)
	sort.Strings(d.Removed)
	return d, nil
}

// read loads the record, or reports that there is none this code may use.
//
// A sidecar is a cache: a broken one costs the next probe its fast path and
// nothing more, so it is never an error a caller dies on.
func read(root, extractor string) (record, bool) {
	b, err := os.ReadFile(Path(root))
	if err != nil {
		return record{}, false
	}
	var rec record
	if err := json.Unmarshal(b, &rec); err != nil {
		return record{}, false
	}
	if rec.Version != recordVersion || rec.Files == nil || rec.Extractor != extractor {
		return record{}, false
	}
	return rec, true
}

// trusted reports whether a recorded print may stand for the file on disk
// without reading it.
//
// This is the PROBE's rule only. A build reads and hashes every file, every
// time: a stat may decide whether a query rebuilds, but not what the rebuild
// looks at -- otherwise `check`, which always re-extracts, could report drift
// that the `build` it recommends then refuses to repair.
//
// An empty hash means the last build never got the bytes, so it is never
// trusted.
func trusted(p print, f sourceset.SourceFile) bool {
	return p.Hash != "" && p.Size == f.Size && p.MTime == f.MTime
}

// hashFile is sha256 over a file's bytes, as full hex.
func hashFile(abs string) (string, error) {
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
```

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/code/freshness/ -count=1 -cover
```

Erwartet: PASS bei 100 % je Funktion. Der Arm „unreadable right now" in `Probe`
braucht eine Datei, die existiert und nicht gelesen werden kann —
`testlock.Lock(t, abs)` baut genau das und überspringt den Test, wo die
Plattform es nicht hergibt.

- [ ] **Schritt 5: Commit**

```sh
git add internal/code/freshness
git commit -F ../msg.txt
```

Nachricht:

```
Probe the working tree without parsing it

One stat per file, and only a file whose size or mtime disagrees gets
read and hashed -- so a touch, or a checkout restoring identical bytes,
costs nothing. Graft measures the same shape at ~3ms for 280 files.

No record means UNKNOWN, never clean. A caller rebuilds; treating it as
clean is how a query answers from a graph that was never built. A record
from another extractor counts as no record: it would match the tree byte
for byte and keep queries answering from nodes the old extractor made.

This package does not import the extractor, which is what will let the
hook path use it under the dependency gate. The stamp arrives as a
string.
```

---

### Task 8: `loomux graph build`

**Files:**
- Create: `internal/cli/graph.go`
- Modify: `internal/cli/commands.go`
- Test: `internal/cli/graph_test.go`

**Interfaces:**
- Consumes: `sourceset`, `golang`, `resolve`, `store`, `freshness`, und
  `projectRoot` aus `internal/cli/wiki.go`.
- Produces: `graphCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int`,
  Unterbefehl `build`; dazu `buildGraph(root string) (*model.Graph, buildStats, error)`
  für Task 9.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/store"
)

// repo writes a small Go repository and returns its root.
func repo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func sample() map[string]string {
	return map[string]string{
		"go.mod":            "module example.com/repo\n",
		"main.go":           "package main\n\nimport \"example.com/repo/lib\"\n\nfunc main() { lib.Run() }\n",
		"lib/lib.go":        "package lib\n\n// Run does the thing.\nfunc Run() {}\n",
		"lib/lib_test.go":   "package lib\n\nimport \"testing\"\n\nfunc TestRun(t *testing.T) { Run() }\n",
	}
}

func TestGraphBuildWritesAGraphAndReportsItsShape(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer

	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	if _, err := os.Stat(store.WiringPath(root)); err != nil {
		t.Fatalf("no graph written: %v", err)
	}
	// The count line is the acceptance view on the extractor, and the number a
	// parity file needs.
	report := out.String()
	for _, want := range []string{"nodes", "edges", "calls", "imports", "contains"} {
		if !strings.Contains(report, want) {
			t.Errorf("report %q is missing %q", report, want)
		}
	}
}

func TestGraphBuildIsByteIdenticalOnASecondRun(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer

	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	first, err := os.ReadFile(store.WiringPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	second, err := os.ReadFile(store.WiringPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Error("two builds of one tree produced different bytes")
	}
}

func TestGraphBuildWiresACallAcrossPackages(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}

	g, err := store.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	// This is the one place the port goes past Graft, which drops a package
	// selector for Go and would leave the call structure to import edges alone.
	found := false
	for _, e := range g.Edges {
		if e.Source == "main.go#main" && e.Target == "lib/lib.go#Run" && e.Relation == "calls" {
			found = true
		}
	}
	if !found {
		t.Errorf("the cross-package call is missing; edges %+v", g.Edges)
	}
}

func TestGraphBuildRecordsTheFingerprintSoCheckIsCleanAfterwards(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(store.Dir(root), "cache", "fingerprint.json")); err != nil {
		t.Fatalf("no fingerprint written: %v", err)
	}
}

func TestGraphBuildRefusesAFileItCannotParse(t *testing.T) {
	root := repo(t, map[string]string{"go.mod": "module x\n", "broken.go": "package ???\n"})
	var out, errOut bytes.Buffer

	code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut)
	// A build is explicit and must not quietly write half a graph: a file that
	// does not parse is the user's next move, not something to skip.
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "broken.go") {
		t.Errorf("stderr %q must name the file", errOut.String())
	}
}

func TestGraphWithoutASubcommandIsAUsageError(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := graphCommand(nil, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "build") || !strings.Contains(errOut.String(), "check") {
		t.Errorf("usage %q must name the subcommands", errOut.String())
	}
}

func TestGraphRejectsAnUnknownSubcommand(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"frobnicate"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
}

func TestGraphBuildRejectsAnUnknownFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--nope"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
}

func TestGraphIsInTheCommandTable(t *testing.T) {
	if _, ok := commands["graph"]; !ok {
		t.Fatal(`commands["graph"] missing`)
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/cli/ -run TestGraph -v
```

Erwartet: Übersetzungsfehler, `undefined: graphCommand`.

- [ ] **Schritt 3: Die Implementierung schreiben**

`internal/cli/graph.go`:

```go
package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
	"github.com/xidus90/loomux/internal/code/sourceset"
	"github.com/xidus90/loomux/internal/code/store"
)

// graphCommand is `loomux graph`: the two commands of stage G2a.
func graphCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		graphUsage(stderr)
		return 2
	}
	switch args[0] {
	case "build":
		return graphBuild(args[1:], stdout, stderr)
	case "check":
		return graphCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "loomux graph: unknown command %q\n", args[0])
		graphUsage(stderr)
		return 2
	}
}

func graphUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: loomux graph <command> [arguments]")
	fmt.Fprintln(w, "  build   analyse the source and write .loomux/state/graph/wiring.json")
	fmt.Fprintln(w, "  check   report whether the graph still matches the code")
}

// graphBuild writes the graph and reports what it wrote.
func graphBuild(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph build: %v\n", err)
		return 1
	}

	started := time.Now()
	g, stats, err := buildGraph(project)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph build: %v\n", err)
		return 1
	}
	if err := store.Write(project, g); err != nil {
		fmt.Fprintf(stderr, "loomux graph build: %v\n", err)
		return 1
	}
	if err := freshness.Write(project, golang.Version, stats.files, stats.hashes); err != nil {
		// The graph is on disk already, so a failed record costs the next probe
		// its fast path and nothing more.
		fmt.Fprintf(stderr, "loomux graph build: freshness record not written: %v\n", err)
	}
	fmt.Fprint(stdout, report(g, stats, time.Since(started)))
	return 0
}

// buildStats is what a build learned on the way, for the report and for the
// freshness record.
type buildStats struct {
	files    []sourceset.SourceFile
	hashes   map[string]string
	noSymbol int
}

// buildGraph reads, parses and resolves the whole tree.
//
// It reads and hashes every file, every time -- never the probe's stat fast
// path. A stat may decide whether a query rebuilds; it may not decide what the
// rebuild looks at, or `check` could report drift that the `build` it
// recommends refuses to repair.
func buildGraph(root string) (*model.Graph, buildStats, error) {
	files, err := sourceset.Stat(root)
	if err != nil {
		return nil, buildStats{}, err
	}
	stats := buildStats{files: files, hashes: map[string]string{}}

	var results []golang.Result
	for _, f := range files {
		b, err := os.ReadFile(f.Abs)
		if err != nil {
			return nil, buildStats{}, err
		}
		sum := sha256.Sum256(b)
		stats.hashes[f.Rel] = hex.EncodeToString(sum[:])

		r, err := golang.File(f.Rel, string(b))
		if err != nil {
			return nil, buildStats{}, err
		}
		if len(r.Nodes) == 1 {
			stats.noSymbol++
		}
		results = append(results, r)
	}

	mods, err := resolve.Modules(root, goModPaths(root))
	if err != nil {
		return nil, buildStats{}, err
	}
	return resolve.Graph(results, mods), stats, nil
}

// goModPaths are the repo-relative go.mod files, found by walking for them --
// sourceset lists Go sources and a go.mod is not one.
func goModPaths(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// report is the count line after a build: what the extractor found, so a
// reader can see at a glance whether it found the right kind of thing.
func report(g *model.Graph, stats buildStats, took time.Duration) string {
	byRelation := map[model.Relation]int{}
	unresolved := 0
	nodes := map[model.NodeID]bool{}
	for _, n := range g.Nodes {
		nodes[n.ID] = true
	}
	for _, e := range g.Edges {
		byRelation[e.Relation]++
		if !nodes[e.Target] {
			unresolved++
		}
	}
	return fmt.Sprintf(
		"%d files, %d nodes, %d edges (%d contains, %d calls, %d imports)\n"+
			"%d unresolved import targets, %d files without a symbol, %s\n",
		len(stats.files), len(g.Nodes), len(g.Edges),
		byRelation[model.RelationContains], byRelation[model.RelationCalls],
		byRelation[model.RelationImports],
		unresolved, stats.noSymbol, took.Round(time.Millisecond),
	)
}
```

Der Import-Block braucht zusätzlich `io/fs`, `path/filepath` und `strings` für
`goModPaths`. `graphCheck` kommt in Task 9; bis dahin ein Rumpf:

```go
// graphCheck is Task 9.
func graphCheck(args []string, stdout, stderr io.Writer) int { return 2 }
```

In `internal/cli/commands.go` eine Zeile, alphabetisch eingeordnet:

```go
	"graph":     graphCommand,
```

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/cli/ -run TestGraph -count=1
```

Erwartet: PASS. Der Test `TestGraphBuildWiresACallAcrossPackages` ist der, der
Task 4 und 5 zusammen prüft — schlägt er fehl, liegt der Fehler dort und nicht
in der Verdrahtung.

- [ ] **Schritt 5: Commit**

```sh
git add internal/cli/graph.go internal/cli/graph_test.go internal/cli/commands.go
git commit -F ../msg.txt
```

Nachricht:

```
Add `loomux graph build`

It reads and hashes every file, every time, and never the probe's stat
fast path: a stat may decide whether a query rebuilds, but not what the
rebuild looks at -- otherwise check reports drift the build it
recommends cannot repair.

A file that does not parse fails the build rather than being skipped. A
build is explicit, and half a graph written in silence is worse than a
named error.

The count line after a build is the acceptance view on the extractor:
nodes, edges per relation, unresolved import targets, files without a
symbol.
```

---

### Task 9: `loomux graph check`

**Files:**
- Modify: `internal/cli/graph.go` (den Rumpf aus Task 8)
- Test: `internal/cli/graph_test.go` (anhängen)

**Interfaces:**
- Consumes: `buildGraph` aus Task 8, `store.Read`.
- Produces: `graphCheck` mit den Exitcodes aus Abschnitt 9 der Spec.

`check` prüft **nicht** die Abdrücke. Es extrahiert neu und diffed über `id` und
`body_hash`.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
func TestGraphCheckIsCleanRightAfterABuild(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	if code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, want 0; stdout %q", code, out.String())
	}
}

func TestGraphCheckIsCleanAfterATouch(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(filepath.Join(root, "lib", "lib.go"), later, later); err != nil {
		t.Fatal(err)
	}

	// check re-extracts, so it sees content and not a timestamp.
	if code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, want 0 after a touch", code)
	}
}

func TestGraphCheckReportsAChangedBody(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()
	body := "package lib\n\n// Run does the thing.\nfunc Run() { println(\"now it does something\") }\n"
	if err := os.WriteFile(filepath.Join(root, "lib", "lib.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(out.String(), "lib/lib.go#Run") {
		t.Errorf("report %q must name the changed node", out.String())
	}
}

func TestGraphCheckIgnoresARewordedDocComment(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	body := "package lib\n\n// Run is documented differently now.\nfunc Run() {}\n"
	if err := os.WriteFile(filepath.Join(root, "lib", "lib.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	// The symbol's hash runs over Pos()..End(), which excludes the comment.
	// The FILE node hashes the whole file, so the file is reported changed --
	// but no symbol is, and that is the distinction worth having.
	code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut)
	if code != 1 {
		t.Fatalf("exit %d, want 1: the file node's hash covers the comment", code)
	}
	if strings.Contains(out.String(), "lib/lib.go#Run") {
		t.Errorf("no symbol changed; report %q must not name Run", out.String())
	}
}

func TestGraphCheckReportsAnAddedAndARemovedSymbol(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()
	if err := os.WriteFile(filepath.Join(root, "lib", "lib.go"), []byte("package lib\n\nfunc Other() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	report := out.String()
	if !strings.Contains(report, "lib/lib.go#Other") || !strings.Contains(report, "lib/lib.go#Run") {
		t.Errorf("report %q must name both the added and the removed symbol", report)
	}
}

func TestGraphCheckReportsAMissingGraphAndPointsAtBuild(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer

	code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut)
	// Graft has no separate code for this, and the pillar-3 spec asks only that
	// it be reported cleanly as not initialised. A third code would be an
	// invention.
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(out.String(), "graph build") {
		t.Errorf("report %q must point at the command that fixes it", out.String())
	}
}

func TestGraphCheckRefusesAGraphFromAnotherExtractor(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	g, err := store.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	g.Meta.Extractor = "go/0"
	if err := store.Write(root, g); err != nil {
		t.Fatal(err)
	}
	out.Reset()

	code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut)
	// Foreign, not stale: its nodes say nothing about this code, so comparing
	// them one by one would be noise.
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(out.String(), "extractor") {
		t.Errorf("report %q must say the graph is from another extractor", out.String())
	}
}

func TestGraphCheckJSONCarriesTheThreeCategories(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	if code := graphCommand([]string{"check", "--root", root, "--json"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	var got struct {
		OK      bool     `json:"ok"`
		Missing bool     `json:"missing"`
		Added   []string `json:"added"`
		Removed []string `json:"removed"`
		Changed []string `json:"changed"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out.String())
	}
	if !got.OK || got.Missing {
		t.Errorf("got %+v, want ok and not missing", got)
	}
}
```

Der Import-Block von `graph_test.go` braucht zusätzlich `encoding/json` und
`time`.

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/cli/ -run TestGraphCheck -v
```

Erwartet: FAIL, jeder Fall mit `exit 2, want 0` oder `want 1` — der Rumpf gibt
immer 2.

- [ ] **Schritt 3: Die Implementierung schreiben**

Den Rumpf in `internal/cli/graph.go` ersetzen:

```go
// checkResult is what `graph check` found.
type checkResult struct {
	OK      bool     `json:"ok"`
	Missing bool     `json:"missing"`
	Foreign string   `json:"foreign,omitempty"`
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Changed []string `json:"changed"`
}

// graphCheck re-extracts the tree and diffs it against the written graph.
//
// It does NOT read the freshness record. That sidecar answers "should a query
// bother rebuilding"; this command answers "does the graph still describe the
// code", and the only honest way to answer it is to extract again. It is also
// why a bare `touch` leaves this command at 0.
func graphCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	asJSON := fs.Bool("json", false, "write the drift as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph check: %v\n", err)
		return 1
	}

	res, err := check(project)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph check: %v\n", err)
		return 1
	}
	if *asJSON {
		body, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "loomux graph check: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", body)
	} else {
		fmt.Fprint(stdout, checkReport(res))
	}
	if res.OK {
		return 0
	}
	return 1
}

// check compares a fresh extraction against the graph on disk.
func check(root string) (checkResult, error) {
	written, err := store.Read(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return checkResult{Missing: true}, nil
		}
		return checkResult{}, err
	}
	// The stamp decides before a single node is compared: a graph from another
	// extractor is not stale, it is foreign, and diffing it node by node would
	// report the whole repository.
	if written.Meta.Extractor != golang.Version {
		return checkResult{Foreign: written.Meta.Extractor}, nil
	}

	fresh, _, err := buildGraph(root)
	if err != nil {
		return checkResult{}, err
	}

	was := map[model.NodeID]string{}
	for _, n := range written.Nodes {
		was[n.ID] = n.BodyHash
	}
	res := checkResult{}
	now := map[model.NodeID]bool{}
	for _, n := range fresh.Nodes {
		now[n.ID] = true
		hash, known := was[n.ID]
		switch {
		case !known:
			res.Added = append(res.Added, string(n.ID))
		case hash != n.BodyHash:
			res.Changed = append(res.Changed, string(n.ID))
		}
	}
	for id := range was {
		if !now[id] {
			res.Removed = append(res.Removed, string(id))
		}
	}
	sort.Strings(res.Added)
	sort.Strings(res.Removed)
	sort.Strings(res.Changed)
	res.OK = len(res.Added)+len(res.Removed)+len(res.Changed) == 0
	return res, nil
}

// checkReport is the human form.
func checkReport(res checkResult) string {
	switch {
	case res.Missing:
		return "loomux graph check: NO GRAPH\n\nNothing built yet. Run `loomux graph build` first.\n"
	case res.Foreign != "":
		return fmt.Sprintf(
			"loomux graph check: FOREIGN GRAPH\n\nThe graph was written by extractor %q, this binary is %q.\nRun `loomux graph build`.\n",
			res.Foreign, golang.Version)
	case res.OK:
		return "loomux graph check: OK\n"
	}
	out := "loomux graph check: DRIFT\n\n"
	for _, group := range []struct {
		label string
		ids   []string
	}{
		{"added", res.Added}, {"removed", res.Removed}, {"changed", res.Changed},
	} {
		for _, id := range group.ids {
			out += fmt.Sprintf("  %-8s %s\n", group.label, id)
		}
	}
	return out + "\nRun `loomux graph build`.\n"
}
```

Der Import-Block braucht zusätzlich `encoding/json` und `errors`.

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/cli/ -count=1
```

Erwartet: PASS, auch die vorhandenen CLI-Tests — `commands.go` hat eine Zeile
mehr, und ein Test, der die Befehlsliste festnagelt, ist mitzuziehen:

```sh
grep -rn "unknown command\|Usage: loomux" internal/cli/cli_test.go | head
```

- [ ] **Schritt 5: Commit**

```sh
git add internal/cli
git commit -F ../msg.txt
```

Nachricht:

```
Add `loomux graph check`

It re-extracts the tree and diffs node ids and body hashes against the
written graph. It deliberately does not read the freshness record: that
sidecar answers whether a query should bother rebuilding, and this
command answers whether the graph still describes the code. Only a fresh
extraction answers the second one, which is also why a bare touch leaves
this at exit 0.

The extractor stamp decides before a single node is compared. A graph
from another extractor is foreign rather than stale, and diffing it node
by node would report the whole repository.

Drift and a missing graph both exit 1, as in the reference; a usage
error exits 2. A third code for "never built" would be an invention --
the message carries that difference.
```

---

### Task 10: Dokumentation und Messungen

**Files:**
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md`
- Modify: `docs/en/architecture.md`, `docs/de/architecture.md`
- Modify: `docs/en/configuration.md`, `docs/de/configuration.md`
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`
- Modify: `README.md`, `README.de.md`

**Interfaces:**
- Consumes: die fertigen Befehle aus Task 8 und 9.
- Produces: nichts im Code.

- [ ] **Schritt 1: Die Form der vorhandenen Dateien lesen**

```sh
sed -n '1,60p' docs/en/cli-reference.md
sed -n '1,40p' docs/en/benchmarks.md
grep -n "stage G1" -A 4 README.md
```

Die neuen Abschnitte folgen der vorhandenen Form; nichts wird umgebaut.

- [ ] **Schritt 2: `cli-reference.md` in beiden Sprachen ergänzen**

Je Befehl: Aufruf, Flaggen, Exitcodes, ein Satz dazu, was er tut. Für `check`
gehört der Unterschied dazu, der sonst rückgefragt wird: es extrahiert neu und
liest die Frischeakte nicht, ein `touch` ist deshalb kein Befund.

- [ ] **Schritt 3: `architecture.md` in beiden Sprachen ergänzen**

Die fünf neuen Pakete mit je einem Satz, und die Ablage:

```
.loomux/state/graph/wiring.json
.loomux/state/graph/cache/fingerprint.json
```

Dazu der Satz, der die Abhängigkeitsrichtung festhält: `freshness` importiert
den Extraktor nicht, damit der Hook-Pfad es später benutzen kann.

- [ ] **Schritt 4: `configuration.md` in beiden Sprachen ergänzen**

Ein kurzer Abschnitt, dass es **keinen** `[graph]`-Abschnitt gibt und warum:
die Sprachen ergeben sich aus dem Extraktor, die Einschränkung eines Baus ist
eine Flagge, und was ein Bau einschränkt, gehört nicht in die Konfiguration des
Repos, das indiziert wird. Ohne diesen Absatz kommt die Frage wieder.

- [ ] **Schritt 5: Die vier Messungen fahren und eintragen**

Jede Zeile in `docs/{en,de}/benchmarks.md` mit Datum und Uhrzeit, Gegenstand,
Basis gegen Änderung, kalt und warm. **Kalt heißt: frischer Prozess, und für
`build` zusätzlich kein Graph und keine Frischeakte auf der Platte.**

Die offene G1-Messung zuerst — sie ist die Schuld, die G2 übernimmt:

```sh
# Die kalte Zahl zur Dangling-Masse. Die warme steht schon: ~9 ms gepoolt
# gegen ~4,5 s je Dangling-Knoten auf 20k Knoten, 2026-09-16.
# Kalt = ein Prozess je Messung, nicht zehn Runden in einem.
go test ./internal/code/pagerank/ -run XXX -bench BenchmarkDangling -benchtime 1x -count 1
```

Gibt es die Bank noch nicht, gehört sie zu diesem Schritt: ein Graph mit 20.000
Knoten, davon ein wachsender Anteil ohne ausgehende Kante, einmal mit gepoolter
Rückführung und einmal je Knoten.

Dann die drei eigenen:

```sh
# build, warm: dreimal hintereinander, der Median zählt.
go run ./cmd/loomux graph build --root .

# build, kalt: Graph und Frischeakte weg, frischer Prozess.
rm -rf .loomux/state/graph && go run ./cmd/loomux graph build --root .

# Die Sonde allein, gegen Grafts ~3 ms für 280 Dateien.
go test ./internal/code/freshness/ -bench BenchmarkProbe -benchtime 10x
```

Für die Sonde ist eine Bank zu schreiben, die einen gebauten Baum voraussetzt
und nur `Probe` misst.

- [ ] **Schritt 6: Die READMEs nachziehen**

`README.md` und `README.de.md`: der Absatz zum Stand sagt jetzt, dass Extraktor,
Frischeprüfung und die beiden Befehle stehen, dass die Abfrage G2b ist, und
nennt die gemessene Bauzeit. Die Fahrplanzeilen zu `graph build`, `graph check`
und dem Go-Extraktor wandern von „Specified" auf den erreichten Stand.

- [ ] **Schritt 7: Commit**

```sh
git add docs README.md README.de.md
git commit -F ../msg.txt
```

Nachricht:

```
Document the graph commands and measure them

The CLI reference carries build and check with their exit codes, and the
one thing about check that would otherwise be asked twice: it
re-extracts rather than reading the freshness record, so a touch is not
a finding.

Configuration states that there is no [graph] section and why, because
the handover from G1 listed one as owed and the answer is that it is not
needed.

Benchmarks carry the cold dangling figure G1 left open, plus build cold
and warm, the probe alone, against the reference's ~3ms for 280 files.
```

---

### Task 11: Das ganze Tor und die Mutationsrunde

**Files:**
- Create: `docs/.superpowers/parity/code-g2a.md`

**Interfaces:**
- Consumes: alles Vorige.
- Produces: die Paritätsakte der Stufe.

- [ ] **Schritt 1: Das Tor ganz fahren**

```sh
gofmt -l cmd internal
go vet ./...
go test ./... -count=1 -covermode=set -coverpkg=github.com/xidus90/loomux/... -coverprofile=coverage.out
go run ./cmd/loomux dev covergate --profile coverage.out
```

Erwartet: `gofmt` schweigt, `vet` schweigt, Tests grün, `covergate` ohne
Befund. Eine Funktion unter 100 % ist ein fehlender Test, nicht ein Kandidat
für `//coverage:exempt` — die Ausnahme ist für das, was eine Plattform nicht
herstellen kann, und trägt dann ihre Begründung.

- [ ] **Schritt 2: Beide Binaries bauen**

```sh
go build -o bin/loomux.exe ./cmd/loomux
```

- [ ] **Schritt 3: Die Mutationsrunde fahren**

```sh
go run ./cmd/loomux dev mutants internal/code/extract/golang internal/code/resolve internal/code/freshness
```

Der Extraktor ist das größte Paket der Stufe; die Runde dauert. Sie läuft
einmal, dann wird nachgezogen, dann läuft sie erneut über alle drei Pakete in
**einem** Befehl — sonst sind die Zahlen nicht vergleichbar.

- [ ] **Schritt 4: Die Paritätsakte schreiben**

`docs/.superpowers/parity/code-g2a.md`, nach dem Muster von
`docs/.superpowers/parity/code-g1.md`: Kopf mit Datum, Kompilat und Zahlen,
dann eine Tabelle mit je einer Zeile pro Überlebendem — Ort, Mutation, Ausgang,
Verfügung, Freigabe. Eine Verfügung sagt entweder „fehlender Test" und nennt den
nachgezogenen Test, oder „äquivalent" und rechnet vor, warum kein Test den
Mutanten töten kann.

Dazu ein Abschnitt „Verfügungen dieser Stufe, die keine Mutation betreffen" für
die drei Entscheidungen, die die Lesart tragen:

- **Der Paketselektor.** Die eine Erweiterung gegenüber Grafts Go-Pfad, mit dem
  Verweis auf 6.2 der Spec.
- **Baubedingungen.** Ein Plattformpaar ist mehrdeutig und fällt; der Graph
  sieht auf jedem Betriebssystem gleich aus.
- **Die Typsignatur.** `type Cache struct` statt Grafts `Cache`, mit dem
  Verweis auf 5.2.1.

- [ ] **Schritt 5: Was G2a offen lässt, festhalten**

Am Ende der Akte, damit G2b und G4 es nicht suchen müssen:

- die Abfrage: `lexicon`, `ask`, die Beiakte — G2b;
- `--lsp` gegen `gopls` — die optionale Folgestufe;
- die Normalisierung von `Depth` und `Direction` — gehört zu `graph callers`,
  G4;
- die `check`-Lanes `graph-freshness` und `blast-audit` und die
  Hook-Anbindung — G4.

- [ ] **Schritt 6: Commit**

```sh
git add docs/.superpowers/parity/code-g2a.md
git commit -F ../msg.txt
```

Nachricht:

```
Sign off the G2a mutation round

Every survivor carries a ruling: a missing test names the test that was
written for it, an equivalent mutant carries the algebra showing no test
could kill it.

Three rulings follow from no mutation and are recorded because they
carry how the table reads: the package selector as the one extension
past the reference, build constraints going unevaluated so the graph
looks the same on every platform, and the type signature following
Graft's comment rather than its code.
```

---

## Self-Review

Gegen die Spec gelesen, Abschnitt für Abschnitt:

| Spec | Task |
|---|---|
| 4 Schema 2, 4.1 Validate | 1 |
| 5.1 Kinds, Dateiknoten | 3 |
| 5.2 Felder, 5.2.1 Typsignatur | 3 |
| 5.3 ID und Ordinal | 3 |
| 5.4 Rohkanten, drei Relationen | 4 |
| 6.1 Aufrufe, Bindungsformen, Baubedingungen | 4, 5 |
| 6.2 Paketselektor, Beschattung, Kandidaten | 4, 5 |
| 6.3 Importe, `go.mod`, Repräsentant | 5 |
| 7 Ablage, Sortierung, Determinismus | 6 |
| 7.1 Dateimenge | 2 |
| 7.2 Extraktor-Stempel | 1 (Feld), 5 (gesetzt), 7 (verglichen) |
| 7.3 kein `[graph]` | 10 |
| 8.1 Sonde | 7 |
| 8.2 `check` | 9 |
| 9 Befehle, Exitcodes | 8, 9 |
| 11 Startzeit, kein `init()` | Global Constraints |
| 12.1 Tor | 11 |
| 12.2 Golden-Files, Testbaum | 3, 4 |
| 12.4 Mutationsrunde | 11 |
| 13 Doku und Messungen | 10 |

**Nicht in G2a und richtig so:** Abschnitt 10 der Spec (`lexicon`, `ask`) ist
G2b; Abschnitt 14 (`--lsp`) ist die optionale Folgestufe; Abschnitt 12.3 (die
portierten `ask`-Vektoren) gehört zu G2b.

**Eine Entscheidung, die eine Messung umdrehen kann:** Task 2 läuft das
Dateisystem ab, statt Git nach der Menge zu fragen. Abschnitt 7.1 der Spec trägt
diese Abweichung samt Begründung — ein Unterprozess je Sondenaufruf kostet mehr
als die Sonde selbst — und macht sie ausdrücklich von der Messung in Task 10
abhängig: steht die Sonde deutlich über 3 ms, wird auf `git ls-files`
umgestellt. Wer diesen Plan ausführt, baut den Verzeichnislauf und trägt die
Zahl ein; er entscheidet die Frage nicht neu.
