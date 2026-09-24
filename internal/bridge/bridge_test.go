package bridge_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/bridge"
	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/mcptools"
	"github.com/xidus90/loomux/internal/serve"
	servebrain "github.com/xidus90/loomux/internal/serve/brain"
	servegraph "github.com/xidus90/loomux/internal/serve/graph"
)

// runTimeout bounds every wait in this file. A test that would otherwise hang
// has to fail instead, and none of them needs more than a moment on loopback.
const runTimeout = 10 * time.Second

// service is a fake serve: one HTTP listener with one tool handler behind the
// token middleware, and a serve.json that names it.
type service struct {
	StateDir string
	stop     func()
	// requests counts what arrived over HTTP, the handshake included. It is
	// how a test tells a reused session from one negotiated again.
	requests *atomic.Int64
	// gets records the GET requests, which a stateless service answers with
	// 405 and no client of ours may send.
	gets *atomic.Int64
}

// fakeService builds one in a state directory of its own and holds serve.lock
// for as long as the test runs -- a bridge that found the lock free would
// rightly decide that nothing is running and try to start a service.
func fakeService(t *testing.T, handler mcp.ToolHandler) *service {
	t.Helper()
	dir := t.TempDir()
	holdServeLock(t, dir)
	return fakeServiceIn(t, dir, handler)
}

// fakeServiceIn puts a service into an existing state directory and overwrites
// the serve.json there. Two of them in one directory are how a service that
// moved to another port is staged.
func fakeServiceIn(t *testing.T, dir string, handler mcp.ToolHandler) *service {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "fake-serve", Version: "1"}, nil)
	for _, tool := range mcptools.Tools() {
		server.AddTool(tool, handler)
	}
	const token = "token-of-the-fake-service"
	requests, gets := &atomic.Int64{}, &atomic.Int64{}
	counted := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			requests.Add(1)
			if req.Method == http.MethodGet {
				gets.Add(1)
			}
			next.ServeHTTP(w, req)
		})
	}
	listener := httptest.NewServer(counted(serve.RequireToken(token, mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true}))))
	t.Cleanup(listener.Close)
	endpoint := serve.Endpoint{URL: listener.URL + serve.MCPPath, Token: token}
	// A modification time in the future, so that this build is never the newer
	// one: a test of the forwarding must not restart anything.
	if err := serve.WriteState(dir, &serve.State{
		Local:      endpoint,
		Cloud:      endpoint,
		PID:        os.Getpid(),
		Executable: "fake",
		ModTime:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("write the state file: %v", err)
	}
	return &service{StateDir: dir, stop: listener.Close, requests: requests, gets: gets}
}

// holdServeLock takes serve.lock the way a running service holds it.
func holdServeLock(t *testing.T, dir string) {
	t.Helper()
	handle, free, err := lock.TryAcquire(serve.LockPath(dir))
	if err != nil || !free {
		t.Fatalf("take %s: free=%v err=%v", serve.LockPath(dir), free, err)
	}
	t.Cleanup(func() { _ = handle.Release() })
}

// connectBridge runs a bridge over an in-memory pair instead of the host's
// stdio and hands back the host's side of it.
func connectBridge(t *testing.T, opts bridge.Options) *mcp.ClientSession {
	t.Helper()
	return connectBridgeWith(t, opts, nil)
}

func connectBridgeWith(t *testing.T, opts bridge.Options, clientOpts *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()
	if opts.Spawn == nil {
		// No test starts a service. A spawner that refuses makes that a
		// failure of the test rather than a process nobody reaps.
		opts.Spawn = func(string) (bool, error) { return false, errors.New("no test may start a service") }
	}
	serverSide, clientSide := mcp.NewInMemoryTransports()
	opts.Transport = serverSide
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- bridge.Run(ctx, opts) }()

	session, err := mcp.NewClient(&mcp.Implementation{Name: "host", Version: "1"}, clientOpts).
		Connect(ctx, clientSide, nil)
	if err != nil {
		cancel()
		t.Fatalf("connect to the bridge: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		select {
		case <-done:
		case <-time.After(runTimeout):
			t.Error("bridge.Run did not return after the host went away")
		}
	})
	return session
}

