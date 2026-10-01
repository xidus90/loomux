// Package setup plans and applies what `loomux init` does to a project:
// it gathers what the project already is, lists the parts each module brings
// and builds the plan of file changes and actions a human approves.
package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/setup/gitfiles"
	"github.com/xidus90/loomux/internal/setup/hostfile"
	"github.com/xidus90/loomux/internal/verify"
)

// configPath is the project configuration, relative to the root.
const configPath = ".loomux/config.toml"

// armedPath holds the armed lanes, relative to the root; init plans it
// empty for a project loomux was not set up in before.
const armedPath = verify.ArmedFile

// checkoutModule is the module path that makes a project a checkout of
// loomux itself.
const checkoutModule = "github.com/xidus90/loomux"

// Facts is what a project already is before init changes anything.
type Facts struct {
	Root      string
	Detect    detect.Facts
	Hosts     []hosts.Host // .claude/ -> claude; .agents/hooks.json, .agents/skills/ or GEMINI.md -> antigravity; neither -> claude
	HooksPath string       // core.hooksPath, "" when unset
	// GitHooksDir is git's own hook directory, absolute, as
	// `git rev-parse --git-path hooks` names it; read only in a repository
	// without core.hooksPath. In a linked worktree or a submodule it lies
	// outside Root and is shared with other checkouts.
	GitHooksDir string
	// GitHooksLive says whether GitHooksDir holds a pre-commit, pre-push or
	// commit-msg; the *.sample files git puts there are no hooks.
	GitHooksLive bool
	// HookPreCommit is the pre-commit hook in the directory git runs hooks
	// from now, "" when there is none. The plan reads a hook inside the
	// project itself; this one is for a directory outside it -- an absolute
	// core.hooksPath, the common hook directory of a linked worktree.
	HookPreCommit string
	Config        string // .loomux/config.toml, "" when missing
	UserMCP       bool   // ~/.claude.json names an mcpServers entry "loomux"
	Registered    bool   // a registry area's Path is Root (config.ReadRegistry(config.StateDir()))
	Checkout      bool   // go.mod declares github.com/xidus90/loomux
	Binary        string // hostfile.Canonical or hostfile.Checkout
	// BinaryThere says whether the binary Binary names stands where
	// BinaryPath puts it. Keeping an installed one current is serve's and
	// upgrade's work, not init's.
	BinaryThere bool
	// CanonicalThere says whether the installed binary stands under
	// LOCALAPPDATA, the one the post-merge hook calls in every project.
	CanonicalThere bool
	// MergeHook says whether the hook directory git uses now holds a
	// post-merge hook of ours.
	MergeHook bool
	// OnMerge says whether the declaration at Root consents to the merge
	// hook with [maintenance] on_merge = true, read as merge-hook reads it.
	OnMerge bool
	// Graph says whether the code graph has been built.
	Graph bool
	// LocalAppDataSpaced says whether LOCALAPPDATA holds whitespace or a
	// character cmd.exe reads as whitespace or syntax. Antigravity's entries name the
	// installed binary as an unquoted %LOCALAPPDATA% path, since agy breaks a
	// quoted one, and cmd.exe would split that path there.
	LocalAppDataSpaced bool
	// Version is the running init's own version, cli.Version.
	Version string
	// Installed is the version the installed binary names with --version,
	// "" when it is not there or names none. Antigravity's entries call it,
	// and an older one does not know them.
	Installed string
	// Model is the global [model] block of the state directory's
	// config.toml, the defaults where it declares none.
	Model config.ModelSettings
	// ModelProblem is why that file does not read, "" when it does. It stops
	// no init; the part model names it.
	ModelProblem string
}

// Running is what Gather is told about the init that runs it.
type Running struct {
	// Version is cli.Version; setup may not import cli.
	Version string
	// VersionOf is what the binary at path names with --version, or "".
	VersionOf func(path string) string
}

// HookWanted says whether the merge hook has an area to serve without area
// add: one registered at Root whose declaration consents. A declaration a
// clone brought along, with no area of this machine's registry behind it,
// gives the hook nobody to record for.
func (f Facts) HookWanted() bool { return f.OnMerge && f.Registered }

