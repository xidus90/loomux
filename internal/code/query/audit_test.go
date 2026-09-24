package query

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
)

// stagedAdd is the graph case repository with Add's body changed and staged.
func stagedAdd(t *testing.T) string {
	t.Helper()
	root := gitRepo(t)
	editAdd(t, root)
	git(t, root, "add", ".")
	return root
}

func TestAuditFindsAStaleAreaAtTheThreshold(t *testing.T) {
	a, _, err := Audit(stagedAdd(t), AuditOptions{Cached: true, Threshold: 2})
	if err != nil {
		t.Fatal(err)
	}
	if a.Range != "index against HEAD" || a.Areas != 1 || len(a.Findings) != 1 {
		t.Fatalf("answer = %+v", a)
	}
	f := a.Findings[0]
	if f.Path != "calc/calc.go" || f.Signal != blast.SignalStale || len(f.Seeds) != 1 ||
		f.Seeds[0].Node.Name != "Add" || f.Seeds[0].InDegree != 2 {
		t.Fatalf("finding = %+v", f)
	}
}

func TestAuditBelowTheThresholdFindsNothing(t *testing.T) {
	a, _, err := Audit(stagedAdd(t), AuditOptions{Cached: true, Threshold: 3})
	if err != nil || len(a.Findings) != 0 || a.Areas != 1 {
		t.Fatalf("answer = %+v, err %v", a, err)
	}
}

func TestAuditSkipsTestCallersWhenAsked(t *testing.T) {
	a, _, err := Audit(stagedAdd(t), AuditOptions{Cached: true, Threshold: 2, SkipTestCallers: true})
	if err != nil || len(a.Findings) != 0 {
		t.Fatalf("answer = %+v, err %v", a, err)
	}
	// main alone still counts.
	a, _, err = Audit(stagedAdd(t), AuditOptions{Cached: true, Threshold: 1, SkipTestCallers: true})
	if err != nil || len(a.Findings) != 1 || a.Findings[0].Seeds[0].InDegree != 1 {
		t.Fatalf("answer = %+v, err %v", a, err)
	}
}

func TestAuditIsGreenWhenTheTestChangedToo(t *testing.T) {
	root := stagedAdd(t)
	p := filepath.Join(root, "calc", "calc_test.go")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, append(data, []byte("\n// touched\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", ".")
	a, _, err := Audit(root, AuditOptions{Cached: true, Threshold: 1})
	if err != nil || len(a.Findings) != 0 {
		t.Fatalf("answer = %+v, err %v", a, err)
	}
}

func TestAuditNeverRebuildsTheGraph(t *testing.T) {
	root := stagedAdd(t)
	wiring := store.WiringPath(root)
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(wiring, past, past); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Audit(root, AuditOptions{Cached: true, Threshold: 2}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(wiring)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(past) {
		t.Fatalf("wiring.json was rewritten at %v", info.ModTime())
	}
	// The fingerprint was stale: a blast that may refresh does rebuild.
	if _, _, err := Blast(root, BlastOptions{Cached: true}); err != nil {
		t.Fatal(err)
	}
	if info, err = os.Stat(wiring); err != nil || info.ModTime().Equal(past) {
		t.Fatalf("a refreshing blast left wiring.json alone (%v)", err)
	}
}

func TestAuditRefusesAThresholdBelowOne(t *testing.T) {
	_, _, err := Audit(t.TempDir(), AuditOptions{Threshold: 0})
	if err == nil || err.Error() != "threshold must be at least 1" {
		t.Fatalf("err = %v", err)
	}
}

func TestAuditPassesBlastsErrorOn(t *testing.T) {
	_, _, err := Audit(t.TempDir(), AuditOptions{Base: "HEAD~1", Cached: true, Threshold: 1})
	if !errors.Is(err, ErrBaseAndCached) {
		t.Fatalf("err = %v", err)
	}
}

func TestAuditReport(t *testing.T) {
	add := &model.Node{Name: "Add"}
	sub := &model.Node{Name: "Sub"}
	got := AuditReport(AuditAnswer{Range: "index against HEAD", Areas: 2, Findings: []AuditFinding{
		{Path: "calc/calc.go", Signal: blast.SignalStale, Seeds: []blast.Seed{{Node: add, InDegree: 4}, {Node: sub, InDegree: 3}}},
		{Path: "lib/lib.go", Signal: blast.SignalNone, Seeds: []blast.Seed{{Node: sub, InDegree: 3}}},
	}}, 3)
	want := "blast audit: index against HEAD, threshold 3\n" +
		"calc/calc.go [stale]: Add in-degree 4, Sub in-degree 3\n" +
		"lib/lib.go [none]: Sub in-degree 3\n"
	if got != want {
		t.Fatalf("report =\n%s\nwant\n%s", got, want)
	}
	got = AuditReport(AuditAnswer{Range: "index against HEAD", Areas: 2}, 3)
	if want := "no area at or above 3 callers lacks a changed test (2 areas)\n"; got != want {
		t.Fatalf("report = %q, want %q", got, want)
	}
}

// The graph the audit reads is the one the blast radius read.
func TestBlastWithReturnsTheGraphItRead(t *testing.T) {
	_, g, _, err := blastWith(stagedAdd(t), BlastOptions{Cached: true, NoRefresh: true})
	if err != nil || g == nil || len(g.Nodes) == 0 {
		t.Fatalf("graph = %v, err %v", g, err)
	}
}
