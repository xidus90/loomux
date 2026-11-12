package graph_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
)

// A call that lacks both the scope and the tool's own required argument hears
// about the latter: the scope may be left out, and the argument is refused
// before resolve reads the registry.
func TestAToolWithNeitherArgumentAsksForItsOwnArgument(t *testing.T) {
	dir, _, _ := registry(t)
	for tool, arg := range map[string]string{"graph_file_api": "file", "graph_trace_calls": "symbol", "graph_find_all": "pattern"} {
		text, isError := call(t, connect(t, privacy.ChannelLocal, (&recorder{}).deps(dir)), tool, map[string]any{})
		if want := tool + " requires a " + arg; !isError || text != want {
			t.Errorf("%s: got %q (isError %v), want %q", tool, text, isError, want)
		}
	}
}

// max_dirs of exactly 1 is taken as given; below 1 (a fraction, zero) it
// falls back to the default of 16 instead of reaching Map as 0.
func TestRepoMapTakesAMaxDirsOfOneAndFallsBackBelowIt(t *testing.T) {
	dir, _, _ := registry(t)
	for _, tc := range []struct {
		given float64
		want  int
	}{{1, 1}, {1.5, 1}, {0.5, 16}, {0, 16}, {-3, 16}} {
		r := &recorder{}
		text, isError := call(t, connect(t, privacy.ChannelLocal, r.deps(dir)), "graph_repo_map",
			map[string]any{"scope": "project/open", "max_dirs": tc.given})
		if isError {
			t.Fatalf("max_dirs %v: unexpected error %q", tc.given, text)
		}
		if r.mapOpts.MaxDirs != tc.want {
			t.Errorf("max_dirs %v: MaxDirs = %d, want %d", tc.given, r.mapOpts.MaxDirs, tc.want)
		}
	}
}
