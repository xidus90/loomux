# loomux G2b — Implementierungsplan: der Graph antwortet

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `loomux graph ask "<anfrage>"` beantwortet eine Frage in Klartext aus
dem Graphen, den G2a schreibt — gerankt, mit Ort und Zeilenspan, auf Wunsch mit
dem Quelltext daneben.

**Architecture:** Zwei neue Pakete und eine Flagge am Bau. `lexicon` trägt die
Tokenisierung und die beim Bauen geschriebene Beiakte mit `df`, `docCount` und
`avgBodyLen`. `ask` rechnet daraus die lexikalischen Werte, gibt sie als **Saat**
an `pagerank` aus G1 und verschmilzt beide Achsen zu einem Rang. Der Befehl baut
bei Drift vorher neu, unter einer Sperre und nie fatal.

**Tech Stack:** Go, Standardbibliothek allein: `math`, `regexp`, `sort`,
`encoding/json`, `strings`. **Keine neue Abhängigkeit.**

**Spec:** `docs/.superpowers/specs/2026-09-17-loomux-code-g2-design.md`,
Abschnitt 10 und 12.3. Bei Widerspruch gilt die Spec, danach diese Datei.

**Voraussetzung:** G2a ist fertig und gemerged. `ask` liest `wiring.json` und
ruft bei Drift `buildGraph` — ohne Extraktor und Schreiber gibt es hier nichts
zu bauen.

**Referenz:** `trailhq/Graft` @ `1e352a3` (MIT). Jede Konstante und jede Regel
unten steht als Literal in diesem Plan.

## Global Constraints

Dieselben wie in G2a; sie gelten unverändert und werden hier nicht wiederholt,
mit drei Zusätzen, die nur diese Stufe betreffen:

- **Arbeitsort:** Worktree `C:\Users\micro\Documents\#GIT\loomux-code-g2`,
  Branch `code-g2`, auf dem Stand nach G2a.
- **Die Konstanten sind nicht zu runden und nicht zu erfinden.** Sie
  entscheiden, welche Antwort oben steht:

  | Konstante | Wert | Ort in Graft |
  |---|---|---|
  | Namensgewicht | 3 | `ask.ts:917–922` |
  | Pfadgewicht | 2 | `ask.ts:917–922` |
  | BM25 `k1` | 1,2 | `ask.ts:263` |
  | BM25 `b` | 0,75 | `ask.ts:264` |
  | `GRAPH_WEIGHT` | 0,5 | `ask.ts:414` |
  | `RESCUE_FLOOR` | 0,15 | `ask.ts:420` |
  | `TEST_RANK_PENALTY` | 0,35 | `ask.ts:205` |
  | `MAX_SPAN_LINES` | 80 | `ask.ts:99` |
  | `maxBodyChars` | 5000 | `extract.ts:133` |
  | Stoppwörter | 32 | `index-file.ts:28–33` |
  | `--limit` | 8 | `cli.ts:596` |
  | PageRank `alpha`, `iters` | 0,25 und 25 | G1, schon gebaut |

- **Die Tokenisierung wird byteweise übernommen, nicht dem Sinn nach.**
  camelCase-Schnitt auf `[a-z0-9][A-Z]`, getrennt an `[^a-z0-9]+`, also reines
  ASCII. `unicode.IsUpper` wäre die schönere Go-Fassung und lieferte auf einem
  Bezeichner mit Umlaut andere Token als die Golden-Werte.

---

## File Structure

| Datei | Verantwortung |
|---|---|
| `internal/code/lexicon/tokenize.go` | `Tokenize`, `Counts`, die Stoppliste |
| `internal/code/lexicon/index.go` | `Index`, `Doc`, `Build`, `Write`, `Read`, `Path` |
| `internal/code/ask/score.go` | `idf`, `overlap`, `bm25` |
| `internal/code/ask/ask.go` | `Options`, `Hit`, `Answer`, `Run` |
| `internal/code/ask/source.go` | `inlineSource`: Span-Schnitt, Kappung, Restmarker |
| `internal/code/ask/refresh.go` | `EnsureFresh`: Sonde, Neubau, Sperre, nie fatal |
| `internal/cli/graph.go` (ändern) | Unterbefehl `ask`; `build` schreibt die Beiakte |
| `docs/{en,de}/cli-reference.md`, `benchmarks.md` (ändern) | `graph ask` und die Messungen |
| `README.md`, `README.de.md` (ändern) | Stand und Fahrplan |
| `docs/.superpowers/parity/code-g2b.md` | Mutationsrunde und die portierten Vektoren |

---

### Task 0: Arbeitsort und Voraussetzung prüfen

**Files:** keine.

**Interfaces:**
- Consumes: G2a.
- Produces: nichts.

- [ ] **Schritt 1: Ort, Basis und G2a lesen**

```sh
cd "C:/Users/micro/Documents/#GIT/loomux-code-g2"
git rev-parse --abbrev-ref HEAD
git log -1 --format='%h %an <%ae>'
go run ./cmd/loomux graph build --root .
go run ./cmd/loomux graph check --root .
```

Erwartet: `code-g2`, Autor der Mensch, `build` schreibt eine Zählzeile,
`check` sagt OK und exitet 0. Tut es das nicht, ist G2a nicht fertig — melden,
nicht hier reparieren.

- [ ] **Schritt 2: Tor einmal leer fahren**

```sh
go test ./... -count=1
```

---

### Task 1: `internal/code/lexicon` — Tokenisierung

**Files:**
- Create: `internal/code/lexicon/tokenize.go`
- Test: `internal/code/lexicon/tokenize_test.go`

**Interfaces:**
- Consumes: nichts.
- Produces:

```go
func Tokenize(text string) []string
func Counts(tokens []string) map[string]int
```

Eine Funktion, die Bauen und Abfragen teilen. Das ist nicht Bequemlichkeit: die
Beiakte kann nur dann ein korrekter Cache der Abfragerechnung sein, wenn beide
Seiten denselben Code rufen. Graft sagt es im Dateikopf und hat `tokenize`
deshalb in `index-file.ts` und nicht in `ask.ts`.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package lexicon_test

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/lexicon"
)

