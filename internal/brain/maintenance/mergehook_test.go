package maintenance_test

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/gitenv"
)

// realGit is the Git the commands get in production, fenced in for a test:
// no global or system configuration can move the hooks directory into the
// user's own, and discovery stops at the temporary directory, so an area that
// is no repository is not taken for part of one that happens to sit above it.
func realGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(gitenv.Clean(os.Environ()),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CEILING_DIRECTORIES="+os.TempDir(),
	)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := realGit(dir, args...)
	if err != nil {
		t.Fatalf("git %s in %s: %v", strings.Join(args, " "), dir, err)
	}
	return out
}

// failingGit is realGit that fails every call in dir naming want, for the
// arms a real repository never reaches.
func failingGit(dir, want string) maintenance.Git {
	return func(at string, args ...string) (string, error) {
		if at == dir {
			for _, arg := range args {
				if arg == want {
					return "", os.ErrInvalid
				}
			}
		}
		return realGit(at, args...)
	}
}

func newRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	repo := t.TempDir()
	mustGit(t, repo, "init", "-q", "-b", "main")
	mustGit(t, repo, "config", "user.name", "Test")
	mustGit(t, repo, "config", "user.email", "test@example.invalid")
	mustGit(t, repo, "config", "commit.gpgsign", "false")
	mustGit(t, repo, "commit", "-q", "--allow-empty", "-m", "base")
	return repo
}

// declare writes an area's declaration into dir.
func declare(t *testing.T, dir, scope, maintenanceTable string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, ".loomux", "config.toml"),
		"[area]\nscope = \""+scope+"\"\n"+maintenanceTable)
}

// consenting is a state directory with one registered area in a fresh
// repository whose declaration consents to merges on branch.
func consenting(t *testing.T, branch string) (config.ArtifactLookup, []config.Area, string) {
	t.Helper()
	state, repo := t.TempDir(), newRepo(t)
	declare(t, repo, "project/a", "[maintenance]\non_merge = true\nbranch = \""+branch+"\"\n")
	areas := []config.Area{{Scope: "project/a", Path: repo, WikiPath: filepath.Join(repo, "docs", "wiki"), Workspace: true}}
	return config.ArtifactLookup{Primary: state}, areas, repo
}

// topLevel is the repository root as the commands store it.
func topLevel(t *testing.T, dir string) string {
	t.Helper()
	return filepath.Clean(filepath.FromSlash(mustGit(t, dir, "rev-parse", "--path-format=absolute", "--show-toplevel")))
}

func hookIn(repo string) string {
	return filepath.Join(repo, ".git", "hooks", "post-merge")
}

