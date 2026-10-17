package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// llmWorld registers a project root as area project/x in a temporary state
// directory.
func llmWorld(t *testing.T, llm bool) (state, root string) {
	t.Helper()
	state = t.TempDir()
	t.Setenv(config.StateDirEnv, state)
	root = configRoot(t, "[area]\nscope = \"project/x\"\n")
	if err := config.AddArea(state, config.Area{Scope: "project/x", Path: root, LLMConfig: llm}); err != nil {
		t.Fatal(err)
	}
	return state, root
}

func TestConfigLLMShowsTheSwitch(t *testing.T) {
	for llm, want := range map[bool]string{false: "llm_config: off (project/x)\n", true: "llm_config: on (project/x)\n"} {
		_, root := llmWorld(t, llm)
		code, out, _ := runConfig(t, "", "llm", "--root", root)
		if code != 0 || out != want {
			t.Errorf("llm=%v: code %d out %q, want %q", llm, code, out, want)
		}
	}
}

func TestConfigLLMTurnsTheSwitch(t *testing.T) {
	state, root := llmWorld(t, false)
	code, out, _ := runConfig(t, "", "llm", "on", "--root", root)
	if code != 0 || out != "llm_config: on (project/x)\n" {
		t.Fatalf("on: code %d out %q", code, out)
	}
	areas, _ := config.ReadRegistry(state)
	if !areas[0].LLMConfig {
		t.Error("registry lacks llm_config after on")
	}
	code, out, _ = runConfig(t, "", "llm", "off", "--root", root)
	if code != 0 || out != "llm_config: off (project/x)\n" {
		t.Fatalf("off: code %d out %q", code, out)
	}
	areas, _ = config.ReadRegistry(state)
	if areas[0].LLMConfig {
		t.Error("registry keeps llm_config after off")
	}
}

func TestConfigLLMOutsideAnArea(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())
	root := configRoot(t, "")
	// First without a registry, then with one that holds another area.
	for round := 0; round < 2; round++ {
		for _, args := range [][]string{{"llm", "--root", root}, {"llm", "on", "--root", root}} {
			code, _, errOut := runConfig(t, "", args...)
			if code != 1 || !strings.Contains(errOut, "loomux init") {
				t.Errorf("round %d %v: code %d stderr %q", round, args, code, errOut)
			}
		}
		other := t.TempDir()
		if round == 0 {
			if err := config.AddArea(os.Getenv(config.StateDirEnv), config.Area{Scope: "project/y", Path: other}); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, args := range [][]string{{"llm", "maybe"}, {"llm", "on", "off"}, {"llm", "--global"}, {"llm", "on", "--yes"}, {"llm", "--bogus"}} {
		if code, _, _ := runConfig(t, "", args...); code != 2 {
			t.Errorf("%v: code %d, want 2", args, code)
		}
	}
}

func TestConfigLLMNeedsAProjectRoot(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())
	t.Chdir(t.TempDir())
	if code, _, errOut := runConfig(t, "", "llm"); code != 1 || errOut == "" {
		t.Errorf("code %d stderr %q", code, errOut)
	}
}

func TestConfigLLMReportsAFailingRegistry(t *testing.T) {
	// A state directory that is a file: neither read nor write can succeed.
	state, root := llmWorld(t, false)
	t.Setenv(config.StateDirEnv, state+"/nope/deeper")
	for _, args := range [][]string{{"llm", "--root", root}, {"llm", "on", "--root", root}} {
		if code, _, _ := runConfig(t, "", args...); code != 1 {
			t.Errorf("%v: code %d, want 1", args, code)
		}
	}
}

func TestConfigListNamesTheOpenSwitch(t *testing.T) {
	_, root := llmWorld(t, true)
	_, out, _ := runConfig(t, "", "list", "--root", root)
	if !strings.HasPrefix(out, "llm_config: on — agents may edit this file\n") {
		t.Errorf("list lacks the line:\n%s", out)
	}
	_, js, _ := runConfig(t, "", "list", "--json", "--root", root)
	if strings.Contains(js, "llm_config") {
		t.Errorf("json carries the switch:\n%s", js)
	}
	_, root = llmWorld(t, false)
	_, out, _ = runConfig(t, "", "list", "--root", root)
	if strings.Contains(out, "llm_config") {
		t.Errorf("closed switch shown:\n%s", out)
	}
}