func TestTokenizeSplitsCamelCaseAndLowercases(t *testing.T) {
	got := lexicon.Tokenize("resolveGoImport")
	want := []string{"resolve", "go", "import"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenizeSplitsOnEveryNonAlphanumeric(t *testing.T) {
	got := lexicon.Tokenize("internal/code/extract/golang_test.go")
	want := []string{"internal", "code", "extract", "golang", "test", "go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenizeDropsSingleCharactersAndStopWords(t *testing.T) {
	// Length must be greater than one, and the stop list is 32 words.
	got := lexicon.Tokenize("how does the a x cache get used")
	want := []string{"cache"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v -- stop words and single characters carry no intent", got, want)
	}
}

func TestTokenizeIsASCIIOnlyByDesign(t *testing.T) {
	// Byte-for-byte Graft's regex: the camel split is [a-z0-9][A-Z], the
	// separator is [^a-z0-9]. unicode.IsUpper would be the nicer Go version and
	// would produce different tokens than the golden values on an identifier
	// with an umlaut.
	got := lexicon.Tokenize("großeZahl")
	want := []string{"gro", "zahl"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenizeHandlesDigitsInsideAName(t *testing.T) {
	got := lexicon.Tokenize("sha256Sum")
	want := []string{"sha256", "sum"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenizeOfEmptyTextIsEmpty(t *testing.T) {
	if got := lexicon.Tokenize(""); len(got) != 0 {
		t.Fatalf("got %v, want nothing", got)
	}
}

func TestCountsSumsRepeats(t *testing.T) {
	got := lexicon.Counts([]string{"cache", "get", "cache"})
	if got["cache"] != 2 || got["get"] != 1 {
		t.Fatalf("got %v, want cache twice", got)
	}
}

func TestStopWordsAreThirtyTwo(t *testing.T) {
	// Counted against index-file.ts:28-33, not taken from a comment. The number
	// is here so a later edit to the list is a deliberate edit.
	if n := lexicon.StopWordCount(); n != 32 {
		t.Fatalf("got %d stop words, want 32", n)
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/lexicon/ -v
```

Erwartet: Übersetzungsfehler, das Paket gibt es nicht.

- [ ] **Schritt 3: Die Implementierung schreiben**

```go
// Package lexicon is the text side of the code graph: how a name, a path and a
// body become tokens, and the build-time sidecar that keeps a query from doing
// that work again.
//
// Tokenize and Counts live here and not in the asking package, for the reason
// Graft gives in the same place: the sidecar can only be a correct cache of the
// query-time arithmetic if both sides call the same function. Two tokenizers
// that agree today are two tokenizers.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/ask/index-file.ts.
package lexicon

import (
	"regexp"
	"strings"
)

// stopWords carry no query intent -- too common or too short to discriminate.
// Thirty-two words, counted against the reference rather than taken from a
// comment.
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "of": true, "to": true, "in": true,
	"is": true, "are": true, "how": true, "does": true, "do": true, "what": true,
	"where": true, "which": true, "that": true, "this": true, "it": true,
	"for": true, "on": true, "and": true, "or": true, "with": true, "i": true,
	"we": true, "get": true, "set": true, "use": true, "used": true,
	"using": true, "when": true, "why": true, "can": true,
}

// StopWordCount is how many words the list holds. A test pins it, so shrinking
// or growing the list is a deliberate edit and not a slip.
func StopWordCount() int { return len(stopWords) }

// camelBoundary is where a lowercase or digit meets an uppercase letter.
//
// ASCII by design, byte for byte as in the reference. A Unicode-aware split
// would be the nicer Go version and would tokenize an identifier carrying an
// umlaut differently than every golden value in this repository.
var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// separator is every run of characters that is not a lowercase letter or digit.
var separator = regexp.MustCompile(`[^a-z0-9]+`)

// Tokenize splits prose and identifiers into lowercased subword tokens.
//
// The order matters: the camel boundary is marked BEFORE lowercasing, or the
// boundary would be gone by the time we look for it.
func Tokenize(text string) []string {
	spaced := camelBoundary.ReplaceAllString(text, "$1 $2")
	var out []string
	for _, tok := range separator.Split(strings.ToLower(spaced), -1) {
		if len(tok) > 1 && !stopWords[tok] {
			out = append(out, tok)
		}
	}
	return out
}

// Counts is a token-frequency bag.
func Counts(tokens []string) map[string]int {
	out := make(map[string]int, len(tokens))
	for _, t := range tokens {
		out[t]++
	}
	return out
}
```

Der Test `TestTokenizeIsASCIIOnlyByDesign` erwartet `["gro", "zahl"]`, und der
Weg dahin ist der eigentliche Punkt: das `e` vor dem `Z` **ist** ein `[a-z]`, die
camelCase-Grenze greift also und macht `große Zahl`; dann senkt `ToLower` das
`Z`, und `ß` ist kein `[a-z0-9]`, trennt also. Übrig bleiben `gro`, ein
einbuchstabiges `e`, das die Längengrenze fällt, und `zahl`. Das Wort zerfällt an
einer Stelle, an der kein Leser eine Grenze sieht — und genau das ist der Preis
der byteweisen Übernahme, den eine Unicode-Fassung nicht hätte, um den Preis
jedes Golden-Werts in diesem Repo.

Der Wert ist am 2026-09-17 in einem Wegwerfskript nachgefahren und **nicht**
hergeleitet; eine frühere Fassung dieses Plans behauptete `["gro", "eZahl"]` und
lag falsch. **Wer die Erwartung ändert, fährt sie erneut nach:** die Reihenfolge
von Grenzmarkierung, Kleinschreibung und Trennung entscheidet das Ergebnis, und
eine Erwartung aus dem Kopf ist hier nichts wert.

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/code/lexicon/ -count=1 -cover
```

- [ ] **Schritt 5: Commit**

```sh
git add internal/code/lexicon
git commit -F ../msg.txt
```

Nachricht:

```
Tokenize names, paths and bodies once, for both sides

One function, shared by the build that writes the sidecar and the query
that reads it. That is the whole point: the sidecar can only be a
correct cache of the query's arithmetic if both call the same code. Two
tokenizers that agree today are two tokenizers.

ASCII by design, byte for byte as in the reference. A Unicode-aware
split would be the nicer Go version and would tokenize an identifier
with an umlaut differently from every golden value here.
```

---

### Task 2: `internal/code/lexicon` — die Beiakte

**Files:**
- Create: `internal/code/lexicon/index.go`
- Test: `internal/code/lexicon/index_test.go`

**Interfaces:**
- Consumes: `Tokenize`, `Counts`, `model.Graph`, `store.CachePath`.
- Produces:

```go
type Doc struct {
	ID   model.NodeID
	Name map[string]int
	Path map[string]int
	Body map[string]int
}

type Index struct {
	Version    int
	AvgBodyLen float64
	DocCount   int
	DF         map[string]int
	Docs       []Doc
}

func Build(g *model.Graph) *Index
func Write(root string, ix *Index) error
func Read(root string) (*Index, error)
func Path(root string) string

// Filter returns the index restricted to documents under a path prefix, with
// df, docCount and avgBodyLen recomputed over the remainder.
func (ix *Index) Filter(prefix string) *Index
```

`Build` bekommt den Graphen **vor** dem Schreiben, also mit `BodyText`. Graft
warnt ausdrücklich davor, die Beiakte aus der geschriebenen Datei zu
rekonstruieren — dort ist der Rumpf gestrippt.

`Filter` ist der Grund, warum `--in` in Graft „filters before scoring" heißt:
ein Wort, das im ganzen Repo häufig und in einem Unterbaum selten ist, muss
dort diskriminieren.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package lexicon_test

import (
	"math"
	"os"
	"testing"

	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
)

func graph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 2, Extractor: "go/1"},
		Nodes: []model.Node{
			{
				ID: "lib/cache.go", Name: "cache.go", Kind: model.KindFile, Path: "lib/cache.go",
				BodyHash: "h", BodyText: "package lib import sync",
			},
			{
				ID: "lib/cache.go#Cache.Get", Name: "Get", Kind: "method", Owner: "Cache",
				Path: "lib/cache.go", BodyHash: "h", BodyText: "return c.entries[key]",
			},
			{
				ID: "web/render.go#Render", Name: "Render", Kind: "function",
				Path: "web/render.go", BodyHash: "h", BodyText: "return template",
			},
		},
	}
}

func TestBuildCountsEveryFieldOfEveryNode(t *testing.T) {
	ix := lexicon.Build(graph())

	if ix.DocCount != 3 {
		t.Fatalf("DocCount = %d, want 3 -- files count too", ix.DocCount)
	}
	var get *lexicon.Doc
	for i := range ix.Docs {
		if ix.Docs[i].ID == "lib/cache.go#Cache.Get" {
			get = &ix.Docs[i]
		}
	}
	if get == nil {
		t.Fatal("the method's doc is missing")
	}
	if get.Name["get"] != 1 {
		t.Errorf("name bag = %v, want the bare name", get.Name)
	}
	if get.Path["cache"] != 1 || get.Path["lib"] != 1 {
		t.Errorf("path bag = %v, want the path's segments", get.Path)
	}
	if get.Body["entries"] != 1 {
		t.Errorf("body bag = %v, want the body's words", get.Body)
	}
}

func TestBuildCountsDocumentFrequencyAcrossFields(t *testing.T) {
	ix := lexicon.Build(graph())

	// "cache" is in the file's path and in the method's path: two documents.
	if ix.DF["cache"] != 2 {
		t.Errorf("DF[cache] = %d, want 2", ix.DF["cache"])
	}
	// A term is counted once per document, however many fields hold it.
	if ix.DF["lib"] != 2 {
		t.Errorf("DF[lib] = %d, want 2", ix.DF["lib"])
	}
}

func TestBuildAveragesTheBodyLength(t *testing.T) {
	ix := lexicon.Build(graph())

	var total int
	for _, d := range ix.Docs {
		for _, n := range d.Body {
			total += n
		}
	}
	want := float64(total) / 3
	if math.Abs(ix.AvgBodyLen-want) > 1e-9 {
		t.Errorf("AvgBodyLen = %v, want %v -- BM25 normalizes against it", ix.AvgBodyLen, want)
	}
}

func TestWriteThenReadRoundTrips(t *testing.T) {
	root := t.TempDir()
	if err := lexicon.Write(root, lexicon.Build(graph())); err != nil {
		t.Fatal(err)
	}
	got, err := lexicon.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.DocCount != 3 || got.DF["cache"] != 2 || len(got.Docs) != 3 {
		t.Fatalf("got %+v", got)
	}
}

func TestReadReportsAMissingSidecar(t *testing.T) {
	if _, err := lexicon.Read(t.TempDir()); err == nil {
		t.Fatal("got nil, want an error -- a caller decides whether to rebuild")
	}
}

func TestReadRefusesASidecarOfAnotherVersion(t *testing.T) {
	root := t.TempDir()
	if err := lexicon.Write(root, lexicon.Build(graph())); err != nil {
		t.Fatal(err)
	}
	body := `{"version":99,"doc_count":1,"df":{},"docs":[]}`
	if err := os.WriteFile(lexicon.Path(root), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := lexicon.Read(root); err == nil {
		t.Fatal("got nil, want a refusal: a shape this code cannot read is no cache")
	}
}

func TestFilterRecomputesDocumentFrequencyOverTheRemainder(t *testing.T) {
	ix := lexicon.Build(graph())
	sub := ix.Filter("lib")

	if sub.DocCount != 2 {
		t.Fatalf("DocCount = %d, want the two documents under lib/", sub.DocCount)
	}
	// This is what makes --in more than a post-filter: within lib/ the word
	// "cache" is in both documents and discriminates nothing, while across the
	// repository it separated lib/ from web/. A query restricted to a subtree
	// must be scored by that subtree's statistics.
	if sub.DF["cache"] != 2 {
		t.Errorf("DF[cache] = %d, want 2 within the subtree", sub.DF["cache"])
	}
	if _, ok := sub.DF["template"]; ok {
		t.Error("a term that lives only outside the prefix must be gone from df")
	}
	for _, d := range sub.Docs {
		if d.ID == "web/render.go#Render" {
			t.Error("a document outside the prefix must be gone")
		}
	}
}

func TestFilterIsSegmentAware(t *testing.T) {
	g := graph()
	g.Nodes = append(g.Nodes, model.Node{
		ID: "libextra/other.go#Other", Name: "Other", Kind: "function",
		Path: "libextra/other.go", BodyHash: "h", BodyText: "x",
	})
	sub := lexicon.Build(g).Filter("lib")

	// "lib" must not match "libextra": a prefix is a path prefix, not a string
	// prefix.
	for _, d := range sub.Docs {
		if d.ID == "libextra/other.go#Other" {
			t.Fatalf("segment-unaware filter matched %q", d.ID)
		}
	}
}

func TestFilterOnAnEmptyPrefixIsTheWholeIndex(t *testing.T) {
	ix := lexicon.Build(graph())
	if got := ix.Filter(""); got.DocCount != ix.DocCount {
		t.Fatalf("DocCount = %d, want the whole index at %d", got.DocCount, ix.DocCount)
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/lexicon/ -run 'TestBuild|TestWrite|TestRead|TestFilter' -v
```

- [ ] **Schritt 3: Die Implementierung schreiben**

```go
package lexicon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
)

// indexVersion is the sidecar's shape. Another version is a shape this code
// cannot read, which is the same as no sidecar.
const indexVersion = 1

// Doc is one node's three token bags.
type Doc struct {
	ID   model.NodeID   `json:"id"`
	Name map[string]int `json:"name"`
	Path map[string]int `json:"path"`
	Body map[string]int `json:"body"`
}

// Index is the build-time sidecar.
//
// It exists because tokenizing every node's name, path and body on every query
// was ~45% of query time at 32k nodes, profiled. It is derived data and lives
// under cache/ next to the graph, not in it.
type Index struct {
	Version    int            `json:"version"`
	AvgBodyLen float64        `json:"avg_body_len"`
	DocCount   int            `json:"doc_count"`
	DF         map[string]int `json:"df"`
	Docs       []Doc          `json:"docs"`
}

// Path is where the sidecar lives.
func Path(root string) string { return store.CachePath(root, "ask-index.json") }

// Build tokenizes a whole graph.
//
// It must be handed the graph as the build holds it in memory, WITH BodyText.
// Reconstructing this from the written wiring.json is impossible: the body is
// stripped there, which is the point of stripping it.
func Build(g *model.Graph) *Index {
	ix := &Index{Version: indexVersion, DF: map[string]int{}}
	bodyTotal := 0
	for _, n := range g.Nodes {
		doc := Doc{
			ID:   n.ID,
			Name: Counts(Tokenize(n.Name)),
			Path: Counts(Tokenize(n.Path)),
			Body: Counts(Tokenize(n.BodyText)),
		}
		ix.Docs = append(ix.Docs, doc)
		for _, n := range doc.Body {
			bodyTotal += n
		}
		// A term counts once per document, however many of its fields hold it:
		// df is a document frequency, not a term frequency.
		for term := range terms(doc) {
			ix.DF[term]++
		}
	}
	ix.DocCount = len(ix.Docs)
	if ix.DocCount > 0 {
		ix.AvgBodyLen = float64(bodyTotal) / float64(ix.DocCount)
	}
	return ix
}

// terms is the set of tokens a document holds, across its three fields.
func terms(d Doc) map[string]bool {
	out := map[string]bool{}
	for _, bag := range []map[string]int{d.Name, d.Path, d.Body} {
		for term := range bag {
			out[term] = true
		}
	}
	return out
}

// Filter restricts the index to documents whose node path lies at or under a
// repo-relative prefix, and recomputes the corpus statistics over what is left.
//
// Recomputing is the whole difference between a filter and a post-filter: a
// word that is common across the repository and rare inside one subtree has to
// discriminate inside that subtree. Graft pins this with a test of its own --
// a term's rank flips relative to another between filtered and unfiltered.
//
// The prefix is segment-aware: "lib" never matches "libextra".
func (ix *Index) Filter(prefix string) *Index {
	if prefix == "" {
		return ix
	}
	out := &Index{Version: ix.Version, DF: map[string]int{}}
	bodyTotal := 0
	for _, d := range ix.Docs {
		if !underPrefix(string(d.ID), prefix) {
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

// underPrefix reports whether a node id's path part lies at or under prefix,
// comparing whole segments.
func underPrefix(id, prefix string) bool {
	path := id
	if i := strings.Index(id, "#"); i >= 0 {
		path = id[:i]
	}
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

// Write serializes the sidecar. Maps are serialized by encoding/json in sorted
// key order, so two writes of one index are byte-identical.
func Write(root string, ix *Index) error {
	body, err := json.MarshalIndent(ix, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(Path(root), body, 0o644)
}

// Read loads the sidecar.
func Read(root string) (*Index, error) {
	b, err := os.ReadFile(Path(root))
	if err != nil {
		return nil, fmt.Errorf("read ask index: %w", err)
	}
	var ix Index
	if err := json.Unmarshal(b, &ix); err != nil {
		return nil, fmt.Errorf("read ask index: %w", err)
	}
	if ix.Version != indexVersion {
		return nil, fmt.Errorf("ask index version %d, want %d", ix.Version, indexVersion)
	}
	return &ix, nil
}

// sortDocs keeps the document order deterministic, so a rebuilt sidecar is
// byte-identical to the one it replaces.
func sortDocs(docs []Doc) {
	sort.Slice(docs, func(i, j int) bool { return docs[i].ID < docs[j].ID })
}
```

`Build` ruft am Ende `sortDocs(ix.Docs)`; der Graph ist schon sortiert, aber
die Beiakte soll nicht davon abhängen.

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/code/lexicon/ -count=1 -cover
```

- [ ] **Schritt 5: Commit**

```sh
git add internal/code/lexicon
git commit -F ../msg.txt
```

Nachricht:

```
Write the ask sidecar at build time

Tokenizing every node's name, path and body on every query was ~45% of
query time at 32k nodes in the reference. The sidecar carries the bags
and the corpus statistics instead, under cache/ beside the graph: it is
derived data, regenerable at any time.

It must be built from the graph as the build holds it, with the body
text. Reconstructing it from the written file is impossible, because the
body is stripped there -- which is the point of stripping it.

Filter recomputes df, doc count and average body length over the
remainder. That is the difference between a filter and a post-filter: a
word common across the repository and rare in one subtree has to
discriminate inside that subtree.
```

---

### Task 3: `graph build` schreibt die Beiakte mit

**Files:**
- Modify: `internal/cli/graph.go`
- Test: `internal/cli/graph_test.go` (anhängen)

**Interfaces:**
- Consumes: `lexicon.Build`, `lexicon.Write`, `buildGraph` aus G2a.
- Produces: `build` legt `cache/ask-index.json` neben den Graphen.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
func TestGraphBuildWritesTheAskSidecar(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}

	ix, err := lexicon.Read(root)
	if err != nil {
		t.Fatalf("no sidecar written: %v", err)
	}
	if ix.DocCount == 0 || len(ix.DF) == 0 {
		t.Fatalf("sidecar is empty: %+v", ix)
	}
	// It must carry the body, which wiring.json does not: a symbol has to be
	// findable by a word that appears only inside it.
	found := false
	for _, d := range ix.Docs {
		if d.ID == "lib/lib.go#Run" && len(d.Body) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("the sidecar must carry body tokens the graph does not")
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/cli/ -run TestGraphBuildWritesTheAskSidecar -v
```

Erwartet: FAIL, „no sidecar written".

- [ ] **Schritt 3: Die Implementierung ändern**

In `graphBuild`, direkt nach `store.Write` und vor `freshness.Write`:

```go
	// From the graph in memory, which still carries the body text: the written
	// file has it stripped, and that is what makes the sidecar necessary rather
	// than redundant.
	if err := lexicon.Write(project, lexicon.Build(g)); err != nil {
		fmt.Fprintf(stderr, "loomux graph build: ask index not written: %v\n", err)
	}
```

Das ist kein Abbruch: die Beiakte ist ein Cache, und ein `ask` ohne sie
tokenisiert live. Der Import kommt dazu.

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/cli/ -count=1
```

- [ ] **Schritt 5: Commit**

```sh
git add internal/cli
git commit -F ../msg.txt
```

Nachricht:

```
Write the ask sidecar as part of a build

From the graph in memory, which still carries the body text. A failure
here is a warning and not an exit: the sidecar is a cache, and a query
without it tokenizes live.
```

---

### Task 4: `internal/code/ask` — die lexikalische Rechnung

**Files:**
- Create: `internal/code/ask/score.go`
- Test: `internal/code/ask/score_test.go`

**Interfaces:**
- Consumes: `lexicon.Index`, `lexicon.Doc`.
- Produces:

```go
func idf(df, docCount int) float64
func overlap(query, doc map[string]int, idf map[string]float64) float64
func bm25(query, body map[string]int, idf map[string]float64, bodyLen, avgLen float64) float64
func lexical(query map[string]int, d lexicon.Doc, idf map[string]float64, avgLen float64) float64
```

Alles paketintern: niemand außerhalb rechnet Teilscores, und eine exportierte
Formel wäre eine Zusage, die diese Stufe nicht geben will.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package ask

import (
	"math"
	"testing"

	"github.com/xidus90/loomux/internal/code/lexicon"
)

func TestIDFFallsAsATermSpreads(t *testing.T) {
	// idf = log(1 + n/(1+df)). A term in one document of a hundred
	// discriminates; a term in every document does not.
	rare := idf(1, 100)
	common := idf(100, 100)
	if !(rare > common) {
		t.Fatalf("rare = %v, common = %v, want rare to weigh more", rare, common)
	}
	want := math.Log(1 + 100.0/2.0)
	if math.Abs(rare-want) > 1e-12 {
		t.Errorf("idf(1, 100) = %v, want %v", rare, want)
	}
}

func TestIDFOfAnUnseenTermIsTheCorpusWideValue(t *testing.T) {
	// df = 0: no document holds it, so the weight is the largest the corpus
	// allows rather than a division by zero.
	if got := idf(0, 10); math.Abs(got-math.Log(1+10.0)) > 1e-12 {
		t.Fatalf("idf(0, 10) = %v", got)
	}
}

func TestOverlapWeighsEachSharedTermByItsIDF(t *testing.T) {
	query := map[string]int{"cache": 1, "get": 1}
	doc := map[string]int{"cache": 2}
	weights := map[string]float64{"cache": 3, "get": 9}

	// 1 * 2 * 3 -- the query count, the document count, the term's weight. The
	// term the document lacks contributes nothing, however heavy.
	if got := overlap(query, doc, weights); math.Abs(got-6) > 1e-12 {
		t.Fatalf("overlap = %v, want 6", got)
	}
}

func TestBM25SaturatesARepeatedTerm(t *testing.T) {
	query := map[string]int{"cache": 1}
	weights := map[string]float64{"cache": 1}

	once := bm25(query, map[string]int{"cache": 1}, weights, 1, 1)
	tenTimes := bm25(query, map[string]int{"cache": 10}, weights, 10, 1)
	// k1 = 1.2 saturates: ten occurrences are worth more than one and nowhere
	// near ten times as much. Raw term frequency is what BM25 exists to avoid.
	if !(tenTimes > once) {
		t.Fatalf("ten = %v, once = %v", tenTimes, once)
	}
	if tenTimes > once*3 {
		t.Errorf("ten = %v against once = %v: a repeat must saturate", tenTimes, once)
	}
}

func TestBM25PenalizesALongBody(t *testing.T) {
	query := map[string]int{"cache": 1}
	weights := map[string]float64{"cache": 1}

	short := bm25(query, map[string]int{"cache": 1}, weights, 5, 100)
	long := bm25(query, map[string]int{"cache": 1}, weights, 500, 100)
	// b = 0.75 normalizes by length, so a long definition does not win on bulk.
	if !(short > long) {
		t.Fatalf("short = %v, long = %v, want the short body to score higher", short, long)
	}
}

func TestBM25OfAnEmptyCorpusIsZeroAndNotNaN(t *testing.T) {
	got := bm25(map[string]int{"x": 1}, map[string]int{"x": 1}, map[string]float64{"x": 1}, 0, 0)
	if math.IsNaN(got) || math.IsInf(got, 0) {
		t.Fatalf("got %v, want a finite value: an empty corpus must not poison a score", got)
	}
}

func TestLexicalWeighsNameThreeAndPathTwo(t *testing.T) {
	weights := map[string]float64{"cache": 1}
	query := map[string]int{"cache": 1}

	byName := lexical(query, lexicon.Doc{Name: map[string]int{"cache": 1}}, weights, 1)
	byPath := lexical(query, lexicon.Doc{Path: map[string]int{"cache": 1}}, weights, 1)
	byBody := lexical(query, lexicon.Doc{Body: map[string]int{"cache": 1}}, weights, 1)

	// 3 and 2 are not to be rounded and not to be invented: they decide whether
	// a name hit beats a path hit.
	if math.Abs(byName-3) > 1e-12 {
		t.Errorf("name score = %v, want 3", byName)
	}
	if math.Abs(byPath-2) > 1e-12 {
		t.Errorf("path score = %v, want 2", byPath)
	}
	if !(byName > byPath && byPath > byBody) {
		t.Errorf("name %v, path %v, body %v -- want that order", byName, byPath, byBody)
	}
}

func TestLexicalOfANonMatchIsZero(t *testing.T) {
	got := lexical(map[string]int{"nothing": 1},
		lexicon.Doc{Name: map[string]int{"cache": 1}},
		map[string]float64{"nothing": 5}, 1)
	if got != 0 {
		t.Fatalf("got %v, want 0 -- a document the query never touched scores nothing", got)
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/ask/ -v
```

- [ ] **Schritt 3: Die Implementierung schreiben**

```go
// Package ask answers a plain-text question from the code graph.
//
// The pipeline is the reference's, and its slogan is worth keeping because it
// is the design: lexical proposes, graph disposes. A word match nominates
// candidates; the walk over the wiring edges decides which of them the question
// was actually about, and rescues the helper the question never named.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/ask/ask.ts and
// src/ask/graphrank.ts.
package ask

import (
	"math"

	"github.com/xidus90/loomux/internal/code/lexicon"
)

// The field weights. A name hit is worth three, a path hit two, and the body
// goes through BM25 at one. They decide whether a name beats a path, so they
// are copied and not rounded.
const (
	nameWeight = 3.0
	pathWeight = 2.0
)

// BM25 parameters: k1 saturates a repeated term, b normalizes by body length.
const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

// idf is the inverse document frequency of a term: log(1 + n/(1+df)).
//
// A word that appears across the whole corpus -- "handler" in a repository of
// handlers -- weighs far less than a rare, discriminating identifier. The 1+ in
// the denominator is what makes a term no document holds finite rather than a
// division by zero.
func idf(df, docCount int) float64 {
	return math.Log(1 + float64(docCount)/float64(1+df))
}

// weights is the idf of every term of a query, against one corpus.
func weights(query map[string]int, ix *lexicon.Index) map[string]float64 {
	out := make(map[string]float64, len(query))
	for term := range query {
		out[term] = idf(ix.DF[term], ix.DocCount)
	}
	return out
}

// overlap is the idf-weighted term overlap of a query and a short field -- a
// name or a path, where length carries no information worth normalizing.
func overlap(query, doc map[string]int, idf map[string]float64) float64 {
	var s float64
	for term, qn := range query {
		if dn, ok := doc[term]; ok {
			s += float64(qn) * float64(dn) * idf[term]
		}
	}
	return s
}

// bm25 scores a query against a body, saturating repeats and normalizing by
// length.
func bm25(query, body map[string]int, idf map[string]float64, bodyLen, avgLen float64) float64 {
	norm := bm25K1 * (1 - bm25B + bm25B*lengthRatio(bodyLen, avgLen))
	var s float64
	// The query's own term frequency does not enter here, and the reference does
	// not use it either: BM25 weighs how often the DOCUMENT says a word, not how
	// often the question did.
	for term := range query {
		tf, ok := body[term]
		if !ok {
			continue
		}
		s += idf[term] * (float64(tf) * (bm25K1 + 1) / (float64(tf) + norm))
	}
	return s
}

// lengthRatio is a body's length against the corpus average, guarded: an empty
// corpus would otherwise divide by zero and poison every score with NaN.
func lengthRatio(bodyLen, avgLen float64) float64 {
	if avgLen <= 0 {
		return 1
	}
	return bodyLen / avgLen
}

// lexical is one document's whole lexical score.
func lexical(query map[string]int, d lexicon.Doc, idf map[string]float64, avgLen float64) float64 {
	return nameWeight*overlap(query, d.Name, idf) +
		pathWeight*overlap(query, d.Path, idf) +
		bm25(query, d.Body, idf, bagLen(d.Body), avgLen)
}

// bagLen is a field's length in tokens.
func bagLen(bag map[string]int) float64 {
	var n int
	for _, c := range bag {
		n += c
	}
	return float64(n)
}
```

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/code/ask/ -count=1 -cover
```

Erwartet: PASS bei 100 % je Funktion.

- [ ] **Schritt 5: Commit**

```sh
git add internal/code/ask
git commit -F ../msg.txt
```

Nachricht:

```
Score a query against a node's name, path and body

Three weights: a name hit is worth three, a path hit two, the body goes
through BM25 at one. They are copied from the reference rather than
chosen, because they decide whether a name hit beats a path hit.

idf is log(1 + n/(1+df)); the 1+ in the denominator keeps a term no
document holds finite. BM25 saturates a repeated term at k1 = 1.2 and
normalizes length at b = 0.75, so a long definition cannot win on bulk,
and the length ratio is guarded so an empty corpus cannot poison every
score with NaN.
```

---

### Task 5: `internal/code/ask` — Saat, Lauf und Verschmelzung

**Files:**
- Create: `internal/code/ask/ask.go`
- Test: `internal/code/ask/ask_test.go`

**Interfaces:**
- Consumes: `score.go` aus Task 4, `lexicon.Index`, `model.Graph`,
  `pagerank.Prepare`, `pagerank.Rank` aus G1.
- Produces:

```go
type Options struct {
	Limit int    // 0 means 8
	In    string // path prefix, segment-aware; "" for the whole repository
}

type Hit struct {
	ID        model.NodeID
	Path      string
	Span      model.Span
	Signature string
	Score     float64
	Lexical   float64
	Graph     float64
	Code      string // only with Options.Source
}

type Answer struct {
	Query string
	Hits  []Hit
	Note  string // set when nothing matched
}

func Run(g *model.Graph, ix *lexicon.Index, query string, opts Options) Answer
```

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package ask_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
)

// symbol is a node with the fields the ranking reads.
func symbol(id, name, path, body string) model.Node {
	return model.Node{
		ID: model.NodeID(id), Name: name, Kind: "function", Path: path,
		Span: "L1-L4", Signature: "func " + name + "()", BodyHash: "h", BodyText: body,
	}
}

func edge(from, to string) model.Edge {
	return model.Edge{
		Source: model.NodeID(from), Target: model.NodeID(to),
		Relation: model.RelationCalls, Confidence: model.ConfidenceExtracted,
	}
}

// world builds a graph and its sidecar together, the way a build does.
func world(nodes []model.Node, edges []model.Edge) (*model.Graph, *lexicon.Index) {
	g := &model.Graph{
		Meta:  model.Meta{Version: 2, Extractor: "go/1"},
		Nodes: nodes, Edges: edges,
	}
	return g, lexicon.Build(g)
}

func idsOf(a ask.Answer) []model.NodeID {
	var out []model.NodeID
	for _, h := range a.Hits {
		out = append(out, h.ID)
	}
	return out
}

func TestRunRanksAWordMatchByName(t *testing.T) {
	g, ix := world([]model.Node{
		symbol("a.go#ParseConfig", "ParseConfig", "a.go", "read the file"),
		symbol("b.go#Unrelated", "Unrelated", "b.go", "nothing here"),
	}, nil)

	got := ask.Run(g, ix, "parse config", ask.Options{})
	if len(got.Hits) == 0 || got.Hits[0].ID != "a.go#ParseConfig" {
		t.Fatalf("got %v, want ParseConfig first", idsOf(got))
	}
	// The top hit is normalized to 1 on the lexical axis before the graph
	// weight is added, so a score is comparable between queries.
	if got.Hits[0].Lexical <= 0 {
		t.Errorf("lexical = %v, want a positive score", got.Hits[0].Lexical)
	}
}

func TestRunLetsConnectivityBreakALexicalNearTie(t *testing.T) {
	// Two nodes share the query word. One sits in the cluster the question is
	// about, the other is isolated -- the window "overlay" against the scroll
	// "overlay", which is the collision graph rank exists to fix.
	nodes := []model.Node{
		symbol("core/overlay.go#Overlay", "Overlay", "core/overlay.go", "the real one"),
		symbol("misc/overlay.go#Overlay", "Overlay", "misc/overlay.go", "the isolated one"),
		symbol("core/a.go#A", "A", "core/a.go", "x"),
		symbol("core/b.go#B", "B", "core/b.go", "y"),
	}
	edges := []model.Edge{
		edge("core/a.go#A", "core/overlay.go#Overlay"),
		edge("core/b.go#B", "core/overlay.go#Overlay"),
	}
	g, ix := world(nodes, edges)

	got := ask.Run(g, ix, "overlay", ask.Options{})
	if len(got.Hits) < 2 {
		t.Fatalf("got %v, want both", idsOf(got))
	}
	if got.Hits[0].ID != "core/overlay.go#Overlay" {
		t.Fatalf("got %v, want the wired-in one first", idsOf(got))
	}
	if got.Hits[0].Graph <= got.Hits[1].Graph {
		t.Errorf("graph scores %v and %v: the connected node must carry more mass",
			got.Hits[0].Graph, got.Hits[1].Graph)
	}
}

func TestRunDoesNotLetConnectivityOverruleAClearLexicalWinner(t *testing.T) {
	// GRAPH_WEIGHT is 0.5: connectivity reorders near-ties, it does not
	// overturn a node the query plainly names.
	nodes := []model.Node{
		symbol("a.go#ParseConfigFile", "ParseConfigFile", "a.go", "parse config file"),
		symbol("hub/hub.go#Hub", "Hub", "hub/hub.go", "unrelated"),
		symbol("hub/x.go#X", "X", "hub/x.go", "unrelated"),
		symbol("hub/y.go#Y", "Y", "hub/y.go", "unrelated"),
	}
	edges := []model.Edge{
		edge("hub/x.go#X", "hub/hub.go#Hub"),
		edge("hub/y.go#Y", "hub/hub.go#Hub"),
	}
	g, ix := world(nodes, edges)

	got := ask.Run(g, ix, "parse config file", ask.Options{})
	if got.Hits[0].ID != "a.go#ParseConfigFile" {
		t.Fatalf("got %v, want the lexical winner first", idsOf(got))
	}
}

func TestRunRescuesACentralNodeTheQueryNeverNamed(t *testing.T) {
	// RESCUE_FLOOR is 0.15: a node the query never word-matched joins the
	// results when the walk gives it at least 15% of the top node's mass. This
	// is what surfaces the helper a task depends on but did not mention.
	nodes := []model.Node{
		symbol("a.go#Seeded", "Seeded", "a.go", "seeded"),
		symbol("b.go#Helper", "Helper", "b.go", "nothing matching"),
	}
	edges := []model.Edge{edge("a.go#Seeded", "b.go#Helper")}
	g, ix := world(nodes, edges)

	got := ask.Run(g, ix, "seeded", ask.Options{})
	ids := idsOf(got)
	if len(ids) != 2 {
		t.Fatalf("got %v, want the seeded node and its rescued neighbour", ids)
	}
	var helper ask.Hit
	for _, h := range got.Hits {
		if h.ID == "b.go#Helper" {
			helper = h
		}
	}
	if helper.Lexical != 0 {
		t.Errorf("the rescued node matched no word; lexical = %v", helper.Lexical)
	}
	if helper.Graph <= 0 {
		t.Errorf("the rescued node is there on graph mass alone; graph = %v", helper.Graph)
	}
}

func TestRunDeRanksATestFile(t *testing.T) {
	nodes := []model.Node{
		symbol("lib/cache.go#Cache", "Cache", "lib/cache.go", "the implementation"),
		symbol("lib/cache_test.go#TestCache", "TestCache", "lib/cache_test.go", "the cache test"),
	}
	g, ix := world(nodes, nil)

	got := ask.Run(g, ix, "cache", ask.Options{})
	if got.Hits[0].ID != "lib/cache.go#Cache" {
		t.Fatalf("got %v, want the implementation first", idsOf(got))
	}
	// Still present -- "where are the tests" is a fair question -- just below.
	if len(got.Hits) != 2 {
		t.Errorf("got %v, want the test in the results as well", idsOf(got))
	}
}

func TestRunDeRanksATestEvenWhenItIsTheStrongestRawMatch(t *testing.T) {
	// The penalty applies to the raw score AND again after normalization. With
	// only the first, dividing by max would restore a winning test to 1.0 and
	// erase the de-rank; with only the second, the test would still seed the
	// walk as if it were the answer.
	nodes := []model.Node{
		symbol("lib/cache_test.go#TestCacheCacheCache", "TestCacheCacheCache", "lib/cache_test.go", "cache cache cache"),
		symbol("lib/cache.go#Cache", "Cache", "lib/cache.go", "implementation"),
	}
	g, ix := world(nodes, nil)

	got := ask.Run(g, ix, "cache", ask.Options{})
	if got.Hits[0].ID != "lib/cache.go#Cache" {
		t.Fatalf("got %v, want the source above the test; scores %+v", idsOf(got), got.Hits)
	}
}

func TestRunLiftsThePenaltyWhenTheQueryAsksForTests(t *testing.T) {
	nodes := []model.Node{
		symbol("lib/cache.go#Cache", "Cache", "lib/cache.go", "implementation"),
		symbol("lib/cache_test.go#TestCache", "TestCache", "lib/cache_test.go", "cache test"),
	}
	g, ix := world(nodes, nil)

	got := ask.Run(g, ix, "cache tests", ask.Options{})
	// A question about tests must answer with tests. Without this the ported
	// vector test/ask.test.ts:40 cannot pass.
	if got.Hits[0].ID != "lib/cache_test.go#TestCache" {
		t.Fatalf("got %v, want the test first for a test-seeking query", idsOf(got))
	}
}

func TestRunHonoursTheLimitAndDefaultsToEight(t *testing.T) {
	var nodes []model.Node
	for _, name := range []string{"CacheA", "CacheB", "CacheC", "CacheD", "CacheE", "CacheF", "CacheG", "CacheH", "CacheI", "CacheJ"} {
		nodes = append(nodes, symbol("p/"+name+".go#"+name, name, "p/"+name+".go", "cache"))
	}
	g, ix := world(nodes, nil)

	if got := ask.Run(g, ix, "cache", ask.Options{}); len(got.Hits) != 8 {
		t.Fatalf("got %d hits, want the default 8", len(got.Hits))
	}
	if got := ask.Run(g, ix, "cache", ask.Options{Limit: 3}); len(got.Hits) != 3 {
		t.Fatalf("got %d hits, want 3", len(got.Hits))
	}
}

func TestRunFiltersBeforeScoringAndSegmentAware(t *testing.T) {
	nodes := []model.Node{
		symbol("widgets/a.go#Render", "Render", "widgets/a.go", "render"),
		symbol("widgets-extra/b.go#Render", "Render", "widgets-extra/b.go", "render"),
	}
	g, ix := world(nodes, nil)

	got := ask.Run(g, ix, "render", ask.Options{In: "widgets"})
	// A prefix is a path prefix: "widgets" never matches "widgets-extra".
	if len(got.Hits) != 1 || got.Hits[0].ID != "widgets/a.go#Render" {
		t.Fatalf("got %v, want only the node under widgets/", idsOf(got))
	}
}

func TestRunNarrowsTheWalkAndNotOnlyTheSeed(t *testing.T) {
	// Without a filter on the walk itself, a neighbour outside the prefix could
	// clear the rescue floor and surface -- defeating "filters before scoring".
	nodes := []model.Node{
		symbol("in/a.go#Seeded", "Seeded", "in/a.go", "seeded"),
		symbol("out/b.go#Neighbour", "Neighbour", "out/b.go", "nothing matching"),
	}
	edges := []model.Edge{edge("in/a.go#Seeded", "out/b.go#Neighbour")}
	g, ix := world(nodes, edges)

	got := ask.Run(g, ix, "seeded", ask.Options{In: "in"})
	for _, h := range got.Hits {
		if h.ID == "out/b.go#Neighbour" {
			t.Fatalf("a neighbour outside the prefix must never surface; got %v", idsOf(got))
		}
	}
}

func TestRunOnAMissedQuerySaysSoInsteadOfAnEmptyList(t *testing.T) {
	g, ix := world([]model.Node{symbol("a.go#F", "F", "a.go", "x")}, nil)

	got := ask.Run(g, ix, "nothing here matches this", ask.Options{})
	if len(got.Hits) != 0 {
		t.Fatalf("got %v, want nothing", idsOf(got))
	}
	// A silent empty list reads as "there is no such code". A note says which
	// it was.
	if got.Note == "" {
		t.Error("a query that matched nothing must say so")
	}
}

func TestRunRanksAFileNodeByItsResidualText(t *testing.T) {
	// A file node is a document of the lexical pass -- that is how a word in an
	// import header finds its file -- and it is not a hit a reader can open at
	// a signature. It ranks, and it ranks as a file.
	g, ix := world([]model.Node{
		{
			ID: "lib/only.go", Name: "only.go", Kind: model.KindFile, Path: "lib/only.go",
			Span: "L1-L9", BodyHash: "h", BodyText: "package lib needle",
		},
	}, nil)

	got := ask.Run(g, ix, "needle", ask.Options{})
	if len(got.Hits) != 1 || got.Hits[0].ID != "lib/only.go" {
		t.Fatalf("got %v, want the file found by a word in its residual", idsOf(got))
	}
}

func TestRunIsDeterministicOnATie(t *testing.T) {
	nodes := []model.Node{
		symbol("b.go#Same", "Same", "b.go", "same"),
		symbol("a.go#Same", "Same", "a.go", "same"),
	}
	g, ix := world(nodes, nil)

	first := ask.Run(g, ix, "same", ask.Options{})
	for i := 0; i < 5; i++ {
		again := ask.Run(g, ix, "same", ask.Options{})
		for j := range first.Hits {
			if first.Hits[j].ID != again.Hits[j].ID {
				t.Fatalf("run %d differs: %v against %v", i, idsOf(first), idsOf(again))
			}
		}
	}
	// Equal score: the id decides, ascending, as in G1's ranking.
	if first.Hits[0].ID != "a.go#Same" {
		t.Errorf("got %v, want the lower id first on a tie", idsOf(first))
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/ask/ -run TestRun -v
```

- [ ] **Schritt 3: Die Implementierung schreiben**

`internal/code/ask/ask.go`:

```go
package ask

import (
	"regexp"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/pagerank"
)

// defaultLimit is how many hits an answer carries when the caller says
// nothing.
const defaultLimit = 8

// graphWeight is how much connectivity may lift a blended score. A
// lexically perfect node scores 1 on its axis; at 0.5 the walk can reorder
// near-ties and separate a wired-in hit from an isolated same-word collision,
// without overruling a clear lexical winner.
const graphWeight = 0.5

// rescueFloor is the share of the top node's mass a node needs to join the
// results without having matched a single word. It is what surfaces the helper
// or the configuration a task depends on and never named.
const rescueFloor = 0.15

// testRankPenalty de-ranks a test file multiplicatively.
//
// Tests mirror the tokens of the code they exercise, so a test can out-score
// the definition on a lexical tie and land on top -- observed as an answer that
// sent a reader into a Test... function. Multiplicative, so tests stay in the
// results: "where are the tests" is a fair question.
const testRankPenalty = 0.35

// testSeeking is a query that asks about tests, which lifts the penalty
// entirely. Without this a question about tests answers with everything except
// the tests.
var testSeeking = regexp.MustCompile(`(?i)\b(tests?|specs?|coverage|assert(?:ion)?s?|fixtures?|mocks?)\b`)

// isTestPath reports whether a path holds tests. Go's _test.go, and the
// directory names the reference also covers.
var isTestPath = regexp.MustCompile(`(^|/)(tests?|__tests__|spec)/|_test\.go$|(\.test|\.spec)\.[a-z]+$`)

// Options are the knobs of the RANKING. Whether the source is inlined is not
// one of them: that is Inline's argument (source.go), because the ranking does
// not read a single file and must not start.
type Options struct {
	Limit int
	In    string
}

// Hit is one answer, with both axes kept apart so a reader can see WHY it is
// here.
type Hit struct {
	ID        model.NodeID `json:"id"`
	Path      string       `json:"path"`
	Span      model.Span   `json:"span"`
	Signature string       `json:"signature,omitempty"`
	Score     float64      `json:"score"`
	Lexical   float64      `json:"lexical"`
	Graph     float64      `json:"graph"`
	Code      string       `json:"code,omitempty"`
}

// Answer is a whole reply.
type Answer struct {
	Query string `json:"query"`
	Hits  []Hit  `json:"hits"`
	Note  string `json:"note,omitempty"`
}

// Run answers one question.
//
// Four steps, in this order and for this reason:
//
//  1. lexical -- every document gets an idf-weighted word score, test files
//     de-ranked already, so a test seeds the walk with less mass too
//  2. seed    -- those scores ARE the personalized PageRank seed
//  3. rescue  -- a node the walk found central joins even with no word match
//  4. blend   -- (lexical/max + 0.5*graph) * testFactor, de-ranked AGAIN
//     because dividing by max would otherwise restore a winning test to 1.0
func Run(g *model.Graph, ix *lexicon.Index, query string, opts Options) Answer {
	if opts.In != "" {
		ix = ix.Filter(opts.In)
	}
	terms := lexicon.Counts(lexicon.Tokenize(query))
	factor := testFactor(query)
	w := weights(terms, ix)

	byID := make(map[model.NodeID]model.Node, len(g.Nodes))
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}

	lex := map[model.NodeID]float64{}
	var maxLex float64
	for _, d := range ix.Docs {
		n, ok := byID[d.ID]
		if !ok {
			continue
		}
		score := lexical(terms, d, w, ix.AvgBodyLen) * factor(n.Path)
		if score <= 0 {
			continue
		}
		lex[d.ID] = score
		if score > maxLex {
			maxLex = score
		}
	}

	pr := walk(g, lex, opts.In)

	candidates := map[model.NodeID]bool{}
	for id := range lex {
		candidates[id] = true
	}
	for _, s := range pr {
		if s.Score >= rescueFloor {
			candidates[s.ID] = true
		}
	}
	prByID := make(map[model.NodeID]float64, len(pr))
	for _, s := range pr {
		prByID[s.ID] = s.Score
	}

	var hits []Hit
	for id := range candidates {
		n, ok := byID[id]
		if !ok {
			continue
		}
		lexNorm := 0.0
		if maxLex > 0 {
			lexNorm = lex[id] / maxLex
		}
		graph := prByID[id]
		score := (lexNorm + graphWeight*graph) * factor(n.Path)
		if score <= 0 {
			continue
		}
		hits = append(hits, Hit{
			ID: id, Path: n.Path, Span: n.Span, Signature: n.Signature,
			Score: score, Lexical: lexNorm, Graph: graph,
		})
	}

	// Descending by score, ascending by id on a tie -- the same rule G1's
	// ranking uses, so two layers of this system cannot disagree about order.
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].ID < hits[j].ID
	})

	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}

	a := Answer{Query: query, Hits: hits}
	if len(hits) == 0 {
		// An empty list reads as "there is no such code". Say which it was.
		a.Note = "no matching nodes -- try different words, or run `loomux graph build` if the graph is empty"
	}
	return a
}

// walk runs personalized PageRank over the wiring edges, seeded by the lexical
// scores.
//
// The prefix narrows the WALK and not only the seed: without that, a neighbour
// outside the prefix could clear the rescue floor and surface, which would
// defeat the promise that a filter applies before scoring. G1's Prepare takes
// exactly this filter.
func walk(g *model.Graph, seed map[model.NodeID]float64, prefix string) []pagerank.Scored {
	if len(seed) == 0 {
		return nil
	}
	keep := func(id model.NodeID) bool { return true }
	if prefix != "" {
		paths := make(map[model.NodeID]string, len(g.Nodes))
		for _, n := range g.Nodes {
			paths[n.ID] = n.Path
		}
		keep = func(id model.NodeID) bool { return underPrefix(paths[id], prefix) }
	}
	return pagerank.Rank(pagerank.Prepare(g, keep), seed, pagerank.Options{})
}

// underPrefix compares whole path segments, so "widgets" never matches
// "widgets-extra".
func underPrefix(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

// testFactor is the de-ranking multiplier for a path, or 1 throughout when the
// query itself asks about tests.
func testFactor(query string) func(path string) float64 {
	if testSeeking.MatchString(query) {
		return func(string) float64 { return 1 }
	}
	return func(path string) float64 {
		if isTestPath.MatchString(path) {
			return testRankPenalty
		}
		return 1
	}
}
```

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/code/ask/ -count=1 -cover
```

Erwartet: PASS. `TestRunDoesNotLetConnectivityOverruleAClearLexicalWinner` und
`TestRunRescuesACentralNodeTheQueryNeverNamed` sind die beiden, die an den
Konstanten hängen — schlägt einer fehl, ist eine Konstante falsch übernommen
und nicht der Test zu lockern.

- [ ] **Schritt 5: Commit**

```sh
git add internal/code/ask
git commit -F ../msg.txt
```

Nachricht:

```
Blend the lexical score with the graph walk

Lexical proposes, graph disposes. The word scores are the personalized
PageRank seed, and the blend is (lexical/max + 0.5*graph) * testFactor.
At 0.5 connectivity reorders near-ties and separates a wired-in hit from
an isolated same-word collision, without overruling a node the query
plainly names.

A node that matched no word joins the results when the walk gives it 15%
of the top node's mass: that is what surfaces the helper a task depends
on and never mentioned.

The test penalty applies twice, to the raw score and again after
normalization. With only the first, dividing by max restores a winning
test to 1.0; with only the second, a test still seeds the walk as though
it were the answer. It lifts entirely when the question asks about
tests, or a question about tests answers with everything but.
```

---

### Task 6: `internal/code/ask` — der Quelltext daneben

**Files:**
- Create: `internal/code/ask/source.go`
- Test: `internal/code/ask/source_test.go`

**Interfaces:**
- Consumes: `Hit`, `Answer` aus Task 5.
- Produces: `func Inline(root string, a *Answer, full bool)`

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package ask_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/ask"
)

func fileWithLines(t *testing.T, n int) (root, rel string) {
	t.Helper()
	root = t.TempDir()
	rel = "p/big.go"
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString("line ")
		b.WriteString(strings.Repeat("x", 3))
		b.WriteString("\n")
	}
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, rel
}

func TestInlineCutsTheSpanOutOfTheFile(t *testing.T) {
	root, rel := fileWithLines(t, 20)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "L3-L5"}}}

	ask.Inline(root, &a, false)
	got := a.Hits[0].Code
	if strings.Count(got, "\n") != 2 {
		t.Fatalf("got %q, want exactly the three lines of the span", got)
	}
}

func TestInlineCapsAtEightyLinesAndSaysWhereTheRestIs(t *testing.T) {
	root, rel := fileWithLines(t, 300)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "L1-L250"}}}

	ask.Inline(root, &a, false)
	got := a.Hits[0].Code
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	// 80 lines plus the marker. The marker matters more than the cap: a reader
	// who needs the rest has to be told how to get it.
	if len(lines) != 81 {
		t.Fatalf("got %d lines, want 80 plus a marker", len(lines))
	}
	marker := lines[len(lines)-1]
	if !strings.Contains(marker, "170") || !strings.Contains(marker, rel+":L1-L250") {
		t.Errorf("marker %q must name how many lines are left and where they are", marker)
	}
}

func TestInlineFullLiftsTheCap(t *testing.T) {
	root, rel := fileWithLines(t, 300)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "L1-L250"}}}

	ask.Inline(root, &a, true)
	lines := strings.Split(strings.TrimRight(a.Hits[0].Code, "\n"), "\n")
	if len(lines) != 250 {
		t.Fatalf("got %d lines, want all 250", len(lines))
	}
}

func TestInlineClampsASpanPastTheEndOfTheFile(t *testing.T) {
	root, rel := fileWithLines(t, 5)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "L3-L900"}}}

	ask.Inline(root, &a, false)
	// A stale graph against an edited file: the span may point past the end.
	// Clamping beats an empty answer, and beats a panic by more.
	if a.Hits[0].Code == "" {
		t.Fatal("a span reaching past the file must still yield what is there")
	}
}

