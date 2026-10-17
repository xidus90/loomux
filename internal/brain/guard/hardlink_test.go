package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hardLink makes a second name for target, or skips where the file system
// keeps none.
func hardLink(t *testing.T, target, name string) {
	t.Helper()
	mkdir(t, filepath.Dir(name))
	if err := os.Link(target, name); err != nil {
		t.Skipf("no hard link here: %v", err)
	}
}

func TestAHardLinkInsideTheBundleToAManifestIsRefused(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	manifest := filepath.Join(tmp, "other", ".loomux", "config.toml")
	write(t, manifest, "[area]\nscope = \"x\"\n")
	link := filepath.Join(tmp, "vault", "demo", "notes.md")
	hardLink(t, manifest, link)
	deny(t, writeCall(link), state, "has 2 links")
}

func TestAHardLinkInMemoryIsRefused(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	homeAt(t, home, "")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	outside := filepath.Join(tmp, "repo", "src", "a.py")
	write(t, outside, "x")
	link := filepath.Join(claudeMemory(home), "a.md")
	hardLink(t, outside, link)
	deny(t, writeCall(link), state, "has 2 links")
}

func TestAFileOfOneLinkInsideTheBundlePasses(t *testing.T) {
	tmp := t.TempDir()
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	page := filepath.Join(tmp, "vault", "demo", "x.md")
	write(t, page, "x")
	allow(t, writeCall(page), state)
}

func TestHardLinkedIsSilentWhereThereIsOneNameOrNone(t *testing.T) {
	tmp := t.TempDir()
	for _, path := range []string{tmp, filepath.Join(tmp, "absent", "x.md")} {
		if reason := HardLinked(path); reason != "" {
			t.Fatalf("HardLinked(%q) = %q, want none", path, reason)
		}
	}
}

func TestHardLinkedRefusesWhatItCannotCount(t *testing.T) {
	if reason := HardLinked("bad\x00name"); !strings.Contains(reason, "cannot count") {
		t.Fatalf("reason %q", reason)
	}
}
