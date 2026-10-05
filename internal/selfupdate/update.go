package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/swap"
)

// DevVersion is what a plain go build reports (cli.Version's default). Such a
// binary is someone's work in progress and is never replaced -- except at the
// canonical place, where no work in progress belongs.
const DevVersion = "0.0.0-dev"

// Options describe the running binary to an update pass. Everything the pass
// would otherwise ask the process for is a field, so a test can be any
// platform and any version; the caller passes cli.Version and cli.Channel,
// because this package may not import cli.
type Options struct {
	Source     string
	StateDir   string
	Executable string
	Version    string
	Channel    string
	GOOS       string
	GOARCH     string
	Run        Runner
	Now        func() time.Time
}

// Result is what a pass came to. Version is the release installed, or, when
// it is current, the running version, or the installed file's when only that
// one is at or above the release. StatusErr is a failure to write update.json, apart
// from the pass's own outcome: a binary swapped in stays swapped in even when
// the record of it could not be written.
type Result struct {
	Outcome   Outcome
	Version   string
	Err       error
	StatusErr error
}

// Run is one update pass, recorded in update.json unless nothing is worth
// recording: another pass held the lock, the pass was cancelled (serve
// stopping cuts it short, which is no failure of the update), or a pass by
// hand skipped, which says nothing about the machine-wide binary and would
// only hide serve's record until serve's next pass.
func Run(ctx context.Context, o Options) Result {
	res := run(ctx, o)
	if res.Outcome == Busy || ctx.Err() != nil || (res.Outcome == Skipped && o.Source == SourceCLI) {
		return res
	}
	st := Status{
		Source:     o.Source,
		CheckedAt:  o.Now().UTC(),
		Executable: o.Executable,
		Running:    o.Version,
		Result:     res.Outcome,
		Version:    res.Version,
	}
	if res.Err != nil {
		st.Error = res.Err.Error()
	}
	res.StatusErr = WriteStatus(o.StateDir, st)
	return res
}

// run decides and acts. A failure anywhere leaves loomux.exe where it was.
func run(ctx context.Context, o Options) Result {
	if o.GOOS != "windows" {
		return Result{Outcome: Skipped, Err: errors.New("updating runs on Windows only")}
	}
	dev := o.Version == DevVersion
	canonical := Canonical(o.StateDir)
	if !IsCanonical(o.Executable, o.StateDir) {
		if dev {
			return Result{Outcome: Skipped, Err: fmt.Errorf("development build %s is never replaced", DevVersion)}
		}
		return Result{Outcome: Skipped, Err: fmt.Errorf("running from %s, not from %s", o.Executable, canonical)}
	}
	// At the canonical place a development build was put there by hand and
	// is replaced like an old release; its version says nothing about what
	// is current, so only the file's own answer counts.
	return installLocked(ctx, o, canonical, !dev)
}

// installLocked is the part of a pass that run and Install share: take the
// lock, ask gh for the newest release, and put it at canonical unless what is
// there is at least as new. With byRunning the running version o.Version
// counts as well: a pass run from the canonical binary is done when that one
// is current. Install runs from anywhere, so only the file at canonical
// counts there.
func installLocked(ctx context.Context, o Options, canonical string, byRunning bool) Result {
	handle, held, err := lock.TryAcquire(filepath.Join(o.StateDir, "update.lock"))
	if err != nil {
		return Result{Outcome: Failed, Err: err}
	}
	if !held {
		return Result{Outcome: Busy, Err: errors.New("update in progress")}
	}
	defer handle.Release()

	rel, err := latest(ctx, o.Run, o.Channel)
	if err != nil {
		return Result{Outcome: Failed, Err: err}
	}
	if byRunning && !Newer(rel, Reported(o.Version, o.Channel)) {
		return Result{Outcome: Current, Version: o.Version}
	}
	// A serve that installed the release keeps running the version before it
	// until a bridge replaces it; asked only the running version, every pass
	// until then would install the same release again.
	if have, ok := InstalledVersion(ctx, o.Run, canonical); ok && !Newer(rel, have) {
		return Result{Outcome: Current, Version: strings.TrimSuffix(have, " (beta)")}
	}
	ver := strings.TrimPrefix(rel.Tag, "v")
	dir := filepath.Dir(canonical)
	if err := fetch(ctx, o, rel.Tag, dir); err != nil {
		return Result{Outcome: Failed, Version: ver, Err: err}
	}
	if err := swap.Swap(dir); err != nil {
		return Result{Outcome: Failed, Version: ver, Err: err}
	}
	return Result{Outcome: Updated, Version: ver}
}

// InstalledVersion is what the binary at path says it is after "loomux ", in
// the form Reported gives. An answer that is not a release version is no
// answer, so that the pass falls back to the running version rather than
// holding back an update on a guess.
func InstalledVersion(ctx context.Context, run Runner, path string) (string, bool) {
	out, err := call(ctx, run, path, "--version")
	if err != nil {
		return "", false
	}
	rest, found := strings.CutPrefix(firstLine(string(out)), "loomux ")
	return rest, found && IsVersion(rest)
}
