package privacy

import (
	"fmt"

	"github.com/xidus90/loomux/internal/config"
)

// Channel defines the transport environment through which brain is accessed.
type Channel string

const (
	ChannelLocal Channel = "local"
	ChannelCloud Channel = "cloud"
)

// ParseChannel parses a channel string into a valid Channel enum.
func ParseChannel(s string) (Channel, error) {
	switch s {
	case "local":
		return ChannelLocal, nil
	case "cloud":
		return ChannelCloud, nil
	default:
		return "", fmt.Errorf("invalid channel: %q (must be 'local' or 'cloud')", s)
	}
}

// VisibleManifest reads the declaration lying in dir and answers it together
// with whether this channel may see the area at all. It is the one gate of
// every surface that serves an area -- search, catalog, read, neighbors,
// status -- and it exists because each of them had the same hole: the
// manifest error was discarded, and IsVisible counts a nil manifest as
// visible, so an area whose `local_only` declaration could not be read was
// served on the cloud channel (finding N3 of the scheibe-6 merge re-review).
//
// Every failure is the caller's error, the absent declaration included. That
// is `_visible_areas` (src/brain/core.py:248-263): `read_manifest` of
// `manifest_path` raises for a file that does not exist, does not read, is not
// TOML or names no known mode, nothing on the way to the command line catches
// it, and so one area without a usable declaration stops the whole call.
// ultra-brain's Go gate answered an absent declaration as visible and hid the
// other failures silently; loomux follows the reference.
//
// The file is `.loomux/config.toml` alone, read by config.ReadAreaDeclaration.
//
// dir is where the manifest lies, which is not always the area: a read-only
// area keeps it in the state directory, so callers pass config.ManifestDir
// -- the directory `registry.manifest_path` reads on the Python side.
func VisibleManifest(dir string, ch Channel) (*config.Manifest, bool, error) {
	manifest, err := config.ReadAreaDeclaration(dir)
	if err != nil {
		return nil, false, err
	}
	return manifest, IsVisible(manifest, ch), nil
}

// IsVisible determines whether an area is visible on a given channel.
// Areas configured with privacy mode "local_only" are hidden from ChannelCloud.
//
// A nil manifest is visible. VisibleManifest never hands one on; the arm
// answers a caller that holds no declaration at all.
func IsVisible(manifest *config.Manifest, ch Channel) bool {
	if manifest == nil {
		return true
	}
	if manifest.PrivacyMode == "local_only" && ch == ChannelCloud {
		return false
	}
	return true
}
