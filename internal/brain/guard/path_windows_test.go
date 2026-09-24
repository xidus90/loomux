//go:build windows

package guard

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
)

// The two spellings only Windows has, asked where only Windows compiles.
// They are here rather than behind a run-time skip because the fixture
// needs the alias itself, and the only honest way to read one is to ask
// the file system.

// shortName is the 8.3 alias the file system keeps for `path`, or "" if
// this volume keeps none. `GetShortPathName` is a different call from the
// one `ResolvePath` uses, so the fixture does not lean on the code it
// tests -- and `dir /x` prints the same aliases to anyone who can list
// the directory, which is why an agent needs no privilege to spell one.
func shortName(t *testing.T, path string) string {
	t.Helper()
	wide, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]uint16, windows.MAX_PATH)
	length, err := windows.GetShortPathName(wide, &buffer[0],
		uint32(len(buffer)))
	if err != nil || length == 0 || int(length) > len(buffer) {
		t.Skipf("no short name for %q: %v", path, err)
	}
	short := windows.UTF16ToString(buffer[:length])
	if strings.EqualFold(short, path) {
		return ""
	}
	return short
}

// absentVolume is the root of a drive letter this machine has not
// assigned. No fixed letter can promise that: `Q:` was one, until another
// session's `subst` mapped exactly it. The mask is asked rather than each
// root stat'ed, because a reader with no medium in it fails the stat too
// while its letter is taken, and the walk would then meet "not ready"
// instead of the absent volume the test is about.
func absentVolume(t *testing.T) string {
	t.Helper()
	assigned, err := windows.GetLogicalDrives()
	if err != nil {
		t.Fatalf("GetLogicalDrives: %v", err)
	}
	volume, ok := unmappedDrive(assigned)
	if !ok {
		t.Skip("every drive letter from A: to Z: is assigned on this " +
			"machine, so no volume is certain to be absent")
	}
	return volume
}

func TestAShortNameNamesTheZoneItStandsFor(t *testing.T) {
	tmp := t.TempDir()
	state, zone := zoneWithALongName(t, tmp)
	short := shortName(t, zone)
	if short == "" {
		t.Skip("this volume keeps no 8.3 alias for the fixture")
	}
	// Measured against the real registration: `Path.resolve()` answers
	// the long name for a component that exists under its alias, so the
	// Python barrier sees the read-only zone. A barrier that compares
	// the alias against the spelled-out roots finds nothing and opens
	// the zone instead.
	deny(t, writeCall(filepath.Join(short, "x.md")), state, "read-only")
}

func TestAShortNameInTheMiddleNamesTheZoneAsWell(t *testing.T) {
	tmp := t.TempDir()
	state, zone := zoneWithALongName(t, tmp)
	mkdir(t, filepath.Join(zone, "knowledge"))
	short := shortName(t, zone)
	if short == "" {
		t.Skip("this volume keeps no 8.3 alias for the fixture")
	}
	// The alias need not be the last component: what stands below it is
	// spelt out and still has to land in the zone.
	deny(t, writeCall(filepath.Join(short, "knowledge", "x.md")), state,
		"read-only")
}

func TestTheDevicePrefixComesOffInBothItsSpellings(t *testing.T) {
	for _, row := range []struct{ given, want string }{
		{`\\?\C:\a\b`, `C:\a\b`},
		{`\\?\UNC\srv\share\a`, `\\srv\share\a`},
		{`C:\a`, `C:\a`},
	} {
		if got := withoutDevicePrefix(row.given); got != row.want {
			t.Errorf("withoutDevicePrefix(%q) = %q, want %q",
				row.given, got, row.want)
		}
	}
}

func TestOnlyAFileSystemErrorAboutAnAbsentPathShortensTheWalk(t *testing.T) {
	// The three answers of `stopsResolving`, and the middle one is why
	// it is a list of numbers rather than "any failure": 4390 says the
	// thing opened is no reparse point, which is a statement *about* a
	// path that is there, and a walk that shortened on it would decide
	// on a path it never resolved.
	if !stopsResolving(syscall.Errno(2)) {
		t.Error("a missing file should shorten the walk")
	}
	if stopsResolving(syscall.Errno(4390)) {
		t.Error("a reparse-point error should not shorten the walk")
	}
	if stopsResolving(errors.New("not the file system speaking")) {
		t.Error("an error that carries no errno should not shorten it")
	}
}

func TestANameTheFileSystemCannotEvenSpellIsRefused(t *testing.T) {
	// A zero byte inside a path is the one spelling `UTF16PtrFromString`
	// refuses outright, so the walk never opens anything and never learns
	// an errno it recognises. Python answers such a path unresolved; this
	// side refuses, which is the safe direction for a barrier whose whole
	// answer is where a write lands.
	if _, err := ResolvePath("C:\\a\x00b"); err == nil {
		t.Error("a path carrying a zero byte was resolved")
	}
}

func TestADeviceNameIsWalkedPastRatherThanOpened(t *testing.T) {
	// `CreateFile` opens `NUL` wherever it is spelt -- it is a device,
	// not a file in the directory named before it -- and then
	// `GetFinalPathNameByHandle` refuses the handle with "invalid
	// parameter". That number is in the list, so the walk shortens the
	// path and the device name comes back as an ordinary tail.
	tmp := longTempDir(t)
	target := filepath.Join(tmp, "NUL")
	got, err := ResolvePath(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(got, target) {
		t.Errorf("ResolvePath(%q) = %q", target, got)
	}
}

func TestAPathTooLongForTheFirstBufferIsAskedAgain(t *testing.T) {
	// `GetFinalPathNameByHandle` answers the size it needs when the
	// buffer is too small, and the loop has to grow and ask again rather
	// than truncate. MAX_PATH is the first guess, so the fixture has to
	// stand past it.
	tmp := longTempDir(t)
	deep := tmp
	for len(deep) < windows.MAX_PATH+40 {
		deep = filepath.Join(deep, strings.Repeat("d", 40))
	}
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Skipf("this machine will not build a long path: %v", err)
	}
	got, err := ResolvePath(deep)
	if err != nil {
		t.Skipf("this machine will not open a long path: %v", err)
	}
	if !strings.EqualFold(got, deep) {
		t.Errorf("ResolvePath of a long path = %q, want %q", got, deep)
	}
}

// longTempDir is t.TempDir() in its long spelling. Where TEMP is an 8.3 short
// path -- C:\Users\RUNNER~1 on a GitHub runner -- t.TempDir() hands out the
// short form, ResolvePath answers with the long one, and a comparison by text
// between the two fails. filepath.EvalSymlinks expands 8.3 names, measured on
// 2026-09-18.
func longTempDir(t *testing.T) string {
	t.Helper()
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return tmp
}
