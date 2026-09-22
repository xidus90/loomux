# loomux G4a — Implementierungsplan: Navigation & MCP-Werkzeuge

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bereitstellung der vollständigen Code-Graph-Navigationspalette (`callers`, `skeleton`, `grep`, `map`, `stats`) auf der Kommandozeile (`loomux graph ...`) und als vier neue MCP-Werkzeuge (`graph_trace_calls`, `graph_file_api`, `graph_find_all`, `graph_repo_map`) in `loomux serve` mit striktem Privacy-Schutz.

**Architecture:** Die puren Rechenalgorithmen arbeiten entkoppelt auf `model.Graph` und injizierten Lesefunktionen (`FileReader`). `internal/code/blast` wird um `Resolve` (Symbolauflösung mit Go-Paketfilter) und `EdgeWalk` (Tiefe-1-Spezialsemantik) erweitert. `skeleton`, `grep` und `repomap` erhalten schlanke Fachpakete. `internal/code/query` orchestriert Frische, Laden und Berichte für CLI und MCP gemeinsam. `internal/serve/graph` bindet die 4 MCP-Tools mit `keep(path)` an.

**Tech Stack:** Go (Standardbibliothek), `github.com/modelcontextprotocol/go-sdk` v1.8.0. **Keine neuen Abhängigkeiten.**

**Spec:** `docs/.superpowers/specs/2026-09-22-loomux-code-g4-delta.md`. Bei Widerspruch gilt die Spec, danach diese Datei.

**Referenz:** `trailhq/Graft` @ `1e352a3` (MIT): `src/graph/traverse.ts`, `src/graph/traverse-cli.ts`, `src/search/grep.ts`, `src/graph/map.ts`, `src/blast/evidence.ts`, `src/mcp/tools.ts`.

---

## Global Constraints

- **Arbeitsort:** Worktree `C:\Users\micro\Documents\#GIT\loomux\.worktrees\code-g4`, Branch `code-g4`, abgezweigt von `master` `27af73f`.
- **Commits:** Conventional Commits (`feat(…)`, `refactor(…)`, `docs(…)`, `test(…)`) — das Tor `check commit-msg` weist alles andere ab. Autor und Committer ist der Nutzer; keine Werbezeilen.
- **Niemand außer dem Nutzer pusht.**
- **Coverage 100 % je Funktion.** Unterschreiten nur mit `//coverage:exempt <grund>` direkt über `func`.
- **Kein `init()`, keine Paketvariable, die eingebettete Daten parst.**
- **Sprachen:** Code, Bezeichner, Kommentare, Fehlermeldungen englisch; Pläne, Paritätsakten und Specs deutsch.
- **`.loomux/config.toml` schreibt kein Agent.**
- **Graft-Paritätswerte:**
  - `limit` über MCP ist 5 (`graph_find_code`), Tiefe Standard 1 (`graph_trace_calls`).
  - MCP `depth`: `"all"` und `"full"` sind unendlich (`DepthAll = -1`), Zahlen abrunden, `< 1` wird 1.
  - `grep`: 160 Runen Kappung je Zeile, Gruppen stabil sortiert nach `inDegree` absteigend, dann Pfad aufsteigend.
  - `repomap`: `DEFAULT_MAX_DIRS = 16`, `DEFAULT_HUBS_PER_DIR = 3`, `DEFAULT_HOTSPOTS = 12`, `SPLIT_THRESHOLD = 0.6`.
  - InDegree zählt ausschließlich `WALK_RELATIONS` (`calls`, `references`, `imports`, `implements`, `extends`), niemals `contains`.

---

## File Structure

