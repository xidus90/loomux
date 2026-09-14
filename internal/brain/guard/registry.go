package guard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/xidus90/loomux/internal/config"
)

// area is one registered area, reduced to the four fields this barrier
// decides on. The paths stay as the file spells them; whoever compares
// them resolves them first.
type area struct {
	scope     string
	path      string
	wikiPath  string
	readOnly  bool
	workspace bool
}

// readRegistry is `read_registry` (src/brain/registry.py:17-83) as the
// barrier meets it, and it is strict where `config.ReadRegistry` is not.
// That reader drops an entry it cannot use and answers the rest, which is
// the right answer for a lint and the wrong one here: every `raise` in
// `read_registry` is a refusal on the Python side, and a barrier that
// silently skipped the entry would open whatever that entry was there to
// close.
//
// The second half is not in `config.ReadRegistry` at all: Python reads
// every area's own manifest while building the registry, because `inbox`
// lives in the manifest rather than in the registration (`_inbox_of`).
// So a broken manifest in any registered area -- including a
// read-only one, whose manifest lives under the state directory -- is a
// registry that cannot be read, and refuses every write. The `inbox`
// value itself is never used here; its *validation* is what has to be
// mirrored.
func readRegistry(stateDir string) ([]area, error) {
	path := filepath.Join(stateDir, "registry.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	document := map[string]any{}
	if err := toml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("%s: not valid TOML: %w", path, err)
	}
	entries, err := areaEntries(path, document)
	if err != nil {
		return nil, err
	}
	areas := make([]area, 0, len(entries))
	seen := map[string]bool{}
	directories := map[string]string{}
	signposted := ""
	for _, raw := range entries {
		// `_required` is where Python asks the shape question, and it
		// asks it about the first key it is given -- so an entry that is
		// no table is refused here rather than three lines further on.
		entry, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf(
				"%s: expected an [[area]] table, found %s", path,
				pyRepr(raw))
		}
		scope, err := required(path, entry, "scope")
		if err != nil {
			return nil, err
		}
		if seen[scope] {
			return nil, fmt.Errorf("%s: duplicate scope %s", path,
				pyReprString(scope))
		}
		seen[scope] = true
		// An all-unsafe scope sanitises to nothing, so its state
		// directory is the shared `areas` parent -- two of them would
		// silently share one.
		directory := areaStateDir(stateDir, scope)
		if directory == areaStateDir(stateDir, "") {
			return nil, fmt.Errorf("%s: scope %s has no usable characters;"+
				" two such scopes would share one state directory and"+
				" overwrite each other", path, pyReprString(scope))
		}
		if other, taken := directories[directory]; taken {
			return nil, fmt.Errorf("%s: scopes %s and %s share the state"+
				" directory %s and would overwrite each other", path,
				pyReprString(other), pyReprString(scope),
				pyReprString(filepath.Base(directory)))
		}
		directories[directory] = scope
		wiki := entry["wiki"]
		signpost := truthy(entry["signpost"])
		if signpost && signposted != "" {
			return nil, fmt.Errorf("%s: scopes %s and %s both declare"+
				" signpost; two starting points are none", path,
				pyReprString(signposted), pyReprString(scope))
		}
		if signpost {
			signposted = scope
		}
		root, err := required(path, entry, "path")
		if err != nil {
			return nil, err
		}
		wikiPath := ""
		if truthy(wiki) {
			// `Path(wiki)` raises TypeError on anything that is not a
			// path-like, and `decide`'s catch turns that into a refusal.
			text, ok := wiki.(string)
			if !ok {
				return nil, fmt.Errorf(
					"%s: [[area]] wiki must be a string, found %s",
					path, pyRepr(wiki))
			}
			wikiPath = text
		}
		areas = append(areas, area{
			scope:     scope,
			path:      root,
			wikiPath:  wikiPath,
			readOnly:  truthy(entry["readonly"]),
			workspace: truthy(entry["workspace"]),
		})
	}
	// A second pass, where `read_registry` calls `_inbox_of` inside the
	// first one. Both sides refuse the same registrations, and the order
	// only shows where two areas are defective at once: Python names the
	// inbox of the earlier area, this side names whatever the *entry*
	// loop found first and reaches the inbox of the earlier one only if
	// no entry is malformed at all. The reason text then differs while
	// the verdict does not, which is why this is named rather than
	// rebuilt -- pulling the check into the loop above would mean
	// reading a manifest per area before the entries are known good.
	for _, registered := range areas {
		if err := checkInbox(registered, stateDir); err != nil {
			return nil, err
		}
	}
	return areas, nil
}

