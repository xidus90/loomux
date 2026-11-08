package query

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/grep"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/spanidx"
)

// trustPatterns are the four benchmark patterns, fitted to the fixture: a
// literal, an alternation, a case-folded literal and one without a literal.
var trustPatterns = []struct {
	name    string
	pattern string
	opts    GrepOptions
}{
	{"literal", "func Run", GrepOptions{}},
	{"alternation", "Run|zzzUniqueWord", GrepOptions{}},
	{"ignore case", "FUNC RUN", GrepOptions{IgnoreCase: true}},
	{"no literal", `[A-Z][a-z]+[A-Z]\w*`, GrepOptions{}},
}

// asJSON is the answer as the CLI prints it, minus the two fields that say
// how it was reached.
func asJSON(t *testing.T, r grep.Result) string {
	t.Helper()
	r.Index, r.IndexBuilt = "", ""
	body, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func trusting(opts GrepOptions) GrepOptions {
	opts.TrustIndex = true
	return opts
}

func TestTrustIndexMatchesCheckedJSON(t *testing.T) {
	root := grepFixture(t)
	for _, c := range trustPatterns {
		t.Run(c.name, func(t *testing.T) {
			checked, _, err := Grep(root, c.pattern, c.opts)
			if err != nil {
				t.Fatal(err)
			}
			trusted, notes, err := Grep(root, c.pattern, trusting(c.opts))
			if err != nil {
				t.Fatal(err)
			}
			if len(notes) != 0 {
				t.Fatalf("notes = %v, want none", notes)
			}
			if trusted.Index != "trusted" {
				t.Fatalf("index = %q, want trusted", trusted.Index)
			}
			if _, err := time.Parse(time.RFC3339, trusted.IndexBuilt); err != nil {
				t.Fatalf("index_built = %q: %v", trusted.IndexBuilt, err)
			}
			if checked.IndexBuilt != "" {
				t.Fatalf("the checked path set index_built = %q", checked.IndexBuilt)
			}
			if len(checked.Groups) == 0 {
				t.Fatal("the pattern found nothing, so the comparison proves little")
			}
			if a, b := asJSON(t, trusted), asJSON(t, checked); a != b {
				t.Fatalf("trusted:\n%s\nchecked:\n%s", a, b)
			}
		})
	}
}

func TestTrustIndexDoesNotWalk(t *testing.T) {
	root := grepFixture(t)
	walks, reads := 0, 0
	oldWalk, oldRead := walkCorpus, readRecord
	walkCorpus = func(r string, m corpus.Matcher) (corpus.Listing, error) {
		walks++
		return oldWalk(r, m)
	}
	readRecord = func(r string) (corpus.Record, bool) {
		reads++
		return oldRead(r)
	}
	t.Cleanup(func() { walkCorpus, readRecord = oldWalk, oldRead })

	res, _, err := Grep(root, "func Run", GrepOptions{TrustIndex: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "trusted" || res.TotalHits == 0 {
		t.Fatalf("index = %q hits = %d, want a trusted answer", res.Index, res.TotalHits)
	}
	if walks != 0 || reads != 0 {
		t.Fatalf("walks = %d, record reads = %d, want none", walks, reads)
	}
}

// The trusted answer is the index's: a file the index never saw stays out,
// and a candidate is read as it is on disk now.
func TestTrustIndexAnswersFromTheIndexOnAChangedTree(t *testing.T) {
	root := grepFixture(t)
	writeTree(t, root, map[string]string{
		"fresh.txt": "zzzUniqueWord\n",
		"README.md": "a\nb\nc\nzzzUniqueWord now\n",
	})
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{TrustIndex: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "trusted" {
		t.Fatalf("index = %q, want trusted", res.Index)
	}
	if got, want := resultHits(res), []string{"README.md:4", "big.txt:2"}; !slices.Equal(got, want) {
		t.Fatalf("hits = %v, want %v", got, want)
	}
}

func TestTrustIndexFallsBack(t *testing.T) {
	cases := []struct {
		name   string
		opts   GrepOptions
		damage func(t *testing.T, root string)
	}{
		{"index missing", GrepOptions{}, func(t *testing.T, root string) {
			if err := os.Remove(indexPath(root)); err != nil {
				t.Fatal(err)
			}
		}},
		{"index broken", GrepOptions{}, func(t *testing.T, root string) {
			raw, err := os.ReadFile(indexPath(root))
			if err != nil {
				t.Fatal(err)
			}
			raw[0] = 'X'
			if err := os.WriteFile(indexPath(root), raw, 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"foreign extractor", GrepOptions{}, func(t *testing.T, root string) {
			raw, err := os.ReadFile(indexPath(root))
			if err != nil {
				t.Fatal(err)
			}
			// magic, format version and the u16 length come first.
			raw[10] ^= 1
			if err := os.WriteFile(indexPath(root), raw, 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"foreign sidecar", GrepOptions{}, func(t *testing.T, root string) {
			if err := spanidx.Write(root, [32]byte{1}, &model.Graph{}); err != nil {
				t.Fatal(err)
			}
		}},
		{"settings differ", GrepOptions{MaxFilesize: limitOf(1024)}, func(*testing.T, string) {}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := grepFixture(t)
			c.damage(t, root)
			res, notes, err := Grep(root, "zzzUniqueWord", trusting(c.opts))
			if err != nil {
				t.Fatal(err)
			}
			if !slices.ContainsFunc(notes, func(n string) bool { return strings.HasPrefix(n, "index not trusted: ") }) {
				t.Fatalf("notes = %v, want an %q note", notes, "index not trusted: ")
			}
			if res.Index == "trusted" || res.IndexBuilt != "" {
				t.Fatalf("index = %q built = %q, want the checked path", res.Index, res.IndexBuilt)
			}
			if got, want := resultHits(res), oracleHits(t, root, "zzzUniqueWord"); !slices.Equal(got, want) {
				t.Fatalf("hits = %v, want %v", got, want)
			}
			if res.TotalHits == 0 {
				t.Fatal("the checked path found nothing")
			}
		})
	}
}

func TestTrustIndexFallsBackWhenTheIndexCannotBeStatted(t *testing.T) {
	root := grepFixture(t)
	old := statIndex
	statIndex = func(string) (os.FileInfo, error) { return nil, errors.New("stat broken") }
	t.Cleanup(func() { statIndex = old })

	res, notes, err := Grep(root, "zzzUniqueWord", GrepOptions{TrustIndex: true})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(notes, "index not trusted: stat broken") {
		t.Fatalf("notes = %v", notes)
	}
	if res.Index == "trusted" || res.TotalHits == 0 {
		t.Fatalf("index = %q hits = %d, want the checked answer", res.Index, res.TotalHits)
	}
}

func TestTrustIndexWithoutGraph(t *testing.T) {
	_, _, err := Grep(t.TempDir(), "Run", GrepOptions{TrustIndex: true})
	if !errors.Is(err, ErrNoGraph) {
		t.Fatalf("want ErrNoGraph, got %v", err)
	}
}

func TestTrustIndexReportsTheSearchError(t *testing.T) {
	root := grepFixture(t)
	if _, _, err := Grep(root, "(", GrepOptions{TrustIndex: true}); err == nil {
		t.Fatal("want an error for an invalid pattern")
	}
}

func TestTrustIndexWithGlobPrefixAndChannel(t *testing.T) {
	root := privacyRepo(t, `"secrets/**"`)
	cases := []struct {
		name string
		opts GrepOptions
		want []string
	}{
		{"glob", GrepOptions{Glob: "**/*.md"}, []string{"README.md:3", "docs/secret.md:1"}},
		{"in", GrepOptions{In: "secrets"}, []string{"secrets/key.txt:1"}},
		{"cloud", GrepOptions{Channel: "cloud"}, []string{"README.md:3", "big.txt:2", "docs/secret.md:1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, _, err := Grep(root, "zzzUniqueWord", trusting(c.opts))
			if err != nil {
				t.Fatal(err)
			}
			if res.Index != "trusted" {
				t.Fatalf("index = %q, want trusted", res.Index)
			}
			if got := resultHits(res); !slices.Equal(got, c.want) {
				t.Fatalf("hits = %v, want %v", got, c.want)
			}
		})
	}
	// Nothing under the prefix is readable: an empty answer, not an error.
	res, notes, err := Grep(root, "zzzUniqueWord", trusting(GrepOptions{Channel: "cloud", In: "secrets"}))
	if err != nil || res.TotalHits != 0 || len(notes) != 1 {
		t.Fatalf("hits = %d notes = %v err = %v, want an empty answer with one note", res.TotalHits, notes, err)
	}
}

func TestTrustIndexRejectsAnUnknownChannel(t *testing.T) {
	root := grepFixture(t)
	if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{TrustIndex: true, Channel: "bogus"}); err == nil {
		t.Fatal("an unknown channel must fail")
	}
}

func limitOf(n int64) *int64 { return &n }

// A literal too short for a trigram leaves the index nothing to narrow by, so
// the trusted answer scans the whole table and still says trusted.
func TestTrustIndexScansTheTableForAShortLiteral(t *testing.T) {
	root := grepFixture(t)
	res, _, err := Grep(root, "zz|ab", GrepOptions{TrustIndex: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "trusted" || res.Candidates != res.FilesInScope {
		t.Fatalf("index = %q candidates = %d of %d, want a scan of the whole scope", res.Index, res.Candidates, res.FilesInScope)
	}
	if got, want := resultHits(res), oracleHits(t, root, "zz|ab"); !slices.Equal(got, want) {
		t.Fatalf("hits = %v, want %v", got, want)
	}
}

// The index's file table holds no binary file, so a trusted scope cannot say
// which binaries lie under --in or match --glob: it counts all of them. The
// hits stay equal to the checked answer; only the two scope counts differ,
// by the binaries outside the filter. A prefix that holds nothing but a
// binary is not in the table at all.
func TestTrustIndexCountsEveryBinaryUnderAFilter(t *testing.T) {
	root := grepFixture(t)
	cases := []struct {
		name    string
		opts    GrepOptions
		checked int // FilesInScope of the checked answer
		trusted int // FilesInScope of the trusted answer
	}{
		{"whole corpus", GrepOptions{}, 7, 7},
		{"prefix", GrepOptions{In: "lib"}, 2, 3},
		{"glob", GrepOptions{Glob: "**/*.md"}, 1, 2},
		{"glob on the binary", GrepOptions{Glob: "**/*.bin"}, 1, 1},
		{"prefix and glob", GrepOptions{In: "lib", Glob: "**/*.bin"}, 0, 1},
	}
	// No literal condition: Candidates is the whole scope, so it moves with it.
	const pattern = `[A-Z]{3}\w+`
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checked, _, err := Grep(root, pattern, c.opts)
			if err != nil {
				t.Fatal(err)
			}
			trusted, _, err := Grep(root, pattern, trusting(c.opts))
			if err != nil {
				t.Fatal(err)
			}
			if checked.FilesInScope != c.checked || trusted.FilesInScope != c.trusted ||
				checked.Candidates != c.checked || trusted.Candidates != c.trusted {
				t.Fatalf("scope checked/trusted = %d/%d candidates = %d/%d, want %d/%d",
					checked.FilesInScope, trusted.FilesInScope, checked.Candidates, trusted.Candidates, c.checked, c.trusted)
			}
			checked.FilesInScope, checked.Candidates = trusted.FilesInScope, trusted.Candidates
			if a, b := asJSON(t, trusted), asJSON(t, checked); a != b {
				t.Fatalf("trusted:\n%s\nchecked:\n%s", a, b)
			}
		})
	}

	if _, _, err := Grep(root, pattern, GrepOptions{In: "blob.bin"}); err != nil {
		t.Fatalf("checked: %v", err)
	}
	if _, _, err := Grep(root, pattern, trusting(GrepOptions{In: "blob.bin"})); err == nil ||
		!strings.Contains(err.Error(), "prefix not indexed") {
		t.Fatalf("trusted err = %v, want prefix not indexed", err)
	}
}
