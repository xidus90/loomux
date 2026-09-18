package serve

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/mcptools"
	servebrain "github.com/xidus90/loomux/internal/serve/brain"
)

// ErrAlreadyRunning is what a second serve gets: the lock is held, and a
// second service would bind ports nobody knows about.
var ErrAlreadyRunning = errors.New("another loomux serve already holds the lock")

const (
	// MCPPath is where each channel answers MCP. It is part of the URL in
	// serve.json, so a client never has to know it.
	MCPPath = "/mcp"
	// StopPath ends the service. It exists on the local listener only: the
	// cloud channel is what a remote model speaks through, and ending the
	// service is not among the things it may ask for.
	StopPath = "/stop"
)

// shutdownTimeout bounds the orderly stop. Whatever is still in flight after
// it loses; the listener is closed at the first instant of the shutdown either
// way, so nothing new arrives while we wait.
const shutdownTimeout = 5 * time.Second

// Options are what one service run needs. Everything hangs off StateDir, never
// off a fixed path: only so does a test isolate its serve from the real one.
type Options struct {
	StateDir    string
	RegistryDir string
	LegacyDir   string
	Foreground  bool
	// BrokeAway goes straight into serve.json, and only the parent that
	// spawned this service knows whether the breakaway succeeded. The entry
	// point of the serve command has to set it from BrokeAwayFromEnv(); left
	// at its zero value, every state file says the service dies with its host
	// and serve status stops telling the ones that really do from the rest.
	BrokeAway bool
	// Answer falls back to answer.Run when nil. It is the seam the HTTP
	// progress test needs, and the same one stage 1b-1 uses for the launcher
	// and the spawner.
	Answer func(answer.Request, string, string, func(string)) (string, []string, error)
}

// answerFunc is the answer this run gives, the real one unless a caller
// brought its own. The real one asks a qmd port built for this service, not
// the command line's; qmdOptions says what the difference is.
func (o Options) answerFunc() func(answer.Request, string, string, func(string)) (string, []string, error) {
	if o.Answer != nil {
		return o.Answer
	}
	return answer.RunFor(qmdOptions(o.StateDir)...)
}

// qmdAttempts is one try and exactly one retry, the rule of this stage for the
// hop from a long-lived service to qmd: dies the daemon underneath the
// service, the call is repeated once against a daemon re-probed and, where
// nothing answered, restarted under the shared lock, and what fails again is
// an error -- never an empty hit list. The command line keeps the reference's
// three attempts (search.ColdAttempts), which stage 1b-1 signed off.
const qmdAttempts = 2

// qmdOptions are what this service's qmd port is built with: the lock of this
// service's state directory, and the attempts above. The lock has to be named
// rather than left to the global state directory -- a service running on a
// state directory of its own would otherwise take a lock file it never uses,
// and its start would race the very starter it is meant to be serialised with.
func qmdOptions(stateDir string) []search.QmdMcpOption {
	return []search.QmdMcpOption{
		search.WithQmdLock(QmdLockPath(stateDir)),
		search.WithColdAttempts(qmdAttempts),
	}
}

// StopURL turns one channel's MCP address into the stop address beside it, so
// that no caller has to assemble the path itself.
func StopURL(endpointURL string) string {
	return strings.TrimSuffix(endpointURL, MCPPath) + StopPath
}

// Run takes serve.lock, binds both listeners, writes serve.json and blocks
// until ctx ends or a stop request arrives. It returns nil on an orderly stop
// and ErrAlreadyRunning when another serve holds the lock.
//
// The order matters: the lock first, so a second serve never binds a port; then
// the listeners, so serve.json can name real addresses; then the state file,
// which is the last thing a bridge waits for.
//
//coverage:exempt one arm resists: BuildIdentity fails only where the operating system cannot name the running program or the binary is gone while it runs, and nothing short of a seam of its own reaches that; every other arm, the failing binds included, is driven by a test
func Run(ctx context.Context, opts Options) error {
	// The lock file is opened, not created by a parent: on a fresh machine the
	// state directory does not exist yet, and TryAcquire does not make it.
	if err := os.MkdirAll(opts.StateDir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", opts.StateDir, err)
	}
	handle, held, err := lock.TryAcquire(LockPath(opts.StateDir))
	if err != nil {
		return err
	}
	if !held {
		return ErrAlreadyRunning
	}
	defer handle.Release()

	var once sync.Once
	stopped := make(chan struct{})
	// Once, because two stop requests are two requests, not two shutdowns.
	stop := func() { once.Do(func() { close(stopped) }) }

	local, err := listen(privacy.ChannelLocal, opts, stop)
	if err != nil {
		return err
	}
	defer local.close()
	cloud, err := listen(privacy.ChannelCloud, opts, nil)
	if err != nil {
		return err
	}
	defer cloud.close()

	// Serving starts before the state is written: a bridge that reads
	// serve.json the instant it appears must find a listener that answers.
	local.start()
	cloud.start()

	executable, size, modTime, err := BuildIdentity()
	if err != nil {
		return err
	}
	if err := WriteState(opts.StateDir, &State{
		Local:      local.endpoint,
		Cloud:      cloud.endpoint,
		PID:        os.Getpid(),
		Executable: executable,
		Size:       size,
		ModTime:    modTime,
		BrokeAway:  opts.BrokeAway,
	}); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
	case <-stopped:
	}
	return nil
}