func recordsOf(lookup config.ArtifactLookup) string {
	return filepath.Join(lookup.Primary, "maintenance", "hooks.tsv")
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return string(data)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func mustInstall(t *testing.T, areas []config.Area, lookup config.ArtifactLookup) []maintenance.HookState {
	t.Helper()
	states, err := maintenance.InstallHooks(areas, lookup, realGit)
	if err != nil {
		t.Fatalf("InstallHooks: %v", err)
	}
	return states
}

func statesOf(states []maintenance.HookState) string {
	var parts []string
	for _, s := range states {
		parts = append(parts, s.Scope+":"+s.State)
	}
	return strings.Join(parts, ",")
}

func TestTheHookTextBakesInNoPathOfThisMachine(t *testing.T) {
	text := maintenance.HookText()
	if !strings.HasPrefix(text, "#!/bin/sh\n"+maintenance.HookMarker+" ") {
		t.Errorf("HookText does not open with the shebang and the marker:\n%s", text)
	}
	if !strings.Contains(text, "\"${LOCALAPPDATA}/loomux/bin/loomux.exe\" merge-hook record >/dev/null 2>&1\nexit 0\n") {
		t.Errorf("HookText does not end in the silent record call:\n%s", text)
	}
	if strings.Contains(text, "\r") {
		t.Error("HookText carries a carriage return; git's sh chokes on it")
	}
}

func TestInstallWritesTheHookWhereGitLooksForIt(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	states := mustInstall(t, areas, lookup)
	top := topLevel(t, repo)
	hook := filepath.Join(top, ".git", "hooks", "post-merge")
	want := []maintenance.HookState{{State: "installed", Scope: "project/a", Repo: top, Detail: hook}}
	if len(states) != 1 || states[0] != want[0] {
		t.Fatalf("states = %+v, want %+v", states, want)
	}
	if got := readText(t, hookIn(repo)); got != maintenance.HookText() {
		t.Errorf("hook = %q, want HookText()", got)
	}
	if got := readText(t, recordsOf(lookup)); got != "project/a\t"+top+"\t"+hook+"\n" {
		t.Errorf("hooks.tsv = %q", got)
	}
	if maintenance.HookFailed(states) {
		t.Error("HookFailed on a clean installation")
	}
}

func TestInstallHonoursACoreHooksPath(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	mustGit(t, repo, "config", "core.hooksPath", ".githooks")
	states := mustInstall(t, areas, lookup)
	hook := filepath.Join(repo, ".githooks", "post-merge")
	if got := readText(t, hook); got != maintenance.HookText() {
		t.Errorf("hook = %q, want HookText()", got)
	}
	if exists(hookIn(repo)) {
		t.Error("the hook was also written to .git/hooks")
	}
	if states[0].Detail != filepath.Join(topLevel(t, repo), ".githooks", "post-merge") {
		t.Errorf("Detail = %q", states[0].Detail)
	}
}

func TestInstallSkipsAnAreaThatDoesNotConsent(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	declare(t, repo, "project/a", "")
	// A second area without any declaration is a silent no as well.
	areas = append(areas, config.Area{Scope: "project/b", Path: t.TempDir()})
	states := mustInstall(t, areas, lookup)
	if len(states) != 0 {
		t.Errorf("states = %+v, want none", states)
	}
	if exists(hookIn(repo)) || exists(recordsOf(lookup)) {
		t.Error("a hook or a record was written without consent")
	}
}

func TestInstallRefusesAForeignHook(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	foreign := "#!/bin/sh\necho mine\n"
	writeFile(t, hookIn(repo), foreign)
	states := mustInstall(t, areas, lookup)
	if statesOf(states) != "project/a:refused" || states[0].Detail != filepath.Join(topLevel(t, repo), ".git", "hooks", "post-merge") {
		t.Errorf("states = %+v", states)
	}
	if readText(t, hookIn(repo)) != foreign {
		t.Error("the foreign hook was changed")
	}
	if !maintenance.HookFailed(states) {
		t.Error("HookFailed is false for a refusal")
	}
	if exists(recordsOf(lookup)) {
		t.Error("a refused installation was recorded")
	}
}

func TestAForeignHookThatIsNotUTF8IsStillRefused(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	foreign := "\xff\xfe\x00compiled\x80"
	writeFile(t, hookIn(repo), foreign)
	states := mustInstall(t, areas, lookup)
	if statesOf(states) != "project/a:refused" || readText(t, hookIn(repo)) != foreign {
		t.Errorf("states = %+v", states)
	}
}

func TestInstallReplacesItsOwnAndTheReferencesHook(t *testing.T) {
	for _, old := range []string{
		"#!/bin/sh\n# brain post-merge hook -- records that something landed.\nevents='C:/x'\nexit 0\n",
		"#!/bin/sh\n" + maintenance.HookMarker + " -- an older text\nexit 0\n",
	} {
		lookup, areas, repo := consenting(t, "main")
		writeFile(t, hookIn(repo), old)
		states := mustInstall(t, areas, lookup)
		if statesOf(states) != "project/a:installed" {
			t.Errorf("states = %+v", states)
		}
		if readText(t, hookIn(repo)) != maintenance.HookText() {
			t.Errorf("hook was not replaced: %q", readText(t, hookIn(repo)))
		}
		// A second run updates the record in place rather than adding one.
		mustInstall(t, areas, lookup)
		if n := strings.Count(readText(t, recordsOf(lookup)), "\n"); n != 1 {
			t.Errorf("hooks.tsv has %d lines, want 1", n)
		}
	}
}

func TestInstallKeepsTheRecordsOfOtherAreas(t *testing.T) {
	lookup, areas, _ := consenting(t, "main")
	writeFile(t, recordsOf(lookup), "project/z\tC:/elsewhere\tC:/elsewhere/.git/hooks/post-merge\n")
	mustInstall(t, areas, lookup)
	got := readText(t, recordsOf(lookup))
	if !strings.HasPrefix(got, "project/z\t") || strings.Count(got, "\n") != 2 {
		t.Errorf("hooks.tsv = %q", got)
	}
}

func TestInstallReportsAnAreaThatIsNoRepository(t *testing.T) {
	requireGit(t)
	lookup := config.ArtifactLookup{Primary: t.TempDir()}
	dir := t.TempDir()
	declare(t, dir, "project/a", "[maintenance]\non_merge = true\n")
	// Slashed, as a registry writes it; the report spells it natively.
	areas := []config.Area{{Scope: "project/a", Path: filepath.ToSlash(dir)}}
	states := mustInstall(t, areas, lookup)
	want := maintenance.HookState{State: "no repository", Scope: "project/a", Repo: dir}
	if len(states) != 1 || states[0] != want {
		t.Fatalf("states = %+v, want %+v", states, want)
	}
	if !maintenance.HookFailed(states) {
		t.Error("HookFailed is false for an area outside any repository")
	}
}

func TestAHooksDirectoryGitCannotNameIsNoRepository(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	// Without the hooks directory a broken install writes a relative
	// post-merge; it must land here, not in the package directory.
	t.Chdir(t.TempDir())
	states, err := maintenance.InstallHooks(areas, lookup, failingGit(repo, "--git-path"))
	if err != nil {
		t.Fatal(err)
	}
	if statesOf(states) != "project/a:no repository" {
		t.Errorf("states = %+v", states)
	}
}

func TestARepositoryGitCannotNameIsNoRepository(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	// The hooks directory alone is no repository: without its top level the
	// hook would go in with an empty repository beside it.
	states, err := maintenance.InstallHooks(areas, lookup, failingGit(repo, "--show-toplevel"))
	if err != nil {
		t.Fatal(err)
	}
	if statesOf(states) != "project/a:no repository" {
		t.Errorf("states = %+v", states)
	}
	if exists(hookIn(repo)) {
		t.Error("a hook was written into a repository git could not name")
	}
}

func TestInstallFailsOnAHookItCannotReplace(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("a read-only file refuses a rename onto it only on Windows")
	}
	lookup, areas, repo := consenting(t, "main")
	earlier := "#!/bin/sh\n# brain post-merge hook\n"
	writeFile(t, hookIn(repo), earlier)
	if err := os.Chmod(hookIn(repo), 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(hookIn(repo), 0o644) })
	// Making the file executable afterwards must not wash out the failed
	// write: the hook would be reported installed with the old text.
	if _, err := maintenance.InstallHooks(areas, lookup, realGit); err == nil {
		t.Error("InstallHooks succeeded on a hook it could not replace")
	}
	if got := readText(t, hookIn(repo)); got != earlier {
		t.Errorf("hook = %q", got)
	}
}

