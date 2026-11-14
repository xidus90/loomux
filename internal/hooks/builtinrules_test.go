package hooks

import "testing"

func TestBuiltinRulesAreReadableAndTheirOwn(t *testing.T) {
	paths := BuiltinPathRules()
	if len(paths) != len(builtinPathRules) || paths[0].Reason == "" || len(paths[0].Match) == 0 {
		t.Fatalf("BuiltinPathRules() = %v, want the guard's %d rules", paths, len(builtinPathRules))
	}
	was := builtinPathRules[0].Match[0]
	paths[0].Match[0] = "changed"
	if builtinPathRules[0].Match[0] != was {
		t.Error("a caller's change reached the guard's own rule")
	}
	commands := BuiltinCommandRules()
	if len(commands) != len(builtinCommands()) || !commands[0].Regex.MatchString("git push") {
		t.Fatalf("BuiltinCommandRules() = %v, want the guard's rules", commands)
	}
}
