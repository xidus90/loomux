package benchreport

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// cpuName is a seam: the processor's name comes from a different place on
// every system, and a test must not depend on the machine it runs on.
var cpuName = processorName

// Current is the environment this process runs in; the caller adds what
// only it knows (qmd, models, profile, port).
func Current(loomux string) Environment {
	return Environment{OS: runtime.GOOS, Arch: runtime.GOARCH, CPU: cpuName(),
		Go: runtime.Version(), Loomux: loomux}
}

// Backbone is the compute backbone this process tells qmd to run on: the
// GPU QMD_LLAMA_GPU names, "cpu" under QMD_FORCE_CPU, and "default" when
// neither is set and qmd picks its own. A daemon that is already running
// keeps the backbone it started with.
func Backbone(getenv func(string) string) string {
	switch {
	case getenv("QMD_LLAMA_GPU") != "":
		return getenv("QMD_LLAMA_GPU")
	case getenv("QMD_FORCE_CPU") != "":
		return "cpu"
	}
	return "default"
}

// processorName reads the processor's name where the system hands it out
// without a subprocess, and the logical CPU count otherwise.
//
//coverage:exempt reads the host's processor name from the environment or /proc, which differs per machine
func processorName() string {
	if name := os.Getenv("PROCESSOR_IDENTIFIER"); name != "" {
		return name
	}
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if key, value, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(key) == "model name" {
				return strings.TrimSpace(value)
			}
		}
	}
	return fmt.Sprintf("%d logical CPUs", runtime.NumCPU())
}