// toolsOf is the list as it comes over the wire.
func toolsOf(t *testing.T, session *mcp.ClientSession) []*mcp.Tool {
	t.Helper()
	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	return res.Tools
}

func TestToolsListIsAnsweredWithoutAService(t *testing.T) {
	// The descriptions are static, and fetching them would put a cold qmd start
	// inside the host's handshake. No service runs in this test at all.
	session := connectBridge(t, bridge.Options{StateDir: t.TempDir()})
	tools := toolsOf(t, session)
	if len(tools) != 12 {
		t.Fatalf("got %d tools without a service, want 12", len(tools))
	}
	// The order is the SDK's, which sorts by name; the brief's brain_search
	// would be a test of mcptools' registration order, which is not observable
	// over the protocol.
	if tools[0].Name != "brain_catalog" {
		t.Errorf("first tool is %q", tools[0].Name)
	}
}

// TestTheBridgeOffersTheToolsItIsGiven: the command line hands down the list
// the project's [modules] leave, and nothing else may reach the host.
func TestTheBridgeOffersTheToolsItIsGiven(t *testing.T) {
	tools := toolsOf(t, connectBridge(t, bridge.Options{StateDir: t.TempDir(), Tools: mcptools.Brain()}))
	if len(tools) != len(mcptools.Brain()) {
		t.Fatalf("got %d tools, want the brain's %d", len(tools), len(mcptools.Brain()))
	}
	for _, tool := range tools {
		if !strings.HasPrefix(tool.Name, "brain_") {
			t.Errorf("%s offered, the bridge was given only the brain's", tool.Name)
		}
	}
}

// TestAnEmptyToolListOffersNothing: empty is a project with every module off,
// not the unset field -- only nil falls back to every tool.
func TestAnEmptyToolListOffersNothing(t *testing.T) {
	tools := toolsOf(t, connectBridge(t, bridge.Options{StateDir: t.TempDir(), Tools: []*mcp.Tool{}}))
	if len(tools) != 0 {
		t.Fatalf("got %d tools from an empty list", len(tools))
	}
}

func TestTheListIsByteIdenticalToTheServices(t *testing.T) {
	bridged, err := json.Marshal(toolsOf(t, connectBridge(t, bridge.Options{StateDir: t.TempDir()})))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	served, err := json.Marshal(toolsOf(t, connectService(t)))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(bridged) != string(served) {
		t.Errorf("the bridge's list and the service's list differ:\n%s\n%s", bridged, served)
	}
}

// connectService registers the tools the way serve does and hands back a
// session to them. The comparison has to be list against list over the wire:
// the order there is the SDK's, not the one mcptools holds.
func connectService(t *testing.T) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "loomux", Version: "1"}, nil)
	servebrain.Register(server, privacy.ChannelLocal, servebrain.Deps{
		Answer: func(answer.Request, string, string, func(string)) (string, []string, error) {
			return "", nil, nil
		},
	})
	// Only the list is compared, so the graph tools need no dependencies.
	servegraph.Register(server, privacy.ChannelLocal, servegraph.Deps{})
	serverSide, clientSide := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx, serverSide) }()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "host", Version: "1"}, nil).Connect(ctx, clientSide, nil)
	if err != nil {
		cancel()
		t.Fatalf("connect to the service: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		select {
		case <-done:
		case <-time.After(runTimeout):
			t.Error("the service did not return")
		}
	})
	return session
}

func TestACallIsForwardedUnchanged(t *testing.T) {
	// Name to name, arguments to arguments. The fake service records what
	// arrived.
	var seen *mcp.CallToolParamsRaw
	service := fakeService(t, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		seen = req.Params
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
	})
	session := connectBridge(t, bridge.Options{StateDir: service.StateDir})
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "brain_read",
		Arguments: map[string]any{"scope": "wiki", "relative": "index.md"},
	}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if seen.Name != "brain_read" {
		t.Errorf("the service saw %q", seen.Name)
	}
	if !strings.Contains(string(seen.Arguments), `"relative":"index.md"`) {
		t.Errorf("the service saw arguments %s", seen.Arguments)
	}
}

