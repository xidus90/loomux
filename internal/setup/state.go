package setup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/setup/write"
)

// answersPath is where a run keeps what the configuration has no key for.
const answersPath = ".loomux/state/answers.toml"

// Answers are the choices of an earlier run that no schema key holds: the
// hosts and, per part, whether it was chosen. Everything else is read from
// .loomux/config.toml, so a second run never weighs two sources.
type Answers struct {
	Hosts []string        `toml:"hosts"`
	Parts map[string]bool `toml:"parts"`
}

// ReadAnswers reads the answers of root; a project that has none yet gets
// the zero value.
func ReadAnswers(root string) (Answers, error) {
	var a Answers
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(answersPath)))
	if errors.Is(err, fs.ErrNotExist) {
		return Answers{}, nil
	}
	if err != nil {
		return Answers{}, fmt.Errorf("%s: %w", answersPath, err)
	}
	if err := toml.Unmarshal(data, &a); err != nil {
		return Answers{}, fmt.Errorf("%s: not valid TOML: %w", answersPath, err)
	}
	return a, nil
}

// installedPath records the last run that changed something; it is written
// last, so a run that stopped halfway leaves none and the next run's plan
// shows what is still open.
const installedPath = ".loomux/state/installed.toml"

// installed is what installedPath holds.
type installed struct {
	Version string    `toml:"version"`
	At      time.Time `toml:"at"`
	Files   []string  `toml:"files"`
	Actions []string  `toml:"actions"`
}

// writeState replaces the state file rel under root with v in TOML. A state
// file belongs to loomux alone, so unlike a project file it is overwritten
// and never backed up.
func writeState(root, rel string, v any) error {
	// Answers and installed hold only strings, booleans and a time, which
	// always encode.
	data, _ := toml.Marshal(v)
	full := filepath.Join(root, filepath.FromSlash(rel))
	err := write.CheckParents(root, rel)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(full), 0o755)
	}
	if err == nil {
		err = lock.ReplaceText(full, string(data))
	}
	if err != nil {
		return fmt.Errorf("%s: %w", rel, err)
	}
	return nil
}
