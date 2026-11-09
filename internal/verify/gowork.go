package verify

import (
	"fmt"
	"os"
	"path/filepath"
)

// GoWork is the environment a Go lane runs with: the go tool's work directory
// and build cache under the user's ~/.loomux, for whichever of GOTMPDIR and
// GOCACHE the caller left unset. A sandbox that only lets a hook write a few
// granted folders refuses %TEMP% and the default cache, and every go command
// then fails before it reads a source; one fixed folder per user is what such
// a sandbox can grant once. It stays outside every repository, so t.TempDir()
// of the tests run there finds no repository above it.
func GoWork() ([]string, State, string) {
	home, _ := os.UserHomeDir()
	return goWork(home, os.Getenv)
}

func goWork(home string, getenv func(string) string) ([]string, State, string) {
	var env []string
	for _, v := range []struct{ key, dir string }{{"GOTMPDIR", "tmp"}, {"GOCACHE", "go-cache"}} {
		if getenv(v.key) != "" {
			continue
		}
		if home == "" {
			return nil, StateUnready, v.key + " is unset and there is no home directory to put it under"
		}
		dir := filepath.Join(home, ".loomux", v.dir)
		if err := writable(dir); err != nil {
			if lowIntegrity() {
				exe, _ := os.Executable()
				return nil, StateUnready, fmt.Sprintf("%s %s is not writable: loomux runs at Low integrity, typically because the folder of its binary %s carries a Low label; run a copy from outside that folder (%v)", v.key, dir, exe, err)
			}
			return nil, StateUnready, fmt.Sprintf("%s %s is not writable: grant it in the sandbox or set %s (%v)", v.key, dir, v.key, err)
		}
		env = append(env, v.key+"="+dir)
	}
	return env, "", ""
}

// lowIntegrity is isLowIntegrity, as a variable so that a test can be Low;
// it is asked only once a folder turned out unwritable.
var lowIntegrity = isLowIntegrity

// writable makes dir and proves a child can be made in it: a sandbox may let
// the directory stand and still refuse what the go tool creates inside.
//
//coverage:exempt the MkdirTemp error arm needs a directory that exists but refuses children, which only a sandbox's access rules produce; a test cannot set those up for its own process
func writable(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	probe, err := os.MkdirTemp(dir, "probe-")
	if err != nil {
		return err
	}
	return os.Remove(probe)
}
