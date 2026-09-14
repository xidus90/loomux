package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckGoFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Formatted file
	cleanFile := filepath.Join(tmpDir, "clean.go")
	if err := os.WriteFile(cleanFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Unformatted file
	dirtyFile := filepath.Join(tmpDir, "dirty.go")
	if err := os.WriteFile(dirtyFile, []byte("package main\nfunc  main( ) {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Non-go file
	otherFile := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(otherFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// 4. Ignored directories (vendor, .git, node_modules)
	for _, ign := range []string{"vendor", ".git", "node_modules"} {
		dir := filepath.Join(tmpDir, ign)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "bad.go"), []byte("package v\nfunc  bad( ) {}\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	unformatted, err := CheckGoFormat([]string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(unformatted) != 1 {
		t.Fatalf("expected exactly 1 unformatted file, got %d: %v", len(unformatted), unformatted)
	}

	if filepath.Base(unformatted[0]) != "dirty.go" {
		t.Fatalf("expected dirty.go, got: %v", unformatted[0])
	}

	// Test non-existent path gracefully skipped
	unformatted2, err := CheckGoFormat([]string{filepath.Join(tmpDir, "nonexistent")})
	if err != nil {
		t.Fatalf("unexpected error on missing path: %v", err)
	}
	if len(unformatted2) != 0 {
		t.Fatalf("expected 0 files, got %v", unformatted2)
	}

	// Test single clean file and other non-go files
	cleanSingle, err := CheckGoFormat([]string{cleanFile, otherFile})
	if err != nil {
		t.Fatalf("unexpected error on clean single file: %v", err)
	}
	if len(cleanSingle) != 0 {
		t.Fatalf("expected 0 unformatted for clean file, got %v", cleanSingle)
	}

	// Test single dirty file input
	unformattedSingle, err := CheckGoFormat([]string{dirtyFile})
	if err != nil {
		t.Fatalf("unexpected error on single file: %v", err)
	}
	if len(unformattedSingle) != 1 {
		t.Fatalf("expected 1 file for single dirty file, got %v", unformattedSingle)
	}

	// Test single broken go file
	syntaxErrFile := filepath.Join(tmpDir, "broken.go")
	if err := os.WriteFile(syntaxErrFile, []byte("package main\nfunc broken( {"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckGoFormat([]string{syntaxErrFile}); err == nil {
		t.Fatal("expected parse error for single broken.go, got nil")
	}

	// Test broken file inside dir walk
	if _, err := CheckGoFormat([]string{tmpDir}); err == nil {
		t.Fatal("expected walk error on broken.go in dir, got nil")
	}

	// Test invalid path with null byte
	if _, err := CheckGoFormat([]string{"invalid\x00path"}); err == nil {
		t.Fatal("expected stat error for invalid path, got nil")
	}

	// Test isFileUnformatted on directory (read error)
	if _, err := isFileUnformatted(tmpDir); err == nil {
		t.Fatal("expected read error on directory, got nil")
	}
}
