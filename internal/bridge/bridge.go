package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/mcptools"
	"github.com/xidus90/loomux/internal/serve"
)

// Options are what one bridge run needs.
type Options struct {
	// StateDir is where serve.json and serve.lock live. Everything hangs off
	// it and off no fixed path, so that a test isolates its service from the
	// real one.
	StateDir string
	// Channel is the address this bridge calls, never an argument of a call:
	// the channel is the address. Empty means local.
	Channel privacy.Channel
	// Spawn starts a service and falls back to the real detached start when
	// nil. A test always brings its own; nothing else may start a process.
	Spawn func(stateDir string) (brokeAway bool, err error)
	// Transport is what the host speaks over, and nil means its stdio. It is
	// the seam a test connects through, the same kind serve.Options.Answer is.
	Transport mcp.Transport
}

// transport is the host's connection, stdio unless a caller named one.
func (o Options) transport() mcp.Transport {
	if o.Transport != nil {
		return o.Transport
	}
	return &mcp.StdioTransport{}
}

// endpoint is the address and token of this bridge's channel. Cloud only when
// it is asked for by name: an unset channel is the local one, as everywhere
// else in loomux.
func (o Options) endpoint(state *serve.State) serve.Endpoint {
	if o.Channel == privacy.ChannelCloud {
		return state.Cloud
	}
	return state.Local
}

// Run serves one host until it goes away.
func Run(ctx context.Context, opts Options) error {
	// This run's own context, and the one every call in flight is bound to: it
	// is the only thing that can tell a handler the host is gone. See
	// untilTheHostIsGone.
	ctx, cancel := context.WithCancel(ctx)
	server := mcp.NewServer(&mcp.Implementation{Name: "loomux", Version: "1"}, &mcp.ServerOptions{
		SetCacheable: setCacheable,
	})
	b := &bridge{opts: opts, ready: make(chan struct{}), hostGone: ctx}
	for _, tool := range mcptools.Tools() {
		// Answered here rather than fetched: the descriptions are static, and
		// asking would put a cold start inside the handshake. The same list
		// object as the service's, so the two cannot drift.
		server.AddTool(tool, b.forward)
	}

	// The two deferred calls in this order, so that they run in the other one:
	// cancel first, so that a start still waiting for its service gives up,
	// and only then the close that waits for it. A host that hung up must not
	// hold this process for the rest of a start timeout.
	defer b.close()
	defer cancel()

	// Nudged in the background: making sure a service exists blocks for the
	// seconds of a cold start, and the host must see a ready server long
	// before that. A failed start stays a failed start -- it must not take the
	// answering front down with it, or the host learns of the outage from a
	// server that vanishes instead of from the call that needed it.
	go func() {
		defer close(b.ready)
		_ = ensure(ctx, opts, opts.deps())
	}()

	return server.Run(ctx, opts.transport())
}

// setCacheable lets the host cache what it listed, exactly as serve does. Seven
// static tools make that free, and the list is what a host asks for on every
// start.
func setCacheable(_ context.Context, _ mcp.Request, c *mcp.Cacheable) {
	c.TTLMs = int(mcptools.CacheTTL / time.Millisecond)
	c.CacheScope = mcptools.CacheScope
}

// bridge is one host's redirector: the session downwards, and the session
// upwards that progress travels back through.
//
// Neither session sits behind a lock that something else has to wait for, and
// that is not tidiness but the fix for a deadlock that stopped a bridge for
// good: closing a client session waits for what that session still has in
// flight (mcp/client.go:579 into jsonrpc2/conn.go:504-509), and a progress
// notification is dispatched on exactly that path (mcp/client.go:1515). A
// close under a lock that a notification needs therefore never returns, and
// with it nothing of this bridge ever answers again.
//
// So: host is atomic, and **no session is closed while connecting is held** --
// in connect and in close alike, the session leaves the field under the lock
// and is closed once the lock is released. TestNoSessionIsClosedUnderTheLock
// measures that rather than believing this paragraph. The rule is wider than
// the deadlock that made it: closing waits for outgoing calls too
// (jsonrpc2/conn.go:144-145 and :504-508), so a close under the lock would
// hold every other call of this bridge for as long as the slowest question on
// the dying session.
type bridge struct {
	opts Options
	// ready closes when the background start has had its say, whether it
	// worked or not. The first call waits for it rather than racing a service
	// that is still coming up.
	ready chan struct{}
	// hostGone is this run's context, and it is done once the host is gone.
	// Every handler is bound to it; untilTheHostIsGone says why a handler
	// cannot learn that from the context it is handed.
	hostGone context.Context
	// host is where progress goes back up to. Every call writes it and every
	// note reads it, so it is the one field that must never wait.
	host atomic.Pointer[mcp.ServerSession]
	// connecting serialises building the session downwards: two calls failing
	// at the same moment need one new session, not two. Nothing that blocks on
	// the network or on another goroutine happens under it except the
	// handshake it exists for.
	connecting sync.Mutex
	session    atomic.Pointer[mcp.ClientSession]
	// closeSession stands in for (*mcp.ClientSession).Close in the one test
	// that measures the invariant above instead of believing it.
	closeSession func(*mcp.ClientSession)
}