func TestStatusNamesEveryState(t *testing.T) {
	for name, row := range map[string]struct {
		change func(t *testing.T, lookup config.ArtifactLookup, areas []config.Area, repo string) []config.Area
		want   string
	}{
		"installed": {func(t *testing.T, lookup config.ArtifactLookup, areas []config.Area, repo string) []config.Area {
			mustInstall(t, areas, lookup)
			return areas
		}, "installed"},
		"missing": {func(t *testing.T, lookup config.ArtifactLookup, areas []config.Area, repo string) []config.Area {
			mustInstall(t, areas, lookup)
			if err := os.Remove(hookIn(repo)); err != nil {
				t.Fatal(err)
			}
			return areas
		}, "missing"},
		"not installed": {func(t *testing.T, lookup config.ArtifactLookup, areas []config.Area, repo string) []config.Area {
			return areas
		}, "not installed"},
		"unrecorded": {func(t *testing.T, lookup config.ArtifactLookup, areas []config.Area, repo string) []config.Area {
			mustInstall(t, areas, lookup)
			if err := os.Remove(recordsOf(lookup)); err != nil {
				t.Fatal(err)
			}
			return areas
		}, "unrecorded"},
		"consent withdrawn": {func(t *testing.T, lookup config.ArtifactLookup, areas []config.Area, repo string) []config.Area {
			mustInstall(t, areas, lookup)
			declare(t, repo, "project/a", "[maintenance]\non_merge = false\n")
			return areas
		}, "orphaned"},
		"area deregistered": {func(t *testing.T, lookup config.ArtifactLookup, areas []config.Area, repo string) []config.Area {
			mustInstall(t, areas, lookup)
			return nil
		}, "orphaned"},
		// The file is still there, but git now looks for hooks elsewhere
		// and never runs it.
		"hooks path changed": {func(t *testing.T, lookup config.ArtifactLookup, areas []config.Area, repo string) []config.Area {
			mustInstall(t, areas, lookup)
			mustGit(t, repo, "config", "core.hooksPath", ".githooks")
			return areas
		}, "moved"},
	} {
		lookup, areas, repo := consenting(t, "main")
		areas = row.change(t, lookup, areas, repo)
		states, err := maintenance.HookStatus(areas, lookup, realGit)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if statesOf(states) != "project/a:"+row.want {
			t.Errorf("%s: states = %+v, want %s", name, states, row.want)
		}
		if states[0].Repo != topLevel(t, repo) {
			t.Errorf("%s: Repo = %q", name, states[0].Repo)
		}
	}
}

