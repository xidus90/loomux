package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
)

// noReviewCentreWarning is the reference's warning word for word, `brain`
// read as `loomux` -- which it does not name, so nothing changed.
const noReviewCentreWarning = "warning: kein registrierter Bereich erklärt ein Prüfzentrum " +
	"([layout] review) -- es wurde nicht abgeglichen. Der Indexlauf schreibt " +
	"geänderte Quellen ins Register, ohne dass ein Prüffall entsteht. Fehlt das " +
	"Manifest des erklärenden Bereichs nur (etwa weil sein Laufwerk nicht " +
	"eingehängt ist), sind diese Änderungen danach nicht mehr auffindbar.\n"

// register is the file an index run rewrites and a refused one leaves alone:
// the identity register of the world's one area, which is writable and so
// keeps its artefacts in its own tree rather than under `<state>/areas/`.
func (w reconcileWorld) register() string {
	return filepath.Join(w.Area, "_identities.tsv")
}

// readRegister is the register's bytes and modification time, so a case can
// tell a rewritten register from one nobody touched.
func readRegister(t *testing.T, w reconcileWorld) (string, int64) {
	t.Helper()
	raw, err := os.ReadFile(w.register())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	info, err := os.Stat(w.register())
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	return string(raw), info.ModTime().UnixNano()
}

// The catch-up is no gate of its own: the cases it opened are named on stderr,
// where no reader of stdout meets them, and the index run goes on and advances
// the register.
func TestReindexIndexesDespiteOpenCases(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{Cite: true})
	stubBrainStatusPort(t, search.NewFakePort())
	before, _ := readRegister(t, world)

	code, out, errOut := run("reindex")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	line := "  " + filepath.Base(world.caseDir(t)) + "\tproject/a\tpage.md\tsource_changed\n"
	if !strings.Contains(errOut, "warning: 1 neuer Fall durch die Aufholung eröffnet:\n"+line) {
		t.Fatalf("stderr does not list the case under its heading: %q", errOut)
	}
	if strings.Contains(out, "source_changed") {
		t.Fatalf("stdout carries the case: %q", out)
	}
	if !strings.Contains(out, "indexed the areas of") {
		t.Fatalf("stdout does not report the index run: %q", out)
	}
	if after, _ := readRegister(t, world); after == before {
		t.Fatalf("the register was not rewritten: %q", after)
	}
}

// The pass covers the registry the run is about to walk, not the registered
// one: a scan of other areas would assure nothing about the sources indexed.
// The state directory holds no registry at all here, so a catch-up that read
// the default would stop the run over a missing file.
func TestReindexCatchesUpOnTheNamedRegistry(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{Cite: true})
	elsewhere := filepath.Join(filepath.Dir(world.State), "elsewhere")
	if err := os.MkdirAll(elsewhere, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.Rename(filepath.Join(world.State, "registry.toml"),
		filepath.Join(elsewhere, "registry.toml")); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	stubBrainStatusPort(t, search.NewFakePort())

	code, _, errOut := run("reindex", "--registry", elsewhere)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, "warning: 1 neuer Fall durch die Aufholung eröffnet:\n") {
		t.Fatalf("stderr does not list the case of the named registry: %q", errOut)
	}
}

// Several cases take the plural, and every one of them gets its line.
func TestReindexCountsSeveralOpenCases(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{Cite: true})
	writeFile(t, filepath.Join(world.Area, "wiki", "second.md"),
		"---\ntype: note\ntitle: Second\nsources:\n  - resource: src/note1.md\n    doc_id: "+
			strings.Repeat("0", 25)+"1\n---\n\n# Second\n\nWhat the source says, again.\n")
	stubBrainStatusPort(t, search.NewFakePort())

	code, _, errOut := run("reindex")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, "warning: 2 neue Fälle durch die Aufholung eröffnet:\n") {
		t.Fatalf("stderr does not count both cases: %q", errOut)
	}
	for _, page := range []string{"page.md", "second.md"} {
		if !strings.Contains(errOut, "\tproject/a\t"+page+"\tsource_changed\n") {
			t.Fatalf("stderr does not list the case of %s: %q", page, errOut)
		}
	}
}

