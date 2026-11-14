// Package target is the one place a configuration file is read, changed and
// written: the project's .loomux/config.toml or the machine-wide config.toml.
// The command line and the web interface both go through it, so a change
// meets the same refusals wherever it is made.
package target

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/brain/model"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/edit"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/lock"
)

// readFile is the seam for every read of the file and of a proposal: a test
// of a caller acts between two reads with it, which a real disk cannot be made
// to do at a chosen moment.
var readFile = os.ReadFile

// SetReadFileForTest replaces the read of the file and of a proposal until the
// returned restore runs; it lets a test of another package reach the seam, and
// nothing but a test calls it.
func SetReadFileForTest(f func(string) ([]byte, error)) (restore func()) {
	readFile = f
	return func() { readFile = os.ReadFile }
}

// replaceText is the seam for the atomic replacement of the file: a write
// that fails after the file was read cannot be provoked on a real disk.
var replaceText = lock.ReplaceText

// writing serialises the read-compare-replace of Write inside one process, so
// two requests of the web interface cannot both pass the stale check.
// ponytail: another process (the command line) can still slip in between the
// compare and the replace; a lock file per target if that ever bites.
var writing sync.Mutex

// Target is the one file a run reads and writes, and the keys it knows.
type Target struct {
	Path   string
	Keys   []schema.Key
	Global bool
}

// Project is the configuration file of the project rooted at root.
func Project(root string) Target {
	return Target{Path: config.ManifestPath(root), Keys: schema.Keys()}
}

// Global is the machine-wide configuration file in stateDir.
func Global(stateDir string) Target {
	return Target{Path: config.GlobalConfigPath(stateDir), Keys: schema.GlobalKeys(), Global: true}
}

// Read answers an empty configuration for a file that is not there yet:
// `set` creates it, and every key then shows its default.
func (t Target) Read() (string, error) {
	data, err := readFile(t.Path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return string(data), err
}

// Entries pairs this target's keys with their values in text: the global
// file shares the project file's reader but not its keys. A parse error
// names the global file by its path; the project file keeps its short name.
func (t Target) Entries(text string) ([]schema.Entry, error) {
	file := schema.ProjectFile
	if t.Global {
		file = t.Path
	}
	return schema.CurrentOf(file, t.Keys, text)
}

// Unknown lists what text holds that none of this target's keys describes.
func (t Target) Unknown(text string) ([]string, error) {
	return schema.Unknown(t.Keys, text)
}

// Lookup finds the key an id names among this target's keys.
func (t Target) Lookup(id string) (schema.Key, bool) {
	return schema.Match(t.Keys, id)
}

// editableKey finds a key that a line edit can change: known to this target
// and not a table.
func (t Target) editableKey(id string) (schema.Key, error) {
	key, ok := t.Lookup(id)
	if !ok {
		return schema.Key{}, fmt.Errorf("unknown key %q", id)
	}
	if key.Kind == schema.Table || key.Kind == schema.TableList {
		return schema.Key{}, fmt.Errorf("%s is a table; edit it by hand", id)
	}
	return key, nil
}

// Change computes the new text for one key; callers show and confirm it.
func (t Target) Change(text, id, input string) (string, error) {
	key, err := t.editableKey(id)
	if err != nil {
		return "", err
	}
	// Without [area] the declaration reader stops before the brain's keys,
	// so Validate below would pass any value there unseen.
	if key.Module == schema.Brain && !t.Global && id != "area.scope" {
		hasArea, err := declaresArea(text)
		if err != nil {
			return "", err
		}
		if !hasArea {
			return "", fmt.Errorf("%s belongs to the brain, which reads nothing without [area]; set area.scope first", id)
		}
	}
	literal, err := edit.Render(key.Kind, input)
	if err != nil {
		return "", err
	}
	// Defaults are never written: a file that repeats them would pin them
	// against a later change of the default.
	var next string
	if literal == key.Default {
		next, err = edit.Remove(text, key.Section, key.Name)
	} else {
		next, err = edit.Set(text, key.Section, key.Name, literal)
	}
	if err != nil {
		return "", err
	}
	return t.Validated(next)
}

// Unset computes the text without the key's line. It needs no [area] for a
// brain key: a line the brain cannot read is the one worth removing.
func (t Target) Unset(text, id string) (string, error) {
	key, err := t.editableKey(id)
	if err != nil {
		return "", err
	}
	next, err := edit.Remove(text, key.Section, key.Name)
	if err != nil {
		return "", err
	}
	return t.Validated(next)
}

// Validated hands a new text to the reader that runs in operation: the
// declaration readers for a project file, the model reader, the client's
// loopback guard and the search reader for the global one. The guard runs
// here too so that the file never holds an address the next pass would
// refuse.
func (t Target) Validated(next string) (string, error) {
	if t.Global {
		settings, err := config.ParseModelSettings(t.Path, next)
		if err != nil {
			return "", err
		}
		if err := model.GuardEndpoint(settings.Endpoint); err != nil {
			return "", fmt.Errorf("%s: %w", t.Path, err)
		}
		if _, err := config.ParseSearchSettings(t.Path, next); err != nil {
			return "", err
		}
		if _, err := config.ParseGlobalPolicy(t.Path, []byte(next)); err != nil {
			return "", err
		}
		return next, nil
	}
	if err := schema.Validate(next); err != nil {
		return "", err
	}
	return next, nil
}

func declaresArea(text string) (bool, error) {
	doc := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &doc); err != nil {
		return false, fmt.Errorf(".loomux/config.toml: not valid TOML: %w", err)
	}
	_, ok := doc["area"].(map[string]any)
	return ok, nil
}

// Write is the one way every form puts a new text in place: set, unset, the
// interactive form, the web form and apply. before is the text next was made
// from; a file that no longer holds it was changed by someone else while the
// human read the diff, and next would take that change back unseen.
func (t Target) Write(before, next string) error {
	writing.Lock()
	defer writing.Unlock()
	current, err := t.Read()
	if err != nil {
		return err
	}
	if current != before {
		return fmt.Errorf("%s changed since it was read; nothing written", t.Path)
	}
	err = os.MkdirAll(filepath.Dir(t.Path), 0o755)
	if err == nil {
		err = replaceText(t.Path, next)
	}
	return err
}
