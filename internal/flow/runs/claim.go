package runs

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// claimAttempts bounds how often a number another run took in the meantime is
// computed again. Ten runs starting within one marker write of each other is
// not a project this has to serve.
const claimAttempts = 10

// Claim gives a new run its number by writing its marker, and returns the
// number.
//
// The marker is the claim: it is created exclusively, so of two runs that
// computed the same number only one gets it, and the other computes again. The
// journal comes after, and so never belongs to two runs.
func Claim(root string, marker Marker) (string, error) {
	return claim(root, marker, NextID)
}

func claim(root string, marker Marker, next func(string) string) (string, error) {
	for attempt := 0; attempt < claimAttempts; attempt++ {
		id := next(root)
		err := WriteMarker(MarkerPath(root, id), marker)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		return id, nil
	}
	return "", fmt.Errorf("no free run number under %s after %d attempts", filepath.Join(root, Dir), claimAttempts)
}
