package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/testlock"
)

func graph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 2, Extractor: "go/1", Languages: []string{"go"}, NodeCount: 2, EdgeCount: 1},
		Nodes: []model.Node{
			{ID: "a.go", Kind: model.KindFile, Path: "a.go", BodyHash: "h1", Exported: true, BodyText: "residual text"},
			{ID: "a.go#F", Kind: "function", Path: "a.go", Name: "F", BodyHash: "h2", BodyText: "body text"},
		},
		Edges: []model.Edge{{
			Source: "a.go", Target: "a.go#F",
			Relation: model.RelationContains, Confidence: model.ConfidenceExtracted,
		}},
	}
}

func TestWriteThenReadRoundTrips(t *testing.T) {
	root := t.TempDir()
	if err := store.Write(root, graph()); err != nil {
		t.Fatal(err)
	}
	got, err := store.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Nodes) != 2 || got.Meta.Extractor != "go/1" {
		t.Fatalf("got %+v", got.Meta)
	}
}

func TestWriteIsByteIdenticalOnAnUnchangedGraph(t *testing.T) {
	root := t.TempDir()
	if err := store.Write(root, graph()); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(store.WiringPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Write(root, graph()); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(store.WiringPath(root))
	if err != nil {
		t.Fatal(err)
	}
	// No timestamps anywhere: a rebuild of an unchanged tree must produce the
	// same bytes, or every build would show up as a diff.
	if string(first) != string(second) {
		t.Error("two writes of one graph produced different bytes")
	}
}

func TestWriteLeavesBodyTextOnTheFloor(t *testing.T) {
	root := t.TempDir()
	if err := store.Write(root, graph()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(store.WiringPath(root))
	if err != nil {
		t.Fatal(err)
	}
	// It is most of the file's bytes on a real repository, and the ask sidecar
	// holds it tokenized. The model's json:"-" is what enforces this; the test
	// is here so a future field rename cannot quietly undo it.
	if strings.Contains(string(b), "body text") || strings.Contains(string(b), "residual text") {
		t.Errorf("body text reached wiring.json:\n%s", b)
	}
}

func TestWriteLeavesNoTemporaryFileBehind(t *testing.T) {
	root := t.TempDir()
	if err := store.Write(root, graph()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(store.Dir(root))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("temporary file left behind: %s", e.Name())
		}
	}
}

func TestReadReportsAMissingGraphAsNotExist(t *testing.T) {
	_, err := store.Read(t.TempDir())
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v, want a wrapped os.ErrNotExist so a caller can tell absent from broken", err)
	}
}

func TestReadRefusesABrokenGraph(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.WiringPath(root), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := store.Read(root)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v, want an error that is not ErrNotExist: broken is not absent", err)
	}
}

func TestWriteFailsWhenTheTempFileCannotBeWritten(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	testlock.LockDir(t, store.Dir(root))
	if err := store.Write(root, graph()); err == nil {
		t.Fatal("want an error when the temp file cannot be written")
	}
}

func TestWriteFailsWhenTheRenameCannotReplaceTheFinalPath(t *testing.T) {
	root := t.TempDir()
	// The final path is occupied by a directory instead of a file: a rename
	// onto it fails the way it would if something else raced to create one.
	if err := os.MkdirAll(store.WiringPath(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.Write(root, graph()); err == nil {
		t.Fatal("want an error when rename cannot replace the final path")
	}
	entries, err := os.ReadDir(store.Dir(root))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("temporary file left behind after a failed rename: %s", e.Name())
		}
	}
}

func TestPathsSitUnderTheStateDirectory(t *testing.T) {
	root := filepath.FromSlash("/repo")
	if got := store.WiringPath(root); !strings.HasSuffix(filepath.ToSlash(got), ".loomux/state/graph/wiring.json") {
		t.Errorf("WiringPath = %q", got)
	}
	if got := store.CachePath(root, "fingerprint.json"); !strings.HasSuffix(filepath.ToSlash(got), ".loomux/state/graph/cache/fingerprint.json") {
		t.Errorf("CachePath = %q", got)
	}
}