| Datei | Verantwortung |
|---|---|
| `internal/code/model/spans.go` | `SymbolSpan`, `FileSpans`, `Enclosing`, `Innermost` |
| `internal/code/model/spans_test.go` | Unit-Tests für Spans-Indizierung und Intervall-Zuordnung |
| `internal/code/blast/resolve.go` | `Resolve` (Port von `resolveSymbol` mit Go-Paketfilter E3) |
| `internal/code/blast/resolve_test.go` | Unit-Tests für `Resolve` gegen alle Namens-, Id- und Pfad-Varianten |
| `internal/code/blast/edgewalk.go` | `EdgeWalk`, `InDegree`, `callersOf`/`calleesOf`-Semantik bei Tiefe 1 |
| `internal/code/blast/edgewalk_test.go` | Unit-Tests für Tiefe 1 vs. BFS, Zyklen und doppelte Kanten |
| `internal/code/blast/quote.go` | `Quote`, heuristischer Beleg für Kanten bei Tiefe 1 |
| `internal/code/blast/quote_test.go` | Unit-Tests für Quelltext-Zitate mit injiziertem Reader |
| `internal/code/skeleton/skeleton.go` | `Extract`: Signaturen, Typen und Spans je Datei |
| `internal/code/skeleton/skeleton_test.go` | Unit-Tests für `skeleton` |
| `internal/code/grep/grep.go` | `Search`: Symbol-gekoppelter Regex-Grep, 160-Rune-Kappung, inDegree-Ranking |
| `internal/code/grep/grep_test.go` | Unit-Tests für Grep mit injiziertem Reader und RE2 |
| `internal/code/repomap/map.go` | `Build`: Verzeichnis-Cluster, 60%-Monolith-Refinement, Hubs & Hotspots |
| `internal/code/repomap/map_test.go` | Unit-Tests für Repo-Map |
| `internal/code/query/callers.go` | Orchestrierung für `callers`: Frische, Laden, Resolve, Walk, Quote, Report |
| `internal/code/query/skeleton.go` | Orchestrierung für `skeleton`: Frische, Laden, Report |
| `internal/code/query/grep.go` | Orchestrierung für `grep`: Frische, Laden, Reader-Injektion, Report |
| `internal/code/query/repomap.go` | Orchestrierung für `map`: Frische, Laden, Report |
| `internal/code/query/stats.go` | `Stats`: Zählung Knoten, Kanten je Relation, Dateien, Sprachen, Graph-Größe |
| `internal/mcptools/tools.go` | Ergänzung um die 4 Tool-Definitionen (11 Tools gesamt) |
| `internal/mcptools/tools_test.go` | Schemavalidierung für alle 11 Tools |
| `internal/serve/graph/tools.go` | Handler für `graph_trace_calls`, `file_api`, `find_all`, `repo_map` mit Privacy |
| `internal/serve/graph/tools_test.go` | MCP-Tests für alle 6 Graph-Tools auf Local- und Cloud-Kanal |
| `internal/cli/graph.go` | Subcommands `callers`, `skeleton`, `grep`, `map`, `stats` |
| `internal/cli/graph_test.go` | CLI-Tests für alle neuen Graph-Befehle |
| `docs/.superpowers/parity/code-g4.md` | Paritätsakte für Stufe G4 mit allen Verfügungen |

---

### Task 1: Model FileSpans Index (`internal/code/model`)

**Files:**
- Create: `internal/code/model/spans.go`
- Test: `internal/code/model/spans_test.go`

**Interfaces:**
- Consumes: `model.Graph`, `model.Node`, `model.Span`
- Produces:
  ```go
  type SymbolSpan struct {
      Node *Node
      From, To int
  }
  func FileSpans(g *Graph) map[string][]SymbolSpan
  func Enclosing(spans []SymbolSpan, line int) *Node
  func Innermost(spans []SymbolSpan, from, to int) []*Node
  ```

- [ ] **Step 1: Write the failing test**

```go
// internal/code/model/spans_test.go
package model_test

import (
	"testing"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestFileSpansAndEnclosing(t *testing.T) {
	g := &model.Graph{
		Nodes: []model.Node{
			{ID: "a.go#A", Path: "a.go", Name: "A", Span: "L10-L40"},
			{ID: "a.go#A.Inner", Path: "a.go", Name: "Inner", Span: "L20-L30"},
			{ID: "a.go", Path: "a.go", Kind: model.KindFile, Span: "L1-L50"},
		},
	}
	fs := model.FileSpans(g)
	spans := fs["a.go"]
	if len(spans) != 2 {
		t.Fatalf("want 2 symbol spans, got %d", len(spans))
	}
	// grep.ts rule: largest span start (innermost in enclosing sense)
	enc := model.Enclosing(spans, 25)
	if enc == nil || enc.ID != "a.go#A.Inner" {
		t.Fatalf("want Inner, got %v", enc)
	}
	encOuter := model.Enclosing(spans, 35)
	if encOuter == nil || encOuter.ID != "a.go#A" {
		t.Fatalf("want A, got %v", encOuter)
	}
	encNone := model.Enclosing(spans, 5)
	if encNone != nil {
		t.Fatalf("want nil for line outside symbols, got %v", encNone)
	}

	// blast.ts rule: innermost contains no other hit symbol
	inner := model.Innermost(spans, 15, 25)
	if len(inner) != 1 || inner[0].ID != "a.go#A.Inner" {
		t.Fatalf("want Inner, got %v", inner)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/code/model`  
