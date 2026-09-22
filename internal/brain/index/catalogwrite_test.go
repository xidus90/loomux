package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

func TestFormatDestination(t *testing.T) {
	if got := formatDestination("simple.md"); got != "simple.md" {
		t.Errorf("expected 'simple.md', got %q", got)
	}
	if got := formatDestination("00 Eingang/note.md"); got != "<00 Eingang/note.md>" {
		t.Errorf("expected wrapped in <...>, got %q", got)
	}
	if got := formatDestination("file (copy).md"); got != "<file (copy).md>" {
		t.Errorf("expected wrapped parens, got %q", got)
	}
	if got := formatDestination("file<1>.md"); got != "<file%3C1%3E.md>" {
		t.Errorf("expected encoded brackets, got %q", got)
	}
}

func TestEscapeLinkText(t *testing.T) {
	if got := escapeLinkText("Normal Title"); got != "Normal Title" {
		t.Errorf("expected 'Normal Title', got %q", got)
	}
	if got := escapeLinkText("[Topic] Architecture"); got != `\[Topic\] Architecture` {
		t.Errorf("expected escaped brackets, got %q", got)
	}
}

func TestRenderCatalog(t *testing.T) {
	entries := []Document{
		{Relative: "b.md", Title: "B Title", Description: "Desc B"},
		{Relative: "a.md", Title: "[A] Title", Description: ""},
	}
	subdirs := []string{"sub2", "01 sub1"}
	intro := "Introductory prose here.\n"

	rendered := RenderCatalog("project/area", entries, subdirs, intro)

	expected := "# project/area\n\n" +
		"Introductory prose here.\n\n" +
		"## Bereiche\n\n" +
		"* [01 sub1](<01 sub1/>)\n" +
		"* [sub2](sub2/)\n\n" +
		"## Dateien\n\n" +
		"* [\\[A\\] Title](a.md)\n" +
		"* [B Title](b.md) - Desc B\n\n" +
		CatalogRule + "\n"

	if rendered != expected {
		t.Fatalf("RenderCatalog mismatch:\ngot:\n%s\nwant:\n%s", rendered, expected)
	}
}

