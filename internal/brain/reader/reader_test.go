package reader_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/reader"
	"github.com/xidus90/loomux/internal/config"
)

func TestExtractSection(t *testing.T) {
	doc := `# Title

Introduction paragraph.

## First

Content of first.

### Subsection

Deep content.

## Second

Content of second.

# Epilogue

Ending.
`

	// 1. Extract ## First (includes ### Subsection, stops at ## Second)
	s1, err := reader.ExtractSection(doc, "First")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected1 := "## First\n\nContent of first.\n\n### Subsection\n\nDeep content.\n"
	if s1 != expected1 {
		t.Errorf("expected %q, got %q", expected1, s1)
	}

	// 2. Extract ## Second (stops at # Epilogue because level 1 <= level 2)
	s2, err := reader.ExtractSection(doc, "Second")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected2 := "## Second\n\nContent of second.\n"
	if s2 != expected2 {
		t.Errorf("expected %q, got %q", expected2, s2)
	}

	// 3. Extract # Epilogue (runs until EOF)
	s3, err := reader.ExtractSection(doc, "Epilogue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected3 := "# Epilogue\n\nEnding.\n"
	if s3 != expected3 {
		t.Errorf("expected %q, got %q", expected3, s3)
	}

	// 4. Missing section
	_, err = reader.ExtractSection(doc, "NonExistent")
	if err == nil {
		t.Fatal("expected error for missing section, got nil")
	}
	if !strings.Contains(err.Error(), "no section titled 'NonExistent'") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReadDocument(t *testing.T) {
	tmp := t.TempDir()
	areaDir := filepath.Join(tmp, "area")
	if err := os.MkdirAll(filepath.Join(areaDir, "review", "cases"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(areaDir, "secrets"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Write normal file
	content := "# Doc\n\n## Sec\n\nSection text.\n"
	if err := os.WriteFile(filepath.Join(areaDir, "doc.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	// Write review centre file
	reviewContent := "# Review Case\n"
	if err := os.WriteFile(filepath.Join(areaDir, "review", "cases", "c1.md"), []byte(reviewContent), 0o644); err != nil {
		t.Fatal(err)
	}
	// Write secret file
	if err := os.WriteFile(filepath.Join(areaDir, "secrets", "key.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	area := config.Area{
		Scope: "project/test",
		Path:  areaDir,
	}
	manifest := &config.Manifest{
		LayoutReview: "review",
		NeverGlobs:   []string{"secrets/**"},
	}

	// 1. Plain read
	got, err := reader.ReadDocument(area, manifest, "doc.md", "", privacy.ChannelLocal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != content {
		t.Errorf("expected %q, got %q", content, got)
	}

	// 2. Read section
	got, err = reader.ReadDocument(area, manifest, "doc.md", "Sec", privacy.ChannelLocal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedSec := "## Sec\n\nSection text.\n"
	if got != expectedSec {
		t.Errorf("expected %q, got %q", expectedSec, got)
	}

	// 3. Path leaves area
	_, err = reader.ReadDocument(area, manifest, "../escape.md", "", privacy.ChannelLocal)
	if err == nil || !strings.Contains(err.Error(), "leaves the area") {
		t.Errorf("expected 'leaves the area' error, got: %v", err)
	}

	// 4. Privacy never exclusion
	_, err = reader.ReadDocument(area, manifest, "secrets/key.txt", "", privacy.ChannelLocal)
	if err == nil || !strings.Contains(err.Error(), "is excluded by [privacy] never") {
		t.Errorf("expected 'excluded by [privacy] never' error, got: %v", err)
	}

	// 5. Cloud channel refuses review centre
	_, err = reader.ReadDocument(area, manifest, "review/cases/c1.md", "", privacy.ChannelCloud)
	if err == nil || !strings.Contains(err.Error(), "is the review centre; refused on the cloud channel") {
		t.Errorf("expected cloud review centre refusal, got: %v", err)
	}

	// 6. Local channel allows review centre
	got, err = reader.ReadDocument(area, manifest, "review/cases/c1.md", "", privacy.ChannelLocal)
	if err != nil {
		t.Fatalf("unexpected error for local review read: %v", err)
	}
	if got != reviewContent {
		t.Errorf("expected %q, got %q", reviewContent, got)
	}

	// 7. Missing file
	_, err = reader.ReadDocument(area, manifest, "missing.md", "", privacy.ChannelLocal)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}

	// 8. Invalid LayoutReview error
	badManifest := &config.Manifest{LayoutReview: "."}
	_, err = reader.ReadDocument(area, badManifest, "doc.md", "", privacy.ChannelCloud)
	if err == nil {
		t.Fatal("expected error for bad LayoutReview on cloud channel, got nil")
	}
}