Expected: FAIL with `undefined: model.FileSpans`

- [ ] **Step 3: Write minimal implementation**

```go
// internal/code/model/spans.go
package model

import "sort"

type SymbolSpan struct {
	Node     *Node
	From, To int
}

func FileSpans(g *Graph) map[string][]SymbolSpan {
	m := make(map[string][]SymbolSpan)
	if g == nil {
		return m
	}
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if n.Kind == KindFile {
			continue
		}
		from, to, ok := n.Span.Lines()
		if !ok {
			continue
		}
		m[n.Path] = append(m[n.Path], SymbolSpan{Node: n, From: from, To: to})
	}
	for path := range m {
		sort.Slice(m[path], func(i, j int) bool {
			if m[path][i].From != m[path][j].From {
				return m[path][i].From < m[path][j].From
			}
			return m[path][i].To > m[path][j].To
		})
	}
	return m
}

func Enclosing(spans []SymbolSpan, line int) *Node {
	var best *Node
	maxStart := -1
	for _, s := range spans {
		if line >= s.From && line <= s.To {
			if s.From > maxStart {
				maxStart = s.From
				best = s.Node
			}
		}
	}
	return best
}

func Innermost(spans []SymbolSpan, from, to int) []*Node {
	var hit []SymbolSpan
	for _, s := range spans {
		if s.From <= to && s.To >= from {
			hit = append(hit, s)
		}
	}
	var out []*Node
	for i, s1 := range hit {
		containsOther := false
		for j, s2 := range hit {
			if i != j && s2.From >= s1.From && s2.To <= s1.To {
				containsOther = true
				break
			}
		}
		if !containsOther {
			out = append(out, s1.Node)
		}
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -cover ./internal/code/model`  
Expected: PASS with 100% coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/code/model/spans.go internal/code/model/spans_test.go
git commit -m "feat(model): index file symbol spans with Enclosing and Innermost queries"
```

---

### Task 2: Symbol Resolution & EdgeWalk in Blast (`internal/code/blast`)

**Files:**
- Create: `internal/code/blast/resolve.go`
- Create: `internal/code/blast/resolve_test.go`
- Create: `internal/code/blast/edgewalk.go`
- Create: `internal/code/blast/edgewalk_test.go`
- Create: `internal/code/blast/quote.go`
- Create: `internal/code/blast/quote_test.go`

**Interfaces:**
- Consumes: `model.Graph`, `model.Node`, `model.NodeID`, `blast.Index`
- Produces:
  ```go
  func Resolve(g *model.Graph, query, in string) ([]*model.Node, error)
  func (x *Index) EdgeWalk(start *model.Node, dir Direction, depth Depth) []Hit
  func (x *Index) InDegree(id model.NodeID) int
  func QuoteLine(read func(string) ([]byte, error), path string, span model.Span, targetName string) (int, string, bool)
  ```

- [ ] **Step 1: Write the failing tests**

Tests covering:
- Strip ordinals `~2` on each dotted segment (`Cache~2.get` matches `Cache.get`).
- Stage 1: Exact name / `#query` / `.query`.
- Go package filter (E3): `hooks.Write` matches `internal/hooks/post_edit.go#Write` and excludes other packages.
- Stage 2: Bare name fallback (`hashstructure.Hash` $\to$ `Hash`).
- Stage 3: File node fallback, ensuring `x.go` is not captured by a symbol named `go`.
- `--in` prefix check with `assertPrefixIndexed` returning error for unknown prefix.
- `EdgeWalk`: Depth 1 preserves duplicate edges and self-loops; Depth > 1 converges.
- `InDegree`: Counts incoming walk relations only.
- `QuoteLine`: Finds first line inside span mentioning symbol name.

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/code/blast`  
Expected: FAIL with undefined identifiers.

- [ ] **Step 3: Implement Resolve, EdgeWalk, InDegree, Quote**

Implement `resolve.go`, `edgewalk.go`, `quote.go` according to Spec §3.1, §3.2, §3.3, §3.4 and §9 (E3).

- [ ] **Step 4: Run tests to verify passing**

Run: `go test -cover ./internal/code/blast`  
Expected: PASS with 100% coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/code/blast/
git commit -m "feat(blast): add Resolve with Go package filter, EdgeWalk and Quote"
```

