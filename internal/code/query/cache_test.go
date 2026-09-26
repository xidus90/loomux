package query

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
)

// graphCase copies the recorded graph repository into a fresh root.
func graphCase(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join("..", "..", "..", "testdata", "cases", "graph", "repo")
	if err := os.CopyFS(root, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	return root
}

// mustBuild runs one build with the given options and fails the test on an
// error.
func mustBuild(t *testing.T, root string, opts BuildOptions, notice func(string)) (*model.Graph, Stats) {
	t.Helper()
	g, stats, err := BuildWith(root, opts, notice)
	if err != nil {
		t.Fatal(err)
	}
	return g, stats
}

func readBytes(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// rawCache reads extract.json as written, bodies still in their own field.
func rawCache(t *testing.T, root string) cacheFile {
	t.Helper()
	var f cacheFile
	if err := json.Unmarshal(readBytes(t, cachePath(root)), &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestWarmBuildEqualsNoReuseBuild(t *testing.T) {
	root := graphCase(t)
	mustBuild(t, root, BuildOptions{}, ignore)
	_, warm := mustBuild(t, root, BuildOptions{}, ignore)
	warmWiring, warmAsk := readBytes(t, store.WiringPath(root)), readBytes(t, lexicon.Path(root))

	_, fresh := mustBuild(t, root, BuildOptions{NoReuse: true}, ignore)
	if !bytes.Equal(warmWiring, readBytes(t, store.WiringPath(root))) {
		t.Error("wiring.json from reused extractions differs from a build that parsed every file")
	}
	if !bytes.Equal(warmAsk, readBytes(t, lexicon.Path(root))) {
		t.Error("ask-index.json from reused extractions differs from a build that parsed every file")
	}

	files := len(warm.Files)
	if got := warm.PerLanguage["go"]; got.Files != files || got.Parsed != 0 || got.Reused != files {
		t.Errorf("warm build go stats = %+v, want all %d files reused and none parsed", got, files)
	}
	if got := fresh.PerLanguage["go"]; got.Files != files || got.Parsed != files || got.Reused != 0 {
		t.Errorf("--no-reuse build go stats = %+v, want all %d files parsed and none reused", got, files)
	}
}

func TestCacheKeepsBodyText(t *testing.T) {
	root := graphCase(t)
	mustBuild(t, root, BuildOptions{}, ignore)
	cold := readBytes(t, lexicon.Path(root))

	// Twice warm: the second reuses what the first wrote back from reused
	// entries, so a body lost on the way back to disk shows up there.
	for i := 0; i < 2; i++ {
		mustBuild(t, root, BuildOptions{}, ignore)
		if !bytes.Equal(cold, readBytes(t, lexicon.Path(root))) {
			t.Fatalf("warm build %d wrote another ask index than the cold one", i+1)
		}
		bodies := rawCache(t, root).Files["calc/calc.go"].Bodies
		if !strings.Contains(bodies["calc/calc.go#Add"], "return a + b") {
			t.Fatalf("after warm build %d the cache holds bodies %q, want Add's", i+1, bodies)
		}
	}

	// "return" stands in Add's body and nowhere in its name, path or
	// signature: only the body text can put it into the index.
	ix, err := lexicon.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range ix.Docs {
		if d.ID == "calc/calc.go#Add" && d.Body["return"] == 0 {
			t.Errorf("Add's body tokens %v lack a word of its body after warm builds", d.Body)
		}
	}
}

func TestChangedFileIsReparsed(t *testing.T) {
	root := graphCase(t)
	mustBuild(t, root, BuildOptions{}, ignore)

	calc := filepath.Join(root, "calc", "calc.go")
	body := string(readBytes(t, calc)) + "\n// Mul multiplies.\nfunc Mul(a, b int) int {\n\treturn a * b\n}\n"
	if err := os.WriteFile(calc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	g, stats := mustBuild(t, root, BuildOptions{}, ignore)

	if got := stats.PerLanguage["go"]; got.Parsed != 1 || got.Reused != got.Files-1 {
		t.Errorf("go stats = %+v, want the one changed file parsed and the rest reused", got)
	}
	found := false
	for _, n := range g.Nodes {
		found = found || n.ID == "calc/calc.go#Mul"
	}
	if !found {
		t.Error("the function added to a changed file is not in the graph")
	}
}

func TestGarbageCacheRebuildsWithNotice(t *testing.T) {
	for _, garbage := range []string{"not json", `{"version": 99, "files": {}}`} {
		t.Run(garbage, func(t *testing.T) {
			root := graphCase(t)
			mustBuild(t, root, BuildOptions{}, ignore)
			if err := os.WriteFile(cachePath(root), []byte(garbage), 0o644); err != nil {
				t.Fatal(err)
			}

			var heard []string
			_, stats := mustBuild(t, root, BuildOptions{}, func(s string) { heard = append(heard, s) })
			if got := stats.PerLanguage["go"]; got.Parsed != got.Files || got.Reused != 0 {
				t.Errorf("go stats = %+v, want every file parsed", got)
			}
			if len(heard) != 1 || !strings.Contains(heard[0], "cache") {
				t.Errorf("notices = %q, want one about the cache", heard)
			}

			// The build wrote a sound cache over the garbage.
			_, again := mustBuild(t, root, BuildOptions{}, ignore)
			if got := again.PerLanguage["go"]; got.Parsed != 0 {
				t.Errorf("the build after it parsed %d files, want every file reused", got.Parsed)
			}
		})
	}
}

func TestForeignLanguageVersionIsReparsed(t *testing.T) {
	root := graphCase(t)
	mustBuild(t, root, BuildOptions{}, ignore)
	entries, err := readCache(root)
	if err != nil {
		t.Fatal(err)
	}
	e := entries["calc/calc.go"]
	e.Extractor = "go/0"
	entries["calc/calc.go"] = e
	if err := writeCache(root, entries); err != nil {
		t.Fatal(err)
	}

	_, stats := mustBuild(t, root, BuildOptions{}, ignore)
	if got := stats.PerLanguage["go"]; got.Parsed != 1 || got.Reused != got.Files-1 {
		t.Errorf("go stats = %+v, want only the file of the other extractor version parsed", got)
	}
	lang, _ := all.For("calc/calc.go")
	if got := rawCache(t, root).Files["calc/calc.go"].Extractor; got != lang.Version() {
		t.Errorf("the rewritten entry says extractor %q, want %q", got, lang.Version())
	}
}

func TestDeletedFileLeavesCache(t *testing.T) {
	root := graphCase(t)
	mustBuild(t, root, BuildOptions{}, ignore)
	if _, ok := rawCache(t, root).Files["calc/calc_test.go"]; !ok {
		t.Fatal("the first build did not cache calc/calc_test.go")
	}
	if err := os.Remove(filepath.Join(root, "calc", "calc_test.go")); err != nil {
		t.Fatal(err)
	}
	mustBuild(t, root, BuildOptions{}, ignore)
	files := rawCache(t, root).Files
	if _, ok := files["calc/calc_test.go"]; ok {
		t.Error("the cache still holds the deleted file")
	}
	if len(files) != 2 {
		t.Errorf("the cache holds %d files, want the 2 left", len(files))
	}
}

func TestUnwritableCacheIsANotice(t *testing.T) {
	root := graphCase(t)
	// A directory in the cache's place fails the read and the rename; the
	// graph and the sidecars are written before either matters.
	if err := os.MkdirAll(cachePath(root), 0o755); err != nil {
		t.Fatal(err)
	}
	var heard []string
	g, _ := mustBuild(t, root, BuildOptions{}, func(s string) { heard = append(heard, s) })
	if g == nil {
		t.Fatal("the build must stand without a cache")
	}
	if !strings.Contains(strings.Join(heard, "\n"), "extract cache not written") {
		t.Errorf("notices = %q, want one about the unwritten cache", heard)
	}
	if _, err := os.Stat(store.WiringPath(root)); err != nil {
		t.Errorf("the graph must be on disk: %v", err)
	}
	if _, err := os.Stat(cachePath(root) + "." + strconv.Itoa(os.Getpid()) + ".tmp"); err == nil {
		t.Error("the failed rename left its temp file behind")
	}
}

func TestReadCache(t *testing.T) {
	root := t.TempDir()
	if entries, err := readCache(root); entries != nil || err != nil {
		t.Errorf("readCache without a file = %v, %v; want nothing and no error", entries, err)
	}
	if err := os.MkdirAll(cachePath(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := readCache(root); err == nil {
		t.Error("readCache of a directory must fail")
	}
}

func TestWriteCacheFailsWhereItCannotWrite(t *testing.T) {
	entries := map[string]cacheEntry{"a.go": {Extractor: "go/1", SHA256: "00"}}

	// A regular file where the cache directory belongs.
	root := t.TempDir()
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Dir(root), "cache"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeCache(root, entries); err == nil {
		t.Error("writeCache under a file named cache must fail")
	}

	// A directory where the temp file belongs.
	root = t.TempDir()
	tmp := cachePath(root) + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeCache(root, entries); err == nil {
		t.Error("writeCache onto a directory in the temp file's place must fail")
	}
	if _, err := os.Stat(cachePath(root)); err == nil {
		t.Error("a failed write must leave no cache behind")
	}
}

func TestWriteCacheRoundTripsBodies(t *testing.T) {
	root := t.TempDir()
	r := extract.Result{Path: "a.go", Language: "go", Nodes: []model.Node{
		{ID: "a.go", Path: "a.go", BodyText: "package a"},
		{ID: "a.go#F", Path: "a.go", BodyText: "func F() { spin() }"},
		{ID: "a.go#T", Path: "a.go"},
	}}
	in := map[string]cacheEntry{"a.go": {Extractor: "go/1", SHA256: "ab", Result: r}}
	if err := writeCache(root, in); err != nil {
		t.Fatal(err)
	}
	// Only the non-empty bodies are written, and none twice: the node's own
	// field stays off the wire.
	raw := rawCache(t, root).Files["a.go"]
	if len(raw.Bodies) != 2 || raw.Bodies["a.go#T"] != "" {
		t.Errorf("written bodies = %q, want the two non-empty ones", raw.Bodies)
	}
	if strings.Count(string(readBytes(t, cachePath(root))), "spin()") != 1 {
		t.Error("a body must be written once, in bodies")
	}

	out, err := readCache(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := out["a.go"]; got.Bodies != nil || !reflect.DeepEqual(got.Result, r) {
		t.Errorf("read back %+v, want %+v with the bodies in the nodes", got, r)
	}
}

// tolerant is a language whose parser skips what it cannot read and counts
// it, which the Go extractor never does: go/parser refuses the whole file.
type tolerant struct{}

func (tolerant) Name() string         { return "tolerant" }
func (tolerant) Version() string      { return "tolerant/1" }
func (tolerant) Extensions() []string { return []string{".go"} }
func (tolerant) File(rel, source string) (extract.Result, error) {
	r := extract.Result{Path: rel, Language: "tolerant", Nodes: []model.Node{
		{ID: model.NodeID(rel), Name: path.Base(rel), Kind: model.KindFile, Path: rel},
	}}
	r.ParseErrors = strings.Count(source, "???")
	return r, nil
}

func TestParseErrorsAreCountedForParsedAndReusedFiles(t *testing.T) {
	claims := languageFor
	t.Cleanup(func() { languageFor = claims })
	languageFor = func(string) (extract.Language, bool) { return tolerant{}, true }

	root := repo(t, map[string]string{
		"go.mod":   "module x\n",
		"b.go":     "??? ???\n",
		"a.go":     "???\n",
		"clean.go": "fine\n",
	})
	want := LangStats{Files: 3, ParseErrors: 3, ErrorFiles: []string{"a.go", "b.go"}}

	_, cold := mustBuild(t, root, BuildOptions{}, ignore)
	want.Parsed = 3
	if got := cold.PerLanguage["tolerant"]; !reflect.DeepEqual(got, want) {
		t.Errorf("cold stats = %+v, want %+v", got, want)
	}

	// A reused file still carries its parse errors: the graph lacks the same
	// parts it lacked when the file was parsed.
	_, warm := mustBuild(t, root, BuildOptions{}, ignore)
	want.Parsed, want.Reused = 0, 3
	if got := warm.PerLanguage["tolerant"]; !reflect.DeepEqual(got, want) {
		t.Errorf("warm stats = %+v, want %+v", got, want)
	}
}

func TestCheckReadsTheCacheAndNeverWritesIt(t *testing.T) {
	root := graphCase(t)
	mustBuild(t, root, BuildOptions{}, ignore)

	calc := filepath.Join(root, "calc", "calc.go")
	body := string(readBytes(t, calc)) + "\nfunc Mul(a, b int) int { return a * b }\n"
	if err := os.WriteFile(calc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	before := readBytes(t, cachePath(root))
	d, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Added) != 1 || d.Added[0] != "calc/calc.go#Mul" {
		t.Errorf("Added = %v, want the new function of the changed file", d.Added)
	}
	if !bytes.Equal(before, readBytes(t, cachePath(root))) {
		t.Error("check rewrote the extract cache")
	}

	// A cache check cannot read costs it the reuse and nothing else; it has
	// no notice to give.
	if err := os.WriteFile(cachePath(root), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if d, err := Check(root); err != nil || len(d.Added) != 1 {
		t.Errorf("Check over a garbage cache = %+v, %v; want the same drift", d, err)
	}
}