func TestReadIntro(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "missing.intro.md")

	// Missing intro returns empty string without error
	text, err := ReadIntro(missing)
	if err != nil {
		t.Fatalf("expected nil error on missing intro, got %v", err)
	}
	if text != "" {
		t.Errorf("expected empty text, got %q", text)
	}

	// Valid intro
	valid := filepath.Join(tmp, "index.intro.md")
	if err := os.WriteFile(valid, []byte("Welcome to the area.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	text, err = ReadIntro(valid)
	if err != nil {
		t.Fatalf("ReadIntro failed: %v", err)
	}
	if text != "Welcome to the area.\n" {
		t.Errorf("expected 'Welcome to the area.\\n', got %q", text)
	}

	// Empty intro must fail loudly
	empty := filepath.Join(tmp, "empty.intro.md")
	if err := os.WriteFile(empty, []byte("   \n\t\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = ReadIntro(empty)
	if err == nil {
		t.Fatal("expected error for empty intro file")
	}
	if !strings.Contains(err.Error(), "intro file is empty") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Reading a directory as intro returns error
	if _, err := ReadIntro(tmp); err == nil {
		t.Error("expected error reading directory as intro file")
	}
}

func TestWriteCatalogs(t *testing.T) {
	tmp := t.TempDir()
	srcRepo := filepath.Join(tmp, "repo")
	wikiDir := filepath.Join(srcRepo, "wiki")
	targetArtifacts := filepath.Join(tmp, "artifacts")

	if err := os.MkdirAll(wikiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetArtifacts, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create intro in sub directory
	subDir := filepath.Join(srcRepo, "topics")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "index.intro.md"), []byte("Topics Intro"), 0o644); err != nil {
		t.Fatal(err)
	}

	area := config.Area{
		Scope:    "project/sample",
		Path:     srcRepo,
		WikiPath: wikiDir,
		ReadOnly: false,
	}

	docs := []Document{
		{Relative: "root.md", Title: "Root Doc"},
		{Relative: "topics/arch.md", Title: "Architecture"},
		{Relative: "topics/deep/leaf.md", Title: "Leaf"},
		{Relative: "empty_ancestor/middle/deep.md", Title: "Deep Doc"}, // Tests intermediate directory creation
		{Relative: "wiki/page.md", Title: "Wiki Page"},                 // In own wiki
	}

	if err := WriteCatalogs(area, docs, targetArtifacts); err != nil {
		t.Fatalf("WriteCatalogs failed: %v", err)
	}

	// 1. Root index.md
	rootCatalogPath := filepath.Join(targetArtifacts, "index.md")
	rootContent, err := os.ReadFile(rootCatalogPath)
	if err != nil {
		t.Fatalf("missing root catalog: %v", err)
	}
	if !strings.Contains(string(rootContent), "# project/sample") {
		t.Errorf("expected root title project/sample, got:\n%s", rootContent)
	}
	if !strings.Contains(string(rootContent), "* [topics](topics/)") {
		t.Errorf("expected topics subdir link, got:\n%s", rootContent)
	}
	if !strings.Contains(string(rootContent), "* [empty_ancestor](empty_ancestor/)") {
		t.Errorf("expected empty_ancestor subdir link, got:\n%s", rootContent)
	}
	if !strings.Contains(string(rootContent), "* [Root Doc](root.md)") {
		t.Errorf("expected root.md doc link, got:\n%s", rootContent)
	}

	// 2. topics/index.md
	topicsCatalogPath := filepath.Join(targetArtifacts, "topics", "index.md")
	topicsContent, err := os.ReadFile(topicsCatalogPath)
	if err != nil {
		t.Fatalf("missing topics catalog: %v", err)
	}
	if !strings.Contains(string(topicsContent), "# topics") {
		t.Errorf("expected topics title, got:\n%s", topicsContent)
	}
	if !strings.Contains(string(topicsContent), "Topics Intro") {
		t.Errorf("expected Topics Intro, got:\n%s", topicsContent)
	}
	if !strings.Contains(string(topicsContent), "* [deep](deep/)") {
		t.Errorf("expected deep subdir link, got:\n%s", topicsContent)
	}
	if !strings.Contains(string(topicsContent), "* [Architecture](arch.md)") {
		t.Errorf("expected arch.md doc link, got:\n%s", topicsContent)
	}

	// 3. topics/deep/index.md
	deepCatalogPath := filepath.Join(targetArtifacts, "topics", "deep", "index.md")
	deepContent, err := os.ReadFile(deepCatalogPath)
	if err != nil {
		t.Fatalf("missing deep catalog: %v", err)
	}
	if !strings.Contains(string(deepContent), "* [Leaf](leaf.md)") {
		t.Errorf("expected leaf.md doc link, got:\n%s", deepContent)
	}

	// 4. wiki/index.md must NOT exist in artifacts because InOwnWiki skipped it
	wikiCatalogPath := filepath.Join(targetArtifacts, "wiki", "index.md")
	if _, err := os.Stat(wikiCatalogPath); !os.IsNotExist(err) {
		t.Error("expected wiki/index.md not to be generated in artifacts")
	}

	// 5. Test write_if_changed idempotency (mod time unchanged on rerun)
	infoBefore, _ := os.Stat(rootCatalogPath)
	if err := WriteCatalogs(area, docs, targetArtifacts); err != nil {
		t.Fatalf("rerun WriteCatalogs failed: %v", err)
	}
	infoAfter, _ := os.Stat(rootCatalogPath)
	if infoBefore.ModTime() != infoAfter.ModTime() {
		t.Error("expected mod time unchanged on identical WriteCatalogs rerun")
	}
}

func TestWriteCatalogsEmptyIntroFails(t *testing.T) {
	tmp := t.TempDir()
	srcRepo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(srcRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcRepo, "index.intro.md"), []byte("   "), 0o644); err != nil {
		t.Fatal(err)
	}

	area := config.Area{Path: srcRepo, Scope: "test"}
	err := WriteCatalogs(area, nil, filepath.Join(tmp, "out"))
	if err == nil {
		t.Fatal("expected error when intro file is empty")
	}
}

func TestWriteCatalogsWriteError(t *testing.T) {
	tmp := t.TempDir()
	// Using a file as target directory will cause MkdirAll / WriteFile to fail
	fileAsTarget := filepath.Join(tmp, "file_target")
	if err := os.WriteFile(fileAsTarget, []byte("blocking"), 0o644); err != nil {
		t.Fatal(err)
	}

	area := config.Area{Path: tmp, Scope: "test"}
	docs := []Document{{Relative: "sub/doc.md", Title: "Doc"}}
	err := WriteCatalogs(area, docs, fileAsTarget)
	if err == nil {
		t.Fatal("expected error when writing catalog into invalid target directory")
	}
}