---

### Task 3: Skeleton Extraction (`internal/code/skeleton`)

**Files:**
- Create: `internal/code/skeleton/skeleton.go`
- Test: `internal/code/skeleton/skeleton_test.go`

**Interfaces:**
- Consumes: `model.Graph`, `model.Node`
- Produces:
  ```go
  type Entry struct {
      Name      string     `json:"name"`
      Kind      model.Kind `json:"kind"`
      Span      model.Span `json:"span"`
      Signature string     `json:"signature,omitempty"`
      Doc       string     `json:"doc,omitempty"`
  }
  func Extract(g *model.Graph, fileQuery string) (string, []Entry, error)
  ```

- [ ] **Step 1: Write the failing test**

Test resolving by exact relative path and by unique basename (e.g. `reach.go`); test error on ambiguous basename (`test.go` in multiple dirs) and unknown file. Verify entries are sorted by `Span.StartLine` and exclude `KindFile`.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/code/skeleton`  
Expected: FAIL with package not found.

- [ ] **Step 3: Write minimal implementation**

Implement `Extract` filtering `g.Nodes` for the resolved file path, sorted by span start.

- [ ] **Step 4: Run test to verify passing**

Run: `go test -cover ./internal/code/skeleton`  
Expected: PASS with 100% coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/code/skeleton/
git commit -m "feat(skeleton): extract file symbol signatures and types from graph"
```

---

### Task 4: Symbol-gekoppelter Grep (`internal/code/grep`)

**Files:**
- Create: `internal/code/grep/grep.go`
- Test: `internal/code/grep/grep_test.go`

**Interfaces:**
- Consumes: `model.Graph`, `model.FileSpans`, `model.Enclosing`, `blast.Index.InDegree`
- Produces:
  ```go
  type Hit struct { Line int; Text string }
  type Group struct {
      Symbol   *model.Node
      Path     string
      InDegree int
      Hits     []Hit
  }
  type Options struct {
      IgnoreCase bool
      Fixed      bool
      In         string
      MaxHits    int // default 300
  }
  type Result struct {
      Pattern       string
      FilesSearched int
      TotalHits     int
      Groups        []Group
      TruncatedHits int
      Unreadable    int
  }
  func Search(g *model.Graph, x *blast.Index, spans map[string][]model.SymbolSpan, pattern string, opts Options, read func(string) ([]byte, error)) (Result, error)
  ```

- [ ] **Step 1: Write the failing test**

Test:
- Match inside a symbol $\to$ group by symbol with `inDegree`.
- Match outside symbols $\to$ group under file (`Symbol: nil`, `inDegree: 0`).
- Stable sort: `inDegree` descending, then `path` ascending.
- Capping at 160 runes at rune boundaries.
- `MaxHits` exceeded: hits beyond cap are counted into `TruncatedHits`.
- Fixed string vs. RE2 regex. Rejection of Lookaround / Backreferences with clear error.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/code/grep`  
Expected: FAIL.

- [ ] **Step 3: Write minimal implementation**

Implement `grep.go` obeying Spec §3.4, §3.10 and §5.

- [ ] **Step 4: Run test to verify passing**

Run: `go test -cover ./internal/code/grep`  
Expected: PASS with 100% coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/code/grep/
git commit -m "feat(grep): implement symbol-coupled regex search ranked by inDegree"
```

---

### Task 5: Repo-Map (`internal/code/repomap`)

