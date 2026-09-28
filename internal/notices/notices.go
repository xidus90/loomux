// Package notices carries NOTICE.md, the licenses of every third-party piece
// in the binary, for the release to ship beside it.
package notices

import _ "embed"

//go:embed NOTICE.md
var text string

// Text is NOTICE.md.
func Text() string { return text }
