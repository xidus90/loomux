package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// legacyBrainDirEnvUntilStage3 points brain/* at another copy of
// ultra-brain's state directory. The recorded cases set it to the staged
// world, the way they set LOOMUX_STATE_DIR; the Python reference reads
// BRAIN_STATE_DIR instead. Expires with LegacyBrainDirUntilStage3.
const legacyBrainDirEnvUntilStage3 = "LOOMUX_LEGACY_BRAIN_DIR"

// manifestNamesUntilStage4 are the names ReadAreaManifestUntilStage4 tries,
// in this order: loomux's own first, then `manifest_path`'s two
// (src/brain/registry.py:132-138), which asks `.ultra-brain/config.toml`
// before `.brain.toml`. Expires with ReadAreaManifestUntilStage4.
var manifestNamesUntilStage4 = []string{
	filepath.Join(".loomux", "config.toml"),
	filepath.Join(".ultra-brain", "config.toml"),
	".brain.toml",
}

// LegacyBrainDirUntilStage3 is ultra-brain's state directory. Since stage 3a
// it is the **fallback**, not the place: ArtifactLookup reads the new state
// directory first and falls back here as long as `loomux migrate` (stage 4)
// has not moved the stock. Nothing is ever written here.
// The registry does not come from here; it is StateDir's.
//
// The name promises an expiry that stage 3a moved: the fallback now lives
// until `loomux migrate` in stage 4, not until stage 3. It keeps the old name
// on purpose: the brain commands of stage 1b, `serve` and ArtifactLookup
// call it, and a rename would touch them for no change in behaviour.
func LegacyBrainDirUntilStage3() string {
	if fromEnv := os.Getenv(legacyBrainDirEnvUntilStage3); fromEnv != "" {
		return fromEnv
	}
	home, _ := os.UserHomeDir()
	return legacyBrainDirUntilStage3For(runtime.GOOS, os.Getenv, home)
}

// legacyBrainDirUntilStage3For is `_platform_default`
// (src/brain/paths.py:25-31) with its inputs made arguments, and expires
// with LegacyBrainDirUntilStage3. It asks defaultStateDir and swaps the last
// element, because the two directories differ in nothing but that name:
// both sit in LOCALAPPDATA, home\AppData\Local, XDG_STATE_HOME or
// home/.local/state.
func legacyBrainDirUntilStage3For(goos string, getenv func(string) string, home string) string {
	return filepath.Join(filepath.Dir(defaultStateDir(goos, getenv, home)), "brain")
}

// ReadAreaManifestUntilStage4 reads an area's declaration under the name loomux writes, or
// under one of ultra-brain's names until stage 4 moves the hosts:
// .loomux/config.toml, else .ultra-brain/config.toml, else .brain.toml. Only brain/* calls it.
//
// The first name that is a regular file decides, and it is checked whole by
// ReadDeclaration. A .loomux/config.toml without [area] is policy only and
// the next name is asked; the old names have no such form, so there the
// missing table is a missing scope.
func ReadAreaManifestUntilStage4(dir string) (*Manifest, error) {
	for _, name := range manifestNamesUntilStage4 {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			continue
		}
		manifest, err := ReadDeclaration(path)
		if errors.Is(err, ErrNoArea) {
			if name == manifestNamesUntilStage4[0] {
				continue
			}
			return nil, fmt.Errorf("%s: [area] is missing %q", path, "scope")
		}
		return manifest, err
	}
	return nil, fmt.Errorf("%s: %w (%s)", dir, ErrNoManifest, strings.Join(manifestNamesUntilStage4, ", "))
}
