//go:build windows

package child

import (
	"errors"

	"golang.org/x/sys/windows"
)

// stillActive is the exit code GetExitCodeProcess reports for a process that
// has not ended.
const stillActive = 259

// alive opens the process for the least right there is. A process that is
// gone cannot be opened; one this user may not open is somebody's and runs.
// A failed exit-code query counts as running too: the caller's other rule
// covers a false yes, nothing covers a false no.
func alive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return errors.Is(err, windows.ERROR_ACCESS_DENIED)
	}
	defer windows.CloseHandle(h)
	var code uint32
	return windows.GetExitCodeProcess(h, &code) != nil || code == stillActive
}