**Files:**
- Create: `internal/code/repomap/map.go`
- Test: `internal/code/repomap/map_test.go`

**Interfaces:**
- Consumes: `model.Graph`, `blast.Index.InDegree`
- Produces:
  ```go
  type Hub struct {
      Name     string
      Kind     model.Kind
      Path     string
      Span     model.Span
      InDegree int
  }
  type DirEntry struct {
      Path      string
      Files     int
      Symbols   int
      Languages []string
      Hubs      []Hub
      IsFile    bool
  }
  type RepoMap struct {
      Totals   struct{ Files, Symbols, Edges int; Languages []string }
      Dirs     []DirEntry
      Hotspots []Hub
      Dropped  int
  }
  type Options struct {
      MaxDirs    int // default 16
      HubsPerDir int // default 3
      Hotspots   int // default 12
  }
  func Build(g *model.Graph, x *blast.Index, opts Options) RepoMap
  func Format(m RepoMap) string
  ```

- [ ] **Step 1: Write the failing test**

Test directory grouping, 60% monolith refinement (`SPLIT_THRESHOLD = 0.6`), Hub ranking (ties: name asc, path asc), Hotspots ranking, dropped directories count.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/code/repomap`  
Expected: FAIL.

- [ ] **Step 3: Write minimal implementation**

Implement `map.go` according to Spec §3.9.

- [ ] **Step 4: Run test to verify passing**

Run: `go test -cover ./internal/code/repomap`  
Expected: PASS with 100% coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/code/repomap/
git commit -m "feat(repomap): compute token-budgeted repo orientation map"
```

---

### Task 6: Query-Orchestrierung & Stats (`internal/code/query`)

**Files:**
- Create: `internal/code/query/callers.go`
- Create: `internal/code/query/skeleton.go`
- Create: `internal/code/query/grep.go`
- Create: `internal/code/query/repomap.go`
- Create: `internal/code/query/stats.go`
- Tests: `internal/code/query/*_test.go`

**Interfaces:**
- Consumes: `store`, `model`, `blast`, `skeleton`, `grep`, `repomap`
- Produces:
  ```go
  func Callers(root, symbol string, opts CallersOptions) (CallersAnswer, []string, error)
  func Skeleton(root, file string, opts SkeletonOptions) (SkeletonAnswer, []string, error)
  func Grep(root, pattern string, opts GrepOptions) (GrepAnswer, []string, error)
  func Map(root string, opts MapOptions) (MapAnswer, []string, error)
  func Stats(root string) (StatsAnswer, error)
  // Formatters: CallersReport, SkeletonReport, GrepReport, MapReport, StatsReport
  ```

- [ ] **Step 1: Write the failing tests**

Test orchestrator calls with:
- Missing graph $\to$ `ErrNoGraph`
- Drift check & auto-rebuild (unless `NoRefresh` is set)
- Report generation for human CLI output
- Stats counting nodes, edges by relation, languages, wiring size

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/code/query`  
Expected: FAIL.

- [ ] **Step 3: Implement query modules**

Implement `callers.go`, `skeleton.go`, `grep.go`, `repomap.go`, `stats.go`.

- [ ] **Step 4: Run tests to verify passing**

Run: `go test -cover ./internal/code/query`  
Expected: PASS with 100% coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/code/query/
git commit -m "feat(query): orchestrate callers, skeleton, grep, map and stats"
```

---

### Task 7: MCP-Werkzeugdefinitionen (`internal/mcptools`)

**Files:**
- Modify: `internal/mcptools/tools.go`
- Modify: `internal/mcptools/tools_test.go`

**Interfaces:**
- Produces: `mcptools.Graph()` returns 6 tools; `mcptools.Tools()` returns 11 tools.

- [ ] **Step 1: Write the failing test**

Verify `len(mcptools.Graph()) == 6` and `len(mcptools.Tools()) == 11`. Verify required parameters and property types for `graph_trace_calls`, `graph_file_api`, `graph_find_all`, `graph_repo_map`.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/mcptools`  
Expected: FAIL with tool count 7 != 11.

- [ ] **Step 3: Add tool definitions in `mcptools/tools.go`**

Add the 4 tool definitions matching Spec §3.10 and Graft `tools.ts`.

- [ ] **Step 4: Run test to verify passing**

Run: `go test -cover ./internal/mcptools`  
Expected: PASS with 100% coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/mcptools/
git commit -m "feat(mcptools): register 4 new graph tools (11 tools total)"
```

