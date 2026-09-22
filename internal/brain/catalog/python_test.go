package catalog_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/brain/catalog"
	"github.com/xidus90/loomux/internal/config"
)

// Every expected value in this file was measured at the Python reference
// (Path.read_text and the root lines of brain.core.catalog, Python 3.14.7,
// ultra-brain tag loomux-1a-source) on 2026-09-15.

// writableArea writes index.md with the given bytes into a fresh area.
func writableArea(t *testing.T, index string) config.Area {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	return config.Area{Scope: "project/x", Path: dir}
}

func TestReadAreaCatalogFoldsNewlinesLikeReadText(t *testing.T) {
	a := writableArea(t, "# A\r\nb\rc\n")
	got, err := catalog.ReadAreaCatalog(a, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "# A\nb\nc\n" {
		t.Errorf("expected %q, got %q", "# A\nb\nc\n", got)
	}
}

func TestReadAreaCatalogRefusesInvalidUTF8(t *testing.T) {
	a := writableArea(t, "# A\n\xff\n")
	_, err := catalog.ReadAreaCatalog(a, "", "")
	want := filepath.Join(a.Path, "index.md") + ": not valid UTF-8"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestReadAreaCatalogKeepsTheByteOrderMark(t *testing.T) {
	a := writableArea(t, "\xef\xbb\xbf# A\n")
	got, err := catalog.ReadAreaCatalog(a, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "\xef\xbb\xbf# A\n" {
		t.Errorf("expected %q, got %q", "\xef\xbb\xbf# A\n", got)
	}
}

func TestReadAreaCatalogHandsOnAMissingIndex(t *testing.T) {
	_, err := catalog.ReadAreaCatalog(config.Area{Scope: "project/x", Path: t.TempDir()}, "", "")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected a not-exist error, got %v", err)
	}
}

func TestRenderRootCatalogSortsByCodePoint(t *testing.T) {
	areas := []config.Area{
		{Scope: "project/beta"},
		{Scope: "project/alpha"},
		{Scope: "core/hub"},
		{Scope: "Zeta"},
		{Scope: "\xc3\xa9t\xc3\xa9"},
	}
	want := "# brain\n\n* [Zeta](brain://Zeta/)\n* [core/hub](brain://core/hub/)\n* [project/alpha](brain://project/alpha/)\n* [project/beta](brain://project/beta/)\n* [\xc3\xa9t\xc3\xa9](brain://\xc3\xa9t\xc3\xa9/)\n"
	if got := catalog.RenderRootCatalog(areas); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
	if areas[0].Scope != "project/beta" || areas[4].Scope != "\xc3\xa9t\xc3\xa9" {
		t.Errorf("the caller's slice was reordered: %v", areas)
	}
}