// hookOf is the hook path the way status names it: from git's answer, in
// its long spelling, which differs from t.TempDir's where TEMP is an 8.3
// short name, as on a GitHub runner.
func hookOf(t *testing.T, repo string) string {
	t.Helper()
	return filepath.Join(topLevel(t, repo), ".git", "hooks", "post-merge")
}

// A file of someone else's where the hook would go is named as such: the
// bare path read as if the hook were missing there.
func TestStatusNamesAForeignHookFile(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	states, err := maintenance.HookStatus(areas, lookup, realGit)
	if err != nil || len(states) != 1 || states[0].Detail != hookOf(t, repo) {
		t.Fatalf("no file: %+v, %v", states, err)
	}
	writeFile(t, hookIn(repo), "#!/bin/sh\necho mine\n")
	states, err = maintenance.HookStatus(areas, lookup, realGit)
	want := maintenance.HookState{State: "not installed", Scope: "project/a", Repo: states[0].Repo, Detail: hookOf(t, repo) + ": another hook"}
	if err != nil || len(states) != 1 || states[0] != want {
		t.Fatalf("foreign file: %+v, %v; want %+v", states, err, want)
	}
}

func TestStatusOfAnAreaThatIsNoRepositoryIsNotInstalled(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	declare(t, dir, "project/a", "[maintenance]\non_merge = true\n")
	states, err := maintenance.HookStatus([]config.Area{{Scope: "project/a", Path: dir}},
		config.ArtifactLookup{Primary: t.TempDir()}, realGit)
	if err != nil {
		t.Fatal(err)
	}
	want := maintenance.HookState{State: "not installed", Scope: "project/a", Repo: dir}
	if len(states) != 1 || states[0] != want {
		t.Errorf("states = %+v, want %+v", states, want)
	}
}

func TestADamagedRecordLineIsSkipped(t *testing.T) {
	lookup, areas, _ := consenting(t, "main")
	mustInstall(t, areas, lookup)
	good := readText(t, recordsOf(lookup))
	writeFile(t, recordsOf(lookup), "only\ttwo\n"+good+"one\ttoo\tmany\tfields\there\tsix\n")
	states, err := maintenance.HookStatus(areas, lookup, realGit)
	if err != nil {
		t.Fatal(err)
	}
	if statesOf(states) != "project/a:installed" {
		t.Errorf("states = %+v", states)
	}
}

func TestAReferenceRecordLineIsReadByItsFirstThreeFields(t *testing.T) {
	for _, tail := range []string{"\tmain", "\tmain\tC:/state/maintenance/merge-events.tsv"} {
		lookup, areas, repo := consenting(t, "main")
		top := topLevel(t, repo)
		// The reference wrote native paths; the record is compared cleaned.
		writeFile(t, hookIn(repo), "#!/bin/sh\n# brain post-merge hook\nexit 0\n")
		writeFile(t, recordsOf(lookup), "project/a\t"+filepath.ToSlash(top)+"\t"+hookIn(repo)+tail+"\n")
		states, err := maintenance.HookStatus(areas, lookup, realGit)
		if err != nil {
			t.Fatal(err)
		}
		if statesOf(states) != "project/a:installed" {
			t.Errorf("%q: states = %+v", tail, states)
		}
	}
}

