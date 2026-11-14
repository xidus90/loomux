//go:build !windows

package web

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// processImage is the executable a process runs: the target of /proc/<pid>/exe
// where the kernel has one, else what ps prints as its command.
//
//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func processImage(pid int) (string, error) {
	if image, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil {
		return image, nil
	}
	out, err := exec.Command("ps", "-p", fmt.Sprint(pid), "-o", "comm=").Output()
	return strings.TrimSpace(string(out)), err
}
