package verify

import (
	_ "embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/config"
)

// NewRunID names one run by its start and process, so two runs started in
// the same second still keep their coverage files apart. The start is spelt
// in UTC: formatting local time loads the zone on first use, about 18 ms on
// Windows, which every check and post-edit would pay.
func NewRunID(now time.Time, pid int) string {
	return now.UTC().Format("20060102T150405") + "-" + strconv.Itoa(pid)
}

func coverDir(root string) string { return filepath.Join(root, ".loomux", "state", "cover") }

// PrepareRun lays out what the lanes of jobs read but the agent may not
// write: the directory every measuring lane writes into, refused under
// .loomux/state by the policy, and stylelint's default configuration in
// loomux's state directory, outside the work tree, where a lane names it.
// Each caller of Run goes through it first.
func PrepareRun(root string, jobs []Job) error {
	if err := os.MkdirAll(coverDir(root), 0o755); err != nil {
		return err
	}
	path := StylelintConfig()
	named := slices.ContainsFunc(jobs, func(j Job) bool {
		return slices.ContainsFunc(j.Argvs, func(argv []string) bool { return slices.Contains(argv, path) })
	})
	if old, err := os.ReadFile(path); !named || err == nil && string(old) == stylelintConfig {
		return nil
	}
	// Written beside its place and renamed into it: a lane of a run beside
	// this one reads the old file or the new one, never half of one. A
	// directory that cannot be made fails the temporary file.
	os.MkdirAll(filepath.Dir(path), 0o755)
	tmp, err := os.CreateTemp(filepath.Dir(path), "stylelint-*.tmp")
	if err != nil {
		return err
	}
	_, err = tmp.WriteString(stylelintConfig)
	if err = errors.Join(err, tmp.Close()); err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return err
}

// stylelintConfig is the configuration stylelint reads where the project has
// none: the shared SCSS standard npx installs beside the pinned stylelint. The
// source keeps a .txt suffix so the repository holds no JavaScript source of
// its own for the code graph and the rules.
//
//go:embed stylelint.config.mjs.txt
var stylelintConfig string

// StylelintConfig is where PrepareRun writes the default configuration.
func StylelintConfig() string {
	return filepath.Join(config.StateDir(), "verify", "stylelint.config.mjs")
}

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
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
