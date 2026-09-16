// Command qmd is the fake qmd the stage 1b-1 recordings put on PATH in front
// of the real one. It is built for recording only and never shipped.
//
// Its directory begins with an underscore so that `./...` does not list it:
// `go install ./...` would otherwise drop a `qmd` into GOBIN, and a GOBIN that
// comes first on PATH would hand every later search to this fake. Recording
// builds it by its explicit path, which still works.
package main

import (
	"os"

	"github.com/xidus90/loomux/internal/dev/fakeqmd"
)

//coverage:exempt process entry; every decision lives in fakeqmd.Main, which is tested
func main() {
	os.Exit(fakeqmd.Main(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}
