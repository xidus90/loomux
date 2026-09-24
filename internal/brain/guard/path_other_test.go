//go:build !windows

package guard

import "testing"

// absentVolume is a root no posix file system carries. There are no drive
// letters to be taken here, so there is nothing to choose and nothing to
// skip.
func absentVolume(*testing.T) string { return "/proc/self/no-such-root" }
