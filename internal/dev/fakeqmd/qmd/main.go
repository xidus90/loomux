// Command qmd is the fake qmd the stage 1b-1 recordings put on PATH in front
// of the real one. It is built for recording only and never shipped.
package main

import (
	"os"

	"github.com/xidus90/loomux/internal/dev/fakeqmd"
)

//coverage:exempt process entry; every decision lives in fakeqmd.Main, which is tested
func main() {
	os.Exit(fakeqmd.Main(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}
