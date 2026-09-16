package graph_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/config"
)

func TestParseGraph_RootThatIsNotAnObjectMissesBothKeys(t *testing.T) {
	path := "/path/to/graph.json"
	want := path + ": graph is missing edges, links; delete it and run `brain reindex`"
	for _, data := range []string{`[]`, `"x"`, `3`, `null`, `"edges"`} {
		_, err := graph.ParseGraph([]byte(data), path)
		if err == nil || err.Error() != want {
			t.Errorf("%s: got %v, want %q", data, err, want)
		}
	}
}

func TestParseGraph_CountsAreIntegersAsJSONLoadsGivesThem(t *testing.T) {
	path := "/path/to/graph.json"
	want := path + ": the link counts are not numbers; delete it and run `brain reindex`"
	for _, links := range []string{
		`{"total": 3.0, "resolved": 0, "dropped": {}}`,
		`{"total": 0, "resolved": 1.5, "dropped": {}}`,
		`{"total": 1e2, "resolved": 0, "dropped": {}}`,
		`{"total": 2E1, "resolved": 0, "dropped": {}}`,
		`{"total": 0, "resolved": 0, "dropped": {"external": 2.0}}`,
		`{"total": true, "resolved": 0, "dropped": {}}`,
	} {
		_, err := graph.ParseGraph([]byte(`{"edges": [], "links": `+links+`}`), path)
		if err == nil || err.Error() != want {
			t.Errorf("%s: got %v, want %q", links, err, want)
		}
	}
	g, err := graph.ParseGraph([]byte(`{"edges": [], "links": {"total": 3, "resolved": -1, "dropped": {"external": 0}}}`), path)
	if err != nil {
		t.Fatal(err)
	}
	if g.Links.Total != 3 || g.Links.Resolved != -1 || g.Links.Dropped["external"] != 0 {
		t.Fatalf("links not parsed: %+v", g.Links)
	}
}

func TestParseGraph_InvalidJSONNamesTheDecoderError(t *testing.T) {
	path := "/path/to/graph.json"
	valid := `{"edges": [], "links": {"total": 0, "resolved": 0, "dropped": {}}}`
	for _, tc := range []struct {
		data   string
		reason string
	}{
		{"not json", "invalid character 'o' in literal null (expecting 'u')"},
		{"", "unexpected end of JSON input"},
		{valid + " x", "invalid character 'x' after top-level value"},
		{valid + " {}", "invalid character '{' after top-level value"},
		{"\xef\xbb\xbf" + valid, "invalid character '\\ufeff' looking for beginning of value"},
		{`{"edges": [], "links": {"total": NaN}}`, "invalid character 'N' looking for beginning of value"},
	} {
		_, err := graph.ParseGraph([]byte(tc.data), path)
		want := path + ": graph is not valid JSON (" + tc.reason + "); delete it and run `brain reindex`"
		if err == nil || err.Error() != want {
			t.Errorf("%q: got %v, want %q", tc.data, err, want)
		}
	}
}

func TestParseGraph_NonStringEndsFailTheTypedDecode(t *testing.T) {
	path := "/path/to/graph.json"
	_, err := graph.ParseGraph([]byte(`{"edges": [{"from": 1, "to": "b"}], "links": {"total": 0, "resolved": 0, "dropped": {}}}`), path)
	want := path + ": json: cannot unmarshal number into Go struct field Graph.edges.0.from of type string"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}

func TestReadGraph_AnUnstatablePathIsNeverIndexed(t *testing.T) {
	// Path.exists() answers False for every OSError, a NUL byte included.
	area := config.Area{Scope: "project/odd", Path: filepath.Join(t.TempDir(), "a\x00b")}
	_, err := graph.ReadGraph(area, "")
	if !errors.Is(err, graph.ErrNotIndexed) || err.Error() != "project/odd: never indexed; run `brain reindex`" {
		t.Fatalf("got %v", err)
	}
}

func TestReadGraph_InvalidUTF8NamesTheFileOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "graph.json")
	if err := os.WriteFile(path, []byte("{\"edges\": [], \"links\": \xff}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := graph.ReadGraph(config.Area{Scope: "project/test", Path: dir}, dir)
	if err == nil || err.Error() != path+": not valid UTF-8" {
		t.Fatalf("got %v", err)
	}
}

func TestReadGraph_AnUnreadableGraphNamesTheFileOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "graph.json")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := graph.ReadGraph(config.Area{Scope: "project/test", Path: dir}, dir)
	if err == nil || strings.Count(err.Error(), path) != 1 {
		t.Fatalf("got %v", err)
	}
}
