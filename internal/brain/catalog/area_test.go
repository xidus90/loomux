package catalog_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/brain/catalog"
	"github.com/xidus90/loomux/internal/config"
)

func TestReadAreaCatalog(t *testing.T) {
	tmp := t.TempDir()
	areaPath := filepath.Join(tmp, "writable")
	if err := os.MkdirAll(areaPath, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "# Writable Catalog\n\n* [Doc](doc.md)\n"
	if err := os.WriteFile(filepath.Join(areaPath, "index.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	writableArea := config.Area{
		Scope:    "project/writable",
		Path:     areaPath,
		ReadOnly: false,
	}

	got, err := catalog.ReadAreaCatalog(writableArea, tmp, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != content {
		t.Errorf("expected %q, got %q", content, got)
	}

	// Readonly area
	stateDir := filepath.Join(tmp, "state")
	roAreaDir := filepath.Join(stateDir, "areas", "project-readonly")
	if err := os.MkdirAll(roAreaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	roContent := "# Readonly Catalog\n"
	if err := os.WriteFile(filepath.Join(roAreaDir, "index.md"), []byte(roContent), 0o644); err != nil {
		t.Fatal(err)
	}

	readonlyArea := config.Area{
		Scope:    "project/readonly",
		Path:     filepath.Join(tmp, "sources"),
		ReadOnly: true,
	}

	got, err = catalog.ReadAreaCatalog(readonlyArea, stateDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != roContent {
		t.Errorf("expected %q, got %q", roContent, got)
	}

	// Missing index.md
	missingArea := config.Area{
		Scope:    "project/missing",
		Path:     filepath.Join(tmp, "nonexistent"),
		ReadOnly: false,
	}
	_, err = catalog.ReadAreaCatalog(missingArea, stateDir, "")
	if err == nil {
		t.Fatal("expected error for missing index.md, got nil")
	}
}
