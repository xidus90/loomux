// Package pathkey compares paths the way the file system of a platform
// tells them apart: in its slash direction, and without regard to case
// where the file system ignores it. Glob is the one exception to the case
// rule: it matches slash-separated paths against the globs of a policy rule
// or a lane, byte for byte.
package pathkey

import (
	"path/filepath"
	"runtime"
	"strings"
)

// Same says whether a and b name the same path on this platform.
func Same(a, b string) bool {
	return Key(a) == Key(b)
}

// Key is path in the one spelling every name of it shares on this platform,
// fit for a map key.
func Key(path string) string {
	return KeyOn(runtime.GOOS, path)
}

// KeyOn is Key for goos. Windows and macOS ignore case by default; Linux
// and the other systems do not.
func KeyOn(goos, path string) string {
	clean := filepath.Clean(filepath.FromSlash(path))
	if goos == "windows" || goos == "darwin" {
		return strings.ToLower(clean)
	}
	return clean
}
