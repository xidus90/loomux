package setup

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

var applyTime = time.Date(2026, 9, 24, 18, 0, 0, 0, time.UTC)

func all(Change) bool { return true }

func there() bool { return true }

func exists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestApplyWritesInOrderAndStateLast(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod, ".git/": ""})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	run := func(a Action) error {
		order = append(order, a.ID)
		if exists(root, installedPath) {
			t.Errorf("%s: installed.toml is there already", a.ID)
		}
		// area-add goes before the files: it declares the area only into a
		// configuration that is not there yet.
		if got, want := exists(root, ".gitignore"), a.ID != "binary-install" && a.ID != "area-add"; got != want {
			t.Errorf("%s: .gitignore there = %v, want %v", a.ID, got, want)
		}
		return nil
	}
	installBinary(t)
	r, err := Apply(root, p, c, all, run, there, "2.12.1", applyTime)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"binary-install", "area-add", "hooks-path", "merge-hook", "graph-build"}
	if !slices.Equal(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
	if len(r.Written) != len(p.Changes)+len(want) || len(r.Failed)+len(r.Skipped)+len(r.Refused) != 0 {
		t.Errorf("report = %+v", r)
	}
	installed := read(t, root, installedPath)
	for _, s := range []string{`version = "2.12.1"`, "at = 2026-09-24T18:00:00Z", `".githooks/pre-commit"`, `"graph-build"`} {
		if !strings.Contains(installed, s) {
			t.Errorf("installed.toml lacks %s:\n%s", s, installed)
		}
	}
	a, err := ReadAnswers(root)
	if err != nil || !slices.Equal(a.Hosts, []string{"claude"}) || len(a.Parts) != len(c.Parts) || !a.Parts["git-hooks"] {
		t.Errorf("answers = %+v, err = %v", a, err)
	}
}

func TestAMissingBinaryDropsWhatCallsIt(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod, ".git/": ""})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	run := func(a Action) error {
		order = append(order, a.ID)
		if a.ID == "binary-install" {
			return errors.New("no network")
		}
		return nil
	}
	r, err := Apply(root, p, c, all, run, func() bool { return false }, "2.12.1", applyTime)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"binary-install", "area-add", "graph-build"}; !slices.Equal(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
	for _, rel := range []string{".claude/settings.json", ".githooks/pre-commit", ".githooks/pre-push", ".githooks/commit-msg"} {
		if exists(root, rel) {
			t.Errorf("%s written without a binary", rel)
		}
	}
	for _, rel := range []string{"AGENTS.md", ".gitignore"} {
		if !exists(root, rel) {
			t.Errorf("%s not written", rel)
		}
	}
	for _, id := range []string{"binary-install", ".claude/settings.json", ".githooks/pre-commit", "hooks-path", "merge-hook"} {
		if !slices.Contains(r.Failed, id) {
			t.Errorf("failed = %v, lacks %s", r.Failed, id)
		}
	}
}

func TestADeclinedChangeIsNotWrittenAndNotAnError(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	approve := func(ch Change) bool { return ch.Path != ".gitignore" }
	r, err := Apply(root, p, c, approve, func(Action) error { return nil }, there, "2.12.1", applyTime)
	if err != nil || !slices.Equal(r.Refused, []string{".gitignore"}) {
		t.Errorf("refused = %v, err = %v", r.Refused, err)
	}
	if exists(root, ".gitignore") || !exists(root, "AGENTS.md") {
		t.Error("the declined file is there, or the approved one is not")
	}
}

func TestAnExistingFileIsBackedUpOnceBeforeItChanges(t *testing.T) {
	root := world(t, map[string]string{".gitignore": "a\n", configPath: "[modules]\n"})
	first := Plan{Changes: []Change{
		{Part: "gitignore", Path: ".gitignore", Before: "a\n", After: "a\nb\n", Exists: true},
		{Part: "config", Path: configPath, Before: "[modules]\n", After: "[modules]\nbrain = false\n", Exists: true},
	}}
	if _, err := Apply(root, first, Choice{}, all, nil, there, "1", applyTime); err != nil {
		t.Fatal(err)
	}
	second := Plan{Changes: []Change{{Part: "gitignore", Path: ".gitignore", Before: "a\nb\n", After: "c\n", Exists: true}}}
	if _, err := Apply(root, second, Choice{}, all, nil, there, "1", applyTime); err != nil {
		t.Fatal(err)
	}
	if got := read(t, root, backupDir+"/.gitignore.bak"); got != "a\n" {
		t.Errorf("backup = %q, want the first text", got)
	}
	if got := read(t, root, ".gitignore"); got != "c\n" {
		t.Errorf(".gitignore = %q", got)
	}
	if got := read(t, root, configPath); got != "[modules]\nbrain = false\n" {
		t.Errorf("config = %q", got)
	}
	if exists(root, backupDir+"/"+configPath+".bak") {
		t.Error("the configuration was backed up")
	}
}