---

### Task 8: MCP-Handler & Privacy-Filter (`internal/serve/graph`)

**Files:**
- Modify: `internal/serve/graph/tools.go`
- Modify: `internal/serve/graph/tools_test.go`

**Interfaces:**
- Consumes: `mcptools.Graph()`, `privacy.Channel`, `query.*`
- Produces: Handlers registered on server for all 6 graph tools.

- [ ] **Step 1: Write the failing tests**

Test calling all 4 new tools over MCP:
- `graph_trace_calls`: test `direction: in/out`, `depth: all/1/2`, verify `Hidden` count on cloud channel.
- `graph_file_api`: test valid file, test private file refused on cloud channel.
- `graph_find_all`: test regex hits, test privacy filter suppresses lines from private files.
- `graph_repo_map`: test map generation, test private directories omitted on cloud channel.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/serve/graph`  
Expected: FAIL with unhandled tools.

- [ ] **Step 3: Implement handlers in `serve/graph/tools.go`**

Implement handlers using `Deps` and `readable(area.Manifest)` fail-closed filter.

- [ ] **Step 4: Run test to verify passing**

Run: `go test -cover ./internal/serve/graph`  
Expected: PASS with 100% coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/serve/graph/
git commit -m "feat(serve/graph): handle trace_calls, file_api, find_all, repo_map with privacy filter"
```

---

### Task 9: CLI-Befehle (`internal/cli`)

**Files:**
- Modify: `internal/cli/graph.go`
- Test: `internal/cli/graph_test.go`

**Interfaces:**
- Produces:
  - `loomux graph callers <symbol> [--direction in|out] [-d <depth>] [--in <prefix>] [--json]`
  - `loomux graph skeleton <file> [--json]`
  - `loomux graph grep <pattern> [-i] [--fixed] [--in <prefix>] [--max-hits <n>] [--json]`
  - `loomux graph map [--max-dirs <n>] [--json]`
  - `loomux graph stats [--json]`

- [ ] **Step 1: Write the failing tests**

Test invocation of all 5 subcommands with valid flags, invalid flags, missing arguments (exit 2), missing graph (exit 1), `--json` marshaling, and stdout output.

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/cli -run TestGraph`  
Expected: FAIL.

- [ ] **Step 3: Wire subcommands in `cli/graph.go`**

Implement subcommand parsers and handlers connecting flags to `query.*`.

- [ ] **Step 4: Run test to verify passing**

Run: `go test -cover ./internal/cli -run TestGraph`  
Expected: PASS with 100% coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/graph.go internal/cli/graph_test.go
git commit -m "feat(cli): provide graph callers, skeleton, grep, map, and stats commands"
```

---

### Task 10: Golden-Tests, Paritätsakte & Dokumentation

**Files:**
- Create: `testdata/cases/graph/...` (Golden files für `callers`, `skeleton`, `grep`, `map`)
- Create: `docs/.superpowers/parity/code-g4.md`
- Modify: `README.md`, `README.de.md`, `docs/en/cli-reference.md`, `docs/de/cli-reference.md`

- [ ] **Step 1: Create golden fixtures and tests**

Add deterministic golden file tests comparing `callers`, `skeleton`, `grep` and `map` on a fixture repository against static `.golden` files.

- [ ] **Step 2: Create parity ledger `docs/.superpowers/parity/code-g4.md`**

Record every ruling from Spec §3, §4, §5 and §9.

- [ ] **Step 3: Update documentation**

Update `README.md`, `README.de.md`, and deep docs in `docs/en/` and `docs/de/` reflecting that G4a navigation and MCP tools are operational.

- [ ] **Step 4: Run full pre-commit gate**

Run: `go run ./cmd/loomux check precommit`  
Expected: All lanes OK (lint, test, 100% coverage, wiki).

- [ ] **Step 5: Commit**

```bash
git add docs/ testdata/ README.md README.de.md
git commit -m "docs(graph): document stage G4a navigation and record parity ledger"
```
