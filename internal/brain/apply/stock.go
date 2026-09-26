package apply

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// The three directory calls of moveStock and the one read of copyStock, as
// variables so that a test can make each of them fail.
var (
	recoverDir = lock.Recover
	stagingDir = lock.StagingDir
	swapDir    = os.Rename
	readStock  = os.ReadFile
)

// beforeSwap runs, when set, between moveStock's decision and its swap: the
// window in which a concurrent `index` run may publish the same area. Set
// by tests alone, the way vcs's beforeUpdateRef is.
var beforeSwap func()

// moveStock moves a read-only area's stock from ultra-brain's state
// directory to loomux's, whole, before the first write into it.
//
// A register is written only to the new place (registerWrite), and
// config.ResolvedAreaDir decides a read-only area by its directory: the
// instant one `_identities.tsv` lies under the new `areas/<scope>`, every
// read of that area comes from there -- the declaration included, which
// privacy.VisibleAreas reads for every registered area before it answers
// anything. A register written alone would therefore make search and status
// fail for every scope. So the directory is copied first, into staging
// beside the target, and swapped in once, the way `index` publishes a stock
// (internal/brain/index/staging.go).
//
// Idempotent: a writable area has no stock to move, and once the new place
// holds the area -- or neither place does -- there is nothing to copy. A
// swap a killed run left half-done is finished first, as `index` does, so
// that an aside is not mistaken for an absent stock.
//
// It never replaces an area the new place holds. The caller holds the
// area's lock, which `index` takes too; but a checkout from before that lock
// can still run beside this one, so between the decision above and the swap
// a reindex may publish the area. That stock is newer than the legacy copy,
// and it is kept. The
// swap is therefore a plain rename, which refuses an existing directory
// (Windows) or a non-empty one (POSIX) -- and not lock.ReplaceDir, which
// would put the fresh stock aside and delete it. A rename that fails while
// the target is there is that case, and the staged copy is dropped.
//
// Outside the barrier by design: it writes below the state directory,
// never into the vault or a wiki, which is what `place.gate` measures.
func moveStock(area config.Area, lookup config.ArtifactLookup) error {
	if !area.ReadOnly {
		return nil
	}
	target := config.ManifestDir(area, lookup.Primary)
	if err := recoverDir(target); err != nil {
		return err
	}
	source := config.ResolvedAreaDir(area, lookup.Primary, lookup.Fallback)
	if samePath(source, target) {
		return nil
	}
	staging, err := stagingDir(target)
	if err != nil {
		return err
	}
	// The staging directory is this call's alone; renamed into place on
	// success, it is gone already, and on failure it must not stay. Its
	// error is dropped on purpose: the error worth reporting is the copy's or
	// the swap's, and a leftover `*.staging` is beside the area, never in it.
	defer os.RemoveAll(staging)
	if err := copyStock(source, staging); err != nil {
		return err
	}
	if beforeSwap != nil {
		beforeSwap()
	}
	if err := swapDir(staging, target); err != nil {
		if _, statErr := os.Lstat(target); statErr == nil {
			return nil
		}
		return err
	}
	return nil
}

// copyStock copies source into target, directories and regular files
// alike, as `index`'s copyTree seeds a staging directory. A second copy and
// not a call, because that one is unexported and an import of `index` would
// hide these writes from the barrier test. The prefix is cut off rather
// than computed with Rel, which is sound for a cleaned source.
func copyStock(source, target string) error {
	source = filepath.Clean(source)
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		destination := filepath.Join(target, strings.TrimPrefix(path, source))
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		data, err := readStock(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0o644)
	})
}
