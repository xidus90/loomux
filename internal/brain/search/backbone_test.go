package search_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/brain/search/backbonetest"
	"github.com/xidus90/loomux/internal/config"
)

// lookupIn is os.LookupEnv over a fixed environment.
func lookupIn(env map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	}
}

// TestResolveBackboneOrder is the precedence: a variable the user set, even an
// empty one, beats the setting, and the setting beats nothing. The setting is
// read from a state of the test's own, as the ports read it; "" is a state
// without a config.toml.
func TestResolveBackboneOrder(t *testing.T) {
	for name, c := range map[string]struct {
		setting string
		env     map[string]string
		vars    map[string]string
		byUser  bool
	}{
		"gpu set":        {"cpu", map[string]string{"QMD_LLAMA_GPU": "cuda"}, map[string]string{}, true},
		"cpu set":        {"vulkan", map[string]string{"QMD_FORCE_CPU": "1"}, map[string]string{}, true},
		"empty gpu set":  {"vulkan", map[string]string{"QMD_LLAMA_GPU": ""}, map[string]string{}, true},
		"empty cpu set":  {"cpu", map[string]string{"QMD_FORCE_CPU": ""}, map[string]string{}, true},
		"user, no file":  {"", map[string]string{"QMD_FORCE_CPU": "1"}, map[string]string{}, true},
		"setting vulkan": {"vulkan", map[string]string{"PATH": "x"}, map[string]string{"QMD_LLAMA_GPU": "vulkan"}, false},
		"setting cpu":    {"cpu", nil, map[string]string{"QMD_FORCE_CPU": "1"}, false},
		"setting cuda":   {"cuda", nil, map[string]string{}, false},
		"nothing":        {"", nil, map[string]string{}, false},
	} {
		state := t.TempDir()
		if c.setting != "" {
			state = writeGlobal(t, "[search]\nbackbone = \""+c.setting+"\"\n")
		}
		configured, err := search.ConfiguredBackbone(state)
		if err != nil {
			t.Fatal(err)
		}
		vars, byUser := search.ResolveBackbone(configured, lookupIn(c.env))
		if !reflect.DeepEqual(vars, c.vars) || byUser != c.byUser {
			t.Errorf("%s: vars %v, by user %v; want %v, %v", name, vars, byUser, c.vars, c.byUser)
		}
	}
}

