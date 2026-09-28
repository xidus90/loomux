package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/selfupdate"
)

// selfUpdateRun is the seam the command and serve's pass are tested through:
// the real one reaches GitHub.
var selfUpdateRun = selfupdate.Run

// upgradeCommand is `loomux upgrade`: one pass by hand, the same one
// serve runs daily.
func upgradeCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: loomux upgrade")
		return 2
	}
	res := selfUpdateRun(context.Background(), selfUpdateOptions(selfupdate.SourceCLI))
	if res.StatusErr != nil {
		fmt.Fprintf(stderr, "loomux upgrade: record update.json: %v\n", res.StatusErr)
	}
	switch res.Outcome {
	case selfupdate.Current:
		fmt.Fprintf(stdout, "already current (v%s)\n", res.Version)
		return 0
	case selfupdate.Updated:
		fmt.Fprintf(stdout, "updated to v%s; serve switches on the next bridge\n", res.Version)
		return 0
	case selfupdate.Skipped:
		fmt.Fprintf(stderr, "loomux upgrade: skipped: %v\n", res.Err)
		return 2
	}
	fmt.Fprintf(stderr, "loomux upgrade: %v\n", res.Err)
	return 1
}

// selfUpdateOptions describe this process to the updater. An executable the
// OS cannot name stays empty, and the pass then skips as not canonical.
// source says who asked for the pass.
func selfUpdateOptions(source string) selfupdate.Options {
	exe, _ := os.Executable()
	return selfupdate.Options{
		Source:     source,
		StateDir:   config.StateDir(),
		Executable: exe,
		Version:    Version,
		Channel:    Channel,
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		Run:        selfupdate.ExecRunner,
		Now:        time.Now,
	}
}
