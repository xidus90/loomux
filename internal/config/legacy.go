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
