//go:build !windows

package mmap

import (
	"os"

	"golang.org/x/sys/unix"
)

// mapFile maps all of f read-only and shared, so the bytes are the file's.
//
//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func mapFile(f *os.File) ([]byte, func() error, error) {
	st, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, nil, errNotRegular
	}
	size := int(st.Size())
	if size == 0 {
		return []byte{}, nil, nil
	}
	data, err := unix.Mmap(int(f.Fd()), 0, size, unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return nil, nil, err
	}
	return data, func() error { return unix.Munmap(data) }, nil
}
