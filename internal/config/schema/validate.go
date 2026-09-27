package schema

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/verify"
	"github.com/xidus90/loomux/internal/verify/commit"
	"github.com/xidus90/loomux/internal/worktree/mirror"
)

// Validate asks the readers that run in operation whether they accept text
// as a project's .loomux/config.toml. They read files, so text goes into a
// scratch project first; a second checker beside them would be a second
// yardstick, and the two would drift.
func Validate(text string) error {
	root, path, err := scratch(text)
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	return named(path, readAll(root, path))
}

// scratch makes a project whose configuration is text, and leaves nothing
// behind when it cannot.
//
//coverage:exempt the directory and the file go into a directory this process created a moment ago; only a full disk or a racing deletion fails them
func scratch(text string) (root, path string, err error) {
	root, err = os.MkdirTemp("", "loomux-config-")
	if err != nil {
		return "", "", err
	}
	path = config.ManifestPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		os.RemoveAll(root)
		return "", "", err
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		os.RemoveAll(root)
		return "", "", err
	}
	return root, path, nil
}

// readAll stops at the first reader that refuses. A file without [area] is
// no refusal: a project may use loomux without the brain.
func readAll(root, path string) error {
	if _, err := config.ReadDeclaration(path); err != nil && !errors.Is(err, config.ErrNoArea) {
		return err
	}
	if _, err := config.ReadModules(root); err != nil {
		return err
	}
	if _, err := config.ReadPolicy(root); err != nil {
		return err
	}
	if _, err := verify.ReadConfig(root); err != nil {
		return err
	}
	if _, err := commit.ReadPolicy(root); err != nil {
		return err
	}
	if _, err := config.ReadAgent(root); err != nil {
		return err
	}
	if _, err := config.ReadFlowSettings(root); err != nil {
		return err
	}
	_, err := mirror.Mirror(root)
	return err
}

// named puts the project's own name where the scratch path stood: the human
// reading the refusal fixes .loomux/config.toml, not a temporary directory.
// The chain is dropped with it; callers only print the text.
func named(scratch string, err error) error {
	if err == nil {
		return nil
	}
	return errors.New(strings.ReplaceAll(err.Error(), scratch, filepath.ToSlash(filepath.Join(".loomux", "config.toml"))))
}
