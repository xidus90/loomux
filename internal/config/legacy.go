package config

import (
	"os"
	"path/filepath"
	"runtime"
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

// LegacyBrainDirUntilStage3 is ultra-brain's state directory. brain/* reads the artefacts of
// read-only areas and the reconcile stamp from it until stage 3 moves reconcile to Go.
//
// Expires with stage 3: until then the Python side writes `graph.json`,
// `_identities.tsv`, `index.md` of read-only areas and
// `maintenance/last-run.txt` there, and loomux has no writer of its own.
// The registry is not read from here; it stays StateDir's.
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

// ReadAreaManifestUntilStage4 reads an area's manifest under the name loomux writes, or under
// one of ultra-brain's names until stage 4 moves the hosts: .loomux/config.toml, else
// .ultra-brain/config.toml, else .brain.toml. Only brain/* calls it.
//
// Expires with stage 4, when the area repositories carry
// `.loomux/config.toml`. Until then a `local_only` area that still declares
// itself in `.brain.toml` would be read as having no manifest, and its
// privacy would go unseen. The write barrier and ReadManifest stay with the
// one loomux name.
//
// Like `read_manifest` (src/brain/manifest.py:19-24) it refuses a manifest
// without a non-empty `[area] scope`; the other checks Python makes there
// are not repeated. A `.loomux/config.toml` without an `[area]` table
// declares nothing, and the next name is asked.
func ReadAreaManifestUntilStage4(dir string) (*Manifest, error) {
	return readManifestAmong(dir, manifestNamesUntilStage4, true)
}
