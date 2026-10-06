package privacy_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
)

// registered is one [[area]] of a test registry. manifestName "" writes no
// manifest; otherwise the declaration goes where config.ManifestDir looks for
// it -- into the area for a writable area, below <state>/areas/<flat> for a
// read-only one.
type registered struct {
	scope, mode, manifestName string
	readOnly                  bool
}

// buildWorld writes a registry into <root>/state and the manifests of its
// areas, and answers the state directory.
func buildWorld(t *testing.T, areas ...registered) (registryDir string) {
	t.Helper()
	root := t.TempDir()
	registryDir = filepath.Join(root, "state")
	var registry strings.Builder
	for i, a := range areas {
		path := filepath.ToSlash(filepath.Join(root, fmt.Sprintf("repo-%d", i)))
		fmt.Fprintf(&registry, "[[area]]\nscope = %q\npath = %q\nreadonly = %v\n\n", a.scope, path, a.readOnly)
		if a.manifestName == "" {
			continue
		}
		dir := config.ManifestDir(config.Area{Scope: a.scope, Path: path, ReadOnly: a.readOnly}, registryDir)
		writeFile(t, filepath.Join(dir, a.manifestName), fmt.Sprintf("[area]\nscope = %q\n\n[privacy]\nmode = %q\n", a.scope, a.mode))
	}
	writeFile(t, filepath.Join(registryDir, "registry.toml"), registry.String())
	return registryDir
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
// local_only, so its manifest can only be found through the state directory.
func twoAreas(t *testing.T) (registryDir string) {
	return buildWorld(t,
		registered{scope: "zeta", mode: "manual_cloud", manifestName: ".loomux/config.toml"},
		registered{scope: "project/alpha", mode: "local_only", manifestName: ".loomux/config.toml", readOnly: true},
	)
}

func TestVisibleAreasKeepsRegistryOrderAndHidesLocalOnlyOnCloud(t *testing.T) {
	registryDir := twoAreas(t)
	local, err := privacy.VisibleAreas(registryDir, "all", privacy.ChannelLocal)
	if err != nil {
		t.Fatal(err)
	}
	if got := scopesOf(local); got != "zeta,project/alpha" {
		t.Fatalf("local: %s", got)
	}
	if m := local[1].Manifest; m == nil || m.PrivacyMode != "local_only" || !local[1].Area.ReadOnly {
		t.Fatalf("project/alpha: %+v", local[1])
	}
	cloud, err := privacy.VisibleAreas(registryDir, "all", privacy.ChannelCloud)
	if err != nil {
		t.Fatal(err)
	}
	if got := scopesOf(cloud); got != "zeta" {
		t.Fatalf("cloud: %s", got)
	}
}

func TestVisibleAreasNamesOneScope(t *testing.T) {
	registryDir := twoAreas(t)
	got, err := privacy.VisibleAreas(registryDir, "project/alpha", privacy.ChannelLocal)
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
	registryDir := twoAreas(t)
	for _, tc := range []struct {
		scope string
		ch    privacy.Channel
		want  string
	}{
		{"zz", privacy.ChannelLocal, "unknown scope 'zz'; known scopes are: project/alpha, zeta"},
		{"project/alpha", privacy.ChannelCloud, "unknown scope 'project/alpha'; known scopes are: zeta"},
	} {
		got, err := privacy.VisibleAreas(registryDir, tc.scope, tc.ch)
		if err == nil || err.Error() != tc.want || got != nil {
			t.Errorf("%s on %s: got %v, %v; want %q", tc.scope, tc.ch, got, err, tc.want)
		}
	}
}

// Python reads the manifest of every registered area before it filters by
// scope, so an area without a declaration fails a call about another area.
func TestVisibleAreasStopsAtTheFirstUnusableManifest(t *testing.T) {
	registryDir := buildWorld(t,
		registered{scope: "a", mode: "manual_cloud", manifestName: ".loomux/config.toml"},
		registered{scope: "b"},
	)
	got, err := privacy.VisibleAreas(registryDir, "a", privacy.ChannelLocal)
	if !errors.Is(err, config.ErrNoManifest) || got != nil {
		t.Fatalf("got %v, %v; want config.ErrNoManifest", got, err)
	}
}

func TestVisibleAreasPassesTheRegistryErrorOn(t *testing.T) {
	got, err := privacy.VisibleAreas(t.TempDir(), "all", privacy.ChannelLocal)
	if !errors.Is(err, fs.ErrNotExist) || got != nil {
		t.Fatalf("got %v, %v; want a missing registry", got, err)
	}
}

func TestVisibleAreasOfAnEmptyRegistry(t *testing.T) {
	registryDir := buildWorld(t)
	all, err := privacy.VisibleAreas(registryDir, "all", privacy.ChannelLocal)
	if err != nil || len(all) != 0 {
		t.Fatalf("got %v, %v", all, err)
	}
	_, err = privacy.VisibleAreas(registryDir, "x", privacy.ChannelLocal)
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

func TestVisibleAreasRefusesADuplicateScope(t *testing.T) {
	registryDir := buildWorld(t,
		registered{scope: "a", mode: "manual_cloud", manifestName: ".loomux/config.toml"},
		registered{scope: "a", mode: "local_only", manifestName: ".loomux/config.toml"},
	)
	got, err := privacy.VisibleAreas(registryDir, "a", privacy.ChannelLocal)
	if err == nil || got != nil || !strings.HasSuffix(err.Error(), `[[area]] #2: duplicate scope "a" (first at #1)`) {
		t.Fatalf("got %v, %v; want the duplicate refused", got, err)
	}
}

func TestVisibleAreasRefusesAnAbsoluteInboxEvenOfAHiddenArea(t *testing.T) {
	// The reference checks every area's inbox while reading the registry,
	// before visibility is asked, so a local_only area cannot hide a broken
	// declaration from a cloud caller.
	root := t.TempDir()
	area := filepath.Join(root, "closed")
	inbox := filepath.ToSlash(filepath.Join(root, "in"))
	writeFile(t, filepath.Join(root, "state", "registry.toml"),
		fmt.Sprintf("[[area]]\nscope = \"closed\"\npath = %q\n", filepath.ToSlash(area)))
	writeFile(t, filepath.Join(area, ".loomux", "config.toml"),
		fmt.Sprintf("[area]\nscope = \"closed\"\n\n[privacy]\nmode = \"local_only\"\n\n[layout]\ninbox = %q\n", inbox))
	got, err := privacy.VisibleAreas(filepath.Join(root, "state"), "all", privacy.ChannelCloud)
	if err == nil || got != nil || !strings.Contains(err.Error(), "[layout] inbox must be relative to the area") {
		t.Fatalf("got %v, %v; want the inbox refused", got, err)
	}
}

// workspaceWorld registers project/a with a declaration, then project/ws,
// whose own config file holds policyBody -- `init --brain=none` leaves one
// with policy and no [area] -- and answers the state directory.
func workspaceWorld(t *testing.T, workspace bool, policyBody string) (registryDir string) {
	t.Helper()
	registryDir = buildWorld(t,
		registered{scope: "project/a", mode: "manual_cloud", manifestName: ".loomux/config.toml"})
	wsPath := filepath.ToSlash(filepath.Join(filepath.Dir(registryDir), "workspace"))
	writeFile(t, filepath.Join(wsPath, ".loomux", "config.toml"), policyBody)
	body, err := os.ReadFile(filepath.Join(registryDir, "registry.toml"))
	if err != nil {
		t.Fatal(err)
	}
	entry := fmt.Sprintf("[[area]]\nscope = \"project/ws\"\npath = %q\nworkspace = %v\n", wsPath, workspace)
	writeFile(t, filepath.Join(registryDir, "registry.toml"), string(body)+entry)
	return registryDir
}

func TestVisibleAreasLeavesOutAWorkspaceWithoutADeclaration(t *testing.T) {
	registryDir := workspaceWorld(t, true, "[verify]\n")
	for _, ch := range []privacy.Channel{privacy.ChannelLocal, privacy.ChannelCloud} {
		got, err := privacy.VisibleAreas(registryDir, "all", ch)
		if err != nil || scopesOf(got) != "project/a" {
			t.Fatalf("channel %v: got %q, %v", ch, scopesOf(got), err)
		}
		if len(got[0].Hidden) != 0 {
			t.Fatalf("channel %v: a workspace is no area to hide: %v", ch, got[0].Hidden)
		}
	}
	_, err := privacy.VisibleAreas(registryDir, "project/ws", privacy.ChannelLocal)
	if err == nil || !strings.Contains(err.Error(), "unknown scope 'project/ws'") {
		t.Fatalf("got %v; want the unknown scope", err)
	}
}

func TestVisibleAreasStillRefusesAnEntryWithoutDeclarationThatIsNoWorkspace(t *testing.T) {
	registryDir := workspaceWorld(t, false, "[verify]\n")
	got, err := privacy.VisibleAreas(registryDir, "all", privacy.ChannelLocal)
	if !errors.Is(err, config.ErrNoArea) || got != nil {
		t.Fatalf("got %v, %v; want config.ErrNoArea", got, err)
	}
}

func TestVisibleAreasStillRefusesABrokenDeclarationOfAWorkspace(t *testing.T) {
	registryDir := workspaceWorld(t, true, "[area\n")
	got, err := privacy.VisibleAreas(registryDir, "all", privacy.ChannelLocal)
	if err == nil || errors.Is(err, config.ErrNoManifest) || got != nil {
		t.Fatalf("got %v, %v; want the parse error", got, err)
	}
}

// A workspace whose declaration cannot be inspected is no absent one: left
// out, its path would drop from Hidden and a local_only workspace inside a
// visible area would be served to the cloud.
func TestVisibleAreasRefusesAWorkspaceWhoseDeclarationCannotBeInspected(t *testing.T) {
	registryDir := workspaceWorld(t, true, "[verify]\n")
	ws := filepath.ToSlash(filepath.Join(filepath.Dir(registryDir), "workspace"))
	registry := filepath.Join(registryDir, "registry.toml")
	body, err := os.ReadFile(registry)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, registry, strings.Replace(string(body), ws, ws+`\u0000x`, 1))
	got, err := privacy.VisibleAreas(registryDir, "all", privacy.ChannelCloud)
	if err == nil || config.IsUndeclared(err) || got != nil {
		t.Fatalf("got %v, %v; want the inspection error", got, err)
	}
}

// A workspace without any config file is left out the same way: both
// answers of "no declaration here" mean no brain area.
func TestVisibleAreasLeavesOutAWorkspaceWithoutAnyConfigFile(t *testing.T) {
	registryDir := workspaceWorld(t, true, "[verify]")
	ws := filepath.Join(filepath.Dir(registryDir), "workspace")
	if err := os.Remove(filepath.Join(ws, ".loomux", "config.toml")); err != nil {
		t.Fatal(err)
	}
	got, err := privacy.VisibleAreas(registryDir, "all", privacy.ChannelLocal)
	if err != nil || scopesOf(got) != "project/a" {
		t.Fatalf("got %q, %v", scopesOf(got), err)
	}
}
