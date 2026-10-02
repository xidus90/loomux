package switchover

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/verify"
)

// templateUnderTest is the template the world runs: the embedded one, or the
// file LOOMUX_SWITCHOVER_TEMPLATE names, so that a mutation round runs the
// world against a changed copy without touching the tree.
func templateUnderTest(t *testing.T) string {
	t.Helper()
	path := os.Getenv("LOOMUX_SWITCHOVER_TEMPLATE")
	if path == "" {
		return applyTemplate
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

const worldIndex = "# Index\n\n- [[seite]]\n"

// world is the switch-over of one project, laid out as it stands on the
// user's machine: a project that already carries its wiki, a vault whose
// folder of the project holds the files next to the pages, the registry, the
// state directory and a loomux that only notes how it was called.
type world struct {
	t                                       *testing.T
	sh, gitExe                              string
	base, project, vault, bare              string
	registry, registryNew, configNew, state string
	loomux, argv, script                    string
	env                                     []string
	p                                       Params
}

func newWorld(t *testing.T) *world {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH")
	}
	gitExe, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git on PATH")
	}
	// A space, a hash and an apostrophe, as the user's paths have them.
	base := filepath.ToSlash(filepath.Join(t.TempDir(), "#GIT x"))
	w := &world{
		t: t, sh: sh, gitExe: gitExe, base: base,
		project:     base + "/proj's",
		vault:       base + "/brain knowledge",
		bare:        base + "/remote.git",
		registry:    base + "/state/registry.toml",
		registryNew: base + "/prep/registry.toml.new",
		configNew:   base + "/prep/config.toml.new",
		state:       base + "/state/areas/project-x",
		loomux:      base + "/bin/loomux",
		argv:        base + "/argv.log",
		script:      base + "/prep/apply.sh",
	}
	// The script finds its tools through /usr/bin, whatever comes first on
	// PATH: a find that lists nothing stands in for the one of Windows.
	w.write(base+"/fakebin/find", "#!/bin/sh\nexit 0\n")
	w.write(base+"/gitconfig", "")
	w.env = append(gitenv.Environ(),
		"PATH="+filepath.FromSlash(base+"/fakebin")+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GIT_CONFIG_GLOBAL="+base+"/gitconfig",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CEILING_DIRECTORIES="+filepath.FromSlash(base),
	)

	w.write(w.project+"/docs/wiki/seite.md", "# Seite\n")
	w.write(w.project+"/docs/wiki/index.md", worldIndex)
	w.write(w.project+"/.ultraloom/config.toml", "[old]\n")
	w.write(w.project+"/.claude/settings.json", fixtureSettings)
	w.write(w.project+"/README.md", "x\n")
	w.repo(w.project)

	w.write(w.vault+"/91 Projekte/x/_schema.md", "schema\n")
	w.write(w.vault+"/91 Projekte/x/audit.md", "audit\n")
	w.write(w.vault+"/91 Projekte/x/index.md", worldIndex)
	w.write(w.vault+"/90 Wiki/a.md", "a\n")
	w.repo(w.vault)
	w.git(base, "init", "-q", "--bare", w.bare)
	w.git(w.vault, "remote", "add", "origin", w.bare)

	w.write(w.registry, "old registry\n")
	w.write(w.registryNew, "new registry\n")
	w.write(w.configNew, "[area]\nname = \"x\"\n")
	w.write(w.state+"/catalog.json", "{}\n")
	w.setLoomux("")

	sum := sha256.Sum256([]byte("old registry\n"))
	w.p = Params{
		Name: "x", Project: w.project, Loomux: w.loomux, ConfigNew: w.configNew,
		Registry: w.registry, RegistryNew: w.registryNew, RegistrySum: hex.EncodeToString(sum[:]),
		WikiSrcs: []string{w.vault + "/91 Projekte/x"}, WikiDst: w.project + "/docs/wiki",
		Vault: w.vault, VaultOld: "91 Projekte/x", StateArea: w.state,
		OldFiles: []string{".ultraloom"}, OldHooks: fixtureNeedles,
	}
	return w
}

// setLoomux writes the stand-in loomux. Asked `dev switchover` alone, it
// prints the group's usage as loomux does, unlogged. Otherwise it appends its
// arguments to the argv file; prune-hooks removes a group on its first call
// and nothing after that, as the real one does on the same file; then it runs
// extra, if any.
func (w *world) setLoomux(extra string) {
	marker := w.base + "/prep/pruned"
	w.write(w.loomux, "#!/bin/sh\n"+
		"if [ \"$#\" = 2 ] && [ \"$1\" = dev ] && [ \"$2\" = switchover ]; then\n"+
		"  printf 'usage: loomux dev switchover <render|prune-hooks> [flags]\\n' >&2; exit 2\nfi\n"+
		"printf '%s\\n' \"$*\" >> '"+w.argv+"'\n"+
		"if [ \"$1 $2 $3\" = 'dev switchover prune-hooks' ]; then\n"+
		"  if [ -e '"+marker+"' ]; then echo 'removed nothing'; else : > '"+marker+"'; echo 'removed: PreToolUse: ulguard --root x'; fi\nfi\n"+
		extra)
}

// setOldLoomux writes a loomux from before the switchover group: it knows
// init and nothing of `dev switchover`.
func (w *world) setOldLoomux() {
	w.write(w.loomux, "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '"+w.argv+"'\n"+
		"if [ \"$1\" = dev ]; then printf 'loomux dev: unknown subcommand \"%s\"\\n' \"$2\" >&2; exit 2; fi\nexit 0\n")
}

