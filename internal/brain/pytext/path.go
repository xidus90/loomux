package pytext

import (
	"strings"
	"unicode/utf8"
)

// PathString is `str(Path(p))` on the platform goos names: PureWindowsPath
// for "windows", PurePosixPath for every other. Taking goos as an argument
// keeps both flavours testable on either system.
//
// Both follow `PurePath._parse_path` and `_format_parsed_parts` of Python
// 3.14 (pathlib/_local.py): split off drive and root, drop empty and `.`
// parts, keep `..`, join with the separator, and answer `.` for nothing.
func PathString(goos, p string) string {
	if goos == "windows" {
		return windowsPath(p)
	}
	return posixPath(p)
}

// posixPath keeps exactly two leading slashes, as `posixpath.splitroot`
// does, and collapses one or three and more to one.
func posixPath(p string) string {
	root, rel := "", p
	switch {
	case !strings.HasPrefix(p, "/"):
	case strings.HasPrefix(p, "//") && !strings.HasPrefix(p, "///"):
		root, rel = "//", p[2:]
	default:
		root, rel = "/", p[1:]
	}
	return orDot(root + strings.Join(pathParts(rel, "/"), "/"))
}

// windowsPath turns slashes into backslashes first, then reads drive and
// root the way `_parse_path` does: a UNC drive of exactly four parts whose
// third is neither empty, `?` nor `.` gains a root (`//server/share` is
// `\\server\share\`), and so does one of six parts. A relative first part
// that reads as a drive gets `.` in front.
func windowsPath(p string) string {
	p = strings.ReplaceAll(p, "/", `\`)
	drive, root, rel := windowsSplitRoot(p)
	if root == "" && strings.HasPrefix(drive, `\`) && !strings.HasSuffix(drive, `\`) {
		driveParts := strings.Split(drive, `\`)
		// `drv_parts[2] not in '?.'` is a substring test, so the empty part
		// counts as in it: `///a` keeps no root.
		if len(driveParts) == 4 && !strings.Contains("?.", driveParts[2]) || len(driveParts) == 6 {
			root = `\`
		}
	}
	tail := pathParts(rel, `\`)
	if drive == "" && root == "" && len(tail) > 0 {
		if first, _, _ := windowsSplitRoot(tail[0]); first != "" {
			tail = append([]string{"."}, tail...)
		}
	}
	return orDot(drive + root + strings.Join(tail, `\`))
}

// windowsSplitRoot is the pure-Python `ntpath.splitroot` of 3.14 on a path
// that holds backslashes only. A drive letter is any one character before
// `:`, counted in characters, not bytes.
func windowsSplitRoot(p string) (drive, root, rel string) {
	if strings.HasPrefix(p, `\`) {
		if !strings.HasPrefix(p, `\\`) {
			return "", `\`, p[1:]
		}
		start := 2
		if len(p) >= 8 && strings.EqualFold(p[:8], `\\?\UNC\`) {
			start = 8
		}
		index := strings.IndexByte(p[start:], '\\')
		if index < 0 {
			return p, "", ""
		}
		index += start
		index2 := strings.IndexByte(p[index+1:], '\\')
		if index2 < 0 {
			return p, "", ""
		}
		index2 += index + 1
		return p[:index2], `\`, p[index2+1:]
	}
	if _, size := utf8.DecodeRuneInString(p); len(p) > size && p[size] == ':' {
		if len(p) > size+1 && p[size+1] == '\\' {
			return p[:size+1], `\`, p[size+2:]
		}
		return p[:size+1], "", p[size+1:]
	}
	return "", "", p
}

// pathParts are the parts of rel between separators, without the empty ones
// and without `.`.
func pathParts(rel, sep string) []string {
	var parts []string
	for _, part := range strings.Split(rel, sep) {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	return parts
}

// orDot is `... or '.'` of `PurePath.__str__`.
func orDot(s string) string {
	if s == "" {
		return "."
	}
	return s
}
