package hooks

import (
	"slices"

	"github.com/xidus90/loomux/internal/config"
)

// BuiltinPathRules are the path rules the guard holds in every project before
// any configuration is read. The caller gets copies it may keep.
func BuiltinPathRules() []config.PathRule {
	rules := slices.Clone(builtinPathRules)
	for i := range rules {
		rules[i].Match = slices.Clone(rules[i].Match)
	}
	return rules
}

// BuiltinCommandRules are the command rules the guard holds in every project
// before any configuration is read.
func BuiltinCommandRules() []config.CommandRule {
	return slices.Clone(builtinCommands())
}
