package verify

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// classNameLine finds a script's class_name, with Godot 3's icon after a comma
// allowed: `class_name GutTest, "res://icon.png"`.
func classNameLine() *regexp.Regexp { return regexp.MustCompile(`(?m)^class_name\s+([A-Za-z_]\w*)`) }

// listedClass finds one entry of project.godot's _global_script_classes.
func listedClass() *regexp.Regexp { return regexp.MustCompile(`"class":\s*"([A-Za-z_]\w*)"`) }

// ClassListMissing names the class_names a Godot 3 project in dir declares
// that its project.godot does not list, sorted, and reports whether dir is a
// Godot 3 project at all. Godot 3 keeps no class cache of its own: it
// resolves a global class through _global_script_classes, which only the
// editor writes. The scripts of addons count, since a test framework's base
// class lives there; .import and .godot directories hold no source.
func ClassListMissing(dir string) ([]string, bool) {
	want, _, err := projectGodot(filepath.Join(dir, "project.godot"))
	if err != nil || want.Major != "3" {
		return nil, false
	}
	data, _ := os.ReadFile(filepath.Join(dir, "project.godot"))
	listed := map[string]bool{}
	for _, m := range listedClass().FindAllStringSubmatch(string(data), -1) {
		listed[m[1]] = true
	}
	declares := classNameLine()
	var missing []string
	// WalkDir hands an error only with a directory, and the root was read
	// above, so an unreadable directory is simply one without scripts. The
	// walk skips what Godot's own scan skips: hidden directories (.import,
	// .git, a worktree copy under .claude), one holding a .gdignore, and a
	// nested project of its own.
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, _ error) error {
		switch {
		case d.IsDir() && p != dir && (strings.HasPrefix(d.Name(), ".") || exists(filepath.Join(p, ".gdignore")) || exists(filepath.Join(p, "project.godot"))):
			return filepath.SkipDir
		case d.IsDir() || !strings.HasSuffix(d.Name(), ".gd"):
			return nil
		}
		b, _ := os.ReadFile(p)
		for _, m := range declares.FindAllStringSubmatch(string(b), -1) {
			if !listed[m[1]] && !slices.Contains(missing, m[1]) {
				missing = append(missing, m[1])
			}
		}
		return nil
	})
	slices.Sort(missing)
	return missing, true
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
