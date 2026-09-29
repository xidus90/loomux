package reader_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/reader"
	"github.com/xidus90/loomux/internal/config"
)

// hubHolding is an open area whose tree holds the tree of a hidden one, as
// privacy.VisibleAreas answers it on the cloud channel, and the same area as
// the local channel sees it.
func hubHolding(t *testing.T) (cloud, local privacy.VisibleArea) {
	t.Helper()
	hub := t.TempDir()
	for name, text := range map[string]string{
		"top.md":           "# Top\n",
		"inner/page.md":    "# Page\n\n## Secret\n\nlocal only\n",
		"inner-neu/new.md": "# New\n",
	} {
		path := filepath.Join(hub, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	local = privacy.VisibleArea{Area: config.Area{Scope: "hub", Path: filepath.ToSlash(hub)}, Manifest: &config.Manifest{}}
	cloud = local
	cloud.Hidden = []string{filepath.ToSlash(filepath.Join(hub, "inner"))}
	return cloud, local
}

// A page inside a hidden tree is refused like a page that is not there, with
// or without a section: the error a cloud caller reads for it is the one a
// missing file in the same directory gives, spelt the same way, so an
// existing hidden page and an absent one cannot be told apart.
func TestReadVisibleRefusesAHiddenPageLikeAMissingOne(t *testing.T) {
	cloud, local := hubHolding(t)
	_, missing := reader.ReadVisible(local, "inner/absent.md", "", privacy.ChannelLocal)
	if !errors.Is(missing, fs.ErrNotExist) {
		t.Fatalf("the missing page: %v", missing)
	}
	absent := filepath.Join(local.Area.Path, "inner", "absent.md")
	present := filepath.Join(local.Area.Path, "inner", "page.md")
	want := strings.Replace(missing.Error(), absent, present, 1)
	for _, section := range []string{"", "Secret"} {
		text, err := reader.ReadVisible(cloud, "inner/page.md", section, privacy.ChannelCloud)
		if !errors.Is(err, fs.ErrNotExist) || err.Error() != want || text != "" {
			t.Errorf("section %q: got %q, %v; want the missing-file error %q", section, text, err, want)
		}
	}
	if _, err := reader.ReadVisible(cloud, "./inner//page.md", "", privacy.ChannelCloud); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("another spelling of the hidden page: %v", err)
	}
}

// Everything the hidden tree does not hold reads as before, and the local
// channel reads the hidden page itself.
func TestReadVisibleReadsWhatIsNotHidden(t *testing.T) {
	cloud, local := hubHolding(t)
	for _, tc := range []struct {
		area     privacy.VisibleArea
		ch       privacy.Channel
		relative string
		want     string
	}{
		{cloud, privacy.ChannelCloud, "top.md", "# Top\n"},
		{cloud, privacy.ChannelCloud, "inner-neu/new.md", "# New\n"},
		{local, privacy.ChannelLocal, "inner/page.md", "# Page\n\n## Secret\n\nlocal only\n"},
	} {
		got, err := reader.ReadVisible(tc.area, tc.relative, "", tc.ch)
		if err != nil || got != tc.want {
			t.Errorf("%s %s: got %q, %v; want %q", tc.ch, tc.relative, got, err, tc.want)
		}
	}
}

// Containment is asked first: a path leaving the area is refused as leaving
// it, whatever the hidden trees are.
func TestReadVisibleRefusesALeavingPathFirst(t *testing.T) {
	cloud, _ := hubHolding(t)
	_, err := reader.ReadVisible(cloud, "../inner/page.md", "", privacy.ChannelCloud)
	if err == nil || !strings.Contains(err.Error(), "leaves the area") {
		t.Errorf("got %v, want the containment refusal", err)
	}
}
