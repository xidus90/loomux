# Graphansicht der Weboberfläche: Umsetzungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die Weboberfläche zeigt je Projekt einen lesenden Graphen. Den
Code-Graphen fasst sie nach Verzeichnissen zusammen, die Dokumente des Wikis
sind eigene Knoten, und Kanten führen von einem Dokument zu dem Code, den es
nennt. Diese Verweise erkennt der Wiki-Indexer und speichert sie in
`graph.json`.

**Architecture:** Das Paket `internal/brain/graph` erkennt Codeverweise und
schreibt sie als `code_refs` nach `graph.json`; `internal/brain/index` füllt
sie beim Lesen jedes Dokuments. Ein neues Paket `internal/web/graphview`
fasst Code-Graph und Wiki-Graph ohne HTTP zu einer Ansicht zusammen.
`internal/web` liefert diese Ansicht als JSON, dazu eine Seite mit einer
JavaScript-Insel (force-graph), die nur zeichnet.

**Tech Stack:** Go (stdlib `net/http`, `html/template`, `embed`), force-graph
(UMD-Bündel, eingebettet), schlankes JavaScript ohne Build.

**Spec:** `docs/.superpowers/specs/2026-11-15-web-graphansicht-design.md`

## Global Constraints

- Jede Insel holt ihre Daten über einen JSON-Endpunkt, nie aus dem gerenderten HTML.
- Logik, die Tests braucht (Zusammenfassen, Auflösen, Filtern der Datenmenge), liegt in Go, nicht in JavaScript.
- Keine Bibliothek wird zur Laufzeit aus dem Netz geladen.
- Die CSP `default-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'` (`internal/web/middleware.go`) bleibt unverändert.
- Ein alter `graph.json` ohne `code_refs` liest sich wie bisher.
- 100 % Coverage je Funktion; Ausnahme nur mit `//coverage:exempt <reason>` direkt über `func`.
- Code, Kommentare, Bezeichner, Commits englisch; Doku deutsch. Kein `Co-Authored-By` im Commit.
- Neue Version von force-graph: neueste, exakt gepinnt, vorher Pflegestand neu prüfen (Stand 2026-11-14: 1.52.0).
- Texte der Oberfläche englisch.

## Review Focus

1. Ein Projekt ohne `wiring.json` oder ohne `graph.json`: Die Seite zeigt das Vorhandene und nennt den fehlenden Befehl (`loomux graph build`, `loomux reindex`), statt mit 500 zu antworten. Getestet in Task 6.
2. `open` mit einem Pfad, den es nicht gibt, oder mit `..`: Er wird ignoriert, kein Fehler und kein Zugriff auf das Dateisystem. Getestet in Task 3.
3. Ein Codeverweis mit falscher Groß- und Kleinschreibung (`Internal/Web`) unter Windows: kein Verweis. Getestet in Task 1.
4. `wiring.json` wird nach dem ersten Laden neu gebaut: Die nächste Anfrage sieht den neuen Graphen. Getestet in Task 6.
5. Eine Suche ohne Treffer und eine Suche mit leerem `q`: leere Liste, Status 200. Getestet in Task 5.

---

### Task 1: Codeverweise erkennen

**Files:**
- Create: `internal/brain/graph/coderefs.go`
- Test: `internal/brain/graph/coderefs_test.go`

**Interfaces:**
- Produces:
  - `type CodeRef struct { From string \`json:"from"\`; Path string \`json:"path"\`; Line int \`json:"line"\`; End int \`json:"end"\` }`
  - `func CandidateRefs(docRel, text string) []CodeRef`: Kandidaten ohne Existenzprüfung, `From` = `docRel`.
  - `func ExistsExact(root, rel string) bool`

- [ ] **Step 1: Write the failing tests**

