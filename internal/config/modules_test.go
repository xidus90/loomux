package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, text string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ManifestPath(root), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestReadModulesWithoutAFileTurnsEverythingOn(t *testing.T) {
	got, err := ReadModules(t.TempDir())
	if err != nil || got != AllModules() {
		t.Fatalf("got %+v, %v; want all on", got, err)
	}
}

func TestReadModulesWithoutTheTableTurnsEverythingOn(t *testing.T) {
	got, err := ReadModules(writeManifest(t, "[area]\nscope = \"x\"\n"))
	if err != nil || got != AllModules() {
		t.Fatalf("got %+v, %v; want all on", got, err)
	}
}

func TestReadModulesKeepsAMissingKeyOn(t *testing.T) {
	got, err := ReadModules(writeManifest(t, "[modules]\ngraph = false\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := (Modules{Hooks: true, Brain: true, Graph: false}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestReadModulesSwitchesEachModuleByItsOwnKey(t *testing.T) {
	got, err := ReadModules(writeManifest(t, "[modules]\nhooks = false\nbrain = false\ngraph = true\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := (Modules{Hooks: false, Brain: false, Graph: true}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestReadModulesRefusesWhatItCannotUse(t *testing.T) {
	for name, text := range map[string]string{
		"unknown key": "[modules]\nos = true\n",
		"not a bool":  "[modules]\nbrain = \"no\"\n",
		"not a table": "modules = 1\n",
		"broken toml": "[modules\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ReadModules(writeManifest(t, text))
			if err == nil || !strings.Contains(err.Error(), "config.toml") {
				t.Fatalf("got %v, want an error naming the file", err)
			}
		})
	}
}

func TestReadModulesReportsAnUnreadableFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(ManifestPath(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadModules(root); err == nil {
		t.Fatal("a directory where the file should be must be an error")
	}
}
