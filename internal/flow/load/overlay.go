package load

import (
	"io/fs"
	"slices"
)

// overlayFS reads a bundled flow with some of its files taken from a project
// folder instead: the files Find checked the bundled flow has.
type overlayFS struct {
	base, top fs.FS
	files     []string
}

func (o overlayFS) Open(name string) (fs.File, error) {
	if slices.Contains(o.files, name) {
		return o.top.Open(name)
	}
	return o.base.Open(name)
}
