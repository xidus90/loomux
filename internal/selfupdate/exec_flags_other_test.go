//go:build !windows

package selfupdate

import (
	"context"
	"testing"
)

func TestCommandAsksForNothingOffWindows(t *testing.T) {
	if cmd := command(context.Background(), "gh", "release", "list"); cmd.SysProcAttr != nil {
		t.Fatalf("command sets %+v", cmd.SysProcAttr)
	}
}
