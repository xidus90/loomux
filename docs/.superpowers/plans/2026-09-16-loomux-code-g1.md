# loomux G1 — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `internal/code/{model,pagerank,blast}` — das Lesemodell des
Code-Graphen und die beiden reinen Rechner darauf, belegt an den Testvektoren
von Graft `1e352a3`.

**Architecture:** Drei neue Pakete ohne Fremdabhängigkeit. `model` hält Knoten,
Kanten und den Dekoder für `wiring.json`. `pagerank` baut aus einem Graphen
einmal eine ungerichtete Topologie (`Prepare`) und rechnet darauf
Personalized PageRank (`Rank`). `blast` baut die gerichtete Adjazenz (`New`)
und läuft sie mit Tiefenlimit ab (`Reach`). Beide Rechner arbeiten auf
Index-Slices statt auf Maps, damit die Summationsreihenfolge und die Ausgabe
deterministisch sind.

**Tech Stack:** Go (Toolchain 1.27.0 im Einsatz, `go.mod` bleibt bei
`go 1.25.0`), Standardbibliothek allein — `encoding/json`, `sort`, `math`.
Keine neue Abhängigkeit, `go.mod` und `go.sum` werden nicht angefasst.

**Spec:** `docs/.superpowers/specs/2026-09-16-loomux-code-g1-delta.md`
(berichtigt und verengt `2026-09-14-loomux-code-graph-design.md`; bei
Widerspruch gilt das Delta, danach diese Datei).

**Referenz:** `trailhq/Graft` @ `1e352a3` (MIT). Der Klon lebt im
Sitzungs-Scratchpad und ist Wegwerf; alles, was aus ihm gebraucht wird, steht
als Literal in diesem Plan. Wer den Klon nicht hat, braucht ihn nicht.

## Global Constraints

- **Arbeitsort:** Worktree `C:\Users\micro\Documents\#GIT\loomux-code-g1`,
  Branch `code-g1`, abgezweigt von `sdd-1b-1` `636138b`. Die Sitzung startet
  dort. Die Schreibschranke braucht keinen eigenen Registry-Eintrag
  (`docs/.superpowers/parity/schranke-worktrees.md`, freigegeben 2026-09-15);
  ein `go build -o bin/loomux.exe ./cmd/loomux` im Worktree ist einmal nötig,
  damit die Hooks dort ein Binary finden.
- **Diese Dateien fasst G1 nicht an:** `internal/cli/**`, `.loomux/config.toml`,
  `docs/en/**`, `docs/de/**`, `README.md`, `README.de.md`, `go.mod`, `go.sum`.
  Sie sind die Konfliktfläche der parallel laufenden Stufe 1b-1. Die
  CLI-Verdrahtung und die Dokumentation schuldet G2, nicht G1 — das ist eine
  bewusste Abweichung von der README-Pflicht der `AGENTS.md`, festgehalten in
  §2 des Deltas.
- **Modulpfad** `github.com/xidus90/loomux`. Neue Pakete liegen unter
  `internal/code/`.
- **Sprache:** Code, Bezeichner, Kommentare, Fehlermeldungen und
  Commit-Nachrichten englisch. Dieser Plan, das Delta und
  `docs/.superpowers/parity/` deutsch.
- **Kein `init()`, keine Paketvariable, die Daten parst.** Geladen wird beim
  ersten Gebrauch.
- **Coverage 100 % je Funktion.** Eine Ausnahme nur mit
  `//coverage:exempt <reason>` direkt über `func`. Das Tor
  (`.githooks/pre-commit`) fährt gofmt, `go vet`,
  `go test ./... -count=1 -covermode=set -coverpkg=./...`, `dev covergate` und
  baut danach das Pilot-Binary.
- **Commits:** der Mensch ist Autor und Committer, kein Modell und kein Agent
  wird genannt. Mehrzeilige Nachrichten über eine Datei und `git commit -F`,
  nie über ein Heredoc. Vor jedem Commit Zweig und HEAD lesen. Niemand außer
  einem Menschen pusht.
- **Determinismus vor Bequemlichkeit.** Keine Map-Iteration, deren Reihenfolge
  in ein Ergebnis oder in eine Gleitkommasumme einfließt.

---

## File Structure

| Datei | Verantwortung |
|---|---|
| `internal/code/model/graph.go` | `NodeID`, `Node`, `Edge`, `Relation`, `Confidence`, `Graph`, `Meta`; `IsWalk`; `SymbolsInFile` |
| `internal/code/model/decode.go` | `Decode` aus einem `io.Reader`, `Validate` |
| `internal/code/model/graph_test.go`, `decode_test.go` | dazu die Tests |
| `internal/code/model/testdata/wiring.json` | ein Graph mit allen sechs Relationen, jeder Konfidenz, einem unaufgelösten Importziel und einer ID mit Dedup-Ordinal |
| `internal/code/pagerank/topology.go` | `Topology`, `Prepare` |
| `internal/code/pagerank/rank.go` | `Options`, `Scored`, `Rank` |
| `internal/code/pagerank/*_test.go` | die neun portierten Vektoren |
| `internal/code/blast/index.go` | `Index`, `New`, `Direction`, `Depth`, `Hit` |
| `internal/code/blast/reach.go` | `Reach` |
| `internal/code/blast/*_test.go` | die zwölf portierten Vektoren |
| `docs/.superpowers/parity/code-g1.md` | Mutationsrunde: jeder Überlebende mit Verfügung |

---

### Task 0: Arbeitsort prüfen

**Files:** keine.

- [ ] **Schritt 1: Ort und Basis lesen**

```sh
cd "C:/Users/micro/Documents/#GIT/loomux-code-g1"
git rev-parse --abbrev-ref HEAD
git log -1 --format='%h %an <%ae>'
```

Erwartet: `code-g1`, HEAD auf dem Delta-Commit, Autor der Mensch.

- [ ] **Schritt 2: Binary für die Hooks bauen**

```sh
go build -o bin/loomux.exe ./cmd/loomux
```

- [ ] **Schritt 3: Tor einmal leer fahren**

```sh
go test ./... -count=1
```

Erwartet: alles grün, bevor eine Zeile G1 dazukommt. Ist es das nicht, liegt
der Fehler in `sdd-1b-1` und nicht in dieser Stufe — melden, nicht reparieren.

---

### Task 1: `internal/code/model` — Typen

**Files:**
- Create: `internal/code/model/graph.go`
- Test: `internal/code/model/graph_test.go`

**Interfaces:**
- Consumes: nichts.
- Produces: `model.NodeID`, `model.Node`, `model.Edge`, `model.Relation` mit
  den sechs Werten, `model.Confidence` mit den vier Werten, `model.Graph`,
  `model.Meta`, `func (Relation) IsWalk() bool`,
  `func SymbolsInFile(g *Graph, path string) []NodeID`.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package model_test

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
)

func TestRelationIsWalk(t *testing.T) {
	walk := []model.Relation{
		model.RelationCalls,
		model.RelationReferences,
		model.RelationImports,
		model.RelationImplements,
		model.RelationExtends,
	}
	for _, r := range walk {
		if !r.IsWalk() {
			t.Errorf("relation %q must be walkable", r)
		}
	}
	// A file contains every symbol defined in it. Walking that edge would make
	// each sibling a neighbour and turn the file into a hub the graph never
	// earned.
	if model.RelationContains.IsWalk() {
		t.Error(`relation "contains" must never be walkable`)
	}
	if model.Relation("nonsense").IsWalk() {
		t.Error("an unknown relation must not be walkable")
	}
}

