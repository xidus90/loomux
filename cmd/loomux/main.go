package main

import (
	"os"

	"github.com/xidus90/loomux/internal/cli"
)

//coverage:exempt process entry; every decision lives in cli.Run, which is tested
func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
