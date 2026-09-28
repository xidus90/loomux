//go:build !windows

package hooks

import "testing"

func TestStrictModeResolvesATrailingDot(t *testing.T) {
	t.Skip("a trailing dot names another file outside Windows; the strict test runs on Windows")
}

func TestStrictModeResolvesAShortName(t *testing.T) {
	t.Skip("8.3 short names are Windows'; the strict test runs on Windows")
}