func TestInlineLeavesAHitAloneWhenTheFileIsGone(t *testing.T) {
	root := t.TempDir()
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: "gone.go", Span: "L1-L2"}}}

	ask.Inline(root, &a, false)
	// A file deleted since the build is drift, not a failure of the answer: the
	// hit keeps its location and loses only its excerpt.
	if a.Hits[0].Code != "" {
		t.Errorf("got %q, want no code and no error", a.Hits[0].Code)
	}
}

func TestInlineIgnoresAnUnreadableSpan(t *testing.T) {
	root, rel := fileWithLines(t, 5)
	a := ask.Answer{Hits: []ask.Hit{{ID: "x", Path: rel, Span: "nonsense"}}}

	ask.Inline(root, &a, false)
	if a.Hits[0].Code != "" {
		t.Errorf("got %q, want nothing for a span that does not parse", a.Hits[0].Code)
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/ask/ -run TestInline -v
```

- [ ] **Schritt 3: Die Implementierung schreiben**

```go
package ask

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// maxSpanLines caps an inlined excerpt. The reference's figure.
//
// There is no crux here, and that is not a gap: Graft's crux is an LLM-chosen
// excerpt stored on the node (its Tier 2), and this binary has no LLM in the
// path. The line slice below is Graft's OWN fallback for a node without one,
// and when a producer for a crux ever exists it takes precedence in exactly
// this function, with the flags unchanged.
const maxSpanLines = 80

// Inline attaches the source of each hit's span.
//
// Never fatal: a file deleted or shortened since the build is drift, and a hit
// then keeps its location and loses only its excerpt. An answer that fails
// because a file moved would be worse than one that is merely thinner.
func Inline(root string, a *Answer, full bool) {
	for i := range a.Hits {
		h := &a.Hits[i]
		from, to, ok := h.Span.Lines()
		if !ok {
			continue
		}
		code, ok := slice(filepath.Join(root, filepath.FromSlash(h.Path)), from, to, full, h.Path, h.Span)
		if !ok {
			continue
		}
		h.Code = code
	}
}

// slice reads lines [from, to] of a file, capped unless full.
func slice(abs string, from, to int, full bool, rel string, span model.Span) (string, bool) {
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", false
	}
	lines := strings.Split(string(b), "\n")
	if from < 1 {
		from = 1
	}
	// A stale graph against an edited file: clamp rather than answer nothing.
	if to > len(lines) {
		to = len(lines)
	}
	if from > to {
		return "", false
	}
	out := lines[from-1 : to]
	if !full && len(out) > maxSpanLines {
		rest := len(out) - maxSpanLines
		out = append(out[:maxSpanLines:maxSpanLines],
			fmt.Sprintf("... (+%d more lines; open %s:%s)", rest, rel, span))
	}
	return strings.Join(out, "\n"), true
}

```

Der Import-Block braucht `model`. **`Span.Lines` wird hier nicht angelegt** — es
steht seit G2a, Task 1, Schritt 5 in `internal/code/model/graph.go`, weil der
Extraktor es dort schon braucht. Ist die Methode nicht da, ist G2a nicht
vollständig ausgeführt: melden, nicht hier nachbauen.

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/code/... -count=1 -cover
```

- [ ] **Schritt 5: Commit**

```sh
git add internal/code
git commit -F ../msg.txt
```

Nachricht:

```
Inline the source of a hit's span, capped

The default answer is a locator: score, id, path, span, signature. The
source arrives only when asked for, and then capped at 80 lines with a
marker naming how many are left and where -- a reader who needs the rest
has to be told how to get it. Full lifts the cap.

There is no crux and that is not a gap. Graft's crux is an LLM-chosen
excerpt stored on the node, and this binary has no LLM in the path. The
line slice is Graft's own fallback for a node without one, and a crux
would take precedence in exactly this function with the flags unchanged.

The span is parsed by model.Span.Lines, which G2a put on the type for
exactly this reason: the extractor writes the form and the answer reads
it, and two copies of one parser are one too many.
```

---

### Task 7: `internal/code/ask` — vor der Antwort neu bauen

**Files:**
- Create: `internal/code/ask/refresh.go`
- Test: `internal/code/ask/refresh_test.go`

**Interfaces:**
- Consumes: `freshness.Probe`.
- Produces:

```go
// Rebuild is what EnsureFresh calls when the tree moved. The CLI passes its
// own build, so this package stays free of the extractor.
type Rebuild func() error

func EnsureFresh(root, extractor string, rebuild Rebuild, notice func(string))
```

`EnsureFresh` nimmt den Bau als Funktion, damit `ask` den Extraktor nicht
importiert — dieselbe Trennung, die `freshness` schon trägt.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
package ask_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/sourceset"
)

