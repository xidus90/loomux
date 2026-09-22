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
	"github.com/xidus90/loomux/internal/config"
)

// Rule rewrites the head of a recorded command line.
type Rule struct {
	From string
	To   string
}

// ExitRule maps a recorded exit code onto the one loomux answers with.
//
// Written down here and not approved per case in the suite: an approved case
// only has to differ, so any wrong code would pass for the deviation. Mapped,
// the translated case names one code and the suite holds it to it.
type ExitRule struct {
	From int
	To   int
}

// Mapping is the set of rules, read from a file of [[command]], [[tool]] and
// [[exit]] tables. A stage brings one command kind or the other: a command
// line is rewritten at its head, an MCP call at its tool's name.
type Mapping struct {
	Commands []Rule     `toml:"command"`
	Tools    []Rule     `toml:"tool"`
	Exits    []ExitRule `toml:"exit"`
}

// mapExit answers the code a translated case expects.
func mapExit(code int, rules []ExitRule) int {
	for _, rule := range rules {
		if rule.From == code {
			return rule.To
		}
	}
	return code
}

// The old tools named their configuration in four places; loomux has one.
var pathLiterals = []string{".ultra-brain/config.toml", ".ultraloom/policy.toml", ".brain.toml"}

const loomuxConfig = ".loomux/config.toml"

// Import copies every case below from into to, rewriting its command, its
// payload and its staged worlds on the way.
//
// The target is pruned first, so the corpus holds the recordings and nothing
// else. The import only ever wrote, so a case dropped from the recordings
// survived in the translated corpus -- and a pinned case count, which counts
// what is there, cannot tell that from a case that was merely replaced. The
// recordings are read before anything is removed: a source that cannot be read
// must not cost the corpus.
func Import(from, to string, m Mapping) error {
	found, err := cases.DiscoverCases(from, "")
	if err != nil {
		return err
	}
	if err := prune(to, found); err != nil {
		return err
	}
	for _, c := range found {
		out := filepath.Join(to, c.Verb, c.Name)
		// The worlds are staged afresh: a file an earlier import added there,
		// such as a merged fixture the recording never had, would otherwise
		// outlive it and be merged into again.
		for _, world := range []string{"world", "world_after"} {
			if err := os.RemoveAll(filepath.Join(out, world)); err != nil {
				return fmt.Errorf("clearing %s: %w", filepath.Join(out, world), err)
			}
		}
		// The files this import rewrites are not copied first: one writer per
		// file keeps the copy from being the one that fails.
		mapped := mapExit(c.ExitCode, m.Exits)
		skip := map[string]bool{"cmd": true, "stdin": len(c.Stdin) > 0, "exit": mapped != c.ExitCode}
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
		if mapped != c.ExitCode {
			if err := os.WriteFile(filepath.Join(out, "exit"), fmt.Appendf(nil, "%d\n", mapped), 0o644); err != nil {
				return err
			}
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

// prune removes every case in the target that no recording backs.
//
// The target is read with the same discovery the corpus itself is read with,
// so what counts as a case here is what counts as one everywhere else. Only
// case directories are touched, never the target as a whole: a `--to` with a
// typo in it would otherwise take a directory with it, and a stray file beside
// the corpus is nobody's evidence to delete. A verb directory that loses its
// last case goes with it; os.Remove says nothing about a directory that still
// holds something, which is the answer wanted here.
func prune(to string, found []*cases.Case) error {
	backed := map[string]bool{}
	for _, c := range found {
		backed[filepath.Join(c.Verb, c.Name)] = true
	}
	existing, err := cases.DiscoverCases(to, "")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading the corpus in %s: %w", to, err)
	}
	places := map[string]string{}
	for _, c := range existing {
		places[filepath.Join(c.Verb, c.Name)] = c.Path
	}
	return prunePaths(backed, places)
}

// prunePaths removes every case directory in places that backed does not name.
// It is shared with the MCP import: what a case is differs between the two
// corpora, what an unbacked one costs does not.
func prunePaths(backed map[string]bool, places map[string]string) error {
	for key, path := range places {
		if backed[key] {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("clearing %s: %w", path, err)
		}
		_ = os.Remove(filepath.Dir(path))
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

// TranslateWorld rewrites the configuration files of the old tools in dir, in
// every dir/areas/<name> and in every directory dir/registry.toml names as
// {{WORLD}}/<path>, into one .loomux/config.toml each. It also folds the old
// hooks' session state and their no-verify marker into loomux's layout.
func TranslateWorld(dir string) error {
	// First: the old hooks' state lives under .ultraloom, which translateDir
	// removes once it has nothing else in it.
	if err := foldHookState(dir); err != nil {
		return err
	}
	if err := translateDir(dir); err != nil {
		return err
	}
	// A world without areas/ has no read-only area to fold.
	areas, _ := os.ReadDir(filepath.Join(dir, "areas"))
	for _, area := range areas {
		if !area.IsDir() {
			continue
		}
		if err := translateDir(filepath.Join(dir, "areas", area.Name())); err != nil {
			return err
		}
	}
	// A registered directory that is also the root or an areas/ entry was
	// folded above; translateDir finds no old file there and writes nothing.
	for _, area := range registeredDirs(dir) {
		if err := translateDir(area); err != nil {
			return err
		}
	}
	return nil
}

// registeredDirs names the directories the world's registry places inside the
// world. A writable area lives where its path says, not only under areas/, and
// its manifest is read where it lives. A path outside the world is no part of
// the recording.
func registeredDirs(dir string) []string {
	areas, err := config.ReadRegistry(dir)
	if err != nil {
		// A missing or unreadable registry names nothing to fold. The replay
		// reads the same file and reports it, as the recording did.
		return nil
	}
	var dirs []string
	for _, area := range areas {
		rest, ok := strings.CutPrefix(area.Path, cases.WorldToken+"/")
		if !ok || !filepath.IsLocal(filepath.FromSlash(rest)) {
			continue
		}
		dirs = append(dirs, filepath.Join(dir, filepath.FromSlash(rest)))
	}
	return dirs
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
	if commit, ok := config["commit"]; ok {
		result["commit"] = commit
	}
	if raw, ok := config["verify"]; ok {
		configPath := filepath.Join(dir, ".ultraloom", "config.toml")
		old, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: [verify] must be a table", configPath)
		}
		folded, err := foldVerify(dir, old)
		if err != nil {
			return fmt.Errorf("%s: %w", configPath, err)
		}
		if len(folded) > 0 {
			result["verify"] = folded
		}
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
