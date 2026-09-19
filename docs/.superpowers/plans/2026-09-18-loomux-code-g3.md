# loomux G3 — Implementierungsplan: der Graph über MCP

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux serve` bietet `graph_find_code` und `graph_check_freshness` an, auf
beiden Kanälen, mit derselben Sichtbarkeitsregel wie `brain_*`, und eine Abfrage baut nie
einen ersten Graphen.

**Architecture:** Die Abfrage wandert aus `internal/cli/graph.go` in ein neues Paket
`internal/code/query`, das CLI und MCP gemeinsam rufen — das Muster von `brain/answer`.
`internal/serve/graph` ist die dünne MCP-Schicht darüber, `internal/mcptools` hält die zwei
neuen Definitionen neben den fünf alten, `serve.go` verdrahtet sie. `ask` bekommt ein
Prädikat `Keep`, das vor dem Scoring greift.

**Tech Stack:** Go, `github.com/modelcontextprotocol/go-sdk` v1.8.0 (schon in `go.mod`
seit 1b-2). **Keine neue Abhängigkeit.**

**Spec:** `docs/.superpowers/specs/2026-09-18-loomux-code-g3-delta.md`. Bei Widerspruch gilt
die Spec, danach diese Datei.

**Referenz:** `trailhq/Graft` @ `1e352a3` (MIT): `src/mcp/tools.ts`, `src/graph/refresh.ts`,
`src/context/check.ts`.

## Global Constraints

- **Arbeitsort:** Worktree `C:\Users\micro\Documents\#GIT\loomux-code-g3`, Branch
  `code-g3`, abgezweigt von `sdd-1b-2` `11222c8`; das Delta liegt als `7d8739b` darauf.
- **Commits:** Conventional Commits (`feat(…)`, `refactor(…)`, `docs(…)`, `test(…)`) — das
  Tor `check commit-msg` weist alles andere ab. Mehrzeilige Nachrichten über eine Datei und
  `git commit -F`. Autor und Committer ist der Nutzer; **keine** `Co-Authored-By`-Zeile,
  keine Werbezeile. Vor jedem Commit `git branch --show-current` und `git log --oneline -1`
  lesen.
- **Niemand außer dem Nutzer pusht.**
- **Coverage 100 % je Funktion.** Unterschreiten nur mit `//coverage:exempt <grund>` direkt
  über `func`.
- **Kein `init()`, keine Paketvariable, die eingebettete Daten parst.** `errors.New` auf
  Paketebene ist erlaubt.
- **Sprachen:** Code, Bezeichner, Kommentare, Fehlermeldungen englisch; diese Datei,
  Paritätsakte und Specs deutsch.
- **`.loomux/config.toml` schreibt kein Agent.**
- **Werkzeugnamen** genügen `^[a-zA-Z0-9_-]{1,64}$`.
- **Die Werte aus Graft sind nicht zu erfinden:**

  | Wert | Hier | Ort in Graft |
  |---|---|---|
  | `limit` über MCP | 5, kein Schema-`default` (nur in der Beschreibung) | `tools.ts:51` |
  | Quelltext über MCP | immer eingefügt | `tools.ts:165` (`source: true`) |
  | Kein Auffrischen vor `check_freshness` | `NO_REFRESH_TOOLS` | `tools.ts:201–203` |
  | Hinweis vor dem Text | `${note}\n${res.text}` | `tools.ts:241` |
  | Kein Erstbau | `ensureFreshGraph` kehrt ohne `wiring.json` zurück | `refresh.ts:156–170` |

- **Nach jedem Subagentenlauf** `git log -1 --format='%an <%ae>'` lesen.

---

## File Structure

| Datei | Verantwortung |
|---|---|
| `internal/code/lexicon/index.go` | `Where(keep)` — die allgemeine Form von `Filter` |
| `internal/code/ask/ask.go` | `Options.Keep`, greift mit `In` vor dem Scoring und im Lauf |
| `internal/code/query/build.go` | `Extract`, `Build`, `Stats`, `goModPaths` (aus `cli/graph.go`) |
| `internal/code/query/check.go` | `Drift`, `Check`, `Drift.Only`, `CheckReport` (aus `cli/graph.go`) |
| `internal/code/query/ask.go` | `ErrNoGraph`, `AskOptions`, `Ask`, `AskReport` |
| `internal/cli/graph.go` | nur noch Flags, Ausgabe, Exit-Codes, `report` |
| `internal/mcptools/tools.go` | `Brain()`, `Graph()`, `Tools()` |
| `internal/serve/brain/tools.go` | registriert `mcptools.Brain()` statt `Tools()` |
| `internal/serve/graph/tools.go` | `Deps`, `Register`, die zwei Handler |
| `internal/serve/serve.go` | `servegraph.Register` neben `servebrain.Register` |
| `docs/{en,de}/benchmarks.md`, `docs/{en,de}/cli-reference.md`, `README.md`, `README.de.md` | Doku und Messungen |
| `docs/.superpowers/parity/code-g3.md` | Paritätsakte |

---

### Task 0: Arbeitsort prüfen

- [ ] **Step 1: Zweig und Stand lesen**

```bash
git -C "C:/Users/micro/Documents/#GIT/loomux-code-g3" branch --show-current
git -C "C:/Users/micro/Documents/#GIT/loomux-code-g3" log --oneline -2
```

Erwartet: `code-g3`, oben der Commit des Plans, darunter `7d8739b docs(spec): narrow stage G3 …`.

- [ ] **Step 2: Hook-Binary bauen und Basis grün sehen**

```bash
go build -o bin/loomux.exe ./cmd/loomux
go test ./internal/code/... ./internal/cli/... ./internal/serve/... ./internal/mcptools/... ./internal/bridge/... -count=1
```

Erwartet: alles `ok`. Wenn nicht: anhalten und melden, nicht reparieren.

---

### Task 1: `Keep` — der Privacy-Filter vor dem Scoring

**Files:**
- Modify: `internal/code/lexicon/index.go` (`Filter`, neu `Where`)
- Modify: `internal/code/ask/ask.go` (`Options`, `Run`, `walk`)
- Test: `internal/code/lexicon/index_test.go`, `internal/code/ask/ask_test.go`

**Interfaces:**
- Produces: `func (ix *lexicon.Index) Where(keep func(path string) bool) *lexicon.Index`;
  `ask.Options{Limit int; In string; Keep func(path string) bool}` — `Keep == nil` heißt
  „alles".

- [ ] **Step 1: Failing tests schreiben**

In `internal/code/lexicon/index_test.go` ergänzen:

```go
func TestWhereKeepsOnlyWhatThePredicateAdmitsAndRecomputesStatistics(t *testing.T) {
	ix := lexicon.Build(graph())
	sub := ix.Where(func(path string) bool { return path != "web/render.go" })

	for _, d := range sub.Docs {
		if d.ID == "web/render.go#Render" {
			t.Fatal("a document the predicate refuses must be gone")
		}
	}
	if sub.DocCount != len(sub.Docs) {
		t.Errorf("DocCount = %d, want %d", sub.DocCount, len(sub.Docs))
	}
	if _, ok := sub.DF["template"]; ok {
		t.Error("a term that lives only in a refused document must be gone from df")
	}
}

func TestWhereHandsThePathNotTheID(t *testing.T) {
	ix := lexicon.Build(graph())
	var seen []string
	ix.Where(func(path string) bool { seen = append(seen, path); return true })
	for _, p := range seen {
		if strings.Contains(p, "#") {
			t.Fatalf("predicate saw %q, want a path without the symbol part", p)
		}
	}
}
```

(`strings` ggf. in den Import aufnehmen; `graph()` ist der vorhandene Helfer der Datei.)

In `internal/code/ask/ask_test.go` ergänzen:

```go
func TestRunKeepFiltersBeforeScoringAndStillFillsTheLimit(t *testing.T) {
	nodes := []model.Node{
		symbol("secrets/a.go#Render", "Render", "secrets/a.go", "render"),
		symbol("lib/b.go#Render", "Render", "lib/b.go", "render"),
		symbol("lib/c.go#Render", "Render", "lib/c.go", "render"),
	}
	g, ix := world(nodes, nil)
	keep := func(path string) bool { return !strings.HasPrefix(path, "secrets/") }

	got := ask.Run(g, ix, "render", ask.Options{Limit: 2, Keep: keep})
	if len(got.Hits) != 2 {
		t.Fatalf("got %d hits, want the limit of 2 filled from what Keep admits", len(got.Hits))
	}
	for _, h := range got.Hits {
		if h.Path == "secrets/a.go" {
			t.Fatal("a node Keep refuses must not be ranked at all")
		}
	}
}

func TestRunKeepNarrowsTheWalkAndNotOnlyTheSeed(t *testing.T) {
	// The twin of TestRunNarrowsTheWalkAndNotOnlyTheSeed: a refused neighbour
	// must not be rescued by the walk.
	nodes := []model.Node{
		symbol("open/a.go#Seeded", "Seeded", "open/a.go", "seeded"),
		symbol("secrets/b.go#Neighbour", "Neighbour", "secrets/b.go", "nothing matching"),
	}
	edges := []model.Edge{edge("open/a.go#Seeded", "secrets/b.go#Neighbour")}
	g, ix := world(nodes, edges)
	keep := func(path string) bool { return !strings.HasPrefix(path, "secrets/") }

	got := ask.Run(g, ix, "seeded", ask.Options{Keep: keep})
	for _, h := range got.Hits {
		if h.Path == "secrets/b.go" {
			t.Fatal("the walk rescued a node Keep refuses")
		}
	}
}

func TestRunKeepAndInApplyTogether(t *testing.T) {
	nodes := []model.Node{
		symbol("lib/a.go#Render", "Render", "lib/a.go", "render"),
		symbol("lib/secret.go#Render", "Render", "lib/secret.go", "render"),
		symbol("web/c.go#Render", "Render", "web/c.go", "render"),
	}
	g, ix := world(nodes, nil)
	keep := func(path string) bool { return path != "lib/secret.go" }

	got := ask.Run(g, ix, "render", ask.Options{In: "lib", Keep: keep})
	if len(got.Hits) != 1 || got.Hits[0].ID != "lib/a.go#Render" {
		t.Fatalf("got %v, want only lib/a.go#Render", idsOf(got))
	}
}
```

- [ ] **Step 2: Tests laufen lassen, Fehlschlag sehen**

Run: `go test ./internal/code/lexicon/ ./internal/code/ask/ -count=1`
Expected: Build-Fehler `ix.Where undefined` und `unknown field Keep`.

- [ ] **Step 3: `Where` bauen, `Filter` darauf zurückführen**

In `internal/code/lexicon/index.go` den Rumpf von `Filter` ersetzen und `Where` daneben stellen:

```go
func (ix *Index) Filter(prefix string) *Index {
	prefix = NormalizePrefix(prefix)
	if prefix == "" {
		return ix
	}
	return ix.Where(func(path string) bool { return UnderPrefix(path, prefix) })
}

// Where keeps the documents whose path the predicate admits, and recomputes
// the statistics over what is left.
//
// Filter is the path-prefix case of it. The general form exists for a
// predicate that is not a prefix -- an area's never globs -- and has to act at
// the same place for the same reason: a filter applied after scoring would let
// a refused document shape df and take a place in the limit.
func (ix *Index) Where(keep func(path string) bool) *Index {
	out := &Index{Version: ix.Version, DF: map[string]int{}}
	bodyTotal := 0
	for _, d := range ix.Docs {
		if !keep(pathOf(string(d.ID))) {
			continue
		}
		out.Docs = append(out.Docs, d)
		for _, n := range d.Body {
			bodyTotal += n
		}
		for term := range terms(d) {
			out.DF[term]++
		}
	}
	out.DocCount = len(out.Docs)
	if out.DocCount > 0 {
		out.AvgBodyLen = float64(bodyTotal) / float64(out.DocCount)
	}
	return out
}
```

`idUnderPrefix` durch `pathOf` ersetzen (die einzige andere Nutzerin ist `Filter`, die jetzt
über `Where` geht):

```go
// pathOf is the path part of a node id: everything before the '#', or the
// whole id for a file node.
func pathOf(id string) string {
	if i := strings.Index(id, "#"); i >= 0 {
		return id[:i]
	}
	return id
}
```

Vorher mit `grep -n idUnderPrefix internal/code/lexicon/*.go` prüfen, dass kein Test sie
direkt ruft; tut es einer, ihn auf `pathOf` umstellen.

- [ ] **Step 4: `Keep` in `ask`**

In `internal/code/ask/ask.go`:

```go
type Options struct {
	Limit int
	In    string
	// Keep refuses a path before anything is scored; nil admits every path.
	// It is how a caller's privacy rule reaches the ranking without the
	// ranking knowing what privacy is.
	Keep func(path string) bool
}
```

Am Anfang von `Run` die Verengung ersetzen:

```go
	in := lexicon.NormalizePrefix(opts.In)
	admit := admission(in, opts.Keep)
	if admit != nil {
		ix = ix.Where(admit)
	}
```

und weiter unten `pr := walk(g, byID, lex, admit)`. Neue Funktion und angepasstes `walk`:

```go
// admission joins the prefix and the caller's predicate into one rule, or
// nil when neither narrows anything. One rule, because the index and the walk
// must not hold two opinions about what is admitted.
func admission(in string, keep func(string) bool) func(string) bool {
	switch {
	case in == "" && keep == nil:
		return nil
	case keep == nil:
		return func(path string) bool { return lexicon.UnderPrefix(path, in) }
	case in == "":
		return keep
	}
	return func(path string) bool { return lexicon.UnderPrefix(path, in) && keep(path) }
}

func walk(g *model.Graph, byID map[model.NodeID]model.Node, seed map[model.NodeID]float64, admit func(string) bool) []pagerank.Scored {
	if len(seed) == 0 {
		return nil
	}
	keep := func(id model.NodeID) bool { return true }
	if admit != nil {
		keep = func(id model.NodeID) bool { return admit(byID[id].Path) }
	}
	return pagerank.Rank(pagerank.Prepare(g, keep), seed, pagerank.Options{})
}
```

Den Kommentar über `walk` auf „The admission narrows the WALK …" anpassen; die Begründung
bleibt dieselbe.

**Achtung, Verhalten von `In`:** Heute ruft `Run` `ix.Filter(in)`, das `NormalizePrefix` noch
einmal anwendet — idempotent, also gleich. `TestRunNormalizesThePrefixTheWayLexiconDoes` muss
grün bleiben.

- [ ] **Step 5: Tests laufen lassen**

Run: `go test ./internal/code/lexicon/ ./internal/code/ask/ -count=1`
Expected: PASS, alle alten Tests eingeschlossen.

- [ ] **Step 6: Commit**

Nachricht in eine Datei schreiben, dann:

```bash
git add internal/code/lexicon internal/code/ask
git commit -F <msgfile>
```

Nachricht: `feat(ask): refuse a path before it is scored` mit einem Absatz, dass `Keep` mit
`In` zu einer Regel verschmilzt, die Index und Lauf teilen.

---

### Task 2: `internal/code/query` — Bau und Prüfung aus der CLI herauslösen

Reine Verschiebung. Die CLI-Tests bleiben unverändert und müssen grün bleiben; sie sind der
Nachweis, dass sich nichts geändert hat.

**Files:**
- Create: `internal/code/query/build.go`, `internal/code/query/check.go`
- Create: `internal/code/query/build_test.go`, `internal/code/query/check_test.go`
- Modify: `internal/cli/graph.go`, `internal/cli/graph_test.go`

**Interfaces:**
- Produces:
  ```go
  type Stats struct {
  	Files    []sourceset.SourceFile
  	Hashes   map[string]string
  	NoSymbol int
  }
  func Extract(root string) (*model.Graph, Stats, error) // was cli.buildGraph
  func Build(root string) (*model.Graph, Stats, error)   // was cli.writeEverything
  type Drift struct {
  	OK       bool     `json:"ok"`
  	Missing  bool     `json:"missing"`
  	Foreign  string   `json:"foreign,omitempty"`
  	Outdated bool     `json:"outdated,omitempty"`
  	Added    []string `json:"added"`
  	Removed  []string `json:"removed"`
  	Changed  []string `json:"changed"`
  	Hidden   int      `json:"hidden,omitempty"`
  }
  func Check(root string) (Drift, error)            // was cli.check
  func (d Drift) Only(keep func(path string) bool) Drift
  func CheckReport(d Drift) string                  // was cli.checkReport
  ```

- [ ] **Step 1: Tests nach `query` umziehen**

`TestBuildGraphFailsWhenTheRootDoesNotExist`, `TestBuildGraphFailsWhenASourceFileCannotBeRead`,
`TestBuildGraphFailsWhenGoModCannotBeRead`, `TestGoModPathsSkipsADirectoryItCannotRead` und
`TestGoModPathsSkipsATestdataDirectory` aus `internal/cli/graph_test.go` nach
`internal/code/query/build_test.go` (`package query`, interner Test) verschieben;
`buildGraph(...)` wird dort `Extract(...)`, `goModPaths` bleibt `goModPaths`. Die Helfer
`repo` und `sample` in `build_test.go` kopieren (die CLI-Tests brauchen ihre eigenen weiter).
Die Testnamen `TestBuildGraph…` in `TestExtract…` umbenennen.

In `internal/code/query/check_test.go` die neue Methode festhalten:

