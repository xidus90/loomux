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
// WikiPath is the one optional value: an entry without `wiki` registers an
// area with WikiPath "", and an entry with an empty `wiki` is refused, so ""
// always means "names no wiki". A pointer would push that distinction into
// every caller for the one field that has it.
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

// ReadRegistry reads the areas registered in stateDir and refuses a registry
// it cannot use whole. The write barrier reads the same file through this
// function, so a registry either works for both or for neither.
//
// A broken entry refuses the call instead of being skipped: two entries of
// one scope or one state directory leave nothing that could safely be
// dropped, and a skipped entry would show brain a different registry than
// the barrier sees.
func ReadRegistry(stateDir string) ([]Area, error) {
	path := filepath.Join(stateDir, registryName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	document := map[string]any{}
	if err := toml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("%s: not valid TOML: %w", path, err)
	}
	entries, err := areaEntries(document)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	reading := registryReading{
		stateDir:    stateDir,
		positions:   map[string]int{},
		directories: map[string]string{},
	}
	areas := make([]Area, 0, len(entries))
	for i, entry := range entries {
		area, err := reading.area(i+1, entry)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		areas = append(areas, area)
	}
	return areas, nil
}

// areaEntries is the `area` array. The decoder answers []map[string]any
// when every element is a table and []any otherwise.
func areaEntries(document map[string]any) ([]any, error) {
	switch value := document["area"].(type) {
	case nil:
		return nil, nil
	case []map[string]any:
		entries := make([]any, len(value))
		for i, entry := range value {
			entries[i] = entry
		}
		return entries, nil
	case []any:
		return value, nil
	default:
		return nil, fmt.Errorf("area must be an array of [[area]] tables, found %s", tomlType(value))
	}
}

// registryReading carries what one entry's rules need to know about the
// entries before it.
type registryReading struct {
	stateDir    string
	positions   map[string]int
	directories map[string]string
	signposted  string
}

// area checks one entry in the order of the spec's rules G4 to G14.
func (r *registryReading) area(position int, raw any) (Area, error) {
	entry, ok := raw.(map[string]any)
	if !ok {
		return Area{}, fmt.Errorf("[[area]] #%d must be a table, found %s", position, tomlType(raw))
	}
	numbered := fmt.Sprintf("[[area]] #%d", position)
	scope, err := requiredString(entry, "scope", numbered, ": ")
	if err != nil {
		return Area{}, err
	}
	if first, taken := r.positions[scope]; taken {
		return Area{}, fmt.Errorf("%s: duplicate scope %q (first at #%d)", numbered, scope, first)
	}
	r.positions[scope] = position
	named := fmt.Sprintf("[[area]] %q", scope)
	directory := stateDirOf(r.stateDir, scope)
	if directory == stateDirOf(r.stateDir, "") {
		return Area{}, fmt.Errorf(`%s: scope has no letter, digit, "_", "." or "-" and cannot name a state directory`, named)
	}
	if other, taken := r.directories[directory]; taken {
		return Area{}, fmt.Errorf("scopes %q and %q share the state directory %q", other, scope, filepath.Base(directory))
	}
	r.directories[directory] = scope
	path, err := requiredString(entry, "path", named, ": ")
	if err != nil {
		return Area{}, err
	}
	wiki, err := optionalString(entry, "wiki", named, ": ")
	if err != nil {
		return Area{}, err
	}
	flags := map[string]bool{}
	for _, key := range []string{"readonly", "signpost", "shared", "workspace"} {
		flag, err := optionalBool(entry, key, named, ": ")
		if err != nil {
			return Area{}, err
		}
		flags[key] = flag
	}
	if flags["signpost"] {
		if r.signposted != "" {
			return Area{}, fmt.Errorf("scopes %q and %q both declare signpost; only one area may", r.signposted, scope)
		}
		r.signposted = scope
	}
	return Area{
		Scope:    scope,
		Path:     path,
		WikiPath: wiki,
		ReadOnly: flags["readonly"],
		Signpost: flags["signpost"],
		Shared:   flags["shared"],

		Workspace: flags["workspace"],
	}, nil
}

// stateDirOf is the directory a read-only area of this scope keeps its
// artefacts in; ManifestDir holds the rule that names it.
func stateDirOf(stateDir, scope string) string {
	return ManifestDir(Area{Scope: scope, ReadOnly: true}, stateDir)
}

// StateDirEnv is the variable `paths._ENV` names. It is exported because a
// spawned child inherits its state directory through it and nowhere else.
const StateDirEnv = "LOOMUX_STATE_DIR"

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
	if fromEnv := os.Getenv(StateDirEnv); fromEnv != "" {
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
// from unset, so this function answers home/.local/state/loomux for both --
// loomux, not brain: the directory is named after this binary everywhere
// below, and only the Python side ever wrote 'brain'.
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