func TestSymbolsInFile(t *testing.T) {
	g := &model.Graph{Nodes: []model.Node{
		{ID: "src/a.ts", Kind: model.KindFile, Path: "src/a.ts"},
		{ID: "src/a.ts#helper", Kind: "function", Path: "src/a.ts"},
		{ID: "src/a.ts#Other.m", Kind: "method", Path: "src/a.ts"},
		{ID: "src/b.ts#useB", Kind: "function", Path: "src/b.ts"},
	}}

	got := model.SymbolsInFile(g, "src/a.ts")
	want := []model.NodeID{"src/a.ts#helper", "src/a.ts#Other.m"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := model.SymbolsInFile(g, "src/none.ts"); got != nil {
		t.Errorf("got %v, want nil for a path with no symbols", got)
	}
	if got := model.SymbolsInFile(nil, "src/a.ts"); got != nil {
		t.Errorf("got %v, want nil for a nil graph", got)
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/model/ -run 'TestRelationIsWalk|TestSymbolsInFile' -v
```

Erwartet: Übersetzungsfehler, `undefined: model.RelationCalls`.

- [ ] **Schritt 3: Die minimale Implementierung schreiben**

```go
// Package model is the read model of the code graph: the nodes and edges an
// extractor writes to wiring.json, and the questions about them that need no
// walk.
//
// A model of its own rather than decoding into a map: a field the extractor
// renames would otherwise reach a caller unnoticed.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/types.ts.
package model

// NodeID identifies a node. It is path-scoped -- "src/cache.ts#Cache.get" --
// and opaque: the span is deliberately not part of it, so a definition moving
// down a file keeps its identity, and a duplicated definition can carry an
// ordinal ("Cache.get~2").
type NodeID string

// Kind is what a node represents, in LSP vocabulary.
type Kind string

// KindFile marks the node that stands for a whole file. It is the only kind
// this package tells apart.
const KindFile Kind = "file"

// Span is a line range in the form "L165-L222".
type Span string

// Relation is what an edge means.
type Relation string

// The six relations of the schema.
const (
	RelationContains   Relation = "contains"   // file -> symbol, class -> method
	RelationCalls      Relation = "calls"      // function -> function it invokes
	RelationImports    Relation = "imports"    // file -> module
	RelationReferences Relation = "references" // symbol -> symbol it names
	RelationImplements Relation = "implements" // class -> interface
	RelationExtends    Relation = "extends"    // class -> base class
)

// Confidence says how sure the extractor is that an edge is true, strongest
// first. G1 reads it and judges nothing by it.
type Confidence string

// The four confidences of the schema.
const (
	ConfidenceLSPResolved Confidence = "lsp_resolved"
	ConfidenceLSPDispatch Confidence = "lsp_dispatch"
	ConfidenceExtracted   Confidence = "extracted"
	ConfidenceInferred    Confidence = "inferred"
)

// IsWalk reports whether an edge of this relation carries dependency meaning
// for a walk or a rank.
//
// "contains" is excluded on purpose: a file contains every symbol defined in
// it, so walking that edge would make every same-file symbol a neighbour and
// let the file act as a hub that floods the walk.
func (r Relation) IsWalk() bool {
	switch r {
	case RelationCalls, RelationReferences, RelationImports,
		RelationImplements, RelationExtends:
		return true
	default:
		return false
	}
}

// Node is one definition: a file, or a symbol inside one.
type Node struct {
	ID        NodeID `json:"id"`
	Name      string `json:"name"`
	Kind      Kind   `json:"kind"`
	Owner     string `json:"owner,omitempty"`
	Path      string `json:"path"`
	Span      Span   `json:"span"`
	Signature string `json:"signature"`
	Exported  bool   `json:"exported"`
}

// Edge wires two nodes.
//
// Target is not always a node: an unresolved import names the module itself
// ("npm:lodash"). A rank drops such an edge, a walk keeps it as a hit without
// a node.
type Edge struct {
	Source     NodeID     `json:"source"`
	Target     NodeID     `json:"target"`
	Relation   Relation   `json:"relation"`
	Confidence Confidence `json:"confidence"`
}

// Meta is what the writer says about the graph it wrote.
type Meta struct {
	Version   int      `json:"version"`
	NodeCount int      `json:"nodeCount"`
	EdgeCount int      `json:"edgeCount"`
	Languages []string `json:"languages"`
}

// Graph is a whole wiring.json.
type Graph struct {
	Meta  Meta   `json:"meta"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// SymbolsInFile returns the symbols defined in the file at path, in graph
// order.
//
// Membership is path equality, not the "contains" edge: an extractor writes
// the path on every node it emits, while "contains" is a courtesy the schema
// allows and no consumer may require.
func SymbolsInFile(g *Graph, path string) []NodeID {
	if g == nil {
		return nil
	}
	var symbols []NodeID
	for _, n := range g.Nodes {
		if n.Kind != KindFile && n.Path == path {
			symbols = append(symbols, n.ID)
		}
	}
	return symbols
}
```

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/code/model/ -run 'TestRelationIsWalk|TestSymbolsInFile' -v
```

Erwartet: PASS.

- [ ] **Schritt 5: Committen**

```sh
git add internal/code/model/graph.go internal/code/model/graph_test.go
git commit -m "Give the code graph a read model of its own"
```

---

### Task 2: `internal/code/model` — Dekoder und Prüfung

**Files:**
- Create: `internal/code/model/decode.go`, `internal/code/model/testdata/wiring.json`
- Test: `internal/code/model/decode_test.go`

**Interfaces:**
- Consumes: alles aus Task 1.
- Produces: `func Decode(r io.Reader) (*Graph, error)`,
  `func (g *Graph) Validate() error`.

- [ ] **Schritt 1: Die Beispieldatei schreiben**

`internal/code/model/testdata/wiring.json` — ein Graph, der jede Eigenheit des
Schemas einmal zeigt: alle sechs Relationen, jede Konfidenz, ein unaufgelöstes
Importziel, eine ID mit Dedup-Ordinal, eine Datei ohne Signatur.

```json
{
  "meta": { "version": 1, "nodeCount": 6, "edgeCount": 6, "languages": ["ts", "go"] },
  "nodes": [
    { "id": "src/cache.ts", "name": "cache.ts", "kind": "file", "path": "src/cache.ts", "span": "L1-L90", "signature": null, "exported": false },
    { "id": "src/cache.ts#Cache", "name": "Cache", "kind": "class", "path": "src/cache.ts", "span": "L10-L80", "signature": "class Cache", "exported": true },
    { "id": "src/cache.ts#Cache.get", "name": "get", "kind": "method", "owner": "Cache", "path": "src/cache.ts", "span": "L20-L40", "signature": "get(k: string): number", "exported": true },
    { "id": "src/cache.ts#Cache~2.get", "name": "get", "kind": "method", "owner": "Cache~2", "path": "src/cache.ts", "span": "L50-L70", "signature": "get(k: string): number", "exported": false },
    { "id": "src/store.ts#Store", "name": "Store", "kind": "interface", "path": "src/store.ts", "span": "L1-L12", "signature": "interface Store", "exported": true },
    { "id": "pkg/hash.go#Hash", "name": "Hash", "kind": "function", "path": "pkg/hash.go", "span": "L3-L9", "signature": "func Hash(b []byte) uint64", "exported": true }
  ],
  "edges": [
    { "source": "src/cache.ts", "target": "src/cache.ts#Cache", "relation": "contains", "confidence": "extracted" },
    { "source": "src/cache.ts#Cache.get", "target": "pkg/hash.go#Hash", "relation": "calls", "confidence": "lsp_resolved" },
    { "source": "src/cache.ts", "target": "npm:lodash", "relation": "imports", "confidence": "inferred" },
    { "source": "src/cache.ts#Cache", "target": "src/store.ts#Store", "relation": "implements", "confidence": "lsp_dispatch" },
    { "source": "src/cache.ts#Cache~2.get", "target": "src/cache.ts#Cache.get", "relation": "references", "confidence": "extracted" },
    { "source": "src/cache.ts#Cache~2.get", "target": "src/cache.ts#Cache", "relation": "extends", "confidence": "inferred" }
  ]
}
```

- [ ] **Schritt 2: Den fehlschlagenden Test schreiben**

```go
package model_test

import (
	"os"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
)

func TestDecodeSample(t *testing.T) {
	f, err := os.Open("testdata/wiring.json")
	if err != nil {
		t.Fatalf("open sample: %v", err)
	}
	defer f.Close()

	g, err := model.Decode(f)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(g.Nodes) != 6 || len(g.Edges) != 6 {
		t.Fatalf("got %d nodes and %d edges, want 6 and 6", len(g.Nodes), len(g.Edges))
	}
	if g.Nodes[0].Signature != "" {
		t.Errorf("a null signature must decode as empty, got %q", g.Nodes[0].Signature)
	}
	if g.Nodes[2].Owner != "Cache" {
		t.Errorf("got owner %q, want %q", g.Nodes[2].Owner, "Cache")
	}
	if g.Edges[2].Target != "npm:lodash" {
		t.Errorf("an unresolved import target must survive decoding, got %q", g.Edges[2].Target)
	}
	if g.Edges[1].Confidence != model.ConfidenceLSPResolved {
		t.Errorf("got confidence %q, want %q", g.Edges[1].Confidence, model.ConfidenceLSPResolved)
	}
}

func TestDecodeRejects(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"broken json", `{"meta":`, "read graph:"},
		{"wrong version", `{"meta":{"version":2},"nodes":[],"edges":[]}`, "graph version 2"},
		{"node without id", `{"meta":{"version":1},"nodes":[{"name":"x","kind":"function","path":"a.ts"}],"edges":[]}`, "node 0 has no id"},
		{"node without path", `{"meta":{"version":1},"nodes":[{"id":"a.ts#x","name":"x","kind":"function"}],"edges":[]}`, `node "a.ts#x" has no path`},
		{"node without kind", `{"meta":{"version":1},"nodes":[{"id":"a.ts#x","name":"x","path":"a.ts"}],"edges":[]}`, `node "a.ts#x" has no kind`},
		{"edge without source", `{"meta":{"version":1},"nodes":[],"edges":[{"target":"b","relation":"calls"}]}`, "edge 0 has no source"},
		{"edge without target", `{"meta":{"version":1},"nodes":[],"edges":[{"source":"a","relation":"calls"}]}`, "edge 0 has no target"},
		{"unknown relation", `{"meta":{"version":1},"nodes":[],"edges":[{"source":"a","target":"b","relation":"summons"}]}`, `edge 0 has relation "summons"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := model.Decode(strings.NewReader(c.in))
			if err == nil {
				t.Fatalf("got no error, want one containing %q", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("got error %q, want it to contain %q", err, c.want)
			}
		})
	}
}

func TestValidateAcceptsEmptyGraph(t *testing.T) {
	g := &model.Graph{Meta: model.Meta{Version: 1}}
	if err := g.Validate(); err != nil {
		t.Errorf("an empty graph of the right version is valid, got %v", err)
	}
}
```

- [ ] **Schritt 3: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/model/ -run 'TestDecode|TestValidate' -v
```

Erwartet: `undefined: model.Decode`.

- [ ] **Schritt 4: Die minimale Implementierung schreiben**

```go
package model

import (
	"encoding/json"
	"fmt"
	"io"
)

// schemaVersion is the only wiring.json this model reads. A writer that bumps
// it changed something; failing loudly beats ranking stale shapes.
const schemaVersion = 1

// Decode reads a wiring.json and validates it.
func Decode(r io.Reader) (*Graph, error) {
	var g Graph
	if err := json.NewDecoder(r).Decode(&g); err != nil {
		return nil, fmt.Errorf("read graph: %w", err)
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return &g, nil
}

// Validate reports the first thing about a graph that no consumer can work
// around: a version it does not know, a node without identity or place, an
// edge with a loose end or a relation the schema never had.
//
// It does not check that every edge end is a node: an unresolved import names
// its module, and that is a fact about the code, not a defect.
func (g *Graph) Validate() error {
	if g.Meta.Version != schemaVersion {
		return fmt.Errorf("graph version %d, want %d", g.Meta.Version, schemaVersion)
	}
	for i, n := range g.Nodes {
		switch {
		case n.ID == "":
			return fmt.Errorf("node %d has no id", i)
		case n.Path == "":
			return fmt.Errorf("node %q has no path", n.ID)
		case n.Kind == "":
			return fmt.Errorf("node %q has no kind", n.ID)
		}
	}
	for i, e := range g.Edges {
		switch {
		case e.Source == "":
			return fmt.Errorf("edge %d has no source", i)
		case e.Target == "":
			return fmt.Errorf("edge %d has no target", i)
		case !e.Relation.IsWalk() && e.Relation != RelationContains:
			return fmt.Errorf("edge %d has relation %q, which the schema does not define", i, e.Relation)
		}
	}
	return nil
}
```

- [ ] **Schritt 5: Lauf, der grün sein muss**

```sh
go test ./internal/code/model/ -v
```

Erwartet: PASS, alle Fälle.

- [ ] **Schritt 6: Committen**

```sh
git add internal/code/model
git commit -m "Read a wiring graph and refuse the shapes no consumer survives"
```

---

### Task 3: `internal/code/pagerank` — Topologie

**Files:**
- Create: `internal/code/pagerank/topology.go`
- Test: `internal/code/pagerank/topology_test.go`

**Interfaces:**
- Consumes: `model.Graph`, `model.NodeID`, `Relation.IsWalk`.
- Produces: `type Topology`, `func Prepare(g *model.Graph, keep func(model.NodeID) bool) Topology`,
  `func (t Topology) Len() int`,
  `func (t Topology) NeighboursOf(id model.NodeID) []model.NodeID`.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package pagerank_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/pagerank"
)

// graphOf builds the hand-made fixtures of this package: every id becomes a
// function node in its own file, every pair becomes one edge.
func graphOf(ids []model.NodeID, edges [][3]string) *model.Graph {
	g := &model.Graph{Meta: model.Meta{Version: 1}}
	for _, id := range ids {
		g.Nodes = append(g.Nodes, model.Node{ID: id, Name: string(id), Kind: "function", Path: string(id) + ".ts"})
	}
	for _, e := range edges {
		rel := model.Relation(e[2])
		if rel == "" {
			rel = model.RelationCalls
		}
		g.Edges = append(g.Edges, model.Edge{
			Source: model.NodeID(e[0]), Target: model.NodeID(e[1]),
			Relation: rel, Confidence: model.ConfidenceExtracted,
		})
	}
	return g
}

func TestPrepareIsUndirectedOverWalkRelations(t *testing.T) {
	g := graphOf(
		[]model.NodeID{"hub", "a", "b"},
		[][3]string{{"hub", "a", ""}, {"hub", "b", "contains"}},
	)
	topo := pagerank.Prepare(g, nil)

	if got := topo.Len(); got != 3 {
		t.Fatalf("got %d nodes, want 3", got)
	}
	// "a" is reachable from "hub" and back; "b" hangs on a contains edge and is
	// therefore dangling.
	if got := topo.NeighboursOf("a"); len(got) != 1 || got[0] != "hub" {
		t.Errorf("got neighbours of a = %v, want [hub]", got)
	}
	if got := topo.NeighboursOf("b"); len(got) != 0 {
		t.Errorf("got neighbours of b = %v, want none -- contains is not walkable", got)
	}
}

func TestPrepareDropsEdgesWithoutTwoNodes(t *testing.T) {
	g := graphOf([]model.NodeID{"a"}, [][3]string{{"a", "npm:lodash", "imports"}})
	topo := pagerank.Prepare(g, nil)

	if got := topo.NeighboursOf("a"); len(got) != 0 {
		t.Errorf("got %v, want none -- a module string is no node", got)
	}
}

func TestPrepareKeepFilter(t *testing.T) {
	g := graphOf(
		[]model.NodeID{"hub", "a", "x"},
		[][3]string{{"hub", "a", ""}, {"a", "x", ""}},
	)
	topo := pagerank.Prepare(g, func(id model.NodeID) bool { return id != "x" })

	if got := topo.Len(); got != 2 {
		t.Fatalf("got %d nodes, want 2", got)
	}
	if got := topo.NeighboursOf("a"); len(got) != 1 || got[0] != "hub" {
		t.Errorf("got %v, want [hub] -- an edge needs both ends inside the filter", got)
	}
	if got := topo.NeighboursOf("x"); got != nil {
		t.Errorf("got %v, want nil for a filtered-out node", got)
	}
}

func TestPrepareNilGraph(t *testing.T) {
	if got := pagerank.Prepare(nil, nil).Len(); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/pagerank/ -v
```

Erwartet: `undefined: pagerank.Prepare`.

- [ ] **Schritt 3: Die minimale Implementierung schreiben**

```go
// Package pagerank ranks a code graph by personalized PageRank: a
// random walk with restart, seeded with whatever a caller found lexically.
// Mass gathers on the nodes wired into the seeded cluster, so a node that
// merely shares a word with the query sinks. Lexical proposes, the graph
// disposes.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/ask/graphrank.ts.
package pagerank

import "github.com/xidus90/loomux/internal/code/model"

// Topology is the undirected adjacency a rank walks. Preparing it apart from
// the walk lets several seed sets share one scan of the graph.
//
// Nodes are held as a slice with an index beside it, not as a map: the
// iteration order of a map would reach the floating-point sums and make two
// runs of the same rank differ in the last bits.
type Topology struct {
	ids   []model.NodeID
	index map[model.NodeID]int
	adj   [][]int
}

// Len returns how many nodes carry rank.
func (t Topology) Len() int { return len(t.ids) }

// NeighboursOf returns the walk neighbours of a node, in graph order. It
// exists for the tests of this package and for callers that want to see the
// prepared shape.
func (t Topology) NeighboursOf(id model.NodeID) []model.NodeID {
	i, ok := t.index[id]
	if !ok {
		return nil
	}
	out := make([]model.NodeID, 0, len(t.adj[i]))
	for _, j := range t.adj[i] {
		out = append(out, t.ids[j])
	}
	return out
}

// Prepare builds the undirected topology over the walk relations. keep narrows
// it to a subgraph; an edge counts only when both ends pass, and a nil keep
// takes everything.
//
// An edge whose end is not a node -- an unresolved import naming its module --
// is dropped: it would otherwise gather rank mass and be reported as a result
// nobody can open.
func Prepare(g *model.Graph, keep func(model.NodeID) bool) Topology {
	t := Topology{index: map[model.NodeID]int{}}
	if g == nil {
		return t
	}
	for _, n := range g.Nodes {
		if keep != nil && !keep(n.ID) {
			continue
		}
		if _, seen := t.index[n.ID]; seen {
			continue
		}
		t.index[n.ID] = len(t.ids)
		t.ids = append(t.ids, n.ID)
	}
	t.adj = make([][]int, len(t.ids))
	for _, e := range g.Edges {
		if !e.Relation.IsWalk() {
			continue
		}
		from, ok := t.index[e.Source]
		if !ok {
			continue
		}
		to, ok := t.index[e.Target]
		if !ok {
			continue
		}
		t.adj[from] = append(t.adj[from], to)
		t.adj[to] = append(t.adj[to], from)
	}
	return t
}
```

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/code/pagerank/ -v
```

Erwartet: PASS.

- [ ] **Schritt 5: Committen**

```sh
git add internal/code/pagerank
git commit -m "Prepare the undirected topology a rank walks"
```

---

### Task 4: `internal/code/pagerank` — die Iteration

**Files:**
- Create: `internal/code/pagerank/rank.go`
- Test: `internal/code/pagerank/rank_test.go`

**Interfaces:**
- Consumes: `Topology` aus Task 3.
- Produces: `type Options struct{ Alpha float64; Iterations int }`,
  `type Scored struct{ ID model.NodeID; Score float64 }`,
  `func Rank(t Topology, seed map[model.NodeID]float64, opts Options) []Scored`.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

Die vier Konstanten stammen aus `test/graphrank.test.ts` von Graft `1e352a3`
und wurden am 2026-09-16 unabhängig nachgerechnet; `c` weicht dabei um 1 ULP
ab, daher der Vergleich mit Toleranz statt auf Gleichheit.

```go
package pagerank_test

import (
	"math"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/pagerank"
)

func scoreOf(got []pagerank.Scored, id model.NodeID) float64 {
	for _, s := range got {
		if s.ID == id {
			return s.Score
		}
	}
	return 0
}

func has(got []pagerank.Scored, id model.NodeID) bool {
	for _, s := range got {
		if s.ID == id {
			return true
		}
	}
	return false
}

func TestRankConnectedSeedBeatsIsolatedSeed(t *testing.T) {
	g := graphOf(
		[]model.NodeID{"hub", "a", "b", "c", "lone"},
		[][3]string{{"hub", "a", ""}, {"hub", "b", ""}, {"hub", "c", ""}},
	)
	got := pagerank.Rank(pagerank.Prepare(g, nil), map[model.NodeID]float64{"hub": 1, "lone": 1}, pagerank.Options{})

	if scoreOf(got, "hub") <= scoreOf(got, "lone") {
		t.Errorf("connected seed %v must beat isolated seed %v", scoreOf(got, "hub"), scoreOf(got, "lone"))
	}
	if got[0].Score != 1 {
		t.Errorf("got top score %v, want 1 -- the output is max-normalized", got[0].Score)
	}
}

func TestRankNeighboursAccrueMassWithoutRestartWeight(t *testing.T) {
	g := graphOf([]model.NodeID{"hub", "a", "b"}, [][3]string{{"hub", "a", ""}, {"hub", "b", ""}})
	got := pagerank.Rank(pagerank.Prepare(g, nil), map[model.NodeID]float64{"hub": 1}, pagerank.Options{})

	if scoreOf(got, "a") <= 0 || scoreOf(got, "b") <= 0 {
		t.Errorf("a neighbour of the only seed gets walk mass, got a=%v b=%v", scoreOf(got, "a"), scoreOf(got, "b"))
	}
}

func TestRankEmptyOrZeroSeeds(t *testing.T) {
	g := graphOf([]model.NodeID{"a", "b"}, [][3]string{{"a", "b", ""}})
	topo := pagerank.Prepare(g, nil)

	if got := pagerank.Rank(topo, nil, pagerank.Options{}); got != nil {
		t.Errorf("got %v, want nil for no seeds", got)
	}
	if got := pagerank.Rank(topo, map[model.NodeID]float64{"a": 0}, pagerank.Options{}); got != nil {
		t.Errorf("got %v, want nil for an all-zero seed", got)
	}
	if got := pagerank.Rank(topo, map[model.NodeID]float64{"a": -3}, pagerank.Options{}); got != nil {
		t.Errorf("got %v, want nil for a negative seed", got)
	}
	if got := pagerank.Rank(topo, map[model.NodeID]float64{"ghost": 1}, pagerank.Options{}); got != nil {
		t.Errorf("got %v, want nil for a seed naming no node", got)
	}
}

func TestRankUnresolvedImportTargetNeverRanks(t *testing.T) {
	g := graphOf([]model.NodeID{"a"}, [][3]string{{"a", "npm:lodash", "imports"}})
	got := pagerank.Rank(pagerank.Prepare(g, nil), map[model.NodeID]float64{"a": 1}, pagerank.Options{})

	if scoreOf(got, "a") != 1 {
		t.Errorf("got %v, want 1 -- the sole seed without neighbours stays on top", scoreOf(got, "a"))
	}
	if has(got, "npm:lodash") {
		t.Error("a module string must never become a ranked node")
	}
}

// The reference values of the dangling fixture, from Graft's own test file and
// recomputed independently on 2026-09-16.
func TestRankDanglingMassFixture(t *testing.T) {
	g := graphOf(
		[]model.NodeID{"a", "b", "c", "d", "e"},
		[][3]string{{"a", "b", ""}, {"b", "c", ""}},
	)
	got := pagerank.Rank(pagerank.Prepare(g, nil), map[model.NodeID]float64{"a": 2, "d": 1}, pagerank.Options{})

	want := map[model.NodeID]float64{
		"a": 0.9577162737326514,
		"b": 1,
		"c": 0.37462976423958994,
		"d": 0.29154325474653076,
	}
	for id, w := range want {
		if diff := math.Abs(scoreOf(got, id) - w); diff > 1e-9 {
			t.Errorf("%s: got %v, want %v (diff %v)", id, scoreOf(got, id), w, diff)
		}
	}
	if got[0].ID != "b" {
		t.Errorf("got top node %q, want b -- the hub linking both chain ends", got[0].ID)
	}
	if has(got, "e") {
		t.Error("an unreached node must be absent from the result")
	}
}

func TestRankTieOrderIsAlphabetical(t *testing.T) {
	// Two seeds of equal weight in mirror-image positions score identically;
	// only the id can break the tie, and it must do so the same way every run.
	g := graphOf([]model.NodeID{"zeta", "alpha"}, nil)
	got := pagerank.Rank(pagerank.Prepare(g, nil), map[model.NodeID]float64{"zeta": 1, "alpha": 1}, pagerank.Options{})

	if len(got) != 2 || got[0].ID != "alpha" || got[1].ID != "zeta" {
		t.Errorf("got %v, want alpha before zeta at equal score", got)
	}
}

func TestRankOptionsOverrideTheDefaults(t *testing.T) {
	g := graphOf([]model.NodeID{"hub", "a"}, [][3]string{{"hub", "a", ""}})
	topo := pagerank.Prepare(g, nil)

	loose := pagerank.Rank(topo, map[model.NodeID]float64{"hub": 1}, pagerank.Options{Alpha: 0.9, Iterations: 1})
	tight := pagerank.Rank(topo, map[model.NodeID]float64{"hub": 1}, pagerank.Options{})
	if scoreOf(loose, "a") >= scoreOf(tight, "a") {
		t.Errorf("a higher restart probability starves the neighbour: got a=%v with alpha 0.9, a=%v with the default",
			scoreOf(loose, "a"), scoreOf(tight, "a"))
	}
}

func TestRankKeepFilterMatchesRankingTheComponentAlone(t *testing.T) {
	componentA := []model.NodeID{"hub", "a", "b", "c"}
	edgesA := [][3]string{{"hub", "a", ""}, {"hub", "b", ""}, {"hub", "c", ""}}
	full := graphOf(append(append([]model.NodeID{}, componentA...), "x", "y", "z"),
		append(append([][3]string{}, edgesA...), [3]string{"x", "y", ""}, [3]string{"y", "z", ""}))
	only := graphOf(componentA, edgesA)

	seeds := map[model.NodeID]float64{"hub": 1, "a": 0.5}
	inA := map[model.NodeID]bool{"hub": true, "a": true, "b": true, "c": true}
	filtered := pagerank.Rank(pagerank.Prepare(full, func(id model.NodeID) bool { return inA[id] }), seeds, pagerank.Options{})
	reference := pagerank.Rank(pagerank.Prepare(only, nil), seeds, pagerank.Options{})

	if len(filtered) != len(reference) {
		t.Fatalf("got %d scores, want %d", len(filtered), len(reference))
	}
	for i := range reference {
		if filtered[i] != reference[i] {
			t.Errorf("at %d: got %v, want %v", i, filtered[i], reference[i])
		}
	}
	if has(filtered, "x") {
		t.Error("a filtered-out component must not appear")
	}
}

func TestRankSeedsOutsideTheFilterAreIgnored(t *testing.T) {
	g := graphOf([]model.NodeID{"hub", "a", "outside"}, [][3]string{{"hub", "a", ""}})
	got := pagerank.Rank(
		pagerank.Prepare(g, func(id model.NodeID) bool { return id != "outside" }),
		map[model.NodeID]float64{"hub": 1, "outside": 5},
		pagerank.Options{},
	)

	if has(got, "outside") {
		t.Error("a filtered-out seed must never appear in the output")
	}
	if scoreOf(got, "hub") <= 0 {
		t.Error("the seed inside the filter is still ranked")
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/pagerank/ -run TestRank -v
```

Erwartet: `undefined: pagerank.Rank`.

- [ ] **Schritt 3: Die minimale Implementierung schreiben**

```go
package pagerank

import (
	"sort"

	"github.com/xidus90/loomux/internal/code/model"
)

// The defaults of the reference: a quarter of the mass restarts at the seeds
// every step, and 25 steps settle a graph of this size.
const (
	defaultAlpha      = 0.25
	defaultIterations = 25
)

// Options tunes the walk. A zero value means the defaults.
type Options struct {
	// Alpha is the restart probability: the share of mass that teleports back
	// to the seed set each step. Higher keeps the walk closer to the seeds.
	Alpha float64
	// Iterations is the power-iteration count.
	Iterations int
}

func (o Options) alpha() float64 {
	if o.Alpha <= 0 || o.Alpha >= 1 {
		return defaultAlpha
	}
	return o.Alpha
}

func (o Options) iterations() int {
	if o.Iterations <= 0 {
		return defaultIterations
	}
	return o.Iterations
}

// Scored is one node and its rank.
type Scored struct {
	ID    model.NodeID
	Score float64
}

// Rank walks the topology from seed and returns the nodes the walk reached,
// best first, ties broken alphabetically by id.
//
// Scores are normalized so the best node scores 1; a node the walk never
// reached is absent rather than zero. Seeds naming no node, and weights that
// are not positive, are ignored -- a seed set that is empty after that yields
// no result at all.
func Rank(t Topology, seed map[model.NodeID]float64, opts Options) []Scored {
	restart := make([]float64, len(t.ids))
	total := 0.0
	for id, w := range seed {
		i, ok := t.index[id]
		if !ok || w <= 0 {
			continue
		}
		restart[i] = w
		total += w
	}
	if total <= 0 {
		return nil
	}
	for i := range restart {
		restart[i] /= total
	}

	alpha, iterations := opts.alpha(), opts.iterations()
	rank := make([]float64, len(restart))
	copy(rank, restart)
	next := make([]float64, len(restart))
	for step := 0; step < iterations; step++ {
		// Teleport first: every step, alpha of the mass returns to the seeds.
		for i, r := range restart {
			next[i] = alpha * r
		}
		dangling := 0.0
		for i, mass := range rank {
			nbrs := t.adj[i]
			if len(nbrs) == 0 {
				dangling += mass
				continue
			}
			share := (1 - alpha) * mass / float64(len(nbrs))
			for _, j := range nbrs {
				next[j] += share
			}
		}
		// The mass of nodes with no walk edges is pooled and returned to the
		// seeds in one pass, weighted like the restart distribution. Doing it
		// per dangling node is the same arithmetic at O(dangling x seeds), and
		// a graph of 20k isolated nodes seeded broadly takes minutes that way.
		if dangling > 0 {
			dm := (1 - alpha) * dangling
			for i, r := range restart {
				if r > 0 {
					next[i] += dm * r
				}
			}
		}
		rank, next = next, rank
	}

	max := 0.0
	for _, v := range rank {
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		return nil
	}
	out := make([]Scored, 0, len(rank))
	for i, v := range rank {
		if v <= 0 {
			continue
		}
		out = append(out, Scored{ID: t.ids[i], Score: v / max})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ID < out[j].ID
	})
	return out
}
```

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/code/pagerank/ -v
```

Erwartet: PASS, besonders `TestRankDanglingMassFixture`.

- [ ] **Schritt 5: Committen**

```sh
git add internal/code/pagerank
git commit -m "Rank a prepared topology by personalized PageRank"
```

---

### Task 5: `internal/code/pagerank` — der pathologische Graph

**Files:**
- Modify: `internal/code/pagerank/rank_test.go`

**Interfaces:** keine neuen.

Dieser Fall ist der Grund, warum die Dangling-Masse gepoolt wird. Er gehört in
die Suite, nicht in eine Messung daneben: er schlägt fehl, sobald jemand die
Verteilung je Knoten zurückbaut.

- [ ] **Schritt 1: Den Test schreiben**

```go
func TestRankBroadSeedsOnMostlyDanglingGraph(t *testing.T) {
	// 20k nodes without edges, 100 of them chained, every node seeded: the
	// shape a common-word query takes on a real 32k-node graph. With the mass
	// redistributed per dangling node this is O(dangling x seeds) and runs for
	// minutes.
	const n = 20000
	ids := make([]model.NodeID, 0, n)
	for i := 0; i < n; i++ {
		ids = append(ids, model.NodeID(fmt.Sprintf("n%d", i)))
	}
	edges := make([][3]string, 0, 100)
	for i := 0; i < 100; i++ {
		edges = append(edges, [3]string{fmt.Sprintf("n%d", i), fmt.Sprintf("n%d", i+1), ""})
	}
	seeds := make(map[model.NodeID]float64, n)
	for _, id := range ids {
		seeds[id] = 1
	}

	start := time.Now()
	got := pagerank.Rank(pagerank.Prepare(graphOf(ids, edges), nil), seeds, pagerank.Options{})
	elapsed := time.Since(start)

	if len(got) == 0 {
		t.Fatal("got no scores")
	}
	if elapsed > 3*time.Second {
		t.Errorf("took %s -- dangling redistribution must not be O(dangling x seeds)", elapsed)
	}
}
```

Der Importblock von `rank_test.go` lautet danach vollständig:

```go
import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/pagerank"
)
```

- [ ] **Schritt 2: Lauf**

```sh
go test ./internal/code/pagerank/ -run TestRankBroadSeeds -v
```

Erwartet: PASS, deutlich unter einer Sekunde.

- [ ] **Schritt 3: Committen**

```sh
git add internal/code/pagerank/rank_test.go
git commit -m "Hold the rank to the shape that made pooling necessary"
```

---

### Task 6: `internal/code/blast` — Index und ein Sprung

**Files:**
- Create: `internal/code/blast/index.go`, `internal/code/blast/reach.go`
- Test: `internal/code/blast/index_test.go`

**Interfaces:**
- Consumes: `model.Graph`, `model.Node`, `Relation.IsWalk`.
- Produces: `type Direction`, `const In, Out`, `type Depth`, `const All`,
  `type Hit struct{ ID model.NodeID; Node *model.Node; Relation model.Relation; Depth int }`,
  `func New(g *model.Graph) *Index`,
  `func (x *Index) Reach(start []model.NodeID, dir Direction, depth Depth) []Hit`.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package blast_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
)

// baseGraph is Graft's traverse fixture: a file containing a method, that
// method calling another, and an import nobody resolved.
func baseGraph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "src/cache.ts", Name: "cache.ts", Kind: model.KindFile, Path: "src/cache.ts"},
			{ID: "src/cache.ts#Cache.get", Name: "get", Kind: "method", Path: "src/cache.ts"},
			{ID: "src/widget.ts#Widget.render", Name: "render", Kind: "method", Path: "src/widget.ts"},
			{ID: "pkg/hash.go#Hash", Name: "Hash", Kind: "function", Path: "pkg/hash.go"},
		},
		Edges: []model.Edge{
			{Source: "src/cache.ts", Target: "src/cache.ts#Cache.get", Relation: model.RelationContains},
			{Source: "src/cache.ts#Cache.get", Target: "src/widget.ts#Widget.render", Relation: model.RelationCalls},
			{Source: "src/cache.ts#Cache.get", Target: "npm:lodash", Relation: model.RelationImports},
		},
	}
}

func hitOf(hits []blast.Hit, id model.NodeID) (blast.Hit, bool) {
	for _, h := range hits {
		if h.ID == id {
			return h, true
		}
	}
	return blast.Hit{}, false
}

func TestReachOutKeepsUnresolvedTargets(t *testing.T) {
	got := blast.New(baseGraph()).Reach([]model.NodeID{"src/cache.ts#Cache.get"}, blast.Out, 1)

	if len(got) != 2 {
		t.Fatalf("got %d hits, want 2: %v", len(got), got)
	}
	called, ok := hitOf(got, "src/widget.ts#Widget.render")
	if !ok {
		t.Fatal("the called method is missing")
	}
	if called.Node == nil || called.Node.Name != "render" {
		t.Errorf("got node %v, want the render method", called.Node)
	}
	if called.Relation != model.RelationCalls || called.Depth != 1 {
		t.Errorf("got %v at depth %d, want calls at depth 1", called.Relation, called.Depth)
	}

	unresolved, ok := hitOf(got, "npm:lodash")
	if !ok {
		t.Fatal("an unresolved import target is kept -- who asks what this depends on wants to see it")
	}
	if unresolved.Node != nil {
		t.Errorf("got node %v, want none for a module string", unresolved.Node)
	}
	if unresolved.Relation != model.RelationImports {
		t.Errorf("got %v, want imports", unresolved.Relation)
	}
}

func TestReachInExcludesContains(t *testing.T) {
	x := blast.New(baseGraph())

	got := x.Reach([]model.NodeID{"src/widget.ts#Widget.render"}, blast.In, 1)
	if len(got) != 1 || got[0].ID != "src/cache.ts#Cache.get" || got[0].Relation != model.RelationCalls {
		t.Fatalf("got %v, want the calling method alone", got)
	}

	// The file that contains the method must never show up as its caller.
	if got := x.Reach([]model.NodeID{"src/cache.ts#Cache.get"}, blast.In, 1); len(got) != 0 {
		t.Errorf("got %v, want none", got)
	}
}

func TestReachWithoutEdges(t *testing.T) {
	x := blast.New(baseGraph())
	if got := x.Reach([]model.NodeID{"pkg/hash.go#Hash"}, blast.In, 1); got != nil {
		t.Errorf("got %v, want nil", got)
	}
	if got := x.Reach([]model.NodeID{"pkg/hash.go#Hash"}, blast.Out, 1); got != nil {
		t.Errorf("got %v, want nil", got)
	}
	if got := x.Reach(nil, blast.In, 1); got != nil {
		t.Errorf("got %v, want nil for no start", got)
	}
	if got := blast.New(nil).Reach([]model.NodeID{"a"}, blast.In, 1); got != nil {
		t.Errorf("got %v, want nil for a nil graph", got)
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/blast/ -v
```

Erwartet: `undefined: blast.New`.

- [ ] **Schritt 3: Die minimale Implementierung schreiben**

```go
// Package blast walks the wiring of a code graph: who breaks if this changes,
// and what this depends on. It answers the callers, the callees and the blast
// radius of a refactoring from the same index.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/traverse.ts.
package blast

import "github.com/xidus90/loomux/internal/code/model"

// Direction is which way a walk follows its edges.
type Direction int

const (
	// In walks incoming edges: who calls, references or imports this.
	In Direction = iota
	// Out walks outgoing edges: what this calls, references or imports.
	Out
)

// Depth caps how far a walk goes. All follows the edges as far as they lead.
type Depth int

// All is the transitive closure.
const All Depth = -1

// Hit is one node a walk reached.
//
// Node is nil when the id is not a node of the graph: an unresolved import
// names its module, and a walk reports it rather than hiding the dependency.
type Hit struct {
	ID       model.NodeID
	Node     *model.Node
	Relation model.Relation
	Depth    int
}

type link struct {
	other    model.NodeID
	relation model.Relation
}

// Index is the adjacency of one graph, built once and walked many times.
type Index struct {
	in    map[model.NodeID][]link
	out   map[model.NodeID][]link
	nodes map[model.NodeID]*model.Node
}

// New indexes a graph for walking. Only walk relations enter the adjacency;
// "contains" would make every file a hub.
func New(g *model.Graph) *Index {
	x := &Index{
		in:    map[model.NodeID][]link{},
		out:   map[model.NodeID][]link{},
		nodes: map[model.NodeID]*model.Node{},
	}
	if g == nil {
		return x
	}
	for i := range g.Nodes {
		x.nodes[g.Nodes[i].ID] = &g.Nodes[i]
	}
	for _, e := range g.Edges {
		if !e.Relation.IsWalk() {
			continue
		}
		x.out[e.Source] = append(x.out[e.Source], link{other: e.Target, relation: e.Relation})
		x.in[e.Target] = append(x.in[e.Target], link{other: e.Source, relation: e.Relation})
	}
	return x
}
```

- [ ] **Schritt 4: `Reach` in `reach.go` schreiben**

```go
package blast

import "github.com/xidus90/loomux/internal/code/model"

// Reach walks from every id in start and returns what it found, in the order
// it found it.
//
// Breadth first: a node is reported once, at the depth it was first reached
// from any start, so a diamond converges to one hit. The start ids are never
// their own hits.
func (x *Index) Reach(start []model.NodeID, dir Direction, depth Depth) []Hit {
	adj := x.out
	if dir == In {
		adj = x.in
	}

	visited := make(map[model.NodeID]bool, len(start))
	frontier := make([]model.NodeID, 0, len(start))
	for _, id := range start {
		if visited[id] {
			continue
		}
		visited[id] = true
		frontier = append(frontier, id)
	}

	var hits []Hit
	for d := 1; (depth == All || Depth(d) <= depth) && len(frontier) > 0; d++ {
		var next []model.NodeID
		for _, current := range frontier {
			for _, l := range adj[current] {
				if visited[l.other] {
					continue
				}
				visited[l.other] = true
				hits = append(hits, Hit{ID: l.other, Node: x.nodes[l.other], Relation: l.relation, Depth: d})
				next = append(next, l.other)
			}
		}
		frontier = next
	}
	return hits
}
```

- [ ] **Schritt 5: Lauf, der grün sein muss**

```sh
go test ./internal/code/blast/ -v
```

Erwartet: PASS.

- [ ] **Schritt 6: Committen**

```sh
git add internal/code/blast
git commit -m "Walk the wiring in both directions from one index"
```

---

### Task 7: `internal/code/blast` — Tiefe, Diamant und mehrere Startknoten

**Files:**
- Create: `internal/code/blast/reach_test.go`

**Interfaces:** keine neuen.

- [ ] **Schritt 1: Den Test schreiben**

```go
package blast_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
)

// diamondGraph: A and B both call X, C calls both A and B. Walking incoming
// edges from X reaches C twice at the same depth.
func diamondGraph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "X", Name: "X", Kind: "function", Path: "x.ts"},
			{ID: "A", Name: "A", Kind: "function", Path: "a.ts"},
			{ID: "B", Name: "B", Kind: "function", Path: "b.ts"},
			{ID: "C", Name: "C", Kind: "function", Path: "c.ts"},
		},
		Edges: []model.Edge{
			{Source: "A", Target: "X", Relation: model.RelationCalls},
			{Source: "B", Target: "X", Relation: model.RelationCalls},
			{Source: "C", Target: "A", Relation: model.RelationCalls},
			{Source: "C", Target: "B", Relation: model.RelationCalls},
		},
	}
}

