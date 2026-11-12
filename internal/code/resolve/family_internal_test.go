package resolve

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
)

func TestFamilyOf(t *testing.T) {
	for lang, want := range map[string]string{
		"javascript": "ecmascript", "typescript": "ecmascript", "go": "go", "cpp": "cpp", "": "",
	} {
		if got := familyOf(lang); got != want {
			t.Errorf("familyOf(%q) = %q, want %q", lang, got, want)
		}
	}
}

func TestByFamilyJoinsJavaScriptAndTypeScriptInArrivalOrder(t *testing.T) {
	files := []extract.Result{
		{Path: "b.ts", Language: "typescript"}, {Path: "a.go", Language: "go"},
		{Path: "a.js", Language: "javascript"}, {Path: "c.ts", Language: "typescript"},
	}
	groups := byFamily(files)
	var got []string
	for _, f := range groups["ecmascript"] {
		got = append(got, f.Path)
	}
	if !reflect.DeepEqual(got, []string{"b.ts", "a.js", "c.ts"}) || len(groups) != 2 {
		t.Fatalf("groups = %+v", groups)
	}
}

func TestImportDirSkipsATSConfig(t *testing.T) {
	mods := []Module{{Dir: "web", TS: &TSConfig{BaseURL: "."}}}
	if dir, ok := importDir("", mods); ok {
		t.Fatalf("a tsconfig answered a Go import: %q", dir)
	}
}

func TestNearestTSConfig(t *testing.T) {
	mods := []Module{
		{Dir: "", Path: "example.com/repo"},
		{Dir: "", TS: &TSConfig{BaseURL: "."}},
		{Dir: "pkg/a", TS: &TSConfig{BaseURL: "src"}},
		{Dir: "pkg", TS: &TSConfig{BaseURL: "lib"}},
	}
	for file, want := range map[string]string{
		"pkg/a/x.ts": "src", "pkg/ab/x.ts": "lib", "pkg/x.ts": "lib", "x.ts": ".", "pkga/x.ts": ".",
	} {
		m := nearestTSConfig(mods, file)
		if m == nil || m.TS.BaseURL != want {
			t.Errorf("nearestTSConfig(%q) = %+v, want baseUrl %q", file, m, want)
		}
	}
	if m := nearestTSConfig(mods[:1], "x.ts"); m != nil {
		t.Errorf("a go.mod is no tsconfig: %+v", m)
	}
	if m := nearestTSConfig([]Module{{Dir: "pkg", TS: &TSConfig{}}}, "other/x.ts"); m != nil {
		t.Errorf("a file outside every tsconfig has none: %+v", m)
	}
}

func TestStripJSONC(t *testing.T) {
	for in, want := range map[string]string{
		`{"a": 1, // c` + "\n" + `}`:            `{"a": 1 ` + "\n" + `}`,
		`{"a": "//not", /* x */ "b": [1, 2,],}`: `{"a": "//not",  "b": [1, 2]}`,
		`{"a": "q\"/*", "b": 1}`:                `{"a": "q\"/*", "b": 1}`,
		`{"a": 1 /* open`:                       `{"a": 1 `,
		`{"a": 1 // end`:                        `{"a": 1 `,
		`{"a": "open`:                           `{"a": "open`,
		`[1, /* c */ ]`:                         `[1  ]`,
	} {
		if got := stripJSONC(in); got != want {
			t.Errorf("stripJSONC(%q) = %q, want %q", in, got, want)
		}
	}
}
