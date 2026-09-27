package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchSettingsDefaultWithoutAFile(t *testing.T) {
	s, err := ReadSearchSettings(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if s.Backbone != DefaultSearchBackbone || DefaultSearchBackbone != "cuda" {
		t.Fatalf("%+v", s)
	}
}

func TestSearchSettingsReadTheGlobalFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[model]\nenabled = true\n[search]\nbackbone = \"vulkan\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ReadSearchSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Backbone != "vulkan" {
		t.Fatalf("%+v", s)
	}
}

func TestSearchSettingsTakeEveryBackbone(t *testing.T) {
	for _, backbone := range SearchBackbones() {
		s, err := ParseSearchSettings("c.toml", "[search]\nbackbone = \""+backbone+"\"\n")
		if err != nil || s.Backbone != backbone {
			t.Fatalf("%s: %+v %v", backbone, s, err)
		}
	}
	if got := strings.Join(SearchBackbones(), ","); got != "cuda,vulkan,cpu" {
		t.Fatal(got)
	}
}

// Every refusal names the file, and one about the value names the key: the
// person who reads it has to find the line.
func TestSearchSettingsRefuse(t *testing.T) {
	for name, c := range map[string]struct{ text, want string }{
		"no TOML":      {"[search\n", "not valid TOML"},
		"no table":     {"search = 1\n", "[search] must be a table"},
		"no string":    {"[search]\nbackbone = 1\n", "[search] backbone must be a non-empty string"},
		"empty":        {"[search]\nbackbone = \"\"\n", "[search] backbone must be a non-empty string"},
		"unknown":      {"[search]\nbackbone = \"metal\"\n", `[search] backbone must be cuda, vulkan or cpu, found "metal"`},
		"another case": {"[search]\nbackbone = \"CUDA\"\n", `found "CUDA"`},
	} {
		_, err := ParseSearchSettings("c.toml", c.text)
		if err == nil || !strings.HasPrefix(err.Error(), "c.toml: ") || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Other blocks are not this reader's to judge: a [model] the model reader
// would refuse does not stop the search from reading its own block.
func TestSearchSettingsReadOnlyTheirBlock(t *testing.T) {
	s, err := ParseSearchSettings("c.toml", "[model]\ntemperature = 9\n[search]\nbackbone = \"cpu\"\n")
	if err != nil || s.Backbone != "cpu" {
		t.Fatalf("%+v %v", s, err)
	}
}

func TestSearchSettingsNameAnUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("[search]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	saved := readModelFile
	readModelFile = func(string) ([]byte, error) { return nil, errors.New("denied") }
	defer func() { readModelFile = saved }()
	_, err := ReadSearchSettings(dir)
	if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "denied") {
		t.Fatal(err)
	}
}

// A directory where the file should be declares nothing, as for [model].
func TestSearchSettingsTakeTheDefaultForANonFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := ReadSearchSettings(dir)
	if err != nil || s.Backbone != DefaultSearchBackbone {
		t.Fatalf("%+v %v", s, err)
	}
}

func TestTheGlobalSearchKeysAreTheOnesTheReaderReads(t *testing.T) {
	if got := strings.Join(GlobalSearchKeys(), ","); got != "backbone" {
		t.Fatal(got)
	}
}