func (w *world) write(path, text string) {
	w.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o755); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) read(path string) string {
	w.t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		w.t.Fatal(err)
	}
	return string(data)
}

// repo makes dir a repository with one commit of everything in it, with an
// identity and settings of its own: the user's global ones are shut out.
func (w *world) repo(dir string) {
	w.t.Helper()
	w.git(dir, "init", "-q")
	for _, kv := range [][2]string{{"user.name", "Test"}, {"user.email", "test@example.invalid"},
		{"commit.gpgsign", "false"}, {"core.autocrlf", "false"}, {"core.hooksPath", w.base + "/nohooks"}} {
		w.git(dir, "config", kv[0], kv[1])
	}
	w.commitAll(dir, "init")
}

func (w *world) commitAll(dir, message string) {
	w.t.Helper()
	w.git(dir, "add", "-A")
	w.git(dir, "commit", "-q", "-m", message)
}

func (w *world) git(dir string, args ...string) string {
	w.t.Helper()
	cmd := exec.Command(w.gitExe, append([]string{"-C", dir}, args...)...)
	cmd.Env = w.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		w.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// run renders the script from the world's parameters and runs it.
func (w *world) run(args ...string) (int, string, string) {
	w.t.Helper()
	text, err := render(templateUnderTest(w.t), w.p)
	if err != nil {
		w.t.Fatal(err)
	}
	w.write(w.script, text)
	cmd := exec.Command(w.sh, append([]string{w.script}, args...)...)
	cmd.Dir = w.base
	cmd.Env = w.env
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), stdout.String(), stderr.String()
	}
	if err != nil {
		w.t.Fatal(err)
	}
	return 0, stdout.String(), stderr.String()
}

// snapshot is every directory and file of the world with the hash of its
// content, but for what git keeps for itself (a status refreshes the index),
// the stand-in's argv file and the script.
func (w *world) snapshot() map[string]string {
	w.t.Helper()
	got := map[string]string{}
	err := filepath.WalkDir(w.base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(strings.TrimPrefix(path, filepath.FromSlash(w.base)))
		switch {
		case d.IsDir() && (d.Name() == ".git" || rel == "/remote.git" || rel == "/prep"):
			return filepath.SkipDir
		case rel == "/argv.log":
			return nil
		case d.IsDir():
			got[rel] = "dir"
			return nil
		}
		data, err := os.ReadFile(path)
		sum := sha256.Sum256(data)
		got[rel] = hex.EncodeToString(sum[:])
		return err
	})
	if err != nil {
		w.t.Fatal(err)
	}
	return got
}

func (w *world) sameAs(before map[string]string) {
	w.t.Helper()
	after := w.snapshot()
	for k, v := range before {
		if after[k] != v {
			w.t.Errorf("%s changed: %q -> %q", k, v, after[k])
		}
	}
	for k := range after {
		if _, ok := before[k]; !ok {
			w.t.Errorf("%s appeared", k)
		}
	}
}

