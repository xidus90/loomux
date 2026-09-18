package lexicon_test

import (
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
				ID: "lib/cache.go#Cache.Fetch", Name: "Fetch", Kind: "method", Owner: "Cache",
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
	var fetch *lexicon.Doc
	for i := range ix.Docs {
		if ix.Docs[i].ID == "lib/cache.go#Cache.Fetch" {
			fetch = &ix.Docs[i]
		}
	}
	if fetch == nil {
		t.Fatal("the method's doc is missing")
	}
	if fetch.Name["fetch"] != 1 {
		t.Errorf("name bag = %v, want the bare name", fetch.Name)
	}
	if fetch.Path["cache"] != 1 || fetch.Path["lib"] != 1 {
		t.Errorf("path bag = %v, want the path's segments", fetch.Path)
	}
	if fetch.Body["entries"] != 1 {
		t.Errorf("body bag = %v, want the body's words", fetch.Body)
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
	_, err := lexicon.Read(t.TempDir())
	if err == nil {
		t.Fatal("got nil, want an error -- a caller decides whether to rebuild")
	}
	// The error has to carry the reason and not only exist: skipping the
	// ReadFile check leaves a nil body that json.Unmarshal rejects as well, so
	// "no sidecar" would arrive as "half a sidecar" and a caller reading the
	// wrapped cause would rebuild for the wrong reason.
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want it to wrap fs.ErrNotExist", err)
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
	if sub.AvgBodyLen <= 0 {
		t.Errorf("AvgBodyLen = %v, want > 0 over remaining documents", sub.AvgBodyLen)
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

func TestBuildLeavesAStopWordNameWithAnEmptyBag(t *testing.T) {
	g := graph()
	g.Nodes = append(g.Nodes, model.Node{
		ID: "lib/cache.go#Cache.Get", Name: "Get", Kind: "method", Owner: "Cache",
		Path: "lib/cache.go", BodyHash: "h", BodyText: "return c.entries[key]",
	})
	ix := lexicon.Build(g)

	// Index time and query time share one tokenizer, so a name that is a stop
	// word leaves no token behind. Pinned because it is surprising: a method
	// called Get is reachable through its path and its body, never its name.
	for _, d := range ix.Docs {
		if d.ID == "lib/cache.go#Cache.Get" && len(d.Name) != 0 {
			t.Errorf("name bag = %v, want empty -- get is a stop word", d.Name)
		}
	}
}

func TestBuildOrdersDocumentsById(t *testing.T) {
	g := graph()
	g.Nodes[0], g.Nodes[2] = g.Nodes[2], g.Nodes[0]
	ix := lexicon.Build(g)

	// A rebuilt sidecar has to be byte-identical to the one it replaces, so the
	// order may not follow the graph's.
	for i := 1; i < len(ix.Docs); i++ {
		if ix.Docs[i-1].ID > ix.Docs[i].ID {
			t.Fatalf("docs out of order at %d: %q before %q", i, ix.Docs[i-1].ID, ix.Docs[i].ID)
		}
	}
}

func TestReadRefusesAMalformedSidecar(t *testing.T) {
	root := t.TempDir()
	if err := lexicon.Write(root, lexicon.Build(graph())); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lexicon.Path(root), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := lexicon.Read(root); err == nil {
		t.Fatal("got nil, want an error -- half a sidecar is no sidecar")
	}
}

func TestWriteReportsAnUnusableCacheDirectory(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Dir(lexicon.Path(root))
	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
		t.Fatal(err)
	}
	// A file where the cache directory belongs: the sidecar cannot be written,
	// and the caller has to hear that rather than read a stale one later.
	if err := os.WriteFile(cache, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := lexicon.Write(root, lexicon.Build(graph())); err == nil {
		t.Fatal("got nil, want an error")
	}
}

func TestBuildAnEmptyGraphHasNoAverage(t *testing.T) {
	ix := lexicon.Build(&model.Graph{Meta: model.Meta{Version: 2, Extractor: "go/1"}})

	if ix.DocCount != 0 || ix.AvgBodyLen != 0 {
		t.Fatalf("got %+v, want an empty index -- not a division by zero", ix)
	}
}

func TestFilterOnAPrefixNothingLiesUnderIsEmpty(t *testing.T) {
	// The prefix a caller mistypes: the answer is no documents, not a panic.
	sub := lexicon.Build(graph()).Filter("nowhere")

	if sub.DocCount != 0 || len(sub.Docs) != 0 || sub.AvgBodyLen != 0 {
		t.Fatalf("got %+v, want an empty index", sub)
	}
}

func ids(ix *lexicon.Index) []string {
	out := make([]string, 0, len(ix.Docs))
	for _, d := range ix.Docs {
		out = append(out, string(d.ID))
	}
	return out
}

func TestFilterNormalizesTheShellCompletedPrefix(t *testing.T) {
	ix := lexicon.Build(graph())
	bare := ids(ix.Filter("lib"))

	// Tab completion hands a caller "lib/", and on Windows "lib\": the same
	// subtree, so the same documents.
	for _, prefix := range []string{"lib/", "/lib", "/lib/", `lib\`, `\lib\`} {
		if got := ids(ix.Filter(prefix)); !slices.Equal(got, bare) {
			t.Errorf("Filter(%q) = %v, want %v", prefix, got, bare)
		}
	}
}

func TestFilterNormalizesAWindowsShapedSubtree(t *testing.T) {
	g := graph()
	g.Nodes = append(g.Nodes, model.Node{
		ID: "lib/deep/inner.go#Inner", Name: "Inner", Kind: "function",
		Path: "lib/deep/inner.go", BodyHash: "h", BodyText: "x",
	})
	ix := lexicon.Build(g)

	if got, want := ids(ix.Filter(`lib\deep`)), ids(ix.Filter("lib/deep")); !slices.Equal(got, want) {
		t.Errorf(`Filter("lib\deep") = %v, want %v`, got, want)
	}
}

func TestFilterOnASlashAloneIsTheWholeIndex(t *testing.T) {
	ix := lexicon.Build(graph())

	// A prefix that normalizes away is no prefix at all.
	for _, prefix := range []string{"/", "//", `\`} {
		if got := ix.Filter(prefix); got.DocCount != ix.DocCount {
			t.Errorf("Filter(%q).DocCount = %d, want the whole index at %d", prefix, got.DocCount, ix.DocCount)
		}
	}
}

func TestFilterStaysSegmentAwareAfterNormalizing(t *testing.T) {
	g := graph()
	g.Nodes = append(g.Nodes, model.Node{
		ID: "libextra/other.go#Other", Name: "Other", Kind: "function",
		Path: "libextra/other.go", BodyHash: "h", BodyText: "x",
	})
	sub := lexicon.Build(g).Filter("lib/")

	for _, d := range sub.Docs {
		if d.ID == "libextra/other.go#Other" {
			t.Fatalf("normalizing a trailing slash made the filter segment-unaware: %q", d.ID)
		}
	}
	if len(sub.Docs) != 2 {
		t.Fatalf("docs = %v, want the two under lib/", ids(sub))
	}
}

func TestNormalizePrefix(t *testing.T) {
	cases := []struct{ in, want string }{
		{"lib", "lib"},
		{"lib/", "lib"},
		{"/lib", "lib"},
		{"/lib/", "lib"},
		{`lib\sub\`, "lib/sub"},
		{`\lib\sub`, "lib/sub"},
		{"", ""},
		{"/", ""},
		{"//", ""},
		{`\`, ""},
	}
	for _, c := range cases {
		if got := lexicon.NormalizePrefix(c.in); got != c.want {
			t.Errorf("NormalizePrefix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizePrefixIsIdempotent(t *testing.T) {
	// Filter normalizes, and so does every caller that walks the graph for the
	// same prefix. Both must be able to do it without changing the answer.
	for _, in := range []string{"lib", "lib/", `\lib\sub\`, "/"} {
		once := lexicon.NormalizePrefix(in)
		if twice := lexicon.NormalizePrefix(once); twice != once {
			t.Errorf("NormalizePrefix(%q) = %q, then %q", in, once, twice)
		}
	}
}

func TestWriteKeepsTheOldSidecarWhenTheReplacementCannotBeWritten(t *testing.T) {
	root := t.TempDir()
	if err := lexicon.Write(root, lexicon.Build(graph())); err != nil {
		t.Fatal(err)
	}
	// A directory where Write's private temporary file belongs: the new body
	// cannot be written. A reader that arrives now -- a run that lost the
	// rebuild lock -- must still find the whole old sidecar, not a truncated one.
	tmp := lexicon.Path(root) + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if err := os.Mkdir(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := lexicon.Write(root, lexicon.Build(&model.Graph{})); err == nil {
		t.Fatal("got nil, want an error")
	}
	ix, err := lexicon.Read(root)
	if err != nil {
		t.Fatalf("old sidecar unreadable after a failed write: %v", err)
	}
	if ix.DocCount != len(graph().Nodes) {
		t.Fatalf("DocCount = %d, want the old %d", ix.DocCount, len(graph().Nodes))
	}
}

func TestWriteReportsATargetItCannotReplace(t *testing.T) {
	root := t.TempDir()
	// A directory where the sidecar belongs: the rename cannot land, and the
	// temporary file must not be left behind.
	if err := os.MkdirAll(filepath.Join(lexicon.Path(root), "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := lexicon.Write(root, lexicon.Build(graph())); err == nil {
		t.Fatal("got nil, want an error")
	}
	tmp := lexicon.Path(root) + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if _, err := os.Stat(tmp); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("temporary file left behind: %v", err)
	}
}

func TestUsableAcceptsWhatWriteWrote(t *testing.T) {
	root := t.TempDir()
	if err := lexicon.Write(root, lexicon.Build(graph())); err != nil {
		t.Fatal(err)
	}
	if !lexicon.Usable(root) {
		t.Fatal("a sidecar this binary just wrote must be usable")
	}
}

func TestUsableRejectsWhatCannotBeRead(t *testing.T) {
	// The freshness probe asks this question, so every answer here decides
	// whether a query rebuilds. Anything but a readable sidecar of this
	// version is drift.
	for _, c := range []struct {
		name string
		body string
	}{
		{"an array", "[1]\n"},
		{"an object without a version", "{}\n"},
		{"another version", `{"version": 2}`},
		{"a version that is not a number", `{"version": "one"}`},
		{"a truncated value before the version", `{"df": `},
		{"a truncated key", `{"df": {}, "ver`},
		{"nothing at all", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Dir(lexicon.Path(root)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(lexicon.Path(root), []byte(c.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if lexicon.Usable(root) {
				t.Fatalf("%s must not count as a usable sidecar", c.name)
			}
		})
	}
}

func TestUsableRejectsASidecarThatIsNotThere(t *testing.T) {
	if lexicon.Usable(t.TempDir()) {
		t.Fatal("no file is no sidecar")
	}
}

func TestUsableReadsTheVersionBehindOtherFields(t *testing.T) {
	// The version is the first field Write emits, and the fast path depends on
	// that. A reordered struct must still be read rather than counted as
	// drift, or a rebuild would run on every question.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(lexicon.Path(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"avg_body_len": 1.5, "df": {"a": 1}, "docs": [], "version": 1}`
	if err := os.WriteFile(lexicon.Path(root), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if !lexicon.Usable(root) {
		t.Fatal("the version must be found wherever it stands")
	}
}