func idsAndDepths(hits []blast.Hit) map[model.NodeID]int {
	got := map[model.NodeID]int{}
	for _, h := range hits {
		got[h.ID] = h.Depth
	}
	return got
}

func TestReachDiamondDedupsAtMinimumDepth(t *testing.T) {
	got := blast.New(diamondGraph()).Reach([]model.NodeID{"X"}, blast.In, 2)

	if len(got) != 3 {
		t.Fatalf("got %d hits, want 3: A and B at depth 1, C once at depth 2", len(got))
	}
	depths := idsAndDepths(got)
	if depths["A"] != 1 || depths["B"] != 1 || depths["C"] != 2 {
		t.Errorf("got depths %v, want A=1 B=1 C=2", depths)
	}
}

func TestReachDepthCap(t *testing.T) {
	got := blast.New(diamondGraph()).Reach([]model.NodeID{"X"}, blast.In, 1)

	depths := idsAndDepths(got)
	if len(depths) != 2 || depths["A"] != 1 || depths["B"] != 1 {
		t.Errorf("got %v, want A and B alone", depths)
	}
}

func TestReachAllFollowsTheWholeChain(t *testing.T) {
	g := &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "a", Kind: "function", Path: "a.ts"},
			{ID: "b", Kind: "function", Path: "b.ts"},
			{ID: "c", Kind: "function", Path: "c.ts"},
			{ID: "d", Kind: "function", Path: "d.ts"},
		},
		Edges: []model.Edge{
			{Source: "a", Target: "b", Relation: model.RelationCalls},
			{Source: "b", Target: "c", Relation: model.RelationCalls},
			{Source: "c", Target: "d", Relation: model.RelationCalls},
		},
	}
	got := blast.New(g).Reach([]model.NodeID{"a"}, blast.Out, blast.All)

	depths := idsAndDepths(got)
	if len(depths) != 3 || depths["b"] != 1 || depths["c"] != 2 || depths["d"] != 3 {
		t.Errorf("got %v, want b=1 c=2 d=3", depths)
	}
}