```go
package query

import (
	"strings"
	"testing"
)

func TestOnlyDropsRefusedIDsAndCountsThem(t *testing.T) {
	d := Drift{
		Added:   []string{"lib/a.go#A", "secrets/k.go#K"},
		Removed: []string{"secrets/old.go"},
		Changed: []string{"lib/b.go#B"},
	}
	got := d.Only(func(path string) bool { return strings.HasPrefix(path, "lib/") })

	if len(got.Added) != 1 || got.Added[0] != "lib/a.go#A" {
		t.Errorf("Added = %v", got.Added)
	}
	if len(got.Removed) != 0 {
		t.Errorf("Removed = %v, want none", got.Removed)
	}
	if len(got.Changed) != 1 {
		t.Errorf("Changed = %v", got.Changed)
	}
	if got.Hidden != 2 {
		t.Errorf("Hidden = %d, want 2", got.Hidden)
	}
	if got.OK {
		t.Error("hiding drift must not turn it into OK")
	}
}

func TestOnlyWithNilKeepsEverything(t *testing.T) {
	d := Drift{Added: []string{"x.go#X"}}
	if got := d.Only(nil); len(got.Added) != 1 || got.Hidden != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestCheckReportNamesHiddenDrift(t *testing.T) {
	out := CheckReport(Drift{Changed: []string{"lib/a.go#A"}, Hidden: 3})
	if !strings.Contains(out, "3 more under the area's never globs") {
		t.Errorf("report %q must say that drift was hidden", out)
	}
}
```

- [ ] **Step 2: Laufen lassen, Fehlschlag sehen**

Run: `go test ./internal/code/query/ -count=1`
Expected: Build-Fehler, das Paket hat noch keinen Code.

- [ ] **Step 3: Code verschieben**

`internal/code/query/build.go` (`package query`): `buildStats` wird `Stats` mit den
exportierten Feldern oben, `buildGraph` wird `Extract`, `writeEverything` wird `Build`,
`goModPaths` wandert mit. Die Kommentare wandern wörtlich mit; im Kommentar von `Build` „Both
callers use it -- `graph build` … and the rebuild `graph ask` triggers" ersetzen durch „Every
caller uses it -- `graph build`, and the rebuild a query triggers, on the command line and over
MCP". Paketkommentar:

```go
// Package query is the graph's question layer: build, check and ask, with the
// answers as text a terminal and a tool call both carry.
//
// It exists so the command line and serve reach the same code. Before it, the
// build sequence lived in internal/cli, where serve cannot import it; a second
// copy there would be a second place to forget the sidecar.
package query
```

`internal/code/query/check.go`: `checkResult` wird `Drift` mit dem Feld `Hidden`, `check`
wird `Check` (ruft `Extract`), `checkReport` wird `CheckReport`. Neu:

```go
// Only drops the ids whose path keep refuses and counts them in Hidden. OK
// stays what it was: drift in a hidden file is still drift, and reporting OK
// would be a lie about the graph.
func (d Drift) Only(keep func(path string) bool) Drift {
	if keep == nil {
		return d
	}
	filter := func(ids []string) []string {
		var out []string
		for _, id := range ids {
			if keep(pathOf(id)) {
				out = append(out, id)
			} else {
				d.Hidden++
			}
		}
		return out
	}
	// The closure counts into d, the receiver's copy, which is what is
	// returned; the three calls run before the assignment reads them.
	d.Added, d.Removed, d.Changed = filter(d.Added), filter(d.Removed), filter(d.Changed)
	return d
}

// pathOf is the path part of a node id.
func pathOf(id string) string {
	if i := strings.Index(id, "#"); i >= 0 {
		return id[:i]
	}
	return id
}
```

In `CheckReport` nach der Schleife über die drei Gruppen, vor dem Schlusssatz:

```go
	if d.Hidden > 0 {
		out += fmt.Sprintf("  (%d more under the area's never globs)\n", d.Hidden)
	}
```

- [ ] **Step 4: CLI auf `query` umstellen**

In `internal/cli/graph.go`: `graphBuild` ruft `query.Build`, `report` nimmt `query.Stats`
(Felder groß), `graphCheck` ruft `query.Check` und `query.CheckReport`. `graphAsk` ruft im
Rebuild vorerst `query.Build` (Task 3 stellt es ganz um). `buildStats`, `buildGraph`,
`writeEverything`, `goModPaths`, `checkResult`, `check`, `checkReport` aus `cli` löschen; die
nicht mehr gebrauchten Importe mit ihnen. Das `//coverage:exempt` über `graphCheck` bleibt,
der Satz darin nennt jetzt `query.Drift` statt `checkResult`.

- [ ] **Step 5: Tests laufen lassen**

Run: `go test ./internal/code/query/ ./internal/cli/ -count=1`
Expected: PASS. Jeder CLI-Test, der vorher grün war, ist es noch.

- [ ] **Step 6: Commit**

`refactor(graph): move build and check into internal/code/query` — Absatz: die CLI behält
Flags, Ausgabe und Exit-Codes; `Drift.Only` und `Hidden` sind neu und werden von `serve`
gebraucht.

---

### Task 3: `query.Ask` — und kein erster Graph

**Files:**
- Create: `internal/code/query/ask.go`, `internal/code/query/ask_test.go`
- Modify: `internal/cli/graph.go` (`graphAsk`, `askReport` weg), `internal/cli/graph_test.go`

**Interfaces:**
- Consumes: `Build` aus Task 2, `ask.Options.Keep` aus Task 1.
- Produces:
  ```go
  var ErrNoGraph = errors.New("no graph. Run `loomux graph build` first.")
  type AskOptions struct {
  	Limit     int
  	In        string
  	Source    bool
  	Full      bool
  	NoRefresh bool
  	Keep      func(path string) bool
  }
  func Ask(root, question string, opts AskOptions) (ask.Answer, []string, error)
  func AskReport(a ask.Answer) string
  ```

- [ ] **Step 1: Failing tests schreiben**

`internal/code/query/ask_test.go` (`package query`; nutzt `repo`/`sample` aus
`build_test.go`):

```go
func TestAskWithoutAGraphRefusesAndBuildsNothing(t *testing.T) {
	root := repo(t, sample())

	_, _, err := Ask(root, "run", AskOptions{})
	if !errors.Is(err, ErrNoGraph) {
		t.Fatalf("err = %v, want ErrNoGraph", err)
	}
	if _, err := os.Stat(store.WiringPath(root)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a query must never build a first graph, stat: %v", err)
	}
}

func TestAskAnswersAfterABuild(t *testing.T) {
	root := repo(t, sample())
	if _, _, err := Build(root); err != nil {
		t.Fatal(err)
	}
	a, _, err := Ask(root, "run", AskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(AskReport(a), "lib/lib.go") {
		t.Errorf("report %q must find lib/lib.go", AskReport(a))
	}
}

func TestAskRefreshesADriftedGraphAndSaysSo(t *testing.T) {
	root := repo(t, sample())
	if _, _, err := Build(root); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(root, "lib", "lib.go")
	if err := os.WriteFile(lib, []byte("package lib\n\n// Run does the thing.\nfunc Run() {}\n\nfunc Walk() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, notes, err := Ask(root, "walk", AskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) == 0 || !strings.Contains(notes[0], "rebuilding") {
		t.Errorf("notes = %v, want the rebuild announced", notes)
	}
	if !strings.Contains(AskReport(a), "Walk") {
		t.Errorf("the answer must come from the rebuilt graph: %q", AskReport(a))
	}
}

func TestAskNoRefreshLeavesDriftAlone(t *testing.T) {
	root := repo(t, sample())
	if _, _, err := Build(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "lib.go"), []byte("package lib\n\nfunc Walk() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, notes, err := Ask(root, "walk", AskOptions{NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Errorf("notes = %v, want none without a refresh", notes)
	}
}

func TestAskFallsBackWhenTheSidecarIsMissingAndSaysSo(t *testing.T) {
	root := repo(t, sample())
	if _, _, err := Build(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(lexicon.Path(root)); err != nil {
		t.Fatal(err)
	}
	a, notes, err := Ask(root, "run", AskOptions{NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "no ask index") {
		t.Errorf("notes = %v, want the missing sidecar named", notes)
	}
	if len(a.Hits) == 0 {
		t.Error("the fallback must still answer")
	}
}

func TestAskReportsAnUnreadableGraph(t *testing.T) {
	root := repo(t, sample())
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.WiringPath(root), []byte("{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Ask(root, "run", AskOptions{NoRefresh: true}); err == nil || errors.Is(err, ErrNoGraph) {
		t.Fatalf("err = %v, want the decode error, not ErrNoGraph", err)
	}
}

func TestAskSourceInlinesAndKeepReachesTheRanking(t *testing.T) {
	root := repo(t, sample())
	if _, _, err := Build(root); err != nil {
		t.Fatal(err)
	}
	keep := func(path string) bool { return path != "lib/lib.go" }
	a, _, err := Ask(root, "run", AskOptions{Source: true, Keep: keep})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range a.Hits {
		if h.Path == "lib/lib.go" {
			t.Fatal("Keep must reach ask.Run")
		}
	}
	a, _, _ = Ask(root, "run", AskOptions{Source: true})
	if a.Hits[0].Code == "" {
		t.Error("Source must inline the span")
	}
}

func TestAskReportOnAMissIsTheNote(t *testing.T) {
	if got := AskReport(ask.Answer{Note: "nothing"}); got != "nothing\n" {
		t.Errorf("got %q", got)
	}
}
```

