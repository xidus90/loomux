package graph_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/graph"
)

func TestResolveTarget_Basic(t *testing.T) {
	wiki := "wiki"

	tests := []struct {
		source     string
		target     string
		wikiPrefix *string
		wantPath   string
		wantOK     bool
	}{
		{"alpha.md", "beta.md", nil, "beta.md", true},
		{"sub/alpha.md", "beta.md", nil, "sub/beta.md", true},
		{"sub/alpha.md", "../beta.md", nil, "beta.md", true},
		{"sub/deep/alpha.md", "../../beta.md", nil, "beta.md", true},
		{"alpha.md", "../escaped.md", nil, "", false},
		{"sub/alpha.md", "../../escaped.md", nil, "", false},
		{"sub/a.md", "my%20page.md", nil, "sub/my page.md", true},
		{"wiki/a.md", "/bundle.md", &wiki, "wiki/bundle.md", true},
		{"wiki/a.md", "/../bundle.md", &wiki, "wiki/bundle.md", true},
		{"outside/a.md", "/bundle.md", &wiki, "bundle.md", true},
		// Anhang B: query and fragment trimming
		{"alpha.md", "page.md#section", nil, "page.md", true},
		{"alpha.md", "page.md?version=1", nil, "page.md", true},
		{"alpha.md", "page.md?version=1#section", nil, "page.md", true},
		{"sub/alpha.md", "page.md#section", nil, "sub/page.md", true},
		// Invalid percent unescape fallback
		{"alpha.md", "invalid%ZZpage.md", nil, "invalid%ZZpage.md", true},
		// Dot in path
		{"alpha.md", "./other.md", nil, "other.md", true},
		// Bundle prefix variants
		{"wiki", "/page.md", &wiki, "wiki/page.md", true},
		{"wiki/sub/a.md", "/page.md", strPtr(""), "page.md", true},
		// Empty target
		{"alpha.md", "", nil, ".", true},
	}

	for _, tc := range tests {
		gotPath, gotOK := graph.ResolveTarget(tc.source, tc.target, tc.wikiPrefix)
		if gotPath != tc.wantPath || gotOK != tc.wantOK {
			t.Errorf("ResolveTarget(%q, %q) = (%q, %v), want (%q, %v)",
				tc.source, tc.target, gotPath, gotOK, tc.wantPath, tc.wantOK)
		}
	}
}

func TestInOwnWiki(t *testing.T) {
	if graph.InOwnWiki("anything.md", nil) {
		t.Error("expected false for nil prefix")
	}
}

func TestDropReason(t *testing.T) {
	known := map[string]bool{
		"page.md":     true,
		"sub/doc.md":  true,
		"without-ext": true,
	}

	tests := []struct {
		name       string
		source     string
		link       string
		wikiPrefix *string
		wantReason *string
	}{
		// Anhang B: General URI schemes classified as "external"
		{"http", "a.md", "http://example.com", nil, strPtr("external")},
		{"https", "a.md", "https://example.com/test", nil, strPtr("external")},
		{"mailto", "a.md", "mailto:alice@example.com", nil, strPtr("external")},
		{"ftp", "a.md", "ftp://files.example.com/file", nil, strPtr("external")},
		{"ssh", "a.md", "ssh://git@github.com", nil, strPtr("external")},
		{"irc", "a.md", "irc://irc.libera.chat", nil, strPtr("external")},
		{"custom scheme", "a.md", "custom+scheme-1.0://test", nil, strPtr("external")},

		// Anchors
		{"anchor", "a.md", "#heading", nil, strPtr("anchor")},

		// Outside area
		{"outside area", "a.md", "../outside.md", nil, strPtr("outside_area")},

		// Unknown target
		{"unknown target", "a.md", "missing.md", nil, strPtr("unknown_target")},

		// Resolved targets (reason == nil)
		{"resolved exact", "a.md", "page.md", nil, nil},
		{"resolved with .md suffix added", "a.md", "without-ext", nil, nil},
		// Anhang B: query and fragment preserved links now resolve!
		{"resolved with fragment", "a.md", "page.md#heading", nil, nil},
		{"resolved with query", "a.md", "page.md?v=1", nil, nil},
		{"resolved in subfolder", "sub/other.md", "doc.md#intro", nil, nil},
	}

	for _, tc := range tests {
		got := graph.DropReason(tc.source, tc.link, known, tc.wikiPrefix)
		if tc.wantReason == nil {
			if got != nil {
				t.Errorf("%s: got reason %q, want nil", tc.name, *got)
			}
		} else {
			if got == nil {
				t.Errorf("%s: got nil, want reason %q", tc.name, *tc.wantReason)
			} else if *got != *tc.wantReason {
				t.Errorf("%s: got reason %q, want %q", tc.name, *got, *tc.wantReason)
			}
		}
	}
}