func TestEnsureFreshRebuildsWhenThereIsNoRecord(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	built := 0

	ask.EnsureFresh(root, "go/1", func() error { built++; return nil }, nil)
	// "No record" means unknown, never clean. A fresh clone has no state at
	// all, and the first question must not answer from nothing.
	if built != 1 {
		t.Fatalf("built %d times, want 1", built)
	}
}

func TestEnsureFreshDoesNothingOnACleanTree(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stat, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{}
	for _, f := range stat {
		hashes[f.Rel] = hashOf(t, f.Abs)
	}
	if err := freshness.Write(root, "go/1", stat, hashes); err != nil {
		t.Fatal(err)
	}
	built := 0

	ask.EnsureFresh(root, "go/1", func() error { built++; return nil }, nil)
	if built != 0 {
		t.Fatalf("built %d times, want none: the probe is the whole point", built)
	}
}

func TestEnsureFreshSurvivesAFailedRebuild(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var notices []string

	ask.EnsureFresh(root, "go/1",
		func() error { return errors.New("disk full") },
		func(s string) { notices = append(notices, s) })

	// Never fatal: a failed rebuild answers from the graph on disk. A question
	// that works today must not start failing because a rebuild could not run.
	if len(notices) == 0 || !strings.Contains(strings.Join(notices, " "), "disk full") {
		t.Fatalf("notices %v must carry the reason", notices)
	}
}