// forward is the whole job: name to name, arguments to arguments, result back.
func (b *bridge) forward(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx, released := b.untilTheHostIsGone(ctx)
	defer released()
	b.remember(req.Session)
	b.awaitService(ctx)
	session, err := b.connect(ctx, nil)
	if err != nil {
		return nil, outage(err)
	}
	res, err := session.CallTool(ctx, downwards(req))
	if err == nil {
		return res, nil
	}
	// A host that took its question back is not an outage, and renegotiating
	// for it would throw away a session that is perfectly well. The error goes
	// up as it came: sending a host that cancelled -- or one that has just
	// gone, which is the ordinary way a call ends now -- after `loomux serve
	// status` would name a service that never failed.
	if ctx.Err() != nil {
		return nil, err
	}
	// Exactly one retry, against a freshly read serve.json: a commit rebuilds
	// the binary, a newer bridge restarts serve, and every other bridge's
	// connection dies with it. One retry hides that from a host that did
	// nothing wrong. Two would hide a real outage.
	session, err = b.connect(ctx, session)
	if err != nil {
		return nil, outage(err)
	}
	res, err = session.CallTool(ctx, downwards(req))
	if err != nil {
		return nil, outage(err)
	}
	return res, nil
}

// untilTheHostIsGone binds one call to the life of this run, and hands back
// the release that ends that binding.
//
// The context a tool handler is given never learns that the host went away.
// The SDK wraps the connection's context in jsonrpc2's notDone before a
// handler sees it (internal/jsonrpc2/conn.go:233 and :803-812): a handler is
// meant to be cancelled by the peer's cancel notification or by a transport
// that failed, and by nothing else. A host that simply ends -- Ctrl+C, the
// signal that ends `loomux mcp`, a cancelled context anywhere above -- is
// neither of those.
//
// Unbound, a call parked downwards then never returns. The session close in
// mcp.Server.Run waits for what is still in flight, so Run never returns, the
// two defers above it never run, and the process outlives the host it exists
// for -- holding its own binary open, which is how this was found. Measured in
// TestTheBridgeEndsWithItsHostWhileACallIsInFlight.
func (b *bridge) untilTheHostIsGone(ctx context.Context) (context.Context, func()) {
	if b.hostGone == nil {
		// Only Run sets the field, and context.AfterFunc would panic on nil
		// with nothing that says where to look. A bridge built by hand in a
		// test needs a run context of its own.
		panic("bridge: hostGone is unset; build the bridge in Run, or give the literal a hostGone context")
	}
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(b.hostGone, cancel)
	return ctx, func() {
		// Both, and in this order: stop takes this call off the run's context,
		// which would otherwise hold every call ever made for as long as the
		// run lasts, and cancel releases what the call itself still holds.
		stop()
		cancel()
	}
}

// downwards is the host's call as it goes on. Meta travels with it, and the
// progress token sits in Meta: without it the warming hint dies here.
//
// The arguments are passed on as the bytes that arrived. The bridge does not
// know what they mean, and anything it did to them would be the second core
// this layer must not be.
func downwards(req *mcp.CallToolRequest) *mcp.CallToolParams {
	return &mcp.CallToolParams{
		Meta:      req.Params.Meta,
		Name:      req.Params.Name,
		Arguments: json.RawMessage(req.Params.Arguments),
	}
}

// outage says what a human can do about it. It is never an empty result: that
// would read as "nothing found" to the model.
func outage(err error) error {
	return fmt.Errorf("the loomux service did not answer (%w); run `loomux serve status`", err)
}

// remember keeps the host's session, because progress from below has to find
// its way back up to it.
func (b *bridge) remember(host *mcp.ServerSession) {
	b.host.Store(host)
}

// awaitService holds the first call until the background start has had its
// say. It is ordering, not a retry: without it every call that arrives during
// a restart would spend the retry this layer has on a service that was merely
// not there yet.
func (b *bridge) awaitService(ctx context.Context) {
	select {
	case <-b.ready:
	case <-ctx.Done():
	}
}

// connect hands out the session downwards, and builds a new one when there is
// none or when the caller's has just failed. A new one means serve.json read
// again: a restarted service listens on another port.
//
// stale is the session the caller held, and nil when it held none. A host asks
// several things at once, so two calls can fail on the same session at the same
// moment; the second one then finds a session that is not its own and takes it
// instead of closing the one the first has just made. Without that only the
// last reconnecter would have a live session, and every other call would report
// an outage that this very retry exists to hide.
func (b *bridge) connect(ctx context.Context, stale *mcp.ClientSession) (*mcp.ClientSession, error) {
	// Asked before the lock: a call that only wants the session that is there
	// never queues behind somebody else's handshake.
	if live := b.session.Load(); live != nil && live != stale {
		return live, nil
	}
	return b.renew(ctx, stale)
}

