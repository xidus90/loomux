package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/selfupdate"
	"github.com/xidus90/loomux/internal/setup"
	"github.com/xidus90/loomux/internal/setup/hostfile"
	"github.com/xidus90/loomux/internal/tui"
	"github.com/xidus90/loomux/internal/verify"
)

// initSeams records what init handed to its seams.
type initSeams struct {
	installed, built int
	actions          []string
	pulls            []string // the models asked for at /api/pull
}

// useOllama points the global configuration of the state directory at a
// fake Ollama that answers /api/tags with tags and /api/pull with the lines
// of pull, recording each pull in s.
func useOllama(t *testing.T, s *initSeams, tags, pull string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/pull" {
			var body struct{ Model string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			s.pulls = append(s.pulls, body.Model)
			_, _ = io.WriteString(w, pull+"\n")
			return
		}
		_, _ = io.WriteString(w, tags)
	}))
	t.Cleanup(server.Close)
	writeAt(t, filepath.Join(config.StateDir(), "config.toml"), "[model]\nendpoint = \""+server.URL+"\"\n")
}

// initWorld is a fresh repository named demo, with LOCALAPPDATA, the state
// directory under it, the home and git's user configuration in throwaway
// places, and every seam replaced: the install puts a file where the
// entries look, no console is there, and the subcommands are recorded;
// area add then runs for real without its index run, the rest does not.
func initWorld(t *testing.T) (string, *initSeams) {
	t.Helper()
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	t.Setenv("LOOMUX_STATE_DIR", filepath.Join(local, "loomux"))
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	global := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(global, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	root := filepath.Join(t.TempDir(), "demo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	mustRunGit(t, root, "init", "-q")
	s := &initSeams{}
	replace(t, &installBinary, func(context.Context) selfupdate.Result {
		s.installed++
		writeAt(t, selfupdate.Canonical(filepath.Join(local, "loomux")), "binary")
		return selfupdate.Result{Outcome: selfupdate.Updated, Version: "9.9.9"}
	})
	replace(t, &buildCheckout, func(root string) error {
		s.built++
		writeAt(t, filepath.Join(root, "bin", "loomux.exe"), "binary")
		return nil
	})
	replace(t, &runAction, func(name string, args []string, stdout, stderr io.Writer) int {
		s.actions = append(s.actions, name+" "+strings.Join(args, " "))
		if name == "area" {
			return runSubcommand(name, append(args, "--no-reindex"), stdout, stderr)
		}
		return 0
	})
	replace(t, &openTerminal, func() (tui.Terminal, func() error, error) {
		return nil, nil, errors.New("not a terminal")
	})
	// No init of a test may reach the machine's Ollama: the global
	// configuration points at a fake one that has the default model.
	useOllama(t, s, `{"models":[{"name":"`+config.DefaultModelName+`"}]}`, `{"status":"success"}`)
	// A release init, whose installed binary, wherever one stands, is as new.
	replace(t, &Version, "2.13.0")
	replace(t, &binaryVersion, func(path string) string {
		if _, err := os.Stat(path); err == nil {
			return Version
		}
		return ""
	})
	return root, s
}

func TestInstalledVersionReadsTheBinarysAnswer(t *testing.T) {
	replace(t, &versionRunner, func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "bin" || strings.Join(args, " ") != "--version" {
			t.Errorf("ran %s %v", name, args)
		}
		return []byte("loomux 2.13.0 (beta)\n"), nil
	})
	if got := installedVersion("bin"); got != "2.13.0" {
		t.Fatalf("installedVersion = %q, want 2.13.0", got)
	}
}

func TestInstalledVersionGivesUpOnAHungBinary(t *testing.T) {
	replace(t, &versionDeadline, 10*time.Millisecond)
	replace(t, &versionRunner, func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Minute):
			return []byte("loomux 2.13.0\n"), nil
		}
	})
	start := time.Now()
	if got := installedVersion("bin"); got != "" {
		t.Fatalf("installedVersion = %q, want none", got)
	}
	if took := time.Since(start); took > 10*time.Second {
		t.Fatalf("took %s, the deadline did not hold", took)
	}
}

// replace sets *seam to v for the test.
func replace[T any](t *testing.T, seam *T, v T) {
	t.Helper()
	old := *seam
	*seam = v
	t.Cleanup(func() { *seam = old })
}

// withTerminal makes term the console of every session.
func withTerminal(t *testing.T, term tui.Terminal, restoreErr error) {
	t.Helper()
	replace(t, &openTerminal, func() (tui.Terminal, func() error, error) {
		return term, func() error { return restoreErr }, nil
	})
}

