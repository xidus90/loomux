package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelSettingsDefaultWithoutAFile(t *testing.T) {
	s, err := ReadModelSettings(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if s.Enabled || s.Endpoint != DefaultModelEndpoint || s.Name != DefaultModelName || s.Temperature != 0 {
		t.Fatalf("%+v", s)
	}
	for _, role := range ModelRoleNames() {
		if !s.Roles[role] {
			t.Fatalf("role %s off by default", role)
		}
	}
}

func TestModelSettingsReadTheGlobalFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[model]\nenabled = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ReadModelSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Enabled || s.Endpoint != DefaultModelEndpoint {
		t.Fatalf("%+v", s)
	}
}

func TestModelSettingsReadTheBlock(t *testing.T) {
	s, err := ParseModelSettings("c.toml", "[model]\nenabled = true\nendpoint = \"http://localhost:1\"\nname = \"m\"\ntemperature = 1\n[model.roles]\npropose = true\nplace = false\n")
	if err != nil {
		t.Fatal(err)
	}
	if !s.Enabled || s.Endpoint != "http://localhost:1" || s.Name != "m" || s.Temperature != 1 {
		t.Fatalf("%+v", s)
	}
	// Once roles is said, an unnamed role is off.
	if !s.Roles["propose"] || s.Roles["place"] || s.Roles["describe"] {
		t.Fatalf("roles %v", s.Roles)
	}
}

func TestModelSettingsTakeAFloatTemperature(t *testing.T) {
	s, err := ParseModelSettings("c.toml", "[model]\ntemperature = 0.7\n")
	if err != nil {
		t.Fatal(err)
	}
	if s.Temperature != 0.7 {
		t.Fatalf("%+v", s)
	}
}

func TestModelSettingsTakeBothEndsOfTheTemperatureRange(t *testing.T) {
	for text, want := range map[string]float64{
		"[model]\ntemperature = 0.0\n": 0,
		"[model]\ntemperature = 2.0\n": 2,
	} {
		s, err := ParseModelSettings("c.toml", text)
		if err != nil || s.Temperature != want {
			t.Errorf("%q: %+v, %v", text, s, err)
		}
	}
}

func TestModelSettingsIgnoreUnknownKeys(t *testing.T) {
	if _, err := ParseModelSettings("c.toml", "[model]\nkeep_alive = 5\n"); err != nil {
		t.Fatal(err)
	}
}

func TestModelSettingsRefuse(t *testing.T) {
	for text, want := range map[string]string{
		"model = 5":                       "[model] must be a table",
		"[model]\nenabled = \"yes\"":      "[model] enabled must be a boolean",
		"[model]\nendpoint = \"\"":        "[model] endpoint must be a non-empty string",
		"[model]\nname = 3":               "[model] name must be a non-empty string",
		"[model]\ntemperature = true":     "[model] temperature must be a number",
		"[model]\ntemperature = 2.5":      "[model] temperature must lie between 0 and 2",
		"[model]\ntemperature = -0.1":     "[model] temperature must lie between 0 and 2",
		"[model]\ntemperature = nan":      "[model] temperature must lie between 0 and 2",
		"[model]\nroles = 1":              "[model] roles must be a table",
		"[model.roles]\nsummarize = true": `[model] roles has unknown "summarize"`,
		"[model.roles]\npropose = \"on\"": "[model] roles.propose must be a boolean",
		"[model":                          "not valid TOML",
	} {
		_, err := ParseModelSettings("c.toml", text)
		if err == nil || !strings.Contains(err.Error(), want) || !strings.HasPrefix(err.Error(), "c.toml: ") {
			t.Errorf("%q: %v, want %q", text, err, want)
		}
	}
}

func TestModelSettingsTreatADirectoryAsNoFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A directory of that name declares nothing, as in ReadAreaManifest.
	if _, err := ReadModelSettings(dir); err != nil {
		t.Fatal(err)
	}
}

func TestModelSettingsNameAFileThatCannotBeRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("[model]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := errors.New("broken")
	saved := readModelFile
	readModelFile = func(string) ([]byte, error) { return nil, broken }
	t.Cleanup(func() { readModelFile = saved })
	_, err := ReadModelSettings(dir)
	if !errors.Is(err, broken) || !strings.HasPrefix(err.Error(), path+": cannot be read") {
		t.Fatalf("%v", err)
	}
}

func TestNarrowedOffBeatsOnAndRolesOnlyShrink(t *testing.T) {
	base := ModelSettings{Enabled: true, Roles: map[string]bool{"propose": true, "place": true}}
	off, on := false, true
	if base.Narrowed(&Manifest{ModelEnabled: &off}).RoleOn("propose") {
		t.Fatal("an area switched the model off and it stayed on")
	}
	if !base.Narrowed(&Manifest{ModelEnabled: &on}).RoleOn("propose") {
		t.Fatal("an area saying on switched it off")
	}
	globalOff := ModelSettings{Roles: base.Roles}
	if globalOff.Narrowed(&Manifest{ModelEnabled: &on}).RoleOn("propose") {
		t.Fatal("an area switched on what is off globally")
	}
	narrowed := base.Narrowed(&Manifest{ModelRoles: map[string]bool{"place": true, "describe": true}})
	if narrowed.RoleOn("propose") || !narrowed.RoleOn("place") || narrowed.RoleOn("describe") {
		t.Fatalf("roles %v", narrowed.Roles)
	}
	if !base.Narrowed(&Manifest{}).RoleOn("propose") {
		t.Fatal("an area that says nothing narrowed the roles")
	}
}

func TestTheDeclarationCarriesTheModelBlock(t *testing.T) {
	m, err := declaration(map[string]any{
		"area":  map[string]any{"scope": "s"},
		"model": map[string]any{"enabled": false, "roles": map[string]any{"propose": true, "place": false}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.ModelEnabled == nil || *m.ModelEnabled || !m.ModelRoles["propose"] || m.ModelRoles["place"] {
		t.Fatalf("%v %v", m.ModelEnabled, m.ModelRoles)
	}
	m, _ = declaration(map[string]any{"area": map[string]any{"scope": "s"}})
	if m.ModelEnabled != nil || m.ModelRoles != nil {
		t.Fatal("an unsaid block must stay unsaid")
	}
}

func TestTheGlobalKeysAreTheOnesTheReaderReads(t *testing.T) {
	want := []string{"enabled", "endpoint", "name", "roles", "temperature"}
	if got := GlobalModelKeys(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatal(got)
	}
}
