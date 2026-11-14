package web

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/target"
	"github.com/xidus90/loomux/internal/pathkey"
)

// Project is a directory the interface shows settings for.
type Project struct {
	// ID is what an address names the project by: the same path always
	// yields the same ID, and an ID carries none of the path's characters.
	ID   string
	Name string
	Root string
	// Scope is the registry scope, empty for a project that is not registered.
	Scope      string
	Registered bool
	// Missing is a project named in ui.toml that has no project
	// configuration file any more.
	Missing   bool
	LLMConfig bool
}

// projectID is the first twelve hex digits of a hash of the path in the
// spelling every name of it shares on this platform.
func projectID(root string) string {
	sum := sha256.Sum256([]byte(pathkey.Key(root)))
	return hex.EncodeToString(sum[:6])
}

// hasConfig says whether root holds a project configuration file.
func hasConfig(root string) bool {
	_, err := os.Stat(config.ManifestPath(root))
	return err == nil
}

// Projects are the registered areas that hold a project configuration file
// and the paths in ui.toml, each path once, by name. An unreadable registry
// is an error; an unreadable ui.toml adds no path. The registry knows areas
// that are no project (read-only, signposts, shared), and a project that only
// uses the hooks is in no list, so neither source alone is the list.
func Projects(stateDir string) ([]Project, error) {
	areas, err := config.ReadRegistry(stateDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	// A ui.toml that does not read counts as missing: the registered projects
	// are still the list. The pages that show or change the file say that it
	// is broken.
	prefs, _ := ReadPrefs(stateDir)
	registered := map[string]config.Area{}
	var roots []string
	for _, area := range areas {
		registered[pathkey.Key(area.Path)] = area
		if hasConfig(area.Path) {
			roots = append(roots, area.Path)
		}
	}
	roots = append(roots, prefs.Projects...)

	var out []Project
	seen := map[string]bool{}
	for _, root := range roots {
		key := pathkey.Key(root)
		if seen[key] {
			continue
		}
		seen[key] = true
		area, isRegistered := registered[key]
		out = append(out, Project{
			ID:         projectID(root),
			Name:       filepath.Base(filepath.FromSlash(root)),
			Root:       root,
			Scope:      area.Scope,
			Registered: isRegistered,
			Missing:    !hasConfig(root),
			LLMConfig:  area.LLMConfig,
		})
	}
	slices.SortFunc(out, func(a, b Project) int {
		return cmp.Or(cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)), cmp.Compare(a.Name, b.Name), cmp.Compare(a.Root, b.Root))
	})
	return out, nil
}

// globalID names the machine-wide settings in an address.
const globalID = "global"

// target finds what an address names: the machine-wide file, or a project by
// its ID. A registry or ui.toml that cannot be read names nothing.
func (s *server) target(id string) (target.Target, Project, bool) {
	if id == globalID {
		return target.Global(s.options.StateDir), Project{}, true
	}
	projects, err := Projects(s.options.StateDir)
	if err != nil {
		return target.Target{}, Project{}, false
	}
	for _, p := range projects {
		if p.ID == id {
			return target.Project(p.Root), p, true
		}
	}
	return target.Target{}, Project{}, false
}
