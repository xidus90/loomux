package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/selfupdate"
)

// selfUpdateRun is the seam the command and serve's pass are tested through:
// the real one reaches GitHub.
var selfUpdateRun = selfupdate.Run

const upgradeUsage = "usage: loomux upgrade [--beta | --stable | --version <x.y.z>]"

// upgradeCommand is `loomux upgrade`: one pass by hand, the same one serve
// runs daily, or one aimed at the newest beta, the newest stable release or
// a single version.
func upgradeCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	beta := fs.Bool("beta", false, "")
	stable := fs.Bool("stable", false, "")
	pin := fs.String("version", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || choices(*beta, *stable, *pin != "") > 1 ||
		(*pin != "" && (strings.Contains(*pin, " ") || !selfupdate.IsVersion(*pin))) {
		fmt.Fprintln(stderr, upgradeUsage)
		return 2
	}
	o := selfUpdateOptions(selfupdate.SourceCLI)
	o.Pin = strings.TrimPrefix(*pin, "v")
	switch {
	case *beta:
		o.Mode = selfupdate.ModeBeta
	case *stable:
		o.Mode = selfupdate.ModeStable
	}
	res := selfUpdateRun(context.Background(), o)
	if res.StatusErr != nil {
		fmt.Fprintf(stderr, "loomux upgrade: record update.json: %v\n", res.StatusErr)
	}
	switch res.Outcome {
	case selfupdate.Current:
		reportMarker(stderr, res)
		fmt.Fprintf(stdout, "already current (v%s)\n", res.Version)
		return 0
	case selfupdate.Updated:
		reportMarker(stderr, res)
		fmt.Fprintf(stdout, "updated to v%s; serve switches on the next bridge\n", res.Version)
		return 0
	case selfupdate.Skipped:
		fmt.Fprintf(stderr, "loomux upgrade: skipped: %v\n", res.Err)
		return 2
	}
	fmt.Fprintf(stderr, "loomux upgrade: %v\n", res.Err)
	return 1
}

// choices counts the flags that each pick the release.
func choices(set ...bool) int {
	n := 0
	for _, s := range set {
		if s {
			n++
		}
	}
	return n
}

// reportMarker names a problem with the channel marker of a pass that
// otherwise succeeded; the binary is in place, only the channel is not.
func reportMarker(stderr io.Writer, res selfupdate.Result) {
	if res.Err != nil {
		fmt.Fprintf(stderr, "loomux upgrade: %v\n", res.Err)
	}
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
