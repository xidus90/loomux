package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ReadAreaDeclaration reads the declaration of the area whose manifest lies in
// dir: `.loomux/config.toml`, checked whole by ReadDeclaration.
//
// Two answers are no defect of the file, and callers tell them apart: no
// regular file of that name is ErrNoManifest, a file without [area] -- policy
// only -- is ErrNoArea.
func ReadAreaDeclaration(dir string) (*Manifest, error) {
	name := filepath.Join(".loomux", "config.toml")
	path := filepath.Join(dir, name)
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
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
