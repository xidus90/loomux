package lock

import (
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

// replaceEventually swaps the file in and gives Windows time to drop an open
// read handle. The last failed attempt returns right away: a pause before the
// error would buy nothing.
func replaceEventually(source, target string) error {
	for attempt := 1; ; attempt++ {
		err := os.Rename(source, target)
		if err == nil {
			return nil
		}
		if attempt >= replaceTries {
			return err
		}
		time.Sleep(replacePause)
	}
}
