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
	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
)

// The package's own suite drives every error hand-back of the answer itself:
// the corpus of internal/cli covers them too, but a change here must fail a
// test here.

// warmingPorts is a search that has to start its engine first: the port it
// builds tells the notice so, then answers one scripted search.
func warmingPorts() answer.Ports {
	ports := stubbedStatusPort()
	ports.Now = func() time.Time { return time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC) }
	ports.Search = func(notice func(string)) search.SearchPort {
		notice("warming")
		port := search.NewFakePort()
		port.Results = []search.ScriptedSearch{{Hits: []search.SearchHit{{Collection: "hub", Relative: "top.md", Title: "Top"}}}}
		return port
	}
	return ports
}

// The caller's notice hears the warming hint; a caller without one gets no
// crash.
func TestTheWarmingHintReachesTheCallersNotice(t *testing.T) {
	dir := nestedVault(t)
	req := answer.Request{Command: "search", Query: "q", Scope: "hub", Count: 5, Profile: "fast", Channel: privacy.ChannelLocal}
	var heard []string
	if _, _, err := answer.RunWith(warmingPorts(), req, dir, dir, func(m string) { heard = append(heard, m) }); err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(heard) != 1 || heard[0] != "warming" {
		t.Errorf("heard %q, want the warming hint", heard)
	}
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("a nil notice crashed the answer: %v", r)
			}
		}()
		if _, _, err := answer.RunWith(warmingPorts(), req, dir, dir, nil); err != nil {
			t.Errorf("search with a nil notice: %v", err)
		}
	}()
}

// An engine that fails is the answer's error, not an empty hit list.
func TestAFailingSearchEngineIsTheAnswersError(t *testing.T) {
	dir := nestedVault(t)
	broken := errors.New("engine down")
	port := search.NewFakePort()
	port.Results = []search.ScriptedSearch{{Err: broken}, {Err: broken}}
	ports := stubbedStatusPort()
	ports.Now = func() time.Time { return time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC) }
	ports.Search = func(func(string)) search.SearchPort { return port }
	var text string
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("a failing engine crashed the answer: %v", r)
			}
		}()
		text, _, err = answer.RunWith(ports, answer.Request{Command: "search", Query: "q", Scope: "hub", Count: 5, Profile: "fast", Channel: privacy.ChannelLocal}, dir, dir, nil)
	}()
	if !errors.Is(err, broken) || text != "" {
		t.Errorf("got %q, %v; want the engine's error", text, err)
	}
}

// withoutRegistry is a state directory whose registry is gone: every command
// that asks the registry has to answer with the read error of that file.
func withoutRegistry(t *testing.T) string {
	t.Helper()
	dir := nestedVault(t)
	if err := os.Remove(filepath.Join(dir, "registry.toml")); err != nil {
		t.Fatal(err)
	}
	return dir
}

// A registry that does not read is the error of catalog, read, neighbours and
// status alike -- not an empty catalog, not an unknown scope, not an empty
// status.
func TestAMissingRegistryIsTheErrorOfEveryRegistryCommand(t *testing.T) {
	dir := withoutRegistry(t)
	for _, req := range []answer.Request{
		{Command: "catalog", Scope: "all", Channel: privacy.ChannelCloud},
		{Command: "catalog", Scope: "hub", Channel: privacy.ChannelCloud},
		{Command: "read", Query: "top.md", Scope: "hub", Channel: privacy.ChannelCloud},
		{Command: "neighbors", Query: "top.md", Scope: "hub", Channel: privacy.ChannelCloud},
		{Command: "status", Channel: privacy.ChannelCloud},
	} {
		text, err := ask(t, dir, req)
		if !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "registry.toml") || text != "" {
			t.Errorf("%s %s: %q, %v; want the registry's read error", req.Command, req.Scope, text, err)
		}
	}
}

// Scope "all" is the root catalog of the visible areas; the hidden area is
// not in it on cloud.
func TestTheCatalogOfAllIsTheRootCatalog(t *testing.T) {
	dir := nestedVault(t)
	for _, tc := range []struct {
		channel privacy.Channel
		want    string
	}{
		{privacy.ChannelCloud, "# brain\n\n* [hub](brain://hub/)\n"},
		{privacy.ChannelLocal, "# brain\n\n* [hub](brain://hub/)\n* [project/inner](brain://project/inner/)\n"},
	} {
		text, err := ask(t, dir, answer.Request{Command: "catalog", Scope: "all", Channel: tc.channel})
		if err != nil || text != tc.want {
			t.Errorf("%s: %q, %v; want %q", tc.channel, text, err, tc.want)
		}
	}
}

// A scope no visible area has is the one unknown-scope message, for catalog,
// read and neighbours alike, and the hidden area answers it on cloud too.
func TestAnUnknownScopeIsTheUnknownScopeError(t *testing.T) {
	dir := nestedVault(t)
	for _, scope := range []string{"nosuch", "project/inner"} {
		for _, req := range []answer.Request{
			{Command: "catalog", Scope: scope, Channel: privacy.ChannelCloud},
			{Command: "read", Query: "top.md", Scope: scope, Channel: privacy.ChannelCloud},
			{Command: "neighbors", Query: "top.md", Scope: scope, Channel: privacy.ChannelCloud},
		} {
			var text string
			var err error
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s %s crashed: %v", req.Command, scope, r)
					}
				}()
				text, err = ask(t, dir, req)
			}()
			want := "unknown scope '" + scope + "'; known scopes are: hub"
			if err == nil || err.Error() != want || text != "" {
				t.Errorf("%s %s: %q, %v; want %q", req.Command, scope, text, err, want)
			}
		}
	}
}

// An area that was never indexed has no graph, and its neighbours are that
// state, not an empty answer.
func TestNeighborsOfAnAreaWithoutAGraphAreNeverIndexed(t *testing.T) {
	dir := nestedVault(t)
	if err := os.Remove(filepath.Join(dir, "hub", "graph.json")); err != nil {
		t.Fatal(err)
	}
	var text string
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("neighbours without a graph crashed: %v", r)
			}
		}()
		text, err = ask(t, dir, answer.Request{Command: "neighbors", Query: "top.md", Scope: "hub", Channel: privacy.ChannelCloud})
	}()
	if !errors.Is(err, graph.ErrNotIndexed) || text != "" {
		t.Errorf("got %q, %v; want the never-indexed error", text, err)
	}
}
