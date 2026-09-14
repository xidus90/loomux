package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

// registryName is the file `paths.registry_path` names: the state directory
// plus this one name, with no search and no fallback.
const registryName = "registry.toml"

// Area is one registered area as the registry declares it.
//
// The paths are strings, not a path type, and they are handed on exactly as
// the file spells them. The registry on this machine writes Windows drive
// letters with forward slashes ("C:/Users/micro/Documents/#GIT/brain-knowledge"),
// so a reader that cleaned or converted them would answer a path the file
// never contained; whoever joins these onto something else owes the joining.
//
// WikiPath is the one optional value. `registry.py` builds it as
// `Path(wiki) if wiki else None`; the empty string is this reader's None,
// because Go has no absent string and a pointer would push the distinction
// into every caller for the one field that has it. Of the nine areas in the
// real registry, none omits `wiki` today -- but `registry.py` allows it, so
// this reader does too.
type Area struct {
	Scope    string
	Path     string
	WikiPath string
	ReadOnly bool
	Signpost bool
	Shared   bool

	// Workspace is independent of ReadOnly, not derived from it: in the
	// registry on this machine `project/space` sets both, `project/ecoflow`
	// only workspace, and `project/iam-wiki` only readonly.
	//
	// No rule of either check axis reads it, and that is deliberate rather
	// than an oversight: this reader's job is to carry what the file says,
	// and `workspace` is a live key of nine entries that the write barrier
	// and the commands after this slice turn on. Dropping it here would make
	// the reader answer less than the file states -- the one thing a reader
	// must not do.
	Workspace bool
}

// registryEntry is the wire shape of one `[[area]]` table. It is separate from
// Area because the two are not the same thing: the file spells the wiki path
// `wiki`, and an entry may be unusable while an Area may not.
type registryEntry struct {
	Scope    string `toml:"scope"`
	Path     string `toml:"path"`
	Wiki     string `toml:"wiki"`
	ReadOnly bool   `toml:"readonly"`
	Signpost bool   `toml:"signpost"`
	Shared   bool   `toml:"shared"`

	Workspace bool `toml:"workspace"`
}

type registryFile struct {
	Area []registryEntry `toml:"area"`
}

// ReadRegistry reads the areas registered in stateDir.
//
// It reads and does not judge. `read_registry` in src/brain/registry.py
// refuses a duplicate scope, two scopes that sanitise to one state directory,
// and a second `signpost`; none of that is repeated here, because the checks
// that need those rules are still to be written and a second place that
// decides them would be a second source of truth.
//
// The one deliberate difference in behaviour, not just in scope: an entry
// without `scope` or `path` is dropped and the rest of the registry is
// answered, where `_required` raises and takes the whole file with it. No
// Python caller softens that -- `index_all` (src/brain/cli.py:236) calls
// `read_registry` first and dies with it. What `index_all` skips one level
// further on (cli.py:238-247) is a different class: an entry that parsed, but
// whose directory or manifest is gone. `read_registry` inspects neither.
// The skip here is therefore new behaviour, asked for by the brief, and not a
// tolerance copied from somewhere.
//
// What this signature cannot do is say which entry was dropped: it returns
// areas and one error, and the error is reserved for the whole file. The
// caller sees a shorter slice and nothing else. Whoever drives this reader
// from a command owes the `index_all` line.
func ReadRegistry(stateDir string) ([]Area, error) {
	path := filepath.Join(stateDir, registryName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	file := registryFile{}
	if err := toml.Unmarshal(data, &file); err != nil {
		// Everything the decoder rejects lands here, and that is wider than
		// `tomllib.TOMLDecodeError` on the Python side. Measured, two inputs
		// that registry.py handles itself end up here instead: `[area]`
		// written as a table gives "TOML value has type map[string]any;
		// destination has type slice" (registry.py has its own message for
		// it), and `readonly = "yes"` fails the whole file, where
		// `bool(entry.get("readonly", False))` reads True.
		return nil, fmt.Errorf("%s: not valid TOML: %w", path, err)
	}
	areas := make([]Area, 0, len(file.Area))
	for _, entry := range file.Area {
		if entry.Scope == "" || entry.Path == "" {
			continue
		}
		areas = append(areas, Area{
			Scope:    entry.Scope,
			Path:     entry.Path,
			WikiPath: entry.Wiki,
			ReadOnly: entry.ReadOnly,
			Signpost: entry.Signpost,
			Shared:   entry.Shared,

			Workspace: entry.Workspace,
		})
	}
	return areas, nil
}

// stateDirEnv is the variable `paths._ENV` names.
const stateDirEnv = "LOOMUX_STATE_DIR"

// StateDir is where the registry lives when nobody says otherwise.
//
// `resolve_state_dir` (src/brain/paths.py) knows three steps -- an explicit
// location, then the environment, then the platform. Only the last two are
// here: the explicit step is the caller's `--state-dir`, and a Go function
// that took an argument only to hand it straight back would put the same
// decision in two places.
//
// The home directory error is dropped rather than handled: the value feeds
// defaultStateDir, which uses it only where the platform variable is missing,
// and an empty home there yields a relative path rather than a wrong one.
func StateDir() string {
	if fromEnv := os.Getenv(stateDirEnv); fromEnv != "" {
		return fromEnv
	}
	home, _ := os.UserHomeDir()
	return defaultStateDir(runtime.GOOS, os.Getenv, home)
}

// defaultStateDir is `_platform_default` with its three inputs made
// arguments. Taking goos, the lookup and the home directory rather than
// reading them keeps both branches reachable from a test on either operating
// system; a build tag would compile one of them away instead, and coverage
// would then report full on half the code.
//
// On this machine both sides resolve the home the same way: os.UserHomeDir
// and Path.home() both answered 'C:\Users\micro', with USERPROFILE and HOME
// set to it. Go reads USERPROFILE alone, where ntpath.expanduser falls back to
// HOMEDRIVE and HOMEPATH -- untested here, and no registry depends on it.
//
// One measured difference to `_platform_default`: an empty XDG_STATE_HOME
// yields the relative path 'brain' there, because `os.environ.get(key,
// default)` returns the empty string it finds. os.Getenv cannot tell empty
// from unset, so this function answers home/.local/state/brain for both.
func defaultStateDir(goos string, getenv func(string) string, home string) string {
	if goos == "windows" {
		if base := getenv("LOCALAPPDATA"); base != "" {
			return filepath.Join(base, "loomux")
		}
		return filepath.Join(home, "AppData", "Local", "loomux")
	}
	if base := getenv("XDG_STATE_HOME"); base != "" {
		return filepath.Join(base, "loomux")
	}
	return filepath.Join(home, ".local", "state", "loomux")
}
