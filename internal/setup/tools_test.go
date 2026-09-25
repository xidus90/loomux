package setup

import (
	"errors"
	"testing"
)

func TestMissingToolsAreNamedWithTheirInstallCommand(t *testing.T) {
	root := world(t, map[string]string{})
	lookPath = func(name string) (string, error) {
		if name == "qmd" {
			return "", errors.New("missing")
		}
		return name, nil
	}
	p := plan(t, gather(t, root, ""))
	if !hasNote(p, "qmd is not on PATH; install it with: npm install -g @tobilu/qmd") {
		t.Errorf("notes = %v", p.Notes)
	}
}
