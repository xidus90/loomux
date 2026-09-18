package ask

// TakeOver exposes the stale-lock takeover, whose losing arm needs a race
// between two runs that no black-box test can stage on demand.
var TakeOver = takeOver

// SwapLinkFile stages a filesystem without hard links and returns the undo.
func SwapLinkFile(f func(oldname, newname string) error) func() {
	old := linkFile
	linkFile = f
	return func() { linkFile = old }
}
