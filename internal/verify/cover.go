package verify

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// NewRunID names one run by its start and process, so two runs started in
// the same second still keep their coverage files apart. The start is spelt
// in UTC: formatting local time loads the zone on first use, about 18 ms on
// Windows, which every check and post-edit would pay.
func NewRunID(now time.Time, pid int) string {
	return now.UTC().Format("20060102T150405") + "-" + strconv.Itoa(pid)
}

func coverDir(root string) string { return filepath.Join(root, ".loomux", "state", "cover") }

// PrepareCover makes the directory every measuring lane writes into. Each
// caller of Run goes through it first: the agent cannot make it, since the
// policy refuses writes under .loomux/state.
func PrepareCover(root string) error { return os.MkdirAll(coverDir(root), 0o755) }

// CoverPaths are the profile and data file one lane of a run measures into.
// The root area is spelt out and a nested area flattened, since every file
// has to stand directly in the cover directory for CleanCover to find it.
func CoverPaths(root, runID, stack, area string) (profile, data string) {
	if area == "." {
		area = "root"
	}
	base := filepath.Join(coverDir(root), runID+"-"+stack+"-"+strings.ReplaceAll(area, "/", "_"))
	return base + ".out", base + ".data"
}

// staleAfter is how long another run's file is left alone: a run that is
// still going, such as a check beside the pre-commit gate, reads its own.
const staleAfter = 24 * time.Hour

// CleanCover removes what earlier runs left behind a day ago or longer, and
// this run's own files only when it was green: a red run keeps them for
// whoever looks into it.
func CleanCover(root, runID string, green bool) error {
	return cleanCover(root, runID, green, time.Now())
}

func cleanCover(root, runID string, green bool, now time.Time) error {
	dir := coverDir(root)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var errs []error
	for _, e := range entries {
		own := strings.HasPrefix(e.Name(), runID+"-")
		if own && !green {
			continue
		}
		if !own {
			info, err := e.Info()
			if err != nil || now.Sub(info.ModTime()) < staleAfter {
				continue
			}
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
