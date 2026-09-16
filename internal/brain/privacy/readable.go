package privacy

import (
	"fmt"
	"regexp"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// MatchesGlobs reports whether relative matches any of the patterns. It is
// `matches_globs` (src/brain/privacy.py:31-74): `PurePosixPath(folded(relative))
// .full_match(folded(pattern))`, folded being NFC and then casefold, on both
// sides and on every platform. The docstring there gives the reason for both
// folds: a spelling variant must not walk around a `never` pattern wherever
// the filesystem opens the file anyway.
//
// No backslash becomes a slash here, because Python turns none: to
// PurePosixPath `secrets\key.txt` is one name. The callers hand in register
// paths and the output of Contained, both spelt with slashes.
//
// Each pattern is compiled on every call. Python caches the compiled form;
// the lists are a handful of globs, and a cache would be state this package
// keeps nowhere else.
func MatchesGlobs(patterns []string, relative string) bool {
	candidate := fullMatchForm(folded(relative))
	for _, pattern := range patterns {
		if regexp.MustCompile(translateGlob(fullMatchForm(folded(pattern)))).MatchString(candidate) {
			return true
		}
	}
	return false
}

// folded is `_folded` (src/brain/privacy.py:73-74).
func folded(text string) string {
	return pytext.CaseFold(pytext.NFC(text))
}

// IsReadable checks whether relative path is readable according to manifest never globs.
func IsReadable(manifest *config.Manifest, relative string) bool {
	if manifest == nil || len(manifest.NeverGlobs) == 0 {
		return true
	}
	return !MatchesGlobs(manifest.NeverGlobs, relative)
}

// ReviewExcludes returns the two patterns that keep the review centre from the
// cloud channel. It is `review_excludes` (src/brain/walk.py:96-113): the value
// is used as written, untrimmed, and refused only when
// `PurePosixPath(review).parts` is empty -- `.`, `./`, `./.` -- because such a
// value would turn the first pattern into a bare `**` and take the whole area
// out. `/` and `\` have parts and pass, as they do in Python.
//
// Python raises ValueError there, which the command line does not catch, so
// the reference ends in a traceback; loomux returns the same words as an error.
func ReviewExcludes(manifest *config.Manifest) ([]string, error) {
	if manifest == nil || manifest.LayoutReview == "" {
		return nil, nil
	}
	review := manifest.LayoutReview
	if root, parts := posixParts(review); root == "" && len(parts) == 0 {
		return nil, fmt.Errorf("[layout] review must not resolve to the area root, found %s", pytext.Repr(review))
	}
	return []string{review + "/**", review}, nil
}
