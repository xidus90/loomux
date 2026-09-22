package gitwork

import (
	"fmt"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

// RemoteTimeout is how long a remote may take to answer. Ten seconds is long
// for ls-remote against a reachable host and short enough that a dead one
// costs nothing anybody notices -- subagent_stop.py's number.
const RemoteTimeout = 10 * time.Second

// remoteStart is the seam a test uses to stand in for a remote that hangs.
var remoteStart = child.Run

// LsRemote maps every ref the remote names to its commit, HEAD included.
//
// Through child, so a hung connection costs its deadline and no more. No
// credential prompt: a hook has no terminal to ask on, and a prompt would
// hold the call until the deadline. SSH is left as the user set it up -- an
// ssh without a terminal refuses rather than asks.
func LsRemote(root, remote string) (map[string]string, error) {
	res := remoteStart(child.Spec{
		Argv:    []string{"git", "ls-remote", remote},
		Dir:     root,
		Env:     []string{"GIT_TERMINAL_PROMPT=0"},
		Timeout: RemoteTimeout,
	})
	switch {
	case res.Err != nil:
		return nil, fmt.Errorf("git ls-remote %s: %w", remote, res.Err)
	case res.TimedOut:
		return nil, fmt.Errorf("git ls-remote %s: no answer within %s", remote, RemoteTimeout)
	case res.Code != 0:
		return nil, fmt.Errorf("git ls-remote %s: exit %d: %s", remote, res.Code, strings.TrimSpace(res.Stderr))
	}
	return parseRefs(res.Stdout), nil
}

// HasRemote says whether a remote of that name is configured. A repository
// without one has nothing to ask ls-remote about, which is not the same as a
// remote that did not answer.
func HasRemote(root, remote string) bool {
	_, err := git(root, "remote", "get-url", remote)
	return err == nil
}
