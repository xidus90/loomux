package spanidx_test

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/code/spanidx"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/lock"
)

var digest = [32]byte{1, 2, 3}

// fixture extracts the recorded graph repository, which has nested symbols
// and calls between them.
func fixture(t *testing.T) *model.Graph {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join("..", "..", "..", "testdata", "cases", "graph", "repo")
	if err := os.CopyFS(root, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	g, _, err := query.Extract(root, query.ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// written writes the fixture's spans into a fresh root.
func written(t *testing.T) (string, *model.Graph) {
	t.Helper()
	g := fixture(t)
	root := t.TempDir()
	if err := spanidx.Write(root, digest, g); err != nil {
		t.Fatal(err)
	}
	return root, g
}

func open(t *testing.T, root string) *spanidx.Reader {
	t.Helper()
	r, err := spanidx.Open(root, digest)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func TestWriteOpenRoundTrip(t *testing.T) {
	root, g := written(t)
	r := open(t, root)
	x := blast.New(g)
	want := model.FileSpans(g)
	if len(want) == 0 {
		t.Fatal("the fixture has no symbols")
	}
	edges := 0
	for path, spans := range want {
		got := r.Spans(path)
		if len(got) != len(spans) {
			t.Fatalf("%s: %d spans, want %d", path, len(got), len(spans))
		}
		for i, s := range spans {
			n := *s.Node
			n.BodyText = ""
			if got[i].From != s.From || got[i].To != s.To || *got[i].Node != n {
				t.Fatalf("%s[%d] = %+v %+v, want %+v %+v", path, i, got[i], *got[i].Node, s, n)
			}
			if d := r.InDegree(n.ID); d != x.InDegree(n.ID) {
				t.Fatalf("InDegree(%s) = %d, want %d", n.ID, d, x.InDegree(n.ID))
			}
			edges += x.InDegree(n.ID)
		}
		// A second lookup is served from what the first decoded.
		if again := r.Spans(path); len(again) == 0 || &again[0] != &got[0] {
			t.Fatalf("%s decoded twice", path)
		}
	}
	if edges == 0 {
		t.Fatal("the fixture has no incoming edges, InDegree is untested")
	}
	if r.Spans("missing.go") != nil || r.InDegree("missing.go#X") != 0 || r.Broken() {
		t.Fatal("a path without spans must yield nothing and break nothing")
	}
}

func TestOpenRefusesAnotherDigest(t *testing.T) {
	root, _ := written(t)
	if _, err := spanidx.Open(root, [32]byte{9}); !errors.Is(err, spanidx.ErrMismatch) {
		t.Fatalf("Open = %v, want ErrMismatch", err)
	}
}

func TestOpenRefusesAnotherVersion(t *testing.T) {
	root, _ := written(t)
	patch(t, root, func(b []byte) []byte { b[4] = 2; return b })
	if _, err := spanidx.Open(root, digest); !errors.Is(err, spanidx.ErrMismatch) {
		t.Fatalf("Open = %v, want ErrMismatch", err)
	}
}

func TestOpenRefusesAnotherMagic(t *testing.T) {
	root, _ := written(t)
	patch(t, root, func(b []byte) []byte { b[0] = 'X'; return b })
	if _, err := spanidx.Open(root, digest); !errors.Is(err, spanidx.ErrCorrupt) {
		t.Fatalf("Open = %v, want ErrCorrupt", err)
	}
}

func TestOpenRefusesTruncatedFile(t *testing.T) {
	root, _ := written(t)
	path := store.CachePath(root, spanidx.Name)
	whole, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for cut := 0; cut < len(whole); cut++ {
		if err := os.WriteFile(path, whole[:cut], 0o644); err != nil {
			t.Fatal(err)
		}
		r, err := spanidx.Open(root, digest)
		if err == nil {
			r.Close()
			t.Fatalf("Open of %d of %d bytes succeeded", cut, len(whole))
		}
	}
}

// An entry offset beyond the end is caught before any lookup follows it.
func TestOpenRefusesOffsetOutOfRange(t *testing.T) {
	root, _ := written(t)
	patch(t, root, func(b []byte) []byte {
		copy(b[44:48], []byte{0xff, 0xff, 0xff, 0x7f})
		return b
	})
	if _, err := spanidx.Open(root, digest); !errors.Is(err, spanidx.ErrCorrupt) {
		t.Fatalf("Open = %v, want ErrCorrupt", err)
	}
}

func TestOpenMissing(t *testing.T) {
	if _, err := spanidx.Open(t.TempDir(), digest); !errors.Is(err, spanidx.ErrNoSpans) {
		t.Fatalf("Open = %v, want ErrNoSpans", err)
	}
}

func TestOpenUnreadable(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(store.CachePath(root, spanidx.Name), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := spanidx.Open(root, digest)
	if err == nil || errors.Is(err, spanidx.ErrNoSpans) || errors.Is(err, spanidx.ErrCorrupt) {
		t.Fatalf("Open = %v, want a read error", err)
	}
}

// A block that does not decode costs that file its symbols, and says so.
func TestSpansOfABrokenBlock(t *testing.T) {
	root, g := written(t)
	last := ""
	for path := range model.FileSpans(g) {
		last = max(last, path)
	}
	// The last entry's block ends the file; its closing bracket goes.
	patch(t, root, func(b []byte) []byte { b[len(b)-1] = 'x'; return b })
	r := open(t, root)
	if r.Spans(last) != nil || !r.Broken() {
		t.Fatalf("Spans(%s) of a broken block: want nil and Broken", last)
	}
}

func TestWriteEmptyGraph(t *testing.T) {
	root := t.TempDir()
	if err := spanidx.Write(root, digest, &model.Graph{}); err != nil {
		t.Fatal(err)
	}
	r := open(t, root)
	if r.Spans("a.go") != nil {
		t.Fatal("an empty index has spans")
	}
}

func TestWriteRefusesALongPath(t *testing.T) {
	long := strings.Repeat("a", 1<<16) + ".go"
	g := &model.Graph{Nodes: []model.Node{{ID: model.NodeID(long + "#A"), Path: long, Span: "L1-L2"}}}
	if err := spanidx.Write(t.TempDir(), digest, g); err == nil {
		t.Fatal("Write of a path longer than u16 succeeded")
	}
}

func TestWriteErrors(t *testing.T) {
	old := lock.SidecarBudget
	lock.SidecarBudget = 50 * time.Millisecond
	t.Cleanup(func() { lock.SidecarBudget = old })
	g := &model.Graph{}
	final := func(root string) string { return store.CachePath(root, spanidx.Name) }
	for name, block := range map[string]func(root string) string{
		// A file where the cache directory belongs.
		"directory": func(root string) string { return filepath.Dir(final(root)) },
		// A directory where the temporary file belongs.
		"temporary": func(root string) string { return final(root) + "." + strconv.Itoa(os.Getpid()) + ".tmp/x" },
		// A directory where the index belongs.
		"target": func(root string) string { return final(root) + "/x" },
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			p := block(root)
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := spanidx.Write(root, digest, g); err == nil {
				t.Fatal("Write succeeded")
			}
		})
	}
}

// patch rewrites the written index through f.
func patch(t *testing.T, root string, f func([]byte) []byte) {
	t.Helper()
	path := store.CachePath(root, spanidx.Name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, f(b), 0o644); err != nil {
		t.Fatal(err)
	}
}
