package query

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
)

// grepChannel reads the --channel value. Empty means local, the behaviour
// every caller had before the channel existed.
func grepChannel(name string) (privacy.Channel, error) {
	if name == "" {
		return privacy.ChannelLocal, nil
	}
	return privacy.ParseChannel(name)
}

// cloudKeep is the predicate the CLI path reads a file through on the cloud
// channel: the project's own declaration, asked exactly as the MCP path asks
// it. An absent declaration hides nothing; a declaration that exists and
// cannot be read fails the call, because an unreadable manifest must never
// turn into an unfiltered answer.
func cloudKeep(root string) (func(string) bool, error) {
	manifest, visible, err := privacy.VisibleManifest(root, privacy.ChannelCloud)
	if errors.Is(err, config.ErrNoManifest) {
		return func(string) bool { return true }, nil
	}
	if err != nil {
		return nil, fmt.Errorf("privacy declaration: %w", err)
	}
	if !visible {
		return nil, fmt.Errorf("%s is local_only and cannot be read on the cloud channel", root)
	}
	area := privacy.VisibleArea{Area: config.Area{Path: root}, Manifest: manifest}
	return func(path string) bool {
		return privacy.IsReadable(manifest, path) && !area.Conceals(path)
	}, nil
}

// globFilter turns --glob into a path predicate; empty means every path.
func globFilter(pattern string) func(string) bool {
	if pattern == "" {
		return nil
	}
	// ripgrep reads a glob without a slash against the file name at every
	// depth; the privacy translator's full match would anchor it at the root.
	if !strings.Contains(pattern, "/") {
		pattern = "**/" + pattern
	}
	return func(path string) bool { return privacy.MatchesGlobs([]string{pattern}, path) }
}