// BinaryPath is the file an entry calling binary runs: the checkout's
// <root>/bin/loomux.exe, or the installed one under LOCALAPPDATA, "" when
// that is not set.
func BinaryPath(root, binary string) string {
	if binary == hostfile.Checkout {
		return filepath.Join(root, "bin", "loomux.exe")
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		return filepath.Join(local, "loomux", "bin", "loomux.exe")
	}
	return ""
}

// isFile says whether path names a file; "" names none.
func isFile(path string) bool {
	info, err := os.Stat(path)
	return path != "" && err == nil && !info.IsDir()
}

// cmdSplits says whether cmd.exe would not take path, expanded into an
// unquoted command line, as one word: whitespace splits it, so do , ; and =,
// which it reads as whitespace, and & | < > ^ ( ) and a quote are its syntax.
func cmdSplits(path string) bool {
	return strings.ContainsFunc(path, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(`,;=&|<>^()"`, r)
	})
}

// isDir says whether path names a directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Gather reads the facts of the project at root; home is the user's home
// directory, running describes the init that asks, and git runs git where
// the facts need it.
func Gather(root, home string, running Running, git detect.Runner) (Facts, error) {
	f := Facts{Root: root, Detect: detect.Detect(os.DirFS(root)), Version: running.Version}
	if isDir(filepath.Join(root, ".claude")) {
		f.Hosts = append(f.Hosts, hosts.HostClaude)
	}
	// .agents/ alone is a name other tools use too; Antigravity is taken
	// only from its hook file, its skills or its GEMINI.md.
	if isFile(filepath.Join(root, ".agents", "hooks.json")) || isDir(filepath.Join(root, ".agents", "skills")) ||
		isFile(filepath.Join(root, "GEMINI.md")) {
		f.Hosts = append(f.Hosts, hosts.HostAntigravity)
	}
	if len(f.Hosts) == 0 {
		f.Hosts = []hosts.Host{hosts.HostClaude}
	}
	var err error
	if f.HooksPath, err = detect.HooksPath(git, root); err != nil {
		return Facts{}, fmt.Errorf("read core.hooksPath: %w", err)
	}
	// Where git runs hooks today, for the merge hook: git's answer, not
	// core.hooksPath read by hand.
	var hooksNow string
	if f.Detect.HasGit {
		if hooksNow, err = gitHookDir(git, root); err != nil {
			return Facts{}, err
		}
	}
	if f.Detect.HasGit && f.HooksPath == "" {
		if err := f.readGitHooks(hooksNow); err != nil {
			return Facts{}, err
		}
	}
	cfg, _, err := readOptional(root, configPath)
	if err != nil {
		return Facts{}, unmergeable{err}
	}
	f.Config = string(cfg)
	gomod, _, err := readOptional(root, "go.mod")
	if err != nil {
		return Facts{}, err
	}
	f.Checkout = declaresModule(string(gomod), checkoutModule)
	settings, _, err := readOptional(root, hostfile.Path(hosts.HostClaude))
	if err != nil {
		return Facts{}, unmergeable{err}
	}
	f.Binary = hostfile.BinaryOf(hosts.HostClaude, settings)
	if f.Checkout {
		f.Binary = hostfile.Checkout
	}
	f.BinaryThere = isFile(BinaryPath(root, f.Binary))
	f.CanonicalThere = isFile(BinaryPath(root, hostfile.Canonical))
	if f.CanonicalThere {
		f.Installed = running.VersionOf(BinaryPath(root, hostfile.Canonical))
	}
	f.Graph = isFile(store.WiringPath(root))
	f.LocalAppDataSpaced = cmdSplits(os.Getenv("LOCALAPPDATA"))
	if hooksNow != "" {
		data, err := os.ReadFile(filepath.Join(hooksNow, "post-merge"))
		f.MergeHook = err == nil && maintenance.OwnsHook(data)
		// A hook that does not read is none: the project then starts in
		// probation, which arms nothing it had armed before.
		preCommit, _ := os.ReadFile(filepath.Join(hooksNow, "pre-commit"))
		f.HookPreCommit = string(preCommit)
	}
	// A declaration that does not read consents to nothing; Build refuses
	// the configuration on its own when the part config is on.
	declared, err := config.ReadAreaManifestUntilStage4(root)
	f.OnMerge = err == nil && declared.OnMerge
	if f.Model, err = config.ReadModelSettings(config.StateDir()); err != nil {
		f.ModelProblem = err.Error()
	}
	f.UserMCP = userMCP(filepath.Join(home, ".claude.json"))
	if f.Registered, err = registered(root); err != nil {
		return Facts{}, err
	}
	return f, nil
}

