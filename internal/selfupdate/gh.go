package selfupdate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Repo is where the releases come from. It is private, so every call goes
// through gh and the login gh already holds; loomux never handles a token.
const Repo = "xidus90/loomux"

// Runner runs one program and answers its standard output. A failure carries
// the first line of standard error, which is where gh says what went wrong.
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// callTimeout bounds every external call. A variable so that a test can reach
// the deadline without waiting two minutes for it.
var callTimeout = 2 * time.Minute

// ExecRunner is the real Runner.
func ExecRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := command(ctx, name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}
	if line := firstLine(stderr.String()); line != "" {
		return out, fmt.Errorf("%s: %s: %w", filepath.Base(name), line, err)
	}
	return out, fmt.Errorf("%s: %w", filepath.Base(name), err)
}

// command is the process ExecRunner starts, built apart so that a test can
// read how it would start without starting it.
func command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = hiddenAttrs()
	return cmd
}

// firstLine is the first non-empty line of s, trimmed.
func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(line)
}

// call runs one program under the deadline and names the two failures a user
// acts on differently from the rest: no gh at all, and a call that hung.
func call(ctx context.Context, run Runner, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	out, err := run(ctx, name, args...)
	switch {
	case err == nil:
		return out, nil
	case name == "gh" && errors.Is(err, exec.ErrNotFound):
		return nil, errors.New("gh not found; install GitHub CLI and run gh auth login")
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return nil, fmt.Errorf("%s timed out after %s", filepath.Base(name), callTimeout)
	}
	return nil, err
}

// latest is the newest release the channel takes. Not `gh release
// view` without a tag: that knows only the release marked latest, and a
// repository of pre-releases has none (measured 2026-09-23: "release not
// found").
func latest(ctx context.Context, run Runner, t takes) (Release, error) {
	out, err := call(ctx, run, "gh", "release", "list", "--repo", Repo,
		"--exclude-drafts", "--limit", "30", "--json", "tagName,isPrerelease")
	if err != nil {
		return Release{}, err
	}
	var releases []Release
	if err := json.Unmarshal(out, &releases); err != nil {
		return Release{}, fmt.Errorf("parse gh release list: %w", err)
	}
	rel, ok := pick(releases, t)
	if !ok {
		return Release{}, fmt.Errorf("no release in channel %s", channelName(t))
	}
	return rel, nil
}

// channelName is how an error names the channel a pass searched.
func channelName(t takes) string {
	if t != takesStable {
		return "beta"
	}
	return "stable"
}

// view is the one release a pinned pass asks for.
func view(ctx context.Context, run Runner, ver string) (Release, error) {
	out, err := call(ctx, run, "gh", "release", "view", "v"+ver, "--repo", Repo, "--json", "tagName,isPrerelease")
	if err != nil {
		return Release{}, fmt.Errorf("no release %s: %w", ver, err)
	}
	var r Release
	if err := json.Unmarshal(out, &r); err != nil {
		return Release{}, fmt.Errorf("parse gh release view: %w", err)
	}
	if _, ok := releaseVersion(r); !ok {
		return Release{}, fmt.Errorf("no release %s: %s is no version", ver, r.Tag)
	}
	return r, nil
}