- [ ] **Step 2: Laufen lassen, Fehlschlag sehen**

Run: `go test ./internal/code/query/ -run Ask -count=1`
Expected: Build-Fehler `undefined: Ask`.

- [ ] **Step 3: `ask.go` schreiben**

```go
package query

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/store"
)

// ErrNoGraph is a query against a repository nobody has built.
var ErrNoGraph = errors.New("no graph. Run `loomux graph build` first.")

// AskOptions are one question's knobs. Source and Full decide what the answer
// carries, Keep what it may consider at all.
type AskOptions struct {
	Limit     int
	In        string
	Source    bool
	Full      bool
	NoRefresh bool
	Keep      func(path string) bool
}

// Ask answers one question from the graph of root.
//
// A missing graph is refused before anything else, and nothing is built: the
// reference's reason holds (src/graph/refresh.ts) -- building a whole
// repository under a query is a surprise, and the one case where nobody has
// switched the graph on yet. Over MCP it is worse than a surprise: any model
// that can name a visible area could start a full build on this machine. A
// graph that exists is refreshed as before.
//
// The notes are what the command line writes to stderr and a tool call sends
// as progress: a rebuild, a missing sidecar.
func Ask(root, question string, opts AskOptions) (ask.Answer, []string, error) {
	if _, err := os.Stat(store.WiringPath(root)); errors.Is(err, os.ErrNotExist) {
		return ask.Answer{}, nil, ErrNoGraph
	}
	var notes []string
	if !opts.NoRefresh {
		ask.EnsureFresh(root, golang.Version,
			func() error { _, _, err := Build(root); return err },
			func(s string) { notes = append(notes, s) })
	}
	g, err := store.Read(root)
	if err != nil {
		return ask.Answer{}, notes, err
	}
	ix, err := lexicon.Read(root)
	if err != nil {
		// The sidecar is a cache: without it, tokenize live off the graph. The
		// body text is gone from the written graph, so a body-only word will
		// not be found -- say so rather than answer worse in silence.
		notes = append(notes, fmt.Sprintf("no ask index, ranking on names and paths only: %v", err))
		ix = lexicon.Build(g)
	}
	a := ask.Run(g, ix, question, ask.Options{Limit: opts.Limit, In: opts.In, Keep: opts.Keep})
	if opts.Source {
		ask.Inline(root, &a, opts.Full)
	}
	return a, notes, nil
}
```

`AskReport` ist `askReport` aus `cli/graph.go`, wörtlich verschoben und exportiert
(`strings` wird dafür gebraucht).

**Warum nur `ErrNotExist` beim `Stat`:** Jeder andere Fehler fällt durch zu `store.Read`, das
ihn mit seinem eigenen Text meldet. Ein zweiter Zweig hier wäre auf keiner Plattform
erreichbar und damit ungedeckt.

**Rennen:** Verschwindet der Graph zwischen `Stat` und `Read` (jemand löscht `.loomux/state`),
meldet `store.Read` einen `ErrNotExist`-Fehler statt `ErrNoGraph`. Das ist ein Fehler mit
richtigem Inhalt und kein Grund für einen dritten Zweig.

- [ ] **Step 4: CLI umstellen und ihren Test umkehren**

`graphAsk` in `internal/cli/graph.go` wird:

```go
	answer, notes, err := query.Ask(project, queries[0], query.AskOptions{
		Limit: *limit, In: *in, Source: *source, Full: *full, NoRefresh: *noRefresh,
	})
	// Notices go to stderr, so a piped answer stays an answer.
	for _, n := range notes {
		fmt.Fprintf(stderr, "loomux graph ask: %s\n", n)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph ask: %v\n", err)
		return 1
	}
```

gefolgt vom unveränderten `--json`-Zweig und `fmt.Fprint(stdout, query.AskReport(answer))`.
`askReport` löschen.

In `internal/cli/graph_test.go` `TestGraphAskBuildsWhenNothingIsBuiltYet` ersetzen durch:

```go
func TestGraphAskDoesNotBuildAFirstGraph(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer

	// A fresh clone has no graph. The reference refuses rather than build a
	// whole repository under a question (src/graph/refresh.ts); so does loomux
	// since G3.
	if code := graphCommand([]string{"ask", "run", "--root", root}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "loomux graph build") {
		t.Errorf("stderr %q must point at the command that fixes it", errOut.String())
	}
	if _, err := os.Stat(store.WiringPath(root)); !os.IsNotExist(err) {
		t.Fatalf("ask must not have built the graph: %v", err)
	}
}
```

- [ ] **Step 5: Tests laufen lassen**

Run: `go test ./internal/code/query/ ./internal/cli/ -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

`feat(graph): never build a first graph under a query` — Absatz mit Grafts Begründung und
dem Verweis auf §3.1 des Deltas; nennen, dass `graph ask` ohne Graph jetzt 1 gibt.

---

### Task 4: `mcptools` — `Brain()` und `Graph()`

`Tools()` bleibt in diesem Task **die fünf** Brain-Werkzeuge. Erst Task 6 macht es zu sieben,
zusammen mit der Verdrahtung in `serve` — sonst wiche die Liste der Brücke dazwischen von der
von `serve` ab, und `TestTheToolListCarriesItsCacheHints` fiele.

**Files:**
- Modify: `internal/mcptools/tools.go`, `internal/mcptools/tools_test.go`
- Modify: `internal/serve/brain/tools.go` (`Register`)

**Interfaces:**
- Produces: `func Brain() []*mcp.Tool` (fünf), `func Graph() []*mcp.Tool` (zwei, in der
  Reihenfolge `graph_find_code`, `graph_check_freshness`), `func Tools() []*mcp.Tool`.
  Alle drei teilen dieselben `*mcp.Tool`-Objekte.

- [ ] **Step 1: Failing tests schreiben**

In `internal/mcptools/tools_test.go`:

```go
func TestGraphToolsAreTheTwoInCanonicalOrder(t *testing.T) {
	got := mcptools.Graph()
	want := []string{"graph_find_code", "graph_check_freshness"}
	if len(got) != len(want) {
		t.Fatalf("got %d tools, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i] {
			t.Errorf("tool %d is %q, want %q", i, got[i].Name, want[i])
		}
	}
}

func TestBrainToolsAreTheFive(t *testing.T) {
	want := []string{"brain_search", "brain_catalog", "brain_read", "brain_neighbors", "brain_status"}
	got := mcptools.Brain()
	if len(got) != len(want) {
		t.Fatalf("got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i] {
			t.Errorf("tool %d is %q, want %q", i, got[i].Name, want[i])
		}
	}
}

func TestEveryToolNameIsValidMCP(t *testing.T) {
	valid := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	for _, tool := range append(append([]*mcp.Tool{}, mcptools.Brain()...), mcptools.Graph()...) {
		if !valid.MatchString(tool.Name) {
			t.Errorf("%q is not a valid MCP tool name", tool.Name)
		}
	}
}

func TestFindCodeRequiresScopeAndQueryAndCarriesNoSchemaDefault(t *testing.T) {
	schema := graphSchemaOf(t, "graph_find_code")
	required := schema["required"].([]any)
	if len(required) != 2 || required[0] != "scope" || required[1] != "query" {
		t.Errorf("required = %v, want [scope query]", required)
	}
	props := schema["properties"].(map[string]any)
	for _, field := range []string{"scope", "query", "limit", "full", "in"} {
		p, ok := props[field].(map[string]any)
		if !ok {
			t.Errorf("graph_find_code lacks %s", field)
			continue
		}
		// The reference names its 5 in the description only (tools.ts:51).
		if _, has := p["default"]; has {
			t.Errorf("graph_find_code.%s carries a schema default", field)
		}
	}
}

func TestCheckFreshnessRequiresOnlyScope(t *testing.T) {
	schema := graphSchemaOf(t, "graph_check_freshness")
	required := schema["required"].([]any)
	if len(required) != 1 || required[0] != "scope" {
		t.Errorf("required = %v, want [scope]", required)
	}
	if props := schema["properties"].(map[string]any); len(props) != 1 {
		t.Errorf("properties = %v, want scope alone", props)
	}
}

