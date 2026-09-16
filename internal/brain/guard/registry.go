package guard

import (
	"errors"
	"path/filepath"

	"github.com/xidus90/loomux/internal/config"
)

// area is one registered area, reduced to the fields this barrier decides
// on. The paths stay as the file spells them; whoever compares them
// resolves them first.
type area struct {
	scope     string
	path      string
	wikiPath  string
	readOnly  bool
	workspace bool
}

// readRegistry reads the registry through config, which brain reads it
// through too, and then every registered area's declaration: a broken
// declaration in any area -- a read-only one included, whose manifest lives
// under the state directory -- refuses every write.
func readRegistry(stateDir string) ([]area, error) {
	registered, err := config.ReadRegistry(stateDir)
	if err != nil {
		return nil, err
	}
	areas := make([]area, 0, len(registered))
	for _, entry := range registered {
		areas = append(areas, area{
			scope:     entry.Scope,
			path:      entry.Path,
			wikiPath:  entry.WikiPath,
			readOnly:  entry.ReadOnly,
			workspace: entry.Workspace,
		})
	}
	for _, registered := range areas {
		if err := checkDeclaration(registered, stateDir); err != nil {
			return nil, err
		}
	}
	return areas, nil
}

// areaStateDir is where a read-only area keeps its artefacts; the rule that
// names it lives in config.ManifestDir.
func areaStateDir(stateDir, scope string) string {
	return config.ManifestDir(
		config.Area{Scope: scope, ReadOnly: true}, stateDir)
}

// manifestPath is the one declaration this barrier reads for a registered
// area: the manifest follows the artefacts, so a read-only area's
// declaration is read from the state directory rather than from the tree
// the barrier does not own.
func manifestPath(registered area, stateDir string) string {
	base := registered.path
	if registered.readOnly {
		base = areaStateDir(stateDir, registered.scope)
	}
	return filepath.Join(base, bundleDir, manifestName)
}

// checkDeclaration refuses a registered area whose declaration exists and is
// broken. Registering an area before it declares itself is normal, so a
// missing file and a configuration without [area] are no defect.
func checkDeclaration(registered area, stateDir string) error {
	declaration := manifestPath(registered, stateDir)
	if !isRegularFile(declaration) {
		return nil
	}
	manifest, err := config.ReadDeclaration(declaration)
	if errors.Is(err, config.ErrNoArea) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = manifest.InboxLayout()
	return err
}
