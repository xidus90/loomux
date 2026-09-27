package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseFlowSettingsReadsDefaultAndOverrides(t *testing.T) {
	doc := agentDoc(t, "[flow]\ndefault = \"example\"\noverrides = [\"dev-cycle\", \"review\"]\n")
	got, err := ParseFlowSettings("c.toml", doc)
	if err != nil {
		t.Fatal(err)
	}
	want := FlowSettings{Default: "example", Overrides: []string{"dev-cycle", "review"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if !got.Allows("review") || got.Allows("x") {
		t.Fatalf("Allows: review %v, x %v; want true, false", got.Allows("review"), got.Allows("x"))
	}
}

func TestParseFlowSettingsWithoutTheTableIsEmpty(t *testing.T) {
	got, err := ParseFlowSettings("c.toml", agentDoc(t, "[modules]\nhooks = true\n"))
	if err != nil || !reflect.DeepEqual(got, FlowSettings{}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestParseFlowSettingsRefuses(t *testing.T) {
	for name, tc := range map[string]struct{ text, want string }{
		"not a table":          {"flow = 3", "[flow] must be a table"},
		"unknown key":          {"[flow]\nname = \"x\"", `[flow] does not know "name"`},
		"default not a name":   {"[flow]\ndefault = \"Dev\"", "[flow] default must be a flow name"},
		"overrides not a list": {"[flow]\noverrides = \"x\"", "[flow] overrides must be a list of flow names"},
		"override not a name":  {"[flow]\noverrides = [\"a\", 3]", "[flow] overrides #2 must be a flow name"},
		"override twice":       {"[flow]\noverrides = [\"a\", \"a\"]", `[flow] overrides names "a" twice`},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseFlowSettings("c.toml", agentDoc(t, tc.text))
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.HasPrefix(err.Error(), "c.toml: ") {
				t.Fatalf("err = %v, want it to start with c.toml and hold %q", err, tc.want)
			}
		})
	}
}

func TestReadFlowSettings(t *testing.T) {
	root := t.TempDir()
	if got, err := ReadFlowSettings(root); err != nil || !reflect.DeepEqual(got, FlowSettings{}) {
		t.Fatalf("without a file: %+v, %v", got, err)
	}
	path := ManifestPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[flow]\ndefault = \"example\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFlowSettings(root)
	if err != nil || got.Default != "example" {
		t.Fatalf("got %+v, %v", got, err)
	}
	if err := os.WriteFile(path, []byte("[flow"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFlowSettings(root); err == nil || !strings.Contains(err.Error(), "not valid TOML") {
		t.Fatalf("broken TOML: %v", err)
	}
}

func TestReadFlowSettingsReportsAnUnreadableFile(t *testing.T) {
	root := t.TempDir()
	path := ManifestPath(root)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFlowSettings(root); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("a directory where the file should be: %v, want an error naming %s", err, path)
	}
}
