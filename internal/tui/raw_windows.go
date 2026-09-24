//go:build windows

package tui

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableVT lets the console interpret the escape sequences the widgets
// write; conhost does not by default.
//
//coverage:exempt changes the mode of a real console handle
func enableVT(out *os.File) func() {
	handle := windows.Handle(out.Fd())
	var mode uint32
	if windows.GetConsoleMode(handle, &mode) != nil {
		return func() {}
	}
	_ = windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	return func() { _ = windows.SetConsoleMode(handle, mode) }
}