func TestEnsureFreshSkipsWhenAnotherRunHoldsTheLock(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ask.LockPath(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ask.LockPath(root), []byte("999999"), 0o644); err != nil {
		t.Fatal(err)
	}
	built := 0

	ask.EnsureFresh(root, "go/1", func() error { built++; return nil }, nil)
	// No stampede: concurrent questions must not pile rebuilds on each other.
	// The loser answers from what is on disk.
	if built != 0 {
		t.Fatalf("built %d times while the lock was held", built)
	}
}

func TestEnsureFreshReleasesTheLock(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ask.EnsureFresh(root, "go/1", func() error { return nil }, nil)
	if _, err := os.Stat(ask.LockPath(root)); !os.IsNotExist(err) {
		t.Fatal("the lock must be gone once the rebuild finished")
	}
}

func TestEnsureFreshBreaksAStaleLock(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ask.LockPath(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ask.LockPath(root), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(ask.LockPath(root), old, old); err != nil {
		t.Fatal(err)
	}
	built := 0

	ask.EnsureFresh(root, "go/1", func() error { built++; return nil }, nil)
	// A run killed mid-rebuild leaves its lock behind. Without a staleness
	// rule, the graph would never refresh again on that checkout.
	if built != 1 {
		t.Fatalf("built %d times, want 1: a stale lock must not be forever", built)
	}
}
```

Dazu der Testhelfer, in derselben Datei; der Import-Block braucht
`crypto/sha256`, `encoding/hex` und `time`:

```go
// hashOf is the hash a build would have recorded for this file.
func hashOf(t *testing.T, abs string) string {
	t.Helper()
	b, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/code/ask/ -run TestEnsureFresh -v
```

- [ ] **Schritt 3: Die Implementierung schreiben**

```go
package ask

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/store"
)

// lockStale is when a lock left behind by a killed run stops counting.
//
// Without a rule like this, one run killed mid-rebuild would stop that checkout
// from ever refreshing again -- and the failure would be silent, which is the
// worst kind.
const lockStale = time.Hour

// Rebuild is the build a caller hands in.
//
// A function and not an import: this package stays free of the extractor, the
// same separation freshness keeps, so the query path and the hook path can both
// use it later without dragging the parser along.
type Rebuild func() error

// LockPath is the file that keeps two rebuilds apart.
func LockPath(root string) string { return store.CachePath(root, "rebuild.lock") }

// EnsureFresh probes the tree and rebuilds when it moved.
//
// Freshness belongs in the query path, and the reference explains why from
// experience: with the rebuild hanging off a session hook, every question
// between the first edit and the end of a turn answered from a graph that did
// not know the file just changed -- and an edit made outside the agent
// triggered nothing at all.
//
// Three properties, all of them load-bearing:
//
//   - never fatal: a failed probe or rebuild answers from the graph on disk
//   - no stampede: a lock, and the loser does not queue -- it answers
//   - writes only what a question reads: the graph, the sidecar, the record
func EnsureFresh(root, extractor string, rebuild Rebuild, notice func(string)) {
	say := func(format string, args ...any) {
		if notice != nil {
			notice(fmt.Sprintf(format, args...))
		}
	}

	drift, err := freshness.Probe(root, extractor)
	if err != nil {
		say("freshness probe failed, answering from the graph on disk: %v", err)
		return
	}
	if drift != nil && drift.Clean() {
		return
	}
	if drift != nil {
		say("%d files moved, rebuilding the graph", drift.Count())
	} else {
		// No record: unknown, never clean.
		say("no freshness record, building the graph")
	}

	if !lock(root) {
		say("another run is rebuilding, answering from the graph on disk")
		return
	}
	defer unlock(root)

	if err := rebuild(); err != nil {
		say("rebuild failed, answering from the graph on disk: %v", err)
	}
}

// lock takes the rebuild lock, or reports that someone else holds it.
//
// O_EXCL is the whole mechanism: the create either wins or it does not, with no
// window between the check and the take.
func lock(root string) bool {
	path := LockPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		fmt.Fprint(f, os.Getpid())
		f.Close()
		return true
	}
	// Held. Stale?
	info, statErr := os.Stat(path)
	if statErr != nil || time.Since(info.ModTime()) < lockStale {
		return false
	}
	if err := os.Remove(path); err != nil {
		return false
	}
	return lock(root)
}

