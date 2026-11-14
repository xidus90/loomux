package web

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// makeProject makes a directory under a name that stresses every path
// handling (a space, a hash, an umlaut) and, with text, gives it a
// configuration file.
func makeProject(t *testing.T, name, text string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "#GIT", name)
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if text != "" {
		if err := os.WriteFile(config.ManifestPath(root), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// remember adds roots to the projects of ui.toml.
func remember(t *testing.T, stateDir string, roots ...string) {
	t.Helper()
	p, err := ReadPrefs(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	p.Projects = append(p.Projects, roots...)
	if err := WritePrefs(stateDir, p); err != nil {
		t.Fatal(err)
	}
}

func names(projects []Project) []string {
	out := make([]string, len(projects))
	for i, p := range projects {
		out[i] = p.Name
	}
	return out
}

func TestProjectsAreTheRegisteredAreasWithAConfigAndThePathsOfUITOML(t *testing.T) {
	state := t.TempDir()
	registered := makeProject(t, "beta", "[modules]\n")
	signpost := makeProject(t, "signpost", "")
	remembered := makeProject(t, "Alpha", "[modules]\n")
	gone := makeProject(t, "gamma", "")
	err := config.WriteRegistry(state, []config.Area{
		{Scope: "project/beta", Path: filepath.ToSlash(registered), LLMConfig: true},
		{Scope: "project/signpost", Path: filepath.ToSlash(signpost), Signpost: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	// The registered project again, spelled differently, and one with none.
	remember(t, state, registered, remembered, gone)

	projects, err := Projects(state)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(names(projects), ","); got != "Alpha,beta,gamma" {
		t.Fatalf("Projects = %s, want Alpha,beta,gamma: by name, each once, the signpost left out", got)
	}
	alpha, beta, gamma := projects[0], projects[1], projects[2]
	if alpha.Registered || alpha.Missing || alpha.Scope != "" {
		t.Errorf("alpha = %+v, want a remembered project that exists", alpha)
	}
	if !beta.Registered || beta.Scope != "project/beta" || !beta.LLMConfig || beta.Missing {
		t.Errorf("beta = %+v, want the registered area with llm_config", beta)
	}
	if !gamma.Missing {
		t.Errorf("gamma = %+v, want Missing: no configuration file", gamma)
	}
}

func TestAProjectIDIsStableAndCarriesNoPathCharacters(t *testing.T) {
	root := makeProject(t, "mein projekt ü", "[modules]\n")
	state := t.TempDir()
	remember(t, state, root)
	first, err := Projects(state)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := Projects(state)
	if first[0].ID != second[0].ID || !regexp.MustCompile(`^[0-9a-f]{12}$`).MatchString(first[0].ID) {
		t.Fatalf("ids %q and %q, want one stable 12 hex digits", first[0].ID, second[0].ID)
	}
	if first[0].Name != "mein projekt ü" {
		t.Errorf("Name = %q, want the last path element", first[0].Name)
	}
	if other := projectID(filepath.Join(root, "x")); other == first[0].ID {
		t.Error("two paths share an ID")
	}
}

func TestProjectsOfABrokenRegistryAreAnError(t *testing.T) {
	state := t.TempDir()
	if err := os.WriteFile(filepath.Join(state, "registry.toml"), []byte("area = 3"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Projects(state); err == nil {
		t.Error("a broken registry gave a project list")
	}
}

// A ui.toml that does not read counts as missing: the registered projects
// still list, and the file's own paths are not there.
func TestABrokenUITomlAddsNoProjectAndHidesNoRegisteredOne(t *testing.T) {
	state := t.TempDir()
	root := makeProject(t, "reg", "[modules]\n")
	if err := config.AddArea(state, config.Area{Scope: "project/reg", Path: filepath.ToSlash(root)}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prefsPath(state), []byte("projects = ["), 0o600); err != nil {
		t.Fatal(err)
	}
	projects, err := Projects(state)
	if err != nil || len(projects) != 1 || filepath.FromSlash(projects[0].Root) != root {
		t.Fatalf("Projects = %+v, %v; want the registered project", projects, err)
	}
}
