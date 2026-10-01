package setup

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/setup/hostfile"
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
	for _, rel := range []string{".claude/settings.json", ".githooks/pre-commit", ".githooks/pre-push", ".githooks/commit-msg", mcpPath} {
		if exists(root, rel) {
			t.Errorf("%s written without a binary", rel)
		}
	}
	for _, rel := range []string{"AGENTS.md", ".gitignore"} {
		if !exists(root, rel) {
			t.Errorf("%s not written", rel)
		}
	}
	for _, id := range []string{"binary-install", ".claude/settings.json", ".githooks/pre-commit", "hooks-path", "merge-hook", mcpPath} {
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
		"backup": {map[string]string{backupDir: "file", ".gitignore": "a"}, Plan{Changes: []Change{
			{Path: ".gitignore", Before: "a", After: "b", Exists: true},
		}}},
		"state": {map[string]string{".loomux/state": "file"}, Plan{}},
		"unreadable": {map[string]string{".gitignore/": ""}, Plan{Changes: []Change{
			{Path: ".gitignore", Before: "a", After: "b", Exists: true},
		}}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			root := world(t, tc.files)
			r, err := Apply(root, tc.plan, Choice{}, all, nil, there, "1", applyTime)
			if err == nil {
				t.Error("no error")
			}
			for _, ch := range tc.plan.Changes {
				if !slices.Contains(r.Failed, ch.Path) {
					t.Errorf("failed = %v, lack %s", r.Failed, ch.Path)
				}
			}
		})
	}
}

func TestAFileChangedSinceThePlanIsNotOverwritten(t *testing.T) {
	root := world(t, map[string]string{".gitignore": "a\n"})
	p := Plan{Changes: []Change{{Part: "gitignore", Path: ".gitignore", Before: "a\n", After: "a\nb\n", Exists: true}}}
	writeFile(t, root, ".gitignore", "someone else\n")
	r, err := Apply(root, p, Choice{}, all, nil, there, "1", applyTime)
	if err == nil || !strings.Contains(err.Error(), ".gitignore changed since the plan") {
		t.Fatalf("err = %v, want one naming .gitignore", err)
	}
	if got := read(t, root, ".gitignore"); got != "someone else\n" {
		t.Errorf(".gitignore = %q, want the other writer's text", got)
	}
	if !slices.Contains(r.Failed, ".gitignore") || exists(root, backupDir+"/.gitignore.bak") {
		t.Errorf("failed = %v, backup there = %v; want the failure and no backup",
			r.Failed, exists(root, backupDir+"/.gitignore.bak"))
	}
}

