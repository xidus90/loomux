package lock

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// replaceTries and replacePause are replace_text's defaults
// (src/brain/locking.py:219), which _replace_eventually takes as it finds
// them. Windows refuses the swap while another process holds the target open,
// and a reader hits that window within milliseconds.
const (
	replaceTries = 5
	replacePause = 20 * time.Millisecond
)

// ReplaceText writes text alongside and then swaps it in.
//
// Without translating the line endings: the registry is compared byte for
// byte with the Python reader, and a switch to CRLF would be a silent
// difference (locking.replace_text, `newline=""`).
func ReplaceText(path, text string) error {
	name, err := writeTemporary(path, text)
	if err != nil {
		return err
	}
	if err := replaceEventually(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// writeTemporary puts text into a file of its own next to path and returns
// that file's name.
//
//coverage:exempt the WriteString and Close error arms need a write to a freshly created temporary file to fail (a full disk, a directory that vanished between create and write); the CreateTemp arm is covered
func writeTemporary(path, text string) (string, error) {
	// A name of its own per writer, not the PID: two writers of one process
	// would otherwise meet on the same temporary file.
	temporary, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return "", err
	}
	name := temporary.Name()
	if _, err := temporary.WriteString(text); err != nil {
		temporary.Close()
		os.Remove(name)
		return "", err
	}
	if err := temporary.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

// SidecarBudget is how long a writer of a derived file waits for a reader to
// let go of the target. A variable so that tests of other packages can shorten
// it; nothing else assigns it.
var SidecarBudget = 2 * time.Second

// OnRetry, when set, is called after every rename that failed. It is nil
// outside tests, which use it to learn that a writer is waiting.
var OnRetry func()

// Seams of the retry loop: the rename, the removal, the clock and the pause.
var (
	replaceRename = os.Rename
	replaceRemove = os.Remove
	replaceNow    = time.Now
	replaceSleep  = time.Sleep
)

// replaceEventually swaps the file in with the budget replace_text has always
// had.
func replaceEventually(source, target string) error {
	return ReplaceEventually(source, target, replaceTries*replacePause)
}

// ReplaceEventually swaps the file in and gives Windows time to drop an open
// read handle, for as long as budget lasts. The last failed attempt returns
// right away: a pause that would use up the budget buys nothing.
func ReplaceEventually(source, target string, budget time.Duration) error {
	return eventually(func() error { return replaceRename(source, target) }, budget)
}

// RemoveEventually deletes the file with the same patience as
// ReplaceEventually: a reader's open handle refuses the removal on Windows
// just as it refuses the swap. A file that is already gone counts as removed.
func RemoveEventually(path string, budget time.Duration) error {
	return eventually(func() error {
		if err := replaceRemove(path); !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}, budget)
}

// eventually repeats op until it succeeds or the budget is used up.
func eventually(op func() error, budget time.Duration) error {
	started := replaceNow()
	for {
		err := op()
		if err == nil {
			return nil
		}
		if OnRetry != nil {
			OnRetry()
		}
		if replaceNow().Sub(started)+replacePause >= budget {
			return err
		}
		replaceSleep(replacePause)
	}
}
