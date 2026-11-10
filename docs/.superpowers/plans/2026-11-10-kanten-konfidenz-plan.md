# Kanten-Konfidenz sichtbar machen: Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Treffer von `graph callers`/`graph_trace_calls`, `graph blast`/`graph_blast` und dem Edit-Hook tragen `inferred`, wenn ihr Weg eine geratene Kante enthält.

**Architecture:** `blast.Index` behält die Konfidenz je Kante; `EdgeWalk`, `Reach` und `Radius` vererben sie als `Hit.Inferred` (schwächstes Glied, sicher gewinnt bei gleicher Tiefe). Die drei Textausgaben hängen `inferred` an; JSON bekommt `"inferred": true`.

**Tech Stack:** Go, Standardbibliothek; Tests mit `go test`.

**Spec:** `docs/.superpowers/specs/2026-11-10-kanten-konfidenz-design.md`

## Global Constraints

- Markiert wird nur `model.ConfidenceInferred`; `extracted`, `lsp_resolved`, `lsp_dispatch` und leere Konfidenz gelten als sicher.
- JSON-Feld: `Inferred bool` mit Tag `json:"inferred,omitempty"`; nichts wird umbenannt.
- Sichere Treffer: Ausgabe byte-gleich zu heute.
- Unverändert: `signal` in `radius.go`, `InDegree`/`InDegreeWhere`, `audit`, `pagerank`, `ask`.
- Kein Flag zum Ausblenden.
- 100 % Coverage (`check coverage`); Code, Kommentare, Commits englisch; Commit ohne `Co-Authored-By`.
- Arbeitsbaum: `.claude/worktrees/edge-confidence`, Branch `feat/edge-confidence`.

## Review Focus

1. Parallele Kanten A→B (`calls`/inferred + `references`/extracted) bei Tiefe > 1: Treffer sicher, Relation `references` — Test in Task 1.
2. Diamant, geratener Arm zuerst gefunden, sicherer Arm danach auf gleicher Tiefe: sicher — Test in Task 1.
3. Erbt ein Kind die Hochstufung seines Elternknotens? Kette X←A←C mit A über zwei Wege — Test in Task 1.
4. `Radius`: zwei Dateien erreichen denselben Treffer auf gleicher Tiefe, erst geraten, dann sicher — Test in Task 2.
5. Treffer ohne Knoten (`Node == nil`, ID-Zeile in `CallersReport`) mit geratener Kante — Test in Task 3.

---

### Task 1: Konfidenz im Walk (`blast`)

**Files:**
- Modify: `internal/code/blast/index.go` (`Hit`, `link`, `New`)
- Modify: `internal/code/blast/edgewalk.go` (Tiefe 1)
- Modify: `internal/code/blast/reach.go`
- Test: `internal/code/blast/index_test.go`, `internal/code/blast/edgewalk_test.go`

**Interfaces:**
- Produces: `blast.Hit.Inferred bool` (`json:"inferred,omitempty"`); `link.inferred bool`, gesetzt in `New` aus `e.Confidence == model.ConfidenceInferred`.

- [ ] **Step 1: Failing tests schreiben**

In `index_test.go` (Graph-Literale wie `diamondGraph`, Kanten mit `Confidence`):

```go
// X <- A (inferred), X <- B (extracted), A <- C, B <- C
func TestReachMarksAHitBehindAnInferredEdge(t *testing.T)
// Reach([X], In, All): A.Inferred == true, B.Inferred == false,
// C.Inferred == false (C reaches X sure via B at the same depth 2).

func TestReachInheritsInferredAlongThePath(t *testing.T)
// X <- A (inferred), A <- C (extracted): C.Inferred == true, C.Depth == 2.

func TestReachUpgradesToTheSureEdgeAndItsRelation(t *testing.T)
// X <- A twice: calls/inferred first, references/extracted second.
// Reach([X], In, All): A.Inferred == false, A.Relation == model.RelationReferences.

func TestReachPassesAnUpgradeOnToTheNextDepth(t *testing.T)
// X <- P (inferred), X <- Q (extracted), P <- M, Q <- M, M <- Z (all others extracted);
// P listed before Q so M is first found via P. M.Inferred == false (sure via Q
// at depth 2), Z.Inferred == false (inherits at depth 3).

func TestReachKeepsAnInferredHitWhenOnlyADeeperPathIsSure(t *testing.T)
// X <- A (inferred), X <- B, B <- A (extracted): A at depth 1 stays Inferred == true.
```

In `edgewalk_test.go`:

