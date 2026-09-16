package pytext

import "testing"

func TestPathStringOnWindowsLikePureWindowsPath(t *testing.T) {
	// print(ascii(str(PureWindowsPath(s)))) on Python 3.14.7.
	cases := []struct{ in, want string }{
		{"C:/a/b", `C:\a\b`},
		{"C:/a//b/./c/", `C:\a\b\c`},
		{"C:/a/../b", `C:\a\..\b`},
		{"//server/share/x", `\\server\share\x`},
		{"//server/share", `\\server\share\`},
		{"//server/share/", `\\server\share\`},
		{"//server", `\\server`},
		{"//server/", `\\server\`},
		{"", `.`},
		{".", `.`},
		{"./", `.`},
		{"a/b", `a\b`},
		{"/a", `\a`},
		{`\a`, `\a`},
		{"C:", `C:`},
		{"C:a/b", `C:a\b`},
		{"C:/", `C:\`},
		{"c:/x", `c:\x`},
		{"1:/x", `1:\x`},
		{"ab:/c", `ab:\c`},
		{"//?/C:/x", `\\?\C:\x`},
		{"//./dev/x", `\\.\dev\x`},
		{"a/./b/", `a\b`},
		{"..", `..`},
		{"C:/Users/micro/Documents/#GIT/loomux", `C:\Users\micro\Documents\#GIT\loomux`},
		{"./a:b", `.\a:b`},
		{"///a", `\\\a`},
		{`\\?\UNC\server\share\x`, `\\?\UNC\server\share\x`},
		{"\u00e4:/x", "\u00e4:\\x"},
		{`a\\b`, `a\b`},
		{"C:.", `C:`},
		{"C:./a", `C:a`},
		{"/", `\`},
		{"//", `\\`},
		{"///", `\\\`},
		{`C:\\x`, `C:\x`},
	}
	for _, c := range cases {
		if got := PathString("windows", c.in); got != c.want {
			t.Errorf("PathString(windows, %q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPathStringElsewhereLikePurePosixPath(t *testing.T) {
	// print(ascii(str(PurePosixPath(s)))) on Python 3.14.7.
	cases := []struct{ in, want string }{
		{"C:/a/b", "C:/a/b"},
		{"C:/a//b/./c/", "C:/a/b/c"},
		{"a/../b", "a/../b"},
		{"//server/share/x", "//server/share/x"},
		{"///a", "/a"},
		{"/", "/"},
		{"", "."},
		{".", "."},
		{"a/", "a"},
		{"./a", "a"},
		{`a\b`, `a\b`},
		{"//", "//"},
		{"./a:b", "a:b"},
		{"a/./b/.", "a/b"},
	}
	for _, c := range cases {
		if got := PathString("linux", c.in); got != c.want {
			t.Errorf("PathString(linux, %q) = %q, want %q", c.in, got, c.want)
		}
	}
}
