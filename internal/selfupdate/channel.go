package selfupdate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ChannelPath is the marker that puts a machine on the beta channel: its
// presence, holding "beta", lets every pass take betas as well as stable
// releases, until upgrade --stable or --version <stable> removes it.
func ChannelPath(stateDir string) string { return filepath.Join(stateDir, "channel") }

// ReadChannel reports whether the machine is on the beta channel. A missing
// marker is the stable channel; one that cannot be read or holds anything
// else counts as stable too, and the error says why.
func ReadChannel(stateDir string) (bool, error) {
	data, err := os.ReadFile(ChannelPath(stateDir))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read channel file: %w", err)
	}
	if got := strings.TrimSpace(string(data)); got != "beta" {
		return false, fmt.Errorf("channel file holds %q, not beta", got)
	}
	return true, nil
}

// WriteChannel sets the marker for beta and removes it otherwise.
func WriteChannel(stateDir string, beta bool) error {
	path := ChannelPath(stateDir)
	if beta {
		return os.WriteFile(path, []byte("beta\n"), 0o644)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