func TestReachExcludesItsOwnStartAndDedupsAcrossThem(t *testing.T) {
	// C calls into both A and B; reached from two starts at the same depth it
	// is still reported once, and neither start reports itself.
	g := &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "A", Kind: "function", Path: "a.ts"},
			{ID: "B", Kind: "function", Path: "b.ts"},
			{ID: "C", Kind: "function", Path: "c.ts"},
		},
		Edges: []model.Edge{
			{Source: "C", Target: "A", Relation: model.RelationCalls},
			{Source: "C", Target: "B", Relation: model.RelationCalls},
		},
	}
	got := blast.New(g).Reach([]model.NodeID{"A", "B"}, blast.In, 2)

	if len(got) != 1 || got[0].ID != "C" || got[0].Depth != 1 {
		t.Errorf("got %v, want C once at depth 1", got)
	}
}

// fileAndSymbolGraph: b.ts imports a.ts as a file AND calls a function defined
// in it. A file's dependents are only complete when the walk starts on the
// file node and on every symbol of that file.
func fileAndSymbolGraph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "src/a.ts", Name: "a.ts", Kind: model.KindFile, Path: "src/a.ts"},
			{ID: "src/a.ts#helper", Name: "helper", Kind: "function", Path: "src/a.ts"},
			{ID: "src/b.ts", Name: "b.ts", Kind: model.KindFile, Path: "src/b.ts"},
			{ID: "src/b.ts#useB", Name: "useB", Kind: "function", Path: "src/b.ts"},
		},
		Edges: []model.Edge{
			{Source: "src/b.ts", Target: "src/a.ts", Relation: model.RelationImports},
			{Source: "src/b.ts#useB", Target: "src/a.ts#helper", Relation: model.RelationCalls},
		},
	}
}

