package wiki

import (
	"os"
	"path/filepath"
	"testing"
)

// A `.loomux/config.toml` that holds only policy declares no types: it reads
// as no declaration, not as an error.
func TestDeclaredTypesInTakesAPolicyOnlyConfigAsNoDeclaration(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := DeclaredTypesIn(dir)
	if err != nil || got != nil {
		t.Fatalf("got %v, %v", got, err)
	}
}
