package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// ReadAreaDeclaration reads the declaration of the area whose manifest lies in
// dir: `.loomux/config.toml`, checked whole by ReadDeclaration.
//
// Two answers are no defect of the file, and callers tell them apart: no
// regular file of that name, or none a path through a regular file can reach,
// is ErrNoManifest, a file without [area] -- policy only -- is ErrNoArea. A
// path the system refuses to inspect (access denied on a parent, a malformed
// name) is neither: it may hold a declaration, so it is an error and never an
// absence.
func ReadAreaDeclaration(dir string) (*Manifest, error) {
	name := filepath.Join(".loomux", "config.toml")
	path := filepath.Join(dir, name)
	info, err := os.Stat(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
		return nil, fmt.Errorf("%s: cannot be read: %w", path, err)
	}
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: %w (%s)", dir, ErrNoManifest, name)
	}
	return ReadDeclaration(path)
}

// IsUndeclared says that err is one of the two answers of ReadAreaDeclaration
// that mean "this directory declares no area": no manifest, or one without
// [area]. Every other error is a declaration that is there and does not read.
func IsUndeclared(err error) bool {
	return errors.Is(err, ErrNoManifest) || errors.Is(err, ErrNoArea)
}