func (w *world) calls() []string {
	w.t.Helper()
	data, err := os.ReadFile(w.argv)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		w.t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func (w *world) wantAbort(code int, stderr, want string) {
	w.t.Helper()
	if code != 3 || !strings.Contains(stderr, "abort: ") || !strings.Contains(stderr, want) {
		w.t.Fatalf("code %d, stderr %q; want 3 with %q", code, stderr, want)
	}
}

func TestApplyCheckWritesNothing(t *testing.T) {
	w := newWorld(t)
	before := w.snapshot()
	code, out, errOut := w.run("--check")
	if code != 0 {
		t.Fatalf("code %d: %s\n%s", code, out, errOut)
	}
	for _, want := range []string{
		"would run: cp " + w.registryNew + " " + w.registry,
		"registry: would be replaced",
		"wiki: would be merged into " + w.project + "/docs/wiki",
		"would run: mkdir -p " + w.project + "/.loomux",
		"would run: " + w.loomux + " init --yes --root " + w.project,
		"would run: " + w.loomux + " dev switchover prune-hooks --file " + w.project + "/.claude/settings.json --match ulguard",
		"would run: rm -rf " + w.project + "/.ultraloom",
		"would run: git -C " + w.vault + " --literal-pathspecs rm -r -q -- 91 Projekte/x",
		"would run: mv " + w.state + " " + w.state + ".alt",
		"done: " + w.project,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout lacks %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "config: would be written\nprobation: would be started, .loomux/armed.toml written\n") {
		t.Errorf("stdout lacks the probation it would start:\n%s", out)
	}
	w.sameAs(before)
	if calls := w.calls(); calls != nil {
		t.Fatalf("--check ran loomux: %q", calls)
	}
}

// Step 4 starts the probation where it writes the configuration: init runs
// after it, finds a configuration and would start none.
func TestApplyStartsTheProbationWhereItWritesTheConfiguration(t *testing.T) {
	w := newWorld(t)
	out := w.full()
	if got := w.read(w.project + "/.loomux/armed.toml"); got != (verify.ArmedSet{Exists: true}).Text() {
		t.Fatalf("armed.toml %q, want the text init writes", got)
	}
	if !strings.Contains(out, "config: written\nprobation: started, .loomux/armed.toml written\n") {
		t.Fatalf("stdout:\n%s", out)
	}
	// A second run finds the configuration and leaves the lanes a human or a
	// commit armed since.
	w.write(w.project+"/.loomux/armed.toml", "armed = [\"lint/go@.\"]\n")
	w.commitAll(w.project, "switch over")
	before := w.snapshot()
	code, out, errOut := w.run()
	if code != 0 || strings.Contains(out, "probation:") {
		t.Fatalf("code %d:\n%s\n%s", code, out, errOut)
	}
	w.sameAs(before)
}

// A project that has armed lanes and no configuration -- a Go project init
// set up, or a run stopped after the lanes -- keeps them: the script writes
// the configuration and leaves the file as it stands.
func TestApplyKeepsStandingArmedLanes(t *testing.T) {
	const armed = "armed = [\"lint/go@.\"]\n"
	w := newWorld(t)
	w.write(w.project+"/.loomux/armed.toml", armed)
	w.commitAll(w.project, "armed lanes")
	code, out, errOut := w.run("--check")
	if code != 0 || !strings.Contains(out, "config: would be written\nprobation: would be kept, .loomux/armed.toml stands\n") {
		t.Fatalf("--check: code %d\n%s\n%s", code, out, errOut)
	}
	out = w.full()
	if got := w.read(w.project + "/.loomux/armed.toml"); got != armed {
		t.Fatalf("armed.toml %q, want it kept", got)
	}
	if !exists(w.project+"/.loomux/config.toml") || strings.Contains(out, "probation: started") ||
		!strings.Contains(out, "config: written\nprobation: kept, .loomux/armed.toml stands\n") {
		t.Fatalf("stdout:\n%s", out)
	}
}

// A project that has a configuration is not put into probation by the script.
func TestApplyStartsNoProbationOverAStandingConfiguration(t *testing.T) {
	w := newWorld(t)
	w.write(w.project+"/.loomux/config.toml", "[area]\nname = \"mine\"\n")
	w.commitAll(w.project, "own config")
	out := w.full()
	if exists(w.project+"/.loomux/armed.toml") || strings.Contains(out, "probation:") {
		t.Fatalf("armed.toml written over a standing configuration:\n%s", out)
	}
}

func TestApplyRefusesAProjectWithUncommittedChanges(t *testing.T) {
	w := newWorld(t)
	f, err := os.OpenFile(w.project+"/docs/wiki/seite.md", os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("x\n")
	f.Close()
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, w.project+" is no repository or has uncommitted changes")
	w.sameAs(before)
	code, _, errOut = w.run("--check")
	w.wantAbort(code, errOut, "uncommitted changes")
}

func TestApplyRefusesAProjectThatIsNoRepository(t *testing.T) {
	w := newWorld(t)
	if err := os.Rename(w.project+"/.git", w.project+"/dot-git"); err != nil {
		t.Fatal(err)
	}
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, w.project+" is no repository")
	w.sameAs(before)
}

func TestApplyRefusesAVaultWithUncommittedChanges(t *testing.T) {
	w := newWorld(t)
	w.write(w.vault+"/new.md", "new\n")
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, w.vault+" is no repository or has uncommitted changes")
	w.sameAs(before)
}

func TestApplyRefusesAVaultWithoutARemote(t *testing.T) {
	w := newWorld(t)
	w.git(w.vault, "remote", "remove", "origin")
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, w.vault+" has no remote")
	w.sameAs(before)
}

func TestApplyRefusesAVaultWithoutACommit(t *testing.T) {
	w := newWorld(t)
	empty := w.base + "/empty vault"
	w.git(w.base, "init", "-q", empty)
	w.git(empty, "remote", "add", "origin", w.bare)
	w.p.Vault = empty
	w.p.WikiSrcs = []string{empty + "/91 Projekte/x"}
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, empty+" has no commit")
	w.sameAs(before)
}

func TestApplyRefusesARegistryThatChanged(t *testing.T) {
	w := newWorld(t)
	w.write(w.registry, "changed\n")
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, "the registry changed since this script was prepared")
	w.sameAs(before)
	if got := w.read(w.registry); got != "changed\n" {
		t.Fatalf("registry %q", got)
	}
}

func TestApplyRefusesAMissingLoomux(t *testing.T) {
	w := newWorld(t)
	w.p.Loomux = w.base + "/bin/gone"
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, w.base+"/bin/gone is no program")
	w.sameAs(before)
}

func TestApplyRefusesAMissingConfiguration(t *testing.T) {
	w := newWorld(t)
	w.p.ConfigNew = w.base + "/prep/gone.toml"
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, w.base+"/prep/gone.toml does not exist")
	w.sameAs(before)
}

func TestApplyNeedsNoNewConfigurationWhereOneStands(t *testing.T) {
	w := newWorld(t)
	w.write(w.project+"/.loomux/config.toml", "[area]\nname = \"mine\"\n")
	w.commitAll(w.project, "own config")
	w.p.ConfigNew = w.base + "/prep/gone.toml"
	code, out, errOut := w.run()
	if code != 0 || !strings.Contains(out, "config: kept") {
		t.Fatalf("code %d:\n%s\n%s", code, out, errOut)
	}
}

func TestApplyRefusesAMissingNewRegistry(t *testing.T) {
	w := newWorld(t)
	w.p.RegistryNew = w.base + "/prep/gone.toml"
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, w.base+"/prep/gone.toml does not exist")
	w.sameAs(before)
}