func TestRemoveTakesOurHookAndForgetsIt(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	mustInstall(t, areas, lookup)
	states, err := maintenance.RemoveHooks(areas, lookup, realGit)
	if err != nil {
		t.Fatal(err)
	}
	if statesOf(states) != "project/a:removed" {
		t.Errorf("states = %+v", states)
	}
	if exists(hookIn(repo)) || exists(recordsOf(lookup)) {
		t.Error("the hook or its record survived the removal")
	}
}

func TestRemoveTakesAHookWhoseRecordWasLost(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	mustInstall(t, areas, lookup)
	if err := os.Remove(recordsOf(lookup)); err != nil {
		t.Fatal(err)
	}
	states, err := maintenance.RemoveHooks(areas, lookup, realGit)
	if err != nil {
		t.Fatal(err)
	}
	if statesOf(states) != "project/a:removed" || exists(hookIn(repo)) {
		t.Errorf("states = %+v", states)
	}
}

func TestRemoveLeavesAForeignHook(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	mustInstall(t, areas, lookup)
	record := readText(t, recordsOf(lookup))
	foreign := "#!/bin/sh\necho mine\n"
	writeFile(t, hookIn(repo), foreign)
	states, err := maintenance.RemoveHooks(areas, lookup, realGit)
	if err != nil {
		t.Fatal(err)
	}
	if statesOf(states) != "project/a:refused" || !maintenance.HookFailed(states) {
		t.Errorf("states = %+v", states)
	}
	if readText(t, hookIn(repo)) != foreign || readText(t, recordsOf(lookup)) != record {
		t.Error("the foreign hook or its record was touched")
	}
}

func TestRemoveOfAMissingHookForgetsIt(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	mustInstall(t, areas, lookup)
	if err := os.Remove(hookIn(repo)); err != nil {
		t.Fatal(err)
	}
	states, err := maintenance.RemoveHooks(areas, lookup, realGit)
	if err != nil {
		t.Fatal(err)
	}
	if statesOf(states) != "project/a:missing" || exists(recordsOf(lookup)) {
		t.Errorf("states = %+v", states)
	}
}

func TestRemoveWithoutAnyInstallationIsNotAnError(t *testing.T) {
	lookup, areas, _ := consenting(t, "main")
	states, err := maintenance.RemoveHooks(areas, lookup, realGit)
	if err != nil || len(states) != 0 {
		t.Errorf("states, err = %+v, %v", states, err)
	}
}

// twoAreas is one repository holding two consenting areas, the second in a
// subdirectory with its own declaration.
func twoAreas(t *testing.T) (config.ArtifactLookup, []config.Area, string) {
	t.Helper()
	lookup, areas, repo := consenting(t, "main")
	sub := filepath.Join(repo, "sub")
	declare(t, sub, "project/b", "[maintenance]\non_merge = true\n")
	return lookup, append(areas, config.Area{Scope: "project/b", Path: sub}), repo
}

func TestTwoAreasInOneRepositoryShareTheHook(t *testing.T) {
	lookup, areas, repo := twoAreas(t)
	states := mustInstall(t, areas, lookup)
	if statesOf(states) != "project/a:installed,project/b:installed" || states[0].Detail != states[1].Detail {
		t.Errorf("states = %+v", states)
	}
	if n := strings.Count(readText(t, recordsOf(lookup)), "\n"); n != 2 {
		t.Errorf("hooks.tsv has %d lines, want 2", n)
	}
	states, err := maintenance.RemoveHooks(areas, lookup, realGit)
	if err != nil {
		t.Fatal(err)
	}
	if statesOf(states) != "project/a:removed,project/b:removed" || exists(hookIn(repo)) {
		t.Errorf("states = %+v", states)
	}
}

func TestAnUnreadableDeclarationStopsEveryCommand(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	merge(t, repo)
	writeFile(t, filepath.Join(repo, ".loomux", "config.toml"), "[area]\nscope = 1\n")
	if _, err := maintenance.InstallHooks(areas, lookup, realGit); err == nil {
		t.Error("InstallHooks read past a broken declaration")
	}
	if _, err := maintenance.HookStatus(areas, lookup, realGit); err == nil {
		t.Error("HookStatus read past a broken declaration")
	}
	if _, err := maintenance.RemoveHooks(areas, lookup, realGit); err == nil {
		t.Error("RemoveHooks read past a broken declaration")
	}
	if _, err := maintenance.RecordMerge(repo, areas, lookup, realGit, time.Now()); err == nil {
		t.Error("RecordMerge read past a broken declaration")
	}
}

