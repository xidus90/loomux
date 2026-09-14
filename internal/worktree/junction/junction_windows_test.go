//go:build windows

package junction

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// mountPointBuffer builds what the kernel answers for a junction, so a test can
// vary one field of it at a time.
func mountPointBuffer(tag uint32, target string) []byte {
	substitute, err := windows.UTF16FromString(target)
	if err != nil {
		panic(err)
	}
	substituteBytes := (len(substitute) - 1) * 2
	buffer := make([]byte, mountPointHeaderSize+substituteBytes+2+2)
	put32(buffer[0:], tag)
	put16(buffer[4:], uint16(8+substituteBytes+2+2))
	put16(buffer[8:], 0)
	put16(buffer[10:], uint16(substituteBytes))
	put16(buffer[12:], uint16(substituteBytes+2))
	put16(buffer[14:], 0)
	copyUTF16(buffer[mountPointHeaderSize:], substitute)
	return buffer
}

// The buffers the filesystem will not produce on demand: another tag needs the
// privilege a symbolic link needs, and a length past the data needs a
// filesystem that lies.
func TestMountPointFromBufferRejectsWhatIsNotAJunction(t *testing.T) {
	sound := mountPointBuffer(windows.IO_REPARSE_TAG_MOUNT_POINT, `\??\C:\dir\`)

	if got, err := mountPointFromBuffer(sound, "link"); err != nil || got != `\??\C:\dir\` {
		t.Fatalf("mountPointFromBuffer = %q, %v; want the stored target and nil", got, err)
	}

	symbolic := mountPointBuffer(windows.IO_REPARSE_TAG_SYMLINK, `\??\C:\dir\`)
	if got, err := mountPointFromBuffer(symbolic, "link"); err != nil || got != "" {
		t.Fatalf("a symbolic link = %q, %v; want empty and nil", got, err)
	}

	if got, err := mountPointFromBuffer(sound[:mountPointHeaderSize-1], "link"); err != nil || got != "" {
		t.Fatalf("a truncated answer = %q, %v; want empty and nil", got, err)
	}

	lying := mountPointBuffer(windows.IO_REPARSE_TAG_MOUNT_POINT, `\??\C:\dir\`)
	put16(lying[10:], uint16(len(lying)))
	if _, err := mountPointFromBuffer(lying, "link"); err == nil {
		t.Fatal("a target reaching past the data was accepted")
	}
}

// A NUL byte cannot be encoded for Windows, so the encoding steps refuse it
// before any system call runs. Create and Target stop such a path earlier, at
// os.Stat and os.Lstat, which is why the two helpers are called directly.
func TestSetMountPointRefusesANulByte(t *testing.T) {
	root := t.TempDir()
	if err := setMountPoint(filepath.Join(root, "link"), "C:\\bad\x00target"); err == nil {
		t.Fatal("setMountPoint encoded a target with a NUL byte")
	}
	if err := setMountPoint(filepath.Join(root, "bad\x00link"), root); err == nil {
		t.Fatal("setMountPoint encoded a link with a NUL byte")
	}
}

func TestReparseTargetRefusesANulByte(t *testing.T) {
	if _, err := reparseTarget(filepath.Join(t.TempDir(), "bad\x00link")); err == nil {
		t.Fatal("reparseTarget encoded a link with a NUL byte")
	}
}

// The claim in Create's docstring, tested rather than asserted: a failed ioctl
// must take the empty directory back out, because `Target` would read one as
// "not a link" and the next run would try again.
func TestCreateLeavesNoEmptyDirectoryBehindWhenTheIoctlFails(t *testing.T) {
	root := t.TempDir()
	// A reparse point holds at most MAXIMUM_REPARSE_DATA_BUFFER_SIZE bytes, so
	// a target too long to store is the one way to make the ioctl refuse
	// without breaking anything else.
	deep := root
	for len(deep) < windows.MAXIMUM_REPARSE_DATA_BUFFER_SIZE/2 {
		deep = filepath.Join(deep, strings.Repeat("d", 200))
	}
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Skip("this filesystem will not carry a path that long:", err)
	}
	link := filepath.Join(root, "link")

	if err := Create(link, deep); err == nil {
		t.Fatal("Create accepted a target too long to store")
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("Create left something behind: %v", err)
	}
}
