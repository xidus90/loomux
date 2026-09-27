// Package backbonetest holds the one test helper every package needs that
// asks which backbone qmd would run on: a test must not inherit the
// QMD_LLAMA_GPU or QMD_FORCE_CPU of the person running it.
package backbonetest

import (
	"os"
	"testing"
)

// Clear removes QMD_LLAMA_GPU and QMD_FORCE_CPU for the rest of t; t.Setenv
// restores what was there. The error of Unsetenv is left: t.Setenv has just
// set the variable, and a variable left set shows as present in the test.
func Clear(t testing.TB) {
	t.Helper()
	for _, key := range []string{"QMD_LLAMA_GPU", "QMD_FORCE_CPU"} {
		t.Setenv(key, "")
		_ = os.Unsetenv(key)
	}
}