func TestApplyRefusesAnUnknownArgument(t *testing.T) {
	w := newWorld(t)
	before := w.snapshot()
	for _, args := range [][]string{{"--chek"}, {"--check", "x"}, {"check"}} {
		code, _, errOut := w.run(args...)
		if code != 2 || !strings.Contains(errOut, "usage: sh apply.sh [--check]") {
			t.Errorf("%q: code %d, stderr %q", args, code, errOut)
		}
	}
	w.sameAs(before)
	if calls := w.calls(); calls != nil {
		t.Fatalf("ran loomux: %q", calls)
	}
}

// full runs the whole switch-over and checks its result.
func (w *world) full() string {
	w.t.Helper()
	code, out, errOut := w.run()
	if code != 0 {
		w.t.Fatalf("code %d:\n%s\n%s", code, out, errOut)
	}
	return out
}

func TestApplyMovesTheWikiIntoTheProject(t *testing.T) {
	w := newWorld(t)
	vaultHead := w.git(w.vault, "rev-parse", "HEAD")
	// A file the project has already is not written again.
	old := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes(w.project+"/docs/wiki/index.md", old, old); err != nil {
		t.Fatal(err)
	}
	out := w.full()
	if info, err := os.Stat(w.project + "/docs/wiki/index.md"); err != nil || !info.ModTime().Equal(old) {
		t.Errorf("index.md was written again: %v %v", info.ModTime(), err)
	}

	if got := w.read(w.project + "/.loomux/config.toml"); got != w.read(w.configNew) {
		t.Errorf("config.toml %q", got)
	}
	for name, want := range map[string]string{"seite.md": "# Seite\n", "index.md": worldIndex,
		"_schema.md": "schema\n", "audit.md": "audit\n"} {
		if got := w.read(w.project + "/docs/wiki/" + name); got != want {
			t.Errorf("docs/wiki/%s: %q, want %q", name, got, want)
		}
	}
	for _, gone := range []string{w.project + "/.ultraloom", w.vault + "/91 Projekte/x", w.project + "/docs/wiki.staging", w.state} {
		if exists(gone) {
			t.Errorf("%s is still there", gone)
		}
	}
	if w.read(w.vault+"/90 Wiki/a.md") != "a\n" || w.read(w.state+".alt/catalog.json") != "{}\n" {
		t.Error("the rest of the vault or the renamed state is not as it was")
	}
	if w.read(w.registry) != "new registry\n" || w.read(w.registry+".bak") != "old registry\n" {
		t.Errorf("registry %q, backup %q", w.read(w.registry), w.read(w.registry+".bak"))
	}
	if parent := w.git(w.vault, "rev-parse", "HEAD~1"); parent != vaultHead {
		t.Errorf("the vault's new commit does not follow %s", vaultHead)
	}
	if subject := w.git(w.vault, "log", "-1", "--format=%s"); subject != "chore: move the wiki 91 Projekte/x into its project" {
		t.Errorf("vault commit %q", subject)
	}
	if status := w.git(w.vault, "status", "--porcelain"); status != "" {
		t.Errorf("vault left dirty: %q", status)
	}
	wantCalls := []string{
		"init --yes --root " + w.project,
		"dev switchover prune-hooks --file " + w.project + "/.claude/settings.json --match ulguard --match brain wiki-gate --match ultraloom hook",
	}
	if calls := w.calls(); !reflect.DeepEqual(calls, wantCalls) {
		t.Errorf("loomux was called\n%q\nwant\n%q", calls, wantCalls)
	}
	for _, want := range []string{
		"registry: replaced, the old one is " + w.registry + ".bak",
		"wiki: merged into " + w.project + "/docs/wiki",
		"config: written",
		"init: ran (idempotent)",
		"removed: PreToolUse: ulguard --root x",
		"hooks: pruned",
		"files: removed .ultraloom",
		"vault: removed 91 Projekte/x and committed",
		"state: renamed to " + w.state + ".alt",
		"done: " + w.project,
	} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("stdout lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "files: already removed") {
		t.Errorf("a run that removed files calls them already removed:\n%s", out)
	}
	if strings.Contains(out, "would run") {
		t.Errorf("a real run speaks of what it would do:\n%s", out)
	}
}

func TestApplyASecondTimeChangesNothing(t *testing.T) {
	w := newWorld(t)
	w.full()
	w.commitAll(w.project, "switch over")
	// A configuration the human changed after the first run stays his.
	w.write(w.project+"/.loomux/config.toml", "[area]\nname = \"mine\"\n")
	w.commitAll(w.project, "own config")
	vaultHead := w.git(w.vault, "rev-parse", "HEAD")
	before := w.snapshot()

	code, out, errOut := w.run()
	if code != 0 {
		t.Fatalf("code %d:\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{
		"registry: already replaced",
		"wiki: already moved",
		"config: kept, " + w.project + "/.loomux/config.toml exists",
		"init: ran (idempotent)",
		"hooks: already pruned",
		"files: already removed",
		"vault: already removed 91 Projekte/x",
		"state: already renamed",
		"done: " + w.project,
	} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("stdout lacks %q:\n%s", want, out)
		}
	}
	w.sameAs(before)
	if head := w.git(w.vault, "rev-parse", "HEAD"); head != vaultHead {
		t.Errorf("the vault got a commit: %s -> %s", vaultHead, head)
	}
}

