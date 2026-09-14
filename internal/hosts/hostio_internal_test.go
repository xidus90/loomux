package hosts

import (
	"errors"
	"strings"
	"testing"
)

// The one failure filepath.Abs can report is os.Getwd's, which no test can
// provoke on any of the platforms this runs on -- README.md's install block
// names Windows, Linux and macOS -- so the resolver is a parameter
// and this test hands it the failure directly. Without it the refusal would
// be an untested line claiming to name the directory it could not resolve.
func TestFindRootReportsAnUnresolvableStart(t *testing.T) {
	boom := errors.New("no working directory")

	_, err := findRoot("somewhere", func(string) (string, error) { return "", boom })

	if !errors.Is(err, boom) {
		t.Fatalf("expected the resolver's error, got %v", err)
	}
	if !strings.Contains(err.Error(), "somewhere") {
		t.Fatalf("the refusal names the directory it could not resolve, got %v", err)
	}
}
