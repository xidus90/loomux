package setup

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/edit"
	"github.com/xidus90/loomux/internal/config/schema"
)

// rule is one entry of the catalog: the stack that brings it, the table it
// is appended to, the value that tells whether it is already there, and the
// pairs of the block.
type rule struct {
	stack   string
	section string
	key     string // "match" or "regex": what decides presence
	value   string // the glob or the expression itself
	pairs   [][2]string
}

// migrationsGlob matches Django migrations that have been applied.
const migrationsGlob = "**/migrations/[0-9][0-9][0-9][0-9]_*.py"

// pipRegex is the example rule of loomux's own configuration, character for
// character; it holds a backtick, so it is no raw string.
const pipRegex = "(^|[\\n;&|(`])\\s*pip\\s+install([^\\w-]|$)"

// catalog is the rule catalog init writes for a detected stack.
func catalog() []rule {
	return []rule{
		{
			stack: "django", section: "policy.paths.rules", key: "match", value: migrationsGlob,
			pairs: [][2]string{
				{"match", "[" + config.QuoteTOML(migrationsGlob) + "]"},
				{"reason", config.QuoteTOML("Applied migrations are history; add a new one instead.")},
			},
		},
		{
			stack: "uv", section: "policy.commands.rules", key: "regex", value: pipRegex,
			pairs: [][2]string{
				{"regex", "'" + pipRegex + "'"},
				{"reason", config.QuoteTOML("uv, never pip.")},
			},
		},
	}
}

// configText is before with what c and the stacks ask of the configuration:
// [modules] where a module is off, [commit] language where it is not the
// default, and the catalog rules still missing. It edits only through
// edit.Set, edit.Remove and edit.AppendBlock, and hands back nothing the
// loaders would refuse -- neither a before they refuse nor an after.
func configText(before string, stacks []string, c Choice) (string, error) {
	if before != "" {
		if err := schema.Validate(before); err != nil {
			return "", namedConfig(err)
		}
	}
	// Validate accepted the text, so it parses.
	entries, _ := schema.Current(before)
	current := map[string]schema.Entry{}
	for _, e := range entries {
		if e.Origin == schema.Set {
			current[e.Key.ID()] = e
		}
	}
	text := before
	var err error
	for _, m := range []schema.Module{schema.Hooks, schema.Brain, schema.Graph} {
		name := string(m)
		switch e, set := current["modules."+name]; {
		case !c.moduleOn(m) && (!set || e.Input != "false"):
			text, err = edit.Set(text, "modules", name, "false")
		case c.moduleOn(m) && set && e.Input == "false":
			// Only false is ever written; a module switched back on loses
			// its line instead of gaining a true.
			text, err = edit.Remove(text, "modules", name)
		}
		if err != nil {
			return "", namedConfig(err)
		}
	}
	// An empty choice means en, the default: it writes no line, and over a
	// language set to anything else it removes that line.
	want := cmp.Or(c.CommitLanguage, "en")
	lang, set := current["commit.language"]
	switch {
	case want != "en" && lang.Input != want:
		text, err = edit.Set(text, "commit", "language", config.QuoteTOML(want))
	case want == "en" && set && lang.Input != "en":
		text, err = edit.Remove(text, "commit", "language")
	}
	if err != nil {
		return "", namedConfig(err)
	}
	for _, r := range catalog() {
		if slices.Contains(stacks, r.stack) && !hasRule(text, r) {
			text = edit.AppendBlock(text, r.section, r.pairs)
		}
	}
	if text != before {
		if err := schema.Validate(text); err != nil {
			return "", namedConfig(err)
		}
	}
	return text, nil
}

// hasRule says whether text already carries r, decoded rather than
// searched: a rule standing in a comment is no rule.
func hasRule(text string, r rule) bool {
	doc := map[string]any{}
	// The text has passed the loaders or was built from text that did.
	_ = toml.Unmarshal([]byte(text), &doc)
	var node any = doc
	for _, part := range strings.Split(r.section, ".") {
		table, _ := node.(map[string]any)
		node = table[part]
	}
	blocks, _ := node.([]map[string]any)
	for _, block := range blocks {
		switch v := block[r.key].(type) {
		case string:
			if v == r.value {
				return true
			}
		case []any:
			if slices.Contains(v, any(r.value)) {
				return true
			}
		}
	}
	return false
}

// namedConfig makes sure a refusal names the file the human fixes.
func namedConfig(err error) error {
	if strings.Contains(err.Error(), "config.toml") {
		return err
	}
	return fmt.Errorf("%s: %w", configPath, err)
}