// areaEntries is `data.get("area", [])` with the one shape check that
// follows it: `[area]` instead of `[[area]]` yields a table, and
// iterating a table yields its keys -- every later access would then fail
// on a string.
func areaEntries(path string, document map[string]any) ([]any, error) {
	value, present := document["area"]
	if !present {
		return nil, nil
	}
	if list, ok := value.([]any); ok {
		return list, nil
	}
	// A TOML decoder hands `[[area]]` back as a slice of tables where
	// every entry is one; the untyped slice above is what a mixed or
	// non-table array yields.
	if list, ok := value.([]map[string]any); ok {
		widened := make([]any, len(list))
		for i, entry := range list {
			widened[i] = entry
		}
		return widened, nil
	}
	return nil, fmt.Errorf(
		"%s: areas must be declared as [[area]] tables, not [area]", path)
}

// required is `_required` (src/brain/registry.py:141-155): a missing key
// must surface as a complaint about the registry, not as a KeyError that
// tells the reader nothing about their file.
func required(path string, entry map[string]any, key string) (string, error) {
	value, present := entry[key]
	if !present {
		named := any(entry)
		if scope, ok := entry["scope"]; ok {
			named = scope
		}
		return "", fmt.Errorf("%s: [[area]] entry %s is missing the %s key",
			path, pyRepr(named), pyReprString(key))
	}
	text, ok := value.(string)
	if !ok || text == "" {
		return "", fmt.Errorf(
			"%s: [[area]] %s must be a non-empty string, found %s",
			path, key, pyRepr(value))
	}
	return text, nil
}

// areaStateDir is `area_state_dir` (src/brain/paths.py:41-43). It is
// asked of `config.ManifestDir` rather than spelt again, because that
// function answers exactly `<state>/areas/<flat scope>` for a read-only
// area -- the sanitising rule lives there, and a second copy of it here
// would be a copy that drifts.
func areaStateDir(stateDir, scope string) string {
	return config.ManifestDir(
		config.Area{Scope: scope, ReadOnly: true}, stateDir)
}

// manifestPath is `manifest_path` (src/brain/registry.py:132-138): the
// manifest follows the artefacts, so a read-only area's declaration is
// read from the state directory rather than from the tree we do not own.
func manifestPath(registered area, stateDir string) string {
	base := registered.path
	if registered.readOnly {
		base = areaStateDir(stateDir, registered.scope)
	}
	return filepath.Join(base, bundleDir, manifestName)
}

// checkInbox is `_inbox_of` (src/brain/registry.py:86-109) kept for its
// refusals. Registering an area before it declares itself is normal, so a
// missing manifest is no defect; a manifest that cannot be read, or one
// whose inbox reaches out of the area, refuses every write.
func checkInbox(registered area, stateDir string) error {
	declaration := manifestPath(registered, stateDir)
	if !isRegularFile(declaration) {
		return nil
	}
	read, err := readManifest(declaration)
	if errors.Is(err, errNoArea) {
		return nil
	}
	if err != nil {
		return err
	}
	inbox := read.layout["inbox"]
	if !truthy(inbox) {
		return nil
	}
	value, ok := inbox.(string)
	if !ok {
		// `Path(1).is_absolute()` raises TypeError, which becomes a
		// refusal like every other failure of this reader.
		return fmt.Errorf("%s: [layout] inbox must be a string, found %s",
			declaration, pyRepr(inbox))
	}
	// `Path("/x")` on the right of `/` replaces the left side outright --
	// an absolute value would silently leave the area tree instead of
	// naming a folder inside it.
	if isAbsolutePath(value) {
		return fmt.Errorf(
			"%s: [layout] inbox must be relative to the area, found %s",
			declaration, pyReprString(value))
	}
	return nil
}

// isAbsolutePath is `Path.is_absolute()` on the platform this runs on,
// and it is deliberately no stricter. Measured here: on Windows `/in` is
// *not* absolute -- a rooted path without a drive is not called one --
// while `C:/in` and `//srv/x` are, and `filepath.IsAbs` answers all three
// the same way. `_declared_review` names that gap itself
// (src/brain/maintenance/reconcile.py:258-261) and uses containment
// instead; `_inbox_of` does not, so a `/in` inbox passes on both sides.
func isAbsolutePath(value string) bool {
	return filepath.IsAbs(value)
}
