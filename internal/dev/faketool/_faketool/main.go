// Command faketool stands in for every checking tool while the stage 2a check
// chain is recorded: copied onto PATH as uv.exe, ruff.exe, ctest.exe and the
// rest, it answers from the fixture LOOMUX_FAKE_TOOL_FIXTURE names. It is built
// for recording only and never shipped.
//
// Its directory begins with an underscore so that `./...` does not list it:
// `go install ./...` would otherwise drop a `faketool` into GOBIN. Recording
// builds it by its explicit path, which still works.
package main

import (
	"os"

	"github.com/xidus90/loomux/internal/dev/faketool"
)

//coverage:exempt process entry; every decision lives in faketool.Main, which is tested
func main() {
	cwd, _ := os.Getwd()
	os.Exit(faketool.Main(os.Args, os.Getenv, cwd, os.Stdout, os.Stderr))
}
