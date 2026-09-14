//go:build !windows

package junction

// Create cannot do on another platform what it must do on this one, and says
// so rather than reaching for the symlink that would almost work.
func Create(link, target string) error {
	return ErrUnsupported
}

// reparseTarget finds nothing, because there is nothing of this kind to find:
// a junction is a Windows reparse point, so "not a junction" is the true
// answer about every path here, not a gap in the implementation.
func reparseTarget(link string) (string, error) {
	return "", nil
}