```go
func TestCandidateRefs_BacktickSpansAreRootRelative(t *testing.T) {
	got := graph.CandidateRefs("topics/x.md", "see `internal/hooks/guard.go:67` and `internal/web`")
	want := []graph.CodeRef{
		{From: "topics/x.md", Path: "internal/hooks/guard.go", Line: 67},
		{From: "topics/x.md", Path: "internal/web"},
	}
	// compare with reflect.DeepEqual
}
func TestCandidateRefs_LineRange(t *testing.T)            // `a/b.go:3-9` -> Line 3, End 9
func TestCandidateRefs_FencedBlocksDoNotCount(t *testing.T) // ``` and ~~~ blocks contribute nothing
func TestCandidateRefs_NeedsASlash(t *testing.T)         // `go.mod`, `docs`, `.` -> none
func TestCandidateRefs_RefusesEscapesAndPrivateDirs(t *testing.T) // `../x/y`, `/abs/p`, `.git/config`, `.loomux/state/x` -> none
func TestCandidateRefs_MarkdownLinksResolveFromTheDocument(t *testing.T) {
	// doc "docs/wiki/topics/x.md", link "[g](../../../internal/a.go)" -> Path "internal/a.go"
	// a link climbing past the root -> none; a link to an http URL -> none
}
func TestCandidateRefs_ProseInBackticksIsNoPath(t *testing.T) // `loomux graph build`, `a b/c` -> none
func TestExistsExact(t *testing.T) {
	// t.TempDir with internal/web/a.go
	// "internal/web/a.go" true, "internal/web" true, "Internal/web" false,
	// "internal/web/missing.go" false
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/brain/graph/ -run 'CandidateRefs|ExistsExact'`
Expected: FAIL, `undefined: graph.CandidateRefs`

- [ ] **Step 3: Implement `CandidateRefs` and `ExistsExact` in `coderefs.go`**

Codeblöcke werden vorher entfernt (Zeilen ab ```` ``` ```` oder `~~~` bis zur
schließenden Marke). Der Inhalt eines Spans muss als Ganzes auf
`^[A-Za-z0-9_.\-/]+(:\d+(-\d+)?)?$` passen und einen `/` enthalten; ein
abschließender `/` fällt weg. Markdown-Links lösen über `path.Join(path.Dir(docRel), target)`
auf; ein Ergebnis, das mit `..` beginnt, entfällt. `ExistsExact` geht die
Segmente durch und vergleicht jedes mit `os.ReadDir` des Elternordners auf
exakte Gleichheit des Namens.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/brain/graph/ -run 'CandidateRefs|ExistsExact'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/brain/graph/coderefs.go internal/brain/graph/coderefs_test.go
git commit -m "feat(brain): recognise code paths a document names"
```

### Task 2: `code_refs` in `graph.json`

**Files:**
- Modify: `internal/brain/graph/model.go` (`Graph`, `Document`)
- Modify: `internal/brain/graph/render.go` (`RenderGraph`)
- Modify: `internal/brain/graph/read.go` (nur falls die Prüfung in `ParseGraph` ein fehlendes Feld ablehnt)
- Modify: `internal/brain/index/document.go` (`Document`, `ReadDocument`)
- Modify: `internal/brain/index/reindex.go` (`toGraphDocs`)
- Modify: `internal/brain/index/staging.go:100`
- Test: `internal/brain/graph/render_test.go`, `internal/brain/graph/read_test.go`, `internal/brain/index/document_test.go`, Test des Stagings dort, wo `RenderGraph` heute getestet wird

**Interfaces:**
- Consumes: `CandidateRefs`, `ExistsExact`, `CodeRef` (Task 1)
- Produces:
  - `graph.Graph.CodeRefs []CodeRef \`json:"code_refs"\``
  - `graph.Document.CodeRefs []CodeRef`
  - `index.Document.CodeRefs []graph.CodeRef`

- [ ] **Step 1: Write the failing tests**