```go
func TestEdgeWalkDepth1MarksEachEdgeOnItsOwn(t *testing.T)
// two edges X <- A: calls/inferred and references/extracted.
// EdgeWalk(X, In, 1) gives two hits: [calls, Inferred true], [references, Inferred false].
```

- [ ] **Step 2: Run, erwarte FAIL** (`Inferred` undefiniert)

Run: `go test ./internal/code/blast/`

- [ ] **Step 3: Implementieren**

`New` und `EdgeWalk` Tiefe 1: Feld durchreichen. `Reach`: zu `visited` eine Map `at map[model.NodeID]int` (Index in `hits`) und für die Start-IDs Konfidenz sicher. Pfad-Konfidenz eines Frontier-Knotens = `hits[at[current]].Inferred`, für Start-IDs `false`. Bei einem bereits besuchten Ziel:

```go
if i, ok := at[l.other]; ok && hits[i].Depth == d && hits[i].Inferred && !pathInferred {
    hits[i].Inferred, hits[i].Relation = false, l.relation
}
```

Die BFS ist ebenenweise; eine Hochstufung auf Tiefe d ist abgeschlossen, bevor Tiefe d+1 sie liest. Kommentar dazu einen Satz.

- [ ] **Step 4: Run, erwarte PASS**

Run: `go test ./internal/code/blast/`

- [ ] **Step 5: Commit**

```bash
git add internal/code/blast/
git commit -m "feat(graph): carry inferred edges through the walk"
```

### Task 2: Zusammenführung in `Radius`

**Files:**
- Modify: `internal/code/blast/radius.go:128-139`
- Test: `internal/code/blast/radius_test.go`

**Interfaces:**
- Consumes: `Hit.Inferred` aus Task 1.

- [ ] **Step 1: Failing tests**

```go
func TestRadiusPrefersTheSureEdgeAtAnEqualDepth(t *testing.T)
// radiusGraph plus other.go -calls/inferred-> lib.go#Add and
// other.go -references/extracted-> main.go#main; radius(g, 1, changed("lib.go",4), changed("main.go",4)).
// other.go: Inferred == false, Relation == model.RelationReferences.

func TestRadiusTakesInferredFromTheShallowerPath(t *testing.T)
// a later file reaches a hit at a smaller depth via an inferred edge:
// Depth, Relation and Inferred all come from the shallower path (Inferred == true).
```

`TestRadiusKeepsTheFirstRelationAtAnEqualDepth` bleibt grün (beide Kanten sicher).

- [ ] **Step 2: Run, erwarte FAIL**

Run: `go test ./internal/code/blast/ -run Radius`

- [ ] **Step 3: Implementieren** im `seen`-Zweig: kleinere Tiefe → `Depth`, `Relation`, `Inferred` übernehmen; gleiche Tiefe und `rep.Hits[i].Inferred && !h.Inferred` → `Relation`, `Inferred` übernehmen.

- [ ] **Step 4: Run, erwarte PASS**

Run: `go test ./internal/code/blast/`

- [ ] **Step 5: Commit**

```bash
git add internal/code/blast/radius.go internal/code/blast/radius_test.go
git commit -m "feat(graph): keep the sure path when blast merges hits"
```

### Task 3: Ausgabe von `callers` und `blast`

**Files:**
- Modify: `internal/code/query/callers.go` (`CallerHit`, `Callers`, `CallersReport`)
- Modify: `internal/code/query/blast.go` (`BlastReport`)
- Test: `internal/code/query/callers_test.go`, `internal/code/query/blast_test.go`

**Interfaces:**
- Consumes: `blast.Hit.Inferred`.
- Produces: `query.CallerHit.Inferred bool` (`json:"inferred,omitempty"`).

- [ ] **Step 1: Failing tests**

```go
func TestCallersMarksAnInferredCaller(t *testing.T)
// built(t) (sample: lib_test.go calls Run in another file of package lib -> inferred;
// main.go calls lib.Run -> extracted). Callers(root,"Run",{In,1}):
// hit TestRun: Inferred == true; hit main: Inferred == false.
// CallersReport contains "(function, depth 1, calls, inferred) in lib/lib_test.go"
// and the main line without ", inferred".

func TestCallersReportMarksAnInferredHitWithoutANode(t *testing.T)
// CallersReport(CallersAnswer{Start: &model.Node{...}, Hits: []CallerHit{{ID: "ext", Relation: "imports", Depth: 1, Inferred: true}}})
// contains "  - ext (depth 1, imports, inferred)\n".
```

