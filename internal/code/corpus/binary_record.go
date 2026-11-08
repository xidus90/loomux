package corpus

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/xidus90/loomux/internal/lock"
)

// mirrorVersion is the shape of the mirror's header. A mirror of another
// version is one this code cannot read, which is the same as none.
const mirrorVersion = 1

// mirrorHeader binds a mirror to the JSON file it was written from: the JSON
// file stays canonical, and a mirror is only as good as its match.
type mirrorHeader struct {
	Version int
	Size    int64
	MTimeNS int64
}

// mirrorPath is where the binary form of a JSON file lives.
func mirrorPath(jsonPath string) string { return jsonPath + ".bin" }

// tempName is the temporary a file is written to before it is swapped in, with
// the pid in the name the way store.Write does.
func tempName(path string) string { return path + "." + strconv.Itoa(os.Getpid()) + ".tmp" }

// replaceFile stores body beside path and swaps it in whole, so that a reader
// never meets a half-written file.
func replaceFile(path string, body []byte) error {
	tmp := tempName(path)
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := lock.ReplaceEventually(tmp, path, lock.SidecarBudget); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// ErrMirror marks the failure to write a mirror, which leaves the JSON file it
// shadows in place: the record stands, only the fast path is lost.
var ErrMirror = errors.New("binary mirror not written")

// WriteMirror stores v in gob form beside jsonPath, bound to the JSON file's
// size and modification time as they are now. Call it after the JSON file is
// in place. Every failure wraps ErrMirror.
func WriteMirror(jsonPath string, v any) error {
	if err := writeMirror(jsonPath, v); err != nil {
		return fmt.Errorf("%w: %w", ErrMirror, err)
	}
	return nil
}

func writeMirror(jsonPath string, v any) error {
	info, err := os.Stat(jsonPath)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	header := mirrorHeader{Version: mirrorVersion, Size: info.Size(), MTimeNS: info.ModTime().UnixNano()}
	if err := errors.Join(enc.Encode(header), enc.Encode(v)); err != nil {
		return err
	}
	return replaceFile(mirrorPath(jsonPath), buf.Bytes())
}

// ReadMirror decodes the mirror of jsonPath into v and reports whether it
// could. It does not when the mirror is missing, of another version, broken, or
// written from a JSON file other than the one there now. A false result may
// leave v half filled: the caller reads the JSON file into a value of its own.
func ReadMirror(jsonPath string, v any) bool {
	info, err := os.Stat(jsonPath)
	if err != nil {
		return false
	}
	b, err := readFile(mirrorPath(jsonPath))
	if err != nil {
		return false
	}
	dec := gob.NewDecoder(bytes.NewReader(b))
	var header mirrorHeader
	want := mirrorHeader{Version: mirrorVersion, Size: info.Size(), MTimeNS: info.ModTime().UnixNano()}
	if err := dec.Decode(&header); err != nil || header != want {
		return false
	}
	return dec.Decode(v) == nil
}
