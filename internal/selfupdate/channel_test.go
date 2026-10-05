package selfupdate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadChannel(t *testing.T) {
	for _, c := range []struct {
		name, body string
		write      bool
		beta       bool
		err        string
	}{
		{"missing", "", false, false, ""},
		{"beta", "beta", true, true, ""},
		{"beta with newline", "beta\n", true, true, ""},
		{"beta with CRLF", "beta\r\n", true, true, ""},
		{"foreign", "nightly\n", true, false, `channel file holds "nightly", not beta`},
	} {
		dir := t.TempDir()
		if c.write {
			if err := os.WriteFile(ChannelPath(dir), []byte(c.body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		beta, err := ReadChannel(dir)
		if beta != c.beta || (err == nil) != (c.err == "") || (err != nil && !strings.Contains(err.Error(), c.err)) {
			t.Errorf("%s: ReadChannel = %v, %v; want %v, %q", c.name, beta, err, c.beta, c.err)
		}
	}
}

func TestReadChannelReportsAnUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(ChannelPath(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if beta, err := ReadChannel(dir); beta || err == nil || !strings.Contains(err.Error(), "read channel file") {
		t.Fatalf("ReadChannel on a directory = %v, %v", beta, err)
	}
}

func TestWriteChannelSetsAndClears(t *testing.T) {
	dir := t.TempDir()
	if err := WriteChannel(dir, false); err != nil {
		t.Fatalf("clearing a missing marker: %v", err)
	}
	if err := WriteChannel(dir, true); err != nil {
		t.Fatal(err)
	}
	if beta, err := ReadChannel(dir); !beta || err != nil {
		t.Fatalf("after set: %v, %v", beta, err)
	}
	if err := WriteChannel(dir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ChannelPath(dir)); !os.IsNotExist(err) {
		t.Fatalf("marker still there: %v", err)
	}
}

func TestWriteChannelFailsWhereItCannotWrite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	if err := WriteChannel(dir, true); err == nil {
		t.Fatal("wrote into a missing directory")
	}
	full := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ChannelPath(full), "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteChannel(full, false); err == nil {
		t.Fatal("removed a non-empty directory")
	}
}
