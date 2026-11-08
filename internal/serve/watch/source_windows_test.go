package watch

import (
	"encoding/binary"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

// A buffer too small for a single entry is what a burst larger than the real
// buffer looks like: the system returns no bytes, and the source must say
// that changes were lost.
func TestSourceReportsAnOverflow(t *testing.T) {
	saved := bufferSize
	bufferSize = 4
	t.Cleanup(func() { bufferSize = saved })
	root := t.TempDir()
	src := newSource(t)
	if err := src.Add(root, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	write(t, filepath.Join(root, "a.go"))
	waitFor(t, src, func(e Event) bool { return e.Root == root && e.Overflow })
}

func TestSourceRefusesARootWithANul(t *testing.T) {
	src := newSource(t)
	if err := src.Add("bad\x00root", nil); err == nil {
		t.Fatal("Add of a root with a NUL succeeded")
	}
}

// notifyRecord encodes one FILE_NOTIFY_INFORMATION entry, padded to a DWORD.
func notifyRecord(name string, last bool) []byte {
	units := utf16.Encode([]rune(name))
	size := 12 + 2*len(units)
	size += (4 - size%4) % 4
	b := make([]byte, size)
	if !last {
		binary.LittleEndian.PutUint32(b[0:], uint32(size))
	}
	binary.LittleEndian.PutUint32(b[4:], windows.FILE_ACTION_MODIFIED)
	binary.LittleEndian.PutUint32(b[8:], uint32(2*len(units)))
	for i, u := range units {
		binary.LittleEndian.PutUint16(b[12+2*i:], u)
	}
	return b
}

func TestDecodeFollowsTheEntryChain(t *testing.T) {
	buf := append(notifyRecord(`pkg\a.go`, false), notifyRecord("b.go", true)...)
	got, ok := decode(`C:\r`, `C:\d`, buf, uint32(len(buf)), nil)
	want := []Event{{Root: `C:\r`, Path: `C:\d\pkg\a.go`}, {Root: `C:\r`, Path: `C:\d\b.go`}}
	if !ok || len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("decode = %v, %v; want %v, true", got, ok, want)
	}
}

func TestDecodeTurnsALostBufferIntoAnOverflow(t *testing.T) {
	for _, tc := range []struct {
		name string
		n    uint32
		err  error
	}{
		{"no bytes", 0, nil},
		{"enum dir", 0, windows.ERROR_NOTIFY_ENUM_DIR},
	} {
		got, ok := decode(`C:\r`, `C:\r`, make([]byte, 16), tc.n, tc.err)
		if !ok || len(got) != 1 || got[0] != (Event{Root: `C:\r`, Overflow: true}) {
			t.Errorf("%s: decode = %v, %v; want one overflow", tc.name, got, ok)
		}
	}
}

// A read that cannot be issued again ends the watch with an error. The
// completion event is set up front over an invalid handle, so the first wait
// reads as a lost buffer and the next start fails.
func TestRunReportsAReadThatCannotStartAgain(t *testing.T) {
	done, err := windows.CreateEvent(nil, 1, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	quit, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	w := &dirWatch{dir: `C:\r`, h: windows.InvalidHandle, quit: quit, buf: make([]byte, 64)}
	w.ov.HEvent = done
	events := make(chan Event)
	ended := make(chan struct{})
	go func() {
		w.run(`C:\r`, events)
		close(ended)
	}()
	if e := <-events; !e.Overflow {
		t.Fatalf("first event %+v, want the overflow of an empty read", e)
	}
	if e := <-events; e.Err == nil || e.Root != `C:\r` {
		t.Fatalf("second event %+v, want the root's error", e)
	}
	windows.SetEvent(quit)
	<-ended
}

func TestDecodeEndsQuietlyOnAQuit(t *testing.T) {
	got, ok := decode(`C:\r`, `C:\r`, nil, 0, windows.ERROR_OPERATION_ABORTED)
	if ok || got != nil {
		t.Fatalf("decode = %v, %v; want nothing and the end", got, ok)
	}
}

func TestDecodeReportsAFailedRead(t *testing.T) {
	got, ok := decode(`C:\r`, `C:\r`, nil, 0, windows.ERROR_NETNAME_DELETED)
	if ok || len(got) != 1 || got[0].Root != `C:\r` || got[0].Err != windows.ERROR_NETNAME_DELETED {
		t.Fatalf("decode = %v, %v; want one error event for the root and the end", got, ok)
	}
}