```go
func TestRenderGraph_WritesCodeRefsSortedAndOnce(t *testing.T) {
	// two docs, b.md names internal/x.go twice, a.md names internal/y.go:3
	// decoded Graph.CodeRefs == [{a.md internal/y.go 3 0} {b.md internal/x.go 0 0}]
}
func TestRenderGraph_NoRefsIsAnEmptyList(t *testing.T) // JSON contains `"code_refs": []`
func TestParseGraph_WithoutCodeRefsReadsAsBefore(t *testing.T) // fixture without the key -> no error, CodeRefs empty
func TestReadDocument_KeepsOnlyRefsThatExist(t *testing.T) {
	// root with internal/a.go; doc names `internal/a.go` and `internal/gone.go`
	// doc.CodeRefs == [{From: rel, Path: "internal/a.go"}]
}
func TestStaging_ReadOnlyAreaWritesNoCodeRefs(t *testing.T) // read-only area -> code_refs []
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/brain/graph/ ./internal/brain/index/`
Expected: FAIL, `CodeRefs undefined`

- [ ] **Step 3: Implement**

`ReadDocument` setzt `CodeRefs` aus `CandidateRefs(rel, text)`, gefiltert mit
`ExistsExact(rootDir, ref.Path)`. `toGraphDocs` reicht das Feld durch.
`staging.go` leert `CodeRefs` aller Dokumente, wenn `area.ReadOnly` gilt.
`RenderGraph` entfernt Dubletten (gleiches From, Path, Line, End), sortiert
nach From, Path, Line, End und schreibt nie `null`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/brain/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/brain/graph internal/brain/index
git commit -m "feat(brain): store the code a document names in graph.json"
```

### Task 3: Code-Graph nach Verzeichnissen zusammenfassen

**Files:**
- Create: `internal/web/graphview/view.go`, `internal/web/graphview/code.go`
- Test: `internal/web/graphview/code_test.go`

**Interfaces:**
- Consumes: `model.Graph`, `model.FileSpans`, `pagerank.Prepare`, `pagerank.Rank`
- Produces:

```go
type Node struct {
	ID      string  `json:"id"`      // "dir:internal/web", "file:internal/web/a.go", symbol NodeID, "doc:topics/x.md"
	Label   string  `json:"label"`
	Kind    string  `json:"kind"`    // "dir" | "file" | "symbol" | "doc"
	Path    string  `json:"path"`
	Group   string  `json:"group"`   // colour key: top directory, or top folder of a document
	Size    float64 `json:"size"`
	Members int     `json:"members"`
	Wiki    bool    `json:"wiki,omitempty"`    // document inside the wiki folder
	Missing []string `json:"missing,omitempty"` // a document's refs not in the code graph
}
type Edge struct {
	Source string         `json:"source"`
	Target string         `json:"target"`
	Kind   string         `json:"kind"`   // "code" | "link" | "ref"
	Counts map[string]int `json:"counts"` // "calls/extracted" -> n; "link" and "ref" use their kind as key
}
type View struct {
	Nodes []Node   `json:"nodes"`
	Edges []Edge   `json:"edges"`
	Open  []string `json:"open"` // directories and files shown expanded, after auto-open and cleaning
}
const DefaultLimit = 150
func Code(g *model.Graph, open []string, limit int) View
```

- [ ] **Step 1: Write the failing tests**

Hilfsfunktion im Test: ein kleiner `model.Graph` mit
`cmd/loomux/main.go`, `internal/a/x.go` (Symbole `A1`, `A2`),
`internal/b/y.go` (Symbol `B1`), Kanten `A1 calls B1` (extracted),
`A2 calls B1` (inferred), `x.go contains A1`, `A1 imports npm:lodash`.

```go
func TestCode_StartsAtDirectoriesAndOpensUnderTheLimit(t *testing.T)
	// limit 3: Open == ["internal"], nodes dir:cmd, dir:internal/a, dir:internal/b
func TestCode_EdgesCountRelationAndConfidence(t *testing.T)
	// edge dir:internal/a -> dir:internal/b, Counts {"calls/extracted":1,"calls/inferred":1}
func TestCode_ContainsAndForeignTargetsMakeNoEdge(t *testing.T)
func TestCode_EdgesInsideOneNodeVanish(t *testing.T)
func TestCode_OpenFileShowsItsSymbols(t *testing.T)
	// open ["internal","internal/a","internal/a/x.go"] -> symbol nodes A1, A2
func TestCode_SizeIsSummedRank(t *testing.T) // dir:internal/b larger than dir:cmd
func TestCode_UnknownOrEscapingOpenIsIgnored(t *testing.T)
	// open ["nope", "../x", "internal/../.."] -> same View as open nil
