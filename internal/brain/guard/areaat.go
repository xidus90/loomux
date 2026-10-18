package guard

import "path/filepath"

// AreaAt is the area whose project root `root` is -- its registered path, or
// a linked worktree of a workspace area -- with that area's llm_config
// switch. scope == "" means no area is rooted there: a directory below an
// area's root is not its root, because only the root carries the manifest.
//
// Every failure is an error rather than "no area", so that a caller deciding
// whether the manifest may be written refuses instead of guessing.
func AreaAt(root, stateDir string) (scope string, llmConfig bool, err error) {
	resolved, err := ResolvePath(root)
	if err != nil {
		return "", false, err
	}
	areas, err := readRegistry(stateDir)
	if err != nil {
		return "", false, err
	}
	for _, registered := range areas {
		path, err := ResolvePath(registered.path)
		if err != nil {
			return "", false, err
		}
		if pathsEqual(path, resolved) {
			return registered.scope, registered.llmConfig, nil
		}
	}
	// The climb starts above the path it is given, so it is handed a path
	// inside the root and has to come back with the root itself.
	inside := filepath.Join(resolved, bundleDir)
	for _, registered := range areas {
		for _, linked := range linkedWorktreeRoots([]string{inside}, []area{registered}) {
			if pathsEqual(linked, resolved) {
				return registered.scope, registered.llmConfig, nil
			}
		}
	}
	return "", false, nil
}

// ManifestUnlinked says whether the manifest at root is reached without a
// link: its path resolves to the manifest below the resolved root, whether
// the file exists yet or not. Through a linked .loomux or a linked file it is
// another file, which no switch of this root opens; so is a manifest of more
// than one name (HardLinked), which `config set` would write in place.
func ManifestUnlinked(root string) bool {
	base, err := ResolvePath(root)
	manifest, manifestErr := ResolvePath(filepath.Join(root, bundleDir, manifestName))
	return err == nil && manifestErr == nil && pathsEqual(manifest, filepath.Join(base, bundleDir, manifestName)) &&
		HardLinked(manifest) == ""
}

// OwnManifest says whether the full path names root's own manifest: it
// resolves to the manifest directly below the resolved root, in that
// spelling up to case. A relative path, a path through a link to another
// file and a hook without a root name none.
func OwnManifest(path, root string) bool {
	base, err := ResolvePath(root)
	resolved, pathErr := ResolvePath(path)
	return root != "" && filepath.IsAbs(path) && err == nil && pathErr == nil &&
		pathsEqual(resolved, filepath.Join(base, bundleDir, manifestName))
}

// ManifestBase is the root whose own manifest the full path names: root
// itself, or a linked worktree of root's repository. A host may name the
// main checkout as the hook's root while its session works in such a
// worktree, and the worktree is the same project. "" where it names neither.
func ManifestBase(path, root string) string {
	if OwnManifest(path, root) {
		return root
	}
	resolved, err := ResolvePath(path)
	if root == "" || !filepath.IsAbs(path) || err != nil {
		return ""
	}
	base := filepath.Dir(filepath.Dir(resolved))
	common := repositoryCommon(root)
	if common == "" || !pathsEqual(linkedCommon(base), common) || !OwnManifest(path, base) {
		return ""
	}
	return base
}

// AtRoot says whether the full directory dir is root itself once both are
// resolved. A shell spells its paths from where it stands, so only there do
// they mean what they would mean at the root.
func AtRoot(dir, root string) bool {
	here, err := ResolvePath(dir)
	base, rootErr := ResolvePath(root)
	return root != "" && filepath.IsAbs(dir) && err == nil && rootErr == nil && pathsEqual(here, base)
}

// ownManifest is whether `resolved` is the own manifest of the hook's root,
// or of a linked worktree of it, while the root's area has its llm_config
// switch on and the manifest is reached without a link. Every other manifest
// stays the barrier's own, another area's with its switch on included.
func ownManifest(resolved, root, stateDir string) bool {
	base := ManifestBase(resolved, root)
	if base == "" || !ManifestUnlinked(base) {
		return false
	}
	_, llmConfig, err := AreaAt(root, stateDir)
	return err == nil && llmConfig
}
