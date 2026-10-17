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
// another file, which no switch of this root opens.
func ManifestUnlinked(root string) bool {
	base, err := ResolvePath(root)
	manifest, manifestErr := ResolvePath(filepath.Join(root, bundleDir, manifestName))
	return err == nil && manifestErr == nil && pathsEqual(manifest, filepath.Join(base, bundleDir, manifestName))
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

// AtRoot says whether the full directory dir is root itself once both are
// resolved. A shell spells its paths from where it stands, so only there do
// they mean what they would mean at the root.
func AtRoot(dir, root string) bool {
	here, err := ResolvePath(dir)
	base, rootErr := ResolvePath(root)
	return root != "" && filepath.IsAbs(dir) && err == nil && rootErr == nil && pathsEqual(here, base)
}

// ownManifest is whether `resolved` is the own manifest of the hook's root
// while that root's area has its llm_config switch on and the manifest is
// reached without a link. Every other manifest stays the barrier's own,
// another area's with its switch on included.
func ownManifest(resolved, root, stateDir string) bool {
	if !OwnManifest(resolved, root) || !ManifestUnlinked(root) {
		return false
	}
	_, llmConfig, err := AreaAt(root, stateDir)
	return err == nil && llmConfig
}