func TestCode_NilGraphIsEmpty(t *testing.T)
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/web/graphview/`
Expected: FAIL, `undefined: graphview.Code`

- [ ] **Step 3: Implement `Code`**

Jeder Knoten des Graphen gehört über `Path` zum tiefsten sichtbaren
Vorfahren: Ist ein Verzeichnis offen, zeigt es seine Kinder; eine offene
Datei zeigt ihre Symbole. Auto-Open (nur wenn `open` leer ist) öffnet
breitenweise das Verzeichnis mit den meisten Mitgliedern, solange die Zahl
sichtbarer Knoten danach `limit` nicht übersteigt. Kanten mit
`Relation.IsWalk()` werden auf ihre sichtbaren Enden abgebildet; gleiche
Enden entfallen. Der Rang kommt aus `pagerank.Rank(pagerank.Prepare(g, nil), seed, Options{})`,
wobei `seed` jeden Knoten mit 1 gewichtet; die Größe ist die Summe über die
Mitglieder. Die Ausgabe ist sortiert, nach ID der Knoten und nach Source und
Target der Kanten.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/web/graphview/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/web/graphview
git commit -m "feat(ui): fold the code graph into directories for the graph view"
```

### Task 4: Wiki-Knoten und Brücken

**Files:**
- Create: `internal/web/graphview/wiki.go`
- Test: `internal/web/graphview/wiki_test.go`

**Interfaces:**
- Consumes: `View`, `Code` (Task 3), `graph.Graph`, `graph.CodeRef` (Task 2), `model.FileSpans`, `model.Enclosing`
- Produces: `func Build(code *model.Graph, wiki *graph.Graph, wikiPrefix string, open []string, limit int) View`. `wikiPrefix` ist relativ zur Wurzel des Bereichs, mit Schrägstrichen und ohne abschließenden `/`. `""` heißt: Der Bereich hat keinen Wiki-Ordner.

- [ ] **Step 1: Write the failing tests**

Testgraph aus Task 3, dazu ein Wiki mit `docs/wiki/topics/x.md`
(Verweise `internal/b`, `internal/a/x.go:5`, `ci/run.ps1`,
`docs/de/y.md`), `docs/wiki/topics/z.md` (Link auf `x.md`),
`docs/de/y.md`, `docs/.superpowers/p.md` (Verweis `internal/b/y.go`).
`wikiPrefix` = `docs/wiki`.

```go
func TestBuild_EveryDocumentIsANodeMarkedWiki(t *testing.T)
	// x.md, z.md Wiki true; y.md, p.md Wiki false; Group = top folder under the area
func TestBuild_LinksBetweenDocumentsAreEdges(t *testing.T) // z.md -> x.md, Kind "link"
func TestBuild_RefToAClosedDirectory(t *testing.T)          // x.md -> dir:internal/b, Kind "ref"
func TestBuild_RefWithLineHitsTheEnclosingSymbolWhenTheFileIsOpen(t *testing.T)
	// open includes internal/a/x.go -> edge x.md -> symbol at line 5
func TestBuild_RefToAnOpenDirectoryFansOutToItsVisibleChildren(t *testing.T)
func TestBuild_RefToADocumentIsALink(t *testing.T)          // x.md -> y.md, Kind "link"
func TestBuild_RefOutsideTheCodeGraphIsListedAsMissing(t *testing.T) // x.md Missing ["ci/run.ps1"]
func TestBuild_NoWikiPrefixMarksEveryDocumentWiki(t *testing.T)
func TestBuild_NilWikiIsCodeOnly(t *testing.T)
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/web/graphview/ -run Build`
Expected: FAIL, `undefined: graphview.Build`

- [ ] **Step 3: Implement `Build` in `wiki.go`**

