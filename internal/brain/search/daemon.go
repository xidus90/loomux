package search

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/xidus90/loomux/internal/config"
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
// It takes no lock and is nobody's entry point: a starter goes through EnsureDaemon or the
// connect of a QmdMcpPort, which probe and start under the shared qmd lock.
// A backbone the user already chose in the environment wins: the port's own backbone
// variables are then not appended (stage 1b-1 spec; qmd_mcp.py overrides the user).
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
	vars, _ := resolveVars(env, os.LookupEnv)
	for k, v := range vars {
		mergedEnv = append(mergedEnv, fmt.Sprintf("%s=%s", k, v))
	}
	return spawner(argv, mergedEnv)
}

// ResolveBackbone is the backbone's order of precedence for one qmd process: a
// QMD_LLAMA_GPU or QMD_FORCE_CPU the user set wins, and the process then gets
// nothing on top of the caller's environment (byUser); otherwise it gets the
// variables of configured, the machine's [search] backbone or the default.
func ResolveBackbone(configured Backbone, lookupEnv func(string) (string, bool)) (vars map[string]string, byUser bool) {
	return resolveVars(BackboneEnv(configured), lookupEnv)
}

// resolveVars is that order over a backbone's variables already worked out,
// the form in which a daemon start receives them.
func resolveVars(vars map[string]string, lookupEnv func(string) (string, bool)) (map[string]string, bool) {
	if backboneChosenByUser(lookupEnv) {
		return map[string]string{}, true
	}
	return vars, false
}

// ConfiguredBackbone is the machine's [search] backbone from the global file of
// stateDir, DefaultBackbone where it says nothing. The reader admits only the
// three names, so the conversion needs no second check.
func ConfiguredBackbone(stateDir string) (Backbone, error) {
	settings, err := config.ReadSearchSettings(stateDir)
	if err != nil {
		return "", err
	}
	return Backbone(settings.Backbone), nil
}

// backboneChosenByUser reports whether QMD_LLAMA_GPU or QMD_FORCE_CPU is set at all, an
// empty value included: both are the user's say over the backbone.
func backboneChosenByUser(lookupEnv func(string) (string, bool)) bool {
	_, gpu := lookupEnv("QMD_LLAMA_GPU")
	_, cpu := lookupEnv("QMD_FORCE_CPU")
	return gpu || cpu
}
