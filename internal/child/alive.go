package child

// Alive says whether a process with this number is still running, for a
// holder written into a lock file: a lock whose process is gone guards
// nothing. Numbers are reused, so a false yes is possible and callers keep
// their other rule for that; a false no is not. Zero and below name no
// process.
func Alive(pid int) bool {
	return pid > 0 && alive(pid)
}
