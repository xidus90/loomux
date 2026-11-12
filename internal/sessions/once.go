package sessions

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// MarkOnce reports whether this is the first time name was marked for one
// agent of a session, and marks it. The mark is a file in the session's own
// folder, so Forget takes it along; it is made with O_EXCL, so two hooks
// racing for it cannot both be first. A call without a session marks nothing
// and is never first: there is no place to remember it.
func MarkOnce(root, sessionID, agentID, name string) (first bool, err error) {
	if sessionID == "" {
		return false, nil
	}
	dir := filepath.Join(root, filepath.FromSlash(StateDir), safeName(sessionID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, fmt.Errorf("marking %s: %w", name, err)
	}
	file := name
	if agentID != "" {
		file += "." + safeName(agentID)
	}
	f, err := os.OpenFile(filepath.Join(dir, file), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("marking %s: %w", name, err)
	}
	return true, f.Close()
}
