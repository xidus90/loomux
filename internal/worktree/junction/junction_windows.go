//go:build windows

package junction

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

// The header size of the buffer below: the 8-byte REPARSE_DATA_BUFFER head
// plus four uint16 -- SubstituteNameOffset, SubstituteNameLength,
// PrintNameOffset, PrintNameLength. Spelled out because x/sys keeps its own
// layout types unexported -- `mountPointReparseBuffer` and `reparseDataBuffer`
// in `windows/types_windows.go:1879,1887` of the pinned v0.18.0 -- so they
// cannot be named from here. The full reasoning is at reparseTarget, which
// reads back what this writes.
const mountPointHeaderSize = 8 + 8

// Create makes `link` a junction pointing at `target`.
//
// Three steps, because that is what Windows wants: the directory has to exist
// before it can carry a reparse point, the point is set with an ioctl, and a
// failure has to take the empty directory back out -- an empty directory where
// a junction belongs is worse than nothing, since `Target` reads it as "not a
// link" and the next run would try to create it again.
//
//coverage:exempt filepath.Abs fails only when the working directory cannot be read, and os.Stat on the same target succeeded just before
func Create(link, target string) error {
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("the target %s is not there: %w", target, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("the target %s is not a directory", target)
	}
	absolute, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolving the target %s: %w", target, err)
	}
	// Mkdir and not MkdirAll: an occupied path must be a failure, and MkdirAll
	// is happy with a directory that is already there.
	if err := os.Mkdir(link, 0o755); err != nil {
		return fmt.Errorf("making room for the junction %s: %w", link, err)
	}
	if err := setMountPoint(link, absolute); err != nil {
		// Best effort, and its failure must not hide the first one.
		_ = os.Remove(link)
		return err
	}
	return nil
}

