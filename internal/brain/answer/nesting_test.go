package answer_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
)

// nestedVault is the registry found on this machine on 2026-09-29, reduced:
// the open area "hub", whose tree is its own wiki, holds the wiki of the
// read-only `local_only` area "project/inner", whose source tree lies beside
// it. The hub carries a catalog naming the nested wiki and a graph with edges
// into it. It answers the state directory, which holds the registry and the
// read-only area's stock both.
func nestedVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	put := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	hub := filepath.Join(root, "hub")
	innerWiki := filepath.Join(hub, "inner")
	innerPath := filepath.Join(root, "inner-src")
	put(filepath.Join(hub, ".loomux", "config.toml"), "[area]\nscope = \"hub\"\n\n[privacy]\nmode = \"manual_cloud\"\n")
	put(filepath.Join(hub, "top.md"), "# Top\n")
	put(filepath.Join(hub, "index.md"), "# hub\n\n## Bereiche\n\n* [inner](inner/)\n* [other](other/)\n\n## Dateien\n\n* [Top](top.md)\n")
	put(filepath.Join(hub, "graph.json"), `{"scope": "hub", "nodes": [], "edges": [`+
		`{"from": "top.md", "to": "inner/page.md"}, {"from": "inner/page.md", "to": "top.md"}, `+
		`{"from": "other/x.md", "to": "top.md"}, {"from": "inner/page.md", "to": "other/x.md"}], `+
		`"links": {"total": 0, "resolved": 0, "dropped": {}}}`+"\n")
	put(filepath.Join(innerWiki, "page.md"), "# Page\n\nlocal only\n")
	put(filepath.Join(innerPath, "source.md"), "# Source\n")
	put(filepath.Join(root, "areas", "project-inner", ".loomux", "config.toml"), "[area]\nscope = \"project/inner\"\n\n[privacy]\nmode = \"local_only\"\n")
	put(filepath.Join(root, "registry.toml"),
		"[[area]]\nscope = \"hub\"\npath = \""+filepath.ToSlash(hub)+"\"\nwiki = \""+filepath.ToSlash(hub)+"\"\n\n"+
			"[[area]]\nscope = \"project/inner\"\npath = \""+filepath.ToSlash(innerPath)+"\"\nwiki = \""+filepath.ToSlash(innerWiki)+"\"\nreadonly = true\n")
	return root
}

func ask(t *testing.T, dir string, req answer.Request) (string, error) {
	t.Helper()
	ports := stubbedStatusPort()
	ports.Now = func() time.Time { return time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC) }
	text, _, err := answer.RunWith(ports, req, dir, nil)
	return text, err
}

// The leak of 2026-09-29: the enclosing scope handed the cloud channel a page
// of the `local_only` area. It is not there on cloud, exactly as a missing
// page is not, while the local channel reads it through either scope.
func TestReadThroughAnEnclosingScopeHidesALocalOnlyPageOnCloud(t *testing.T) {
	dir := nestedVault(t)
	_, err := ask(t, dir, answer.Request{Command: "read", Query: "inner/page.md", Scope: "hub", Channel: privacy.ChannelCloud})
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("cloud read through hub: %v, want the missing-file error", err)
	}
	text, err := ask(t, dir, answer.Request{Command: "read", Query: "inner/page.md", Scope: "hub", Channel: privacy.ChannelLocal})
	if err != nil || text != "# Page\n\nlocal only\n" {
		t.Errorf("local read through hub: %q, %v", text, err)
	}
	text, err = ask(t, dir, answer.Request{Command: "read", Query: "top.md", Scope: "hub", Channel: privacy.ChannelCloud})
	if err != nil || text != "# Top\n" {
		t.Errorf("cloud read of an open page: %q, %v", text, err)
	}
}

