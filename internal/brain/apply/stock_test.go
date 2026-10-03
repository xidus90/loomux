package apply

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// legacyArea is a read-only area whose stock lies only in ultra-brain's
// state directory: a declaration, a register and a nested catalog.
func legacyArea(t *testing.T) (config.Area, config.ArtifactLookup, string) {
	t.Helper()
	ro := config.Area{Scope: "project/ro", Path: t.TempDir(), ReadOnly: true}
	lookup := config.ArtifactLookup{Primary: t.TempDir(), Fallback: t.TempDir()}
	old := config.ManifestDir(ro, lookup.Fallback)
	writeFile(t, filepath.Join(old, ".loomux", "config.toml"), "[area]\nscope = \"project/ro\"\n")
	writeRegister(t, old, row("A", "docs/a.md"))
	writeFile(t, filepath.Join(old, "catalog", "notes.md"), "# notes\n")
	return ro, lookup, old
}

// noStaging asserts that no staging directory is left beside target.
func noStaging(t *testing.T, target string) {
	t.Helper()
	left, err := filepath.Glob(target + ".*.staging")
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Fatalf("staging left behind: %v", left)
	}
}

// After the move the area resolves to the new place with every file, and a
// register written there is the one the lookup reads.
func TestMoveStockMovesALegacyAreaWhole(t *testing.T) {
	ro, lookup, _ := legacyArea(t)
	target := config.ManifestDir(ro, lookup.Primary)
	if err := moveStock(ro, lookup); err != nil {
		t.Fatal(err)
	}
	if got := config.ResolvedAreaDir(ro, lookup.Primary, lookup.Fallback); got != target {
		t.Fatalf("area resolves to %s, want %s", got, target)
	}
	for _, name := range []string{filepath.Join(".loomux", "config.toml"), registerName, filepath.Join("catalog", "notes.md")} {
		if !isFile(filepath.Join(target, name)) {
			t.Fatalf("%s did not move", name)
		}
	}
	if _, err := config.ReadAreaDeclaration(target); err != nil {
		t.Fatalf("the declaration no longer reads: %v", err)
	}
	noStaging(t, target)

	writeRegister(t, target, row("A", "docs/a.md"), row("B", "docs/b.md"))
	identities, err := identity.ReadIdentities(registerRead(ro, lookup))
	if err != nil {
		t.Fatal(err)
	}
	if _, found := identities["docs/b.md"]; !found {
		t.Fatalf("the written register is not the one read: %v", identities)
	}
}

// Idempotent: once the new place holds the area nothing is copied again.
func TestMoveStockCopiesOnlyOnce(t *testing.T) {
	ro, lookup, old := legacyArea(t)
	if err := moveStock(ro, lookup); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(old, "later.md"), "x")
	if err := moveStock(ro, lookup); err != nil {
		t.Fatal(err)
	}
	absent(t, filepath.Join(config.ManifestDir(ro, lookup.Primary), "later.md"))
}

// A writable area has no stock in a state directory, and an area neither
// place holds has nothing to move.
func TestMoveStockLeavesWhatHasNothingToMove(t *testing.T) {
	primary := t.TempDir()
	lookup := config.ArtifactLookup{Primary: primary, Fallback: t.TempDir()}
	if err := moveStock(config.Area{Scope: "w", Path: t.TempDir()}, lookup); err != nil {
		t.Fatal(err)
	}
	ro := config.Area{Scope: "project/ro", Path: t.TempDir(), ReadOnly: true}
	if err := moveStock(ro, lookup); err != nil {
		t.Fatal(err)
	}
	absent(t, filepath.Join(primary, "areas"))
}

// A swap a killed run left half-done is finished before anything is
// decided: the aside goes back, and the area already holds the new place.
func TestMoveStockRecoversAnAsideFirst(t *testing.T) {
	ro, lookup, _ := legacyArea(t)
	target := config.ManifestDir(ro, lookup.Primary)
	writeFile(t, filepath.Join(target+lock.AsideSuffix, "kept.md"), "x")
	if err := moveStock(ro, lookup); err != nil {
		t.Fatal(err)
	}
	if !isFile(filepath.Join(target, "kept.md")) {
		t.Fatal("the aside did not go back")
	}
	absent(t, filepath.Join(target, registerName))
}

func TestMoveStockPassesOnEveryFailure(t *testing.T) {
	boom := errors.New("boom")
	for name, fail := range map[string]func(t *testing.T){
		"recover": func(t *testing.T) { seam(t, &recoverDir, func(string) error { return boom }) },
		"staging": func(t *testing.T) { seam(t, &stagingDir, func(string) (string, error) { return "", boom }) },
		"read":    func(t *testing.T) { seam(t, &readStock, func(string) ([]byte, error) { return nil, boom }) },
		"swap":    func(t *testing.T) { seam(t, &swapDir, func(string, string) error { return boom }) },
	} {
		t.Run(name, func(t *testing.T) {
			ro, lookup, _ := legacyArea(t)
			fail(t)
			if err := moveStock(ro, lookup); !errors.Is(err, boom) {
				t.Fatalf("want the failure passed on, got %v", err)
			}
			target := config.ManifestDir(ro, lookup.Primary)
			absent(t, target)
			noStaging(t, target)
		})
	}
}

// A reindex that publishes the area between moveStock's decision and its
// swap wins: its stock is newer than the legacy copy, and no lock is shared
// that could have kept it out. moveStock keeps it and drops its own copy.
func TestMoveStockKeepsAnAreaPublishedBeforeTheSwap(t *testing.T) {
	ro, lookup, _ := legacyArea(t)
	target := config.ManifestDir(ro, lookup.Primary)
	seam(t, &beforeSwap, func() {
		writeFile(t, filepath.Join(target, "published.md"), "fresh")
	})
	if err := moveStock(ro, lookup); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(target, "published.md")); got != "fresh" {
		t.Fatalf("the published area was replaced: %q", got)
	}
	absent(t, filepath.Join(target, "catalog"))
	noStaging(t, target)
}

// A source that cannot be walked is an error, not an empty copy.
func TestCopyStockRefusesASourceThatIsNotThere(t *testing.T) {
	dir := t.TempDir()
	if err := copyStock(filepath.Join(dir, "gone"), filepath.Join(dir, "to")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want not-exist, got %v", err)
	}
}