// listenTCP is the seam a test binds through: the second channel refusing its
// port is the one arm where a mistake costs the most -- the lock is already
// held by then -- and it cannot be provoked from outside.
var listenTCP = net.Listen

// channel is one of the two listeners with everything that belongs to it.
type channel struct {
	endpoint Endpoint
	listener net.Listener
	server   *http.Server
	done     chan error
	// started says whether Serve ever took the listener over. Everything about
	// closing differs between the two states, and only start knows.
	started bool
}

// listen binds one channel's loopback port and builds the server behind it.
// The port is 0 and the operating system chooses; the address that comes back
// is what serve.json names. stop is the shutdown the stop route triggers, and
// nil on the channel that has no such route.
func listen(name privacy.Channel, opts Options, stop func()) (*channel, error) {
	listener, err := listenTCP("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("bind the %s listener: %w", name, err)
	}
	token := NewToken()
	address := "http://" + listener.Addr().String()
	return &channel{
		endpoint: Endpoint{URL: address + MCPPath, Token: token},
		listener: listener,
		server:   &http.Server{Handler: RequireToken(token, protected(handlers(name, opts, stop)))},
		done:     make(chan error, 1),
	}, nil
}

// handlers is the routing of one channel: MCP, and on the local channel the
// stop request beside it.
func handlers(name privacy.Channel, opts Options, stop func()) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "loomux", Version: "1"}, &mcp.ServerOptions{
		SetCacheable: setCacheable,
	})
	servebrain.Register(server, name, servebrain.Deps{
		Answer:      opts.answerFunc(),
		RegistryDir: opts.RegistryDir,
		LegacyDir:   opts.LegacyDir,
	})
	mux := http.NewServeMux()
	mux.Handle(MCPPath, mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{
			// Stateless follows the 2026-07-28 direction (SEP-2567). It also
			// kills the hardest restart case: there is no session to go
			// unknown, only a port that is gone.
			Stateless: true,
		}))
	if stop != nil {
		mux.HandleFunc("POST "+StopPath, func(w http.ResponseWriter, _ *http.Request) {
			// Answer first, shut down after: the caller wants to hear that the
			// request arrived, and it arrived over the connection that is
			// about to close.
			w.WriteHeader(http.StatusNoContent)
			stop()
		})
	}
	return mux
}

// protected puts cross-origin protection in front of everything, not only in
// front of MCP: without it any web page in the user's browser could shoot at
// 127.0.0.1, and the token does not help once it has been in a URL.
//
// The SDK has a field for this, and it is deprecated in favour of exactly this
// wrapping; wrapping also covers the stop route, which the field would not.
func protected(next http.Handler) http.Handler {
	return http.NewCrossOriginProtection().Handler(next)
}

// setCacheable lets a host cache what it listed. Five static tools make that
// free, and the tool list is the one result a host asks for on every start.
func setCacheable(_ context.Context, _ mcp.Request, c *mcp.Cacheable) {
	c.TTLMs = int(mcptools.CacheTTL / time.Millisecond)
	c.CacheScope = mcptools.CacheScope
}

// start serves in the background. The result waits in done until close reads
// it, so no goroutine outlives the run.
func (c *channel) start() {
	c.started = true
	go func() { c.done <- c.server.Serve(c.listener) }()
}

// close gives the port back. What that takes depends on whether Serve ever ran:
//
// Before start, the listener is bound and nothing else holds it. Shutdown would
// close nothing -- it only closes listeners Serve registered -- and waiting for
// a result that no goroutine will ever send would hang here forever, with
// serve.lock held, because Release is deferred earlier and runs later. That is
// the state a second channel leaves behind when its port refuses to bind.
//
// After start, Shutdown is the orderly way: it closes the listener, lets what
// is in flight finish, and makes Serve return. Neither error is worth a return
// value -- Shutdown fails only by running out of time, Serve always ends in
// ErrServerClosed -- but the wait is: nothing may be left listening once close
// has returned, or on Windows the test's own directory cannot go away.
func (c *channel) close() {
	if !c.started {
		_ = c.listener.Close()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	_ = c.server.Shutdown(ctx)
	<-c.done
}
