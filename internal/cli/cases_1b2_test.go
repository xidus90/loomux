package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/bridge"
	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/dev/fakeqmd"
	"github.com/xidus90/loomux/internal/serve"
)

// caseTimeout bounds every wait of one replayed case. Nothing here leaves the
// machine, so a case that needs more than this is wedged, not slow.
const caseTimeout = 30 * time.Second

// TestRecordedMCPCasesOfStage1b2 replays the recordings of the reference's MCP
// front against `loomux mcp` over `loomux serve`.
//
// What is compared is the text of the CallToolResult and isError, never the
// envelope: the reference speaks through the Python MCP SDK and loomux through
// the Go one, so initialize alone differs in capabilities, in serverInfo and in
// the revision the two negotiate. Holding that against the reference would be
// holding one library against another, and it would grow again at every SDK
// update. The envelope is held against the specification, in the unit tests of
// internal/serve and internal/bridge.
//
// Every case runs on a state directory of its own, and every case ends its own
// service: the whole stage hangs off LOOMUX_STATE_DIR for exactly this reason.
func TestRecordedMCPCasesOfStage1b2(t *testing.T) {
	all, err := cases.DiscoverMCPCases(filepath.Join("..", "..", "testdata", "cases", "1b-2"))
	if err != nil {
		t.Fatal(err)
	}
	// The number is pinned, not merely non-zero: a partial import must not
	// pass as parity. Raise it with the corpus when a case is added.
	if len(all) != 54 {
		t.Fatalf("expected 54 recorded cases, found %d", len(all))
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			outcome, err := cases.RunMCPCase(c, func(call cases.MCPCall, dir string) (cases.MCPOutcome, error) {
				return answerOneCall(t, call, dir)
			})
			if err != nil {
				t.Fatal(err)
			}
			// An outcome case pins one boolean, so what loomux said is pinned
			// here instead -- see loomuxWording.
			if c.Compare == cases.CompareOutcome {
				want, ok := loomuxWording[c.Verb+"/"+c.Name]
				if !ok {
					t.Fatalf("no wording pinned for the outcome case %s/%s; add one to loomuxWording", c.Verb, c.Name)
				}
				if !strings.Contains(outcome.Actual.Text, want) {
					t.Errorf("loomux said %q, which does not carry %q", outcome.Actual.Text, want)
				}
			}
			if !outcome.Passed {
				t.Fatalf("%s\n%s", strings.Join(outcome.Mismatches, "\n"), c.Notes)
			}
		})
	}
}

// loomuxWording pins what loomux itself says in the fifteen cases graded
// `outcome`.
//
// An outcome case compares one boolean, so it would pass on any text at all --
// `brain-read/missing-section` would be green if loomux answered "the registry
// is on fire". The parity list writes loomux's wording down for all fifteen,
// and until this map existed nothing held the code to what it says there. A
// substring, not the whole sentence: what is pinned is that the refusal names
// its cause, not the punctuation around it.
//
// Every key is a case with a `compare` file, and the test below fails when the
// two sets drift apart -- a new outcome case without a line here is a case
// whose text nothing checks.
var loomuxWording = map[string]string{
	"brain-catalog/broken-registry":          "not valid TOML",
	"brain-catalog/manifest-without-scope":   "[area] scope must be a non-empty string",
	"brain-catalog/manifest-wrong-type":      "[area] scope must be a non-empty string, found integer",
	"brain-catalog/missing-index":            "index.md",
	"brain-catalog/missing-manifest":         "no manifest found",
	"brain-catalog/missing-registry":         "registry.toml",
	"brain-neighbors/manifest-without-scope": "[area] scope must be a non-empty string",
	"brain-read/missing-file":                "gone.md",
	"brain-read/missing-manifest":            "no manifest found",
	"brain-read/missing-section":             "no section titled 'Nowhere'",
	"brain-search/invalid-profile":           `invalid profile "deep"; choose from fast, full, keyword`,
	"brain-search/missing-query":             "query is required",
	"brain-search/missing-registry":          "registry.toml",
	"brain-status/broken-register":           "expected 4 tab-separated fields, found 3",
	"brain-status/missing-manifest":          "no manifest found",
}

// answerOneCall is the whole loomux side of one case: a service on this case's
// state directory, the real bridge in front of it, and one tool call through
// both.
func answerOneCall(t *testing.T, call cases.MCPCall, dir string) (cases.MCPOutcome, error) {
	t.Helper()
	// Both ends are deferred here and registered with no t.Cleanup, because
	// t.Cleanup runs after the subtest closure and therefore after RunMCPCase
	// has removed the staged world: serve.Stop would then begin by reading a
	// serve.json that is gone and could only ever fail. Deferred here, the
	// service is stopped while its state directory still exists, which is what
	// makes the stop a real stop and its failure worth reporting.
	//
	// The bridge goes first, the service second: a session still open when the
	// listener closes is a shutdown waiting for what it has in flight.
	stopService := serveInDir(t, dir)
	defer stopService()
	session, closeBridge := bridgeTo(t, dir, call.Channel)
	defer closeBridge()

	ctx, cancel := context.WithTimeout(context.Background(), caseTimeout)
	defer cancel()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      call.Tool,
		Arguments: json.RawMessage(call.Arguments),
	})
	if err != nil {
		// A call that never became a result is the protocol error the
		// reference's own crashes are recorded as.
		return cases.MCPOutcome{RPCError: err.Error()}, nil
	}
	var texts []string
	for _, block := range res.Content {
		if text, ok := block.(*mcp.TextContent); ok {
			texts = append(texts, text.Text)
		}
	}
	return cases.MCPOutcome{IsError: res.IsError, Text: strings.Join(texts, "\n")}, nil
}