func TestARecordFileThatIsNotUTF8StopsEveryCommand(t *testing.T) {
	lookup, areas, _ := consenting(t, "main")
	writeFile(t, recordsOf(lookup), "\xff\xfe\n")
	if _, err := maintenance.InstallHooks(areas, lookup, realGit); err == nil {
		t.Error("InstallHooks read past a broken record file")
	}
	if _, err := maintenance.HookStatus(areas, lookup, realGit); err == nil {
		t.Error("HookStatus read past a broken record file")
	}
	if _, err := maintenance.RemoveHooks(areas, lookup, realGit); err == nil {
		t.Error("RemoveHooks read past a broken record file")
	}
}

func TestInstallFailsWhereTheHooksDirectoryCannotBeMade(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	blocker := filepath.Join(repo, "blocker")
	writeFile(t, blocker, "a file, not a directory\n")
	// git itself refuses a core.hooksPath below a file, so the answer that
	// names one is put into its mouth.
	git := func(dir string, args ...string) (string, error) {
		if args[len(args)-1] == "hooks" {
			return filepath.ToSlash(filepath.Join(blocker, "hooks")), nil
		}
		return realGit(dir, args...)
	}
	_, err := maintenance.InstallHooks(areas, lookup, git)
	// The directory that could not be made is the cause worth naming, not
	// the temporary file that then has nowhere to go.
	if !failedToMakeADirectory(err) {
		t.Errorf("InstallHooks without a hooks directory = %v, want the mkdir failure", err)
	}
}

// failedToMakeADirectory says whether err carries a failed mkdir.
func failedToMakeADirectory(err error) bool {
	var failed *fs.PathError
	return errors.As(err, &failed) && failed.Op == "mkdir"
}

func TestInstallFailsWhereTheRecordCannotBeWritten(t *testing.T) {
	lookup, areas, _ := consenting(t, "main")
	writeFile(t, filepath.Join(lookup.Primary, "maintenance"), "a file, not a directory\n")
	if _, err := maintenance.InstallHooks(areas, lookup, realGit); !failedToMakeADirectory(err) {
		t.Errorf("InstallHooks without a record directory = %v, want the mkdir failure", err)
	}
}

func TestRemoveFailsWhereTheEmptyRecordCannotBeDeleted(t *testing.T) {
	lookup, areas, _ := consenting(t, "main")
	// A directory in the record's place reads as no record and does not
	// go away.
	writeFile(t, filepath.Join(recordsOf(lookup), "inside"), "x\n")
	if _, err := maintenance.RemoveHooks(areas, lookup, realGit); err == nil {
		t.Error("RemoveHooks reported success over a record it could not delete")
	}
}

// Windows only: a file another handle holds open refuses deletion there,
// while POSIX unlinks it regardless.
func TestRemoveFailsOnAHookThatWillNotGo(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("an open file does not refuse deletion outside Windows")
	}
	for _, loseRecord := range []bool{false, true} {
		lookup, areas, repo := consenting(t, "main")
		mustInstall(t, areas, lookup)
		if loseRecord {
			if err := os.Remove(recordsOf(lookup)); err != nil {
				t.Fatal(err)
			}
		}
		held, err := os.Open(hookIn(repo))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := maintenance.RemoveHooks(areas, lookup, realGit); err == nil {
			t.Errorf("lost record %v: RemoveHooks succeeded on a hook it could not delete", loseRecord)
		}
		held.Close()
	}
}

// merge lands branch side on main in dir with a merge commit, so ORIG_HEAD
// and HEAD name a range.
func merge(t *testing.T, dir string) {
	t.Helper()
	base := mustGit(t, dir, "symbolic-ref", "--short", "HEAD")
	mustGit(t, dir, "checkout", "-q", "-b", "side-"+filepath.Base(dir))
	mustGit(t, dir, "commit", "-q", "--allow-empty", "-m", "side")
	mustGit(t, dir, "checkout", "-q", base)
	mustGit(t, dir, "merge", "-q", "--no-ff", "--no-edit", "side-"+filepath.Base(dir))
}

// someTime is not UTC and carries nanoseconds, so both the conversion and
// the cut to the second are seen.
var someTime = time.Date(2026, 9, 24, 14, 30, 15, 123456789, time.FixedZone("CEST", 2*3600))

