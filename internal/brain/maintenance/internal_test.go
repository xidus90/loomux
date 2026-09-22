package maintenance

import (
	"os"
	"path/filepath"
	"testing"
)

// A handle opened for reading refuses the write, which is the arm `appendLine`
// itself cannot reach: it opens the file it hands over. Windows answers
// "access is denied" and POSIX EBADF; both are an error, which is all this
// asks for.
func TestWriteAndCloseCarriesAFailedWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "read-only.tsv")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := writeAndClose(file, "a\tb\tc\n"); err == nil {
		t.Fatal("writeAndClose wrote through a read-only handle")
	}
}
