package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// OldManifestNames are ultra-brain's two names for an area declaration, in
// the order its reader asked them. loomux reads neither; they are named so
// that `area check` can show what to carry over and ReadAreaDeclaration can
// point there. They go when `area check` goes.
func OldManifestNames() []string {
	return []string{filepath.Join(".ultra-brain", "config.toml"), ".brain.toml"}
}

// ReadAreaDeclaration reads the declaration of the area whose manifest lies in
// dir: `.loomux/config.toml`, checked whole by ReadDeclaration.
//
// Two answers are no defect of the file, and callers tell them apart: no
// regular file of that name is ErrNoManifest, a file without [area] -- policy
// only -- is ErrNoArea. Where an old manifest lies beside a missing one, the
// error says so and names `area check`, since a reader that ignores it
// silently would leave the area undeclared without a word.
func ReadAreaDeclaration(dir string) (*Manifest, error) {
	name := filepath.Join(".loomux", "config.toml")
	path := filepath.Join(dir, name)
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		for _, old := range OldManifestNames() {
			if info, err := os.Stat(filepath.Join(dir, old)); err == nil && info.Mode().IsRegular() {
				return nil, fmt.Errorf("%s: %w (%s); an old manifest lies there (%s): `loomux area check %s` shows what to carry over",
					dir, ErrNoManifest, name, old, dir)
			}
		}
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
