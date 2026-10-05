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

// The modes of a pass by hand. A pass in the machine's channel is the
// default and serve's only mode.
const (
	ModeChannel = ""
	ModeBeta    = "beta"
	ModeStable  = "stable"
)

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
	// Mode or Pin, set by upgrade; Pin is a version without the v.
	Mode string
	Pin  string
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
// lock, choose the release, and put it at canonical unless what is there is
// at least as new. With byRunning the running version o.Version counts as
// well: a pass run from the canonical binary is done when that one is
// current. Install runs from anywhere, so only the file at canonical counts
// there. A pass that came to current or updated also leaves the channel
// marker as it asked; a problem with the marker travels in Err beside that
// outcome.
func installLocked(ctx context.Context, o Options, canonical string, byRunning bool) Result {
	handle, held, err := lock.TryAcquire(filepath.Join(o.StateDir, "update.lock"))
	if err != nil {
		return Result{Outcome: Failed, Err: err}
	}
	if !held {
		return Result{Outcome: Busy, Err: errors.New("update in progress")}
	}
	defer handle.Release()

	running := Reported(o.Version, o.Channel)
	rel, forced, markerErr, err := choose(ctx, o, running)
	if err != nil {
		return Result{Outcome: Failed, Err: err}
	}
	res := place(ctx, o, canonical, byRunning, running, rel, forced)
	if res.Outcome == Current || res.Outcome == Updated {
		markerErr = errors.Join(markerErr, settle(o, rel))
	}
	if markerErr != nil {
		res.Err = errors.Join(res.Err, markerErr)
	}
	return res
}

// choose is the release a pass aims at. A pin or --stable aims at one
// release even when it is older than what runs (forced); --beta and the
// machine's channel aim at the newest one they take. A marker that cannot
// be read leaves the machine on stable, and its error travels on.
func choose(ctx context.Context, o Options, running string) (rel Release, forced bool, markerErr, err error) {
	switch {
	case o.Pin != "":
		rel, err = view(ctx, o.Run, o.Pin)
		return rel, true, nil, err
	case o.Mode == ModeStable:
		rel, err = latest(ctx, o.Run, takesStable)
		return rel, true, nil, err
	case o.Mode == ModeBeta:
		rel, err = latest(ctx, o.Run, takesAll)
		return rel, false, nil, err
	}
	beta, markerErr := ReadChannel(o.StateDir)
	t := takesStable
	switch v, ok := parseReported(running); {
	case beta:
		t = takesAll
	case ok && v.old:
		// Frozen logic of the last old release: it follows its own count and
		// the first stable one, but no new beta it never asked for.
		t = takesOld
	}
	rel, err = latest(ctx, o.Run, t)
	return rel, false, markerErr, err
}

// settle leaves the marker as the pass asked: --beta sets it, --stable
// clears it, a pin follows the kind of release it pinned, and a pass in the
// channel leaves it alone.
func settle(o Options, rel Release) error {
	switch {
	case o.Pin != "":
		v, _ := releaseVersion(rel)
		return WriteChannel(o.StateDir, v.beta > 0)
	case o.Mode == ModeBeta:
		return WriteChannel(o.StateDir, true)
	case o.Mode == ModeStable:
		return WriteChannel(o.StateDir, false)
	}
	return nil
}

// place puts rel at canonical unless what runs or what is there already
// is it (forced) or is at least as new (otherwise).
func place(ctx context.Context, o Options, canonical string, byRunning bool, running string, rel Release, forced bool) Result {
	ver := strings.TrimPrefix(rel.Tag, "v")
	// Forced compares number and beta only, not the count: a 7.1.0 does not
	// exist after the restart, and before it the same number means the same
	// release.
	done := func(have string) bool {
		if forced {
			h, ok := parseReported(have)
			r, _ := releaseVersion(rel)
			return ok && h.num == r.num && h.beta == r.beta
		}
		return !Newer(rel, have)
	}
	if byRunning && done(running) {
		return Result{Outcome: Current, Version: o.Version}
	}
	// A serve that installed the release keeps running the version before it
	// until a bridge replaces it; asked only the running version, every pass
	// until then would install the same release again.
	if have, ok := InstalledVersion(ctx, o.Run, canonical); ok && done(have) {
		return Result{Outcome: Current, Version: strings.TrimSuffix(have, " (beta)")}
	}
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