func TestApplyStopsAtAWikiFileThatDiffers(t *testing.T) {
	w := newWorld(t)
	w.write(w.vault+"/91 Projekte/x/index.md", "# Index of the vault\n")
	w.commitAll(w.vault, "other index")
	w.write(w.project+"/docs/wiki/_schema.md", "the project's schema\n")
	w.commitAll(w.project, "own schema")
	vaultHead := w.git(w.vault, "rev-parse", "HEAD")
	before := w.snapshot()

	// --check builds no staging directory, so it cannot compare: it says so
	// and writes nothing.
	code, out, errOut := w.run("--check")
	if code != 0 || !strings.Contains(out, "--check compares nothing, a file there that differs stops the real run") {
		t.Fatalf("--check: code %d\n%s\n%s", code, out, errOut)
	}
	w.sameAs(before)

	code, _, errOut = w.run()
	w.wantAbort(code, errOut, "a file of the wiki differs from the one at "+w.project+"/docs/wiki")
	for _, want := range []string{"differs: ./index.md\n", "differs: ./_schema.md\n"} {
		if !strings.Contains(errOut, want) {
			t.Errorf("stderr lacks %q:\n%s", want, errOut)
		}
	}
	w.sameAs(before)
	if head := w.git(w.vault, "rev-parse", "HEAD"); head != vaultHead {
		t.Errorf("the vault got a commit")
	}
	if calls := w.calls(); calls != nil {
		t.Errorf("loomux ran: %q", calls)
	}
}

func TestApplyLetsTheLaterSourceWin(t *testing.T) {
	for _, c := range []struct {
		name       string
		stateFirst bool
		want       string
	}{{"vault then state", false, "state\n"}, {"state then vault", true, "vault\n"}} {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			w.write(w.vault+"/91 Projekte/x/log.md", "vault\n")
			w.commitAll(w.vault, "log")
			w.write(w.state+"/log.md", "state\n")
			w.p.WikiSrcs = []string{w.vault + "/91 Projekte/x", w.state}
			if c.stateFirst {
				w.p.WikiSrcs = []string{w.state, w.vault + "/91 Projekte/x"}
			}
			w.full()
			if got := w.read(w.project + "/docs/wiki/log.md"); got != c.want {
				t.Fatalf("log.md %q, want %q", got, c.want)
			}
			if got := w.read(w.project + "/docs/wiki/catalog.json"); got != "{}\n" {
				t.Fatalf("catalog.json %q", got)
			}
		})
	}
}

func TestApplyKeepsTheStateWhenItsOldNameIsTaken(t *testing.T) {
	w := newWorld(t)
	w.write(w.state+".alt/other.txt", "earlier\n")
	out := w.full()
	if !strings.Contains(out, "state: kept, "+w.state+" and "+w.state+".alt both exist\n") {
		t.Errorf("stdout:\n%s", out)
	}
	if w.read(w.state+"/catalog.json") != "{}\n" || exists(w.state+".alt/project-x") {
		t.Error("the state directory was moved")
	}
	entries, _ := os.ReadDir(w.state + ".alt")
	if len(entries) != 1 {
		t.Errorf("%s holds %d entries", w.state+".alt", len(entries))
	}
}

func TestApplyTakesAnEmptySourceForAMovedWiki(t *testing.T) {
	w := newWorld(t)
	if err := os.MkdirAll(w.base+"/empty/topics", 0o755); err != nil {
		t.Fatal(err)
	}
	w.p.VaultOld = ""
	w.p.WikiSrcs = []string{w.base + "/empty"}
	out := w.full()
	if !strings.Contains(out, "wiki: already moved\n") {
		t.Fatalf("stdout:\n%s", out)
	}
}

func TestApplyWithoutASourceLeftNeedsTheWikiInPlace(t *testing.T) {
	w := newWorld(t)
	w.p.VaultOld = ""
	w.p.WikiSrcs = []string{w.base + "/gone"}
	w.p.WikiDst = w.project + "/wiki"
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, "no source of the wiki is left and "+w.project+"/wiki does not exist")
	w.sameAs(before)
}

func TestApplyStopsWhenInitFails(t *testing.T) {
	w := newWorld(t)
	w.setLoomux("[ \"$1\" = init ] && exit 7\nexit 0\n")
	code, out, errOut := w.run()
	if code == 0 {
		t.Fatalf("code 0:\n%s", out)
	}
	if calls := w.calls(); len(calls) != 1 || !strings.HasPrefix(calls[0], "init ") {
		t.Errorf("calls %q", calls)
	}
	for _, still := range []string{w.vault + "/91 Projekte/x/index.md", w.project + "/.ultraloom", w.state} {
		if !exists(still) {
			t.Errorf("%s is gone after init failed (%s)", still, errOut)
		}
	}
	if exists(w.project + "/docs/wiki.staging") {
		t.Error("the staging directory was left behind")
	}
}

func TestApplyHandsTheModuleChoiceToInit(t *testing.T) {
	w := newWorld(t)
	w.p = Params{Name: "x", Project: w.project, Loomux: w.loomux, ConfigNew: w.configNew,
		InitArgs: []string{"--brain=none", "--graph=all"}}
	w.full()
	want := []string{"init --yes --brain=none --graph=all --root " + w.project}
	if calls := w.calls(); !reflect.DeepEqual(calls, want) {
		t.Errorf("calls %q, want %q", calls, want)
	}
}

func TestApplyWithOnlyTheRequiredParts(t *testing.T) {
	w := newWorld(t)
	w.p = Params{Name: "x", Project: w.project, Loomux: w.loomux, ConfigNew: w.configNew}
	before := w.snapshot()
	out := w.full()
	for _, want := range []string{
		"registry: kept, this switch-over leaves it alone",
		"wiki: kept, this switch-over moves none",
		"config: written",
		"hooks: kept, no old hook is named",
		"files: kept, no old file is named",
		"vault: kept, this switch-over leaves it alone",
		"state: kept, none is named",
	} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("stdout lacks %q:\n%s", want, out)
		}
	}
	if calls := w.calls(); len(calls) != 1 {
		t.Errorf("calls %q", calls)
	}
	after := w.snapshot()
	delete(after, "/proj's/.loomux")
	delete(after, "/proj's/.loomux/config.toml")
	delete(after, "/proj's/.loomux/armed.toml")
	if !reflect.DeepEqual(after, before) {
		t.Error("more than the configuration and the armed lanes changed")
	}
}

