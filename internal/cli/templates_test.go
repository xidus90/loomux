package cli

import (
	"regexp"
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/setup/templates"
)

// TestEveryLoomuxCommandInASkillExists holds the shipped texts to the command
// table: a skill that tells an agent to run `loomux <word>` names a command
// this binary has. It stands here because templates may not import cli.
func TestEveryLoomuxCommandInASkillExists(t *testing.T) {
	names := append(templates.SkillNames("hooks"), templates.SkillNames("brain")...)
	files, err := templates.Skills(names, hosts.HostClaude)
	if err != nil {
		t.Fatal(err)
	}
	agents, err := templates.AgentsMD(templates.Vars{Project: "demo", CommitLanguage: "en",
		Hosts: []hosts.Host{hosts.HostClaude, hosts.HostAntigravity}})
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, templates.File{Path: "AGENTS.md", Text: agents})

	// The commands whose first argument is a subcommand; for them the second
	// word must be one they take, or `loomux brain lookup` would pass.
	subcommands := map[string][]string{
		"brain":  append(slices.Clone(brainSubcommands), "check"),
		"config": {"list", "get", "set", "unset", "proposals", "apply", "reject"},
		"wiki":   {"init", "types", "retype"},
	}
	call := regexp.MustCompile(`\bloomux ([a-z][a-z-]*)(?: ([a-z][a-z-]*))?`)
	seen := 0
	for _, file := range files {
		for _, match := range call.FindAllStringSubmatch(file.Text, -1) {
			seen++
			if _, ok := commands[match[1]]; !ok {
				t.Errorf("%s calls %q, which is no loomux command", file.Path, match[0])
			}
			if subs, ok := subcommands[match[1]]; ok && !slices.Contains(subs, match[2]) {
				t.Errorf("%s calls %q, which is no subcommand of loomux %s", file.Path, match[0], match[1])
			}
		}
	}
	if seen == 0 {
		t.Fatal("no loomux call found; the pattern no longer matches the texts")
	}
}
