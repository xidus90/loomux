//go:build windows

package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessImageIsTheExecutableOfARunningProcess(t *testing.T) {
	want, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	got, err := processImage(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(filepath.Base(got), filepath.Base(want)) {
		t.Errorf("processImage = %q, want the file %q", got, want)
	}
}

func TestProcessImageOfAPIDThatIsNoProcessIsAnError(t *testing.T) {
	if got, err := processImage(0x7ffffff0); err == nil {
		t.Errorf("processImage = %q, want an error", got)
	}
}