func TestAnInterruptedRunIsOpenOnTheNextOne(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod, ".git/": ""})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	run := func(a Action) error {
		if a.ID == "merge-hook" {
			return errors.New("hooks locked")
		}
		return nil
	}
	installBinary(t)
	r, err := Apply(root, p, c, all, run, there, "2.12.1", applyTime)
	if err == nil || !strings.Contains(err.Error(), "merge-hook") || !slices.Contains(r.Failed, "merge-hook") {
		t.Errorf("err = %v, failed = %v", err, r.Failed)
	}
	if exists(root, installedPath) || exists(root, answersPath) {
		t.Error("an interrupted run wrote its state")
	}
	next := plan(t, gather(t, root, ""))
	if !slices.Contains(actions(next), "merge-hook") {
		t.Errorf("actions = %v, lack merge-hook", actions(next))
	}
	for _, c := range p.Changes {
		if slices.Contains(paths(next), c.Path) {
			t.Errorf("%s is planned again", c.Path)
		}
	}
}

func TestSkippedFilesAndAQuietRun(t *testing.T) {
	root := world(t, map[string]string{"AGENTS.md": "mine\n"})
	p := Plan{Changes: []Change{{Part: "agents-md", Path: "AGENTS.md", After: "ours\n"}}}
	r, err := Apply(root, p, Choice{}, all, nil, there, "1", applyTime)
	if err != nil || !slices.Equal(r.Skipped, []string{"AGENTS.md"}) || len(r.Written) != 0 {
		t.Errorf("report = %+v, err = %v", r, err)
	}
	if read(t, root, "AGENTS.md") != "mine\n" || exists(root, installedPath) || !exists(root, answersPath) {
		t.Error("a run that did nothing wrote installed.toml, or no answers, or overwrote a file")
	}
}

func TestApplyStopsAtAFailedWrite(t *testing.T) {
	cases := map[string]struct {
		files map[string]string
		plan  Plan
	}{
		"bad name": {nil, Plan{Changes: []Change{{Path: "a:b", After: "x"}}}},
		"missing directory": {nil, Plan{Changes: []Change{
			{Path: configPath, Before: "a", After: "b", Exists: true},
		}}},
		"backup": {map[string]string{backupDir: "file"}, Plan{Changes: []Change{
			{Path: ".gitignore", Before: "a", After: "b", Exists: true},
		}}},
		"state": {map[string]string{".loomux/state": "file"}, Plan{}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			root := world(t, tc.files)
			if _, err := Apply(root, tc.plan, Choice{}, all, nil, there, "1", applyTime); err == nil {
				t.Error("no error")
			}
		})
	}
}

func TestAreaAddRunsFirstAndTheConfigurationIsMadeOverIt(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod, ".git/": ""})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	c.CommitLanguage = "de"
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	if !hasNote(p, "area-add writes [area], [layout], [index], [privacy] and [maintenance] first") {
		t.Errorf("notes = %v", p.Notes)
	}
	const declaration = "[area]\nscope = \"project/demo\"\n\n[maintenance]\non_merge = true\n"
	const rule = "## rule\n"
	run := func(a Action) error {
		if a.ID == "area-add" {
			writeFile(t, root, configPath, declaration)
			writeFile(t, root, "AGENTS.md", rule)
		}
		return nil
	}
	r, err := Apply(root, p, c, all, run, there, "1", applyTime)
	if err != nil {
		t.Fatal(err)
	}
	cfg := read(t, root, configPath)
	if !strings.HasPrefix(cfg, declaration) || !strings.Contains(cfg, "[commit]\nlanguage = \"de\"") {
		t.Errorf("config =\n%s", cfg)
	}
	agents := read(t, root, "AGENTS.md")
	if !strings.HasSuffix(agents, "\n\n"+rule) || strings.HasPrefix(agents, rule) {
		t.Errorf("AGENTS.md =\n%s", agents)
	}
	if exists(root, backupDir+"/AGENTS.md.bak") || !slices.Contains(r.Written, configPath) {
		t.Errorf("a file this run made was backed up, or the config not reported: %+v", r)
	}
	// Once area add has made what the change asked for, nothing is left.
	again := Plan{Changes: []Change{{Part: "config", Path: configPath, After: "x", Redo: func(string) (string, error) { return cfg, nil }}}}
	if r, err := Apply(root, again, c, all, nil, there, "1", applyTime); err != nil || len(r.Written) != 0 {
		t.Errorf("report = %+v, err = %v", r, err)
	}
}

