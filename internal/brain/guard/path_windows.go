//go:build windows

package guard

import (
	"errors"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// The Windows half of the resolver: the one call `Path.resolve()` makes
// that has no portable spelling. It is a file of its own because
// `golang.org/x/sys/windows` builds nowhere else, and the barrier has to
// compile on a runner that is not this machine.

// The two flags `GetFinalPathNameByHandle` is asked for. Both are zero
// and both are the API's defaults -- `FILE_NAME_NORMALIZED` for the
// spelling the file system keeps rather than the one the handle was
// opened under, `VOLUME_NAME_DOS` for a drive letter rather than a volume
// GUID -- and they are named here because a zero passed as a flag word
// says nothing about which defaults were meant. They are not in
// `golang.org/x/sys/windows` at the version this module pins.
const (
	fileNameNormalized = 0x0
	volumeNameDOS      = 0x0
)

// finalName is `nt._getfinalpathname`, the call `ntpath.realpath` reaches
// for first: open the path and ask the file system what it is really
// called.
//
// `OPEN_REPARSE_POINT` is deliberately *not* among the flags, so the open
// follows junctions and symlinks rather than answering the link itself --
// which is the whole reason this call stands here and
// `filepath.EvalSymlinks` does not. `FILE_FLAG_BACKUP_SEMANTICS` is what
// lets a directory be opened at all, and a request of no access rights
// means the caller needs no permission to read the file's contents to ask
// for its name.
//
// The `\\?\` prefix comes off unconditionally, unlike Python's, which
// keeps it wherever the plain path would not resolve back to the same
// file. The comment above `ResolvePath` carries the ground and the one
// call it decides.
func finalName(path string) (string, error) {
	wide, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	handle, err := windows.CreateFile(wide, 0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|
			windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)
	buffer := make([]uint16, windows.MAX_PATH)
	for {
		length, err := windows.GetFinalPathNameByHandle(handle,
			&buffer[0], uint32(len(buffer)),
			fileNameNormalized|volumeNameDOS)
		if err != nil {
			return "", err
		}
		// On success the answer is the length without the closing zero;
		// on a buffer too small it is the length *with* it. Either way a
		// number that does not fit means growing and asking again.
		//
		// `<` in place of `<=` survives (a3), and rightly: a successful
		// answer needs room for the zero too, so a length exactly the
		// size of the buffer can only be the second kind, where both
		// spellings grow.
		if int(length) <= len(buffer) {
			return withoutDevicePrefix(
				windows.UTF16ToString(buffer[:length])), nil
		}
		buffer = make([]uint16, length)
	}
}

// withoutDevicePrefix strips the `\\?\` that `GetFinalPathNameByHandle`
// always puts in front, the way `ntpath.realpath` does -- including the
// UNC spelling, where `\\?\UNC\srv\share` is the plain `\\srv\share`.
func withoutDevicePrefix(path string) string {
	const device, unc = `\\?\`, `\\?\UNC\`
	if strings.HasPrefix(path, unc) {
		return `\\` + path[len(unc):]
	}
	return strings.TrimPrefix(path, device)
}

// stopsResolving is `allowed_winerror` of `_getfinalpathname_nonstrict`
// (ntpath.py:626), copied by number because there is nothing to import.
// Every one of them says "this path, as spelt, is not there" in some
// wording -- a missing name, a missing directory, a volume with no
// medium, a name the file system will not even parse, a link that cannot
// be followed -- and the walk answers all of them by shortening the path
// and asking again. Anything else is a fault the barrier has no business
// guessing past, so it comes back as a refusal.
func stopsResolving(err error) bool {
	var code syscall.Errno
	if !errors.As(err, &code) {
		return false
	}
	switch uintptr(code) {
	case 1, 2, 3, 5, 21, 32, 50, 53, 65, 67, 87, 123, 161, 1005, 1920,
		1921:
		return true
	}
	return false
}

// One arm of Python's is not ported: where the open fails with one of a
// smaller set of errors, `_getfinalpathname_nonstrict` calls
// `_findfirstfile` to learn the on-disk spelling of the component it is
// about to move onto the tail. It answers a *name*, not a place, so it
// can only ever change the spelling this side prints; and no call in the
// corpus reaches it, because a component that can be listed can be
// opened for no access at all. Named here rather than left silent.
