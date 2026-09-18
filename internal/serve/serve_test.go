package serve_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/mcptools"
	"github.com/xidus90/loomux/internal/serve"
)

// start runs serve in the background and waits for serve.json to appear.
func start(t *testing.T, dir string) (*serve.State, context.CancelFunc) {
	t.Helper()
	state, cancel, _ := startWith(t, dir, serve.Options{StateDir: dir, RegistryDir: dir, LegacyDir: dir})
	return state, cancel
}

// startWith is start with options of the test's own, and it hands back what
// Run returned: the stop test needs the result of an orderly stop, and no
// test may leave a listener behind.
func startWith(t *testing.T, dir string, opts serve.Options) (*serve.State, context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	// Buffered and closed afterwards, so a test that already read the result
	// and the cleanup that reads it again both get an answer.
	done := make(chan error, 1)
	go func() {
		done <- serve.Run(ctx, opts)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("serve did not stop within ten seconds")
		}
	})
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if state, err := serve.ReadState(dir); err == nil && state.Local.URL != "" {
			return state, cancel, done
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("serve did not write its state within ten seconds")
	return nil, cancel, done
}

// tokenRoundTripper puts one listener's token on every request.
type tokenRoundTripper struct {
	token string
}

func (rt tokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+rt.token)
	return http.DefaultTransport.RoundTrip(clone)
}

func tokenClient(token string) *http.Client {
	return &http.Client{Transport: tokenRoundTripper{token: token}}
}

// request sends one bare HTTP request, with the given token or none at all.
func request(t *testing.T, method, url, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

// connect speaks MCP to one channel over its real listener.
func connect(t *testing.T, endpoint serve.Endpoint, opts *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, opts)
	transport := &mcp.StreamableClientTransport{
		Endpoint: endpoint.URL,
		// A stateless server answers GET with 405; a client that opens a
		// standalone SSE stream would see an error that is none.
		DisableStandaloneSSE: true,
		HTTPClient:           tokenClient(endpoint.Token),
	}
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

// noticeAnswer is an answer that says something while it works. The notice has
// to happen before the answer returns: only inside a running request does a
// stateless server have a way back to the client.
func noticeAnswer(_ answer.Request, _, _ string, notice func(string)) (string, []string, error) {
	notice("warming the engine")
	return "ok", nil, nil
}

// giveUp runs serve where it must fail and returns the error. It never lets a
// stuck Run hang the suite.
func giveUp(t *testing.T, dir string) error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- serve.Run(context.Background(), serve.Options{StateDir: dir, RegistryDir: dir, LegacyDir: dir})
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Run started although it could not")
		}
		return err
	case <-time.After(20 * time.Second):
		t.Fatal("Run neither started nor gave up")
		return nil
	}
}

func TestRunSaysSoWhenTheStateDirectoryIsAFile(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "state")
	if err := os.WriteFile(blocked, nil, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	giveUp(t, blocked)
}

func TestRunSaysSoWhenTheLockCannotBeOpened(t *testing.T) {
	dir := t.TempDir()
	// A directory where the lock file belongs: opening it for writing fails,
	// which is the arm between "nobody has taken the lock" and "somebody else
	// holds it".
	if err := os.Mkdir(serve.LockPath(dir), 0o700); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	giveUp(t, dir)
}

func TestRunGivesUpTheLockWhenTheStateCannotBeWritten(t *testing.T) {
	dir := t.TempDir()
	// Both listeners come up and only the state file fails. The listeners and
	// the lock have to go back all the same.
	if err := os.Mkdir(serve.StatePath(dir), 0o700); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	giveUp(t, dir)

	// A serve that kept the lock here would refuse every later start.
	second := make(chan error, 1)
	go func() {
		second <- serve.Run(context.Background(), serve.Options{StateDir: dir, RegistryDir: dir, LegacyDir: dir})
	}()
	select {
	case err := <-second:
		if errors.Is(err, serve.ErrAlreadyRunning) {
			t.Error("the serve that failed kept serve.lock")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the second Run hung")
	}
}

func TestBothListenersAnswerOnLoopbackWithDifferentTokens(t *testing.T) {
	dir := t.TempDir()
	state, _ := start(t, dir)

	for name, endpoint := range map[string]serve.Endpoint{"local": state.Local, "cloud": state.Cloud} {
		if !strings.HasPrefix(endpoint.URL, "http://127.0.0.1:") {
			t.Errorf("%s listens on %q, not on loopback", name, endpoint.URL)
		}
	}
	if state.Local.URL == state.Cloud.URL {
		t.Error("both channels share one address")
	}
	if state.Local.Token == state.Cloud.Token {
		t.Error("both channels share one token")
	}
}

func TestTheWrongTokenIsRefused(t *testing.T) {
	dir := t.TempDir()
	state, _ := start(t, dir)

	// The cloud token on the local listener: a caller who has one channel's
	// secret must not reach the other's door.
	res := request(t, http.MethodPost, state.Local.URL, state.Cloud.Token)
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", res.StatusCode)
	}
}

func TestEveryPathOnTheListenerNeedsTheToken(t *testing.T) {
	dir := t.TempDir()
	state, _ := start(t, dir)

	// Not the MCP route: the token guards the whole listener, so a path that
	// does not even exist is refused before anyone learns that it does not.
	base := strings.TrimSuffix(state.Local.URL, serve.MCPPath)
	for _, path := range []string{"/", "/nowhere", serve.StopPath} {
		res := request(t, http.MethodPost, base+path, "")
		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("POST %s gave status %d, want 401", path, res.StatusCode)
		}
	}
}

