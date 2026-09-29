package reader_test

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/reader"
)

// An area registered inside a hidden tree conceals its own root, and so
// every path that reaches the concealment check. A path leaving the area
// must still be refused as leaving it, not answered as missing: containment
// ends ReadVisible before the concealment check is asked anything.
func TestReadVisibleRefusesALeavingPathFromAnAreaInsideAHiddenTree(t *testing.T) {
	cloud, _ := hubHolding(t)
	cloud.Hidden = []string{filepath.ToSlash(filepath.Dir(cloud.Area.Path))}
	if !cloud.Conceals(".") {
		t.Fatal("the area root is not concealed; the test would not separate the two orders")
	}
	_, err := reader.ReadVisible(cloud, "../elsewhere.md", "", privacy.ChannelCloud)
	if err == nil || errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "leaves the area") {
		t.Errorf("got %v, want the containment refusal", err)
	}
}