func TestARefusedConnectionIsRetriedExactlyOnce(t *testing.T) {
	// A commit rebuilds bin/loomux.exe, a newer bridge restarts serve, and every
	// other bridge's connection dies with it. One retry hides that from a host
	// that did nothing wrong. Two retries would hide a real outage.
	attempts := 0
	service := fakeService(t, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("connection refused")
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
	})
	session := connectBridge(t, bridge.Options{StateDir: service.StateDir})
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if attempts != 2 {
		t.Errorf("the service was called %d times, want 2", attempts)
	}
	if res.IsError {
		t.Error("the retried call came back as an error")
	}
}

func TestTheRetryReadsTheStateFileAgain(t *testing.T) {
	// The one retry is worth nothing if it goes to the port that just died: a
	// restarted service listens somewhere else. Here the first service is taken
	// away after the bridge has spoken to it, and a second one takes its place
	// in the same state directory on another port.
	first := fakeService(t, ok)
	session := connectBridge(t, bridge.Options{StateDir: first.StateDir})
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"}); err != nil {
		t.Fatalf("the first call: %v", err)
	}
	first.stop()

	reached := false
	fakeServiceIn(t, first.StateDir, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		reached = true
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
	})
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"}); err != nil {
		t.Fatalf("the call after the move: %v", err)
	}
	if !reached {
		t.Error("the retry did not reach the service at its new address")
	}
}

// ok is the handler of a service that has nothing to say but yes.
func ok(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
}

func TestAnOutageIsAnErrorAndNeverAnEmptyResult(t *testing.T) {
	// Reporting an outage as an empty hit list would teach the model to read a
	// dead service as "nothing found". And a service that is down for good is
	// asked twice and no oftener: a third attempt would hide an outage the
	// host has to learn about.
	attempts := 0
	service := fakeService(t, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		attempts++
		return nil, errors.New("connection refused")
	})
	session := connectBridge(t, bridge.Options{StateDir: service.StateDir})
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_search",
		Arguments: map[string]any{"query": "anything"}})
	if err == nil {
		t.Fatal("an outage came back as a result")
	}
	if !strings.Contains(err.Error(), "loomux serve status") {
		t.Errorf("the error does not say what to do: %v", err)
	}
	if attempts != 2 {
		t.Errorf("the dead service was asked %d times, want 2", attempts)
	}
}

func TestASecondCallKeepsTheNegotiatedSession(t *testing.T) {
	// One negotiation per bridge, not one per call: the handshake is two more
	// round trips on every question a model asks. And no GET at all -- a
	// stateless service answers one with 405, which is why the client opens no
	// standalone stream.
	service := fakeService(t, ok)
	session := connectBridge(t, bridge.Options{StateDir: service.StateDir})
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"}); err != nil {
		t.Fatalf("the first call: %v", err)
	}
	before := service.requests.Load()
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"}); err != nil {
		t.Fatalf("the second call: %v", err)
	}
	if grew := service.requests.Load() - before; grew != 1 {
		t.Errorf("the second call cost %d requests, want 1", grew)
	}
	if gets := service.gets.Load(); gets != 0 {
		t.Errorf("the bridge sent %d GET requests to a stateless service", gets)
	}
}

func TestACallTheHostTookBackIsNotRetried(t *testing.T) {
	// A cancelled call is not an outage. Renegotiating for it would throw away
	// a session that is perfectly well and spend the one retry on nothing.
	// The service holds the call until the test lets go, so that the host can
	// take it back while it is in flight.
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	service := fakeService(t, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entered <- struct{}{}
		<-release
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
	})
	session := connectBridge(t, bridge.Options{StateDir: service.StateDir})

	ctx, cancel := context.WithCancel(context.Background())
	answered := make(chan error, 1)
	go func() {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "brain_status"})
		answered <- err
	}()
	select {
	case <-entered:
	case <-time.After(runTimeout):
		t.Fatal("the service never saw the call")
	}
	cancel()
	select {
	case err := <-answered:
		if err == nil {
			t.Fatal("a call the host took back came back as a result")
		}
	case <-time.After(runTimeout):
		t.Fatal("the cancelled call never came back")
	}
	// Nothing arrives a second time: the retry belongs to an outage, and this
	// was none.
	select {
	case <-entered:
		t.Error("the bridge retried a call the host had taken back")
	case <-time.After(250 * time.Millisecond):
	}
	close(release)
}