`Build` ruft `Code` und hängt Dokumente und Kanten an. Ein Verweis findet
seinen sichtbaren Knoten über denselben Vorfahren wie in `Code`. Ein offenes
Verzeichnis fächert zu seinen sichtbaren Kindern auf, die Code enthalten.
Mit Zeile in einer offenen Datei löst `model.Enclosing` auf. Eine Obergrenze
für Dokumentknoten wird nicht eingebaut: Die Insel blendet die Dokumente
außerhalb des Wikis standardmäßig aus (Task 8). Wird die Menge zum Problem,
kommt der Deckel später.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/web/graphview/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/web/graphview
git commit -m "feat(ui): join wiki documents and their code references into the graph view"
```

### Task 5: Suche

**Files:**
- Create: `internal/web/graphview/search.go`
- Test: `internal/web/graphview/search_test.go`

**Interfaces:**
- Consumes: Testgraphen aus Task 3 und 4
- Produces:

```go
type Hit struct {
	ID    string   `json:"id"`   // node ID as Build would emit it once Open is applied
	Label string   `json:"label"`
	Kind  string   `json:"kind"`
	Open  []string `json:"open"` // the open list under which ID is visible
}
const MaxHits = 20
func Search(code *model.Graph, wiki *graph.Graph, q string) []Hit
```

- [ ] **Step 1: Write the failing tests**

```go
func TestSearch_SymbolNeedsItsDirectoriesAndFileOpen(t *testing.T)
	// q "a1" -> Hit{ID: "<A1 NodeID>", Open: ["internal","internal/a","internal/a/x.go"]}
func TestSearch_MatchesFilesDirectoriesAndDocuments(t *testing.T) // case-insensitive substring on name and path, title for documents
func TestSearch_EmptyQueryAndNoMatchAreEmpty(t *testing.T)        // non-nil, length 0
func TestSearch_CapsAtMaxHits(t *testing.T)
```

- [ ] **Step 2: Run, expect FAIL** — `go test ./internal/web/graphview/ -run Search`
- [ ] **Step 3: Implement `Search`**: exakte Namenstreffer zuerst, dann nach Pfad sortiert.
- [ ] **Step 4: Run, expect PASS** — `go test ./internal/web/graphview/`
- [ ] **Step 5: Commit**

```bash
git add internal/web/graphview
git commit -m "feat(ui): search the graph view"
```

### Task 6: Routen, Seite und Cache

**Files:**
- Create: `internal/web/graph.go`, `internal/web/assets/graph.html`
- Modify: `internal/web/server.go` (Feld `codes`, Routen), `internal/web/templates.go` (`pageNames` += `"graph"`), `internal/web/assets/layout.html` (Link „Graph“ je Projekt neben dem Einstellungslink), `internal/web/assets/settings.html` (Link zur Graphseite), Vorlage der Kachel „Index state“ (Link je Zeile)
- Test: `internal/web/graph_test.go`

**Interfaces:**
- Consumes: `graphview.Build`, `graphview.Search`, `graphview.DefaultLimit`, `store.Read`, `store.WiringPath`, `graph.ReadGraph`, `config.ReadRegistry`, `index.OwnWikiPrefix`, `(*server).target`
- Produces:
  - `GET /graph/{id}`: Seite `graph`. Die Daten der Vorlage sind nur Projekt-ID und Name, keine Graphdaten.
  - `GET /graph/{id}/data?open=…&open=…` → JSON `{"view": View, "problems": []string}`
  - `GET /graph/{id}/search?q=` → JSON `[]Hit`
  - `type codeCache struct` mit `func (c *codeCache) get(root string) (*model.Graph, error)`: neu lesen, wenn sich die Änderungszeit von `store.WiringPath(root)` ändert; ein Mutex.

- [ ] **Step 1: Write the failing tests** (Muster der bestehenden `httptest`-Tests mit Sitzung)

```go
func TestGraphPage_RendersWithoutGraphData(t *testing.T)   // 200, body has no "nodes"
func TestGraphData_ServesTheView(t *testing.T)            // project with wiring.json and graph.json -> nodes of both, Kind "ref" edge
func TestGraphData_MissingCodeGraphNamesTheCommand(t *testing.T)  // problems contains "loomux graph build"; wiki nodes still there
func TestGraphData_MissingWikiGraphNamesTheCommand(t *testing.T)  // problems contains "loomux reindex"
func TestGraphData_UnregisteredProjectHasNoWiki(t *testing.T)     // no doc nodes, problem says it is no registered area
func TestGraphData_RereadsARebuiltCodeGraph(t *testing.T)         // rewrite wiring.json with a new mtime -> new node visible
func TestGraphData_UnknownProjectIs404(t *testing.T)
func TestGraphData_NeedsASession(t *testing.T)                    // same redirect/refusal as /settings/{id}
func TestGraphSearch_ReturnsHits(t *testing.T)                    // and `[]` for empty q
```

- [ ] **Step 2: Run, expect FAIL** — `go test ./internal/web/ -run Graph`
- [ ] **Step 3: Implement** handlers in `graph.go`. Die globale ID (`global`) ergibt 404. Ein kaputtes `wiring.json` oder `graph.json` ist ein Eintrag in `problems` mit der Fehlermeldung, kein 500.
- [ ] **Step 4: Run, expect PASS** — `go test ./internal/web/...`
- [ ] **Step 5: Commit**

```bash
git add internal/web
git commit -m "feat(ui): serve the graph page and its data"
```

### Task 7: force-graph einbetten

**Files:**
- Create: `internal/web/assets/vendor/force-graph.min.js`, `internal/web/assets/vendor/LICENSES.md`
- Modify: `internal/web/templates.go` oder `graph.go` (Auslieferung unter `GET /assets/vendor/force-graph.min.js`, `Content-Type: text/javascript`), `internal/web/assets/style.css` (die Regeln, die das Bündel per `<style>` einfügen würde), `internal/dev/notices/notices.go` (`LICENSES.md` anhängen), `internal/notices/NOTICE.md` (neu erzeugt), `AGENTS.md` (Zeile neben `third_party/toml`: Version, SHA-256, Herkunft `https://cdn.jsdelivr.net/npm/force-graph@<v>/dist/force-graph.min.js`, Aktualisierung von Hand)
- Test: `internal/web/vendor_test.go`, `internal/dev/notices/notices_test.go`

