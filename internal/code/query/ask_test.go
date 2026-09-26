package query

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/store"
)

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
	if _, _, err := Build(root, ignore); err != nil {
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
	if _, _, err := Build(root, ignore); err != nil {
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

// The probe and the build must name the same stamp, the combined one: a
// record another extractor wrote is rebuilt once, and the record that rebuild
// writes is then trusted. Two different strings would rebuild on every
// question.
func TestAskRebuildsARecordAnotherExtractorWroteOnce(t *testing.T) {
	root := repo(t, sample())
	_, stats, err := Build(root, ignore)
	if err != nil {
		t.Fatal(err)
	}
	if err := freshness.Write(root, "go/0", stats.Files, stats.Hashes); err != nil {
		t.Fatal(err)
	}
	_, notes, err := Ask(root, "run", AskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) == 0 || !strings.Contains(notes[0], "building the graph") {
		t.Errorf("notes = %v, want a rebuild for a record another extractor wrote", notes)
	}
	a, notes, err := Ask(root, "run", AskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Errorf("notes = %v, want none: the rebuilt record is this binary's", notes)
	}
	if !strings.Contains(AskReport(a), "lib/lib.go") {
		t.Errorf("report %q must find lib/lib.go", AskReport(a))
	}
}

func TestAskNoRefreshLeavesDriftAlone(t *testing.T) {
	root := repo(t, sample())
	if _, _, err := Build(root, ignore); err != nil {
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
	if _, _, err := Build(root, ignore); err != nil {
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
	if _, _, err := Build(root, ignore); err != nil {
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

func TestAskWithoutSourceCarriesNoCode(t *testing.T) {
	root := built(t)
	a, _, err := Ask(root, "run", AskOptions{NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Hits) == 0 {
		t.Fatal("want a hit")
	}
	for _, h := range a.Hits {
		if h.Code != "" {
			t.Errorf("hit %s carries code without Source", h.ID)
		}
	}
}

func TestAskReportPrintsSignatureAndCodeOnlyWhenThere(t *testing.T) {
	a := ask.Answer{Hits: []ask.Hit{
		{ID: "lib/lib.go#Run", Path: "lib/lib.go", Span: "L4-L5", Signature: "func Run()", Code: "func Run() {\n}", Score: 1, Lexical: 0.5, Graph: 0.25},
		{ID: "main.go", Path: "main.go", Span: "L1-L5"},
	}}
	want := "1. lib/lib.go#Run  lib/lib.go:L4-L5  (1.000 lex 0.500 graph 0.250)\n" +
		"   func Run()\n" +
		"   | func Run() {\n" +
		"   | }\n" +
		"2. main.go  main.go:L1-L5  (0.000 lex 0.000 graph 0.000)\n"
	if got := AskReport(a); got != want {
		t.Errorf("AskReport =\n%q\nwant\n%q", got, want)
	}
}
