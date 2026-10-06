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

func writeTestRegistry(t *testing.T, dir string, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "registry.toml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func setupTestArea(t *testing.T, areaDir string, manifestContent string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(areaDir, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(areaDir, ".loomux", "config.toml")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReindexFullWorkflow(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", tmp)

	areaDir := filepath.Join(tmp, "myarea")
	setupTestArea(t, areaDir, "[area]\nscope = \"my-area\"\n[index]\ninclude = [\"**/*.md\"]\n")

	// Create intro file and notes
	if err := os.WriteFile(filepath.Join(areaDir, "index.intro.md"), []byte("Intro prose"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc1 := "---\ntitle: Doc One\ntags: [tag1]\n---\n# Doc One\nContent with link [Doc Two](doc2.md)\n"
	doc2 := "---\ntitle: Doc Two\ntags: [tag2]\n---\n# Doc Two\nContent without link\n"
	if err := os.WriteFile(filepath.Join(areaDir, "doc1.md"), []byte(doc1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(areaDir, "doc2.md"), []byte(doc2), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create subfolder with doc
	subDir := filepath.Join(areaDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc3 := "---\ntitle: Doc Three\n---\n# Doc Three\nSubfolder doc\n"
	if err := os.WriteFile(filepath.Join(subDir, "doc3.md"), []byte(doc3), 0o644); err != nil {
		t.Fatal(err)
	}

	registryContent := "[[area]]\nscope = \"my-area\"\npath = \"" + filepath.ToSlash(areaDir) + "\"\n"
	regPath := writeTestRegistry(t, stateDir, registryContent)

	port := search.NewFakePort()
	var stderr bytes.Buffer

	code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
	if err != nil {
		t.Fatalf("Reindex failed: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	// Check catalogs generated
	rootCatalog, err := os.ReadFile(filepath.Join(areaDir, "index.md"))
	if err != nil {
		t.Fatalf("missing root index.md: %v", err)
	}
	if !strings.Contains(string(rootCatalog), "Intro prose") {
		t.Errorf("catalog missing intro prose: %s", rootCatalog)
	}
	if !strings.Contains(string(rootCatalog), "[Doc One](doc1.md)") {
		t.Errorf("catalog missing Doc One link: %s", rootCatalog)
	}

	subCatalog, err := os.ReadFile(filepath.Join(subDir, "index.md"))
	if err != nil {
		t.Fatalf("missing sub index.md: %v", err)
	}
	if !strings.Contains(string(subCatalog), "[Doc Three](doc3.md)") {
		t.Errorf("sub catalog missing Doc Three link: %s", subCatalog)
	}

	// Check graph.json
	graphData, err := os.ReadFile(filepath.Join(areaDir, "graph.json"))
	if err != nil {
		t.Fatalf("missing graph.json: %v", err)
	}
	if !strings.Contains(string(graphData), `"scope": "my-area"`) {
		t.Errorf("graph.json missing scope: %s", graphData)
	}

	// Check _identities.tsv
	idData, err := os.ReadFile(filepath.Join(areaDir, "_identities.tsv"))
	if err != nil {
		t.Fatalf("missing _identities.tsv: %v", err)
	}
	if !strings.Contains(string(idData), "doc1.md") {
		t.Errorf("_identities.tsv missing doc1.md: %s", idData)
	}

	// Check port refreshed
	if len(port.Refreshed) != 1 || len(port.Refreshed[0]) != 1 || port.Refreshed[0][0] != "my-area" {
		t.Errorf("expected port refresh on my-area, got %#v", port.Refreshed)
	}

	// Check stderr output reported updated collections
	if !strings.Contains(stderr.String(), "updated qmd collections: my-area") {
		t.Errorf("expected updated qmd collections in stderr, got: %s", stderr.String())
	}

	// Second run: no file changed, revision stays same, port refreshed
	stderr.Reset()
	code, err = ReindexWithOutput(regPath, stateDir, port, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("second Reindex failed: %v, code %d", err, code)
	}
	if strings.Contains(stderr.String(), "updated qmd collections") {
		t.Errorf("second run should not update qmd collections: %s", stderr.String())
	}

	// Modify a document: content hash moves, revision advances to 2
	modifiedDoc1 := doc1 + "\nAppended text\n"
	if err := os.WriteFile(filepath.Join(areaDir, "doc1.md"), []byte(modifiedDoc1), 0o644); err != nil {
		t.Fatal(err)
	}

	code, err = ReindexWithOutput(regPath, stateDir, port, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("reindex after edit failed: %v, code %d", err, code)
	}
	newIds, err := identity.ReadIdentities(filepath.Join(areaDir, "_identities.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if newIds["doc1.md"].Revision != 2 {
		t.Errorf("expected doc1.md revision 2, got %d", newIds["doc1.md"].Revision)
	}

	// Rename doc2.md to doc2_renamed.md: identity matches and doc_id survives
	oldDoc2ID := newIds["doc2.md"].DocID
	if err := os.Rename(filepath.Join(areaDir, "doc2.md"), filepath.Join(areaDir, "doc2_renamed.md")); err != nil {
		t.Fatal(err)
	}
	code, err = ReindexWithOutput(regPath, stateDir, port, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("reindex after rename failed: %v, code %d", err, code)
	}
	renamedIds, err := identity.ReadIdentities(filepath.Join(areaDir, "_identities.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if renamedIds["doc2_renamed.md"].DocID != oldDoc2ID {
		t.Errorf("expected doc_id preserved across rename: %s vs %s", renamedIds["doc2_renamed.md"].DocID, oldDoc2ID)
	}
}

func TestReindexReadOnlyArea(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)

	roAreaDir := filepath.Join(tmp, "ro_area")
	if err := os.MkdirAll(roAreaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(roAreaDir, "note.md"), []byte("# Note\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Read-only area manifest sits in stateDir/areas/ro-scope
	roArtifactDir := filepath.Join(stateDir, "areas", "ro-scope")
	setupTestArea(t, roArtifactDir, "[area]\nscope = \"ro-scope\"\n")

	registryContent := "[[area]]\nscope = \"ro-scope\"\npath = \"" + filepath.ToSlash(roAreaDir) + "\"\nreadonly = true\n"
	regPath := writeTestRegistry(t, stateDir, registryContent)

	port := search.NewFakePort()
	var stderr bytes.Buffer

	code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("Reindex read-only area failed: %v, code %d", err, code)
	}

	// Catalogs, graph, identities are in roArtifactDir, NOT in roAreaDir
	if _, err := os.Stat(filepath.Join(roArtifactDir, "index.md")); err != nil {
		t.Errorf("expected index.md in state artifact dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(roAreaDir, "index.md")); !os.IsNotExist(err) {
		t.Errorf("read-only area directory should not have index.md")
	}
}

func TestReindexSkippingMissingAreaAndManifest(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)

	missingAreaPath := filepath.Join(tmp, "does_not_exist")
	noManifestAreaDir := filepath.Join(tmp, "no_manifest")
	if err := os.MkdirAll(noManifestAreaDir, 0o755); err != nil {
		t.Fatal(err)
	}

	validAreaDir := filepath.Join(tmp, "valid_area")
	setupTestArea(t, validAreaDir, "[area]\nscope = \"valid\"\n")
	if err := os.WriteFile(filepath.Join(validAreaDir, "valid.md"), []byte("# Valid\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	registryContent := "[[area]]\nscope = \"missing-dir\"\npath = \"" + filepath.ToSlash(missingAreaPath) + "\"\n\n" +
		"[[area]]\nscope = \"missing-manifest\"\npath = \"" + filepath.ToSlash(noManifestAreaDir) + "\"\n\n" +
		"[[area]]\nscope = \"valid\"\npath = \"" + filepath.ToSlash(validAreaDir) + "\"\n"
	regPath := writeTestRegistry(t, stateDir, registryContent)

	port := search.NewFakePort()
	var stderr bytes.Buffer

	code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
	if err != nil {
		t.Fatalf("Reindex failed: %v", err)
	}
	if code != 0 {
		t.Errorf("expected code 0, got %d", code)
	}

	out := stderr.String()
	if !strings.Contains(out, "skipping missing-dir:") || !strings.Contains(out, "does_not_exist does not exist") {
		t.Errorf("expected missing-dir skip, got: %s", out)
	}
	if !strings.Contains(out, "skipping missing-manifest") {
		t.Errorf("expected missing-manifest skip, got: %s", out)
	}
	// Valid area was still indexed
	if !strings.Contains(out, "updated qmd collections: valid") {
		t.Errorf("expected valid area to be updated, got: %s", out)
	}
}

// A path the system cannot inspect (here one with a NUL byte, which both
// platforms refuse) is not an absent one: it is named as such, the other
// areas are still indexed, and the run ends with exit 1.
func TestReindexAreaPathThatCannotBeInspectedFailsTheRun(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("LOOMUX_STATE_DIR", stateDir)

	validAreaDir := filepath.Join(tmp, "valid_area")
	setupTestArea(t, validAreaDir, "[area]\nscope = \"valid\"\n")
	if err := os.WriteFile(filepath.Join(validAreaDir, "valid.md"), []byte("# Valid\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	registryContent := "[[area]]\nscope = \"broken\"\npath = \"" + filepath.ToSlash(tmp) + "/bad\\u0000dir\"\n\n" +
		"[[area]]\nscope = \"valid\"\npath = \"" + filepath.ToSlash(validAreaDir) + "\"\n"
	regPath := writeTestRegistry(t, stateDir, registryContent)

	var stderr bytes.Buffer
	code, err := ReindexWithOutput(regPath, stateDir, search.NewFakePort(), &stderr)
	if err != nil {
		t.Fatalf("Reindex failed: %v", err)
	}
	out := stderr.String()
	if code != 1 {
		t.Errorf("expected code 1, got %d; stderr: %s", code, out)
	}
	if strings.Contains(out, "does not exist") {
		t.Errorf("an uninspectable path must not be called absent, got: %s", out)
	}
	if !strings.Contains(out, "skipping broken: ") || !strings.Contains(out, " cannot be inspected: ") {
		t.Errorf("expected the cannot-be-inspected skip, got: %s", out)
	}
	if !strings.Contains(out, "updated qmd collections: valid") {
		t.Errorf("expected the other area to be indexed, got: %s", out)
	}
}

// A declaration that is there and does not read is no absence like a missing
// one: its area is skipped, the others run, and the run fails.
func TestReindexDeclarationThatDoesNotReadFailsTheRun(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("LOOMUX_STATE_DIR", stateDir)

	brokenDir := filepath.Join(tmp, "broken_area")
	setupTestArea(t, brokenDir, "[area\n")
	validDir := filepath.Join(tmp, "valid_area")
	setupTestArea(t, validDir, "[area]\nscope = \"valid\"\n")
	if err := os.WriteFile(filepath.Join(validDir, "valid.md"), []byte("# Valid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	regPath := writeTestRegistry(t, stateDir,
		"[[area]]\nscope = \"broken\"\npath = \""+filepath.ToSlash(brokenDir)+"\"\n\n"+
			"[[area]]\nscope = \"valid\"\npath = \""+filepath.ToSlash(validDir)+"\"\n")

	var stderr bytes.Buffer
	code, err := ReindexWithOutput(regPath, stateDir, search.NewFakePort(), &stderr)
	out := stderr.String()
	if err != nil || code != 1 {
		t.Fatalf("got %d, %v; want exit 1 without an error; stderr: %s", code, err, out)
	}
	if !strings.Contains(out, "skipping broken: ") || !strings.Contains(out, "not valid TOML") {
		t.Errorf("expected the skip naming the parse error, got: %s", out)
	}
	if !strings.Contains(out, "updated qmd collections: valid") {
		t.Errorf("expected the other area to be indexed, got: %s", out)
	}
}

func TestReindexCollisionRefused(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)

	areaDir := filepath.Join(tmp, "area")
	setupTestArea(t, areaDir, "[area]\nscope = \"colliding\"\n")
	_ = os.WriteFile(filepath.Join(areaDir, "a.md"), []byte("# A\n"), 0o644)

	regPath := writeTestRegistry(t, stateDir, "[[area]]\nscope = \"colliding\"\npath = \""+filepath.ToSlash(areaDir)+"\"\n")

	// Write qmd config with existing unowned colliding collection
	qmdConfigDir := filepath.Join(tmp, "qmd")
	_ = os.MkdirAll(qmdConfigDir, 0o755)
	_ = os.WriteFile(filepath.Join(qmdConfigDir, "index.yml"), []byte("collections:\n  colliding:\n    path: C:/someone/colliding\n"), 0o644)

	port := search.NewFakePort()
	var stderr bytes.Buffer

	code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 1 {
		t.Errorf("expected code 1 on collision refusal, got %d", code)
	}
	if !strings.Contains(stderr.String(), "error: colliding already exists in the search engine") {
		t.Errorf("expected refusal message in stderr, got: %s", stderr.String())
	}
}

func TestReindexSearchPortError(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)

	areaDir := filepath.Join(tmp, "area")
	setupTestArea(t, areaDir, "[area]\nscope = \"p-area\"\n")
	_ = os.WriteFile(filepath.Join(areaDir, "a.md"), []byte("# A\n"), 0o644)

	regPath := writeTestRegistry(t, stateDir, "[[area]]\nscope = \"p-area\"\npath = \""+filepath.ToSlash(areaDir)+"\"\n")

	port := search.NewFakePort()
	port.Refreshes = []error{errors.New("connection failed")}
	var stderr bytes.Buffer

	code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
	if err == nil {
		t.Fatal("expected error on port refresh failure")
	}
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "error: connection failed") {
		t.Errorf("expected port error in stderr, got: %s", stderr.String())
	}
}

func TestReindexRegistryError(t *testing.T) {
	tmp := t.TempDir()
	port := search.NewFakePort()
	var stderr bytes.Buffer

	// Invalid registry path
	code, err := ReindexWithOutput(filepath.Join(tmp, "nonexistent_reg.toml"), tmp, port, &stderr)
	if err == nil {
		t.Fatal("expected error on non-existent registry")
	}
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

// An empty registry path is the state directory's registry.toml. Nothing is
// read from the environment for it: the state directory is the argument, and
// the run that gets none reads no registry of another test.
func TestReindexTakesTheRegistryFromTheStateDirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	writeTestRegistry(t, tmp, "")
	port := search.NewFakePort()

	code, err := Reindex("", tmp, port)
	if err != nil || code != 0 {
		t.Errorf("expected Reindex with an empty registry path to succeed: %v, code %d", err, code)
	}
}

func TestEmbedWorkflow(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")

	regContent := "[[area]]\nscope = \"b-area\"\npath = \"/b\"\n\n[[area]]\nscope = \"a-area\"\npath = \"/a\"\n"
	regPath := writeTestRegistry(t, stateDir, regContent)

	port := search.NewFakePort()
	var stderr bytes.Buffer

	code, err := EmbedWithOutput(regPath, "", port, &stderr)
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	// Scopes sorted
	if len(port.Embedded) != 1 || len(port.Embedded[0]) != 2 {
		t.Fatalf("expected 1 embed call with 2 scopes, got %#v", port.Embedded)
	}
	if port.Embedded[0][0] != "a-area" || port.Embedded[0][1] != "b-area" {
		t.Errorf("expected sorted scopes [a-area, b-area], got %#v", port.Embedded[0])
	}
	if !strings.Contains(stderr.String(), "embedded 2 area(s)") {
		t.Errorf("expected 'embedded 2 area(s)' in stderr, got: %s", stderr.String())
	}

	t.Run("embed port error", func(t *testing.T) {
		failPort := &failingEmbedPort{err: errors.New("embed failure")}
		var errBuf bytes.Buffer
		c, err := EmbedWithOutput(regPath, "", failPort, &errBuf)
		if err == nil || c != 1 {
			t.Errorf("expected failure on port embed error, got code %d, err %v", c, err)
		}
		if !strings.Contains(errBuf.String(), "error: embed failure") {
			t.Errorf("expected error message in stderr, got: %s", errBuf.String())
		}
	})

	t.Run("embed registry read error", func(t *testing.T) {
		var errBuf bytes.Buffer
		c, err := EmbedWithOutput(filepath.Join(tmp, "gone.toml"), "", port, &errBuf)
		if err == nil || c != 1 {
			t.Errorf("expected failure on missing registry, got code %d, err %v", c, err)
		}
	})

	t.Run("embed default registry", func(t *testing.T) {
		c, err := Embed("", stateDir, port)
		if err != nil || c != 0 {
			t.Errorf("expected Embed with empty registry to succeed: %v, code %d", err, c)
		}
	})
}

type failingEmbedPort struct {
	search.SearchPort
	err error
}

func (f *failingEmbedPort) Embed(collections []string) error {
	return f.err
}

func TestReindexNestedAreas(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)

	parentDir := filepath.Join(tmp, "parent")
	childDir := filepath.Join(parentDir, "subarea")
	childWiki := filepath.Join(childDir, "wiki")
	setupTestArea(t, parentDir, "[area]\nscope = \"parent\"\n")
	setupTestArea(t, childDir, "[area]\nscope = \"child\"\n")

	_ = os.WriteFile(filepath.Join(parentDir, "parent.md"), []byte("# Parent\n"), 0o644)
	_ = os.WriteFile(filepath.Join(childDir, "child.md"), []byte("# Child\n"), 0o644)

	regContent := "[[area]]\nscope = \"parent\"\npath = \"" + filepath.ToSlash(parentDir) + "\"\n\n" +
		"[[area]]\nscope = \"child\"\npath = \"" + filepath.ToSlash(childDir) + "\"\nwiki = \"" + filepath.ToSlash(childWiki) + "\"\n"
	regPath := writeTestRegistry(t, stateDir, regContent)

	port := search.NewFakePort()
	code, err := Reindex(regPath, stateDir, port)
	if err != nil || code != 0 {
		t.Fatalf("reindex nested areas failed: %v, code %d", err, code)
	}
}

func TestReindexNilStderrAndEmptyRegistryPath(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)

	areaDir := filepath.Join(tmp, "area")
	setupTestArea(t, areaDir, "[area]\nscope = \"my-scope\"\n")
	_ = os.WriteFile(filepath.Join(areaDir, "a.md"), []byte("# A\n"), 0o644)

	_ = writeTestRegistry(t, stateDir, "[[area]]\nscope = \"my-scope\"\npath = \""+filepath.ToSlash(areaDir)+"\"\n")

	port := search.NewFakePort()
	// Pass empty registryPath and nil stderr
	code, err := ReindexWithOutput("", stateDir, port, nil)
	if err != nil || code != 0 {
		t.Errorf("expected success with empty registryPath and nil stderr: %v, code %d", err, code)
	}
}

func TestReindexDroppedCollectionsOutput(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)

	areaDir := filepath.Join(tmp, "area")
	setupTestArea(t, areaDir, "[area]\nscope = \"my-scope\"\n")
	_ = os.WriteFile(filepath.Join(areaDir, "a.md"), []byte("# A\n"), 0o644)

	regPath := writeTestRegistry(t, stateDir, "[[area]]\nscope = \"my-scope\"\npath = \""+filepath.ToSlash(areaDir)+"\"\n")

	// Pre-create owned file containing an abandoned collection
	qmdConfigDir := filepath.Join(tmp, "qmd")
	_ = os.MkdirAll(qmdConfigDir, 0o755)
	_ = os.WriteFile(filepath.Join(qmdConfigDir, "index.yml"), []byte("collections:\n  my-scope: {}\n  abandoned: {}\n"), 0o644)
	_ = os.WriteFile(filepath.Join(stateDir, "qmd-collections.json"), []byte(`["my-scope", "abandoned"]`), 0o644)

	port := search.NewFakePort()
	var stderr bytes.Buffer
	code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("reindex failed: %v, code %d", err, code)
	}
	if !strings.Contains(stderr.String(), "dropped qmd collections: abandoned") {
		t.Errorf("expected dropped collections in stderr, got: %s", stderr.String())
	}
}

func TestReindexErrorBranches(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)
	port := search.NewFakePort()

	t.Run("FindFiles error on root review layout", func(t *testing.T) {
		areaDir := filepath.Join(tmp, "bad_area")
		setupTestArea(t, areaDir, "[area]\nscope = \"bad\"\n[layout]\nreview = \".\"\n")
		regPath := writeTestRegistry(t, stateDir, "[[area]]\nscope = \"bad\"\npath = \""+filepath.ToSlash(areaDir)+"\"\n")
		var stderr bytes.Buffer
		code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
		if err == nil || code != 1 {
			t.Errorf("expected error on root review layout, got %v, code %d", err, code)
		}
	})

	t.Run("WriteCatalogs error on blocked catalog file", func(t *testing.T) {
		areaDir := filepath.Join(tmp, "blocked_area")
		setupTestArea(t, areaDir, "[area]\nscope = \"blocked\"\n")
		_ = os.WriteFile(filepath.Join(areaDir, "note.md"), []byte("# Note\n"), 0o644)
		// Make index.md an existing non-empty directory so writing catalog fails
		catalogDir := filepath.Join(areaDir, "index.md")
		_ = os.MkdirAll(catalogDir, 0o755)
		_ = os.WriteFile(filepath.Join(catalogDir, "child"), []byte("data"), 0o644)

		regPath := writeTestRegistry(t, stateDir, "[[area]]\nscope = \"blocked\"\npath = \""+filepath.ToSlash(areaDir)+"\"\n")
		var stderr bytes.Buffer
		code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
		if err == nil || code != 1 {
			t.Errorf("expected error on blocked catalog target, got %v, code %d", err, code)
		}
	})

	t.Run("SyncCollections error", func(t *testing.T) {
		areaDir := filepath.Join(tmp, "sync_err_area")
		setupTestArea(t, areaDir, "[area]\nscope = \"sync-err\"\n")
		_ = os.WriteFile(filepath.Join(areaDir, "note.md"), []byte("# Note\n"), 0o644)

		// Make qmd index.yml a directory to trigger sync error
		badXDG := filepath.Join(tmp, "bad_xdg")
		t.Setenv("XDG_CONFIG_HOME", badXDG)
		_ = os.MkdirAll(filepath.Join(badXDG, "qmd", "index.yml"), 0o755)

		regPath := writeTestRegistry(t, stateDir, "[[area]]\nscope = \"sync-err\"\npath = \""+filepath.ToSlash(areaDir)+"\"\n")
		var stderr bytes.Buffer
		code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
		if err == nil || code != 1 {
			t.Errorf("expected error on broken qmd config, got %v, code %d", err, code)
		}
	})

	t.Run("PruneCollections error", func(t *testing.T) {
		areaDir := filepath.Join(tmp, "prune_err_area")
		setupTestArea(t, areaDir, "[area]\nscope = \"prune-err\"\n")
		_ = os.WriteFile(filepath.Join(areaDir, "note.md"), []byte("# Note\n"), 0o644)

		orig := pruneCollectionsFn
		t.Cleanup(func() { pruneCollectionsFn = orig })
		pruneCollectionsFn = func(string, []string, OwnershipRecord) ([]string, error) {
			return nil, errors.New("prune failed")
		}

		regPath := writeTestRegistry(t, stateDir, "[[area]]\nscope = \"prune-err\"\npath = \""+filepath.ToSlash(areaDir)+"\"\n")
		var stderr bytes.Buffer
		code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
		if err == nil || code != 1 {
			t.Errorf("expected error on pruneCollectionsFn failure, got %v, code %d", err, code)
		}
	})

	t.Run("ReadDocument error", func(t *testing.T) {
		areaDir := filepath.Join(tmp, "readdoc_err_area")
		setupTestArea(t, areaDir, "[area]\nscope = \"readdoc-err\"\n")
		_ = os.WriteFile(filepath.Join(areaDir, "note.md"), []byte("# Note\n"), 0o644)

		orig := readDocFn
		t.Cleanup(func() { readDocFn = orig })
		readDocFn = func(filePath, rootDir string) (*Document, error) {
			return nil, errors.New("read doc failed")
		}

		regPath := writeTestRegistry(t, stateDir, "[[area]]\nscope = \"readdoc-err\"\npath = \""+filepath.ToSlash(areaDir)+"\"\n")
		var stderr bytes.Buffer
		code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
		if err == nil || code != 1 {
			t.Errorf("expected error on readDocFn failure, got %v, code %d", err, code)
		}
	})

	t.Run("ContentHash error", func(t *testing.T) {
		areaDir := filepath.Join(tmp, "hash_err_area")
		setupTestArea(t, areaDir, "[area]\nscope = \"hash-err\"\n")
		_ = os.WriteFile(filepath.Join(areaDir, "note.md"), []byte("# Note\n"), 0o644)

		orig := contentHashFn
		t.Cleanup(func() { contentHashFn = orig })
		contentHashFn = func(filePath string) (string, error) {
			return "", errors.New("hash failed")
		}

		regPath := writeTestRegistry(t, stateDir, "[[area]]\nscope = \"hash-err\"\npath = \""+filepath.ToSlash(areaDir)+"\"\n")
		var stderr bytes.Buffer
		code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
		if err == nil || code != 1 {
			t.Errorf("expected error on contentHashFn failure, got %v, code %d", err, code)
		}
	})

	t.Run("Embed nil stderr", func(t *testing.T) {
		regPath := writeTestRegistry(t, stateDir, "")
		code, err := EmbedWithOutput(regPath, "", port, nil)
		if err != nil || code != 0 {
			t.Errorf("expected success with nil stderr in Embed: %v, code %d", err, code)
		}
	})
}

// The prune after the sync must read the record the sync has just written.
// Read once for both, the prune would read the list from before the sync,
// which does not name the collection this run created, and write it over the
// new one: the next run would refuse that collection as someone else's.
func TestReindexPrunesFromTheRecordTheSyncJustWrote(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)

	registry := ""
	for _, scope := range []string{"knowledge", "fresh"} {
		areaDir := filepath.Join(tmp, scope)
		setupTestArea(t, areaDir, "[area]\nscope = \""+scope+"\"\n")
		_ = os.WriteFile(filepath.Join(areaDir, "a.md"), []byte("# A\n"), 0o644)
		registry += "[[area]]\nscope = \"" + scope + "\"\npath = \"" + filepath.ToSlash(areaDir) + "\"\n"
	}
	regPath := writeTestRegistry(t, stateDir, registry)

	if err := os.WriteFile(filepath.Join(stateDir, "qmd-collections.json"),
		[]byte("[\n \"knowledge\",\n \"projekt\"\n]"), 0o644); err != nil {
		t.Fatal(err)
	}
	qmdConfigDir := filepath.Join(tmp, "qmd")
	_ = os.MkdirAll(qmdConfigDir, 0o755)
	_ = os.WriteFile(filepath.Join(qmdConfigDir, "index.yml"), []byte("collections:\n"+
		"  knowledge:\n    path: C:/old/place\n    pattern: '**/*.md'\n    ignore: []\n"+
		"  projekt:\n    path: C:/gone\n    pattern: '**/*.md'\n    ignore: []\n"), 0o644)

	for run := 1; run <= 2; run++ {
		var stderr bytes.Buffer
		code, err := ReindexWithOutput(regPath, stateDir, search.NewFakePort(), &stderr)
		if err != nil || code != 0 {
			t.Fatalf("run %d: expected exit 0, got %d, %v\n%s", run, code, err, stderr.String())
		}
		written, err := os.ReadFile(filepath.Join(stateDir, "qmd-collections.json"))
		if err != nil {
			t.Fatal(err)
		}
		if want := "[\n \"fresh\",\n \"knowledge\"\n]\n"; string(written) != want {
			t.Fatalf("run %d: new record = %q, want %q", run, written, want)
		}
	}
}

// reindex reads an area's register in collect and writes it in publish; an
// approve advancing it in between would lose its row. The area's lock is
// therefore held while the stock is read.
func TestReindexHoldsTheAreaLockWhileItReadsAndWritesTheRegister(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)
	areaDir := filepath.Join(tmp, "area")
	setupTestArea(t, areaDir, "[area]\nscope = \"lock-scope\"\n")
	_ = os.WriteFile(filepath.Join(areaDir, "note.md"), []byte("# Note\n"), 0o644)
	regPath := writeTestRegistry(t, stateDir, "[[area]]\nscope = \"lock-scope\"\npath = \""+filepath.ToSlash(areaDir)+"\"\n")

	area := config.Area{Scope: "lock-scope", Path: areaDir}
	held := false
	restore := readDocFn
	t.Cleanup(func() { readDocFn = restore })
	readDocFn = func(path, root string) (*Document, error) {
		handle, free, err := lock.TryAcquire(config.AreaLockPath(area, stateDir))
		if err != nil {
			return nil, err
		}
		if free {
			_ = handle.Release()
		} else {
			held = true
		}
		return restore(path, root)
	}
	if code, err := ReindexWithOutput(regPath, stateDir, search.NewFakePort(), nil); err != nil || code != 0 {
		t.Fatalf("reindex: %v, code %d", err, code)
	}
	if !held {
		t.Fatal("the area lock was free while the stock was read")
	}
	// Released once the area is published.
	handle, free, err := lock.TryAcquire(config.AreaLockPath(area, stateDir))
	if err != nil || !free {
		t.Fatal(free, err)
	}
	_ = handle.Release()
}

func TestReindexStopsWhenTheAreaCannotBeLocked(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)
	areaDir := filepath.Join(tmp, "area")
	setupTestArea(t, areaDir, "[area]\nscope = \"lock-scope\"\n")
	regPath := writeTestRegistry(t, stateDir, "[[area]]\nscope = \"lock-scope\"\npath = \""+filepath.ToSlash(areaDir)+"\"\n")
	// A file where the lock's directory belongs.
	if err := os.WriteFile(filepath.Join(stateDir, "areas"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, err := ReindexWithOutput(regPath, stateDir, search.NewFakePort(), nil); err == nil || code != 1 {
		t.Fatalf("expected a refusal, got %v, code %d", err, code)
	}
}

// A child area whose directory name starts with two dots is nested in its
// parent; the parent must leave its files to the child.
func TestNestedAreasSeesADotDotNamedChild(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "a")
	child := filepath.Join(parent, "..b")
	areas := []config.Area{{Scope: "p/parent", Path: parent}, {Scope: "p/child", Path: child}}

	got := nestedAreas(areas[0], areas)
	if len(got) != 1 || got[0] != child {
		t.Fatalf("nestedAreas = %v, want [%s]", got, child)
	}
	beside := []config.Area{areas[0], {Scope: "p/other", Path: filepath.Join(filepath.Dir(parent), "other")}}
	if got := nestedAreas(beside[0], beside); len(got) != 0 {
		t.Fatalf("an area beside the parent is not nested, got %v", got)
	}
	if got := nestedAreas(areas[1], areas); len(got) != 0 {
		t.Fatalf("the parent of an area is not nested in it, got %v", got)
	}
}
