package apply

import (
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// recoverDir is lock.Recover, as a variable so that a test can make it fail.
var recoverDir = lock.Recover

// recoverStock finishes a swap a killed `index` run left half-done in a
// read-only area's stock, before the first write into it.
//
// `index` publishes that stock with lock.ReplaceDir, which puts the old one
// aside for an instant. Killed in that instant, it leaves the aside and no
// target. A register written now would create the target with that one file,
// and the next Recover would find a target and delete the aside -- the
// area's catalog, graph and declaration with it. So the aside goes back
// first, as `index` does before it reads.
//
// A writable area keeps its stock in its own tree, which no swap touches.
// Outside the barrier by design: it renames below the state directory, never
// into the vault or a wiki, which is what `place.gate` measures.
func recoverStock(area config.Area, lookup config.ArtifactLookup) error {
	if !area.ReadOnly {
		return nil
	}
	return recoverDir(config.ManifestDir(area, lookup.Primary))
}
