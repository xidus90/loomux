package watch

import (
	"errors"
	"path/filepath"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// bufferSize is the change buffer of one directory; 64 KiB is the most a
// network share accepts. A test shrinks it to provoke an overflow.
var bufferSize = 64 << 10

const notifyFilter = windows.FILE_NOTIFY_CHANGE_FILE_NAME | windows.FILE_NOTIFY_CHANGE_DIR_NAME |
	windows.FILE_NOTIFY_CHANGE_SIZE | windows.FILE_NOTIFY_CHANGE_LAST_WRITE

// winSource reads ReadDirectoryChangesW: one recursive handle per root, and
// one flat handle per directory of the root's extra files.
type winSource struct {
	events chan Event
	wg     sync.WaitGroup
	mu     sync.Mutex
	roots  map[string][]*dirWatch
}

// dirWatch is one directory handle and the goroutine reading it. The
// goroutine owns every handle: it closes them once quit is set, so a Remove
// never waits on a reader that is blocked sending to a Run busy in that very
// Remove.
type dirWatch struct {
	dir     string
	subtree bool
	h       windows.Handle
	quit    windows.Handle
	ov      windows.Overlapped
	buf     []byte
}

// NewSource returns the platform's change source.
func NewSource() (Source, error) {
	return &winSource{events: make(chan Event), roots: map[string][]*dirWatch{}}, nil
}

func (s *winSource) Events() <-chan Event { return s.events }

// Add watches root recursively and the directories of extra flatly. A
// directory of extra that does not exist is skipped: git creates info/ only
// when it needs it.
func (s *winSource) Add(root string, extra []string) error {
	w, err := open(root, true)
	if err != nil {
		return err
	}
	watches := []*dirWatch{w}
	seen := map[string]bool{root: true}
	for _, p := range extra {
		dir := filepath.Dir(p)
		if seen[dir] {
			continue
		}
		seen[dir] = true
		if w, err := open(dir, false); err == nil {
			watches = append(watches, w)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roots[root] = watches
	for _, w := range watches {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			w.run(root, s.events)
		}()
	}
	return nil
}

// Remove stops the root's readers without waiting for them.
func (s *winSource) Remove(root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.roots[root] {
		windows.SetEvent(w.quit)
	}
	delete(s.roots, root)
}

// Close stops every reader and returns once all handles are closed. Nobody
// reads Events any more, so Close drains it until the last reader is gone.
func (s *winSource) Close() error {
	s.mu.Lock()
	for root := range s.roots {
		for _, w := range s.roots[root] {
			windows.SetEvent(w.quit)
		}
	}
	s.roots = nil
	s.mu.Unlock()
	go func() {
		s.wg.Wait()
		close(s.events)
	}()
	for range s.events {
	}
	return nil
}

// open makes the handles of one directory. A failed CreateFile returns
// INVALID_HANDLE_VALUE, whose CloseHandle is a harmless no-op.
func open(dir string, subtree bool) (*dirWatch, error) {
	name, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(name, windows.FILE_LIST_DIRECTORY,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OVERLAPPED, 0)
	// Manual reset, so the wait in read does not consume the completion that
	// GetOverlappedResult waits for again.
	done, doneErr := windows.CreateEvent(nil, 1, 0, nil)
	quit, quitErr := windows.CreateEvent(nil, 1, 0, nil)
	w := &dirWatch{dir: dir, subtree: subtree, h: h, quit: quit, buf: make([]byte, bufferSize)}
	w.ov.HEvent = done
	// The first read is issued here, not by the goroutine: the system records
	// changes only from the first call on, and a write right after Add would
	// otherwise go unseen.
	err = errors.Join(err, doneErr, quitErr)
	if err == nil {
		err = w.start()
	}
	if err != nil {
		w.close()
		return nil, err
	}
	return w, nil
}

// run delivers the directory's changes until quit is set or reading fails,
// then waits for quit and closes the handles.
func (w *dirWatch) run(root string, events chan<- Event) {
	for more := true; more; {
		n, err := w.wait()
		var evs []Event
		evs, more = decode(root, w.dir, w.buf, n, err)
		for _, e := range evs {
			events <- e
		}
		if more {
			if err := w.start(); err != nil {
				events <- Event{Root: root, Err: err}
				more = false
			}
		}
	}
	windows.WaitForSingleObject(w.quit, windows.INFINITE)
	w.close()
}

// start issues one read; between reads the system keeps recording changes.
func (w *dirWatch) start() error {
	return windows.ReadDirectoryChanges(w.h, &w.buf[0], uint32(len(w.buf)), w.subtree, notifyFilter, nil, &w.ov, 0)
}

// wait waits for the pending read to complete or for quit. The cancel after
// the wait is a no-op when the read already completed, and turns a pending
// one into ERROR_OPERATION_ABORTED when quit woke the wait.
func (w *dirWatch) wait() (uint32, error) {
	var n uint32
	windows.WaitForMultipleObjects([]windows.Handle{w.ov.HEvent, w.quit}, false, windows.INFINITE)
	windows.CancelIoEx(w.h, &w.ov)
	err := windows.GetOverlappedResult(w.h, &w.ov, &n, true)
	return n, err
}

func (w *dirWatch) close() {
	windows.CloseHandle(w.h)
	windows.CloseHandle(w.ov.HEvent)
	windows.CloseHandle(w.quit)
}

// decode turns one completed read into events; more is false when the reader
// has to stop. No bytes or ERROR_NOTIFY_ENUM_DIR mean the system dropped
// changes, which only a check of the whole area makes good. An aborted read is
// the quit wait's own cancel; any other error, a deleted root for one, ends
// the watch of the area.
func decode(root, dir string, buf []byte, n uint32, err error) (evs []Event, more bool) {
	switch {
	case err == windows.ERROR_NOTIFY_ENUM_DIR || err == nil && n == 0:
		return []Event{{Root: root, Overflow: true}}, true
	case err == windows.ERROR_OPERATION_ABORTED:
		return nil, false
	case err != nil:
		return []Event{{Root: root, Err: err}}, false
	}
	for off := uint32(0); ; {
		info := (*windows.FileNotifyInformation)(unsafe.Pointer(&buf[off]))
		name := unsafe.Slice(&info.FileName, info.FileNameLength/2)
		evs = append(evs, Event{Root: root, Path: filepath.Join(dir, windows.UTF16ToString(name))})
		if info.NextEntryOffset == 0 {
			return evs, true
		}
		off += info.NextEntryOffset
	}
}
