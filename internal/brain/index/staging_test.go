package index

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// writeFile puts one file on disk, parents and all.
func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

// readOnlyArea is a registered area whose stock may not lie in its own tree.
func readOnlyArea(path string) config.Area {
	return config.Area{Scope: "read/only", Path: path, ReadOnly: true}
}

// A read-only area's stock lands under the state directory, and it lands
// whole: the declaration that still lay in the old state directory comes
// along, because every read of the area -- the declaration included --
// switches to the new place the instant anything lies there.
func TestReindexWritesAReadOnlyAreaWholeIntoTheStateDirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	areaDir := filepath.Join(tmp, "notes")
	writeFile(t, filepath.Join(areaDir, "one.md"), "---\ntitle: One\n---\n# One\n")

	legacyDir := filepath.Join(tmp, "legacy")
	writeFile(t, filepath.Join(legacyDir, "areas", "read-only", ".brain.toml"), "[area]\nscope = \"read/only\"\n")

	stateDir := filepath.Join(tmp, "state")
	registry := "[[area]]\nscope = \"read/only\"\npath = \"" + filepath.ToSlash(areaDir) + "\"\nreadonly = true\n"
	regPath := writeTestRegistry(t, stateDir, registry)

	var stderr bytes.Buffer
	code, err := ReindexWithOutput(regPath, stateDir, legacyDir, search.NewFakePort(), &stderr)
	if err != nil || code != 0 {
		t.Fatalf("Reindex = %d, %v; stderr: %s", code, err, stderr.String())
	}

	target := filepath.Join(stateDir, "areas", "read-only")
	for _, name := range []string{".brain.toml", "index.md", graphName, identitiesName} {
		if _, err := os.Stat(filepath.Join(target, name)); err != nil {
			t.Errorf("%s missing from the swapped-in stock: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(areaDir, "index.md")); err == nil {
		t.Error("a read-only area was written in its own tree")
	}
}

// The run breaks in the middle of one area's stock. What the readers see
// afterwards is the old stock whole -- never the new root catalog beside the
// old subdirectory, which is the state that would make the whole vault
// answer nothing.
func TestReindexLeavesTheOldStockWholeWhenAWriteBreaks(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	areaDir := filepath.Join(tmp, "notes")
	writeFile(t, filepath.Join(areaDir, "one.md"), "---\ntitle: One\n---\n# One\n")
	writeFile(t, filepath.Join(areaDir, "sub", "two.md"), "---\ntitle: Two\n---\n# Two\n")
	// An intro nobody wrote anything into: ReadIntro refuses it rather than
	// ignoring the author's file, and it refuses it after the root catalog of
	// this area has already been written.
	writeFile(t, filepath.Join(areaDir, "sub", "index.intro.md"), "   \n")

	stateDir := filepath.Join(tmp, "state")
	target := filepath.Join(stateDir, "areas", "read-only")
	writeFile(t, filepath.Join(target, ".brain.toml"), "[area]\nscope = \"read/only\"\n")
	writeFile(t, filepath.Join(target, "index.md"), "# old\n")
	writeFile(t, filepath.Join(target, "sub", "index.md"), "# old sub\n")

	registry := "[[area]]\nscope = \"read/only\"\npath = \"" + filepath.ToSlash(areaDir) + "\"\nreadonly = true\n"
	regPath := writeTestRegistry(t, stateDir, registry)

	var stderr bytes.Buffer
	code, err := ReindexWithOutput(regPath, stateDir, "", search.NewFakePort(), &stderr)
	if err == nil || code != 1 {
		t.Fatalf("Reindex = %d, %v; want the empty intro to end the run", code, err)
	}

	for path, want := range map[string]string{
		filepath.Join(target, "index.md"):        "# old\n",
		filepath.Join(target, "sub", "index.md"): "# old sub\n",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile %s: %v", path, err)
		}
		if string(data) != want {
			t.Errorf("%s = %q, want the old stock %q", path, data, want)
		}
	}

	entries, err := os.ReadDir(filepath.Join(stateDir, "areas"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	// The area and the lock index holds beside it, never inside it.
	if len(entries) != 2 || entries[0].Name() != "read-only" || entries[1].Name() != "read-only.lock" {
		t.Errorf("entries = %v, want only the area itself and its lock -- no staging, no aside", entries)
	}
}

// A staging directory that cannot be made ends the area before anything is
// read, let alone written.
func TestPublishReportsAStagingItCannotMake(t *testing.T) {
	tmp := t.TempDir()
	blocked := filepath.Join(tmp, "state")
	writeFile(t, blocked, "not a directory\n")

	err := publish(readOnlyArea(tmp), nil, map[string]identity.Identity{}, tmp, blocked)
	if err == nil {
		t.Fatal("expected an error below a plain file")
	}
}

// The stock that is to be carried forward is unreadable. Nothing is swapped
// in: a staging seeded from nothing would drop the declaration and make the
// area the half state this whole movement exists to prevent.
func TestPublishReportsAStockItCannotCopy(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "gone")

	err := publish(readOnlyArea(tmp), nil, map[string]identity.Identity{}, missing, filepath.Join(tmp, "state"))
	if err == nil {
		t.Fatal("expected an error copying a stock that is not there")
	}
	entries, err := os.ReadDir(filepath.Join(tmp, "state", "areas"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entries = %v, want the staging cleaned up", entries)
	}
}

func TestPublishReportsAStockItCannotWrite(t *testing.T) {
	tmp := t.TempDir()
	areaDir := filepath.Join(tmp, "notes")
	writeFile(t, filepath.Join(areaDir, "sub", "index.intro.md"), "\n")
	source := filepath.Join(tmp, "source")
	writeFile(t, filepath.Join(source, ".brain.toml"), "[area]\nscope = \"read/only\"\n")

	documents := []Document{{Relative: "sub/two.md", Title: "Two"}}
	err := publish(readOnlyArea(areaDir), documents, map[string]identity.Identity{}, source, filepath.Join(tmp, "state"))
	if err == nil || !strings.Contains(err.Error(), "intro file is empty") {
		t.Fatalf("err = %v, want the empty intro", err)
	}
}

// The graph is the second artefact written, so a directory of that name is
// what fails a run after its catalogs went through.
func TestWriteStockReportsAFailedGraphWrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, graphName), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	err := writeStock(readOnlyArea(dir), nil, map[string]identity.Identity{}, dir)
	if err == nil {
		t.Fatal("expected an error writing over a directory")
	}
}

