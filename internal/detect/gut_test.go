package detect

import (
	"testing"
	"testing/fstest"
)

// GUT installs itself as an addon; its plugin.cfg marks the project, at the
// root and in a Godot project one level down.
func TestGutIsDetectedByItsAddon(t *testing.T) {
	for _, prefix := range []string{"", "game/"} {
		facts := Detect(fstest.MapFS{
			prefix + "project.godot":         {Data: []byte("config_version=4\n")},
			prefix + "addons/gut/plugin.cfg": {Data: []byte("[plugin]\nname=\"Gut\"\n")},
		})
		if !has(facts.Stacks, "gut") || !has(facts.Stacks, "gdscript") {
			t.Errorf("%q: stacks = %v, want gut and gdscript", prefix, facts.Stacks)
		}
	}
}