// serveInDir runs a service on this case's state directory and hands back the
// one call that ends it.
//
// The stop goes through serve.Stop, the same call `loomux serve stop` makes,
// and not merely through the context: a case that only cancelled would leave
// serve.json behind naming a port nobody listens on. **A stop that fails is a
// failed case**, because a silently swallowed one would let a permanently
// broken reaping pass for 54 green cases. The context cancellation stands
// behind it as the way a service that did not take the stop still dies.
func serveInDir(t *testing.T, dir string) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	// Built here and not inside the goroutine below, because it is the
	// testing.T's own API: answerFrom calls t.Cleanup for the fixture's server
	// and t.Fatal when the fixture cannot be read, and neither belongs on a
	// goroutine that is not the test's. The Fatal arm is unreachable today --
	// fakeqmd.Load answers a missing file with an empty fixture, which 15 of the
	// 54 worlds rely on -- and that is a reason to keep the call in the right
	// place, not a reason to care less where it stands.
	answering := answerFrom(t, dir)
	done := make(chan error, 1)
	go func() {
		done <- serve.Run(ctx, serve.Options{
			StateDir:    dir,
			RegistryDir: dir,
			LegacyDir:   dir,
			Foreground:  true,
			Answer:      answering,
		})
	}()
	stop := func() {
		// Last of all, and on every way out: a cancel that only the failing
		// path reached would leave a service behind on the passing one.
		defer cancel()
		if err := serve.Stop(dir, false); err != nil {
			t.Errorf("serve stop in %s: %v", dir, err)
			cancel()
		}
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("serve.Run: %v", err)
			}
		case <-time.After(caseTimeout):
			t.Error("serve.Run did not return after the stop")
		}
	}
	if err := awaitState(dir); err != nil {
		// Stopped before the case is given up on: what came up half way is
		// still a process holding a lock in a directory about to be removed.
		stop()
		t.Fatal(err)
	}
	return stop
}

// awaitState holds the case until the service has written serve.json, which is
// the last thing it does and the first thing a bridge reads.
func awaitState(dir string) error {
	deadline := time.Now().Add(caseTimeout)
	for {
		if _, err := serve.ReadState(dir); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("the service in %s never wrote its state", dir)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// answerFrom is the service's answer function, with both search seams pointed
// at the world's fixture: the MCP port over httptest, the command line port
// through the Runner seam. No replay starts qmd, exactly as no recording did.
func answerFrom(t *testing.T, dir string) func(answer.Request, string, string, func(string)) (string, []string, error) {
	t.Helper()
	fixture, err := fakeqmd.Load(filepath.Join(dir, fakeqmd.FixtureName))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(fixture.MCPHandler())
	t.Cleanup(server.Close)
	runner := func(argv []string) ([]byte, []byte, int, error) {
		var out, errOut bytes.Buffer
		code := fixture.RunCLI(argv[1:], &out, &errOut)
		return out.Bytes(), errOut.Bytes(), code, nil
	}
	cli := &search.QmdPort{Executable: "qmd", Runner: runner}
	ports := answer.Ports{
		Search: func(notice func(string)) search.SearchPort {
			return search.NewQmdMcpPort(
				search.WithNotice(notice),
				search.WithConnect(func(map[string]string) (search.Session, error) {
					return &search.HTTPSession{URL: server.URL}, nil
				}),
				search.WithCLI(cli),
			)
		},
		Status: func() search.SearchPort { return cli },
		Now:    time.Now,
	}
	return func(req answer.Request, registryDir, legacyDir string, notice func(string)) (string, []string, error) {
		return answer.RunWith(ports, req, registryDir, legacyDir, notice)
	}
}

// bridgeTo runs the real bridge against the service in dir and hands back the
// host's end of it together with the call that hangs up.
//
// Hanging up is the caller's to time, not t.Cleanup's, for the reason
// answerOneCall gives: it has to happen while the case's state directory is
// still there, and before the service is stopped.
//
// The bridge's Spawn refuses: the service is already there, and a spawner that
// started one would put a detached process beside every case.
func bridgeTo(t *testing.T, dir, channel string) (*mcp.ClientSession, func()) {
	t.Helper()
	serverSide, clientSide := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- bridge.Run(ctx, bridge.Options{
			StateDir:  dir,
			Channel:   privacy.Channel(channel),
			Transport: serverSide,
			Spawn:     func(string) (bool, error) { return false, errors.New("no case may start a service") },
		})
	}()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "host", Version: "1"}, nil).
		Connect(ctx, clientSide, nil)
	if err != nil {
		cancel()
		t.Fatalf("connect to the bridge: %v", err)
	}
	return session, func() {
		_ = session.Close()
		cancel()
		select {
		case <-done:
		case <-time.After(caseTimeout):
			t.Error("bridge.Run did not return after the host went away")
		}
	}
}

// Every pinned wording belongs to a case that is graded `outcome`, and every
// such case has one. A line left behind after a case was re-recorded at the
// text grade would pin nothing; a case added at the outcome grade without a
// line would have its text checked by nobody.
func TestEveryOutcomeCaseHasItsWordingPinned(t *testing.T) {
	all, err := cases.DiscoverMCPCases(filepath.Join("..", "..", "testdata", "cases", "1b-2"))
	if err != nil {
		t.Fatal(err)
	}
	graded := map[string]bool{}
	for _, c := range all {
		if c.Compare == cases.CompareOutcome {
			graded[c.Verb+"/"+c.Name] = true
		}
	}
	for name := range graded {
		if _, ok := loomuxWording[name]; !ok {
			t.Errorf("%s is graded outcome and pins no wording", name)
		}
	}
	for name := range loomuxWording {
		if !graded[name] {
			t.Errorf("%s pins a wording but is no outcome case", name)
		}
	}
}