func TestAServiceThatVanishesWithItsStateFileIsAnOutage(t *testing.T) {
	// The retry re-reads serve.json. When there is nothing left to read, the
	// call is an error naming the command a human can run -- not a result.
	service := fakeService(t, ok)
	session := connectBridge(t, bridge.Options{StateDir: service.StateDir})
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"}); err != nil {
		t.Fatalf("the first call: %v", err)
	}
	service.stop()
	if err := os.Remove(serve.StatePath(service.StateDir)); err != nil {
		t.Fatalf("remove the state file: %v", err)
	}
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"})
	if err == nil {
		t.Fatal("a vanished service came back as a result")
	}
	if !strings.Contains(err.Error(), "loomux serve status") {
		t.Errorf("the error does not say what to do: %v", err)
	}
}

func TestACallWithoutAServiceIsAnOutage(t *testing.T) {
	// Nothing to read, nothing to connect to: still an error that names the
	// command a human can run, and still not an empty result.
	session := connectBridge(t, bridge.Options{StateDir: t.TempDir()})
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"})
	if err == nil {
		t.Fatal("a missing service came back as a result")
	}
	if !strings.Contains(err.Error(), "loomux serve status") {
		t.Errorf("the error does not say what to do: %v", err)
	}
}

func TestTheProgressTokenTravelsDownwards(t *testing.T) {
	// Without it the warming hint dies in the bridge.
	var token any
	service := fakeService(t, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token = req.Params.GetProgressToken()
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
	})
	session := connectBridge(t, bridge.Options{StateDir: service.StateDir})
	params := &mcp.CallToolParams{Name: "brain_status"}
	params.SetProgressToken("p1")
	if _, err := session.CallTool(context.Background(), params); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if token == nil {
		t.Error("the service saw no progress token")
	}
}

func TestAProgressNoteReachesTheHost(t *testing.T) {
	// The token travelling down is half the way. A note that stopped in the
	// bridge would leave the host waiting in silence through a cold start.
	service := fakeService(t, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
			ProgressToken: req.Params.GetProgressToken(),
			Message:       "warming the index",
		}); err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
	})
	notes := make(chan string, 4)
	session := connectBridgeWith(t, bridge.Options{StateDir: service.StateDir}, &mcp.ClientOptions{
		ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
			notes <- req.Params.Message
		},
	})
	params := &mcp.CallToolParams{Name: "brain_search", Arguments: map[string]any{"query": "anything"}}
	params.SetProgressToken("p1")
	if _, err := session.CallTool(context.Background(), params); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	select {
	case note := <-notes:
		if note != "warming the index" {
			t.Errorf("the host heard %q", note)
		}
	case <-time.After(runTimeout):
		t.Error("the host heard nothing")
	}
}

func TestTheCloudBridgeCallsTheCloudAddress(t *testing.T) {
	// The channel is the address, not an argument: a bridge started with
	// --channel cloud speaks to the cloud listener and to no other.
	dir := t.TempDir()
	holdServeLock(t, dir)
	local := fakeServiceIn(t, dir, ok)
	reached := false
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		http.Error(w, "the cloud listener answers", http.StatusTeapot)
	}))
	t.Cleanup(cloud.Close)
	state, err := serve.ReadState(local.StateDir)
	if err != nil {
		t.Fatalf("read the state file: %v", err)
	}
	state.Cloud = serve.Endpoint{URL: cloud.URL + serve.MCPPath, Token: "cloud-token"}
	if err := serve.WriteState(dir, state); err != nil {
		t.Fatalf("write the state file: %v", err)
	}
	session := connectBridge(t, bridge.Options{StateDir: dir, Channel: privacy.ChannelCloud})
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"}); err == nil {
		t.Fatal("the teapot answered as a service")
	}
	if !reached {
		t.Error("the cloud bridge did not call the cloud listener")
	}
}