In `TestBlastReport` ein Fall `"inferred hit"`: wie `"everything"`, Hit mit `Inferred: true`; erwartete Zeile
`"  main (function, depth 1, calls, inferred) in main.go:L5-L7 from calc/calc.go, calc/doc.go\n"`.
Der bestehende Fall `"everything"` bleibt byte-gleich.

Zu prüfen in Step 2: die Annahme, dass `TestRun -> Run` im Beispiel `inferred` ist (vgl. `testdata/cases/graph/mixed/graph.golden:13`, gleiche Paketform). Trifft sie nicht zu, Fixture in einem eigenen Testrepo anlegen statt die Erwartung zu ändern.

Dazu ein Marshal-Test: `json.Marshal(CallerHit{ID: "a", Inferred: true})` enthält `"inferred":true`; ohne `Inferred` fehlt der Schlüssel. In Task 2 analog `TestRadiusReportEncodesItsFieldNames` um einen geratenen Treffer erweitern.

Ausgaben anderer Pakete über dieselben Fixtures ändern sich mit: `testdata/cases/graph/callers.golden`, `testdata/cases/graph/blast.golden`, `internal/hooks/post_edit_test.go`, `internal/verify/report_test.go`, `internal/cli/graph_test.go`. Erwartungen anpassen, Fixtures nicht.

- [ ] **Step 2: Run, erwarte FAIL**

Run: `go test ./internal/code/query/ -run "Callers|BlastReport"`

- [ ] **Step 3: Implementieren**: `Callers` kopiert `h.Inferred`; beide Reports hängen `", inferred"` als letztes Element in die Klammer.

- [ ] **Step 4: Run, erwarte PASS**

Run: `go test ./internal/code/query/`

- [ ] **Step 5: Commit**

```bash
git add internal/code/query/
git commit -m "feat(graph): mark inferred hits in callers and blast output"
```

### Task 4: Edit-Hook

**Files:**
- Modify: `internal/hooks/blast_monitor.go:82-86`
- Test: `internal/hooks/blast_monitor_test.go`

**Interfaces:**
- Consumes: `blast.Hit.Inferred`.

- [ ] **Step 1: Failing test**

```go
func TestBlastAsideMarksAnInferredCaller(t *testing.T)
// graphRepo(t), edit Add as in TestBlastAsideNamesTheCallersOfAChangedFunction.
// got contains "  TestAdd (calc/calc_test.go, inferred)" and "  main (main.go)\n" or ends with "  main (main.go)".
```

`TestBlastAsideNamesTheCallersOfAChangedFunction` prüft `"  TestAdd (calc/calc_test.go)"` per `Contains`; das passt nach der Änderung nicht mehr — Erwartung dort auf die neue Zeile anpassen.

- [ ] **Step 2: Run, erwarte FAIL**

Run: `go test ./internal/hooks/ -run BlastAside`

- [ ] **Step 3: Implementieren**: Zeile `"  %s (%s, inferred)"`, wenn `h.Inferred`.

- [ ] **Step 4: Run, erwarte PASS**

Run: `go test ./internal/hooks/`

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/
git commit -m "feat(hooks): mark inferred callers in the edit aside"
```

### Task 5: Werkzeugbeschreibung und Wiki

**Files:**
- Modify: `internal/mcptools/tools.go` (Beschreibungen `graph_trace_calls`, `graph_blast`)
- Modify: `docs/wiki/topics/code-graph.md` (Abschnitt „Die Oberflächen“ oder „Wie der Graph entsteht“)

- [ ] **Step 1: Beschreibungen ergänzen**, je ein Halbsatz: `Hits reached only through a call resolved by a repo-wide unique name are marked "inferred".`

- [ ] **Step 2: Wiki**: ein Absatz (deutsch): `inferred` = Aufruf nur über einen im ganzen Repo eindeutigen Namen zugeordnet (Go: auch Aufrufe in eine andere Datei desselben Pakets); Walks geben es nach dem schwächsten Glied weiter, bei gleicher Tiefe gewinnt der sichere Weg; erscheint in `graph callers`, `graph blast`, ihren MCP-Werkzeugen und im Edit-Hook; Zählungen und Signale ignorieren es. Frontmatter-Datum/Quellen nach `docs/wiki/_schema.md` nachziehen.

- [ ] **Step 3: Volle Prüfkette**

Run: `go run ./cmd/loomux check precommit` (oder der Hook beim Commit)
Expected: alle Prüfungen `ok`, Coverage 100 %.

- [ ] **Step 4: Commit**

```bash
git add internal/mcptools/tools.go docs/wiki/topics/code-graph.md
git commit -m "docs(graph): explain the inferred marker"
```
