package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/maintenance"
)

// fenceGit keeps every git the command starts away from the machine: no
// system configuration, and a home and XDG directory without a .gitconfig.
// GIT_CONFIG_GLOBAL would not do -- gitenv strips it -- and a global
// core.hooksPath would send an install into the user's own hooks directory.
// The state directories are fenced too, so no run reads or writes the
// machine's registry.
func fenceGit(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	state := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", state)
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", t.TempDir())
	return state
}

func mustRunGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := runGit(dir, args...)
	if err != nil {
		t.Fatalf("git %s in %s: %v", strings.Join(args, " "), dir, err)
	}
	return out
}

// consentingRepo is a fresh repository on main with one commit, registered
// as project/a in state, whose declaration consents to merges on main. It
// answers the repository's top level as git spells it, in native slashes.
func consentingRepo(t *testing.T, state string) string {
	t.Helper()
	repo := t.TempDir()
	mustRunGit(t, repo, "init", "-q", "-b", "main")
	mustRunGit(t, repo, "-c", "user.name=t", "-c", "user.email=t@t.invalid", "commit", "-q", "--allow-empty", "-m", "base")
	writeFile(t, filepath.Join(repo, ".brain.toml"),
		"[area]\nscope = \"project/a\"\n\n[maintenance]\non_merge = true\nbranch = \"main\"\n")
	writeFile(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/a\"\npath = \""+filepath.ToSlash(repo)+"\"\n")
	return filepath.FromSlash(mustRunGit(t, repo, "rev-parse", "--path-format=absolute", "--show-toplevel"))
}

func TestMergeHookInstallPrintsOneLinePerArea(t *testing.T) {
	state := fenceGit(t)
	repo := consentingRepo(t, state)
	code, out, errOut := run("merge-hook", "install")
	hook := filepath.Join(repo, ".git", "hooks", "post-merge")
	if want := "installed: project/a — " + repo + " [" + hook + "]\n"; code != 0 || out != want {
		t.Fatalf("code %d, out %q, want %q; stderr %q", code, out, want, errOut)
	}
	body, err := os.ReadFile(hook)
	if err != nil || string(body) != maintenance.HookText() {
		t.Fatalf("hook %q, %v", body, err)
	}
}