func TestApplyTakesTheNameOfAFolderLiterally(t *testing.T) {
	w := newWorld(t)
	// x[1] is a glob that matches x1: neither the shell nor git may read it
	// as one, or x1 would be merged and removed in its place. A git glob
	// matches a whole path, so x1 is a file.
	w.write(w.vault+"/91 Projekte/x[1]/own.md", "own\n")
	w.write(w.vault+"/91 Projekte/x1", "keep\n")
	w.commitAll(w.vault, "two folders")
	w.p.VaultOld = "91 Projekte/x[1]"
	w.p.WikiSrcs = []string{w.vault + "/91 Projekte/x[1]"}
	w.full()
	if w.read(w.project+"/docs/wiki/own.md") != "own\n" {
		t.Error("the wiki was merged from the wrong folder")
	}
	if files := w.git(w.vault, "ls-files", "91 Projekte"); files != "91 Projekte/x/_schema.md\n91 Projekte/x/audit.md\n91 Projekte/x/index.md\n91 Projekte/x1" {
		t.Errorf("the vault keeps\n%s", files)
	}
	w.commitAll(w.project, "switch over")
	code, out, errOut := w.run()
	if code != 0 || !strings.Contains(out, "vault: already removed 91 Projekte/x[1]\n") {
		t.Fatalf("second run: code %d\n%s\n%s", code, out, errOut)
	}
}

func TestApplyStopsWhenTheCopyFails(t *testing.T) {
	w := newWorld(t)
	// A file stands where the copy needs a directory. It sorts before the
	// files that copy fine, so the loop does not end on the failure.
	w.write(w.vault+"/91 Projekte/x/a-sub/deep.md", "deep\n")
	w.commitAll(w.vault, "deep")
	w.write(w.project+"/docs/wiki/a-sub", "a file\n")
	w.commitAll(w.project, "sub")
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, "copying the wiki to "+w.project+"/docs/wiki failed")
	if !exists(w.vault+"/91 Projekte/x/index.md") || exists(w.project+"/docs/wiki.staging") {
		t.Error("the vault folder went or the staging directory stayed")
	}
}

func TestApplyStopsWhenStagingTheVaultFails(t *testing.T) {
	w := newWorld(t)
	// An earlier source leaves a file where the vault's files need a
	// directory; a-sub sorts before the files that stage fine.
	w.write(w.state+"/a-sub", "a file\n")
	w.write(w.vault+"/91 Projekte/x/a-sub/deep.md", "deep\n")
	w.commitAll(w.vault, "deep")
	w.p.WikiSrcs = []string{w.state, w.vault + "/91 Projekte/x"}
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, "staging the files of 91 Projekte/x failed")
	w.sameAs(before)
}

func TestApplyStopsWhenStagingAMiddleSourceFails(t *testing.T) {
	w := newWorld(t)
	// The vault stages a file a-sub; the middle source needs a directory
	// there, and the last one would have staged fine.
	w.write(w.vault+"/91 Projekte/x/a-sub", "a file\n")
	w.commitAll(w.vault, "a-sub")
	w.write(w.base+"/extra/a-sub/deep.md", "deep\n")
	w.p.WikiSrcs = []string{w.vault + "/91 Projekte/x", w.base + "/extra", w.state}
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, "staging "+w.base+"/extra failed")
	w.sameAs(before)
}

func TestApplyRefusesAVaultFolderSpeltOtherwiseThanGitHasIt(t *testing.T) {
	w := newWorld(t)
	// On a disk that ignores case the folder is found, but git lists
	// nothing under this spelling: the wiki would pass as moved.
	w.p.VaultOld = "91 projekte/X"
	w.p.WikiSrcs = []string{w.vault + "/91 projekte/X"}
	before := w.snapshot()
	for _, args := range [][]string{nil, {"--check"}} {
		code, _, errOut := w.run(args...)
		w.wantAbort(code, errOut, "git holds 91 projekte/X of "+w.vault+" in another spelling")
	}
	w.sameAs(before)
}

func TestApplyTakesFileNamesAsTheyAre(t *testing.T) {
	w := newWorld(t)
	// git quotes an umlaut unless told not to; a[b].md is a glob that
	// matches ab.md, which the project holds with other text.
	w.write(w.vault+"/91 Projekte/x/Übersicht.md", "ü\n")
	w.write(w.vault+"/91 Projekte/x/a[b].md", "bracket\n")
	w.commitAll(w.vault, "odd names")
	w.write(w.project+"/docs/wiki/Übersicht.md", "ü\n")
	w.write(w.project+"/docs/wiki/ab.md", "plain\n")
	w.commitAll(w.project, "odd names")
	old := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes(w.project+"/docs/wiki/Übersicht.md", old, old); err != nil {
		t.Fatal(err)
	}
	w.full()
	if info, err := os.Stat(w.project + "/docs/wiki/Übersicht.md"); err != nil || !info.ModTime().Equal(old) {
		t.Errorf("Übersicht.md was not taken for the same file: %v", err)
	}
	if w.read(w.project+"/docs/wiki/a[b].md") != "bracket\n" || w.read(w.project+"/docs/wiki/ab.md") != "plain\n" {
		t.Error("a[b].md was read as a glob")
	}
	code, out, errOut := w.secondRun()
	if code != 0 || !strings.Contains(out, "wiki: already moved\n") {
		t.Fatalf("second run: code %d\n%s\n%s", code, out, errOut)
	}
}

