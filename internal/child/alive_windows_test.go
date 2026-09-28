//go:build windows

package child

import (
	"os"
	"os/exec"
	"testing"

	"golang.org/x/sys/windows"
)

// A process that ended while someone still holds a handle to it can be
// opened; its exit code, not the open, says it is gone.
func TestAliveForgetsAnEndedProcessSomeoneStillHolds(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if Alive(pid) {
		t.Fatalf("ended process %d still counts as alive", pid)
	}
}
