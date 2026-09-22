package vcs

// SetBeforeUpdateRef installs the seam for the window a foreign process would
// use, and hands back what puts the previous one back. It lives in a test file
// so the tests of package vcs_test, which own the repository fixtures, can
// reach a variable no caller of this package may touch.
func SetBeforeUpdateRef(hook func()) (restore func()) {
	previous := beforeUpdateRef
	beforeUpdateRef = hook
	return func() { beforeUpdateRef = previous }
}
