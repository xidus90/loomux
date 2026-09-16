package privacy_test

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
)

// registered is one [[area]] of a test registry. manifestName "" writes no
// manifest; otherwise the declaration goes where config.ManifestDir looks for
// it -- into the area for a writable area, below <legacy>/areas/<flat> for a
// read-only one.
type registered struct {
	scope, mode, manifestName string
	readOnly                  bool
}

// buildWorld writes a registry into <root>/state and the manifests of its
// areas, and answers the registry and legacy directories.
func buildWorld(t *testing.T, areas ...registered) (registryDir, legacyDir string) {
	t.Helper()
	root := t.TempDir()
	registryDir = filepath.Join(root, "state")
	legacyDir = filepath.Join(root, "legacy")
	var registry strings.Builder
	for i, a := range areas {
		path := filepath.ToSlash(filepath.Join(root, fmt.Sprintf("repo-%d", i)))
		fmt.Fprintf(&registry, "[[area]]\nscope = %q\npath = %q\nreadonly = %v\n\n", a.scope, path, a.readOnly)
		if a.manifestName == "" {
			continue
		}
		dir := config.ManifestDir(config.Area{Scope: a.scope, Path: path, ReadOnly: a.readOnly}, legacyDir)
		writeFile(t, filepath.Join(dir, a.manifestName), fmt.Sprintf("[area]\nscope = %q\n\n[privacy]\nmode = %q\n", a.scope, a.mode))
	}
	writeFile(t, filepath.Join(registryDir, "registry.toml"), registry.String())
	return registryDir, legacyDir
}

func scopesOf(areas []privacy.VisibleArea) string {
	scopes := make([]string, 0, len(areas))
	for _, entry := range areas {
		scopes = append(scopes, entry.Area.Scope)
	}
	return strings.Join(scopes, ",")
}

// twoAreas lists zeta before project/alpha, the reverse of sorted order, so a
// test can tell registry order from sorting. project/alpha is read-only and
// local_only, so its manifest can only be found through the legacy directory.
func twoAreas(t *testing.T) (registryDir, legacyDir string) {
	return buildWorld(t,
		registered{scope: "zeta", mode: "manual_cloud", manifestName: ".ultra-brain/config.toml"},
		registered{scope: "project/alpha", mode: "local_only", manifestName: ".brain.toml", readOnly: true},
	)
}

func TestVisibleAreasKeepsRegistryOrderAndHidesLocalOnlyOnCloud(t *testing.T) {
	registryDir, legacyDir := twoAreas(t)
	local, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelLocal)
	if err != nil {
		t.Fatal(err)
	}
	if got := scopesOf(local); got != "zeta,project/alpha" {
		t.Fatalf("local: %s", got)
	}
	if m := local[1].Manifest; m == nil || m.PrivacyMode != "local_only" || !local[1].Area.ReadOnly {
		t.Fatalf("project/alpha: %+v", local[1])
	}
	cloud, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelCloud)
	if err != nil {
		t.Fatal(err)
	}
	if got := scopesOf(cloud); got != "zeta" {
		t.Fatalf("cloud: %s", got)
	}
}

func TestVisibleAreasNamesOneScope(t *testing.T) {
	registryDir, legacyDir := twoAreas(t)
	got, err := privacy.VisibleAreas(registryDir, legacyDir, "project/alpha", privacy.ChannelLocal)
	if err != nil {
		t.Fatal(err)
	}
	if scopesOf(got) != "project/alpha" {
		t.Fatalf("got %s", scopesOf(got))
	}
}

// One message for an unknown and a hidden scope, naming only what the channel
// sees: telling a cloud caller that the area exists would disclose what
// `local_only` hides (`_unknown_scope`, src/brain/core.py:266-275).
func TestVisibleAreasRefusesAnUnknownScopeNamingTheVisibleOnes(t *testing.T) {
	registryDir, legacyDir := twoAreas(t)
	for _, tc := range []struct {
		scope string
		ch    privacy.Channel
		want  string
	}{
		{"zz", privacy.ChannelLocal, "unknown scope 'zz'; known scopes are: project/alpha, zeta"},
		{"project/alpha", privacy.ChannelCloud, "unknown scope 'project/alpha'; known scopes are: zeta"},
	} {
		got, err := privacy.VisibleAreas(registryDir, legacyDir, tc.scope, tc.ch)
		if err == nil || err.Error() != tc.want || got != nil {
			t.Errorf("%s on %s: got %v, %v; want %q", tc.scope, tc.ch, got, err, tc.want)
		}
	}
}

// Python reads the manifest of every registered area before it filters by
// scope, so an area without a declaration fails a call about another area.
func TestVisibleAreasStopsAtTheFirstUnusableManifest(t *testing.T) {
	registryDir, legacyDir := buildWorld(t,
		registered{scope: "a", mode: "manual_cloud", manifestName: ".brain.toml"},
		registered{scope: "b"},
	)
	got, err := privacy.VisibleAreas(registryDir, legacyDir, "a", privacy.ChannelLocal)
	if !errors.Is(err, config.ErrNoManifest) || got != nil {
		t.Fatalf("got %v, %v; want config.ErrNoManifest", got, err)
	}
}

func TestVisibleAreasPassesTheRegistryErrorOn(t *testing.T) {
	got, err := privacy.VisibleAreas(t.TempDir(), t.TempDir(), "all", privacy.ChannelLocal)
	if !errors.Is(err, fs.ErrNotExist) || got != nil {
		t.Fatalf("got %v, %v; want a missing registry", got, err)
	}
}

func TestVisibleAreasOfAnEmptyRegistry(t *testing.T) {
	registryDir, legacyDir := buildWorld(t)
	all, err := privacy.VisibleAreas(registryDir, legacyDir, "all", privacy.ChannelLocal)
	if err != nil || len(all) != 0 {
		t.Fatalf("got %v, %v", all, err)
	}
	_, err = privacy.VisibleAreas(registryDir, legacyDir, "x", privacy.ChannelLocal)
	if want := "unknown scope 'x'; known scopes are: "; err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}

func TestSingle(t *testing.T) {
	areas := []privacy.VisibleArea{
		{Area: config.Area{Scope: "b", Path: "B"}},
		{Area: config.Area{Scope: "a", Path: "A"}},
	}
	got, err := privacy.Single(areas, "a")
	if err != nil || got.Area.Path != "A" {
		t.Fatalf("got %+v, %v", got, err)
	}
	got, err = privacy.Single(areas, "c")
	if want := "unknown scope 'c'; known scopes are: a, b"; err == nil || err.Error() != want || got.Area.Scope != "" {
		t.Fatalf("got %+v, %v; want %q", got, err, want)
	}
}

// Measured against `_unknown_scope` on 2026-09-15 under Python 3.14.7.
func TestUnknownScopeQuotesLikePython(t *testing.T) {
	for _, tc := range []struct {
		scope string
		areas []privacy.VisibleArea
		want  string
	}{
		{"zz", []privacy.VisibleArea{{Area: config.Area{Scope: "zeta"}}, {Area: config.Area{Scope: "alpha"}}}, "unknown scope 'zz'; known scopes are: alpha, zeta"},
		{"it's", nil, `unknown scope "it's"; known scopes are: `},
		{`say "hi"`, []privacy.VisibleArea{{Area: config.Area{Scope: "b"}}}, `unknown scope 'say "hi"'; known scopes are: b`},
	} {
		if got := privacy.UnknownScope(tc.scope, tc.areas).Error(); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}