//coverage:exempt the second UTF16FromString sees the same target the first one already accepted, so nothing can make it the one that fails
func setMountPoint(link, target string) error {
	// The NT form, which is what a reparse point stores -- with the trailing
	// separator, so `\??\C:\dir\`. Windows itself does not insist: measured on
	// 2026-09-07, `mklink /J` stores `\??\C:\dir` without one and that resolves
	// too, so this is a choice and not a requirement. It is kept because the
	// round-trip test proves this form works and because a stored target that
	// ends in a separator cannot be mistaken for a file; `Target` says in turn
	// that no caller may compare on it.
	substitute, err := windows.UTF16FromString(`\??\` + target + `\`)
	if err != nil {
		return fmt.Errorf("encoding the target %s: %w", target, err)
	}
	print16, err := windows.UTF16FromString(target)
	if err != nil {
		return fmt.Errorf("encoding the target %s: %w", target, err)
	}
	// UTF16FromString appends a NUL that is not part of either length.
	substituteBytes := (len(substitute) - 1) * 2
	printBytes := (len(print16) - 1) * 2

	buffer := make([]byte, mountPointHeaderSize+substituteBytes+2+printBytes+2)
	put32(buffer[0:], windows.IO_REPARSE_TAG_MOUNT_POINT)
	put16(buffer[4:], uint16(8+substituteBytes+2+printBytes+2))
	put16(buffer[6:], 0)
	put16(buffer[8:], 0)
	put16(buffer[10:], uint16(substituteBytes))
	put16(buffer[12:], uint16(substituteBytes+2))
	put16(buffer[14:], uint16(printBytes))
	copyUTF16(buffer[mountPointHeaderSize:], substitute)
	copyUTF16(buffer[mountPointHeaderSize+substituteBytes+2:], print16)

	path, err := windows.UTF16PtrFromString(link)
	if err != nil {
		return fmt.Errorf("encoding the link %s: %w", link, err)
	}
	// FILE_FLAG_OPEN_REPARSE_POINT, so the open lands on the directory itself
	// rather than following what may already be there; FILE_FLAG_BACKUP_SEMANTICS,
	// because a directory cannot be opened without it.
	handle, err := windows.CreateFile(
		path,
		windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return fmt.Errorf("opening %s: %w", link, err)
	}
	defer windows.CloseHandle(handle)

	var returned uint32
	if err := windows.DeviceIoControl(
		handle,
		windows.FSCTL_SET_REPARSE_POINT,
		&buffer[0],
		uint32(len(buffer)),
		nil,
		0,
		&returned,
		nil,
	); err != nil {
		return fmt.Errorf("pointing %s at %s: %w", link, target, err)
	}
	return nil
}

// reparseTarget reads back what `setMountPoint` wrote, and answers "" for
// everything that is not a mount point.
//
// The buffer is parsed by hand because x/sys gives no way not to. Checked in
// v0.18.0, the version this module pins: it carries the tag and the ioctl, and
// it does carry the layout -- `mountPointReparseBuffer` and
// `reparseDataBuffer` in `windows/types_windows.go` -- but all of it
// unexported, so none of it can be named from here. Writing and reading the
// same offsets in one file is then the next best thing: the two sides move
// together or not at all.
//
// Those types do confirm the arithmetic, which is worth more than reusing them
// would have been: tag, length and reserved make 8, the four name fields
// another 8, and `PathBuffer` follows -- the 16 of `mountPointHeaderSize`.
//
//coverage:exempt CreateFile fails only if the OS locks the path between the attribute query and the open, and DeviceIoControl(FSCTL_GET_REPARSE_POINT) only on a reparse point the filesystem cannot read back
func reparseTarget(link string) (string, error) {
	path, err := windows.UTF16PtrFromString(link)
	if err != nil {
		return "", fmt.Errorf("encoding the link %s: %w", link, err)
	}
	attributes, err := windows.GetFileAttributes(path)
	if err != nil {
		return vanished(link, fmt.Errorf("inspecting %s: %w", link, err))
	}
	if attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0 {
		return "", nil
	}
	// No access rights at all, because reading a reparse point needs none, and
	// every share flag, because refusing to answer about a directory somebody
	// else has open would make the sweep depend on who is looking.
	handle, err := windows.CreateFile(
		path,
		0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return vanished(link, fmt.Errorf("opening %s: %w", link, err))
	}
	defer windows.CloseHandle(handle)

	buffer := make([]byte, windows.MAXIMUM_REPARSE_DATA_BUFFER_SIZE)
	var returned uint32
	if err := windows.DeviceIoControl(
		handle,
		windows.FSCTL_GET_REPARSE_POINT,
		nil,
		0,
		&buffer[0],
		uint32(len(buffer)),
		&returned,
		nil,
	); err != nil {
		return "", fmt.Errorf("reading the reparse point %s: %w", link, err)
	}
	return mountPointFromBuffer(buffer[:returned], link)
}

// mountPointFromBuffer reads the target out of what FSCTL_GET_REPARSE_POINT
// answered, and "" out of every reparse point that is not a mount point.
//
// A function of its own so a test can hand it the buffers the filesystem
// cannot be made to produce here: another tag needs a symbolic link and thus a
// privilege, and a length that runs past the data needs a filesystem that
// lies. Offsets read here mirror the ones `setMountPoint` writes, which is the
// whole reason the two sit in one file.
func mountPointFromBuffer(buffer []byte, link string) (string, error) {
	// A short answer cannot carry the offsets, and a tag of another kind is a
	// symbolic link or a placeholder -- not ours to report as a junction and
	// above all not ours to remove.
	if len(buffer) < mountPointHeaderSize {
		return "", nil
	}
	if get32(buffer[0:]) != windows.IO_REPARSE_TAG_MOUNT_POINT {
		return "", nil
	}
	offset := mountPointHeaderSize + int(get16(buffer[8:]))
	end := offset + int(get16(buffer[10:]))
	if end > len(buffer) {
		return "", fmt.Errorf("the reparse point %s names a target past its own data", link)
	}
	return decodeUTF16(buffer[offset:end]), nil
}

func put16(destination []byte, value uint16) {
	destination[0] = byte(value)
	destination[1] = byte(value >> 8)
}

func put32(destination []byte, value uint32) {
	destination[0] = byte(value)
	destination[1] = byte(value >> 8)
	destination[2] = byte(value >> 16)
	destination[3] = byte(value >> 24)
}

func get16(source []byte) uint16 {
	return uint16(source[0]) | uint16(source[1])<<8
}

func get32(source []byte) uint32 {
	return uint32(source[0]) | uint32(source[1])<<8 |
		uint32(source[2])<<16 | uint32(source[3])<<24
}

func copyUTF16(destination []byte, source []uint16) {
	for index, unit := range source[:len(source)-1] {
		put16(destination[index*2:], unit)
	}
}

// decodeUTF16 turns a stretch of the reparse buffer back into a string. An odd
// trailing byte cannot belong to a UTF-16 unit and is dropped rather than
// panicking: the length came from the filesystem, not from this package.
func decodeUTF16(source []byte) string {
	units := make([]uint16, 0, len(source)/2)
	for index := 0; index+1 < len(source); index += 2 {
		units = append(units, get16(source[index:]))
	}
	return string(utf16.Decode(units))
}

// vanished turns a failure on a path that is no longer there into the empty
// answer Target promises for one; any other failure stays the error it was.
// It asks Lstat again rather than testing the errno, because Lstat handles
// long paths and the raw calls do not.
func vanished(link string, err error) (string, error) {
	if _, statErr := os.Lstat(link); os.IsNotExist(statErr) {
		return "", nil
	}
	return "", err
}