// gitHookDir asks git which directory it runs hooks from in root: the one
// core.hooksPath names, expanded as git expands it, or its own.
func gitHookDir(git detect.Runner, root string) (string, error) {
	out, err := git(root, "git", "rev-parse", "--path-format=absolute", "--git-path", "hooks")
	if err != nil {
		return "", fmt.Errorf("read git's hook directory: %w", err)
	}
	dir := strings.TrimSpace(out)
	if dir == "" {
		return "", errors.New("read git's hook directory: git named none")
	}
	// Asked for an absolute path; a git too old for --path-format answers
	// relative to the directory it ran in.
	if dir = filepath.FromSlash(dir); !filepath.IsAbs(dir) {
		dir = filepath.Join(root, dir)
	}
	return filepath.Clean(dir), nil
}

// readGitHooks looks for live hooks in dir, git's own hook directory. A
// hook that cannot be looked at is an error, not an absent hook: guessing
// wrong would let init switch it off.
func (f *Facts) readGitHooks(dir string) error {
	f.GitHooksDir = dir
	for name := range gitfiles.Hooks("") {
		info, err := os.Stat(filepath.Join(f.GitHooksDir, name))
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return fmt.Errorf("git hook %s: %w", name, err)
		case !info.IsDir():
			f.GitHooksLive = true
		}
	}
	return nil
}

// unmergeable is the error of a file init merges into and cannot read.
type unmergeable struct{ error }

// Unmergeable says whether Gather stopped at a file init merges into -- the
// configuration or a host file -- rather than at one it only looks at.
func Unmergeable(err error) bool {
	var u unmergeable
	return errors.As(err, &u)
}

// readOptional reads rel under root; a missing file is no error.
func readOptional(root, rel string) ([]byte, bool, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", rel, err)
	}
	return data, true, nil
}

// declaresModule says whether the module line of gomod names module exactly.
func declaresModule(gomod, module string) bool {
	for _, line := range strings.Split(gomod, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" && strings.Trim(fields[1], `"`) == module {
			return true
		}
	}
	return false
}

// userMCP says whether the user scope of Claude Code already names a server
// loomux. A file that is missing or does not parse names none: it belongs
// to the user, and init only reads it.
func userMCP(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var doc struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return false
	}
	_, ok := doc.MCPServers["loomux"]
	return ok
}

// registered says whether an area of the registry stands at root. A machine
// without a registry has none; a registry that does not read stops the run,
// because the write barrier reads the same file.
func registered(root string) (bool, error) {
	areas, err := config.ReadRegistry(config.StateDir())
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, a := range areas {
		// By identity, as the merge hook's filter compares: the registry
		// may name the root through another spelling -- a junction, a
		// symlink, an 8.3 name. A path that is gone only has its spelling.
		match := samePath(a.Path, root)
		if _, err := os.Stat(a.Path); err == nil {
			match = SameDir(root, a.Path)
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}

// SameDir says whether path names the directory root, compared by identity
// as within compares, not by spelling.
func SameDir(root, path string) bool {
	rel, inside := within(root, path)
	return inside && rel == "."
}

// samePath compares two paths the way Windows does: cleaned, in any case.
func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(filepath.FromSlash(a)), filepath.Clean(filepath.FromSlash(b)))
}

// hasArea says whether text declares an [area] table. Text that does not
// parse declares none; Build refuses it on its own.
func hasArea(text string) bool {
	doc := map[string]any{}
	if toml.Unmarshal([]byte(text), &doc) != nil {
		return false
	}
	_, ok := doc["area"]
	return ok
}