// The hub's catalog names the nested wiki; on cloud that line is gone.
func TestCatalogOfAnEnclosingScopeDropsTheHiddenWikiOnCloud(t *testing.T) {
	dir := nestedVault(t)
	full := "# hub\n\n## Bereiche\n\n* [inner](inner/)\n* [other](other/)\n\n## Dateien\n\n* [Top](top.md)\n"
	for _, tc := range []struct {
		channel privacy.Channel
		want    string
	}{
		{privacy.ChannelCloud, strings.Replace(full, "* [inner](inner/)\n", "", 1)},
		{privacy.ChannelLocal, full},
	} {
		text, err := ask(t, dir, answer.Request{Command: "catalog", Scope: "hub", Channel: tc.channel})
		if err != nil || text != tc.want {
			t.Errorf("%s: %q, %v; want %q", tc.channel, text, err, tc.want)
		}
	}
}

// A catalog that is not there is still the read error, not an empty answer.
func TestCatalogOfAnAreaWithoutOneIsTheReadError(t *testing.T) {
	dir := nestedVault(t)
	if err := os.Remove(filepath.Join(dir, "hub", "index.md")); err != nil {
		t.Fatal(err)
	}
	text, err := ask(t, dir, answer.Request{Command: "catalog", Scope: "hub", Channel: privacy.ChannelCloud})
	if !errors.Is(err, fs.ErrNotExist) || text != "" {
		t.Errorf("got %q, %v; want the missing-file error", text, err)
	}
}

// The hub's graph has edges into the nested wiki. On cloud a hidden page has
// no neighbours, as a page unknown to the graph has none, and a hidden page
// is no neighbour of an open one.
func TestNeighborsThroughAnEnclosingScopeHideTheHiddenPagesOnCloud(t *testing.T) {
	dir := nestedVault(t)
	for _, tc := range []struct {
		channel  privacy.Channel
		relative string
		want     string
	}{
		{privacy.ChannelCloud, "top.md", "incoming: other/x.md\noutgoing: -\n"},
		{privacy.ChannelCloud, "inner/page.md", "incoming: -\noutgoing: -\n"},
		{privacy.ChannelCloud, "other/x.md", "incoming: -\noutgoing: top.md\n"},
		{privacy.ChannelLocal, "top.md", "incoming: inner/page.md, other/x.md\noutgoing: inner/page.md\n"},
		{privacy.ChannelLocal, "inner/page.md", "incoming: top.md\noutgoing: other/x.md, top.md\n"},
	} {
		text, err := ask(t, dir, answer.Request{Command: "neighbors", Query: tc.relative, Scope: "hub", Channel: tc.channel})
		if err != nil || text != tc.want {
			t.Errorf("%s %s: %q, %v; want %q", tc.channel, tc.relative, text, err, tc.want)
		}
	}
	if _, err := ask(t, dir, answer.Request{Command: "neighbors", Query: "../x.md", Scope: "hub", Channel: privacy.ChannelCloud}); err == nil {
		t.Error("a path leaving the area was not refused")
	}
}

// A search through the enclosing scope, the path the MCP brain_search tool
// takes as well, keeps the hidden page from the cloud channel.
func TestSearchThroughAnEnclosingScopeHidesTheHiddenPageOnCloud(t *testing.T) {
	dir := nestedVault(t)
	for _, tc := range []struct {
		channel privacy.Channel
		hidden  bool
	}{
		{privacy.ChannelCloud, true},
		{privacy.ChannelLocal, false},
	} {
		port := search.NewFakePort()
		port.Results = []search.ScriptedSearch{{Hits: []search.SearchHit{
			{Collection: "hub", Relative: "inner/page.md", Title: "Page"},
			{Collection: "hub", Relative: "top.md", Title: "Top"},
		}}}
		ports := stubbedStatusPort()
		ports.Search = func(func(string)) search.SearchPort { return port }
		ports.Now = func() time.Time { return time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC) }
		text, _, err := answer.RunWith(ports, answer.Request{Command: "search", Query: "q", Scope: "hub", Count: 5, Profile: "fast", Channel: tc.channel}, dir, nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.channel, err)
		}
		if strings.Contains(text, "inner/page.md") == tc.hidden || !strings.Contains(text, "brain://hub/top.md") {
			t.Errorf("%s: %q", tc.channel, text)
		}
	}
}
