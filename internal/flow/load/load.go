package load

import (
	"io/fs"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/flow"
)

// Load reads a found flow and returns the graph it stands for, or the
// findings of the first stage that had any.
//
// A stage reports everything it found, and then Load stops: every stage builds
// on what the one before it declared sound, and a later stage run on an
// unsound graph would report mistakes the file does not have.
func Load(found Found, catalog *flow.Catalog) (*flow.Graph, error) {
	raw, err := fs.ReadFile(found.Files, "flow.toml")
	if err != nil {
		return nil, Findings{found.File + ": " + err.Error()}
	}
	var doc map[string]any
	if _, err := toml.Decode(string(raw), &doc); err != nil {
		return nil, Findings{found.File + ": " + err.Error()}
	}
	made, findings := declarations(found.File, doc)
	made.graph.Name, made.graph.File, made.graph.Texts = found.Name, found.File, found.Files
	made.graph.Origin, made.graph.Overlays = found.Origin, found.Overlays
	if len(findings) > 0 {
		return nil, findings
	}
	for _, check := range []stage{stage2, stage3, stage4, stage5, stage6} {
		if findings := check(found.File, made, catalog); len(findings) > 0 {
			return nil, findings
		}
	}
	return made.graph, nil
}