func TestASecondServeRefusesToStart(t *testing.T) {
	dir := t.TempDir()
	start(t, dir)

	// Run must return at once: TryAcquire refuses and Run gives up. The timeout
	// is a safety net for a broken implementation, never a wait -- an
	// implementation that blocks here and passes the test is wrong.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := serve.Run(ctx, serve.Options{StateDir: dir, RegistryDir: dir, LegacyDir: dir})
	if !errors.Is(err, serve.ErrAlreadyRunning) {
		t.Errorf("second Run returned %v, want ErrAlreadyRunning", err)
	}
}

func TestStateCarriesTheRunningBuild(t *testing.T) {
	dir := t.TempDir()
	state, _ := start(t, dir)
	if state.Executable == "" || state.Size == 0 || state.ModTime.IsZero() {
		t.Errorf("state does not describe the build: %+v", state)
	}
	if state.PID == 0 {
		t.Errorf("state names no process: %+v", state)
	}
}

func TestStateRecordsWhetherTheServiceBrokeAway(t *testing.T) {
	dir := t.TempDir()
	state, _, _ := startWith(t, dir, serve.Options{StateDir: dir, RegistryDir: dir, LegacyDir: dir, BrokeAway: true})
	if !state.BrokeAway {
		t.Error("a service that broke away must say so in its state")
	}
}

func TestGetIsRefusedInStatelessMode(t *testing.T) {
	dir := t.TempDir()
	state, _ := start(t, dir)

	req, err := http.NewRequest(http.MethodGet, state.Local.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+state.Local.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET gave status %d, want 405", res.StatusCode)
	}
}

func TestAStopRequestEndsTheServiceInOrder(t *testing.T) {
	dir := t.TempDir()
	state, _, done := startWith(t, dir, serve.Options{StateDir: dir, RegistryDir: dir, LegacyDir: dir})

	res := request(t, http.MethodPost, serve.StopURL(state.Local.URL), state.Local.Token)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("stop gave status %d, want 204", res.StatusCode)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("an orderly stop returned %v, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve kept running after a stop request")
	}
}

func TestOnlyTheLocalListenerStops(t *testing.T) {
	dir := t.TempDir()
	state, _ := start(t, dir)

	// The cloud channel is the one a remote model speaks through. Ending the
	// service is not among the things it may ask for.
	res := request(t, http.MethodPost, serve.StopURL(state.Cloud.URL), state.Cloud.Token)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("a stop on the cloud listener gave status %d, want 404", res.StatusCode)
	}
}

func TestTheToolListCarriesItsCacheHints(t *testing.T) {
	dir := t.TempDir()
	state, _ := start(t, dir)
	session := connect(t, state.Local, nil)

	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(res.Tools) != len(mcptools.Tools()) {
		t.Errorf("the listener offers %d tools, want %d", len(res.Tools), len(mcptools.Tools()))
	}
	// The TTL is the assertion that discriminates: the scope defaults to
	// "public" on the wire even when nothing set it.
	if want := int(mcptools.CacheTTL / time.Millisecond); res.TTLMs != want {
		t.Errorf("tools/list offers a TTL of %d ms, want %d", res.TTLMs, want)
	}
	if res.CacheScope != mcptools.CacheScope {
		t.Errorf("tools/list offers scope %q, want %q", res.CacheScope, mcptools.CacheScope)
	}
}

func TestAProgressNotificationReachesTheClientOverHTTP(t *testing.T) {
	dir := t.TempDir()
	state, _, _ := startWith(t, dir, serve.Options{
		StateDir: dir, RegistryDir: dir, LegacyDir: dir, Answer: noticeAnswer,
	})

	heard := make(chan string, 4)
	session := connect(t, state.Local, &mcp.ClientOptions{
		ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
			heard <- req.Params.Message
		},
	})

	params := &mcp.CallToolParams{Name: "brain_status"}
	params.SetProgressToken("p1")
	if _, err := session.CallTool(context.Background(), params); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	select {
	case message := <-heard:
		if message != "warming the engine" {
			t.Errorf("heard %q, want the warming hint", message)
		}
	case <-time.After(5 * time.Second):
		t.Error("no progress notification arrived over HTTP")
	}
}

func TestTheCloudListenerAnswersOnItsOwnChannel(t *testing.T) {
	dir := t.TempDir()
	state, _, _ := startWith(t, dir, serve.Options{
		StateDir: dir, RegistryDir: dir, LegacyDir: dir,
		Answer: func(req answer.Request, _, _ string, _ func(string)) (string, []string, error) {
			return string(req.Channel), nil, nil
		},
	})

	for name, endpoint := range map[string]serve.Endpoint{"local": state.Local, "cloud": state.Cloud} {
		session := connect(t, endpoint, nil)
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "brain_status"})
		if err != nil {
			t.Fatalf("%s CallTool: %v", name, err)
		}
		text, ok := res.Content[0].(*mcp.TextContent)
		if !ok || text.Text != name {
			t.Errorf("%s listener answered on channel %v", name, res.Content[0])
		}
	}
}