func graphSchemaOf(t *testing.T, name string) map[string]any {
	t.Helper()
	for _, tool := range mcptools.Graph() {
		if tool.Name != name {
			continue
		}
		raw, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatal(err)
		}
		return schema
	}
	t.Fatalf("no graph tool named %s", name)
	return nil
}
```

(Importe `regexp` und `github.com/modelcontextprotocol/go-sdk/mcp` ergänzen.)

- [ ] **Step 2: Laufen lassen, Fehlschlag sehen**

Run: `go test ./internal/mcptools/ -count=1`
Expected: Build-Fehler `undefined: mcptools.Graph`.

- [ ] **Step 3: Implementieren**

In `internal/mcptools/tools.go`: die Paketvariable `tools` bleibt der eine Speicher; `build()`
hängt an die fünf Brain-Werkzeuge die zwei Graph-Werkzeuge an:

```go
		{
			Name: "graph_find_code",
			Description: "Query one area's code graph in plain words. Returns ranked " +
				"symbols with exact file:line spans and the relevant source inlined.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope": areaScope,
					"query": map[string]any{"type": "string", "description": "what you want to understand, in plain words"},
					"limit": map[string]any{"type": "integer", "description": "max results (default 5)"},
					"full":  map[string]any{"type": "boolean", "description": "inline whole definition spans instead of the capped excerpt"},
					"in":    map[string]any{"type": "string", "description": "narrow to nodes under this path prefix, filtered before scoring"},
				},
				"required": []string{"scope", "query"},
			},
		},
		{
			Name:        "graph_check_freshness",
			Description: "Report whether one area's code graph is in sync with the code (drift check).",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"scope": areaScope},
				"required":   []string{"scope"},
			},
		},
```

mit

```go
	// One area, never "all": a graph belongs to one repository, and answering
	// across several is federation, which this stage does not have.
	areaScope := map[string]any{"type": "string", "description": "the registered area whose code to ask"}
```

Und die Zugriffe:

```go
// brainCount is how many of tools are the brain's; the graph's follow them.
const brainCount = 5

// Brain are the five knowledge tools, the ones serve/brain answers.
func Brain() []*mcp.Tool {
	once.Do(build)
	return tools[:brainCount:brainCount]
}

// Graph are the code-graph tools, the ones serve/graph answers.
func Graph() []*mcp.Tool {
	once.Do(build)
	return tools[brainCount:len(tools):len(tools)]
}

// Tools are every tool, as the bridge lists them and serve registers them.
func Tools() []*mcp.Tool {
	return Brain()
}
```

Den Kommentar über `Tools` ergänzen: „Until stage G3 wires serve/graph, the graph's tools are
not in here: a bridge that listed them would forward calls serve cannot answer." (Task 6
ändert Rumpf und Kommentar.)

In `internal/serve/brain/tools.go`, `Register`: `mcptools.Tools()` → `mcptools.Brain()`.

- [ ] **Step 4: Tests laufen lassen**

Run: `go test ./internal/mcptools/ ./internal/serve/... ./internal/bridge/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

`feat(mcptools): declare the two graph tools beside the five brain tools`.

---

### Task 5: `internal/serve/graph` — die zwei Handler

**Files:**
- Create: `internal/serve/graph/tools.go`, `internal/serve/graph/tools_test.go`

**Interfaces:**
- Consumes: `query.Ask`, `query.AskOptions`, `query.AskReport`, `query.Drift`,
  `query.CheckReport`, `Drift.Only`, `mcptools.Graph()`, `privacy.VisibleAreas`,
  `privacy.Single`, `privacy.IsReadable`.
- Produces:
  ```go
  type Deps struct {
  	RegistryDir string
  	LegacyDir   string
  	Ask         func(root, question string, opts query.AskOptions) (ask.Answer, []string, error)
  	Check       func(root string) (query.Drift, error)
  }
  func Register(server *mcp.Server, channel privacy.Channel, deps Deps)
  ```

- [ ] **Step 1: Failing tests schreiben**

`internal/serve/graph/tools_test.go` (`package graph_test`). Gerüst wie
`internal/serve/brain/tools_test.go` (`connect`, `progressSink`), nur mit
`servegraph.Register`. Dazu ein Registry-Helfer:

```go
// registry writes a registry with two areas into a fresh directory: open,
// which every channel sees, and private, which is local_only. open's manifest
// declares one never glob.
func registry(t *testing.T) (dir, open, private string) {
	t.Helper()
	dir = t.TempDir()
	open, private = t.TempDir(), t.TempDir()
	manifests := map[string]string{
		open:    "[area]\nscope = \"project/open\"\n\n[privacy]\nnever = [\"secrets/**\"]\n",
		private: "[area]\nscope = \"project/private\"\n\n[privacy]\nmode = \"local_only\"\n",
	}
	for root, body := range manifests {
		path := filepath.Join(root, ".loomux", "config.toml")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	reg := "[[area]]\nscope = \"project/open\"\npath = " + strconv.Quote(filepath.ToSlash(open)) + "\n\n" +
		"[[area]]\nscope = \"project/private\"\npath = " + strconv.Quote(filepath.ToSlash(private)) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), []byte(reg), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, open, private
}

// call runs one tool and returns its text and error flag.
func call(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	return res.Content[0].(*mcp.TextContent).Text, res.IsError
}
```

Fakes und die Tests:

```go
// recorder is a fake Ask and Check that remembers what reached it.
type recorder struct {
	root      string
	question  string
	opts      query.AskOptions
	asked     bool
	checked   bool
	answer    ask.Answer
	notes     []string
	askErr    error
	drift     query.Drift
	checkErr  error
	askPanic  any
	checkPanic  any
}

func (r *recorder) deps(registryDir string) servegraph.Deps {
	return servegraph.Deps{
		RegistryDir: registryDir,
		LegacyDir:   registryDir,
		Ask: func(root, question string, opts query.AskOptions) (ask.Answer, []string, error) {
			if r.askPanic != nil {
				panic(r.askPanic)
			}
			r.asked, r.root, r.question, r.opts = true, root, question, opts
			return r.answer, r.notes, r.askErr
		},
		Check: func(root string) (query.Drift, error) {
			if r.checkPanic != nil {
				panic(r.checkPanic)
			}
			r.checked, r.root = true, root
			return r.drift, r.checkErr
		},
	}
}

func sameDir(t *testing.T, got, want string) {
	t.Helper()
	if filepath.Clean(filepath.FromSlash(got)) != filepath.Clean(want) {
		t.Errorf("root = %q, want %q", got, want)
	}
}

func TestTheTwoToolsAreListed(t *testing.T) {
	dir, _, _ := registry(t)
	session := connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir))
	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	got := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		got = append(got, tool.Name)
	}
	// Sorted by the SDK, see TestTheFiveToolsAreListed in serve/brain.
	if strings.Join(got, ",") != "graph_check_freshness,graph_find_code" {
		t.Errorf("tools = %v", got)
	}
}

func TestFindCodeWithoutAScopeIsRefused(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code", map[string]any{"query": "x"})
	if !isError || text != "graph_find_code requires a scope" || r.asked {
		t.Errorf("got %q (isError %v, asked %v)", text, isError, r.asked)
	}
}

func TestFindCodeWithoutAQueryIsRefused(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code", map[string]any{"scope": "project/open"})
	if !isError || text != "graph_find_code requires a query" || r.asked {
		t.Errorf("got %q (isError %v, asked %v)", text, isError, r.asked)
	}
}

func TestFindCodeReachesTheAreaRootWithGraftsDefaults(t *testing.T) {
	dir, open, _ := registry(t)
	r := &recorder{}
	call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "cache"})
	sameDir(t, r.root, open)
	if r.question != "cache" {
		t.Errorf("question = %q", r.question)
	}
	o := r.opts
	if o.Limit != 5 || !o.Source || o.Full || o.In != "" || o.NoRefresh || o.Keep == nil {
		t.Errorf("opts = %+v, want limit 5, source on, full off, refresh on, a Keep", o)
	}
}

func TestFindCodePassesLimitFullAndIn(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	session := connect(t, privacy.ChannelLocal, r.deps(dir))
	call(t, session, "graph_find_code", map[string]any{
		"scope": "project/open", "query": "q", "limit": 3, "full": true, "in": "lib",
	})
	if r.opts.Limit != 3 || !r.opts.Full || r.opts.In != "lib" {
		t.Errorf("opts = %+v", r.opts)
	}
	for _, bad := range []any{0, -2, "seven"} {
		call(t, session, "graph_find_code", map[string]any{"scope": "project/open", "query": "q", "limit": bad})
		if r.opts.Limit != 5 {
			t.Errorf("limit %v became %d, want 5", bad, r.opts.Limit)
		}
	}
}

func TestKeepIsTheAreasNeverGlobs(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "q"})
	if r.opts.Keep("secrets/a.go") {
		t.Error("secrets/a.go lies under the never glob and must be refused")
	}
	if !r.opts.Keep("lib/a.go") {
		t.Error("lib/a.go must be admitted")
	}
}

func TestALocalOnlyAreaDoesNotExistOnTheCloudChannel(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	session := connect(t, privacy.ChannelCloud, r.deps(dir))
	for _, tool := range []string{"graph_find_code", "graph_check_freshness"} {
		text, isError := call(t, session, tool, map[string]any{"scope": "project/private", "query": "q"})
		if !isError || !strings.HasPrefix(text, "unknown scope 'project/private'") {
			t.Errorf("%s: got %q (isError %v)", tool, text, isError)
		}
		if strings.Contains(text, "project/private;") || strings.Contains(text, ", project/private") {
			t.Errorf("%s: the list of known scopes must not name the hidden area: %q", tool, text)
		}
	}
	if r.asked || r.checked {
		t.Error("a hidden area must be refused before anything is read")
	}
}

func TestALocalOnlyAreaIsVisibleOnTheLocalChannel(t *testing.T) {
	dir, _, private := registry(t)
	r := &recorder{}
	call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/private", "query": "q"})
	sameDir(t, r.root, private)
}

func TestAllIsNotAScope(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "all", "query": "q"})
	if !isError || !strings.HasPrefix(text, "unknown scope 'all'") || r.asked {
		t.Errorf("got %q (isError %v, asked %v)", text, isError, r.asked)
	}
}

func TestNotesComeBeforeTheAnswerAndAsProgress(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{answer: ask.Answer{Note: "the answer"}, notes: []string{"3 files moved, rebuilding the graph"}}
	sink := newProgressSink()
	session := connectWith(t, privacy.ChannelLocal, r.deps(dir), sink.options())
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "graph_find_code",
		Meta:      mcp.Meta{"progressToken": "token-1"},
		Arguments: map[string]any{"scope": "project/open", "query": "q"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if text := res.Content[0].(*mcp.TextContent).Text; text != "3 files moved, rebuilding the graph\nthe answer" {
		t.Errorf("text = %q", text)
	}
	if got := sink.next(t); got != "3 files moved, rebuilding the graph" {
		t.Errorf("progress = %q", got)
	}
}

func TestNoGraphIsAnErrorForTheModel(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{askErr: query.ErrNoGraph}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "q"})
	if !isError || !strings.Contains(text, "loomux graph build") {
		t.Errorf("got %q (isError %v)", text, isError)
	}
}

func TestCheckFreshnessRunsWithoutARefreshAndDriftIsNoError(t *testing.T) {
	dir, open, _ := registry(t)
	r := &recorder{drift: query.Drift{Changed: []string{"lib/a.go#A", "secrets/k.go#K"}}}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_check_freshness",
		map[string]any{"scope": "project/open"})
	if isError {
		t.Fatalf("drift is a report, not an error: %q", text)
	}
	sameDir(t, r.root, open)
	if !strings.Contains(text, "lib/a.go#A") || strings.Contains(text, "secrets/k.go#K") {
		t.Errorf("text %q must show lib and hide secrets", text)
	}
	if !strings.Contains(text, "1 more under the area's never globs") {
		t.Errorf("text %q must say that drift was hidden", text)
	}
	if r.asked {
		t.Error("check_freshness must not go through Ask, which refreshes")
	}
}

func TestTheCloudChannelHearsNoCountOfHiddenDrift(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{drift: query.Drift{Changed: []string{"secrets/k.go#K"}}}
	text, isError := call(t, connect(t, privacy.ChannelCloud, r.deps(dir)), "graph_check_freshness",
		map[string]any{"scope": "project/open"})
	if isError {
		t.Fatalf("got an error: %q", text)
	}
	if strings.Contains(text, "never globs") || strings.Contains(text, "secrets/") {
		t.Errorf("the cloud channel must learn nothing about hidden drift: %q", text)
	}
	if !strings.Contains(text, "DRIFT") {
		t.Errorf("hidden drift is still drift, never OK: %q", text)
	}
}

func TestCheckFreshnessOnAMissingGraphIsText(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{drift: query.Drift{Missing: true}}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_check_freshness",
		map[string]any{"scope": "project/open"})
	if isError || !strings.Contains(text, "NO GRAPH") {
		t.Errorf("got %q (isError %v)", text, isError)
	}
}

func TestCheckFreshnessReportsAReadFailure(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{checkErr: errors.New("disk on fire")}
	text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_check_freshness",
		map[string]any{"scope": "project/open"})
	if !isError || text != "disk on fire" {
		t.Errorf("got %q (isError %v)", text, isError)
	}
}

func TestCheckFreshnessWithoutAScopeIsRefused(t *testing.T) {
	dir, _, _ := registry(t)
	text, isError := call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_check_freshness", nil)
	if !isError || text != "graph_check_freshness requires a scope" {
		t.Errorf("got %q (isError %v)", text, isError)
	}
}

func TestAPanicBecomesAnErrorResult(t *testing.T) {
	dir, _, _ := registry(t)
	r := &recorder{askPanic: "boom", checkPanic: "bang"}
	session := connect(t, privacy.ChannelLocal, r.deps(dir))
	text, isError := call(t, session, "graph_find_code", map[string]any{"scope": "project/open", "query": "q"})
	if !isError || !strings.Contains(text, "boom") {
		t.Errorf("find_code: got %q (isError %v)", text, isError)
	}
	text, isError = call(t, session, "graph_check_freshness", map[string]any{"scope": "project/open"})
	if !isError || !strings.Contains(text, "bang") {
		t.Errorf("check_freshness: got %q (isError %v)", text, isError)
	}
}

func TestABrokenRegistryIsAnErrorResult(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), []byte("not toml ["), 0o600); err != nil {
		t.Fatal(err)
	}
	text, isError := call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), "graph_find_code",
		map[string]any{"scope": "project/open", "query": "q"})
	if !isError || text == "" {
		t.Errorf("got %q (isError %v)", text, isError)
	}
}

func TestArgumentsThatAreNotAnObjectEndEmpty(t *testing.T) {
	// The twin of the serve/brain test of that name: broken arguments are a
	// missing scope, not an outage.
	dir, _, _ := registry(t)
	session := connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir))
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "graph_check_freshness", Arguments: []any{1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if text := res.Content[0].(*mcp.TextContent).Text; !res.IsError || text != "graph_check_freshness requires a scope" {
		t.Errorf("got %q (isError %v)", text, res.IsError)
	}
}
```

Die Formen sind gegen `serve/brain/tools_test.go` geprüft: das Progress-Token geht über
`Meta` (`TestBothHintPathsBecomeProgress`), kaputte Argumente sind `[]any{1}`
(`TestArgumentsThatAreNotAnObjectEndEmpty`), und `UnknownScope` setzt den Scope über
`pytext.Repr` in einfache Anführungszeichen. Importe: `context`, `errors`, `os`,
`path/filepath`, `strconv`, `strings`, `testing`, `time`, das SDK, `privacy`, `ask`, `query`,
`servegraph`.

- [ ] **Step 2: Laufen lassen, Fehlschlag sehen**

Run: `go test ./internal/serve/graph/ -count=1`
Expected: Build-Fehler, das Paket existiert nicht.

- [ ] **Step 3: Implementieren**

`internal/serve/graph/tools.go`:

