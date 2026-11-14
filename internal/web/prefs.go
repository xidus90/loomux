package web

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// Prefs is what the interface remembers for its user: the projects added
// through it and the dashboard tiles shown. It lives in ui.toml beside the
// other machine-wide files, which agents do not write.
type Prefs struct {
	Projects []string
	Tiles    []string
	// TilesSet tells an empty list the user chose from the absence of one.
	TilesSet bool
}

// prefsFile is the shape of ui.toml. A pointer tells `tiles = []` from no
// tiles key.
type prefsFile struct {
	Projects  []string `toml:"projects"`
	Dashboard struct {
		Tiles *[]string `toml:"tiles"`
	} `toml:"dashboard"`
}

// prefsPath is ui.toml under the state directory.
func prefsPath(stateDir string) string { return filepath.Join(stateDir, "ui.toml") }

// ReadPrefs reads ui.toml. A file that is not there is the zero value; one
// that does not parse is an error naming it.
func ReadPrefs(stateDir string) (Prefs, error) {
	data, err := os.ReadFile(prefsPath(stateDir))
	if errors.Is(err, fs.ErrNotExist) {
		return Prefs{}, nil
	}
	if err != nil {
		return Prefs{}, fmt.Errorf("read %s: %w", prefsPath(stateDir), err)
	}
	var file prefsFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return Prefs{}, fmt.Errorf("%s: %w", prefsPath(stateDir), err)
	}
	p := Prefs{Projects: file.Projects}
	if file.Dashboard.Tiles != nil {
		p.Tiles, p.TilesSet = *file.Dashboard.Tiles, true
	}
	return p, nil
}

// WritePrefs replaces ui.toml with p. The file is rewritten whole, so a
// comment a human added to it does not survive.
func WritePrefs(stateDir string, p Prefs) error {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", stateDir, err)
	}
	var out strings.Builder
	out.WriteString("projects = " + tomlList(p.Projects) + "\n")
	if p.TilesSet {
		out.WriteString("\n[dashboard]\ntiles = " + tomlList(p.Tiles) + "\n")
	}
	return lock.ReplaceText(prefsPath(stateDir), out.String())
}

// tomlList is items as one TOML array of strings.
func tomlList(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = config.QuoteTOML(item)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