func TestCopyTreeReportsASourceItCannotWalk(t *testing.T) {
	tmp := t.TempDir()
	if err := copyTree(filepath.Join(tmp, "gone"), tmp); err == nil {
		t.Fatal("expected an error walking a directory that is not there")
	}
}

func TestCopyTreeReportsATargetItCannotMake(t *testing.T) {
	tmp := t.TempDir()
	blocked := filepath.Join(tmp, "file")
	writeFile(t, blocked, "x\n")
	if err := copyTree(tmp, filepath.Join(blocked, "below")); err == nil {
		t.Fatal("expected an error below a plain file")
	}
}

func TestCopyFileReportsASourceItCannotRead(t *testing.T) {
	tmp := t.TempDir()
	err := copyFile(filepath.Join(tmp, "gone"), filepath.Join(tmp, "copy"))
	if err == nil {
		t.Fatal("expected an error reading a file that is not there")
	}
}

// A swap a killed run left half-done is finished before this run reads
// anything of the area -- not at the end, where the stock it carried forward
// would already have come from the wrong place.
func TestReindexFinishesAnInterruptedSwapBeforeItReads(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	areaDir := filepath.Join(tmp, "notes")
	writeFile(t, filepath.Join(areaDir, "one.md"), "---\ntitle: One\n---\n# One\n")

	stateDir := filepath.Join(tmp, "state")
	target := filepath.Join(stateDir, "areas", "read-only")
	aside := target + lock.AsideSuffix
	writeFile(t, filepath.Join(aside, ".brain.toml"), "[area]\nscope = \"read/only\"\n")
	writeFile(t, filepath.Join(aside, identitiesName), identity.IdentitiesHeader+"\n")

	// The old state directory holds a stock of its own. Were the recovery to
	// wait until the swap, this run would read that one and undo what the
	// killed run had already replaced.
	legacyDir := filepath.Join(tmp, "legacy")
	writeFile(t, filepath.Join(legacyDir, "areas", "read-only", ".brain.toml"), "[area]\nscope = \"read/only\"\n[index]\ninclude = [\"nothing/*.md\"]\n")

	registry := "[[area]]\nscope = \"read/only\"\npath = \"" + filepath.ToSlash(areaDir) + "\"\nreadonly = true\n"
	regPath := writeTestRegistry(t, stateDir, registry)

	var stderr bytes.Buffer
	code, err := ReindexWithOutput(regPath, stateDir, legacyDir, search.NewFakePort(), &stderr)
	if err != nil || code != 0 {
		t.Fatalf("Reindex = %d, %v; stderr: %s", code, err, stderr.String())
	}
	if _, err := os.Stat(aside); !os.IsNotExist(err) {
		t.Errorf("aside still there: %v", err)
	}
	catalog, err := os.ReadFile(filepath.Join(target, "index.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(catalog), "one.md") {
		t.Errorf("catalog = %q, want the recovered declaration's include, not the old state directory's", catalog)
	}
}

// A recovery that fails ends the run for that area rather than reading it
// from a place the aside beside it contradicts. The refusal is an operating
// system declining a rename, which no test can ask for on both platforms --
// hence the seam.
func TestReindexStopsWhenAnInterruptedSwapCannotBeFinished(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	areaDir := filepath.Join(tmp, "notes")
	writeFile(t, filepath.Join(areaDir, "one.md"), "# One\n")

	stateDir := filepath.Join(tmp, "state")
	registry := "[[area]]\nscope = \"read/only\"\npath = \"" + filepath.ToSlash(areaDir) + "\"\nreadonly = true\n"
	regPath := writeTestRegistry(t, stateDir, registry)

	refused := errors.New("refused")
	previous := recoverStockFn
	recoverStockFn = func(config.Area, string) error { return refused }
	defer func() { recoverStockFn = previous }()

	var stderr bytes.Buffer
	code, err := ReindexWithOutput(regPath, stateDir, "", search.NewFakePort(), &stderr)
	if !errors.Is(err, refused) || code != 1 {
		t.Fatalf("Reindex = %d, %v; want it to wrap %v", code, err, refused)
	}
}

// The staging is full and the swap itself is what breaks -- the half the
// filling test cannot reach. What the readers find afterwards is the old
// stock whole: both catalogs as they were, and nothing beside them.
//
// This is the test that holds the movement in place. Write the new stock into
// the target file by file instead of swapping it in, and it dies: the old
// root catalog is then already overwritten when the break comes.
func TestReindexLeavesTheOldStockWholeWhenTheSwapBreaks(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	areaDir := filepath.Join(tmp, "notes")
	writeFile(t, filepath.Join(areaDir, "one.md"), "---\ntitle: One\n---\n# One\n")
	writeFile(t, filepath.Join(areaDir, "sub", "two.md"), "---\ntitle: Two\n---\n# Two\n")

	stateDir := filepath.Join(tmp, "state")
	target := filepath.Join(stateDir, "areas", "read-only")
	writeFile(t, filepath.Join(target, ".brain.toml"), "[area]\nscope = \"read/only\"\n")
	writeFile(t, filepath.Join(target, "index.md"), "# old\n")
	writeFile(t, filepath.Join(target, "sub", "index.md"), "# old sub\n")

	registry := "[[area]]\nscope = \"read/only\"\npath = \"" + filepath.ToSlash(areaDir) + "\"\nreadonly = true\n"
	regPath := writeTestRegistry(t, stateDir, registry)

	// A run killed in the swap. The staging is complete at this instant --
	// every catalog of the new stock is written -- and none of it may have
	// reached the target.
	refused := errors.New("refused")
	previous := replaceDirFn
	replaceDirFn = func(staging, _ string) error {
		if _, err := os.Stat(filepath.Join(staging, "index.md")); err != nil {
			t.Errorf("the staging was not complete when the swap began: %v", err)
		}
		return refused
	}
	defer func() { replaceDirFn = previous }()

	var stderr bytes.Buffer
	code, err := ReindexWithOutput(regPath, stateDir, "", search.NewFakePort(), &stderr)
	if !errors.Is(err, refused) || code != 1 {
		t.Fatalf("Reindex = %d, %v; want it to wrap %v", code, err, refused)
	}

	for path, want := range map[string]string{
		filepath.Join(target, "index.md"):        "# old\n",
		filepath.Join(target, "sub", "index.md"): "# old sub\n",
	} {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("ReadFile %s: %v", path, readErr)
		}
		if string(data) != want {
			t.Errorf("%s = %q, want the old stock %q -- the new stock reached the target", path, data, want)
		}
	}

	entries, err := os.ReadDir(filepath.Join(stateDir, "areas"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	// The area and the lock index holds beside it, never inside it.
	if len(entries) != 2 || entries[0].Name() != "read-only" || entries[1].Name() != "read-only.lock" {
		t.Errorf("entries = %v, want only the area itself and its lock -- no staging, no aside", entries)
	}
}

// A swap replaces, a copy merges. What the old stock carried and the new one
// does not is gone afterwards -- otherwise an area would accumulate the
// remains of every stock it ever had, and a reader could not tell which run
// wrote what it is looking at.
func TestPublishReplacesTheStockInsteadOfMergingIntoIt(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	target := filepath.Join(stateDir, "areas", "read-only")
	writeFile(t, filepath.Join(target, "stale.md"), "# from a stock long gone\n")

	source := filepath.Join(tmp, "source")
	writeFile(t, filepath.Join(source, ".brain.toml"), "[area]\nscope = \"read/only\"\n")

	areaDir := filepath.Join(tmp, "notes")
	if err := os.MkdirAll(areaDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := publish(readOnlyArea(areaDir), nil, map[string]identity.Identity{}, source, stateDir); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "stale.md")); !os.IsNotExist(err) {
		t.Errorf("stale.md survived: %v -- the stock was merged into, not replaced", err)
	}
	if _, err := os.Stat(filepath.Join(target, ".brain.toml")); err != nil {
		t.Errorf("the declaration did not come along: %v", err)
	}
}
