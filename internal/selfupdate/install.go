package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Install puts the newest release at the canonical location, whatever runs
// it. Run replaces only the binary it runs from, which a first install has
// not got: init runs from a checkout or go run, and the place it fills is
// empty. The chain is Run's -- gh, SHA256SUMS, the fetched binary's own
// --version, swap -- so nothing reaches the canonical location that an update
// would not have put there. It never copies the running binary: a checkout
// build at that place would be replaced by the next update anyway, and until
// then every host would run someone's work in progress.
//
// update.json stays serve's and upgrade's record; an install says nothing
// about the last update pass.
func Install(ctx context.Context, o Options) Result {
	if o.GOOS != "windows" {
		return Result{Outcome: Skipped, Err: errors.New("the machine-wide binary is installed on Windows only")}
	}
	canonical := Canonical(o.StateDir)
	dir := filepath.Dir(canonical)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{Outcome: Failed, Err: fmt.Errorf("create %s: %w", dir, err)}
	}
	// An init from a beta brings the counterpart of itself onto the machine,
	// not an older stable release: the entries it writes then call no older
	// binary than the one that wrote them.
	if v, _ := parseVersion(o.Version); o.Mode == ModeChannel && o.Pin == "" && v.beta > 0 {
		o.Mode = ModeBeta
	}
	return installLocked(ctx, o, canonical, false)
}
