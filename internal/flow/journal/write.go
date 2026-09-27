package journal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Append writes one entry as one line, creating the file and its parents.
//
// Entry carries no omitempty, so every one of its thirteen keys is written, and
// a nil Model, Role or DefinitionHash as null rather than left out. The line is
// built before the file is touched, so an entry JSON cannot express leaves
// nothing behind. LF on every platform: the journal is a data format whose
// bytes a resume and the golden-journal test in flows/ compare.
func Append(path string, entry Entry) error {
	line, err := Canonical(entry)
	if err != nil {
		return fmt.Errorf("journal entry for node %s: %w", entry.Node, err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	_, writeErr := file.Write(append(line, '\n'))
	return closed(path, errors.Join(writeErr, file.Close()))
}

// closed reports a failed write or close. Apart from Append, because a disk
// that refuses an open file is the one failure no test can provoke.
func closed(path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("writing %s: %w", path, err)
}
