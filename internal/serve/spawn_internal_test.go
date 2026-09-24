package serve

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

func TestSpawnFailsWhenTheProgramCannotBeNamed(t *testing.T) {
	saved := executablePath
	executablePath = func() (string, error) { return "", errors.New("no name for this program") }
	t.Cleanup(func() { executablePath = saved })

	_, err := Spawn(t.TempDir(), func(*exec.Cmd) error {
		t.Error("the spawner ran although there is nothing to start")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "no name for this program") {
		t.Fatalf("Spawn: %v, want the failure of naming the running program", err)
	}
}

func TestSpawnFailsWhenTheStateDirectoryCannotBeResolved(t *testing.T) {
	// filepath.Abs fails only where os.Getwd does: a working directory pulled
	// out from under the running process. Nothing in this package reaches that
	// arm without the seam.
	saved := absolutePath
	absolutePath = func(string) (string, error) { return "", errors.New("no working directory") }
	t.Cleanup(func() { absolutePath = saved })

	_, err := Spawn("state", func(*exec.Cmd) error {
		t.Error("the spawner ran although nobody knows where the state is")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "no working directory") {
		t.Fatalf("Spawn: %v, want the failure of resolving the state directory", err)
	}
}

func TestChildEnvReplacesWhatItSetsAndKeepsTheRest(t *testing.T) {
	// An entry without '=' is not a variable; Windows puts such entries in a
	// process block for its own bookkeeping, and they are passed through.
	base := []string{"PATH=/bin", "LOOMUX_STATE_DIR=/old", "loomux_state_dir=/older", "=C:=C:\\", "odd"}
	got := childEnv(base, "/new", false)

	want := []string{"PATH=/bin", "=C:=C:\\", "odd", "LOOMUX_STATE_DIR=/new"}
	if len(got) != len(want) {
		t.Fatalf("childEnv = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d is %q, want %q", i, got[i], want[i])
		}
	}
}

func TestChildEnvAnnouncesTheBreakawayOnlyWhenThereWasOne(t *testing.T) {
	with := childEnv(nil, "/state", true)
	if len(with) != 2 || with[1] != BrokeAwayEnv+"=1" {
		t.Errorf("childEnv = %v, want the breakaway announced", with)
	}
	without := childEnv([]string{BrokeAwayEnv + "=1"}, "/state", false)
	for _, entry := range without {
		if strings.HasPrefix(entry, BrokeAwayEnv+"=") {
			t.Errorf("childEnv = %v, want no breakaway left over from the parent", without)
		}
	}
}

// gh reads its login from the user's configuration, and serve runs it. The
// child keeps every variable it does not replace, these included.
func TestChildEnvKeepsWhatGhNeeds(t *testing.T) {
	base := []string{`PATH=C:\bin`, `APPDATA=C:\a`, `USERPROFILE=C:\u`, `GH_CONFIG_DIR=C:\gh`}
	got := childEnv(base, "/state", false)
	for _, want := range base {
		if !slices.Contains(got, want) {
			t.Errorf("childEnv dropped %s: %v", want, got)
		}
	}
}