func TestAProgressNoteDoesNotBlockAReconnect(t *testing.T) {
	// The deadlock this guards: a progress note arriving while another call
	// renegotiates. Closing a session waits for what that session still has in
	// flight, and a note is dispatched on exactly that path -- so a close that
	// runs under the same lock the note wants never returns, and with it the
	// whole bridge stops for good.
	//
	// One tool streams notes, another fails every time and therefore
	// renegotiates on every call.
	streaming := make(chan struct{})
	service := fakeService(t, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if req.Params.Name != "brain_search" {
			return nil, errors.New("connection refused")
		}
		close(streaming)
		for i := 0; i < 200; i++ {
			_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
				ProgressToken: req.Params.GetProgressToken(),
				Message:       "warming the index",
			})
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
	})
	session := connectBridge(t, bridge.Options{StateDir: service.StateDir})

	notes := make(chan error, 1)
	go func() {
		params := &mcp.CallToolParams{Name: "brain_search", Arguments: map[string]any{"query": "anything"}}
		params.SetProgressToken("p1")
		_, err := session.CallTool(context.Background(), params)
		notes <- err
	}()
	<-streaming

	reconnects := make(chan struct{}, 1)
	go func() {
		for i := 0; i < 20; i++ {
			_, _ = session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"})
		}
		close(reconnects)
	}()

	for waiting := 2; waiting > 0; waiting-- {
		select {
		case err := <-notes:
			if err != nil {
				t.Errorf("the streaming call: %v", err)
			}
			notes = nil
		case <-reconnects:
			reconnects = nil
		case <-time.After(runTimeout):
			t.Fatal("the bridge stopped answering: a note and a reconnect blocked each other")
		}
	}
}

// TestTheBridgeEndsWithItsHostWhileACallIsInFlight is the one that measures
// what a comment cannot: a bridge whose host goes away while a call is parked
// in the service must still return from Run.
//
// The context a tool handler is given never reports a host that went away --
// the SDK hands it out through jsonrpc2's notDone -- so without the binding in
// untilTheHostIsGone the parked call never ends, the session close inside
// mcp.Server.Run waits for it forever, and the process outlives its host. That
// is not a hypothetical: the path exists and nothing bounds it, and two bridges
// found alive holding open the bin/loomux.exe they were started from are what
// brought it to light -- whether it was their own way there is not established.
//
// The service is released from the test's own cleanup, so nothing is left
// parked when this returns.
func TestTheBridgeEndsWithItsHostWhileACallIsInFlight(t *testing.T) {
	entered, release := make(chan struct{}, 1), make(chan struct{})
	svc := fakeService(t, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Buffered and non-blocking: a retry of the same call must not park a
		// second goroutine on a send nobody reads.
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		return &mcp.CallToolResult{}, nil
	})
	t.Cleanup(func() { close(release) })

	serverSide, clientSide := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- bridge.Run(ctx, bridge.Options{
			StateDir:  svc.StateDir,
			Transport: serverSide,
			Spawn:     func(string) (bool, error) { return false, errors.New("no test may start a service") },
		})
	}()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "host", Version: "1"}, nil).Connect(ctx, clientSide, nil)
	if err != nil {
		cancel()
		t.Fatalf("connect to the bridge: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	// The host's own context, not the run's: a call that was cancelled from
	// above would prove nothing about a host that went away without saying so.
	parked := make(chan error, 1)
	go func() {
		_, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"})
		parked <- err
	}()
	select {
	case <-entered:
	case <-time.After(runTimeout):
		cancel()
		t.Fatal("the service never saw the call")
	}

	// This is the host going away, and the only thing that says so.
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run returned %v, want the cancellation that ended it", err)
		}
	case <-time.After(runTimeout):
		t.Fatalf("bridge.Run did not return while a call was in flight\n%s", stacks())
	}
	// The parked call ends with the run, and as a cancellation rather than as
	// an outage: nothing failed down there, so there is nothing to look at.
	select {
	case err := <-parked:
		if err == nil {
			t.Error("the parked call came back as a result")
		} else if strings.Contains(err.Error(), "loomux serve status") {
			t.Errorf("a cancelled call was reported as an outage: %v", err)
		}
	case <-time.After(runTimeout):
		t.Errorf("the parked call never came back\n%s", stacks())
	}

}

// stacks is every goroutine of this test binary, for the one failure that is
// unreadable without them: a Run that did not return says nothing about where
// it stands.
func stacks() []byte {
	buf := make([]byte, 1<<20)
	return buf[:runtime.Stack(buf, true)]
}
