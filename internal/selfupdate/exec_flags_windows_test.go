package selfupdate

import (
	"context"
	"testing"
)

// serve runs detached, without a console; a console program it starts gets a
// console of its own, and on a desktop that is a window popping up once a day
// for every call of the pass.
func TestCommandStartsWithoutAConsoleWindow(t *testing.T) {
	cmd := command(context.Background(), "gh", "release", "list")
	if cmd.SysProcAttr == nil {
		t.Fatal("command sets no creation flags")
	}
	if cmd.SysProcAttr.CreationFlags != createNoWindow || !cmd.SysProcAttr.HideWindow {
		t.Fatalf("creation flags %#x, hide window %v; want CREATE_NO_WINDOW and a hidden window",
			cmd.SysProcAttr.CreationFlags, cmd.SysProcAttr.HideWindow)
	}
}
