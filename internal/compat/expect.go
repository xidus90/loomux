package compat

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Break is a change that makes one kind of check fail: the file is overwritten
// or created with Content.
type Break struct{ Kind, File, Content string }

// Expect is what a fixture promises, read from its expect.toml: the stacks and
// variants detect finds, the start of the command each kind plans, the graph
// edges (source, relation, target) and one Break per kind that can fail.
type Expect struct {
	Stacks, Variants []string
	Lanes            map[string]string
	Edges            [][3]string
	Breaks           []Break `toml:"break"`
}

// LoadExpect reads the expect.toml of a fixture.
func LoadExpect(fixture string) (Expect, error) {
	data, err := fs.ReadFile(Fixtures(), path.Join(fixture, "expect.toml"))
	if err != nil {
		return Expect{}, err
	}
	return decodeExpect(fixture, data)
}

func decodeExpect(fixture string, data []byte) (e Expect, err error) {
	if _, err := toml.Decode(string(data), &e); err != nil {
		return Expect{}, fmt.Errorf("%s/expect.toml: %w", fixture, err)
	}
	return e, nil
}

// CopyFixture writes the fixture project to dst. The module file go.mod.txt
// becomes go.mod there (see Fixtures).
func CopyFixture(name, dst string) error {
	fsys := Fixtures()
	return fs.WalkDir(fsys, name, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(name, filepath.FromSlash(p)) // p lies below name
		if rel == "go.mod.txt" {
			rel = "go.mod"
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, _ := fs.ReadFile(fsys, p) // the walk just listed it
		return os.WriteFile(target, data, 0o644)
	})
}