func writeAt(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func there(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

func readAt(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// treeHash hashes every file under root with its path.
func treeHash(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		fmt.Fprintf(h, "%s\x00%x\x00", path, sha256.Sum256(data))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func TestInitDetectOnlyPrintsTheFacts(t *testing.T) {
	root, _ := initWorld(t)
	before := treeHash(t, root)
	code, out, errOut := run("init", "--detect-only", "--root", root)
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	var facts map[string]any
	if err := json.Unmarshal([]byte(out), &facts); err != nil {
		t.Fatalf("%v:\n%s", err, out)
	}
	for _, key := range []string{"Hosts", "Detect", "HooksPath"} {
		if _, ok := facts[key]; !ok {
			t.Errorf("no %s in\n%s", key, out)
		}
	}
	if !strings.Contains(out, "\n  \"Root\"") {
		t.Errorf("not indented by two spaces:\n%s", out)
	}
	if treeHash(t, root) != before {
		t.Error("detect-only wrote")
	}
}

func TestInitDryRunShowsEveryChangeAndWritesNothing(t *testing.T) {
	for _, args := range [][]string{{"--dry-run"}, {"--dry-run", "--yes"}} {
		root, s := initWorld(t)
		before := treeHash(t, root)
		code, out, errOut := run(append([]string{"init", "--root", root}, args...)...)
		if code != 0 {
			t.Fatalf("%v: code %d: %s", args, code, errOut)
		}
		for _, want := range []string{"--- .claude/settings.json", "--- .githooks/pre-commit", "--- .loomux/armed.toml", "binary-install", "+"} {
			if !strings.Contains(out, want) {
				t.Errorf("%v: no %q in\n%s", args, want, out)
			}
		}
		if treeHash(t, root) != before || s.installed+len(s.actions) != 0 {
			t.Errorf("%v: dry run changed something: %+v", args, s)
		}
	}
}

func TestInitDryRunAsksOnAConsole(t *testing.T) {
	root, _ := initWorld(t)
	// base none, then enter for everything else.
	withTerminal(t, tui.Script(100, 40, tui.Keys("tab", "tab", "enter", "enter", "enter", "enter", "enter", "enter")...), nil)
	code, out, errOut := run("init", "--root", root, "--dry-run")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if strings.Contains(out, "binary-install") || strings.Contains(out, "AGENTS.md") {
		t.Errorf("base parts planned after none:\n%s", out)
	}
}

func TestInitYesSetsUpAFreshRepository(t *testing.T) {
	root, s := initWorld(t)
	code, out, errOut := run("init", "--root", root, "--yes")
	if code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	for _, rel := range []string{
		// No .loomux/config.toml: the defaults leave nothing to write there.
		".gitignore", "AGENTS.md", ".mcp.json", ".claude/settings.json",
		".loomux/armed.toml",
		".githooks/pre-commit", ".githooks/pre-push", ".githooks/commit-msg",
		".claude/skills/verify-until-green/SKILL.md", ".claude/skills/brain-ingest/SKILL.md",
		".loomux/state/answers.toml", ".loomux/state/installed.toml",
	} {
		if !there(root, rel) {
			t.Errorf("no %s", rel)
		}
	}
	if got := mustRunGit(t, root, "config", "core.hooksPath"); got != ".githooks" {
		t.Errorf("core.hooksPath = %q", got)
	}
	if s.installed != 1 || s.built != 0 {
		t.Errorf("seams: %+v", s)
	}
	wantActions := []string{"area add --path " + root + " --scope project/demo --yes", "graph build --root " + root}
	for _, a := range s.actions {
		if !slices.Contains(wantActions, a) {
			t.Errorf("unexpected action %q", a)
		}
	}
	if !strings.Contains(out, "written: .claude/settings.json") || !strings.Contains(out, "written: binary-install") {
		t.Errorf("report:\n%s", out)
	}
	installed := readAt(t, root, ".loomux/state/installed.toml")
	if !strings.Contains(installed, `version = "2.13.0"`) {
		t.Errorf("installed.toml:\n%s", installed)
	}
	if !strings.Contains(readAt(t, root, ".githooks/post-merge"), maintenance.HookMarker) {
		t.Error("no post-merge hook")
	}
}

func TestInitWritesNoHookEntryWhenTheBinaryIsMissing(t *testing.T) {
	root, s := initWorld(t)
	replace(t, &installBinary, func(context.Context) selfupdate.Result {
		return selfupdate.Result{Outcome: selfupdate.Failed, Err: errors.New("gh: not logged in")}
	})
	code, out, errOut := run("init", "--root", root, "--yes")
	if code != 1 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if there(root, ".claude/settings.json") || there(root, ".githooks/pre-commit") {
		t.Error("entries written without a binary")
	}
	if !there(root, "AGENTS.md") {
		t.Error("no AGENTS.md")
	}
	if !strings.Contains(errOut, "gh auth login") || !strings.Contains(errOut, "loomux upgrade") {
		t.Errorf("stderr: %s", errOut)
	}
	if !strings.Contains(out, "failed: binary-install") {
		t.Errorf("report:\n%s", out)
	}
	if there(root, ".githooks/post-merge") || there(root, ".git/hooks/post-merge") || len(s.actions) != 1 {
		t.Errorf("merge-hook ran without a binary, or actions %v", s.actions)
	}
	if got, err := gitFacts(root, "git", "config", "--get", "core.hooksPath"); got != "" || err != nil {
		t.Errorf("core.hooksPath = %q, err %v", got, err)
	}
}

func TestInitPresetsTheAnswersFromTheProject(t *testing.T) {
	for name, world := range map[string]func(*testing.T) (string, *initSeams){"fresh": initWorld, "checkout": checkoutWorld} {
		root, _ := world(t)
		_, want, _ := run("init", "--root", root, "--dry-run", "--yes")
		// Taking every offered answer is taking the defaults; enter in a
		// confirmation is ignored.
		term := tui.Script(100, 40, tui.Keys("enter", "enter", "enter", "enter", "enter",
			"enter", "enter", "enter", "enter", "enter", "enter")...)
		withTerminal(t, term, nil)
		code, got, errOut := run("init", "--root", root, "--dry-run")
		if code != 0 || got != want {
			t.Errorf("%s: code %d: %s\ninteractive:\n%s\nwant:\n%s", name, code, errOut, got, want)
		}
	}
	root, _ := initWorld(t)
	// base, hooks, brain and its part list, graph.
	term := tui.Script(100, 40, tui.Keys("enter", "enter", "enter", "enter", "enter")...)
	withTerminal(t, term, nil)
	run("init", "--root", root, "--dry-run", "--graph=all")
	if !strings.Contains(term.Output(), "module graph (graph-build): all") {
		t.Errorf("the flag does not win:\n%s", term.Output())
	}
}

func TestInitChecksThePathTheEntriesCall(t *testing.T) {
	root, s := initWorld(t)
	moved := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", moved)
	replace(t, &installBinary, func(context.Context) selfupdate.Result {
		s.installed++
		writeAt(t, selfupdate.Canonical(moved), "binary")
		return selfupdate.Result{Outcome: selfupdate.Updated}
	})
	code, _, errOut := run("init", "--root", root, "--yes")
	if code != 1 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if there(root, ".claude/settings.json") || there(root, ".githooks/pre-commit") {
		t.Error("entries written")
	}
	if !strings.Contains(errOut, "LOOMUX_STATE_DIR") || !strings.Contains(errOut, `%LOCALAPPDATA%\loomux\bin\loomux.exe`) {
		t.Errorf("stderr: %s", errOut)
	}
}

func TestInitNeedsLocalAppData(t *testing.T) {
	root, _ := initWorld(t)
	t.Setenv("LOCALAPPDATA", "")
	code, _, errOut := run("init", "--root", root, "--yes")
	if code != 1 || !strings.Contains(errOut, "LOOMUX_STATE_DIR") || there(root, ".claude/settings.json") {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestInitReportsAnInstallThatDidNotRun(t *testing.T) {
	root, _ := initWorld(t)
	replace(t, &installBinary, func(context.Context) selfupdate.Result {
		return selfupdate.Result{Outcome: selfupdate.Busy, Err: errors.New("update in progress")}
	})
	code, _, errOut := run("init", "--root", root, "--yes")
	if code != 1 || !strings.Contains(errOut, "busy: update in progress") {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestInitKeepsACurrentBinary(t *testing.T) {
	root, _ := initWorld(t)
	replace(t, &installBinary, func(context.Context) selfupdate.Result {
		writeAt(t, filepath.Join(os.Getenv("LOCALAPPDATA"), "loomux", "bin", "loomux.exe"), "binary")
		return selfupdate.Result{Outcome: selfupdate.Current}
	})
	if code, _, errOut := run("init", "--root", root, "--yes"); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestInitAsksPerModule(t *testing.T) {
	root, _ := initWorld(t)
	keys := tui.Keys(
		"enter",                              // base: all
		"tab", "enter", "down", " ", "enter", // hooks: each, git-hooks off
		"tab", "enter", // brain: none, one tab from the offered each
		"tab", "enter", // graph: none as offered, tab to all
		"tab", "enter", // language: de
	)
	for range 40 {
		keys = append(keys, tui.Keys("y")...)
	}
	term := tui.Script(100, 40, keys...)
	withTerminal(t, term, nil)
	code, out, errOut := run("init", "--root", root)
	if code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if there(root, ".githooks/pre-commit") || there(root, ".claude/skills/brain-ingest/SKILL.md") {
		t.Error("parts written that were switched off")
	}
	if !there(root, ".claude/settings.json") || !there(root, ".claude/skills/verify-until-green/SKILL.md") {
		t.Error("parts missing that were chosen")
	}
	cfg := readAt(t, root, ".loomux/config.toml")
	if !strings.Contains(cfg, "[commit]\nlanguage = \"de\"") || !strings.Contains(cfg, "[modules]\nbrain = false") {
		t.Errorf("config:\n%s", cfg)
	}
	if !strings.Contains(term.Output(), "module brain (wiki)") {
		t.Errorf("screen:\n%s", term.Output())
	}
}

func TestInitAsksForTheScope(t *testing.T) {
	root, s := initWorld(t)
	keys := tui.Keys("enter", "enter", "enter", "enter", "enter", "enter",
		"backspace", "backspace", "backspace", "backspace", " ", "enter", // project/ + space: refused
		"backspace", "x", "enter")
	for range 40 {
		keys = append(keys, tui.Keys("y")...)
	}
	term := tui.Script(100, 40, keys...)
	withTerminal(t, term, nil)
	code, _, errOut := run("init", "--root", root, "--graph=none")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if !slices.Contains(s.actions, "area add --path "+root+" --scope project/x --yes") {
		t.Errorf("actions: %v", s.actions)
	}
	if !strings.Contains(term.Output(), "has no spaces") {
		t.Errorf("screen:\n%s", term.Output())
	}
	if !strings.Contains(readAt(t, root, ".loomux/config.toml"), "graph = false") {
		t.Error("graph not switched off")
	}
}

func TestInitPicksBaseParts(t *testing.T) {
	root, s := initWorld(t)
	// base each: binary off; everything else as offered.
	keys := tui.Keys("tab", "enter", " ", "enter", "enter", "enter", "enter", "enter", "enter", "enter", "enter")
	for range 40 {
		keys = append(keys, tui.Keys("y")...)
	}
	withTerminal(t, tui.Script(100, 40, keys...), nil)
	code, out, errOut := run("init", "--root", root)
	if code != 0 || !strings.Contains(errOut, "loomux init: no loomux binary where the entries call it; "+
		// With the binary switched off the merge hook is not even planned.
		"host entries and git hooks are left out\n") || there(root, ".claude/settings.json") {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if s.installed != 0 {
		t.Error("binary installed though switched off")
	}
}

func TestInitWritesOnlyWhatIsApproved(t *testing.T) {
	root, s := initWorld(t)
	// A binary from an earlier install: nothing is left out for want of one.
	writeAt(t, filepath.Join(os.Getenv("LOCALAPPDATA"), "loomux", "bin", "loomux.exe"), "binary")
	keys := tui.Keys("enter", "enter", "enter", "enter", "enter", "enter", "enter", "enter")
	for range 40 {
		keys = append(keys, tui.Keys("n")...)
	}
	withTerminal(t, tui.Script(100, 40, keys...), nil)
	code, out, errOut := run("init", "--root", root)
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if there(root, "AGENTS.md") || s.installed != 0 || len(s.actions) != 0 {
		t.Errorf("wrote or ran what was refused: %+v", s)
	}
	if !strings.Contains(out, "refused: AGENTS.md") || !strings.Contains(out, "refused: area-add") {
		t.Errorf("report:\n%s", out)
	}
}

func TestInitEndsWhenTheHumanCancels(t *testing.T) {
	for name, keys := range map[string][]tui.Key{
		"module":   tui.Keys("esc"),
		"pick":     tui.Keys("tab", "enter", "esc"),
		"language": tui.Keys("enter", "enter", "enter", "enter", "enter", "enter", "esc"),
		"scope":    tui.Keys("enter", "enter", "enter", "enter", "enter", "enter", "enter", "esc"),
		"approval": tui.Keys("enter", "enter", "enter", "enter", "enter", "enter", "enter", "enter"),
	} {
		root, s := initWorld(t)
		withTerminal(t, tui.Script(100, 40, keys...), nil)
		before := treeHash(t, root)
		code, _, errOut := run("init", "--root", root)
		if code != 0 || !strings.Contains(errOut, "cancelled; nothing written") {
			t.Errorf("%s: code %d: %s", name, code, errOut)
		}
		if treeHash(t, root) != before || s.installed != 0 {
			t.Errorf("%s: wrote", name)
		}
	}
}

// brokenTerminal fails every read.
type brokenTerminal struct{ *tui.Scripted }

func (brokenTerminal) ReadKey() (tui.Key, error) { return tui.Key{}, errors.New("console gone") }

func TestInitReportsABrokenConsole(t *testing.T) {
	root, _ := initWorld(t)
	withTerminal(t, brokenTerminal{tui.Script(100, 40)}, nil)
	code, _, errOut := run("init", "--root", root)
	if code != 1 || !strings.Contains(errOut, "console gone") {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestInitReportsATerminalItCannotRestore(t *testing.T) {
	root, _ := initWorld(t)
	withTerminal(t, tui.Script(100, 40), errors.New("mode lost"))
	code, _, errOut := run("init", "--root", root)
	if code != 1 || !strings.Contains(errOut, "restoring the terminal: mode lost") {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestInitNeedsTheConsoleForTheApprovalToo(t *testing.T) {
	root, _ := initWorld(t)
	opened := 0
	term := tui.Script(100, 40, tui.Keys("enter", "enter", "enter", "enter", "enter", "enter", "enter", "enter")...)
	replace(t, &openTerminal, func() (tui.Terminal, func() error, error) {
		opened++
		if opened > 1 {
			return nil, nil, errors.New("gone")
		}
		return term, func() error { return nil }, nil
	})
	code, _, errOut := run("init", "--root", root)
	if code != 2 || !strings.Contains(errOut, noTerminal) || there(root, "AGENTS.md") {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestInitRefusesWithoutATerminal(t *testing.T) {
	root, _ := initWorld(t)
	before := treeHash(t, root)
	code, _, errOut := run("init", "--root", root)
	if code != 2 || !strings.Contains(errOut, noTerminal) {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if treeHash(t, root) != before {
		t.Error("wrote")
	}
}

func TestInitRefusesABrokenSettingsFile(t *testing.T) {
	root, _ := initWorld(t)
	writeAt(t, filepath.Join(root, ".claude", "settings.json"), "{not json")
	code, _, errOut := run("init", "--root", root, "--yes")
	if code != 2 || !strings.Contains(errOut, ".claude/settings.json") {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if readAt(t, root, ".claude/settings.json") != "{not json" || there(root, "AGENTS.md") {
		t.Error("changed the project")
	}
}

func TestInitRefusesBrokenAnswers(t *testing.T) {
	root, _ := initWorld(t)
	writeAt(t, filepath.Join(root, ".loomux", "state", "answers.toml"), "hosts = [")
	code, _, errOut := run("init", "--root", root, "--yes")
	if code != 2 || !strings.Contains(errOut, "answers.toml") {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestInitReportsFactsItCannotRead(t *testing.T) {
	root, _ := initWorld(t)
	if err := os.MkdirAll(filepath.Join(root, "go.mod"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run("init", "--root", root, "--yes")
	if code != 1 || !strings.Contains(errOut, "go.mod") {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestInitStopsAtAFailingAction(t *testing.T) {
	root, _ := initWorld(t)
	replace(t, &runAction, func(name string, _ []string, _, _ io.Writer) int { return 1 })
	code, out, errOut := run("init", "--root", root, "--yes")
	if code != 1 || !strings.Contains(errOut, "area-add: loomux area add --path") || !strings.Contains(out, "failed: area-add") {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if there(root, ".loomux/state/installed.toml") {
		t.Error("installed.toml after a failed run")
	}
}

func TestInitSecondRunChangesNothing(t *testing.T) {
	root, s := initWorld(t)
	writeAt(t, filepath.Join(root, "go.mod"), "module example.com/demo\n\ngo 1.26\n")
	// area add is the real one; the graph build leaves what the real one
	// leaves.
	replace(t, &runAction, func(name string, args []string, stdout, stderr io.Writer) int {
		s.actions = append(s.actions, name+" "+strings.Join(args, " "))
		if name == "area" {
			return runSubcommand(name, append(args, "--no-reindex"), stdout, stderr)
		}
		writeAt(t, filepath.Join(root, ".loomux", "state", "graph", "wiring.json"), "{}")
		return 0
	})
	if code, out, errOut := run("init", "--root", root, "--yes"); code != 0 {
		t.Fatalf("first run: %d: %s\n%s", code, errOut, out)
	}
	if len(s.actions) != 2 || s.installed != 1 {
		t.Fatalf("first run: actions %v, installs %d", s.actions, s.installed)
	}
	installed := readAt(t, root, ".loomux/state/installed.toml")
	code, out, errOut := run("init", "--root", root, "--yes")
	if code != 0 {
		t.Fatalf("second run: %d: %s", code, errOut)
	}
	if !strings.HasPrefix(out, "nothing to change\n") || strings.Contains(out, "written: ") {
		t.Errorf("second run:\n%s", out)
	}
	if len(s.actions) != 2 || s.installed != 1 || readAt(t, root, ".loomux/state/installed.toml") != installed {
		t.Errorf("second run acted: actions %v, installs %d", s.actions, s.installed)
	}
}

func TestApprovePlanStopsAtAClosedConsole(t *testing.T) {
	p := setup.Plan{Changes: []setup.Change{{Path: "a", After: "x\n"}}, Actions: []setup.Action{{ID: "hooks-path"}}}
	approved := map[string]bool{}
	ok, err := approvePlan(tui.Script(100, 40, tui.Keys("y")...), p, approved)
	if ok || !errors.Is(err, io.EOF) || !approved["a"] {
		t.Fatalf("ok %v, err %v, approved %v", ok, err, approved)
	}
}

func TestInitRefusesAWrongCall(t *testing.T) {
	root, _ := initWorld(t)
	file := filepath.Join(root, "file")
	writeAt(t, file, "")
	for _, args := range [][]string{
		{"--detect-only", "--yes"},
		{"--hooks=some"},
		{"extra"},
		{"--dry-run", "--dry-run=false"},
		{"--detect-only=false"},
		{"--", "--dry-run"},
		{"--nope"},
		{"--hosts=claude,nothing"},
		{"--root", file},
	} {
		before := treeHash(t, root)
		code, _, errOut := run(append([]string{"init", "--root", root}, args...)...)
		if code != 2 || errOut == "" {
			t.Errorf("%v: code %d, stderr %q", args, code, errOut)
		}
		if treeHash(t, root) != before {
			t.Errorf("%v: wrote", args)
		}
	}
	// The guard lets these through for their --dry-run or --detect-only; as
	// the value of --root they would name a directory, and a run that found
	// one would write.
	for _, args := range [][]string{{"--yes", "--root", "--dry-run"}, {"--root", "--detect-only"}} {
		code, _, errOut := run(append([]string{"init"}, args...)...)
		if code != 2 || !strings.Contains(errOut, "expected one argument") {
			t.Errorf("%v: code %d, stderr %q", args, code, errOut)
		}
	}
}

func TestInitWritesForTheHostsGiven(t *testing.T) {
	root, _ := initWorld(t)
	code, out, errOut := run("init", "--root", root, "--yes", "--dry-run", "--hosts=antigravity", "--hooks=none", "--brain=each")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if strings.Contains(out, ".claude/") || strings.Contains(out, ".githooks") {
		t.Errorf("plan:\n%s", out)
	}
}

// Both hosts' hook files are planned; Antigravity's entries call the
// installed binary through cmd.exe from .agents/, and its skills go under
// .agents/skills.
func TestInitPlansAntigravityBesideClaude(t *testing.T) {
	root, _ := initWorld(t)
	code, out, errOut := run("init", "--root", root, "--dry-run", "--yes", "--hosts=claude,antigravity")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	for _, want := range []string{
		"--- .claude/settings.json",
		"--- .agents/hooks.json",
		"%LOCALAPPDATA%/loomux/bin/loomux.exe hook session-start --host antigravity --root ..",
		"%LOCALAPPDATA%/loomux/bin/loomux.exe hook pre-tool-use --host antigravity --root ..",
		"%LOCALAPPDATA%/loomux/bin/loomux.exe hook post-tool-use --host antigravity --root ..",
		"%LOCALAPPDATA%/loomux/bin/loomux.exe hook stop --host antigravity --root .. --budget 270s",
		"--- .agents/skills/verify-until-green/SKILL.md",
		"--- .agents/skills/brain-land/SKILL.md",
		"antigravity: .agents/hooks.json loads only in a folder agy trusts (trustedWorkspaces)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("plan lacks %q:\n%s", want, out)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".agents")); err == nil {
		t.Error("a dry run wrote .agents/")
	}
}

func TestInitNamesAModuleSwitchedOffWithoutAConfiguration(t *testing.T) {
	root, _ := initWorld(t)
	keys := tui.Keys("tab", "enter", "down", " ", "enter", // base each: config off
		"enter", "tab", "enter", "enter", "enter", "enter") // brain: none, one tab from the offered each
	withTerminal(t, tui.Script(100, 40, keys...), nil)
	code, out, errOut := run("init", "--root", root, "--dry-run")
	if code != 0 || !strings.Contains(out, "modules.brain = false is not written; the part config is off") {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
}

// checkoutWorld is the fresh clone of loomux: its go.mod and configuration.
func checkoutWorld(t *testing.T) (string, *initSeams) {
	t.Helper()
	root, s := initWorld(t)
	writeAt(t, filepath.Join(root, "go.mod"), "module github.com/xidus90/loomux\n\ngo 1.26.0\n")
	cfg, err := os.ReadFile(filepath.Join("..", "setup", "testdata", "loomux", "loomux-config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	writeAt(t, filepath.Join(root, ".loomux", "config.toml"), string(cfg))
	return root, s
}

func TestInitOnTheCheckoutBuildsInsteadOfInstalling(t *testing.T) {
	root, s := checkoutWorld(t)
	code, out, errOut := run("init", "--root", root, "--yes")
	if code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if s.built != 1 || s.installed != 0 {
		t.Errorf("seams: %+v", s)
	}
	if !strings.Contains(readAt(t, root, ".claude/settings.json"), "${CLAUDE_PROJECT_DIR}/bin/loomux.exe") {
		t.Error("entries do not call the checkout's binary")
	}
}

func TestInitRefusesToBuildOutsideACheckout(t *testing.T) {
	root, s := initWorld(t)
	entries, err := json.Marshal(map[string]any{"hooks": map[string]any{"PreToolUse": []any{map[string]any{
		"matcher": "*", "hooks": []any{map[string]any{"type": "command", "command": `"${CLAUDE_PROJECT_DIR}/bin/loomux.exe" hook pre-tool-use --host claude`}},
	}}}})
	if err != nil {
		t.Fatal(err)
	}
	writeAt(t, filepath.Join(root, ".claude", "settings.json"), string(entries))
	code, out, errOut := run("init", "--root", root, "--yes")
	if code != 1 || s.built != 0 {
		t.Fatalf("code %d, seams %+v: %s\n%s", code, s, errOut, out)
	}
	if !strings.Contains(out, "binary-build: "+notCheckout) || !strings.Contains(errOut, notCheckout) {
		t.Errorf("stdout:\n%s\nstderr:\n%s", out, errOut)
	}
}

func TestInitRunsNoUnknownAction(t *testing.T) {
	r := &initRun{}
	if err := r.action(setup.Action{ID: "dance"}); err == nil || !strings.Contains(err.Error(), "no action dance") {
		t.Fatalf("err = %v", err)
	}
}

func TestInitFindsNoBinaryWithoutLocalAppData(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")
	if binaryThere(t.TempDir(), setup.Facts{Binary: hostfile.Canonical})() {
		t.Fatal("found a binary")
	}
}

func TestRunSubcommandReachesEachCommand(t *testing.T) {
	for _, name := range []string{"area", "graph", "merge-hook", "dance"} {
		var out, errOut strings.Builder
		if code := runSubcommand(name, []string{"--no-such-subcommand"}, &out, &errOut); code != 2 {
			t.Errorf("%s: code %d: %s", name, code, errOut.String())
		}
	}
}

func TestGitFactsReadsAnUnsetSettingAsEmpty(t *testing.T) {
	root, _ := initWorld(t)
	out, err := gitFacts(root, "git", "config", "--get", "loomux.nothing")
	if out != "" || err != nil {
		t.Fatalf("out %q, err %v", out, err)
	}
	if _, err := gitFacts(filepath.Join(root, "missing"), "git", "status"); err == nil {
		t.Fatal("no error for a missing directory")
	}
}

func TestCheckScope(t *testing.T) {
	for scope, ok := range map[string]bool{"project/a": true, "": false, "project/a b": false, "a\tb": false} {
		if got := checkScope(scope) == nil; got != ok {
			t.Errorf("%q: ok = %v", scope, got)
		}
	}
}

func TestModuleTitle(t *testing.T) {
	if moduleTitle(schema.Brain) != "brain (wiki)" || moduleTitle(schema.Graph) != "graph" {
		t.Fatal("titles")
	}
}

func TestPrintPlanSaysWhenNothingChanges(t *testing.T) {
	var out strings.Builder
	printPlan(&out, setup.Plan{})
	if out.String() != "nothing to change\n" {
		t.Fatalf("%q", out.String())
	}
}

func TestBuildPilotBuildsAndSwaps(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the Go toolchain")
	}
	cache, err := exec.Command("go", "env", "GOCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOCACHE", strings.TrimSpace(string(cache)))
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOFLAGS", "")
	root := t.TempDir()
	writeAt(t, filepath.Join(root, "go.mod"), "module example.com/pilot\n\ngo 1.21\n")
	writeAt(t, filepath.Join(root, "cmd", "loomux", "main.go"), "package main\n\nfunc main() {}\n")
	if err := buildPilot(root); err != nil {
		t.Fatal(err)
	}
	if !there(root, "bin/loomux.exe") || there(root, "bin/loomux.new.exe") {
		t.Fatal("no swapped binary")
	}
	if err := os.Remove(filepath.Join(root, "cmd", "loomux", "main.go")); err != nil {
		t.Fatal(err)
	}
	if err := buildPilot(root); err == nil || !strings.Contains(err.Error(), "go build") {
		t.Fatalf("err = %v", err)
	}
	blocked := t.TempDir()
	writeAt(t, filepath.Join(blocked, "bin"), "a file where the directory goes")
	if err := buildPilot(blocked); err == nil {
		t.Fatal("no error for a blocked bin directory")
	}
}

func TestInitLetsAreaAddDeclareTheAreaFirst(t *testing.T) {
	root, _ := initWorld(t)
	code, out, errOut := run("init", "--root", root, "--yes", "--graph=none")
	if code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	cfg := readAt(t, root, ".loomux/config.toml")
	for _, want := range []string{"[area]\nscope = \"project/demo\"", "[maintenance]\non_merge = true", "[modules]\ngraph = false"} {
		if !strings.Contains(cfg, want) {
			t.Errorf("no %q in\n%s", want, cfg)
		}
	}
	agents := readAt(t, root, "AGENTS.md")
	if !strings.HasPrefix(agents, "# demo") || !strings.Contains(agents, routingHeading) {
		t.Errorf("AGENTS.md:\n%s", agents)
	}
	code, out, errOut = run("init", "--root", root, "--yes")
	if code != 0 || !strings.HasPrefix(out, "nothing to change\n") {
		t.Errorf("second run: code %d: %s\n%s", code, errOut, out)
	}
}

// registerArea writes a registry holding one area per scope and path.
func registerArea(t *testing.T, areas ...[2]string) {
	t.Helper()
	var text strings.Builder
	for _, a := range areas {
		text.WriteString("[[area]]\nscope = " + strconv.Quote(a[0]) + "\npath = " + strconv.Quote(filepath.ToSlash(a[1])) + "\n\n")
	}
	writeAt(t, filepath.Join(os.Getenv("LOOMUX_STATE_DIR"), "registry.toml"), text.String())
}

func TestInitInstallsTheMergeHookForItsOwnRootOnly(t *testing.T) {
	root, _ := initWorld(t)
	// A consenting area whose directory is no repository: merge-hook install
	// over the whole registry answers "no repository" and fails.
	stale := t.TempDir()
	writeAt(t, filepath.Join(stale, ".loomux", "config.toml"),
		"[area]\nscope = \"project/stale\"\n\n[maintenance]\non_merge = true\n")
	registerArea(t, [2]string{"project/stale", stale})
	code, out, errOut := run("init", "--root", root, "--yes", "--graph=none")
	if code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if !strings.Contains(readAt(t, root, ".githooks/post-merge"), maintenance.HookMarker) || strings.Contains(out, "project/stale") {
		t.Errorf("stdout:\n%s", out)
	}
}

func TestInitPlansNoMergeHookWithoutConsent(t *testing.T) {
	root, _ := initWorld(t)
	writeAt(t, filepath.Join(root, ".loomux", "config.toml"), "[area]\nscope = \"project/demo\"\n")
	registerArea(t, [2]string{"project/demo", root})
	code, out, errOut := run("init", "--root", root, "--yes", "--graph=none")
	if code != 0 || strings.Contains(out, "merge-hook:  ") || strings.Contains(out, "  merge-hook: install") {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if there(root, ".githooks/post-merge") || !strings.Contains(out, "merge-hook: skipped") {
		t.Errorf("stdout:\n%s", out)
	}
	if code, out, _ := run("init", "--root", root, "--yes"); code != 0 || !strings.HasPrefix(out, "nothing to change\n") {
		t.Errorf("second run: code %d:\n%s", code, out)
	}
}

func TestInitJudgesTheMergeHookByItsOwnLines(t *testing.T) {
	root, _ := initWorld(t)
	var out strings.Builder
	r := &initRun{root: root, stdout: &out}
	if err := r.mergeHook(); err == nil || !strings.Contains(err.Error(), "consents") {
		t.Errorf("no registry: err = %v", err)
	}
	writeAt(t, filepath.Join(root, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[maintenance]\non_merge = true\n")
	registerArea(t, [2]string{"project/demo", root})
	writeAt(t, filepath.Join(root, ".git", "hooks", "post-merge"), "#!/bin/sh\necho mine\n")
	if err := r.mergeHook(); err == nil || !strings.Contains(err.Error(), "post-merge hook of project/demo is refused") || !strings.Contains(out.String(), "refused: project/demo") {
		t.Errorf("foreign hook: err = %v, out %s", err, out.String())
	}
	writeAt(t, filepath.Join(os.Getenv("LOOMUX_STATE_DIR"), "registry.toml"), "not toml [")
	if err := r.mergeHook(); err == nil {
		t.Error("no error for a broken registry")
	}
	writeAt(t, filepath.Join(root, ".loomux", "config.toml"), "[area]\nscope = 1\n")
	registerArea(t, [2]string{"project/demo", root})
	if err := r.mergeHook(); err == nil {
		t.Error("no error for a declaration that does not read")
	}
}

func TestInitDecliningTheBinaryIsNoFailure(t *testing.T) {
	root, s := initWorld(t)
	_, plan, _ := run("init", "--root", root, "--dry-run", "--yes")
	keys := tui.Keys("enter", "enter", "enter", "enter", "enter", "enter", "enter", "enter")
	// Every change, then no to binary-install, the first action.
	for range strings.Count(plan, "\n--- ") + 1 {
		keys = append(keys, tui.Keys("y")...)
	}
	keys = append(keys, tui.Keys("n")...)
	for range 10 {
		keys = append(keys, tui.Keys("y")...)
	}
	withTerminal(t, tui.Script(100, 40, keys...), nil)
	code, out, errOut := run("init", "--root", root)
	if code != 0 || s.installed != 0 || !strings.Contains(out, "refused: binary-install") || !strings.Contains(errOut,
		"loomux init: no loomux binary where the entries call it; host entries, git hooks and the merge hook are left out\n") {
		t.Fatalf("code %d, installs %d: %s\n%s", code, s.installed, errOut, out)
	}
}

func TestInitStopsAtAConfigurationItCannotRead(t *testing.T) {
	root, _ := initWorld(t)
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := treeHash(t, root)
	code, _, errOut := run("init", "--root", root, "--yes")
	if code != 2 || !strings.Contains(errOut, "config.toml") || treeHash(t, root) != before {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestInitPlansNoMergeHookForAClonedDeclaration(t *testing.T) {
	root, _ := initWorld(t)
	// The declaration came with the clone; the registry of this machine has
	// no area here.
	writeAt(t, filepath.Join(root, ".loomux", "config.toml"),
		"[area]\nscope = \"project/demo\"\n\n[maintenance]\non_merge = true\n")
	for i := range 2 {
		code, out, errOut := run("init", "--root", root, "--yes", "--graph=none")
		if code != 0 || there(root, ".githooks/post-merge") || !strings.Contains(out, "registry of this machine has no area here") {
			t.Fatalf("run %d: code %d: %s\n%s", i+1, code, errOut, out)
		}
		if i == 1 && !strings.HasPrefix(out, "nothing to change\n") {
			t.Errorf("second run:\n%s", out)
		}
	}
}

func TestInitOverAConfigurationWithoutAnAreaInstallsNoMergeHook(t *testing.T) {
	root, s := initWorld(t)
	writeAt(t, filepath.Join(root, ".loomux", "config.toml"), "[commit]\nlanguage = \"de\"\n")
	code, out, errOut := run("init", "--root", root, "--yes", "--graph=none")
	if code != 0 || there(root, ".githooks/post-merge") || strings.Contains(out, "merge-hook: install") ||
		strings.Contains(out, "merge-hook") && !strings.Contains(out, "merge-hook: skipped") {
		t.Fatalf("code %d, actions %v: %s\n%s", code, s.actions, errOut, out)
	}
	if !slices.Contains(s.actions, "area add --path "+root+" --scope project/demo --yes") {
		t.Errorf("actions: %v", s.actions)
	}
}

func TestInitSetsCoreHooksPathWithTheHooks(t *testing.T) {
	for name, keep := range map[string]string{"none": "", "one": ".githooks/pre-push"} {
		root, _ := initWorld(t)
		_, plan, _ := run("init", "--root", root, "--dry-run", "--yes")
		keys := tui.Keys("enter", "enter", "enter", "enter", "enter", "enter", "enter", "enter")
		for _, line := range strings.Split(plan, "\n") {
			path, ok := strings.CutPrefix(line, "--- ")
			if !ok {
				continue
			}
			answer := "y"
			if strings.HasPrefix(path, ".githooks/") && path != keep {
				answer = "n"
			}
			keys = append(keys, tui.Keys(answer)...)
		}
		for range 10 {
			keys = append(keys, tui.Keys("y")...)
		}
		term := tui.Script(100, 40, keys...)
		withTerminal(t, term, nil)
		code, out, errOut := run("init", "--root", root)
		if code != 0 || strings.Contains(term.Output(), "run hooks-path?") {
			t.Fatalf("%s: code %d: %s\n%s", name, code, errOut, out)
		}
		got, _ := gitFacts(root, "git", "config", "--get", "core.hooksPath")
		if want := map[string]string{"none": "", "one": ".githooks\n"}[name]; got != want {
			t.Errorf("%s: core.hooksPath = %q, want %q\n%s", name, got, want, out)
		}
	}
}

func TestInitOnTheCheckoutSkipsAMergeHookWithoutTheInstalledBinary(t *testing.T) {
	root, _ := checkoutWorld(t)
	code, out, errOut := run("init", "--root", root, "--yes", "--brain=all")
	if code != 0 || strings.Contains(out, "merge-hook: install") || there(root, ".githooks/post-merge") ||
		!strings.Contains(out, "merge-hook: skipped; the hook calls ${LOCALAPPDATA}/loomux/bin/loomux.exe, which is not installed") {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
}

// A project loomux is set up in for the first time starts in probation, and
// only then: the second run finds a configuration, the file and its own
// hook, and leaves what a commit or a human armed since.
func TestInitStartsANewProjectInProbationOnce(t *testing.T) {
	root, _ := initWorld(t)
	if code, out, errOut := run("init", "--root", root, "--yes"); code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if got := readAt(t, root, ".loomux/armed.toml"); got != (verify.ArmedSet{Exists: true}).Text() {
		t.Fatalf("armed.toml %q", got)
	}
	if !verify.HookArms(readAt(t, root, ".githooks/pre-commit")) {
		t.Fatalf("the hook init wrote does not arm:\n%s", readAt(t, root, ".githooks/pre-commit"))
	}
	const armed = "armed = [\"lint/go@.\"]\n"
	writeAt(t, filepath.Join(root, ".loomux", "armed.toml"), armed)
	if code, out, errOut := run("init", "--root", root, "--yes"); code != 0 || readAt(t, root, ".loomux/armed.toml") != armed {
		t.Fatalf("second run: code %d, armed.toml %q: %s\n%s", code, readAt(t, root, ".loomux/armed.toml"), errOut, out)
	}
	// Taking the file away does not bring the probation back: the project
	// is set up, and without the file every lane is armed.
	if err := os.Remove(filepath.Join(root, ".loomux", "armed.toml")); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := run("init", "--root", root, "--yes"); code != 0 || there(root, ".loomux/armed.toml") {
		t.Fatalf("third run: code %d, the file is back: %s", code, errOut)
	}
}
