package query

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/store"
)

// A record another extractor wrote is rebuilt once and the rebuilt graph
// carries the combined stamp; the next load trusts it and says nothing.
func TestLoadGraphRebuildsARecordAnotherExtractorWroteOnce(t *testing.T) {
	root := repo(t, sample())
	_, stats, err := Build(root, ignore)
	if err != nil {
		t.Fatal(err)
	}
	if err := freshness.Write(root, "go/0", stats.Files, stats.Hashes); err != nil {
		t.Fatal(err)
	}
	g, notes, err := loadGraph(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) == 0 || !strings.Contains(notes[0], "building the graph") {
		t.Errorf("notes = %v, want a rebuild for a record another extractor wrote", notes)
	}
	if g.Meta.Extractor != all.Version() {
		t.Errorf("Meta.Extractor = %q, want %q", g.Meta.Extractor, all.Version())
	}
	if _, notes, err = loadGraph(root, false); err != nil || len(notes) != 0 {
		t.Errorf("second load: notes = %v, err = %v; want neither", notes, err)
	}
}

func TestLoadGraphErrors(t *testing.T) {
	root := repo(t, sample())
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.WiringPath(root), []byte("{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := loadGraph(root, true)
	if err == nil || errors.Is(err, ErrNoGraph) {
		t.Fatalf("want decode error, got %v", err)
	}
}
