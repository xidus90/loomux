package pathkey

import (
	"path"
	"strings"
)

// Glob matches a slash-separated path against a glob pattern supporting
// `**`, and answers an error for a pattern it cannot read. A leading `**/`
// stands for any directory, the root included.
//
// It is `path.Match`, not `filepath.Match`: the path is slash-separated on
// every platform, and on Windows `filepath.Match` separates on `\` only, so a
// `*` there ran across a `/`.
//
// The error is not dropped, and that is the point. `config.ReadPolicy` refuses
// a malformed glob at load, but its check -- `path.Match(glob, "")` --
// stops at the first chunk that does not match an empty name, so a bad class in
// a later chunk (`foo/*[x`) still arrives here. Treating that as "no match"
// made the rule protect nothing without a word; the caller turns it into a
// refusal instead.
func Glob(pattern, name string) (bool, error) {
	// A leading **/ is any directory, the root included: the rest is tried
	// against the name and against every tail of it that starts after a
	// slash, so an element is matched whole.
	if rest, anywhere := strings.CutPrefix(pattern, "**/"); anywhere {
		for {
			if matched, err := Glob(rest, name); err != nil || matched {
				return matched, err
			}
			slash := strings.IndexByte(name, '/')
			if slash < 0 {
				return false, nil
			}
			name = name[slash+1:]
		}
	}
	if pattern == name {
		return true, nil
	}
	// Direct wildcard suffix like .aws/**
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		if name == prefix || strings.HasPrefix(name, prefix+"/") {
			return true, nil
		}
	}
	if strings.Contains(pattern, "/") {
		return path.Match(pattern, name)
	}
	// For patterns without slashes (e.g. *.pem or .env.* or uv.lock)
	// they match either at the root or base name depending on rule semantics
	return path.Match(pattern, path.Base(name))
}
