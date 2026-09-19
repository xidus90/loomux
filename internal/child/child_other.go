//go:build !windows

package child

import (
	"os/exec"
	"syscall"
)

type tree struct{ pid int }

// start puts the child into a process group of its own, so one signal to the
// group reaches every grandchild that did not leave it.
//
//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func start(cmd *exec.Cmd) (*tree, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &tree{pid: cmd.Process.Pid}, nil
}

//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func (t *tree) kill() { syscall.Kill(-t.pid, syscall.SIGKILL) }

//coverage:exempt POSIX-only; there is no job to release
func (t *tree) release() {}
