package verify

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
)

// CheckGoFormat walks given root paths and returns all unformatted .go files.
func CheckGoFormat(roots []string) ([]string, error) {
	var unformatted []string

	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat root %s: %w", root, err)
		}

		if !info.IsDir() {
			if filepath.Ext(root) == ".go" {
				dirty, err := isFileUnformatted(root)
				if err != nil {
					return nil, err
				}
				if dirty {
					unformatted = append(unformatted, root)
				}
			}
			continue
		}

		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if name == "vendor" || name == ".git" || name == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}

			dirty, err := isFileUnformatted(path)
			if err != nil {
				return err
			}
			if dirty {
				unformatted = append(unformatted, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk root %s: %w", root, err)
		}
	}

	return unformatted, nil
}

func isFileUnformatted(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read file %s: %w", path, err)
	}

	formatted, err := format.Source(content)
	if err != nil {
		return false, fmt.Errorf("format source %s: %w", path, err)
	}

	return !bytes.Equal(content, formatted), nil
}
