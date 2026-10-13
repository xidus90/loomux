package setup

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config/edit"
	"github.com/xidus90/loomux/internal/config/schema"
)

func TestAModuleSwitchedOffIsWrittenAsFalse(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod, ".git/": ""})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	c.Modules[schema.Hooks] = false
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	cfg, ok := changeOf(p, configPath)
	if !ok || cfg.After != "[modules]\nhooks = false\n" {
		t.Errorf("config = %q", cfg.After)
	}
	for _, path := range paths(p) {
		if path == ".claude/settings.json" || strings.HasPrefix(path, ".githooks/") || strings.Contains(path, "verify-until-green") {
			t.Errorf("hooks off still writes %s", path)
		}
	}
	if slices.Contains(actions(p), "hooks-path") {
		t.Errorf("hooks off still sets core.hooksPath")
	}

	// Switched back on, the line goes and nothing takes its place.
	writeFile(t, root, configPath, cfg.After)
	f = gather(t, root, "")
	c = DefaultChoice(f, Answers{})
	if c.Modules[schema.Hooks] {
		t.Fatalf("the default choice lost hooks = false")
	}
	c.Modules[schema.Hooks] = true
	p, err = Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	if cfg, _ := changeOf(p, configPath); cfg.After != "" {
		t.Errorf("all modules on leaves %q", cfg.After)
	}
}

func TestTheCommitLanguageIsWrittenOnlyWhenItIsNotTheDefault(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	if c.CommitLanguage != "en" {
		t.Fatalf("default language = %q", c.CommitLanguage)
	}
	p, _ := Build(f, c, reader(root))
	if _, ok := changeOf(p, configPath); ok {
		t.Errorf("en writes the configuration")
	}
	c.CommitLanguage = "de"
	p, _ = Build(f, c, reader(root))
	if cfg, _ := changeOf(p, configPath); cfg.After != "[commit]\nlanguage = \"de\"\n" {
		t.Errorf("de writes %q", cfg.After)
	}

	// Back to en from a file that says de: the line goes.
	writeFile(t, root, configPath, "[commit]\nlanguage = \"de\"\n")
	f = gather(t, root, "")
	c = DefaultChoice(f, Answers{})
	c.CommitLanguage = "en"
	p, _ = Build(f, c, reader(root))
	if cfg, _ := changeOf(p, configPath); cfg.After != "" {
		t.Errorf("en over de leaves %q", cfg.After)
	}
}

func TestTheRuleCatalogFollowsTheStacks(t *testing.T) {
	for _, tc := range []struct {
		name, marker, want string
	}{
		{"django", "manage.py", `match = ["**/migrations/[0-9][0-9][0-9][0-9]_*.py"]`},
		{"uv", "uv.lock", "regex = '" + pipRegex + "'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := world(t, map[string]string{tc.marker: ""})
			f := gather(t, root, "")
			cfg, ok := changeOf(plan(t, f), configPath)
			if !ok || !strings.Contains(cfg.After, tc.want) {
				t.Fatalf("config =\n%s", cfg.After)
			}
			writeFile(t, root, configPath, cfg.After)
			f = gather(t, root, "")
			if again, ok := changeOf(plan(t, f), configPath); ok {
				t.Errorf("a second run adds again:\n%s", again.After)
			}
		})
	}
}

func TestARuleInACommentIsNoRule(t *testing.T) {
	text := "# [[policy.commands.rules]]\n# regex = '" + pipRegex + "'\n"
	root := world(t, map[string]string{"uv.lock": "", configPath: text})
	cfg, _ := changeOf(plan(t, gather(t, root, "")), configPath)
	if !strings.Contains(cfg.After, "\n[[policy.commands.rules]]\n") {
		t.Errorf("config =\n%s", cfg.After)
	}
}

func TestAConfigTheLoadersRejectIsNotTouched(t *testing.T) {
	root := world(t, map[string]string{configPath: "[commit]\nthreshold = \"x\"\n"})
	_, err := Build(gather(t, root, ""), DefaultChoice(gather(t, root, ""), Answers{}), reader(root))
	if err == nil || !strings.Contains(err.Error(), "config.toml") {
		t.Fatalf("err = %v", err)
	}
}

