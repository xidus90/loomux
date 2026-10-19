package ask

// TakeOver exposes the stale-lock takeover, whose losing arm needs a race
// between two runs that no black-box test can stage on demand.
var TakeOver = takeOver

// Unlock exposes the release, whose retry needs a reader holding the lock
// open at the moment of the remove.
var Unlock = unlock

// SwapRemoveLock stages a remove that fails the way Windows fails one while
// another process reads the file, and returns the undo.
func SwapRemoveLock(f func(name string) error) func() {
	old := removeLock
	removeLock = f
	return func() { removeLock = old }
}

// SwapLinkFile stages a filesystem without hard links and returns the undo.
func SwapLinkFile(f func(oldname, newname string) error) func() {
	old := linkFile
	linkFile = f
	return func() { linkFile = old }
}