// unlock releases the lock. A failure here is not worth reporting: the next run
// finds it stale.
func unlock(root string) {
	_ = os.Remove(LockPath(root))
}
```

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/code/ask/ -count=1 -cover
```

Der rekursive Zweig in `lock` (abgelaufene Sperre, zweiter Versuch) ist genau
der Fall, den `TestEnsureFreshBreaksAStaleLock` fährt.

- [ ] **Schritt 5: Commit**

```sh
git add internal/code/ask
git commit -F ../msg.txt
```

Nachricht:

```
Rebuild before answering, under a lock, never fatally

Freshness belongs in the query path. With the rebuild hanging off a
session hook, every question between the first edit and the end of a
turn answered from a graph that did not know the file just changed, and
an edit made outside the agent triggered nothing at all.

Never fatal: a failed probe or rebuild answers from the graph on disk. A
question that works today must not start failing because a rebuild could
not run. No stampede: the lock is O_EXCL, and the loser does not queue,
it answers. A lock left behind by a killed run goes stale after an hour,
because a silent refusal to ever refresh again is the worst failure of
the three.

The build arrives as a function, so this package never imports the
extractor -- the separation freshness already keeps.
```

---

### Task 8: `loomux graph ask`

**Files:**
- Modify: `internal/cli/graph.go`
- Test: `internal/cli/graph_test.go` (anhängen)

