package godotsuite

import (
	"embed"
	"os"
	"path/filepath"
)

// GlueDir is where WriteGlue puts the scripts, relative to the project root.
// Under .godot/ so they never show up as project files.
const GlueDir = ".godot/loomux"

//go:embed gd/*.gd
var glue embed.FS

// WriteGlue writes the runner, the coverage hook and the disk
// instrumentation script into root/GlueDir, overwriting what is there.
func WriteGlue(root string) error {
	dir := filepath.Join(root, filepath.FromSlash(GlueDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"run_GdUnitCmdTool.gd", "hook.gd", "disk.gd"} {
		data, _ := glue.ReadFile("gd/" + name) // embedded at build time; cannot fail
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