// renew builds the session downwards, one at a time.
//
// It is the queue, and everything that happens in it happens for the call that
// waits behind: it asks again, because while it waited another call may have
// renegotiated and that session is the one to use.
func (b *bridge) renew(ctx context.Context, stale *mcp.ClientSession) (*mcp.ClientSession, error) {
	session, dying, err := b.swap(ctx, stale)
	// Closed last, with nothing of this bridge held. Closing a session waits
	// for what it still has in flight -- a progress note being dispatched, a
	// call still outstanding -- and no other call of this bridge may wait
	// behind that. It is also what keeps a handler that took a lock of ours
	// from deadlocking the whole bridge, which is what happened when this
	// close ran under the lock.
	if dying != nil {
		b.shut(dying)
	}
	if err != nil {
		return nil, err
	}
	return session, nil
}

// swap is the whole of the locked region, and it is a function of its own so
// that a defer releases the lock.
//
// The two manual unlocks it replaces were correct, but a panic in the
// handshake -- serve.ReadState, mcp.NewClient, Connect -- would have left the
// lock held for the life of the process, and a bridge whose reconnect lock is
// held answers nothing ever again. That is the deadlock of the first fix round
// through another door, and worth one function to shut.
//
// It hands back the session to use and, separately, the one the caller closes
// once this has returned. On the early way out that is none: the session
// handed back is another call's and very much alive.
func (b *bridge) swap(ctx context.Context, stale *mcp.ClientSession) (session, dying *mcp.ClientSession, err error) {
	b.connecting.Lock()
	defer b.connecting.Unlock()
	current := b.session.Load()
	if current != nil && current != stale {
		return current, nil, nil
	}
	// Out of the field first, so that nothing hands the dead one out again.
	b.session.Store(nil)
	session, err = b.negotiate(ctx)
	if err == nil {
		b.session.Store(session)
	}
	return session, current, err
}

// negotiate reads where the service is and speaks the handshake to it. It is
// the only thing this bridge does with connecting held, and it is what the
// lock exists for: two calls that failed together need one new session, not
// two.
func (b *bridge) negotiate(ctx context.Context) (*mcp.ClientSession, error) {
	state, err := serve.ReadState(b.opts.StateDir)
	if err != nil {
		return nil, err
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "loomux-bridge", Version: "1"}, &mcp.ClientOptions{
		ProgressNotificationHandler: b.relay,
	})
	return client.Connect(ctx, transportTo(b.opts.endpoint(state)), nil)
}

// shut closes one session downwards.
//
// It is a field rather than a plain call so that a test can stand at the exact
// moment of the close and see that this bridge holds nothing -- the invariant
// the deadlock of the first fix round broke, and one that no comment can hold
// on its own.
func (b *bridge) shut(session *mcp.ClientSession) {
	if b.closeSession != nil {
		b.closeSession(session)
		return
	}
	_ = session.Close()
}

// transportTo is the client side of one channel's address, with that channel's
// token on every request.
//
// DisableStandaloneSSE, because the service is stateless and answers a GET with
// 405: a client that opens a standalone stream there shows the user an error at
// every start that is none.
func transportTo(endpoint serve.Endpoint) mcp.Transport {
	return &mcp.StreamableClientTransport{
		Endpoint:             endpoint.URL,
		HTTPClient:           &http.Client{Transport: bearer{token: endpoint.Token}},
		DisableStandaloneSSE: true,
	}
}

// bearer puts the channel's token on every request. The whole header value is
// what serve.RequireToken compares, scheme included.
type bearer struct {
	token string
}

// RoundTrip sends the request with the token added, on a copy: a RoundTripper
// may not modify the request it is given.
func (b bearer) RoundTrip(req *http.Request) (*http.Response, error) {
	with := req.Clone(req.Context())
	with.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(with)
}

// relay carries a progress notification from the service up to the host. The
// token in it is the host's own, so nothing has to be translated -- the
// notification only has to find the way back.
func (b *bridge) relay(ctx context.Context, req *mcp.ProgressNotificationClientRequest) {
	host := b.host.Load()
	if host == nil {
		return
	}
	// A failed notification must not fail anything: the host asked for the
	// answer, not for the commentary.
	_ = host.NotifyProgress(ctx, req.Params)
}

// close gives up the session downwards and waits for the background start, so
// that nothing of this bridge outlives the call to Run.
//
// The session is taken out of the field under connecting -- a handshake in
// flight would otherwise store its result after this -- and closed once that
// is released, for the same reason connect closes outside it.
func (b *bridge) close() {
	<-b.ready
	b.connecting.Lock()
	session := b.session.Swap(nil)
	b.connecting.Unlock()
	if session != nil {
		b.shut(session)
	}
}
