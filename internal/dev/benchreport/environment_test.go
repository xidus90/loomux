package benchreport

import (
	"runtime"
	"testing"
)

func TestCurrentNamesTheRuntime(t *testing.T) {
	env := Current("3.2.0")
	if env.OS != runtime.GOOS || env.Arch != runtime.GOARCH || env.Go != runtime.Version() || env.Loomux != "3.2.0" {
		t.Fatalf("env = %+v", env)
	}
}

func TestCurrentTakesTheProcessorFromTheSeam(t *testing.T) {
	old := cpuName
	t.Cleanup(func() { cpuName = old })
	cpuName = func() string { return "test cpu" }
	if env := Current("dev"); env.CPU != "test cpu" {
		t.Fatalf("CPU = %q", env.CPU)
	}
}

func TestProcessorNameIsWhatWindowsHandsOut(t *testing.T) {
	// Windows names the processor in PROCESSOR_IDENTIFIER; where that is
	// set, it is the name, whatever else the machine offers.
	t.Setenv("PROCESSOR_IDENTIFIER", "Test Family 6 Model 1")
	if got := processorName(); got != "Test Family 6 Model 1" {
		t.Fatalf("processorName() = %q", got)
	}
}

func TestBackboneIsWhatQmdIsToldToRunOn(t *testing.T) {
	for _, c := range []struct {
		gpu, cpu, want string
	}{
		{"vulkan", "", "vulkan"},
		{"cuda", "1", "cuda"}, // both set: the named GPU is recorded
		{"", "1", "cpu"},
		{"", "", "default"},
	} {
		env := map[string]string{"QMD_LLAMA_GPU": c.gpu, "QMD_FORCE_CPU": c.cpu}
		if got := Backbone(func(key string) string { return env[key] }); got != c.want {
			t.Errorf("gpu %q, cpu %q: backbone %q, want %q", c.gpu, c.cpu, got, c.want)
		}
	}
}