func TestReachFromAFileNodeAloneMissesTheCallers(t *testing.T) {
	g := fileAndSymbolGraph()
	x := blast.New(g)

	// The file node alone sees only the file-level import edge: the call
	// targets the symbol id and is invisible from there.
	fileOnly := idsAndDepths(x.Reach([]model.NodeID{"src/a.ts"}, blast.In, 2))
	if len(fileOnly) != 1 || fileOnly["src/b.ts"] != 1 {
		t.Fatalf("got %v, want the importing file alone", fileOnly)
	}

	// Starting on the file and its symbols recovers both dependents. This is
	// what a caller builds with model.SymbolsInFile.
	start := append([]model.NodeID{"src/a.ts"}, model.SymbolsInFile(g, "src/a.ts")...)
	both := idsAndDepths(x.Reach(start, blast.In, 2))
	if len(both) != 2 || both["src/b.ts"] != 1 || both["src/b.ts#useB"] != 1 {
		t.Errorf("got %v, want both the importing file and the calling symbol", both)
	}
}
```

- [ ] **Schritt 2: Lauf**

```sh
go test ./internal/code/blast/ -v
```

Erwartet: PASS.

- [ ] **Schritt 3: Committen**

```sh
git add internal/code/blast/reach_test.go
git commit -m "Pin what a walk does with depth, diamonds and several starts"
```

---

### Task 8: Das ganze Tor und die Mutationsrunde

**Files:**
- Create: `docs/.superpowers/parity/code-g1.md`

**Interfaces:** keine.

- [ ] **Schritt 1: Coverage prüfen**

```sh
go test ./... -count=1 -covermode=set -coverpkg=./... -coverprofile=coverage.out
go run ./cmd/loomux dev covergate --profile coverage.out
```

Erwartet: kein Befund unter 100 % in `internal/code/**`. Meldet das Tor eine
Funktion, fehlt ein Fall — Test nachziehen, **nicht** `//coverage:exempt`
setzen. Die einzige vertretbare Ausnahme wäre eine Funktion, die ohne
Plattformfehler nicht erreichbar ist; in diesen drei Paketen gibt es keine.

- [ ] **Schritt 2: Die Mutationsrunde fahren**

```sh
go run ./cmd/loomux dev mutants internal/code/pagerank internal/code/blast
```

- [ ] **Schritt 3: Die Runde aufschreiben**

`docs/.superpowers/parity/code-g1.md` bekommt Kopf und Tabelle im Format der
Stufe 1b-1: Quelle, Spec, Regel, dann je Mutant eine Zeile mit Ort, Mutation,
Ausgang und Verfügung. Ein Überlebender ist entweder ein fehlender Test — dann
kommt der Test dazu und die Runde läuft erneut — oder eine bewusst nicht
festgeschriebene Freiheit, dann steht die Begründung in der Zeile.

Kopf der Datei:

```markdown
# Paritätsliste Code-Graph G1

**Quelle:** `trailhq/Graft` @ `1e352a3` (MIT), `src/ask/graphrank.ts` und
`src/graph/traverse.ts`.
**Spec:** [2026-09-16-loomux-code-g1-delta.md](../specs/2026-09-16-loomux-code-g1-delta.md)
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor die Stufe als
fertig gilt.
```

- [ ] **Schritt 4: Committen**

```sh
git add docs/.superpowers/parity/code-g1.md
git commit -F <nachrichtendatei>
```

- [ ] **Schritt 5: Melden, was offen bleibt**

G1 ist fertig, wenn Tor und Runde grün sind. Offen und **an G2 übergeben**:
die CLI-Verdrahtung (`loomux graph build|check`), der Abschnitt `[graph]` in
`.loomux/config.toml`, `docs/{en,de}/cli-reference.md`, die READMEs und die
Einträge in `docs/{en,de}/benchmarks.md`. Sie liegen in genau den Dateien, die
Stufe 1b-1 parallel ändert; sie werden nachgeholt, sobald 1b-1 nach `master`
gegangen ist und `code-g1` darauf steht.

---

## Self-Review

**Spec-Abdeckung.** §2 Zuschnitt → Tasks 1–7 legen genau die drei Pakete an und
fassen keine der gesperrten Dateien an. §3.1 IDs ohne Span → `NodeID` ist
undurchsichtig, die Beispieldatei führt eine ID mit Ordinal.
§3.2 Dangling-Masse → Task 4, Schritt 3 mit der Fixture in Schritt 1.
§3.3 sechs Relationen, fünf laufbar → Task 1, plus `contains` als
Ausschluss-Test in Task 6. Pfadgleichheit statt `contains` → `SymbolsInFile`
in Task 1 und der Dateifall in Task 7. §3.4 Konfidenz und unaufgelöste Ziele →
Task 2 (Dekodieren), Task 3 (Rank verwirft), Task 6 (Walk behält).
§3.5 max-normierte Ausgabe → Task 4. §3.6 Saat ausgeschlossen, kleinste Tiefe,
`In` auf `edge.target` → Tasks 6 und 7. §4 Schnittstellen → Tasks 1, 3, 4, 6.
§5 Nachweise → neun Rank-Vektoren in Tasks 4 und 5, zwölf Walk-Vektoren in
Tasks 6 und 7. §6 Tor → Task 8. §8 offene Punkte → Task 8, Schritt 5.

**Platzhalter.** Keine. Jeder Codeschritt enthält den Code, jeder Laufschritt
den Befehl und die Erwartung.

**Typkonsistenz.** `model.NodeID` durchgehend; `Prepare(g, keep)` in Task 3
und in den Tests von Task 4 gleich benannt; `Reach(start, dir, depth)` in Task
6 definiert und in Task 7 genauso gerufen; `blast.All` in Task 6 definiert und
in Task 7 benutzt; `Options{Alpha, Iterations}` in Task 4 definiert und dort
benutzt. `graphOf` ist der Helfer des `pagerank`-Tests (Task 3), `baseGraph`,
`diamondGraph` und `fileAndSymbolGraph` die des `blast`-Tests (Tasks 6, 7) —
getrennte Pakete, keine geteilten Helfer.