func TestApplyStopsWhenTheStagedFilesCannotBeListed(t *testing.T) {
	w := newWorld(t)
	// Without the PATH line the stand-in find, which lists nothing, is the
	// one the script runs, as the find of Windows would be.
	guard := "PATH=/usr/bin:$PATH\n"
	text, err := render(templateUnderTest(t), w.p)
	if err != nil || !strings.Contains(text, guard) {
		t.Fatalf("no PATH line: %v", err)
	}
	w.write(w.script, strings.Replace(text, guard, "", 1))
	before := w.snapshot()
	cmd := exec.Command(w.sh, w.script)
	cmd.Env = w.env
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err = cmd.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("the script ended with %v", err)
	}
	w.wantAbort(exit.ExitCode(), stderr.String(), "the sources of the wiki hold no file")
	w.sameAs(before)
}

func TestApplyReadsARegistryPathWithBackslashes(t *testing.T) {
	w := newWorld(t)
	// sha256sum puts a backslash before the sum of such a name.
	odd := w.base + `/state/reg\istry.toml`
	if runtime.GOOS == "windows" {
		odd = filepath.FromSlash(w.registry)
	}
	w.write(odd, "old registry\n")
	w.p.Registry = odd
	w.full()
	if w.read(odd) != "new registry\n" {
		t.Errorf("registry %q", w.read(odd))
	}
}

// secondRun commits what the first run left in the project and runs again.
func (w *world) secondRun() (int, string, string) {
	w.t.Helper()
	w.commitAll(w.project, "switch over")
	return w.run()
}

func TestApplyASecondRunPassesTheEmptyFoldersTheVaultKeeps(t *testing.T) {
	w := newWorld(t)
	// git tracks no directory: these stay after git rm, as in the vault.
	for _, d := range []string{"entities", "sources", "syntheses", "topics"} {
		if err := os.MkdirAll(w.vault+"/91 Projekte/x/"+d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	w.full()
	code, out, errOut := w.secondRun()
	if code != 0 {
		t.Fatalf("second run: code %d\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{"wiki: already moved\n", "vault: already removed 91 Projekte/x\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout lacks %q:\n%s", want, out)
		}
	}
}

func TestApplyWithOnlyEmptyFoldersLeftNeedsTheWikiInPlace(t *testing.T) {
	w := newWorld(t)
	w.git(w.vault, "rm", "-r", "-q", "91 Projekte/x")
	w.git(w.vault, "commit", "-q", "-m", "gone")
	if err := os.MkdirAll(w.vault+"/91 Projekte/x/topics", 0o755); err != nil {
		t.Fatal(err)
	}
	w.p.WikiDst = w.project + "/wiki"
	before := w.snapshot()
	code, _, errOut := w.run()
	w.wantAbort(code, errOut, "no source of the wiki is left and "+w.project+"/wiki does not exist")
	w.sameAs(before)
}

// ignoredInVault gives the vault a .gitignore for .brain/ at every depth and
// an ignored file in the project's folder.
func (w *world) ignoredInVault() {
	w.t.Helper()
	w.write(w.vault+"/.gitignore", ".brain/\n")
	w.write(w.vault+"/91 Projekte/x/.brain/cache.db", "cache v1\n")
	w.commitAll(w.vault, "ignore the cache")
}

func TestApplyCopiesNoIgnoredFileOfTheVault(t *testing.T) {
	w := newWorld(t)
	w.ignoredInVault()
	w.full()
	if exists(w.project + "/docs/wiki/.brain") {
		t.Fatal("an ignored file of the vault was copied into the project")
	}
	w.write(w.vault+"/91 Projekte/x/.brain/cache.db", "cache v2\n")
	before := w.snapshot()
	code, out, errOut := w.secondRun()
	if code != 0 || !strings.Contains(out, "wiki: already moved\n") {
		t.Fatalf("second run: code %d\n%s\n%s", code, out, errOut)
	}
	w.sameAs(before)
}

func TestApplyASecondRunWithAnIgnoredFileReachesTheEnd(t *testing.T) {
	w := newWorld(t)
	w.ignoredInVault()
	w.full()
	code, out, errOut := w.secondRun()
	if code != 0 || !strings.Contains(out, "vault: already removed 91 Projekte/x\n") || !strings.Contains(out, "done: ") {
		t.Fatalf("second run: code %d\n%s\n%s", code, out, errOut)
	}
}

func TestApplyRefusesALoomuxWithoutTheSwitchover(t *testing.T) {
	for _, args := range [][]string{nil, {"--check"}} {
		w := newWorld(t)
		w.setOldLoomux()
		before := w.snapshot()
		code, _, errOut := w.run(args...)
		w.wantAbort(code, errOut, w.loomux+" does not know dev switchover prune-hooks; build it first")
		w.sameAs(before)
		for _, c := range w.calls() {
			if c != "dev switchover" {
				t.Errorf("%q: loomux was called with %q", args, c)
			}
		}
	}
	// Without old hooks the script never calls the pruner and asks nothing.
	w := newWorld(t)
	w.setOldLoomux()
	w.p.OldHooks = nil
	w.full()
}

func TestApplyReplacesTheRegistryInOneStep(t *testing.T) {
	w := newWorld(t)
	before, err := os.Stat(w.registry)
	if err != nil {
		t.Fatal(err)
	}
	// On Windows the file's identity is read on first use, by its path: read
	// it now, while the path still names the old file.
	os.SameFile(before, before)
	w.full()
	after, err := os.Stat(w.registry)
	if err != nil {
		t.Fatal(err)
	}
	// A new file took the name: a reader sees the old or the new one whole.
	if os.SameFile(before, after) {
		t.Error("the registry was written in place")
	}
	entries, _ := os.ReadDir(w.base + "/state")
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "registry.toml.new") {
			t.Errorf("%s was left behind", e.Name())
		}
	}
}

