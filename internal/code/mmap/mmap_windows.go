package mmap

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// mapFile maps all of f read-only. The mapping object is closed right away:
// the view keeps it alive until UnmapViewOfFile.
//
//coverage:exempt the Stat, CreateFileMapping and MapViewOfFile error arms need the kernel to refuse an open regular file; the empty and the mapped path are covered
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
	h, err := windows.CreateFileMapping(windows.Handle(f.Fd()), nil, windows.PAGE_READONLY, 0, 0, nil)
	if err != nil {
		return nil, nil, err
	}
	defer windows.CloseHandle(h)
	addr, err := windows.MapViewOfFile(h, windows.FILE_MAP_READ, 0, 0, 0)
	if err != nil {
		return nil, nil, err
	}
	// The view lives outside the Go heap, so the address cannot move; reading
	// it through a pointer to the uintptr keeps vet's unsafeptr check quiet.
	data := unsafe.Slice(*(**byte)(unsafe.Pointer(&addr)), size)
	return data, func() error { return windows.UnmapViewOfFile(addr) }, nil
}