**Interfaces:**
- Consumes: `ask.Run`, `ask.Inline`, `ask.EnsureFresh`, `lexicon.Read`,
  `store.Read`, `buildGraph` aus G2a.
- Produces: Unterbefehl `ask` mit `--limit`, `--source`, `--full`, `--in`,
  `--json`, `--no-refresh`, `--root`.

- [ ] **Schritt 1: Den fehlschlagenden Test schreiben**

```go
func TestGraphAskAnswersWithLocationsAndNoCode(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	if code := graphCommand([]string{"ask", "run the thing", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	report := out.String()
	if !strings.Contains(report, "lib/lib.go") {
		t.Fatalf("answer %q must name the file", report)
	}
	// The default is a locator. Source arrives only when asked for.
	if strings.Contains(report, "func Run() {}") {
		t.Errorf("answer %q must not inline source without --source", report)
	}
}

func TestGraphAskSourceInlinesTheSpan(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	if code := graphCommand([]string{"ask", "run", "--root", root, "--source"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "func Run()") {
		t.Errorf("answer %q must carry the definition", out.String())
	}
}

func TestGraphAskFindsASymbolByAWordOnlyInItsBody(t *testing.T) {
	files := sample()
	files["lib/lib.go"] = "package lib\n\nfunc Run() { retryWithBackoff() }\n\nfunc retryWithBackoff() {}\n"
	root := repo(t, files)
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	// The word lives in Run's body and in retryWithBackoff's name. Body
	// indexing is what makes the first findable at all -- wiring.json does not
	// carry the body, the sidecar does.
	if code := graphCommand([]string{"ask", "backoff", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "retryWithBackoff") {
		t.Errorf("answer %q must find the symbol", out.String())
	}
}

func TestGraphAskFindsAFileByAWordInItsImportHeader(t *testing.T) {
	files := sample()
	files["lib/lib.go"] = "package lib\n\nimport \"encoding/json\"\n\nfunc Run() { _ = json.Marshal }\n"
	root := repo(t, files)
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	// The file node's residual carries what no symbol span covers.
	if code := graphCommand([]string{"ask", "encoding", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "lib/lib.go") {
		t.Errorf("answer %q must find the file", out.String())
	}
}

func TestGraphAskBuildsWhenNothingIsBuiltYet(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer

	// A fresh clone has no state: the graph is gitignored and was never checked
	// out. No record means unknown, so the first question builds.
	if code := graphCommand([]string{"ask", "run", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if _, err := os.Stat(store.WiringPath(root)); err != nil {
		t.Fatalf("ask must have built the graph: %v", err)
	}
}

func TestGraphAskNoRefreshAnswersFromWhatIsThere(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer

	code := graphCommand([]string{"ask", "run", "--root", root, "--no-refresh"}, nil, &out, &errOut)
	// Nothing built and no rebuild allowed: say so rather than answer from
	// nothing.
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "graph build") {
		t.Errorf("stderr %q must point at the command that fixes it", errOut.String())
	}
}

func TestGraphAskJSONCarriesTheHits(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	if code := graphCommand([]string{"ask", "run", "--root", root, "--json"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var got struct {
		Query string `json:"query"`
		Hits  []struct {
			ID      string  `json:"id"`
			Score   float64 `json:"score"`
			Lexical float64 `json:"lexical"`
			Graph   float64 `json:"graph"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out.String())
	}
	if got.Query != "run" || len(got.Hits) == 0 {
		t.Fatalf("got %+v", got)
	}
	// Both axes are reported, so a reader can see WHY a hit is there.
	if got.Hits[0].Lexical == 0 && got.Hits[0].Graph == 0 {
		t.Error("a hit must say which axis put it there")
	}
}

func TestGraphAskWithoutAQueryIsAUsageError(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"ask"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
}

func TestGraphAskOnAMissedQuerySaysSoAndExitsZero(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	code := graphCommand([]string{"ask", "quantum entanglement", "--root", root}, nil, &out, &errOut)
	// Asking and finding nothing is a successful question with an empty answer,
	// not a failure -- a script must be able to tell the two apart.
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), "no matching nodes") {
		t.Errorf("answer %q must say it found nothing", out.String())
	}
}
```

- [ ] **Schritt 2: Lauf, der fehlschlagen muss**

```sh
go test ./internal/cli/ -run TestGraphAsk -v
```

Erwartet: FAIL, `exit 2` überall — `ask` ist kein Unterbefehl.

- [ ] **Schritt 3: Die Implementierung schreiben**

In `graphCommand` einen Fall ergänzen:

```go
	case "ask":
		return graphAsk(args[1:], stdout, stderr)
```

und in `graphUsage` eine Zeile. Dann:

```go
// graphAsk answers a question from the graph.
func graphAsk(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph ask", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	limit := fs.Int("limit", 0, "how many hits to report (default 8)")
	in := fs.String("in", "", "narrow to nodes under this path prefix, before scoring")
	source := fs.Bool("source", false, "inline the source at each hit")
	full := fs.Bool("full", false, "with --source: inline the whole span, uncapped")
	asJSON := fs.Bool("json", false, "write the answer as JSON")
	noRefresh := fs.Bool("no-refresh", false, "never rebuild, answer from the graph on disk")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, `loomux graph ask: one query required: loomux graph ask "<question>" [flags]`)
		return 2
	}
	query := fs.Arg(0)

	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph ask: %v\n", err)
		return 1
	}

	if !*noRefresh {
		// Notices go to stderr, so a piped answer stays an answer.
		ask.EnsureFresh(project, golang.Version,
			func() error { _, _, err := writeEverything(project); return err },
			func(s string) { fmt.Fprintf(stderr, "loomux graph ask: %s\n", s) })
	}

	g, err := store.Read(project)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(stderr, "loomux graph ask: no graph. Run `loomux graph build` first.")
			return 1
		}
		fmt.Fprintf(stderr, "loomux graph ask: %v\n", err)
		return 1
	}
	ix, err := lexicon.Read(project)
	if err != nil {
		// The sidecar is a cache: without it, tokenize live off the graph. The
		// body text is gone from the written graph, so a body-only word will
		// not be found -- say so rather than answer worse in silence.
		fmt.Fprintf(stderr, "loomux graph ask: no ask index, ranking on names and paths only: %v\n", err)
		ix = lexicon.Build(g)
	}

	answer := ask.Run(g, ix, query, ask.Options{Limit: *limit, In: *in})
	if *source {
		ask.Inline(project, &answer, *full)
	}

	if *asJSON {
		body, err := json.MarshalIndent(answer, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "loomux graph ask: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", body)
		return 0
	}
	fmt.Fprint(stdout, askReport(answer))
	return 0
}

// writeEverything is one whole build: the graph, the ask sidecar and the
// freshness record.
//
// Both callers use it -- `graph build`, which owns the flags and the report, and
// the rebuild `graph ask` triggers, which owns neither. Two copies of this
// sequence would be two places where the sidecar can be forgotten, and a
// forgotten sidecar is a silently worse answer.
func writeEverything(root string) (*model.Graph, buildStats, error) {
	g, stats, err := buildGraph(root)
	if err != nil {
		return nil, buildStats{}, err
	}
	if err := store.Write(root, g); err != nil {
		return nil, buildStats{}, err
	}
	// From the graph in memory, which still carries the body text: the written
	// file has it stripped, and that is what makes the sidecar necessary rather
	// than redundant.
	if err := lexicon.Write(root, lexicon.Build(g)); err != nil {
		return nil, buildStats{}, err
	}
	if err := freshness.Write(root, golang.Version, stats.files, stats.hashes); err != nil {
		return nil, buildStats{}, err
	}
	return g, stats, nil
}

// askReport is the human form: one block per hit, location first.
func askReport(a ask.Answer) string {
	if a.Note != "" {
		return a.Note + "\n"
	}
	var b strings.Builder
	for i, h := range a.Hits {
		fmt.Fprintf(&b, "%d. %s  %s:%s  (%.3f lex %.3f graph %.3f)\n",
			i+1, h.ID, h.Path, h.Span, h.Score, h.Lexical, h.Graph)
		if h.Signature != "" {
			fmt.Fprintf(&b, "   %s\n", h.Signature)
		}
		if h.Code != "" {
			for _, line := range strings.Split(h.Code, "\n") {
				fmt.Fprintf(&b, "   | %s\n", line)
			}
		}
	}
	return b.String()
}
```

`graphBuild` aus G2a wird dabei auf `writeEverything` umgestellt: sein Rumpf
schrumpft auf Flaggen lesen, `writeEverything` rufen, Bericht schreiben. Die
beiden Warnzeilen, die G2a für eine fehlgeschlagene Beiakte und eine
fehlgeschlagene Frischeakte ausgab, entfallen — hier ist beides ein Fehler des
Baus, weil ein Bau, der die Beiakte nicht schreibt, eine stillschweigend
schlechtere Antwort hinterlässt.

- [ ] **Schritt 4: Lauf, der grün sein muss**

```sh
go test ./internal/cli/ -count=1
```

- [ ] **Schritt 5: Eine echte Frage an dieses Repo stellen**

```sh
go build -o bin/loomux.exe ./cmd/loomux
./bin/loomux.exe graph build --root .
./bin/loomux.exe graph ask "how does the write barrier decide what to refuse" --limit 5
./bin/loomux.exe graph ask "personalized pagerank dangling mass" --limit 5 --source
```

Das ist die Abnahme dieser Stufe, die kein Test leistet: **stehen die Antworten
auf den Plätzen, an denen ein Mensch sie erwartet?** Wenn nicht, liegt der
Fehler an einer Konstante oder an der Feldgewichtung, und der Befund gehört in
die Paritätsakte — nicht in eine stillschweigende Korrektur.

- [ ] **Schritt 6: Commit**

```sh
git add internal/cli
git commit -F ../msg.txt
```

Nachricht:

```
Add `loomux graph ask`