func TestApplyKeepsAnEarlierBackupOfTheRegistry(t *testing.T) {
	w := newWorld(t)
	w.write(w.registry+".bak", "the first registry\n")
	out := w.full()
	if got := w.read(w.registry + ".bak"); got != "the first registry\n" {
		t.Errorf("backup %q", got)
	}
	if w.read(w.registry) != "new registry\n" || !strings.Contains(out, "registry: backup kept, "+w.registry+".bak exists\n") {
		t.Errorf("registry %q\n%s", w.read(w.registry), out)
	}
}

func TestApplyCheckWithoutASettingsFileNamesThePruner(t *testing.T) {
	w := newWorld(t)
	if err := os.Remove(w.project + "/.claude/settings.json"); err != nil {
		t.Fatal(err)
	}
	w.commitAll(w.project, "no settings")
	// init may write the file, so --check cannot know it stays missing.
	code, out, errOut := w.run("--check")
	if code != 0 || !strings.Contains(out, "would run: "+w.loomux+" dev switchover prune-hooks") {
		t.Fatalf("code %d\n%s\n%s", code, out, errOut)
	}
}

func TestApplyCleansUpWhenItIsStopped(t *testing.T) {
	w := newWorld(t)
	// One large file of the last source: staging it takes long enough that
	// TERM comes while the copy runs, and the script ends once it is done.
	w.write(w.state+"/big.bin", "")
	if err := os.Truncate(w.state+"/big.bin", 512<<20); err != nil {
		t.Fatal(err)
	}
	w.p.WikiSrcs = []string{w.vault + "/91 Projekte/x", w.state}
	text, err := render(templateUnderTest(t), w.p)
	if err != nil {
		t.Fatal(err)
	}
	w.write(w.script, text)
	driver := w.base + "/prep/driver.sh"
	w.write(driver, "sh \"$SCRIPT\" >/dev/null 2>&1 &\npid=$!\n"+
		// A script that ends before it stages would leave the loop waiting
		// for ever; it ends with the script, and the exit code tells.
		"while [ ! -e \"$STAGING/big.bin\" ] && kill -0 $pid 2>/dev/null; do :; done\n"+
		"kill -TERM $pid\nwait $pid\necho \"exit=$?\"\n")
	cmd := exec.Command(w.sh, driver)
	cmd.Env = append(append([]string{}, w.env...), "SCRIPT="+w.script, "STAGING="+w.project+"/docs/wiki.staging")
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "exit=130") {
		t.Fatalf("driver: %v %s", err, out)
	}
	if exists(w.project + "/docs/wiki.staging") {
		t.Error("the staging directory was left behind")
	}
	if w.read(w.registry) != "old registry\n" {
		t.Error("the registry was replaced")
	}
}

// A run that ends before the configuration is in place has started the
// probation already: a configuration left without the armed lanes would be
// kept by the next run, and the project would never get them.
func TestApplyWritesTheArmedLanesBeforeTheConfiguration(t *testing.T) {
	w := newWorld(t)
	// The configuration lies in the staging directory, which git ignores: it
	// is there for the checks of step 1 and gone once the staging begins, so
	// that the copy of step 4 fails.
	w.write(w.project+"/.gitignore", "docs/wiki.staging/\n")
	w.commitAll(w.project, "ignore the staging")
	w.p.ConfigNew = w.project + "/docs/wiki.staging/config.toml.new"
	w.write(w.p.ConfigNew, "[area]\nname = \"x\"\n")
	code, out, errOut := w.run()
	// The copy names its source when it fails; the wiki is merged by then.
	if code == 0 || !strings.Contains(errOut, "config.toml.new") || !strings.Contains(out, "wiki: merged") {
		t.Fatalf("code %d, want the copy of the configuration to fail:\n%s\n%s", code, out, errOut)
	}
	if exists(w.project + "/.loomux/config.toml") {
		t.Error("config.toml was written")
	}
	if got := w.read(w.project + "/.loomux/armed.toml"); got != (verify.ArmedSet{Exists: true}).Text() {
		t.Errorf("armed.toml %q, want the text init writes", got)
	}
}

func TestApplyPrunesNoHooksWithoutASettingsFile(t *testing.T) {
	w := newWorld(t)
	if err := os.Remove(w.project + "/.claude/settings.json"); err != nil {
		t.Fatal(err)
	}
	w.commitAll(w.project, "no settings")
	out := w.full()
	if !strings.Contains(out, "hooks: kept, "+w.project+"/.claude/settings.json does not exist\n") {
		t.Errorf("stdout:\n%s", out)
	}
	if calls := w.calls(); len(calls) != 1 {
		t.Errorf("calls %q", calls)
	}
}
