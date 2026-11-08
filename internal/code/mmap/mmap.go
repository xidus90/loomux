// Package mmap maps a file read-only into memory, so a reader pays only for
// the pages it touches instead of reading the whole file.
package mmap

import (
	"errors"
	"io/fs"
	"os"
)

// errNotRegular refuses a directory or a device: Windows opens a directory
// like a file, and its size of zero would pass as an empty file.
var errNotRegular = errors.New("not a regular file")

// File is a read-only mapping of a file. Its bytes are valid until Close.
type File struct {
	data  []byte
	unmap func() error
}

// Open maps the file at path. An empty file needs no mapping and yields an
// empty slice; a missing file is an error that matches fs.ErrNotExist.
func Open(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// The mapping keeps the file's contents alive; the handle is not needed.
	defer f.Close()
	data, unmap, err := mapFile(f)
	if err != nil {
		return nil, &fs.PathError{Op: "mmap", Path: path, Err: err}
	}
	return &File{data: data, unmap: unmap}, nil
}

// Bytes returns the mapped bytes, nil after Close. Writing to them faults.
func (m *File) Bytes() []byte { return m.data }

// Close releases the mapping. A second Close does nothing.
func (m *File) Close() error {
	unmap := m.unmap
	m.data, m.unmap = nil, nil
	if unmap == nil {
		return nil
	}
	return unmap()
}
