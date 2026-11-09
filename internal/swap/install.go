package swap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// create is os.Create, as a variable so that a test can refuse the copy the
// way a Low-integrity process is refused outside its folder.
var create = os.Create

// DevDir is where the hooks of the checkout at root find its pilot binary:
// one folder per checkout under LOCALAPPDATA, named after the checkout's
// folder. "" when LOCALAPPDATA is not set.
func DevDir(root string) string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return ""
	}
	abs, _ := filepath.Abs(root)
	return filepath.Join(local, "loomux", "dev", filepath.Base(abs))
}

// Install copies src into dir as a new file and swaps it in there. A copy and
// not a rename: a moved file keeps the integrity label of the folder it came
// from, a created one takes that of dir.
func Install(src, dir string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	next := filepath.Join(dir, "loomux.new.exe")
	f, err := create(next)
	if err != nil {
		return fmt.Errorf("copying to %s: %w", dir, err)
	}
	_, werr := f.Write(data)
	if err := errors.Join(werr, f.Close()); err != nil {
		return fmt.Errorf("copying to %s: %w", dir, err)
	}
	return Swap(dir)
}
