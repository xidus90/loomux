package rules

import (
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

func layer(name string, paths ...string) config.Layer {
	return config.Layer{Name: name, Paths: paths, Checked: true}
}

func TestAssign(t *testing.T) {
	cases := []struct {
		name   string
		layers []config.Layer
		rels   []string
		want   map[string]string // rel -> layer; absent: outside every layer
		errs   []string
	}{
		{"file beats directory",
			[]config.Layer{layer("dir", "core/**"), layer("file", "core/a.gd")},
			[]string{"core/a.gd", "core/b.gd", "other/c.gd"},
			map[string]string{"core/a.gd": "file", "core/b.gd": "dir"}, nil},
		{"deeper beats shallower",
			[]config.Layer{layer("outer", "godot/**"), layer("inner", "godot/core/**")},
			[]string{"godot/core/a.gd", "godot/b.gd"},
			map[string]string{"godot/core/a.gd": "inner", "godot/b.gd": "outer"}, nil},
		{"directory beats base-name glob",
			[]config.Layer{layer("any", "*.gd"), layer("dir", "core/**")},
			[]string{"core/a.gd", "x/b.gd"},
			map[string]string{"core/a.gd": "dir", "x/b.gd": "any"}, nil},
		{"leading any-directory is least specific",
			[]config.Layer{layer("any", "**/a.gd"), layer("dir", "core/*.gd")},
			[]string{"core/a.gd"},
			map[string]string{"core/a.gd": "dir"}, nil},
		{"tie between layers",
			[]config.Layer{layer("a", "core/*.gd"), layer("b", "core/a.*")},
			[]string{"core/a.gd"},
			map[string]string{}, []string{"core/a.gd matches layers a and b equally"}},
		{"tie between three layers",
			[]config.Layer{layer("a", "core/*.gd"), layer("b", "core/a.*"), layer("c", "core/a.g?")},
			[]string{"core/a.gd"},
			map[string]string{}, []string{"core/a.gd matches layers a, b and c equally"}},
		{"tie within one layer is fine",
			[]config.Layer{layer("a", "core/*.gd", "core/a.*")},
			[]string{"core/a.gd"},
			map[string]string{"core/a.gd": "a"}, nil},
		{"glob without match",
			[]config.Layer{layer("x", "godot/sim/**", "core/**")},
			[]string{"core/a.gd"},
			map[string]string{"core/a.gd": "x"}, []string{"layer x: godot/sim/** matches no file"}},
		{"bad glob",
			[]config.Layer{layer("x", "[")},
			[]string{"a.gd"},
			map[string]string{}, []string{"layer x: glob \"[\": syntax error in pattern"}},
		{"errors sorted",
			[]config.Layer{layer("z", "nope/**"), layer("a", "gone/**")},
			nil,
			map[string]string{}, []string{"layer a: gone/** matches no file", "layer z: nope/** matches no file"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, errs := Assign(config.Rules{Present: true, Layers: tc.layers}, tc.rels)
			if !slices.Equal(errs, tc.errs) {
				t.Fatalf("errors = %q, want %q", errs, tc.errs)
			}
			for _, rel := range tc.rels {
				got, ok := m.Of(rel)
				want, wantOK := tc.want[rel]
				if ok != wantOK || got.Name != want {
					t.Errorf("Of(%q) = %q, %v; want %q, %v", rel, got.Name, ok, want, wantOK)
				}
			}
		})
	}
}
