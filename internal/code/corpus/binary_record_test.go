package corpus

import (
	"bytes"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

// mirrored is a small value and a JSON file beside it, the way a record and
// its mirror lie next to each other.
type mirrored struct {
	Name  string
	Sizes map[string]int64
}

func mirrorFixture(t *testing.T) (string, mirrored) {
	t.Helper()
	name := filepath.Join(t.TempDir(), "record.json")
	if err := os.WriteFile(name, []byte("{\"canonical\":true}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return name, mirrored{Name: "a", Sizes: map[string]int64{"x": 1}}
}

// writeMirrorFile stores header and body as a mirror would, for the tests that
// need a mirror this code did not write.
func writeMirrorFile(t *testing.T, name string, header mirrorHeader, body any) {
	t.Helper()
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := errors.Join(enc.Encode(header), enc.Encode(body)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mirrorPath(name), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func headerOf(t *testing.T, name string) mirrorHeader {
	t.Helper()
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	return mirrorHeader{Version: mirrorVersion, Size: info.Size(), MTimeNS: info.ModTime().UnixNano()}
}

func TestReadMirrorRoundTrip(t *testing.T) {
	name, want := mirrorFixture(t)
	if err := WriteMirror(name, want); err != nil {
		t.Fatalf("WriteMirror: %v", err)
	}
	var got mirrored
	if !ReadMirror(name, &got) || !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadMirror = %+v, want %+v", got, want)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(name), "*.tmp")); len(left) != 0 {
		t.Fatalf("temporaries left behind: %v", left)
	}
}

func TestReadMirrorRefusesAChangedJSON(t *testing.T) {
	name, want := mirrorFixture(t)
	if err := WriteMirror(name, want); err != nil {
		t.Fatal(err)
	}
	var got mirrored

	// Same size, later mtime.
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(name, later, later); err != nil {
		t.Fatal(err)
	}
	if ReadMirror(name, &got) {
		t.Fatal("ReadMirror accepted a JSON file whose mtime moved")
	}

	// Rewritten with another size: the mirror of the old bytes is stale.
	if err := WriteMirror(name, want); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte("{\"canonical\":false,\"more\":1}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ReadMirror(name, &got) {
		t.Fatal("ReadMirror accepted a JSON file whose size changed")
	}
}

func TestReadMirrorRefusesAnotherVersion(t *testing.T) {
	name, want := mirrorFixture(t)
	header := headerOf(t, name)
	header.Version++
	writeMirrorFile(t, name, header, want)

	var got mirrored
	if ReadMirror(name, &got) {
		t.Fatal("ReadMirror accepted a mirror of another version")
	}
}

func TestReadMirrorRefusesGarbage(t *testing.T) {
	name, _ := mirrorFixture(t)
	var got mirrored
	if ReadMirror(name, &got) {
		t.Fatal("ReadMirror accepted a missing mirror")
	}
	if err := os.WriteFile(mirrorPath(name), []byte("not gob at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ReadMirror(name, &got) {
		t.Fatal("ReadMirror accepted garbage")
	}
	// A valid header over a body of the wrong shape.
	writeMirrorFile(t, name, headerOf(t, name), "a string, not a struct")
	if ReadMirror(name, &got) {
		t.Fatalf("ReadMirror accepted a body of the wrong shape: %+v", got)
	}
}

func TestReadMirrorWithoutJSON(t *testing.T) {
	var got mirrored
	if ReadMirror(filepath.Join(t.TempDir(), "gone.json"), &got) {
		t.Fatal("ReadMirror accepted a mirror whose JSON file is missing")
	}
}

func TestWriteMirrorReportsWhatItCannotWrite(t *testing.T) {
	name, want := mirrorFixture(t)

	// No JSON file to bind to.
	if err := WriteMirror(filepath.Join(t.TempDir(), "gone.json"), want); err == nil {
		t.Fatal("WriteMirror bound a mirror to a missing JSON file")
	}
	// A value gob cannot encode.
	if err := WriteMirror(name, make(chan int)); err == nil {
		t.Fatal("WriteMirror encoded a channel")
	}
	// A directory under the temporary's name: the write cannot open it.
	tmp := tempName(mirrorPath(name))
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteMirror(name, want); err == nil {
		t.Fatal("WriteMirror wrote through a directory")
	}
	if err := os.RemoveAll(tmp); err != nil {
		t.Fatal(err)
	}
	// A directory in the mirror's place refuses the swap on every platform.
	if err := os.MkdirAll(mirrorPath(name), 0o755); err != nil {
		t.Fatal(err)
	}
	old := lock.SidecarBudget
	lock.SidecarBudget = 50 * time.Millisecond
	t.Cleanup(func() { lock.SidecarBudget = old })
	if err := WriteMirror(name, want); err == nil {
		t.Fatal("WriteMirror replaced a directory")
	}
	if _, err := os.Stat(tmp); err == nil {
		t.Fatal("the failed swap left its temporary behind")
	}
}