// writeGlobal writes the machine-wide config.toml of a state of the test's own.
func writeGlobal(t *testing.T, text string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestConfiguredBackboneReadsTheGlobalFile(t *testing.T) {
	if b, err := search.ConfiguredBackbone(t.TempDir()); err != nil || b != search.DefaultBackbone {
		t.Fatalf("no file: %q %v", b, err)
	}
	if b, err := search.ConfiguredBackbone(writeGlobal(t, "[search]\nbackbone = \"cpu\"\n")); err != nil || b != search.BackboneCPU {
		t.Fatalf("cpu: %q %v", b, err)
	}
	dir := writeGlobal(t, "[search]\nbackbone = \"metal\"\n")
	if _, err := search.ConfiguredBackbone(dir); err == nil || !strings.Contains(err.Error(), filepath.Join(dir, "config.toml")) {
		t.Fatalf("metal: %v", err)
	}
}

// The default is one value, whichever package names it.
func TestTheDefaultBackboneIsTheConfigsDefault(t *testing.T) {
	if string(search.DefaultBackbone) != config.DefaultSearchBackbone {
		t.Fatalf("%q, %q", search.DefaultBackbone, config.DefaultSearchBackbone)
	}
}

// helperEnv makes this test binary, started as a child, print the backbone
// variables it was given instead of running tests.
const helperEnv = "LOOMUX_TEST_PRINT_BACKBONE"

// TestMain lets the test binary stand in for qmd: started with helperEnv, it
// answers `ls` with one listed path per backbone variable, which says whether
// the variable was there and what it held.
func TestMain(m *testing.M) {
	if os.Getenv(helperEnv) == "1" {
		for _, key := range []string{"QMD_LLAMA_GPU", "QMD_FORCE_CPU"} {
			value, ok := os.LookupEnv(key)
			fmt.Printf("1  2026-09-27  qmd://c/%s=%q %v\n", key, value, ok)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// childEnv runs this test binary as qmd through port, with the runner the
// port picks itself, and returns what the child saw of the two variables.
func childEnv(t *testing.T, port *search.QmdPort) string {
	t.Helper()
	t.Setenv(helperEnv, "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	port.Executable = executable
	seen, err := port.Indexed("c")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(seen, "\n")
}

// TestTheCommandLineGetsTheBackbone: a qmd process started through a port
// gets the backbone's variables on top of the caller's environment, and
// nothing when the user chose one.
func TestTheCommandLineGetsTheBackbone(t *testing.T) {
	backbonetest.Clear(t)
	got := childEnv(t, &search.QmdPort{Backbone: search.BackboneVulkan})
	if !strings.Contains(got, `QMD_LLAMA_GPU="vulkan" true`) || !strings.Contains(got, `QMD_FORCE_CPU="" false`) {
		t.Fatalf("vulkan: %s", got)
	}
	got = childEnv(t, &search.QmdPort{Backbone: search.BackboneCUDA})
	if !strings.Contains(got, `QMD_LLAMA_GPU="" false`) || !strings.Contains(got, `QMD_FORCE_CPU="" false`) {
		t.Fatalf("cuda: %s", got)
	}
	// The zero port runs as it always did: with the caller's environment.
	got = childEnv(t, &search.QmdPort{})
	if !strings.Contains(got, `QMD_LLAMA_GPU="" false`) {
		t.Fatalf("zero: %s", got)
	}
	t.Setenv("QMD_FORCE_CPU", "")
	got = childEnv(t, &search.QmdPort{Backbone: search.BackboneVulkan})
	if !strings.Contains(got, `QMD_LLAMA_GPU="" false`) || !strings.Contains(got, `QMD_FORCE_CPU="" true`) {
		t.Fatalf("user's: %s", got)
	}
}

func TestBackboneRunnerRefusesAnEmptyCommand(t *testing.T) {
	if _, _, code, err := search.BackboneRunner(search.BackboneCPU)(nil); err == nil || code != 1 {
		t.Fatalf("%d %v", code, err)
	}
}

// TestTheDaemonGetsTheBackbone goes the way a real start goes, down to the
// spawner: the port's backbone reaches the daemon's environment, and the
// user's variable keeps it out.
func TestTheDaemonGetsTheBackbone(t *testing.T) {
	for name, c := range map[string]struct {
		user   map[string]string
		want   string
		absent string
	}{
		"setting":     {nil, "QMD_LLAMA_GPU=vulkan", ""},
		"user's word": {map[string]string{"QMD_FORCE_CPU": "1"}, "QMD_FORCE_CPU=1", "QMD_LLAMA_GPU=vulkan"},
	} {
		t.Run(name, func(t *testing.T) {
			backbonetest.Clear(t)
			for key, value := range c.user {
				t.Setenv(key, value)
			}
			var env []string
			spawner := func(_ []string, e []string) error { env = e; return errors.New("spawned") }
			launcher := func(string) ([]string, error) { return []string{"qmd"}, nil }
			connect := search.DefaultConnectWith(filepath.Join(t.TempDir(), "qmd.lock"), 64993, launcher, spawner, time.Millisecond, nil)
			port := search.NewQmdMcpPort(search.WithBackbone(search.BackboneVulkan), search.WithConnect(connect), search.WithPort(64993))
			if _, err := port.Search("q", []string{"c"}, search.ProfileFast, 1); err == nil {
				t.Fatal("a daemon that never started answered")
			}
			if !slices.Contains(env, c.want) || (c.absent != "" && slices.Contains(env, c.absent)) {
				t.Fatalf("env %v", env)
			}
		})
	}
}

// An unavailable port answers every question with its error: a setting that
// does not read stops the search, not the answer around it.
func TestUnavailableAnswersWithItsError(t *testing.T) {
	want := errors.New("broken")
	port := search.Unavailable(want)
	_, searchErr := port.Search("q", nil, search.ProfileFast, 1)
	_, indexedErr := port.Indexed("c")
	_, pendingErr := port.NotYetSearchable()
	for i, err := range []error{searchErr, indexedErr, port.Refresh(nil), pendingErr, port.Embed(nil)} {
		if !errors.Is(err, want) {
			t.Errorf("%d: %v", i, err)
		}
	}
}
