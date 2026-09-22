package index

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseFrontmatterValid(t *testing.T) {
	content := `---
title: "Sample Note"
description: "A note for testing"
type: Topic
tags:
  - architecture
  - go
---
# Sample Note Body
Some text here.
`
	meta, err := ParseFrontmatter(content, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta["title"] != "Sample Note" {
		t.Errorf("expected title 'Sample Note', got %v", meta["title"])
	}
	if meta["description"] != "A note for testing" {
		t.Errorf("expected description 'A note for testing', got %v", meta["description"])
	}
	if meta["type"] != "Topic" {
		t.Errorf("expected type 'Topic', got %v", meta["type"])
	}
	tags, ok := meta["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", meta["tags"])
	}
}

func TestParseFrontmatterMissing(t *testing.T) {
	content := `# Just a Markdown Note
No frontmatter at all.
`
	meta, err := ParseFrontmatter(content, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(meta) != 0 {
		t.Errorf("expected empty map, got %v", meta)
	}

	metaStrict, err := ParseFrontmatter(content, true)
	if err != nil {
		t.Fatalf("unexpected error in strict mode: %v", err)
	}
	if len(metaStrict) != 0 {
		t.Errorf("expected empty map, got %v", metaStrict)
	}
}

func TestParseFrontmatterBroken(t *testing.T) {
	broken := `---
title: [unclosed list
---
Content
`
	// Non-strict swallows YAML errors
	meta, err := ParseFrontmatter(broken, false)
	if err != nil {
		t.Fatalf("expected non-strict mode to swallow error, got: %v", err)
	}
	if len(meta) != 0 {
		t.Errorf("expected empty map on broken yaml, got %v", meta)
	}

	// Strict returns an error
	_, err = ParseFrontmatter(broken, true)
	if err == nil {
		t.Fatal("expected strict mode to return error for broken yaml")
	}
}

func TestParseFrontmatterNotMapping(t *testing.T) {
	notMap := `---
- item 1
- item 2
---
Content
`
	meta, err := ParseFrontmatter(notMap, false)
	if err != nil {
		t.Fatalf("expected non-strict mode to swallow non-mapping, got: %v", err)
	}
	if len(meta) != 0 {
		t.Errorf("expected empty map for non-mapping yaml, got %v", meta)
	}

	_, err = ParseFrontmatter(notMap, true)
	if err == nil {
		t.Fatal("expected strict mode to return error for non-mapping frontmatter")
	}
}

func TestReadDocument(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "subdir", "sample.md")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}

	content := `---
title: "Custom Title"
description: "My Description"
type: "Source"
tags:
  - alpha
  - beta
---
Here is a [markdown link](target.md) and an image ![ignore me](image.png).
Also a [[wikilink#section|alias]] and an image ![[ignore_wiki.png]].
Another [link without spaces](path/to/another.md).
Duplicate [link](target.md).
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := ReadDocument(filePath, tmp)
	if err != nil {
		t.Fatalf("ReadDocument failed: %v", err)
	}

	if doc.Relative != "subdir/sample.md" {
		t.Errorf("expected relative 'subdir/sample.md', got %q", doc.Relative)
	}
	if doc.Title != "Custom Title" {
		t.Errorf("expected title 'Custom Title', got %q", doc.Title)
	}
	if doc.Description != "My Description" {
		t.Errorf("expected description 'My Description', got %q", doc.Description)
	}
	if doc.DocType != "Source" {
		t.Errorf("expected doc_type 'Source', got %q", doc.DocType)
	}
	expectedTags := []string{"alpha", "beta"}
	if !reflect.DeepEqual(doc.Tags, expectedTags) {
		t.Errorf("expected tags %v, got %v", expectedTags, doc.Tags)
	}

	expectedLinks := []string{"path/to/another.md", "target.md", "wikilink"}
	if !reflect.DeepEqual(doc.Links, expectedLinks) {
		t.Errorf("expected links %v, got %v", expectedLinks, doc.Links)
	}
}

func TestReadDocumentFallbackTitle(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "note-one.md")

	content := `# Heading Title
Some content without frontmatter.
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := ReadDocument(filePath, tmp)
	if err != nil {
		t.Fatalf("ReadDocument failed: %v", err)
	}

	// Python behavior: meta.get("title") or path.stem ("note-one")
	if doc.Title != "note-one" {
		t.Errorf("expected title 'note-one', got %q", doc.Title)
	}
	if doc.Relative != "note-one.md" {
		t.Errorf("expected relative 'note-one.md', got %q", doc.Relative)
	}
	if len(doc.Tags) != 0 {
		t.Errorf("expected empty tags, got %v", doc.Tags)
	}
	if len(doc.Links) != 0 {
		t.Errorf("expected empty links, got %v", doc.Links)
	}
}

func TestReadDocumentMissingFile(t *testing.T) {
	_, err := ReadDocument("/nonexistent/file.md", "/nonexistent")
	if err == nil {
		t.Fatal("expected error reading nonexistent file")
	}
}

func TestReadDocumentRelError(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "doc.md")
	if err := os.WriteFile(filePath, []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// On Windows, paths on different drives cannot be made relative to each other.
	_, err := ReadDocument(filePath, "Z:\\different\\drive")
	if err == nil {
		t.Fatal("expected error making path relative to different drive")
	}
}
