package apply

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// A swap a killed `index` left half-done is finished: the aside goes back
// whole, and nothing of it is lost.
func TestRecoverStockPutsAnAsideBack(t *testing.T) {
	lookup := config.ArtifactLookup{Primary: t.TempDir()}
	ro := config.Area{Scope: "project/ro", Path: t.TempDir(), ReadOnly: true}
	target := config.ManifestDir(ro, lookup.Primary)
	writeFile(t, filepath.Join(target+lock.AsideSuffix, "catalog.tsv"), "stock\n")
	if err := recoverStock(ro, lookup); err != nil {
		t.Fatal(err)
	}
	if readFile(t, filepath.Join(target, "catalog.tsv")) != "stock\n" {
		t.Fatal("the aside did not go back")
	}
	absent(t, target+lock.AsideSuffix)
}

// A writable area is never swapped: a directory beside its tree that is
// named like an aside is someone else's, and stays.
func TestRecoverStockLeavesAWritableAreaAlone(t *testing.T) {
	lookup := config.ArtifactLookup{Primary: t.TempDir()}
	area := config.Area{Scope: "w", Path: t.TempDir()}
	aside := area.Path + lock.AsideSuffix
	writeFile(t, filepath.Join(aside, "kept.md"), "x")
	if err := recoverStock(area, lookup); err != nil {
		t.Fatal(err)
	}
	if !isFile(filepath.Join(aside, "kept.md")) {
		t.Fatal("a writable area's neighbour was touched")
	}
}

func TestRecoverStockPassesOnTheFailure(t *testing.T) {
	boom := errors.New("boom")
	seam(t, &recoverDir, func(string) error { return boom })
	ro := config.Area{Scope: "project/ro", Path: t.TempDir(), ReadOnly: true}
	if err := recoverStock(ro, config.ArtifactLookup{Primary: t.TempDir()}); !errors.Is(err, boom) {
		t.Fatalf("want the failure passed on, got %v", err)
	}
}