// A corpus without a review centre has no gate to walk around, so there is no
// order to enforce: the run warns, because the state reads the same as a
// declaring vault whose manifest merely went missing, and indexes.
func TestReindexWarnsWithoutAReviewCentreAndStillIndexes(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{Cite: true, NoReview: true})
	stubBrainStatusPort(t, search.NewFakePort())
	before, _ := readRegister(t, world)

	code, _, errOut := run("reindex")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, noReviewCentreWarning) {
		t.Fatalf("stderr does not carry the warning: %q", errOut)
	}
	if after, _ := readRegister(t, world); after == before {
		t.Fatalf("the register was not rewritten: %q", after)
	}
}

// The one state in which nothing is indexed. A second area whose declaration
// does not read stops the catch-up -- and the index run alone would have gone
// on: it skips an area it cannot read and rewrites the register of the first.
// So the untouched register is what shows the refusal, not the exit code.
func TestReindexRefusesWhenTheCatchUpFails(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{Cite: true})
	broken := filepath.Join(filepath.Dir(world.Area), "broken")
	registry := filepath.Join(world.State, "registry.toml")
	raw, err := os.ReadFile(registry)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	writeFile(t, registry, string(raw)+
		"\n[[area]]\nscope = \"project/b\"\npath = "+strconv.Quote(filepath.ToSlash(broken))+"\n")
	writeFile(t, filepath.Join(broken, ".loomux", "config.toml"), "this is not TOML {{{\n")
	port := search.NewFakePort()
	stubBrainStatusPort(t, port)
	before, stamp := readRegister(t, world)

	code, out, errOut := run("reindex")
	if code != 1 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.HasPrefix(errOut, "error: die Aufholung vor dem Indexlauf ist fehlgeschlagen (") {
		t.Fatalf("stderr does not open with the refusal: %q", errOut)
	}
	if !strings.Contains(errOut, broken) {
		t.Fatalf("stderr does not name the cause: %q", errOut)
	}
	if !strings.HasSuffix(errOut, "). Es wurde nicht indiziert: ein Indexlauf ohne vorherigen "+
		"Abgleich schreibt geänderte Quellen ins Register, ohne dass je ein Prüffall entsteht. "+
		"Ursache beheben, dann `loomux reconcile` und danach `loomux reindex`.\n") {
		t.Fatalf("stderr does not name the way out: %q", errOut)
	}
	if out != "" {
		t.Fatalf("stdout reports a run that did not happen: %q", out)
	}
	after, afterStamp := readRegister(t, world)
	if after != before || afterStamp != stamp {
		t.Fatalf("the register was touched: %q", after)
	}
	if len(port.Refreshed) != 0 {
		t.Fatalf("the engine was asked to refresh: %v", port.Refreshed)
	}
}

// A broken case file is named, as `reconcile` names it, but here it scores
// nothing: the reference drops the exit `_report_unreadable` hands back and
// goes on to index.
func TestReindexNamesAnUnreadableCaseAndStillIndexes(t *testing.T) {
	world := newReconcileWorld(t, reconcileOptions{Cite: true})
	broken := filepath.Join(world.Review, "project-a", "handmade", "case.toml")
	writeFile(t, broken, "this is not TOML {{{\n")
	stubBrainStatusPort(t, search.NewFakePort())
	before, _ := readRegister(t, world)

	code, _, errOut := run("reindex")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, "unreadable case: "+broken+"\n") {
		t.Fatalf("stderr does not name the broken case: %q", errOut)
	}
	if after, _ := readRegister(t, world); after == before {
		t.Fatalf("the register was not rewritten: %q", after)
	}
}
