package guard

import (
	"os"
	"strings"
	"testing"
)

// BenchmarkDecideAgainstTheRealRegistry measures what the barrier costs on
// a registry of real size -- the ~40 ms the reference's guard spent above the Go
// start floor is decided here, not in the process start. The registry is
// not in the repository: point LOOMUX_BENCH_REGISTRY at a copy of the
// state directory (`%LOCALAPPDATA%\brain`) in a temporary place, and the
// benchmark skips itself when nobody did.
func BenchmarkDecideAgainstTheRealRegistry(b *testing.B) {
	stateDir := os.Getenv("LOOMUX_BENCH_REGISTRY")
	if stateDir == "" {
		b.Skip("set LOOMUX_BENCH_REGISTRY to a copy of the brain state directory")
	}
	target := os.Getenv("LOOMUX_BENCH_TARGET")
	if target == "" {
		b.Skip("set LOOMUX_BENCH_TARGET to a file inside a registered area")
	}
	payload := map[string]any{
		"tool_name":  "Edit",
		"tool_input": map[string]any{"file_path": target},
	}
	// A refusal that names the registry itself would measure the error
	// path instead of the decision -- the numbers would be honest about
	// nothing.
	if reason, refused := Decide(payload, stateDir); refused &&
		strings.Contains(reason, "registry.toml") {
		b.Fatalf("the registry was not read: %s", reason)
	}
	for b.Loop() {
		Decide(payload, stateDir)
	}
}