```go
// Package graph registers the code-graph tools on an MCP server.
//
// The same thin layer as serve/brain: one tool call, one query call, and the
// results turned into what MCP has for them. The area comes from the registry
// through the channel's eyes, never from a path the caller names.
package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/mcptools"
)

// Deps are what the tools need. A test replaces Ask and Check and needs no
// tree; serve passes query.Ask and query.Check.
type Deps struct {
	RegistryDir string
	LegacyDir   string
	Ask         func(root, question string, opts query.AskOptions) (ask.Answer, []string, error)
	Check       func(root string) (query.Drift, error)
}

// mcpLimit is the reference's MCP default (src/mcp/tools.ts), not the command
// line's 8.
const mcpLimit = 5

// Register adds the graph tools to server. The channel is the listener's.
func Register(server *mcp.Server, channel privacy.Channel, deps Deps) {
	handlers := map[string]mcp.ToolHandler{
		"graph_find_code":       findCode(channel, deps),
		"graph_check_freshness": checkFreshness(channel, deps),
	}
	for _, tool := range mcptools.Graph() {
		server.AddTool(tool, guarded(handlers[tool.Name]))
	}
}

func findCode(channel privacy.Channel, deps Deps) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := arguments(req)
		area, refusal := resolve("graph_find_code", channel, deps, str(args, "scope"))
		if refusal != nil {
			return refusal, nil
		}
		question := str(args, "query")
		if question == "" {
			return failure("graph_find_code requires a query"), nil
		}
		answer, notes, err := deps.Ask(area.Area.Path, question, query.AskOptions{
			Limit: limit(args), In: str(args, "in"), Source: true, Full: flag(args, "full"),
			Keep: readable(area.Manifest),
		})
		for _, note := range notes {
			report(ctx, req, note)
		}
		if err != nil {
			return failure(withNotes(notes, err.Error())), nil
		}
		return success(withNotes(notes, strings.TrimSuffix(query.AskReport(answer), "\n"))), nil
	}
}

// checkFreshness reports drift without refreshing first: a refresh would make
// it answer about a graph it had just repaired, which is always OK
// (NO_REFRESH_TOOLS in src/mcp/tools.ts).
func checkFreshness(channel privacy.Channel, deps Deps) mcp.ToolHandler {
	return func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		area, refusal := resolve("graph_check_freshness", channel, deps, str(arguments(req), "scope"))
		if refusal != nil {
			return refusal, nil
		}
		drift, err := deps.Check(area.Area.Path)
		if err != nil {
			return failure(err.Error()), nil
		}
		drift = drift.Only(readable(area.Manifest))
		if channel == privacy.ChannelCloud {
			// The count alone tells a remote model that something under the
			// never globs changed. Locally that is the user's own business;
			// on the cloud channel it is a disclosure. The report still says
			// DRIFT rather than OK, because OK would be a lie about the graph.
			drift.Hidden = 0
		}
		return success(strings.TrimSuffix(query.CheckReport(drift), "\n")), nil
	}
}

// resolve turns a scope into the one area the channel may see, before anything
// is read. A hidden area and an unknown one get the same answer: telling a
// cloud caller that an area exists would disclose what local_only hides.
func resolve(tool string, channel privacy.Channel, deps Deps, scope string) (privacy.VisibleArea, *mcp.CallToolResult) {
	if scope == "" {
		return privacy.VisibleArea{}, failure(tool + " requires a scope")
	}
	areas, err := privacy.VisibleAreas(deps.RegistryDir, deps.LegacyDir, scope, channel)
	if err != nil {
		return privacy.VisibleArea{}, failure(err.Error())
	}
	area, err := privacy.Single(areas, scope)
	if err != nil {
		return privacy.VisibleArea{}, failure(err.Error())
	}
	return area, nil
}

// readable is the area's never globs as the predicate query takes.
func readable(manifest *config.Manifest) func(string) bool {
	return func(path string) bool { return privacy.IsReadable(manifest, path) }
}

// withNotes puts the notes in front of the text, as the reference does
// (`${note}\n${res.text}`): a rebuild note explains the answer, so it comes
// first. serve/brain appends its findings instead, because a finding devalues
// an answer; two kinds of note, two places.
func withNotes(notes []string, text string) string {
	if len(notes) == 0 {
		return text
	}
	return strings.Join(notes, "\n") + "\n" + text
}

// guarded turns a panic into a tool error. The reference promises the same
// (callTool never throws); a panic here must not take the listener down.
func guarded(next mcp.ToolHandler) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (res *mcp.CallToolResult, err error) {
		defer func() {
			if r := recover(); r != nil {
				res, err = failure(fmt.Sprintf("internal error: %v", r)), nil
			}
		}()
		return next(ctx, req)
	}
}

func success(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func failure(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

// limit reads the count, and falls back to the reference's MCP default for a
// missing, fractional-to-zero or negative one. JSON numbers arrive as float64.
func limit(args map[string]any) int {
	if n, ok := args["limit"].(float64); ok && n >= 1 {
		return int(n)
	}
	return mcpLimit
}

func flag(args map[string]any, key string) bool {
	value, _ := args[key].(bool)
	return value
}

// arguments, str and report are serve/brain's, repeated rather than shared:
// three small functions do not earn a package, and a shared one would couple
// two tool families that otherwise know nothing of each other.
func arguments(req *mcp.CallToolRequest) map[string]any {
	var args map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return map[string]any{}
	}
	return args
}

func str(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func report(ctx context.Context, req *mcp.CallToolRequest, message string) {
	token := req.Params.GetProgressToken()
	if token == nil {
		return
	}
	_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		ProgressToken: token,
		Message:       message,
	})
}
```

- [ ] **Step 4: Tests laufen lassen**

Run: `go test ./internal/serve/graph/ -count=1 -coverprofile=cov.out` und
`go tool cover -func=cov.out`
Expected: PASS, jede Funktion 100 %.

- [ ] **Step 5: Commit**

`feat(serve): answer graph_find_code and graph_check_freshness`.

---

### Task 6: `serve` verdrahten — sieben Werkzeuge

**Files:**
- Modify: `internal/serve/serve.go` (`handlers`)
- Modify: `internal/mcptools/tools.go` (`Tools`), `internal/mcptools/tools_test.go`
- Test: `internal/serve/serve_test.go`

**Interfaces:**
- Consumes: `servegraph.Register`, `servegraph.Deps`, `query.Ask`, `query.Check`.

- [ ] **Step 1: Failing tests schreiben**

In `internal/mcptools/tools_test.go` `TestToolsAreTheFiveInCanonicalOrder` ersetzen:

```go
func TestToolsAreTheSevenInCanonicalOrder(t *testing.T) {
	got := mcptools.Tools()
	want := []string{
		"brain_search", "brain_catalog", "brain_read", "brain_neighbors", "brain_status",
		"graph_find_code", "graph_check_freshness",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d tools, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i] {
			t.Errorf("tool %d is %q, want %q", i, got[i].Name, want[i])
		}
	}
}
```

In `internal/serve/serve_test.go` einen Ende-zu-Ende-Test über HTTP:

```go
// graphRegistry registers one Go repository as project/code, and a second one
// as project/private with local_only, and builds both graphs.
func graphRegistry(t *testing.T, dir string) {
	t.Helper()
	areas := map[string]string{
		"project/code":    "",
		"project/private": "\n[privacy]\nmode = \"local_only\"\n",
	}
	var reg strings.Builder
	for scope, privacyBlock := range areas {
		root := t.TempDir()
		files := map[string]string{
			"go.mod":               "module example.com/repo\n",
			"lib/lib.go":           "package lib\n\n// Run does the thing.\nfunc Run() {}\n",
			".loomux/config.toml":  "[area]\nscope = \"" + scope + "\"\n" + privacyBlock,
		}
		for rel, body := range files {
			abs := filepath.Join(root, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if _, _, err := query.Build(root); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&reg, "[[area]]\nscope = %q\npath = %q\n\n", scope, filepath.ToSlash(root))
	}
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), []byte(reg.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestFindCodeAnswersOverHTTPAndHidesALocalOnlyAreaFromTheCloud(t *testing.T) {
	dir := t.TempDir()
	graphRegistry(t, dir)
	state, _ := start(t, dir)

	local := connect(t, state.Local, nil)
	res, err := local.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "graph_find_code", Arguments: map[string]any{"scope": "project/code", "query": "run"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if text := res.Content[0].(*mcp.TextContent).Text; res.IsError || !strings.Contains(text, "lib/lib.go") {
		t.Fatalf("local answer %q (isError %v) must find lib/lib.go", text, res.IsError)
	}

	cloud := connect(t, state.Cloud, nil)
	res, err = cloud.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "graph_find_code", Arguments: map[string]any{"scope": "project/private", "query": "run"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if text := res.Content[0].(*mcp.TextContent).Text; !res.IsError || !strings.HasPrefix(text, "unknown scope") {
		t.Fatalf("cloud answer %q (isError %v) must not know project/private", text, res.IsError)
	}
}
```

(Importe `fmt`, `os`, `path/filepath`, `strings`, `github.com/xidus90/loomux/internal/code/query`
ergänzen, soweit nicht vorhanden. Die Reihenfolge der Map ist egal: jeder Bereich hat seinen
eigenen `scope`. `%q` in TOML: Go-Anführung ist für diese Pfade gültiges TOML, weil
`filepath.ToSlash` keine Backslashes lässt; `writeRegistryWithOneArea` nutzt dasselbe mit
`strconv.Quote`.)

- [ ] **Step 2: Laufen lassen, Fehlschlag sehen**

Run: `go test ./internal/mcptools/ ./internal/serve/ -count=1 -run "Seven|FindCode"`
Expected: FAIL — `Tools()` liefert fünf, und `graph_find_code` ist unbekannt.

- [ ] **Step 3: Implementieren**

`mcptools.Tools()`:

```go
// Tools are every tool, as the bridge lists them and serve registers them:
// the brain's five, then the graph's two.
func Tools() []*mcp.Tool {
	once.Do(build)
	return tools
}
```

`serve.go`, in `handlers` nach `servebrain.Register(...)`:

```go
	servegraph.Register(server, name, servegraph.Deps{
		RegistryDir: opts.RegistryDir,
		LegacyDir:   opts.LegacyDir,
		Ask:         query.Ask,
		Check:       query.Check,
	})
```

mit den Importen `servegraph "github.com/xidus90/loomux/internal/serve/graph"` und
`"github.com/xidus90/loomux/internal/code/query"`.

Zwei Kommentare nennen danach die falsche Zahl und werden mitgezogen:
`internal/bridge/bridge.go` über `setCacheable` („Five static tools make that free" → „Seven
static tools make that free") und `internal/bridge/bridge_test.go` über `connectService`
(„registers the five tools the way serve does" → „registers the tools the way serve does").

- [ ] **Step 4: Tests laufen lassen — alles, was Werkzeuge zählt**

Run: `go test ./internal/mcptools/ ./internal/serve/... ./internal/bridge/ ./cmd/loomux/ -count=1`
Expected: PASS, darunter `TestTheToolListCarriesItsCacheHints` (7 == 7) und
`TestStartDoesNoWorkInPackageInit` (keine Paketinitialisierung über 500 Allokationen).
`extract/golang` und `go/parser` sind seit G2a gelinkt, weil `internal/cli` sie importiert;
G3 fügt dem Binary keine Abhängigkeit hinzu, der Test bleibt aus demselben Grund grün wie auf
`11222c8`.

Fällt `TestStartDoesNoWorkInPackageInit`: anhalten und melden, nicht die Grenze anheben.

- [ ] **Step 5: Commit**

`feat(serve): wire the graph tools into both listeners`.

---

### Task 7: Doku und Messungen

**Files:**
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md`
- Modify: `README.md`, `README.de.md`
- Modify: `docs/.superpowers/specs/2026-09-14-loomux-code-graph-design.md` (Kopfzeile „Stand")

- [ ] **Step 1: Messen**

Vorher-Binary aus `11222c8` bauen und nachher-Binary aus `HEAD`, beide in den Scratchpad:

```bash
git worktree add --detach "$SCRATCH/g3-before" 11222c8
go -C "$SCRATCH/g3-before" build -o "$SCRATCH/loomux-before.exe" ./cmd/loomux
go build -o "$SCRATCH/loomux-after.exe" ./cmd/loomux
```

(`$SCRATCH` ist der Sitzungs-Scratchpad.) Dann in PowerShell, je Binary:

```powershell
$runs = 1..20 | ForEach-Object { (Measure-Command { & $bin --version | Out-Null }).TotalMilliseconds }
($runs | Sort-Object)[10]
```

Kalt = erster Lauf nach dem Bau, warm = Median der 20. Dazu die Dateigröße beider Binaries.
Das ist die **Kontrolle**: G3 linkt nichts Neues (`extract/golang` hängt seit G2a über
`internal/cli` im Binary), erwartet ist gleich innerhalb des Rauschens. Weicht es deutlich ab,
anhalten und melden.

Die neue Zahl ist die Antwortzeit, gemessen auf dem loomux-Hauptcheckout
(`C:/Users/micro/Documents/#GIT/loomux`, Scope `project/loomux` in `registry.toml`), dessen
Graph vorher einmal mit `loomux-after.exe graph build --root <checkout>` gebaut wird:

- **CLI:** `loomux-after.exe graph ask "run" --root <checkout>`, 10 warme Läufe, Median.
- **MCP über die Brücke:** ein Wegwerf-Programm in einem nicht committeten Unterordner
  `scratch-bench/` dieses Worktrees (`package main`, dasselbe Modul, `go run ./scratch-bench`).
  Es verbindet sich mit
  `mcp.NewClient(&mcp.Implementation{Name: "bench", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: exec.Command(bin, "mcp")}, nil)`
  — `CommandTransport` gibt es im SDK v1.8.0 (`mcp/cmd.go`) —, ruft elfmal `graph_find_code`
  mit `{"scope": "project/loomux", "query": "run"}` und misst jeden Aufruf mit `time.Since`.
  Der erste Aufruf kann ein `serve` starten und wird getrennt als „kalt" notiert; der Median
  der übrigen zehn ist „warm". Lief vorher kein `serve`, danach `loomux-after.exe serve stop`.
  `scratch-bench/` danach löschen.

Danach den Vorher-Worktree entfernen: `git worktree remove "$SCRATCH/g3-before"`.

- [ ] **Step 2: Benchmarks eintragen**

Je einen datierten Abschnitt ans Ende von `docs/en/benchmarks.md` und `docs/de/benchmarks.md`
(Format der vorhandenen Einträge übernehmen): Datum und Uhrzeit, was gemessen wurde, Basis
`11222c8` gegen `HEAD`, kalt und warm, Binärgröße, Antwortzeit CLI gegen MCP. Nur gemessene
Zahlen.

- [ ] **Step 3: CLI-Referenz**

In beiden Sprachen:
- bei `serve`/`mcp`: die zwei Werkzeuge mit ihren Parametern (Tabelle aus §3.3 des Deltas) und
  der Sichtbarkeitsregel in einem Satz;
- bei `graph ask`: „Without a graph, exits 1 and points at `loomux graph build`; a query never
  builds a first graph." bzw. deutsch sinngleich.

- [ ] **Step 4: READMEs**

`README.md` und `README.de.md`: Werkzeugliste um die zwei `graph_*` ergänzen, Stufenstand G3
als umgesetzt (G4 und G5 offen).

- [ ] **Step 5: Säule-3-Spec**

Kopfzeile „Stand" um „G3 umgesetzt 2026-MM-TT, verengt durch
[`2026-09-18-loomux-code-g3-delta.md`](2026-09-18-loomux-code-g3-delta.md)" ergänzen (Datum
des tatsächlichen Abschlusses).

- [ ] **Step 6: Commit**

`docs(graph): record G3 in the references and measure its start floor`.

---

### Task 8: Das ganze Tor, die Mutationsrunde, die Paritätsakte

**Files:**
- Create: `docs/.superpowers/parity/code-g3.md`

- [ ] **Step 1: Das ganze Tor von Hand**

```bash
gofmt -l cmd internal
go vet ./...
go test ./... -count=1 -covermode=set "-coverpkg=github.com/xidus90/loomux/..." -coverprofile=coverage.out
go run ./cmd/loomux dev covergate --profile coverage.out
go build -o bin/loomux.exe ./cmd/loomux
```

Expected: `gofmt` und `vet` still, alle Pakete `ok`, `covergate` Exit 0.

- [ ] **Step 2: Mutationsrunde**

```bash
go run ./cmd/loomux dev mutants internal/code/query internal/serve/graph
```

Jeden Überlebenden gegen den Code nachrechnen: äquivalent (mit Begründung aus dem Code) oder
ein fehlender Test (dann Test schreiben, Runde wiederholen).

- [ ] **Step 3: Paritätsakte schreiben**

`docs/.superpowers/parity/code-g3.md` im Format von `parity/code-g2b.md`: Kopf mit Quelle
(`src/mcp/tools.ts`, `src/graph/refresh.ts`, `src/context/check.ts`), Spec, Plan, Regel
„jede Zeile braucht eine Freigabe des Nutzers". Mindestens diese Verfügungen, je mit Ort in
Graft und in loomux:

| Punkt | Graft | loomux | Art |
|---|---|---|---|
| Werkzeugnamen | `graft_find_code`, `graft_check_freshness` + Aliase | `graph_find_code`, `graph_check_freshness`, keine Aliase | Abweichung: keine Altnamen zu bedienen |
| `scope` | fehlt, ein Repo je Server | Pflicht | Zusatz (§3.3) |
| `limit` Typ | `number` | `integer` | Abweichung: loomux' Schemas nutzen `integer` |
| `limit` ≤ 0 | übernommen wie gegeben | fällt auf 5 | Abweichung, begründen |
| Hinweis vor Text | ja | ja | gleich |
| Kein Erstbau | ja | ja, auch CLI | gleich; CLI ist Berichtigung an G2 |
| `.context/`-Bericht in `check_freshness` | ja | entfällt | keine `.context/`-Schicht |
| Worktree-Saat (`seedUnderLock`) | ja | nein | offen (§8 Delta) |
| `NeverGlobs`/`local_only` | — | ja | Zusatz (§3.4) |
| Überlebende Mutanten | — | je eine Zeile | aus Step 2 |

Stand-Zeile oben: „offen, wartet auf Freigabe".

- [ ] **Step 4: Commit**

`docs(parity): list what G3 keeps and changes against the reference`.

- [ ] **Step 5: Übergabe**

Dem Nutzer melden: Tor-Ausgabe, Zahl der Commits, Überlebende, und dass die Paritätsakte
seine Freigabe braucht. **Nicht pushen, nicht mergen.** Reihenfolge bleibt: erst `sdd-1b-2`
nach `master`, dann `code-g3`.