func TestRecordAppendsOneEventForAMergeOnTheConsentingBranch(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	merge(t, repo)
	recorded, err := maintenance.RecordMerge(repo, areas, lookup, realGit, someTime)
	if err != nil || !recorded {
		t.Fatalf("RecordMerge = %v, %v", recorded, err)
	}
	events, err := maintenance.ReadEvents(lookup.Primary)
	if err != nil || len(events) != 1 {
		t.Fatalf("ReadEvents = %+v, %v", events, err)
	}
	e := events[0]
	if e.Repo != mustGit(t, repo, "rev-parse", "--show-toplevel") ||
		e.First != mustGit(t, repo, "rev-parse", "ORIG_HEAD") ||
		e.Last != mustGit(t, repo, "rev-parse", "HEAD") || e.Branch != "main" {
		t.Errorf("event = %+v", e)
	}
	if !e.At.Equal(someTime.Truncate(time.Second)) {
		t.Errorf("At = %v, want %v", e.At, someTime.Truncate(time.Second))
	}
	if line := readText(t, maintenance.EventsPath(lookup.Primary)); !strings.HasSuffix(line, "\tmain\t2026-09-24T12:30:15Z\n") {
		t.Errorf("line = %q", line)
	}
}

func TestRecordIsSilentOnAnotherBranch(t *testing.T) {
	lookup, areas, repo := consenting(t, "trunk")
	merge(t, repo)
	recorded, err := maintenance.RecordMerge(repo, areas, lookup, realGit, someTime)
	if err != nil || recorded {
		t.Errorf("RecordMerge = %v, %v", recorded, err)
	}
	if exists(maintenance.EventsPath(lookup.Primary)) {
		t.Error("an event file was written for another branch")
	}
}

func TestRecordIgnoresAMergeInAForeignRepository(t *testing.T) {
	lookup, areas, _ := consenting(t, "main")
	foreign := newRepo(t)
	merge(t, foreign)
	recorded, err := maintenance.RecordMerge(foreign, areas, lookup, realGit, someTime)
	if err != nil || recorded {
		t.Errorf("RecordMerge = %v, %v", recorded, err)
	}
}

func TestRecordIgnoresAnAreaWhoseRepositoryGitCannotName(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	merge(t, repo)
	recorded, err := maintenance.RecordMerge(repo, areas, lookup, failingGit(repo, "--git-common-dir"), someTime)
	if err != nil || recorded {
		t.Errorf("RecordMerge = %v, %v", recorded, err)
	}
}

func TestRecordIgnoresAnAreaThatDoesNotConsent(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	declare(t, repo, "project/a", "")
	merge(t, repo)
	recorded, err := maintenance.RecordMerge(repo, areas, lookup, realGit, someTime)
	if err != nil || recorded {
		t.Errorf("RecordMerge = %v, %v", recorded, err)
	}
}

func TestRecordCountsAMergeInAWorktreeOfTheArea(t *testing.T) {
	lookup, areas, repo := consenting(t, "feature")
	worktree := filepath.Join(t.TempDir(), "wt")
	mustGit(t, repo, "worktree", "add", "-q", "-b", "feature", worktree)
	merge(t, worktree)
	recorded, err := maintenance.RecordMerge(worktree, areas, lookup, realGit, someTime)
	if err != nil || !recorded {
		t.Fatalf("RecordMerge = %v, %v", recorded, err)
	}
	events, err := maintenance.ReadEvents(lookup.Primary)
	if err != nil || len(events) != 1 || events[0].Repo != mustGit(t, worktree, "rev-parse", "--show-toplevel") {
		t.Errorf("events = %+v, %v", events, err)
	}
}

// The repository is compared by the common git directory git names, and git
// names it in its long spelling whichever way it is reached: an area
// registered through a junction, and a merge made through one, still meet.
func TestRecordMeetsAnAreaReachedThroughAJunction(t *testing.T) {
	requireJunctions(t)
	lookup, areas, repo := consenting(t, "main")
	link := filepath.Join(t.TempDir(), "link")
	junction(t, link, repo)
	merge(t, repo)
	for name, c := range map[string]struct{ area, dir string }{
		"area through the junction":  {link, repo},
		"merge through the junction": {repo, link},
	} {
		through := []config.Area{areas[0]}
		through[0].Path = c.area
		recorded, err := maintenance.RecordMerge(c.dir, through, lookup, realGit, someTime)
		if err != nil || !recorded {
			t.Errorf("%s: RecordMerge = %v, %v", name, recorded, err)
		}
	}
}

