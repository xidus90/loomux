//go:build !windows

package verify

// isLowIntegrity is false where there are no integrity levels.
func isLowIntegrity() bool { return false }
