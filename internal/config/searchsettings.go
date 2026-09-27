package config

// The search engine's settings in the per-user file `<state>/config.toml`.
// They are machine-wide only: the backbone is a property of this machine's
// graphics card and of the one qmd daemon every project shares, so no area
// has a say in it.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/BurntSushi/toml"
)

// DefaultSearchBackbone is the backbone qmd runs on when nothing is said.
// search.DefaultBackbone is the same value; this package cannot import it.
const DefaultSearchBackbone = "cuda"

// SearchBackbones are the backbones [search] backbone takes, in the order a
// refusal names them.
func SearchBackbones() []string { return []string{"cuda", "vulkan", "cpu"} }

// GlobalSearchKeys are the keys ParseSearchSettings reads; the schema of
// `loomux config --global` is held against them.
func GlobalSearchKeys() []string { return []string{"backbone"} }

// SearchSettings are the [search] block of the global file.
type SearchSettings struct {
	Backbone string // one of SearchBackbones
}

// ReadSearchSettings reads the global file. Anything but a regular file
// declares nothing and yields the defaults.
func ReadSearchSettings(stateDir string) (SearchSettings, error) {
	path := filepath.Join(stateDir, "config.toml")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return ParseSearchSettings(path, "")
	}
	data, err := readModelFile(path)
	if err != nil {
		return SearchSettings{}, fmt.Errorf("%s: cannot be read: %w", path, err)
	}
	return ParseSearchSettings(path, string(data))
}

// ParseSearchSettings reads the [search] block of text; path only names the
// file in a refusal. The other blocks are left to their own readers.
func ParseSearchSettings(path, text string) (SearchSettings, error) {
	s, err := parseSearchSettings(text)
	if err != nil {
		return SearchSettings{}, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

func parseSearchSettings(text string) (SearchSettings, error) {
	document := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &document); err != nil {
		return SearchSettings{}, fmt.Errorf("not valid TOML: %w", err)
	}
	block := map[string]any{}
	if value, present := document["search"]; present {
		table, ok := value.(map[string]any)
		if !ok {
			return SearchSettings{}, errors.New("[search] must be a table")
		}
		block = table
	}
	if _, present := block["backbone"]; !present {
		return SearchSettings{Backbone: DefaultSearchBackbone}, nil
	}
	backbone, err := optionalString(block, "backbone", "[search]", " ")
	if err != nil {
		return SearchSettings{}, err
	}
	// Spelt exactly: the file is read by people too, and "CUDA" there would
	// suggest a difference that is none.
	if !slices.Contains(SearchBackbones(), backbone) {
		return SearchSettings{}, fmt.Errorf("[search] backbone must be cuda, vulkan or cpu, found %q", backbone)
	}
	return SearchSettings{Backbone: backbone}, nil
}
