//go:build windows

package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// 8.3 aliases exist on Windows alone, so the real one is asked here; the
// rule itself is measured on every platform through the resolver seam in
// place_test.go.

// shortAlias is the 8.3 alias the volume keeps for path, through a call the
// barrier does not use itself, so the fixture does not lean on the code it
// tests. The test is skipped where the volume keeps no alias.
func shortAlias(t *testing.T, path string) string {
	t.Helper()
	wide, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]uint16, windows.MAX_PATH)
	length, err := windows.GetShortPathName(wide, &buffer[0], uint32(len(buffer)))
	if err != nil || length == 0 || int(length) > len(buffer) {
		t.Skipf("no short name for %q: %v", path, err)
	}
	short := windows.UTF16ToString(buffer[:length])
	if strings.EqualFold(filepath.Base(short), filepath.Base(path)) {
		t.Skip("this volume keeps no 8.3 alias, so there is no alias to refuse")
	}
	return short
}

// test_a_short_name_alias_of_the_register_is_refused (test_apply.py:1532),
// against the file system rather than a model of it: `_IDENT~1.TSV` opens
// the register, adds no directory entry and is no link.
func TestGateRefusesARealShortNameOfTheRegister(t *testing.T) {
	p, vault := newPlace(t)
	register := filepath.Join(vault, "wiki", "_identities.tsv")
	writeFile(t, register, "standing\n")
	alias := filepath.Join(vault, "wiki", filepath.Base(shortAlias(t, register)))
	refused(t, p.write(alias, "PWN"), alias+": a scaffold file, not a page a case may change")
	if got := readFile(t, register); got != "standing\n" {
		t.Fatalf("the register was written through its alias: %q", got)
	}
	if _, err := os.Stat(alias); err != nil {
		t.Fatalf("the alias no longer opens the register: %v", err)
	}
}