**Interfaces:**
- Produces: `const forceGraphSHA256 = "<hex>"` in `internal/web/vendor_test.go` (nur Test)

- [ ] **Step 1: Version und Pflegestand prüfen**: `npm view force-graph version time.modified dependencies`, GitHub `vasturiano/force-graph` (letzte Commits, offene Issues, archiviert?). Ergebnis in die Commit-Nachricht.
- [ ] **Step 2: Datei holen** in einen leeren Scratchpad-Ordner, dann prüfen: `grep -c 'eval(' ` = 0, `Function("return this")` ist nur der Fallback von lodash, die Stellen mit `createElement("style")` sind gefunden. Datei nach `internal/web/assets/vendor/` kopieren.
- [ ] **Step 3: `LICENSES.md` schreiben**: die Lizenz von force-graph und von jedem Paket im Bündel (rekursiv aus `npm view <pkg>@<v> dependencies license`), jeweils der volle Lizenztext.
- [ ] **Step 4: Write the failing tests**

```go
func TestForceGraphIsThePinnedFile(t *testing.T) // sha256 of embedded file == forceGraphSHA256
func TestForceGraphIsServedAsScript(t *testing.T) // 200, Content-Type text/javascript
// notices: the existing TestTheNoticeIsCurrent fails until NOTICE.md is regenerated;
// add TestRenderIncludesBundledScriptLicenses: Render output contains the first line of LICENSES.md
```

- [ ] **Step 5: Run, expect FAIL** — `go test ./internal/web/ ./internal/dev/notices/ ./internal/notices/`
- [ ] **Step 6: Implement**, dann `loomux dev notices` ausführen, damit `NOTICE.md` neu geschrieben wird.
- [ ] **Step 7: Run, expect PASS** — gleicher Befehl
- [ ] **Step 8: Commit**

```bash
git add internal/web internal/dev/notices internal/notices AGENTS.md
git commit -m "feat(ui): embed force-graph <version> for the graph view"
```

### Task 8: Die Insel