A locator by default: id, path, span, signature, and both score axes so
a reader can see whether a hit is there on words or on wiring. --source
inlines the span, --full lifts the cap, --in narrows before scoring.

It rebuilds first unless told not to, because no record means unknown
and a fresh clone has no state at all -- the graph is gitignored and was
never checked out. Notices go to stderr so a piped answer stays an
answer.

A question that matches nothing exits 0 with a note. Asking and finding
nothing is a successful question, and a script has to be able to tell it
from a broken one.
```

---

### Task 9: Dokumentation und Messungen

**Files:**
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md`
- Modify: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`
- Modify: `docs/en/architecture.md`, `docs/de/architecture.md`
- Modify: `README.md`, `README.de.md`

**Interfaces:**
- Consumes: der fertige Befehl aus Task 8.
- Produces: nichts im Code.

- [ ] **Schritt 1: `cli-reference.md` in beiden Sprachen ergänzen**

`graph ask` mit allen Flaggen, dem Standard `--limit 8`, und den beiden Sätzen,
die sonst rückgefragt werden: dass ohne `--source` kein Quelltext kommt, und
dass der Befehl bei Drift selbst neu baut, sofern `--no-refresh` fehlt.

- [ ] **Schritt 2: `architecture.md` in beiden Sprachen ergänzen**

Die beiden neuen Pakete, die Beiakte, und der Satz über die Richtung: `ask`
importiert den Extraktor nicht, der Bau kommt als Funktion herein.

- [ ] **Schritt 3: Die Messungen fahren und eintragen**

Mit Datum und Uhrzeit, Gegenstand, Basis gegen Änderung, kalt und warm:

```sh
# Die Antwortzeit, warm, mit Beiakte. Zehn Läufe, Median.
./bin/loomux.exe graph ask "write barrier refuse" --limit 8 --no-refresh

# Dieselbe Frage ohne Beiakte -- das ist die Messung, die den Existenzgrund
# der Beiakte für loomux belegt oder widerlegt. Grafts Zahl: ~45 % der
# Abfragezeit bei 32k Knoten.
mv .loomux/state/graph/cache/ask-index.json /tmp/ && \
  ./bin/loomux.exe graph ask "write barrier refuse" --limit 8 --no-refresh

# Kalt: frischer Prozess, Graph da, Beiakte da.
```

**Fällt die Differenz auf diesem Repo klein aus** — es hat rund 4.000 Knoten,
nicht 32.000 —, gehört genau das in den Eintrag. Eine Beiakte, die hier nichts
bringt, ist trotzdem richtig gebaut, weil sie mit dem Repo skaliert; aber die
Zahl darf nicht schöngeschrieben werden.

- [ ] **Schritt 4: Die READMEs nachziehen**

Der Absatz zum Stand sagt, dass die Abfrage steht, und nennt die gemessene
Antwortzeit. Die Fahrplanzeilen zu `graph ask` und zum Retrieval wandern auf
den erreichten Stand. Die Zeile, die in G1 noch sagte, die Retrieval-Zeit sei
„therefore still unmeasured", wird durch die Zahl ersetzt.

- [ ] **Schritt 5: Commit**

```sh
git add docs README.md README.de.md
git commit -F ../msg.txt
```

Nachricht:

```
Document the query and measure it

The CLI reference carries ask with its flags and the two things that
would otherwise be asked twice: no source without --source, and it
rebuilds on drift unless told not to.

Benchmarks carry the answer time with and without the sidecar. This
repository has roughly four thousand nodes against the reference's
thirty-two thousand, so a small difference here is a fact about the
repository and is recorded as one rather than rounded up.
```

---

### Task 10: Das ganze Tor, die Vektoren und die Mutationsrunde

**Files:**
- Create: `docs/.superpowers/parity/code-g2b.md`

**Interfaces:**
- Consumes: alles Vorige.
- Produces: die Paritätsakte der Stufe.

- [ ] **Schritt 1: Die portierten Vektoren gegenprüfen**

Abschnitt 12.3 der Spec nennt sie namentlich. Jeder hat einen Test in dieser
Stufe; hier wird die Zuordnung festgehalten, damit keiner stillschweigend
fehlt:

| Quelle in Graft | Test dieser Stufe |
|---|---|
| `test/ask.test.ts:112` — ohne `--source` nur Zeiger | `TestGraphAskAnswersWithLocationsAndNoCode` |
| `test/ask.test.ts:154` — Span-Rückfall mit `--source` | `TestGraphAskSourceInlinesTheSpan`, `TestInlineCutsTheSpanOutOfTheFile` |
| `test/ask.test.ts:470` — Symbol über ein Wort im Rumpf | `TestGraphAskFindsASymbolByAWordOnlyInItsBody` |
| `test/ask.test.ts:491` — Datei über ein Wort im Modulrumpf | `TestGraphAskFindsAFileByAWordInItsImportHeader` |
| `test/ask.test.ts:40`, `:72` — Test-De-Rankung, auch als stärkster Rohtreffer | `TestRunDeRanksATestFile`, `TestRunDeRanksATestEvenWhenItIsTheStrongestRawMatch`, `TestRunLiftsThePenaltyWhenTheQueryAsksForTests` |
| `test/ask.test.ts:890` — `--in` segmentbewusst | `TestRunFiltersBeforeScoringAndSegmentAware`, `TestFilterIsSegmentAware` |
| `test/ask.test.ts:928` — IDF je Bereich weicht vom globalen ab | `TestFilterRecomputesDocumentFrequencyOverTheRemainder` |
| `test/ask-index.test.ts` — die Beiakte ist ein exakter Cache | `TestWriteThenReadRoundTrips`, `TestBuildCountsDocumentFrequencyAcrossFields` |

Fehlt einer, ist er jetzt zu schreiben.

- [ ] **Schritt 2: Das Tor ganz fahren**

```sh
gofmt -l cmd internal
go vet ./...
go test ./... -count=1 -covermode=set -coverpkg=./... -coverprofile=coverage.out
go run ./cmd/loomux dev covergate --profile coverage.out
go build -o bin/loomux.exe ./cmd/loomux
```

- [ ] **Schritt 3: Die Mutationsrunde fahren**

```sh
go run ./cmd/loomux dev mutants internal/code/lexicon internal/code/ask
```

Die Konstanten sind hier die interessanten Mutanten: eine Runde, die `0.5` zu
`0.4` macht oder `0.15` zu `0.16`, und deren Mutant überlebt, sagt, dass kein
Test den Wert wirklich festnagelt. Das ist ein fehlender Test und keine
Äquivalenz.

- [ ] **Schritt 4: Die Paritätsakte schreiben**

`docs/.superpowers/parity/code-g2b.md`, nach dem Muster von `code-g1.md`: Kopf
mit Datum, Kompilat und Zahlen, die Tabelle der Überlebenden mit je einer
Verfügung, und die Zuordnungstabelle aus Schritt 1.

Dazu ein Abschnitt für die Verfügungen ohne Mutation:

- **Kein Crux.** Der Zeilenschnitt ist Grafts eigener Rückfall, nicht ein
  Ersatz für etwas Fehlendes; Verweis auf 3.1 der Spec.
- **Die Beiakte ist ein Cache.** Ihr Verlust verschlechtert die Antwort und
  bricht sie nicht: ohne sie wird live tokenisiert, aber der Rumpf fehlt, und
  der Befehl sagt das auf stderr.
- **Die Messung zur Beiakte.** Was Schritt 3 von Task 9 ergeben hat, mit der
  Zahl, auch wenn sie den Aufwand auf diesem Repo nicht rechtfertigt.
- **Der Testfaktor wirkt zweimal.** Mit der Begründung, warum eine Stelle nicht
  genügt.

- [ ] **Schritt 5: Was G2 ganz offen lässt**

Am Ende der Akte, als Übergabe:

- `--lsp` gegen `gopls` — die optionale Folgestufe, Abschnitt 14 der Spec;
- `graph callers`, `blast`, `grep`, `skeleton`, `map`, `stats`, `viz` — G4, und
  mit `callers` die Normalisierung von `Depth` und `Direction`;
- die MCP-Anbindung und die sechs `graph_*`-Werkzeuge — G3;
- die `check`-Lanes und der Post-Edit-Hook — G4;
- `FileCard`, Crux, `confidence`-Rangfolge — mit ihren Erzeugern.

- [ ] **Schritt 6: Commit**

```sh
git add docs/.superpowers/parity/code-g2b.md
git commit -F ../msg.txt
```

Nachricht:

```
Sign off the G2b mutation round and the ported vectors

Every survivor carries a ruling, and the interesting ones here are the
constants: a mutant that turns 0.5 into 0.4 and survives says no test
pins the value, which is a missing test and not an equivalence.

The table maps each ported vector of the reference to the test that
carries it, so none can go missing quietly -- the G1 delta's "four ask
tests" turned out not to be an identifiable set, and this is the answer
to that.
```

---

## Self-Review

Gegen die Spec gelesen:

| Spec | Task |
|---|---|
| 10.1 Tokenisierung, Stoppliste, ASCII | 1 |
| 10.1 Beiakte, `df`, `docCount`, `avgBodyLen` | 2, 3 |
| 10.2 Feldgewichte, IDF, BM25 | 4 |
| 10.3 Saat, Verschmelzung, `GRAPH_WEIGHT`, `RESCUE_FLOOR` | 5 |
| 10.3 `--in` verengt Lauf **und** Lexik | 2 (`Filter`), 5 (`walk`) |
| 10.4 Test-De-Rankung zweimal, `wantsTests` | 5 |
| 10.5 Neubau vor der Antwort, Sperre, nie fatal | 7 |
| 10.6 Ausgabe, `--source`, `--full`, Kappung 80 | 6, 8 |
| 12.3 portierte Vektoren | 10 (Zuordnung), 2/5/6/8 (die Tests) |
| 12.4 Mutationsrunde | 10 |
| 13 Doku und Messungen | 9 |

**Drei Stellen, an denen dieser Plan eine Entscheidung trifft, die die Spec
offenlässt** — sie sind mit Begründung im Text und gehören in die Paritätsakte:

1. **Die Sperre.** Die Spec sagt „unter einem Lock", ohne zu sagen welcher. Das
   Repo hat keinen; `internal/testlock` ist Testunterstützung und kein
   Prozess-Lock. Task 7 baut einen über `O_CREATE|O_EXCL` mit einer Stunde
   Verfall. Der Verfall ist der Teil, der eine Begründung braucht: ohne ihn
   hört ein Checkout nach einem abgebrochenen Lauf **still** auf, je wieder zu
   aktualisieren.
2. **Ein `ask` ohne Beiakte.** Die Spec sagt nicht, was dann gilt. Task 8
   tokenisiert live aus dem Graphen und sagt auf stderr, dass der Rumpf fehlt —
   eine schlechtere Antwort ist besser als keine, und eine stillschweigend
   schlechtere ist schlechter als beides.
3. **Exit 0 auf eine Frage ohne Treffer.** Die Spec nennt Exitcodes nur für
   `check`. Fragen und nichts finden ist eine erfolgreiche Frage; ein Skript
   muss das von einem Fehler unterscheiden können.

**Eine Voraussetzung aus G2a, die hier leicht übersehen wird:** `Span.Lines`
steht in `internal/code/model` und wird in Task 6 nur benutzt. Fehlt sie, ist
G2a nicht vollständig ausgeführt — und wer sie hier nachbaut, hat zwei Parser
für dieselbe Zeichenkette.
