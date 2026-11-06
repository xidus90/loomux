package query

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/testlock"
)

// A declaration that exists but cannot be read must name itself in the error:
// the cloud channel refuses it rather than answering unfiltered.
func TestCloudKeepNamesAnUnreadableDeclaration(t *testing.T) {
	root := grepFixture(t)
	writeManifest(t, root, `"secrets/**"`)
	testlock.Lock(t, filepath.Join(root, ".loomux", "config.toml"))
	if _, err := cloudKeep(root); err == nil || !strings.Contains(err.Error(), "privacy declaration") {
		t.Fatalf("err = %v; want the unreadable declaration named", err)
	}
}