**Files:**
- Create: `internal/web/assets/graph.js`
- Modify: `internal/web/assets/graph.html` (Zeichenfläche, Seitenleiste, `<script src>` für Bündel und `graph.js`, Projekt-ID als `data-`-Attribut), `internal/web/assets/style.css`, Auslieferung `GET /assets/graph.js`
- Test: `internal/web/graph_test.go` (Auslieferung), Handprüfung im Browser

**Interfaces:**
- Consumes: `/graph/{id}/data`, `/graph/{id}/search`, `View`, `Hit` (Task 3 bis 6)

- [ ] **Step 1: Write the failing test** `TestGraphScriptIsServed` (200, `text/javascript`, Seite referenziert `/assets/graph.js` und `/assets/vendor/force-graph.min.js`, kein Inline-Skript).
- [ ] **Step 2: Run, expect FAIL** — `go test ./internal/web/ -run GraphScript`
- [ ] **Step 3: `graph.js` schreiben.** Verhalten laut Spec, Abschnitt 4:
  - Daten holen, `ForceGraph()` füttern; `cooldownTicks` setzen, damit das Layout stillsteht.
  - Größe aus `size`, Farbe aus `group`, Linienstärke aus der Summe der sichtbaren `counts`.
  - Haken je Relation und Konfidenz (Schlüssel aus `counts`), „Wiki“ (an), „other documents“ (aus); sie schalten Sichtbarkeit über `nodeVisibility`/`linkVisibility`, ohne neu zu laden.
  - Klick: Infofeld mit Art, Pfad, Mitgliedern, Nachbarn samt Zählern, bei Dokumenten `missing`. Doppelklick auf `dir`/`file`: `open` ergänzen und neu laden. Pfadleiste aus `open` führt zurück.
  - Zustand (`open`, Filter) im Fragment (`#open=…&off=…`), `hashchange` lädt neu.
  - Suche: Eingabe, Trefferliste; ein Treffer setzt `open` aus dem Hit, lädt und hebt `id` hervor.
  - Texte über `textContent`, nie `innerHTML`.
- [ ] **Step 4: Run, expect PASS** — `go test ./internal/web/...`
- [ ] **Step 5: Handprüfung**: `loomux ui` auf loomux selbst, nach `loomux reindex`. Erwartet: eine Übersicht mit höchstens 150 Code-Knoten, die Wikiseiten mit Kanten zum Code, keine CSP-Meldung außer den geblockten `<style>`-Einfügungen des Bündels; Doppelklick, Zurück, Suche und Filter funktionieren. Bildschirmfoto in den PR.
- [ ] **Step 6: Commit**

```bash
git add internal/web
git commit -m "feat(ui): draw the graph view in the browser"
```

### Task 9: Doku, Roadmap, Wiki

**Files:**
- Modify: `README.md`, `README.de.md` (Funktion „Graphansicht“; Roadmap-Zeile „Graph-Visualizer“ nur noch 3D)
- Modify: `docs/en/` und `docs/de/`: die Seite der Weboberfläche bzw. `cli-reference.md` (`/graph/{id}`), die Beschreibung von `graph.json` (`code_refs`), die Regel für Codeverweise (Backticks, mindestens ein `/`, Pfad existiert in exakter Schreibweise, keine Codeblöcke, Markdown-Links relativ zum Dokument), die bekannten Grenzen (veraltete Zeile, kein `reindex` bei Codeänderung, Wikis in eigenem Bereich)
- Modify: betroffene Seiten unter `docs/wiki/` (welche, sagt `AGENTS.md` des Projekts), `docs/wiki/log.md`

- [ ] **Step 1: Stellen finden**: Suche nach `graph.json`, `Graph-Visualizer` und `loomux ui` in `README*.md`, `docs/en`, `docs/de`, `docs/wiki`.
- [ ] **Step 2: Schreiben.**
- [ ] **Step 3: Prüfen**: `loomux check precommit` grün, insbesondere `lint/wiki`.
- [ ] **Step 4: Commit**

```bash
git add README.md README.de.md docs
git commit -m "docs: describe the graph view and code references"
```
