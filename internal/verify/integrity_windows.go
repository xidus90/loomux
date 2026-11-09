//go:build windows

package verify

import (
	"encoding/binary"
	"unsafe"

	"golang.org/x/sys/windows"
)

// mediumRID is SECURITY_MANDATORY_MEDIUM_RID, the integrity level of an
// ordinary process; x/sys/windows does not name it.
const mediumRID = 0x2000

// isLowIntegrity reads the integrity label of the process's own token. A
// process started from a file carrying a Low label runs at Low, and Low may
// only write where the folder is labelled Low as well.
//
//coverage:exempt the error arm needs GetTokenInformation to fail on the process's own token, which a test cannot provoke
func isLowIntegrity() bool {
	// A label holds a SID with one sub-authority: 16 bytes of struct plus
	// 12 of SID on 64-bit, so 64 leaves room. uint64 words keep the buffer
	// aligned for the pointer the struct holds, and the heap keeps it in
	// place: the kernel writes the SID's absolute address into it, which a
	// moving goroutine stack would leave dangling.
	buf := make([]uint64, 8)
	var n uint32
	if err := windows.GetTokenInformation(windows.GetCurrentProcessToken(), windows.TokenIntegrityLevel, (*byte)(unsafe.Pointer(&buf[0])), uint32(len(buf)*8), &n); err != nil {
		return false
	}
	// The SID is read from the bytes, not through SID.SubAuthority: that goes
	// through GetSidSubAuthority, whose result x/sys turns from an integer into
	// a pointer, which -race's checkptr refuses. A SID is a revision byte, a
	// count byte, six bytes of authority and count little-endian uint32s; the
	// level is the last of them.
	raw := unsafe.Slice((*byte)(unsafe.Pointer(&buf[0])), len(buf)*8)
	sid := (*windows.Tokenmandatorylabel)(unsafe.Pointer(&buf[0])).Label.Sid
	off := int(uintptr(unsafe.Pointer(sid)) - uintptr(unsafe.Pointer(&buf[0])))
	last := off + 8 + 4*(int(raw[off+1])-1)
	return binary.LittleEndian.Uint32(raw[last:last+4]) < mediumRID
}
