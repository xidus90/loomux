package selfupdate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestReadStatusIsNilWithoutAFile(t *testing.T) {
	st, err := ReadStatus(t.TempDir())
	if st != nil || err != nil {
		t.Fatalf("ReadStatus = %v, %v; want nil, nil", st, err)
	}
}

func TestStatusRoundTrips(t *testing.T) {
	dir := t.TempDir()
	want := Status{
		CheckedAt:  time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC),
		Executable: `C:\loomux.exe`,
		Running:    "2.7.0",
		Result:     Updated,
		Version:    "2.8.0",
	}
	if err := WriteStatus(dir, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadStatus(dir)
	if err != nil || got == nil || *got != want {
		t.Fatalf("ReadStatus = %+v, %v; want %+v", got, err, want)
	}
	data, _ := os.ReadFile(StatusPath(dir))
	if !strings.Contains(string(data), `"result": "updated"`) {
		t.Fatalf("update.json does not carry the spec's field names:\n%s", data)
	}
}

func TestReadStatusRefusesBrokenJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(StatusPath(dir), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadStatus(dir); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("err = %v, want a parse error", err)
	}
}

func TestReadStatusReportsAnUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(StatusPath(dir), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadStatus(dir); err == nil || !strings.Contains(err.Error(), "read") {
		t.Fatalf("err = %v, want a read error", err)
	}
}

func TestWriteStatusFailsWhenTheStateDirectoryIsAFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "state")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteStatus(file, Status{}); err == nil {
		t.Fatal("WriteStatus into a file succeeded")
	}
}

// JSON holds a time only up to year 9999; a status beyond it is an error, not
// an empty update.json.
func TestWriteStatusRefusesATimeJSONCannotHold(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	err := WriteStatus(dir, Status{CheckedAt: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err == nil || !strings.Contains(err.Error(), "encode") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("an unencodable status left %s behind: %v", dir, err)
	}
}

func TestIsCanonical(t *testing.T) {
	dir := t.TempDir()
	canonical := Canonical(dir)
	if IsCanonical(canonical, dir) {
		t.Fatal("a canonical path that does not exist counts as canonical")
	}
	other := filepath.Join(dir, "other.exe")
	if err := os.WriteFile(other, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if IsCanonical(other, dir) {
		t.Fatal("an existing file counts as canonical while no canonical binary exists")
	}
	if err := os.MkdirAll(filepath.Dir(canonical), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(canonical, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsCanonical(canonical, dir) {
		t.Fatal("the canonical binary is not canonical")
	}
	if IsCanonical(other, dir) {
		t.Fatal("another file counts as canonical")
	}
	if IsCanonical(filepath.Join(dir, "missing.exe"), dir) {
		t.Fatal("a missing file counts as canonical")
	}
	if runtime.GOOS == "windows" && !IsCanonical(strings.ToUpper(canonical), dir) {
		t.Fatal("the same file in other letter case is not canonical")
	}
}
