package search_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/testlock"
)

// closedWorld registers, beside the open area "alpha", the read-only "zz-lent" whose
// manifest is the `local_only` one in the legacy state directory while its checkout
// carries an open `.brain.toml`. With sealed it adds "zz-sealed", whose `local_only`
// `.ultra-brain/config.toml` cannot be read beside a stale `manual_cloud` `.brain.toml`.
// It returns the directory that holds the registry and the legacy state both.
func closedWorld(t *testing.T, sealed bool) string {
	t.Helper()
	root := t.TempDir()
	const open = "[privacy]\nmode = \"manual_cloud\"\n"
	const closed = "[privacy]\nmode = \"local_only\"\n"
	put := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	alpha := filepath.Join(root, "alpha")
	lent := filepath.Join(root, "lent")
	put(filepath.Join(alpha, ".brain.toml"), "[area]\nscope = \"alpha\"\n")
	put(filepath.Join(lent, ".brain.toml"), "[area]\nscope = \"zz-lent\"\n"+open)
	put(filepath.Join(root, "areas", "zz-lent", ".brain.toml"), "[area]\nscope = \"zz-lent\"\n"+closed)
	registry := "[[area]]\nscope = \"alpha\"\npath = \"" + filepath.ToSlash(alpha) + "\"\n\n" +
		"[[area]]\nscope = \"zz-lent\"\npath = \"" + filepath.ToSlash(lent) + "\"\nreadonly = true\n\n"
	if sealed {
		dir := filepath.Join(root, "sealed")
		put(filepath.Join(dir, ".brain.toml"), "[area]\nscope = \"zz-sealed\"\n"+open)
		config := filepath.Join(dir, ".ultra-brain", "config.toml")
		put(config, "[area]\nscope = \"zz-sealed\"\n"+closed)
		testlock.Lock(t, config)
		registry += "[[area]]\nscope = \"zz-sealed\"\npath = \"" + filepath.ToSlash(dir) + "\"\n"
	}
	writeRegistry(t, root, registry)
	return root
}

// N3 of the scheibe-6 merge re-review, on the surface where it matters most:
// the engine must not be asked about a closed area at all, because the
// question is already the disclosure. A read-only area's own declaration is the
// one in the legacy state directory, not the one in its checkout.
func TestExecuteSearch_AClosedDeclarationIsNotAsked(t *testing.T) {
	stateDir := closedWorld(t, false)
	for _, tc := range []struct {
		channel privacy.Channel
		want    []string
	}{
		{privacy.ChannelCloud, []string{"alpha"}},
		{privacy.ChannelLocal, []string{"alpha", "zz-lent"}},
	} {
		var requested []string
		port := &mockSearchPort{
			searchFunc: func(_ string, collections []string, _ search.Profile, _ int) ([]search.SearchHit, error) {
				requested = collections
				return nil, nil
			},
		}
		if _, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, tc.channel, port, stateDir, stateDir, searchNow); err != nil {
			t.Fatalf("%s: %v", tc.channel, err)
		}
		if !reflect.DeepEqual(requested, tc.want) {
			t.Errorf("%s: engine asked about %v, want %v", tc.channel, requested, tc.want)
		}
	}

	port := &mockSearchPort{}
	_, err := search.ExecuteSearch("q", "zz-lent", search.ProfileFast, 5, privacy.ChannelCloud, port, stateDir, stateDir, searchNow)
	if err == nil || !strings.Contains(err.Error(), "unknown scope") {
		t.Errorf("cloud search of zz-lent: err = %v, want the unknown-scope refusal", err)
	}
	if port.calls != 0 {
		t.Errorf("cloud search of zz-lent asked the engine %d times", port.calls)
	}
}

// A declaration that exists and cannot be read is not skipped: core._visible_areas reads the
// manifest of every registered area and lets the OSError out, so the whole search fails on
// either channel and the engine is never asked.
func TestExecuteSearch_AnUnreadableDeclarationFailsTheWholeSearch(t *testing.T) {
	stateDir := closedWorld(t, true)
	for _, channel := range []privacy.Channel{privacy.ChannelCloud, privacy.ChannelLocal} {
		port := &mockSearchPort{}
		if _, err := search.ExecuteSearch("q", "all", search.ProfileFast, 5, channel, port, stateDir, stateDir, searchNow); err == nil {
			t.Errorf("%s: expected the unreadable declaration to fail the search", channel)
		}
		if port.calls != 0 {
			t.Errorf("%s: the engine was asked %d times", channel, port.calls)
		}
	}
}