func TestAFailedStepLeavesNoInstalledRecord(t *testing.T) {
	root := world(t, nil)
	p := Plan{Actions: []Action{{Part: "binary", ID: "binary-install"}},
		Changes: []Change{{Part: "gitignore", Path: ".gitignore", After: "x\n"}}}
	fail := func(Action) error { return errors.New("offline") }
	r, err := Apply(root, p, Choice{}, all, fail, there, "1", applyTime)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !slices.Contains(r.Failed, "binary-install") || !exists(root, ".gitignore") {
		t.Errorf("report = %+v; want binary-install failed and .gitignore written", r)
	}
	if exists(root, installedPath) {
		t.Error("installed.toml written by a run in which a step failed")
	}
	if !exists(root, answersPath) {
		t.Error("answers.toml missing; the choices of a run that went through are kept")
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
		!hasNote(p, "merge-hook: skipped; the hook calls ${LOCALAPPDATA}/loomux/bin/loomux.exe, which is not installed; run loomux upgrade") {
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

// In a checkout, Claude's entries call the checkout's binary and
// Antigravity's the installed one; each hook file is written only while the
// binary it calls stands.
func TestEachHookFileIsJudgedByTheBinaryItCalls(t *testing.T) {
	root := world(t, map[string]string{".agents/skills/": "", ".claude/": "", "go.mod": checkoutGoMod})
	// The plan needs the installed binary to plan Antigravity's file at all.
	installBinary(t)
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	full, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	installed := filepath.Join(os.Getenv("LOCALAPPDATA"), "loomux", "bin", "loomux.exe")
	if err := os.Remove(installed); err != nil {
		t.Fatal(err)
	}
	var p Plan
	for _, ch := range full.Changes {
		if ch.Part == "host-entries" {
			p.Changes = append(p.Changes, ch)
		}
	}
	if len(p.Changes) != 2 {
		t.Fatalf("host changes = %v", paths(p))
	}
	none := func(Action) error { return nil }
	// The checkout's binary stands, the installed one does not.
	r, err := Apply(root, p, c, all, none, there, "1", applyTime)
	if err != nil || !slices.Equal(r.Written, []string{".claude/settings.json"}) || !slices.Equal(r.Failed, []string{".agents/hooks.json"}) {
		t.Errorf("checkout only: report %+v, err %v", r, err)
	}
	if err := os.Remove(filepath.Join(root, ".claude", "settings.json")); err != nil {
		t.Fatal(err)
	}
	// The installed binary stands, the checkout's does not.
	installBinary(t)
	r, err = Apply(root, p, c, all, none, func() bool { return false }, "1", applyTime)
	if err != nil || !slices.Equal(r.Written, []string{".agents/hooks.json"}) || !slices.Equal(r.Failed, []string{".claude/settings.json"}) {
		t.Errorf("installed only: report %+v, err %v", r, err)
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

// .mcp.json calls the installed binary whichever binary the entries call,
// so a checkout's bin/loomux.exe does not keep it.
func TestMCPJSONIsJudgedByTheInstalledBinaryAlone(t *testing.T) {
	root := world(t, nil)
	p := Plan{Changes: []Change{{Part: "mcp-json", Path: mcpPath, After: "{}\n", Binary: hostfile.Canonical}}}
	r, err := Apply(root, p, Choice{}, all, nil, there, "1", applyTime)
	if err != nil || exists(root, mcpPath) || !slices.Contains(r.Failed, mcpPath) {
		t.Errorf("err %v, written %v, failed %v; want .mcp.json dropped", err, exists(root, mcpPath), r.Failed)
	}
	installBinary(t)
	if r, err := Apply(root, p, Choice{}, all, nil, func() bool { return false }, "1", applyTime); err != nil || !exists(root, mcpPath) {
		t.Errorf("err %v, report %+v; want .mcp.json written beside the installed binary", err, r)
	}
}

// A change dropped for want of its binary is not put to the human: a yes
// to it would write nothing.
func TestADroppedChangeIsNotAskedFor(t *testing.T) {
	root := world(t, nil)
	p := Plan{Changes: []Change{
		{Part: "mcp-json", Path: mcpPath, After: "{}\n", Binary: hostfile.Canonical},
		{Part: "gitignore", Path: ".gitignore", After: "x\n"},
	}}
	var asked []string
	approve := func(ch Change) bool { asked = append(asked, ch.Path); return true }
	r, err := Apply(root, p, Choice{}, approve, nil, there, "1", applyTime)
	if err != nil || !slices.Equal(asked, []string{".gitignore"}) || !slices.Contains(r.Failed, mcpPath) {
		t.Errorf("asked %v, report %+v, err %v; want only .gitignore asked", asked, r, err)
	}
}

// A file calling the checkout's bin/loomux.exe goes with that binary, which
// binaryThere reports, not with the installed one.
func TestACheckoutFileIsJudgedByTheCheckoutBinary(t *testing.T) {
	root := world(t, nil)
	installBinary(t)
	p := Plan{Changes: []Change{{Part: "host-entries", Path: ".claude/settings.json", After: "{}\n", Binary: hostfile.Checkout}}}
	r, err := Apply(root, p, Choice{}, all, nil, func() bool { return false }, "1", applyTime)
	if err != nil || exists(root, ".claude/settings.json") || !slices.Contains(r.Failed, ".claude/settings.json") {
		t.Errorf("err %v, report %+v; want the entries dropped without bin/loomux.exe", err, r)
	}
	if _, err := Apply(root, p, Choice{}, all, nil, there, "1", applyTime); err != nil || !exists(root, ".claude/settings.json") {
		t.Errorf("err %v; want the entries written beside bin/loomux.exe", err)
	}
}

// modeInfo is a file that stands with a mode, for the stat seam: Windows
// keeps no POSIX permissions to read back.
type modeInfo fs.FileMode

func (m modeInfo) Name() string       { return "" }
func (m modeInfo) Size() int64        { return 0 }
func (m modeInfo) Mode() fs.FileMode  { return fs.FileMode(m) }
func (m modeInfo) ModTime() time.Time { return time.Time{} }
func (m modeInfo) IsDir() bool        { return false }
func (m modeInfo) Sys() any           { return nil }

// A replaced file keeps the permissions it had -- a .mcp.json or
// settings.json a human closed to 0600 may hold tokens -- and a replaced
// script gains the execute bit a new one would have: the swap goes through a
// temporary file whose mode is neither. A new file is not asked: it gets the
// mode of its text when it is created.
func TestAReplacedFileKeepsItsModeAndAScriptGainsItsExecBit(t *testing.T) {
	root := world(t, map[string]string{
		".githooks/pre-commit": "#!/bin/sh\nold\n", ".githooks/pre-push": "#!/bin/sh\nold\n",
		"secret.json": "{}\n", "open.json": "{}\n",
	})
	modes := map[string]fs.FileMode{
		"/.githooks/pre-commit": 0o644, "/.githooks/pre-push": 0o755,
		"/secret.json": 0o600, "/open.json": 0o644,
	}
	rel := func(path string) string { return filepath.ToSlash(strings.TrimPrefix(path, root)) }
	oldStat := stat
	stat = func(path string) (fs.FileInfo, error) { return modeInfo(modes[rel(path)]), nil }
	t.Cleanup(func() { stat = oldStat })
	var got []string
	oldChmod := chmod
	chmod = func(path string, mode fs.FileMode) error {
		got = append(got, rel(path)+" "+mode.String())
		return nil
	}
	t.Cleanup(func() { chmod = oldChmod })
	p := Plan{Changes: []Change{
		{Part: "git-hooks", Path: ".githooks/pre-commit", Before: "#!/bin/sh\nold\n", After: "#!/bin/sh\nnew\n", Exists: true},
		{Part: "git-hooks", Path: ".githooks/pre-push", Before: "#!/bin/sh\nold\n", After: "#!/bin/sh\nnew\n", Exists: true},
		{Part: "git-hooks", Path: ".githooks/commit-msg", After: "#!/bin/sh\nnew\n"},
		{Part: "mcp-json", Path: "secret.json", Before: "{}\n", After: "{\"a\":1}\n", Exists: true},
		{Part: "mcp-json", Path: "open.json", Before: "{}\n", After: "{\"a\":1}\n", Exists: true},
	}}
	if _, err := Apply(root, p, Choice{}, all, func(Action) error { return nil }, there, "1.0.0", time.Now()); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"/.githooks/pre-commit -rwxr-xr-x", "/.githooks/pre-push -rwxr-xr-x",
		"/secret.json -rw-------", "/open.json -rw-r--r--",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("chmod %v, want %v", got, want)
	}
	chmod = func(string, fs.FileMode) error { return errors.New("no chmod") }
	p.Changes[0].Before, p.Changes[0].After = "#!/bin/sh\nnew\n", "#!/bin/sh\nnewer\n"
	if _, err := Apply(root, Plan{Changes: p.Changes[:1]}, Choice{}, all, func(Action) error { return nil }, there, "1.0.0", time.Now()); err == nil || !strings.Contains(err.Error(), ".githooks/pre-commit") {
		t.Fatalf("a chmod that fails: %v", err)
	}
	// A file whose mode cannot be read is not replaced.
	stat = func(string) (fs.FileInfo, error) { return nil, errors.New("no stat") }
	p.Changes[0].Before, p.Changes[0].After = "#!/bin/sh\nnewer\n", "#!/bin/sh\nnewest\n"
	if _, err := Apply(root, Plan{Changes: p.Changes[:1]}, Choice{}, all, func(Action) error { return nil }, there, "1.0.0", time.Now()); err == nil || !strings.Contains(err.Error(), ".githooks/pre-commit: no stat") {
		t.Fatalf("a stat that fails: %v", err)
	}
	if text := read(t, root, ".githooks/pre-commit"); text != "#!/bin/sh\nnewer\n" {
		t.Fatalf("replaced without its mode: %q", text)
	}
}
