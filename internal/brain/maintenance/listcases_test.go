package maintenance_test

import (
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/config"
)

// reviewWorld is a vault whose one area is the review centre, with its
// registry written, and the directory the cases go under.
func reviewWorld(t *testing.T) (*world, string) {
	t.Helper()
	w := newWorld(t)
	w.addArea(t, areaOptions{Files: map[string]string{"a.md": "x\n"}, Review: "review"})
	if err := config.WriteRegistry(w.StateDir, w.Areas); err != nil {
		t.Fatal(err)
	}
	root, err := maintenance.ReviewRoot(w.Areas, w.Lookup())
	if err != nil {
		t.Fatal(err)
	}
	return w, root
}

const standing = "id = \"x\"\narea = \"project/one\"\ntarget = \"a.md\"\ntarget_hash = \"aa\"\n" +
	"state = \"due\"\ntrigger = \"source_change\"\nweight = \"change\"\ncreated = 2026-09-03T12:00:00Z\n"

func TestListCasesAddressesEachCaseByItsDirectoryAndNamesTheBrokenOnes(t *testing.T) {
	w, root := reviewWorld(t)
	writeUnder(t, root, "project-one/beta/case.toml", standing)
	writeUnder(t, root, "project-one/alpha/case.toml", standing)
	writeUnder(t, root, "project-one/broken/case.toml", "not toml [")
	listed, broken, err := maintenance.ListCases(w.StateDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 || listed[0].Address != "alpha" || listed[1].Address != "beta" {
		t.Fatalf("listed = %+v, want alpha and beta", listed)
	}
	if want := filepath.Join(root, "project-one", "alpha"); listed[0].Directory != want || listed[0].Case.Target != "a.md" {
		t.Errorf("alpha = %+v, want directory %s and the case read", listed[0], want)
	}
	if len(broken) != 1 {
		t.Errorf("broken = %q, want the one unreadable case", broken)
	}
}

func TestListCasesOfAnEmptyReviewCentreIsEmpty(t *testing.T) {
	w, _ := reviewWorld(t)
	if listed, broken, err := maintenance.ListCases(w.StateDir); err != nil || len(listed) != 0 || len(broken) != 0 {
		t.Fatalf("ListCases = %v, %v, %v, want nothing", listed, broken, err)
	}
}

func TestListCasesRefusesABrokenRegistryAndAVaultWithoutAReviewCentre(t *testing.T) {
	w := newWorld(t)
	writeUnder(t, w.StateDir, "registry.toml", "[[area]\n")
	if _, _, err := maintenance.ListCases(w.StateDir); err == nil {
		t.Error("a broken registry listed cases")
	}
	w = newWorld(t)
	w.addArea(t, areaOptions{Files: map[string]string{"a.md": "x\n"}})
	if err := config.WriteRegistry(w.StateDir, w.Areas); err != nil {
		t.Fatal(err)
	}
	if _, _, err := maintenance.ListCases(w.StateDir); err == nil {
		t.Error("a vault without a review centre listed cases")
	}
}