func TestAnEditThatCannotBePlacedStopsThePlan(t *testing.T) {
	// An inline table is valid TOML the loaders accept, and edit refuses to
	// guess where a key inside it goes.
	_, err := configText("modules = { graph = true }\n", nil, Choice{Modules: map[schema.Module]bool{schema.Graph: false}})
	if err == nil || !strings.HasPrefix(err.Error(), configPath+": ") {
		t.Errorf("err = %v", err)
	}
}

func TestAnEmptyCommitLanguageIsTheDefault(t *testing.T) {
	text, err := configText("", nil, Choice{})
	if err != nil || text != "" {
		t.Errorf("text = %q, err = %v", text, err)
	}
	text, err = configText("[commit]\nlanguage = \"de\"\n", nil, Choice{})
	if err != nil || text != "" {
		t.Errorf("over de: text = %q, err = %v", text, err)
	}
}

func TestALanguageTheLoadersRejectIsNotWritten(t *testing.T) {
	_, err := configText("", nil, Choice{CommitLanguage: "fr"})
	if err == nil || !strings.Contains(err.Error(), "config.toml") {
		t.Fatalf("err = %v", err)
	}
	_, err = configText("commit = { language = \"de\" }\n", nil, Choice{CommitLanguage: "en"})
	if err == nil {
		t.Errorf("an ambiguous [commit] is edited")
	}
}

// Rules written as an inline array are the form the getting-started guide
// shows; init adds its rule into that array and keeps the human's lines.
func TestTheCatalogRuleJoinsAnInlineRulesArray(t *testing.T) {
	const pipRule = "{ regex = '" + pipRegex + "', reason = \"uv, never pip.\" }"
	for _, tc := range []struct {
		name, stack, before, after string
	}{
		{
			"commands", "uv",
			"[policy.commands]\n# Block\nrules = [\n  { regex = 'git', reason = \"no\" }  # ask first\n]\n\n[commit]\nthreshold = 2\n",
			"[policy.commands]\n# Block\nrules = [\n  { regex = 'git', reason = \"no\" },  # ask first\n  " + pipRule + "\n]\n\n[commit]\nthreshold = 2\n",
		},
		{
			"paths", "django",
			"[policy.paths]\nrules = [\n  { match = [\"go.sum\"], reason = \"x\" },\n]\n",
			"[policy.paths]\nrules = [\n  { match = [\"go.sum\"], reason = \"x\" },\n  { match = [\"" + migrationsGlob + "\"], reason = \"Applied migrations are history; add a new one instead.\" },\n]\n",
		},
		{
			"empty", "uv",
			"[policy.commands]\nrules = []\n",
			"[policy.commands]\nrules = [" + pipRule + "]\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := configText(tc.before, []string{tc.stack}, Choice{})
			if err != nil || got != tc.after {
				t.Fatalf("got %q, %v\nwant %q", got, err, tc.after)
			}
			// The rule just added is found again: a second run adds nothing.
			if again, err := configText(got, []string{tc.stack}, Choice{}); err != nil || again != got {
				t.Errorf("a second run changes it: %q, %v", again, err)
			}
		})
	}
}

func TestAnInlineRuleThatIsThereIsNotAddedAgain(t *testing.T) {
	text := "[policy.commands]\nrules = [{ regex = '" + pipRegex + "', reason = \"mine\" }]\n"
	got, err := configText(text, []string{"uv"}, Choice{})
	if err != nil || got != text {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestInlineRulesThatCannotBePlacedStopTheRun(t *testing.T) {
	// A dotted key is valid TOML the loaders accept; edit will not guess
	// which table it belongs to.
	_, err := configText("policy.commands.rules = []\n", []string{"uv"}, Choice{})
	if !errors.Is(err, edit.ErrAmbiguous) || !strings.HasPrefix(err.Error(), configPath+": ") {
		t.Errorf("err = %v", err)
	}
}
