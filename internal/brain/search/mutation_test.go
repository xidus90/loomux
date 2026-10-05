package search_test

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/search"
)

// The tests the mutation round of stage 1b-1 asked for. They are gathered
// here rather than scattered because what they have in common is how they
// were found: each one is the case a surviving mutant proved nobody had
// asked for.
//
// `internal/brain/search` has two halves with two references. The CLI port in
// `qmd.go` mirrors `src/brain/search/qmd.py`, and the tests below cite it by
// line. The MCP daemon path in `http.go` and `mcp.go` is loomux's own -- the
// Python side speaks to qmd only through the command line -- so those tests
// cite the JSON-RPC and SSE framing they implement, or loomux's own constants,
// and never the Go code that has to satisfy them.

// answering is a server that answers every JSON-RPC request with an empty
// result, which is all Handshake and Call need to succeed.
func answering(t *testing.T) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int64 `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  map[string]any{},
		})
	}))
	t.Cleanup(ts.Close)
	return ts
}

// addressOf answers the host and port a test server listens on.
func addressOf(t *testing.T, ts *httptest.Server) (string, int) {
	t.Helper()
	addr, ok := ts.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("the test server does not listen on TCP: %v", ts.Listener.Addr())
	}
	return addr.IP.String(), addr.Port
}

// serving is a server that answers every request with the given body and
// status, so a test can hand the session exactly the bytes it wants read.
func serving(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(ts.Close)
	return ts
}

// refusingTransport fails every request, so a test can see which client the
// session actually used.
type refusingTransport struct{}

func (refusingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("this client refuses")
}

// reportingTransport fails every request with a word for whether the request
// it was handed carries a deadline, which is the only place the timeout of a
// call is visible from outside.
type reportingTransport struct{}

func (reportingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if _, ok := req.Context().Deadline(); ok {
		return nil, errors.New("had deadline")
	}
	return nil, errors.New("no deadline")
}

func TestASessionAskedForNoPortGetsTheDaemonsOwn(t *testing.T) {
	// `DefaultPort` (http.go:19) is the port the qmd daemon listens on, and
	// `brain` reaches it by asking for no port in particular. A zero that
	// stayed zero would build a URL nothing answers.
	s := search.NewHTTPSession(0)
	if s.Port != search.DefaultPort {
		t.Fatalf("Port = %d, want %d", s.Port, search.DefaultPort)
	}
	if !strings.Contains(s.URL, ":8765/mcp") {
		t.Fatalf("URL = %q, want the default port in it", s.URL)
	}
}

func TestAClosedProbeAddressMeansUnreachable(t *testing.T) {
	// `Reachable` asks two questions: is the port at `Host:Port` open, and
	// does the daemon behind the URL answer. The first has to be asked at
	// the address the session names. 192.0.2.1 is TEST-NET-1 (RFC 5737),
	// reserved for documentation and routed nowhere, so the probe there
	// cannot succeed -- while the URL beside it answers perfectly well.
	ts := answering(t)
	_, port := addressOf(t, ts)
	s := &search.HTTPSession{URL: ts.URL, Host: "192.0.2.1", Port: port}
	if s.Reachable() {
		t.Fatal("a session whose probe address is unroutable must not be reachable")
	}
	// The other half of the same decision: a session that names no host
	// probes `DefaultHost`, where the server is.
	unsaid := &search.HTTPSession{URL: ts.URL, Port: port}
	if !unsaid.Reachable() {
		t.Fatal("a session without a host must probe the default one, where the server is")
	}
}

func TestASessionWithoutAHostProbesLocalhostAlsoOnItsIPv6Address(t *testing.T) {
	// An empty host left in the dial address is not refused outright: Go
	// dials `:port` as the local system, which reaches a listener on
	// 127.0.0.1 -- so a server there cannot tell `DefaultHost` from no host
	// at all. `localhost` also names ::1, and the unspecified address does
	// not: a daemon that listens on the IPv6 loopback alone is open to the
	// first probe and closed to the second.
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("no IPv6 loopback to listen on: %v", err)
	}
	ts := httptest.NewUnstartedServer(answering(t).Config.Handler)
	ts.Listener = listener
	ts.Start()
	t.Cleanup(ts.Close)
	_, port := addressOf(t, ts)
	s := &search.HTTPSession{URL: ts.URL, Port: port}
	if !s.Reachable() {
		t.Fatal("a session without a host must probe localhost, which names the IPv6 loopback too")
	}
}

// countingTransport fails every request and counts them, so a test can see
// how often a waiting session asked.
type countingTransport struct{ calls atomic.Int64 }

func (c *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	c.calls.Add(1)
	return nil, errors.New("not yet")
}

// waitingAttempts is how many handshakes WaitUntilReachable tried within
// timeout against a port that is open but a daemon that never answers.
func waitingAttempts(t *testing.T, interval, timeout time.Duration) int64 {
	t.Helper()
	ts := answering(t)
	host, port := addressOf(t, ts)
	counter := &countingTransport{}
	s := &search.HTTPSession{Host: host, Port: port, URL: ts.URL,
		HTTPClient: &http.Client{Transport: counter}, PollInterval: interval}
	if err := s.WaitUntilReachable(timeout); err == nil {
		t.Fatal("a daemon that never answers must not count as reached")
	}
	return counter.calls.Load()
}

func TestAWaitingSessionAsksAgainAfterItsOwnPollInterval(t *testing.T) {
	// A caller that sets `PollInterval` wants to be asked that often. 10 ms
	// within 200 ms is about twenty attempts; the default of 250 ms in its
	// place would allow two.
	if got := waitingAttempts(t, 10*time.Millisecond, 200*time.Millisecond); got < 4 {
		t.Fatalf("attempts = %d, want the session to follow its 10 ms interval", got)
	}
}

func TestAWaitingSessionWithoutAPollIntervalWaitsAQuarterSecond(t *testing.T) {
	// No interval is not "ask without pause": the session waits 250 ms
	// between attempts, so within 50 ms it asks once and, after the pause,
	// once more. Without the default it would spin on the daemon.
	if got := waitingAttempts(t, 0, 50*time.Millisecond); got > 2 {
		t.Fatalf("attempts = %d, want at most two with the 250 ms default", got)
	}
}

// statusTransport answers every request with the given status and a body
// that would be a perfectly good reply.
type statusTransport struct{ status int }

func (s statusTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: s.status, Header: http.Header{},
		Body: io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":{}}`))}, nil
}

