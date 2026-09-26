package benchreport

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Targets names the two files of a run and refuses before the first
// measurement if either exists: a run that measures for minutes and then
// cannot save has wasted them.
func Targets(dir, base string) (string, string, error) {
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return "", "", fmt.Errorf("no directory at %s; name one with --out", dir)
	}
	md, js := filepath.Join(dir, base+".md"), filepath.Join(dir, base+".json")
	for _, target := range []string{md, js} {
		if _, err := os.Stat(target); err == nil {
			return "", "", fmt.Errorf("%s already exists", target)
		}
	}
	return md, js, nil
}

// WriteBoth writes both files or neither: a markdown without its JSON is a
// half report, and it would take the name a rerun in the same minute needs.
// Neither file is ever overwritten: Targets checked at the start, and a run
// of the same minute may have taken the names since.
func WriteBoth(md, js string, text, payload []byte) error {
	if err := writeNew(md, text); err != nil {
		return err
	}
	if err := writeNew(js, payload); err != nil {
		_ = os.Remove(md)
		return err
	}
	return nil
}

// writeNew writes data to a file that must not exist yet.
func writeNew(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	return errors.Join(werr, f.Close())
}
