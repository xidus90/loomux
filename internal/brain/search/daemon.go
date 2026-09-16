package search

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Backbone names the compute backbone used by the search model.
type Backbone string

const (
	BackboneCUDA   Backbone = "cuda"
	BackboneVulkan Backbone = "vulkan"
	BackboneCPU    Backbone = "cpu"
)

// DefaultBackbone is CUDA for fastest warm response times.
const DefaultBackbone = BackboneCUDA

// ParseBackbone parses a backbone string into a Backbone enum value.
func ParseBackbone(s string) (Backbone, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "cuda", "":
		return BackboneCUDA, nil
	case "vulkan":
		return BackboneVulkan, nil
	case "cpu":
		return BackboneCPU, nil
	default:
		return "", fmt.Errorf("unknown backbone %q (expected cuda, vulkan, or cpu)", s)
	}
}

// BackboneEnv returns the environment variables required to select backbone.
func BackboneEnv(backbone Backbone) map[string]string {
	switch backbone {
	case BackboneVulkan:
		return map[string]string{"QMD_LLAMA_GPU": "vulkan"}
	case BackboneCPU:
		return map[string]string{"QMD_FORCE_CPU": "1"}
	default:
		return map[string]string{}
	}
}

// DaemonSpawner starts a background command.
type DaemonSpawner func(argv []string, env []string) error

// DefaultSpawner spawns a detached process without waiting for it.
func DefaultSpawner(argv []string, env []string) error {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = env
	cmd.SysProcAttr = detachAttrs()
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

// StartDaemonWith starts a detached qmd daemon using the provided launcher and spawner.
func StartDaemonWith(env map[string]string, port int, launcher func(string) ([]string, error), spawner DaemonSpawner) error {
	if launcher == nil {
		launcher = Launcher
	}
	if spawner == nil {
		spawner = DefaultSpawner
	}
	if port <= 0 {
		port = DefaultPort
	}
	baseCmd, err := launcher("qmd")
	if err != nil {
		return err
	}
	argv := append(baseCmd, "mcp", "--http", "--daemon", "--port", strconv.Itoa(port))
	mergedEnv := os.Environ()
	for k, v := range env {
		mergedEnv = append(mergedEnv, fmt.Sprintf("%s=%s", k, v))
	}
	return spawner(argv, mergedEnv)
}

// StartDaemon starts a detached qmd daemon on port with the given environment.
func StartDaemon(env map[string]string, port int) error {
	return StartDaemonWith(env, port, Launcher, DefaultSpawner)
}
