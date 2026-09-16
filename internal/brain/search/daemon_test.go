package search_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
)

func TestParseBackbone(t *testing.T) {
	tests := []struct {
		input   string
		want    search.Backbone
		wantErr bool
	}{
		{"cuda", search.BackboneCUDA, false},
		{"CUDA", search.BackboneCUDA, false},
		{"", search.BackboneCUDA, false},
		{"vulkan", search.BackboneVulkan, false},
		{"cpu", search.BackboneCPU, false},
		{"metal", "", true},
		{"unknown", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := search.ParseBackbone(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.input)
				}
				if !strings.Contains(err.Error(), "unknown backbone") {
					t.Errorf("expected 'unknown backbone' error, got: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for %q: %v", tc.input, err)
				}
				if got != tc.want {
					t.Errorf("got %q, want %q", got, tc.want)
				}
			}
		})
	}
}

func TestBackboneEnv(t *testing.T) {
	tests := []struct {
		backbone search.Backbone
		want     map[string]string
	}{
		{search.BackboneVulkan, map[string]string{"QMD_LLAMA_GPU": "vulkan"}},
		{search.BackboneCPU, map[string]string{"QMD_FORCE_CPU": "1"}},
		{search.BackboneCUDA, map[string]string{}},
		{"unrecognized", map[string]string{}},
	}

	for _, tc := range tests {
		t.Run(string(tc.backbone), func(t *testing.T) {
			got := search.BackboneEnv(tc.backbone)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got env %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStartDaemonWith(t *testing.T) {
	var capturedArgv []string
	var capturedEnv []string

	mockLauncher := func(tool string) ([]string, error) {
		if tool != "qmd" {
			return nil, errors.New("expected qmd tool")
		}
		return []string{"node", "qmd.js"}, nil
	}

	mockSpawner := func(argv []string, env []string) error {
		capturedArgv = argv
		capturedEnv = env
		return nil
	}

	env := map[string]string{"QMD_LLAMA_GPU": "vulkan"}
	err := search.StartDaemonWith(env, 9000, mockLauncher, mockSpawner)
	if err != nil {
		t.Fatalf("unexpected StartDaemonWith error: %v", err)
	}

	expectedArgv := []string{"node", "qmd.js", "mcp", "--http", "--daemon", "--port", "9000"}
	if !reflect.DeepEqual(capturedArgv, expectedArgv) {
		t.Errorf("got argv %v, want %v", capturedArgv, expectedArgv)
	}

	foundEnv := false
	for _, e := range capturedEnv {
		if e == "QMD_LLAMA_GPU=vulkan" {
			foundEnv = true
			break
		}
	}
	if !foundEnv {
		t.Errorf("expected QMD_LLAMA_GPU=vulkan in captured env: %v", capturedEnv)
	}
}

func TestStartDaemonWith_DefaultPort(t *testing.T) {
	var capturedArgv []string

	mockLauncher := func(tool string) ([]string, error) {
		return []string{"qmd"}, nil
	}

	mockSpawner := func(argv []string, env []string) error {
		capturedArgv = argv
		return nil
	}

	err := search.StartDaemonWith(nil, 0, mockLauncher, mockSpawner)
	if err != nil {
		t.Fatalf("unexpected StartDaemonWith error: %v", err)
	}

	expectedArgv := []string{"qmd", "mcp", "--http", "--daemon", "--port", "8765"}
	if !reflect.DeepEqual(capturedArgv, expectedArgv) {
		t.Errorf("got argv %v, want %v", capturedArgv, expectedArgv)
	}
}

func TestStartDaemonWith_LauncherError(t *testing.T) {
	mockLauncher := func(tool string) ([]string, error) {
		return nil, errors.New("qmd not on PATH")
	}

	err := search.StartDaemonWith(nil, 8765, mockLauncher, nil)
	if err == nil {
		t.Fatal("expected launcher error, got nil")
	}
	if !strings.Contains(err.Error(), "qmd not on PATH") {
		t.Errorf("expected 'qmd not on PATH' error, got: %v", err)
	}
}

func TestStartDaemonWith_SpawnerError(t *testing.T) {
	mockLauncher := func(tool string) ([]string, error) {
		return []string{"qmd"}, nil
	}
	mockSpawner := func(argv []string, env []string) error {
		return errors.New("cannot spawn daemon")
	}

	err := search.StartDaemonWith(nil, 8765, mockLauncher, mockSpawner)
	if err == nil {
		t.Fatal("expected spawner error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot spawn daemon") {
		t.Errorf("expected 'cannot spawn daemon' error, got: %v", err)
	}
}

func TestDefaultSpawner(t *testing.T) {
	err := search.DefaultSpawner([]string{"go", "version"}, nil)
	if err != nil {
		t.Fatalf("unexpected DefaultSpawner error: %v", err)
	}
}

func TestStartDaemon(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := search.StartDaemon(nil, 0)
	if err == nil {
		t.Fatal("expected error when qmd is absent from PATH, got nil")
	}
}

func TestStartDaemonWith_DefaultSpawner(t *testing.T) {
	mockLauncher := func(tool string) ([]string, error) {
		return []string{"go", "version"}, nil
	}
	err := search.StartDaemonWith(nil, 8765, mockLauncher, nil)
	if err != nil {
		t.Fatalf("unexpected error with default spawner: %v", err)
	}
}

func TestStartDaemonWith_DefaultLauncher(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	mockSpawner := func(argv []string, env []string) error {
		return nil
	}
	err := search.StartDaemonWith(nil, 8765, nil, mockSpawner)
	if err == nil {
		t.Fatal("expected error with empty PATH and default launcher, got nil")
	}
}
