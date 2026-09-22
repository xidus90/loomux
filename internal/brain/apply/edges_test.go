package apply

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/lock"
)

// Edges no ported test of test_apply.py reaches: each pins a promise the
// code makes and a changed line would break without any other test noticing.

// Two sources of one case in one register: both rows move on, and the
// register is written and staged once.
func TestTwoSourcesInOneRegisterBothAdvance(t *testing.T) {
	v := newAppVault(t)
	second := appSources + "/zweite.md"
	writeFile(t, filepath.Join(v.root, filepath.FromSlash(second)), "Die zweite Quelle.\n")
	digest := hashOf(t, filepath.Join(v.root, filepath.FromSlash(second)))
	identities, err := identity.ReadIdentities(v.register())
	if err != nil {
		t.Fatal(err)
	}
	identities[second] = identity.Identity{DocID: "01DOC1", Relative: second, ContentHash: "sha256:old", Revision: 3}
	writeFile(t, v.register(), identity.RenderIdentities(identities))
	v.editCase(t, func(c *maintenance.Case) {
		c.Sources = append(c.Sources, maintenance.SourceState{DocID: "01DOC1", Revision: 3, ContentHash: digest})
	})

	v.mustApprove(t)

	after, err := identity.ReadIdentities(v.register())
	if err != nil {
		t.Fatal(err)
	}
	if got := after[appSources+"/quelle.md"]; got.Revision != 2 || got.ContentHash != hashOf(t, v.source()) {
		t.Fatalf("first row = %+v", got)
	}
	if got := after[second]; got.Revision != 4 || got.ContentHash != digest {
		t.Fatalf("second row = %+v", got)
	}
	add := (*v.calls)[0].add
	if n := len(slices.DeleteFunc(slices.Clone(add), func(p string) bool { return p != registerName })); n != 1 {
		t.Fatalf("register staged %d times: %q", n, add)
	}
}

// `_same_file` asks both paths for a file before it hashes either: a second
// path that is no file is "not the same", not a failure.
func TestSameFileIsFalseWhenEitherPathIsNoFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.md")
	writeFile(t, file, "x")
	for _, pair := range [][2]string{{file, filepath.Join(dir, "fehlt.md")}, {filepath.Join(dir, "fehlt.md"), file}, {file, dir}} {
		same, err := sameFile(pair[0], pair[1])
		if same || err != nil {
			t.Errorf("sameFile(%q, %q) = %v, %v; want false, nil", pair[0], pair[1], same, err)
		}
	}
}

// `write_if_changed` compares only a file that exists: an empty text for a
// missing file is a change, and the file is created.
func TestReplaceIfChangedCreatesAnEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "leer.md")
	changed, err := replaceIfChanged(path, "")
	if !changed || err != nil {
		t.Fatalf("replaceIfChanged = %v, %v; want true, nil", changed, err)
	}
	if data, err := os.ReadFile(path); err != nil || len(data) != 0 {
		t.Fatalf("file = %q, %v; want it empty", data, err)
	}
}

// `name.split(":", 1)[0]`: a name that begins with a stream separator is
// the empty name, not the name after it.
func TestNormalisedNameCutsAStreamAtTheFirstCharacter(t *testing.T) {
	if got := normalisedName(":_identities.tsv"); got != "" {
		t.Fatalf("normalisedName = %q, want empty", got)
	}
}

// pathlib drops a `.` component when it builds the path, so a spelling that
// carries one names the same vault-relative path.
func TestRelativeDropsADotComponent(t *testing.T) {
	p, vault := newPlace(t)
	path := vault + string(filepath.Separator) + "." + string(filepath.Separator) + filepath.Join("wiki", "a.md")
	if got := p.relative(path); got != "wiki/a.md" {
		t.Fatalf("relative = %q, want wiki/a.md", got)
	}
}

// A resolver failure on the written path is passed on as it is, also under
// a wiki outside the vault -- where guessing the vault as the anchor would
// turn it into a refusal that names the wrong cause.
func TestAnOutsideWikiPassesOnAResolverFailure(t *testing.T) {
	p, _ := newPlace(t)
	p.wiki = t.TempDir()
	target := filepath.Join(p.wiki, "a.md")
	cycle := errors.New("its links lead in a circle")
	real := resolvePath
	seam(t, &resolvePath, func(path string) (string, error) {
		if path == target {
			return "", cycle
		}
		return real(path)
	})
	if err := p.write(target, "x"); !errors.Is(err, cycle) {
		t.Fatalf("want the resolver's error, got %v", err)
	}
}

// A writable area has no stock to move, and nothing beside it is touched:
// not even a directory named like a swap's leftover.
func TestAWritableAreasNeighbourIsLeftAlone(t *testing.T) {
	v := newAppVault(t)
	aside := v.root + lock.AsideSuffix
	writeFile(t, filepath.Join(aside, "keep.md"), "not a stock")
	v.mustApprove(t)
	if _, err := os.Stat(filepath.Join(aside, "keep.md")); err != nil {
		t.Fatalf("the directory beside the vault is gone: %v", err)
	}
}