func TestAStatusBelowTwoHundredIsNotASuccessfulStatus(t *testing.T) {
	// Success is the 2xx band from below as well: 199 is informational, not
	// an answer. An httptest server cannot send it as the final status (Go
	// turns it into an interim line and answers 200), so a transport hands
	// it over.
	s := &search.HTTPSession{URL: "http://daemon/mcp", HTTPClient: &http.Client{Transport: statusTransport{status: 199}}}
	_, err := s.Call("query", nil)
	if err == nil || !strings.Contains(err.Error(), "server returned status 199") {
		t.Fatalf("err = %v, want the refused status", err)
	}
}

func TestASessionWithoutAURLBuildsOneFromHostAndPort(t *testing.T) {
	// The URL is optional: a session that carries only a host and a port
	// spells the address out itself. Taking an unsaid URL as the address
	// would send every request to nowhere.
	ts := answering(t)
	host, port := addressOf(t, ts)
	s := &search.HTTPSession{Host: host, Port: port}
	if err := s.Handshake(); err != nil {
		t.Fatalf("Handshake: %v", err)
	}
}

func TestTheWaitingErrorNamesTheAddressItWaitedOn(t *testing.T) {
	// The message of `WaitUntilReachable` is what a user reads when no
	// daemon came up, so it has to name the address that was tried -- the
	// host and port of the session, and `DefaultPort` where the session
	// named none. 192.0.2.1 is reserved for documentation (RFC 5737) and
	// answers nothing.
	for _, c := range []struct {
		host string
		port int
		want string
	}{
		{host: "192.0.2.1", port: 1, want: "http://192.0.2.1:1/mcp"},
		{host: "192.0.2.1", port: 0, want: "http://192.0.2.1:8765/mcp"},
		{host: "", port: 1, want: "http://localhost:1/mcp"},
	} {
		s := &search.HTTPSession{Host: c.host, Port: c.port}
		err := s.WaitUntilReachable(0)
		if err == nil {
			t.Fatalf("host %q port %d: want an error", c.host, c.port)
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Fatalf("host %q port %d: error %q, want it to name %q", c.host, c.port, err, c.want)
		}
	}
}

