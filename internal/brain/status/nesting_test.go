package status

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
)

// A register of an area whose tree holds the wiki of a `local_only` area
// lists the inner pages too. On the cloud channel status names none of them
// and counts none: neither among the documents the engine cannot return nor
// under a shared content hash, where even "excluded by never" would say
// that a hidden twin exists. The local channel names them as before.
func TestLinesKeepThePagesOfAHiddenNestedWikiFromTheCloud(t *testing.T) {
	build := func(t *testing.T) (*world, *search.FakePort) {
		w := newWorld(t)
		hub := w.writable("hub", "hub", "\n[privacy]\nmode = \"manual_cloud\"\n")
		w.graph(hub, 0, 0, "")
		w.identities(hub, row("01", "inner/page.md", "sha256:1"), row("02", "top.md", "sha256:2"),
			row("03", "inner/copy.md", "sha256:2"), row("04", "inner/twin.md", "sha256:3"), row("05", "twin.md", "sha256:3"))
		innerPath := filepath.Join(w.repos, "inner-src")
		if err := os.MkdirAll(innerPath, 0o755); err != nil {
			t.Fatal(err)
		}
		inner := w.readOnly("project/inner", "project-inner", innerPath, "\n[privacy]\nmode = \"local_only\"\n")
		fmt.Fprintf(&w.entries, "wiki = '%s'\n\n", filepath.ToSlash(filepath.Join(hub, "inner")))
		w.graph(inner, 0, 0, "")
		port := search.NewFakePort()
		port.Listings = map[string]search.ScriptedIndexed{"hub": listing("top.md", "twin.md"), "project-inner": listing()}
		return w, port
	}

	w, port := build(t)
	got, err := w.lines(privacy.ChannelCloud, port)
	expect(t, got, err, neverReconciled)

	w, port = build(t)
	got, err = w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"hub: 3 of 5 indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. inner/copy.md, inner/page.md, inner/twin.md",
		"same content hash under 2 paths: hub/inner/copy.md, hub/top.md",
		"same content hash under 2 paths: hub/inner/twin.md, hub/twin.md")
}
