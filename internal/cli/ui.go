package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/serve"
	"github.com/xidus90/loomux/internal/web"
)

// The seams of this file: the interface that binds a port and blocks, the
// detached start, the wait for it, the end of a running one, and the user's
// browser. Each stands for something that must not happen in a test.
var (
	uiRun        = web.Run
	uiStop       = web.Stop
	uiWait       = web.WaitReady
	uiDetach     = serve.Detach
	openBrowser  = startBrowser
	uiReadyAfter = 10 * time.Second
)

func uiUsage() string {
	return "usage: loomux ui [--no-browser]\n       loomux ui stop"
}

// uiCommand is `loomux ui`: start the local web interface, or end it.
//
// The foreground form is what the detached start runs, and is not for a
// terminal: it reads the login code from one line of stdin.
func uiCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	switch {
	case len(args) == 1 && (args[0] == "--help" || args[0] == "-h"):
		fmt.Fprintln(stdout, uiUsage())
		return 0
	case len(args) == 1 && args[0] == "stop":
		return uiStopCommand(stdout, stderr)
	case len(args) == 1 && args[0] == "--foreground":
		return uiForeground(stdin, stderr)
	}
	noBrowser, extra := onlyFlag(args, "--no-browser")
	if extra != "" {
		brainRefuse(stderr, uiUsage(), "loomux ui", "unrecognized arguments: "+extra)
		return 2
	}
	return uiStart(noBrowser, stdout, stderr)
}

// uiStopCommand ends the running interface. Nothing running is a state, not a
// failure, as with `loomux serve stop`.
func uiStopCommand(stdout, stderr io.Writer) int {
	stopped, err := uiStop(config.StateDir())
	if err != nil {
		fmt.Fprintf(stderr, "loomux ui stop: %v\n", err)
		return 1
	}
	if !stopped {
		fmt.Fprintln(stdout, "loomux ui: not running")
		return 0
	}
	fmt.Fprintln(stdout, "loomux ui: stopped")
	return 0
}

// uiForeground serves the interface in this process, with the login code from
// the first line of stdin.
func uiForeground(stdin io.Reader, stderr io.Writer) int {
	// A read that fails or ends early leaves the line empty, and no code is
	// the same refusal either way.
	line, _ := bufio.NewReader(stdin).ReadString('\n')
	code := strings.TrimSpace(line)
	if code == "" {
		fmt.Fprintln(stderr, "loomux ui: no login code on stdin; the interface is started by `loomux ui`")
		return 1
	}
	// Ctrl+C ends the interface through Run's own shutdown, which removes
	// ui.json.
	ctx, stop := serveNotify(context.Background(), os.Interrupt)
	defer stop()
	if err := uiRun(ctx, web.Options{
		StateDir:     config.StateDir(),
		Version:      bareVersion(),
		Code:         code,
		SetupDeps:    setupDeps,
		Gather:       gatherFacts,
		SearchStatus: brainStatus,
	}); err != nil {
		fmt.Fprintf(stderr, "loomux ui: %v\n", err)
		return 1
	}
	return 0
}

// uiStart ends a running interface, starts a new one beside this process and
// opens the browser on its login address.
//
// The login code is drawn here and goes to the child on its stdin: the child's
// command line and every file are free of it. The address handed to the
// browser carries it, and so does the opener's command line while it runs,
// which any process of the same user can read; the code is single-use and
// short-lived for that reason.
func uiStart(noBrowser bool, stdout, stderr io.Writer) int {
	stateDir := config.StateDir()
	// A second `loomux ui` replaces the first, and with it every session.
	if _, err := uiStop(stateDir); err != nil {
		fmt.Fprintf(stderr, "loomux ui: %v\n", err)
		return 1
	}
	code := serve.NewToken()
	// The code goes to the child this start spawned and to no other process:
	// the wait below accepts the state of that PID only.
	var child int
	spawn := func(cmd *exec.Cmd) error {
		err := startProcess(cmd)
		if err == nil {
			child = cmd.Process.Pid
		}
		return err
	}
	if _, err := uiDetach(serve.DetachSpec{
		StateDir: stateDir,
		Args:     []string{"ui", "--foreground"},
		LogPath:  web.LogPath(stateDir),
		Stdin:    strings.NewReader(code + "\n"),
		// The interface runs git and other console programs on every page.
		Console: true,
	}, spawn); err != nil {
		fmt.Fprintf(stderr, "loomux ui: %v\n", err)
		return 1
	}
	state, err := uiWait(stateDir, child, uiReadyAfter)
	if err != nil {
		fmt.Fprintf(stderr, "loomux ui: %v\n", err)
		return 1
	}
	address := fmt.Sprintf("http://127.0.0.1:%d/", state.Port)
	login := address + "login?code=" + code
	if noBrowser {
		fmt.Fprintf(stdout, "loomux ui: open %s within two minutes; it works once\n", login)
		return 0
	}
	if err := openBrowser(login); err != nil {
		// The login address is the one way in, so it is said here.
		fmt.Fprintf(stderr, "loomux ui: could not open the browser: %v\n", err)
		fmt.Fprintf(stdout, "loomux ui: open %s within two minutes; it works once\n", login)
		return 0
	}
	fmt.Fprintf(stdout, "loomux ui: running at %s\n", address)
	return 0
}

// startBrowser opens a URL in the user's default browser. The address carries
// no character a shell or the URL handler would read twice: it is a loopback
// address with a base64url code.
//
//coverage:exempt it starts the user's browser
func startBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// brainStatus is what `loomux brain status` prints on the local channel for
// the vault of stateDir: the dashboard shows the same lines.
func brainStatus(stateDir string) (string, error) {
	out, _, err := answer.RunWith(brainPorts(), answer.Request{Command: "status", Channel: privacy.ChannelLocal}, stateDir, nil)
	return out, err
}
