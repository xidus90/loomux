package search_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
)

// nestedWorld registers the open area "hub", whose tree holds the wiki of
// the read-only `local_only` area "project/inner", as the registry on this
// machine does. It returns the directory that holds the registry and the
// legacy state both.
func nestedWorld(t *testing.T) string {
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
	put(filepath.Join(innerWiki, "page.md"), "# Page\n")
	put(filepath.Join(innerPath, "source.md"), "# Source\n")
	put(filepath.Join(root, "areas", "project-inner", ".loomux", "config.toml"), "[area]\nscope = \"project/inner\"\n\n[privacy]\nmode = \"local_only\"\n")
	writeRegistry(t, root, areaEntry("hub", hub)+
		"[[area]]\nscope = \"project/inner\"\npath = \""+filepath.ToSlash(innerPath)+"\"\nwiki = \""+filepath.ToSlash(innerWiki)+"\"\nreadonly = true\n")
	return root
}

// A hit of the enclosing area that lies in the hidden area's wiki does not
// reach a cloud caller, and is counted with the other withheld hits; the
// local channel keeps it.
func TestExecuteSearch_AHitInsideAHiddenTreeIsWithheldOnCloud(t *testing.T) {
	stateDir := nestedWorld(t)
	hits := []search.SearchHit{
		{Collection: "hub", Relative: "inner/page.md", Title: "Page"},
		{Collection: "hub", Relative: "top.md", Title: "Top"},
	}
	for _, tc := range []struct {
		channel  privacy.Channel
		want     []string
		withheld bool
	}{
		{privacy.ChannelCloud, []string{"top.md"}, true},
		{privacy.ChannelLocal, []string{"inner/page.md", "top.md"}, false},
	} {
		port := &mockSearchPort{searchFunc: func(string, []string, search.Profile, int) ([]search.SearchHit, error) {
			return hits, nil
		}}
		found, err := search.ExecuteSearch("q", "hub", search.ProfileFast, 5, tc.channel, port, stateDir, stateDir, searchNow)
		if err != nil {
			t.Fatalf("%s: %v", tc.channel, err)
		}
		var got []string
		for _, hit := range found.Hits {
			got = append(got, hit.Relative)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: hits %v, want %v", tc.channel, got, tc.want)
		}
		counted := false
		for _, finding := range found.Findings {
			counted = counted || finding == withheldOne
		}
		if counted != tc.withheld {
			t.Errorf("%s: findings %q, withheld count wanted: %v", tc.channel, found.Findings, tc.withheld)
		}
	}
}
