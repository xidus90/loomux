// Package importcases translates recordings of the old binaries into loomux's
// world: the command line follows a mapping, the path literals of both old
// tools become loomux's one configuration file, and the staged worlds are
// rewritten the same way.
package importcases

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/cases"
)

// Rule rewrites the head of a recorded command line.
type Rule struct {
	From string
	To   string
}

// Mapping is the set of rules, read from a file of [[command]] tables.
type Mapping struct {
	Commands []Rule `toml:"command"`
}

// The old tools named their configuration in four places; loomux has one.
var pathLiterals = []string{".ultra-brain/config.toml", ".ultraloom/policy.toml", ".brain.toml"}

const loomuxConfig = ".loomux/config.toml"

// Import copies every case below from into to, rewriting its command, its
// payload and its staged worlds on the way.
func Import(from, to string, m Mapping) error {
	found, err := cases.DiscoverCases(from, "")
	if err != nil {
		return err
	}
	for _, c := range found {
		out := filepath.Join(to, c.Verb, c.Name)
		// The two files this import rewrites are not copied first: one writer
		// per file keeps the copy from being the one that fails.
		skip := map[string]bool{"cmd": true, "stdin": len(c.Stdin) > 0}
		if err := copyTree(c.Path, out, skip); err != nil {
			return err
		}
		cmd, err := rewriteCommand(c, m)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, "cmd"), []byte(cmd+"\n"), 0o644); err != nil {
			return err
		}
		if len(c.Stdin) > 0 {
			payload := rewritePaths(string(c.Stdin))
			if err := os.WriteFile(filepath.Join(out, "stdin"), []byte(payload), 0o644); err != nil {
				return err
			}
		}
		for _, world := range []string{"world", "world_after"} {
			dir := filepath.Join(out, world)
			if info, err := os.Stat(dir); err != nil || !info.IsDir() {
				continue
			}
			if err := TranslateWorld(dir); err != nil {
				return err
			}
		}
	}
	return nil
}

// rewriteCommand applies the first rule whose head the command carries.
func rewriteCommand(c *cases.Case, m Mapping) (string, error) {
	for _, rule := range m.Commands {
		if strings.HasPrefix(c.Cmd, rule.From) {
			return rewritePaths(rule.To + strings.TrimPrefix(c.Cmd, rule.From)), nil
		}
	}
	return "", fmt.Errorf("no rule for the command of case %s/%s: %s", c.Verb, c.Name, c.Cmd)
}

// rewritePaths puts loomux's configuration in place of the old tools' files.
func rewritePaths(s string) string {
	for _, literal := range pathLiterals {
		s = strings.ReplaceAll(s, literal, loomuxConfig)
	}
	return s
}

// TranslateWorld rewrites the configuration files of the old tools in dir
// (and in every dir/areas/<name>) into one .loomux/config.toml.
func TranslateWorld(dir string) error {
	if err := translateDir(dir); err != nil {
		return err
	}
	areas, err := os.ReadDir(filepath.Join(dir, "areas"))
	if err != nil {
		return nil
	}
	for _, area := range areas {
		if !area.IsDir() {
			continue
		}
		if err := translateDir(filepath.Join(dir, "areas", area.Name())); err != nil {
			return err
		}
	}
	return nil
}

// translateDir folds the old files of one directory into one config.
func translateDir(dir string) error {
	result := map[string]any{}

	manifestNames := []string{".ultra-brain/config.toml", ".brain.toml"}
	for _, name := range manifestNames {
		decoded, ok, err := decode(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if ok {
			delete(decoded, "check")
			result = decoded
			break
		}
	}

	// The file already carries "policy" as its top-level table.
	policy, _, err := decode(filepath.Join(dir, ".ultraloom", "policy.toml"))
	if err != nil {
		return err
	}
	if rules, ok := policy["policy"]; ok {
		result["policy"] = rules
	}

	config, _, err := decode(filepath.Join(dir, ".ultraloom", "config.toml"))
	if err != nil {
		return err
	}
	if worktree, ok := config["worktree"]; ok {
		result["worktree"] = worktree
	}

	answers, _, err := decode(filepath.Join(dir, ".ultraloom", "answers.toml"))
	if err != nil {
		return err
	}
	if bundle, ok := answers["bundle"].(string); ok && !hasWikiLayout(result) {
		layout, _ := result["layout"].(map[string]any)
		if layout == nil {
			layout = map[string]any{}
		}
		layout["wiki"] = strings.TrimSuffix(bundle, "/")
		result["layout"] = layout
	}

	if len(result) == 0 {
		return nil
	}
	if err := write(dir, result); err != nil {
		return err
	}
	return removeOld(dir, manifestNames)
}

// hasWikiLayout answers whether the manifest already named the wiki.
func hasWikiLayout(result map[string]any) bool {
	layout, ok := result["layout"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = layout["wiki"]
	return ok
}

// decode reads one TOML file; a missing file is not an error.
func decode(path string) (map[string]any, bool, error) {
	decoded := map[string]any{}
	if _, err := toml.DecodeFile(path, &decoded); err != nil {
		if os.IsNotExist(err) {
			return decoded, false, nil
		}
		return nil, false, fmt.Errorf("reading %s: %w", path, err)
	}
	return decoded, true, nil
}

// write puts the folded configuration where loomux looks for it.
func write(dir string, result map[string]any) error {
	target := filepath.Join(dir, ".loomux")
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(target, "config.toml"))
	if err != nil {
		return err
	}
	defer file.Close()
	return toml.NewEncoder(file).Encode(result)
}

// removeOld drops the files the translation replaced, and the directories that
// held nothing else.
func removeOld(dir string, manifestNames []string) error {
	old := append([]string{}, manifestNames...)
	for _, name := range []string{"policy.toml", "config.toml", "answers.toml"} {
		old = append(old, filepath.Join(".ultraloom", name))
	}
	for _, name := range old {
		if err := remove(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	// An old directory that still holds something is left standing.
	_ = os.Remove(filepath.Join(dir, ".ultraloom"))
	_ = os.Remove(filepath.Join(dir, ".ultra-brain"))
	return nil
}

// remove deletes path, and says nothing when it was not there.
func remove(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// copyTree copies the recorded case; the recording itself is never touched.
//
//coverage:exempt the filepath.Rel arm needs a path WalkDir found below src that is not below src, and the WalkDir err and ReadFile arms a recording the OS stops handing out while it is being read
func copyTree(src, dst string, skip map[string]bool) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." || skip[rel] {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
