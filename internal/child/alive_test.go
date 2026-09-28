package child

import (
	"os"
	"os/exec"
	"testing"
)

func TestAliveKnowsThisProcess(t *testing.T) {
	if !Alive(os.Getpid()) {
		t.Fatal("this process is not alive")
	}
}

// A process that ran and was reaped is gone; its number is only a number.
func TestAliveForgetsAFinishedChild(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if pid := cmd.Process.Pid; Alive(pid) {
		t.Fatalf("process %d still counts as alive", pid)
	}
}

// Zero and below name no process; on Unix a signal there would reach a
// process group.
func TestAliveRefusesNumbersThatAreNoProcess(t *testing.T) {
	for _, pid := range []int{0, -1, -2} {
		if Alive(pid) {
			t.Fatalf("%d counts as alive", pid)
		}
	}
}
