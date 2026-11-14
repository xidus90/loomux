//go:build windows

package web

import "golang.org/x/sys/windows"

// processImage is the path of the executable a process runs, read with the
// least right there is. A process this user may not open has no answer.
func processImage(pid int) (string, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	// A failed query leaves the buffer empty, so its path is empty with it.
	err = windows.QueryFullProcessImageName(h, 0, &buf[0], &size)
	return windows.UTF16ToString(buf[:size]), err
}