func TestMergeHookInstallFailsOnARefusal(t *testing.T) {
	state := fenceGit(t)
	repo := consentingRepo(t, state)
	writeFile(t, filepath.Join(repo, ".git", "hooks", "post-merge"), "#!/bin/sh\necho mine\n")
	code, out, _ := run("merge-hook", "install")
	if code != 1 || !strings.HasPrefix(out, "refused: project/a — ") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestMergeHookStatusSaysSoWhenThereIsNothing(t *testing.T) {
	state := fenceGit(t)
	writeFile(t, filepath.Join(state, "registry.toml"), "")
	code, out, _ := run("merge-hook", "status")
	if code != 0 || out != nothingConsents+"\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestMergeHookRemoveTakesTheHookBack(t *testing.T) {
	state := fenceGit(t)
	repo := consentingRepo(t, state)
	if code, _, errOut := run("merge-hook", "install"); code != 0 {
		t.Fatalf("install: %d %s", code, errOut)
	}
	code, out, _ := run("merge-hook", "remove")
	if !strings.HasPrefix(out, "removed: project/a — ") || code != 0 {
		t.Fatalf("code %d, out %q", code, out)
	}
	if _, err := os.Stat(filepath.Join(repo, ".git", "hooks", "post-merge")); !os.IsNotExist(err) {
		t.Fatalf("hook still there: %v", err)
	}
}

func TestMergeHookReadsAMissingRegistryAsEmpty(t *testing.T) {
	for _, sub := range []string{"status", "install", "remove"} {
		t.Run(sub, func(t *testing.T) {
			fenceGit(t)
			code, out, errOut := run("merge-hook", sub)
			if code != 0 || out != nothingConsents+"\n" || errOut != "" {
				t.Fatalf("code %d, out %q, stderr %q", code, out, errOut)
			}
		})
	}
}

func TestMergeHookFailsOnARegistryItCannotRead(t *testing.T) {
	state := fenceGit(t)
	writeFile(t, filepath.Join(state, "registry.toml"), "[[area]\nnot toml")
	code, out, errOut := run("merge-hook", "status")
	if code != 1 || out != "" || !strings.Contains(errOut, "loomux merge-hook status: ") {
		t.Fatalf("code %d, out %q, stderr %q", code, out, errOut)
	}
}

func TestMergeHookFailsWhenTheRecordsDoNotRead(t *testing.T) {
	state := fenceGit(t)
	consentingRepo(t, state)
	// A directory where the record file belongs: reading it fails.
	if err := os.MkdirAll(filepath.Join(state, "maintenance", "hooks.tsv"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := run("merge-hook", "install")
	if code != 1 || out != "" || !strings.Contains(errOut, "loomux merge-hook install: ") {
		t.Fatalf("code %d, out %q, stderr %q", code, out, errOut)
	}
}

func TestMergeHookRecordWritesTheMerge(t *testing.T) {
	state := fenceGit(t)
	repo := consentingRepo(t, state)
	// What git merge leaves behind before it runs the hook.
	mustRunGit(t, repo, "update-ref", "ORIG_HEAD", "HEAD")
	t.Chdir(repo)
	code, out, errOut := run("merge-hook", "record")
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("code %d, out %q, stderr %q", code, out, errOut)
	}
	events, err := os.ReadFile(filepath.Join(state, "maintenance", "merge-events.tsv"))
	if err != nil || strings.Count(string(events), "\tmain\t") != 1 {
		t.Fatalf("events %q, %v", events, err)
	}
}

func TestMergeHookRecordAlwaysExitsZero(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T, state string){
		"no registry": func(t *testing.T, state string) {
			t.Chdir(t.TempDir())
		},
		"outside a repository": func(t *testing.T, state string) {
			consentingRepo(t, state)
			t.Chdir(t.TempDir())
		},
		"broken registry": func(t *testing.T, state string) {
			writeFile(t, filepath.Join(state, "registry.toml"), "[[area]\nnot toml")
			t.Chdir(t.TempDir())
		},
		"unwritable state": func(t *testing.T, state string) {
			repo := consentingRepo(t, state)
			mustRunGit(t, repo, "update-ref", "ORIG_HEAD", "HEAD")
			// A file where the maintenance directory belongs.
			writeFile(t, filepath.Join(state, "maintenance"), "")
			t.Chdir(repo)
		},
		"arguments": func(t *testing.T, state string) {
			t.Chdir(t.TempDir())
		},
	} {
		t.Run(name, func(t *testing.T) {
			state := fenceGit(t)
			setup(t, state)
			args := []string{"merge-hook", "record"}
			if name == "arguments" {
				args = append(args, "--nothing", "x")
			}
			code, out, errOut := run(args...)
			if code != 0 || out != "" || errOut != "" {
				t.Fatalf("code %d, out %q, stderr %q", code, out, errOut)
			}
		})
	}
}

func TestMergeHookRefusesAWrongCall(t *testing.T) {
	fenceGit(t)
	for _, args := range [][]string{{"merge-hook"}, {"merge-hook", "x"}, {"merge-hook", "status", "extra"}} {
		code, out, errOut := run(args...)
		if code != 2 || out != "" || errOut != mergeHookUsage+"\n" {
			t.Fatalf("%v: code %d, out %q, stderr %q", args, code, out, errOut)
		}
	}
}

// The guard lets status and record through and refuses install and remove
// by name (wordsWriteConfiguration in internal/hooks/guard.go). A new
// subcommand would pass it unjudged, so this list fails first.
func TestTheMergeHookSubcommandsAreTheOnesTheGuardJudges(t *testing.T) {
	got := []string{"record"}
	for name := range mergeHookCommands {
		got = append(got, name)
	}
	slices.Sort(got)
	if want := []string{"install", "record", "remove", "status"}; !slices.Equal(got, want) {
		t.Fatalf("merge-hook subcommands = %v, want %v; judge the new one in wordsWriteConfiguration "+
			"(internal/hooks/guard.go) before adding it here", got, want)
	}
}