func TestTheSessionAsksThroughTheClientItWasGiven(t *testing.T) {
	// `HTTPClient` is the seam a caller uses to put its own timeouts or
	// transport in front of the daemon. A session that reached past it to
	// the package-wide default would ignore every one of them.
	ts := answering(t)
	s := &search.HTTPSession{URL: ts.URL, HTTPClient: &http.Client{Transport: refusingTransport{}}}
	if _, err := s.Call("query", nil); err == nil {
		t.Fatal("want the error of the client the session was given")
	}
}

func TestThreeHundredIsNotASuccessfulStatus(t *testing.T) {
	// Success is the 2xx band; 300 is the first status outside it. A
	// session that took 300 for an answer would try to read a body that
	// carries no reply.
	ts := serving(t, http.StatusMultipleChoices, "nothing here")
	s := &search.HTTPSession{URL: ts.URL}
	_, err := s.Call("query", nil)
	if err == nil || !strings.Contains(err.Error(), "server returned status 300") {
		t.Fatalf("err = %v, want the refused status", err)
	}
}

func TestANullErrorMemberIsNoError(t *testing.T) {
	// JSON-RPC 2.0 §5: the `error` member is either absent or an error
	// object. A daemon that writes it out as `null` has reported no error,
	// and the result beside it is the answer.
	ts := serving(t, http.StatusOK, `{"jsonrpc":"2.0","id":1,"error":null,"result":{"ok":true}}`)
	s := &search.HTTPSession{URL: ts.URL}
	got, err := s.Call("query", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if got["ok"] != true {
		t.Fatalf("result = %v, want the ok of the reply", got)
	}
}

func TestAStreamWhoseOnlyDataLineIsItsFirstOne(t *testing.T) {
	// A server-sent-events body may be one `data:` line and nothing else.
	// A reader that stopped one line short of the start would fall through
	// to parsing the frame as plain JSON, which it is not.
	ts := serving(t, http.StatusOK, `data: {"jsonrpc":"2.0","id":1,"result":{"ok":true}}`)
	s := &search.HTTPSession{URL: ts.URL}
	got, err := s.Call("query", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if got["ok"] != true {
		t.Fatalf("result = %v, want the ok of the data line", got)
	}
}

func TestALineWithoutTheDataFieldIsNotTheReply(t *testing.T) {
	// In the SSE framing only a `data:` line carries the payload. A line
	// beside it is not the reply even when it happens to be valid JSON, and
	// the reader takes the last `data:` line, not the last line.
	body := "data: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"which\":\"data\"}}\n" +
		"{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"which\":\"bare\"}}"
	ts := serving(t, http.StatusOK, body)
	s := &search.HTTPSession{URL: ts.URL}
	got, err := s.Call("query", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if got["which"] != "data" {
		t.Fatalf("result = %v, want the one the data line carried", got)
	}
}

func TestAnOpenSessionIsNotConnectedTwice(t *testing.T) {
	// Connecting starts a daemon where none runs and waits for it. Doing
	// that again for a session that is already open would pay the whole
	// cold start for every query.
	connects := 0
	port := search.NewQmdMcpPort(search.WithConnect(func(map[string]string) (search.Session, error) {
		connects++
		return &mockSession{callFunc: func(string, map[string]any) (map[string]any, error) {
			return map[string]any{}, nil
		}}, nil
	}))
	for i := 0; i < 2; i++ {
		if _, err := port.Search("q", nil, search.ProfileFull, 5); err != nil {
			t.Fatalf("Search: %v", err)
		}
	}
	if connects != 1 {
		t.Fatalf("connected %d times, want once", connects)
	}
}

func TestASessionThatFailedIsClosedAndLetGo(t *testing.T) {
	// A session whose call failed is not to be asked again: the retry is
	// there to get a *new* daemon connection. Holding on to the broken one
	// would spend every attempt on the same dead socket and close none.
	connects, closes := 0, 0
	port := search.NewQmdMcpPort(
		search.WithColdAttempts(2),
		search.WithConnect(func(map[string]string) (search.Session, error) {
			connects++
			return &mockSession{
				callFunc:  func(string, map[string]any) (map[string]any, error) { return nil, errors.New("dead") },
				closeFunc: func() error { closes++; return nil },
			}, nil
		}),
	)
	if _, err := port.Search("q", nil, search.ProfileFull, 5); err == nil {
		t.Fatal("want the failure of both attempts")
	}
	if connects != 2 || closes != 2 {
		t.Fatalf("connects = %d, closes = %d, want 2 and 2", connects, closes)
	}
}

func TestAReplyWithoutAResultsListIsNotAnEmptyList(t *testing.T) {
	// A reply whose `structuredContent` carries no `results` is malformed,
	// not a search that found nothing. The two leave this function as a
	// missing list and as an empty one, and a caller that tells them apart
	// reads which of the two it got.
	port := search.NewQmdMcpPort(search.WithConnect(func(map[string]string) (search.Session, error) {
		return &mockSession{callFunc: func(string, map[string]any) (map[string]any, error) {
			return map[string]any{"structuredContent": map[string]any{}}, nil
		}}, nil
	}))
	hits, err := port.Search("q", nil, search.ProfileFull, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if hits != nil {
		t.Fatalf("hits = %#v, want no list at all", hits)
	}
}

func TestAHitIsSplitEvenWhenNoCollectionWasAsked(t *testing.T) {
	// A search over no collection in particular still gets hits back, and
	// they still have to be split into a collection and a path. Reaching
	// for the first of the collections that were asked for would reach into
	// an empty list.
	port := search.NewQmdMcpPort(search.WithConnect(func(map[string]string) (search.Session, error) {
		return &mockSession{callFunc: func(string, map[string]any) (map[string]any, error) {
			return map[string]any{"structuredContent": map[string]any{
				"results": []any{map[string]any{"file": "vault/doc.md"}},
			}}, nil
		}}, nil
	}))
	hits, err := port.Search("q", nil, search.ProfileFull, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 || hits[0].Collection != "" || hits[0].Relative != "vault/doc.md" {
		t.Fatalf("hits = %#v, want one hit with no collection and the whole path", hits)
	}
}

func TestAJSONListThatHoldsNoHitsIsAReadingFailure(t *testing.T) {
	// `_parse` (src/brain/search/qmd.py:238-244) complains that it expected
	// a list of hits only when what it read is not a list at all; a list
	// whose entries are wrong fails later and in other words. Output that
	// is a list of numbers is the second case.
	q := &search.QmdPort{Executable: "qmd", Runner: func([]string) ([]byte, []byte, int, error) {
		return []byte("[1,2]"), nil, 0, nil
	}}
	_, err := q.Search("q", nil, search.ProfileFull, 5)
	if err == nil || !strings.Contains(err.Error(), "could not read the search output") {
		t.Fatalf("err = %v, want the reading failure", err)
	}
}

func TestAHitMayCarryNothingButItsFileAndItsDocID(t *testing.T) {
	// loomux answers a hit without `line` or `score` with 0 where Python
	// refuses it -- the divergence is recorded in
	// `docs/.superpowers/parity/stufe-1b-1.md` in the working papers of the
	// archive release `archive/parity-recordings`, row "Parser der CLI-Suche".
	// `title` and `snippet` default in Python too:
	// `str(row.get("title", ""))` and `str(row.get("snippet", ""))`
	// (src/brain/search/qmd.py:262-263). Reading any of the four without
	// looking first would take the parser down.
	q := &search.QmdPort{Executable: "qmd", Runner: func([]string) ([]byte, []byte, int, error) {
		return []byte(`[{"file":"qmd://col/rel.md","docid":"d1"}]`), nil, 0, nil
	}}
	hits, err := q.Search("q", nil, search.ProfileFull, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits = %#v, want one", hits)
	}
	got := hits[0]
	if got.Line != 0 || got.Score != 0 || got.Title != "" || got.Snippet != "" {
		t.Fatalf("hit = %#v, want the four optional fields at their defaults", got)
	}
	if got.Collection != "col" || got.Relative != "rel.md" || got.ContentKey != "d1" {
		t.Fatalf("hit = %#v, want the file and the docid read", got)
	}
}

func TestAListingLineMayBeginWithTheLocation(t *testing.T) {
	// `indexed` (src/brain/search/qmd.py:172-173) reads
	// `start = line.find(_URI_PREFIX)` and drops the line only on
	// `if start < 0`. A marker at the very start of the line is found at 0,
	// and 0 is a position like any other.
	q := &search.QmdPort{Executable: "qmd", Runner: func([]string) ([]byte, []byte, int, error) {
		return []byte("qmd://col/a.md\n  12  2026-09-16  qmd://col/b.md\n"), nil, 0, nil
	}}
	got, err := q.Indexed("col")
	if err != nil {
		t.Fatalf("Indexed: %v", err)
	}
	if len(got) != 2 || got[0] != "a.md" || got[1] != "b.md" {
		t.Fatalf("Indexed = %q, want both paths", got)
	}
}

func TestEveryCommandFallsBackToTheQmdExecutable(t *testing.T) {
	// The Python port takes `executable: str = "qmd"` (src/brain/search/
	// qmd.py:145) and puts it in front of `update`, `status` and `embed`
	// (qmd.py:186, :205, :222). A port that was given no name uses that
	// one; a port that was given one uses what it was given, on every
	// command and not only on the search.
	run := func(seen *[][]string) search.RunnerFunc {
		return func(argv []string) ([]byte, []byte, int, error) {
			*seen = append(*seen, argv)
			return []byte(""), nil, 0, nil
		}
	}
	for _, c := range []struct{ executable, want string }{
		{executable: "", want: "qmd"},
		{executable: "qmd-of-mine", want: "qmd-of-mine"},
	} {
		var seen [][]string
		q := &search.QmdPort{Executable: c.executable, Runner: run(&seen)}
		if err := q.Refresh(nil); err != nil {
			t.Fatalf("Refresh: %v", err)
		}
		if _, err := q.NotYetSearchable(); err != nil {
			t.Fatalf("NotYetSearchable: %v", err)
		}
		if err := q.Embed(nil); err != nil {
			t.Fatalf("Embed: %v", err)
		}
		if len(seen) != 3 {
			t.Fatalf("executable %q: ran %d commands, want 3", c.executable, len(seen))
		}
		for _, argv := range seen {
			if argv[0] != c.want {
				t.Fatalf("executable %q: ran %q, want %q in front", c.executable, argv, c.want)
			}
		}
	}
}

func TestACallCarriesItsTimeoutOnTheRequest(t *testing.T) {
	// The timeout of a call is put on the request context, which is what
	// makes a daemon that never answers give up after `QueryTimeout`
	// instead of holding the command forever. A request that went out
	// without a deadline would have nothing to give up on.
	ts := answering(t)
	s := &search.HTTPSession{URL: ts.URL, HTTPClient: &http.Client{Transport: reportingTransport{}}}
	_, err := s.Call("query", nil)
	if err == nil || !strings.Contains(err.Error(), "had deadline") {
		t.Fatalf("err = %v, want the request to have carried a deadline", err)
	}
}

func TestAPortBuiltWithoutACLIStillHasOne(t *testing.T) {
	// The maintenance commands go through the CLI port, and a port that was
	// given none falls back to the plain `qmd` executable. The empty PATH
	// is what keeps this test from starting anything: the launcher looks
	// the name up and refuses before a process exists.
	t.Setenv("PATH", t.TempDir())
	port := search.NewQmdMcpPort()
	_, err := port.Indexed("c")
	if err == nil || !strings.Contains(err.Error(), "cannot find 'qmd' on PATH") {
		t.Fatalf("err = %v, want the refusal of the fallback executable", err)
	}
}
