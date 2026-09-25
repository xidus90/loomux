package templates

import (
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/xidus90/loomux/internal/hosts"
)

// allSkills is every skill any module ships.
func allSkills() []string {
	return append(SkillNames("hooks"), SkillNames("brain")...)
}

// shipped is every text a project can get from this package: the skills at
// Claude's location and AGENTS.md for both hosts.
func shipped(t *testing.T) map[string]string {
	t.Helper()
	files, err := Skills(allSkills(), hosts.HostClaude)
	if err != nil {
		t.Fatal(err)
	}
	texts := map[string]string{}
	for _, file := range files {
		texts[file.Path] = file.Text
	}
	agents, err := AgentsMD(Vars{Project: "demo", CommitLanguage: "en",
		Hosts: []hosts.Host{hosts.HostClaude, hosts.HostAntigravity}})
	if err != nil {
		t.Fatal(err)
	}
	texts["AGENTS.md"] = agents
	return texts
}

func TestAgentsMDNamesTheLanguageAndTheProposalRule(t *testing.T) {
	got, err := AgentsMD(Vars{Project: "demo", CommitLanguage: "de", Hosts: []hosts.Host{hosts.HostClaude}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# demo\n",
		"Commit messages are written in `de`.",
		"`.loomux/config.toml` is written by a human, never by an agent.",
		"`loomux config set … --propose`",
		"`CLAUDE.md`",
		"`.claude/settings.json`",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("AGENTS.md lacks %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"GEMINI.md", ".agents/", "uv", "ulguard", ".ultraloom", "shim"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("AGENTS.md names %q:\n%s", unwanted, got)
		}
	}
}

func TestAgentsMDNamesTheAntigravityFilesForAntigravity(t *testing.T) {
	got, err := AgentsMD(Vars{Project: "demo", CommitLanguage: "en", Hosts: []hosts.Host{hosts.HostAntigravity}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "`GEMINI.md` and whatever lives under `.agents/`") {
		t.Errorf("AGENTS.md lacks the Antigravity files:\n%s", got)
	}
	if strings.Contains(got, "CLAUDE.md") {
		t.Errorf("AGENTS.md names CLAUDE.md for an Antigravity-only project:\n%s", got)
	}
}

func TestAgentsMDReportsABrokenTemplate(t *testing.T) {
	for name, fsys := range map[string]fstest.MapFS{
		"missing":   {},
		"unparsed":  {"files/AGENTS.md.tmpl": {Data: []byte("{{ if }")}},
		"unrenders": {"files/AGENTS.md.tmpl": {Data: []byte("{{ .Nowhere }}")}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := agentsMD(fsys, Vars{}); err == nil {
				t.Fatal("no error for a broken template")
			}
		})
	}
}

func TestSkillsGoWhereTheHostLooks(t *testing.T) {
	files, err := Skills(allSkills(), hosts.HostClaude)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 6 {
		t.Fatalf("got %d skills, want 6", len(files))
	}
	for i, name := range allSkills() {
		if want := ".claude/skills/" + name + "/SKILL.md"; files[i].Path != want {
			t.Errorf("path %q, want %q", files[i].Path, want)
		}
		if head := "---\nname: " + name + "\ndescription: "; !strings.HasPrefix(files[i].Text, head) {
			t.Errorf("%s does not start with its frontmatter: %.60q", name, files[i].Text)
		}
	}

	// Antigravity's project skill location is not measured yet: nothing.
	none, err := Skills(allSkills(), hosts.HostAntigravity)
	if err != nil || none != nil {
		t.Errorf("Antigravity got %v, %v; want nil, nil", none, err)
	}
}

func TestSkillsRefuseAnUnknownHostOrName(t *testing.T) {
	if _, err := Skills([]string{"verify-until-green"}, hosts.HostCodex); err == nil {
		t.Error("no error for a host without a skill location")
	}
	if _, err := Skills([]string{"no-such-skill"}, hosts.HostClaude); err == nil {
		t.Error("no error for an unknown skill")
	}
}

func TestSkillNamesPerModule(t *testing.T) {
	if got := SkillNames("hooks"); strings.Join(got, ",") != "verify-until-green" {
		t.Errorf("hooks: %v", got)
	}
	want := "brain-ingest,brain-land,brain-research,brain-review,brain-wiki-plan"
	if got := SkillNames("brain"); strings.Join(got, ",") != want {
		t.Errorf("brain: %v", got)
	}
	if got := SkillNames("graph"); got != nil {
		t.Errorf("graph: %v", got)
	}
}

func TestNoSkillCallsAnOldCommand(t *testing.T) {
	bareBrain := regexp.MustCompile("(?m)(^|`)brain ")
	for path, text := range shipped(t) {
		for _, old := range []string{"uv run", "brain-mcp", "ultraloom", "ulguard"} {
			if strings.Contains(text, old) {
				t.Errorf("%s names %q", path, old)
			}
		}
		if found := bareBrain.FindString(text); found != "" {
			t.Errorf("%s calls a bare brain command: %q", path, found)
		}
	}
}

func TestSkillsAreEnglish(t *testing.T) {
	for path, text := range shipped(t) {
		if i := strings.IndexAny(text, "äöüÄÖÜß"); i >= 0 {
			t.Errorf("%s holds a German letter at byte %d: %.40q", path, i, text[i:])
		}
	}
}