func TestARedoThatFailsStopsTheRun(t *testing.T) {
	root := world(t, map[string]string{configPath: "[modules]\n"})
	refuse := func(string) (string, error) { return "", errors.New("refused") }
	for name, ch := range map[string]Change{
		"redo": {Path: configPath, After: "x", Redo: refuse},
		"read": {Path: "a\x00b", After: "x", Redo: refuse},
	} {
		if _, err := Apply(root, Plan{Changes: []Change{ch}}, Choice{}, all, nil, there, "1", applyTime); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestAFailedAreaAddStopsBeforeTheFiles(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod, ".git/": ""})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	p := plan(t, f)
	run := func(a Action) error {
		if a.ID == "area-add" {
			return errors.New("registry locked")
		}
		return nil
	}
	r, err := Apply(root, p, c, all, run, there, "1", applyTime)
	if err == nil || !strings.Contains(err.Error(), "area-add") || exists(root, ".gitignore") {
		t.Errorf("err = %v, report = %+v", err, r)
	}
}

// installBinary puts the binary where binary-install puts it, the one the
// merge hook calls.
func installBinary(t *testing.T) {
	t.Helper()
	writeFile(t, os.Getenv("LOCALAPPDATA"), "loomux/bin/loomux.exe", "binary")
}

func TestTheMergeHookNeedsTheInstalledBinary(t *testing.T) {
	root := world(t, map[string]string{".git/": "", "go.mod": checkoutGoMod})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	c.Parts["merge-hook"], c.Parts["area"] = true, true
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	if f.CanonicalThere || slices.Contains(actions(p), "merge-hook") ||
		!hasNote(p, "merge-hook: skipped; the hook calls ${LOCALAPPDATA}/loomux/bin/loomux.exe, which is not installed; run loomux self-update") {
		t.Errorf("actions = %v, notes = %v", actions(p), p.Notes)
	}
	// Apply drops it too while the installed binary is missing, whatever
	// binary the entries call.
	var ran []string
	r, err := Apply(root, Plan{Actions: []Action{{Part: "merge-hook", ID: "merge-hook"}}}, c, all,
		func(a Action) error { ran = append(ran, a.ID); return nil }, there, "1", applyTime)
	if err != nil || len(ran) != 0 || !slices.Equal(r.Failed, []string{"merge-hook"}) {
		t.Errorf("ran %v, report %+v, err %v", ran, r, err)
	}
	installBinary(t)
	if f := gather(t, root, ""); !f.CanonicalThere {
		t.Error("the installed binary is not found")
	}
}

func TestTheMergeHookIsJudgedByTheInstalledBinaryAlone(t *testing.T) {
	root := world(t, map[string]string{})
	p := Plan{Actions: []Action{{Part: "merge-hook", ID: "merge-hook"}}}
	missing := func() bool { return false }
	var ran []string
	run := func(a Action) error { ran = append(ran, a.ID); return nil }
	// The checkout's binary is missing, the installed one stands.
	installBinary(t)
	if r, err := Apply(root, p, Choice{}, all, run, missing, "1", applyTime); err != nil || !slices.Equal(ran, []string{"merge-hook"}) {
		t.Errorf("ran %v, report %+v, err %v", ran, r, err)
	}
	if err := os.Remove(filepath.Join(os.Getenv("LOCALAPPDATA"), "loomux", "bin", "loomux.exe")); err != nil {
		t.Fatal(err)
	}
	ran = nil
	if r, err := Apply(root, p, Choice{}, all, run, missing, "1", applyTime); err != nil || len(ran) != 0 || !slices.Equal(r.Failed, []string{"merge-hook"}) {
		t.Errorf("ran %v, report %+v, err %v", ran, r, err)
	}
}
