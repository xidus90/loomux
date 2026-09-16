package privacy

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Contained verifies that relative stays inside the area of scope and returns
// it the way `PurePosixPath(...).as_posix()` spells it. It is `_contained`
// (src/brain/core.py:649-668), checked lexically and before anything touches
// the filesystem, "so a refusal cannot be told apart from a path that simply
// is not there".
//
// Backslashes become slashes first. The path is refused when it has a root,
// when PureWindowsPath finds a drive in it, or when a `..` component remains.
// Python 3.14 builds the PureWindowsPath from the cleaned posix form
// (`PurePath.__init__` takes `as_posix()` of a path of the other flavour), so
// the drive is asked of the cleaned path: `./1:foo.md` is refused like
// `1:foo.md`. A drive is any one character before `:` -- measured, `é:x.md`
// is refused, `:x.md` and `ab:c.md` pass. The empty path and `.` are not
// refused; they clean to `.`, as in Python.
func Contained(scope, relative string) (string, error) {
	root, parts := posixParts(strings.ReplaceAll(relative, `\`, "/"))
	cleaned := strings.Join(parts, "/")
	if root != "" || hasDrive(cleaned) {
		return "", fmt.Errorf("%s/%s leaves the area", scope, relative)
	}
	for _, part := range parts {
		if part == ".." {
			return "", fmt.Errorf("%s/%s leaves the area", scope, relative)
		}
	}
	if cleaned == "" {
		return ".", nil
	}
	return cleaned, nil
}

// hasDrive is `PureWindowsPath(p).drive != ""` for a path without a root:
// one character, counted in runes as pathlib counts code points, then `:`.
func hasDrive(p string) bool {
	_, size := utf8.DecodeRuneInString(p)
	return size > 0 && len(p) > size && p[size] == ':'
}