func TestRecordWritesOneLineWhenTwoAreasShareTheRepository(t *testing.T) {
	lookup, areas, repo := twoAreas(t)
	merge(t, repo)
	recorded, err := maintenance.RecordMerge(repo, areas, lookup, realGit, someTime)
	if err != nil || !recorded {
		t.Fatalf("RecordMerge = %v, %v", recorded, err)
	}
	if n := strings.Count(readText(t, maintenance.EventsPath(lookup.Primary)), "\n"); n != 1 {
		t.Errorf("%d lines, want 1", n)
	}
}

func TestRecordCreatesTheMaintenanceDirectory(t *testing.T) {
	_, areas, repo := consenting(t, "main")
	lookup := config.ArtifactLookup{Primary: filepath.Join(t.TempDir(), "not", "yet")}
	merge(t, repo)
	recorded, err := maintenance.RecordMerge(repo, areas, lookup, realGit, someTime)
	if err != nil || !recorded || !exists(maintenance.EventsPath(lookup.Primary)) {
		t.Errorf("RecordMerge = %v, %v", recorded, err)
	}
}

func TestRecordWithoutORIGHEADWritesNothing(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	recorded, err := maintenance.RecordMerge(repo, areas, lookup, realGit, someTime)
	if err != nil || recorded {
		t.Errorf("RecordMerge = %v, %v", recorded, err)
	}
	if exists(maintenance.EventsPath(lookup.Primary)) {
		t.Error("an event file was written without a merge")
	}
}

func TestRecordCarriesAFailedAppend(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	merge(t, repo)
	writeFile(t, filepath.Join(lookup.Primary, "maintenance"), "a file, not a directory\n")
	if _, err := maintenance.RecordMerge(repo, areas, lookup, realGit, someTime); err == nil {
		t.Error("RecordMerge succeeded without writing its event")
	}
}

// A hook that cannot be written stops the run, but the hooks written before
// it keep their records: without one, status calls them unrecorded and the
// next run cannot tell them from a hook somebody else left.
func TestInstallRecordsWhatItWroteBeforeAFailure(t *testing.T) {
	lookup, areas, repo := consenting(t, "main")
	other := newRepo(t)
	declare(t, other, "project/b", "[maintenance]\non_merge = true\n")
	areas = append(areas, config.Area{Scope: "project/b", Path: other})
	blocker := filepath.Join(other, "blocker")
	writeFile(t, blocker, "a file, not a directory\n")
	git := func(dir string, args ...string) (string, error) {
		if dir == other && args[len(args)-1] == "hooks" {
			return filepath.ToSlash(filepath.Join(blocker, "hooks")), nil
		}
		return realGit(dir, args...)
	}
	if _, err := maintenance.InstallHooks(areas, lookup, git); err == nil {
		t.Fatal("InstallHooks succeeded without a hooks directory for project/b")
	}
	top := topLevel(t, repo)
	want := "project/a\t" + top + "\t" + filepath.Join(top, ".git", "hooks", "post-merge") + "\n"
	if got := readText(t, recordsOf(lookup)); got != want {
		t.Errorf("hooks.tsv = %q, want the record of project/a", got)
	}
}

// A record may spell the hook another way than git does now -- an 8.3 short
// name, a junction -- while naming the same file; that hook is installed,
// not moved.
func TestStatusComparesTheHookByIdentityNotSpelling(t *testing.T) {
	requireJunctions(t)
	lookup, areas, repo := consenting(t, "main")
	mustInstall(t, areas, lookup)
	link := filepath.Join(t.TempDir(), "hooks-link")
	junction(t, link, filepath.Dir(hookIn(repo)))
	top := topLevel(t, repo)
	writeFile(t, recordsOf(lookup), "project/a\t"+top+"\t"+filepath.Join(link, "post-merge")+"\n")
	states, err := maintenance.HookStatus(areas, lookup, realGit)
	if err != nil {
		t.Fatal(err)
	}
	if statesOf(states) != "project/a:installed" {
		t.Errorf("states = %+v, want installed", states)
	}
}