func TestRenderGraph_SortingAndDeduplication(t *testing.T) {
	docs := []graph.Document{
		{
			Relative: "beta.md",
			Title:    "Beta",
			Tags:     []string{"tagB"},
			Links:    []string{"alpha.md", "alpha.md#duplicate", "https://example.com"},
		},
		{
			Relative: "alpha.md",
			Title:    "Alpha",
			Tags:     nil,
			Links:    []string{"beta.md", "#self-anchor", "missing.md", "../outside.md"},
		},
	}

	data, err := graph.RenderGraph("project/test", docs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasSuffix(string(data), "\n") {
		t.Error("expected trailing newline on rendered graph")
	}

	var parsed graph.Graph
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal rendered graph: %v", err)
	}

	if parsed.Scope != "project/test" {
		t.Errorf("got scope %q, want project/test", parsed.Scope)
	}

	// Nodes sorted by relative
	if len(parsed.Nodes) != 2 || parsed.Nodes[0].ID != "alpha.md" || parsed.Nodes[1].ID != "beta.md" {
		t.Errorf("nodes not sorted by relative: %+v", parsed.Nodes)
	}

	// Edges deduplicated: beta -> alpha only once despite two links
	wantEdges := []graph.Edge{
		{From: "alpha.md", To: "beta.md"},
		{From: "beta.md", To: "alpha.md"},
	}
	if !reflect.DeepEqual(parsed.Edges, wantEdges) {
		t.Errorf("got edges %+v, want %+v", parsed.Edges, wantEdges)
	}

	// Links counts:
	// alpha.md:
	//   beta.md: resolved (1)
	//   #self-anchor: anchor (1)
	//   missing.md: unknown_target (1)
	//   ../outside.md: outside_area (1)
	// beta.md:
	//   alpha.md: resolved (1)
	//   alpha.md#duplicate: resolved (1) (Anhang B fix!)
	//   https://example.com: external (1)
	// Total = 7, Resolved = 3, Dropped: anchor=1, external=1, outside_area=1, unknown_target=1
	if parsed.Links.Total != 7 {
		t.Errorf("got total %d, want 7", parsed.Links.Total)
	}
	if parsed.Links.Resolved != 3 {
		t.Errorf("got resolved %d, want 3", parsed.Links.Resolved)
	}
	wantDropped := map[string]int{
		"anchor":         1,
		"external":       1,
		"outside_area":   1,
		"unknown_target": 1,
	}
	if !reflect.DeepEqual(parsed.Links.Dropped, wantDropped) {
		t.Errorf("got dropped %+v, want %+v", parsed.Links.Dropped, wantDropped)
	}
}

func strPtr(s string) *string {
	return &s
}

// A first path component that merely starts with two dots is a file name, not
// a climb out of the area.
func TestResolveTargetKeepsADotDotNamedFileInside(t *testing.T) {
	tests := []struct {
		source, link, want string
		ok                 bool
	}{
		{"a.md", "..draft.md", "..draft.md", true},
		{"sub/a.md", "../..draft.md", "..draft.md", true},
		{"sub/a.md", "..draft.md", "sub/..draft.md", true},
		{"a.md", "../out.md", "", false},
		{"a.md", "..", "", false},
	}
	for _, tt := range tests {
		got, ok := graph.ResolveTarget(tt.source, tt.link, nil)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ResolveTarget(%q, %q) = %q, %v; want %q, %v", tt.source, tt.link, got, ok, tt.want, tt.ok)
		}
	}
	if r := graph.DropReason("a.md", "..draft.md", map[string]bool{"..draft.md": true}, nil); r != nil {
		t.Errorf("a link to a ..-named page dropped as %q", *r)
	}
}
